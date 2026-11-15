package replayregen

// compress_test.go — la compression des rejeux bruts à la fin d'une passe lancée par l'utilisateur : ceux que la file
// n'a pas pu refaire (film absent de l'archive) restent en `.json` brut, et sont compressés par la porte ; jamais au
// démarrage, jamais sans le geste, interruptible, un fichier à la fois.

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replayartifact"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// seedBruts range n artefacts BRUTS (`.json`) dont le film est absent : la file ne peut pas les refaire. current :
// à jour (ni refaits ni en retard) ou en retard. Rend les identifiants et leurs octets.
func (f *fixture) seedBruts(t *testing.T, n int, current bool) (ids []string, contenus map[string][]byte) {
	t.Helper()
	contenus = map[string][]byte{}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%08x", 0xb00+i)
		doc := map[string]any{"matchId": id, "schemaVersion": 1}
		if current {
			doc["schemaVersion"] = replay.SchemaVersion
			doc["layers"] = replay.RevisionsCourantesDesCouches()
		}
		raw, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		_, legacy := f.legacyPaths(id)
		if err := os.MkdirAll(f.paths.ReplayArtifactsDir(testSlug), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(legacy, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
		contenus[id] = raw
	}
	return ids, contenus
}

// legacyPaths rend le chemin compressé et le chemin brut d'un rejeu.
func (f *fixture) legacyPaths(id string) (compressed, legacy string) {
	p := f.paths.ReplayArtifactPath(testSlug, id)
	return p, p[:len(p)-len(".gz")]
}

// bruts rend les identifiants dont le `.json` brut existe encore.
func (f *fixture) bruts(ids []string) []string {
	var out []string
	for _, id := range ids {
		if _, legacy := f.legacyPaths(id); fileExists(legacy) {
			out = append(out, id)
		}
	}
	return out
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

func (f *fixture) attendreFinDeFile(t *testing.T) {
	t.Helper()
	waitFor(t, "fin de file", func() bool { return !f.svc.Status().Running })
}

// TestCompression_FinDUnePasseLancee : les rejeux refaits sont écrits compressés par la reconstruction ; les bruts
// que la file n'a pas pu refaire (film absent) sont compressés à la fin, par la porte, contenu identique.
func TestCompression_FinDUnePasseLancee(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 2)
	ids, contenus := f.seedBruts(t, 3, false)
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	f.attendreFinDeFile(t)
	st := f.svc.Status()
	if st.Done != 2 || st.Total != 2 || st.Compressed != 3 || st.CompressFailed != 0 || st.Compressing {
		t.Fatalf("état final = %+v", st)
	}
	if left := f.bruts(ids); len(left) != 0 {
		t.Fatalf("des bruts restent : %v", left)
	}
	for id, brut := range contenus {
		compressed, _ := f.legacyPaths(id)
		got, err := replayartifact.ReadBytes(compressed)
		if err != nil || !bytes.Equal(got, brut) {
			t.Errorf("%s : contenu relu différent (%v)", id, err)
		}
	}
}

// TestCompression_AussiLesBrutsAJour : un brut à jour (ni refait ni à refaire) est compressé de même.
func TestCompression_AussiLesBrutsAJour(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 1)
	ids, _ := f.seedBruts(t, 2, true)
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	f.attendreFinDeFile(t)
	if st := f.svc.Status(); st.Compressed != 2 || len(f.bruts(ids)) != 0 {
		t.Fatalf("état final = %+v, bruts restants %v", st, f.bruts(ids))
	}
}

// TestCompression_JamaisSansLeGeste : ni le démarrage du serveur (liaison, reprise sans drapeau, décompte), ni la
// lecture de l'état ne compressent quoi que ce soit.
func TestCompression_JamaisSansLeGeste(t *testing.T) {
	f := newFixture(t)
	ids, _ := f.seedBruts(t, 3, false)
	f.svc.ResumeIfPending() // sans drapeau : ne fait rien
	if _, err := f.svc.Rescan(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "décompte", func() bool { return f.svc.Status().Checked })
	time.Sleep(60 * time.Millisecond)
	if left := f.bruts(ids); len(left) != 3 {
		t.Fatalf("des bruts ont été compressés sans geste : %v restants", left)
	}
	if st := f.svc.Status(); st.Compressed != 0 || st.Compressing {
		t.Fatalf("état = %+v", st)
	}
}

