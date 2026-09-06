//go:build integration

// Package duckdb — player_positions_repo_test.go : LA LECTURE des positions joueurs.
//
// LE REPO EST EN LECTURE SEULE DEPUIS LE 2026-09-06 (décision utilisateur 1) : `WriteMatch`
// (DELETE-then-INSERT sur le handle de lecture) a été supprimée, la table est projetée de
// l'artefact de rejeu par `persist.PlayerPositionsPersister` sous le lease RW. Les tests qui
// éprouvaient l'écriture ont donc suivi la fonction ; ceux qui restent éprouvent ce qui compte
// désormais — LA LECTURE PASSE-T-ELLE PAR LA VUE `_latest`.
//
// LES PASSES SONT SEMÉES EN INSERT PURS, comme le persister le fait : c'est la seule écriture
// que la table accepte, et cela permet d'éprouver la propriété qui n'existait pas avant — une
// projection remplace la précédente SANS effacer quoi que ce soit.
//
// Lancer avec : go test -tags=integration -run PlayerPositions ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	titlepkg "levelup/go-api/internal/domain/title"

	"levelup/go-api/internal/analysis/positions"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/migration"
)

const ppTestMatchID = "m_positions_001"

// newPlayerPositionsTestPlayerDB ouvre une mem DB, applique TOUTES les migrations
// shared (dont la conversion append-only de match_player_positions), puis construit un
// PlayerDB dont le SharedReader pointe sur cette conn (RW en legacy).
func newPlayerPositionsTestPlayerDB(t *testing.T) *PlayerDB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open mem: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if err := migration.RunForDB(sqlDB, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	shared := newTestDB(sqlDB, ":memory:")

	return &PlayerDB{
		Shared:       shared,
		SharedReader: LegacySharedReader(shared),
		XUID:         pTestXUID,
		Gamertag:     pTestGamertag,
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

func samplePositions() []positions.PlayerPosition {
	return []positions.PlayerPosition{
		{TimeMS: 0, X: 25.6, Y: 10.4, Z: 1.2, Team: 0},
		{TimeMS: 0, X: -6.0, Y: -24.0, Z: -2.8, Team: 1},
		{TimeMS: 20000, X: 34.8, Y: 13.5, Z: 0.5, Team: positions.TeamUnknown},
	}
}

// semerPasse écrit UNE passe de positions en INSERT purs — la seule forme d'écriture que la
// table accepte. `pass` et l'horodatage sont partagés par toutes les lignes de la passe : c'est
// exactement ce que fait `persist.PlayerPositionsPersister`, et c'est ce qui rend la vue
// `_latest` capable de retenir une génération entière.
func semerPasse(t *testing.T, pdb *PlayerDB, matchID, pass string, decalageSec int, pos []positions.PlayerPosition) {
	t.Helper()
	ctx := context.Background()
	for _, p := range pos {
		_, err := pdb.Shared.Exec(ctx, `
			INSERT INTO match_player_positions
				(match_id, positions_pass, written_at, time_ms, x, y, z, team)
			VALUES (?, ?, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) + INTERVAL (?) SECOND, ?, ?, ?, ?, ?)`,
			matchID, pass, decalageSec, p.TimeMS, p.X, p.Y, p.Z, p.Team)
		if err != nil {
			t.Fatalf("INSERT passe %s: %v", pass, err)
		}
	}
}

// TestPlayerPositionsRepo_LoadMatch_ServesLatestPass : le cas nominal ET la propriété neuve.
func TestPlayerPositionsRepo_LoadMatch_ServesLatestPass(t *testing.T) {
	pdb := newPlayerPositionsTestPlayerDB(t)
	repo := NewPlayerPositionsRepo(pdb)
	ctx := context.Background()

	semerPasse(t, pdb, ppTestMatchID, "passe-a", 0, samplePositions())

	got, err := repo.LoadMatch(ctx, ppTestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(positions) = %d, want 3", len(got))
	}
	// Ordonné par time_ms ASC : les 2 premiers à t=0, le 3e à t=20000.
	if got[0].TimeMS != 0 || got[2].TimeMS != 20000 {
		t.Fatalf("time order = [%d,..,%d], want [0,..,20000]", got[0].TimeMS, got[2].TimeMS)
	}
	// Valeurs float32 préservées (REAL = float32 natif, pas d'arrondi attendu).
	last := got[2]
	if last.X != 34.8 || last.Y != 13.5 || last.Z != 0.5 {
		t.Errorf("last pos = (%.2f,%.2f,%.2f), want (34.80,13.50,0.50)", last.X, last.Y, last.Z)
	}
	if last.Team != positions.TeamUnknown {
		t.Errorf("last.Team = %d, want %d (TeamUnknown)", last.Team, positions.TeamUnknown)
	}
	var hasTeam1 bool
	for _, p := range got {
		if p.Team == 1 {
			hasTeam1 = true
		}
	}
	if !hasTeam1 {
		t.Errorf("aucune position team=1 retrouvée")
	}
}

// TestPlayerPositionsRepo_SecondePasseRemplaceLaPremiere — LA PROPRIÉTÉ QUI REMPLACE LE
// DELETE-then-INSERT : une nouvelle projection supersède la précédente SANS rien effacer.
//
// Une lecture BRUTE de la table rendrait ici 4 lignes (3 + 1) : c'est exactement le piège de la
// règle ART n°2, et c'est pour cela que le repo lit `match_player_positions_latest`.
func TestPlayerPositionsRepo_SecondePasseRemplaceLaPremiere(t *testing.T) {
	pdb := newPlayerPositionsTestPlayerDB(t)
	repo := NewPlayerPositionsRepo(pdb)
	ctx := context.Background()

	semerPasse(t, pdb, ppTestMatchID, "passe-a", 0, samplePositions())
	semerPasse(t, pdb, ppTestMatchID, "passe-b", 60, []positions.PlayerPosition{
		{TimeMS: 5000, X: 1.0, Y: 2.0, Z: 3.0, Team: 0},
	})

	got, err := repo.LoadMatch(ctx, ppTestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(positions) = %d, want 1 — la vue _latest doit servir la DERNIÈRE passe, "+
			"pas les deux empilées (règle ART n°2)", len(got))
	}
	if got[0].TimeMS != 5000 {
		t.Errorf("got[0].TimeMS = %d, want 5000 (la passe la plus récente)", got[0].TimeMS)
	}

	// Et la passe précédente est TOUJOURS EN BASE : rien n'a été effacé (append-only).
	var brut int
	if err := pdb.Shared.QueryRow(ctx,
		`SELECT COUNT(*) FROM match_player_positions WHERE match_id = ?`, ppTestMatchID).Scan(&brut); err != nil {
		t.Fatalf("count brut: %v", err)
	}
	if brut != 4 {
		t.Errorf("table brute = %d ligne(s), want 4 — une projection ne doit RIEN effacer", brut)
	}
}

// LoadMatch sur un match absent retourne un slice vide, pas une erreur.
func TestPlayerPositionsRepo_LoadMatch_Empty(t *testing.T) {
	pdb := newPlayerPositionsTestPlayerDB(t)
	repo := NewPlayerPositionsRepo(pdb)

	got, err := repo.LoadMatch(context.Background(), "no_such_match")
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(positions) = %d, want 0", len(got))
	}
}

// TestPlayerPositionsRepo_CapabilityNotSupported_NoTable : un titre sans la table (ni sa vue)
// dégrade en ErrCapabilityNotSupported, jamais en erreur SQL brute.
func TestPlayerPositionsRepo_CapabilityNotSupported_NoTable(t *testing.T) {
	pdb := newPlayerPositionsTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Shared.Exec(ctx, "DROP VIEW match_player_positions_latest"); err != nil {
		t.Fatalf("drop view: %v", err)
	}
	if _, err := pdb.Shared.Exec(ctx, "DROP TABLE match_player_positions"); err != nil {
		t.Fatalf("drop: %v", err)
	}

	repo := NewPlayerPositionsRepo(pdb)
	if _, err := repo.LoadMatch(ctx, ppTestMatchID); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("LoadMatch err = %v, want ErrCapabilityNotSupported", err)
	}
}
