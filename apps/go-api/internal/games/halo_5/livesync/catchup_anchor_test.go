package livesync

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/historystate"
)

// prependIDs place ces entrées, dans cet ordre, en haut de la liste ; celles dont l'identifiant commence
// par « w » sont des parties Warzone : vues par la capture, écartées, jamais rangées ni connues.
func (w *historyWorld) prependIDs(ids ...string) {
	for _, id := range ids {
		if id[0] == 'w' {
			w.excluded[id] = true
		}
	}
	w.ids = append(append([]string{}, ids...), w.ids...)
}

// idsOf numérote n identifiants de préfixe prefix, du plus récent au plus ancien.
func idsOf(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s%04d", prefix, i)
	}
	return out
}

// Trente parties Warzone en tête de liste, présentes à deux passes de suite : la deuxième reprend
// exactement au marque-page (page de recouvrement = décalage rangé - une page), sans repartir de 0 et
// sans rien relire de la tête de liste en plus.
func TestRunDelta_Ancre_TrenteWarzoneEnTeteNeFontPasRepartirDe0(t *testing.T) {
	w := newHistoryWorld(450)
	w.prependIDs(idsOf("w", 30)...)
	w.run(t, domain.SyncOptions{})
	before := w.cursor
	if before == nil || before.HeadID != "w0000" {
		t.Fatalf("marque-page après la première passe = %+v, attendu une ancre w0000 (tête de liste)", before)
	}
	beforeOffset := before.Offset
	missedBefore := observability.LoadCounter(metricCursorMissed)
	w.pageStarts, w.listStarts = nil, nil

	w.run(t, domain.SyncOptions{})

	if want := beforeOffset - backfillDefaultPageSize; len(w.pageStarts) == 0 || w.pageStarts[0] != want {
		t.Fatalf("pages de reprise = %v, attendu une première page à %d (marque-page %d - une page)", w.pageStarts, want, beforeOffset)
	}
	for _, s := range w.pageStarts {
		if s == 0 {
			t.Errorf("la passe est repartie de 0 : pages %v", w.pageStarts)
		}
	}
	if got := observability.LoadCounter(metricCursorMissed) - missedBefore; got != 0 {
		t.Errorf("compteur h5_history_cursor_missed_total +%d, attendu +0", got)
	}
	if len(w.listStarts) != 0 {
		t.Errorf("lecture de tête de liste %v : le delta a vu l'ancre, aucune lecture de plus attendue", w.listStarts)
	}
	if w.cursor == nil || w.cursor.HeadID != "w0000" || w.cursor.Offset <= beforeOffset {
		t.Errorf("marque-page après la deuxième passe = %+v, attendu plus loin que %d, ancre w0000", w.cursor, beforeOffset)
	}
}

// N parties neuves, mélange d'Arena et de Warzone, entre deux passes : le décalage est exact (l'ancienne
// tête est elle-même une Warzone, jamais connue : « jusqu'au premier connu » aurait compté ses voisines).
func TestRunDelta_Ancre_NeufsMelangesDecalageExact(t *testing.T) {
	w := newHistoryWorld(450)
	w.prependIDs(idsOf("w", 3)...) // l'ancienne tête et deux écartées derrière elle
	w.run(t, domain.SyncOptions{})
	before := *w.cursor

	fresh := []string{"a0000", "w9000", "a0001", "a0002", "w9001", "a0003", "w9002", "a0004", "a0005", "w9003"}
	w.prependIDs(fresh...)
	w.pageStarts = nil
	w.run(t, domain.SyncOptions{})

	for _, id := range fresh {
		if id[0] == 'a' && !w.stored[id] {
			t.Errorf("match neuf %s non rangé par le delta du dessus", id)
		}
	}
	if want := before.Offset + len(fresh) - backfillDefaultPageSize; len(w.pageStarts) == 0 || w.pageStarts[0] != want {
		t.Fatalf("pages de reprise = %v, attendu une première page à %d (marque-page %d + %d neufs - une page)",
			w.pageStarts, want, before.Offset, len(fresh))
	}
}

// Plus de huit pages d'entrées arrivées en tête (écartées) : l'ancre est perdue, la passe repart de 0,
// le compteur des marque-pages perdus l'enregistre.
func TestRunDelta_Ancre_PerdueRepartDe0Comptee(t *testing.T) {
	w := newHistoryWorld(450)
	w.run(t, domain.SyncOptions{})
	w.prependIDs(idsOf("w", historystate.AnchorScanLimit(backfillDefaultPageSize)+30)...)
	missedBefore := observability.LoadCounter(metricCursorMissed)
	w.pageStarts, w.listStarts = nil, nil

	w.run(t, domain.SyncOptions{})

	if len(w.pageStarts) == 0 || w.pageStarts[0] != 0 {
		t.Fatalf("pages de reprise = %v, attendu une reprise depuis 0 sans page de recouvrement", w.pageStarts)
	}
	if got := observability.LoadCounter(metricCursorMissed) - missedBefore; got != 1 {
		t.Errorf("compteur h5_history_cursor_missed_total +%d, attendu +1", got)
	}
}

