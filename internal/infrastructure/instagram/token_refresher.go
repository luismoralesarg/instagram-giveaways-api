package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// TokenRefresher implementa domain.InstagramTokenRefresher contra la Graph
// API real, con el flujo de intercambio de long-lived token de Facebook
// Login (grant_type=fb_exchange_token) — coherente con que GraphClient ya
// habla contra graph.facebook.com (Instagram Business conectado vía
// Facebook Page), no graph.instagram.com (Instagram Login, otro flujo).
//
// NOTA: no se pudo ejercitar contra una cuenta real en este entorno (no
// hay INSTAGRAM_APP_ID/INSTAGRAM_APP_SECRET reales disponibles). Cubierto
// por tests contra un servidor HTTP fake (token_refresher_test.go).
type TokenRefresher struct {
	appID      string
	appSecret  string
	httpClient *http.Client
	baseURL    string
}

func NewTokenRefresher(appID, appSecret string) *TokenRefresher {
	return &TokenRefresher{
		appID:      appID,
		appSecret:  appSecret,
		httpClient: http.DefaultClient,
		baseURL:    defaultGraphBaseURL,
	}
}

type exchangeTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"` // segundos
}

func (r *TokenRefresher) Refresh(ctx context.Context, currentToken string) (string, time.Duration, error) {
	u := fmt.Sprintf(
		"%s/oauth/access_token?grant_type=fb_exchange_token&client_id=%s&client_secret=%s&fb_exchange_token=%s",
		r.baseURL,
		url.QueryEscape(r.appID),
		url.QueryEscape(r.appSecret),
		url.QueryEscape(currentToken),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", 0, err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, graphAPIError(resp.StatusCode, body)
	}

	var parsed exchangeTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", 0, err
	}
	if parsed.AccessToken == "" {
		return "", 0, fmt.Errorf("graph api: respuesta sin access_token")
	}

	return parsed.AccessToken, time.Duration(parsed.ExpiresIn) * time.Second, nil
}
