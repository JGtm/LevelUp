// Package persist — queue.go : BatchQueue durable avec WAL JSON + channels.
//
// Architecture :
//   - Submit : sérialise le batch en JSON, l'écrit dans walDir/{batch_id}.json
//     (durabilité), puis pousse dans le channel approprié (par DBTarget).
//   - Worker : lit le channel, appelle Persister, ACK = supprime le WAL.
//   - RecoverPending : au boot, lit walDir/*.json et re-pousse dans les
//     channels (les batches non ACKés au crash précédent reprennent).
//   - Corruption : WAL invalide → déplacé dans walDir/corrupted/.

package persist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrQueueClosed est retourné par Submit après que Close ait été appelé.
var ErrQueueClosed = errors.New("persist: queue closed")

// BatchQueueConfig configure une instance BatchQueue.
type BatchQueueConfig struct {
	// WALDir : répertoire où sont écrits les fichiers JSON WAL.
	// Créé automatiquement s'il n'existe pas. Sous-dossier `corrupted/`
	// créé à la demande pour les fichiers JSON invalides.
	WALDir string

	// ChanBufSize : taille du buffer de chaque channel (1 par DBTarget).
	// Si full, Submit bloque (backpressure naturelle). Défaut : 1000.
	ChanBufSize int
}

// BatchQueue route les batches vers le bon worker (par DBTarget) et garantit
// la durabilité via WAL JSON sur disque.
type BatchQueue struct {
	walDir string

	// channels : 1 par DBTarget. Tous les batches qui touchent une DB
	// donnée passent par le même channel (sérialisation côté worker).
	// Pour simplifier Phase 1 : 1 SEUL channel partagé. Les workers liront
	// tous depuis le même et chacun filtrera selon sa DBTarget.
	// → décision révisable si bottleneck observé.
	chMain chan *MatchBatch

	closed   bool
	closedMu sync.RWMutex
}

// NewBatchQueue crée une BatchQueue. WALDir est créé s'il n'existe pas.
// Retourne erreur si l'I/O échoue.
func NewBatchQueue(cfg BatchQueueConfig) (*BatchQueue, error) {
	if cfg.WALDir == "" {
		return nil, errors.New("persist: WALDir requis")
	}
	bufSize := cfg.ChanBufSize
	if bufSize <= 0 {
		bufSize = 1000
	}
	if err := os.MkdirAll(cfg.WALDir, 0o755); err != nil {
		return nil, fmt.Errorf("persist: mkdir WALDir: %w", err)
	}
	return &BatchQueue{
		walDir: cfg.WALDir,
		chMain: make(chan *MatchBatch, bufSize),
	}, nil
}

// Submit écrit le batch dans le WAL puis le pousse dans le channel.
// Bloquant si le channel est plein (backpressure).
//
// Property : si Submit retourne nil, le batch est DURABLE (présent sur disque).
// Crash après ce point → recovery au boot le retrouvera.
//
// Retourne ErrQueueClosed si Close a déjà été appelé.
func (q *BatchQueue) Submit(batch *MatchBatch) error {
	q.closedMu.RLock()
	if q.closed {
		q.closedMu.RUnlock()
		return ErrQueueClosed
	}
	q.closedMu.RUnlock()

	// 1. Sérialise en JSON.
	data, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return fmt.Errorf("persist: marshal batch %s: %w", batch.BatchID, err)
	}

	// 2. Écrit le WAL ATOMIQUEMENT (write tmp + rename).
	walPath := filepath.Join(q.walDir, batch.BatchID+".json")
	tmpPath := walPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("persist: write WAL tmp %s: %w", batch.BatchID, err)
	}
	if err := os.Rename(tmpPath, walPath); err != nil {
		_ = os.Remove(tmpPath) // best-effort cleanup
		return fmt.Errorf("persist: rename WAL %s: %w", batch.BatchID, err)
	}

	// 3. Push dans le channel (peut bloquer).
	q.chMain <- batch
	return nil
}

