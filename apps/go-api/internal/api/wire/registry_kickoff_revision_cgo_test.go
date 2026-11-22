//go:build cgo

package wire

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/sync/kickoff"
	"levelup/go-api/internal/testutil"
)

// Un titre sans base partagée n'a rien à rattraper : ni erreur, ni base créée.
func TestKickoffRevisionStep_SansBasePartagee(t *testing.T) {
	root := t.TempDir()
	reg := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: root}}
	applied, err := kickoffRevisionStep{r: reg}.Apply(context.Background(), titlePkg.DefaultSlug)
	if applied || err != nil {
		t.Fatalf("applied=%v err=%v, attendu rien", applied, err)
	}
	if _, err := os.Stat(titlePkg.NewPathResolver(root).SharedDBPath(titlePkg.DefaultSlug)); !os.IsNotExist(err) {
		t.Fatalf("la base a ete creee (err %v)", err)
	}
}

// Une base partagée sans provider câblé : l'étape refuse (elle ne lit ni n'écrit hors du provider).
func TestKickoffRevisionStep_SansProvider(t *testing.T) {
	root := t.TempDir()
	creerBase(t, titlePkg.NewPathResolver(root).SharedDBPath(titlePkg.DefaultSlug), `CREATE TABLE x (a INTEGER)`)
	reg := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: root}}
	if applied, err := (kickoffRevisionStep{r: reg}).Apply(context.Background(), titlePkg.DefaultSlug); applied || err == nil {
		t.Fatalf("applied=%v err=%v, attendu un refus", applied, err)
	}
}

// Un titre sans rejeu (capacité film.replay_artifact absente) : le rattrapage tourne sans lire
// d'artefact, garde un coup d'envoi rangé `film`, et range la révision.
func TestKickoffRevisionStep_TitreSansRejeu(t *testing.T) {
	const slug = "halo_5"
	root := t.TempDir()
	repo, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot: %v", err)
	}
	copierConfigDuTitre(t, repo, root, slug)
	sharedPath := titlePkg.NewPathResolver(root).SharedDBPath(slug)
	creerBase(t, sharedPath,
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP NOT NULL,
			start_time_utc TIMESTAMPTZ, duration_seconds INTEGER, real_start_time TIMESTAMP, real_start_source VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, first_joined_time TIMESTAMPTZ,
			last_leave_time TIMESTAMPTZ, present_at_beginning BOOLEAN, time_played_seconds INTEGER)`,
		`CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR, updated_at TIMESTAMP)`,
		`INSERT INTO match_registry VALUES ('m1', TIMESTAMP '2026-11-18 12:00:00', NULL, 600,
			TIMESTAMP '2026-11-18 12:00:26', 'film')`)
	prov, err := sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	t.Cleanup(func() { _ = prov.Close() })
	reg := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: root, SharedProvider: prov}}

	applied, err := kickoffRevisionStep{r: reg}.Apply(context.Background(), slug)
	if !applied || err != nil {
		t.Fatalf("applied=%v err=%v, attendu un rattrapage", applied, err)
	}
	db, release, err := prov.Get(context.Background())
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	defer release()
	var src, rev string
	if err := db.QueryRow(`SELECT real_start_source FROM match_registry WHERE match_id = 'm1'`).Scan(&src); err != nil || src != "film" {
		t.Fatalf("source = %q (err %v), attendu film garde", src, err)
	}
	if err := db.QueryRow(`SELECT value FROM sync_meta WHERE key = ?`, kickoff.SyncMetaKey).Scan(&rev); err != nil || rev != timeline.KickoffRev {
		t.Fatalf("revision = %q (err %v)", rev, err)
	}
}

// copierConfigDuTitre recopie config/titles/<slug> du dépôt vers une racine de test.
func copierConfigDuTitre(t *testing.T, from, to, slug string) {
	t.Helper()
	src := filepath.Join(from, "config", "titles", slug)
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		dst := filepath.Join(to, "config", "titles", slug, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o600)
	})
	if err != nil {
		t.Fatalf("copie de la config du titre: %v", err)
	}
}
