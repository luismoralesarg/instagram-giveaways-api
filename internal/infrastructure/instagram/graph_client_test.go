package instagram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

// fakeTokenRepo evita depender de Postgres para testear GraphClient — solo
// necesitamos que devuelva un access token fijo.
type fakeTokenRepo struct {
	token *domain.InstagramToken
}

func (f *fakeTokenRepo) Get(ctx context.Context) (*domain.InstagramToken, error) {
	if f.token == nil {
		return nil, domain.ErrInstagramTokenNotFound
	}
	return f.token, nil
}

func (f *fakeTokenRepo) Save(ctx context.Context, t *domain.InstagramToken) error {
	f.token = t
	return nil
}

func tokenRepoWith(accessToken string) *fakeTokenRepo {
	return &fakeTokenRepo{token: &domain.InstagramToken{AccessToken: accessToken, ExpiresAt: time.Now().Add(time.Hour)}}
}

func TestGraphClient_FetchComments_SinglePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, quiero %q", got, "Bearer test-token")
		}
		if strings.Contains(r.URL.RawQuery, "access_token") {
			t.Errorf("el token no debería ir en la query string: %q", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": [
				{"text": "hola @amigo", "from": {"id": "u1", "username": "user1"}},
				{"text": "sin mencion", "from": {"id": "u2", "username": "user2"}}
			],
			"paging": {}
		}`))
	}))
	defer server.Close()

	client := &GraphClient{tokens: tokenRepoWith("test-token"), httpClient: server.Client(), baseURL: server.URL}
	comments, err := client.FetchComments(context.Background(), "media-123")
	if err != nil {
		t.Fatalf("FetchComments() error = %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("len(comments) = %d, quiero 2", len(comments))
	}
	if comments[0].InstagramUserID != "u1" || comments[0].Text != "hola @amigo" {
		t.Errorf("comments[0] = %+v", comments[0])
	}
}

func TestGraphClient_FetchComments_FollowsPagination(t *testing.T) {
	var requests int
	mux := http.NewServeMux()

	// httptest.NewUnstartedServer ya reserva el listener (y por lo tanto la
	// URL) antes de Start() — la necesitamos para que la respuesta de la
	// primera página pueda referenciar la URL absoluta de la segunda, tal
	// como hace paging.next en la Graph API real.
	server := httptest.NewUnstartedServer(mux)
	server.Start()
	defer server.Close()

	mux.HandleFunc("/media-123/comments", func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": [{"text": "p1", "from": {"id": "u1"}}], "paging": {"next": "` + server.URL + `/next-page"}}`))
	})
	mux.HandleFunc("/next-page", func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": [{"text": "p2", "from": {"id": "u2"}}], "paging": {}}`))
	})

	client := &GraphClient{tokens: tokenRepoWith("t"), httpClient: server.Client(), baseURL: server.URL}
	comments, err := client.FetchComments(context.Background(), "media-123")
	if err != nil {
		t.Fatalf("FetchComments() error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, quiero 2 (debe seguir paging.next)", requests)
	}
	if len(comments) != 2 || comments[0].InstagramUserID != "u1" || comments[1].InstagramUserID != "u2" {
		t.Fatalf("comments = %+v, quiero [u1, u2] en orden", comments)
	}
}

func TestGraphClient_FetchComments_GraphAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"message": "Invalid OAuth access token", "type": "OAuthException", "code": 190}}`))
	}))
	defer server.Close()

	client := &GraphClient{tokens: tokenRepoWith("expired"), httpClient: server.Client(), baseURL: server.URL}
	_, err := client.FetchComments(context.Background(), "media-123")
	if err == nil {
		t.Fatal("FetchComments() error = nil, quiero un error")
	}
	if !strings.Contains(err.Error(), "Invalid OAuth access token") {
		t.Errorf("err = %q, debería incluir el mensaje real de la Graph API", err.Error())
	}
}

func TestGraphClient_FetchComments_NonJSONErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("<html>servicio caído</html>"))
	}))
	defer server.Close()

	client := &GraphClient{tokens: tokenRepoWith("t"), httpClient: server.Client(), baseURL: server.URL}
	_, err := client.FetchComments(context.Background(), "media-123")
	if err == nil {
		t.Fatal("FetchComments() error = nil, quiero un error")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("err = %q, debería caer al código de estado cuando el body no es el shape esperado", err.Error())
	}
}

func TestGraphClient_FetchComments_NoTokenAvailable(t *testing.T) {
	client := &GraphClient{tokens: &fakeTokenRepo{}, httpClient: http.DefaultClient, baseURL: "http://unused"}
	_, err := client.FetchComments(context.Background(), "media-123")
	if err == nil {
		t.Fatal("FetchComments() error = nil, quiero un error (no hay token cargado)")
	}
}
