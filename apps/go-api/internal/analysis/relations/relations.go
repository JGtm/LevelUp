// Package relations — algorithmes purs du hub Communauté > Relations.
//
// 0 DB, 0 HTTP, 0 Streamlit. Entrée : RelationStats (agrégats par joueur
// récurrent). Sortie : badges, catégorie, sélection binôme/bête noire et
// compteurs d'aperçu. Les seuils sont des constantes nommées.
//
// Les badges "existants" (ally_plus / tough_enemy / coriace / ordinal) sont
// délégués à internal/analysis/narrative.ComputeEncounterBadges pour rester
// alignés sur la page Carrière et la Match View. Tous les badges de RELATION
// (existants + nouveaux) sont rendus en style "solid" (homogénéité front).
package relations

import (
	"time"

	"levelup/go-api/internal/analysis/narrative"
)

// Catégories de relation.
const (
	CategoryAlly  = "ally"
	CategoryEnemy = "enemy"
	CategoryMixed = "mixed"
)

// Styles de badge.
const (
	BadgeStyleTinted = "tinted"
	BadgeStyleSolid  = "solid"
)

// Seuils des nouveaux badges (style "solid"). Constantes nommées (pas de magic
// number) — toute modification produit/UX passe ici.
const (
	// duo_gagnant : binôme performant.
	DuoGagnantWinRateThreshold   = 0.60
	DuoGagnantMinTeammateMatches = 10

	// cameleon : joue autant en allié qu'en ennemi.
	CameleonMixRatioThreshold = 0.40
	CameleonMinTotalMatches   = 10

	// de_longue_date : relation ancienne ou très fréquente.
	DeLongueDateMinMonths       = 6
	DeLongueDateMinTotalMatches = 80

	// recrue : relation récente déjà significative.
	RecrueMaxDays         = 30
	RecrueMinTotalMatches = 4

	// proie_favorite : domination nette en duel.
	ProieFavoriteDuelRatioThreshold = 1.5
	ProieFavoriteMinEnemyMatches    = 6

	// cross_game : aussi croisé sur un AUTRE titre (best-effort, additif).
	// Seuil minimal de matchs communs sur l'autre titre pour mériter le badge.
	CrossGameMinMatchesTogether = 3
)

// Seuils des compteurs d'aperçu et de la sélection binôme/bête noire.
const (
	// CoreMinTotalMatches / CoreMinTeammate : "noyau dur" (seuil enemy retiré, cf. IsCore).
	CoreMinTotalMatches = 20
	CoreMinTeammate     = 3

	// TopAllyMinTeammateMatches : seuil pour candidater au binôme.
	TopAllyMinTeammateMatches = 8
	// TopNemesisMinEnemyMatches : seuil pour candidater à la bête noire.
	TopNemesisMinEnemyMatches = 8
)

// daysPerMonth approxime un mois pour le badge de_longue_date (sans calendrier
// exact : 6 mois ≈ 182 jours).
const monthsToDaysApprox = 30.4375

// RelationStats : agrégats par joueur récurrent, indépendants de la DB.
// WinRate* sont nil quand le dénominateur correspondant est nul.
type RelationStats struct {
	XUID            string
	Gamertag        string
	TotalMatches    int
	TeammateMatches int
	EnemyMatches    int
	TeammateWins    int
	TeammateWinRate *float64
	EnemyWins       int
	EnemyWinRate    *float64
	KillsDealt      int
	DeathsSuffered  int
	DuelRatio       *float64
	FirstSeen       *time.Time
	LastSeen        *time.Time
}

// Badge : badge résolu (LabelKey + ColorToken + Style + Detail).
type Badge struct {
	LabelKey   string
	ColorToken string
	Style      string
	Detail     map[string]any
}

// Categorize retourne la catégorie de la relation : mixed si allié ET ennemi,
// sinon ally / enemy selon ce qui domine (égalité → ally).
func Categorize(s RelationStats) string {
	if s.TeammateMatches > 0 && s.EnemyMatches > 0 {
		return CategoryMixed
	}
	if s.EnemyMatches > s.TeammateMatches {
		return CategoryEnemy
	}
	return CategoryAlly
}

// ComputeBadges retourne les badges applicables à une relation : d'abord les
// badges de rencontre existants (ally_plus / tough_enemy / coriace / ordinal),
// puis les nouveaux, dans un ordre stable. Tous en style solid (badges de joueur).
func ComputeBadges(s RelationStats, now time.Time) []Badge {
	out := make([]Badge, 0, 8)
	out = append(out, encounterBadges(s)...)
	out = append(out, solidBadges(s, now)...)
	return out
}

// encounterBadges délègue aux badges narratifs de RENCONTRE (ordinal / ally_plus /
// tough_enemy / coriace) via narrative.ComputeEncounterBadges, en réutilisant
// LabelKey + ColorToken d'origine. Rendus en style solid (badges de joueur,
// homogènes avec les nouveaux). NB : dominance + rôles d'impact ne passent PAS
// par ici et restent teintés.
func encounterBadges(s RelationStats) []Badge {
	ordinal := s.TotalMatches - 1
	if ordinal < 0 {
		ordinal = 0
	}
	raw := narrative.ComputeEncounterBadges(narrative.EncounterStats{
		XUID:            s.XUID,
		Gamertag:        s.Gamertag,
		TotalEncounters: s.TotalMatches,
		AllyCount:       s.TeammateMatches,
		EnemyCount:      s.EnemyMatches,
		WinrateAsAlly:   s.TeammateWinRate,
		WinrateVsEnemy:  s.EnemyWinRate,
		KillsDealt:      s.KillsDealt,
		DeathsSuffered:  s.DeathsSuffered,
		LastSeen:        s.LastSeen,
	}, ordinal)
	out := make([]Badge, 0, len(raw))
	for _, b := range raw {
		out = append(out, Badge{
			LabelKey:   b.LabelKey,
			ColorToken: b.ColorToken,
			Style:      BadgeStyleSolid,
			Detail:     b.Detail,
		})
	}
	return out
}
