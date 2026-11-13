// Package authz owns hybrid RBAC + ABAC authorization. Permission codes are
// defined here; the database stores codes only.
package authz

import (
	"sort"

	"kinstack/internal/platform/i18n"
)

type Permission string

// Authenticated is a sentinel meaning "any authenticated actor". It is used for
// self-service routes that must not require a dedicated permission.
const Authenticated Permission = "authenticated"

// BrowserSession is a sentinel for "signed in through the browser session".
// Guard credential-management routes with it: a stolen API token is
// authenticated, so Authenticated would let it mint or rotate credentials.
const BrowserSession Permission = "browser_session"

const (
	PermissionUsersView   Permission = "users.view"
	PermissionUsersManage Permission = "users.manage"

	PermissionAccessManage Permission = "access.manage"

	PermissionAuditView Permission = "audit.view"

	PermissionMediaView   Permission = "media.view"
	PermissionMediaManage Permission = "media.manage"
)

const (
	CategorySystem = "System"
)

type Definition struct {
	Code        Permission
	Category    string
	Label       string
	Description string
}

var catalog = []Definition{
	{
		PermissionUsersView,
		CategorySystem,
		i18n.T("authz.users_view.label"),
		i18n.T("authz.users_view.description"),
	},
	{
		PermissionUsersManage,
		CategorySystem,
		i18n.T("authz.users_manage.label"),
		i18n.T("authz.users_manage.description"),
	},
	{
		PermissionAccessManage,
		CategorySystem,
		i18n.T("authz.access_manage.label"),
		i18n.T("authz.access_manage.description"),
	},
	{
		PermissionAuditView,
		CategorySystem,
		i18n.T("authz.audit_view.label"),
		i18n.T("authz.audit_view.description"),
	},
	{
		PermissionMediaView,
		CategorySystem,
		i18n.T("authz.media_view.label"),
		i18n.T("authz.media_view.description"),
	},
	{
		PermissionMediaManage,
		CategorySystem,
		i18n.T("authz.media_manage.label"),
		i18n.T("authz.media_manage.description"),
	},
}

func Catalog() []Definition {
	definitions := make([]Definition, len(catalog))
	copy(definitions, catalog)
	return definitions
}

func All() []Permission {
	codes := make([]Permission, 0, len(catalog))
	for _, definition := range catalog {
		codes = append(codes, definition.Code)
	}
	return codes
}

func Lookup(code Permission) (Definition, bool) {
	for _, definition := range catalog {
		if definition.Code == code {
			return definition, true
		}
	}
	return Definition{}, false
}

func IsKnown(code Permission) bool {
	_, ok := Lookup(code)
	return ok
}

func KnownCodes() map[Permission]struct{} {
	codes := make(map[Permission]struct{}, len(catalog))
	for _, definition := range catalog {
		codes[definition.Code] = struct{}{}
	}
	return codes
}

func CategoryOrder() []string {
	return []string{CategorySystem}
}

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
