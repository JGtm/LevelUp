// Package objectiveevents — décodage des timelines d'events objectif (CTF
// captures, Strongholds/KOTH zones, Oddball crâne) depuis les chunks film Halo,
// vers des []domain.ObjectiveEvent (mode-agnostique, cf.
// .ai/PLAN_WEAPON_ATTRIBUTION_V3.md §10).
//
// Algos PURS : zéro accès DB, zéro Streamlit, ET DEPUIS LE 2026-09-02 ZÉRO LECTURE DE FILM.
// Les neuf points d'entrée reçoivent un `*filmsource.Film` DÉJÀ CHARGÉ — chunks décompressés
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
//   - sélection des paquets FRAME parmi ceux que `filmsource` a découpés ;
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
// `walkFrames` et `decompressChunk` ont suivi le 2026-09-02, remplacées par `filmsource`.)
//
// Faits de décode VALIDÉS (RESEARCH_THEATER_RE.md §M, §M-ter) :
//   - footer = chunk de plus haut index, chunk_type 3 ; ses events th=10 = les
//     interactions objectif (t=horloge BE @octets 48-51, slot b36, team b37/b55,
//     xuid). Le champ team du film est NON FIABLE sur certains matchs -> l'équipe
//     canonique vient du roster (xuid->team_id de match_participants).
//   - capture CTF = une FRAME re-transmettant la table objectif complète
//     (tiers==6) ; team de la capture = l'event th=10 de t MAX dans le cluster
//     coïncident. objective_id (zone/colline) non récupérable -> NULL.
package objectiveevents

import (
	"bytes"
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/filmsource"
)

// packetFrame = le type de paquet consommé ici : FRAME, snapshot/delta d'état horodaté (us).
// Le découpage lui-même (en-tête 16 octets, terminateur CHUNK_END) appartient à `filmsource`.
const packetFrame = 0

// Types de chunk du MANIFESTE, tels que [filmsource.ChunkMeta.ChunkType] les porte.
//
// ZÉRO N'EST PAS UN TYPE : c'est ce que `filmsource.LoadDir` synthétise pour un `chunk_NN.bin`
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
// que prennent [filmsource.Film.Chunk] et [filmsource.Film.Packets]) et ses métadonnées.
type manifestChunk struct {
	pos  int
	meta filmsource.ChunkMeta
}

