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
	auth_platform "levelup/go-api/internal/platform/auth"
	session_platform "levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/replayregen"
	"levelup/go-api/internal/resourcepack"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/transfer"
)

// DesktopGateFor construit l'entrée de la coquille du port lié de cfg (SetBoundPort a déjà posé le
// port ; la clé du cookie est le secret de session du fichier). Exportée pour la porte de démarrage
// (cmd/server), qui applique le même Host strict, avec la même exemption, avant que le routeur existe.
//
// Sous la capacité LoginViaBrowser, le retour de Microsoft (GET /auth/xbox/callback) est la seule route
// qui passe sans cookie : le navigateur par défaut n'a pas celui de la coquille.
func DesktopGateFor(cfg *config.AppConfig) *middleware.DesktopGate {
	g := middleware.NewDesktopGate(cfg.DesktopAuthority(), cfg.APIPort, cfg.SessionSecret, cfg.DesktopSecret)
	if cfg.Capabilities.LoginViaBrowser {
		g.AllowBrowserCallback()
	}
	return g
}

// useDesktopGate monte le garde (Host strict puis cookie exigé) sur le routeur racine, sous
// l'entrée de la coquille seulement. Appelé juste après PreserveRemoteAddr : une requête refusée ne
// voit rien d'autre de la pile.
func useDesktopGate(r chi.Router, cfg *config.AppConfig) {
	if cfg.Capabilities.ShellEntryGate {
		r.Use(DesktopGateFor(cfg).Middleware)
	}
}

// mountDesktopEntry monte GET /desktop/enter à la racine, derrière LoopbackOnly (la décision porte
// sur le pair TCP, que le demandeur ne choisit pas), sous l'entrée de la coquille seulement.
func mountDesktopEntry(r chi.Router, cfg *config.AppConfig) {
	if cfg.Capabilities.ShellEntryGate {
		r.With(middleware.NoStore, middleware.LoopbackOnly).Get(middleware.DesktopEnterPath, DesktopGateFor(cfg).EnterHandler)
	}
}

// mountDesktopStrings monte GET /api/v1/desktop/strings (sous le préfixe /api/v1 de r), sous
// l'entrée de la coquille seulement ; le cookie est exigé par le garde racine.
func mountDesktopStrings(r chi.Router, cfg *config.AppConfig, opts ...humacore.MountOption) {
	if cfg.Capabilities.ShellEntryGate {
		handlers.NewDesktopStringsHandler().Mount(r, opts...)
	}
}

// mountDesktopDiagnostic monte POST /api/v1/desktop/diagnostic sous la capacité DiagnosticReport ; le cookie de
// la coquille est exigé par le garde racine.
func mountDesktopDiagnostic(r chi.Router, cfg *config.AppConfig, opts ...humacore.MountOption) {
	if cfg.Capabilities.DiagnosticReport {
		handlers.NewDesktopDiagnosticHandler(cfg).Mount(r, opts...)
	}
}

// newBrowserLogin construit le handler de la connexion par le navigateur sous la capacité
// LoginViaBrowser (nil sinon : aucune route montée). Le port est celui que le serveur a lié : c'est
// lui que porte l'adresse de retour déclarée à Microsoft.
func newBrowserLogin(
	cfg *config.AppConfig, sessions *session_platform.Store, attempts *auth_platform.AttemptStore,
	provider auth_platform.TokenProvider,
) *handlers.BrowserLoginHandler {
	if !cfg.Capabilities.LoginViaBrowser {
		return nil
	}
	return handlers.NewBrowserLoginHandler(sessions, attempts, provider, cfg.APIPort)
}

// mountBrowserLoginStart monte GET /auth/xbox/login (Huma) sous /api/v1 (la page de la coquille l'appelle
// en fetch avec son cookie) : sans effet quand la connexion par le navigateur n'est pas offerte.
func mountBrowserLoginStart(r chi.Router, h *handlers.BrowserLoginHandler, opts ...humacore.MountOption) {
	if h != nil {
		h.MountStart(r, opts...)
	}
}

// mountBrowserLoginCallback monte GET /auth/xbox/callback à la RACINE (l'adresse déclarée côté Azure),
// derrière LoopbackOnly. Le garde de la coquille la laisse passer sans cookie (AllowBrowserCallback) ;
// le `state` à usage unique de la tentative est sa protection.
func mountBrowserLoginCallback(r chi.Router, h *handlers.BrowserLoginHandler) {
	if h != nil {
		r.With(middleware.NoStore, middleware.LoopbackOnly).Get(handlers.BrowserLoginCallbackPath, h.Callback)
	}
}

// mountSiteImport monte l'import des données depuis une instance hébergée (POST et GET /import/site) sous la
// capacité ImportFromSite ; le cookie de la coquille est exigé par le garde racine.
func mountSiteImport(r chi.Router, cfg *config.AppConfig, opts ...humacore.MountOption) {
	if cfg.Capabilities.ImportFromSite {
		handlers.NewSiteImportHandler(service.DefaultSiteImport()).Mount(r, opts...)
	}
}

// mountExportDesktop monte l'export des données du joueur pour l'application de bureau sous la capacité
// ExportForDesktop (instance hébergée) : la route du jeton lit l'identité Xbox liée à la session et refuse
// (401) sans elle, l'archive s'authentifie par jeton ou par session (handlers/export_desktop.go).
func mountExportDesktop(r chi.Router, cfg *config.AppConfig, opts ...humacore.MountOption) {
	if !cfg.Capabilities.ExportForDesktop {
		return
	}
	h := handlers.NewExportDesktopHandler(cfg, transfer.NewTokenStore())
	h.MountToken(r.With(middleware.NoStore), opts...)
	h.MountExport(r)
}

// mountDesktopResources monte GET /api/v1/desktop/resources et POST .../retry sous la capacité UpdateApp (le paquet de
// ressources se met à jour avec l'application) ; le cookie de la coquille est exigé par le garde racine.
func mountDesktopResources(r chi.Router, cfg *config.AppConfig, opts ...humacore.MountOption) {
	if cfg.Capabilities.UpdateApp {
		handlers.NewDesktopResourcesHandler(func() handlers.ResourcesService {
			if m := resourcepack.Default(); m != nil {
				return m
			}
			return nil
		}).Mount(r, opts...)
	}
}

// mountDesktopReplayRegen monte les routes de la régénération des rejeux (GET, POST start|pause|resume sous
// /api/v1/desktop/replay-regen) sous la capacité ReplayRegen ; le cookie de la coquille est exigé par le garde racine.
func mountDesktopReplayRegen(r chi.Router, cfg *config.AppConfig, opts ...humacore.MountOption) {
	if cfg.Capabilities.ReplayRegen {
		handlers.NewDesktopReplayRegenHandler(replayregen.Default()).Mount(r, opts...)
	}
}
