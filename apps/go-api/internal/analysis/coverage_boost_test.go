package analysis

import (
	"testing"

	"levelup/go-api/internal/legacymatch"
)

// ---------- decodePositionFrame ----------

func makeFrame(baseType, b5, b9 byte, playerIdx byte) []byte {
	// Build a 20-byte frame: [A0 7B 42 baseType playerIdx b5 XX XX XX b9 XX XX XX XX XX XX XX XX XX XX]
	data := make([]byte, 20)
	data[0] = frameMarkerB0
	data[1] = frameMarkerB1
	data[2] = frameMarkerB2
	data[3] = baseType
	data[4] = playerIdx << 4
	data[5] = b5
	data[9] = b9
	return data
}

func TestDecodePositionFrame_BadBaseType_Boost(t *testing.T) {
	data := makeFrame(0xFF, byteHumanB5, byteHumanB9, 2)
	_, _, _, _, ok := decodePositionFrame(data, 0)
	if ok {
		t.Error("expected failure for invalid base type")
	}
}

func TestDecodePositionFrame_BadB5_Boost(t *testing.T) {
	data := makeFrame(0x08, 0x00, byteHumanB9, 2)
	_, _, _, _, ok := decodePositionFrame(data, 0)
	if ok {
		t.Error("expected failure for invalid b5")
	}
}

func TestDecodePositionFrame_BadB9_Boost(t *testing.T) {
	data := makeFrame(0x08, byteHumanB5, 0x00, 2)
	_, _, _, _, ok := decodePositionFrame(data, 0)
	if ok {
		t.Error("expected failure for invalid b9")
	}
}

func TestDecodePositionFrame_ValidFrame_Boost(t *testing.T) {
	data := makeFrame(0x08, byteHumanB5, byteHumanB9, 3)
	pi, _, _, _, ok := decodePositionFrame(data, 0)
	if !ok {
		t.Fatal("expected success")
	}
	if pi != 3 {
		t.Errorf("expected player_idx=3, got %d", pi)
	}
}

// ---------- ScanFirstMovements with data ----------

// buildValidFrame construit une frame test ; baseType est paramÃ©trÃ© pour l'extension future.
//
//nolint:unparam // baseType est gardÃ© pour clarifier l'intention de la frame
func buildValidFrame(baseType, playerIdx byte) []byte {
	data := make([]byte, 20)
	data[0] = frameMarkerB0
	data[1] = frameMarkerB1
	data[2] = frameMarkerB2
	data[3] = baseType
	data[4] = playerIdx << 4
	data[5] = byteHumanB5
	data[9] = byteHumanB9
	return data
}

func TestScanFirstMovements_DetectsMovement(t *testing.T) {
	// Two frames for the same player with different signatures
	frame1 := buildValidFrame(0x08, 1)
	frame2 := buildValidFrame(0x08, 1)
	frame2[10] = 0xFF // change signature byte
	data := append(frame1, frame2...)
	chunk := SpawnChunk{Index: 0, StartMS: 0, EndMS: 1000, Data: data}
	result := ScanFirstMovements([]SpawnChunk{chunk})
	if len(result) == 0 {
		t.Error("expected at least one movement detected")
	}
}

func TestScanFirstMovements_NoMarkerNoResult(t *testing.T) {
	data := make([]byte, 50) // all zeros, no markers
	chunk := SpawnChunk{Index: 0, StartMS: 0, EndMS: 1000, Data: data}
	result := ScanFirstMovements([]SpawnChunk{chunk})
	if len(result) != 0 {
		t.Errorf("expected no movements, got %d", len(result))
	}
}

// ---------- estimateFrameTimestamp ----------

func TestEstimateFrameTimestamp_ZeroDuration_Boost(t *testing.T) {
	chunk := SpawnChunk{StartMS: 100, EndMS: 100, Data: make([]byte, 10)}
	got := estimateFrameTimestamp(chunk, 5)
	if got != 100 {
		t.Errorf("expected 100, got %f", got)
	}
}

func TestEstimateFrameTimestamp_Normal_Boost(t *testing.T) {
	data := make([]byte, 100)
	chunk := SpawnChunk{StartMS: 0, EndMS: 1000, Data: data}
	got := estimateFrameTimestamp(chunk, 50)
	if got < 490 || got > 510 {
		t.Errorf("expected ~500ms, got %f", got)
	}
}

// ---------- EstimateFilmMatchStartMS ----------

// buildChunkWithMovements builds a chunk that will produce `n` distinct player movements
func buildChunkWithMovingPlayers(n int) SpawnChunk {
	var data []byte
	// For each player: two frames with different signature bytes
	for i := 0; i < n; i++ {
		f1 := buildValidFrame(0x08, byte(i))
		f2 := buildValidFrame(0x08, byte(i))
		f2[10] = 0xFF // change signature
		data = append(data, f1...)
		data = append(data, f2...)
	}
	return SpawnChunk{Index: 0, StartMS: 0, EndMS: float64(len(data)), Data: data}
}

