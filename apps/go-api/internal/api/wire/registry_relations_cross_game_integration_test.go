//go:build integration

// Package api — test d'intégration de la sélection du titre cross-jeu le plus
// pertinent. Couvre la branche que le test unitaire ne peut pas atteindre :
// l'agrégation réelle countForTitle → choix du titre où la co-occurrence est
// MAXIMALE (registry_relations_cross_game.go lignes 64-72). Seeds deux shared
// DuckDB réels via le PathResolver, en lecture seule best-effort.
package wire

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// seedSharedForTitle crée le fichier shared d'un titre sous repoRoot et y insère
// des rows match_participants (xuid global, ADR 0008). pairs = (matchID, xuid).
func seedSharedForTitle(t *testing.T, repoRoot, slug string, pairs [][2]string) {
	t.Helper()
	resolver := title.NewPathResolver(repoRoot, title.NewRegistry())
	path := resolver.SharedDBPath(slug)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir warehouse %s: %v", slug, err)
	}
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open shared %s: %v", slug, err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		duckdb.EvictAndCloseCached(path) // libère le verrou fichier (Windows) avant le TempDir cleanup
	})
	ctx := context.Background()
	if _, err := db.Exec(ctx, `CREATE TABLE IF NOT EXISTS match_participants (match_id VARCHAR, xuid VARCHAR)`); err != nil {
		t.Fatalf("ddl %s: %v", slug, err)
	}
	for _, p := range pairs {
		if _, err := db.Exec(ctx, `INSERT INTO match_participants VALUES (?, ?)`, p[0], p[1]); err != nil {
			t.Fatalf("insert %s: %v", slug, err)
		}
	}
}

// seedSharedWithoutParticipants crée un shared VALIDE (le fichier s'ouvre, donc
// OpenReadForQuery réussit) mais SANS table match_participants — comme un shard
// de titre fraîchement créé / partiellement migré. La requête de co-occurrence
// échoue alors (« table does not exist »), ce qui doit déclencher la branche
// erreur-requête de countForTitle (best-effort : titre sauté, jamais propagé).
func seedSharedWithoutParticipants(t *testing.T, repoRoot, slug string) {
	t.Helper()
	resolver := title.NewPathResolver(repoRoot, title.NewRegistry())
	path := resolver.SharedDBPath(slug)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir warehouse %s: %v", slug, err)
	}
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open shared %s: %v", slug, err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		duckdb.EvictAndCloseCached(path)
	})
	// Une table arbitraire suffit à matérialiser un fichier DuckDB valide,
	// sans fournir match_participants.
	if _, err := db.Exec(context.Background(), `CREATE TABLE schema_marker (k VARCHAR)`); err != nil {
		t.Fatalf("ddl marker %s: %v", slug, err)
	}
}

// coocPairs génère, pour chaque match commun à `me` et `opp`, les deux rows
// (me + opp) avec un match_id unique préfixé. n matchs communs.
func coocPairs(prefix, me, opp string, n int) [][2]string {
	out := make([][2]string, 0, n*2)
	for i := 0; i < n; i++ {
		mid := prefix + string(rune('a'+i))
		out = append(out, [2]string{mid, me}, [2]string{mid, opp})
	}
	return out
}

