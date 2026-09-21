//go:build research

package grammar

// mouvement_5_3_2_tableaux_research_test.go — CE QUE LA MESURE DE 5.3.2 PUBLIE.
//
// Scinde de `mouvement_5_3_2_research_test.go` par DEPLACEMENT PUR le 2026-09-21, sur refus du
// ratchet de taille (`TestTailleDesFichiersDuFilmNeCroitPas` : 826 lignes pour un seuil de 500).
// Aucune ligne de logique n a change : la LECTURE reste dans le fichier d origine, les TABLEAUX
// et leurs outils sont ici. Les deux fichiers forment un seul instrument, et le seul point
// d entree reste `TestMouvement532`.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

func m532NbSlots(ech []m532Ech) int {
	vus := map[uint32]bool{}
	for _, e := range ech {
		vus[e.slot] = true
	}
	return len(vus)
}

func m532Couverture(t *testing.T, ech []m532Ech) {
	t.Helper()
	n := len(ech)
	complet, v, ctl, mot, cr, mo, po, sl := 0, 0, 0, 0, 0, 0, 0, 0
	for _, e := range ech {
		if e.marcheComplete {
			complet++
		}
		if e.aVitesse {
			v++
		}
		if e.aControle {
			ctl++
		}
		if e.aMot32 {
			mot++
		}
		if e.aCrouch {
			cr++
		}
		if e.aMobilite {
			mo++
		}
		if e.aPosture {
			po++
		}
		if e.aSlide {
			sl++
		}
	}
	t.Logf("COUVERTURE : marche complete %d/%d (%.1f %%)", complet, n, m532Pct(complet, n))
	m532Bloquants(t, ech)
	t.Logf("  i1 vitesse %d (%.1f %%) · i18 controle %d (%.1f %%) dont mot de 32 bits %d (%.1f %%)",
		v, m532Pct(v, n), ctl, m532Pct(ctl, n), mot, m532Pct(mot, n))
	t.Logf("  i29 crouch %d (%.1f %%) · i54 mobilite %d (%.1f %%) · i55 posture %d (%.1f %%) · i62 slide %d (%.1f %%)",
		cr, m532Pct(cr, n), mo, m532Pct(mo, n), po, m532Pct(po, n), sl, m532Pct(sl, n))
}

// m532Masques est LE DENOMINATEUR DU NEGATIF : dire « i29 n est sur aucun record » ne vaut rien
// tant qu on n a pas montre CE QUE les masques delta declarent. Un masque qui s arrete a i21 ne
// dit pas que l accroupissement n existe pas : il dit que ce record-la ne le transporte pas.
// m532Bloquants nomme CE QUE LA MARCHE N ATTEINT PAS. Un composant a zero n est pas forcement
// absent du film : il peut etre DERRIERE le composant qui arrete la marche. C est la difference
// entre « le film ne le dit pas » et « le decodeur ne va pas jusque-la », et elle est capitale.
func m532Bloquants(t *testing.T, ech []m532Ech) {
	t.Helper()
	par := map[string]int{}
	for _, e := range ech {
		if e.bloquant != "" {
			par[e.bloquant]++
		}
	}
	if len(par) == 0 {
		return
	}
	noms := make([]string, 0, len(par))
	for k := range par {
		noms = append(noms, k)
	}
	sort.Slice(noms, func(a, b int) bool { return par[noms[a]] > par[noms[b]] })
	var parts []string
	for i, n := range noms {
		if i >= 4 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s:%d", n, par[n]))
	}
	t.Logf("  BLOQUANTS (ce qui est DERRIERE eux n est pas lu) : %s", strings.Join(parts, " · "))
}

