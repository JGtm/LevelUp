// Package domain — timeseries.go : types pour la page Timeseries (contrat FastAPI).
//
// Sprint 33 :
//
//	POST /api/v1/players/{slug}/pages/timeseries → TimeseriesPageResponse
//
// Architecture data-only : le Go envoie uniquement les data points bruts.
// Phase 3 P3.B + P3.F+G : le frontend consomme ces points via les wrappers
// ECharts (`HistogramChart`, `ScatterChart`, `Heatmap2DChart`,
// `TimeseriesLineChart`, `TimeseriesCombatYield`, `TimeseriesKdaBars`).
// Les anciens champs `*PlotlyFigurePayload` jamais populés ont été retirés
// en P3 cleanup.
package domain

import "time"

// ---------------------------------------------------------------------------
// Requête
// ---------------------------------------------------------------------------

// TimeseriesQueryRequest est le corps de POST /pages/timeseries.
type TimeseriesQueryRequest struct {
	Filters FilterContextInput `json:"filters"`
}

// ---------------------------------------------------------------------------
// Onglets
// ---------------------------------------------------------------------------

// TimeseriesKpiCard est une carte KPI dans l'onglet résumé.
//
// P7.1 (revue 2026-04-29) : champ `Color` retiré — le ton/couleur est résolu
// côté front via tokens sémantiques (`tokenCssVar` + delta sign), pas
// transporté dans le DTO. Réduit le couplage présentation/transport.
type TimeseriesKpiCard struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Value string  `json:"value"`
	Delta *string `json:"delta"`
}

// TimeseriesSummaryTab est l'onglet Résumé.
type TimeseriesSummaryTab struct {
	KpiCards []TimeseriesKpiCard `json:"kpi_cards"`
}

// TimeseriesWeaponKill agrège les kills d'une arme sur le scope filtré.
// Alimente le chart timeseries.04 (Top weapons by kills).
type TimeseriesWeaponKill struct {
	WeaponID int64  `json:"weapon_id"`
	Label    string `json:"label"`
	Kills    int    `json:"kills"`
	// Class : axe manipulation de l'arme (registre, résolu par ResolveRoles) —
	// recolore chaque barre par classe (cohérence sunburst v2). Vide si non résolu.
	Class string `json:"class,omitempty"`
}

// TimeseriesKillTypes agrège la « répartition des frags » par TYPE sur le scope
// filtré (1er onglet, donut). Types d'arme de base + mécaniques natives Halo 5
// (assassinats + compétences spartiate). Les 3 mécaniques sont 0 pour les titres
// qui ne les fournissent pas ; le donut est capability-gated côté front.
type TimeseriesKillTypes struct {
	TotalKills        int `json:"total_kills"`
	MeleeKills        int `json:"melee_kills"`
	GrenadeKills      int `json:"grenade_kills"`
	PowerWeaponKills  int `json:"power_weapon_kills"`
	Assassinations    int `json:"assassinations"`
	GroundPoundKills  int `json:"ground_pound_kills"`
	ShoulderBashKills int `json:"shoulder_bash_kills"`
}

// OutcomesPeriodPoint agrège les outcomes (V/D/N/X) sur une période (jour/semaine/mois).
// Alimente le chart timeseries.05 (Outcomes over time).
type OutcomesPeriodPoint struct {
	PeriodLabel string `json:"period_label"` // ex. "2025-W42" ou "2025-10"
	StartDate   string `json:"start_date"`   // ISO date début de la période
	Wins        int    `json:"wins"`
	Losses      int    `json:"losses"`
	Ties        int    `json:"ties"`
	DNF         int    `json:"dnf"`
}

// FirstEventBucket agrège la distribution superposée du premier kill et de la
// première mort par bin de N secondes depuis le début du match. Alimente le
// chart timeseries.11 (Premier événement).
type FirstEventBucket struct {
	LowerSeconds float64 `json:"lower_seconds"`
	UpperSeconds float64 `json:"upper_seconds"`
	FirstKills   int     `json:"first_kills"`
	FirstDeaths  int     `json:"first_deaths"`
}

// FirstEventDistribution est la distribution complète + les moyennes pour les
// markLines (Moy. 38s sur le mock).
type FirstEventDistribution struct {
	Buckets               []FirstEventBucket `json:"buckets"`
	MeanFirstKillSeconds  *float64           `json:"mean_first_kill_seconds,omitempty"`
	MeanFirstDeathSeconds *float64           `json:"mean_first_death_seconds,omitempty"`
}

