// Package domain — squad_v2.go : DTOs pour la nouvelle version de la page
// Squad construite sur les fondations Phase 0 (PLAN_META_FOUNDATIONS_GO).
//
// Vit en parallèle de squad.go (legacy, mono-coéquipier) jusqu'à migration
// complète des consommateurs frontend (cf. PLAN_SQUAD_GO_PORTAGE).
package domain

import (
	"time"

	"levelup/go-api/internal/games/canonical"
)

// SquadPageV2Response est le DTO de la page Squad V2.
//
// Phase 1 chunks S1-S11 : structure complete avec en-tete + 17 charts +
// 3 tableaux. Capabilities porte les CapabilityGap rencontrees (joueurs avec
// capability match.history absente, sections impossibles a remplir) pour que
// le frontend affiche un <CapabilityGap mode="placeholder|cta"> approprie.
//
// Les sections charts/tables sont nilables : un nil signifie "non charge"
// (capability absente ou erreur) — le front affiche un <CapabilityGap>.
// Une section presente mais vide (Datapoints=nil ou len=0) signifie "calcul
// effectue mais aucune donnee a afficher" (front affiche un empty state).
type SquadPageV2Response struct {
	MainPlayer         string             `json:"main_player"`
	Teammates          []string           `json:"teammates"`
	Period             string             `json:"period"`
	SharedMatchesCount int                `json:"shared_matches_count"`
	SharedMatches      []SquadSharedMatch `json:"shared_matches"`

	// Header : KPIs personnels du joueur principal + score d'equipe + cartes
	// individuelles (chunk S2). Nil si SharedMatches est vide ou si capability
	// gap principal.
	Header *SquadHeader `json:"header,omitempty"`

	// Charts : payloads des 17 charts Squad V2 regroupes par onglet/section.
	Charts *SquadCharts `json:"charts,omitempty"`

	// Tables : tableaux historique + armes + galerie medailles (chunk S9).
	Tables *SquadTables `json:"tables,omitempty"`

	// Capabilities reprend canonical.CapabilityGap pour signaler les sections
	// degradees ou absentes (events non charges, weapons repo absent, etc.).
	Capabilities []canonical.CapabilityGap `json:"capabilities,omitempty"`
}

// SquadCharts regroupe tous les payloads chart Squad V2 par onglet.
type SquadCharts struct {
	// Onglet Synergies (chunks S3+S4)
	MapBreakdownLollipop *ChartSeries[ChartPointStacked] `json:"map_breakdown_lollipop,omitempty"`
	BulletWinrate        *ChartSeries[ChartPointStacked] `json:"bullet_winrate,omitempty"`
	PerfVsHistorical     *ChartSeries[ChartPoint2D]      `json:"perf_vs_historical,omitempty"`
	HeatmapPlayerMap     *ChartSeries[ChartPointHeatmap] `json:"heatmap_player_map,omitempty"`
	TimelineMultiPlayer  []ChartSeries[ChartPoint2D]     `json:"timeline_multi_player,omitempty"`
	FormScore            *ChartSeries[ChartPoint2D]      `json:"form_score,omitempty"`

	// Onglet Synergies suite (chunks S6 — Cadence + Intensite)
	Cadence          *ChartSeries[ChartPointStacked] `json:"cadence,omitempty"`
	IntensityHeatmap *ChartSeries[ChartPointHeatmap] `json:"intensity_heatmap,omitempty"`

	// Onglet Impact (chunk S5) — payload custom (ImpactRolesMatrix + ranking).
	ImpactMatrix  *ImpactRolesMatrix `json:"impact_matrix,omitempty"`
	ImpactRanking []ImpactRanking    `json:"impact_ranking,omitempty"`

	// Onglet Contributions (chunk S7)
	PerMinuteStats        *ChartSeries[ChartPointStacked] `json:"per_minute_stats,omitempty"`
	FragsDeathsCombined   *ChartSeries[ChartPointStacked] `json:"frags_deaths_combined,omitempty"`
	HsPkStacked           *ChartSeries[ChartPointStacked] `json:"hs_pk_stacked,omitempty"`
	KillingSpreeMax       []ChartSeries[ChartPoint2D]     `json:"killing_spree_max,omitempty"`
	AssistsTimeseries     []ChartSeries[ChartPoint2D]     `json:"assists_timeseries,omitempty"`
	KDATimeseries         []ChartSeries[ChartPoint2D]     `json:"kda_timeseries,omitempty"`
	AccuracyTimeseries    []ChartSeries[ChartPoint2D]     `json:"accuracy_timeseries,omitempty"`
	AvgLifeTimeseries     []ChartSeries[ChartPoint2D]     `json:"avg_life_timeseries,omitempty"`
	PerformanceTimeseries []ChartSeries[ChartPoint2D]     `json:"performance_timeseries,omitempty"`

	// Onglet Contributions suite (chunk S8 — Radar 6 axes).
	// Type opaque (any) car le payload Radar est defini cote service
	// (RadarChartSeries) — laisse le service amont serializer.
	Radar []any `json:"radar,omitempty"`
}

