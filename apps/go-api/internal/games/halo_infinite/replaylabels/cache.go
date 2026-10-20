package replaylabels

// cache.go — LE CATALOGUE D'UN TITRE, LU UNE FOIS PAR PROCESSUS (lot perf L2, D2.6).
//
// Load lit et assemble deux TOML versionnés du titre : c'est un chargement hors ligne (cf.
// en-tête du paquet). Les blocs « servi ou gâché » et « formes retenues » de la page Escouade
// l'appelaient pourtant à CHAQUE requête, deux fois. Catalogue le lit une fois par (racine du
// dépôt, titre) et sert ensuite le même catalogue : ces fichiers sont versionnés, ils ne
// changent pas pendant la vie du processus.

import (
	"sync"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

type cleCatalogue struct{ repoRoot, titleSlug string }

var (
	cataloguesMu sync.Mutex
	catalogues   = map[cleCatalogue]replay.LabelCatalog{}
)

// Catalogue rend le catalogue du titre, lu au premier appel puis servi de mémoire.
//
// LE RÉSULTAT EST PARTAGÉ entre tous les appelants du processus : lecture seule, ses cartes ne
// se modifient jamais. Un ÉCHEC n'est pas mémorisé — l'appel suivant relit les fichiers, comme
// Load l'aurait fait.
func Catalogue(repoRoot, titleSlug string) (replay.LabelCatalog, error) {
	cle := cleCatalogue{repoRoot: repoRoot, titleSlug: titleSlug}
	cataloguesMu.Lock()
	defer cataloguesMu.Unlock()
	if cat, ok := catalogues[cle]; ok {
		return cat, nil
	}
	cat, err := Load(repoRoot, titleSlug)
	if err != nil {
		return replay.LabelCatalog{}, err
	}
	catalogues[cle] = cat
	return cat, nil
}
