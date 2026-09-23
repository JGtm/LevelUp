package teammates

// teammates_service_composition_legere_test.go — la lecture légère des sessions d'une
// composition (CompositionSessions, lot perf L4b, 2026-09-23) :
//
//   - PARITÉ avec GetPage : pour la même composition et la même option, la même liste de
//     sessions (même ordre, mêmes match_count, match_count_roster et
//     excluded_by_exact_composition) et la même dernière session — sur les scénarios déjà
//     posés par le paquet (composition exacte, écart nommé, départ en cours de partie,
//     coéquipier hors top ou introuvable, lectures en échec, sans coéquipier) — hors l'équipe
//     alliée illisible sous l'option, une ERREUR ici (lot L9-go), une dégradation dite sur la
//     page ;
//   - ce qu'elle NE lit PAS : aucune lecture d'une section de la page, pas d'historique de
//     membre, pas d'équipe alliée hors option ;
//   - annulation et sections de durée.

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// casDeParite : un scénario existant, une composition, une option. `page` complète la
// requête de GetPage (filtres de page : ils ne changent pas les sessions de la composition).
type casDeParite struct {
	nom   string
	repo  func() *mockSquadRepo
	amis  []string
	gts   []string
	exact bool
	page  func(*domain.TeammatesQueryRequest)
	// attendu : les labels publiés, dans l'ordre — le cas n'est pas vide de sens.
	attendu []string
}

// avecAliasHorsTop : AllyB n'est plus dans le top 50, il se résout par les alias.
func avecAliasHorsTop() *mockSquadRepo {
	repo := newExtraTeammateRepo()
	repo.topRows = slices.DeleteFunc(slices.Clone(repo.topRows), func(r domain.TopTeammateRow) bool {
		return r.Gamertag == "AllyB"
	})
	repo.lookupAliases = map[string]string{"allyb": "xb"}
	return repo
}

// sansEquipeConnueSurM2 : le scénario de l'écart, sans aucune ligne d'allié pour m2.
func sansEquipeConnueSurM2() *mockSquadRepo {
	repo := newExactCompositionGapRepo()
	repo.allyRows = slices.DeleteFunc(slices.Clone(repo.allyRows), func(a domain.AllyParticipant) bool {
		return a.MatchID == "m2"
	})
	return repo
}

// historiqueDuPrincipal : sans coéquipier, deux sessions escouade et une session solo, à
// horodatages FIXES (les deux chemins lisent le même historique).
func historiqueDuPrincipal() *mockSquadRepo {
	t0 := time.Date(2026, 9, 1, 20, 0, 0, 0, time.UTC)
	row := func(id, session string, ts time.Time, avecAmis bool) legacymatch.SynthesisMatchRow {
		return legacymatch.SynthesisMatchRow{
			MatchID: id, StartTime: ts, IsWithFriends: avecAmis, SessionLabel: strPtr(session),
			PlaylistName: "Arene classee",
		}
	}
	return &mockSquadRepo{synthRows: []legacymatch.SynthesisMatchRow{
		row("h1", "S_ancienne (2)", t0, true),
		row("h2", "S_ancienne (2)", t0.Add(20*time.Minute), true),
		row("h3", "S_solo (1)", t0.Add(24*time.Hour), false),
		row("h4", "S_recente (1)", t0.Add(48*time.Hour), true),
	}}
}

