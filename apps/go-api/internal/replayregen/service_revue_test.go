package replayregen

// service_revue_test.go — les gardes de la revue adversariale du lot L22b (2026-11-03) : portes d'un nouvel essai,
// file finie avant son décompte, décompte périmé jeté, file qui cède à la synchronisation, décompte sans lecture
// des artefacts, artefact produit par une version plus récente.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

func TestStatus_NeLanceJamaisDeDecompte(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 3)
	for range 20 {
		f.svc.Status()
	}
	time.Sleep(60 * time.Millisecond)
	if n := f.presentN.Load(); n != 0 {
		t.Fatalf("FilmPresent appelé %d fois : Status ne lance jamais de décompte", n)
	}
	if f.svc.Status().Checked {
		t.Fatal("aucun décompte n'a été fait")
	}
}

func TestRescan_AttendLaPorteDuModeJeu(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 2)
	f.heavy.Store(false)
	if _, err := f.svc.Rescan(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	if f.presentN.Load() != 0 {
		t.Fatal("aucun artefact n'est lu pendant une partie")
	}
	f.heavy.Store(true)
	waitFor(t, "décompte", func() bool { s := f.svc.Status(); return s.Checked && s.Stale == 2 })
}

func TestRescan_ServiceNonRelie(t *testing.T) {
	if _, err := New().Rescan(); !errors.Is(err, ErrNotBound) {
		t.Fatalf("décompte sans liaison : %v", err)
	}
}

// TestScan_FilmAbsentNeLitAucunArtefact : le test le moins cher (le film est-il dans l'archive) passe AVANT toute
// lecture d'artefact.
func TestScan_FilmAbsentNeLitAucunArtefact(t *testing.T) {
	f := newFixture(t)
	for i := range 5 {
		writeArtifact(t, f.paths, fmt.Sprintf("%08x", 0x900+i), false) // film absent de f.films
	}
	avant := observability.LoadCounter(replaybuild.CompteurOctetsLusEntetes)
	cands, err := f.svc.scan(context.Background())
	if err != nil || len(cands) != 0 {
		t.Fatalf("candidats = %v, %v", cands, err)
	}
	if lus := observability.LoadCounter(replaybuild.CompteurOctetsLusEntetes) - avant; lus != 0 {
		t.Fatalf("octets lus = %d, attendu 0 : un film absent écarte l'artefact avant sa lecture", lus)
	}
}

// TestScan_DeuxMilleArtefacts_OctetsLus : MESURE. 2 000 artefacts synthétiques de 16 Kio ; le premier décompte lit
// chaque artefact une fois (et écrit son en-tête), le suivant ne lit que les petits fichiers d'en-tête.
func TestScan_DeuxMilleArtefacts_OctetsLus(t *testing.T) {
	f := newFixture(t)
	const n = 2000
	pad := make([]byte, 16<<10)
	for i := range pad {
		pad[i] = 'x'
	}
	var total int64
	for i := range n {
		id := fmt.Sprintf("%08x", 0x10000+i)
		raw, _ := json.Marshal(map[string]any{"matchId": id, "schemaVersion": 1, "pad": string(pad)})
		p := f.paths.ReplayArtifactPath(testSlug, id)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		f.films[id] = true
		total += int64(len(raw))
	}
	octets := func() int64 { return observability.LoadCounter(replaybuild.CompteurOctetsLusEntetes) }

	avant := octets()
	cands, err := f.svc.scan(context.Background())
	if err != nil || len(cands) != n {
		t.Fatalf("premier décompte : %d candidats, %v", len(cands), err)
	}
	if lus := octets() - avant; lus < total {
		t.Fatalf("premier décompte : %d octets lus, attendu au moins %d (lecture entière, une fois)", lus, total)
	}

	avant = octets()
	cands, err = f.svc.scan(context.Background())
	if err != nil || len(cands) != n {
		t.Fatalf("second décompte : %d candidats, %v", len(cands), err)
	}
	lus := octets() - avant
	t.Logf("second décompte : %d octets lus pour %d artefacts (%d octets au total)", lus, n, total)
	if lus*50 > total { // moins de 2 % des octets des artefacts
		t.Fatalf("second décompte : %d octets lus sur %d : seuls les en-têtes doivent se lire", lus, total)
	}
}

