//go:build integration

// Package persist — usage_summary_persister_test.go : ce que [UsageSummaryPersister] ECRIT,
// et surtout que la lecture passe par les vues `_latest` (ADR 0026) et que leur semantique
// est LA DERNIERE PASSE PAR MATCH — jamais la derniere ligne par cle.
//
// Le schema est celui des migrations REELLES (RunForDB sur TargetShared), pas un DDL recopie —
// meme doctrine que kill_position_persister_test.go.

package persist

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis/replay"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

func openUsageSummaryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	return db
}

// resumeUsageDeTest — un resume minimal : deux joueurs, un socle d arme, un socle de bonus.
func resumeUsageDeTest() *replay.UsageSummary {
	return &replay.UsageSummary{
		Match: replay.UsageMatchSummary{
			SchemaVersion:   replay.SchemaVersion,
			FrameIntervalMS: 100,
			FrameCount:      1000,
			DurationMS:      100000,
			PadOccupancies:  4,
			PadNamed:        2,
			PadUnnamed:      1,
			PowerupPadPickups: map[string]int{
				"powerup_camo": 1,
			},
			WeaponPads: []replay.UsageWeaponPad{
				{Weapon: "11223344", Occupations: 3, Named: 2},
			},
		},
		Players: []replay.UsagePlayerSummary{
			{
				XUID: "111", GrapplePulls: 2,
				CamoEpisodes: 1, CamoMS: 6000, CamoKills: 2,
				DeployedByFamily: map[string]int{"wall": 1},
				DroppedObjects:   3, GrenadesThrown: 5,
				PadPickups:         2,
				PadPickupsByWeapon: map[string]int{"11223344": 2},
			},
			{XUID: "222", OvershieldEpisodes: 1, OvershieldMS: 1000},
		},
	}
}

