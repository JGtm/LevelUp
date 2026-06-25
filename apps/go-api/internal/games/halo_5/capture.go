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

// AppearanceCapableSource = source live exposant les endpoints PROFIL (appearance +
// rendu Spartan + emblème). Séparée de CaptureSource (sync de matchs) : l'identité
// Spartan est un flux distinct, consommé par le hook appearance du Home. *Client
// l'implémente. Définie ici pour fournir NewAppearanceSource (point d'entrée câblage
// hors package, miroir de NewCaptureSource).
type AppearanceCapableSource interface {
	GetAppearance(ctx context.Context, gamertag string) (*H5Appearance, error)
	GetSpartanRenderPNG(ctx context.Context, gamertag string) ([]byte, string, error)
	GetEmblemPNG(ctx context.Context, gamertag string) ([]byte, string, error)
}

var _ AppearanceCapableSource = (*Client)(nil)

// NewAppearanceSource construit une AppearanceCapableSource live depuis le
// SpartanToken du contexte (par joueur+session, comme NewSpartanTokenSource). Erreur
// si pas de token. Point d'entrée EXPORTÉ pour le câblage live hors package (CLI
// backfill appearance / hook live).
func NewAppearanceSource(ctx context.Context) (AppearanceCapableSource, error) {
	src, err := NewSpartanTokenSource(ctx)
	if err != nil {
		return nil, err
	}
	as, ok := src.(AppearanceCapableSource)
	if !ok {
		return nil, fmt.Errorf("h5: source live ne supporte pas l'appearance (profils)")
	}
	return as, nil
}

const (
	h5CaptureDefaultPageSize   = 25
	h5CaptureDefaultMaxMatches = 100
	captureSourceLabel         = "h5_capture"
)

// CaptureOptions borne la collecte + porte la stratégie de classification ranked.
type CaptureOptions struct {
	MaxMatches  int                             // borne dure du nb de matchs collectés (<=0 -> défaut 100)
	PageSize    int                             // taille de page GetPlayerMatches (<=0 -> défaut 25)
	Source      string                          // libellé source du batch ("" -> "h5_capture")
	Classifier  classification.RankedClassifier // verdicts ranked/PvE depuis le HopperId (peut être nil)
	StopOnKnown bool                            // delta-stop au 1er match connu (sync live) ; false = skip-known SANS stop (backfill profond)
}

// CaptureStats résume une passe de collecte (alimente le domain.SyncResult downstream + logs).
type CaptureStats struct {
	MatchesSeen      int  // résumés parcourus
	MatchesCollected int  // batches produits (matchs nouveaux)
	MatchesSkipped   int  // matchs déjà connus sautés (backfill : skip-known SANS stop)
	StoppedOnKnown   bool // delta-stop atteint (1er match déjà connu, mode StopOnKnown)
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
	// Sync live = delta-stop au 1er connu (préserve le comportement historique).
	opts.StopOnKnown = true
	resolveXUID = resolveXUIDOrEmpty(resolveXUID)
	isKnown = isKnownOrFalse(isKnown)

	var (
		batches []*persist.MatchBatch
		stats   CaptureStats
	)
	seen := make(map[string]struct{}) // garde anti-boucle si l'API n'honore pas `start`
	for start := 0; stats.MatchesCollected < maxMatches; start += pageSize {
		pageBatches, hasMore, err := capturePage(ctx, src, viewer, resolveXUID, isKnown, opts, start, pageSize, maxMatches, seen, &stats)
		batches = append(batches, pageBatches...)
		if err != nil || !hasMore {
			return batches, stats, err
		}
	}
	return batches, stats, nil
}

// CapturePageAt capture UNE page de l'historique h5 à un offset donné (backfill
// incrémental/résumable). Contrairement à CollectRecentMatches (delta-stop au 1er
// connu), cette fonction SAUTE les matchs déjà connus (isKnown == true) mais
// continue à les parcourir — le caller paginne en profondeur sur tout l'historique.
//
//   - start    : offset GetPlayerMatches (0, pageSize, 2*pageSize, ...).
//   - pageSize : taille de page (<=0 -> défaut 25).
//   - isKnown  : matchID -> déjà persisté (sauté, stats.MatchesSkipped++). nil -> rien de connu.
//
// Retourne les batches NOUVEAUX de la page + hasMore (false = fin d'historique : page
// vide OU page incomplète) + stats cumulées (le caller passe le MÊME *CaptureStats sur
// toutes les pages). seen est l'anti-boucle PARTAGÉ entre pages (un match revu →
// hasMore=false, stop défensif). opts.MaxMatches/StopOnKnown sont ignorés ici (pas de
// borne dure, pas de delta-stop : le backfill veut TOUT l'historique).
func CapturePageAt(
	ctx context.Context,
	src CaptureSource,
	viewer canonical.PlayerIdentity,
	resolveXUID func(gamertag string) string,
	isKnown func(matchID string) bool,
	opts CaptureOptions,
	start, pageSize int,
	seen map[string]struct{},
	stats *CaptureStats,
) (batches []*persist.MatchBatch, hasMore bool, err error) {
	if pageSize <= 0 {
		pageSize = h5CaptureDefaultPageSize
	}
	opts.StopOnKnown = false // backfill : skip-known SANS delta-stop
	resolveXUID = resolveXUIDOrEmpty(resolveXUID)
	isKnown = isKnownOrFalse(isKnown)
	if seen == nil {
		seen = make(map[string]struct{})
	}
	// maxMatches = 0 → pas de cap dur (le backfill borne par pages, pas par matchs).
	return capturePage(ctx, src, viewer, resolveXUID, isKnown, opts, start, pageSize, 0, seen, stats)
}

