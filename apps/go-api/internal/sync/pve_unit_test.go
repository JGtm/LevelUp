package sync

import (
	"testing"
)

// ── cleanXUID ────────────────────────────────────────────────────────────────

func TestCleanXUID_WithPrefix(t *testing.T) {
	if got := cleanXUID("xuid(1234567890)"); got != "1234567890" {
		t.Fatalf("expected 1234567890, got %s", got)
	}
}

func TestCleanXUID_NoPrefix(t *testing.T) {
	if got := cleanXUID("1234567890"); got != "1234567890" {
		t.Fatalf("expected 1234567890, got %s", got)
	}
}

func TestCleanXUID_Empty(t *testing.T) {
	if got := cleanXUID(""); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

// ── float64From ──────────────────────────────────────────────────────────────

func TestFloat64From_Valid(t *testing.T) {
	m := map[string]any{"dmg": 1234.5}
	if got := float64From(m, "dmg"); got != 1234.5 {
		t.Fatalf("expected 1234.5, got %f", got)
	}
}

func TestFloat64From_Missing(t *testing.T) {
	m := map[string]any{}
	if got := float64From(m, "dmg"); got != 0 {
		t.Fatalf("expected 0, got %f", got)
	}
}

func TestFloat64From_WrongType(t *testing.T) {
	m := map[string]any{"dmg": "abc"}
	if got := float64From(m, "dmg"); got != 0 {
		t.Fatalf("expected 0, got %f", got)
	}
}

// ── extractPlayerXUID ────────────────────────────────────────────────────────

func TestExtractPlayerXUID_Xuid(t *testing.T) {
	pm := map[string]any{"Xuid": "xuid(111)"}
	if got := extractPlayerXUID(pm); got != "111" {
		t.Fatalf("expected 111, got %s", got)
	}
}

func TestExtractPlayerXUID_PlayerId(t *testing.T) {
	pm := map[string]any{"PlayerId": "xuid(222)"}
	if got := extractPlayerXUID(pm); got != "222" {
		t.Fatalf("expected 222, got %s", got)
	}
}

func TestExtractPlayerXUID_Fallback(t *testing.T) {
	pm := map[string]any{
		"PlayerProperties": []any{
			map[string]any{"PlayerId": "xuid(333)"},
		},
	}
	if got := extractPlayerXUID(pm); got != "333" {
		t.Fatalf("expected 333, got %s", got)
	}
}

func TestExtractPlayerXUID_Empty(t *testing.T) {
	pm := map[string]any{}
	if got := extractPlayerXUID(pm); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

// ── findPveStatsDict ─────────────────────────────────────────────────────────

func TestFindPveStatsDict_FirefightStats(t *testing.T) {
	pm := map[string]any{
		"PlayerTeamStats": []any{
			map[string]any{
				"PlayerStats": map[string]any{
					"FirefightStats": map[string]any{"WavesCompleted": float64(5)},
				},
			},
		},
	}
	got := findPveStatsDict(pm)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got["WavesCompleted"] != float64(5) {
		t.Fatalf("expected WavesCompleted=5, got %v", got["WavesCompleted"])
	}
}

func TestFindPveStatsDict_InlinedGruntKills(t *testing.T) {
	pm := map[string]any{
		"PlayerTeamStats": []any{
			map[string]any{
				"PlayerStats": map[string]any{
					"GruntKills": float64(10),
				},
			},
		},
	}
	got := findPveStatsDict(pm)
	if got == nil {
		t.Fatal("expected non-nil for inlined GruntKills")
	}
}

func TestFindPveStatsDict_NilWhenEmpty(t *testing.T) {
	pm := map[string]any{}
	if got := findPveStatsDict(pm); got != nil {
		t.Fatal("expected nil")
	}
}

func TestFindPveStatsDict_NilWhenNoStats(t *testing.T) {
	pm := map[string]any{
		"PlayerTeamStats": []any{
			map[string]any{
				"PlayerStats": map[string]any{},
			},
		},
	}
	if got := findPveStatsDict(pm); got != nil {
		t.Fatal("expected nil when no PvE stats keys")
	}
}

// ── computePveBits ───────────────────────────────────────────────────────────

func TestComputePveBits_AllZero(t *testing.T) {
	row := PveMatchStatsRow{}
	if got := computePveBits(row); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestComputePveBits_GruntOnly(t *testing.T) {
	row := PveMatchStatsRow{GruntKills: 1, TotalKills: 1}
	bits := computePveBits(row)
	if bits&PveBitGrunt == 0 {
		t.Fatal("expected PveBitGrunt set")
	}
	if bits&PveBitTotalKills == 0 {
		t.Fatal("expected PveBitTotalKills set")
	}
}

func TestComputePveBits_AllEnemies(t *testing.T) {
	row := PveMatchStatsRow{
		TotalKills:    14,
		BossKills:     1,
		GruntKills:    1,
		EliteKills:    1,
		JackalKills:   1,
		BruteKills:    1,
		HunterKills:   1,
		SkimmerKills:  1,
		CrawlerKills:  1,
		SoldierKills:  1,
		KnightKills:   1,
		WardenKills:   1,
		SentinelKills: 1,
		MarineKills:   1,
	}
	bits := computePveBits(row)
	// All enemy bits should be set
	if bits&PveBitAllEnemies != PveBitAllEnemies {
		t.Fatalf("expected all enemy bits set, got %b", bits)
	}
	if bits&PveBitTotalKills == 0 {
		t.Fatal("expected PveBitTotalKills")
	}
	if bits&PveBitBossKills == 0 {
		t.Fatal("expected PveBitBossKills")
	}
}

// ── fillEnemyKills ───────────────────────────────────────────────────────────

func TestFillEnemyKills_EnemyKillsByType(t *testing.T) {
	row := &PveMatchStatsRow{}
	pve := map[string]any{
		"EnemyKillsByType": map[string]any{
			"Grunt": float64(5),
			"Elite": float64(3),
		},
	}
	fillEnemyKills(row, pve)
	if row.GruntKills != 5 {
		t.Fatalf("expected GruntKills=5, got %d", row.GruntKills)
	}
	if row.EliteKills != 3 {
		t.Fatalf("expected EliteKills=3, got %d", row.EliteKills)
	}
}

func TestFillEnemyKills_DirectFields(t *testing.T) {
	row := &PveMatchStatsRow{}
	pve := map[string]any{
		"GruntKills": float64(7),
		"BruteKills": float64(2),
	}
	fillEnemyKills(row, pve)
	if row.GruntKills != 7 {
		t.Fatalf("expected GruntKills=7, got %d", row.GruntKills)
	}
	if row.BruteKills != 2 {
		t.Fatalf("expected BruteKills=2, got %d", row.BruteKills)
	}
}

// ── buildPveRow ──────────────────────────────────────────────────────────────

func TestBuildPveRow_Basic(t *testing.T) {
	pm := map[string]any{
		"Deaths":           float64(3),
		"TotalDamageDealt": float64(1000.0),
	}
	pve := map[string]any{
		"WavesCompleted": float64(4),
		"BossKills":      float64(1),
		"GruntKills":     float64(10),
	}
	row := buildPveRow("match-1", "xuid-1", pm, pve)
	if row.MatchID != "match-1" {
		t.Fatalf("expected match-1, got %s", row.MatchID)
	}
	if row.XUID != "xuid-1" {
		t.Fatalf("expected xuid-1, got %s", row.XUID)
	}
	if row.WavesCompleted != 4 {
		t.Fatalf("expected WavesCompleted=4, got %d", row.WavesCompleted)
	}
	if row.Deaths != 3 {
		t.Fatalf("expected Deaths=3, got %d", row.Deaths)
	}
	if row.DamageDealt != 1000.0 {
		t.Fatalf("expected DamageDealt=1000, got %f", row.DamageDealt)
	}
	if row.GruntKills != 10 {
		t.Fatalf("expected GruntKills=10, got %d", row.GruntKills)
	}
	if row.TotalKills != 10 {
		t.Fatalf("expected TotalKills=10, got %d", row.TotalKills)
	}
	if row.PveBitsValue == 0 {
		t.Fatal("expected non-zero PveBitsValue")
	}
}

// ── ExtractPveStats ──────────────────────────────────────────────────────────

func TestExtractPveStats_NoPlayers(t *testing.T) {
	json := map[string]any{}
	if got := ExtractPveStats("m1", json); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestExtractPveStats_EmptyPlayers(t *testing.T) {
	json := map[string]any{"Players": []any{}}
	if got := ExtractPveStats("m1", json); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestExtractPveStats_ValidPlayer(t *testing.T) {
	json := map[string]any{
		"Players": []any{
			map[string]any{
				"Xuid":   "xuid(123)",
				"Deaths": float64(2),
				"PlayerTeamStats": []any{
					map[string]any{
						"PlayerStats": map[string]any{
							"FirefightStats": map[string]any{
								"WavesCompleted": float64(3),
								"BossKills":      float64(1),
								"EnemyKillsByType": map[string]any{
									"Grunt": float64(5),
									"Elite": float64(2),
								},
							},
						},
					},
				},
			},
		},
	}
	rows := ExtractPveStats("match-1", json)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].XUID != "123" {
		t.Fatalf("expected XUID=123, got %s", rows[0].XUID)
	}
	if rows[0].GruntKills != 5 {
		t.Fatalf("expected GruntKills=5, got %d", rows[0].GruntKills)
	}
	if rows[0].TotalKills != 7 {
		t.Fatalf("expected TotalKills=7, got %d", rows[0].TotalKills)
	}
}

func TestExtractPveStats_SkipInvalidPlayer(t *testing.T) {
	json := map[string]any{
		"Players": []any{
			"not-a-map",
			map[string]any{}, // no xuid
		},
	}
	rows := ExtractPveStats("m1", json)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}
