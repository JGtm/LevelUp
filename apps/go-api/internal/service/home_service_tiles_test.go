// Package service — home_service_tiles_test.go : la liste de tuiles de l'Accueil est une
// FENÊTRE (D-1, D-8) : la tête de GetHomePage porte au plus HomeTilesHeadSize tuiles, la suite se
// lit par GetHomeMatches, et la concaténation des pages redonne la liste complète que servait
// l'ancienne réponse, tuile pour tuile.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// fixedHomeMatches sert des lignes canoniques fixes (copie à chaque lecture, comme le chargeur
// en cache) ; `rows` est réassignable entre deux lectures pour simuler une synchro.
type fixedHomeMatches struct{ rows []canonical.PlayerMatchRow }

func (f *fixedHomeMatches) LoadPlayerMatches(_ context.Context, _, _ string, _ port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error) {
	out := make([]canonical.PlayerMatchRow, len(f.rows))
	copy(out, f.rows)
	return out, nil
}
func (f *fixedHomeMatches) InvalidatePlayer(_, _ string) {}

// fixedFavorites ne répond qu'à GetFavoriteMatchIDs.
type fixedFavorites struct {
	port.SocialRepository
	ids map[string]bool
}

func (f fixedFavorites) GetFavoriteMatchIDs(context.Context, string) (map[string]bool, error) {
	return f.ids, nil
}

// homeTileRows : n matchs, un par 10 minutes, du plus récent au plus ancien (id m000 en tête),
// avec de quoi nourrir la construction d'une tuile (issue, frags, performance, session).
func homeTileRows(n int, start time.Time) []canonical.PlayerMatchRow {
	rows := make([]canonical.PlayerMatchRow, n)
	for i := range rows {
		k, d, a := 10+i%7, 5+i%4, i%5
		perf := float64(40 + i%50)
		ranked := i%3 == 0
		session := fmt.Sprintf("S%02d", i/6)
		outcome := []canonical.Outcome{canonical.OutcomeWin, canonical.OutcomeLoss, canonical.OutcomeTie}[i%3]
		rows[i] = canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{
				MatchID:      fmt.Sprintf("m%03d", i),
				StartedAtUTC: start.Add(-time.Duration(i) * 10 * time.Minute),
				IsRanked:     &ranked,
				Outcome:      outcome,
				Map:          &canonical.AssetReference{Kind: "map", ID: "map1", DefaultLabel: "Aquarius", Labels: map[string]string{"en": "Aquarius"}},
			},
			Self:       canonical.MatchParticipant{Kills: &k, Deaths: &d, Assists: &a, Outcome: outcome},
			Enrichment: canonical.PlayerMatchEnrichment{PerformanceScore: &perf, SessionLabel: &session},
		}
	}
	return rows
}

func newTilesService(slug string, rows []canonical.PlayerMatchRow, favorites map[string]bool) (*HomeService, *fixedHomeMatches) {
	pm := &fixedHomeMatches{rows: rows}
	svc := NewHomeService(&mockHomeRepo{}).WithPlayerMatchesRepo(pm, slug, "TestGT")
	if favorites != nil {
		svc = svc.WithSocial(fixedFavorites{ids: favorites}, "testgt")
	}
	return svc, pm
}

// readAllTiles lit la tête puis toutes les pages suivantes, comme le client.
func readAllTiles(t *testing.T, svc *HomeService, head *domain.HomePageResponse, pageSize int) []domain.RecentMatchItem {
	t.Helper()
	all := append([]domain.RecentMatchItem(nil), head.RecentMatches...)
	hasMore, cursor := head.RecentMatchesHasMore, head.RecentMatchesNextBefore
	for pages := 0; hasMore; pages++ {
		if pages > 400 {
			t.Fatal("pagination sans fin")
		}
		page, err := svc.GetHomeMatches(context.Background(), "fr", domain.HomeMatchesRequest{Before: cursor, Limit: pageSize})
		if err != nil {
			t.Fatalf("GetHomeMatches: %v", err)
		}
		all = append(all, page.Items...)
		hasMore, cursor = page.HasMore, page.NextBefore
	}
	return all
}

