// migrate-hls-audio : maintenance EN PLACE des arbres HLS multipistes EXISTANTS.
//
// Deux modes (exclusifs) :
//
//   - défaut (codec) : rend mono-codec AAC le groupe audio des arbres dont les
//     renditions mélangent des codecs (game en Opus, voices/full en AAC). Sans ça la
//     bascule de piste ne marche pas sur Firefox (changement de codec MSE) ni Safari.
//
//   - --collapse-redundant : sur les clips dont les renditions game/voices portent le
//     MÊME son (clips sans voix isolée — ex. sessions solo OBS), réécrit le master
//     pour n'exposer que `game` (copie propre). Supprime l'écho de `full` (= amix
//     doublé) et le sélecteur Jeu/Voix trompeur. Un vrai clip jeu+voix (renditions
//     distinctes) n'est PAS touché. Réversible (réécriture master, pas de ré-encodage).
//
// Idempotents, relançables. Nécessitent ffmpeg + ffprobe dans le PATH.
//
// IMPORTANT : à lancer serveur ARRÊTÉ (ou trafic faible) — remplace des fichiers
// servis ; une lecture concurrente d'un clip en cours de swap pourrait hoqueter.
//
// Usage :
//
//	migrate-hls-audio --root /opt/levelup/data/media [--slug JGtm] [--limit 0] [--dry-run]
//	migrate-hls-audio --root /opt/levelup/data/media --collapse-redundant [--dry-run]
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
	limit := flag.Int("limit", 0, "nombre max d'arbres à traiter (0 = tous)")
	dryRun := flag.Bool("dry-run", false, "lister sans rien écrire")
	collapse := flag.Bool("collapse-redundant", false, "réécrire les clips multipistes REDONDANTS (game≈voices, écho) vers une piste unique propre")
	flag.Parse()

	if *root == "" {
		fmt.Fprintln(os.Stderr, "--root requis")
		os.Exit(2)
	}
	scope := *root
	if *onlySlug != "" {
		scope = filepath.Join(*root, *onlySlug)
	}
	mode := "migrate-codec"
	if *collapse {
		mode = "collapse-redundant"
	}
	fmt.Printf("Root: %s\nSlug: %q\nMode: %s\nDryRun: %v\n\n", *root, *onlySlug, mode, *dryRun)

	masters, err := findMasters(scope)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)
		os.Exit(1)
	}

	var failed int
	if *collapse {
		failed = processCollapse(masters, *dryRun, *limit)
	} else {
		failed = processMigrate(masters, *dryRun, *limit)
	}
	if *dryRun {
		fmt.Println("\n(dry-run : aucune écriture)")
	}
	if failed > 0 {
		os.Exit(1)
	}
}

// processMigrate rend mono-codec AAC chaque arbre (mode défaut). Retourne le nb d'échecs.
func processMigrate(masters []string, dryRun bool, limit int) int {
	var scanned, converted, alreadyAAC, mono, failed int
	for _, master := range masters {
		dir := filepath.Dir(master)
		scanned++
		res, err := mediapkg.MigrateHLSAudioToAAC(context.Background(), dir, dryRun)
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
			if dryRun {
				verb = "à convertir"
			}
			fmt.Printf("  %s %s : %v\n", verb, dir, res.Converted)
		}
		if limit > 0 && converted >= limit {
			fmt.Printf("  (limite %d atteinte, arrêt)\n", limit)
			break
		}
	}
	fmt.Printf("\nRésultats:\n  arbres scannés   : %d\n  convertis        : %d\n  déjà AAC         : %d\n  mono-piste       : %d\n  échecs           : %d\n",
		scanned, converted, alreadyAAC, mono, failed)
	return failed
}

// processCollapse réécrit les clips redondants vers une piste unique. Retourne le nb d'échecs.
func processCollapse(masters []string, dryRun bool, limit int) int {
	var scanned, collapsed, skipped, failed int
	for _, master := range masters {
		dir := filepath.Dir(master)
		scanned++
		res, err := mediapkg.CollapseRedundantHLSAudio(context.Background(), dir, dryRun)
		switch {
		case err != nil:
			failed++
			fmt.Printf("  ERREUR %s : %v\n", dir, err)
		case res.Collapsed:
			collapsed++
			verb := "collapsé"
			if dryRun {
				verb = "à collapser"
			}
			fmt.Printf("  %s %s (corr=%.3f)\n", verb, dir, res.Corr)
		default:
			skipped++
			if res.Skipped != "" {
				fmt.Printf("  ignoré %s : %s (corr=%.3f)\n", dir, res.Skipped, res.Corr)
			}
		}
		if limit > 0 && collapsed >= limit {
			fmt.Printf("  (limite %d atteinte, arrêt)\n", limit)
			break
		}
	}
	fmt.Printf("\nRésultats (collapse):\n  arbres scannés   : %d\n  collapsés        : %d\n  ignorés          : %d\n  échecs           : %d\n",
		scanned, collapsed, skipped, failed)
	return failed
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
