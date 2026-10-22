package anonymisegamertags

// outil_test.go — LES GARANTIES DE L OUTIL, sur des tampons synthetiques (aucune fixture, aucun
// nom reel) : la substitution a chaque decalage, les trois formes de fichier, les refus.

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"math/rand/v2"
	"strings"
	"testing"
)

var tablePourTest = []paire{{origine: "Alpha Bravo7", pseudonyme: "Joueur000001"}, {origine: "Tango9", pseudonyme: "J00002"}}

// bruitDeterministe : un tampon pseudo-aleatoire a graine fixe.
func bruitDeterministe(n int, graine uint64) []byte {
	r := rand.New(rand.NewPCG(graine, graine^0x9e3779b97f4a7c15))
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(r.IntN(256))
	}
	return out
}

// TestRemplacerAChaqueDecalage : un nom pose a n importe quel bit, en UTF-16LE ou BE, est remplace,
// et RIEN d autre ne bouge — c est la propriete qui laisse le film decodable a l identique.
func TestRemplacerAChaqueDecalage(t *testing.T) {
	for _, e := range encodages[:2] {
		for bit := 0; bit < 16; bit++ {
			base := bruitDeterministe(64, uint64(bit+1))
			attendu := append([]byte(nil), base...)
			pos := 8*8 + bit
			ecrireOctetsAuBit(base, pos, e.coder("Alpha Bravo7"))
			ecrireOctetsAuBit(attendu, pos, e.coder("Joueur000001"))
			compte := remplacer(base, tablePourTest)
			if compte["Alpha Bravo7"] < 1 {
				t.Fatalf("%s bit %d : occurrence non trouvee", e.nom, bit)
			}
			if !bytes.Equal(base, attendu) {
				t.Fatalf("%s bit %d : le tampon differe hors du nom", e.nom, bit)
			}
		}
	}
}

// TestUTF8SeulementAligne : l UTF-8 se remplace aligne, jamais a un decalage (le texte est aligne,
// et un motif de 8 bits par caractere a un decalage arbitraire est du bruit).
func TestUTF8SeulementAligne(t *testing.T) {
	texte := []byte(`{"name":"Tango9","x":"Tango9x"}`)
	compte := remplacer(texte, tablePourTest)
	if got := string(texte); got != `{"name":"J00002","x":"J00002x"}` || compte["Tango9"] != 2 {
		t.Fatalf("texte : %q (%v)", got, compte)
	}
	decale := make([]byte, 16)
	ecrireOctetsAuBit(decale, 3, []byte("Tango9"))
	avant := append([]byte(nil), decale...)
	if c := remplacer(decale, tablePourTest); len(c) != 0 || !bytes.Equal(decale, avant) {
		t.Fatalf("UTF-8 decale remplace : %v", c)
	}
}

func TestFormesDeFichier(t *testing.T) {
	clair := append(bruitDeterministe(300, 7), coderUTF16("Tango9", true)...)
	clair = append(clair, bruitDeterministe(50, 8)...)
	var z, g bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&z, zlib.BestCompression)
	_, _ = zw.Write(clair)
	_ = zw.Close()
	gw := gzip.NewWriter(&g)
	gw.Name = "fixture"
	_, _ = gw.Write(clair)
	_ = gw.Close()
	brut := append([]byte{0x01}, clair...)
	for _, cas := range []struct {
		nom    string
		octets []byte
		forme  forme
	}{{"zlib", z.Bytes(), formeZlib}, {"gzip", g.Bytes(), formeGzip}, {"brut", brut, formeBrute}} {
		sortie, compte, frm, err := anonymiserOctets(cas.octets, tablePourTest)
		if err != nil || frm != cas.forme || compte["Tango9"] != 1 {
			t.Fatalf("%s : forme %s, compte %v, err %v", cas.nom, frm, compte, err)
		}
		relu, err := ouvrir(sortie)
		if err != nil || relu.forme != cas.forme {
			t.Fatalf("%s : relecture %v", cas.nom, err)
		}
		if bytes.Contains(relu.clair, coderUTF16("Tango9", true)) || !bytes.Contains(relu.clair, coderUTF16("J00002", true)) {
			t.Fatalf("%s : substitution absente du fichier relu", cas.nom)
		}
		if cas.forme == formeGzip && relu.entete.Name != "fixture" {
			t.Fatalf("gzip : en-tete perdu (%q)", relu.entete.Name)
		}
	}
}

// TestFichierSansOccurrenceIntact : rien a remplacer, rien de reecrit — pas meme recompresse.
func TestFichierSansOccurrenceIntact(t *testing.T) {
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	_, _ = zw.Write(bruitDeterministe(200, 3))
	_ = zw.Close()
	sortie, compte, _, err := anonymiserOctets(z.Bytes(), tablePourTest)
	if err != nil || len(compte) != 0 || !bytes.Equal(sortie, z.Bytes()) {
		t.Fatalf("fichier sans occurrence modifie (compte %v, err %v)", compte, err)
	}
}

func TestZlibTronqueRefuse(t *testing.T) {
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	_, _ = zw.Write(bruitDeterministe(5000, 4))
	_ = zw.Close()
	if _, _, _, err := anonymiserOctets(z.Bytes()[:z.Len()/2], tablePourTest); err == nil {
		t.Fatal("un flux zlib tronque doit etre refuse")
	}
}

func TestCorrespondanceRefus(t *testing.T) {
	for _, cas := range []struct{ nom, table string }{
		{"longueur", "Tango9\tJ0002\n"},
		{"trop court", "Ab\tJ1\n"},
		{"non ASCII", "Tangö9\tJ00002\n"},
		{"pseudonyme double", "Tango9\tJ00002\nTango8\tJ00002\n"},
		{"original double", "Tango9\tJ00002\nTango9\tJ00003\n"},
		{"pseudonyme original", "Tango9\tTango8\nTango8\tJ00003\n"},
		{"colonnes", "Tango9 J00002\n"},
		{"vide", "# rien\n"},
	} {
		if _, err := lireCorrespondance(strings.NewReader(cas.table)); err == nil {
			t.Errorf("%s : table acceptee", cas.nom)
		}
	}
	ps, err := lireCorrespondance(strings.NewReader("# table\nTango9\tJ00002\r\n\nAlpha Bravo7\tJoueur000001\n"))
	if err != nil || len(ps) != 2 || ps[0].origine != "Alpha Bravo7" {
		t.Fatalf("table valide refusee ou mal ordonnee : %v %v", ps, err)
	}
}
