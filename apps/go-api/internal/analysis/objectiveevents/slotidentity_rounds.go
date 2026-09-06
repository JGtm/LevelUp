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

// CompletedByLines COMPLETE l'identite par manche avec le pont par TRIPLET
// ([SlotIdentityFrom]), sur les seuls slots que le pont par morts n'a pas su nommer, et
// UNIQUEMENT sur un film mono-manche.
//
// # LE TROU QU'ELLE BOUCHE, ET COMBIEN IL COUTAIT
//
// Le pont par morts exige `deathInstantMin` = 3 instants coincidents pour nommer un slot. Un
// joueur qui MEURT MOINS DE TROIS FOIS est donc STRUCTURELLEMENT hors de sa portee — et ce sont
// les MEILLEURS joueurs, ceux qui portent le drapeau. Mesure sur `c0a82e88` (Husky Raid:CTF,
// une manche) : le pont par morts nomme 5 slots sur 7 et laisse tomber le slot 22
// (2 morts, 7 frags, LE voleur ET LE captureur de drapeau du match) et le slot 20 (1 mort).
// Le calque `objectives` perdait avec eux ses DEUX SEULES actions de famille `flag`.
//
// # POURQUOI CE N'EST PAS UNE REGRESSION DE PRUDENCE
//
// Les deux ponts sont DISJOINTS par construction — l'un lit les TOTAUX du match, l'autre des
// INSTANTS du film — et sur ce film ils ne se contredisent nulle part : 3 slots nommes par les
// deux, a l'identique ; 2 que seul le triplet nomme ; 2 que seul le pont par morts nomme. Mieux,
// le CONTROLE CROISE confirme les deux slots litigieux : les progressions de mort du slot 22
// coincident avec le fil de 2535463878425995 et de LUI SEUL (2 sur 2), celles du slot 20 avec
// 2535465632069522 et lui seul (1 sur 1). Le pont par morts ne se tait pas parce qu'il doute :
// il se tait parce qu'il n'a pas ASSEZ D'ANCRES pour que sa marge s'applique.
//
// # LES TROIS GARDES, ET AUCUNE N'EST NEGOCIABLE
//
//  1. MONO-MANCHE SEULEMENT. Le triplet apparie des TOTAUX DE MATCH ; en multi-manche le slot
//     d'entite est reattribue et le compteur repart de zero — c'est exactement le defaut que
//     `d173b1a8c` a corrige, et il n'est pas question de le reintroduire.
//  2. COMPLETER, JAMAIS CONTREDIRE. Un slot deja nomme par le pont par morts garde son nom.
//  3. AUCUN XUID DEUX FOIS. Un xuid deja porte par un autre slot de la manche n'est pas ajoute —
//     meme regle que la seconde passe de [SlotIdentityFrom] et que [withoutContestedXUID].
//
// `lines` vide rend l'identite INCHANGEE : le calque reste publiable hors ligne, sans base.
func (ri RoundIdentity) CompletedByLines(recs []StatRecord, lines []PlayerLine) RoundIdentity {
	if len(lines) == 0 || len(ri.byRound) != 1 {
		return ri
	}
	triplet := SlotIdentityFrom(recs, lines)
	if len(triplet) == 0 {
		return ri
	}
	round, base := 0, map[int]string(nil)
	for r, m := range ri.byRound {
		round, base = r, m
	}
	fusion := make(map[int]string, len(base)+len(triplet))
	pris := make(map[string]bool, len(base))
	for slot, xuid := range base {
		fusion[slot] = xuid
		pris[xuid] = true
	}
	// Ordre STABLE : `triplet` est une map, et deux slots peuvent revendiquer le meme xuid.
	// Sans tri, lequel des deux l'emporte changerait d'une execution a l'autre.
	slots := make([]int, 0, len(triplet))
	for slot := range triplet {
		slots = append(slots, slot)
	}
	sort.Ints(slots)
	for _, slot := range slots {
		xuid := triplet[slot]
		if _, deja := fusion[slot]; deja || pris[xuid] {
			continue
		}
		fusion[slot] = xuid
		pris[xuid] = true
	}
	return RoundIdentity{byRound: map[int]map[int]string{round: fusion}, starts: ri.starts}
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
