package analysis

// weapon_range_regroup_test.go — LES DEUX AJOUTS DU PROFIL D'ARMES DU FACE-À-FACE
// (plan .ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md, lot 1) : les extrêmes observés (D5) et
// le rekeyage vers le grain RÔLE (D1/D8).
//
// CE QUE CES TESTS ÉPINGLENT, ET QUI SE TROMPE EN SILENCE SANS EUX :
//
//  1. Min/Max se lisent sur la série TRIÉE. Les prendre au fil d'un parcours non trié rendrait
//     le premier et le dernier frag REÇUS — deux nombres plausibles et faux, que rien
//     n'attrape à l'œil sur un graphe où ils ne sont pas tracés.
//  2. Le rekeyage NE DOIT PAS muter l'entrée : le même corpus de frags sert la portée par rôle
//     ET les couvertures globales du même côté. Une mutation ferait lire des rôles là où
//     l'appelant croit lire des armes.
//  3. Une clé sans rôle est ÉCARTÉE et COMPTÉE. La ranger sous un seau fourre-tout
//     fabriquerait une ligne qui n'est pas une portée mais un mélange (D8).

import (
	"math"
	"testing"
)

// TestWeaponRangeMinMaxNominal : sur une série 1..8, min et max sont les deux bouts, et ils
// ENCADRENT strictement p10 et p90 — c'est tout l'intérêt de les publier à part.
func TestWeaponRangeMinMaxNominal(t *testing.T) {
	rows, _ := WeaponRangeAggregate(serie("br75", SideKiller, 8), WeaponRangeMinMeasured)
	if len(rows) != 1 {
		t.Fatalf("1 ligne attendue, obtenu %d", len(rows))
	}
	r := rows[0]
	if math.Abs(r.Min-1) > epsPortee || math.Abs(r.Max-8) > epsPortee {
		t.Fatalf("min/max = %v/%v, attendu 1/8 sur la série 1..8 (%+v)", r.Min, r.Max, r)
	}
	if !(r.Min < r.P10 && r.P90 < r.Max) {
		t.Fatalf("min <= p10 <= p90 <= max attendu, obtenu min=%v p10=%v p90=%v max=%v",
			r.Min, r.P10, r.P90, r.Max)
	}
}

// TestWeaponRangeMinMaxUneSeuleMesure : sur UN frag, min == max == médiane. Le seuil de
// publication est abaissé à 1 pour atteindre ce cas — c'est le seul endroit du dépôt qui le
// fait, et c'est délibéré : l'invariant porte sur le calcul, pas sur la doctrine de
// publication (D9 reste 8 partout ailleurs).
func TestWeaponRangeMinMaxUneSeuleMesure(t *testing.T) {
	rows, _ := WeaponRangeAggregate([]MeasuredKill{kilFrag("epee", SideKiller, 3.5, 0)}, 1)
	if len(rows) != 1 {
		t.Fatalf("1 ligne attendue, obtenu %d", len(rows))
	}
	r := rows[0]
	for _, c := range []struct {
		nom string
		got float64
	}{{"min", r.Min}, {"max", r.Max}, {"médiane", r.Median}, {"p10", r.P10}, {"p90", r.P90}} {
		if math.Abs(c.got-3.5) > epsPortee {
			t.Errorf("%s = %v, attendu 3.5 (mesure unique)", c.nom, c.got)
		}
	}
}

// TestWeaponRangeMinMaxSerieNonTriee : l'ordre d'ARRIVÉE des frags ne décide de rien. La
// série entre en désordre, min et max restent les extrêmes de la série.
func TestWeaponRangeMinMaxSerieNonTriee(t *testing.T) {
	var kills []MeasuredKill
	for _, d := range []float64{40, 2, 9, 15, 4, 30, 7, 12} {
		kills = append(kills, kilFrag("sniper", SideKiller, d, 0))
	}
	rows, _ := WeaponRangeAggregate(kills, WeaponRangeMinMeasured)
	if len(rows) != 1 {
		t.Fatalf("1 ligne attendue, obtenu %d", len(rows))
	}
	if math.Abs(rows[0].Min-2) > epsPortee || math.Abs(rows[0].Max-40) > epsPortee {
		t.Fatalf("min/max = %v/%v, attendu 2/40 (le premier et le dernier REÇUS sont 40 et 12 —"+
			" les lire au fil du parcours serait le bug)", rows[0].Min, rows[0].Max)
	}
}

// roleDeuxArmes est un `keyOf` de test : deux armes vers le même rôle, une arme sans rôle.
func roleDeuxArmes(weaponKey string) string {
	switch weaponKey {
	case "br75", "commando":
		return "precision"
	case "sniper":
		return "sniper"
	default:
		return "" // absente du registre : aucun rôle
	}
}

