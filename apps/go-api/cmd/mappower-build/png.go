package main

// png.go — LES IMAGES DE CONTROLE. Une par signal, sur le fond de carte publie, avec les
// contours des zones nommees par-dessus pour se reperer.
//
// POURQUOI DES IMAGES ET PAS SEULEMENT DES CHIFFRES. Un CSV de 4 000 cellules ne dit pas
// si le signal dessine des LIEUX ou du bruit disperse. C'est la seule verification qui
// tienne avant de choisir une formule : le nuage doit se refermer sur des endroits qu'on
// reconnait (une tour, un couloir, un surplomb), pas saupoudrer la carte.
//
// LE CALAGE N'EST PAS DEVINE : il vient du sidecar publie a cote de l'image
// (replay.MapBackground.Calibration), et la convention y est ecrite en clair :
//
//	xMonde = originX + (px + 0.5) * metersPerPixel
//	yMonde = originY - (py + 0.5) * metersPerPixel
//
// — d'ou l'inversion utilisee ici. OriginY est le bord HAUT : l'image descend en Y quand le
// monde monte. C'est la MEME convention que `layers/mapBackground.ts` cote web.

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log/slog"
	"math"
	"os"

	_ "golang.org/x/image/webp" // les fonds publies sont en WebP sans perte

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// LES COULEURS D'HABILLAGE SONT DES `color.NRGBA`, JAMAIS DES `color.RGBA`. Le modele de
// `image/draw` est ALPHA-PREMULTIPLIE : un `color.RGBA{R:235, A:100}` est une couleur
// INVALIDE (composante superieure a l'alpha), et la composition en rend n'importe quoi —
// constate le 2026-09-20, la premiere planche de controle sortait verte. `color.NRGBA`
// porte des composantes non premultipliees et se convertit correctement.

// minEngagementsPNG est le nombre minimal d'eliminations touchant une cellule pour qu'elle
// soit PEINTE sur l'image du duel. Une cellule a un seul engagement ne vaut que 0 ou 1 :
// la peindre en extreme ferait un damier qui cache le signal.
const minEngagementsPNG = 6

// minMatchsPNG est le plancher de rarete applique a l'affichage, repris de
// `tactical.PlancherMatchsParCellule` : une cellule vue dans un seul match n'est pas un
// lieu, c'est une anecdote (mesure de cmd/mappos-build, 2026-08-30).
const minMatchsPNG = 3

// fondCarte rassemble ce qu'il faut pour projeter le monde sur l'image.
type fondCarte struct {
	img image.Image
	cal replay.MapBackgroundCalibration
}

// EcrisPNGDeControle produit les deux images d'une carte : `<base>_duel.png` et
// `<base>_occupation.png`.
func EcrisPNGDeControle(res *title.PathResolver, opts options, c *Cible,
	cellules []powerpos.Cellule, base string) error {
	fond, err := chargeFond(res, opts.titleSlug, c.Carte)
	if err != nil {
		return err
	}
	zones := chargeZones(res, opts.titleSlug, c.Carte, mapIDDominant(c))

	duel := peins(fond, zones, cellules, couleurDuel)
	if err := encode(base+"_duel.png", duel); err != nil {
		return err
	}
	occupation := peins(fond, zones, cellules, couleurOccupation)
	if err := encode(base+"_occupation.png", occupation); err != nil {
		return err
	}
	if err := encode(base+"_score.png", peinsScore(fond, zones, c)); err != nil {
		return err
	}
	// Les deux axes angulaires (v2), lus sur le DISQUE de chaque cellule scorable — c'est
	// l'echelle a laquelle le score les lit. Exposition = 1 - abri : on colorie ce qui se
	// voit sur le terrain (« par combien d'angles ce lieu se prend »).
	exposition := peinsAxe(fond, zones, c, func(s powerpos.CelluleScoree) (float64, bool) {
		return s.DirEntranteDisque.Dispersion(), s.DirEntranteDisque.N >= minDirectionsPNG
	})
	if err := encode(base+"_exposition.png", exposition); err != nil {
		return err
	}
	couverture := peinsAxe(fond, zones, c, func(s powerpos.CelluleScoree) (float64, bool) {
		return s.DirSortanteDisque.Dispersion(), s.DirSortanteDisque.N >= minDirectionsPNG
	})
	if err := encode(base+"_couverture.png", couverture); err != nil {
		return err
	}
	slog.Info("mappower: PNG de controle ecrits", "carte", c.Carte,
		"base", base, "zones_nommees", len(zones), "positions", len(c.Positions))
	return nil
}

