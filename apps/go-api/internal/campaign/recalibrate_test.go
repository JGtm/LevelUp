package campaign

import (
	"context"
	"errors"
	"testing"
	"time"
)

// recalibrate_test.go — recalage au démarrage du départ des campagnes de l'axe Objectif.

// Fakes : le dépôt de service_test.go reçoit les deux méthodes de recalage.

func (r *fakeRepo) ListOpen(_ context.Context, userID, titleSlug string) ([]ImprovementCampaign, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []ImprovementCampaign
	for _, c := range r.byID {
		if c.UserID == userID && c.TitleSlug == titleSlug && (c.Status == StatusActive || c.Status == StatusPaused) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *fakeRepo) RestartMeasure(_ context.Context, id string, start MeasureStart) error {
	if err := r.restartErr[id]; err != nil {
		return err
	}
	c, ok := r.byID[id]
	if !ok || (c.Status != StatusActive && c.Status != StatusPaused) {
		return nil
	}
	c.SnapshotValue, c.SnapshotSample, c.MatchesSinceStart = start.SnapshotValue, start.SnapshotSample, start.MatchesSinceStart
	c.CurrentValueRaw, c.CurrentValueLOWESS, c.MannWhitneyP, c.LastEvaluatedAt = nil, nil, nil, nil
	c.ProgressionConfirmed, c.AutoClosureSuggested, c.AutoClosureReason = false, false, ""
	r.byID[id] = c
	r.restarts = append(r.restarts, id)
	return nil
}

// fakeRevisionStore : révision rangée en mémoire.
type fakeRevisionStore struct {
	rev    string
	stores int
}

func (f *fakeRevisionStore) Stored(context.Context) (string, error) { return f.rev, nil }
func (f *fakeRevisionStore) Store(_ context.Context, rev string) error {
	f.rev = rev
	f.stores++
	return nil
}

var recalNow = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

// seedEvaluated : une campagne déjà évaluée (valeurs courantes, test, suggestion) à l'ancien départ.
func seedEvaluated(repo *fakeRepo, id, axis string, kind AxisKind, status CampaignStatus) {
	raw, lowess, p := 61.0, 62.0, 0.01
	evaluated := recalNow.Add(-time.Hour)
	repo.byID[id] = ImprovementCampaign{
		ID: id, UserID: "u1", TitleSlug: "halo", Axis: axis, AxisKind: kind,
		StartedAt: recalNow.AddDate(0, 0, -30), Status: status, PlaylistGroup: "all",
		SnapshotValue: 40, SnapshotSample: 12, MatchesSinceStart: 9,
		CurrentValueRaw: &raw, CurrentValueLOWESS: &lowess, MannWhitneyP: &p, LastEvaluatedAt: &evaluated,
		ProgressionConfirmed: true, AutoClosureSuggested: true, AutoClosureReason: "plateau_60d",
	}
}

// Les campagnes de seedEvaluated démarrent le 2026-10-21 12:00 (recalNow - 30 jours) : fenêtre de départ
// [2026-04-24 12:00, 2026-10-21 12:00], fenêtre depuis le départ [2026-10-21 12:00, recalNow].
var (
	recalStart       = utc(2026, 10, 21, 12)
	recalStartWindow = window{utc(2026, 4, 24, 12), recalStart}
	recalSinceWindow = window{recalStart, recalNow}
)

func newRecalSamples() *windowSamples {
	return &windowSamples{byWindow: map[window][]float64{
		recalStartWindow: {10, 20, 30},
		recalSinceWindow: {50, 60},
	}}
}

func newRecalService(repo *fakeRepo) *Service { return NewService(repo, newRecalSamples()) }

// Une campagne active ou en pause sur l'axe Objectif repart d'un départ mesuré avec la mesure courante,
// évaluation vide ; une campagne close, ou sur un autre axe (radar ou LUSR), ne bouge pas ; la révision
// est rangée.
func TestApplyObjectiveAxisRev_RecaleLesCampagnesOuvertesDeLAxe(t *testing.T) {
	repo := newFakeRepo()
	seedEvaluated(repo, "active", AxisObjective, AxisKindRadar, StatusActive)
	seedEvaluated(repo, "paused", AxisObjective, AxisKindRadar, StatusPaused)
	seedEvaluated(repo, "completed", AxisObjective, AxisKindRadar, StatusCompleted)
	seedEvaluated(repo, "abandoned", AxisObjective, AxisKindRadar, StatusAbandoned)
	seedEvaluated(repo, "combat", "combat", AxisKindRadar, StatusActive)
	seedEvaluated(repo, "lusr", AxisObjective, AxisKindLUSRComponent, StatusActive)
	store := &fakeRevisionStore{}

	n, err := newRecalService(repo).ApplyObjectiveAxisRev(context.Background(), store, "u1", "halo", recalNow)
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v, attendu 2 campagnes recalées sans erreur", n, err)
	}
	for _, id := range []string{"active", "paused"} {
		c := repo.byID[id]
		if c.SnapshotValue != 20 || c.SnapshotSample != 3 || c.MatchesSinceStart != 2 {
			t.Errorf("%s : départ = %v sur %d échantillons, %d matchs depuis ; attendu 20 sur 3, 2 matchs", id,
				c.SnapshotValue, c.SnapshotSample, c.MatchesSinceStart)
		}
		if c.CurrentValueRaw != nil || c.CurrentValueLOWESS != nil || c.MannWhitneyP != nil || c.LastEvaluatedAt != nil ||
			c.ProgressionConfirmed || c.AutoClosureSuggested || c.AutoClosureReason != "" {
			t.Errorf("%s : évaluation non remise à vide : %+v", id, c)
		}
	}
	for _, id := range []string{"completed", "abandoned", "combat", "lusr"} {
		if c := repo.byID[id]; c.SnapshotValue != 40 || c.SnapshotSample != 12 || c.CurrentValueRaw == nil || !c.ProgressionConfirmed {
			t.Errorf("%s : campagne modifiée alors qu'elle n'est pas concernée : %+v", id, c)
		}
	}
	if store.rev != ObjectiveAxisRev || store.stores != 1 {
		t.Errorf("révision rangée = %q (%d écriture), attendu %q (1)", store.rev, store.stores, ObjectiveAxisRev)
	}
}

