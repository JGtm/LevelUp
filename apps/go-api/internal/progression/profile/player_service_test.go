package profile

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/port"
)

// TestPlayerService_JoueurIntrouvable : un slug que le résolveur refuse est un refus typé de nature
// ProgErrPlayerNotFound, qui porte la cause, sur les deux opérations.
func TestPlayerService_JoueurIntrouvable(t *testing.T) {
	cause := errors.New("slug inconnu")
	svc := NewPlayerService(func(context.Context, string) (port.ProfileStore, string, error) { return nil, "", cause }, nil)

	_, err := svc.Build(context.Background(), "x", "halo_infinite", 30, time.Now())
	assertPlayerNotFound(t, "Build", err, cause)
	_, err = svc.ActivityCalendar(context.Background(), "x", time.Now().AddDate(0, 0, -7), time.Now())
	assertPlayerNotFound(t, "ActivityCalendar", err, cause)
}

func assertPlayerNotFound(t *testing.T, op string, err, cause error) {
	t.Helper()
	var pe *port.ProgressionError
	if !errors.As(err, &pe) || pe.Kind != port.ProgErrPlayerNotFound || !errors.Is(err, cause) {
		t.Errorf("%s : %v ; attendu ProgErrPlayerNotFound portant la cause", op, err)
	}
}
