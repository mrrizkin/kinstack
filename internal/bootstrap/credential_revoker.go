package bootstrap

import (
	"context"
	"errors"

	"kinstack/internal/features/apitoken"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/platform/sessionregistry"
)

// credentialRevoker revokes every session and personal API token of an account.
// Password resets (self-service and administrative) use it so credentials that
// predate the reset cannot keep acting on the account.
//
// Services are attached after construction because the composition root builds
// the session registry before the auth feature and the API token feature after
// it. Wiring finishes before the server accepts requests.
type credentialRevoker struct {
	log      applog.Log
	sessions *sessionregistry.Service
	tokens   *apitoken.Service
}

var _ interface {
	RevokeAll(ctx context.Context, userPublicID string) error
} = (*credentialRevoker)(nil)

// RevokeAll signs the account out everywhere. Both revocations run so a failure
// in one does not skip the other; the joined error aborts the credential change
// so the caller can retry.
func (r *credentialRevoker) RevokeAll(ctx context.Context, userPublicID string) error {
	var errs []error
	if r.sessions != nil {
		if err := r.sessions.RevokeAll(ctx, userPublicID); err != nil {
			errs = append(errs, err)
		}
	}
	if r.tokens != nil {
		if err := r.tokens.RevokeAll(ctx, userPublicID); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		if r.log != nil {
			r.log.Errorw(
				"credential revocation failed",
				"user", userPublicID,
				"error", errors.Join(errs...),
			)
		}
		return errors.Join(errs...)
	}
	return nil
}
