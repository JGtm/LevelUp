package replay

// score_timeline_players.go — LES COMPTEURS VIVANTS DES JOUEURS : les deux chemins (mono-manche
// par totaux, multi-manche par identite de manche), leur cumul et leur tri par xuid.
//
// DEPLACEMENT PUR depuis `score_timeline.go` (lot 2.7 volet publication, 2026-09-16) : le
// fichier passait 500 lignes en portant TROIS sujets — l horloge et les series (restees
// la-bas), la courbe d EQUIPE (restee la-bas), et la courbe par JOUEUR (ici), dont l identite de
// manche est un sujet entier a elle seule. Aucune ligne n a change : memes fonctions, meme
// ordre, memes commentaires de mesure.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
)

// buildPlayerScores rend les compteurs vivants des joueurs, tries par xuid. DEUX CHEMINS, ET LE
// MONO-MANCHE EST L'ANCIEN, MOT POUR MOT.
//
//	MONO-MANCHE (<= 1 manche reelle)  le slot d'entite n'est PAS reattribue : l'identite plate
//	   par TOTAUX (le triplet des lignes de match) apparie chaque slot a son joueur, et le total
//	   d'un slot est celui de `SeriesTotal` — exactement comme avant cette migration.
//	MULTI-MANCHE (> 1 manche reelle)  le slot EST reattribue (slot 22 = joueur A en manche 0,
//	   joueur B ensuite) et le compteur repart de zero par manche : le pont plat ne verrait que
//	   la manche 0 (un seul joueur apparie). La courbe de chaque slot est DECOUPEE aux bornes de
//	   manche (`SeriesByRound` la donne deja par manche), chaque segment rattache au joueur de SA
//	   manche (identite par les instants de mort, `AtRound`), et les segments d'un meme xuid
//	   FUSIONNES en une seule entree, `Total` recompose dans l'ordre des manches.
//
// Un slot (mono) ou un couple (slot, manche) (multi) que le pont n'apparie pas sans ambiguite
// n'est PAS publie : attribuer les compteurs d'un joueur a un autre serait indetectable a l'ecran.
//
// LA FEUILLE ENTRE AUSSI DANS LE CHEMIN MULTI-MANCHE (correctif R4, 2026-09-08). Elle n'y entrait
// pas : `buildPlayerScores` ne recevait meme pas `lines`, si bien qu'un couple (slot, manche) que
// le pont par morts ne pouvait pas nommer — un joueur qui meurt moins de trois fois dans la
// manche — n'etait publie NULLE PART. Mesure : `51ebbc0f`, un joueur a 0 mort en manche 0, sa
// manche entiere absente, ecart cumule K/D/A de 9 contre la feuille.
// [objectives.RoundIdentity.CompletedByElimination] la ferme dans le cas d'unicite, controle
// par le residu de la feuille, et
// [objectives.RoundIdentity.CompletedByRoundResidue] des que la manche laisse PLUSIEURS
// slots muets (lot 6.7-B1). LA MEME CHAINE QUE LE PONT DE LA CUISSON (`replaybuild`, pontParManche) :
// deux lecteurs du meme pont doivent dire la meme chose du meme match. `lines` vide rend
// l'identite inchangee.
func buildPlayerScores(recs []objectives.StatRecord, flat map[int]string,
	lines []objectives.PlayerLine, deaths []Death, c scoreClock) []PlayerScore {
	if len(objectives.RealRounds(recs)) > 1 {
		round := objectives.ResolveRoundIdentity(recs, deathInstantsOf(deaths)).
			CompletedByElimination(recs, lines).
			CompletedByRoundResidue(recs, lines)
		return buildPlayerScoresByRound(recs, round, c)
	}
	return buildPlayerScoresFlat(recs, flat, c)
}

// buildPlayerScoresFlat est le chemin MONO-MANCHE : un PlayerScore par slot apparie par le pont
// des totaux, total lu tel quel dans `SeriesTotal`. Comportement d'avant la migration, inchange.
func buildPlayerScoresFlat(recs []objectives.StatRecord, identity map[int]string, c scoreClock) []PlayerScore {
	if len(identity) == 0 {
		return nil
	}
	personal := loadScoreSeries(recs, objectives.PersonalScoreComponent, false)
	kills := loadScoreSeries(recs, objectives.KillsComponent, false)
	deaths := loadScoreSeries(recs, objectives.DeathsComponent, false)
	assists := loadScoreSeries(recs, objectives.AssistsComponent, false)

	slots := make([]int, 0, len(identity))
	for slot := range identity {
		slots = append(slots, slot)
	}
	sort.Ints(slots)

	out := make([]PlayerScore, 0, len(slots))
	for _, slot := range slots {
		p := PlayerScore{
			XUID:    identity[slot],
			Score:   personal.at(slot, c),
			Kills:   kills.at(slot, c),
			Deaths:  deaths.at(slot, c),
			Assists: assists.at(slot, c),
		}
		if p.empty() {
			continue
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].XUID < out[j].XUID })
	if len(out) == 0 {
		return nil
	}
	return out
}

