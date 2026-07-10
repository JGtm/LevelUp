// Package duckdb — LeaderboardRepo : classement CSR depuis les données locales.
//
// Sprint 54 E : LeaderboardRepository.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// LeaderboardRepo implémente port.LeaderboardRepository.
type LeaderboardRepo struct {
	pdb *PlayerDB
}

// NewLeaderboardRepo crée un LeaderboardRepo.
func NewLeaderboardRepo(pdb *PlayerDB) *LeaderboardRepo {
	return &LeaderboardRepo{pdb: pdb}
}

// GetLocalLeaderboard retourne les joueurs locaux triés par CSR DESC.
// Utilise match_skill_rank (table v5.3) depuis la player DB du joueur courant.
// Dans l'architecture actuelle, ce chemin ne connaît pas les autres stats.duckdb locaux ;
// il expose donc un classement dégradé mais fiable avec le CSR courant du joueur résolu.
// titleSlug, season et playlist sont des filtres optionnels (chaîne vide = tous).
//
// Split cross-DB en 2 phases (ADR 0016) :
//   - Phase A : match_skill_rank (player) sur pdb.Player.
//   - Phase B : match_registry (shared) via SharedReader avec IN match_ids.
//   - Phase C : merge + effective_type + filtre playlist + tri sort_time +
//     csr_value + LIMIT 1, côté Go.
//
// filtres optionnels (titleSlug/season/playlist) : la complexité reflète la cardinalité
// métier, pas un défaut de conception. Splitter rendrait le flow A→B→C illisible.
//
//nolint:funlen,gocyclo // Cross-DB merge avec 3 phases séquentielles et branches de
func (r *LeaderboardRepo) GetLocalLeaderboard(ctx context.Context, titleSlug, season, playlist string) ([]domain.LeaderboardEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Phase A : tous les match_skill_rank du joueur.
	type msrRow struct {
		matchID       string
		ratingValue   float64
		tier          *string
		tierLabel     *string
		subTier       int
		ratingType    string
		playlistGroup string
		startTime     *time.Time
		updatedAt     *time.Time
		createdAt     *time.Time
	}
	const qMSR = `
		SELECT match_id, rating_value, tier, tier_label, COALESCE(sub_tier, 0),
		       COALESCE(rating_type, ''), COALESCE(playlist_group, ''),
		       start_time, updated_at, created_at
		FROM match_skill_rank_latest
		WHERE rating_value > 0`
	rows, err := r.pdb.Player.Query(ctx, qMSR)
	if err != nil {
		return nil, fmt.Errorf("LeaderboardRepo.GetLocalLeaderboard: phase A: %w", err)
	}
	var msrs []msrRow
	matchIDs := make([]string, 0)
	for rows.Next() {
		var m msrRow
		var startT, updatedT, createdT sql.NullTime
		var tier, tierLabel sql.NullString
		if err := rows.Scan(&m.matchID, &m.ratingValue, &tier, &tierLabel, &m.subTier,
			&m.ratingType, &m.playlistGroup, &startT, &updatedT, &createdT); err != nil {
			rows.Close()
			return nil, fmt.Errorf("LeaderboardRepo.GetLocalLeaderboard scan A: %w", err)
		}
		if tier.Valid {
			s := tier.String
			m.tier = &s
		}
		if tierLabel.Valid {
			s := tierLabel.String
			m.tierLabel = &s
		}
		if startT.Valid {
			t := startT.Time
			m.startTime = &t
		}
		if updatedT.Valid {
			t := updatedT.Time
			m.updatedAt = &t
		}
		if createdT.Valid {
			t := createdT.Time
			m.createdAt = &t
		}
		msrs = append(msrs, m)
		matchIDs = append(matchIDs, m.matchID)
	}
	rows.Close()
	if len(msrs) == 0 {
		return nil, nil
	}

	// Phase B : enrich match_registry via SharedReader.
	type registryRow struct {
		isRanked     bool
		playlistName string
		pairName     string
		startTimeUTC *time.Time
	}
	registryByMatch := make(map[string]registryRow, len(matchIDs))
	{
		sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("LeaderboardRepo.GetLocalLeaderboard: shared reader: %w", err)
		}
		query := fmt.Sprintf(`
			SELECT match_id,
			       COALESCE(is_ranked, FALSE),
			       COALESCE(playlist_name, ''),
			       COALESCE(pair_name, ''),
			       `+StartTimeCanonicalSQL("")+`
			FROM match_registry
			WHERE match_id IN (%s)`, Placeholders(len(matchIDs)))
		regRows, err := sharedDB.QueryContext(ctx, query, ToAnySlice(matchIDs)...)
		if err != nil {
			release()
			return nil, fmt.Errorf("LeaderboardRepo.GetLocalLeaderboard: phase B: %w", err)
		}
		for regRows.Next() {
			var mid string
			var info registryRow
			var startT sql.NullTime
			if err := regRows.Scan(&mid, &info.isRanked, &info.playlistName, &info.pairName, &startT); err != nil {
				regRows.Close()
				release()
				return nil, fmt.Errorf("LeaderboardRepo.GetLocalLeaderboard scan B: %w", err)
			}
			if startT.Valid {
				t := startT.Time
				info.startTimeUTC = &t
			}
			registryByMatch[mid] = info
		}
		regRows.Close()
		release()
	}

	// Phase C : merge + effective_type + filtre playlist + tri + top 1.
	type candidateRow struct {
		csrValue     float64
		tier         string
		subTier      int
		playlistName string
		sortTime     *time.Time
	}
	cands := make([]candidateRow, 0, len(msrs))
	playlistLower := strings.ToLower(playlist)
	for _, m := range msrs {
		reg, hasRegistry := registryByMatch[m.matchID]
		// effective_type : 'CSR' si registry+is_ranked-like, sinon 'LUSR' si
		// registry présent, sinon UPPER(rating_type) du msr.
		isCSR := false
		if hasRegistry {
			plLower := strings.ToLower(reg.playlistName)
			pnLower := strings.ToLower(reg.pairName)
			if reg.isRanked || strings.Contains(plLower, "ranked") || strings.Contains(pnLower, "ranked") {
				isCSR = true
			}
		} else if strings.EqualFold(m.ratingType, "CSR") {
			isCSR = true
		}
		if !isCSR {
			continue
		}

		// playlist_name = COALESCE(mr.playlist_name, msr.playlist_group, '')
		plName := ""
		if hasRegistry && reg.playlistName != "" {
			plName = reg.playlistName
		} else {
			plName = m.playlistGroup
		}

		// Filtre playlist (équivalent ILIKE %playlist%).
		if playlist != "" {
			if !strings.Contains(strings.ToLower(plName), playlistLower) {
				continue
			}
		}

		// sort_time = COALESCE(mr.start_time_utc, ..., msr.start_time, msr.updated_at, msr.created_at).
		var sortTime *time.Time
		switch {
		case hasRegistry && reg.startTimeUTC != nil:
			sortTime = reg.startTimeUTC
		case m.startTime != nil:
			sortTime = m.startTime
		case m.updatedAt != nil:
			sortTime = m.updatedAt
		case m.createdAt != nil:
			sortTime = m.createdAt
		}

		// tier = COALESCE(NULLIF(tier,''), NULLIF(tier_label,''), '—').
		tier := "—"
		if m.tier != nil && strings.TrimSpace(*m.tier) != "" {
			tier = *m.tier
		} else if m.tierLabel != nil && strings.TrimSpace(*m.tierLabel) != "" {
			tier = *m.tierLabel
		}

		cands = append(cands, candidateRow{
			csrValue:     m.ratingValue,
			tier:         tier,
			subTier:      m.subTier,
			playlistName: plName,
			sortTime:     sortTime,
		})
	}
	// Tri : sort_time DESC NULLS LAST, csr_value DESC.
	sort.SliceStable(cands, func(i, j int) bool {
		ai, aj := cands[i].sortTime, cands[j].sortTime
		if ai != nil && aj != nil {
			if !ai.Equal(*aj) {
				return ai.After(*aj)
			}
		} else if ai != nil {
			return true
		} else if aj != nil {
			return false
		}
		return cands[i].csrValue > cands[j].csrValue
	})
	// LIMIT 1.
	if len(cands) > 1 {
		cands = cands[:1]
	}

	entries := make([]domain.LeaderboardEntry, 0, len(cands))
	for i, c := range cands {
		entry := domain.LeaderboardEntry{
			XUID:      r.pdb.XUID,
			Gamertag:  r.pdb.Gamertag,
			TitleSlug: titleSlug,
			Season:    season,
			Playlist:  playlist,
			Tier:      c.tier,
			SubTier:   c.subTier,
			CSR:       int(math.Round(c.csrValue)),
			CSRValue:  int(math.Round(c.csrValue)),
			IsLocal:   true,
			Rank:      i + 1,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
