// Package historyrepo lit, sans jamais écrire, l'état de l'historique d'un joueur : la clé de fin
// d'historique de sa base (règle de sync/historystate) et, pour l'affichage, ce que la base partagée
// du titre a rangé de ses matchs.
//
// Sous-paquet de platform/duckdb : il compose sync/historystate, que platform/duckdb n'importe pas
// lui-même. Il implémente port.HistoryCompletionReader.
package historyrepo

import (
	"context"
	"errors"
	"fmt"
	"os"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/historystate"
)

// PlayerDBPathFunc rend le chemin de la base joueur d'un titre (en production :
// PathResolver.PlayerDBPath du profil de déploiement).
type PlayerDBPathFunc func(titleSlug, gamertag string) string

// Repo lit l'état de l'historique des joueurs.
type Repo struct {
	playerDBPath PlayerDBPathFunc
}

var _ port.HistoryCompletionReader = (*Repo)(nil)

// New construit le lecteur ; playerDBPath est requis.
func New(playerDBPath PlayerDBPathFunc) *Repo {
	return &Repo{playerDBPath: playerDBPath}
}

// HistoryComplete lit la clé de fin d'historique de la base joueur par duckdb.OpenReadForQuery : le
// handle tenu par le process est réutilisé s'il existe (jamais d'ouverture lecture seule forcée sur
// un fichier que le serveur tient en écriture). Une base absente rend faux : aucune passe n'a encore
// parcouru l'historique de ce joueur.
func (r *Repo) HistoryComplete(ctx context.Context, titleSlug, gamertag string) (bool, error) {
	path := r.playerDBPath(titleSlug, gamertag)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("historyrepo: base joueur %s/%s: %w", titleSlug, gamertag, err)
	}
	db, release, err := duckdb.OpenReadForQuery(path)
	if err != nil {
		return false, fmt.Errorf("historyrepo: ouverture de la base joueur %s/%s: %w", titleSlug, gamertag, err)
	}
	defer release()
	return historystate.Complete(ctx, db), nil
}
