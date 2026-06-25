// migrate-hls-audio : rend mono-codec (AAC) le groupe audio des arbres HLS
// multipistes EXISTANTS dont les renditions mélangent des codecs (game en Opus,
// voices/full en AAC). Sans cette migration, la bascule de piste audio ne marche
// pas sur Firefox (changement de codec MSE) ni sur Safari (HLS natif ne lit pas
// l'Opus) : l'utilisateur entend toujours la rendition par défaut.
//
// Ré-encode EN PLACE la (ou les) rendition(s) Opus existante(s) vers AAC ; la
// vidéo et les renditions déjà AAC ne sont pas touchées. Idempotent : relançable
// sans risque (les arbres déjà mono-codec AAC sont ignorés). Nécessite ffmpeg +
// ffprobe dans le PATH.
//
// IMPORTANT : à lancer serveur ARRÊTÉ (ou trafic faible) — la migration remplace
// des fichiers segments servis ; une lecture concurrente d'un clip en cours de
// swap pourrait hoqueter.
//
// Usage :
//
//	migrate-hls-audio --root /opt/levelup/data/media [--slug JGtm] [--limit 0] [--dry-run]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	mediapkg "levelup/go-api/internal/media"
)

func main() {
	root := flag.String("root", "", "racine des médias (contient {gamertag}/hls/{clip}/master.m3u8) (requis)")
	onlySlug := flag.String("slug", "", "ne traiter qu'un seul joueur (dossier sous --root, optionnel)")
	limit := flag.Int("limit", 0, "nombre max d'arbres à convertir (0 = tous)")
	dryRun := flag.Bool("dry-run", false, "lister sans rien écrire")
	flag.Parse()

	if *root == "" {
		fmt.Fprintln(os.Stderr, "--root requis")
		os.Exit(2)
	}
	scope := *root
	if *onlySlug != "" {
		scope = filepath.Join(*root, *onlySlug)
	}
	fmt.Printf("Root: %s\nSlug: %q\nDryRun: %v\n\n", *root, *onlySlug, *dryRun)

	masters, err := findMasters(scope)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)
		os.Exit(1)
	}

	var scanned, converted, alreadyAAC, mono, failed int
	for _, master := range masters {
		dir := filepath.Dir(master)
		scanned++
		res, err := mediapkg.MigrateHLSAudioToAAC(context.Background(), dir, *dryRun)
		switch {
		case err != nil:
			failed++
			fmt.Printf("  ERREUR %s : %v\n", dir, err)
		case res.NotMultiTrack:
			mono++
		case res.AlreadyAAC:
			alreadyAAC++
		case len(res.Converted) > 0:
			converted++
			verb := "converti"
			if *dryRun {
				verb = "à convertir"
			}
			fmt.Printf("  %s %s : %v\n", verb, dir, res.Converted)
		}
		if *limit > 0 && converted >= *limit {
			fmt.Printf("  (limite %d atteinte, arrêt)\n", *limit)
			break
		}
	}

	fmt.Printf("\nRésultats:\n  arbres scannés   : %d\n  convertis        : %d\n  déjà AAC         : %d\n  mono-piste       : %d\n  échecs           : %d\n",
		scanned, converted, alreadyAAC, mono, failed)
	if *dryRun {
		fmt.Println("\n(dry-run : aucune écriture)")
	}
	if failed > 0 {
		os.Exit(1)
	}
}

// findMasters retourne tous les master.m3u8 sous root (récursif).
func findMasters(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "master.m3u8" {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}
