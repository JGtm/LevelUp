//go:build cgo

package wire

import (
	"context"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
)

// TestUnmanagedModeMatches_MemePredicatQueLaCuisson — le décompte de la file de régénération écarte d'emblée
// les matchs que la cuisson refuse par règle : le même prédicat (sqlfrag.SQLModeNonGere) que replayMatchIdentity,
// rendu en forme courte (celle du nom du fichier rangé).
func TestUnmanagedModeMatches_MemePredicatQueLaCuisson(t *testing.T) {
	repoRoot := t.TempDir()
	creerBase(t, title.NewPathResolver(repoRoot).SharedDBPath(title.DefaultSlug),
		`CREATE TABLE match_registry (match_id VARCHAR, map_name VARCHAR, map_id VARCHAR, is_firefight BOOLEAN)`,
		`INSERT INTO match_registry VALUES
			('edc5daf6-0df8-4eb0-b578-b8eb892ce184', 'Cole Protocol', 'b', TRUE),
			('39910eb1-8c01-4e26-bc60-965cb2dcc789', 'Recharge', 'a', FALSE),
			('7b7b7b7b-0000-4000-8000-000000000000', 'Sans drapeau', 'c', NULL)`)
	reg := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: repoRoot}}
	got, err := reg.unmanagedModeMatches(context.Background(), title.DefaultSlug)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got[title.FilmShortMatchID("edc5daf6-0df8-4eb0-b578-b8eb892ce184")]; !ok || len(got) != 1 {
		t.Fatalf("modes non gérés = %v, attendu le seul Firefight (forme courte)", got)
	}
}
