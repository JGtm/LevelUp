// Package api — registry_replay_regen.go : les dépendances de la file de régénération des rejeux
// (internal/replayregen, application de bureau) tirées du registre de services.
//
// La file ne connaît aucune recette de cuisson : elle rejoue celle de l'action admin « construire le rejeu »
// (`RunReplayBuild` : enfant borné sous le verrou solo, rangement par `StoreArtifact`), puis applique les
// dérivations de l'artefact rangé par le point d'entrée du dépôt d'un ouvrier (`deriverArtefactRange`). Un seul
// film est décodé à la fois : l'action admin est single-flight, et la file n'en lance qu'une.
package wire

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmrun"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/gameproc"
	"levelup/go-api/internal/replayregen"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
)

// ReplayRegenDeps assemble les dépendances de la file : rien d'autre que ce que le registre sait déjà (chemins,
// titres, reconstruction d'un match). pending est le drapeau persisté de reprise.
func (r *ServiceRegistry) ReplayRegenDeps(pending replayregen.PendingStore) replayregen.Deps {
	paths := r.cfg.PathResolver()
	return replayregen.Deps{
		Paths:  paths,
		Titles: func() []string { return titlesWithReplay(r.cfg.Roots().Resources, title.DefaultRegistry()) },
		FilmPresent: func(titleSlug, matchID string) bool {
			_, found, err := filmcache.Open(paths.FilmArchiveDir(), title.FilmShortMatchID(matchID))
			return found && err == nil
		},
		RefusedByRule: r.unmanagedModeMatches,
		Rebuild:       r.regenerateReplay,
		Pending:       pending,
		HeavyAllowed:  gameproc.Default().HeavyWorkAllowed,
	}
}

// unmanagedModeMatches rend les matchs (forme courte) dont le mode n'est pas géré par le rejeu : le MÊME prédicat que
// le refus de la cuisson (sqlfrag.SQLModeNonGere, lu au registre par duckdb.ReplayRegistryRepo ; replayMatchIdentity en
// tire replaybuild.ErrModeNonGere). Le décompte de la file s'en sert pour écarter d'emblée ces rejeux (Firefight d'avant la
// décision du 2026-10-15) au lieu de les compter « en retard » jusqu'à leur premier refus.
func (r *ServiceRegistry) unmanagedModeMatches(ctx context.Context, titleSlug string) (map[string]struct{}, error) {
	sharedSQL, release, err := r.titleBases().ReadShared(ctx, titleSlug)
	if err != nil {
		return nil, err
	}
	defer release()
	ids, err := duckdb.NewReplayRegistryRepo(sharedSQL, nil).UnmanagedModeMatchIDs(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		out[title.FilmShortMatchID(id)] = struct{}{}
	}
	return out, nil
}

// titlesWithReplay : les titres actifs qui déclarent la capacité `film.replay_artifact`. Une configuration de
// titre illisible se journalise et écarte le titre : la file ne devine pas ce qu'il produit.
func titlesWithReplay(repoRoot string, reg *title.Registry) []string {
	var out []string
	for _, desc := range reg.Active() {
		if desc == nil {
			continue
		}
		caps, err := games.LoadCapabilityMap(repoRoot, desc.Slug)
		if err != nil {
			slog.Warn("replay_regen: capacités du titre illisibles, titre écarté", "titleSlug", desc.Slug, "err", err)
			continue
		}
		if caps.Has(games.CapFilmReplayArtifact) {
			out = append(out, desc.Slug)
		}
	}
	return out
}

// regenerateReplay reconstruit le rejeu d'UN match puis rejoue ses dérivations. Un décodage déjà en cours sur la
// machine (action admin, cycle de synchronisation) se rend en `replayregen.ErrBusy` : la file attend et recommence.
func (r *ServiceRegistry) regenerateReplay(ctx context.Context, titleSlug, matchID string) error {
	res, err := r.RunReplayBuild(ctx, titleSlug, matchID)
	if err != nil {
		return busyOr(err)
	}
	path, _ := res["path"].(string)
	fullID, _ := res["match_id"].(string)
	if path == "" || fullID == "" {
		return fmt.Errorf("régénération de %s : le rangement n'a rendu ni chemin ni identité", matchID)
	}
	r.deriverArtefactRange(ctx, titleSlug, fullID, path)
	return nil
}

// busyOr rend err, enveloppée en `replayregen.ErrBusy` quand elle dit qu'un autre décodage tient la machine.
func busyOr(err error) error {
	if errors.Is(err, ErrActionBusy) || errors.Is(err, filmrun.ErrDecodeBusy) {
		return fmt.Errorf("%w: %w", replayregen.ErrBusy, err)
	}
	return err
}
