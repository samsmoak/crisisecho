package controller

import (
	"github.com/gofiber/fiber/v2"

	"crisisecho/internal/apps/user/service"
)

type UserController struct {
	svc service.UserService
}

func NewUserController(svc service.UserService) *UserController {
	return &UserController{svc: svc}
}

func (c *UserController) RegisterRoutes(router fiber.Router) {
	router.Get("/", c.GetUser)
	router.Put("/", c.UpdateUser)
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
		Name string `json:"name"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	user, err := c.svc.UpdateUser(ctx.UserContext(), uid, body.Name)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(user)
}
