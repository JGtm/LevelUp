// Package config — config_paths.go : chemins dérivés de la configuration qui
// ne relèvent pas du PathResolver (celui-ci couvre l'arbre data/titles, les
// tokens et les caches ; les fichiers d'auth locale vivent sous AuthDir, qui
// est lui-même surchargeable par LEVELUP_AUTH_DIR).
package config

import "path/filepath"

// UsersFilePath rend le chemin du store des comptes utilisateurs
// (data/auth/users.json par défaut). Trois appelants le construisaient à la main
// le 2026-09-16 (cmd/server, api/server, cmd/levelup identity) : au 3e
// exemplaire, la règle 6 de CLAUDE.md impose le helper ET le garde-rail
// internal/archlint/no_users_json_literal_test.go.
func (c *AppConfig) UsersFilePath() string {
	return UsersFilePathIn(c.AuthDir)
}

// UsersFilePathIn rend le chemin du store des comptes sous un répertoire d'auth
// donné — pour les CLI qui reçoivent ce répertoire en drapeau (cmd/admin) sans
// charger AppConfig. Seul endroit du dépôt où le nom du fichier est écrit.
func UsersFilePathIn(authDir string) string {
	return filepath.Join(authDir, "users.json")
}