// TestUsageSummaryPersistPass_EcritEtRelitParLesVues — le chemin nominal : une passe, les
// deux tables, lecture par `_latest` uniquement, JSON relus tels qu ecrits.
func TestUsageSummaryPersistPass_EcritEtRelitParLesVues(t *testing.T) {
	db := openUsageSummaryTestDB(t)
	ctx := context.Background()

	if err := NewUsageSummaryPersister(db).PersistPass(ctx, "m1", resumeUsageDeTest()); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}

	var rev string
	var schema, named, unnamed int
	var powerups, weaponPads string
	err := db.QueryRow(`SELECT summary_rev, artifact_schema, pad_named, pad_unnamed,
			powerup_pickups_json, weapon_pads_json
		FROM match_usage_films_latest WHERE match_id = 'm1'`).
		Scan(&rev, &schema, &named, &unnamed, &powerups, &weaponPads)
	if err != nil {
		t.Fatalf("select films_latest: %v", err)
	}
	if rev != replay.UsageSummaryRev || schema != replay.SchemaVersion {
		t.Errorf("(rev, schema) = (%s, %d), attendu (%s, %d)", rev, schema,
			replay.UsageSummaryRev, replay.SchemaVersion)
	}
	if named != 2 || unnamed != 1 {
		t.Errorf("pad_named/pad_unnamed = %d/%d, attendu 2/1", named, unnamed)
	}
	if powerups != `{"powerup_camo":1}` {
		t.Errorf("powerup_pickups_json = %s, attendu {\"powerup_camo\":1}", powerups)
	}
	if weaponPads != `[{"weapon":"11223344","occupations":3,"named":2}]` {
		t.Errorf("weapon_pads_json inattendu : %s", weaponPads)
	}

	var joueurs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_usage_players_latest
		WHERE match_id = 'm1'`).Scan(&joueurs); err != nil {
		t.Fatalf("select players_latest: %v", err)
	}
	if joueurs != 2 {
		t.Fatalf("match_usage_players_latest = %d lignes, attendu 2", joueurs)
	}
	var pads int
	var padsJSON, deployed string
	err = db.QueryRow(`SELECT pad_pickups, pad_pickups_json, deployed_json
		FROM match_usage_players_latest WHERE match_id = 'm1' AND xuid = '111'`).
		Scan(&pads, &padsJSON, &deployed)
	if err != nil {
		t.Fatalf("select joueur 111: %v", err)
	}
	if pads != 2 || padsJSON != `{"11223344":2}` || deployed != `{"wall":1}` {
		t.Errorf("joueur 111 relu : pad_pickups=%d pads=%s deployed=%s", pads, padsJSON, deployed)
	}
	// Les maps vides du joueur 222 sont des JSON VIDES, jamais NULL (colonnes NOT NULL).
	var deployed222 string
	if err := db.QueryRow(`SELECT deployed_json FROM match_usage_players_latest
		WHERE match_id = 'm1' AND xuid = '222'`).Scan(&deployed222); err != nil {
		t.Fatalf("select joueur 222: %v", err)
	}
	if deployed222 != "{}" {
		t.Errorf("deployed_json(222) = %q, attendu {}", deployed222)
	}
}

// TestUsageSummaryPersistPass_LaSecondePasseRemplaceEntierement — la semantique de
// generation : la vue `_latest` retient LA DERNIERE PASSE PAR MATCH. Un joueur present
// dans la passe A et absent de la passe B disparait avec la passe A — un `_latest` ligne
// par ligne le laisserait survivre, et c est exactement le melange que la vue interdit.
func TestUsageSummaryPersistPass_LaSecondePasseRemplaceEntierement(t *testing.T) {
	db := openUsageSummaryTestDB(t)
	ctx := context.Background()
	p := NewUsageSummaryPersister(db)

	if err := p.PersistPass(ctx, "m2", resumeUsageDeTest()); err != nil {
		t.Fatalf("passe A: %v", err)
	}
	passeB := resumeUsageDeTest()
	passeB.Players = passeB.Players[:1] // le joueur 222 n est plus produit
	passeB.Players[0].PadPickups = 9
	if err := p.PersistPass(ctx, "m2", passeB); err != nil {
		t.Fatalf("passe B: %v", err)
	}

	var joueurs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_usage_players_latest
		WHERE match_id = 'm2'`).Scan(&joueurs); err != nil {
		t.Fatalf("select players_latest: %v", err)
	}
	if joueurs != 1 {
		t.Fatalf("players_latest = %d lignes, attendu 1 (le joueur 222 de la passe A ne doit pas survivre)", joueurs)
	}
	var pads int
	if err := db.QueryRow(`SELECT pad_pickups FROM match_usage_players_latest
		WHERE match_id = 'm2' AND xuid = '111'`).Scan(&pads); err != nil {
		t.Fatalf("select 111: %v", err)
	}
	if pads != 9 {
		t.Errorf("pad_pickups(111) = %d, attendu 9 (la passe la PLUS RECENTE)", pads)
	}
	// La table brute, elle, garde TOUT (append-only) : 2 lignes A + 1 ligne B.
	var brut int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_usage_players WHERE match_id = 'm2'`).Scan(&brut); err != nil {
		t.Fatalf("select table brute: %v", err)
	}
	if brut != 3 {
		t.Errorf("table brute = %d lignes, attendu 3 (aucun DELETE, jamais)", brut)
	}
	var films int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_usage_films_latest WHERE match_id = 'm2'`).Scan(&films); err != nil {
		t.Fatalf("select films_latest: %v", err)
	}
	if films != 1 {
		t.Errorf("films_latest = %d lignes, attendu 1", films)
	}
}

// TestUsageSummaryPersistPass_SansLigneJoueur — un match ou personne n a rien fait
// d attribuable reste un match MESURE : la ligne films s ecrit, zero ligne joueur.
func TestUsageSummaryPersistPass_SansLigneJoueur(t *testing.T) {
	db := openUsageSummaryTestDB(t)
	s := &replay.UsageSummary{Match: replay.UsageMatchSummary{
		SchemaVersion: replay.SchemaVersion, FrameIntervalMS: 100, FrameCount: 500,
	}}
	if err := NewUsageSummaryPersister(db).PersistPass(context.Background(), "m3", s); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	var films, joueurs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_usage_films_latest WHERE match_id = 'm3'`).Scan(&films); err != nil {
		t.Fatalf("select films: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_usage_players WHERE match_id = 'm3'`).Scan(&joueurs); err != nil {
		t.Fatalf("select players: %v", err)
	}
	if films != 1 || joueurs != 0 {
		t.Errorf("films/joueurs = %d/%d, attendu 1/0", films, joueurs)
	}
}

