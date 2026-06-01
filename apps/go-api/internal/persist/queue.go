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
	"sync/atomic"
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

	// sendWG track les Submit "en vol" (qui ont passé le check closed mais
	// n'ont pas encore fini leur send sur chMain). Close attend sendWG AVANT
	// close(chMain) → un send ne peut JAMAIS toucher un channel déjà fermé
	// (sinon : panic "send on closed channel" au shutdown si un cycle de sync
	// straggler Submit pendant Close, cf. main.go Wait scheduler borné à 3s).
	// Add(1) est fait SOUS closedMu (mutuellement exclusif avec le Lock de
	// Close) : tout Submit est soit enregistré avant que Close lise closed=true,
	// soit voit closed=true et retourne sans send.
	sendWG sync.WaitGroup

	// Phase 6 PLAN_FIX_SYNC_RELIABILITY_2026-05-24 : compteurs pour le
	// circuit-breaker sur Drain. Le worker appelle RecordPersistResult()
	// apres chaque batch — la queue track les echecs consecutifs pour fail
	// fast le Drain au lieu d'attendre le timeout 60s sur un worker casse.
	consecutiveFailures atomic.Int32
}

// circuitBreakerThreshold est le nombre d'echecs Persist consecutifs au-dela
// duquel Drain considere le worker comme casse et fail fast.
//
// Choix de 5 : tolere quelques erreurs transitoires (lock retry, lease
// race) mais detecte rapidement un probleme systematique (DSN mismatch,
// schema absent, etc.). 5 echecs × 200ms typique = 1s avant fail fast,
// soit 60x mieux que les 60s du timeout fixe.
const circuitBreakerThreshold = 5

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
	// S'enregistrer comme sender en vol AVANT de relâcher le lock — Close
	// (Lock exclusif) ne pourra close(chMain) qu'après sendWG.Wait().
	q.sendWG.Add(1)
	defer q.sendWG.Done()
	q.closedMu.RUnlock()

	// Phase 4 du PLAN_FIX_SYNC_RELIABILITY_2026-05-24 : sanitize defensif
	// NaN/Inf avant marshal. Cas observe en prod (cycle 20:33-20:38) :
	// match Chocoboflor 508bd2fb + ed8adf67 + XxDaemonGamerxX cf23bfed
	// droppes silencieusement avec "json: unsupported value: NaN".
	// SanitizeBatch est idempotent (no-op si batch deja sanitize).
	SanitizeBatch(batch)

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

// RecordPersistResult notifie la queue du resultat du dernier Persist
// effectue par le worker. Utilise pour le circuit-breaker dans Drain
// (Phase 6 du PLAN_FIX_SYNC_RELIABILITY_2026-05-24).
//
// success=true → reset le compteur d'echecs consecutifs.
// success=false → incremente le compteur.
//
// Idempotent et thread-safe (atomic). Le worker l'appelle via les hooks
// OnPersistOK/OnPersistError configures par main.go.
func (q *BatchQueue) RecordPersistResult(success bool) {
	if success {
		q.consecutiveFailures.Store(0)
	} else {
		q.consecutiveFailures.Add(1)
	}
}

// ConsecutiveFailures retourne le nombre d'echecs Persist consecutifs
// depuis le dernier succes. Utile pour le monitoring expvar.
func (q *BatchQueue) ConsecutiveFailures() int32 {
	return q.consecutiveFailures.Load()
}

// ErrDrainCircuitBreaker est retourne par Drain quand le worker accumule
// trop d'echecs consecutifs (circuit-breaker ouvert).
var ErrDrainCircuitBreaker = errors.New("persist: drain aborted — worker error rate too high (circuit-breaker)")

// Drain attend que tous les batches en attente (PendingCount == 0) soient
// persistés. Sondage périodique du dossier walDir/. Retourne ctx.Err() si
// le contexte est annulé avant que le drain ne soit complet.
//
// Phase 6 PLAN_FIX_SYNC_RELIABILITY_2026-05-24 : circuit-breaker sur les
// echecs Persist consecutifs. Avant : 60s de wait fixe meme quand le worker
// echouait sur 100% des batches (Bug #1 amplifiait Bug #2 — 285s/cycle).
// Apres : Drain abort en ~1s (5 echecs × 200ms typique) si le worker est
// casse, log ErrDrainCircuitBreaker.
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
		// Phase 6 circuit-breaker : si le worker a accumule trop d'echecs
		// consecutifs ET il reste du pending, c'est qu'il est casse — fail
		// fast au lieu d'attendre le timeout 60s.
		if q.consecutiveFailures.Load() >= int32(circuitBreakerThreshold) {
			return fmt.Errorf("%w (failures=%d, pending=%d)",
				ErrDrainCircuitBreaker,
				q.consecutiveFailures.Load(),
				n)
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
//
// Ordre critique anti-panic : on pose closed=true (sous Lock), on relâche, on
// attend que tous les Submit en vol aient fini leur send (sendWG.Wait), PUIS
// on close(chMain). Tout Submit acquérant le RLock après ce point voit
// closed=true et retourne sans send → aucun send sur channel fermé possible.
func (q *BatchQueue) Close() error {
	q.closedMu.Lock()
	if q.closed {
		q.closedMu.Unlock()
		return nil
	}
	q.closed = true
	q.closedMu.Unlock()

	q.sendWG.Wait() // attendre les Submit en vol (leur send sur chMain est terminé)
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
