package main

// png.go — LES PLANCHES DE CONTROLE : une par variable, sur le fond de carte publie, les
// noeuds peints a leur emprise (les niveaux hauts par-dessus les bas, comme en vue de
// dessus), les zones nommees en contour pour se reperer, et les positions retenues en
// contour sur la planche du score.
//
// Le calage vient du sidecar publie (replay.MapBackground.Calibration) :
//
//	xMonde = originX + (px + 0.5) * metersPerPixel ; yMonde = originY - (py + 0.5) * metersPerPixel
//
// C'est la DEUXIEME copie de ce peintre dans le depot (la premiere : cmd/mappower-build/png.go).
// A la troisieme, la regle du depot impose un helper partage avec garde-rail.

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log/slog"
	"math"
	"os"
	"sort"

	_ "golang.org/x/image/webp" // les fonds publies sont en WebP sans perte

	"levelup/go-api/internal/analysis/powerpos/geo"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// fondCarte rassemble ce qu'il faut pour projeter le monde sur l'image.
type fondCarte struct {
	img image.Image
	cal replay.MapBackgroundCalibration
}

// planche : une variable a peindre et sa couleur.
type planche struct {
	suffixe string
	teinte  func(geo.NoeudMesure) color.NRGBA
}

// EcrisPNG ecrit les planches d'une carte : `_sol`, `_H`, `_V`, `_E`, `_R`, `_M`, `_score`.
func EcrisPNG(ctx context.Context, res *title.PathResolver, titleSlug, base string, c *Cuite) error {
	fond, err := chargeFond(res, titleSlug, c.Cible.Carte)
	if err != nil {
		return err
	}
	zones := chargeZones(ctx, res, titleSlug, c.Cible)
	zMin, zMax := c.NiveauDeJeu-ProfondeurSousJeuM, c.NiveauDeJeu+HauteurSurJeuM
	comp := c.Resultat.Graphe.Composante
	planches := []planche{
		{"sol", func(n geo.NoeudMesure) color.NRGBA { return sequentiel((n.Z - zMin) / (zMax - zMin)) }},
		{"composantes", func(n geo.NoeudMesure) color.NRGBA { return teinteComposante(comp, c.Resultat.Graphe, n) }},
		{"H", func(n geo.NoeudMesure) color.NRGBA { return divergent(n.Norm.H*2 - 1) }},
		{"V", func(n geo.NoeudMesure) color.NRGBA { return sequentiel(n.Norm.V) }},
		{"E", func(n geo.NoeudMesure) color.NRGBA { return sequentiel(n.Norm.E) }},
		{"R", func(n geo.NoeudMesure) color.NRGBA { return sequentiel(n.Norm.R) }},
		{"M", func(n geo.NoeudMesure) color.NRGBA { return sequentiel(n.Norm.M) }},
	}
	for _, p := range planches {
		if err := encode(fmt.Sprintf("%s_%s.png", base, p.suffixe), peins(fond, zones, c.Resultat.Noeuds, p.teinte)); err != nil {
			return err
		}
	}
	scores := make([]float64, len(c.Resultat.Noeuds))
	for i, n := range c.Resultat.Noeuds {
		scores[i] = n.Score
	}
	lo, hi := quantiles(scores, 0.05)[0], quantiles(scores, 0.99)[0]
	toile := peins(fond, zones, c.Resultat.Noeuds, func(n geo.NoeudMesure) color.NRGBA {
		return sequentiel((n.Score - lo) / math.Max(hi-lo, 1e-9))
	})
	trait := color.NRGBA{R: 60, G: 255, B: 140, A: 230}
	for _, p := range c.Positions {
		tracePolygone(toile, fond.cal, p.Polygone, trait)
	}
	if err := encode(base+"_score.png", toile); err != nil {
		return err
	}
	slog.InfoContext(ctx, "mapgeo: planches ecrites", "carte", c.Cible.Carte, "base", base, "zones_nommees", len(zones))
	return nil
}

