package hinavmesh

// tagfile.go — LE FICHIER-TAG HAVOK, LU PAR SA PROPRE REFLEXION.
//
// Un fichier-tag porte la table de ses types : noms de types, noms de champs, taille de
// chaque structure et offset de chaque membre. On ne code donc EN DUR aucune disposition
// de structure, et ce n'est pas du zele : la `hkaiNavMesh::Face` de Havok 2022.1.0 porte
// des indices int32 (startEdgeIndex +0, numEdges +8) la ou l'en-tete public de Project
// Anarchy (Havok 2013) declare des int16 — pour la MEME taille de 12 octets. Un decodeur
// qui recopierait l'en-tete public lirait des indices plausibles et FAUX.
//
// ARBORESCENCE DES SECTIONS. En-tete de section = [u32 GROS-BOUTISTE][4 octets ASCII] ;
// les 2 bits de poids fort du u32 sont des drapeaux, les 30 restants la taille TOTALE de
// la section, en-tete compris. Le bit 0x40000000 marque une feuille (pas de sous-section).
//
//	TAG0
//	  SDKV = "20220100"          version du SDK Havok
//	  DATA                       les octets des objets
//	  TYPE
//	    TST1  chaines des noms de types      TNA1  table des types
//	    FST1  chaines des noms de champs     TBDY  corps des types
//	    TPTR / THSH / TPAD
//	  INDX
//	    ITEM  {u32 type et drapeaux | u32 offset dans DATA | u32 compte}
//	    PTCH  {u32 type | u32 n | n x u32 offset} — les offsets a rapiecer dans DATA
//
// LES POINTEURS SONT DES INDICES D'ITEM. Un `hkArray` serialise ne porte PAS son compte
// dans DATA (ses champs taille et capacite y sont a zero) : le u64 a l'offset du membre
// est l'indice de l'entree ITEM qui, elle, donne l'offset ET le compte. C'est la seule
// lecture correcte — deduire un compte du pas entre deux entrees ITEM est une heuristique
// qui casse des que l'ordre des tableaux change.

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	// itemRacine est l'indice, dans ITEM, de l'objet racine. ITEM[0] est l'entree nulle.
	itemRacine = 1
	// tailleItem est le pas d'une entree ITEM.
	tailleItem = 12
	// masqueTypeItem isole l'indice de type des 24 bits de poids faible d'une entree ITEM.
	masqueTypeItem = 0xFFFFFF
	// modeleElement est le parametre de modele qui porte le type element d'un hkArray.
	modeleElement = "tT"
	// profondeurSectionsMax borne la recursion du parcours des sections.
	profondeurSectionsMax = 8
)

// membreHavok est un champ d'une structure, tel que le fichier le declare.
type membreHavok struct {
	Nom    string
	Offset int
	Type   int
}

// typeHavok est une entree de la table des types.
type typeHavok struct {
	Nom     string
	Parent  int
	Taille  int
	Membres []membreHavok
	// Modeles porte les parametres de modele : "tT" -> indice du type element pour un
	// hkArray. C'est ce qui permet de VERIFIER qu'un tableau porte le type attendu.
	Modeles map[string]int
}

// tableTypes est la table des types d'un fichier-tag, indexee par indice de type.
type tableTypes []typeHavok

// taille rend la taille en octets d'un type, en suivant la chaine des parents pour les
// alias (hkVector4 -> hkVector4f, hkaiIndex -> hkInt32 -> int). Rend 0 si la chaine
// n'aboutit pas.
func (t tableTypes) taille(i int) int {
	for tours := 0; i > 0 && i < len(t) && tours < len(t); tours++ {
		if t[i].Taille > 0 {
			return t[i].Taille
		}
		i = t[i].Parent
	}
	return 0
}

// membre rend le membre nomme d'un type, en remontant la chaine des parents (un champ
// herite est declare sur la classe de base).
func (t tableTypes) membre(i int, nom string) (membreHavok, bool) {
	for tours := 0; i > 0 && i < len(t) && tours < len(t); tours++ {
		for _, m := range t[i].Membres {
			if m.Nom == nom {
				return m, true
			}
		}
		i = t[i].Parent
	}
	return membreHavok{}, false
}

// itemHavok est une entree de la section ITEM.
type itemHavok struct {
	Type   int
	Offset int // relatif au debut de DATA
	Compte int
}

// fichierTag est un fichier-tag Havok deja localise et indexe.
type fichierTag struct {
	data  []byte // le contenu de la section DATA
	types tableTypes
	items []itemHavok
}

// nomType rend le nom d'un indice de type, ou "?" hors bornes.
func (f *fichierTag) nomType(i int) string {
	if i <= 0 || i >= len(f.types) {
		return "?"
	}
	return f.types[i].Nom
}

// racine rend l'objet racine du fichier-tag.
func (f *fichierTag) racine() (itemHavok, error) {
	if len(f.items) <= itemRacine {
		return itemHavok{}, fmt.Errorf("hinavmesh: fichier-tag sans objet racine (%d entrees ITEM)", len(f.items))
	}
	return f.items[itemRacine], nil
}

