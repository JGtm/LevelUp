package narrative

import "time"

// EncounterStats represente les stats agregees d'un joueur rencontre dans un
// historique de matchs partages (ally + enemy mix).
type EncounterStats struct {
	XUID            string
	Gamertag        string
	TotalEncounters int
	AllyCount       int
	EnemyCount      int
	WinrateAsAlly   *float64 // nil si AllyCount == 0
	WinrateVsEnemy  *float64 // nil si EnemyCount == 0
	KillsDealt      int      // kills du joueur courant CONTRE cet adversaire
	DeathsSuffered  int      // kills de cet adversaire CONTRE le joueur courant
	LastSeen        *time.Time
}

// EncounterKind classe le type de badge attribue a une rencontre.
// Mappés 1:1 sur les badges Python historiques (cf.
// src/ui/pages/match_view_encounters_logic.py sur main).
type EncounterKind string

const (
	// EncounterAllyPlus = "Allié+" : coéquipier avec winrate >= 0.65 sur >= 2
	// matchs en allié. Mirror de Python `badge_ally_plus`.
	EncounterAllyPlus EncounterKind = "ally_plus"

	// EncounterToughEnemy = "Dur à cuire" : adversaire dont le K/D contre le
	// joueur courant > 2.0 sur >= 3 morts subies. Cas particulier KillsDealt=0
	// avec DeathsSuffered >= 3 → qualifie d'office. Mirror de Python
	// `badge_tough_nut`. Pas de check EnemyCount ici (la garde est sur
	// DeathsSuffered >= 3 directement).
	EncounterToughEnemy EncounterKind = "tough_enemy"

	// EncounterCoriace = "Coriace" : adversaire face auquel mon winrate est
	// faible (<= 0.35) sur >= 3 matchs en ennemi. Mirror de Python
	// `badge_coriace`. Distinct de tough_enemy (qui est K/D-based).
	EncounterCoriace EncounterKind = "coriace"

	// EncounterOrdinal indique combien de fois la rencontre s'est repetee.
	// Ajoute systematiquement quand ordinal > 0.
	EncounterOrdinal EncounterKind = "ordinal"
)

// Seuils utilisés par ComputeEncounterBadges — alignés Python (main).
// Constantes exportées pour permettre override en tests.
const (
	// AllyPlusWinrateThreshold : winrate min en allié pour qualifier
	// (`>=`, pas `>`). Python : 0.65.
	AllyPlusWinrateThreshold = 0.65
	// MinAllyCountForBadge : nombre min de matchs en allié. Python : 2.
	MinAllyCountForBadge = 2

	// ToughEnemyKDThreshold : ratio (DeathsSuffered/KillsDealt) min strict
	// (`>`). Python : 2.0.
	ToughEnemyKDThreshold = 2.0
	// MinDeathsForToughEnemy : morts subies min. Python : 3.
	MinDeathsForToughEnemy = 3

	// CoriaceWinrateThreshold : winrate max en ennemi (`<=`). Python : 0.35.
	CoriaceWinrateThreshold = 0.35
	// MinEnemyCountForCoriace : nombre min de matchs en ennemi. Python : 3.
	MinEnemyCountForCoriace = 3
)

// EncounterBadge est un badge d'encounter resolu (ColorToken + LabelKey).
type EncounterBadge struct {
	Kind       EncounterKind
	LabelKey   string
	ColorToken string
	Detail     map[string]any // payload contextuel (winrate, kd_against_me, ordinal)
}

// ComputeEncounterBadges retourne les badges applicables aux stats fournies.
// Si ordinal > 0, EncounterOrdinal est toujours present en premier.
//
// Ordre stable de sortie :
//
//	[Ordinal] (si ordinal > 0)
//	[AllyPlus]
//	[ToughEnemy]
//	[Coriace]
func ComputeEncounterBadges(stats EncounterStats, ordinal int) []EncounterBadge {
	badges := make([]EncounterBadge, 0, 4)
	if ordinal > 0 {
		badges = append(badges, EncounterBadge{
			Kind:       EncounterOrdinal,
			LabelKey:   "narrative.encounter.ordinal",
			ColorToken: "narrative-encounter-ordinal",
			Detail:     map[string]any{"ordinal": ordinal},
		})
	}
	if badge := allyPlusBadge(stats); badge != nil {
		badges = append(badges, *badge)
	}
	if badge := toughEnemyBadge(stats); badge != nil {
		badges = append(badges, *badge)
	}
	if badge := coriaceBadge(stats); badge != nil {
		badges = append(badges, *badge)
	}
	return badges
}

// allyPlusBadge : Python `badge_ally_plus` — winrate >= 0.65 ET ally_count >= 2.
func allyPlusBadge(stats EncounterStats) *EncounterBadge {
	if stats.WinrateAsAlly == nil {
		return nil
	}
	if stats.AllyCount < MinAllyCountForBadge {
		return nil
	}
	if *stats.WinrateAsAlly < AllyPlusWinrateThreshold {
		return nil
	}
	return &EncounterBadge{
		Kind:       EncounterAllyPlus,
		LabelKey:   "narrative.encounter.ally_plus",
		ColorToken: "narrative-encounter-ally-plus",
		Detail:     map[string]any{"winrate": *stats.WinrateAsAlly},
	}
}

// toughEnemyBadge : Python `badge_tough_nut` (label "Dur à cuire").
//
//	deaths_suffered >= 3 AND kills_dealt > 0 AND deaths/kills > 2.0
//	OU deaths_suffered >= 3 AND kills_dealt == 0
//
// Pas de check EnemyCount (Python n'en a pas) — la garde MinDeathsForToughEnemy
// suffit pour exclure les badges sur un seul match malheureux.
func toughEnemyBadge(stats EncounterStats) *EncounterBadge {
	if stats.DeathsSuffered < MinDeathsForToughEnemy {
		return nil
	}
	var kdAgainstMe float64
	var qualifies bool
	if stats.KillsDealt > 0 {
		kdAgainstMe = float64(stats.DeathsSuffered) / float64(stats.KillsDealt)
		qualifies = kdAgainstMe > ToughEnemyKDThreshold
	} else {
		// KillsDealt == 0 : K/D infini, qualifie d'office (DeathsSuffered >= 3
		// déjà validé ci-dessus).
		kdAgainstMe = -1 // sentinelle
		qualifies = true
	}
	if !qualifies {
		return nil
	}
	detail := map[string]any{
		"deaths_suffered": stats.DeathsSuffered,
		"kills_dealt":     stats.KillsDealt,
	}
	if kdAgainstMe >= 0 {
		detail["kd_against_me"] = kdAgainstMe
	}
	return &EncounterBadge{
		Kind:       EncounterToughEnemy,
		LabelKey:   "narrative.encounter.tough_enemy",
		ColorToken: "narrative-encounter-tough-enemy",
		Detail:     detail,
	}
}

// coriaceBadge : Python `badge_coriace` — winrate_vs_enemy <= 0.35 ET enemy_count >= 3.
func coriaceBadge(stats EncounterStats) *EncounterBadge {
	if stats.WinrateVsEnemy == nil {
		return nil
	}
	if stats.EnemyCount < MinEnemyCountForCoriace {
		return nil
	}
	if *stats.WinrateVsEnemy > CoriaceWinrateThreshold {
		return nil
	}
	return &EncounterBadge{
		Kind:       EncounterCoriace,
		LabelKey:   "narrative.encounter.coriace",
		ColorToken: "narrative-encounter-coriace",
		Detail: map[string]any{
			"winrate":     *stats.WinrateVsEnemy,
			"enemy_count": stats.EnemyCount,
		},
	}
}
