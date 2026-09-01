package hinavmesh

// typetable.go — LA TABLE DES TYPES D'UN FICHIER-TAG (sections TNA1 et TBDY).
//
// TNA1 : [n] puis n-1 entrees {indice de nom, nb de modeles, (indice de nom, valeur)*}.
// L'entree 0 est le type nul. Un parametre de modele dont le nom commence par 't' porte
// un indice de TYPE, un nom commencant par 'v' porte une VALEUR.
//
// TBDY : suite d'entrees {indice de type, indice du parent, drapeaux, champs optionnels
// selon les drapeaux}. Les drapeaux commandent, dans cet ordre : 0x01 format, 0x02
// sous-type, 0x04 version, 0x08 taille + alignement, 0x10 champ inutilise ici, 0x20 la
// table des membres {nom, drapeaux, offset, type}, 0x40 la table des interfaces.
//
// ENTIERS EMPAQUETES. Tous les nombres de TNA1 et TBDY sont a longueur variable, prefixe
// en tete de premier octet. Les deux formes rencontrees sur les cartes mesurees sont la
// forme a 1 octet et la forme a 2 octets ; les formes a 3 et 4 octets sont implementees
// et exercees (le format `int` = 0x8204 est encode sur 3 octets). Toute autre forme
// remonte une ERREUR : on ne saute jamais un octet en silence, un decalage d'un octet
// produirait une table de types coherente en apparence et fausse.
//
// PREUVE DU DECODAGE (Isolation) : TBDY se consomme a l'octet pres (808/808), et le
// format lu pour `int` vaut 0x8204 la ou `unsigned char` vaut 0x2004 — soit, dans les
// deux cas, (nb de bits << 10) | drapeau signe | 4 (genre entier). Deux lectures
// independantes qui concordent.

import (
	"fmt"
)

// Drapeaux d'un corps de type dans TBDY.
const (
	drapeauFormat     = 0x01
	drapeauSousType   = 0x02
	drapeauVersion    = 0x04
	drapeauTaille     = 0x08
	drapeauInconnu    = 0x10
	drapeauMembres    = 0x20
	drapeauInterfaces = 0x40
	drapeauReserve    = 0x80
)

// lecteurEmpaquete lit les entiers a longueur variable de TNA1 et TBDY.
type lecteurEmpaquete struct {
	buf []byte
	pos int
	err error
}

// entier lit un entier empaquete. Apres une erreur, rend 0 sans avancer : l'appelant
// verifie err une fois, en fin de parcours.
func (l *lecteurEmpaquete) entier() int {
	if l.err != nil {
		return 0
	}
	if l.pos >= len(l.buf) {
		l.err = fmt.Errorf("hinavmesh: fin de section a l'offset %d", l.pos)
		return 0
	}
	b0 := l.buf[l.pos]
	var longueur, valeur int
	switch {
	case b0&0x80 == 0x00:
		longueur, valeur = 1, int(b0)
	case b0&0xC0 == 0x80:
		longueur = 2
	case b0&0xE0 == 0xC0:
		longueur = 3
	case b0&0xF0 == 0xE0:
		longueur = 4
	default:
		l.err = fmt.Errorf("hinavmesh: forme d'entier empaquete inconnue 0x%02x a l'offset %d", b0, l.pos)
		return 0
	}
	if l.pos+longueur > len(l.buf) {
		l.err = fmt.Errorf("hinavmesh: entier empaquete de %d octets tronque a l'offset %d", longueur, l.pos)
		return 0
	}
	if longueur > 1 {
		masque := []int{0, 0x7F, 0x3F, 0x1F, 0x0F}[longueur]
		valeur = int(b0) & masque
		for i := 1; i < longueur; i++ {
			valeur = valeur<<8 | int(l.buf[l.pos+i])
		}
	}
	l.pos += longueur
	return valeur
}

