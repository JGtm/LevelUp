package replay

// identity_registry_corps.go — LE CORPS QUI TIENT UN SLOT A UN INSTANT (lot J5.4 du plan de suite
// de l'audit du decodeur, 2026-09-27 ; constats RA2-1, RA2-2, RA2-3).
//
// # LE SLOT EST UN SIEGE, LE CORPS EST (SLOT, GENERATION)
//
// Le pool de handles de bipede reboucle : un slot porte plusieurs corps successifs, chacun ouvert
// par son record de creation (identity_registry_creation.go). Tant que les corps de generation
// >= 2 n'avaient pas de positions (GB-1, corrige au lot J5.2), les lecteurs qui raisonnaient sur
// le slot seul ne voyaient qu'un corps par slot et le defaut restait latent. Il ne l'est plus : un
// lecteur qui NOMME une piste ou RATTACHE un evenement doit demander le corps a l'instant, et la
// reponse est la regle de partage des vies ([corpsLu.recordA]) — pas une seconde regle.
//
// # CE QUE CE FICHIER SERT
//
//	[IdentityRegistry.Occupants]   le pont slot -> index de joueur A L'INSTANT : le record de
//	                               creation du corps vivant quand les records d'un slot divergent,
//	                               le pont par slot sinon, rien sur un slot ambigu (la meme
//	                               abstention que [IdentityRegistry.PontDeSlot] et que les equipes)
//	memeCorps / pontDuCorps        le nommage final par occupation borne au corps (unnamed_lives.go)

import (
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// occupantsDesSlots repond « quel index de joueur tient ce slot a cet instant ».
//
// UNE VALEUR, PAS UNE MAP : le pont par slot (`plat`) n'est servi QUE la ou il ne peut pas se
// tromper de corps — un slot dont les records ne divergent pas, et qui n'est pas ambigu.
type occupantsDesSlots struct {
	// plat : le pont slot -> index du registre (lectures, creations concordantes, fermetures).
	plat map[uint32]int
	// corps : les corps successifs de chaque slot, dates par leurs records de creation.
	corps map[uint32]corpsLu
	// ambigus : les slots dont les vies nommees designent des joueurs differents.
	ambigus map[uint32]bool
}

// occupantsPlats rend des occupants sans corps connus : un pont par slot seul. C'est la forme
// d'un appelant qui n'a lu aucun record de creation (fermetures en cours de construction du pont,
// instruments) — la meme reponse que le pont d'avant le lot J5.4, garde d'ambiguite comprise.
func occupantsPlats(owner map[uint32]int) occupantsDesSlots {
	return occupantsDesSlots{plat: owner}
}

// Occupants rend le pont slot -> index de joueur A L'INSTANT du registre.
func (r IdentityRegistry) Occupants() occupantsDesSlots {
	return occupantsDesSlots{plat: r.IndexParSlot(), corps: r.corps, ambigus: r.SlotsAmbigus()}
}

// vide dit qu'aucun slot n'est connu : ni pont, ni corps.
func (o occupantsDesSlots) vide() bool { return len(o.plat) == 0 && len(o.corps) == 0 }

// indexA rend l'index de joueur du corps qui tient `slot` a `tUS` (horloge du film).
//
// L'ORDRE EST CELUI DE LA FORCE DE PREUVE : sur un slot dont les records de creation portent des
// index DIFFERENTS, le record du corps vivant a l'instant fait foi (c'est une lecture, et le pont
// par slot y garderait le premier occupant) ; avant le premier record d'un tel slot, aucun corps
// n'est etabli et l'on se tait. Ailleurs, le pont par slot — sauf sur un slot ambigu, ou il
// servirait un nom arbitraire.
func (o occupantsDesSlots) indexA(slot uint32, tUS int64) (int, bool) {
	if c, ok := o.corps[slot]; ok && len(c.index) > 1 {
		d, connu := c.recordA(tUS)
		if !connu {
			return 0, false
		}
		return int(d.index), true
	}
	if o.ambigus[slot] {
		return 0, false
	}
	pi, ok := o.plat[slot]
	return pi, ok
}

// corpsDuSlot est le corps qui tient un slot a un instant : sa cle (slot, generation), et si elle
// est ETABLIE. Le drapeau fait partie de la valeur comparee — une cle non etablie ne se confond
// jamais avec la generation 0 d'un corps lu.
type corpsDuSlot struct {
	cle    types.LifeKey
	etabli bool
}

// corpsA rend le corps qui tient `slot` a `tUS`. Un slot sans record de creation, ou un instant
// anterieur au premier record d'un slot aux lectures divergentes, n'a aucun corps etabli.
func (r IdentityRegistry) corpsA(slot uint32, tUS int64) corpsDuSlot {
	c, ok := r.corps[slot]
	if !ok {
		return corpsDuSlot{cle: types.LifeKey{Slot: slot}}
	}
	d, connu := c.recordA(tUS)
	if !connu {
		return corpsDuSlot{cle: types.LifeKey{Slot: slot}}
	}
	return corpsDuSlot{cle: types.LifeKey{Slot: slot, Gen: d.gen}, etabli: true}
}

// memeCorps garde, parmi les vies nommees d'un slot, celles du corps qui tient ce slot a `tUS`.
//
// DEUX CORPS DONT AUCUN N'EST ETABLI sont tenus pour le meme : sans record de creation, le film ne
// dit aucune frontiere, et le nommage par occupation reste ce qu'il etait (un repli compte).
func (r IdentityRegistry) memeCorps(named []lifeSpan, slot uint32, tUS int64) []lifeSpan {
	corps := r.corpsA(slot, tUS)
	var out []lifeSpan
	for _, l := range named {
		if r.corpsA(slot, l.from) == corps {
			out = append(out, l)
		}
	}
	return out
}

// pontDuCorps rend le pont par slot ([IdentityRegistry.PontDeSlot]) quand il vaut pour TOUS les
// corps du slot — ses records ne divergent pas —, et rien sinon : le pont garde le premier
// occupant nomme, et le servir a un autre corps franchirait la frontiere.
func (r IdentityRegistry) pontDuCorps(slot uint32) string {
	if c, ok := r.corps[slot]; ok && len(c.index) > 1 {
		return ""
	}
	return r.PontDeSlot(slot)
}
