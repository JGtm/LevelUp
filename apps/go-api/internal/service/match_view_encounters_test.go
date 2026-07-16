package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func TestConvertEncounters_OrdinalBadgeAttributed(t *testing.T) {
	t.Parallel()
	raw := []domain.EncounterRaw{
		{XUID: "x_p1", Gamertag: "PlayerOne", CountTogether: 5, IsAlly: true},
		{XUID: "x_p2", Gamertag: "PlayerTwo", CountTogether: 2, IsAlly: false},
		{XUID: "x_p3", Gamertag: "PlayerThree", CountTogether: 1, IsAlly: true}, // 1 seul match = pas d'ordinal
	}
	rows := convertEncounters(raw, nil, time.Now())
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}

	// p1 : 5 matchs ensemble = ordinal 4
	hasOrdinal1 := false
	for _, b := range rows[0].Badges {
		if b.Kind == "ordinal" {
			hasOrdinal1 = true
			if got := b.Detail["ordinal"]; got != 4 {
				t.Errorf("p1 ordinal value want 4, got %v", got)
			}
		}
	}
	if !hasOrdinal1 {
		t.Errorf("p1 should have ordinal badge, got %+v", rows[0].Badges)
	}

	// p2 : 2 matchs ensemble = ordinal 1
	if len(rows[1].Badges) == 0 {
		t.Errorf("p2 should have ordinal badge")
	}

	// p3 : 1 seul match = pas d'ordinal
	for _, b := range rows[2].Badges {
		if b.Kind == "ordinal" {
			t.Errorf("p3 (count=1) should NOT have ordinal badge, got %+v", b)
		}
	}
}

func TestConvertEncounters_BadgesEmptyForFreshEncounter(t *testing.T) {
	t.Parallel()
	raw := []domain.EncounterRaw{
		{XUID: "x_new", Gamertag: "NewPlayer", CountTogether: 0, IsAlly: true},
	}
	rows := convertEncounters(raw, nil, time.Now())
	if len(rows[0].Badges) != 0 {
		t.Errorf("count_together=0 should yield no badges, got %+v", rows[0].Badges)
	}
}

func TestConvertEncounters_TypedLabelKey(t *testing.T) {
	t.Parallel()
	raw := []domain.EncounterRaw{
		{XUID: "x", Gamertag: "P", CountTogether: 3, IsAlly: true},
	}
	rows := convertEncounters(raw, nil, time.Now())
	for _, b := range rows[0].Badges {
		if b.Kind == "ordinal" && b.LabelKey != "narrative.encounter.ordinal" {
			t.Errorf("LabelKey want narrative.encounter.ordinal, got %s", b.LabelKey)
		}
	}
}

// TestConvertEncounters_AllyPlusBadge_FromRichStats : MV4.C' — vérifie que
// le badge ally_plus est attribué quand winrate_as_ally > seuil + ally_count
// >= MinEncountersForBadge.
func TestConvertEncounters_AllyPlusBadge_FromRichStats(t *testing.T) {
	t.Parallel()
	raw := []domain.EncounterRaw{
		{XUID: "x_great_ally", Gamertag: "GreatAlly", CountTogether: 10, IsAlly: true},
	}
	stats := []domain.EncounterStatsRaw{
		{
			XUID:          "x_great_ally",
			AllyCount:     8,
			WinsAsAlly:    7,
			LossesAsAlly:  1, // winrate = 7/8 = 0.875 > 0.7 seuil
			EnemyCount:    2,
			WinsVsEnemy:   1,
			LossesVsEnemy: 1,
		},
	}
	rows := convertEncounters(raw, stats, time.Now())
	hasAllyPlus := false
	for _, b := range rows[0].Badges {
		if b.Kind == "ally_plus" {
			hasAllyPlus = true
		}
	}
	if !hasAllyPlus {
		t.Errorf("ally_plus should be attributed when winrate_as_ally > 0.7, got %+v", rows[0].Badges)
	}
}

// TestConvertEncounters_ToughEnemyBadge_FromRichStats : MV4.C' — vérifie que
// le badge tough_enemy est attribué quand kd_against_me dépasse le seuil.
func TestConvertEncounters_ToughEnemyBadge_FromRichStats(t *testing.T) {
	t.Parallel()
	raw := []domain.EncounterRaw{
		{XUID: "x_nemesis", Gamertag: "Nemesis", CountTogether: 5, IsAlly: false},
	}
	stats := []domain.EncounterStatsRaw{
		{
			XUID:           "x_nemesis",
			AllyCount:      0,
			EnemyCount:     5,
			KillsDealt:     2, // moi tue Nemesis 2 fois
			DeathsSuffered: 8, // Nemesis tue moi 8 fois -> kd_against_me = 4 > 1.5 seuil
		},
	}
	rows := convertEncounters(raw, stats, time.Now())
	hasToughEnemy := false
	for _, b := range rows[0].Badges {
		if b.Kind == "tough_enemy" {
			hasToughEnemy = true
		}
	}
	if !hasToughEnemy {
		t.Errorf("tough_enemy should be attributed when kd_against_me > 1.5, got %+v", rows[0].Badges)
	}
}

