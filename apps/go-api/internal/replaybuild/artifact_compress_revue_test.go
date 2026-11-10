package replaybuild

// artifact_compress_revue_test.go — LA CONVERSION FACE AUX ÉTATS INTERROMPUS ET AUX COURSES : doublon identique
// résorbé, doublon différent laissé, suppression qui échoue, serveur qui pose un `.gz` pendant la conversion,
// annulation en cours de parc, date reportée et `.meta` fiable.

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replayartifact"
)

// doublon pose un match aux deux formes : le `.json` brut et le `.json.gz` du contenu `gz`.
func doublon(t *testing.T, dir, court string, brut, gz []byte) {
	t.Helper()
	ecrireTest(t, filepath.Join(dir, court+".json"), brut)
	ecrireTest(t, filepath.Join(dir, court+".json.gz"), gzipTest(t, gz))
}

// TestCompressDir_ResorbeUnDoublonIdentique : le `.gz` décodé est identique au `.json` : le `.json` est supprimé.
func TestCompressDir_ResorbeUnDoublonIdentique(t *testing.T) {
	dir := t.TempDir()
	doc := docJSON(t, "d0000001-1234-4abc-9def-0123456789ab", true)
	doublon(t, dir, "d0000001", doc, doc)
	avant, err := CompressDir(context.Background(), dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if avant.Resolved != 1 || avant.Different != 0 || avant.Converted != 0 {
		t.Fatalf("simulation = %+v", avant)
	}
	if _, err := os.Stat(filepath.Join(dir, "d0000001.json")); err != nil {
		t.Fatalf("la simulation ne supprime rien : %v", err)
	}
	rep, err := CompressDir(context.Background(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep != avant {
		t.Errorf("la simulation chiffre autrement que la résorption : %+v / %+v", avant, rep)
	}
	if _, err := os.Stat(filepath.Join(dir, "d0000001.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("le .json identique doit être supprimé : %v", err)
	}
}

// TestCompressDir_NeTouchePasUnDoublonDifferent : contenus différents : aucun fichier supprimé, compté à part.
func TestCompressDir_NeTouchePasUnDoublonDifferent(t *testing.T) {
	dir := t.TempDir()
	doublon(t, dir, "d0000002", []byte(`{"schemaVersion":1}`), []byte(`{"schemaVersion":2}`))
	rep, err := CompressDir(context.Background(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Different != 1 || rep.Resolved != 0 || rep.Failed != 0 {
		t.Fatalf("compte rendu = %+v", rep)
	}
	for _, n := range []string{"d0000002.json", "d0000002.json.gz"} {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			t.Errorf("%s ne doit pas être touché : %v", n, err)
		}
	}
}

// TestCompressDir_EchecDuRemoveEstARelancerPuisResorbe : un `.json` qui ne se supprime pas (lecteur sous Windows,
// process mort entre la pose et la suppression : même état sur disque) n'est pas une conversion ratée. Le `.gz`
// vérifié est en place, le match est compté « à relancer », et la relance résorbe le doublon.
func TestCompressDir_EchecDuRemoveEstARelancerPuisResorbe(t *testing.T) {
	ctx := context.Background()
	dir, contenus := parcBrut(t, 1)
	orig := retirerLeBrut
	retirerLeBrut = func(string) error { return errors.New("fichier utilisé par un autre processus") }
	t.Cleanup(func() { retirerLeBrut = orig })
	rep, err := CompressDir(ctx, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.ToRetry != 1 || rep.Failed != 0 || rep.Converted != 0 || rep.BytesBefore != 0 {
		t.Fatalf("compte rendu = %+v", rep)
	}
	for court, blob := range contenus {
		if lu, rerr := replayartifact.ReadBytes(filepath.Join(dir, court+".json.gz")); rerr != nil || !bytes.Equal(lu, blob) {
			t.Fatalf("le .json.gz vérifié doit rester en place : %v", rerr)
		}
		if _, serr := os.Stat(filepath.Join(dir, court+".json")); serr != nil {
			t.Fatalf("le .json reste : %v", serr)
		}
	}
	retirerLeBrut = orig
	rep, err = CompressDir(ctx, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Resolved != 1 || rep.ToRetry != 0 || rep.Failed != 0 {
		t.Fatalf("la relance doit résorber le doublon : %+v", rep)
	}
	for court := range contenus {
		if _, serr := os.Stat(filepath.Join(dir, court+".json")); !errors.Is(serr, fs.ErrNotExist) {
			t.Errorf("le .json doit avoir disparu : %v", serr)
		}
	}
}

// TestCompressDir_ServeurQuiPoseUnCompressePendantLaConversion : la conversion lit M.json ; avant sa pose, le serveur
// écrit un M.json.gz plus récent et supprime M.json. Le .gz du serveur n'est JAMAIS écrasé, l'absence du brut au
// Remove est tolérée, et rien n'est compté en échec. Injection, sans délai.
func TestCompressDir_ServeurQuiPoseUnCompressePendantLaConversion(t *testing.T) {
	dir, _ := parcBrut(t, 1)
	court := "a0000000"
	recent := docJSON(t, court+"-1234-4abc-9def-0123456789ab", false)
	var serveurGz []byte
	orig := apresLectureDuBrut
	apresLectureDuBrut = func(brut string) {
		serveurGz = gzipTest(t, recent)
		ecrireTest(t, brut+".gz", serveurGz)
		if err := os.Remove(brut); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { apresLectureDuBrut = orig })
	rep, err := CompressDir(context.Background(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Failed != 0 || rep.ToRetry != 0 {
		t.Fatalf("la course n'est pas un échec : %+v", rep)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, court+".json.gz")); !bytes.Equal(got, serveurGz) {
		t.Fatal("le .json.gz plus récent du serveur a été écrasé par l'ancien contenu")
	}
}

// TestCompressDir_ArretEnCoursDeParc : l'annulation arrive APRÈS le premier fichier ; le premier est converti, les
// suivants restent intacts, l'erreur du contexte remonte.
func TestCompressDir_ArretEnCoursDeParc(t *testing.T) {
	dir, contenus := parcBrut(t, 3)
	ctx, cancel := context.WithCancel(context.Background())
	orig := retirerLeBrut
	retirerLeBrut = func(p string) error {
		cancel() // le premier fichier vient d'être posé et vérifié : on coupe avant le suivant
		return os.Remove(p)
	}
	t.Cleanup(func() { retirerLeBrut = orig })
	rep, err := CompressDir(ctx, dir, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("l'erreur du contexte doit remonter : %v", err)
	}
	if rep.Converted != 1 {
		t.Fatalf("compte rendu = %+v", rep)
	}
	restes := 0
	for court, blob := range contenus {
		if got, rerr := os.ReadFile(filepath.Join(dir, court+".json")); rerr == nil {
			restes++
			if !bytes.Equal(got, blob) {
				t.Errorf("%s : brut modifié", court)
			}
		}
	}
	if restes != 2 {
		t.Errorf("%d bruts intacts, attendu 2", restes)
	}
}

// TestCompressDir_ReporteLaDateEtLeMetaResteFiable : le .gz garde la date du .json (le rattrapage trie sur la date de
// l'artefact) ; le .meta n'est pas plus ancien que lui, et fait foi SANS relire l'artefact.
func TestCompressDir_ReporteLaDateEtLeMetaResteFiable(t *testing.T) {
	ctx := context.Background()
	dir, _ := parcBrut(t, 1)
	brut := filepath.Join(dir, "a0000000.json")
	vieux := time.Now().Add(-72 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(brut, vieux, vieux); err != nil {
		t.Fatal(err)
	}
	if _, err := CompressDir(ctx, dir, false); err != nil {
		t.Fatal(err)
	}
	gz := filepath.Join(dir, "a0000000.json.gz")
	st, err := os.Stat(gz)
	if err != nil {
		t.Fatal(err)
	}
	if !st.ModTime().Equal(vieux) {
		t.Errorf("date du .gz = %v, attendu celle du .json (%v)", st.ModTime(), vieux)
	}
	mst, err := os.Stat(filepath.Join(dir, "a0000000.meta"))
	if err != nil || mst.ModTime().Before(st.ModTime()) {
		t.Fatalf("le .meta ne doit pas être plus ancien que l'artefact : %v", err)
	}
	lectures := observability.LoadCounter(CompteurArtefactsLusEnEntier)
	h, ok := ArtifactHeader(ctx, gz)
	if !ok || h.Bytes != int(st.Size()) || h.SchemaVersion == 0 {
		t.Fatalf("en-tête = %+v, %v", h, ok)
	}
	if observability.LoadCounter(CompteurArtefactsLusEnEntier) != lectures {
		t.Error("le .meta doit faire foi sans relire l'artefact")
	}
}
