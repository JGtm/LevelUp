//go:build integration

package replayartifacts

// bombstats_integration_test.go — LE CROCHET DES STATISTIQUES D'ASSAUT, EXECUTE POUR DE VRAI.
//
// MEME RAISON QUE usage_integration_test.go : le chemin complet — artefact range sur disque ->
// relecture JSON -> transport -> persister INSERT-only -> vue `match_bomb_stats_latest` +
// `match_objective_events` — traverse trois paquets et deux serialisations. Un test a doubles
// laisserait une colonne renommee ou une vue cassee rendre le fil de l'eau MUET (WARN
// best-effort a chaque cycle) sans que rien ne rougisse.
//
// LE GATE PAR CAPABILITY EST TESTE SUR LES VRAIS TOML DU DEPOT : halo_infinite declare
// `film.bomb_stats`, halo_5 ne le declare pas.
//
// LES TROIS SILENCES SONT JUGES, un par test : mode non-Assaut (document sans `bombStats`),
// capability absente, artefact illisible.

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// artefactBombe forge un artefact d'ASSAUT minimal mais complet : deux joueurs, quatre
// statistiques mesurees (la cinquieme, `carriersKilled`, reste ABSENTE — c'est l'etat reel du
// chantier), et trois faits dates dont UN SANS ACTEUR.
func artefactBombe(t *testing.T, dir, matchID string) string {
	t.Helper()
	n := func(v int) *int { return &v }
	sec := func(v float64) *float64 { return &v }
	doc := replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion,
		MatchID:       matchID,
		BombStats: &replay.BombMatchStats{
			Players: []replay.BombPlayerStats{
				{XUID: "111", Detonations: n(2), Arms: n(1), Grabs: n(3), TimeAsCarrierSeconds: sec(41.5)},
				{XUID: "222", Detonations: n(0), Arms: n(0), Grabs: n(1), TimeAsCarrierSeconds: sec(4)},
			},
			Coverage: replay.BombStatsCoverage{
				DetonationsRead: true, CarryRead: true, ArmingsRead: true,
				Detonations: 2, Armings: 2, ArmingsAttributed: 1, ArmingsByDrop: 1,
				ArmingsNoBridge: 1, Periods: 4, Players: 2,
			},
		},
		BombEvents: []replay.BombEvent{
			{Type: replay.BombEventArmed, TimeMS: 299176, XUID: "111",
				ActorSource: replay.BombActorSourceDrop},
			{Type: replay.BombEventArmed, TimeMS: 782064}, // slot non ponte : PUBLIE sans acteur
			{Type: replay.BombEventDetonated, TimeMS: 304013, XUID: "111"},
		},
	}
	return ecrireArtefactBombe(t, dir, matchID, doc)
}

// artefactHorsAssaut forge un artefact SANS calque de bombe — le cas majoritaire d'un cycle.
func artefactHorsAssaut(t *testing.T, dir, matchID string) string {
	t.Helper()
	return ecrireArtefactBombe(t, dir, matchID, replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: matchID,
	})
}

func ecrireArtefactBombe(t *testing.T, dir, matchID string, doc replay.ReplayDocument) string {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal artefact: %v", err)
	}
	path := filepath.Join(dir, matchID+".json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write artefact: %v", err)
	}
	return path
}