// IntensityMatchRow est une ligne du heatmap d'intensité solo (chart
// "Intensité — frags par phase de match"). 1 match × 10 phases normalisées
// (0..1) + label affichable (carte + date).
//
// Format aligné avec domain.SquadIntensityMatchRow pour réutiliser le
// composant SquadIntensityHeatmapChart côté front. Pas de toggle joueur :
// solo n'a qu'un seul "joueur".
type IntensityMatchRow struct {
	MatchID string      `json:"match_id"`
	Label   string      `json:"label"`
	Phases  [10]float64 `json:"phases"`
}

// SoloSessionPerfPoint est l'agrégat par session/semaine/mois pour le chart
// "Performance solo par session" (Synthèse).
//
// Calculé sur la population solo complète (post match_context), indépendant
// des filtres period/sessions/cascade — sinon picker une session ne ramène
// qu'un seul point, ce qui défait l'intérêt du chart.
type SoloSessionPerfPoint struct {
	SessionLabel string   `json:"session_label"`
	StartedAtUTC string   `json:"started_at_utc"` // ISO date pour tri chronologique
	MatchCount   int      `json:"match_count"`
	Wins         int      `json:"wins"`
	WinRate      float64  `json:"win_rate"`               // 0..1
	PerfAvg      *float64 `json:"perf_avg,omitempty"`     // moyenne PerfScoreComputed
	TeamMMRAvg   *float64 `json:"team_mmr_avg,omitempty"` // moyenne TeamMMR
}

// SoloSessionPerfBlock contient les points + leur granularité (session, week,
// month) pour permettre au frontend de labelliser correctement.
type SoloSessionPerfBlock struct {
	// Granularity : "session" | "week" | "month" — choisie automatiquement
	// selon la densité (≤30 points → session, sinon week, sinon month).
	Granularity string                 `json:"granularity"`
	Points      []SoloSessionPerfPoint `json:"points"`
}

// TimeseriesCumulTab est l'onglet Cumul.
type TimeseriesCumulTab struct {
	CumulativeKD  []CumulativePoint `json:"cumulative_kd"`
	CumulativeNet []CumulativePoint `json:"cumulative_net"`
	RollingKD     []CumulativePoint `json:"rolling_kd"`
}

// TimeseriesIntensityTab est l'onglet Intensité.
type TimeseriesIntensityTab struct {
	HeatmapData     []IntensityHeatmapPoint `json:"heatmap_data"`
	ScorePerMinData []CumulativePoint       `json:"score_per_min_data"`
}

// IntensityHeatmapPoint est un point de la heatmap jour×heure.
type IntensityHeatmapPoint struct {
	DayOfWeek int     `json:"day_of_week"` // 0=lundi, 6=dimanche
	Hour      int     `json:"hour"`        // 0-23
	Count     int     `json:"count"`
	AvgKD     float64 `json:"avg_kd"`
}

// TimeseriesDistributionsTab est l'onglet Distributions.
type TimeseriesDistributionsTab struct {
	KDABuckets         []DistributionBucket `json:"kda_buckets"`
	KillsBuckets       []DistributionBucket `json:"kills_buckets"`
	AccuracyBuckets    []DistributionBucket `json:"accuracy_buckets"`
	ScorePerMinBuckets []DistributionBucket `json:"score_per_min_buckets"`
	RollingWRBuckets   []DistributionBucket `json:"rolling_wr_buckets"`
	// LifeBuckets : distribution de la durée de vie moyenne par match
	// (time_played_seconds / (deaths + 1) — même formule que buildCorrelationPoints).
	// Bins de 5 secondes. Alimente l'histogramme "Average life" timeseries.09.
	LifeBuckets []DistributionBucket `json:"life_buckets"`
	// PerfScoreBuckets : distribution du performance_score par match
	// (PerfScoreComputed du sync). Bins de 5 points sur [0, 100].
	// Alimente l'histogramme "Performance" timeseries.09.
	PerfScoreBuckets []DistributionBucket `json:"perf_score_buckets"`
	// PersonalScoreBuckets : distribution du score personnel par match
	// (PersonalScore — synced depuis match_participants). Bins de 250 points.
	PersonalScoreBuckets []DistributionBucket `json:"personal_score_buckets"`
	// MaxKillingSpreeBuckets : distribution du max killing spree par match
	// (MaxKillingSpree — synced depuis match_participants). Bin = 1 (entiers).
	MaxKillingSpreeBuckets []DistributionBucket  `json:"max_killing_spree_buckets"`
	CorrelationPoints      []CorrelationDataPair `json:"correlation_points"`
}

