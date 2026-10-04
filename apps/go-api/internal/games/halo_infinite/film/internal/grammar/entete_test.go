package grammar

// entete_test.go — L EN-TETE DE LA MARCHE : chaque parametre hors flux avec sa provenance (ADR 0037
// IR-7), et la marche construite depuis lui.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// TestLEnTeteDitLaProvenanceDeChaqueParametre : sur les sept bobines par build, la largeur de
// l identifiant bas est presumee, le decoupage MPP est celui de la table du format quand elle le
// determine, et le decoupage d i0 n est pas resolu sans carte (l auto-detection reste a la demande).
func TestLEnTeteDitLaProvenanceDeChaqueParametre(t *testing.T) {
	poses := 0
	for _, court := range closureMiniFilms() {
		film := bobineParBuild(t, court)
		h := NewFilmContext(film).EnTete()
		if h.IDLowBits != (lecture.Parametre[int]{Valeur: idLowBitsPresume, Provenance: lecture.ProvenancePresumee}) {
			t.Errorf("%s : largeur de l identifiant bas %+v", court, h.IDLowBits)
		}
		w, ok, err := mppDuFormat(film)
		switch {
		case err == nil && ok:
			poses++
			if h.MPP != (lecture.Parametre[profile.MPPWidths]{Valeur: w, Provenance: lecture.ProvenancePresumee}) {
				t.Errorf("%s : decoupage MPP %+v, attendu %+v presume", court, h.MPP, w)
			}
		case h.MPP.Provenance != lecture.ProvenanceNonRenseignee:
			t.Errorf("%s : decoupage MPP %+v pour un format qui ne le determine pas", court, h.MPP)
		}
		if h.I0.Provenance != lecture.ProvenanceNonRenseignee {
			t.Errorf("%s : decoupage d i0 %+v resolu sans carte", court, h.I0)
		}
	}
	if poses == 0 {
		t.Fatal("aucune bobine dont le format determine le decoupage MPP : le test ne prouve rien")
	}
}

// TestLEnTeteDitQuiImposeLeDecoupageDI0 : force par l appelant, il est impose ; venu du catalogue
// de la carte, il est presume.
func TestLEnTeteDitQuiImposeLeDecoupageDI0(t *testing.T) {
	film := chargerMiniBobine(t)
	entry := entreeCatalogue(t, "Cliffhanger")
	force := profile.I0Layout{GateBits: 6, AxisW: [3]uint{12, 12, 11}, Region: 1}
	for _, c := range []struct {
		nom    string
		fc     *FilmContext
		valeur profile.I0Layout
		prov   lecture.Provenance
	}{
		{"catalogue", NewFilmContextForMap(film, &entry, nil), entry.Layout(), lecture.ProvenancePresumee},
		{"force", NewFilmContextForMap(film, &entry, &force), force, lecture.ProvenanceImposee},
	} {
		if got := c.fc.EnTete().I0; got != (lecture.Parametre[profile.I0Layout]{Valeur: c.valeur, Provenance: c.prov}) {
			t.Errorf("%s : decoupage d i0 %+v, attendu %+v de provenance %d", c.nom, got, c.valeur, c.prov)
		}
	}
}

// TestLaMarcheDesTramesEstConstruiteDepuisLEnTete : la largeur de l identifiant bas sous laquelle
// la marche lit est celle de l en-tete.
func TestLaMarcheDesTramesEstConstruiteDepuisLEnTete(t *testing.T) {
	fc := contexteDeBobine(bobineDeTrames(t, "000d5950"))
	mt, err := fc.nouveauMarcheurDesTrames(nil)
	if err != nil {
		t.Fatalf("marche : %v", err)
	}
	if mt.cfg.IDLowBits != fc.EnTete().IDLowBits.Valeur {
		t.Errorf("la marche lit sous %d bits d identifiant bas, l en-tete en dit %d",
			mt.cfg.IDLowBits, fc.EnTete().IDLowBits.Valeur)
	}
}
