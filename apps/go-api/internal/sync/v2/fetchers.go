// Package v2 — fetchers.go : adapters V2-native pour MatchListProvider +
// SharedMatchFetcher + PlayerEnrichmentFetcher (D6.3 du plan ADR 0027).
//
// Stratégie : V2 importe uniquement les TYPES exportés du package sync
// (MatchHistoryEntry, MatchSkillData) — pas la logique V1. Les adapters
// font les appels HTTP via une interface HaloClient narrow définie ici,
// implémentée en runtime par sync.PooledHaloClient et en test par un mock.
//
// Cette approche conserve la règle "V1 untouched" tout en évitant la
// duplication des types complexes (CSRRankSnapshot, etc.). Pour D8
// (cleanup V1), il faudra déplacer ces types vers internal/halo/types/
// ou les dupliquer.
package v2

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/historyretry"
)

// HaloClient est l'interface narrow utilisée par les adapters V2. Sous-ensemble
// de sync.HaloClient — seulement les méthodes appelées en Phase 1/3.
// Permet aux tests d'utiliser un mock local sans dépendre de
// halo_client_mock_test.go (qui est dans le package sync).
//
// T2 : ajout GetHighlightEventsChunk pour fetcher les chunks film en
// Phase 3 et préserver la parité V1↔V2 (V1 fetche les highlights inline
// dans fetchMatchData).
type HaloClient interface {
	GetMatchHistory(ctx context.Context, gamertag, matchType string, start, count int) ([]syncpkg.MatchHistoryEntry, error)
	GetMatchStats(ctx context.Context, matchID string) (map[string]any, error)
	GetMatchSkill(ctx context.Context, matchID string, xuids []string) (map[string]*syncpkg.MatchSkillData, error)
	// GetHighlightEventsChunk retourne (data, filmMajorVersion, found, err).
	// found=false (sans erreur) si le film est absent (404/410) — cas normal
	// pour les vieux matchs. Best-effort : l'échec ne tue pas la Phase 3.
	GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error)
}

// HaloClientFactory construit un HaloClient pinné sur un joueur (token
// du joueur via pool.Pool). Runtime : wrap sync.NewPooledHaloClient. Test :
// retourne un mock.
type HaloClientFactory func(gamertag, xuid string) HaloClient

// ─── MatchListProvider ────────────────────────────────────────────────

// matchListProviderV2 implémente MatchListProvider via GetMatchHistory.
type matchListProviderV2 struct {
	clientFactory HaloClientFactory
	matchType     string // "matchmaking" | "all" | "custom" | "local"
	pageSize      int    // typique 25 (cf. V1 historyPageSize)
	maxPages      int    // safety : stop après N pages si jamais aucun connu (typique 20)
	// catchUp : un jeton, pris pendant la lecture de rattrapage d'un joueur (historyCatchUp) : les rattrapages
	// d'un cycle se lisent un joueur à la fois, les deltas restent parallèles.
	catchUp chan struct{}
}

// NewMatchListProvider construit un MatchListProvider prêt à être injecté
// dans le CycleOrchestrator.
//
// Defaults appliqués (si 0) :
//   - matchType = "matchmaking"
//   - pageSize  = 25
//   - maxPages  = 20 (garde-rail : stop après 500 matchs si bug)
func NewMatchListProvider(factory HaloClientFactory, matchType string, pageSize, maxPages int) MatchListProvider {
	if matchType == "" {
		matchType = "matchmaking"
	}
	if pageSize <= 0 {
		pageSize = 25
	}
	if maxPages <= 0 {
		maxPages = 20
	}
	return &matchListProviderV2{
		clientFactory: factory,
		matchType:     matchType,
		pageSize:      pageSize,
		maxPages:      maxPages,
		catchUp:       make(chan struct{}, 1),
	}
}

// ListUnknownMatches liste les matchs inconnus d'un joueur, en ordre de l'API. Format URL
// `xuid(NNN)` requis (anti-régression incident mai 2026 — gamertag brut → réponse stale).
//
// Deux modes, selon l'état de l'historique du joueur (hist) :
//   - complet : delta. Pagination en ordre reverse-chronologique, arrêt après
//     domain.DeltaKnownRun matchs connus d'affilée (ordre API préservé), au plus maxPages pages
//     (garde-rail : V1 n'en a pas, peut paginer 1000 pages en cas de known set vide) ;
//   - incomplet : le delta d'abord, puis le rattrapage (fetchers_history.go), qui saute les connus au lieu
//     de s'arrêter dessus et reprend au marque-page ; une erreur de lecture du rattrapage ne retire pas les
//     nouveaux matchs du delta.
//
// Chaque lecture de page passe par historyretry.Page (rejeu borné des 429 et du parc en pause).
func (p *matchListProviderV2) ListUnknownMatches(ctx context.Context, prof PlayerProfile, known map[string]bool, hist HistoryState) (Listing, error) {
	client := p.clientFactory(prof.Gamertag, prof.XUID)
	if client == nil {
		return Listing{}, fmt.Errorf("HaloClient nil pour %s", prof.Gamertag)
	}

	if hist.Incomplete {
		return p.listWithCatchUp(ctx, client, prof, known, hist)
	}
	top, err := p.listDelta(ctx, client, deltaArgs{prof: prof, known: known, skip: hist.skipped()})
	if err != nil {
		return Listing{}, err
	}
	unknown := top.unknown
	if hist.Deferred && len(unknown) > catchUpMinBudget {
		unknown = unknown[:catchUpMinBudget] // rattrapage reporté : un delta, pas un joueur neuf à 500 matchs en mémoire
	}
	return Listing{Unknown: unknown}, nil
}

