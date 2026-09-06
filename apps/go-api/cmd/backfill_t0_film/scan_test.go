package main

// scan_test.go — LE BALAYAGE DU CORPUS NE COMPTE QUE DES ARTEFACTS (constat C6, revue A-R1).
//
// Les derivations posent `<short8>.derived.json` A COTE de `<short8>.json`, dans le MEME
// dossier. Le balayage retenait tout ce qui finit par `.json` : chaque marque etait donc lue
// comme un artefact, tombait en « sans match_id », et le bilan annoncait jusqu'a DEUX FOIS le
// corpus avec autant d'entrees fantomes — alors que sa propriete affichee est « chaque artefact
// tombe dans EXACTEMENT une categorie, c'est ce qui rend le total verifiable ».

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/replaybuild"
)

// TestScannerArtefacts_IgnoreLesMarquesDeDerivation : deux artefacts, leurs deux marques, et
// une sauvegarde manuelle -> deux verdicts, aucune categorie fantome.
func TestScannerArtefacts_IgnoreLesMarquesDeDerivation(t *testing.T) {
	dir := t.TempDir()
	for _, court := range []string{"aaaa0001", "bbbb0002"} {
		artefact := filepath.Join(dir, court+".json")
		if err := os.WriteFile(artefact,
			[]byte(`{"schemaVersion":39,"matchId":"`+court+`","frameIntervalMs":100,"originMs":0}`),
			0o600); err != nil {
			t.Fatal(err)
		}
		if err := replaybuild.WriteDerivationsMark(artefact, 39, 42); err != nil {
			t.Fatalf("marque %s: %v", court, err)
		}
	}
	// Une sauvegarde manuelle, deja ecartee avant ce lot : elle reste ecartee.
	if err := os.WriteFile(filepath.Join(dir, "cccc0003.json.ancien"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	verdicts, err := scannerArtefacts(dir)
	if err != nil {
		t.Fatalf("scannerArtefacts: %v", err)
	}
	if len(verdicts) != 2 {
		noms := make([]string, 0, len(verdicts))
		for _, v := range verdicts {
			noms = append(noms, v.fichier)
		}
		t.Fatalf("%d verdict(s) %v, attendu 2 — les marques de derivation sont comptees comme "+
			"des artefacts et gonflent le bilan d'entrees fantomes (constat C6)", len(verdicts), noms)
	}
	for _, v := range verdicts {
		if v.raison == raisonSansMatchID {
			t.Errorf("%s tombe en « sans match_id » — c'est la signature d'une marque scannee "+
				"comme un artefact", v.fichier)
		}
	}
}
