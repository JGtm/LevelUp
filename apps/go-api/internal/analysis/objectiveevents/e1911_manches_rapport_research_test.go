//go:build research

package objectiveevents

// e1911_manches_rapport_research_test.go — INSTRUMENT 1.9.11, LE RAPPORT.
//
// Tout ce qui IMPRIME la mesure de `e1911_manches_mesure_research_test.go` : le detail par film,
// la table de synthese collable, et le croisement des deux sens de l'hypothese de l'utilisateur
// avec ses contre-exemples NOMMES.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// TestE1911DesignateurDeManche imprime, film par film, les designateurs ecrits, ce que la garde
// en retient, et le score des deux camps a la fin du temps reglementaire.
func TestE1911DesignateurDeManche(t *testing.T) {
	if cacheRoot() == "" {
		t.Skipf("%s absent : instrument saute", filmCacheEnv)
	}
	corpus := e1911Corpus(t)
	detail := len(corpus) <= e1911MaxDetail
	bilans := make([]e1911Bilan, 0, len(corpus))
	for _, m := range corpus {
		film, ok := newDiskFilm(t, m.ID)
		if !ok {
			t.Logf("%s : ECARTE (absent du cache)", m.ID)
			continue
		}
		b := e1911Mesure(m, StatRecords(film))
		bilans = append(bilans, b)
		if detail || len(b.Jetees) > 0 {
			e1911ImprimeFilm(t, b)
		}
	}
	e1911ImprimeSynthese(t, bilans)
	e1911ImprimeVerdict(t, bilans)
}

// e1911MaxDetail : au-dela de ce nombre de films, seuls ceux dont un designateur est JETE sont
// detailles — la synthese et le verdict restent complets.
const e1911MaxDetail = 25

// e1911ImprimeFilm imprime le detail d'un film.
func e1911ImprimeFilm(t *testing.T, b e1911Bilan) {
	t.Helper()
	t.Logf("=== %s | %s | duree %d s | reglementaire %d s | depassement %+d s | %d enregistrement(s) ===",
		b.M.ID, b.M.Variante, b.M.Duree, b.M.Regl, b.M.Depassement(), b.Records)
	t.Logf("  fenetre du film [%d, %d] ms ; fin du reglementaire : ancre FIN %d ms, ancre DEBUT %d ms",
		b.FilmT0, b.FilmT1, b.ReglFin, b.ReglDebut)
	t.Logf("  %-6s %8s %8s %8s %10s %10s %9s %9s %7s %7s %10s %6s %7s",
		"desig.", "records", "joueur", "equipe", "premier", "dernier",
		"courant", "finalise", "issues", "entree", "apres regl", "slots", "part %")
	for _, d := range b.Desig {
		t.Logf("  %-6d %8d %8d %8d %10d %10d %9d %9d %7d %7d %10d %6d %7d",
			d.Valeur, d.Records, d.Joueur, d.Equipe, d.Premier, d.Dernier,
			d.Courants, d.Finalise, d.Issues, d.Entree, d.Apres, d.Declarants, d.Part)
	}
	for _, d := range b.Desig {
		t.Logf("  histogramme 60 s du designateur %d : %v", d.Valeur, d.Tranches)
		t.Logf("  amas du designateur %d : %s", d.Valeur, e1911Amas(d.Blocs))
	}
	e1911ImprimePistes(t, b)
	t.Logf("  vus %v | RealRounds %v | JETES par la garde %v", b.Vues, b.Retenues, b.Jetees)
	t.Logf("  score de mode par camp : fin du reglementaire (ancre FIN) %v -> egalite %v ; (ancre DEBUT) %v -> egalite %v ; final au registre [%d %d]",
		b.ScoreFin, b.EgaliteFin(), b.ScoreDeb, b.EgaliteDebut(), b.M.S0, b.M.S1)
}

// e1911ImprimePistes imprime la piste de score de mode de chaque camp, point par point.
func e1911ImprimePistes(t *testing.T, b e1911Bilan) {
	t.Helper()
	for _, s := range b.Slots {
		parManche := b.Pistes[s]
		if len(parManche) == 0 {
			t.Logf("  piste du camp (slot %d) : MUETTE — le camp n'a jamais marque (score 0)", s)
			continue
		}
		for _, round := range sortedIntKeys(parManche) {
			t.Logf("  piste du camp (slot %d), manche %d : %s", s, round, e1911Piste(parManche[round]))
		}
	}
}

// e1911Piste rend une piste sous la forme `valeur@instant`, au plus [e1911MaxPoints] points.
func e1911Piste(pts []ScorePoint) string {
	const e1911MaxPoints = 40
	parts := make([]string, 0, e1911MaxPoints+1)
	for i, p := range pts {
		if i >= e1911MaxPoints {
			parts = append(parts, fmt.Sprintf("... (%d points)", len(pts)))
			break
		}
		parts = append(parts, fmt.Sprintf("%d@%d", p.Value, p.TimeMS))
	}
	return strings.Join(parts, " ")
}

