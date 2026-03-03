package controller

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"crisisecho/internal/apps/community/model"
	"crisisecho/internal/apps/community/service"
	// "crisisecho/internal/middleware"
)

// CommunityController handles HTTP requests for the community reports domain.
type CommunityController struct {
	svc service.CommunityService
}

// NewCommunityController constructs a CommunityController with the given service.
func NewCommunityController(svc service.CommunityService) *CommunityController {
	return &CommunityController{svc: svc}
}

// RegisterRoutes mounts community report routes. /near is before /:id per route ordering rule.
func (c *CommunityController) RegisterRoutes(router fiber.Router) {
	router.Get("/near", c.GetNear)
	router.Get("/", c.GetAll)
	router.Get("/:id", c.GetByID)
	// router.Post("/", middleware.FirebaseAuth(), c.CreateReport)
	router.Post("/", c.CreateReport)
}

// POST /api/community-reports
func (c *CommunityController) CreateReport(ctx *fiber.Ctx) error {
	var report model.CommunityReport
	if err := ctx.BodyParser(&report); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	// Populate UserID from auth token when middleware is enabled:
	// report.UserID = ctx.Locals("firebase_uid").(string)
	if err := c.svc.CreateReport(ctx.UserContext(), &report); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(report)
}

// GET /api/community-reports/near?lat=&lng=&radius=
func (c *CommunityController) GetNear(ctx *fiber.Ctx) error {
	lat, err := strconv.ParseFloat(ctx.Query("lat"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lat"})
	}
	lng, err := strconv.ParseFloat(ctx.Query("lng"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lng"})
	}
	radius, _ := strconv.ParseFloat(ctx.Query("radius", "50"), 64)

	reports, err := c.svc.GetNearby(ctx.UserContext(), lat, lng, radius)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if reports == nil {
		reports = []*model.CommunityReport{}
	}
	return ctx.JSON(reports)
}

// GET /api/community-reports/:id
func (c *CommunityController) GetByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	report, err := c.svc.GetByID(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(report)
}

// GET /api/community-reports
func (c *CommunityController) GetAll(ctx *fiber.Ctx) error {
	reports, err := c.svc.GetAll(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if reports == nil {
		reports = []*model.CommunityReport{}
	}
	return ctx.JSON(reports)
}
