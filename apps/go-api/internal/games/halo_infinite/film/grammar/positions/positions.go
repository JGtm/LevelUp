// Package positions décode les positions joueurs KEYFRAME des films Halo
// Infinite (§N de .ai/RESEARCH_THEATER_RE.md). Pur, sans accès DB.
//
// Modèle de décodage (prouvé sur 000d5950, cf. cmd/tmp_posdecode) :
//   - Les positions full-state vivent dans le payload TYPE_2 de chaque chunk.
//   - Chaque record joueur est délimité par un « comb » : motif binaire 1⁸0¹⁶
//     répété 4× (96 bits), recherché au niveau BIT.
//   - Position = triplet float32-LITTLE-ENDIAN (x, y, z) à combStartBit-273
//     (x@-273, y@-273+32, z@-273+64).
//   - Les records delta (compression) omettent la position absolue : on les
//     filtre (|x|+|y|+|z| < 1). On filtre aussi 2 artefacts structurels
//     récurrents non-joueur : (0,2,0) et (-2.1,0,0).
//
// LIMITE (honnête, §N) : l'attribution per-joueur (xuid) reste BLOQUÉE
// (delta-compression, index non fixe). v1 = MATCH-LEVEL : toutes les positions
// full-state, sans attribution xuid. Un team-split best-effort est tenté quand
// un clustering spatial net en 2 groupes émerge ; sinon Team = -1.
// DEPLACE LE 2026-09-16 (lot 2.5.e, decision V15 (4)) d `internal/analysis/positions` : ce
// paquet LIT des bits de film — il portait sa propre copie de `bitAt` et son propre marcheur de
// paquets de seize octets (§4 D1 (2.4) du plan) — donc il descend dans la couche `grammar`. Ses
// lectures d octets passent desormais par la couche `source` : [source.BitAt] pour le bit,
// [source.U16LE] et [source.U32LE] pour l en-tete de bloc. Son marcheur, lui, n est PAS fondu
// dans [source.Paquets] : les deux grammaires d arret different (terminateur emis, taille nulle)
// et les opposer est un changement de CONTENU, hors d un lot de descente — consigne au §4 du
// plan. Le TYPE qu il rend,
// `PlayerPosition`, est remonte en `domain/playerposition` le meme jour : ses lecteurs (ports,
// service, DuckDB, corps HTTP) n ont rien a savoir d un titre.
package positions

