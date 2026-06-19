// Package service — ProfileService : gestion des profils joueur dans db_profiles.json.
//
// Sprint 37 : extrait de setup.go pour découpler les handlers de l'accès fichier.
// Sprint 44 : format v3 title-aware avec rétrocompatibilité v2.1.
// Pass B (multi-titre) : les écritures passent par platform/dbprofiles.Store
// (writer unique, atomique, verrou process) — plus de read-modify-write ad hoc.
package service

import (
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/dbprofiles"
)

// ProfileService gère les opérations CRUD sur les profils joueur.
type ProfileService struct {
	store    *dbprofiles.Store
	repoRoot string
}

// NewProfileService crée un ProfileService.
func NewProfileService(dbProfilesPath, repoRoot string) *ProfileService {
	return &ProfileService{store: dbprofiles.NewStore(dbProfilesPath), repoRoot: repoRoot}
}

// CreatePlayer crée ou met à jour un profil (couple titre × gamertag) dans
// db_profiles.json. Retourne la clé du profil et les warnings éventuels.
// L'écriture est atomique (via le Store) et migre v2.1 → v3.0 si besoin.
func (s *ProfileService) CreatePlayer(req domain.CreatePlayerProfileRequest) (string, []string, error) {
	titleSlug := req.TitleSlug
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	pr := title.NewPathResolver(s.repoRoot)

	var finalKey string
	mutErr := s.store.Mutate(func(f *dbprofiles.File) error {
		// Clé réelle (insensible à la casse) ou le gamertag tel quel si nouveau.
		key, ok := f.FindKey(titleSlug, req.Gamertag)
		if !ok {
			key = req.Gamertag
		}
		finalKey = key

		// Partir de l'entrée existante : préserve XUID, sync_enabled,
		// initial_max_matches ET les champs non typés (ex. auth_only). Entrée
		// vierge si le couple est nouveau.
		entry, _ := f.Get(titleSlug, key)
		entry.DBPath = relPlayerDBPath(pr, s.repoRoot, titleSlug, key)
		entry.WaypointPlayer = req.Gamertag
		if req.XUID != "" {
			entry.XUID = req.XUID
		}
		if req.InitialMaxMatches > 0 {
			entry.InitialMaxMatches = req.InitialMaxMatches
		}
		f.Set(titleSlug, key, entry)
		return nil
	})
	if mutErr != nil {
		return "", nil, mutErr
	}

	// Créer le dossier joueur title-aware (hors verrou fichier : IO disque).
	playerDir := pr.PlayerDir(titleSlug, finalKey)
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		return finalKey, []string{"Dossier joueur non créé : " + err.Error()}, nil
	}
	return finalKey, nil, nil
}

// relPlayerDBPath calcule le chemin de la player DB relatif au repo root (comme
// stocké dans db_profiles.json). Retombe sur le chemin absolu si la relativisation
// échoue (volumes distincts sous Windows, etc.).
func relPlayerDBPath(pr *title.PathResolver, repoRoot, titleSlug, gamertag string) string {
	dbPath := pr.PlayerDBPath(titleSlug, gamertag)
	rel, err := filepath.Rel(repoRoot, dbPath)
	if err != nil {
		return dbPath
	}
	return rel
}
