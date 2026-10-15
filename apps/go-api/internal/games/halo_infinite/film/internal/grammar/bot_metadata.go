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
// sous la largeur de personnalisation du build. Elle seule donne l equipe, dans le bloc de 44 octets,
// a la position que le profil du build etablit ([profile.PositionEquipeBotPour] : elle change entre
// HI_1_12_0 et HI_1_13_0). Une equipe n en est retenue que si la marche FERME : la derniere entree
// finit a moins d un octet de la fin du paquet (l ecrivain arrondit a l octet).
//
// # LES FRAGMENTS DU BALAYAGE
//
// Le bloc de 44 octets porte une seconde copie du nom, en UTF-16 PETIT-boutiste. Lue un octet plus
// loin comme du gros-boutiste, elle donne le nom ampute de son premier caractere (`43 Hollis` pour
// `343 Hollis`), et aux offsets negatifs de cette lecture slot et `bid` valent 0 : le balayage y voit
// un bot de plus. Sur les builds ou le nom suit un octet non nul du bloc (HI_1_10_0 a HI_1_12_0,
// chaque entree), le fragment est systematique. Dans un paquet qui FERME, un nom balaye que la marche
// ne lit pas et qui est le suffixe strict d un nom qu elle lit est un tel fragment : il est ecarte et
// compte ([PaquetBotMetadata.FragmentsEcartes]), et il n entre pas au roster.
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
	"strings"
	"unicode/utf16"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
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

// LectureDesBots : ce que le profil du build dit de la lecture d une entree BOT_METADATA — la largeur
// du bloc de personnalisation, qui place tout ce qui suit, et la position de l equipe dans le bloc de
// 44 octets (`fiche + 0x1400`). PersoConnue faux : la grammaire de l ecrivain ne lit rien ;
// EquipeConnue faux : elle lit les entrees, sans leur equipe.
type LectureDesBots struct {
	PersoBits    int
	PersoConnue  bool
	Equipe       profile.PositionEquipeBot
	EquipeConnue bool
}

// LectureDesBotsPour rend la lecture des entrees BOT_METADATA d un build, telle que le profil la
// donne.
func LectureDesBotsPour(build string) LectureDesBots {
	octets, persoConnue := profile.PersonnalisationOctets(build)
	pos, equipeConnue := profile.PositionEquipeBotPour(build)
	return LectureDesBots{PersoBits: octets * 8, PersoConnue: persoConnue, Equipe: pos,
		EquipeConnue: equipeConnue}
}

// EntreeDeBotBalayee est une entree que le balayage des noms trouve dans un paquet BOT_METADATA :
// son slot, son `bid`, son nom et le bit ou le nom commence.
type EntreeDeBotBalayee struct {
	Slot, BotID int
	Nom         string
	Bit         int
}

// EntreeDeBotEcrite est une entree que la grammaire de l ecrivain lit : son slot, son `bid`, son nom
// et, quand le profil du build en etablit la position (EquipeLue), les deux octets signes du bloc de
// 44 octets — l equipe et son jumeau, -1 pour aucune.
type EntreeDeBotEcrite struct {
	Slot, BotID    int
	Nom            string
	Equipe, Jumeau int
	EquipeLue      bool
}

// PaquetBotMetadata est un paquet BOT_METADATA lisible (au moins le mot du nombre de bots), dans
// l ordre du film.
type PaquetBotMetadata struct {
	// Instant est l horodatage du paquet — celui de l image-cle de son chunk quand il appartient a
	// l instantane de tete.
	Instant uint64
	// NBots est le nombre de bots que le paquet annonce (premier u32, gros-boutiste).
	NBots int
	// Balayees sont les entrees que le balayage des noms trouve, fragments ecartes.
	Balayees []EntreeDeBotBalayee
	// Ecrites sont les entrees que la grammaire de l ecrivain lit quand la largeur de
	// personnalisation est connue, et Ferme dit que sa marche ferme le paquet ; nil et faux sinon.
	Ecrites []EntreeDeBotEcrite
	Ferme   bool
	// FragmentsEcartes : les noms du balayage ecartes de ce paquet ferme comme fragments de la copie
	// petit-boutiste d un nom que la marche lit (cf. l en-tete).
	FragmentsEcartes int
}

