// Package duckdb — sync_history_repo.go : lecture de l'état « fin de l'historique vue » d'un
// joueur, la clé de sync_meta que pose le rattrapage de l'historique (une passe de sync qui
// atteint la fin de l'historique du titre sans erreur).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"
)

// syncHistoryRecheckEvery est le délai minimal avant de relire une clé de fin d'historique
// absente : l'état ne passe que de faux à vrai, et chaque lecture peut ouvrir la base du joueur
// en lecture seule pendant qu'un sync l'ouvre en écriture.
const syncHistoryRecheckEvery = time.Minute

// PlayerDBPathFunc rend le chemin de la player DB d'un joueur pour un titre (câblage : PathResolver).
type PlayerDBPathFunc func(playerSlug, titleSlug string) (string, error)

// historyMemo est ce que le dépôt retient d'une player DB : l'état lu et l'instant de la lecture.
type historyMemo struct {
	complete  bool
	checkedAt time.Time
}

// SyncHistoryRepo dit si la clé de fin d'historique est posée dans la player DB d'un joueur.
// La lecture passe par OpenReadForQuery : le handle que le process tient déjà (sync, pool) est
// réutilisé, sinon la base est ouverte en lecture seule le temps d'une requête. Une player DB
// absente (titre jamais synchronisé) n'est jamais créée par cette lecture.
//
// Les lectures sont mémorisées par chemin de player DB : un état « complet » est gardé jusqu'à
// Forget (plus aucune ouverture), un état « non complet » n'est relu qu'après
// syncHistoryRecheckEvery, une erreur n'est jamais mémorisée. Sûr pour un usage concurrent.
type SyncHistoryRepo struct {
	pathFor PlayerDBPathFunc
	key     string
	now     func() time.Time
	read    func(ctx context.Context, path string) (bool, error) // couture des tests : la lecture réelle

	mu   sync.Mutex
	memo map[string]historyMemo
}

// NewSyncHistoryRepo lie la lecture à la clé sync_meta `key` et au résolveur de chemin.
func NewSyncHistoryRepo(pathFor PlayerDBPathFunc, key string) *SyncHistoryRepo {
	r := &SyncHistoryRepo{pathFor: pathFor, key: key, now: time.Now, memo: make(map[string]historyMemo)}
	r.read = r.readKey
	return r
}

// HistoryComplete rend vrai quand la clé est posée (valeur non vide) pour le joueur et le titre ;
// faux quand la player DB n'existe pas encore ou que la clé est absente.
func (r *SyncHistoryRepo) HistoryComplete(ctx context.Context, playerSlug, titleSlug string) (bool, error) {
	path, err := r.pathFor(playerSlug, titleSlug)
	if err != nil {
		return false, fmt.Errorf("history complete: path %s/%s: %w", playerSlug, titleSlug, err)
	}
	if cached, ok := r.cached(path); ok {
		return cached, nil
	}
	complete, err := r.read(ctx, path)
	if err != nil {
		return false, fmt.Errorf("history complete: %s/%s: %w", playerSlug, titleSlug, err)
	}
	r.mu.Lock()
	r.memo[path] = historyMemo{complete: complete, checkedAt: r.now()}
	r.mu.Unlock()
	return complete, nil
}

// Forget oublie l'état mémorisé de la player DB `path` : à appeler quand ses données sont
// supprimées, sans quoi un état « complet » périmé survivrait à la purge.
func (r *SyncHistoryRepo) Forget(path string) {
	r.mu.Lock()
	delete(r.memo, path)
	r.mu.Unlock()
}

// cached rend l'état mémorisé de `path` tant qu'il fait foi : toujours pour « complet », moins
// de syncHistoryRecheckEvery après sa lecture pour « non complet ».
func (r *SyncHistoryRepo) cached(path string) (complete, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, found := r.memo[path]
	if !found {
		return false, false
	}
	if m.complete || r.now().Sub(m.checkedAt) < syncHistoryRecheckEvery {
		return m.complete, true
	}
	return false, false
}

// readKey lit la clé dans la player DB `path` ; faux quand la base ou la clé est absente.
func (r *SyncHistoryRepo) readKey(ctx context.Context, path string) (bool, error) {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	db, release, err := OpenReadForQuery(path)
	if err != nil {
		return false, fmt.Errorf("open: %w", err)
	}
	defer release()
	var v sql.NullString
	err = db.QueryRowContext(ctx, `SELECT value FROM sync_meta WHERE key = ?`, r.key).Scan(&v)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("read: %w", err)
	}
	return v.String != "", nil
}
