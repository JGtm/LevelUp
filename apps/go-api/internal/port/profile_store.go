package port

// profile_store.go — les lectures du profil de progression d'un joueur (Ascension) : LUSR et tendance,
// agrégats de la fenêtre pour le radar et l'engagement, jours joués, composantes de performance, modèles de
// défis suggérés. Un magasin est lié à UN joueur (ses bases) ; le service qui l'orchestre ne connaît aucun
// type SQL.

import (
	"context"
	"time"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/prestige"
)

// ProfileLUSR est la dernière note LUSR du joueur (valeurs nulles quand il n'en a aucune) et le nombre de
// matchs notés.
type ProfileLUSR struct {
	Mu           float64
	Sigma        float64
	LastMatchAt  *time.Time
	MatchesCount int
}

// ProfileLUSRPoint est une note LUSR datée.
type ProfileLUSRPoint struct {
	At    time.Time
	Value float64
}

// ProfileRadarBase porte les moyennes par match de la fenêtre qui alimentent le radar ; un champ nil est
// une moyenne inconnue (NULL).
type ProfileRadarBase struct {
	AvgKills        *float64
	AvgDeaths       *float64
	AvgAssists      *float64
	AvgScore        *float64
	AvgKillingSpree *float64
	MatchCount      int
}

// ProfileAwardTotal est le cumul d'un award du joueur sur les matchs de la fenêtre.
type ProfileAwardTotal struct {
	Name  string
	Total int64
}

// ProfileMatchTime est l'instant de début (canonique) d'un match.
type ProfileMatchTime struct {
	MatchID string
	At      time.Time
}

// ProfileComponentAggregate agrège une composante LUSR sur l'historique du joueur ; un champ nil est une
// valeur inconnue (NULL).
type ProfileComponentAggregate struct {
	Name     string
	Avg      *float64
	Top20    *float64
	LastAvg  *float64
	FirstAvg *float64
	N        int
}

// ProfileStore lit tout ce que le service de profil assemble. Les fenêtres sont [since, until] ; la
// Campagne est exclue des lectures de matchs selon le titre du joueur.
type ProfileStore interface {
	// TitleSlug rend le titre des bases du joueur.
	TitleSlug() string

	// LatestLUSR rend la dernière note LUSR et le nombre de matchs notés.
	LatestLUSR(ctx context.Context) (ProfileLUSR, error)
	// LUSRSeries rend les notes LUSR de la fenêtre, en ordre chronologique.
	LUSRSeries(ctx context.Context, since, until time.Time) ([]float64, error)
	// LUSRSeriesPoints rend les mêmes notes avec leur date (les points sans date sont omis).
	LUSRSeriesPoints(ctx context.Context, since, until time.Time) ([]ProfileLUSRPoint, error)
	// LUSRComponents rend l'agrégat de chaque composante LUSR, hors matchs exclus. Une requête qui échoue
	// rend une liste vide sans erreur (l'interface dégrade) ; la lecture des exclusions, elle, remonte.
	LUSRComponents(ctx context.Context) ([]ProfileComponentAggregate, error)

	// CountWindowMatches compte les matchs distincts du joueur dans la fenêtre.
	CountWindowMatches(ctx context.Context, xuid string, since, until time.Time) (int, error)
	// RadarBase rend les moyennes par match du joueur dans la fenêtre.
	RadarBase(ctx context.Context, xuid string, since, until time.Time) (ProfileRadarBase, error)
	// WindowMatchIDs rend les matchs du registre dans la fenêtre, tous joueurs confondus.
	WindowMatchIDs(ctx context.Context, since, until time.Time) ([]string, error)
	// PlayerWindowMatchIDs rend les matchs où le joueur a participé dans la fenêtre.
	PlayerWindowMatchIDs(ctx context.Context, xuid string, since, until time.Time) ([]string, error)
	// AwardTotals cumule par award les points du joueur sur les matchs donnés (les cumuls nuls ou négatifs
	// sont omis). En cas d'erreur de lecture en cours de route, les cumuls déjà lus sont rendus avec l'erreur.
	AwardTotals(ctx context.Context, xuid string, matchIDs []string) ([]ProfileAwardTotal, error)
	// MatchStartTimes rend les instants de début des matchs du joueur dans la fenêtre, en ordre croissant.
	MatchStartTimes(ctx context.Context, xuid string, since, until time.Time) ([]time.Time, error)
	// MatchDays rend (match, début) des matchs du joueur dans la fenêtre, pour le calendrier d'activité.
	MatchDays(ctx context.Context, xuid string, since, until time.Time) ([]ProfileMatchTime, error)

	// ObjectiveIndexInputs rend, par joueur, les agrégats de l'index des objectifs sur les matchs donnés.
	ObjectiveIndexInputs(ctx context.Context, matchIDs, xuids []string) (map[string]narrative.ObjectiveIndexInput, error)
	// KillDeathEvents rend les événements de frag et de mort des matchs donnés.
	KillDeathEvents(ctx context.Context, matchIDs []string) ([]canonical.HighlightEvent, error)

	// SuggestionTemplates rend les modèles de défis du titre, avec leurs textes par langue. available est
	// faux quand le joueur n'a pas de base de référentiels : aucune suggestion n'est alors possible.
	SuggestionTemplates(ctx context.Context, titleSlug string) (templates []prestige.Template, available bool, err error)
}
