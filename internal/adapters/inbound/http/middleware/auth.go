package middleware

import (
	"fiber_test/internal/adapters/inbound/http/handlers"
	"fiber_test/internal/application/services"

	"github.com/gofiber/fiber/v3"
)

// RequireAuth rejects requests without a valid session and, on success, makes
// the account available via handlers.UserLocalsKey.
func RequireAuth(auth *services.AuthService) fiber.Handler {
	return func(c fiber.Ctx) error {
		sid := c.Cookies(handlers.SessionCookieName)
		if sid == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		user, err := auth.CurrentUser(c.Context(), sid)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		c.Locals(handlers.UserLocalsKey, user)
		return c.Next()
	}
}
