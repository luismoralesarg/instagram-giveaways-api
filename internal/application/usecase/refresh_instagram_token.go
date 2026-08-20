package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

// defaultRefreshWindow: si al token le quedan 10 días o menos, se
// refresca. Meta recomienda refrescar con margen — el token debe seguir
// vigente al momento de pedir el intercambio (fb_exchange_token no
// funciona sobre un token ya vencido).
const defaultRefreshWindow = 10 * 24 * time.Hour

type RefreshInstagramTokenResult struct {
	Refreshed bool
	ExpiresAt time.Time
}

// RefreshInstagramTokenIfNeeded es el job "refresh-instagram-token"
// (ver ARCHITECTURE.md §3: el sistema debe soportar refresh programado y
// alertar si el token está por vencer — acá "alertar" es, por ahora, el
// log de cmd/scheduler, no hay canal de notificaciones en el proyecto).
type RefreshInstagramTokenIfNeeded struct {
	tokens        domain.InstagramTokenRepository
	refresher     domain.InstagramTokenRefresher
	now           func() time.Time
	refreshWindow time.Duration
}

func NewRefreshInstagramTokenIfNeeded(tokens domain.InstagramTokenRepository, refresher domain.InstagramTokenRefresher) *RefreshInstagramTokenIfNeeded {
	return &RefreshInstagramTokenIfNeeded{
		tokens:        tokens,
		refresher:     refresher,
		now:           time.Now,
		refreshWindow: defaultRefreshWindow,
	}
}

func (uc *RefreshInstagramTokenIfNeeded) Execute(ctx context.Context) (RefreshInstagramTokenResult, error) {
	token, err := uc.tokens.Get(ctx)
	if err != nil {
		return RefreshInstagramTokenResult{}, err
	}

	if !token.NeedsRefresh(uc.now(), uc.refreshWindow) {
		return RefreshInstagramTokenResult{Refreshed: false, ExpiresAt: token.ExpiresAt}, nil
	}

	newAccessToken, expiresIn, err := uc.refresher.Refresh(ctx, token.AccessToken)
	if err != nil {
		return RefreshInstagramTokenResult{}, fmt.Errorf("refrescar instagram token: %w", err)
	}

	updated := &domain.InstagramToken{
		AccessToken: newAccessToken,
		ExpiresAt:   uc.now().Add(expiresIn),
	}
	if err := uc.tokens.Save(ctx, updated); err != nil {
		return RefreshInstagramTokenResult{}, err
	}

	return RefreshInstagramTokenResult{Refreshed: true, ExpiresAt: updated.ExpiresAt}, nil
}