// TestGetHomePage_FenetreBornee est le garde-rail de l'invariant I8 (ADR 0036) : la tête de
// l'Accueil ne croît pas avec l'historique. Les totaux (héros, total des tuiles) restent ceux de
// l'historique entier. Rejoué sous deux slugs de titre : aucune comparaison de slug.
func TestGetHomePage_FenetreBornee(t *testing.T) {
	const history = 200
	for _, slug := range []string{"halo_infinite", "halo_5"} {
		t.Run(slug, func(t *testing.T) {
			svc, _ := newTilesService(slug, homeTileRows(history, time.Now()), nil)
			resp, err := svc.GetHomePage(context.Background(), "TestGT", "fr")
			if err != nil {
				t.Fatalf("GetHomePage: %v", err)
			}
			if got := len(resp.RecentMatches); got != domain.HomeTilesHeadSize {
				t.Errorf("tête = %d tuiles pour %d matchs, attendu %d", got, history, domain.HomeTilesHeadSize)
			}
			if resp.RecentMatchesTotal != history {
				t.Errorf("recent_matches_total = %d, attendu %d (compte de l'historique)", resp.RecentMatchesTotal, history)
			}
			if !resp.RecentMatchesHasMore || resp.RecentMatchesNextBefore == "" {
				t.Errorf("has_more=%v next_before=%q : la suite doit être annoncée", resp.RecentMatchesHasMore, resp.RecentMatchesNextBefore)
			}
			if resp.Hero.KPIs.TotalMatches != history {
				t.Errorf("total du héros = %d, attendu %d : les totaux se calculent sur tout l'historique", resp.Hero.KPIs.TotalMatches, history)
			}
		})
	}
}

// TestGetHomePage_PetitHistorique : sous la taille d'une tête, tout est servi et rien n'est annoncé.
func TestGetHomePage_PetitHistorique(t *testing.T) {
	svc, _ := newTilesService("halo_infinite", homeTileRows(12, time.Now()), nil)
	resp, err := svc.GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("GetHomePage: %v", err)
	}
	if len(resp.RecentMatches) != 12 || resp.RecentMatchesTotal != 12 || resp.RecentMatchesHasMore || resp.RecentMatchesNextBefore != "" {
		t.Errorf("tuiles=%d total=%d has_more=%v next=%q", len(resp.RecentMatches), resp.RecentMatchesTotal, resp.RecentMatchesHasMore, resp.RecentMatchesNextBefore)
	}
	empty, _ := newTilesService("halo_infinite", nil, nil)
	resp, err = empty.GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("GetHomePage vide: %v", err)
	}
	if resp.RecentMatches == nil || resp.LastSessionOutcomes == nil || resp.RecentMatchesTotal != 0 {
		t.Errorf("historique vide : tranches non nulles attendues, obtenu %+v", resp)
	}
}

// TestGetHomeMatches_ParitePagesEtListeComplete : la concaténation de la tête et des pages donne,
// champ pour champ et dans le même ordre, la liste que l'ancienne réponse servait (toutes les
// tuiles construites d'un bloc par le même constructeur), quelle que soit la taille de page.
func TestGetHomeMatches_ParitePagesEtListeComplete(t *testing.T) {
	rows := homeTileRows(200, time.Now())
	favorites := map[string]bool{"m003": true, "m090": true, "m199": true}
	svc, _ := newTilesService("halo_infinite", rows, favorites)

	// Référence = l'ancien chemin : TOUTES les tuiles, une seule construction.
	ordered := analysis.OrderHomeTileRows(rows)
	want := svc.buildTiles(context.Background(), tilesInput{
		allRows: rows, window: ordered, favoriteIDs: favorites, locale: "fr", list: homeSectionListRecent,
	})
	if len(want) != 200 {
		t.Fatalf("référence = %d tuiles, attendu 200", len(want))
	}
	head, err := svc.GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("GetHomePage: %v", err)
	}
	for _, size := range []int{0, 1, 17, 30, 60, 1000} {
		got := readAllTiles(t, svc, head, size)
		if len(got) != len(want) {
			t.Fatalf("pages de %d : %d tuiles, attendu %d", size, len(got), len(want))
		}
		for i := range want {
			if !reflect.DeepEqual(got[i], want[i]) {
				t.Fatalf("pages de %d : la tuile %d (%s) diffère de la liste complète\n got: %+v\nwant: %+v", size, i, want[i].MatchID, got[i], want[i])
			}
		}
	}
	// Le même ordre que l'ancien chargeur (instant décroissant) : la liste n'a pas été permutée.
	for i := range want {
		if want[i].MatchID != rows[i].Summary.MatchID {
			t.Fatalf("tuile %d = %s, attendu %s (ordre du chargeur)", i, want[i].MatchID, rows[i].Summary.MatchID)
		}
	}
	for _, it := range want {
		if it.IsFavorite != favorites[it.MatchID] {
			t.Errorf("%s : is_favorite=%v, attendu %v", it.MatchID, it.IsFavorite, favorites[it.MatchID])
		}
	}
}

