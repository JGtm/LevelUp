//go:build integration

package persist

import (
	"strings"
	"testing"
)

// sentinelleDeType rend un littéral SQL non nul et « vrai » pour le type d'une colonne : de quoi
// distinguer, dans la vue, une valeur lue d'une valeur absente (le booléen absent vaut FALSE).
func sentinelleDeType(dataType string) string {
	switch strings.ToUpper(dataType) {
	case "VARCHAR":
		return "'x'"
	case "BOOLEAN":
		return "TRUE"
	default:
		return "1"
	}
}

// TestEnrichmentColumnStage_LieALaVueMergeOnRead lie les deux tables de stages : enrichmentColumnStage
// (persist, écrivain post-sync) et pmeColumnStage (migration, vue player_match_enrichment_latest). Une
// colonne écrite par BatchUpdateColumn doit exister, être projetée par la vue, et y être lue sous le
// stage que l'écrivain lui donne : sinon la ligne est écrite puis jamais servie.
func TestEnrichmentColumnStage_LieALaVueMergeOnRead(t *testing.T) {
	db := openEnrichmentTestDB(t)

	for col, stage := range enrichmentColumnStage {
		var dataType string
		if err := db.QueryRow(`SELECT data_type FROM information_schema.columns
			WHERE table_name = 'player_match_enrichment' AND column_name = ?`, col).Scan(&dataType); err != nil {
			t.Errorf("%s : absente de player_match_enrichment (%v) — la migration doit l'ajouter", col, err)
			continue
		}
		var projetee int
		if err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
			WHERE table_name = 'player_match_enrichment_latest' AND column_name = ?`, col).Scan(&projetee); err != nil || projetee != 1 {
			t.Errorf("%s : non projetée par player_match_enrichment_latest — ajouter {%q, %q} à pmeColumnStage "+
				"(internal/migration/steps_player_append_only_match_enrichment.go)", col, col, stage)
			continue
		}

		mid := "lien-" + col
		lit := sentinelleDeType(dataType)
		// col vient de la table blanche enrichmentColumnStage : jamais d'entrée externe.
		if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, `+col+`, stage) VALUES (?, `+lit+`, ?)`, mid, stage); err != nil {
			t.Errorf("%s : insertion sous le stage %q : %v", col, stage, err)
			continue
		}
		var lue bool
		if err := db.QueryRow(`SELECT `+col+` IS NOT NULL AND CAST(`+col+` AS VARCHAR) <> 'false'
			FROM player_match_enrichment_latest WHERE match_id = ?`, mid).Scan(&lue); err != nil || !lue {
			t.Errorf("%s : la vue ne sert pas la valeur écrite sous le stage %q — enrichmentColumnStage et "+
				"pmeColumnStage divergent sur le stage propriétaire de cette colonne", col, stage)
		}
	}
}
