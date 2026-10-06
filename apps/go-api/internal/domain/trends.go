// Package domain — trends.go : types de la page Tendances.
//
//	POST /api/v1/players/{slug}/pages/trends → TrendsPageResponse
//
// Une seule réponse porte la matrice (12 mois et 4 horizons), les séries à tous
// les pas, le calendrier et les blocs par horizon : le client découpe sans
// refaire de requête. Les taux et parts sont des ratios 0..1 (ADR 0006).
// Dans une réponse, un tableau n'est jamais nil (vide = []).
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Vues de la page.
const (
	// TrendsViewSolo : matchs joués sans amis (vue par défaut).
	TrendsViewSolo = "solo"
	// TrendsViewSquad : matchs joués avec une composition d'escouade.
	TrendsViewSquad = "squad"
)

// Groupes d'indicateurs (TrendsIndicator.Group).
const (
	TrendsGroupLevel      = "level"
	TrendsGroupResults    = "results"
	TrendsGroupCombat     = "combat"
	TrendsGroupStyle      = "style"
	TrendsGroupObjectives = "objectives"
	TrendsGroupActivity   = "activity"
	TrendsGroupSquad      = "squad"
	TrendsGroupMembers    = "members"
)

// Unités d'un indicateur (TrendsIndicator.Unit).
const (
	TrendsUnitNumber  = "number"
	TrendsUnitRatio   = "ratio"
	TrendsUnitSeconds = "seconds"
	TrendsUnitHours   = "hours"
)

// Groupes de statistiques du bloc « Victoires et défaites » (TrendsWinLossRow.Group).
const (
	TrendsWinLossStats     = "stats"
	TrendsWinLossComposite = "composite"
	TrendsWinLossContext   = "context"
)

// Clés d'indicateur de la vue Solo.
const (
	TrendsKeyEnemyMMR             = "enemy_mmr"
	TrendsKeyTeamMMR              = "team_mmr"
	TrendsKeyCSRValue             = "csr_value"
	TrendsKeyLUSRValue            = "lusr_value"
	TrendsKeyWinRate              = "win_rate"
	TrendsKeyPerformanceScore     = "performance_score"
	TrendsKeyKDA                  = "kda"
	TrendsKeyDamageBalance        = "damage_balance"
	TrendsKeyAccuracy             = "accuracy"
	TrendsKeyAssistsPerMatch      = "assists_per_match"
	TrendsKeyAvgMaxKillingSpree   = "avg_max_killing_spree"
	TrendsKeyDefensiveResistance  = "defensive_resistance"
	TrendsKeyOffensiveConversion  = "offensive_conversion"
	TrendsKeyAvgLifeSeconds       = "avg_life_seconds"
	TrendsKeyHeadshotShare        = "headshot_share"
	TrendsKeyPowerWeaponShare     = "power_weapon_share"
	TrendsKeyEquipmentUsedShare   = "equipment_used_share"
	TrendsKeyObjectiveTakeShare   = "objective_take_share"
	TrendsKeyObjectiveDefendShare = "objective_defend_share"
	TrendsKeyObjectiveHoldShare   = "objective_hold_share"
	TrendsKeyMatchCount           = "match_count"
	TrendsKeyHoursPlayed          = "hours_played"
	TrendsKeyDaysPlayed           = "days_played"
	TrendsKeyDNFRate              = "dnf_rate"
	TrendsKeyKillsPerMatch        = "kills_per_match"
	TrendsKeyDeathsPerMatch       = "deaths_per_match"
	TrendsKeyAvgDamageDealt       = "avg_damage_dealt"
	TrendsKeyAvgDamageTaken       = "avg_damage_taken"
	TrendsKeyObjectiveParity      = "objective_parity"
	TrendsKeyDeaths               = "deaths"
	TrendsKeyKills                = "kills"
	TrendsKeyAssists              = "assists"
	TrendsKeyHeadshotKills        = "headshot_kills"
	TrendsKeyMaxKillingSpree      = "max_killing_spree"
	TrendsKeyMMRGap               = "mmr_gap"

	// Clés propres à la vue Escouade (groupes squad et members ; mmr_gap est partagée avec la vue Solo).
	TrendsKeyWinRateAlone            = "win_rate_alone"
	TrendsKeySquadShareOfTeamKills   = "squad_share_of_team_kills"
	TrendsKeyMemberShareOfSquadKills = "member_share_of_squad_kills"
)

