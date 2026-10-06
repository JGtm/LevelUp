package service

// session_page_blocks_test.go — LES BLOCS DU FILM DE LA COLONNE DE SESSION (plan
// `.ai/PLAN_SESSIONS_EMPRISE_2026-10-06.md`, lot S2) : une lecture du résumé d'usage par session
// partagée par tous les blocs, la coordination qui garde son joueur et son effectif de camp,
// l'Emprise d'un seul joueur, les vies et l'objectif de chaque session, l'emblème.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability/timing"
)

// serviceDeBlocs — la page Sessions avec la feuille de match et, si fourni, le résumé d'usage.
func serviceDeBlocs(usage *mockSessionUsageRepo) *SessionPageService {
	svc := NewSessionPageService(nil).
		WithPlayerMatchesRepo(nil, "halo_infinite", "Papa").
		WithSessionEmprise(feuilleDeTest(), "P")
	if usage != nil {
		svc = svc.WithSessionUsageSummary(usage, "")
	}
	return svc
}

// sessionsDeTest — la session affichée (m1, m2) et, si demandé, la session comparée (m3).
func sessionsDeTest(compare bool) (sessionBlocksScope, []canonical.PlayerMatchRow) {
	sc := sessionBlocksScope{Matches: matchsDeTest("m1", "m2"), Locale: "fr"}
	canon := fenetreDeTest()
	if compare {
		sc.CompareMatches = matchsDeTest("m3")
		canon = append(canon, ligneCanon("m3", "Aquarius", canonical.OutcomeWin, 30))
	}
	return sc, canon
}

// Le résumé d'usage d'une session est lu UNE fois pour tous les blocs qui le lisent (ADR 0036
// I4) : une lecture pour la session affichée, une pour la session comparée.
func TestAttachSessionBlocks_UneLectureDuResumeDUsageParSession(t *testing.T) {
	for _, cas := range []struct {
		compare bool
		want    int
	}{{false, 1}, {true, 2}} {
		usage := &usageCompte{mockSessionUsageRepo: usageTestRepoMock()}
		svc := NewSessionPageService(nil).
			WithPlayerMatchesRepo(nil, "halo_infinite", "Papa").
			WithSessionEmprise(feuilleDeTest(), "P").
			WithSessionUsageSummary(usage, "")
		sc, canon := sessionsDeTest(cas.compare)
		var resp domain.SessionPageResponse
		svc.attachSessionBlocks(context.Background(), &resp, sc, canon)
		if usage.films != cas.want {
			t.Errorf("comparaison %v : %d lecture(s) des films, attendu %d (une par session)", cas.compare, usage.films, cas.want)
		}
		if resp.Emprise == nil || resp.FormesRetenues == nil {
			t.Errorf("comparaison %v : emprise %v, formes %v — la lecture partagée doit les nourrir tous",
				cas.compare, resp.Emprise != nil, resp.FormesRetenues != nil)
		}
	}
}

// D9 — LA COORDINATION APRÈS LE RECÂBLAGE : même joueur, même effectif de camp, donc le MÊME
// bloc que l'ancien chemin (effectifs du bloc d'usage). L'oracle est l'appel direct avec les
// effectifs que l'ancien bloc d'usage calculait sur les mêmes participants.
func TestAttachSessionBlocks_AppuiIdentiqueAuCheminDuBlocDUsage(t *testing.T) {
	ctx := context.Background()
	sc := sessionBlocksScope{Matches: matchsDeTest("m1")}
	appuis := func() *appuisRepoStub {
		return &appuisRepoStub{rows: []domain.CoordinationAppuiRow{
			{MatchID: "m1", AssistXUID: "A", KillerXUID: "P", Nombre: 1},
			{MatchID: "m1", AssistXUID: "", KillerXUID: "P", Nombre: 1},
		}}
	}
	caps := games.CapabilityMap{games.CapFilmKillSource: games.CapSupported}
	var resp domain.SessionPageResponse
	serviceDeBlocs(usageTestRepoMock()).WithSessionCoordination(&tacticalRepoParScope{}, appuis(), caps).
		attachSessionBlocks(ctx, &resp, sc, fenetreDeTest()[:1])

	var want domain.SessionPageResponse
	effectifs := sessionusage.BuildTeamContext("P", usageTestRepoMock().participants).TeamSize
	serviceDeCoordination(&tacticalRepoParScope{}).attachSessionCoordination(ctx, &want, sc, effectifs, nil)

	if !reflect.DeepEqual(resp.Coordination, want.Coordination) {
		t.Errorf("coordination :\n got %+v\nwant %+v", resp.Coordination, want.Coordination)
	}
	if resp.Coordination == nil || resp.Coordination.Appui.ParityPct == nil || *resp.Coordination.Appui.ParityPct != 50 {
		t.Errorf("parité de l'appui = %+v, attendu 50 (deux joueurs dans mon camp)", resp.Coordination)
	}
}

