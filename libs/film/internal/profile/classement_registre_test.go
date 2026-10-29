package profile

// classement_registre_test.go — LA TABLE DE VERITE DES TROIS STATUTS, ET LE GEL DE LA
// POPULATION `presumee` DU CATALOGUE (lot 3.1.1-b).

import (
	"strings"
	"testing"
)

// Les empreintes du catalogue citees par les cas, nommees une fois.
const (
	empreinteReference = 0x36ca8c3d2a2f9b88 // HI_1_12_0 et HI_1_13_0
	empreinte8Et9      = 0x33c7e724716d8cc5 // HI_1_8_0 et HI_1_9_0
	empreinte41Et33    = 0x40531a0d86ce90ce // HI_1_4_1 et majeure=33
	empreinteMajeure31 = 0xba34fa35f781d1a7
	// empreinteJamaisVue : mesuree le 2026-09-17 sur `58e6f72a` (build `HI_1_5_1`, 49 blocs,
	// 1 034 slots) — le SEUL film du cache dont la grammaire de composants n est nulle part au
	// catalogue. C est le controle positif du statut `inconnue`, et il n est pas invente.
	empreinteJamaisVue = 0x8d6dec5f4fc182c1
)

// TestClassementDuRegistreTableDeVerite : les trois cas, un par un.
func TestClassementDuRegistreTableDeVerite(t *testing.T) {
	cas := []struct {
		nom       string
		build     string
		majeure   int
		empreinte uint64
		attendu   StatutRegistre
	}{
		{"build de reference, son empreinte", buildHI1131, 41, empreinteReference, StatutRegistreConnue},
		{"HI_1_12_0 partage l empreinte de reference", buildHI1120, 40, empreinteReference, StatutRegistreConnue},
		{"HI_1_8_0, son empreinte", buildHI180, 37, empreinte8Et9, StatutRegistreConnue},
		{"HI_1_9_0 partage celle de HI_1_8_0", buildHI190, 38, empreinte8Et9, StatutRegistreConnue},
		{"HI_1_4_1, son empreinte", buildHI141, 33, empreinte41Et33, StatutRegistreConnue},
		{"majeure 33 sans section", "", majeureSansSection33, empreinte41Et33, StatutRegistreConnue},
		{"majeure 31 sans section", "", majeureSansSection31, empreinteMajeure31, StatutRegistreConnue},

		{"clef connue, empreinte d une AUTRE clef", buildHI1131, 41, empreinteMajeure31, StatutRegistrePresumee},
		{"clef INCONNUE, empreinte du catalogue", "HI_9_99_0", 41, empreinteReference, StatutRegistrePresumee},
		{"majeure inconnue, empreinte du catalogue", "", 29, empreinte8Et9, StatutRegistrePresumee},

		{"clef connue, empreinte jamais vue", buildHI1131, 41, empreinteJamaisVue, StatutRegistreInconnue},
		{"clef inconnue, empreinte jamais vue", "HI_1_5_1", 41, empreinteJamaisVue, StatutRegistreInconnue},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := ClasserEmpreinteRegistre(c.build, c.majeure, c.empreinte)
			if got != c.attendu {
				t.Errorf("ClasserEmpreinteRegistre(%q, %d, 0x%016x) = %q, attendu %q",
					c.build, c.majeure, c.empreinte, got, c.attendu)
			}
		})
	}
}

