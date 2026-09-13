// Package domain — synthesis.go : types dédiés à la page Synthèse.
//
// Sprint 55 D1 : extraction depuis squad.go — Synthèse devient une page autonome,
// distincte d'Escouade au niveau handler, service, types domaine et contrat OpenAPI.
//
// Endpoint cible : POST /api/v1/players/{slug}/pages/synthesis
package domain

import "time"

// ---------------------------------------------------------------------------
// Types de requête (POST body)
// ---------------------------------------------------------------------------

// SynthesisRequest : corps de POST /pages/synthesis.
// Sprint 55 D2 : period et filters réellement appliqués par le service.
type SynthesisRequest struct {
	Period    string             `json:"period,omitempty"`     // "all" | "1w" | "1m" | "1y" | "2y"
	StartDate string             `json:"start_date,omitempty"` // ISO date "YYYY-MM-DD" (plage explicite)
	EndDate   string             `json:"end_date,omitempty"`   // ISO date "YYYY-MM-DD"
	Filters   FilterContextInput `json:"filters,omitempty"`    // filtres réellement appliqués en D2
}

// ---------------------------------------------------------------------------
// Bloc scope (D3) — écho explicite du scope réellement appliqué
// ---------------------------------------------------------------------------

// SynthesisScope décrit le scope réellement appliqué lors du calcul Synthèse.
// Retourné en tête de SynthesisPageV2Response pour ancrer la crédibilité de la page.
type SynthesisScope struct {
	Period         string    `json:"period"`                    // période effective appliquée
	MatchCount     int       `json:"match_count"`               // matchs dans le scope
	FiltersApplied []string  `json:"filters_applied,omitempty"` // filtres réellement utilisés
	FiltersIgnored []string  `json:"filters_ignored,omitempty"` // filtres déclarés mais ignorés
	Description    string    `json:"description"`               // résumé lisible du scope
	ComputedAt     time.Time `json:"computed_at"`               // instant de calcul
}

// ---------------------------------------------------------------------------
// Bloc overview (D4) — cumuls, moyennes et pics fiables
// ---------------------------------------------------------------------------

// SynthesisOverview est le premier vrai bloc analytique de la page.
// Contient uniquement les métriques fiables depuis la DB locale (pas de simulation).
type SynthesisOverview struct {
	// Volumes cumulés
	TotalMatches int `json:"total_matches"`
	TotalWins    int `json:"total_wins"`
	TotalLosses  int `json:"total_losses"`
	TotalTies    int `json:"total_ties"`
	TotalDNF     int `json:"total_dnf"`
	TotalKills   int `json:"total_kills"`
	TotalDeaths  int `json:"total_deaths"`
	TotalAssists int `json:"total_assists"`

	// Moyennes d'efficacité
	AvgKDA       *float64 `json:"avg_kda,omitempty"`
	AvgKills     *float64 `json:"avg_kills,omitempty"`
	AvgDeaths    *float64 `json:"avg_deaths,omitempty"`
	WinRate      float64  `json:"win_rate"`
	AvgPerfScore *float64 `json:"avg_perf_score,omitempty"`

	// TotalKDR exposé par P2.5 (revue 2026-04-29 ADR 0006).
	// Calcul canonique : sum(kills) / max(1, sum(deaths)) — distinct du
	// recompute front cassé qui faisait sum/sum (mathématiquement faux car
	// `sum(K)/sum(D)` ≠ `avg(K/D)`). Débloque la suppression du recompute
	// SynthesisPage.tsx:139-141 (B3).
	TotalKDR *float64 `json:"total_kdr,omitempty"`

	// Records / pics
	// BestKillsMatch / BestKDAMatch sont conservés pour compatibilité (valeurs scalaires).
	// Les nouveaux champs *Ref ajoutent le match_id associé pour permettre la navigation
	// depuis les cartes "Top X" / "Meilleur X" de la page Synthesis (POST 2026-05-27).
	BestKillsMatch   *int     `json:"best_kills_match,omitempty"`
	BestKDAMatch     *float64 `json:"best_kda_match,omitempty"`
	LongestWinStreak int      `json:"longest_win_streak,omitempty"`
	// LongestLossStreak : plus longue série de défaites consécutives sur le scope
	// (rompue par tout non-loss : win / tie / dnf). Calculée sur la MÊME source de
	// matchs que LongestWinStreak via le helper canonique analysis.LongestRun —
	// cohérence garantie. omitempty : le front omet la carte quand la série <= 1.
	LongestLossStreak int `json:"longest_loss_streak,omitempty"`

	// Refs cliquables vers le match record pour chaque métrique. Nil si aucune
	// donnée exploitable sur le scope (ex : accuracy nil sur tous les matchs).
	BestKillsRef         *BestMatchRef `json:"best_kills_ref,omitempty"`
	BestKDARef           *BestMatchRef `json:"best_kda_ref,omitempty"`
	BestPerfRef          *BestMatchRef `json:"best_perf_ref,omitempty"`
	BestAccuracyRef      *BestMatchRef `json:"best_accuracy_ref,omitempty"`
	BestDamageRef        *BestMatchRef `json:"best_damage_ref,omitempty"`
	BestKillingSpreeRef  *BestMatchRef `json:"best_killing_spree_ref,omitempty"`
	BestHeadshotsRef     *BestMatchRef `json:"best_headshots_ref,omitempty"`
	BestPersonalScoreRef *BestMatchRef `json:"best_personal_score_ref,omitempty"`
}