// TestPersisterStatsBombe_UnMatchDAssautEcritSesLignesEtSesFaits : le chemin nominal.
func TestPersisterStatsBombe_UnMatchDAssautEcritSesLignesEtSesFaits(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)
	ctx := context.Background()

	persisterStatsBombe(ctx, d, lus(d,
		ArtefactRange{MatchID: "m-bombe", Path: artefactBombe(t, dir, "m-bombe")},
	))

	if acquis != 1 || relaches != 1 {
		t.Fatalf("writer acquis %d fois, relache %d fois — attendu 1 et 1 (burst unique)", acquis, relaches)
	}
	// LES LIGNES DE STATISTIQUES, VUE `_latest` (regle ART n2 : jamais la table brute).
	var deton, arms, grabs sql.NullInt64
	var carrier sql.NullFloat64
	var tues sql.NullInt64
	err := db.QueryRowContext(ctx, `
		SELECT bomb_detonations, bomb_arms, bomb_grabs, time_as_bomb_carrier_seconds,
		       bomb_carriers_killed
		FROM match_bomb_stats_latest WHERE match_id = 'm-bombe' AND xuid = '111'`).
		Scan(&deton, &arms, &grabs, &carrier, &tues)
	if err != nil {
		t.Fatalf("lecture match_bomb_stats_latest: %v", err)
	}
	if deton.Int64 != 2 || arms.Int64 != 1 || grabs.Int64 != 3 || carrier.Float64 != 41.5 {
		t.Errorf("joueur 111 : (%v, %v, %v, %v), attendu (2, 1, 3, 41.5)",
			deton.Int64, arms.Int64, grabs.Int64, carrier.Float64)
	}
	// ABSENT N'EST PAS ZERO, ET C'EST LE COEUR DU DESIGN : le noyau n'a pas lu les kills, la
	// colonne doit etre NULL — un 0 se sommerait dans un agregat sans que rien ne le signale.
	if tues.Valid {
		t.Errorf("bomb_carriers_killed = %d, attendu NULL (source non lue)", tues.Int64)
	}
	// LES FAITS DATES, dont UN SANS ACTEUR : il s'ecrit quand meme, sans ligne d'acteur.
	var faits, acteurs int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM match_objective_events
		WHERE match_id = 'm-bombe' AND objective_type = 'bomb'`).Scan(&faits); err != nil {
		t.Fatalf("count faits: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM match_objective_event_players WHERE match_id = 'm-bombe'`).
		Scan(&acteurs); err != nil {
		t.Fatalf("count acteurs: %v", err)
	}
	if faits != 3 || acteurs != 2 {
		t.Errorf("faits=%d acteurs=%d, attendu 3 et 2 : l'armement au slot non ponte est PUBLIE sans acteur",
			faits, acteurs)
	}
	// LA PROVENANCE EST ECRITE : la table refuse un fait qui ne dit pas d'ou il vient.
	var src, conf string
	if err := db.QueryRowContext(ctx, `
		SELECT source, confidence FROM match_objective_events
		WHERE match_id = 'm-bombe' AND event_type = 'bomb_armed' AND time_ms = 299176`).
		Scan(&src, &conf); err != nil {
		t.Fatalf("lecture provenance: %v", err)
	}
	if src != replay.BombSourceNavpointRing || conf == "" {
		t.Errorf("provenance de l'armement = (%q, %q), attendu (%q, non vide)",
			src, conf, replay.BombSourceNavpointRing)
	}
}

// TestPersisterStatsBombe_ModeHorsAssautNEcritRien : un artefact sans calque de bombe n'est PAS
// un echec — c'est le cas majoritaire. Aucun writer, aucune ligne.
func TestPersisterStatsBombe_ModeHorsAssautNEcritRien(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)

	persisterStatsBombe(context.Background(), d, lus(d,
		ArtefactRange{MatchID: "m-slayer", Path: artefactHorsAssaut(t, dir, "m-slayer")},
	))

	if acquis != 0 {
		t.Fatalf("writer acquis %d fois pour un lot sans Assaut, attendu 0", acquis)
	}
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM match_bomb_stats`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("un mode hors Assaut a ecrit %d ligne(s), attendu 0", n)
	}
}

// TestPersisterStatsBombe_TitreSansCapability : halo_5 ne declare pas `film.bomb_stats` dans SON
// capabilities.toml — le lot entier est ecarte AVANT toute projection et tout writer
// (title-agnostic, jamais un slug).
func TestPersisterStatsBombe_TitreSansCapability(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_5", &acquis, &relaches)

	persisterStatsBombe(context.Background(), d, lus(d,
		ArtefactRange{MatchID: "m-h5", Path: artefactBombe(t, dir, "m-h5")},
	))

	if acquis != 0 {
		t.Fatalf("writer acquis %d fois pour un titre sans capability, attendu 0", acquis)
	}
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM match_bomb_stats`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("un titre sans capability a ecrit %d ligne(s), attendu 0", n)
	}
}

// TestPersisterStatsBombe_ArtefactIllisibleNArretePasLeLot : l'echec d'UN match degrade ce match
// seul — le reste du lot s'ecrit, sous le meme burst.
func TestPersisterStatsBombe_ArtefactIllisibleNArretePasLeLot(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)

	persisterStatsBombe(context.Background(), d, lus(d,
		ArtefactRange{MatchID: "m-absent", Path: filepath.Join(dir, "inexistant.json")},
		ArtefactRange{MatchID: "m-ok", Path: artefactBombe(t, dir, "m-ok")},
	))

	if acquis != 1 {
		t.Fatalf("writer acquis %d fois, attendu 1 (le lot valide s'ecrit)", acquis)
	}
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM match_bomb_stats_latest WHERE match_id = 'm-ok'`).Scan(&n); err != nil {
		t.Fatalf("count m-ok: %v", err)
	}
	if n != 2 {
		t.Fatalf("m-ok = %d ligne(s), attendu 2 (deux joueurs)", n)
	}
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM match_bomb_stats WHERE match_id = 'm-absent'`).Scan(&n); err != nil {
		t.Fatalf("count m-absent: %v", err)
	}
	if n != 0 {
		t.Fatalf("m-absent = %d ligne(s), attendu 0", n)
	}
}