// SquadTables regroupe les tableaux + galerie de la page Squad V2 (chunk S9).
//
// History : ligne par match partage, K/D/A par joueur (gamertag).
// Weapons : agregation par arme x joueur, tri total desc.
// Medals : 1 entree par match × joueur, medailles triees count desc.
//
// Les types concrets (HistoryTableRow, WeaponsTableRow, MedalsGalleryEntry)
// sont definis cote service. Comme ces structs ne sont pas dans le package
// domain (ils consomment port.WeaponKillRow et port.MedalRow), on expose
// des `any` — le service amont serialize en JSON correctement.
type SquadTables struct {
	History []any `json:"history,omitempty"`
	Weapons []any `json:"weapons,omitempty"`
	Medals  []any `json:"medals,omitempty"`
}

// SquadHeader regroupe les blocs en-tete de la page Squad (cf. audit § 2 :
// "Bandeau Mes stats" + "Score d'equipe + scores individuels").
type SquadHeader struct {
	// SoloKPIs : stats personnelles du joueur principal sur le scope courant
	// (apres filtres). 8 cartes affichees.
	SoloKPIs *KPIStats `json:"solo_kpis,omitempty"`
	// AllTimeKPIs : stats personnelles du joueur principal sur tout l'historique.
	// Sert de reference pour les fleches de tendance (▲▼) sur SoloKPIs.
	AllTimeKPIs *KPIStats `json:"all_time_kpis,omitempty"`
	// SquadScore : score d'equipe (base + bonus + grade lettre).
	SquadScore *SquadScoreCard `json:"squad_score,omitempty"`
	// PlayerCards : 1 carte par joueur (main + coequipiers) avec score + ▲▼ vs avg.
	PlayerCards []PlayerScoreCard `json:"player_cards,omitempty"`
	// KPIsByXUID : KPIs par xuid sur le scope courant (drill-down SessionBriefing).
	// Cle = xuid (main + coequipiers). Pre-calcule au meme moment que PlayerCards.
	KPIsByXUID map[string]*KPIStats `json:"kpis_by_xuid,omitempty"`
	// TeamAvgKPIs : moyenne arithmetique field-by-field des KPIsByXUID.
	// Sert de reference pour les fleches de tendance (▲▼) du SessionBriefing
	// (PAS l'historique all-time — la comparaison est intra-session).
	TeamAvgKPIs *KPIStats `json:"team_avg_kpis,omitempty"`
}

