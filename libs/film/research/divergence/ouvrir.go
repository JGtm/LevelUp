//go:build research

package divergence

// ouvrir.go : l OUVERTURE D UN FILM par l instrument, sous l un de deux contextes.
//
//   - SANS CARTE (`grammar.ContexteDeFilm`, defaut, celui de la carte de fermeture A0.1) : les largeurs
//     d axe de i0 sont LUES DANS LE FILM, aucune entree de catalogue n est posee sur le profil. C est une
//     enveloppe d instrument, HORS PRODUCTION ; sans entree, les tables de position de la vue A (genres 5 et
//     6) sont vides et la lecture s y arrete.
//   - SOUS CARTE (`DIVERGENCE_CARTE=<map_quant_bounds.json>`) : la voie de la cuisson,
//     `grammar.NewFilmContextForMap` avec l entree du catalogue de la carte du match. Le niveau du film
//     (colonne 4 de la liste des films) designe l entree : la premiere dont `module` le porte.

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// entreeDuNiveau rend l entree du catalogue dont le module est `niveau` (ordre des cles : le resultat est
// deterministe ; toutes les entrees d un meme module portent les memes bornes).
func entreeDuNiveau(cat *profile.MapQuantCatalog, niveau string) (*profile.MapQuantEntry, error) {
	noms := make([]string, 0, len(cat.Maps))
	for nom := range cat.Maps {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	for _, nom := range noms {
		if e := cat.Maps[nom]; e.Module == niveau {
			return &e, nil
		}
	}
	return nil, fmt.Errorf("aucune entree de catalogue pour le niveau %q", niveau)
}

// ouvrirFilm ouvre le contexte du film `id` sous la racine, avec ou sans catalogue.
func ouvrirFilm(racine, id, niveau string, cat *profile.MapQuantCatalog) (*grammar.FilmContext, error) {
	dir := filepath.Join(racine, id)
	if cat == nil {
		fc, _, err := grammar.ContexteDeFilm(dir)
		if fc == nil {
			return nil, fmt.Errorf("film illisible : %w", err)
		}
		return fc, nil
	}
	entree, err := entreeDuNiveau(cat, niveau)
	if err != nil {
		return nil, err
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	fc := grammar.NewFilmContextForMap(film, entree, nil)
	fc.PoserLaCarteEtLeDecoupage() // le geste de replay.installWorldObjectPrecision, apres le constructeur
	return fc, nil
}

// entreesDuNiveau rend TOUTES les entrees du catalogue dont le module est `niveau`, dans l ordre des cles.
func entreesDuNiveau(cat *profile.MapQuantCatalog, niveau string) []profile.MapQuantEntry {
	noms := make([]string, 0, len(cat.Maps))
	for nom := range cat.Maps {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	var out []profile.MapQuantEntry
	for _, nom := range noms {
		if e := cat.Maps[nom]; e.Module == niveau {
			out = append(out, e)
		}
	}
	return out
}
