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
	"strings"
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

// TestCompressDir_CompresseVideOuSansEnteteEstReprisCommeIncomplet : un fichier NOMMÉ `.json.gz` de 0 ou 1 octet (arrêt
// entre la création exclusive et la première écriture du repli sans lien dur) est illisible pour la conversion : il est
// retiré, puis le `.json` valide est converti. Sans cela le match restait « contenus différents » pour toujours.
func TestCompressDir_CompresseVideOuSansEnteteEstReprisCommeIncomplet(t *testing.T) {
	for _, octets := range [][]byte{{}, {0x1f}, []byte("pas du gzip")} {
		dir := t.TempDir()
		doc := docJSON(t, "d0000006-1234-4abc-9def-0123456789ab", true)
		ecrireTest(t, filepath.Join(dir, "d0000006.json"), doc)
		ecrireTest(t, filepath.Join(dir, "d0000006.json.gz"), octets)

		blanc, err := CompressDir(context.Background(), dir, true)
		if err != nil || blanc.Converted != 1 || blanc.Different != 0 || blanc.Failed != 0 {
			t.Fatalf("%d octets, simulation : %+v, %v", len(octets), blanc, err)
		}
		rep, err := CompressDir(context.Background(), dir, false)
		if err != nil || rep != blanc {
			t.Fatalf("%d octets : %+v (simulation %+v), %v", len(octets), rep, blanc, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "d0000006.json")); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%d octets : le .json doit être supprimé : %v", len(octets), err)
		}
		if lu, err := replayartifact.ReadBytes(filepath.Join(dir, "d0000006.json.gz")); err != nil || !bytes.Equal(lu, doc) {
			t.Errorf("%d octets : relecture : %v", len(octets), err)
		}
	}
}

// TestCompressDir_BrutTronqueOuVideNEstPasUnBrutValide : un `.json` brut tronqué ou vide ne sert jamais à reprendre un
// `.json.gz` incomplet : rien n'est retiré, l'échec est compté.
func TestCompressDir_BrutTronqueOuVideNEstPasUnBrutValide(t *testing.T) {
	complet := gzipTest(t, docJSON(t, "d0000007-1234-4abc-9def-0123456789ab", true))
	for _, brut := range [][]byte{{}, []byte(`{"matchId":"d0000007","tracks":[`)} {
		dir := t.TempDir()
		ecrireTest(t, filepath.Join(dir, "d0000007.json"), brut)
		ecrireTest(t, filepath.Join(dir, "d0000007.json.gz"), complet[:len(complet)/2])
		for _, dryRun := range []bool{true, false} {
			rep, err := CompressDir(context.Background(), dir, dryRun)
			if err != nil || rep.Failed != 1 || rep.Converted != 0 {
				t.Fatalf("brut de %d octets, simulation=%v : %+v, %v", len(brut), dryRun, rep, err)
			}
		}
		for _, n := range []string{"d0000007.json", "d0000007.json.gz"} {
			if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
				t.Errorf("%s doit rester : %v", n, err)
			}
		}
	}
}

// TestCompressDir_UnAutreEcrivainQuiRemplaceLeCompresseNEstPasSupprime : après notre pose, le serveur re-cuit le match :
// il remplace le `.json.gz` et retire le `.json`. Le fichier du serveur n'est JAMAIS supprimé, le cas n'est pas un échec.
func TestCompressDir_UnAutreEcrivainQuiRemplaceLeCompresseNEstPasSupprime(t *testing.T) {
	dir, _ := parcBrut(t, 1)
	serveur := gzipTest(t, docJSON(t, "a0000000-1234-4abc-9def-0123456789ab", false))
	orig := apresLaPose
	apresLaPose = func(ecrit string) {
		ecrireTest(t, ecrit, serveur) // le rejeu re-cuit remplace le nôtre
		_ = os.Remove(strings.TrimSuffix(ecrit, ".gz"))
	}
	t.Cleanup(func() { apresLaPose = orig })
	rep, err := CompressDir(context.Background(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Replaced != 1 || rep.Failed != 0 || rep.Converted != 0 {
		t.Fatalf("compte rendu = %+v", rep)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "a0000000.json.gz")); !bytes.Equal(got, serveur) {
		t.Fatal("le .json.gz du serveur a été touché")
	}
}

// TestCompressDir_DoublonRetireParUnAutreNEstPasCompteCommeResorbe : le `.json` d'un doublon disparaît (un serveur l'a
// retiré) entre le listage et le jugement : le match est déjà compressé, pas « résorbé » par nous.
func TestCompressDir_DoublonRetireParUnAutreNEstPasCompteCommeResorbe(t *testing.T) {
	dir := t.TempDir()
	doc := docJSON(t, "d0000008-1234-4abc-9def-0123456789ab", true)
	doublon(t, dir, "d0000008", doc, doc)
	orig := avantLeJugement
	avantLeJugement = func(compressed string) { _ = os.Remove(strings.TrimSuffix(compressed, ".gz")) }
	t.Cleanup(func() { avantLeJugement = orig })
	rep, err := CompressDir(context.Background(), dir, false)
	if err != nil || rep.Resolved != 0 || rep.AlreadyCompressed != 1 {
		t.Fatalf("compte rendu = %+v, %v", rep, err)
	}
}

// TestCompressDirWith_UnFichierALaFois : la porte du fichier suivant n'est consultée qu'après le compte rendu du
// précédent : jamais deux conversions en cours.
func TestCompressDirWith_UnFichierALaFois(t *testing.T) {
	dir, _ := parcBrut(t, 5)
	var evenements []string
	_, err := CompressDirWith(context.Background(), dir, CompressOptions{
		Gate:     func(context.Context) error { evenements = append(evenements, "porte"); return nil },
		Progress: func(CompressReport) { evenements = append(evenements, "fini") },
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, e := range evenements {
		if want := []string{"porte", "fini"}[i%2]; e != want {
			t.Fatalf("événements %v : rang %d = %s, attendu %s", evenements, i, e, want)
		}
	}
	if len(evenements) != 10 {
		t.Fatalf("événements = %v", evenements)
	}
}
