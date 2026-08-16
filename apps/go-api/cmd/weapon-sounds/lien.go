package main

// lien.go — ETAPE 3 voie B : relier une arme (`weap`) a ses evenements Wwise via `snd!`.
//
// PROCESSUS SEPARE, VOLONTAIREMENT. Le mode `map` ouvre le module de 7,24 Go qui porte les
// `sbnk` ; ce mode-ci ouvre celui de 0,62 Go qui porte les `snd!` et les `weap`. Les deux
// ne doivent JAMAIS etre charges par le meme processus : `himodule.Open` lit le module
// entier en memoire. L'echange se fait donc par le JSON produit par `map`.
//
// Chaine : identifiants d'evenements (JSON) -> tags `snd!` qui les portent -> tags `weap`
// qui dependent de ces `snd!`. Aucun nom n'intervient.

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"

	"levelup/go-api/internal/himodule"
)

// relier resout arme -> `snd!` -> evenements a partir du rapport JSON du mode `map`.
func relier(cheminModule, cheminJSON string) error {
	rap, err := chargerRapport(cheminJSON)
	if err != nil {
		return err
	}
	cible := map[uint32]bool{}
	for _, e := range rap.Evenements {
		cible[e.IDEvent] = true
	}
	fmt.Printf("arme     : %s\n", rap.Arme)
	fmt.Printf("evenements a relier : %d\n", len(cible))

	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	// Les deux groupes de son : `snd!` (ponctuels) ET `lsnd` (boucles). Ne scanner que les
	// `snd!` a fait manquer le lien — les sons d'arme sont des `lsnd`.
	sonds := append(m.Files("snd!"), m.Files("lsnd")...)
	armes := m.Files("weap")
	fmt.Printf("module   : %d tags snd!+lsnd, %d tags weap\n\n", len(sonds), len(armes))

	porteurs := chercherPorteurs(m, sonds, cible)
	fmt.Printf("tags snd! portant un de nos evenements : %d\n", len(porteurs))
	rapporterMemoire("scan snd! termine")
	if len(porteurs) == 0 {
		fmt.Println("\nAUCUN LIEN : les snd! de ce module ne portent pas ces identifiants.")
		fmt.Println("Verifier un autre module (pc/globals, common) avant de conclure.")
		return nil
	}
	afficherPorteurs(porteurs)
	rattacherArmes(m, armes, porteurs)
	return nil
}

// sonderSnd cherche des identifiants ARBITRAIRES dans les tags `snd!` et disseque le
// premier porteur. Sert a repondre a « par quoi un snd! designe-t-il son son ? » quand
// l'identifiant d'evenement, lui, ne s'y trouve pas.
func sonderSnd(cheminModule string, ids []uint32) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")
	cible := map[uint32]bool{}
	for _, id := range ids {
		cible[id] = true
	}
	sonds := m.Files("snd!")
	fmt.Printf("module   : %d tags snd!\n", len(sonds))
	fmt.Printf("recherche: %v\n\n", ids)

	porteurs := chercherPorteurs(m, sonds, cible)
	fmt.Printf("tags snd! porteurs : %d\n", len(porteurs))
	if len(porteurs) == 0 {
		fmt.Println("aucun — cet identifiant n'est pas stocke en u32 dans les snd!")
		disséquerPremierSnd(m, sonds)
		return nil
	}
	afficherPorteurs(porteurs)
	return nil
}

// disséquerPremierSnd affiche la structure d'un `snd!` pour comprendre son adressage.
func disséquerPremierSnd(m *himodule.Module, sonds []himodule.File) {
	for _, f := range sonds {
		data, err := m.Extract(f)
		if err != nil || len(data) < tailleEnteteT {
			continue
		}
		deps := dependances(data)
		fmt.Printf("\n--- anatomie du snd! %08x (%d octets, %d dependances) ---\n",
			f.GlobalID, len(data), len(deps))
		groupes := map[string]int{}
		for _, d := range deps {
			groupes[d.Groupe]++
		}
		for g, n := range groupes {
			fmt.Printf("  depend de %d tag(s) '%s'\n", n, g)
		}
		fin := len(data)
		if fin > 160 {
			fin = 160
		}
		fmt.Printf("  octets apres l'en-tete+deps :\n")
		debut := tailleEnteteT + len(deps)*tailleDep
		if debut < fin {
			for o := debut; o < fin; o += 16 {
				bout := o + 16
				if bout > fin {
					bout = fin
				}
				fmt.Printf("    +%03x  % x\n", o, data[o:bout])
			}
		}
		return
	}
}

