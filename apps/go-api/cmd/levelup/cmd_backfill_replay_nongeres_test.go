package main

// cmd_backfill_replay_nongeres_test.go — LES MATCHS DE MODE NON GERE (Firefight) dans
// `backfill-replay` : la selection les ecarte et les compte (y compris sous --force et
// --only-existing), `--one` refuse, la purge retire les artefacts deja cuits (dry-run : liste).

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"
)

const (
	idGere    = "39910eb1-8c01-4e26-bc60-965cb2dcc789"
	idNonGere = "edc5daf6-0df8-4eb0-b578-b8eb892ce184"
)

// registreFirefightDeTest pose un registre partage minimal au chemin canonique et un film (manifeste a un
// chunk) par match.
func registreFirefightDeTest(t *testing.T) (*titlePkg.PathResolver, string) {
	t.Helper()
	root := t.TempDir()
	pr := titlePkg.NewPathResolver(root)
	shared := pr.SharedDBPath(titlePkg.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(shared), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", shared)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR, map_name VARCHAR, map_id VARCHAR, is_firefight BOOLEAN)`,
		`INSERT INTO match_registry VALUES ('` + idGere + `', 'Recharge', 'm1', FALSE), ('` + idNonGere + `', 'Cole Protocol', 'm2', TRUE)`,
		`CHECKPOINT`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	cache := pr.CacheRootDir()
	if err := os.MkdirAll(filepath.Join(cache, "film_manifests"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{idGere, idNonGere} {
		short := titlePkg.FilmShortMatchID(id)
		if err := os.WriteFile(filepath.Join(cache, "film_manifests", short+".json"), []byte(`{"chunks":[{}]}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return pr, cache
}

// TestReplaysACuire_EcarteLeModeNonGere — le Firefight sort de la selection, compte « mode non
// gere », sous toutes les formes de la passe.
func TestReplaysACuire_EcarteLeModeNonGere(t *testing.T) {
	pr, cache := registreFirefightDeTest(t)
	for _, o := range []replayBackfillOptions{
		{titleSlug: titlePkg.DefaultSlug},
		{titleSlug: titlePkg.DefaultSlug, force: true},
		{titleSlug: titlePkg.DefaultSlug, onlyExisting: true},
	} {
		var report replayBackfillReport
		cands, err := replaysACuire(context.Background(), pr, cache, o, &report)
		if err != nil {
			t.Fatalf("replaysACuire(%+v) : %v", o, err)
		}
		if len(cands) != 1 || cands[0].matchID != idGere {
			t.Fatalf("candidats %+v, attendu le seul match gere (%+v)", cands, o)
		}
		if report.nonGeres != 1 || report.horsRegistre != 0 {
			t.Fatalf("rapport %+v, attendu 1 mode non gere (%+v)", report, o)
		}
	}
}

// TestGarderUnMatchNonGere — `--one` refuse un match non gere avec la sentinelle, laisse passer
// un match gere ET un match inconnu du registre.
func TestGarderUnMatchNonGere(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, q := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR, is_firefight BOOLEAN)`,
		`INSERT INTO match_registry VALUES ('` + idGere + `', FALSE), ('` + idNonGere + `', TRUE), ('nul', NULL)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	if err := garderUnMatchNonGere(ctx, db, idNonGere); !errors.Is(err, replaybuild.ErrModeNonGere) {
		t.Fatalf("match Firefight : %v, attendu ErrModeNonGere", err)
	}
	for _, id := range []string{idGere, "nul", "inconnu"} {
		if err := garderUnMatchNonGere(ctx, db, id); err != nil {
			t.Errorf("%s : %v, attendu aucun refus", id, err)
		}
	}
}

// TestPurgerNonGeres — dry-run liste sans supprimer ; la purge retire l'artefact du Firefight et
// laisse celui du match gere.
func TestPurgerNonGeres(t *testing.T) {
	pr, _ := registreFirefightDeTest(t)
	slug := titlePkg.DefaultSlug
	for _, id := range []string{idGere, idNonGere} {
		p := pr.ReplayArtifactPath(slug, id)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	if err := purgerNonGeres(ctx, pr, replayBackfillOptions{titleSlug: slug, dryRun: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pr.ReplayArtifactPath(slug, idNonGere)); err != nil {
		t.Fatalf("dry-run ne doit rien supprimer : %v", err)
	}
	if err := purgerNonGeres(ctx, pr, replayBackfillOptions{titleSlug: slug}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pr.ReplayArtifactPath(slug, idNonGere)); !os.IsNotExist(err) {
		t.Fatalf("l'artefact Firefight devait etre supprime : %v", err)
	}
	if _, err := os.Stat(pr.ReplayArtifactPath(slug, idGere)); err != nil {
		t.Fatalf("l'artefact du match gere ne doit pas etre touche : %v", err)
	}
}
