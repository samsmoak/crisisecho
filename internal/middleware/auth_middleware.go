package middleware

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"

	"crisisecho/internal/firebase"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// FirebaseAuth returns a Fiber middleware that verifies Firebase ID tokens.
// Tokens are read from the Authorization: Bearer <token> header.
// On success, the following values are stored in c.Locals():
//   - "firebase_uid"     (string) — Firebase UID
//   - "firebase_email"   (string) — user email from token claims
//   - "firebase_name"    (string) — display name from token claims
//   - "firebase_picture" (string) — profile picture URL from token claims
func FirebaseAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !firebase.IsAvailable() {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "auth service unavailable",
			})
		}

		idToken, err := extractBearerToken(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		token, err := firebase.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		c.Locals("firebase_uid", token.UID)
		if email, ok := token.Claims["email"].(string); ok {
			c.Locals("firebase_email", email)
		}
		if name, ok := token.Claims["name"].(string); ok {
			c.Locals("firebase_name", name)
		}
		if picture, ok := token.Claims["picture"].(string); ok {
			c.Locals("firebase_picture", picture)
		}

		return c.Next()
	}
}

// JWTAuth returns a Fiber middleware that verifies app-issued JWTs (from the auth service).
// On success, the following values are stored in c.Locals():
//   - "user_id"      (string) — MongoDB ObjectID hex
//   - "firebase_uid" (string) — Firebase UID
//   - "role"         (string) — user role (e.g. "free", "pro")
func JWTAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenStr, err := extractBearerToken(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return jwtSecret, nil
		})
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token",
			})
		}

		if userID, ok := claims["user_id"].(string); ok {
			c.Locals("user_id", userID)
		}
		if uid, ok := claims["firebase_uid"].(string); ok {
			c.Locals("firebase_uid", uid)
		}
		if role, ok := claims["role"].(string); ok {
			c.Locals("role", role)
		}

		return c.Next()
	}
}

// extractBearerToken pulls the token string from the Authorization header.
func extractBearerToken(c *fiber.Ctx) (string, error) {
	authHeader := c.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return "", errors.New("missing or malformed Authorization header")
	}
	return strings.TrimPrefix(authHeader, "Bearer "), nil
}
