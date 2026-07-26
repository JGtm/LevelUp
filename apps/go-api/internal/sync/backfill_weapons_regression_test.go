//go:build integration

package sync

// backfill_weapons_regression_test.go — non-régression et tests unitaires des
// helpers DB du pipeline weapon kills.
//
// Chaque suite est annotée du bug ou comportement qu'elle couvre.

import (
	"testing"

	"levelup/go-api/internal/analysis"
)

// ─────────────────────────────────────────────────────────────────────────────
// getXuidToPI — ordre par colonne `rank` (pas `rank_in_team`)
// Bug corrigé 2026-05-09 : la requête ORDER BY rank_in_team levait une erreur
// DuckDB car la colonne s'appelle `rank` dans match_participants.
// ─────────────────────────────────────────────────────────────────────────────

// TestGetXuidToPI_OrderByRank vérifie que le tri est bien effectué sur `rank`,
// donnant un player_index stable team_id ASC, rank ASC.
func TestGetXuidToPI_OrderByRank(t *testing.T) {
	db := openWeaponDB(t)
	// Équipe 0 : xuid2 (rank=1) doit obtenir PI=0, xuid1 (rank=2) PI=1.
	// Équipe 1 : xuid3 (rank=1) obtient PI=2.
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid2', 0, 1)`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 2)`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid3', 1, 1)`)

	result, err := getXuidToPI(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}
	if result["xuid2"] != 0 {
		t.Errorf("xuid2 (rank=1) must be PI=0, got %d", result["xuid2"])
	}
	if result["xuid1"] != 1 {
		t.Errorf("xuid1 (rank=2) must be PI=1, got %d", result["xuid1"])
	}
	if result["xuid3"] != 2 {
		t.Errorf("xuid3 (team=1, rank=1) must be PI=2, got %d", result["xuid3"])
	}
}

// TestGetXuidToPI_NullRankLast vérifie que les participants sans rank (NULL)
// viennent en dernier (NULLS LAST dans la requête).
func TestGetXuidToPI_NullRankLast(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_participants(match_id, xuid, team_id, rank) VALUES ('m1', 'xuid1', 0, 1)`)
	db.Exec(`INSERT INTO match_participants(match_id, xuid, team_id, rank) VALUES ('m1', 'xuidNull', 0, NULL)`)

	result, err := getXuidToPI(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if result["xuid1"] != 0 {
		t.Errorf("xuid1 (rank=1) must be PI=0, got %d", result["xuid1"])
	}
	if result["xuidNull"] != 1 {
		t.Errorf("xuidNull (rank=NULL) must be PI=1 via NULLS LAST, got %d", result["xuidNull"])
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// getAllKillsForMatch / getKillsForPlayer — déduplication GROUP BY
// Bug corrigé 2026-05-09 : sans GROUP BY xuid, time_ms, chaque ligne dupliquée
// dans highlight_events produisait un kill supplémentaire, doublant weapon_kills.
// ─────────────────────────────────────────────────────────────────────────────

// TestGetAllKillsForMatch_DeduplicatesDuplicateRows simule le bug observé en
// production : deux lignes identiques (même xuid+time_ms) dans highlight_events.
func TestGetAllKillsForMatch_DeduplicatesDuplicateRows(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`) // doublon
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 10000)`)

	kills, err := getAllKillsForMatch(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 2 {
		t.Fatalf("expected 2 deduplicated kills, got %d (regression: GROUP BY missing)", len(kills))
	}
}

// TestGetKillsForPlayer_DeduplicatesDuplicateRows — même régression côté
// pipeline single-player (weaponKillsOneShot = COLLECT+FLUSH).
func TestGetKillsForPlayer_DeduplicatesDuplicateRows(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`) // doublon
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 10000)`)

	kills, err := getKillsForPlayer(t.Context(), db, "m1", "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 2 {
		t.Fatalf("expected 2 deduplicated kills, got %d (regression: GROUP BY missing)", len(kills))
	}
}

// TestGetAllKillsForMatch_MeleeTypeAggregated vérifie que MAX(CASE WHEN LIKE '%melee%')
// fusionne 'kill' + 'melee_kill' pour un même (xuid, time_ms) en un seul kill is_melee=true.
func TestGetAllKillsForMatch_MeleeTypeAggregated(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'melee_kill', 5000)`)

	kills, err := getAllKillsForMatch(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 1 {
		t.Fatalf("expected 1 kill (kill+melee_kill at same ts collapsed), got %d", len(kills))
	}
	if !kills[0].IsMelee {
		t.Error("MAX aggregation must yield is_melee=true")
	}
}

// TestGetKillsForPlayer_GrenadeTypeAggregated — même logique pour grenade.
func TestGetKillsForPlayer_GrenadeTypeAggregated(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'grenade_kill', 5000)`)

	kills, err := getKillsForPlayer(t.Context(), db, "m1", "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 1 {
		t.Fatalf("expected 1 kill (kill+grenade_kill collapsed), got %d", len(kills))
	}
	if !kills[0].IsGrenade {
		t.Error("MAX aggregation must yield is_grenade=true")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// getAllKillsForMatch — comportements attendus
// ─────────────────────────────────────────────────────────────────────────────

func TestGetAllKillsForMatch_MultipleParticipants(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 10000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid2', 'kill', 7000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid2', 'melee_kill', 9000)`)

	kills, err := getAllKillsForMatch(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 4 {
		t.Fatalf("expected 4 kills (2 per player), got %d", len(kills))
	}
	counts := map[string]int{}
	for _, k := range kills {
		counts[k.XUID]++
	}
	if counts["xuid1"] != 2 {
		t.Errorf("expected 2 kills for xuid1, got %d", counts["xuid1"])
	}
	if counts["xuid2"] != 2 {
		t.Errorf("expected 2 kills for xuid2, got %d", counts["xuid2"])
	}
}

func TestGetAllKillsForMatch_ExcludesNonKillEvents(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'death', 6000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'assist', 7000)`)

	kills, err := getAllKillsForMatch(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 1 {
		t.Fatalf("expected 1 kill (death+assist excluded), got %d", len(kills))
	}
}

