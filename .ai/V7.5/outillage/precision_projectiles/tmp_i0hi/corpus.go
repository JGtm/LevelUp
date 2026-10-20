package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

// POURQUOI MUTUALISER LES CARTES. Sur la seule Cliffhanger, 30 films ne donnent que 127
// records opaques encadres par deux positions vraies. C'est trop peu pour distinguer un vrai
// negatif d'un negatif sur la COUVERTURE — le piege que ce chantier a deja paye avec T3.
// Le corpus entier porte 15 cartes du catalogue ; chaque film se decode avec les bornes de SA
// carte, et les paires se mutualisent ensuite.
//
// LES DEUX REGIMES DE MUTUALISATION NE TESTENT PAS LA MEME CHOSE :
//   - coordonnee MONDE  : juste si la branche opaque encode dans une plage FIXE (+-100 ou
//                         +-20000). La relation entier -> position est alors la meme partout.
//   - coordonnee NORMALISEE : juste si elle encode aux largeurs de la CARTE, comme la branche
//                         basse. C'est alors la position ramenee dans [0,1] qui est comparable.
// Les deux sont calcules ; si aucun ne mord, aucune des deux hypotheses ne tient.

// filmJob est un film a balayer avec les bornes de sa carte.
type filmJob struct {
	dir     string
	mapName string
	wr      filmdec.Vec3Range
}

// jobsFromCatalog croise le cache de films, la table matchID -> carte et le catalogue de
// bornes. Un film dont la carte n'a pas de bornes est ECARTE, jamais decode avec les bornes
// d'une autre : c'est le bug que `map_bounds.go` documente en tete.
func jobsFromCatalog(root, csvPath string, cat *filmdec.MapQuantCatalog, limit int) ([]filmJob, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	byID := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		i := strings.Index(line, ",")
		if i <= 0 {
			continue
		}
		byID[line[:i]] = line[i+1:]
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var jobs []filmJob
	skippedNoMap, skippedNoBounds := 0, 0
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		m, ok := byID[name]
		if !ok {
			skippedNoMap++
			continue
		}
		entry, err := cat.Lookup(m)
		if err != nil {
			skippedNoBounds++
			continue
		}
		jobs = append(jobs, filmJob{dir: filepath.Join(root, name), mapName: m, wr: entry.Range()})
		if limit > 0 && len(jobs) >= limit {
			break
		}
	}
	fmt.Printf("corpus : %d films retenus ; ecartes %d sans carte connue, %d sans bornes au catalogue\n",
		len(jobs), skippedNoMap, skippedNoBounds)
	return jobs, nil
}