// Channel retourne le channel pour une DBTarget donnée. Phase 1 retourne
// le même channel pour toutes les targets (worker filtre côté lecture).
// Phase 2 pourra introduire 1 channel par target si besoin.
func (q *BatchQueue) Channel(target DBTarget) <-chan *MatchBatch {
	// Phase 1 : channel unique partagé.
	// Les workers (1 par target) liront tous depuis ce channel et filtreront
	// via switch sur batch DBTarget si nécessaire. À refactorer en Phase 2
	// si bottleneck observé.
	_ = target
	return q.chMain
}

// ACK signale qu'un batch a été persisté avec succès — supprime le WAL.
// Idempotent : ACK d'un batch inexistant retourne nil (pas d'erreur).
func (q *BatchQueue) ACK(batchID string) error {
	walPath := filepath.Join(q.walDir, batchID+".json")
	err := os.Remove(walPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("persist: ACK remove %s: %w", batchID, err)
	}
	return nil
}

// RecoverPending lit walDir/*.json, désérialise chaque batch, et re-pousse
// dans le channel principal. À appeler UNE FOIS au boot AVANT de démarrer
// les workers.
//
// Les fichiers JSON invalides sont déplacés vers walDir/corrupted/ + log ERROR.
//
// Retourne nil même si des fichiers individuels sont corrompus (best-effort).
// Retourne erreur si l'I/O sur walDir échoue.
func (q *BatchQueue) RecoverPending() error {
	entries, err := os.ReadDir(q.walDir)
	if err != nil {
		return fmt.Errorf("persist: read WALDir: %w", err)
	}

	corruptedDir := filepath.Join(q.walDir, "corrupted")

	for _, entry := range entries {
		if entry.IsDir() || !isJSONFile(entry.Name()) {
			continue
		}
		walPath := filepath.Join(q.walDir, entry.Name())
		data, rdErr := os.ReadFile(walPath)
		if rdErr != nil {
			slog.Error("persist: recover read failed",
				"file", entry.Name(), "err", rdErr)
			continue
		}

		var batch MatchBatch
		if uErr := json.Unmarshal(data, &batch); uErr != nil {
			// JSON invalide → déplacer dans corrupted/
			if mkErr := os.MkdirAll(corruptedDir, 0o755); mkErr != nil {
				slog.Error("persist: mkdir corrupted failed",
					"err", mkErr)
				continue
			}
			corruptedPath := filepath.Join(corruptedDir, entry.Name())
			if mvErr := os.Rename(walPath, corruptedPath); mvErr != nil {
				slog.Error("persist: move to corrupted failed",
					"file", entry.Name(), "err", mvErr)
				continue
			}
			slog.Error("persist: WAL corrompu déplacé",
				"file", entry.Name(), "corrupted_path", corruptedPath, "parse_err", uErr)
			continue
		}

		// Push (peut bloquer si channel plein — back-pressure)
		q.chMain <- &batch
		slog.Info("persist: WAL recovery pushed",
			"batch_id", batch.BatchID, "source", batch.Source)
	}
	return nil
}

// PendingCount retourne le nombre de fichiers WAL en attente (= batches
// submitted mais pas encore ACKés). Lit le dossier walDir/ — exclut le
// sous-dossier corrupted/.
//
// Utilisé par Drain() pour savoir quand tous les batches d'un cycle ont
// été persistés. Implémentation simple (count files on disk) plutôt qu'un
// compteur mémoire — robuste aux crash + recovery, pas de désynchro
// possible entre l'état mémoire et le disque.
func (q *BatchQueue) PendingCount() (int, error) {
	entries, err := os.ReadDir(q.walDir)
	if err != nil {
		return 0, fmt.Errorf("persist: read WALDir for pending count: %w", err)
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if isJSONFile(e.Name()) {
			n++
		}
	}
	return n, nil
}

