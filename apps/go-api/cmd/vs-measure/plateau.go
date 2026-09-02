package main

// plateau.go — sous-commande `plateau` du driver jetable vs-measure (Warthog arme, 2026-09-02).
//
// Pose l'arme arriere d'un Warthog AU CENTRE DU PLATEAU (le grand rectangle blanc sans trait a
// l'arriere du chassis, consigne utilisateur), au lieu du noeud d'attache n[006] rejete :
//
//  1. rend le chassis seul (permutation `default` de chaque region) au canevas fixe ;
//  2. detecte, dans ce rendu, la plus grande composante connexe de pixels BLANCS SANS TRAIT dans
//     la moitie ARRIERE. CORRECTION UTILISATEUR (V2, 2026-09-02) : pour la famille Warthog
//     +X = AVANT (capot en V, pare-chocs anguleux) et -X = ARRIERE (grand rectangle du plateau) ;
//     dans le rendu (remap {Y, -X}) l'arriere est donc en HAUT de l'image. La moitie fouillee
//     est X local * xplateau > 0 (defaut -xplateau=-1 : X < 0, haut de l'image). Rectangle
//     englobant -> centre (cx, cy) en pixels -> coordonnees locales (X, Y) ; Z = mediane ;
//  3. pour chaque arme : centre de sa base (centroide XY de la tranche la plus basse) ->
//     translation qui l'amene sur (X, Y) ; en Z, base posee sur le plateau. Les armes sont
//     authored canon vers +X (= vers l'avant du Warthog, par-dessus l'habitacle) : c'est
//     l'orientation AU REPOS de la LAAG et du Gauss, elle est CONSERVEE (V3, correction
//     utilisateur 2026-09-02 : la rotation de 180 degres autour de Z de V2 etait une
//     sur-interpretation). -rotarme (defaut false, garde pour trace) la ferait pivoter de 180
//     degres autour de Z via l'IMAGE de l'arme rendue au canevas fixe (canevas symetrique
//     autour de l'origine locale : pivoter l'image = (X, Y) -> (-X, -Y) exactement), himap
//     n'offrant qu'une translation par piece ;
//  4. assemblage en ORDRE PEINTRE (chassis, puis arme au-dessus), rognage du composite ;
//  5. -rot180 : rotation finale de 180 degres du sprite dans le plan de l'image (rotation, pas
//     miroir : la chiralite du Gauss est conservee) -> nez (+X) EN HAUT, arme en bas, comme les
//     autres vehicules du lot. -pivote=fichiers : applique la meme rotation a des PNG existants
//     (razorback.png, meme chassis, valide en forme).
//
// Sorties (dans -out) : chassis.png, chassis_detect.png (composantes candidates), et par arme
// arme_<nom>_isolee.png, arme_<nom>_posee.png, <nom>.png (sprite final, pivote si -rot180),
// <nom>_detect.png (assemblage non pivote + rectangle et centres), <nom>_detect_rot.png (idem,
// pivote). Toutes les mesures sont imprimees (lignes PLATEAU / ARME / PIVOTE).
import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"levelup/go-api/internal/himap"
)

// pivotTourelle : noeud partage par les enfants warthog_g/b_g (axe de rotation de l'arme).
const pivotTourelle = 0x99d45ed9

type plateauCfg struct {
	out   string
	cadre float64
	cell  float64
	epais float64 // epaisseur (m) de la tranche basse qui definit la base d'une arme
	marge int     // marge (px) du rognage des composites
	erode int     // erosions (px) du masque blanc avant etiquetage des composantes
	// seuilUnion : les composantes dont l'aire atteint cette fraction de la plus grande sont
	// reunies dans le plateau (deux zones voisines separees par de simples traits).
	seuilUnion float64
	// xplateau : signe de X local de la moitie ou chercher le plateau (-1 = X < 0 = haut de
	// l'image = ARRIERE de la famille Warthog ; +1 = X > 0 = bas = avant, V1 erronee).
	xplateau float64
	rotArme  bool // pivoter l'arme de 180 degres autour de Z avant la pose (V2, refuse : defaut false)
	rot180   bool // pivoter le sprite final de 180 degres dans le plan de l'image (nez en haut)
}

