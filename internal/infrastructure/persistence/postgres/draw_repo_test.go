package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

func createTestParticipant(t *testing.T, ctx context.Context, pool *pgxpool.Pool, campaignID string) *domain.Participant {
	t.Helper()
	p := &domain.Participant{CampaignID: campaignID, InstagramUserID: uniqueID(t), Username: "ganador", SourceType: domain.ParticipantSourceComment}
	if err := NewParticipantRepository(pool).Create(ctx, p); err != nil {
		t.Fatalf("crear participante de prueba: %v", err)
	}
	return p
}

func TestDrawRepository_CreateAndFindByID(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	campaign := createTestCampaign(t, ctx, pool)
	p1 := createTestParticipant(t, ctx, pool, campaign.ID)
	p2 := createTestParticipant(t, ctx, pool, campaign.ID)

	repo := NewDrawRepository(pool)
	draw := &domain.Draw{
		CampaignID:   campaign.ID,
		WinnersCount: 2,
		RandomSeed:   123456789,
		Winners: []domain.Winner{
			{ParticipantID: p1.ID, InstagramUserID: p1.InstagramUserID, Username: p1.Username, Position: 1},
			{ParticipantID: p2.ID, InstagramUserID: p2.InstagramUserID, Username: p2.Username, Position: 2},
		},
	}

	if err := repo.Create(ctx, draw); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if draw.ID == "" {
		t.Fatal("Create() no asignó ID al draw")
	}
	if draw.CreatedAt.IsZero() {
		t.Error("Create() no asignó CreatedAt")
	}
	for i, w := range draw.Winners {
		if w.ID == "" || w.DrawID != draw.ID {
			t.Errorf("Winners[%d] = %+v, falta ID o DrawID no coincide", i, w)
		}
	}

	found, err := repo.FindByID(ctx, campaign.ID, draw.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.WinnersCount != 2 || found.RandomSeed != 123456789 {
		t.Errorf("found = %+v", found)
	}
	if len(found.Winners) != 2 {
		t.Fatalf("len(Winners) = %d, quiero 2", len(found.Winners))
	}
	// ORDER BY position: el primero tiene que ser p1 (position=1).
	if found.Winners[0].ParticipantID != p1.ID || found.Winners[0].Position != 1 {
		t.Errorf("Winners[0] = %+v, quiero participant_id=%q position=1", found.Winners[0], p1.ID)
	}
	if found.Winners[1].ParticipantID != p2.ID || found.Winners[1].Position != 2 {
		t.Errorf("Winners[1] = %+v, quiero participant_id=%q position=2", found.Winners[1], p2.ID)
	}
}

func TestDrawRepository_FindByID_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewDrawRepository(pool)

	_, err := repo.FindByID(context.Background(), "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, domain.ErrDrawNotFound) {
		t.Fatalf("err = %v, quiero %v", err, domain.ErrDrawNotFound)
	}
}

func TestDrawRepository_FindByID_WrongCampaign(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	campaign := createTestCampaign(t, ctx, pool)
	otherCampaign := createTestCampaign(t, ctx, pool)
	p1 := createTestParticipant(t, ctx, pool, campaign.ID)

	repo := NewDrawRepository(pool)
	draw := &domain.Draw{
		CampaignID:   campaign.ID,
		WinnersCount: 1,
		RandomSeed:   1,
		Winners:      []domain.Winner{{ParticipantID: p1.ID, InstagramUserID: p1.InstagramUserID, Position: 1}},
	}
	if err := repo.Create(ctx, draw); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err := repo.FindByID(ctx, otherCampaign.ID, draw.ID)
	if !errors.Is(err, domain.ErrDrawNotFound) {
		t.Fatalf("err = %v, quiero %v (el draw es de otra campaña)", err, domain.ErrDrawNotFound)
	}
}
