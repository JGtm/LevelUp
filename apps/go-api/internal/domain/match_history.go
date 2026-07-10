// Package domain — types de l'historique des parties.
package domain

import (
	"fmt"
	"time"
)

// MatchHistoryRawRow est le type de transfert entre platform/duckdb et les services.
type MatchHistoryRawRow struct {
	MatchID        string
	StartTime      *time.Time
	MapName        *string
	MapNameFR      *string
	PairName       *string
	PairNameFR     *string
	PlaylistName   *string // FR si dispo (COALESCE), sinon EN
	PlaylistNameEN *string // EN brut (pour comparer EN==FR et déclencher translate)
	MapID          *string // UUID asset, clé de lookup asset_translations
	PairID         *string // UUID asset
	PlaylistID     *string // UUID asset
	// GameVariant* : source de mode ALTERNATIVE au pair_name. Halo 5 n'a pas de
	// pair_name (PairMode nil au mapping) ; son mode vient du GameBaseVariantId
	// (game_variant_id). Le nom EN/registry est souvent NULL → résolu depuis
	// metadata.asset_translations (asset_type='game_variant'). Sert de FALLBACK au
	// mode_ui quand le pair est absent (cf. analysis.modeLabels, parité home).
	GameVariantID      *string
	GameVariantName    *string // EN (registry ou asset_translations) — souvent NULL en H5
	GameVariantNameFR  *string // FR résolu depuis asset_translations
	SeasonID           *string // ex. "CsrSeason13-1" — utilisé pour résoudre le threshold de placement CSR (5 ou 10 selon saison)
	IsFirefight        bool
	IsRanked           bool
	SessionID          *string
	SessionLabel       *string
	IsWithFriends      bool
	Outcome            int
	TeamMMR            *float64
	EnemyMMR           *float64
	Kills              int
	Deaths             int
	Assists            int
	KDA                *float64
	Accuracy           *float64
	PersonalScore      *int
	AverageLifeSeconds *float64
	TimePlayedSeconds  *int
	IsExcluded         bool
	PerformanceScore   *float64
	SkillTier          *string // e.g. "Diamond" (EN, clé de filtre)
	SkillTierFR        *string // e.g. "Diamant" (affichage FR)
	SkillRatingType    *string // "LUSR" | "CSR"
	SkillTierLabel     *string // e.g. "Diamant IV" (label formaté DB)
	// SkillExpectedWinProb : proba de victoire pré-match ∈ [0,1] (LUSR v2), lue
	// depuis match_skill_rank.expected_win_prob. Nil si pré-v2 / non disponible.
	SkillExpectedWinProb *float64
	// PlacementDone/PlacementTotal : progression dans la phase de placement.
	// CSR : parsé depuis SkillTierLabel "Placement (N restants)" + threshold via csr_placement_thresholds.
	// LUSR : 10 plus anciens matchs sans LUSR par chaîne (arena_slayer/arena_objectif/btb/chaos).
	// Tous deux nil si le match n'est pas en placement.
	PlacementDone  *int
	PlacementTotal *int
	MyTeamScore    *int // score de l'équipe du joueur (depuis team_id)
	EnemyTeamScore *int // score de l'équipe adverse
	// DominanceFlag : 0=none, 1=domination, 2=humiliation, 3=remontada,
	// 4=débandade, 5=contre-remontada (cf. canonical.DominanceFlag).
	// Peuplé par sync.BackfillDominanceFlags via engine.RunBackfillComebackBadges.
	DominanceFlag int
}

// PaginationRequest représente les paramètres de pagination d'une requête.
type PaginationRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// PaginationMeta représente les métadonnées de pagination dans une réponse.
type PaginationMeta struct {
	Total    int  `json:"total"`
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
	HasNext  bool `json:"has_next"`
	HasPrev  bool `json:"has_prev"`
}

