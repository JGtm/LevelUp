//go:build integration

package main

// cmd_backfill_killsource_selection_placement_integration_test.go — LA FRAICHEUR DU PLACEMENT DES
// VIES DANS LA SELECTION DU RATTRAPAGE (plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`, decision V12).
//
// `matchsAJour` exige, pour tout match qui a des vies, une passe de `match_life_placement_latest`
// a `killcollector.PlacementRev`. La requete tourne ici sur une vraie base migree : les quatre
// cas ci-dessous ne peuvent pas mentir sur la syntaxe ni sur l'ensemble selectionne. Le cinquieme
// (lot V2b) : la convergence sur un match au pont slot->xuid non publiable.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/film/types"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/killcollector"
)

// baseDeSelection ouvre un shared migre par les VRAIES migrations.
func baseDeSelection(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "selection.duckdb"))
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

// executer : un ordre de fixture, qui echoue le test s'il echoue.
func executer(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("%s : %v", q, err)
	}
}

// matchAJourDeSesVies inscrit un match dont le journal ET les faits d'isolement portent leurs
// revisions courantes : positions presentes, equipes en base, vies a IsolationDecoderRev.
func matchAJourDeSesVies(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	executer(t, db, `INSERT INTO match_kill_events (match_id, decode_pass, decoder_rev, time_ms,
		victim_gamertag, feed_present, assist_known, publishable, read_path, read_origin)
		VALUES (?, 'k1', ?, 1000, 'V', TRUE, TRUE, TRUE, ?, 'concordant-test')`,
		id, decfilm.Rev, killscope.ReadPathFilmWalk)
	executer(t, db, `INSERT INTO kill_positions (match_id, decode_pass, killer_xuid, time_ms)
		VALUES (?, 'p1', '111', 1000)`, id)
	executer(t, db, `INSERT INTO match_participants (match_id, xuid, team_id) VALUES (?, '111', 0)`, id)
	executer(t, db, `INSERT INTO match_lives (match_id, decode_pass, decoder_rev, xuid, start_ms,
		end_ms, end_cause, named_by) VALUES (?, 'v1', ?, '111', 0, 10000, 'death', 'death')`,
		id, killcollector.IsolationDecoderRev)
}

// placementA ecrit une passe de placement d'une ligne, a la revision `rev`.
func placementA(t *testing.T, db *sql.DB, id, rev string) {
	t.Helper()
	executer(t, db, `INSERT INTO match_life_placement (match_id, decode_pass, decoder_rev, xuid,
		start_ms, end_ms, duration_ms, measured_ms, carrier_ms, team_down_ms, unplaced_ms,
		teammate_unplaced_ms, kills) VALUES (?, 'pl1', ?, '111', 0, 10000, 10000, 10100, 0, 0, 0, 0, 0)`,
		id, rev)
}

// TestMatchsAJour_ExigeLePlacementDesVies — decision V12.
//
//	sans-placement     des vies, aucun placement           -> a redecoder
//	placement-perime   des vies, placement a une autre rev -> a redecoder
//	placement-courant  des vies, placement a PlacementRev  -> a jour
//	sans-vies          journal a jour, ni positions ni vies -> a jour (le placement ne peut naitre)
func TestMatchsAJour_ExigeLePlacementDesVies(t *testing.T) {
	db := baseDeSelection(t)
	for _, id := range []string{"sans-placement", "placement-perime", "placement-courant"} {
		matchAJourDeSesVies(t, db, id)
	}
	placementA(t, db, "placement-perime", "placement-2020-01-01")
	placementA(t, db, "placement-courant", killcollector.PlacementRev)
	executer(t, db, `INSERT INTO match_kill_events (match_id, decode_pass, decoder_rev, time_ms,
		victim_gamertag, feed_present, assist_known, publishable, read_path, read_origin)
		VALUES ('sans-vies', 'k1', ?, 1000, 'V', TRUE, TRUE, TRUE, ?, 'concordant-test')`,
		decfilm.Rev, killscope.ReadPathFilmWalk)

	aJour, err := matchsAJour(context.Background(), db)
	if err != nil {
		t.Fatalf("matchsAJour: %v", err)
	}
	attendu := map[string]bool{"placement-courant": true, "sans-vies": true}
	for _, id := range []string{"sans-placement", "placement-perime", "placement-courant", "sans-vies"} {
		if aJour[id] != attendu[id] {
			t.Errorf("%s : a jour = %v, attendu %v", id, aJour[id], attendu[id])
		}
	}
}

