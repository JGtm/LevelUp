package replay

// document_structure.go — LES DEUX TYPES DU FOND DE CARTE : l'emprise au sol d'un element de
// structure, et le prop Forge projete en 2D.
//
// EXTRAITS DE `document.go` le 2026-08-18 (plan `.ai/V7.5/replay2d/PLAN_DRAPEAU_OBJET.md`,
// phase 2). Le fichier hote etait EXACTEMENT au seuil de 500 lignes du depot : la chronique du
// schema 15 ne pouvait pas y entrer sans le franchir. Ces deux types-la sont ceux qui partent, et
// pas au hasard — ils ne decrivent pas le REJEU (des joueurs, des tirs, des objets qui bougent)
// mais la CARTE sur laquelle il se joue, ils sont remplis par une chaine a part (la geometrie des
// `.module`, cf. structure.go) et rien dans `document.go` ne les lit. La chronique du schema,
// elle, reste chez le document : c'est la version de l'artefact ENTIER qu'elle date.

// Surface est l'emprise au sol d'un élément de structure : la projection sur (x, y) de
// l'AABB MONDE d'une instance de géométrie instanciée, plus les altitudes de ses faces
// haute et basse. Rectangle aligné sur les axes — PAS le maillage : le lien instance ->
// géométrie n'est pas résolu (layout du champ meshRef inconnu), on publie donc la boîte
// englobante et rien de plus. Une boîte de plateforme ou de mur suffit à une carte
// reconnaissable en vue de dessus ; elle ne rend pas les formes courbes.
//
// CE QUE LE CHAMP GARANTIT : Z est l'altitude de la face SUPÉRIEURE, celle sur laquelle un
// joueur se tient. Mesure : sur le film 000d5950, 80,6 % des positions à vitesse verticale
// quasi nulle sont à moins de 5 cm au-dessus de la surface la plus haute sous elles (11,9 %
// attendus par hasard, 37,5 % pour le témoin le plus sévère — altitudes permutées entre
// emprises), écart médian 8 mm.
type Surface struct {
	X0 float32 `json:"x0"`
	Y0 float32 `json:"y0"`
	X1 float32 `json:"x1"`
	Y1 float32 `json:"y1"`
	// Z est l'altitude de la face supérieure de la boîte (le « sol » de l'élément).
	Z float32 `json:"z"`
	// ZB est l'altitude de la face inférieure. PIÈGE : pas d'omitempty ici — une face à
	// exactement 0 serait omise et relue comme « au niveau de la mer », ce qui déplacerait
	// l'élément d'un étage.
	ZB float32 `json:"zb"`
	// Poly est l'emprise ORIENTÉE, quand elle est connue : 4 à 8 sommets XY dans le repère
	// monde. Absente, le client retombe sur le rectangle X0/Y0/X1/Y1.
	//
	// POURQUOI : X0..Y1 est une boîte alignée sur les axes du MONDE, alors que l'instance
	// porte sa propre base (forward, left, up). Pour une instance tournée, l'AABB déborde
	// largement de la pièce réelle. En prenant la boîte orientée, la surface totale de la
	// carte tombe de 47,4 %.
	//
	// CE QUE ÇA NE FAIT PAS, et il faut le savoir avant de s'en réjouir : l'emprise orientée
	// ne CREUSE rien. Sur la zone « Fer à cheval », le vide passe de 0,00 m² à 0,00 m² —
	// les neuf instances qui en bouchent le centre sont à yaw nul et échelle unité, donc leur
	// boîte orientée est identique à leur boîte alignée. Un anneau vit dans les TRIANGLES du
	// maillage ; aucune boîte, même exacte, ne le rendra.
	Poly [][2]float32 `json:"poly,omitempty"`
}

// Area renvoie l'aire au sol de l'emprise, en m².
func (s Surface) Area() float32 {
	w, h := s.X1-s.X0, s.Y1-s.Y0
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h
}

// MapObject est un prop Forge projeté en 2D : centre orienté + emprise de sa bounding box.
// Ce sont de PETITS objets (0,25 m² en moyenne) — décor et repères, pas les sols/murs.
type MapObject struct {
	// TypeID est l'identifiant global du tag Forge (permet un style par famille d'objet).
	TypeID int64 `json:"typeId"`
	// X, Y sont le centre au sol ; Z l'altitude (indication d'étage).
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z,omitempty"`
	// DX, DY sont l'emprise (largeur/profondeur) du modèle, avant rotation.
	DX float32 `json:"dx,omitempty"`
	DY float32 `json:"dy,omitempty"`
	// Yaw est la rotation autour de la verticale, en degrés.
	Yaw float32 `json:"yaw,omitempty"`
}
