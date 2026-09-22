//go:build research

package grammar

// mouvement_5_19_tableaux_research_test.go — LES TABLEAUX DE LA DIFFERENTIELLE (lot 5.19.1).
//
// Deplacement PUR depuis `mouvement_5_19_differentielle_research_test.go`, qui depassait le
// seuil de 500 lignes du ratchet de taille (`archlint/film_file_size_test.go`) une fois la
// differentielle interne (g) ecrite. La COLLECTE reste dans le fichier d origine ; ce
// fichier-ci ne fait qu ECRIRE ce qu elle a compte. Aucune ligne de mesure n y est.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// d519Publier ecrit le tableau de la differentielle.
func d519Publier(t *testing.T, b *d519Bilan) {
	t.Helper()
	t.Logf("(0) GATE REPRODUIT — %d paquets (%d non localises) · FERMES a reste NUL %d · "+
		"REJET de slot inconnu %d · debordements %d · autres %d",
		b.paquets, b.nonLocalise, b.fermes, b.rejets, b.debordements, b.autres)
	for _, k := range d519Cles(b.causes) {
		t.Logf("    %-52s %7d", k, b.causes[k])
	}
	t.Logf("    records lus : %d dans les fautifs (%d sans record) · %d dans les fermes (%d sans)",
		b.rejRecords, b.rejSansRecord, b.fermeRecords, b.fermeSansRe)

	t.Logf("(a) ARCHETYPE DU DERNIER RECORD AVANT LE REJET :")
	d519HistTI(t, b.rejTI)
	t.Logf("    temoin — archetype du dernier record des paquets FERMES :")
	d519HistTI(t, b.fermeTI)

	t.Logf("(b) DERNIER RECORD AVANT LE REJET (dernier composant a CORPS NON VIDE) :")
	d519Top(t, b.rejDernier, 25)
	t.Logf("    temoin — dernier record des paquets FERMES :")
	d519Top(t, b.fermeDernier, 25)

	t.Logf("(c) MASQUE COMPLET DU DERNIER RECORD AVANT LE REJET :")
	d519Top(t, b.rejMasque, 30)
	t.Logf("    temoin — masque du dernier record des paquets FERMES :")
	d519Top(t, b.fermeMasque, 15)

	t.Logf("(e) LE RECORD D AVANT (n-2) :")
	d519Top(t, b.rejAvant1, 15)
	t.Logf("    ET CELUI D AVANT (n-3) :")
	d519Top(t, b.rejAvant2, 15)
	t.Logf("    CHAINE DES TROIS ARCHETYPES :")
	d519Top(t, b.rejChaine, 15)

	d519Suspects(t, b)
	d519Dose(t, b)
	d519Interne(t, b)
}

// d519Interne publie (g) : dans les MEMES paquets fautifs, le dernier record (celui dont la
// lecture laisse le curseur faux) contre tous ceux qui le precedent (lus juste, puisque le
// record suivant s est decode). Aucun confondant de densite, de carte ni de film.
func d519Interne(t *testing.T, b *d519Bilan) {
	t.Helper()
	t.Logf("(g) DIFFERENTIELLE INTERNE — %d derniers records contre %d records precedents, "+
		"dans les MEMES paquets fautifs :", b.dernN, b.avantN)
	t.Logf("    TAUX DE FAUTE D UNE CLASSE = dernier / (dernier + precedents) : la part des " +
		"lectures de cette classe apres lesquelles le curseur est faux.")
	l := d519Taux(b.dernMasq, b.avantMasq, 20)
	sort.Slice(l, func(i, j int) bool { return l[i].dern > l[j].dern })
	t.Logf("    PAR VOLUME DE FAUTES :")
	d519Ecrire(t, l, 25)
	sort.Slice(l, func(i, j int) bool { return l[i].taux > l[j].taux })
	t.Logf("    PAR TAUX DE FAUTE :")
	d519Ecrire(t, l, 25)
	t.Logf("    LARGEURS — taux de faute par composant ET par largeur lue :")
	d519LargInterne(t, b)
}

// d519Ligne est une classe avec son taux de faute.
type d519Ligne struct {
	nom         string
	dern, avant int
	taux        float64
}

// d519Taux calcule le taux de faute de chaque classe portant au moins `seuil` dernieres lectures.
func d519Taux(dern, avant map[string]int, seuil int) []d519Ligne {
	var l []d519Ligne
	for k, v := range dern {
		if v < seuil {
			continue
		}
		l = append(l, d519Ligne{k, v, avant[k], m533bPart(v, v+avant[k])})
	}
	return l
}