// D9 — sans résumé d'usage (titre qui nomme ses tueurs sans publier d'usage), la coordination
// garde son joueur : bloc servi, sans parité (aucun effectif de camp inventé).
func TestAttachSessionBlocks_CoordinationSansResumeDUsage(t *testing.T) {
	var resp domain.SessionPageResponse
	NewSessionPageService(nil).
		WithSessionEmprise(nil, "P").
		WithSessionCoordination(&tacticalRepoParScope{}, &appuisRepoStub{rows: []domain.CoordinationAppuiRow{
			{MatchID: "m1", AssistXUID: "A", KillerXUID: "P", Nombre: 1},
		}}, games.CapabilityMap{games.CapFilmKillSource: games.CapSupported}).
		attachSessionBlocks(context.Background(), &resp, sessionBlocksScope{Matches: matchsDeTest("m1")}, nil)
	if resp.Coordination == nil || !resp.Coordination.Available {
		t.Fatalf("coordination = %+v, attendu servie (le joueur vient de WithSessionEmprise)", resp.Coordination)
	}
	if resp.Coordination.Appui.ParityPct != nil {
		t.Errorf("parité = %v, attendu absente sans effectif de camp", *resp.Coordination.Appui.ParityPct)
	}
}

// V1 — un seul joueur des fiches, même quand un coéquipier est dans les participants ; ni
// habitude, ni placement, ni grille par carte ; la session comparée a son propre bloc.
func TestAttachSessionBlocks_EmpriseDUnSeulJoueurParSession(t *testing.T) {
	sc, canon := sessionsDeTest(true)
	var resp domain.SessionPageResponse
	serviceDeBlocs(usageTestRepoMock()).attachSessionBlocks(context.Background(), &resp, sc, canon)
	e := resp.Emprise
	if e == nil || e.MatchesTotal != 2 || e.FilmUnavailable != "" || e.SheetUnavailable != "" {
		t.Fatalf("emprise = %+v, attendu 2 matchs, film et feuille lus", e)
	}
	if len(e.Players) != 1 || e.Players[0].XUID != "P" || e.Players[0].Gamertag != "Papa" {
		t.Errorf("fiches = %+v, attendu le seul joueur de la page (Alpha est un coéquipier)", e.Players)
	}
	if e.Habit != nil || e.Placement != nil || e.Maps != nil {
		t.Errorf("habitude %v, placement %v, grille %v : attendus absents sur Sessions", e.Habit, e.Placement, e.Maps)
	}
	if e.Equipment == nil {
		t.Error("équipement absent, attendu sur le match filmé")
	}
	if resp.CompareEmprise == nil || resp.CompareEmprise.MatchesTotal != 1 {
		t.Errorf("emprise comparée = %+v, attendu le seul match de la session comparée", resp.CompareEmprise)
	}
	var ferme domain.SessionPageResponse
	sc, canon = sessionsDeTest(false)
	serviceDeBlocs(usageTestRepoMock()).attachSessionBlocks(context.Background(), &ferme, sc, canon)
	if ferme.CompareEmprise != nil {
		t.Errorf("tiroir fermé : emprise comparée = %+v, attendu nil", ferme.CompareEmprise)
	}
}

