package main

// Le relief d'une carte : hauteur du dessus par pixel, sols du maillage et traits, calculés par la
// brique `himap` dans le calage du fond PUBLIÉ, puis écrits à côté du fond (`<cle>.relief.png`,
// `<cle>.relief.json`) avec leurs deux noms dans le sidecar. Un seul calcul pour la cuisson complète
// et pour `-relief-seul`.
//
// Le relief ne survit pas à son fond : une cuisson complète le refait dans le même geste (elle a le
// rendu en mémoire) ou, à défaut, retire ses deux champs du sidecar et ses fichiers.

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/mapfondsidecar"

	"github.com/JGtm/halo-infinite-map-images/himap"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

// entreesRelief est ce que le relief d'une carte consomme : le rendu cuit, le maillage et les faces
// que la cuisson garde (nil sans maillage), le calage du fond publié et ce fond.
type entreesRelief struct {
	rendu    *himap.Rendu
	maillage *hinavmesh.Maillage
	garde    []bool
	cal      himap.CalageFond
	fond     *image.RGBA
}

// reliefFond est le relief calculé d'une carte et ce que son écriture a pesé.
type reliefFond struct {
	Relief *himap.Relief
	Traits himap.BilanTraits
	Duree  time.Duration
	// OctetsImage et OctetsDonnees : poids de `<cle>.relief.png` et de `<cle>.relief.json` écrits.
	OctetsImage, OctetsDonnees int64
}

// gardeDuMaillage rend les faces que la cuisson garde : les composantes retenues avec la matière du
// fond publié `matiere`, mêmes ancres que le plan. Nil sans maillage (toutes les faces gardées).
func gardeDuMaillage(m *hinavmesh.Maillage, ancres [][3]float64, matiere func(x, y float64) bool) []bool {
	if m == nil {
		return nil
	}
	return himap.ComposantesRetenues(m, ancres, matiere).Garde
}

// masqueDeMatiere rend, par pixel d'un fond (indice py*largeur + px), s'il porte de la matière : un
// pixel d'alpha non nul.
func masqueDeMatiere(img *image.RGBA) []bool {
	b := img.Bounds()
	out := make([]bool, b.Dx()*b.Dy())
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out[y*b.Dx()+x] = img.Pix[img.PixOffset(b.Min.X+x, b.Min.Y+y)+3] != 0
		}
	}
	return out
}

// calculeRelief calcule le relief du rendu dans le calage publié : hauteurs, mesure du fond publié
// sans hauteur, sols du maillage, traits. Une échelle de cuisson différente de celle du fond publié
// rend himap.ErrEchelleDifferente et rien d'autre.
func calculeRelief(ctx context.Context, in entreesRelief) (*reliefFond, error) {
	debut := time.Now()
	rel, err := himap.CalculeRelief(ctx, in.rendu, in.maillage, in.garde, in.cal)
	if err != nil {
		return nil, err
	}
	if err := rel.MesureFondPublie(masqueDeMatiere(in.fond)); err != nil {
		return nil, err
	}
	if err := rel.CalculeSols(ctx, in.maillage, in.garde); err != nil {
		return nil, fmt.Errorf("sols du relief : %w", err)
	}
	traits, err := rel.CalculeTraits(ctx)
	if err != nil {
		return nil, fmt.Errorf("traits du relief : %w", err)
	}
	return &reliefFond{Relief: rel, Traits: traits, Duree: time.Since(debut)}, nil
}

