// Package duckdb — MetadataRepo : gestion des saisons et snapshots Waypoint.
//
// Sprint 54 A : tables season_calendars, csr_season_calendars, waypoint_resource_snapshots.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// MetadataRepo implémente port.MetadataRepository.
type MetadataRepo struct {
	meta *DB
}

// NewMetadataRepo crée un MetadataRepo depuis un PlayerDB (utilise la DB metadata).
func NewMetadataRepo(pdb *PlayerDB) *MetadataRepo {
	return &MetadataRepo{meta: pdb.Metadata}
}

// NewMetadataRepoFromDB crée un MetadataRepo directement depuis une *DB (pour le CLI).
func NewMetadataRepoFromDB(meta *DB) *MetadataRepo {
	return &MetadataRepo{meta: meta}
}

// EnsureSeasonTables crée les tables de saisons si elles n'existent pas (idempotent).
func (r *MetadataRepo) EnsureSeasonTables(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS season_calendars (
			title_id     VARCHAR NOT NULL,
			season_id    VARCHAR NOT NULL,
			version      VARCHAR NOT NULL DEFAULT '',
			name         VARCHAR NOT NULL DEFAULT '',
			start_date   TIMESTAMP WITH TIME ZONE NOT NULL,
			end_date     TIMESTAMP WITH TIME ZONE,
			fetched_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
			content_hash VARCHAR NOT NULL DEFAULT '',
			etag         VARCHAR NOT NULL DEFAULT '',
			source_url   VARCHAR NOT NULL DEFAULT '',
			PRIMARY KEY (title_id, season_id)
		)`,
		`CREATE TABLE IF NOT EXISTS csr_season_calendars (
			title_id     VARCHAR NOT NULL,
			season_id    VARCHAR NOT NULL,
			version      VARCHAR NOT NULL DEFAULT '',
			name         VARCHAR NOT NULL DEFAULT '',
			start_date   TIMESTAMP WITH TIME ZONE NOT NULL,
			end_date     TIMESTAMP WITH TIME ZONE,
			fetched_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
			content_hash VARCHAR NOT NULL DEFAULT '',
			etag         VARCHAR NOT NULL DEFAULT '',
			source_url   VARCHAR NOT NULL DEFAULT '',
			PRIMARY KEY (title_id, season_id)
		)`,
		`CREATE TABLE IF NOT EXISTS waypoint_resource_snapshots (
			title_id     VARCHAR NOT NULL,
			resource_key VARCHAR NOT NULL,
			version      VARCHAR NOT NULL DEFAULT '',
			fetched_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
			content_hash VARCHAR NOT NULL DEFAULT '',
			etag         VARCHAR NOT NULL DEFAULT '',
			source_url   VARCHAR NOT NULL DEFAULT '',
			payload      VARCHAR NOT NULL DEFAULT '',
			PRIMARY KEY (title_id, resource_key, version)
		)`,
	}
	for _, q := range queries {
		if _, err := r.meta.Exec(ctx, q); err != nil {
			return fmt.Errorf("MetadataRepo.EnsureSeasonTables: %w", err)
		}
	}
	return nil
}

// GetCurrentSeason retourne la saison la plus récente sans end_date (ou MAX start_date).
func (r *MetadataRepo) GetCurrentSeason(ctx context.Context, titleID string) (*domain.SeasonCalendar, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		SELECT title_id, season_id, version, name, start_date, end_date,
		       fetched_at, content_hash, etag, source_url
		FROM season_calendars
		WHERE title_id = ?
		ORDER BY CASE WHEN end_date IS NULL THEN 0 ELSE 1 END, start_date DESC
		LIMIT 1`

	row := r.meta.QueryRow(ctx, q, titleID)
	return scanSeasonCalendar(row)
}

