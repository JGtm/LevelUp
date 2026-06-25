// Package v2 — snapshot.go : interface du producteur de snapshot immuable (Phase 6bis
// du cycle, plan PLAN_DURABILITE_SNAPSHOT_IMMUABLE).
//
// Le cycle ne dépend que de cette interface (consumer-side) ; l'implémentation concrète
// (internal/sync.SnapshotCutter → internal/ops.ProduceSnapshot) est injectée au câblage
// par WithSnapshotProducer. Cette indirection évite un cycle d'import (v2 importe déjà
// sync) : la conformité est structurelle, l'implémentation n'importe pas v2.
package v2

import "context"

// SnapshotProducer fige une version immuable et cohérente du dataset d'un titre à la
// fin d'un cycle de sync (après libération du write-lease shared). Optionnel et
// best-effort : un orchestrator sans producteur n'émet aucun snapshot, et un échec de
// cut n'invalide jamais le cycle.
type SnapshotProducer interface {
	// CutSnapshot produit au besoin une nouvelle version pour titleSlug à partir des
	// matchs ready des joueurs donnés. No-op interne si rien n'a changé depuis le
	// dernier cut (idempotent, change-gated).
	CutSnapshot(ctx context.Context, titleSlug string, gamertags []string) error
}
