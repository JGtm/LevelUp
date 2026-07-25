// Package service — explorer_target_seasons.go : bucketing "matchs par saison"
// du joueur cible à partir des start_time de ses matchs (shared DB) + les
// plages temporelles de saison (SeasonsCatalog). Joueur local = historique
// complet ; adversaire = matchs observés (communs).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/domain"
)

// seasonBreakdownConcurrency plafonne le fan-out d'appels live du breakdown par
// saison (1 saison = 1 appel SR par chemin CMS + au plus 1 pic CSR, ce dernier
// uniquement si la saison a des matchs — cf. fetchSeasonRow).
const seasonBreakdownConcurrency = 6

// seasonMatchmadePathFormat est le gabarit du chemin CMS matchmade d'une saison
// NUMÉROTÉE du catalogue ("season7" → "Seasons/Season7.json"). C'est la même
// convention que celle déjà consommée en lecture par
// groupMatchmadeSeasonsByNumber (préfixe "Seasons/").
const seasonMatchmadePathFormat = "Seasons/Season%d.json"

// seasonMatchmadePathExtra est la clé `extra` du TOML saisons portant le chemin
// CMS matchmade EXPLICITE d'une saison dont l'ID n'est pas numéroté (ex.
// season_winter_22 → "Seasons/Season-Winter-Break-22.json", chemin relevé sur
// les appels réels). Prioritaire sur le gabarit déductible.
const seasonMatchmadePathExtra = "matchmade_path"

