package mapdecoupe

// decoupe.go — l'intersection d'un pavé de callout avec le masque praticable.
//
// LA CHAÎNE, en quatre temps : rastériser le pavé sur la grille EXACTE du fond (pas de
// re-échantillonnage, donc pas de décalage), croiser avec le masque, re-vectoriser en
// contours et trous, simplifier sous la cellule. La sortie est au format que le rendu
// accepte DÉJÀ (contour principal + parties + trous, règle pair-impair) — ce plan ne change
// que la PRODUCTION.

import (
	"math"
	"sort"
)

// Options règle le découpage.
type Options struct {
	// SimplifieM : tolérance de simplification du contour, en mètres. Doit rester SOUS la
	// cellule du fond : au-delà, la simplification déplacerait la frontière de plus que ce
	// que la mesure sait résoudre.
	SimplifieM float64
	// AireMinM2 : sous ce seuil, la zone découpée est DÉGÉNÉRÉE. L'appelant garde alors le
	// brut et la SIGNALE — une zone nommée qui disparaît dit que le masque a un trou là,
	// pas que la zone n'existe pas.
	AireMinM2 float64
	// RendLesEnclaves : ne retirer que le vide qui COMMUNIQUE avec le dehors de la zone.
	// Voir reprendLesEnclaves — l'instrument mesure les deux lectures, la production n'en
	// sert qu'une.
	RendLesEnclaves bool
}

// SimplifieParDefaut : 0,08 m, soit 0,87 cellule du fond publié (0,092 m).
//
// Sous la cellule par construction — la borne du plan. Assez pour effacer les marches d'un
// escalier de rastérisation le long d'une droite, trop peu pour lisser un vrai décrochement.
const SimplifieParDefaut = 0.08

// AireMinParDefaut : 1 m². Un carré d'un mètre de côté est plus petit qu'un Spartan au sol ;
// une zone nommée réduite à cela n'est plus une zone, c'est un artefact.
const AireMinParDefaut = 1.0

// ToleranceParDefaut : rayon de la fermeture appliquée au masque avant découpage, en mètres.
//
// CE QUE CE NOMBRE COMPENSE, et pourquoi il est GRAND. Le fond publié n'est pas le décor :
// c'est une reconstruction, et elle écarte les instances au maillage grossier
// (`himap.AireMaxTriangleJouable`) — mesuré à 10,8 % de la carte validée manquants sur
// Cliffhanger. Ce qui manque manque par INSTANCE ENTIÈRE : une rampe, une dalle, un segment
// de passerelle, donc des trous de quelques mètres. La fermeture est à l'échelle de ce qui
// manque, pas à celle d'un défaut de rastérisation.
//
// ÉTALONNÉ SUR DEUX ORACLES INDÉPENDANTS, et 4,00 m est le PREMIER rayon qui les passe TOUS
// LES DEUX sur tout l'échantillon (mesures du 2026-08-16, `TestOracleTolerance` et
// `TestOracleIoUContreLeDecoupePOC` ; 7 films de 7 cartes) :
//
//	rayon   IoU médian    pire perte de positions (carte)
//	0,00       0,809        -40,47 pt  (btb_highpower)
//	1,00       0,816        -17,53 pt  (btb_highpower)
//	2,00       0,819         -7,39 pt  (ctf_breaker)
//	3,00       0,865         -2,58 pt  (ctf_breaker)     seuil IoU passé, positions non
//	4,00       0,872         -1,09 pt  (ctf_breaker)     <- retenu : les deux passent
//	5,00          —          -0,16 pt  (ctf_breaker)     chasm s'effondre à 7 % de rognage
//
// CE QU'IL NE NEUTRALISE PAS. À 4 m la fermeture n'ajoute que 3,4 points de matière au cadre
// de Ridgeline (18,6 % -> 22,0 %), et le découpage retire toujours 47,7 % de l'aire des pavés
// sur btb_highpower, 56,5 % sur ctf_breaker, 22,9 % sur Ridgeline — soit, sur cette dernière,
// exactement l'ordre de grandeur du découpage par étage du POC (21 %). Le grand vide
// extérieur, la demande de l'utilisateur, reste coupé ; seuls les trous de reconstruction se
// referment. Au-delà, à 5 m, chasm tombe de 20,6 % à 7,3 % de rognage : la fermeture
// commencerait à recoller la carte à son décor.
const ToleranceParDefaut = 4.00

