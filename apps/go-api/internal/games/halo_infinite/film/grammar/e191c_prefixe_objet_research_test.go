//go:build research

package grammar

// e191c_prefixe_objet_research_test.go — LOT 1.9.1 bis, PAS 2 : LA FERMETURE DES CINQ
// ARCHETYPES « OBJET DU MONDE », ET LE TEST DE LA DICHOTOMIE POSEE AU PAS 1.
//
// # POURQUOI CET INSTRUMENT EXISTE
//
// Le pas 1 a mesure que les archetypes portant `object-position-component` ferment 0,85 % de
// leurs records d image-cle contre 44,29 % pour les autres, et en a conclu que le defaut etait
// dans le PREFIXE OBJET (D1 (1.9.1 bis)). Le pas 2 devait relire les six composants nommes chez
// l ecrivain, puis re-mesurer la fermeture des cinq archetypes ENSEMBLE. La relecture est faite
// (les six sont bit-exacts, aucune largeur ne change), donc la fermeture ne peut pas bouger —
// et la question devient : LA DICHOTOMIE DU PAS 1 EST-ELLE LA BONNE VARIABLE ?
//
// # LES DEUX MESURES
//
//	[A] LA FERMETURE DES CINQ (ti=37, 38, 41, 42, 43), bobine par bobine et en cumul. C est le
//	    tableau « avant / apres » du pas 2 : il se rejoue apres chaque composant porte.
//	[B] LE CONFONDANT. La meme population, classee cette fois par le NOMBRE DE COMPOSANTS de
//	    l archetype au registre. Si les archetypes qui ferment sont les archetypes SIMPLES —
//	    quel que soit leur prefixe —, alors « porter object-position-component » n est pas la
//	    cause mais une correlation, et la cible du lot doit se re-formuler. La mesure publie
//	    les deux classements cote a cote pour que le lecteur tranche sur pieces.
//
// CE QUE CA NE PROUVE PAS : une correlation ne nomme aucun defaut. L instrument DECLASSE une
// hypothese ou la laisse debout ; c est l ecrivain (Ghidra) qui donne une grammaire (D13).
//
// LECTURE SEULE, sans garde d environnement : les 7 bobines par build sont VERSIONNEES
// (`../replay/testdata/minifilm_*`), comme le ratchet 0.A.3.
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191cPrefixeObjet$' -v -count=1

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// e191cCinq sont les cinq archetypes « objet du monde » du perimetre re-cadre du lot : ceux
// que le pas 1 a mesures comme portant `object-position-component`.
var e191cCinq = []int{37, 38, 41, 42, 43}

// e191cFermeture est le couple (fermes, bornes) d une population.
type e191cFermeture struct{ Fermes, Bornes int }

// pct rend le taux de fermeture en pourcentage, 0 sur une population vide.
func (f e191cFermeture) pct() float64 {
	if f.Bornes == 0 {
		return 0
	}
	return 100 * float64(f.Fermes) / float64(f.Bornes)
}

// add cumule une autre mesure.
func (f *e191cFermeture) add(o e191cFermeture) {
	f.Fermes, f.Bornes = f.Fermes+o.Fermes, f.Bornes+o.Bornes
}

// e191cMesure est la lecture d UNE bobine : la fermeture par archetype et le nombre de
// composants que le registre de CETTE bobine donne a chaque archetype.
type e191cMesure struct {
	Court     string
	Fermeture map[int]e191cFermeture
	NbComps   map[int]int
	Prefixe   map[int]bool
}

// e191cLire charge une bobine et rend sa mesure.
func e191cLire(t *testing.T, court string) e191cMesure {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre %s : %v", court, err)
	}
	stats, err := KeyframeClosure(fc)
	if err != nil {
		t.Fatalf("KeyframeClosure %s : %v", court, err)
	}
	m := e191cMesure{Court: court, Fermeture: map[int]e191cFermeture{},
		NbComps: map[int]int{}, Prefixe: map[int]bool{}}
	for ti, s := range stats {
		i := int(ti) //nolint:gosec // ti est un index d archetype, jamais negatif
		m.Fermeture[i] = e191cFermeture{Fermes: s.Closed, Bornes: s.Total}
		if arch, ok := reg.Archetype(i); ok {
			m.NbComps[i] = len(arch.Components)
			m.Prefixe[i] = e191bPorteLePrefixe(reg, i)
		}
	}
	return m
}

