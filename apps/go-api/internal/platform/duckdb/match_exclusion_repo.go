// Package duckdb — MatchExclusionRepo : lecture/écriture du flag is_excluded.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
)

// MatchExclusionRepo implémente port.MatchExclusionRepository.
type MatchExclusionRepo struct {
	pdb *PlayerDB
}

// NewMatchExclusionRepo crée un MatchExclusionRepo depuis un PlayerDB.
func NewMatchExclusionRepo(pdb *PlayerDB) *MatchExclusionRepo {
	return &MatchExclusionRepo{pdb: pdb}
}

// SetExclusion positionne is_excluded pour un match s'il diffère de l'état courant, et dit
// s'il l'a écrit. La lecture de l'état, la décision et l'écriture se font sous le verrou
// d'écriture de la base joueur (lease KindPlayer) et sur la même connexion : deux demandes
// qui se chevauchent (exclure puis réinclure) se sérialisent, la seconde décide sur l'état
// que laisse la première. ErrDBLocked remonte et est mappé en 503 par le handler.
func (r *MatchExclusionRepo) SetExclusion(ctx context.Context, matchID string, excluded bool) (changed bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	w, err := r.pdb.AcquirePlayerWriterTimeout(dblease.PlayerLeaseTimeout)
	if err != nil {
		return false, fmt.Errorf("MatchExclusionRepo.SetExclusion: lease: %w", err)
	}
	defer w.Release()

	rwDB, err := OpenReadWrite(r.pdb.Player.Path())
	if err != nil {
		return false, fmt.Errorf("MatchExclusionRepo.SetExclusion: open rw: %w", err)
	}
	defer rwDB.Close()

	current, err := excludedState(ctx, rwDB, matchID)
	if err != nil {
		return false, err
	}
	if current == excluded {
		return false, nil
	}
	// Append-only #23645 : INSERT pur stage='exclusion', valeur booléenne EXPLICITE
	// (is_excluded bidirectionnel, jamais NULL). Le merge-on-read prend la dernière
	// row du stage 'exclusion' ; loadExcludedPMERows lit player_match_enrichment_latest.
	_, err = rwDB.Exec(ctx, `
		INSERT INTO player_match_enrichment (match_id, is_excluded, stage)
		VALUES (?, ?, 'exclusion')
	`, matchID, excluded)
	if err != nil {
		return false, fmt.Errorf("MatchExclusionRepo.SetExclusion exec: %w", err)
	}
	// Toute écriture de player_match_enrichment invalide les lectures joueur mises en
	// cache : un champ de la table qu'elles porteraient resterait sinon périmé jusqu'au TTL.
	InvalidatePlayerReadCaches(ctx, r.pdb.XUID, r.pdb.TitleSlug)
	return true, nil
}

// excludedState rend l'état courant du flag is_excluded d'un match, lu sur la vue
// player_match_enrichment_latest (append-only) ; faux si aucune ligne n'existe.
func excludedState(ctx context.Context, db *DB, matchID string) (bool, error) {
	var excluded bool
	err := db.QueryRow(ctx, `
		SELECT COALESCE(is_excluded, FALSE)
		FROM player_match_enrichment_latest
		WHERE match_id = ?
		LIMIT 1`, matchID).Scan(&excluded)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("MatchExclusionRepo.SetExclusion état courant: %w", err)
	}
	return excluded, nil
}

// GetMatchRegistryInfo lit shared.match_registry pour un match donné.
// Renvoie domain.ErrMatchNotFound si le match n'existe pas.
func (r *MatchExclusionRepo) GetMatchRegistryInfo(ctx context.Context, matchID string) (domain.MatchRegistryInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var (
		info        domain.MatchRegistryInfo
		startTime   time.Time
		isRanked    sql.NullBool
		isFirefight sql.NullBool
		pairName    sql.NullString
	)
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return domain.MatchRegistryInfo{}, fmt.Errorf("MatchExclusionRepo.GetMatchRegistryInfo: %w", err)
	}
	defer release()

	err = db.QueryRowContext(ctx, `
		SELECT
			match_id,
			start_time,
			COALESCE(is_ranked, FALSE)    AS is_ranked,
			COALESCE(is_firefight, FALSE) AS is_firefight,
			COALESCE(pair_name, '')       AS pair_name
		FROM match_registry
		WHERE match_id = ?
		LIMIT 1
	`, matchID).Scan(&info.MatchID, &startTime, &isRanked, &isFirefight, &pairName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.MatchRegistryInfo{}, domain.ErrMatchNotFound
		}
		return domain.MatchRegistryInfo{}, fmt.Errorf("MatchExclusionRepo.GetMatchRegistryInfo: %w", err)
	}
	info.StartTime = startTime
	if isRanked.Valid {
		info.IsRanked = isRanked.Bool
	}
	if isFirefight.Valid {
		info.IsFirefight = isFirefight.Bool
	}
	if pairName.Valid {
		info.PairName = pairName.String
	}
	return info, nil
}