// e1911ImprimeSynthese imprime la table d'ensemble, une ligne par film.
func e1911ImprimeSynthese(t *testing.T, bilans []e1911Bilan) {
	t.Helper()
	t.Logf("=== SYNTHESE (%d film(s)) ===", len(bilans))
	t.Logf("%-9s %-26s %6s %6s %7s %-12s %-12s %-12s %-12s %-8s %-9s %-9s %-9s %s",
		"film", "variante", "duree", "regl", "depass", "vus", "retenus", "sansgarde",
		"jetes", "n.jetes", "ecart", "egal.FIN", "egal.DEB", "final")
	for _, b := range bilans {
		t.Logf("%-9s %-26s %6d %6d %+7d %-12s %-12s %-12s %-12s %-8d %-9s %-9v %-9v [%d %d]",
			b.M.ID, b.M.Variante, b.M.Duree, b.M.Regl, b.M.Depassement(),
			e1911Liste(b.Vues), e1911Liste(b.Retenues), e1911Liste(b.SansGarde),
			e1911Liste(b.Jetees), e1911RecordsJetes(b), e1911Liste(b.Ecart),
			b.EgaliteFin(), b.EgaliteDebut(), b.M.S0, b.M.S1)
	}
	e1911ImprimeEcarts(t, bilans)
}

// e1911ImprimeEcarts isole les films que le RETRAIT de la garde changerait.
func e1911ImprimeEcarts(t *testing.T, bilans []e1911Bilan) {
	t.Helper()
	var lignes []string
	for _, b := range bilans {
		if len(b.Ecart) == 0 {
			continue
		}
		lignes = append(lignes, fmt.Sprintf("%s(%s: %s -> %s, %+ds, %d enr. ajoutes)",
			b.M.ID, b.M.Variante, e1911Liste(b.Retenues), e1911Liste(b.SansGarde),
			b.M.Depassement(), e1911RecordsDe(b, b.Ecart)))
	}
	t.Logf("=== ECART DU RETRAIT DE LA GARDE : %d film(s) sur %d changent ===",
		len(lignes), len(bilans))
	for _, l := range lignes {
		t.Logf("  %s", l)
	}
}

// e1911RecordsDe compte les enregistrements des designateurs nommes.
func e1911RecordsDe(b e1911Bilan, valeurs []int) int {
	in := map[int]bool{}
	for _, v := range valeurs {
		in[v] = true
	}
	n := 0
	for _, d := range b.Desig {
		if in[d.Valeur] {
			n += d.Records
		}
	}
	return n
}

// e1911Amas rend les amas d'un designateur sous la forme `n@[t0, t1]`.
func e1911Amas(blocs []e1911Bloc) string {
	parts := make([]string, 0, len(blocs))
	for _, b := range blocs {
		parts = append(parts, fmt.Sprintf("%d@[%d, %d]", b.Records, b.T0, b.T1))
	}
	return strings.Join(parts, " ")
}

// e1911Liste rend une liste de manches sous forme compacte.
func e1911Liste(v []int) string {
	if len(v) == 0 {
		return "-"
	}
	parts := make([]string, len(v))
	for i, x := range v {
		parts[i] = fmt.Sprint(x)
	}
	return strings.Join(parts, ",")
}

// e1911RecordsJetes compte les enregistrements que la garde jette sur ce film.
func e1911RecordsJetes(b e1911Bilan) int {
	jete := map[int]bool{}
	for _, v := range b.Jetees {
		jete[v] = true
	}
	n := 0
	for _, d := range b.Desig {
		if jete[d.Valeur] {
			n += d.Records
		}
	}
	return n
}

// e1911ImprimeVerdict croise les deux sens de l'hypothese de l'utilisateur et NOMME les
// contre-exemples de chaque sens.
//
// Le croisement se fait sur le designateur MATERIEL, pas sur n'importe quel designateur jete :
// un enregistrement isole (`bcb6d393` en porte UN a 7) est un ancrage fortuit, et le confondre
// avec les 148 de `fb1a1a72` noierait le signal. Le seuil de materialite est celui que la
// production emploie deja ([statMinRoundRecords]).
func e1911ImprimeVerdict(t *testing.T, bilans []e1911Bilan) {
	t.Helper()
	t.Logf("=== VERDICT (materialite : >= %d enregistrements, seuil de production) ===",
		statMinRoundRecords)
	var materiels, fortuits []string
	for _, b := range bilans {
		for _, d := range b.Desig {
			if !e1911EstJete(b, d.Valeur) {
				continue
			}
			s := fmt.Sprintf("%s:d%d(n=%d,apres=%d,%+ds,egal=%v)",
				b.M.ID, d.Valeur, d.Records, d.Apres, b.M.Depassement(), b.EgaliteFin())
			if d.Records >= statMinRoundRecords {
				materiels = append(materiels, s)
			} else {
				fortuits = append(fortuits, s)
			}
		}
	}
	t.Logf("  designateurs JETES et MATERIELS (%d) : %s", len(materiels), strings.Join(materiels, " "))
	t.Logf("  designateurs JETES et FORTUITS  (%d) : %s", len(fortuits), strings.Join(fortuits, " "))
	e1911Croise(t, bilans)
	e1911Populations(t, bilans)
}

