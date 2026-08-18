package replay

// powerup_socle_oracle_test.go — PHASE 1 : la position du socle, MESUREE par les ramassages.
//
// Helpers communs et phase 0 : `powerup_socle_research_test.go`. Garde `OBJ_FILM_ART`.

import (
	"math"
	"sort"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------------------
// PHASE 1 — L'ORACLE DES RAMASSAGES.
//
// CE QUE L'ORACLE EST. `equipmentEpisodes` date l'etat ACTIF du surbouclier PAR VIE de bipede
// (schema 7). Hors Fiesta — et `01e1f945` est un KOTH — le jeu n'a AUCUNE autre source de
// surbouclier qu'un power-up ramasse sur la carte. Chaque `T0` d'episode date donc un
// RAMASSAGE, et un ramassage se fait EN MARCHANT SUR l'objet : la position du bipede a cet
// instant EST la position du socle, a la taille d'un Spartan pres.
//
// CE QUI LE REND MESURABLE PLUTOT QUE PLAUSIBLE : cinq episodes independants, cinq vies
// differentes, cinq instants espaces de 34,7 s a 521,9 s. S'ils tombent tous au meme endroit,
// cet endroit est un point FIXE de la carte — et c'est un fait, pas une lecture. Le temoin
// (les memes vies, des instants tires au hasard) dit ce que « tomber au meme endroit » vaut
// quand l'instant ne veut rien dire.
// ---------------------------------------------------------------------------------------

// psCentreCatalyst rend le centre de Catalyst pour les phases suivantes, calcule par la MEME
// regle et le MEME code que la phase 0 — jamais un litteral recopie, qui divergerait au
// premier correctif de la regle.
func psCentreCatalyst(t *testing.T) psPoint {
	t.Helper()
	dir := psArtDir(t)
	var docs []ReplayDocument
	for _, f := range psFilmsCatalyst {
		if doc, ok := psLoadDoc(t, dir, f.ID); ok {
			docs = append(docs, doc)
		}
	}
	if len(docs) == 0 {
		t.Skipf("aucun artefact Catalyst dans %s : le centre ne se calcule pas", dir)
	}
	ct := psCentreDesSocles(psSoclesUniques(docs))
	if ct.Paires < 2 || ct.SurAxe < 2 {
		t.Fatalf("centre non etabli : %d paires miroir, %d socles sur l'axe", ct.Paires, ct.SurAxe)
	}
	return ct.C
}

// psRayonOracle est le rayon MAXIMAL admis pour les positions de ramassage, en metres.
// Seuil ECRIT AVANT LA MESURE (plan, section 3) : deux Spartans qui marchent sur le meme
// objet sont a moins de 2 m l'un de l'autre.
const psRayonOracle = 2.0

// psFacteurTemoin est le rapport EXIGE entre la dispersion du temoin et celle de l'oracle.
// Ecrit avant la mesure : sans lui, « les cinq points sont proches » n'a aucun denominateur.
const psFacteurTemoin = 3.0

// psRamassage est UN ramassage date et situe.
type psRamassage struct {
	// Slot / T0 viennent de l'episode ; P est la position du bipede a cet instant.
	Slot uint32
	T0   int
	P    psPoint
	Z    float32
	// Ecart est la distance EN IMAGES entre `T0` et le point de piste retenu. Publie parce
	// qu'un point vieux de plusieurs secondes ne mesure plus le lieu du ramassage.
	Ecart int
}

// psPosDeLaVie rend la position de la vie `slot` a l'instant `t`, en prenant le point de
// piste le plus proche dans le temps. Rend ok=false si aucune piste ne porte ce slot.
//
// LE SLOT SEUL NE DESIGNE PAS UNE VIE — le pool reboucle — donc on retient, parmi TOUTES les
// pistes de ce slot, celle dont un point est le plus proche de `t`. C'est la seule facon de
// choisir sans supposer que `StartFrame`/`EndFrame` sont renseignes.
func psPosDeLaVie(doc ReplayDocument, slot uint32, t int) (psPoint, float32, int, bool) {
	best, bz, becart, trouve := psPoint{}, float32(0), 0, false
	for _, tr := range doc.Tracks {
		if tr.Slot != slot {
			continue
		}
		for _, p := range tr.Points {
			e := p.T - t
			if e < 0 {
				e = -e
			}
			if !trouve || e < becart {
				best, bz, becart, trouve = psPoint{X: p.X, Y: p.Y}, p.Z, e, true
			}
		}
	}
	return best, bz, becart, trouve
}

// psDispersion rend le centroide d'un nuage et son rayon MAXIMAL (la distance du point le
// plus eloigne au centroide) — deux chiffres, parce qu'un rayon sans centre ne se situe pas.
func psDispersion(pts []psPoint) (psPoint, float64) {
	if len(pts) == 0 {
		return psPoint{}, 0
	}
	var sx, sy float64
	for _, p := range pts {
		sx, sy = sx+float64(p.X), sy+float64(p.Y)
	}
	c := psPoint{X: float32(sx / float64(len(pts))), Y: float32(sy / float64(len(pts)))}
	var rmax float64
	for _, p := range pts {
		if d := psDist(p, c); d > rmax {
			rmax = d
		}
	}
	return c, rmax
}

// psInstantsTemoin rend, pour chaque episode, un instant TIRE de la meme vie mais decorrele
// du ramassage : le point de piste au rang `(indice * 7919) % n` de la vie. Deterministe
// (aucun generateur a graine cachee), et il passe par le MEME code de position que la mesure.
func psInstantsTemoin(doc ReplayDocument, eps []EquipmentEpisode) []int {
	out := make([]int, 0, len(eps))
	for i, e := range eps {
		var pts []Point
		for _, tr := range doc.Tracks {
			if tr.Slot == e.Slot {
				pts = append(pts, tr.Points...)
			}
		}
		if len(pts) == 0 {
			out = append(out, e.T0)
			continue
		}
		out = append(out, pts[(i*7919+3571)%len(pts)].T)
	}
	return out
}

// TestPowerupSocleOracle — PHASE 1 du plan : les cinq ramassages de `01e1f945`.
func TestPowerupSocleOracle(t *testing.T) {
	dir := psArtDir(t)
	doc, ok := psLoadDoc(t, dir, "01e1f945")
	if !ok {
		t.Skipf("artefact 01e1f945 absent de %s", dir)
	}
	var eps []EquipmentEpisode
	for _, e := range doc.EquipmentEpisodes {
		if e.Fam == EquipFamilyOvershield {
			eps = append(eps, e)
		}
	}
	if len(eps) == 0 {
		t.Skip("aucun episode de surbouclier : l'oracle n'a pas de matiere")
	}

	centre := psCentreCatalyst(t)
	t.Logf("=== 1.1 RAMASSAGES (centre de la carte %.3f ; %.3f) ===", centre.X, centre.Y)
	ram := make([]psRamassage, 0, len(eps))
	pts := make([]psPoint, 0, len(eps))
	for _, e := range eps {
		p, z, ecart, trouve := psPosDeLaVie(doc, e.Slot, e.T0)
		if !trouve {
			t.Errorf("  episode slot %d t0 %d : AUCUNE piste pour ce slot", e.Slot, e.T0)
			continue
		}
		ram = append(ram, psRamassage{Slot: e.Slot, T0: e.T0, P: p, Z: z, Ecart: ecart})
		pts = append(pts, p)
		t.Logf("  slot %3d  t0 %5d (%6.1f s)  ->  (%7.3f ; %7.3f ; z %6.3f)"+
			"  ecart %d image(s)  distance au centre %.2f m",
			e.Slot, e.T0, float64(e.T0)/10, p.X, p.Y, z, ecart, psDist(p, centre))
	}
	if len(pts) < 2 {
		t.Fatalf("moins de deux ramassages situes : rien a disperser")
	}

	c, rmax := psDispersion(pts)
	t.Log("=== 1.2 DISPERSION ===")
	t.Logf("  centroide (%.3f ; %.3f) | rayon max %.3f m | distance centroide-centre %.3f m",
		c.X, c.Y, rmax, psDist(c, centre))

	tem := psInstantsTemoin(doc, eps)
	tpts := make([]psPoint, 0, len(tem))
	t.Log("=== 1.3 TEMOIN (memes vies, instants decorreles) ===")
	for i, ti := range tem {
		p, _, _, trouve := psPosDeLaVie(doc, eps[i].Slot, ti)
		if !trouve {
			continue
		}
		tpts = append(tpts, p)
		t.Logf("  slot %3d  t %5d  ->  (%7.3f ; %7.3f)", eps[i].Slot, ti, p.X, p.Y)
	}
	tc, trmax := psDispersion(tpts)
	t.Logf("  centroide (%.3f ; %.3f) | rayon max %.3f m", tc.X, tc.Y, trmax)

	facteur := math.Inf(1)
	if rmax > 0 {
		facteur = trmax / rmax
	}
	t.Log("=== 1.4 VERDICT D'ETAPE ===")
	t.Logf("  seuils ecrits avant mesure : rayon <= %.1f m, temoin >= %.0fx",
		psRayonOracle, psFacteurTemoin)
	t.Logf("  mesure : rayon %.3f m, temoin %.3f m, facteur %.2f", rmax, trmax, facteur)
	switch {
	case rmax <= psRayonOracle && facteur >= psFacteurTemoin:
		t.Logf("  ATTEINT : la position du socle est MESUREE a (%.3f ; %.3f)", c.X, c.Y)
	default:
		t.Logf("  NON ATTEINT : le nuage des ramassages ne designe pas un point fixe")
	}
}

// ---------------------------------------------------------------------------------------
// 1.5 — REMONTER LA VIE DU PORTEUR.
//
// POURQUOI IL FAUT REMONTER. `EquipmentEpisode.T0` ne date pas le ramassage : il date
// l'instant ou la LECTURE du bouclier passe au-dessus du plein. Entre les deux, le porteur a
// continue de courir. Prendre sa position a `T0` mesure donc « ou il etait un instant APRES »,
// et un Spartan couvre 4 a 5 m par seconde.
//
// LA MESURE, ET C'EST ELLE QUI TRANCHE : on recule de `k` images (0 a 40, soit 0 a 4 s) et on
// regarde la DISPERSION du nuage des porteurs a chaque `k`. Si un socle existe, les cinq
// trajectoires se CROISENT en un point — la dispersion passe par un minimum net a un `k`
// commun. Si elle ne fait que grandir, les porteurs ne venaient pas du meme endroit et il n'y
// a pas de socle unique. Aucune des deux issues n'est supposee.
// ---------------------------------------------------------------------------------------

// psLagMax est la profondeur de remontee, en images du document (100 ms chacune) : 4 s, de
// quoi couvrir 16 a 20 m de course — bien au-dela de tout ecart de lecture de bouclier.
const psLagMax = 40

// psNuageA rend les positions des porteurs `k` images AVANT leur `T0`, et le nombre de
// porteurs situes. Le MEME code de position que la mesure a k=0 (`psPosDeLaVie`).
func psNuageA(doc ReplayDocument, eps []EquipmentEpisode, k int) ([]psPoint, []float32) {
	pts := make([]psPoint, 0, len(eps))
	zs := make([]float32, 0, len(eps))
	for _, e := range eps {
		p, z, _, ok := psPosDeLaVie(doc, e.Slot, e.T0-k)
		if !ok {
			continue
		}
		pts, zs = append(pts, p), append(zs, z)
	}
	return pts, zs
}

// psExpliqueParUnLacher dit si l'episode `e` ramasse un power-up LACHE A UNE MORT plutot
// qu'un power-up de socle, et rend la pose en cause.
//
// LA REGLE EST ECRITE AVANT LA MESURE et elle est SYMETRIQUE des constantes de production :
// une pose `powerup_*` d'origine `dropped`, ANTERIEURE a l'episode et a moins de 3 m de la
// position du porteur au ramassage. 3 m, c'est `equipOwnerMaxDist` — la distance au-dela de
// laquelle la proximite ne veut plus rien dire.
func psExpliqueParUnLacher(doc ReplayDocument, e EquipmentEpisode, p psPoint) (EquipmentPlacement, bool) {
	for _, pl := range doc.EquipmentPlacements {
		if !strings.HasPrefix(pl.Family, "powerup_") || pl.Origin != OriginDropped {
			continue
		}
		if pl.T0 > e.T0 {
			continue
		}
		if psDist(p, psPoint{X: pl.X, Y: pl.Y}) <= equipOwnerMaxDist {
			return pl, true
		}
	}
	return EquipmentPlacement{}, false
}

// TestPowerupSocleRemontee — 1.5 et 1.6 du plan : ou les porteurs se croisent, et quel
// ramassage n'est PAS celui d'un socle.
func TestPowerupSocleRemontee(t *testing.T) {
	dir := psArtDir(t)
	doc, ok := psLoadDoc(t, dir, "01e1f945")
	if !ok {
		t.Skipf("artefact 01e1f945 absent de %s", dir)
	}
	var eps []EquipmentEpisode
	for _, e := range doc.EquipmentEpisodes {
		if e.Fam == EquipFamilyOvershield {
			eps = append(eps, e)
		}
	}
	if len(eps) == 0 {
		t.Skip("aucun episode de surbouclier")
	}
	centre := psCentreCatalyst(t)

	t.Log("=== 1.6 QUEL RAMASSAGE N'EST PAS CELUI D'UN SOCLE ===")
	garde := make([]EquipmentEpisode, 0, len(eps))
	for _, e := range eps {
		p, _, _, ok := psPosDeLaVie(doc, e.Slot, e.T0)
		if !ok {
			continue
		}
		if pl, lache := psExpliqueParUnLacher(doc, e, p); lache {
			t.Logf("  slot %3d t0 %5d : ECARTE — pose `%s` %s lachee a t0 %d, a %.2f m",
				e.Slot, e.T0, pl.Family, pl.ID, pl.T0, psDist(p, psPoint{X: pl.X, Y: pl.Y}))
			continue
		}
		garde = append(garde, e)
	}
	t.Logf("  %d episodes retenus sur %d", len(garde), len(eps))
	if len(garde) < 2 {
		t.Skip("moins de deux ramassages de socle : rien a croiser")
	}

	t.Log("=== 1.5 REMONTEE : dispersion du nuage k images AVANT T0 ===")
	bestK, bestR, bestC := -1, math.Inf(1), psPoint{}
	for k := 0; k <= psLagMax; k++ {
		pts, zs := psNuageA(doc, garde, k)
		if len(pts) < len(garde) {
			continue
		}
		c, r := psDispersion(pts)
		var zmin, zmax float32 = zs[0], zs[0]
		for _, z := range zs {
			zmin, zmax = min(zmin, z), max(zmax, z)
		}
		if k%2 == 0 || r < bestR {
			t.Logf("  k=%2d (%4.1f s)  centroide (%7.3f ; %7.3f)  rayon %6.3f m"+
				"  z de %6.2f a %6.2f  d(centre) %.2f m",
				k, float64(k)/10, c.X, c.Y, r, zmin, zmax, psDist(c, centre))
		}
		if r < bestR {
			bestK, bestR, bestC = k, r, c
		}
	}
	t.Log("=== 1.5 MINIMUM ===")
	t.Logf("  k=%d (%.1f s avant T0) : rayon %.3f m, centroide (%.3f ; %.3f), a %.2f m du centre",
		bestK, float64(bestK)/10, bestR, bestC.X, bestC.Y, psDist(bestC, centre))
	t.Logf("  seuils ecrits avant mesure : rayon <= %.1f m ET centroide a <= 3 m du centre",
		psRayonOracle)
	if bestR <= psRayonOracle && psDist(bestC, centre) <= 3 {
		t.Logf("  ATTEINT : SOCLE MESURE en (%.3f ; %.3f)", bestC.X, bestC.Y)
	} else {
		t.Log("  NON ATTEINT : aucun point de croisement au centre")
	}
}

// psRemontee est le resultat de la remontee : le point de croisement et ses pieces.
type psRemontee struct {
	// C est le centroide au minimum de dispersion — LE SOCLE ; Z l altitude MOYENNE des
	// porteurs a cet instant. R le rayon a ce minimum, K le decalage en images entre le
	// ramassage et le `T0` d episode.
	C psPoint
	Z float32
	R float64
	K int
	// Garde / Total : episodes retenus (pas expliques par un lacher) sur episodes lus.
	Garde, Total int
	// Instants sont les ramassages DATES sur la grille du document (`T0 - K`), tries.
	Instants []int
}

// psSocleParRemontee applique la regle de la phase 1 : ecarter les ramassages expliques par
// un lacher, puis remonter la vie des porteurs jusqu'au minimum de dispersion.
//
// C'EST LA SEULE COPIE DE LA REGLE. La phase 2 appelle cette fonction plutot que de recopier
// le point (0,393 ; -0,012) : un litteral recopie ne suit pas un correctif de la regle.
func psSocleParRemontee(doc ReplayDocument) (psRemontee, bool) {
	var out psRemontee
	var garde []EquipmentEpisode
	for _, e := range doc.EquipmentEpisodes {
		if e.Fam != EquipFamilyOvershield {
			continue
		}
		out.Total++
		p, _, _, ok := psPosDeLaVie(doc, e.Slot, e.T0)
		if !ok {
			continue
		}
		if _, lache := psExpliqueParUnLacher(doc, e, p); lache {
			continue
		}
		garde = append(garde, e)
	}
	out.Garde = len(garde)
	if len(garde) < 2 {
		return out, false
	}
	out.K, out.R = -1, math.Inf(1)
	for k := 0; k <= psLagMax; k++ {
		pts, zs := psNuageA(doc, garde, k)
		if len(pts) < len(garde) {
			continue
		}
		c, r := psDispersion(pts)
		if r >= out.R {
			continue
		}
		var sz float64
		for _, z := range zs {
			sz += float64(z)
		}
		out.C, out.R, out.K = c, r, k
		out.Z = float32(sz / float64(len(zs)))
	}
	if out.K < 0 {
		return out, false
	}
	for _, e := range garde {
		out.Instants = append(out.Instants, e.T0-out.K)
	}
	sort.Ints(out.Instants)
	return out, out.R <= psRayonOracle
}

// psSocleMesure rend la position ET l altitude du socle mesurees par la phase 1, ou saute
// l etape appelante.
func psSocleMesure(t *testing.T) (psPoint, float32) {
	t.Helper()
	dir := psArtDir(t)
	doc, ok := psLoadDoc(t, dir, "01e1f945")
	if !ok {
		t.Skipf("artefact 01e1f945 absent de %s : le socle n est pas mesure", dir)
	}
	r, ok := psSocleParRemontee(doc)
	if !ok {
		t.Skipf("la remontee ne rend pas de socle (%d episodes retenus, rayon %.3f m)",
			r.Garde, r.R)
	}
	return r.C, r.Z
}