// Une fois la révision rangée, un second passage ne touche à rien : l'évaluation faite depuis reste.
func TestApplyObjectiveAxisRev_SecondPassageNeFaitRien(t *testing.T) {
	repo := newFakeRepo()
	seedEvaluated(repo, "active", AxisObjective, AxisKindRadar, StatusActive)
	store := &fakeRevisionStore{}
	svc := newRecalService(repo)
	if _, err := svc.ApplyObjectiveAxisRev(context.Background(), store, "u1", "halo", recalNow); err != nil {
		t.Fatal(err)
	}
	raw := 55.0
	c := repo.byID["active"]
	c.CurrentValueRaw = &raw // évaluation postérieure au recalage
	repo.byID["active"] = c

	n, err := svc.ApplyObjectiveAxisRev(context.Background(), store, "u1", "halo", recalNow)
	if err != nil || n != 0 {
		t.Fatalf("second passage : n=%d err=%v, attendu 0, nil", n, err)
	}
	if len(repo.restarts) != 1 || repo.byID["active"].CurrentValueRaw == nil {
		t.Errorf("second passage : recalages = %v, évaluation conservée = %v ; attendu un seul recalage et l'évaluation gardée",
			repo.restarts, repo.byID["active"].CurrentValueRaw != nil)
	}
	if store.stores != 1 {
		t.Errorf("la révision à jour est réécrite (%d écritures)", store.stores)
	}
}

// Un joueur sans campagne sur l'axe reçoit quand même la révision (rien à retenter au prochain démarrage).
func TestApplyObjectiveAxisRev_SansCampagneRangeLaRevision(t *testing.T) {
	store := &fakeRevisionStore{}
	n, err := newRecalService(newFakeRepo()).ApplyObjectiveAxisRev(context.Background(), store, "u1", "halo", recalNow)
	if err != nil || n != 0 || store.rev != ObjectiveAxisRev {
		t.Fatalf("n=%d err=%v rev=%q, attendu 0, nil, %q", n, err, store.rev, ObjectiveAxisRev)
	}
}

// Un échec d'écriture sur une campagne n'arrête pas les autres, est rendu, et ne range pas la révision :
// le prochain démarrage retente.
func TestApplyObjectiveAxisRev_EchecNeRangePasLaRevision(t *testing.T) {
	repo := newFakeRepo()
	seedEvaluated(repo, "a", AxisObjective, AxisKindRadar, StatusActive)
	seedEvaluated(repo, "b", AxisObjective, AxisKindRadar, StatusPaused)
	boom := errors.New("écriture refusée")
	repo.restartErr = map[string]error{"a": boom}
	store := &fakeRevisionStore{}

	n, err := newRecalService(repo).ApplyObjectiveAxisRev(context.Background(), store, "u1", "halo", recalNow)
	if !errors.Is(err, boom) || n != 1 {
		t.Fatalf("n=%d err=%v, attendu 1 campagne recalée et l'erreur d'écriture", n, err)
	}
	if store.rev != "" || repo.byID["b"].SnapshotValue != 20 {
		t.Errorf("révision rangée = %q (attendu vide) ; départ de b = %v (attendu 20, recalée malgré l'échec de a)",
			store.rev, repo.byID["b"].SnapshotValue)
	}
}

