package replayregen

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// rasterRig : les fichiers tactiques du faux disque (périmé ou non) et ce que la file en a refait.
type rasterRig struct {
	mu      sync.Mutex
	stale   map[string]bool
	asked   []string // matchs dont l'état du fichier tactique a été lu
	rebuilt []string
	err     error
}

func (r *rasterRig) isStale(_ context.Context, _, id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.asked = append(r.asked, id)
	return r.stale[id]
}

func (r *rasterRig) rebuild(_ context.Context, _, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.rebuilt = append(r.rebuilt, id)
	r.stale[id] = false
	return nil
}

func (r *rasterRig) rebuiltIDs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.rebuilt...)
}

func (r *rasterRig) askedIDs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.asked...)
}

// newRasterFixture relie la file avec les deux gestes du fichier tactique ; stale : les matchs dont il est périmé.
func newRasterFixture(t *testing.T, stale ...string) (*fixture, *rasterRig) {
	t.Helper()
	f := newFixture(t)
	r := &rasterRig{stale: map[string]bool{}}
	for _, id := range stale {
		r.stale[id] = true
	}
	f.cancel()
	f.wg.Wait()
	f.deps.RasterStale, f.deps.RebuildRaster = r.isStale, r.rebuild
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.svc = New()
	f.svc.Bind(ctx, &f.wg, f.deps)
	t.Cleanup(func() { cancel(); f.wg.Wait() })
	return f, r
}

func (f *fixture) runBatch(t *testing.T) {
	t.Helper()
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "fin de file", func() bool { return !f.svc.Status().Running })
}

// Film absent et fichier tactique d'un ancien format : le fichier est refait depuis le rejeu rangé, sans film ni
// reconstruction, et le décompte final ne compte plus rien en retard.
func TestRaster_FilmAbsentEtFichierPerimeEstRefait(t *testing.T) {
	f, r := newRasterFixture(t, "ab000001")
	writeArtifact(t, f.paths, "ab000001", true) // le rejeu est à jour : seul son fichier tactique est périmé
	films, rasters, err := f.svc.scan(context.Background())
	if err != nil || len(films) != 0 || len(rasters) != 1 || rasters[0].MatchID != "ab000001" || !rasters[0].rasterOnly {
		t.Fatalf("scan = films %+v rasters %+v err %v", films, rasters, err)
	}
	f.runBatch(t)
	if got := r.rebuiltIDs(); len(got) != 1 || got[0] != "ab000001" {
		t.Fatalf("fichiers tactiques refaits = %v", got)
	}
	if f.calls.Load() != 0 {
		t.Fatal("aucun rejeu ne doit être reconstruit (pas de film)")
	}
	st := f.svc.Status()
	if st.Done != 1 || st.Total != 1 || st.Failed != 0 {
		t.Fatalf("état final = %+v", st)
	}
	waitFor(t, "décompte final", func() bool { s := f.svc.Status(); return s.Checked && s.Stale == 0 })
}

// Un rejeu rangé en retard de schéma dont le film a expiré a lui aussi son fichier refait ; le rejeu n'est pas touché.
func TestRaster_FilmAbsentRejeuEnRetardFichierRefait(t *testing.T) {
	f, r := newRasterFixture(t, "ab000002")
	writeArtifact(t, f.paths, "ab000002", false)
	f.runBatch(t)
	if got := r.rebuiltIDs(); len(got) != 1 || f.calls.Load() != 0 {
		t.Fatalf("fichiers refaits = %v, reconstructions = %d", got, f.calls.Load())
	}
}

// Film absent et fichier tactique à jour : rien n'est refait ni compté.
func TestRaster_FilmAbsentEtFichierAJourNeFaitRien(t *testing.T) {
	f, r := newRasterFixture(t)
	writeArtifact(t, f.paths, "ab000003", true)
	films, rasters, err := f.svc.scan(context.Background())
	if err != nil || len(films) != 0 || len(rasters) != 0 {
		t.Fatalf("scan = films %+v rasters %+v err %v", films, rasters, err)
	}
	f.runBatch(t)
	if got := r.rebuiltIDs(); len(got) != 0 {
		t.Fatalf("fichiers refaits = %v, attendu aucun", got)
	}
	if st := f.svc.Status(); st.Total != 0 || st.Done != 0 {
		t.Fatalf("état final = %+v", st)
	}
}

