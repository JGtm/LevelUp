package replay

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
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
//
// ENVELOPPE D2, HORS PRODUCTION (lot 1, 2026-09-02) : la cuisson appelle [ScanDeaths] sur un
// film déjà chargé.
func ScanFilmDeaths(filmDir string) ([]Death, error) {
	film, err := filmsource.LoadDir(filmDir, nil)
	if err != nil {
		return nil, err
	}
	return ScanDeaths(film)
}

// ScanDeaths lit le fil des morts d'un film DEJA CHARGE.
//
// LES OCTETS SONT DEJA DECOMPRESSES, et `analysis.ParseHighlightEvents` l'accepte : il tente un
// `zlib.NewReader` et, s'il echoue, traite l'entree comme du clair — c'est la double tolerance
// qu'il porte depuis l'incident du 2026-05-22 (le cache historique stockait les chunks
// compresses, les telechargements recents ne le font plus). Lui donner le chunk deja inflate
// rend donc EXACTEMENT les memes evenements, sans une seconde decompression du plus gros chunk
// du film.
func ScanDeaths(film *filmsource.Film) ([]Death, error) {
	nums := filmdec.FilmChunkNumbers(film)
	if len(nums) == 0 {
		return nil, filmdec.ErrNoReadableFilmChunk
	}
	// Le chunk des highlight events est le DERNIER du manifest : c'est sa définition, pas
	// une constante à deviner par film.
	n := nums[len(nums)-1]
	raw, _, ok := filmdec.FilmChunkAt(film, n)
	if !ok {
		return nil, fmt.Errorf("chunk highlight (%d) : absent du film", n)
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
