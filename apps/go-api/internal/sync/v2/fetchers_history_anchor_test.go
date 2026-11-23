// Package v2 — fetchers_history_anchor_test.go : le marque-page situé par son ancre (la tête de
// liste de la passe précédente), au niveau de la découverte et du cycle complet.
package v2

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/historystate"
)

func fallbackCount() int64 {
	return observability.LoadCounterT(ctxkeys.TitleSlug(context.Background()), counterHistoryCursorFallback)
}

// N matchs neufs entre deux passes : la reprise est exactement à marque-page + N, lue en UNE page
// dont la première entrée est le contrôle de position ; ni page de recouvrement, ni rebalayage.
func TestListCatchUp_Ancre_NMatchsNeufs_RepriseExacte(t *testing.T) {
	budget := historystate.PassBudget()
	old := matchIDs("m", 1000)
	known := knownOfIDs(old[:600])
	cursor := &historystate.Cursor{Offset: 600, LastMatchID: old[599], HeadID: old[0]}
	const n = 7
	fresh := matchIDs("n", n)
	c := newPagedClient(append(append([]string{}, fresh...), old...))

	before := fallbackCount()
	l := listCatchUpOf(t, c, known, cursor)

	want := append(append([]string{}, fresh...), old[600:600+budget-n]...)
	if !equalSlice(l.Unknown, want) {
		t.Fatalf("Unknown = %d ids, want les %d neufs puis la suite du marque-page (%d)", len(l.Unknown), n, len(want))
	}
	last := 600 + budget - n - 1
	wantCur := historystate.Cursor{Offset: n + last + 1, LastMatchID: old[last], HeadID: fresh[0]}
	if l.Proposed == nil || *l.Proposed != wantCur {
		t.Errorf("Proposed = %+v, want %+v", l.Proposed, wantCur)
	}
	// Le marque-page 600 devient la position 607 : la lecture reprend à 606 (l'entrée de contrôle).
	if got := c.startsBetween(25, 606); got != 0 {
		t.Errorf("%d lectures entre 25 et 606 : la liste a été reparcourue (%v)", got, c.starts)
	}
	if c.startsBetween(606, 607) != 1 {
		t.Errorf("lectures %v : la reprise devait se faire à 606, une seule fois", c.starts)
	}
	if got := fallbackCount() - before; got != 0 {
		t.Errorf("reprise depuis 0 comptée %d fois, want 0", got)
	}
}

// Aucun match neuf : l'ancre est la position 0, la reprise est exactement au marque-page.
func TestListCatchUp_Ancre_SansNouveau(t *testing.T) {
	old := matchIDs("m", 1000)
	c := newPagedClient(old)
	l := listCatchUpOf(t, c, knownOfIDs(old[:600]), &historystate.Cursor{Offset: 600, LastMatchID: old[599], HeadID: old[0]})
	if !equalSlice(l.Unknown, old[600:600+historystate.PassBudget()]) {
		t.Fatalf("Unknown = %d ids, want la suite du marque-page", len(l.Unknown))
	}
	if c.startsBetween(25, 599) != 0 || c.startsBetween(599, 600) != 1 {
		t.Errorf("lectures %v : la reprise devait se faire à 599 sans lire le milieu", c.starts)
	}
}

// 30 matchs introuvables (jamais rangés, donc jamais « connus ») en haut de liste : ils sont comptés
// comme arrivés par l'ancre, la reprise reste exacte, à la passe qui les voit puis aux suivantes qui
// ne les recomptent plus. Sans l'ancre, la troisième passe reprendrait depuis 0.
func TestCycleHistory_Ancre_TrenteIntrouvablesEnHaut_ReprisesExactes(t *testing.T) {
	budget := historystate.PassBudget()
	e := newHistEnv(t, 3*budget)
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if c, ok := e.cursor(); !ok || c.Offset != budget || c.HeadID != e.ids[0] {
		t.Fatalf("passe 1 : marque-page %+v (%v), want offset %d et ancre %s", c, ok, budget, e.ids[0])
	}

	const gone = 30
	fresh := matchIDs("g", gone)
	for _, id := range fresh {
		e.fetcher.perMatchErr[id] = notFoundErr()
	}
	e.client.ids = append(append([]string{}, fresh...), e.client.ids...)
	before := fallbackCount()

	// Passe 2 : les 30 introuvables + 170 matchs du marque-page ; position 30+200+170.
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	wantOffset := gone + budget + (budget - gone)
	c, ok := e.cursor()
	if !ok || c.Offset != wantOffset || c.LastMatchID != e.ids[budget+(budget-gone)-1] || c.HeadID != fresh[0] {
		t.Fatalf("passe 2 : marque-page %+v (%v), want offset %d sur %s, ancre %s", c, ok, wantOffset, e.ids[budget+(budget-gone)-1], fresh[0])
	}
	// Le haut est lu jusqu'à l'ancre (positions 0 et 25), puis la reprise à 229 : rien entre les deux.
	if e.client.startsBetween(50, 229) != 0 {
		t.Errorf("passe 2 : lectures %v, rien attendu entre 50 et 229", e.client.starts)
	}

	// Passe 3 : plus rien d'arrivé, les introuvables ne sont pas recomptés : reprise exacte.
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	c, ok = e.cursor()
	if !ok || c.Offset != wantOffset+budget {
		t.Fatalf("passe 3 : marque-page %+v (%v), want offset %d", c, ok, wantOffset+budget)
	}
	if e.client.startsBetween(25, wantOffset-1) != 0 {
		t.Errorf("passe 3 : lectures %v, la liste a été reparcourue", e.client.starts)
	}
	if got := fallbackCount() - before; got != 0 {
		t.Errorf("reprises depuis 0 comptées : %d, want 0", got)
	}
}

