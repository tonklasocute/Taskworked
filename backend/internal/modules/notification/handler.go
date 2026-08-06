package notification

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/pkg/httpctx"
	"github.com/khomkrittk/taskworked/backend/internal/pkg/response"
)

type Handler struct {
	service  Service
	validate *validator.Validate
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service, validate: validator.New()}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/notifications", h.list)
	router.Post("/notifications/:id/read", h.markRead)
	router.Post("/notifications/read-all", h.markAllRead)
	router.Get("/notification-preferences", h.getPreference)
	router.Patch("/notification-preferences", h.updatePreference)
}

func (h *Handler) list(c *fiber.Ctx) error {
	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	result, err := h.service.ListNotifications(c.Context(), actorID)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) markRead(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid notification id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	if err := h.service.MarkRead(c.Context(), actorID, id); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, fiber.Map{"message": "marked read"})
}

func (h *Handler) markAllRead(c *fiber.Ctx) error {
	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	if err := h.service.MarkAllRead(c.Context(), actorID); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, fiber.Map{"message": "all marked read"})
}

func (h *Handler) getPreference(c *fiber.Ctx) error {
	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	result, err := h.service.GetPreference(c.Context(), actorID)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) updatePreference(c *fiber.Ctx) error {
	var req UpdatePreferenceRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, err.Error())
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	result, err := h.service.UpdatePreference(c.Context(), actorID, req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}
