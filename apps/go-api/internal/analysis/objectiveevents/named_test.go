package objectiveevents

import "testing"

// named_test.go — la verite terrain de la lecture par COMPOSANT.
//
// Les tests adosses au cache film SKIPPENT proprement si le film est absent (meme
// convention qu'extract_test.go), donc verts en CI sans films. Les tests purs, eux,
// tournent partout : ce sont eux qui gardent les pieges deja payes.
//
// L'oracle est `personal_score_awards` (bases joueur), fige ici en dur pour garder le test
// sans dependance DuckDB. Il a ete regenere le 2026-08-02 et reproduit a l'identique le
// balayage de la session precedente.

// slotOwners fige la correspondance slot d'entite -> joueur suivi sur les films de
// reference.
//
// Pourquoi en dur plutot que deduit : l'identification par le nombre de frags n'est PAS
// unique sur ces films (sur 696a9d7c, quatre slots ont 15 frags et deux en ont 9). La
// correspondance vient de l'appariement des increments de +100 aux instants de frag
// (etat de l'art §16.2), qui, lui, ne laisse aucune collision.
var slotOwners = map[string]map[int]string{
	"696a9d7c": {22: "JGtm", 20: "Madina97294"},
	"1bc77d2e": {18: "JGtm", 24: "Chocoboflor", 16: "Madina97294"},
}

// oracle fige `personal_score_awards` pour les couples (film, joueur) utilises ici.
// Une recompense absente vaut ZERO, et ce zero compte : il interdit au decodeur
// d'inventer des evenements que l'API ne connait pas.
var oracle = map[string]map[string]map[string]int{
	"696a9d7c": {
		"JGtm":        {"killed_player": 9, "kill_assist": 7, "zone_captured": 7, "zone_secured": 2},
		"Madina97294": {"killed_player": 15, "kill_assist": 5, "zone_captured": 10, "zone_secured": 0},
	},
	"1bc77d2e": {
		"JGtm": {
			"killed_player": 24, "kill_assist": 2, "flag_captured": 1, "flag_returned": 2,
			"flag_stolen": 4, "runner_stopped": 2, "flag_capture_assist": 0,
		},
		"Chocoboflor": {
			"killed_player": 16, "kill_assist": 7, "flag_captured": 1, "flag_returned": 1,
			"flag_stolen": 0, "runner_stopped": 1, "flag_capture_assist": 2,
		},
		"Madina97294": {
			"killed_player": 11, "kill_assist": 13, "flag_captured": 0, "flag_returned": 0,
			"flag_stolen": 2, "runner_stopped": 1, "flag_capture_assist": 2,
		},
	},
}

// checkFilmAgainstOracle confronte les comptes decodes a l'oracle de l'API, recompense par
// recompense, pour chaque joueur suivi du film.
func checkFilmAgainstOracle(t *testing.T, film, objectiveType string) {
	t.Helper()
	src, ok := newDiskFilmSource(t, film)
	if !ok {
		t.Skipf("film %s absent du cache local", film)
	}
	counts := CountsBySlot(NamedEvents(src, objectiveType))
	if len(counts) == 0 {
		t.Fatalf("%s : aucun evenement nomme decode", film)
	}
	for slot, player := range slotOwners[film] {
		for award, want := range oracle[film][player] {
			if got := counts[slot][award]; got != want {
				t.Errorf("%s slot %d (%s) : %s = %d, attendu %d (personal_score_awards)",
					film, slot, player, award, got, want)
			}
		}
	}
}

// TestNamedEventsZoneGroundTruth — Strongholds `696a9d7c`. La lecture par composant doit
// rendre, pour les deux joueurs suivis, EXACTEMENT les comptes de l'API.
//
// Ce que ce test prouve et que la valeur du score ne pouvait pas prouver : `zone_captured`
// et `zone_secured` valent tous deux 25 points par action, donc aucune lecture par valeur
// ne pouvait les separer. Ils vivent dans deux emplacements distincts (comp 20 B et
// comp 21 A) et s'y lisent sans ambiguite.
func TestNamedEventsZoneGroundTruth(t *testing.T) {
	checkFilmAgainstOracle(t, "696a9d7c", ObjectiveTypeZone)
}

// TestNamedEventsCTFGroundTruth — CTF `1bc77d2e`, le film sur lequel l'ambiguite « l'un de
// trois noms » bloquait. `flag_returned`, `flag_stolen` et `runner_stopped` valent tous
// 25 points ; ils doivent ici se separer parfaitement.
//
// `flag_taken` est volontairement exclu de cette egalite — il ne compte pas la meme chose
// que la recompense de l'API (cf. TestNamedEventsFlagTakenNeverUndercounts).
func TestNamedEventsCTFGroundTruth(t *testing.T) {
	checkFilmAgainstOracle(t, "1bc77d2e", ObjectiveTypeFlag)
}

