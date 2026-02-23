package controller

import (
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
// /verified is registered before any wildcard to avoid route conflicts.
func (c *CrisisController) RegisterRoutes(router fiber.Router) {
	router.Get("/verified", c.GetVerifiedCrises)
	router.Get("/", c.GetAllCrises)
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
