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

	// Bornes de progression « Héros » (carte Carrière) PAR TITRE. XPMax = XP de
	// compte cumulé au rang MAX ; RankMax = numéro du rang max. Title-agnostic :
	// l'adapter du titre (qui connaît son système de rangs) les renseigne. Nil =
	// le service retombe sur les bornes par défaut (Halo Infinite). HINF laisse
	// nil (source unique = les constantes du service), Halo 5 les fixe (152 SR /
	// 50 000 000 XP) car son barème diffère.
	XPMax   *int
	RankMax *int

	// MaxRank (optionnel) — référence d'asset du rang MAXIMUM du titre (libellés
	// localisés). Sert au libellé « progression vers le rang max » de la page
	// Carrière, title-agnostic : l'adapter du titre (qui connaît le nom de son rang
	// sommet) le renseigne — Halo 5 = « SR 152 », Halo Infinite = laissé nil (le
	// service résout le libellé « Héros » depuis son catalogue de rangs). Nil =
	// le service retombe sur le catalogue du titre puis, à défaut, un libellé
	// générique côté front.
	MaxRank *AssetReference

	// Bloc Rank* enrichi (utile pour la page Carrière complète, pas seulement
	// pour la home preview). Ces champs sont peuplés à partir du provider
	// quand disponibles, sinon laissés à zéro.
	RankNumber int
	RankTier   *string
	RankName   *string
	IsMaxRank  bool
	RecordedAt *time.Time

	// État « placement » (modes classés). Le nombre de matchs de placement varie
	// par titre (Halo Infinite 5/10 par saison, Halo 5 = 10) : il est porté ici,
	// title-agnostic. Champs additifs (politique d'évolution canonical).
	//   - MeasurementMatchesRemaining : matchs de placement restants (> 0 = en cours
	//     de placement, rating non encore stabilisé). Nil = pas en placement / inconnu.
	//   - PlacementTotal : nombre total de matchs de placement du titre (5, 10…),
	//     fourni par TitleDescriptor.PlacementMatches. Nil = inconnu.
	MeasurementMatchesRemaining *int
	PlacementTotal              *int
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

// LUSRCheckpoint est un point d'historique de rating LUSR/CSR dans le canonique
// (Phase 2 HIGH-C). Porteur 1:1 de domain.LUSRCheckpointDTO : aucun type canonique
// existant ne convient (CareerSnapshot.History = progression XP, pas rating ;
// HighestLUSR = pic scalaire ; SkillSnapshot = skill de scoreboard par match).
// RatingType reste un string brut (le coercer en enum changerait les octets).
type LUSRCheckpoint struct {
	MatchID     string
	RatingType  string
	RatingValue float64
	// Tier (nom canonique anglais) + SubTier : le code dont le service déduit le libellé du palier
	// dans la langue de la requête (la colonne stockée `tier_label` n'est pas portée).
	Tier          string
	SubTier       *int
	PlaylistGroup *string
	PlaylistName  string
	PlaylistID    string
	RecordedAt    *time.Time
	RatingDelta   *float64
	BadgeImageURL *string
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
