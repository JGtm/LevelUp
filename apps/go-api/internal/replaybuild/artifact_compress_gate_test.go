package replaybuild

// artifact_compress_gate_test.go — LA CONVERSION PILOTÉE PAR UN APPELANT (la file de régénération des rejeux du
// bureau) : une porte avant chaque artefact, un compte rendu après chacun, et la reprise d'un `.json.gz` laissé
// incomplet par une écriture interrompue.

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/replayartifact"
)

var errPorteFermee = errors.New("porte fermée")

// TestCompressDirWith_PorteAvantChaqueArtefactAConvertir : la porte est consultée une fois par artefact qui demande du
// travail, jamais pour un artefact déjà compressé ; son refus arrête le parc, rend son erreur, et laisse le reste
// intact.
func TestCompressDirWith_PorteAvantChaqueArtefactAConvertir(t *testing.T) {
	dir, contenus := parcBrut(t, 3)
	deja := docJSON(t, "b0000001-1234-4abc-9def-0123456789ab", true)
	ecrireTest(t, filepath.Join(dir, "b0000001.json.gz"), gzipTest(t, deja))

	appels := 0
	rep, err := CompressDirWith(context.Background(), dir, CompressOptions{Gate: func(context.Context) error {
		appels++
		if appels > 1 {
			return errPorteFermee
		}
		return nil
	}})
	if !errors.Is(err, errPorteFermee) {
		t.Fatalf("l'erreur de la porte doit remonter : %v", err)
	}
	if rep.Converted != 1 || appels != 2 {
		t.Fatalf("compte rendu = %+v, portes = %d : un seul artefact converti avant le refus", rep, appels)
	}
	bruts := 0
	for court, blob := range contenus {
		if got, rerr := os.ReadFile(filepath.Join(dir, court+".json")); rerr == nil {
			bruts++
			if !bytes.Equal(got, blob) {
				t.Errorf("%s : brut modifié", court)
			}
		}
	}
	if bruts != 2 {
		t.Errorf("%d bruts restés intacts, attendu 2", bruts)
	}
	// Reprise : sans refus, le reste se convertit, et rien n'est refait.
	rep, err = CompressDirWith(context.Background(), dir, CompressOptions{})
	if err != nil || rep.Converted != 2 || rep.AlreadyCompressed != 2 {
		t.Fatalf("reprise : %+v, %v", rep, err)
	}
}

// TestCompressDirWith_UnCompteRenduParArtefact : l'appelant reçoit le compte de chaque artefact, dont la somme est le
// compte rendu du parc (la file l'affiche au fil de l'eau).
func TestCompressDirWith_UnCompteRenduParArtefact(t *testing.T) {
	dir, _ := parcBrut(t, 3)
	casse := filepath.Join(dir, "c0000001.json")
	tronque := gzipTest(t, docJSON(t, "c0000001-1234-4abc-9def-0123456789ab", true))
	ecrireTest(t, casse, tronque[:len(tronque)/2])
	var somme CompressReport
	n := 0
	rep, err := CompressDirWith(context.Background(), dir, CompressOptions{Progress: func(r CompressReport) {
		n++
		somme.Add(r)
	}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 || somme != rep || rep.Converted != 3 || rep.Failed != 1 {
		t.Fatalf("%d appels, somme %+v, compte rendu %+v", n, somme, rep)
	}
}

// TestCompressDir_ReprendUnCompresseLaisseIncomplet : un `.json.gz` illisible à côté d'un `.json` valide (reste d'une
// écriture interrompue sur un dossier sans lien dur) est retiré, puis le brut est converti : le match finit compressé.
func TestCompressDir_ReprendUnCompresseLaisseIncomplet(t *testing.T) {
	dir := t.TempDir()
	doc := docJSON(t, "d0000003-1234-4abc-9def-0123456789ab", true)
	complet := gzipTest(t, doc)
	ecrireTest(t, filepath.Join(dir, "d0000003.json"), doc)
	ecrireTest(t, filepath.Join(dir, "d0000003.json.gz"), complet[:len(complet)/2])

	blanc, err := CompressDir(context.Background(), dir, true)
	if err != nil || blanc.Converted != 1 || blanc.Failed != 0 {
		t.Fatalf("simulation : %+v, %v", blanc, err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "d0000003.json.gz")); !bytes.Equal(got, complet[:len(complet)/2]) {
		t.Fatal("la simulation ne touche à rien")
	}
	rep, err := CompressDir(context.Background(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep != blanc || rep.Converted != 1 {
		t.Fatalf("compte rendu = %+v (simulation %+v)", rep, blanc)
	}
	if _, err := os.Stat(filepath.Join(dir, "d0000003.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("le .json doit être supprimé après la reprise : %v", err)
	}
	lu, err := replayartifact.ReadBytes(filepath.Join(dir, "d0000003.json.gz"))
	if err != nil || !bytes.Equal(lu, doc) {
		t.Fatalf("le .json.gz repris doit se relire identique : %v", err)
	}
}

// TestCompressDir_CompresseEtBrutIllisiblesNeSupprimeRien : si le `.json` est lui aussi illisible, aucun des deux n'est
// supprimé : c'est un échec, à décider par l'opérateur.
func TestCompressDir_CompresseEtBrutIllisiblesNeSupprimeRien(t *testing.T) {
	dir := t.TempDir()
	complet := gzipTest(t, docJSON(t, "d0000004-1234-4abc-9def-0123456789ab", true))
	brut := filepath.Join(dir, "d0000004.json")
	gz := filepath.Join(dir, "d0000004.json.gz")
	ecrireTest(t, brut, complet[:len(complet)/3])
	ecrireTest(t, gz, complet[:len(complet)/2])
	rep, err := CompressDir(context.Background(), dir, false)
	if err != nil || rep.Failed != 1 || rep.Converted != 0 {
		t.Fatalf("compte rendu = %+v, %v", rep, err)
	}
	for _, p := range []string{brut, gz} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s doit rester : %v", p, err)
		}
	}
}

// TestCompressDir_CompresseLuEnEchecDeLectureNEstPasRetire : une erreur de lecture du disque (pas un contenu
// illisible) ne fait jamais supprimer le `.json.gz`, qui peut être le plus récent.
func TestCompressDir_CompresseLuEnEchecDeLectureNEstPasRetire(t *testing.T) {
	dir := t.TempDir()
	doc := docJSON(t, "d0000005-1234-4abc-9def-0123456789ab", true)
	ecrireTest(t, filepath.Join(dir, "d0000005.json"), doc)
	ecrireTest(t, filepath.Join(dir, "d0000005.json.gz"), gzipTest(t, docJSON(t, "d0000005-1234-4abc-9def-0123456789ab", false)))
	orig := relireCompresse
	relireCompresse = func(string) ([]byte, error) { return nil, errors.New("lecture disque en échec") }
	t.Cleanup(func() { relireCompresse = orig })
	rep, err := CompressDir(context.Background(), dir, false)
	if err != nil || rep.Failed != 1 {
		t.Fatalf("compte rendu = %+v, %v", rep, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "d0000005.json.gz")); err != nil {
		t.Errorf("le .json.gz doit rester : %v", err)
	}
}
