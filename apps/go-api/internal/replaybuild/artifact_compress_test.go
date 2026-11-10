package replaybuild

// artifact_compress_test.go — LA CONVERSION D'UN PARC BRUT EN PARC COMPRESSÉ : sur place, vérifiée avant
// toute suppression, idempotente, à blanc possible, avec un compte rendu chiffré.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/replayartifact"
)

// parcBrut pose n artefacts bruts `{short8}.json` valides dans un dossier neuf et rend leurs contenus.
func parcBrut(t *testing.T, n int) (dir string, contenus map[string][]byte) {
	t.Helper()
	dir = t.TempDir()
	contenus = map[string][]byte{}
	for i := 0; i < n; i++ {
		court := fmt.Sprintf("%08x", 0xa0000000+i)
		blob := docJSON(t, court+"-1234-4abc-9def-0123456789ab", i%2 == 0)
		contenus[court] = blob
		ecrireTest(t, filepath.Join(dir, court+".json"), blob)
	}
	return dir, contenus
}

func TestCompressDir_ConvertitSurPlaceEtVerifie(t *testing.T) {
	ctx := context.Background()
	dir, contenus := parcBrut(t, 3)
	// Un artefact déjà compressé, qui ne doit pas être touché.
	dejaBlob := docJSON(t, "b0000001-1234-4abc-9def-0123456789ab", true)
	dejaGz := filepath.Join(dir, "b0000001.json.gz")
	ecrireTest(t, dejaGz, gzipTest(t, dejaBlob))
	dejaAvant, _ := os.ReadFile(dejaGz)
	// Un fichier qui n'est pas un artefact : ni marque, ni sauvegarde ne se convertissent.
	ecrireTest(t, filepath.Join(dir, "a0000000.json.ancien-11juillet"), []byte("x"))
	// La marque de dérivation de a0000000 est à jour pour le brut : elle doit le rester pour le compressé.
	brut0 := filepath.Join(dir, "a0000000.json")
	st0, _ := os.Stat(brut0)
	if err := WriteDerivationsMark(brut0, 1, int(st0.Size())); err != nil {
		t.Fatal(err)
	}
	// Celle de a0000001 est périmée (autre taille) : elle ne doit pas être « réparée » à tort.
	brut1 := filepath.Join(dir, "a0000001.json")
	if err := WriteDerivationsMark(brut1, 1, 3); err != nil {
		t.Fatal(err)
	}

	rep, err := CompressDir(ctx, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Converted != 3 || rep.AlreadyCompressed != 1 || rep.Failed != 0 || rep.Duplicates != 0 {
		t.Fatalf("compte rendu = %+v", rep)
	}
	var avant, apres int64
	for court, blob := range contenus {
		gz := filepath.Join(dir, court+".json.gz")
		st, serr := os.Stat(gz)
		if serr != nil {
			t.Fatalf("%s : %v", court, serr)
		}
		apres += st.Size()
		avant += int64(len(blob))
		if _, serr := os.Stat(filepath.Join(dir, court+".json")); !errors.Is(serr, fs.ErrNotExist) {
			t.Errorf("%s : le .json doit être supprimé : %v", court, serr)
		}
		lu, rerr := replayartifact.ReadBytes(gz)
		if rerr != nil || !bytes.Equal(lu, blob) {
			t.Errorf("%s : contenu décompressé différent (%v)", court, rerr)
		}
		h, ok := ArtifactHeader(ctx, gz)
		if !ok || h.Bytes != int(st.Size()) {
			t.Errorf("%s : en-tête = %+v, %v (taille stockée %d)", court, h, ok, st.Size())
		}
	}
	if rep.BytesBefore != avant || rep.BytesAfter != apres {
		t.Errorf("octets avant/après = %d/%d, attendu %d/%d", rep.BytesBefore, rep.BytesAfter, avant, apres)
	}
	if apres >= avant {
		t.Errorf("la conversion n'a rien gagné : %d -> %d", avant, apres)
	}
	if got, _ := os.ReadFile(dejaGz); !bytes.Equal(got, dejaAvant) {
		t.Error("un artefact déjà compressé ne doit pas être réécrit")
	}
	if !DerivationsUpToDate(filepath.Join(dir, "a0000000.json.gz")) {
		t.Error("la marque à jour doit suivre la nouvelle taille stockée")
	}
	if DerivationsUpToDate(filepath.Join(dir, "a0000001.json.gz")) {
		t.Error("une marque périmée ne doit pas devenir à jour")
	}
	if _, err := os.Stat(filepath.Join(dir, "a0000000.json.ancien-11juillet")); err != nil {
		t.Errorf("la sauvegarde manuelle ne se touche pas : %v", err)
	}
}

func TestCompressDir_Idempotente(t *testing.T) {
	ctx := context.Background()
	dir, _ := parcBrut(t, 2)
	if _, err := CompressDir(ctx, dir, false); err != nil {
		t.Fatal(err)
	}
	rep, err := CompressDir(ctx, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Converted != 0 || rep.AlreadyCompressed != 2 || rep.Failed != 0 || rep.BytesBefore != 0 || rep.BytesAfter != 0 {
		t.Fatalf("seconde passe = %+v", rep)
	}
}

func TestCompressDir_ABlancNEcritRienEtChiffreCommeLaConversion(t *testing.T) {
	ctx := context.Background()
	dir, _ := parcBrut(t, 3)
	avant := listeNoms(t, dir)
	blanc, err := CompressDir(ctx, dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if apres := listeNoms(t, dir); fmt.Sprint(apres) != fmt.Sprint(avant) {
		t.Fatalf("la simulation a modifié le dossier : %v -> %v", avant, apres)
	}
	reel, err := CompressDir(ctx, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if blanc != reel {
		t.Errorf("la simulation chiffre autrement que la conversion :\n  à blanc %+v\n  réel    %+v", blanc, reel)
	}
}

func listeNoms(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

// TestCompressDir_FichierIllisibleEstUnEchecEtResteEnPlace : un brut qui ne se décode pas est compté, journalisé,
// et laissé intact ; les autres se convertissent.
func TestCompressDir_FichierIllisibleEstUnEchecEtResteEnPlace(t *testing.T) {
	ctx := context.Background()
	dir, _ := parcBrut(t, 2)
	casse := filepath.Join(dir, "c0000001.json")
	tronque := gzipTest(t, docJSON(t, "c0000001-1234-4abc-9def-0123456789ab", true))
	ecrireTest(t, casse, tronque[:len(tronque)/2]) // octets magiques gzip, flux tronqué
	rep, err := CompressDir(ctx, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Converted != 2 || rep.Failed != 1 {
		t.Fatalf("compte rendu = %+v", rep)
	}
	if _, err := os.Stat(casse); err != nil {
		t.Errorf("le fichier en échec doit rester en place : %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "c0000001.json.gz")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("aucun .json.gz ne doit être posé pour un fichier en échec : %v", err)
	}
}

// TestCompressDir_NeSupprimeLeBrutQueSiLeCompresseSeRelitIdentique : une relecture qui rend autre chose que le JSON
// d'origine garde le .json et retire le .json.gz posé.
func TestCompressDir_NeSupprimeLeBrutQueSiLeCompresseSeRelitIdentique(t *testing.T) {
	ctx := context.Background()
	dir, contenus := parcBrut(t, 1)
	orig := relireCompresse
	relireCompresse = func(string) ([]byte, error) { return []byte(`{"corrompu":true}`), nil }
	t.Cleanup(func() { relireCompresse = orig })
	rep, err := CompressDir(ctx, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Converted != 0 || rep.Failed != 1 {
		t.Fatalf("compte rendu = %+v", rep)
	}
	for court, blob := range contenus {
		if got, _ := os.ReadFile(filepath.Join(dir, court+".json")); !bytes.Equal(got, blob) {
			t.Error("le .json d'origine doit rester intact")
		}
		if _, err := os.Stat(filepath.Join(dir, court+".json.gz")); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("le .json.gz non vérifié doit être retiré : %v", err)
		}
	}
}

// TestCompressDir_ParcMixteDuMemeMatchEstCompte : un match qui a les deux formes n'est ni converti ni supprimé.
func TestCompressDir_ParcMixteDuMemeMatchEstCompte(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	ecrireTest(t, filepath.Join(dir, "d0000001.json"), []byte(`{"schemaVersion":1}`))
	ecrireTest(t, filepath.Join(dir, "d0000001.json.gz"), gzipTest(t, []byte(`{"schemaVersion":2}`)))
	rep, err := CompressDir(ctx, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Duplicates != 1 || rep.Converted != 0 || rep.AlreadyCompressed != 0 {
		t.Fatalf("compte rendu = %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(dir, "d0000001.json")); err != nil {
		t.Errorf("le .json d'un doublon est laissé au geste de l'opérateur : %v", err)
	}
}

func TestCompressDir_DossierAbsent(t *testing.T) {
	if _, err := CompressDir(context.Background(), filepath.Join(t.TempDir(), "absent"), false); err == nil {
		t.Fatal("un dossier absent doit être une erreur")
	}
}

func TestCompressDir_ArretSurContexteAnnule(t *testing.T) {
	dir, _ := parcBrut(t, 2)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CompressDir(ctx, dir, false); err == nil {
		t.Fatal("un contexte annulé doit arrêter la conversion")
	}
	if _, err := os.Stat(filepath.Join(dir, "a0000000.json")); err != nil {
		t.Errorf("rien n'a dû être touché : %v", err)
	}
}
