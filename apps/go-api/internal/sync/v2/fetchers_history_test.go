// Package v2 — fetchers_history_test.go : découverte d'un joueur à l'historique incomplet
// (rattrapage, marque-page) et non-régression du delta ordinaire.
package v2

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/historystate"
)

// pagedClient est un HaloClient dont seule la liste des matchs sert : la liste de l'API d'un joueur,
// du plus récent au plus ancien, lue par page (start, count) ; il note chaque décalage demandé.
type pagedClient struct {
	ids     []string
	starts  []int
	failAt  int // décalage qui échoue ; -1 = aucun
	failErr error
}

func newPagedClient(ids []string) *pagedClient { return &pagedClient{ids: ids, failAt: -1} }

func (c *pagedClient) GetMatchHistory(_ context.Context, _, _ string, start, count int) ([]syncpkg.MatchHistoryEntry, error) {
	c.starts = append(c.starts, start)
	if c.failAt >= 0 && start == c.failAt {
		return nil, c.failErr
	}
	if start >= len(c.ids) {
		return nil, nil
	}
	end := min(start+count, len(c.ids))
	out := make([]syncpkg.MatchHistoryEntry, 0, end-start)
	for _, id := range c.ids[start:end] {
		out = append(out, syncpkg.MatchHistoryEntry{MatchID: id})
	}
	return out, nil
}

func (c *pagedClient) GetMatchStats(context.Context, string) (map[string]any, error) { return nil, nil }
func (c *pagedClient) GetMatchSkill(context.Context, string, []string) (map[string]*syncpkg.MatchSkillData, error) {
	return nil, nil
}
func (c *pagedClient) GetHighlightEventsChunk(context.Context, string) ([]byte, int, bool, error) {
	return nil, 0, false, nil
}

// startsBetween compte les lectures dont le décalage est dans [lo, hi).
func (c *pagedClient) startsBetween(lo, hi int) int {
	n := 0
	for _, s := range c.starts {
		if s >= lo && s < hi {
			n++
		}
	}
	return n
}

// matchIDs fabrique n identifiants ordonnés "<prefix>0000" ...
func matchIDs(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s%05d", prefix, i)
	}
	return out
}

// knownOfIDs range des identifiants dans un ensemble connu.
func knownOfIDs(idLists ...[]string) map[string]bool {
	k := map[string]bool{}
	for _, l := range idLists {
		for _, id := range l {
			k[id] = true
		}
	}
	return k
}

func catchUpProvider(c *pagedClient) MatchListProvider {
	return NewMatchListProvider(func(string, string) HaloClient { return c }, "matchmaking", 25, 20)
}

func listCatchUpOf(t *testing.T, c *pagedClient, known map[string]bool, cur *historystate.Cursor) Listing {
	t.Helper()
	l, err := catchUpProvider(c).ListUnknownMatches(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "999"},
		known, HistoryState{Incomplete: true, Cursor: cur})
	if err != nil {
		t.Fatalf("ListUnknownMatches: %v", err)
	}
	return l
}

// Rattrapage sans marque-page : les connus sont sautés (le delta s'y arrêterait), les inconnus
// gardés dans l'ordre de l'API, jusqu'à la fin de la liste.
func TestListCatchUp_FinVue(t *testing.T) {
	all := matchIDs("m", 100)
	var wantUnknown []string
	known := map[string]bool{}
	for i, id := range all {
		if i%3 == 0 {
			known[id] = true
		} else {
			wantUnknown = append(wantUnknown, id)
		}
	}
	cases := []struct {
		name string
		n    int // taille de la liste de l'API
	}{
		{"fin sur une page vide (liste multiple de la page)", 100},
		{"fin sur une page partielle", 90},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ids := all[:tc.n]
			var want []string
			for _, id := range wantUnknown {
				if id <= ids[len(ids)-1] {
					want = append(want, id)
				}
			}
			l := listCatchUpOf(t, newPagedClient(ids), known, nil)
			if !equalSlice(l.Unknown, want) {
				t.Errorf("Unknown = %d ids, want %d (ordre de l'API, connus sautés)", len(l.Unknown), len(want))
			}
			if !l.EndSeen {
				t.Error("EndSeen = false, la fin de la liste a été lue")
			}
			if l.Proposed != nil {
				t.Errorf("Proposed = %+v, aucune position à proposer quand la fin est vue", l.Proposed)
			}
		})
	}
}

