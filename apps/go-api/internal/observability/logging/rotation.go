// Package logging — rotation.go : rotation PAR TAILLE des fichiers
// `{LogsDir}/{module}.log`, uniforme pour TOUTES les catégories.
//
// Contexte (2026-07-26) : la prod servait 2,1 Go de logs (provider.log 1,5 Go,
// auth.log 205 Mo, general.log 170 Mo, sync.log 127 Mo). Les `app.log.1/.2/.3`
// observés sur le VPS sont des RELIQUATS de l'ère Python (RotatingFileHandler
// 5 Mo × 3 de l'ancien `src/utils/log_config.py`, supprimé avec la migration Go) :
// le Go n'avait AUCUN mécanisme de rotation — cf. README « Limitations connues ».
// Ce fichier est donc LE mécanisme unique, appliqué à chaque catégorie sans
// exception (il n'y a pas de 2e implémentation à étendre).
//
// Nommage des archives : `{module}.log.1` … `{module}.log.N` — la convention des
// `app.log.1/.2/.3` déjà présents en prod. Effet de bord utile : une archive ne
// se termine pas par `.log`, donc `ops.ListLogModules` (viewer admin) l'ignore
// déjà sans modification, et `logtail` ne la propose jamais comme « module ».
//
// Multi-process : le serveur ET les CLIs (`cmd/*` via InstallCLI) écrivent dans
// le même dossier. L'append O_APPEND reste atomique et inchangé ; seule la
// rotation ajoute un rename. Le writer qui perd la course détecte, via
// os.SameFile, que le fichier sous le chemin n'est plus son inode et se contente
// de le ré-ouvrir au lieu de décaler les archives une 2e fois (ce qui aurait
// détruit le fichier neuf du gagnant). Pire cas : quelques lignes écrites dans
// `{module}.log.1` par le process qui tenait encore l'ancien descripteur — borné,
// sans perte ni corruption.
package logging

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	// DefaultRotationMaxSizeMB : taille max d'un `{module}.log` avant rotation.
	DefaultRotationMaxSizeMB = 100
	// DefaultRotationMaxBackups : nombre d'archives conservées par catégorie.
	// Pire cas sur disque par catégorie = MaxSizeMB × (1 + MaxBackups) = 400 Mo.
	DefaultRotationMaxBackups = 3

	bytesPerMB = 1024 * 1024

	// rotationRetryAfter : cooldown après un échec de rotation (disque plein,
	// permission, fichier verrouillé par un autre process sous Windows). Sans
	// lui, chaque write suivant retenterait et flooderait stderr, puisque la
	// taille reste au-dessus du plafond.
	rotationRetryAfter = time.Minute
)

// RotationPolicy borne la taille sur disque d'une catégorie de logs.
type RotationPolicy struct {
	// MaxSizeBytes : taille au-delà de laquelle le fichier est roté.
	// 0 (ou négatif) = rotation désactivée, croissance illimitée.
	MaxSizeBytes int64
	// MaxBackups : nombre d'archives `.1`…`.N` conservées. 0 = aucune archive
	// (le fichier courant repart de zéro à chaque rotation).
	MaxBackups int
}

// DefaultRotationPolicy retourne la politique par défaut (100 Mo × 3 archives).
func DefaultRotationPolicy() RotationPolicy {
	return RotationPolicy{
		MaxSizeBytes: DefaultRotationMaxSizeMB * bytesPerMB,
		MaxBackups:   DefaultRotationMaxBackups,
	}
}

// enabled indique si la rotation est active pour cette politique.
func (p RotationPolicy) enabled() bool { return p.MaxSizeBytes > 0 }

// backups retourne le nombre d'archives à conserver, clampé à >= 0.
func (p RotationPolicy) backups() int {
	if p.MaxBackups < 0 {
		return 0
	}
	return p.MaxBackups
}

// rotatingWriter est un io.WriteCloser append-only qui rote son fichier dès que
// la taille dépasse policy.MaxSizeBytes. Thread-safe (mu) — un seul writer par
// chemin, partagé par tous les clones du MultiModuleHandler.
type rotatingWriter struct {
	path   string
	policy RotationPolicy

	mu           sync.Mutex
	file         *os.File
	size         int64
	retryAfterTS time.Time // rotation en cooldown jusqu'à cet instant (échec)
}

