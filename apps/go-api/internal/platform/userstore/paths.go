// Package userstore — paths.go : le chemin du store des comptes.
//
// Seul endroit du dépôt où le nom du fichier est écrit (ratchet
// internal/archlint/no_users_json_literal_test.go). Vit ICI, et non dans
// internal/config, pour que les CLI légers (cmd/admin) qui ne connaissent que
// le répertoire d'auth n'importent pas config — qui tire DuckDB/CGO (revue de
// clôture du 2026-09-16).
package userstore

import "path/filepath"

// FilePathIn rend le chemin du store des comptes (users.json) sous le
// répertoire d'auth donné.
func FilePathIn(authDir string) string {
	return filepath.Join(authDir, "users.json")
}
