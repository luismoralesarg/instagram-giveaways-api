package instagram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// VerifySignature valida el header X-Hub-Signature-256 que Meta agrega a
// cada webhook: HMAC-SHA256 del body crudo, firmado con el App Secret.
func VerifySignature(appSecret string, body []byte, signatureHeader string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(signatureHeader, prefix) {
		return false
	}
	expected := strings.TrimPrefix(signatureHeader, prefix)

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	computed := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(computed))
}
