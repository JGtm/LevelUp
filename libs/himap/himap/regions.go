// Package himap — regions.go : le marcheur regions -> permutations -> sections d'un `mode`.
//
// POURQUOI. Les variantes d'armement d'un vehicule (Rockethog, Gauss, chaingun) ne sont pas des
// modeles separes — mesure V4 : le `weap` de variante n'a AUCUN render_model, la chaine
// vcdd/uwfa/sofa non plus. Ce sont des PERMUTATIONS d'une region du render_model PARTAGE du
// chassis. Le rendu par defaut les superpose toutes ; ce marcheur isole, region par region, les
// permutations et leurs plages de sections, pour rendre chaque variante seule.
//
// LAYOUT (mesure sur le mode Warthog 0x561f2ca7) : bloc regions au champ racine +40, 24 o par
// region = { Name StringId (+0), Permutations TagBlock (+4, count a +20) }. Chaque bloc de
// permutations est resolu via la struct-table (field_block == bloc regions, field_offset ==
// index_region*24 + 4), meme mecanique que `lods()` pour les sections. Meme patron que le
// walker sbsp/rtgo.
package himap

import "fmt"

const (
	modeOffRegions         = 40 // champ racine du bloc regions dans un `mode`
	regionStride           = 24 // un enregistrement region
	regionPermsFieldOffset = 4  // le champ TagBlock `permutations` dans un enregistrement region
	tagBlockCountOffset    = 16 // le compteur d'un TagBlock inline (u32) a +16 (apres 16 o de pointeur)
)

// Permutation : une variante d'une region, avec sa plage de sections dans le bloc Sections.
type Permutation struct {
	Name         uint32 // StringId
	SectionIndex int
	SectionCount int
}

// Region : une region du render_model (ex. corps, tourelle), avec ses permutations.
type Region struct {
	Name         uint32 // StringId
	Permutations []Permutation
}

// ModeRegions marche les regions d'un tag `mode` deja decompresse.
func ModeRegions(tag []byte) ([]Region, error) {
	for _, ti := range tagCandidates(tag) {
		offs, targets, err := ti.rootBlockRefs()
		if err != nil {
			continue
		}
		regionsTarget := -1
		for i, o := range offs {
			if o == modeOffRegions {
				regionsTarget = targets[i]
				break
			}
		}
		if regionsTarget < 0 {
			continue
		}
		rAbs, rSize := ti.blockAbs(regionsTarget)
		if rAbs < 0 || rSize <= 0 || rSize%regionStride != 0 || rAbs+rSize > len(ti.tag) {
			continue
		}
		return ti.parseRegions(regionsTarget, rAbs, rSize/regionStride), nil
	}
	return nil, fmt.Errorf("himap: bloc regions du mode introuvable")
}

func (ti tagInfo) parseRegions(regionsTarget, rAbs, n int) []Region {
	out := make([]Region, 0, n)
	for r := 0; r < n; r++ {
		rec := rAbs + r*regionStride
		reg := Region{Name: uint32(u32(ti.tag, rec))}
		permCount := u32(ti.tag, rec+regionPermsFieldOffset+tagBlockCountOffset)
		if pt := ti.subBlockTarget(regionsTarget, r*regionStride+regionPermsFieldOffset); pt >= 0 {
			pAbs, pSize := ti.blockAbs(pt)
			reg.Permutations = ti.parsePermutations(pAbs, pSize, permCount)
		}
		out = append(out, reg)
	}
	return out
}

// subBlockTarget resout un TagBlock imbrique : le data-block designe par le champ (fieldOffset)
// d'un enregistrement du bloc parent (fieldBlock). Meme lecture de struct-table que `lods()`.
func (ti tagInfo) subBlockTarget(fieldBlock, fieldOffset int) int {
	for i := 0; i < ti.structs; i++ {
		b := ti.structTab + i*structEntrySize
		if u16(ti.tag, b+0x10) != 1 || i32(ti.tag, b+0x18) != fieldBlock {
			continue
		}
		if u32(ti.tag, b+0x1C) == fieldOffset {
			return i32(ti.tag, b+0x14)
		}
	}
	return -1
}

// parsePermutations lit les enregistrements de permutation. Le stride se deduit de la taille du
// bloc et du compteur ; SectionIndex/SectionCount sont lus aux offsets +4/+6, VALIDES par le
// walker (plages coherentes couvrant les 90 sections du mode Warthog, SectionIndex -1 = herite).
func (ti tagInfo) parsePermutations(abs, size, count int) []Permutation {
	if abs < 0 || count <= 0 || size <= 0 || abs+size > len(ti.tag) || size%count != 0 {
		return nil
	}
	stride := size / count
	out := make([]Permutation, 0, count)
	for p := 0; p < count; p++ {
		rec := abs + p*stride
		perm := Permutation{Name: uint32(u32(ti.tag, rec))}
		if stride >= permOffSectionCount+2 {
			perm.SectionIndex = int(int16(u16(ti.tag, rec+permOffSectionIndex)))
			perm.SectionCount = int(u16(ti.tag, rec+permOffSectionCount))
		}
		out = append(out, perm)
	}
	return out
}

// Offsets Section index / count dans un enregistrement de permutation (12 o : Name StringId +0,
// SectionIndex u16 +4, SectionCount u16 +6) — VERIFIES par le walker sur le mode Warthog.
const (
	permOffSectionIndex = 4
	permOffSectionCount = 6
)