func casDeLaParite() []casDeParite {
	echecQ30 := func() *mockSquadRepo {
		repo := newExtraTeammateRepo()
		repo.squadErr = errors.New("q30 en echec")
		return repo
	}
	// Q32b en échec sous l'option n'est plus un cas de parité (lot perf L9-go) : la page
	// dégrade et le dit, la lecture légère rend une erreur —
	// TestCompositionSessions_EquipeAllieeIllisible_Erreur.
	debut := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)
	filtresDePage := func(req *domain.TeammatesQueryRequest) {
		req.PickedSquadSessions = []string{"S_exact"}
		req.Filters = &domain.FilterContextInput{
			Sessions: domain.SessionsFilter{PickedSessions: []string{"S_exact"}},
			Period:   domain.PeriodInput{StartDate: &debut},
		}
	}
	both := []string{"AllyA", "AllyB"}
	return []casDeParite{
		{nom: "roster, option off", repo: newExtraTeammateRepo, gts: both, attendu: []string{"S_with_C", "S_exact"}},
		{nom: "composition exacte : AllyC ecarte m2", repo: newExtraTeammateRepo, gts: both, exact: true, attendu: []string{"S_exact"}},
		{nom: "amis du joueur dans l extraPool", repo: newExtraTeammateRepo, amis: []string{"AllyC"}, gts: both, exact: true, attendu: []string{"S_exact"}},
		{nom: "ecart publie et nomme", repo: newExactCompositionGapRepo, gts: both, exact: true, attendu: []string{"S1"}},
		{nom: "ecart, option off", repo: newExactCompositionGapRepo, gts: both, attendu: []string{"S1"}},
		{nom: "match sans equipe connue", repo: sansEquipeConnueSurM2, gts: both, exact: true, attendu: []string{"S1"}},
		{nom: "depart en cours de partie, option on", repo: newCrashedTeammateRepo, gts: both, exact: true, attendu: []string{"S_27_08"}},
		{nom: "depart en cours de partie, option off", repo: newCrashedTeammateRepo, gts: both, attendu: []string{"S_27_08"}},
		{nom: "gamertags en casse differente", repo: newExtraTeammateRepo, gts: []string{"allya", "ALLYB"}, exact: true, attendu: []string{"S_exact"}},
		{nom: "coequipier hors top 50 (alias)", repo: avecAliasHorsTop, gts: both, exact: true, attendu: []string{"S_exact"}},
		{nom: "coequipier introuvable", repo: newExtraTeammateRepo, gts: []string{"AllyA", "Inconnu"}, attendu: []string{"S_with_C", "S_exact"}},
		{nom: "Q30 en echec : personne dans l intersection", repo: echecQ30, gts: both, exact: true, attendu: nil},
		{nom: "filtres de page sans effet sur les sessions", repo: newExtraTeammateRepo, gts: both, page: filtresDePage, attendu: []string{"S_with_C", "S_exact"}},
		{nom: "sans coequipier : sessions escouade du principal", repo: historiqueDuPrincipal, attendu: []string{"S_recente (1)", "S_ancienne (2)"}},
		{nom: "sans coequipier ni match", repo: func() *mockSquadRepo { return &mockSquadRepo{} }, attendu: nil},
	}
}

// serviceDe : le service des deux chemins d'un cas (même dépôt, même historique).
func serviceDe(repo *mockSquadRepo, amis []string) *TeammatesService {
	var resolver FriendGamertagsResolver
	if amis != nil {
		resolver = func(context.Context) []string { return amis }
	}
	return NewTeammatesService(repo, resolver).WithPlayerMatchesRepo(
		newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test",
	)
}

func labelsDe(sessions []domain.CompositionSessionEntry) []string {
	var out []string
	for _, s := range sessions {
		out = append(out, s.Label)
	}
	return out
}

func TestCompositionSessions_PariteAvecGetPage(t *testing.T) {
	for _, c := range casDeLaParite() {
		t.Run(c.nom, func(t *testing.T) {
			svc := serviceDe(c.repo(), c.amis)
			req := domain.TeammatesQueryRequest{SelectedGamertags: c.gts, FilterExactComposition: c.exact}
			if c.page != nil {
				c.page(&req)
			}
			resp, err := svc.GetPage(context.Background(), "px", req)
			if err != nil {
				t.Fatalf("GetPage : %v", err)
			}
			sessions, latest, err := svc.CompositionSessions(context.Background(), "px", c.gts, c.exact)
			if err != nil {
				t.Fatalf("CompositionSessions : %v", err)
			}

			if got := labelsDe(sessions); !slices.Equal(got, c.attendu) {
				t.Fatalf("sessions légères : %v, attendu %v", got, c.attendu)
			}
			if len(resp.CompositionSessions) != len(sessions) ||
				(len(sessions) > 0 && !reflect.DeepEqual(resp.CompositionSessions, sessions)) {
				t.Errorf("composition_sessions différentes :\n page   %+v\n légère %+v", resp.CompositionSessions, sessions)
			}
			if resp.LatestCompositionSession != latest {
				t.Errorf("latest_composition_session : page %q, légère %q", resp.LatestCompositionSession, latest)
			}
		})
	}
}

