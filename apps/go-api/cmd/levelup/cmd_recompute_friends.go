// cmd_recompute_friends.go : commande CLI bootstrap recompute is_with_friends.
//
// §4 du plan Squad/Sessions overhaul (mode bootstrap initial). Itère toutes
// les player DBs configurées (multi-titres) et applique le recompute additif
// avec la liste actuelle de settings.friend_gamertags. Idempotent : la garde
// FALSE dans friends_recompute.go protège les retries.
//
// Usage typique :
//   levelup recompute-friends           # tous les joueurs configurés
//   levelup recompute-friends --dry-run # affiche les amis résolus sans UPDATE
package main

import (
	"context"
	"flag"
	"fmt"

	"levelup/go-api/internal/config"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/service"
)

func runRecomputeFriends(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("recompute-friends", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Affiche les amis configurés sans lancer le recompute")
	if err := fs.Parse(args); err != nil {
		return err
	}

	settingsStore := settings_platform.NewStore(cfg.AppSettingsPath)
	loadFriends := func() ([]string, error) {
		s, err := settingsStore.Load()
		if err != nil {
			return nil, err
		}
		return s.FriendGamertags, nil
	}

	if *dryRun {
		friends, err := loadFriends()
		if err != nil {
			return fmt.Errorf("load settings: %w", err)
		}
		players, err := cfg.LoadPlayers()
		if err != nil {
			return fmt.Errorf("load players: %w", err)
		}
		fmt.Printf("dry-run recompute-friends: friends=%d players=%d\n", len(friends), len(players))
		for _, gt := range friends {
			fmt.Printf("  ami: %s\n", gt)
		}
		for _, p := range players {
			if p.IsDemo {
				continue
			}
			fmt.Printf("  player: title=%s slug=%s gamertag=%s xuid=%s\n",
				p.TitleSlug, p.PlayerSlug, p.Gamertag, p.XUID)
		}
		return nil
	}

	orch := service.NewFriendsOrchestratorService(cfg, loadFriends)
	res, err := orch.RecomputeAll(context.Background())
	if err != nil {
		return fmt.Errorf("recompute: %w", err)
	}
	fmt.Printf(
		"recompute-friends OK: processed=%d failed=%d total_promoted=%d duration=%.2fs\n",
		res.Processed, res.Failed, res.TotalPromoted, res.Duration.Seconds(),
	)
	if res.Failed > 0 {
		fmt.Println("erreurs par joueur:")
		for slug, msg := range res.PerPlayerErrors {
			fmt.Printf("  %s: %s\n", slug, msg)
		}
	}
	return nil
}
