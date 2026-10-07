package grammar

// bot_metadata.go — LES PAQUETS BOT_METADATA (type 12) : le film nomme ses bots (RE_LOG 7ter.62), et
// l entree de chacun porte son equipe. Lecture descendue de `facts/killsource` (`botmeta.go`,
// `botmeta_equipe.go`) au lot 2.7.c1 de la representation intermediaire, a l identique ;
// `killsource` en garde l agregat (declarations, roster).
//
// # DEUX LECTEURS DU MEME PAQUET
//
// Le BALAYAGE ([balayerLesEntreesDeBots]) cherche les noms bit a bit : un nom est un run UTF-16BE
// d ASCII imprimable ferme par 0x0000, le slot et le `bid` se lisent a deux offsets negatifs
// constants. C est le lecteur historique, qui fixe le roster de killsource : le stride fixe de 2076
// octets est vrai pour un bot et faux des le deuxieme (une entree y est decalee d un demi-octet).
//
// La GRAMMAIRE DE L ECRIVAIN ([lireLesEntreesDeBots]) lit le paquet en entier, entree par entree :
// `FUN_14299bda0` ecrit W(32) le nombre d entrees, puis par bot W(32) son index absolu, W(32) son
// slot, W(32) son `bid` et le corps de sa fiche (`FUN_1407edea8`) — le corps des fiches de joueur que
// la table de `chunk_00` et le paquet de type 8 portent aussi ([decodeSlotListes], [decodeSlotCorps]),
// sous la largeur de personnalisation du build. Elle seule donne l equipe, dans le bloc de 44 octets.
// Une equipe n en est retenue que si la marche FERME : la derniere entree finit a moins d un octet
// de la fin du paquet (l ecrivain arrondit a l octet).
//
// # L INSTANT D UN PAQUET
//
// Le paquet est un ETAT : reecrit en tete de chaque chunk et a chaque changement de bots. Le paquet
// de l INSTANTANE DE TETE d un chunk (apres son image-cle, avant sa premiere trame) porte l etat des
// bots A L IMAGE-CLE ; il est seulement ecrit apres elle (390 us sur le chunk 2 de `b1ad85eb`). Son
// instant est donc celui de l image-cle (sonde P4, 2026-09-23).

