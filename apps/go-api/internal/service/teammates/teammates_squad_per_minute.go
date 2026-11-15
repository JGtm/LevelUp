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

// paceSums : les sommes d'un joueur sur les matchs du périmètre. Frags, morts et assistances
// se rapportent aux secondes de tous les matchs joués ; chaque dégât a ses propres secondes,
// celles des seuls matchs où il est connu (un compte inconnu n'est pas un zéro : le match sort du
// numérateur ET du dénominateur, comme dans le groupe « Rythme » de Tendances).
type paceSums struct {
	kills, deaths, assists, secs, matches int

	damageDealt, damageDealtSecs float64
	damageTaken, damageTakenSecs float64
}

// addDamage ajoute les dégâts d'un match (nil = inconnu) et ses secondes jouées. Un match sans
// seconde jouée n'entre pas : sa cadence n'est pas définie.
func (a *paceSums) addDamage(dealt, taken *int, secs int, withTaken bool) {
	if secs <= 0 {
		return
	}
	if dealt != nil {
		a.damageDealt += float64(*dealt)
		a.damageDealtSecs += float64(secs)
	}
	if withTaken && taken != nil {
		a.damageTaken += float64(*taken)
		a.damageTakenSecs += float64(secs)
	}
}

// entry convertit les sommes en cadences, toutes par analysis.PerMinute. Les dégâts sont des
// pointeurs : absents (nil) tant qu'aucun match du joueur n'en porte un.
func (a paceSums) entry(player string) domain.SquadPerMinuteEntry {
	e := domain.SquadPerMinuteEntry{Player: player, MatchCount: a.matches}
	if a.secs > 0 {
		secs := float64(a.secs)
		e.KillsPerMinute = round2(analysis.PerMinute(float64(a.kills), secs))
		e.DeathsPerMinute = round2(analysis.PerMinute(float64(a.deaths), secs))
		e.AssistsPerMinute = round2(analysis.PerMinute(float64(a.assists), secs))
	}
	if a.damageDealtSecs > 0 {
		v := round2(analysis.PerMinute(a.damageDealt, a.damageDealtSecs))
		e.DamageDealtPerMinute = &v
	}
	if a.damageTakenSecs > 0 {
		v := round2(analysis.PerMinute(a.damageTaken, a.damageTakenSecs))
		e.DamageTakenPerMinute = &v
	}
	return e
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

// mainPaceSums : frags, morts, assistances et secondes du joueur principal, lus dans les lignes
// de l'escouade dédoublonnées par match (sa pipeline Q30).
func mainPaceSums(allSquadRows []domain.SquadMatchRow, allowed map[string]struct{}) paceSums {
	var agg paceSums
	seen := make(map[string]struct{}, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, dup := seen[m.MatchID]; dup {
			continue
		}
		seen[m.MatchID] = struct{}{}
		if _, ok := allowed[m.MatchID]; !ok {
			continue
		}
		agg.kills += m.Kills
		agg.deaths += m.Deaths
		agg.assists += m.Assists
		agg.secs += m.TimePlayedSecs
		agg.matches++
	}
	return agg
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

	mainAgg := mainPaceSums(allSquadRows, allowed)
	// Le damage du main vient de ses lignes canoniques ; sans lecteur ou en cas d'échec, ses
	// cadences de frags restent servies et ses dégâts sont absents (journalisé).
	if rows, ok := s.loadPaceRows(ctx, mainGamertag); ok {
		for _, r := range rows {
			if _, in := allowed[r.Summary.MatchID]; in {
				mainAgg.addDamage(r.Self.DamageDealt, r.Self.DamageTaken, intPtrOrZero(r.Self.TimePlayed), withTaken)
			}
		}
	}
	entries := []domain.SquadPerMinuteEntry{mainAgg.entry(mainGamertag)}

	// Coéquipiers : squadLoader est obligatoire — playerMatchesRepo est lié au gamertag principal
	// et ignore l'argument gt (cf. doc TeammatesService) ; l'utiliser ici réagrégeait les stats du
	// main pour chaque coéquipier.
	for _, gt := range selectedGamertags {
		rows, ok := s.loadPaceRows(ctx, gt)
		if !ok {
			entries = append(entries, domain.SquadPerMinuteEntry{Player: gt})
			continue
		}
		var agg paceSums
		for _, r := range rows {
			if _, in := allowed[r.Summary.MatchID]; !in {
				continue
			}
			secs := intPtrOrZero(r.Self.TimePlayed)
			agg.kills += intPtrOrZero(r.Self.Kills)
			agg.deaths += intPtrOrZero(r.Self.Deaths)
			agg.assists += intPtrOrZero(r.Self.Assists)
			agg.secs += secs
			agg.matches++
			agg.addDamage(r.Self.DamageDealt, r.Self.DamageTaken, secs, withTaken)
		}
		entries = append(entries, agg.entry(gt))
	}

	if !perMinuteHasSignal(entries) {
		return nil
	}
	return entries
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
