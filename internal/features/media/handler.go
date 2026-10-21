package media

import (
	"errors"
	"io"
	"strings"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/i18n"
	platforminertia "kinstack/internal/platform/inertia"

	"github.com/gofiber/fiber/v3"
)

// HandlerDependency contains the service, actor boundary, Inertia adapter, and
// logger used by the handler.
type HandlerDependency struct {
	Service *Service
	Actor   *actor.Actor
	Logger  log.Log
	Inertia *platforminertia.Adapter
}

// Handler owns the HTTP boundary for media management.
type Handler struct {
	service *Service
	actor   *actor.Actor
	logger  log.Log
	inertia *platforminertia.Adapter
}

// NewHandler constructs the media HTTP handler.
func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "media service is required")
	assert.Assert(d.Actor != nil, "media actor boundary is required")
	assert.Assert(d.Logger != nil, "media logger is required")
	assert.Assert(d.Inertia != nil, "media inertia adapter is required")

	return &Handler{service: d.Service, actor: d.Actor, logger: d.Logger, inertia: d.Inertia}
}

func (h *Handler) mediaPath() string {
	return "/media"
}

// Mount registers media routes on the guarded group.
func (h *Handler) Mount(router fiber.Router) {
	router.Get("/media", h.index)
	router.Get("/media/:publicID/file", h.file)
	router.Post("/media", h.upload)
	router.Post("/media/:publicID/delete", h.delete)
}

func (h *Handler) index(c fiber.Ctx) error {
	assets, err := h.service.List(c.Context(), h.actorPublicID(c))
	if err != nil {
		return h.internalServerError(c, "list media", err)
	}
	return c.Render("media/IndexView", fiber.Map{"assets": assets})
}

func (h *Handler) file(c fiber.Ctx) error {
	asset, body, err := h.service.Open(c.Context(), h.actorPublicID(c), c.Params("publicID"))
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": ErrNotFound.Error()})
		case errors.Is(err, ErrInvalidInput):
			return c.Status(fiber.StatusUnprocessableEntity).
				JSON(fiber.Map{"error": ErrInvalidInput.Error()})
		case errors.Is(err, ErrStorage):
			// The wrapped storage error carries bucket and object details; keep
			// them in the log and return the shared, generic message.
			h.logger.Errorw("media storage unavailable", "request_id", c.RequestID(), "error", err)
			return c.Status(fiber.StatusServiceUnavailable).
				JSON(fiber.Map{"error": ErrStorage.Error()})
		default:
			return h.internalServerError(c, "open media", err)
		}
	}
	defer func() { _ = body.Close() }()

	// Clamp to the upload ceiling: a corrupted row must not turn into an
	// unbounded read even though the recorded size says otherwise.
	limit := asset.SizeBytes + 1
	if limit > maxUploadSize+1 {
		limit = maxUploadSize + 1
	}
	data, err := io.ReadAll(io.LimitReader(body, limit))
	if err != nil {
		return h.internalServerError(c, "read media", err)
	}
	if int64(len(data)) != asset.SizeBytes {
		// The stored object no longer matches its recorded size. Serve what was
		// read but leave a trail: a truncated download is a storage incident.
		h.logger.Warnw(
			"media object size mismatch",
			"request_id", c.RequestID(),
			"public_id", asset.PublicID,
			"recorded_size", asset.SizeBytes,
			"read_size", len(data),
		)
	}
	c.Set(fiber.HeaderContentType, asset.ContentType)
	// Filenames are user supplied; quote and strip control characters so they
	// cannot break out of the Content-Disposition header.
	c.Set(
		fiber.HeaderContentDisposition,
		"inline; filename=\""+headerSafeFilename(asset.Filename)+"\"",
	)
	return c.Send(data)
}

// headerSafeFilename keeps a stored filename printable inside a quoted header
// parameter.
func headerSafeFilename(filename string) string {
	safe := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, filename)
	if safe == "" {
		return "download"
	}
	return safe
}

func (h *Handler) upload(c fiber.Ctx) error {
	if session := h.actor.ActorFromContext(c); session != nil && session.Superuser {
		return h.inertia.FailedAction(c,
			h.mediaPath(),
			fiber.StatusForbidden,
			i18n.T("media.upload_regular_only"),
		)
	}
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		return h.uploadValidationError(c, i18n.T("media.upload_required"))
	}
	body, err := file.Open()
	if err != nil {
		return h.uploadValidationError(c, i18n.T("media.unreadable"))
	}
	defer func() { _ = body.Close() }()

	if _, err := h.service.Upload(
		c.Context(),
		h.actorPublicID(c),
		file.Filename,
		file.Header.Get("Content-Type"),
		file.Size,
		body,
	); err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			return h.uploadValidationError(c, i18n.T("media.unsupported"))
		case errors.Is(err, ErrStorage):
			h.logger.Errorw("media storage unavailable", "request_id", c.RequestID(), "error", err)
			return h.uploadValidationError(c, i18n.T("media.storage_unavailable"))
		default:
			return h.internalServerError(c, "upload media", err)
		}
	}
	return h.inertia.RedirectWithFlash(c, h.mediaPath(), i18n.T("media.uploaded"))
}

func (h *Handler) uploadValidationError(c fiber.Ctx, message string) error {
	return h.inertia.ValidationErrors(c,
		h.mediaPath(),
		map[string]string{"file": message},
	)
}

func (h *Handler) delete(c fiber.Ctx) error {
	if err := h.service.Delete(c.Context(), h.actorPublicID(c), c.Params("publicID")); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return h.inertia.FailedAction(
				c,
				h.mediaPath(),
				fiber.StatusNotFound,
				ErrNotFound.Error(),
			)
		case errors.Is(err, ErrInvalidInput):
			return h.inertia.FailedAction(c,
				h.mediaPath(),
				fiber.StatusUnprocessableEntity,
				ErrInvalidInput.Error(),
			)
		case errors.Is(err, ErrStorage):
			h.logger.Errorw("media storage unavailable", "request_id", c.RequestID(), "error", err)
			return h.inertia.FailedAction(
				c,
				h.mediaPath(),
				fiber.StatusServiceUnavailable,
				ErrStorage.Error(),
			)
		default:
			return h.internalServerError(c, "delete media", err)
		}
	}
	return h.inertia.RedirectWithFlash(c, h.mediaPath(), i18n.T("media.deleted"))
}

func (h *Handler) actorPublicID(c fiber.Ctx) string {
	sessionActor := h.actor.ActorFromContext(c)
	if sessionActor == nil {
		return ""
	}
	return sessionActor.PublicID
}

func (h *Handler) internalServerError(c fiber.Ctx, message string, err error) error {
	h.logger.Errorw(message, "request_id", c.RequestID(), "error", err)
	return fiber.ErrInternalServerError
}
