//go:build research

package grammar

// reapparition_37_masques_research_test.go — LOT 3.7 : CE QUI SEPARE LE DECODEUR DU BASSIN DE
// MINUTEURS, mesure sur les sept mini-bobines par build.
//
// # LA QUESTION
//
// Le verdict du 2026-09-01 (`objectif_ti11_minuteurs_verdict_test.go`) etablit que `ti=11 i0`
// ne porte qu'un COUPLE D'INDEX de minuteur, FIGE pour la vie d'un objectif, et que la VALEUR du
// compte a rebours vit « derriere l'index — dans `ti=0 i15 managed-engine-timers-component` ».
// Le lot 3.7 a relu cet ecrivain dans l'executable (`FUN_1407ee7b8`, note
// `NOTE_3_7_REAPPARITION_2026-09-17.md` § 2) : c'est un BASSIN de 64 fentes, masque `R(64)` puis,
// par fente presente, `R(2)` d'etiquette et, sauf etiquette 0, le MEME enregistrement de minuteur
// que l'horloge de manche — `R(16) + R(16) + R(5)` en SECONDES.
//
// Reste une question de film, et une seule : QU'EST-CE QUI EMPECHE de le lire aujourd'hui ?
//
// # CE QUE CE FICHIER MESURE, ET CE QU'IL A REFUSE DE MESURER
//
// Il mesure, par archetype et par bobine : le nombre de records d'image-cle BORNES, le nombre qui
// FERMENT (la marche atterrit exactement sur la frontiere suivante), et l'histogramme du
// BLOQUANT — l'index et le nom du premier composant present non porte. Pour l'entite du moteur de
// jeu, cet histogramme EST le chemin de port : il nomme, dans l'ordre, ce qu'il faut porter avant
// d'atteindre `i15`.
//
// IL NE MESURE PAS LA PRESENCE PAR MASQUE, ET C'EST UN NEGATIF D'INSTRUMENT A CONSIGNER. Une
// premiere version de ce fichier comptait les bits de `EntityTrace.Mask` pour dire quels
// composants une image-cle DECLARE — la recette qui avait debloque le dead-state des vehicules
// (`NOTE_V13_DEADSTATE_VEHICULE` § 3). Elle ne vaut pas ici : une image-cle est un ETAT COMPLET
// (`NOTE_IMAGECLE_ETAT_COMPLET_2026-09-13`), il n'y a donc PAS de masque de presence a lire, et
// le champ rendait 64 bits a UN sur `ti=2`, `ti=11` et `ti=12` — y compris au-dela du nombre de
// composants declares au registre (18, 34, 28). Un masque sature n'est pas une mesure. La recette
// V13 vaut pour les records DELTA, pas pour les images-cles ; c'est la difference de cadre qui
// l'invalide, pas le corpus.
//
// # REGIME
//
// Aucun film du cache : les sept mini-bobines par build de `replay/testdata`, qui portent les
// images-cles REELLES de leur film (`PROVENANCE.txt`) — deux CTF (`bcb6d393` Cliffhanger,
// `fb1a1a72` Banished Narrows), un Oddball (`60ae07c4`), quatre modes a vehicules (`a521164d`
// Total Control Heavies ; `111fa685`, `11de8353`, `e5adf7b2` BTB). C'est le corpus du ratchet de
// fermeture 0.A.3, deja sous garde.
//
//	go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run Reapparition37 -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// reap37Archetypes sont les archetypes recenses, et la raison de chacun.
//
// `ti=0` ET `ti=2` sont tous deux la : l'index d'archetype de l'entite du moteur de jeu CHANGE
// avec le build (regle du depot, `equipment_state.go:157`), et la mesure le confirme — sur
// `bcb6d393` c'est `ti=2` qui porte 19 records (un par image-cle) et `ti=0` un seul.
var reap37Archetypes = map[uint32]string{
	0:  "game-engine — le BASSIN de minuteurs (i15) y vit",
	2:  "game-engine (jumeau) — meme table de composants",
	11: "managed-objective — i0 = le couple d'INDEX de minuteur",
	12: "managed-navpoint — i11/i12 = duree manuelle, R(17), pas de 50 ms",
	20: "spawn-filter — les filtres de reapparition",
	29: "respawn-block — le blocage de reapparition d'un JOUEUR",
	40: "vehicule — l'archetype ou un minuteur de reapparition DEVRAIT etre",
	43: "device — les distributeurs (i31..i37)",
}

