package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool devuelve un pool contra el DATABASE_URL del entorno. Estos son
// tests de integración: se saltean si no hay Postgres disponible (en CI,
// .github/workflows/release.yml levanta uno y corre las migraciones antes
// de este paso — ver la nota ahí sobre qué prueba cada cosa).
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL no está seteada — test de integración salteado")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("conectar a postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

// uniqueID da un sufijo único por test para no colisionar con datos de
// otras corridas contra la misma base (ej. el índice único de media_id).
func uniqueID(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano())
}