// OptionsParDefaut rend les réglages de PRODUCTION du catalogue.
func OptionsParDefaut() Options {
	return Options{SimplifieM: SimplifieParDefaut, AireMinM2: AireMinParDefaut, RendLesEnclaves: true}
}

// Resultat porte le découpage d'une zone et de quoi le JUGER.
type Resultat struct {
	// Contour est la partie principale (la plus grande), Parties les autres, Trous les
	// évidements — le tout en mètres monde, anneaux OUVERTS (le rendu referme).
	Contour [][2]float64
	Parties [][][2]float64
	Trous   [][][2]float64
	// AireM2 est l'aire retenue, AireBrutM2 celle du pavé d'origine — toutes deux mesurées
	// sur la MÊME grille, donc directement comparables.
	AireM2     float64
	AireBrutM2 float64
	// Degenere dit que le découpage n'a rien laissé d'exploitable : l'appelant garde le brut.
	Degenere bool
}

// PartGardee est la part d'aire que le découpage conserve. 0 quand le brut est vide.
func (r Resultat) PartGardee() float64 {
	if r.AireBrutM2 <= 0 {
		return 0
	}
	return r.AireM2 / r.AireBrutM2
}

// Decoupe intersecte un polygone monde avec le masque praticable.
func Decoupe(brut [][2]float64, m *Masque, o Options) Resultat {
	if m == nil || len(brut) < 3 {
		return Resultat{Degenere: true}
	}
	anneaux := [][][2]float64{brut}
	e, ok := m.emprise(anneaux)
	if !ok {
		return Resultat{Degenere: true}
	}
	zone := m.rasterise(anneaux, e)
	garde := make([]bool, len(zone))
	nBrut := 0
	for k, d := range zone {
		if !d {
			continue
		}
		nBrut++
		garde[k] = m.dur[(e.j0+k/e.nx)*m.NX+e.i0+k%e.nx]
	}
	if o.RendLesEnclaves {
		reprendLesEnclaves(zone, garde, e)
	}
	nGarde := 0
	for _, g := range garde {
		if g {
			nGarde++
		}
	}
	cell := m.CelluleM2()
	res := Resultat{AireBrutM2: float64(nBrut) * cell, AireM2: float64(nGarde) * cell}
	if res.AireM2 < o.AireMinM2 {
		res.Degenere = true
		return res
	}
	res.Contour, res.Parties, res.Trous = e.anneaux(contours(garde, e.nx, e.ny), m, o.SimplifieM)
	if len(res.Contour) < 3 {
		res.Degenere = true
	}
	return res
}

// reprendLesEnclaves rend à la zone le vide qu'elle ENFERME.
//
// LA DEMANDE, mot pour mot : « les zones dépassent sur le grand fond transparent, zone
// inaccessible sans mourir » (item 9.2). Ce qui est visé, c'est le DÉBORDEMENT — le vide qui
// communique avec le dehors de la zone. Un vide entouré de décor, lui, n'est pas un
// débordement : c'est un trou de reconstruction (passerelle ajourée, joint entre deux
// instances) ou une fosse au milieu d'une place, et le rogner retirerait de l'emprise jouée.
//
// La règle sépare les deux sans réglage : on inonde le vide depuis le bord de la zone ; ce
// que l'eau atteint est du débordement, le reste revient à la zone. Un paramètre de moins
// qu'une dilatation, et il n'y a rien à étalonner.
func reprendLesEnclaves(zone, garde []bool, e emprise) {
	in := inondation{zone: zone, garde: garde, vu: make([]bool, len(zone)), e: e}
	in.amorce()
	in.propage()
	for k := range zone {
		if zone[k] && !in.vu[k] {
			garde[k] = true
		}
	}
}

// inondation porte l'état de l'inondation du vide d'une zone depuis son bord.
type inondation struct {
	zone, garde, vu []bool
	e               emprise
	file            []int
}