// canevas : un rendu au cadre fixe et son PNG, avec les conversions pixel <-> repere local.
type canevas struct {
	r   *himap.Rendu
	img *image.NRGBA
}

// local rend (X, Y) du repere local du vehicule au centre du pixel (px, py) de l'image.
// Remap objet_isole : Xr = Y, Yr = -X ; l'image inverse Y (py = NY-1-j).
func (c canevas) local(px, py int) (x, y float64) {
	j := c.r.NY - 1 - py
	xr := c.r.Min[0] + (float64(px)+0.5)*c.r.Cell
	yr := c.r.Min[1] + (float64(j)+0.5)*c.r.Cell
	return -yr, xr
}

// pixel rend la position (px, py) flottante d'un point local (X, Y) — inverse de local.
func (c canevas) pixel(x, y float64) (px, py float64) {
	xr, yr := y, -x
	i := (xr-c.r.Min[0])/c.r.Cell - 0.5
	j := (yr-c.r.Min[1])/c.r.Cell - 0.5
	return i, float64(c.r.NY-1) - j
}

// symetrique verifie que l'origine locale est au centre exact de la grille : condition pour que
// pivoter l'image de 180 degres equivale a (X, Y) -> (-X, -Y).
func (c canevas) symetrique() bool {
	e0 := c.r.Min[0] + float64(c.r.NX)*c.r.Cell/2
	e1 := c.r.Min[1] + float64(c.r.NY)*c.r.Cell/2
	return math.Abs(e0) < 1e-9 && math.Abs(e1) < 1e-9
}

type armeSpec struct {
	nom string
	id  uint32
}

func plateauMain(args []string) {
	fs := flag.NewFlagSet("plateau", flag.ExitOnError)
	mods := fs.String("modules", "", "modules a ouvrir (basenames ou variant:basename, virgule)")
	variant := fs.String("variant", "any", "variante deploy: any|pc|ds")
	chassis := fs.String("chassis", "0x561f2ca7", "mode du chassis (hex)")
	armes := fs.String("armes", "", "armes a poser : nom=0xHEX,... (mode de l'objet-enfant)")
	out := fs.String("out", ".", "dossier de sortie")
	cadre := fs.Float64("cadre", 5, "demi-emprise du canevas fixe (m)")
	cellmm := fs.Float64("cellmm", 10, "mm/pixel")
	epais := fs.Float64("epais", 0.08, "tranche basse (m) definissant la base d'une arme")
	marge := fs.Int("marge", 6, "marge (px) du rognage")
	erode := fs.Int("erode", 1, "erosions (px) du masque blanc avant etiquetage")
	seuilUnion := fs.Float64("union", 0.6, "fraction de l'aire de la 1re composante a partir de laquelle une composante rejoint le plateau")
	xplateau := fs.Float64("xplateau", -1, "signe de X local de la moitie fouillee (-1 = X<0 = haut de l'image = arriere Warthog)")
	rotArme := fs.Bool("rotarme", false, "pivoter l'arme de 180 degres autour de Z avant la pose (canon vers -X ; V2 refusee, l'orientation authored canon vers +X = avant est la bonne)")
	rot180 := fs.Bool("rot180", true, "pivoter le sprite final de 180 degres (nez +X en haut)")
	pivote := fs.String("pivote", "", "PNG existants a pivoter de 180 degres vers -out (virgule)")
	_ = fs.Parse(args)

	chemins, err := cheminsModules(*variant, listeModules(*mods))
	must(err)
	fmt.Printf("ouverture de %d modules...\n", len(chemins))
	idx, err := himap.NewModuleIndex(chemins...)
	must(err)
	must(os.MkdirAll(*out, 0o755))
	cfg := plateauCfg{out: *out, cadre: *cadre, cell: *cellmm / 1000.0, epais: *epais, marge: *marge, erode: *erode,
		seuilUnion: *seuilUnion, xplateau: *xplateau, rotArme: *rotArme, rot180: *rot180}

	ids := splitHex(*chassis)
	if len(ids) != 1 {
		must(fmt.Errorf("un seul chassis attendu"))
	}
	ch, plat := traiteChassis(idx, ids[0], cfg)
	for _, a := range parseArmes(*armes) {
		traiteArme(idx, a, ch, plat, cfg)
	}
	for _, p := range strings.Split(*pivote, ",") {
		if p = strings.TrimSpace(p); p != "" {
			pivoteFichier(p, cfg)
		}
	}
}

