// Package watcher — daemon_remove.go : retrait d'un joueur du suivi live
// (ADR 0035 D6, première étape d'une purge d'identité).
//
// Fichier séparé de daemon.go, déjà au-dessus du seuil de 500 lignes
// (CLAUDE.md règle 5).
//
// POURQUOI PAR XUID. Le retrait existant (`UpdateSubscriptions`) travaille par
// GAMERTAG, parce qu'il applique une liste d'abonnement écrite par un humain. Une
// purge, elle, part d'une IDENTITÉ : la clé est le xuid (ADR 0035 D1). Un gamertag
// se renomme, un xuid non — et purger « le joueur qui s'appelait X » laisserait en
// vie le watcher de celui qui s'appelle X depuis.
package watcher

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain/title"
)

// RemovePlayer retire du suivi live TOUS les watchers portant ce xuid (un par
// titre suivi) et rend les titres effectivement retirés.
//
// Symétrique exact de ce que fait `UpdateSubscriptions` pour un joueur désabonné :
// cancel du REST poller (sans quoi sa goroutine survit et continue d'interroger la
// présence — fuite W2), arrêt du MatchPoller, puis retrait des deux maps. Le tout
// sous `playersMu`, comme tous les accès à ces maps.
//
// Récepteur nil et xuid vide tolérés (rendent nil) : la purge s'exécute aussi
// dans un process sans watcher (CLI), et n'a alors simplement rien à retirer.
// Idempotent : un second appel ne retire plus rien et rend une liste vide.
func (d *Daemon) RemovePlayer(ctx context.Context, xuid string) []string {
	if d == nil || xuid == "" {
		return nil
	}
	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	var removed []string
	for key, pw := range d.players {
		if pw.xuid != xuid {
			continue
		}
		if cancel := d.playerCancels[key]; cancel != nil {
			cancel() // stoppe le REST poller du joueur
			delete(d.playerCancels, key)
		}
		pw.stopPoller() // stoppe le MatchPoller (+ live_refresh lié au pollerCtx)
		delete(d.players, key)

		slug := pw.titleSlug
		if slug == "" {
			slug = title.DefaultSlug
		}
		removed = append(removed, slug)
		slog.InfoContext(ctx, "watcher_daemon: joueur retiré du suivi live",
			"gamertag", pw.gamertag, "xuid", xuid, "title_slug", slug)
	}
	return removed
}
