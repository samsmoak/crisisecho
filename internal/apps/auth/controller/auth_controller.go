package controller

import (
	"github.com/gofiber/fiber/v2"

	"crisisecho/internal/apps/auth/model"
	"crisisecho/internal/apps/auth/service"
)

// AuthController handles HTTP requests for the auth domain.
type AuthController struct {
	svc service.AuthService
}

// NewAuthController constructs an AuthController with the given service.
func NewAuthController(svc service.AuthService) *AuthController {
	return &AuthController{svc: svc}
}

// RegisterRoutes mounts the auth routes on the provided router group.
func (c *AuthController) RegisterRoutes(router fiber.Router) {
	router.Post("/google", c.GoogleAuth)
	router.Post("/apple", c.AppleAuth)
	// Placeholder phone-based OTP routes:
	// router.Post("/phone/send-otp", c.SendOTP)
	// router.Post("/phone/verify", c.VerifyPhoneOTP)
}

// POST /api/auth/google
//
// Authenticates a user via Google Sign-In (Firebase ID token).
// Creates the user on first login, returns existing user otherwise.
func (c *AuthController) GoogleAuth(ctx *fiber.Ctx) error {
	var req model.GoogleAuthRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if req.IDToken == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "id_token is required",
		})
	}

	token, user, err := c.svc.GoogleAuth(ctx.UserContext(), req.IDToken)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(model.AuthResponse{
		Token: token,
		User:  user,
	})
}

// POST /api/auth/apple
//
// Authenticates a user via Apple Sign-In (placeholder — returns 501).
func (c *AuthController) AppleAuth(ctx *fiber.Ctx) error {
	var req model.AppleAuthRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	token, user, err := c.svc.AppleAuth(ctx.UserContext(), req.IDToken)
	if err != nil {
		return ctx.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(model.AuthResponse{
		Token: token,
		User:  user,
	})
}
