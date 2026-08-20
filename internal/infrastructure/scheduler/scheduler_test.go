package scheduler

import (
	"context"
	"errors"
	"testing"
)

func TestScheduler_RunNow_ExecutesJob(t *testing.T) {
	s := New()
	var ran bool
	err := s.Register(Job{
		Name:     "test-job",
		Schedule: "@yearly", // no importa, RunNow no depende del schedule
		Run: func(ctx context.Context) error {
			ran = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if err := s.RunNow("test-job"); err != nil {
		t.Fatalf("RunNow() error = %v", err)
	}
	if !ran {
		t.Error("el job no se ejecutó")
	}
}

func TestScheduler_RunNow_UnknownJob(t *testing.T) {
	s := New()
	if err := s.RunNow("no-existe"); err == nil {
		t.Fatal("RunNow() error = nil, quiero un error para un job no registrado")
	}
}

func TestScheduler_Register_DuplicateName(t *testing.T) {
	s := New()
	job := Job{Name: "dup", Schedule: "@yearly", Run: func(ctx context.Context) error { return nil }}
	if err := s.Register(job); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := s.Register(job); err == nil {
		t.Fatal("Register() error = nil, quiero un error por nombre duplicado")
	}
}

func TestScheduler_Register_InvalidSchedule(t *testing.T) {
	s := New()
	err := s.Register(Job{Name: "bad", Schedule: "no es un cron válido", Run: func(ctx context.Context) error { return nil }})
	if err == nil {
		t.Fatal("Register() error = nil, quiero un error por expresión cron inválida")
	}
}

func TestScheduler_RunNow_JobError(t *testing.T) {
	s := New()
	boom := errors.New("boom")
	s.Register(Job{Name: "fails", Schedule: "@yearly", Run: func(ctx context.Context) error { return boom }})

	// RunNow loguea el error pero no lo propaga (correr un job siempre
	// "funciona" desde la perspectiva del scheduler; el fallo del job en
	// sí queda en el log, no interrumpe al caller).
	if err := s.RunNow("fails"); err != nil {
		t.Fatalf("RunNow() error = %v, quiero nil (el error del job se loguea, no se propaga)", err)
	}
}
