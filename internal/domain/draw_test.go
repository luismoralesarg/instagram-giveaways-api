package domain

import (
	"errors"
	"reflect"
	"testing"
)

func eligibleParticipants(n int) []Participant {
	out := make([]Participant, n)
	for i := range out {
		out[i] = Participant{ID: string(rune('a' + i)), InstagramUserID: string(rune('a' + i))}
	}
	return out
}

// La reproducibilidad (ARCHITECTURE.md §2.4) es la regla de negocio no
// negociable de todo UC-3.1: mismo pool + misma seed → mismo resultado.
func TestSelectWinners_ReproducibleGivenSameSeed(t *testing.T) {
	pool := eligibleParticipants(10)

	first, err := SelectWinners(pool, 3, 42)
	if err != nil {
		t.Fatalf("SelectWinners() error = %v", err)
	}
	second, err := SelectWinners(pool, 3, 42)
	if err != nil {
		t.Fatalf("SelectWinners() error = %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("dos corridas con la misma seed dieron resultados distintos:\n%+v\n%+v", first, second)
	}
}

func TestSelectWinners_NoRepeatedWinners(t *testing.T) {
	pool := eligibleParticipants(5)

	winners, err := SelectWinners(pool, 3, 7)
	if err != nil {
		t.Fatalf("SelectWinners() error = %v", err)
	}

	seen := map[string]bool{}
	for _, w := range winners {
		if seen[w.ParticipantID] {
			t.Fatalf("ParticipantID %q repetido en los ganadores: %+v", w.ParticipantID, winners)
		}
		seen[w.ParticipantID] = true
	}
}

func TestSelectWinners_PositionsAreOneBased(t *testing.T) {
	winners, err := SelectWinners(eligibleParticipants(5), 3, 1)
	if err != nil {
		t.Fatalf("SelectWinners() error = %v", err)
	}
	for i, w := range winners {
		if w.Position != i+1 {
			t.Errorf("winners[%d].Position = %d, quiero %d", i, w.Position, i+1)
		}
	}
}

func TestSelectWinners_InvalidWinnersCount(t *testing.T) {
	for _, n := range []int{0, 4, -1} {
		_, err := SelectWinners(eligibleParticipants(5), n, 1)
		if !errors.Is(err, ErrInvalidWinnersCount) {
			t.Errorf("winnersCount=%d: err = %v, quiero %v", n, err, ErrInvalidWinnersCount)
		}
	}
}

func TestSelectWinners_NotEnoughParticipants(t *testing.T) {
	_, err := SelectWinners(eligibleParticipants(2), 3, 1)
	if !errors.Is(err, ErrNotEnoughParticipants) {
		t.Fatalf("err = %v, quiero %v", err, ErrNotEnoughParticipants)
	}
}
