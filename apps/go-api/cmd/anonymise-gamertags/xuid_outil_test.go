package anonymisegamertags

// xuid_outil_test.go — LES GARANTIES DE L OUTIL SUR LES XUIDS, sur des tampons synthetiques (aucune
// fixture, aucun identifiant reel) : chaque forme a chaque decalage, les bornes du texte, le
// remplissage d un nom a largeur fixe, le determinisme de la table, ses refus.

import (
	"bytes"
	"math/bits"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/identitefictive"
)

// xuidsPourTest : deux identifiants synthetiques de la plage, hors de tout film.
var xuidsPourTest = []uint64{0x0009000000000123, 0x0009ABCDEF012345}

func jeuDeXuids(t *testing.T) *jeu {
	t.Helper()
	xs, err := tableXuids(append([]uint64(nil), xuidsPourTest...))
	if err != nil {
		t.Fatal(err)
	}
	return nouveauJeu(nil, xs)
}

// TestXuidBinaireAChaqueDecalage : un xuid pose a n importe quel bit, petit-boutiste (paquets
// d identite) ou gros-boutiste MSB d abord (table du registre), est remplace par son fictif, et
// RIEN d autre ne bouge.
func TestXuidBinaireAChaqueDecalage(t *testing.T) {
	j := jeuDeXuids(t)
	for _, cas := range []struct {
		forme string
		coder func(uint64) []byte
	}{{formeXuidLE, octetsLE}, {formeXuidMSB, octetsBE}} {
		for bit := 0; bit < 16; bit++ {
			base := bruitDeterministe(64, uint64(bit+101))
			attendu := append([]byte(nil), base...)
			pos := 8*8 + bit
			ecrireOctetsAuBit(base, pos, cas.coder(xuidsPourTest[0]))
			ecrireOctetsAuBit(attendu, pos, cas.coder(identitefictive.Xuid(xuidsPourTest[0])))
			compte := remplacer(base, j)
			if compte[cas.forme] != 1 {
				t.Fatalf("%s bit %d : %d occurrence(s), attendu 1 (%v)", cas.forme, bit, compte[cas.forme], compte)
			}
			if !bytes.Equal(base, attendu) {
				t.Fatalf("%s bit %d : le tampon differe hors du xuid", cas.forme, bit)
			}
		}
	}
	if !j.vus[xuidsPourTest[0]] || j.vus[xuidsPourTest[1]] {
		t.Fatalf("xuids vus : %v", j.vus)
	}
}

// TestXuidTexteABornesNettes : decimal et hexadecimal se remplacent dans un texte, et jamais a
// l interieur d un nombre plus long.
func TestXuidTexteABornesNettes(t *testing.T) {
	j := jeuDeXuids(t)
	x := xuidsPourTest[1] // sa forme hexadecimale porte des lettres : majuscules et minuscules different
	f := identitefictive.Xuid(x)
	texte := []byte(strings.Join([]string{
		`{"xuid":"` + strconv.FormatUint(x, 10) + `"}`,
		`0x` + hexa(x, false) + ` / ` + hexa(x, true),
		`cle 0x` + hexa(bits.ReverseBytes64(x), false),
		`colle : 1` + strconv.FormatUint(x, 10) + ` et ` + strconv.FormatUint(x, 10) + `7`,
	}, "\n"))
	attendu := strings.Join([]string{
		`{"xuid":"` + strconv.FormatUint(f, 10) + `"}`,
		`0x` + hexa(f, false) + ` / ` + hexa(f, true),
		`cle 0x` + hexa(bits.ReverseBytes64(f), false),
		`colle : 1` + strconv.FormatUint(x, 10) + ` et ` + strconv.FormatUint(x, 10) + `7`,
	}, "\n")
	compte := remplacer(texte, j)
	if string(texte) != attendu {
		t.Fatalf("texte :\n%s\nattendu :\n%s", texte, attendu)
	}
	if compte[formeXuidDecimal] != 1 || compte[formeXuidHexa] != 2 || compte[formeXuidHexaLE] != 1 {
		t.Fatalf("comptes : %v", compte)
	}
}