// Ancre perdue (la tête de la passe précédente n'est plus dans les premières pages) : reprise depuis
// 0, comptée, et les mêmes inconnus sont trouvés (les connus sont sautés).
func TestListCatchUp_Ancre_Perdue_RepriseDepuisZeroComptee(t *testing.T) {
	old := matchIDs("m", 1000)
	known := knownOfIDs(old[:600])
	cursor := &historystate.Cursor{Offset: 600, LastMatchID: old[599], HeadID: "disparue"}
	c := newPagedClient(old)

	before := fallbackCount()
	l := listCatchUpOf(t, c, known, cursor)

	if !equalSlice(l.Unknown, old[600:600+historystate.PassBudget()]) {
		t.Errorf("Unknown = %d ids : la reprise depuis le début doit trouver les inconnus du marque-page", len(l.Unknown))
	}
	if got := fallbackCount() - before; got != 1 {
		t.Errorf("reprises depuis 0 comptées : %d, want 1", got)
	}
	if c.startsBetween(200, 575) == 0 {
		t.Errorf("lectures %v : la reprise depuis le début n'a pas eu lieu", c.starts)
	}
	want := historystate.Cursor{Offset: 600 + historystate.PassBudget(), LastMatchID: old[600+historystate.PassBudget()-1], HeadID: old[0]}
	if l.Proposed == nil || *l.Proposed != want {
		t.Errorf("Proposed = %+v, want %+v", l.Proposed, want)
	}
}

// Ancre retrouvée mais l'identifiant rangé n'est pas à sa place (liste remaniée) : reprise depuis 0,
// comptée.
func TestListCatchUp_Ancre_ControleDePositionRate_RepriseDepuisZero(t *testing.T) {
	old := matchIDs("m", 1000)
	known := knownOfIDs(old[:600])
	cursor := &historystate.Cursor{Offset: 600, LastMatchID: "autre", HeadID: old[0]}

	before := fallbackCount()
	l := listCatchUpOf(t, newPagedClient(old), known, cursor)
	if !equalSlice(l.Unknown, old[600:600+historystate.PassBudget()]) {
		t.Errorf("Unknown = %d ids, want les inconnus trouvés depuis le début", len(l.Unknown))
	}
	if got := fallbackCount() - before; got != 1 {
		t.Errorf("reprises depuis 0 comptées : %d, want 1", got)
	}
}

// Liste plus courte que le marque-page (ancre absente, fin vue dans le haut) : reprise comptée, fin vue.
func TestListCatchUp_Ancre_ListeFinieAvantL_Ancre(t *testing.T) {
	ids := matchIDs("m", 30)
	before := fallbackCount()
	l := listCatchUpOf(t, newPagedClient(ids), knownOfIDs(ids[:10]), &historystate.Cursor{Offset: 10, LastMatchID: ids[9], HeadID: "disparue"})
	if !equalSlice(l.Unknown, ids[10:]) || !l.EndSeen {
		t.Errorf("Listing = {%d inconnus, EndSeen %v}, want {20, true}", len(l.Unknown), l.EndSeen)
	}
	if got := fallbackCount() - before; got != 1 {
		t.Errorf("reprises depuis 0 comptées : %d, want 1", got)
	}
}

// Le budget est atteint dans les matchs arrivés avant même de retrouver l'ancre : aucune position
// proposée, pas une ancre perdue (aucune reprise comptée).
func TestListCatchUp_Ancre_BudgetAtteintAvantLAncre(t *testing.T) {
	old := matchIDs("m", 400)
	fresh := matchIDs("n", 250)
	c := newPagedClient(append(append([]string{}, fresh...), old...))
	before := fallbackCount()
	l := listCatchUpOf(t, c, knownOfIDs(old[:100]), &historystate.Cursor{Offset: 100, LastMatchID: old[99], HeadID: old[0]})
	if len(l.Unknown) != historystate.PassBudget() || l.EndSeen || l.Proposed != nil {
		t.Errorf("Listing = {%d inconnus, EndSeen %v, Proposed %+v}, want {budget, false, nil}", len(l.Unknown), l.EndSeen, l.Proposed)
	}
	if got := fallbackCount() - before; got != 0 {
		t.Errorf("reprises depuis 0 comptées : %d, want 0", got)
	}
}

// Un marque-page sans ancre n'existe pas pour la découverte V2 (le cycle range toujours la tête avec lui) :
// il vaut pas de marque-page. La lecture repart du début, la reprise est journalisée et comptée, et la
// passe range une ancre pour la suivante.
func TestListCatchUp_SansAncre_RepriseDepuisLeDebut(t *testing.T) {
	old := matchIDs("m", 1000)
	c := newPagedClient(old)
	before := fallbackCount()
	l := listCatchUpOf(t, c, knownOfIDs(old[:600]), &historystate.Cursor{Offset: 600, LastMatchID: old[599]})
	if !equalSlice(l.Unknown, old[600:600+historystate.PassBudget()]) {
		t.Errorf("Unknown = %d ids, want la suite des connus trouvée depuis le début", len(l.Unknown))
	}
	if c.startsBetween(25, 575) == 0 {
		t.Errorf("lectures %v : la reprise depuis le début n'a pas eu lieu", c.starts)
	}
	if got := fallbackCount() - before; got != 1 {
		t.Errorf("reprises depuis 0 comptées : %d, want 1", got)
	}
	if l.Proposed == nil || l.Proposed.HeadID != old[0] {
		t.Errorf("Proposed = %+v, want une ancre %s pour la passe suivante", l.Proposed, old[0])
	}
}
