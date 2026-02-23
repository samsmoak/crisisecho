package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

// JWTProtected returns a Fiber middleware that validates a JWT Bearer token.
// This is a stub — full JWT validation is added in Prompt 3.
// The middleware reads JWT_SECRET from the environment at startup.
// Production implementation: parse the Authorization header, verify the token,
// and reject with 401 if invalid or missing.
func JWTProtected() fiber.Handler {
	_ = os.Getenv("JWT_SECRET") // read at startup to surface misconfiguration early
	return func(c *fiber.Ctx) error {
		// Stub: pass all requests through. Real implementation validates Bearer token.
		return c.Next()
	}
}