// TestNomAlargeurFixeRestePadde : dans un enregistrement de temps forts le gamertag occupe 32
// octets UTF-16LE, remplis de zeros ; le remplacant, de meme longueur, laisse le remplissage et
// l octet qui suit a leur place. Dans la table du registre, la longueur varie et un terminateur
// nul suit : meme garantie.
func TestNomAlargeurFixeRestePadde(t *testing.T) {
	ps := []paire{{origine: "Tango9", pseudonyme: "J00002"}}
	champ := func(nom string) []byte {
		out := make([]byte, 32)
		copy(out, coderUTF16(nom, false))
		return append(append([]byte{0xAA}, out...), 0xBB)
	}
	tampon := append(bruitDeterministe(9, 5), champ("Tango9")...)
	attendu := append(append([]byte(nil), tampon[:9]...), champ("J00002")...)
	if compte := remplacer(tampon, jeuDeNoms(ps)); compte["Tango9"] != 1 || !bytes.Equal(tampon, attendu) {
		t.Fatalf("champ fixe : %v, tampon %x", compte, tampon)
	}
	variable := append(coderUTF16("Tango9", true), 0x00, 0x00, 0x7F)
	attenduVar := append(coderUTF16("J00002", true), 0x00, 0x00, 0x7F)
	if compte := remplacer(variable, jeuDeNoms(ps)); compte["Tango9"] != 1 || !bytes.Equal(variable, attenduVar) {
		t.Fatalf("champ variable : %v, tampon %x", compte, variable)
	}
}

// TestTableDesXuidsDeterministe : meme liste, meme table, quel que soit l ordre de lecture ; et ses
// refus.
func TestTableDesXuidsDeterministe(t *testing.T) {
	a, errA := lireXuidsOriginaux(strings.NewReader("# liste\n2533274790395905\n\n2550000000000001\n"))
	b, errB := lireXuidsOriginaux(strings.NewReader("2550000000000001\n2533274790395905\n"))
	if errA != nil || errB != nil || len(a) != 2 || len(a) != len(b) {
		t.Fatalf("lecture : %v %v %v %v", a, b, errA, errB)
	}
	for i := range a {
		if a[i] != b[i] || a[i].fictif != identitefictive.Xuid(a[i].reel) || !identitefictive.DansLaPlage(a[i].fictif) {
			t.Fatalf("paire %d : %v contre %v", i, a[i], b[i])
		}
	}
	for _, cas := range []struct{ nom, liste string }{
		{"hors plage", "2000000000000000\n"},
		{"double", "2533274790395905\n2533274790395905\n"},
		{"pas un nombre", "abc\n"},
		{"vide", "# rien\n"},
	} {
		if _, err := lireXuidsOriginaux(strings.NewReader(cas.liste)); err == nil {
			t.Errorf("%s : liste acceptee", cas.nom)
		}
	}
}

// TestNomDonneesSeulesAbsentDuCode : un nom marque `donnees` se remplace dans une fixture (sous
// `testdata`) et reste intact dans le code, `go.mod` ou une doc ; un xuid se remplace partout.
func TestNomDonneesSeulesAbsentDuCode(t *testing.T) {
	ps, err := lireCorrespondance(strings.NewReader("Tango9\tJ00002\tdonnees\nAlpha Bravo7\tJoueur000001\n"))
	if err != nil || len(ps) != 2 {
		t.Fatalf("table : %v %v", ps, err)
	}
	xs, err := tableXuids([]uint64{xuidsPourTest[0]})
	if err != nil {
		t.Fatal(err)
	}
	j := nouveauJeu(ps, xs)
	if got := len(j.pour("libs/film/replay/testdata/fixture.json").noms); got != 2 {
		t.Fatalf("fixture : %d noms", got)
	}
	for _, chemin := range []string{"libs/film/replay/code.go", "libs/film/go.mod", "libs/film/README.md"} {
		if got := j.pour(chemin).noms; len(got) != 1 || got[0].origine != "Alpha Bravo7" {
			t.Fatalf("%s : %v", chemin, got)
		}
		if len(j.pour(chemin).xuids) != 1 {
			t.Fatalf("%s : les xuids se remplacent aussi hors des donnees", chemin)
		}
	}
}
