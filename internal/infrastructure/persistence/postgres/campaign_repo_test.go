package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

func TestCampaignRepository_CreateAndFindByID(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewCampaignRepository(pool)

	c := &domain.Campaign{
		Type:        domain.CampaignTypePost,
		MediaID:     uniqueID(t),
		Name:        "Sorteo de prueba",
		MustFollow:  true,
		MinMentions: 2,
		Status:      domain.CampaignStatusDraft,
	}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if c.ID == "" {
		t.Fatal("Create() no asignó ID")
	}

	found, err := repo.FindByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.MediaID != c.MediaID || found.Name != c.Name || found.MustFollow != true ||
		found.MinMentions != 2 || found.Status != domain.CampaignStatusDraft || found.Type != domain.CampaignTypePost {
		t.Errorf("FindByID() = %+v, no coincide con lo creado (%+v)", found, c)
	}
}

func TestCampaignRepository_FindByID_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewCampaignRepository(pool)

	_, err := repo.FindByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, domain.ErrCampaignNotFound) {
		t.Fatalf("err = %v, quiero %v", err, domain.ErrCampaignNotFound)
	}
}

func TestCampaignRepository_FindByID_InvalidUUID(t *testing.T) {
	pool := testPool(t)
	repo := NewCampaignRepository(pool)

	// Un id mal formado tiene que mapear al mismo "no encontrado" (404),
	// no filtrar un error 500 de Postgres sobre el tipo de columna.
	_, err := repo.FindByID(context.Background(), "no-es-un-uuid")
	if !errors.Is(err, domain.ErrCampaignNotFound) {
		t.Fatalf("err = %v, quiero %v", err, domain.ErrCampaignNotFound)
	}
}

func TestCampaignRepository_ExistsOpenByMediaID(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewCampaignRepository(pool)
	mediaID := uniqueID(t)

	exists, err := repo.ExistsOpenByMediaID(ctx, mediaID)
	if err != nil {
		t.Fatalf("ExistsOpenByMediaID() error = %v", err)
	}
	if exists {
		t.Fatal("no debería existir todavía")
	}

	c := &domain.Campaign{Type: domain.CampaignTypePost, MediaID: mediaID, Status: domain.CampaignStatusDraft}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	exists, err = repo.ExistsOpenByMediaID(ctx, mediaID)
	if err != nil {
		t.Fatalf("ExistsOpenByMediaID() error = %v", err)
	}
	if !exists {
		t.Fatal("debería existir (draft cuenta como open)")
	}

	if err := repo.UpdateStatus(ctx, c.ID, domain.CampaignStatusClosed); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	exists, err = repo.ExistsOpenByMediaID(ctx, mediaID)
	if err != nil {
		t.Fatalf("ExistsOpenByMediaID() error = %v", err)
	}
	if exists {
		t.Fatal("cerrada no debería contar como open")
	}
}

func TestCampaignRepository_Create_DuplicateMediaID(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewCampaignRepository(pool)
	mediaID := uniqueID(t)

	first := &domain.Campaign{Type: domain.CampaignTypePost, MediaID: mediaID, Status: domain.CampaignStatusDraft}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	second := &domain.Campaign{Type: domain.CampaignTypePost, MediaID: mediaID, Status: domain.CampaignStatusDraft}
	err := repo.Create(ctx, second)
	if !errors.Is(err, domain.ErrDuplicateMediaID) {
		t.Fatalf("err = %v, quiero %v (índice único parcial de campaigns)", err, domain.ErrDuplicateMediaID)
	}
}

func TestCampaignRepository_UpdateStatus(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewCampaignRepository(pool)

	c := &domain.Campaign{Type: domain.CampaignTypePost, MediaID: uniqueID(t), Status: domain.CampaignStatusDraft}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.UpdateStatus(ctx, c.ID, domain.CampaignStatusActive); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	found, err := repo.FindByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Status != domain.CampaignStatusActive {
		t.Errorf("Status = %q, quiero %q", found.Status, domain.CampaignStatusActive)
	}
}

func TestCampaignRepository_SingleActiveStoryConstraint(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewCampaignRepository(pool)

	first := &domain.Campaign{Type: domain.CampaignTypeStory, MediaID: uniqueID(t), Status: domain.CampaignStatusDraft}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := repo.UpdateStatus(ctx, first.ID, domain.CampaignStatusActive); err != nil {
		t.Fatalf("UpdateStatus(first) error = %v", err)
	}

	exists, err := repo.ExistsActiveStoryCampaign(ctx)
	if err != nil {
		t.Fatalf("ExistsActiveStoryCampaign() error = %v", err)
	}
	if !exists {
		t.Fatal("debería haber una campaña de historia activa")
	}

	active, err := repo.FindActiveStoryCampaign(ctx)
	if err != nil {
		t.Fatalf("FindActiveStoryCampaign() error = %v", err)
	}
	if active.ID != first.ID {
		t.Errorf("FindActiveStoryCampaign() = %q, quiero %q", active.ID, first.ID)
	}

	second := &domain.Campaign{Type: domain.CampaignTypeStory, MediaID: uniqueID(t), Status: domain.CampaignStatusDraft}
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	// El índice único parcial campaigns_single_active_story_idx es la
	// última palabra — este UpdateStatus tiene que fallar aunque nada más
	// lo haya chequeado antes (ver ActivateCampaign para el pre-chequeo).
	err = repo.UpdateStatus(ctx, second.ID, domain.CampaignStatusActive)
	if !errors.Is(err, domain.ErrActiveStoryCampaignExists) {
		t.Fatalf("err = %v, quiero %v", err, domain.ErrActiveStoryCampaignExists)
	}
}
