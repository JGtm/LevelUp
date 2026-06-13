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
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/assetnames"
	"levelup/go-api/internal/persist"
	syncpkg "levelup/go-api/internal/sync"
)

// cycleBatchPersisterV1Bridge implémente CycleBatchPersister via les
// wrappers V1 (engine_v2bridge.go) + persist.BatchQueue (partagé).
//
// Note D7 prep : la map PlayerBySlug est lue depuis CycleBatch.PlayerBySlug
// à chaque PersistCycle (pas stockée dans le struct). Permet au persister
// d'être instancié une seule fois au boot pendant que la liste de joueurs
// est dynamique cycle-par-cycle.
type cycleBatchPersisterV1Bridge struct {
	titleSlug       string
	queue           *persist.BatchQueue
	drainCtxTimeout time.Duration // typique 60s, configurable pour tests

	// metaDB : handle metadata RW PARTAGÉ (celui de main.go via OpenReadWriteShared).
	// Sert à la résolution autonome des noms d'assets (peuplement asset_translations)
	// ET à l'enrich registry au primary write V2. nil → feature off (tests).
	metaDB *sql.DB
	// assetFetcher : source des noms d'assets (token-free, API publique GameCMS).
	// nil → résolution désactivée (parité legacy / gating caller).
	assetFetcher assetnames.Fetcher
}

// NewCycleBatchPersister construit un CycleBatchPersister V2.
//
// Paramètres :
//   - titleSlug : slug du titre (halo_infinite par défaut). Passé à BuildBatchFromRawForV2.
//   - queue : BatchQueue partagée (process-wide). Si nil, le persister
//     échoue avec une erreur explicite (impossible de persister sans queue).
//   - drainTimeout : timeout du drain à la fin du cycle. 0 → 60s default.
//
// playerBySlug est passé via CycleBatch.PlayerBySlug à chaque PersistCycle
// (cf. RunPersist → orchestrator cycle.go).
// metaDB : handle metadata RW partagé (résolution + enrich noms d'assets) ; nil → off.
// assetFetcher : source des noms d'assets (token-free) ; nil → résolution désactivée.
func NewCycleBatchPersister(
	titleSlug string,
	queue *persist.BatchQueue,
	drainTimeout time.Duration,
	metaDB *sql.DB,
	assetFetcher assetnames.Fetcher,
) CycleBatchPersister {
	if drainTimeout <= 0 {
		drainTimeout = 60 * time.Second
	}
	return &cycleBatchPersisterV1Bridge{
		titleSlug:       titleSlug,
		queue:           queue,
		drainCtxTimeout: drainTimeout,
		metaDB:          metaDB,
		assetFetcher:    assetFetcher,
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

	// Résolution autonome des noms d'assets (primary write) : peuple
	// metadata.asset_translations pour les assets neufs du cycle AVANT le
	// build/enrich des batches, pour que BuildBatchFromRawForV2WithMeta écrive un
	// vrai nom en registry dès le 1er passage. Best-effort, gated (nil → no-op).
	if p.assetFetcher != nil && p.metaDB != nil {
		statsList := make([]map[string]any, 0, len(batch.Matches))
		for _, sd := range batch.Matches {
			if sd.Stats != nil {
				statsList = append(statsList, sd.Stats)
			}
		}
		syncpkg.ResolveAssetsFromStats(ctx, p.assetFetcher, p.metaDB, p.titleSlug, statsList)
	}

	submitted := 0
	parseErrors := 0
	for mID, sd := range batch.Matches {
		// Lookup direct via batch.PlayerBySlug (renseigné par l'orchestrator
		// à chaque cycle, cf. cycle.go Phase 5).
		fetcherProfile, ok := batch.PlayerBySlug[sd.Fetcher]
		if !ok {
			slog.WarnContext(ctx, "PersistCycle: fetcher inconnu — skip match",
				"match_id", mID, "fetcher_slug", sd.Fetcher,
				"player_by_slug_size", len(batch.PlayerBySlug))
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
		matchBatch, err := syncpkg.BuildBatchFromRawForV2WithMeta(
			ctx,
			p.metaDB,
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
