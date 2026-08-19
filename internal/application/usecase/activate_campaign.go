package usecase

import (
	"context"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

// ActivateCampaign implementa UC-1.2.
type ActivateCampaign struct {
	campaigns domain.CampaignRepository
}

func NewActivateCampaign(campaigns domain.CampaignRepository) *ActivateCampaign {
	return &ActivateCampaign{campaigns: campaigns}
}

func (uc *ActivateCampaign) Execute(ctx context.Context, campaignID string) error {
	campaign, err := uc.campaigns.FindByID(ctx, campaignID)
	if err != nil {
		return err
	}

	if err := campaign.Activate(); err != nil {
		return err
	}

	return uc.campaigns.UpdateStatus(ctx, campaign.ID, campaign.Status)
}
