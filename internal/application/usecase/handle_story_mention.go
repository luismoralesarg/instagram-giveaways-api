package usecase

import (
	"context"
	"errors"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type HandleStoryMentionInput struct {
	// InstagramUserID es lo único que trae de forma confiable el payload
	// del webhook de story mention (sender.id) — no incluye username.
	InstagramUserID string
}

// HandleStoryMention implementa UC-2.2. Agnóstico de HTTP: la validación
// de firma y el parseo del payload de Instagram viven en la capa de
// infraestructura (infrastructure/instagram y http/fiber/webhook_handlers.go).
type HandleStoryMention struct {
	campaigns    domain.CampaignRepository
	participants domain.ParticipantRepository
}

func NewHandleStoryMention(campaigns domain.CampaignRepository, participants domain.ParticipantRepository) *HandleStoryMention {
	return &HandleStoryMention{campaigns: campaigns, participants: participants}
}

func (uc *HandleStoryMention) Execute(ctx context.Context, in HandleStoryMentionInput) error {
	campaign, err := uc.campaigns.FindActiveStoryCampaign(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrCampaignNotFound) {
			return nil // FA-2.2.2: no hay campaña de historia activa, se descarta
		}
		return err
	}

	exists, err := uc.participants.ExistsByCampaignAndInstagramUserID(ctx, campaign.ID, in.InstagramUserID)
	if err != nil {
		return err
	}
	if exists {
		return nil // FA-2.2.3: ya registrado, se ignora sin duplicar
	}

	participant := &domain.Participant{
		CampaignID:      campaign.ID,
		InstagramUserID: in.InstagramUserID,
		SourceType:      domain.ParticipantSourceStoryMention,
	}
	if err := uc.participants.Create(ctx, participant); err != nil {
		if errors.Is(err, domain.ErrParticipantAlreadyExists) {
			// El pre-chequeo de arriba no vio nada porque perdió la carrera
			// contra otra entrega del mismo evento (Instagram reintenta
			// webhooks at-least-once) — mismo desenlace que FA-2.2.3.
			return nil
		}
		return err
	}
	return nil
}
