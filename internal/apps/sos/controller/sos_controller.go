package controller

import (
	"github.com/gofiber/fiber/v2"

	"crisisecho/internal/apps/sos/model"
	"crisisecho/internal/apps/sos/service"
	// "crisisecho/internal/middleware"
)

// SOSController handles HTTP requests for the SOS domain.
type SOSController struct {
	svc service.SOSService
}

// NewSOSController constructs a SOSController with the given service.
func NewSOSController(svc service.SOSService) *SOSController {
	return &SOSController{svc: svc}
}

// RegisterRoutes mounts the SOS routes on the provided router group.
func (c *SOSController) RegisterRoutes(router fiber.Router) {
	// router.Post("/profiles", middleware.FirebaseAuth(), c.CreateProfile)
	// router.Get("/profiles", middleware.FirebaseAuth(), c.GetProfiles)
	// router.Put("/profiles/:id", middleware.FirebaseAuth(), c.UpdateProfile)
	// router.Delete("/profiles/:id", middleware.FirebaseAuth(), c.DeleteProfile)
	// router.Post("/trigger/:profileId", middleware.FirebaseAuth(), c.TriggerSOS)
	// router.Post("/resolve/:id", middleware.FirebaseAuth(), c.ResolveAlert)
	// router.Get("/active", middleware.FirebaseAuth(), c.GetActiveAlerts)
	router.Post("/profiles", c.CreateProfile)
	router.Get("/profiles", c.GetProfiles)
	router.Put("/profiles/:id", c.UpdateProfile)
	router.Delete("/profiles/:id", c.DeleteProfile)
	router.Post("/trigger/:profileId", c.TriggerSOS)
	router.Post("/resolve/:id", c.ResolveAlert)
	router.Get("/active", c.GetActiveAlerts)
}

// POST /api/sos/profiles
func (c *SOSController) CreateProfile(ctx *fiber.Ctx) error {
	var profile model.SOSProfile
	if err := ctx.BodyParser(&profile); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	// profile.UserID = ctx.Locals("firebase_uid").(string)
	if err := c.svc.CreateProfile(ctx.UserContext(), &profile); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(profile)
}

// GET /api/sos/profiles
func (c *SOSController) GetProfiles(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("firebase_uid").(string)
	if userID == "" {
		userID = ctx.Query("user_id")
	}
	profiles, err := c.svc.GetProfilesByUser(ctx.UserContext(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if profiles == nil {
		profiles = []*model.SOSProfile{}
	}
	return ctx.JSON(profiles)
}

// PUT /api/sos/profiles/:id
func (c *SOSController) UpdateProfile(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var profile model.SOSProfile
	if err := ctx.BodyParser(&profile); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := c.svc.UpdateProfile(ctx.UserContext(), id, &profile); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(profile)
}

// DELETE /api/sos/profiles/:id
func (c *SOSController) DeleteProfile(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.svc.DeleteProfile(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// POST /api/sos/trigger/:profileId
func (c *SOSController) TriggerSOS(ctx *fiber.Ctx) error {
	profileID := ctx.Params("profileId")
	userID, _ := ctx.Locals("firebase_uid").(string)

	var body struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	alert, err := c.svc.TriggerSOS(ctx.UserContext(), profileID, userID, body.Lat, body.Lng)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(alert)
}

// POST /api/sos/resolve/:id
func (c *SOSController) ResolveAlert(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.svc.ResolveSOSAlert(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// GET /api/sos/active
func (c *SOSController) GetActiveAlerts(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("firebase_uid").(string)
	if userID == "" {
		userID = ctx.Query("user_id")
	}
	alerts, err := c.svc.GetActiveAlerts(ctx.UserContext(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if alerts == nil {
		alerts = []*model.SOSAlert{}
	}
	return ctx.JSON(alerts)
}
