package objectiveevents

// pied_equipe_test.go — L'ÉQUIPE D'UN ÉVÉNEMENT DU PIED EST À L'OCTET 37 (lot 1.1.3).
//
// # CE QUE CE TEST VERROUILLE, ET POURQUOI IL N'EST PAS CIRCULAIRE
//
// Il fait tourner le décodeur de production sur un BLOC RÉEL du pied de `53ce4390`, recopié
// octet pour octet depuis le film (cf. le fichier de provenance à côté du binaire), et exige
// l'équipe 1.
//
// Le bloc est choisi pour être DISCRIMINANT : son octet 37 vaut 1, son octet 55 vaut 0. Remettre
// `footerByteTeam` à 55 rend donc 0 et ce test ROUGIT — c'est la mutation, elle a été jouée. Sur
// un bloc d'équipe 0, les deux lectures coïncideraient et le test ne prouverait rien : c'est
// exactement le piège de méthode qui a laissé l'octet 55 vivre dans la production pendant des
// mois (un champ constant à zéro « coïncide » avec l'équipe 0 sur la moitié des événements).
//
// La valeur attendue ne vient PAS du même octet qu'elle vérifie : l'équipe 1 de
// `2533274823110022` sur ce match est ÉTABLIE AILLEURS, par la trame d'état du film (rang de la
// table des slots de `chunk_00` -> xuid, i-ème entité ti=9 -> désignateur d'équipe). C'est ce
// que publie l'oracle corpus `filmdec.TestResidusPiedOctetEquipe`, qui reste la mesure de
// référence — 665/665 sur quatorze films le 2026-09-13, rejoué le 2026-09-14 sur trois films
// (`53ce4390`, `64e8adfa`, `7344d24f`) : 173 événements sur 173 pour l'octet 37, 84 sur 173 pour
// l'octet 55, et QUATRE lectures en accord parfait sur cent quatre-vingts essayées.
//
// # CE QU'IL NE FAIT PAS
//
// Il ne rejoue pas le balayage aveugle des soixante octets : ça, c'est l'instrument corpus, qui
// exige le cache de films. Ce test-ci tient dans le dépôt, sans film et sans base, et il tient
// la seule chose qu'un test unitaire peut tenir — que le décodeur lit BIEN l'octet que la mesure
// a désigné.

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// Le bloc de référence, et d'où il vient. Ces constantes sont la provenance MISE EN CODE : le
// fichier `testdata/pied_bloc_53ce4390.PROVENANCE.txt` dit la même chose en français.
const (
	piedFixture = "testdata/pied_bloc_53ce4390.bin"
	// piedLo / piedHi : la tranche, EN OCTETS, du pied DÉCOMPRESSÉ du film `53ce4390`
	// (`chunk_40.bin`, type 3 au manifeste, dernier chunk du film).
	piedLo = 34977
	piedHi = 36907
	// Ce que le décodeur doit rendre sur cette tranche.
	piedAttenduTime = 61155
	piedAttenduSlot = 1
	piedAttenduTeam = 1
	piedAttenduXUID = uint64(2533274823110022)
	// piedOctet55 : ce que l'ANCIENNE lecture rendait sur ce même bloc — la mutation.
	piedOctet55 = 0
)

// updatePiedBloc : LA PORTE D'ÉCRITURE DE CETTE FIXTURE, ET D'AUCUNE AUTRE.
//
// Nommée (et non accrochée au `-update` générique) pour la raison qu'a déjà nommée la revue R1
// du lot 0.A : un `-update` partagé laisse `go test ./...` refiger en silence une fixture dont
// la grammaire vient de casser.
var updatePiedBloc = flag.Bool("update-pied-bloc", false,
	"réécrire testdata/pied_bloc_53ce4390.bin depuis PIED_FILM_DIR — CETTE fixture seulement")

// TestPiedEquipeOctet37 : le décodeur de production rend l'équipe 1 sur le bloc de référence.
func TestPiedEquipeOctet37(t *testing.T) {
	bloc := lirePiedFixture(t)
	evs := scanTh10Events(bloc)
	if len(evs) != 1 {
		t.Fatalf("la tranche porte %d événement(s) th=10, attendu exactement 1 — "+
			"la fixture n'est plus celle que la provenance décrit", len(evs))
	}
	e := evs[0]
	if e.TimeMS != piedAttenduTime || e.Slot != piedAttenduSlot || e.XUID != piedAttenduXUID {
		t.Fatalf("bloc inattendu : t=%d slot=%d xuid=%d (attendu t=%d slot=%d xuid=%d)",
			e.TimeMS, e.Slot, e.XUID, piedAttenduTime, piedAttenduSlot, piedAttenduXUID)
	}
	if e.Team != piedAttenduTeam {
		t.Errorf("ÉQUIPE LUE AU MAUVAIS OCTET.\n"+
			"  équipe rendue  : %d\n  équipe prouvée : %d (trame d'état du film, oracle "+
			"filmdec.TestResidusPiedOctetEquipe)\n"+
			"  octet lu       : %d\n"+
			"L'équipe d'un événement du pied est à l'octet 37 du bloc de %d (665/665 sur "+
			"quatorze films). L'octet 55 vaut %d sur ce bloc : s'il est revenu dans "+
			"`footerByteTeam`, c'est la régression que ce test ferme.",
			e.Team, piedAttenduTeam, footerByteTeam, footerBlockBytes, piedOctet55)
	}
}

