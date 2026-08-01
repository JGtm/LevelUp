package objectiveevents

import "testing"

// jgtmQuota1bc77d2e : ce que `personal_score_awards` donne pour JGtm sur le match
// 1bc77d2e (CTF). Verite terrain figee ici pour garder le test pur (aucune dependance
// DuckDB), comme le fait deja extract_test.go pour les rosters.
func jgtmQuota1bc77d2e() []Award {
	return []Award{
		{Name: "killed_player", Category: "kill", Unit: 100, Count: 24},
		{Name: "flag_captured", Category: "objective", Unit: 300, Count: 1},
		{Name: "flag_stolen", Category: "objective", Unit: 25, Count: 4},
		{Name: "kill_assist", Category: "assist", Unit: 50, Count: 2},
		{Name: "runner_stopped", Category: "objective", Unit: 25, Count: 2},
		{Name: "flag_returned", Category: "objective", Unit: 25, Count: 2},
		{Name: "flag_taken", Category: "objective", Unit: 10, Count: 1},
	}
}

// TestLabelPersonalScoreReconciliesJGtm — la reconciliation qui ferme le nommage : les
// increments que le film porte pour JGtm (22 x 100, 6 x 25, 2 x 50, 2 x 125, 1 x 300,
// 1 x 10) doivent rendre EXACTEMENT les comptes de l'API une fois les +125 decomposes.
func TestLabelPersonalScoreReconciliesJGtm(t *testing.T) {
	// Suite d'increments observee sur le film, rejouee comme une courbe cumulee.
	deltas := []int64{}
	for i := 0; i < 22; i++ {
		deltas = append(deltas, 100)
	}
	for i := 0; i < 6; i++ {
		deltas = append(deltas, 25)
	}
	deltas = append(deltas, 50, 50, 125, 125, 300, 10)

	var pts []ScorePoint
	cum := int64(0)
	for i, d := range deltas {
		cum += d
		pts = append(pts, ScorePoint{TimeMS: 1000 * (i + 1), Slot: 18, Value: cum})
	}

	events := LabelPersonalScore(pts, map[int][]Award{18: jgtmQuota1bc77d2e()})
	s := SummarizeLabels(events)

	// 22 + 6 + 2 + 2x2 + 1 + 1 = 36 actions une fois les composes decomposes.
	if s.Total != 36 {
		t.Errorf("evenements produits = %d, attendu 36 (les +125 doivent se decomposer)", s.Total)
	}
	if s.Compound != 4 {
		t.Errorf("evenements issus d'un increment compose = %d, attendu 4", s.Compound)
	}
	if s.Unnamed != 0 {
		t.Errorf("evenements sans nom ni candidate = %d, attendu 0", s.Unnamed)
	}

	// Les valeurs uniques dans le quota se nomment sans ambiguite.
	for name, want := range map[string]int{
		"killed_player": 24, "flag_captured": 1, "kill_assist": 2, "flag_taken": 1,
	} {
		if got := s.ByAward[name]; got != want {
			t.Errorf("%s : %d evenements nommes, attendu %d", name, got, want)
		}
	}

	// Les six recompenses a 25 points partagent leur valeur : elles ne DOIVENT PAS etre
	// nommees au hasard, mais porter leurs candidates.
	amb := 0
	for _, e := range events {
		if e.Value != 25 {
			continue
		}
		amb++
		if e.Resolved() {
			t.Errorf("un increment de 25 a ete nomme %q alors que trois recompenses "+
				"du quota partagent cette valeur", e.Award)
		}
		if len(e.Candidates) != 3 {
			t.Errorf("candidates pour un +25 = %v, attendu 3 noms", e.Candidates)
		}
		if e.Category != "objective" {
			t.Errorf("categorie commune des candidates = %q, attendu objective", e.Category)
		}
	}
	if amb != 8 {
		t.Errorf("increments de 25 = %d, attendu 8 (4 flag_stolen + 2 returned + 2 runner_stopped)", amb)
	}

	if s.Resolved != 28 {
		t.Errorf("evenements nommes sans ambiguite = %d, attendu 28 sur 36", s.Resolved)
	}
}

