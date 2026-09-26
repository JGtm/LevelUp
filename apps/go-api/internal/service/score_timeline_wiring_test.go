// score_timeline_wiring_test.go — branchement de `score_timeline_kind` sur l'en-tête de la
// Match View : comment le bloc « Score dans le temps » doit se montrer sur le mode joué.
//
// CE QUE CES TESTS PROTÈGENT :
//  1. la SOURCE est le `pair_name` BRUT, dont on ne retire que le suffixe de CARTE. Ni
//     `ModeUI` (locale-aware : sous UI FR « Slayer » y est déjà « Assassin »), ni
//     `NormalizeModeLabel`, qui MANGE le jeton sur toute une famille de pair_name — 460
//     matchs du registre local au mauvais verdict, dont les 429 de « Super Fiesta:Slayer » ;
//  2. le REPLI SE DIT PAR L'ABSENCE : `curve` étant le défaut du client, l'en-tête laisse
//     le champ VIDE plutôt que de redire le défaut sur chaque match ;
//  3. l'absence de règle (titre sans table) laisse le champ vide, sans erreur.
package service

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
)

// scoreTimelineRule : la règle réelle du titre, chargée depuis un extrait du TOML de prod.
func scoreTimelineRule(t *testing.T) func(string) string {
	t.Helper()
	set, err := mappings.LoadRegulationFromBytes("regulation.toml", []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 5

[score_timeline]
"Slayer"           = "hidden"
"Team Snipers"     = "hidden"
"CTF"              = "events"
"Neutral Flag"     = "events"
"King of the Hill" = "events"
"Assault"          = "events"
"One Bomb"         = "events"
`))
	if err != nil {
		t.Fatalf("chargement de la règle: %v", err)
	}
	return set.ScoreTimelineKind
}

// Les témoins RÉELS du registre local, en assertion nominative. `""` = le repli sûr, qui se
// dit en se taisant : le client garde la courbe.
func TestApplyMatchHeaderScoreTimeline_RealPairNames(t *testing.T) {
	resolve := scoreTimelineRule(t)
	cases := map[string]string{
		// Le mode le plus joué du corpus (429 matchs) — normalisé, il rendait
		// « Super Fiesta » et ratait son jeton.
		"Super Fiesta:Slayer on Forbidden - Forge": "hidden",
		// Témoin f0220a96.
		"Community:Team Slayer on Starboard": "hidden",
		"Arena:Team Snipers":                 "hidden",
		"Team Slayer:Arena":                  "hidden",
		"Husky Raid:CTF on Catalyst":         "events",
		"Arena:Neutral Flag CTF":             "events",
		"Assault:One Bomb":                   "events",
		"Arena:King of the Hill":             "events",
		// Témoin 696a9d7c, et les deux autres modes non déclarés : repli, donc champ VIDE.
		"Arena:Strongholds": "",
		"Arena:Oddball":     "",
		"BTB:Total Control": "",
	}
	for pair, want := range cases {
		h := domain.MatchViewHeader{}
		applyMatchHeaderScoreTimeline(&h, &domain.MatchMetaRaw{PairName: strPtr(pair)}, resolve)
		if h.ScoreTimelineKind != want {
			t.Errorf("pair_name %q -> %q, want %q", pair, h.ScoreTimelineKind, want)
		}
	}
}

// Sans règle injectée (titre qui ne déclare pas la table), sans pair_name, ou sur un libellé
// que le retrait du suffixe de carte vide : le champ reste VIDE et le client garde la
// courbe. Jamais d'erreur, jamais un bloc effacé par accident.
func TestApplyMatchHeaderScoreTimeline_DegradesToEmpty(t *testing.T) {
	resolve := scoreTimelineRule(t)
	h := domain.MatchViewHeader{}
	applyMatchHeaderScoreTimeline(&h, &domain.MatchMetaRaw{PairName: strPtr("Arena:Slayer")}, nil)
	if h.ScoreTimelineKind != "" {
		t.Errorf("règle absente -> %q, want vide", h.ScoreTimelineKind)
	}
	applyMatchHeaderScoreTimeline(&h, &domain.MatchMetaRaw{}, resolve)
	if h.ScoreTimelineKind != "" {
		t.Errorf("pair_name absent -> %q, want vide", h.ScoreTimelineKind)
	}
	applyMatchHeaderScoreTimeline(&h, &domain.MatchMetaRaw{PairName: strPtr("  ")}, resolve)
	if h.ScoreTimelineKind != "" {
		t.Errorf("libellé vide -> %q, want vide", h.ScoreTimelineKind)
	}
	applyMatchHeaderScoreTimeline(nil, &domain.MatchMetaRaw{PairName: strPtr("Arena:Slayer")}, resolve)
	applyMatchHeaderScoreTimeline(&h, nil, resolve)
}
