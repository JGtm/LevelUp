package filmdec

// default_state_n2_constant_test.go — `n2` EST CONSTANT QUAND L'ETAT PAR DEFAUT FAIT LA BONNE
// LARGEUR (lot 1.3, 2026-09-14). PERMANENT : aucune garde d'environnement, il tourne en CI sur
// les sept bobines par build du lot 0.A.2.
//
// # L'ORACLE, ET POURQUOI IL EST GRATUIT
//
// `FUN_142e2bfd0` lit, autour de l'etat par defaut d'un record d'image-cle :
//
//	[108 bits d'en-tete] R(32) n1 | etat par defaut (largeur w) | R(32) n2 | composants
//
// `n1` et `n2` sont les deux TAILLES DE TAMPON que le jeu alloue pour l'archetype
// (`vtable[0x20]` et `vtable[0x10]` de `FUN_1408f1aa4`) : elles sont CONSTANTES par archetype et
// par build. `n1` se lit a une position FIXE (108) ; `n2` se lit APRES l'etat par defaut. Donc :
// une largeur d'etat par defaut fausse fait atterrir `n2` sur des bits quelconques, et sa
// DISPERSION le dit — sans capture live, sans base, sans oracle externe.
//
// # CE QUE CE TEST AFFIRME, ET CE QU'IL N'AFFIRME PAS
//
// AFFIRME : pour tout archetype dont l'etat par defaut est PORTE (`defaultStateDeserByTI`) et
// consomme une largeur FIXE sur la bobine, `n2` prend une seule valeur.
//
// N'AFFIRME PAS que la reciproque tient : `n2` constant ne prouve pas qu'une largeur est juste
// — `ti=29` le montre, ou `n2` etait deja constant a 0 bit alors que la vraie largeur est 1
// (releve B.4 de `NOTE_IMAGECLE_ETAT_COMPLET_2026-09-13.md`). C'est un DETECTEUR de largeur
// fausse, pas un certificat de largeur juste ; la fermeture (`keyframe_closure.go`) est l'autre
// chaine, et les deux sont necessaires.
//
// Les archetypes a etat VARIABLE (`ti=3`, `8`, `10`, `13`, `24`, `36`..`39`, `48`, le bipede)
// sont ECARTES du jugement, et c'est la grammaire qui les ecarte, pas une liste ecrite a la
// main : un etat a branches n'a pas de « la » largeur, donc `n2` n'y est pas lu au meme endroit
// d'un record a l'autre. Ils restent COMPTES dans le bilan, pour qu'un archetype qui passerait
// de fixe a variable se voie.
//
// # LA PREUVE QUE CE TEST MORD
//
// `consumeDefaultStateTI21` mis a `br.ReadBits(17)` au lieu de 18 : ROUGE sur les cinq bobines
// qui portent des records ti=21 (mesure du 2026-09-14, cf. §5 du plan). Remis a 18 : vert.
//
// # LE FILTRE DES ANCRES FORTUITES
//
// Trois a quinze records par groupe portent un `n1` different du modal (en pratique 0) : ce ne
// sont pas des records de l'archetype mais des ancres retenues par hasard par le balayeur
// (mesure du 2026-09-13, releve B.1 de la meme note). Les garder rendrait tout jugement de
// largeur impossible — ils sont donc retires, et leur compte est publie.

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

// n2MinRecords : en dessous de ce nombre de records retenus, un groupe n'est pas juge. Une
// poignee de records rend « constant » trop facilement pour que le verdict veuille dire quelque
// chose, et une ancre fortuite survivante y pese trop.
const n2MinRecords = 8

// n2GroupesJugesMin : plancher de non-trivialite. Le test doit JUGER au moins ce nombre de
// groupes (bobine x archetype) ; mesure du 2026-09-14 : 77. Sous ce plancher, c'est le chargement
// des bobines qui a casse, pas la grammaire qui est devenue parfaite.
const n2GroupesJugesMin = 60

// n2Mesure est ce qu'un record rend a cet oracle.
type n2Mesure struct {
	n1, n2 uint64
	ds     int // largeur consommee par le deserialiseur d'etat par defaut porte
}

// n2Groupe rassemble les records d'UN archetype sur UNE bobine.
type n2Groupe struct {
	mesures []n2Mesure
}