// minDirectionsPNG : nombre de directions minimal dans le disque pour peindre un axe
// angulaire. Sous dix directions, la correction de biais rend une dispersion tronquee a
// zero ou a un plus souvent qu'une mesure ; on ne peint pas ce qu'on ne sait pas.
const minDirectionsPNG = 10

// peinsAxe colorie une grandeur dans [0, 1] lue sur les cellules scorables.
func peinsAxe(fond fondCarte, zones []replay.CalloutZone, c *Cible,
	valeur func(powerpos.CelluleScoree) (float64, bool)) *image.RGBA {
	cellules := make([]powerpos.Cellule, 0, len(c.Scorees))
	valeurs := make(map[[2]int]float64, len(c.Scorees))
	for _, s := range c.Scorees {
		v, ok := valeur(s)
		if !ok {
			continue
		}
		cellules = append(cellules, s.Cellule)
		valeurs[[2]int{s.Col, s.Lig}] = v
	}
	return peins(fond, zones, cellules, func(cel powerpos.Cellule) (color.NRGBA, bool) {
		return sequentiel(valeurs[[2]int{cel.Col, cel.Lig}]), true
	})
}

// peinsScore dessine le score FIGE par cellule, puis le CONTOUR des positions retenues.
// C'est la planche qui dit si la formule tombe sur des lieux ou sur du bruit.
//
// L'ECHELLE DE COULEUR EST ETALEE ENTRE LE p10 ET LE p99 DES SCORES DE LA CARTE : un score
// vit dans une bande etroite (0,50 a 0,65 en v2, 0,52 a 0,75 en v1) et une echelle
// absolue sur [0, 1] rendait une planche uniformement rose ou rien ne se lisait. C'est un
// choix d'AFFICHAGE : le score, ses seuils et la selection n'en dependent pas.
func peinsScore(fond fondCarte, zones []replay.CalloutZone, c *Cible) *image.RGBA {
	cellules := make([]powerpos.Cellule, 0, len(c.Scorees))
	scores := make(map[[2]int]float64, len(c.Scorees))
	tous := make([]float64, 0, len(c.Scorees))
	for _, s := range c.Scorees {
		cellules = append(cellules, s.Cellule)
		scores[[2]int{s.Col, s.Lig}] = s.Score
		tous = append(tous, s.Score)
	}
	bas, haut := powerpos.Quantile(tous, 0.10), powerpos.Quantile(tous, 0.99)
	toile := peins(fond, zones, cellules, func(cel powerpos.Cellule) (color.NRGBA, bool) {
		v := scores[[2]int{cel.Col, cel.Lig}]
		if haut > bas {
			v = (v - bas) / (haut - bas)
		}
		return sequentiel(v), true
	})
	trait := color.NRGBA{R: 60, G: 255, B: 140, A: 230}
	for _, p := range c.Positions {
		tracePolygone(toile, fond.cal, p.Polygone, trait)
	}
	return toile
}

// sequentiel colorie un score de [0, 1] : sombre et transparent en bas, vif et opaque en
// haut. Une echelle sequentielle et non divergente — le score n'a pas de zero naturel.
func sequentiel(v float64) color.NRGBA {
	v = math.Max(0, math.Min(1, v))
	return color.NRGBA{
		R: uint8(40 + 215*v), G: uint8(30 + 60*v), B: uint8(120 - 80*v),
		A: uint8(40 + 200*v*v),
	}
}

