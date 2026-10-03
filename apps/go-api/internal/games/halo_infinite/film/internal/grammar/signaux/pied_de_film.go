package signaux

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/finalise"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// pied_de_film.go — les EVENEMENTS D'OBJECTIF DU PIED DE FILM (type_hint 10), horodates sur
// l'horloge du match : l'acteur (xuid), son slot de pied et son equipe.
//
// Faits de decode VALIDES (RESEARCH_THEATER_RE.md §M, §M-ter) : le pied = chunk de plus haut
// numero, chunk_type 3 ; ses events th=10 = les interactions objectif (t = horloge BE aux octets
// 48-51, slot a l'octet 36, ÉQUIPE A L'OCTET 37, xuid). L'equipe est donc DANS le film : mesure du
// 2026-09-13, 665 evenements sur 665, quatorze films de modes a objectif — cf. [FooterEvent.Team].
//
// LA CONVENTION DE BORD EST NOMMEE : ce lecteur S'ARRETE a la fin du tampon SANS bourrer —
// [source.OctetAuBit] et [source.U64LEAuBit], jamais une lecture qui bourre a zero et rendrait
// d'autres valeurs sur les derniers bits d'un bloc.
//
// DEUX LECTEURS DU MEME PIED VIVENT DANS LA GRAMMAIRE : celui-ci (les seuls blocs th=10, equipe a
// l'octet 37) et le parseur des temps forts (`grammar.ParseHighlightEvents`, tous les types
// d'evenement, decoupage du gamertag par version). Ils ne lisent pas les memes champs ; leurs bornes
// de xuid sont les memes, recopiees ([minXUID], [maxXUID]) parce que ce paquet n'importe pas
// `grammar` (cf. la doc du paquet).

// Bornes plausibles d'un xuid Halo (filtre anti-bruit du balayage du pied).
const (
	minXUID = uint64(2e15)
	maxXUID = uint64(3e15)
)

// Offsets, EN OCTETS DEPUIS LE DEBUT DU BLOC DE 60, des champs du bloc d'evenement du pied.
//
// L'EQUIPE EST A L'OCTET 37, PAS A L'OCTET 55 (correctif du 2026-09-14, lot 1.1). Ce n'est pas
// un choix de repli, c'est une lecture : le balayage AVEUGLE des soixante octets du bloc, trois
// lectures chacun (valeur brute, valeur moins un, bit de poids faible) confrontees a l'equipe
// PROUVEE DANS LE FILM SEUL (rang de la table des slots de `chunk_00` -> xuid, i-eme entite
// ti=9 -> designateur d'equipe), donne QUATRE lectures en accord parfait sur cent quatre-vingts
// essayees, et ce sont les deux lectures de l'octet 37 et les deux de l'octet 38 :
// 665 evenements sur 665, quatorze films de modes a objectif, sept builds.
//
// L'octet 55 — celui que le lecteur d'objectifs lisait sous le nom `teamRaw` — vaut 0 sur les 665.
// Son « accord » de 318/665 etait MECANIQUE : c'est le nombre d'evenements dont l'acteur est
// d'equipe 0. Le commentaire « NON fiable sur certains matchs » decrivait donc un champ
// TOUJOURS faux pour l'equipe 1, pas un champ intermittent.
//
// L'octet 38 est un DOUBLON OBSERVE (meme 665/665, memes valeurs terme a terme). Il n'est PAS
// lu : deux lectures d'un meme fait se contrediraient un jour sans que rien ne le dise, et
// rien n'etablit laquelle des deux est l'ecriture et laquelle la copie.
//
// Mesure : `.ai/V7.5/film_re/NOTE_RESIDUS_CHUNK00_2026-09-13.md` §6 ; instrument (oracle
// corpus, rejouable) `TestResidusPiedOctetEquipe` (etiquette `research`).
const (
	footerBlockBytes = 60 // taille du bloc d'evenement qui precede le marqueur de fin
	footerByteSlot   = 36 // b36 : 0..3, stable par xuid — PAS le player_index (mesure)
	footerByteTeam   = 37 // b37 : l'index d'equipe, 665/665
	footerByteType   = 47 // b47 : type_hint, filtre sur 10
	footerByteTime   = 48 // b48..b51 : horloge du match (ms), gros-boutiste
)

// FooterEvent = un evenement highlight de type_hint==10 (interaction objectif) decode depuis le
// pied de film (chunk de type 3).
type FooterEvent struct {
	// TimeMS est l'instant de l'interaction sur l'horloge du match (octets 48 a 51, BE).
	TimeMS int
	// Slot est l'octet 36 : 0..3, stable par xuid sur un match. Ce n'est PAS le player_index
	// (mesure : valeurs 0..3 reparties sur des films a 8 comme a 24 joueurs).
	Slot int
	// Team est l'index d'equipe BRUT tel que le film l'ecrit a l'octet 37 du bloc — jamais un
	// libelle, jamais une valeur de la base.
	//
	// IL N'Y A PAS DE VALEUR « ABSENTE », et c'est voulu (D14 : pas de repli qui ne peut pas
	// tirer). Un [FooterEvent] n'existe que si [decodeTh10Block] a trouve le marqueur de fin de
	// son bloc ; ses soixante octets sont alors dans les bornes par construction, et l'octet 37
	// est toujours lu. Un sentinelle -1 que rien ne peut produire se lirait comme un contrat
	// que la grammaire ne porte pas — « le film est parfois muet ici ».
	Team int
	// XUID de l'acteur, valeur brute du film.
	XUID uint64
}

