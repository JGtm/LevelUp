package sync_test

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/sync"
)

// TestSkillChainLiterals_NoDrift verrouille l'égalité byte-identique entre les 4
// valeurs de chaîne DUPLIQUÉES dans skillchain (interdiction d'importer sync =
// cycle) et les constantes canoniques sync.LUSRChain* (persistées dans
// match_skill_rank.playlist_group). Si l'une dérive, les rows seraient mal taguées
// silencieusement — ce test échoue d'abord.
//
// On compare via la SORTIE de ClassifyLUSRChain sur des pair_names représentatifs
// (les littéraux skillchain sont non exportés) : chaque chaîne canonique est
// atteinte au moins une fois.
func TestSkillChainLiterals_NoDrift(t *testing.T) {
	cases := []struct {
		pairName string
		want     string
	}{
		{"Arena:Slayer on Bazaar", sync.LUSRChainArenaSlayer},
		{"Arena:CTF on Recharge", sync.LUSRChainArenaObjectif},
		{"BTB:Slayer on Fragmentation", sync.LUSRChainBTB},
		{"Fiesta:Slayer on Bazaar", sync.LUSRChainChaos},
		{"Ranked:Slayer on Aquarius", ""}, // exclu
	}
	for _, tc := range cases {
		if got := skillchain.ClassifyLUSRChain(tc.pairName); got != tc.want {
			t.Errorf("ClassifyLUSRChain(%q) = %q, want sync const %q (dérive de littéral)",
				tc.pairName, got, tc.want)
		}
	}
}

// TestDispatcherEqualsClassifier prouve que le dispatcher sync.GetLUSRChain (câblé
// dans le TestMain) délègue exactement à skillchain.ClassifyLUSRChain sur un corpus
// — équivalence de bout en bout du seam.
func TestDispatcherEqualsClassifier(t *testing.T) {
	corpus := []string{
		"Ranked:Slayer on Aquarius", "Firefight:KotH on Argyle",
		"BTB:Slayer on Fragmentation", "BTB Heavies:CTF on Highpower",
		"Fiesta:Slayer on Bazaar", "Super Husky Raid:CTF on Pharaoh", "Castle Wars",
		"Infection:Slayer on Bazaar", "Rocket Hog Race:BTB on Highpower",
		"Rumble Pit:Slayer on Bazaar", "Arena:Slayer on Bazaar",
		"Arena:CTF on Recharge", "Arena:Strongholds on Streets", "BTB:CTF on Highpower", "",
	}
	for _, p := range corpus {
		if d, c := sync.GetLUSRChain(p), skillchain.ClassifyLUSRChain(p); d != c {
			t.Errorf("GetLUSRChain(%q)=%q != ClassifyLUSRChain=%q", p, d, c)
		}
	}
}