// capturePage fetch + traite UNE page de l'historique à l'offset start. Cœur partagé
// par CollectRecentMatches (live, StopOnKnown) et CapturePageAt (backfill, skip-known).
// Met à jour stats + seen en place. hasMore=false signale l'arrêt (fin d'historique,
// delta-stop, cap MaxMatches atteint, ou pagination non avançante).
func capturePage(
	ctx context.Context,
	src CaptureSource,
	viewer canonical.PlayerIdentity,
	resolveXUID func(gamertag string) string,
	isKnown func(matchID string) bool,
	opts CaptureOptions,
	start, pageSize, maxMatches int,
	seen map[string]struct{},
	stats *CaptureStats,
) (batches []*persist.MatchBatch, hasMore bool, err error) {
	source := opts.Source
	if source == "" {
		source = captureSourceLabel
	}
	resp, err := src.GetPlayerMatches(ctx, viewer.Gamertag, start, pageSize)
	if err != nil {
		return nil, false, fmt.Errorf("h5 capture: GetPlayerMatches(%s, start=%d): %w", viewer.Gamertag, start, err)
	}
	summaries := mapMatchSummaries(resp, viewer.Gamertag, opts.Classifier)
	if len(summaries) == 0 {
		return nil, false, nil // fin de l'historique
	}
	for i := range summaries {
		s := summaries[i]
		if _, dup := seen[s.MatchID]; dup {
			return batches, false, nil // pagination non avançante -> stop défensif
		}
		seen[s.MatchID] = struct{}{}
		stats.MatchesSeen++
		if isKnown(s.MatchID) {
			if opts.StopOnKnown {
				stats.StoppedOnKnown = true
				return batches, false, nil // delta-stop (live)
			}
			stats.MatchesSkipped++
			continue // backfill : sauter le connu, mais continuer à paginer plus profond
		}
		// Warzone exclu de la collecte (produit + anti-storm PeopleHub) : on saute
		// AVANT carnage/events/rosters — aucun appel coûteux pour ces matchs.
		if isExcludedH5GameMode(resp.Results[i].Id.GameMode) {
			stats.ExcludedWarzone++
			continue
		}
		timeline := captureMatchTimeline(ctx, src, s.MatchID, stats)
		participants, commendations, team0, team1 := captureParticipants(ctx, src, s.MatchID, h5GameModeSegment(resp.Results[i].Id.GameMode), resolveXUID, stats)
		batches = append(batches, ingest.CollectMatchBatch(TitleSlug, source, viewer, s, timeline, participants, commendations, team0, team1, resolveXUID))
		stats.MatchesCollected++
		if maxMatches > 0 && stats.MatchesCollected >= maxMatches {
			return batches, false, nil // cap MaxMatches (live)
		}
	}
	if len(summaries) < pageSize {
		return batches, false, nil // dernière page (page incomplète)
	}
	return batches, true, nil // page pleine -> il reste probablement des pages
}

// resolveXUIDOrEmpty garantit un resolver non-nil (tout "" si nil).
func resolveXUIDOrEmpty(f func(string) string) func(string) string {
	if f == nil {
		return func(string) string { return "" }
	}
	return f
}

// isKnownOrFalse garantit un prédicat isKnown non-nil (rien de connu si nil).
func isKnownOrFalse(f func(string) bool) func(string) bool {
	if f == nil {
		return func(string) bool { return false }
	}
	return f
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

// captureParticipants fetch le carnage UNE FOIS + mappe le roster complet ET les
// commendations natives (AXE B) depuis ce MÊME carnage (pas de 2e fetch).
// Indisponibilité (404/410, token expiré, decode) -> (nil, nil) + CarnageFailed++ :
// le match est TOUT DE MÊME collecté (sans participants ni commendations) — squad/
// rencontres dégradent pour ce match seul, et un futur passage le re-tentera
// (match_registry idempotent).
func captureParticipants(ctx context.Context, src CaptureSource, matchID, mode string, resolveXUID func(string) string, stats *CaptureStats) ([]domain.MatchParticipantRow, []persist.CommendationInsert, *int, *int) {
	carnage, err := src.GetMatchCarnage(ctx, matchID, mode)
	if err != nil {
		stats.CarnageFailed++
		slog.WarnContext(ctx, "h5 capture: carnage indisponible (sans participants)", "match_id", matchID, "err", err)
		return nil, nil, nil, nil
	}
	team0, team1 := carnageTeamScores(carnage)
	return mapCarnageParticipants(matchID, carnage, resolveXUID), mapCarnageCommendations(matchID, carnage, resolveXUID), team0, team1
}

// carnageTeamScores extrait les scores objectif d'équipe (TeamStats[].Score = score
// du mode, captures de drapeau / zones incluses) en *int (team 0, team 1) pour
// registry.Team{0,1}Score. nil si < 2 équipes (FFA / carnage vide) — pas de score
// d'équipe 2-camps à persister.
func carnageTeamScores(c *H5CarnageResponse) (*int, *int) {
	if c == nil {
		return nil, nil
	}
	t0, t1 := -1, -1
	for i := range c.TeamStats {
		switch c.TeamStats[i].TeamId {
		case 0:
			t0 = c.TeamStats[i].Score
		case 1:
			t1 = c.TeamStats[i].Score
		}
	}
	if t0 < 0 || t1 < 0 {
		return nil, nil
	}
	return &t0, &t1
}
