package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

const defaultGraphBaseURL = "https://graph.facebook.com/v19.0"

// GraphClient implementa domain.InstagramClient contra la Graph API real.
//
// NOTA: no se pudo ejercitar contra una cuenta de Instagram real en este
// entorno de desarrollo (no hay INSTAGRAM_ACCESS_TOKEN disponible). El
// shape de la respuesta sigue la documentación pública de
// GET /{media-id}/comments, pero conviene validarlo contra una cuenta real
// antes de depender de esto en producción.
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

	url := fmt.Sprintf("%s/%s/comments?fields=text,from&access_token=%s", g.baseURL, mediaID, g.accessToken)
	for url != "" {
		page, next, err := g.fetchPage(ctx, url)
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

		url = next
	}

	return comments, nil
}

func (g *GraphClient) fetchPage(ctx context.Context, url string) (commentsPage, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return commentsPage{}, "", err
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return commentsPage{}, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return commentsPage{}, "", fmt.Errorf("graph api: status %d", resp.StatusCode)
	}

	var page commentsPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return commentsPage{}, "", err
	}

	return page, page.Paging.Next, nil
}