// tracePolygone trace le contour ferme d'un polygone monde.
func tracePolygone(dst *image.RGBA, cal replay.MapBackgroundCalibration,
	polygone [][2]float64, c color.NRGBA) {
	if len(polygone) < 2 {
		return
	}
	for i := range polygone {
		a, b := polygone[i], polygone[(i+1)%len(polygone)]
		ax, ay := versPixel(cal, a[0], a[1])
		bx, by := versPixel(cal, b[0], b[1])
		segment(dst, ax, ay, bx, by, c)
	}
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
			slog.Error("mappower: fermeture de l'image de fond", "err", errFerme, "image", meta.Image)
		}
	}()
	img, _, err := image.Decode(f)
	if err != nil {
		return fondCarte{}, fmt.Errorf("image de fond indecodable (%s) : %w", meta.Image, err)
	}
	return fondCarte{img: img, cal: meta.Calibration}, nil
}

// chargeZones resout les zones nommees de la carte par la MEME cascade que le service
// (`service.zonesPourIdentites`) : module installe d'abord, asset UGC ensuite. L'ordre
// compte — une carte integree a des polygones de meilleure qualite, une carte Forge n'a pas
// de module et n'est atteignable que par son map_id.
//
// Leur absence n'est pas une erreur : l'image sort sans contours.
func chargeZones(res *title.PathResolver, titleSlug, carte, mapID string) []replay.CalloutZone {
	cat, err := replay.LoadMapCallouts(res.MapCalloutsPath(titleSlug))
	if err != nil {
		slog.Warn("mappower: catalogue de zones nommees illisible", "err", err)
		return nil
	}
	if zones, ok := zonesParModule(res, titleSlug, carte, cat); ok {
		return zones
	}
	entree, err := cat.LookupByID(mapID)
	if err != nil {
		if !errors.Is(err, replay.ErrCalloutsUnknownMap) {
			slog.Warn("mappower: lookup des zones par map_id en echec", "err", err, "map_id", mapID)
		}
		slog.Warn("mappower: carte sans zones nommees — image sans contours",
			"carte", carte, "map_id", mapID)
		return nil
	}
	return entree.Zones
}

// zonesParModule tente l'essai 1 : nom de carte -> module -> zones. La resolution du
// module est partagee avec la sortie JSON des positions (`moduleDeCarte`) : une seule
// lecture du catalogue de bornes fait autorite sur ce rattachement.
func zonesParModule(res *title.PathResolver, titleSlug, carte string,
	cat *replay.MapCalloutsCatalog) ([]replay.CalloutZone, bool) {
	module := moduleDeCarte(res, titleSlug, carte)
	if module == "" {
		return nil, false
	}
	zones, err := cat.Lookup(module)
	if err != nil {
		return nil, false
	}
	return zones.Zones, true
}

// mapIDDominant rend le map_id le plus frequent parmi les matchs de la cible — une carte
// republiee au fil des saisons en porte plusieurs, et c'est celui du gros du corpus qui a
// le plus de chances d'etre au catalogue.
func mapIDDominant(c *Cible) string {
	comptes := map[string]int{}
	for _, m := range c.Matchs {
		if m.MapID != "" {
			comptes[m.MapID]++
		}
	}
	meilleur, meilleurN := "", 0
	for id, n := range comptes {
		if n > meilleurN || (n == meilleurN && id < meilleur) {
			meilleur, meilleurN = id, n
		}
	}
	return meilleur
}

