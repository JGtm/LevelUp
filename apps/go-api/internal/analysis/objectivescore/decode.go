// Package objectivescore — décodage de la TIMELINE de score per-équipe des modes
// à objectif (Strongholds, KOTH) depuis les chunks film Halo, vers des
// []ScoreFrame (un point de score par chunk type-2, ~20s de granularité).
//
// Algos PURS : zéro accès DB, zéro Streamlit. L'IO (lecture/décompression des
// chunks) est fait par l'appelant qui fournit des ChunkInput déjà décompressés.
// Les finals exacts (calibration Strongholds) sont passés en argument par
// l'appelant (depuis match_registry.team_0_score / team_1_score).
//
// SUPERSÉDÉ (2026-08-01) — la « RE plus poussée » attendue ci-dessous a abouti :
// objectiveevents.ScoreCurve décode le VRAI score de mode, per-équipe ET per-joueur,
// à la milliseconde (une émission par changement, au lieu d'un point par chunk de 20 s),
// SANS calibration sur les finals. Vérifié position de bit par position de bit contre une
// capture Cheat Engine (283/284 et 6/6) et contre un relevé terrain écrit avant décodage.
// Ne rien construire de neuf sur ce paquet-ci ; sa condition de conservation a expiré et
// sa suppression est proposée dans .ai/HANDOFF_MODE_SCORE_CHAINE_2026-08-01.md §6.
//
// ANCRE BIT-LEVEL (PROUVÉE, cmd/tmp_shbits + cmd/tmp_scoreval, 2026-06-02) :
// le score continu vit dans le payload du paquet TYPE_2, ancré sur le token
// 12-bit 0x7B6 (MSB-first) cherché AU NIVEAU BIT (n'importe quelle phase) dans
// la fenêtre byte ~[835,912). Le token est stable bit-à-bit sur tous les chunks
// d'un même match (drift=0 observé).
//
// ATTENTION — STATUT RÉEL (cross-validation 38 Strongholds + 11 KOTH, 2026-06-03).
// CODE EXPÉRIMENTAL, NON CÂBLÉ EN PROD, TABLE NON BACKFILLÉE (conservé à la demande
// user, en attente d'une RE plus poussée). La cross-validation multi-matchs a
// INVALIDÉ la courbe de score per-équipe Strongholds :
//   - STRONGHOLDS : NON VALIDÉ. team1(+23) décode des MARQUEURS STRUCTURELS de fin
//     de film (les valeurs 50/32 sont IDENTIQUES sur des matchs aux finals très
//     différents : 193-112 vs 78-178), PAS un score. team0(+24) CAPPE à ~50 (efface
//     la magnitude : 169/193/178 → tous 50) et track la team LOCALE-FILM, pas DB
//     team_0 (suit le perdant sur matchs déséquilibrés). La "courbe" produite ici
//     n'est donc PAS un vrai score per-équipe : elle ne "matche le final" QUE parce
//     qu'on la calibre dessus (final ajusté par construction, shape bidon). PARKÉ
//     pending RE (assignation film-team→DB-team + cap=50). NE PAS afficher comme score.
//   - KOTH variante B (Ranked, finals bas ≤ ~5) : team0 = t2[ab+12], team1 =
//     t2[ab+16], score = meter/5 (ab = byte du token). SEUL cas VALIDÉ EXACT
//     (0a247154 Ranked 4-2 → 4-2 parfait, monotone).
//   - KOTH variante A (points hauts) / KOTH:Arena : NON fiable (les meters type-2 =
//     compteurs d'occupation de colline, PAS le score registry). Non validé.
//   - ODDBALL EXCLU : accumulateur TYPE_2 GLOBAL (1 crâne, pas per-équipe).
//
// Confidence : 'approx' systématique (granularité ~20s du snapshot type-2, +
// calibration par final). Pas de prétention ms-exact.
package objectivescore

import (
	"encoding/binary"
)

