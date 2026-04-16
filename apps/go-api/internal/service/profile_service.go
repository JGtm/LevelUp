// Package service — ProfileService : gestion des profils joueur dans db_profiles.json.
//
// Sprint 37 : extrait de setup.go pour découpler les handlers de l'accès fichier.
// Sprint 44 : format v3 title-aware avec rétrocompatibilité v2.1.
package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

// ProfileService gère les opérations CRUD sur les profils joueur.
type ProfileService struct {
	dbProfilesPath string
	repoRoot       string
}

// NewProfileService crée un ProfileService.
func NewProfileService(dbProfilesPath, repoRoot string) *ProfileService {
	return &ProfileService{dbProfilesPath: dbProfilesPath, repoRoot: repoRoot}
}

// dbProfilesFileV2 représente le format de db_profiles.json v2.1.
type dbProfilesFileV2 struct {
	Version  string                    `json:"version"`
	Profiles map[string]dbProfileEntry `json:"profiles"`
}

// dbProfilesFileV3 représente le format v3 title-aware de db_profiles.json.
type dbProfilesFileV3 struct {
	Version  string                               `json:"version"`
	Profiles map[string]map[string]dbProfileEntry `json:"profiles"`
}

type dbProfileEntry struct {
	DBPath         string `json:"db_path"`
	WaypointPlayer string `json:"waypoint_player,omitempty"`
	XUID           string `json:"xuid,omitempty"`
}

// CreatePlayer crée ou met à jour un profil dans db_profiles.json.
// Retourne la clé du profil et les warnings éventuels.
// Supporte les formats v2.1 et v3.0. Écrit toujours en v3.0.
func (s *ProfileService) CreatePlayer(req domain.CreatePlayerProfileRequest) (string, []string, error) {
	titleSlug := req.TitleSlug
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}

	data, err := os.ReadFile(s.dbProfilesPath)
	if err != nil && !os.IsNotExist(err) {
		return "", nil, err
	}

	profiles := s.parseOrMigrateProfiles(data)

	if profiles.Profiles[titleSlug] == nil {
		profiles.Profiles[titleSlug] = make(map[string]dbProfileEntry)
	}
	titleProfiles := profiles.Profiles[titleSlug]

	// Recherche insensible à la casse
	finalKey := req.Gamertag
	for k := range titleProfiles {
		if strings.EqualFold(k, req.Gamertag) {
			finalKey = k
			break
		}
	}

	pr := title.NewPathResolver(s.repoRoot)
	dbPath := pr.PlayerDBPath(titleSlug, finalKey)
	// Stocker un chemin relatif au repo root
	relPath, relErr := filepath.Rel(s.repoRoot, dbPath)
	if relErr != nil {
		relPath = dbPath
	}

	entry := dbProfileEntry{
		DBPath:         relPath,
		WaypointPlayer: req.Gamertag,
	}
	if req.XUID != "" {
		entry.XUID = req.XUID
	}

	// Merge avec l'existant
	if existing, ok := titleProfiles[finalKey]; ok {
		if req.XUID == "" && existing.XUID != "" {
			entry.XUID = existing.XUID
		}
	}
	titleProfiles[finalKey] = entry
	profiles.Profiles[titleSlug] = titleProfiles

	// Écriture v3
	out, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return "", nil, err
	}
	if err := os.WriteFile(s.dbProfilesPath, out, 0o644); err != nil {
		return "", nil, err
	}

	// Créer le dossier joueur title-aware
	playerDir := pr.PlayerDir(titleSlug, finalKey)
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		return finalKey, []string{"Dossier joueur non créé : " + err.Error()}, nil
	}

	return finalKey, nil, nil
}

// parseOrMigrateProfiles lit db_profiles.json et retourne un format v3.
// Si le fichier est v2.1, les profils sont migrés sous "halo_infinite".
func (s *ProfileService) parseOrMigrateProfiles(data []byte) dbProfilesFileV3 {
	if len(data) == 0 {
		return dbProfilesFileV3{
			Version:  "3.0",
			Profiles: make(map[string]map[string]dbProfileEntry),
		}
	}

	var versionProbe struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &versionProbe); err != nil {
		return dbProfilesFileV3{Version: "3.0", Profiles: make(map[string]map[string]dbProfileEntry)}
	}

	if versionProbe.Version == "3.0" {
		var v3 dbProfilesFileV3
		if err := json.Unmarshal(data, &v3); err == nil {
			if v3.Profiles == nil {
				v3.Profiles = make(map[string]map[string]dbProfileEntry)
			}
			return v3
		}
	}

	// Lire en v2.1 et migrer vers v3
	var v2 dbProfilesFileV2
	if err := json.Unmarshal(data, &v2); err != nil {
		return dbProfilesFileV3{Version: "3.0", Profiles: make(map[string]map[string]dbProfileEntry)}
	}

	result := dbProfilesFileV3{
		Version:  "3.0",
		Profiles: make(map[string]map[string]dbProfileEntry),
	}
	if len(v2.Profiles) > 0 {
		result.Profiles[title.DefaultSlug] = v2.Profiles
	}
	return result
}
