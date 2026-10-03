package main

// vehicules_sons.go — mode `vehi-sons` (LECTURE SEULE) : la chaine son de DEPLACEMENT.
//
// POURQUOI CE MODE. Le son de deplacement d'un vehicule n'est PAS un event `snd!` de la banque
// du vehicule pris par structure (essai refuse deux fois a l'oreille) : c'est le `lsnd`
// (sound_looping) que le tag `vehi` REFERENCE — demarrage / boucle / arret, module par un RTPC
// de vitesse. Ce mode ouvre la chaine, soeur de celle des equipements (`eqip.go`) : seule la
// RACINE change (`vehi` au lieu de `eqip`), et la reference se lit INLINE dans le corps du
// `vehi` (comme la ref `hlmt`, cf. `internal/himap/vehicules.go`) autant que dans la table de
// dependances — on prend l'union des deux.
//
// PASSE 1, module `any/globals` (0,62 Go) : vehi -> lsnd/snd!/effe -> sbnk + mots (candidats
// d'evenement). PASSE 2 (module `pc/globals`) : `lot -banks <sbnk>` resout les events -> wems.

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/himodule"
)

// VehiSon est la chaine son d'UN tag `vehi`, telle que la passe 1 la rend.
type VehiSon struct {
	Vehi  string   `json:"vehi"`
	Weaps []string `json:"weaps,omitempty"` // refs `weap` inline — identifie le vehicule (croise lot2)
	Sons  []SndSon `json:"sons"`
	Banks []string `json:"banks"`
}

// RapportVehiSons est le JSON echange avec la passe 2.
type RapportVehiSons struct {
	Module    string    `json:"module"`
	Total     int       `json:"total_vehi"`
	AvecSon   int       `json:"vehi_avec_son"`
	Vehicules []VehiSon `json:"vehicules"`
}

// sonsDeVehicules est la PASSE 1 : vehi -> lsnd/snd!/effe -> sbnk, sur le module des definitions.
func sonsDeVehicules(cheminModule, sortie string) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")
	sons := map[uint32]himodule.File{}
	for _, g := range []string{"snd!", "lsnd"} {
		for _, f := range m.Files(g) {
			sons[f.GlobalID] = f
		}
	}
	effets := map[uint32]himodule.File{}
	for _, f := range m.Files("effe") {
		effets[f.GlobalID] = f
	}
	weaps := map[uint32]bool{}
	for _, f := range m.Files("weap") {
		weaps[f.GlobalID] = true
	}
	vehis := m.Files("vehi")
	fmt.Printf("module : %d tags vehi, %d snd!+lsnd, %d effe\n\n", len(vehis), len(sons), len(effets))
	rap := RapportVehiSons{Module: cheminModule, Total: len(vehis)}
	for _, f := range vehis {
		if v, ok := chaineSonVehi(m, f, sons, effets); ok {
			v.Weaps = weapsInline(m, f, weaps)
			rap.Vehicules = append(rap.Vehicules, v)
			if len(v.Banks) > 0 {
				rap.AvecSon++
			}
		}
	}
	sort.Slice(rap.Vehicules, func(i, j int) bool { return rap.Vehicules[i].Vehi < rap.Vehicules[j].Vehi })
	for _, v := range rap.Vehicules {
		fmt.Printf("  vehi %s weap%v : %d son(s) [%s], banks %v\n", v.Vehi, v.Weaps, len(v.Sons), groupesDeSons(v.Sons), v.Banks)
	}
	fmt.Printf("\n%d/%d vehi portent un son, %d atteignent une bank -> %s\n", len(rap.Vehicules), len(vehis), rap.AvecSon, sortie)
	if sortie == "" {
		return nil
	}
	b, err := json.MarshalIndent(rap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sortie, b, 0o644)
}

