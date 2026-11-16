package campaign

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// start_window_test.go — la fenêtre qui fonde le point de départ (loadStartSamples) : mêmes bornes pour
// StartCampaign, l'évaluation et le recalage, plafond de 100, départ non mesuré. Les dates attendues sont
// écrites en dur : elles ne se calculent pas avec les constantes du code testé.

// window est une fenêtre [since, until] demandée au fournisseur d'échantillons.
type window struct{ since, until time.Time }

// windowSamples : fournisseur d'échantillons qui enregistre chaque fenêtre demandée et rend les valeurs
// associées à cette fenêtre exacte (rien pour une fenêtre inconnue).
type windowSamples struct {
	byWindow map[window][]float64
	errs     map[window]error // fenêtres dont la lecture échoue
	calls    []window
}

func (w *windowSamples) LoadAxisSamples(
	_ context.Context, _, _, _ string, _ AxisKind, _ string, since, until time.Time,
) ([]float64, error) {
	k := window{since, until}
	w.calls = append(w.calls, k)
	if err := w.errs[k]; err != nil {
		return nil, err
	}
	return w.byWindow[k], nil
}

func utc(y int, m time.Month, d, h int) time.Time { return time.Date(y, m, d, h, 0, 0, 0, time.UTC) }

// sequence rend 1, 2, ..., n.
func sequence(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = float64(i + 1)
	}
	return out
}

func wantCalls(t *testing.T, got []window, want ...window) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("fenêtres demandées = %v, attendu %v", got, want)
	}
	for i := range want {
		if !got[i].since.Equal(want[i].since) || !got[i].until.Equal(want[i].until) {
			t.Fatalf("fenêtre %d = [%v, %v], attendu [%v, %v]", i, got[i].since, got[i].until, want[i].since, want[i].until)
		}
	}
}

var startParamsRadar = StartParams{UserID: "u1", TitleSlug: "halo", Axis: "combat", AxisKind: AxisKindRadar}

// StartCampaign lit les 180 jours qui précèdent « maintenant », bornes comprises.
func TestStartCampaign_FenetreDe180Jours(t *testing.T) {
	now := utc(2026, 5, 18, 12)
	w := &windowSamples{byWindow: map[window][]float64{{utc(2025, 11, 19, 12), now}: {2, 4}}}

	c, err := NewService(newFakeRepo(), w).StartCampaign(context.Background(), startParamsRadar, now)
	if err != nil {
		t.Fatal(err)
	}
	wantCalls(t, w.calls, window{utc(2025, 11, 19, 12), now})
	if c.SnapshotValue != 3 || c.SnapshotSample != 2 {
		t.Errorf("départ = %v sur %d, attendu 3 sur 2", c.SnapshotValue, c.SnapshotSample)
	}
}

// Au-delà de 100 échantillons dans la fenêtre, seuls les 100 derniers fondent le départ.
func TestStartCampaign_PlafondA100Echantillons(t *testing.T) {
	now := utc(2026, 5, 18, 12)
	w := &windowSamples{byWindow: map[window][]float64{{utc(2025, 11, 19, 12), now}: sequence(150)}}

	c, err := NewService(newFakeRepo(), w).StartCampaign(context.Background(), startParamsRadar, now)
	if err != nil {
		t.Fatal(err)
	}
	// 51..150 : moyenne 100,5.
	if c.SnapshotSample != 100 || c.SnapshotValue != 100.5 {
		t.Errorf("départ = %v sur %d, attendu 100,5 sur 100", c.SnapshotValue, c.SnapshotSample)
	}
}

// Sans aucun match mesuré, le départ est NON MESURÉ : zéro échantillon, et aucun écart n'est chiffré
// même une fois la campagne évaluée.
func TestStartCampaign_SansMatchMesureDepartNonMesure(t *testing.T) {
	now := utc(2026, 5, 18, 12)
	repo := newFakeRepo()
	svc := NewService(repo, &windowSamples{})

	c, err := svc.StartCampaign(context.Background(), startParamsRadar, now)
	if err != nil {
		t.Fatal(err)
	}
	if c.SnapshotSample != 0 || c.StartMeasured() {
		t.Fatalf("SnapshotSample = %d, StartMeasured = %v, attendu 0 et faux", c.SnapshotSample, c.StartMeasured())
	}
	raw, lowess := 55.0, 56.0
	c.CurrentValueRaw, c.CurrentValueLOWESS = &raw, &lowess
	if d := c.Delta(); d != nil {
		t.Errorf("Delta d'un départ non mesuré = %v, attendu aucun", *d)
	}
}

// L'évaluation lit la même fenêtre de départ (population « avant ») puis la fenêtre depuis le départ, et
// persiste ce qu'elle calcule.
func TestEvaluate_FenetresEtResultatPersiste(t *testing.T) {
	started, now := utc(2026, 5, 18, 12), utc(2026, 6, 1, 12)
	w := &windowSamples{byWindow: map[window][]float64{
		{utc(2025, 11, 19, 12), started}: {1, 2, 3},
		{started, now}:                   {10, 20},
	}}
	repo := newFakeRepo()
	c := ImprovementCampaign{
		ID: "c1", UserID: "u1", TitleSlug: "halo", Axis: "combat", AxisKind: AxisKindRadar,
		StartedAt: started, Status: StatusActive, PlaylistGroup: "all", SnapshotValue: 2, SnapshotSample: 3,
	}
	repo.byID[c.ID] = c

	if err := NewService(repo, w).Evaluate(context.Background(), c, now); err != nil {
		t.Fatal(err)
	}
	wantCalls(t, w.calls, window{utc(2025, 11, 19, 12), started}, window{started, now})
	if len(repo.evals) != 1 {
		t.Fatalf("évaluations persistées = %d, attendu 1", len(repo.evals))
	}
	e := repo.evals[0]
	if e.MatchesSinceStart != 2 || !e.CurrentRaw.Valid || e.CurrentRaw.Float64 != 15 || !e.EvaluatedAt.Equal(now) {
		t.Errorf("évaluation = %+v, attendu 2 matchs, valeur brute 15, évaluée à %v", e, now)
	}
}

