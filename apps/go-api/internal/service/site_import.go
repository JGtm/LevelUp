package service

// site_import.go — l'import des données d'un joueur depuis une instance hébergée (lot L22.b, ADR 0045) côté
// application de bureau : résout le joueur suivi localement, branche les chemins d'écriture EXISTANTS de
// l'application (fournisseur de la base partagée, bail et ouverture de la base joueur) sur internal/transfer,
// et tient l'état du téléchargement et de l'import pour la page des réglages.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/transfer"
)

// Erreurs de la résolution du joueur local ; leur identité fixe le code d'API rendu.
var (
	ErrSiteImportUnbound     = errors.New("service: site import not bound")
	ErrSiteImportNoPlayer    = errors.New("service: no locally tracked player")
	ErrSiteImportAmbiguous   = errors.New("service: several tracked players, name the gamertag")
	ErrSiteImportPlayerNoXID = errors.New("service: the tracked player has no resolved Xbox id")
)

// SiteImportRequest : la demande d'import. Gamertag n'est nécessaire que si plusieurs joueurs sont suivis.
type SiteImportRequest struct {
	URL      string
	Token    string
	Gamertag string
}

// SiteImport relie l'import depuis le site aux écritures de l'application.
type SiteImport struct {
	mu     sync.Mutex
	cfg    *config.AppConfig
	ctx    context.Context
	wg     *sync.WaitGroup
	runner *transfer.Runner
}

var defaultSiteImport = &SiteImport{runner: transfer.NewRunner()}

// DefaultSiteImport rend l'import du processus : le même pour la route de démarrage et la route d'état.
func DefaultSiteImport() *SiteImport { return defaultSiteImport }

// Bind relie l'import à la configuration et au contexte du serveur : ctx annule un import en cours à
// l'arrêt, wg le fait attendre avant la fermeture des bases.
func (s *SiteImport) Bind(ctx context.Context, wg *sync.WaitGroup, cfg *config.AppConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx, s.wg, s.cfg = ctx, wg, cfg
}

// Status rend l'état de l'import courant ou du dernier.
func (s *SiteImport) Status() transfer.Status { return s.runner.Status() }

// Start lance l'import en arrière-plan. [ErrSiteImportUnbound] sans liaison, [transfer.ErrImportRunning] si
// un import est en cours.
func (s *SiteImport) Start(req SiteImportRequest) error {
	s.mu.Lock()
	cfg, ctx, wg := s.cfg, s.ctx, s.wg
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
			AcquireShared: acquireShared(cfg, slug),
			AcquirePlayer: acquirePlayer(cfg, slug, player.Gamertag),
		},
		WorkDir: cfg.PathResolver().CacheRootDir(),
		// Les lectures du joueur déjà en cache (liste des matchs, filtres) ignorent les lignes importées : même
		// invalidation qu'en fin de synchronisation.
		AfterImport: func(ctx context.Context) { duckdbpkg.InvalidatePlayerReadCaches(ctx, player.XUID, slug) },
	}
	return s.runner.Start(ctx, wg, in)
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
		return domainPlayer{}, ErrSiteImportNoPlayer
	case len(matches) > 1:
		return domainPlayer{}, ErrSiteImportAmbiguous
	case matches[0].XUID == "":
		return domainPlayer{}, ErrSiteImportPlayerNoXID
	}
	return matches[0], nil
}

// domainPlayer : le joueur local, réduit à ce que l'import lit.
type domainPlayer struct{ Gamertag, XUID string }

// acquireShared : l'écriture de la base partagée du titre par le fournisseur de l'application (coordonné avec
// la synchronisation et les lecteurs, bascule B-swap comprise).
func acquireShared(cfg *config.AppConfig, slug string) transfer.Acquire {
	return func(ctx context.Context) (*sql.DB, func(), error) {
		return syncpkg.AcquireSharedWriterStandalone(ctx, cfg.SharedProvider, config.SharedDBPath(cfg, slug))
	}
}

// acquirePlayer : l'écriture de la base joueur sous son bail (un seul écrivain), par l'ouverture du cache du
// processus qui applique aussi le schéma.
func acquirePlayer(cfg *config.AppConfig, slug, gamertag string) transfer.Acquire {
	path := cfg.PathResolver().PlayerDBPath(slug, gamertag)
	return func(ctx context.Context) (*sql.DB, func(), error) {
		lease, err := dblease.AcquireWriterCtx(ctx, nil, path, dblease.KindPlayer)
		if err != nil {
			return nil, nil, err
		}
		handle, err := syncpkg.OpenPlayerDB(path)
		if err != nil {
			lease.Release()
			return nil, nil, err
		}
		return handle.SQLDb(), func() { _ = handle.Close(); lease.Release() }, nil
	}
}