// Le delta s'arrête sur un match déjà connu (rangé par un coéquipier) AVANT l'ancre : la tête de liste
// est relue page à page jusqu'à l'ancre, et le décalage est exact.
func TestRunDelta_Ancre_ConnuAuDessusDeLAncreRelitLaTete(t *testing.T) {
	w := newHistoryWorld(450)
	w.run(t, domain.SyncOptions{})
	before := *w.cursor

	fresh := idsOf("a", 5)
	w.stored[fresh[0]] = true // connu : le delta s'arrête dès la première entrée
	w.prependIDs(fresh...)
	w.pageStarts, w.listStarts = nil, nil
	missedBefore := observability.LoadCounter(metricCursorMissed)
	w.run(t, domain.SyncOptions{})

	if len(w.listStarts) != 1 || w.listStarts[0] != 0 {
		t.Errorf("lectures de tête de liste = %v, attendu une page à 0", w.listStarts)
	}
	if want := before.Offset + len(fresh) - backfillDefaultPageSize; len(w.pageStarts) == 0 || w.pageStarts[0] != want {
		t.Fatalf("pages de reprise = %v, attendu une première page à %d", w.pageStarts, want)
	}
	if got := observability.LoadCounter(metricCursorMissed) - missedBefore; got != 0 {
		t.Errorf("ancre retrouvée mais compteur de marque-pages perdus +%d", got)
	}
}

// Lecture de tête de liste en échec : pas de panique, pas de reprise depuis 0 sur un doute ; la passe
// retombe sur la règle sans ancre (décalage = entrées vues jusqu'au premier connu).
func TestRunDelta_Ancre_LectureDeTeteEnEchecRetombeSurLaRegleSansAncre(t *testing.T) {
	w := newHistoryWorld(450)
	w.run(t, domain.SyncOptions{})
	before := *w.cursor

	fresh := idsOf("a", 5)
	w.stored[fresh[0]] = true
	w.prependIDs(fresh...)
	w.pageStarts = nil
	deps := w.deps()
	deps.ListIDs = func(context.Context, canonical.PlayerIdentity, int, int) ([]string, error) {
		return nil, errors.New("GetPlayerMatches: 500")
	}
	if _, err := NewRunner(deps, nil).RunDelta(context.Background(), domain.SyncOptions{}); err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	// Règle sans ancre : un seul match vu (le connu), décalage 1 - 1 = 0.
	if want := before.Offset - backfillDefaultPageSize; len(w.pageStarts) == 0 || w.pageStarts[0] != want {
		t.Errorf("pages de reprise = %v, attendu une première page à %d (règle sans ancre)", w.pageStarts, want)
	}
}

// Marque-page ancien, sans ancre : comportement d'avant (décalage = entrées vues jusqu'au premier
// connu), aucune lecture de plus ; le marque-page rangé ensuite porte l'ancre.
func TestRunDelta_Ancre_MarquePageSansAncreGardeLaRegleActuelle(t *testing.T) {
	w := newHistoryWorld(450)
	w.run(t, domain.SyncOptions{})
	w.cursor.HeadID = "" // marque-page rangé avant l'ancre
	before := *w.cursor

	w.prependIDs(idsOf("a", 7)...)
	w.pageStarts, w.listStarts = nil, nil
	w.run(t, domain.SyncOptions{})

	if want := before.Offset + 7 - backfillDefaultPageSize; len(w.pageStarts) == 0 || w.pageStarts[0] != want {
		t.Fatalf("pages de reprise = %v, attendu une première page à %d (règle sans ancre)", w.pageStarts, want)
	}
	if len(w.listStarts) != 0 {
		t.Errorf("lectures de tête de liste %v : sans ancre, aucune attendue", w.listStarts)
	}
	if w.cursor == nil || w.cursor.HeadID != "a0000" {
		t.Errorf("marque-page rangé = %+v, attendu une ancre a0000 (tête de liste)", w.cursor)
	}
}

// Sans ancre, trente Warzone en tête à deux passes de suite : la règle actuelle les recompte comme
// nouvelles, perd le marque-page (la page de recouvrement est trop loin) et repart de 0, comme avant :
// c'est le défaut que l'ancre corrige.
func TestRunDelta_Ancre_SansAncreTrenteWarzoneRepartDe0(t *testing.T) {
	w := newHistoryWorld(450)
	w.prependIDs(idsOf("w", 30)...)
	w.run(t, domain.SyncOptions{})
	w.cursor.HeadID = ""
	w.pageStarts = nil
	missedBefore := observability.LoadCounter(metricCursorMissed)

	w.run(t, domain.SyncOptions{})

	if len(w.pageStarts) < 2 || w.pageStarts[1] != 0 {
		t.Errorf("pages de reprise = %v, attendu la page de recouvrement puis une reprise depuis 0", w.pageStarts)
	}
	if got := observability.LoadCounter(metricCursorMissed) - missedBefore; got != 1 {
		t.Errorf("compteur h5_history_cursor_missed_total +%d, attendu +1", got)
	}
}