// TestNamedEventsZoneTotalsMatchAPI — le controle d'ensemble, sur les HUIT joueurs et non
// sur les seuls joueurs suivis : sur `696a9d7c`, comp 20 B totalise 61 et comp 21 A 16 ;
// leur somme vaut 77, exactement le total `zone_captures + zone_secures` de l'API.
//
// Il vaut d'etre garde parce qu'il ne depend d'AUCUNE correspondance slot -> joueur : meme
// si l'identite des slots derivait, ce total tiendrait.
func TestNamedEventsZoneTotalsMatchAPI(t *testing.T) {
	src, ok := newDiskFilmSource(t, "696a9d7c")
	if !ok {
		t.Skip("film 696a9d7c absent du cache local")
	}
	total := map[string]int{}
	for _, e := range NamedEvents(src, ObjectiveTypeZone) {
		total[e.Award]++
	}
	if total[AwardZoneCaptured] != 61 {
		t.Errorf("zone_captured total = %d, attendu 61", total[AwardZoneCaptured])
	}
	if total[AwardZoneSecured] != 16 {
		t.Errorf("zone_secured total = %d, attendu 16", total[AwardZoneSecured])
	}
	if sum := total[AwardZoneCaptured] + total[AwardZoneSecured]; sum != 77 {
		t.Errorf("zone_captured + zone_secured = %d, attendu 77 (total API du match)", sum)
	}
}

// TestNamedEventsFlagTakenNeverUndercounts — `flag_taken` est le seul emplacement ou le
// film et l'API divergent, et la divergence a UN SEUL SENS.
//
// Mesure (6 films CTF + les 3 joueurs de `1bc77d2e`, soit 8 couples) : le film compte
// parfois PLUS, JAMAIS MOINS — ecart total +11, pire ecart +5, zero contre-exemple. C'est
// la signature d'un compteur d'actions REELLES face a une recompense que le jeu ne verse
// pas a chaque fois : ramasser le drapeau au sol pendant sa course se compte plusieurs
// fois, ne se recompense pas plusieurs fois.
//
// Le test encode donc l'INEGALITE et non l'egalite. Un « film moins » serait fatal a cette
// lecture : on ne peut pas rater une action qu'on recompense.
func TestNamedEventsFlagTakenNeverUndercounts(t *testing.T) {
	src, ok := newDiskFilmSource(t, "1bc77d2e")
	if !ok {
		t.Skip("film 1bc77d2e absent du cache local")
	}
	counts := CountsBySlot(NamedEvents(src, ObjectiveTypeFlag))
	// `personal_score_awards` sur ce film : JGtm 1, Chocoboflor 3, Madina97294 4.
	for slot, want := range map[int]int{18: 1, 24: 3, 16: 4} {
		if got := counts[slot][AwardFlagTaken]; got < want {
			t.Errorf("slot %d : flag_taken = %d, INFERIEUR aux %d de l'API — "+
				"une action recompensee ne peut pas manquer au film", slot, got, want)
		}
	}
}

// TestNamedEventsCrossCheck — les emplacements REDONDANTS doivent reproduire exactement
// leur emplacement canonique. C'est un controle interne au film, sans oracle externe : le
// statborg duplique certaines statistiques, donc un desaccord signale un decodage qui a
// derape sur ce slot.
//
// Ce test a deja servi : il a demasque une valeur parasite a -115 sur comp 0 A qui faisait
// remonter le compteur en 116 evenements au lieu d'1.
func TestNamedEventsCrossCheck(t *testing.T) {
	for film, objectiveType := range map[string]string{
		"696a9d7c": ObjectiveTypeZone, "1bc77d2e": ObjectiveTypeFlag,
	} {
		src, ok := newDiskFilmSource(t, film)
		if !ok {
			continue
		}
		for slot, byAward := range CrossCheckNamedEvents(src, objectiveType) {
			for award, pair := range byAward {
				t.Errorf("%s slot %d : %s = %d sur l'emplacement canonique mais %d sur le "+
					"redondant", film, slot, award, pair[0], pair[1])
			}
		}
	}
}

// TestNamedEventsUnknownModeIsSilent — un mode sans table ne rend rien, et surtout
// n'invente aucun nom. KOTH et Oddball sont dans ce cas : leurs emplacements n'ont pas
// encore ete nommes.
func TestNamedEventsUnknownModeIsSilent(t *testing.T) {
	recs := []StatRecord{{TimeMS: 1000, Slot: 10, Comps: map[int]StatValue{20: {A: 0, B: 3}}}}
	for _, mode := range []string{ObjectiveTypeHill, ObjectiveTypeSkull, "", "slayer"} {
		if got := namedEventsFrom(recs, mode); got != nil {
			t.Errorf("mode %q : %d evenements rendus, attendu aucun", mode, len(got))
		}
	}
}

