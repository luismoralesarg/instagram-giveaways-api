package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

const defaultGraphBaseURL = "https://graph.facebook.com/v19.0"

// GraphClient implementa domain.InstagramClient contra la Graph API real.
//
// NOTA: paginación, autenticación por header y manejo de errores están
// cubiertos por tests contra un servidor HTTP fake (graph_client_test.go),
// pero esto no reemplaza probarlo contra una cuenta de Instagram real — no
// se pudo ejercitar contra una en este entorno de desarrollo (no hay
// INSTAGRAM_ACCESS_TOKEN disponible). El shape de la respuesta sigue la
// documentación pública de GET /{media-id}/comments; conviene validarlo
// contra una cuenta real antes de confiar en esto en producción.
type GraphClient struct {
	tokens     domain.InstagramTokenRepository
	httpClient *http.Client
	baseURL    string
}

// NewGraphClient recibe el repositorio del token en vez de un string fijo:
// el token se lee de Postgres en cada llamada, así un refresh persistido
// por el scheduler tiene efecto sin reiniciar el proceso de la API.
func NewGraphClient(tokens domain.InstagramTokenRepository) *GraphClient {
	return &GraphClient{
		tokens:     tokens,
		httpClient: http.DefaultClient,
		baseURL:    defaultGraphBaseURL,
	}
}

type commentsPage struct {
	Data []struct {
		Text string `json:"text"`
		From struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
	} `json:"data"`
	Paging struct {
		Next string `json:"next"`
	} `json:"paging"`
}

func (g *GraphClient) FetchComments(ctx context.Context, mediaID string) ([]domain.InstagramComment, error) {
	token, err := g.tokens.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("obtener instagram access token: %w", err)
	}

	var comments []domain.InstagramComment

	next := fmt.Sprintf("%s/%s/comments?fields=text,from", g.baseURL, url.PathEscape(mediaID))
	for next != "" {
		page, err := g.fetchPage(ctx, next, token.AccessToken)
		if err != nil {
			return nil, err
		}

		for _, c := range page.Data {
			comments = append(comments, domain.InstagramComment{
				InstagramUserID: c.From.ID,
				Username:        c.From.Username,
				Text:            c.Text,
			})
		}

		next = page.Paging.Next
	}

	return comments, nil
}

func (g *GraphClient) fetchPage(ctx context.Context, pageURL, accessToken string) (commentsPage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return commentsPage{}, err
	}
	// El token va en el header, no en la URL: evita romper la URL si el
	// token trae caracteres especiales (+, /, =) y evita que quede
	// expuesto en logs de servidores/proxies intermedios.
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return commentsPage{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return commentsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return commentsPage{}, graphAPIError(resp.StatusCode, body)
	}

	var page commentsPage
	if err := json.Unmarshal(body, &page); err != nil {
		return commentsPage{}, err
	}

	return page, nil
}
