package handlers

import (
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// sessionsRecalcPaths rend la base joueur et la base partagée du TITRE d'un profil, pour le
// recalcul des sessions. La route parcourt les profils de tous les titres : ses chemins se
// résolvent par profil. Un titre vide (profil sans titre) vaut le titre par défaut, comme
// partout dans config.PlayerDBPath / config.SharedDBPath.
func sessionsRecalcPaths(cfg *config.AppConfig, p domain.PlayerSummary) (playerDBPath, sharedDBPath string) {
	return config.PlayerDBPath(cfg, p.TitleSlug, p.Gamertag), config.SharedDBPath(cfg, p.TitleSlug)
}