// deltaArgs est ce que lit listDelta : le joueur, l'ensemble connu et les matchs à ne plus demander
// (historystate.SkipSetKey).
type deltaArgs struct {
	prof  PlayerProfile
	known map[string]bool
	skip  map[string]bool
	// limit : nombre d'inconnus à partir duquel la lecture s'arrête ; 0 = aucune borne (maxPages seul).
	limit int
}

// deltaRead est le résultat de listDelta.
type deltaRead struct {
	unknown []string
	// closed : la lecture s'est arrêtée sur domain.DeltaKnownRun connus d'affilée, donc les inconnus lus
	// touchent la zone déjà connue : ce sont des matchs neufs, pas des matchs plus anciens du rattrapage.
	closed bool
}

// listDelta pagine /matches et s'arrête après domain.DeltaKnownRun matchs connus d'affilée (les
// inconnus rencontrés avant, y compris entre deux connus, sont pris : un match rangé par un
// coéquipier ne cache pas les inconnus qui le suivent), à la fin de la liste, après maxPages pages ou
// à a.limit inconnus. Un match de `skip` (introuvable, abandonné : historystate.SkipSetKey) est neutre,
// comme une entrée écartée de Halo 5 : il n'est pas demandé, n'allonge pas la suite de connus et ne la
// remet pas à zéro. Chaque page passe par historyretry.Page.
// Format `xuid(NNN)` requis (anti-régression incident mai 2026 : gamertag brut, réponse périmée).
func (p *matchListProviderV2) listDelta(ctx context.Context, client HaloClient, a deltaArgs) (deltaRead, error) {
	gamertag, histArg := a.prof.Gamertag, fmt.Sprintf("xuid(%s)", a.prof.XUID)
	out := deltaRead{unknown: make([]string, 0, p.pageSize)}
	run := 0 // matchs connus d'affilée, jusqu'à l'entrée courante
	for page := 0; page < p.maxPages; page++ {
		start := page * p.pageSize
		entries, err := historyretry.Page(ctx, gamertag, start, func() ([]syncpkg.MatchHistoryEntry, error) {
			return client.GetMatchHistory(ctx, histArg, p.matchType, start, p.pageSize)
		})
		if err != nil {
			return deltaRead{}, fmt.Errorf("GetMatchHistory page %d for %s: %w", page, gamertag, err)
		}
		if len(entries) == 0 {
			break // fin de l'historique
		}
		for _, e := range entries {
			if a.known[e.MatchID] {
				if run++; run >= domain.DeltaKnownRun {
					break
				}
				continue
			}
			if a.skip[e.MatchID] {
				continue
			}
			run = 0
			out.unknown = append(out.unknown, e.MatchID)
		}
		if run >= domain.DeltaKnownRun {
			out.closed = true
			break
		}
		if a.limit > 0 && len(out.unknown) >= a.limit {
			break
		}
		// Page partielle = fin de l'historique ; page pleine : on continue à la suivante.
		if len(entries) < p.pageSize {
			break
		}
	}
	slog.DebugContext(ctx, "v2 ListUnknownMatches",
		"gamertag", gamertag, "unknown_count", len(out.unknown))
	return out, nil
}

// ─── SharedMatchFetcher ───────────────────────────────────────────────

// sharedMatchFetcherV2 implémente SharedMatchFetcher via GetMatchStats +
// GetMatchSkill. Token du fetcher canonical (cf. Phase 2).
type sharedMatchFetcherV2 struct {
	clientFactory HaloClientFactory
}

// NewSharedMatchFetcher construit un SharedMatchFetcher prêt à être injecté.
func NewSharedMatchFetcher(factory HaloClientFactory) SharedMatchFetcher {
	return &sharedMatchFetcherV2{clientFactory: factory}
}

