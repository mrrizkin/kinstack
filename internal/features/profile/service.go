package profile

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/platform/i18n"
	"kinstack/internal/platform/password"
)

const (
	minDisplayNameLength = 3
	maxDisplayNameLength = 120
	minPasswordLength    = 8
	maxPasswordLength    = 72
)

// ServiceDependency contains the boundaries used by the profile service.
type ServiceDependency struct {
	Repository Repository
}

// Service owns profile business rules.
type Service struct {
	repository Repository
}

// NewService constructs a profile service around its repository.
func NewService(dependency ServiceDependency) *Service {
	assert.Assert(dependency.Repository != nil, "profile repository is required")
	return &Service{repository: dependency.Repository}
}

// Show returns the profile contract for the signed-in user.
func (s *Service) Show(ctx context.Context, publicID string) (ProfilePageProps, error) {
	result, err := s.repository.FindProfile(ctx, publicID)
	if err != nil {
		return ProfilePageProps{}, err
	}
	return ProfilePageProps{
		Profile: toProfileDTO(result),
		Preferences: PreferencesDTO{
			Theme:        result.ThemePreference,
			SoundEnabled: result.SoundEnabled,
		},
	}, nil
}

// UpdateDisplayName stores a new display name for the signed-in user.
func (s *Service) UpdateDisplayName(
	ctx context.Context,
	publicID string,
	request UpdateProfileRequest,
) (ProfileDTO, error) {
	displayName := strings.TrimSpace(request.DisplayName)
	if len(displayName) < minDisplayNameLength || len(displayName) > maxDisplayNameLength {
		return ProfileDTO{}, fmt.Errorf(
			"%w: %s",
			ErrInvalidInput,
			i18n.T("profile.display_name_length", minDisplayNameLength, maxDisplayNameLength),
		)
	}

	result, err := s.repository.UpdateDisplayName(ctx, publicID, displayName)
	if err != nil {
		return ProfileDTO{}, err
	}
	return toProfileDTO(result), nil
}

// UpdatePreferences stores personalization settings for the signed-in user.
func (s *Service) UpdatePreferences(
	ctx context.Context,
	publicID string,
	request UpdatePreferencesRequest,
) (PreferencesDTO, error) {
	theme := strings.ToLower(strings.TrimSpace(request.Theme))
	switch theme {
	case "light", "dark", "system":
	default:
		return PreferencesDTO{}, fmt.Errorf(
			"%w: %s",
			ErrInvalidInput,
			i18n.T("profile.theme_invalid"),
		)
	}

	result, err := s.repository.UpdatePreferences(ctx, publicID, theme, request.SoundEnabled)
	if err != nil {
		return PreferencesDTO{}, err
	}
	return PreferencesDTO{Theme: result.ThemePreference, SoundEnabled: result.SoundEnabled}, nil
}

// VerifyCurrentPassword confirms the supplied password for sensitive changes.
func (s *Service) VerifyCurrentPassword(
	ctx context.Context,
	publicID string,
	rawPassword string,
) error {
	if strings.TrimSpace(rawPassword) == "" {
		return fmt.Errorf("%w: %s", ErrInvalidInput, i18n.T("profile.password_required"))
	}
	currentHash, err := s.repository.FindPasswordHash(ctx, publicID)
	if err != nil {
		return err
	}
	if err := password.Compare(currentHash, rawPassword); err != nil {
		return ErrWrongPassword
	}
	return nil
}

// ChangePassword stores a new password. Accounts that are still invited may set
// their first password without supplying a current one.
func (s *Service) ChangePassword(
	ctx context.Context,
	publicID string,
	request ChangePasswordRequest,
) error {
	newPassword := request.NewPassword
	switch {
	case len(newPassword) < minPasswordLength:
		return fmt.Errorf("%w: %s", ErrInvalidInput, i18n.T("profile.new_password_min", minPasswordLength))
	case len(newPassword) > maxPasswordLength:
		return fmt.Errorf("%w: %s", ErrInvalidInput, i18n.T("profile.new_password_max", maxPasswordLength))
	case newPassword != request.ConfirmPassword:
		return ErrPasswordMismatch
	}

	profile, err := s.repository.FindProfile(ctx, publicID)
	if err != nil {
		return err
	}

	if !profile.Invited {
		if strings.TrimSpace(request.CurrentPassword) == "" {
			return fmt.Errorf("%w: %s", ErrInvalidInput, i18n.T("profile.current_password_required"))
		}
		currentHash, err := s.repository.FindPasswordHash(ctx, publicID)
		if err != nil {
			return err
		}
		if err := password.Compare(currentHash, request.CurrentPassword); err != nil {
			return ErrWrongPassword
		}
		if err := password.Compare(currentHash, newPassword); err == nil {
			return fmt.Errorf("%w: %s", ErrInvalidInput, i18n.T("profile.password_must_differ"))
		}
	}

	hash, err := password.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.repository.MarkPasswordSet(ctx, publicID, hash)
}

func toProfileDTO(result ProfileResult) ProfileDTO {
	lastLogin := ""
	if result.LastLoginAt != nil {
		lastLogin = result.LastLoginAt.UTC().Format(time.RFC3339)
	}
	return ProfileDTO{
		PublicID:    result.PublicID,
		Username:    result.Username,
		DisplayName: result.DisplayName,
		Role:        result.Role,
		Invited:     result.Invited,
		LastLoginAt: lastLogin,
		CreatedAt:   result.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// IsNotFound reports whether the error is a missing profile.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