// manifestChunks rend les chunks du film décrits par le manifeste, dans l'ordre du film.
//
// POURQUOI CE FILTRE EXISTE, ET CE QU'IL PRÉSERVE. Avant le 2026-09-02, ce paquet recevait une
// `FilmSource` et itérait l'index du MANIFESTE : un fichier de chunk présent au cache mais
// absent du manifeste n'était jamais lu. Le film chargé par `filmsource.LoadDir`, lui, porte
// TOUS les fichiers présents. Sans ce filtre, ces chunks-là seraient balayés avec un `start_ms`
// de zéro — donc datés faux. Un film du cache est dans ce cas (`7b0d89c4`, chunks 31 et 32).
//
// Un film sans aucune métadonnée de manifeste rend une liste VIDE : c'est le seul résultat
// honnête (rien n'est datable), et [chunksDatables] le journalise plutôt que de le taire.
func manifestChunks(film *filmsource.Film) []manifestChunk {
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
func chunksDatables(ctx context.Context, film *filmsource.Film, matchID string) []manifestChunk {
	chunks := manifestChunks(film)
	if len(chunks) == 0 {
		slog.InfoContext(ctx, "objectiveevents: film sans chunk décrit par le manifeste — rien à dater",
			"match_id", matchID, "chunks_du_film", filmChunkCount(film))
	}
	return chunks
}

// filmChunkCount rend le nombre de chunks du film, ou 0 pour un film absent — de quoi dire au
// journal si « aucun chunk du manifeste » veut dire « pas de film » ou « pas de manifeste ».
func filmChunkCount(film *filmsource.Film) int {
	if film == nil {
		return 0
	}
	return film.NumChunks()
}

// framesOf rend les paquets FRAME (type 0) du chunk à la POSITION `pos` dans le film, dans
// l'ordre du chunk. C'est tout ce qui reste de l'ancien `walkFrames` : le découpage est fait
// une fois par `filmsource`, il ne reste qu'à choisir le type.
func framesOf(film *filmsource.Film, pos int) []filmsource.Packet {
	pks := film.Packets(pos)
	out := make([]filmsource.Packet, 0, len(pks))
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

// readBitsBE lit n bits big-endian (MSB-first) à partir de bitPos. Adapté de
// tmp_film_explore (identique dans filmx/ctfcap/t2score).
func readBitsBE(data []byte, bitPos, n int) uint64 {
	var r uint64
	for i := 0; i < n; i++ {
		bi := (bitPos + i) / 8
		if bi >= len(data) {
			return r
		}
		off := 7 - ((bitPos + i) % 8)
		bit := uint64((data[bi] >> uint(off)) & 1)
		r = (r << 1) | bit
	}
	return r
}

// readByteAtBit lit un octet à un offset BIT arbitraire (gère le non-alignement).
func readByteAtBit(data []byte, bit int) byte {
	if bit < 0 || bit+8 > len(data)*8 {
		return 0
	}
	bi := bit / 8
	off := uint(bit % 8)
	if off == 0 {
		return data[bi]
	}
	return data[bi]<<off | data[bi+1]>>(8-off)
}

// readU64LEAtBit lit un uint64 little-endian à un offset bit arbitraire.
func readU64LEAtBit(data []byte, bit int) uint64 {
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(readByteAtBit(data, bit+i*8)) << (uint(i) * 8)
	}
	return x
}

// th10Event = un event highlight de type_hint==10 (interaction objectif) décodé
// depuis le footer. t = horloge match (ms) ; slot = b36 (0-3, stable par xuid) ;
// teamRaw = b55 (NON fiable sur certains matchs -> à confirmer via roster) ; xuid
// = acteur. Structure validée RESEARCH_THEATER_RE.md §M.
type th10Event struct {
	t       int
	slot    int
	teamRaw int
	xuid    uint64
}

// scanTh10Events extrait tous les events th=10 d'un chunk décompressé (typiquement
// le footer chunk_type 3). Adapté de evDump/thTally (filmx) : on localise chaque
// XUID (préfixe 0x2d/0x25, suffixe 0xc0, valeur LE plausible), on cherche le
// end-marker [00 00 2e e0] de son bloc, on recule de 60 octets, on lit th@b47,
// t=BE@b48-51, slot@b36, team@b55. Filtré sur th==10 et dédupliqué par bloc.
func scanTh10Events(data []byte) []th10Event {
	total := len(data) * 8
	var out []th10Event
	seen := map[int]bool{}
	for ms := 8; ms <= total-8; ms++ {
		if readByteAtBit(data, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		if p := readByteAtBit(data, xe); p != 0x2d && p != 0x25 {
			continue
		}
		xstart := xe - 64
		if seen[xstart] {
			continue
		}
		x := readU64LEAtBit(data, xstart)
		if x <= minXUID || x >= maxXUID {
			continue
		}
		seen[xstart] = true
		if ev, ok := decodeTh10Block(data, xstart, total); ok {
			out = append(out, th10Event{ev.t, ev.slot, ev.teamRaw, x})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].t < out[j].t })
	return out
}

// decodeTh10Block, à partir du début d'XUID, cherche le end-marker du bloc
// d'event puis décode le bloc de 60 octets le précédant. Renvoie ok=false si le
// bloc n'est pas un th=10. (Le xuid est rempli par l'appelant.)
func decodeTh10Block(data []byte, xstart, total int) (th10Event, bool) {
	win := xstart + 20000
	if win > total {
		win = total
	}
	for b := xstart; b <= win-32; b++ {
		if readByteAtBit(data, b) == 0 && readByteAtBit(data, b+8) == 0 &&
			readByteAtBit(data, b+16) == 0x2e && readByteAtBit(data, b+24) == 0xe0 {
			ebs := b - 60*8
			if ebs < xstart {
				return th10Event{}, false
			}
			if int(readByteAtBit(data, ebs+47*8)) != 10 {
				return th10Event{}, false
			}
			t := int(readByteAtBit(data, ebs+48*8))<<24 | int(readByteAtBit(data, ebs+49*8))<<16 |
				int(readByteAtBit(data, ebs+50*8))<<8 | int(readByteAtBit(data, ebs+51*8))
			return th10Event{
				t:       t,
				slot:    int(readByteAtBit(data, ebs+36*8)),
				teamRaw: int(readByteAtBit(data, ebs+55*8)),
			}, true
		}
	}
	return th10Event{}, false
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
func scanCaptureBursts(frames []filmsource.Packet, startMS int) []captureBurst {
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
