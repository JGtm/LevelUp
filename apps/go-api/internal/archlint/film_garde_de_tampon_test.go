package archlint

// film_garde_de_tampon_test.go — LE GARDE-RAIL DE LA GARDE DE TAMPON DE LA SECTION D IDENTIFICATION
// (regle des deux copies, CLAUDE.md n. 6).
//
// « Les `n` bits a partir de `bit` tiennent-ils dans `d` » s ecrit en UN seul endroit :
// [grammar.tientDansLeTampon] (`grammar/film_identity.go`). Ce test interdit, dans la production de
// `film/**` (hors `film/research/` et hors fichiers `//go:build research`, tests exclus), toute autre
// ecriture de l arrondi a l octet compare a la longueur du tampon — `bit < 0 || (bit+n+7)/8 >
// len(d)`, la forme que la garde a remplacee, ou son inverse —, et exige que l hote la porte une
// fois.

import (
	"regexp"
	"testing"
)

const (
	// hoteGardeDeTampon : le seul fichier qui ecrit la garde.
	hoteGardeDeTampon = "internal/games/halo_infinite/film/internal/grammar/film_identity.go"
	// plancherFichiersGardeDeTampon : plancher contre un balayage muet.
	plancherFichiersGardeDeTampon = 500
)

// formeDeLaGardeDeTampon : l arrondi a l octet d une fin de lecture, compare a la longueur d un
// tampon.
var formeDeLaGardeDeTampon = regexp.MustCompile(`\+\s*7\s*\)\s*/\s*8\s*(>|<=)\s*len\(`)

// TestGardeDeTamponUnique interdit les copies de la garde de tampon hors de son hote.
func TestGardeDeTamponUnique(t *testing.T) {
	hote := 0
	fichiers := balayerLaProductionHorsResearch(t, []string{racineLocalisateur}, func(rel string, blob []byte) {
		n := len(formeDeLaGardeDeTampon.FindAll(blob, -1))
		if rel == hoteGardeDeTampon {
			hote = n
			return
		}
		if n > 0 {
			t.Errorf("%s : %d garde(s) de tampon ecrite(s) a la main — appeler grammar.tientDansLeTampon "+
				"(%s)", rel, n, hoteGardeDeTampon)
		}
	})
	if fichiers < plancherFichiersGardeDeTampon {
		t.Fatalf("balayage muet : %d fichiers de production vus, plancher %d", fichiers,
			plancherFichiersGardeDeTampon)
	}
	if hote != 1 {
		t.Errorf("%s porte %d garde(s) de tampon, 1 attendue — deplacer le garde-rail avec la garde",
			hoteGardeDeTampon, hote)
	}
}

// TestGardeRailGardeDeTamponVecteurs : la forme remplacee et son inverse rougissent ; un appel a la
// garde et une division sans comparaison au tampon passent.
func TestGardeRailGardeDeTamponVecteurs(t *testing.T) {
	for _, c := range []struct {
		src     string
		trouvee bool
	}{
		{"if bit < 0 || (bit+32+7)/8 > len(d) {", true},
		{"if bodyBit < 0 || (bodyBit + 2*largeur + 7) / 8 > len(chunk0) {", true},
		{"return bit >= 0 && (bit+n+7)/8 <= len(d)", true},
		{"if !tientDansLeTampon(d, bit, 32) {", false},
		{"octets := (bits + 7) / 8", false},
	} {
		if got := formeDeLaGardeDeTampon.MatchString(c.src); got != c.trouvee {
			t.Errorf("%q : trouvee %v, attendu %v", c.src, got, c.trouvee)
		}
	}
}
