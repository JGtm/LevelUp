// Package api — registry_citations_relance.go : le câblage de la relance des citations du film
// (service citationsrelance) sur les bases du serveur — base partagée du titre en lecture par
// son provider, base metadata déjà ouverte par le processus, bases joueurs sous leur lease.
package wire

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/citationsrelance"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/citationsfilm"
)

// NewCitationsRelance construit le service de relance des citations du film. Le serveur le
// branche comme notifieur des dérivations et fait tourner sa boucle sur le contexte des tâches
// de fond (cmd/server).
func (r *ServiceRegistry) NewCitationsRelance() *citationsrelance.Service {
	return citationsrelance.New(citationsRecomputer{r: r})
}

// citationsRecomputer implémente port.FilmCitationsRecomputer sur le registre.
type citationsRecomputer struct {
	r *ServiceRegistry
}

var _ port.FilmCitationsRecomputer = citationsRecomputer{}

func (c citationsRecomputer) Participants(ctx context.Context, titleSlug, matchID string) ([]string, error) {
	lire := sharedReadDepot(c.r.sharedProviderForTitle(titleSlug), titleSlug)
	if lire == nil {
		return nil, fmt.Errorf("base partagée du titre %s non câblée", titleSlug)
	}
	var (
		out []string
		err = errors.New("base partagée indisponible")
	)
	lire(ctx, "citations_relance_participants", func(db *sql.DB) {
		out, err = citationsfilm.ParticipantsDuMatch(ctx, db, matchID)
	})
	return out, err
}

func (c citationsRecomputer) FollowedPlayers(titleSlug string) ([]domain.PlayerSummary, error) {
	return c.r.cfg.LoadPlayers(titleSlug)
}

func (c citationsRecomputer) RecomputeForPlayer(
	ctx context.Context, titleSlug string, player domain.PlayerSummary, matchID string,
) (domain.CitationsRelanceIssue, error) {
	lire := sharedReadDepot(c.r.sharedProviderForTitle(titleSlug), titleSlug)
	if lire == nil {
		return "", fmt.Errorf("base partagée du titre %s non câblée", titleSlug)
	}
	metaPath := config.MetadataDBPath(c.r.cfg, titleSlug)
	repoRoot := c.r.cfg.RepoRoot
	return citationsfilm.RelancerPourJoueur(ctx, citationsfilm.RelanceDeps{
		LireShared: lire,
		MetaDB: func() *sql.DB {
			if db, ok := duckdbpkg.LookupCachedDB(metaPath); ok {
				return db.SQLDb()
			}
			return nil
		},
		LireJoueur:   duckdbpkg.OpenReadForQuery,
		OuvrirJoueur: syncpkg.OpenPlayerDB,
		Calculer: func(ctx context.Context, metaDB, sharedDB, playerDB *sql.DB, xuid string, ids []string) error {
			return syncpkg.RecalculerCitations(ctx, syncpkg.RecalculCitations{
				RepoRoot: repoRoot, TitleSlug: titleSlug, XUID: xuid,
				MetaDB: metaDB, SharedDB: sharedDB, PlayerDB: playerDB, MatchIDs: ids,
			})
		},
	}, citationsfilm.RelanceJoueur{
		XUID:         player.XUID,
		PlayerDBPath: config.PlayerDBPath(c.r.cfg, titleSlug, player.Gamertag),
	}, matchID)
}
