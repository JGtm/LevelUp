// Package himap — geometry.go : des maillages du tag `rtgo` aux TRIANGLES en repere local.
//
// CHAINE (handoff `.ai/V7.5/cartes/HANDOFF_GEOMETRIE_TRIANGLES.md` §1) :
//
//	rtgo -> Sections (60 o par maillage) -> LOD render data (148 o) -> index de tampons
//	     -> descripteurs (80 o pour les sommets, 72 o pour les indices)
//	     -> octets dans le BLOB DE RESSOURCES -> dequantification par les bornes du maillage
//
// LA CORRECTION QUI COMPTE (handoff §2.1). Les sommets sont des `u16` BRUTS, jamais des
// `i16 + 32768`. Ecart aux bornes mesure : 5,8 mm avec la bonne lecture contre 84 mm avec
// la mauvaise. Le prototype Python porte encore la mauvaise formule (`geo2.py` ligne 143) —
// ce portage ne la reprend PAS, et tout resultat geometrique produit avant le 2026-07-26
// est entache d'une erreur mediane de 8,4 cm.
package himap

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Tailles d'enregistrement, etablies par la mesure (cf. rtgo.go pour le pas de 144).
const (
	sectionStride    = 60  // un maillage dans le bloc Sections
	lodStride        = 148 // un niveau de detail
	vertexDescStride = 80  // 0x50
	indexDescStride  = 72  // 0x48
)

// Offsets dans un enregistrement de LOD render data.
const (
	lodOffVertexBuffer = 0x64
	lodOffIndexBuffer  = 0x8A
)

// Offsets des bornes dans un enregistrement `Per Mesh Data` (144 o).
//
// Les bornes sont PAR MAILLAGE, pas uniques par tag. Le doute etait explicite (§3.2 du
// handoff : Reclaimer place un `BoundingBoxes` unique de 84 octets) et il est tranche par
// la mesure — c'est la lecture par maillage qui rend un ecart aux bornes sous le
// centimetre, cf. TestGeometryDequantification.
const (
	meshOffBoundsMin = 0x44
	meshOffBoundsMax = 0x50
)

// vertexUsagePosition / vertexStridePosition : le tampon des POSITIONS se reconnait a son
// usage et a son pas de 8 octets (u16 x4, la 4e composante nulle).
const (
	vertexUsagePosition  = 10
	vertexStridePosition = 8
)

// quantMax : les sommets sont quantifies sur 16 bits pleins.
const quantMax = 65535.0

// bufferDesc decrit un tampon dans le blob de ressources.
type bufferDesc struct {
	Usage  uint32
	Stride int
	Count  int
	Offset int
	Size   int
}

// Mesh est un maillage decode, en repere LOCAL a son instance.
type Mesh struct {
	Vertices  [][3]float64
	Triangles [][3]int
}

// RuntimeGeoAsset est un tag `rtgo` pret a rendre des maillages.
type RuntimeGeoAsset struct {
	info        tagInfo
	geo         RuntimeGeo
	blob        []byte
	vertexDescs []bufferDesc
	indexDescs  []bufferDesc
}

// VertexDescCount / IndexDescCount exposent la taille des tables (diagnostic et temoins).
func (a *RuntimeGeoAsset) VertexDescCount() int { return len(a.vertexDescs) }
func (a *RuntimeGeoAsset) IndexDescCount() int  { return len(a.indexDescs) }

// MeshCount rend le nombre de maillages du tag.
func (a *RuntimeGeoAsset) MeshCount() int { return a.geo.MeshCount }

// NewRuntimeGeoAsset assemble tout ce qu'il faut pour decoder les maillages d'un tag.
func NewRuntimeGeoAsset(tag, resourceBlob []byte) (*RuntimeGeoAsset, error) {
	for _, ti := range tagCandidates(tag) {
		geo, err := runtimeGeoFromTagInfo(ti)
		if err != nil {
			continue
		}
		a := &RuntimeGeoAsset{info: ti, geo: geo, blob: resourceBlob}
		a.vertexDescs = a.findDescTable(vertexDescStride, decodeVertexDesc)
		a.indexDescs = a.findDescTable(indexDescStride, decodeIndexDesc)
		return a, nil
	}
	return nil, fmt.Errorf("himap: tag rtgo illisible")
}

