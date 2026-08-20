package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

func TestSyncComments_FiltersAndDedupes(t *testing.T) {
	campaigns := newFakeCampaignRepo()
	participants := newFakeParticipantRepo()

	campaign := &domain.Campaign{
		ID:          "c1",
		Type:        domain.CampaignTypePost,
		MediaID:     "media-123",
		Status:      domain.CampaignStatusActive,
		MinMentions: 1,
	}
	campaigns.put(campaign)
	participants.put(&domain.Participant{ID: "p1", CampaignID: "c1", InstagramUserID: "already"})

	instagram := &fakeInstagramClient{comments: []domain.InstagramComment{
		{InstagramUserID: "already", Username: "u1", Text: "@amigo"},        // FA-2.1.1: dedup
		{InstagramUserID: "sin_mencion", Username: "u2", Text: "sin nada"},  // FA-2.1.1: no cumple min_mentions
		{InstagramUserID: "con_mencion", Username: "u3", Text: "yo @amigo"}, // válido
	}}

	uc := NewSyncComments(campaigns, participants, instagram)
	out, err := uc.Execute(context.Background(), SyncCommentsInput{CampaignID: "c1"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.NewParticipants != 1 {
		t.Fatalf("NewParticipants = %d, quiero 1", out.NewParticipants)
	}

	all, _ := participants.ListByCampaign(context.Background(), "c1")
	var found bool
	for _, p := range all {
		if p.InstagramUserID == "con_mencion" {
			found = true
			if p.SourceType != domain.ParticipantSourceComment {
				t.Errorf("SourceType = %q, quiero %q", p.SourceType, domain.ParticipantSourceComment)
			}
		}
		if p.InstagramUserID == "sin_mencion" {
			t.Errorf("sin_mencion no debería haberse persistido (no cumple min_mentions)")
		}
	}
	if !found {
		t.Errorf("con_mencion debería haberse persistido como participante nuevo")
	}
}

func TestSyncComments_RejectsNonPostCampaign(t *testing.T) {
	campaigns := newFakeCampaignRepo()
	campaigns.put(&domain.Campaign{ID: "c1", Type: domain.CampaignTypeStory, Status: domain.CampaignStatusActive})

	uc := NewSyncComments(campaigns, newFakeParticipantRepo(), &fakeInstagramClient{})
	_, err := uc.Execute(context.Background(), SyncCommentsInput{CampaignID: "c1"})
	if !errors.Is(err, domain.ErrCampaignNotPost) {
		t.Fatalf("err = %v, quiero %v", err, domain.ErrCampaignNotPost)
	}
}

func TestSyncComments_RejectsNonActiveCampaign(t *testing.T) {
	campaigns := newFakeCampaignRepo()
	campaigns.put(&domain.Campaign{ID: "c1", Type: domain.CampaignTypePost, Status: domain.CampaignStatusDraft})

	uc := NewSyncComments(campaigns, newFakeParticipantRepo(), &fakeInstagramClient{})
	_, err := uc.Execute(context.Background(), SyncCommentsInput{CampaignID: "c1"})
	if !errors.Is(err, domain.ErrCampaignNotActive) {
		t.Fatalf("err = %v, quiero %v", err, domain.ErrCampaignNotActive)
	}
}

// Carrera: el pre-chequeo de ExistsByCampaignAndInstagramUserID no ve nada
// (ej. otra corrida de sync-comments en simultáneo), pero Create() choca
// contra el índice único — no debe abortar el resto del lote ni contar
// ese comentario como nuevo.
func TestSyncComments_CreateRace_SkippedNotAborted(t *testing.T) {
	campaigns := newFakeCampaignRepo()
	campaigns.put(&domain.Campaign{
		ID:      "c1",
		Type:    domain.CampaignTypePost,
		MediaID: "media-123",
		Status:  domain.CampaignStatusActive,
	})

	instagram := &fakeInstagramClient{comments: []domain.InstagramComment{
		{InstagramUserID: "u1", Username: "user1", Text: "hola"},
	}}

	uc := NewSyncComments(campaigns, newFakeParticipantRepoAlwaysConflicts(), instagram)
	out, err := uc.Execute(context.Background(), SyncCommentsInput{CampaignID: "c1"})
	if err != nil {
		t.Fatalf("Execute() error = %v, quiero nil (ErrParticipantAlreadyExists no debería abortar el lote)", err)
	}
	if out.NewParticipants != 0 {
		t.Errorf("NewParticipants = %d, quiero 0 (el que chocó no cuenta como nuevo)", out.NewParticipants)
	}
}

func TestCountMentions(t *testing.T) {
	cases := map[string]int{
		"":                        0,
		"sin mencion":             0,
		"@uno":                    1,
		"@uno @dos":               2,
		"hola @uno.dos participo": 1,
	}
	for text, want := range cases {
		if got := countMentions(text); got != want {
			t.Errorf("countMentions(%q) = %d, quiero %d", text, got, want)
		}
	}
}
