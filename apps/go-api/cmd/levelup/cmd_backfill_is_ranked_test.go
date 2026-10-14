package main

import (
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

func lireIsRanked(t *testing.T, path, matchID string) bool {
	t.Helper()
	db, release, err := duckdb.OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer release()
	var v *bool
	if err := db.QueryRow(`SELECT is_ranked FROM match_registry WHERE match_id = ?`, matchID).Scan(&v); err != nil {
		t.Fatalf("lecture: %v", err)
	}
	return v != nil && *v
}

// TestBackfillIsRanked_Commande : --dry-run n'écrit rien ; la passe réelle classe les matchs à CSR
// et laisse les autres ; une seconde passe est sans effet. Base fichier sous un dépôt temporaire :
// aucune base réelle.
func TestBackfillIsRanked_Commande(t *testing.T) {
	repoRoot := t.TempDir()
	sharedPath := titlePkg.NewPathResolver(repoRoot).SharedDBPath(titlePkg.DefaultSlug)
	creerBaseDeTest(t, sharedPath,
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, is_ranked BOOLEAN)`,
		`CREATE TABLE match_csrs_latest (match_id VARCHAR, xuid VARCHAR, rating_type VARCHAR)`,
		`INSERT INTO match_registry VALUES ('m1', FALSE), ('m2', NULL), ('m3', TRUE), ('m4', FALSE)`,
		`INSERT INTO match_csrs_latest VALUES ('m1', 'x', 'CSR'), ('m2', 'x', 'CSR'), ('m3', 'x', 'CSR')`)
	cfg := &config.AppConfig{RepoRoot: repoRoot}

	if err := runBackfillIsRanked(cfg, []string{"--dry-run"}); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if lireIsRanked(t, sharedPath, "m1") || lireIsRanked(t, sharedPath, "m2") {
		t.Fatal("--dry-run a écrit")
	}
	for i := 0; i < 2; i++ { // seconde passe : idempotente
		if err := runBackfillIsRanked(cfg, nil); err != nil {
			t.Fatalf("passe %d: %v", i+1, err)
		}
	}
	for id, want := range map[string]bool{"m1": true, "m2": true, "m3": true, "m4": false} {
		if got := lireIsRanked(t, sharedPath, id); got != want {
			t.Errorf("%s is_ranked = %v, want %v", id, got, want)
		}
	}
}
