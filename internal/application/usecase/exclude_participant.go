package usecase

import (
	"context"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type ExcludeParticipantInput struct {
	CampaignID    string
	ParticipantID string
	Reason        string
}

// ExcludeParticipant implementa UC-2.3.
type ExcludeParticipant struct {
	participants domain.ParticipantRepository
}

func NewExcludeParticipant(participants domain.ParticipantRepository) *ExcludeParticipant {
	return &ExcludeParticipant{participants: participants}
}

func (uc *ExcludeParticipant) Execute(ctx context.Context, in ExcludeParticipantInput) error {
	participant, err := uc.participants.FindByID(ctx, in.CampaignID, in.ParticipantID)
	if err != nil {
		return err
	}

	participant.Exclude(in.Reason)

	return uc.participants.Exclude(ctx, participant.ID, participant.ExcludedReason)
}
