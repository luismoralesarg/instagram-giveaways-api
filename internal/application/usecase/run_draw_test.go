package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

func TestRunDraw_ClosedCampaign_Success(t *testing.T) {
	campaigns := newFakeCampaignRepo()
	participants := newFakeParticipantRepo()
	draws := newFakeDrawRepo()

	campaigns.put(&domain.Campaign{ID: "c1", Status: domain.CampaignStatusClosed})
	participants.put(&domain.Participant{ID: "p1", CampaignID: "c1", InstagramUserID: "u1"})
	participants.put(&domain.Participant{ID: "p2", CampaignID: "c1", InstagramUserID: "u2"})

	uc := NewRunDraw(campaigns, participants, draws, fakeRandomGenerator{seed: 99})
	out, err := uc.Execute(context.Background(), RunDrawInput{CampaignID: "c1", WinnersCount: 1})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(out.Draw.Winners) != 1 {
		t.Fatalf("len(Winners) = %d, quiero 1", len(out.Draw.Winners))
	}
	if out.Draw.RandomSeed != 99 {
		t.Errorf("RandomSeed = %d, quiero 99", out.Draw.RandomSeed)
	}

	campaign, _ := campaigns.FindByID(context.Background(), "c1")
	if campaign.Status != domain.CampaignStatusDrawn {
		t.Errorf("campaign.Status = %q, quiero %q", campaign.Status, domain.CampaignStatusDrawn)
	}
}

func TestRunDraw_RejectsDraftOrActive(t *testing.T) {
	for _, status := range []domain.CampaignStatus{domain.CampaignStatusDraft, domain.CampaignStatusActive} {
		campaigns := newFakeCampaignRepo()
		campaigns.put(&domain.Campaign{ID: "c1", Status: status})

		uc := NewRunDraw(campaigns, newFakeParticipantRepo(), newFakeDrawRepo(), fakeRandomGenerator{seed: 1})
		_, err := uc.Execute(context.Background(), RunDrawInput{CampaignID: "c1", WinnersCount: 1})
		if !errors.Is(err, domain.ErrCampaignNotDrawable) {
			t.Errorf("status=%q: err = %v, quiero %v", status, err, domain.ErrCampaignNotDrawable)
		}
	}
}

// FA-3.1.3: re-sorteo sobre una campaña ya sorteada.
func TestRunDraw_AllowsRedrawOnDrawnCampaign(t *testing.T) {
	campaigns := newFakeCampaignRepo()
	participants := newFakeParticipantRepo()
	draws := newFakeDrawRepo()

	campaigns.put(&domain.Campaign{ID: "c1", Status: domain.CampaignStatusDrawn})
	participants.put(&domain.Participant{ID: "p1", CampaignID: "c1", InstagramUserID: "u1"})

	uc := NewRunDraw(campaigns, participants, draws, fakeRandomGenerator{seed: 5})
	if _, err := uc.Execute(context.Background(), RunDrawInput{CampaignID: "c1", WinnersCount: 1}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestRunDraw_ExcludedParticipantsIgnored(t *testing.T) {
	campaigns := newFakeCampaignRepo()
	participants := newFakeParticipantRepo()
	draws := newFakeDrawRepo()

	campaigns.put(&domain.Campaign{ID: "c1", Status: domain.CampaignStatusClosed})
	participants.put(&domain.Participant{ID: "p1", CampaignID: "c1", InstagramUserID: "u1", IsExcluded: true})
	participants.put(&domain.Participant{ID: "p2", CampaignID: "c1", InstagramUserID: "u2"})

	uc := NewRunDraw(campaigns, participants, draws, fakeRandomGenerator{seed: 3})
	out, err := uc.Execute(context.Background(), RunDrawInput{CampaignID: "c1", WinnersCount: 1})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.Draw.Winners[0].ParticipantID != "p2" {
		t.Fatalf("ganador = %q, quiero p2 (p1 está excluido)", out.Draw.Winners[0].ParticipantID)
	}
}

func TestRunDraw_NotEnoughParticipants(t *testing.T) {
	campaigns := newFakeCampaignRepo()
	participants := newFakeParticipantRepo()
	draws := newFakeDrawRepo()

	campaigns.put(&domain.Campaign{ID: "c1", Status: domain.CampaignStatusClosed})
	participants.put(&domain.Participant{ID: "p1", CampaignID: "c1", InstagramUserID: "u1"})

	uc := NewRunDraw(campaigns, participants, draws, fakeRandomGenerator{seed: 1})
	_, err := uc.Execute(context.Background(), RunDrawInput{CampaignID: "c1", WinnersCount: 3})
	if !errors.Is(err, domain.ErrNotEnoughParticipants) {
		t.Fatalf("err = %v, quiero %v", err, domain.ErrNotEnoughParticipants)
	}
}
