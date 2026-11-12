package duckdb

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/port"
)

func TestNotFound_MarqueLAbsenceEtGardeLeTexte(t *testing.T) {
	wrapped := fmt.Errorf("MatchViewRepo.GetMatchMeta: %w", sql.ErrNoRows)
	got := NotFound(wrapped)
	if !errors.Is(got, port.ErrNotFound) {
		t.Errorf("l'absence doit être reconnue comme port.ErrNotFound : %v", got)
	}
	if !errors.Is(got, sql.ErrNoRows) {
		t.Errorf("la cause du moteur reste reconnaissable pour les appelants qui la testent encore : %v", got)
	}
	if got.Error() != wrapped.Error() {
		t.Errorf("texte = %q, attendu inchangé %q", got.Error(), wrapped.Error())
	}
}

func TestNotFound_UnePanneNEstPasUneAbsence(t *testing.T) {
	boom := errors.New("Catalog Error: Table does not exist")
	if got := NotFound(boom); got != boom || errors.Is(got, port.ErrNotFound) {
		t.Errorf("une panne doit rester telle quelle : %v", got)
	}
	if NotFound(nil) != nil {
		t.Error("NotFound(nil) doit rendre nil")
	}
}
