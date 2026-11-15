package port

// world_leaderboard.go — ce que le cron du classement mondial lit et écrit dans la base partagée : joueurs
// de la saison, fraîcheur et saison des snapshots, qualité du lot servi, et la fenêtre d'écriture des
// snapshots, des stats par joueur et du catalogue de saisons. La coordination avec la synchronisation (lecteur
// en vol, bascule B-swap) est celle du fournisseur de la base partagée, cachée derrière le magasin.

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
)

// WorldBatchStats décrit la qualité d'un lot du classement mondial : nombre de lignes et nombre d'entrées
// portant un xuid.
type WorldBatchStats struct {
	Rows     int
	WithXUID int
}

// WorldBatchVerdict compare un lot scrapé au lot servi pour une playlist.
type WorldBatchVerdict struct {
	// HasServed est faux quand aucun lot n'est servi pour (titre, saison, playlist) : première capture,
	// rien à protéger.
	HasServed bool
	Served    WorldBatchStats
	Candidate WorldBatchStats
	// Degraded est non vide quand le lot candidat s'est effondré (volume ou identification) face au lot
	// servi : il se refuse ; c'est la raison, en clair.
	Degraded string
}

// WorldLeaderboardWriter est une fenêtre d'écriture de la base partagée, valable jusqu'à sa libération.
type WorldLeaderboardWriter interface {
	// InsertPlayerSeasonStats écrit en append-only les stats agrégées de joueurs ; rend le nombre de lignes.
	InsertPlayerSeasonStats(ctx context.Context, stats []domain.WorldPlayerSeasonStats) (int, error)
	// InsertCSRSnapshot écrit en append-only les entrées d'un snapshot CSR du titre ; rend le nombre de lignes.
	InsertCSRSnapshot(ctx context.Context, titleSlug string, entries []domain.LeaderboardEntry) (int, error)
	// RefreshSeasonCatalog met à jour le catalogue de saisons du titre ; rend le nombre de saisons écrites.
	RefreshSeasonCatalog(ctx context.Context, titleSlug string, seasons []domain.WorldSeasonRef) (int, error)
}

// WorldLeaderboardStore lit et écrit le classement mondial dans la base partagée. Les lectures prennent un
// lecteur court, relâché avant le retour : le scrape, lui, dure plusieurs minutes et ne tient aucune base.
type WorldLeaderboardStore interface {
	// SeasonPlayers rend les joueurs distincts (gamertag et xuid) du top affiché d'une saison.
	SeasonPlayers(ctx context.Context, season string) ([]domain.WorldPlayerRef, error)
	// SnapshotAge rend l'âge du dernier snapshot de la saison ; ok est faux quand elle n'en a aucun.
	SnapshotAge(ctx context.Context, season string) (age time.Duration, ok bool, err error)
	// LatestSeason rend la saison persistée la plus récente ; ok est faux quand aucun snapshot n'existe.
	LatestSeason(ctx context.Context) (season string, ok bool, err error)
	// AssessBatch compare le lot scrapé au lot servi ; depthLimit est la profondeur que le cycle avait le
	// droit de ramener.
	AssessBatch(ctx context.Context, titleSlug, season, playlist string, entries []domain.LeaderboardEntry, depthLimit int) (WorldBatchVerdict, error)
	// AcquireWriter prend la fenêtre d'écriture et la fonction qui la libère. Une vidange des lecteurs expirée
	// rend l'erreur du fournisseur (sharedprovider.ErrDrainTimeout), retentable.
	AcquireWriter(ctx context.Context) (WorldLeaderboardWriter, func(), error)
}
