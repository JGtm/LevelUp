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

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/classification"
	"levelup/go-api/internal/games/halo_5/ingest"
	"levelup/go-api/internal/persist"
)

// CaptureSource = source live enrichie pour la capture sync : ajoute le carnage
// (roster complet match_participants — la liste de matchs ne porte que le self).
// Séparée de h5Source (interface minimale de l'adapter read-only) pour ne pas
// élargir l'interface de l'adapter. *Client l'implémente.
type CaptureSource interface {
	h5Source
	GetMatchCarnage(ctx context.Context, matchID, mode string) (*H5CarnageResponse, error)
}

var _ CaptureSource = (*Client)(nil)

// NewCaptureSource construit une CaptureSource live depuis le SpartanToken du
// contexte (par joueur+session, comme NewSpartanTokenSource). Erreur si pas de
// token (le caller — runner de sync — dégrade en re-auth). Point d'entrée EXPORTE
// pour le câblage live hors package (livesync runner).
func NewCaptureSource(ctx context.Context) (CaptureSource, error) {
	src, err := NewSpartanTokenSource(ctx)
	if err != nil {
		return nil, err
	}
	cs, ok := src.(CaptureSource)
	if !ok {
		return nil, fmt.Errorf("h5: source live ne supporte pas la capture (GetMatchCarnage)")
	}
	return cs, nil
}

const (
	h5CaptureDefaultPageSize   = 25
	h5CaptureDefaultMaxMatches = 100
	captureSourceLabel         = "h5_capture"
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
	CarnageFailed    int  // matchs dont le carnage n'a pu être fetché (batch sans participants)
	ExcludedWarzone  int  // matchs Warzone écartés à la collecte (cf. isExcludedH5GameMode)
}

// h5GameModeWarzone est le GameMode des matchs Warzone Halo 5 (rosters 24 joueurs).
const h5GameModeWarzone = 4

// isExcludedH5GameMode : modes Halo 5 NON ingérés. Warzone (GameMode 4) est exclu —
// décision produit (pas géré côté app) ET il évite de marteler PeopleHub à la
// résolution de rosters géants (24 joueurs/match). Arena (2-équipes) reste collecté.
func isExcludedH5GameMode(gameMode int) bool {
	return gameMode == h5GameModeWarzone
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
	src CaptureSource,
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
		source = captureSourceLabel
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
			// Warzone exclu de la collecte (produit + anti-storm PeopleHub) : on saute
			// AVANT carnage/events/rosters — aucun appel coûteux pour ces matchs.
			if isExcludedH5GameMode(resp.Results[i].Id.GameMode) {
				stats.ExcludedWarzone++
				continue
			}
			timeline := captureMatchTimeline(ctx, src, s.MatchID, &stats)
			participants := captureParticipants(ctx, src, s.MatchID, h5GameModeSegment(resp.Results[i].Id.GameMode), resolveXUID, &stats)
			batches = append(batches, ingest.CollectMatchBatch(TitleSlug, source, viewer, s, timeline, participants, resolveXUID))
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

// captureParticipants fetch le carnage + mappe le roster complet. Indisponibilité
// (404/410, token expiré, decode) -> nil + CarnageFailed++ : le match est TOUT DE
// MÊME collecté (sans match_participants) — squad/rencontres dégradent pour ce
// match seul, et un futur passage le re-tentera (match_registry idempotent).
func captureParticipants(ctx context.Context, src CaptureSource, matchID, mode string, resolveXUID func(string) string, stats *CaptureStats) []domain.MatchParticipantRow {
	carnage, err := src.GetMatchCarnage(ctx, matchID, mode)
	if err != nil {
		stats.CarnageFailed++
		slog.WarnContext(ctx, "h5 capture: carnage indisponible (sans participants)", "match_id", matchID, "err", err)
		return nil
	}
	return mapCarnageParticipants(matchID, carnage, resolveXUID)
}
