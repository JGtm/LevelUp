package killsource

// botmeta.go — BOT_METADATA : LE FILM NOMME SES BOTS (RE_LOG 7ter.62).
//
// Le paquet de type 12 n est pas un ChunkType : il vit A L INTERIEUR d un chunk de replication
// decompresse. Il etait sur le disque depuis le debut. Il porte `nbBots`, le SLOT de chaque bot,
// son identifiant `bid(N.0)` et son NOM — le tout en BIG-ENDIAN, nom en UTF-16BE, alors que
// l en-tete de paquet qui l encadre est, lui, little-endian.
//
// Grammaire :
//
//	0x000  u32 BE  nbBots   (0 => le paquet fait 4 octets et s arrete la)
//	puis nbBots entrees, la premiere a l octet 4 :
//	+0x004 u32 BE  slot     indice ABSOLU du bot dans le roster de replication
//	+0x008 u32 BE  botID    l entier N de bid(N.0)
//	+0x078 UTF-16BE nom, termine par 0x0000
//
// DEUX LECTEURS, ET LA RAISON EST MESUREE : le stride fixe de 2076 octets est VRAI pour
// nbBots = 1 et FAUX des nbBots >= 2 (une entree y est decalee d un DEMI-OCTET). `nbBots` se lit
// donc au premier u32, sans hypothese de stride ; les slots et les noms passent par un scan
// bit-precis aux memes offsets negatifs constants.
//
// CRITERE PRE-ENREGISTRE ATTEINT : `343 Aloysius`/bid(39.0) et `343 PardonMy`/bid(7.0), les deux
// declarant `slot=8`. L identification << indice 8 = le bot >> est donc LUE, plus deduite d une
// coincidence de K/D.
//
// # LE PAQUET EST UN ETAT, ET SON INSTANT EST UNE LECTURE (sonde P4, 2026-09-23 ; lot M2.1)
//
// Mesure sur `b1ad85eb` (41 paquets) : le paquet de type 12 est REECRIT en tete de CHAQUE chunk
// et a chaque CHANGEMENT de bots en milieu de chunk ; un paquet de 4 octets (`nbBots=0`) dit
// « plus aucun bot ». Le depart d un bot est donc DATE a la frame pres par le premier paquet qui
// ne le declare plus (`343 Hundy` f273, `343 PardonMy` f831), et son arrivee par le premier qui
// le declare (exacte sur un paquet de changement, `343 Brew Dog` f3155 ; bornee par la tete de
// chunk sinon). L agregat d avant dedupliquait par (slot, bid) et perdait tout instant : trois
// bots qui se relaient sur l index 8 ne se distinguaient plus que par leur nom.
//
// [bot.declarations] porte ces intervalles. Un paquet dont le scan ne retrouve pas `nbBots`
// entrees est INCOMPLET : il ouvre ce qu il lit, il ne FERME rien (un bot manque a la lecture
// n est pas un bot parti), et il se compte ([botMeta.Incomplets]).

import (
	"sort"
	"unicode/utf16"
)

// bot : un bot declare par le film.
type bot struct {
	Slot   int
	BotID  int
	Name   string
	bitPos int // position dans le payload : sert a ecarter la copie bit-decalee
	// declarations : les intervalles pendant lesquels BOT_METADATA declare ce bot, dans l ordre
	// du film (cf. l en-tete).
	declarations []BotDeclaration
}

// BotDeclaration est UN intervalle pendant lequel BOT_METADATA declare un bot : du premier paquet
// qui le declare (inclus) au premier paquet COMPLET suivant qui ne le declare plus (exclu).
type BotDeclaration struct {
	// FromUS est l instant du premier paquet qui le declare — celui de l IMAGE-CLE de son chunk
	// quand ce paquet appartient a l instantane de tete (cf. [paquetsBotMeta]).
	FromUS uint64
	// ToUS est l horodatage du premier paquet complet qui ne le declare plus. ZERO = il est
	// encore declare au dernier paquet du film.
	ToUS uint64
}

// BotEntry : un bot tel que le film le declare.
//
// DEPLACE DE roster.go LE 2026-09-23 (lot M2.1) avec son champ neuf : roster.go est a la borne des
// 500 lignes, et le type est la forme PUBLIEE d un [bot] de ce fichier.
type BotEntry struct {
	Slot  int
	BotID int
	Name  string
	// Declarations : les intervalles de declaration BOT_METADATA (cf. [BotDeclaration]). Ils
	// lient le bot a SON entite `ti=9` par le temps (publication du rejeu) ; aucune ligne de kill
	// ne les lit.
	Declarations []BotDeclaration
}

// entree rend la forme publiee d un bot.
func (b bot) entree() BotEntry {
	return BotEntry{Slot: b.Slot, BotID: b.BotID, Name: b.Name,
		Declarations: append([]BotDeclaration(nil), b.declarations...)}
}

