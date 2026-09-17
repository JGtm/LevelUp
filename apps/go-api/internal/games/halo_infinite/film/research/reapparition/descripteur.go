//go:build research

package reapparition

// descripteur.go — LES QUATRE PAS DE LA NOTE DE METHODE, EN OCTETS.
//
// `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` § 1 : chaine -> accesseur de nom -> slot
// `descripteur + 0x18` -> ecrivain `descripteur + 0x40`. Chacun des trois sauts est une
// RECHERCHE D'UNIQUE : si le balayage rend zero ou plusieurs candidats, la chaine s'arrete et le
// dit. Un composant instancie N fois (`ti=11` seize `sub-objective-entities`) rassemble ses noms
// dans une TABLE de pointeurs et n'a pas d'accesseur : le rapport le signale au lieu de choisir.
//
// LA SIGNATURE DE FAMILLE N'EST PAS RECOPIEE DE LA NOTE, ELLE EST MESUREE. La note en donne une
// version, et sa propre section 17 la corrige sur deux slots. Cet instrument la DEDUIT des
// composants de calibration — ceux dont l'adresse d'ecrivain est deja connue du depot — et
// n'appelle « meme famille » qu'un descripteur dont les slots constants coincident avec elle.

import (
	"encoding/binary"
	"fmt"
)

// Les offsets de la note, dans le descripteur.
const (
	OffsetNom         = 0x18 // le slot qui pointe l'accesseur de nom
	OffsetEcrivain    = 0x40 // le deserialiseur / lecteur de bits du composant
	tailleDescripteur = 0x50 // 10 slots
)

// Descripteur est le resultat de la chaine pour UN nom de composant.
type Descripteur struct {
	// Nom est le nom exact du composant, tel qu'il est ecrit dans le registre du film.
	Nom string
	// ChaineVA est l'adresse de la chaine de caracteres en `.rdata`.
	ChaineVA uint64
	// ThunkVA est l'accesseur de nom (`LEA reg,[rip+chaine] ; RET`).
	ThunkVA uint64
	// SlotVA est le slot de table qui pointe le thunk, DescripteurVA son debut (slot - 0x18).
	SlotVA, DescripteurVA uint64
	// EcrivainVA est le contenu de `descripteur + 0x40`.
	EcrivainVA uint64
	// Slots porte les dix mots du descripteur, dans l'ordre — la matiere de la signature.
	Slots [tailleDescripteur / 8]uint64
	// Bornes sont les bornes exactes de l'ecrivain, lues dans `.pdata`.
	Bornes FuncRange
	// BornesConnues dit si `.pdata` a rendu ces bornes.
	BornesConnues bool
	// Echec porte la raison de l'arret quand la chaine n'aboutit pas ; vide sinon.
	Echec string
	// Multiples porte les candidats quand un pas en rend plusieurs (diagnostic, jamais un choix).
	Multiples []string
}

// Aboutie dit si la chaine est allee jusqu'a l'ecrivain.
func (d *Descripteur) Aboutie() bool { return d.Echec == "" && d.EcrivainVA != 0 }

