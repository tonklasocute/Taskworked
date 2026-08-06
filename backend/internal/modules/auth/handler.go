package auth

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
	"github.com/khomkrittk/taskworked/backend/internal/pkg/response"
)

type Handler struct {
	service  Service
	validate *validator.Validate
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service, validate: validator.New()}
}

func (h *Handler) RegisterRoutes(router fiber.Router, authRequired fiber.Handler) {
	r := router.Group("/auth")
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.Post("/refresh", h.refresh)
	r.Post("/logout", authRequired, h.logout)
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
		return writeErr(c, err)
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
		return writeErr(c, err)
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
		return writeErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) logout(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	if err := h.service.Logout(c.Context(), userID); err != nil {
		return writeErr(c, err)
	}
	return response.OK(c, fiber.Map{"message": "logged out"})
}

func writeErr(c *fiber.Ctx, err error) error {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return response.Err(c, appErr.Status, appErr.Message)
	}
	return response.Err(c, fiber.StatusInternalServerError, "internal server error")
}
