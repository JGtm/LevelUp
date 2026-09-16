// Package objectives — décodage des timelines d'events objectif (CTF
// captures, Strongholds/KOTH zones, Oddball crâne) depuis les chunks film Halo,
// vers des []domain.ObjectiveEvent (mode-agnostique, cf.
// .ai/PLAN_WEAPON_ATTRIBUTION_V3.md §10).
//
// Algos PURS : zéro accès DB, zéro Streamlit, ET DEPUIS LE 2026-09-02 ZÉRO LECTURE DE FILM.
// Les neuf points d'entrée reçoivent un `*source.Film` DÉJÀ CHARGÉ — chunks décompressés
// et paquets découpés une fois pour toute la cuisson (lot 1 de PLAN_CUISSON_PERF, item 1.5).
// Ce paquet n'a plus ni inflate ni marcheur de paquets : il en portait le troisième, et les
// trois divergeaient (mesure sur 1 378 films, §2b de MESURES_CUISSON_PERF.md). La résolution
// xuid->team reste fournie par l'appelant (Roster, issu de match_participants).
//
// LES CHUNKS CONSOMMÉS SONT CEUX DU MANIFESTE, et rien d'autre (cf. [manifestChunks]) : c'est
// exactement ce que faisait l'ancienne `FilmSource`, qui itérait l'index du manifeste.
//
// film.go porte les PRIMITIVES de décodage adaptées des décodeurs jetables
// validés (tmp_film_explore/{filmx,ctfcap,ctfsig,t2score,firemap}) :
//   - sélection des paquets FRAME parmi ceux que `source` a découpés ;
//   - extraction des events footer th=10 (interactions objectif horodatées) ;
//   - détection des bursts de capture CTF (échelle 6-tiers) avec match_ms.
//
// (Retrait 2026-08-01, lot C de PLAN_DETTE_AVANT_MERGE : `extractType2` — extraction du
// payload du premier paquet TYPE_2 — n'avait aucun appelant. La marche de paquets vivait dans
// walkFrames, qui parcourait le même conteneur ; elle ne portait pas de grammaire coûteuse à
// rétablir. `readBitsBE`, retirée au même titre, est REVENUE à l'intégration de
// `feat/re-mode-score` le 2026-08-05 : cette branche est partie d'AVANT le retrait et lui a
// donné son appelant — `statborg.go`, qui lit la chaîne d'enregistrements du statborg à des
// offsets non alignés. Le retrait était exact au moment où il a été fait ; il ne l'est plus.
// `walkFrames` et `decompressChunk` ont suivi le 2026-09-02, remplacées par `source`.)
//
// Faits de décode VALIDÉS (RESEARCH_THEATER_RE.md §M, §M-ter) :
//   - footer = chunk de plus haut index, chunk_type 3 ; ses events th=10 = les
//     interactions objectif (t=horloge BE @octets 48-51, slot b36, ÉQUIPE À L'OCTET 37,
//     xuid). L'équipe est donc DANS le film : mesure du 2026-09-13, 665 événements sur
//     665, quatorze films de modes à objectif — cf. [FooterEvent.Team].
//   - capture CTF = une FRAME re-transmettant la table objectif complète
//     (tiers==6) ; team de la capture = l'event th=10 de t MAX dans le cluster
//     coïncident. objective_id (zone/colline) non récupérable -> NULL.
package objectives

