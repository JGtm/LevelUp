package replay

// hill_hold_ticks.go — LA GARDE DE LA COLLINE, LUE ET NON RECONSTRUITE.
//
// EN KOTH IL N'Y A PAS DE CAPTURE : la colline se prend instantanement quand aucun adversaire n'y
// est, et c'est la GARDE qui marque. Le film porte le compteur de cette garde, et ce fichier le
// publie par camp.
//
// LE COMPTEUR EST `comp 23 A`, ET C'EST MESURE (lot E1-bis du 2026-08-30) : il reproduit
// `ZonesStats.StrongholdScoringTicks` de l'API EXACTEMENT, joueur par joueur, sur 31 joueurs de
// 4 films — apres pont slot -> xuid. Sur le premier film il est le SEUL des 26 composants a bonne
// cardinalite a reproduire l'ensemble. Meme discriminant que VIP (`comp 22 A`) : l'exactitude par
// joueur, jamais la couverture.
//
// LA BARRE D'UN CAMP EST L'UNION DES INSTANTS DE SES JOUEURS, et les deux formules evidentes sont
// fausses (lot E1-ter, meme jour) :
//
//	la SOMME     compterait deux fois le meme tic — quand deux joueurs d'un camp sont sur la
//	             colline, ils prennent le MEME tic au MEME instant.
//	le MAXIMUM   sur toute la periode, il perd les RELAIS : un camp peut tenir la colline en se
//	             relayant, et alors aucun joueur n'accumule la totalite (releve : 18 a 35 selon
//	             la periode, la ou la barre du jeu vaut toujours la meme chose).
//
// L'UNION, elle, decoupe la periode aux instants d'emission, prend le maximum PAR TRANCHE et
// somme. Mesure : le camp qui marque rend **exactement 35** sur **15 periodes sur 16** (4 films,
// 4 cartes ; l'unique ecart vaut 33). Le controle est dans la mesure : le camp qui NE marque pas
// rend 1 a 25, jamais 35. Et 35 est aussi le chiffre annonce par la documentation du jeu — deux
// chaines independantes concordent.
//
// GARDE DE MODE CHEZ L'APPELANT, comme pour la couronne VIP et le porteur du crane : `comp 23 A`
// est un emplacement de statistique de TOUT mode, il ne porte des tics de colline que sur un mode
// a colline. Le calque n'est construit que si la variante est declaree a `[hold_ticks_per_point]`
// — jamais devine dans le film.

import (
	"sort"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// holdTicksComponent est l'emplacement du compteur de garde : composant 23, valeur A.
var holdTicksComponent = objectiveevents.StatComponent{Comp: 23}

// buildHoldTicks rend la serie CUMULATIVE de tics de garde par camp, en escalier sur l'axe de
// frames du document. Nil quand aucun camp n'est situable ou qu'aucun tic n'est lu.
//
// `teamByXUID` vient du ROSTER, jamais du film (meme regle que le proprietaire de zone) : le
// film numerote ses entites, il ne dit pas quel camp elles servent.
func buildHoldTicks(recs []objectiveevents.StatRecord, identity map[int]string,
	teamByXUID map[string]int, c scoreClock,
) []TeamHold {
	slotsParCamp := holdSlotsByTeam(identity, teamByXUID)
	if len(slotsParCamp) == 0 {
		return nil
	}
	series := objectiveevents.SeriesTotal(recs, holdTicksComponent, false)
	instants := holdInstants(series, slotsParCamp)
	if len(instants) == 0 {
		return nil
	}

	// `cursors` suit, pour chaque slot, la derniere valeur vue — l'union se calcule alors en un
	// seul passage sur les instants, sans re-balayer chaque serie a chaque tranche.
	cursors := map[int]*holdCursor{}
	for _, slots := range slotsParCamp {
		for _, slot := range slots {
			cursors[slot] = &holdCursor{pts: series[slot]}
		}
	}

	cumul := map[int]int{}
	points := map[int][]ScoreTick{}
	for _, tMS := range instants {
		for team, slots := range slotsParCamp {
			tranche := 0
			for _, slot := range slots {
				if d := cursors[slot].advance(tMS); d > tranche {
					tranche = d
				}
			}
			if tranche == 0 {
				continue
			}
			cumul[team] += tranche
			if f, ok := c.frameOf(tMS); ok {
				points[team] = append(points[team], ScoreTick{T: f, V: cumul[team]})
			}
		}
	}
	return holdSeriesOf(points)
}

// holdCursor lit une serie cumulative en avancant, et rend l'increment depuis le dernier appel.
type holdCursor struct {
	pts []objectiveevents.ScorePoint
	i   int
	val int
}

func (h *holdCursor) advance(tMS int) int {
	prev := h.val
	for h.i < len(h.pts) && h.pts[h.i].TimeMS <= tMS {
		h.val = int(h.pts[h.i].Value)
		h.i++
	}
	if h.val < prev {
		return 0 // une serie cumulative ne recule pas ; si elle le fait, on ne l'invente pas
	}
	return h.val - prev
}

// holdSlotsByTeam range les slots d'entite par camp du roster. Un slot dont le joueur n'est pas
// situe est ECARTE : il ne peut faire avancer aucune barre.
func holdSlotsByTeam(identity map[int]string, teamByXUID map[string]int) map[int][]int {
	out := map[int][]int{}
	for slot, xuid := range identity {
		team, ok := teamByXUID[xuid]
		if !ok {
			continue
		}
		out[team] = append(out[team], slot)
	}
	for team := range out {
		sort.Ints(out[team])
	}
	return out
}

// holdInstants rend les instants d'emission du compteur, tous camps confondus, tries et
// dedoublonnes. Ce sont les bornes des tranches sur lesquelles l'union se calcule.
func holdInstants(series map[int][]objectiveevents.ScorePoint, slotsParCamp map[int][]int) []int {
	vus := map[int]bool{}
	for _, slots := range slotsParCamp {
		for _, slot := range slots {
			for _, p := range series[slot] {
				vus[p.TimeMS] = true
			}
		}
	}
	out := make([]int, 0, len(vus))
	for t := range vus {
		out = append(out, t)
	}
	sort.Ints(out)
	return out
}

// holdSeriesOf met les series en forme publiee, triees par camp (ordre deterministe).
func holdSeriesOf(points map[int][]ScoreTick) []TeamHold {
	teams := make([]int, 0, len(points))
	for team := range points {
		teams = append(teams, team)
	}
	sort.Ints(teams)
	out := make([]TeamHold, 0, len(teams))
	for _, team := range teams {
		if len(points[team]) == 0 {
			continue
		}
		id := team
		out = append(out, TeamHold{TeamID: &id, Ticks: points[team]})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