// botMeta : ce que le film declare sur ses bots, tous chunks confondus.
type botMeta struct {
	NBots int // max des nbBots vus (le roster peut se remplir en cours de film)
	NPkt  int
	Bots  []bot
	// Incomplets : paquets dont le scan n a pas retrouve `nbBots` entrees. Ils n ont ferme aucune
	// declaration (cf. l en-tete).
	Incomplets int
}

const (
	botSlotBackBits = 0x74 * 8 // bits AVANT le debut du nom
	botIDBackBits   = 0x70 * 8
	botNameMin      = 4
	botNameMax      = 48
	botMaxSlot      = 64
	botMaxID        = 4096
)

// loadBotMeta : agrege tous les paquets type 12 d un film deja decoupe.
//
// L AGREGAT EST INCHANGE (lot M2.1) : memes bots, meme ordre, meme `NBots` — c est lui qu epingle
// le roster du kill-feed, et aucune ligne de kill ne doit bouger. Ce qui s ajoute est l INSTANT :
// les paquets sont parcourus dans l ordre du film et chaque bot garde ses intervalles de
// declaration (cf. l en-tete).
func loadBotMeta(f *film) botMeta {
	m := botMeta{}
	rang := map[[2]int]int{}       // (slot, bid) -> position dans m.Bots
	ouverts := map[[2]int]uint64{} // declares au dernier paquet lu -> debut de leur declaration
	for _, pi := range paquetsBotMeta(f) {
		p := pi.p
		m.NPkt++
		n := int(uint32(p.payload[0])<<24 | uint32(p.payload[1])<<16 |
			uint32(p.payload[2])<<8 | uint32(p.payload[3]))
		if n < 0 || n > botMaxSlot {
			continue
		}
		if n > m.NBots {
			m.NBots = n
		}
		entrees := scanBotEntries(p.payload)
		declares := make(map[[2]int]bool, len(entrees))
		for _, b := range entrees {
			k := [2]int{b.Slot, b.BotID}
			declares[k] = true
			if _, vu := rang[k]; !vu {
				rang[k] = len(m.Bots)
				m.Bots = append(m.Bots, b)
			}
			if _, ouvert := ouverts[k]; !ouvert {
				ouverts[k] = pi.instant
			}
		}
		if len(entrees) != n {
			m.Incomplets++ // un bot manque a la LECTURE n est pas un bot parti : rien ne se ferme
			continue
		}
		fermerLesAbsents(&m, rang, ouverts, declares, pi.instant)
	}
	for _, k := range clesTriees(ouverts) {
		i := rang[k]
		m.Bots[i].declarations = append(m.Bots[i].declarations, BotDeclaration{FromUS: ouverts[k]})
	}
	sort.Slice(m.Bots, func(i, j int) bool { return m.Bots[i].Slot < m.Bots[j].Slot })
	return m
}

// paquetBotMeta : un paquet type 12 et l INSTANT qu il declare.
type paquetBotMeta struct {
	p *packet
	// instant : l horodatage du paquet — ou celui de l IMAGE-CLE de son chunk quand le paquet
	// appartient a l INSTANTANE DE TETE (cf. [paquetsBotMeta]).
	instant uint64
}

// paquetsBotMeta rend les paquets type 12 LISIBLES (au moins le mot `nbBots`), dans l ORDRE DU
// FILM — chunk par chunk, paquet par paquet, c est-a-dire l ordre des horodatages.
//
// L ORDRE N EST PAS RETRIE, ET C EST VOULU : c est celui dans lequel l agregat d avant decouvrait
// ses bots, et deux bots d un meme slot gardent ainsi leur rang relatif — celui qui decide lequel
// nomme l indice au kill-feed ([roster.pinBots]). Retrier ici pourrait deplacer une ligne de kill.
//
// L INSTANTANE DE TETE (mesure du 2026-09-23, `b1ad85eb`) : un chunk s ouvre par son image-cle
// (paquet 1), puis quelques paquets d etat — dont le BOT_METADATA de tete (paquet 4) — AVANT sa
// premiere trame de replication (type 0). Ce paquet de tete porte l etat des bots A L IMAGE-CLE ;
// il est seulement ecrit apres elle (390 us plus tard sur le chunk 2 de `b1ad85eb`). Son instant
// est donc celui de l image-cle : sans cela, un bot vu a UNE seule image-cle (`343 PardonMy`,
// f813) se trouverait declare « apres » l unique instant ou son entite est lue.
func paquetsBotMeta(f *film) []paquetBotMeta {
	var out []paquetBotMeta
	chunk, imageCle, enTete := -1, uint64(0), false
	for i := range f.packets {
		p := &f.packets[i]
		if p.chunk != chunk {
			chunk, imageCle, enTete = p.chunk, 0, true
		}
		switch {
		case p.typ == packetTypeKeyframe && enTete && imageCle == 0:
			imageCle = p.ts
		case p.typ == packetType0:
			enTete = false
		case p.typ == packetTypeBotMeta && len(p.payload) >= 4:
			instant := p.ts
			if enTete && imageCle != 0 {
				instant = imageCle
			}
			out = append(out, paquetBotMeta{p: p, instant: instant})
		}
	}
	return out
}

