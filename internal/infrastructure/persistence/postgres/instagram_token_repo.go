package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

// InstagramTokenRepository implementa domain.InstagramTokenRepository
// contra una tabla de una sola fila (id fijo en 1).
type InstagramTokenRepository struct {
	pool *pgxpool.Pool
}

func NewInstagramTokenRepository(pool *pgxpool.Pool) *InstagramTokenRepository {
	return &InstagramTokenRepository{pool: pool}
}

func (r *InstagramTokenRepository) Get(ctx context.Context) (*domain.InstagramToken, error) {
	const query = `SELECT access_token, expires_at, updated_at FROM instagram_token WHERE id = 1`

	var t domain.InstagramToken
	err := r.pool.QueryRow(ctx, query).Scan(&t.AccessToken, &t.ExpiresAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInstagramTokenNotFound
		}
		return nil, err
	}

	return &t, nil
}

func (r *InstagramTokenRepository) Save(ctx context.Context, t *domain.InstagramToken) error {
	const query = `
		INSERT INTO instagram_token (id, access_token, expires_at)
		VALUES (1, $1, $2)
		ON CONFLICT (id) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			expires_at = EXCLUDED.expires_at,
			updated_at = now()
		RETURNING updated_at
	`

	return r.pool.QueryRow(ctx, query, t.AccessToken, t.ExpiresAt).Scan(&t.UpdatedAt)
}
