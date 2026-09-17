// Package watcher — daemon_remove_title.go : retrait d'UN couple (joueur, titre)
// du suivi live.
//
// POURQUOI (revue adversariale du 2026-09-16, constat P1). Mettre un titre en
// pause (PATCH /titles/{slug}/sync enabled=false) ou le purger (DELETE) retire
// le couple des profils suivis, mais le PlayerWatcher et son poller restaient
// vivants jusqu'au redémarrage. Depuis la porte « profil suivi » (ADR 0035 D3),
// chaque match détecté par ce poller fantôme était refusé par le coordinateur,
// incrémentait `sync_refused_no_profile` — le compteur qui signale une identité
// inconnue — et l'annuaire affichait `watched_without_profile` en warning : une
// action d'administration légitime déclenchait l'alarme d'intrusion.
//
// RemovePlayer (daemon_remove.go) retire TOUS les titres d'un xuid (purge
// d'identité) ; ici on ne retire que le titre concerné, les autres restent suivis.
package watcher

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain/title"
)

// RemovePlayerTitle retire le couple (xuid, titleSlug) du suivi live : cancel du
// REST poller, arrêt du MatchPoller, retrait de la map. Rend true si un couple a
// été retiré, false s'il n'était pas suivi (idempotent). Titre vide = titre par
// défaut, comme partout ailleurs dans le daemon. nil-safe.
func (d *Daemon) RemovePlayerTitle(ctx context.Context, xuid, titleSlug string) bool {
	if d == nil || xuid == "" {
		return false
	}
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	d.playersMu.Lock()
	defer d.playersMu.Unlock()
	for key, pw := range d.players {
		slug := pw.titleSlug
		if slug == "" {
			slug = title.DefaultSlug
		}
		if pw.xuid != xuid || slug != titleSlug {
			continue
		}
		if cancel := d.playerCancels[key]; cancel != nil {
			cancel()
			delete(d.playerCancels, key)
		}
		pw.stopPoller()
		delete(d.players, key)
		slog.InfoContext(ctx, "watcher_daemon: titre retiré du suivi live",
			"gamertag", pw.gamertag, "xuid", xuid, "title_slug", slug)
		return true
	}
	return false
}