// findDescTable retient le data-block qui est un tableau COHERENT de descripteurs.
//
// La table n'est pas designee par un chemin de champs : on la reconnait a son invariant
// interne — `size == count * stride` sur TOUS ses enregistrements. C'est un test qui separe
// (un bloc quelconque ne le verifie pas), et a egalite on garde le plus grand.
func (a *RuntimeGeoAsset) findDescTable(stride int, dec func([]byte) (bufferDesc, bool)) []bufferDesc {
	var best []bufferDesc
	for i := 0; i < a.info.dataBlocks; i++ {
		abs, size := a.info.blockAbs(i)
		if size < stride || size%stride != 0 || abs < 0 || abs+size > len(a.info.tag) {
			continue
		}
		n := size / stride
		out := make([]bufferDesc, 0, n)
		ok := true
		for k := 0; k < n; k++ {
			d, valide := dec(a.info.tag[abs+k*stride : abs+(k+1)*stride])
			if !valide {
				ok = false
				break
			}
			out = append(out, d)
		}
		if ok && len(out) > len(best) {
			best = out
		}
	}
	return best
}

func decodeVertexDesc(r []byte) (bufferDesc, bool) {
	d := bufferDesc{
		Usage:  binary.LittleEndian.Uint32(r[4:]),
		Stride: int(binary.LittleEndian.Uint16(r[8:])),
		Count:  int(binary.LittleEndian.Uint32(r[0x0C:])),
		Offset: int(binary.LittleEndian.Uint32(r[0x10:])),
		Size:   int(binary.LittleEndian.Uint32(r[0x14:])),
	}
	return d, d.Stride > 0 && d.Count > 0 && d.Size == d.Count*d.Stride
}

func decodeIndexDesc(r []byte) (bufferDesc, bool) {
	d := bufferDesc{
		Stride: int(r[1]),
		Count:  int(binary.LittleEndian.Uint32(r[4:])),
		Offset: int(binary.LittleEndian.Uint32(r[8:])),
		Size:   int(binary.LittleEndian.Uint32(r[0x0C:])),
	}
	return d, (d.Stride == 2 || d.Stride == 4) && d.Count > 0 && d.Size == d.Count*d.Stride
}

// lodEntry : un niveau de detail d'un maillage.
type lodEntry struct{ vertexBuf, indexBuf int }

// lods rend les niveaux de detail d'un maillage.
func (a *RuntimeGeoAsset) lods(meshIndex int) []lodEntry {
	sections := a.geo.SectionsTarget
	if sections < 0 {
		return nil
	}
	cible := -1
	for i := 0; i < a.info.structs; i++ {
		b := a.info.structTab + i*structEntrySize
		if u16(a.info.tag, b+0x10) != 1 || i32(a.info.tag, b+0x18) != sections {
			continue
		}
		if u32(a.info.tag, b+0x1C) == meshIndex*sectionStride {
			cible = i32(a.info.tag, b+0x14)
			break
		}
	}
	if cible < 0 {
		return nil
	}
	abs, size := a.info.blockAbs(cible)
	if abs < 0 || size <= 0 || abs+size > len(a.info.tag) || size%lodStride != 0 {
		return nil
	}
	out := make([]lodEntry, 0, size/lodStride)
	for k := 0; k < size/lodStride; k++ {
		r := a.info.tag[abs+k*lodStride:]
		out = append(out, lodEntry{
			vertexBuf: int(binary.LittleEndian.Uint16(r[lodOffVertexBuffer:])),
			indexBuf:  int(binary.LittleEndian.Uint16(r[lodOffIndexBuffer:])),
		})
	}
	return out
}

// bounds rend les bornes de dequantification d'un maillage.
func (a *RuntimeGeoAsset) bounds(meshIndex int) (mn, mx [3]float64, ok bool) {
	off := a.geo.PerMeshAbs + meshIndex*perMeshStride
	if meshIndex < 0 || meshIndex >= a.geo.MeshCount || off+perMeshStride > len(a.info.tag) {
		return mn, mx, false
	}
	for i := 0; i < 3; i++ {
		mn[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(a.info.tag[off+meshOffBoundsMin+i*4:])))
		mx[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(a.info.tag[off+meshOffBoundsMax+i*4:])))
	}
	for i := 0; i < 3; i++ {
		if math.IsNaN(mn[i]) || math.IsNaN(mx[i]) || mx[i] < mn[i] {
			return mn, mx, false
		}
	}
	return mn, mx, true
}

// MaxTrianglesPerMesh : plafond de finesse. On prend le LOD le plus fin qui tient dessous,
// sinon le plus grossier — un fond de carte n'a pas besoin du LOD 0 d'un rocher.
const MaxTrianglesPerMesh = 40000