// TestCompositionSessions_EcartPublieCommeLaPage : le cas le plus riche, relu champ par
// champ (la parité ci-dessus le compare en bloc) — 1 match gardé sur 5, quatre écartés et
// leurs responsables nommés.
func TestCompositionSessions_EcartPublieCommeLaPage(t *testing.T) {
	svc := serviceDe(newExactCompositionGapRepo(), nil)
	sessions, latest, err := svc.CompositionSessions(context.Background(), "px", []string{"AllyA", "AllyB"}, true)
	if err != nil {
		t.Fatalf("CompositionSessions : %v", err)
	}
	if latest != "S1" || len(sessions) != 1 {
		t.Fatalf("sessions : %v (dernière %q), attendu [S1]", labelsDe(sessions), latest)
	}
	s1 := sessions[0]
	if s1.MatchCount != 1 || s1.MatchCountRoster != 5 || len(s1.ExcludedByExactComposition) != 4 {
		t.Fatalf("comptes : %d sur %d, %d écartés — attendu 1 sur 5, 4 écartés",
			s1.MatchCount, s1.MatchCountRoster, len(s1.ExcludedByExactComposition))
	}
	responsables := map[string][]string{}
	for _, ex := range s1.ExcludedByExactComposition {
		responsables[ex.MatchID] = ex.ExtraGamertags
	}
	if got := responsables["m5"]; !slices.Equal(got, []string{"Nilton410", "passivemarquise"}) {
		t.Errorf("m5 : responsables %v", got)
	}
	if got := responsables["m4"]; !slices.Equal(got, []string{"Joueur 0009"}) {
		t.Errorf("m4 : responsable %v (repli attendu « Joueur 0009 »)", got)
	}
}

// lecturesComptees : le dépôt Escouade dont chaque lecture se note — l'oracle de « la
// lecture légère ne fait QUE les lectures dont les sessions dépendent ».
type lecturesComptees struct {
	*mockSquadRepo
	lues []string
}

func (l *lecturesComptees) LoadTopTeammates(ctx context.Context, xuid string) ([]domain.TopTeammateRow, error) {
	l.lues = append(l.lues, "Q29")
	return l.mockSquadRepo.LoadTopTeammates(ctx, xuid)
}

func (l *lecturesComptees) LookupXUIDByGamertag(ctx context.Context, gt string) (string, bool, error) {
	l.lues = append(l.lues, "alias")
	return l.mockSquadRepo.LookupXUIDByGamertag(ctx, gt)
}

func (l *lecturesComptees) LoadSquadMatches(ctx context.Context, p, x string) ([]domain.SquadMatchRow, error) {
	l.lues = append(l.lues, "Q30")
	return l.mockSquadRepo.LoadSquadMatches(ctx, p, x)
}

func (l *lecturesComptees) LoadMainTeamParticipants(ctx context.Context, x string, ids []string) ([]domain.AllyParticipant, error) {
	l.lues = append(l.lues, "Q32b")
	return l.mockSquadRepo.LoadMainTeamParticipants(ctx, x, ids)
}

// Les lectures des SECTIONS de la page : aucune n'est permise à la lecture légère.

func (l *lecturesComptees) LoadTeammateMatches(ctx context.Context, p, x string) ([]domain.TeammateMatchRow, error) {
	l.lues = append(l.lues, "section:Q31")
	return l.mockSquadRepo.LoadTeammateMatches(ctx, p, x)
}

func (l *lecturesComptees) LoadImpactEvents(ctx context.Context, ids []string) ([]domain.ImpactEventRow, error) {
	l.lues = append(l.lues, "section:Q32")
	return l.mockSquadRepo.LoadImpactEvents(ctx, ids)
}

func (l *lecturesComptees) LoadKVPairs(ctx context.Context, ids []string) ([]domain.KVPairRaw, error) {
	l.lues = append(l.lues, "section:kv")
	return l.mockSquadRepo.LoadKVPairs(ctx, ids)
}

func (l *lecturesComptees) LoadSquadAssistPairs(ctx context.Context, ids, x []string) ([]domain.SquadAssistPairRaw, int, error) {
	l.lues = append(l.lues, "section:assists")
	return l.mockSquadRepo.LoadSquadAssistPairs(ctx, ids, x)
}

func (l *lecturesComptees) LoadSquadKillLog(ctx context.Context, ids, x []string) ([]domain.SquadKillLogRow, error) {
	l.lues = append(l.lues, "section:killlog")
	return l.mockSquadRepo.LoadSquadKillLog(ctx, ids, x)
}

func (l *lecturesComptees) LoadSynthesisHeatmap(ctx context.Context, x string) ([]domain.SynthesisHeatmapRow, error) {
	l.lues = append(l.lues, "section:heatmap")
	return l.mockSquadRepo.LoadSynthesisHeatmap(ctx, x)
}

