package mapcatalog

// callouts_store.go — LE CATALOGUE GÉNÉRÉ DES ZONES NOMMÉES : écrit par le runtime, jamais le
// fichier versionné.
//
// Même discipline que l'overlay des socles (store.go), et pour les mêmes raisons :
//
//	DEUX FICHIERS       `reference/map_callouts.json` est VERSIONNÉ (produit à la main par
//	                    `cmd/mapcallouts-build`, relu en revue) ; `reference/generated/
//	                    map_callouts.json` est la SORTIE du runtime, ignorée par git. Le
//	                    runtime n'écrit QUE celle-ci (`AddCalloutsOverlayEntry`), et le
//	                    garde-rail `archlint/no_runtime_versioned_catalog_write_test.go`
//	                    interdit de lui repasser le chemin versionné.
//	AJOUT SEUL          une carte déjà au catalogue généré n'est jamais réécrite : ce que
//	                    l'application sert sur une carte déjà rattrapée ne change pas sans
//	                    qu'on le décide.
//	ATOMIQUE + VERROU   temporaire à nom unique puis `rename`, sous le verrou consultatif du
//	                    paquet : la CLI et un cycle de sync peuvent écrire en même temps sans
//	                    perdre la carte de l'autre.
//
// LA FORME EST CELLE DU CATALOGUE VERSIONNÉ (`replay.MapCalloutsCatalog`), section `maps_by_id`
// seule : un seul lecteur, une seule forme, et une carte relue en revue peut être promue au
// fichier versionné par simple copie de son entrée.

import (
	"errors"
	"fmt"
	"io/fs"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// sourceCatalogueGenere : le champ `source` du catalogue généré — d'où viennent ses entrées.
const sourceCatalogueGenere = "runtime : zones nommées lues dans la variante .mvar de la carte " +
	"(mapcatalog.EntreeCalloutsForge), libellés du lexique versionné"

// AddCalloutsOverlayEntry ajoute UNE carte au catalogue GÉNÉRÉ des zones si son map_id n'y est
// pas encore. `overlay` est le chemin du catalogue généré (PathResolver.MapCalloutsOverlayPath),
// jamais celui du catalogue versionné.
//
// Le catalogue généré absent est le cas NOMINAL du premier rattrapage : il est créé. Un
// catalogue généré ILLISIBLE fait échouer : l'écraser effacerait les cartes déjà rattrapées.
//
// Rend `ErrEntryExists` quand la carte y est déjà — y compris quand elle y est arrivée entre le
// moment où l'appelant a constaté son absence et celui-ci.
func AddCalloutsOverlayEntry(overlay, titleSlug, mapID string, entry replay.MapCalloutsEntry) error {
	if mapID == "" {
		return fmt.Errorf("catalogue généré des zones : map_id vide")
	}
	defer prendreVerrou(overlay)()
	cat, err := ChargerCatalogueGenere(overlay, titleSlug)
	if err != nil {
		return err
	}
	if _, deja := cat.MapsByID[mapID]; deja {
		return ErrEntryExists
	}
	cat.MapsByID[mapID] = entry
	return ecrireJSONAtomique(cat, overlay)
}

// ChargerCatalogueGenere lit le catalogue généré des zones. ABSENT, il rend un catalogue VIDE
// sans erreur (aucune carte rattrapée, l'état d'une instance neuve) ; illisible ou d'une autre
// version de schéma, il rend l'erreur.
func ChargerCatalogueGenere(path, titleSlug string) (*replay.MapCalloutsCatalog, error) {
	cat, err := replay.LoadMapCallouts(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return &replay.MapCalloutsCatalog{
			SchemaVersion: replay.MapCalloutsSchemaVersion,
			TitleSlug:     titleSlug,
			Source:        sourceCatalogueGenere,
			Maps:          map[string]replay.MapCalloutsEntry{},
			MapsByID:      map[string]replay.MapCalloutsEntry{},
		}, nil
	case err != nil:
		return nil, fmt.Errorf("catalogue généré des zones illisible : %w", err)
	}
	if cat.MapsByID == nil {
		cat.MapsByID = map[string]replay.MapCalloutsEntry{}
	}
	return cat, nil
}
