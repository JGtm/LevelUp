package main

// tir_vehicules.go — mode `tir-vehi` : le son de TIR d'une ARME DE VEHICULE, par le champ
// NOMME du tag `weap`, sans passer par la passe 1 des `.pck` d'armes.
//
// POURQUOI CE MODE EXISTE (mesure du 2026-09-02, lot V3F). La chaine `lot` -> `lot-tir` ne
// balaie que les packs `sb_010_wea_*`, `sb_010_tur_*` et `sb_010_whizby_*` : les chassis
// `sb_010_veh_*` n'y entrent pas, et un `weap` de vehicule n'a donc jamais ete resolu par
// elle. Or la banque du chassis ne porte PAS le tir : dump HIRC integral des quatre banques
// covenant / bannis (`ccd43fa8` Ghost, `fda12da2` Wraith, `c682f736` Banshee, `1bb9f097`
// Chopper) — 26 evenements distincts, tous sur les bus du moteur (`5a880943`, `1f17314c`),
// du contact au sol (`8165b6c5`) ou du moteur lointain (`0f233096`). Aucun evenement de
// tir. Le tir vit ailleurs, et ce mode dit OU en lisant le champ « Weapon Fire Sound ».
//
// Chaine, identique a celle de `lot_tir.go` mais amorcee par le `vehi` et non par le pack :
//
//	vehi --[refs `weap` INLINE]--> weap
//	weap --[champ nomme « Weapon Fire Sound » de weap.xml]--> tags `lsnd`/`snd!` de tir
//	tag de son --[table de dependances]--> `sbnk`   ET   --[corps]--> mots (candidats event)
//
// Le rattachement final (quel mot du corps est un evenement) se tranche en PASSE 2, en
// intersectant les mots avec les evenements reellement declares par la banque
// (`-mode hirc-event -banks <sbnk>`). Ce mode ne devine rien : il publie les deux listes.
//
// Module attendu : `any/globals` (0,62 Go), celui qui porte `vehi`, `weap`, `snd!`, `lsnd`.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/himodule"
)

// SonTirVehi : un tag de son designe par « Weapon Fire Sound », avec ce qu'il atteint.
type SonTirVehi struct {
	Tag    string   `json:"tag"`
	Groupe string   `json:"groupe"`
	Mode   int      `json:"mode"` // index de l'element du tableau « Weapon Fire Sounds »
	Banks  []string `json:"banks"`
	Mots   []uint32 `json:"mots,omitempty"`
}

// ArmeVehi : un `weap` de vehicule et son son de tir.
type ArmeVehi struct {
	Weap       string       `json:"weap"`
	Vehis      []string     `json:"vehis,omitempty"`
	CadenceCPM int          `json:"cadence_cpm,omitempty"`
	Modes      int          `json:"modes"`
	Sons       []SonTirVehi `json:"sons_de_tir"`
	Banks      []string     `json:"banks"`
}

// RapportTirVehi est la sortie du mode.
type RapportTirVehi struct {
	Module string     `json:"module"`
	Armes  []ArmeVehi `json:"armes"`
}

// tirDesVehicules est le mode `tir-vehi`.
//
// `cibles` vide = toutes les armes referencees par au moins un `vehi` ; sinon, exactement
// les `weap` demandes (utile quand l'arme est portee par un `scen` et non par le `vehi`).
func tirDesVehicules(cheminModule string, cibles map[uint32]bool, sortie string) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	o, err := calculerOffsets()
	if err != nil {
		return err
	}
	groupes := make(map[uint32]string, 1<<17)
	for _, f := range m.Files("") {
		groupes[f.GlobalID] = f.Group
	}
	sons := map[uint32]himodule.File{}
	for _, g := range []string{"snd!", "lsnd"} {
		for _, f := range m.Files(g) {
			sons[f.GlobalID] = f
		}
	}
	weaps := m.Files("weap")
	porteurs := vehisParWeap(m, weaps)
	fmt.Printf("module : %d weap, %d snd!+lsnd, %d weap portes par un vehi\n\n",
		len(weaps), len(sons), len(porteurs))

	rap := RapportTirVehi{Module: cheminModule}
	for _, f := range weaps {
		if !retenirWeap(f.GlobalID, cibles, porteurs) {
			continue
		}
		a, ok := armeDeVehicule(m, f, o, groupes, sons, porteurs)
		if !ok {
			continue
		}
		rap.Armes = append(rap.Armes, a)
	}
	sort.Slice(rap.Armes, func(i, j int) bool { return rap.Armes[i].Weap < rap.Armes[j].Weap })
	afficherTirVehi(rap)
	if sortie == "" {
		return nil
	}
	b, err := json.MarshalIndent(rap, "", " ")
	if err != nil {
		return err
	}
	fmt.Printf("\nrapport ecrit : %s\n", sortie)
	return os.WriteFile(sortie, b, 0o644)
}

