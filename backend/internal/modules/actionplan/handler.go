package actionplan

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
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
	router.Post("/goals", h.createGoal)
	router.Patch("/goals/:id", h.updateGoal)
	router.Delete("/goals/:id", h.deleteGoal)

	router.Post("/milestones", h.createMilestone)
	router.Patch("/milestones/:id", h.updateMilestone)
	router.Delete("/milestones/:id", h.deleteMilestone)

	router.Get("/projects/:id/action-plan", h.actionPlanView)
}

func (h *Handler) createGoal(c *fiber.Ctx) error {
	var req CreateGoalRequest
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

	result, err := h.service.CreateGoal(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.Created(c, result)
}

func (h *Handler) updateGoal(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid goal id")
	}

	var req UpdateGoalRequest
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

	result, err := h.service.UpdateGoal(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id, req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) deleteGoal(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid goal id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	if err := h.service.DeleteGoal(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, fiber.Map{"message": "goal deleted"})
}

func (h *Handler) createMilestone(c *fiber.Ctx) error {
	var req CreateMilestoneRequest
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

	result, err := h.service.CreateMilestone(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.Created(c, result)
}

func (h *Handler) updateMilestone(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid milestone id")
	}

	var req UpdateMilestoneRequest
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

	result, err := h.service.UpdateMilestone(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id, req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) deleteMilestone(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid milestone id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	if err := h.service.DeleteMilestone(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, fiber.Map{"message": "milestone deleted"})
}

func (h *Handler) actionPlanView(c *fiber.Ctx) error {
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	result, err := h.service.GetActionPlanView(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), projectID)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}
