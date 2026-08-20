package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

// DrawRepository implementa domain.DrawRepository contra Postgres.
type DrawRepository struct {
	pool *pgxpool.Pool
}

func NewDrawRepository(pool *pgxpool.Pool) *DrawRepository {
	return &DrawRepository{pool: pool}
}

// Create persiste el Draw y todos sus Winner en una transacción — o quedan
// los dos, o no queda ninguno.
func (r *DrawRepository) Create(ctx context.Context, d *domain.Draw) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // no-op si Commit tuvo éxito

	const insertDraw = `
		INSERT INTO draws (campaign_id, winners_count, random_seed)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	if err := tx.QueryRow(ctx, insertDraw, d.CampaignID, d.WinnersCount, d.RandomSeed).Scan(&d.ID, &d.CreatedAt); err != nil {
		return err
	}

	const insertWinner = `
		INSERT INTO winners (draw_id, participant_id, instagram_user_id, username, position)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	for i := range d.Winners {
		w := &d.Winners[i]
		w.DrawID = d.ID
		if err := tx.QueryRow(ctx, insertWinner, d.ID, w.ParticipantID, w.InstagramUserID, w.Username, w.Position).Scan(&w.ID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *DrawRepository) FindByID(ctx context.Context, campaignID, drawID string) (*domain.Draw, error) {
	const drawQuery = `
		SELECT id, campaign_id, winners_count, random_seed, created_at
		FROM draws
		WHERE id = $1 AND campaign_id = $2
	`

	var d domain.Draw
	err := r.pool.QueryRow(ctx, drawQuery, drawID, campaignID).
		Scan(&d.ID, &d.CampaignID, &d.WinnersCount, &d.RandomSeed, &d.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isPgError(err, pgErrInvalidTextRepresentation) {
			return nil, domain.ErrDrawNotFound
		}
		return nil, err
	}

	const winnersQuery = `
		SELECT id, draw_id, participant_id, instagram_user_id, username, position
		FROM winners
		WHERE draw_id = $1
		ORDER BY position
	`

	rows, err := r.pool.Query(ctx, winnersQuery, d.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var w domain.Winner
		if err := rows.Scan(&w.ID, &w.DrawID, &w.ParticipantID, &w.InstagramUserID, &w.Username, &w.Position); err != nil {
			return nil, err
		}
		d.Winners = append(d.Winners, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &d, nil
}
