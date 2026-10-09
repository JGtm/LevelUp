package hinavmesh

// conteneur_test.go — TEMOINS DE ROBUSTESSE de l'emballage et du lecteur d'entiers.
//
// La regle du package est celle de mapvar : un octet inattendu remonte une ERREUR, on ne
// saute jamais rien en silence. Un decalage d'un octet dans la table des types produirait
// des offsets de membres plausibles et des coordonnees fausses — un fond de carte faux est
// pire que pas de fond du tout, parce que rien ne le signale.

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEntierEmpaqueteFormes(t *testing.T) {
	cas := []struct {
		nom      string
		octets   []byte
		veut     int
		longueur int
	}{
		{"1 octet", []byte{0x43}, 0x43, 1},
		{"1 octet, zero", []byte{0x00}, 0, 1},
		{"1 octet, maximum", []byte{0x7F}, 0x7F, 1},
		// Forme a 2 octets : offset de membre 128 (`flags` de hkaiNavMesh).
		{"2 octets", []byte{0x80, 0x80}, 0x80, 2},
		{"2 octets, maximum", []byte{0xBF, 0xFF}, 0x3FFF, 2},
		// Forme a 3 octets : le format du type `int`, (32 bits << 10) | signe | genre entier.
		{"3 octets", []byte{0xC0, 0x82, 0x04}, 0x8204, 3},
		{"4 octets", []byte{0xE1, 0x02, 0x03, 0x04}, 0x1020304, 4},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			l := &lecteurEmpaquete{buf: c.octets}
			if got := l.entier(); got != c.veut {
				t.Errorf("entier() = 0x%x, 0x%x attendu", got, c.veut)
			}
			if l.err != nil {
				t.Errorf("erreur inattendue: %v", l.err)
			}
			if l.pos != c.longueur {
				t.Errorf("%d octets consommes, %d attendus", l.pos, c.longueur)
			}
		})
	}
}

func TestEntierEmpaqueteRefuseLInconnu(t *testing.T) {
	// La forme a 5 octets et au-dela n'a jamais ete observee. Elle doit remonter une
	// erreur, pas etre devinee : c'est la seule facon d'apprendre qu'une carte l'utilise.
	for _, b := range []byte{0xF0, 0xF8, 0xFF} {
		l := &lecteurEmpaquete{buf: []byte{b, 1, 2, 3, 4, 5}}
		l.entier()
		if l.err == nil {
			t.Errorf("l'octet 0x%02x devrait etre refuse", b)
		}
	}
}

func TestEntierEmpaqueteRefuseLaTroncature(t *testing.T) {
	l := &lecteurEmpaquete{buf: []byte{0xC0, 0x82}} // forme a 3 octets, 2 disponibles
	l.entier()
	if l.err == nil {
		t.Fatal("un entier tronque devrait remonter une erreur")
	}
	if !strings.Contains(l.err.Error(), "tronque") {
		t.Errorf("erreur peu parlante: %v", l.err)
	}
}

func TestDecompresseRefuseUnEnteteFaux(t *testing.T) {
	bon := chargeBlobTemoin(t, "01af558d-53ab-4f05-ba68-92d805fc6260")
	cas := []struct {
		nom   string
		abime func([]byte)
		motif string
	}{
		{"version inconnue", func(b []byte) { binary.BigEndian.PutUint32(b[0:], 3) }, "version"},
		{"taille annoncee fausse", func(b []byte) { binary.BigEndian.PutUint32(b[4:], 42) }, "annonce"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			abime := append([]byte(nil), bon...)
			c.abime(abime)
			_, err := decompresse(abime)
			if err == nil {
				t.Fatal("un en-tete faux devrait remonter une erreur")
			}
			if !strings.Contains(err.Error(), c.motif) {
				t.Errorf("erreur peu parlante: %v", err)
			}
		})
	}
}

func TestDecodeRefuseUnBlobTronque(t *testing.T) {
	bon := chargeBlobTemoin(t, "01af558d-53ab-4f05-ba68-92d805fc6260")
	for _, garde := range []int{0, 8, 12, 100, len(bon) / 2, len(bon) - 1} {
		if _, err := Decode(bon[:garde]); err == nil {
			t.Errorf("un blob tronque a %d octets devrait remonter une erreur", garde)
		}
	}
}

// TestDecodeRefuseUnFluxAbime abime le flux COMPRESSE. Ce temoin est FAIBLE et il faut le
// dire : c'est la somme de controle zlib qui attrape presque tout, pas le decodeur. Le
// temoin qui porte vraiment sur la lecture des types est dans reflexion_test.go, ou la
// mutation porte sur la charge INFLATEE.
func TestDecodeRefuseUnFluxAbime(t *testing.T) {
	bon := chargeBlobTemoin(t, "01af558d-53ab-4f05-ba68-92d805fc6260")
	echecs := 0
	const essais = 64
	for i := 0; i < essais; i++ {
		abime := append([]byte(nil), bon...)
		// On mord dans le flux zlib, apres l'en-tete de 12 octets et le prefixe Bond.
		pos := 40 + i*(len(abime)-80)/essais
		abime[pos] ^= 0xFF
		if _, err := Decode(abime); err != nil {
			echecs++
		}
	}
	if echecs != essais {
		t.Errorf("%d/%d mutations detectees : %d octets abimes ont laisse un decodage passer",
			echecs, essais, essais-echecs)
	}
}

func chargeBlobTemoin(t *testing.T, assetID string) []byte {
	t.Helper()
	chemin := filepath.Join("testdata", assetID+".navmesh.blob")
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, versionne
	if err != nil {
		t.Fatalf("lecture de %s: %v", chemin, err)
	}
	return blob
}
