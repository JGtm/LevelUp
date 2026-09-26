//go:build integration

package killcollector

// registry_flags_integration_test.go — LE MARQUEUR TERMINAL, POSE PAR L ETAPE 1.57.
//
// POURQUOI CE FICHIER EXISTE. Le poseur des deux marqueurs de film a demenage le 2026-09-01 :
// il vivait dans l etape 1.55 (`MarkWeaponKillsDone`), supprimee avec le producteur de
// correlation. Si le demenagement etait rate — bit jamais pose — RIEN ne le dirait : la passe
// continuerait de rendre « film absent » sans erreur, et les deux rattrapages (1.57 et 1.58)
// redemanderaient A VIE les ~29 % de films irrecuperables. La panne serait un cycle qui
// travaille pour rien, indefiniment, en silence.
//
// Le test verifie donc la BOUCLE COMPLETE, pas seulement l ecriture : la passe tourne, le bit
// est pose, et le backlog cesse de proposer le match.

import (
	"context"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/sync/matchflags"
)

// TestRunPostSync_FilmAbsent_PoseLeMarqueurEtDraineLeBacklog : la propriete du demenagement.
//
//	AVANT la passe   le match est au backlog, sans marqueur
//	APRES la passe   le bit MBitFilmAbsent est pose, et le backlog ne le rend plus
func TestRunPostSync_FilmAbsent_PoseLeMarqueurEtDraineLeBacklog(t *testing.T) {
	db := baseBacklog(t)
	t0 := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	inscrireMatch(t, db, "sans-film", t0, 0)

	// Etat initial : le match est bien candidat.
	if ids, total := backlogAJour(context.Background(), db, 10); total != 1 || len(ids) != 1 || ids[0] != "sans-film" {
		t.Fatalf("backlog initial = %v (total %d), attendu [sans-film] — le test ne mesure rien "+
			"si le match n est pas candidat au depart", ids, total)
	}

	// `filmsTraces` rend systematiquement « film absent » : c est l outcome que le marqueur
	// terminal doit enregistrer.
	h := NewPostSyncHook(racineDepot(t), 5)
	RunPostSync(context.Background(), h, depsDeTest(db, &filmsTraces{}, nil), nil)

	var bits int64
	if err := db.QueryRowContext(context.Background(),
		`SELECT COALESCE(backfill_completed, 0) FROM match_registry WHERE match_id = 'sans-film'`,
	).Scan(&bits); err != nil {
		t.Fatalf("lecture backfill_completed: %v", err)
	}
	if bits&int64(matchflags.MBitFilmAbsent) == 0 {
		t.Fatalf("MBitFilmAbsent non pose (bits=%d) — le poseur n a pas suivi l etape 1.55 dans "+
			"son demenagement, et les rattrapages redemanderont ce film a vie", bits)
	}
	if bits&int64(matchflags.MBitWeaponKills) != 0 {
		t.Errorf("MBitWeaponKills pose alors qu aucune ligne n a ete publiee (bits=%d) — "+
			"la garde bit-honnete est perdue", bits)
	}

	// La boucle : le backlog draine.
	if ids, total := backlogAJour(context.Background(), db, 10); total != 0 || len(ids) != 0 {
		t.Errorf("backlog apres passe = %v (total %d), attendu vide — le marqueur terminal ne "+
			"draine pas", ids, total)
	}
}
