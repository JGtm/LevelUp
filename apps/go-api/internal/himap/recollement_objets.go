package himap

// recollement_objets.go — UN ELEMENT EST GARDE ENTIER OU RETIRE ENTIER.
//
// L'IDEE VIENT DE L'UTILISATEUR, le 2026-08-30, sur Dredge : « comme y a eu des elements coupes
// et y a du crenelage. Tu peux lisser ou ajuster les elements coupes partiellement pour les virer
// completement ? »
//
// LE DEFAUT EST STRUCTUREL, PAS UN REGLAGE MAL CHOISI. Le masque des positions jouees vit sur une
// GRILLE — celle du corpus, decimee au demi-metre, dilatee d'un rayon. Sa frontiere ne connait
// donc rien aux objets : elle coupe une passerelle en deux, laisse la moitie d'une caisse, et
// dessine ses bords en escalier parce qu'ils suivent la grille et non la matiere. Aucun reglage
// de rayon ne corrige cela : quel que soit le rayon, la frontiere reste celle d'une grille.
//
// CE QUE FAIT LE RECOLLEMENT : il remonte du pixel a l OBJET qui l a peint (`objetGagnant`), et
// decide par objet. Un element dont le masque ne garde pas au moins `seuil` de la surface est
// RETIRE EN ENTIER. La frontiere cesse alors de suivre la grille pour suivre la silhouette des
// objets — ce qui supprime du meme coup les elements coupes en deux ET le crenelage, qui sont
// les deux faces du meme defaut.
//
// IL NE RAJOUTE JAMAIS DE MATIERE, et c est le point qui a ete paye. Premiere version, le
// 2026-08-30 : un objet majoritairement garde etait COMPLETE, ses cellules manquantes rendues au
// masque. Resultat mesure a l image — les grandes dalles du canevas, effleurees par le masque
// sur un tiers de leur surface, revenaient ENTIERES et posaient d immenses rectangles gris bien
// au-dela de l arene. La carte etait moins propre qu avant le recollement. Le sens est donc
// UNIQUE : on retire, on ne complete pas. Un seuil eleve (0,9) veut alors dire « un objet doit
// etre presque entierement dans le terrain parcouru pour rester ».
//
// POURQUOI L'INSTANCE ET NON LE TYPE. Tous les murs d'un meme modele partagent leur type :
// raisonner par type traiterait ensemble des objets poses aux quatre coins de la carte, et un
// seul mur garde sauverait tous les autres. L'instance designe un objet POSE, et c'est la seule
// maille ou « garder entier » veut dire quelque chose.
//
// GARDE-FOU : sans provenance par instance (`ArmeObjetGagnant` non appelee), le recollement ne
// fait rien et rend le masque tel quel — jamais un masque vide.

// SeuilRecollement : part de la surface d'un objet que le masque doit garder pour que l'objet
// soit conserve. Trois quarts : un objet dont le masque laisse un quart dehors est encore lu
// comme entier a l'oeil ; au-dela, c'est un element coupe, et l'utilisateur demande qu'il parte.
const SeuilRecollement = 0.75

// recolleAuxObjets rend un masque de garde d ou les objets trop partiellement gardes ont ete
// entierement retires, et le nombre d objets retires. Le masque ne peut que RETRECIR.
//
// Les pixels sans objet connu (fond, matiere posee hors de la boucle Forge) gardent leur
// decision d origine : le recollement ne parle que de ce qu il sait nommer.
func (r *Rendu) recolleAuxObjets(garde []bool, seuil float64) (out []bool, retires int) {
	if r.objetGagnant == nil || seuil <= 0 {
		return garde, 0
	}
	type part struct{ gardes, total int }
	parts := map[int32]*part{}
	for k, o := range r.objetGagnant {
		if o == 0 {
			continue
		}
		p := parts[o]
		if p == nil {
			p = &part{}
			parts[o] = p
		}
		p.total++
		if garde[k] {
			p.gardes++
		}
	}
	retire := make(map[int32]bool, len(parts))
	for o, p := range parts {
		if p.total == 0 || p.gardes == 0 {
			continue
		}
		if float64(p.gardes)/float64(p.total) < seuil {
			retire[o] = true
			retires++
		}
	}
	out = make([]bool, len(garde))
	copy(out, garde)
	for k, o := range r.objetGagnant {
		if o != 0 && retire[o] {
			out[k] = false
		}
	}
	return out, retires
}
