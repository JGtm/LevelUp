// Package himap — rendu.go : la carte est un RENDU DU MAILLAGE VU DU DESSUS.
//
// POURQUOI CE FICHIER EXISTE, ET POURQUOI IL REMPLACE `volume.go` COMME VOIE DE RENDU.
//
// L'artefact valide le 2026-07-26 n'est pas une carte de praticabilite. C'est un rendu 3D a
// plat : chaque pixel porte la surface la plus HAUTE, et sa teinte vient de l'inclinaison de
// cette surface. Rochers, terrain et toits y figurent — ils ne genent pas la lecture, ils la
// FONT, parce que l'ombrage donne le relief. C'est exactement ce que produit un exporteur de
// maillage comme `Gravemind2401/Reclaimer` suivi d'une vue du dessus.
//
// L'erreur qui a coute deux jours : avoir lu « l'arene est illisible sous les rochers » comme
// un probleme de SELECTION (quelles surfaces garder) alors que c'etait un probleme
// d'ECLAIRAGE (comment les dessiner). Un champ d'altitude peint par une rampe de couleur est
// illisible ; le meme champ ombre par sa normale est une carte.
//
// La recette tient en trois lignes et n'a aucun reglage par carte :
//
//  1. z-buffer : pour chaque triangle, garder par pixel l'altitude la plus haute ;
//  2. memoriser la normale de la face retenue ;
//  3. teinter par un eclairage de Lambert, lumiere fixe et oblique.
package himap

import "math"

// LumiereRendu : direction de la lumiere, oblique pour que le relief se detache. Fixe et
// universelle — une lumiere par carte n'aurait aucun sens.
var LumiereRendu = normalise([3]float64{-0.4, 0.5, 0.75})

// TrancheDeJeuMin / TrancheDeJeuMax : la tranche d'altitude hors de laquelle la matiere
// n'appartient plus a la carte. Elles sont RELATIVES AU SOL JOUE — cf. `TrancheDeJeu`.
//
// D'OU ELLES VIENNENT. Du prototype, qui les portait depuis le debut — `s31_raster.py` borne
// son volume a `ZB0, ZB1 = -12.0, 28.0` — et le handoff §3 les nommait « la tranche de jeu ».
// Le portage en z-buffer n'avait retenu que le plafond, et c'est ce qui noyait la carte.
//
// CE QUI LES CONFIRME (ridgeline, comparaison pixel a pixel avec la carte validee) : sous
// -10 m, la matiere dessinee est en trop a 96 % (~807 000 px contre ~30 000 justes). Remonter
// le plancher a -6 fait passer les pixels manquants de 4,0 a 7,3 % ; le plafond entre +20 et
// +28 ne change rien.
//
// CE QU'ELLES NE SONT PAS : une regle generale en EPAISSEUR. L'epaisseur de 40 m est mesuree
// sur UNE carte (registre des reports). Ce qui a ete tranche le 2026-08-10, c'est leur ORIGINE :
// relative, pas absolue.
const (
	TrancheDeJeuMin = -12.0
	TrancheDeJeuMax = 28.0
)

// TrancheDeJeu rend la tranche d'altitude d'une carte, translatee au sol JOUE.
//
// POURQUOI RELATIVE, ET POURQUOI CA A CHANGE LE 2026-08-10. Ces bornes etaient appliquees en
// altitudes ABSOLUES sur les cartes natives — un accident propre a Cliffhanger, dont le sol
// joue est a -2,2 m. La cuisson des 17 cartes a montre que le niveau joue s'etale de
// **-136,7 m (`chasm`) a +52,3 m (Vagabond)** : la tranche absolue decapitait les cartes qui ne
// jouent pas vers zero. Mesure :
//
//	                     absolue [-12;+28]   relative au sol des ancres
//	chasm                      5/17 ancres        17/17
//	btb_highpower             14/51              38/51
//	catalyst                  24/24              24/24
//	ridgeline                 14/14              14/14
//	ridgeline, oracle FORT    accord 66,7 %      accord 64,7 % · positions 93,95 %
//
// Le cout est reel et assume par decision utilisateur du 2026-08-10 : sur Cliffhanger l'exces
// passe de 33,8 a 39,1 % (la vallee entre dans le cadre) et l'accord perd 2 points. Deux cartes
// inexploitables valent plus qu'un point d'accord sur une carte deja lisible.
//
// LA CHAINE FORGE FAISAIT DEJA CECI. C'est meme d'elle que vient la lecture juste : le sol de
// Vagabond vit vers z=52, personne n'aurait pu lui appliquer une tranche absolue.
//
// UNE SEULE EXPRESSION, partagee par la production et par le banc de non-regression : deux
// copies de cette translation finiraient par diverger, et le banc cesserait de garder la
// production.
func TrancheDeJeu(zJeu float64) (min, max float64) {
	return zJeu + TrancheDeJeuMin, zJeu + TrancheDeJeuMax
}

