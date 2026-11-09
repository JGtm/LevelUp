package service

import (
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// buildAppInfo rend la version et la racine des données pour la fenêtre « À propos », seulement
// quand l'instance offre le chrome d'application de bureau : une instance hébergée ne publie pas
// le chemin de ses données.
func buildAppInfo(cfg *config.AppConfig) *domain.AppInfo {
	if !cfg.Capabilities.CompactChrome {
		return nil
	}
	return &domain.AppInfo{Version: cfg.AppVersion, DataRoot: cfg.Roots().Data}
}
