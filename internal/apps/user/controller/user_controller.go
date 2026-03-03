package controller

import (
	"github.com/gofiber/fiber/v2"

	"crisisecho/internal/apps/user/model"
	"crisisecho/internal/apps/user/service"
)

// UserController handles HTTP requests for the user domain.
type UserController struct {
	svc service.UserService
}

// NewUserController constructs a UserController with the given service.
func NewUserController(svc service.UserService) *UserController {
	return &UserController{svc: svc}
}

// RegisterRoutes mounts the user routes on the provided router group.
func (c *UserController) RegisterRoutes(router fiber.Router) {
	router.Get("/", c.GetUser)
	router.Put("/", c.UpdateUser)
	router.Get("/locations", c.GetSavedLocations)
	router.Post("/locations", c.AddSavedLocation)
	router.Delete("/locations/:label", c.RemoveSavedLocation)
}

// GET /api/users
func (c *UserController) GetUser(ctx *fiber.Ctx) error {
	uid, _ := ctx.Locals("firebase_uid").(string)
	email, _ := ctx.Locals("firebase_email").(string)
	name, _ := ctx.Locals("firebase_name").(string)
	picture, _ := ctx.Locals("firebase_picture").(string)

	user, err := c.svc.GetOrCreateUser(ctx.UserContext(), uid, email, name, picture)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(user)
}

// PUT /api/users
func (c *UserController) UpdateUser(ctx *fiber.Ctx) error {
	uid, _ := ctx.Locals("firebase_uid").(string)

	var body struct {
		Name           string               `json:"name"`
		SavedLocations []model.SavedLocation `json:"saved_locations"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	user, err := c.svc.UpdateUser(ctx.UserContext(), uid, body.Name, body.SavedLocations)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(user)
}

// GET /api/users/locations
func (c *UserController) GetSavedLocations(ctx *fiber.Ctx) error {
	uid, _ := ctx.Locals("firebase_uid").(string)

	locs, err := c.svc.GetSavedLocations(ctx.UserContext(), uid)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if locs == nil {
		locs = []model.SavedLocation{}
	}
	return ctx.JSON(locs)
}

// POST /api/users/locations
func (c *UserController) AddSavedLocation(ctx *fiber.Ctx) error {
	uid, _ := ctx.Locals("firebase_uid").(string)

	var loc model.SavedLocation
	if err := ctx.BodyParser(&loc); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if loc.Label == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "label is required"})
	}

	user, err := c.svc.AddSavedLocation(ctx.UserContext(), uid, loc)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(user)
}

// DELETE /api/users/locations/:label
func (c *UserController) RemoveSavedLocation(ctx *fiber.Ctx) error {
	uid, _ := ctx.Locals("firebase_uid").(string)
	label := ctx.Params("label")

	user, err := c.svc.RemoveSavedLocation(ctx.UserContext(), uid, label)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(user)
}
