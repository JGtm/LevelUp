package scheduler_test

// data_health_fake_store_test.go — le contrôle de santé sur un faux port.DataHealthStore : aucune base.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/scheduler"
)

type fakeHealthShared struct {
	counts      map[port.HealthProbe]int
	failing     map[port.HealthProbe]error
	notApplic   map[port.HealthProbe]bool
	pending     int
	pendingErr  error
	lusr        port.LUSRGaps
	lusrErr     error
	closeCalled bool
}

func (f *fakeHealthShared) Close() error { f.closeCalled = true; return nil }
func (f *fakeHealthShared) CountProbe(_ context.Context, p port.HealthProbe) (int, bool, error) {
	if err := f.failing[p]; err != nil {
		return 0, true, err
	}
	if f.notApplic[p] {
		return 0, false, nil
	}
	return f.counts[p], true, nil
}
func (f *fakeHealthShared) PendingKillSummaries(context.Context) (int, error) {
	return f.pending, f.pendingErr
}
func (f *fakeHealthShared) ScanLUSRGaps(context.Context, string, string) (port.LUSRGaps, error) {
	return f.lusr, f.lusrErr
}

type fakeHealthStore struct {
	shared     *fakeHealthShared
	openErr    error
	banners    int
	bannersErr error
	passesErr  error
	opened     int
}

func (f *fakeHealthStore) OpenShared(string) (port.DataHealthShared, error) {
	f.opened++
	if f.openErr != nil {
		return nil, f.openErr
	}
	return f.shared, nil
}
func (f *fakeHealthStore) CountPlayerProbe(context.Context, port.HealthProbe, string) (int, error) {
	return f.banners, f.bannersErr
}
func (f *fakeHealthStore) MeasurePasses(_ context.Context, _, _ string, table string) (migration.JaugePasses, bool, error) {
	return migration.JaugePasses{Table: table}, false, f.passesErr
}

// fakeHealthRoot : une racine dont le titre par défaut a un fichier de base partagée (vide : le faux magasin
// ne l'ouvre pas) et un joueur résolu.
func fakeHealthRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	pr := title.NewPathResolver(root)
	if err := os.MkdirAll(filepath.Dir(pr.SharedDBPath("halo_infinite")), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pr.SharedDBPath("halo_infinite"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pr.PlayerDir("halo_infinite", "Joueur"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pr.PlayerDir("halo_infinite", "Joueur"), "xuid.txt"), []byte("2603225103667623\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	playerPath := pr.PlayerDBPath("halo_infinite", "Joueur")
	if err := os.MkdirAll(filepath.Dir(playerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(playerPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestHealthFake_LesCompteursViennentDuMagasin(t *testing.T) {
	shared := &fakeHealthShared{
		counts: map[port.HealthProbe]int{
			port.ProbeUUIDsMapName: 2, port.ProbeUUIDsPairName: 1, port.ProbeLyingBitsEvents: 3,
			port.ProbeLyingBitsWeapons: 4, port.ProbeOrphanXUIDs: 5,
		},
		pending: 7,
		lusr:    port.LUSRGaps{InteriorGaps: 6, PendingRecent: 2},
	}
	store := &fakeHealthStore{shared: shared, banners: 9}
	res := scheduler.NewDataHealthScheduler(fakeHealthRoot(t), store).RunOnce(context.Background())

	if res.UUIDsRawCount != 3 || res.LyingBitsEvents != 3 || res.LyingBitsWeaponKills != 4 ||
		res.OrphanXUIDs != 5 || res.GarbageBannerURLs != 9 || res.KillSummariesPending != 7 {
		t.Errorf("compteurs = %+v", res)
	}
	if res.LUSRInteriorGaps != 6 || res.LUSRPendingRecent != 2 || res.LUSRPlayersScanned != 1 || res.LUSRPlayersUnmeasured != 0 {
		t.Errorf("trous LUSR = %+v", res)
	}
	if res.ProbeErrors != 0 || !shared.closeCalled {
		t.Errorf("sondes en échec = %d, base partagée rendue = %v", res.ProbeErrors, shared.closeCalled)
	}
}

// Une sonde en échec est comptée dans ProbeErrors et le rapport du cycle le montre : elle ne passe pas pour 0.
func TestHealthFake_UneSondeEnEchecReste_Comptee(t *testing.T) {
	boom := errors.New("table renommée")
	shared := &fakeHealthShared{
		counts:     map[port.HealthProbe]int{port.ProbeUUIDsPairName: 1},
		failing:    map[port.HealthProbe]error{port.ProbeUUIDsMapName: boom, port.ProbeOrphanXUIDs: boom},
		notApplic:  map[port.HealthProbe]bool{port.ProbeLyingBitsWeapons: true},
		pendingErr: boom,
		lusrErr:    boom,
	}
	store := &fakeHealthStore{shared: shared, bannersErr: boom}
	sched := scheduler.NewDataHealthScheduler(fakeHealthRoot(t), store)
	res := sched.RunOnce(context.Background())

	// map_name, orphelins, résumés du journal, bannières, scan LUSR : cinq sondes en échec.
	if res.ProbeErrors < 5 {
		t.Errorf("ProbeErrors = %d, attendu au moins 5 (une par sonde en échec)", res.ProbeErrors)
	}
	if res.UUIDsRawCount != 1 || res.LyingBitsWeaponKills != 0 {
		t.Errorf("les sondes réussies ou non applicables gardent leur valeur : %+v", res)
	}
	if res.LUSRPlayersUnmeasured != 1 {
		t.Errorf("un scan LUSR en échec laisse le joueur non mesuré : %+v", res)
	}
	last, _ := sched.LastResult()
	if last == nil || last.ProbeErrors != res.ProbeErrors {
		t.Errorf("le rapport du dernier cycle doit montrer les sondes en échec : %+v", last)
	}
}

func TestHealthFake_BaseInouvrableOuJoueurIndisponible(t *testing.T) {
	// Base partagée inouvrable : cycle avorté, rien de mémorisé.
	store := &fakeHealthStore{openErr: errors.New("tenue en écriture")}
	sched := scheduler.NewDataHealthScheduler(fakeHealthRoot(t), store)
	sched.RunOnce(context.Background())
	if last, _ := sched.LastResult(); last != nil {
		t.Errorf("un cycle sans base lisible ne doit rien mémoriser : %+v", last)
	}

	// Joueur indisponible : non mesuré (jauge gelée), sans sonde en échec ; bannière sautée.
	shared := &fakeHealthShared{lusrErr: port.ErrDataHealthUnavailable}
	store = &fakeHealthStore{shared: shared, bannersErr: port.ErrDataHealthUnavailable}
	res := scheduler.NewDataHealthScheduler(fakeHealthRoot(t), store).RunOnce(context.Background())
	if res.LUSRPlayersUnmeasured != 1 || res.ProbeErrors != 0 || res.GarbageBannerURLs != 0 {
		t.Errorf("joueur indisponible : %+v", res)
	}
}
