// Package geo derive les cinq variables GEOMETRIQUES d'une position de force, a partir de
// la seule forme de la carte — sans un seul match.
//
// POURQUOI UNE VOIE GEOMETRIQUE. La voie empirique (paquet parent `powerpos`) a rendu
// NO-GO contre l'oracle des guides pro le 2026-09-20 : le rapport de duel retrouve les
// bases et les points de defense, pas les lieux que les equipes cherchent a TENIR. Elle a
// deux angles morts structurels — nos matchs ne sont pas joues par des professionnels, et
// une carte sans corpus n'a aucun signal. La geometrie n'a ni l'un ni l'autre : elle dit ce
// que la carte PERMET, indifferemment du niveau de ceux qui y jouent.
//
// LES CINQ VARIABLES (formulation de la pre-analyse du 2026-09-20, plan etape 2bis.C ; c'est
// aussi la facon dont un moteur de jeu evalue une position — les requetes d'environnement
// d'Unreal) :
//
//	H  altitude relative   altitude du noeud moins l'altitude moyenne de son voisinage
//	V  visibilite sortante fraction du terrain que l'on voit depuis ce noeud
//	E  exposition          ouverture ANGULAIRE des directions d'ou l'on peut etre vu
//	R  ressources          proximite, EN DISTANCE DE DEPLACEMENT, des armes et objectifs
//	M  echappatoire        proximite d'un couvert ou l'on cesse d'etre vu par ses guetteurs
//
//	SCORE = w1*H + w2*V - w3*E + w4*R + w5*M
//
// DEUX VARIABLES SONT DES PROXIMITES, PAS DES DISTANCES, et c'est le seul moyen que la
// formule ci-dessus ait le bon signe : R et M valent 1 quand la ressource (ou le couvert)
// est SOUS LE PIED et 0 quand elle est au bout de la carte. Une position de force est
// proche de son arme et proche de son repli, pas loin.
//
// V ET E SORTENT DU MEME CALCUL, et c'est voulu. La visibilite entre deux paires d'yeux a
// la meme hauteur est SYMETRIQUE : l'ensemble de ce qu'on voit est l'ensemble de ceux qui
// nous voient. V le compte (combien), E le disperse (depuis combien de directions
// differentes). Une bonne position voit beaucoup et n'est vue que d'un secteur : les deux
// agregations du meme ensemble ne se confondent donc pas, mais elles ne sont pas non plus
// independantes — le reglage doit le savoir.
//
// # CE QUE CE PAQUET NE SAIT PAS
//
//   - LE MODELE D'OCCLUSION EST UNE GRILLE DE VOXELS, pas la soupe de triangles. Un rayon
//     est bloque des qu'il traverse un voxel occupe : un garde-corps ajoure bloque comme un
//     mur plein, une lucarne de moins d'un voxel n'existe pas. C'est le compromis qui rend
//     des millions de rayons calculables ; il SURESTIME l'occlusion, donc il sous-estime V
//     et E de facon homogene sur toute la carte.
//   - LE SOL PRATICABLE EST DERIVE, PAS DECLARE. Les cartes natives du studio n'ont pas de
//     maillage de navigation publie (cf. `hinavmesh` : `navmesh.blob` n'existe que pour les
//     cartes Forge). Le sol est donc reconstruit : surface horizontale + hauteur libre
//     au-dessus. Une rampe trop raide, une caisse trop haute, un rebord d'un demi-metre
//     entrent ou sortent selon des seuils, et ces seuils sont des constantes nommees.
//   - LA DISTANCE DE DEPLACEMENT EST CELLE DU GRAPHE DE CE SOL, 8-connexe, avec une marche
//     maximale. Elle ignore le saut, le grappin, le rebond — donc elle SURESTIME les
//     distances la ou un joueur prend un raccourci vertical.
//
// Le paquet est PUR : il ne lit aucun fichier, n'ouvre aucune base, ne connait ni carte ni
// titre. Les triangles, les ancres et les ressources lui sont donnes.
package geo