// TestConvertEncounters_NoAllyPlus_BelowThreshold : winrate sous le seuil.
func TestConvertEncounters_NoAllyPlus_BelowThreshold(t *testing.T) {
	t.Parallel()
	raw := []domain.EncounterRaw{
		{XUID: "x", Gamertag: "P", CountTogether: 5, IsAlly: true},
	}
	stats := []domain.EncounterStatsRaw{
		{XUID: "x", AllyCount: 5, WinsAsAlly: 2, LossesAsAlly: 3}, // 0.4 < 0.7
	}
	rows := convertEncounters(raw, stats, time.Now())
	for _, b := range rows[0].Badges {
		if b.Kind == "ally_plus" {
			t.Errorf("ally_plus should NOT be attributed when winrate < 0.7, got %+v", b)
		}
	}
}

// TestConvertEncounters_DegradesGracefullyWhenStatsMissing : si stats manquant
// pour un xuid, on retombe sur le badge ordinal seul.
func TestConvertEncounters_DegradesGracefullyWhenStatsMissing(t *testing.T) {
	t.Parallel()
	raw := []domain.EncounterRaw{
		{XUID: "x_with_stats", Gamertag: "WithStats", CountTogether: 3, IsAlly: true},
		{XUID: "x_no_stats", Gamertag: "NoStats", CountTogether: 5, IsAlly: false},
	}
	// stats seulement pour x_with_stats
	stats := []domain.EncounterStatsRaw{
		{XUID: "x_with_stats", AllyCount: 2, WinsAsAlly: 2, LossesAsAlly: 0},
	}
	rows := convertEncounters(raw, stats, time.Now())
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	// x_no_stats should still have ordinal badge (count=5)
	hasOrdinal := false
	for _, b := range rows[1].Badges {
		if b.Kind == "ordinal" {
			hasOrdinal = true
		}
	}
	if !hasOrdinal {
		t.Errorf("x_no_stats should keep ordinal badge via fallback, got %+v", rows[1].Badges)
	}
}

func TestEncounterWinrate_NilWhenEmpty(t *testing.T) {
	t.Parallel()
	if got := encounterWinrate(0, 0); got != nil {
		t.Errorf("(0+0): want nil, got %v", got)
	}
	if got := encounterWinrate(3, 1); got == nil || *got != 0.75 {
		t.Errorf("(3+1): want 0.75, got %v", got)
	}
}

// TestConvertEncounters_RelationSolidBadge_DuoGagnant : parité avec le hub
// Communauté > Relations — le tableau « Historique des rencontres » attribue
// désormais les badges « solid » (ici duo_gagnant : taux de victoire allié
// >= 0.60 sur >= 10 matchs en allié).
func TestConvertEncounters_RelationSolidBadge_DuoGagnant(t *testing.T) {
	t.Parallel()
	raw := []domain.EncounterRaw{
		{XUID: "x_duo", Gamertag: "Duo", CountTogether: 12, IsAlly: true},
	}
	stats := []domain.EncounterStatsRaw{
		{XUID: "x_duo", AllyCount: 12, WinsAsAlly: 8, LossesAsAlly: 4}, // 0.667 >= 0.60
	}
	rows := convertEncounters(raw, stats, time.Now())
	if !hasEncounterBadgeKind(rows[0].Badges, "duo_gagnant") {
		t.Errorf("duo_gagnant attendu (winrate allié 0.667 sur 12 matchs), got %+v", rows[0].Badges)
	}
}

// TestConvertEncounters_RelationSolidBadge_Recrue vérifie le câblage de FirstSeen
// (Q23b -> EncounterStatsRaw -> relations) : badge recrue quand la relation est
// récente (< 30 j) et déjà significative (>= 4 matchs).
func TestConvertEncounters_RelationSolidBadge_Recrue(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	raw := []domain.EncounterRaw{
		{XUID: "x_new", Gamertag: "Rookie", CountTogether: 5, IsAlly: true},
	}
	stats := []domain.EncounterStatsRaw{
		{XUID: "x_new", AllyCount: 5, FirstSeen: now.AddDate(0, 0, -10)}, // il y a 10 j (< 30)
	}
	rows := convertEncounters(raw, stats, now)
	if !hasEncounterBadgeKind(rows[0].Badges, "recrue") {
		t.Errorf("recrue attendu (first_seen il y a 10 j, 5 matchs), got %+v", rows[0].Badges)
	}
}

// hasEncounterBadgeKind : helper test — vrai si un badge du kind donné est présent.
func hasEncounterBadgeKind(badges []domain.MatchEncounterBadge, kind string) bool {
	for _, b := range badges {
		if b.Kind == kind {
			return true
		}
	}
	return false
}
