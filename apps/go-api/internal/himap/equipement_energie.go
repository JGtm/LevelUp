package himap

// equipement_energie.go — L ENERGIE DES EQUIPEMENTS, LUE DANS LES TAGS `eqip` DU JEU (lot
// « equipement porte » du plan d ajustements du 2026-10-14).
//
// # LA CHAINE, TELLE QUE LE JEU LA PORTE
//
// Un tag `sofa` (la fiche d un equipement de la palette) reference DEUX `eqip` : l objet RAMASSABLE
// (celui que le film cree au sol, nomme par son GlobalID) et l objet PORTE (celui que le film cree
// attache au joueur qui le prend). Le second porte la capacite ; le premier, l energie de depart.
//
//	eqip + 0x8ac  energie initiale (f32). `FUN_140ea3f68` la copie dans l instantane reseau de
//	              l objet : un objet porte cree avec une autre energie l ecrit dans son record NEW
//	              (i24) — c est l energie de l objet ramasse.
//	eqip + 0x904  bloc des capacites (element de 0x344 octets, `FUN_14049a384`) :
//	              capacite + 0x1c  seuil (f32) ; capacite + 0x24  variation par usage (f32).
//	              A l usage, `FUN_143181708` ajoute la variation a l energie (bornee a [0, 1]) ;
//	              le HUD compte les usages restants par `FUN_1407e9384` :
//	              seuil > 0 → trunc((energie + 1e-4) / -variation) ; sinon le compteur objet+0x50b.
//
// LECTURE SEULE des modules du jeu installe, UN module en memoire a la fois.

import (
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/himodule"
)

// Les offsets du bloc racine d un `eqip` et de son element de capacite, lus dans l executable.
const (
	EqipOffsetEnergieInitiale = 0x8ac
	EqipOffsetCapacites       = 0x904
	CapaciteOffsetSeuil       = 0x1c
	CapaciteOffsetVariation   = 0x24
	capaciteTaille            = 0x344
)

// ModulesDesEquipements sont les modules qui portent les `eqip` et les `sofa` (chemins relatifs a la
// racine de deploiement) : les seuls ou le balayage en trouve sur le jeu installe.
var ModulesDesEquipements = []string{"any/globals/globals-rtx-new.module", "any/globals/common-rtx-new.module"}

// EnergieDEquipement est UNE paire d equipements qu une fiche `sofa` relie, avec ses constantes.
type EnergieDEquipement struct {
	// Sofa est la fiche ; Ramassable et Porte les GlobalID des deux `eqip`.
	Sofa, Ramassable, Porte uint32
	// EnergieRamassable et EnergiePorte : les energies initiales (+0x8ac) des deux objets.
	EnergieRamassable, EnergiePorte float32
	// Seuil et Variation : la capacite de l objet porte (+0x904 → +0x1c, +0x24).
	Seuil, Variation float32
	// Module : le module ou les tags ont ete lus.
	Module string
}

// tagDEquipement est ce qu un `eqip` donne a la table : ses constantes et son module.
type tagDEquipement struct {
	energie, seuil, variation float32
	capacite                  bool
	module                    string
}