// TestCooccurrencesByXUID_PicksMaxTitle : opp1 co-occurre 5× avec moi sur halo_5
// et 3× sur halo_other (tous deux >= seuil). Le hit retenu doit nommer halo_5
// (co-occurrence MAX) avec MatchesTogether=5. opp2 co-occurre 1× partout (< seuil
// défaut 3) → absent. Le titre courant (halo_infinite) et un titre interne sont
// présents mais exclus.
func TestCooccurrencesByXUID_PicksMaxTitle(t *testing.T) {
	repoRoot := t.TempDir()

	// halo_5 : opp1 ×5 (>= seuil), opp2 ×1 (< seuil)
	seedSharedForTitle(t, repoRoot, "halo_5", append(
		coocPairs("h5opp1_", "me", "opp1", 5),
		coocPairs("h5opp2_", "me", "opp2", 1)...,
	))
	// halo_other : opp1 ×3 (>= seuil mais < halo_5)
	seedSharedForTitle(t, repoRoot, "halo_other", coocPairs("hoopp1_", "me", "opp1", 3))
	// titre courant : opp1 ×9, mais doit être IGNORÉ (currentSlug)
	seedSharedForTitle(t, repoRoot, "halo_infinite", coocPairs("hicur_", "me", "opp1", 9))
	// titre interne : opp1 ×9, mais doit être IGNORÉ (IsInternal)
	seedSharedForTitle(t, repoRoot, "synthetic_title_b", coocPairs("synth_", "me", "opp1", 9))

	reg := title.NewRegistry()
	reg.Register(&title.TitleDescriptor{Slug: "halo_infinite", Name: "Halo Infinite", Status: title.StatusActive})
	reg.Register(&title.TitleDescriptor{Slug: "halo_5", Name: "Halo 5", Status: title.StatusActive})
	reg.Register(&title.TitleDescriptor{Slug: "halo_other", Name: "Halo Other", Status: title.StatusActive})
	reg.Register(&title.TitleDescriptor{Slug: "synthetic_title_b", Name: "Synthetic B", Status: title.StatusActive, IsInternal: true})

	c := &crossGameCooccurrence{
		repoRoot:    repoRoot,
		timezone:    "UTC",
		currentSlug: "halo_infinite",
		myXUID:      "me",
		registry:    reg,
		resolver:    title.NewPathResolver(repoRoot, reg),
	}

	out := c.CooccurrencesByXUID(context.Background(), []string{"opp1", "opp2"})

	hit, ok := out["opp1"]
	if !ok {
		t.Fatalf("opp1 absent du résultat: %v", out)
	}
	if hit.TitleDisplayName != "Halo 5" {
		t.Errorf("opp1 titre = %q, want Halo 5 (co-occurrence max, jamais courant/interne)", hit.TitleDisplayName)
	}
	if hit.MatchesTogether != 5 {
		t.Errorf("opp1 MatchesTogether = %d, want 5", hit.MatchesTogether)
	}
	if _, ok := out["opp2"]; ok {
		t.Errorf("opp2 ne devrait pas passer le seuil (1 < %d)", 3)
	}
}

// TestCooccurrencesByXUID_SkipsTitleOnQueryError : un AUTRE titre actif dont le
// shared s'ouvre (fichier valide) mais N'A PAS la table match_participants → la
// requête de co-occurrence échoue. countForTitle doit avaler l'erreur (branche
// erreur-requête, distincte de l'erreur d'ouverture déjà couverte) et SAUTER ce
// titre sans rien propager. Un autre titre sain en parallèle doit, lui, produire
// son hit : on prouve que l'échec d'un titre n'empoisonne pas l'agrégat global.
func TestCooccurrencesByXUID_SkipsTitleOnQueryError(t *testing.T) {
	repoRoot := t.TempDir()

	// halo_5 : sain, opp1 ×4 (>= seuil) → doit produire un hit.
	seedSharedForTitle(t, repoRoot, "halo_5", coocPairs("h5opp1_", "me", "opp1", 4))
	// halo_broken : fichier valide mais sans match_participants → requête échoue.
	seedSharedWithoutParticipants(t, repoRoot, "halo_broken")

	reg := title.NewRegistry()
	reg.Register(&title.TitleDescriptor{Slug: "halo_infinite", Name: "Halo Infinite", Status: title.StatusActive})
	reg.Register(&title.TitleDescriptor{Slug: "halo_5", Name: "Halo 5", Status: title.StatusActive})
	reg.Register(&title.TitleDescriptor{Slug: "halo_broken", Name: "Halo Broken", Status: title.StatusActive})

	c := &crossGameCooccurrence{
		repoRoot:    repoRoot,
		timezone:    "UTC",
		currentSlug: "halo_infinite",
		myXUID:      "me",
		registry:    reg,
		resolver:    title.NewPathResolver(repoRoot, reg),
	}

	out := c.CooccurrencesByXUID(context.Background(), []string{"opp1"})

	hit, ok := out["opp1"]
	if !ok {
		t.Fatalf("opp1 absent: le titre sain doit survivre à l'échec d'un autre titre, got %v", out)
	}
	if hit.TitleDisplayName != "Halo 5" || hit.MatchesTogether != 4 {
		t.Errorf("opp1 hit = %+v, want {Halo 5, 4} (halo_broken sauté en silence)", hit)
	}
}
