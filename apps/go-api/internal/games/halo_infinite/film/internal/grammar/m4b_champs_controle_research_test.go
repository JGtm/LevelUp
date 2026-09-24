//go:build research

package grammar

// m4b_champs_controle_research_test.go — LOT M4b : CE QUE LES AUTRES CHAMPS DE L ENTREE DE CONTROLE
// DISENT D UNE MONTURE. Mesure seule.
//
// La marche de production (celle de [ScanMarcheDesTrames]) est rejouee a l identique, un crochet
// de vue C en plus qui garde, pour l index S3_INDEX, chaque entree lue avec son bloc : l instant et
// les champs hors bloc d action ([ChampsDeControle]). Les valeurs sont ventilees DANS et HORS des
// fenetres de monture (S3_EPISODES + M4B_MONTURES) : un champ qui les separe est un signal de
// monture ecrit dans le film.
//
// Rejouable : memes variables que `m4b_tir_continu_research_test.go`.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// m4bEntreeLue est une entree de controle du joueur designe, a son instant.
type m4bEntreeLue struct {
	ts uint64
	e  EntreeDeControle
}

// m4bMarcheCapturee rejoue la marche de production et rend les entrees lues de l index `index`.
func m4bMarcheCapturee(t *testing.T, tc t516Temoin, index int) []m4bEntreeLue {
	t.Helper()
	var st MarcheDesTrames
	reg, err := tc.fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	sc := &movementStateScanner{st: &st.MovementStateStats, crouch: -1, slide: -1, mobility: -1, ability: -1,
		marche: tc.fc.MarcheDImageCle(), tir: nouveauCollecteurTirContinu(&st.ContinuousFireStats)}
	var out []m4bEntreeLue
	cfg := tc.fc.CadreDeBalayage()
	obs := NouvelleObservation()
	obs.VueControleHook = func(l LectureVueC) {
		if !sc.tir.recu && l.Fermee {
			for _, e := range l.Entrees {
				if e.Index == index && e.Bloc {
					out = append(out, m4bEntreeLue{ts: sc.tir.ts, e: e})
				}
			}
		}
		sc.tir.recevoir(l)
	}
	cfg.Obs = obs
	sc.obs = obs
	sc.monde = NewWorld(reg)
	sc.monde.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		sc.monde.PoserChunkCourant(c)
		sc.lierLeMonde(data, pks)
		for _, pk := range pks {
			sc.paquet(c, pk, data, cfg)
		}
	}
	return out
}

// TestM4bChampsDeControle ventile les champs de l entree du joueur designe dans et hors monture.
func TestM4bChampsDeControle(t *testing.T) {
	cad := s3LireCadre(t)
	fen := append([][2]int(nil), cad.episodes...)
	fen = append(fen, m4bFenetres("M4B_MONTURES")...)
	tc := t516Cadre(t)
	ents := m4bMarcheCapturee(t, tc, cad.index)
	type cle struct {
		champ string
		v     int
	}
	dedans, dehors := map[cle]int{}, map[cle]int{}
	nd, nh := 0, 0
	for _, x := range ents {
		c := x.e.Champs
		vals := map[string]int{"cdc04": c.Cdc04, "court": c.Court, "troisieme": c.Troisieme,
			"champ10": c.Champ10, "drapeaux": c.Drapeaux, "arme0": x.e.Action.Arme[0],
			"ana0": c.Analogique[0], "ana1": c.Analogique[1]}
		cible := dehors
		if m4bDans(cad.trame(x.ts), fen) {
			cible = dedans
			nd++
		} else {
			nh++
		}
		for k, v := range vals {
			cible[cle{k, v}]++
		}
	}
	t.Logf("index %d : %d entrees lues avec bloc — %d dans les montures, %d hors", cad.index, len(ents), nd, nh)
	champs := []string{"cdc04", "court", "troisieme", "champ10", "drapeaux", "arme0", "ana0", "ana1"}
	for _, ch := range champs {
		var lignes []string
		vus := map[int]bool{}
		for k := range dedans {
			if k.champ == ch {
				vus[k.v] = true
			}
		}
		for k := range dehors {
			if k.champ == ch {
				vus[k.v] = true
			}
		}
		vs := make([]int, 0, len(vus))
		for v := range vus {
			vs = append(vs, v)
		}
		sort.Ints(vs)
		for _, v := range vs {
			lignes = append(lignes, fmt.Sprintf("%d:%d/%d", v, dedans[cle{ch, v}], dehors[cle{ch, v}]))
		}
		t.Logf("   %-9s (valeur:dans/hors) %s", ch, strings.Join(lignes, " "))
	}
	// La frise : une ligne par changement d un champ HORS couple analogique, autour des montures.
	sansAnalogique := func(c ChampsDeControle) ChampsDeControle {
		c.Analogique = [2]int{}
		return c
	}
	for i, x := range ents {
		if i > 0 && sansAnalogique(x.e.Champs) == sansAnalogique(ents[i-1].e.Champs) {
			continue
		}
		tr := cad.trame(x.ts)
		if !m4bDans(tr, m4bElargir(fen, 50)) {
			continue
		}
		t.Logf("   t%d champs %+v arme %v tire %v", tr, x.e.Champs, x.e.Action.Arme, x.e.Action.Tire())
	}
}

// m4bFenetres lit une liste de fenetres `a-b` separees par des virgules.
func m4bFenetres(nom string) [][2]int {
	var out [][2]int
	for _, s := range m4bListeBrute(nom) {
		var a, b int
		if _, err := fmt.Sscanf(s, "%d-%d", &a, &b); err == nil {
			out = append(out, [2]int{a, b})
		}
	}
	return out
}

// m4bElargir elargit chaque fenetre de `marge` trames des deux cotes.
func m4bElargir(fen [][2]int, marge int) [][2]int {
	out := make([][2]int, len(fen))
	for i, f := range fen {
		out[i] = [2]int{f[0] - marge, f[1] + marge}
	}
	return out
}

// m4bListeBrute lit une variable d environnement comme une liste separee par des virgules.
func m4bListeBrute(nom string) []string { return strings.Split(os.Getenv(nom), ",") }
