package wire

// registry_pages_sessions.go — LE CABLAGE DES BLOCS DU FILM DE LA PAGE SESSIONS (Emprise du joueur,
// vies, feuille d'objectif, emblème, score des lignes ; plan `.ai/V7.5/PLAN_SESSIONS_EMPRISE_2026-10-06.md`,
// lot S2.9), appelé par la factory `SessionPage` (registry_pages.go, qui dépasse déjà le seuil de
// taille du dépôt), résumé d'usage compris (film.usage_summary).

import (
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// cablerBlocsSessions pose les dépendances des blocs du film de la colonne de session.
func (r *ServiceRegistry) cablerBlocsSessions(svc *service.SessionPageService, pdb *duckdb.PlayerDB) *service.SessionPageService {
	// La feuille de match (frags aux armes spéciales) est écrite par tous les titres, et le joueur
	// sert aussi la coordination : câblage INCONDITIONNEL. Jamais slug==.
	svc = svc.WithSessionEmprise(duckdb.NewSquadEmpriseRepo(pdb, r.killSourceClassifierFor(pdb)), pdb.XUID).
		// Emblème de la fiche « Ma part à l'objectif » : le MÊME chargeur que l'Escouade et les Séries
		// temporelles ; dégradation silencieuse (initiale) par contrat du chargeur.
		WithSessionEmblemLoader(duckdb.NewSquadV2LoaderAdapter(r.resolveByGT)).
		// « Mes vies » : la portée COURANTE du radar de chaque match (MÊME table que la Tactique).
		WithSessionRadarRange(r.radarRangeFor(pdb)).
		// Score des lignes de match : manches pour les variantes déclarées (ADR 0032).
		WithRoundsDecide(r.roundsDecideFor(pdb))
	caps := r.capabilitiesForPDB(pdb)
	// Résumé d'usage (Emprise, objectif, effectif de camp de la coordination) et catalogues du
	// titre : film.usage_summary.
	if caps.Has(games.CapFilmUsageSummary) {
		svc = svc.WithSessionUsageSummary(duckdb.NewSessionUsageRepo(pdb), r.cfg.RepoRoot)
	}
	// Les vies du joueur : MÊME porte que les vies au sync, film.kill_positions.
	if caps.Has(games.CapFilmKillPositions) {
		svc = svc.WithSessionLives(duckdb.NewSoloLivesRepo(pdb))
	}
	// Ressource véhicules : la capability fine film.vehicle_usage, comme l'Escouade.
	if caps.Has(games.CapFilmVehicleUsage) {
		svc = svc.WithSessionVehicleUsage(duckdb.NewSquadVehicleRepo(pdb, r.killSourceClassifierFor(pdb)))
	}
	// Feuille d'objectif : les colonnes d'objectif quand le titre les publie (match.objective.stats).
	if caps.Has(games.CapMatchObjectiveStats) {
		svc = svc.WithSessionObjectives(duckdb.NewObjectiveStatsRepo(pdb))
	}
	return svc
}