func (l *lecturesComptees) LoadAssetTranslationsFR(ctx context.Context, kind string, ids []string) (map[string]string, error) {
	l.lues = append(l.lues, "section:assets")
	return l.mockSquadRepo.LoadAssetTranslationsFR(ctx, kind, ids)
}

func (l *lecturesComptees) LoadModeTranslationsFR(ctx context.Context, modes []string) (map[string]string, error) {
	l.lues = append(l.lues, "section:modes")
	return l.mockSquadRepo.LoadModeTranslationsFR(ctx, modes)
}

func (l *lecturesComptees) LoadMapStatsForSquad(ctx context.Context, x string, squad, excl []string) (map[string]domain.MapSquadStats, error) {
	l.lues = append(l.lues, "section:mapstats")
	return l.mockSquadRepo.LoadMapStatsForSquad(ctx, x, squad, excl)
}

// historiqueCompte : l'historique du joueur principal, lectures comptées.
type historiqueCompte struct {
	*mockSynthPlayerMatches
	lectures int
}

func (h *historiqueCompte) LoadPlayerMatches(
	ctx context.Context, slug, gt string, f port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	h.lectures++
	return h.mockSynthPlayerMatches.LoadPlayerMatches(ctx, slug, gt, f)
}

// TestCompositionSessions_NeLitQueSesDependances : Q29 puis Q30 par coéquipier, Q32b sous
// l'option seulement, l'alias pour un coéquipier hors top ; jamais une lecture de section,
// jamais l'historique d'un membre (LoadFor), jamais l'historique du principal quand une
// composition est désignée — et, sans coéquipier, cet historique seul.
func TestCompositionSessions_NeLitQueSesDependances(t *testing.T) {
	cas := []struct {
		nom        string
		repo       func() *mockSquadRepo
		gts        []string
		exact      bool
		lues       []string
		historique int
	}{
		{"option off", newExtraTeammateRepo, []string{"AllyA", "AllyB"}, false, []string{"Q29", "Q30", "Q30"}, 0},
		{"option on", newExtraTeammateRepo, []string{"AllyA", "AllyB"}, true, []string{"Q29", "Q30", "Q30", "Q32b"}, 0},
		{"coequipier hors top", avecAliasHorsTop, []string{"AllyA", "AllyB"}, false, []string{"Q29", "Q30", "alias", "Q30"}, 0},
		{"sans coequipier", historiqueDuPrincipal, nil, true, nil, 1},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			base := c.repo()
			repo := &lecturesComptees{mockSquadRepo: base}
			historique := &historiqueCompte{mockSynthPlayerMatches: newSynthMockFromRows(base.synthRows, nil)}
			membres := &fakeSquadLoader{}
			svc := NewTeammatesService(repo, nil).
				WithPlayerMatchesRepo(historique, "halo_infinite", "Test").
				WithSquadLoader(membres)

			if _, _, err := svc.CompositionSessions(context.Background(), "px", c.gts, c.exact); err != nil {
				t.Fatalf("CompositionSessions : %v", err)
			}
			if !slices.Equal(repo.lues, c.lues) {
				t.Errorf("lectures Escouade : %v, attendu %v", repo.lues, c.lues)
			}
			if historique.lectures != c.historique {
				t.Errorf("historique du principal lu %d fois, attendu %d", historique.lectures, c.historique)
			}
			if len(membres.calls) != 0 {
				t.Errorf("historiques de membres lus (LoadFor) : %v, attendu aucun", membres.calls)
			}
		})
	}
}

// annuleApresQ30 : la première lecture Q30 annule la requête (client parti pendant la lecture).
type annuleApresQ30 struct {
	*mockSquadRepo
	annuler context.CancelFunc
	q30     int
}

func (a *annuleApresQ30) LoadSquadMatches(ctx context.Context, p, x string) ([]domain.SquadMatchRow, error) {
	a.q30++
	a.annuler()
	return a.mockSquadRepo.LoadSquadMatches(ctx, p, x)
}

// TestCompositionSessions_RequeteAnnulee : une requête annulée entre deux coéquipiers rend
// l'erreur du contexte (le handler en fait un 499), jamais des sessions partielles, et ne lit
// pas le coéquipier suivant.
func TestCompositionSessions_RequeteAnnulee(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := &annuleApresQ30{mockSquadRepo: newExtraTeammateRepo(), annuler: cancel}
	svc := NewTeammatesService(repo, nil)

	sessions, latest, err := svc.CompositionSessions(ctx, "px", []string{"AllyA", "AllyB"}, true)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("erreur : %v, attendu context.Canceled", err)
	}
	if sessions != nil || latest != "" {
		t.Errorf("résultat partiel rendu : %v %q", labelsDe(sessions), latest)
	}
	if repo.q30 != 1 {
		t.Errorf("Q30 lue %d fois, attendu 1 (le coéquipier suivant n'est pas lu)", repo.q30)
	}
}

