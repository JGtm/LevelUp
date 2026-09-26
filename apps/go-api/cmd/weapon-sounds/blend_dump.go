package main

// blend_dump.go — le mode `blend` : LA TABLE DE FONDU D'UN CONTENEUR, courbe par courbe.
//
// POURQUOI CE MODE EXISTE. Le parcours d'evenement evalue un `Blend` AU POINT DE REFERENCE
// (le x le plus a gauche de chaque courbe, donc le parametre de jeu a sa valeur minimale) et
// laisse tomber les enfants qui y sont inaudibles. C'est une decision assumee et datee — le
// rejeu 2D ne pilote aucun parametre de jeu. Mais elle a un effet de bord qui vient d'etre
// mesure : les DEUX PLUS LONGS sons de la banque du translocateur (6,77 s et 6,22 s) sont
// exactement ces enfants-la. Ils pendent sous le `Blend 33f7ed7c` de l'evenement `388207de`,
// et le mode `orphelins` le dit sur pieces.
//
// CE QUE CA CHANGE POUR LA RECONSTITUTION. Un enfant inaudible a x minimal et audible plus
// loin n'est pas un son mort : c'est une MONTEE. L'utilisateur decrit le geste du
// translocateur ainsi — « c'est comme si on le chargeait, ca monte en intensite, et ensuite
// il est pose ». Une couche pilotee par un parametre croissant EST cette montee. Pour la
// rendre, il faut lire la courbe : a quel x l'enfant entre, a quel x il domine, et avec
// quelle interpolation.
//
// Ce mode ne rend rien : il MONTRE la table. Le rendu se decide apres, sur pieces.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/himodule"
)

// dumperBlend est le mode `blend`. `cibles` vide = tous les conteneurs `Blend` de la banque.
// SANS `-sbnk`, le mode bascule en RECENSEMENT : quels parametres de jeu pilotent des fondus,
// et sur combien de banques. Un parametre present dans des centaines de banques n'est pas
// specifique a un objet — c'est un parametre GLOBAL (la distance en est le candidat evident),
// et cela change la lecture d'une phase : « loin/pres » n'est pas « charge/pose ».
func dumperBlend(cheminModule string, gidBank uint32, cibles map[uint32]bool) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	if gidBank == 0 {
		return recenserRTPC(m)
	}
	b, err := banqueParGid(m, gidBank)
	if err != nil {
		return err
	}
	connu := func(id uint32) bool { _, ok := b.Objets[id]; return ok }

	ids := make([]uint32, 0, len(b.Objets))
	for id, o := range b.Objets {
		if o.Type == typeBlend && (len(cibles) == 0 || cibles[id]) {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	fmt.Printf("\nbanque %08x : %d conteneur(s) Blend a montrer\n", gidBank, len(ids))

	for _, id := range ids {
		o := b.Objets[id]
		c := lireBlend(o.Data, connu)
		enfants := lireEnfants(o.Data, connu)
		fmt.Printf("\n=== Blend %08x : %d enfant(s) declare(s), table %s ===\n",
			id, len(enfants), lisible(c.Lu))
		for _, e := range enfants {
			fmt.Printf("  enfant %08x %s%s\n", e, nomType(b.Objets[e].Type), decrireSon(b, e))
		}
		if !c.Lu {
			continue
		}
		for _, l := range c.Cies {
			fmt.Printf("  -- couche %08x, parametre de jeu (RTPC) %d --\n", l.ID, l.RTPC)
			for _, a := range l.Assocs {
				if len(a.C) == 0 {
					fmt.Printf("     enfant %08x : aucune courbe, joue tel quel\n", a.Enfant)
					continue
				}
				fmt.Printf("     enfant %08x%s :", a.Enfant, decrireSon(b, a.Enfant))
				for _, p := range a.C {
					fmt.Printf("  (x=%g y=%g i=%d)", p.X, p.Y, p.Interp)
				}
				fmt.Println()
			}
		}
		aud := c.Audibles(enfants)
		fmt.Print("  AU POINT DE REFERENCE (x minimal), enfants retenus :")
		var gardes []string
		for _, e := range enfants {
			if g, ok := aud[e]; ok {
				gardes = append(gardes, fmt.Sprintf(" %08x(%+.1f dB)", e, g))
			}
		}
		if len(gardes) == 0 {
			fmt.Print(" aucun")
		}
		for _, g := range gardes {
			fmt.Print(g)
		}
		fmt.Println()
	}
	return nil
}

// recenserRTPC compte, sur tout un module, quels parametres de jeu pilotent des fondus.
func recenserRTPC(m *himodule.Module) error {
	parRTPC := map[uint32]int{}
	banquesParRTPC := map[uint32]map[uint32]bool{}
	var blends, banques int
	for _, f := range m.Files("sbnk") {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		debut := indexBKHD(data)
		if debut < 0 {
			continue
		}
		b, err := parserBank(data[debut:], func(uint32) bool { return false })
		if err != nil {
			continue
		}
		banques++
		for _, c := range b.Blends {
			blends++
			for _, l := range c.Cies {
				if l.RTPC == 0 {
					continue
				}
				parRTPC[l.RTPC]++
				if banquesParRTPC[l.RTPC] == nil {
					banquesParRTPC[l.RTPC] = map[uint32]bool{}
				}
				banquesParRTPC[l.RTPC][f.GlobalID] = true
			}
		}
	}
	fmt.Printf("\n=== RECENSEMENT DES PARAMETRES DE FONDU ===\n")
	fmt.Printf("  banques lues : %d, conteneurs Blend : %d, parametres distincts : %d\n",
		banques, blends, len(parRTPC))
	ids := make([]uint32, 0, len(parRTPC))
	for id := range parRTPC {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return parRTPC[ids[i]] > parRTPC[ids[j]] })
	for i, id := range ids {
		if i >= 20 {
			fmt.Printf("  ... et %d autres\n", len(ids)-20)
			break
		}
		fmt.Printf("  parametre %10d : %5d couche(s) sur %4d banque(s)\n",
			id, parRTPC[id], len(banquesParRTPC[id]))
	}
	return nil
}

