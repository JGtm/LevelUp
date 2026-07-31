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
}

// botMeta : ce que le film declare sur ses bots, tous chunks confondus.
type botMeta struct {
	NBots int // max des nbBots vus (le roster peut se remplir en cours de film)
	NPkt  int
	Bots  []bot
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
func loadBotMeta(f *film) botMeta {
	m := botMeta{}
	seen := map[[2]int]bool{}
	for i := range f.packets {
		p := &f.packets[i]
		if p.typ != packetTypeBotMeta || len(p.payload) < 4 {
			continue
		}
		m.NPkt++
		n := int(uint32(p.payload[0])<<24 | uint32(p.payload[1])<<16 |
			uint32(p.payload[2])<<8 | uint32(p.payload[3]))
		if n < 0 || n > botMaxSlot {
			continue
		}
		if n > m.NBots {
			m.NBots = n
		}
		for _, b := range scanBotEntries(p.payload) {
			k := [2]int{b.Slot, b.BotID}
			if seen[k] {
				continue
			}
			seen[k] = true
			m.Bots = append(m.Bots, b)
		}
	}
	sort.Slice(m.Bots, func(i, j int) bool { return m.Bots[i].Slot < m.Bots[j].Slot })
	return m
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