// pousse enrôle une cellule si elle est du vide de la zone, pas encore atteint.
func (in *inondation) pousse(i, j int) {
	if i < 0 || i >= in.e.nx || j < 0 || j >= in.e.ny {
		return
	}
	k := j*in.e.nx + i
	if !in.zone[k] || in.garde[k] || in.vu[k] {
		return
	}
	in.vu[k] = true
	in.file = append(in.file, k)
}

// amorce met à l'eau tout vide qui touche le dehors de la zone — ou le bord de l'emprise,
// qui est le dehors quand la grille a clippé la zone.
func (in *inondation) amorce() {
	for j := 0; j < in.e.ny; j++ {
		for i := 0; i < in.e.nx; i++ {
			k := j*in.e.nx + i
			if !in.zone[k] || in.garde[k] {
				continue
			}
			if i == 0 || j == 0 || i == in.e.nx-1 || j == in.e.ny-1 || in.touche(i, j) {
				in.pousse(i, j)
			}
		}
	}
}

// touche dit si une cellule de la zone a un voisin hors zone.
func (in *inondation) touche(i, j int) bool {
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		vi, vj := i+d[0], j+d[1]
		if vi < 0 || vi >= in.e.nx || vj < 0 || vj >= in.e.ny || !in.zone[vj*in.e.nx+vi] {
			return true
		}
	}
	return false
}

// propage étend l'inondation de proche en proche.
func (in *inondation) propage() {
	for len(in.file) > 0 {
		k := in.file[len(in.file)-1]
		in.file = in.file[:len(in.file)-1]
		i, j := k%in.e.nx, k/in.e.nx
		in.pousse(i+1, j)
		in.pousse(i-1, j)
		in.pousse(i, j+1)
		in.pousse(i, j-1)
	}
}

// emprise : la fenêtre de cellules qu'occupe une zone dans la grille du fond.
type emprise struct{ i0, j0, nx, ny int }

// emprise borne un jeu d'anneaux monde en cellules, avec une cellule de marge, clippée à la
// grille.
func (m *Masque) emprise(anneaux [][][2]float64) (emprise, bool) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, poly := range anneaux {
		for _, p := range poly {
			minX, maxX = math.Min(minX, p[0]), math.Max(maxX, p[0])
			minY, maxY = math.Min(minY, p[1]), math.Max(maxY, p[1])
		}
	}
	mpp := m.Calage.MetersPerPixel
	if mpp <= 0 {
		return emprise{}, false
	}
	i0 := int(math.Floor((minX-m.Calage.OriginX)/mpp)) - 1
	i1 := int(math.Floor((maxX-m.Calage.OriginX)/mpp)) + 1
	j0 := int(math.Floor((m.Calage.OriginY-maxY)/mpp)) - 1
	j1 := int(math.Floor((m.Calage.OriginY-minY)/mpp)) + 1
	i0, j0 = max(i0, 0), max(j0, 0)
	i1, j1 = min(i1, m.NX-1), min(j1, m.NY-1)
	if i1 < i0 || j1 < j0 {
		return emprise{}, false
	}
	return emprise{i0: i0, j0: j0, nx: i1 - i0 + 1, ny: j1 - j0 + 1}, true
}

// rasterise remplit les cellules dont le CENTRE tombe dans la figure (règle pair-impair sur
// TOUS les anneaux, balayage par lignes). Le centre, pas le coin : c'est la même convention
// que le calage publié, donc le découpage et le fond parlent de la même cellule.
//
// Prendre les anneaux ensemble, et non un par un, est ce qui rend la règle pair-impair
// exacte : un trou creuse, une partie posée dans un trou re-remplit — la même règle que le
// rendu applique au dessin.
func (m *Masque) rasterise(anneaux [][][2]float64, e emprise) []bool {
	out := make([]bool, e.nx*e.ny)
	mpp := m.Calage.MetersPerPixel
	xs := make([]float64, 0, 16)
	for j := 0; j < e.ny; j++ {
		y := m.Calage.OriginY - (float64(e.j0+j)+0.5)*mpp
		xs = xs[:0]
		for _, poly := range anneaux {
			xs = coupures(poly, y, xs)
		}
		if len(xs) < 2 {
			continue
		}
		sort.Float64s(xs)
		for k := 0; k+1 < len(xs); k += 2 {
			m.remplitSegment(out, e, j, xs[k], xs[k+1])
		}
	}
	return out
}

