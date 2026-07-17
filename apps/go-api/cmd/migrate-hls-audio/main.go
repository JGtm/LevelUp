// migrate-hls-audio : maintenance EN PLACE des arbres HLS multipistes EXISTANTS +
// inspection du layout audio d'un fichier source.
//
// Modes (exclusifs) :
//
//   - défaut (codec, --root) : rend mono-codec AAC le groupe audio des arbres dont les
//     renditions mélangent des codecs (game en Opus, voices/full en AAC). Sans ça la
//     bascule de piste ne marche pas sur Firefox (changement de codec MSE) ni Safari.
//
//   - --collapse-redundant (--root) : sur les clips dont les renditions game/voices
//     portent le MÊME son (clips sans voix isolée — ex. sessions solo OBS), réécrit le
//     master pour n'exposer que `game` (copie propre). GARDÉ par la corrélation
//     d'enveloppe : un vrai clip jeu+voix (renditions distinctes) n'est PAS touché.
//
//   - --collapse-dir DOSSIER : collapse FORCÉ d'UN SEUL arbre HLS (dossier contenant
//     master.m3u8), SANS le critère de corrélation. Pour les clips legacy dont la
//     redondance est STRUCTURELLE (game = piste 0 = mix complet AVEC voix, voices = amix)
//     mais dont la corrélation tombe sous le seuil à cause de la voix, ET dont les
//     sources sont supprimées (ni re-transcodage ni collapse auto possibles). À n'utiliser
//     que sur un arbre dont on SAIT la redondance — d'où l'arbre unique explicite.
//
//   - --analyze FICHIER : inspection à sec — analyse le layout audio d'un FICHIER source
//     (détection full-mix + composante jeu + métriques) et imprime la décision. Aucune
//     écriture. Outil de diagnostic durable.
//
// Migrations idempotentes, relançables. --root/--collapse-redundant/--analyze nécessitent
// ffmpeg + ffprobe ; --collapse-dir non (réécriture master seule).
//
// IMPORTANT : les modes d'écriture remplacent des fichiers servis — lancer serveur ARRÊTÉ
// (ou trafic faible) ; une lecture concurrente d'un clip en cours de swap pourrait hoqueter.
//
// Usage :
//
//	migrate-hls-audio --root /opt/levelup/data/media [--slug JGtm] [--limit 0] [--dry-run]
//	migrate-hls-audio --root /opt/levelup/data/media --collapse-redundant [--dry-run]
//	migrate-hls-audio --collapse-dir /opt/levelup/data/media/JGtm/hls/2026-06-... [--dry-run]
//	migrate-hls-audio --analyze "/chemin/Replay 2026-07-03 21-38-44.mkv"
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
	root := flag.String("root", "", "racine des médias (contient {gamertag}/hls/{clip}/master.m3u8)")
	onlySlug := flag.String("slug", "", "ne traiter qu'un seul joueur (dossier sous --root, optionnel)")
	limit := flag.Int("limit", 0, "nombre max d'arbres à traiter (0 = tous)")
	dryRun := flag.Bool("dry-run", false, "lister sans rien écrire")
	collapse := flag.Bool("collapse-redundant", false, "réécrire les clips multipistes REDONDANTS (game≈voices, écho) vers une piste unique propre")
	collapseDir := flag.String("collapse-dir", "", "collapse FORCÉ d'UN arbre HLS (dossier avec master.m3u8), SANS critère de corrélation (clips legacy redondants dont les sources sont supprimées)")
	analyze := flag.String("analyze", "", "inspection à sec du layout audio d'un FICHIER source (aucune écriture)")
	flag.Parse()

	// Modes hors --root (exclusifs). --analyze puis --collapse-dir court-circuitent.
	if *analyze != "" && *collapseDir != "" {
		fmt.Fprintln(os.Stderr, "--analyze et --collapse-dir sont exclusifs")
		os.Exit(2)
	}
	if *analyze != "" {
		os.Exit(runAnalyze(*analyze))
	}
	if *collapseDir != "" {
		os.Exit(runCollapseDir(*collapseDir, *dryRun))
	}

	if *root == "" {
		fmt.Fprintln(os.Stderr, "--root requis (ou utiliser --analyze / --collapse-dir)")
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

// runCollapseDir collapse FORCÉ d'un unique arbre HLS (dossier contenant master.m3u8).
// Retourne le code de sortie.
func runCollapseDir(dir string, dryRun bool) int {
	fmt.Printf("Collapse FORCÉ (sans corrélation)\nDir: %s\nDryRun: %v\n\n", dir, dryRun)
	res, err := mediapkg.ForceCollapseHLSAudioTree(dir, dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERREUR %s : %v\n", dir, err)
		return 1
	}
	switch {
	case res.Collapsed && dryRun:
		fmt.Printf("  à collapser (forcé) : %s → rendition game seule\n", dir)
	case res.Collapsed:
		fmt.Printf("  collapsé (forcé) : %s → rendition game seule\n", dir)
	default:
		fmt.Printf("  ignoré %s : %s\n", dir, res.Skipped)
	}
	if dryRun {
		fmt.Println("\n(dry-run : aucune écriture)")
	}
	return 0
}

// runAnalyze imprime l'analyse du layout audio d'un fichier source. Code de sortie.
func runAnalyze(file string) int {
	rep, err := mediapkg.AnalyzeAudioLayoutReport(context.Background(), file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERREUR analyse %s : %v\n", file, err)
		return 1
	}
	printAnalyzeReport(file, rep)
	return 0
}

// printAnalyzeReport formate un AudioLayoutReport de façon lisible (diag).
func printAnalyzeReport(file string, rep mediapkg.AudioLayoutReport) {
	fmt.Printf("Analyse: %s\n", file)
	fmt.Printf("  pistes audio     : %d\n", rep.AudioTracks)
	if rep.AudioTracks < 2 {
		fmt.Println("  mono-piste       : pas d'analyse full-mix (hors périmètre)")
		return
	}
	fmt.Printf("  piste 0 full-mix : %v\n", rep.Track0FullMix)
	if rep.Track0FullMix {
		fmt.Printf("  composante jeu   : 0:a:%d\n", rep.GameComponent)
	}
	fmt.Printf("  R² ajustement    : %.4f\n", rep.R2)
	fmt.Printf("  env0 stddev (dB) : %.2f\n", rep.Env0StdDB)
	fmt.Println("  composantes (0:a:i, i≥1) :")
	for i := range rep.Gains {
		fmt.Printf("    0:a:%d  gain=%.3f part=%.3f corr_puiss=%.3f p90dB=%.1f actif=%v silence=%.2f\n",
			i+1, rep.Gains[i], rep.Shares[i], rep.PowerCorr[i], rep.P90[i], rep.Active[i], rep.SilenceRatios[i])
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
