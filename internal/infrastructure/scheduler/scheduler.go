// Package scheduler es el módulo de tareas programadas: un registro de
// jobs con expresión cron, cada uno invocando un caso de uso. No vive en
// application/usecase porque no es un caso de uso en sí — es un mecanismo
// de disparo (delivery), igual que infrastructure/http/fiber dispara
// casos de uso por HTTP en vez de por reloj.
package scheduler

import (
	"context"
	"fmt"
	"log"

	"github.com/robfig/cron/v3"
)

// Job es una tarea programada: un nombre para loguearla, una expresión
// cron estándar de 5 campos, y la función que ejecuta.
type Job struct {
	Name     string
	Schedule string
	Run      func(ctx context.Context) error
}

// Scheduler ejecuta un conjunto de Job en background según su expresión
// cron, logueando inicio/éxito/error de cada corrida.
type Scheduler struct {
	cron *cron.Cron
	jobs map[string]Job
}

func New() *Scheduler {
	return &Scheduler{
		cron: cron.New(),
		jobs: map[string]Job{},
	}
}

// Register agrega un Job al scheduler. No lo ejecuta todavía — recién
// arranca a correr según su Schedule cuando se llama a Run.
func (s *Scheduler) Register(j Job) error {
	if _, exists := s.jobs[j.Name]; exists {
		return fmt.Errorf("ya hay un job registrado con el nombre %q", j.Name)
	}

	_, err := s.cron.AddFunc(j.Schedule, func() {
		s.runOnce(j)
	})
	if err != nil {
		return fmt.Errorf("registrar job %q: %w", j.Name, err)
	}

	s.jobs[j.Name] = j
	return nil
}

func (s *Scheduler) runOnce(j Job) {
	log.Printf("scheduler: job %q iniciando", j.Name)
	if err := j.Run(context.Background()); err != nil {
		log.Printf("scheduler: job %q error: %v", j.Name, err)
		return
	}
	log.Printf("scheduler: job %q OK", j.Name)
}

// RunNow ejecuta un job registrado inmediatamente, fuera de su schedule —
// para operar a demanda o para verificar que un job funciona sin esperar
// a que le toque el turno.
func (s *Scheduler) RunNow(name string) error {
	j, ok := s.jobs[name]
	if !ok {
		return fmt.Errorf("no hay ningún job registrado con el nombre %q", name)
	}
	s.runOnce(j)
	return nil
}

// Start arranca el scheduler en background (no bloquea).
func (s *Scheduler) Start() {
	log.Printf("scheduler: arrancando %d job(s)", len(s.jobs))
	s.cron.Start()
}

// Stop espera a que terminen los jobs en curso y frena el scheduler.
func (s *Scheduler) Stop() {
	<-s.cron.Stop().Done()
}
