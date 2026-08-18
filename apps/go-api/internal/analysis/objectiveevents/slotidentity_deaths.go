package objectiveevents

import "sort"

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

// DeathInstant est une mort DATEE ET NOMMEE, telle que le fil des morts du film la donne.
// L'appelant la fournit : ce paquet ne decode pas le fil des morts (il a un seul proprietaire
// dans le depot, `analysis/replay`) et n'ouvre aucune base.
type DeathInstant struct {
	// XUID de la victime, en decimal — meme ecriture que [PlayerLine.XUID].
	XUID string
	// TimeMS est l'instant de la mort sur l'horloge du MATCH, la meme que celle des
	// [StatRecord].
	TimeMS int
}

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
func SlotIdentityResolved(src FilmSource, lines []PlayerLine, deaths []DeathInstant) (map[int]string, IdentityStats) {
	return slotIdentityResolvedFrom(StatRecords(src), lines, deaths)
}

// slotIdentityResolvedFrom est le coeur pur : il travaille sur des enregistrements deja
// decodes, donc testable sans film.
func slotIdentityResolvedFrom(recs []StatRecord, lines []PlayerLine, deaths []DeathInstant) (map[int]string, IdentityStats) {
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
func SlotIdentityFromDeaths(src FilmSource, deaths []DeathInstant) map[int]string {
	return slotIdentityFromDeaths(StatRecords(src), deaths)
}

// slotIdentityFromDeaths est le coeur pur du pont par instants.
func slotIdentityFromDeaths(recs []StatRecord, deaths []DeathInstant) map[int]string {
	if len(deaths) == 0 {
		return map[int]string{}
	}
	thread := deathThreadByXUID(deaths)
	claim := map[int]string{}
	for slot, pts := range deathProgressions(recs) {
		if xuid, ok := bestDeathClaim(pts, thread); ok {
			claim[slot] = xuid
		}
	}
	return withoutContestedXUID(claim)
}

// deathThreadByXUID range les morts du fil par joueur, chaque serie triee.
func deathThreadByXUID(deaths []DeathInstant) map[string][]int {
	out := map[string][]int{}
	for _, d := range deaths {
		if d.XUID == "" {
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
// Les emissions negatives (ancrages parasites) et les reculs sont ecartes : le compteur d'un
// joueur ne redescend pas, et une valeur qui redescend est une lecture fausse.
func deathProgressions(recs []StatRecord) map[int][]int {
	// StatRecords trie par instant : la serie d'un slot arrive donc deja chronologique.
	raw := map[int][]deathCount{}
	for _, r := range recs {
		if IsTeamSlot(r.Slot) {
			continue
		}
		v, ok := r.Comps[coreKillsComp]
		if !ok || v.B < 0 {
			continue
		}
		raw[r.Slot] = append(raw[r.Slot], deathCount{timeMS: r.TimeMS, deaths: v.B})
	}
	out := make(map[int][]int, len(raw))
	for slot, serie := range raw {
		prev := int64(0)
		var instants []int
		for _, p := range serie {
			for ; prev < p.deaths; prev++ {
				instants = append(instants, p.timeMS)
			}
		}
		out[slot] = instants
	}
	return out
}

// deathCount est une emission datee du compteur de morts d'un slot.
type deathCount struct {
	timeMS int
	deaths int64
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
