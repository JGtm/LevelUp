package snapshot

// snapshot_cutter.go — pont entre le cycle V2 (Phase 6bis) et le producteur ops.
//
// SnapshotCutter satisfait STRUCTURELLEMENT sync/v2.SnapshotProducer sans importer v2
// (qui importe déjà sync → un import inverse créerait un cycle). Le câblage cmd/server
// l'injecte via orchestrator.WithSnapshotProducer.
//
// Il ouvre shared + chaque player DB en LECTURE via OpenReadForQuery (réutilise le
// handle process-wide cached RW/RO — JAMAIS `?access_mode=read_only` direct ni ATTACH,
// incident 2026-06-01), délègue à ops.ProduceSnapshot, et enregistre les métriques.

import (
	"context"
	"database/sql"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	duckdb "levelup/go-api/internal/platform/duckdb"
)

// snapshotRetentionDefault : nombre de versions complètes conservées par défaut
// (rollback). Override possible au câblage (env côté cmd/server).
const snapshotRetentionDefault = 5

// SnapshotCutter produit les snapshots immuables d'un titre en fin de cycle.
type SnapshotCutter struct {
	paths         *title.PathResolver
	retentionKeep int
}

// NewSnapshotCutter construit un cutter. retentionKeep <= 0 → snapshotRetentionDefault.
func NewSnapshotCutter(paths *title.PathResolver, retentionKeep int) *SnapshotCutter {
	if retentionKeep <= 0 {
		retentionKeep = snapshotRetentionDefault
	}
	return &SnapshotCutter{paths: paths, retentionKeep: retentionKeep}
}

// CutSnapshot produit (au besoin) une nouvelle version de snapshot pour titleSlug à
// partir des matchs ready des joueurs donnés. Best-effort, idempotent (change-gated).
func (c *SnapshotCutter) CutSnapshot(ctx context.Context, titleSlug string, gamertags []string) error {
	if c == nil || c.paths == nil {
		return nil
	}
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	start := time.Now()
	res, err := ops.ProduceSnapshot(ctx, ops.SnapshotOptions{
		TitleSlug: titleSlug,
		Paths:     c.paths,
		Shared: ops.SharedReadOpenerFunc(func(context.Context) (*sql.DB, func(), error) {
			return duckdb.OpenReadForQuery(c.paths.SharedDBPath(titleSlug))
		}),
		PlayerOpener: ops.PlayerReadOpenerFunc(func(_ context.Context, gt string) (*sql.DB, func(), error) {
			return duckdb.OpenReadForQuery(c.paths.PlayerDBPath(titleSlug, gt))
		}),
		Players:       gamertags,
		RetentionKeep: c.retentionKeep,
	})
	recordSnapshotCut(titleSlug, res, err, time.Since(start))
	// Gauges de backlog (pending/partial/oldest-age) rafraîchies à chaque cycle, même
	// si le cut a été un no-op : l'opérateur voit l'évolution du backlog en continu.
	c.reportPending(ctx, titleSlug, gamertags)
	return err
}