// retenirWeap dit si une arme entre dans le rapport.
func retenirWeap(gid uint32, cibles map[uint32]bool, porteurs map[uint32][]string) bool {
	if len(cibles) > 0 {
		return cibles[gid]
	}
	return len(porteurs[gid]) > 0
}

// armeDeVehicule resout UNE arme : ses modes de tir, leurs tags de son, leurs banques.
func armeDeVehicule(m *himodule.Module, f himodule.File, o offsetsTir,
	groupes map[uint32]string, sons map[uint32]himodule.File,
	porteurs map[uint32][]string) (ArmeVehi, bool) {
	data, err := m.Extract(f)
	if err != nil {
		return ArmeVehi{}, false
	}
	t, err := ouvrirTagWeap(data)
	if err != nil {
		return ArmeVehi{}, false
	}
	modes := modesDansTag(t, data, groupes)
	if len(modes) == 0 {
		return ArmeVehi{}, false
	}
	a := ArmeVehi{
		Weap:  fmt.Sprintf("%08x", f.GlobalID),
		Vehis: porteurs[f.GlobalID],
		Modes: len(modes),
	}
	if c := lireCadences(data); len(c) == 1 && c[0].Max > 0 {
		a.CadenceCPM = int(c[0].Max * 60)
	}
	vues := map[string]bool{}
	for i, mode := range modes {
		for _, gid := range mode {
			s := SonTirVehi{Tag: fmt.Sprintf("%08x", gid), Groupe: groupes[gid], Mode: i}
			if sf, ok := sons[gid]; ok {
				s.Banks, s.Mots = banksDuSon(m, sf)
			}
			for _, b := range s.Banks {
				if !vues[b] {
					vues[b] = true
					a.Banks = append(a.Banks, b)
				}
			}
			a.Sons = append(a.Sons, s)
		}
	}
	sort.Strings(a.Banks)
	return a, true
}

// vehisParWeap rend, par `weap`, les `vehi` qui le referencent INLINE.
//
// Lecture inline et non table de dependances : un `vehi` designe son armement par
// GlobalID brut (regle R-VEHICULE, `vehicules_sons.go`), et la table de dependances ne le
// porte pas toujours.
func vehisParWeap(m *himodule.Module, weaps []himodule.File) map[uint32][]string {
	ens := make(map[uint32]bool, len(weaps))
	for _, f := range weaps {
		ens[f.GlobalID] = true
	}
	out := map[uint32][]string{}
	for _, f := range m.Files("vehi") {
		for _, s := range weapsInline(m, f, ens) {
			var gid uint32
			if _, err := fmt.Sscanf(s, "%08x", &gid); err != nil {
				continue
			}
			out[gid] = append(out[gid], fmt.Sprintf("%08x", f.GlobalID))
		}
	}
	for gid := range out {
		sort.Strings(out[gid])
	}
	return out
}

// afficherTirVehi imprime le tableau, une ligne par arme puis une par tag de son.
func afficherTirVehi(rap RapportTirVehi) {
	fmt.Printf("--- %d arme(s) de vehicule avec un son de tir ---\n", len(rap.Armes))
	for _, a := range rap.Armes {
		fmt.Printf("  weap %s  vehis%v  %d mode(s)  cadence=%d cpm  banks%v\n",
			a.Weap, a.Vehis, a.Modes, a.CadenceCPM, a.Banks)
		for _, s := range a.Sons {
			fmt.Printf("      mode %d  %s %s  banks%v  %d mots\n",
				s.Mode, s.Groupe, s.Tag, s.Banks, len(s.Mots))
		}
	}
}
