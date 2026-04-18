// Package jobs gère le cycle de vie des jobs asynchrones longs.
// Sprint 17 : persistance JSON, redémarrage → interrupted, TTL 1h.
package jobs

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	// jobRetention est la durée de conservation des jobs terminés.
	jobRetention = time.Hour
)

// Store gère le cycle de vie des jobs asynchrones (thread-safe, persistant).
type Store struct {
	mu   sync.RWMutex
	jobs map[string]*domain.AsyncJobStatus
	path string
}

// NewStore crée un JobStore et charge la persistance depuis path.
// Les jobs en état "running" sont marqués "interrupted" (le process qui les exécutait est mort).
func NewStore(path string) *Store {
	s := &Store{
		jobs: make(map[string]*domain.AsyncJobStatus),
		path: path,
	}
	s.load()
	return s
}

// Create crée un nouveau job de type donné, le persiste et le retourne.
func (s *Store) Create(jobType domain.JobType, playerSlug string) *domain.AsyncJobStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.purgeExpiredLocked()

	id := newJobID()
	now := time.Now().UTC()
	job := &domain.AsyncJobStatus{
		JobID:      id,
		JobType:    string(jobType),
		Status:     domain.JobStatusQueued,
		Warnings:   []string{},
		StartedAt:  &now,
		PlayerSlug: playerSlug,
	}
	s.jobs[id] = job
	s.saveLocked()
	return job
}

// Get retourne un job par son ID. Retourne nil si inconnu ou expiré.
func (s *Store) Get(jobID string) *domain.AsyncJobStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return nil
	}
	// Vérifier expiration : job terminal + finishedAt > 1h
	if job.IsTerminal() && job.FinishedAt != nil {
		if time.Since(*job.FinishedAt) > jobRetention {
			return nil
		}
	}
	// Copie pour éviter une mutation externe
	cp := *job
	return &cp
}

// Update applique une mutation sur un job existant.
// fn reçoit un pointeur vers le job, peut modifier tous les champs.
// Retourne false si le job est inconnu.
func (s *Store) Update(jobID string, fn func(*domain.AsyncJobStatus)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return false
	}
	fn(job)
	s.saveLocked()
	return true
}

// SetStatus est un raccourci pour changer le status + horodater la fin.
func (s *Store) SetStatus(jobID string, status domain.JobStatus, step *string) bool {
	return s.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.Status = status
		if step != nil {
			j.CurrentStep = step
		}
		if status == domain.JobStatusRunning && j.StartedAt == nil {
			now := time.Now().UTC()
			j.StartedAt = &now
		}
		if j.IsTerminal() {
			now := time.Now().UTC()
			j.FinishedAt = &now
		}
	})
}

// FindActiveInitialSync retourne le job "initial_sync" actif (non terminal) pour un joueur.
func (s *Store) FindActiveInitialSync(playerSlug string) *domain.AsyncJobStatus {
	return s.FindActiveJob(domain.JobTypeInitialSync, playerSlug)
}

// FindActiveJob retourne un job actif (non terminal) du type et du joueur donnés.
func (s *Store) FindActiveJob(jobType domain.JobType, playerSlug string) *domain.AsyncJobStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, job := range s.jobs {
		if job.JobType == string(jobType) &&
			job.PlayerSlug == playerSlug &&
			!job.IsTerminal() {
			cp := *job
			return &cp
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Persistance interne
// ---------------------------------------------------------------------------

// load lit le fichier JSON de persistance et initialise la map.
// Les jobs "running" sont marqués "interrupted" (redémarrage du process).
func (s *Store) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("jobs.Store: cannot load persistence file", "path", s.path, "err", err)
		}
		return
	}

	var loaded map[string]*domain.AsyncJobStatus
	if err := json.Unmarshal(data, &loaded); err != nil {
		slog.Warn("jobs.Store: corrupt persistence file, starting fresh", "path", s.path, "err", err)
		return
	}

	interrupted := domain.JobStatusInterrupted
	_ = interrupted
	for id, job := range loaded {
		if job.Status == domain.JobStatusRunning || job.Status == domain.JobStatusQueued {
			job.Status = domain.JobStatusInterrupted
			now := time.Now().UTC()
			job.FinishedAt = &now
		}
		s.jobs[id] = job
	}
	slog.Info("jobs.Store: loaded from disk", "count", len(s.jobs), "path", s.path)
}

// saveLocked écrit la map en JSON. Doit être appelé avec mu verrouillé en écriture.
func (s *Store) saveLocked() {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		slog.Warn("jobs.Store: cannot create cache dir", "err", err)
		return
	}
	data, err := json.MarshalIndent(s.jobs, "", "  ")
	if err != nil {
		slog.Warn("jobs.Store: marshal error", "err", err)
		return
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		slog.Warn("jobs.Store: write error", "path", s.path, "err", err)
	}
}

// purgeExpiredLocked supprime les jobs terminaux expirés (>1h).
// Doit être appelé avec mu verrouillé en écriture.
func (s *Store) purgeExpiredLocked() {
	for id, job := range s.jobs {
		if job.IsTerminal() && job.FinishedAt != nil {
			if time.Since(*job.FinishedAt) > jobRetention {
				delete(s.jobs, id)
			}
		}
	}
}

// newJobID génère un identifiant unique pour un job.
func newJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}
