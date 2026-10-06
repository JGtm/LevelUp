package teammates

// teammates_service_emprise_placement_test.go — le bloc « Groupés ou isolés » publié par GetPage
// (plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V3.3), avec un dépôt simulé : une lecture
// bornée par requête (ADR 0036 I4), la portée courante résolue par la source unique, et les
// dégradations (capability absente, lecture en échec, portée périmée) — la page ne tombe jamais.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// fakePlacement — le dépôt simulé : rend `read` / `err` et note chaque appel.
type fakePlacement struct {
	read   squademprise.PlacementRead
	err    error
	appels [][2][]string
}

func (f *fakePlacement) LoadLifePlacement(_ context.Context, matchIDs, xuids []string) (squademprise.PlacementRead, error) {
	f.appels = append(f.appels, [2][]string{append([]string(nil), matchIDs...), append([]string(nil), xuids...)})
	return f.read, f.err
}

var _ port.SquadLifePlacementRepository = (*fakePlacement)(nil)

func vieDuPlacement(xuid string, start int64, medianM, radarM float64, kills int) squademprise.PlacementRow {
	beyond := int64(3000)
	return squademprise.PlacementRow{
		MatchID: "m1", XUID: xuid, StartMS: start, EndMS: start + 30000, DurationMS: 30000,
		MeasuredMS: 12000, MedianM: &medianM, BeyondMS: &beyond, RadarM: &radarM, Kills: kills,
	}
}

// pagePlacement — GetPage sur le périmètre D2 de l'Emprise (m1, Main + Ally1), placement câblé
// ou non (`repo` nil), portée « Slayer:Arena » = 18 m.
func pagePlacement(t *testing.T, repo port.SquadLifePlacementRepository) *domain.SquadEmpriseBlock {
	t.Helper()
	squad, usage := usageFixture(time.Now().UTC().Add(-time.Hour))
	svc := NewTeammatesService(squad, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(squad.synthRows, squad.synthErr), "halo_infinite", "Main").
		WithUsageSummary(usage).
		WithRadarRange(map[string]int{"Slayer:Arena": 18})
	if repo != nil {
		svc = svc.WithLifePlacement(repo)
	}
	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"Ally1"},
	})
	if err != nil {
		t.Fatalf("GetPage : %v — le placement ne doit jamais faire tomber la page", err)
	}
	if resp.SquadEmprise == nil {
		t.Fatal("bloc Emprise absent")
	}
	return resp.SquadEmprise
}

// ligneDeLog rend la ligne JSON du message `msg`, ou "".
func ligneDeLog(logs, msg string) string {
	for _, l := range strings.Split(logs, "\n") {
		if strings.Contains(l, `"msg":"`+msg+`"`) {
			return l
		}
	}
	return ""
}

func TestGetPage_Placement_UneLectureBorneeEtLaPorteeCourante(t *testing.T) {
	repo := &fakePlacement{read: squademprise.PlacementRead{
		Rows: []squademprise.PlacementRow{
			vieDuPlacement("player-xuid", 1000, 9, 18, 2),
			vieDuPlacement("x1", 1000, 27, 18, 0),
		},
		// La variante telle que la base la porte, espace compris : résolue par mappings.PorteeDuRadar.
		Variants: map[string]string{"m1": " Slayer:Arena "},
	}}
	b := pagePlacement(t, repo)
	if len(repo.appels) != 1 {
		t.Fatalf("%d lecture(s) du placement, attendu une par requête", len(repo.appels))
	}
	if ids, xuids := repo.appels[0][0], repo.appels[0][1]; fmt.Sprint(ids) != "[m1]" || fmt.Sprint(xuids) != "[player-xuid x1]" {
		t.Errorf("lecture bornée par %v / %v, attendu le périmètre [m1] et la composition [player-xuid x1]", ids, xuids)
	}
	p := b.Placement
	if p == nil || len(p.Players) != 2 || p.Players[0].Gamertag != "Main" || p.Players[1].Gamertag != "Ally1" {
		t.Fatalf("placement = %+v, attendu Main puis Ally1 (ordre des fiches)", p)
	}
	if v := p.Players[1].Lives[0]; v.RadarRatio != 1.5 || v.Quadrant != domain.EmprisePlacementIsolatedCostly {
		t.Errorf("vie d'Ally1 = %+v, attendu X 1,5 (27 m / 18 m) et « isolé et coûteux »", v)
	}
	if p.Coverage.MatchesTotal != 1 || p.Coverage.LivesMeasured != 2 || p.IsolatedFromRatio != 1 {
		t.Errorf("couverture = %+v", p.Coverage)
	}
}

