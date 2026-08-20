package instagram

import (
	"encoding/json"
	"fmt"
)

// graphAPIError intenta extraer el mensaje real de un error de la Graph
// API (siempre {"error": {"message": "..."}}), y si el body no tiene ese
// shape cae al código de estado HTTP. Compartido por GraphClient y
// TokenRefresher — ambos hablan con graph.facebook.com y reciben errores
// con el mismo formato.
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
