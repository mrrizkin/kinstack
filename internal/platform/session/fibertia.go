package session

import (
	"encoding/json"
	"fmt"

	"codeberg.org/mrrizkin/fibertia"
	"github.com/gofiber/fiber/v3"
	fibersession "github.com/gofiber/fiber/v3/middleware/session"
)

const (
	fibertiaFlashKey = "_fibertia_flash"
	fibertiaErrorKey = "_fibertia_errors_%s"
)

// InertiaSessionStore implements fibertia.SessionStore using Fiber's session
// store. It also works with the request-scoped session middleware when a
// handler calls Fibertia before the middleware has saved the response.
type InertiaSessionStore struct {
	store *fibersession.Store
}

func NewInertiaSessionStore(store *fibersession.Store) *InertiaSessionStore {
	return &InertiaSessionStore{store: store}
}

func (i *InertiaSessionStore) sess(c fiber.Ctx) (*fibersession.Session, bool, error) {
	if middleware := fibersession.FromContext(c); middleware != nil && middleware.Session != nil {
		return middleware.Session, true, nil
	}
	if i.store == nil {
		return nil, false, fmt.Errorf("session store is not configured")
	}
	sess, err := i.store.Get(c)
	return sess, false, err
}

func saveExternalSession(sess *fibersession.Session, managed bool) error {
	if managed {
		return nil
	}
	return sess.Save()
}

// GetFlash returns flash data and clears it from the current session.
func (i *InertiaSessionStore) GetFlash(c fiber.Ctx) (map[string]any, error) {
	sess, managed, err := i.sess(c)
	if err != nil {
		return nil, err
	}
	if !managed {
		defer sess.Release()
	}
	raw := sess.Get(fibertiaFlashKey)
	if raw == nil {
		return nil, nil
	}
	sess.Delete(fibertiaFlashKey)
	value, err := decodeAnyMap(raw)
	if err != nil {
		return nil, err
	}
	if err := saveExternalSession(sess, managed); err != nil {
		return nil, err
	}
	return value, nil
}

// SetFlash stores flash data for the next request.
func (i *InertiaSessionStore) SetFlash(c fiber.Ctx, data map[string]any) error {
	sess, managed, err := i.sess(c)
	if err != nil {
		return err
	}
	if !managed {
		defer sess.Release()
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	sess.Set(fibertiaFlashKey, string(encoded))
	return saveExternalSession(sess, managed)
}

// GetErrors returns validation errors and clears them from the current session.
func (i *InertiaSessionStore) GetErrors(c fiber.Ctx, bag string) (map[string][]string, error) {
	sess, managed, err := i.sess(c)
	if err != nil {
		return nil, err
	}
	if !managed {
		defer sess.Release()
	}
	key := fmt.Sprintf(fibertiaErrorKey, bag)
	raw := sess.Get(key)
	if raw == nil {
		return nil, nil
	}
	sess.Delete(key)
	value, err := decodeStringSliceMap(raw)
	if err != nil {
		return nil, err
	}
	if err := saveExternalSession(sess, managed); err != nil {
		return nil, err
	}
	return value, nil
}

// SetErrors stores validation errors for the next request.
func (i *InertiaSessionStore) SetErrors(
	c fiber.Ctx,
	bag string,
	validationErrors map[string][]string,
) error {
	sess, managed, err := i.sess(c)
	if err != nil {
		return err
	}
	if !managed {
		defer sess.Release()
	}
	encoded, err := json.Marshal(validationErrors)
	if err != nil {
		return err
	}
	sess.Set(fmt.Sprintf(fibertiaErrorKey, bag), string(encoded))
	return saveExternalSession(sess, managed)
}

// Reflash is intentionally a no-op because values remain in the Fiber session
// until the next request reads and clears them.
func (i *InertiaSessionStore) Reflash(fiber.Ctx) error {
	return nil
}

func decodeAnyMap(raw any) (map[string]any, error) {
	encoded, err := sessionValueString(raw)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return nil, err
	}
	return value, nil
}

func decodeStringSliceMap(raw any) (map[string][]string, error) {
	encoded, err := sessionValueString(raw)
	if err != nil {
		return nil, err
	}
	var value map[string][]string
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return nil, err
	}
	return value, nil
}

func sessionValueString(raw any) (string, error) {
	switch value := raw.(type) {
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	default:
		return "", fmt.Errorf("unexpected session value type %T", raw)
	}
}

var _ fibertia.SessionStore = (*InertiaSessionStore)(nil)
