package filmprofile_test

// catalogue_commis_test.go — LE FICHIER QUI EST DANS L ARBRE EST VALIDE, ET SOLIDAIRE.
//
// Deux gardes, une par nature d entree du catalogue :
//
//	SAISIE   le fichier commis se charge et se valide (schema, clefs analysables, paires
//	         uniques, provenances connues, preuves non vides, dates datees).
//	DERIVEE  son bloc `derived` decrit bien le catalogue de bornes qui est dans l arbre a
//	         cote de lui. C est la moitie SANS jeu installe du gate « catalogue commis =
//	         catalogue regenere » ; l autre moitie (les bornes elles-memes regenerees depuis
//	         les .module) est le test gamefiles de `cmd/film-profiles-build`.
//
// La racine vient de `testutil.RepoRoot()` : un fichier VERSIONNE absent est une installation
// cassee, pas un cas a skipper (ratchet archlint TestNoProdRepoRootHelperInTests).

import (
	"os"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/filmprofile"
	"levelup/go-api/internal/testutil"
)

// cartesDerivesPlancher : le catalogue de bornes commis en portait 79 le 2026-09-16. Un
// catalogue qui en rendrait moins a perdu des cartes en silence.
const cartesDerivesPlancher = 79

func chargerCatalogueCommis(t *testing.T) (*filmprofile.Catalogue, string) {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	res := title.NewPathResolver(racine)
	cat, err := filmprofile.Charger(res.FilmProfilesPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("le catalogue commis ne se charge pas : %v", err)
	}
	return cat, res.MapQuantBoundsPath(title.DefaultSlug)
}

// TestCatalogueCommisEstValide — la part SAISIE.
func TestCatalogueCommisEstValide(t *testing.T) {
	cat, _ := chargerCatalogueCommis(t)
	if cat.TitleSlug != title.DefaultSlug {
		t.Errorf("titleSlug %q, attendu %q", cat.TitleSlug, title.DefaultSlug)
	}
	if cat.SchemaVersion != filmprofile.SchemaVersionCourante {
		t.Errorf("schemaVersion %d, attendu %d", cat.SchemaVersion, filmprofile.SchemaVersionCourante)
	}
	// Chaque sorte de clef du film est effectivement representee : un catalogue qui aurait
	// perdu une des trois clefs se chargerait sans broncher.
	sortes := map[filmprofile.SorteDeCle]int{}
	for _, e := range cat.Entrees {
		cle, err := filmprofile.ParseCle(e.Cle)
		if err != nil {
			t.Fatalf("clef %q refusee apres validation — incoherence : %v", e.Cle, err)
		}
		sortes[cle.Sorte]++
	}
	for _, sorte := range []filmprofile.SorteDeCle{filmprofile.SorteFormat, filmprofile.SorteBuild,
		filmprofile.SorteMajeure, filmprofile.SorteToutes} {
		if sortes[sorte] == 0 {
			t.Errorf("aucune entree de sorte %q — le catalogue a perdu une des clefs que le "+
				"film ecrit", sorte)
		}
	}
}

// TestBlocDeriveSolidaireDesBornesCommises — la part DERIVEE, sans le jeu.
func TestBlocDeriveSolidaireDesBornesCommises(t *testing.T) {
	cat, cheminBornes := chargerCatalogueCommis(t)
	blob, err := os.ReadFile(cheminBornes)
	if err != nil {
		t.Fatalf("lecture du catalogue de bornes commis (%s) : %v", cheminBornes, err)
	}
	resume, err := filmprofile.ResumerBornesDeCarte(blob)
	if err != nil {
		t.Fatalf("resume des bornes : %v", err)
	}
	if resume.Cartes < cartesDerivesPlancher {
		t.Errorf("%d carte(s) dans le catalogue de bornes, plancher %d (mesure du 2026-09-16)",
			resume.Cartes, cartesDerivesPlancher)
	}
	if err := cat.Derive.CorrespondAuxBornes(resume); err != nil {
		t.Errorf("%v", err)
	}
}

// TestEmpreinteDesBornesIgnoreLaTraceDeFabrication — l empreinte porte sur la DONNEE.
//
// Le champ `source` du catalogue de bornes cite le chemin d installation de la machine qui l a
// produit. S il entrait dans l empreinte, le gate serait rouge d un poste a l autre sans qu une
// seule borne ait bouge — et on l aurait « corrige » en le desactivant.
func TestEmpreinteDesBornesIgnoreLaTraceDeFabrication(t *testing.T) {
	const carte = `{"module":"ctf_aquarius","min":[-1,-2,-3],"max":[1,2,3],"axisWidths":[15,15,17]}`
	avec := []byte(`{"schemaVersion":1,"source":"D:\\SteamLibrary\\...","maps":{"aquarius":` + carte + `}}`)
	sans := []byte(`{"schemaVersion":1,"source":"C:\\ailleurs","maps":{"aquarius":` + carte + `}}`)
	reindente := []byte("{\n \"schemaVersion\": 1,\n \"maps\": {\n  \"aquarius\": " + carte + "\n }\n}")

	ra, err := filmprofile.ResumerBornesDeCarte(avec)
	if err != nil {
		t.Fatalf("resume (avec source) : %v", err)
	}
	rs, err := filmprofile.ResumerBornesDeCarte(sans)
	if err != nil {
		t.Fatalf("resume (autre source) : %v", err)
	}
	rr, err := filmprofile.ResumerBornesDeCarte(reindente)
	if err != nil {
		t.Fatalf("resume (reindente) : %v", err)
	}
	if ra.Empreinte != rs.Empreinte || ra.Empreinte != rr.Empreinte {
		t.Errorf("l empreinte depend de la trace de fabrication ou de l indentation : %s / %s / %s",
			ra.Empreinte, rs.Empreinte, rr.Empreinte)
	}

	// ... et elle mord bien sur la donnee.
	autre := []byte(`{"schemaVersion":1,"maps":{"aquarius":{"module":"ctf_aquarius","min":[-1,-2,-3],` +
		`"max":[1,2,4],"axisWidths":[15,15,17]}}}`)
	rd, err := filmprofile.ResumerBornesDeCarte(autre)
	if err != nil {
		t.Fatalf("resume (borne changee) : %v", err)
	}
	if rd.Empreinte == ra.Empreinte {
		t.Error("une borne changee ne change pas l empreinte — elle ne garde rien")
	}
}
