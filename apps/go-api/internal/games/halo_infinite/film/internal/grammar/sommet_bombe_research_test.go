//go:build research

package grammar

// sommet_bombe_research_test.go — LOT 5.1.8 : CE QUE SONT LES ECHANTILLONS APRES LE SOMMET.
//
// Le portage de `ti=12` (5.1.1) fait passer les lectures de l anneau radial de 1 148 a 2 012 sur
// `c75f33b8`, et le classement d armement (`replay/bomb_armings.go`) cesse de reconnaitre le
// moindre armement : `EndsAtSummit` exige que le DERNIER echantillon du segment soit au sommet, et
// les echantillons ajoutes le suivent. Avant de changer le predicat, il faut NOMMER ce que sont
// ces echantillons : restent-ils au plein, ou retombent-ils ?
//
// LECTURE SEULE, UN FILM PAR INVOCATION, aucune cuisson :
//
//	SOM_FILM=<abs>/film_chunks/c75f33b8 SOM_MANIFESTE=<abs>/film_manifests/c75f33b8.json \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run '^TestSommetBombe$' -v -timeout 60m

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
)

// somPlein est le quantum PLEIN de l anneau (`bombArmedFullQuantum` cote publication).
const somPlein = 254

func TestSommetBombe(t *testing.T) {
	dir, man := os.Getenv("SOM_FILM"), os.Getenv("SOM_MANIFESTE")
	if dir == "" || man == "" {
		t.Skip("instrument de mesure : SOM_FILM et SOM_MANIFESTE requis")
	}
	sc, err := ScanFilmNavpointRadial(dir, somHorloge(t, man))
	if err != nil {
		t.Fatalf("anneau radial : %v", err)
	}
	segs := NavpointSegments(sc.Reads)
	sort.Slice(segs, func(i, j int) bool { return segs[i].EndMS < segs[j].EndMS })
	plein, finAuSommet := 0, 0
	t.Logf("lectures=%d segments=%d", len(sc.Reads), len(segs))
	for _, g := range segs {
		if int(g.QMax) < somPlein {
			continue
		}
		plein++
		if int(g.QEnd) >= int(g.QMax)-NavpointSummitToleranceQ {
			finAuSommet++
		}
		t.Logf("    slot=%-5d %7d..%7d ms  ech=%-4d qStart=%-3d qMin=%-3d qMax=%-3d qEnd=%-3d %s",
			g.Slot, g.StartMS, g.EndMS, g.Samples, g.QStart, g.QMin, g.QMax, g.QEnd,
			somForme(g))
	}
	t.Logf("BILAN : %d segments atteignent le plein, %d y terminent (tolerance %d)",
		plein, finAuSommet, NavpointSummitToleranceQ)
}

// somForme nomme ce que devient le segment APRES son sommet.
func somForme(g NavpointSegment) string {
	switch {
	case int(g.QEnd) >= somPlein-NavpointSummitToleranceQ:
		return "-> RESTE AU PLEIN"
	case g.QEnd == 0:
		return "-> RETOMBE A ZERO"
	default:
		return "-> retombe partiellement"
	}
}

// somHorloge lit `index de chunk -> start_ms` dans le manifeste du film.
func somHorloge(t *testing.T, chemin string) map[int]int {
	t.Helper()
	raw, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("manifeste %s : %v", chemin, err)
	}
	var m struct {
		Chunks []struct {
			Index   int `json:"index"`
			StartMS int `json:"start_ms"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("manifeste %s : %v", chemin, err)
	}
	out := make(map[int]int, len(m.Chunks))
	for _, c := range m.Chunks {
		out[c.Index] = c.StartMS
	}
	if len(out) == 0 {
		t.Fatalf("manifeste %s : aucun chunk", chemin)
	}
	return out
}
