// Package adminstate — persistance JSON légère et HORS DuckDB de l'état runtime
// du dashboard admin, conçue pour SURVIVRE au redémarrage du serveur.
//
// Deux consommateurs (Lot C, plan diag apparence admin) :
//   - le snapshot post-sync du scheduler (timeline + matrice par joueur +
//     horodatage du cycle) — réhydraté au boot (C1) ;
//   - le journal des actions globales (dernière exécution / issue / déclencheur),
//     écrit par la couche service et le scheduler (C2).
//
// Contraintes NON négociables :
//   - AUCUNE écriture DuckDB (invariants anti-ART intouchés) ;
//   - écriture ATOMIQUE (fichier temporaire dans le même répertoire + rename) pour
//     survivre à un kill en pleine écriture — jamais de fichier tronqué relu ;
//   - lecture tolérante : fichier absent = premier boot (pas une erreur), fichier
//     corrompu = erreur remontée au caller qui LOG et dégrade sur l'état mémoire ;
//   - chemin résolu par le PathResolver canonique (jamais de filepath.Join "data"
//     à la main côté caller).
package adminstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileStore lit/écrit un unique fichier JSON de façon atomique et thread-safe.
// Primitive partagée par le snapshot post-sync et le journal des actions.
type FileStore struct {
	path string
	mu   sync.Mutex
}

// NewFileStore crée un store sur le chemin donné (résolu par PathResolver).
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// Path retourne le chemin du fichier (diagnostic / logs).
func (s *FileStore) Path() string { return s.path }

// Load lit et désérialise le fichier dans v.
//   - found=false, err=nil : le fichier n'existe pas encore (premier boot) ou est
//     vide (kill entre create et write) — le caller garde son état mémoire.
//   - found=true, err!=nil  : fichier présent mais illisible/corrompu — le caller
//     LOG et dégrade (ne PAS supprimer : un opérateur peut vouloir l'inspecter).
func (s *FileStore) Load(v any) (found bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, readErr := os.ReadFile(s.path)
	if errors.Is(readErr, os.ErrNotExist) {
		return false, nil
	}
	if readErr != nil {
		return false, fmt.Errorf("lecture %s: %w", s.path, readErr)
	}
	if len(data) == 0 {
		return false, nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return true, fmt.Errorf("parse JSON %s: %w", s.path, err)
	}
	return true, nil
}

// Save sérialise v et l'écrit ATOMIQUEMENT (fichier temporaire dans le même
// répertoire + rename — os.Rename remplace la cible, y compris sous Windows).
// Crée le répertoire parent si besoin.
func (s *FileStore) Save(v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("sérialisation JSON: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("création répertoire %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".tmp-adminstate-*")
	if err != nil {
		return fmt.Errorf("création fichier temporaire: %w", err)
	}
	tmpName := tmp.Name()
	// Nettoyage défensif si on sort en erreur avant le rename (après un rename
	// réussi le fichier n'existe plus → Remove est un no-op silencieux).
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("écriture temporaire: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temporaire: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("fermeture temporaire: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("rename atomique %s: %w", s.path, err)
	}
	return nil
}
