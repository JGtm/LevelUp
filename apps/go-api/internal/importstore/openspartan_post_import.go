package importstore

import (
	"context"
	"database/sql"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync"
)

// postImportLabel étiquette les writers pris par le recalcul d'après-import.
const postImportLabel = "openspartan_post_import"

// PostImportStore ouvre la base du joueur pour le recalcul d'après-import OpenSpartan.
type PostImportStore struct {
	provider sharedprovider.Provider
}

var _ port.OpenSpartanPostImportStore = (*PostImportStore)(nil)

// NewPostImportStore lie le recalcul au fournisseur de la base partagée (nil : repli sur le bail
// applicatif et l'ouverture directe, comme sync.AcquireSharedWriterStandalone).
func NewPostImportStore(provider sharedprovider.Provider) *PostImportStore {
	return &PostImportStore{provider: provider}
}

// Open ouvre la base du joueur par le cache de processus de sync.OpenPlayerDB (schéma appliqué, base
// créée si absente).
func (s *PostImportStore) Open(_ context.Context, paths port.OpenSpartanPostImportPaths) (port.OpenSpartanPostImportSession, error) {
	handle, err := sync.OpenPlayerDB(paths.Player)
	if err != nil {
		return nil, err
	}
	return &postImportSession{provider: s.provider, paths: paths, closeFn: handle.Close, player: handle.SQLDb()}, nil
}

// postImportSession : les étapes de recalcul d'un joueur, sur sa base ouverte.
type postImportSession struct {
	provider sharedprovider.Provider
	paths    port.OpenSpartanPostImportPaths
	closeFn  func() error
	player   *sql.DB
}

// Close ferme la base du joueur.
func (s *postImportSession) Close() error { return s.closeFn() }

// acquireShared prend le writer partagé d'une étape ; un échec porte le nom de l'étape suivi de _acquire.
func (s *postImportSession) acquireShared(ctx context.Context, stage string) (*sql.DB, func(), error) {
	db, release, err := sync.AcquireSharedWriterStandalone(ctxkeys.WithDBWriterLabel(ctx, postImportLabel), s.provider, s.paths.Shared)
	if err != nil {
		return nil, nil, &port.OpenSpartanStageError{Stage: stage + "_acquire", Err: err}
	}
	return db, release, nil
}

// PrimeEnrichment amorce la ligne de base des matchs importés par le persister d'enrichissement.
func (s *postImportSession) PrimeEnrichment(ctx context.Context, matchIDs []string) ([]port.OpenSpartanEnrichmentFailure, error) {
	_, failures, err := persist.NewPostSyncEnrichmentPersister(s.player).PrimeBaseline(ctx, matchIDs)
	if err != nil {
		return nil, err
	}
	out := make([]port.OpenSpartanEnrichmentFailure, 0, len(failures))
	for _, f := range failures {
		out = append(out, port.OpenSpartanEnrichmentFailure{MatchID: f.MatchID, Err: f.Err})
	}
	return out, nil
}

// ProjectCSR projette shared.match_csrs (écrit à l'import depuis RankRecap) vers player.match_skill_rank.
func (s *postImportSession) ProjectCSR(ctx context.Context, xuid string) (int, error) {
	shared, release, err := s.acquireShared(ctx, "csr")
	if err != nil {
		return 0, err
	}
	defer release()
	return sync.BackfillCSRFromShared(ctx, shared, s.player, xuid)
}

// RecomputeLUSR rejoue le LUSR v2 du joueur sur tout son historique ; les composantes de performance (dont
// medal_exploit) se lisent dans la metadata du titre des bases.
func (s *postImportSession) RecomputeLUSR(ctx context.Context, xuid string) (int, error) {
	shared, release, err := s.acquireShared(ctx, "lusr")
	if err != nil {
		return 0, err
	}
	defer release()
	return sync.RecomputeLUSRCanonicalForPlayer(ctx, s.player, shared, xuid,
		sync.MedalExploitOption(s.paths.Metadata, xuid))
}

// ComputePerfScores calcule les notes de performance du joueur.
func (s *postImportSession) ComputePerfScores(ctx context.Context, xuid string, force bool) (int, error) {
	shared, release, err := s.acquireShared(ctx, "perf_scores")
	if err != nil {
		return 0, err
	}
	defer release()
	return sync.BatchComputePerformanceScores(ctx, s.player, shared, xuid, force)
}

// BackfillCitations recalcule les citations des matchs donnés. shared_pve est ouvert en lecture seule et
// passé au pipeline pour que les citations pve_stat (Firefight) soient calculées ; absent, l'étape est
// dégradée.
func (s *postImportSession) BackfillCitations(ctx context.Context, xuid string, matchIDs []string) error {
	meta, releaseMeta, err := sync.AcquireMetadataWriterStandalone(ctx, s.paths.Metadata)
	if err != nil {
		return &port.OpenSpartanStageError{Stage: "open_metadata", Err: err}
	}
	defer releaseMeta()
	shared, release, err := s.acquireShared(ctx, "citations")
	if err != nil {
		return err
	}
	defer release()
	pve := sync.OpenPveReadForCitations(ctx, s.paths.PVE)
	defer pve.Close()
	return sync.BackfillMatchCitations(ctx, meta, shared, s.player, pve, xuid, matchIDs)
}