// DistributionBucket est un bucket pour un histogramme de distribution.
//
// P7.1 (revue 2026-04-29) : champs renommés `BinStart/BinEnd` (terme ECharts)
// → `BucketLower/BucketUpper` (sémantique métier). Le bucket représente une
// tranche [BucketLower, BucketUpper) d'une métrique (KDA, kills, accuracy…) ;
// les noms ECharts ne portaient pas cette intention.
type DistributionBucket struct {
	BucketLower float64 `json:"bucket_lower"`
	BucketUpper float64 `json:"bucket_upper"`
	Count       int     `json:"count"`
}

// CorrelationDataPair est une paire (x, y) pour un scatter plot de corrélation.
//
// P7.1 (revue 2026-04-29) : champs renommés vers une sémantique métier.
// `Label` (composite ECharts "kills_vs_kd") → couple `MetricXKey/MetricYKey`
// (clés canoniques séparées) ; `X/Y` (axes ECharts) → `XValue/YValue` (valeurs
// pour ces deux métriques).
//
// Exemples MetricXKey/MetricYKey : ("kills","kd_ratio"), ("lifespan","kills"),
// ("accuracy","kda"), ("kills","deaths"), ("mmr_team","mmr_enemy").
type CorrelationDataPair struct {
	MetricXKey string  `json:"metric_x_key"`
	MetricYKey string  `json:"metric_y_key"`
	XValue     float64 `json:"x_value"`
	YValue     float64 `json:"y_value"`
	Outcome    *int    `json:"outcome"` // nil si inconnu ; 2=victoire, 3=défaite, 1=égalité
}

// TimeseriesMatchRow est une ligne de données par match pour les charts timeline.
// Fourni dans TimeseriesPageResponse.MatchRows pour permettre au frontend de
// construire les graphes K/D/A, assists, dégâts, perf score, etc.
type TimeseriesMatchRow struct {
	MatchID   string    `json:"match_id"`
	Index     int       `json:"index"`
	StartTime time.Time `json:"start_time"`
	Kills     int       `json:"kills"`
	Deaths    int       `json:"deaths"`
	Assists   int       `json:"assists"`
	// KDA et KDRatio exposés par P2.5 (revue 2026-04-29 ADR 0006).
	// Ils débloquent la suppression du recompute K/D côté front
	// (TimeseriesKdaBars.tsx:78 — voir B3).
	KDA               *float64 `json:"kda,omitempty"`
	KDRatio           *float64 `json:"kd_ratio,omitempty"`
	Accuracy          *float64 `json:"accuracy"`
	Outcome           *int     `json:"outcome"`
	PersonalScore     *int     `json:"personal_score"`
	DamageDealt       *float64 `json:"damage_dealt"`
	DamageTaken       *float64 `json:"damage_taken"`
	PerfScore         *float64 `json:"perf_score"`
	Rank              *int     `json:"rank"`
	PlaylistName      string   `json:"playlist_name"`
	TimePlayedSeconds *int     `json:"time_played_seconds"`
	// Métriques alimentant les charts Forme (timeseries.16) — sync depuis
	// match_participants (max_killing_spree, headshot_kills, perfect_kills).
	MaxKillingSpree *int `json:"max_killing_spree,omitempty"`
	HeadshotKills   *int `json:"headshot_kills,omitempty"`
	PerfectKills    *int `json:"perfect_kills,omitempty"`
	// Nom de carte pour les étiquettes X compactes (timeseries.14 "Stats par minute"
	// reprend le format `#N\nMap` de la page Contributions Squad).
	MapName   string `json:"map_name,omitempty"`
	MapNameFR string `json:"map_name_fr,omitempty"`
	// Skill rank (CSR ou LUSR) — rating brut + type + contexte playlist/saison.
	SkillRatingValue          *float64 `json:"skill_rating_value,omitempty"`
	SkillRatingType           string   `json:"skill_rating_type,omitempty"`
	SkillPlaylistGroup        *string  `json:"skill_playlist_group,omitempty"`
	SkillSeasonID             *string  `json:"skill_season_id,omitempty"`
	SkillMeasurementRemaining *int     `json:"skill_measurement_remaining,omitempty"`
	// Session de rattachement — alimente l'agrégat "Performance solo par session".
	SessionLabel *string `json:"session_label,omitempty"`
	// MMR équipe — alimente le chart "Performance par session" (axe MMR moyen).
	TeamMMR *float64 `json:"team_mmr,omitempty"`
	// Stats attendues (écart au FDA attendu, chart « Écart au FDA attendu »).
	// KdaExpected = kills_expected + assists_expected/3 − deaths_expected
	// (analysis.ExpectedFDA). Tous nil hors titre à CapExpectedStats (Halo 5) ou
	// match sans attendu → le chart est gaté par la capability côté front.
	KillsExpected   *float64 `json:"kills_expected,omitempty"`
	DeathsExpected  *float64 `json:"deaths_expected,omitempty"`
	AssistsExpected *float64 `json:"assists_expected,omitempty"`
	KdaExpected     *float64 `json:"kda_expected,omitempty"`
}

