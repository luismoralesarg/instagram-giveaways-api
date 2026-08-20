package domain

import (
	"context"
	"time"
)

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

	// ListEligibleByCampaign devuelve los participantes no excluidos, en un
	// orden determinístico (ORDER BY id) — necesario para que UC-3.1 sea
	// reproducible dado el mismo random_seed.
	ListEligibleByCampaign(ctx context.Context, campaignID string) ([]Participant, error)
}

// DrawRepository persiste y consulta sorteos y sus ganadores.
type DrawRepository interface {
	// Create persiste el Draw y todos sus Winner de forma atómica.
	Create(ctx context.Context, d *Draw) error
	FindByID(ctx context.Context, campaignID, drawID string) (*Draw, error)
}

// RandomGenerator produce la semilla aleatoria que cada Draw persiste (ver
// ARCHITECTURE.md §2.4). Implementado en infrastructure/random con
// crypto/rand — no confundir con el math/rand seedeado con esa semilla
// dentro de domain.SelectWinners, que es el que necesita ser
// determinístico.
type RandomGenerator interface {
	Seed() int64
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

// InstagramTokenRepository persiste el único InstagramToken de la cuenta
// (no hay soporte multi-cuenta, ver CLAUDE.md) — una tabla de una sola
// fila. GraphClient lo consulta en cada llamada en vez de guardar el
// token en memoria, para que un refresh persistido tenga efecto sin
// reiniciar el proceso de la API.
type InstagramTokenRepository interface {
	Get(ctx context.Context) (*InstagramToken, error)
	Save(ctx context.Context, token *InstagramToken) error
}

// InstagramTokenRefresher intercambia un long-lived token vigente por uno
// nuevo con una expiración fresca, contra la Graph API
// (grant_type=fb_exchange_token — ver ARCHITECTURE.md §3).
type InstagramTokenRefresher interface {
	Refresh(ctx context.Context, currentToken string) (newAccessToken string, expiresIn time.Duration, err error)
}
