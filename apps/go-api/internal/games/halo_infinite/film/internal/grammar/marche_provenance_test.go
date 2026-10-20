package grammar

// marche_provenance_test.go — T3 DE LA SPECIFICATION DE LA REPRESENTATION INTERMEDIAIRE (ADR 0037
// IR-1, IR-4) : toute lecture d etat de mouvement et de tir continu cite une etendue qui existe dans
// la structure. Une lecture d etat de mouvement publiee pendant la marche d un paquet a, dans ce
// paquet, le record de son slot et l occurrence traversee du composant publie — INTERPRETEE quand le
// canal des etats l interprete, delimitee sinon ; une entree de controle rendue au tir continu a,
// dans la vue C du paquet, le tour de controle de son index. Les canaux sont ceux de la production
// ([ScanMarcheDesTrames]), distribues sur la marche et doubles de temoins qui retiennent ce que leurs
// crochets recoivent.

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// publicationDEtat : une lecture d etat de mouvement, telle que le crochet la recoit.
type publicationDEtat struct {
	comp EtatMouvementComposant
	slot uint32
}

// TestChaqueLectureCiteUneEtendueDeLaStructure : T3, sur les bobines du depot qui portent des
// trames delta.
// MUTATION — rendre les records d un paquet sans leurs composants (`rangerLesRecords` qui n ajoute
// rien a l arene) : chaque lecture d etat perd son occurrence, ROUGE.
// MUTATION — le canal des etats sans la vitesse de translation dans ses interets
// ([movementStateScanner.Interets]) : ses lectures citent une occurrence delimitee, ROUGE ; la
// posture ajoutee a ses interets : ses lectures citent une occurrence interpretee, ROUGE.
func TestChaqueLectureCiteUneEtendueDeLaStructure(t *testing.T) {
	var etats, interpretees, controles int
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
		etats, interpretees, controles = etats+e.cites, interpretees+e.interpretees, controles+c
	}
	t.Logf("%d lecture(s) d etat de mouvement (%d interpretee(s)) et %d entree(s) de controle citees",
		etats, interpretees, controles)
	if interpretees == 0 || etats == interpretees || controles == 0 {
		t.Fatalf("%d lecture(s) d etat dont %d interpretee(s), %d entree(s) de controle : le test ne prouve "+
			"rien", etats, interpretees, controles)
	}
}

// citerLesLecturesDUneBobine distribue les deux canaux de production d un film, doubles de leurs
// temoins, qui verifient trame par trame que chaque publication cite une etendue de la structure.
// Rend le temoin des etats et le nombre d entrees de controle citees.
func citerLesLecturesDUneBobine(t *testing.T, nom string, fc *FilmContext) (*temoinDesEtats, int) {
	t.Helper()
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("%s : registre : %v", nom, err)
	}
	arch, err := fc.bipedArchetype()
	if err != nil {
		t.Fatalf("%s : archetype bipede : %v", nom, err)
	}
	var st types.MovementStateStats
	var tir types.ContinuousFireStats
	etats := &temoinDesEtats{movementStateScanner: nouveauCanalDesEtats(&st, reg, arch), t: t, nom: nom, reg: reg}
	controle := &temoinDuControle{collecteurTirContinu: nouveauCollecteurTirContinu(&tir), t: t, nom: nom}
	if err := Distribuer(fc, etats, controle); err != nil {
		t.Fatalf("%s : %v", nom, err)
	}
	return etats, controle.cites
}

// temoinDesEtats double le canal des etats de mouvement de production : il retient ce que le crochet
// du canal recoit pendant la marche d une trame, et le cite dans la structure de la trame.
type temoinDesEtats struct {
	*movementStateScanner
	t                   *testing.T
	nom                 string
	reg                 *Registry
	publiees            []publicationDEtat
	cites, interpretees int
}

