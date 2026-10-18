package wire

// registry_analysis_pending.go — LA PORTE DE TITRE DE L'INDICATEUR « ANALYSE EN COURS ».
//
// `analysis_pending` (tuiles de l'accueil, en-tête de la vue match) ne se calcule que pour un
// titre qui analyse des films : sa CapabilityMap déclare `film.kill_source`. Sans elle, aucun
// résolveur n'est câblé, le service rend le champ faux sans lire la moindre base — même
// patron que `withFilmArtifactRepos`. La condition « rejeu manquant » exige en plus
// `film.replay_artifact`. Jamais de `slug ==` : tout vient de la CapabilityMap du titre.

import (
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// newAnalysisPendingResolver applique la porte : nil si le titre ne déclare pas
// `film.kill_source`. La CapabilityMap et le dépôt sont PASSÉS (jamais résolus ici) pour que
// le garde-rail rejoue la décision sur les capabilities.toml LIVRÉS.
func newAnalysisPendingResolver(caps games.CapabilityMap, repo port.AnalysisProbeRepository) *service.AnalysisPendingResolver {
	if !caps.Has(games.CapFilmKillSource) {
		return nil
	}
	return service.NewAnalysisPendingResolver(repo, caps.Has(games.CapFilmReplayArtifact))
}

// analysisPendingFor est le point d'appel du wire : la CapabilityMap du titre du joueur et le
// dépôt réel.
func (r *ServiceRegistry) analysisPendingFor(pdb *duckdb.PlayerDB) *service.AnalysisPendingResolver {
	// Le verrou machine de décodage vit sous la racine du cache film (filmproc.AcquireSolo) ;
	// l'observateur n'existe que là où le résolveur existe (nil-sûr).
	cacheRoot := titlePkg.NewPathResolver(r.cfg.RepoRoot).CacheRootDir()
	return newAnalysisPendingResolver(r.capabilitiesForPDB(pdb), duckdb.NewAnalysisProbeRepo(pdb)).
		WithDecodeObserver(filmproc.NewDecodeObserver(cacheRoot))
}