// Rendu porte un z-buffer et la normale retenue par pixel.
type Rendu struct {
	// TypeCourant / typeGagnant : DIAGNOSTIC. La cuisson Forge annonce, avant de poser un
	// objet, de quel TYPE il est ; le rendu retient alors, pour chaque pixel, le type qui a
	// gagne le z-buffer. C est la seule facon de nommer ce qui peint reellement une zone de
	// l image — trente rendus et une dizaine d exclusions n y sont pas parvenus, et le type
	// le plus soupconne (les branches d Isolation) s est revele n occuper AUCUN pixel : son
	// exclusion ne changeait pas un octet du fichier.
	TypeCourant int32
	typeGagnant []int32
	// ObjetCourant / objetGagnant : la meme idee que TypeCourant, mais a l INSTANCE. Le type
	// ne suffit pas a nommer un element : tous les murs d un meme modele le partagent, et
	// raisonner par type reviendrait a traiter ensemble des objets poses aux quatre coins de la
	// carte. L instance, elle, designe UN objet pose — c est la maille du recollement
	// (recollement_objets.go), qui decide de garder ou de retirer un element ENTIER.
	ObjetCourant int32
	objetGagnant []int32
	// RecollementRetires : DIAGNOSTIC du recollement — combien d objets ont ete retires entiers
	// parce que le masque n en gardait qu une part trop faible. Publie au journal de cuisson.
	RecollementRetires int
	// zBas / nBas : LA SURFACE LA PLUS BASSE AU-DESSUS DU SOL JOUE, par pixel.
	//
	// Le z-buffer ordinaire retient la surface la plus HAUTE : sur une carte a ciel ferme, il
	// ne montre donc jamais que le plafond. Mesure du 2026-08-27 sur Isolation : le type qui
	// peint 82,7 pour cent de l image est pose entre Z 136 et 160 quand le sol joue est a
	// Z 117 — c est un DOME. Et comme sa paroi descend jusqu au sol, aucune coupe en altitude
	// ne l en separe : c est ce qui a fait echouer l ecretage a 4, 2 et 1 m, la tranche
	// plafonnee a +3, +6 et +12, et le bornage.
	//
	// D ou cette seconde voie : garder, pour chaque pixel, la surface la plus BASSE qui reste
	// au-dessus du sol joue. Sous un dome, c est le sol ; a ciel ouvert, c est la meme surface
	// que la voie haute. On ne retire rien : on regarde d en dessous.
	zBas []float64
	nBas [][3]float64
	// couvertureNavmesh : les cellules que le maillage de navigation couvre. Memorisee au
	// moment ou la reference est armee, parce que les tampons de reference sont liberes des
	// que la substitution a decide.
	couvertureNavmesh []bool
	// referenceNavmesh : l altitude du sol donnee par le maillage, gardee apres que la voie de
	// reference a libere ses tampons. NaN hors du maillage.
	referenceNavmesh []float64
	// SeuilArete : denivele entre voisins au-dela duquel on trace un bord. Zero = le defaut
	// (SeuilAreteMetres). Reglage PAR CARTE, cf. rendu_couleur.go.
	SeuilArete float64
	Cell       float64
	Min        [2]float64
	NX, NY     int
	z          []float64
	plafond    float64
	plancher   float64
	// niveauJeu : altitude du sol JOUE, deduite des ancres. NaN = inconnu.
	niveauJeu float64
	n         [][3]float64
	// ref / dRef / zRef / nRef : la voie de REFERENCE (cf. rendu_reference.go). Nil tant que
	// `ArmeReference` n'a pas ete appelee — le z-buffer « plus haut » reste alors seul.
	ref  []float64
	dRef []float64
	zRef []float64
	nRef [][3]float64
	// eau : cellules couvertes par un volume d'eau (PoseEau, cf. sddt.go). Un habillage —
	// jamais consulte par le z-buffer ni par les metriques du banc.
	eau []bool
	// ecrete marque les cellules RETIREES de la carte — par l ecretage des toits ou par le
	// masque des zones nommees. L eau ne s y pose pas :
	// elle est peinte meme sans matiere (c est sa raison d etre), et sur une carte couverte
	// elle remplissait alors le trou laisse par l ecretage — 30 970 cellules d eau devenues
	// 325 353 sur Recharge, une dalle bleue en travers de la carte (mesure du 2026-08-26).
	ecrete []bool
	// solSuppose marque les cellules SANS matiere comblees par un aplat (CombleTrous).
	solSuppose []bool
}

