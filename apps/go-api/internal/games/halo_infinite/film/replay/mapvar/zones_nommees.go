package mapvar

// zones_nommees.go — LES ZONES DE CALLOUT D'UNE CARTE FORGE, lues dans les objets de sa
// variante.
//
// Chaque zone nommee est un OBJET de la variante, de type `TypeIDZoneNommee`, qui porte son
// StringId de lieu (`Object.LocationID`) et sa forme Forge ordinaire a cote. Un CANEVAS n'en
// pose aucune : il n'a que ses barrieres. Recherche et mesures :
// `.ai/V7.5/cartes/CALLOUTS_FORGE_2026-08-27.md`.
//
// CE QUE CE FICHIER FOURNIT : la conversion des objets en polygones du monde, et le tri des
// RATELIERS (voir `ratelier`). Il ne nomme rien (le libelle se joint par StringId, au
// catalogue) et ne decide d'aucun rognage de fond de carte.
//
// POURQUOI ICI, A COTE DES SOCLES ET DES POINTS D'APPARITION : c'est la meme nature de
// lecture — des objets de la variante vers un ensemble semantique de la carte — et elle doit
// tourner AU RUNTIME (rattrapage des zones au fetch de film) comme dans les chaines de
// fabrication. Ce paquet est pur Go ; `internal/himap`, qui lit les fichiers du jeu, ne l'est
// pas (decodeur Oodle en cgo) et ne peut donc pas entrer dans le serveur.

import "math"

// TypeIDZoneNommee : le type d'objet « named location » des variantes Forge.
//
// ETABLI PAR EXHAUSTIVITE, pas par echantillon : sur les 4 161 objets porteurs d'un StringId de
// lieu dans les 257 variantes du dump de recherche, 4 161 portent ce type et aucun autre type
// n'en porte.
const TypeIDZoneNommee int32 = -696190206

// cotesCylindreZone : nombre de cotes du polygone qui approche un cylindre. 24 suffisent —
// l'erreur de corde vaut alors 0,9 % du rayon, tres en dessous du pixel du fond (0,05 a 0,2 m
// pour des rayons de plusieurs metres).
const cotesCylindreZone = 24

// ZoneNommee est une zone de callout posee sur la carte : son identite et son contour.
type ZoneNommee struct {
	// Index est le rang de l'objet dans la variante — la SEULE cle de tracabilite d'une zone
	// Forge (elle n'a pas d'indice de volume `levl`). C'est elle que le catalogue publie sous
	// `volume_index`, et elle qui permet de retrouver l'objet dans un dump de variante.
	Index int
	// StringID est le condensat du nom, a resoudre contre le lexique des noms de lieu. Il n'est
	// PAS un texte. Une zone dont le nom manque borne quand meme la carte.
	StringID uint32
	// Pos est le point de reference de l'objet, en metres monde. C'est le centre 3D dont le
	// rejeu se sert pour affecter un joueur a sa zone (distance 3D, sinon deux etages
	// superposes se confondent). Le contour seul ne le donne pas : il est plat.
	Pos [3]float64
	// Contour est le polygone au sol, en coordonnees monde.
	Contour [][2]float64
	// ZBas, ZHaut : l'extension verticale de la zone, en absolu.
	ZBas, ZHaut float64
}

// ZonesNommeesForge extrait les zones de callout d'une variante deja decodee, et ECARTE les
// rateliers (voir `ratelier`). L'ordre du fichier est conserve : deux appels rendent la meme
// chose, ce qu'un rendu reproductible exige.
func ZonesNommeesForge(objs []Object) []ZoneNommee {
	var out []ZoneNommee
	for _, o := range objs {
		if o.TypeID != TypeIDZoneNommee || o.LocationID == 0 {
			continue
		}
		sh := o.Shape()
		if sh == nil || !prismeLisible(sh) {
			continue
		}
		c := contourDeZone(o, sh)
		if len(c) < 3 {
			continue
		}
		out = append(out, ZoneNommee{
			Index:    o.Index,
			StringID: o.LocationID,
			Pos:      [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z},
			Contour:  c,
			ZBas:     o.Pos.Z - sh.DownZ,
			ZHaut:    o.Pos.Z + sh.UpZ,
		})
	}
	if ratelier(out) {
		return nil
	}
	return out
}