// LireLesEnergiesDEquipement lit, sous la racine de deploiement `root`, chaque fiche `sofa` qui
// relie exactement un `eqip` porteur de capacite (seuil > 0) et un `eqip` sans capacite, et rend
// la paire avec ses constantes, triee par GlobalID de l objet ramassable.
func LireLesEnergiesDEquipement(root string) ([]EnergieDEquipement, error) {
	eqips := map[uint32]tagDEquipement{}
	sofas := map[uint32][]uint32{}
	sofaModule := map[uint32]string{}
	for _, rel := range ModulesDesEquipements {
		m, err := himodule.Open(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("module %s : %w", rel, err)
		}
		for _, f := range m.Files("eqip") {
			if _, deja := eqips[f.GlobalID]; deja {
				continue
			}
			raw, err := m.Extract(f)
			if err != nil {
				return nil, fmt.Errorf("eqip 0x%08x (%s) : %w", f.GlobalID, rel, err)
			}
			if t, ok := lireUnEqip(raw); ok {
				t.module = rel
				eqips[f.GlobalID] = t
			}
		}
		for _, f := range m.Files("sofa") {
			if _, deja := sofas[f.GlobalID]; deja {
				continue
			}
			raw, err := m.Extract(f)
			if err != nil {
				return nil, fmt.Errorf("sofa 0x%08x (%s) : %w", f.GlobalID, rel, err)
			}
			sofas[f.GlobalID] = dependancesDuGroupe(raw, "eqip")
			sofaModule[f.GlobalID] = rel
		}
	}
	var out []EnergieDEquipement
	for sofa, deps := range sofas {
		if e, ok := paireDeLaFiche(sofa, deps, eqips); ok {
			e.Module = sofaModule[sofa]
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ramassable != out[j].Ramassable {
			return out[i].Ramassable < out[j].Ramassable
		}
		return out[i].Sofa < out[j].Sofa
	})
	return out, nil
}

// paireDeLaFiche rend la paire d une fiche : exactement un `eqip` a capacite et un sans.
func paireDeLaFiche(sofa uint32, deps []uint32, eqips map[uint32]tagDEquipement) (EnergieDEquipement, bool) {
	var porte, ramassable []uint32
	for _, d := range deps {
		t, ok := eqips[d]
		switch {
		case !ok:
		case t.capacite:
			porte = append(porte, d)
		default:
			ramassable = append(ramassable, d)
		}
	}
	if len(porte) != 1 || len(ramassable) != 1 {
		return EnergieDEquipement{}, false
	}
	p, r := eqips[porte[0]], eqips[ramassable[0]]
	return EnergieDEquipement{Sofa: sofa, Ramassable: ramassable[0], Porte: porte[0],
		EnergieRamassable: r.energie, EnergiePorte: p.energie, Seuil: p.seuil, Variation: p.variation}, true
}

// lireUnEqip lit l energie initiale et la capacite d un `eqip`.
func lireUnEqip(raw []byte) (tagDEquipement, bool) {
	ti, err := meilleurTagInfo(raw)
	if err != nil {
		return tagDEquipement{}, false
	}
	ri, err := ti.rootBlockIndex()
	if err != nil {
		return tagDEquipement{}, false
	}
	abs, size := ti.blockAbs(ri)
	if size < EqipOffsetEnergieInitiale+4 || abs+size > len(raw) {
		return tagDEquipement{}, false
	}
	t := tagDEquipement{energie: f32Brut(raw, abs+EqipOffsetEnergieInitiale)}
	if c := ti.subBlockTarget(ri, EqipOffsetCapacites); c >= 0 {
		cabs, csize := ti.blockAbs(c)
		if csize >= capaciteTaille && cabs+csize <= len(raw) {
			t.seuil = f32Brut(raw, cabs+CapaciteOffsetSeuil)
			t.variation = f32Brut(raw, cabs+CapaciteOffsetVariation)
			t.capacite = t.seuil > 0
		}
	}
	return t, true
}

// dependancesDuGroupe rend les GlobalID des dependances d un tag dont le groupe est `grp`.
func dependancesDuGroupe(raw []byte, grp string) []uint32 {
	ti, err := meilleurTagInfo(raw)
	if err != nil {
		return nil
	}
	var out []uint32
	tab := ti.blockTab - ti.deps*depEntrySize
	for i := 0; i < ti.deps; i++ {
		e := tab + i*depEntrySize
		if e < 0 || e+depEntrySize > len(raw) {
			continue
		}
		v := binary.LittleEndian.Uint32(raw[e:])
		if string([]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}) == grp {
			out = append(out, binary.LittleEndian.Uint32(raw[e+0x10:]))
		}
	}
	return out
}

// f32Brut lit un flottant de 32 bits petit-boutiste.
func f32Brut(b []byte, o int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b[o:]))
}