// chargeFond resout la cle du fond de la carte, lit son sidecar et son image.
func chargeFond(res *title.PathResolver, titleSlug, carte string) (fondCarte, error) {
	index, err := replay.BuildMapBackgroundIndex(res.MapBackgroundDir(titleSlug))
	if err != nil {
		return fondCarte{}, fmt.Errorf("index des fonds illisible : %w", err)
	}
	cle, ok := index.Lookup(carte)
	if !ok {
		return fondCarte{}, fmt.Errorf("aucun fond publie pour la carte %q", carte)
	}
	meta, err := replay.LoadMapBackground(res.MapBackgroundMetaPath(titleSlug, cle))
	if err != nil {
		return fondCarte{}, fmt.Errorf("sidecar de fond illisible (%s) : %w", cle, err)
	}
	if meta.Calibration.MetersPerPixel <= 0 {
		return fondCarte{}, fmt.Errorf("calage du fond %q sans echelle", cle)
	}
	f, err := os.Open(res.MapBackgroundImageFilePath(titleSlug, meta.Image))
	if err != nil {
		return fondCarte{}, fmt.Errorf("image de fond illisible (%s) : %w", meta.Image, err)
	}
	defer func() {
		if errFerme := f.Close(); errFerme != nil {
			slog.Error("mapgeo: fermeture de l'image de fond", "err", errFerme, "image", meta.Image)
		}
	}()
	img, _, err := image.Decode(f)
	if err != nil {
		return fondCarte{}, fmt.Errorf("image de fond indecodable (%s) : %w", meta.Image, err)
	}
	return fondCarte{img: img, cal: meta.Calibration}, nil
}

// chargeZones resout les zones nommees par le module, puis par le premier map_id. Leur
// absence n'est pas une erreur : l'image sort sans contours, et le journal le dit.
func chargeZones(ctx context.Context, res *title.PathResolver, titleSlug string, c *Cible) []replay.CalloutZone {
	cat, err := replay.LoadMapCallouts(res.MapCalloutsPath(titleSlug))
	if err != nil {
		slog.WarnContext(ctx, "mapgeo: catalogue de zones nommees illisible", "err", err)
		return nil
	}
	if e, err := cat.Lookup(c.Module); err == nil {
		return e.Zones
	}
	for _, id := range c.MapIDs {
		e, err := cat.LookupByID(id)
		if err == nil {
			return e.Zones
		}
		if !errors.Is(err, replay.ErrCalloutsUnknownMap) {
			slog.WarnContext(ctx, "mapgeo: lookup des zones par map_id en echec", "err", err, "map_id", id)
		}
	}
	slog.WarnContext(ctx, "mapgeo: carte sans zones nommees — planches sans contours", "carte", c.Carte)
	return nil
}