// TestLesCinqEmpreintesDuCorpusPassentConnues : LE GAIN ATTENDU AU CORPUS GATE, PROUVE SANS
// DECODER.
//
// Les cinq empreintes qui sortaient `inconnue` sur les 17 temoins du lot 2.6 (§5 du plan)
// passent `connue` des lors que la table les porte. Ce test est la forme UNITAIRE de ce gain :
// il ne remplace pas le corpus gate, il dit ce que celui-ci doit trouver.
func TestLesCinqEmpreintesDuCorpusPassentConnues(t *testing.T) {
	cinq := []struct {
		build     string
		empreinte uint64
	}{
		{buildHI180, empreinte8Et9},
		{buildHI190, empreinte8Et9},
		{buildHI1100, 0x9b6397b3ad58e258},
		{buildHI1110, 0x8879e2b6746ba047},
		{buildHI141, empreinte41Et33},
	}
	for _, c := range cinq {
		if got := ClasserEmpreinteRegistre(c.build, 0, c.empreinte); got != StatutRegistreConnue {
			t.Errorf("%s / 0x%016x : statut %q, attendu %q — le gain du lot n est pas la",
				c.build, c.empreinte, got, StatutRegistreConnue)
		}
	}
}

// TestLaTableDesEmpreintesCouvreLesNeufClefs : la table des empreintes porte EXACTEMENT les
// clefs que le profil connait — les sept builds et les deux majeures sans section.
//
// C EST LE TROISIEME COTE DU TRIANGLE (cf. [TestLesTablesDuProfilPortentLesMemesClefs]) : sans
// lui, un build ajoute aux largeurs de personnalisation n aurait pas d empreinte attendue, et
// tous ses films sortiraient `presumee` sans que rien ne le dise.
func TestLaTableDesEmpreintesCouvreLesNeufClefs(t *testing.T) {
	for _, b := range buildsDeLaTable() {
		if _, ok := EmpreinteRegistreAttendue(b, 0); !ok {
			t.Errorf("build %q : connu du profil, AUCUNE empreinte de registre attendue", b)
		}
	}
	for _, m := range []int{majeureSansSection31, majeureSansSection33} {
		if _, ok := EmpreinteRegistreAttendue("", m); !ok {
			t.Errorf("majeure %d : connue du profil, AUCUNE empreinte de registre attendue", m)
		}
	}
	if n := len(EmpreintesRegistre()); n != len(buildsDeLaTable())+2 {
		t.Errorf("la table porte %d clefs, %d attendues (7 builds + 2 majeures) — une clef "+
			"ajoutee sans sa ligne ailleurs", n, len(buildsDeLaTable())+2)
	}
}

// empreintesPresumeesGelees : les clefs dont la MESURE ne couvre qu un film alors que le cache
// en porte d autres, gelees au 2026-09-17 (statut `presumee` du catalogue).
//
// CE RATCHET NE MONTE PAS. Une clef qui entre ici est une clef dont on sait moins qu avant ; une
// clef qui en sort le fait avec la lecture d un second film et la mise a jour de sa ligne, dans
// le catalogue ET dans la table. C est le pendant, pour les empreintes, de `presumesGeles`.
var empreintesPresumeesGelees = []string{
	"build=HI_1_12_0",
	"build=HI_1_11_0",
	"build=HI_1_8_0",
	"majeure=33",
	"majeure=31",
}

// TestEmpreintesPresumeesGelees LISTE les empreintes dont la couverture est PRESUMEE, une par
// une, et gele leur liste.
func TestEmpreintesPresumeesGelees(t *testing.T) {
	var vues []string
	for _, e := range EmpreintesRegistre() {
		if e.Statut != StatutCataloguePresumee {
			continue
		}
		vues = append(vues, e.Cle)
		t.Logf("PRESUMEE  %-18s 0x%016x  %d temoin(s) : %v", e.Cle, e.Empreinte, len(e.Temoins),
			e.Temoins)
	}
	if strings.Join(vues, "\n") != strings.Join(empreintesPresumeesGelees, "\n") {
		t.Errorf("la liste des empreintes PRESUMEES a bouge.\nobtenu :\n  %s\ngele :\n  %s\n"+
			"En ajouter une est une regression de connaissance ; en retirer une se fait avec la "+
			"lecture d un second film de la clef, dans le catalogue ET dans la table.",
			strings.Join(vues, "\n  "), strings.Join(empreintesPresumeesGelees, "\n  "))
	}
}
