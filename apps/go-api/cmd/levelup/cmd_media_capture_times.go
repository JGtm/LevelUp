package main

// cmd_media_capture_times.go — `levelup media-capture-times` : recalcul HORS SERVEUR de la
// fenêtre (début, fin) des médias indexés avant la règle par source (ops.RecalcMediaCaptureTimes,
// internal/ops/media_capture_recalc.go).
//
// LE SERVEUR N'EN A PAS BESOIN : chaque indexation (envoi, balayage après synchronisation, bouton
// « Scanner » des réglages) fait ce recalcul sur sa base, sous son verrou. La commande sert à
// SIMULER avant le déploiement, et à écrire sur une base que rien ne tient.
//
// Usage :
//
//	levelup media-capture-times --dry-run --db <copie de shared_social.duckdb>  # serveur en marche
//	levelup media-capture-times --dry-run [--title halo_5]                       # serveur ARRÊTÉ
//	levelup media-capture-times [--title halo_5]                                 # écriture, serveur ARRÊTÉ
//
// DuckDB n'admet qu'un processus par fichier : sur la base que le serveur tient, l'ouverture
// échoue (rien n'est lu ni écrit) ; la simulation se fait alors sur une copie.

import (
	"context"
	"flag"
	"fmt"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
)

func runMediaCaptureTimes(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("media-capture-times", flag.ExitOnError)
	titleSlug := fs.String("title", title.DefaultSlug, "Slug du titre (ex: halo_infinite, halo_5)")
	dbPath := fs.String("db", "", "Base shared_social à traiter (défaut : celle du titre ; une copie pour simuler serveur en marche)")
	dryRun := fs.Bool("dry-run", false, "Compter par source ce qui changerait, sans rien écrire")
	if err := fs.Parse(args); err != nil {
		return err
	}
	path := *dbPath
	if path == "" {
		path = title.NewPathResolver(cfg.RepoRoot).SharedSocialDBPath(*titleSlug)
	}
	report, err := ops.RecalcMediaCaptureTimes(context.Background(), ops.CaptureRecalcOptions{
		DBPath:   path,
		Timezone: cfg.UserTimezone,
		DryRun:   *dryRun,
	})
	if err != nil {
		return err
	}
	printCaptureRecalcReport(path, report)
	return nil
}

func printCaptureRecalcReport(path string, r ops.CaptureRecalcReport) {
	mode := "ÉCRITURE"
	if r.DryRun {
		mode = "SIMULATION (rien n'est écrit)"
	}
	fmt.Printf("Fenêtres de capture des médias — %s\n  base : %s\n", mode, path)
	fmt.Printf("  lignes à recalculer : %d, dont décalées : %d\n", r.Examined, r.Shifted)
	for _, k := range r.SourceKeys() {
		c := r.BySource[k]
		fmt.Printf("  %-22s %5d lignes, %5d décalées, %5d vidéos sans durée\n", k, c.Examined, c.Shifted, c.NoDuration)
	}
}
