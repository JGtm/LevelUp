// Package dbprofiles fournit la lecture/écriture ATOMIQUE de db_profiles.json,
// le substrat d'activation par titre : profiles[title_slug][gamertag] → entrée.
//
// db_profiles.json est le fichier le plus critique de l'instance : il gate la
// liste de TOUS les couples (joueur, titre) synchronisés. Une écriture partielle
// (crash en cours d'écriture) corromprait l'ensemble. Ce store centralise donc :
//   - l'écriture ATOMIQUE (fichier temporaire dans le même dossier + rename) ;
//   - un verrou process (sync.Mutex) couvrant tout le cycle read-modify-write ;
//   - la préservation des clés top-level inconnues (round-trip fidèle) ;
//   - la migration EN MÉMOIRE v2.1 (flat) → v3.0 (title-scoped) à la lecture.
//
// Il est l'UNIQUE writer de db_profiles.json : la création de profil
// (service.ProfileService) et les mutations de réglages (toggle/purge) passent
// toutes par lui pour garantir l'atomicité et l'invariant « ≥ 1 titre actif ».
package dbprofiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const version3 = "3.0"

// defaultSlug — titre implicite des profils v2.1 lors de la migration mémoire.
// Dupliqué ici (et non importé de domain/title) pour garder ce package de
// persistance sans dépendance métier ; la valeur est stable ("halo_infinite").
const defaultSlug = "halo_infinite"

// ErrLastActiveTitle est retourné quand une opération laisserait un gamertag sans
// AUCUN titre actif (sync_enabled != false). Le handler le mappe en 409.
var ErrLastActiveTitle = errors.New("dbprofiles: dernier titre actif du joueur, opération refusée")

// ErrEntryNotFound est retourné quand le couple (slug, gamertag) ciblé n'existe pas.
var ErrEntryNotFound = errors.New("dbprofiles: entrée (titre, joueur) introuvable")

// Entry représente une entrée profiles[slug][gamertag] de db_profiles.json.
//
// Les champs INCONNUS d'une entrée (ex. "auth_only" pour les joueurs suivis en
// présence sans player DB) sont préservés via extra : ce store étant l'unique
// writer, perdre un champ non typé corromprait silencieusement le comportement.
type Entry struct {
	DBPath         string `json:"db_path"`
	XUID           string `json:"xuid,omitempty"`
	WaypointPlayer string `json:"waypoint_player,omitempty"`
	// SyncEnabled : nil ou true = actif ; false = sync en PAUSE (données conservées).
	SyncEnabled *bool `json:"sync_enabled,omitempty"`
	// InitialMaxMatches : nombre de matchs demandés à l'onboarding (0 = défaut).
	InitialMaxMatches int `json:"initial_max_matches,omitempty"`
	// extra conserve les champs JSON non typés de l'entrée (round-trip fidèle).
	extra map[string]json.RawMessage
}

// IsActive résout la règle nil/true = actif.
func (e Entry) IsActive() bool { return e.SyncEnabled == nil || *e.SyncEnabled }

// entryKnownFields — clés JSON typées d'Entry (exclues de extra).
var entryKnownFields = map[string]struct{}{
	"db_path": {}, "xuid": {}, "waypoint_player": {},
	"sync_enabled": {}, "initial_max_matches": {},
}

// UnmarshalJSON décode une entrée en capturant les champs inconnus dans extra.
func (e *Entry) UnmarshalJSON(data []byte) error {
	type alias Entry // évite la récursion sur UnmarshalJSON
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*e = Entry(a)
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	for k := range all {
		if _, known := entryKnownFields[k]; known {
			delete(all, k)
		}
	}
	if len(all) > 0 {
		e.extra = all
	}
	return nil
}

