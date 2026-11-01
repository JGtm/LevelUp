// server_desktop.go — le montage de l'entrée de la coquille desktop (ADR 0045 D-3) : le garde
// (Host strict, cookie exigé), la route /desktop/enter et les libellés de la coquille. Tout est
// conditionné par la capacité ShellEntryGate du profil de déploiement ; hors de cette capacité le
// routeur est inchangé.
package api

import (
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
)

// desktopGateFor construit l'entrée de la coquille du port lié de cfg (SetBoundPort a déjà posé le
// port ; la clé du cookie est le secret de session du fichier).
func desktopGateFor(cfg *config.AppConfig) *middleware.DesktopGate {
	return middleware.NewDesktopGate(cfg.DesktopAuthority(), cfg.APIPort, cfg.SessionSecret, cfg.DesktopSecret)
}

// useDesktopGate monte le garde (Host strict puis cookie exigé) sur le routeur racine, sous
// l'entrée de la coquille seulement. Appelé juste après PreserveRemoteAddr : une requête refusée ne
// voit rien d'autre de la pile.
func useDesktopGate(r chi.Router, cfg *config.AppConfig) {
	if cfg.Capabilities.ShellEntryGate {
		r.Use(desktopGateFor(cfg).Middleware)
	}
}

// mountDesktopEntry monte GET /desktop/enter à la racine, derrière LoopbackOnly (la décision porte
// sur le pair TCP, que le demandeur ne choisit pas), sous l'entrée de la coquille seulement.
func mountDesktopEntry(r chi.Router, cfg *config.AppConfig) {
	if cfg.Capabilities.ShellEntryGate {
		r.With(middleware.NoStore, middleware.LoopbackOnly).Get(middleware.DesktopEnterPath, desktopGateFor(cfg).EnterHandler)
	}
}

// mountDesktopStrings monte GET /api/v1/desktop/strings (sous le préfixe /api/v1 de r), sous
// l'entrée de la coquille seulement ; le cookie est exigé par le garde racine.
func mountDesktopStrings(r chi.Router, cfg *config.AppConfig, opts ...humacore.MountOption) {
	if cfg.Capabilities.ShellEntryGate {
		handlers.NewDesktopStringsHandler().Mount(r, opts...)
	}
}
