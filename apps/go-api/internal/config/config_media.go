// config_media.go — la politique de rétention de l'original d'une vidéo convertie.

package config

import "os"

// MediaDeleteSource rend la politique effective de suppression de l'original (.mkv, .avi...) après
// une conversion HLS réussie. store est la valeur écrite dans app_settings.json
// (media_delete_source_after_transcode), nil quand elle est absente.
//
// Sous la capacité KeepMediaSources la réponse est toujours faux : l'application ne supprime jamais
// l'original d'un utilisateur de bureau, quels que soient la variable d'environnement
// LEVELUP_MEDIA_DELETE_SOURCE et le réglage. Sinon : variable d'environnement, puis réglage, puis le
// défaut de l'environnement (supprimer en production, conserver ailleurs).
//
// Unique point d'entrée des appelants : la résolution brute reste privée à ce paquet.
func (c *AppConfig) MediaDeleteSource(store *bool) bool {
	if c.Capabilities.KeepMediaSources {
		return false
	}
	return resolveMediaDeleteSource(os.Getenv(EnvMediaDeleteSource), store, c.IsProduction())
}
