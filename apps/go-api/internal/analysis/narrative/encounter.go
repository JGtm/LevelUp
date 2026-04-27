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
type EncounterKind string

// Constantes des badges d'encounter. Chaque kind a un seuil et une regle
// d'attribution documentes ci-dessous.
const (
	// EncounterAllyPlus : coequipier avec winrate > AllyPlusWinrateThreshold
	// sur au moins MinEncountersForBadge matchs en commun comme allies.
	EncounterAllyPlus EncounterKind = "ally_plus"

	// EncounterToughEnemy : adversaire dont le K/D contre le joueur courant
	// excede ToughEnemyKDThreshold (= DeathsSuffered / KillsDealt > seuil),
	// sur au moins MinEncountersForBadge rencontres comme ennemi.
	// Cas particulier : si KillsDealt == 0 et DeathsSuffered >= MinEncountersForBadge,
	// considere comme tough_enemy d'office (le joueur ne le tue jamais).
	EncounterToughEnemy EncounterKind = "tough_enemy"

	// EncounterOrdinal indique combien de fois la rencontre s'est repetee.
	// Ajoute systematiquement quand ordinal > 0.
	EncounterOrdinal EncounterKind = "ordinal"
)

// Seuils utilises par ComputeEncounterBadges. Constantes pour permettre
// override en tests.
const (
	AllyPlusWinrateThreshold = 0.7
	ToughEnemyKDThreshold    = 1.5
	MinEncountersForBadge    = 3
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
// Les badges sont retournes dans un ordre stable :
//
//	[Ordinal] (si ordinal > 0)
//	[AllyPlus]
//	[ToughEnemy]
func ComputeEncounterBadges(stats EncounterStats, ordinal int) []EncounterBadge {
	badges := make([]EncounterBadge, 0, 3)
	if ordinal > 0 {
		badges = append(badges, EncounterBadge{
			Kind:       EncounterOrdinal,
			LabelKey:   "narrative.encounter.ordinal",
			ColorToken: "narrative.encounter.ordinal",
			Detail:     map[string]any{"ordinal": ordinal},
		})
	}
	if stats.AllyCount >= MinEncountersForBadge &&
		stats.WinrateAsAlly != nil &&
		*stats.WinrateAsAlly > AllyPlusWinrateThreshold {
		badges = append(badges, EncounterBadge{
			Kind:       EncounterAllyPlus,
			LabelKey:   "narrative.encounter.ally_plus",
			ColorToken: "narrative.encounter.ally_plus",
			Detail:     map[string]any{"winrate": *stats.WinrateAsAlly},
		})
	}
	if badge := toughEnemyBadge(stats); badge != nil {
		badges = append(badges, *badge)
	}
	return badges
}

// toughEnemyBadge isole la regle complexe de tough_enemy (cas KillsDealt == 0).
func toughEnemyBadge(stats EncounterStats) *EncounterBadge {
	if stats.EnemyCount < MinEncountersForBadge {
		return nil
	}
	if stats.DeathsSuffered == 0 {
		return nil
	}
	var kdAgainstMe float64
	var qualifies bool
	if stats.KillsDealt > 0 {
		kdAgainstMe = float64(stats.DeathsSuffered) / float64(stats.KillsDealt)
		qualifies = kdAgainstMe > ToughEnemyKDThreshold
	} else {
		// Pas de kill contre lui = K/D infini. On qualifie si on a quand meme
		// au moins MinEncountersForBadge morts subies (donnee robuste).
		kdAgainstMe = -1 // sentinelle
		qualifies = stats.DeathsSuffered >= MinEncountersForBadge
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
		ColorToken: "narrative.encounter.tough_enemy",
		Detail:     detail,
	}
}
