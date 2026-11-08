package main

import (
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/identitefictive"
	"levelup/go-api/internal/replaybuild"
)

// faitsDeTest : un fichier de faits dont les joueurs portent les xuids donnes.
func faitsDeTest(xuids ...uint64) replaybuild.FactsFile {
	var f replaybuild.FactsFile
	for _, x := range xuids {
		f.Players = append(f.Players, domain.MatchPlayerFact{XUID: strconv.FormatUint(x, 10)})
	}
	return f
}

// TestRetraduireFaits : un xuid fictif redevient son original ; un xuid hors registre (des faits
// deja reels) reste tel quel ; un fictif sans original est refuse, de meme que des faits fictifs
// sans liste.
func TestRetraduireFaits(t *testing.T) {
	const reelA, reelB, autre = uint64(2533274790395905), uint64(2550000000000001), uint64(2560000000000007)
	fictifA, fictifB := identitefictive.Xuid(reelA), identitefictive.Xuid(reelB)
	registre := map[uint64]bool{fictifA: true, fictifB: true}
	inverse, err := lireRetraducteur(strings.NewReader("# originaux\n" + strconv.FormatUint(reelA, 10) + "\n" +
		strconv.FormatUint(reelB, 10) + "\n"))
	if err != nil {
		t.Fatal(err)
	}

	faits := faitsDeTest(fictifA, autre, fictifB)
	n, err := retraduireFaits(&faits, inverse, registre)
	if err != nil || n != 2 {
		t.Fatalf("retraduction : %d, %v", n, err)
	}
	for i, attendu := range []uint64{reelA, autre, reelB} {
		if faits.Players[i].XUID != strconv.FormatUint(attendu, 10) {
			t.Fatalf("joueur %d : %s, attendu %d", i, faits.Players[i].XUID, attendu)
		}
	}

	sansListe := faitsDeTest(fictifA)
	if _, err := retraduireFaits(&sansListe, retraducteur{}, registre); err == nil ||
		!strings.Contains(err.Error(), envIdentitesOriginales) {
		t.Fatalf("des faits fictifs sans liste doivent etre refuses en nommant le remede : %v", err)
	}
	reels := faitsDeTest(reelA, autre)
	if n, err := retraduireFaits(&reels, retraducteur{}, registre); err != nil || n != 0 {
		t.Fatalf("des faits reels traversent tels quels : %d, %v", n, err)
	}
}

func TestLireRetraducteurRefus(t *testing.T) {
	for nom, liste := range map[string]string{
		"hors plage":    "2000000000000000\n",
		"pas un nombre": "abc\n",
		"vide":          "# rien\n",
	} {
		if _, err := lireRetraducteur(strings.NewReader(liste)); err == nil {
			t.Errorf("%s : liste acceptee", nom)
		}
	}
}