// TestE191cPrefixeObjet publie les deux mesures du pas 2.
func TestE191cPrefixeObjet(t *testing.T) {
	mesures := make([]e191cMesure, 0, 7)
	for _, court := range closureMiniFilms() {
		mesures = append(mesures, e191cLire(t, court))
	}
	t.Logf("######## PAS 2 — FERMETURE DES CINQ ARCHETYPES OBJET ET TEST DE LA DICHOTOMIE ########")
	t.Logf("")
	t.Logf("==== [A] FERMETURE DES CINQ ARCHETYPES OBJET (ti=37, 38, 41, 42, 43) ====")
	e191cLogCinq(t, mesures)
	t.Logf("")
	t.Logf("==== [B] LE CONFONDANT : fermeture PAR NOMBRE DE COMPOSANTS, tous archetypes ====")
	e191cLogConfondant(t, mesures)
	t.Logf("")
	t.Logf("==== [C] LES COMPOSANTS BLANCHIS PAR UNE FERMETURE, ET LES SUSPECTS DE ti=37 ====")
	e191cLogBlanchis(t)
	t.Logf("")
	t.Logf("==== [D] LA SONDE FINE : les archetypes de moins de %d composants ====", e191cPetitMax+1)
	e191cLogPetits(t)
}

// e191cSeuilBlanchi est le taux de fermeture a partir duquel un archetype BLANCHIT ses
// composants : au-dessus, ses records atterrissent sur la frontiere, donc TOUTES les largeurs
// qu il consomme sont justes dans CE contexte. 50 % est ecrit ici parce que la mesure du
// 2026-09-15 ne laisse aucun archetype entre 1,49 % et 68,57 % — le seuil ne departage rien
// d ambigu, il nomme une population deja separee.
const e191cSeuilBlanchi = 50.0

// e191cLogBlanchis publie, pour les 31 composants de ti=37, lesquels apparaissent dans un
// archetype QUI FERME (donc dont la largeur est prouvee juste ailleurs) et lesquels
// n apparaissent nulle part ailleurs qu au milieu de records qui ne ferment pas.
//
// C EST UNE PREUVE BORNANTE, PAS UNE CORRELATION : un record qui atterrit exactement sur la
// frontiere a lu JUSTE tous ses composants ; un composant present dans un tel record est donc
// blanchi dans ce contexte, et le defaut de ti=37 est a chercher dans les autres.
func e191cLogBlanchis(t *testing.T) {
	t.Helper()
	blanchis, cible := map[string]int{}, []string{}
	for _, court := range closureMiniFilms() {
		dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
		film, err := source.LoadDir(dir, nil)
		if err != nil {
			t.Fatalf("LoadDir %s : %v", dir, err)
		}
		fc := NewFilmContext(film)
		reg, err := fc.Registry()
		if err != nil {
			t.Fatalf("registre %s : %v", court, err)
		}
		stats, err := KeyframeClosure(fc)
		if err != nil {
			t.Fatalf("KeyframeClosure %s : %v", court, err)
		}
		for ti, s := range stats {
			i := int(ti) //nolint:gosec // index d archetype
			arch, ok := reg.Archetype(i)
			if !ok {
				continue
			}
			if i == e191bTI && len(cible) == 0 {
				cible = append(cible, arch.Components...)
			}
			f := e191cFermeture{Fermes: s.Closed, Bornes: s.Total}
			if f.Bornes > 0 && f.pct() >= e191cSeuilBlanchi {
				for _, n := range arch.Components {
					blanchis[n]++
				}
			}
		}
	}
	e191cLogListeSuspects(t, cible, blanchis)
}