// TrendsQueryRequest est le corps de POST /pages/trends.
type TrendsQueryRequest struct {
	// View : TrendsViewSolo (défaut) ou TrendsViewSquad.
	View string `json:"view,omitempty"`
	// GameType : chaîne de performance à isoler ; vide = tous les types.
	GameType string `json:"game_type,omitempty"`
	// SelectedGamertags : membres de l'escouade (vue Escouade).
	SelectedGamertags []string `json:"selected_gamertags,omitempty"`
	// ExactComposition : composition stricte (vue Escouade).
	ExactComposition bool `json:"exact_composition,omitempty"`
	// Locale : langue des noms de médailles.
	Locale string `json:"locale,omitempty"`
}

// TrendsPageResponse est la réponse de POST /pages/trends.
type TrendsPageResponse struct {
	AsOf         time.Time            `json:"as_of"`
	View         string               `json:"view"`
	GameType     string               `json:"game_type"`
	Timezone     string               `json:"timezone"`
	GameTypes    []TrendsGameType     `json:"game_types"`
	Capabilities TrendsCapabilities   `json:"capabilities"`
	Months       []string             `json:"months"`
	Indicators   []TrendsIndicator    `json:"indicators"`
	Calendar     []TrendsCalendarDay  `json:"calendar"`
	WinLoss      []TrendsWinLossBlock `json:"win_loss"`
	Medals       []TrendsMedalsBlock  `json:"medals"`
	Mix          TrendsMix            `json:"mix"`
	// Members : membres de l'escouade (joueur principal d'abord) ; vide en vue Solo.
	Members []TrendsMember `json:"members"`
}

// TrendsGameType est une chaîne de performance jouée sur 365 jours.
type TrendsGameType struct {
	Key     string `json:"key"`
	Matches int    `json:"matches"`
}

// TrendsCapabilities indique ce que le titre sait fournir (jamais le slug).
type TrendsCapabilities struct {
	MMR        bool `json:"mmr"`
	CSR        bool `json:"csr"`
	LUSR       bool `json:"lusr"`
	Objectives bool `json:"objectives"`
	Equipment  bool `json:"equipment"`
}

// TrendsIndicator est une ligne de la matrice et ses séries.
//
// Variant précise la déclinaison (groupe de classement CSR / LUSR, gamertag) ;
// vide sinon. Better vaut 1 (plus haut = mieux), -1 (plus bas = mieux) ou 0
// (neutre). Months compte 12 cellules et Horizons 4 (365, 90, 30, 7 jours).
type TrendsIndicator struct {
	Key      string              `json:"key"`
	Variant  string              `json:"variant,omitempty"`
	Group    string              `json:"group"`
	Unit     string              `json:"unit"`
	Decimals int                 `json:"decimals"`
	Better   int                 `json:"better"`
	InMatrix bool                `json:"in_matrix"`
	Months   []TrendsMonthCell   `json:"months"`
	Horizons []TrendsHorizonCell `json:"horizons"`
	Series   TrendsSeries        `json:"series"`
}

// TrendsMonthCell est la valeur d'un mois local (absente sous 5 matchs).
type TrendsMonthCell struct {
	Value   *float64 `json:"value,omitempty"`
	Matches int      `json:"matches"`
	Z       *float64 `json:"z,omitempty"`
}

// TrendsHorizonCell compare un horizon à la période d'avant de même durée.
// PrevValue et Z ne sont renseignés que si les deux fenêtres comptent au moins
// 10 matchs.
type TrendsHorizonCell struct {
	Days        int      `json:"days"`
	Value       *float64 `json:"value,omitempty"`
	Matches     int      `json:"matches"`
	PrevValue   *float64 `json:"prev_value,omitempty"`
	PrevMatches int      `json:"prev_matches"`
	Z           *float64 `json:"z,omitempty"`
}

// TrendsSeries porte les courbes d'un indicateur aux quatre pas.
type TrendsSeries struct {
	Match []TrendsPoint `json:"match"`
	Day   []TrendsPoint `json:"day"`
	Week  []TrendsPoint `json:"week"`
	Month []TrendsPoint `json:"month"`
}

// TrendsPoint est un point de série, daté du début de son intervalle.
type TrendsPoint struct {
	T       time.Time `json:"t"`
	Value   float64   `json:"value"`
	Matches int       `json:"matches"`
}

// TrendsCalendarDay résume un jour local joué.
type TrendsCalendarDay struct {
	Date             string   `json:"date"`
	Matches          int      `json:"matches"`
	Wins             int      `json:"wins"`
	Losses           int      `json:"losses"`
	WinRate          *float64 `json:"win_rate,omitempty"`
	PerformanceScore *float64 `json:"performance_score,omitempty"`
}