// histogrammeBanks rend, pour tous les `snd!` du module, les `sbnk` qu'ils referencent.
//
// A QUOI CA REPOND. Un `snd!` depend d'exactement un `sbnk` : l'ensemble de ces references
// EST la liste des banks atteignables depuis le systeme de tags. Si la bank d'une arme n'y
// figure pas, c'est que le lien arme -> son ne passe pas par ce module — le savoir evite de
// chercher un chainon qui n'existe pas ici.
func histogrammeBanks(cheminModule string, attendu uint32) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")
	sonds := m.Files("snd!")
	compte := map[uint32]int{}
	sansBank := 0
	for i, f := range sonds {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		vu := false
		for _, d := range dependances(data) {
			if d.Groupe == "sbnk" {
				compte[d.IDGlobal]++
				vu = true
			}
		}
		if !vu {
			sansBank++
		}
		if i%5000 == 0 && i > 0 {
			rapporterMemoire(fmt.Sprintf("snd! %d/%d", i, len(sonds)))
		}
	}
	fmt.Printf("\n%d tags snd! | %d sans dependance sbnk\n", len(sonds), sansBank)
	fmt.Printf("%d banks distinctes referencees\n\n", len(compte))

	type ligne struct {
		gid uint32
		n   int
	}
	var l []ligne
	for g, n := range compte {
		l = append(l, ligne{g, n})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	fmt.Println("--- banks les plus referencees ---")
	for i, x := range l {
		if i >= 15 {
			break
		}
		fmt.Printf("  sbnk %08x : %d snd!\n", x.gid, x.n)
	}
	if n, ok := compte[attendu]; ok {
		fmt.Printf("\nBANK ATTENDUE %08x : PRESENTE, %d snd! la referencent\n", attendu, n)
	} else {
		fmt.Printf("\nBANK ATTENDUE %08x : ABSENTE de ce module\n", attendu)
	}
	return nil
}

// quiRefere balaie TOUS les groupes de tags du module et rend ceux qui dependent de `gid`.
//
// C'est la question posee a l'envers : plutot que de supposer que le chemin passe par un
// `snd!`, on demande au graphe qui pointe reellement vers cette bank. Si personne ne
// pointe, le lien arme -> bank ne transite pas par ce module, et il faut le dire.
func quiRefere(cheminModule string, gid uint32) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")
	tous := m.Files("")
	fmt.Printf("module   : %d entrees\n", len(tous))
	fmt.Printf("recherche: qui depend du tag %08x ?\n\n", gid)

	parGroupe := map[string][]uint32{}
	for i, f := range tous {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		for _, d := range dependances(data) {
			if d.IDGlobal == gid {
				parGroupe[f.Group] = append(parGroupe[f.Group], f.GlobalID)
				break
			}
		}
		if i%20000 == 0 && i > 0 {
			rapporterMemoire(fmt.Sprintf("entrees %d/%d", i, len(tous)))
		}
	}
	if len(parGroupe) == 0 {
		fmt.Printf("AUCUN tag de ce module ne depend de %08x\n", gid)
		return nil
	}
	fmt.Println("--- referents, par groupe ---")
	for g, ids := range parGroupe {
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		fmt.Printf("  '%s' : %d tag(s)\n", g, len(ids))
		for _, id := range ids {
			fmt.Printf("      %08x  (decimal %d)\n", id, id)
		}
	}
	return nil
}

// remonter suit le graphe de dependances A L'ENVERS depuis un tag, niveau par niveau,
// jusqu'a atteindre un `weap` (ou epuiser le budget de niveaux).
//
// Un seul balayage du module par niveau : on cherche un ENSEMBLE d'identifiants a la fois,
// pas un seul. C'est ce qui rend la remontee praticable sur 78 174 entrees.
func remonter(cheminModule string, depart uint32, maxNiveaux int) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")
	tous := m.Files("")
	fmt.Printf("module   : %d entrees\n", len(tous))
	fmt.Printf("depart   : tag %08x\n\n", depart)

	courant := map[uint32]bool{depart: true}
	for niveau := 1; niveau <= maxNiveaux; niveau++ {
		parID := referents(m, tous, courant)
		if len(parID) == 0 {
			fmt.Printf("niveau %d : aucun referent, la remontee s'arrete\n", niveau)
			return nil
		}
		groupes := map[string]int{}
		for _, g := range parID {
			groupes[g]++
		}
		fmt.Printf("niveau %d : %d referent(s)", niveau, len(parID))
		for g, n := range groupes {
			fmt.Printf(" | '%s' x%d", g, n)
		}
		fmt.Println()

		suivant := map[uint32]bool{}
		var armes []uint32
		for id, g := range parID {
			suivant[id] = true
			if g == "weap" {
				armes = append(armes, id)
			}
		}
		if len(armes) > 0 {
			sort.Slice(armes, func(i, j int) bool { return armes[i] < armes[j] })
			fmt.Println("\n=== TAGS weap ATTEINTS ===")
			for _, id := range armes {
				fmt.Printf("  weap %08x\n", id)
			}
			return nil
		}
		courant = suivant
	}
	fmt.Printf("\nbudget de %d niveaux epuise sans atteindre un weap\n", maxNiveaux)
	return nil
}