// Le budget d'une passe coupe la lecture : la fin n'est pas vue, la position proposée est juste
// après le dernier inconnu listé.
func TestListCatchUp_Budget(t *testing.T) {
	budget := historystate.PassBudget()
	ids := matchIDs("m", 1000)
	l := listCatchUpOf(t, newPagedClient(ids), map[string]bool{}, nil)
	if len(l.Unknown) != budget {
		t.Fatalf("len(Unknown) = %d, want budget %d", len(l.Unknown), budget)
	}
	if !equalSlice(l.Unknown, ids[:budget]) {
		t.Error("les inconnus gardés ne sont pas les plus récents dans l'ordre de l'API")
	}
	if l.EndSeen {
		t.Error("EndSeen = true alors que le budget a coupé la lecture")
	}
	want := historystate.Cursor{Offset: budget, LastMatchID: ids[budget-1], HeadID: ids[0]}
	if l.Proposed == nil || *l.Proposed != want {
		t.Errorf("Proposed = %+v, want %+v", l.Proposed, want)
	}
}

// Le budget compte les inconnus, pas les entrées lues : des connus au milieu ne le consomment pas,
// et le marque-page proposé saute au-delà d'eux.
func TestListCatchUp_BudgetNeComptePasLesConnus(t *testing.T) {
	budget := historystate.PassBudget()
	ids := matchIDs("m", 1000)
	known := knownOfIDs(ids[:600]) // le haut de la liste est déjà rangé : le delta s'arrêterait en 0
	l := listCatchUpOf(t, newPagedClient(ids), known, nil)
	if !equalSlice(l.Unknown, ids[600:600+budget]) {
		t.Fatalf("Unknown = %d ids, want %d ids à partir de la position 600", len(l.Unknown), budget)
	}
	want := historystate.Cursor{Offset: 600 + budget, LastMatchID: ids[600+budget-1], HeadID: ids[0]}
	if l.Proposed == nil || *l.Proposed != want {
		t.Errorf("Proposed = %+v, want %+v", l.Proposed, want)
	}
}

// Borne de sûreté de profondeur : une liste de connus plus profonde que historystate.MaxDepth
// s'arrête là, sans fin vue, avec la position de la dernière entrée lue.
func TestListCatchUp_BorneDeProfondeur(t *testing.T) {
	ids := matchIDs("m", historystate.MaxDepth+500)
	c := newPagedClient(ids)
	l := listCatchUpOf(t, c, knownOfIDs(ids), nil)
	if len(l.Unknown) != 0 {
		t.Errorf("len(Unknown) = %d, want 0 (tout est connu)", len(l.Unknown))
	}
	if l.EndSeen {
		t.Error("EndSeen = true : la borne de profondeur n'est pas une fin d'historique")
	}
	want := historystate.Cursor{Offset: historystate.MaxDepth, LastMatchID: ids[historystate.MaxDepth-1], HeadID: ids[0]}
	if l.Proposed == nil || *l.Proposed != want {
		t.Errorf("Proposed = %+v, want %+v", l.Proposed, want)
	}
	if got := c.startsBetween(historystate.MaxDepth, historystate.MaxDepth*2); got != 0 {
		t.Errorf("%d lectures au-delà de la borne de profondeur", got)
	}
}

// Reprise au marque-page : le haut est lu pour compter les nouveaux matchs, puis la lecture saute
// directement à la page de recouvrement, sans reparcourir le milieu de la liste.
func TestListCatchUp_RepriseAuMarquePage(t *testing.T) {
	budget := historystate.PassBudget()
	old := matchIDs("m", 1000)
	known := knownOfIDs(old[:600])
	cursor := &historystate.Cursor{Offset: 600, LastMatchID: old[599]}

	t.Run("sans nouveau match", func(t *testing.T) {
		c := newPagedClient(old)
		l := listCatchUpOf(t, c, known, cursor)
		if !equalSlice(l.Unknown, old[600:600+budget]) {
			t.Fatalf("Unknown = %d ids, want %d ids à partir de la position 600", len(l.Unknown), budget)
		}
		want := historystate.Cursor{Offset: 600 + budget, LastMatchID: old[600+budget-1], HeadID: old[0]}
		if l.Proposed == nil || *l.Proposed != want {
			t.Errorf("Proposed = %+v, want %+v", l.Proposed, want)
		}
		// Pas de rebalayage : seules la première page (delta du dessus) et la page de
		// recouvrement (575) précèdent la zone à lire.
		if got := c.startsBetween(25, 575); got != 0 {
			t.Errorf("%d lectures entre les positions 25 et 575 : la liste a été reparcourue", got)
		}
	})

	t.Run("avec N nouveaux matchs", func(t *testing.T) {
		const n = 10
		fresh := matchIDs("n", n)
		list := append(append([]string{}, fresh...), old...)
		c := newPagedClient(list)
		l := listCatchUpOf(t, c, known, cursor)
		// Les N nouveaux comptent dans le budget : N + (budget - N) anciens.
		want := append(append([]string{}, fresh...), old[600:600+budget-n]...)
		if !equalSlice(l.Unknown, want) {
			t.Fatalf("Unknown = %d ids, want %d (nouveaux d'abord, puis la suite du marque-page)", len(l.Unknown), len(want))
		}
		last := 600 + budget - n - 1
		wantCur := historystate.Cursor{Offset: n + last + 1, LastMatchID: old[last], HeadID: fresh[0]}
		if l.Proposed == nil || *l.Proposed != wantCur {
			t.Errorf("Proposed = %+v, want %+v (position dans la liste d'aujourd'hui)", l.Proposed, wantCur)
		}
		if got := c.startsBetween(25, 575); got != 0 {
			t.Errorf("%d lectures entre les positions 25 et 575 : la liste a été reparcourue", got)
		}
	})

	t.Run("fin vue après la reprise", func(t *testing.T) {
		short := old[:650]
		l := listCatchUpOf(t, newPagedClient(short), knownOfIDs(old[:600]), cursor)
		if !equalSlice(l.Unknown, old[600:650]) || !l.EndSeen || l.Proposed != nil {
			t.Errorf("Listing = {%d inconnus, EndSeen %v, Proposed %+v}, want {50, true, nil}", len(l.Unknown), l.EndSeen, l.Proposed)
		}
	})
}

