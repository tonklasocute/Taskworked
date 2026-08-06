package project

import (
	"strconv"

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
	r := router.Group("/projects")
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Get("/:id", h.get)
	r.Patch("/:id", h.update)
	r.Delete("/:id", h.delete)
	r.Post("/:id/members", h.addMember)
	r.Get("/:id/members", h.listMembers)
	r.Delete("/:id/members/:userId", h.removeMember)
}

func (h *Handler) create(c *fiber.Ctx) error {
	var req CreateRequest
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

	result, err := h.service.Create(c.Context(), actorID, req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.Created(c, result)
}

func (h *Handler) get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	result, err := h.service.Get(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) list(c *fiber.Ctx) error {
	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	filter := ListFilter{
		Status:   Status(c.Query("status")),
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.service.List(c.Context(), actorID, filter)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}

	var req UpdateRequest
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

	result, err := h.service.Update(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id, req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	if err := h.service.Delete(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, fiber.Map{"message": "project deleted"})
}

func (h *Handler) addMember(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}

	var req AddMemberRequest
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

	if err := h.service.AddMember(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id, req); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.Created(c, fiber.Map{"message": "member added"})
}

func (h *Handler) listMembers(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	result, err := h.service.ListMembers(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) removeMember(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}
	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid user id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	if err := h.service.RemoveMember(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id, targetUserID); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, fiber.Map{"message": "member removed"})
}
