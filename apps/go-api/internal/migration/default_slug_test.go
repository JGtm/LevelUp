package migration_test

import (
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

// TestDefaultSlugMatchesTitle verrouille l'égalité entre migration.DefaultSlug
// (dupliqué pour garder le package migration sans dépendance sur domain/title) et
// title.DefaultSlug, source de vérité. Toute dérive casserait le routage par
// titre + le ledger (title_slug écrit sous un slug ≠ de celui des chemins DB).
func TestDefaultSlugMatchesTitle(t *testing.T) {
	if migration.DefaultSlug != title.DefaultSlug {
		t.Errorf("migration.DefaultSlug = %q, title.DefaultSlug = %q — doivent être identiques",
			migration.DefaultSlug, title.DefaultSlug)
	}
}