func parseArmes(spec string) []armeSpec {
	var out []armeSpec
	for _, s := range strings.Split(spec, ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		i := strings.IndexByte(s, '=')
		if i <= 0 {
			fmt.Printf("arme %q illisible (attendu nom=0xHEX)\n", s)
			continue
		}
		v, err := strconv.ParseUint(strings.TrimSpace(s[i+1:]), 0, 32)
		if err != nil {
			fmt.Printf("arme %q : id illisible (%v)\n", s, err)
			continue
		}
		out = append(out, armeSpec{nom: strings.TrimSpace(s[:i]), id: uint32(v)})
	}
	return out
}

// traiteChassis rend le chassis `default` au canevas fixe, detecte le plateau et ecrit
// chassis.png + chassis_detect.png. Rend le canevas (fond des composites) et le plateau retenu.
func traiteChassis(idx *himap.ModuleIndex, id uint32, cfg plateauCfg) (canevas, plateau) {
	tag, blob, err := idx.ExtractWithResources(id)
	must(err)
	asset, err := himap.NewRenderModelAsset(tag, blob)
	must(err)
	regions, err := himap.ModeRegions(tag)
	must(err)
	base := sectionsDeVariante(regions, permDefault)
	ch := rendCanevas([]himap.PartAssemblage{{Asset: asset, SectionsChoisies: base}}, cfg)
	imprimeMesure("chassis_default", asset, base)
	ecrisPNG(filepath.Join(cfg.out, "chassis.png"), ch.img)
	ox, oy := ch.pixel(0, 0)
	e := emprise(ch.img, 0)
	fmt.Printf("CANEVAS %dx%d cell=%.4f m Min=(%+.4f,%+.4f) ; origine locale (0,0) au pixel (%.1f,%.1f) ; symetrique=%v ; chassis opaque %dx%d px = x[%d..%d] y[%d..%d]\n",
		ch.r.NX, ch.r.NY, ch.r.Cell, ch.r.Min[0], ch.r.Min[1], ox, oy, ch.symetrique(), e.Dx(), e.Dy(), e.Min.X, e.Max.X-1, e.Min.Y, e.Max.Y-1)

	cands := composantesBlanches(ch, cfg.erode, cfg.xplateau)
	if len(cands) == 0 {
		must(fmt.Errorf("aucune composante blanche dans la moitie X*%.0f > 0", cfg.xplateau))
	}
	for i, c := range cands {
		if i >= 5 {
			break
		}
		fmt.Printf("CANDIDAT %d aire=%d px rect=x[%d..%d] y[%d..%d] (%dx%d px) centre=(%.1f,%.1f) px -> local X=%+.3f Y=%+.3f ; centroide local X=%+.3f Y=%+.3f ; Z mediane=%.3f\n",
			i+1, c.aire, c.rect.Min.X, c.rect.Max.X-1, c.rect.Min.Y, c.rect.Max.Y-1, c.rect.Dx(), c.rect.Dy(),
			c.cx, c.cy, c.X, c.Y, c.gX, c.gY, c.Z)
	}
	plat, nMembres := unionPlateau(ch, cands, cfg.seuilUnion)
	fmt.Printf("PLATEAU retenu (moitie X*%.0f > 0) = reunion des %d plus grandes composantes (aire >= %.0f%% de la 1re) : centre (cx,cy)=(%.1f,%.1f) px = local (X=%+.3f, Y=%+.3f), Z=%.3f (max des medianes), rect %.2f x %.2f m = x[%d..%d] y[%d..%d]\n",
		cfg.xplateau, nMembres, cfg.seuilUnion*100, plat.cx, plat.cy, plat.X, plat.Y, plat.Z,
		float64(plat.rect.Dx())*cfg.cell, float64(plat.rect.Dy())*cfg.cell,
		plat.rect.Min.X, plat.rect.Max.X-1, plat.rect.Min.Y, plat.rect.Max.Y-1)

	ov := cloneNRGBA(ch.img)
	couleurs := []color.NRGBA{{255, 140, 0, 255}, {200, 60, 200, 255}, {40, 170, 220, 255}}
	for i := len(couleurs) - 1; i >= 0; i-- {
		if i < len(cands) {
			dessineRect(ov, cands[i].rect, couleurs[i])
			dessineCroix(ov, cands[i].cx, cands[i].cy, 4, couleurs[i])
		}
	}
	vert := color.NRGBA{0, 200, 80, 255}
	dessineRect(ov, plat.rect, vert)
	dessineRect(ov, plat.rect.Inset(-1), vert)
	dessineCroix(ov, plat.cx, plat.cy, 7, vert)
	ecrisPNG(filepath.Join(cfg.out, "chassis_detect.png"), rogne(ov, emprise(ov, cfg.marge)))
	ecrisPNG(filepath.Join(cfg.out, "chassis_rogne.png"), rogne(ch.img, emprise(ch.img, cfg.marge)))
	return ch, plat
}

