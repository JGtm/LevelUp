//go:build integration

// Package prestige — test d'intégration SquadUsualContexts : résolution FR des
// playlists par IDENTIFIANT (V72-10 suite). Vérifie sur une DuckDB réelle que le
// trou de données "playlist_name_fr vide dans match_registry" (cas réel : "Quick
// Play", "Big Team Battle") est comblé par la traduction par playlist_id
// (metadata.asset_translations) plutôt que par nom — priorité : traduction par
// id > name_fr de match_registry > name EN, jamais vide.
package prestige

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// squadTestSharedReader implémente duckdb.SharedReader (structurellement, sans
// import du package platform/duckdb) sur une DB in-memory — même contrat que le
// SharedReader de prod : Get renvoie le handle + une fonction release no-op.
// Miroir de memSharedReader (internal/platform/duckdb/halo5/testsupport_test.go) ;
// dupliqué ici (packages de test disjoints, 2e copie tolérée — un helper commun
// exporté serait sur-ingénierie pour 2 sites).
type squadTestSharedReader struct{ db *sql.DB }

func (r *squadTestSharedReader) Get(_ context.Context) (*sql.DB, func(), error) {
	return r.db, func() {}, nil
}

// setupSquadUsualContextsDB crée le schéma minimal (match_registry,
// match_participants, v_match_full) nécessaire à SquadUsualContexts, sans passer
// par le pipeline de migration complet — seules les colonnes lues par la requête
// sont nécessaires ici.
func setupSquadUsualContextsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Exec séparés (une instruction par appel) — le driver duckdb ne garantit pas
	// l'exécution fiable d'un script multi-instructions en un seul Exec (cf.
	// internal/migration/helpers.go execScript/splitSQL, qui découpe pour la
	// même raison).
	stmts := []string{
		`CREATE TABLE match_registry (
			match_id         VARCHAR PRIMARY KEY,
			start_time       TIMESTAMP,
			start_time_utc   TIMESTAMP,
			playlist_id      VARCHAR,
			playlist_name    VARCHAR,
			playlist_name_fr VARCHAR,
			pair_name        VARCHAR,
			pair_name_fr     VARCHAR
		)`,
		`CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid     VARCHAR
		)`,
		`CREATE VIEW v_match_full AS SELECT mr.* FROM match_registry mr`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("create schema (%.40s...): %v", stmt, err)
		}
	}
	return db
}

// insertSquadMatch insère un match joué par tout le roster (un participant par
// xuid) avec la playlist/mode donnés.
func insertSquadMatch(t *testing.T, db *sql.DB, matchID, playlistID, playlistNameEN, playlistNameFR string, roster []string) {
	t.Helper()
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `INSERT INTO match_registry
		(match_id, start_time_utc, playlist_id, playlist_name, playlist_name_fr, pair_name, pair_name_fr)
		VALUES (?, ?, ?, ?, ?, 'Slayer', '')`,
		matchID, time.Now().UTC(), playlistID, playlistNameEN, playlistNameFR)
	if err != nil {
		t.Fatalf("insert match_registry: %v", err)
	}
	for _, x := range roster {
		if _, err := db.ExecContext(ctx, `INSERT INTO match_participants (match_id, xuid) VALUES (?, ?)`, matchID, x); err != nil {
			t.Fatalf("insert match_participants: %v", err)
		}
	}
}

// TestSquadUsualContexts_PlaylistFRByID couvre le cas réel V72-10 suite : une
// playlist dont playlist_name_fr est vide dans match_registry (« Quick Play »,
// trou de données) est néanmoins servie en FR grâce à la traduction par
// playlist_id (asset_translations, même résolveur que la page Carrière) — jamais
// l'EN brut quand une traduction par id existe.
func TestSquadUsualContexts_PlaylistFRByID(t *testing.T) {
	db := setupSquadUsualContextsDB(t)
	roster := []string{"x1", "x2"}
	const playlistID = "pl-quickplay-uuid"
	insertSquadMatch(t, db, "m1", playlistID, "Quick Play", "", roster)

	translatePlaylists := func(_ context.Context, ids []string) (map[string]string, error) {
		out := map[string]string{}
		for _, id := range ids {
			if id == playlistID {
				out[id] = "Partie rapide"
			}
		}
		return out, nil
	}

	p := NewPrestigeSquadMatchProvider(&squadTestSharedReader{db: db}).
		WithPlaylistTranslatorFR(translatePlaylists)

	playlists, _, err := p.SquadUsualContexts(context.Background(), roster, "halo_infinite", 60)
	if err != nil {
		t.Fatalf("SquadUsualContexts: %v", err)
	}
	if len(playlists) != 1 || playlists[0] != "Partie rapide" {
		t.Fatalf("playlists = %v, want [Partie rapide] (name_fr vide comblé par traduction par id)", playlists)
	}
}

// TestSquadUsualContexts_PlaylistFallbackWithoutTranslator vérifie la
// dégradation gracieuse : sans traducteur de playlists injecté (WithPlaylistTranslatorFR
// non appelé, comme avant V72-10 suite), le libellé COALESCE(playlist_name_fr,
// playlist_name) existant est servi tel quel (jamais vide).
func TestSquadUsualContexts_PlaylistFallbackWithoutTranslator(t *testing.T) {
	db := setupSquadUsualContextsDB(t)
	roster := []string{"x1", "x2"}
	insertSquadMatch(t, db, "m1", "pl-quickplay-uuid", "Quick Play", "", roster)

	p := NewPrestigeSquadMatchProvider(&squadTestSharedReader{db: db})
	playlists, _, err := p.SquadUsualContexts(context.Background(), roster, "halo_infinite", 60)
	if err != nil {
		t.Fatalf("SquadUsualContexts: %v", err)
	}
	if len(playlists) != 1 || playlists[0] != "Quick Play" {
		t.Fatalf("playlists = %v, want [Quick Play] (fallback EN sans traducteur)", playlists)
	}
}

// TestSquadUsualContexts_PlaylistNameFRAlreadyPresent vérifie qu'une playlist
// dont playlist_name_fr est déjà renseigné dans match_registry (pas de trou de
// données) est correctement traduite quand le traducteur par id fournit
// également une valeur — cohérence attendue, les deux sources devant s'accorder.
func TestSquadUsualContexts_PlaylistNameFRAlreadyPresent(t *testing.T) {
	db := setupSquadUsualContextsDB(t)
	roster := []string{"x1", "x2"}
	const playlistID = "pl-ranked-uuid"
	insertSquadMatch(t, db, "m1", playlistID, "Ranked Arena", "Arène Classée", roster)

	translatePlaylists := func(_ context.Context, ids []string) (map[string]string, error) {
		out := map[string]string{}
		for _, id := range ids {
			if id == playlistID {
				out[id] = "Arène Classée"
			}
		}
		return out, nil
	}

	p := NewPrestigeSquadMatchProvider(&squadTestSharedReader{db: db}).
		WithPlaylistTranslatorFR(translatePlaylists)

	playlists, _, err := p.SquadUsualContexts(context.Background(), roster, "halo_infinite", 60)
	if err != nil {
		t.Fatalf("SquadUsualContexts: %v", err)
	}
	if len(playlists) != 1 || playlists[0] != "Arène Classée" {
		t.Fatalf("playlists = %v, want [Arène Classée]", playlists)
	}
}