// referents rend les tags qui dependent d'au moins un identifiant de `cible`,
// associes a leur groupe (fourCC).
func referents(m *himodule.Module, tous []himodule.File, cible map[uint32]bool) map[uint32]string {
	out := map[uint32]string{}
	for _, f := range tous {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		for _, d := range dependances(data) {
			if cible[d.IDGlobal] {
				out[f.GlobalID] = f.Group
				break
			}
		}
	}
	return out
}

// chargerRapport relit le JSON produit par le mode `map`.
func chargerRapport(chemin string) (rapportArme, error) {
	var rap rapportArme
	blob, err := os.ReadFile(chemin)
	if err != nil {
		return rap, err
	}
	if err := json.Unmarshal(blob, &rap); err != nil {
		return rap, err
	}
	if len(rap.Evenements) == 0 {
		return rap, fmt.Errorf("rapport sans evenement : %s", chemin)
	}
	return rap, nil
}

// chercherPorteurs rend, par tag `snd!`, les identifiants d'evenements qu'il contient.
func chercherPorteurs(m *himodule.Module, sonds []himodule.File, cible map[uint32]bool) map[uint32][]uint32 {
	aiguilles := make(map[uint32][4]byte, len(cible))
	for id := range cible {
		var a [4]byte
		binary.LittleEndian.PutUint32(a[:], id)
		aiguilles[id] = a
	}
	out := map[uint32][]uint32{}
	for i, f := range sonds {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		for id, a := range aiguilles {
			if bytes.Contains(data, a[:]) {
				out[f.GlobalID] = append(out[f.GlobalID], id)
			}
		}
		if i%4000 == 0 && i > 0 {
			rapporterMemoire(fmt.Sprintf("snd! %d/%d", i, len(sonds)))
		}
	}
	return out
}

// afficherPorteurs liste les `snd!` retenus, du plus couvrant au moins couvrant.
func afficherPorteurs(porteurs map[uint32][]uint32) {
	type ligne struct {
		gid    uint32
		events []uint32
	}
	var l []ligne
	for gid, ev := range porteurs {
		sort.Slice(ev, func(i, j int) bool { return ev[i] < ev[j] })
		l = append(l, ligne{gid, ev})
	}
	sort.Slice(l, func(i, j int) bool { return len(l[i].events) > len(l[j].events) })
	fmt.Println("\n--- tags snd! et evenements portes ---")
	for i, x := range l {
		if i >= 20 {
			fmt.Printf("  ... et %d autres\n", len(l)-20)
			break
		}
		fmt.Printf("  snd! %08x : %d evenement(s) %08x...\n", x.gid, len(x.events), x.events[0])
	}
}

// rattacherArmes cherche les `weap` qui dependent des `snd!` retenus.
func rattacherArmes(m *himodule.Module, armes []himodule.File, porteurs map[uint32][]uint32) {
	fmt.Println("\n--- tags weap dependant de ces snd! ---")
	trouve := 0
	for _, f := range armes {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		var touches []uint32
		for _, d := range dependances(data) {
			if d.Groupe == "snd!" && porteurs[d.IDGlobal] != nil {
				touches = append(touches, d.IDGlobal)
			}
		}
		if len(touches) == 0 {
			continue
		}
		trouve++
		fmt.Printf("  weap %08x depend de %d snd! de notre ensemble\n", f.GlobalID, len(touches))
	}
	if trouve == 0 {
		fmt.Println("  aucun — la reference weap -> snd! n'est pas directe (relais effe/effet probable)")
	}
	rapporterMemoire("scan weap termine")
}

// rapporterMemoire affiche l'empreinte courante : ces modules sont gros, on veut le voir.
func rapporterMemoire(etape string) {
	var st runtime.MemStats
	runtime.ReadMemStats(&st)
	fmt.Printf("[memoire] %-22s alloue %5.1f Go | systeme %5.1f Go\n",
		etape, float64(st.Alloc)/(1<<30), float64(st.Sys)/(1<<30))
}
