package usecase

import (
	"context"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type CreateCampaignInput struct {
	Type        domain.CampaignType
	MediaID     string
	Name        string
	MustFollow  bool
	MinMentions int
}

type CreateCampaignOutput struct {
	CampaignID string
}

// CreateCampaign implementa UC-1.1.
type CreateCampaign struct {
	campaigns domain.CampaignRepository
}

func NewCreateCampaign(campaigns domain.CampaignRepository) *CreateCampaign {
	return &CreateCampaign{campaigns: campaigns}
}

func (uc *CreateCampaign) Execute(ctx context.Context, in CreateCampaignInput) (CreateCampaignOutput, error) {
	exists, err := uc.campaigns.ExistsOpenByMediaID(ctx, in.MediaID)
	if err != nil {
		return CreateCampaignOutput{}, err
	}
	if exists {
		return CreateCampaignOutput{}, domain.ErrDuplicateMediaID
	}

	campaign, err := domain.NewCampaign(in.Type, in.MediaID, in.Name, in.MustFollow, in.MinMentions)
	if err != nil {
		return CreateCampaignOutput{}, err
	}

	if err := uc.campaigns.Create(ctx, campaign); err != nil {
		return CreateCampaignOutput{}, err
	}

	return CreateCampaignOutput{CampaignID: campaign.ID}, nil
}
