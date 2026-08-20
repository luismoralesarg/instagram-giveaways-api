package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

// CampaignRepository implementa domain.CampaignRepository contra Postgres.
type CampaignRepository struct {
	pool *pgxpool.Pool
}

func NewCampaignRepository(pool *pgxpool.Pool) *CampaignRepository {
	return &CampaignRepository{pool: pool}
}

func (r *CampaignRepository) Create(ctx context.Context, c *domain.Campaign) error {
	const query = `
		INSERT INTO campaigns (type, media_id, name, must_follow, min_mentions, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(ctx, query, c.Type, c.MediaID, c.Name, c.MustFollow, c.MinMentions, c.Status).
		Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		if isPgError(err, pgErrUniqueViolation) {
			return domain.ErrDuplicateMediaID
		}
		return err
	}

	return nil
}

func (r *CampaignRepository) FindByID(ctx context.Context, id string) (*domain.Campaign, error) {
	const query = `
		SELECT id, type, media_id, name, must_follow, min_mentions, status, created_at
		FROM campaigns
		WHERE id = $1
	`

	var c domain.Campaign
	err := r.pool.QueryRow(ctx, query, id).
		Scan(&c.ID, &c.Type, &c.MediaID, &c.Name, &c.MustFollow, &c.MinMentions, &c.Status, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isPgError(err, pgErrInvalidTextRepresentation) {
			return nil, domain.ErrCampaignNotFound
		}
		return nil, err
	}

	return &c, nil
}

func (r *CampaignRepository) ExistsOpenByMediaID(ctx context.Context, mediaID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM campaigns WHERE media_id = $1 AND status IN ('draft', 'activa')
		)
	`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, mediaID).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (r *CampaignRepository) ExistsActiveStoryCampaign(ctx context.Context) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM campaigns WHERE type = 'historia' AND status = 'activa'
		)
	`

	var exists bool
	if err := r.pool.QueryRow(ctx, query).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (r *CampaignRepository) FindActiveStoryCampaign(ctx context.Context) (*domain.Campaign, error) {
	const query = `
		SELECT id, type, media_id, name, must_follow, min_mentions, status, created_at
		FROM campaigns
		WHERE type = 'historia' AND status = 'activa'
		LIMIT 1
	`

	var c domain.Campaign
	err := r.pool.QueryRow(ctx, query).
		Scan(&c.ID, &c.Type, &c.MediaID, &c.Name, &c.MustFollow, &c.MinMentions, &c.Status, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCampaignNotFound
		}
		return nil, err
	}

	return &c, nil
}

func (r *CampaignRepository) UpdateStatus(ctx context.Context, id string, status domain.CampaignStatus) error {
	const query = `UPDATE campaigns SET status = $2 WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		if constraint, ok := uniqueViolationConstraint(err); ok && constraint == "campaigns_single_active_story_idx" {
			return domain.ErrActiveStoryCampaignExists
		}
		return err
	}

	return nil
}
