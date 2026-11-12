package service

// site_import.go — l'import des données d'un joueur depuis une instance hébergée (lot L22.b, ADR 0045) côté
// application de bureau : résout le joueur suivi localement, branche les chemins d'écriture EXISTANTS de
// l'application (fournisseur de la base partagée, bail et ouverture de la base joueur : internal/importstore) sur internal/transfer,
// et tient l'état du téléchargement et de l'import pour la page des réglages.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/importstore"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/transfer"
)

// ErrSiteImportUnbound : l'import n'est pas encore relié à la configuration du serveur (Bind) ; la couche
// HTTP le répond comme toute indisponibilité (503).
var ErrSiteImportUnbound = errors.New("service: site import not bound")

// SiteImport relie l'import depuis le site aux écritures de l'application.
type SiteImport struct {
	mu     sync.Mutex
	cfg    *config.AppConfig
	ctx    context.Context
	wg     *sync.WaitGroup
	runner *transfer.Runner
	// caches : invalidation des lectures du joueur après un import (nil : aucune).
	caches port.PlayerReadCacheInvalidator
}

var defaultSiteImport = &SiteImport{runner: transfer.NewRunner()}

// DefaultSiteImport rend l'import du processus : le même pour la route de démarrage et la route d'état.
func DefaultSiteImport() *SiteImport { return defaultSiteImport }

// Bind relie l'import à la configuration et au contexte du serveur : ctx annule un import en cours à
// l'arrêt, wg le fait attendre avant la fermeture des bases, caches est invalidé une fois l'import écrit.
func (s *SiteImport) Bind(ctx context.Context, wg *sync.WaitGroup, cfg *config.AppConfig, caches port.PlayerReadCacheInvalidator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx, s.wg, s.cfg, s.caches = ctx, wg, cfg, caches
}

// Status rend l'état de l'import courant ou du dernier, en réponse d'API ; Matches est le nombre de matchs du
// registre ajoutés.
func (s *SiteImport) Status() domain.SiteImportStatus {
	st := s.runner.Status()
	return domain.SiteImportStatus{Phase: string(st.Phase), ErrorCode: st.ErrorCode, Inserted: st.Inserted,
		Matches: st.Inserted["shared/match_registry"]}
}

// Start lance l'import en arrière-plan. [ErrSiteImportUnbound] sans liaison, [port.ErrSiteImportRunning] si
// un import est en cours ; les refus de la résolution du joueur sont [port.port.ErrSiteImportNoPlayer],
// [port.port.ErrSiteImportAmbiguous] et [port.port.ErrSiteImportPlayerNoXID].
func (s *SiteImport) Start(req port.SiteImportRequest) error {
	s.mu.Lock()
	cfg, ctx, wg, caches := s.cfg, s.ctx, s.wg, s.caches
	s.mu.Unlock()
	if cfg == nil || ctx == nil {
		return ErrSiteImportUnbound
	}
	player, err := localPlayer(cfg, req.Gamertag)
	if err != nil {
		return err
	}
	slug := title.DefaultSlug
	in := transfer.RunInput{
		Fetch: transfer.FetchInput{BaseURL: req.URL, Token: req.Token},
		Import: transfer.ImportInput{
			ExpectXuid:    player.XUID,
			AcquireShared: importstore.SiteShared(cfg.SharedProvider, config.SharedDBPath(cfg, slug)),
			AcquirePlayer: importstore.SitePlayer(cfg.PathResolver().PlayerDBPath(slug, player.Gamertag)),
		},
		WorkDir: cfg.PathResolver().CacheRootDir(),
		// Les lectures du joueur déjà en cache (liste des matchs, filtres) ignorent les lignes importées : même
		// invalidation qu'en fin de synchronisation.
		AfterImport: func(ctx context.Context) {
			if caches != nil {
				caches.InvalidatePlayerReadCaches(ctx, player.XUID, slug)
			}
		},
	}
	if err := s.runner.Start(ctx, wg, in); err != nil {
		if errors.Is(err, transfer.ErrImportRunning) {
			return fmt.Errorf("%w: %w", port.ErrSiteImportRunning, err)
		}
		return err
	}
	return nil
}

// localPlayer résout le joueur suivi qui reçoit l'import : celui qu'on nomme, ou l'unique joueur suivi.
func localPlayer(cfg *config.AppConfig, gamertag string) (domainPlayer, error) {
	players, err := cfg.LoadPlayers(title.DefaultSlug)
	if err != nil {
		return domainPlayer{}, fmt.Errorf("joueurs suivis : %w", err)
	}
	var matches []domainPlayer
	for _, p := range players {
		if gamertag == "" || strings.EqualFold(p.Gamertag, gamertag) {
			matches = append(matches, domainPlayer{Gamertag: p.Gamertag, XUID: p.XUID})
		}
	}
	switch {
	case len(matches) == 0:
		return domainPlayer{}, port.ErrSiteImportNoPlayer
	case len(matches) > 1:
		return domainPlayer{}, port.ErrSiteImportAmbiguous
	case matches[0].XUID == "":
		return domainPlayer{}, port.ErrSiteImportPlayerNoXID
	}
	return matches[0], nil
}

// domainPlayer : le joueur local, réduit à ce que l'import lit.
type domainPlayer struct{ Gamertag, XUID string }
