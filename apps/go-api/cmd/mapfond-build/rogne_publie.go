package main

// Le rognage du fond PUBLIÉ à la silhouette du maillage, sans recuisson (`-plan-seul -rogne-publie`).
//
// C'est le « fond publié croisé avec la silhouette du maillage » : la matière du fond déjà publié
// qui sort du masque est effacée (alpha 0). Le masque est celui de la cuisson en mode silhouette :
// les composantes retenues (ancres, matière du fond publié), dilatées de la marge résolue, trous
// intérieurs remplis. La brique ne rasterise ni ne remplit en dehors d'une cuisson ; la silhouette
// exacte (trous remplis) est celle de l'image plan (`himap.PlanPNG`), que l'on reprend ici et que
// l'on dilate : aucun second algorithme de rasterisation.

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"math"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/mapfondsidecar"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

// papierPlan est la teinte de fond de `himap.PlanPNG` (pixel hors silhouette) ; un test garde
// qu'elle est toujours celle de la brique.
var papierPlan = color.RGBA{R: 245, G: 245, B: 242, A: 255}

// marqueRogne est le texte que `-rogne-publie` ajoute à `source` : il sert aussi à ne pas rogner deux fois.
const marqueRogne = "fond publie rogne a la silhouette du maillage"

// rognagePublie est ce qu'un rognage du fond publié a retiré et ce que l'ancrage y a perdu.
type rognagePublie struct {
	Retire, Avant            int
	Marge                    float64
	AncresAvant, AncresApres int
}

// part rend la part de matière retirée, entre 0 et 1.
func (r rognagePublie) part() float64 {
	if r.Avant == 0 {
		return 0
	}
	return float64(r.Retire) / float64(r.Avant)
}

// masqueSilhouette rend le masque (indice py*largeur + px) des pixels du plan hors papier, dilatés
// de `rayon` pixels : la silhouette des composantes retenues, trous remplis, élargie de la marge.
func masqueSilhouette(plan *image.RGBA, rayon int) []bool {
	b := plan.Bounds()
	nx, ny := b.Dx(), b.Dy()
	base := make([]bool, nx*ny)
	for py := 0; py < ny; py++ {
		for px := 0; px < nx; px++ {
			o := plan.PixOffset(b.Min.X+px, b.Min.Y+py)
			base[py*nx+px] = color.RGBA{R: plan.Pix[o], G: plan.Pix[o+1], B: plan.Pix[o+2], A: plan.Pix[o+3]} != papierPlan
		}
	}
	if rayon <= 0 {
		return base
	}
	return dilate(dilate(base, nx, ny, rayon, true), nx, ny, rayon, false)
}

// dilate élargit un masque de `rayon` pixels le long des lignes (`enX`) ou des colonnes : deux
// passes donnent la dilatation carrée de la brique.
func dilate(base []bool, nx, ny, rayon int, enX bool) []bool {
	out := make([]bool, len(base))
	for py := 0; py < ny; py++ {
		for px := 0; px < nx; px++ {
			for d := -rayon; d <= rayon && !out[py*nx+px]; d++ {
				x, y := px, py+d
				if enX {
					x, y = px+d, py
				}
				if x >= 0 && x < nx && y >= 0 && y < ny && base[y*nx+x] {
					out[py*nx+px] = true
				}
			}
		}
	}
	return out
}

// effaceHorsMasque met à zéro les pixels de matière hors du masque et rend leur nombre.
func effaceHorsMasque(img *image.RGBA, masque []bool) int {
	b := img.Bounds()
	efface := 0
	for py := 0; py < b.Dy(); py++ {
		for px := 0; px < b.Dx(); px++ {
			if masque[py*b.Dx()+px] {
				continue
			}
			o := img.PixOffset(b.Min.X+px, b.Min.Y+py)
			if img.Pix[o+3] == 0 {
				continue
			}
			img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = 0, 0, 0, 0
			efface++
		}
	}
	return efface
}

// ancresAvecMatiere compte les ancres du cadre qui ont de la matière à moins de `rayon` pixels.
func ancresAvecMatiere(img *image.RGBA, cal himap.CalageFond, ancres [][3]float64, rayon int) int {
	b := img.Bounds()
	n := 0
	for _, a := range ancres {
		px := int(math.Floor((a[0] - cal.OrigineX) / cal.MetresParPixel))
		py := int(math.Floor((cal.OrigineY - a[1]) / cal.MetresParPixel))
		if px < 0 || py < 0 || px >= b.Dx() || py >= b.Dy() {
			continue
		}
		if matiereAutour(img, px, py, rayon) {
			n++
		}
	}
	return n
}

