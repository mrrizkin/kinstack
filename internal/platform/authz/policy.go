package authz

import (
	"time"

	"kinstack/internal/platform/i18n"
)

// Principal is the authenticated subject of an authorization decision.
type Principal struct {
	PublicID    string `json:"public_id"`
	Role        string `json:"role"`
	DisplayName string `json:"display_name,omitempty"`
	// Superuser marks the ephemeral break-glass session, which is granted every
	// known permission without a database account or role.
	Superuser bool `json:"superuser,omitempty"`
	// Scopes, when non-nil, restricts the principal to the intersection of the
	// resolved permissions and this allowlist. A nil slice means no restriction
	// (a normal browser session); an empty slice denies every permission.
	Scopes []string `json:"scopes,omitempty"`
}

// Resource carries the attributes an ABAC policy may inspect. Values use stable
// keys (see ResourceOwner, ResourceAmount, ResourceShiftOpen).
type Resource map[string]any

// Well-known resource attribute keys.
const (
	ResourceOwner     = "owner_id"
	ResourceAmount    = "amount"
	ResourceShiftOpen = "shift_open"
	ResourceScope     = "scope"
)

// String reads a string attribute, returning "" when absent or mistyped.
func (r Resource) String(key string) string {
	value, _ := r[key].(string)
	return value
}

// Float reads a numeric attribute, accepting the common numeric Go types.
func (r Resource) Float(key string) (float64, bool) {
	switch value := r[key].(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case int32:
		return float64(value), true
	default:
		return 0, false
	}
}

// Bool reads a boolean attribute.
func (r Resource) Bool(key string) (bool, bool) {
	value, ok := r[key].(bool)
	return value, ok
}

// Request is the full input to an ABAC policy.
type Request struct {
	Principal  Principal
	Permission Permission
	Resource   Resource
	Now        time.Time
}

// Decision is the outcome of an authorization check.
type Decision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
	Source  string `json:"source,omitempty"`
}

// Allow builds an allowing decision.
func Allow(source, reason string) Decision {
	return Decision{Allowed: true, Source: source, Reason: reason}
}

// Deny builds a denying decision.
func Deny(source, reason string) Decision {
	return Decision{Allowed: false, Source: source, Reason: reason}
}

// Policy is an ABAC guardrail. Evaluate returns nil to abstain.
type Policy interface {
	Name() string
	Evaluate(Request) *Decision
}

// OwnershipPolicy allows an action only when the principal owns the resource or
// holds the exempt role. It abstains when the resource has no owner attribute.
type OwnershipPolicy struct {
	PolicyName string
	Permission Permission
	ExemptRole string
	Message    string
}

// Name implements Policy.
func (p OwnershipPolicy) Name() string { return p.PolicyName }

// Evaluate implements Policy.
func (p OwnershipPolicy) Evaluate(req Request) *Decision {
	if req.Permission != p.Permission {
		return nil
	}
	owner := req.Resource.String(ResourceOwner)
	if owner == "" {
		return nil
	}
	if req.Principal.Role == p.ExemptRole || req.Principal.PublicID == owner {
		return nil
	}
	message := p.Message
	if message == "" {
		message = "hanya dapat mengubah data milik sendiri"
	}
	return decisionPtr(Deny(p.PolicyName, message))
}

// AmountLimitPolicy denies actions whose resource amount exceeds Limit unless
// the principal holds the exempt role. It abstains when the amount is absent.
type AmountLimitPolicy struct {
	PolicyName string
	Permission Permission
	ExemptRole string
	Limit      float64
	Message    string
}

// Name implements Policy.
func (p AmountLimitPolicy) Name() string { return p.PolicyName }

// Evaluate implements Policy.
func (p AmountLimitPolicy) Evaluate(req Request) *Decision {
	if req.Permission != p.Permission {
		return nil
	}
	amount, ok := req.Resource.Float(ResourceAmount)
	if !ok {
		return nil
	}
	if req.Principal.Role == p.ExemptRole {
		return nil
	}
	if amount <= p.Limit {
		return nil
	}
	message := p.Message
	if message == "" {
		message = i18n.T("authz.reason.amount_limit")
	}
	return decisionPtr(Deny(p.PolicyName, message))
}

// TimeWindowPolicy allows a permission only between FromHour and ToHour on the
// permitted weekdays. It abstains when the request has no clock.
type TimeWindowPolicy struct {
	PolicyName string
	Permission Permission
	ExemptRole string
	FromHour   int
	ToHour     int
	Weekdays   []time.Weekday
	Message    string
}

// Name implements Policy.
func (p TimeWindowPolicy) Name() string { return p.PolicyName }

// Evaluate implements Policy.
func (p TimeWindowPolicy) Evaluate(req Request) *Decision {
	if req.Permission != p.Permission {
		return nil
	}
	if req.Principal.Role == p.ExemptRole {
		return nil
	}
	now := req.Now
	if now.IsZero() {
		return nil
	}
	if len(p.Weekdays) > 0 {
		allowed := false
		for _, weekday := range p.Weekdays {
			if now.Weekday() == weekday {
				allowed = true
				break
			}
		}
		if !allowed {
			return decisionPtr(Deny(p.PolicyName, p.reason()))
		}
	}
	hour := now.Hour()
	if p.ToHour >= p.FromHour {
		if hour >= p.FromHour && hour < p.ToHour {
			return nil
		}
	} else if hour >= p.FromHour || hour < p.ToHour {
		return nil
	}
	return decisionPtr(Deny(p.PolicyName, p.reason()))
}

func (p TimeWindowPolicy) reason() string {
	if p.Message != "" {
		return p.Message
	}
	return "aksi hanya tersedia pada jam operasional"
}

// ShiftPolicy requires an open shift attribute for the permission. It abstains
// when the resource does not carry shift state.
type ShiftPolicy struct {
	PolicyName string
	Permission Permission
	Message    string
}

// Name implements Policy.
func (p ShiftPolicy) Name() string { return p.PolicyName }

// Evaluate implements Policy.
func (p ShiftPolicy) Evaluate(req Request) *Decision {
	if req.Permission != p.Permission {
		return nil
	}
	open, ok := req.Resource.Bool(ResourceShiftOpen)
	if !ok {
		return nil
	}
	if open {
		return nil
	}
	message := p.Message
	if message == "" {
		message = "buka shift terlebih dahulu"
	}
	return decisionPtr(Deny(p.PolicyName, message))
}

func decisionPtr(decision Decision) *Decision {
	return &decision
}