// numberedSeasonIDRe reconnaît un ID de saison de catalogue strictement numéroté
// ("season7", "season13"). Un ID hors gabarit (ex. "season_winter_22") n'a pas de
// chemin CMS déductible : il passe par extra.matchmade_path ou par le live.
var numberedSeasonIDRe = regexp.MustCompile(`^season(\d+)$`)

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
// cible pour TOUTES les saisons du catalogue : total de matchs matchmade (service
// record live filtré par saison) + pic de rang CSR (badge) des saisons jouées.
//
// V721-06 — les chemins CMS interrogés sont l'UNION de deux sources (cf.
// seasonCMSPaths) : le chemin déterministe de chaque saison du catalogue ET les
// chemins remontés par le live (Subqueries.SeasonIds). Auparavant seule la
// seconde source était utilisée : l'API ne liste pas tout l'historique du joueur,
// d'où des saisons réellement jouées affichées en barre vide (5/14 observé le
// 2026-07-22), indistinguables d'une saison non jouée.
//
// Best-effort, borné par le contexte (résultat partiel si le budget est dépassé).
// Fallback sur le bucketing local (computeMatchesPerSeason) sans auth / sans
// provider / sans saison / sans aucun chemin CMS exploitable. Statut (A3) :
// no_auth si jamais tenté, local_partial si repli local ou si une partie
// seulement des saisons a abouti, failed si aucune, ok sinon.
func (s *ExplorerService) computeSeasonBreakdown(
	ctx context.Context, targetXUID, targetGamertag string, hasAuth bool,
	playedSeasonIDs, engagedPlaylistIDs []string,
) ([]domain.SeasonMatchCount, domain.ExplorerLiveSectionStatus) {
	if !hasAuth {
		return s.computeMatchesPerSeason(ctx, targetXUID), domain.ExplorerLiveNoAuth
	}
	if s.deps.SeasonSR == nil || len(s.deps.Seasons) == 0 {
		return s.computeMatchesPerSeason(ctx, targetXUID), domain.ExplorerLiveLocalPartial
	}
	liveByNum := groupMatchmadeSeasonsByNumber(playedSeasonIDs)
	pathsPerSeason, attempted := planSeasonFetches(s.deps.Seasons, liveByNum)
	if attempted == 0 {
		// Aucun chemin CMS ni déductible ni remonté par le live (titre dont les
		// saisons ne portent pas d'ID numéroté ni de matchmade_path) : dégradation
		// propre vers le bucketing local plutôt qu'un axe intégralement vide.
		slog.InfoContext(ctx, "explorer_season_breakdown_no_cms_path",
			"xuid", targetXUID, "seasons_total", len(s.deps.Seasons))
		return s.computeMatchesPerSeason(ctx, targetXUID), domain.ExplorerLiveLocalPartial
	}

	out := make([]domain.SeasonMatchCount, len(s.deps.Seasons))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(seasonBreakdownConcurrency)
	for i := range s.deps.Seasons {
		i, entry := i, &s.deps.Seasons[i]
		row := domain.SeasonMatchCount{SeasonID: entry.ID, SeasonName: seasonShortLabel(entry)}
		paths := pathsPerSeason[i]
		if len(paths) == 0 {
			// Saison non interrogeable : on ne peut RIEN conclure (surtout pas
			// "0 match") → indéterminée, jamais une barre vide silencieuse.
			row.Unresolved = true
			out[i] = row
			continue
		}
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

	return out, seasonBreakdownStatus(ctx, targetXUID, out, attempted)
}

// planSeasonFetches calcule, pour chaque saison du catalogue (index préservé),
// les chemins CMS à interroger, et retourne le nombre de saisons réellement
// interrogeables. Fonction pure — aucun appel réseau.
func planSeasonFetches(seasons []SeasonCatalogEntry, liveByNum map[int][]string) ([][]string, int) {
	paths := make([][]string, len(seasons))
	attempted := 0
	for i := range seasons {
		paths[i] = seasonCMSPaths(&seasons[i], liveByNum)
		if len(paths[i]) > 0 {
			attempted++
		}
	}
	return paths, attempted
}

// seasonBreakdownStatus statue la section à partir des lignes produites et
// journalise le bilan (jamais de dégradation muette) :
//   - aucune saison résolue → failed
//   - au moins une indéterminée mais pas toutes → local_partial (« Live partiel »)
//   - sinon → ok
func seasonBreakdownStatus(
	ctx context.Context, targetXUID string, rows []domain.SeasonMatchCount, attempted int,
) domain.ExplorerLiveSectionStatus {
	unresolved, resolved, played := 0, 0, 0
	for i := range rows {
		if rows[i].Unresolved {
			unresolved++
			continue
		}
		resolved++
		if rows[i].Matches > 0 {
			played++
		}
	}
	slog.InfoContext(ctx, "explorer_season_breakdown",
		"xuid", targetXUID, "seasons_total", len(rows), "seasons_attempted", attempted,
		"seasons_played", played, "seasons_unresolved", unresolved)
	if unresolved == 0 {
		return domain.ExplorerLiveOK
	}
	slog.WarnContext(ctx, "explorer_season_breakdown_partial",
		"xuid", targetXUID, "seasons_unresolved", unresolved, "seasons_resolved", resolved)
	if resolved == 0 {
		return domain.ExplorerLiveFailed
	}
	return domain.ExplorerLiveLocalPartial
}

// seasonCMSPaths construit l'UNION DÉDUPLIQUÉE des chemins CMS matchmade d'une
// saison du catalogue :
//
//  1. le chemin déterministe (extra.matchmade_path, sinon déduit d'un ID
//     numéroté) — couvre les saisons que l'API ne remonte pas ;
//  2. les chemins remontés par le live pour ce numéro de saison — apportent les
//     opérations intra-saison ("Seasons/Season6-2.json"), non déductibles.
//
// Le chemin déterministe vient en tête (ordre stable). La déduplication garantit
// qu'un chemin présent dans les deux sources n'est compté QU'UNE fois (pas de
// double comptage sur le total de la saison). nil si aucune source.
func seasonCMSPaths(entry *SeasonCatalogEntry, liveByNum map[int][]string) []string {
	out := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if _, dup := seen[p]; dup {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	add(deterministicSeasonPath(entry))
	if num := seasonNumberFor(entry); num >= 0 {
		for _, p := range liveByNum[num] {
			add(p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// deterministicSeasonPath retourne le chemin CMS matchmade déductible d'une
// entrée du catalogue : extra.matchmade_path s'il est fourni (saisons hors
// gabarit, ex. l'opération hivernale), sinon "Seasons/Season{N}.json" pour un ID
// strictement numéroté. "" si non déductible (le live reste alors la seule
// source pour cette saison).
func deterministicSeasonPath(entry *SeasonCatalogEntry) string {
	if entry.Extra != nil {
		if p := strings.TrimSpace(entry.Extra[seasonMatchmadePathExtra]); p != "" {
			return p
		}
	}
	m := numberedSeasonIDRe.FindStringSubmatch(strings.ToLower(strings.TrimSpace(entry.ID)))
	if m == nil {
		return ""
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n <= 0 {
		return ""
	}
	return fmt.Sprintf(seasonMatchmadePathFormat, n)
}

// fetchSeasonRow remplit une SeasonMatchCount via les appels live : total des
// matchs matchmade de la saison (service record filtré par seasonId, sommé sur
// tous les chemins CMS de la saison — ex. opérations Season6 + Season6-2) + pic
// de rang CSR (badge). Best-effort par champ.
//
// Trois états produits (V721-06) : aucun chemin n'a répondu → Unresolved (on ne
// sait pas) ; l'API a répondu 0 → saison non jouée (aucun appel CSR, un badge
// sur une saison sans match n'a pas de sens) ; total > 0 → saison jouée + badge.
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
	answered := 0
	for _, p := range cmsPaths {
		total, err := s.deps.SeasonSR.FetchSeasonServiceRecord(ctx, targetGamertag, p, nil)
		if err != nil {
			// Chemin CMS inconnu de l'API (404) ou appel en échec : on ne peut pas
			// conclure "0 match". Loggué AVANT dégradation ; le bilan agrégé est
			// remonté en WARN par seasonBreakdownStatus.
			slog.DebugContext(ctx, "explorer_season_path_failed",
				"gamertag", targetGamertag, "season_id", row.SeasonID, "cms_path", p, "err", err)
			continue
		}
		answered++
		row.Matches += total
	}
	if answered == 0 {
		row.Unresolved = true
		return row
	}
	if row.Matches <= 0 {
		return row // saison non jouée (réponse API) → aucun appel CSR
	}

	// CSR : n'interroge que les playlists ranked réellement engagées par le
	// joueur (intersection côté provider) → 0 appel si engagedPlaylistIDs vide.
	if s.deps.SeasonCSR != nil && csrSeasonID != "" && targetXUID != "" && len(engagedPlaylistIDs) > 0 {
		peak, err := s.deps.SeasonCSR.SeasonPeakCSR(ctx, targetXUID, csrSeasonID, engagedPlaylistIDs)
		switch {
		case err != nil:
			slog.DebugContext(ctx, "explorer_season_csr_failed",
				"xuid", targetXUID, "csr_season_id", csrSeasonID, "err", err)
		case peak != nil:
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
//
// C'est la SECONDE source de seasonCMSPaths : la liste live est incomplète (elle
// ne couvre pas tout l'historique du joueur) mais elle est la seule à porter les
// opérations intra-saison, non déductibles du numéro.
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