// e191cLogListeSuspects colle les deux listes : les composants de ti=37 blanchis ailleurs et
// ceux qui ne le sont pas.
func e191cLogListeSuspects(t *testing.T, cible []string, blanchis map[string]int) {
	t.Helper()
	nbBlanchis := 0
	for i, n := range cible {
		if blanchis[n] > 0 {
			nbBlanchis++
			t.Logf("  i%-2d BLANCHI (%3d records fermes ailleurs)  %s", i, blanchis[n], n)
			continue
		}
		t.Logf("  i%-2d SUSPECT  (jamais dans un record ferme)   %s", i, n)
	}
	t.Logf("  BILAN ti=%d : %d composants blanchis, %d suspects sur %d",
		e191bTI, nbBlanchis, len(cible)-nbBlanchis, len(cible))
}

// e191cLogCinq publie le tableau [A] : les cinq archetypes, bobine par bobine puis en cumul.
func e191cLogCinq(t *testing.T, mesures []e191cMesure) {
	t.Helper()
	parTI := map[int]*e191cFermeture{}
	for _, ti := range e191cCinq {
		parTI[ti] = &e191cFermeture{}
	}
	for _, m := range mesures {
		cols := make([]string, 0, len(e191cCinq))
		for _, ti := range e191cCinq {
			f := m.Fermeture[ti]
			parTI[ti].add(f)
			cols = append(cols, fmt.Sprintf("ti=%-2d %4d/%-5d", ti, f.Fermes, f.Bornes))
		}
		t.Logf("  %-10s %s", m.Court, joinCols(cols))
	}
	tout := e191cFermeture{}
	for _, ti := range e191cCinq {
		f := *parTI[ti]
		tout.add(f)
		t.Logf("  CUMUL ti=%-2d : %5d fermes / %5d bornes (%.2f %%)", ti, f.Fermes, f.Bornes, f.pct())
	}
	t.Logf("  CUMUL LES CINQ ENSEMBLE : %5d fermes / %5d bornes (%.2f %%)", tout.Fermes, tout.Bornes, tout.pct())
}

// joinCols colle des colonnes deja formatees, separees de deux espaces.
func joinCols(cols []string) string {
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += "  "
		}
		out += c
	}
	return out
}

// e191cLogConfondant publie le tableau [B] : la MEME population classee par nombre de
// composants, avec le detail par archetype, et les deux dichotomies cote a cote.
func e191cLogConfondant(t *testing.T, mesures []e191cMesure) {
	t.Helper()
	parTI := map[int]*e191cFermeture{}
	nbComps, prefixe := map[int]int{}, map[int]bool{}
	for _, m := range mesures {
		for ti, f := range m.Fermeture {
			if parTI[ti] == nil {
				parTI[ti] = &e191cFermeture{}
			}
			parTI[ti].add(f)
			if n, ok := m.NbComps[ti]; ok {
				nbComps[ti], prefixe[ti] = n, m.Prefixe[ti]
			}
		}
	}
	tis := make([]int, 0, len(parTI))
	for ti := range parTI {
		tis = append(tis, ti)
	}
	sort.Slice(tis, func(i, j int) bool { return nbComps[tis[i]] < nbComps[tis[j]] })
	t.Logf("  %-6s %-7s %-9s %14s %9s", "ti", "nbComps", "prefixeObj", "fermes/bornes", "taux")
	for _, ti := range tis {
		f := *parTI[ti]
		t.Logf("  ti=%-3d %-7d %-9v %7d/%-6d %8.2f %%", ti, nbComps[ti], prefixe[ti], f.Fermes, f.Bornes, f.pct())
	}
	e191cLogDichotomies(t, parTI, nbComps, prefixe)
}

// e191cSeuilSimple est la frontiere « archetype SIMPLE / COMPLEXE » du classement [B] : le
// nombre de composants au registre. 12 est le premier palier qui separe les archetypes qui
// ferment de ceux qui ne ferment pas dans la mesure du 2026-09-15 ; il est ECRIT ICI pour que
// le lecteur voie sur quel seuil la dichotomie est calculee, pas pour l optimiser.
const e191cSeuilSimple = 12