// TestOccupe_PuisModeJeu : un nouvel essai d'un film occupé repasse par la porte du mode jeu.
func TestOccupe_PuisModeJeu(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 1)
	var tries atomic.Int32
	f.rebuild = func(_ context.Context, _, id string) error {
		if tries.Add(1) == 1 {
			f.heavy.Store(false) // la partie démarre pendant l'attente
			return fmt.Errorf("décodage en cours: %w", ErrBusy)
		}
		writeArtifact(t, f.paths, id, true)
		return nil
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "premier essai", func() bool { return tries.Load() == 1 })
	time.Sleep(120 * time.Millisecond) // l'attente d'occupation (5 ms) est largement écoulée
	if tries.Load() != 1 {
		t.Fatalf("essais = %d : aucun nouvel essai ne part pendant une partie", tries.Load())
	}
	f.heavy.Store(true)
	waitFor(t, "fin", func() bool { return !f.svc.Status().Running })
	if st := f.svc.Status(); st.Done != 1 || st.Failed != 0 {
		t.Fatalf("état = %+v", st)
	}
}

// TestOccupe_PuisPause : un nouvel essai d'un film occupé respecte la pause demandée entre-temps.
func TestOccupe_PuisPause(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 1)
	var tries atomic.Int32
	f.rebuild = func(_ context.Context, _, id string) error {
		if tries.Add(1) == 1 {
			if _, err := f.svc.Pause(); err != nil {
				t.Error(err)
			}
			return fmt.Errorf("décodage en cours: %w", ErrBusy)
		}
		writeArtifact(t, f.paths, id, true)
		return nil
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "premier essai", func() bool { return tries.Load() == 1 })
	time.Sleep(120 * time.Millisecond)
	if tries.Load() != 1 {
		t.Fatalf("essais = %d : aucun nouvel essai ne part en pause", tries.Load())
	}
	if _, err := f.svc.Resume(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "fin", func() bool { return !f.svc.Status().Running })
	if tries.Load() != 2 {
		t.Fatalf("essais = %d, attendu 2", tries.Load())
	}
}

// TestFinish_PauseDuDernierFilm_ReprendreRefuse : une pause pendant le dernier film, puis la fin de la file : la file
// est finie dès que le dernier film l'est, même si son décompte final tourne encore. Reprendre est refusé (409) et ne
// repose pas le drapeau de reprise.
func TestFinish_PauseDuDernierFilm_ReprendreRefuse(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 1)
	gate := make(chan struct{})
	f.rebuild = func(_ context.Context, _, id string) error {
		<-gate
		writeArtifact(t, f.paths, id, true)
		return nil
	}
	var armed atomic.Bool
	reached, release := make(chan struct{}, 4), make(chan struct{})
	f.filmHook = func(string) {
		if armed.Load() {
			reached <- struct{}{}
			<-release
		}
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "film en cours", func() bool { return f.inFly.Load() == 1 })
	armed.Store(true) // le décompte final (et lui seul) se tient en vol
	if _, err := f.svc.Pause(); err != nil {
		t.Fatal(err)
	}
	close(gate)
	select {
	case <-reached:
	case <-time.After(5 * time.Second):
		t.Fatal("le décompte final n'a pas démarré")
	}
	if _, err := f.svc.Resume(); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("reprise d'une file finie : %v, attendu ErrNotRunning", err)
	}
	if f.pending.get() {
		t.Fatal("une reprise après la fin ne repose pas le drapeau")
	}
	if st := f.svc.Status(); st.Running || st.Paused || st.Checked {
		t.Fatalf("état pendant le décompte final = %+v : file arrêtée, décompte pas encore connu", st)
	}
	close(release)
	waitFor(t, "décompte final", func() bool { s := f.svc.Status(); return s.Checked && s.Stale == 0 })
}