// fini indique si le reste de la section n'est que du remplissage nul. Les sections sont
// alignees : TNA1 laisse un octet de queue sur Isolation (203 octets utiles sur 204).
func (l *lecteurEmpaquete) fini() bool {
	for _, b := range l.buf[l.pos:] {
		if b != 0 {
			return false
		}
	}
	return true
}

// sectionsChaines : les deux ecritures possibles des tables de chaines d un fichier-tag Havok.
//
// DEUX GENERATIONS, MESUREES LE 2026-08-30. Les blobs d Isolation et consorts portent TST1 et
// FST1 ; ceux d Absolution et d Insolence portent TSTR et FSTR — memes voisins (TPTR, TNA1,
// TBDY, THSH, TPAD), meme place dans la section TYPE, seul le nom des deux tables de chaines
// change. Le decodeur refusait donc trois cartes pour un nom de section, pas pour un format :
// le contenu se lit de la meme facon, chaines nul-terminees a la queue leu leu.
//
// C est ce qui condamnait Absolution, Insolence et Insolence Heavies a la bouillie — sans
// maillage lisible, rien ne fait tomber leurs coques.
//
// UNE SECONDE DIFFERENCE, TROUVEE EN SUIVANT LA PREMIERE : les tables TSTR/FSTR sont indexees a
// partir de 1, la chaine vide occupant l indice 0 sans etre stockee. Sur Absolution, un membre
// demandait l indice 98 d une table qui en portait 98 (0 a 97) — l ecart d exactement un, sur la
// borne haute, est la signature d un decalage d origine et non d une table tronquee.
var sectionsChaines = [][2]string{
	{"TST1", "FST1"},
	{"TSTR", "FSTR"},
}

// lireTypes construit la table des types a partir des tables de chaines, de TNA1 et de TBDY.
func lireTypes(buf []byte, sections map[string][2]int) (tableTypes, int, error) {
	var secTypes, secChamps [2]int
	trouve := false
	for _, paire := range sectionsChaines {
		t, okT := sections[paire[0]]
		f, okF := sections[paire[1]]
		if okT && okF {
			secTypes, secChamps, trouve = t, f, true
			break
		}
	}
	if !trouve {
		return nil, 0, fmt.Errorf("hinavmesh: fichier-tag sans table de chaines (ni TST1/FST1, ni TSTR/FSTR)")
	}
	for _, tag := range []string{"TNA1", "TBDY"} {
		if _, ok := sections[tag]; !ok {
			return nil, 0, fmt.Errorf("hinavmesh: fichier-tag sans section %s", tag)
		}
	}
	nomsTypes := chaines(buf, secTypes)
	nomsChamps := chaines(buf, secChamps)

	types, err := lireNomsTypes(buf, sections["TNA1"], nomsTypes)
	if err != nil {
		return nil, 0, err
	}
	recuperations, err := lireCorpsTypes(buf, sections["TBDY"], types, nomsChamps)
	if err != nil {
		return nil, 0, err
	}
	return types, recuperations, nil
}

// lireNomsTypes decode TNA1 : nom et parametres de modele de chaque type.
func lireNomsTypes(buf []byte, sec [2]int, nomsTypes []string) (tableTypes, error) {
	l := &lecteurEmpaquete{buf: buf[sec[0] : sec[0]+sec[1]]}
	n := l.entier()
	if l.err != nil {
		return nil, l.err
	}
	if n <= 0 || n > sec[1] {
		return nil, fmt.Errorf("hinavmesh: TNA1 annonce %d types pour %d octets", n, sec[1])
	}
	nom := func(i int) (string, error) {
		if i < 0 || i >= len(nomsTypes) {
			return "", fmt.Errorf("hinavmesh: indice de nom %d hors des %d chaines de la table des types", i, len(nomsTypes))
		}
		return nomsTypes[i], nil
	}
	types := make(tableTypes, n)
	for i := 1; i < n; i++ {
		s, err := nom(l.entier())
		if err != nil {
			return nil, err
		}
		types[i].Nom = s
		nbModeles := l.entier()
		for j := 0; j < nbModeles && l.err == nil; j++ {
			cle, err := nom(l.entier())
			if err != nil {
				return nil, err
			}
			if types[i].Modeles == nil {
				types[i].Modeles = map[string]int{}
			}
			types[i].Modeles[cle] = l.entier()
		}
		if l.err != nil {
			return nil, fmt.Errorf("hinavmesh: TNA1, type %d: %w", i, l.err)
		}
	}
	if !l.fini() {
		return nil, fmt.Errorf("hinavmesh: TNA1 laisse %d octets non nuls apres %d types", sec[1]-l.pos, n)
	}
	return types, nil
}

