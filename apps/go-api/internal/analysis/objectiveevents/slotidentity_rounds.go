package objectiveevents

import "sort"

// slotidentity_rounds.go — L'IDENTITE slot d'entite -> joueur PAR MANCHE.
//
// # Pourquoi un troisieme pont, et ce qu'il repare
//
// [SlotIdentityByDeaths] rend UN SEUL `map[slot]->xuid` pour tout le match. En MULTI-MANCHE
// c'est FAUX, et la mesure le montre : le SLOT d'entite est REATTRIBUE d'une manche a l'autre.
// Sur d9781168 (Oddball, 3 manches), les instants de mort du slot 22 valent scuderiasven en
// manche 0 puis LadyJezz en manches 1-2 — un slot n'est donc PAS une identite stable sur le
// match. En plus, le compteur de morts (`comp 2 B`) REPART DE ZERO a chaque manche : un
// deroulage monotone sur tout le match ne voit que la manche 0. Les deux defauts se corrigent
// d'un coup en resolvant l'identite MANCHE PAR MANCHE.
//
// # La resolution reutilise l'algorithme prudent du pont plat
//
// Chaque manche est resolue par LA MEME regle que [slotIdentityFromDeaths] — meilleur candidat
// avec MARGE sur le suivant, rejet des xuid contestes par deux slots — appliquee aux
// progressions du compteur de morts RESTREINTES a la manche, appariees au fil des morts COMPLET
// du film (les instants de mort sont sur l'horloge du match ; seule la VALEUR du compteur repart
// de zero par manche). Rien de neuf dans l'appariement : seule la SOURCE (les progressions par
// manche) change.
//
// # NEUTRALITE MONO-MANCHE, garantie par construction
//
// Un film a AU PLUS UNE manche reelle rend `{manche: SlotIdentityByDeaths(recs, deaths)}` — LA
// MEME fonction que le pont plat, indexee de la meme facon. [RoundIdentity.At] ignore alors le
// temps (il n'y a qu'une manche). Les calques deja livres (couronne VIP, drapeau CTF) rendent
// donc EXACTEMENT ce qu'ils rendaient sur un film mono-manche ; seuls les films multi-manche
// changent, et ils s'ameliorent.

// SlotIdentityByRound resout le slot d'entite de chaque joueur POUR CHAQUE MANCHE reelle du
// film. Rend `manche -> slot -> xuid`.
//
// Pour <= 1 manche reelle, retourne `{manche: SlotIdentityByDeaths(recs, deaths)}` : le pont
// plat, mot pour mot (cf. l'en-tete, NEUTRALITE MONO-MANCHE).
func SlotIdentityByRound(recs []StatRecord, deaths []DeathInstant) map[int]map[int]string {
	rounds := realRoundsSorted(recs)
	if len(rounds) <= 1 {
		r := 0
		if len(rounds) == 1 {
			r = rounds[0]
		}
		return map[int]map[int]string{r: slotIdentityFromDeaths(recs, deaths)}
	}
	thread := deathThreadByXUID(deaths)
	out := make(map[int]map[int]string, len(rounds))
	for _, round := range rounds {
		claim := map[int]string{}
		for slot, pts := range deathProgressionsForRound(recs, round) {
			if xuid, ok := bestDeathClaim(pts, thread); ok {
				claim[slot] = xuid
			}
		}
		out[round] = withoutContestedXUID(claim)
	}
	return out
}

// realRoundsSorted rend la liste triee des manches REELLES (cf. [RealRounds]).
func realRoundsSorted(recs []StatRecord) []int {
	real := RealRounds(recs)
	out := make([]int, 0, len(real))
	for r, ok := range real {
		if ok {
			out = append(out, r)
		}
	}
	sort.Ints(out)
	return out
}