import (
	"cmp"
	"slices"
	"unicode/utf16"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// PacketTypeBotMetadata est le type du paquet BOT_METADATA.
const PacketTypeBotMetadata = 12

// Le balayage des noms : offsets, en bits, du slot et du `bid` AVANT le debut du nom ; bornes du nom
// (en unites UTF-16), du slot et du `bid`.
const (
	botSlotAvantLeNom = 0x74 * 8
	botBIDAvantLeNom  = 0x70 * 8
	botNomMin         = 4
	botNomMax         = 48
	botSlotMax        = 64
	botBIDMax         = 4096
)

// Position, dans le bloc de 44 octets d une fiche (`fiche + 0x1400`, recopie par `FUN_1424d512c` en
// `configuration + 0xCC0`), de l equipe (`+ 0xCE5`, posee sur le joueur par `FUN_140ad37f8`) et de
// son jumeau (`+ 0xCE6`) : deux octets signes, -1 pour aucune.
const (
	slotEquipeDansBloc44 = 0x25 * 8
	slotJumeauDansBloc44 = 0x26 * 8
)

// EntreeDeBotBalayee est une entree que le balayage des noms trouve dans un paquet BOT_METADATA :
// son slot, son `bid`, son nom et le bit ou le nom commence.
type EntreeDeBotBalayee struct {
	Slot, BotID int
	Nom         string
	Bit         int
}

// EntreeDeBotEcrite est une entree que la grammaire de l ecrivain lit : son slot, son `bid`, son nom
// et les deux octets signes du bloc de 44 octets — l equipe et son jumeau.
type EntreeDeBotEcrite struct {
	Slot, BotID    int
	Nom            string
	Equipe, Jumeau int
}

// PaquetBotMetadata est un paquet BOT_METADATA lisible (au moins le mot du nombre de bots), dans
// l ordre du film.
type PaquetBotMetadata struct {
	// Instant est l horodatage du paquet — celui de l image-cle de son chunk quand il appartient a
	// l instantane de tete.
	Instant uint64
	// NBots est le nombre de bots que le paquet annonce (premier u32, gros-boutiste).
	NBots int
	// Balayees sont les entrees que le balayage des noms trouve.
	Balayees []EntreeDeBotBalayee
	// Ecrites sont les entrees que la grammaire de l ecrivain lit quand la largeur de
	// personnalisation est connue, et Ferme dit que sa marche ferme le paquet ; nil et faux sinon.
	Ecrites []EntreeDeBotEcrite
	Ferme   bool
}

// PaquetsBotMetadata rend les paquets BOT_METADATA lisibles du film, dans l ORDRE DU FILM — chunk par
// chunk, paquet par paquet : c est l ordre dans lequel `killsource` decouvre ses bots, et deux bots
// d un meme slot y gardent leur rang relatif. `persoBits` est la largeur du bloc de personnalisation
// du build ; `persoConnue` faux (build absent du profil) : la grammaire de l ecrivain ne lit rien.
func PaquetsBotMetadata(f *source.Film, persoBits int, persoConnue bool) []PaquetBotMetadata {
	var out []PaquetBotMetadata
	chunk, imageCle, enTete := -1, uint64(0), false
	for _, p := range f.AllPackets() {
		if p.Chunk != chunk {
			chunk, imageCle, enTete = p.Chunk, 0, true
		}
		switch {
		case p.Type == int(PacketTypeKeyframe) && enTete && imageCle == 0:
			imageCle = p.TS
		case p.Type == int(PacketTypeDelta):
			enTete = false
		case p.Type == PacketTypeBotMetadata && len(p.Payload) >= 4:
			instant := p.TS
			if enTete && imageCle != 0 {
				instant = imageCle
			}
			out = append(out, lirePaquetBotMetadata(p.Payload, instant, persoBits, persoConnue))
		}
	}
	return out
}

// lirePaquetBotMetadata lit un payload BOT_METADATA d au moins quatre octets par ses deux lecteurs.
func lirePaquetBotMetadata(pl []byte, instant uint64, persoBits int, persoConnue bool) PaquetBotMetadata {
	n, _ := lireU32GrosBoutiste(pl, 0)
	out := PaquetBotMetadata{Instant: instant, NBots: int(n), Balayees: balayerLesEntreesDeBots(pl)}
	if persoConnue {
		out.Ecrites, out.Ferme = lireLesEntreesDeBots(pl, persoBits)
	}
	return out
}

// lireLesEntreesDeBots lit un payload BOT_METADATA par la grammaire de l ecrivain (cf. l en-tete),
// sous la largeur de personnalisation `persoBits`. Faux quand la marche ne ferme pas le paquet.
func lireLesEntreesDeBots(pl []byte, persoBits int) ([]EntreeDeBotEcrite, bool) {
	r := &slotReader{br: LecteurSur(pl), fin: len(pl) * 8, ok: true}
	n := int(r.bits(32))
	if !r.ok || n > botSlotMax {
		return nil, false
	}
	out := make([]EntreeDeBotEcrite, 0, n)
	for range n {
		r.saute(32) // index absolu du bot
		slot, bid := int(r.bits(32)), int(r.bits(32))
		var e slotEnr
		if !decodeSlotListes(r, &e) || !decodeSlotCorps(r, &e, persoBits) {
			return nil, false
		}
		out = append(out, EntreeDeBotEcrite{Slot: slot, BotID: bid, Nom: e.slot.Gamertag,
			Equipe: e.equipe, Jumeau: e.jumeau})
	}
	return out, len(pl)*8-r.br.BitPos() < 8
}

// lireLeBloc44 lit le bloc de 44 octets d une fiche : l equipe et son jumeau, le reste saute.
func lireLeBloc44(r *slotReader, e *slotEnr) {
	r.saute(slotEquipeDansBloc44)
	e.equipe = int(int8(r.bits(8))) //nolint:gosec // octet signe de l ecrivain, -1 = aucune
	e.jumeau = int(int8(r.bits(8))) //nolint:gosec // idem
	r.saute(slotBloc44Bits - slotJumeauDansBloc44 - 8)
}

// balayerLesEntreesDeBots enumere les entrees d un payload BOT_METADATA SANS hypothese de stride : un
// nom est un run UTF-16BE d ASCII imprimable ferme par 0x0000 ; slot et `bid` se lisent aux deux
// offsets negatifs constants. Le terminateur obligatoire elimine les sous-chaines.
func balayerLesEntreesDeBots(pl []byte) []EntreeDeBotBalayee {
	var out []EntreeDeBotBalayee
	total := len(pl) * 8
	for bit := 0; bit+16 <= total; bit++ {
		nom, suivant := lireUnNomDeBot(pl, bit)
		if nom == "" {
			continue
		}
		s, okS := lireU32GrosBoutiste(pl, bit-botSlotAvantLeNom)
		d, okD := lireU32GrosBoutiste(pl, bit-botBIDAvantLeNom)
		if okS && okD && s < botSlotMax && d < botBIDMax {
			out = append(out, EntreeDeBotBalayee{Slot: int(s), BotID: int(d), Nom: nom, Bit: bit})
		}
		bit = suivant
	}
	return premiereCopieSeulement(out)
}

// lireUnNomDeBot rend le nom qui commence au bit `bit`, et la position de son terminateur ; "" si ce
// n est pas un nom (trop court, caractere non imprimable, terminateur absent).
func lireUnNomDeBot(pl []byte, bit int) (string, int) {
	nom := make([]byte, 0, botNomMax)
	p := bit
	for len(nom) < botNomMax {
		c, ok := lireU16GrosBoutiste(pl, p)
		if !ok || c < 0x20 || c > 0x7e {
			break
		}
		nom = append(nom, byte(c))
		p += 16
	}
	if len(nom) < botNomMin {
		return "", p
	}
	if c, ok := lireU16GrosBoutiste(pl, p); !ok || c != 0 {
		return "", p
	}
	u := make([]uint16, len(nom))
	for i, c := range nom {
		u[i] = uint16(c)
	}
	return string(utf16.Decode(u)), p
}

// premiereCopieSeulement garde, par nom, l occurrence de plus BASSE position : le nom d un bot
// apparait DEUX FOIS par entree, la seconde copie etant bit-decalee, et aux offsets negatifs de cette
// seconde copie slot et `bid` se lisent en zeros — un faux `slot=0 bid(0.0)` qui volerait l indice 0
// a un humain. Ordre total : position de bit, puis nom.
func premiereCopieSeulement(in []EntreeDeBotBalayee) []EntreeDeBotBalayee {
	premiere := map[string]EntreeDeBotBalayee{}
	for _, b := range in {
		if e, ok := premiere[b.Nom]; !ok || b.Bit < e.Bit {
			premiere[b.Nom] = b
		}
	}
	out := make([]EntreeDeBotBalayee, 0, len(premiere))
	for _, b := range premiere {
		out = append(out, b)
	}
	slices.SortFunc(out, func(a, b EntreeDeBotBalayee) int { // nom : cle de `premiere`, unique
		return cmp.Or(cmp.Compare(a.Bit, b.Bit), cmp.Compare(a.Nom, b.Nom))
	})
	return out
}

// lireU16GrosBoutiste lit un u16 gros-boutiste a une position de bit quelconque.
func lireU16GrosBoutiste(d []byte, bit int) (uint16, bool) {
	if bit < 0 || bit+16 > len(d)*8 {
		return 0, false
	}
	return uint16(source.OctetAuBit(d, bit))<<8 | uint16(source.OctetAuBit(d, bit+8)), true
}

// lireU32GrosBoutiste lit un u32 gros-boutiste a une position de bit quelconque.
func lireU32GrosBoutiste(d []byte, bit int) (uint32, bool) {
	if bit < 0 || bit+32 > len(d)*8 {
		return 0, false
	}
	var v uint32
	for i := range 4 {
		v = v<<8 | uint32(source.OctetAuBit(d, bit+i*8))
	}
	return v, true
}