// fermerLesAbsents ferme, a l instant d un paquet COMPLET, la declaration de chaque bot ouvert que
// ce paquet ne porte plus.
func fermerLesAbsents(m *botMeta, rang map[[2]int]int, ouverts map[[2]int]uint64,
	declares map[[2]int]bool, ts uint64) {
	for _, k := range clesTriees(ouverts) {
		if declares[k] {
			continue
		}
		i := rang[k]
		m.Bots[i].declarations = append(m.Bots[i].declarations,
			BotDeclaration{FromUS: ouverts[k], ToUS: ts})
		delete(ouverts, k)
	}
}

// clesTriees rend les cles d une table de declarations ouvertes, triees : l ordre d iteration
// d une map Go est aleatoire, et les faits doivent etre reproductibles a l octet.
func clesTriees(m map[[2]int]uint64) [][2]int {
	out := make([][2]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	return out
}

// scanBotEntries : enumere les entrees d un payload type 12, SANS hypothese de stride. Un nom
// est un run UTF-16BE d ASCII imprimable ferme par 0x0000 ; slot et botID se lisent aux deux
// offsets negatifs constants. Le terminateur obligatoire elimine les sous-chaines.
func scanBotEntries(pl []byte) []bot {
	var out []bot
	total := len(pl) * 8
	for bit := 0; bit+16 <= total; bit++ {
		name, next := readBotName(pl, bit)
		if name == "" {
			continue
		}
		s, okS := readU32BE(pl, bit-botSlotBackBits)
		d, okD := readU32BE(pl, bit-botIDBackBits)
		if okS && okD && s < botMaxSlot && d < botMaxID {
			out = append(out, bot{Slot: int(s), BotID: int(d), Name: name, bitPos: bit})
		}
		bit = next
	}
	return firstCopyOnly(out)
}

// readBotName : le nom qui commence au bit `bit`, et la position du terminateur. Rend "" si ce
// n est pas un nom (trop court, caractere non imprimable, terminateur absent).
func readBotName(pl []byte, bit int) (string, int) {
	name := make([]byte, 0, botNameMax)
	p := bit
	for len(name) < botNameMax {
		c, ok := readU16BE(pl, p)
		if !ok || c < 0x20 || c > 0x7e {
			break
		}
		name = append(name, byte(c))
		p += 16
	}
	if len(name) < botNameMin {
		return "", p
	}
	if c, ok := readU16BE(pl, p); !ok || c != 0 {
		return "", p
	}
	u := make([]uint16, len(name))
	for i, c := range name {
		u[i] = uint16(c)
	}
	return string(utf16.Decode(u)), p
}

// firstCopyOnly : le nom d un bot apparait DEUX FOIS par entree, la seconde copie etant
// bit-decalee. Aux offsets negatifs de cette seconde copie, slot et botID se lisent en ZEROS —
// d ou un faux `slot=0 bid(0.0)` qui volerait l indice 0 a un humain. On ne garde donc, PAR NOM,
// que l occurrence de plus BASSE position : c est l entree primaire, celle dont les offsets ont
// ete mesures.
func firstCopyOnly(in []bot) []bot {
	first := map[string]bot{}
	for _, b := range in {
		if e, ok := first[b.Name]; !ok || b.bitPos < e.bitPos {
			first[b.Name] = b
		}
	}
	out := make([]bot, 0, len(first))
	for _, b := range first {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].bitPos < out[j].bitPos })
	return out
}

// byteAtBit : un octet lu a une position de BIT quelconque.
func byteAtBit(d []byte, bit int) byte {
	if bit < 0 || bit+8 > len(d)*8 {
		return 0
	}
	i, off := bit/8, uint(bit%8)
	if off == 0 {
		return d[i]
	}
	return d[i]<<off | d[i+1]>>(8-off)
}

func readU16BE(d []byte, bit int) (uint16, bool) {
	if bit < 0 || bit+16 > len(d)*8 {
		return 0, false
	}
	return uint16(byteAtBit(d, bit))<<8 | uint16(byteAtBit(d, bit+8)), true
}

func readU32BE(d []byte, bit int) (uint32, bool) {
	if bit < 0 || bit+32 > len(d)*8 {
		return 0, false
	}
	var v uint32
	for i := 0; i < 4; i++ {
		v = v<<8 | uint32(byteAtBit(d, bit+i*8))
	}
	return v, true
}