// TestCompression_RepriseDUnePasseEnAttente : la reprise d'une passe lancée avant l'arrêt (drapeau posé) découle du
// geste : elle compresse à sa fin.
func TestCompression_RepriseDUnePasseEnAttente(t *testing.T) {
	f := newFixture(t)
	ids, _ := f.seedBruts(t, 2, false)
	if err := f.pending.SetReplayRegenPending(true); err != nil {
		t.Fatal(err)
	}
	f.svc.ResumeIfPending()
	waitFor(t, "compression reprise", func() bool { return len(f.bruts(ids)) == 0 && !f.svc.Status().Running })
	if st := f.svc.Status(); st.Compressed != 2 {
		t.Fatalf("état = %+v", st)
	}
	if f.pending.get() {
		t.Fatal("la fin de passe efface le drapeau")
	}
}

// TestCompression_Idempotente : une seconde passe ne touche plus rien ; ses compteurs repartent de zéro.
func TestCompression_Idempotente(t *testing.T) {
	f := newFixture(t)
	f.seedBruts(t, 2, false)
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	f.attendreFinDeFile(t)
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	f.attendreFinDeFile(t)
	if st := f.svc.Status(); st.Compressed != 0 || st.CompressFailed != 0 {
		t.Fatalf("seconde passe : %+v", st)
	}
}

