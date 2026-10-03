package grammar

// marche_provenance_test.go — T3 DE LA SPECIFICATION DE LA REPRESENTATION INTERMEDIAIRE (ADR 0037
// IR-1) : toute lecture d etat de mouvement et de tir continu cite une etendue qui existe dans la
// structure. Une lecture d etat de mouvement publiee pendant la marche d un paquet a, dans ce
// paquet, le record de son slot et l occurrence traversee du composant publie ; une entree de
// controle rendue au tir continu a, dans la vue C du paquet, le tour de controle de son index.

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// publicationDEtat : une lecture d etat de mouvement, telle que le crochet la recoit.
type publicationDEtat struct {
	comp EtatMouvementComposant
	slot uint32
}

// TestChaqueLectureCiteUneEtendueDeLaStructure : T3, sur les bobines du depot qui portent des
// trames delta. MUTATION — rendre les records d un paquet sans leurs composants (`rangerLesRecords`
// qui n ajoute rien a l arene) : chaque lecture d etat perd son occurrence, ROUGE.
func TestChaqueLectureCiteUneEtendueDeLaStructure(t *testing.T) {
	etats, controles := 0, 0
	for _, bo := range frameClosureBobines() {
		if _, err := os.Stat(bo.dir); err != nil {
			t.Fatalf("bobine absente (%s) : %v", bo.dir, err)
		}
		film, err := source.LoadDir(bo.dir, nil)
		if err != nil {
			t.Fatalf("LoadDir %s : %v", bo.dir, err)
		}
		if _, ok := FilmRegistryChunk(film); !ok {
			continue
		}
		e, c := citerLesLecturesDUneBobine(t, bo.nom, contexteDeBobine(film))
		etats, controles = etats+e, controles+c
	}
	t.Logf("%d lecture(s) d etat de mouvement et %d entree(s) de controle citees", etats, controles)
	if etats == 0 || controles == 0 {
		t.Fatalf("%d lecture(s) d etat, %d entree(s) de controle : le test ne prouve rien", etats, controles)
	}
}

// citerLesLecturesDUneBobine marche les trames d un film sous une observation qui retient ce que
// la marche publie, et verifie, trame par trame, que chaque publication cite une etendue de la
// structure. Rend le nombre de lectures d etat et d entrees de controle citees.
func citerLesLecturesDUneBobine(t *testing.T, nom string, fc *FilmContext) (etats, controles int) {
	t.Helper()
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("%s : registre : %v", nom, err)
	}
	var publiees []publicationDEtat
	var verdict LectureVueC
	obs := NouvelleObservation()
	obs.EtatMouvementHook = func(c EtatMouvementComposant, slot uint32, _ []uint64) {
		publiees = append(publiees, publicationDEtat{comp: c, slot: slot})
	}
	obs.VueControleHook = func(l LectureVueC) { verdict = l }
	mt, err := fc.nouveauMarcheurDesTrames(obs)
	if err != nil {
		t.Fatalf("%s : %v", nom, err)
	}
	mt.parcourir(func(tl *trameLue) bool {
		p := tl.paquet
		for _, e := range publiees {
			if !citeUneOccurrence(reg, p, e) {
				t.Errorf("%s chunk %d paquet %d : la lecture %s du slot %d ne cite aucune occurrence traversee",
					nom, p.Chunk, p.Index, e.comp, e.slot)
			}
		}
		etats += len(publiees)
		publiees = publiees[:0]
		if verdict.Fermee {
			controles += citerLesEntrees(t, fmt.Sprintf("%s chunk %d paquet %d", nom, p.Chunk, p.Index), p, verdict)
		}
		verdict = LectureVueC{}
		return true
	})
	return etats, controles
}

// citeUneOccurrence dit si un record du slot publie porte, dans le paquet, une occurrence
// traversee du composant publie ([nomDeBase]).
func citeUneOccurrence(reg *Registry, p *lecture.Paquet, e publicationDEtat) bool {
	voulu := nomDeBase(e.comp.String())
	for _, r := range p.Records {
		arch, ok := reg.Archetype(int(r.TI))
		if r.Vie.Slot != e.slot || !ok {
			continue
		}
		for _, c := range p.Comps[r.Comps[0]:r.Comps[1]] {
			if c.Etat != lecture.EtatInfranchissable && int(c.Index) < len(arch.Components) &&
				nomDeBase(arch.Components[c.Index]) == voulu {
				return true
			}
		}
	}
	return false
}

// citerLesEntrees verifie que chaque entree de controle d une vue C fermee a, dans la structure, le
// tour de controle de son index, dans l ordre du flux. Rend le nombre d entrees citees.
func citerLesEntrees(t *testing.T, ou string, p *lecture.Paquet, l LectureVueC) int {
	t.Helper()
	var tours []lecture.EntreeVueC
	for _, e := range p.VueC.Entrees {
		if e.Kind == kindVueCControle {
			tours = append(tours, e)
		}
	}
	if len(tours) != len(l.Entrees) {
		t.Errorf("%s : %d entree(s) de controle rendue(s), %d tour(s) de controle dans la structure", ou, len(l.Entrees), len(tours))
		return 0
	}
	for k, e := range l.Entrees {
		if int(tours[k].Index) != e.Index || tours[k].Bits == 0 {
			t.Errorf("%s : entree %d d index %d, tour %+v", ou, k, e.Index, tours[k])
		}
	}
	return len(l.Entrees)
}

// nomDeBase rend le nom d un composant sans ses suffixes : le registre nomme un meme composant avec
// ou sans `-component`, et la vitesse de translation que publie [EtatVitesse] est lue par le
// variant `-dynamic-precision` (`dispatch_object.go`, i1), que l etiquette de la publication ne
// nomme pas.
func nomDeBase(nom string) string {
	return strings.TrimSuffix(strings.TrimSuffix(nom, "-component"), "-dynamic-precision")
}
