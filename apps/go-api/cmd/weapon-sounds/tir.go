package main

// tir.go — mode `tir` : rend les tags de son de TIR d'une arme, par le champ nomme.
//
// Chemin suivi, tout entier tire de `weap.xml` (aucun nom devine) :
//
//	bloc racine -> tableau « Weapon Fire Sounds » -> pour chaque element, tableau
//	« Weapon Fire Sound Variations » (offset de champ 12) -> champ `_41`
//	« Weapon Fire Sound » (offset 0) -> identifiant du tag de son.
//
// Sont rendus au passage les autres sons NOMMES du meme bloc (« Dry Fire Sound »,
// « Bullet Shell Sound ») : ils servent de contre-epreuve — s'ils tombaient sur les memes
// identifiants que le tir, l'appariement serait faux.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/himodule"
)

// offsets internes d'un element de « Weapon Fire Sounds », calcules depuis le plugin.
type offsetsTir struct {
	fireSounds int // offset du tableau dans le bloc racine
	variations int // offset du tableau dans un element de Weapon Fire Sounds
	shell      int // offset de « Bullet Shell Sound » dans ce meme element
	dryFire    int // offset de « Dry Fire Sound » dans ce meme element
}

// calculerOffsets lit `weap.xml` et rend les offsets du chemin de tir.
func calculerOffsets() (offsetsTir, error) {
	var o offsetsTir
	racine, err := champsPlugin()
	if err != nil {
		return o, err
	}
	bloc, ok := trouverChamp(racine, "Weapon Fire Sounds")
	if !ok {
		return o, fmt.Errorf("champ « Weapon Fire Sounds » absent du plugin")
	}
	o.fireSounds = bloc.off

	var interne []champ
	parcourir(bloc.noeud, 0, &interne)
	for _, c := range interne {
		switch c.nom {
		case "Weapon Fire Sound Variations":
			o.variations = c.off
		case "Bullet Shell Sound":
			o.shell = c.off
		case "Dry Fire Sound":
			o.dryFire = c.off
		}
	}
	if o.variations == 0 && o.fireSounds == 0 {
		return o, fmt.Errorf("offsets non resolus")
	}
	return o, nil
}

// sonsDeTir resout, pour un `weap` donne, ses tags de son de tir.
func sonsDeTir(cheminModule string, gidWeap uint32) error {
	o, err := calculerOffsets()
	if err != nil {
		return err
	}
	fmt.Printf("offsets plugin : Weapon Fire Sounds +%d | Variations +%d | Shell +%d | DryFire +%d\n\n",
		o.fireSounds, o.variations, o.shell, o.dryFire)

	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	var cible himodule.File
	trouve := false
	for _, f := range m.Files("weap") {
		if f.GlobalID == gidWeap {
			cible, trouve = f, true
			break
		}
	}
	if !trouve {
		return fmt.Errorf("tag weap %08x absent de ce module", gidWeap)
	}
	data, err := m.Extract(cible)
	if err != nil {
		return err
	}
	// Groupe de chaque tag du module, lu dans la TABLE D'ENTREES : aucune decompression,
	// donc gratuit. Sert a valider qu'une reference pointe bien vers un tag de son.
	groupes := make(map[uint32]string, 1<<17)
	for _, f := range m.Files("") {
		groupes[f.GlobalID] = f.Group
	}
	return decrireTir(data, gidWeap, o, groupes)
}

const tailleRef = 28

// decrireTir identifie le tableau « Weapon Fire Sounds » PAR SON CONTENU.
//
// POURQUOI PAS PAR OFFSET. Le plugin place ce tableau a +4288, offset qu'AUCUN tableau du
// tag reel n'occupe : la definition et le build du jeu ont derive. Le meme piege est
// documente dans `weapon-icons-build/weapui.go`, ou l'appariement par rang rendait un bloc
// decale. On identifie donc par une signature structurelle, verifiable :
//
//	un element de « Weapon Fire Sounds » porte, a l'offset de champ « Variations »,
//	un sous-tableau dont CHAQUE entree de 28 octets reference un tag de son.
//
// Si aucun bloc ne satisfait ce critere, la sonde le DIT au lieu de rendre un chiffre faux.
func decrireTir(data []byte, gidWeap uint32, o offsetsTir, groupes map[uint32]string) error {
	t, err := ouvrirTagWeap(data)
	if err != nil {
		return err
	}
	racine, err := t.blocRacine()
	if err != nil {
		return err
	}
	enfants := t.enfantsDe(racine)
	fmt.Printf("weap %08x : %d tableaux dans le bloc racine\n", gidWeap, len(enfants))
	if _, ok := enfants[o.fireSounds]; !ok {
		fmt.Printf("  NOTE : le plugin annonce +%d, aucun tableau ne s'y trouve -> identification par contenu\n",
			o.fireSounds)
	}

	type trouvaille struct {
		offChamp int
		gids     []uint32
	}
	var trouves []trouvaille
	for offChamp, bloc := range enfants {
		for _, blocVar := range t.enfantsDe(bloc) {
			absV, tailleV := t.blocAbs(blocVar)
			if absV < 0 || tailleV < tailleRef {
				continue
			}
			gids, tousSons := refsDeSons(data, absV, tailleV, groupes)
			if tousSons && len(gids) > 0 {
				trouves = append(trouves, trouvaille{offChamp, gids})
			}
		}
	}
	sort.Slice(trouves, func(i, j int) bool { return trouves[i].offChamp < trouves[j].offChamp })

	fmt.Printf("\n=== SONS DE TIR (champ « Weapon Fire Sound ») ===\n")
	if len(trouves) == 0 {
		fmt.Println("  aucun tableau ne presente la signature attendue — rien affirme")
		return nil
	}
	for _, tr := range trouves {
		fmt.Printf("  tableau a l'offset de champ +%d : %d variation(s)\n", tr.offChamp, len(tr.gids))
		for _, g := range tr.gids {
			fmt.Printf("      son %08x  (decimal %d, groupe '%s')\n", g, g, groupes[g])
		}
	}
	return nil
}

// refsDeSons lit les references d'un sous-tableau et dit si TOUTES pointent vers un son.
func refsDeSons(data []byte, abs, taille int, groupes map[uint32]string) ([]uint32, bool) {
	var gids []uint32
	for i := 0; (i+1)*tailleRef <= taille; i++ {
		g := gidRef(data, abs+i*tailleRef)
		if g == 0 || g == 0xffffffff {
			continue
		}
		grp := groupes[g]
		if grp != "lsnd" && grp != "snd!" {
			return nil, false
		}
		gids = append(gids, g)
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })
	return gids, true
}

// offsetsTries rend les offsets de champ des tableaux d'un bloc, tries.
func offsetsTries(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
