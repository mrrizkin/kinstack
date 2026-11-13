package authz

import (
	"context"
	"errors"
	"fmt"

	"kinstack/internal/platform/i18n"
)

const superuserRank = 1000

type principalContextKey struct{}

// WithPrincipal attaches the acting principal so services can enforce delegation rules
// without threading an actor through every call.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok && principal.PublicID != ""
}

// AuthorityError marks a delegation rule violation: the actor may manage access in
// general but not this particular role, user, or permission.
var (
	// ErrPrivilegeEscalation identifies a change that would grant access the
	// actor does not hold themselves.
	ErrPrivilegeEscalation = errors.New(i18n.T("authz.reason.privilege_escalation"))
	// ErrSelfChange identifies an attempt to change the actor's own authority.
	ErrSelfChange = errors.New(i18n.T("authz.reason.self_change"))
	// ErrManagementSession identifies a token actor trying to manage authority.
	ErrManagementSession = errors.New(i18n.T("authz.reason.browser_session_required"))
)

// AuthorizeRoleGrant requires access.manage, a role that does not outrank the actor, and
// every permission of that role already effective for them.
func (s *Service) AuthorizeRoleGrant(ctx context.Context, roleCode string) error {
	actor, err := s.actingPrincipal(ctx)
	if err != nil {
		return err
	}
	_, err = s.authorizeRoleGrant(ctx, actor, roleCode)
	return err
}

// AuthorizeRoleEdit keeps built-in roles immutable.
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

// AuthorizePermissions prevents handing out access the actor does not hold.
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
		// Accounts created moments ago have no role row yet; the repository still validates
		// that the account exists.
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

// actingPrincipal requires an interactive session: authority changes are never allowed
// from an API token, even one holding access.manage.
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
