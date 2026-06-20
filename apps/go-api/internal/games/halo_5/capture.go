package halo_5

// capture.go — moteur de COLLECTE du sync live Halo 5 (tranche 1).
//
// Fetch cryptum (GetPlayerMatches + GetMatchEvents) -> mappers canoniques PRIVÉS
// (mapMatchSummaries, mapH5Events) -> ingest.CollectMatchBatch -> *persist.MatchBatch.
// Vit DANS le package halo_5 pour accéder aux mappers privés + au type events privé.
//
// PUR vis-à-vis de la DB : aucune écriture ici. La persistance (SharedPersister +
// lease B-swap) + l'exposition en DeltaRunner sont la tranche suivante. Le point de
// convergence est *persist.MatchBatch — exactement ce que le chemin Infinite
// (submitMatchAsBatch) consomme — donc on réutilise tout le SharedPersister sans
// jamais toucher SyncEngine.run (Infinite byte-identique).

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/classification"
	"levelup/go-api/internal/games/halo_5/ingest"
	"levelup/go-api/internal/persist"
)

const (
	h5CaptureDefaultPageSize   = 25
	h5CaptureDefaultMaxMatches = 100
	h5CaptureSource            = "h5_capture"
)

// CaptureOptions borne la collecte + porte la stratégie de classification ranked.
type CaptureOptions struct {
	MaxMatches int                             // borne dure du nb de matchs collectés (<=0 -> défaut 100)
	PageSize   int                             // taille de page GetPlayerMatches (<=0 -> défaut 25)
	Source     string                          // libellé source du batch ("" -> "h5_capture")
	Classifier classification.RankedClassifier // verdicts ranked/PvE depuis le HopperId (peut être nil)
}

// CaptureStats résume une passe de collecte (alimente le domain.SyncResult downstream + logs).
type CaptureStats struct {
	MatchesSeen      int  // résumés parcourus
	MatchesCollected int  // batches produits (matchs nouveaux)
	StoppedOnKnown   bool // delta-stop atteint (1er match déjà connu)
	EventsFailed     int  // matchs dont la timeline n'a pu être fetchée (batch registry-only)
}

// CollectRecentMatches récupère les matchs h5 récents du viewer et retourne UN
// *persist.MatchBatch par match NOUVEAU (isKnown == false), prêt pour SharedPersister.
//
//   - src         : source live h5 (GetPlayerMatches + GetMatchEvents) ; mockable en test.
//   - viewer      : joueur consulté (self) — owner du batch + first_sync_by. Son XUID
//     doit être l'xuid Xbox RÉSOLU en amont (l'API h5 ne le fournit jamais).
//   - resolveXUID : gamertag -> xuid Xbox (CachingResolver câblé live ; "" toléré, l'identité
//     reste dans le gamertag pour le kill-feed). nil -> tout "".
//   - isKnown     : matchID -> déjà persisté (delta-stop au 1er connu). nil -> tout nouveau.
//
// Délta-stop : on s'arrête au 1er match déjà connu (miroir du known-set Infinite).
// Idempotence garantie en aval par match_registry (un re-collect d'un match déjà
// persisté est no-opé par SharedPersister).
func CollectRecentMatches(
	ctx context.Context,
	src h5Source,
	viewer canonical.PlayerIdentity,
	resolveXUID func(gamertag string) string,
	isKnown func(matchID string) bool,
	opts CaptureOptions,
) ([]*persist.MatchBatch, CaptureStats, error) {
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = h5CaptureDefaultPageSize
	}
	maxMatches := opts.MaxMatches
	if maxMatches <= 0 {
		maxMatches = h5CaptureDefaultMaxMatches
	}
	source := opts.Source
	if source == "" {
		source = h5CaptureSource
	}
	if resolveXUID == nil {
		resolveXUID = func(string) string { return "" }
	}
	if isKnown == nil {
		isKnown = func(string) bool { return false }
	}

	var (
		batches []*persist.MatchBatch
		stats   CaptureStats
	)
	seen := make(map[string]struct{}) // garde anti-boucle si l'API n'honore pas `start`
	for start := 0; stats.MatchesCollected < maxMatches; start += pageSize {
		resp, err := src.GetPlayerMatches(ctx, viewer.Gamertag, start, pageSize)
		if err != nil {
			return batches, stats, fmt.Errorf("h5 capture: GetPlayerMatches(%s, start=%d): %w", viewer.Gamertag, start, err)
		}
		summaries := mapMatchSummaries(resp, viewer.Gamertag, opts.Classifier)
		if len(summaries) == 0 {
			break // fin de l'historique
		}
		for i := range summaries {
			s := summaries[i]
			if _, dup := seen[s.MatchID]; dup {
				return batches, stats, nil // pagination non avançante -> stop défensif
			}
			seen[s.MatchID] = struct{}{}
			stats.MatchesSeen++
			if isKnown(s.MatchID) {
				stats.StoppedOnKnown = true
				return batches, stats, nil // delta-stop
			}
			timeline := captureMatchTimeline(ctx, src, s.MatchID, &stats)
			batches = append(batches, ingest.CollectMatchBatch(TitleSlug, source, viewer, s, timeline, resolveXUID))
			stats.MatchesCollected++
			if stats.MatchesCollected >= maxMatches {
				return batches, stats, nil
			}
		}
		if len(summaries) < pageSize {
			break // dernière page (page incomplète)
		}
	}
	return batches, stats, nil
}

// captureMatchTimeline fetch + mappe la timeline d'un match. Indisponibilité
// (404/410, token expiré, decode) -> timeline vide + EventsFailed++ : le match est
// TOUT DE MÊME collecté (registry seul) — on ne perd jamais le match pour un events
// KO, et un futur passage le re-tentera (match_registry idempotent).
func captureMatchTimeline(ctx context.Context, src h5Source, matchID string, stats *CaptureStats) []canonical.MatchEvent {
	resp, err := src.GetMatchEvents(ctx, matchID)
	if err != nil {
		stats.EventsFailed++
		slog.WarnContext(ctx, "h5 capture: timeline indisponible (registry seul)", "match_id", matchID, "err", err)
		return nil
	}
	return mapH5Events(resp, canonical.MatchEventOptions{}) // Types vide = tous les events
}
