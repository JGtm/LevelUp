package migration

// steps_shared_social_purge_data_health.go — la migration purge_data_health_warning_notifs
// a été migrée vers internal/games/halo_infinite/migrations/steps_shared_social.go
// (Phase 1.5 b19, voie B). Le nom reste dans internal/migration/order.go (canonicalOrder).
//
// MAIS le helper firstWords RESTE ici : il est utilisé par 9 sites in-package
// (rebuilds/append-only) + exposé via migration.FirstWords (helpers_export.go, b13).
// Le retirer casserait le build.

// firstWords retourne les n premiers mots d'une SQL statement pour des messages
// d'erreur lisibles sans dumper la requête complète.
func firstWords(s string, n int) string {
	out := make([]byte, 0, 32)
	wordCount := 0
	inWord := false
	for i := 0; i < len(s) && wordCount < n; i++ {
		c := s[i]
		isSpace := c == ' ' || c == '\n' || c == '\t' || c == '\r'
		if isSpace {
			if inWord {
				wordCount++
				if wordCount < n {
					out = append(out, ' ')
				}
				inWord = false
			}
			continue
		}
		out = append(out, c)
		inWord = true
	}
	return string(out)
}
