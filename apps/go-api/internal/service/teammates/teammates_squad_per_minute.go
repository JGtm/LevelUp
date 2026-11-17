// Package service - teammates_squad_per_minute.go : teammates.14, stats par minute de
// l'escouade (frags, morts, assistances, dégâts infligés et subis). Sorti de
// teammates_squad_charts_intensity_perminute.go pour y porter les dégâts.
package teammates

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// paceMatch : les frags, morts, assistances et secondes jouées d'un match du périmètre.
type paceMatch struct{ kills, deaths, assists, secs int }

// paceDamage : les dégâts d'un match (nil = inconnu) et ses secondes jouées. Chaque dégât a ses propres
// matchs retenus : un compte inconnu n'est pas un zéro, le match sort du numérateur ET du dénominateur
// ([analysis.PooledPerMinute], la règle du groupe « Rythme » de Tendances).
type paceDamage struct {
	dealt, taken *int
	secs         int
}

// paceEntry convertit les matchs en cadences, toutes par [analysis.PooledPerMinute] : un match sans durée
// n'entre ni au numérateur ni au dénominateur, mais reste compté dans MatchCount. Les dégâts sont des
// pointeurs : absents (nil) tant qu'aucun match du joueur n'en porte un ; les dégâts subis ne sont rendus
// que si le titre les fournit (`withTaken`).
func paceEntry(player string, matches []paceMatch, damage []paceDamage, withTaken bool) domain.SquadPerMinuteEntry {
	e := domain.SquadPerMinuteEntry{Player: player, MatchCount: len(matches)}
	e.KillsPerMinute = matchRate(matches, func(m paceMatch) int { return m.kills })
	e.DeathsPerMinute = matchRate(matches, func(m paceMatch) int { return m.deaths })
	e.AssistsPerMinute = matchRate(matches, func(m paceMatch) int { return m.assists })
	e.DamageDealtPerMinute = damageRate(damage, func(d paceDamage) *int { return d.dealt })
	if withTaken {
		e.DamageTakenPerMinute = damageRate(damage, func(d paceDamage) *int { return d.taken })
	}
	return e
}

// matchRate rend la cadence arrondie d'un compte sur les matchs qui ont une durée, 0 sans match retenu.
func matchRate(matches []paceMatch, get func(paceMatch) int) float64 {
	v, ok := analysis.PooledPerMinute(matches,
		func(m paceMatch) (float64, bool) { return float64(get(m)), true },
		func(m paceMatch) float64 { return float64(m.secs) })
	if !ok {
		return 0
	}
	return round2(v)
}

// damageRate rend la cadence arrondie d'un dégât sur les matchs où il est connu, nil sans match retenu.
func damageRate(damage []paceDamage, get func(paceDamage) *int) *float64 {
	v, ok := analysis.PooledPerMinute(damage,
		func(d paceDamage) (float64, bool) {
			if p := get(d); p != nil {
				return float64(*p), true
			}
			return 0, false
		},
		func(d paceDamage) float64 { return float64(d.secs) })
	if !ok {
		return nil
	}
	r := round2(v)
	return &r
}

// perMinuteHasSignal : au moins une cadence non nulle (sinon le graphe n'a rien à dire).
func perMinuteHasSignal(entries []domain.SquadPerMinuteEntry) bool {
	for _, e := range entries {
		if e.KillsPerMinute > 0 || e.DeathsPerMinute > 0 || e.AssistsPerMinute > 0 ||
			e.DamageDealtPerMinute != nil || e.DamageTakenPerMinute != nil {
			return true
		}
	}
	return false
}

// perMinuteMatchIDs : les matchs du périmètre (lignes post-filtres), restreints à la session
// piquée quand il y en a une. Un ensemble vide non nil = session sans match = aucun match.
func perMinuteMatchIDs(allSquadRows []domain.SquadMatchRow, sessionMatchIDs map[string]bool) map[string]struct{} {
	allowed := make(map[string]struct{}, len(allSquadRows))
	for _, m := range allSquadRows {
		if sessionMatchIDs == nil || sessionMatchIDs[m.MatchID] {
			allowed[m.MatchID] = struct{}{}
		}
	}
	return allowed
}

