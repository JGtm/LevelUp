package main

// conteneurs.go — INVENTAIRE DES CONTENEURS, et decodage du conteneur `Switch`.
//
// POURQUOI CE FICHIER EXISTE. Le mode `audit` recensait les chunks, les types d'objets, les
// types d'action et la charge utile des `Sound`. Il ne disait rien des CONTENEURS — et c'est
// la que se cachait le quatrieme oubli de format du chantier : un conteneur `Switch` etait
// traite comme un conteneur aleatoire. Or les deux n'ont pas le meme sens :
//
//   - `RandomSequence` tire une variante au hasard : ses enfants sont INTERCHANGEABLES ;
//   - `Switch` choisit selon un ETAT DE JEU (distance, materiau, nombre d'aiguilles
//     plantees) : ses enfants ne le sont PAS, et il peut ne rien jouer si aucun etat ne
//     correspond.
//
// Mesure qui a motive ce travail : 31 coups reconstitues sur 107 portent une couche
// `Switch`, dont 28 en 1re personne, et elle y pese jusqu'a 71 % du melange.
//
// PRINCIPE INCHANGE : ne rien postuler sur le layout. La liste d'enfants est deja localisee
// par `lireEnfants` ; on lit autour d'elle et on VALIDE — les enfants cites par les paquets
// d'etats doivent appartenir a la liste d'enfants deja resolue.

import "encoding/binary"

// paquetSwitch : les enfants a jouer pour un etat donne d'un conteneur `Switch`.
type paquetSwitch struct {
	Etat    uint32
	Enfants []uint32
}

// conteneurSwitch : ce qu'on sait lire d'un conteneur `Switch`.
type conteneurSwitch struct {
	Lu         bool
	Groupe     uint32 // identifiant du groupe de commutation (Switch ou State)
	ParEtat    bool   // vrai si le groupe est un Switch, faux si c'est un State
	EtatDefaut uint32
	Paquets    []paquetSwitch
}

// EnfantsDe rend les enfants a jouer pour un etat ; a defaut, ceux de l'etat par defaut.
// Rend nil si aucun etat ne correspond : un `Switch` sans etat valide ne joue RIEN, et
// c'est une information, pas un echec.
func (c conteneurSwitch) EnfantsDe(etat uint32) []uint32 {
	for _, p := range c.Paquets {
		if p.Etat == etat {
			return p.Enfants
		}
	}
	return nil
}

// EnfantsParDefaut rend les enfants de l'etat par defaut declare par le conteneur.
func (c conteneurSwitch) EnfantsParDefaut() []uint32 {
	if e := c.EnfantsDe(c.EtatDefaut); len(e) > 0 {
		return e
	}
	// L'etat par defaut peut ne porter aucun enfant : dans ce cas le conteneur reste muet
	// tant que le jeu n'impose pas d'etat. On ne substitue rien.
	return nil
}

// positionEnfants retrouve l'offset de la liste d'enfants dans la charge utile.
//
// `lireEnfants` rend la liste mais pas sa position, et c'est la position qui sert d'ancre :
// tout ce qui precede est l'en-tete du conteneur, tout ce qui suit est le reste. On refait
// donc le meme balayage en gardant l'offset de la meilleure liste.
func positionEnfants(d []byte, connu func(uint32) bool) (int, int) {
	meilleurOff, meilleurN := -1, 0
	for off := 0; off+4 <= len(d); off++ {
		n := int(binary.LittleEndian.Uint32(d[off:]))
		if n < 1 || n > 512 || off+4+4*n > len(d) || n <= meilleurN {
			continue
		}
		ok := true
		for i := 0; i < n; i++ {
			if !connu(binary.LittleEndian.Uint32(d[off+4+4*i:])) {
				ok = false
				break
			}
		}
		if ok {
			meilleurOff, meilleurN = off, n
		}
	}
	return meilleurOff, meilleurN
}

// tailleEnTeteSwitch : octets separant le debut de l'en-tete de commutation du debut de la
// liste d'enfants, dans `AkSwitchCntrInitialValues` :
//
//	u8 eGroupType | u32 ulGroupID | u32 ulDefaultSwitch | u8 bIsContinuousValidation
//
// La valeur n'est PAS postulee : elle est verifiee par `lireSwitch`, qui n'accepte le
// resultat que si l'etat par defaut se retrouve parmi les etats reellement declares.
const tailleEnTeteSwitch = 10

// lireSwitch decode un conteneur `Switch` a partir de sa charge utile.
//
// Structure visee, apres la liste d'enfants :
//
//	u32 ulNumSwitchGroups
//	  pour chacun : u32 switchID | u32 ulNumItems | ulNumItems x u32 idEnfant
//
// VALIDATION, dans cet ordre — un seul echec fait rendre `Lu: false` plutot qu'une
// approximation : les enfants cites doivent tous appartenir a la liste d'enfants deja
// resolue, et l'etat par defaut doit figurer parmi les etats declares.
func lireSwitch(d []byte, connu func(uint32) bool) conteneurSwitch {
	off, n := positionEnfants(d, connu)
	if off < 0 {
		return conteneurSwitch{}
	}
	enfants := map[uint32]bool{}
	for i := 0; i < n; i++ {
		enfants[binary.LittleEndian.Uint32(d[off+4+4*i:])] = true
	}
	paquets, ok := lirePaquetsSwitch(d, off+4+4*n, enfants)
	if !ok {
		return conteneurSwitch{}
	}
	c := conteneurSwitch{Lu: true, Paquets: paquets}
	if off >= tailleEnTeteSwitch {
		tete := off - tailleEnTeteSwitch
		c.ParEtat = d[tete] != 0
		c.Groupe = binary.LittleEndian.Uint32(d[tete+1:])
		c.EtatDefaut = binary.LittleEndian.Uint32(d[tete+5:])
	}
	// L'en-tete n'est retenu que s'il se recoupe avec les paquets : sinon on garde les
	// paquets, qui sont valides par leurs enfants, et on avoue ne pas savoir l'etat.
	if c.EtatDefaut != 0 && len(c.EnfantsDe(c.EtatDefaut)) == 0 {
		c.Groupe, c.EtatDefaut, c.ParEtat = 0, 0, false
	}
	return c
}

// lirePaquetsSwitch lit la table etat -> enfants qui suit la liste d'enfants.
func lirePaquetsSwitch(d []byte, off int, enfants map[uint32]bool) ([]paquetSwitch, bool) {
	if off+4 > len(d) {
		return nil, false
	}
	nb := int(binary.LittleEndian.Uint32(d[off:]))
	if nb < 1 || nb > 256 {
		return nil, false
	}
	off += 4
	out := make([]paquetSwitch, 0, nb)
	for i := 0; i < nb; i++ {
		if off+8 > len(d) {
			return nil, false
		}
		etat := binary.LittleEndian.Uint32(d[off:])
		nItems := int(binary.LittleEndian.Uint32(d[off+4:]))
		off += 8
		if nItems < 0 || nItems > 512 || off+4*nItems > len(d) {
			return nil, false
		}
		ids := make([]uint32, 0, nItems)
		for j := 0; j < nItems; j++ {
			id := binary.LittleEndian.Uint32(d[off+4*j:])
			// Un enfant cite hors de la liste d'enfants du conteneur = mauvais offset.
			if !enfants[id] {
				return nil, false
			}
			ids = append(ids, id)
		}
		off += 4 * nItems
		out = append(out, paquetSwitch{Etat: etat, Enfants: ids})
	}
	return out, true
}