// TrendsWinLossBlock est le bloc « Victoires et défaites » d'un horizon. Rows
// est vide tant que Matches est sous Required.
type TrendsWinLossBlock struct {
	Days     int                `json:"days"`
	Matches  int                `json:"matches"`
	Required int                `json:"required"`
	Rows     []TrendsWinLossRow `json:"rows"`
}

// TrendsWinLossRow compare une statistique en victoire et en défaite. ZWin et
// ZLoss sont exprimés en écarts-types de la statistique sur l'horizon ; R est
// la corrélation de Pearson entre la statistique et l'issue.
type TrendsWinLossRow struct {
	Key      string  `json:"key"`
	Group    string  `json:"group"`
	Matches  int     `json:"matches"`
	WinMean  float64 `json:"win_mean"`
	LossMean float64 `json:"loss_mean"`
	ZWin     float64 `json:"z_win"`
	ZLoss    float64 `json:"z_loss"`
	R        float64 `json:"r"`
}

// TrendsMedalsBlock liste les médailles d'un horizon. Compared vaut faux sans
// période d'avant comparable.
type TrendsMedalsBlock struct {
	Days     int              `json:"days"`
	Compared bool             `json:"compared"`
	Rows     []TrendsMedalRow `json:"rows"`
}

// TrendsMedalRow est le taux par match d'une médaille.
type TrendsMedalRow struct {
	MedalID  int64    `json:"medal_id"`
	Name     string   `json:"name"`
	Rate     float64  `json:"rate"`
	PrevRate *float64 `json:"prev_rate,omitempty"`
}

// TrendsMix compte les matchs par type de partie à trois pas.
type TrendsMix struct {
	Day   []TrendsMixBucket `json:"day"`
	Week  []TrendsMixBucket `json:"week"`
	Month []TrendsMixBucket `json:"month"`
}

// TrendsMixBucket est le décompte d'un intervalle, par chaîne de performance.
type TrendsMixBucket struct {
	T      time.Time      `json:"t"`
	Counts map[string]int `json:"counts"`
}

// TrendsGameTypeOther est la clé d'un type de partie sans chaîne de performance.
const TrendsGameTypeOther = "other"

// Bornes de validation de TrendsQueryRequest.
const (
	trendsMaxSelectedGamertags = 3
	trendsMaxGameTypeLen       = 64
)

// Validate contrôle le corps de la requête. La vue Escouade exige 1 à 3 gamertags
// non vides ; en vue Solo la sélection est ignorée mais reste bornée à 3. Un type
// de partie inconnu du joueur n'est pas une erreur : la page revient simplement
// sans indicateur.
func (r TrendsQueryRequest) Validate() error {
	switch r.View {
	case "", TrendsViewSolo:
	case TrendsViewSquad:
		if len(r.SelectedGamertags) == 0 {
			return errors.New("la vue Escouade exige au moins un gamertag")
		}
		for _, g := range r.SelectedGamertags {
			if strings.TrimSpace(g) == "" {
				return errors.New("gamertag vide dans la sélection")
			}
		}
	default:
		return fmt.Errorf("vue inconnue %q", r.View)
	}
	if aDesDoublons(r.SelectedGamertags) {
		return errors.New("gamertag en double dans la sélection")
	}
	if len(r.SelectedGamertags) > trendsMaxSelectedGamertags {
		return fmt.Errorf("au plus %d gamertags sélectionnés", trendsMaxSelectedGamertags)
	}
	if len(r.GameType) > trendsMaxGameTypeLen {
		return fmt.Errorf("type de partie trop long (%d caractères au plus)", trendsMaxGameTypeLen)
	}
	for _, c := range r.GameType {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return errors.New("type de partie : lettres minuscules, chiffres et _ uniquement")
		}
	}
	return nil
}

// TrendsMember est un membre de la composition de la vue Escouade.
type TrendsMember struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`
}

// aDesDoublons indique si deux gamertags sont égaux, espaces de bord retirés et sans
// tenir compte de la casse.
func aDesDoublons(gamertags []string) bool {
	seen := make(map[string]struct{}, len(gamertags))
	for _, g := range gamertags {
		k := strings.ToLower(strings.TrimSpace(g))
		if _, dup := seen[k]; dup {
			return true
		}
		seen[k] = struct{}{}
	}
	return false
}