// Page de recouvrement qui ne contient pas le dernier match du marque-page (liste remaniée) : la
// position n'est plus fiable, la lecture repart du début, et le compteur le dit.
func TestListCatchUp_RecouvrementRate_RepriseDepuisZero(t *testing.T) {
	ids := matchIDs("m", 1000)
	known := knownOfIDs(ids[:600])
	cursor := &historystate.Cursor{Offset: 600, LastMatchID: "disparu"}
	title := ctxkeys.TitleSlug(context.Background())
	before := observability.LoadCounterT(title, counterHistoryCursorFallback)

	c := newPagedClient(ids)
	l := listCatchUpOf(t, c, known, cursor)

	if !equalSlice(l.Unknown, ids[600:600+historystate.PassBudget()]) {
		t.Errorf("Unknown = %d ids : la reprise depuis le début doit sauter les connus et trouver les mêmes inconnus", len(l.Unknown))
	}
	if got := c.startsBetween(25, 575); got == 0 {
		t.Error("aucune lecture du milieu de la liste : la reprise depuis le début n'a pas eu lieu")
	}
	if got := observability.LoadCounterT(title, counterHistoryCursorFallback) - before; got != 1 {
		t.Errorf("compteur de reprise depuis le début +%d, want +1", got)
	}
}

// Marque-page proche du haut : la page de recouvrement est bornée à 0, la lecture ordinaire suffit.
func TestListCatchUp_MarquePageProcheDuHaut(t *testing.T) {
	ids := matchIDs("m", 60)
	known := knownOfIDs(ids[:10])
	l := listCatchUpOf(t, newPagedClient(ids), known, &historystate.Cursor{Offset: 10, LastMatchID: ids[9]})
	if !equalSlice(l.Unknown, ids[10:]) || !l.EndSeen {
		t.Errorf("Listing = {%d inconnus, EndSeen %v}, want {50, true}", len(l.Unknown), l.EndSeen)
	}
}

// Avec un marque-page, tant qu'aucun match connu n'est rencontré en haut de liste, le décalage
// n'est pas translatable : budget atteint sans position proposée ; liste courte vue en entier = fin.
func TestListCatchUp_MarquePageSansPremierConnu(t *testing.T) {
	cursor := &historystate.Cursor{Offset: 600, LastMatchID: "x"}

	l := listCatchUpOf(t, newPagedClient(matchIDs("m", 1000)), map[string]bool{}, cursor)
	if len(l.Unknown) != historystate.PassBudget() || l.EndSeen || l.Proposed != nil {
		t.Errorf("Listing = {%d inconnus, EndSeen %v, Proposed %+v}, want {budget, false, nil}", len(l.Unknown), l.EndSeen, l.Proposed)
	}

	l = listCatchUpOf(t, newPagedClient(matchIDs("m", 30)), map[string]bool{}, cursor)
	if len(l.Unknown) != 30 || !l.EndSeen || l.Proposed != nil {
		t.Errorf("Listing = {%d inconnus, EndSeen %v, Proposed %+v}, want {30, true, nil}", len(l.Unknown), l.EndSeen, l.Proposed)
	}
}

