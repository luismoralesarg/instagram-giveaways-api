package usecase

import (
	"context"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type RunDrawInput struct {
	CampaignID   string
	WinnersCount int
}

type RunDrawOutput struct {
	Draw domain.Draw
}

// RunDraw implementa UC-3.1.
//
// Precondición real (decisión del usuario): la campaña debe estar
// 'cerrada' o 'sorteada' — nunca 'draft'/'activa'. Permitir 'sorteada'
// habilita el re-sorteo de FA-3.1.3, que de otro modo sería imposible
// (una campaña sorteada no puede volver a 'cerrada').
type RunDraw struct {
	campaigns    domain.CampaignRepository
	participants domain.ParticipantRepository
	draws        domain.DrawRepository
	randomGen    domain.RandomGenerator
}

func NewRunDraw(campaigns domain.CampaignRepository, participants domain.ParticipantRepository, draws domain.DrawRepository, randomGen domain.RandomGenerator) *RunDraw {
	return &RunDraw{campaigns: campaigns, participants: participants, draws: draws, randomGen: randomGen}
}

func (uc *RunDraw) Execute(ctx context.Context, in RunDrawInput) (RunDrawOutput, error) {
	campaign, err := uc.campaigns.FindByID(ctx, in.CampaignID)
	if err != nil {
		return RunDrawOutput{}, err
	}

	if campaign.Status != domain.CampaignStatusClosed && campaign.Status != domain.CampaignStatusDrawn {
		return RunDrawOutput{}, domain.ErrCampaignNotDrawable
	}

	eligible, err := uc.participants.ListEligibleByCampaign(ctx, campaign.ID)
	if err != nil {
		return RunDrawOutput{}, err
	}

	seed := uc.randomGen.Seed()
	winners, err := domain.SelectWinners(eligible, in.WinnersCount, seed)
	if err != nil {
		return RunDrawOutput{}, err
	}

	draw := &domain.Draw{
		CampaignID:   campaign.ID,
		WinnersCount: in.WinnersCount,
		RandomSeed:   seed,
		Winners:      winners,
	}
	if err := uc.draws.Create(ctx, draw); err != nil {
		return RunDrawOutput{}, err
	}

	if campaign.Status != domain.CampaignStatusDrawn {
		if err := uc.campaigns.UpdateStatus(ctx, campaign.ID, domain.CampaignStatusDrawn); err != nil {
			return RunDrawOutput{}, err
		}
	}

	return RunDrawOutput{Draw: *draw}, nil
}