// ListSeasons retourne toutes les saisons d'un titre, triées par StartDate ASC.
//
// Utilisé par SeasonsCatalog pour la fusion TOML+DB (V2 saisons : pattern
// lazy-fetch + persist symétrique au battle pass — cf. season_pass_service).
func (r *MetadataRepo) ListSeasons(ctx context.Context, titleID string) ([]domain.SeasonCalendar, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		SELECT title_id, season_id, version, name, start_date, end_date,
		       fetched_at, content_hash, etag, source_url
		FROM season_calendars
		WHERE title_id = ?
		ORDER BY start_date ASC`

	rows, err := r.meta.Query(ctx, q, titleID)
	if err != nil {
		return nil, fmt.Errorf("MetadataRepo.ListSeasons: %w", err)
	}
	defer rows.Close()

	var results []domain.SeasonCalendar
	for rows.Next() {
		var s domain.SeasonCalendar
		var endDate sql.NullTime
		if err := rows.Scan(
			&s.TitleID, &s.SeasonID, &s.Version, &s.Name,
			&s.StartDate, &endDate,
			&s.FetchedAt, &s.ContentHash, &s.ETag, &s.SourceURL,
		); err != nil {
			return nil, fmt.Errorf("MetadataRepo.ListSeasons scan: %w", err)
		}
		if endDate.Valid {
			s.EndDate = &endDate.Time
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

// GetCSRSeasons retourne toutes les saisons CSR pour un titre.
func (r *MetadataRepo) GetCSRSeasons(ctx context.Context, titleID string) ([]domain.CSRSeasonCalendar, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		SELECT title_id, season_id, version, name, start_date, end_date,
		       fetched_at, content_hash, etag, source_url
		FROM csr_season_calendars
		WHERE title_id = ?
		ORDER BY start_date DESC`

	rows, err := r.meta.Query(ctx, q, titleID)
	if err != nil {
		return nil, fmt.Errorf("MetadataRepo.GetCSRSeasons: %w", err)
	}
	defer rows.Close()

	var results []domain.CSRSeasonCalendar
	for rows.Next() {
		var s domain.CSRSeasonCalendar
		var endDate sql.NullTime
		if err := rows.Scan(
			&s.TitleID, &s.SeasonID, &s.Version, &s.Name,
			&s.StartDate, &endDate,
			&s.FetchedAt, &s.ContentHash, &s.ETag, &s.SourceURL,
		); err != nil {
			return nil, fmt.Errorf("MetadataRepo.GetCSRSeasons scan: %w", err)
		}
		if endDate.Valid {
			s.EndDate = &endDate.Time
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

// GetSeasonByDate retourne la saison active à la date donnée (format "2006-01-02").
func (r *MetadataRepo) GetSeasonByDate(ctx context.Context, titleID, date string) (*domain.SeasonCalendar, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		SELECT title_id, season_id, version, name, start_date, end_date,
		       fetched_at, content_hash, etag, source_url
		FROM season_calendars
		WHERE title_id = ?
		  AND start_date <= ?::TIMESTAMP
		  AND (end_date IS NULL OR end_date > ?::TIMESTAMP)
		ORDER BY start_date DESC
		LIMIT 1`

	row := r.meta.QueryRow(ctx, q, titleID, date, date)
	return scanSeasonCalendar(row)
}

// UpsertSeason insère ou met à jour une saison dans season_calendars.
func (r *MetadataRepo) UpsertSeason(ctx context.Context, s domain.SeasonCalendar) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var endDate interface{}
	if s.EndDate != nil {
		endDate = *s.EndDate
	}
	// SELECT-then-write anti-ART (cf. (*DB).UpsertNoConflict) sur metadata.duckdb.
	err := r.meta.UpsertNoConflict(ctx,
		`SELECT 1 FROM season_calendars WHERE title_id = ? AND season_id = ?`,
		[]any{s.TitleID, s.SeasonID},
		`UPDATE season_calendars SET
		   version=?, name=?, start_date=?, end_date=?, fetched_at=?,
		   content_hash=?, etag=?, source_url=?
		 WHERE title_id = ? AND season_id = ?`,
		[]any{s.Version, s.Name, s.StartDate, endDate, s.FetchedAt,
			s.ContentHash, s.ETag, s.SourceURL, s.TitleID, s.SeasonID},
		`INSERT INTO season_calendars
		   (title_id, season_id, version, name, start_date, end_date, fetched_at, content_hash, etag, source_url)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{s.TitleID, s.SeasonID, s.Version, s.Name, s.StartDate, endDate,
			s.FetchedAt, s.ContentHash, s.ETag, s.SourceURL},
	)
	if err != nil {
		return fmt.Errorf("MetadataRepo.UpsertSeason: %w", err)
	}
	return nil
}

// UpsertCSRSeason insère ou met à jour une saison CSR dans csr_season_calendars.
func (r *MetadataRepo) UpsertCSRSeason(ctx context.Context, s domain.CSRSeasonCalendar) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var endDate interface{}
	if s.EndDate != nil {
		endDate = *s.EndDate
	}
	// SELECT-then-write anti-ART (cf. (*DB).UpsertNoConflict) sur metadata.duckdb.
	err := r.meta.UpsertNoConflict(ctx,
		`SELECT 1 FROM csr_season_calendars WHERE title_id = ? AND season_id = ?`,
		[]any{s.TitleID, s.SeasonID},
		`UPDATE csr_season_calendars SET
		   version=?, name=?, start_date=?, end_date=?, fetched_at=?,
		   content_hash=?, etag=?, source_url=?
		 WHERE title_id = ? AND season_id = ?`,
		[]any{s.Version, s.Name, s.StartDate, endDate, s.FetchedAt,
			s.ContentHash, s.ETag, s.SourceURL, s.TitleID, s.SeasonID},
		`INSERT INTO csr_season_calendars
		   (title_id, season_id, version, name, start_date, end_date, fetched_at, content_hash, etag, source_url)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{s.TitleID, s.SeasonID, s.Version, s.Name, s.StartDate, endDate,
			s.FetchedAt, s.ContentHash, s.ETag, s.SourceURL},
	)
	if err != nil {
		return fmt.Errorf("MetadataRepo.UpsertCSRSeason: %w", err)
	}
	return nil
}

// UpsertSnapshot enregistre un snapshot de ressource Waypoint.
func (r *MetadataRepo) UpsertSnapshot(ctx context.Context, snap domain.WaypointResourceSnapshot) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// SELECT-then-write anti-ART (cf. (*DB).UpsertNoConflict) sur metadata.duckdb.
	err := r.meta.UpsertNoConflict(ctx,
		`SELECT 1 FROM waypoint_resource_snapshots WHERE title_id = ? AND resource_key = ? AND version = ?`,
		[]any{snap.TitleID, snap.ResourceKey, snap.Version},
		`UPDATE waypoint_resource_snapshots SET
		   fetched_at=?, content_hash=?, etag=?, source_url=?, payload=?
		 WHERE title_id = ? AND resource_key = ? AND version = ?`,
		[]any{snap.FetchedAt, snap.ContentHash, snap.ETag, snap.SourceURL, snap.Payload,
			snap.TitleID, snap.ResourceKey, snap.Version},
		`INSERT INTO waypoint_resource_snapshots
		   (title_id, resource_key, version, fetched_at, content_hash, etag, source_url, payload)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{snap.TitleID, snap.ResourceKey, snap.Version,
			snap.FetchedAt, snap.ContentHash, snap.ETag, snap.SourceURL, snap.Payload},
	)
	if err != nil {
		return fmt.Errorf("MetadataRepo.UpsertSnapshot: %w", err)
	}
	return nil
}

// GetSnapshot retourne le dernier snapshot d'une ressource (ORDER BY fetched_at DESC).
func (r *MetadataRepo) GetSnapshot(ctx context.Context, titleID, resourceKey string) (*domain.WaypointResourceSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		SELECT title_id, resource_key, version, fetched_at, content_hash, etag, source_url, payload
		FROM waypoint_resource_snapshots
		WHERE title_id = ? AND resource_key = ?
		ORDER BY fetched_at DESC
		LIMIT 1`

	row := r.meta.QueryRow(ctx, q, titleID, resourceKey)
	var snap domain.WaypointResourceSnapshot
	err := row.Scan(
		&snap.TitleID, &snap.ResourceKey, &snap.Version,
		&snap.FetchedAt, &snap.ContentHash, &snap.ETag, &snap.SourceURL, &snap.Payload,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("MetadataRepo.GetSnapshot: %w", err)
	}
	return &snap, nil
}

// GetAssistsCoef retourne slope et intercept pour expected_assists.
// Priorité au mode exact ; fallback sur '__global__' si absent ou table vide.
func (r *MetadataRepo) GetAssistsCoef(ctx context.Context, gameVariantName string) (slope, intercept float64, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const q = `
		SELECT COALESCE(m.slope, g.slope), COALESCE(m.intercept, g.intercept)
		FROM assists_model_coefs g
		LEFT JOIN assists_model_coefs m ON m.game_variant_name = ?
		WHERE g.game_variant_name = '__global__'`

	row := r.meta.QueryRow(ctx, q, gameVariantName)
	if scanErr := row.Scan(&slope, &intercept); scanErr != nil {
		return 0, 0, fmt.Errorf("GetAssistsCoef: %w", scanErr)
	}
	return slope, intercept, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func scanSeasonCalendar(row *sql.Row) (*domain.SeasonCalendar, error) {
	var s domain.SeasonCalendar
	var endDate sql.NullTime
	err := row.Scan(
		&s.TitleID, &s.SeasonID, &s.Version, &s.Name,
		&s.StartDate, &endDate,
		&s.FetchedAt, &s.ContentHash, &s.ETag, &s.SourceURL,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanSeasonCalendar: %w", err)
	}
	if endDate.Valid {
		s.EndDate = &endDate.Time
	}
	return &s, nil
}

// ── Sprint 54-D : tables staging Waypoint ─────────────────────────────────────

// EnsureStagingTables crée les tables de staging brutes si elles n'existent pas (idempotent).
// Ces tables reçoivent les données brutes Waypoint (médailles, assets) en attendant
// leur migration vers le schéma définitif.
func (r *MetadataRepo) EnsureStagingTables(ctx context.Context) error {
	queries := []string{
		// Médailles brutes telles que retournées par l'API Waypoint.
		`CREATE TABLE IF NOT EXISTS waypoint_medals_raw (
			title_id       VARCHAR NOT NULL,
			medal_id       BIGINT  NOT NULL,
			name_id        VARCHAR NOT NULL DEFAULT '',
			description_id VARCHAR NOT NULL DEFAULT '',
			sprite_index   INTEGER NOT NULL DEFAULT 0,
			difficulty     VARCHAR NOT NULL DEFAULT '',
			medal_type     VARCHAR NOT NULL DEFAULT '',
			personal_score INTEGER NOT NULL DEFAULT 0,
			raw_json       VARCHAR NOT NULL DEFAULT '',
			fetched_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
			content_hash   VARCHAR NOT NULL DEFAULT '',
			PRIMARY KEY (title_id, medal_id)
		)`,
		// Assets (images, sons, etc.) bruts Waypoint.
		`CREATE TABLE IF NOT EXISTS waypoint_assets_raw (
			title_id     VARCHAR NOT NULL,
			asset_id     VARCHAR NOT NULL,
			asset_type   VARCHAR NOT NULL DEFAULT '',
			version_id   VARCHAR NOT NULL DEFAULT '',
			name         VARCHAR NOT NULL DEFAULT '',
			description  VARCHAR NOT NULL DEFAULT '',
			raw_json     VARCHAR NOT NULL DEFAULT '',
			fetched_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
			content_hash VARCHAR NOT NULL DEFAULT '',
			PRIMARY KEY (title_id, asset_id, version_id)
		)`,
	}
	for _, q := range queries {
		if _, err := r.meta.Exec(ctx, q); err != nil {
			return fmt.Errorf("MetadataRepo.EnsureStagingTables: %w", err)
		}
	}
	return nil
}

// PromoteMedalDifficultyType met à jour les colonnes difficulty et medal_type
// dans medal_definitions en joignant waypoint_medals_raw sur name_id → medal_name_id.
// Idempotent : ne touche que les lignes où difficulty = 'Normal' ET medal_type = ”
// (valeurs par défaut), pour ne pas écraser des surcharges manuelles.
func (r *MetadataRepo) PromoteMedalDifficultyType(ctx context.Context, titleID string) (int64, error) {
	result, err := r.meta.Exec(ctx, `
		UPDATE medal_definitions
		SET difficulty = wr.difficulty,
		    medal_type  = wr.medal_type
		FROM waypoint_medals_raw wr
		WHERE medal_definitions.medal_name_id = TRY_CAST(wr.name_id AS BIGINT)
		  AND wr.title_id   = ?
		  AND wr.difficulty != ''
		  AND wr.medal_type != ''
	`, titleID)
	if err != nil {
		return 0, fmt.Errorf("PromoteMedalDifficultyType: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}
