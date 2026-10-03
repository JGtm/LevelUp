package main

// melee.go — le son de CORPS A CORPS, lu par son champ nomme.
//
// POURQUOI CE MODE. Les sons de l'epee a energie et du marteau antigravite remontent au
// `weap` par les graphes d'ANIMATION (`jmad`), pas par la chaine sonore : mesure sur la
// bank de l'epee, `bank <- 21 snd!/lsnd <- 100 jmad <- 8 weap`. Traverser `jmad` serait
// desastreux — c'est un carrefour (98 `Rani` au niveau suivant), tout s'y rattacherait a
// tout.
//
// Il n'y a pas besoin de le traverser : `weap.xml` expose un champ NOMME « melee sound »,
// reference de tag (`_41`, 28 octets) au niveau racine de l'objet. On saute donc
// directement au bon endroit, exactement comme pour « Weapon Fire Sound ».
//
// DERIVE : le plugin annonce des offsets qui ne correspondent plus au build (mesure : deux
// champs independants decales de +64). On essaie donc l'offset annonce ET les decalages
// observes, en ne retenant qu'un candidat qui pointe VERS UN TAG DE SON.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/himodule"
)

// derivesConnues : decalages a essayer entre l'offset du plugin et le tag reel. Le +64 est
// mesure sur « Weapon Fire Sounds » (+4288 -> +4352) et « barrels » (+3220 -> +3284).
var derivesConnues = []int{64, 0, 32, 96, 128}

// refNommee cherche une reference de tag `_41` designee par son nom dans le plugin.
// Rend l'identifiant vise et le decalage retenu.
func refNommee(data []byte, nomChamp string, groupes map[uint32]string) (uint32, int, bool) {
	champs, err := champsPlugin()
	if err != nil {
		return 0, 0, false
	}
	c, ok := trouverChamp(champs, nomChamp)
	if !ok {
		return 0, 0, false
	}
	t, err := ouvrirTagWeap(data)
	if err != nil {
		return 0, 0, false
	}
	racine, err := t.blocRacine()
	if err != nil {
		return 0, 0, false
	}
	abs, taille := t.blocAbs(racine)
	if abs < 0 {
		return 0, 0, false
	}
	for _, d := range derivesConnues {
		off := c.off + d
		if off < 0 || off+tailleRef > taille {
			continue
		}
		gid := gidRef(data, abs+off)
		if gid == 0 || gid == 0xffffffff {
			continue
		}
		if g := groupes[gid]; g == "snd!" || g == "lsnd" {
			return gid, d, true
		}
	}
	return 0, 0, false
}

// sonsDeMelee affiche, pour chaque `weap` du module, son son de corps a corps.
func sonsDeMelee(cheminModule string, gidVoulu uint32) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")
	groupes := make(map[uint32]string, 1<<17)
	bankDe := banksDesSons(m)
	for _, f := range m.Files("") {
		groupes[f.GlobalID] = f.Group
	}

	type ligne struct {
		weap, son, bank uint32
		derive          int
	}
	var out []ligne
	for _, f := range m.Files("weap") {
		if gidVoulu != 0 && f.GlobalID != gidVoulu {
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		son, d, ok := refNommee(data, "melee sound", groupes)
		if !ok {
			continue
		}
		out = append(out, ligne{f.GlobalID, son, bankDe[son], d})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].weap < out[j].weap })
	fmt.Printf("\n%d tag(s) weap avec un son de melee resolu\n\n", len(out))
	fmt.Printf("%-10s %-10s %-10s %s\n", "weap", "son", "bank", "derive")
	for _, l := range out {
		bank := "(aucune)"
		if l.bank != 0 {
			bank = fmt.Sprintf("%08x", l.bank)
		}
		fmt.Printf("%08x   %08x   %-10s +%d\n", l.weap, l.son, bank, l.derive)
	}
	return nil
}
