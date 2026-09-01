package filmdec

// objectif_ti11_minuteurs_verdict_test.go — LES QUATRE PORTES DU LOT ti=11 i0, et le journal par
// film. Le CRITERE est ecrit dans l'en-tete d'`objectif_ti11_minuteurs_test.go` ; ce fichier ne
// fait que l'appliquer et publier les chiffres, y compris quand ils disent non.
//
// L'ORACLE DES EXPLOSIONS N'EST PAS RECOPIE UNE FOIS DE PLUS : `ti12Explosions` porte deja la
// copie du releve A0.3, gardee par `TestNavpointTi12OracleFige` (28 explosions, 9 films). Une
// troisieme copie serait une troisieme occasion de deriver.
//
// # CE QUE LA MESURE A RENDU (passe unique du 2026-09-01, 11 films, 136 s, pic 0,02 Gio)
//
// LA LECTURE B GAGNE, ET SANS AMBIGUITE. Voie IMAGE-CLE, 1 112 lectures sur les neuf films
// d'Assaut et 37 sur le temoin CTF : LEGALITE 100,0 % sur v0, 100,0 % sur v1, 100,0 % sur la
// paire. Zero valeur hors domaine sur 1 149 lectures et 113 slots. Sous l'hypothese d'une fenetre
// mal posee, la probabilite qu'une paire tombe legale vaut (68/128)^2 = 0,282 ; sur les 113 slots
// distincts, l'ordre de grandeur est 1e-62. Et le nombre de valeurs DISTINCTES enfonce le clou :
// HUIT pour v0 (-1, 1, 3, 15, 17, 19, 21, 23) et TROIS pour v1 (-1, 15, 47), la ou un tirage
// uniforme sur sept bits en aurait montre environ 128.
//
// DEUX CONSEQUENCES, ET LA PREMIERE DEPASSE LE CANAL.
//
//	1. L'ANCRAGE DES IMAGES-CLES DE ti=11 EST JUSTE, et c'est la premiere preuve INDEPENDANTE du
//	   chainage qu'ait ce chantier. `objective_scan.go` reclamait un oracle de largeur : le voila,
//	   et il ne coute aucun portage. Portee exacte : il valide l'ANCRE du record et la largeur du
//	   PREMIER composant, pas les composants situes plus loin dans le masque.
//	2. LA VOIE DELTA EST DU BRUIT, MEME FILTREE SUR `Chained`. 1 552 lectures delta d'Assaut :
//	   legalite 45,7 % / 40,3 % / 21,7 % en paire, SOUS le 53,1 % d'un tirage uniforme ; 120 et
//	   121 valeurs distinctes ; 841 et 923 lectures hors domaine. Les 38 lectures survivant au
//	   filtre `Chained` ne remontent qu'a 57,9 % / 71,1 %. Le meme oracle qui valide une voie
//	   condamne l'autre.
//
// GATE 2 — ET LA REPONSE A LA QUESTION DU CHANTIER. Sur les 112 slots d'objectif d'Assaut,
// ZERO porte une valeur qui BOUGE : i0 est FIGE pour toute la vie d'un objectif (jusqu'a
// 37 echantillons sur 24 minutes). Les trois minuteurs reserves n'apparaissent JAMAIS
// (0 lecture a 65, 66 ou 67 sur 1 149) : l'Assaut ne branche pas le chrono de manche sur ses
// objectifs, il ne designe que des fentes du bassin. UN INDEX FIGE NE DATE RIEN.
//
// GATE 3 — CRITERE NON REMPLI, sur les trois voies.
//
//	IMAGE-CLE      0/28 explosions precedees d'une descente (aucune descente n'existe : les
//	               valeurs sont constantes). 28/28 precedees d'une simple lecture.
//	DELTA          22/28, mais dispersion 1,400 pour un plafond de 0,20, et 5 delais hors sens.
//	               Les delais s'etalent de 134 ms a 342 s : c'est la signature du bruit.
//	DELTA CHAINEE  1/28.
//
// TEMOIN — ET IL DIT « GENERIQUE ». Le CTF `cde26226` porte le MEME motif : un slot, valeur
// (v0 = -1, v1 = 15) constante sur 37 echantillons de 20 s a 1 460 s. Le KOTH `7f1bbf06` ne rend
// aucune lecture d'image-cle (12 records, masques nuls) et 16 lectures delta a 53 % de legalite,
// soit exactement le hasard. Les deux Strongholds n'ont AUCUN slot ti=11 dans leurs images-cles —
// le MIROIR du chantier, reconfirme en passant.
//
// VERDICT DU LOT : NEGATIF SUR ti=11 i0 POUR DATER L'ARMEMENT, positif sur l'ancrage des
// images-cles. Ce que i0 rend est le COUPLE D'INDEX (minuteur principal, minuteur secondaire) de
// l'affichage d'objectif, pose une fois pour toutes. La VALEUR du compte a rebours, si elle
// existe, est derriere l'index — dans ti=0 i15 `managed-engine-timers-component`, non porte.
//
// # L'ECART D'HORLOGE, MESURE ET NON PLUS SUPPOSE
//
// Sur les onze films, l'ecart moteur -> manifeste releve chunk par chunk s'etend de 1 a 36 ms
// pour des ecarts de 387 000 a 9 724 000 ms, sur 17 a 49 chunks. Rapporte a sa valeur, la derive
// est inferieure a 1e-4 — c'est-a-dire l'arrondi a la milliseconde des deux bases. LA DOCTRINE DU
// CHANTIER EST VERIFIEE SUR PIECES : l'ecart est bien constant, et les delais ABSOLUS sont donc
// lisibles, pas seulement leur dispersion.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// mntPrefixeTemoin marque, dans `ti11Corpus`, les films d'un autre mode que l'Assaut.
const (
	mntPrefixeTemoin = "TEMOIN"
	mntCampAssaut    = "ASSAUT"
)

