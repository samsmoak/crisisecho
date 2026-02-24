package controller

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"crisisecho/internal/apps/crisis/model"
	"crisisecho/internal/apps/crisis/service"
)

// CrisisController handles HTTP requests for the crisis domain.
type CrisisController struct {
	svc service.CrisisService
}

// NewCrisisController constructs a CrisisController with the given service.
func NewCrisisController(svc service.CrisisService) *CrisisController {
	return &CrisisController{svc: svc}
}

// RegisterRoutes mounts the crisis routes on the provided router group.
// /near and /verified are registered before / to avoid route conflicts.
func (c *CrisisController) RegisterRoutes(router fiber.Router) {
	router.Get("/near", c.GetNear)
	router.Get("/verified", c.GetVerifiedCrises)
	router.Get("/", c.GetAllCrises)
}

// GET /api/crises/near?lat=&lng=&radius=
// Returns crisis events near the given coordinates. Primary map data source.
func (c *CrisisController) GetNear(ctx *fiber.Ctx) error {
	lat, err := strconv.ParseFloat(ctx.Query("lat"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lat"})
	}
	lng, err := strconv.ParseFloat(ctx.Query("lng"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lng"})
	}
	radius, err := strconv.ParseFloat(ctx.Query("radius", "50"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid radius"})
	}

	crises, err := c.svc.GetNearby(ctx.UserContext(), lat, lng, radius)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if crises == nil {
		crises = []*model.Crisis{}
	}
	return ctx.JSON(crises)
}

// GET /api/crises/verified
// Returns only confirmed crisis events.
func (c *CrisisController) GetVerifiedCrises(ctx *fiber.Ctx) error {
	crises, err := c.svc.GetVerifiedCrises(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if crises == nil {
		crises = []*model.Crisis{}
	}
	return ctx.JSON(crises)
}

// GET /api/crises
// Returns all crisis events regardless of confirmation status.
func (c *CrisisController) GetAllCrises(ctx *fiber.Ctx) error {
	crises, err := c.svc.GetAllCrises(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if crises == nil {
		crises = []*model.Crisis{}
	}
	return ctx.JSON(crises)
}
