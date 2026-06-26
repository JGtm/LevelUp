// Package api — registry_relations_cross_game.go : construction de la dépendance
// cross-jeu (Phase 3b) injectée dans RelationsService. BEST-EFFORT / LECTURE
// SEULE / TITLE-AGNOSTIC : énumère les AUTRES titres via le TitleRegistry,
// résout leur shared via PathResolver, lit en RO (OpenReadForQuery, respecte le
// cache RW/RO in-process) et compte les co-occurrences par xuid (global, ADR
// 0008). Toute erreur d'accès → skip + log, jamais propagée à /relations.
package api

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/relations"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

// crossGameCooccurrence implémente port.CrossGameCooccurrence en lisant le
// shared de chaque autre titre actif. Mono-process safe : OpenReadForQuery
// réutilise un handle caché RW/RO plutôt que de forcer un OpenReadOnly
// concurrent (ADR 0016).
type crossGameCooccurrence struct {
	repoRoot    string
	timezone    string
	currentSlug string
	myXUID      string
	registry    *title.Registry
	resolver    *title.PathResolver
}

// buildCrossGameCooccurrence construit la dépendance cross-jeu pour le joueur
// courant. Retourne nil (badge inerte) si la config est indisponible ou si le
// joueur n'a pas d'xuid — dégradation gracieuse.
func (r *ServiceRegistry) buildCrossGameCooccurrence(pdb *duckdb.PlayerDB) port.CrossGameCooccurrence {
	if r.cfg == nil || pdb == nil || pdb.XUID == "" {
		return nil
	}
	reg := title.DefaultRegistry()
	return &crossGameCooccurrence{
		repoRoot:    r.cfg.RepoRoot,
		timezone:    r.cfg.UserTimezone,
		currentSlug: pdb.TitleSlug,
		myXUID:      pdb.XUID,
		registry:    reg,
		resolver:    title.NewPathResolver(r.cfg.RepoRoot, reg),
	}
}

// CooccurrencesByXUID parcourt les autres titres actifs et agrège les
// co-occurrences >= seuil. Pour chaque oppXUID, le hit retenu est le titre où la
// co-occurrence est la plus forte (choix simple : « le plus pertinent »).
func (c *crossGameCooccurrence) CooccurrencesByXUID(ctx context.Context, oppXUIDs []string) map[string]port.CrossGameHit {
	out := make(map[string]port.CrossGameHit)
	if len(oppXUIDs) == 0 {
		return out
	}
	for _, desc := range c.registry.Active() {
		if desc == nil || desc.Slug == c.currentSlug || desc.IsInternal {
			continue // titre courant + titres internes (fixtures) exclus — un badge
			// user-facing ne doit jamais nommer un titre de test (ex. synthetic_title_b)
		}
		counts := c.countForTitle(ctx, desc.Slug, oppXUIDs)
		for xuid, n := range counts {
			if n < relations.CrossGameMinMatchesTogether {
				continue
			}
			if prev, ok := out[xuid]; ok && prev.MatchesTogether >= n {
				continue // garde le titre le plus pertinent (co-occurrence max)
			}
			out[xuid] = port.CrossGameHit{TitleDisplayName: desc.Name, MatchesTogether: n}
		}
	}
	return out
}

// countForTitle lit le shared d'un autre titre et compte les co-occurrences.
// BEST-EFFORT : toute erreur (DB absente, lock, requête) → map vide + log debug.
func (c *crossGameCooccurrence) countForTitle(ctx context.Context, slug string, oppXUIDs []string) map[string]int {
	path := c.resolver.SharedDBPath(slug)
	sqlDB, release, err := duckdb.OpenReadForQuery(path, c.timezone)
	if err != nil {
		slog.DebugContext(ctx, "cross-game: shared indisponible (badge omis pour ce titre)",
			"title", slug, "path", path, "err", err)
		return nil
	}
	defer release()

	counts, err := duckdb.CountCrossTitleCooccurrences(
		ctx,
		sqlDB,
		c.myXUID,
		oppXUIDs,
		relations.CrossGameMinMatchesTogether,
	)
	if err != nil {
		slog.DebugContext(ctx, "cross-game: requête co-occurrence échouée (badge omis pour ce titre)",
			"title", slug, "err", err)
		return nil
	}
	return counts
}
