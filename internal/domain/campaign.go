package domain

import "time"

type CampaignType string

const (
	CampaignTypePost  CampaignType = "post"
	CampaignTypeStory CampaignType = "historia"
)

type CampaignStatus string

const (
	CampaignStatusDraft  CampaignStatus = "draft"
	CampaignStatusActive CampaignStatus = "activa"
	CampaignStatusClosed CampaignStatus = "cerrada"
	CampaignStatusDrawn  CampaignStatus = "sorteada"
)

// Campaign es una campaña de sorteo asociada a un post o historia de
// Instagram. Ciclo de vida: draft → activa → cerrada → sorteada.
type Campaign struct {
	ID          string
	Type        CampaignType
	MediaID     string
	Name        string
	MustFollow  bool
	MinMentions int
	Status      CampaignStatus
	CreatedAt   time.Time
}

// NewCampaign implementa la parte de validación de UC-1.1: crea una
// campaña en estado draft.
func NewCampaign(campaignType CampaignType, mediaID, name string, mustFollow bool, minMentions int) (*Campaign, error) {
	if campaignType != CampaignTypePost && campaignType != CampaignTypeStory {
		return nil, ErrInvalidCampaignType
	}
	if mediaID == "" {
		return nil, ErrMediaIDRequired
	}

	return &Campaign{
		Type:        campaignType,
		MediaID:     mediaID,
		Name:        name,
		MustFollow:  mustFollow,
		MinMentions: minMentions,
		Status:      CampaignStatusDraft,
	}, nil
}

// Activate implementa UC-1.2: solo se puede activar desde draft.
func (c *Campaign) Activate() error {
	if c.Status != CampaignStatusDraft {
		return ErrCampaignNotDraft
	}
	c.Status = CampaignStatusActive
	return nil
}

// Close implementa UC-1.3: solo se puede cerrar desde activa.
func (c *Campaign) Close() error {
	if c.Status != CampaignStatusActive {
		return ErrCampaignNotActive
	}
	c.Status = CampaignStatusClosed
	return nil
}
