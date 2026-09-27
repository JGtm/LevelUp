package objectives

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// slotidentity_deaths.go — LE SECOND PONT slot statborg -> joueur : par les INSTANTS DE
// MORT, et sans jamais consulter la base.
//
// # Pourquoi un second pont, et ce qu'il repare
//
// [SlotIdentity] apparie un slot au TRIPLET FINAL (frags, morts, assistances) de la ligne de
// match du joueur. C'est exact quand les compteurs du film atteignent leurs valeurs finales,
// et c'est MESURE que ce n'est pas toujours le cas : un film que le Theater rend TRONQUE
// s'arrete avant. Mesure du 2026-08-18 (plan `.ai/V7.5/replay2d/PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md`,
// phase 0) : **0 slot sur 8** sur `64e8adfa` et `24dbb67d` (`64e8adfa` slot 24 = 10/18/7 dans le
// film contre 10/21/7 a l'API), **8 sur 8** sur `530820e5` et `53ce4390`.
//
// # La voie de remplacement n'emprunte rien a l'API : les MORTS
//
// Le statborg replique le compteur de morts de chaque joueur (`comp 2 B`, confirme 8/8 contre
// `match_participants`, cf. slotidentity.go) et le film porte par ailleurs son FIL DES MORTS,
// qui date chaque mort et NOMME sa victime par son xuid. Les deux sont sur l'horloge du MATCH.
// Apparier les INSTANTS plutot que les TOTAUX identifie donc le slot a partir du film SEUL, et
// tient sur un film tronque : il suffit qu'assez de morts soient communes aux deux lectures.
//
// Resultat de la phase 0 : **8/8 sur les quatre films**, et **8 accords / 0 desaccord** avec le
// pont par triplets la ou celui-ci repond — deux chaines totalement disjointes qui disent la
// meme chose.
//
// # La regle de prudence est celle du paquet
//
// Un slot dont le meilleur candidat ne devance pas NETTEMENT le suivant n'est pas apparie, et
// un xuid que deux slots se disputent n'est attribue a aucun des deux. Se taire vaut mieux
// qu'attribuer le drapeau au mauvais joueur : sur une carte, l'erreur serait invisible et
// credible.

const (
	// deathInstantToleranceMS : tolerance d'appariement entre l'instant d'une mort du fil et
	// celui de la progression du compteur de morts du statborg. Meme ordre que la fenetre qui
	// borne deja le bruit d'horloge du pont bipede du rejeu (150 ms).
	deathInstantToleranceMS = 150
	// deathInstantMargin : le meilleur candidat doit devancer le suivant d'au moins ce
	// facteur. Sans marge, un slot se ferait attribuer un joueur sur une poignee de
	// coincidences.
	deathInstantMargin = 2
	// deathInstantMin : nombre minimal de morts communes pour qu'un appariement compte.
	deathInstantMin = 3
)

// IdentityStats porte les denominateurs du pont resolu : combien de slots chaque voie nomme,
// laquelle a ete retenue, et combien de slots ont ete ECARTES parce que les deux se
// contredisaient.
//
// Publie a part parce qu'une table d'identites sans ses denominateurs ne se juge pas : huit
// slots nommes par les totaux et huit nommes par defaut de mieux ne valent pas la meme chose.
type IdentityStats struct {
	// ByTotals / ByDeaths : nombre de slots que chaque pont nomme seul.
	ByTotals, ByDeaths int
	// Conflicts : slots que les deux ponts nomment DIFFEREMMENT. Ils sont retires du
	// resultat — aucun des deux ne peut etre cru.
	Conflicts int
	// Source vaut [IdentitySourceTotals] ou [IdentitySourceDeaths] : la voie retenue.
	Source string
}

// Les deux voies possibles de [IdentityStats.Source].
const (
	IdentitySourceTotals = "totals"
	IdentitySourceDeaths = "deaths"
)

// SlotIdentityResolved resout le slot d'entite statborg de chaque joueur en preferant le pont
// par TOTAUX, et en se repliant sur le pont par INSTANTS DE MORT quand celui-ci nomme plus de
// slots — c'est-a-dire quand le film est tronque.
//
// LE REPLI NE SE DECLENCHE QUE S'IL AJOUTE QUELQUE CHOSE : le pont par instants doit nommer
// STRICTEMENT plus de slots que celui par totaux. Un film complet rend donc exactement ce que
// [SlotIdentity] rendait — la voie neuve ne peut pas degrader l'existant.
//
// LES DESACCORDS SONT ECARTES, PAS ARBITRES : un slot que les deux ponts nomment differemment
// sort de la table (et se compte dans [IdentityStats.Conflicts]). Les deux chaines etant
// disjointes, un desaccord signale que l'une des deux lit de travers, et rien ne dit laquelle.
//
// `deaths` vide (fil des morts illisible) : seul le pont par totaux repond, comme avant.
func SlotIdentityResolved(film *source.Film, lines []types.PlayerLine, deaths []types.DeathInstant) (map[int]string, IdentityStats) {
	return slotIdentityResolvedFrom(StatRecords(film), lines, deaths)
}

