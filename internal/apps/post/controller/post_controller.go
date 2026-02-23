package controller

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"crisisecho/internal/apps/post/model"
	"crisisecho/internal/apps/post/service"
)

// PostController handles HTTP requests for the post domain.
type PostController struct {
	svc service.PostService
}

// NewPostController constructs a PostController with the given service.
func NewPostController(svc service.PostService) *PostController {
	return &PostController{svc: svc}
}

// RegisterRoutes mounts the post routes on the provided router group.
func (c *PostController) RegisterRoutes(router fiber.Router) {
	router.Get("/nearby", c.GetNearbyPosts)
	router.Get("/recent", c.GetRecentPosts)
	router.Post("/", c.CreateUnifiedPost)
}

// GET /api/posts/nearby?lat=&lng=&radius=
// Returns unified posts within radius km of the given coordinates.
func (c *PostController) GetNearbyPosts(ctx *fiber.Ctx) error {
	lat, err := strconv.ParseFloat(ctx.Query("lat"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lat"})
	}
	lng, err := strconv.ParseFloat(ctx.Query("lng"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lng"})
	}
	radius, err := strconv.ParseFloat(ctx.Query("radius", "10"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid radius"})
	}

	posts, err := c.svc.GetNearbyPosts(ctx.UserContext(), lat, lng, radius)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if posts == nil {
		posts = []*model.UnifiedPost{}
	}
	return ctx.JSON(posts)
}

// GET /api/posts/recent?minutes=
// Returns recent relevant posts within the given number of minutes (default 30).
func (c *PostController) GetRecentPosts(ctx *fiber.Ctx) error {
	minutes, err := strconv.Atoi(ctx.Query("minutes", "30"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid minutes"})
	}

	posts, err := c.svc.GetRecentPosts(ctx.UserContext(), minutes)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if posts == nil {
		posts = []*model.UnifiedPost{}
	}
	return ctx.JSON(posts)
}

// POST /api/posts
// Internal endpoint called by the Python preprocessing pipeline to save a UnifiedPost.
func (c *PostController) CreateUnifiedPost(ctx *fiber.Ctx) error {
	var post model.UnifiedPost
	if err := ctx.BodyParser(&post); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := c.svc.CreateUnifiedPost(ctx.UserContext(), &post); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(post)
}
