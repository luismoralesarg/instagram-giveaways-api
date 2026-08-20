package fiber

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/application/usecase"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/infrastructure/instagram"
)

type WebhookHandler struct {
	handleStoryMention *usecase.HandleStoryMention
	verifyToken        string
	appSecret          string
}

func NewWebhookHandler(handleStoryMention *usecase.HandleStoryMention, verifyToken, appSecret string) *WebhookHandler {
	return &WebhookHandler{handleStoryMention: handleStoryMention, verifyToken: verifyToken, appSecret: appSecret}
}

// Verify implementa el challenge de suscripción de Meta: GET /webhooks/instagram.
func (h *WebhookHandler) Verify(c *fiber.Ctx) error {
	if c.Query("hub.mode") != "subscribe" || c.Query("hub.verify_token") != h.verifyToken {
		return fiber.NewError(fiber.StatusForbidden, "verify token inválido")
	}
	return c.SendString(c.Query("hub.challenge"))
}

// webhookPayload modela únicamente lo que necesitamos del evento de story
// mention (Messenger Platform / Instagram Messaging). El resto del payload
// se ignora.
type webhookPayload struct {
	Entry []struct {
		Messaging []struct {
			Sender struct {
				ID string `json:"id"`
			} `json:"sender"`
			Message struct {
				Attachments []struct {
					Type string `json:"type"`
				} `json:"attachments"`
			} `json:"message"`
		} `json:"messaging"`
	} `json:"entry"`
}

// Receive implementa POST /webhooks/instagram (UC-2.2): valida la firma
// (FA-2.2.1) y procesa cada mención de historia del payload.
func (h *WebhookHandler) Receive(c *fiber.Ctx) error {
	body := c.Body()

	if !instagram.VerifySignature(h.appSecret, body, c.Get("X-Hub-Signature-256")) {
		return fiber.NewError(fiber.StatusUnauthorized, "firma inválida")
	}

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "payload inválido")
	}

	for _, entry := range payload.Entry {
		for _, m := range entry.Messaging {
			for _, a := range m.Message.Attachments {
				if a.Type != "story_mention" {
					continue
				}
				if err := h.handleStoryMention.Execute(c.Context(), usecase.HandleStoryMentionInput{
					InstagramUserID: m.Sender.ID,
				}); err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "error interno")
				}
			}
		}
	}

	return c.SendStatus(fiber.StatusOK)
}
