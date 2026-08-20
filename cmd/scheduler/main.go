// cmd/scheduler es el módulo de tareas programadas de la API: un binario
// standalone (mismo patrón que cmd/api y cmd/seedadmin) que registra jobs
// con expresión cron y los corre en background.
//
// Uso:
//
//	go run ./cmd/scheduler                          # arranca el scheduler (bloquea)
//	go run ./cmd/scheduler -run-once=<job-name>      # corre un job ya y sale, sin esperar su cron
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/application/usecase"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/config"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/infrastructure/instagram"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/infrastructure/persistence/postgres"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/infrastructure/scheduler"
)

// refreshInstagramTokenSchedule: todos los días a las 3am. El job en sí
// solo actúa si al token le quedan <= 10 días para vencer (ver
// defaultRefreshWindow en el caso de uso) — correrlo a diario es barato y
// da margen de sobra frente a una ventana de 60 días.
const refreshInstagramTokenSchedule = "0 3 * * *"

func main() {
	runOnce := flag.String("run-once", "", "nombre de un job para correr una vez ahora y salir, sin esperar su cron")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("conexión a postgres: %v", err)
	}
	defer pool.Close()

	instagramTokenRepo := postgres.NewInstagramTokenRepository(pool)
	tokenRefresher := instagram.NewTokenRefresher(cfg.InstagramAppID, cfg.InstagramAppSecret)
	refreshInstagramTokenUC := usecase.NewRefreshInstagramTokenIfNeeded(instagramTokenRepo, tokenRefresher)

	sched := scheduler.New()
	err = sched.Register(scheduler.Job{
		Name:     "refresh-instagram-token",
		Schedule: refreshInstagramTokenSchedule,
		Run: func(ctx context.Context) error {
			result, err := refreshInstagramTokenUC.Execute(ctx)
			if err != nil {
				return err
			}
			log.Printf("refresh-instagram-token: refreshed=%v expires_at=%s", result.Refreshed, result.ExpiresAt)
			return nil
		},
	})
	if err != nil {
		log.Fatalf("registrar jobs: %v", err)
	}

	if *runOnce != "" {
		if err := sched.RunNow(*runOnce); err != nil {
			log.Fatalf("run-once: %v", err)
		}
		return
	}

	sched.Start()
	log.Println("scheduler: corriendo, Ctrl+C para salir")

	stop, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	<-stop.Done()

	log.Println("scheduler: deteniendo...")
	sched.Stop()
}
