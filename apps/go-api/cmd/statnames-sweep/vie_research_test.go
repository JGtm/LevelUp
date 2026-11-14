//go:build research

package main

// vie_research_test.go — MESURE DE LA VIE D'UN SPARTAN : le compteur « dégâts subis » du film.
//
// Question : un Spartan vaut-il 229 ou 230 points de vie effective (90 de santé + 140 de
// bouclier d'après les fichiers du jeu) ? Le compteur « dégâts subis » est le composant 10,
// côté A, du statborg d'un joueur (le côté B compte les vies) ; il monte UNE fois par vie, à la
// réapparition, de tout ce que le joueur a encaissé pendant cette vie. Une vie sans recharge de
// bouclier encaisse exactement la vie effective : son incrément est la valeur cherchée.
//
// Le test imprime l'histogramme des incréments de 220 à 235 par film et au total : le mode et la
// place de 229 contre 230. Lecture seule du cache de films, aucune base ouverte.
//
//	LEVELUP_FILM_CACHE=<racine du cache> [LEVELUP_FILM_IDS=a,b,c] \
//	  go test -tags=research ./cmd/statnames-sweep -run TestResearchVieEffective -v
//
// Sans LEVELUP_FILM_IDS, tous les films de <cache>/film_chunks sont lus, un par un.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
)

const (
	// statDegatsSubis : composant du compteur « dégâts subis » (côté A).
	statDegatsSubis = 10
	vieBasse        = 220
	vieHaute        = 235
)

func TestResearchVieEffective(t *testing.T) {
	cache := os.Getenv("LEVELUP_FILM_CACHE")
	if cache == "" {
		t.Skip("LEVELUP_FILM_CACHE requis")
	}
	ids := splitIDs(os.Getenv("LEVELUP_FILM_IDS"))
	if len(ids) == 0 {
		entries, err := os.ReadDir(filepath.Join(cache, "film_chunks"))
		if err != nil {
			t.Fatalf("film_chunks : %v", err)
		}
		for _, e := range entries {
			if e.IsDir() {
				ids = append(ids, e.Name())
			}
		}
		sort.Strings(ids)
	}
	hist := map[int64]int{}
	var films, withCounter, rises int
	for _, id := range ids {
		film, ok, err := filmcache.LoadFilm(cache, id)
		if err != nil || !ok {
			continue
		}
		films++
		recs, _, _ := decfilm.StatRecordsBornes(film, id)
		series := decfilm.SeriesTotal(recs, decfilm.StatComponent{Comp: statDegatsSubis}, false, nil)
		got := false
		for _, pts := range series {
			prev := int64(0)
			for _, p := range pts {
				if p.Value != prev {
					hist[p.Value-prev]++
					rises++
					got = true
					prev = p.Value
				}
			}
		}
		if got {
			withCounter++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "films lus %d, avec compteur %d, hausses %d\n", films, withCounter, rises)
	for d := int64(vieBasse); d <= vieHaute; d++ {
		fmt.Fprintf(&b, "%d\t%d\n", d, hist[d])
	}
	t.Log("\n" + b.String())
}