// deathProgressionsForRound est [deathProgressions] RESTREINT a une manche : par slot de joueur,
// un instant par unite gagnee par le compteur de morts (`comp 2 B`) DANS cette manche. Memes
// gardes que le pont plat (slot de joueur, valeur dans [0, maxDeathsPerSlot], reculs jetes).
func deathProgressionsForRound(recs []StatRecord, round int) map[int][]int {
	raw := map[int][]deathCount{}
	for _, r := range recs {
		if r.Round != round || IsTeamSlot(r.Slot) {
			continue
		}
		v, ok := r.Comps[coreKillsComp]
		if !ok || v.B < 0 || v.B > maxDeathsPerSlot {
			continue
		}
		raw[r.Slot] = append(raw[r.Slot], deathCount{timeMS: r.TimeMS, deaths: v.B})
	}
	out := make(map[int][]int, len(raw))
	for slot, serie := range raw {
		// StatRecords trie deja par instant : la serie d'un slot arrive chronologique.
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

// RoundIdentity resout le slot d'entite en xuid EN TENANT COMPTE DE LA MANCHE. C'est la forme que
// les calques consomment : ils ont un slot et soit un INSTANT (evenement nomme), soit une MANCHE
// (train de tics du porteur), et le slot ne veut rien dire sans l'une des deux.
//
// Sur un film mono-manche, `At` ignore le temps et rend l'identite unique — le comportement du
// pont plat, a l'octet pres.
type RoundIdentity struct {
	// byRound : manche -> slot -> xuid.
	byRound map[int]map[int]string
	// starts : debut (ms, horloge des enregistrements) de chaque manche, TRIE. Sert a
	// [RoundIdentity.At] a placer un instant dans sa manche. Vide ou singleton : `At` rend
	// toujours l'unique manche.
	starts []roundStart
}

// roundStart associe une manche a son premier instant sur l'horloge des enregistrements.
type roundStart struct {
	round   int
	startMS int
}

// ResolveRoundIdentity construit le resolveur : l'identite par manche PLUS les bornes de manche
// (en ms) qui permettent de placer un instant dans sa manche.
func ResolveRoundIdentity(recs []StatRecord, deaths []DeathInstant) RoundIdentity {
	byRound := SlotIdentityByRound(recs, deaths)
	return RoundIdentity{byRound: byRound, starts: roundStartsOf(recs, byRound)}
}

// FlatRoundIdentity fabrique un resolveur d'UNE seule manche a partir d'une table plate. Sert aux
// tests des calques (scenarios mono-manche synthetiques) : `At` et `AtRound` y rendent la table
// telle quelle, sans dependance au temps.
func FlatRoundIdentity(identity map[int]string) RoundIdentity {
	return RoundIdentity{byRound: map[int]map[int]string{0: identity}}
}

// roundStartsOf rend, par manche presente dans `byRound`, le plus petit instant d'enregistrement
// observe pour cette manche, la liste triee par instant croissant.
func roundStartsOf(recs []StatRecord, byRound map[int]map[int]string) []roundStart {
	if len(byRound) <= 1 {
		return nil
	}
	min := map[int]int{}
	for _, r := range recs {
		if _, ok := byRound[r.Round]; !ok {
			continue
		}
		if cur, seen := min[r.Round]; !seen || r.TimeMS < cur {
			min[r.Round] = r.TimeMS
		}
	}
	out := make([]roundStart, 0, len(min))
	for round, start := range min {
		out = append(out, roundStart{round: round, startMS: start})
	}
	// LE NUMERO DE MANCHE DEPARTAGE, et il le faut : `out` est bati en iterant la MAP `min`,
	// `sort.Slice` n'est pas stable, et deux manches peuvent porter le MEME instant de debut
	// (une manche vide, ou deux enregistrements au meme horodatage). Sans ce departage, l'ordre
	// des ex aequo changeait a chaque execution (meme defaut que lessTrack, filmdec/projectiles.go
	// — correction du 2026-09-02, item 0.4bis de PLAN_CUISSON_PERF).
	sort.Slice(out, func(i, j int) bool {
		if out[i].startMS != out[j].startMS {
			return out[i].startMS < out[j].startMS
		}
		return out[i].round < out[j].round
	})
	return out
}

// At rend le xuid du slot A L'INSTANT donne (horloge des enregistrements / du match). L'instant
// choisit la manche ; hors multi-manche, l'unique manche repond sans regarder le temps.
func (ri RoundIdentity) At(slot, timeMS int) string {
	if len(ri.byRound) == 0 {
		return ""
	}
	if len(ri.byRound) == 1 {
		for _, m := range ri.byRound {
			return m[slot]
		}
	}
	return ri.byRound[ri.roundOfTime(timeMS)][slot]
}

// AtRound rend le xuid du slot POUR UNE MANCHE connue — la voie du porteur, qui itere deja les
// trains de tics manche par manche.
func (ri RoundIdentity) AtRound(round, slot int) string {
	if m := ri.byRound[round]; m != nil {
		return m[slot]
	}
	return ""
}

// Rounds rend la liste triee des manches resolues.
func (ri RoundIdentity) Rounds() []int {
	out := make([]int, 0, len(ri.byRound))
	for r := range ri.byRound {
		out = append(out, r)
	}
	sort.Ints(out)
	return out
}

// NamedCount rend le nombre total de slots nommes, toutes manches confondues — un denominateur
// pour la journalisation.
func (ri RoundIdentity) NamedCount() int {
	n := 0
	for _, m := range ri.byRound {
		n += len(m)
	}
	return n
}

// roundOfTime rend la manche dont le debut est le plus grand qui ne depasse pas `timeMS`. Un
// instant anterieur a toute manche connue retombe sur la premiere (les manches se jouent dans
// l'ordre : rien avant la premiere).
func (ri RoundIdentity) roundOfTime(timeMS int) int {
	round := ri.starts[0].round
	for _, s := range ri.starts {
		if s.startMS > timeMS {
			break
		}
		round = s.round
	}
	return round
}