// FooterEvents rend les evenements th=10 du PIED d'un film deja charge, tries par instant.
//
// C'est le point d'entree unique du pied : il choisit le chunk (plus haut numero de type 3, cf.
// [footerData]) puis le balaye. Un film sans pied au manifeste rend nil — pas d'erreur, et pas
// de chunk devine.
func FooterEvents(film *source.Film) []FooterEvent {
	footer, ok := footerData(film)
	if !ok {
		return nil
	}
	return scanTh10Events(footer)
}

// footerData renvoie le contenu DECOMPRESSE du pied (chunk de plus haut numero, chunk_type 3), ou
// (nil,false). Si le pied n'est pas en cache, l'equipe par evenement manque -> degradation
// gracieuse : le film charge ne porte QUE les chunks reellement presents, donc un pied manquant au
// cache n'a pas d'entree.
func footerData(film *source.Film) ([]byte, bool) {
	footerPos, footerIdx := -1, -1
	for _, c := range manifestChunks(film) {
		if finalise.EstTempsForts(c.meta.ChunkType) && c.meta.Index > footerIdx {
			footerPos, footerIdx = c.pos, c.meta.Index
		}
	}
	if footerPos < 0 {
		return nil, false
	}
	return film.Chunk(footerPos), true
}

// scanTh10Events extrait tous les events th=10 d'un chunk decompresse (typiquement
// le pied, chunk_type 3). Adapte de evDump/thTally (filmx) : on localise chaque
// XUID (prefixe 0x2d/0x25, suffixe 0xc0, valeur LE plausible), on cherche le
// end-marker [00 00 2e e0] de son bloc, on recule de 60 octets, on lit th@b47,
// t=BE@b48-51, slot@b36, equipe@b37. Filtre sur th==10 et deduplique par bloc.
func scanTh10Events(data []byte) []FooterEvent {
	total := len(data) * 8
	var lus []piedLu
	seen := map[int]bool{}
	for ms := 8; ms <= total-8; ms++ {
		if source.OctetAuBit(data, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		if p := source.OctetAuBit(data, xe); p != 0x2d && p != 0x25 {
			continue
		}
		xstart := xe - 64
		if seen[xstart] {
			continue
		}
		x := source.U64LEAuBit(data, xstart)
		if x <= minXUID || x >= maxXUID {
			continue
		}
		seen[xstart] = true
		if ev, ok := decodeTh10Block(data, xstart, total); ok {
			ev.XUID = x
			lus = append(lus, piedLu{ev: ev, bit: xstart})
		}
	}
	trierPied(lus)
	if len(lus) == 0 {
		return nil // nil : aucun evenement lu
	}
	out := make([]FooterEvent, 0, len(lus))
	for _, l := range lus {
		out = append(out, l.ev)
	}
	return out
}

// decodeTh10Block, a partir du debut d'XUID, cherche le end-marker du bloc
// d'event puis decode le bloc de 60 octets le precedant. Renvoie ok=false si le
// bloc n'est pas un th=10. (Le xuid est rempli par l'appelant.)
func decodeTh10Block(data []byte, xstart, total int) (FooterEvent, bool) {
	win := min(xstart+20000, total)
	for b := xstart; b <= win-32; b++ {
		if source.OctetAuBit(data, b) == 0 && source.OctetAuBit(data, b+8) == 0 &&
			source.OctetAuBit(data, b+16) == 0x2e && source.OctetAuBit(data, b+24) == 0xe0 {
			ebs := b - footerBlockBytes*8
			if ebs < xstart {
				return FooterEvent{}, false
			}
			if int(source.OctetAuBit(data, ebs+footerByteType*8)) != 10 {
				return FooterEvent{}, false
			}
			return FooterEvent{
				TimeMS: footerTimeMS(data, ebs),
				Slot:   int(source.OctetAuBit(data, ebs+footerByteSlot*8)),
				Team:   int(source.OctetAuBit(data, ebs+footerByteTeam*8)),
			}, true
		}
	}
	return FooterEvent{}, false
}

// footerTimeMS lit l'horloge du match (ms) aux octets 48 a 51 du bloc, gros-boutiste.
func footerTimeMS(data []byte, ebs int) int {
	return int(source.OctetAuBit(data, ebs+footerByteTime*8))<<24 |
		int(source.OctetAuBit(data, ebs+(footerByteTime+1)*8))<<16 |
		int(source.OctetAuBit(data, ebs+(footerByteTime+2)*8))<<8 |
		int(source.OctetAuBit(data, ebs+(footerByteTime+3)*8))
}

// piedLu est un evenement du pied et la position de bit de son XUID dans le chunk.
type piedLu struct {
	ev  FooterEvent
	bit int
}

// trierPied range les evenements du pied dans un ordre TOTAL : instant, puis position du XUID dans
// le chunk — unique (`seen`). Deux evenements de la meme milliseconde ordonnent les actions
// publiees (numero `Seq` dense) et le choix de l'acteur d'une capture garde le PREMIER du plus
// grand instant : leur rang doit tenir au film, pas au tri.
func trierPied(lus []piedLu) {
	slices.SortFunc(lus, func(a, b piedLu) int {
		return cmp.Or(cmp.Compare(a.ev.TimeMS, b.ev.TimeMS), cmp.Compare(a.bit, b.bit))
	})
}
