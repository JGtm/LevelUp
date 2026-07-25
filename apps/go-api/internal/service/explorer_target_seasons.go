// Package service — explorer_target_seasons.go : bucketing "matchs par saison"
// du joueur cible à partir des start_time de ses matchs (shared DB) + les
// plages temporelles de saison (SeasonsCatalog). Joueur local = historique
// complet ; adversaire = matchs observés (communs).
package service

import (
	"context"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/domain"
)

// seasonBreakdownConcurrency plafonne le fan-out d'appels live du breakdown par
// saison (1 saison = ~3 appels : SR classé + SR non-classé + pic CSR).
const seasonBreakdownConcurrency = 6

// seasonShortLabel retourne le libellé compact d'une saison (extra.short_label
// si présent, ex. "S13", sinon le Label complet).
func seasonShortLabel(s *SeasonCatalogEntry) string {
	if s.Extra != nil {
		if sl := s.Extra["short_label"]; sl != "" {
			return sl
		}
	}
	return s.Label
}

// buildMatchesPerSeason compte les matchs par saison en rangeant chaque
// start_time dans la première saison dont l'intervalle [Start, End) le contient.
// Les saisons sont parcourues dans l'ordre fourni (DisplayOrder). Seules les
// saisons avec au moins 1 match sont retournées (ordre préservé). nil si vide.
func buildMatchesPerSeason(starts []time.Time, seasons []SeasonCatalogEntry) []domain.SeasonMatchCount {
	if len(starts) == 0 || len(seasons) == 0 {
		return nil
	}
	counts := make(map[string]int, len(seasons))
	for _, t := range starts {
		for i := range seasons {
			s := &seasons[i]
			if !t.Before(s.Start) && (s.End == nil || t.Before(*s.End)) {
				counts[s.ID]++
				break
			}
		}
	}
	out := make([]domain.SeasonMatchCount, 0, len(seasons))
	for i := range seasons {
		s := &seasons[i]
		if n := counts[s.ID]; n > 0 {
			out = append(out, domain.SeasonMatchCount{
				SeasonID:   s.ID,
				SeasonName: seasonShortLabel(s),
				Matches:    n,
			})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// computeSeasonBreakdown construit le breakdown "matchs par saison" du joueur
// cible pour TOUTES les saisons du catalogue : matchs classés/non-classés (via le
// service record live filtré par saison + isRanked) + pic de rang CSR (badge).
// Best-effort, borné par le contexte (résultat partiel si le budget est dépassé).
// Fallback sur le bucketing local (computeMatchesPerSeason) sans auth / sans
// provider / sans saison jouée. Statut (A3) : no_auth si jamais tenté,
// local_partial si repli local (matchs communs uniquement — cf. Lot A2, non
// traité : la somme peut rester partielle même en mode ok), ok si le breakdown
// live par saison a tourné.
func (s *ExplorerService) computeSeasonBreakdown(
	ctx context.Context, targetXUID, targetGamertag string, hasAuth bool,
	playedSeasonIDs, engagedPlaylistIDs []string,
) ([]domain.SeasonMatchCount, domain.ExplorerLiveSectionStatus) {
	if !hasAuth {
		return s.computeMatchesPerSeason(ctx, targetXUID), domain.ExplorerLiveNoAuth
	}
	playedByNum := groupMatchmadeSeasonsByNumber(playedSeasonIDs)
	if s.deps.SeasonSR == nil || len(s.deps.Seasons) == 0 || len(playedByNum) == 0 {
		return s.computeMatchesPerSeason(ctx, targetXUID), domain.ExplorerLiveLocalPartial
	}

	out := make([]domain.SeasonMatchCount, len(s.deps.Seasons))
	resolved := 0
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(seasonBreakdownConcurrency)
	for i := range s.deps.Seasons {
		i, entry := i, &s.deps.Seasons[i]
		row := domain.SeasonMatchCount{SeasonID: entry.ID, SeasonName: seasonShortLabel(entry)}
		num := seasonNumberFor(entry)
		paths := playedByNum[num]
		if num < 0 || len(paths) == 0 {
			out[i] = row // saison non jouée → barre vide
			continue
		}
		resolved++
		csrSeasonID := ""
		if entry.Extra != nil {
			csrSeasonID = entry.Extra["csr_season_id"]
		}
		g.Go(func() error {
			out[i] = s.fetchSeasonRow(gctx, targetXUID, targetGamertag, row, paths, csrSeasonID, engagedPlaylistIDs)
			return nil
		})
	}
	_ = g.Wait()

	slog.InfoContext(ctx, "explorer_season_breakdown",
		"xuid", targetXUID, "seasons_total", len(out), "seasons_played", resolved)
	return out, domain.ExplorerLiveOK
}

// fetchSeasonRow remplit une SeasonMatchCount via les appels live : total des
// matchs matchmade de la saison (service record filtré par seasonId, sommé sur
// tous les chemins CMS de la saison — ex. opérations Season6 + Season6-2) + pic
// de rang CSR (badge). Best-effort par champ ; une erreur laisse le champ à zéro.
//
// NB : le split classé/non-classé n'est PAS récupérable ici. L'API officielle
// rejette `seasonId + isRanked` (HTTP 400) — le filtre isRanked exige aussi
// gameVariantCategory, donc un split fiable demanderait N catégories × 2 appels
// par saison (explosion). On expose donc le total par saison (combo seasonId
// seul, valide). Le proxy HaloDotAPI de SpartanRecord offrait ce split via un
// paramètre `?filter=ranked` qui n'existe pas sur l'API Waypoint brute.
func (s *ExplorerService) fetchSeasonRow(
	ctx context.Context, targetXUID, targetGamertag string,
	row domain.SeasonMatchCount, cmsPaths []string, csrSeasonID string, engagedPlaylistIDs []string,
) domain.SeasonMatchCount {
	for _, p := range cmsPaths {
		if total, err := s.deps.SeasonSR.FetchSeasonServiceRecord(ctx, targetGamertag, p, nil); err == nil {
			row.Matches += total
		}
	}

	// CSR : n'interroge que les playlists ranked réellement engagées par le
	// joueur (intersection côté provider) → 0 appel si engagedPlaylistIDs vide.
	if s.deps.SeasonCSR != nil && csrSeasonID != "" && targetXUID != "" && len(engagedPlaylistIDs) > 0 {
		if peak, err := s.deps.SeasonCSR.SeasonPeakCSR(ctx, targetXUID, csrSeasonID, engagedPlaylistIDs); err == nil && peak != nil {
			row.CSRTier = peak.Tier
			row.CSRSubTier = peak.SubTier
			row.CSRBadgeImageURL = peak.BadgeURL
		}
	}
	return row
}

// seasonNumberFor extrait le numéro d'une saison du catalogue (short_label "S7"
// → 7, sinon l'ID "season7" → 7). -1 si introuvable.
func seasonNumberFor(entry *SeasonCatalogEntry) int {
	if n := extractSeasonNumber(seasonShortLabel(entry)); n >= 0 {
		return n
	}
	return extractSeasonNumber(entry.ID)
}

// groupMatchmadeSeasonsByNumber indexe par numéro de saison les chemins CMS
// MATCHMADE ("Seasons/SeasonN.json") issus de Subqueries.SeasonIds. Ignore les
// chemins CSR ("Csr/Seasons/..."). Un même numéro peut avoir plusieurs chemins
// (opérations intra-saison, ex. Season6.json + Season6-2.json → tous deux n°6).
func groupMatchmadeSeasonsByNumber(seasonIDs []string) map[int][]string {
	out := make(map[int][]string)
	for _, id := range seasonIDs {
		if !strings.HasPrefix(id, "Seasons/") {
			continue
		}
		if n := extractSeasonNumber(id); n >= 0 {
			out[n] = append(out[n], id)
		}
	}
	return out
}

// seasonNumberRe capture le premier entier d'une chaîne.
var seasonNumberRe = regexp.MustCompile(`\d+`)

// extractSeasonNumber retourne le premier entier d'une chaîne ("Seasons/Season7.json"
// → 7, "S13" → 13, "season10" → 10, "Winter 22" → 22). -1 si aucun.
func extractSeasonNumber(s string) int {
	m := seasonNumberRe.FindString(s)
	if m == "" {
		return -1
	}
	n, err := strconv.Atoi(m)
	if err != nil {
		return -1
	}
	return n
}