// TestNamedEventsIgnoresNegativeValues — non-regression du piege le plus couteux du
// fichier. Une emission parasite a valeur negative ne doit produire AUCUN evenement, et ne
// doit pas non plus permettre de recompter les unites deja acquises.
//
// Sans ce garde-fou, la suite ci-dessous (1 puis -115 puis 1) rendait 116 evenements.
func TestNamedEventsIgnoresNegativeValues(t *testing.T) {
	recs := []StatRecord{
		{TimeMS: 1000, Slot: 10, Comps: map[int]StatValue{23: {A: 1}}},
		{TimeMS: 2000, Slot: 10, Comps: map[int]StatValue{23: {A: -115}}},
		{TimeMS: 3000, Slot: 10, Comps: map[int]StatValue{23: {A: 1}}},
	}
	got := namedEventsFrom(recs, ObjectiveTypeFlag)
	if len(got) != 1 {
		t.Fatalf("%d evenements rendus, attendu 1 (la valeur negative est un parasite)", len(got))
	}
	if got[0].Award != AwardFlagReturned || got[0].TimeMS != 1000 {
		t.Errorf("evenement = %s a t=%d, attendu flag_returned a t=1000",
			got[0].Award, got[0].TimeMS)
	}
}

// TestNamedEventsRedundantSlotsDoNotDoubleCount — un emplacement redondant ne doit jamais
// emettre : sinon chaque frag serait compte deux fois (comp 2 A et comp 12 A portent tous
// deux le nombre de frags).
func TestNamedEventsRedundantSlotsDoNotDoubleCount(t *testing.T) {
	recs := []StatRecord{{
		TimeMS: 1000, Slot: 10,
		Comps: map[int]StatValue{2: {A: 3}, 12: {A: 3, B: 2}, 3: {A: 2}},
	}}
	counts := CountsBySlot(namedEventsFrom(recs, ObjectiveTypeZone))
	if got := counts[10][AwardKilledPlayer]; got != 3 {
		t.Errorf("killed_player = %d, attendu 3 (comp 12 A redouble comp 2 A)", got)
	}
	if got := counts[10][AwardKillAssist]; got != 2 {
		t.Errorf("kill_assist = %d, attendu 2 (comp 12 B redouble comp 3 A)", got)
	}
}

// TestNamedEventsRepeatedValueIsNotAnEvent — un composant est reemis des que l'UNE de ses
// deux valeurs bouge. Une reemission a valeur inchangee n'est donc pas un evenement.
func TestNamedEventsRepeatedValueIsNotAnEvent(t *testing.T) {
	recs := []StatRecord{
		{TimeMS: 1000, Slot: 10, Comps: map[int]StatValue{20: {B: 1}}},
		{TimeMS: 2000, Slot: 10, Comps: map[int]StatValue{20: {B: 1}}},
		{TimeMS: 3000, Slot: 10, Comps: map[int]StatValue{20: {B: 2}}},
	}
	got := namedEventsFrom(recs, ObjectiveTypeZone)
	if len(got) != 2 {
		t.Fatalf("%d evenements rendus, attendu 2 (la reemission a valeur egale n'en est pas un)",
			len(got))
	}
	if got[0].TimeMS != 1000 || got[1].TimeMS != 3000 {
		t.Errorf("instants = %d et %d, attendus 1000 et 3000", got[0].TimeMS, got[1].TimeMS)
	}
}

// TestKnownAwardsCoverBothModes — l'inventaire d'un mode dit ce que la lecture couvre.
// Il garde aussi le piege central du chantier : `comp 21 A` vaut `zone_secured` en zones et
// `flag_captured` en CTF, donc les deux inventaires DOIVENT differer.
func TestKnownAwardsCoverBothModes(t *testing.T) {
	flag, zone := KnownAwards(ObjectiveTypeFlag), KnownAwards(ObjectiveTypeZone)
	for _, name := range []string{AwardFlagReturned, AwardFlagStolen, AwardRunnerStopped} {
		if !flag[name] {
			t.Errorf("CTF : %s absent de l'inventaire", name)
		}
		if zone[name] {
			t.Errorf("zones : %s ne devrait pas y figurer", name)
		}
	}
	if !zone[AwardZoneCaptured] || !zone[AwardZoneSecured] {
		t.Error("zones : zone_captured / zone_secured absents de l'inventaire")
	}
	if len(KnownAwards(ObjectiveTypeHill)) != 0 {
		t.Error("hill : l'inventaire devrait etre vide, les emplacements ne sont pas nommes")
	}
}