// d519Ecrire publie les `n` premieres lignes d une table de taux.
func d519Ecrire(t *testing.T, l []d519Ligne, n int) {
	t.Helper()
	for i, e := range l {
		if i >= n {
			break
		}
		t.Logf("      %6.1f %% de faute · dernier %6d · precedents %7d  %s",
			e.taux, e.dern, e.avant, e.nom)
	}
}

// d519LargInterne compare, composant par composant et largeur par largeur, ce que lit le dernier
// record et ce que lisent ceux qui le precedent.
func d519LargInterne(t *testing.T, b *d519Bilan) {
	t.Helper()
	var l []d519Ligne
	for nom, m := range b.dernLarg {
		for w, n := range m {
			if n < 20 {
				continue
			}
			a := b.avantLarg[nom][w]
			l = append(l, d519Ligne{fmt.Sprintf("%-70s %4d bits", nom, w), n, a,
				m533bPart(n, n+a)})
		}
	}
	sort.Slice(l, func(i, j int) bool { return l[i].taux > l[j].taux })
	d519Ecrire(t, l, 30)
	sort.Slice(l, func(i, j int) bool { return l[i].dern > l[j].dern })
	t.Logf("    ... et par VOLUME :")
	d519Ecrire(t, l, 20)
}

// d519Dose publie (f) : la fermeture d un paquet contre le NOMBRE de deltas de bipede qu il
// porte. Une loi geometrique en ce nombre dirait que la faute est PAR RECORD, et designerait
// comme temoin le paquet qui n en porte qu UN et qui ne ferme pas.
func d519Dose(t *testing.T, b *d519Bilan) {
	t.Helper()
	t.Logf("(f) DOSE — fermeture contre nombre de deltas de bipede (ti=%d) dans le paquet :",
		BipedTypeIndex)
	t.Logf("    %6s %10s %10s %10s", "n ti=35", "fermes", "fautifs", "taux ferme")
	for _, k := range d519ClesInt(b.doseRej, b.doseFerme) {
		if k > 24 {
			continue
		}
		t.Logf("    %6d %10d %10d %9.1f %%", k, b.doseFerme[k], b.doseRej[k],
			m533bPart(b.doseFerme[k], b.doseFerme[k]+b.doseRej[k]))
	}
	t.Logf("    TEMOIN — paquets a UN SEUL delta de bipede : masque de ce record, FAUTIFS :")
	d519Top(t, b.unRejMasq, 12)
	t.Logf("    TEMOIN — les memes, FERMES :")
	d519Top(t, b.unFermeMasq, 12)
}

// d519Suspects publie la table (d) : ce qui distingue un paquet fautif d un paquet ferme.
func d519Suspects(t *testing.T, b *d519Bilan) {
	t.Helper()
	t.Logf("(d) TABLE DES SUSPECTS — presence PAR PAQUET (%d fautifs, %d fermes) :",
		b.rejets, b.fermes)
	t.Logf("    %-46s %8s %7s %8s %7s", "composant", "fautifs", "%", "fermes", "%")
	type ligne struct {
		nom        string
		rej, ferme int
	}
	l := make([]ligne, 0, len(b.presRej))
	for k, v := range b.presRej {
		l = append(l, ligne{k, v, b.presFerme[k]})
	}
	for k, v := range b.presFerme {
		if _, ok := b.presRej[k]; !ok {
			l = append(l, ligne{k, 0, v})
		}
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].rej != l[j].rej {
			return l[i].rej > l[j].rej
		}
		return l[i].nom < l[j].nom
	})
	for i, e := range l {
		if i >= 60 {
			t.Logf("    ... %d composants de plus", len(l)-60)
			break
		}
		t.Logf("    %-46s %8d %6.1f%% %8d %6.1f%%", e.nom, e.rej,
			m533bPart(e.rej, b.rejets), e.ferme, m533bPart(e.ferme, b.fermes))
	}
	t.Logf("    ARCHETYPES — presence par paquet (fautifs / fermes) :")
	for _, k := range d519ClesInt(b.tiPresRej, b.tiPresFerme) {
		t.Logf("      ti=%-3d %8d %6.1f%% %8d %6.1f%%", k, b.tiPresRej[k],
			m533bPart(b.tiPresRej[k], b.rejets), b.tiPresFerme[k],
			m533bPart(b.tiPresFerme[k], b.fermes))
	}
	t.Logf("(d') LARGEURS LUES PAR COMPOSANT dans le DERNIER record — fautifs contre fermes ; " +
		"une largeur que seuls les fautifs portent est un suspect nomme :")
	noms := map[string]bool{}
	for k := range b.largRej {
		noms[k] = true
	}
	for k := range b.largFerme {
		noms[k] = true
	}
	ordre := make([]string, 0, len(noms))
	for k := range noms {
		ordre = append(ordre, k)
	}
	sort.Slice(ordre, func(i, j int) bool {
		return d519Somme(b.largRej[ordre[i]]) > d519Somme(b.largRej[ordre[j]])
	})
	for i, nom := range ordre {
		if i >= 40 {
			t.Logf("    ... %d composants de plus", len(ordre)-40)
			break
		}
		t.Logf("    %-46s fautifs %s", nom, d519Largs(b.largRej[nom]))
		t.Logf("    %-46s fermes  %s", "", d519Largs(b.largFerme[nom]))
		if seules := d519Seules(b.largRej[nom], b.largFerme[nom]); seules != "" {
			t.Logf("    %-46s  -> LARGEURS QUE SEULS LES FAUTIFS PORTENT : %s", "", seules)
		}
	}
}

