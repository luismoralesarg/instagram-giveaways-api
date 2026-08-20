package domain

import (
	"math/rand"
	"time"
)

// Draw es la ejecución de un sorteo sobre una campaña. Persiste siempre la
// semilla aleatoria usada (RandomSeed) — dado el mismo conjunto de
// elegibles y la misma semilla, el resultado es reproducible (ver
// ARCHITECTURE.md §2.4).
type Draw struct {
	ID           string
	CampaignID   string
	WinnersCount int
	RandomSeed   int64
	CreatedAt    time.Time
	Winners      []Winner
}

// Winner es un participante seleccionado por un Draw. InstagramUserID y
// Username son un snapshot al momento del sorteo (decisión confirmada):
// si el participante cambia de username después en Instagram, el
// resultado histórico anunciado no cambia.
type Winner struct {
	ID              string
	DrawID          string
	ParticipantID   string
	InstagramUserID string
	Username        string

	// Position es 1-based. Sin significado cuando WinnersCount == 1 (un
	// único ganador no tiene posición, ver UC-3.1) — no se expone en ese
	// caso en la capa HTTP.
	Position int
}

// SelectWinners implementa el algoritmo de selección de UC-3.1: partial
// Fisher-Yates sobre `eligible` usando un math/rand.Source sembrado con
// seed. eligible debe venir en un orden determinístico (ver
// ParticipantRepository.ListEligibleByCampaign, ORDER BY id) — el orden de
// entrada es tan parte de la reproducibilidad como la seed misma.
func SelectWinners(eligible []Participant, winnersCount int, seed int64) ([]Winner, error) {
	if winnersCount < 1 || winnersCount > 3 {
		return nil, ErrInvalidWinnersCount
	}
	if len(eligible) < winnersCount {
		return nil, ErrNotEnoughParticipants
	}

	pool := make([]Participant, len(eligible))
	copy(pool, eligible)

	rng := rand.New(rand.NewSource(seed))
	winners := make([]Winner, winnersCount)
	for i := 0; i < winnersCount; i++ {
		j := i + rng.Intn(len(pool)-i)
		pool[i], pool[j] = pool[j], pool[i]
		winners[i] = Winner{
			ParticipantID:   pool[i].ID,
			InstagramUserID: pool[i].InstagramUserID,
			Username:        pool[i].Username,
			Position:        i + 1,
		}
	}

	return winners, nil
}
