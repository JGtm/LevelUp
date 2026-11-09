package himap

// altitude_proche.go — NE GARDER QUE CE QUI EST AU NIVEAU DE JEU.
//
// L'IDEE VIENT DE L'UTILISATEUR, le 2026-08-30 : « tu peux pas prendre les bords blancs, garder
// une petite marge et couper ce qui est a l'exterieur ? » Elle est juste, et elle s'appuie sur
// une propriete que l'habillage rend visible sans qu'on l'ait cherchee.
//
// EN HABILLAGE ENCRE, LA CLARTE EST UN THERMOMETRE D'ALTITUDE. La teinte vaut
// `(0,30 + 0,70 x eclairement) x (1 - 0,45 x |dz| / PorteeNiveauDeJeu)` : plus une surface
// s'ecarte du niveau de jeu — au-dessus comme en dessous — plus elle fonce. Le sol joue est donc
// clair, et les masses sombres qui entourent Shogun et Smallhalla sont, PAR CONSTRUCTION, loin du
// niveau de jeu. Ce que l'oeil lit comme « les bords blancs » est la mesure `|z - niveauJeu|`.
//
// CE QUE CE FICHIER FAIT, ET EN QUOI IL DIFFERE DES LEVIERS DEJA REFUTES. Il travaille sur la
// surface DEJA DESSINEE, pas sur la geometrie en amont :
//
//   - la TRANCHE DE RENDU ecarte la geometrie avant projection, a une hauteur absolue : essayee
//     a +8 m sur Shogun, elle a rendu 0 ancre sur 13 parce qu'elle a coupe le sol lui-meme ;
//   - l'EXCLUSION PAR TYPE retire un modele partout : essayee sur les deux types dominants, elle
//     a rendu 0/13 aussi — ces types SONT le sol ;
//   - ici on garde les cellules proches du niveau de jeu, on DILATE ce masque d'une marge, et on
//     efface le reste. Le sol ne peut pas disparaitre : il est, par definition, au niveau de jeu.
//
// LA MARGE N'EST PAS UN DETAIL. Un mur, un rebord, une rampe bordent le sol sans etre a son
// altitude ; sans marge, la carte perdrait ses contours et ne se lirait plus. La dilatation les
// rattrape en gardant tout ce qui touche le terrain de pres.

import "math"

// SeuilAltitudeProche : ecart maximal au niveau de jeu, en metres, pour qu'une cellule soit tenue
// pour du terrain. 6 m couvrent un etage et les reliefs d'une arene ; au-dela on est sur une
// falaise, un rocher ou une dalle de ciel.
const SeuilAltitudeProche = 6.0

// MargeAltitudeProche : dilatation du masque, en metres. 4 m ≈ la largeur d'un couloir — assez
// pour garder le mur qui borde le sol, assez peu pour ne pas ramener la masse d'a cote.
const MargeAltitudeProche = 4.0

// RogneAuxAltitudesProches efface la matiere dont l'altitude s'ecarte de plus de `seuil` metres
// du niveau de jeu, SAUF si elle tombe dans la dilatation de `marge` autour de ce qui reste.
// Rend le nombre de cellules effacees.
//
// Le niveau de jeu doit avoir ete pose (`Rendu.NiveauDeJeu`) : sans lui il n'y a pas de reference
// et la fonction ne fait rien plutot que de couper au hasard.
func (r *Rendu) RogneAuxAltitudesProches(seuil, marge float64) int {
	if math.IsNaN(r.niveauJeu) || seuil <= 0 {
		return 0
	}
	proche := make([]bool, len(r.z))
	for k := range r.z {
		if math.IsInf(r.z[k], -1) {
			continue
		}
		if math.Abs(r.z[k]-r.niveauJeu) <= seuil {
			proche[k] = true
		}
	}
	garde := proche
	if marge > 0 && r.Cell > 0 {
		if n := int(marge/r.Cell + 0.5); n > 0 {
			garde = dilate(proche, r.NX, r.NY, n)
		}
	}
	efface := 0
	for k := range r.z {
		if garde[k] {
			continue
		}
		vide := math.IsInf(r.z[k], -1)
		if vide && (r.solSuppose == nil || !r.solSuppose[k]) {
			continue
		}
		r.z[k] = math.Inf(-1)
		if r.solSuppose != nil {
			r.solSuppose[k] = false
		}
		efface++
	}
	return efface
}
