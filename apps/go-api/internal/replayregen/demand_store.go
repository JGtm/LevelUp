package replayregen

// demand_store.go — le magasin fichier des demandes explicites en vol : un JSON écrit atomiquement
// (platform/atomicfile) à chaque changement de la file. Il ne porte aucun secret : titre, match, joueur, date.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/atomicfile"
)

// demandFileVersion : la version du format du fichier ; un fichier d'une autre version est refusé.
const demandFileVersion = 1

type demandFile struct {
	Version int                             `json:"version"`
	Demands []domain.ReplayGenerationDemand `json:"demands"`
}

// FileDemandStore : [DemandStore] sur un fichier (title.PathResolver.ReplayGenerationQueuePath).
type FileDemandStore struct{ path string }

// NewFileDemandStore crée le magasin du fichier donné.
func NewFileDemandStore(path string) *FileDemandStore { return &FileDemandStore{path: path} }

// LoadDemands lit le fichier ; un fichier absent rend une liste vide, un fichier illisible une erreur.
func (f *FileDemandStore) LoadDemands(_ context.Context) ([]domain.ReplayGenerationDemand, error) {
	raw, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lecture des demandes de génération: %w", err)
	}
	var doc demandFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("demandes de génération illisibles (%s): %w", filepath.Base(f.path), err)
	}
	if doc.Version != demandFileVersion {
		return nil, fmt.Errorf("demandes de génération: version %d inconnue", doc.Version)
	}
	return doc.Demands, nil
}

// SaveDemands remplace le contenu du fichier (écriture atomique).
func (f *FileDemandStore) SaveDemands(ctx context.Context, demands []domain.ReplayGenerationDemand) error {
	raw, err := json.Marshal(demandFile{Version: demandFileVersion, Demands: demands})
	if err != nil {
		return fmt.Errorf("encodage des demandes de génération: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return fmt.Errorf("dossier des demandes de génération: %w", err)
	}
	return atomicfile.WriteFile(ctx, f.path, raw, 0o644)
}
