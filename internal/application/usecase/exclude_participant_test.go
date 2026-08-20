package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

func TestExcludeParticipant_MarksAndPersists(t *testing.T) {
	participants := newFakeParticipantRepo()
	participants.put(&domain.Participant{ID: "p1", CampaignID: "c1", InstagramUserID: "u1"})

	uc := NewExcludeParticipant(participants)
	err := uc.Execute(context.Background(), ExcludeParticipantInput{
		CampaignID:    "c1",
		ParticipantID: "p1",
		Reason:        "cuenta falsa",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got, _ := participants.FindByID(context.Background(), "c1", "p1")
	if !got.IsExcluded || got.ExcludedReason != "cuenta falsa" {
		t.Fatalf("participant = %+v, quiero IsExcluded=true con el motivo guardado", got)
	}
}

func TestExcludeParticipant_NotFound(t *testing.T) {
	uc := NewExcludeParticipant(newFakeParticipantRepo())
	err := uc.Execute(context.Background(), ExcludeParticipantInput{CampaignID: "c1", ParticipantID: "no-existe"})
	if !errors.Is(err, domain.ErrParticipantNotFound) {
		t.Fatalf("err = %v, quiero %v", err, domain.ErrParticipantNotFound)
	}
}

// FA-2.3.2: excluir sobre una campaña ya sorteada también se permite (no
// hay chequeo de estado en Exclude).
func TestExcludeParticipant_AllowedRegardlessOfCampaignState(t *testing.T) {
	participants := newFakeParticipantRepo()
	participants.put(&domain.Participant{ID: "p1", CampaignID: "c1", InstagramUserID: "u1"})

	uc := NewExcludeParticipant(participants)
	if err := uc.Execute(context.Background(), ExcludeParticipantInput{CampaignID: "c1", ParticipantID: "p1", Reason: "empleado"}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}
