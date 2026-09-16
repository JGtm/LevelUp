package filmdec

// build_profile_test.go — LE PROFIL PAR BUILD SOUS GARDE-RAIL (lot 1.9.1 bis, pas 3).
//
// Deux choses sont gardees ici, et une seule d'entre elles est un compte :
//
//	LA TABLE   chaque couple (build, version de format) connu rend ses deux valeurs ; un build
//	           inconnu rend `ErrUnknownBuild`, un FORMAT inconnu rend `ErrUnknownFormat`, et
//	           JAMAIS le profil du voisin le plus proche (D-4 d'ADR 0034).
//	DEUX CLES  depuis le lot 1.9.1 ter, la largeur MPP est keyee par la VERSION DE FORMAT
//	           (`chunk_00+4`) et la personnalisation par le BUILD. Le cas `HI_1_10_0` /
//	           `HI_1_9_0` — meme format 24, personnalisations 1 492 et 1 312 — est ce qui
//	           INTERDIT de n'en garder qu'une, et il est dans la table ci-dessous pour ca.
//	L'ORACLE   `n2` est constant au decoupage du profil sur le build dont l'executable est
//	           ouvert, et il CESSE de l'etre des qu'on fausse ce decoupage d'un seul bit.
//	           C'est la mutation qui rougit : sans elle, la ligne `HI_1_13_0` du profil ne
//	           serait qu'une affirmation.

import (
	"errors"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// TestBuildProfileTable fige la table et son refus.
func TestBuildProfileTable(t *testing.T) {
	// LES VALEURS SONT ECRITES EN CLAIR : un test qui relit la constante qu'il verifie ne
	// verifie rien.
	cas := []struct {
		build          string
		format         int
		perso          int
		lead, index    int
		mppIndetermine bool
	}{
		{build: "HI_1_13_0", format: 27, perso: 1852, lead: 9, index: 5},
		{build: "HI_1_12_0", format: 27, perso: 1852, lead: 9, index: 5},
		{build: "HI_1_11_0", format: 25, perso: 1492, mppIndetermine: true},
		{build: "HI_1_10_0", format: 24, perso: 1492, mppIndetermine: true},
		{build: "HI_1_9_0", format: 24, perso: 1312, mppIndetermine: true},
		{build: "HI_1_8_0", format: 24, perso: 1312, mppIndetermine: true},
		{build: "HI_1_4_1", format: 21, perso: 2052, mppIndetermine: true},
	}
	for _, c := range cas {
		p, err := BuildProfileFor(c.build, c.format)
		if err != nil {
			t.Errorf("%s : profil refuse (%v)", c.build, err)
			continue
		}
		if p.FormatVersion != c.format {
			t.Errorf("%s : version de format %d, %d attendue", c.build, p.FormatVersion, c.format)
		}
		if p.PersoBytes != c.perso {
			t.Errorf("%s : bloc de personnalisation %d octets, %d attendus", c.build, p.PersoBytes, c.perso)
		}
		if c.mppIndetermine {
			if p.MPP.Valid() {
				t.Errorf("%s : largeur MPP %s POSEE alors que les deux oracles se contredisent — "+
					"si elle vient d'etre tranchee, ecrire sa provenance ET mettre ce cas a jour",
					c.build, p.MPP)
			}
			continue
		}
		if p.MPP.Lead != c.lead || p.MPP.Index != c.index {
			t.Errorf("%s : largeurs MPP %s, %d/%d attendues", c.build, p.MPP, c.lead, c.index)
		}
	}
}

// TestBuildProfileRefuseUnBuildInconnu — D-4 : jamais le profil du plus proche.
func TestBuildProfileRefuseUnBuildInconnu(t *testing.T) {
	for _, b := range []string{"HI_1_14_0", "HI_1_7_0", "", "HI_1_13_1"} {
		if _, err := BuildProfileFor(b, 27); !errors.Is(err, ErrUnknownBuild) {
			t.Errorf("build %q : erreur %v, ErrUnknownBuild attendue", b, err)
		}
	}
}

// TestBuildProfileRefuseUnFormatInconnu — D-4 sur l'AUTRE cle. Un format 28 que ce depot ne
// connait pas n'herite PAS du decoupage du 27 : il est mis de cote. Le 26 non plus, alors qu'il
// est ENTRE deux formats connus — c'est exactement le cas ou l'interpolation serait tentante.
func TestBuildProfileRefuseUnFormatInconnu(t *testing.T) {
	for _, f := range []int{0, 19, 22, 23, 26, 28, 40} {
		if _, err := BuildProfileFor("HI_1_13_0", f); !errors.Is(err, ErrUnknownFormat) {
			t.Errorf("format %d : erreur %v, ErrUnknownFormat attendue", f, err)
		}
	}
}

// TestBuildProfileMPPMutationRougit — LA MUTATION.
//
// Sur `fb1a1a72` (HI_1_13_0, le build de l'executable desassemble), le decoupage du profil rend
// `n2` CONSTANT sur l'archetype 37. Le fausser d'UN SEUL BIT, dans un sens ou dans l'autre, le
// disperse. C'est ce qui rend la ligne `HI_1_13_0` du profil verifiable plutot qu'affirmee.
func TestBuildProfileMPPMutationRougit(t *testing.T) {
	rel := LockProcessDecode()
	defer rel()
	prev := CurrentMPPWidths()
	defer SetMPPWidths(prev)

	dir := filepath.Join("..", "replay", "testdata", "minifilm_fb1a1a72")
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("bobine fb1a1a72 : %v", err)
	}
	fc := NewFilmContext(film)
	var ancres []e191cAncre
	for _, p := range e191cPayloads(fc) {
		for _, b := range keyframeBornes(p) {
			if b.TI == 37 {
				ancres = append(ancres, e191cAncre{Pay: p, Bit: b.Bit})
			}
		}
	}
	if len(ancres) < 100 {
		t.Fatalf("fb1a1a72 : %d records ti=37, au moins 100 attendus — bobine a regenerer", len(ancres))
	}
	prof, err := BuildProfileFor("HI_1_13_0", 27)
	if err != nil {
		t.Fatalf("profil HI_1_13_0 : %v", err)
	}
	SetMPPWidths(prof.MPP)
	juste, _, _ := e191cN2Part(ancres, 37)
	if juste < 0.95 {
		t.Fatalf("au decoupage du profil (%s), `n2` n'est constant que sur %.3f des records — "+
			"la ligne HI_1_13_0 du profil ne tient plus", prof.MPP, juste)
	}
	for _, m := range []MPPWidths{
		{Lead: prof.MPP.Lead - 1, Index: prof.MPP.Index},
		{Lead: prof.MPP.Lead + 1, Index: prof.MPP.Index},
		{Lead: prof.MPP.Lead, Index: prof.MPP.Index - 1},
		{Lead: prof.MPP.Lead, Index: prof.MPP.Index + 1},
	} {
		SetMPPWidths(m)
		part, _, _ := e191cN2Part(ancres, 37)
		if part >= juste {
			t.Errorf("decoupage %s : `n2` constant sur %.3f des records, soit autant que le profil "+
				"(%.3f) — la mutation ne rougit pas, l'oracle ne garde rien", m, part, juste)
		}
	}
}
