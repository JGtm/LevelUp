// cmd/mapfond-webp — BANC D'ESSAI : PNG -> WebP sans perte pour les fonds de carte.
//
// POURQUOI CET OUTIL. Le plan `.ai/PLAN_FONDS_CARTE_WEBP_ETAG_2026-09-09.md` propose de
// remplacer les 109 PNG de `data/titles/{slug}/reference/map_backgrounds/` par du WebP sans
// perte (D1) pour reduire leur poids (44 Mo actuellement). Avant de toucher un seul octet de
// donnee versionnee, l'Etape 0 exige de PROUVER que l'aller-retour est identique au bit pres et
// de CHIFFRER le gain reel — pas une estimation. C'est le role de cet outil, et de lui seul.
//
// DEUX MODES, UNE SEULE LOGIQUE DE VERIFICATION (roundtrip.go) :
//
//   - -verifier : n'ECRIT RIEN. Decode chaque PNG, l'encode en WebP sans perte
//     (github.com/HugoSmits86/nativewebp, D5), redecode ce WebP (golang.org/x/image/webp, D6)
//     et compare le resultat a l'original, octet a octet. Mesure et journalise ; ne modifie
//     jamais `data/`.
//   - -convertir : ECRIT. Meme verification prealable, obligatoire, fichier par fichier ; si
//     l'aller-retour n'est pas identique, l'outil REFUSE d'ecrire ce fichier (Etape 4 du plan).
//     Sinon : `<cle>.webp` est ecrit, le sidecar `<cle>.json` voit son champ `image` mis a jour
//     (D3 : le format est une propriete de la donnee), et `<cle>.png` est supprime.
//
// CE QUE CET OUTIL NE FAIT PAS A L'ETAPE 0 : le mode -convertir existe et est teste (voir
// main_test.go, sur repertoire temporaire), mais n'est PAS execute sur `data/` a cette etape —
// c'est une decision d'execution du chantier, pas une limite du code. L'execution reelle attend
// la decision D10 (poursuite ou abandon, au vu du gain mesure par -verifier).
//
// Usage :
//
//	go run ./cmd/mapfond-webp -verifier [-echantillon N] [-dir chemin] [-title slug]
//	go run ./cmd/mapfond-webp -convertir [-echantillon N] [-dir chemin] [-title slug]
//
// Sans -dir, le repertoire vient du PathResolver (`internal/domain/title`), jamais d'un
// `filepath.Join(..., "data", ...)` a la main.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/domain/title"
)

// fondPNG identifie un fond de carte PNG et sa taille sur disque — connue une seule fois, au
// lieu d'un `os.Stat` repete a chaque comparaison de tri.
type fondPNG struct {
	chemin string
	octets int64
}

func main() {
	verifier := flag.Bool("verifier", false, "mode mesure seule : n'ecrit rien dans data/")
	convertir := flag.Bool("convertir", false, "mode ecriture : convertit les fonds selectionnes")
	echantillon := flag.Int("echantillon", 0, "limite aux N plus gros PNG (0 = tous)")
	dir := flag.String("dir", "", "repertoire des fonds ; vide = PathResolver")
	slug := flag.String("title", title.DefaultSlug, "slug du titre")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	if *verifier == *convertir {
		slog.ErrorContext(ctx, "mapfond-webp: choisir exactement un mode", "verifier", *verifier, "convertir", *convertir)
		os.Exit(1)
	}

	dossier, err := resoutDossier(*slug, *dir)
	if err != nil {
		slog.ErrorContext(ctx, "mapfond-webp: dossier introuvable", "err", err)
		os.Exit(1)
	}

	fonds, err := selectionnePNG(dossier, *echantillon)
	if err != nil {
		slog.ErrorContext(ctx, "mapfond-webp: selection des fonds", "err", err, "dossier", dossier)
		os.Exit(1)
	}
	if len(fonds) == 0 {
		slog.ErrorContext(ctx, "mapfond-webp: aucun fond PNG trouve", "dossier", dossier)
		os.Exit(1)
	}
	slog.InfoContext(ctx, "mapfond-webp: fonds selectionnes", "dossier", dossier, "n", len(fonds), "echantillon", *echantillon)

	if *verifier {
		executeVerifier(ctx, fonds)
		return
	}
	executeConvertir(ctx, fonds)
}

// resoutDossier rend le repertoire des fonds : celui force par -dir, sinon celui du
// PathResolver. Jamais de `filepath.Join(..., "data", ...)` a la main (CLAUDE.md, regle
// transverse PathResolver).
func resoutDossier(slug, dir string) (string, error) {
	if dir != "" {
		return dir, nil
	}
	racine, err := title.FindRepoRoot()
	if err != nil {
		return "", fmt.Errorf("racine du depot introuvable: %w", err)
	}
	return title.NewPathResolver(racine).MapBackgroundDir(slug), nil
}

// selectionnePNG liste les fonds PNG du dossier, tries par taille DECROISSANTE (les plus gros
// fonds sont ceux qui pesent le plus sur le gain — c'est sur eux que porte la mesure de
// l'Etape 0). echantillon > 0 tronque la liste a ses N premiers elements ; 0 garde tout.
func selectionnePNG(dossier string, echantillon int) ([]fondPNG, error) {
	motif := filepath.Join(dossier, "*.png")
	chemins, err := filepath.Glob(motif)
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", motif, err)
	}
	fonds := make([]fondPNG, 0, len(chemins))
	for _, c := range chemins {
		info, err := os.Stat(c)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", c, err)
		}
		fonds = append(fonds, fondPNG{chemin: c, octets: info.Size()})
	}
	sort.Slice(fonds, func(i, j int) bool { return fonds[i].octets > fonds[j].octets })
	if echantillon > 0 && echantillon < len(fonds) {
		fonds = fonds[:echantillon]
	}
	return fonds, nil
}

// gainPct rend le gain de poids en pourcentage (0..100, peut etre negatif si le WebP est plus
// gros). avant=0 est le cas degenere d'un fichier vide : pas de division par zero, gain nul.
func gainPct(avant, apres int) float64 {
	if avant == 0 {
		return 0
	}
	return 100 * (1 - float64(apres)/float64(avant))
}