// e1911EstJete dit si ce designateur est jete par la garde sur ce film.
func e1911EstJete(b e1911Bilan, valeur int) bool {
	for _, v := range b.Jetees {
		if v == valeur {
			return true
		}
	}
	return false
}

// e1911SecondeManche dit si le film ECRIT une seconde manche MATERIELLE, que la garde la
// retienne ou la jette : c'est le predicat de l'hypothese de l'utilisateur (« ca ressemble a une
// prolongation »), et il ne doit RIEN devoir a la garde qu'on juge.
func e1911SecondeManche(b e1911Bilan) bool {
	for _, d := range b.Desig {
		if d.Valeur > 0 && d.Records >= statMinRoundRecords {
			return true
		}
	}
	return false
}

// e1911Croise imprime les deux implications et leurs contre-exemples.
func e1911Croise(t *testing.T, bilans []e1911Bilan) {
	t.Helper()
	var sens1, sens2 []string
	for _, b := range bilans {
		mat, eg := e1911SecondeManche(b), b.EgaliteFin()
		s := fmt.Sprintf("%s(%+ds,%v,fin[%d %d])", b.M.ID, b.M.Depassement(), b.ScoreFin, b.M.S0, b.M.S1)
		switch {
		case mat && !eg:
			sens1 = append(sens1, s)
		case !mat && eg:
			sens2 = append(sens2, s)
		}
	}
	t.Logf("  CONTRE-EXEMPLES sens 1 (seconde manche materielle SANS egalite) : %d %v", len(sens1), sens1)
	t.Logf("  CONTRE-EXEMPLES sens 2 (egalite SANS seconde manche materielle) : %d %v", len(sens2), sens2)
	e1911Depassements(t, bilans)
	e1911Manches2(t, bilans)
}

// e1911Manches2 imprime, pour chaque film qui ECRIT une seconde manche materielle, le score des
// camps JUSTE AVANT son premier enregistrement — la mesure qui ne doit rien a l'alignement des
// horloges.
func e1911Manches2(t *testing.T, bilans []e1911Bilan) {
	t.Helper()
	t.Logf("  --- score au DEBUT de la seconde manche (lu au film, sans alignement d'horloge) ---")
	t.Logf("  %-9s %-26s %7s %6s %10s %-12s %-10s %s",
		"film", "variante", "depass", "desig.", "premier ms", "score avant", "egalite", "final")
	egaux, total := 0, 0
	for _, b := range bilans {
		if b.Manche2 < 0 {
			continue
		}
		total++
		eg := e1911Egalite(b.Manche2Score)
		if eg {
			egaux++
		}
		t.Logf("  %-9s %-26s %+7d %6d %10d %-12v %-10v [%d %d]",
			b.M.ID, b.M.Variante, b.M.Depassement(), b.Manche2, b.Manche2T,
			b.Manche2Score, eg, b.M.S0, b.M.S1)
	}
	t.Logf("  EGALITE au debut de la seconde manche : %d sur %d", egaux, total)
}

// e1911Depassements imprime les deux populations de DEPASSEMENT : celle des films qui ecrivent
// une seconde manche materielle, celle des autres. C'est le second sens de l'hypothese, teste
// sur la grandeur que la production connait deja (`analysis.ComputeOvertime`).
func e1911Depassements(t *testing.T, bilans []e1911Bilan) {
	t.Helper()
	var avec, sans []int
	for _, b := range bilans {
		if b.M.Regl <= 0 {
			continue
		}
		if e1911SecondeManche(b) {
			avec = append(avec, b.M.Depassement())
		} else {
			sans = append(sans, b.M.Depassement())
		}
	}
	sort.Ints(avec)
	sort.Ints(sans)
	t.Logf("  depassements AVEC seconde manche materielle (%d) : %v", len(avec), avec)
	t.Logf("  depassements SANS seconde manche materielle (%d, %d derniers) : %v",
		len(sans), e1911Queue, e1911NDerniers(sans, e1911Queue))
}

// e1911Queue borne la queue imprimee de la population « sans seconde manche ».
const e1911Queue = 20

// e1911NDerniers rend les n derniers elements d'une tranche triee.
func e1911NDerniers(v []int, n int) []int {
	if len(v) <= n {
		return v
	}
	return v[len(v)-n:]
}

// e1911Populations imprime les deux populations qui separent un ancrage fortuit d'un
// designateur reel : le nombre d'enregistrements des designateurs jetes, trie.
func e1911Populations(t *testing.T, bilans []e1911Bilan) {
	t.Helper()
	var n []int
	for _, b := range bilans {
		for _, d := range b.Desig {
			if e1911EstJete(b, d.Valeur) {
				n = append(n, d.Records)
			}
		}
	}
	sort.Ints(n)
	t.Logf("  comptes des designateurs JETES, tries (%d) : %v", len(n), n)
}
