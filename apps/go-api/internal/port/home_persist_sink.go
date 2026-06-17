package port

import "context"

// HomePersistSink est le sink de persistance fire-and-forget consommé par
// HomeService lors du fetch BattlePass/Challenges (écriture best-effort des
// snapshots, hors payload de lecture). Extrait en port (Phase 2 canonical) pour
// que le service ne dépende plus du type concret internal/platform/duckdb.PersistSink.
//
// Implémenté par internal/platform/duckdb.PersistSink.
type HomePersistSink interface {
	// PersistBattlePassSync persiste le snapshot BattlePass d'un reward track.
	PersistBattlePassSync(ctx context.Context, trackPath string, rawBody []byte) error
	// PersistChallengesSync persiste le snapshot des défis.
	PersistChallengesSync(ctx context.Context, rawBody []byte) error
}
