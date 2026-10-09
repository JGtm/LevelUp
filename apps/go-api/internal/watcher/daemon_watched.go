// Package watcher — daemon_watched.go : lecture du suivi live pour l'annuaire
// des joueurs (ADR 0035 D2/D7).
//
// Fichier séparé de daemon.go, déjà au-dessus du seuil de 500 lignes
// (CLAUDE.md règle 5).
package watcher

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

// WatchedPlayers rend les couples (joueur, titre) actuellement suivis en live,
// lus sous playersMu comme tous les accès à la map des watchers.
//
// C'est le quatrième registre de l'annuaire (ADR 0035 D2) : le seul qui vive en
// mémoire, et celui qui a pris en charge le compte du 2026-07-23 sans qu'aucun
// profil ne le déclare. L'annuaire le confronte aux trois autres pour poser
// l'anomalie `watched_without_profile`.
//
// Récepteur nil toléré (rend nil) : côté annuaire, le daemon est optionnel —
// un serveur sans watcher, ou la CLI, lisent les trois registres fichiers sans
// avoir à traiter un cas particulier. Le titre est normalisé sur DefaultSlug,
// comme le fait playerKey : un watcher sans titre explicite suit le titre par
// défaut, et l'annuaire doit le rattacher à CE titre.
func (d *Daemon) WatchedPlayers() []domain.WatchedPlayerRef {
	if d == nil {
		return nil
	}
	d.playersMu.RLock()
	defer d.playersMu.RUnlock()

	out := make([]domain.WatchedPlayerRef, 0, len(d.players))
	for _, pw := range d.players {
		slug := pw.titleSlug
		if slug == "" {
			slug = title.DefaultSlug
		}
		out = append(out, domain.WatchedPlayerRef{
			XUID:      pw.xuid,
			Gamertag:  pw.gamertag,
			TitleSlug: slug,
		})
	}
	return out
}
