package controller

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"crisisecho/internal/apps/notify/model"
	"crisisecho/internal/apps/notify/service"
	"crisisecho/internal/geo"
)

// NotifyController handles HTTP requests for the notify domain.
type NotifyController struct {
	svc service.NotifyService
}

// NewNotifyController constructs a NotifyController with the given service.
func NewNotifyController(svc service.NotifyService) *NotifyController {
	return &NotifyController{svc: svc}
}

// RegisterRoutes mounts the notify routes directly on the provided router.
// Routes mount at /subscribe (not under a sub-group) per the spec.
func (c *NotifyController) RegisterRoutes(router fiber.Router) {
	router.Post("/", c.Subscribe)
	router.Delete("/:id", c.Unsubscribe)
}

// POST /api/subscribe
// Creates a new alert subscription for the given user and location.
func (c *NotifyController) Subscribe(ctx *fiber.Ctx) error {
	var body struct {
		UserID      string   `json:"user_id"`
		Lat         float64  `json:"lat"`
		Lng         float64  `json:"lng"`
		RadiusKm    float64  `json:"radius_km"`
		CrisisTypes []string `json:"crisis_types"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.UserID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id is required"})
	}
	if body.RadiusKm <= 0 {
		body.RadiusKm = 10 // default 10 km
	}

	sub := &model.Subscription{
		UserID:      body.UserID,
		Location:    geo.NewPoint(body.Lat, body.Lng),
		RadiusKm:    body.RadiusKm,
		CrisisTypes: body.CrisisTypes,
		Active:      true,
		CreatedAt:   time.Now().UTC(),
	}
	if err := c.svc.Subscribe(ctx.UserContext(), sub); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(sub)
}

// DELETE /api/subscribe/:id
// Deactivates or removes an existing subscription.
func (c *NotifyController) Unsubscribe(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.svc.Unsubscribe(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