// NewRendu prepare un rendu sur une emprise et une resolution donnees.
func NewRendu(min, max [2]float64, cell float64) *Rendu {
	nx := int((max[0]-min[0])/cell) + 1
	ny := int((max[1]-min[1])/cell) + 1
	r := &Rendu{Cell: cell, Min: min, NX: nx, NY: ny,
		z: make([]float64, nx*ny), n: make([][3]float64, nx*ny),
		plafond: math.NaN(), plancher: math.NaN(), niveauJeu: math.NaN()}
	for i := range r.z {
		r.z[i] = math.Inf(-1)
	}
	return r
}

// AddMesh projette un maillage, place par son instance.
func (r *Rendu) AddMesh(m *Mesh, in Instance) {
	if m == nil {
		return
	}
	monde := make([][3]float64, len(m.Vertices))
	for i, s := range m.Vertices {
		monde[i] = in.LocalToWorld(s)
	}
	for _, t := range m.Triangles {
		r.triangle(monde[t[0]], monde[t[1]], monde[t[2]])
	}
}

// AddMeshBorne projette un maillage en n'ecrivant que dans la boite monde declaree de son
// instance.
//
// Meme raison que pour le volume : la boite (sbsp @0x7C) et le maillage (tag rtgo) viennent de
// deux sources independantes, et quelques instances des modules globaux debordent d'un facteur
// 42,8 de leur diagonale au 99e centile. Sans bornage elles deversent du decor sur toute la
// carte. Avec, on peut enfin prendre TOUS les modules — donc les dalles de terrain qui portent
// le second pont — sans ramener l'aberration.
func (r *Rendu) AddMeshBorne(m *Mesh, in Instance, marge float64) {
	if m == nil {
		return
	}
	monde := make([][3]float64, len(m.Vertices))
	for i, s := range m.Vertices {
		monde[i] = in.LocalToWorld(s)
	}
	lo := [3]float64{in.AABBMin[0] - marge, in.AABBMin[1] - marge, in.AABBMin[2] - marge}
	hi := [3]float64{in.AABBMax[0] + marge, in.AABBMax[1] + marge, in.AABBMax[2] + marge}
	for _, t := range m.Triangles {
		r.triangleBorne(monde[t[0]], monde[t[1]], monde[t[2]], lo, hi)
	}
}

func (r *Rendu) triangle(a, b, c [3]float64) {
	inf := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	sup := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	r.triangleBorne(a, b, c, inf, sup)
}