// ListExcluded retourne les matchs marqués is_excluded = TRUE avec métadonnées.
//
// Split cross-DB en 2 phases (ADR 0016) :
//   - Phase A : player_match_enrichment (player) sur pdb.Player.
//   - Phase B : match_registry (shared) via SharedReader avec IN match_ids.
//   - Phase C : merge + start_time fallback + tri DESC côté Go.
type excludedPMERow struct {
	matchID   string
	updatedAt *time.Time
}

type excludedRegistryRow struct {
	startTime *time.Time
	mapName   string
	pairName  string
}

func (r *MatchExclusionRepo) ListExcluded(ctx context.Context) ([]domain.ExcludedMatch, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pmes, matchIDs, err := r.loadExcludedPMERows(ctx)
	if err != nil {
		return nil, err
	}
	if len(pmes) == 0 {
		return nil, nil
	}

	registryByMatch, err := r.loadExcludedRegistryRows(ctx, matchIDs)
	if err != nil {
		return nil, err
	}

	results := mergeExcludedMatches(pmes, registryByMatch)
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].StartTime.After(results[j].StartTime)
	})
	return results, nil
}

// loadExcludedPMERows charge les match_id exclus depuis player_match_enrichment.
func (r *MatchExclusionRepo) loadExcludedPMERows(ctx context.Context) ([]excludedPMERow, []string, error) {
	// Append-only #23645 : lire la vue _latest — sinon un match exclu puis ré-inclus
	// (row FALSE plus récente) resterait listé si une vieille row TRUE subsiste.
	rows, err := r.pdb.Player.QueryRecovered(ctx, `
		SELECT match_id, updated_at
		FROM player_match_enrichment_latest
		WHERE is_excluded = TRUE`)
	if err != nil {
		return nil, nil, fmt.Errorf("MatchExclusionRepo.ListExcluded: phase A: %w", err)
	}
	defer rows.Close()
	var pmes []excludedPMERow
	matchIDs := make([]string, 0)
	for rows.Next() {
		var p excludedPMERow
		var updatedT sql.NullTime
		if err := rows.Scan(&p.matchID, &updatedT); err != nil {
			return nil, nil, fmt.Errorf("MatchExclusionRepo.ListExcluded scan A: %w", err)
		}
		if updatedT.Valid {
			t := updatedT.Time
			p.updatedAt = &t
		}
		pmes = append(pmes, p)
		matchIDs = append(matchIDs, p.matchID)
	}
	return pmes, matchIDs, nil
}

// loadExcludedRegistryRows enrichit avec match_registry via SharedReader.
func (r *MatchExclusionRepo) loadExcludedRegistryRows(ctx context.Context, matchIDs []string) (map[string]excludedRegistryRow, error) {
	out := make(map[string]excludedRegistryRow, len(matchIDs))
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchExclusionRepo.ListExcluded: shared reader: %w", err)
	}
	defer release()
	query := fmt.Sprintf(`
		SELECT match_id,
		       `+StartTimeCanonicalSQL("")+`,
		       COALESCE(map_name, ''),
		       COALESCE(pair_name, '')
		FROM match_registry
		WHERE match_id IN (%s)`, Placeholders(len(matchIDs)))
	regRows, err := sharedDB.QueryContext(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("MatchExclusionRepo.ListExcluded: phase B: %w", err)
	}
	defer regRows.Close()
	for regRows.Next() {
		var mid string
		var info excludedRegistryRow
		var startT sql.NullTime
		if err := regRows.Scan(&mid, &startT, &info.mapName, &info.pairName); err != nil {
			return nil, fmt.Errorf("MatchExclusionRepo.ListExcluded scan B: %w", err)
		}
		if startT.Valid {
			t := startT.Time
			info.startTime = &t
		}
		out[mid] = info
	}
	return out, nil
}

// mergeExcludedMatches assemble les DTOs en croisant PME et registry,
// avec fallback start_time → updated_at si absent.
func mergeExcludedMatches(pmes []excludedPMERow, registryByMatch map[string]excludedRegistryRow) []domain.ExcludedMatch {
	results := make([]domain.ExcludedMatch, 0, len(pmes))
	for _, p := range pmes {
		m := domain.ExcludedMatch{MatchID: p.matchID}
		reg, hasRegistry := registryByMatch[p.matchID]
		if hasRegistry && reg.startTime != nil {
			m.StartTime = *reg.startTime
		} else if p.updatedAt != nil {
			m.StartTime = *p.updatedAt
		}
		if hasRegistry {
			m.MapName = reg.mapName
			m.ModeName = reg.pairName
		}
		results = append(results, m)
	}
	return results
}