// Drain attend que tous les batches en attente (PendingCount == 0) soient
// persistés. Sondage périodique du dossier walDir/. Retourne ctx.Err() si
// le contexte est annulé avant que le drain ne soit complet.
//
// Cas d'usage : appelé à la fin d'un cycle de sync pour garantir que tous
// les batches submitted ont été persistés AVANT que /sync retourne (parité
// de comportement avec le path synchrone legacy).
//
// Sondage à 50ms : compromis entre réactivité (le worker termine vite) et
// charge CPU (ReadDir n'est pas gratuit). Pour un cycle typique 10-100
// matchs, le drain prend < 1s en nominal.
func (q *BatchQueue) Drain(ctx context.Context) error {
	const pollInterval = 50 * time.Millisecond
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		n, err := q.PendingCount()
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// retry
		}
	}
}

// Close termine la queue : Submit ultérieurs retourneront ErrQueueClosed.
// Les batches déjà dans le channel restent lisibles par les workers.
func (q *BatchQueue) Close() error {
	q.closedMu.Lock()
	defer q.closedMu.Unlock()
	if q.closed {
		return nil
	}
	q.closed = true
	close(q.chMain)
	return nil
}

// isJSONFile retourne true si le nom de fichier se termine par ".json"
// (case-sensitive).
func isJSONFile(name string) bool {
	if len(name) < 5 {
		return false
	}
	return name[len(name)-5:] == ".json"
}

// PurgeOldWAL supprime les fichiers WAL plus vieux que maxAge dans walDir
// et walDir/corrupted/. Idempotent — safe à appeler périodiquement.
//
// Best-effort : continue sur erreur fichier, retourne la 1ère erreur I/O
// fatale (ex: ReadDir échoue). Retourne le nombre de fichiers supprimés.
//
// Cas d'usage : janitor périodique (ex: 1× / jour) qui retire les batches
// qui n'ont jamais pu être persistés (DB indisponible long terme, WAL
// corrompus laissés post-recovery). Sans cleanup, walDir/ peut accumuler
// indéfiniment et masquer des incidents avec des recovery anciens.
//
// Default maxAge recommandé : 7 jours. Au-delà, un batch non-persisté
// signale un incident grave (alerte ops) — soit on l'a manqué, soit la
// donnée est obsolète (un re-sync delta couvrira les matchs récents).
func (q *BatchQueue) PurgeOldWAL(maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge)
	purged := 0

	// Dossier principal walDir/*.json
	if n, err := purgeJSONFilesOlderThan(q.walDir, cutoff, false); err != nil {
		return purged, err
	} else {
		purged += n
	}

	// Sous-dossier corrupted/ — best-effort (peut ne pas exister)
	corruptedDir := filepath.Join(q.walDir, "corrupted")
	if _, err := os.Stat(corruptedDir); err == nil {
		if n, err := purgeJSONFilesOlderThan(corruptedDir, cutoff, true); err != nil {
			slog.Warn("persist: purge corrupted dir failed",
				"dir", corruptedDir, "err", err)
		} else {
			purged += n
		}
	}

	if purged > 0 {
		slog.Info("persist: WAL purge",
			"wal_dir", q.walDir, "purged_files", purged, "max_age", maxAge)
	}
	return purged, nil
}

// purgeJSONFilesOlderThan supprime les fichiers .json plus vieux que cutoff
// dans dir. `corruptedDir=true` ajoute un log warn par fichier supprimé
// (les fichiers corrompus méritent une trace, contrairement aux WAL normaux).
func purgeJSONFilesOlderThan(dir string, cutoff time.Time, corruptedDir bool) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("persist: read %s: %w", dir, err)
	}
	purged := 0
	for _, entry := range entries {
		if entry.IsDir() || !isJSONFile(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue // skip fichier inaccessible
		}
		if info.ModTime().After(cutoff) {
			continue // récent → garder
		}
		path := filepath.Join(dir, entry.Name())
		if err := os.Remove(path); err != nil {
			slog.Warn("persist: purge remove failed",
				"file", path, "err", err)
			continue
		}
		purged++
		if corruptedDir {
			slog.Warn("persist: corrupted WAL purged",
				"file", entry.Name(), "age", time.Since(info.ModTime()))
		}
	}
	return purged, nil
}