// TestLabelPersonalScoreRefusesAmbiguousDecomposition — une decomposition n'est retenue
// que si elle est UNIQUE a nombre de parts minimal. Le cas est REEL : `zone_captured` vaut
// tantot 50 tantot 75, si bien qu'un increment de 125 admet 25+100 ET 50+75. Il doit rester
// brut plutot qu'etre tranche arbitrairement — se taire vaut mieux qu'un faux nom.
func TestLabelPersonalScoreRefusesAmbiguousDecomposition(t *testing.T) {
	quota := []Award{
		{Name: "zone_secured", Category: "objective", Unit: 25, Count: 4},
		{Name: "kill_assist", Category: "assist", Unit: 50, Count: 4},
		{Name: "zone_captured", Category: "objective", Unit: 75, Count: 2},
		{Name: "killed_player", Category: "kill", Unit: 100, Count: 4},
	}
	pts := []ScorePoint{{TimeMS: 1000, Slot: 10, Value: 125}}
	events := LabelPersonalScore(pts, map[int][]Award{10: quota})
	if len(events) != 1 {
		t.Fatalf("evenements = %d, attendu 1 (aucune decomposition ne doit etre choisie)", len(events))
	}
	if events[0].Value != 125 || events[0].Compound {
		t.Errorf("increment rendu %d (compose=%v), attendu 125 brut", events[0].Value, events[0].Compound)
	}
	if events[0].Resolved() {
		t.Errorf("increment nomme %q alors qu'aucune recompense ne vaut 125", events[0].Award)
	}
}

// TestLabelPersonalScoreDecomposesWhenUnique — le pendant du precedent : quand une seule
// decomposition existe a nombre de parts minimal, elle DOIT etre appliquee, sinon les
// increments composes du CTF (125 = 100 + 25) resteraient anonymes.
func TestLabelPersonalScoreDecomposesWhenUnique(t *testing.T) {
	quota := []Award{
		{Name: "flag_returned", Category: "objective", Unit: 25, Count: 2},
		{Name: "killed_player", Category: "kill", Unit: 100, Count: 2},
	}
	pts := []ScorePoint{{TimeMS: 1000, Slot: 10, Value: 125}}
	events := LabelPersonalScore(pts, map[int][]Award{10: quota})
	if len(events) != 2 {
		t.Fatalf("evenements = %d, attendu 2 (125 = 100 + 25)", len(events))
	}
	got := map[string]bool{events[0].Award: true, events[1].Award: true}
	if !got["killed_player"] || !got["flag_returned"] {
		t.Errorf("noms rendus %q et %q, attendu killed_player et flag_returned",
			events[0].Describe(), events[1].Describe())
	}
	for _, e := range events {
		if !e.Compound {
			t.Errorf("evenement %q non marque comme issu d'un increment compose", e.Describe())
		}
		if e.TimeMS != 1000 {
			t.Errorf("instant %d, attendu 1000 : les deux actions sont dans le meme paquet", e.TimeMS)
		}
	}
}

// TestLabelPersonalScoreHandlesNegativeAwards — le score personnel n'est pas monotone :
// self_destruction vaut -100. Un increment negatif doit se nommer comme les autres.
func TestLabelPersonalScoreHandlesNegativeAwards(t *testing.T) {
	quota := []Award{
		{Name: "killed_player", Category: "kill", Unit: 100, Count: 2},
		{Name: "self_destruction", Category: "penalty", Unit: -100, Count: 1},
	}
	pts := []ScorePoint{
		{TimeMS: 1000, Slot: 12, Value: 100},
		{TimeMS: 2000, Slot: 12, Value: 0},
		{TimeMS: 3000, Slot: 12, Value: 100},
	}
	events := LabelPersonalScore(pts, map[int][]Award{12: quota})
	if len(events) != 3 {
		t.Fatalf("evenements = %d, attendu 3", len(events))
	}
	if events[1].Award != "self_destruction" {
		t.Errorf("increment negatif nomme %q, attendu self_destruction", events[1].Describe())
	}
	if events[0].Award != "killed_player" || events[2].Award != "killed_player" {
		t.Errorf("increments positifs nommes %q et %q, attendu killed_player deux fois",
			events[0].Describe(), events[2].Describe())
	}
}
