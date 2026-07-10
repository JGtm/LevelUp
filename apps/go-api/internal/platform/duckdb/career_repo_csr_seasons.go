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
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// qDistinctCSRSeasons : season_id distincts présents dans les snapshots du joueur.
const qDistinctCSRSeasons = `
SELECT DISTINCT COALESCE(season_id, '')
FROM player_csr_snapshots_latest
WHERE season_id IS NOT NULL AND TRIM(season_id) != ''`

// qDistinctMatchCSRSeasons : season_id distincts présents dans le CSR par-match
// (shared.match_csrs). Couvre les joueurs importés (OpenSpartan) qui ont le CSR
// par-match — avec season_id — mais pas (encore) de snapshots Waypoint.
// Sur la table de base (pas la vue latest) : le DISTINCT season_id est identique
// et c'est portable (test + prod exposent match_csrs).
const qDistinctMatchCSRSeasons = `
SELECT DISTINCT season_id
FROM match_csrs
WHERE xuid = ? AND season_id IS NOT NULL AND TRIM(season_id) != ''`

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

	// CSR par-match (shared.match_csrs) : couvre les joueurs importés (OpenSpartan)
	// sans snapshots Waypoint mais dont le CSR par-match porte un season_id.
	r.collectSeasonsFromMatchCSRs(ctx, seen)

	// La saison courante est toujours proposable, même sans snapshot encore.
	if cur := strings.TrimSpace(r.currentCSRSID); cur != "" {
		seen[cur] = struct{}{}
	}
	if len(seen) == 0 {
		return []domain.CSRSeasonOption{}, nil
	}

	// C2b : libellé autoritatif "Saison N · Nom" (localisé) depuis season_catalog
	// (scrape Waypoint). Best-effort : nil map (indisponible) → fallback "Saison N"
	// dérivé (csrSeasonLabel). Match season_id insensible à la casse (API carrière
	// "CsrSeason13-2" vs Waypoint "csrseason13-2").
	locale := ctxkeys.Locale(ctx)
	names := r.seasonCatalogNames(ctx)
	out := make([]domain.CSRSeasonOption, 0, len(seen))
	for sid := range seen {
		out = append(out, domain.CSRSeasonOption{
			SeasonID:  sid,
			Label:     SeasonSelectorLabel(locale, sid, names, csrSeasonLabel(sid)),
			IsCurrent: sid == r.currentCSRSID,
		})
	}
	sortCSRSeasonsDesc(out)
	return out, nil
}

// seasonCatalogNames lit season_catalog (shared) best-effort pour les libellés de
// saison. nil si shared indisponible ou table absente (DB legacy) — l'appelant
// retombe alors sur le libellé dérivé.
func (r *CareerRepo) seasonCatalogNames(ctx context.Context) map[string]SeasonName {
	if r.pdb == nil || r.pdb.SharedReader == nil {
		return nil
	}
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "AvailableCSRSeasons: shared indisponible pour season_catalog (libellés dérivés)",
			"xuid", r.pdb.XUID, "err", err)
		return nil
	}
	defer release()
	names, err := LoadSeasonCatalogNames(ctx, db, r.titleSlug())
	if err != nil {
		slog.WarnContext(ctx, "AvailableCSRSeasons: season_catalog illisible (libellés dérivés)",
			"xuid", r.pdb.XUID, "err", err)
		return nil
	}
	return names
}

// collectSeasonsFromMatchCSRs ajoute à `seen` les season_id présents dans le CSR
// par-match (shared.match_csrs) du joueur. Best-effort : toute indisponibilité
// shared est loggée et ignorée (les snapshots + la saison courante restent
// proposés). Source du fix « joueur importé OpenSpartan » : le CSR par-match a
// déjà le season_id, inutile d'importer les snapshots Waypoint.
func (r *CareerRepo) collectSeasonsFromMatchCSRs(ctx context.Context, seen map[string]struct{}) {
	if r.pdb == nil || r.pdb.SharedReader == nil {
		return
	}
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "AvailableCSRSeasons: shared indisponible pour match_csrs (skip)",
			"xuid", r.pdb.XUID, "err", err)
		return
	}
	defer release()
	rows, err := db.QueryContext(ctx, qDistinctMatchCSRSeasons, r.pdb.XUID)
	if err != nil {
		slog.WarnContext(ctx, "AvailableCSRSeasons: query match_csrs seasons échouée (skip)",
			"xuid", r.pdb.XUID, "err", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			slog.WarnContext(ctx, "AvailableCSRSeasons: scan match_csrs season échoué (skip)",
				"xuid", r.pdb.XUID, "err", err)
			return
		}
		if sid = strings.TrimSpace(sid); sid != "" {
			seen[sid] = struct{}{}
		}
	}
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
