// Package service - MatchHistoryService : pagination et enrichissement de l'historique.
//
// Port Go de match_history_service.py (apps/api/app/services/match_history_service.py).
//
// Le code est decoupe en fichiers thematiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient le type service,
// constants, constructor, Withers, GetPage et loadBriefingKPIs. Les autres
// responsabilites vivent dans :
//
//   - match_history_service_filters.go : filtres (sessions, ranked,
//     outcomes, tiers, explorer date/
//     types/playlists/maps/modes/squad/
//     whitelist) + sessions labels +
//     filterMatchHistoryRows global +
//     ExportCSV
//   - match_history_service_enrich.go  : toFilterMatchRow + computeMapWinRates +
//     enrichRows/enrichRow + sortItems +
//     paginate + format helpers
package service

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// Outcomes mappés selon les codes Halo Infinite.
var outcomeLabels = map[int]string{
	1: "Égalité",
	2: "Victoire",
	3: "Défaite",
	4: "Abandon",
}

// Labels de scope/contexte partagés entre filters, tri, options Explorer.
// Externalisés pour goconst (utilisés à plusieurs endroits + côté tests).
const (
	scopeKills         = "kills"
	scopeSessions      = "sessions"
	scopeRanked        = "ranked"
	scopeUnranked      = "unranked"
	scopeSolo          = "solo"
	scopeSquad         = "squad"
	expTypePVPRanked   = "PVP classé"
	expTypePVPUnranked = "PVP non classé"
	expTypePVE         = "PVE"
)

// availableSortFields est la liste des champs de tri autorisés.
var availableSortFields = []string{
	"start_time", "outcome_code", "performance_score_relative",
	"team_mmr", "delta_mmr", "win_rate_hist",
	"kda", scopeKills,
}

// availableColumns expose les colonnes disponibles dans MatchHistoryRow.
var availableColumns = []string{
	"match_id", "start_time", "outcome_label", "score_label",
	"map_ui", "mode_ui", "playlist_label",
	"team_mmr", "enemy_mmr", "delta_mmr",
	"win_rate_hist", "performance_score_relative",
	"average_life_mmss", "match_url",
}

// MatchHistoryService construit la réponse paginée pour la page historique.
type MatchHistoryService struct {
	repo           port.MatchHistoryRepository
	waypointPlayer string
	// dataAdapter (optionnel, Phase C+ multi-titres) : point d'extension pour
	// router LoadMatchSummaries via la couche canonique quand elle évoluera
	// pour porter tous les champs MatchHistoryRawRow. À ce jour, le service
	// utilise le repo direct car canonical.MatchSummary ne couvre pas encore
	// la totalité du payload (delta_mmr, performance_score, etc.). Le hook
	// est en place pour permettre une bascule incrémentale.
	dataAdapter games.TitleDataAdapter
	// playerMatchesRepo (optionnel) : utilisé pour charger les canonical rows
	// nécessaires au calcul de BriefingKPIs (alimente <SessionBriefing> en haut
	// de page). Si non câblé, BriefingKPIs reste nil → briefing absent côté front.
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
	// csrThreshold (optionnel) : callback pour résoudre le threshold de
	// placement CSR par saison (5 depuis S3, 10 avant). Si nil, fallback à 5.
	// Utilisé par applyMatchPlacements pour calculer PlacementDone/Total.
	csrThreshold CSRThresholdResolver
	// assetURL (optionnel) : adapter d'URLs d'assets du titre, utilisé pour le
	// lien vers la page publique du match (Waypoint pour Infinite). nil ou titre
	// sans page publique → pas de lien (dégradation gracieuse, F3).
	assetURL games.TitleAssetURLAdapter
	// semantic (optionnel) : adapter sémantique du titre, utilisé pour résoudre les
	// libellés d'outcome depuis outcomes.toml (source de vérité) plutôt qu'en dur
	// (F4). nil → fallback FR canonique via outcomeLabel().
	semantic games.TitleSemanticAdapter
}

// outcomeCodeToKey mappe le code outcome Halo (domain.Outcome*) vers la clé
// canonique outcomes.toml (win/loss/tie/dnf). "" si code inconnu.
var outcomeCodeToKey = map[int]string{
	domain.OutcomeDraw: "tie",
	domain.OutcomeWin:  "win",
	domain.OutcomeLoss: "loss",
	domain.OutcomeDNF:  "dnf",
}

// NewMatchHistoryService crée un MatchHistoryService.
func NewMatchHistoryService(repo port.MatchHistoryRepository, waypointPlayer string) *MatchHistoryService {
	return &MatchHistoryService{repo: repo, waypointPlayer: waypointPlayer}
}

