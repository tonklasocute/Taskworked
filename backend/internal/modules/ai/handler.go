package ai

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
	router.Post("/ai/generate-tasks", h.generateTasks)
	router.Get("/ai/daily-summary", h.dailySummary)
	router.Get("/ai/projects/:projectId/weekly-summary", h.weeklySummary)
	router.Post("/ai/estimate-duration", h.estimateDuration)
	router.Post("/ai/suggest-priority", h.suggestPriority)
	router.Post("/ai/suggest-assignee", h.suggestAssignee)
	router.Get("/ai/projects/:projectId/predict-late", h.predictLateTasks)
	router.Get("/ai/projects/:projectId/productivity", h.analyzeProductivity)
	router.Post("/ai/meeting-summary", h.meetingSummary)
}

func (h *Handler) actor(c *fiber.Ctx) (uuid.UUID, auth.Role, error) {
	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return uuid.Nil, "", err
	}
	return actorID, auth.Role(httpctx.ActorRole(c)), nil
}

func (h *Handler) generateTasks(c *fiber.Ctx) error {
	actorID, actorRole, err := h.actor(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	var req GenerateTasksRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.GenerateTasks(c.Context(), actorID, actorRole, req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) dailySummary(c *fiber.Ctx) error {
	actorID, _, err := h.actor(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	result, err := h.service.SummarizeDailyTasks(c.Context(), actorID)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) weeklySummary(c *fiber.Ctx) error {
	actorID, actorRole, err := h.actor(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	projectID, err := uuid.Parse(c.Params("projectId"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}

	result, err := h.service.SummarizeWeeklyReport(c.Context(), actorID, actorRole, projectID)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) estimateDuration(c *fiber.Ctx) error {
	var req EstimateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.EstimateDuration(c.Context(), req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) suggestPriority(c *fiber.Ctx) error {
	var req PriorityRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.SuggestPriority(c.Context(), req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) suggestAssignee(c *fiber.Ctx) error {
	actorID, actorRole, err := h.actor(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	var req SuggestAssigneeRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.SuggestAssignee(c.Context(), actorID, actorRole, req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) predictLateTasks(c *fiber.Ctx) error {
	actorID, actorRole, err := h.actor(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	projectID, err := uuid.Parse(c.Params("projectId"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}

	result, err := h.service.PredictLateTasks(c.Context(), actorID, actorRole, projectID)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) analyzeProductivity(c *fiber.Ctx) error {
	actorID, actorRole, err := h.actor(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	projectID, err := uuid.Parse(c.Params("projectId"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}

	result, err := h.service.AnalyzeTeamProductivity(c.Context(), actorID, actorRole, projectID)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) meetingSummary(c *fiber.Ctx) error {
	var req MeetingSummaryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.GenerateMeetingSummary(c.Context(), req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}
