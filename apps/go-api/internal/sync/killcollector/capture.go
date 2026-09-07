package killcollector

// capture.go — LES DEUX DEPENDANCES DE LA CAPTURE DES POSITIONS, construites UNE FOIS.
//
// # POURQUOI CE FICHIER EXISTE (revue de 7C, P0-1)
//
// `WithPositionCapture` n'etait appele QUE par `levelup backfill-killsource`. Ni l'etape
// post-sync du serveur ni `backfill-killsource --online` ne le cablaient : `collectPositions`
// sortait en Debug des sa deuxieme garde, et AUCUNE position — ni, depuis le lot 7C, aucun fait
// d'isolement — n'etait jamais ecrite au fil du sync en production. Le defaut etait
// PREEXISTANT pour `kill_positions` (jamais produites au sync depuis leur mise en place) ; le
// lot 7C le rendait bloquant pour son propre objectif.
//
// Le cablage vit desormais ICI, et les trois chemins l'appellent.
//
// # CE QUE CE PAQUET NE FAIT PAS
//
// Il ne construit PAS le resolveur de carte : celui-la demande `platform/duckdb`, et
// `killcollector` ne depend d'aucune base concrete (il ne connait que `port` et `*sql.DB`).
// L'appelant le fournit — c'est lui qui sait quel handle metadata il a le droit d'ouvrir, et la
// reponse n'est pas la meme selon qu'un serveur tourne ou non (modele mono-process, ADR 0013).

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// DepsCapture porte ce dont la capture a besoin. Les deux champs sont exiges ensemble : sans
// bornes de carte un quantum n'est pas une coordonnee, et sans identite de carte on ne sait pas
// quelles bornes chercher.
type DepsCapture struct {
	MapNames port.ReplayMapNameRepo
	Bounds   *filmdec.MapQuantCatalog
}

// Cablee dit si les deux dependances sont la.
func (d DepsCapture) Cablee() bool { return d.MapNames != nil && d.Bounds != nil }

// CaptureDepuisCatalogue charge le catalogue de bornes du titre et l'apparie au resolveur de
// carte fourni.
//
// LE CATALOGUE EST UNE DONNEE DE REFERENCE VERSIONNEE
// (`data/titles/{slug}/reference/map_quant_bounds.json`, ~22 Ko commites), pas une sortie de
// sync : il est disponible meme sur une installation sans historique. Le chemin passe par
// `PathResolver` (CLAUDE.md : jamais de `filepath.Join(..., "data", ...)` a la main).
//
// L'ERREUR EST RENDUE, PAS AVALEE : c'est a l'appelant de decider s'il degrade en « positions
// desactivees » (le cas de tous les appelants d'aujourd'hui) ou s'il refuse. Une fonction qui
// rend silencieusement des deps vides fabriquerait exactement le silence que ce lot corrige.
func CaptureDepuisCatalogue(repoRoot, titleSlug string, mapNames port.ReplayMapNameRepo) (DepsCapture, error) {
	if mapNames == nil {
		return DepsCapture{}, fmt.Errorf("capture positions %s: aucun resolveur de carte", titleSlug)
	}
	chemin := titlePkg.NewPathResolver(repoRoot).MapQuantBoundsPath(titleSlug)
	catalogue, err := filmdec.LoadMapQuantCatalog(chemin)
	if err != nil {
		return DepsCapture{}, fmt.Errorf("capture positions %s: catalogue de bornes (%s): %w",
			titleSlug, chemin, err)
	}
	return DepsCapture{MapNames: mapNames, Bounds: catalogue}, nil
}

// AvecCapture applique les deps au collecteur si elles sont completes. Chainable, no-op sinon —
// l'appelant a deja journalise la degradation.
func (c *KillSourceCollector) AvecCapture(d DepsCapture) *KillSourceCollector {
	if !d.Cablee() {
		return c
	}
	return c.WithPositionCapture(d.MapNames, d.Bounds)
}

// CaptureCablee dit si CE collecteur produira des positions (et donc des faits d'isolement).
//
// EXPOSEE POUR LES TESTS DE CABLAGE : le defaut P0-1 etait invisible parce qu'aucun test ne
// pouvait constater qu'un collecteur construit par le serveur n'avait pas ses dependances.
func (c *KillSourceCollector) CaptureCablee() bool {
	return c != nil && c.mapNames != nil && c.mapBounds != nil
}
