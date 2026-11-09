package grammar

// components_moteur_de_partie_test.go — LES VECTEURS DU MOTEUR DE PARTIE, ECRITS D APRES L ECRIVAIN.
//
// Chaque vecteur est l etat qu un ecrivain du jeu serialise, ecrit bit a bit dans l ordre ou il
// l ecrit (champ poids fort en tete) : FUN_142f0688c -> FUN_1406d60f4 (i11), FUN_142f06530 (i13),
// FUN_142f06308 (i14), FUN_142edad74 / FUN_142ecf980 / FUN_142ba7c74 / FUN_142b6f75c (i15),
// FUN_142edd05c / FUN_142ed15a0 (i16), 0x142edbef4 (i17), FUN_142b6f76c (FUN_140d580d0). Le test
// exige que le dispatch consomme EXACTEMENT les bits ecrits et rende `ported`. La forme courte
// d i14 (niveau < 2) n a pas d ecrivain dans cet executable : son vecteur suit le LECTEUR
// FUN_142f0328c (branche `JC 0x142f03405`).

import (
	"strings"
	"testing"
)

// vecteurMoteur : un etat ecrit, le niveau declare par le registre, et le compte attendu.
type vecteurMoteur struct {
	id, composant string
	niveau        uint32
	bits          string // separateurs ' ' et '|' ignores
	attendus      int
}

func (v vecteurMoteur) flux() string {
	return strings.NewReplacer(" ", "", "|", "").Replace(v.bits)
}

// minuteurMoteur36000 : FUN_142b6f76c, n = 16 : 36000,0 (code 65535), 0,0, queue 0.
const minuteurMoteur36000 = "1111111111111111 0000000000000000 00000"

var vecteursMoteur = []vecteurMoteur{
	{"V11", compGameEngineSoftCeilings, 1, strings.Repeat("10", 64), 128},
	{"V13a aucun volume", compGameEngineDisabledKillVolumes, 1, "0000000000000", 13},
	{"V13b trois volumes", compGameEngineDisabledKillVolumes, 1, "0000000000011 | 101", 16},
	// i14, niveau 2 (FUN_142f06308) : drapeau 1 ; reel 1,0 (code 65535) ; index 5 puis trois -1 ;
	// mot 0x1234 puis trois 0xffff (non ecrits).
	{"V14 forme longue", compGameEngineComposerLetterbox, 2,
		"1 | 1111111111111111 | 0 0000101 | 1 | 1 | 1 | 1 0001001000110100 | 0 | 0 | 0", 48},
	// Meme tronc au niveau 1 : le lecteur lit R(64) au lieu des quatre mots.
	{"V14 forme courte", compGameEngineComposerLetterbox, 1,
		"1 | 1111111111111111 | 0 0000101 | 1 | 1 | 1 | " + strings.Repeat("0", 64), 92},
	// i15 : fentes 0 et 1 ; fente 0 etiquette 2 (FUN_142b6f75c) ; fente 1 etiquette 0.
	{"V15a", compManagedEngineTimers, 1,
		strings.Repeat("0", 62) + "11 | 10 " + minuteurMoteur36000 + " | 00", 105},
	// i15 : fente 0 seule, etiquette 1 (FUN_142ba7c74) : 3600,0, 0,0, queue 1, 0,0.
	{"V15b", compManagedEngineTimers, 1,
		strings.Repeat("0", 63) + "1 | 01 1111111111111111 0000000000000000 00001 0000000000000000", 119},
	// i15 : fente 63 seule (bit de poids fort du masque), etiquette 3 : la forme a trois champs.
	{"V15c", compManagedEngineTimers, 1, "1" + strings.Repeat("0", 63) + " | 11 " + minuteurMoteur36000, 103},
	{"V15d masque vide", compManagedEngineTimers, 1, strings.Repeat("0", 64), 64},
	// i16 : scenario 0, ecrit 1 (FUN_142ed15a0 ecrit valeur + 1 sur 7 bits), drapeau 1.
	{"V16", compScenarioIntro, 4, "0000001 1", 8},
	{"V17", compMatchflowIsPlayingFlags, 1, "00000101", 8},
}

// TestComposantsDuMoteurSuiventLEcrivain : chaque etat ecrit est relu au bit pres.
func TestComposantsDuMoteurSuiventLEcrivain(t *testing.T) {
	for _, v := range vecteursMoteur {
		flux := v.flux()
		if len(flux) != v.attendus {
			t.Fatalf("%s : le vecteur porte %d bits, %d annonces — vecteur mal ecrit", v.id, len(flux), v.attendus)
		}
		// Un fond de uns apres le vecteur : un lecteur qui lit trop loin le voit.
		br := lecteurDInstrument(bitsDe(flux+strings.Repeat("1", 256), (len(flux)+256)/8+8))
		_, _, ported := consumeByName(br, v.composant, 2, v.niveau)
		if !ported {
			t.Errorf("%s : le dispatch rend ported=false", v.id)
		}
		if br.BitPos() != v.attendus {
			t.Errorf("%s (%s, niveau %d) : %d bits consommes, l ecrivain en a ecrit %d",
				v.id, v.composant, v.niveau, br.BitPos(), v.attendus)
		}
	}
}

// TestCompteDeVolumesNonBorne : FUN_142f03498 ne borne pas le compte R(13) ; le port non plus.
func TestCompteDeVolumesNonBorne(t *testing.T) {
	br := lecteurDInstrument(bitsDe(strings.Repeat("1", 13), 1100))
	consumeByName(br, compGameEngineDisabledKillVolumes, 0, 1)
	if got, want := br.BitPos(), 13+8191; got != want {
		t.Fatalf("compte 8191 : %d bits consommes, %d attendus", got, want)
	}
}

// TestLecteurDeMinuteurRendLesQuanta : FUN_140d580d0 rend les quanta dans l ordre ou
// FUN_142b6f76c les ecrit ; FUN_142ba78dc consomme les 3n + 5 bits de FUN_142ba7c74, ni plus ni
// moins (le marqueur qui suit se relit intact).
func TestLecteurDeMinuteurRendLesQuanta(t *testing.T) {
	br := lecteurDInstrument(bitsDe("100110101000111", 8))
	m := lireMinuteur140d580d0(br, 5)
	if m != (Minuteur{A: 0b10011, B: 0b01010, Queue: 0b00111}) || br.BitPos() != 15 {
		t.Fatalf("n = 5 : %+v apres %d bits", m, br.BitPos())
	}
	flux := strings.ReplaceAll("0000000000000001 0000000000000010 10101 0000000000000011 101", " ", "")
	br = lecteurDInstrument(bitsDe(flux, 16))
	lireMinuteur142ba78dc(br, 16)
	if fin, marqueur := br.BitPos(), br.ReadBits(3); fin != 53 || marqueur != 0b101 {
		t.Fatalf("n = 16, forme a trois reels : arret a %d (53 attendu), marqueur %03b (101 attendu)",
			fin, marqueur)
	}
}
