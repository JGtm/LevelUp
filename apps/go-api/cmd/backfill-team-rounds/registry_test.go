package main

// registry_test.go — LA JONCTION, OBSERVÉE. Ces tests ne valident pas la règle (c'est le
// rôle de decide_test.go) mais le CÂBLAGE : quelle colonne atterrit dans quel champ, et
// quels arguments partent vraiment vers DuckDB. Une permutation ici produirait un journal
// parfaitement correct et écrirait l'inverse de ce qui a été décidé.

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

// scannerDouble rend des valeurs FIXES et DISTINCTES pour prouver la correspondance
// position -> champ. Les trois valeurs sont différentes à dessein : avec 1/1/1 une
// permutation passerait inaperçue.
type scannerDouble struct{ v0, v1, tot int64 }

func (s scannerDouble) Scan(dest ...any) error {
	vals := []int64{s.v0, s.v1, s.tot}
	if len(dest) != len(vals) {
		return errors.New("le SELECT doit rendre exactement 3 colonnes")
	}
	for i, d := range dest {
		p, ok := d.(*sql.NullInt64)
		if !ok {
			return errors.New("destination inattendue")
		}
		*p = sql.NullInt64{Int64: vals[i], Valid: true}
	}
	return nil
}

func TestScanRegistryRounds_OrdreDesColonnes(t *testing.T) {
	got, found, err := scanRegistryRounds(scannerDouble{v0: 2, v1: 1, tot: 3})
	if err != nil || !found {
		t.Fatalf("scan: err=%v found=%v", err, found)
	}
	if got.Team0Won == nil || *got.Team0Won != 2 {
		t.Errorf("Team0Won = %v, want 2 (1re colonne du SELECT)", got.Team0Won)
	}
	if got.Team1Won == nil || *got.Team1Won != 1 {
		t.Errorf("Team1Won = %v, want 1 (2e colonne)", got.Team1Won)
	}
	if got.Total == nil || *got.Total != 3 {
		t.Errorf("Total = %v, want 3 (3e colonne)", got.Total)
	}
}

// nullScanner simule une ligne dont les trois colonnes sont NULL.
type nullScanner struct{}

func (nullScanner) Scan(dest ...any) error {
	for _, d := range dest {
		if p, ok := d.(*sql.NullInt64); ok {
			*p = sql.NullInt64{}
		}
	}
	return nil
}

func TestScanRegistryRounds_NullResteNull(t *testing.T) {
	got, found, err := scanRegistryRounds(nullScanner{})
	if err != nil || !found {
		t.Fatalf("scan: err=%v found=%v", err, found)
	}
	if got.Team0Won != nil || got.Team1Won != nil || got.Total != nil {
		t.Error("une colonne NULL ne doit JAMAIS devenir un zéro")
	}
}

type noRowsScanner struct{}

func (noRowsScanner) Scan(...any) error { return sql.ErrNoRows }

func TestScanRegistryRounds_MatchAbsent(t *testing.T) {
	_, found, err := scanRegistryRounds(noRowsScanner{})
	if err != nil || found {
		t.Errorf("match absent : found=%v err=%v, want false, nil", found, err)
	}
}

// execDouble capture le SQL et les arguments réellement transmis.
type execDouble struct {
	query string
	args  []any
}

type fakeResult struct{}

func (fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (fakeResult) RowsAffected() (int64, error) { return 1, nil }

func (e *execDouble) ExecContext(_ context.Context, q string, args ...any) (sql.Result, error) {
	e.query, e.args = q, args
	return fakeResult{}, nil
}

func TestWriteRounds_OrdreDesArguments(t *testing.T) {
	ex := &execDouble{}
	r := sqlRegistry{ex: ex}
	if err := r.WriteRounds(context.Background(), "m-42", 2, 1, 3); err != nil {
		t.Fatalf("WriteRounds: %v", err)
	}
	if len(ex.args) != 4 {
		t.Fatalf("%d arguments liés, 4 attendus : %v", len(ex.args), ex.args)
	}
	want := []any{2, 1, 3, "m-42"}
	for i := range want {
		if ex.args[i] != want[i] {
			t.Errorf("argument %d = %v, want %v (ordre : camp 0, camp 1, total, match_id)", i, ex.args[i], want[i])
		}
	}
	if !strings.Contains(ex.query, "WHERE match_id = ?") {
		t.Errorf("l'UPDATE doit cibler UNE ligne : %q", ex.query)
	}
}

func TestWriteRounds_SansDroitDEcritureRefuse(t *testing.T) {
	r := sqlRegistry{}
	if err := r.WriteRounds(context.Background(), "m-1", 1, 0, 1); err == nil {
		t.Error("écrire sans execer doit échouer, pas paniquer")
	}
}

func TestReadRounds_SansBaseRefuse(t *testing.T) {
	r := sqlRegistry{}
	if _, _, err := r.ReadRounds(context.Background(), "m-1"); err == nil {
		t.Error("lire sans base doit échouer proprement")
	}
}

func TestPendingIDsSQL_CibleLesLignesSansManches(t *testing.T) {
	if !strings.Contains(pendingIDsSQL, "rounds_total IS NULL") {
		t.Errorf("la liste de travail doit être « manches inconnues » : %q", pendingIDsSQL)
	}
	if !strings.Contains(orderClause, "ORDER BY match_id") {
		t.Errorf("l'ordre doit être déterministe pour que --limit soit reproductible : %q", orderClause)
	}
}

// Le filtre par variante est le DÉFAUT : sans lui, rattraper l'historique coûterait ~1 900
// appels d'API pour une poignée de lignes utiles. Les variantes doivent partir en arguments
// LIÉS, jamais interpolées dans le SQL — elles viennent d'un fichier de config, mais un
// fichier de config reste une entrée.
func TestPendingMatchIDs_FiltreParVarianteEstLie(t *testing.T) {
	if got := placeholders(3); got != "?, ?, ?" {
		t.Errorf("placeholders(3) = %q, want \"?, ?, ?\"", got)
	}
	if got := placeholders(0); got != "" {
		t.Errorf("placeholders(0) = %q, want vide", got)
	}
}
