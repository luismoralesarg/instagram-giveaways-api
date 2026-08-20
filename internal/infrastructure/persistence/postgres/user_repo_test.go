package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

func TestUserRepository_FindByUsername(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	username := uniqueID(t)

	_, err := pool.Exec(ctx, `INSERT INTO users (username, password_hash) VALUES ($1, $2)`, username, "hash-de-prueba")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	repo := NewUserRepository(pool)
	user, err := repo.FindByUsername(ctx, username)
	if err != nil {
		t.Fatalf("FindByUsername() error = %v", err)
	}
	if user.Username != username || user.PasswordHash != "hash-de-prueba" {
		t.Errorf("user = %+v", user)
	}
	if user.ID == "" {
		t.Error("ID vacío")
	}
	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt vacío")
	}
}

func TestUserRepository_FindByUsername_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)

	_, err := repo.FindByUsername(context.Background(), uniqueID(t))
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("err = %v, quiero %v", err, domain.ErrUserNotFound)
	}
}
