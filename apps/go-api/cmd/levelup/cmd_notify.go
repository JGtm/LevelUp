// cmd_notify.go — sous-commandes de notifications Discord :
// notify-version, notify-sync.
package main

import (
	"flag"
	"fmt"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/notify"
)

// ─────────────────────────────────────────────────────────────────────────────
// notify-version
// ─────────────────────────────────────────────────────────────────────────────

func runNotifyVersion(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("notify-version", flag.ExitOnError)
	version := fs.String("version", "", "Version à notifier ex: v6.5.0 (obligatoire)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *version == "" {
		return fmt.Errorf("--version est obligatoire")
	}
	notifyCfg := notify.LoadNotifyConfig(cfg.AppSettingsPath)
	sent := notify.NotifyNewVersion(notifyCfg, *version)
	if sent {
		fmt.Printf("✅ Notification version %s envoyée\n", *version)
	} else {
		fmt.Printf("ℹ️  Notification version %s ignorée (déjà notifiée, patch seul, ou désactivée)\n", *version)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// notify-sync  (test / debug)
// ─────────────────────────────────────────────────────────────────────────────

func runNotifySync(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("notify-sync", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	op := fs.String("op", "sync_delta", "Opération : sync_delta | sync_full | backfill")
	durationSec := fs.Int("duration", 0, "Durée de l'opération en secondes")
	matches := fs.Int("matches", 0, "Nombre de matchs synchronisés")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *gamertag == "" {
		return fmt.Errorf("--gamertag est obligatoire")
	}
	notifyCfg := notify.LoadNotifyConfig(cfg.AppSettingsPath)
	finishedAt := time.Now()
	startedAt := finishedAt.Add(-time.Duration(*durationSec) * time.Second)
	players := []notify.PlayerSyncResult{
		{Gamertag: *gamertag, MatchesSynced: *matches},
	}
	notify.NotifySync(notifyCfg, *op, startedAt, finishedAt, players, true, false)
	fmt.Printf("✅ Notification sync envoyée pour %s\n", *gamertag)
	return nil
}
