package canonical

import "time"

// CareerSnapshot représente la progression carrière d'un joueur.
type CareerSnapshot struct {
	Player          PlayerIdentity
	CurrentRank     *AssetReference
	CurrentXP       *int
	XPForNextRank   *int
	NextRank        *AssetReference
	History         []CareerHistoryEntry
	HighestCSR      *int
	HighestLUSR     *float64
	HighestRatingAt *time.Time
}

// CareerHistoryEntry est une entrée de progression carrière historisée.
type CareerHistoryEntry struct {
	RecordedAt time.Time
	Rank       *AssetReference
	XP         *int
}

// EncounterRow représente un joueur croisé fréquemment, dans le canonique.
//
// Les compteurs `AsTeammate` et `AsEnemy` permettent au consommateur de
// décider du rangement (équipe vs adversaire) sans logique titre-spécifique.
type EncounterRow struct {
	Identity   PlayerIdentity
	MatchCount int
	AsTeammate int
	AsEnemy    int
	AvgKDA     *float64
}
