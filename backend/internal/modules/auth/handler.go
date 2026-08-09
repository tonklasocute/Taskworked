package auth

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

// authLimiter is applied only to the unauthenticated, credential-guessable
// endpoints (register/login/refresh) — a stricter limit than the rest of
// the API, since these are the ones brute-force/credential-stuffing attacks
// target. logout doesn't need it: it already requires a valid access token.
func (h *Handler) RegisterRoutes(router fiber.Router, authRequired fiber.Handler, authLimiter fiber.Handler) {
	r := router.Group("/auth")
	r.Post("/register", authLimiter, h.register)
	r.Post("/login", authLimiter, h.login)
	r.Post("/refresh", authLimiter, h.refresh)
	r.Post("/logout", authRequired, h.logout)
}

// RegisterUserRoutes wires the org-wide directory endpoints. Unlike
// RegisterRoutes, every route here needs auth, so it's registered by the
// caller against an already-authenticated group (see main.go), matching
// how every other module's RegisterRoutes works.
func (h *Handler) RegisterUserRoutes(router fiber.Router) {
	router.Get("/users", h.listUsers)
	router.Patch("/users/:id/role", h.updateRole)
	router.Patch("/users/:id/department", h.updateDepartment)
}

func (h *Handler) listUsers(c *fiber.Ctx) error {
	result, err := h.service.ListUsers(c.Context())
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) updateRole(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid user id")
	}

	var req UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.UpdateRole(c.Context(), Role(httpctx.ActorRole(c)), id, req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) updateDepartment(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid user id")
	}

	var req UpdateDepartmentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.UpdateDepartment(c.Context(), Role(httpctx.ActorRole(c)), id, req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.Register(c.Context(), req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.Created(c, result)
}

func (h *Handler) login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.Login(c.Context(), req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) refresh(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.Err(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.Refresh(c.Context(), req)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) logout(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if err := h.service.Logout(c.Context(), userID); err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, fiber.Map{"message": "logged out"})
}
