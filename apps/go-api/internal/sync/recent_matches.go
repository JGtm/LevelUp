// Package sync — recent_matches.go : fetch LIVE read-only des N derniers matchs
// PvP d'un joueur arbitraire (xuid), SANS persistance.
//
// Contrairement au pipeline de sync (qui écrit en base via persist/BatchBuilder),
// ce fetcher est strictement en lecture : il sert les graphes "profil de combat"
// d'une cible non suivie et l'échantillon récent du Face à face. Le résultat est
// mis en cache mémoire par le décorateur service.CachedRecentMatchesProvider.
//
// Réutilise les primitives HTTP existantes (GetMatchHistory + GetMatchStats) et
// la projection ExtractParticipants. Aucune nouvelle route HTTP.
package sync

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// recentMatchesType : seuls les matchs matchmade alimentent les graphes profil de
// combat / l'échantillon Face à face (cohérent avec le filtre PvP local).
const recentMatchesType = "matchmaking"

// RecentMatchesFetcher implémente port.RecentMatchesProvider en live read-only.
// Stateless : un HaloAPIClient éphémère est construit par requête à partir des
// tokens du contexte (un même Spartan token lit l'historique de tout xuid).
type RecentMatchesFetcher struct {
	rps        int
	httpClient *http.Client // optionnel (tests) — injecté dans le client par requête
}

// NewRecentMatchesFetcher crée le fetcher. rps borne le rate-limit du client
// éphémère (défaut interne si <= 0).
func NewRecentMatchesFetcher(rps int) *RecentMatchesFetcher {
	return &RecentMatchesFetcher{rps: rps}
}

// WithHTTPClient injecte un *http.Client custom (tests d'intégration via
// httptest + RoundTripper). nil est ignoré.
func (f *RecentMatchesFetcher) WithHTTPClient(c *http.Client) *RecentMatchesFetcher {
	if c != nil {
		f.httpClient = c
	}
	return f
}

// FetchRecentMatches récupère les `limit` derniers matchs matchmade de `xuid` en
// live (liste + stats par match), projette la cible et retourne les lignes en
// ordre chronologique ASCENDANT (timeline gauche→droite des graphes). (nil, nil)
// sans auth ou xuid vide. Best-effort par match : un match en échec est ignoré.
func (f *RecentMatchesFetcher) FetchRecentMatches(
	ctx context.Context, xuid string, limit int,
) ([]domain.ExplorerTargetRecentMatch, error) {
	if xuid == "" || limit <= 0 {
		return nil, nil
	}
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil || tokens.SpartanToken == "" {
		slog.DebugContext(ctx, "recent_matches_skipped_no_auth", "xuid", xuid)
		return nil, nil
	}
	client := NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, f.rps)
	if f.httpClient != nil {
		client = client.WithHTTPClient(f.httpClient)
	}

	// IMPÉRATIF : format xuid(NNN) sinon l'API renvoie une réponse stale figée
	// (cf. mémoire reference_halo_api_xuid_format / incident mai 2026).
	history, err := client.GetMatchHistory(ctx, fmt.Sprintf("xuid(%s)", xuid), recentMatchesType, 0, limit)
	if err != nil {
		return nil, fmt.Errorf("RecentMatchesFetcher: historique %s: %w", xuid, err)
	}
	if len(history) == 0 {
		return nil, nil
	}

	statsByID := make(map[string]map[string]any, len(history))
	for _, h := range history {
		raw, sErr := client.GetMatchStats(ctx, h.MatchID)
		if sErr != nil {
			slog.WarnContext(ctx, "recent_matches_stats_failed", "xuid", xuid, "match_id", h.MatchID, "err", sErr)
			continue
		}
		statsByID[h.MatchID] = raw
	}
	rows := buildRecentMatchesFromStats(history, statsByID, xuid)
	slog.DebugContext(ctx, "recent_matches_fetched", "xuid", xuid, "requested", limit, "got", len(rows))
	return rows, nil
}

