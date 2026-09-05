package filmdec

// navpoint_ti12_meche_test.go — LA LECTURE « MECHE PAUSABLE » : One Bomb explique, et le temoin.
//
// # CE QUE L'INSPECTION A MONTRE (navpoint_ti12_onebomb_test.go, log du 2026-09-01)
//
// Sur les trois films One Bomb, l'anneau ti=12 i14 du site porte TROIS figures distinctes, et le
// plancher les melangeait :
//
//	ARMEMENT   un segment contigu qui monte 131 -> 254 et FINIT PLEIN (n=30, ~2,9 s, identique
//	           aux dix occurrences) ;
//	DESARMEMENT une tenue de defenseur : segment strictement DESCENDANT depuis ~251, pente
//	           mesuree 14 a 26 quanta/s, interrompu (fins a 185..246) ou complet (fin a 127) ;
//	RESET      apres chaque evenement terminal du site, l'autre paire de slots joue un cycle
//	           complet 130 -> 253 -> 127 en 5,0 s (n=51). Sa sous-montee close au sommet passait
//	           le filtre « montee » du plancher et fournissait des fins parasites.
//
// La chute de l'anneau a l'explosion, elle, est RAPIDE : 251 -> 127 a ~138 quanta/s.
//
// # LA LECTURE, definitions FIGEES AVANT LA MESURE
//
//	SEGMENT    suite maximale d'echantillons d'un meme slot, trous <= tpTrouMaxMS (500 ms).
//	ARMEMENT   segment avec n >= ti12MonteeMinEch, amplitude montante (dernier - min) >=
//	           ti12MonteeMinAmpl, ET dernier echantillon >= max du segment - la tolerance :
//	           le segment FINIT a son sommet (l'anneau reste plein). Le cycle RESET finit a
//	           127, il est ecarte ; une descente finit a son minimum, ecartee.
//	PAUSE      segment avec n >= 2, dernier < premier, max <= premier + la tolerance (jamais
//	           au-dessus de son depart : le RESET remonte, il est ecarte), pente moyenne
//	           (premier - dernier) / duree < la pente maximale. La chute d'explosion (~138 q/s)
//	           est au-dessus du seuil, les tenues de desarmement (14-26 q/s) en dessous.
//	DELAI      pour une cible : (cible - derniere fin d'ARMEMENT avant cible) moins la somme
//	           des durees des PAUSES du MEME slot strictement entre cette fin et la cible.
//	           Fenetre de sens et statistique inchangees (tpSensMaxMS, tpMedCV).
//
// Seuils NOUVEAUX, justifies par la mesure d'inspection, et EN PRODUCTION depuis le
// 2026-09-04 (navpoint_radial_segments.go) : NavpointSummitToleranceQ = 4 quanta (1/64 de
// course, sous l'amplitude minimale 16 — tolerance de bruit de fin de segment) ;
// NavpointPauseMaxSlopeQS = 60 quanta/s, au milieu du vide entre les pentes de desarmement
// observees (14 a 26) et la chute d'explosion (138). AUCUN seuil existant n'est modifie.
//
// # LA REGLE DE DECISION, ecrite ici et appliquee telle quelle
//
//	One Bomb : couverture, CV (ecart-type / mediane), et tirage nul (1 000, graine 1, cibles
//	uniformes par film, memes effectifs, meme lecture) publies pour LES ONZE explosions ET pour
//	les NEUF explosions PORTEES par un joueur. La partition est ANTERIEURE a cette mesure :
//	les deux explosions sans slot de joueur porteur sont relevees dans a5SansPorteur
//	(assaut_a5_explosions_test.go, 2026-08-31) — recopiees ici dans mpSansPorteur.
//	RESOLU ssi couverture pleine, CV <= tpCVSeuil, et moins de tpNulSeuilPc % des tirages
//	nuls font aussi bien.
//
//	TEMOIN : la MEME lecture rejouee sur les 13 explosions Neutral Bomb et les 4 de Husky
//	Raid. Elle doit rendre 13/13 avec CV <= 0,02 (et 4/4 sur Husky) — une lecture qui
//	ameliore One Bomb en cassant Neutral est fausse.
//
// REGIME : garde ASSAUT_CACHE. Aucune base, aucun reseau, sentinelle memoire armee, un seul
// decodage a la fois (LockProcessDecode).
//
//	go test ./internal/analysis/filmdec/ -run NavpointTi12Meche -v -timeout 60m

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"testing"
)

