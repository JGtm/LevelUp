package port

import "errors"

// ErrAppearanceProfileNotFound : le player_slug du diagnostic d'apparence ne correspond à aucun
// profil de db_profiles.json (la couche HTTP le répond en 404).
var ErrAppearanceProfileNotFound = errors.New("profil apparence introuvable")
