//go:build research

package grammar

// p4_entites_ti9_research_test.go — SONDE P4 (campagne « retours rejeu », 2026-09-23), volet FILM.
//
// Rejoue le balayage des entites `ti=9` de `sieges_remplacants_research_test.go`
// (`noterEntitesDuPaquet`, meme lecteur, meme marche d'image-cle que la production) sur un film
// COMPLET du cache, et rend ce que M2 doit lire :
//
//	ENTITES       slot, index, designateur, premiere / derniere image-cle (rang ET frame du rejeu),
//	              images-cles portees et TROUS (une entite absente d'une image-cle entre sa premiere
//	              et sa derniere : perte de marche, cf. P2, ou vraie absence) ;
//	IMAGES-CLES   par paquet : records ti=9 VUS (lus ou non), occupants lus par designateur ;
//	TIRS          pour chaque index de tireur (record 105), la couverture par les fenetres des
//	              entites DE MEME INDEX, et, pour les tirs qu'elles ne couvrent pas, les entites
//	              ARRIVANTES qui les couvrent TOUS (l'heritier de la place).
//
// HORLOGE : frame = (ts - P4_ORIGIN_US) / 100 ms, l'origine etant le premier horodatage de position
// des faits (volet `replay/p4_equipes_faits_research_test.go`), pour parler la langue des documents.
// Fenetre d'une entite vue aux images-cles [a..b] : stricte [f(a), f(b)], large ]f(a-1), f(b+1)[.
//
// Lecture seule, un film, sous la voie film :
//
//	P4_FILM=<data>/cache/film_chunks/b1ad85eb P4_ORIGIN_US=<origine> \
//	  go test -tags research -count=1 -run '^TestP4EntitesTi9$' -v ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// p4ImageCle : une image-cle porteuse, et ce qu'elle a montre des entites ti=9.
type p4ImageCle struct {
	rang   int
	ts     uint64
	vus    int // records ti=9 rencontres par la marche (lus ou non)
	lus    map[int]*siegeEntite
	frame  int64
	chunk  int
	paquet int
}

func TestP4EntitesTi9(t *testing.T) {
	dir := os.Getenv("P4_FILM")
	if dir == "" {
		t.Skip("P4_FILM absent : sonde sautee")
	}
	origine, err := strconv.ParseUint(os.Getenv("P4_ORIGIN_US"), 10, 64)
	if err != nil {
		t.Fatalf("P4_ORIGIN_US illisible : %v", err)
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film : %v", err)
	}
	fc := NewFilmContext(film)
	fr := func(ts uint64) int64 { return (int64(ts) - int64(origine)) / 100_000 }
	ics, ents := p4Balayer(t, fc, fr)
	p4RapportImagesCles(t, ics)
	tri := p4RapportEntites(t, ics, ents)
	tirs, err := ScanFireEvents(film)
	if err != nil {
		t.Fatalf("tirs : %v", err)
	}
	p4TirsContreEntites(t, tirs, tri, ics, fr)
}

// p4Balayer : une passe sur les images-cles ; chaque paquet est lu DEUX fois par le meme lecteur,
// une fois dans sa propre table (ce que CETTE image-cle montre), une fois dans la table globale.
func p4Balayer(t *testing.T, fc *FilmContext, fr func(uint64) int64) ([]*p4ImageCle, map[int]*siegeEntite) {
	t.Helper()
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	ents := map[int]*siegeEntite{}
	var ics []*p4ImageCle
	rang := -1
	for _, c := range fc.ChunkNumbers() {
		raw, paquets, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range paquets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			rang++
			pay := pk.Payload(raw)
			ic := &p4ImageCle{rang: rang, ts: pk.TimestampUS, lus: map[int]*siegeEntite{},
				frame: fr(pk.TimestampUS), chunk: c, paquet: pk.Index}
			for _, b := range keyframeBornesToutes(pay) {
				if b.TI == managedPlayerTypeIndex {
					ic.vus++
				}
			}
			noterEntitesDuPaquet(pay, reg, rang, pk.TimestampUS, ic.lus)
			noterEntitesDuPaquet(pay, reg, rang, pk.TimestampUS, ents)
			ics = append(ics, ic)
		}
	}
	return ics, ents
}

func p4RapportImagesCles(t *testing.T, ics []*p4ImageCle) {
	t.Helper()
	vus, lus := 0, 0
	for _, ic := range ics {
		parDes := map[int][]string{}
		for _, e := range ic.lus {
			for d := range e.Designateurs {
				parDes[d] = append(parDes[d], fmt.Sprintf("%d@%d", e.Index, e.Slot))
			}
		}
		var sb strings.Builder
		for _, d := range []int{0, 1, 2, 3} {
			if l := parDes[d]; len(l) > 0 {
				sort.Strings(l)
				fmt.Fprintf(&sb, " des%d(%d)=%v", d, len(l), l)
			}
		}
		vus += ic.vus
		lus += len(ic.lus)
		t.Logf("IMAGE-CLE rang %2d chunk %2d pk %3d ts %d f%5d | ti9 vus=%d lus=%d |%s",
			ic.rang, ic.chunk, ic.paquet, ic.ts, ic.frame, ic.vus, len(ic.lus), sb.String())
	}
	t.Logf("IMAGES-CLES : %d paquet(s), %d record(s) ti=9 vus, %d lus", len(ics), vus, lus)
}

