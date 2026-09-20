// cmd/mapgeo-build — MESURE GEOMETRIQUE des positions de force d'une carte, sans un seul
// match : les cinq variables H / V / E / R / M du plan (etape 2bis.C), par noeud d'une grille
// de 50 cm posee sur le sol praticable, puis un score, une selection et des planches de
// controle.
//
// D'OU VIENT LA GEOMETRIE. Des triangles de rendu des `.module` du jeu (internal/himap, la
// chaine du fond de carte), voxelises ; le sol praticable en est DERIVE (aucune carte native
// ne publie de maillage de navigation). La coquille de mort (`sddt`) borne l'arene quand
// elle garde toutes les ancres — meme regle que le fond.
//
// LICENCE : la chaine passe par internal/himodule -> internal/ooz (GPLv3). Outillage HORS
// LIGNE, comme `mapfond-build` ; l'application ne le linke jamais.
//
// Usage :
//
//	CGO_ENABLED=1 go run ./cmd/mapgeo-build --cartes "recharge" --sortie DIR
//	                                        [--data-root DIR] [--title slug] [--sans-png]
//
// `--data-root` est la racine qui CONTIENT `data/` (le `repoRoot` du PathResolver).
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/analysis/powerpos/geo"
	"levelup/go-api/internal/domain/title"
)

// options rassemble les drapeaux.
type options struct {
	dataRoot  string
	titleSlug string
	sortie    string
	cartes    []string
	sansPNG   bool
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	opts, err := lireOptions()
	if err != nil {
		slog.Error("mapgeo: options invalides", "err", err)
		os.Exit(2)
	}
	ctx := context.Background()
	res := title.NewPathResolver(opts.dataRoot)
	cibles, err := ResoutCibles(res, opts.titleSlug, opts.cartes)
	if err != nil {
		slog.ErrorContext(ctx, "mapgeo: resolution des cartes", "err", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(opts.sortie, 0o755); err != nil {
		slog.ErrorContext(ctx, "mapgeo: dossier de sortie", "err", err, "path", opts.sortie)
		os.Exit(1)
	}
	params, reglage := geo.ParametresParDefaut(), geo.ReglageGeoV1()
	var cuites []*Cuite
	echecs := 0
	for _, c := range cibles {
		cuite, err := Cuit(ctx, c, params, reglage)
		if err != nil {
			echecs++
			slog.ErrorContext(ctx, "mapgeo: cuisson en echec", "err", err, "carte", c.Carte)
			continue
		}
		if err := publie(ctx, res, opts, cuite); err != nil {
			echecs++
			slog.ErrorContext(ctx, "mapgeo: publication en echec", "err", err, "carte", c.Carte)
			continue
		}
		cuites = append(cuites, cuite)
	}
	if err := EcrisPositionsJSON(filepath.Join(opts.sortie, "positions_geo.json"), opts.titleSlug, params, reglage, cuites); err != nil {
		slog.ErrorContext(ctx, "mapgeo: positions_geo.json", "err", err)
		echecs++
	}
	if err := EcrisRapport(filepath.Join(opts.sortie, "_rapport.md"), params, reglage, cuites); err != nil {
		slog.ErrorContext(ctx, "mapgeo: rapport", "err", err)
		echecs++
	}
	slog.InfoContext(ctx, "mapgeo: termine", "cartes", len(cuites), "echecs", echecs, "sortie", opts.sortie)
	if echecs > 0 {
		os.Exit(1)
	}
}

// publie ecrit le CSV et les planches d'une carte.
func publie(ctx context.Context, res *title.PathResolver, opts options, c *Cuite) error {
	base := filepath.Join(opts.sortie, nomDeFichier(c.Cible.Carte))
	if err := EcrisCSV(base+".csv", c); err != nil {
		return err
	}
	if err := EcrisRejetsCSV(base+"_rejets.csv", c); err != nil {
		return err
	}
	if err := EcrisCouturesCSV(base+"_coutures.csv", c, geo.ParametresParDefaut()); err != nil {
		return err
	}
	if opts.sansPNG {
		return nil
	}
	return EcrisPNG(ctx, res, opts.titleSlug, base, c)
}

// nomDeFichier rend un nom de carte sur pour un fichier.
func nomDeFichier(carte string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(carte)), " ", "_")
}

// lireOptions lit et valide les drapeaux.
func lireOptions() (options, error) {
	var opts options
	var cartes string
	flag.StringVar(&opts.dataRoot, "data-root", "", "racine qui CONTIENT data/ (defaut : racine du depot)")
	flag.StringVar(&opts.titleSlug, "title", title.DefaultSlug, "slug du titre")
	flag.StringVar(&opts.sortie, "sortie", "", "dossier des CSV, PNG et JSON")
	flag.StringVar(&cartes, "cartes", "", "cartes a cuire, noms normalises separes par des virgules")
	flag.BoolVar(&opts.sansPNG, "sans-png", false, "ne produire que les CSV et le JSON")
	flag.Parse()
	if opts.sortie == "" || strings.TrimSpace(cartes) == "" {
		return opts, fmt.Errorf("--sortie et --cartes sont obligatoires")
	}
	for _, c := range strings.Split(cartes, ",") {
		if nom := strings.TrimSpace(c); nom != "" {
			opts.cartes = append(opts.cartes, nom)
		}
	}
	if opts.dataRoot == "" {
		racine, err := title.FindRepoRoot()
		if err != nil {
			return opts, fmt.Errorf("racine du depot introuvable et --data-root absent : %w", err)
		}
		opts.dataRoot = racine
	}
	return opts, nil
}
