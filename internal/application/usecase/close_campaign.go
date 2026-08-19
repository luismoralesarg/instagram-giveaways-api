package usecase

import (
	"context"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

// CloseCampaign implementa UC-1.3.
type CloseCampaign struct {
	campaigns domain.CampaignRepository
}

func NewCloseCampaign(campaigns domain.CampaignRepository) *CloseCampaign {
	return &CloseCampaign{campaigns: campaigns}
}

func (uc *CloseCampaign) Execute(ctx context.Context, campaignID string) error {
	campaign, err := uc.campaigns.FindByID(ctx, campaignID)
	if err != nil {
		return err
	}

	if err := campaign.Close(); err != nil {
		return err
	}

	return uc.campaigns.UpdateStatus(ctx, campaign.ID, campaign.Status)
}