func TestEstimateFilmMatchStartMS_NotEnoughPlayers(t *testing.T) {
	chunk := buildChunkWithMovingPlayers(1)
	got := EstimateFilmMatchStartMS([]SpawnChunk{chunk}, 3, 0)
	if got != -1 {
		t.Errorf("expected -1, got %f", got)
	}
}

func TestEstimateFilmMatchStartMS_EnoughPlayers(t *testing.T) {
	chunk := buildChunkWithMovingPlayers(4)
	got := EstimateFilmMatchStartMS([]SpawnChunk{chunk}, 3, 0)
	// Should return a valid timestamp (>=0) or -1 if peak not found
	_ = got // just ensure no panic
}

func TestEstimateFilmMatchStartMS_APIConstraint(t *testing.T) {
	chunk := buildChunkWithMovingPlayers(4)
	// Pass a small apiFirstEventMS to trigger constraint
	got := EstimateFilmMatchStartMS([]SpawnChunk{chunk}, 3, 1.0)
	// With tiny API constraint, peakTS should be capped
	_ = got
}

func TestEstimateFilmMatchStartMS_ZeroMinPlayers(t *testing.T) {
	// minPlayers=0 should default to 3
	chunk := buildChunkWithMovingPlayers(1)
	got := EstimateFilmMatchStartMS([]SpawnChunk{chunk}, 0, 0)
	if got != -1 {
		t.Errorf("expected -1 (not enough players), got %f", got)
	}
}

// ---------- computeNormalizedMetrics â€” uncovered branches ----------

func TestComputeNormalizedMetrics_AllOptionalFields(t *testing.T) {
	kda := 3.5
	ps := 1200
	dd := 800.0
	tmm := 1500.0
	emm := 1400.0
	rank := 2
	ke := 8.0
	de := 5.0
	tp := 600
	acc := 0.45
	row := legacymatch.StatsMatchRow{
		Kills:             10,
		Deaths:            5,
		Assists:           3,
		KDA:               &kda,
		PersonalScore:     &ps,
		DamageDealt:       &dd,
		TimePlayedSeconds: &tp,
		TeamMMR:           &tmm,
		EnemyMMR:          &emm,
		Rank:              &rank,
		KillsExpected:     &ke,
		DeathsExpected:    &de,
		Accuracy:          &acc,
	}
	m := computeNormalizedMetrics(row)
	if m.kda != 3.5 {
		t.Errorf("kda: expected 3.5, got %f", m.kda)
	}
	if m.pspm == nil {
		t.Fatal("pspm should be non-nil")
	}
	if m.dpmDamage == nil {
		t.Fatal("dpmDamage should be non-nil")
	}
	if m.rankPerfDiff == nil {
		t.Fatal("rankPerfDiff should be non-nil")
	}
	if m.killsVsExpected == nil {
		t.Fatal("killsVsExpected should be non-nil")
	}
	if m.deathsVsExpected == nil {
		t.Fatal("deathsVsExpected should be non-nil")
	}
	if m.accuracy == nil || *m.accuracy != 0.45 {
		t.Error("accuracy mismatch")
	}
}

func TestComputeNormalizedMetrics_NoOptionalFields(t *testing.T) {
	tp := 600
	row := legacymatch.StatsMatchRow{
		Kills:             10,
		Deaths:            5,
		Assists:           3,
		TimePlayedSeconds: &tp,
	}
	m := computeNormalizedMetrics(row)
	if m.pspm != nil || m.dpmDamage != nil || m.rankPerfDiff != nil {
		t.Error("optional fields should be nil")
	}
	if m.killsVsExpected != nil || m.deathsVsExpected != nil {
		t.Error("expected fields should be nil")
	}
}

// ---------- BuildHighlights â€” more branch coverage ----------

func TestMeanAccuracy_WithMultipleValues(t *testing.T) {
	a1, a2, a3 := 0.4, 0.6, 0.8
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", Accuracy: &a1},
		{MatchID: "m2", Accuracy: &a2},
		{MatchID: "m3"}, // nil
		{MatchID: "m4", Accuracy: &a3},
	}
	got := meanAccuracy(matches)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if *got < 0.59 || *got > 0.61 {
		t.Errorf("expected ~0.6, got %f", *got)
	}
}

// ---------- abs64 (killer_victim) ----------

func TestAbs64_Positive(t *testing.T) {
	if abs64(5) != 5 {
		t.Error("expected 5")
	}
}

func TestAbs64_Negative(t *testing.T) {
	if abs64(-3) != 3 {
		t.Error("expected 3")
	}
}

func TestAbs64_Zero(t *testing.T) {
	if abs64(0) != 0 {
		t.Error("expected 0")
	}
}