// buildRecentMatchesFromStats projette la cible `xuid` de chaque match (via
// ExtractParticipants) vers ExplorerTargetRecentMatch. Pure (testable sans HTTP).
// Tri chronologique ascendant par StartTime de l'historique. Les matchs sans
// stats ou sans la cible sont ignorés. mode_ui = sous-mode EN normalisé (via
// resolveLiveModeUI/ResolveModeUI ; FR appliqué ensuite par l'ExplorerService) ;
// map_ui = nom de carte brut. perfect_kills reste 0 (vient des médailles dans le
// chemin local, hors scope live).
func buildRecentMatchesFromStats(
	history []MatchHistoryEntry, statsByID map[string]map[string]any, xuid string,
) []domain.ExplorerTargetRecentMatch {
	rows := make([]domain.ExplorerTargetRecentMatch, 0, len(history))
	for _, h := range history {
		raw, ok := statsByID[h.MatchID]
		if !ok {
			continue
		}
		part := findParticipantByXUID(ExtractParticipants(raw), xuid)
		if part == nil {
			continue
		}
		// Mode/carte depuis MatchInfo (même source que ExtractRegistry) — sinon le
		// donut "Répartition des modes" regrouperait tout sous "Inconnu". Le mode est
		// normalisé en sous-mode EN via analysis.ResolveModeUI (MÊME résolution que le
		// chemin local) ; la traduction FR (mode_name_tr) est appliquée ensuite par
		// l'ExplorerService via le repo (metadata). matchInfo nil → "" (nil-safe).
		matchInfo, _ := raw["MatchInfo"].(map[string]any)
		rows = append(rows, domain.ExplorerTargetRecentMatch{
			MatchID:   h.MatchID,
			StartTime: parseRecentStartTime(h.StartTime),
			MapUI:     extractPublicName(matchInfo, "MapVariant"),
			ModeUI:    resolveLiveModeUI(matchInfo),
			// AssetId du pair : le MatchInfo live n'a pas de PublicName → ModeUI reste
			// souvent vide ici ; le repo le résout via shared.match_registry
			// (pair_id → pair_name) dans translateModeUIsFR.
			ModePairAssetID: extractAssetID(matchInfo, "PlaylistMapModePair"),
			Outcome:         recentDerefInt(part.Outcome),
			Rank:            part.Rank,
			Kills:           recentDerefInt(part.Kills),
			Deaths:          recentDerefInt(part.Deaths),
			Assists:         recentDerefInt(part.Assists),
			KDA:             recentDerefFloat(part.KDA),
			Score:           recentDerefInt(part.PersonalScore),
			DamageDealt:     int(recentDerefFloat(part.DamageDealt)),
			DamageTaken:     int(recentDerefFloat(part.DamageTaken)),
			MaxKillingSpree: recentDerefInt(part.MaxKillingSpree),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].StartTime.Before(rows[j].StartTime) })
	return rows
}

// resolveLiveModeUI extrait le sous-mode EN normalisé d'un match live depuis son
// MatchInfo, via analysis.ResolveModeUI (pair d'abord, repli game variant) — même
// normalisation que le chemin local. "" si rien d'exploitable.
func resolveLiveModeUI(matchInfo map[string]any) string {
	pair := extractPublicName(matchInfo, "PlaylistMapModePair")
	if m := analysis.ResolveModeUI(&pair, nil); m != nil && *m != "" {
		return *m
	}
	gv := extractPublicName(matchInfo, "UgcGameVariant")
	if m := analysis.ResolveModeUI(&gv, nil); m != nil {
		return *m
	}
	return ""
}

// findParticipantByXUID retourne la ligne participant du xuid cible, ou nil.
func findParticipantByXUID(parts []ParticipantRow, xuid string) *ParticipantRow {
	for i := range parts {
		if parts[i].XUID == xuid {
			return &parts[i]
		}
	}
	return nil
}

// parseRecentStartTime parse le StartTime ISO de l'historique ; time.Time zéro si vide/invalide.
func parseRecentStartTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if ts, err := parseISO(s); err == nil {
		return ts
	}
	return time.Time{}
}

func recentDerefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func recentDerefFloat(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