// tableau suit le pointeur du membre nomme de l'objet donne et rend l'entree ITEM visee,
// apres avoir VERIFIE que son type est le type element declare par le hkArray.
func (f *fichierTag) tableau(obj itemHavok, nom string) (itemHavok, error) {
	m, ok := f.types.membre(obj.Type, nom)
	if !ok {
		return itemHavok{}, fmt.Errorf("hinavmesh: %s ne declare pas le membre %q", f.nomType(obj.Type), nom)
	}
	if m.Type <= 0 || m.Type >= len(f.types) {
		return itemHavok{}, fmt.Errorf("hinavmesh: le membre %q porte le type %d, hors table", nom, m.Type)
	}
	elem, ok := f.types[m.Type].Modeles[modeleElement]
	if !ok {
		return itemHavok{}, fmt.Errorf("hinavmesh: le membre %q est un %s sans parametre de modele %s",
			nom, f.nomType(m.Type), modeleElement)
	}
	pos := obj.Offset + m.Offset
	if pos < 0 || pos+8 > len(f.data) {
		return itemHavok{}, fmt.Errorf("hinavmesh: pointeur du membre %q hors de DATA (offset %d)", nom, pos)
	}
	idx := int(binary.LittleEndian.Uint64(f.data[pos:]))
	if idx <= 0 || idx >= len(f.items) {
		return itemHavok{}, fmt.Errorf("hinavmesh: le membre %q pointe l'item %d, hors des %d entrees",
			nom, idx, len(f.items))
	}
	it := f.items[idx]
	if it.Type != elem {
		return itemHavok{}, fmt.Errorf("hinavmesh: le membre %q vise un tableau de %s, %s attendu",
			nom, f.nomType(it.Type), f.nomType(elem))
	}
	return it, nil
}

// octets rend la tranche de DATA d'une entree ITEM, pour un pas donne.
func (f *fichierTag) octets(it itemHavok, pas int) ([]byte, error) {
	if pas <= 0 {
		return nil, fmt.Errorf("hinavmesh: pas nul pour un tableau de %s", f.nomType(it.Type))
	}
	fin := it.Offset + it.Compte*pas
	if it.Offset < 0 || it.Compte < 0 || fin > len(f.data) || fin < it.Offset {
		return nil, fmt.Errorf("hinavmesh: tableau de %d x %s (pas %d) hors de DATA (%d..%d sur %d)",
			it.Compte, f.nomType(it.Type), pas, it.Offset, fin, len(f.data))
	}
	return f.data[it.Offset:fin], nil
}

// lireFichierTag indexe un fichier-tag : sections, table des types, table des items.
func lireFichierTag(region []byte) (*fichierTag, error) {
	sections := map[string][2]int{}
	if err := parcoursSections(region, 0, len(region), sections, 0); err != nil {
		return nil, err
	}
	if _, ok := sections["TAG0"]; !ok {
		return nil, fmt.Errorf("hinavmesh: region sans section TAG0")
	}
	data, ok := sections["DATA"]
	if !ok {
		return nil, fmt.Errorf("hinavmesh: fichier-tag sans section DATA")
	}
	types, err := lireTypes(region, sections)
	if err != nil {
		return nil, err
	}
	items, err := lireItems(region, sections, len(types))
	if err != nil {
		return nil, err
	}
	return &fichierTag{data: region[data[0] : data[0]+data[1]], types: types, items: items}, nil
}

// parcoursSections remplit la carte avec [offset du contenu, longueur] par etiquette.
// Premiere occurrence gagnante : les etiquettes sont uniques dans un fichier-tag.
func parcoursSections(buf []byte, debut, fin int, dans map[string][2]int, profondeur int) error {
	if profondeur > profondeurSectionsMax {
		return fmt.Errorf("hinavmesh: sections imbriquees au-dela de %d niveaux", profondeurSectionsMax)
	}
	for p := debut; p+8 <= fin; {
		entete := binary.BigEndian.Uint32(buf[p:])
		taille := int(entete & 0x3FFFFFFF)
		etiquette := string(buf[p+4 : p+8])
		if taille < 8 || p+taille > fin {
			return fmt.Errorf("hinavmesh: section %q de taille %d a l'offset %d deborde (fin %d)",
				etiquette, taille, p, fin)
		}
		if _, vu := dans[etiquette]; !vu {
			dans[etiquette] = [2]int{p + 8, taille - 8}
		}
		if entete&0x40000000 == 0 {
			if err := parcoursSections(buf, p+8, p+taille, dans, profondeur+1); err != nil {
				return err
			}
		}
		p += taille
	}
	return nil
}

// lireItems decode la section ITEM.
func lireItems(buf []byte, sections map[string][2]int, nbTypes int) ([]itemHavok, error) {
	sec, ok := sections["ITEM"]
	if !ok {
		return nil, fmt.Errorf("hinavmesh: fichier-tag sans section ITEM")
	}
	items := make([]itemHavok, 0, sec[1]/tailleItem)
	for p := 0; p+tailleItem <= sec[1]; p += tailleItem {
		o := sec[0] + p
		typ := int(binary.LittleEndian.Uint32(buf[o:]) & masqueTypeItem)
		if typ >= nbTypes {
			return nil, fmt.Errorf("hinavmesh: entree ITEM %d de type %d, hors des %d types",
				len(items), typ, nbTypes)
		}
		items = append(items, itemHavok{
			Type:   typ,
			Offset: int(binary.LittleEndian.Uint32(buf[o+4:])),
			Compte: int(binary.LittleEndian.Uint32(buf[o+8:])),
		})
	}
	return items, nil
}

// chaines decoupe une section de chaines terminees par un octet nul.
func chaines(buf []byte, sec [2]int) []string {
	morceaux := bytes.Split(buf[sec[0]:sec[0]+sec[1]], []byte{0})
	out := make([]string, len(morceaux))
	for i, m := range morceaux {
		out[i] = string(m)
	}
	return out
}
