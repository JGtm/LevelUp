package trends

import (
	"time"

	"levelup/go-api/internal/domain"
)

// Options paramètre BuildSolo.
type Options struct {
	// Now : instant de référence (horloge injectée).
	Now time.Time
	// Loc : fuseau du joueur ; nil = UTC. Les Match.Start doivent être exprimés dans ce fuseau
	// (FromCanonical le garantit) : les intervalles de série en héritent.
	Loc *time.Location
	// GameType : chaîne de performance à isoler ; vide = tous les types.
	GameType string
	// HpToKill : points de vie effectifs pour un frag (rendement, résistance).
	HpToKill float64
	// Medals : médailles du joueur par match. nil = source absente ou en échec
	// (aucun bloc de médailles) ; non nil mais vide = lecture réussie sans médaille
	// (un bloc par horizon, sans ligne).
	Medals []MedalCount
	// MedalNames : nom de chaque médaille ; à défaut, l'identifiant en chiffres.
	MedalNames map[int64]string
}

// BuildSolo calcule la réponse de la vue Solo à partir de matchs triés par
// heure croissante (déjà restreints au contexte Solo par l'appelant). Il
// remplit AsOf, GameType, Timezone, Months, GameTypes, Indicators, Calendar,
// WinLoss, Mix et Medals (vide sans Options.Medals) ; View et Capabilities
// restent au service. GameTypes et Mix portent sur tous les matchs reçus ; le reste sur
// les matchs de la chaîne GameType (tous si vide). Les matchs postérieurs à
// Now sont ignorés.
func BuildSolo(matches []Match, opts Options) domain.TrendsPageResponse {
	loc := opts.Loc
	if loc == nil {
		loc = time.UTC
	}
	all := upTo(matches, opts.Now)
	scope := filterChain(all, opts.GameType)
	f := newFrame(scope, opts.Now, loc)
	medals := []domain.TrendsMedalsBlock{}
	if opts.Medals != nil {
		medals = buildMedals(f, opts.Medals, opts.MedalNames)
	}
	return domain.TrendsPageResponse{
		AsOf:       opts.Now,
		GameType:   opts.GameType,
		Timezone:   loc.String(),
		GameTypes:  buildGameTypes(all, opts.Now),
		Months:     f.monthKeys(),
		Indicators: buildIndicators(registry(scope, opts.HpToKill), f),
		Calendar:   buildCalendar(f),
		WinLoss:    buildWinLoss(f, opts.HpToKill),
		Medals:     medals,
		Mix:        buildMix(all, opts.Now),
		Members:    []domain.TrendsMember{},
	}
}

// upTo retourne les matchs qui ne commencent pas après now.
func upTo(ms []Match, now time.Time) []Match {
	out := make([]Match, 0, len(ms))
	for _, m := range ms {
		if !m.Start.After(now) {
			out = append(out, m)
		}
	}
	return out
}

// filterChain garde les matchs de la chaîne donnée ; chain vide = tous.
func filterChain(ms []Match, chain string) []Match {
	if chain == "" {
		return ms
	}
	out := make([]Match, 0, len(ms))
	for _, m := range ms {
		if chainKey(m) == chain {
			out = append(out, m)
		}
	}
	return out
}