// TestMatchsAJour_PontNonPubliable_Converge — lot V2b.2 (2026-09-29). Un match dont le pont
// slot->xuid n'est pas publiable a des vies ; avant le lot, le placement n'ecrivait AUCUNE ligne
// et `matchsAJour` le re-selectionnait a chaque passe. La chaine de production est jouee : le
// calcul pur (`replay.PlacementDesVies`) sur un registre au pont refuse, puis le persister de
// production (qui valide les lignes) a `killcollector.PlacementRev`. Le match est candidat avant,
// a jour apres — sans regle de plus dans `matchsAJour`.
func TestMatchsAJour_PontNonPubliable_Converge(t *testing.T) {
	db := baseDeSelection(t)
	const id = "pont-refuse"
	matchAJourDeSesVies(t, db, id)
	ctx := context.Background()
	if aJour, err := matchsAJour(ctx, db); err != nil || aJour[id] {
		t.Fatalf("avant le placement : a jour = %v (%v), attendu candidat", aJour[id], err)
	}

	pos := make([]decfilm.BipedPosition, 0, 101)
	for ms := int64(0); ms <= 10_000; ms += 100 {
		pos = append(pos, decfilm.BipedPosition{Slot: 1, TimestampUS: uint64(ms) * 1000, HasWorld: true})
	}
	reg := replay.BuildIdentityRegistry(ctx, replay.IdentityInput{
		Positions: pos, Deaths: []types.Death{{XUID: 111, TimeMS: 10_000}},
		PlayerIndices: types.PlayerIndexTable{ByXUID: map[uint64]int{111: 0}, Readings: 26, Disagreements: 1},
	})
	if reg.PontPubliable() {
		t.Fatal("fixture : le pont devait etre refuse")
	}
	vies, bilan := replay.PlacementDesVies(replay.EntreePlacement{
		Positions: pos, Registre: reg, Equipes: map[uint64]int{111: 0},
	})
	if !bilan.PontNonPubliable || len(vies) == 0 {
		t.Fatalf("calcul pur : %d vies, pont non publiable %v — attendu des vies et le refus dit",
			len(vies), bilan.PontNonPubliable)
	}
	rows := make([]persist.LifePlacementInsert, 0, len(vies))
	for _, v := range vies {
		rows = append(rows, persist.LifePlacementInsert{
			XUID: "111", StartMS: v.DebutMS, EndMS: v.FinMS, DurationMS: v.DureeMS,
			MeasuredMS: v.MesureMS, MedianM: v.MedianeM, UnplacedMS: v.NonSitueMS,
			CarrierMS: v.PorteurMS, TeamDownMS: v.EquipeATerreMS,
			TeammateUnplacedMS: v.CoequipierNonSitueMS, Kills: v.Frags,
		})
	}
	if err := persist.NewLifePlacementPersister(db).PersistPass(ctx, persist.LifePlacementBatch{
		MatchID: id, DecoderRev: killcollector.PlacementRev, Rows: rows,
	}); err != nil {
		t.Fatalf("le persister refuse les lignes d'un pont non publiable : %v", err)
	}

	aJour, err := matchsAJour(ctx, db)
	if err != nil {
		t.Fatalf("matchsAJour: %v", err)
	}
	if !aJour[id] {
		t.Fatalf("%s reste candidat apres son placement : le rattrapage ne converge pas", id)
	}
}

// TestMatchsAJour_PlacementPosterieurAuxVies — revue adversariale V5.1, R2 (2026-09-30). Un
// placement a `PlacementRev` calcule sur d'anciennes vies ne rend pas le match a jour : les vies
// reecrites PLUS TARD (revision d'isolement montee, puis ecriture du placement en echec) le
// remettent candidat, et il ne redevient a jour que quand le placement est reecrit apres elles.
// Les horodatages sont poses explicitement : le critere est `written_at`, pas l'ordre des ordres.
func TestMatchsAJour_PlacementPosterieurAuxVies(t *testing.T) {
	db := baseDeSelection(t)
	ctx := context.Background()
	const id = "vies-reecrites"
	matchAJourDeSesVies(t, db, id)
	executer(t, db, `UPDATE match_lives SET written_at = TIMESTAMP '2026-09-20 10:00:00' WHERE match_id = ?`, id)
	executer(t, db, `INSERT INTO match_life_placement (match_id, decode_pass, decoder_rev, written_at,
		xuid, start_ms, end_ms, duration_ms, measured_ms, carrier_ms, team_down_ms, unplaced_ms,
		teammate_unplaced_ms, kills) VALUES (?, 'pl1', ?, TIMESTAMP '2026-09-20 10:00:00', '111', 0,
		10000, 10000, 10100, 0, 0, 0, 0, 0)`, id, killcollector.PlacementRev)
	verifier := func(etape string, attendu bool) {
		t.Helper()
		aJour, err := matchsAJour(ctx, db)
		if err != nil {
			t.Fatalf("%s : matchsAJour: %v", etape, err)
		}
		if aJour[id] != attendu {
			t.Fatalf("%s : a jour = %v, attendu %v", etape, aJour[id], attendu)
		}
	}
	verifier("placement ecrit avec les vies (meme instant)", true)

	// Les vies sont reecrites plus tard (nouvelle passe) ; le placement, lui, n'a pas suivi.
	executer(t, db, `INSERT INTO match_lives (match_id, decode_pass, decoder_rev, written_at, xuid,
		start_ms, end_ms, end_cause, named_by)
		VALUES (?, 'v2', ?, TIMESTAMP '2026-09-25 10:00:00', '111', 0, 10000, 'death', 'death')`,
		id, killcollector.IsolationDecoderRev)
	verifier("vies reecrites, placement d'avant", false)

	// Le placement est reecrit apres les vies : le match converge.
	executer(t, db, `INSERT INTO match_life_placement (match_id, decode_pass, decoder_rev, written_at,
		xuid, start_ms, end_ms, duration_ms, measured_ms, carrier_ms, team_down_ms, unplaced_ms,
		teammate_unplaced_ms, kills) VALUES (?, 'pl2', ?, TIMESTAMP '2026-09-25 10:00:01', '111', 0,
		10000, 10000, 10100, 0, 0, 0, 0, 0)`, id, killcollector.PlacementRev)
	verifier("placement reecrit apres les vies", true)
}
