// Package worldrepo — le classement mondial (base partagée) derrière port.WorldLeaderboardStore, adossé au
// fournisseur de la base partagée (lecteur en vol, bascule B-swap). Les requêtes sont celles de
// platform/duckdb (WorldSeasonPlayers, WorldCSRSnapshotAge, ...) ; le paquet les compose avec le fournisseur
// et ops.RefreshSeasonCatalog, que platform/duckdb ne peut pas importer.
package worldrepo

import (
	"context"
	"database/sql"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/port"
)

// Repo lit et écrit le classement mondial par le fournisseur de la base partagée.
type Repo struct {
	provider sharedprovider.Provider
}

var _ port.WorldLeaderboardStore = (*Repo)(nil)

// New lie le dépôt au fournisseur de la base partagée.
func New(provider sharedprovider.Provider) *Repo {
	return &Repo{provider: provider}
}

// ForProvider rend le magasin d'un fournisseur, ou nil sans fournisseur : le cron qui le reçoit est alors un
// no-op.
func ForProvider(provider sharedprovider.Provider) port.WorldLeaderboardStore {
	if provider == nil {
		return nil
	}
	return New(provider)
}

// read prend un lecteur court de la base partagée, le passe à fn et le relâche.
func (r *Repo) read(ctx context.Context, fn func(db *sql.DB) error) error {
	db, release, err := r.provider.Get(ctx)
	if err != nil {
		return err
	}
	defer release()
	return fn(db)
}

// SeasonPlayers lit les joueurs distincts de la saison. Top N par playlist = profondeur affichée :
// l'enrichissement auto ne fetch pas les rangs jamais montrés (duckdb.WorldLeaderboardTopN).
func (r *Repo) SeasonPlayers(ctx context.Context, season string) (out []domain.WorldPlayerRef, err error) {
	err = r.read(ctx, func(db *sql.DB) error {
		out, err = duckdb.WorldSeasonPlayers(ctx, db, season, duckdb.WorldLeaderboardTopN)
		return err
	})
	return out, err
}

// SnapshotAge rend l'âge du dernier snapshot de la saison.
func (r *Repo) SnapshotAge(ctx context.Context, season string) (age time.Duration, ok bool, err error) {
	err = r.read(ctx, func(db *sql.DB) error {
		age, ok, err = duckdb.WorldCSRSnapshotAge(ctx, db, season)
		return err
	})
	return age, ok, err
}

// LatestSeason rend la saison persistée la plus récente (rang numérique, pas lexicographique).
func (r *Repo) LatestSeason(ctx context.Context) (season string, ok bool, err error) {
	err = r.read(ctx, func(db *sql.DB) error {
		season, ok, err = duckdb.WorldCSRLatestSeason(ctx, db)
		return err
	})
	return season, ok, err
}

// AssessBatch lit la qualité du lot servi (reader RO court : jamais de lease writer pendant le scrape) et la
// compare à celle du lot candidat, mesuré EN MÉMOIRE avec la même définition (WorldCSRStatsOfEntries).
func (r *Repo) AssessBatch(ctx context.Context, titleSlug, season, playlist string, entries []domain.LeaderboardEntry, depthLimit int) (port.WorldBatchVerdict, error) {
	var served duckdb.WorldCSRBatchStats
	var hasServed bool
	err := r.read(ctx, func(db *sql.DB) (err error) {
		served, hasServed, err = duckdb.WorldCSRServedBatchStats(ctx, db, titleSlug, season, playlist)
		return err
	})
	if err != nil {
		return port.WorldBatchVerdict{}, err
	}
	if !hasServed {
		return port.WorldBatchVerdict{}, nil
	}
	candidate := duckdb.WorldCSRStatsOfEntries(entries)
	return port.WorldBatchVerdict{
		HasServed: true,
		Served:    port.WorldBatchStats{Rows: served.Rows, WithXUID: served.WithXUID},
		Candidate: port.WorldBatchStats{Rows: candidate.Rows, WithXUID: candidate.WithXUID},
		Degraded:  duckdb.DegradedBatchReason(served, candidate, depthLimit),
	}, nil
}

// AcquireWriter prend le writer partagé par le fournisseur (drain des lecteurs, bascule RW).
func (r *Repo) AcquireWriter(ctx context.Context) (port.WorldLeaderboardWriter, func(), error) {
	wh, err := r.provider.AcquireWriter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return &writer{db: wh.DB()}, wh.Release, nil
}

// writer écrit dans le handle partagé pris par AcquireWriter.
type writer struct {
	db *sql.DB
}

func (w *writer) InsertPlayerSeasonStats(ctx context.Context, stats []domain.WorldPlayerSeasonStats) (int, error) {
	return duckdb.InsertPlayerSeasonStats(ctx, w.db, stats)
}

func (w *writer) InsertCSRSnapshot(ctx context.Context, titleSlug string, entries []domain.LeaderboardEntry) (int, error) {
	return duckdb.InsertWorldCSRSnapshot(ctx, w.db, titleSlug, entries)
}

func (w *writer) RefreshSeasonCatalog(ctx context.Context, titleSlug string, seasons []domain.WorldSeasonRef) (int, error) {
	return ops.RefreshSeasonCatalog(ctx, w.db, titleSlug, seasons)
}
