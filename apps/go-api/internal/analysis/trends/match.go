// Package trends calcule la page Tendances : l'évolution des statistiques d'un
// joueur sur 7 / 30 / 90 / 365 jours, chaque horizon comparé à la période d'avant
// de même durée. Le paquet est pur : aucune lecture de base, aucun accès réseau ;
// les données arrivent en lignes aplaties (Match), le fuseau, l'horloge et la
// classification des types de partie sont injectés.
package trends

import (
	"sort"
	"time"

	"levelup/go-api/internal/games/canonical"
)

// Match est la ligne aplatie d'un match joué. Start est exprimé dans le fuseau
// du joueur : les jours, semaines et mois se découpent dessus. Les champs
// optionnels sont des pointeurs (nil = inconnu).
type Match struct {
	// ID : identifiant du match (clé de jonction des échantillons et des médailles).
	ID      string
	Start   time.Time
	Outcome canonical.Outcome

	Kills   int
	Deaths  int
	Assists int
	// KDA est le FDA natif du match (valeur de l'API), jamais un quotient.
	KDA *float64

	ShotsFired int
	ShotsHit   int

	DamageDealt *float64
	DamageTaken *float64

	// AvgLife : durée de vie moyenne en secondes.
	AvgLife *float64
	// MaxSpree : meilleure série de frags.
	MaxSpree *float64

	HeadshotKills int
	PowerKills    int

	TeamMMR  *float64
	EnemyMMR *float64

	PerformanceScore *float64
	// Seconds : secondes jouées.
	Seconds float64
	// IsWithFriends : match joué avec au moins un ami (exclu de la vue Solo par le service).
	IsWithFriends bool
	// Chain : chaîne de performance (type de partie) ; vide = "other".
	Chain string

	// Instantané de classement : RatingType vide si absent ; RatingValue nil pour
	// un match de placement.
	RatingType  canonical.RatingType
	RatingValue *float64
	RatingGroup string

	// Objective : rôles d'objectif du joueur et de son camp ; nil = match sans
	// objectif mesuré (Attach le pose).
	Objective *ObjectiveSample
	// Equipment : usage de l'équipement ; nil = match non mesuré (Attach le pose).
	Equipment *EquipmentSample
	// Squad : frags et bilans de l'équipe alliée ; nil = match sans lecture
	// d'équipe (AttachSquad le pose).
	Squad *SquadSample
}

// GetStartTime implémente temporal.HasStartTime.
func (m Match) GetStartTime() time.Time { return m.Start }

// FromOptions paramètre FromCanonical.
type FromOptions struct {
	// Loc : fuseau du joueur ; nil = UTC.
	Loc *time.Location
	// ChainOf classe un match en chaîne de performance ; nil = aucune chaîne.
	ChainOf func(pairName string, isRanked, isPvE bool) string
}

// FromCanonical aplatit les lignes canoniques en Match, triées par heure de
// début croissante. Les matchs PvE sont écartés. Secondes jouées = temps joué du
// joueur, à défaut la durée du match. Un instantané en phase de placement
// (matchs de mesure restants) ne porte pas de valeur de classement.
func FromCanonical(rows []canonical.PlayerMatchRow, opts FromOptions) []Match {
	loc := opts.Loc
	if loc == nil {
		loc = time.UTC
	}
	out := make([]Match, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		if r.Summary.IsPvE != nil && *r.Summary.IsPvE {
			continue
		}
		out = append(out, flatten(r, loc, opts.ChainOf))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

// flatten convertit une ligne canonique en Match.
func flatten(r *canonical.PlayerMatchRow, loc *time.Location, chainOf func(string, bool, bool) string) Match {
	s := &r.Self
	m := Match{
		ID:               r.Summary.MatchID,
		Start:            r.Summary.StartedAtUTC.In(loc),
		Outcome:          s.Outcome,
		Kills:            intOr0(s.Kills),
		Deaths:           intOr0(s.Deaths),
		Assists:          intOr0(s.Assists),
		KDA:              s.KDA,
		ShotsFired:       intOr0(s.ShotsFired),
		ShotsHit:         intOr0(s.ShotsHit),
		DamageDealt:      intToFloat(s.DamageDealt),
		DamageTaken:      intToFloat(s.DamageTaken),
		AvgLife:          s.AvgLifeSeconds,
		MaxSpree:         intToFloat(s.MaxKillingSpree),
		HeadshotKills:    intOr0(s.HeadshotKills),
		PowerKills:       intOr0(s.PowerWeaponKills),
		TeamMMR:          r.Enrichment.TeamMMR,
		EnemyMMR:         r.Enrichment.EnemyMMR,
		PerformanceScore: r.Enrichment.PerformanceScore,
		IsWithFriends:    r.Enrichment.IsWithFriends,
		Seconds:          secondsPlayed(r),
	}
	if m.Outcome == "" {
		m.Outcome = r.Summary.Outcome
	}
	if chainOf != nil {
		pair := ""
		if r.Enrichment.PairName != nil {
			pair = *r.Enrichment.PairName
		}
		isRanked := r.Summary.IsRanked != nil && *r.Summary.IsRanked
		m.Chain = chainOf(pair, isRanked, false)
	}
	if sk := r.Enrichment.SkillSnapshot; sk != nil {
		m.RatingType = sk.RatingType
		if sk.PlaylistGroup != nil {
			m.RatingGroup = *sk.PlaylistGroup
		}
		placement := sk.MeasurementRemaining != nil && *sk.MeasurementRemaining > 0
		if !placement {
			m.RatingValue = sk.RatingValue
		}
	}
	return m
}

// secondsPlayed : temps joué du joueur, à défaut durée du match, sinon 0.
func secondsPlayed(r *canonical.PlayerMatchRow) float64 {
	if r.Self.TimePlayed != nil {
		return float64(*r.Self.TimePlayed)
	}
	if r.Summary.DurationSeconds != nil {
		return float64(*r.Summary.DurationSeconds)
	}
	return 0
}

func intOr0(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func intToFloat(p *int) *float64 {
	if p == nil {
		return nil
	}
	v := float64(*p)
	return &v
}

// chainKey retourne la chaîne du match, "other" si elle est vide.
func chainKey(m Match) string {
	if m.Chain == "" {
		return otherChain
	}
	return m.Chain
}
