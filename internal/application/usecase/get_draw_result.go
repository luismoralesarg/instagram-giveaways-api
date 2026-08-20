package usecase

import (
	"context"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type GetDrawResultInput struct {
	CampaignID string
	DrawID     string
}

// GetDrawResult implementa UC-3.2.
type GetDrawResult struct {
	draws domain.DrawRepository
}

func NewGetDrawResult(draws domain.DrawRepository) *GetDrawResult {
	return &GetDrawResult{draws: draws}
}

func (uc *GetDrawResult) Execute(ctx context.Context, in GetDrawResultInput) (domain.Draw, error) {
	draw, err := uc.draws.FindByID(ctx, in.CampaignID, in.DrawID)
	if err != nil {
		return domain.Draw{}, err
	}
	return *draw, nil
}
