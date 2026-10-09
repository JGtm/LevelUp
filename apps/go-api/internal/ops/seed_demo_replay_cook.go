// Package ops — seed_demo_replay_cook.go : la RECUISSON d'un rejeu figé de la démo.
//
// Même chemin que l'étape 1.58 du post-sync et que l'action admin : le parent lit les faits du
// match et ses cartes candidates dans la base SOURCE, relâche ses handles, puis délègue le
// décodage à UN enfant borné (`replaychild`, plafond mémoire `internal/filmproc`) qui rend les
// octets ; le parent les range lui-même (`replaybuild.StoreArtifactAt`, validations et garde
// anti-régression compris), dans la disposition démo. Le film lu est celui EMBARQUÉ par la
// démo, jamais un téléchargement.
//
// LE VERROU DE DÉCODAGE EST CELUI DE LA MACHINE : il vit dans le cache du dépôt dont viennent
// les films (`SeedDemoOptions.CacheRepoRoot`), et la recuisson l'ATTEND (borne
// demoReplayLockWait) au lieu de renoncer — un seed n'a pas de cycle suivant.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/replaychild"
)

// demoReplayLockWait borne l'attente du verrou de décodage par film : au-delà, le rejeu reste
// à son schéma précédent pour cette régénération (bilan « périmé ») plutôt que de bloquer le
// déploiement de la démo derrière une passe de cuisson.
const demoReplayLockWait = 10 * time.Minute

// demoReplayTool nomme le détenteur du verrou de décodage pendant la recuisson.
const demoReplayTool = "seed-demo"

// sharedDBReader adapte un handle DuckDB en lecture au port SharedReader des dépôts.
type sharedDBReader struct{ db *duckdb.DB }

func (r sharedDBReader) Get(context.Context) (*sql.DB, func(), error) {
	return r.db.SQLDb(), func() {}, nil
}

// cookDemoReplay recuit le rejeu d'un match depuis le film embarqué par la démo.
func cookDemoReplay(ctx context.Context, opts SeedDemoOptions, layout titlePkg.DemoLayout, matchID string) error {
	facts, mapNames, err := demoReplayInputs(ctx, opts, matchID)
	if err != nil {
		return err
	}
	slug := opts.TitleSlug
	req := replayChildRequest(opts, layout, matchID, mapNames, facts)
	lockRoot := titlePkg.NewPathResolver(opts.cacheRepoRoot()).CacheRootDir()
	res, err := replaychild.SpawnWaiting(ctx, req, lockRoot, demoReplayTool, demoReplayLockWait)
	if err != nil {
		return err
	}
	stored, err := replaybuild.StoreArtifactAt(ctx, layout.ReplayArtifactPath(slug, matchID), slug, matchID, res.Blob)
	if err != nil {
		return fmt.Errorf("rangement de l'artefact démo: %w", err)
	}
	if !replaybuild.ArtifactUpToDate(stored.Path) {
		return fmt.Errorf("artefact démo de %s non mis à jour (garde anti-régression, schéma %d)",
			matchID, stored.SchemaVersion)
	}
	return nil
}

// replayChildRequest : la requête de l'enfant — le constructeur lit les données de référence du
// dépôt source, le film embarqué par la démo.
func replayChildRequest(opts SeedDemoOptions, layout titlePkg.DemoLayout, matchID string,
	mapNames []string, facts port.MatchFacts) replaychild.Request {
	short := titlePkg.FilmShortMatchID(matchID)
	return replaychild.Request{
		MatchID: matchID, TitleSlug: opts.TitleSlug, RepoRoot: opts.RepoRoot, MapNames: mapNames,
		FilmDir: filmcache.ChunkDir(layout.ReplayFilmsCacheRoot(opts.TitleSlug), short), Facts: facts,
	}
}

// demoReplayInputs lit, dans les bases SOURCE et en lecture seule, les faits du match et ses
// cartes candidates ; les handles sont fermés avant le décodage.
func demoReplayInputs(ctx context.Context, opts SeedDemoOptions, matchID string) (port.MatchFacts, []string, error) {
	shared, err := duckdb.OpenReadOnly(opts.SourceSharedDB)
	if err != nil {
		return port.MatchFacts{}, nil, fmt.Errorf("base partagée source: %w", err)
	}
	defer func() { _ = shared.Close() }()
	meta, err := duckdb.OpenReadOnly(opts.SourceMetaDB)
	if err != nil {
		return port.MatchFacts{}, nil, fmt.Errorf("métadonnées source: %w", err)
	}
	defer func() { _ = meta.Close() }()

	keys, err := duckdb.NewReplayMapRepo(sharedDBReader{db: shared}, meta).MapKeysForMatch(ctx, matchID)
	if err != nil {
		return port.MatchFacts{}, nil, fmt.Errorf("carte du match %s: %w", matchID, err)
	}
	facts, err := duckdb.NewReplayFactsRepo(shared.SQLDb()).FactsForMatch(ctx, matchID)
	if err != nil {
		return port.MatchFacts{}, nil, fmt.Errorf("faits du match %s: %w", matchID, err)
	}
	return facts, keys.Names, nil
}
