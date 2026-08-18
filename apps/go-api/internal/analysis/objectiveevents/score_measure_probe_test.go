package objectiveevents

import (
	"fmt"
	"sort"
)

// score_measure_probe_test.go — la SONDE DE MODE de la phase 0 du lot A.
//
// # Pourquoi elle existe
//
// « Le sens d'un emplacement DEPEND DU MODE » (named.go) : la table nommee ne couvre que
// flag et zone, donc rien ne garantit que `comp 2 A / 2 B / 3 A` portent les frags, les morts
// et les assistances dans TOUS les modes. L'item A.0.3 du plan demande de l'ETABLIR pour
// Slayer, KOTH et Oddball.
//
// Quand aucun slot d'un film n'est apparie, deux conclusions sont possibles et il ne faut pas
// les confondre : soit le mode ne replique pas ces trois statistiques, soit il les replique
// AILLEURS. La sonde tranche : elle confronte le multi-ensemble des valeurs finales de CHAQUE
// emplacement de l'archetype aux trois multi-ensembles de l'oracle. Un emplacement qui
// reproduit exactement les frags des huit joueurs est une piste nommee ; aucun emplacement
// candidat est un negatif de mode, pas une ignorance.
//
// Elle ne s'arme que sur un film SANS aucun triplet trouve : sur les autres, la reponse est
// deja connue et le balayage ne serait que du bruit.

// writeProbe balaie les emplacements de l'archetype et ecrit ceux qui reproduisent les
// compteurs de l'oracle. `n` borne le nombre de slots joueurs attendus (8 au statborg) : la
// comparaison n'a de sens que si l'oracle a exactement ce nombre de lignes.
func writeProbe(m *measureRows, recs []StatRecord, or oracleMatch) {
	if len(or.Lines) != statPlayerSlots {
		m.row("probe", "non arme", fmt.Sprintf("%d lignes d'oracle pour %d slots",
			len(or.Lines), statPlayerSlots))
		return
	}
	want := map[string][]int64{
		"kills":   sortedInt64(fieldOf(or.Lines, "k")),
		"deaths":  sortedInt64(fieldOf(or.Lines, "d")),
		"assists": sortedInt64(fieldOf(or.Lines, "a")),
	}
	hits := 0
	for comp := 0; comp < statMaxComp; comp++ {
		for _, side := range []string{sideA, sideB} {
			got := lastBySlot(recs, statSlotKey{comp, side})
			if len(got) != statPlayerSlots {
				continue
			}
			vals := make([]int64, 0, len(got))
			for _, slot := range sortedSlots(got) {
				vals = append(vals, got[slot])
			}
			sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
			for name, w := range want {
				if sameInt64(vals, w) {
					m.row("probe", "candidat", comp, side, name)
					hits++
				}
			}
		}
	}
	m.row("probe", "total", hits)
}

// statPlayerSlots est le nombre de slots de joueur de l'archetype statborg (10..24 pairs).
const statPlayerSlots = 8

// roundMinSegment borne la longueur d'un segment pris pour une MANCHE. Un ancrage fortuit
// arrive isole ; une manche reelle emet plusieurs fois. Trois emissions est le plus petit
// seuil qui ecarte les paires fortuites sans jeter une manche courte.
const roundMinSegment = 3

// writeRounds teste l'hypothese que porte le NOM du composant : il s'appelle
// `statborg-current-round-value-stat-component` — « current round ». Le compteur pourrait donc
// repartir de zero a chaque MANCHE, alors que `team_0_score`/`team_1_score` portent le CUMUL du
// match. Le filtre de monotonie de [ScoreCurve] ne garde qu'une suite croissante : sur un match
// a deux manches il retient UNE manche et laisse croire a un sous-comptage.
//
// La mesure : decouper la suite d'emissions a chaque CHUTE de valeur, ne garder que les
// segments assez longs pour ne pas etre un ancrage fortuit, filtrer chacun par le meme critere
// de plus longue suite croissante que la production, et sommer leurs dernieres valeurs.
//
// C'est une sonde d'EXPLICATION : elle ne remplace pas le verdict A.0.1, elle dit si l'ecart
// mesure a une cause nommee.
func writeRounds(m *measureRows, recs []StatRecord, or oracleMatch) {
	sum := map[int]int64{}
	segs := map[int]int{}
	for _, slot := range []int{6, 8} {
		var cur []ScorePoint
		for _, p := range rawSerieOfSlot(recs, slot) {
			if len(cur) > 0 && p.Value < cur[len(cur)-1].Value {
				sum[slot], segs[slot] = addSegment(sum[slot], segs[slot], cur)
				cur = nil
			}
			cur = append(cur, p)
		}
		sum[slot], segs[slot] = addSegment(sum[slot], segs[slot], cur)
	}
	verdict := "ecart"
	if (sum[6] == int64(or.Team0) && sum[8] == int64(or.Team1)) ||
		(sum[6] == int64(or.Team1) && sum[8] == int64(or.Team0)) {
		verdict = "exact"
	}
	m.row("a01b", verdict, fmt.Sprintf("%d/%d", sum[6], sum[8]),
		fmt.Sprintf("%d/%d", or.Team0, or.Team1), segs[6], segs[8])
}

// addSegment ajoute la derniere valeur retenue d'un segment a la somme des manches.
func addSegment(sum int64, segs int, cur []ScorePoint) (int64, int) {
	if len(cur) < roundMinSegment {
		return sum, segs
	}
	kept := longestRun(cur, true)
	if len(kept) == 0 {
		return sum, segs
	}
	return sum + kept[len(kept)-1].Value, segs + 1
}

// rawSerieOfSlot rend les emissions du score de mode d'un slot, dans l'ordre du film, les
// valeurs negatives (ancrages parasites) jetees mais AUCUN filtre de monotonie applique.
func rawSerieOfSlot(recs []StatRecord, slot int) []ScorePoint {
	var out []ScorePoint
	for _, r := range recs {
		if r.Slot != slot {
			continue
		}
		if v, ok := r.Comps[modeScoreComp]; ok && v.A >= 0 {
			out = append(out, ScorePoint{TimeMS: r.TimeMS, Slot: slot, Value: v.A})
		}
	}
	return out
}

// fieldOf extrait un compteur de toutes les lignes de match.
func fieldOf(lines []PlayerLine, which string) []int64 {
	out := make([]int64, 0, len(lines))
	for _, l := range lines {
		switch which {
		case "k":
			out = append(out, int64(l.Kills))
		case "d":
			out = append(out, int64(l.Deaths))
		default:
			out = append(out, int64(l.Assists))
		}
	}
	return out
}

func sortedInt64(v []int64) []int64 {
	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	return v
}

func sameInt64(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