// buildPlayerScoresByRound est le chemin MULTI-MANCHE : la courbe de chaque slot est decoupee par
// manche, chaque segment rattache au joueur de SA manche (`AtRound`), les segments d'un meme xuid
// fusionnes en une entree — courbe recomposee dans l'ordre du temps.
func buildPlayerScoresByRound(recs []objectives.StatRecord,
	round objectives.RoundIdentity, c scoreClock) []PlayerScore {
	personal := playerRoundsByXUID(recs, objectives.PersonalScoreComponent, round)
	kills := playerRoundsByXUID(recs, objectives.KillsComponent, round)
	deaths := playerRoundsByXUID(recs, objectives.DeathsComponent, round)
	assists := playerRoundsByXUID(recs, objectives.AssistsComponent, round)

	out := make([]PlayerScore, 0)
	for _, xuid := range sortedXUIDs(personal, kills, deaths, assists) {
		p := PlayerScore{
			XUID:    xuid,
			Score:   seriesOfRounds(personal[xuid], c),
			Kills:   seriesOfRounds(kills[xuid], c),
			Deaths:  seriesOfRounds(deaths[xuid], c),
			Assists: seriesOfRounds(assists[xuid], c),
		}
		if p.empty() {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// playerRoundsByXUID regroupe les segments par MANCHE d'un composant sous le xuid qui occupait le
// slot A CETTE MANCHE (`AtRound`). Un couple (slot, manche) sans joueur nomme est ecarte. Rend
// xuid -> manche -> points (valeurs propres a la manche, deja triees et filtrees par
// `SeriesByRound`). L'identite par manche garantit qu'aucun xuid n'est revendique par deux slots
// dans la meme manche (`withoutContestedXUID`) : chaque (xuid, manche) recoit au plus un segment.
func playerRoundsByXUID(recs []objectives.StatRecord, comp objectives.StatComponent,
	round objectives.RoundIdentity) map[string]map[int][]objectives.ScorePoint {
	out := map[string]map[int][]objectives.ScorePoint{}
	for slot, byRound := range objectives.SeriesByRound(recs, comp, false) {
		for r, pts := range byRound {
			xuid := round.AtRound(r, slot)
			if xuid == "" {
				continue
			}
			if out[xuid] == nil {
				out[xuid] = map[int][]objectives.ScorePoint{}
			}
			out[xuid][r] = pts
		}
	}
	return out
}

// seriesOfRounds pose sur la grille les segments par manche d'un xuid : `Rounds` (les manches
// telles quelles) et `Total` (le cumul recompose dans l'ordre des manches).
//
// LE CUMUL PASSE PAR [objectives.ChronologicalTotal] : concatener les manches dans l'ordre
// des MANCHES ne donne une courbe chronologique que si la decoupe par manche est juste. Le
// controle refuse de publier une courbe qui recule dans le temps, et le dit au journal.
func seriesOfRounds(byRound map[int][]objectives.ScorePoint, c scoreClock) ScoreSeries {
	return ScoreSeries{
		Rounds: scoreRoundsOf(byRound, c),
		Total:  scoreTicksOf(objectives.ChronologicalTotal(cumulateXUIDRounds(byRound)), c),
	}
}

// cumulateXUIDRounds recompose la courbe cumulee d'un xuid a partir de ses segments par manche :
// chaque manche, dans l'ordre, decalee du total des manches precedentes. Les segments viennent de
// `SeriesByRound` (deja tries par instant et filtres par la plus longue sous-suite non
// decroissante — les quatre composants joueur sont tous NON stricts), donc leur dernier point est
// le total de la manche. C'est `cumulateRounds` d'`objectives`, applique par JOUEUR : un
// joueur qui garde son slot retrouve exactement `SeriesTotal`, un joueur reassigne voit ses
// manches fusionner dans l'ordre du temps.
func cumulateXUIDRounds(byRound map[int][]objectives.ScorePoint) []objectives.ScorePoint {
	rounds := make([]int, 0, len(byRound))
	for r := range byRound {
		rounds = append(rounds, r)
	}
	sort.Ints(rounds)
	var out []objectives.ScorePoint
	var offset int64
	for _, r := range rounds {
		pts := byRound[r]
		for _, p := range pts {
			out = append(out, objectives.ScorePoint{TimeMS: p.TimeMS, Slot: p.Slot, Value: p.Value + offset})
		}
		if len(pts) > 0 {
			offset += pts[len(pts)-1].Value
		}
	}
	return out
}

// sortedXUIDs rend l'union triee des xuids presents dans les composants fournis.
func sortedXUIDs(maps ...map[string]map[int][]objectives.ScorePoint) []string {
	seen := map[string]bool{}
	for _, m := range maps {
		for xuid := range m {
			seen[xuid] = true
		}
	}
	out := make([]string, 0, len(seen))
	for xuid := range seen {
		out = append(out, xuid)
	}
	sort.Strings(out)
	return out
}

// empty dit qu'aucun des quatre compteurs d'un joueur n'a rien a publier.
func (p PlayerScore) empty() bool {
	for _, s := range []ScoreSeries{p.Score, p.Kills, p.Deaths, p.Assists} {
		if len(s.Rounds) > 0 || len(s.Total) > 0 {
			return false
		}
	}
	return true
}
