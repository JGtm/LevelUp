//go:build cgo

package skill

// skill_v2_parity_test.go — preuve de parité du lot perf L6 (2026-09-23).
//
// Le lot décide de ce qui est nouveau AVANT l'écrivain (filigrane et éligibilité
// lus sur le lecteur, skill_v2_watermark.go) et remplace les rafales par lots de 3
// par UNE rafale par joueur et par cycle. Il ne doit rien changer au calcul LUSR v2
// (ADR 0024, chemin de RecomputeLUSRCanonicalForPlayer) : mêmes lignes écrites,
// dans le même ordre, avec les mêmes valeurs. Le test joue l'orchestration d'avant
// (référence figée ci-dessous) et celle d'aujourd'hui sur deux bases jumelles, sur
// un jeu dont une partie est déjà traitée, et compare TOUTES les lignes écrites
// (player_skill_state_v2 côté shared, match_skill_rank côté joueur), identifiants
// compris. Seules les horloges d'écriture (written_at, created_at, updated_at) sont
// hors comparaison.

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb"
)

// parityFailingMatch : le match dont l'écriture canonical échoue (CHECK de la base
// joueur) — son groupe est tenu, le match suivant du groupe est sauté.
const parityFailingMatch = "n_btb_fail"

// parityWorld : une base shared et une base joueur (Stratégie C), jumelles.
type parityWorld struct {
	shared *sql.DB
	player *sql.DB
}

func newParityWorld(t *testing.T) parityWorld {
	t.Helper()
	w := parityWorld{
		shared: openShadowTestDB(t),
		player: openCanonicalPlayerTestDBWithCheck(t, "match_id <> '"+parityFailingMatch+"'"),
	}
	// Offsets d'escouade et hyperparamètre ré-estimé : les chemins de calcul qui
	// lisent la base pendant la rafale sont exercés, à l'identique des deux côtés.
	for _, q := range []string{
		`INSERT INTO player_squad_offset (xuid, partner_xuid, playlist_group, offset_value, match_count, source)
		 VALUES ('owner', 'mate', 'arena_slayer', 1.5, 20, 'test'), ('mate', 'owner', 'arena_slayer', 1.5, 20, 'test')`,
		`INSERT INTO lusr_hyperparams_v2 (playlist_group, name, value, source)
		 VALUES ('arena_objectif', 'draw_probability_empirical', 0.15, 'test')`,
	} {
		if _, err := w.shared.Exec(q); err != nil {
			t.Fatalf("seed parité: %v", err)
		}
	}
	return w
}

// parityHistory : la partie DÉJÀ TRAITÉE (4 matchs notables sur 3 groupes, 1 sans
// chaîne). Filigranes après traitement : arena_slayer 240, arena_objectif 120,
// btb 180.
func parityHistory() []shadowFixture {
	return []shadowFixture{
		{"h_slayer_1", pairSlayer, fixtureAt(60), seats2v2(2, 18, 6)},
		{"h_ctf_1", pairCTF, fixtureAt(120), seats2v2(3, 8, 11)},
		{"h_btb_1", pairBTB, fixtureAt(180), seats2v2(2, 15, 9)},
		{"h_slayer_2", pairSlayer, fixtureAt(240), seats2v2(1, 10, 10)},
		{"h_nochain_1", pairNoChain, fixtureAt(300), seats2v2(2, 20, 2)},
	}
}

// parityArrivals : le cycle suivant — tous les cas que l'orchestration traite.
func parityArrivals() []shadowFixture {
	return []shadowFixture{
		{"n_slayer_late", pairSlayer, fixtureAt(210), seats2v2(2, 16, 7)},       // SOUS le filigrane : arrivé hors ordre, déjà vu
		{"n_slayer_three_teams", pairSlayer, fixtureAt(360), seatsThreeTeams()}, // au-dessus, non notable
		{"n_slayer_new", pairSlayer, fixtureAt(420), seats2v2(2, 22, 5)},        // au-dessus, notable → écrit
		{"n_ctf_imbalance", pairCTF, fixtureAt(480), seatsThreeVsOne()},         // au-dessus, non notable
		{parityFailingMatch, pairBTB, fixtureAt(540), seats2v2(3, 9, 14)},       // notable, écriture en échec → groupe tenu
		{"n_nochain", pairNoChain, fixtureAt(600), seats2v2(2, 13, 8)},          // sans chaîne
		{"n_ctf_new", pairCTF, fixtureAt(660), seats2v2(2, 17, 6)},              // au-dessus, notable → écrit
		{"n_slayer_quit", pairSlayer, fixtureAt(720), seatsOwnerQuit()},         // au-dessus, issue non notable
		{"n_fiesta_new", pairFiesta, fixtureAt(780), seats2v2(2, 19, 4)},        // groupe neuf, sans filigrane → écrit
		{"n_btb_after_fail", pairBTB, fixtureAt(840), seats2v2(2, 12, 8)},       // groupe tenu → sauté
	}
}