// Halo 5 (sans résumé d'usage) : feuille seule ; feuille non supportée ou en échec : nommée ;
// session sans match : aucun bloc.
func TestAttachSessionBlocks_EmpriseDegradations(t *testing.T) {
	sc, canon := sessionsDeTest(false)
	var h5 domain.SessionPageResponse
	serviceDeBlocs(nil).attachSessionBlocks(context.Background(), &h5, sc, canon)
	if h5.Emprise == nil || h5.Emprise.FilmUnavailable != domain.EmpriseFilmUnsupported || h5.Emprise.Equipment != nil {
		t.Errorf("Halo 5 : emprise = %+v, attendu film_unsupported sans équipement", h5.Emprise)
	}
	if h5.FormesRetenues == nil || h5.FormesRetenues.UnavailableReason != domain.SessionUsageUnsupported {
		t.Errorf("Halo 5 : formes = %+v, attendu indisponibles (unsupported)", h5.FormesRetenues)
	}
	for _, cas := range []struct {
		err  error
		want string
	}{
		{fmt.Errorf("x : %w", games.ErrCapabilityNotSupported), domain.EmpriseSheetUnsupported},
		{errors.New("base indisponible"), domain.EmpriseSheetLoadFailed},
	} {
		var resp domain.SessionPageResponse
		NewSessionPageService(nil).WithPlayerMatchesRepo(nil, "halo_infinite", "Papa").
			WithSessionEmprise(feuilleFixe{err: cas.err}, "P").WithSessionUsageSummary(usageTestRepoMock(), "").
			attachSessionBlocks(context.Background(), &resp, sc, canon)
		if resp.Emprise == nil || resp.Emprise.SheetUnavailable != cas.want {
			t.Errorf("feuille %v : emprise = %+v, attendu %q", cas.err, resp.Emprise, cas.want)
		}
	}
	var vide domain.SessionPageResponse
	serviceDeBlocs(usageTestRepoMock()).attachSessionBlocks(context.Background(), &vide, sessionBlocksScope{}, canon)
	if vide.Emprise != nil || vide.LivesNearTeammate != nil || vide.FormesRetenues != nil {
		t.Errorf("session sans match : emprise %v, vies %v, formes %v, attendus nil", vide.Emprise, vide.LivesNearTeammate, vide.FormesRetenues)
	}
}

// viesEnregistreur — le dépôt des vies qui note CHAQUE lecture (matchs, joueur).
type viesEnregistreur struct {
	lues   domain.ViesLues
	err    error
	appels [][]string
	xuids  []string
}

func (v *viesEnregistreur) LoadLivesNearTeammate(_ context.Context, ids []string, xuid string) (domain.ViesLues, error) {
	v.appels = append(v.appels, ids)
	v.xuids = append(v.xuids, xuid)
	return v.lues, v.err
}

// Une lecture bornée par session (les matchs de CETTE session, le joueur de la page) ;
// capability absente ou lecture en échec : pas de bloc.
func TestAttachSessionBlocks_ViesBorneesParSession(t *testing.T) {
	sc, canon := sessionsDeTest(true)
	repo := &viesEnregistreur{lues: viesDeTest()}
	var resp domain.SessionPageResponse
	serviceDeBlocs(usageTestRepoMock()).WithSessionLives(repo).WithSessionRadarRange(map[string]int{"Slayer:Arena": 18}).
		attachSessionBlocks(context.Background(), &resp, sc, canon)
	if got := fmt.Sprint(repo.appels); got != "[[m1 m2] [m3]]" {
		t.Errorf("lectures = %s, attendu [[m1 m2] [m3]] : une par session, bornée à ses matchs", got)
	}
	if fmt.Sprint(repo.xuids) != "[P P]" {
		t.Errorf("joueurs lus = %v, attendu le xuid du joueur de la page", repo.xuids)
	}
	if resp.LivesNearTeammate == nil || resp.LivesNearTeammate.Near.Lives != 1 || resp.CompareLivesNearTeammate == nil {
		t.Errorf("vies = %+v, comparées = %+v", resp.LivesNearTeammate, resp.CompareLivesNearTeammate)
	}
	for _, svc := range []*SessionPageService{
		serviceDeBlocs(usageTestRepoMock()),
		serviceDeBlocs(usageTestRepoMock()).WithSessionLives(&viesEnregistreur{err: errors.New("base indisponible")}),
	} {
		var sans domain.SessionPageResponse
		svc.attachSessionBlocks(context.Background(), &sans, sc, canon)
		if sans.LivesNearTeammate != nil || sans.CompareLivesNearTeammate != nil {
			t.Errorf("vies = %+v / %+v, attendu absentes", sans.LivesNearTeammate, sans.CompareLivesNearTeammate)
		}
	}
}

// objectifsSession — les colonnes d'objectif et les prises nettes de drapeau du scope.
type objectifsSession struct {
	colonnes []squadformes.ObjectiveColumnRow
	prises   []sessionusage.FlagGrabsNetRow
}

func (o objectifsSession) LoadObjectiveColumnRows(context.Context, []string) ([]squadformes.ObjectiveColumnRow, error) {
	return o.colonnes, nil
}

