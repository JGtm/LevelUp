// Package service — ProfileService : gestion des profils joueur dans db_profiles.json.
//
// Sprint 37 : extrait de setup.go pour découpler les handlers de l'accès fichier.
package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain"
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

// dbProfilesFile représente le format de db_profiles.json v2.1.
type dbProfilesFile struct {
	Version  string                    `json:"version"`
	Profiles map[string]dbProfileEntry `json:"profiles"`
}

type dbProfileEntry struct {
	DBPath         string `json:"db_path"`
	WaypointPlayer string `json:"waypoint_player,omitempty"`
	XUID           string `json:"xuid,omitempty"`
}

// CreatePlayer crée ou met à jour un profil dans db_profiles.json.
// Retourne la clé du profil et les warnings éventuels.
func (s *ProfileService) CreatePlayer(req domain.CreatePlayerProfileRequest) (string, []string, error) {
	var profiles dbProfilesFile

	data, err := os.ReadFile(s.dbProfilesPath)
	if err != nil && !os.IsNotExist(err) {
		return "", nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &profiles); err != nil {
			return "", nil, err
		}
	}
	if profiles.Version == "" {
		profiles.Version = "2.1"
	}
	if profiles.Profiles == nil {
		profiles.Profiles = make(map[string]dbProfileEntry)
	}

	// Recherche insensible à la casse
	finalKey := req.Gamertag
	for k := range profiles.Profiles {
		if strings.EqualFold(k, req.Gamertag) {
			finalKey = k
			break
		}
	}

	dbPath := filepath.Join("data", "players", finalKey, "stats.duckdb")
	entry := dbProfileEntry{
		DBPath:         dbPath,
		WaypointPlayer: req.Gamertag,
	}
	if req.XUID != "" {
		entry.XUID = req.XUID
	}

	// Merge avec l'existant
	if existing, ok := profiles.Profiles[finalKey]; ok {
		if req.XUID == "" && existing.XUID != "" {
			entry.XUID = existing.XUID
		}
	}
	profiles.Profiles[finalKey] = entry

	// Écriture
	out, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return "", nil, err
	}
	if err := os.WriteFile(s.dbProfilesPath, out, 0o644); err != nil {
		return "", nil, err
	}

	// Créer le dossier joueur
	playerDir := filepath.Join(s.repoRoot, "data", "players", finalKey)
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		return finalKey, []string{"Dossier joueur non créé : " + err.Error()}, nil
	}

	return finalKey, nil, nil
}
