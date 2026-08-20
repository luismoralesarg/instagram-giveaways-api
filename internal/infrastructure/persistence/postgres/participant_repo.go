package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

// ParticipantRepository implementa domain.ParticipantRepository contra
// Postgres.
type ParticipantRepository struct {
	pool *pgxpool.Pool
}

func NewParticipantRepository(pool *pgxpool.Pool) *ParticipantRepository {
	return &ParticipantRepository{pool: pool}
}

func (r *ParticipantRepository) Create(ctx context.Context, p *domain.Participant) error {
	const query = `
		INSERT INTO participants (campaign_id, instagram_user_id, username, source_type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	return r.pool.QueryRow(ctx, query, p.CampaignID, p.InstagramUserID, p.Username, p.SourceType).
		Scan(&p.ID, &p.CreatedAt)
}

func (r *ParticipantRepository) ExistsByCampaignAndInstagramUserID(ctx context.Context, campaignID, instagramUserID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM participants WHERE campaign_id = $1 AND instagram_user_id = $2
		)
	`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, campaignID, instagramUserID).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (r *ParticipantRepository) FindByID(ctx context.Context, campaignID, participantID string) (*domain.Participant, error) {
	const query = `
		SELECT id, campaign_id, instagram_user_id, username, source_type, is_excluded, excluded_reason, created_at
		FROM participants
		WHERE id = $1 AND campaign_id = $2
	`

	var p domain.Participant
	err := r.pool.QueryRow(ctx, query, participantID, campaignID).
		Scan(&p.ID, &p.CampaignID, &p.InstagramUserID, &p.Username, &p.SourceType, &p.IsExcluded, &p.ExcludedReason, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isPgError(err, pgErrInvalidTextRepresentation) {
			return nil, domain.ErrParticipantNotFound
		}
		return nil, err
	}

	return &p, nil
}

func (r *ParticipantRepository) Exclude(ctx context.Context, participantID, reason string) error {
	const query = `UPDATE participants SET is_excluded = true, excluded_reason = $2 WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, participantID, reason)
	return err
}

func (r *ParticipantRepository) ListByCampaign(ctx context.Context, campaignID string) ([]domain.Participant, error) {
	const query = `
		SELECT id, campaign_id, instagram_user_id, username, source_type, is_excluded, excluded_reason, created_at
		FROM participants
		WHERE campaign_id = $1
		ORDER BY created_at
	`

	rows, err := r.pool.Query(ctx, query, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []domain.Participant
	for rows.Next() {
		var p domain.Participant
		if err := rows.Scan(&p.ID, &p.CampaignID, &p.InstagramUserID, &p.Username, &p.SourceType, &p.IsExcluded, &p.ExcludedReason, &p.CreatedAt); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	return participants, rows.Err()
}

// ListEligibleByCampaign devuelve los participantes no excluidos, en un
// orden determinístico (ORDER BY id) — necesario para que UC-3.1 sea
// reproducible dado el mismo random_seed.
func (r *ParticipantRepository) ListEligibleByCampaign(ctx context.Context, campaignID string) ([]domain.Participant, error) {
	const query = `
		SELECT id, campaign_id, instagram_user_id, username, source_type, is_excluded, excluded_reason, created_at
		FROM participants
		WHERE campaign_id = $1 AND is_excluded = false
		ORDER BY id
	`

	rows, err := r.pool.Query(ctx, query, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []domain.Participant
	for rows.Next() {
		var p domain.Participant
		if err := rows.Scan(&p.ID, &p.CampaignID, &p.InstagramUserID, &p.Username, &p.SourceType, &p.IsExcluded, &p.ExcludedReason, &p.CreatedAt); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	return participants, rows.Err()
}
