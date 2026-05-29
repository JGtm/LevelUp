// Package duckdb — career_repo_csr_seasons.go : liste des saisons CSR
// proposables dans le menu déroulant "Classements" (page Carrière).
//
// Une saison apparaît si le joueur y a des snapshots CSR (= a joué en classé
// pendant la fenêtre de cette saison), plus la saison courante configurée.
// Source pure player-DB (player_csr_snapshots_latest) — pas de jointure shared.
package duckdb

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/domain"
)

// qDistinctCSRSeasons : season_id distincts présents dans les snapshots du joueur.
const qDistinctCSRSeasons = `
SELECT DISTINCT COALESCE(season_id, '')
FROM player_csr_snapshots_latest
WHERE season_id IS NOT NULL AND TRIM(season_id) != ''`

// AvailableCSRSeasons retourne les saisons CSR sélectionnables, triées récentes
// d'abord. Inclut toujours la saison courante (même sans snapshot). Best-effort :
// si player_csr_snapshots est absente, retourne la seule saison courante.
func (r *CareerRepo) AvailableCSRSeasons(ctx context.Context) ([]domain.CSRSeasonOption, error) {
	seen := make(map[string]struct{})

	rows, err := r.pdb.ReadDB().Query(ctx, qDistinctCSRSeasons)
	if err != nil {
		if !isTableNotFoundErr(err) {
			return nil, fmt.Errorf("AvailableCSRSeasons: %w", err)
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var sid string
			if err := rows.Scan(&sid); err != nil {
				return nil, fmt.Errorf("AvailableCSRSeasons scan: %w", err)
			}
			if sid = strings.TrimSpace(sid); sid != "" {
				seen[sid] = struct{}{}
			}
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	// La saison courante est toujours proposable, même sans snapshot encore.
	if cur := strings.TrimSpace(r.currentCSRSID); cur != "" {
		seen[cur] = struct{}{}
	}
	if len(seen) == 0 {
		return []domain.CSRSeasonOption{}, nil
	}

	out := make([]domain.CSRSeasonOption, 0, len(seen))
	for sid := range seen {
		out = append(out, domain.CSRSeasonOption{
			SeasonID:  sid,
			Label:     csrSeasonLabel(sid),
			IsCurrent: sid == r.currentCSRSID,
		})
	}
	sortCSRSeasonsDesc(out)
	return out, nil
}

// sortCSRSeasonsDesc trie les saisons de la plus récente à la plus ancienne via
// le numéro de saison parsé (le tri lexicographique placerait "CsrSeason2" après
// "CsrSeason13" — incorrect).
func sortCSRSeasonsDesc(opts []domain.CSRSeasonOption) {
	sort.SliceStable(opts, func(i, j int) bool {
		mi, ni := parseCSRSeasonNumber(opts[i].SeasonID)
		mj, nj := parseCSRSeasonNumber(opts[j].SeasonID)
		if mi != mj {
			return mi > mj
		}
		return ni > nj
	})
}

// csrSeasonLabel dérive un libellé court depuis le season_id.
// "CsrSeason13-1" → "Saison 13" ; fallback : le season_id brut.
func csrSeasonLabel(seasonID string) string {
	if major, _ := parseCSRSeasonNumber(seasonID); major > 0 {
		return fmt.Sprintf("Saison %d", major)
	}
	return seasonID
}

// parseCSRSeasonNumber extrait (major, minor) depuis "CsrSeason{major}-{minor}".
// minor vaut 0 si absent ("CsrSeason8"). (0,0) si non parsable.
func parseCSRSeasonNumber(seasonID string) (int, int) {
	s := strings.TrimSpace(seasonID)
	const prefix = "CsrSeason"
	if !strings.HasPrefix(s, prefix) {
		return 0, 0
	}
	s = s[len(prefix):]
	majorStr, minorStr, _ := strings.Cut(s, "-")
	major, err := strconv.Atoi(strings.TrimSpace(majorStr))
	if err != nil {
		return 0, 0
	}
	minor, _ := strconv.Atoi(strings.TrimSpace(minorStr))
	return major, minor
}
