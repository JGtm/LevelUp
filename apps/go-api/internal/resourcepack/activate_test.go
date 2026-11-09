package resourcepack

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

var errInjected = errors.New("test: échec injecté")

// faultPlan scénarise les opérations d'écriture d'une bascule : coupure du processus avant la n-ième (elle et toutes
// les suivantes ne s'exécutent pas, comme un processus tué), ou échec persistant de certaines opérations pendant
// que les autres s'exécutent.
type faultPlan struct {
	crashAt int // -1 : aucune coupure
	// failRenamesFrom : à partir du n-ième renommage (0 = tous), les renommages échouent ; -1 : jamais.
	failRenamesFrom int
	// failIf : une opération pour laquelle il rend vrai échoue, les autres s'exécutent.
	failIf  func(kind, path string) bool
	count   int
	renames int
}

func noFault() *faultPlan { return &faultPlan{crashAt: -1, failRenamesFrom: -1} }

func (p *faultPlan) step(kind, path string, run func() error) error {
	n := p.count
	p.count++
	if p.crashAt >= 0 && n >= p.crashAt {
		return errInjected
	}
	if p.failIf != nil && p.failIf(kind, path) {
		return errInjected
	}
	return run()
}

func (p *faultPlan) ops() fileOps {
	return fileOps{
		rename: func(a, b string) error {
			idx := p.renames
			p.renames++
			if p.failRenamesFrom >= 0 && idx >= p.failRenamesFrom {
				return errInjected
			}
			return p.step("rename", a, func() error { return os.Rename(a, b) })
		},
		remove:    func(a string) error { return p.step("remove", a, func() error { return os.Remove(a) }) },
		removeAll: func(a string) error { return p.step("removeAll", a, func() error { return os.RemoveAll(a) }) },
		writeFile: func(a string, data []byte) error {
			return p.step("write", a, func() error { return osOps.writeFile(a, data) })
		},
	}
}

// wantTrees : l'arbre avant la bascule et l'arbre après.
func wantTrees(t *testing.T, root string) (before, after map[string]string) {
	t.Helper()
	before = readTree(t, root)
	after = map[string]string{"levelup-server.exe": "ENGINE", "web/index.html": "INDEX", "web/assets/app.js": "JS", "config/x.toml": "C"}
	for k, v := range newPackFiles() {
		after[k] = v
	}
	return before, after
}