// TestCompression_ChaqueEchecEstCompte : un brut illisible reste en place et se compte à part ; les autres sont
// compressés ; les échecs de refonte (Failed) et de compression ne se mélangent pas.
func TestCompression_ChaqueEchecEstCompte(t *testing.T) {
	f := newFixture(t)
	ids, _ := f.seedBruts(t, 3, false)
	_, casse := f.legacyPaths(ids[1])
	var tronque bytes.Buffer
	zw := gzip.NewWriter(&tronque)
	_, _ = zw.Write(bytes.Repeat([]byte(`{"a":1}`), 200))
	_ = zw.Close()
	if err := os.WriteFile(casse, tronque.Bytes()[:tronque.Len()/2], 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	f.attendreFinDeFile(t)
	st := f.svc.Status()
	if st.Compressed != 2 || st.CompressFailed != 1 || st.Failed != 0 {
		t.Fatalf("état final = %+v", st)
	}
	if left := f.bruts(ids); len(left) != 1 || left[0] != ids[1] {
		t.Fatalf("seul le fichier en échec doit rester brut : %v", left)
	}
}

// TestCompression_PauseEntreDeuxFichiers : la pause tient la compression entre deux fichiers (le fichier en cours va à
// son terme) ; la reprise la termine.
func TestCompression_PauseEntreDeuxFichiers(t *testing.T) {
	f := newFixture(t)
	ids, _ := f.seedBruts(t, 4, false)
	var paused atomic.Bool
	f.heavyHook = func() {
		if st := f.svc.Status(); st.Compressing && st.Compressed == 1 && paused.CompareAndSwap(false, true) {
			if _, err := f.svc.Pause(); err != nil {
				t.Error(err)
			}
		}
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "pause en cours de compression", func() bool { return f.svc.Status().Paused })
	// Le deuxième fichier va à son terme (sous charge, après la pause) ; ensuite plus rien ne part.
	waitFor(t, "deuxième fichier converti", func() bool { return f.svc.Status().Compressed == 2 })
	time.Sleep(60 * time.Millisecond)
	// La pause tombe pendant la consultation de la porte du deuxième fichier : celui-ci va à son terme, le troisième
	// attend.
	if st := f.svc.Status(); st.Compressed != 2 || !st.Running || !st.Compressing {
		t.Fatalf("la compression doit attendre en pause : %+v", st)
	}
	if left := f.bruts(ids); len(left) != 2 {
		t.Fatalf("bruts restants = %d, attendu 2", len(left))
	}
	if _, err := f.svc.Resume(); err != nil {
		t.Fatal(err)
	}
	f.attendreFinDeFile(t)
	if st := f.svc.Status(); st.Compressed != 4 {
		t.Fatalf("état final = %+v", st)
	}
}

// TestCompression_ModeJeuEtSynchroCedentLaPlace : pendant une partie, ou un cycle de synchronisation (qui cuit), aucun
// fichier n'est converti.
func TestCompression_ModeJeuEtSynchroCedentLaPlace(t *testing.T) {
	for _, cas := range []string{"partie", "synchro"} {
		t.Run(cas, func(t *testing.T) {
			f := newFixture(t)
			ids, _ := f.seedBruts(t, 2, false)
			if cas == "partie" {
				f.heavy.Store(false)
			} else {
				f.syncing.Store(true)
			}
			if _, err := f.svc.Start(); err != nil {
				t.Fatal(err)
			}
			waitFor(t, "compression en attente", func() bool { return f.svc.Status().Compressing })
			time.Sleep(60 * time.Millisecond)
			if left := f.bruts(ids); len(left) != 2 {
				t.Fatalf("des bruts ont été convertis malgré l'attente : %v restants", left)
			}
			f.heavy.Store(true)
			f.syncing.Store(false)
			f.attendreFinDeFile(t)
			if st := f.svc.Status(); st.Compressed != 2 {
				t.Fatalf("état final = %+v", st)
			}
		})
	}
}

// TestCompression_ArretDuServeurSansPerteEtReprise : un arrêt en cours de compression laisse les bruts restants intacts
// et le drapeau posé ; la reprise suivante termine, sans rien refaire.
func TestCompression_ArretDuServeurSansPerteEtReprise(t *testing.T) {
	f := newFixture(t)
	ids, contenus := f.seedBruts(t, 4, false)
	var arret atomic.Bool
	f.heavyHook = func() {
		if st := f.svc.Status(); st.Compressing && st.Compressed == 1 && arret.CompareAndSwap(false, true) {
			f.cancel()
		}
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "file arrêtée", func() bool { return !f.svc.Status().Running })
	if !f.pending.get() {
		t.Fatal("un arrêt laisse le drapeau posé")
	}
	restes := f.bruts(ids)
	if len(restes) != 2 { // l'arrêt tombe pendant la porte du deuxième fichier : il va à son terme, pas le suivant
		t.Fatalf("bruts restants = %d, attendu 2", len(restes))
	}
	for _, id := range restes {
		_, legacy := f.legacyPaths(id)
		if got, _ := os.ReadFile(legacy); !bytes.Equal(got, contenus[id]) {
			t.Errorf("%s : brut modifié", id)
		}
	}
	// Redémarrage : un nouveau service sur les mêmes dossiers reprend (drapeau posé) et termine.
	f.wg.Wait()
	f.heavyHook = nil
	g := New()
	var wg2 sync.WaitGroup
	g.Bind(context.Background(), &wg2, f.deps)
	g.ResumeIfPending()
	waitFor(t, "reprise terminée", func() bool { return len(f.bruts(ids)) == 0 && !g.Status().Running })
	wg2.Wait()
	if st := g.Status(); st.Compressed != 2 {
		t.Fatalf("la reprise ne compresse que le reste : %+v", st)
	}
}

// TestCompression_DossierAbsentNEstPasUneErreur : un titre sans dossier de rejeux est sauté sans bruit.
func TestCompression_DossierAbsentNEstPasUneErreur(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	f.attendreFinDeFile(t)
	if st := f.svc.Status(); st.Compressed != 0 || st.CompressFailed != 0 {
		t.Fatalf("état = %+v", st)
	}
	if _, err := os.Stat(f.paths.ReplayArtifactsDir(testSlug)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("la compression ne doit pas créer le dossier : %v", err)
	}
}

// TestCompression_Compteurs : les compteurs expvar suivent le compte rendu.
func TestCompression_Compteurs(t *testing.T) {
	f := newFixture(t)
	f.seedBruts(t, 2, false)
	avant := observability.LoadCounter(MetricCompressed)
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	f.attendreFinDeFile(t)
	if got := observability.LoadCounter(MetricCompressed); got != avant+2 {
		t.Errorf("%s = %d, attendu %d", MetricCompressed, got, avant+2)
	}
}

// TestDecompte_ExposeLesBrutsSansLireLeurContenu : le décompte donne le nombre de rejeux bruts restants par le listage de
// la porte (les noms), sans ouvrir un seul fichier : des octets illisibles se comptent pareil. Un match aux deux formes
// n'est pas un brut à compresser (la conversion le juge à part).
func TestDecompte_ExposeLesBrutsSansLireLeurContenu(t *testing.T) {
	f := newFixture(t)
	ids, _ := f.seedBruts(t, 3, false)
	for _, id := range ids { // contenu inexploitable : seul le nom compte
		_, legacy := f.legacyPaths(id)
		if err := os.WriteFile(legacy, []byte("pas du json"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	f.films[ids[0]] = true // film présent mais rejeu illisible : jamais lu par le décompte des bruts
	doublonID := "0000d0b1"
	writeArtifact(t, f.paths, doublonID, true)
	_, legacy := f.legacyPaths(doublonID)
	if err := os.WriteFile(legacy, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Rescan(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "décompte", func() bool { return f.svc.Status().Checked })
	if st := f.svc.Status(); st.Raw != 3 {
		t.Fatalf("bruts comptés = %d, attendu 3 : %+v", st.Raw, st)
	}
	if len(f.bruts(ids)) != 3 {
		t.Fatal("le décompte ne touche à aucun fichier")
	}
}

// TestPasseDeCompressionSeule : sans aucun rejeu refaisable mais des bruts, la passe lancée par l'utilisateur ne fait que
// compresser ; le décompte final dit ce qu'il reste (zéro).
func TestPasseDeCompressionSeule(t *testing.T) {
	f := newFixture(t)
	ids, _ := f.seedBruts(t, 3, false)
	if _, err := f.svc.Rescan(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "décompte", func() bool { return f.svc.Status().Checked })
	if st := f.svc.Status(); st.Stale != 0 || st.Raw != 3 {
		t.Fatalf("avant la passe : %+v", st)
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	f.attendreFinDeFile(t)
	waitFor(t, "décompte final", func() bool { s := f.svc.Status(); return s.Checked && s.Raw == 0 })
	st := f.svc.Status()
	if st.Compressed != 3 || f.calls.Load() != 0 || len(f.bruts(ids)) != 0 {
		t.Fatalf("état = %+v, reconstructions = %d", st, f.calls.Load())
	}
}

// TestDecompteFinal_LesBrutsEnEchecRestentComptes : un brut que la compression n'a pas pu convertir reste compté.
func TestDecompteFinal_LesBrutsEnEchecRestentComptes(t *testing.T) {
	f := newFixture(t)
	ids, _ := f.seedBruts(t, 2, false)
	_, casse := f.legacyPaths(ids[0])
	var tronque bytes.Buffer
	zw := gzip.NewWriter(&tronque)
	_, _ = zw.Write(bytes.Repeat([]byte(`{"a":1}`), 200))
	_ = zw.Close()
	if err := os.WriteFile(casse, tronque.Bytes()[:tronque.Len()/2], 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	f.attendreFinDeFile(t)
	waitFor(t, "décompte final", func() bool { return f.svc.Status().Checked })
	if st := f.svc.Status(); st.Raw != 1 || st.CompressFailed != 1 {
		t.Fatalf("état = %+v", st)
	}
}
