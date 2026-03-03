package controller

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"

	"crisisecho/internal/apps/analytics/service"
)

// AnalyticsController handles HTTP requests for the analytics domain.
type AnalyticsController struct {
	svc service.AnalyticsService
}

// NewAnalyticsController constructs an AnalyticsController with the given service.
func NewAnalyticsController(svc service.AnalyticsService) *AnalyticsController {
	return &AnalyticsController{svc: svc}
}

// RegisterRoutes mounts the analytics routes on the provided router group.
func (c *AnalyticsController) RegisterRoutes(router fiber.Router) {
	router.Get("/trends", c.GetTrends)
	router.Get("/heatmap", c.GetHeatmap)
}

// GET /api/analytics/trends?event_type=&days=30
func (c *AnalyticsController) GetTrends(ctx *fiber.Ctx) error {
	eventType := ctx.Query("event_type")
	days, _ := strconv.Atoi(ctx.Query("days", "30"))

	results, err := c.svc.GetTrend(ctx.UserContext(), eventType, days)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if results == nil {
		results = []bson.M{}
	}
	return ctx.JSON(results)
}

// GET /api/analytics/heatmap?lat=&lng=&radius=
func (c *AnalyticsController) GetHeatmap(ctx *fiber.Ctx) error {
	lat, err := strconv.ParseFloat(ctx.Query("lat"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lat"})
	}
	lng, err := strconv.ParseFloat(ctx.Query("lng"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lng"})
	}
	radius, _ := strconv.ParseFloat(ctx.Query("radius", "50"), 64)

	results, err := c.svc.GetHeatmap(ctx.UserContext(), lat, lng, radius)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if results == nil {
		results = []bson.M{}
	}
	return ctx.JSON(results)
}