// ecritLeRelief écrit `<cle>.relief.png` et `<cle>.relief.json` dans `dir`, pose leurs noms dans le
// sidecar `s` et journalise la carte : poids écrits, part du relief tirée du maillage, pixels du fond
// publié sans hauteur, dégradations. Une erreur d'écriture laisse `s` sans relief.
func ecritLeRelief(ctx context.Context, dir, cle string, rf *reliefFond, s *mapfondsidecar.MapBackground) error {
	s.ReliefImage, s.ReliefData = "", ""
	nomImage := mapfondsidecar.NomImageRelief(cle, ".png")
	nomDonnees := mapfondsidecar.NomDonneesRelief(cle)
	octets, err := ecritPNG(filepath.Join(dir, nomImage), rf.Relief.Image)
	if err != nil {
		return fmt.Errorf("écriture de l'image du relief %s : %w", nomImage, err)
	}
	rf.OctetsImage = octets
	blob, err := json.Marshal(rf.Relief.Donnees)
	if err != nil {
		return fmt.Errorf("sérialisation des données du relief %s : %w", nomDonnees, err)
	}
	blob = append(blob, '\n')
	if err := os.WriteFile(filepath.Join(dir, nomDonnees), blob, 0o644); err != nil { //nolint:gosec // asset versionné
		return fmt.Errorf("écriture des données du relief %s : %w", nomDonnees, err)
	}
	rf.OctetsDonnees = int64(len(blob))
	s.ReliefImage, s.ReliefData = nomImage, nomDonnees
	journaliseLeRelief(ctx, cle, rf)
	return nil
}

// journaliseLeRelief écrit la ligne de journal d'une carte.
func journaliseLeRelief(ctx context.Context, cle string, rf *reliefFond) {
	b := rf.Relief.Bilan
	slog.InfoContext(ctx, "mapfond: relief ecrit", "carte", cle,
		"octets_image_png", rf.OctetsImage, "octets_donnees", rf.OctetsDonnees,
		"part_maillage", b.PartMaillage, "zone", b.Zone,
		"fond_matiere", b.FondPublieMatiere, "fond_sans_hauteur", b.FondPublieSansHauteur,
		"etages", len(rf.Relief.Donnees.Floors.Niveaux),
		"degradations", rf.Relief.Donnees.Degradations, "duree", rf.Duree)
}

// retireLeRelief retire du sidecar `s` les deux champs du relief et supprime ses fichiers du dossier :
// un relief calculé pour un fond qui n'est plus le publié ne doit pas rester.
func retireLeRelief(ctx context.Context, dir, cle string, s *mapfondsidecar.MapBackground) {
	s.ReliefImage, s.ReliefData = "", ""
	for _, nom := range []string{
		mapfondsidecar.NomImageRelief(cle, ".png"), mapfondsidecar.NomImageRelief(cle, ".webp"),
		mapfondsidecar.NomDonneesRelief(cle),
	} {
		if err := os.Remove(filepath.Join(dir, nom)); err != nil && !os.IsNotExist(err) {
			slog.WarnContext(ctx, "mapfond: fichier du relief perime non supprime", "err", err, "carte", cle, "fichier", nom)
		}
	}
}

// entreesDeCible rassemble les entrées du relief d'une carte qu'on vient de cuire : le fond publié est
// l'image rognée `img` au calage `cal`, les faces gardées sont celles du plan quand il existe.
func entreesDeCible(c cible, rendu *himap.Rendu, img *image.RGBA, cal himap.CalageFond, pf *planFond) entreesRelief {
	garde := gardeDuMaillage(c.maillage, c.ancres, matiereDeImage(img, cal))
	if pf != nil {
		garde = pf.Retenues.Garde
	}
	return entreesRelief{rendu: rendu, maillage: c.maillage, garde: garde, cal: cal, fond: img}
}

// reliefDeCible calcule le relief d'une carte qu'on vient de cuire. Un relief impossible est
// journalisé et rend nil : le fond reste publié sans relief.
func reliefDeCible(ctx context.Context, cle string, in entreesRelief) *reliefFond {
	rf, err := calculeRelief(ctx, in)
	if err != nil {
		slog.ErrorContext(ctx, "mapfond: relief impossible, fond publie sans relief", "err", err, "carte", cle)
		return nil
	}
	return rf
}

// poseLeRelief écrit le relief et pose ses noms dans le sidecar `s` ; sans relief (nil) ou sur une
// écriture impossible, la carte est publiée sans relief et ses fichiers périmés sont retirés.
func poseLeRelief(ctx context.Context, dir, cle string, rf *reliefFond, s *mapfondsidecar.MapBackground) {
	if rf != nil {
		err := ecritLeRelief(ctx, dir, cle, rf, s)
		if err == nil {
			return
		}
		slog.ErrorContext(ctx, "mapfond: relief non ecrit, fond publie sans relief", "err", err, "carte", cle)
	}
	retireLeRelief(ctx, dir, cle, s)
}
