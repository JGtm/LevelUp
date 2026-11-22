package replaybuild

// matchfacts_vip_test.go — LE VIP NOMME SES ACTIONS DANS `objectives[]`.
//
// Le décodeur ne range pas le VIP parmi les familles de variante (`ObjectiveTypeOf` rend "" pour
// un mode VIP) : sans la demande explicite de [objectiveFamilyOfVariant], `NamedEventsFrom` rend
// nil et aucune action VIP n'atteint le document.

import (
	"context"
	"testing"

	"levelup/go-api/internal/port"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

func TestObjectiveFamilyOfVariant(t *testing.T) {
	for variant, want := range map[string]string{
		"Arena:VIP":     decfilm.ObjectiveTypeVip,
		"Ranked:vip":    decfilm.ObjectiveTypeVip,
		"Arena:CTF":     decfilm.ObjectiveTypeFlag,
		"Strongholds":   decfilm.ObjectiveTypeZone,
		"Slayer:Arena":  "",
		"":              "",
		"Assault:Arena": decfilm.ObjectiveTypeBomb,
	} {
		if got := objectiveFamilyOfVariant(variant); got != want {
			t.Errorf("%q : famille %q, attendu %q", variant, got, want)
		}
	}
	if decfilm.ObjectiveTypeOf("Arena:VIP") != "" {
		t.Error("le décodeur range désormais le VIP parmi les variantes : retirer le détour de objectiveFamilyOfVariant")
	}
}

// TestIdentifiedEventsNommeLesTroisActionsVip — une élimination du VIP adverse (`20 B`), une
// assistance (`21 A`) et une élimination en étant VIP (`21 B`) arrivent attribuées dans la liste
// des objectifs, avec la désignation (`22 A`) ; le même film sous une variante sans famille n'en
// publie aucune.
func TestIdentifiedEventsNommeLesTroisActionsVip(t *testing.T) {
	rec := func(t, comp int, v decfilm.StatValue) decfilm.StatRecord {
		return decfilm.StatRecord{TimeMS: t, Slot: 22, Round: 0, Comps: map[int]decfilm.StatValue{comp: v}}
	}
	recs := []decfilm.StatRecord{
		rec(900, 0, decfilm.StatValue{A: 10}), rec(1900, 0, decfilm.StatValue{A: 20}), rec(2900, 0, decfilm.StatValue{A: 30}),
		rec(1000, 2, decfilm.StatValue{B: 1}), rec(2000, 2, decfilm.StatValue{B: 2}), rec(3000, 2, decfilm.StatValue{B: 3}),
		rec(1500, 22, decfilm.StatValue{A: 1}),       // vip_selected
		rec(2100, 20, decfilm.StatValue{B: 1}),       // vip_kills
		rec(2200, 21, decfilm.StatValue{A: 1}),       // vip_assists
		rec(2300, 21, decfilm.StatValue{A: 1, B: 1}), // vip_kills_as_vip (21 A garde sa valeur)
	}
	deaths := []types.Death{{XUID: 7, TimeMS: 1000}, {XUID: 7, TimeMS: 2000}, {XUID: 7, TimeMS: 3000}}
	pont := func() *replay.PontParManche {
		return replay.NouveauPontParManche(recs, deathInstantsOf(deaths), nil, nil)
	}
	ctx := context.Background()

	got, nonNommes, refuses := identifiedEvents(ctx, "m", filmDeaths{list: deaths}, recs,
		port.MatchFacts{GameVariantName: "Arena:VIP"}, pont())
	stats := map[string]string{}
	for _, e := range got {
		stats[e.Stat] = e.XUID
	}
	for _, stat := range []string{"vip_kills", "vip_assists", "vip_kills_as_vip"} {
		if stats[stat] != "7" {
			t.Errorf("%s : attribuée à %q, attendu \"7\" (publiées : %v, non nommées %d, refusées %d)",
				stat, stats[stat], stats, nonNommes, refuses)
		}
	}

	autre, _, _ := identifiedEvents(ctx, "m", filmDeaths{list: deaths}, recs,
		port.MatchFacts{GameVariantName: "Slayer:Arena"}, pont())
	if len(autre) != 0 {
		t.Errorf("variante sans famille : %d action(s) publiée(s), attendu 0", len(autre))
	}
}