func mntEstTemoin(b *mntBilan) bool { return strings.HasPrefix(b.mode, mntPrefixeTemoin) }

// mntAgr agrege une voie sur un sous-ensemble de films.
type mntAgr struct {
	films, lectures, slots, descentes int
	legaux                            [2]int
	legalPaire                        int
	slotsVariables                    [2]int
	ordre                             [3]int
	histo                             [2]map[int]int
}

// mntAgreger cumule la voie `voie` des films retenus par `garde`.
func mntAgreger(bs []*mntBilan, voie int, garde func(*mntBilan) bool) mntAgr {
	a := mntAgr{histo: [2]map[int]int{{}, {}}}
	for _, b := range bs {
		if !garde(b) {
			continue
		}
		v := b.voies[voie]
		a.films++
		a.lectures += v.lectures
		a.slots += v.slots
		a.descentes += len(v.descentes)
		a.legalPaire += v.legalPaire
		for k := 0; k < 2; k++ {
			a.legaux[k] += v.legaux[k]
			a.slotsVariables[k] += v.slotsVariables[k]
			for val, n := range v.histo[k] {
				a.histo[k][val] += n
			}
		}
		for k := 0; k < 3; k++ {
			a.ordre[k] += v.ordre[k]
		}
	}
	return a
}

// mntJournalFilm publie ce qu'un film a rendu.
func mntJournalFilm(t *testing.T, b *mntBilan) {
	t.Helper()
	sc := b.sc
	t.Logf("%-9s %-26s DELTA %d ancres, %d marches, %d chainees (%.1f %%) | IMAGE-CLE %d records, "+
		"%d marches, %d chainees (%.1f %%) | bande %d slot(s) · balayage en %s",
		b.id, b.mode, sc.Records, sc.Walked, sc.Chained, ti11Part(sc.Chained, sc.Walked),
		sc.KeyRecords, sc.KeyWalked, sc.KeyChained, ti11Part(sc.KeyChained, sc.KeyWalked),
		sc.Slots, b.duree.Round(time.Second))
	t.Logf("           HORLOGE : ecart moteur -> manifeste %s · %d horodatage(s) ambigu(s) · "+
		"%d lecture(s) sans horloge · %d lecture(s) sans seconde valeur",
		mntEcart(b.ecarts), b.ambigus, b.sansHorloge, b.sansSecond)
	for i := 0; i < mntVoies; i++ {
		mntLigneVoie(t, b.voies[i], mntNomVoie[i])
	}
}

// mntLigneVoie publie une voie d'un film.
func mntLigneVoie(t *testing.T, v *mntVoieBilan, nom string) {
	t.Helper()
	if v.lectures == 0 {
		t.Logf("           %-13s aucune lecture", nom)
		return
	}
	t.Logf("           %-13s %d lecture(s) · %d slot(s) (%d/%d a valeur variable) · "+
		"LEGALITE v0 %.1f %% v1 %.1f %% paire %.1f %% · ordre v0>v1 %d, = %d, v0<v1 %d · "+
		"%d descente(s)", nom, v.lectures, v.slots, v.slotsVariables[0], v.slotsVariables[1],
		ti11Part(v.legaux[0], v.lectures), ti11Part(v.legaux[1], v.lectures),
		ti11Part(v.legalPaire, v.lectures), v.ordre[0], v.ordre[1], v.ordre[2], len(v.descentes))
	t.Logf("                         v0 %s", mntClasses(v.histo[0]))
	t.Logf("                         v1 %s", mntClasses(v.histo[1]))
	for _, e := range v.extraits {
		t.Logf("                         %s", e)
	}
}

