package livesync

import (
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
)

// Une lecture du rattrapage qui échoue (429 sur la page de reprise) ne fait pas perdre les nouveaux matchs : le
// delta du dessus les a déjà rangés quand la reprise démarre. La fin n'est pas rangée, le joueur reste en
// attente, et la passe suivante finit le travail une fois l'API redevenue saine.
func TestRunDelta_Rattrapage_ErreurDeLectureGardeLesNouveauxMatchs(t *testing.T) {
	w := newHistoryWorld(450)
	w.run(t, domain.SyncOptions{}) // marque-page {200 m0199}
	if w.cursor == nil || w.cursor.Offset != 200 {
		t.Fatalf("marque-page après la première passe = %+v, attendu {200 ...}", w.cursor)
	}

	const fresh = 7
	w.prepend(fresh)
	// m0199 est maintenant à l'entrée 206 : la page de recouvrement commence à 207 - 25 = 182.
	w.errAtStart[182] = errors.New("GetPlayerMatches: HTTP 429")
	res := w.run(t, domain.SyncOptions{})

	for i := 0; i < fresh; i++ {
		if id := fmt.Sprintf("n%04d", i); !w.stored[id] {
			t.Errorf("match neuf %s perdu : le delta du dessus doit le ranger avant la reprise", id)
		}
	}
	if res.MatchesInserted != fresh || res.CatchUpInserted != 0 {
		t.Errorf("insérés=%d dont rattrapage=%d, attendu %d nouveaux et 0 du rattrapage", res.MatchesInserted, res.CatchUpInserted, fresh)
	}
	if w.marked != 0 || !res.HistoryPending || len(res.Errors) == 0 {
		t.Errorf("fin rangée=%d pending=%v erreurs=%v, attendu 0/true/une erreur", w.marked, res.HistoryPending, res.Errors)
	}

	delete(w.errAtStart, 182)
	for i := 0; i < 4 && w.marked == 0; i++ {
		w.run(t, domain.SyncOptions{})
	}
	if len(w.stored) != 450+fresh || w.marked != 1 {
		t.Errorf("rangés=%d fin rangée=%d, attendu %d/1", len(w.stored), w.marked, 450+fresh)
	}
}
