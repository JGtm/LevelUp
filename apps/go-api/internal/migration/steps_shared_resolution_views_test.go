//go:build integration

package migration

import (
	"strings"
	"testing"
)

// TestApplyResolutionViews_VMatchFullEnEchecRemonte : la création de v_match_full qui échoue
// (ici : match_registry absente) est RENDUE par ApplyResolutionViews, jamais avalée. Une
// base sans v_match_full démarrait autrefois sans signal, puis chaque page lisant la vue
// tombait en erreur.
func TestApplyResolutionViews_VMatchFullEnEchecRemonte(t *testing.T) {
	db := openMemDB(t)
	for _, ddl := range []string{
		`CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("préparation %q : %v", ddl, err)
		}
	}
	err := ApplyResolutionViews(db)
	if err == nil {
		t.Fatal("ApplyResolutionViews doit remonter l'échec de v_match_full (match_registry absente)")
	}
	if !strings.Contains(err.Error(), "v_match_full") {
		t.Errorf("l'erreur doit nommer v_match_full, reçu : %v", err)
	}
}
