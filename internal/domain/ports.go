package domain

import "context"

// UserRepository persiste y consulta usuarios administradores. No hay
// método de alta: los usuarios se crean manualmente (ver cmd/seedadmin).
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*User, error)
}

// PasswordHasher hashea y compara contraseñas. Implementado en
// infrastructure/auth con bcrypt.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// TokenIssuer emite y valida los tokens de acceso (JWT) usados para
// autenticar al administrador en las rutas protegidas.
type TokenIssuer interface {
	Issue(userID string) (string, error)
	Parse(token string) (userID string, err error)
}

// CampaignRepository persiste y consulta campañas.
type CampaignRepository interface {
	Create(ctx context.Context, c *Campaign) error
	FindByID(ctx context.Context, id string) (*Campaign, error)

	// ExistsOpenByMediaID implementa la regla de UC-1.1: un media_id no
	// puede tener más de una campaña en draft o activa a la vez.
	ExistsOpenByMediaID(ctx context.Context, mediaID string) (bool, error)

	// ExistsActiveStoryCampaign y FindActiveStoryCampaign implementan la
	// regla acordada para UC-2.2: como máximo una campaña de tipo historia
	// puede estar activa a la vez (el webhook no permite desambiguar).
	ExistsActiveStoryCampaign(ctx context.Context) (bool, error)
	FindActiveStoryCampaign(ctx context.Context) (*Campaign, error)

	UpdateStatus(ctx context.Context, id string, status CampaignStatus) error
}

// ParticipantRepository persiste y consulta participantes de una campaña.
type ParticipantRepository interface {
	Create(ctx context.Context, p *Participant) error

	// ExistsByCampaignAndInstagramUserID implementa la dedup por
	// instagram_user_id (UC-2.1 y UC-2.2).
	ExistsByCampaignAndInstagramUserID(ctx context.Context, campaignID, instagramUserID string) (bool, error)

	FindByID(ctx context.Context, campaignID, participantID string) (*Participant, error)
	Exclude(ctx context.Context, participantID, reason string) error
	ListByCampaign(ctx context.Context, campaignID string) ([]Participant, error)
}

// InstagramComment es un comentario ya normalizado, tal como lo devuelve
// InstagramClient.FetchComments — desacoplado del formato específico de
// la respuesta de la Graph API.
type InstagramComment struct {
	InstagramUserID string
	Username        string
	Text            string
}

// InstagramClient encapsula la integración con la Graph API de Instagram.
// La única operación de esta fase es traer comentarios de un post
// (UC-2.1); las menciones de historia llegan por webhook, no por acá
// (ver §3 de ARCHITECTURE.md).
type InstagramClient interface {
	FetchComments(ctx context.Context, mediaID string) ([]InstagramComment, error)
}