// mesuresArme : ce qu'on mesure sur une arme isolee (repere propre de l'enfant).
type mesuresArme struct {
	zmin, zmax float64
	bbMin      [2]float64 // emprise XY
	bbMax      [2]float64
	base       [2]float64 // centroide XY de la tranche basse [zmin, zmin+epais]
	nBase      int
	pivot      [3]float64
	pivotOK    bool
}

// mesureArme parcourt tous les sommets : emprise, Z min/max, centroide de la tranche basse ;
// et cherche le pivot de tourelle dans le squelette.
func mesureArme(tag []byte, asset *himap.RuntimeGeoAsset, epais float64) mesuresArme {
	m := mesuresArme{zmin: math.Inf(1), zmax: math.Inf(-1),
		bbMin: [2]float64{math.Inf(1), math.Inf(1)}, bbMax: [2]float64{math.Inf(-1), math.Inf(-1)}}
	for i := 0; i < asset.MeshCount(); i++ {
		mesh := asset.Mesh(i)
		if mesh == nil {
			continue
		}
		for _, v := range mesh.Vertices {
			m.zmin, m.zmax = math.Min(m.zmin, v[2]), math.Max(m.zmax, v[2])
			for a := 0; a < 2; a++ {
				m.bbMin[a], m.bbMax[a] = math.Min(m.bbMin[a], v[a]), math.Max(m.bbMax[a], v[a])
			}
		}
	}
	var sx, sy float64
	for i := 0; i < asset.MeshCount(); i++ {
		mesh := asset.Mesh(i)
		if mesh == nil {
			continue
		}
		for _, v := range mesh.Vertices {
			if v[2] <= m.zmin+epais {
				sx, sy = sx+v[0], sy+v[1]
				m.nBase++
			}
		}
	}
	if m.nBase > 0 {
		m.base = [2]float64{sx / float64(m.nBase), sy / float64(m.nBase)}
	}
	if nds, err := himap.ModeNodes(tag); err == nil {
		for i, n := range nds {
			if n.Name == pivotTourelle {
				m.pivot, m.pivotOK = himap.NodeModelTransform(nds, i).Trans, true
			}
		}
	}
	return m
}

// poseArme : la pose d'une arme = rotation optionnelle de 180 degres autour de Z (s = -1),
// puis translation tEff dans le repere du chassis. tRendu est la translation passee a himap
// AVANT la rotation d'image : pivoter l'image envoie p + tRendu sur -(p + tRendu), qui doit
// valoir s*p + tEff -> tRendu = -tEff (XY) quand s = -1. En Z rien ne tourne.
type poseArme struct {
	s      float64 // +1 : arme telle quelle ; -1 : pivotee de 180 degres autour de Z
	tEff   [3]float64
	tRendu [3]float64
}