func m532Masques(t *testing.T, ech []m532Ech) {
	t.Helper()
	parIndex := map[int]int{}
	tailles := map[int]int{}
	maxIdx := map[int]int{}
	for _, e := range ech {
		n, hi := 0, -1
		for i := 0; i < 64; i++ {
			if e.masque&(uint64(1)<<uint(i)) != 0 {
				parIndex[i]++
				n++
				hi = i
			}
		}
		tailles[n]++
		maxIdx[hi]++
	}
	var vus []int
	for i := range parIndex {
		vus = append(vus, i)
	}
	sort.Ints(vus)
	var parts []string
	for _, i := range vus {
		parts = append(parts, fmt.Sprintf("i%d:%.1f%%", i, m532Pct(parIndex[i], len(ech))))
	}
	t.Logf("MASQUES DELTA — index declares : %s", strings.Join(parts, " "))
	var hauts []int
	for i := range maxIdx {
		hauts = append(hauts, i)
	}
	sort.Ints(hauts)
	parts = parts[:0]
	for _, i := range hauts {
		parts = append(parts, fmt.Sprintf("i%d:%d", i, maxIdx[i]))
	}
	t.Logf("MASQUES DELTA — index LE PLUS HAUT de chaque record : %s", strings.Join(parts, " "))
}

func m532Etats(t *testing.T, ech []m532Ech) {
	t.Helper()
	accroupi, glisse := 0, 0
	histoCrouch := map[int]int{}
	for _, e := range ech {
		if e.aCrouch && e.crouch {
			accroupi++
		}
		if e.aCrouch {
			histoCrouch[int(e.crouchQ)/128]++
		}
		if e.aSlide && e.slide {
			glisse++
		}
	}
	t.Logf("ETATS : accroupi %d records · glissade %d records", accroupi, glisse)
	var parts []string
	for b := 0; b < 8; b++ {
		if histoCrouch[b] > 0 {
			parts = append(parts, fmt.Sprintf("[%.2f-%.2f[:%d", float64(b)/8, float64(b+1)/8, histoCrouch[b]))
		}
	}
	t.Logf("  progression d accroupissement (8 classes sur [0,1]) : %s", strings.Join(parts, " "))
	m532VitesseParEtat(t, ech)
}

// m532VitesseParEtat est L ORACLE : la magnitude quantifiee doit SEPARER les etats.
func m532VitesseParEtat(t *testing.T, ech []m532Ech) {
	t.Helper()
	classes := map[string][]uint32{}
	for _, e := range ech {
		if !e.aVitesse {
			continue
		}
		switch {
		case e.aSlide && e.slide:
			classes["glissade"] = append(classes["glissade"], e.magQ)
		case e.aCrouch && e.crouch:
			classes["accroupi"] = append(classes["accroupi"], e.magQ)
		case e.aMobilite && e.flag1:
			classes["action de mobilite"] = append(classes["action de mobilite"], e.magQ)
		default:
			classes["debout"] = append(classes["debout"], e.magQ)
		}
	}
	noms := make([]string, 0, len(classes))
	for k := range classes {
		noms = append(noms, k)
	}
	sort.Strings(noms)
	for _, nom := range noms {
		v := classes[nom]
		sort.Slice(v, func(a, b int) bool { return v[a] < v[b] })
		t.Logf("  VITESSE (magnitude quantifiee) %-20s n=%6d  p10=%4d  median=%4d  p90=%4d",
			nom, len(v), m532Quantile(v, 0.10), m532Quantile(v, 0.50), m532Quantile(v, 0.90))
	}
}

func m532TableauMobilite(t *testing.T, ech []m532Ech) {
	t.Helper()
	f1, f2, ident := 0, 0, 0
	q7, q2, ids := map[uint32]int{}, map[uint32]int{}, map[uint32]int{}
	parSlot := map[uint32]int{}
	for _, e := range ech {
		if !e.aMobilite {
			continue
		}
		if e.flag2 {
			f2++
		}
		if !e.flag1 {
			continue
		}
		f1++
		parSlot[e.slot]++
		q7[e.queue7]++
		q2[e.queue2]++
		if e.aIdent {
			ident++
			ids[e.ident]++
		}
	}
	t.Logf("i54 : flag1 (une action est transmise) %d · flag2 %d · identifiant present %d", f1, f2, ident)
	t.Logf("  `+0x9c` R(2) : %s", m532Histo(q2, 4))
	t.Logf("  `+0x98` R(7) : %s", m532Histo(q7, 8))
	t.Logf("  identifiant 10 b : %s", m532Histo(ids, 8))
	t.Logf("  slots porteurs d une action : %d", len(parSlot))
}

