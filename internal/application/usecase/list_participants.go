package usecase

import (
	"context"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

// ListParticipants implementa GET /campaigns/:id/participants
// (ver ARCHITECTURE.md §5; no tiene un UC dedicado en USE_CASES.md).
type ListParticipants struct {
	campaigns    domain.CampaignRepository
	participants domain.ParticipantRepository
}

func NewListParticipants(campaigns domain.CampaignRepository, participants domain.ParticipantRepository) *ListParticipants {
	return &ListParticipants{campaigns: campaigns, participants: participants}
}

func (uc *ListParticipants) Execute(ctx context.Context, campaignID string) ([]domain.Participant, error) {
	if _, err := uc.campaigns.FindByID(ctx, campaignID); err != nil {
		return nil, err
	}
	return uc.participants.ListByCampaign(ctx, campaignID)
}