// MatchHistoryRow représente une ligne dans la table historique des parties.
type MatchHistoryRow struct {
	MatchID                  string    `json:"match_id"`
	StartTime                time.Time `json:"start_time"`
	StartTimeLabel           string    `json:"start_time_label"`
	OutcomeCode              int       `json:"outcome_code"`
	OutcomeLabel             string    `json:"outcome_label"`
	ScoreLabel               string    `json:"score_label"`
	MapUI                    *string   `json:"map_ui"`
	ModeUI                   *string   `json:"mode_ui"`
	PlaylistLabel            *string   `json:"playlist_label"`
	TeamMMR                  *float64  `json:"team_mmr"`
	EnemyMMR                 *float64  `json:"enemy_mmr"`
	DeltaMMR                 *float64  `json:"delta_mmr"`
	WinRateHist              *float64  `json:"win_rate_hist"`
	WinRateHistTotal         *int      `json:"win_rate_hist_total"`
	PerformanceScoreRelative *int      `json:"performance_score_relative"`
	PerfTier                 int       `json:"perf_tier,omitempty"` // 1-5 ; 0 si score absent
	KDA                      *float64  `json:"kda,omitempty"`
	Kills                    int       `json:"kills,omitempty"`
	Deaths                   int       `json:"deaths,omitempty"`
	Assists                  int       `json:"assists,omitempty"`
	SkillTierLabel           *string   `json:"skill_tier_label,omitempty"`  // "Diamant IV" ou nil
	SkillRatingType          *string   `json:"skill_rating_type,omitempty"` // "CSR" | "LUSR" | nil
	// ExpectedWinProb : proba de victoire pré-match de l'équipe ∈ [0,1] (LUSR v2).
	// Alimente la colonne « Prob. vic. ». Nil si pré-v2 / non disponible.
	ExpectedWinProb *float64 `json:"expected_win_prob,omitempty"`
	// PlacementDone/PlacementTotal : phase de placement (X/Y).
	// CSR : remaining parsé de "Placement (N restants)" + threshold csr_placement_thresholds.
	// LUSR : rang chronologique parmi les 10 plus anciens matchs sans LUSR de la chaîne.
	PlacementDone       *int   `json:"placement_done,omitempty"`
	PlacementTotal      *int   `json:"placement_total,omitempty"`
	AverageLifeMMSS     string `json:"average_life_mmss"`
	DurationSeconds     *int   `json:"duration_seconds,omitempty"`
	MatchURL            string `json:"match_url"`
	IsExcluded          bool   `json:"is_excluded"`
	IsWithFriends       bool   `json:"is_with_friends"`
	ExperienceTypeLabel string `json:"experience_type_label,omitempty"`
	// DominanceFlag : 0=none, 1=domination, 2=humiliation, 3=remontada,
	// 4=débandade, 5=contre-remontada. Le front résout le label via
	// narrative.dominance.* (manifest match_view) et affiche "-" pour 0.
	DominanceFlag int `json:"dominance_flag,omitempty"`
}

// MatchHistoryQuerySummary est le résumé de la requête historique.
type MatchHistoryQuerySummary struct {
	TotalMatchesScoped     int     `json:"total_matches_scoped"`
	TotalMatchesUnfiltered int     `json:"total_matches_unfiltered"`
	PeriodLabel            *string `json:"period_label"`
	ActiveFilterMode       string  `json:"active_filter_mode"`
	// Options disponibles pour les filtres Explorer (valeurs distinctes triées).
	// AvailableExperienceTypes : LABEL localisé (locale de requête), VALUE FR
	// canonique = clé de filtre intacte (GH6-1, miroir GH5-2 Omnibar).
	AvailableExperienceTypes []LabelValue `json:"available_experience_types,omitempty"`
	AvailablePlaylists       []string     `json:"available_playlists,omitempty"`
	AvailableMaps            []string     `json:"available_maps,omitempty"`
	AvailableModes           []string     `json:"available_modes,omitempty"`
	// Options Explorer-spécifiques avec count cascade-aware (sémantique OR au sein
	// d'une dimension, AND entre dimensions). Permettent au front de griser les
	// valeurs à count=0 et d'afficher le compte par option.
	AvailableOutcomes       []LabelValue `json:"available_outcomes,omitempty"`
	AvailablePerfTiers      []LabelValue `json:"available_perf_tiers,omitempty"`
	AvailableSkillTiers     []LabelValue `json:"available_skill_tiers,omitempty"`
	AvailableRankedContexts []LabelValue `json:"available_ranked_contexts,omitempty"`
	AvailableSquadScopes    []LabelValue `json:"available_squad_scopes,omitempty"`
}

// MatchHistoryTable est la table paginée de l'historique.
type MatchHistoryTable struct {
	Items      []MatchHistoryRow `json:"items"`
	Pagination PaginationMeta    `json:"pagination"`
	Freshness  *string           `json:"freshness"`
}

