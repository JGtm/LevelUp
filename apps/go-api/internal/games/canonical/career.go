package canonical

import "time"

// CareerSnapshot représente la progression carrière d'un joueur.
//
// Le bloc Rank* est volontairement riche pour permettre aux services
// produit de reconstruire un libellé complet (RankLabel + RankTier + RankNumber)
// sans dépendre du SemanticAdapter. RecordedAt et XPTotal alimentent les
// projections d'historique XP.
type CareerSnapshot struct {
	Player          PlayerIdentity
	CurrentRank     *AssetReference
	CurrentXP       *int
	XPForNextRank   *int
	XPTotal         *int
	NextRank        *AssetReference
	History         []CareerHistoryEntry
	HighestCSR      *int
	HighestLUSR     *float64
	HighestRatingAt *time.Time

	// Bloc Rank* enrichi (utile pour la page Carrière complète, pas seulement
	// pour la home preview). Ces champs sont peuplés à partir du provider
	// quand disponibles, sinon laissés à zéro.
	RankNumber int
	RankTier   *string
	RankName   *string
	IsMaxRank  bool
	RecordedAt *time.Time
}

// CareerHistoryEntry est une entrée de progression carrière historisée.
//
// Les champs RankNumber/CurrentXP/XPTotal (Phase 2 HIGH-C) portent l'historique XP
// numérique tel que servi par le titre : Rank *AssetReference (libellé/asset) et
// XP *int (XP générique) ne couvrent pas losslessly le triplet (rangNum, currentXP,
// xpTotal) du payload Carrière → champs additifs dédiés, byte-identique au domaine.
type CareerHistoryEntry struct {
	RecordedAt time.Time
	Rank       *AssetReference
	XP         *int
	RankNumber int  // numéro de rang brut (domain.XPHistoryPoint.Rank)
	CurrentXP  *int // XP courant dans le rang (domain.XPHistoryPoint.CurrentXP)
	XPTotal    *int // XP cumulé (domain.XPHistoryPoint.XPTotal)
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
