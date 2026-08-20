// Comando de alta manual del primer InstagramToken (no hay flujo de login
// OAuth en esta app — el long-lived token inicial se obtiene a mano, fuera
// del sistema, siguiendo el flujo de Meta for Developers). Uso:
//
//	DATABASE_URL=... go run ./cmd/seedinstagramtoken -token=EAAB... [-expires-in-days=60]
//
// A partir de acá, cmd/scheduler lo mantiene fresco automáticamente
// (ver ARCHITECTURE.md §3).
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/infrastructure/persistence/postgres"
)

func main() {
	token := flag.String("token", "", "long-lived access token obtenido manualmente vía Meta for Developers")
	expiresInDays := flag.Int("expires-in-days", 60, "días hasta que vence el token (Meta no siempre lo informa al generarlo; 60 es el default habitual)")
	flag.Parse()

	if *token == "" {
		log.Fatal("-token es requerido")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("falta la variable de entorno DATABASE_URL")
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("conexión a postgres: %v", err)
	}
	defer pool.Close()

	repo := postgres.NewInstagramTokenRepository(pool)
	instagramToken := &domain.InstagramToken{
		AccessToken: *token,
		ExpiresAt:   time.Now().Add(time.Duration(*expiresInDays) * 24 * time.Hour),
	}
	if err := repo.Save(ctx, instagramToken); err != nil {
		log.Fatalf("guardar instagram token: %v", err)
	}

	log.Printf("instagram token cargado, expira %s", instagramToken.ExpiresAt.Format(time.RFC3339))
}