func (o objectifsSession) LoadFlagGrabsNet(context.Context, []string) ([]sessionusage.FlagGrabsNetRow, error) {
	return o.prises, nil
}

// D7 — la feuille d'objectif de la session : le joueur seul en escouade, les prises nettes de
// drapeau dans les valeurs du joueur ; sans colonnes d'objectif, aucun match à objectif.
func TestAttachSessionBlocks_ObjectifDeLaSession(t *testing.T) {
	sc, canon := sessionsDeTest(false)
	objectifs := objectifsSession{
		colonnes: []squadformes.ObjectiveColumnRow{
			{MatchID: "m1", XUID: "P", Family: narrative.FamilyCTF, Values: map[string]float64{"flag_captures": 1}},
			{MatchID: "m1", XUID: "A", Family: narrative.FamilyCTF, Values: map[string]float64{"flag_captures": 2}},
		},
		prises: []sessionusage.FlagGrabsNetRow{{MatchID: "m1", XUID: "P", Raw: 5, Net: 3, Openings: 9, WindowMS: 1500}},
	}
	var resp domain.SessionPageResponse
	serviceDeBlocs(usageTestRepoMock()).WithSessionObjectives(objectifs).
		attachSessionBlocks(context.Background(), &resp, sc, canon)
	f := resp.FormesRetenues
	if f == nil || !f.Available || len(f.Squad) != 1 || f.Squad[0].XUID != "P" {
		t.Fatalf("formes = %+v, attendu disponibles avec le seul joueur de la page", f)
	}
	if len(f.Matches) != 1 || f.Matches[0].Objective == nil {
		t.Fatalf("matchs à objectif = %+v, attendu m1", f.Matches)
	}
	var net float64
	for _, p := range f.Matches[0].Objective.Players {
		if p.XUID == "P" {
			net = p.Values[narrative.GrandeurFlagGrabsNet]
		}
	}
	if net != 3 {
		t.Errorf("prises nettes du joueur = %v, attendu 3", net)
	}
	var sans domain.SessionPageResponse
	serviceDeBlocs(usageTestRepoMock()).attachSessionBlocks(context.Background(), &sans, sc, canon)
	if sans.FormesRetenues == nil || len(sans.FormesRetenues.Matches) != 0 {
		t.Errorf("sans colonnes d'objectif : formes = %+v, attendu aucun match à objectif", sans.FormesRetenues)
	}
}

func TestAttachSessionBlocks_Embleme(t *testing.T) {
	sc, canon := sessionsDeTest(false)
	var resp domain.SessionPageResponse
	serviceDeBlocs(nil).WithSessionEmblemLoader(emblemesFixes{"Papa": "/emblem/papa.png"}).
		attachSessionBlocks(context.Background(), &resp, sc, canon)
	if resp.PlayerEmblemURL != "/emblem/papa.png" {
		t.Errorf("emblème = %q, attendu celui du joueur de la page", resp.PlayerEmblemURL)
	}
	var sans domain.SessionPageResponse
	serviceDeBlocs(nil).attachSessionBlocks(context.Background(), &sans, sc, canon)
	if sans.PlayerEmblemURL != "" {
		t.Errorf("sans chargeur : %q, attendu vide", sans.PlayerEmblemURL)
	}
}

// ADR 0036 I6 — chaque lecture des blocs du film déclare sa section de durée : résumé d'usage,
// Emprise (feuille de match et film), vies, objectif, emblème.
func TestAttachSessionBlocks_SectionsDeDuree(t *testing.T) {
	sc, canon := sessionsDeTest(false)
	ctx, chrono := timing.WithTimings(context.Background())
	var resp domain.SessionPageResponse
	serviceDeBlocs(usageTestRepoMock()).WithSessionLives(&viesEnregistreur{lues: viesDeTest()}).
		WithSessionEmblemLoader(emblemesFixes{"Papa": "/emblem/papa.png"}).
		attachSessionBlocks(ctx, &resp, sc, canon)
	vues := map[string]bool{}
	for _, s := range chrono.Snapshot() {
		vues[s.Name] = true
	}
	for _, section := range []string{"usage_summary", "emprise", "lives", "squad_formes", "emblem"} {
		if !vues[section] {
			t.Errorf("section %q absente des durées de la page (sections vues : %v)", section, vues)
		}
	}
}
