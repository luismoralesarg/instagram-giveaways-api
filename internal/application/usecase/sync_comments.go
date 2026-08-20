package usecase

import (
	"context"
	"errors"
	"regexp"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type SyncCommentsInput struct {
	CampaignID string
}

type SyncCommentsOutput struct {
	NewParticipants int
}

// SyncComments implementa UC-2.1.
//
// NOTA sobre must_follow (decisión del usuario): la Graph API no permite
// verificar si un comentarista arbitrario sigue la cuenta sin el token de
// ESE usuario, así que esta regla no se aplica acá como filtro automático
// — queda documentada en Campaign.MustFollow solo a título informativo.
type SyncComments struct {
	campaigns    domain.CampaignRepository
	participants domain.ParticipantRepository
	instagram    domain.InstagramClient
}

func NewSyncComments(campaigns domain.CampaignRepository, participants domain.ParticipantRepository, instagram domain.InstagramClient) *SyncComments {
	return &SyncComments{campaigns: campaigns, participants: participants, instagram: instagram}
}

var mentionPattern = regexp.MustCompile(`@[\w.]+`)

func countMentions(text string) int {
	return len(mentionPattern.FindAllString(text, -1))
}

func (uc *SyncComments) Execute(ctx context.Context, in SyncCommentsInput) (SyncCommentsOutput, error) {
	campaign, err := uc.campaigns.FindByID(ctx, in.CampaignID)
	if err != nil {
		return SyncCommentsOutput{}, err
	}
	if campaign.Type != domain.CampaignTypePost {
		return SyncCommentsOutput{}, domain.ErrCampaignNotPost
	}
	if campaign.Status != domain.CampaignStatusActive {
		return SyncCommentsOutput{}, domain.ErrCampaignNotActive
	}

	comments, err := uc.instagram.FetchComments(ctx, campaign.MediaID)
	if err != nil {
		return SyncCommentsOutput{}, err
	}

	var added int
	for _, comment := range comments {
		exists, err := uc.participants.ExistsByCampaignAndInstagramUserID(ctx, campaign.ID, comment.InstagramUserID)
		if err != nil {
			return SyncCommentsOutput{}, err
		}
		if exists {
			continue // FA-2.1.1 (ya registrado) — dedup, no es un error
		}

		if countMentions(comment.Text) < campaign.MinMentions {
			continue // FA-2.1.1 (no cumple la regla de min_mentions)
		}

		participant := &domain.Participant{
			CampaignID:      campaign.ID,
			InstagramUserID: comment.InstagramUserID,
			Username:        comment.Username,
			SourceType:      domain.ParticipantSourceComment,
		}
		if err := uc.participants.Create(ctx, participant); err != nil {
			if errors.Is(err, domain.ErrParticipantAlreadyExists) {
				// Otra corrida de sync-comments (o el webhook de historia,
				// si alguna vez comparten usuario) ganó la carrera contra
				// el pre-chequeo de arriba — mismo desenlace que FA-2.1.1,
				// no aborta el resto del lote.
				continue
			}
			return SyncCommentsOutput{}, err
		}
		added++
	}

	return SyncCommentsOutput{NewParticipants: added}, nil
}