// p4RapportEntites : la table entite -> (index, designateur, premiere/derniere image-cle), trous compris.
func p4RapportEntites(t *testing.T, ics []*p4ImageCle, ents map[int]*siegeEntite) []*siegeEntite {
	t.Helper()
	tri := make([]*siegeEntite, 0, len(ents))
	for _, e := range ents {
		tri = append(tri, e)
	}
	sort.Slice(tri, func(i, j int) bool {
		if tri[i].PremierPk != tri[j].PremierPk {
			return tri[i].PremierPk < tri[j].PremierPk
		}
		return tri[i].Index < tri[j].Index
	})
	dernier := len(ics) - 1
	for _, e := range tri {
		trous := []int{}
		for r := e.PremierPk; r <= e.DernierPk; r++ {
			if ics[r].lus[e.Slot] == nil {
				trous = append(trous, r)
			}
		}
		avant, apres := "debut", "fin"
		if e.PremierPk > 0 {
			avant = fmt.Sprintf("f%d", ics[e.PremierPk-1].frame)
		}
		if e.DernierPk < dernier {
			apres = fmt.Sprintf("f%d", ics[e.DernierPk+1].frame)
		}
		t.Logf("ENTITE slot=%5d idx=%2d des=%s | rang [%2d..%2d] | stricte [f%d..f%d] | large ]%s..%s[ | "+
			"paquets=%d trous=%v%s", e.Slot, e.Index, histogrammeDesSieges(e.Designateurs),
			e.PremierPk, e.DernierPk, ics[e.PremierPk].frame, ics[e.DernierPk].frame, avant, apres,
			e.Paquets, trous, marqueInstable(e.IndexInstables))
	}
	parIdx := map[int]int{}
	for _, e := range tri {
		parIdx[e.Index]++
	}
	t.Logf("ENTITES : %d ; par index %v", len(tri), parIdx)
	return tri
}

// p4Designateur : le designateur majoritaire d'une entite (stable, cf. lot 1.9.14 : 88/88).
func p4Designateur(e *siegeEntite) int {
	best, n := -1, -1
	for d, c := range e.Designateurs {
		if c > n || (c == n && d < best) {
			best, n = d, c
		}
	}
	return best
}

// p4Couvre : l'entite couvre-t-elle la frame, en fenetre large (image-cle voisine exclue) ?
func p4Couvre(e *siegeEntite, ics []*p4ImageCle, f0 int64) (stricte, large bool) {
	a, b := ics[e.PremierPk].frame, ics[e.DernierPk].frame
	stricte = f0 >= a && f0 <= b
	la, lb := int64(-1<<62), int64(1<<62)
	if e.PremierPk > 0 {
		la = ics[e.PremierPk-1].frame
	}
	if e.DernierPk < len(ics)-1 {
		lb = ics[e.DernierPk+1].frame
	}
	return stricte, f0 > la && f0 < lb
}

func p4TirsContreEntites(t *testing.T, tirs []FireEvent, tri []*siegeEntite, ics []*p4ImageCle, fr func(uint64) int64) {
	t.Helper()
	parTireur := map[int][]int64{}
	for _, e := range tirs {
		parTireur[e.ShooterIndex5] = append(parTireur[e.ShooterIndex5], fr(e.TimestampUS))
	}
	// ARRIVANT = entite absente de la PREMIERE image-cle porteuse (la toute premiere du film peut
	// etre vide : b1ad85eb rang 0, f-186, preambule).
	depart := 1 << 30
	for _, e := range tri {
		if e.PremierPk < depart {
			depart = e.PremierPk
		}
	}
	cles := make([]int, 0, len(parTireur))
	for k := range parTireur {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	for _, s := range cles {
		fs := parTireur[s]
		sort.Slice(fs, func(i, j int) bool { return fs[i] < fs[j] })
		couverts, horsLarge := 0, []int64{}
		for _, f0 := range fs {
			ok := false
			for _, e := range tri {
				if e.Index != s {
					continue
				}
				if _, l := p4Couvre(e, ics, f0); l {
					ok = true
				}
			}
			if ok {
				couverts++
			} else {
				horsLarge = append(horsLarge, f0)
			}
		}
		heritiers := []string{}
		if len(horsLarge) > 0 {
			for _, e := range tri {
				if e.PremierPk == depart || e.Index == s {
					continue
				}
				tous := true
				for _, f0 := range horsLarge {
					if _, l := p4Couvre(e, ics, f0); !l {
						tous = false
						break
					}
				}
				if tous {
					heritiers = append(heritiers, fmt.Sprintf("idx%d@slot%d(des%d)", e.Index, e.Slot, p4Designateur(e)))
				}
			}
		}
		t.Logf("TIREUR %2d : %3d tirs [f%d..f%d] | couverts par une entite d'index %d : %d | hors : %d "+
			"[%s] | arrivants couvrant TOUS les hors : %v", s, len(fs), fs[0], fs[len(fs)-1], s, couverts,
			len(horsLarge), p4Bornes(horsLarge), heritiers)
	}
}

func p4Bornes(fs []int64) string {
	if len(fs) == 0 {
		return "-"
	}
	return fmt.Sprintf("f%d..f%d", fs[0], fs[len(fs)-1])
}