// refsSonInline scanne le corps du `vehi` et rend les gids qui sont un tag de son ou d'effet,
// avec leur groupe. C'est la lecture INLINE (`himap.RefsInline`) : robuste au layout du `vehi`,
// qui reference ses tags par GlobalID brut et non toujours par la table de dependances.
func refsSonInline(data []byte, sons, effets map[uint32]himodule.File) map[uint32]string {
	out := map[uint32]string{}
	for o := 0; o+4 <= len(data); o++ {
		h := binary.LittleEndian.Uint32(data[o:])
		if f, ok := sons[h]; ok {
			out[h] = f.Group
		} else if _, ok := effets[h]; ok {
			out[h] = "effe"
		}
	}
	return out
}

// weapsInline rend les refs `weap` lues INLINE dans le corps du `vehi` — l'identite du
// vehicule (R-VEHICULE : un `vehi` reference son armement par GlobalID), a croiser avec lot2.
func weapsInline(m *himodule.Module, f himodule.File, weaps map[uint32]bool) []string {
	data, err := m.Extract(f)
	if err != nil {
		return nil
	}
	vus := map[uint32]bool{}
	var out []string
	for o := 0; o+4 <= len(data); o++ {
		h := binary.LittleEndian.Uint32(data[o:])
		if weaps[h] && !vus[h] {
			vus[h] = true
			out = append(out, fmt.Sprintf("%08x", h))
		}
	}
	sort.Strings(out)
	return out
}

// chaineSonVehi descend un `vehi` vers ses sons (lsnd/snd! directs + une couche d'effe).
// Rend false si le tag n'atteint aucun son. Union des refs inline et de la table de deps.
func chaineSonVehi(m *himodule.Module, f himodule.File, sons, effets map[uint32]himodule.File) (VehiSon, bool) {
	data, err := m.Extract(f)
	if err != nil {
		return VehiSon{}, false
	}
	refs := refsSonInline(data, sons, effets)
	for _, d := range dependances(data) {
		if _, ok := sons[d.IDGlobal]; ok {
			refs[d.IDGlobal] = d.Groupe
		} else if _, ok := effets[d.IDGlobal]; ok {
			refs[d.IDGlobal] = "effe"
		}
	}
	v := VehiSon{Vehi: fmt.Sprintf("%08x", f.GlobalID)}
	banks := map[string]bool{}
	gids := make([]uint32, 0, len(refs))
	for g := range refs {
		gids = append(gids, g)
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })
	for _, gid := range gids {
		if refs[gid] == "effe" {
			for _, s := range sonsParEffet(m, effets[gid]) {
				ajouterSonVehi(m, &v, banks, sons, s, "effe>snd!")
			}
			continue
		}
		ajouterSonVehi(m, &v, banks, sons, gid, refs[gid])
	}
	if len(v.Sons) == 0 {
		return VehiSon{}, false
	}
	for b := range banks {
		v.Banks = append(v.Banks, b)
	}
	sort.Strings(v.Banks)
	return v, true
}

// ajouterSonVehi ajoute un tag de son a la chaine, avec ses banks et les mots de son corps.
func ajouterSonVehi(m *himodule.Module, v *VehiSon, banks map[string]bool, sons map[uint32]himodule.File, gid uint32, groupe string) {
	for _, s := range v.Sons {
		if s.Tag == fmt.Sprintf("%08x", gid) {
			return
		}
	}
	s := SndSon{Tag: fmt.Sprintf("%08x", gid), Groupe: groupe}
	if sf, ok := sons[gid]; ok {
		b, mots := banksDuSon(m, sf)
		s.Banks, s.Mots = b, mots
		for _, x := range b {
			banks[x] = true
		}
	}
	v.Sons = append(v.Sons, s)
}

// groupesDeSons rend un resume compact des groupes de son atteints (pour l'affichage).
func groupesDeSons(sons []SndSon) string {
	compte := map[string]int{}
	for _, s := range sons {
		compte[s.Groupe]++
	}
	var parts []string
	for g, n := range compte {
		parts = append(parts, fmt.Sprintf("%s:%d", g, n))
	}
	sort.Strings(parts)
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}
