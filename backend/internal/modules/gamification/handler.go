package gamification

import (
	"github.com/gofiber/fiber/v2"
	"github.com/khomkrittk/taskworked/backend/internal/pkg/httpctx"
	"github.com/khomkrittk/taskworked/backend/internal/pkg/response"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/gamification/profile", h.profile)
	router.Get("/gamification/leaderboard", h.leaderboard)
}

func (h *Handler) profile(c *fiber.Ctx) error {
	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	result, err := h.service.GetProfile(c.Context(), actorID)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) leaderboard(c *fiber.Ctx) error {
	result, err := h.service.GetLeaderboard(c.Context())
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}
