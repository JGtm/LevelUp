package wire

// registry_pages_matchview.go — LE CABLAGE DES BLOCS DU FILM DE LA VUE MATCH (distance par arme,
// Emprise du match ; plan `.ai/V7.5/PLAN_MATCHVIEW_EMPRISE_2026-10-06.md`, D18), appele par la factory
// `MatchView` (registry_pages.go, qui depasse deja le seuil de taille du depot). Chaque dependance
// sous la porte de sa donnee ; jamais slug==.

import (
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// cablerFilmMatchView pose les dependances des blocs du film de la Vue match.
func (r *ServiceRegistry) cablerFilmMatchView(svc *service.MatchViewService, pdb *duckdb.PlayerDB) *service.MatchViewService {
	if repo := r.killDistanceRepoFor(pdb); repo != nil {
		svc = svc.WithKillDistanceRepo(repo)
	}
	// La feuille de match (frags aux armes speciales) est ecrite par tous les titres : cablage
	// INCONDITIONNEL, comme l'Escouade et les Series temporelles. La portee du radar aussi (table
	// vide pour un titre sans mesure : les vies d'un match sans portee sont ecartees et comptees).
	svc = svc.WithEmpriseSheet(duckdb.NewSquadEmpriseRepo(pdb, r.killSourceClassifierFor(pdb))).
		WithRadarRange(r.radarRangeFor(pdb))
	// Les frags par categorie de source du film (objet explosif, chute) : le MEME lecteur d'armes que
	// Sessions et l'Escouade ; un lecteur qui ne sait pas les lire laisse les deux lignes au reliquat.
	if cats, ok := r.weaponKillsRepoFor(pdb).(port.KillSourceCategoryRepository); ok {
		svc = svc.WithKillSourceCategories(cats)
	}
	caps := r.capabilitiesForPDB(pdb)
	// Le resume d'usage : la porte film.usage_summary (absente pour Halo 5 -> l'Emprise dit
	// film_unsupported, seule la feuille reste).
	if caps.Has(games.CapFilmUsageSummary) {
		svc = svc.WithEmpriseUsageSummary(duckdb.NewSessionUsageRepo(pdb), r.cfg.RepoRoot)
	}
	// Les vies de l'equipe : la porte des vies et du contexte des morts au sync, film.kill_positions.
	if caps.Has(games.CapFilmKillPositions) {
		svc = svc.WithCampLives(duckdb.NewSoloLivesRepo(pdb))
	}
	// La ressource vehicules : la capability fine film.vehicle_usage.
	if caps.Has(games.CapFilmVehicleUsage) {
		svc = svc.WithEmpriseVehicles(duckdb.NewSquadVehicleRepo(pdb, r.killSourceClassifierFor(pdb)))
	}
	return svc
}
