//go:build integration

package replayartifacts

// usage_integration_test.go — LA PERSISTANCE DU RESUME D'USAGE, EXECUTEE POUR DE VRAI.
//
// POURQUOI UN TEST D'INTEGRATION. Le chemin complet — artefact range sur disque -> relecture
// JSON -> projection -> persister INSERT-only -> vues `_latest` — traverse trois paquets et
// deux serialisations. Un test a doubles laisserait un INSERT invalide, une colonne renommee
// ou une vue cassee rendre le fil de l'eau muet (WARN best-effort a chaque cycle) sans que
// rien ne rougisse : exactement le mode de panne silencieuse que t0film_integration_test.go
// ferme deja pour le report du T0. Ici la passe tourne sur une base migree par les VRAIES
// migrations (baseRegistre) et le test relit ce que les vues servent APRES.
//
// LE GATE PAR CAPABILITY EST TESTE SUR LES VRAIS TOML du depot (config/titles/) : halo_infinite
// declare `film.usage_summary`, halo_5 ne le declare pas — un test qui fabriquerait ses TOML
// prouverait la mecanique du gate, pas la configuration livree.

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// racineDepot remonte du package au depot (apps/go-api/internal/sync/replayartifacts -> racine)
// pour que capabilityUsageArmee lise les capabilities.toml LIVRES.
func racineDepot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	root := filepath.Join(wd, "..", "..", "..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, "config", "titles")); err != nil {
		t.Fatalf("racine du depot introuvable depuis %s: %v", wd, err)
	}
	return root
}

// artefactUsage forge un artefact minimal mais COMPLET pour la projection : un joueur, un
// socle d'ARME (prise nommee) et un socle de BONUS (occupation anonyme) — le piege central
// du chantier, present jusque dans le test de cablage.
func artefactUsage(t *testing.T, dir, matchID string) string {
	t.Helper()
	xuid := "111"
	doc := replay.ReplayDocument{
		SchemaVersion:   replay.SchemaVersion,
		MatchID:         matchID,
		FrameIntervalMS: 100,
		FrameCount:      1000,
		DurationMS:      100000,
		Roster:          []replay.RosterEntry{{XUID: xuid, FilmIndex: 0}},
		Tracks:          []replay.Track{{Slot: 1, XUID: xuid, StartFrame: 0, EndFrame: 900}},
		WeaponPads: []replay.WeaponPad{
			{Weapon: "0x11223344"},
			{Weapon: "powerup_camo"},
		},
		PadPickups: []replay.PadPickup{
			{Pad: 0, TLow: 10, THigh: 20, XUID: &xuid}, // arme, nommee
			{Pad: 1, TLow: 30, THigh: 40, XUID: nil},   // bonus, anonyme
		},
		GrappleLines: []replay.GrappleLine{{Slot: 1, T0: 50, T1: 60}},
	}
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

// depsUsage : le cablage minimal de la persistance, writer compte.
func depsUsage(t *testing.T, db *sql.DB, slug string, acquis, relaches *int) Deps {
	t.Helper()
	return Deps{
		Gamertag:  "testeur",
		RepoRoot:  racineDepot(t),
		TitleSlug: slug,
		AcquireWriter: func(context.Context) (*sql.DB, func(), error) {
			*acquis++
			return db, func() { *relaches++ }, nil
		},
	}
}

// TestPersisterResumesUsage_BurstUniqueEtVuesLatest : le chemin nominal du fil de l'eau —
// un lot de deux artefacts, UN seul segment writer, et les vues `_latest` servent ce que la
// projection a mesure, frontiere socle d'arme / socle de bonus comprise.
func TestPersisterResumesUsage_BurstUniqueEtVuesLatest(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)

	rapports := []artefactCuit{
		{matchID: "m-usage-1", path: artefactUsage(t, dir, "m-usage-1")},
		{matchID: "m-usage-2", path: artefactUsage(t, dir, "m-usage-2")},
	}
	persisterResumesUsage(context.Background(), d, rapports)

	if acquis != 1 || relaches != 1 {
		t.Fatalf("writer acquis %d fois, relache %d fois — attendu 1 et 1 (burst unique)", acquis, relaches)
	}
	for _, id := range []string{"m-usage-1", "m-usage-2"} {
		var named, unnamed int
		var rev, powerups string
		err := db.QueryRowContext(context.Background(), `
			SELECT pad_named, pad_unnamed, summary_rev, powerup_pickups_json
			FROM match_usage_films_latest WHERE match_id = ?`, id).
			Scan(&named, &unnamed, &rev, &powerups)
		if err != nil {
			t.Fatalf("films_latest %s: %v", id, err)
		}
		if named != 1 || unnamed != 0 || rev != replay.UsageSummaryRev {
			t.Errorf("%s : (named=%d, unnamed=%d, rev=%q), attendu (1, 0, %q)",
				id, named, unnamed, rev, replay.UsageSummaryRev)
		}
		if powerups != `{"powerup_camo":1}` {
			t.Errorf("%s : powerup_pickups_json = %s — l'occupation du socle de bonus doit y etre, anonyme", id, powerups)
		}
		var pads, grapples int
		var padsJSON string
		err = db.QueryRowContext(context.Background(), `
			SELECT pad_pickups, pad_pickups_json, grapple_pulls
			FROM match_usage_players_latest WHERE match_id = ? AND xuid = '111'`, id).
			Scan(&pads, &padsJSON, &grapples)
		if err != nil {
			t.Fatalf("players_latest %s: %v", id, err)
		}
		// L'assertion nominative du piege, au niveau du CABLAGE aussi : la prise du socle de
		// bonus ne descend jamais sur la ligne du joueur.
		if pads != 1 || padsJSON != `{"11223344":1}` || grapples != 1 {
			t.Errorf("%s joueur 111 : (pad_pickups=%d, json=%s, grapples=%d), attendu (1, {\"11223344\":1}, 1)",
				id, pads, padsJSON, grapples)
		}
	}
}