func m532Postures(t *testing.T, ech []m532Ech) {
	t.Helper()
	tags := map[uint32]int{}
	n := 0
	for _, e := range ech {
		if !e.aPosture {
			continue
		}
		n++
		tags[e.tag]++
	}
	if n == 0 {
		t.Logf("D1 SUR LE CHEMIN DELTA : `i55` n est declare par AUCUN des %d records delta. "+
			"Le `Skip(2)` n y est jamais exerce.", len(ech))
		return
	}
	nonNuls := n - tags[0]
	t.Logf("D1 SUR LE CHEMIN DELTA : `i55` present sur %d/%d records (%.1f %%) · tags %s · "+
		"NON NULS %d (%.1f %%)", n, len(ech), m532Pct(n, len(ech)), m532Histo(tags, 4),
		nonNuls, m532Pct(nonNuls, n))
}

// m532BitsDeControle — LA MESURE DU SAUT. Pour chaque bit du mot de 32 bits d `i18 +0x544`, on
// compare la composante verticale de la vitesse au record SUIVANT du MEME slot : un bit de saut
// s allume AVANT que `vz` ne devienne positif.
func m532BitsDeControle(t *testing.T, ech []m532Ech) {
	t.Helper()
	parSlot := map[uint32][]m532Ech{}
	for _, e := range ech {
		parSlot[e.slot] = append(parSlot[e.slot], e)
	}
	var allume [32]int
	var monteApres [32]int
	total, avecSuite := 0, 0
	for _, serie := range parSlot {
		sort.Slice(serie, func(a, b int) bool { return serie[a].tUS < serie[b].tUS })
		for i, e := range serie {
			if !e.aMot32 {
				continue
			}
			total++
			var suite *m532Ech
			for j := i + 1; j < len(serie) && j <= i+3; j++ {
				if serie[j].aVitesse {
					suite = &serie[j]
					break
				}
			}
			if suite == nil {
				continue
			}
			avecSuite++
			monte := suite.dirZ > 0.30
			for b := 0; b < 32; b++ {
				if e.mot32&(1<<uint(b)) == 0 {
					continue
				}
				allume[b]++
				if monte {
					monteApres[b]++
				}
			}
		}
	}
	if total == 0 {
		t.Logf("MOT DE 32 BITS : aucun record ne le porte — rien a ventiler")
		return
	}
	// LE PLANCHER : la proportion de montees TOUS RECORDS CONFONDUS. Un bit n est un candidat
	// que s il la depasse FRANCHEMENT — sans ce denominateur, un bit toujours allume aurait
	// l air de predire tout ce qui arrive.
	montees := 0
	for _, serie := range parSlot {
		for _, e := range serie {
			if e.aVitesse && e.dirZ > 0.30 {
				montees++
			}
		}
	}
	plancher := m532Pct(montees, len(ech))
	t.Logf("MOT DE 32 BITS d `i18 +0x544` : %d records le portent, %d ont une suite lisible. "+
		"PLANCHER de montee (dirZ > 0,30) sur tout le film : %.1f %%", total, avecSuite, plancher)
	for b := 0; b < 32; b++ {
		if allume[b] == 0 {
			continue
		}
		p := m532Pct(monteApres[b], allume[b])
		marque := ""
		if p > plancher*2 && allume[b] >= 20 {
			marque = "   <-- CANDIDAT"
		}
		t.Logf("  bit %2d : allume %6d fois (%.1f %% des records) · monte ensuite %.1f %%%s",
			b, allume[b], m532Pct(allume[b], total), p, marque)
	}
}