import (
	"math"

	"levelup/go-api/internal/domain/playerposition"
	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// combOffsetBits est le recul en bits depuis le début du comb jusqu'au premier
// octet du triplet float32 de position (x@-273, y@-241, z@-209).
const combOffsetBits = 273

// deltaMagnitudeThreshold : un record dont |x|+|y|+|z| est sous ce seuil est un
// record delta (position absolue omise par la compression) — à ignorer.
const deltaMagnitudeThreshold = 1.0

// chunkTypeSnapshot est le type de chunk (TYPE_2) portant l'état de jeu
// full-state où vivent les positions.
const chunkTypeSnapshot = 2

// ChunkInput est un chunk film déjà décompressé fourni au décodeur.
//
// Data doit être le contenu DÉCOMPRESSÉ du chunk (le décodeur en extrait le
// payload TYPE_2). StartMS donne l'horodatage (résolution keyframe ~20s).
type ChunkInput struct {
	Data      []byte
	StartMS   int
	ChunkType int
}

// DecodeKeyframePositions décode toutes les positions joueurs full-state des
// chunks TYPE_2 fournis. Les positions sont match-level (pas d'attribution
// xuid) ; un team-split best-effort est tenté par chunk.
func DecodeKeyframePositions(chunks []ChunkInput) []playerposition.PlayerPosition {
	var out []playerposition.PlayerPosition
	for _, c := range chunks {
		if c.ChunkType != chunkTypeSnapshot {
			continue
		}
		payload := type2Payload(c.Data)
		if payload == nil {
			continue
		}
		raw := decodeChunkPositions(payload, c.StartMS)
		assignTeamsBestEffort(raw)
		out = append(out, raw...)
	}
	return out
}

// decodeChunkPositions extrait les positions full-state d'un payload TYPE_2.
// Toutes les positions retournées ont Team = TeamUnknown (assignation déférée).
func decodeChunkPositions(payload []byte, startMS int) []playerposition.PlayerPosition {
	combs := findCombs(payload)
	out := make([]playerposition.PlayerPosition, 0, len(combs))
	for _, cb := range combs {
		o := cb - combOffsetBits
		x := readFloat32LE(payload, o)
		y := readFloat32LE(payload, o+32)
		z := readFloat32LE(payload, o+64)
		if !(plausible(x) && plausible(y) && plausible(z)) {
			continue
		}
		mag := math.Abs(float64(x)) + math.Abs(float64(y)) + math.Abs(float64(z))
		if mag < deltaMagnitudeThreshold {
			continue // record delta / quasi-nul
		}
		if isStructuralArtifact(x, y, z) {
			continue // (0,2,0) ou (-2.1,0,0) — entité non-joueur récurrente
		}
		out = append(out, playerposition.PlayerPosition{
			TimeMS: startMS, X: x, Y: y, Z: z, Team: playerposition.TeamUnknown,
		})
	}
	return out
}

// isStructuralArtifact filtre les 2 records constants non-joueur récurrents
// (§N) : (0,2,0) et (-2.1,0,0), à tolérance ±0.1.
func isStructuralArtifact(x, y, z float32) bool {
	near := func(v, target float32) bool {
		return math.Abs(float64(v-target)) <= 0.1
	}
	if near(x, 0) && near(y, 2) && near(z, 0) {
		return true
	}
	if near(x, -2.1) && near(y, 0) && near(z, 0) {
		return true
	}
	return false
}

// type2Payload parcourt les blocs (header 16o : u16 type @0, u32 size @4) du
// chunk décompressé et renvoie le payload du premier bloc TYPE_2, ou nil.
func type2Payload(d []byte) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := int(source.U16LE(d, off))
		size := int(source.U32LE(d, off+4))
		if size < 0 || off+16+size > len(d) {
			break
		}
		if typ == chunkTypeSnapshot {
			return d[off+16 : off+16+size]
		}
		off += 16 + size
		if typ == 7 { // REPLICATION_DATA_END
			break
		}
	}
	return nil
}

// findCombs renvoie les positions BIT de tous les combs (motif 1⁸0¹⁶ ×4) du
// payload, sans chevauchement.
func findCombs(p []byte) []int {
	var combs []int
	maxBit := len(p) * 8
	for bp := 0; bp+96 <= maxBit; bp++ {
		if combAt(p, bp) {
			combs = append(combs, bp)
			bp += 95 // éviter le chevauchement
		}
	}
	return combs
}

// combAt teste le motif comb (1⁸0¹⁶ répété ×4 = 96 bits) à la position bit bp.
func combAt(p []byte, bp int) bool {
	if bp < 0 || (bp+96)>>3 >= len(p) {
		return false
	}
	for rep := 0; rep < 4; rep++ {
		base := bp + rep*24
		for i := 0; i < 8; i++ {
			if source.BitAt(p, base+i) != 1 {
				return false
			}
		}
		for i := 8; i < 24; i++ {
			if source.BitAt(p, base+i) != 0 {
				return false
			}
		}
	}
	return true
}

// readFloat32LE lit 32 bits MSB-first à l'offset bit o, byteswap, puis
// interprète en float32 (little-endian).
func readFloat32LE(p []byte, o int) float32 {
	var v uint32
	for i := 0; i < 32; i++ {
		v = (v << 1) | uint32(source.BitAt(p, o+i))
	}
	sw := (v&0xff)<<24 | (v&0xff00)<<8 | (v&0xff0000)>>8 | (v&0xff000000)>>24
	return math.Float32frombits(sw)
}

// plausible rejette les NaN/Inf et les magnitudes hors d'une carte Halo.
func plausible(x float32) bool {
	a := float64(x)
	return !math.IsNaN(a) && !math.IsInf(a, 0) && a > -200 && a < 200
}
