package duckdb

// media_tx.go — la transaction des écritures médias durables (port.MediaTxOpener) : acquisition du writer de
// la base sociale, ouverture de la transaction, validation avec CHECKPOINT (ADR 0021 Phase 3.2 bis / ADR 0022),
// libération du writer. Les écritures elles-mêmes sont celles de MediaRepo (SetMediaLikeAtomic,
// MarkMediaDeleted, ApplyMediaMerge), exécutées sur la transaction.

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/port"
)

// MediaTxOpener ouvre les transactions des écritures médias sur le writer partagé que lui rend acquire.
type MediaTxOpener struct {
	repo    *MediaRepo
	acquire func() (*dblease.LeasedWriter, error)
}

var _ port.MediaTxOpener = (*MediaTxOpener)(nil)

// NewMediaTxOpener lie les écritures de repo au writer que rend acquire (en production : le writer de
// la base sociale du joueur, sous son bail).
func NewMediaTxOpener(repo *MediaRepo, acquire func() (*dblease.LeasedWriter, error)) *MediaTxOpener {
	return &MediaTxOpener{repo: repo, acquire: acquire}
}

// BeginMediaTx acquiert le writer puis ouvre la transaction ; le writer est rendu si l'ouverture échoue.
func (o *MediaTxOpener) BeginMediaTx(ctx context.Context, op string) (port.MediaTx, error) {
	w, err := o.acquire()
	if err != nil {
		return nil, err
	}
	tx, err := w.BeginTx(ctx, nil)
	if err != nil {
		w.Release()
		return nil, fmt.Errorf("%s begin tx: %w", op, err)
	}
	return &mediaTx{repo: o.repo, w: w, tx: tx}, nil
}

// mediaTx : une transaction ouverte sur le writer partagé.
type mediaTx struct {
	repo *MediaRepo
	w    *dblease.LeasedWriter
	tx   *sql.Tx
}

func (t *mediaTx) SetMediaLike(ctx context.Context, filePath, likerSlug, likerGamertag string, liked bool) (bool, error) {
	return t.repo.SetMediaLikeAtomic(ctx, t.tx, filePath, likerSlug, likerGamertag, liked)
}

func (t *mediaTx) MarkMediaDeleted(ctx context.Context, filePath string) (bool, error) {
	return t.repo.MarkMediaDeleted(ctx, t.tx, filePath)
}

func (t *mediaTx) ApplyMediaMerge(ctx context.Context, missing, target domain.MediaFileRef) error {
	return t.repo.ApplyMediaMerge(ctx, t.tx, missing, target)
}

// Commit valide puis force le CHECKPOINT : ne jamais revenir à un tx.Commit() nu (verrouillé par
// TestMediaService_SetMediaLike_Atomic_SurvivesWALLoss et TestMediaService_DeleteMedia_CommitsWithCheckpoint).
func (t *mediaTx) Commit(ctx context.Context) error {
	defer t.w.Release()
	return t.w.CommitWithCheckpoint(ctx, t.tx)
}

func (t *mediaTx) Rollback() error {
	defer t.w.Release()
	return t.tx.Rollback()
}