// ---------------------------------------------------------------------------
// Réponse
// ---------------------------------------------------------------------------

// TimeseriesPageResponse est la réponse de POST /pages/timeseries.
type TimeseriesPageResponse struct {
	TotalMatches int                  `json:"total_matches"`
	MatchRows    []TimeseriesMatchRow `json:"match_rows"`
	SummaryTab   TimeseriesSummaryTab `json:"summary_tab"`
	CumulTab     TimeseriesCumulTab   `json:"cumul_tab"`

	IntensityTab     TimeseriesIntensityTab     `json:"intensity_tab"`
	DistributionsTab TimeseriesDistributionsTab `json:"distributions_tab"`
	// TopWeapons : top 10 armes par kills sur le scope filtré (chart .04).
	// Vide si WeaponKillsRepository non câblé ou aucun kill enregistré.
	TopWeapons []TimeseriesWeaponKill `json:"top_weapons"`
	// KillTypes : « répartition des frags » par TYPE sur la période (1er onglet,
	// donut). Types d'arme + mécaniques natives Halo 5. Nil si aucun match.
	// Capability-gated côté front (donut masqué hors h5).
	KillTypes *TimeseriesKillTypes `json:"kill_types,omitempty"`
	// OutcomesOverTime : V/D/N/X agrégés par période (chart .05).
	// La granularité (jour/semaine/mois) est choisie automatiquement selon
	// la durée du scope : <=14j → jour, <=120j → semaine, sinon mois.
	OutcomesOverTime []OutcomesPeriodPoint `json:"outcomes_over_time"`
	// MapBreakdown : stats par carte — session courante (filtrée) avec leurs
	// pendants historiques (tous matchs solo). Alimente teammates.02 (bullet
	// winrate) et teammates.13 (grouped-bar perf) sur la page Stats. Vide si
	// aucun match dans le scope.
	MapBreakdown []MapBreakdownRow `json:"map_breakdown"`
	// FirstEvents : distribution superposée du premier kill et de la première
	// mort, alimentée via highlight_events. Nil si HighlightEventsRepo absent
	// ou aucun event dans le scope.
	FirstEvents *FirstEventDistribution `json:"first_events,omitempty"`
	// IntensityRows : 1 ligne par match × 10 phases normalisées (0..1) — frags
	// du joueur sur la timeline du match, source highlight_events.
	IntensityRows []IntensityMatchRow `json:"intensity_rows,omitempty"`
	// SoloSessionPerf : agrégat par session/semaine/mois sur la population
	// solo complète (ignore filtres period/sessions/cascade). Alimente
	// "Performance solo par session" sur l'onglet Synthèse. Granularité
	// auto-adaptative selon la densité.
	SoloSessionPerf *SoloSessionPerfBlock `json:"solo_session_perf,omitempty"`
	// BriefingKPIs alimente le composant <SessionBriefing> en haut de la page
	// (mode solo : pas de squad verdict). Calcule sur les memes match_ids que
	// les autres onglets (apres filtres). Nil si aucun match.
	BriefingKPIs *KPIStats `json:"briefing_kpis,omitempty"`
	// FragDistribution : répartition hiérarchique classe→rôle (sunburst v2) sur le
	// scope filtré. Réutilise le builder partagé buildFragDistribution (classes API
	// melee/grenade/spartan + total, classes gun depuis le registre). Nil si aucun
	// frag. Title-agnostic (spartan_ability capability-gated native_kill_mechanics).
	FragDistribution *FragDistribution `json:"frag_distribution,omitempty"`
	// WeaponAccuracy : précision par arme (tirs au but / tirs tirés, 0..1) sur le
	// scope filtré — donnée native Halo 5, omise sur les titres sans la donnée
	// (Infinite → nil → le front retombe sur « Outils de destruction »). MÊME builder
	// partagé que Synthesis/Sessions (buildWeaponAccuracy). Nil si aucune arme valide.
	WeaponAccuracy []SynthesisWeaponAccuracyEntry `json:"weapon_accuracy,omitempty"`
}
