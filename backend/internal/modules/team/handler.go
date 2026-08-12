package team

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
	router.Get("/team", h.directory)
	router.Post("/departments", h.createDepartment)
	router.Delete("/departments/:id", h.deleteDepartment)
}

func (h *Handler) directory(c *fiber.Ctx) error {
	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	result, err := h.service.GetDirectory(c.Context(), actorID)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) createDepartment(c *fiber.Ctx) error {
	var req CreateDepartmentRequest
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
	result, err := h.service.CreateDepartment(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.Created(c, result)
}

func (h *Handler) deleteDepartment(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid department id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	if err := h.service.DeleteDepartment(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, fiber.Map{"message": "department deleted"})
}