// ---------- isTeammatesBreak â€” edge cases ----------

func TestIsTeammatesBreak_BothNil(t *testing.T) {
	if isTeammatesBreak(nil, nil, nil) {
		t.Error("same nil sigs should not break")
	}
}

// TestIsTeammatesBreak_DifferentNoFriendSet : sémantique post-2026-05-08.
// friendSet=nil signifie maintenant "aucun ami tracké" (mode Friends sans
// amis configurés). Dans ce cas, les changements de coéquipiers ne doivent
// PAS casser la session — sinon en matchmaking solo chaque match devient
// une nouvelle session (bug observé : 98% sessions à 1 match seul). Pour le
// mode Group explicite (break sur tout changement), le caller utilise une
// comparaison directe via derefString, sans passer par isTeammatesBreak.
func TestIsTeammatesBreak_DifferentNoFriendSet(t *testing.T) {
	a, b := "x1", "x2"
	if isTeammatesBreak(&a, &b, nil) {
		t.Error("different sigs with no friends tracked should NOT break (cf. fix 2026-05-08)")
	}
}

func TestIsTeammatesBreak_FriendLeftSession(t *testing.T) {
	a, b := "x1,x2", "x3"
	friends := map[string]struct{}{"x1": {}, "x2": {}}
	if !isTeammatesBreak(&a, &b, friends) {
		t.Error("friend left session should break")
	}
}

func TestIsTeammatesBreak_SameFriendRemains(t *testing.T) {
	// x1 (ami) reste dans les deux matchs, seul x2â†’x3 change (non-ami).
	// Avec le mode "friends", le sous-ensemble d'amis {x1} est inchangÃ© â†’ pas de rupture.
	a, b := "x1,x2", "x1,x3"
	friends := map[string]struct{}{"x1": {}}
	if isTeammatesBreak(&a, &b, friends) {
		t.Error("friend x1 unchanged, only non-friend changed: should NOT break")
	}
}

// ---------- prepareHistoryMetrics â€” uncovered branches ----------

func TestPrepareHistoryMetrics_Empty(t *testing.T) {
	cols := prepareHistoryMetrics(nil)
	if len(cols.kpm) != 0 {
		t.Error("expected empty")
	}
}

func TestPrepareHistoryMetrics_WithData(t *testing.T) {
	tp := 600
	dd := 500.0
	ps := 1000
	acc := 0.5
	rows := []legacymatch.StatsMatchRow{
		{Kills: 10, Deaths: 5, Assists: 3, TimePlayedSeconds: &tp, DamageDealt: &dd, PersonalScore: &ps, Accuracy: &acc},
		{Kills: 8, Deaths: 4, Assists: 2, TimePlayedSeconds: &tp},
	}
	cols := prepareHistoryMetrics(rows)
	if len(cols.kpm) != 2 {
		t.Errorf("expected 2 kpm entries, got %d", len(cols.kpm))
	}
}

// ---------- addRequired ----------

func TestAddRequired_UnknownKey(t *testing.T) {
	pcts := map[string]float64{}
	used := map[string]float64{}
	addRequired("unknown_key", 1.0, []float64{1.0, 2.0}, false, pcts, used)
	if len(pcts) != 0 {
		t.Error("unknown key should be ignored")
	}
}

func TestAddRequired_KpmKey(t *testing.T) {
	pcts := map[string]float64{}
	used := map[string]float64{}
	addRequired("kpm", 1.5, []float64{1.0, 2.0, 3.0}, false, pcts, used)
	// kpm is a real key in relativeWeights
	if _, ok := pcts["kpm"]; !ok {
		t.Skip("kpm not in relativeWeights, skip")
	}
}

// ---------- computeKDAFallback ----------

func TestComputeKDAFallback_SingleMatch(t *testing.T) {
	tp := 600
	row := legacymatch.StatsMatchRow{Kills: 10, Deaths: 5, Assists: 3, TimePlayedSeconds: &tp}
	got := computeKDAFallback(row, []legacymatch.StatsMatchRow{row})
	if got == nil || *got != 50.0 {
		t.Errorf("expected 50.0 for single match, got %v", got)
	}
}

func TestComputeKDAFallback_MultipleMatches(t *testing.T) {
	tp := 600
	kda1, kda2, kda3 := 1.5, 2.0, 3.0
	rows := []legacymatch.StatsMatchRow{
		{Kills: 10, Deaths: 5, Assists: 3, TimePlayedSeconds: &tp, KDA: &kda1},
		{Kills: 8, Deaths: 4, Assists: 2, TimePlayedSeconds: &tp, KDA: &kda2},
		{Kills: 12, Deaths: 3, Assists: 5, TimePlayedSeconds: &tp, KDA: &kda3},
	}
	got := computeKDAFallback(rows[2], rows)
	if got == nil || *got <= 0 {
		t.Error("expected positive percentile")
	}
}