func calculePose(m mesuresArme, plat plateau, rot bool) poseArme {
	p := poseArme{s: 1}
	if rot {
		p.s = -1
	}
	// s*base + tEff = plat.
	p.tEff = [3]float64{plat.X - p.s*m.base[0], plat.Y - p.s*m.base[1], plat.Z - m.zmin}
	p.tRendu = [3]float64{p.s * p.tEff[0], p.s * p.tEff[1], p.tEff[2]}
	return p
}

// emprisePosee rend l'intervalle [lo, hi] d'une composante XY apres pose (s, t).
func emprisePosee(lo, hi, s, t float64) (float64, float64) {
	if s < 0 {
		return -hi + t, -lo + t
	}
	return lo + t, hi + t
}

// traiteArme mesure l'arme, calcule la pose (rotation Z optionnelle + translation vers le
// centre du plateau), rend l'arme isolee et posee, compose en ordre peintre sur le chassis,
// ecrit le sprite (pivote si -rot180) et ses versions de controle.
func traiteArme(idx *himap.ModuleIndex, a armeSpec, ch canevas, plat plateau, cfg plateauCfg) {
	tag, blob, err := idx.ExtractWithResources(a.id)
	if err != nil {
		fmt.Printf("ARME %s %#08x extraction KO: %v\n", a.nom, a.id, err)
		return
	}
	asset, err := himap.NewRenderModelAsset(tag, blob)
	if err != nil {
		fmt.Printf("ARME %s %#08x render_model KO: %v\n", a.nom, a.id, err)
		return
	}
	m := mesureArme(tag, asset, cfg.epais)
	p := calculePose(m, plat, cfg.rotArme)
	fmt.Printf("ARME %s mode=%#08x emprise X[%+.3f..%+.3f] Y[%+.3f..%+.3f] Z[%+.3f..%+.3f] ; base (tranche %.2f m, %d sommets) centre=(%+.3f,%+.3f) ; emprise-centre=(%+.3f,%+.3f)",
		a.nom, a.id, m.bbMin[0], m.bbMax[0], m.bbMin[1], m.bbMax[1], m.zmin, m.zmax, cfg.epais, m.nBase,
		m.base[0], m.base[1], (m.bbMin[0]+m.bbMax[0])/2, (m.bbMin[1]+m.bbMax[1])/2)
	if m.pivotOK {
		fmt.Printf(" ; pivot %#08x=(%+.3f,%+.3f,%+.3f)", uint32(pivotTourelle), m.pivot[0], m.pivot[1], m.pivot[2])
	}
	x0, x1 := emprisePosee(m.bbMin[0], m.bbMax[0], p.s, p.tEff[0])
	y0, y1 := emprisePosee(m.bbMin[1], m.bbMax[1], p.s, p.tEff[1])
	fmt.Printf("\nARME %s POSE : rotation Z=%s ; T=(%+.3f, %+.3f, %+.3f) m (apres rotation ; T rendu avant pivot d'image = (%+.3f, %+.3f)) ; apres pose : X[%+.3f..%+.3f] Y[%+.3f..%+.3f] Z[%+.3f..%+.3f]\n",
		a.nom, map[bool]string{true: "180 deg (canon vers -X = arriere)", false: "aucune (orientation authored, canon vers +X = avant)"}[p.s < 0],
		p.tEff[0], p.tEff[1], p.tEff[2], p.tRendu[0], p.tRendu[1],
		x0, x1, y0, y1, m.zmin+p.tEff[2], m.zmax+p.tEff[2])

	iso := rendCanevas([]himap.PartAssemblage{{Asset: asset}}, cfg)
	ecrisPNG(filepath.Join(cfg.out, "arme_"+a.nom+"_isolee.png"), rogne(iso.img, emprise(iso.img, cfg.marge)))
	pose := rendCanevas([]himap.PartAssemblage{{Asset: asset, Translation: p.tRendu}}, cfg)
	poseImg := pose.img
	if p.s < 0 {
		if !ch.symetrique() {
			must(fmt.Errorf("canevas non symetrique : la rotation d'image ne vaut pas (X,Y)->(-X,-Y)"))
		}
		poseImg = pivote180(pose.img)
	}
	ecrisPNG(filepath.Join(cfg.out, "arme_"+a.nom+"_posee.png"), poseImg)

	comp := cloneNRGBA(ch.img)
	superposeNRGBA(comp, poseImg)
	cadre := emprise(comp, cfg.marge)
	final := rogne(comp, cadre)
	if cfg.rot180 {
		final = pivote180(final)
	}
	ecrisPNG(filepath.Join(cfg.out, a.nom+".png"), final)

	// Controle : centre de base rendu -> pixel -> pivote comme l'image (independant de tEff).
	bx, by := ch.pixel(m.base[0]+p.tRendu[0], m.base[1]+p.tRendu[1])
	if p.s < 0 {
		bx, by = float64(ch.r.NX-1)-bx, float64(ch.r.NY-1)-by
	}
	ov := cloneNRGBA(comp)
	dessineRect(ov, plat.rect, color.NRGBA{0, 200, 80, 255})
	dessineCroix(ov, plat.cx, plat.cy, 6, color.NRGBA{0, 200, 80, 255})
	dessineCroix(ov, bx, by, 4, color.NRGBA{230, 30, 30, 255})
	ecrisPNG(filepath.Join(cfg.out, a.nom+"_detect.png"), rogne(ov, cadre))
	ecrisPNG(filepath.Join(cfg.out, a.nom+"_detect_rot.png"), pivote180(rogne(ov, cadre)))
	fmt.Printf("ARME %s CONTROLE : centre base pose au pixel (%.1f,%.1f) vs centre plateau (%.1f,%.1f) : ecart (%.2f,%.2f) cellules ; sprite %dx%d px (rogne x[%d..%d] y[%d..%d]) ; rot180 final=%v\n",
		a.nom, bx, by, plat.cx, plat.cy, bx-plat.cx, by-plat.cy, cadre.Dx(), cadre.Dy(), cadre.Min.X, cadre.Max.X-1, cadre.Min.Y, cadre.Max.Y-1, cfg.rot180)
}