// WithAssetURL injecte l'adapter d'URLs d'assets du titre (lien page publique du
// match). Sans injection, aucun lien n'est produit (dégradation gracieuse, F3).
func (s *MatchHistoryService) WithAssetURL(a games.TitleAssetURLAdapter) *MatchHistoryService {
	s.assetURL = a
	return s
}

// WithSemantic injecte l'adapter sémantique du titre (libellés d'outcome résolus
// depuis outcomes.toml, source de vérité). Sans injection, fallback FR canonique. F4.
func (s *MatchHistoryService) WithSemantic(a games.TitleSemanticAdapter) *MatchHistoryService {
	s.semantic = a
	return s
}

// rowFormatters construit les résolveurs title-agnostic injectés dans
// l'enrichissement d'une ligne : URL de page publique du match (F3, via l'adapter
// d'assets + gamertag) et libellé d'outcome (F4, via l'adapter sémantique). Champs
// nil si l'adapter correspondant n'est pas câblé → dégradation gracieuse.
func (s *MatchHistoryService) rowFormatters() rowFormatters {
	f := rowFormatters{}
	if s.assetURL != nil {
		gt := s.waypointPlayer
		f.matchURL = func(matchID string) string { return s.assetURL.PlayerMatchWebURL(gt, matchID) }
	}
	if s.semantic != nil {
		f.outcomeLabel = func(code int) string {
			if oc := s.semantic.Outcomes(); oc != nil {
				if m, ok := oc.Get(outcomeCodeToKey[code]); ok {
					if lbl, _ := m.Label("fr"); lbl != "" {
						return lbl
					}
				}
			}
			return outcomeLabel(code) // failsafe FR canonique
		}
	}
	return f
}

// WithDataAdapter injecte le DataAdapter multi-titres pour activer une
// future bascule LoadMatchSummaries. Dégradation gracieuse si nil.
func (s *MatchHistoryService) WithDataAdapter(a games.TitleDataAdapter) *MatchHistoryService {
	s.dataAdapter = a
	return s
}

// WithPlayerMatchesRepo injecte le loader canonical-aware utilisé pour calculer
// BriefingKPIs (composant <SessionBriefing> en haut de la page Stats).
// Dégradation gracieuse si non câblé : BriefingKPIs reste nil.
func (s *MatchHistoryService) WithPlayerMatchesRepo(repo port.PlayerMatchesRepository, titleSlug, gamertag string) *MatchHistoryService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
}

// WithCSRThresholds injecte le résolveur season_id → threshold de placement CSR.
// Sans appel, le service utilise le default (5, S3+). Côté tests : laisser nil
// pour valider le fallback ; côté prod : passer (*duckdb.CSRThresholdsRepo).Get.
func (s *MatchHistoryService) WithCSRThresholds(resolver CSRThresholdResolver) *MatchHistoryService {
	s.csrThreshold = resolver
	return s
}