// peins dessine les cellules sur une copie du fond, puis les contours des zones.
func peins(fond fondCarte, zones []replay.CalloutZone, cellules []powerpos.Cellule,
	teinte func(powerpos.Cellule) (color.NRGBA, bool)) *image.RGBA {
	b := fond.img.Bounds()
	toile := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(toile, toile.Bounds(), &image.Uniform{C: color.NRGBA{R: 18, G: 18, B: 24, A: 255}},
		image.Point{}, draw.Src)
	draw.Draw(toile, toile.Bounds(), fond.img, b.Min, draw.Over)

	demiPas := 0.25 // moitie du pas de 0,5 m : les cellules sont peintes a leur emprise
	for _, c := range cellules {
		col, ok := teinte(c)
		if !ok {
			continue
		}
		x0, y0 := versPixel(fond.cal, c.CentreX-demiPas, c.CentreY+demiPas)
		x1, y1 := versPixel(fond.cal, c.CentreX+demiPas, c.CentreY-demiPas)
		remplis(toile, x0, y0, x1, y1, col)
	}
	contourZones(toile, fond.cal, zones)
	return toile
}

// couleurDuel colorie le rapport kills_depuis / (kills_depuis + morts_dedans).
func couleurDuel(c powerpos.Cellule) (color.NRGBA, bool) {
	n := c.TotalEngagements()
	if n < minEngagementsPNG || c.MatchsKills < minMatchsPNG {
		return color.NRGBA{}, false
	}
	ratio := float64(c.KillsDepuis) / float64(n)
	return divergent(2*ratio - 1), true
}

// couleurOccupation colorie l'ecart d'occupation gagnants - perdants, normalise par le
// temps total passe dans la cellule (sinon les cellules de passage, tenues par tout le
// monde, dominent l'echelle par leur seul volume).
func couleurOccupation(c powerpos.Cellule) (color.NRGBA, bool) {
	total := c.TotalOccupationMS()
	if total <= 0 || c.MatchsPresence < minMatchsPNG {
		return color.NRGBA{}, false
	}
	return divergent((c.OccupationGagnantsMS - c.OccupationPerdantsMS) / total), true
}

// divergent rend une couleur pour une valeur dans [-1, 1] : bleu en bas, rouge en haut,
// transparent au centre — une cellule neutre ne doit pas masquer le fond.
func divergent(v float64) color.NRGBA {
	v = math.Max(-1, math.Min(1, v))
	intensite := math.Abs(v)
	alpha := uint8(60 + 175*intensite)
	if v >= 0 {
		return color.NRGBA{R: 235, G: uint8(200 * (1 - intensite)), B: 40, A: alpha}
	}
	return color.NRGBA{R: 40, G: uint8(160 * (1 - intensite)), B: 235, A: alpha}
}

// versPixel applique l'inverse de la convention du sidecar.
func versPixel(cal replay.MapBackgroundCalibration, x, y float64) (int, int) {
	px := (x-cal.OriginX)/cal.MetersPerPixel - 0.5
	py := (cal.OriginY-y)/cal.MetersPerPixel - 0.5
	return int(math.Round(px)), int(math.Round(py))
}

// remplis peint un rectangle (bornes incluses), en respectant l'alpha.
func remplis(dst *image.RGBA, x0, y0, x1, y1 int, c color.NRGBA) {
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	rect := image.Rect(x0, y0, x1+1, y1+1).Intersect(dst.Bounds())
	if rect.Empty() {
		return
	}
	draw.Draw(dst, rect, &image.Uniform{C: c}, image.Point{}, draw.Over)
}

// contourZones trace le contour de chaque zone nommee, pour se reperer.
func contourZones(dst *image.RGBA, cal replay.MapBackgroundCalibration, zones []replay.CalloutZone) {
	trait := color.NRGBA{R: 255, G: 255, B: 255, A: 110}
	for _, z := range zones {
		polygones := append([][][2]float64{z.Polygon}, z.Parts...)
		for _, poly := range polygones {
			tracePolygone(dst, cal, poly, trait)
		}
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
		if errFerme := f.Close(); errFerme != nil {
			slog.Error("mappower: fermeture du PNG apres echec", "err", errFerme, "path", chemin)
		}
		return fmt.Errorf("encodage du PNG (%s) : %w", chemin, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("fermeture du PNG (%s) : %w", chemin, err)
	}
	return nil
}
