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
	"log/slog"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/port"
)

// DepsCapture porte ce dont la capture a besoin. Les deux champs sont exiges ensemble : sans
// bornes de carte un quantum n'est pas une coordonnee, et sans identite de carte on ne sait pas
// quelles bornes chercher.
type DepsCapture struct {
	MapNames port.ReplayMapNameRepo
	Bounds   *decfilm.MapQuantCatalog
	// Libelles et Objectifs : ce que la lecture des PORTEURS du placement des vies demande (plan
	// Emprise vies, lot V2) — le catalogue de libelles du titre (objets d'objectif du drapeau) et
	// le catalogue d'objectifs de carte (socles de drapeau), les MEMES fichiers que la cuisson.
	// OPTIONNELS, a la difference des deux premiers : leur absence degrade le calque du drapeau
	// comme a la cuisson, jamais les positions.
	Libelles  replay.LabelCatalog
	Objectifs *replay.MapObjectivesCatalog
	// Portee : la portee du radar par variante (`regulation.toml [radar_range_m]`), resolue par
	// `mappings.PorteeDuRadar` sur la MEME table que la lecture (plan Emprise vies, lot V2b).
	// OPTIONNELLE comme les deux precedentes : nil = aucune table lisible, les lignes du
	// placement s'ecrivent sans portee (`radar_m` et `beyond_ms` NULL) et se comptent.
	Portee PorteeDuRadar
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
	catalogue, err := decfilm.LoadMapQuantCatalog(chemin)
	if err != nil {
		return DepsCapture{}, fmt.Errorf("capture positions %s: catalogue de bornes (%s): %w",
			titleSlug, chemin, err)
	}
	deps := DepsCapture{MapNames: mapNames, Bounds: catalogue}
	deps.Libelles, deps.Objectifs = cataloguesDuPlacement(repoRoot, titleSlug)
	deps.Portee = porteeDuTitre(repoRoot, titleSlug)
	return deps, nil
}

// porteeDuTitre charge la table des portees du radar du titre par LE chargeur du registre des
// mappings (`mappings.LoadRegulationForTitle` : meme chemin, meme validation que
// `server_apiv1` -> `LoadFromConfigDir` -> `GetRegulation`), et la resout par le helper unique.
//
// BEST-EFFORT, comme les deux catalogues des porteurs : un fichier illisible ou absent se
// JOURNALISE puis degrade en « aucune portee » — les vies s'ecrivent quand meme, sans part hors
// radar, et chaque match se compte `killsource_placement_matchs_sans_portee`.
func porteeDuTitre(repoRoot, titleSlug string) PorteeDuRadar {
	reglement, err := mappings.LoadRegulationForTitle(repoRoot, titleSlug)
	if err != nil || reglement == nil {
		slog.Warn("killsource: placement — regulation.toml illisible ou absent, vies sans portee "+
			"du radar", "titleSlug", titleSlug, "path", mappings.RegulationPath(repoRoot, titleSlug),
			"err", err)
		return nil
	}
	table := reglement.RadarRangeMap()
	return func(variante string) (float64, bool) { return mappings.PorteeDuRadar(table, variante) }
}

// cataloguesDuPlacement charge les deux catalogues des porteurs, BEST-EFFORT : un fichier
// illisible se JOURNALISE (installation incomplete — ils sont versionnes) puis degrade, comme a la
// cuisson (`replaybuild.objectivesCatalog`). Les positions et les vies n'en dependent pas.
func cataloguesDuPlacement(repoRoot, titleSlug string) (replay.LabelCatalog, *replay.MapObjectivesCatalog) {
	libelles, err := replaylabels.Load(repoRoot, titleSlug)
	if err != nil {
		slog.Warn("killsource: placement — catalogue de libelles illisible, porteurs du drapeau "+
			"sans objets d'objectif nommes", "titleSlug", titleSlug, "err", err)
	}
	chemin := titlePkg.NewPathResolver(repoRoot).MapObjectivesPath(titleSlug)
	objectifs, err := replay.LoadMapObjectives(chemin)
	if err != nil {
		slog.Warn("killsource: placement — catalogue d'objectifs illisible, drapeaux sans equipe "+
			"proprietaire", "titleSlug", titleSlug, "path", chemin, "err", err)
		objectifs = nil
	}
	return libelles, objectifs
}

// AvecCapture applique les deps au collecteur si elles sont completes. Chainable, no-op sinon —
// l'appelant a deja journalise la degradation.
//
// LA PORTEE DU RADAR VOYAGE ICI (plan Emprise vies, lot V2b) : les trois lieux de naissance du
// collecteur (etape post-sync, `backfill-killsource`, `--online`) appliquent tous la capture
// (garde-rail `archlint/no_collecteur_sans_capture_test.go`), ils ont donc la portee sans code
// de plus. `TestAvecCapture_PoseLaPorteeDuRadar` echoue si cette ligne disparait.
func (c *KillSourceCollector) AvecCapture(d DepsCapture) *KillSourceCollector {
	if !d.Cablee() {
		return c
	}
	c.placement.libelles, c.placement.objectifs = d.Libelles, d.Objectifs
	return c.AvecPorteeDuRadar(d.Portee).WithPositionCapture(d.MapNames, d.Bounds)
}

// CaptureCablee dit si CE collecteur produira des positions (et donc des faits d'isolement).
//
// EXPOSEE POUR LES TESTS DE CABLAGE : le defaut P0-1 etait invisible parce qu'aucun test ne
// pouvait constater qu'un collecteur construit par le serveur n'avait pas ses dependances.
func (c *KillSourceCollector) CaptureCablee() bool {
	return c != nil && c.mapNames != nil && c.mapBounds != nil
}