// La population « avant » de l'évaluation est plafonnée comme le départ : les 100 derniers.
func TestLoadCampaignSnapshots_PlafondA100(t *testing.T) {
	started, now := utc(2026, 5, 18, 12), utc(2026, 6, 1, 12)
	w := &windowSamples{byWindow: map[window][]float64{{utc(2025, 11, 19, 12), started}: sequence(150)}}
	c := ImprovementCampaign{UserID: "u1", TitleSlug: "halo", Axis: "combat", AxisKind: AxisKindRadar, StartedAt: started}

	pre, _, err := NewService(newFakeRepo(), w).loadCampaignSnapshots(context.Background(), c, now)
	if err != nil {
		t.Fatal(err)
	}

	if len(pre) != 100 || pre[0] != 51 || pre[99] != 150 {
		t.Fatalf("population avant = %d valeurs de %v à %v, attendu les 100 dernières (51 à 150)", len(pre), pre[0], pre[len(pre)-1])
	}
}

// Un départ non mesuré n'est pas une référence de plateau : sans lui, une valeur courante proche de 0 ne
// suggère pas la clôture « plateau_60d », alors qu'elle la suggère contre un départ mesuré identique.
func TestEvaluate_PlateauSeulementContreUnDepartMesure(t *testing.T) {
	started := utc(2026, 3, 1, 12)
	now := started.AddDate(0, 0, 70)
	post := []float64{0, 0, 0, 0}
	w := &windowSamples{byWindow: map[window][]float64{{started, now}: post}}
	for _, tc := range []struct {
		name   string
		sample int
		want   bool
	}{{"départ non mesuré", 0, false}, {"départ mesuré", 12, true}} {
		repo := newFakeRepo()
		c := ImprovementCampaign{
			ID: "c1", UserID: "u1", TitleSlug: "halo", Axis: "combat", AxisKind: AxisKindRadar,
			StartedAt: started, Status: StatusActive, PlaylistGroup: "all", SnapshotValue: 0, SnapshotSample: tc.sample,
		}
		repo.byID[c.ID] = c
		if err := NewService(repo, w).Evaluate(context.Background(), c, now); err != nil {
			t.Fatalf("%s : %v", tc.name, err)
		}
		if got := repo.evals[0].AutoClosureSuggested; got != tc.want {
			t.Errorf("%s : clôture suggérée = %v, attendu %v", tc.name, got, tc.want)
		}
	}
}

// Les échantillons depuis le départ illisibles : l'erreur est journalisée (campagne, axe, cause) et rendue,
// et aucune évaluation n'est écrite (une évaluation vide effacerait la précédente). Une population
// « avant » illisible, elle, dégrade l'évaluation sans l'empêcher.
func TestEvaluate_EchantillonsIllisibles(t *testing.T) {
	started, now := utc(2026, 5, 18, 12), utc(2026, 6, 1, 12)
	startWin, sinceWin := window{utc(2025, 11, 19, 12), started}, window{started, now}
	boom := errors.New("lecture refusée")
	c := ImprovementCampaign{
		ID: "c-illisible", UserID: "u1", TitleSlug: "halo", Axis: "combat", AxisKind: AxisKindRadar,
		StartedAt: started, Status: StatusActive, PlaylistGroup: "all", SnapshotValue: 2, SnapshotSample: 3,
	}

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	repo := newFakeRepo()
	repo.byID[c.ID] = c
	w := &windowSamples{errs: map[window]error{sinceWin: boom}, byWindow: map[window][]float64{startWin: {1, 2, 3}}}
	if err := NewService(repo, w).Evaluate(context.Background(), c, now); !errors.Is(err, boom) {
		t.Fatalf("Evaluate = %v, attendu l'erreur de lecture", err)
	}
	if len(repo.evals) != 0 {
		t.Errorf("évaluations écrites = %d, attendu aucune", len(repo.evals))
	}
	logs := buf.String()
	for _, want := range []string{"level=ERROR", "campaign_id=c-illisible", "axis=combat", "lecture refusée"} {
		if !strings.Contains(logs, want) {
			t.Errorf("le journal ne contient pas %q :"+string(rune(10))+"%s", want, logs)
		}
	}

	// Population « avant » illisible : l'évaluation se fait, sans test de progression.
	repo = newFakeRepo()
	repo.byID[c.ID] = c
	w = &windowSamples{errs: map[window]error{startWin: boom}, byWindow: map[window][]float64{sinceWin: {10, 20}}}
	if err := NewService(repo, w).Evaluate(context.Background(), c, now); err != nil {
		t.Fatalf("Evaluate avec départ illisible = %v, attendu nil", err)
	}
	if len(repo.evals) != 1 || repo.evals[0].MannWhitneyP.Valid {
		t.Errorf("évaluations = %+v, attendu une, sans test de progression", repo.evals)
	}
}