// newRotatingWriter ouvre (ou crée) path en append et retourne le writer.
func newRotatingWriter(path string, policy RotationPolicy) (*rotatingWriter, error) {
	w := &rotatingWriter{path: path, policy: policy}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.openLocked(); err != nil {
		return nil, err
	}
	return w, nil
}

// Write écrit p, en rotant d'abord si le plafond serait dépassé.
func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		if err := w.openLocked(); err != nil {
			return 0, err
		}
	}
	if w.shouldRotateLocked(len(p)) {
		if err := w.rotateLocked(); err != nil {
			// Best-effort : on continue d'écrire dans le fichier courant plutôt
			// que de perdre la ligne. Jamais avalé en silence (CLAUDE.md règle 3) —
			// stderr est la seule sortie possible ici (on EST le logger).
			w.retryAfterTS = time.Now().Add(rotationRetryAfter)
			fmt.Fprintf(os.Stderr, "logging: rotation de %s impossible: %v\n", w.path, err)
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

// Close ferme le descripteur courant. Idempotent.
func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

// shouldRotateLocked : rote quand l'écriture ferait dépasser le plafond. Un
// fichier VIDE n'est jamais roté — sinon un record plus gros que le plafond
// déclencherait une rotation à chaque ligne (boucle de purge des archives).
func (w *rotatingWriter) shouldRotateLocked(n int) bool {
	if !w.policy.enabled() || w.size <= 0 {
		return false
	}
	if !w.retryAfterTS.IsZero() && time.Now().Before(w.retryAfterTS) {
		return false
	}
	return w.size+int64(n) > w.policy.MaxSizeBytes
}

// openLocked ouvre le fichier courant en append et recalcule la taille connue.
func (w *rotatingWriter) openLocked() error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", w.path, err)
	}
	w.file = f
	w.size = 0
	if info, statErr := f.Stat(); statErr == nil {
		w.size = info.Size()
	}
	return nil
}

// rotateLocked ferme le fichier courant, décale les archives et ré-ouvre.
// La fermeture PRÉCÈDE le rename : sous Windows, renommer un fichier encore
// ouvert échoue (Go n'ouvre pas avec FILE_SHARE_DELETE).
func (w *rotatingWriter) rotateLocked() error {
	var current os.FileInfo
	if w.file != nil {
		current, _ = w.file.Stat()
		closeErr := w.file.Close()
		w.file = nil
		if closeErr != nil {
			// Ré-ouvrir pour ne pas perdre la sortie, puis signaler.
			_ = w.openLocked()
			return fmt.Errorf("close %s: %w", w.path, closeErr)
		}
	}
	if w.rotatedByAnotherProcess(current) {
		w.retryAfterTS = time.Time{}
		return w.openLocked()
	}
	shiftErr := w.shiftArchives()
	openErr := w.openLocked()
	if shiftErr != nil {
		return shiftErr
	}
	if openErr != nil {
		return openErr
	}
	w.retryAfterTS = time.Time{}
	return nil
}

// rotatedByAnotherProcess indique qu'un autre writer (autre process) a déjà roté :
// le fichier présent sous w.path n'est plus l'inode que nous tenions.
func (w *rotatingWriter) rotatedByAnotherProcess(current os.FileInfo) bool {
	if current == nil {
		return false
	}
	onDisk, err := os.Stat(w.path)
	if err != nil {
		return false // absent : shiftArchives tolère déjà le ENOENT
	}
	return !os.SameFile(current, onDisk)
}

// shiftArchives fait glisser `.N-1`→`.N` … `.1`→`.2` puis le fichier courant → `.1`,
// après avoir supprimé l'archive la plus ancienne (au-delà de la rétention).
func (w *rotatingWriter) shiftArchives() error {
	keep := w.policy.backups()
	if keep == 0 {
		if err := removeIfExists(w.path); err != nil {
			return err
		}
		return nil
	}
	if err := removeIfExists(archivePath(w.path, keep)); err != nil {
		return err
	}
	for i := keep - 1; i >= 1; i-- {
		if err := renameIfExists(archivePath(w.path, i), archivePath(w.path, i+1)); err != nil {
			return err
		}
	}
	return renameIfExists(w.path, archivePath(w.path, 1))
}

// archivePath retourne `{path}.{n}` (ex. `provider.log.2`).
func archivePath(path string, n int) string {
	return path + "." + strconv.Itoa(n)
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

func renameIfExists(from, to string) error {
	if err := os.Rename(from, to); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("rename %s -> %s: %w", from, to, err)
	}
	return nil
}
