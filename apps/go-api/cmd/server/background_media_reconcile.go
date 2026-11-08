// background_media_reconcile.go — la passe unique, au démarrage, qui aligne l'index des médias sur
// le dossier des captures : ce que l'utilisateur a renommé, déplacé ou supprimé dans l'explorateur
// pendant que l'application était éteinte.

package main

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/service"
)

// mediaScan : la passe d'indexation d'un joueur sous le contexte donné (le titre y est posé).
type mediaScan func(ctx context.Context, gamertag string)

// mediaReconcileTask lance UNE passe d'indexation des médias pour chaque joueur de chaque titre qui
// porte des médias. Elle suit la même route que la passe d'après synchronisation
// ([service.NewLiveMediaScanHook]) : réglages relus à l'appel, politique de marquage décidée par la
// fiabilité du dossier. Coupée en démo : elle écrit les bases joueurs.
func mediaReconcileTask(ctx context.Context, cfg *config.AppConfig, store *settings.Store) backgroundTask {
	scan := func(c context.Context, gamertag string) {
		service.NewLiveMediaScanHook(cfg, store, gamertag)(c)
	}
	return backgroundTask{name: taskMediaReconcile, tracked: true, run: func() {
		runMediaReconcile(ctx, mediaTitles(), cfg.LoadPlayers, scan)
	}}
}

// mediaTitles rend les titres du registre qui portent la capacité média.
func mediaTitles() []string {
	var slugs []string
	for _, t := range title.DefaultRegistry().All() {
		if t.HasCapability(title.CapMedia) {
			slugs = append(slugs, t.Slug)
		}
	}
	return slugs
}

// runMediaReconcile passe en revue chaque joueur de chaque titre, un à la fois, et s'arrête entre
// deux joueurs dès que ctx est annulé. Un titre dont la liste de joueurs est illisible est journalisé
// et sauté ; les profils réservés aux jetons (AuthOnly) n'ont pas de base joueur.
func runMediaReconcile(
	ctx context.Context,
	slugs []string,
	loadPlayers func(titleFilter ...string) ([]domain.PlayerSummary, error),
	scan mediaScan,
) {
	for _, slug := range slugs {
		players, err := loadPlayers(slug)
		if err != nil {
			slog.WarnContext(ctx, "médias: liste des joueurs illisible, titre sauté", "titleSlug", slug, "err", err)
			continue
		}
		titleCtx := ctxkeys.WithTitleSlug(ctx, slug)
		for _, p := range players {
			if ctx.Err() != nil {
				return
			}
			if p.AuthOnly {
				continue
			}
			scan(titleCtx, p.Gamertag)
		}
	}
}
