package replay

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

// deaths_source.go — LE FIL DES MORTS, LU DANS LE FILM.
//
// OÙ IL VIT. Le dernier chunk du manifest film est le chunk des « highlight events » : un
// enregistrement par événement de match, dont les morts, chacune portant le XUID de la
// victime et son instant. Il est DANS LE FILM — aucune base n'intervient, ce qui préserve
// la propriété que tout le rejeu tient hors ligne.
//
// POURQUOI CE FICHIER EST SÉPARÉ. C'est le seul point du paquet qui fait des I/O disque et
// qui dépend du paquet `analysis` (pour son parseur, déjà en production et éprouvé). Le
// reste de `replay` reste pur et testable sans fichier : `BuildFromPositions` reçoit les
// morts par `Options.Deaths`, comme il reçoit déjà les loadouts et les grenades.
//
// CE QU'ON NE FAIT PAS ICI, et c'est délibéré : on ne recopie pas le parseur. Il vit dans
// `analysis.ParseHighlightEvents`, il est testé là-bas, et une seconde implémentation
// divergerait — la règle du dépôt sur les copies vaut aussi pour les décodeurs.

// ScanFilmDeaths lit le fil des morts du film de filmDir.
//
// HORS LIGNE (I/O disque) — jamais depuis un chemin de requête ; l'API sert l'artefact
// pré-construit.
func ScanFilmDeaths(filmDir string) ([]Death, error) {
	n := filmdec.CountFilmChunks(filmDir)
	if n == 0 {
		return nil, fmt.Errorf("aucun chunk film lisible dans %s", filmDir)
	}
	// Le chunk des highlight events est le DERNIER du manifest : c'est sa définition, pas
	// une constante à deviner par film.
	raw, err := os.ReadFile(filepath.Join(filmDir, fmt.Sprintf("chunk_%02d.bin", n)))
	if err != nil {
		return nil, fmt.Errorf("chunk highlight (%d) : %w", n, err)
	}
	evs, err := analysis.ParseHighlightEvents(raw, 0)
	if err != nil {
		return nil, fmt.Errorf("chunk highlight (%d) : %w", n, err)
	}
	out := make([]Death, 0, len(evs))
	for _, e := range evs {
		if e.EventType != analysis.EventTypeDeath {
			continue
		}
		out = append(out, Death{XUID: e.XUID, Gamertag: e.Gamertag, TimeMS: int64(e.TimeMS)})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("chunk highlight (%d) : aucune mort", n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TimeMS < out[j].TimeMS })
	return out, nil
}
