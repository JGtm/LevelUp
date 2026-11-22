// Package v2 — fetchers_known_run_test.go : le delta des nouveaux matchs ne s'arrête plus au premier
// match connu mais après une suite de domain.DeltaKnownRun connus d'affilée (D-23) : un match rangé
// d'abord par un coéquipier ne cache pas les inconnus juste en dessous.
package v2

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync/historystate"
)

// listDeltaPaged rend la découverte d'un joueur à l'historique complet (delta).
func listDeltaPaged(t *testing.T, c *pagedClient, known map[string]bool) []string {
	t.Helper()
	l, err := catchUpProvider(c).ListUnknownMatches(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "999"},
		known, HistoryState{})
	if err != nil {
		t.Fatalf("ListUnknownMatches: %v", err)
	}
	return l.Unknown
}

// Le joueur joue A seul, puis B avec un coéquipier ; le coéquipier range B en premier : B est connu du
// joueur, A non. La liste (du plus récent au plus ancien) est [B, A, ...anciens connus].
func TestListDelta_ConnuRangeParUnCoequipier_LInconnuEnDessousEstPris(t *testing.T) {
	ids := matchIDs("m", 100)
	b, a := ids[0], ids[1]
	known := knownOfIDs(ids[2:])
	known[b] = true

	if got := listDeltaPaged(t, newPagedClient(ids), known); !equalSlice(got, []string{a}) {
		t.Errorf("Unknown = %v, want [%s] : le delta ne doit pas s'arrêter sur %s", got, a, b)
	}
}

// 24 connus puis un inconnu : la suite est incomplète, l'inconnu est pris (il est le dernier de la page 0).
func TestListDelta_VingtQuatreConnusPuisUnInconnu_LInconnuEstPris(t *testing.T) {
	ids := matchIDs("m", 100)
	known := knownOfIDs(ids[:24], ids[25:])
	if got := listDeltaPaged(t, newPagedClient(ids), known); !equalSlice(got, []string{ids[24]}) {
		t.Errorf("Unknown = %v, want [%s]", got, ids[24])
	}
}

// 25 connus d'affilée : arrêt, l'inconnu qui suit n'est pas pris ; un inconnu avant la suite remet le
// compte à zéro (le second inconnu, à 24 connus du premier, est pris).
func TestListDelta_VingtCinqConnusDAffilee_Arret(t *testing.T) {
	if domain.DeltaKnownRun != 25 {
		t.Fatalf("DeltaKnownRun = %d, les scénarios supposent 25", domain.DeltaKnownRun)
	}
	ids := matchIDs("m", 200)
	c := newPagedClient(ids)
	known := knownOfIDs(ids[:25], ids[26:])
	if got := listDeltaPaged(t, c, known); len(got) != 0 {
		t.Errorf("Unknown = %v, want vide : 25 connus d'affilée arrêtent le delta avant %s", got, ids[25])
	}
	if len(c.starts) != 1 {
		t.Errorf("%d pages lues %v, want 1 (la suite se complète à la fin de la page 0)", len(c.starts), c.starts)
	}

	known = knownOfIDs(ids[:10], ids[11:35], ids[36:])
	want := []string{ids[10], ids[35]}
	if got := listDeltaPaged(t, newPagedClient(ids), known); !equalSlice(got, want) {
		t.Errorf("Unknown = %v, want %v : l'inconnu remet le compte à zéro", got, want)
	}
}

// Appels d'API de la découverte en historique complet. Sans nouveau match : une page, comme avant.
// Avec un nouveau match : deux pages (avant : une), la seconde complète la suite de 25 connus.
func TestListDelta_AppelsDAPI(t *testing.T) {
	ids := matchIDs("m", 300)

	c := newPagedClient(ids)
	if got := listDeltaPaged(t, c, knownOfIDs(ids)); len(got) != 0 || len(c.starts) != 1 {
		t.Errorf("sans nouveau match : %d inconnus, %d pages %v, want 0 et 1 (avant : 1)", len(got), len(c.starts), c.starts)
	}

	c = newPagedClient(ids)
	if got := listDeltaPaged(t, c, knownOfIDs(ids[1:])); len(got) != 1 || len(c.starts) != 2 {
		t.Errorf("un nouveau match : %d inconnus, %d pages %v, want 1 et 2 (avant : 1)", len(got), len(c.starts), c.starts)
	}

	// Garde-rail de pages inchangé : aucun connu ne se rencontre, 20 pages au plus.
	c = newPagedClient(matchIDs("x", 1000))
	if got := listDeltaPaged(t, c, map[string]bool{}); len(got) != 500 || len(c.starts) != 20 {
		t.Errorf("sans connu : %d inconnus, %d pages, want 500 et 20", len(got), len(c.starts))
	}
}

// Rattrapage, marque-page sans ancre : le delta du dessus prend l'inconnu A sous le connu B, et
// l'estimation du nombre d'entrées arrivées (A et B) situe le marque-page sans reprise depuis 0.
func TestListCatchUp_SansAncre_ConnuRangeParUnCoequipier(t *testing.T) {
	old := matchIDs("m", 1000)
	b, a := "b-coequipier", "a-seul"
	known := knownOfIDs(old[:600])
	known[b] = true
	cursor := &historystate.Cursor{Offset: 600, LastMatchID: old[599]}
	list := append([]string{b, a}, old...)

	before := fallbackCount()
	l := listCatchUpOf(t, newPagedClient(list), known, cursor)

	if len(l.Unknown) == 0 || l.Unknown[0] != a {
		t.Fatalf("Unknown = %v..., want %s en tête", l.Unknown[:min(3, len(l.Unknown))], a)
	}
	if !equalSlice(l.Unknown[1:], old[600:600+historystate.PassBudget()-1]) {
		t.Errorf("la suite du marque-page n'est pas lue à la bonne position (%d inconnus)", len(l.Unknown))
	}
	if got := fallbackCount() - before; got != 0 {
		t.Errorf("reprise depuis 0 comptée %d fois, want 0", got)
	}
}

// Rattrapage, marque-page avec ancre : le haut de liste est lu jusqu'à l'ancre, A est pris.
func TestListCatchUp_Ancre_ConnuRangeParUnCoequipier(t *testing.T) {
	old := matchIDs("m", 1000)
	b, a := "b-coequipier", "a-seul"
	known := knownOfIDs(old[:600])
	known[b] = true
	cursor := &historystate.Cursor{Offset: 600, LastMatchID: old[599], HeadID: old[0]}
	list := append([]string{b, a}, old...)

	before := fallbackCount()
	l := listCatchUpOf(t, newPagedClient(list), known, cursor)

	if len(l.Unknown) == 0 || l.Unknown[0] != a {
		t.Fatalf("Unknown = %v..., want %s en tête", l.Unknown[:min(3, len(l.Unknown))], a)
	}
	if !equalSlice(l.Unknown[1:], old[600:600+historystate.PassBudget()-1]) {
		t.Errorf("la suite du marque-page n'est pas lue à la bonne position (%d inconnus)", len(l.Unknown))
	}
	if got := fallbackCount() - before; got != 0 {
		t.Errorf("reprise depuis 0 comptée %d fois, want 0", got)
	}
}
