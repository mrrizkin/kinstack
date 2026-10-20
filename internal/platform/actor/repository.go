package actor

import "context"

// Repository is the persistence boundary owned by the actor platform.
// Feature slices must not be used to resolve an authenticated actor.
type Repository interface {
	FindByPublicID(context.Context, string) error
}
