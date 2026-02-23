package controller

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"crisisecho/internal/apps/alert/model"
	"crisisecho/internal/apps/alert/service"
)

// AlertController handles HTTP requests for the alert domain.
type AlertController struct {
	svc service.AlertService
}

// NewAlertController constructs an AlertController with the given service.
func NewAlertController(svc service.AlertService) *AlertController {
	return &AlertController{svc: svc}
}

// RegisterRoutes mounts the alert routes on the provided router group.
// /recent is registered before / to avoid potential ordering issues.
func (c *AlertController) RegisterRoutes(router fiber.Router) {
	router.Get("/recent", c.GetRecentAlerts)
	router.Get("/", c.GetAllAlerts)
}

// GET /api/alerts
// Returns all published alerts.
func (c *AlertController) GetAllAlerts(ctx *fiber.Ctx) error {
	alerts, err := c.svc.GetAllAlerts(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if alerts == nil {
		alerts = []*model.Alert{}
	}
	return ctx.JSON(alerts)
}

// GET /api/alerts/recent?hours=
// Returns alerts published within the last N hours (default 24).
func (c *AlertController) GetRecentAlerts(ctx *fiber.Ctx) error {
	hours, err := strconv.Atoi(ctx.Query("hours", "24"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid hours"})
	}
	alerts, err := c.svc.GetRecentAlerts(ctx.UserContext(), hours)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if alerts == nil {
		alerts = []*model.Alert{}
	}
	return ctx.JSON(alerts)
}
