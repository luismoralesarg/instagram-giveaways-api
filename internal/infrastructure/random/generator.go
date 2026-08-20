package random

import (
	"crypto/rand"
	"math/big"
	"time"
)

var maxInt63 = big.NewInt(1 << 62)

// Generator implementa domain.RandomGenerator con crypto/rand — la seed en
// sí se genera con una fuente criptográficamente segura, aunque el
// algoritmo de selección que la consume (domain.SelectWinners) use
// math/rand de forma determinística a partir de ella.
type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) Seed() int64 {
	n, err := rand.Int(rand.Reader, maxInt63)
	if err != nil {
		// Prácticamente inalcanzable (crypto/rand sin entropía disponible).
		// No hay reproducibilidad que perder acá: el seed que se use, sea
		// cual sea su origen, es el que va a quedar persistido en el Draw.
		return time.Now().UnixNano()
	}
	return n.Int64()
}