// TestEtatParDefautN2Constant : la largeur portee ne disperse pas `n2`.
func TestEtatParDefautN2Constant(t *testing.T) {
	rel := LockProcessDecode()
	defer rel()
	juges, fixes, variables, courts := 0, 0, 0, 0
	for _, court := range closureMiniFilms() {
		dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
		groupes := n2Recenser(t, dir)
		for _, ti := range n2ArchetypesPortes(groupes) {
			retenus, fortuites := n2Retenus(groupes[ti])
			if len(retenus) < n2MinRecords {
				courts++
				continue
			}
			largeurs := n2Histo(retenus, func(m n2Mesure) uint64 { return uint64(m.ds) }) //nolint:gosec // largeur >= 0
			if len(largeurs) != 1 {
				variables++
				continue
			}
			fixes++
			juges++
			n2s := n2Histo(retenus, func(m n2Mesure) uint64 { return m.n2 })
			if len(n2s) != 1 {
				t.Errorf("%s ti=%d : n2 DISPERSE (%d valeurs) alors que l'etat par defaut porte "+
					"une largeur FIXE de %d bits sur %d records (%d ancres fortuites retirees) — "+
					"la largeur est fausse. Valeurs : %s",
					court, ti, len(n2s), n2Unique(largeurs), len(retenus), fortuites, n2Texte(n2s))
				continue
			}
			t.Logf("%s ti=%-2d : %4d records · largeur %3d bits · n2 = %d",
				court, ti, len(retenus), n2Unique(largeurs), n2Unique(n2s))
		}
	}
	t.Logf("BILAN : %d groupes juges (largeur FIXE), %d ecartes (largeur VARIABLE), "+
		"%d ecartes (moins de %d records retenus)", fixes, variables, courts, n2MinRecords)
	if juges < n2GroupesJugesMin {
		t.Fatalf("seulement %d groupe(s) juge(s) pour un plancher de %d : l'oracle ne mesure plus "+
			"rien (bobines absentes ? archetypes plus portes ?)", juges, n2GroupesJugesMin)
	}
}

// n2Recenser lit une bobine et rend, par archetype, les mesures de tous ses records d'image-cle.
func n2Recenser(t *testing.T, dir string) map[int]*n2Groupe {
	t.Helper()
	f, ok := imcCharger(t, dir)
	if !ok {
		t.Fatalf("bobine %s inexploitable : regenerer les bobines du lot 0.A.2", dir)
	}
	out := map[int]*n2Groupe{}
	for _, pay := range f.Pays {
		for _, s := range KeyframeRecordSpans(pay) {
			g := out[s.TI]
			if g == nil {
				g = &n2Groupe{}
				out[s.TI] = g
			}
			e := profilLireEtatComplet(pay, s.BitStart, s.TI)
			g.mesures = append(g.mesures, n2Mesure{n1: e.N1, n2: e.N2, ds: e.DSBits})
		}
	}
	return out
}

// n2ArchetypesPortes rend, triees, les cles du recensement dont l'etat par defaut est PORTE.
// Le bipede est ecarte : son etat par defaut est a branches et se joue hors de la table.
func n2ArchetypesPortes(groupes map[int]*n2Groupe) []int {
	out := make([]int, 0, len(groupes))
	for ti := range groupes {
		if ti < 0 || ti == bipedDefaultStateTypeIndex {
			continue
		}
		if _, porte := defaultStateDeserByTI[uint32(ti)]; porte { //nolint:gosec // ti est un index d'archetype
			out = append(out, ti)
		}
	}
	sort.Ints(out)
	return out
}

// n2Retenus retire les ancres fortuites (`n1` different du modal) et rend le reste avec leur
// compte.
func n2Retenus(g *n2Groupe) (retenus []n2Mesure, fortuites int) {
	modal, best := uint64(0), -1
	compte := map[uint64]int{}
	for _, m := range g.mesures {
		compte[m.n1]++
	}
	for v, n := range compte {
		if n > best || (n == best && v < modal) {
			modal, best = v, n
		}
	}
	for _, m := range g.mesures {
		if m.n1 == modal {
			retenus = append(retenus, m)
			continue
		}
		fortuites++
	}
	return retenus, fortuites
}

// n2Histo rend l'histogramme d'un champ des mesures retenues.
func n2Histo(ms []n2Mesure, cle func(n2Mesure) uint64) map[uint64]int {
	out := map[uint64]int{}
	for _, m := range ms {
		out[cle(m)]++
	}
	return out
}

// n2Unique rend la seule cle d'un histogramme qui n'en a qu'une (0 sinon).
func n2Unique(h map[uint64]int) uint64 {
	if len(h) != 1 {
		return 0
	}
	for k := range h {
		return k
	}
	return 0
}

// n2Texte rend un histogramme trie, pour que le message d'echec soit reproductible.
func n2Texte(h map[uint64]int) string {
	ks := make([]uint64, 0, len(h))
	for k := range h {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	s := ""
	for i, k := range ks {
		if i >= 6 {
			s += fmt.Sprintf(" ...(%d valeurs)", len(ks))
			break
		}
		if s != "" {
			s += " "
		}
		s += fmt.Sprintf("%d:x%d", k, h[k])
	}
	return s
}