// slotIdentityResolvedFrom est le coeur pur : il travaille sur des enregistrements deja
// decodes, donc testable sans film.
func slotIdentityResolvedFrom(recs []types.StatRecord, lines []types.PlayerLine, deaths []types.DeathInstant) (map[int]string, IdentityStats) {
	byTotals := SlotIdentityFrom(recs, lines)
	byDeaths := slotIdentityFromDeaths(recs, deaths)
	st := IdentityStats{ByTotals: len(byTotals), ByDeaths: len(byDeaths), Source: IdentitySourceTotals}
	if len(byDeaths) <= len(byTotals) {
		return byTotals, st
	}
	st.Source = IdentitySourceDeaths
	out := make(map[int]string, len(byDeaths))
	for slot, xuid := range byDeaths {
		if other, ok := byTotals[slot]; ok && other != xuid {
			st.Conflicts++
			continue
		}
		out[slot] = xuid
	}
	return out, st
}

// SlotIdentityFromDeaths apparie chaque slot statborg a un xuid par les seuls INSTANTS DE MORT
// du film. Aucune ligne de match, aucune base — c'est ce qui le rend employable sur un film
// tronque.
func SlotIdentityFromDeaths(film *source.Film, deaths []types.DeathInstant) map[int]string {
	return slotIdentityFromDeaths(StatRecords(film), deaths)
}

// SlotIdentityByDeaths est la MEME regle sur des enregistrements DEJA DECODES — le suffixe de ce
// paquet pour les variantes pures est `From` (cf. [SlotIdentityFrom]), mais il se lirait ici
// « FromDeathsFrom ». Elle existe pour l'appelant qui a deja balaye le film une fois : le calque
// du drapeau vivant du rejeu 2D le fait, et rebalayer coutait 0,6 a 2,4 s et jusqu'a 21 Mo par
// film.
func SlotIdentityByDeaths(recs []types.StatRecord, deaths []types.DeathInstant) map[int]string {
	return slotIdentityFromDeaths(recs, deaths)
}

// slotIdentityFromDeaths est le coeur pur du pont par instants.
func slotIdentityFromDeaths(recs []types.StatRecord, deaths []types.DeathInstant) map[int]string {
	return slotIdentityFromDeathsCompte(recs, deaths, nil)
}

// slotIdentityFromDeathsCompte est [slotIdentityFromDeaths], qui compte ses deux replis dans `c`
// (nil : rien) — la table vide faute de morts, les morts sans xuid (lot J8.7).
func slotIdentityFromDeathsCompte(recs []types.StatRecord, deaths []types.DeathInstant, c *ComptesDesReplis) map[int]string {
	if len(deaths) == 0 {
		if c != nil {
			c.TablesIdentiteVides++
		}
		return map[int]string{}
	}
	thread := deathThreadByXUIDCompte(deaths, c)
	claim := map[int]string{}
	for slot, pts := range deathProgressions(recs) {
		if xuid, ok := bestDeathClaim(pts, thread); ok {
			claim[slot] = xuid
		}
	}
	return withoutContestedXUID(claim)
}

// deathThreadByXUIDCompte range les morts du fil par joueur, chaque serie triee, et compte dans `c` (nil : rien) les morts
// ignorees faute de xuid — `repli_mort_sans_xuid_ignoree` (lot J8.7).
func deathThreadByXUIDCompte(deaths []types.DeathInstant, c *ComptesDesReplis) map[string][]int {
	out := map[string][]int{}
	for _, d := range deaths {
		if d.XUID == "" {
			if c != nil {
				c.MortsSansXUID++
			}
			continue
		}
		out[d.XUID] = append(out[d.XUID], d.TimeMS)
	}
	for x := range out {
		sort.Ints(out[x])
	}
	return out
}