// Resoudre execute les quatre pas pour un nom de composant.
func (e *Executable) Resoudre(nom string) *Descripteur {
	d := &Descripteur{Nom: nom}

	chaines := e.chercherChaine(nom)
	if len(chaines) == 0 {
		d.Echec = "chaine absente de l'image"
		return d
	}
	if len(chaines) > 1 {
		for _, va := range chaines {
			d.Multiples = append(d.Multiples, fmt.Sprintf("chaine %#x", va))
		}
		d.Echec = fmt.Sprintf("%d chaines pour ce nom", len(chaines))
		return d
	}
	d.ChaineVA = chaines[0]

	thunks := e.chercherThunk(d.ChaineVA)
	if len(thunks) != 1 {
		for _, va := range thunks {
			d.Multiples = append(d.Multiples, fmt.Sprintf("thunk %#x", va))
		}
		d.Echec = fmt.Sprintf("%d accesseurs de nom (attendu 1) — nom en TABLE de pointeurs ?", len(thunks))
		return d
	}
	d.ThunkVA = thunks[0]

	slots := e.chercherMot(d.ThunkVA, false)
	if len(slots) != 1 {
		for _, va := range slots {
			d.Multiples = append(d.Multiples, fmt.Sprintf("slot %#x", va))
		}
		d.Echec = fmt.Sprintf("%d slots pointant l'accesseur (attendu 1)", len(slots))
		return d
	}
	d.SlotVA = slots[0]
	d.DescripteurVA = d.SlotVA - OffsetNom

	for i := range d.Slots {
		v, ok := e.U64(d.DescripteurVA + uint64(i*8))
		if !ok {
			d.Echec = fmt.Sprintf("descripteur %#x hors section au slot %d", d.DescripteurVA, i)
			return d
		}
		d.Slots[i] = v
	}
	d.EcrivainVA = d.Slots[OffsetEcrivain/8]
	if d.EcrivainVA == 0 {
		d.Echec = "ecrivain nul en +0x40"
		return d
	}
	d.Bornes, d.BornesConnues = e.Fonction(d.EcrivainVA)
	return d
}

// chercherChaine rend les VA des occurrences du nom en chaine C isolee (precedee et suivie d'un
// octet nul), dans les sections de DONNEES.
func (e *Executable) chercherChaine(nom string) []uint64 {
	motif := append([]byte(nom), 0)
	var out []uint64
	for i := range e.sections {
		s := &e.sections[i]
		if s.execCode {
			continue
		}
		for off := 0; ; {
			k := indexOf(s.contenu[off:], motif)
			if k < 0 {
				break
			}
			abs := off + k
			if abs == 0 || s.contenu[abs-1] == 0 {
				out = append(out, s.va+uint64(abs))
			}
			off = abs + 1
		}
	}
	return out
}

// chercherThunk rend les VA des accesseurs de nom pointant la chaine : `REX.W LEA reg,[rip+d]`
// suivi d'un `RET`, soit huit octets exactement.
func (e *Executable) chercherThunk(chaineVA uint64) []uint64 {
	const tailleThunk = 8
	var out []uint64
	for i := range e.sections {
		s := &e.sections[i]
		if !s.execCode {
			continue
		}
		c := s.contenu
		for off := 0; off+tailleThunk <= len(c); off++ {
			if c[off] != 0x48 || c[off+1] != 0x8d || c[off+7] != 0xc3 {
				continue
			}
			if modrm := c[off+2]; modrm&0xc7 != 0x05 { // mod=00, rm=101 : rip-relatif
				continue
			}
			disp := int32(binary.LittleEndian.Uint32(c[off+3:]))
			suivante := s.va + uint64(off) + 7
			if uint64(int64(suivante)+int64(disp)) == chaineVA {
				out = append(out, s.va+uint64(off))
			}
		}
	}
	return out
}

// chercherMot rend les VA des mots de 64 bits alignes valant la cible. `code` dit s'il faut
// balayer les sections executables (faux : seulement les donnees).
func (e *Executable) chercherMot(cible uint64, code bool) []uint64 {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], cible)
	var out []uint64
	for i := range e.sections {
		s := &e.sections[i]
		if s.execCode != code {
			continue
		}
		for off := 0; off+8 <= len(s.contenu); off += 8 {
			if s.contenu[off] == buf[0] && s.contenu[off+7] == buf[7] &&
				binary.LittleEndian.Uint64(s.contenu[off:]) == cible {
				out = append(out, s.va+uint64(off))
			}
		}
	}
	return out
}

// indexOf est un `bytes.Index` local — garde l'import a zero dependance de plus.
func indexOf(hay, needle []byte) int {
	n := len(needle)
	if n == 0 || len(hay) < n {
		return -1
	}
	for i := 0; i+n <= len(hay); i++ {
		if hay[i] != needle[0] {
			continue
		}
		ok := true
		for j := 1; j < n; j++ {
			if hay[i+j] != needle[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}
