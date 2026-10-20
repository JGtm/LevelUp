package main

// meleefx.go — le coup de corps a corps designe par des champs NOMMES, pas par un critere.
//
// LE PROBLEME QUE CECI RESOUT. Pour les armes de melee, la chaine de tags ne designait
// jusqu'ici aucun evenement : le champ « melee sound » du niveau racine pointe vers une bank
// de melee GENERIQUE, commune, pas vers l'evenement de cette arme. Faute de designation, le
// rendu departageait 34 evenements candidats par un critere degenere — `max(couches, wem)` —
// qui, a egalite parfaite, rendait le premier du JSON. C'est ce qui a donne a l'epee infectee
// une reference de 25,71 secondes.
//
// CE QUE `weap.xml` DIT VRAIMENT. Le tableau nomme « melee damage parameters » porte, par
// element, treize references de tag, dont deux nous interessent directement :
//
//	_40 "melee damage parameters"        <- tableau
//	    _41 "melee attack effect"        <- le coup PORTE (balayage)
//	    _41 "biped melee hit effect"     <- l'IMPACT sur un biped
//
// Cette paire repond a DEUX questions d'un coup : quel evenement est le coup de melee de
// CETTE arme, et la distinction coup touche / coup manque — un coup qui rate ne joue que
// l'attaque, sans l'effet d'impact.
//
// PRINCIPE : on ne postule pas la cible. La sonde RAPPORTE le groupe de tag vise, elle ne
// filtre pas dessus. C'est l'inventaire qui dit ce que ces champs designent reellement.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/himodule"
)

// champsDuTableau rend les champs d'un `_40`, avec leurs offsets DANS un element, et la
// taille d'un element. Les offsets d'un tableau repartent de zero : ses champs vivent dans
// un bloc de donnees separe, pas dans l'espace d'offsets du parent.
func champsDuTableau(nom string) ([]champ, int, int, bool) {
	champs, err := champsPlugin()
	if err != nil {
		return nil, 0, 0, false
	}
	tab, ok := trouverChamp(champs, nom)
	if !ok || !tab.tableau {
		return nil, 0, 0, false
	}
	var sous []champ
	taille := parcourir(tab.noeud, 0, &sous)
	return sous, taille, tab.off, true
}

// refsMelee : ce qu'un element du tableau « melee damage parameters » designe.
type refsMelee struct {
	Attaque, Impact uint32
	GroupeAttaque   string
	GroupeImpact    string
}

// meleeDuTag lit les references d'effet de chaque element du tableau, pour un `weap` donne.
//
// DERIVE : les offsets du plugin sont decales de +64 au niveau RACINE (mesure sur deux
// champs independants). Rien ne dit que la meme derive s'applique DANS un element de
// tableau — on essaie donc les decalages connus au niveau racine seulement, et on prend
// l'element tel quel ensuite. Le controle est la taille : si le bloc n'est pas un multiple
// de la taille d'element calculee, l'appariement est refuse.
func meleeDuTag(data []byte, groupes map[uint32]string) ([]refsMelee, bool) {
	sous, tailleElem, offTab, ok := champsDuTableau("melee damage parameters")
	if !ok || tailleElem <= 0 {
		return nil, false
	}
	cAtt, okA := trouverChamp(sous, "melee attack effect")
	cImp, okI := trouverChamp(sous, "biped melee hit effect")
	if !okA || !okI {
		return nil, false
	}
	t, err := ouvrirTagWeap(data)
	if err != nil {
		return nil, false
	}
	racine, err := t.blocRacine()
	if err != nil {
		return nil, false
	}
	nRefs := compterRefs(sous)
	for o, bloc := range t.enfantsDe(racine) {
		if o < offTab || o > offTab+128 {
			continue
		}
		abs, taille := t.blocAbs(bloc)
		if abs < 0 || taille == 0 {
			continue
		}
		elem, ok := tailleElementReelle(taille, tailleElem, nRefs)
		if !ok {
			continue
		}
		// L'EN-TETE D'ELEMENT DIFFERE DU PLUGIN. Mesure sur l'epee : le bloc fait 396
		// octets la ou le plugin annonce 392, et la suite des 13 references de 28 octets
		// commence a +32 et non a +28. Les references sont donc adressees depuis la FIN
		// du bloc, ou le plugin et le build s'accordent, plutot que depuis le debut.
		debut := elem - nRefs*tailleRef
		return lireElementsMelee(data, abs, taille/elem, elem, debut, cAtt, cImp, sous, groupes), true
	}
	return nil, false
}

