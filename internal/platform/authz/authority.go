package authz

import (
	"context"
	"errors"
	"fmt"

	"kinstack/internal/platform/i18n"
)

// superuserRank is the rank projected for the break-glass account, above every
// seeded or custom role.
const superuserRank = 1000

// principalContextKey stores the acting principal for authorization mutations.
type principalContextKey struct{}

// WithPrincipal attaches the acting principal to a request context. Handlers do
// this once so services can enforce delegation rules without threading an actor
// through every call.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

// PrincipalFromContext returns the acting principal attached by a handler.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok && principal.PublicID != ""
}

// AuthorityError marks a delegation rule violation: the actor is allowed to
// manage access in general but not this particular role, user, or permission.
var (
	// ErrPrivilegeEscalation identifies a change that would grant access the
	// actor does not hold themselves.
	ErrPrivilegeEscalation = errors.New(i18n.T("authz.reason.privilege_escalation"))
	// ErrSelfChange identifies an attempt to change the actor's own authority.
	ErrSelfChange = errors.New(i18n.T("authz.reason.self_change"))
	// ErrManagementSession identifies a token actor trying to manage authority.
	ErrManagementSession = errors.New(i18n.T("authz.reason.browser_session_required"))
)

// AuthorizeRoleGrant reports whether the actor may assign roleCode to someone
// else: they must hold access.manage, the role must not outrank them, and every
// permission of that role must already be effective for them.
func (s *Service) AuthorizeRoleGrant(ctx context.Context, roleCode string) error {
	actor, err := s.actingPrincipal(ctx)
	if err != nil {
		return err
	}
	_, err = s.authorizeRoleGrant(ctx, actor, roleCode)
	return err
}

// AuthorizeRoleEdit reports whether the actor may rename, re-rank, re-permission
// or delete a role. Built-in roles stay immutable.
func (s *Service) AuthorizeRoleEdit(ctx context.Context, roleCode string, newRank int) error {
	actor, err := s.actingPrincipal(ctx)
	if err != nil {
		return err
	}
	permissions, rank, err := s.authorityOf(ctx, actor)
	if err != nil {
		return err
	}
	role, err := s.repository.FindRole(ctx, roleCode)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return ErrSystemRole
	}
	if role.Rank > rank {
		return fmt.Errorf("%w: %s outranks you", ErrPrivilegeEscalation, role.Code)
	}
	if newRank < 0 || newRank > rank {
		return fmt.Errorf("%w: rank %d is above your rank", ErrPrivilegeEscalation, newRank)
	}
	return ensureSubset(ctx, s, permissions, role.Code)
}

// AuthorizeNewRole reports whether the actor may create a role at this rank.
func (s *Service) AuthorizeNewRole(ctx context.Context, rank int) error {
	actor, err := s.actingPrincipal(ctx)
	if err != nil {
		return err
	}
	_, actorRank, err := s.authorityOf(ctx, actor)
	if err != nil {
		return err
	}
	if rank < 0 || rank > actorRank {
		return fmt.Errorf("%w: rank %d is above your rank", ErrPrivilegeEscalation, rank)
	}
	return nil
}

// AuthorizePermissions reports whether the actor holds every listed permission,
// so they cannot hand out access they do not have.
func (s *Service) AuthorizePermissions(ctx context.Context, codes []Permission) error {
	actor, err := s.actingPrincipal(ctx)
	if err != nil {
		return err
	}
	permissions, _, err := s.authorityOf(ctx, actor)
	if err != nil {
		return err
	}
	for _, code := range codes {
		if _, ok := permissions[code]; !ok {
			return fmt.Errorf("%w: %s", ErrPrivilegeEscalation, code)
		}
	}
	return nil
}

// AuthorizeUserChange reports whether the actor may change another account's
// role or overrides. Actors cannot change their own authority, and they cannot
// modify an account whose role outranks them.
func (s *Service) AuthorizeUserChange(ctx context.Context, targetPublicID string) error {
	actor, err := s.actingPrincipal(ctx)
	if err != nil {
		return err
	}
	if actor.Superuser {
		return nil
	}
	if targetPublicID == actor.PublicID {
		return ErrSelfChange
	}
	_, actorRank, err := s.authorityOf(ctx, actor)
	if err != nil {
		return err
	}
	roleCode, err := s.repository.UserRole(ctx, targetPublicID)
	if err != nil {
		return err
	}
	if roleCode == "" {
		// Accounts that were just created have no role row yet; the repository
		// still validates that the account exists.
		return nil
	}
	target, err := s.repository.FindRole(ctx, roleCode)
	if err != nil {
		return err
	}
	if target.Rank > actorRank {
		return fmt.Errorf("%w: %s outranks you", ErrPrivilegeEscalation, targetPublicID)
	}
	return nil
}

// actingPrincipal returns the principal attached by a handler and requires an
// interactive session: authority changes are never allowed from an API token,
// even one holding access.manage.
func (s *Service) actingPrincipal(ctx context.Context) (Principal, error) {
	actor, ok := PrincipalFromContext(ctx)
	if !ok {
		return Principal{}, ErrPrivilegeEscalation
	}
	if actor.Scopes != nil {
		return Principal{}, ErrManagementSession
	}
	return actor, nil
}

// authorityOf returns the actor's effective permissions and rank.
func (s *Service) authorityOf(
	ctx context.Context,
	actor Principal,
) (map[Permission]struct{}, int, error) {
	if actor.Superuser {
		return KnownCodes(), superuserRank, nil
	}
	if decision := s.Can(ctx, actor, PermissionAccessManage, nil); !decision.Allowed {
		return nil, 0, fmt.Errorf("%w: access.manage is required", ErrPrivilegeEscalation)
	}
	permissions, err := s.effective(ctx, actor)
	if err != nil {
		return nil, 0, err
	}
	rank := 0
	if role, err := s.repository.FindRole(ctx, actor.Role); err == nil {
		rank = role.Rank
	}
	return permissions, rank, nil
}

func (s *Service) authorizeRoleGrant(
	ctx context.Context,
	actor Principal,
	roleCode string,
) (int, error) {
	permissions, rank, err := s.authorityOf(ctx, actor)
	if err != nil {
		return 0, err
	}
	role, err := s.repository.FindRole(ctx, roleCode)
	if err != nil {
		return 0, err
	}
	if role.Rank > rank {
		return 0, fmt.Errorf("%w: %s outranks you", ErrPrivilegeEscalation, role.Code)
	}
	return rank, ensureSubset(ctx, s, permissions, role.Code)
}

func ensureSubset(
	ctx context.Context,
	service *Service,
	actorPermissions map[Permission]struct{},
	roleCode string,
) error {
	rolePermissions, err := service.repository.RolePermissions(ctx, roleCode)
	if err != nil {
		return err
	}
	for _, permission := range rolePermissions {
		if _, ok := actorPermissions[permission]; !ok {
			return fmt.Errorf("%w: %s", ErrPrivilegeEscalation, permission)
		}
	}
	return nil
}
