package domain

import "time"

// InstagramToken es el long-lived access token de la (única) cuenta de
// Instagram Business conectada — no hay soporte multi-cuenta (CLAUDE.md).
// Expira cada ~60 días (ver ARCHITECTURE.md §3) y se refresca
// automáticamente antes de vencer vía el job programado
// "refresh-instagram-token".
type InstagramToken struct {
	AccessToken string
	ExpiresAt   time.Time
	UpdatedAt   time.Time
}

// NeedsRefresh es true si al token le quedan `within` o menos para vencer
// (o ya venció).
func (t InstagramToken) NeedsRefresh(now time.Time, within time.Duration) bool {
	return !t.ExpiresAt.After(now.Add(within))
}