// mntEcart rend la plage des ecarts horloge moteur -> manifeste, chunk par chunk. Un ecart
// CONSTANT valide la doctrine du chantier ; un ecart qui bouge la corrige.
func mntEcart(ecarts []int) string {
	if len(ecarts) == 0 {
		return "(aucun chunk date)"
	}
	tri := append([]int(nil), ecarts...)
	sort.Ints(tri)
	lo, hi := tri[0], tri[len(tri)-1]
	if lo == hi {
		return fmt.Sprintf("CONSTANT a %d ms sur %d chunk(s)", lo, len(tri))
	}
	return fmt.Sprintf("VARIABLE [%d .. %d] ms (etendue %d) sur %d chunk(s)",
		lo, hi, hi-lo, len(tri))
}

// mntClasses rend la distribution d'un champ par classe du domaine, puis ses valeurs dominantes.
func mntClasses(m map[int]int) string {
	var absent, bassin, hors64, manche, subite, grace, illegal int
	for v, n := range m {
		switch {
		case v == mntAbsent:
			absent += n
		case v >= 0 && v <= mntBassinMax:
			bassin += n
		case v == mntBassinMax+1:
			hors64 += n
		case v == mntManche:
			manche += n
		case v == mntMortSubite:
			subite += n
		case v == mntGrace:
			grace += n
		default:
			illegal += n
		}
	}
	return fmt.Sprintf("absent %d · bassin[0..63] %d · 64 %d · manche(65) %d · subite(66) %d · "+
		"grace(67) %d · illegal(>=68) %d · %d valeur(s) distincte(s) · dominantes %s",
		absent, bassin, hors64, manche, subite, grace, illegal, len(m), mntTop(m, 6))
}

// mntTop rend les n valeurs les plus frequentes, « valeur x compte ».
func mntTop(m map[int]int, n int) string {
	vs := make([]int, 0, len(m))
	for v := range m {
		vs = append(vs, v)
	}
	sort.Slice(vs, func(i, j int) bool {
		if m[vs[i]] != m[vs[j]] {
			return m[vs[i]] > m[vs[j]]
		}
		return vs[i] < vs[j]
	})
	out := make([]string, 0, n)
	for i, v := range vs {
		if i == n {
			out = append(out, fmt.Sprintf("(+%d autres)", len(vs)-n))
			break
		}
		out = append(out, fmt.Sprintf("%d x%d", v, m[v]))
	}
	return strings.Join(out, ", ")
}

// mntGate0 — LEGALITE. L'oracle d'ancrage : le domaine legal ne couvre que 68 des 128 valeurs
// encodables, donc un ancrage faux se lit a 53 % et un ancrage juste a ~100 %.
func mntGate0(t *testing.T, bs []*mntBilan) {
	t.Helper()
	attendu := 100 * float64(mntLegales) / float64(mntValeursSept)
	t.Logf("########## GATE 0 LEGALITE — domaine {-1} U [0,%d] U {%d,%d,%d} = %d valeurs sur %d ; "+
		"un tirage UNIFORME donnerait %.1f %%", mntBassinMax, mntManche, mntMortSubite, mntGrace,
		mntLegales, mntValeursSept, attendu)
	for i := 0; i < mntVoies; i++ {
		mntLigneLegalite(t, mntCampAssaut, mntAgreger(bs, i, func(b *mntBilan) bool {
			return !mntEstTemoin(b)
		}), mntNomVoie[i])
		mntLigneLegalite(t, mntPrefixeTemoin, mntAgreger(bs, i, mntEstTemoin), mntNomVoie[i])
	}
}

// mntLigneLegalite publie le taux de legalite d'un agregat et le lit.
func mntLigneLegalite(t *testing.T, camp string, a mntAgr, voie string) {
	t.Helper()
	if a.lectures == 0 {
		t.Logf("    %-6s %-13s aucune lecture sur %d film(s)", camp, voie, a.films)
		return
	}
	p := ti11Part(a.legalPaire, a.lectures)
	lecture := "MELANGE — ancrage partiellement faux"
	switch {
	case p >= 95:
		lecture = "ANCRAGE JUSTE — i0 est un oracle d'ancrage"
	case p <= 60:
		lecture = "UNIFORME — la fenetre est mal posee"
	}
	t.Logf("    %-6s %-13s %d lecture(s) sur %d film(s) : v0 %.1f %% · v1 %.1f %% · PAIRE %.1f %% "+
		"-> %s", camp, voie, a.lectures, a.films, ti11Part(a.legaux[0], a.lectures),
		ti11Part(a.legaux[1], a.lectures), p, lecture)
}