// e191cLogDichotomies met les deux classements cote a cote : par prefixe (l hypothese du
// pas 1) et par complexite (l hypothese concurrente).
func e191cLogDichotomies(t *testing.T, parTI map[int]*e191cFermeture, nbComps map[int]int, prefixe map[int]bool) {
	t.Helper()
	var avecP, sansP, simple, complexe e191cFermeture
	for ti, f := range parTI {
		if prefixe[ti] {
			avecP.add(*f)
		} else {
			sansP.add(*f)
		}
		if nbComps[ti] < e191cSeuilSimple {
			simple.add(*f)
		} else {
			complexe.add(*f)
		}
	}
	t.Logf("")
	t.Logf("  DICHOTOMIE 1 (hypothese du pas 1) — porte object-position-component :")
	t.Logf("    AVEC : %5d / %5d (%.2f %%)   SANS : %5d / %5d (%.2f %%)",
		avecP.Fermes, avecP.Bornes, avecP.pct(), sansP.Fermes, sansP.Bornes, sansP.pct())
	t.Logf("  DICHOTOMIE 2 (hypothese concurrente) — moins de %d composants au registre :", e191cSeuilSimple)
	t.Logf("    SIMPLE : %5d / %5d (%.2f %%)   COMPLEXE : %5d / %5d (%.2f %%)",
		simple.Fermes, simple.Bornes, simple.pct(), complexe.Fermes, complexe.Bornes, complexe.pct())
}

// e191cPetitMax borne la taille des archetypes de la sonde [D] : au-dela de quatre composants,
// un residu ne designe plus un champ, il designe une derive.
const e191cPetitMax = 4

// e191cLogPetits publie la SONDE LA PLUS FINE du lot : les archetypes de moins de cinq
// composants, avec leur liste de composants, la presence d un etat par defaut porte, et
// l HISTOGRAMME du residu `EndBit - Want`.
//
// POURQUOI ELLE TRANCHE. `ti=22` et `ti=25` portent UN SEUL composant chacun ; le premier ferme
// 113 records sur 113, le second zero sur 113. Le cadre (en-tete 108 + n1 + etat par defaut +
// n2) est le meme pour les deux : ce qui les separe est donc l etat par defaut de l archetype
// OU son unique composant, et un residu CONSTANT dit de combien de bits. Sur un archetype a
// trente composants la meme mesure ne dirait rien.
func e191cLogPetits(t *testing.T) {
	t.Helper()
	type petit struct {
		Comps   []string
		Residus map[int]int
		Fermes  int
		Bornes  int
		Desync  int
	}
	tab := map[int]*petit{}
	for _, court := range closureMiniFilms() {
		dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
		film, err := source.LoadDir(dir, nil)
		if err != nil {
			t.Fatalf("LoadDir %s : %v", dir, err)
		}
		fc := NewFilmContext(film)
		reg, err := fc.Registry()
		if err != nil {
			t.Fatalf("registre %s : %v", court, err)
		}
		e191cScannerPetits(t, fc, reg, func(ti int, comps []string, residu, desync int) {
			p := tab[ti]
			if p == nil {
				p = &petit{Comps: comps, Residus: map[int]int{}}
				tab[ti] = p
			}
			p.Bornes++
			switch {
			case desync >= 0:
				p.Desync++
			case residu == 0:
				p.Fermes++
				p.Residus[0]++
			default:
				p.Residus[residu]++
			}
		})
	}
	tis := make([]int, 0, len(tab))
	for ti := range tab {
		tis = append(tis, ti)
	}
	sort.Ints(tis)
	for _, ti := range tis {
		p := tab[ti]
		_, porte := defaultStateDeserByTI[uint32(ti)] //nolint:gosec // index d archetype
		t.Logf("  ti=%-3d %d comp %-6v etatParDefautPorte=%-5v  %4d fermes / %4d bornes, %d desync",
			ti, len(p.Comps), "", porte, p.Fermes, p.Bornes, p.Desync)
		for _, n := range p.Comps {
			t.Logf("          comp : %s", n)
		}
		t.Logf("          residus (EndBit-Want) : %s", e191cTopResidus(p.Residus))
	}
}

