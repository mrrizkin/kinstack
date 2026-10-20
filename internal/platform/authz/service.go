package authz

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/platform/i18n"
)

const (
	defaultCacheTTL = 30 * time.Second
	maxReasonLength = 200
)

var roleCodePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{2,31}$`)

// ServiceDependency contains the persistence boundary and optional tuning for
// the authorization service.
type ServiceDependency struct {
	Repository Repository
	Engine     *Engine
	CacheTTL   time.Duration
	Now        func() time.Time
}

// Service resolves effective permissions (RBAC + overrides) and evaluates ABAC
// policies. It also owns the validation rules for role and override mutations.
type Service struct {
	repository Repository
	engine     *Engine
	ttl        time.Duration
	now        func() time.Time

	mu    sync.RWMutex
	cache map[string]resolvedPermissions
}

type resolvedPermissions struct {
	role      string
	set       map[Permission]struct{}
	expiresAt time.Time
}

// New constructs the authorization service. A zero CacheTTL disables caching
// and is used by tests.
func New(d ServiceDependency) *Service {
	assert.Assert(d.Repository != nil, "authz repository is required")

	engine := d.Engine
	if engine == nil {
		engine = DefaultEngine()
	}
	now := d.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		repository: d.Repository,
		engine:     engine,
		ttl:        d.CacheTTL,
		now:        now,
		cache:      make(map[string]resolvedPermissions),
	}
}

// Engine returns the ABAC engine used by the service.
func (s *Service) Engine() *Engine { return s.engine }

// Can evaluates one permission for a principal, optionally with resource
// attributes for ABAC policies.
func (s *Service) Can(
	ctx context.Context,
	principal Principal,
	permission Permission,
	resource Resource,
) Decision {
	if permission == Authenticated {
		if principal.PublicID == "" {
			return Deny("authenticated", i18n.T("authz.reason.invalid_session"))
		}
		return Allow("authenticated", i18n.T("authz.reason.authenticated"))
	}
	if permission == BrowserSession {
		if principal.PublicID == "" {
			return Deny("browser_session", i18n.T("authz.reason.invalid_session"))
		}
		if principal.Scopes != nil {
			return Deny("browser_session", i18n.T("authz.reason.browser_session_required"))
		}
		return Allow("browser_session", i18n.T("authz.reason.browser_session"))
	}
	if !IsKnown(permission) {
		return Deny("catalog", i18n.T("authz.reason.unknown_permission"))
	}
	granted, err := s.effective(ctx, principal)
	if err != nil {
		return Deny("resolution", i18n.T("authz.reason.resolution_failed"))
	}
	return s.engine.Decide(ctx, principal, granted, permission, resource)
}

// Permissions returns the sorted effective permission codes for a principal.
func (s *Service) Permissions(ctx context.Context, principal Principal) ([]Permission, error) {
	granted, err := s.effective(ctx, principal)
	if err != nil {
		return nil, err
	}
	codes := make([]Permission, 0, len(granted))
	for code := range granted {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	return codes, nil
}

// Has reports whether the principal holds a permission, ignoring ABAC policies.
func (s *Service) Has(
	ctx context.Context,
	principal Principal,
	permission Permission,
) (bool, error) {
	granted, err := s.effective(ctx, principal)
	if err != nil {
		return false, err
	}
	_, ok := granted[permission]
	return ok, nil
}

// Invalidate drops the cached resolution for one user.
func (s *Service) Invalidate(publicID string) {
	if s.ttl <= 0 || publicID == "" {
		return
	}
	s.mu.Lock()
	delete(s.cache, publicID)
	s.mu.Unlock()
}

// InvalidateRole drops cached resolutions for every user holding the role.
func (s *Service) InvalidateRole(role string) {
	if s.ttl <= 0 || role == "" {
		return
	}
	s.mu.Lock()
	for publicID, entry := range s.cache {
		if entry.role == role {
			delete(s.cache, publicID)
		}
	}
	s.mu.Unlock()
}

func (s *Service) effective(
	ctx context.Context,
	principal Principal,
) (map[Permission]struct{}, error) {
	if principal.Superuser {
		return KnownCodes(), nil
	}
	var set map[Permission]struct{}
	if s.ttl > 0 && principal.PublicID != "" {
		if cached, ok := s.fromCache(principal); ok {
			set = cached
		}
	}
	if set == nil {
		loaded, err := s.load(ctx, principal)
		if err != nil {
			return nil, err
		}
		set = loaded
		if s.ttl > 0 && principal.PublicID != "" {
			s.mu.Lock()
			s.cache[principal.PublicID] = resolvedPermissions{
				role:      principal.Role,
				set:       set,
				expiresAt: s.now().Add(s.ttl),
			}
			s.mu.Unlock()
		}
	}
	return restrictToScopes(set, principal.Scopes), nil
}

// restrictToScopes intersects a resolved permission set with a token allowlist.
// A nil allowlist keeps the set unchanged; an empty allowlist denies everything.
func restrictToScopes(set map[Permission]struct{}, scopes []string) map[Permission]struct{} {
	if scopes == nil {
		return set
	}
	restricted := make(map[Permission]struct{}, len(scopes))
	for _, raw := range scopes {
		permission := Permission(strings.TrimSpace(raw))
		if _, ok := set[permission]; ok {
			restricted[permission] = struct{}{}
		}
	}
	return restricted
}

func (s *Service) load(
	ctx context.Context,
	principal Principal,
) (map[Permission]struct{}, error) {
	rolePermissions, err := s.repository.RolePermissions(ctx, principal.Role)
	if err != nil {
		return nil, err
	}
	overrides, err := s.repository.UserOverrides(ctx, principal.PublicID)
	if err != nil {
		return nil, err
	}
	set := make(map[Permission]struct{}, len(rolePermissions)+len(overrides))
	for _, permission := range rolePermissions {
		if IsKnown(permission) {
			set[permission] = struct{}{}
		}
	}
	now := s.now()
	for _, override := range overrides {
		if override.Expired(now) || override.Effect != EffectAllow ||
			!IsKnown(override.Permission) {
			continue
		}
		set[override.Permission] = struct{}{}
	}
	for _, override := range overrides {
		if override.Expired(now) || override.Effect != EffectDeny {
			continue
		}
		delete(set, override.Permission)
	}
	return set, nil
}

func (s *Service) fromCache(principal Principal) (map[Permission]struct{}, bool) {
	s.mu.RLock()
	entry, ok := s.cache[principal.PublicID]
	s.mu.RUnlock()
	if !ok || entry.role != principal.Role || !s.now().Before(entry.expiresAt) {
		if ok {
			s.Invalidate(principal.PublicID)
		}
		return nil, false
	}
	clone := make(map[Permission]struct{}, len(entry.set))
	for permission := range entry.set {
		clone[permission] = struct{}{}
	}
	return clone, true
}

// FindRole resolves a single role.
func (s *Service) FindRole(ctx context.Context, code string) (Role, error) {
	return s.repository.FindRole(ctx, strings.TrimSpace(code))
}

// ListRoles returns every role ordered by rank then code.
// CountRoles returns the total number of roles.
func (s *Service) CountRoles(ctx context.Context) (int, error) {
	return s.repository.CountRoles(ctx)
}

func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	roles, err := s.repository.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(roles, func(i, j int) bool {
		if roles[i].Rank != roles[j].Rank {
			return roles[i].Rank > roles[j].Rank
		}
		return roles[i].Code < roles[j].Code
	})
	return roles, nil
}

// RolePermissions returns the permission codes granted to a role.
func (s *Service) RolePermissions(ctx context.Context, code string) ([]Permission, error) {
	permissions, err := s.repository.RolePermissions(ctx, code)
	if err != nil {
		return nil, err
	}
	result := make([]Permission, 0, len(permissions))
	for _, permission := range permissions {
		if IsKnown(permission) {
			result = append(result, permission)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

// CreateRole validates and persists a new role.
func (s *Service) CreateRole(
	ctx context.Context,
	code string,
	name string,
	description string,
	rank int,
) (Role, error) {
	code = strings.TrimSpace(strings.ToLower(code))
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if !roleCodePattern.MatchString(code) || name == "" {
		return Role{}, ErrInvalidRole
	}
	if rank < 0 || rank > 1000 {
		return Role{}, ErrInvalidRole
	}
	if err := s.AuthorizeNewRole(ctx, rank); err != nil {
		return Role{}, err
	}
	role, err := s.repository.CreateRole(ctx, CreateRoleInput{
		Code:        code,
		Name:        name,
		Description: description,
		Rank:        rank,
	})
	if err != nil {
		return Role{}, err
	}
	return role, nil
}

// UpdateRole edits a custom role's metadata. Built-in roles are immutable and
// the actor may not raise a role above their own rank.
func (s *Service) UpdateRole(
	ctx context.Context,
	code string,
	name string,
	description string,
	rank int,
) (Role, error) {
	if _, err := s.repository.FindRole(ctx, code); err != nil {
		return Role{}, err
	}
	if err := s.AuthorizeRoleEdit(ctx, code, rank); err != nil {
		return Role{}, err
	}
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" || rank < 0 || rank > 1000 {
		return Role{}, ErrInvalidRole
	}
	role, err := s.repository.UpdateRole(ctx, UpdateRoleInput{
		Code:        code,
		Name:        name,
		Description: description,
		Rank:        rank,
	})
	if err != nil {
		return Role{}, err
	}
	s.InvalidateRole(code)
	return role, nil
}

// DeleteRole removes a custom role that is not assigned to any user.
func (s *Service) DeleteRole(ctx context.Context, code string) error {
	existing, err := s.repository.FindRole(ctx, code)
	if err != nil {
		return err
	}
	if err := s.AuthorizeRoleEdit(ctx, code, existing.Rank); err != nil {
		return err
	}
	assigned, err := s.repository.CountRoleAssignments(ctx, code)
	if err != nil {
		return err
	}
	if assigned > 0 {
		return ErrRoleInUse
	}
	if err := s.repository.DeleteRole(ctx, code); err != nil {
		return err
	}
	s.InvalidateRole(code)
	return nil
}

// ReplaceRolePermissions validates and replaces a role's permission set.
func (s *Service) ReplaceRolePermissions(
	ctx context.Context,
	code string,
	codes []Permission,
) error {
	role, err := s.repository.FindRole(ctx, code)
	if err != nil {
		return err
	}
	validated, err := validatePermissions(codes)
	if err != nil {
		return err
	}
	if err := s.AuthorizeRoleEdit(ctx, code, role.Rank); err != nil {
		return err
	}
	if err := s.AuthorizePermissions(ctx, validated); err != nil {
		return err
	}
	if err := s.repository.ReplaceRolePermissions(ctx, code, validated); err != nil {
		return err
	}
	s.InvalidateRole(code)
	return nil
}

// UserOverrides returns the active and expired overrides for a user.
func (s *Service) UserOverrides(ctx context.Context, publicID string) ([]Override, error) {
	overrides, err := s.repository.UserOverrides(ctx, publicID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(overrides, func(i, j int) bool {
		return overrides[i].Permission < overrides[j].Permission
	})
	return overrides, nil
}

// UserOverridesMany returns overrides for several users in one query. Users
// without overrides map to an empty slice.
func (s *Service) UserOverridesMany(
	ctx context.Context,
	publicIDs []string,
) (map[string][]Override, error) {
	grouped, err := s.repository.UserOverridesMany(ctx, publicIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]Override, len(publicIDs))
	for _, publicID := range publicIDs {
		overrides := grouped[publicID]
		if overrides == nil {
			overrides = []Override{}
		}
		sort.SliceStable(overrides, func(i, j int) bool {
			return overrides[i].Permission < overrides[j].Permission
		})
		result[publicID] = overrides
	}
	return result, nil
}

// SetOverride validates and upserts a per-user permission override.
func (s *Service) SetOverride(
	ctx context.Context,
	publicID string,
	permission Permission,
	effect Effect,
	reason string,
	expiresAt *time.Time,
) (Override, error) {
	publicID = strings.TrimSpace(publicID)
	reason = strings.TrimSpace(reason)
	if publicID == "" || !IsKnown(permission) {
		return Override{}, ErrUnknownPermission
	}
	if !effect.Valid() || len(reason) > maxReasonLength {
		return Override{}, ErrInvalidOverride
	}
	if err := s.AuthorizePermissions(ctx, []Permission{permission}); err != nil {
		return Override{}, err
	}
	if err := s.AuthorizeUserChange(ctx, publicID); err != nil {
		return Override{}, err
	}
	override, err := s.repository.SetUserOverride(ctx, SetOverrideInput{
		UserPublicID: publicID,
		Permission:   permission,
		Effect:       effect,
		Reason:       reason,
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		return Override{}, err
	}
	s.Invalidate(publicID)
	return override, nil
}

// DeleteOverride removes a per-user override.
func (s *Service) DeleteOverride(
	ctx context.Context,
	publicID string,
	permission Permission,
) error {
	if publicID == "" || !IsKnown(permission) {
		return ErrUnknownPermission
	}
	if err := s.AuthorizePermissions(ctx, []Permission{permission}); err != nil {
		return err
	}
	if err := s.AuthorizeUserChange(ctx, publicID); err != nil {
		return err
	}
	if err := s.repository.DeleteUserOverride(ctx, publicID, permission); err != nil {
		return err
	}
	s.Invalidate(publicID)
	return nil
}

// AssignRole changes a user's role.
func (s *Service) AssignRole(ctx context.Context, publicID string, role string) error {
	publicID = strings.TrimSpace(publicID)
	role = strings.TrimSpace(strings.ToLower(role))
	if publicID == "" || role == "" {
		return ErrInvalidRole
	}
	if _, err := s.repository.FindRole(ctx, role); err != nil {
		return err
	}
	if err := s.AuthorizeRoleGrant(ctx, role); err != nil {
		return err
	}
	if err := s.AuthorizeUserChange(ctx, publicID); err != nil {
		return err
	}
	if err := s.repository.AssignUserRole(ctx, publicID, role); err != nil {
		return err
	}
	s.Invalidate(publicID)
	return nil
}

func validatePermissions(codes []Permission) ([]Permission, error) {
	seen := make(map[Permission]struct{}, len(codes))
	result := make([]Permission, 0, len(codes))
	for _, code := range codes {
		if !IsKnown(code) {
			return nil, ErrUnknownPermission
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}