func workDirEntries(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(WorkDir(root))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// TestActivateCoupureAChaqueEtapeLaisseUnPaquetCoherent : le processus meurt avant l'opération n de la bascule, pour
// CHAQUE n de la séquence (journal, renommages, validation, ménage). Au démarrage suivant le paquet est celui d'avant
// ou celui d'après, jamais un mélange, jamais absent, et la reprise mène au paquet du moteur.
func TestActivateCoupureAChaqueEtapeLaisseUnPaquetCoherent(t *testing.T) {
	probe := installedRoot(t)
	pm := stagedNew(t, probe)
	dry := noFault()
	if err := activateWith(t.Context(), dry.ops(), probe, pm.Fingerprint); err != nil {
		t.Fatal(err)
	}
	total := dry.count
	if total < 10 {
		t.Fatalf("séquence trop courte pour être crédible : %d opérations", total)
	}
	for crashAt := 0; crashAt <= total; crashAt++ {
		root := installedRoot(t)
		oldFP := ActiveFingerprint(root)
		m := stagedNew(t, root)
		before, after := wantTrees(t, root)
		plan := noFault()
		plan.crashAt = crashAt
		_ = activateWith(t.Context(), plan.ops(), root, m.Fingerprint) // le processus meurt ici

		// Démarrage suivant, première étape : la reprise seule laisse un paquet entier.
		if _, err := Recover(t.Context(), root); err != nil {
			t.Fatalf("coupure avant l'opération %d : reprise : %v", crashAt, err)
		}
		switch tree := readTree(t, root); {
		case reflect.DeepEqual(tree, before):
			if v := Check(root, oldFP); v.State != StateMatch {
				t.Fatalf("coupure %d : arbre d'avant mais verdict %+v", crashAt, v)
			}
		case reflect.DeepEqual(tree, after):
			if v := Check(root, m.Fingerprint); v.State != StateMatch {
				t.Fatalf("coupure %d : arbre d'après mais verdict %+v", crashAt, v)
			}
		default:
			t.Fatalf("coupure %d : arbre mélangé\n%v", crashAt, tree)
		}
		// Puis le démarrage complet : le paquet du moteur est en place, sans reste.
		out := Settle(t.Context(), root, m.Fingerprint)
		if out.Verdict.State != StateMatch || out.ActivateErr != nil || out.RecoverErr != nil {
			t.Fatalf("coupure %d : démarrage : %+v", crashAt, out)
		}
		if got := readTree(t, root); !reflect.DeepEqual(got, after) {
			t.Fatalf("coupure %d : arbre final\n%v", crashAt, got)
		}
		if left := workDirEntries(t, root); len(left) != 0 {
			t.Fatalf("coupure %d : restes dans le dossier de travail : %v", crashAt, left)
		}
	}
}

// TestEchecDeMenageApresBasculeNEstPasUnEchecDActivation : la bascule est complète ; l'échec d'une suppression
// (ancien paquet, dossier extrait, marqueur, journal) ne la défait pas et ne la déclare pas échouée ; le ménage
// reprend au démarrage suivant.
func TestEchecDeMenageApresBasculeNEstPasUnEchecDActivation(t *testing.T) {
	cases := map[string]func(kind, path string) bool{
		"ancien paquet": func(kind, p string) bool { return kind == "removeAll" && filepath.Base(p) == oldDirName },
		"dossier extrait": func(kind, p string) bool {
			return kind == "removeAll" && strings.HasPrefix(filepath.Base(p), incomingPre)
		},
		"marqueur": func(kind, p string) bool { return kind == "remove" && strings.HasPrefix(filepath.Base(p), readyPre) },
		"journal":  func(kind, p string) bool { return kind == "remove" && filepath.Base(p) == journalName },
	}
	for name, failIf := range cases {
		root := installedRoot(t)
		m := stagedNew(t, root)
		_, after := wantTrees(t, root)
		plan := noFault()
		plan.failIf = failIf
		if err := activateWith(t.Context(), plan.ops(), root, m.Fingerprint); err != nil {
			t.Errorf("%s : la bascule complète est déclarée échouée : %v", name, err)
			continue
		}
		if v := Check(root, m.Fingerprint); v.State != StateMatch {
			t.Errorf("%s : verdict %+v", name, v)
		}
		out := Settle(t.Context(), root, m.Fingerprint)
		if out.Verdict.State != StateMatch || out.ActivateErr != nil || out.RecoverErr != nil {
			t.Errorf("%s : démarrage suivant : %+v", name, out)
		}
		if got := readTree(t, root); !reflect.DeepEqual(got, after) {
			t.Errorf("%s : arbre après le démarrage suivant\n%v", name, got)
		}
		if left := workDirEntries(t, root); len(left) != 0 {
			t.Errorf("%s : le ménage n'a pas été repris : %v", name, left)
		}
	}
}

// errorLogs relève les enregistrements d'erreur du journal par défaut le temps d'un test.
type errorLogs struct {
	mu sync.Mutex
	n  int
}

func (l *errorLogs) Enabled(context.Context, slog.Level) bool { return true }
func (l *errorLogs) Handle(_ context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		l.mu.Lock()
		l.n++
		l.mu.Unlock()
	}
	return nil
}
func (l *errorLogs) WithAttrs([]slog.Attr) slog.Handler { return l }
func (l *errorLogs) WithGroup(string) slog.Handler      { return l }
func (l *errorLogs) count() int                         { l.mu.Lock(); defer l.mu.Unlock(); return l.n }

func captureErrorLogs(t *testing.T) *errorLogs {
	t.Helper()
	l := &errorLogs{}
	prev := slog.Default()
	slog.SetDefault(slog.New(l))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return l
}