// e191cScannerPetits rejoue la marche d etat complet sur les records des archetypes de moins
// de cinq composants et rend, par record, le residu et l index de desynchronisation.
func e191cScannerPetits(t *testing.T, fc *FilmContext, reg *Registry, f func(ti int, comps []string, residu, desync int)) {
	t.Helper()
	for _, num := range fc.ChunkNumbers() {
		data, packets, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range packets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, b := range keyframeBornes(pay) {
				arch, ok := reg.Archetype(b.TI)
				if !ok || len(arch.Components) > e191cPetitMax {
					continue
				}
				tr := WalkKeyframeFullState(pay, b.Bit, reg, contexteDInstrument())
				f(b.TI, arch.Components, tr.EndBit-b.Want, tr.DesyncAt)
			}
		}
	}
}

// e191cTopResidus colle les cinq residus les plus frequents, du plus frequent au moins.
func e191cTopResidus(m map[int]int) string {
	type kv struct{ r, n int }
	l := make([]kv, 0, len(m))
	for r, n := range m {
		l = append(l, kv{r, n})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].n != l[j].n {
			return l[i].n > l[j].n
		}
		return l[i].r < l[j].r
	})
	out := fmt.Sprintf("%d valeurs distinctes ;", len(l))
	for i, e := range l {
		if i >= 5 {
			break
		}
		out += fmt.Sprintf(" %+d x%d", e.r, e.n)
	}
	return out
}

// e191cPaires sont les COUPLES D ARCHETYPES DE MEME TAILLE dont l un ferme et l autre pas.
// C est la sonde la plus discriminante du lot : `ti=18` et `ti=19` portent 32 composants
// chacun et 113 records chacun sur les sept bobines, et le premier ferme 113/113 quand le
// second ferme 0. Le cadre (en-tete, `n1`, etat par defaut, `n2`) est le meme ; ce qui les
// separe est donc la LISTE DES COMPOSANTS, et le diff la nomme.
var e191cPaires = [][2]int{{18, 19}, {22, 25}, {14, 20}, {17, 3}}

// TestE191cPairesDiscriminantes colle, pour chaque couple, les composants propres a chacun.
// Les composants de l archetype QUI FERME sont prouves justes ; ceux de l autre, non.
func TestE191cPairesDiscriminantes(t *testing.T) {
	t.Logf("######## PAS 2 BIS — LES COUPLES DE MEME TAILLE, L UN FERME, L AUTRE PAS ########")
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+closureMiniFilms()[0])
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	reg, err := NewFilmContext(film).Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	for _, p := range e191cPaires {
		e191cLogPaire(t, reg, p[0], p[1])
	}
}

// e191cLogPaire colle le diff des composants de deux archetypes.
func e191cLogPaire(t *testing.T, reg *Registry, ferme, casse int) {
	t.Helper()
	a, okA := reg.Archetype(ferme)
	b, okB := reg.Archetype(casse)
	if !okA || !okB {
		t.Logf("  ti=%d / ti=%d : archetype absent du registre", ferme, casse)
		return
	}
	dansA := map[string]bool{}
	for _, n := range a.Components {
		dansA[n] = true
	}
	t.Logf("  ---- ti=%d (FERME, %d comp) contre ti=%d (NE FERME PAS, %d comp) ----",
		ferme, len(a.Components), casse, len(b.Components))
	communs := 0
	for i, n := range b.Components {
		if dansA[n] {
			communs++
			continue
		}
		t.Logf("     PROPRE a ti=%d : i%-2d %s", casse, i, n)
	}
	t.Logf("     %d composants communs sur %d", communs, len(b.Components))
}