// mpCVSeuilTemoin : plafond du CV exige du temoin Neutral (le brief du chantier).
//
// LES DEUX SEUILS DE LA LECTURE ONT QUITTE CE FICHIER le 2026-09-04 : la tolerance de sommet
// (4 quanta) et la pente maximale d'une tenue (60 quanta/s) vivent en PRODUCTION sous les noms
// `NavpointSummitToleranceQ` et `NavpointPauseMaxSlopeQS` (navpoint_radial_segments.go), avec
// la mesure qui les justifie. Cet instrument les applique en appelant les predicats de
// production : il ne peut plus mesurer autre chose que ce qui est livre.
const mpCVSeuilTemoin = 0.02

// mpSansPorteur : les deux explosions SANS slot de joueur porteur, recopiees de a5SansPorteur
// (analysis/replay/assaut_a5_explosions_test.go, mesure du 2026-08-31, anterieure a ce lot —
// l'import direct ferait un cycle). Le point de mode n'y existe que sur le slot d'EQUIPE.
var mpSansPorteur = map[string]int32{"df8fcbef": 778033, "c75f33b8": 395724}

// TestNavpointTi12MecheSansPorteurFige garde la copie de mpSansPorteur contre une derive :
// chaque instant doit exister dans l'oracle fige ti12Explosions, sur le meme film.
func TestNavpointTi12MecheSansPorteurFige(t *testing.T) {
	if len(mpSansPorteur) != 2 {
		t.Fatalf("mpSansPorteur : %d entrees, attendu 2", len(mpSansPorteur))
	}
	for id, ms := range mpSansPorteur {
		trouve := false
		for _, e := range ti12Explosions[id] {
			if int32(e) == ms {
				trouve = true
			}
		}
		if !trouve {
			t.Fatalf("mpSansPorteur[%s]=%d absent de l'oracle ti12Explosions", id, ms)
		}
	}
}

// mpArmement est une fin de segment d'armement (l'anneau vient de finir plein).
type mpArmement struct {
	slot  uint32
	finMS int32
}

// mpPause est une tenue de desarmement datee.
type mpPause struct {
	t0, t1 int32
}

// mpFilm porte la digestion d'un film pour la lecture meche pausable.
type mpFilm struct {
	id        string
	exps      []int32
	armements []mpArmement
	pauses    map[uint32][]mpPause
	min, max  int32
}

// TestNavpointTi12MecheOneBomb applique la lecture aux trois films One Bomb.
func TestNavpointTi12MecheOneBomb(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer tpSentinelle(t)()
	release := LockProcessDecode()
	defer release()

	films := mpChargerGroupe(t, cache, obFilms)
	t.Logf("########## DETAIL PAR EXPLOSION (lecture meche pausable)")
	for _, f := range films {
		for _, e := range f.exps {
			mpDetail(t, f, e)
		}
	}
	mpVerdict(t, "ONE BOMB — 11 explosions", films, nil)
	portees := func(id string, e int32) bool { return mpSansPorteur[id] != e }
	mpVerdict(t, "ONE BOMB — 9 explosions PORTEES (partition a5SansPorteur)", films, portees)
}

// TestNavpointTi12MecheTemoin rejoue la MEME lecture sur Neutral Bomb et Husky Raid.
func TestNavpointTi12MecheTemoin(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer tpSentinelle(t)()
	release := LockProcessDecode()
	defer release()

	neutral := mpChargerGroupe(t, cache, tpFilms)
	mpVerdict(t, "TEMOIN NEUTRAL BOMB — 13 explosions", neutral, nil)
	couv, cv, _ := mpStat(neutral, nil, nil)
	if couv == 13 && cv <= mpCVSeuilTemoin {
		t.Logf("TEMOIN : EXIGENCE TENUE (13/13, CV %.3f <= %.2f)", cv, mpCVSeuilTemoin)
	} else {
		t.Errorf("TEMOIN CASSE : couverture %d/13, CV %.3f (exige 13/13 et <= %.2f) — la "+
			"lecture meche pausable est fausse", couv, cv, mpCVSeuilTemoin)
	}

	husky := mpChargerGroupe(t, cache, []struct {
		id   string
		exps []int32
	}{{"1c01e34f", []int32{150546, 273787, 335637, 400853}}})
	mpVerdict(t, "TEMOIN HUSKY RAID — 4 explosions", husky, nil)
	// L'EXIGENCE HUSKY EST JUGEE, PAS SEULEMENT IMPRIMEE (ajout du 2026-09-04, avec le portage
	// en production) : 4/4 au chiffre pres, meme plafond de dispersion que Neutral.
	couvH, cvH, _ := mpStat(husky, nil, nil)
	if couvH == 4 && cvH <= mpCVSeuilTemoin {
		t.Logf("TEMOIN HUSKY : EXIGENCE TENUE (4/4, CV %.3f <= %.2f)", cvH, mpCVSeuilTemoin)
	} else {
		t.Errorf("TEMOIN HUSKY CASSE : couverture %d/4, CV %.3f (exige 4/4 et <= %.2f) — la "+
			"lecture meche pausable est fausse", couvH, cvH, mpCVSeuilTemoin)
	}
}