// KPIStats agrege les indicateurs personnels affiches dans le bandeau header.
type KPIStats struct {
	MatchesCount     int     `json:"matches_count"`
	TotalPlaySeconds int64   `json:"total_play_seconds"`
	AvgMatchSeconds  float64 `json:"avg_match_seconds"`
	KillsPerGame     float64 `json:"kills_per_game"`
	KillsPerMinute   float64 `json:"kills_per_minute"`
	DeathsPerGame    float64 `json:"deaths_per_game"`
	DeathsPerMinute  float64 `json:"deaths_per_minute"`
	AssistsPerGame   float64 `json:"assists_per_game"`
	AssistsPerMinute float64 `json:"assists_per_minute"`
	AvgAccuracy      float64 `json:"avg_accuracy"` // 0..100
	AvgLifeSeconds   float64 `json:"avg_life_seconds"`
	// RankDelta : delta de skill rating sur le scope (somme signee des
	// per-match deltas). Kind = "csr" ou "lusr" — exclusifs au sein d'un
	// scope cohérent (une session est soit classee soit non, par construction
	// metier). Nil si aucun match avec rating dans le scope. C'est le bucket
	// MAJORITAIRE (cf. RankDeltas pour le détail par type).
	RankDelta *RankDelta `json:"rank_delta,omitempty"`
	// RankDeltas : un delta agrégé par type de rating rencontré dans le scope
	// (CSR et LUSR séparés, jamais fusionnés). Ordre déterministe : type
	// majoritaire d'abord, puis par Count décroissant, tie-break Kind. Vide si
	// aucun match avec rating. Consommé par le module « Classement » du briefing
	// Explorer (une ligne par type) ; RankDelta reste le majoritaire pour les
	// consommateurs qui n'en veulent qu'un.
	RankDeltas []RankDelta `json:"rank_deltas,omitempty"`
	// PerformanceScore : score 0-100 du joueur sur le scope filtre.
	// Moyenne des performance_score par match (sync). Nil si aucun match enrichi.
	PerformanceScore *float64 `json:"performance_score,omitempty"`
	// AvgOffensiveConversion : moyenne du rendement offensif (225*(kills+assists/3)/damage_dealt).
	// Nil si aucun match avec damage_dealt > 0.
	AvgOffensiveConversion *float64 `json:"avg_offensive_conversion,omitempty"`
	// AvgDefensiveResistance : moyenne de la résistance défensive (damage_taken/(225*deaths)).
	// Nil si aucun match avec deaths > 0.
	AvgDefensiveResistance *float64 `json:"avg_defensive_resistance,omitempty"`
	// CombatProfile : profil combat 3 axes avec descripteurs textuels.
	// Nil si < 15 matchs dans le scope ou si aucune donnée dégâts.
	// Ref : PLAN_COMBAT_PROFILE_WIRING.md Phase 2.
	CombatProfile *CombatProfileBlock `json:"combat_profile,omitempty"`
	Outcomes      struct {
		Wins   int `json:"wins"`
		Losses int `json:"losses"`
		Ties   int `json:"ties"`
		DNF    int `json:"dnf"`
	} `json:"outcomes"`
}

// RankDelta agrege le delta de rating sur le scope filtre.
//
// Kind : "csr" (classe) ou "lusr" (non classe). Au sein d'un scope coherent
// (session unique, ou plusieurs sessions du meme type), Kind est uniforme.
// Si le scope mixe les deux types (cas pathologique : period filter qui
// englobe plusieurs sessions de natures differentes), on prend le type
// majoritaire et on ignore l'autre — Count reflete le compte du type retenu.
type RankDelta struct {
	Kind  string  `json:"kind"`  // "csr" | "lusr"
	Value float64 `json:"value"` // somme signee des per-match deltas dans le scope
	Count int     `json:"count"` // nombre de matchs du Kind retenu dans le scope
}

// SquadScoreCard est la carte "Score d'equipe" (base + bonus + grade).
//
// Calcul : base = moyenne des scores individuels, bonus +5 si winrate >60%,
// bonus +5 si min(K/D) > 1.0, bonus +3 si stddev kills < 3. Grade = lettre
// resolu via SCORE_THRESHOLDS (S+/A/B/C/D/F).
type SquadScoreCard struct {
	Score        float64 `json:"score"`          // 0..100, clamped
	Grade        string  `json:"grade"`          // "S+", "A", "B", ...
	BaseAvg      float64 `json:"base_avg"`       // avant bonus
	BonusWinRate int     `json:"bonus_win_rate"` // 0 ou 5
	BonusMinKD   int     `json:"bonus_min_kd"`   // 0 ou 5
	BonusBalance int     `json:"bonus_balance"`  // 0 ou 3
	TeamWinRate  float64 `json:"team_win_rate"`  // 0..1
	MinKD        float64 `json:"min_kd"`
	KillsStdDev  float64 `json:"kills_std_dev"`
}

