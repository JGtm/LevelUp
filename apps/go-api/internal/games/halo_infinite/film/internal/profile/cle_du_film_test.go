package profile

// cle_du_film_test.go — LA TABLE DE VERITE DE [CleConnue], ET L UNITE DES CLEFS DU PROFIL.

import (
	"errors"
	"strings"
	"testing"
)

// formatConnu / formatInconnu : deux versions de format, l une dans la table, l autre non.
const (
	formatConnu    = 27
	formatInconnuT = 999
)

// TestCleConnueTableDeVerite : la table de verite, cas par cas.
//
// LE CAS QUI COMPTE LE PLUS EST LE TROISIEME : un film SANS section d identification dont la
// majeure est connue est une cle CONNUE, alors que [Profile.Err] porte [ErrUnknownBuild] pour
// lui. Cinq films du cache sont dans ce cas, dont deux temoins du corpus gate — les mettre de
// cote serait deux PERTES au gate.
func TestCleConnueTableDeVerite(t *testing.T) {
	cas := []struct {
		nom     string
		format  int
		build   string
		majeure int
		connue  bool
	}{
		{"build de reference", formatConnu, buildHI1131, 41, true},
		{"build ancien de la table", formatAnciens24, buildHI180, 37, true},
		{"sans section, majeure 31", formatAnciens20, "", majeureSansSection31, true},
		{"sans section, majeure 33", formatAnciens20, "", majeureSansSection33, true},
		{"build hors table", formatConnu, "HI_9_99_0", 41, false},
		{"sans section, majeure hors table", formatAnciens20, "", 29, false},
		{"format hors table", formatInconnuT, buildHI1131, 41, false},
		{"format ET build hors table", formatInconnuT, "HI_9_99_0", 41, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := CleConnue(c.format, c.build, c.majeure); got != c.connue {
				t.Errorf("CleConnue(%d, %q, %d) = %v, attendu %v",
					c.format, c.build, c.majeure, got, c.connue)
			}
			err := ErreurCleInconnue(c.format, c.build, c.majeure)
			if c.connue && err != nil {
				t.Fatalf("cle connue mais erreur rendue : %v", err)
			}
			if !c.connue && err == nil {
				t.Fatal("cle inconnue et AUCUNE erreur rendue — l orchestrateur ne verrait rien")
			}
		})
	}
}

// TestErreurCleInconnueEstTypeeEtNommeLaCle : `errors.Is` reste vrai sur les deux sentinelles, et
// le message porte la cle refusee.
//
// C EST LE CONTRAT QUE LES DEUX ORCHESTRATEURS LISENT (D-4) : sans `errors.Is`, ils ne
// sauraient pas distinguer « cle inconnue » d une panne de lecture.
func TestErreurCleInconnueEstTypeeEtNommeLaCle(t *testing.T) {
	err := ErreurCleInconnue(formatConnu, "HI_9_99_0", 41)
	if !errors.Is(err, ErrUnknownBuild) {
		t.Fatalf("build hors table : %v n est pas ErrUnknownBuild", err)
	}
	if errors.Is(err, ErrUnknownFormat) {
		t.Errorf("le format 27 est connu : %v ne doit pas porter ErrUnknownFormat", err)
	}
	if !strings.Contains(err.Error(), "HI_9_99_0") {
		t.Errorf("le message ne NOMME pas la cle refusee : %q", err.Error())
	}

	fmtErr := ErreurCleInconnue(formatInconnuT, buildHI1131, 41)
	if !errors.Is(fmtErr, ErrUnknownFormat) {
		t.Fatalf("format hors table : %v n est pas ErrUnknownFormat", fmtErr)
	}
	if !strings.Contains(fmtErr.Error(), "999") {
		t.Errorf("le message ne NOMME pas la version refusee : %q", fmtErr.Error())
	}

	sansSection := ErreurCleInconnue(formatAnciens20, "", 29)
	if !errors.Is(sansSection, ErrUnknownBuild) {
		t.Fatalf("majeure hors table : %v n est pas ErrUnknownBuild", sansSection)
	}
}

// TestLesTablesDuProfilPortentLesMemesClefs : LES TABLES DU PROFIL NE DIVERGENT PAS.
//
// [CleConnue] tranche sur la table des largeurs de personnalisation ([PersonnalisationOctets])
// et sur celle des majeures sans section. La table des amorces de grenade
// ([AmorceGrenadePour]) est keyee par les MEMES clefs. Une clef ajoutee a l une et pas a
// l autre ferait un film que le profil dit connaitre et pour lequel il n a pas de grammaire —
// ou l inverse.
func TestLesTablesDuProfilPortentLesMemesClefs(t *testing.T) {
	for _, b := range buildsDeLaTable() {
		if _, ok := AmorceGrenadePour(b, 0); !ok {
			t.Errorf("build %q : largeur de personnalisation posee, amorce de grenade ABSENTE", b)
		}
		if !CleConnue(formatConnu, b, 0) {
			t.Errorf("build %q : dans la table des largeurs, refuse par CleConnue", b)
		}
	}
	for _, m := range []int{majeureSansSection31, majeureSansSection33} {
		if _, ok := AmorceGrenadePour("", m); !ok {
			t.Errorf("majeure %d : clef sans section connue, amorce de grenade ABSENTE", m)
		}
		if !CleConnue(formatAnciens20, "", m) {
			t.Errorf("majeure %d : clef sans section connue, refusee par CleConnue", m)
		}
	}
}

// buildsDeLaTable : les sept builds que le depot connait, nommes une fois pour les gardes.
func buildsDeLaTable() []string {
	return []string{buildHI1131, buildHI1120, buildHI1110, buildHI1100, buildHI190, buildHI180,
		buildHI141}
}
