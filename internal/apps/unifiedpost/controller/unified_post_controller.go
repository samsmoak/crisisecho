package controller

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"crisisecho/internal/apps/unifiedpost/model"
	"crisisecho/internal/apps/unifiedpost/service"
)

// UnifiedPostController handles HTTP requests for the unified post domain.
type UnifiedPostController struct {
	svc service.UnifiedPostService
}

// NewUnifiedPostController constructs a UnifiedPostController with the given service.
func NewUnifiedPostController(svc service.UnifiedPostService) *UnifiedPostController {
	return &UnifiedPostController{svc: svc}
}

// RegisterRoutes mounts the unified post routes on the provided router group.
// /near is registered before /:id to prevent "near" being captured as an ID param.
func (c *UnifiedPostController) RegisterRoutes(router fiber.Router) {
	router.Get("/near", c.GetNear)
	router.Get("/:id", c.GetByID)
}

// GET /api/unified-posts/near?lat=&lng=&radius=
// Returns unified posts near the given coordinates. Used by the frontend map.
func (c *UnifiedPostController) GetNear(ctx *fiber.Ctx) error {
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

	posts, err := c.svc.GetNearby(ctx.UserContext(), lat, lng, radius)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if posts == nil {
		posts = []*model.UnifiedPost{}
	}
	return ctx.JSON(posts)
}

// GET /api/unified-posts/:id
// Returns a single unified post by ID. Called when the user clicks a crisis dot on the map.
func (c *UnifiedPostController) GetByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	post, err := c.svc.GetUnifiedPost(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(post)
}
