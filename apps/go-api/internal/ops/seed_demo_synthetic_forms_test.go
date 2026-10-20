// Package ops — seed_demo_synthetic_forms_test.go : GARDE-RAIL DE LA FORME DES DONNÉES DU
// CORPUS DE DÉMO SYNTHÉTIQUE.
//
// POURQUOI CE FICHIER EXISTE. Le corpus est écrit à la main, hors des chemins de
// production : une règle que le sync réel applique (forme d'un identifiant de match,
// chaîne LUSR d'une ligne de rang) ne s'impose pas à lui. Il a dérivé deux fois sans qu'un
// test ne tombe : des identifiants « demo-match-0001 » rendaient toute vue de match
// injoignable (404 `match_not_participant`), et des lignes LUSR en `arena` /
// `big_team_battle` s'affichaient en clés brutes dans la légende du graphe de carrière.
package ops

import (
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/sync/skill"
)

// TestSynthGroups_EgalentLesConstantesDeSync : les groupes dupliqués dans ops
// (seed_demo_synthetic.go) restent byte-identiques à ceux que le sync écrit.
func TestSynthGroups_EgalentLesConstantesDeSync(t *testing.T) {
	for _, c := range []struct{ name, got, want string }{
		{"ranked", synthGroupRanked, skill.PerfChainRanked},
		{"arena_slayer", synthGroupArenaSlayer, skill.LUSRChainArenaSlayer},
		{"btb", synthGroupBTB, skill.LUSRChainBTB},
		{"chaos", synthGroupChaos, skill.LUSRChainChaos},
	} {
		if c.got != c.want {
			t.Errorf("groupe %s : ops=%q, sync/skill=%q", c.name, c.got, c.want)
		}
	}
}

// TestSynthPlan_IdentifiantsEtChainesConformes : chaque match du plan porte un identifiant
// accepté par les routes /matches/{id} et un playlist_group que le sync réel écrirait
// (`ranked` pour le CSR, une des quatre chaînes LUSR sinon).
func TestSynthPlan_IdentifiantsEtChainesConformes(t *testing.T) {
	lusrChains := map[string]bool{
		skill.LUSRChainArenaSlayer:   true,
		skill.LUSRChainArenaObjectif: true,
		skill.LUSRChainBTB:           true,
		skill.LUSRChainChaos:         true,
	}
	plan := buildSynthPlan()
	if len(plan) == 0 {
		t.Fatal("plan synthétique vide")
	}
	seen := make(map[string]bool, len(plan))
	for _, m := range plan {
		if !title.IsValidMatchID(m.matchID) {
			t.Errorf("match %d : identifiant %q refusé par title.IsValidMatchID", m.idx, m.matchID)
		}
		if seen[m.matchID] {
			t.Errorf("identifiant de match en double : %q", m.matchID)
		}
		seen[m.matchID] = true
		switch {
		case m.pl.ranked && m.pl.group != skill.PerfChainRanked:
			t.Errorf("match %d : playlist classée en groupe %q, attendu %q", m.idx, m.pl.group, skill.PerfChainRanked)
		case !m.pl.ranked && !lusrChains[m.pl.group]:
			t.Errorf("match %d : playlist non classée en groupe %q, hors des chaînes LUSR", m.idx, m.pl.group)
		}
	}
}
