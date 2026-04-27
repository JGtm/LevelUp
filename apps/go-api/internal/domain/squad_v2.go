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
// Phase 1 chunk S1 : structure squelette avec uniquement l'intersection des
// matchs partagés. Les sections riches (KPI, score d'équipe, charts synergies,
// impact 8 rôles, radar, etc.) sont remplies par les chunks S2-S11.
//
// Capabilities porte les CapabilityGap rencontrées (joueurs avec capability
// match.history absente, sections impossibles à remplir) pour que le frontend
// affiche un <CapabilityGap mode="placeholder|cta"> approprié.
type SquadPageV2Response struct {
	MainPlayer         string             `json:"main_player"`
	Teammates          []string           `json:"teammates"`
	Period             string             `json:"period"`
	SharedMatchesCount int                `json:"shared_matches_count"`
	SharedMatches      []SquadSharedMatch `json:"shared_matches"`
	// Header porte les KPIs personnels du joueur principal + score d'equipe +
	// cartes individuelles (cf. PLAN_SQUAD_GO_PORTAGE § 1.1, P2). Nil si
	// SharedMatches est vide ou si capability gap principal.
	Header *SquadHeader `json:"header,omitempty"`
	// Capabilities reprend canonical.CapabilityGap (CapabilityKey + ReasonCode +
	// Severity + Message + Retryable) pour signaler les sections degradees.
	Capabilities []canonical.CapabilityGap `json:"capabilities,omitempty"`
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
	Outcomes         struct {
		Wins   int `json:"wins"`
		Losses int `json:"losses"`
		Ties   int `json:"ties"`
		DNF    int `json:"dnf"`
	} `json:"outcomes"`
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
