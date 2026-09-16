package grammar

// film_format_version_test.go — LA VERSION DE FORMAT SOUS GARDE-RAIL (lot 1.9.1 ter).
//
// Trois choses sont gardees, et la troisieme est celle qui compte :
//
//	LES VALEURS     les sept bobines rendent leur version de format, EN CLAIR. Un test qui
//	                relit la constante qu'il verifie ne verifie rien.
//	LA TRONCATURE   un `chunk_00` coupe avant l'octet 8 ne rend pas une version partielle et
//	                ne panique pas : il rend `ok=false`.
//	LA SEPARATION   la version de format SEPARE les deux groupes de largeur MPP mesures au lot
//	                1.9.1 bis. C'est elle qui justifie que `profile.MPPPourFormat` soit keyee par
//	                cette valeur ; si elle cesse d'etre vraie, la cle est fausse et ce test doit
//	                le dire AVANT que le decoupage ne soit pose sur le mauvais film.

import (
	"path/filepath"
	"testing"
)

// formatVersionLargeMPP : la largeur MPP MESUREE de chaque bobine au lot 1.9.1 bis (oracle `n2`,
// part modale des records) — `true` = 9/5, `false` = 8/3.
//
// ELLE EST RECOPIEE ICI, ET C EST DELIBERE. La meme table vit dans l instrument
// `e191c_typeversions_research_test.go`, qui porte le tag `research` : un test du build PAR
// DEFAUT ne peut pas la lire sans entrainer tout l instrument dans le build court. Et la
// recopie est ce que la regle du depot demande ailleurs pour les tables figees — un test qui
// relit la constante qu il verifie ne verifie rien. Si ces deux tables divergent, c est que la
// mesure a bouge : la refaire, pas les recoller.
var formatVersionLargeMPP = map[string]bool{
	"a521164d": false, "60ae07c4": false, "11de8353": false, "111fa685": false,
	"e5adf7b2": false, "bcb6d393": true, "fb1a1a72": true,
}

// TestFilmFormatVersionDesBobines fige la version de format des sept bobines par build.
func TestFilmFormatVersionDesBobines(t *testing.T) {
	cas := []struct {
		court  string
		format int
	}{
		{"a521164d", 21}, // HI_1_4_1
		{"60ae07c4", 24}, // HI_1_8_0
		{"11de8353", 24}, // HI_1_9_0
		{"111fa685", 24}, // HI_1_10_0
		{"e5adf7b2", 25}, // HI_1_11_0
		{"bcb6d393", 27}, // HI_1_12_0
		{"fb1a1a72", 27}, // HI_1_13_0
	}
	for _, c := range cas {
		dir := filepath.Join("..", "replay", "testdata", "minifilm_"+c.court)
		_, d0 := readChunk00(t, dir)
		got, ok := FilmFormatVersionFromHeader(d0)
		if !ok {
			t.Errorf("%s : version de format illisible", c.court)
			continue
		}
		if got != c.format {
			t.Errorf("%s : version de format %d, %d attendue", c.court, got, c.format)
		}
		id, err := ReadFilmIdentity(d0)
		if err != nil {
			t.Errorf("%s : identite illisible (%v)", c.court, err)
			continue
		}
		if id.FormatVersion != got {
			t.Errorf("%s : profile.FilmIdentity.FormatVersion = %d, %d attendu — les deux lectures de la "+
				"MEME valeur ont diverge", c.court, id.FormatVersion, got)
		}
	}
}

// TestFilmFormatVersionTronquee — un tampon trop court ne rend jamais de version partielle.
func TestFilmFormatVersionTronquee(t *testing.T) {
	plein := []byte{0x29, 0, 0, 0, 0x1b, 0, 0, 0, 0xff}
	for n := 0; n < 8; n++ {
		if v, ok := FilmFormatVersionFromHeader(plein[:n]); ok || v != FilmFormatVersionUnknown {
			t.Errorf("tampon de %d octets : (%d, %v), (0, false) attendu", n, v, ok)
		}
	}
	if v, ok := FilmFormatVersionFromHeader(plein); !ok || v != 27 {
		t.Errorf("tampon complet : (%d, %v), (27, true) attendu", v, ok)
	}
}

// TestFilmFormatVersionSepareLesLargeursMPP — LA JUSTIFICATION DE LA CLE.
//
// Les cinq bobines dont l'oracle `n2` designe `8/3` portent un format STRICTEMENT INFERIEUR a
// celui des deux bobines `9/5`. Tant que c'est vrai, keyer `profile.MPPPourFormat` par cette
// valeur est fonde ; le jour ou une bobine casse l'ordre, la cle est fausse.
func TestFilmFormatVersionSepareLesLargeursMPP(t *testing.T) {
	maxPetit, minGrand := -1, 1<<30
	for _, court := range closureMiniFilms() {
		dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
		_, d0 := readChunk00(t, dir)
		v, ok := FilmFormatVersionFromHeader(d0)
		if !ok {
			t.Fatalf("%s : version de format illisible", court)
		}
		if formatVersionLargeMPP[court] {
			if v < minGrand {
				minGrand = v
			}
			continue
		}
		if v > maxPetit {
			maxPetit = v
		}
	}
	if maxPetit >= minGrand {
		t.Fatalf("la version de format NE SEPARE PLUS les deux groupes de largeur MPP "+
			"(plus grand format `8/3` = %d, plus petit format `9/5` = %d) : la cle de "+
			"`profile.MPPPourFormat` n'est plus fondee", maxPetit, minGrand)
	}
	if maxPetit != 25 || minGrand != 27 {
		t.Errorf("la frontiere a bouge : ]%d, %d] au lieu de ]25, 27] — mesure du 2026-09-15",
			maxPetit, minGrand)
	}
}