// compterRefs compte les references de tag d'un element : c'est le seul repere commun au
// plugin et au build, l'en-tete ayant grossi entre les deux.
func compterRefs(sous []champ) int {
	n := 0
	for _, c := range sous {
		if c.noeud.XMLName.Local == "_41" {
			n++
		}
	}
	return n
}

// tailleElementReelle cherche la taille d'element qui divise le bloc, au plus pres de celle
// annoncee par le plugin, et qui laisse la place aux references.
func tailleElementReelle(taille, annonce, nRefs int) (int, bool) {
	mini := nRefs * tailleRef
	for ecart := 0; ecart <= 64; ecart += 4 {
		for _, e := range []int{annonce + ecart, annonce - ecart} {
			if e >= mini && e > 0 && taille%e == 0 {
				return e, true
			}
		}
	}
	return 0, false
}

// lireElementsMelee lit chaque element. Les deux champs vises sont adresses par leur RANG
// dans la suite de references, deduit du plugin, et non par leur offset annonce.
func lireElementsMelee(data []byte, abs, nb, tailleElem, debut int, cAtt, cImp champ,
	sous []champ, groupes map[uint32]string) []refsMelee {
	rgAtt, rgImp := rangDeRef(sous, cAtt.nom), rangDeRef(sous, cImp.nom)
	out := make([]refsMelee, 0, nb)
	for k := 0; k < nb; k++ {
		base := abs + k*tailleElem + debut
		r := refsMelee{
			Attaque: gidRef(data, base+rgAtt*tailleRef),
			Impact:  gidRef(data, base+rgImp*tailleRef),
		}
		r.GroupeAttaque = groupeOuVide(groupes, r.Attaque)
		r.GroupeImpact = groupeOuVide(groupes, r.Impact)
		out = append(out, r)
	}
	return out
}

// rangDeRef rend l'indice d'une reference parmi les references de l'element.
func rangDeRef(sous []champ, nom string) int {
	n := 0
	for _, c := range sous {
		if c.noeud.XMLName.Local != "_41" {
			continue
		}
		if c.nom == nom {
			return n
		}
		n++
	}
	return -1
}

func groupeOuVide(groupes map[uint32]string, gid uint32) string {
	if gid == 0 || gid == 0xffffffff {
		return ""
	}
	if g, ok := groupes[gid]; ok {
		return g
	}
	return "?"
}

// effetsDeMelee parcourt les `weap` et rapporte ce que designent les deux champs.
func effetsDeMelee(cheminModule string, gidVoulu uint32) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")
	groupes := make(map[uint32]string, 1<<17)
	for _, f := range m.Files("") {
		groupes[f.GlobalID] = f.Group
	}

	type ligne struct {
		weap uint32
		refs []refsMelee
	}
	var out []ligne
	paires := map[string]int{}
	for _, f := range m.Files("weap") {
		if gidVoulu != 0 && f.GlobalID != gidVoulu {
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		refs, ok := meleeDuTag(data, groupes)
		if !ok || len(refs) == 0 {
			// MONTRER LA STRUCTURE plutot que de deviner l'offset. Deux echecs de lecture
			// de suite sur ce chantier (`Blend`, puis ce tableau) ont ete resolus en une
			// lecture des donnees reelles, apres plusieurs hypotheses infructueuses.
			if gidVoulu != 0 && len(out) == 0 {
				diagnostiquerTableau(data, groupes)
			}
			continue
		}
		for _, r := range refs {
			paires[r.GroupeAttaque+" / "+r.GroupeImpact]++
		}
		out = append(out, ligne{f.GlobalID, refs})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].weap < out[j].weap })

	fmt.Printf("\n%d tag(s) weap avec un tableau « melee damage parameters » lisible\n", len(out))
	fmt.Println("\n=== VERS QUOI POINTENT CES CHAMPS (attaque / impact) ===")
	cles := make([]string, 0, len(paires))
	for k := range paires {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return paires[cles[i]] > paires[cles[j]] })
	for _, k := range cles {
		fmt.Printf("  %-24s %5d element(s)\n", k, paires[k])
	}
	fmt.Println("\n=== DETAIL ===")
	fmt.Printf("%-10s %-4s %-12s %-12s\n", "weap", "elem", "attaque", "impact")
	for _, l := range out {
		for i, r := range l.refs {
			fmt.Printf("%08x   %-4d %-12s %-12s\n", l.weap, i,
				refCourte(r.Attaque, r.GroupeAttaque), refCourte(r.Impact, r.GroupeImpact))
		}
	}
	return nil
}

