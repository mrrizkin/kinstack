package bootstrap

import (
	"context"

	authfeature "kinstack/internal/features/auth"

	"github.com/gofiber/fiber/v3"
)

// passkeySessionStarter joins the passkeys slice to auth's session
// establishment. Only the composition root may connect two slices.
type passkeySessionStarter struct {
	auth *authfeature.Feature
}

func (s passkeySessionStarter) StartSession(
	ctx context.Context,
	c fiber.Ctx,
	publicID string,
) error {
	session, err := s.auth.Service().Session(ctx, publicID)
	if err != nil {
		return err
	}
	return s.auth.SetSession(c, session)
}
