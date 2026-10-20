package authz

import (
	"context"
	"fmt"
	"time"

	"kinstack/internal/platform/i18n"
)

// Engine evaluates code-defined ABAC policies. It is deliberately free of I/O so
// it can be unit tested in isolation.
type Engine struct {
	policies map[Permission][]Policy
}

// NewEngine constructs an ABAC engine from registered policies.
func NewEngine(policies ...Policy) *Engine {
	indexed := make(map[Permission][]Policy)
	for _, policy := range policies {
		if policy == nil {
			continue
		}
		if err := appendPolicy(indexed, policy); err != nil {
			panic(err)
		}
	}
	return &Engine{policies: indexed}
}

// DefaultEngine returns the production policy set. The base ships no ABAC
// policies; feature slices register their own when they introduce a permission
// that needs attribute guardrails.
func DefaultEngine() *Engine {
	return NewEngine()
}

// Evaluate applies policies for the permission on top of a base (RBAC) decision.
// A deny from any policy wins; an allow may lift a base denial.
func (e *Engine) Evaluate(req Request, base Decision) Decision {
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	decision := base
	if decision.Source == "" {
		decision.Source = "role"
	}
	for _, policy := range e.policies[req.Permission] {
		verdict := policy.Evaluate(req)
		if verdict == nil {
			continue
		}
		if !verdict.Allowed {
			return *verdict
		}
		if !decision.Allowed {
			decision = *verdict
		}
	}
	return decision
}

// Policies returns the policy names registered for a permission.
func (e *Engine) Policies(permission Permission) []string {
	registered := e.policies[permission]
	names := make([]string, 0, len(registered))
	for _, policy := range registered {
		names = append(names, policy.Name())
	}
	return names
}

func appendPolicy(index map[Permission][]Policy, policy Policy) error {
	permission := policyPermission(policy)
	if permission == "" {
		return fmt.Errorf("policy %s does not declare a permission", policy.Name())
	}
	if !IsKnown(permission) {
		return fmt.Errorf("policy %s references unknown permission %s", policy.Name(), permission)
	}
	index[permission] = append(index[permission], policy)
	return nil
}

// baseDecision builds the RBAC baseline decision for a resolved permission set.
func baseDecision(principal Principal, granted map[Permission]struct{}, permission Permission) Decision {
	if permission == Authenticated {
		return Allow("authenticated", i18n.T("authz.reason.authenticated"))
	}
	if _, ok := granted[permission]; ok {
		return Allow("role", i18n.T("authz.reason.role_allowed"))
	}
	return Deny("role", i18n.T("authz.reason.role_denied"))
}

func policyPermission(policy Policy) Permission {
	switch typed := policy.(type) {
	case OwnershipPolicy:
		return typed.Permission
	case AmountLimitPolicy:
		return typed.Permission
	case TimeWindowPolicy:
		return typed.Permission
	case ShiftPolicy:
		return typed.Permission
	default:
		return ""
	}
}

// Evaluate combines the RBAC baseline and ABAC policies for one request. It does
// not resolve role or override data; callers supply those via a Principal and
// the resolved effective permission set.
func (e *Engine) Decide(
	ctx context.Context,
	principal Principal,
	granted map[Permission]struct{},
	permission Permission,
	resource Resource,
) Decision {
	_ = ctx
	base := baseDecision(principal, granted, permission)
	return e.Evaluate(Request{
		Principal:  principal,
		Permission: permission,
		Resource:   resource,
		Now:        time.Now(),
	}, base)
}