func refCourte(gid uint32, groupe string) string {
	if groupe == "" {
		return "-"
	}
	return fmt.Sprintf("%08x %s", gid, groupe)
}

// diagnostiquerTableau affiche ce que le plugin ANNONCE en regard de ce que le tag CONTIENT.
// C'est le pas qu'on a omis deux fois : quand un appariement echoue, il faut voir les deux
// cotes, pas essayer un offset de plus.
func diagnostiquerTableau(data []byte, groupes map[uint32]string) {
	sous, tailleElem, offTab, ok := champsDuTableau("melee damage parameters")
	if !ok {
		fmt.Println("diagnostic : le tableau n'est meme pas dans le plugin")
		return
	}
	fmt.Printf("\n--- diagnostic ---\nplugin : offset du tableau %d, taille d'element %d\n",
		offTab, tailleElem)
	for _, c := range sous {
		if c.nom == "melee attack effect" || c.nom == "biped melee hit effect" {
			fmt.Printf("  champ « %s » a +%d dans l'element\n", c.nom, c.off)
		}
	}
	t, err := ouvrirTagWeap(data)
	if err != nil {
		fmt.Println("  tag illisible :", err)
		return
	}
	racine, err := t.blocRacine()
	if err != nil {
		fmt.Println("  racine introuvable :", err)
		return
	}
	enfants := t.enfantsDe(racine)
	offs := make([]int, 0, len(enfants))
	for o := range enfants {
		offs = append(offs, o)
	}
	sort.Ints(offs)
	// Le bloc le plus proche de l'offset annonce, vers l'avant : la derive du plugin est
	// positive et modeste. On SCANNE ensuite ce bloc au lieu de deviner la taille d'element.
	for _, o := range offs {
		if o < offTab || o > offTab+128 {
			continue
		}
		abs, taille := t.blocAbs(enfants[o])
		if abs < 0 || taille == 0 {
			continue
		}
		fmt.Printf("  bloc retenu : +%d (derive +%d), %d octets, element annonce %d\n",
			o, o-offTab, taille, tailleElem)
		scannerRefs(data, abs, taille, groupes)
		return
	}
	fmt.Println("  aucun bloc dans la fenetre de derive")
}

// scannerRefs lit un `_41` a chaque position alignee et rapporte celles qui portent un
// identifiant non nul. Une suite de references espacees de 28 octets revele le debut REEL
// de la table, sans avoir a deviner quel champ du plugin a change de taille.
func scannerRefs(data []byte, abs, taille int, groupes map[uint32]string) {
	var trouves []int
	for o := 0; o+28 <= taille; o += 4 {
		// VALIDER contre l'ensemble des tags du module : une valeur qui n'est pas un tag
		// est un flottant ou du remplissage, pas une reference.
		if g := groupeOuVide(groupes, gidRef(data, abs+o)); g != "" && g != "?" {
			trouves = append(trouves, o)
		}
	}
	fmt.Printf("  %d positions pointant vers un TAG REEL :\n", len(trouves))
	for _, o := range trouves {
		gid := gidRef(data, abs+o)
		fmt.Printf("    +%-4d %08x  %s\n", o, gid, groupes[gid])
	}
}