// mainPaceMatches : frags, morts, assistances et secondes du joueur principal, un élément par match, lus
// dans les lignes de l'escouade dédoublonnées par match (sa pipeline Q30).
func mainPaceMatches(allSquadRows []domain.SquadMatchRow, allowed map[string]struct{}) []paceMatch {
	var out []paceMatch
	seen := make(map[string]struct{}, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, dup := seen[m.MatchID]; dup {
			continue
		}
		seen[m.MatchID] = struct{}{}
		if _, ok := allowed[m.MatchID]; !ok {
			continue
		}
		out = append(out, paceMatch{kills: m.Kills, deaths: m.Deaths, assists: m.Assists, secs: m.TimePlayedSecs})
	}
	return out
}

// buildSquadPerMinuteStats agrège sur le scope filtré, par joueur :
//   - main : frags, morts, assistances et secondes depuis allSquadRows dédupliqué par match_id
//     (sa pipeline Q30) ; dégâts depuis ses lignes canoniques (squadLoader), SquadMatchRow n'en
//     portant pas ;
//   - teammates : squadLoader.LoadFor (résolution par gamertag) filtré sur les match_ids de
//     l'escouade, lit r.Self.{Kills,Deaths,Assists,TimePlayed,DamageDealt,DamageTaken}.
//
// Les dégâts subis ne sont lus que si le titre les fournit (games.ProvidesDamageTaken, Halo 5 non).
// Retourne nil si aucun joueur n'a de cadence.
func (s *TeammatesService) buildSquadPerMinuteStats(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag string,
	selectedGamertags []string,
	sessionMatchIDs map[string]bool,
) []domain.SquadPerMinuteEntry {
	defer timing.FromContext(ctx).Section("per_minute")()
	if len(allSquadRows) == 0 {
		return nil
	}
	allowed := perMinuteMatchIDs(allSquadRows, sessionMatchIDs)
	withTaken := games.ProvidesDamageTaken(s.titleSlug)

	mainMatches := mainPaceMatches(allSquadRows, allowed)
	// Le damage du main vient de ses lignes canoniques ; sans lecteur ou en cas d'échec, ses
	// cadences de frags restent servies et ses dégâts sont absents (journalisé).
	var mainDamage []paceDamage
	if rows, ok := s.loadPaceRows(ctx, mainGamertag); ok {
		mainDamage = paceDamageOf(rows, allowed)
	}
	entries := []domain.SquadPerMinuteEntry{paceEntry(mainGamertag, mainMatches, mainDamage, withTaken)}

	// Coéquipiers : squadLoader est obligatoire — playerMatchesRepo est lié au gamertag principal
	// et ignore l'argument gt (cf. doc TeammatesService) ; l'utiliser ici réagrégeait les stats du
	// main pour chaque coéquipier.
	for _, gt := range selectedGamertags {
		rows, ok := s.loadPaceRows(ctx, gt)
		if !ok {
			entries = append(entries, domain.SquadPerMinuteEntry{Player: gt})
			continue
		}
		var matches []paceMatch
		for _, r := range rows {
			if _, in := allowed[r.Summary.MatchID]; !in {
				continue
			}
			matches = append(matches, paceMatch{
				kills: intPtrOrZero(r.Self.Kills), deaths: intPtrOrZero(r.Self.Deaths),
				assists: intPtrOrZero(r.Self.Assists), secs: intPtrOrZero(r.Self.TimePlayed),
			})
		}
		entries = append(entries, paceEntry(gt, matches, paceDamageOf(rows, allowed), withTaken))
	}

	if !perMinuteHasSignal(entries) {
		return nil
	}
	return entries
}

// paceDamageOf extrait les dégâts et les secondes des lignes canoniques d'un joueur, restreints aux
// matchs du périmètre.
func paceDamageOf(rows []canonical.PlayerMatchRow, allowed map[string]struct{}) []paceDamage {
	var out []paceDamage
	for _, r := range rows {
		if _, in := allowed[r.Summary.MatchID]; in {
			out = append(out, paceDamage{dealt: r.Self.DamageDealt, taken: r.Self.DamageTaken, secs: intPtrOrZero(r.Self.TimePlayed)})
		}
	}
	return out
}

// loadPaceRows charge les lignes canoniques d'un joueur ; false sans lecteur ou sur erreur
// (l'erreur est journalisée ici).
func (s *TeammatesService) loadPaceRows(ctx context.Context, gt string) ([]canonical.PlayerMatchRow, bool) {
	if s.squadLoader == nil {
		return nil, false
	}
	rows, err := s.squadLoader.LoadFor(ctx, s.titleSlug, gt, port.PlayerMatchFilters{})
	if err != nil {
		slog.WarnContext(ctx, "teammates_per_minute_load_failed", "gamertag", gt, "err", err)
		return nil, false
	}
	return rows, true
}
