package filmdec

// registry_compressed_test.go — UN TAMPON ENCORE COMPRESSE SE REFUSE, IL NE SE LIT PAS A VIDE.
//
// LE SINISTRE QUE CE TEST FERME. Le 2026-09-02, `c17f4941f` (lot 1a de PLAN_CUISSON_PERF) a
// retire l'inflate de [ParseRegistryChunk] : la decompression se fait une fois par film, dans
// `filmsource`. Le retrait etait juste ; ce qui l'accompagnait ne l'etait pas — un tampon encore
// compresse rendait un registre VIDE et une erreur NULLE. Chaque lecteur d'archetype rendait
// ensuite « archetype N absent du registre », un message qui accuse le BUILD DU JEU d'un defaut
// de l'APPELANT, et le fixture E2E `film_e2e/c0a82e88` (deux couches zlib sur ses chunks 00 et
// 07) a decode quatre jours SANS registre — donc sans biped, sans arme au sol, sans equipement,
// sans vehicule, sans objet d'objectif — CI verte comprise.

import (
	"bytes"
	"errors"
	"testing"
)

// enteteZlibDeflate est un en-tete zlib valide (RFC 1950) : CM=8, CINFO=7, FCHECK ajuste.
var enteteZlibDeflate = []byte{0x78, 0x9c}

// TestParseRegistryChunkRefuseUnTamponCompresse : l'entree encore compressee est une ERREUR
// NOMMEE, jamais un registre vide.
func TestParseRegistryChunkRefuseUnTamponCompresse(t *testing.T) {
	for _, cas := range []struct {
		nom     string
		entete  []byte
		refuser bool
	}{
		// Les trois en-tetes zlib rencontres sur les chunks de film : 0x78 0x01 (aucune
		// compression), 0x78 0x5e (le CDN Halo), 0x78 0x9c (compression par defaut).
		{"zlib 0x7801", []byte{0x78, 0x01}, true},
		{"zlib 0x785e", []byte{0x78, 0x5e}, true},
		{"zlib 0x789c", []byte{0x78, 0x9c}, true},
		{"zlib 0x78da", []byte{0x78, 0xda}, true},
		// Un registre INFLATE commence par le `kind` u32 LE de son premier slot : 0x29 sur le
		// build de reference. Rien de tout cela ne doit etre pris pour un en-tete zlib.
		{"registre inflate", []byte{0x29, 0x00}, false},
		{"zeros", []byte{0x00, 0x00}, false},
		// Piege : premier octet 0x78 mais somme de controle FAUSSE — ce n'est pas du zlib, et
		// le test du seul premier octet (celui de `filmsource.inflate`) s'y tromperait.
		{"0x78 sans somme valide", []byte{0x78, 0x00}, false},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			data := append(append([]byte{}, cas.entete...), bytes.Repeat([]byte{0}, archetypeBlockSize)...)
			reg, err := ParseRegistryChunk(data)
			if cas.refuser {
				if !errors.Is(err, ErrRegistryStillCompressed) {
					t.Fatalf("tampon compresse accepte : err=%v, registre=%v — l'appelant doit "+
						"apprendre qu'il n'a pas decompresse, pas croire que le build n'a pas d'archetype", err, reg)
				}
				if reg != nil {
					t.Errorf("un refus doit rendre un registre nil, pas %v", reg)
				}
				return
			}
			if err != nil {
				t.Fatalf("tampon NON compresse refuse a tort : %v", err)
			}
			if reg == nil {
				t.Fatal("registre nil sans erreur")
			}
		})
	}
}

// TestParseRegistryChunkAccepteUnRegistreInflate : le chemin nominal reste ouvert — un bloc
// d'archetype construit a la main se lit, et le refus ne l'attrape pas.
func TestParseRegistryChunkAccepteUnRegistreInflate(t *testing.T) {
	data := make([]byte, archetypeBlockSize)
	copy(data[8:], "object-position-dynamic-precision-component")
	reg, err := ParseRegistryChunk(data)
	if err != nil {
		t.Fatalf("registre inflate refuse : %v", err)
	}
	arch, ok := reg.Archetype(0)
	if !ok || len(arch.Components) != 1 {
		t.Fatalf("archetype 0 : present=%v composants=%d, attendu 1", ok, len(arch.Components))
	}
}

// TestLooksZlibNeSeFiePasAuSeulPremierOctet fige la regle : les DEUX octets de l'en-tete, avec
// leur somme de controle. Un faux positif ici refuserait un registre valide.
func TestLooksZlibNeSeFiePasAuSeulPremierOctet(t *testing.T) {
	if looksZlib(enteteZlibDeflate) != true {
		t.Error("0x789c doit etre reconnu comme un en-tete zlib")
	}
	if looksZlib([]byte{0x78, 0x00}) {
		t.Error("0x7800 n'est pas un en-tete zlib (somme de controle fausse)")
	}
	if looksZlib([]byte{0x78}) {
		t.Error("un octet seul ne peut pas etre un en-tete zlib")
	}
	if looksZlib(nil) {
		t.Error("nil n'est pas un en-tete zlib")
	}
}