// TestDecompte_PerimeNEcraseJamaisLeFinal : un décompte de fond lancé AVANT la file, qui finit APRÈS son décompte
// final, est jeté.
func TestDecompte_PerimeNEcraseJamaisLeFinal(t *testing.T) {
	f := newFixture(t)
	ids := f.seed(t, 2) // ReadDir rend les noms triés : ids[0] puis ids[1]
	var once atomic.Bool
	reached, release := make(chan struct{}), make(chan struct{})
	f.filmHook = func(id string) {
		if id == ids[1] && once.CompareAndSwap(false, true) {
			close(reached)
			<-release
		}
	}
	if _, err := f.svc.Rescan(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-reached: // ids[0] est compté « en retard », le décompte attend sur ids[1]
	case <-time.After(5 * time.Second):
		t.Fatal("le décompte de fond n'a pas démarré")
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "file finie", func() bool { s := f.svc.Status(); return !s.Running && s.Checked && s.Stale == 0 })
	close(release)
	waitFor(t, "décompte de fond fini", func() bool {
		f.svc.mu.Lock()
		defer f.svc.mu.Unlock()
		return !f.svc.scanning
	})
	if st := f.svc.Status(); st.Stale != 0 || !st.Checked {
		t.Fatalf("état = %+v : le décompte d'avant la file ne doit pas écraser le décompte final", st)
	}
}

// TestSynchro_LaFileCedeLaPlace : un cycle de synchronisation qui démarre entre deux films retient le suivant ; la
// file reprend à la fin du cycle, sans jamais décoder deux films en parallèle.
func TestSynchro_LaFileCedeLaPlace(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 2)
	f.rebuild = func(_ context.Context, _, id string) error {
		f.syncing.Store(true) // un cycle démarre pendant ce film
		writeArtifact(t, f.paths, id, true)
		return nil
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "premier film", func() bool { return f.svc.Status().Done == 1 })
	time.Sleep(100 * time.Millisecond)
	if f.calls.Load() != 1 {
		t.Fatalf("reconstructions = %d : le film suivant attend la fin du cycle", f.calls.Load())
	}
	f.syncing.Store(false)
	waitFor(t, "fin", func() bool { return !f.svc.Status().Running })
	if f.calls.Load() != 2 || f.maxFly.Load() != 1 {
		t.Fatalf("reconstructions = %d, parallèles = %d", f.calls.Load(), f.maxFly.Load())
	}
}

// writeStaleSameSchema range un artefact au schéma COURANT dont une révision de couche est périmée, et son fichier
// d'en-tête qui dit quelle version de l'application l'a produit.
func writeStaleSameSchema(t *testing.T, f *fixture, id, producer string) {
	t.Helper()
	layers := map[string]string{}
	for fam, rev := range replay.RevisionsCourantesDesCouches() {
		layers[fam] = rev
	}
	for fam := range layers {
		if fam != "publication" {
			layers[fam] = fam + "-ancienne"
			break
		}
	}
	raw, _ := json.Marshal(map[string]any{"matchId": id, "schemaVersion": replay.SchemaVersion, "layers": layers})
	p := f.paths.ReplayArtifactPath(testSlug, id)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(map[string]any{"matchId": id, "schemaVersion": replay.SchemaVersion, "layers": layers,
		"bytes": len(raw), "appVersion": producer})
	if err := os.WriteFile(replaybuild.MetaPath(p), meta, 0o644); err != nil {
		t.Fatal(err)
	}
	f.films[id] = true
}

// TestScan_ArtefactProduitParUneVersionPlusRecente : après un retour à une version antérieure, un artefact de même
// schéma mais aux révisions de couche plus récentes (produit par la version suivante) n'est pas « en retard » ; un
// artefact produit par une version plus ancienne, ou de producteur inconnu, l'est.
func TestScan_ArtefactProduitParUneVersionPlusRecente(t *testing.T) {
	replaybuild.SetAppVersion("8.1.0")
	t.Cleanup(func() { replaybuild.SetAppVersion("") })
	f := newFixture(t)
	writeStaleSameSchema(t, f, "aa000001", "8.2.0") // plus récent que le binaire : jamais en retard
	writeStaleSameSchema(t, f, "aa000002", "8.0.9") // plus ancien
	writeStaleSameSchema(t, f, "aa000003", "")      // inconnu : admissible
	writeStaleSameSchema(t, f, "aa000004", "8.1.0") // même version
	cands, err := f.svc.scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, c := range cands {
		got[c.MatchID] = true
	}
	if got["aa000001"] || !got["aa000002"] || !got["aa000003"] || !got["aa000004"] {
		t.Fatalf("candidats = %v : seul l'artefact d'une version plus récente que le binaire est écarté", got)
	}
}