// MeshBuffers rend les descripteurs de tampons du LOD retenu pour un maillage.
//
// Expose pour le diagnostic et pour les temoins : sans lui, on ne peut pas comparer deux
// lectures de dequantification sur les MEMES octets, donc pas departager la bonne.
func (a *RuntimeGeoAsset) MeshBuffers(meshIndex int) (vertex, index bufferDesc, ok bool) {
	l, ok := a.chooseLOD(meshIndex)
	if !ok {
		return vertex, index, false
	}
	return a.vertexDescs[l.vertexBuf], a.indexDescs[l.indexBuf], true
}

// chooseLOD applique la regle de finesse : le LOD le plus fin sous le plafond de
// triangles, sinon le plus grossier.
func (a *RuntimeGeoAsset) chooseLOD(meshIndex int) (lodEntry, bool) {
	lods := a.lods(meshIndex)
	if len(lods) == 0 {
		return lodEntry{}, false
	}
	choix := -1
	for k, l := range lods {
		if l.indexBuf >= len(a.indexDescs) || l.vertexBuf >= len(a.vertexDescs) {
			continue
		}
		if a.indexDescs[l.indexBuf].Count/3 <= MaxTrianglesPerMesh {
			choix = k
			break
		}
	}
	if choix < 0 {
		choix = len(lods) - 1
	}
	l := lods[choix]
	if l.vertexBuf >= len(a.vertexDescs) || l.indexBuf >= len(a.indexDescs) {
		return lodEntry{}, false
	}
	return l, true
}

// Blob rend le blob de ressources (lecture seule, pour les temoins).
func (a *RuntimeGeoAsset) Blob() []byte { return a.blob }

// Bounds rend les bornes de dequantification d'un maillage.
func (a *RuntimeGeoAsset) Bounds(meshIndex int) (mn, mx [3]float64, ok bool) {
	return a.bounds(meshIndex)
}

// Mesh decode un maillage en repere local. Renvoie nil si le maillage n'est pas exploitable
// (aucun LOD, tampon de positions absent, descripteur hors du blob) — c'est un cas normal,
// pas une erreur : le compte des maillages rendus est publie par l'appelant.
func (a *RuntimeGeoAsset) Mesh(meshIndex int) *Mesh {
	vd, id, ok := a.MeshBuffers(meshIndex)
	if !ok {
		return nil
	}
	if vd.Usage != vertexUsagePosition || vd.Stride != vertexStridePosition {
		return nil
	}
	if vd.Offset < 0 || vd.Offset+vd.Size > len(a.blob) || id.Offset < 0 || id.Offset+id.Size > len(a.blob) {
		return nil
	}
	mn, mx, okB := a.bounds(meshIndex)
	if !okB {
		return nil
	}

	verts := make([][3]float64, vd.Count)
	for i := 0; i < vd.Count; i++ {
		r := a.blob[vd.Offset+i*vertexStridePosition:]
		for ax := 0; ax < 3; ax++ {
			// u16 BRUT — cf. l'en-tete du fichier. Pas de `+ 32768`.
			q := float64(binary.LittleEndian.Uint16(r[ax*2:]))
			verts[i][ax] = mn[ax] + q/quantMax*(mx[ax]-mn[ax])
		}
	}

	return &Mesh{Vertices: verts, Triangles: a.triangles(id, len(verts))}
}

// triangles lit le tampon d'indices et rend les faces dont les trois sommets existent.
//
// Une face dont un indice depasse le tampon de sommets est ECARTEE, pas corrigee : elle
// signale un appariement LOD/tampon faux, et la rendre inventerait de la geometrie.
func (a *RuntimeGeoAsset) triangles(id bufferDesc, nVerts int) [][3]int {
	out := make([][3]int, 0, id.Count/3)
	for i := 0; i+2 < id.Count; i += 3 {
		var t [3]int
		hors := false
		for k := 0; k < 3; k++ {
			var v int
			if id.Stride == 2 {
				v = int(binary.LittleEndian.Uint16(a.blob[id.Offset+(i+k)*2:]))
			} else {
				v = int(binary.LittleEndian.Uint32(a.blob[id.Offset+(i+k)*4:]))
			}
			if v >= nVerts {
				hors = true
				break
			}
			t[k] = v
		}
		if !hors {
			out = append(out, t)
		}
	}
	return out
}