// lireCorpsTypes decode TBDY : taille, parent et membres de chaque type.
func lireCorpsTypes(buf []byte, sec [2]int, types tableTypes, nomsChamps []string) (int, error) {
	corps := buf[sec[0] : sec[0]+sec[1]]
	l := &lecteurEmpaquete{buf: corps}
	recuperations := 0
	for l.err == nil && !l.fini() {
		debut := l.pos
		idx := l.entier()
		if l.err != nil {
			break
		}
		if idx <= 0 || idx >= len(types) {
			return recuperations, fmt.Errorf("hinavmesh: TBDY, indice de type %d hors des %d types (offset %d)",
				idx, len(types), l.pos)
		}
		avant := *(&types[idx])
		if err := lireUnCorps(l, &types[idx], nomsChamps); err != nil {
			// L entree est illisible. On ne devine pas son contenu : on cherche le plus court
			// saut apres lequel TOUT le reste de la section se lit (resynchronisation.go). Sans
			// un tel saut, l erreur remonte comme avant.
			reprise, ok := resynchronise(corps, debut, types, nomsChamps)
			if !ok {
				return recuperations, erreurEntreeIllisible(idx, types[idx].Nom, debut, err)
			}
			types[idx] = avant // le type reste OPAQUE : aucun membre invente
			types[idx].Opaque = true
			l.pos, l.err = reprise, nil
			recuperations++
			continue
		}
	}
	if l.err != nil {
		return recuperations, fmt.Errorf("hinavmesh: TBDY: %w", l.err)
	}
	return recuperations, nil
}

// lireUnCorps decode une entree de TBDY, drapeau par drapeau.
func lireUnCorps(l *lecteurEmpaquete, t *typeHavok, nomsChamps []string) error {
	t.Parent = l.entier()
	drapeaux := l.entier()
	if drapeaux&drapeauReserve != 0 {
		return fmt.Errorf("drapeau 0x80 inconnu (drapeaux 0x%02x)", drapeaux)
	}
	for _, d := range []int{drapeauFormat, drapeauSousType, drapeauVersion} {
		if drapeaux&d != 0 {
			l.entier()
		}
	}
	if drapeaux&drapeauTaille != 0 {
		t.Taille = l.entier()
		l.entier() // alignement
	}
	if drapeaux&drapeauInconnu != 0 {
		l.entier()
	}
	if drapeaux&drapeauMembres != 0 {
		nb := l.entier()
		for j := 0; j < nb && l.err == nil; j++ {
			iNom := l.entier()
			l.entier() // drapeaux du membre
			offset := l.entier()
			typ := l.entier()
			if l.err != nil {
				break
			}
			if iNom < 0 || iNom >= len(nomsChamps) {
				return fmt.Errorf("membre %d: indice de nom %d hors des %d chaines de la table des noms de champs",
					j, iNom, len(nomsChamps))
			}
			t.Membres = append(t.Membres, membreHavok{Nom: nomsChamps[iNom], Offset: offset, Type: typ})
		}
	}
	if drapeaux&drapeauInterfaces != 0 {
		nb := l.entier()
		for j := 0; j < nb && l.err == nil; j++ {
			l.entier() // type de l'interface
			l.entier() // drapeaux de l'interface
		}
	}
	return l.err
}
