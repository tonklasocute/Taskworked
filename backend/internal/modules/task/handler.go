package task

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
	r := router.Group("/tasks")
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Get("/:id", h.get)
	r.Patch("/:id", h.update)
	r.Delete("/:id", h.delete)
	r.Put("/:id/tags", h.setTags)

	r.Get("/:id/checklist", h.listChecklist)
	r.Post("/:id/checklist", h.addChecklistItem)
	r.Patch("/:id/checklist/:itemId", h.updateChecklistItem)
	r.Delete("/:id/checklist/:itemId", h.deleteChecklistItem)

	r.Post("/:id/dependencies", h.addDependency)
	r.Delete("/:id/dependencies/:dependsOnId", h.removeDependency)

	router.Get("/projects/:id/gantt", h.ganttView)
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

	result, err := h.service.Create(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.Created(c, result)
}

func (h *Handler) get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid task id")
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
		ProjectID:  c.Query("project_id"),
		Status:     Status(c.Query("status")),
		Priority:   Priority(c.Query("priority")),
		AssigneeID: c.Query("assignee_id"),
		Search:     c.Query("search"),
		Page:       page,
		PageSize:   pageSize,
	}

	result, err := h.service.List(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), filter)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid task id")
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
		return response.Err(c, fiber.StatusBadRequest, "invalid task id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	if err := h.service.Delete(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, fiber.Map{"message": "task deleted"})
}

func (h *Handler) setTags(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid task id")
	}

	var req SetTagsRequest
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

	tags, err := h.service.SetTags(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id, req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, fiber.Map{"tags": tags})
}

func (h *Handler) listChecklist(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid task id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	items, err := h.service.ListChecklistItems(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, items)
}

func (h *Handler) addChecklistItem(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid task id")
	}

	var req AddChecklistItemRequest
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

	item, err := h.service.AddChecklistItem(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id, req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.Created(c, item)
}

func (h *Handler) updateChecklistItem(c *fiber.Ctx) error {
	taskID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid task id")
	}
	itemID, err := uuid.Parse(c.Params("itemId"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid checklist item id")
	}

	var req UpdateChecklistItemRequest
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

	item, err := h.service.UpdateChecklistItem(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), taskID, itemID, req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, item)
}

func (h *Handler) deleteChecklistItem(c *fiber.Ctx) error {
	taskID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid task id")
	}
	itemID, err := uuid.Parse(c.Params("itemId"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid checklist item id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	if err := h.service.DeleteChecklistItem(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), taskID, itemID); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, fiber.Map{"message": "checklist item deleted"})
}

func (h *Handler) addDependency(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid task id")
	}

	var req AddDependencyRequest
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

	if err := h.service.AddDependency(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id, req); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.Created(c, fiber.Map{"message": "dependency added"})
}

func (h *Handler) removeDependency(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid task id")
	}
	dependsOnID, err := uuid.Parse(c.Params("dependsOnId"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid depends_on_task_id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	if err := h.service.RemoveDependency(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), id, dependsOnID); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, fiber.Map{"message": "dependency removed"})
}

func (h *Handler) ganttView(c *fiber.Ctx) error {
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	result, err := h.service.GetGanttView(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), projectID)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}
