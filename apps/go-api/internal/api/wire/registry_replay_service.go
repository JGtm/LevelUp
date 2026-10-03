// Package wire — registry_replay_service.go : construction du service de rejeu 2D.
//
// Sorti de registry_pages.go (fichier au-delà du seuil de 500 lignes, dette gelée) au lot
// B-C7 du backlog 2026-09-26.
package wire

import (
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// replayServiceFor construit le service de rejeu d'un joueur — UN SEUL endroit, partagé par
// l'endpoint /replay et la Match View (qui n'en appelle qu'IsAvailable). Deux constructions
// divergentes, ce serait une Match View qui annonce un rejeu que l'endpoint ne sert pas.
//
// La résolution de carte (fond de carte) lit le registre partagé et les traductions d'assets ;
// elle est passée au service, jamais reconstruite ailleurs.
func (r *ServiceRegistry) replayServiceFor(pdb *duckdb.PlayerDB) port.ReplayService {
	maps := duckdb.NewReplayMapRepo(pdb.SharedReadDB(), pdb.Metadata)
	return replayServiceFrom(r.cfg, pdb.TitleSlug, maps)
}

// replayServiceFrom : la construction elle-même, sans base, pour qu'un test l'exerce telle
// que la production l'appelle.
//
// DEUX RACINES (lot B-C7). Les DONNÉES VERSIONNÉES (fonds de carte, catalogues de référence,
// mappings, libellés, zones, règles de tiers) se lisent à la racine du dépôt, en démo comme
// ailleurs. Les ARTEFACTS D'EXÉCUTION (rejeux construits) se lisent sous cfg.RuntimePaths :
// `<démo>/runtime/` en démo (lot B5.5), le dépôt sinon. Tout enraciner sous RuntimePaths
// rendait les fonds et les catalogues introuvables en démo (régression de B5, R2-1).
func replayServiceFrom(cfg *config.AppConfig, titleSlug string, maps port.ReplayMapNameRepo) port.ReplayService {
	return service.NewReplayServiceRoots(titleSlug, cfg.RepoRoot, cfg.RuntimePaths().RepoRoot(), maps)
}