func (r *Rendu) triangleBorne(a, b, c [3]float64, lo, hi [3]float64) {
	minX := math.Min(a[0], math.Min(b[0], c[0]))
	maxX := math.Max(a[0], math.Max(b[0], c[0]))
	minY := math.Min(a[1], math.Min(b[1], c[1]))
	maxY := math.Max(a[1], math.Max(b[1], c[1]))
	if maxX < r.Min[0] || maxY < r.Min[1] ||
		minX > r.Min[0]+float64(r.NX)*r.Cell || minY > r.Min[1]+float64(r.NY)*r.Cell {
		return
	}
	minX, maxX = math.Max(minX, lo[0]), math.Min(maxX, hi[0])
	minY, maxY = math.Max(minY, lo[1]), math.Min(maxY, hi[1])
	if minX > maxX || minY > maxY {
		return
	}
	nrm := normaleFace(a, b, c)
	i0 := borne(int((minX-r.Min[0])/r.Cell), r.NX-1)
	i1 := borne(int((maxX-r.Min[0])/r.Cell), r.NX-1)
	j0 := borne(int((minY-r.Min[1])/r.Cell), r.NY-1)
	j1 := borne(int((maxY-r.Min[1])/r.Cell), r.NY-1)
	det := (b[1]-c[1])*(a[0]-c[0]) + (c[0]-b[0])*(a[1]-c[1])
	for j := j0; j <= j1; j++ {
		y := r.Min[1] + (float64(j)+0.5)*r.Cell
		for i := i0; i <= i1; i++ {
			x := r.Min[0] + (float64(i)+0.5)*r.Cell
			z, dedans := altitudeAuPoint(a, b, c, det, x, y)
			if !dedans || !r.accepteZ(z, lo[2], hi[2]) {
				continue
			}
			r.deposeAltitude(j*r.NX+i, z, nrm)
		}
	}
}

// deposeAltitude confronte une altitude retenue aux trois voies du rendu pour la cellule `k` :
// la surface la plus BASSE au-dessus du plancher vu, la surface la plus HAUTE (voie principale,
// qui emporte aussi le type et l'objet gagnants), et la surface la plus proche du sol de
// reference. Extrait du corps de boucle de triangleBorne, a l'identique.
func (r *Rendu) deposeAltitude(k int, z float64, nrm [3]float64) {
	if r.zBas != nil && z >= r.plancherSolVu() && z < r.zBas[k] {
		r.zBas[k], r.nBas[k] = z, nrm
	}
	if z > r.z[k] {
		r.z[k], r.n[k] = z, nrm
		if r.typeGagnant != nil {
			r.typeGagnant[k] = r.TypeCourant
		}
		if r.objetGagnant != nil {
			r.objetGagnant[k] = r.ObjetCourant
		}
	}
	// Voie de reference (rendu_reference.go) : retenir AUSSI la surface la plus proche du sol de
	// reference. Strictement `<` : la premiere face gagne les ex aequo, comme sur la voie haute —
	// le determinisme au bit en depend.
	if r.zRef != nil {
		if d := math.Abs(z - r.ref[k]); d < r.dRef[k] {
			r.dRef[k], r.zRef[k], r.nRef[k] = d, z, nrm
		}
	}
}

// accepteZ dit si une altitude passe les DEUX filtres : la boite de l'instance (bornage) et
// la tranche de jeu du rendu (plancher / plafond). Ils sont independants et se cumulent.
func (r *Rendu) accepteZ(z, zLo, zHi float64) bool {
	if z < zLo || z > zHi {
		return false
	}
	if !math.IsNaN(r.plafond) && z > r.plafond {
		return false
	}
	return math.IsNaN(r.plancher) || z >= r.plancher
}

// Eclairement rend l'intensite d'un pixel dans [0,1], et dit si le pixel porte de la matiere.
//
// La normale est prise en VALEUR ABSOLUE sur la verticale : l'ordre des sommets n'est pas
// coherent d'un maillage a l'autre dans ces tags, et une face vue du dessus doit s'eclairer
// pareil quel que soit son enroulement.
func (r *Rendu) Eclairement(i, j int) (float64, bool) {
	k := j*r.NX + i
	if i < 0 || i >= r.NX || j < 0 || j >= r.NY || math.IsInf(r.z[k], -1) {
		return 0, false
	}
	n := r.n[k]
	if n[2] < 0 {
		n = [3]float64{-n[0], -n[1], -n[2]}
	}
	d := n[0]*LumiereRendu[0] + n[1]*LumiereRendu[1] + n[2]*LumiereRendu[2]
	// Eclairage hemispherique : jamais de noir total, le relief reste lisible dans l'ombre.
	return 0.25 + 0.75*math.Max(0, d), true
}