// MarshalJSON ré-émet les champs typés PUIS les champs inconnus préservés.
func (e Entry) MarshalJSON() ([]byte, error) {
	type alias Entry
	typed, err := json.Marshal(alias(e))
	if err != nil {
		return nil, err
	}
	if len(e.extra) == 0 {
		return typed, nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(typed, &m); err != nil {
		return nil, err
	}
	for k, v := range e.extra {
		if _, ok := m[k]; !ok { // un champ typé ne doit jamais être écrasé par extra
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// File est la projection typée de db_profiles.json, toujours normalisée en v3
// après Load. Les clés top-level inconnues sont conservées dans extra.
type File struct {
	Version  string
	Admin    string
	Profiles map[string]map[string]Entry
	extra    map[string]json.RawMessage
}

// Store gère la lecture/écriture atomique de db_profiles.json.
type Store struct {
	mu   sync.Mutex
	path string
}

// NewStore crée un Store pour le fichier db_profiles.json donné.
func NewStore(path string) *Store { return &Store{path: path} }

// Path retourne le chemin du fichier géré.
func (s *Store) Path() string { return s.path }

// Load lit db_profiles.json (verrou partagé) et retourne un File normalisé v3.
// Fichier absent → File v3 vide (pas une erreur : instance fraîche).
func (s *Store) Load() (*File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

// loadLocked lit et parse le fichier ; suppose le verrou déjà tenu.
func (s *Store) loadLocked() (*File, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Version: version3, Profiles: map[string]map[string]Entry{}}, nil
		}
		return nil, fmt.Errorf("dbprofiles.Load read: %w", err)
	}
	return parse(data)
}

// parse décode un payload db_profiles.json (v2.1 ou v3.0) en File v3.
func parse(data []byte) (*File, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, fmt.Errorf("dbprofiles.parse top: %w", err)
	}
	f := &File{Version: version3, Profiles: map[string]map[string]Entry{}, extra: map[string]json.RawMessage{}}

	version := ""
	if raw, ok := top["version"]; ok {
		_ = json.Unmarshal(raw, &version)
	}
	if raw, ok := top["admin"]; ok {
		_ = json.Unmarshal(raw, &f.Admin)
	}
	// Préserver toute clé top-level autre que version/admin/profiles.
	for k, v := range top {
		if k == "version" || k == "admin" || k == "profiles" {
			continue
		}
		f.extra[k] = v
	}

	profilesRaw, hasProfiles := top["profiles"]
	if !hasProfiles {
		return f, nil
	}
	if version == version3 {
		if err := json.Unmarshal(profilesRaw, &f.Profiles); err != nil {
			return nil, fmt.Errorf("dbprofiles.parse v3 profiles: %w", err)
		}
		if f.Profiles == nil {
			f.Profiles = map[string]map[string]Entry{}
		}
		return f, nil
	}
	// v2.1 (flat gamertag → entry) : migrer en mémoire sous le titre par défaut.
	var flat map[string]Entry
	if err := json.Unmarshal(profilesRaw, &flat); err != nil {
		return nil, fmt.Errorf("dbprofiles.parse v2 profiles: %w", err)
	}
	if len(flat) > 0 {
		f.Profiles[defaultSlug] = flat
	}
	return f, nil
}

// Save écrit le File en v3.0 de façon ATOMIQUE (temp + rename), verrou exclusif.
func (s *Store) Save(f *File) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(f)
}

// saveLocked sérialise et écrit atomiquement ; suppose le verrou déjà tenu.
func (s *Store) saveLocked(f *File) error {
	out := make(map[string]json.RawMessage, len(f.extra)+3)
	for k, v := range f.extra {
		out[k] = v
	}
	put := func(key string, val any) error {
		raw, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("dbprofiles.Save marshal %s: %w", key, err)
		}
		out[key] = raw
		return nil
	}
	if err := put("version", version3); err != nil {
		return err
	}
	if f.Admin != "" {
		if err := put("admin", f.Admin); err != nil {
			return err
		}
	}
	profiles := f.Profiles
	if profiles == nil {
		profiles = map[string]map[string]Entry{}
	}
	if err := put("profiles", profiles); err != nil {
		return err
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("dbprofiles.Save indent: %w", err)
	}
	return atomicWrite(s.path, data)
}

