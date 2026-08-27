package himap

// zones_forge.go — LES ZONES DE CALLOUT D'UNE CARTE FORGE.
//
// CE QUI ETAIT ECRIT ET QUI ETAIT FAUX. Nos en-tetes affirmaient depuis des semaines qu'une
// carte Forge n'a pas de callouts, et `masque_zones.go` en tirait la conclusion que le rognage
// aux zones nommees « ne vaudra jamais pour les cartes Forge ». La mesure du 2026-08-27 dit le
// contraire : chaque zone nommee est un OBJET de la variante, de type `TypeIDZoneNommee`, qui
// porte son StringId de lieu au chemin `#8/4[]/0/0` et sa forme Forge ordinaire a cote. Le
// fichier ne manquait pas — c'est le `map.mvar`, celui-la meme que la cuisson lit deja.
//
// Le zero reste vrai pour les CANEVAS : un canevas ne place pas de zone, il n'a que ses
// barrieres. C'est de la que venait la confusion.
//
// CE QUE CE FICHIER FOURNIT : la conversion des objets en polygones du monde, et le tri des
// RATELIERS (voir `ratelier`). Il ne decide pas du rognage — cette decision se prend carte par
// carte, sur la mesure, exactement comme sur la chaine native.

import (
	"math"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// TypeIDZoneNommee : le type d'objet « named location » des variantes Forge.
//
// ETABLI PAR EXHAUSTIVITE, pas par echantillon : sur les 4 161 objets porteurs d'un StringId de
// lieu dans les 257 variantes du dump, 4 161 portent ce type et aucun autre type n'en porte.
const TypeIDZoneNommee int32 = -696190206

// cotesCylindreZone : nombre de cotes du polygone qui approche un cylindre. 24 suffisent —
// l'erreur de corde vaut alors 0,9 % du rayon, tres en dessous du pixel du fond (0,05 a 0,2 m
// pour des rayons de plusieurs metres).
const cotesCylindreZone = 24

// ZoneNommee est une zone de callout posee sur la carte : son identite et son contour.
type ZoneNommee struct {
	// StringID est le condensat du nom, a resoudre contre le tableau du tag `locs`. Il n'est
	// PAS un texte : la traduction vit dans `callouts_i18n.csv`, qui n'en couvre encore qu'une
	// partie. Une zone dont le nom nous manque borne quand meme la carte — le rognage n'a
	// jamais eu besoin de savoir lire.
	StringID uint32
	// Contour est le polygone au sol, en coordonnees monde.
	Contour [][2]float64
	// ZBas, ZHaut : l'extension verticale de la zone, en absolu.
	ZBas, ZHaut float64
}

// ZonesNommeesForge extrait les zones de callout d'une variante deja decodee, et ECARTE les
// rateliers (voir `ratelier`). L'ordre du fichier est conserve : deux appels rendent la meme
// chose, ce qu'un rendu reproductible exige.
func ZonesNommeesForge(objs []mapvar.Object) []ZoneNommee {
	var out []ZoneNommee
	for _, o := range objs {
		if o.TypeID != TypeIDZoneNommee || o.LocationID == 0 {
			continue
		}
		sh := o.Shape()
		if sh == nil {
			continue
		}
		c := contourDeZone(o, sh)
		if len(c) < 3 {
			continue
		}
		out = append(out, ZoneNommee{
			StringID: o.LocationID,
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

// ContoursDeZones rend les seuls polygones, dans la forme qu'attend `MasqueZones`.
func ContoursDeZones(zs []ZoneNommee) [][][2]float64 {
	if len(zs) == 0 {
		return nil
	}
	out := make([][][2]float64, 0, len(zs))
	for _, z := range zs {
		out = append(out, z.Contour)
	}
	return out
}

// contourDeZone construit le polygone au sol d'une zone, ORIENTE par son vecteur avant.
//
// L'orientation n'est pas un raffinement : sur une zone tournee, un rectangle aligne sur les
// axes du monde declare « dedans » de larges coins qui sont dehors, et « dehors » des pans de
// la zone. La chaine des objectifs a paye cette lecon (31 % de faux positifs sur une zone
// d'Extraction tournee de 20 degres) — on ne la repaie pas ici.
func contourDeZone(o mapvar.Object, sh *mapvar.Shape) [][2]float64 {
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
		// (mesure : quelques zones du dump) retombe sur les axes du monde plutot que de
		// produire un polygone degenere.
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

// ratelier reconnait une PALETTE d'objets non poses. Deux variantes du dump en portent une :
// `illusion` aligne 57 boites de 0,5 m a abscisse constante, `forbidden` 33 boites de 1,0 m sur
// une grille a altitude constante, avec des noms venus de toute la franchise
// (« ridgeline icicle » sur une carte qui n'est pas Ridgeline). Ce ne sont pas des zones : les
// prendre pour telles rognerait la carte sur une droite.
//
// CRITERE : la quasi-totalite des zones partage une forme identique. C'est le signe d'une
// palette, jamais celui d'une carte — une carte donne a chaque lieu la taille de son lieu.
func ratelier(zs []ZoneNommee) bool {
	if len(zs) < 8 {
		return false // trop peu pour conclure, et une palette est toujours nombreuse
	}
	compte := map[[2]float64]int{}
	for _, z := range zs {
		compte[gabarit(z.Contour)]++
	}
	max := 0
	for _, n := range compte {
		if n > max {
			max = n
		}
	}
	return float64(max) >= partRatelierMin*float64(len(zs))
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