// Film présent : le chemin existant est inchangé. Le rejeu en retard se reconstruit, et l'état du fichier tactique n'est
// même pas lu (la reconstruction le refait par ses dérivations).
func TestRaster_FilmPresentCheminExistantInchange(t *testing.T) {
	f, r := newRasterFixture(t, "ab000004")
	writeArtifact(t, f.paths, "ab000004", false)
	f.films["ab000004"] = true
	films, rasters, err := f.svc.scan(context.Background())
	if err != nil || len(films) != 1 || len(rasters) != 0 || films[0].rasterOnly {
		t.Fatalf("scan = films %+v rasters %+v err %v", films, rasters, err)
	}
	f.runBatch(t)
	if f.calls.Load() != 1 || len(r.rebuiltIDs()) != 0 || len(r.askedIDs()) != 0 {
		t.Fatalf("reconstructions = %d, fichiers refaits = %v, états lus = %v",
			f.calls.Load(), r.rebuiltIDs(), r.askedIDs())
	}
	// À jour, film présent : ni film ni fichier tactique.
	f2, r2 := newRasterFixture(t, "ab000005")
	writeArtifact(t, f2.paths, "ab000005", true)
	f2.films["ab000005"] = true
	if films, rasters, _ := f2.svc.scan(context.Background()); len(films)+len(rasters) != 0 || len(r2.askedIDs()) != 0 {
		t.Fatalf("un rejeu à jour dont le film est là ne se liste pas : films %+v rasters %+v", films, rasters)
	}
}

// Les deux listes se mêlent dans une file : les fichiers tactiques d'abord, puis les films, un seul décompte.
func TestRaster_LesDeuxListesDansUneMemeFile(t *testing.T) {
	f, r := newRasterFixture(t, "ab000006", "ab000007")
	writeArtifact(t, f.paths, "ab000006", true)
	writeArtifact(t, f.paths, "ab000007", false)
	writeArtifact(t, f.paths, "ab000008", false)
	f.films["ab000008"] = true
	f.runBatch(t)
	if got := r.rebuiltIDs(); len(got) != 2 {
		t.Fatalf("fichiers refaits = %v", got)
	}
	if f.calls.Load() != 1 {
		t.Fatalf("reconstructions = %d, attendu 1 (le seul film présent)", f.calls.Load())
	}
	if st := f.svc.Status(); st.Done != 3 || st.Total != 3 {
		t.Fatalf("état final = %+v", st)
	}
}

// Un fichier tactique refait entre le décompte et le tour du match se saute ; un échec se compte et reste visible.
func TestRaster_DejaRefaitSeSauteEtUnEchecSeCompte(t *testing.T) {
	f, r := newRasterFixture(t, "ab000009")
	writeArtifact(t, f.paths, "ab000009", true)
	r.err = errors.New("rejeu illisible")
	f.runBatch(t)
	if st := f.svc.Status(); st.Failed != 1 || st.Done != 1 {
		t.Fatalf("état final = %+v, attendu un échec compté", st)
	}
	// Refait entre-temps par un cycle de synchronisation : la file ne le refait pas.
	f2, r2 := newRasterFixture(t, "ab00000a")
	writeArtifact(t, f2.paths, "ab00000a", true)
	films, rasters, err := f2.svc.scan(context.Background())
	if err != nil || len(films) != 0 || len(rasters) != 1 {
		t.Fatal(err)
	}
	r2.mu.Lock()
	r2.stale["ab00000a"] = false
	r2.mu.Unlock()
	f2.svc.processRaster(context.Background(), rasters[0])
	if len(r2.rebuiltIDs()) != 0 {
		t.Fatal("un fichier déjà à jour ne se refait pas")
	}
}

// Sans les deux gestes injectés, un rejeu dont le film est absent n'est pas listé (comportement d'avant).
func TestRaster_SansLesGestesInjectesRienNeSeListe(t *testing.T) {
	f := newFixture(t)
	writeArtifact(t, f.paths, "ab00000b", true)
	films, rasters, err := f.svc.scan(context.Background())
	if err != nil || len(films) != 0 || len(rasters) != 0 {
		t.Fatalf("scan = films %+v rasters %+v err %v", films, rasters, err)
	}
}
