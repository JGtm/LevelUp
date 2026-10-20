package main

// eqip.go — LA CHAINE SON D'UN OBJET D'EQUIPEMENT, soeur de la chaine d'arme.
//
// POURQUOI ELLE N'EXISTAIT PAS. Tout l'outil part du champ NOMME « Weapon Fire Sounds » du
// tag `weap` (cf. `weapfire.go`) : c'est le chemin qui donne LE TIR parmi les 80 a 360 sons
// d'une arme. Un equipement n'a pas de tir, pas de `weap`, et le groupe `eqip` n'apparaissait
// nulle part dans l'outil — d'ou le negatif du lot R3-V (« la chaine d'extraction ne connait
// pas le groupe `eqip` »). Ce fichier ouvre ce chemin-la.
//
// LA MESURE QUI LE JUSTIFIE (2026-08-18, sonde `himap/sonde_eqip_inventaire_gamefiles_test.go`) :
// sur les 116 tags `eqip` du jeu, **41 portent une dependance `snd!`** (69 references). Le lien
// tag -> son EXISTE donc pour l'equipement ; il ne passe simplement pas par un champ nomme.
//
// DEUX PASSES, DEUX PROCESSUS — la contrainte memoire de la recette est inchangee :
//
//	passe 1 (`eqip-sons`)  module `any/globals` (0,62 Go) : eqip -> snd!/lsnd -> sbnk
//	passe 2 (`eqip-banks`) module `pc/globals` (7,24 Go)  : sbnk -> .wem -> pack nomme
//
// L'echange se fait par JSON, comme `lot` -> `lot-tir`. Ne JAMAIS charger les deux modules
// dans le meme processus.
//
// LE PONT DE NOMMAGE. Les `.pck` du jeu portent des noms EN CLAIR
// (`sb_007_abl_repairfield.pck`) quand les tags n'en portent aucun (`stringsSize` = 0). Un
// equipement dont la chaine atteint les `.wem` d'un pack nomme est donc NOMME par ce pack —
// a condition qu'aucun autre equipement n'atteigne le meme pack (temoin de selectivite, sans
// quoi la coincidence n'est pas un nom). C'est la passe 2 qui rend ce croisement.

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/himodule"
)

// EqipSon est la chaine son d'UN tag `eqip`, telle que la passe 1 la rend.
type EqipSon struct {
	Eqip  string   `json:"eqip"`
	Sofa  []string `json:"sofa,omitempty"`
	Sons  []SndSon `json:"sons"`
	Banks []string `json:"banks"`
}

// SndSon est un tag de son (`snd!` ou `lsnd`), les banks qu'il reference, et les mots de
// 32 bits de son corps.
//
// POURQUOI LES MOTS VOYAGENT. Un `snd!` DESIGNE un evenement Wwise par son identifiant, et
// c'est la seule chose qui distingue « le son du deploiement » de « le son du ramassage »
// dans une bank qui en porte trente. Mais les `snd!` vivent dans `any/globals` et les banks
// dans `pc/globals` : les deux ne peuvent pas etre ouverts par le meme processus. Le rapport
// transporte donc les mots BRUTS, et la passe 2 les intersecte avec les Events de la bank —
// exactement la methode que `chercherPorteurs` emploie pour les armes, retournee.
type SndSon struct {
	Tag    string   `json:"tag"`
	Groupe string   `json:"groupe"`
	Banks  []string `json:"banks"`
	Mots   []uint32 `json:"mots,omitempty"`
}

// RapportEqipSons est le JSON echange entre les deux passes.
type RapportEqipSons struct {
	Module     string    `json:"module"`
	Total      int       `json:"total_eqip"`
	AvecSon    int       `json:"eqip_avec_son"`
	Equipement []EqipSon `json:"equipement"`
}

// sonsDEquipement est la PASSE 1 : eqip -> snd!/lsnd -> sbnk, sur le module des definitions.
//
// `cibles` vide = tous les `eqip` du module. La sortie JSON alimente la passe 2.
func sonsDEquipement(cheminModule string, cibles map[uint32]bool, sortie string) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	// Index des tags de son ET des effets : leur GROUPE est necessaire pour lire leurs
	// dependances sans re-balayer le module a chaque fois.
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
	eqips := m.Files("eqip")
	fmt.Printf("module : %d tags eqip, %d tags snd!+lsnd, %d tags effe\n\n",
		len(eqips), len(sons), len(effets))

	rap := RapportEqipSons{Module: cheminModule, Total: len(eqips)}
	for _, f := range eqips {
		if len(cibles) > 0 && !cibles[f.GlobalID] {
			continue
		}
		e, ok := chaineSonEqip(m, f, sons, effets)
		if !ok {
			continue
		}
		rap.Equipement = append(rap.Equipement, e)
		if len(e.Banks) > 0 {
			rap.AvecSon++
		}
	}
	sort.Slice(rap.Equipement, func(i, j int) bool {
		return rap.Equipement[i].Eqip < rap.Equipement[j].Eqip
	})
	afficherChaineEqip(rap)
	if sortie == "" {
		return nil
	}
	b, err := json.MarshalIndent(rap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sortie, b, 0o644)
}

