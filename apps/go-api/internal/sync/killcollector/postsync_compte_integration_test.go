//go:build integration

package killcollector

// postsync_compte_integration_test.go — LA TAILLE DU BACKLOG SE COMPTE UNE FOIS PAR CYCLE (revue du
// correctif J7, 2026-09-27).
//
// Le `COUNT(*)` sans borne est la plus chere des lectures du backlog. La pagination (matchs sans
// carte en tete) lit jusqu a 8 pages par cycle : si chaque page recomptait, un cycle payait 8
// `COUNT(*)` jetes, sous le verrou de passe.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : lire les pages suivantes par `backlogAJour` (qui compte) au
// lieu de `pageDuBacklog`.

import (
	"context"
	"database/sql"
	"testing"
)

func TestRunPostSync_UnSeulComptageParCyclePagine(t *testing.T) {
	hookDuTest = NewPostSyncHook(racineDepot(t), 2)
	hookDuTest.horizon = 2
	// m6, m5, m4 sans carte : la premiere page (m6, m5) est vide, le cycle lit au moins deux
	// autres pages (m4, m3 puis m2, m1).
	cycle := backlogAvecCarte(t, 6, 3)

	comptages := 0
	dOrigine := tailleDuBacklog
	tailleDuBacklog = func(ctx context.Context, db *sql.DB) int {
		comptages++
		return dOrigine(ctx, db)
	}
	t.Cleanup(func() { tailleDuBacklog = dOrigine })

	verifierDemandes(t, cycle(), []string{"m3", "m2"})
	if comptages != 1 {
		t.Errorf("taille du backlog comptee %d fois sur un cycle de trois pages, attendu 1 — chaque "+
			"page relance le COUNT(*) complet", comptages)
	}
}
