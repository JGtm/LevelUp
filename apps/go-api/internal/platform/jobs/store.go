// Package jobs gère le cycle de vie des jobs asynchrones longs.
// Sprint 17 : persistance JSON, redémarrage → interrupted, TTL 1h.
package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
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

// Create crée un nouveau job de type donné, le persiste et retourne une COPIE.
//
// La copie (et non le pointeur vivant stocké dans la map) est cruciale : les
// callers lancent typiquement `go runX(job.JobID, ...)` qui mute le job via
// SetStatus/Update sous lock, PUIS lisent `job.Status`/`job.JobID` hors lock
// pour la réponse HTTP. Renvoyer le pointeur vivant créait une data race entre
// cette lecture et la mutation concurrente (cf. -race openspartan_import).
// Cohérent avec Get/FindActiveJob qui renvoient aussi des copies. Le JobID
// (immuable) reste la clé pour toute mutation ultérieure via le store.
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
	cp := *job
	return &cp
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

// List retourne jusqu'à limit jobs non expirés (copies), actifs d'abord puis
// par StartedAt décroissant — pour le dashboard monitoring admin. limit <= 0
// applique le défaut 20. Les jobs terminaux au-delà de la rétention sont
// filtrés (la purge physique reste portée par Create).
func (s *Store) List(limit int) []*domain.AsyncJobStatus {
	if limit <= 0 {
		limit = 20
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*domain.AsyncJobStatus, 0, len(s.jobs))
	for _, job := range s.jobs {
		if job.IsTerminal() && job.FinishedAt != nil &&
			time.Since(*job.FinishedAt) > jobRetention {
			continue
		}
		cp := *job
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		ai, aj := out[i].IsTerminal(), out[j].IsTerminal()
		if ai != aj {
			return !ai // actifs avant terminaux
		}
		ti, tj := out[i].StartedAt, out[j].StartedAt
		switch {
		case ti == nil && tj == nil:
			return out[i].JobID > out[j].JobID
		case ti == nil:
			return false
		case tj == nil:
			return true
		default:
			return ti.After(*tj)
		}
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
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

// newJobID génère un identifiant de job non énumérable.
//
// V3 (sécurité) : l'ancien format `job_<UnixNano>` était horodaté donc énumérable
// (deviner un ID d'un autre flux à partir de l'heure). Nouveau format
// `job_<YYYYMMDD>_<hex>` : le préfixe date reste lisible (debug/tri lexical
// grossièrement chronologique — le seul usage du JobID comme ordre est un
// tiebreaker de Store.List quand StartedAt est nil, cas dégénéré), le suffixe est
// 16 octets crypto-aléatoires (collision-safe : 128 bits). AUCUN consommateur ne
// parse le timestamp de l'ID (vérifié V3b : grep parsing/Atoi/Split sur JobID → 0).
func newJobID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand ne devrait jamais échouer ; si c'est le cas, logger et
		// dégrader vers l'horloge nanoseconde (unicité process préservée) plutôt
		// que de renvoyer un ID vide/collisionnable en silence (règle slog n°3).
		slog.Error("jobs.newJobID: crypto/rand failed, falling back to timestamp", "err", err)
		return fmt.Sprintf("job_%s_%d", time.Now().UTC().Format("20060102"), time.Now().UnixNano())
	}
	return fmt.Sprintf("job_%s_%s", time.Now().UTC().Format("20060102"), hex.EncodeToString(b[:]))
}