// reap37Mesure accumule, pour un archetype, les denominateurs et l'histogramme du bloquant.
type reap37Mesure struct {
	bornes, fermes int
	bloquant       map[int]int
	noms           []string
}

func TestReapparition37CheminVersLeBassin(t *testing.T) {
	for _, court := range closureMiniFilms() {
		t.Run(court, func(t *testing.T) { reap37UneBobine(t, court) })
	}
}

func reap37UneBobine(t *testing.T, court string) {
	t.Helper()
	dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre %s : %v", court, err)
	}
	par := map[uint32]*reap37Mesure{}
	for ti := range reap37Archetypes {
		m := &reap37Mesure{bloquant: map[int]int{}}
		if a, ok := reg.Archetype(int(ti)); ok {
			m.noms = a.Components
		}
		par[ti] = m
	}
	for _, pay := range e191cPayloads(fc) {
		for _, b := range keyframeBornes(pay) {
			m := par[uint32(b.TI)] //nolint:gosec // TI borne par le format (< 50)
			if m == nil {
				continue
			}
			m.bornes++
			tr := WalkKeyframeFullState(pay, b.Bit, reg, contexteDInstrument())
			switch {
			case tr.DesyncAt >= 0:
				m.bloquant[tr.DesyncAt]++
			case tr.EndBit == b.Want:
				m.fermes++
			default:
				m.bloquant[-1]++ // marche complete, mais au mauvais bit
			}
		}
	}
	reap37Publier(t, par)
}

func reap37Publier(t *testing.T, par map[uint32]*reap37Mesure) {
	t.Helper()
	tis := make([]int, 0, len(par))
	for ti := range par {
		tis = append(tis, int(ti))
	}
	sort.Ints(tis)
	for _, ti := range tis {
		m := par[uint32(ti)] //nolint:gosec // borne par reap37Archetypes
		if m.bornes == 0 {
			continue
		}
		t.Logf("  ti=%-3d %-60s %4d borne(s), %4d ferme(s), %d composant(s) au registre",
			ti, reap37Archetypes[uint32(ti)], m.bornes, m.fermes, len(m.noms)) //nolint:gosec // idem
		for _, l := range reap37Histogramme(m) {
			t.Logf("      %s", l)
		}
	}
}

// reap37Histogramme rend les lignes du bloquant, de la plus frequente a la moins frequente.
func reap37Histogramme(m *reap37Mesure) []string {
	type kv struct{ idx, n int }
	l := make([]kv, 0, len(m.bloquant))
	for i, n := range m.bloquant {
		l = append(l, kv{i, n})
	}
	sort.Slice(l, func(a, b int) bool {
		if l[a].n != l[b].n {
			return l[a].n > l[b].n
		}
		return l[a].idx < l[b].idx
	})
	out := make([]string, 0, len(l))
	for _, e := range l {
		if e.idx < 0 {
			out = append(out, fmt.Sprintf("%4d record(s) : marche COMPLETE, fin au mauvais bit", e.n))
			continue
		}
		out = append(out, fmt.Sprintf("%4d record(s) : BLOQUANT i%d %s", e.n, e.idx, reap37Nom(m.noms, e.idx)))
	}
	return out
}

func reap37Nom(noms []string, i int) string {
	if i < 0 || i >= len(noms) {
		return fmt.Sprintf("(index hors registre : %d composants declares)", len(noms))
	}
	return noms[i]
}