// d519Largs rend « largeur x compte » par largeur croissante, au plus huit classes.
func d519Largs(m map[int]int) string {
	if len(m) == 0 {
		return "(jamais lu)"
	}
	cles := make([]int, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var sb []string
	for i, k := range cles {
		if i >= 8 {
			sb = append(sb, fmt.Sprintf("... %d classes", len(cles)-8))
			break
		}
		sb = append(sb, fmt.Sprintf("%d bits x%d", k, m[k]))
	}
	return strings.Join(sb, " · ")
}

// d519Seules rend les largeurs presentes chez les fautifs et absentes chez les fermes.
func d519Seules(rej, ferme map[int]int) string {
	cles := make([]int, 0, len(rej))
	for k := range rej {
		if _, ok := ferme[k]; !ok {
			cles = append(cles, k)
		}
	}
	if len(cles) == 0 || len(ferme) == 0 {
		return ""
	}
	sort.Ints(cles)
	var sb []string
	for i, k := range cles {
		if i >= 10 {
			sb = append(sb, fmt.Sprintf("... %d de plus", len(cles)-10))
			break
		}
		sb = append(sb, fmt.Sprintf("%d bits x%d", k, rej[k]))
	}
	return strings.Join(sb, " · ")
}

// d519Somme additionne les comptes d une ventilation de largeurs.
func d519Somme(m map[int]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

// d519Top publie les `n` classes les plus peuplees d une ventilation.
func d519Top(t *testing.T, m map[string]int, n int) {
	t.Helper()
	type kv struct {
		k string
		v int
	}
	l, tot := make([]kv, 0, len(m)), 0
	for k, v := range m {
		l = append(l, kv{k, v})
		tot += v
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].v != l[j].v {
			return l[i].v > l[j].v
		}
		return l[i].k < l[j].k
	})
	for i := 0; i < n && i < len(l); i++ {
		t.Logf("    %7d (%5.1f %%)  %s", l[i].v, m533bPart(l[i].v, tot), l[i].k)
	}
	t.Logf("    TOTAL %d · %d classes", tot, len(l))
}

// d519HistTI publie une ventilation par archetype, par compte decroissant.
func d519HistTI(t *testing.T, m map[int]int) {
	t.Helper()
	cles, tot := make([]int, 0, len(m)), 0
	for k, v := range m {
		cles = append(cles, k)
		tot += v
	}
	sort.Slice(cles, func(i, j int) bool { return m[cles[i]] > m[cles[j]] })
	var sb []string
	for i, k := range cles {
		if i >= 20 {
			sb = append(sb, fmt.Sprintf("... %d archetypes de plus", len(cles)-20))
			break
		}
		sb = append(sb, fmt.Sprintf("ti=%d : %d (%.1f %%)", k, m[k], m533bPart(m[k], tot)))
	}
	t.Logf("    %s", strings.Join(sb, " · "))
}

// d519Cles rend les cles d une ventilation par compte decroissant.
func d519Cles(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if m[out[i]] != m[out[j]] {
			return m[out[i]] > m[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}

// d519ClesInt rend l union des cles de deux ventilations entieres, par ordre croissant.
func d519ClesInt(a, b map[int]int) []int {
	vus := map[int]bool{}
	for k := range a {
		vus[k] = true
	}
	for k := range b {
		vus[k] = true
	}
	out := make([]int, 0, len(vus))
	for k := range vus {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
