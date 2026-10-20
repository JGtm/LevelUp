//go:build integration

package sync

// citations_relance_integration_test.go — LA RELANCE DES CITATIONS DU FILM À LA LIVRAISON, sur
// une vraie base joueur (fichier, lease, schéma posé par OpenPlayerDB) : un match calculé avant
// son film est recalculé dès que ses dérivations sont écrites, une seule fois ; un match jamais
// calculé est laissé au post-sync du joueur ; une base tenue n'est pas attendue au-delà de la
// borne.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/citationsfilm"
)

func relanceDeps(f *pipelineFixture, repoRoot string) citationsfilm.RelanceDeps {
	return citationsfilm.RelanceDeps{
		LireShared:   func(_ context.Context, _ string, fn func(*sql.DB)) { fn(f.shared) },
		MetaDB:       func() *sql.DB { return f.metadata },
		LireJoueur:   duckdbpkg.OpenReadForQuery,
		OuvrirJoueur: OpenPlayerDB,
		Calculer: func(ctx context.Context, metaDB, sharedDB, playerDB *sql.DB, xuid string, ids []string) error {
			return RecalculerCitations(ctx, RecalculCitations{
				RepoRoot: repoRoot, TitleSlug: "halo_infinite", XUID: xuid,
				MetaDB: metaDB, SharedDB: sharedDB, PlayerDB: playerDB, MatchIDs: ids,
			})
		},
		Attente: 200 * time.Millisecond,
	}
}

// preparerBaseJoueur crée la base joueur fichier avec la table des citations (forme legacy de la
// fixture) convertie en append-only, comme les migrations joueur la laissent en production.
func preparerBaseJoueur(t *testing.T, path string) {
	t.Helper()
	h, err := OpenPlayerDB(path)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer h.Close()
	if _, err := h.SQLDb().Exec(`CREATE TABLE IF NOT EXISTS match_citations (
		match_id VARCHAR NOT NULL, citation_name_norm VARCHAR NOT NULL, value INTEGER DEFAULT 1,
		created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
		PRIMARY KEY (match_id, citation_name_norm))`); err != nil {
		t.Fatalf("table des citations: %v", err)
	}
	if err := migration.EnsureMatchCitationsAppendOnly(h.SQLDb()); err != nil {
		t.Fatalf("EnsureMatchCitationsAppendOnly: %v", err)
	}
}

// calculerAvantLeFilm pose une première génération de citations dans la base joueur fichier,
// comme le post-sync du joueur avant que son film soit dérivé.
func calculerAvantLeFilm(t *testing.T, f *pipelineFixture, repoRoot, path string) time.Time {
	t.Helper()
	h, err := OpenPlayerDB(path)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer h.Close()
	if err := RecalculerCitations(context.Background(), RecalculCitations{
		RepoRoot: repoRoot, TitleSlug: "halo_infinite", XUID: fixXUID,
		MetaDB: f.metadata, SharedDB: f.shared, PlayerDB: h.SQLDb(), MatchIDs: []string{fixM1},
	}); err != nil {
		t.Fatalf("premier calcul: %v", err)
	}
	var calcule time.Time
	if err := h.SQLDb().QueryRow(`SELECT MAX(written_at) FROM match_citations_latest WHERE match_id = ?`, fixM1).
		Scan(&calcule); err != nil {
		t.Fatalf("génération courante: %v", err)
	}
	return calcule
}

func TestRelanceCitations_RecalculeApresLeFilmPuisSeTait(t *testing.T) {
	f := buildPipelineFixture(t)
	repoRoot := t.TempDir()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	preparerBaseJoueur(t, path)
	joueur := citationsfilm.RelanceJoueur{XUID: fixXUID, PlayerDBPath: path}
	d := relanceDeps(f, repoRoot)
	ctx := context.Background()

	// Sans citations encore calculées pour ce match, la relance ne fait rien : le premier calcul
	// appartient au post-sync du joueur (il a lu ses récompenses).
	if issue, err := citationsfilm.RelancerPourJoueur(ctx, d, joueur, fixM1); err != nil || issue != domain.CitationsRelanceAJour {
		t.Fatalf("avant tout calcul : issue=%q err=%v, attendu a_jour", issue, err)
	}

	calcule := calculerAvantLeFilm(t, f, repoRoot, path)
	ajouterResumeUsage(t, f, calcule.Add(time.Millisecond))
	time.Sleep(20 * time.Millisecond)

	if issue, err := citationsfilm.RelancerPourJoueur(ctx, d, joueur, fixM1); err != nil || issue != domain.CitationsRelanceRecalculee {
		t.Fatalf("film dérivé après les citations : issue=%q err=%v, attendu recalculee", issue, err)
	}
	if issue, err := citationsfilm.RelancerPourJoueur(ctx, d, joueur, fixM1); err != nil || issue != domain.CitationsRelanceAJour {
		t.Fatalf("seconde relance : issue=%q err=%v, attendu a_jour", issue, err)
	}
}

func TestRelanceCitations_BaseTenueNEstPasAttendue(t *testing.T) {
	f := buildPipelineFixture(t)
	repoRoot := t.TempDir()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	preparerBaseJoueur(t, path)
	joueur := citationsfilm.RelanceJoueur{XUID: fixXUID, PlayerDBPath: path}
	tenu, err := dblease.AcquireWriterCtx(context.Background(), nil, path, dblease.KindPlayer)
	if err != nil {
		t.Fatalf("lease: %v", err)
	}
	defer func() { tenu.Release() }() // le dernier lease pris

	// Rien à faire (aucune citation encore) : réponse immédiate, sans attendre la base tenue.
	debut := time.Now()
	if issue, err := citationsfilm.RelancerPourJoueur(context.Background(), relanceDeps(f, repoRoot), joueur, fixM1); err != nil || issue != domain.CitationsRelanceAJour {
		t.Fatalf("base tenue, rien à faire : issue=%q err=%v, attendu a_jour", issue, err)
	}
	if time.Since(debut) >= 200*time.Millisecond {
		t.Fatalf("relance sans objet : %v d'attente, attendu aucune", time.Since(debut))
	}

	// Quelque chose à faire, base tenue : borne d'attente puis « joueur occupé ».
	tenu.Release()
	calcule := calculerAvantLeFilm(t, f, repoRoot, path)
	ajouterResumeUsage(t, f, calcule.Add(time.Millisecond))
	time.Sleep(20 * time.Millisecond)
	tenu, err = dblease.AcquireWriterCtx(context.Background(), nil, path, dblease.KindPlayer)
	if err != nil {
		t.Fatalf("lease: %v", err)
	}
	debut = time.Now()
	issue, err := citationsfilm.RelancerPourJoueur(context.Background(), relanceDeps(f, repoRoot), joueur, fixM1)
	if err != nil || issue != domain.CitationsRelanceJoueurOccupe {
		t.Fatalf("base tenue : issue=%q err=%v, attendu joueur_occupe", issue, err)
	}
	if time.Since(debut) > 5*time.Second {
		t.Fatalf("attente %v au-delà de la borne", time.Since(debut))
	}
}
