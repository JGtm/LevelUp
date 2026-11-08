package resourcepack

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func oldPackFiles() map[string]string {
	return map[string]string{
		"static/old.png":                                            "OLD",
		"static/gone.png":                                           "GONE",
		"web/titles/halo_5/a.png":                                   "OLD-A",
		"data/titles/halo_infinite/reference/maps.json":             "OLD-MAPS",
		"data/titles/halo_infinite/warehouse/metadata-prebuilt.zip": "OLD-META",
		"ffmpeg.exe":                                                "OLD-FF",
	}
}

func newPackFiles() map[string]string {
	return map[string]string{
		"static/new.png":                                            "NEW",
		"web/titles/halo_5/a.png":                                   "NEW-A",
		"data/titles/halo_infinite/reference/maps.json":             "NEW-MAPS",
		"data/titles/halo_infinite/warehouse/metadata-prebuilt.zip": "NEW-META",
		"ffmpeg.exe":  "NEW-FF",
		"avcodec.dll": "NEW-DLL",
	}
}

// installedRoot : une racine de ressources qui porte le paquet « ancien » ET des fichiers du moteur.
func installedRoot(t *testing.T) string {
	t.Helper()
	root, _ := buildPack(t, oldPackFiles())
	writeTree(t, root, map[string]string{
		"levelup-server.exe": "ENGINE", "web/index.html": "INDEX", "web/assets/app.js": "JS", "config/x.toml": "C",
	})
	return root
}