// Une liste des campagnes illisible rend l'erreur sans ranger la révision.
func TestApplyObjectiveAxisRev_ListeIllisible(t *testing.T) {
	repo := newFakeRepo()
	repo.listErr = errors.New("base illisible")
	store := &fakeRevisionStore{}
	if _, err := newRecalService(repo).ApplyObjectiveAxisRev(context.Background(), store, "u1", "halo", recalNow); err == nil {
		t.Fatal("erreur attendue")
	}
	if store.rev != "" {
		t.Errorf("révision rangée malgré l'échec : %q", store.rev)
	}
}

// Le recalage lit, pour chaque campagne, la fenêtre de départ de son démarrage puis la fenêtre depuis le
// départ : mêmes bornes que StartCampaign et l'évaluation (dates écrites en dur).
func TestApplyObjectiveAxisRev_FenetresDeLecture(t *testing.T) {
	repo := newFakeRepo()
	seedEvaluated(repo, "active", AxisObjective, AxisKindRadar, StatusActive)
	w := newRecalSamples()

	if _, err := NewService(repo, w).ApplyObjectiveAxisRev(context.Background(), &fakeRevisionStore{}, "u1", "halo", recalNow); err != nil {
		t.Fatal(err)
	}
	wantCalls(t, w.calls, recalStartWindow, recalSinceWindow)
}

// Plus de 100 échantillons dans la fenêtre de départ : le recalage garde les 100 derniers.
func TestApplyObjectiveAxisRev_PlafondA100(t *testing.T) {
	repo := newFakeRepo()
	seedEvaluated(repo, "active", AxisObjective, AxisKindRadar, StatusActive)
	w := &windowSamples{byWindow: map[window][]float64{recalStartWindow: sequence(150)}}

	if _, err := NewService(repo, w).ApplyObjectiveAxisRev(context.Background(), &fakeRevisionStore{}, "u1", "halo", recalNow); err != nil {
		t.Fatal(err)
	}
	if c := repo.byID["active"]; c.SnapshotSample != 100 || c.SnapshotValue != 100.5 {
		t.Errorf("départ = %v sur %d, attendu 100,5 sur 100", c.SnapshotValue, c.SnapshotSample)
	}
}

// Sans aucun match mesuré dans la fenêtre, le recalage écrit un départ NON MESURÉ (zéro échantillon) :
// l'ancien départ chiffré ne survit pas, et le 0 qui l'accompagne ne se lit pas comme une mesure
// (StartMeasured faux, aucun écart chiffré).
func TestApplyObjectiveAxisRev_SansMatchMesureDepartNonMesure(t *testing.T) {
	repo := newFakeRepo()
	seedEvaluated(repo, "active", AxisObjective, AxisKindRadar, StatusActive)
	store := &fakeRevisionStore{}

	n, err := NewService(repo, &windowSamples{}).ApplyObjectiveAxisRev(context.Background(), store, "u1", "halo", recalNow)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v, attendu 1 campagne recalée", n, err)
	}
	c := repo.byID["active"]
	if c.SnapshotSample != 0 || c.SnapshotValue != 0 || c.MatchesSinceStart != 0 || c.StartMeasured() {
		t.Errorf("départ = %v sur %d (%d matchs depuis), mesuré = %v ; attendu 0 sur 0, non mesuré",
			c.SnapshotValue, c.SnapshotSample, c.MatchesSinceStart, c.StartMeasured())
	}
	raw := 50.0
	c.CurrentValueRaw = &raw
	if c.Delta() != nil {
		t.Errorf("écart chiffré contre un départ non mesuré : %v", *c.Delta())
	}
	if store.rev != ObjectiveAxisRev {
		t.Errorf("révision rangée = %q, attendu %q", store.rev, ObjectiveAxisRev)
	}
}

// Un titre sans statistiques d'objectif (capacité match.objective.stats absente) n'a rien à recaler : sa
// campagne sur l'axe n'est pas réécrite à 0 sur 0, la révision est rangée et aucune mesure n'est lue.
func TestApplyObjectiveAxisRev_TitreSansStatistiquesDObjectif(t *testing.T) {
	repo := newFakeRepo()
	seedEvaluated(repo, "active", AxisObjective, AxisKindRadar, StatusActive)
	w := newRecalSamples()
	store := &fakeRevisionStore{}

	n, err := NewService(repo, w).WithObjectiveMeasure(false).
		ApplyObjectiveAxisRev(context.Background(), store, "u1", "halo", recalNow)
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v, attendu 0, nil", n, err)
	}
	c := repo.byID["active"]
	if c.SnapshotValue != 40 || c.SnapshotSample != 12 || c.CurrentValueRaw == nil || !c.ProgressionConfirmed {
		t.Errorf("campagne réécrite alors que le titre ne mesure pas l'axe : %+v", c)
	}
	if len(repo.restarts) != 0 || len(w.calls) != 0 {
		t.Errorf("recalages = %v, mesures lues = %v, attendu aucun", repo.restarts, w.calls)
	}
	if store.rev != ObjectiveAxisRev {
		t.Errorf("révision rangée = %q, attendu %q", store.rev, ObjectiveAxisRev)
	}
}
