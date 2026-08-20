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
	accessToken string
	httpClient  *http.Client
	baseURL     string
}

func NewGraphClient(accessToken string) *GraphClient {
	return &GraphClient{
		accessToken: accessToken,
		httpClient:  http.DefaultClient,
		baseURL:     defaultGraphBaseURL,
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
	var comments []domain.InstagramComment

	next := fmt.Sprintf("%s/%s/comments?fields=text,from", g.baseURL, url.PathEscape(mediaID))
	for next != "" {
		page, err := g.fetchPage(ctx, next)
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

func (g *GraphClient) fetchPage(ctx context.Context, pageURL string) (commentsPage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return commentsPage{}, err
	}
	// El token va en el header, no en la URL: evita romper la URL si el
	// token trae caracteres especiales (+, /, =) y evita que quede
	// expuesto en logs de servidores/proxies intermedios.
	req.Header.Set("Authorization", "Bearer "+g.accessToken)

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

// graphAPIError intenta extraer el mensaje real de un error de la Graph
// API (siempre {"error": {"message": "..."}}), y si el body no tiene ese
// shape cae al código de estado HTTP.
func graphAPIError(status int, body []byte) error {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return fmt.Errorf("graph api: %s (status %d)", errResp.Error.Message, status)
	}
	return fmt.Errorf("graph api: status %d", status)
}
