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
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
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
	// equipe / equipeLue : l equipe que ses paquets FERMES lui donnent (botmeta_equipe.go).
	equipe    int
	equipeLue bool
}

// BotDeclaration est UN intervalle pendant lequel BOT_METADATA declare un bot : du premier paquet
// qui le declare (inclus) au premier paquet COMPLET suivant qui ne le declare plus (exclu).
type BotDeclaration struct {
	// FromUS est l instant du premier paquet qui le declare — celui de l IMAGE-CLE de son chunk
	// quand ce paquet appartient a l instantane de tete (cf. [grammar.PaquetsBotMetadata]).
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
	// Team : l equipe que l entree BOT_METADATA du bot ECRIT (botmeta_equipe.go) — -1 pour aucune,
	// 0..8 sinon, la valeur du designateur d equipe de `ti=9`. NIL quand aucun paquet ferme ne la
	// donne, ou que deux la donnent differente : l absence n est pas « aucune equipe ». Aucune ligne
	// de kill ne la lit.
	Team *int `json:",omitempty"`
}

// entree rend la forme publiee d un bot.
func (b bot) entree() BotEntry {
	e := BotEntry{Slot: b.Slot, BotID: b.BotID, Name: b.Name,
		Declarations: append([]BotDeclaration(nil), b.declarations...)}
	if b.equipeLue {
		v := b.equipe
		e.Team = &v
	}
	return e
}

// botMeta : ce que le film declare sur ses bots, tous chunks confondus.
type botMeta struct {
	NBots int // max des nbBots vus (le roster peut se remplir en cours de film)
	NPkt  int
	Bots  []bot
	// Incomplets : paquets dont le scan n a pas retrouve `nbBots` entrees. Ils n ont ferme aucune
	// declaration (cf. l en-tete).
	Incomplets int
	// Equipes : le bilan de la lecture de l equipe des bots (botmeta_equipe.go).
	Equipes EquipesDesBots
}

// botMaxSlot : le plus grand nombre de bots qu un paquet peut annoncer ; au-dela, le paquet est ecarte.
const botMaxSlot = 64

// loadBotMeta : agrege les paquets type 12 du film, tels que la grammaire les lit
// ([grammar.PaquetsBotMetadata] : l instant de chacun, son `nbBots`, les entrees du balayage des noms).
//
// L AGREGAT EST INCHANGE (lot M2.1) : memes bots, meme ordre, meme `NBots` — c est lui qu epingle
// le roster du kill-feed, et aucune ligne de kill ne doit bouger. Ce qui s ajoute est l INSTANT :
// les paquets sont parcourus dans l ordre du film et chaque bot garde ses intervalles de
// declaration (cf. l en-tete).
func loadBotMeta(paquets []grammar.PaquetBotMetadata) botMeta {
	m := botMeta{}
	rang := map[[2]int]int{}       // (slot, bid) -> position dans m.Bots
	ouverts := map[[2]int]uint64{} // declares au dernier paquet lu -> debut de leur declaration
	for _, pi := range paquets {
		m.NPkt++
		n := pi.NBots
		if n < 0 || n > botMaxSlot {
			continue
		}
		if n > m.NBots {
			m.NBots = n
		}
		entrees := botsBalayes(pi.Balayees)
		declares := make(map[[2]int]bool, len(entrees))
		for _, b := range entrees {
			k := [2]int{b.Slot, b.BotID}
			declares[k] = true
			if _, vu := rang[k]; !vu {
				rang[k] = len(m.Bots)
				m.Bots = append(m.Bots, b)
			}
			if _, ouvert := ouverts[k]; !ouvert {
				ouverts[k] = pi.Instant
			}
		}
		if len(entrees) != n {
			m.Incomplets++ // un bot manque a la LECTURE n est pas un bot parti : rien ne se ferme
			continue
		}
		fermerLesAbsents(&m, rang, ouverts, declares, pi.Instant)
	}
	for _, k := range clesTriees(ouverts) {
		i := rang[k]
		m.Bots[i].declarations = append(m.Bots[i].declarations, BotDeclaration{FromUS: ouverts[k]})
	}
	trierBotsParSlot(m.Bots)
	return m
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
	slices.SortFunc(out, func(a, b [2]int) int { // cles d une map : uniques
		return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1]))
	})
	return out
}

// botsBalayes rend les entrees que le balayage des noms a trouvees dans un paquet, dans la forme du
// decodeur ; la position du nom sert a ecarter la copie bit-decalee.
func botsBalayes(entrees []grammar.EntreeDeBotBalayee) []bot {
	out := make([]bot, 0, len(entrees))
	for _, e := range entrees {
		out = append(out, bot{Slot: e.Slot, BotID: e.BotID, Name: e.Nom, bitPos: e.Bit})
	}
	return out
}
