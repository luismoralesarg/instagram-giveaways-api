package instagram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestTokenRefresher_Refresh_Success(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token": "nuevo-token", "expires_in": 5184000}`)) // 60 días
	}))
	defer server.Close()

	r := &TokenRefresher{appID: "app-id", appSecret: "app-secret", httpClient: server.Client(), baseURL: server.URL}
	token, expiresIn, err := r.Refresh(context.Background(), "token-viejo")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if token != "nuevo-token" {
		t.Errorf("token = %q, quiero %q", token, "nuevo-token")
	}
	if want := 60 * 24 * time.Hour; expiresIn != want {
		t.Errorf("expiresIn = %v, quiero %v", expiresIn, want)
	}

	if gotQuery.Get("grant_type") != "fb_exchange_token" {
		t.Errorf("grant_type = %q, quiero fb_exchange_token", gotQuery.Get("grant_type"))
	}
	if gotQuery.Get("client_id") != "app-id" {
		t.Errorf("client_id = %q, quiero app-id", gotQuery.Get("client_id"))
	}
	if gotQuery.Get("client_secret") != "app-secret" {
		t.Errorf("client_secret = %q, quiero app-secret", gotQuery.Get("client_secret"))
	}
	if gotQuery.Get("fb_exchange_token") != "token-viejo" {
		t.Errorf("fb_exchange_token = %q, quiero token-viejo", gotQuery.Get("fb_exchange_token"))
	}
}

func TestTokenRefresher_Refresh_GraphAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"message": "El token ya expiró, no se puede refrescar"}}`))
	}))
	defer server.Close()

	r := &TokenRefresher{appID: "a", appSecret: "s", httpClient: server.Client(), baseURL: server.URL}
	_, _, err := r.Refresh(context.Background(), "vencido")
	if err == nil {
		t.Fatal("Refresh() error = nil, quiero un error")
	}
	if !strings.Contains(err.Error(), "ya expiró") {
		t.Errorf("err = %q, debería incluir el mensaje real de la Graph API", err.Error())
	}
}

func TestTokenRefresher_Refresh_EmptyAccessTokenInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"expires_in": 100}`))
	}))
	defer server.Close()

	r := &TokenRefresher{appID: "a", appSecret: "s", httpClient: server.Client(), baseURL: server.URL}
	_, _, err := r.Refresh(context.Background(), "t")
	if err == nil {
		t.Fatal("Refresh() error = nil, quiero un error (respuesta sin access_token)")
	}
}
