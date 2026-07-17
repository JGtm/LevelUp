// Package wire — registry_monitoring_freshness_grouped_test.go : garde-rail E3
// (revue 2026-07). Le rapport de fraîcheur lit le dernier match de TOUS les
// joueurs suivis d'un titre en UNE requête groupée sur le shared reader, sans
// résoudre la moindre player-DB — donc SANS créer de DB pour les profils
// auth_only (sans match). Requiert le driver DuckDB (CGO).
package wire

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// seedFreshnessShared crée le shared d'un titre sous repoRoot avec
// match_registry (timestamp canonique) + match_participants. Retourne le path.
func seedFreshnessShared(t *testing.T, repoRoot, slug string, rows []struct {
	MatchID string
	XUID    string
	At      time.Time
}) string {
	t.Helper()
	path := titlePkg.NewPathResolver(repoRoot).SharedDBPath(slug)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir warehouse: %v", err)
	}
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	ctx := context.Background()
	if _, err := db.Exec(ctx, `CREATE TABLE match_registry (match_id VARCHAR, start_time_utc TIMESTAMP, start_time TIMESTAMP)`); err != nil {
		t.Fatalf("ddl registry: %v", err)
	}
	if _, err := db.Exec(ctx, `CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)`); err != nil {
		t.Fatalf("ddl participants: %v", err)
	}
	for _, r := range rows {
		if _, err := db.Exec(ctx, `INSERT INTO match_registry VALUES (?, ?, NULL)`, r.MatchID, r.At.UTC()); err != nil {
			t.Fatalf("insert registry: %v", err)
		}
		if _, err := db.Exec(ctx, `INSERT INTO match_participants VALUES (?, ?)`, r.MatchID, r.XUID); err != nil {
			t.Fatalf("insert participant: %v", err)
		}
	}
	_ = db.Close()
	duckdb.EvictAndCloseCached(path) // libère le verrou fichier (Windows) avant lecture
	t.Cleanup(func() { duckdb.EvictAndCloseCached(path) })
	return path
}

func TestLastMatchByXUID_GroupedNoPlayerDBCreation(t *testing.T) {
	tmp := t.TempDir()
	slug := "halo_infinite"
	t2 := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	t1 := t2.Add(-48 * time.Hour)
	t3 := time.Date(2026, 7, 10, 8, 0, 0, 0, time.UTC)
	seedFreshnessShared(t, tmp, slug, []struct {
		MatchID string
		XUID    string
		At      time.Time
	}{
		{"m1", "xuid-A", t1},
		{"m2", "xuid-A", t2}, // A : dernier = t2
		{"m3", "xuid-B", t3}, // B : dernier = t3
		// xuid-C : AUCUN match (profil auth_only) → absent du résultat.
	})

	r := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: tmp}}

	// UNE requête groupée couvrant A, B et le profil auth_only C.
	out, errMsg := r.lastMatchByXUID(context.Background(), slug, []string{"xuid-A", "xuid-B", "xuid-C"})
	if errMsg != "" {
		t.Fatalf("errMsg inattendu: %q", errMsg)
	}
	a, okA := out["xuid-A"]
	b, okB := out["xuid-B"]
	if !okA || !okB {
		t.Fatalf("A présent=%v, B présent=%v — les deux joueurs avec matchs doivent apparaître", okA, okB)
	}
	// Comparaison tz-invariante : le round-trip du driver applique un offset
	// uniforme sur un TIMESTAMP naïf. L'écart A−B doit égaler t2−t3, ce qui valide
	// à la fois le GROUP BY par xuid ET la sélection du MAX (m2 pour A, pas m1).
	if a.Sub(b) != t2.Sub(t3) {
		t.Errorf("écart A−B = %v, attendu %v (GROUP BY + MAX(canonical) par xuid)", a.Sub(b), t2.Sub(t3))
	}
	if _, ok := out["xuid-C"]; ok {
		t.Errorf("xuid-C (auth_only) ne devrait pas apparaître dans le résultat")
	}

	// Invariant clé E3 : AUCUNE player-DB créée (le rapport ne les résout plus).
	playersDir := filepath.Join(tmp, "data", "titles", slug, "players")
	if _, err := os.Stat(playersDir); !os.IsNotExist(err) {
		t.Errorf("répertoire players créé (%s) — le rapport ne doit résoudre aucune player-DB (err=%v)", playersDir, err)
	}
}