// coupures rend les abscisses où le contour croise la ligne y.
func coupures(poly [][2]float64, y float64, dst []float64) []float64 {
	n := len(poly)
	for k := 0; k < n; k++ {
		a, b := poly[k], poly[(k+1)%n]
		if (a[1] > y) == (b[1] > y) {
			continue
		}
		dst = append(dst, a[0]+(y-a[1])*(b[0]-a[0])/(b[1]-a[1]))
	}
	return dst
}

// remplitSegment allume les cellules d'une ligne dont le centre tombe dans [x0, x1).
func (m *Masque) remplitSegment(out []bool, e emprise, j int, x0, x1 float64) {
	mpp := m.Calage.MetersPerPixel
	iMin := int(math.Ceil((x0-m.Calage.OriginX)/mpp - 0.5))
	iMax := int(math.Ceil((x1-m.Calage.OriginX)/mpp-0.5)) - 1
	iMin, iMax = max(iMin, e.i0), min(iMax, e.i0+e.nx-1)
	for i := iMin; i <= iMax; i++ {
		out[j*e.nx+i-e.i0] = true
	}
}

// anneaux convertit les boucles du treillis en anneaux monde, simplifiés et TRIÉS : le
// contour principal est le plus grand, l'ordre du reste est déterministe (le fichier
// produit est versionné — deux exécutions doivent rendre le même octet).
func (e emprise) anneaux(boucles [][][2]int, m *Masque, eps float64) ([][2]float64, [][][2]float64, [][][2]float64) {
	type anneau struct {
		pts  [][2]float64
		aire float64
	}
	var exts, trous []anneau
	for _, b := range boucles {
		a := float64(aireSignee(b)) / 2
		pts := arrondiCentimetre(simplifieAnneau(e.versMonde(b, m), eps))
		if len(pts) < 3 {
			continue
		}
		if a > 0 {
			exts = append(exts, anneau{pts, a})
			continue
		}
		trous = append(trous, anneau{pts, -a})
	}
	tri := func(s []anneau) {
		sort.SliceStable(s, func(i, j int) bool {
			if s[i].aire != s[j].aire {
				return s[i].aire > s[j].aire
			}
			return s[i].pts[0][0] < s[j].pts[0][0]
		})
	}
	tri(exts)
	tri(trous)
	if len(exts) == 0 {
		return nil, nil, nil
	}
	parties := make([][][2]float64, 0, len(exts)-1)
	for _, a := range exts[1:] {
		parties = append(parties, a.pts)
	}
	sorties := make([][][2]float64, 0, len(trous))
	for _, a := range trous {
		sorties = append(sorties, a.pts)
	}
	return exts[0].pts, parties, sorties
}

// arrondiCentimetre ramène un anneau au centimètre.
//
// Le catalogue est un fichier VERSIONNÉ, servi par match : publier quinze chiffres
// significatifs pour une géométrie que la grille résout à 9,2 cm — et que la simplification
// déplace déjà de 8 cm — n'ajoute pas de précision, seulement des octets (mesuré :
// 2,78 Mo -> 2,06 Mo sur les 22 cartes). Le centimètre est un neuvième de cellule : très
// au-delà de ce que la mesure sait dire.
//
// Les pavés BRUTS conservés ne passent pas ici : ce sont les valeurs du jeu, gardées telles
// quelles.
func arrondiCentimetre(pts [][2]float64) [][2]float64 {
	for k, p := range pts {
		pts[k] = [2]float64{math.Round(p[0]*100) / 100, math.Round(p[1]*100) / 100}
	}
	return pts
}

// versMonde place une boucle du treillis local dans le repère monde. Les sommets sont des
// COINS de cellule, pas des centres : le coin (i, j) est le bord haut-gauche de la cellule.
func (e emprise) versMonde(b [][2]int, m *Masque) [][2]float64 {
	mpp := m.Calage.MetersPerPixel
	out := make([][2]float64, len(b))
	for k, p := range b {
		out[k] = [2]float64{
			m.Calage.OriginX + float64(e.i0+p[0])*mpp,
			m.Calage.OriginY - float64(e.j0+p[1])*mpp,
		}
	}
	return out
}
