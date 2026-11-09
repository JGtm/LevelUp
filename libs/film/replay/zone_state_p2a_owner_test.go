package replay

// zone_state_p2a_owner_test.go — CB.2a.2, second volet : LA VALEUR DU TAG 4 EST-ELLE
// L'EQUIPE PROPRIETAIRE ?
//
// LA QUESTION PRODUIT DE TOUT LE LOT. Une jauge de capture sans proprietaire ne teinte pas une
// zone : elle dit qu'on capture, pas POUR QUI. Le proprietaire, s'il existe cote film, ne peut
// venir que d'un canal enumerable — et le tag 4 est le seul du corpus.
//
// L'EQUIPE DU CAPTEUR VIENT DU ROSTER, JAMAIS DU FILM. `game-engine-team-mapping` (ti=0 i0) lit
// ses bits et les jette, sans hook (phase 1, decouverte 3) : l'equipe se lit donc dans
// `match_participants`, gele dans le corpus de la phase 2a. Le film fournit la valeur, la base
// fournit l'equipe, et la mesure les confronte.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// p2aTag4Neutre est la valeur « personne » du tag 4 : -1 sur 32 bits.
//
// ELLE N'EST PAS SUPPOSEE, ELLE EST MESUREE. La distribution des valeurs du tag 4 sur les deux
// Strongholds ne contient que TROIS valeurs pour les slots bavards — `0xFFFFFFFF`, `0x00000000`
// et `0x00000001` —, et les slots se repartissent en deux familles : ceux qui n'emettent que
// {0, 1} et ceux qui emettent les trois. C'est la forme exacte d'un proprietaire de zone :
// aucune equipe, equipe 0, equipe 1.
const p2aTag4Neutre = 0xFFFFFFFF

// p2aTag4Proprietaire teste si la valeur du tag 4 designe l'EQUIPE du capteur.
//
// LA MESURE EST FAITE PAR SLOT, et c'est ce qui la distingue d'un simple comptage. Agreger tous
// les slots melangerait des proprietes differentes (les deux familles ci-dessus) et noierait la
// correspondance : c'est ce qu'a fait la premiere passe, qui rendait `0xFFFFFFFF` dominant pour
// LES DEUX equipes et concluait a tort. La concordance est donc publiee slot par slot, avec et
// sans les emissions neutres.
func p2aTag4Proprietaire(t *testing.T, sb *strings.Builder, e p2aEntree,
	slots []p2aTag4Slot, app p2aAppariement,
) {
	t.Helper()
	teams := e.film.p2aTeams()
	series := p2aSeries(e.sc.scal, p2aTagU32)
	var concord, total, concordSansNeutre, totalSansNeutre int
	for _, s := range slots {
		parEquipe := map[int]map[uint64]int{}
		for _, c := range s.captures {
			team, ok := teams[c.xuid]
			if !ok {
				continue
			}
			v, ok := p2aValeurApres(series[s.slot], c.tMS)
			if !ok {
				continue
			}
			if parEquipe[team] == nil {
				parEquipe[team] = map[uint64]int{}
			}
			parEquipe[team][v]++
			total++
			if v == uint64(team) {
				concord++
			}
			if v != p2aTag4Neutre {
				totalSansNeutre++
				if v == uint64(team) {
					concordSansNeutre++
				}
			}
		}
		p2aLogSlotEquipe(t, sb, e, s, parEquipe)
	}
	p2aVerdictProprietaire(t, sb, e, [4]int{concord, total, concordSansNeutre, totalSansNeutre})
	p2aConteste(t, sb, e, app)
}

// p2aLogSlotEquipe publie, pour un slot, la valeur dominante du tag 4 apres les captures de
// CHAQUE equipe. Deux equipes, deux valeurs differentes : c'est la signature attendue.
func p2aLogSlotEquipe(t *testing.T, sb *strings.Builder, e p2aEntree, s p2aTag4Slot,
	parEquipe map[int]map[uint64]int,
) {
	t.Helper()
	teams := make([]int, 0, len(parEquipe))
	for team := range parEquipe {
		teams = append(teams, team)
	}
	sort.Ints(teams)
	for _, team := range teams {
		best, bestN, tot := uint64(0), -1, 0
		for v, n := range parEquipe[team] {
			tot += n
			if n > bestN {
				best, bestN = v, n
			}
		}
		t.Logf("    slot %-5d zone %d · captures de l'equipe %d : valeur dominante 0x%08X"+
			" (%d/%d) · toutes valeurs %s", s.slot, s.zone, team, best, bestN, tot,
			p2aValeursDe(parEquipe[team]))
		fmt.Fprintf(sb, "a2_slot_equipe\t%s\t%d\t%d\t%d\t0x%08X\t%d\t%d\n", e.short, s.slot,
			s.zone, team, best, bestN, tot)
	}
}