func TestLUSRV2Shadow_ParityWithLegacyOrchestration(t *testing.T) {
	t.Setenv(lusrV2EnvFlag, "1")
	t.Setenv(lusrCanonicalEnvFlag, "LUSR_V2") // écrit aussi match_skill_rank (Stratégie C)
	t.Setenv(lusrModeCouplingEnvFlag, "1")    // fuite inter-modes : écrit d'autres groupes
	t.Setenv(lusrSquadOffsetEnvFlag, "1")     // offsets d'escouade lus sous la rafale

	legacy, current := newParityWorld(t), newParityWorld(t)

	// Phase 1 — la partie déjà traitée, par l'orchestration d'avant, sur les deux.
	for _, w := range []parityWorld{legacy, current} {
		seedShadowFixtures(t, w.shared, parityHistory()...)
		if n := runLegacyOrchestration(t, w, "owner"); n != 4 {
			t.Fatalf("phase 1 : processed = %d, want 4", n)
		}
	}
	requireSameWrites(t, legacy, current, "phase 1 (jumelles)")
	statesBefore := countStateRows(t, current.shared)

	// Phase 2 — les arrivées, sur l'historique déjà traité : avant contre après.
	for _, w := range []parityWorld{legacy, current} {
		seedShadowFixtures(t, w.shared, parityArrivals()...)
	}
	nLegacy := runLegacyOrchestration(t, legacy, "owner")
	acc := &orderTrackingAccess{db: current.shared}
	nCurrent, err := RunLUSRV2ShadowOwnerOnly(context.Background(), current.player, acc, "owner")
	if err != nil {
		t.Fatalf("phase 2 : RunLUSRV2ShadowOwnerOnly: %v", err)
	}
	if nLegacy != 3 || nCurrent != 3 {
		t.Errorf("phase 2 : processed avant=%d après=%d, want 3 et 3", nLegacy, nCurrent)
	}
	if acc.writeCalls != 1 {
		t.Errorf("phase 2 : %d rafales d'écrivain, want 1 (une par joueur et par cycle)", acc.writeCalls)
	}
	if acc.violation != "" {
		t.Errorf("garde anti-deadlock violée : %s", acc.violation)
	}
	requireSameWrites(t, legacy, current, "phase 2")
	if after := countStateRows(t, current.shared); after <= statesBefore {
		t.Fatalf("phase 2 n'a rien écrit (%d → %d lignes) : la comparaison ne prouverait rien", statesBefore, after)
	}
}

// runLegacyOrchestration rejoue À L'IDENTIQUE l'orchestration d'avant le lot perf
// L6 (base 97cc0d0c8 : runLUSRV2Shadow, skill_v2_shadow.go:126-187, et
// processShadowChunk, skill_v2_shared_access.go:79-93) : TOUS les candidats SQL,
// aucun pré-filtre, par lots de 3 (une rafale chacun), le filigrane testé DANS la
// rafale par processOneShadowMatch, heldGroups partagé d'un lot à l'autre. Sur une
// base unique, les lots ne changent que la rafale, pas les écritures. Seul le corps
// per-match (processOneShadowMatch, que le lot ne touche pas) est commun avec le
// code courant : c'est l'orchestration qu'on compare.
func runLegacyOrchestration(t *testing.T, w parityWorld, xuid string) int {
	t.Helper()
	ctx := context.Background()
	matches, err := loadShadowMatches(ctx, w.shared, xuid)
	if err != nil {
		t.Fatalf("référence : loadShadowMatches: %v", err)
	}
	base := newShadowRunContext(w.player, xuid, IsLUSRV2Canonical(), true)
	var s shadowRunStats
	heldGroups := make(map[string]bool)
	const legacyBurstChunk = 3 // l'ancien postsyncLUSRBurstChunk
	for start := 0; start < len(matches); start += legacyBurstChunk {
		end := min(start+legacyBurstChunk, len(matches))
		c := base
		c.sharedDB = w.shared
		c.repo = duckdb.NewSkillV2Repo(w.shared)
		if c.squadEnabled {
			c.squadRepo = duckdb.NewSquadOffsetRepo(w.shared)
		}
		for _, m := range matches[start:end] {
			processOneShadowMatch(ctx, c, m, &s, heldGroups)
		}
	}
	return s.processed
}