// Altitude rend l'altitude retenue par un pixel.
func (r *Rendu) Altitude(i, j int) (float64, bool) {
	k := j*r.NX + i
	if i < 0 || i >= r.NX || j < 0 || j >= r.NY || math.IsInf(r.z[k], -1) {
		return 0, false
	}
	return r.z[k], true
}

func normaleFace(a, b, c [3]float64) [3]float64 {
	u := [3]float64{b[0] - a[0], b[1] - a[1], b[2] - a[2]}
	v := [3]float64{c[0] - a[0], c[1] - a[1], c[2] - a[2]}
	return normalise([3]float64{
		u[1]*v[2] - u[2]*v[1],
		u[2]*v[0] - u[0]*v[2],
		u[0]*v[1] - u[1]*v[0],
	})
}

func normalise(v [3]float64) [3]float64 {
	n := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
	if n == 0 {
		return [3]float64{0, 0, 1}
	}
	return [3]float64{v[0] / n, v[1] / n, v[2] / n}
}

// Plafond limite le rendu aux surfaces sous une altitude donnee. NaN = aucun plafond.
//
// C'est le discriminant qui manquait. Ni le module ni la boite de l'instance ne separent
// l'arene de ce qui l'enterre : mesure du 2026-08-09, le bornage a la boite ne deplace la
// couverture que de 86,7 % a 86,5 %. Les rochers ne sont pas des maillages aberrants, ce sont
// de vrais rochers — SUSPENDUS AU-DESSUS de l'aire de jeu, et un z-buffer qui garde la surface
// la plus haute les garde eux. Le prototype le disait deja : son volume etait « borne a la
// tranche de jeu ».
//
// Le plafond se DEDUIT des ancres d'objectifs (cf. reference.go), il ne se regle pas par carte.
func (r *Rendu) Plafond(z float64) { r.plafond = z }

// Plancher limite le rendu aux surfaces AU-DESSUS d'une altitude donnee. NaN = aucun plancher.
//
// IL MANQUAIT, et c'est ce qui noyait la carte des qu'on prenait les modules globaux. Le
// prototype le portait depuis le debut — `s31_raster.py` borne son volume a
// `ZB0, ZB1 = -12.0, 28.0`, la « tranche de jeu » citee par le handoff §3 — et la traduction
// en z-buffer n'a retenu que le plafond.
//
// MESURE (ridgeline, tous modules, contre la carte validee) : sous -10 m, la matiere dessinee
// est en trop a 96 % (~807 000 pixels contre ~30 000 justes). Ce n'est pas de la carte, c'est
// la vallee autour.
//
// La TRANCHE se deduit, elle ne se regle pas : cf. `TrancheDeJeu`.
func (r *Rendu) Plancher(z float64) { r.plancher = z }

// Tranche pose plancher et plafond d'un coup.
func (r *Rendu) Tranche(zmin, zmax float64) {
	r.Plancher(zmin)
	r.Plafond(zmax)
}

// NiveauDeJeu declare l'altitude du sol joue, deduite des ancres d'objectifs. Elle sert aux
// habillages qui font reculer ce qui n'est pas a hauteur de jeu (cf. `rendu_couleur.go`).
func (r *Rendu) NiveauDeJeu(z float64) { r.niveauJeu = z }

// EcartAuNiveauDeJeu rend l'ecart signe d'une cellule au niveau joue. `ok` est faux si la
// cellule est vide ou si aucun niveau n'a ete declare.
func (r *Rendu) EcartAuNiveauDeJeu(i, j int) (float64, bool) {
	if math.IsNaN(r.niveauJeu) {
		return 0, false
	}
	z, ok := r.Altitude(i, j)
	if !ok {
		return 0, false
	}
	return z - r.niveauJeu, true
}