// pivoteFichier lit un PNG existant, le pivote de 180 degres et l'ecrit sous -out/<base>.
func pivoteFichier(chemin string, cfg plateauCfg) {
	f, err := os.Open(chemin)
	must(err)
	defer f.Close()
	src, err := png.Decode(f)
	must(err)
	b := src.Bounds()
	im := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			im.Set(x, y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	e := emprise(im, 0)
	fmt.Printf("PIVOTE %s : %dx%d px, opaque %dx%d px -> rotation 180 deg\n", filepath.Base(chemin), b.Dx(), b.Dy(), e.Dx(), e.Dy())
	ecrisPNG(filepath.Join(cfg.out, filepath.Base(chemin)), pivote180(im))
}

// rendCanevas rasterise des pieces au canevas fixe (-cadre, -cellmm) et encode le PNG.
func rendCanevas(parts []himap.PartAssemblage, cfg plateauCfg) canevas {
	mn := [2]float64{-cfg.cadre, -cfg.cadre}
	mx := [2]float64{cfg.cadre, cfg.cadre}
	o := himap.OptionsSprite{AxeHaut: himap.HautZ, CellMetres: cfg.cell, CadreMin: &mn, CadreMax: &mx}
	r, err := himap.RenduAssemblage(parts, o)
	must(err)
	return canevas{r: r, img: himap.SpriteObjetPNG(r, o)}
}

func ecrisPNG(chemin string, img image.Image) {
	f, err := os.Create(chemin)
	must(err)
	defer f.Close()
	must(png.Encode(f, img))
	b := img.Bounds()
	fmt.Printf("  -> %s (%dx%d)\n", filepath.Base(chemin), b.Dx(), b.Dy())
}