// prismeLisible refuse les enregistrements de forme ABERRANTS : un prisme dont la base passe
// au-dessus du sommet (emplacement 8 a -56,00 m en virgule fixe 16.16 sur une poignee de zones
// du corpus, emplacement 7 parfois absent).
//
// POURQUOI ON REFUSE AU LIEU DE REDRESSER. Les emplacements 5 a 8 sont lus a la file : si 7
// manque ou si 8 porte une valeur impossible, rien ne garantit que 5 et 6 — la largeur et la
// profondeur, donc LE POLYGONE — soient encore a leur place. Redresser la hauteur publierait
// une empreinte au sol dont on ne sait plus si elle est la bonne.
func prismeLisible(sh *Shape) bool { return sh.UpZ+sh.DownZ > 0 }

// contourDeZone construit le polygone au sol d'une zone, ORIENTE par son vecteur avant.
//
// L'orientation n'est pas un raffinement : sur une zone tournee, un rectangle aligne sur les
// axes du monde declare « dedans » de larges coins qui sont dehors, et « dehors » des pans de
// la zone.
func contourDeZone(o Object, sh *Shape) [][2]float64 {
	cx, cy := o.Pos.X, o.Pos.Y
	switch {
	case sh.Radius != nil:
		r := *sh.Radius
		if r <= 0 {
			return nil
		}
		p := make([][2]float64, 0, cotesCylindreZone)
		for i := 0; i < cotesCylindreZone; i++ {
			a := 2 * math.Pi * float64(i) / cotesCylindreZone
			p = append(p, [2]float64{cx + r*math.Cos(a), cy + r*math.Sin(a)})
		}
		return p
	case sh.HalfX != nil && sh.HalfY != nil:
		hx, hy := *sh.HalfX, *sh.HalfY
		if hx <= 0 || hy <= 0 {
			return nil
		}
		// Base orthonormee du plan, prise sur le vecteur avant. Un avant nul ou vertical
		// retombe sur les axes du monde plutot que de produire un polygone degenere.
		fx, fy := o.Forward.X, o.Forward.Y
		if n := math.Hypot(fx, fy); n > 1e-6 {
			fx, fy = fx/n, fy/n
		} else {
			fx, fy = 1, 0
		}
		px, py := -fy, fx
		return [][2]float64{
			{cx - fx*hx - px*hy, cy - fy*hx - py*hy},
			{cx + fx*hx - px*hy, cy + fy*hx - py*hy},
			{cx + fx*hx + px*hy, cy + fy*hx + py*hy},
			{cx - fx*hx + px*hy, cy - fy*hx + py*hy},
		}
	}
	return nil
}

// partRatelierMin : part des zones qui doivent partager la MEME forme pour qu'on parle de
// ratelier. 90 % laisse passer une carte qui pose sciemment des zones jumelles (symetrie
// d'arene) tout en attrapant les palettes, qui sont uniformes a 97-100 %.
const partRatelierMin = 0.9

// ratelierZonesMin : en dessous, trop peu de zones pour conclure — une palette est toujours
// nombreuse.
const ratelierZonesMin = 8

// ratelier reconnait une PALETTE d'objets non poses : des boites identiques alignees ou en
// grille, portant des noms venus de toute la franchise. Ce ne sont pas des zones : les prendre
// pour telles rognerait la carte sur une droite.
//
// CRITERE : la quasi-totalite des zones partage une forme identique. C'est le signe d'une
// palette, jamais celui d'une carte — une carte donne a chaque lieu la taille de son lieu.
func ratelier(zs []ZoneNommee) bool {
	if len(zs) < ratelierZonesMin {
		return false
	}
	compte := map[[2]float64]int{}
	for _, z := range zs {
		compte[gabarit(z.Contour)]++
	}
	plusFrequent := 0
	for _, n := range compte {
		if n > plusFrequent {
			plusFrequent = n
		}
	}
	return float64(plusFrequent) >= partRatelierMin*float64(len(zs))
}

// gabarit rend la taille du contour, arrondie au decimetre : deux zones de meme gabarit ont la
// meme forme, ou qu'elles soient posees.
func gabarit(c [][2]float64) [2]float64 {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range c {
		minX, maxX = math.Min(minX, p[0]), math.Max(maxX, p[0])
		minY, maxY = math.Min(minY, p[1]), math.Max(maxY, p[1])
	}
	return [2]float64{math.Round((maxX-minX)*10) / 10, math.Round((maxY-minY)*10) / 10}
}
