package port

// post_sync_progression.go — les lectures de l'orchestrateur de progression qui suit une synchronisation
// (séries, records, jalons, retour de pause) : les matchs récents d'UN joueur, ses agrégats cumulés, et son
// état courant (capture avant/après une synchronisation).

import (
	"context"
	"time"
)

// ProgressionMatchRow : un match du joueur avec les métriques dont les détecteurs de séries et de records ont
// besoin. Les valeurs absentes de la base valent zéro.
type ProgressionMatchRow struct {
	MatchID   string
	StartTime time.Time
	KDA       float64
	Kills     float64
	Deaths    float64
	Assists   float64
	// Accuracy est la précision du match en ratio 0..1 (ADR 0006) ; la colonne source est en pourcentage.
	Accuracy float64
	// PersonalScore est le score personnel du match.
	PersonalScore float64
	// TimeSeconds est la durée jouée, en secondes.
	TimeSeconds float64
	DamageDealt float64
	DamageTaken float64
}

// ProgressionAggregateParams : les paramètres des agrégats cumulés. Les seuils et le nombre de points de vie
// à tuer viennent de la configuration du titre ; le dépôt ne les décide pas.
type ProgressionAggregateParams struct {
	// AccuracyThreshold : un jour compte dans AccuracyDays si un match y atteint cette précision.
	AccuracyThreshold float64
	// OCThreshold et DRThreshold : seuils de conversion offensive et de résistance défensive des jalons de
	// combat.
	OCThreshold float64
	DRThreshold float64
	// HPToKill est la baseline de points de vie pour tuer du titre.
	HPToKill float64
}

// ProgressionAggregates : les compteurs cumulés d'un joueur, hors Campagne.
type ProgressionAggregates struct {
	MatchesPlayed int64
	Wins          int64
	Kills         int64
	Headshots     int64
	Assists       int64
	// AccuracyDays est le nombre de jours UTC distincts où un match atteint le seuil de précision.
	AccuracyDays int64
	// PrecisionMatches, EnduranceMatches, ExcellenceMatches : les matchs franchissant les seuils de combat.
	PrecisionMatches  int64
	EnduranceMatches  int64
	ExcellenceMatches int64
}

// PostSyncProgressionReader lit, pour UN joueur, ce que le pipeline de progression consomme. Il est lié au
// joueur à la construction. Les lectures de la base partagée passent par son fournisseur résilient à la fenêtre
// de swap (ADR 0016).
type PostSyncProgressionReader interface {
	// MatchRows rend les matchs du joueur depuis `since`, du plus ancien au plus récent, sans ceux dont
	// l'instant de début est inconnu.
	MatchRows(ctx context.Context, since time.Time) ([]ProgressionMatchRow, error)
	// PerformanceScores rend le score de performance par match, lu dans la base du joueur.
	PerformanceScores(ctx context.Context, matchIDs []string) (map[string]float64, error)
	// Aggregates rend les compteurs cumulés du joueur.
	Aggregates(ctx context.Context, params ProgressionAggregateParams) (ProgressionAggregates, error)
	// LastMatchTimes rend les instants de début des deux matchs les plus récents, du plus récent au plus ancien.
	LastMatchTimes(ctx context.Context) ([]time.Time, error)
}
