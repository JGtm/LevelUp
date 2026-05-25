// Package v2 — persist_v1bridge.go : adapter CycleBatchPersister qui
// délègue le parsing à V1 (BuildBatchFromRawForV2) et la persistance à
// la BatchQueue partagée.
//
// Stratégie de duplication ciblée : V2 a fetché les raw JSON en Phase 3,
// V1 sait les transformer en *persist.MatchBatch (parsing pur).
// On submit ensuite chaque batch à la queue async existante. Le worker
// monothread persiste en série mais SANS concurrence avec d'autres
// syncs (V2 est seul à écrire pendant ce cycle).
package v2

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/persist"
	syncpkg "levelup/go-api/internal/sync"
)

// cycleBatchPersisterV1Bridge implémente CycleBatchPersister via les
// wrappers V1 (engine_v2bridge.go) + persist.BatchQueue (partagé).
type cycleBatchPersisterV1Bridge struct {
	titleSlug       string
	playerByXuid    map[string]PlayerProfile // pour résoudre fetcher slug → xuid
	queue           *persist.BatchQueue
	drainCtxTimeout time.Duration // typique 60s, configurable pour tests
}

// NewCycleBatchPersister construit un CycleBatchPersister V2.
//
// Paramètres :
//   - titleSlug : slug du titre (halo_infinite par défaut). Passé à BuildBatchFromRawForV2.
//   - playerBySlug : index PlayerSlug → PlayerProfile pour résoudre les
//     gamertag/xuid au moment de construire le batch (le fetcher est dans
//     SharedMatchData.Fetcher).
//   - queue : BatchQueue partagée (process-wide). Si nil, le persister
//     échoue avec une erreur explicite (impossible de persister sans queue).
//   - drainTimeout : timeout du drain à la fin du cycle. 0 → 60s default.
func NewCycleBatchPersister(
	titleSlug string,
	playerBySlug map[string]PlayerProfile,
	queue *persist.BatchQueue,
	drainTimeout time.Duration,
) CycleBatchPersister {
	if drainTimeout <= 0 {
		drainTimeout = 60 * time.Second
	}
	// Indexer aussi par xuid pour fast lookup côté fetcher slug.
	byXuid := make(map[string]PlayerProfile, len(playerBySlug))
	for _, p := range playerBySlug {
		byXuid[p.XUID] = p
	}
	return &cycleBatchPersisterV1Bridge{
		titleSlug:       titleSlug,
		playerByXuid:    byXuid,
		queue:           queue,
		drainCtxTimeout: drainTimeout,
	}
}

// PersistCycle parse chaque SharedMatchData via V1.BuildBatchFromRawForV2,
// soumet chaque batch à la queue, puis attend le drain.
//
// Comportement transactionnel : un batch qui échoue à Submit est loggué
// mais ne fait pas échouer le cycle (cohérent avec V1). Le drain final
// retourne l'erreur si le worker accumule trop d'échecs (circuit-breaker
// déjà en place dans BatchQueue, cf. ADR 0019).
func (p *cycleBatchPersisterV1Bridge) PersistCycle(ctx context.Context, batch CycleBatch) error {
	if p.queue == nil {
		return fmt.Errorf("PersistCycle: BatchQueue nil (le pipeline V2 nécessite une queue partagée)")
	}

	submitted := 0
	parseErrors := 0
	for mID, sd := range batch.Matches {
		// Recovera le fetcher PlayerProfile pour avoir son gamertag.
		// SharedMatchData.Fetcher est le PlayerSlug, mais nous n'avons
		// pas de map slug→profile ici ; on bridge via playerByXuid en
		// cherchant par xuid stocké dans Skill (si présent) ou via la
		// résolution faite côté orchestrator. Pour simplifier le bridge
		// initial, on fallback : si Fetcher slug est dans playerByXuid
		// (en indexant par slug = clé secondaire), on utilise ; sinon
		// on prend le 1er profile disponible.
		fetcherProfile, ok := p.resolveFetcherProfile(sd.Fetcher)
		if !ok {
			slog.WarnContext(ctx, "PersistCycle: fetcher inconnu — skip match",
				"match_id", mID, "fetcher_slug", sd.Fetcher)
			continue
		}

		// Convert Skill map[string]any → map[string]*syncpkg.MatchSkillData
		skillTyped := make(map[string]*syncpkg.MatchSkillData, len(sd.Skill))
		for xuid, val := range sd.Skill {
			if sk, ok := val.(*syncpkg.MatchSkillData); ok {
				skillTyped[xuid] = sk
			}
		}

		// T2 (parité V1) : V2 Phase 3 fetche désormais les highlights via
		// SharedMatchFetcher. On les propage à BuildBatchFromRawForV2 pour
		// qu'elles soient insérées en même temps que le reste.
		matchBatch, err := syncpkg.BuildBatchFromRawForV2(
			ctx,
			p.titleSlug,
			fetcherProfile.Gamertag,
			fetcherProfile.XUID,
			mID,
			sd.Stats,
			skillTyped,
			sd.HighlightChunk,
			sd.FilmMajorVer,
			sd.HasHighlights,
		)
		if err != nil {
			parseErrors++
			slog.WarnContext(ctx, "PersistCycle: parse batch failed",
				"match_id", mID, "fetcher", sd.Fetcher, "err", err)
			continue
		}
		if matchBatch == nil {
			continue
		}

		if submitErr := p.queue.Submit(matchBatch); submitErr != nil {
			slog.WarnContext(ctx, "PersistCycle: queue.Submit failed",
				"match_id", mID, "err", submitErr)
			continue
		}
		submitted++
	}

	// Drain : attendre que tous les batches soient ACKés (WAL vide).
	drainCtx, cancel := context.WithTimeout(ctx, p.drainCtxTimeout)
	defer cancel()
	if drainErr := p.queue.Drain(drainCtx); drainErr != nil {
		return fmt.Errorf("PersistCycle: drain (submitted=%d parse_errors=%d): %w",
			submitted, parseErrors, drainErr)
	}

	slog.InfoContext(ctx, "PersistCycle terminé",
		"cycle_id", batch.CycleID,
		"matches_in_batch", len(batch.Matches),
		"submitted", submitted,
		"parse_errors", parseErrors,
	)
	return nil
}

// resolveFetcherProfile lookup secondaire par slug. La construction du
// persister via NewCycleBatchPersister indexe par xuid pour l'usage
// principal ; ici on accepte aussi un slug en re-scannant la map (linéaire,
// acceptable pour <100 joueurs).
func (p *cycleBatchPersisterV1Bridge) resolveFetcherProfile(slug string) (PlayerProfile, bool) {
	for _, prof := range p.playerByXuid {
		if prof.PlayerSlug == slug {
			return prof, true
		}
	}
	return PlayerProfile{}, false
}