// chaineSonEqip descend d'un tag `eqip` vers ses banks. Rend false si le tag n'a aucun son.
func chaineSonEqip(
	m *himodule.Module, f himodule.File, sons, effets map[uint32]himodule.File,
) (EqipSon, bool) {
	data, err := m.Extract(f)
	if err != nil {
		return EqipSon{}, false
	}
	e := EqipSon{Eqip: fmt.Sprintf("%08x", f.GlobalID)}
	banks := map[string]bool{}
	ajoute := func(gid uint32, groupe string) {
		s := SndSon{Tag: fmt.Sprintf("%08x", gid), Groupe: groupe}
		if sf, ok := sons[gid]; ok {
			b, mots := banksDuSon(m, sf)
			s.Banks, s.Mots = b, mots
			for _, x := range b {
				banks[x] = true
			}
		}
		e.Sons = append(e.Sons, s)
	}
	for _, d := range dependances(data) {
		switch d.Groupe {
		case "sofa":
			e.Sofa = append(e.Sofa, fmt.Sprintf("%08x", d.IDGlobal))
		case "snd!", "lsnd":
			ajoute(d.IDGlobal, d.Groupe)
		case "effe":
			// LE SON PROPRE A CHAQUE EQUIPEMENT NE PEUT PAS ETRE DANS LA TABLE DE
			// DEPENDANCES DU `eqip` : mesure du 2026-08-18, les memes deux `snd!`
			// (7b5cbe75, 725186aa) y sont partages par 21 objets d'equipement, du mur au
			// surbouclier. Un effet (`effe`), lui, est PROPRE a un geste — c'est le maillon
			// que la chaine doit traverser. On descend d'UN niveau, pas davantage : un
			// parcours recursif ramasserait tout le graphe et perdrait la selectivite.
			ef, ok := effets[d.IDGlobal]
			if !ok {
				continue
			}
			for _, s := range sonsParEffet(m, ef) {
				ajoute(s, "effe>snd!")
			}
		}
	}
	if len(e.Sons) == 0 {
		return EqipSon{}, false
	}
	for b := range banks {
		e.Banks = append(e.Banks, b)
	}
	sort.Strings(e.Banks)
	return e, true
}

// sonsParEffet rend les tags `snd!`/`lsnd` qu'un tag `effe` reference.
func sonsParEffet(m *himodule.Module, f himodule.File) []uint32 {
	data, err := m.Extract(f)
	if err != nil {
		return nil
	}
	var out []uint32
	for _, d := range dependances(data) {
		if d.Groupe == "snd!" || d.Groupe == "lsnd" {
			out = append(out, d.IDGlobal)
		}
	}
	return out
}

// banksDuSon rend les `sbnk` qu'un tag de son reference et les mots de 32 bits de son CORPS.
//
// Un `snd!` porte exactement une bank (mesure du chantier armes, cf. `histogrammeBanks`) ;
// on ne le postule pas pour autant. Les mots sont lus APRES l'en-tete et la table de
// dependances : le corps d'un `snd!` fait quelques dizaines d'octets, et c'est la que
// l'identifiant d'evenement se trouve.
func banksDuSon(m *himodule.Module, f himodule.File) ([]string, []uint32) {
	data, err := m.Extract(f)
	if err != nil {
		return nil, nil
	}
	deps := dependances(data)
	var out []string
	for _, d := range deps {
		if d.Groupe == "sbnk" {
			out = append(out, fmt.Sprintf("%08x", d.IDGlobal))
		}
	}
	var mots []uint32
	for o := tailleEnteteT + len(deps)*tailleDep; o+4 <= len(data); o += 4 {
		mots = append(mots, binary.LittleEndian.Uint32(data[o:]))
	}
	return out, mots
}

// afficherChaineEqip rend le tableau de la passe 1, une ligne par `eqip` sonore.
func afficherChaineEqip(rap RapportEqipSons) {
	fmt.Printf("%d tags eqip portent un son, %d atteignent une bank\n\n",
		len(rap.Equipement), rap.AvecSon)
	parBank := map[string][]string{}
	for _, e := range rap.Equipement {
		var libs []string
		for _, s := range e.Sons {
			libs = append(libs, fmt.Sprintf("%s:%s->[%s]", s.Groupe, s.Tag, strings.Join(s.Banks, ",")))
		}
		fmt.Printf("  eqip %s  sofa[%s]  %s\n",
			e.Eqip, strings.Join(e.Sofa, ","), strings.Join(libs, " "))
		for _, b := range e.Banks {
			parBank[b] = append(parBank[b], e.Eqip)
		}
	}
	cles := make([]string, 0, len(parBank))
	for b := range parBank {
		cles = append(cles, b)
	}
	sort.Strings(cles)
	fmt.Printf("\n--- %d banks atteintes, et QUI les atteint (temoin de selectivite) ---\n", len(cles))
	for _, b := range cles {
		fmt.Printf("  sbnk %s : %d eqip  [%s]\n", b, len(parBank[b]), strings.Join(parBank[b], " "))
	}
}