// Constantes du conteneur film + ancre.
const (
	packetType2  = 2     // TYPE_2 : snapshot game-state par chunk (score)
	packetEnd    = 7     // CHUNK_END
	packetHdr    = 16    // taille de l'en-tête de paquet (type u16 LE@0, size u32 LE@4)
	scoreToken   = 0x7B6 // token 12-bit ancre du bloc score (MSB-first)
	scoreTokenW  = 12    // largeur du token en bits
	tokenWinLo   = 835   // borne basse (byte) de la fenêtre de recherche du token
	tokenWinHi   = 912   // borne haute (byte) de la fenêtre de recherche du token
	shOffTeam0   = 24    // Strongholds : team0 = varAtBit(token + 24 bits)
	shOffTeam1   = 23    // Strongholds : team1 = varAtBit(token + 23 bits) [fragile]
	kothBOffT0   = 12    // KOTH var B : team0 = byte(token_byte + 12)
	kothBOffT1   = 16    // KOTH var B : team1 = byte(token_byte + 16)
	kothBDivisor = 5     // KOTH var B : score = meter / 5
	kothAOffTot  = 14    // KOTH var A : total = byte(token_byte + 14)
	kothAOffT1   = 13    // KOTH var A : team1 = byte(token_byte + 13) >> 4
)

// ChunkInput = un chunk film DÉJÀ décompressé, avec ses métadonnées manifest.
// L'appelant (CLI / test) fait la décompression zlib et fournit Data brut.
type ChunkInput struct {
	Data      []byte // payload du chunk décompressé (conteneur de paquets 16o)
	StartMS   int    // start_ms du chunk (manifest) → TimeMS du ScoreFrame
	ChunkType int    // chunk_type du manifest (on ne décode que ==2)
}

// ScoreFrame = un point de la timeline de score per-équipe (1 par chunk type-2
// ancré). TimeMS = start_ms du chunk. Team0/Team1 = score per-équipe (calibré).
type ScoreFrame struct {
	TimeMS     int
	Team0      int
	Team1      int
	Source     string // ex. "film_strongholds_t2", "film_koth_t2_b"
	Confidence string // "approx" (granularité ~20s + calibration par final)
}

// --- Primitives bit-level (portées de cmd/tmp_shbits, PROUVÉES) ---

// bitAt lit le bit d'index i (MSB-first) ; 0 hors-bornes.
func bitAt(p []byte, i int) int {
	if i < 0 || i>>3 >= len(p) {
		return 0
	}
	return int((p[i>>3] >> uint(7-(i&7))) & 1)
}

// readBits lit n bits MSB-first à partir de start.
func readBits(p []byte, start, n int) int {
	v := 0
	for i := 0; i < n; i++ {
		v = (v << 1) | bitAt(p, start+i)
	}
	return v
}

// tokenBit cherche le token scoreToken (0x7B6, 12 bits MSB-first) dans la fenêtre
// byte [loByte,hiByte) et renvoie la 1ère position de BIT, ou -1.
func tokenBit(p []byte, loByte, hiByte int) int {
	lo, hi := loByte*8, hiByte*8
	if hi > len(p)*8-scoreTokenW {
		hi = len(p)*8 - scoreTokenW
	}
	for bp := lo; bp < hi; bp++ {
		if readBits(p, bp, scoreTokenW) == scoreToken {
			return bp
		}
	}
	return -1
}

// varAtBit décode le varint à continuation à la position de bit o : lit un octet
// MSB-first ; continuation si bit 0x40 (alors ((b&0x3f)<<8)|octet_suivant).
func varAtBit(p []byte, o int) int {
	b := readBits(p, o, 8)
	if b&0x40 == 0 {
		return b
	}
	return ((b & 0x3f) << 8) | readBits(p, o+8, 8)
}

// type2Payload extrait le payload du 1er paquet TYPE_2 d'un chunk décompressé,
// ou nil. (Header 16o : type u16 LE@0, size u32 LE@4.)
func type2Payload(d []byte) []byte {
	off := 0
	for off+packetHdr <= len(d) {
		typ := int(binary.LittleEndian.Uint16(d[off:]))
		size := int(binary.LittleEndian.Uint32(d[off+4:]))
		if size < 0 || off+packetHdr+size > len(d) {
			break
		}
		if typ == packetType2 {
			return d[off+packetHdr : off+packetHdr+size]
		}
		off += packetHdr + size
		if typ == packetEnd {
			break
		}
	}
	return nil
}

// anchoredPayload renvoie (payload TYPE_2, position bit du token) pour un chunk
// type-2 ancré, ou (nil,-1) si le chunk n'est pas type-2, sans TYPE_2, ou sans
// ancre.
func anchoredPayload(c ChunkInput) ([]byte, int) {
	if c.ChunkType != packetType2 {
		return nil, -1
	}
	p := type2Payload(c.Data)
	if p == nil {
		return nil, -1
	}
	tb := tokenBit(p, tokenWinLo, tokenWinHi)
	if tb < 0 {
		return nil, -1
	}
	return p, tb
}
