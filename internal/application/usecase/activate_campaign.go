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

	// Regla acordada para UC-2.2: a lo sumo una campaña de tipo historia
	// activa a la vez (el webhook de Instagram no permite desambiguar).
	if campaign.Type == domain.CampaignTypeStory {
		exists, err := uc.campaigns.ExistsActiveStoryCampaign(ctx)
		if err != nil {
			return err
		}
		if exists {
			return domain.ErrActiveStoryCampaignExists
		}
	}

	if err := campaign.Activate(); err != nil {
		return err
	}

	return uc.campaigns.UpdateStatus(ctx, campaign.ID, campaign.Status)
}
