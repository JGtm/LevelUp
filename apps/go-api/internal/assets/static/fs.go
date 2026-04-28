package static

import "path/filepath"

// AbsRoot retourne le chemin absolu de la racine /static/ sur le filesystem
// pour un repoRoot donné.
//
// Utilisé par le FileServer (couche 1) et l'outillage de migration (cmd/migrate-static-paths).
func AbsRoot(repoRoot string) string {
	return filepath.Join(repoRoot, "static")
}

// AbsKindRoot retourne le répertoire absolu d'un (kind, titleSlug) sur le filesystem.
//
// Format : {repoRoot}/static/{folder(k)}/{titleSlug}.
//
// Retourne "" si k est invalide ou titleSlug est vide. repoRoot peut être "."
// pour un chemin relatif au CWD.
func AbsKindRoot(repoRoot string, k Kind, titleSlug string) string {
	if !k.Valid() || titleSlug == "" {
		return ""
	}
	return filepath.Join(AbsRoot(repoRoot), Folder(k), titleSlug)
}
