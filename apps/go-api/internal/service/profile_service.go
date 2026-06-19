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
	// evictDB ferme les handles DuckDB cachés d'un chemin de player DB avant sa
	// suppression (purge). Injecté par le caller pour garder ce package SANS
	// dépendance directe à platform/duckdb (archlint no_duckdb_import). nil-safe.
	evictDB func(playerDBPath string)
}

// NewProfileService crée un ProfileService.
func NewProfileService(dbProfilesPath, repoRoot string) *ProfileService {
	return &ProfileService{store: dbprofiles.NewStore(dbProfilesPath), repoRoot: repoRoot}
}

// WithDBEvictor injecte la fonction d'éviction des handles DuckDB cachés (appelée
// avant la suppression disque lors d'une purge). Sans evictor, la purge tente
// quand même la suppression (best-effort).
func (s *ProfileService) WithDBEvictor(fn func(playerDBPath string)) *ProfileService {
	s.evictDB = fn
	return s
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

// SetTitleSyncEnabled bascule l'état de sync (actif/pause) du couple
// (titleSlug, gamertag). Délègue au store atomique. Erreurs sentinelles propagées :
// dbprofiles.ErrEntryNotFound (couple inexistant), dbprofiles.ErrLastActiveTitle
// (refus de mettre en pause le dernier titre actif du joueur).
func (s *ProfileService) SetTitleSyncEnabled(titleSlug, gamertag string, enabled bool) error {
	return s.store.SetSyncEnabled(titleSlug, gamertag, enabled)
}

// PurgeTitleData retire le couple (titleSlug, gamertag) de db_profiles.json puis
// supprime les fichiers de données du joueur pour ce titre. L'entrée profil est
// retirée d'abord (atomique) ; la suppression disque est best-effort.
//
// Retourne dataRemoved=false (sans erreur) si la suppression des fichiers a
// échoué malgré le retrait du profil (ex. verrou Windows résiduel) : le titre
// n'est de toute façon plus actif/visible, les fichiers orphelins sont inertes.
// Erreurs sentinelles (ErrEntryNotFound / ErrLastActiveTitle) propagées telles
// quelles par le store AVANT toute suppression disque.
func (s *ProfileService) PurgeTitleData(titleSlug, gamertag string) (dataRemoved bool, err error) {
	if rmErr := s.store.RemoveEntry(titleSlug, gamertag); rmErr != nil {
		return false, rmErr
	}
	pr := title.NewPathResolver(s.repoRoot)
	// Évincer les handles DuckDB cachés AVANT la suppression (verrou fichier Windows).
	if s.evictDB != nil {
		s.evictDB(pr.PlayerDBPath(titleSlug, gamertag))
	}
	if err := os.RemoveAll(pr.PlayerDir(titleSlug, gamertag)); err != nil {
		return false, nil // profil retiré, fichiers non supprimés (best-effort)
	}
	return true, nil
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