// PaquetsBotMetadata rend les paquets BOT_METADATA lisibles du film, dans l ORDRE DU FILM — chunk par
// chunk, paquet par paquet : c est l ordre dans lequel `killsource` decouvre ses bots, et deux bots
// d un meme slot y gardent leur rang relatif. `lecture` est celle du build du film
// ([LectureDesBotsPour]).
func PaquetsBotMetadata(f *source.Film, lecture LectureDesBots) []PaquetBotMetadata {
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
			out = append(out, lirePaquetBotMetadata(p.Payload, instant, lecture))
		}
	}
	return out
}

// lirePaquetBotMetadata lit un payload BOT_METADATA d au moins quatre octets par ses deux lecteurs,
// puis ecarte du balayage les fragments d un paquet qui ferme.
func lirePaquetBotMetadata(pl []byte, instant uint64, lecture LectureDesBots) PaquetBotMetadata {
	n, _ := lireU32GrosBoutiste(pl, 0)
	out := PaquetBotMetadata{Instant: instant, NBots: int(n), Balayees: balayerLesEntreesDeBots(pl)}
	if lecture.PersoConnue {
		out.Ecrites, out.Ferme = lireLesEntreesDeBots(pl, lecture)
	}
	if out.Ferme {
		out.Balayees, out.FragmentsEcartes = ecarterLesFragments(out.Balayees, out.Ecrites)
	}
	return out
}

// ecarterLesFragments retire du balayage d un paquet FERME chaque nom que la marche ne lit pas et qui
// est le suffixe strict d un nom qu elle lit (cf. l en-tete), et rend combien il en retire.
func ecarterLesFragments(balayees []EntreeDeBotBalayee, ecrites []EntreeDeBotEcrite) ([]EntreeDeBotBalayee, int) {
	out := make([]EntreeDeBotBalayee, 0, len(balayees))
	for _, b := range balayees {
		if fragmentDUnNomEcrit(b.Nom, ecrites) {
			continue
		}
		out = append(out, b)
	}
	return out, len(balayees) - len(out)
}

// fragmentDUnNomEcrit dit qu un nom balaye n est ecrit par aucune entree et qu il termine un nom ecrit
// plus long.
func fragmentDUnNomEcrit(nom string, ecrites []EntreeDeBotEcrite) bool {
	suffixe := false
	for _, e := range ecrites {
		if e.Nom == nom {
			return false
		}
		if len(e.Nom) > len(nom) && strings.HasSuffix(e.Nom, nom) {
			suffixe = true
		}
	}
	return suffixe
}

// lireLesEntreesDeBots lit un payload BOT_METADATA par la grammaire de l ecrivain (cf. l en-tete),
// sous la largeur de personnalisation du build ; l equipe se lit a la position du build quand elle
// est etablie. Faux quand la marche ne ferme pas le paquet.
func lireLesEntreesDeBots(pl []byte, lecture LectureDesBots) ([]EntreeDeBotEcrite, bool) {
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
		if !decodeSlotListes(r, &e) || !decodeSlotCorps(r, &e, lecture.PersoBits) {
			return nil, false
		}
		entree := EntreeDeBotEcrite{Slot: slot, BotID: bid, Nom: e.slot.Gamertag}
		if lecture.EquipeConnue {
			entree.Equipe = int(int8(e.bloc44[lecture.Equipe.Equipe])) //nolint:gosec // octet signe, -1 = aucune
			entree.Jumeau = int(int8(e.bloc44[lecture.Equipe.Jumeau])) //nolint:gosec // idem
			entree.EquipeLue = true
		}
		out = append(out, entree)
	}
	return out, len(pl)*8-r.br.BitPos() < 8
}

// lireLeBloc44 lit le bloc de 44 octets bruts d une fiche ; sa disposition depend du build
// ([profile.PositionEquipeBotPour]).
func lireLeBloc44(r *slotReader, e *slotEnr) {
	for i := range e.bloc44 {
		e.bloc44[i] = byte(r.bits(8))
	}
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
