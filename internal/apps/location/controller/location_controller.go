package controller

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/apps/location/model"
	"crisisecho/internal/apps/location/service"
	"crisisecho/internal/middleware"
)

type LocationController struct {
	svc service.LocationService
}

func NewLocationController(svc service.LocationService) *LocationController {
	return &LocationController{svc: svc}
}

func (c *LocationController) RegisterRoutes(router fiber.Router) {
	auth := router.Group("", middleware.JWTAuth())
	auth.Post("/", c.Create)
	auth.Get("/", c.GetByUser)
	auth.Put("/:id", c.Update)
	auth.Delete("/:id", c.Delete)
}

// userOID extracts the MongoDB ObjectID from the JWT "user_id" claim.
func userOID(c *fiber.Ctx) (primitive.ObjectID, error) {
	uid, _ := c.Locals("user_id").(string)
	return primitive.ObjectIDFromHex(uid)
}

// POST /api/locations
func (c *LocationController) Create(ctx *fiber.Ctx) error {
	userID, err := userOID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user"})
	}

	var body struct {
		Label    string  `json:"label"`
		Lat      float64 `json:"lat"`
		Lng      float64 `json:"lng"`
		RadiusKm float64 `json:"radius_km"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Label == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "label is required"})
	}

	loc, err := c.svc.Create(ctx.UserContext(), userID, body.Label, body.Lat, body.Lng, body.RadiusKm)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(loc)
}

// GET /api/locations
func (c *LocationController) GetByUser(ctx *fiber.Ctx) error {
	userID, err := userOID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user"})
	}

	locs, err := c.svc.GetByUser(ctx.UserContext(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if locs == nil {
		locs = []*model.SavedLocation{}
	}
	return ctx.JSON(locs)
}

// PUT /api/locations/:id
func (c *LocationController) Update(ctx *fiber.Ctx) error {
	userID, err := userOID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user"})
	}

	var body struct {
		Label    string  `json:"label"`
		Lat      float64 `json:"lat"`
		Lng      float64 `json:"lng"`
		RadiusKm float64 `json:"radius_km"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	loc, err := c.svc.Update(ctx.UserContext(), ctx.Params("id"), userID, body.Label, body.Lat, body.Lng, body.RadiusKm)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(loc)
}

// DELETE /api/locations/:id
func (c *LocationController) Delete(ctx *fiber.Ctx) error {
	userID, err := userOID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user"})
	}

	if err := c.svc.Delete(ctx.UserContext(), ctx.Params("id"), userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
