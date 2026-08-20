package fiber

import (
	"github.com/gofiber/fiber/v2"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type Handlers struct {
	Auth        *AuthHandler
	Campaign    *CampaignHandler
	Participant *ParticipantHandler
	Webhook     *WebhookHandler
	Draw        *DrawHandler
}

// NewRouter arma el árbol de rutas completo. Todas las rutas de
// administración requieren JWT (vía RequireAuth) salvo /auth/login y
// /webhooks/instagram (este último se valida con la firma de Instagram).
func NewRouter(app *fiber.App, h Handlers, tokens domain.TokenIssuer) {
	app.Post("/auth/login", h.Auth.Login)
	app.Get("/webhooks/instagram", h.Webhook.Verify)
	app.Post("/webhooks/instagram", h.Webhook.Receive)

	admin := app.Group("", RequireAuth(tokens))
	admin.Post("/campaigns", h.Campaign.Create)
	admin.Post("/campaigns/:id/activate", h.Campaign.Activate)
	admin.Post("/campaigns/:id/close", h.Campaign.Close)
	admin.Post("/campaigns/:id/sync-comments", h.Participant.SyncComments)
	admin.Get("/campaigns/:id/participants", h.Participant.List)
	admin.Patch("/campaigns/:id/participants/:pid/exclude", h.Participant.Exclude)
	admin.Post("/campaigns/:id/draws", h.Draw.Run)
	admin.Get("/campaigns/:id/draws/:draw_id", h.Draw.Get)
}