// GetPage charge tous les matchs, applique filtres+pagination et retourne la réponse.
func (s *MatchHistoryService) GetPage(
	ctx context.Context,
	req domain.MatchHistoryQueryRequest,
) (domain.MatchHistoryPageResponse, error) {
	rawRows, err := s.repo.LoadAll(ctx)
	if err != nil {
		return domain.MatchHistoryPageResponse{}, fmt.Errorf("MatchHistoryService.GetPage: %w", err)
	}
	totalUnfiltered := len(rawRows)

	// Placement (X/Y) calculé sur l'ensemble brut AVANT filtrage : la
	// stratégie LUSR (10 plus anciens par chaîne) a besoin de l'ordre
	// chronologique global pour rester stable quels que soient les filtres
	// Explorer appliqués ensuite.
	applyMatchPlacements(ctx, rawRows, s.csrThreshold)

	// Exclusions manuelles — avant tout autre filtrage
	rawRows = filterExcludedRows(rawRows)

	// Filtrage (réutilise la logique pure du FiltersService)
	filtered := filterMatchHistoryRows(rawRows, req.Filters)

	// §5 plan Squad/Sessions : filtre multi-sessions solo (post-FilterContext).
	if len(req.PickedSoloSessionLabels) > 0 {
		filtered = filterMatchHistoryRowsBySoloSessions(filtered, req.PickedSoloSessionLabels)
	}

	// Capture des rows post-cascade post-PickedSoloSessions, avant tout filtre
	// Explorer (ranked/outcome/skill/perf + Explorer-cascade). C'est la base pour
	// le calcul des "available_*" cascade-aware avec sémantique OR.
	baseForExplorerOptions := filtered

	// Filtres supplémentaires (ranked context, outcome, skill tier, perf tier).
	filtered = filterByRankedContext(filtered, req.RankedContext)
	filtered = filterByOutcome(filtered, req.OutcomeFilter)
	filtered = filterBySkillTier(filtered, req.SkillTiers, req.RankedContext)
	filtered = filterByPerfTiers(filtered, req.PerfTiers)

	// Options Explorer disponibles calculées AVANT les filtres Explorer additionnels.
	availExpTypes, availPlaylists, availMaps, availModes := computeExplorerAvailableOptions(filtered)

	// Options Explorer-spécifiques avec count cascade-aware (sémantique OR au sein
	// d'une dimension, AND entre dimensions). Calculées sur baseForExplorerOptions
	// pour que chaque dimension reflète "ce qu'on aurait si on cochait X".
	availOutcomes := computeAvailableOutcomes(baseForExplorerOptions, req)
	availPerfTiers := computeAvailablePerfTiers(baseForExplorerOptions, req)
	availSkillTiers := computeAvailableSkillTiers(baseForExplorerOptions, req)
	availRankedCtxs := computeAvailableRankedContexts(baseForExplorerOptions, req)
	availSquadScopes := computeAvailableSquadScopes(baseForExplorerOptions, req)

	// Filtres Explorer additionnels (date, experience, playlist, carte, mode, squad, match ID).
	filtered = applyExplorerMatchFilters(filtered, req)

	totalScoped := len(filtered)

	// Calcul win_rate par carte sur l'histotique complet (avant filtre)
	mapWinRates := computeMapWinRates(rawRows)

	// Enrichissement
	items := enrichRows(filtered, mapWinRates, s.rowFormatters())

	// Tri
	sortItems(items, req.SortField, req.SortDir)

	// Pagination
	page, pageItems := paginate(items, req.Pagination)

	var exportHint *domain.ExportHint
	if req.IncludeExportHint && totalScoped > 0 {
		exportHint = &domain.ExportHint{
			FileName:      "levelup_matches.csv",
			EstimatedRows: totalScoped,
		}
	}

	// §5 plan Squad/Sessions : sessions dispo dérivées de l'ensemble brut
	// (avant filtrage scope) pour permettre au front de proposer le picker
	// solo/squad indépendamment du filtre période courant.
	sessionLabels := buildMatchHistorySessionLabels(rawRows)

	briefingKPIs := s.loadBriefingKPIs(ctx, filtered)

	return domain.MatchHistoryPageResponse{
		Summary: domain.MatchHistoryQuerySummary{
			TotalMatchesScoped:       totalScoped,
			TotalMatchesUnfiltered:   totalUnfiltered,
			PeriodLabel:              buildPeriodLabel(req.Filters),
			ActiveFilterMode:         req.Filters.FilterMode,
			AvailableExperienceTypes: availExpTypes,
			AvailablePlaylists:       availPlaylists,
			AvailableMaps:            availMaps,
			AvailableModes:           availModes,
			AvailableOutcomes:        availOutcomes,
			AvailablePerfTiers:       availPerfTiers,
			AvailableSkillTiers:      availSkillTiers,
			AvailableRankedContexts:  availRankedCtxs,
			AvailableSquadScopes:     availSquadScopes,
		},
		Table: domain.MatchHistoryTable{
			Items:      pageItems,
			Pagination: page,
		},
		AvailableSortFields: availableSortFields,
		AvailableColumns:    availableColumns,
		ExportHint:          exportHint,
		SessionLabels:       sessionLabels,
		BriefingKPIs:        briefingKPIs,
	}, nil
}

// loadBriefingKPIs charge les canonical rows + filtre par match_ids du scope filtré
// puis calcule les KPIs. Best-effort : nil si playerMatchesRepo absent ou échec.
func (s *MatchHistoryService) loadBriefingKPIs(
	ctx context.Context, filtered []domain.MatchHistoryRawRow,
) *domain.KPIStats {
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" || len(filtered) == 0 {
		return nil
	}
	canonicalRows, cerr := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if cerr != nil {
		slog.WarnContext(ctx, "match_history.briefing_kpis.load_failed", "err", cerr)
		return nil
	}
	keep := make(map[string]struct{}, len(filtered))
	for _, r := range filtered {
		keep[r.MatchID] = struct{}{}
	}
	scoped := make([]canonical.PlayerMatchRow, 0, len(filtered))
	for _, c := range canonicalRows {
		if _, ok := keep[c.Summary.MatchID]; ok {
			scoped = append(scoped, c)
		}
	}
	if len(scoped) == 0 {
		return nil
	}
	kpis := analysis.ComputeKPIStats(scoped, games.EffectiveHpToKill(s.titleSlug))
	return &kpis
}

// filterMatchHistoryRowsBySoloSessions garde les rows dont SessionLabel
// figure dans labels et qui sont des sessions solo (IsWithFriends=FALSE).
// Les rows sans label sont exclues. Comparaison case-sensitive (les labels
// sont des identifiants normalisés côté ingestion).