// TestRetourArriereIncompletConserveLeJournalEtLogueChaqueErreur : un retour arrière qui n'aboutit pas ne passe pas
// pour fait : chaque erreur est journalisée et le journal reste pour que la reprise défasse le reste.
func TestRetourArriereIncompletConserveLeJournalEtLogueChaqueErreur(t *testing.T) {
	logs := captureErrorLogs(t)
	root := installedRoot(t)
	m := stagedNew(t, root)
	before, _ := wantTrees(t, root)
	oldFP := ActiveFingerprint(root)
	plan := noFault()
	plan.failRenamesFrom = 3 // le disque se verrouille au milieu de la bascule : le retour arrière échoue aussi
	if err := activateWith(t.Context(), plan.ops(), root, m.Fingerprint); err == nil {
		t.Fatal("la bascule coupée en deux devait échouer")
	}
	if _, err := os.Stat(journalPath(root)); err != nil {
		t.Fatalf("retour arrière incomplet : le journal doit rester pour la reprise : %v", err)
	}
	if logs.count() == 0 {
		t.Fatal("aucune erreur de retour arrière journalisée")
	}
	// Une nouvelle bascule sur un journal non résolu est refusée.
	if err := Activate(t.Context(), root, m.Fingerprint); err == nil {
		t.Error("activation acceptée malgré un journal non résolu")
	}
	// Le disque se libère : la reprise défait ce qui reste.
	if recovered, err := Recover(t.Context(), root); err != nil || recovered != RecoveryUndone {
		t.Fatalf("reprise : %v %v", recovered, err)
	}
	if got := readTree(t, root); !reflect.DeepEqual(got, before) {
		t.Fatalf("arbre après reprise\n%v", got)
	}
	if v := Check(root, oldFP); v.State != StateMatch {
		t.Fatalf("verdict %+v", v)
	}
	// Le paquet prêt est intact : le démarrage du nouveau moteur l'active.
	if out := Settle(t.Context(), root, m.Fingerprint); !out.Activated || out.Verdict.State != StateMatch {
		t.Fatalf("démarrage : %+v", out)
	}
}

// TestSettleNActiveRienSurUneRepriseEnEchec : un journal illisible laisse un état incertain, Settle n'y superpose
// pas une bascule.
func TestSettleNActiveRienSurUneRepriseEnEchec(t *testing.T) {
	root := installedRoot(t)
	m := stagedNew(t, root)
	if err := os.WriteFile(journalPath(root), []byte("{pas du json"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := Settle(t.Context(), root, m.Fingerprint)
	if out.RecoverErr == nil || out.Activated || out.ActivateErr != nil {
		t.Fatalf("démarrage : %+v", out)
	}
	if got := readTree(t, root); got["static/new.png"] != "" {
		t.Fatal("une bascule a été faite sur un état incertain")
	}
}

// TestStageFromZipForceSurLeDisqueChaqueFichierLeManifesteEtLeMarqueur : un fichier écrit n'est posé qu'une fois
// forcé sur le disque ; un échec de la force est un échec du rangement.
func TestStageFromZipForceSurLeDisqueChaqueFichierLeManifesteEtLeMarqueur(t *testing.T) {
	var mu sync.Mutex
	var synced []string
	orig := syncFile
	t.Cleanup(func() { syncFile = orig })
	syncFile = func(f *os.File) error {
		mu.Lock()
		synced = append(synced, filepath.ToSlash(f.Name()))
		mu.Unlock()
		return f.Sync()
	}
	root := t.TempDir()
	pack, pm := buildPack(t, newPackFiles())
	if _, err := StageFromZip(root, zipPack(t, pack, nil), pm.Fingerprint); err != nil {
		t.Fatal(err)
	}
	inc := filepath.ToSlash(incomingDir(root, pm.Fingerprint))
	want := []string{filepath.ToSlash(readyMarker(root, pm.Fingerprint)), inc + "/" + ManifestFileName}
	for _, f := range pm.Files {
		want = append(want, inc+"/"+f.Path)
	}
	for _, w := range want {
		found := false
		for _, s := range synced {
			found = found || s == w
		}
		if !found {
			t.Errorf("%s n'a pas été forcé sur le disque", w)
		}
	}

	syncFile = func(*os.File) error { return errInjected }
	root2 := t.TempDir()
	if _, err := StageFromZip(root2, zipPack(t, pack, nil), pm.Fingerprint); err == nil {
		t.Error("un échec de la force sur disque est accepté")
	}
	if Staged(root2, pm.Fingerprint) {
		t.Error("paquet marqué prêt malgré l'échec de la force sur disque")
	}
}
