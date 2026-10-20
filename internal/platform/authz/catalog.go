// Package authz owns hybrid RBAC + ABAC authorization for KinStack.
//
// RBAC provides the baseline: a role grants a set of permission codes, and a
// user may carry explicit allow/deny overrides for individual permissions. ABAC
// adds attribute guardrails on top: registered policies can inspect the request
// (principal attributes plus resource attributes such as ownership, amount, or
// shift state) and deny or, in narrow cases, grant a decision.
//
// Permission codes are the single source of truth in this package. The database
// stores codes only; the catalog provides the label, category and description
// used by the access management UI.
//
// Feature slices register their own permission codes here as they are built.
package authz

import (
	"sort"

	"kinstack/internal/platform/i18n"
)

// Permission is a fine-grained capability code, e.g. "users.manage".
type Permission string

// Authenticated is a sentinel meaning "any authenticated actor". It is used for
// self-service routes that must not require a dedicated permission.
const Authenticated Permission = "authenticated"

// Permission codes recognised by the application.
const (
	PermissionUsersView   Permission = "users.view"
	PermissionUsersManage Permission = "users.manage"

	PermissionAccessManage Permission = "access.manage"

	PermissionAuditView Permission = "audit.view"

	PermissionMediaView   Permission = "media.view"
	PermissionMediaManage Permission = "media.manage"
)

// Categories group permissions in the access management UI.
const (
	CategorySystem = "System"
)

// Definition describes one permission exposed by the catalog.
type Definition struct {
	Code        Permission
	Category    string
	Label       string
	Description string
}

// catalog is the ordered, authoritative permission catalog.
var catalog = []Definition{
	{PermissionUsersView, CategorySystem, i18n.T("authz.users_view.label"), i18n.T("authz.users_view.description")},
	{PermissionUsersManage, CategorySystem, i18n.T("authz.users_manage.label"), i18n.T("authz.users_manage.description")},
	{PermissionAccessManage, CategorySystem, i18n.T("authz.access_manage.label"), i18n.T("authz.access_manage.description")},
	{PermissionAuditView, CategorySystem, i18n.T("authz.audit_view.label"), i18n.T("authz.audit_view.description")},
	{PermissionMediaView, CategorySystem, i18n.T("authz.media_view.label"), i18n.T("authz.media_view.description")},
	{PermissionMediaManage, CategorySystem, i18n.T("authz.media_manage.label"), i18n.T("authz.media_manage.description")},
}

// Catalog returns a copy of the ordered permission catalog.
func Catalog() []Definition {
	definitions := make([]Definition, len(catalog))
	copy(definitions, catalog)
	return definitions
}

// All returns every known permission code in catalog order.
func All() []Permission {
	codes := make([]Permission, 0, len(catalog))
	for _, definition := range catalog {
		codes = append(codes, definition.Code)
	}
	return codes
}

// Lookup resolves a permission code to its catalog definition.
func Lookup(code Permission) (Definition, bool) {
	for _, definition := range catalog {
		if definition.Code == code {
			return definition, true
		}
	}
	return Definition{}, false
}

// IsKnown reports whether the permission exists in the catalog (excluding the
// Authenticated sentinel).
func IsKnown(code Permission) bool {
	_, ok := Lookup(code)
	return ok
}

// KnownCodes returns the catalog as a set, useful for validation.
func KnownCodes() map[Permission]struct{} {
	codes := make(map[Permission]struct{}, len(catalog))
	for _, definition := range catalog {
		codes[definition.Code] = struct{}{}
	}
	return codes
}

// CategoryOrder returns the canonical UI ordering of permission categories.
func CategoryOrder() []string {
	return []string{CategorySystem}
}

// SortByCategory returns catalog definitions ordered by category then label.
func SortByCategory() []Definition {
	definitions := Catalog()
	order := make(map[string]int, len(CategoryOrder()))
	for index, category := range CategoryOrder() {
		order[category] = index
	}
	sort.SliceStable(definitions, func(i, j int) bool {
		left, right := order[definitions[i].Category], order[definitions[j].Category]
		if left != right {
			return left < right
		}
		return definitions[i].Label < definitions[j].Label
	})
	return definitions
}
