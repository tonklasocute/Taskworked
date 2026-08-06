package report

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
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
	router.Get("/projects/:id/reports", h.periodReport)
	router.Get("/projects/:id/reports/export.csv", h.exportCSV)
}

func (h *Handler) periodReport(c *fiber.Ctx) error {
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}

	from, to, err := resolvePeriod(c.Query("period", "week"), c.Query("from"), c.Query("to"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, err.Error())
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	result, err := h.service.GetPeriodReport(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), projectID, from, to)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}
	return response.OK(c, result)
}

func (h *Handler) exportCSV(c *fiber.Ctx) error {
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Err(c, fiber.StatusBadRequest, "invalid project id")
	}

	actorID, err := httpctx.ActorID(c)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	data, err := h.service.ExportCSV(c.Context(), actorID, auth.Role(httpctx.ActorRole(c)), projectID)
	if err != nil {
		return httpctx.WriteErr(c, err)
	}

	c.Set(fiber.HeaderContentType, "text/csv")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="tasks.csv"`)
	return c.Send(data)
}

// resolvePeriod supports either an explicit from/to (YYYY-MM-DD) or a
// period=week|month shorthand — "Weekly Report" and "Monthly Report" from
// the spec are just these two presets over one underlying report.
func resolvePeriod(period, fromStr, toStr string) (from, to time.Time, err error) {
	if fromStr != "" && toStr != "" {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, errInvalidDate("from")
		}
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			return time.Time{}, time.Time{}, errInvalidDate("to")
		}
		return from, to, nil
	}

	now := time.Now()
	switch period {
	case "month":
		return now.AddDate(0, 0, -30), now, nil
	default:
		return now.AddDate(0, 0, -7), now, nil
	}
}

func errInvalidDate(field string) error {
	return errors.New("invalid " + field + ", expected YYYY-MM-DD")
}