// requireSameWrites exige des écritures identiques (lignes, ordre, valeurs) dans
// les deux mondes.
func requireSameWrites(t *testing.T, legacy, current parityWorld, stage string) {
	t.Helper()
	if a, b := dumpParityStates(t, legacy.shared), dumpParityStates(t, current.shared); !reflect.DeepEqual(a, b) {
		t.Fatalf("%s : player_skill_state_v2 diverge\navant = %+v\naprès = %+v", stage, a, b)
	}
	if a, b := dumpParityRanks(t, legacy.player), dumpParityRanks(t, current.player); !reflect.DeepEqual(a, b) {
		t.Fatalf("%s : match_skill_rank diverge\navant = %+v\naprès = %+v", stage, a, b)
	}
}

// parityStateRow : une ligne player_skill_state_v2, sans son horloge d'écriture.
type parityStateRow struct {
	id                       int64
	xuid, group              string
	mu, sigma                float64
	experience               int64
	lastMatchID, lastMatchAt string // "" si NULL ; horodatage en RFC 3339 UTC
}

func dumpParityStates(t *testing.T, db *sql.DB) []parityStateRow {
	t.Helper()
	rows, err := db.Query(`SELECT id, xuid, playlist_group, mu, sigma, experience, last_match_id, last_match_at
		FROM player_skill_state_v2 ORDER BY id`)
	if err != nil {
		t.Fatalf("dump player_skill_state_v2: %v", err)
	}
	defer rows.Close() //nolint:errcheck
	var out []parityStateRow
	for rows.Next() {
		var r parityStateRow
		var lastID sql.NullString
		var lastAt sql.NullTime
		if err := rows.Scan(&r.id, &r.xuid, &r.group, &r.mu, &r.sigma, &r.experience, &lastID, &lastAt); err != nil {
			t.Fatalf("scan player_skill_state_v2: %v", err)
		}
		r.lastMatchID, r.lastMatchAt = lastID.String, formatNullTime(lastAt)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows player_skill_state_v2: %v", err)
	}
	return out
}

// parityRankRow : une ligne match_skill_rank, sans ses horloges d'écriture.
type parityRankRow struct {
	id                                     int64
	matchID, ratingType                    string
	ratingValue, ratingDeviation, delta    sql.NullFloat64
	expectedWinProb                        sql.NullFloat64
	tier, tierFR, tierLabel, playlistGroup sql.NullString
	subTier                                sql.NullInt64
	startTime                              string
}

func dumpParityRanks(t *testing.T, db *sql.DB) []parityRankRow {
	t.Helper()
	rows, err := db.Query(`SELECT id, match_id, rating_type, rating_value, rating_deviation, rating_delta,
		expected_win_prob, tier, tier_fr, tier_label, playlist_group, sub_tier, start_time
		FROM match_skill_rank ORDER BY id`)
	if err != nil {
		t.Fatalf("dump match_skill_rank: %v", err)
	}
	defer rows.Close() //nolint:errcheck
	var out []parityRankRow
	for rows.Next() {
		var r parityRankRow
		var start sql.NullTime
		if err := rows.Scan(&r.id, &r.matchID, &r.ratingType, &r.ratingValue, &r.ratingDeviation, &r.delta,
			&r.expectedWinProb, &r.tier, &r.tierFR, &r.tierLabel, &r.playlistGroup, &r.subTier, &start); err != nil {
			t.Fatalf("scan match_skill_rank: %v", err)
		}
		r.startTime = formatNullTime(start)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows match_skill_rank: %v", err)
	}
	return out
}

func formatNullTime(v sql.NullTime) string {
	if !v.Valid {
		return ""
	}
	return v.Time.UTC().Format(time.RFC3339Nano)
}