// m532Instants rend, pour l oeil de l utilisateur, cinq instants par etat en TEMPS DE BARRE
// THEATER (mm:ss depuis le debut du film), espaces d au moins dix secondes pour ne pas donner
// cinq fois le meme geste.
func m532Instants(t *testing.T, ech []m532Ech, tzero uint64) {
	t.Helper()
	type cas struct {
		nom  string
		test func(m532Ech) bool
	}
	cas5 := []cas{
		{"ACCROUPI", func(e m532Ech) bool { return e.aCrouch && e.crouch && e.crouchQ > 800 }},
		{"GLISSADE", func(e m532Ech) bool { return e.aSlide && e.slide }},
		{"ACTION DE MOBILITE (sprint / escalade)", func(e m532Ech) bool { return e.aMobilite && e.flag1 }},
		{"MONTEE (candidat saut)", func(e m532Ech) bool { return e.aVitesse && e.dirZ > 0.60 }},
	}
	tri := append([]m532Ech(nil), ech...)
	sort.Slice(tri, func(a, b int) bool { return tri[a].tUS < tri[b].tUS })
	for _, c := range cas5 {
		var lignes []string
		var dernier uint64
		for _, e := range tri {
			if !c.test(e) {
				continue
			}
			if len(lignes) > 0 && e.tUS < dernier+10_000_000 {
				continue
			}
			dernier = e.tUS
			lignes = append(lignes, fmt.Sprintf("%s (slot %d)", m532Barre(e.tUS, tzero), e.slot))
			if len(lignes) == 5 {
				break
			}
		}
		if len(lignes) == 0 {
			t.Logf("INSTANTS %-40s AUCUN", c.nom)
			continue
		}
		t.Logf("INSTANTS %-40s %s", c.nom, strings.Join(lignes, " · "))
	}
}

// ---------------------------------------------------------------------------
// OUTILS
// ---------------------------------------------------------------------------

func m532Barre(tUS, tzero uint64) string {
	if tUS < tzero {
		return "00:00"
	}
	s := (tUS - tzero) / 1_000_000
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

func m532Pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}

func m532Quantile(v []uint32, q float64) uint32 {
	if len(v) == 0 {
		return 0
	}
	i := int(q * float64(len(v)-1))
	return v[i]
}

func m532Histo(m map[uint32]int, max int) string {
	if len(m) == 0 {
		return "(aucun)"
	}
	cles := make([]int, 0, len(m))
	for k := range m {
		cles = append(cles, int(k)) //nolint:gosec // valeurs de champs courts
	}
	sort.Ints(cles)
	var parts []string
	for i, k := range cles {
		if i >= max {
			parts = append(parts, fmt.Sprintf("... %d autres valeurs", len(cles)-max))
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%d", k, m[uint32(k)])) //nolint:gosec // clef issue d un uint32
	}
	return strings.Join(parts, " ")
}

func m532EcrireTSV(t *testing.T, ech []m532Ech, tzero uint64, chemin string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("barre\tslot\tcomplet\tdirZ\tmagQ\tmot32\tcrouch\tcrouchQ\tflag1\tident\tq7\tq2\ttag\tslide\n")
	for _, e := range ech {
		fmt.Fprintf(&b, "%s\t%d\t%t\t%.3f\t%d\t%08x\t%t\t%d\t%t\t%d\t%d\t%d\t%d\t%t\n",
			m532Barre(e.tUS, tzero), e.slot, e.marcheComplete, e.dirZ, e.magQ, e.mot32,
			e.crouch, e.crouchQ, e.flag1, e.ident, e.queue7, e.queue2, e.tag, e.slide)
	}
	if err := os.WriteFile(chemin, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", chemin, err)
	}
	t.Logf("TSV : %s (%d lignes)", chemin, len(ech))
}