// TestCompositionSessions_SectionsDeDuree : chaque chargement a sa section (lot L1), et le
// calcul des sessions la sienne — les noms de GetPage quand la lecture est la même.
func TestCompositionSessions_SectionsDeDuree(t *testing.T) {
	cas := []struct {
		nom      string
		repo     func() *mockSquadRepo
		gts      []string
		exact    bool
		sections []string
	}{
		{"composition exacte", newExtraTeammateRepo, []string{"AllyA", "AllyB"}, true,
			[]string{"composition_sessions", "main_team_allies", "squad_matches", "top_teammates"}},
		{"sans coequipier", historiqueDuPrincipal, nil, false,
			[]string{"composition_sessions", "player_matches"}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			ctx, tm := timing.WithTimings(context.Background())
			if _, _, err := serviceDe(c.repo(), nil).CompositionSessions(ctx, "px", c.gts, c.exact); err != nil {
				t.Fatalf("CompositionSessions : %v", err)
			}
			var noms []string
			for _, s := range tm.Snapshot() {
				noms = append(noms, s.Name)
			}
			slices.Sort(noms)
			if !slices.Equal(noms, c.sections) {
				t.Errorf("sections : %v, attendu %v", noms, c.sections)
			}
		})
	}
}

// q32bEnEchecAuPremierAppel : l'équipe alliée (Q32b) illisible au PREMIER appel seulement —
// une base occupée le temps d'une bascule, le scénario de la revue adversariale A.
type q32bEnEchecAuPremierAppel struct {
	*mockSquadRepo
	err    error
	appels int
}

func (r *q32bEnEchecAuPremierAppel) LoadMainTeamParticipants(ctx context.Context, main string, ids []string) ([]domain.AllyParticipant, error) {
	r.appels++
	if r.appels == 1 {
		return nil, r.err
	}
	return r.mockSquadRepo.LoadMainTeamParticipants(ctx, main, ids)
}

// TestCompositionSessions_EquipeAllieeIllisible_Erreur (lot perf L9-go, revue adversariale A :
// TestRevA_LegereDegradeeSansSignal inversé) : sous l'option composition exacte, une équipe
// alliée illisible est une ERREUR qui garde sa cause — le handler en fait un 503 quand la base
// est occupée, un 500 sinon, et le front se replie sur la page. Jamais un 200 aux sessions non
// filtrées : le front s'y ancrait (« S_with_C ») alors que la page, elle, publie « S_exact ».
// Hors option, Q32b n'est pas lue : aucune erreur.
func TestCompositionSessions_EquipeAllieeIllisible_Erreur(t *testing.T) {
	baseOccupee := errors.New("database is locked")
	gts := []string{"AllyA", "AllyB"}
	repo := &q32bEnEchecAuPremierAppel{mockSquadRepo: newExtraTeammateRepo(), err: baseOccupee}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	sessions, latest, err := svc.CompositionSessions(context.Background(), "px", gts, true)
	if !errors.Is(err, baseOccupee) {
		t.Fatalf("lecture légère, Q32b illisible : err=%v, want une erreur enveloppant la cause", err)
	}
	if sessions != nil || latest != "" {
		t.Errorf("lecture légère en erreur : sessions %v, dernière %q rendues", labelsDe(sessions), latest)
	}
	// Le repli du front : la page, Q32b lisible cette fois, filtre la composition.
	page, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags: gts, FilterExactComposition: true})
	if err != nil || page.LatestCompositionSession != "S_exact" {
		t.Errorf("page de repli : err=%v, dernière session %q, want S_exact", err, page.LatestCompositionSession)
	}

	horsOption := newExtraTeammateRepo()
	horsOption.allyErr = baseOccupee
	sessions, _, err = serviceDe(horsOption, nil).CompositionSessions(context.Background(), "px", gts, false)
	if err != nil || !slices.Equal(labelsDe(sessions), []string{"S_with_C", "S_exact"}) {
		t.Errorf("hors option (Q32b non lue) : err=%v, sessions %v", err, labelsDe(sessions))
	}
}
