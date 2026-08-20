package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

func TestInstagramTokenRepository_SaveAndGet(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewInstagramTokenRepository(pool)

	expiresAt := time.Now().Add(60 * 24 * time.Hour).Truncate(time.Second)
	token := &domain.InstagramToken{AccessToken: "token-" + uniqueID(t), ExpiresAt: expiresAt}
	if err := repo.Save(ctx, token); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if token.UpdatedAt.IsZero() {
		t.Error("Save() no asignó UpdatedAt")
	}

	got, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.AccessToken != token.AccessToken || !got.ExpiresAt.Equal(expiresAt) {
		t.Errorf("Get() = %+v, quiero access_token=%q expires_at=%v", got, token.AccessToken, expiresAt)
	}
}

func TestInstagramTokenRepository_Save_UpsertsSingleRow(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewInstagramTokenRepository(pool)

	first := &domain.InstagramToken{AccessToken: "primero-" + uniqueID(t), ExpiresAt: time.Now().Add(time.Hour)}
	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("Save(first) error = %v", err)
	}

	second := &domain.InstagramToken{AccessToken: "segundo-" + uniqueID(t), ExpiresAt: time.Now().Add(2 * time.Hour)}
	if err := repo.Save(ctx, second); err != nil {
		t.Fatalf("Save(second) error = %v", err)
	}

	got, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.AccessToken != second.AccessToken {
		t.Errorf("AccessToken = %q, quiero %q (el segundo Save debe pisar al primero, no agregar una fila)", got.AccessToken, second.AccessToken)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_token`).Scan(&count); err != nil {
		t.Fatalf("contar filas: %v", err)
	}
	if count != 1 {
		t.Fatalf("count(*) = %d, quiero 1 (tabla de una sola fila)", count)
	}
}

func TestInstagramTokenRepository_Get_NotFound(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `DELETE FROM instagram_token WHERE id = 1`); err != nil {
		t.Fatalf("limpiar tabla: %v", err)
	}

	repo := NewInstagramTokenRepository(pool)
	_, err := repo.Get(ctx)
	if !errors.Is(err, domain.ErrInstagramTokenNotFound) {
		t.Fatalf("err = %v, quiero %v", err, domain.ErrInstagramTokenNotFound)
	}
}