// TestPiedBlocContreLecture55 : LE CONTRE-TEST, écrit pour que la mutation soit lisible sans être
// jouée à la main. Il rejoue le décodage en lisant l'octet 55 au lieu du 37 et exige que les deux
// DIVERGENT sur ce bloc : c'est ce qui fait de cette fixture une preuve et non une tautologie.
func TestPiedBlocContreLecture55(t *testing.T) {
	bloc := lirePiedFixture(t)
	ebs := piedDebutBlocBits(t, bloc)
	par37 := int(readByteAtBit(bloc, ebs+footerByteTeam*8))
	par55 := int(readByteAtBit(bloc, ebs+55*8))
	if par37 == par55 {
		t.Fatalf("LA FIXTURE NE DISCRIMINE PLUS : octet 37 = octet 55 = %d. Un bloc où les deux "+
			"lectures coïncident ne peut pas faire rougir la régression — en choisir un dont "+
			"l'octet 37 vaut 1", par37)
	}
	if par37 != piedAttenduTeam || par55 != piedOctet55 {
		t.Fatalf("bloc inattendu : octet 37 = %d (attendu %d), octet 55 = %d (attendu %d)",
			par37, piedAttenduTeam, par55, piedOctet55)
	}
}

// piedDebutBlocBits rend l'offset BIT du début du bloc de 60 octets dans la tranche, en refaisant
// le seul chemin qui le connaisse : le marqueur de fin. Il n'y a qu'un bloc dans la fixture.
func piedDebutBlocBits(t *testing.T, bloc []byte) int {
	t.Helper()
	total := len(bloc) * 8
	for b := 0; b <= total-32; b++ {
		if readByteAtBit(bloc, b) == 0 && readByteAtBit(bloc, b+8) == 0 &&
			readByteAtBit(bloc, b+16) == 0x2e && readByteAtBit(bloc, b+24) == 0xe0 {
			ebs := b - footerBlockBytes*8
			if ebs >= 0 && int(readByteAtBit(bloc, ebs+footerByteType*8)) == 10 {
				return ebs
			}
		}
	}
	t.Fatal("aucun bloc th=10 dans la fixture")
	return 0
}

// lirePiedFixture charge le bloc versionné.
func lirePiedFixture(t *testing.T) []byte {
	t.Helper()
	blob, err := os.ReadFile(piedFixture)
	if err != nil {
		t.Fatalf("fixture %s illisible : %v — elle est VERSIONNÉE, son absence est une erreur",
			piedFixture, err)
	}
	if len(blob) != piedHi-piedLo {
		t.Fatalf("fixture %s : %d octets, attendu %d (la tranche [%d, %d) du pied)",
			piedFixture, len(blob), piedHi-piedLo, piedLo, piedHi)
	}
	return blob
}

// TestPiedBlocProvenance : LA FIXTURE EST-ELLE BIEN CES OCTETS-LÀ DU FILM ?
//
// Sauté sans `PIED_FILM_DIR` (le cache de films ne vit pas dans le dépôt). Avec, il recoupe la
// tranche depuis le film et la compare octet pour octet ; avec `-update-pied-bloc`, il la
// réécrit. C'est la seule porte d'écriture : une fixture binaire sans provenance vérifiable est
// un fait sans source.
func TestPiedBlocProvenance(t *testing.T) {
	dir := os.Getenv("PIED_FILM_DIR")
	if dir == "" {
		t.Skip("PIED_FILM_DIR vide : la tranche ne peut pas être recoupée sur le film")
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("chargement de %s : %v", dir, err)
	}
	if filepath.Base(dir) != "53ce4390" {
		t.Fatalf("PIED_FILM_DIR désigne %s : la fixture vient de 53ce4390", filepath.Base(dir))
	}
	pied := film.Chunk(film.NumChunks() - 1)
	if len(pied) < piedHi {
		t.Fatalf("pied de %d octets : la tranche [%d, %d) n'y tient pas", len(pied), piedLo, piedHi)
	}
	tranche := make([]byte, piedHi-piedLo)
	copy(tranche, pied[piedLo:piedHi])
	if *updatePiedBloc {
		if err := os.WriteFile(piedFixture, tranche, 0o600); err != nil {
			t.Fatalf("écriture de %s : %v", piedFixture, err)
		}
		t.Fatalf("1 fixture réécrite : %s (%d octets) ; relancer sans -update-pied-bloc pour vérifier",
			piedFixture, len(tranche))
	}
	fixture := lirePiedFixture(t)
	for i := range tranche {
		if tranche[i] != fixture[i] {
			t.Fatalf("divergence à l'octet %d de la tranche (%d du pied) : film %#02x, fixture %#02x",
				i, piedLo+i, tranche[i], fixture[i])
		}
	}
	t.Log(fmt.Sprintf("tranche [%d, %d) du pied de 53ce4390 : %d octets identiques à la fixture",
		piedLo, piedHi, len(tranche)))
}
