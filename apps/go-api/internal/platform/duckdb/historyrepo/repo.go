// Package historyrepo lit, sans jamais écrire, l'état de l'historique d'un joueur : la clé de fin
// d'historique de sa base (règle de sync/historystate) et, pour l'affichage, ce que la base partagée
// du titre a rangé de ses matchs.
//
// Sous-paquet de platform/duckdb : il compose sync/historystate, que platform/duckdb n'importe pas
// lui-même. Il implémente port.HistoryCompletionReader, port.HistoryProgressReader et port.HistoryMarkerReader.
package historyrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sqlfrag"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/historystate"
)

// PlayerDBPathFunc rend le chemin de la base joueur d'un titre (en production :
// PathResolver.PlayerDBPath du profil de déploiement).
type PlayerDBPathFunc func(titleSlug, gamertag string) string

// SharedReaderFunc rend le fournisseur de lecture de la base partagée d'un titre (en production : le
// fournisseur B-swap du titre).
type SharedReaderFunc func(titleSlug string) duckdb.SharedReader

// Repo lit l'état de l'historique des joueurs.
type Repo struct {
	playerDBPath PlayerDBPathFunc
	sharedReader SharedReaderFunc
}

var (
	_ port.HistoryCompletionReader = (*Repo)(nil)
	_ port.HistoryProgressReader   = (*Repo)(nil)
	_ port.HistoryMarkerReader     = (*Repo)(nil)
)

// New construit le lecteur ; playerDBPath est requis. Sans sharedReader (nil), ReadHistoryProgress ne
// rend que la clé de fin d'historique : l'ordonnanceur, qui n'a besoin que d'elle, n'en câble pas.
func New(playerDBPath PlayerDBPathFunc, sharedReader SharedReaderFunc) *Repo {
	return &Repo{playerDBPath: playerDBPath, sharedReader: sharedReader}
}

// HistoryComplete lit la clé de fin d'historique de la base joueur par duckdb.OpenPlayerReadForQuery :
// le handle que partage l'écrivain du processus (référence prise ; sinon ouverture dans sa
// configuration), pour que cette lecture, qui tourne pendant que la file de persistance écrit la même
// base, ne fasse jamais échouer le worker ni ne lise sur un handle fermé sous elle. Une base absente
// rend faux : aucune passe n'a encore parcouru l'historique de ce joueur. Une lecture de la clé en
// échec rend l'erreur (état inconnu, jamais « complet »).
func (r *Repo) HistoryComplete(ctx context.Context, titleSlug, gamertag string) (bool, error) {
	m, err := r.ReadHistoryMarker(ctx, titleSlug, gamertag)
	return m.Complete, err
}

// ReadHistoryMarker lit la position de l'historique du joueur dans sa base, sur le même handle partagé que
// HistoryComplete (même contrat de base absente et de lecture en échec) : la clé de fin et le marque-page du
// rattrapage (historystate.LoadCursor). Le marque-page se lit sur la même ouverture, sans second accès.
func (r *Repo) ReadHistoryMarker(ctx context.Context, titleSlug, gamertag string) (port.HistoryMarker, error) {
	path := r.playerDBPath(titleSlug, gamertag)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return port.HistoryMarker{}, nil
		}
		return port.HistoryMarker{}, fmt.Errorf("historyrepo: base joueur %s/%s: %w", titleSlug, gamertag, err)
	}
	db, release, err := duckdb.OpenPlayerReadForQuery(path)
	if err != nil {
		return port.HistoryMarker{}, fmt.Errorf("historyrepo: ouverture de la base joueur %s/%s: %w", titleSlug, gamertag, err)
	}
	defer release()
	complete, err := historystate.Complete(ctx, db)
	if err != nil {
		return port.HistoryMarker{}, fmt.Errorf("historyrepo: état de l'historique %s/%s: %w", titleSlug, gamertag, err)
	}
	m := port.HistoryMarker{Complete: complete}
	if c, ok := historystate.LoadCursor(ctx, db); ok {
		m.Offset, m.LastMatchID = c.Offset, c.LastMatchID
	}
	return m, nil
}

// ReadHistoryProgress rend la clé de fin d'historique du joueur (HistoryComplete) et ce que la base
// partagée du titre a rangé de ses matchs : leur nombre et le début du plus ancien. La Campagne est
// retirée du compte, comme dans toutes les lectures de l'historique du joueur. Le début du match
// suit l'expression canonique de fuseau (sqlfrag.SQLStartTimeCanonical), jamais `start_time` brut.
func (r *Repo) ReadHistoryProgress(ctx context.Context, titleSlug, gamertag, xuid string) (port.HistoryProgress, error) {
	complete, err := r.HistoryComplete(ctx, titleSlug, gamertag)
	if err != nil {
		return port.HistoryProgress{}, err
	}
	out := port.HistoryProgress{Complete: complete}
	if r.sharedReader == nil || xuid == "" {
		return out, nil
	}
	reader := r.sharedReader(titleSlug)
	if reader == nil {
		return out, fmt.Errorf("historyrepo: aucune base partagée pour le titre %s", titleSlug)
	}
	db, release, err := reader.Get(ctx)
	if err != nil {
		return out, fmt.Errorf("historyrepo: base partagée %s: %w", titleSlug, err)
	}
	defer release()

	query := `SELECT COUNT(*), MIN(` + sqlfrag.SQLStartTimeCanonical("mr") + `)
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?` + sqlfrag.SQLExcludeCampaignVariants(titleSlug, "mr")
	var oldest sql.NullTime
	if err := db.QueryRowContext(ctx, query, xuid).Scan(&out.MatchCount, &oldest); err != nil {
		return out, fmt.Errorf("historyrepo: comptage des matchs %s/%s: %w", titleSlug, gamertag, err)
	}
	if oldest.Valid {
		out.OldestAt = oldest.Time.UTC().Truncate(time.Second)
	}
	return out, nil
}
