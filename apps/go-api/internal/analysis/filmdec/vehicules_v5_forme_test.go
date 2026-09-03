package filmdec

// vehicules_v5_forme_test.go — LOT V5 : LA FORME DU RECORD comme état d'occupation.
//
// D'OÙ VIENT CETTE PISTE. Le test de présence (`TestV5Appariement`) rend un résultat qui
// n'est pas seulement négatif, il est ANORMAL : le slot du véhicule apparié n'apparaît JAMAIS
// (0/11) dans le record d'image-clé de son occupant, alors qu'il apparaît dans 22,7 % des
// records des AUTRES bipèdes au même instant. Un champ absent rendrait le taux de l'occupant
// ÉGAL au taux de fond, pas nul. Un taux nul dit que le record de l'occupant n'est pas le
// même OBJET : il est plus court, ou il ne porte pas les mêmes composants.
//
// C'est cohérent avec le modèle V1a.4 (« l'enfant attaché ne réplique plus ») : une entité
// attachée n'a plus de position propre, plus de vélocité propre, plus d'orientation propre —
// son état sérialisé RÉTRÉCIT. Si le rétrécissement est mesurable et net, il donne l'état
// d'occupation BINAIRE (à bord / à pied) à n'importe quel instant d'image-clé, sans nommer le
// véhicule ni le siège. C'est moins que l'objectif, et c'est dit tel quel.
//
//	CGO_ENABLED=0 V5_ROOT=<cache> V5_FILMS=... \
//	  go test ./internal/analysis/filmdec/ -run TestV5Forme -v -timeout 120m

import (
	"fmt"
	"sort"
	"testing"
)

// v5Distrib accumule une distribution de longueurs pour en tirer médiane et quartiles.
type v5Distrib struct {
	vals []int
}

func (d *v5Distrib) Ajoute(v int) { d.vals = append(d.vals, v) }

func (d *v5Distrib) Resume() string {
	if len(d.vals) == 0 {
		return "n=0"
	}
	s := append([]int(nil), d.vals...)
	sort.Ints(s)
	q := func(f float64) int {
		i := int(f * float64(len(s)-1))
		return s[i]
	}
	return fmt.Sprintf("n=%d min=%d q1=%d méd=%d q3=%d max=%d",
		len(s), s[0], q(0.25), q(0.5), q(0.75), s[len(s)-1])
}

// Median rend la médiane (0 si vide).
func (d *v5Distrib) Median() int {
	if len(d.vals) == 0 {
		return 0
	}
	s := append([]int(nil), d.vals...)
	sort.Ints(s)
	return s[len(s)/2]
}

// TestV5Forme compare la LONGUEUR du record d'image-clé d'un bipède À BORD à celle d'un
// bipède À PIED au MÊME instant, et fait de même côté véhicule (occupé / vide).
func TestV5Forme(t *testing.T) {
	for _, dir := range v5Films(t) {
		v5FormeUnFilm(t, dir)
	}
}

func v5FormeUnFilm(t *testing.T, dir string) {
	t.Helper()
	eps, _, err := v5Episodes(dir)
	if err != nil {
		t.Logf("V5 FORME %s : %v", dir, err)
		return
	}
	app, err := v5Apparier(dir, eps)
	if err != nil {
		t.Logf("V5 FORME %s : %v", dir, err)
		return
	}
	kfs := v5Keyframes(dir)
	var bAbord, bApied, vOccupe, vVide v5Distrib
	// TÉMOIN PAR DÉCALAGE TEMPOREL : les mêmes slots, aux mêmes images-clés, mais avec les
	// épisodes décalés de 37 s. Si la longueur distingue vraiment « à bord », le témoin doit
	// retomber sur la distribution des piétons.
	var bAbordTemoin v5Distrib
	for _, kf := range kfs {
		if len(kf) == 0 {
			continue
		}
		ts := kf[0].TS
		aBord, aBordT := map[int]bool{}, map[int]bool{}
		vOcc := map[int]bool{}
		for _, e := range app {
			if ts > e.DebutUS && ts < e.FinUS {
				aBord[int(e.Slot)] = true
				if e.Ok {
					vOcc[int(e.Vehicule)] = true
				}
			}
			if ts > e.DebutUS+v5DecalTemoinUS && ts < e.FinUS+v5DecalTemoinUS {
				aBordT[int(e.Slot)] = true
			}
		}
		for _, r := range kf {
			switch r.TI {
			case v5BipedeTI:
				if aBord[r.Slot] {
					bAbord.Ajoute(r.LongueurEnBits)
				} else {
					bApied.Ajoute(r.LongueurEnBits)
				}
				if aBordT[r.Slot] && !aBord[r.Slot] {
					bAbordTemoin.Ajoute(r.LongueurEnBits)
				}
			case v5VehiculeTI:
				if vOcc[r.Slot] {
					vOccupe.Ajoute(r.LongueurEnBits)
				} else {
					vVide.Ajoute(r.LongueurEnBits)
				}
			}
		}
	}
	t.Logf("V5 FORME %s — longueur du record d'image-clé, en bits", dir)
	t.Logf("    bipède À BORD          : %s", bAbord.Resume())
	t.Logf("    bipède à pied (témoin) : %s", bApied.Resume())
	t.Logf("    bipède témoin décalé   : %s", bAbordTemoin.Resume())
	t.Logf("    véhicule OCCUPÉ        : %s", vOccupe.Resume())
	t.Logf("    véhicule vide (témoin) : %s", vVide.Resume())
	t.Logf("    écart de médiane bipède à bord - à pied : %d bits",
		bAbord.Median()-bApied.Median())

	// SÉPARABILITÉ : un seuil sur la longueur classe-t-il « à bord » ? On prend le meilleur
	// seuil possible et on publie sa performance — un seuil choisi APRÈS coup est optimiste
	// par construction, et c'est dit.
	v5Separabilite(t, bAbord, bApied)
}

// v5Separabilite cherche le seuil de longueur qui maximise (rappel à bord - taux de fausse
// alarme chez les piétons) et publie le couple.
func v5Separabilite(t *testing.T, pos, neg v5Distrib) {
	t.Helper()
	if len(pos.vals) == 0 || len(neg.vals) == 0 {
		t.Logf("    séparabilité : n/a (une classe est vide)")
		return
	}
	cand := map[int]bool{}
	for _, v := range pos.vals {
		cand[v] = true
	}
	for _, v := range neg.vals {
		cand[v] = true
	}
	bestSeuil, bestEcart, bestP, bestN := 0, -2.0, 0.0, 0.0
	for s := range cand {
		p, n := 0, 0
		for _, v := range pos.vals {
			if v <= s {
				p++
			}
		}
		for _, v := range neg.vals {
			if v <= s {
				n++
			}
		}
		tp := float64(p) / float64(len(pos.vals))
		fp := float64(n) / float64(len(neg.vals))
		if tp-fp > bestEcart {
			bestSeuil, bestEcart, bestP, bestN = s, tp-fp, tp, fp
		}
	}
	t.Logf("    séparabilité (meilleur seuil, choisi APRÈS coup — optimiste) : longueur <= %d bits"+
		" -> à bord %.1f %%, fausse alarme piéton %.1f %%, écart %+.1f pts",
		bestSeuil, bestP*100, bestN*100, bestEcart*100)
}
