package usecase

import (
	"context"
	"testing"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

func TestHandleStoryMention_CreatesParticipant(t *testing.T) {
	campaigns := newFakeCampaignRepo()
	participants := newFakeParticipantRepo()
	campaigns.put(&domain.Campaign{ID: "c1", Type: domain.CampaignTypeStory, Status: domain.CampaignStatusActive})

	uc := NewHandleStoryMention(campaigns, participants)
	if err := uc.Execute(context.Background(), HandleStoryMentionInput{InstagramUserID: "u1"}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	all, _ := participants.ListByCampaign(context.Background(), "c1")
	if len(all) != 1 || all[0].InstagramUserID != "u1" || all[0].SourceType != domain.ParticipantSourceStoryMention {
		t.Fatalf("participantes = %+v, quiero un story_mention de u1", all)
	}
}

// FA-2.2.2: sin campaña de historia activa, el evento se descarta sin error.
func TestHandleStoryMention_NoActiveCampaign_NoError(t *testing.T) {
	uc := NewHandleStoryMention(newFakeCampaignRepo(), newFakeParticipantRepo())
	if err := uc.Execute(context.Background(), HandleStoryMentionInput{InstagramUserID: "u1"}); err != nil {
		t.Fatalf("Execute() error = %v, quiero nil", err)
	}
}

// FA-2.2.3: usuario ya registrado, se ignora sin duplicar.
func TestHandleStoryMention_DuplicateIgnored(t *testing.T) {
	campaigns := newFakeCampaignRepo()
	participants := newFakeParticipantRepo()
	campaigns.put(&domain.Campaign{ID: "c1", Type: domain.CampaignTypeStory, Status: domain.CampaignStatusActive})
	participants.put(&domain.Participant{ID: "p1", CampaignID: "c1", InstagramUserID: "u1"})

	uc := NewHandleStoryMention(campaigns, participants)
	if err := uc.Execute(context.Background(), HandleStoryMentionInput{InstagramUserID: "u1"}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	all, _ := participants.ListByCampaign(context.Background(), "c1")
	if len(all) != 1 {
		t.Fatalf("len(participantes) = %d, quiero 1 (no debería duplicar)", len(all))
	}
}

// Carrera: el pre-chequeo de ExistsByCampaignAndInstagramUserID no ve nada
// (ej. Instagram reentregó el mismo webhook en simultáneo), pero Create()
// choca contra el índice único — debe tratarse igual que FA-2.2.3, no
// como una falla.
func TestHandleStoryMention_CreateRace_TreatedAsAlreadyRegistered(t *testing.T) {
	campaigns := newFakeCampaignRepo()
	campaigns.put(&domain.Campaign{ID: "c1", Type: domain.CampaignTypeStory, Status: domain.CampaignStatusActive})

	uc := NewHandleStoryMention(campaigns, newFakeParticipantRepoAlwaysConflicts())
	if err := uc.Execute(context.Background(), HandleStoryMentionInput{InstagramUserID: "u1"}); err != nil {
		t.Fatalf("Execute() error = %v, quiero nil (ErrParticipantAlreadyExists no debería propagarse)", err)
	}
}