// TestGetHomeMatches_SynchroEntreDeuxPages : une synchro entre deux pages ajoute des matchs en
// tête ; la page suivante reprend à la tuile désignée par le curseur, sans doublon ni trou.
func TestGetHomeMatches_SynchroEntreDeuxPages(t *testing.T) {
	start := time.Now()
	svc, pm := newTilesService("halo_infinite", homeTileRows(100, start), nil)
	head, err := svc.GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("GetHomePage: %v", err)
	}
	// Trois matchs plus récents arrivent entre les deux requêtes.
	fresh := homeTileRows(3, start.Add(time.Hour))
	for i := range fresh {
		fresh[i].Summary.MatchID = fmt.Sprintf("n%d", i)
	}
	pm.rows = append(fresh, pm.rows...)

	page, err := svc.GetHomeMatches(context.Background(), "fr", domain.HomeMatchesRequest{Before: head.RecentMatchesNextBefore, Limit: 30})
	if err != nil {
		t.Fatalf("GetHomeMatches: %v", err)
	}
	seen := map[string]bool{}
	for _, it := range head.RecentMatches {
		seen[it.MatchID] = true
	}
	for i, it := range page.Items {
		if seen[it.MatchID] {
			t.Errorf("doublon %s dans la page suivante", it.MatchID)
		}
		if want := fmt.Sprintf("m%03d", 30+i); it.MatchID != want {
			t.Fatalf("tuile %d de la page suivante = %s, attendu %s (aucun trou)", i, it.MatchID, want)
		}
	}
}

// TestGetHomeMatches_Favoris : les favoris sont servis en entier, dans l'ordre d'affichage, y
// compris ceux qui sont hors de la tête ; ni curseur ni limite ne s'y appliquent.
func TestGetHomeMatches_Favoris(t *testing.T) {
	favorites := map[string]bool{"m150": true, "m002": true, "m077": true}
	svc, _ := newTilesService("halo_5", homeTileRows(200, time.Now()), favorites)
	page, err := svc.GetHomeMatches(context.Background(), "fr", domain.HomeMatchesRequest{
		Kind: domain.HomeMatchesKindFavorites, Before: "ignoré", Limit: 1,
	})
	if err != nil {
		t.Fatalf("GetHomeMatches favoris: %v", err)
	}
	var ids []string
	for _, it := range page.Items {
		ids = append(ids, it.MatchID)
		if !it.IsFavorite {
			t.Errorf("%s : is_favorite attendu", it.MatchID)
		}
	}
	if fmt.Sprint(ids) != "[m002 m077 m150]" || page.HasMore || page.NextBefore != "" {
		t.Errorf("favoris = %v has_more=%v next=%q", ids, page.HasMore, page.NextBefore)
	}
	none, _ := newTilesService("halo_5", homeTileRows(10, time.Now()), nil)
	page, err = none.GetHomeMatches(context.Background(), "fr", domain.HomeMatchesRequest{Kind: domain.HomeMatchesKindFavorites})
	if err != nil || page.Items == nil || len(page.Items) != 0 {
		t.Errorf("sans favori : tranche vide non nulle attendue, obtenu (%+v, %v)", page, err)
	}
}

