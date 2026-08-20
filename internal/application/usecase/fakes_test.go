package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

// Fakes en memoria de los ports de domain, usados solo en tests de esta
// capa. No hablan con Postgres ni con la Graph API real.

type fakeCampaignRepo struct {
	mu        sync.Mutex
	campaigns map[string]*domain.Campaign
	nextID    int
}

func newFakeCampaignRepo() *fakeCampaignRepo {
	return &fakeCampaignRepo{campaigns: map[string]*domain.Campaign{}}
}

func (r *fakeCampaignRepo) put(c *domain.Campaign) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c.ID == "" {
		r.nextID++
		c.ID = fmt.Sprintf("campaign-%d", r.nextID)
	}
	cp := *c
	r.campaigns[c.ID] = &cp
}

func (r *fakeCampaignRepo) Create(ctx context.Context, c *domain.Campaign) error {
	r.mu.Lock()
	r.nextID++
	c.ID = fmt.Sprintf("campaign-%d", r.nextID)
	c.CreatedAt = time.Now()
	r.mu.Unlock()
	r.put(c)
	return nil
}

func (r *fakeCampaignRepo) FindByID(ctx context.Context, id string) (*domain.Campaign, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.campaigns[id]
	if !ok {
		return nil, domain.ErrCampaignNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *fakeCampaignRepo) ExistsOpenByMediaID(ctx context.Context, mediaID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.campaigns {
		if c.MediaID == mediaID && (c.Status == domain.CampaignStatusDraft || c.Status == domain.CampaignStatusActive) {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeCampaignRepo) ExistsActiveStoryCampaign(ctx context.Context) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.campaigns {
		if c.Type == domain.CampaignTypeStory && c.Status == domain.CampaignStatusActive {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeCampaignRepo) FindActiveStoryCampaign(ctx context.Context) (*domain.Campaign, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.campaigns {
		if c.Type == domain.CampaignTypeStory && c.Status == domain.CampaignStatusActive {
			cp := *c
			return &cp, nil
		}
	}
	return nil, domain.ErrCampaignNotFound
}

func (r *fakeCampaignRepo) UpdateStatus(ctx context.Context, id string, status domain.CampaignStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.campaigns[id]
	if !ok {
		return domain.ErrCampaignNotFound
	}
	c.Status = status
	return nil
}

type fakeParticipantRepo struct {
	mu           sync.Mutex
	participants map[string]*domain.Participant
	nextID       int
}

func newFakeParticipantRepo() *fakeParticipantRepo {
	return &fakeParticipantRepo{participants: map[string]*domain.Participant{}}
}

func (r *fakeParticipantRepo) put(p *domain.Participant) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p.ID == "" {
		r.nextID++
		p.ID = fmt.Sprintf("participant-%d", r.nextID)
	}
	cp := *p
	r.participants[p.ID] = &cp
}

func (r *fakeParticipantRepo) Create(ctx context.Context, p *domain.Participant) error {
	r.mu.Lock()
	r.nextID++
	p.ID = fmt.Sprintf("participant-%d", r.nextID)
	p.CreatedAt = time.Now()
	r.mu.Unlock()
	r.put(p)
	return nil
}

func (r *fakeParticipantRepo) ExistsByCampaignAndInstagramUserID(ctx context.Context, campaignID, instagramUserID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.participants {
		if p.CampaignID == campaignID && p.InstagramUserID == instagramUserID {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeParticipantRepo) FindByID(ctx context.Context, campaignID, participantID string) (*domain.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.participants[participantID]
	if !ok || p.CampaignID != campaignID {
		return nil, domain.ErrParticipantNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *fakeParticipantRepo) Exclude(ctx context.Context, participantID, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.participants[participantID]
	if !ok {
		return domain.ErrParticipantNotFound
	}
	p.IsExcluded = true
	p.ExcludedReason = reason
	return nil
}

func (r *fakeParticipantRepo) ListByCampaign(ctx context.Context, campaignID string) ([]domain.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Participant
	for _, p := range r.participants {
		if p.CampaignID == campaignID {
			out = append(out, *p)
		}
	}
	return out, nil
}

type fakeInstagramClient struct {
	comments []domain.InstagramComment
	err      error
}

func (f *fakeInstagramClient) FetchComments(ctx context.Context, mediaID string) ([]domain.InstagramComment, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.comments, nil
}