import (
	"bytes"
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// packetFrame = le type de paquet consommé ici : FRAME, snapshot/delta d'état horodaté (us).
// Le découpage lui-même (en-tête 16 octets, terminateur CHUNK_END) appartient à `source`.
const packetFrame = 0

// Types de chunk du MANIFESTE, tels que [source.ChunkMeta.ChunkType] les porte.
//
// ZÉRO N'EST PAS UN TYPE : c'est ce que `source.LoadDir` synthétise pour un `chunk_NN.bin`
// PRÉSENT au cache mais ABSENT du manifeste. Mesure du 2026-09-02 sur les 1 380 manifestes du
// cache : trois valeurs seulement — 1 pour l'en-tête (`chunk_00`, le registre), 2 pour les
// chunks de jeu, 3 pour le pied — et jamais 0. Un chunk hors manifeste n'a donc ni type ni
// `start_ms` connus, et ce paquet ne le consomme pas (cf. [manifestChunks]).
const (
	chunkTypeInconnu = 0
	chunkTypeJeu     = 2
	chunkTypePied    = 3
)

// manifestChunk = un chunk du film QUE LE MANIFESTE DÉCRIT : sa position dans le film (l'index
// que prennent [source.Film.Chunk] et [source.Film.Packets]) et ses métadonnées.
type manifestChunk struct {
	pos  int
	meta source.ChunkMeta
}

// manifestChunks rend les chunks du film décrits par le manifeste, dans l'ordre du film.
//
// POURQUOI CE FILTRE EXISTE, ET CE QU'IL PRÉSERVE. Avant le 2026-09-02, ce paquet recevait une
// `FilmSource` et itérait l'index du MANIFESTE : un fichier de chunk présent au cache mais
// absent du manifeste n'était jamais lu. Le film chargé par `source.LoadDir`, lui, porte
// TOUS les fichiers présents. Sans ce filtre, ces chunks-là seraient balayés avec un `start_ms`
// de zéro — donc datés faux. Un film du cache est dans ce cas (`7b0d89c4`, chunks 31 et 32).
//
// Un film sans aucune métadonnée de manifeste rend une liste VIDE : c'est le seul résultat
// honnête (rien n'est datable), et [chunksDatables] le journalise plutôt que de le taire.
func manifestChunks(film *source.Film) []manifestChunk {
	if film == nil {
		return nil
	}
	meta := film.Meta()
	out := make([]manifestChunk, 0, len(meta))
	for i, m := range meta {
		if m.ChunkType == chunkTypeInconnu {
			continue
		}
		out = append(out, manifestChunk{pos: i, meta: m})
	}
	return out
}

// chunksDatables rend les chunks du manifeste et JOURNALISE le cas où il n'y en a aucun.
//
// Un film chargé SANS manifeste porte des paquets mais aucun `start_ms` : rien n'y est datable.
// Se taire ferait lire « ce film ne porte rien » là où il faut lire « on ne sait pas dater ce
// film » — deux faits différents, et le second est réparable (le manifeste, lui, se retélécharge).
func chunksDatables(ctx context.Context, film *source.Film, matchID string) []manifestChunk {
	chunks := manifestChunks(film)
	if len(chunks) == 0 {
		slog.InfoContext(ctx, "objectives: film sans chunk décrit par le manifeste — rien à dater",
			"match_id", matchID, "chunks_du_film", filmChunkCount(film))
	}
	return chunks
}

// filmChunkCount rend le nombre de chunks du film, ou 0 pour un film absent — de quoi dire au
// journal si « aucun chunk du manifeste » veut dire « pas de film » ou « pas de manifeste ».
func filmChunkCount(film *source.Film) int {
	if film == nil {
		return 0
	}
	return film.NumChunks()
}

// framesOf rend les paquets FRAME (type 0) du chunk à la POSITION `pos` dans le film, dans
// l'ordre du chunk. C'est tout ce qui reste de l'ancien `walkFrames` : le découpage est fait
// une fois par `source`, il ne reste qu'à choisir le type.
func framesOf(film *source.Film, pos int) []source.Packet {
	pks := film.Packets(pos)
	out := make([]source.Packet, 0, len(pks))
	for _, p := range pks {
		if p.Type == packetFrame {
			out = append(out, p)
		}
	}
	return out
}

// Bornes plausibles d'un xuid Halo (filtre anti-bruit du scan footer).
const (
	minXUID = uint64(2e15)
	maxXUID = uint64(3e15)
)

// LES TROIS LECTEURS DU PIED DE FILM ONT DESCENDU DANS LA COUCHE SOURCE (lot 2.4.2, ADR 0034
// D-2). Ils formaient le TROISIEME des sept lecteurs de bits du depot ; ce paquet ne touche plus
// un octet de film autrement que par `source` :
//
//	source.BitsTronques(data, bitPos, n)  ->  source.BitsTronques(data, bitPos, n)
//	source.OctetAuBit(data, bit)     ->  source.OctetAuBit(data, bit)
//	source.U64LEAuBit(data, bit)    ->  source.U64LEAuBit(data, bit)
//
// LA CONVENTION DE BORD EST PRESERVEE, ET ELLE EST NOMMEE : ce lecteur-ci S ARRETE a la fin du
// tampon SANS bourrer — [source.BitsTronques], et surtout PAS [source.BitsAt], qui porte
// le bourrage a zero du moteur et rendrait d autres valeurs sur les derniers bits d un bloc.

// Offsets, EN OCTETS DEPUIS LE DÉBUT DU BLOC DE 60, des champs du bloc d'événement du pied.
//
// L'ÉQUIPE EST À L'OCTET 37, PAS À L'OCTET 55 (correctif du 2026-09-14, lot 1.1). Ce n'est pas
// un choix de repli, c'est une lecture : le balayage AVEUGLE des soixante octets du bloc, trois
// lectures chacun (valeur brute, valeur moins un, bit de poids faible) confrontées à l'équipe
// PROUVÉE DANS LE FILM SEUL (rang de la table des slots de `chunk_00` -> xuid, i-ème entité
// ti=9 -> désignateur d'équipe), donne QUATRE lectures en accord parfait sur cent quatre-vingts
// essayées, et ce sont les deux lectures de l'octet 37 et les deux de l'octet 38 :
// 665 événements sur 665, quatorze films de modes à objectif, sept builds.
//
// L'octet 55 — celui que ce paquet lisait sous le nom `teamRaw` — vaut 0 sur les 665. Son
// « accord » de 318/665 était MÉCANIQUE : c'est le nombre d'événements dont l'acteur est
// d'équipe 0. Le commentaire « NON fiable sur certains matchs » décrivait donc un champ
// TOUJOURS faux pour l'équipe 1, pas un champ intermittent.
//
// L'octet 38 est un DOUBLON OBSERVÉ (même 665/665, mêmes valeurs terme à terme). Il n'est PAS
// lu : deux lectures d'un même fait se contrediraient un jour sans que rien ne le dise, et
// rien n'établit laquelle des deux est l'écriture et laquelle la copie.
//
// Mesure : `.ai/V7.5/film_re/NOTE_RESIDUS_CHUNK00_2026-09-13.md` §6 ; instrument (oracle
// corpus, rejouable) `grammar.TestResidusPiedOctetEquipe`.
const (
	footerBlockBytes = 60 // taille du bloc d'événement qui précède le marqueur de fin
	footerByteSlot   = 36 // b36 : 0..3, stable par xuid — PAS le player_index (mesuré)
	footerByteTeam   = 37 // b37 : l'index d'équipe, 665/665
	footerByteType   = 47 // b47 : type_hint, filtré sur 10
	footerByteTime   = 48 // b48..b51 : horloge du match (ms), gros-boutiste
)

// FooterEvent = un événement highlight de type_hint==10 (interaction objectif) décodé depuis le
// pied de film (chunk de type 3).
//
// EXPORTÉ SANS CONSOMMATEUR HORS DE CE PAQUET, ET C'EST DÉLIBÉRÉ (lot 1.1.2 du
// `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`) : [FooterEvent.Team] est l'équipe que le film écrit,
// et le lot 1.7 la fera prendre par `domain.ObjectiveEvent.TeamID` à la place du roster de la
// base (décision V4 : le film est la seule source). Le champ vit en mémoire dans ce paquet et
// nulle part ailleurs — ni dans un document cuit, ni dans une colonne DuckDB.
type FooterEvent struct {
	// TimeMS est l'instant de l'interaction sur l'horloge du match (octets 48 à 51, BE).
	TimeMS int
	// Slot est l'octet 36 : 0..3, stable par xuid sur un match. Ce n'est PAS le player_index
	// (mesuré : valeurs 0..3 réparties sur des films à 8 comme à 24 joueurs).
	Slot int
	// Team est l'index d'équipe BRUT tel que le film l'écrit à l'octet 37 du bloc — jamais un
	// libellé, jamais une valeur de la base.
	//
	// IL N'Y A PAS DE VALEUR « ABSENTE », et c'est voulu (D14 : pas de repli qui ne peut pas
	// tirer). Un [FooterEvent] n'existe que si [decodeTh10Block] a trouvé le marqueur de fin de
	// son bloc ; ses soixante octets sont alors dans les bornes par construction, et l'octet 37
	// est toujours lu. Un sentinelle -1 que rien ne peut produire se lirait comme un contrat
	// que la grammaire ne porte pas — « le film est parfois muet ici ».
	Team int
	// XUID de l'acteur, valeur brute du film.
	XUID uint64
}

// FooterEvents rend les événements th=10 du PIED d'un film déjà chargé, triés par instant.
//
// C'est le point d'entrée unique du pied : il choisit le chunk (plus haut index de type 3, cf.
// [footerData]) puis le balaye. Un film sans pied au manifeste rend nil — pas d'erreur, et pas
// de chunk deviné.
func FooterEvents(film *source.Film) []FooterEvent {
	footer, ok := footerData(film)
	if !ok {
		return nil
	}
	return scanTh10Events(footer)
}

// scanTh10Events extrait tous les events th=10 d'un chunk décompressé (typiquement
// le footer chunk_type 3). Adapté de evDump/thTally (filmx) : on localise chaque
// XUID (préfixe 0x2d/0x25, suffixe 0xc0, valeur LE plausible), on cherche le
// end-marker [00 00 2e e0] de son bloc, on recule de 60 octets, on lit th@b47,
// t=BE@b48-51, slot@b36, équipe@b37. Filtré sur th==10 et dédupliqué par bloc.
func scanTh10Events(data []byte) []FooterEvent {
	total := len(data) * 8
	var out []FooterEvent
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
			out = append(out, ev)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TimeMS < out[j].TimeMS })
	return out
}

