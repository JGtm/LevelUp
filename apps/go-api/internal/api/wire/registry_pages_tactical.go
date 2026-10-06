package wire

// registry_pages_tactical.go — LA FACTORY DE L'ONGLET TACTIQUE, sortie de registry_pages.go (au-delà
// du seuil de taille du dépôt) quand le détail d'une zone a gagné ses sources (plan Tactique v2,
// L3.5). Garde-rail : registry_pages_tactical_wiring_test.go.

import (
	"context"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// Tactical retourne un TacticalService pour le joueur : l'onglet Tactique
// (lectures de placement par carte + KPI d'echange). UN SEUL endroit de
// construction, comme replayServiceFor.
//
// La portee du RADAR (regulation.toml [radar_range_m]) est injectee par titre : elle borne
// la lecture « ou je meurs isole ». Titre ou variante absents -> pas de lecture pour ces
// matchs, et le compte des ecartes le dit.
//
// Multi-titre : les trois portes data-level (`film.kill_positions` pour les
// lectures de placement, `film.kill_source` pour l'echange, `film.replay_artifact`
// pour l'occupation) sont lues sur la CapabilityMap de
// l'adapter du titre du joueur (capabilitiesForPDB → dataAdapterForPDB, avec repli
// sur les capabilities HI du boot). JAMAIS une comparaison de slug. Un titre qui
// n'expose pas les positions rend ErrCapabilityNotSupported → 503 propre.
//
// AUCUNE TAXONOMIE DE MODES (retrait phase 4 bis, 2026-09-06) : le lecteur tactique
// ne filtre plus par mode. Son périmètre est une liste blanche de match_id, résolue
// en amont par le pipeline de filtres sur la base joueur.
func (r *ServiceRegistry) Tactical(ctx context.Context, slug string) (port.TacticalService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	repo := duckdb.NewTacticalRepo(pdb)
	// LE LECTEUR DE SIDECARS D'OCCUPATION (phase 6) : la seule source de l'onglet qui ne
	// soit pas une base. Il est monte ici, au seul endroit de construction du service —
	// sans lui, la lecture « ou je passe mon temps » degrade en 503 en le disant.
	rasters := service.NewTacticalRasterStore(r.cfg.RuntimePaths().RepoRoot(), pdb.TitleSlug)
	// LES ZONES NOMMEES viennent du MEME catalogue versionne que le rejeu 2D, par la MEME
	// cascade (module puis asset UGC) : elles nomment les grappes de reapparition. Magasin
	// nil impossible ici ; catalogue absent -> grappes MUETTES, jamais une erreur.
	callouts := service.NewTacticalCalloutsStore(r.cfg.RepoRoot, pdb.TitleSlug,
		duckdb.NewReplayMapRepo(pdb.SharedReadDB(), pdb.Metadata))
	return service.NewTacticalService(repo, r.capabilitiesForPDB(pdb), pdb.XUID).
		WithRasterStore(rasters).
		WithCalloutsStore(callouts).
		WithRetentionMois(r.retentionMoisRejeu).
		WithRadarRange(r.radarRangeFor(pdb)).
		// LES SOURCES DU DÉTAIL D'UNE ZONE, toutes sans condition : chacune retire son champ quand elle
		// manque. Le classificateur est nil sur un titre sans film.kill_source (killSourceClassifierFor
		// le décide sur les capabilities du titre) : ni arme ni catégorie, jamais slug==.
		WithPlayerMatches(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithRoundsDecide(r.roundsDecideFor(pdb)).
		WithKillSourceClassifier(r.killSourceClassifierFor(pdb)).
		WithWeaponLabels(duckdb.NewWeaponRangeRepo(pdb, r.killSourceClassifierFor(pdb))).
		WithReplay(r.replayServiceFor(pdb)), nil
}