// FetchSharedMatch enchaîne GetMatchStats + GetMatchSkill + GetHighlightEventsChunk
// avec le token du canonical fetcher.
//
// Best-effort sur GetMatchSkill et GetHighlightEventsChunk : si l'endpoint
// répond une erreur, on continue avec le champ correspondant à nil
// (V1-compatible — V1 marque SkillError/HighlightError dans fetchedMatch
// mais continue le batch).
func (f *sharedMatchFetcherV2) FetchSharedMatch(
	ctx context.Context,
	matchID string,
	fetcher PlayerProfile,
	participants []PlayerProfile,
) (SharedMatchData, error) {
	client := f.clientFactory(fetcher.Gamertag, fetcher.XUID)
	if client == nil {
		return SharedMatchData{}, fmt.Errorf("HaloClient nil pour fetcher %s", fetcher.Gamertag)
	}

	stats, err := client.GetMatchStats(ctx, matchID)
	if err != nil {
		return SharedMatchData{}, fmt.Errorf("GetMatchStats: %w", err)
	}

	// Skill : prendre les xuids des participants tracked uniquement.
	xuids := make([]string, 0, len(participants))
	for _, p := range participants {
		if p.XUID != "" {
			xuids = append(xuids, p.XUID)
		}
	}
	var skillMap map[string]any
	if len(xuids) > 0 {
		skill, skillErr := client.GetMatchSkill(ctx, matchID, xuids)
		if skillErr != nil {
			// Best-effort : log + continue avec Skill nil. V1 fait pareil
			// (engine_skill_heal.go absorbe les erreurs sur les vieux matchs).
			// Warn : sans skill, is_ranked et le CSR ne sont pas posés (règle D-10) ; le match
			// reste orphelin jusqu'au rattrapage (backfill --shared-csr puis backfill-is-ranked).
			slog.WarnContext(ctx, "v2 GetMatchSkill failed — continuing with Skill nil",
				"match_id", matchID, "err", skillErr)
		} else {
			// Conversion map[string]*MatchSkillData → map[string]any pour
			// rester compatible avec le shape opaque de SharedMatchData.Skill.
			skillMap = make(map[string]any, len(skill))
			for xuid, sd := range skill {
				skillMap[xuid] = sd
			}
		}
	}

	// T2 — Highlight events chunk : best-effort (vieux matchs renvoient 404/410).
	// V1 fait ce fetch inline dans fetchMatchData (engine_fetch.go:134-143).
	// Sans ce fetch en Phase 3 V2, les highlight_events sont insérés avec 1
	// cycle de retard via healEventsForRecentMatches.
	highlightData, filmMajorVer, hasHighlights, hlErr := client.GetHighlightEventsChunk(ctx, matchID)
	if hlErr != nil {
		slog.DebugContext(ctx, "v2 GetHighlightEventsChunk failed — continuing without highlights",
			"match_id", matchID, "err", hlErr)
		highlightData = nil
		filmMajorVer = 0
		hasHighlights = false
	}

	return SharedMatchData{
		MatchID:        matchID,
		Fetcher:        fetcher.PlayerSlug,
		Stats:          stats,
		Skill:          skillMap,
		HighlightChunk: highlightData,
		FilmMajorVer:   filmMajorVer,
		HasHighlights:  hasHighlights,
	}, nil
}

// ─── PlayerEnrichmentFetcher ──────────────────────────────────────────

// playerEnrichmentFetcherV2 implémente PlayerEnrichmentFetcher.
//
// Pour D6.3 : implémentation no-op. Pendant le main sync, V1 ne fait aucun
// appel API per-player (PersonalScores extraction = transformation des stats
// JSON, pas un appel HTTP — délégué à Phase 5 dans D6.4). Cette interface
// reste exposée pour les futures extensions (CSR snapshots déplacés du
// post-sync, achievements sync per-cycle, etc.) — le no-op permet de
// satisfaire l'orchestrator sans bouger l'interface.
type playerEnrichmentFetcherV2 struct{}

// NewPlayerEnrichmentFetcher construit le fetcher per-player no-op.
func NewPlayerEnrichmentFetcher() PlayerEnrichmentFetcher {
	return &playerEnrichmentFetcherV2{}
}

// FetchPlayerEnrichment retourne un PlayerEnrichmentData vide (no-op D6.3).
// Phase 5 (persister) extrait personal_score_awards depuis SharedMatchData.Stats
// au moment de la construction du méga-batch.
func (f *playerEnrichmentFetcherV2) FetchPlayerEnrichment(
	_ context.Context,
	p PlayerProfile,
	matchID string,
) (PlayerEnrichmentData, error) {
	return PlayerEnrichmentData{
		PlayerSlug: p.PlayerSlug,
		MatchID:    matchID,
		Data:       nil, // no per-player API call needed during main sync
	}, nil
}
