// Package domain — teammates.go : types pour la page Teammates (contrat FastAPI).
//
// Sprint 33 :
//
//	POST /api/v1/players/{slug}/pages/teammates → TeammatesPageResponse
package domain

import "time"

// ---------------------------------------------------------------------------
// Requête
// ---------------------------------------------------------------------------

// TeammatesQueryRequest est le corps de POST /pages/teammates.
type TeammatesQueryRequest struct {
	SelectedGamertags []string            `json:"selected_gamertags"`
	Filters           *FilterContextInput `json:"filters,omitempty"`
	// Multi-sessions : l'union des labels sélectionnés est appliquée côté service.
	PickedSoloSessions  []string `json:"picked_solo_session_labels,omitempty"`
	PickedSquadSessions []string `json:"picked_squad_session_labels,omitempty"`
	// Locale de l'utilisateur (ex. "fr", "en") pour les libellés localisés.
	Locale string `json:"locale,omitempty"`
}

// ---------------------------------------------------------------------------
// Réponse
// ---------------------------------------------------------------------------

// TeammateOption est un coéquipier fréquent sélectionnable.
type TeammateOption struct {
	Gamertag       string     `json:"gamertag"`
	XUID           *string    `json:"xuid,omitempty"`
	EncounterCount int        `json:"encounter_count"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
}

// TeammateKPIs sont les KPIs agrégés pour un groupe de matchs.
type TeammateKPIs struct {
	MatchCount     int      `json:"match_count"`
	Wins           int      `json:"wins"`
	KDRatio        *float64 `json:"kd_ratio"`
	WinRate        float64  `json:"win_rate"`
	Accuracy       *float64 `json:"accuracy"`
	KillsPerGame   *float64 `json:"kills_per_game"`
	AssistsPerGame *float64 `json:"assists_per_game"`
	// Sprint N : précision avancée
	HeadshotKillsPerGame *float64 `json:"headshot_kills_per_game,omitempty"`
	PerfectKillsPerGame  *float64 `json:"perfect_kills_per_game,omitempty"`
}

// MapSquadStats agrège (sur l'historique complet, sans filtre temporel) les
// matchs du joueur principal sur une carte donnée joués avec exactement
// l'escouade sélectionnée (intersection stricte des xuids). Sert à alimenter
// HistoricalWinRate / HistoricalPerformanceAvg sur MapBreakdownRow et le
// taux historique du tableau Squad/Synergies.
type MapSquadStats struct {
	Wins    int
	Total   int
	PerfAvg *float64 // nil si aucun match n'a de performance_score
}

// MapBreakdownRow est la performance par carte pour la heatmap.
type MapBreakdownRow struct {
	MapID             string   `json:"-"` // UUID interne pour les lookups historiques — non exposé
	MapUI             string   `json:"map_ui"`
	MatchCount        int      `json:"match_count"`
	WinRate           float64  `json:"win_rate"`
	HistoricalWinRate *float64 `json:"historical_win_rate,omitempty"`
	// HistoricalMatchCount : nombre de matchs du joueur principal sur cette carte
	// avec exactement l'escouade sélectionnée sur TOUT l'historique (dénominateur
	// de HistoricalWinRate). Nil si aucune donnée historique. Alimente l'affichage
	// « Historique : N parties » du bullet chart teammates.02.
	HistoricalMatchCount *int `json:"historical_match_count,omitempty"`
	// PerformanceAvg : moyenne du performance_score sur les matchs escouade
	// filtrés (session courante). Nil si aucun match n'a de score renseigné.
	PerformanceAvg *float64 `json:"performance_avg,omitempty"`
	// HistoricalPerformanceAvg : moyenne du performance_score du joueur principal
	// sur TOUT son historique pour cette carte. Sert de référence pour le chart
	// teammates.13. Nil si aucune donnée historique avec score.
	HistoricalPerformanceAvg *float64 `json:"historical_performance_avg,omitempty"`
}

// SquadMatchSeriesPoint est un point de la série par match (perf/timeline).
type SquadMatchSeriesPoint struct {
	MatchID          string   `json:"match_id"`
	StartTime        string   `json:"start_time"` // ISO 8601
	Outcome          int      `json:"outcome"`
	PerformanceScore *float64 `json:"performance_score,omitempty"`
	TeamMMRAvg       float64  `json:"team_mmr_avg"`
	SessionLabel     *string  `json:"session_label,omitempty"`
}

// SquadFirstEventsRow est la ligne d'un joueur dans le butterfly first-events
// teammates.17 : 2 vecteurs alignés sur les bins du chart parent (KillCounts
// pour les premiers frags, DeathCounts pour les premières morts).
type SquadFirstEventsRow struct {
	Player      string `json:"player"`
	KillCounts  []int  `json:"kill_counts"`
	DeathCounts []int  `json:"death_counts"`
}

// SquadFirstEvents alimente teammates.17 (butterfly : premier frag positifs
// haut, première mort négatifs bas — bins 15 s).
type SquadFirstEvents struct {
	BinSizeSeconds int                   `json:"bin_size_seconds"`
	BinLabels      []string              `json:"bin_labels"` // ex: ["15s", "30s", "1m00s", ...]
	Rows           []SquadFirstEventsRow `json:"rows"`
}

// SquadWeaponBar est une ligne du chart kills par arme teammates.09 :
// 1 arme avec ses kills par joueur de l'escouade + total cumulé.
type SquadWeaponBar struct {
	WeaponID       int64          `json:"weapon_id"`
	Label          string         `json:"label"`
	IsGrenadeMelee bool           `json:"is_grenade_melee,omitempty"`
	KillsByPlayer  map[string]int `json:"kills_by_player"` // gamertag → kills
	TotalSquad     int            `json:"total_squad"`
}

// SquadWeaponKills alimente teammates.09 (barres horizontales groupées par
// arme, 1 trace par joueur). Players est l'ordre canonique (main puis
// teammates) ; Bars est trié par TotalSquad ASC (peu utilisées en haut).
type SquadWeaponKills struct {
	Players []string         `json:"players"`
	Bars    []SquadWeaponBar `json:"bars"`
}

// SquadKillMechanicBar est une barre du breakdown « mécaniques de kill » de la
// page Escouade (Halo 5) : 1 mécanique (assassination | ground_pound |
// shoulder_bash) avec ses kills par joueur de l'escouade + total cumulé.
type SquadKillMechanicBar struct {
	// Mechanic : clé stable (assassination | ground_pound | shoulder_bash).
	Mechanic      string         `json:"mechanic"`
	KillsByPlayer map[string]int `json:"kills_by_player"` // gamertag → kills
	TotalSquad    int            `json:"total_squad"`
}

// SquadKillMechanics alimente le breakdown « mécaniques natives Halo 5 » (barres
// empilées par coéquipier) de la page Escouade. Players = ordre canonique (main
// puis teammates) ; Bars = 1 par mécanique présente (total > 0). nil hors h5
// (capability native_kill_mechanics).
type SquadKillMechanics struct {
	Players []string               `json:"players"`
	Bars    []SquadKillMechanicBar `json:"bars"`
}

// SquadPerformanceSeriesPoint est une mesure 1-match × 1-joueur pour le
// chart family teammates.16. Toutes les métriques sont optionnelles (nil =
// non disponible côté DB / pas calculé). MatchOrder est un index 0..N-1
// commun à tous les joueurs (inner-join sur les matchs partagés).
type SquadPerformanceSeriesPoint struct {
	MatchID                   string   `json:"match_id"`
	StartTime                 string   `json:"start_time"` // ISO 8601
	MatchOrder                int      `json:"match_order"`
	MapName                   string   `json:"map_name,omitempty"` // libellé localisé de la carte
	Kills                     int      `json:"kills"`
	Deaths                    int      `json:"deaths"`
	Assists                   int      `json:"assists"`
	KDA                       *float64 `json:"kda,omitempty"`
	Accuracy                  *float64 `json:"accuracy,omitempty"`
	AvgLifeSeconds            *float64 `json:"avg_life_seconds,omitempty"`
	PerformanceScore          *float64 `json:"performance_score,omitempty"`
	MaxKillingSpree           *int     `json:"max_killing_spree,omitempty"`
	HeadshotKills             *int     `json:"headshot_kills,omitempty"`
	PerfectKills              *int     `json:"perfect_kills,omitempty"`
	MeleeKills                *int     `json:"melee_kills,omitempty"`        // frags mêlée (donut « répartition des frags »)
	PowerWeaponKills          *int     `json:"power_weapon_kills,omitempty"` // frags arme lourde
	GrenadeKills              *int     `json:"grenade_kills,omitempty"`      // frags grenade
	DamageDealt               *int     `json:"damage_dealt,omitempty"`       // dégâts bruts infligés (chart dégâts/frag)
	DamageTaken               *int     `json:"damage_taken,omitempty"`       // dégâts bruts subis (chart dégâts/mort)
	RendementOffensif         *float64 `json:"rendement_offensif,omitempty"`
	ResistanceDefensive       *float64 `json:"resistance_defensive,omitempty"`
	TeamMMR                   *float64 `json:"team_mmr,omitempty"`                    // MMR équipe ce match (issu de match_skill_rank)
	SkillRating               *float64 `json:"skill_rating,omitempty"`                // CSR ou LUSR mu du joueur (exclusifs par match)
	SkillDelta                *float64 `json:"skill_delta,omitempty"`                 // points gagnés/perdus ce match (positif/négatif)
	SkillRatingType           string   `json:"skill_rating_type,omitempty"`           // "csr" | "lusr" — vide si non disponible
	SkillPlaylistGroup        *string  `json:"skill_playlist_group,omitempty"`        // groupe normalisé (ex: "ranked-arena")
	SkillSeasonID             *string  `json:"skill_season_id,omitempty"`             // saison Halo (rupture de courbe si changement)
	SkillMeasurementRemaining *int     `json:"skill_measurement_remaining,omitempty"` // matchs de placement restants
}

// SquadSynergyRadarAxis est l'un des 6 axes du radar de participation
// (combat / survival / support / score / objective / impact). Value est
// normalisé 0..100, Raw garde la donnée brute pour debug/tooltip.
type SquadSynergyRadarAxis struct {
	Axis  string  `json:"axis"`
	Value float64 `json:"value"`
	Raw   float64 `json:"raw"`
}

// SquadSynergyRadarSeries est un profil radar pour un joueur de l'escouade
// sur les matchs PARTAGÉS (intersection des match_ids de tous les membres
// sélectionnés). Cf. teammates.06.
type SquadSynergyRadarSeries struct {
	Player     string                  `json:"player"`
	Axes       []SquadSynergyRadarAxis `json:"axes"`
	ModeFamily string                  `json:"mode_family,omitempty"`
}

// SquadIntensityMatchRow est une ligne du heatmap d'intensité teammates.15 :
// 1 match × 10 phases normalisées (0..1) + un label affichable côté front
// (carte + date).
type SquadIntensityMatchRow struct {
	MatchID string      `json:"match_id"`
	Label   string      `json:"label"`
	Phases  [10]float64 `json:"phases"`
}

// SquadIntensityOption est une entrée du segmented control du heatmap
// d'intensité (toggle "all" ou un joueur).
type SquadIntensityOption struct {
	Key   string `json:"key"`   // "all" | gamertag (utilisé pour l'index `Rows`)
	Label string `json:"label"` // texte affiché dans le toggle
}

// SquadIntensityProfile alimente teammates.15 (heatmap d'intensité avec
// toggle Tous/joueur). Les phases sont déjà bucket-isées et normalisées
// côté serveur.
type SquadIntensityProfile struct {
	Options []SquadIntensityOption              `json:"options"`
	Rows    map[string][]SquadIntensityMatchRow `json:"rows"` // optionKey → lignes (1 par match)
}

// SquadPerMinuteEntry est l'agrégat par joueur des stats /minute pour le
// chart teammates.14. Calculé à partir de sum(kills/deaths/assists) / sum(time_played_secs / 60).
// Si TimePlayedSecs == 0, les ratios sont nuls.
type SquadPerMinuteEntry struct {
	Player           string  `json:"player"`
	KillsPerMinute   float64 `json:"kills_per_minute"`
	DeathsPerMinute  float64 `json:"deaths_per_minute"`
	AssistsPerMinute float64 `json:"assists_per_minute"`
	MatchCount       int     `json:"match_count"`
}

// SquadSessionPoint est un point agrégé par session pour le chart timeline
// (teammates.04). Calculé sur l'union dédupliquée de allSquadRows groupé par
// SessionLabel. team_mmr_avg / win_rate facultatifs (toutes-Nil → omitempty).
type SquadSessionPoint struct {
	SessionLabel string   `json:"session_label"`
	SquadPerf    float64  `json:"squad_perf"` // moyenne PerformanceScore (0..100), 0 si aucun score
	MatchCount   int      `json:"match_count"`
	Wins         int      `json:"wins"`
	Losses       int      `json:"losses"`
	WinRate      *float64 `json:"win_rate,omitempty"`     // wins/match_count si match_count>0
	TeamMMRAvg   *float64 `json:"team_mmr_avg,omitempty"` // moyenne TeamMMR si dispo
}

// SquadMapHeatmapCell est une cellule (joueur, carte, perf_avg) pour le
// heatmap teammates.03. n_matches sert au tooltip.
type SquadMapHeatmapCell struct {
	Player     string   `json:"player"`
	MapUI      string   `json:"map_ui"`
	PerfAvg    *float64 `json:"perf_avg,omitempty"` // nil si pas de match avec score
	MatchCount int      `json:"match_count"`
}

// SquadMapHeatmap regroupe les axes ordonnés + cellules pour teammates.03.
// MapsTopN est limité à 15 cartes (les plus jouées toutes équipes confondues).
type SquadMapHeatmap struct {
	Players  []string              `json:"players"`   // ordre Y (moi en tête)
	MapsTopN []string              `json:"maps_topn"` // ordre X
	Cells    []SquadMapHeatmapCell `json:"cells"`
}

// SquadImpactBadgeCount est l'agrégat d'un badge sur un joueur (col aggrégat
// du scoreboard teammates.07).
type SquadImpactBadgeCount struct {
	BadgeKey string `json:"badge_key"`
	Count    int    `json:"count"`
}

// SquadImpactCell est une cellule joueur×match du scoreboard teammates.07 :
// la liste des badges obtenus par ce joueur dans ce match.
type SquadImpactCell struct {
	Player    string   `json:"player"`
	MatchID   string   `json:"match_id"`
	BadgeKeys []string `json:"badge_keys"`
}

// SquadImpactPlayerSummary agrège les comptes par badge + score pondéré pour
// un joueur sur l'ensemble des matchs.
type SquadImpactPlayerSummary struct {
	Player string                  `json:"player"`
	Counts []SquadImpactBadgeCount `json:"counts"`
	Score  float64                 `json:"score"`
}

// SquadImpactMatchHeader est un en-tête de colonne match dans teammates.07.
type SquadImpactMatchHeader struct {
	MatchID string `json:"match_id"`
	Outcome int    `json:"outcome"` // outcome du joueur principal
}

// SquadImpactMatrix regroupe toutes les données nécessaires au scoreboard
// teammates.07. Les players sont triés par Score DESC côté serveur ; le front
// rend le tableau tel quel + applique badges (Champion / Maillon faible /
// Passager clandestin) selon position.
type SquadImpactMatrix struct {
	Matches  []SquadImpactMatchHeader   `json:"matches"`
	Players  []SquadImpactPlayerSummary `json:"players"`
	Cells    []SquadImpactCell          `json:"cells"`
	BadgeOrd []string                   `json:"badge_ord"` // ordre canonique des badges (= colonnes agg)
}

// SquadMatchHistoryRow est une ligne du tableau historique des matchs partagés
// (teammates.11). Une ligne par match unique sur le scope filtré (cascade,
// période, sessions escouade, sélection coéquipiers). Tri client par
// StartTime DESC. Pagination assurée côté front (TanStack Table).
type SquadMatchHistoryRow struct {
	MatchID          string   `json:"match_id"`
	StartTime        string   `json:"start_time"` // ISO 8601
	MapUI            string   `json:"map_ui"`
	PlaylistName     string   `json:"playlist_name,omitempty"`
	PairName         string   `json:"pair_name,omitempty"` // libellé brut
	ModeUI           string   `json:"mode_ui,omitempty"`   // mode normalisé (NormalizeModeLabel)
	Outcome          int      `json:"outcome"`
	Kills            int      `json:"kills"`
	Deaths           int      `json:"deaths"`
	Assists          int      `json:"assists"`
	Accuracy         *float64 `json:"accuracy,omitempty"`
	PerformanceScore *float64 `json:"performance_score,omitempty"`
	// TeamMMRAvg : MMR moyen de l'équipe. nil quand le titre ne fournit pas de MMR
	// d'équipe (Halo 5, games.ProvidesTeamMMR=false) → le front masque la colonne MMR
	// au lieu d'afficher 0. omitempty pour distinguer nil (masquer) de 0 (réel).
	TeamMMRAvg      *float64 `json:"team_mmr_avg,omitempty"`
	EnemyMMRAvg     *float64 `json:"enemy_mmr_avg,omitempty"`
	DeltaMMR        *float64 `json:"delta_mmr,omitempty"`
	ScoreLabel      string   `json:"score_label,omitempty"`
	DurationSeconds int      `json:"duration_seconds,omitempty"`
	// GameplayDurationSeconds : durée réelle de gameplay (countdown retranché),
	// préférée par le front pour l'affichage de la durée du match.
	GameplayDurationSeconds int `json:"gameplay_duration_seconds,omitempty"`
	// WinRateHist : taux de victoire historique du joueur sur cette carte
	// (ratio 0..1). Calculé sur l'historique complet (canonicalRows non
	// filtré). WinRateHistTotal = nombre de matchs joués sur la carte. Sert
	// de référence à comparer avec le résultat du match courant.
	WinRateHist      *float64 `json:"win_rate_hist,omitempty"`
	WinRateHistTotal *int     `json:"win_rate_hist_total,omitempty"`
	// ExpectedWinProb : proba de victoire pré-match de l'équipe ∈ [0,1] (LUSR v2).
	// Alimente la colonne « Prob. vic. ». Nil si pré-v2 / non disponible.
	ExpectedWinProb *float64 `json:"expected_win_prob,omitempty"`
	SessionLabel    *string  `json:"session_label,omitempty"`
}

// TeammateRow est une ligne de résultat (stats avec vs sans un coéquipier).
type TeammateRow struct {
	Gamertag       string        `json:"gamertag"`
	XUID           *string       `json:"xuid,omitempty"`
	EncounterCount int           `json:"encounter_count"`
	LastSeenAt     *time.Time    `json:"last_seen_at,omitempty"`
	WithKPIs       TeammateKPIs  `json:"with_kpis"`
	WithoutKPIs    *TeammateKPIs `json:"without_kpis,omitempty"`
}

// SessionLabelEntry est une session avec sa plage temporelle et ses métadonnées
// d'expérience/playlist pour le filtre de navigation client.
type SessionLabelEntry struct {
	Label       string    `json:"label"`
	StartedAt   time.Time `json:"started_at"`
	EndedAt     time.Time `json:"ended_at"`
	Experiences []string  `json:"experiences,omitempty"`
	Playlists   []string  `json:"playlists,omitempty"`
}

// SessionLabelsList contient les sessions disponibles pour les deux scopes (solo/escouade).
// Triées par StartedAt DESC côté service.
type SessionLabelsList struct {
	Solo  []SessionLabelEntry `json:"solo"`
	Squad []SessionLabelEntry `json:"squad"`
}

// MedalDigestItem est une médaille agrégée sur tous les matchs partagés
// pour un joueur donné.
//
// Icône title-aware : ImageURL (PNG, HINF) OU les champs Sprite* (feuille + offset,
// Halo 5). Mutuellement exclusifs ; mêmes tags JSON que l'Asset Drawer (AssetMeta).
type MedalDigestItem struct {
	MedalID       int64  `json:"medal_id"`
	Label         string `json:"label,omitempty"`
	Description   string `json:"description,omitempty"`
	ImageURL      string `json:"image_url,omitempty"`
	TotalCount    int    `json:"total_count"`              // total sur tous les matchs
	MatchCount    int    `json:"match_count"`              // nb matchs où obtenu
	Category      string `json:"category,omitempty"`       // multikill | spree | skill | style | mode | proficiency
	Difficulty    string `json:"difficulty,omitempty"`     // Normal | Heroic | Legendary | Mythic
	PersonalScore int    `json:"personal_score,omitempty"` // XP de carrière par médaille (0 si absent)
	SpriteSheet   string `json:"sprite_sheet,omitempty"`
	SpriteLeft    int    `json:"sprite_left,omitempty"`
	SpriteTop     int    `json:"sprite_top,omitempty"`
	SpriteWidth   int    `json:"sprite_width,omitempty"`
	SpriteHeight  int    `json:"sprite_height,omitempty"`
}

// MedalDigestEntry est le résumé médailles d'un joueur sur les matchs partagés.
// Alimente <MedalDigest> dans SquadSynergiesPage (bottom card narrative).
type MedalDigestEntry struct {
	Player        string            `json:"player"`               // gamertag
	EmblemURL     string            `json:"emblem_url,omitempty"` // URL emblème Spartan (optionnel)
	DistinctTypes int               `json:"distinct_types"`       // nb types distincts
	TotalCount    int               `json:"total_count"`          // total toutes médailles
	AvgPerMatch   float64           `json:"avg_per_match"`        // moy. par match avec médaille
	PeakInMatch   int               `json:"peak_in_match"`        // max en 1 match
	TopMedals     []MedalDigestItem `json:"top_medals"`           // top 5 par count
	AllMedals     []MedalDigestItem `json:"all_medals"`           // tous triés count desc
}

// TeammatesPageResponse est la réponse de POST /pages/teammates.
//
// Le champ historique `solo_reference` a été retiré : la page Solo a son
// propre endpoint dédié, et la page Escouade ne compare plus contre une
// baseline solo (cf. .ai/thought_log.md 2026-04-26 refonte UX Escouade).
type TeammatesPageResponse struct {
	Options       []TeammateOption  `json:"options"`
	Teammates     []TeammateRow     `json:"teammates"`
	TotalMatches  int               `json:"total_matches"`
	SessionLabels SessionLabelsList `json:"session_labels"`
	// FriendsCount : nombre total d'amis configurés (settings.friend_gamertags).
	// Le label UI "parmi N amis" s'appuie dessus.
	FriendsCount int `json:"friends_count"`
	// Sprint N : données graphiques par coéquipier sélectionné
	Timeseries   []SquadTimeseriesPoint             `json:"timeseries,omitempty"`
	MapBreakdown []MapBreakdownRow                  `json:"map_breakdown,omitempty"`
	MatchSeries  map[string][]SquadMatchSeriesPoint `json:"match_series,omitempty"`
	// MatchHistory alimente le tableau teammates.11 (TanStack Table). Une ligne
	// par match unique partagé sur le scope filtré, triée par StartTime DESC.
	// La pagination est gérée côté client (20/page par défaut).
	MatchHistory []SquadMatchHistoryRow `json:"match_history,omitempty"`
	// SessionTimeline alimente teammates.04 (bars perf colorées + line winrate
	// + line MMR sur axe Y2 conditionnel). Un point par session label.
	SessionTimeline []SquadSessionPoint `json:"session_timeline,omitempty"`
	// MapHeatmap alimente teammates.03 (heatmap player×map top 15).
	MapHeatmap *SquadMapHeatmap `json:"map_heatmap,omitempty"`
	// ImpactMatrix alimente teammates.07 (scoreboard impact, 8 badges).
	ImpactMatrix *SquadImpactMatrix `json:"impact_matrix,omitempty"`
	// PerMinuteStats alimente teammates.14 (bars groupées K/D/A par minute par joueur).
	PerMinuteStats []SquadPerMinuteEntry `json:"per_minute_stats,omitempty"`
	// SynergyRadar alimente teammates.06 (radar 6 axes par joueur sur les
	// matchs PARTAGÉS). Nil si aucun match commun.
	SynergyRadar []SquadSynergyRadarSeries `json:"synergy_radar,omitempty"`
	// IntensityProfile alimente teammates.15 (heatmap d'intensité avec toggle
	// Tous/joueur). Nil si <3 matchs ou aucun kill event.
	IntensityProfile *SquadIntensityProfile `json:"intensity_profile,omitempty"`
	// PerformanceSeries alimente teammates.16 (8 sous-charts par joueur sur
	// matchs partagés). Map gamertag → série triée par MatchOrder ASC. Nil
	// si aucun match commun.
	PerformanceSeries map[string][]SquadPerformanceSeriesPoint `json:"performance_series,omitempty"`
	// WeaponKills alimente teammates.09 (kills par arme, comparatif multi-joueurs).
	// Nil si aucune donnée weapon_kills disponible (capability absente ou shared
	// match_ids vides).
	WeaponKills *SquadWeaponKills `json:"weapon_kills,omitempty"`
	// NativeKillMechanics : breakdown des mécaniques natives Halo 5 par coéquipier
	// (assassinats + compétences spartiate, barres empilées). Nil hors h5
	// (capability native_kill_mechanics) ou aucune mécanique sur les matchs partagés.
	NativeKillMechanics *SquadKillMechanics `json:"native_kill_mechanics,omitempty"`
	// FirstEvents alimente teammates.17 (butterfly premier frag/première mort,
	// bins 15 s). Nil si aucune donnée highlight_events.
	FirstEvents *SquadFirstEvents `json:"first_events,omitempty"`
	// Header alimente <SessionBriefing> en haut de SquadLayout (Synergies +
	// Contributions). Mode solo (SoloKPIs uniquement) si aucun coequipier
	// selectionne ; mode squad complet (KPIsByXUID + TeamAvgKPIs + PlayerCards
	// + SquadScore) si selection >= 1.
	Header *SquadHeader `json:"header,omitempty"`
	// MainPlayer est le gamertag du joueur principal (proprietaire de la page)
	// — necessaire au front pour identifier le card "moi" dans Header.PlayerCards.
	MainPlayer string `json:"main_player,omitempty"`
	// MedalDigest alimente <MedalDigest> en bas de SquadSynergiesPage :
	// résumé narratif médailles par joueur sur les matchs partagés.
	// Nil si aucune médaille disponible ou squad vide.
	MedalDigest []MedalDigestEntry `json:"medal_digest,omitempty"`
	// CompositionSessions : sessions où la composition EXACTE (joueur principal +
	// TOUS les coéquipiers sélectionnés) a joué ensemble — intersection des
	// matchs, historique complet (non filtré par session). Alimente le
	// SessionMultiSelect ET le ré-ancrage front. Sans coéquipier sélectionné,
	// reprend les sessions squad du joueur principal (SessionLabels.Squad).
	CompositionSessions []SessionLabelEntry `json:"composition_sessions,omitempty"`
	// LatestCompositionSession : label de la session la plus récente de la
	// composition exacte (1re entrée de CompositionSessions). Vide si la
	// composition n'a jamais joué ensemble. Le front s'y ré-ancre quand la
	// sélection change.
	LatestCompositionSession string `json:"latest_composition_session,omitempty"`
}