// BestMatchRef identifie le match record pour une métrique donnée et porte la
// valeur observée. Permet à la page Synthesis d'ouvrir le match en question
// via useNavigateToMatch côté front.
type BestMatchRef struct {
	MatchID string  `json:"match_id"`
	Value   float64 `json:"value"`
}

// ---------------------------------------------------------------------------
// Réponse principale — SynthesisPageResponse v2 (Sprint 55)
// ---------------------------------------------------------------------------

// SynthesisPageV2Response est la nouvelle réponse de POST /pages/synthesis.
// Sprint 55 D3/D4 : ajoute scope explicite et overview en tête.
// P9 : ajoute DetailedStats pour le KPI Grid par catégories.
// Conserve solo_kpis/squad_kpis/comparison_metrics/heatmap/top_weeks pour compatibilité.
type SynthesisPageV2Response struct {
	// Bloc 0 — scope (D3)
	Scope SynthesisScope `json:"scope"`

	// Bloc 1 — overview (D4)
	Overview SynthesisOverview `json:"overview"`

	// Blocs existants conservés (compatibilité Sprint 43)
	SoloKPIs          SynthesisKPIs          `json:"solo_kpis"`
	SquadKPIs         SynthesisKPIs          `json:"squad_kpis"`
	ComparisonMetrics []ComparisonMetricItem `json:"comparison_metrics"`
	HeatmapData       []TemporalHeatmapCell  `json:"heatmap_data"`
	TopWeeks          []TopWeekEntry         `json:"top_weeks"`

	// Blocs previews (D5/D7) — D6 (Rivalries) retiré le 2026-05-27, voir
	// thought_log : la section "Relations de jeu" a été supprimée de la
	// page Synthesis ; la page palmares/relations reste alimentée par
	// CareerRepo.GetEncounters indépendamment.
	HighlightsPreview SynthesisHighlightsPreview `json:"highlights_preview"`
	Breakdowns        SynthesisBreakdowns        `json:"breakdowns"`

	// Bloc détails (P9)
	DetailedStats SynthesisDetailedStats `json:"detailed_stats"`

	// Bloc frags par arme (top 20, label résolu, weapon ID non-résolu exclus)
	TopWeaponKills []SynthesisWeaponKillEntry `json:"top_weapon_kills,omitempty"`

	// Bloc répartition hiérarchique des frags (sunburst v2, classe→rôle). Title-
	// agnostic, réconcilié (Σ classes == total). nil si scope vide (total 0). Voir
	// FragDistribution (frag_distribution.go) + PLAN_FRAG_DISTRIBUTION_V2.md §2.
	FragDistribution *FragDistribution `json:"frag_distribution,omitempty"`

	// Bloc précision par arme (toutes les armes tirées, pourcentage = tirs au but
	// / tirs tirés). Alimenté par la table weapon_accuracy (Halo 5 natif). Omis
	// (nil → absent) pour les titres qui ne peuplent pas cette donnée (Infinite).
	WeaponAccuracy []SynthesisWeaponAccuracyEntry `json:"weapon_accuracy,omitempty"`

	// Bloc profil combat (OC + DR + descripteurs) — nil si < 15 matchs dans le scope.
	// Ref : PLAN_COMBAT_PROFILE_WIRING.md Phase 1.
	CombatProfile *CombatProfileBlock `json:"combat_profile,omitempty"`

	// Bloc KPI objectifs (cumul CTF/Zones/Oddball sur le scope) — nil pour un titre
	// sans capability match.objective.stats (Halo 5) ou un scope sans match à objectif.
	// Gated (registry SynthesisCtx) + data-driven (front n'affiche que les KPI > 0).
	// Cf. PLAN_V72_OBJECTIVE_STATS.md.
	ObjectiveStats *ObjectiveAggregate `json:"objective_stats,omitempty"`
}

// SynthesisWeaponKillEntry est une ligne du classement frags par arme. Class/Role
// (clés canoniques du registre, cf. frag_distribution.go) permettent au breakdown
// par arme du front de recolorer chaque barre par la couleur de sa CLASSE
// (fragClassColor) — cohérence visuelle avec le sunburst. Vides si le registre n'a
// pas résolu l'arme (dégradation propre ; omis du JSON).
type SynthesisWeaponKillEntry struct {
	Label string `json:"label"`
	Kills int    `json:"kills"`
	Class string `json:"class,omitempty"`
	Role  string `json:"role,omitempty"`
}

// SynthesisWeaponAccuracyEntry est une ligne du classement précision par arme.
// Accuracy = ShotsLanded / ShotsFired, en unité 0..1 (convention API canonique,
// cf. ADR 0006) ; le front multiplie par 100 pour l'affichage en pourcentage.
type SynthesisWeaponAccuracyEntry struct {
	Label       string  `json:"label"`
	ShotsFired  int     `json:"shots_fired"`
	ShotsLanded int     `json:"shots_landed"`
	Accuracy    float64 `json:"accuracy"`
}