// MatchHistoryQueryRequest est le corps de POST match-history/query.
type MatchHistoryQueryRequest struct {
	Filters           FilterContextInput `json:"filters"`
	Pagination        PaginationRequest  `json:"pagination"`
	SortField         string             `json:"sort_field"`
	SortDir           string             `json:"sort_dir"`
	IncludeExportHint bool               `json:"include_export_hint"`
	// Columns permet au client de préciser les colonnes souhaitées dans la réponse.
	// Nil/vide = toutes les colonnes disponibles (comportement par défaut).
	Columns []string `json:"columns,omitempty"`
	// §5 plan Squad/Sessions : filtre multi-sessions solo. Liste vide = pas
	// de filtre. Filtré côté service (post-LoadAll, avant pagination).
	PickedSoloSessionLabels []string `json:"picked_solo_session_labels,omitempty"`
	// PerfTiers filtre par palier de performance (1=Excellent … 5=Mauvais).
	// Liste vide = pas de filtre. Valeurs hors [1-5] ignorées.
	PerfTiers []int `json:"perf_tiers,omitempty"`
	// SkillTiers filtre par tier ranked/LUSR ("Bronze","Silver","Gold","Platinum","Diamond","Onyx").
	// Nécessite RankedContext non vide pour éviter le mélange CSR/LUSR.
	SkillTiers []string `json:"skill_tiers,omitempty"`
	// RankedContext restreint aux matchs ranked ("ranked") ou non-ranked ("unranked").
	// Valeur vide = pas de filtre. Requis pour activer SkillTiers.
	RankedContext string `json:"ranked_context,omitempty"`
	// OutcomeFilter liste les codes de résultats acceptés (1=Égalité, 2=Victoire, 3=Défaite, 4=Abandon).
	// Liste vide = pas de filtre.
	OutcomeFilter []int `json:"outcome_filter,omitempty"`
	// Filtres Explorer additionnels (no-op pour match-history classique si non renseignés).
	MatchStartDate  *time.Time `json:"match_start_date,omitempty"`
	MatchEndDate    *time.Time `json:"match_end_date,omitempty"`
	ExperienceTypes []string   `json:"experience_types,omitempty"`
	Playlists       []string   `json:"playlists,omitempty"`
	MapNames        []string   `json:"map_names,omitempty"`
	ModeNames       []string   `json:"mode_names,omitempty"`
	SquadScope      string     `json:"squad_scope,omitempty"`
	MatchIDSearch   string     `json:"match_id_search,omitempty"`
	// MatchIDs : whitelist de match_id à conserver. Si non vide, seules les
	// rows dont match_id ∈ MatchIDs sont gardées (filtre exact). Utilisé par
	// l'Explorer mode Joueur pour scoper aux matchs en commun avec une cible.
	MatchIDs []string `json:"match_ids,omitempty"`
}

// maxPageSize est la taille de page maximale acceptée.
// Relevé de 200 à 10000 pour couvrir le cas Explorer qui charge tout l'historique
// d'un joueur en une requête (pagination client). 10000 = ordre de grandeur de
// l'historique max plausible d'un joueur très actif (Halo Infinite, plusieurs saisons).
const maxPageSize = 10000

// Validate vérifie la cohérence des paramètres de la requête historique.
// Page=0 et PageSize=0 sont acceptés (defaults : 1 et 20 côté service).
func (r MatchHistoryQueryRequest) Validate() error {
	if r.Pagination.Page < 0 {
		return fmt.Errorf("MatchHistoryQueryRequest: page doit être ≥ 0 (reçu %d)", r.Pagination.Page)
	}
	if r.Pagination.PageSize < 0 {
		return fmt.Errorf("MatchHistoryQueryRequest: page_size doit être ≥ 0 (reçu %d)", r.Pagination.PageSize)
	}
	if r.Pagination.PageSize > maxPageSize {
		return fmt.Errorf("MatchHistoryQueryRequest: page_size maximal est %d (reçu %d)", maxPageSize, r.Pagination.PageSize)
	}
	if err := r.Filters.Validate(); err != nil {
		return fmt.Errorf("MatchHistoryQueryRequest.Filters: %w", err)
	}
	return nil
}

// MatchHistoryPageResponse est la réponse de POST match-history/query.
type MatchHistoryPageResponse struct {
	Summary             MatchHistoryQuerySummary `json:"summary"`
	Table               MatchHistoryTable        `json:"table"`
	AvailableSortFields []string                 `json:"available_sort_fields"`
	AvailableColumns    []string                 `json:"available_columns"`
	ExportHint          *ExportHint              `json:"export_hint"`
	// Sprint 54 B : avertissement de privacy si le compte a des matchs privés.
	PrivacyWarning *MatchPrivacyWarning `json:"privacy_warning,omitempty"`
	// §5 plan Squad/Sessions : sessions disponibles (split solo/squad) pour
	// alimenter SessionMultiSelect côté front. Triées StartedAt DESC.
	SessionLabels SessionLabelsList `json:"session_labels"`
	// BriefingKPIs alimente le composant <SessionBriefing> en haut de la page
	// Stats Solo (mode solo : pas de squad verdict). Calculé sur les canonical
	// rows correspondant aux match_id filtrés. Nil si aucun match.
	BriefingKPIs *KPIStats `json:"briefing_kpis,omitempty"`
}

// ExportHint indique qu'un export CSV est disponible.
type ExportHint struct {
	FileName      string  `json:"file_name"`
	EstimatedRows int     `json:"estimated_rows"`
	Token         *string `json:"token"`
}