// mntGate1 — TEMOIN. Sans lui, un resultat sur l'Assaut ne vaut rien.
func mntGate1(t *testing.T, bs []*mntBilan) {
	t.Helper()
	vus, total := 0, 0
	for _, b := range bs {
		if !mntEstTemoin(b) {
			continue
		}
		total++
		n := b.voies[mntVoieCle].lectures + b.voies[mntVoieDelta].lectures
		if n > 0 {
			vus++
		}
		t.Logf("    %-9s %-26s %d lecture(s) image-cle, %d delta (%d chainees), %d descente(s)",
			b.id, b.mode, b.voies[mntVoieCle].lectures, b.voies[mntVoieDelta].lectures,
			b.voies[mntVoieDeltaChainee].lectures,
			len(b.voies[mntVoieCle].descentes)+len(b.voies[mntVoieDelta].descentes))
	}
	t.Logf("########## GATE 1 TEMOIN — %d temoin(s) sur %d rendent des lectures de i0", vus, total)
	if vus == 0 {
		t.Logf("VERDICT GATE 1 : L'INSTRUMENT NE VOIT RIEN HORS ASSAUT. Un silence sur l'Assaut "+
			"ne vaudrait alors rien ; c'est l'instrument qu'il faut mettre en cause.%s", "")
	}
}

// mntGate2 — SEMANTIQUE. Ce que les valeurs SONT, voie par voie, camp par camp.
func mntGate2(t *testing.T, bs []*mntBilan) {
	t.Helper()
	t.Logf("########## GATE 2 SEMANTIQUE — distribution des deux valeurs, par voie")
	for i := 0; i < mntVoies; i++ {
		mntGate2Voie(t, mntCampAssaut, mntAgreger(bs, i, func(b *mntBilan) bool {
			return !mntEstTemoin(b)
		}), mntNomVoie[i])
		mntGate2Voie(t, mntPrefixeTemoin, mntAgreger(bs, i, mntEstTemoin), mntNomVoie[i])
	}
}

// mntGate2Voie publie la distribution d'un agregat et la lit selon le critere ecrit d'avance.
func mntGate2Voie(t *testing.T, camp string, a mntAgr, voie string) {
	t.Helper()
	if a.lectures == 0 {
		return
	}
	t.Logf("    %-6s %-13s %d lecture(s) · %d slot(s), dont %d a v0 variable et %d a v1 variable "+
		"· ordre v0>v1 %d, = %d, v0<v1 %d", camp, voie, a.lectures, a.slots,
		a.slotsVariables[0], a.slotsVariables[1], a.ordre[0], a.ordre[1], a.ordre[2])
	t.Logf("           v0 %s", mntClasses(a.histo[0]))
	t.Logf("           v1 %s", mntClasses(a.histo[1]))
	if camp != mntCampAssaut {
		return
	}
	bassin := 0
	for v, n := range a.histo[0] {
		if v >= 0 && v <= mntBassinMax {
			bassin += n
		}
	}
	for v, n := range a.histo[1] {
		if v >= 0 && v <= mntBassinMax {
			bassin += n
		}
	}
	if bassin == 0 {
		t.Logf("           LECTURE %s : aucun index de bassin (0..63) — i0 ne designe que des "+
			"minuteurs reserves ou l'absence, il ne parle PAS de la bombe", voie)
		return
	}
	t.Logf("           LECTURE %s : %d lecture(s) portent un index de bassin (0..63) — un "+
		"minuteur du moteur est branche sur un objectif d'Assaut", voie, bassin)
}

// mntGate3 — LE CRITERE DU CHANTIER contre les 28 explosions.
func mntGate3(t *testing.T, bs []*mntBilan) {
	t.Helper()
	par := make(map[string]*mntBilan, len(bs))
	for _, b := range bs {
		par[b.id] = b
	}
	t.Logf("########## GATE 3 CRITERE — 28 explosions, delai = explosion moins DEPART de descente "+
		"(descente : >= %d ech., amplitude >= %d quanta, l'absence coupe la suite)",
		mntDescenteMinEch, mntDescenteMinAmpl)
	for i := 0; i < mntVoies; i++ {
		mntPasse(t, mntNomVoie[i], par, i)
	}
	mntDetail(t, par)
}

