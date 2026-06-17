package canonical

import "time"

// explorer.go — types canoniques cross-joueur / profil-cible de l'Explorer
// (Phase 2 HIGH-B). Concepts PvP universels (intersection 2 joueurs, kills croisés,
// profil de combat récent, agrégat de stats sur un set de matchs) qu'aucun type
// canonique mono-joueur existant ne couvre.

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