// TestPersisterResumesUsage_LotVideNAcquiertAucunWriter : le regime stationnaire — un cycle
// qui n'a rien cuit ne prend pas le writer partage, et ne lit meme pas les capabilities.
func TestPersisterResumesUsage_LotVideNAcquiertAucunWriter(t *testing.T) {
	acquis, relaches := 0, 0
	d := Deps{AcquireWriter: func(context.Context) (*sql.DB, func(), error) {
		acquis++
		return nil, func() { relaches++ }, nil
	}}
	persisterResumesUsage(context.Background(), d, nil)
	if acquis != 0 {
		t.Fatalf("writer acquis %d fois sur un lot vide, attendu 0", acquis)
	}
}

// TestPersisterResumesUsage_TitreSansCapability : halo_5 ne declare pas `film.usage_summary`
// dans SON capabilities.toml (absence = non supporte) — le lot entier est ecarte AVANT toute
// projection et tout writer : silence propre, pas une erreur (title-agnostic, jamais un slug).
func TestPersisterResumesUsage_TitreSansCapability(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_5", &acquis, &relaches)

	persisterResumesUsage(context.Background(), d, []artefactCuit{
		{matchID: "m-h5", path: artefactUsage(t, dir, "m-h5")},
	})
	if acquis != 0 {
		t.Fatalf("writer acquis %d fois pour un titre sans capability, attendu 0", acquis)
	}
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM match_usage_films`).Scan(&n); err != nil {
		t.Fatalf("count films: %v", err)
	}
	if n != 0 {
		t.Fatalf("un titre sans capability a ecrit %d ligne(s), attendu 0", n)
	}
}

// TestPersisterResumesUsage_ArtefactIllisibleNArretePasLeLot : l'echec d'UN match degrade ce
// match seul — le reste du lot s'ecrit, sous le meme burst.
func TestPersisterResumesUsage_ArtefactIllisibleNArretePasLeLot(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)

	persisterResumesUsage(context.Background(), d, []artefactCuit{
		{matchID: "m-absent", path: filepath.Join(dir, "inexistant.json")},
		{matchID: "m-ok", path: artefactUsage(t, dir, "m-ok")},
	})
	if acquis != 1 {
		t.Fatalf("writer acquis %d fois, attendu 1 (le lot valide s'ecrit)", acquis)
	}
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM match_usage_films_latest WHERE match_id = 'm-ok'`).Scan(&n); err != nil {
		t.Fatalf("count m-ok: %v", err)
	}
	if n != 1 {
		t.Fatalf("m-ok = %d ligne(s) dans films_latest, attendu 1", n)
	}
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM match_usage_films WHERE match_id = 'm-absent'`).Scan(&n); err != nil {
		t.Fatalf("count m-absent: %v", err)
	}
	if n != 0 {
		t.Fatalf("m-absent = %d ligne(s), attendu 0 (artefact illisible : rien d'ecrit)", n)
	}
}
