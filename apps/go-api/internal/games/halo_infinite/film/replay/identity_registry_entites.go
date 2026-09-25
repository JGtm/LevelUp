package replay

// identity_registry_entites.go — LE CORPS D'UN INDEX PARTAGE EST NOMME PAR L'ENTITE QUI VIT A SA
// CREATION (lot M2.3 de la campagne « retours rejeu », 2026-09-23).
//
// # LE TROU QU'IL FERME (rapport `RAPPORT_equipes_b1ad85eb.md`, C4)
//
// Le record de creation d'un corps porte l'INDEX de son occupant (lot E2). Quand plusieurs
// occupants se relaient sur un meme index — trois bots sur l'index 8 de `b1ad85eb`, ou un bot puis
// l'humain qui prend son siege sur l'index 26 de `4f77afc1` —, l'index ne dit plus QUI : la lecture
// directe refusait ces corps (`index_hors_table`), `bidsParIndex` et `botNamesBySeat`
// s'abstenaient, et seul le relais par la base en nommait quelques-uns, en retard de ~20 s. Sur
// `b1ad85eb`, neuf corps de bots restaient anonymes, dont celui de `343 Hundy`, present au coup
// d'envoi et sans fiche.
//
// # LA LECTURE (sonde P4)
//
// Chaque occupant a SON entite `ti=9`. Le corps se lie a l'entite de son index dont la fenetre
// LARGE (image-cle voisine exclue de chaque cote) contient sa CREATION — pas l'intervalle
// BOT_METADATA : le corps de `343 PardonMy` precede de 39 frames sa premiere declaration. L'entite
// se lie a son occupant : un bot par ses declarations BOT_METADATA (fenetre STRICTE), l'unique
// humain de l'index pour l'entite qu'aucun bot ne revendique. Sur `b1ad85eb` les trois fenetres
// larges de l'index 8 sont disjointes : 512 -> Hundy, 526 -> PardonMy, 564..594 -> Brew Dog.
//
// Deux entites candidates d'occupants differents, ou aucune : la lecture se TAIT et les voies
// d'avant reprennent (tableau de l'API, relais), comme sans entite.

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar"

// proprietaireDEntite est une entite d'un index partage : sa fenetre LARGE et son occupant (un
// `bid` de bot ou un xuid d'humain, jamais les deux).
type proprietaireDEntite struct {
	fenetre fenetreLarge
	bid     string
	xuid    uint64
}

// meme dit si deux entites designent le meme occupant.
func (p proprietaireDEntite) meme(o proprietaireDEntite) bool {
	return p.bid == o.bid && p.xuid == o.xuid
}

// entitesDesIndexPartages : pour chaque index que PLUSIEURS occupants declarent, ses entites dont
// l'occupant est lu.
type entitesDesIndexPartages map[int][]proprietaireDEntite

// lireEntitesDesIndexPartages construit la table. Nil sans entite lue : la lecture se tait, et
// tout se passe comme avant le lot M2.3.
func lireEntitesDesIndexPartages(scan grammar.PlayerEntityScan, bots []BotIdentity,
	idx PlayerIndexTable) entitesDesIndexPartages {
	if !scan.Scanned || len(scan.Entities) == 0 {
		return nil
	}
	humains := map[int][]uint64{}
	for x, i := range idx.ByXUID {
		humains[i] = append(humains[i], x)
	}
	botsDeLIndex := map[int][]BotIdentity{}
	for _, b := range bots {
		if b.Bid() != "" {
			botsDeLIndex[b.FilmIndex] = append(botsDeLIndex[b.FilmIndex], b)
		}
	}
	out := entitesDesIndexPartages{}
	for i, bs := range botsDeLIndex {
		if len(bs)+len(humains[i]) < 2 {
			continue // un seul occupant declare : l'index le dit deja, rien a departager
		}
		for _, e := range scan.Entities {
			if e.Index != i || e.Unstable {
				continue
			}
			if p, ok := proprietaireDe(scan, e, bs, humains[i]); ok {
				out[i] = append(out[i], p)
			}
		}
	}
	return out
}

// proprietaireDe lit l'occupant d'une entite d'un index partage : l'unique bot dont une
// declaration croise sa fenetre stricte ; a defaut l'unique humain de l'index.
func proprietaireDe(scan grammar.PlayerEntityScan, e grammar.PlayerEntity, bots []BotIdentity,
	humains []uint64) (proprietaireDEntite, bool) {
	p := proprietaireDEntite{fenetre: fenetreLargeDe(scan, e)}
	var elu []BotIdentity
	for _, b := range bots {
		if entiteDeclareeParLeBot(scan, e, b.Declarations) {
			elu = append(elu, b)
		}
	}
	switch {
	case len(elu) == 1:
		p.bid = elu[0].Bid()
	case len(elu) == 0 && len(humains) == 1:
		p.xuid = humains[0]
	default:
		return p, false
	}
	return p, true
}

// occupantA rend l'occupant de l'index partage `i` dont l'entite vit a l'instant `t` (fenetre
// large), s'il est unique.
func (m entitesDesIndexPartages) occupantA(i int, t uint64) (proprietaireDEntite, bool) {
	var trouve proprietaireDEntite
	vu := false
	for _, p := range m[i] {
		if !p.fenetre.contient(t) {
			continue
		}
		if vu && !trouve.meme(p) {
			return proprietaireDEntite{}, false
		}
		trouve, vu = p, true
	}
	return trouve, vu
}

// nommerLesPistesDeBotParLeurVie pose le NOM du bot sur chaque piste anonyme que recouvre une vie
// du meme slot nommee par un `bid` (lecture du corps par son entite, ou tableau de l'API).
//
// POURQUOI UNE PASSE DE PLUS : `nameTracksByLives` ne pose que des xuids, et `nameBotTracks`
// nomme par le SIEGE — il s'abstient sur un index que plusieurs bots declarent. Une vie qui porte
// le `bid` de son bot est une lecture bornee dans le temps : c'est elle qui sait lequel des bots
// d'un index partage tenait ce corps.
func nommerLesPistesDeBotParLeurVie(tracks []Track, vies []lifeSpan, bots []BotIdentity, h replayClock) {
	nomDuBid := map[string]string{}
	for _, b := range bots {
		if bid := b.Bid(); bid != "" && b.Name != "" {
			nomDuBid[bid] = b.Name
		}
	}
	if len(nomDuBid) == 0 {
		return
	}
	for i := range tracks {
		if tracks[i].XUID != "" || tracks[i].Bot != "" {
			continue
		}
		de := int64(instantDeFrame(h, tracks[i].StartFrame))
		a := int64(instantDeFrame(h, tracks[i].EndFrame)+h.step) - 1
		var meilleur int64
		nom := ""
		for _, l := range vies {
			if l.slot != tracks[i].Slot || l.bid == "" || nomDuBid[l.bid] == "" {
				continue
			}
			if ov := minI64(a, l.to) - maxI64(de, l.from); ov > meilleur {
				meilleur, nom = ov, nomDuBid[l.bid]
			}
		}
		if nom != "" {
			tracks[i].Bot = nom
		}
	}
}
