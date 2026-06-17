package canonical

import "time"

// explorer.go — types canoniques cross-joueur / profil-cible de l'Explorer
// (Phase 2 HIGH-B). Concepts PvP universels (intersection 2 joueurs, kills croisés,
// profil de combat récent, agrégat de stats sur un set de matchs) qu'aucun type
// canonique mono-joueur existant ne couvre.

// PlayerMatchSetStats est l'agrégat des stats d'un joueur sur un ensemble de matchs
// fourni (Explorer sample stats, Phase 2 HIGH-B). Porteur 1:1 de
// domain.ParticipantStatsAggregate — somme de primitives de scoreboard universelles.
// DamageDealt/DamageTaken restent en float64 (tronquer en int casserait
// ComputeCombatYield → OffensiveConversion/DefensiveResistance).
type PlayerMatchSetStats struct {
	Kills             int
	Deaths            int
	Assists           int
	Wins              int
	Losses            int
	Draws             int
	ShotsFired        int
	ShotsHit          int
	DamageDealt       float64
	DamageTaken       float64
	HeadshotKills     int
	MeleeKills        int
	PowerWeaponKills  int
	GrenadeKills      int
	TimePlayedSeconds int
	PersonalScore     int
}

// RecentMatchRow est un match récent du profil de combat d'un joueur cible
// (Explorer), dans le canonique. Porteur 1:1 de domain.ExplorerTargetRecentMatch
// SAUF ModePairAssetID (indice de source LIVE transient, json:"-", hors surface
// byte-identique — vide sur le chemin local). Outcome reste le code BRUT (2/3/1/4)
// comme CareerTopMatch (canonical.Outcome string serait lossy pour 0/unknown).
type RecentMatchRow struct {
	MatchID         string
	StartTime       time.Time
	MapUI           string
	ModeUI          string
	Outcome         int  // code BRUT
	Rank            *int // placement 1-based ; nil si DNF/non classé
	Kills           int
	Deaths          int
	Assists         int
	KDA             float64
	Score           int
	DamageDealt     int
	DamageTaken     int
	MaxKillingSpree int
	PerfectKills    int
}