// decodeTh10Block, à partir du début d'XUID, cherche le end-marker du bloc
// d'event puis décode le bloc de 60 octets le précédant. Renvoie ok=false si le
// bloc n'est pas un th=10. (Le xuid est rempli par l'appelant.)
func decodeTh10Block(data []byte, xstart, total int) (FooterEvent, bool) {
	win := xstart + 20000
	if win > total {
		win = total
	}
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

// footerTimeMS lit l'horloge du match (ms) aux octets 48 à 51 du bloc, gros-boutiste.
func footerTimeMS(data []byte, ebs int) int {
	return int(source.OctetAuBit(data, ebs+footerByteTime*8))<<24 |
		int(source.OctetAuBit(data, ebs+(footerByteTime+1)*8))<<16 |
		int(source.OctetAuBit(data, ebs+(footerByteTime+2)*8))<<8 |
		int(source.OctetAuBit(data, ebs+(footerByteTime+3)*8))
}

// ladderTiers = têtes des 6 records de l'échelle de score-contribution CTF (la
// valeur gauche double ; préfixes constants inter-match). Une FRAME re-transmettant
// les 6 tiers = un burst de capture. Adapté de ctfcap.
var ladderTiers = [][]byte{
	{0xa4, 0x00, 0x00, 0x00},
	{0x03, 0x48, 0x00, 0x00, 0x01},
	{0x06, 0x90, 0x00, 0x00, 0x02},
	{0x0d, 0x20, 0x00, 0x00, 0x05},
	{0x1a, 0x40, 0x00, 0x00, 0x0a},
	{0x34, 0x80, 0x00, 0x00, 0x15},
}

// captureMinTiers = nombre de tiers distincts requis pour qualifier un burst de
// capture CTF. tiers==6 = rock-solid (0 manque / 0 faux positif sur 4 matchs de
// ground-truth) ; tiers<6 sur-compte (RESEARCH_THEATER_RE.md §M-ter).
const captureMinTiers = 6

// countDistinctTiers renvoie le nombre de tiers distincts présents dans payload.
func countDistinctTiers(pay []byte) int {
	distinct := 0
	for _, t := range ladderTiers {
		if bytes.Contains(pay, t) {
			distinct++
		}
	}
	return distinct
}

// captureBurst = un burst de capture CTF détecté, horodaté en ms match.
type captureBurst struct {
	matchMS int
}

// scanCaptureBursts détecte les bursts de capture CTF dans les FRAME d'un chunk gameplay
// (type 2). startMS = start_ms du chunk (manifest) ; on convertit l'us de chaque FRAME en ms
// match via le premier FRAME comme ancre (la première frame du chunk = état complet, ignorée).
// Adapté de ctfcap detect.
func scanCaptureBursts(frames []source.Packet, startMS int) []captureBurst {
	if len(frames) == 0 {
		return nil
	}
	first := frames[0].TS
	var out []captureBurst
	for i, f := range frames {
		if i == 0 {
			continue // frame d'état complet de début de chunk
		}
		if countDistinctTiers(f.Payload) >= captureMinTiers {
			matchMS := startMS + int(int64(f.TS-first)/1000)
			out = append(out, captureBurst{matchMS: matchMS})
		}
	}
	return out
}