// TestGetHomeMatches_LimiteEtCurseur : taille par défaut, borne haute, curseur illisible refusé.
func TestGetHomeMatches_LimiteEtCurseur(t *testing.T) {
	svc, _ := newTilesService("halo_infinite", homeTileRows(300, time.Now()), nil)
	for _, c := range []struct{ limit, want int }{
		{0, domain.HomeTilesHeadSize}, {-4, domain.HomeTilesHeadSize}, {10, 10}, {5000, domain.HomeTilesMaxPageSize},
	} {
		page, err := svc.GetHomeMatches(context.Background(), "fr", domain.HomeMatchesRequest{Limit: c.limit})
		if err != nil || len(page.Items) != c.want {
			t.Errorf("limit=%d : %d tuiles (err %v), attendu %d", c.limit, len(page.Items), err, c.want)
		}
	}
	if _, err := svc.GetHomeMatches(context.Background(), "fr", domain.HomeMatchesRequest{Before: "pas un curseur"}); !errors.Is(err, domain.ErrHomeCursorInvalid) {
		t.Errorf("curseur invalide : err = %v, attendu ErrHomeCursorInvalid", err)
	}
}

// TestGetHomePage_SequenceDeLaDerniereSession : last_session_outcomes porte les issues de la
// dernière session résumée, calculées sur l'historique entier (la tête fenêtrée ne la porterait
// pas toujours), dans l'ordre d'affichage, avec le libellé de la tuile.
func TestGetHomePage_SequenceDeLaDerniereSession(t *testing.T) {
	rows := homeTileRows(200, time.Now())
	svc, _ := newTilesService("halo_infinite", rows, nil)
	resp, err := svc.GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("GetHomePage: %v", err)
	}
	if resp.SoloSession == nil {
		t.Fatal("session solo attendue")
	}
	label := resp.SoloSession.SessionLabel
	var want []string
	for i := range rows {
		if sessionLabelOf(&rows[i]) == label {
			want = append(want, rows[i].Summary.MatchID)
		}
	}
	var got []string
	for _, o := range resp.LastSessionOutcomes {
		got = append(got, o.MatchID)
		if o.OutcomeTone == "" || o.Detail == "" {
			t.Errorf("%s : issue incomplète %+v", o.MatchID, o)
		}
	}
	if len(want) == 0 || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("séquence = %v, attendu %v (session %s)", got, want, label)
	}
	tiles := map[string]domain.RecentMatchItem{}
	for _, it := range resp.RecentMatches {
		tiles[it.MatchID] = it
	}
	for _, o := range resp.LastSessionOutcomes {
		if it, ok := tiles[o.MatchID]; ok && (it.OutcomeTone != o.OutcomeTone || it.Detail != o.Detail) {
			t.Errorf("%s : l'issue (%s, %q) diffère de la tuile (%s, %q)", o.MatchID, o.OutcomeTone, o.Detail, it.OutcomeTone, it.Detail)
		}
	}
}

// TestLastSessionOutcomes_RepliSansSessionConnue : aucune tuile ne porte un libellé de session
// résumé -> les 20 plus récentes.
func TestLastSessionOutcomes_RepliSansSessionConnue(t *testing.T) {
	ordered := analysis.OrderHomeTileRows(homeTileRows(50, time.Now()))
	got := lastSessionOutcomes(ordered, map[string]bool{"inconnue": true}, analysis.RecentMatchesOptions{Locale: "fr"})
	if len(got) != lastSessionFallbackTiles || got[0].MatchID != "m000" {
		t.Errorf("repli = %d issues dont la première %v, attendu %d depuis m000", len(got), got[0], lastSessionFallbackTiles)
	}
}

// TestGetHomePage_SansFavorisDansLaTete : les favoris ne sont plus servis avec la tête (ils ont
// leur propre lecture) : la réponse ne porte aucune clé favorite_matches.
func TestGetHomePage_SansFavorisDansLaTete(t *testing.T) {
	svc, _ := newTilesService("halo_infinite", homeTileRows(40, time.Now()), map[string]bool{"m001": true, "m035": true})
	resp, err := svc.GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("GetHomePage: %v", err)
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := keys["favorite_matches"]; ok {
		t.Error("la tête ne doit plus porter favorite_matches")
	}
	for _, it := range resp.RecentMatches {
		if it.MatchID == "m001" && !it.IsFavorite {
			t.Error("m001 est un favori : sa tuile de la tête doit porter is_favorite")
		}
	}
}