// mpChargerGroupe charge et digere une liste de films (id + explosions).
func mpChargerGroupe(t *testing.T, cache string, liste []struct {
	id   string
	exps []int32
},
) []*mpFilm {
	t.Helper()
	var films []*mpFilm
	for _, f := range liste {
		series, ok := obCharger(t, cache, f.id)
		if !ok {
			t.Fatalf("%s : film indispensable absent", f.id)
		}
		d := mpDigerer(f.id, series, f.exps)
		films = append(films, d)
		t.Logf("%-9s : %d armement(s), %d slot(s) a pauses, lectures de %d a %d ms",
			f.id, len(d.armements), len(d.pauses), d.min, d.max)
	}
	return films
}

// mpDigerer segmente les series d'un film et classe armements et pauses.
func mpDigerer(id string, series map[uint32][]ti12Ech, exps []int32) *mpFilm {
	f := &mpFilm{id: id, exps: exps, pauses: map[uint32][]mpPause{},
		min: math.MaxInt32, max: math.MinInt32}
	for slot, s := range series {
		if s[0].tMS < f.min {
			f.min = s[0].tMS
		}
		if s[len(s)-1].tMS > f.max {
			f.max = s[len(s)-1].tMS
		}
		for _, g := range obSegmenter(slot, s) {
			switch {
			case mpEstArmement(g):
				f.armements = append(f.armements, mpArmement{slot, g.t1})
			case mpEstPause(g):
				f.pauses[slot] = append(f.pauses[slot], mpPause{g.t0, g.t1})
			}
		}
	}
	sort.Slice(f.armements, func(i, j int) bool {
		if f.armements[i].finMS != f.armements[j].finMS {
			return f.armements[i].finMS < f.armements[j].finMS
		}
		return f.armements[i].slot < f.armements[j].slot
	})
	for _, p := range f.pauses {
		sort.Slice(p, func(i, j int) bool { return p[i].t0 < p[j].t0 })
	}
	return f
}

// mpEstArmement dit si un segment finit plein au sens de la lecture — PAR LE PREDICAT DE
// PRODUCTION (`NavpointSegment.EndsAtSummit`, porte le 2026-09-04 depuis ce fichier).
func mpEstArmement(g obSeg) bool { return g.nav().EndsAtSummit() }

// mpEstPause dit si un segment est une tenue de desarmement au sens de la lecture — PAR LE
// PREDICAT DE PRODUCTION (`NavpointSegment.IsDisarmHold`).
func mpEstPause(g obSeg) bool { return g.nav().IsDisarmHold() }

// mpDelai rend le delai corrige d'une cible : cible moins la derniere fin d'armement, moins les
// pauses du meme slot strictement entre les deux.
func mpDelai(f *mpFilm, cible int32) (float64, bool) {
	i := sort.Search(len(f.armements), func(k int) bool { return f.armements[k].finMS >= cible })
	if i == 0 {
		return 0, false
	}
	a := f.armements[i-1]
	d := cible - a.finMS
	if d <= 0 || d > tpSensMaxMS {
		return 0, false
	}
	d -= mpPausesEntre(f, a, cible)
	if d <= 0 {
		return 0, false
	}
	return float64(d), true
}

// mpPausesEntre somme les durees des pauses du slot d'un armement, strictement entre sa fin et
// la cible.
func mpPausesEntre(f *mpFilm, a mpArmement, cible int32) int32 {
	var p int32
	for _, x := range f.pauses[a.slot] {
		if x.t0 > a.finMS && x.t1 < cible {
			p += x.t1 - x.t0
		}
	}
	return p
}