func TestGetAllKillsForMatch_XUIDAndMatchIDSet(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid_alpha', 'kill', 3000)`)

	kills, err := getAllKillsForMatch(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 1 {
		t.Fatal("expected 1 kill")
	}
	if kills[0].XUID != "xuid_alpha" {
		t.Errorf("expected XUID xuid_alpha, got %s", kills[0].XUID)
	}
	if kills[0].MatchID != "m1" {
		t.Errorf("expected MatchID m1, got %s", kills[0].MatchID)
	}
}

func TestGetAllKillsForMatch_OtherMatchExcluded(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m2', 'xuid1', 'kill', 5000)`) // autre match

	kills, err := getAllKillsForMatch(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 1 {
		t.Fatalf("expected 1 kill (m2 excluded), got %d", len(kills))
	}
}

func TestGetAllKillsForMatch_NullTimeMS(t *testing.T) {
	db := openWeaponDB(t)
	// NULL time_ms doit être traité comme 0 (pas de panic).
	db.Exec(`INSERT INTO highlight_events(match_id, xuid, event_type, time_ms) VALUES ('m1', 'xuid1', 'kill', NULL)`)

	kills, err := getAllKillsForMatch(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 1 {
		t.Fatalf("expected 1 kill with NULL time_ms, got %d", len(kills))
	}
	if kills[0].TimeMS != 0 {
		t.Errorf("NULL time_ms should map to 0, got %d", kills[0].TimeMS)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// getMatchParticipantXuids
// ─────────────────────────────────────────────────────────────────────────────

func TestGetMatchParticipantXuids_ReturnsDistinct(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 1)`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid2', 0, 2)`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid3', 1, 1)`)

	xuids, err := getMatchParticipantXuids(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(xuids) != 3 {
		t.Fatalf("expected 3 xuids, got %d", len(xuids))
	}
}

func TestGetMatchParticipantXuids_OtherMatchExcluded(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 1)`)
	db.Exec(`INSERT INTO match_participants VALUES ('m2', 'xuid2', 0, 1)`)

	xuids, err := getMatchParticipantXuids(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(xuids) != 1 || xuids[0] != "xuid1" {
		t.Fatalf("expected only xuid1 for m1, got %v", xuids)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// attributionsToRows — filtrage xuid et copie des champs
// ─────────────────────────────────────────────────────────────────────────────

func TestAttributionsToRows_FiltersXUID(t *testing.T) {
	attrs := []analysis.KillAttribution{
		{XUID: "xuid1", TimeMS: 5000},
		{XUID: "xuid2", TimeMS: 6000},
		{XUID: "xuid1", TimeMS: 7000},
	}
	rows := attributionsToRows(attrs, "xuid1")
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows for xuid1, got %d", len(rows))
	}
	if rows[0].TimeMS != 5000 || rows[1].TimeMS != 7000 {
		t.Errorf("unexpected TimeMS: %d, %d", rows[0].TimeMS, rows[1].TimeMS)
	}
}

func TestAttributionsToRows_UnknownXUID(t *testing.T) {
	attrs := []analysis.KillAttribution{{XUID: "xuid1", TimeMS: 5000}}
	rows := attributionsToRows(attrs, "unknownXUID")
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows for unknown xuid, got %d", len(rows))
	}
}

// TestAttributionsToRows_AllFieldsCopied vérifie que tous les champs de
// KillAttribution (y compris les pointeurs) sont copiés sans perte vers WeaponKillRow.
func TestAttributionsToRows_AllFieldsCopied(t *testing.T) {
	delta := 300
	pi := 2
	chunk := 1
	reconc := uint64(9999)
	wid := uint64(12345)
	attrs := []analysis.KillAttribution{{
		XUID:            "xuid1",
		TimeMS:          8000,
		WeaponID:        &wid,
		ReconciledAs:    &reconc,
		DeltaMS:         &delta,
		Confidence:      "high",
		AttributionPath: "fire_event",
		SwapDetected:    true,
		DelayedDamage:   true,
		PlayerIndex:     &pi,
		SourceChunkIdx:  &chunk,
	}}
	rows := attributionsToRows(attrs, "xuid1")
	if len(rows) != 1 {
		t.Fatal("expected 1 row")
	}
	r := rows[0]
	if r.TimeMS != 8000 {
		t.Errorf("TimeMS: want 8000, got %d", r.TimeMS)
	}
	if r.WeaponID == nil || *r.WeaponID != 12345 {
		t.Error("WeaponID: want 12345")
	}
	if r.ReconciledAs == nil || *r.ReconciledAs != 9999 {
		t.Error("ReconciledAs: want 9999")
	}
	if r.DeltaMS == nil || *r.DeltaMS != 300 {
		t.Error("DeltaMS: want 300")
	}
	if r.Confidence != "high" {
		t.Errorf("Confidence: want high, got %s", r.Confidence)
	}
	if r.AttributionPath != "fire_event" {
		t.Errorf("AttributionPath: want fire_event, got %s", r.AttributionPath)
	}
	if !r.SwapDetected {
		t.Error("SwapDetected: want true")
	}
	if !r.DelayedDamage {
		t.Error("DelayedDamage: want true")
	}
	if r.PlayerIndex == nil || *r.PlayerIndex != 2 {
		t.Error("PlayerIndex: want 2")
	}
}

func TestAttributionsToRows_NilPointersSafe(t *testing.T) {
	attrs := []analysis.KillAttribution{{
		XUID: "xuid1", TimeMS: 5000,
		WeaponID: nil, DeltaMS: nil, PlayerIndex: nil,
	}}
	rows := attributionsToRows(attrs, "xuid1")
	if len(rows) != 1 {
		t.Fatal("expected 1 row even with nil pointers")
	}
	if rows[0].WeaponID != nil {
		t.Error("WeaponID should be nil")
	}
	if rows[0].DeltaMS != nil {
		t.Error("DeltaMS should be nil")
	}
	if rows[0].PlayerIndex != nil {
		t.Error("PlayerIndex should be nil")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// InsertWeaponKills — idempotence et UBIGINT
// ─────────────────────────────────────────────────────────────────────────────

// TestInsertWeaponKills_Idempotent vérifie que deux insertions consécutives pour
// le même (match_id, xuid) produisent le bon nombre de lignes (DELETE + re-INSERT).
func TestInsertWeaponKills_Idempotent(t *testing.T) {
	db := openWeaponDB(t)
	rows := []WeaponKillRow{
		{TimeMS: 5000, WeaponID: ptrU64(123), Confidence: "high", AttributionPath: "fire_event"},
		{TimeMS: 10000, WeaponID: ptrU64(456), Confidence: "medium", AttributionPath: "formula_a"},
	}

	if err := InsertWeaponKills(t.Context(), db, "m1", "xuid1", rows); err != nil {
		t.Fatal(err)
	}
	if err := InsertWeaponKills(t.Context(), db, "m1", "xuid1", rows); err != nil {
		t.Fatalf("idempotent second insert: %v", err)
	}

	// Append-only #23046 (Phase 2) : idempotence LOGIQUE via v_weapon_kills (dernière
	// génération) ; le physique croît, la vue reste à 2 rows.
	var count int
	db.QueryRow("SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND xuid='xuid1'").Scan(&count)
	if count != 2 {
		t.Fatalf("expected 2 rows (v_weapon_kills) after idempotent re-insert, got %d", count)
	}
}

// TestInsertWeaponKills_EmptyPreservesExisting vérifie le garde-fou
// anti-perte de données : un appel avec attrs=[] doit être un no-op total et
// NE PAS supprimer les rows existantes pour (match_id, xuid).
//
// Régression historique (cf. thought_log 2026-05-09) : DELETE+INSERT avec
// attrs=[] avait silencieusement vidé ~1010 matchs en mai 2026 quand un
// --weapons --force retombait sur des films expirés (extraction = 0 attrib).
// La sémantique correcte d'un Insert vide est "rien à faire", pas "tout
// remplacer par rien".
func TestInsertWeaponKills_EmptyPreservesExisting(t *testing.T) {
	db := openWeaponDB(t)
	rows := []WeaponKillRow{
		{TimeMS: 5000, WeaponID: ptrU64(123), Confidence: "high", AttributionPath: "fire_event"},
	}
	if err := InsertWeaponKills(t.Context(), db, "m1", "xuid1", rows); err != nil {
		t.Fatal(err)
	}

	// Appel avec attrs=[] — ne doit PAS effacer la ligne précédente.
	if err := InsertWeaponKills(t.Context(), db, "m1", "xuid1", nil); err != nil {
		t.Fatalf("unexpected error on empty insert: %v", err)
	}
	if err := InsertWeaponKills(t.Context(), db, "m1", "xuid1", []WeaponKillRow{}); err != nil {
		t.Fatalf("unexpected error on empty slice insert: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND xuid='xuid1'").Scan(&count)
	if count != 1 {
		t.Fatalf("empty Insert must preserve existing rows, expected 1, got %d", count)
	}
}

// TestInsertWeaponKills_DoesNotDeleteOtherXuid vérifie que le DELETE ciblé
// (match+xuid) ne touche pas les lignes d'autres joueurs du même match.
func TestInsertWeaponKills_DoesNotDeleteOtherXuid(t *testing.T) {
	db := openWeaponDB(t)
	rows1 := []WeaponKillRow{{TimeMS: 5000, WeaponID: ptrU64(100), Confidence: "high", AttributionPath: "fire_event"}}
	rows2 := []WeaponKillRow{{TimeMS: 7000, WeaponID: ptrU64(200), Confidence: "medium", AttributionPath: "formula_a"}}

	if err := InsertWeaponKills(t.Context(), db, "m1", "xuid1", rows1); err != nil {
		t.Fatal(err)
	}
	if err := InsertWeaponKills(t.Context(), db, "m1", "xuid2", rows2); err != nil {
		t.Fatal(err)
	}
	// Re-insert xuid1 — doit laisser les lignes xuid2 intactes.
	if err := InsertWeaponKills(t.Context(), db, "m1", "xuid1", rows1); err != nil {
		t.Fatal(err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND xuid='xuid2'").Scan(&count)
	if count != 1 {
		t.Fatalf("xuid2 row must survive xuid1 re-insert, got %d", count)
	}
}

// TestInsertWeaponKills_UBigintMaxRoundtrip vérifie que les weapon_id UBIGINT
// dont le bit 63 est set (valeur > MaxInt64) survivent au round-trip DuckDB.
// Workaround : ubigintArg sérialise en string décimale + CAST AS UBIGINT.
func TestInsertWeaponKills_UBigintMaxRoundtrip(t *testing.T) {
	db := openWeaponDB(t)
	maxU64 := uint64(18446744073709551615)
	rows := []WeaponKillRow{
		{TimeMS: 1000, WeaponID: &maxU64, Confidence: "high", AttributionPath: "fire_event"},
	}
	if err := InsertWeaponKills(t.Context(), db, "m1", "xuid1", rows); err != nil {
		t.Fatalf("UBIGINT max insert: %v", err)
	}
	var readBack uint64
	db.QueryRow("SELECT weapon_id FROM weapon_kills WHERE match_id='m1'").Scan(&readBack)
	if readBack != maxU64 {
		t.Errorf("UBIGINT round-trip: want %d, got %d", maxU64, readBack)
	}
}

// TestInsertWeaponKills_NilWeaponID vérifie que weapon_id=nil → NULL en DB
// (pas d'erreur CAST NULL AS UBIGINT).
func TestInsertWeaponKills_NilWeaponID(t *testing.T) {
	db := openWeaponDB(t)
	rows := []WeaponKillRow{
		{TimeMS: 1000, WeaponID: nil, Confidence: "none", AttributionPath: "none"},
	}
	if err := InsertWeaponKills(t.Context(), db, "m1", "xuid1", rows); err != nil {
		t.Fatalf("nil weapon_id insert: %v", err)
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND weapon_id IS NULL").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 row with NULL weapon_id, got %d", count)
	}
}