func stagedNew(t *testing.T, root string) Manifest {
	t.Helper()
	pack, m := buildPack(t, newPackFiles())
	if _, err := StageFromZip(root, zipPack(t, pack, nil), m.Fingerprint); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestExtractRefuseCeQuiNEstPasLeManifeste(t *testing.T) {
	pack, m := buildPack(t, map[string]string{"static/a": "A"})
	root := t.TempDir()
	cases := map[string]func(*zip.Writer){
		"fichier en trop": func(w *zip.Writer) { z, _ := w.Create("static/extra"); _, _ = z.Write([]byte("x")) },
		"chemin qui sort": func(w *zip.Writer) { z, _ := w.Create("../evil"); _, _ = z.Write([]byte("x")) },
	}
	for name, mutate := range cases {
		if _, err := StageFromZip(root, zipPack(t, pack, mutate), m.Fingerprint); err == nil {
			t.Errorf("%s : accepté", name)
		}
		if Staged(root, m.Fingerprint) {
			t.Errorf("%s : paquet marqué prêt malgré l'échec", name)
		}
	}
	// Contenu altéré après la fabrication du manifeste.
	if err := os.WriteFile(filepath.Join(pack, "static", "a"), []byte("TAMPERED"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := StageFromZip(root, zipPack(t, pack, nil), m.Fingerprint); err == nil {
		t.Error("contenu différent du manifeste : accepté")
	}
	// Empreinte attendue différente de celle de l'archive.
	good, gm := buildPack(t, map[string]string{"static/b": "B"})
	if _, err := StageFromZip(root, zipPack(t, good, nil), strings.Repeat("1", 64)); err == nil || gm.Fingerprint == "" {
		t.Error("autre empreinte que celle attendue : acceptée")
	}
}

func TestActivateBasculeLeLotEtLaisseLeMoteur(t *testing.T) {
	root := installedRoot(t)
	m := stagedNew(t, root)
	if !Staged(root, m.Fingerprint) {
		t.Fatal("le paquet devrait être prêt")
	}
	if err := Activate(t.Context(), root, m.Fingerprint); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"levelup-server.exe": "ENGINE", "web/index.html": "INDEX", "web/assets/app.js": "JS", "config/x.toml": "C",
	}
	for k, v := range newPackFiles() {
		want[k] = v
	}
	if got := readTree(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("arbre après bascule :\n%v\nattendu :\n%v", got, want)
	}
	if v := Check(root, m.Fingerprint); v.State != StateMatch {
		t.Fatalf("verdict : %+v", v)
	}
	if _, err := os.Stat(WorkDir(root)); err == nil {
		entries, _ := os.ReadDir(WorkDir(root))
		if len(entries) != 0 {
			t.Fatalf("le dossier de travail devrait être vide : %v", entries)
		}
	}
}

func TestRecoverDefaitUneBasculeInterrompue(t *testing.T) {
	root := installedRoot(t)
	before := readTree(t, root)
	oldM, _ := ReadManifest(root)
	m := stagedNew(t, root)

	// On rejoue Activate jusqu'à deux unités basculées, puis « l'arrêt brutal ».
	next, _ := ReadManifest(incomingDir(root, m.Fingerprint))
	all, in := map[string]struct{}{}, map[string]struct{}{}
	unitsOf(next, in)
	unitsOf(next, all)
	unitsOf(oldM, all)
	j := journal{To: m.Fingerprint, Units: append(sortedKeys(all), ManifestFileName), New: append(sortedKeys(in), ManifestFileName)}
	if err := writeJournal(osOps, root, j); err != nil {
		t.Fatal(err)
	}
	for _, u := range j.Units[:2] {
		if err := newActivation(t.Context(), osOps, root, m.Fingerprint).swapUnit(u); err != nil {
			t.Fatal(err)
		}
	}
	if reflect.DeepEqual(readTree(t, root), before) {
		t.Fatal("la bascule partielle devrait avoir changé l'arbre")
	}
	recovered, err := Recover(t.Context(), root)
	if err != nil || recovered != RecoveryUndone {
		t.Fatalf("Recover : %v %v", recovered, err)
	}
	if got := readTree(t, root); !reflect.DeepEqual(got, before) {
		t.Fatalf("arbre après reprise :\n%v\nattendu :\n%v", got, before)
	}
	if v := Check(root, oldM.Fingerprint); v.State != StateMatch {
		t.Fatalf("l'ancien paquet doit être actif : %+v", v)
	}
	// Le paquet prêt est intact : l'activation complète fonctionne ensuite.
	if err := Activate(t.Context(), root, m.Fingerprint); err != nil {
		t.Fatalf("activation après reprise : %v", err)
	}
	if v := Check(root, m.Fingerprint); v.State != StateMatch {
		t.Fatalf("verdict après activation : %+v", v)
	}
}

func TestActivateSansPaquetPretEtRecoverSansJournal(t *testing.T) {
	root := installedRoot(t)
	if err := Activate(t.Context(), root, strings.Repeat("a", 64)); err == nil {
		t.Fatal("aucun paquet prêt : l'activation doit refuser")
	}
	if ok, err := Recover(t.Context(), root); ok != RecoveryNone || err != nil {
		t.Fatalf("sans journal : %v %v", ok, err)
	}
}

func TestUnitOf(t *testing.T) {
	cases := map[string]string{
		"static/medals/a.png":                                "static",
		"web/titles/halo_5/x.png":                            "web/titles",
		"web/index.html":                                     "web/index.html",
		"data/titles/halo_infinite/reference/a/b.json":       "data/titles/halo_infinite/reference",
		"data/titles/halo_5/warehouse/metadata-prebuilt.zip": "data/titles/halo_5/warehouse/metadata-prebuilt.zip",
		"ffmpeg.exe":                                         "ffmpeg.exe",
	}
	for in, want := range cases {
		if got := unitOf(in); got != want {
			t.Errorf("unitOf(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func TestPurgeGardeLePaquetPret(t *testing.T) {
	root := installedRoot(t)
	m := stagedNew(t, root)
	other := strings.Repeat("c", 64)
	for _, p := range []string{PartPath(root, other), incomingDir(root, other) + "/x"} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Purge(root, m.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if !Staged(root, m.Fingerprint) {
		t.Fatal("le paquet prêt a été purgé")
	}
	if _, err := os.Stat(PartPath(root, other)); err == nil {
		t.Fatal("l'archive partielle étrangère devrait être purgée")
	}
}

func TestSettleActiveLePaquetPretSeulementSiCEstLePaquetDuMoteur(t *testing.T) {
	root := installedRoot(t)
	oldM, _ := ReadManifest(root)
	m := stagedNew(t, root)

	// Un moteur qui attend l'ancien paquet n'active rien.
	out := Settle(t.Context(), root, oldM.Fingerprint)
	if out.Activated || out.Verdict.State != StateMatch {
		t.Fatalf("moteur de l'ancien paquet : %+v", out)
	}
	// Le nouveau moteur attend le paquet prêt : il l'active au démarrage.
	out = Settle(t.Context(), root, m.Fingerprint)
	if !out.Activated || out.Verdict.State != StateMatch || out.ActivateErr != nil {
		t.Fatalf("nouveau moteur : %+v", out)
	}
	// Sans empreinte embarquée : aucun contrôle, aucune bascule.
	if out := Settle(t.Context(), root, ""); out.Verdict.State != StateUnchecked {
		t.Fatalf("sans empreinte : %+v", out)
	}
}

func TestRemoveRetireLePaquetEtLuiSeul(t *testing.T) {
	root := installedRoot(t)
	m := stagedNew(t, root)
	if err := Activate(t.Context(), root, m.Fingerprint); err != nil {
		t.Fatal(err)
	}
	// Un fichier posé par l'utilisateur dans un dossier du paquet empêche seulement la suppression de ce dossier.
	writeTree(t, root, map[string]string{"static/mon-fichier.txt": "A MOI"})
	if err := os.MkdirAll(WorkDir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	n, err := Remove(root)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(newPackFiles()) {
		t.Fatalf("%d fichiers retirés, attendu %d", n, len(newPackFiles()))
	}
	want := map[string]string{
		"levelup-server.exe": "ENGINE", "web/index.html": "INDEX", "web/assets/app.js": "JS", "config/x.toml": "C",
		"static/mon-fichier.txt": "A MOI",
	}
	if got := readTree(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("arbre après retrait :\n%v\nattendu :\n%v", got, want)
	}
	for _, gone := range []string{"web/titles", "data", ".pack", ManifestFileName} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(gone))); err == nil {
			t.Errorf("%s devrait avoir disparu", gone)
		}
	}
	if n, err := Remove(root); n != 0 || err != nil {
		t.Fatalf("second retrait sans manifeste : %d %v", n, err)
	}
}
