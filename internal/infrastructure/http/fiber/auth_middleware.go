package fiber

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

const localsUserIDKey = "userID"

// RequireAuth valida el Bearer token en el header Authorization contra
// domain.TokenIssuer. Se aplica a todas las rutas de administración,
// salvo /auth/login y /webhooks/instagram (CLAUDE.md).
func RequireAuth(tokens domain.TokenIssuer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			return fiber.NewError(fiber.StatusUnauthorized, "falta el header Authorization: Bearer <token>")
		}

		token := strings.TrimPrefix(header, prefix)
		userID, err := tokens.Parse(token)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "token inválido o expirado")
		}

		c.Locals(localsUserIDKey, userID)
		return c.Next()
	}
}
