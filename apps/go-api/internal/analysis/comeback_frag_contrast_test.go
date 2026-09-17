package analysis

import "testing"

// kills construit une timeline : chaque entrée (temps en s, équipe) est une frag.
func kills(spec ...[2]int) []KillEvent {
	out := make([]KillEvent, 0, len(spec))
	for _, s := range spec {
		out = append(out, KillEvent{TimeMS: int64(s[0]) * 1000, TeamID: s[1]})
	}
	return out
}

// steady : l'équipe « lead » ouvre le score à t=1 s, puis les deux équipes
// alternent jusqu'à t=599 s ; « lead » ajoute extra frags en fin de match.
// lead reste donc devant sur ~100 % d'un match de 600 s.
func steady(lead, pairs, extra int) []KillEvent {
	other := 1 - lead
	spec := [][2]int{{1, lead}}
	for i := 0; i < pairs; i++ {
		spec = append(spec, [2]int{10 + i*10, other}, [2]int{15 + i*10, lead})
	}
	for i := 0; i < extra; i++ {
		spec = append(spec, [2]int{590 + i, lead})
	}
	return kills(spec...)
}

func TestComputeFragContrastDominance(t *testing.T) {
	t.Parallel()
	const endMS = 600_000
	// Équipe 0 : 1 + 20 + 4 = 25 frags ; équipe 1 : 20 → 20/25 = 0.80 ≤ 0.85.
	dom0 := steady(0, 20, 4)
	// Équipe 0 : 1 + 20 + 2 = 23 ; équipe 1 : 20 → 20/23 = 0.87 > 0.85.
	thin0 := steady(0, 20, 2)
	// Équipe 1 mène les 50 premières secondes, puis l'équipe 0 dès 60 s et jusqu'à la
	// fin : équipe 0 devant ~ 90 % du temps, 13 frags contre 3.
	late := kills([2]int{0, 1}, [2]int{10, 1}, [2]int{20, 1}, [2]int{30, 0}, [2]int{40, 0},
		[2]int{50, 0}, [2]int{60, 0}, [2]int{100, 0}, [2]int{150, 0}, [2]int{200, 0},
		[2]int{250, 0}, [2]int{300, 0}, [2]int{350, 0}, [2]int{400, 0}, [2]int{450, 0}, [2]int{500, 0})
	// Équipe 0 finit à 13 contre 3 mais n'est devant que ~ 25 % du temps (remontée tardive).
	lateComeback := kills([2]int{0, 1}, [2]int{10, 1}, [2]int{20, 1}, [2]int{440, 0}, [2]int{445, 0},
		[2]int{450, 0}, [2]int{455, 0}, [2]int{460, 0}, [2]int{465, 0}, [2]int{470, 0},
		[2]int{475, 0}, [2]int{480, 0}, [2]int{485, 0}, [2]int{490, 0}, [2]int{495, 0}, [2]int{500, 0})
	small := steady(0, 3, 2) // 6 frags contre 3 : sous le volume minimal

	cases := []struct {
		name    string
		events  []KillEvent
		team    int
		outcome int
		want    int
	}{
		{"sabordage : equipe 0 perd au score, domine aux frags", dom0, 0, OutcomeLoss, DominanceFlagSabordage},
		{"abnegation : equipe 1 gagne au score, dominee aux frags", dom0, 1, OutcomeWin, DominanceFlagAbnegation},
		{"memes frags, resultat coherent : pas de badge (defaite equipe 1)", dom0, 1, OutcomeLoss, DominanceFlagNone},
		{"memes frags, resultat coherent : pas de badge (victoire equipe 0)", dom0, 0, OutcomeWin, DominanceFlagNone},
		{"ecart final sous 15 %", thin0, 0, OutcomeLoss, DominanceFlagNone},
		{"devant 90 % du temps apres un debut mene", late, 0, OutcomeLoss, DominanceFlagSabordage},
		{"gros ecart final mais remontee tardive", lateComeback, 0, OutcomeLoss, DominanceFlagNone},
		{"volume trop faible", small, 0, OutcomeLoss, DominanceFlagNone},
		{"egalite", dom0, 0, OutcomeTie, DominanceFlagNone},
		{"abandon", dom0, 0, OutcomeDNF, DominanceFlagNone},
		{"sans timeline", nil, 0, OutcomeLoss, DominanceFlagNone},
		{"equipe hors 0/1", dom0, 2, OutcomeLoss, DominanceFlagNone},
	}
	for _, c := range cases {
		if got := ComputeFragContrastDominance(c.events, endMS, c.team, c.outcome); got != c.want {
			t.Errorf("%s : flag = %d, attendu %d", c.name, got, c.want)
		}
	}
}

// TestComputeFragContrastDominance_EndFromDuration — la durée du match prolonge
// l'avance tenue après la dernière frag ; la dernière frag fait foi si la durée
// est absente (0).
func TestComputeFragContrastDominance_EndFromDuration(t *testing.T) {
	t.Parallel()
	// Équipe 1 devant de 0 à 100 s, équipe 0 devant de 110 s à la fin (11 frags contre 2).
	events := kills([2]int{0, 1}, [2]int{50, 1}, [2]int{100, 0}, [2]int{110, 0}, [2]int{111, 0},
		[2]int{112, 0}, [2]int{113, 0}, [2]int{114, 0}, [2]int{115, 0}, [2]int{116, 0},
		[2]int{117, 0}, [2]int{118, 0}, [2]int{119, 0}, [2]int{120, 0})
	if got := ComputeFragContrastDominance(events, 0, 0, OutcomeLoss); got != DominanceFlagNone {
		t.Errorf("fin = derniere frag (120 s), devant 8 %% du temps : flag %d, attendu 0", got)
	}
	if got := ComputeFragContrastDominance(events, 900_000, 0, OutcomeLoss); got != DominanceFlagSabordage {
		t.Errorf("fin = 900 s, devant 88 %% du temps : flag %d, attendu %d", got, DominanceFlagSabordage)
	}
}