// PlayerScoreCard est la carte d'un joueur (main + chaque coequipier).
//
// Comparison signale si le joueur tire l'equipe vers le haut ou vers le bas
// par rapport a la moyenne squad : "above" (▲), "below" (▼) ou "near" (=).
type PlayerScoreCard struct {
	XUID       string  `json:"xuid"` // pour matcher avec SquadHeader.KPIsByXUID au click drill-down
	Gamertag   string  `json:"gamertag"`
	Score      float64 `json:"score"`
	Label      string  `json:"label"`      // excellent / good / average / poor / bad
	Comparison string  `json:"comparison"` // above / below / near
	KDRatio    float64 `json:"kd_ratio"`
	WinRate    float64 `json:"win_rate"`
	Accuracy   float64 `json:"accuracy"`
	Kills      int     `json:"kills"`
}

// SquadSharedMatch représente un match commun entre tous les joueurs sélectionnés.
//
// Players[gamertag] donne accès aux stats du joueur sur ce match précis. Les
// champs au niveau de la struct (StartedAt, Map, Outcome) sont hydratés depuis
// le joueur principal — ces données sont identiques pour tous les joueurs du
// match (même match_id).
type SquadSharedMatch struct {
	MatchID   string                              `json:"match_id"`
	StartedAt time.Time                           `json:"started_at_utc"`
	Map       *canonical.AssetReference           `json:"map,omitempty"`
	Mode      *canonical.AssetReference           `json:"mode,omitempty"`
	Playlist  *canonical.AssetReference           `json:"playlist,omitempty"`
	Outcome   canonical.Outcome                   `json:"outcome"`
	Players   map[string]canonical.PlayerMatchRow `json:"players"`
}

// ImpactRolesMatrix porte la heatmap roles 8 x N joueurs (cf. PLAN_SQUAD_GO_PORTAGE
// Phase P5). Chaque cellule rassemble les roles attribues a un xuid sur un match.
type ImpactRolesMatrix struct {
	// MatchRows : 1 entree par match partage (ordre = ordre des SharedMatches).
	MatchRows []ImpactRolesMatchRow `json:"match_rows"`
	// SquadGamertags : ordre stable des colonnes (joueur principal + coequipiers,
	// dans l'ordre d'arrivee sur la page).
	SquadGamertags []string `json:"squad_gamertags"`
}

// ImpactRolesMatchRow est une ligne de la heatmap : un match avec les roles
// par joueur du squad.
type ImpactRolesMatchRow struct {
	MatchID     string            `json:"match_id"`
	StartedAt   time.Time         `json:"started_at_utc"`
	MainOutcome canonical.Outcome `json:"main_outcome"`
	// RolesByPlayer : gamertag -> liste des cles de roles attribues sur ce match.
	// Vide si le joueur n'a recu aucun role sur ce match.
	RolesByPlayer map[string][]ImpactRoleCell `json:"roles_by_player"`
}

// ImpactRoleCell decrit un role attribue (label key + token couleur, etc.).
type ImpactRoleCell struct {
	RoleKey    string `json:"role_key"`    // canonical.ImpactRole (first_blood, top_killer, ...)
	LabelKey   string `json:"label_key"`   // i18n manifest key
	ColorToken string `json:"color_token"` // CSS variable name
	Inverted   bool   `json:"inverted"`    // true pour roles negatifs (couleur opposee)
}

// ImpactRanking represente une colonne du tableau MVP/Boulet (1 par role) :
// classement des joueurs du squad par count desc sur ce role precis.
type ImpactRanking struct {
	RoleKey  string               `json:"role_key"`
	LabelKey string               `json:"label_key"`
	Inverted bool                 `json:"inverted"` // role negatif → gradient couleur inverse
	Entries  []ImpactRankingEntry `json:"entries"`
}

// ImpactRankingEntry est une ligne du ranking pour un role : gamertag + count.
type ImpactRankingEntry struct {
	Gamertag string `json:"gamertag"`
	Count    int    `json:"count"`
}