// TestRegroupMeasuredKillsFusionneVersLeRole : deux armes distinctes deviennent UNE ligne de
// rôle, et l'agrégat qui suit compte bien leurs frags ensemble — c'est la raison d'être du
// rekeyage (D1 : le grain publié est le rôle).
func TestRegroupMeasuredKillsFusionneVersLeRole(t *testing.T) {
	kills := append(serie("br75", SideKiller, 4), serie("commando", SideKiller, 4)...)
	out, dropped := RegroupMeasuredKills(kills, roleDeuxArmes)
	if dropped != 0 {
		t.Fatalf("aucun frag ne devait être écarté, obtenu %d", dropped)
	}
	if len(out) != 8 {
		t.Fatalf("8 frags attendus en sortie, obtenu %d", len(out))
	}
	for _, k := range out {
		if k.WeaponKey != "precision" {
			t.Fatalf("clé rekeyée attendue « precision », obtenu %q", k.WeaponKey)
		}
	}
	rows, sum := WeaponRangeAggregate(out, WeaponRangeMinMeasured)
	if len(rows) != 1 || rows[0].Measured != 8 {
		t.Fatalf("une ligne de rôle à 8 mesures attendue, obtenu %+v (résumé %+v)", rows, sum)
	}
}

// TestRegroupMeasuredKillsClefInconnueEcartee : une arme sans rôle sort du corpus et est
// COMPTÉE. Sans ce compte, l'appelant prendrait une absence pour un zéro.
func TestRegroupMeasuredKillsClefInconnueEcartee(t *testing.T) {
	kills := []MeasuredKill{
		kilFrag("br75", SideKiller, 12, 0),
		kilFrag("arme_hors_registre", SideKiller, 3, 0),
		kilFrag("sniper", SideVictim, 40, 0),
		kilFrag("", SideKiller, 1, 0),
	}
	out, dropped := RegroupMeasuredKills(kills, roleDeuxArmes)
	if dropped != 2 {
		t.Fatalf("2 frags écartés attendus (clé hors registre + clé vide), obtenu %d", dropped)
	}
	if len(out) != 2 {
		t.Fatalf("2 frags conservés attendus, obtenu %d : %+v", len(out), out)
	}
	if out[0].WeaponKey != "precision" || out[1].WeaponKey != "sniper" {
		t.Fatalf("clés rekeyées attendues precision/sniper, obtenu %q/%q",
			out[0].WeaponKey, out[1].WeaponKey)
	}
}

// TestRegroupMeasuredKillsConserveCoteDistanceDenivele : le rekeyage ne touche QUE la clé.
// Côté, distance, dénivelé et clé de frag traversent intacts — sans quoi la ventilation du
// dénivelé et l'appariement d'entame parleraient d'autres frags.
func TestRegroupMeasuredKillsConserveCoteDistanceDenivele(t *testing.T) {
	src := MeasuredKill{
		MatchID: "m1", KillerXUID: "x1", TimeMS: 4242,
		WeaponKey: "sniper", Side: SideVictim, DistanceM: 37.5, DeltaZ: -2.25,
	}
	out, dropped := RegroupMeasuredKills([]MeasuredKill{src}, roleDeuxArmes)
	if dropped != 0 || len(out) != 1 {
		t.Fatalf("un frag conservé attendu, obtenu %d (écartés %d)", len(out), dropped)
	}
	attendu := src
	attendu.WeaponKey = "sniper" // le rôle, homonyme de la clé d'arme ici
	if out[0] != attendu {
		t.Fatalf("seul WeaponKey devait changer : %+v -> %+v", src, out[0])
	}
}

// TestRegroupMeasuredKillsNeMutePasLEntree : PURETÉ. Le même corpus sert ensuite les
// couvertures globales par côté ; une mutation les ferait porter sur des rôles.
func TestRegroupMeasuredKillsNeMutePasLEntree(t *testing.T) {
	kills := []MeasuredKill{
		kilFrag("br75", SideKiller, 12, 0),
		kilFrag("commando", SideKiller, 9, 1),
	}
	avant := append([]MeasuredKill(nil), kills...)
	RegroupMeasuredKills(kills, roleDeuxArmes)
	for i := range avant {
		if kills[i] != avant[i] {
			t.Fatalf("entrée mutée au rang %d : %+v -> %+v", i, avant[i], kills[i])
		}
	}
}

// TestRegroupMeasuredKillsEntreeVideEtResolveurNil : les deux dégradations. Entrée vide → rien
// à écarter ; résolveur nil → aucune clé résoluble, donc tout est écarté et compté (un panic
// ne dirait rien de plus, et ferait tomber une page sur une absence de câblage).
func TestRegroupMeasuredKillsEntreeVideEtResolveurNil(t *testing.T) {
	out, dropped := RegroupMeasuredKills(nil, roleDeuxArmes)
	if len(out) != 0 || dropped != 0 {
		t.Fatalf("entrée vide : (0, 0) attendu, obtenu (%d, %d)", len(out), dropped)
	}
	kills := serie("br75", SideKiller, 3)
	out, dropped = RegroupMeasuredKills(kills, nil)
	if len(out) != 0 || dropped != 3 {
		t.Fatalf("résolveur nil : (0, 3) attendu, obtenu (%d, %d)", len(out), dropped)
	}
}