// atomicWrite écrit data dans path via un fichier temporaire du même dossier
// suivi d'un rename (atomique sur le même volume, y compris Windows).
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".db_profiles-*.tmp")
	if err != nil {
		return fmt.Errorf("dbprofiles.atomicWrite temp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op si le rename a réussi
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("dbprofiles.atomicWrite write: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("dbprofiles.atomicWrite sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("dbprofiles.atomicWrite close: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("dbprofiles.atomicWrite rename: %w", err)
	}
	return nil
}

// Mutate exécute fn sur le File chargé puis le persiste, le tout sous verrou
// exclusif (cycle read-modify-write atomique). Si fn retourne une erreur, le
// fichier n'est PAS modifié.
func (s *Store) Mutate(fn func(*File) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadLocked()
	if err != nil {
		return err
	}
	if err := fn(f); err != nil {
		return err
	}
	return s.saveLocked(f)
}

// --- Helpers sur File (pas d'IO) ---

// FindKey retourne la clé gamertag réelle (insensible à la casse) pour (slug, gt)
// et si elle existe.
func (f *File) FindKey(slug, gt string) (string, bool) {
	for k := range f.Profiles[slug] {
		if strings.EqualFold(k, gt) {
			return k, true
		}
	}
	return "", false
}

// Get retourne l'entrée (slug, gt) résolue insensiblement à la casse.
func (f *File) Get(slug, gt string) (Entry, bool) {
	if key, ok := f.FindKey(slug, gt); ok {
		return f.Profiles[slug][key], true
	}
	return Entry{}, false
}

// Set insère/remplace l'entrée (slug, key). Crée la sous-map si besoin.
func (f *File) Set(slug, key string, e Entry) {
	if f.Profiles == nil {
		f.Profiles = map[string]map[string]Entry{}
	}
	if f.Profiles[slug] == nil {
		f.Profiles[slug] = map[string]Entry{}
	}
	f.Profiles[slug][key] = e
}

// Remove supprime l'entrée (slug, gt) (insensible à la casse) et retourne true
// si une entrée a été retirée. Nettoie la sous-map du titre si elle devient vide.
func (f *File) Remove(slug, gt string) bool {
	key, ok := f.FindKey(slug, gt)
	if !ok {
		return false
	}
	delete(f.Profiles[slug], key)
	if len(f.Profiles[slug]) == 0 {
		delete(f.Profiles, slug)
	}
	return true
}

// ActiveTitlesForGamertag retourne les slugs des titres ACTIFS (sync_enabled !=
// false) pour ce gamertag (insensible à la casse), en excluant éventuellement un
// slug (utile pour « après cette opération, en reste-t-il ? »).
func (f *File) ActiveTitlesForGamertag(gt string, exclude ...string) []string {
	excluded := ""
	if len(exclude) > 0 {
		excluded = exclude[0]
	}
	var out []string
	for slug, players := range f.Profiles {
		if slug == excluded {
			continue
		}
		for k, e := range players {
			if strings.EqualFold(k, gt) && e.IsActive() {
				out = append(out, slug)
				break
			}
		}
	}
	return out
}

// --- Mutations de haut niveau, validation-aware (invariant « ≥ 1 titre actif ») ---

// SetSyncEnabled bascule sync_enabled pour (slug, gt). Désactiver le DERNIER titre
// actif d'un gamertag est refusé (ErrLastActiveTitle). Entrée absente → ErrEntryNotFound.
func (s *Store) SetSyncEnabled(slug, gt string, enabled bool) error {
	return s.Mutate(func(f *File) error {
		key, ok := f.FindKey(slug, gt)
		if !ok {
			return ErrEntryNotFound
		}
		if !enabled && len(f.ActiveTitlesForGamertag(gt, slug)) == 0 {
			return ErrLastActiveTitle // ce titre est le dernier actif
		}
		e := f.Profiles[slug][key]
		v := enabled
		e.SyncEnabled = &v
		f.Profiles[slug][key] = e
		return nil
	})
}

// RemoveEntry retire le couple (slug, gt) de db_profiles.json (étape « purge »).
// Refusé si ce titre est le dernier ACTIF du gamertag (ErrLastActiveTitle) afin de
// ne jamais laisser un joueur sans aucun titre. Entrée absente → ErrEntryNotFound.
// La suppression des fichiers de données est de la responsabilité du caller.
func (s *Store) RemoveEntry(slug, gt string) error {
	return s.Mutate(func(f *File) error {
		entry, ok := f.Get(slug, gt)
		if !ok {
			return ErrEntryNotFound
		}
		// Retirer un titre ACTIF qui est le dernier actif laisserait 0 actif.
		if entry.IsActive() && len(f.ActiveTitlesForGamertag(gt, slug)) == 0 {
			return ErrLastActiveTitle
		}
		f.Remove(slug, gt)
		return nil
	})
}
