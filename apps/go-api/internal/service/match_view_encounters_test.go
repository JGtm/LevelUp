package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func TestConvertEncounters_OrdinalBadgeAttributed(t *testing.T) {
	t.Parallel()
	raw := []domain.EncounterRaw{
		{XUID: "x_p1", Gamertag: "PlayerOne", CountTogether: 5, IsAlly: true},
		{XUID: "x_p2", Gamertag: "PlayerTwo", CountTogether: 2, IsAlly: false},
		{XUID: "x_p3", Gamertag: "PlayerThree", CountTogether: 1, IsAlly: true}, // 1 seul match = pas d'ordinal
	}
	rows := convertEncounters(raw)
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
	rows := convertEncounters(raw)
	if len(rows[0].Badges) != 0 {
		t.Errorf("count_together=0 should yield no badges, got %+v", rows[0].Badges)
	}
}

func TestConvertEncounters_TypedLabelKey(t *testing.T) {
	t.Parallel()
	raw := []domain.EncounterRaw{
		{XUID: "x", Gamertag: "P", CountTogether: 3, IsAlly: true},
	}
	rows := convertEncounters(raw)
	for _, b := range rows[0].Badges {
		if b.Kind == "ordinal" && b.LabelKey != "narrative.encounter.ordinal" {
			t.Errorf("LabelKey want narrative.encounter.ordinal, got %s", b.LabelKey)
		}
	}
}