// Capability absente : le lecteur n'est pas câblé (porte `film.kill_positions` du câblage) —
// bloc absent, ErrCapabilityNotSupported journalisé en Debug, aucune lecture.
func TestGetPage_Placement_CapabilityAbsente(t *testing.T) {
	var b *domain.SquadEmpriseBlock
	logs := withCapturedLogs(t, func() { b = pagePlacement(t, nil) })
	if b.Placement != nil {
		t.Errorf("placement publié sans capability : %+v", b.Placement)
	}
	l := ligneDeLog(logs, "teammates_emprise_placement_capability_absente")
	if !strings.Contains(l, `"level":"DEBUG"`) || !strings.Contains(l, games.ErrCapabilityNotSupported.Error()) {
		t.Errorf("journal attendu en Debug avec ErrCapabilityNotSupported, obtenu : %q", l)
	}
}

// Le dépôt dit la capability absente (table manquante) : même dégradation.
func TestGetPage_Placement_DepotNonSupporte(t *testing.T) {
	var b *domain.SquadEmpriseBlock
	repo := &fakePlacement{err: fmt.Errorf("lecture : %w", games.ErrCapabilityNotSupported)}
	logs := withCapturedLogs(t, func() { b = pagePlacement(t, repo) })
	l := ligneDeLog(logs, "teammates_emprise_placement_capability_absente")
	if b.Placement != nil || !strings.Contains(l, `"level":"DEBUG"`) {
		t.Errorf("placement %+v, journal %q ; attendu absent et Debug", b.Placement, l)
	}
}

// Lecture en échec : bloc absent, journalisé en Error, la page et l'Emprise restent servies.
func TestGetPage_Placement_LectureEnEchec(t *testing.T) {
	var b *domain.SquadEmpriseBlock
	logs := withCapturedLogs(t, func() { b = pagePlacement(t, &fakePlacement{err: errors.New("base indisponible")}) })
	l := ligneDeLog(logs, "teammates_emprise_placement_en_echec")
	if b.Placement != nil || !strings.Contains(l, `"level":"ERROR"`) || b.MatchesTotal != 1 {
		t.Errorf("placement %+v, journal %q, périmètre %d ; attendu absent, Error, Emprise servie",
			b.Placement, l, b.MatchesTotal)
	}
}

// Portée périmée : la vie est écartée par le calcul, comptée, et le service le dit en Warn.
func TestGetPage_Placement_PorteePerimeeJournalisee(t *testing.T) {
	var b *domain.SquadEmpriseBlock
	repo := &fakePlacement{read: squademprise.PlacementRead{
		Rows: []squademprise.PlacementRow{
			vieDuPlacement("player-xuid", 1000, 9, 18, 2),
			vieDuPlacement("x1", 1000, 9, 24, 1),
		},
		Variants: map[string]string{"m1": "Slayer:Arena"},
	}}
	logs := withCapturedLogs(t, func() { b = pagePlacement(t, repo) })
	if b.Placement == nil || b.Placement.Coverage.StaleLives != 1 || b.Placement.Coverage.LivesTotal != 1 {
		t.Fatalf("placement = %+v, attendu 1 vie périmée écartée", b.Placement)
	}
	l := ligneDeLog(logs, "teammates_emprise_placement_portee_perimee")
	if !strings.Contains(l, `"level":"WARN"`) || !strings.Contains(l, `"vies":1`) || !strings.Contains(l, "m1") {
		t.Errorf("journal attendu en Warn avec le compte et le match, obtenu : %q", l)
	}
}

// Sans bloc Emprise (aucun film, aucun périmètre) : rien à attacher, aucune lecture du placement.
func TestAttacherPlacement_SansBlocEmprise_NeLitRien(t *testing.T) {
	repo := &fakePlacement{}
	svc := NewTeammatesService(nil, nil).WithLifePlacement(repo)
	svc.attacherPlacement(context.Background(), nil, nil)
	if len(repo.appels) != 0 {
		t.Errorf("%d lecture(s) du placement sans bloc Emprise, attendu aucune", len(repo.appels))
	}
}