// peins dessine les noeuds (du plus bas au plus haut) sur une copie du fond, puis les zones.
func peins(fond fondCarte, zones []replay.CalloutZone, noeuds []geo.NoeudMesure,
	teinte func(geo.NoeudMesure) color.NRGBA) *image.RGBA {
	b := fond.img.Bounds()
	toile := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(toile, toile.Bounds(), &image.Uniform{C: color.NRGBA{R: 18, G: 18, B: 24, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(toile, toile.Bounds(), fond.img, b.Min, draw.Over)
	ordre := make([]int, len(noeuds))
	for i := range ordre {
		ordre[i] = i
	}
	sort.SliceStable(ordre, func(i, j int) bool { return noeuds[ordre[i]].Z < noeuds[ordre[j]].Z })
	const demiPas = 0.25
	for _, i := range ordre {
		n := noeuds[i]
		x0, y0 := versPixel(fond.cal, n.X-demiPas, n.Y+demiPas)
		x1, y1 := versPixel(fond.cal, n.X+demiPas, n.Y-demiPas)
		remplis(toile, x0, y0, x1, y1, teinte(n))
	}
	trait := color.NRGBA{R: 255, G: 255, B: 255, A: 110}
	for _, z := range zones {
		for _, poly := range append([][][2]float64{z.Polygon}, z.Parts...) {
			tracePolygone(toile, fond.cal, poly, trait)
		}
	}
	return toile
}

// teinteComposante colorie le rang de la composante ancree du noeud : la plus grande en
// vert, les suivantes en teintes tournantes vives — la fragmentation du sol se voit.
func teinteComposante(comp []int, g *geo.Graphe, n geo.NoeudMesure) color.NRGBA {
	idx := indexDuNoeud(g, n)
	if idx < 0 || idx >= len(comp) {
		return color.NRGBA{R: 255, A: 200}
	}
	switch r := comp[idx]; r {
	case 0:
		return color.NRGBA{R: 60, G: 200, B: 90, A: 170}
	default:
		teintes := []color.NRGBA{{R: 250, G: 80, B: 80, A: 230}, {R: 250, G: 200, B: 40, A: 230},
			{R: 80, G: 140, B: 255, A: 230}, {R: 230, G: 80, B: 230, A: 230}, {R: 40, G: 220, B: 220, A: 230}}
		return teintes[(r-1)%len(teintes)]
	}
}

// indexDuNoeud retrouve l'indice d'un noeud dans le graphe par sa cellule et son niveau.
func indexDuNoeud(g *geo.Graphe, n geo.NoeudMesure) int {
	for _, i := range g.NoeudsDeCellule(n.Index) {
		if g.Noeuds[i].Niveau == n.Niveau {
			return i
		}
	}
	return -1
}

// sequentiel colorie [0, 1] : sombre et transparent en bas, vif et opaque en haut.
func sequentiel(v float64) color.NRGBA {
	v = math.Max(0, math.Min(1, v))
	return color.NRGBA{R: uint8(40 + 215*v), G: uint8(30 + 60*v), B: uint8(120 - 80*v), A: uint8(60 + 190*v*v)}
}

// divergent colorie [-1, 1] : bleu en bas, rouge en haut, presque transparent au centre.
func divergent(v float64) color.NRGBA {
	v = math.Max(-1, math.Min(1, v))
	intensite := math.Abs(v)
	alpha := uint8(50 + 190*intensite)
	if v >= 0 {
		return color.NRGBA{R: 235, G: uint8(200 * (1 - intensite)), B: 40, A: alpha}
	}
	return color.NRGBA{R: 40, G: uint8(160 * (1 - intensite)), B: 235, A: alpha}
}

// versPixel applique l'inverse de la convention du sidecar.
func versPixel(cal replay.MapBackgroundCalibration, x, y float64) (int, int) {
	return int(math.Round((x-cal.OriginX)/cal.MetersPerPixel - 0.5)), int(math.Round((cal.OriginY-y)/cal.MetersPerPixel - 0.5))
}

// remplis peint un rectangle (bornes incluses), en respectant l'alpha.
func remplis(dst *image.RGBA, x0, y0, x1, y1 int, c color.NRGBA) {
	rect := image.Rect(min(x0, x1), min(y0, y1), max(x0, x1)+1, max(y0, y1)+1).Intersect(dst.Bounds())
	if !rect.Empty() {
		draw.Draw(dst, rect, &image.Uniform{C: c}, image.Point{}, draw.Over)
	}
}

// tracePolygone trace le contour ferme d'un polygone monde.
func tracePolygone(dst *image.RGBA, cal replay.MapBackgroundCalibration, polygone [][2]float64, c color.NRGBA) {
	for i := range polygone {
		if len(polygone) < 2 {
			return
		}
		a, b := polygone[i], polygone[(i+1)%len(polygone)]
		ax, ay := versPixel(cal, a[0], a[1])
		bx, by := versPixel(cal, b[0], b[1])
		segment(dst, ax, ay, bx, by, c)
	}
}

// segment trace une ligne (Bresenham entier).
func segment(dst *image.RGBA, x0, y0, x1, y1 int, c color.NRGBA) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sens(x0, x1), sens(y0, y1)
	err := dx + dy
	for {
		remplis(dst, x0, y0, x0, y0, c)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sens(de, vers int) int {
	if de < vers {
		return 1
	}
	return -1
}

// encode ecrit l'image.
func encode(chemin string, img image.Image) error {
	f, err := os.Create(chemin)
	if err != nil {
		return fmt.Errorf("creation du PNG (%s) : %w", chemin, err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return fmt.Errorf("encodage du PNG (%s) : %w", chemin, err)
	}
	return f.Close()
}