// deathProgressions rend, par slot de joueur, UN instant par unite gagnee par le compteur de
// morts (`comp 2 B`) — la serie que le fil des morts doit reproduire.
//
// ELLE DEROULE LA SERIE TOTALE PUBLIEE ([SeriesTotal] de [DeathsComponent]), ET RIEN D AUTRE
// (lot J8.5 du plan de suite d audit, constat FO-3, 2026-09-27). Le pont appliquait ses propres
// gardes (slot de joueur, valeur dans [0, 1000]) quand la serie que le document publie passe par
// d autres filtres — manche confrontee au temps, manches fantomes, plus longue sous-suite non
// decroissante, borne par pas. Une emission que la serie publiee jette fabriquait ici des morts
// (49 au meme instant pour une emission a 50), qui noyaient les coincidences du slot et le
// faisaient taire. Lire LA serie publiee rend le desaccord impossible par construction.
//
// LA BORNE MEMOIRE DU BLOQUANT DU 2026-08-18 EST TENUE PAR LA MEME SERIE : un pas au-dela de
// `maxUnrollPerStep` n y laisse aucune unite ([boundSteps]), donc une emission aberrante (des
// centaines de millions) ne deroule rien. Le plafond propre au pont (`maxDeathsPerSlot`, 1 000) a
// disparu avec ses gardes : il bornait une valeur que la serie publiee ne laisse plus passer.
func deathProgressions(recs []types.StatRecord) map[int][]int {
	return instantsDesMorts(SeriesTotal(recs, DeathsComponent, false))
}

// deathProgressionsByRound est [deathProgressions] MANCHE PAR MANCHE : `manche -> slot ->
// instants`, deroule depuis la serie PAR MANCHE publiee ([SeriesByRound]), dont les valeurs
// repartent de zero a chaque manche comme le compteur du jeu.
func deathProgressionsByRound(recs []types.StatRecord) map[int]map[int][]int {
	parManche := map[int]map[int][]types.ScorePoint{}
	for slot, byRound := range SeriesByRound(recs, DeathsComponent, false) {
		for round, pts := range byRound {
			if parManche[round] == nil {
				parManche[round] = map[int][]types.ScorePoint{}
			}
			parManche[round][slot] = pts
		}
	}
	out := make(map[int]map[int][]int, len(parManche))
	for round, series := range parManche {
		out[round] = instantsDesMorts(series)
	}
	return out
}

// instantsDesMorts deroule une serie de compteur de morts (valeurs non decroissantes, partant de
// zero) en un instant par unite gagnee.
func instantsDesMorts(series map[int][]types.ScorePoint) map[int][]int {
	out := make(map[int][]int, len(series))
	for slot, pts := range series {
		prev := int64(0)
		var instants []int
		for _, p := range pts {
			for ; prev < p.Value; prev++ {
				instants = append(instants, p.TimeMS)
			}
		}
		out[slot] = instants
	}
	return out
}

// bestDeathClaim designe le joueur dont le fil des morts coincide le mieux avec la serie d'un
// slot, sous les deux regles de prudence (minimum de coincidences, marge sur le suivant).
func bestDeathClaim(instants []int, thread map[string][]int) (string, bool) {
	best, second, winner := 0, 0, ""
	for xuid, fil := range thread {
		n := coincidences(instants, fil)
		switch {
		case n > best:
			best, second, winner = n, best, xuid
		case n > second:
			second = n
		}
	}
	if best < deathInstantMin || best < deathInstantMargin*second {
		return "", false
	}
	return winner, true
}

// withoutContestedXUID ecarte les xuid revendiques par plusieurs slots — meme regle que la
// seconde passe de [SlotIdentity].
func withoutContestedXUID(claim map[int]string) map[int]string {
	bySlotsOf := map[string][]int{}
	for slot, xuid := range claim {
		bySlotsOf[xuid] = append(bySlotsOf[xuid], slot)
	}
	out := make(map[int]string, len(claim))
	for xuid, slots := range bySlotsOf {
		if len(slots) == 1 {
			out[slots[0]] = xuid
		}
	}
	return out
}

// coincidences compte les instants appariables entre deux series, chaque instant du fil ne
// servant qu'une fois (appariement glouton par ordre croissant).
func coincidences(a, b []int) int {
	used := make([]bool, len(b))
	n := 0
	for _, t := range a {
		best, bd := -1, deathInstantToleranceMS+1
		for i, u := range b {
			if used[i] {
				continue
			}
			if d := abs(u - t); d < bd {
				bd, best = d, i
			}
		}
		if best >= 0 {
			used[best] = true
			n++
		}
	}
	return n
}
