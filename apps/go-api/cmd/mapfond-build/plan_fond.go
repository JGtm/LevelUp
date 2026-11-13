package main

// Le plan d'une carte : les arêtes de bord des composantes du maillage que la cuisson garde, et
// l'image plan qui se superpose au fond. Un seul calcul pour la cuisson et pour `-plan-seul`.

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/mapfondsidecar"

	"github.com/JGtm/halo-infinite-map-images/himap"
	"golang.org/x/image/webp"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

// planFond est le plan d'une carte : ses arêtes pour le sidecar, son image pour la superposition,
// et le sort des composantes du maillage dont il est tiré.
type planFond struct {
	Plan     *replaydoc.MapBackgroundPlan
	Image    *image.RGBA
	Retenues himap.Retenues
}

// calculePlan tire le plan d'un maillage dans le cadre `cal` du fond publié. Les composantes
// gardées sont celles que la cuisson garde : mêmes ancres, matière du fond publié (`matiere`).
// Les arêtes sont coupées au cadre et arrondies au millimètre. Nil, sans erreur, sans maillage.
func calculePlan(source string, m *hinavmesh.Maillage, ancres [][3]float64,
	cal himap.CalageFond, matiere func(x, y float64) bool) (*planFond, error) {
	if m == nil {
		return nil, nil
	}
	res := himap.ComposantesRetenues(m, ancres, matiere)
	murs, err := himap.PlanDesMurs(m, res.Garde)
	if err != nil {
		return nil, fmt.Errorf("plan des murs : %w", err)
	}
	murs = himap.PlanRogneAuCadre(murs, cal)
	img, err := himap.PlanPNG(m, res.Garde, cal)
	if err != nil {
		return nil, fmt.Errorf("image plan : %w", err)
	}
	return &planFond{
		Plan: &replaydoc.MapBackgroundPlan{
			Source: source, Vertices: murs.Sommets, Faces: murs.Faces, Edges: aretesAuMillimetre(murs.Aretes),
		},
		Image:    img,
		Retenues: res,
	}, nil
}

// aretesAuMillimetre arrondit les arêtes au millimètre : le sidecar reste d'une taille
// raisonnable sans que le trait se voie à aucune échelle du fond.
func aretesAuMillimetre(in [][4]float64) [][4]float64 {
	out := make([][4]float64, len(in))
	for i, a := range in {
		for k, v := range a {
			r := math.Round(v*millimetre) / millimetre
			if r == 0 {
				r = 0 // ecarte le zero negatif, que JSON ecrirait « -0 »
			}
			out[i][k] = r
		}
	}
	return out
}

// calageDuFond rend le calage du fond publié : celui du rendu, décalé du rognage à la matière.
func calageDuFond(rendu *himap.Rendu, rogne image.Rectangle) himap.CalageFond {
	x0, y1, mpp := himap.CalageDuRendu(rendu)
	x0, y1 = himap.CalageRogne(x0, y1, mpp, rogne)
	return himap.CalageFond{OrigineX: x0, OrigineY: y1, MetresParPixel: mpp, Largeur: rogne.Dx(), Hauteur: rogne.Dy()}
}

// matiereDeImage rend le prédicat « le fond porte de la matière en (x, y) monde » : un pixel
// d'alpha non nul. Hors du cadre : pas de matière.
func matiereDeImage(img *image.RGBA, cal himap.CalageFond) func(x, y float64) bool {
	b := img.Bounds()
	return func(x, y float64) bool {
		px := int(math.Floor((x - cal.OrigineX) / cal.MetresParPixel))
		py := int(math.Floor((cal.OrigineY - y) / cal.MetresParPixel))
		if px < 0 || py < 0 || px >= b.Dx() || py >= b.Dy() {
			return false
		}
		return img.Pix[img.PixOffset(b.Min.X+px, b.Min.Y+py)+3] != 0
	}
}

// compteMatiere compte les pixels de matière d'un fond.
func compteMatiere(img *image.RGBA) int {
	n := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0 {
			n++
		}
	}
	return n
}

// ecritImagePlan écrit `<cle>.plan.png` dans `dir` et rend son nom de fichier.
func ecritImagePlan(dir, cle string, img *image.RGBA) (string, error) {
	nom := mapfondsidecar.NomImagePlan(cle, ".png")
	if _, err := ecritPNG(filepath.Join(dir, nom), img); err != nil {
		return "", fmt.Errorf("écriture de l'image plan %s : %w", nom, err)
	}
	return nom, nil
}

// lisFondPublie décode l'image du fond d'un sidecar (PNG ou WebP, selon son champ `image`) en RGBA.
func lisFondPublie(dir string, s *mapfondsidecar.MapBackground) (*image.RGBA, error) {
	chemin := filepath.Join(dir, filepath.Base(s.Image))
	f, err := os.Open(chemin) //nolint:gosec // asset versionné, chemin résolu par PathResolver
	if err != nil {
		return nil, err
	}
	defer func() {
		if errClose := f.Close(); errClose != nil {
			slog.Warn("mapfond: fermeture du fond impossible", "err", errClose, "path", chemin)
		}
	}()
	var img image.Image
	if strings.EqualFold(filepath.Ext(chemin), ".webp") {
		img, err = webp.Decode(f)
	} else {
		img, _, err = image.Decode(f)
	}
	if err != nil {
		return nil, fmt.Errorf("fond %s illisible : %w", chemin, err)
	}
	rgba := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	return rgba, nil
}

// poseLePlan écrit l'image plan et pose plan et nom d'image dans le sidecar `s`. Un plan impossible
// ou une image impossible à écrire est journalisé : le fond reste publié sans plan.
func poseLePlan(ctx context.Context, dir, cle string, pf *planFond, s *mapfondsidecar.MapBackground) {
	s.Plan, s.PlanImage = nil, ""
	if pf == nil {
		return
	}
	nom, err := ecritImagePlan(dir, cle, pf.Image)
	if err != nil {
		slog.ErrorContext(ctx, "mapfond: image plan impossible, fond publie sans plan", "err", err, "carte", cle)
		return
	}
	s.Plan, s.PlanImage = pf.Plan, nom
}
