package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

func createTestCampaign(t *testing.T, ctx context.Context, pool *pgxpool.Pool) *domain.Campaign {
	t.Helper()

	c := &domain.Campaign{Type: domain.CampaignTypePost, MediaID: uniqueID(t), Status: domain.CampaignStatusActive}
	if err := NewCampaignRepository(pool).Create(ctx, c); err != nil {
		t.Fatalf("crear campaña de prueba: %v", err)
	}
	return c
}

func TestParticipantRepository_CreateAndFindByID(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	campaign := createTestCampaign(t, ctx, pool)
	repo := NewParticipantRepository(pool)

	p := &domain.Participant{
		CampaignID:      campaign.ID,
		InstagramUserID: uniqueID(t),
		Username:        "usuario_prueba",
		SourceType:      domain.ParticipantSourceComment,
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if p.ID == "" {
		t.Fatal("Create() no asignó ID")
	}

	found, err := repo.FindByID(ctx, campaign.ID, p.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.InstagramUserID != p.InstagramUserID || found.Username != "usuario_prueba" ||
		found.SourceType != domain.ParticipantSourceComment || found.IsExcluded {
		t.Errorf("FindByID() = %+v", found)
	}
}

func TestParticipantRepository_FindByID_WrongCampaign(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	campaign := createTestCampaign(t, ctx, pool)
	otherCampaign := createTestCampaign(t, ctx, pool)
	repo := NewParticipantRepository(pool)

	p := &domain.Participant{CampaignID: campaign.ID, InstagramUserID: uniqueID(t), SourceType: domain.ParticipantSourceComment}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Un participante real, pero de otra campaña — no debe poder leerse
	// "cruzado" (UC-2.3 valida que el participante pertenezca a la campaña
	// indicada en la URL).
	_, err := repo.FindByID(ctx, otherCampaign.ID, p.ID)
	if !errors.Is(err, domain.ErrParticipantNotFound) {
		t.Fatalf("err = %v, quiero %v", err, domain.ErrParticipantNotFound)
	}
}

// Regresión: antes de este fix, Create() devolvía el *pgconn.PgError crudo
// de la violación de índice único en vez de un error de dominio — un
// caller que no esperara ese tipo concreto (ej. el 500 en vez de tratarlo
// como "ya existe") es exactamente el bug que encontró la revisión.
func TestParticipantRepository_Create_DuplicateInstagramUserID(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	campaign := createTestCampaign(t, ctx, pool)
	repo := NewParticipantRepository(pool)
	instagramUserID := uniqueID(t)

	first := &domain.Participant{CampaignID: campaign.ID, InstagramUserID: instagramUserID, SourceType: domain.ParticipantSourceComment}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	second := &domain.Participant{CampaignID: campaign.ID, InstagramUserID: instagramUserID, SourceType: domain.ParticipantSourceStoryMention}
	err := repo.Create(ctx, second)
	if !errors.Is(err, domain.ErrParticipantAlreadyExists) {
		t.Fatalf("err = %v, quiero %v (índice único de participants)", err, domain.ErrParticipantAlreadyExists)
	}
}

func TestParticipantRepository_ExistsByCampaignAndInstagramUserID(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	campaign := createTestCampaign(t, ctx, pool)
	repo := NewParticipantRepository(pool)
	instagramUserID := uniqueID(t)

	exists, err := repo.ExistsByCampaignAndInstagramUserID(ctx, campaign.ID, instagramUserID)
	if err != nil {
		t.Fatalf("ExistsByCampaignAndInstagramUserID() error = %v", err)
	}
	if exists {
		t.Fatal("no debería existir todavía")
	}

	p := &domain.Participant{CampaignID: campaign.ID, InstagramUserID: instagramUserID, SourceType: domain.ParticipantSourceComment}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	exists, err = repo.ExistsByCampaignAndInstagramUserID(ctx, campaign.ID, instagramUserID)
	if err != nil {
		t.Fatalf("ExistsByCampaignAndInstagramUserID() error = %v", err)
	}
	if !exists {
		t.Fatal("debería existir")
	}
}

func TestParticipantRepository_Exclude(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	campaign := createTestCampaign(t, ctx, pool)
	repo := NewParticipantRepository(pool)

	p := &domain.Participant{CampaignID: campaign.ID, InstagramUserID: uniqueID(t), SourceType: domain.ParticipantSourceComment}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Exclude(ctx, p.ID, "cuenta falsa"); err != nil {
		t.Fatalf("Exclude() error = %v", err)
	}

	found, err := repo.FindByID(ctx, campaign.ID, p.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if !found.IsExcluded || found.ExcludedReason != "cuenta falsa" {
		t.Errorf("found = %+v, quiero IsExcluded=true con el motivo guardado", found)
	}
}

func TestParticipantRepository_ListByCampaignAndEligible(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	campaign := createTestCampaign(t, ctx, pool)
	repo := NewParticipantRepository(pool)

	included := &domain.Participant{CampaignID: campaign.ID, InstagramUserID: uniqueID(t) + "-a", SourceType: domain.ParticipantSourceComment}
	excluded := &domain.Participant{CampaignID: campaign.ID, InstagramUserID: uniqueID(t) + "-b", SourceType: domain.ParticipantSourceComment}
	if err := repo.Create(ctx, included); err != nil {
		t.Fatalf("Create(included) error = %v", err)
	}
	if err := repo.Create(ctx, excluded); err != nil {
		t.Fatalf("Create(excluded) error = %v", err)
	}
	if err := repo.Exclude(ctx, excluded.ID, "empleado"); err != nil {
		t.Fatalf("Exclude() error = %v", err)
	}

	all, err := repo.ListByCampaign(ctx, campaign.ID)
	if err != nil {
		t.Fatalf("ListByCampaign() error = %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("ListByCampaign() = %d participantes, quiero 2 (incluye excluidos)", len(all))
	}

	eligible, err := repo.ListEligibleByCampaign(ctx, campaign.ID)
	if err != nil {
		t.Fatalf("ListEligibleByCampaign() error = %v", err)
	}
	if len(eligible) != 1 || eligible[0].ID != included.ID {
		t.Fatalf("ListEligibleByCampaign() = %+v, quiero solo %q (el excluido no cuenta)", eligible, included.ID)
	}
}
