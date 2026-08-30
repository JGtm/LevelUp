package sync_test

import (
	"testing"

	halo5 "levelup/go-api/internal/games/halo_5"
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

// objectiveFamilyCorpus : pair_names couvrant les deux familles, en forme sociale
// et en forme classée (mêmes sous-modes, préfixes différents).
var objectiveFamilyCorpus = []string{
	"Ranked:CTF on Recharge", "Ranked:Oddball on Streets", "Ranked:Strongholds on Live Fire",
	"Ranked:King of the Hill on Solitude", "Ranked:Slayer on Aquarius",
	"Ranked:CTF 3 Captures on Argyle", "Arena:CTF on Recharge", "Arena:Slayer on Bazaar",
	"Arena:Total Control on Argyle", "BTB:CTF on Highpower", "Truc Inconnu XYZ", "",
}

// TestObjectiveFamilyDispatcherEqualsClassifier prouve que le seam de famille
// (sync.IsObjectiveFamilyForTitle, câblé dans le TestMain) délègue exactement à
// skillchain.IsObjectiveSubMode — miroir de TestDispatcherEqualsClassifier pour la
// chaîne de performance classée.
func TestObjectiveFamilyDispatcherEqualsClassifier(t *testing.T) {
	for _, p := range objectiveFamilyCorpus {
		if d, c := sync.IsObjectiveFamilyForTitle("", p), skillchain.IsObjectiveSubMode(p); d != c {
			t.Errorf("IsObjectiveFamilyForTitle(%q)=%v != IsObjectiveSubMode=%v", p, d, c)
		}
	}
}

// TestPerfChainFamilyMatchesLUSRFamily verrouille l'INVARIANT que la liste
// partagée existe pour garantir : un sous-mode classé en famille objectif côté
// social (chaîne LUSR arena_objectif) l'est aussi côté classé (ranked_objectif),
// et réciproquement. Deux copies de la liste divergeraient ici en premier.
func TestPerfChainFamilyMatchesLUSRFamily(t *testing.T) {
	for _, p := range objectiveFamilyCorpus {
		// Côté social : la chaîne LUSR du MÊME sous-mode, avec un préfixe Assassin
		// (le préfixe Ranked est exclu du LUSR par construction → "").
		wantObjectif := skillchain.IsObjectiveSubMode(p)

		gotRanked := sync.GetPerformanceChain("", p, true, false)
		switch {
		case wantObjectif && gotRanked != sync.PerfChainRankedObjectif:
			t.Errorf("GetPerformanceChain(%q, ranked) = %q, want %q", p, gotRanked, sync.PerfChainRankedObjectif)
		case !wantObjectif && gotRanked != sync.PerfChainRankedSlayer:
			t.Errorf("GetPerformanceChain(%q, ranked) = %q, want %q", p, gotRanked, sync.PerfChainRankedSlayer)
		}
	}
}

// TestPerfChainNeverEmitsLegacyRanked verrouille la fin de vie de la chaîne
// "ranked" comme sortie de classification : elle ne subsiste que comme valeur
// STOCKÉE (recalculée par le skip de chaîne de performance.go). Aucun combo
// d'entrée ne doit plus la produire.
func TestPerfChainNeverEmitsLegacyRanked(t *testing.T) {
	for _, p := range append(objectiveFamilyCorpus, "Firefight:KotH on Argyle", "Fiesta:Slayer on Bazaar") {
		for _, flags := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
			if got := sync.GetPerformanceChain("", p, flags[0], flags[1]); got == sync.PerfChainRanked {
				t.Errorf("GetPerformanceChain(%q, ranked=%v, ff=%v) a émis la chaîne legacy %q",
					p, flags[0], flags[1], sync.PerfChainRanked)
			}
		}
	}
}

// TestPerfChainHalo5RankedFallsBackToSlayer couvre le titre SANS notion de famille :
// Halo 5 n'a pas de pair_name (games/halo_5/lusr_chain.go), son classifier de
// famille dédié répond toujours false → tout son classé va en ranked_slayer, y
// compris si un pair_name Infinite lui était passé par erreur (le classifier
// title-aware prime sur le défaut Infinite).
func TestPerfChainHalo5RankedFallsBackToSlayer(t *testing.T) {
	sync.SetObjectiveFamilyClassifierForTitle(halo5.TitleSlug, halo5.IsObjectiveSubMode)

	for _, p := range []string{"", "Ranked:CTF on Recharge", "Ranked:Slayer on Aquarius", "Arena:Oddball"} {
		if got := sync.GetPerformanceChain(halo5.TitleSlug, p, true, false); got != sync.PerfChainRankedSlayer {
			t.Errorf("GetPerformanceChain(halo_5, %q, ranked) = %q, want %q", p, got, sync.PerfChainRankedSlayer)
		}
	}
	// Le titre par défaut n'est PAS affecté par le classifier h5.
	if got := sync.GetPerformanceChain("", "Ranked:CTF on Recharge", true, false); got != sync.PerfChainRankedObjectif {
		t.Errorf("titre par défaut contaminé par le classifier h5 : %q", got)
	}
}