// mntPasse applique le critere sur une voie et publie couverture, constance et sens.
func mntPasse(t *testing.T, titre string, par map[string]*mntBilan, voie int) {
	t.Helper()
	var delais []int
	total, couvLecture, horsSens := 0, 0, 0
	for _, id := range ti12FilmsOracle() {
		b := par[id]
		for _, ms := range ti12Explosions[id] {
			total++
			if b == nil {
				continue
			}
			v := b.voies[voie]
			if ti12LectureAvant(v.instants, int32(ms)) {
				couvLecture++
			}
			d, ok := mntDelaiAvant(v.descentes, int32(ms))
			if !ok {
				continue
			}
			if d > mntSensMaxMS {
				horsSens++
				continue
			}
			delais = append(delais, d)
		}
	}
	mntVerdict(t, titre, total, couvLecture, horsSens, delais)
}

// mntDelaiAvant rend le delai entre le DEPART de la derniere descente commencee avant t et t.
func mntDelaiAvant(ds []mntDescente, t int32) (int, bool) {
	i := sort.Search(len(ds), func(k int) bool { return ds[k].debutMS > t })
	if i == 0 {
		return 0, false
	}
	return int(t - ds[i-1].debutMS), true
}

// mntVerdict publie couverture, constance et sens pour une voie.
func mntVerdict(t *testing.T, titre string, total, couvLecture, horsSens int, delais []int) {
	t.Helper()
	sort.Ints(delais)
	t.Logf("--- %-13s %d/%d explosion(s) precedees d'une DESCENTE dans le sens (%d hors sens > "+
		"%d ms) · %d/%d precedees d'une simple lecture de i0",
		titre, len(delais), total, horsSens, mntSensMaxMS, couvLecture, total)
	if len(delais) == 0 {
		t.Logf("    COUVERTURE 0 — critere NON REMPLI, rien a mesurer sur la constance")
		return
	}
	med := ti12Quantile(delais, 0.5)
	p25, p75 := ti12Quantile(delais, 0.25), ti12Quantile(delais, 0.75)
	disp := 0.0
	if med != 0 {
		disp = float64(p75-p25) / float64(med)
	}
	t.Logf("    DELAIS (ms) : min %d · p25 %d · mediane %d · p75 %d · max %d",
		delais[0], p25, med, p75, delais[len(delais)-1])
	t.Logf("    DISPERSION (p75-p25)/mediane = %.3f (plafond %.2f) · COUVERTURE %d/%d · SENS %d "+
		"hors [0, %d] ms", disp, mntDispersionMax, len(delais), total, horsSens, mntSensMaxMS)
	verdict := "NEGATIF sur ce canal"
	if len(delais) == total && disp <= mntDispersionMax && horsSens == 0 {
		verdict = "CANDIDAT — les trois criteres passent"
	}
	t.Logf("    VERDICT %s : %s", titre, verdict)
}

// mntDetail imprime une ligne PAR EXPLOSION — la descente qui la precede, son slot, sa course.
// Un tableau agrege cache toujours le cas qui explique tout ; celui-la ne cache rien. Il n'est
// imprime que si une voie porte au moins une descente : sans descente, il ne dirait que « rien ».
func mntDetail(t *testing.T, par map[string]*mntBilan) {
	t.Helper()
	voie := -1
	for i := 0; i < mntVoies && voie < 0; i++ {
		for _, b := range par {
			if len(b.voies[i].descentes) > 0 {
				voie = i
				break
			}
		}
	}
	if voie < 0 {
		t.Logf("--- DETAIL PAR EXPLOSION : aucune descente sur aucune voie — rien a detailler")
		return
	}
	t.Logf("--- DETAIL PAR EXPLOSION (voie %s, descente la plus recente commencee avant)",
		mntNomVoie[voie])
	for _, id := range ti12FilmsOracle() {
		b := par[id]
		if b == nil {
			continue
		}
		ds := b.voies[voie].descentes
		for _, ms := range ti12Explosions[id] {
			i := sort.Search(len(ds), func(k int) bool { return ds[k].debutMS > int32(ms) })
			if i == 0 {
				t.Logf("    %s %7d ms : AUCUNE descente avant", id, ms)
				continue
			}
			d := ds[i-1]
			t.Logf("    %s %7d ms : delai %6d ms · slot %d champ v%d · %d -> %d en %d ech.",
				id, ms, ms-int(d.debutMS), d.slot, d.champ, d.haut, d.bas, d.n)
		}
	}
}