// decrireSon rend, pour un noeud, le media qu'il porte quand il en porte un — c'est ce qui
// permet de reconnaitre a l'oeil l'enfant que le point de reference ecarte.
func decrireSon(b *bank, n uint32) string {
	if w, ok := b.Sons[n]; ok {
		return fmt.Sprintf(" -> .wem %d", w)
	}
	var wems []uint32
	vus := map[uint32]bool{}
	var descendre func(uint32)
	descendre = func(x uint32) {
		if vus[x] {
			return
		}
		vus[x] = true
		if w, ok := b.Sons[x]; ok {
			wems = append(wems, w)
			return
		}
		for _, e := range lireEnfants(b.Objets[x].Data, func(id uint32) bool { _, k := b.Objets[id]; return k }) {
			descendre(e)
		}
	}
	descendre(n)
	if len(wems) == 0 {
		return ""
	}
	sort.Slice(wems, func(i, j int) bool { return wems[i] < wems[j] })
	return fmt.Sprintf(" -> %v", wems)
}

// banqueParGid extrait et parse une banque du module.
func banqueParGid(m *himodule.Module, gid uint32) (*bank, error) {
	for _, f := range m.Files("sbnk") {
		if f.GlobalID != gid {
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			return nil, err
		}
		debut := indexBKHD(data)
		if debut < 0 {
			return nil, fmt.Errorf("banque %08x : chunk BKHD absent", gid)
		}
		return parserBank(data[debut:], func(uint32) bool { return false })
	}
	return nil, fmt.Errorf("banque %08x introuvable", gid)
}

func lisible(b bool) string {
	if b {
		return "LUE"
	}
	return "NON LUE"
}
