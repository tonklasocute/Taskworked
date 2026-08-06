package response

import "github.com/gofiber/fiber/v2"

type Envelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func OK(c *fiber.Ctx, data any) error {
	return c.JSON(Envelope{Success: true, Data: data})
}

func Created(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusCreated).JSON(Envelope{Success: true, Data: data})
}

func Err(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(Envelope{Success: false, Error: msg})
}