// ArmeObjetGagnant fait retenir, pour chaque pixel, l INSTANCE qui a gagne le z-buffer.
// Zero vaut « aucune » : la numerotation des objets commence donc a 1.
func (r *Rendu) ArmeObjetGagnant() {
	r.objetGagnant = make([]int32, r.NX*r.NY)
}

// ArmeTypeGagnant fait retenir, pour chaque pixel, le TYPE d'objet qui a gagne le z-buffer.
// Diagnostic pur : il ne change rien au rendu, il permet seulement de demander a l'image
// « qui t'a peint ici ».
func (r *Rendu) ArmeTypeGagnant() {
	r.typeGagnant = make([]int32, r.NX*r.NY)
}

// TypeGagnant rend le type qui occupe ce pixel, et s'il y en a un.
func (r *Rendu) TypeGagnant(i, j int) (int32, bool) {
	if r.typeGagnant == nil || i < 0 || j < 0 || i >= r.NX || j >= r.NY {
		return 0, false
	}
	k := j*r.NX + i
	if math.IsInf(r.z[k], -1) {
		return 0, false
	}
	return r.typeGagnant[k], true
}

// PixelsParType compte, pour chaque type, le nombre de pixels qu'il occupe dans l'image.
// C'est la mesure qui DESIGNE ce qui couvre l'arene, la ou aucun critere porte par le modele
// (emprise, aire du maillage, couverture de son emprise au sol) n'y est parvenu.
func (r *Rendu) PixelsParType() map[int32]int {
	out := map[int32]int{}
	if r.typeGagnant == nil {
		return out
	}
	for k := range r.z {
		if math.IsInf(r.z[k], -1) {
			continue
		}
		out[r.typeGagnant[k]]++
	}
	return out
}

// MargeSolVuDuDessous : de combien on descend SOUS le niveau de jeu pour accepter une surface
// comme candidate au sol. Le sol d'une arene n'est pas plan — 4 m couvrent les marches et les
// creux sans laisser entrer un sous-sol.
const MargeSolVuDuDessous = 4.0

// MargeSolVuDuDessousCarte, quand elle est > 0, remplace la marge par defaut. Reglable par
// carte parce que 4 m sont faits pour EXCLURE un sous-sol, et qu il existe des cartes ou le
// sous-sol est justement ce qu on veut voir (Vagabond, verdict utilisateur du 2026-08-30 :
// « on a surtout pas le sous-sol »). Elargir la marge le fait entrer — au prix du niveau du
// dessus, la ou les deux se superposent : une vue de dessus ne montre qu une surface par pixel.
var MargeSolVuDuDessousCarte = 0.0

// ArmeSurfaceBasse fait retenir, pour chaque pixel, la surface la plus BASSE au-dessus du sol
// joue. A appeler apres NiveauDeJeu et avant de projeter.
func (r *Rendu) ArmeSurfaceBasse() {
	r.zBas = make([]float64, r.NX*r.NY)
	r.nBas = make([][3]float64, r.NX*r.NY)
	for i := range r.zBas {
		r.zBas[i] = math.Inf(1)
	}
}

// plancherSolVu rend l'altitude sous laquelle une surface n'est plus candidate au sol.
func (r *Rendu) plancherSolVu() float64 {
	if math.IsNaN(r.niveauJeu) {
		return math.Inf(-1)
	}
	m := MargeSolVuDuDessous
	if MargeSolVuDuDessousCarte > 0 {
		m = MargeSolVuDuDessousCarte
	}
	return r.niveauJeu - m
}

// AdopteSurfaceBasse remplace la surface haute par la surface basse, la ou il y en a une, et
// rend le nombre de pixels changes. Un pixel sans candidate garde ce qu'il avait : on ne cree
// ni ne supprime jamais de matiere.
func (r *Rendu) AdopteSurfaceBasse() int {
	if r.zBas == nil {
		return 0
	}
	changes := 0
	for k := range r.z {
		if math.IsInf(r.z[k], -1) || math.IsInf(r.zBas[k], 1) || r.zBas[k] == r.z[k] {
			continue
		}
		r.z[k], r.n[k] = r.zBas[k], r.nBas[k]
		changes++
	}
	return changes
}
