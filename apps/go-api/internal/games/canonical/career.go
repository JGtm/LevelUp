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