// TestUsageSummaryPersistPass_PasseVideDeJoueursSupplante — LE CAS DEGENERE attrape par la
// revue adversariale du 2026-09-04 (deux relecteurs, independamment) : une passe B LEGALE
// sans aucune ligne joueur (artefact re-cuit appauvri, roster perdu) doit supplanter les
// joueurs de la passe A. L autorite de passe est la ligne `match_usage_films` — une vue
// joueurs partitionnee sur sa propre table servirait ici les joueurs perimes de A pendant
// que films_latest sert B : le melange de passes exact que les vues interdisent.
func TestUsageSummaryPersistPass_PasseVideDeJoueursSupplante(t *testing.T) {
	db := openUsageSummaryTestDB(t)
	ctx := context.Background()
	p := NewUsageSummaryPersister(db)

	if err := p.PersistPass(ctx, "m5", resumeUsageDeTest()); err != nil {
		t.Fatalf("passe A: %v", err)
	}
	passeB := resumeUsageDeTest()
	passeB.Players = nil // re-projection legale qui n attribue plus rien a personne
	passeB.Match.PadNamed = 0
	if err := p.PersistPass(ctx, "m5", passeB); err != nil {
		t.Fatalf("passe B: %v", err)
	}

	var joueurs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_usage_players_latest
		WHERE match_id = 'm5'`).Scan(&joueurs); err != nil {
		t.Fatalf("select players_latest: %v", err)
	}
	if joueurs != 0 {
		t.Fatalf("players_latest = %d lignes, attendu 0 : les joueurs de la passe A survivent "+
			"a une passe B vide — la passe courante doit se decider sur match_usage_films", joueurs)
	}
	var named int
	if err := db.QueryRow(`SELECT pad_named FROM match_usage_films_latest
		WHERE match_id = 'm5'`).Scan(&named); err != nil {
		t.Fatalf("select films_latest: %v", err)
	}
	if named != 0 {
		t.Errorf("films_latest.pad_named = %d, attendu 0 (la passe B)", named)
	}
	// La table brute garde tout : 2 lignes joueur de A, 0 de B.
	var brut int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_usage_players WHERE match_id = 'm5'`).Scan(&brut); err != nil {
		t.Fatalf("select table brute: %v", err)
	}
	if brut != 2 {
		t.Errorf("table brute = %d lignes, attendu 2 (append-only)", brut)
	}
}

// TestUsageSummaryPersistPass_Refus — matchID vide, resume nil, xuid vide, xuid double :
// quatre refus AVANT la transaction, qui ne laissent aucune ligne derriere eux.
func TestUsageSummaryPersistPass_Refus(t *testing.T) {
	db := openUsageSummaryTestDB(t)
	ctx := context.Background()
	p := NewUsageSummaryPersister(db)

	if err := p.PersistPass(ctx, "", resumeUsageDeTest()); err == nil {
		t.Error("matchID vide : attendu un refus")
	}
	if err := p.PersistPass(ctx, "m4", nil); err == nil {
		t.Error("resume nil : attendu un refus")
	}
	sansXUID := resumeUsageDeTest()
	sansXUID.Players[0].XUID = ""
	if err := p.PersistPass(ctx, "m4", sansXUID); err == nil {
		t.Error("xuid vide : attendu un refus")
	}
	double := resumeUsageDeTest()
	double.Players[1].XUID = "111"
	if err := p.PersistPass(ctx, "m4", double); err == nil {
		t.Error("xuid double : attendu un refus")
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_usage_players`).Scan(&n); err != nil {
		t.Fatalf("select table: %v", err)
	}
	if n != 0 {
		t.Errorf("les refus doivent laisser la table intacte (0 ligne), lu %d", n)
	}
}