// p2aValeursDe rend la distribution d'une carte valeur -> compte, triee.
func p2aValeursDe(m map[uint64]int) string {
	vals := make([]uint64, 0, len(m))
	for v := range m {
		vals = append(vals, v)
	}
	sort.Slice(vals, func(i, j int) bool { return m[vals[i]] > m[vals[j]] })
	var parts []string
	for _, v := range vals {
		parts = append(parts, fmt.Sprintf("0x%08X x%d", v, m[v]))
	}
	return strings.Join(parts, " ")
}

// p2aVerdictProprietaire tranche la clause de la valeur : la valeur du tag 4 EST-ELLE l'index
// d'equipe du capteur ?
func p2aVerdictProprietaire(t *testing.T, sb *strings.Builder, e p2aEntree, n [4]int) {
	t.Helper()
	if n[1] == 0 {
		t.Logf("  PROPRIETAIRE : aucune capture ne porte de valeur de tag 4 dans la fenetre — NON MESURABLE")
		fmt.Fprintf(sb, "# a2_prop %s : non mesurable (aucune valeur post-capture)\n", e.short)
		return
	}
	part, partSN := p2aRate(n[0], n[1]), p2aRate(n[2], n[3])
	v := "NON ETABLI"
	if partSN >= p2aSeuilProprietaire {
		v = "PROPRIETAIRE"
	}
	t.Logf("  VERDICT CB.2a.2 (valeur = index d'equipe du capteur) : %d/%d = %.1f %% toutes"+
		" emissions · %d/%d = %.1f %% hors emissions neutres (0x%08X) — seuil %.0f %% : %s",
		n[0], n[1], 100*part, n[2], n[3], 100*partSN, uint32(p2aTag4Neutre),
		100*p2aSeuilProprietaire, v)
	fmt.Fprintf(sb, "a2_prop_verdict\t%s\t%d\t%d\t%.4f\t%d\t%d\t%.4f\t%s\n", e.short, n[0], n[1],
		part, n[2], n[3], partSN, v)
}

// p2aValeurApres rend la valeur du tag 4 au premier echantillon a l'instant de la capture ou
// APRES, dans la fenetre. C'est le sens du volet : la propriete change QUAND la zone est prise.
func p2aValeurApres(ss []p2aEch, tMS int) (uint64, bool) {
	i := sort.Search(len(ss), func(k int) bool { return ss[k].tMS >= tMS })
	if i >= len(ss) || ss[i].tMS > tMS+p2aFenetreMS {
		return 0, false
	}
	return ss[i].pay, true
}

// p2aConteste cherche la piste « conteste » : la valeur du tag 4 pendant les rampes NON abouties
// (aucune capture dans la fenetre du sommet) differe-t-elle de celle des rampes abouties ?
func p2aConteste(t *testing.T, sb *strings.Builder, e p2aEntree, app p2aAppariement) {
	t.Helper()
	series := p2aSeries(e.sc.scal, p2aTagU32)
	var capT []int
	for _, c := range app.captures {
		capT = append(capT, c.tMS)
	}
	sort.Ints(capT)
	abouties, sechees := map[uint64]int{}, map[uint64]int{}
	for _, r := range p2aRampes(e.sc) {
		v, ok := p2aValeurApres(series[r.slot], r.tMax)
		if !ok {
			continue
		}
		if p2aProche(capT, r.tMax) {
			abouties[v]++
			continue
		}
		sechees[v]++
	}
	t.Logf("  CONTESTE : %d valeurs distinctes apres une rampe ABOUTIE, %d apres une rampe SECHE"+
		" (%d vs %d rampes)", len(abouties), len(sechees), p2aSomme(abouties), p2aSomme(sechees))
	for _, v := range p2aValeursTriees(abouties, sechees) {
		t.Logf("    valeur 0x%08X : %d abouties · %d seches", v, abouties[v], sechees[v])
		fmt.Fprintf(sb, "a2_conteste\t%s\t0x%08X\t%d\t%d\n", e.short, v, abouties[v], sechees[v])
	}
}

func p2aSomme(m map[uint64]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func p2aValeursTriees(a, b map[uint64]int) []uint64 {
	vus := map[uint64]bool{}
	for v := range a {
		vus[v] = true
	}
	for v := range b {
		vus[v] = true
	}
	out := make([]uint64, 0, len(vus))
	for v := range vus {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