// ---------------------------------------------------------------------------
// Bloc previews — Highlights (D5)
// ---------------------------------------------------------------------------

// SynthesisMatchHighlight est un match notable (top/pire) dans la Synthèse.
type SynthesisMatchHighlight struct {
	MatchID   string   `json:"match_id"`
	Kills     int      `json:"kills"`
	Deaths    int      `json:"deaths"`
	KDA       *float64 `json:"kda,omitempty"`
	Outcome   int      `json:"outcome"` // 2=WIN, 3=LOSS
	PerfScore *float64 `json:"perf_score,omitempty"`
}

// SynthesisHighlightsPreview contient les matchs remarquables extraits du scope.
type SynthesisHighlightsPreview struct {
	TopByKills    []SynthesisMatchHighlight `json:"top_by_kills"`
	TopByKDA      []SynthesisMatchHighlight `json:"top_by_kda"`
	WorstByDeaths []SynthesisMatchHighlight `json:"worst_by_deaths"`
}

// ---------------------------------------------------------------------------
// Bloc previews — Breakdowns (D7)
// ---------------------------------------------------------------------------

// SynthesisMapEntry est une ligne agrégée par carte dans la Synthèse.
type SynthesisMapEntry struct {
	MapName    string  `json:"map_name"`
	MatchCount int     `json:"match_count"`
	Wins       int     `json:"wins"`
	Losses     int     `json:"losses"`
	Ties       int     `json:"ties"`
	Unfinished int     `json:"unfinished"`
	WinRate    float64 `json:"win_rate"`
}

// SynthesisModeEntry est une ligne agrégée par mode de jeu dans la Synthèse.
type SynthesisModeEntry struct {
	ModeName   string  `json:"mode_name"`
	MatchCount int     `json:"match_count"`
	Wins       int     `json:"wins"`
	Losses     int     `json:"losses"`
	Ties       int     `json:"ties"`
	Unfinished int     `json:"unfinished"`
	WinRate    float64 `json:"win_rate"`
}

// SynthesisBreakdowns contient les breakdowns carte et mode du scope.
type SynthesisBreakdowns struct {
	TopMaps  []SynthesisMapEntry  `json:"top_maps"`
	TopModes []SynthesisModeEntry `json:"top_modes"`
}

// ---------------------------------------------------------------------------
// Bloc détails — SynthesisDetailedStats (P9 KPI Grid)
// ---------------------------------------------------------------------------

// SynthesisDetailedStats contient les métriques détaillées par catégories.
// Combat, Tir, Dégâts, Fun stats extraits du scope filtré.
type SynthesisDetailedStats struct {
	// Combat
	TotalHeadshotKills    int `json:"total_headshot_kills"`
	TotalPerfectKills     int `json:"total_perfect_kills"`
	TotalGrenadeKills     int `json:"total_grenade_kills"`
	TotalMeleeKills       int `json:"total_melee_kills"`
	TotalPowerWeaponKills int `json:"total_power_weapon_kills"`

	// Mécaniques de kill NATIVES Halo 5 (assassinats + compétences spartiate :
	// ground pound, shoulder bash). 0 pour les titres qui ne les fournissent pas
	// (ex. Infinite) — le gating d'affichage se fait côté front via la capability
	// du titre, jamais via la valeur (0 assassinat reste une valeur légitime).
	TotalAssassinations    int `json:"total_assassinations"`
	TotalGroundPoundKills  int `json:"total_ground_pound_kills"`
	TotalShoulderBashKills int `json:"total_shoulder_bash_kills"`

	// MaxKillingSpree : MAX du max killing spree sur le scope. nil quand le titre ne
	// porte pas ce champ (Halo 5, games.ProvidesMaxKillingSpree=false) → le front
	// masque la stat au lieu d'afficher un 0 trompeur. omitempty pour distinguer nil
	// (masquer) de 0 (réel, aucun spree atteint).
	MaxKillingSpree *int `json:"max_killing_spree,omitempty"` // MAX sur le scope

	// Temps de jeu
	TotalTimePlayedSeconds int `json:"total_time_played_seconds"`

	// Tir
	TotalShotsFired int `json:"total_shots_fired"`
	TotalShotsHit   int `json:"total_shots_hit"`

	// Dégâts
	TotalDamageDealt float64 `json:"total_damage_dealt"`
	TotalDamageTaken float64 `json:"total_damage_taken"`

	// Fun (via personal_score_awards)
	TotalBetrayals         int `json:"total_betrayals"`
	TotalSuicides          int `json:"total_suicides"`
	TotalVehiclesDestroyed int `json:"total_vehicles_destroyed"`
	TotalHijacks           int `json:"total_hijacks"`
}

// ---------------------------------------------------------------------------
// Lignes brutes DuckDB — Synthèse déplacées vers `internal/legacymatch`
// (P4.3 finale cleanup).
