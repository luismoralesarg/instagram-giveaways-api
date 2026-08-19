// Comando de alta manual de administradores (no hay endpoint de registro,
// ver CLAUDE.md). Uso:
//
//	DATABASE_URL=... go run ./cmd/seedadmin -username=admin -password=...
//
// Si el username ya existe, actualiza el password_hash (sirve también para
// resetear una contraseña).
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/infrastructure/auth"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/infrastructure/persistence/postgres"
)

func main() {
	username := flag.String("username", "", "username del administrador")
	password := flag.String("password", "", "contraseña en texto plano (se hashea antes de guardar)")
	flag.Parse()

	if *username == "" || *password == "" {
		log.Fatal("-username y -password son requeridos")
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

	hash, err := auth.NewBcryptHasher().Hash(*password)
	if err != nil {
		log.Fatalf("hash de contraseña: %v", err)
	}

	const query = `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash
		RETURNING id
	`

	var id string
	if err := pool.QueryRow(ctx, query, *username, hash).Scan(&id); err != nil {
		log.Fatalf("guardar usuario: %v", err)
	}

	log.Printf("usuario %q listo (id=%s)", *username, id)
}