// Brancher pose le crochet du canal, double d une retenue.
func (e *temoinDesEtats) Brancher(obs *Observation, m *MarcheDistribuee) {
	e.movementStateScanner.Brancher(obs, m)
	production := obs.EtatMouvementHook
	if production == nil {
		return // un film qui ne transmet pas les etats : rien a citer
	}
	obs.EtatMouvementHook = func(c EtatMouvementComposant, slot uint32, v []uint64) {
		e.publiees = append(e.publiees, publicationDEtat{comp: c, slot: slot})
		production(c, slot, v)
	}
}

// Trame cite les publications de la trame, puis la rend au canal.
func (e *temoinDesEtats) Trame(p *lecture.Paquet) {
	for _, pub := range e.publiees {
		interpretee, ok := citeUneOccurrence(e.reg, p, pub)
		if !ok {
			e.t.Errorf("%s chunk %d paquet %d : la lecture %s du slot %d ne cite aucune occurrence dans "+
				"l etat ou le canal la met", e.nom, p.Chunk, p.Index, pub.comp, pub.slot)
		}
		if interpretee {
			e.interpretees++
		}
	}
	e.cites += len(e.publiees)
	e.publiees = e.publiees[:0]
	e.movementStateScanner.Trame(p)
}

// temoinDuControle double le collecteur du tir continu de production : il retient le verdict que le
// crochet du collecteur recoit pendant la marche d une trame, et en cite les entrees dans la vue C
// de la trame.
type temoinDuControle struct {
	*collecteurTirContinu
	t       *testing.T
	nom     string
	verdict LectureVueC
	cites   int
}

// Brancher pose le crochet du collecteur, double d une retenue.
func (c *temoinDuControle) Brancher(obs *Observation, m *MarcheDistribuee) {
	c.collecteurTirContinu.Brancher(obs, m)
	production := obs.VueControleHook
	obs.VueControleHook = func(l LectureVueC) {
		c.verdict = l
		production(l)
	}
}

// Trame cite les entrees de controle de la trame, puis la rend au collecteur.
func (c *temoinDuControle) Trame(p *lecture.Paquet) {
	if c.verdict.Fermee {
		c.cites += citerLesEntrees(c.t, fmt.Sprintf("%s chunk %d paquet %d", c.nom, p.Chunk, p.Index), p,
			c.verdict)
	}
	c.verdict = LectureVueC{}
	c.collecteurTirContinu.Trame(p)
}

// citeUneOccurrence dit si un record du slot publie porte, dans le paquet, une occurrence du
// composant publie ([nomDeBase]) dans l etat ou le canal des etats la met — interpretee quand le
// record est un bipede et que le crochet du canal garde l etat ([etatInterprete]), delimitee sinon —,
// et si elle est interpretee.
func citeUneOccurrence(reg *Registry, p *lecture.Paquet, e publicationDEtat) (interpretee, ok bool) {
	voulu := nomDeBase(e.comp.String())
	for _, r := range p.Records {
		arch, connu := reg.Archetype(int(r.TI))
		if r.Vie.Slot != e.slot || !connu {
			continue
		}
		attendu := lecture.EtatDelimite
		if r.TI == BipedTypeIndex && etatInterprete(e.comp) {
			attendu = lecture.EtatInterprete
		}
		for _, c := range p.Comps[r.Comps[0]:r.Comps[1]] {
			if c.Etat == attendu && int(c.Index) < len(arch.Components) &&
				nomDeBase(arch.Components[c.Index]) == voulu {
				return attendu == lecture.EtatInterprete, true
			}
		}
	}
	return false, false
}

// etatInterprete dit si le crochet du canal des etats garde un etat publie
// ([movementStateScanner.recevoir]) : tous, sauf la posture et le controle d unite, que le meme
// deserialiseur publie. C est la SPECIFICATION que le test oppose aux interets du canal.
func etatInterprete(c EtatMouvementComposant) bool {
	return c != EtatPosture && c != EtatControleUnite
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