// Une erreur de l'API en cours de rattrapage est rendue (le cycle note l'échec de découverte du
// joueur et ne range rien).
func TestListCatchUp_ErreurAPI(t *testing.T) {
	ids := matchIDs("m", 500)
	boom := errors.New("api 500")
	for _, failAt := range []int{0, 100} {
		c := newPagedClient(ids)
		c.failAt, c.failErr = failAt, boom
		_, err := catchUpProvider(c).ListUnknownMatches(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "999"},
			map[string]bool{}, HistoryState{Incomplete: true})
		if !errors.Is(err, boom) {
			t.Errorf("failAt=%d : err = %v, want l'erreur de l'API enveloppée", failAt, err)
		}
	}
}

// Historique complet : delta inchangé (arrêt au premier connu, au plus 20 pages), sans fin vue ni
// position proposée.
func TestListUnknown_HistoriqueComplet_DeltaInchange(t *testing.T) {
	ids := matchIDs("m", 1000)

	known := knownOfIDs(ids[30:])
	l, err := catchUpProvider(newPagedClient(ids)).ListUnknownMatches(context.Background(), PlayerProfile{XUID: "999"}, known, HistoryState{})
	if err != nil {
		t.Fatal(err)
	}
	if !equalSlice(l.Unknown, ids[:30]) || l.EndSeen || l.Proposed != nil {
		t.Errorf("delta : {%d inconnus, EndSeen %v, Proposed %+v}, want {30, false, nil}", len(l.Unknown), l.EndSeen, l.Proposed)
	}

	c := newPagedClient(ids)
	l, err = catchUpProvider(c).ListUnknownMatches(context.Background(), PlayerProfile{XUID: "999"}, map[string]bool{}, HistoryState{})
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Unknown) != 20*25 || len(c.starts) != 20 {
		t.Errorf("sans connu : %d inconnus en %d pages, want 500 en 20 pages (garde-rail maxPages)", len(l.Unknown), len(c.starts))
	}
}

// Un match abandonné est sauté comme un connu, sans consommer le budget ; un match en échec pas
// encore abandonné reste listé.
func TestListCatchUp_MatchsAbandonnes(t *testing.T) {
	budget := historystate.PassBudget()
	ids := matchIDs("m", 1000)
	hist := HistoryState{Incomplete: true, Failures: []historystate.Failure{
		{MatchID: ids[0], Attempts: maxCatchUpAttempts},
		{MatchID: ids[1], Attempts: maxCatchUpAttempts + 2},
		{MatchID: ids[2], Attempts: maxCatchUpAttempts - 1}, // pas encore abandonné
	}}
	l, err := catchUpProvider(newPagedClient(ids)).ListUnknownMatches(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "999"}, map[string]bool{}, hist)
	if err != nil {
		t.Fatal(err)
	}
	if !equalSlice(l.Unknown, ids[2:2+budget]) {
		t.Errorf("Unknown = %d ids, want les %d ids à partir de la position 2 (abandonnés sautés, budget intact)", len(l.Unknown), budget)
	}
	want := historystate.Cursor{Offset: 2 + budget, LastMatchID: ids[2+budget-1], HeadID: ids[0]}
	if l.Proposed == nil || *l.Proposed != want {
		t.Errorf("Proposed = %+v, want %+v", l.Proposed, want)
	}
}

// En haut de liste, un abandonné n'est pas un nouveau match : il ne décale pas la reprise du marque-page.
func TestListCatchUp_AbandonneEnHautNeDecalePasLaReprise(t *testing.T) {
	ids := matchIDs("m", 1000)
	known := knownOfIDs(ids[2:600])
	hist := HistoryState{
		Incomplete: true,
		Cursor:     &historystate.Cursor{Offset: 600, LastMatchID: ids[599]},
		Failures: []historystate.Failure{
			{MatchID: ids[0], Attempts: maxCatchUpAttempts},
			{MatchID: ids[1], Attempts: maxCatchUpAttempts},
		},
	}
	c := newPagedClient(ids)
	l, err := catchUpProvider(c).ListUnknownMatches(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "999"}, known, hist)
	if err != nil {
		t.Fatal(err)
	}
	if !equalSlice(l.Unknown, ids[600:600+historystate.PassBudget()]) {
		t.Fatalf("Unknown = %d ids, want %d ids à partir de la position 600", len(l.Unknown), historystate.PassBudget())
	}
	if c.startsBetween(575, 576) == 0 {
		t.Errorf("lectures %v : la reprise devait se faire à 600-25=575 (aucun nouveau match)", c.starts)
	}
}