// matiereAutour dit si un pixel de matière tombe dans le carré de `rayon` pixels autour de (px, py).
func matiereAutour(img *image.RGBA, px, py, rayon int) bool {
	b := img.Bounds()
	for y := max(0, py-rayon); y <= min(b.Dy()-1, py+rayon); y++ {
		for x := max(0, px-rayon); x <= min(b.Dx()-1, px+rayon); x++ {
			if img.Pix[img.PixOffset(b.Min.X+x, b.Min.Y+y)+3] != 0 {
				return true
			}
		}
	}
	return false
}

// rognePublieA rogne le fond publié `fond` de la carte à la silhouette de son plan `pf`, dans le
// mode de l'avis, à la marge résolue de la carte. Rend le bilan du rognage ; `fond` est modifié.
// Une carte que l'avis laisse (publie, sans maillage), ou qui cuit sans son maillage, n'est pas
// rognée : ok vaut faux. Seul le mode silhouette est pris en charge (le masque des faces seules
// demande une rasterisation que la brique n'expose pas) ; tout autre mode est une erreur.
func (e *environnement) rognePublieA(ctx context.Context, c carteDuPlan, fond *image.RGBA, cal himap.CalageFond, pf *planFond) (rognagePublie, bool, error) {
	d := e.decisionFond(c.cle)
	if d.Saute || !d.Rogne || e.sansNavmeshDe(c.cle) {
		slog.InfoContext(ctx, "mapfond: fond publie laisse non rogne", "carte", c.cle, "etat", d.Etat)
		return rognagePublie{}, false, nil
	}
	if d.Mode != himap.ModeSilhouette {
		return rognagePublie{}, false, fmt.Errorf("mode %q : seul le mode silhouette rogne un fond publie", d.Mode)
	}
	marge := e.margeNavmeshDe(c.cle)
	if marge <= 0 {
		marge = himap.MargeNavmesh
	}
	rayon := int(marge/cal.MetresParPixel + 0.5)
	r := rognagePublie{Marge: marge, Avant: compteMatiere(fond)}
	r.AncresAvant = ancresAvecMatiere(fond, cal, c.ancres, himap.RayonAncreComposante)
	r.Retire = effaceHorsMasque(fond, masqueSilhouette(pf.Image, rayon))
	r.AncresApres = ancresAvecMatiere(fond, cal, c.ancres, himap.RayonAncreComposante)
	return r, true, nil
}

// poseLeRognage met à jour le sidecar d'un fond publié que l'on vient de rogner : `source` le dit,
// la date de génération est celle du rognage, les cellules retirées s'ajoutent à `cellsClipped` et
// les ancres au sol perdues par le rognage (matière à moins de `RayonAncreComposante` pixels)
// sortent de `anchorsWithGround`. Le fond est écrit en PNG (le WebP suit par `mapfond-webp`).
func poseLeRognage(s *mapfondsidecar.MapBackground, cle string, r rognagePublie) {
	s.Source = strings.TrimSpace(s.Source) + fmt.Sprintf(" ; %s (marge %g m, %.1f %% de la matiere retiree)", marqueRogne, r.Marge, 100*r.part())
	s.GeneratedAt = time.Now().UTC()
	s.Stats.CellsClipped += r.Retire
	if perdues := r.AncresAvant - r.AncresApres; perdues > 0 {
		s.Stats.AnchorsWithGround = max(0, s.Stats.AnchorsWithGround-perdues)
	}
	s.Image = cle + ".png"
}

// rogneLePublie rogne le fond publié de la carte à la silhouette de son plan, écrit le fond en PNG
// (le WebP suit par `mapfond-webp`) et met son sidecar à jour. Un fond déjà rogné (sa `source` le
// dit) n'est pas rogné deux fois : son plan se retire de la matière déjà rognée, un second passage
// mesurerait une part nulle. `suivi` reçoit l'action et la part rognée.
func (e *environnement) rogneLePublie(ctx context.Context, c carteDuPlan, s *mapfondsidecar.MapBackground,
	fond *image.RGBA, cal himap.CalageFond, suivi *suiviFond) error {
	if strings.Contains(s.Source, marqueRogne) {
		slog.InfoContext(ctx, "mapfond: fond publie deja rogne, laisse tel quel", "carte", c.cle)
		return nil
	}
	r, ok, err := e.rognePublieA(ctx, c, fond, cal, suivi.Plan)
	if err != nil || !ok {
		return err
	}
	if _, err := ecritPNG(filepath.Join(e.sortieDir, c.cle+".png"), fond); err != nil {
		return fmt.Errorf("ecriture du fond rogne : %w", err)
	}
	poseLeRognage(s, c.cle, r)
	part := r.part()
	suivi.Action, suivi.PartRognee, suivi.Rognage = actionPublieRogne, &part, &r
	slog.InfoContext(ctx, "mapfond: fond publie rogne a la silhouette", "carte", c.cle, "part", part,
		"ancresAvant", r.AncresAvant, "ancresApres", r.AncresApres)
	return nil
}