// mpStat calcule (couverture, CV, mediane) sur les explosions que `garde` retient (nil = toutes),
// transformees par `dec` (nil = identite — la voie des decalages temoins).
func mpStat(films []*mpFilm, garde func(string, int32) bool, dec func(int32) int32,
) (int, float64, float64) {
	var delais []float64
	couv, total := 0, 0
	for _, f := range films {
		for _, e := range f.exps {
			if garde != nil && !garde(f.id, e) {
				continue
			}
			total++
			c := e
			if dec != nil {
				c = dec(e)
			}
			if d, ok := mpDelai(f, c); ok {
				couv++
				delais = append(delais, d)
			}
		}
	}
	med, cv := tpMedCV(delais)
	_ = total
	return couv, cv, med
}

// mpStatNulle tire les cibles uniformement dans l'etendue de chaque film, memes effectifs que
// les explosions retenues par `garde`.
func mpStatNulle(films []*mpFilm, garde func(string, int32) bool, rng *rand.Rand,
) (int, float64) {
	var delais []float64
	couv := 0
	for _, f := range films {
		span := f.max - f.min
		for _, e := range f.exps {
			if garde != nil && !garde(f.id, e) {
				continue
			}
			c := f.min + int32(rng.Int63n(int64(span)+1))
			if d, ok := mpDelai(f, c); ok {
				couv++
				delais = append(delais, d)
			}
		}
	}
	_, cv := tpMedCV(delais)
	return couv, cv
}

// mpVerdict publie la statistique reelle, les decalages, et la nulle d'un groupe.
func mpVerdict(t *testing.T, titre string, films []*mpFilm, garde func(string, int32) bool) {
	t.Helper()
	total := 0
	for _, f := range films {
		for _, e := range f.exps {
			if garde == nil || garde(f.id, e) {
				total++
			}
		}
	}
	couv, cv, med := mpStat(films, garde, nil)
	t.Logf("########## %s : couverture %d/%d, delai median %6.2f s, CV %5.3f",
		titre, couv, total, med/1000, cv)
	for _, dec := range []int32{45000, -45000, 120000} {
		d := dec
		c, v, m := mpStat(films, garde, func(e int32) int32 { return e + d })
		t.Logf("           decalage %+6d ms : couverture %d/%d, mediane %6.2f s, CV %5.3f",
			dec, c, total, m/1000, v)
	}
	rng := rand.New(rand.NewSource(tpGraine))
	pleins, aussiBien := 0, 0
	for i := 0; i < tpTirages; i++ {
		c, v := mpStatNulle(films, garde, rng)
		if c == total {
			pleins++
			if v <= cv {
				aussiBien++
			}
		}
	}
	t.Logf("           nulle (%d tirages, graine %d) : %d pleins, %d aussi bien (%.1f %%)",
		tpTirages, tpGraine, pleins, aussiBien, 100*float64(aussiBien)/float64(tpTirages))
	ok := couv == total && cv <= tpCVSeuil && 100*float64(aussiBien)/float64(tpTirages) < tpNulSeuilPc
	verdict := "NON RESOLU sous la regle"
	if ok {
		verdict = "RESOLU sous la regle (couverture pleine, CV <= 0,20, nulle < 1 %)"
	}
	t.Logf("           VERDICT %s : %s", titre, verdict)
}

// mpDetail imprime une explosion : l'armement retenu, ses pauses, le delai brut et corrige.
func mpDetail(t *testing.T, f *mpFilm, e int32) {
	t.Helper()
	i := sort.Search(len(f.armements), func(k int) bool { return f.armements[k].finMS >= e })
	if i == 0 {
		t.Logf("    %s %7d ms : AUCUN armement avant", f.id, e)
		return
	}
	a := f.armements[i-1]
	brut := e - a.finMS
	pauses := ""
	var somme int32
	for _, x := range f.pauses[a.slot] {
		if x.t0 > a.finMS && x.t1 < e {
			somme += x.t1 - x.t0
			pauses += fmt.Sprintf(" [%.1fs..%.1fs %0.1fs]", float64(x.t0)/1000,
				float64(x.t1)/1000, float64(x.t1-x.t0)/1000)
		}
	}
	if pauses == "" {
		pauses = " (aucune)"
	}
	sansPorteur := ""
	if mpSansPorteur[f.id] == e {
		sansPorteur = " · SANS PORTEUR (a5SansPorteur)"
	}
	t.Logf("    %s %7d ms : armement slot %d fin %.1fs · brut %6.1fs · pauses%s · "+
		"CORRIGE %6.1fs%s", f.id, e, a.slot, float64(a.finMS)/1000, float64(brut)/1000,
		pauses, float64(brut-somme)/1000, sansPorteur)
}
