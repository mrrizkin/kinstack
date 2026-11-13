package authz

import (
	"time"

	"kinstack/internal/platform/i18n"
)

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

// Resource values use stable keys (see ResourceOwner, ResourceAmount, ResourceShiftOpen).
type Resource map[string]any

const (
	ResourceOwner     = "owner_id"
	ResourceAmount    = "amount"
	ResourceShiftOpen = "shift_open"
	ResourceScope     = "scope"
)

func (r Resource) String(key string) string {
	value, _ := r[key].(string)
	return value
}

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

func (r Resource) Bool(key string) (bool, bool) {
	value, ok := r[key].(bool)
	return value, ok
}

type Request struct {
	Principal  Principal
	Permission Permission
	Resource   Resource
	Now        time.Time
}

type Decision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
	Source  string `json:"source,omitempty"`
}

func Allow(source, reason string) Decision {
	return Decision{Allowed: true, Source: source, Reason: reason}
}

func Deny(source, reason string) Decision {
	return Decision{Allowed: false, Source: source, Reason: reason}
}

type Policy interface {
	Name() string
	Evaluate(Request) *Decision
}

// OwnershipPolicy allows the owner or the exempt role, and abstains when the resource has
// no owner attribute.
type OwnershipPolicy struct {
	PolicyName string
	Permission Permission
	ExemptRole string
	Message    string
}

func (p OwnershipPolicy) Name() string { return p.PolicyName }

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

// AmountLimitPolicy denies over Limit unless the principal holds the exempt role, and
// abstains when the amount is absent.
type AmountLimitPolicy struct {
	PolicyName string
	Permission Permission
	ExemptRole string
	Limit      float64
	Message    string
}

func (p AmountLimitPolicy) Name() string { return p.PolicyName }

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

type TimeWindowPolicy struct {
	PolicyName string
	Permission Permission
	ExemptRole string
	FromHour   int
	ToHour     int
	Weekdays   []time.Weekday
	Message    string
}

func (p TimeWindowPolicy) Name() string { return p.PolicyName }

func (p TimeWindowPolicy) Evaluate(req Request) *Decision {
	if req.Permission != p.Permission || req.Principal.Role == p.ExemptRole ||
		req.Now.IsZero() {
		return nil
	}
	if !p.weekdayAllowed(req.Now) || !p.hourAllowed(req.Now) {
		return decisionPtr(Deny(p.PolicyName, p.reason()))
	}
	return nil
}

// weekdayAllowed abstains when no weekdays are named, so a time-only window does not deny
// on an unset field.
func (p TimeWindowPolicy) weekdayAllowed(now time.Time) bool {
	if len(p.Weekdays) == 0 {
		return true
	}
	for _, weekday := range p.Weekdays {
		if now.Weekday() == weekday {
			return true
		}
	}
	return false
}

// hourAllowed treats to >= from as a same-day window and anything else as wrapping
// midnight.
func (p TimeWindowPolicy) hourAllowed(now time.Time) bool {
	hour := now.Hour()
	if p.ToHour >= p.FromHour {
		return hour >= p.FromHour && hour < p.ToHour
	}
	return hour >= p.FromHour || hour < p.ToHour
}

func (p TimeWindowPolicy) reason() string {
	if p.Message != "" {
		return p.Message
	}
	return "aksi hanya tersedia pada jam operasional"
}

type ShiftPolicy struct {
	PolicyName string
	Permission Permission
	Message    string
}

func (p ShiftPolicy) Name() string { return p.PolicyName }

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
