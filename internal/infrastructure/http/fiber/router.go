package fiber

import (
	"github.com/gofiber/fiber/v2"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type Handlers struct {
	Auth *AuthHandler
}

// NewRouter arma el árbol de rutas. Todas las rutas de administración
// requieren JWT (vía RequireAuth) salvo /auth/login y /webhooks/instagram
// (este último se valida con la firma de Instagram). Las rutas de
// campañas/participantes/sorteos se agregan en las próximas fases.
func NewRouter(app *fiber.App, h Handlers, tokens domain.TokenIssuer) {
	app.Post("/auth/login", h.Auth.Login)
}
