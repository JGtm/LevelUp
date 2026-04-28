package static

import "path"

// URL compose l'URL absolue (relative au domaine) d'un asset statique title-scopé.
//
// Format : /static/{folder(k)}/{titleSlug}/{id}{ext}.
//
// L'id doit être pré-encodé path-safe par le caller (couche 3 / adapter title) —
// ce package ne fait pas d'encodage spécifique au Kind (espaces dans map names,
// id numérique pour médailles, format CSR rank, etc. — toutes ces règles vivent
// dans l'adapter title-spécifique).
//
// Retourne "" si k est invalide, ou si titleSlug ou id sont vides.
// Si ext est vide, aucune extension n'est ajoutée — utile pour les chemins déjà
// extension-included composés en amont.
func URL(k Kind, titleSlug, id, ext string) string {
	if !k.Valid() || titleSlug == "" || id == "" {
		return ""
	}
	return path.Join(MountPoint, Folder(k), titleSlug, id+ext)
}
