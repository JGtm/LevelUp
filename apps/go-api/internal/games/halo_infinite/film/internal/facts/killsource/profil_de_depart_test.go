package killsource

// profil_de_depart_test.go — UNE CARTE FOURNIE EST UNE CARTE APPLIQUEE, MEME QUAND ELLE EGALE
// L INVARIANT (lot J7.7 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat FK-7).
//
// [ProfilDeDepartPourCarte] rendait « appliquee » quand les largeurs du profil CHANGEAIENT. Sur
// Cliffhanger — dont l entree de catalogue EST l invariant du profil — rien ne change, donc la carte
// passait pour absente : faux avertissement « carte du match absente » a chaque decodage, et le
// repli `repli_carte_absente_largeurs_par_defaut` declare alors que la carte avait ete lue.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : rendre de nouveau `p.LargeursObjetDuMonde() != avant`.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestProfilDeDepartPourCarte_CarteEgaleALInvariantEstAppliquee(t *testing.T) {
	inv := ProfilDeDepart().LargeursObjetDuMonde()
	carte := profile.MapQuantEntry{Module: "invariant", AxisWidths: inv.AxisW, Region: inv.Region,
		RegionIndexBits: inv.IndexW}
	if carte.PrecisionAbsolue() != inv {
		t.Fatalf("temoin sans valeur : l entree %+v ne rend pas l invariant %+v", carte.PrecisionAbsolue(), inv)
	}
	p, appliquee := ProfilDeDepartPourCarte(&carte)
	if !appliquee {
		t.Error("carte fournie, largeurs egales a l invariant : rendue NON appliquee — faux repli " +
			"`repli_carte_absente_largeurs_par_defaut` et faux avertissement a chaque decodage")
	}
	if p.LargeursObjetDuMonde() != carte.PrecisionAbsolue() {
		t.Errorf("le profil porte %+v, la carte impose %+v", p.LargeursObjetDuMonde(), carte.PrecisionAbsolue())
	}
}

// TestProfilDeDepartPourCarte_SansLargeursNEstPasAppliquee : les deux cas du repli restent le repli.
func TestProfilDeDepartPourCarte_SansLargeursNEstPasAppliquee(t *testing.T) {
	if _, appliquee := ProfilDeDepartPourCarte(nil); appliquee {
		t.Error("sans carte : rendue appliquee")
	}
	sansLargeurs := profile.MapQuantEntry{Module: "entree fabriquee a la main"}
	if _, appliquee := ProfilDeDepartPourCarte(&sansLargeurs); appliquee {
		t.Error("entree sans largeurs : rendue appliquee alors que l invariant est conserve")
	}
}

// TestCarteLueSurCliffhangerDuCatalogue : LE CAS DE L ENQUETE DU 2026-09-27 (constat 6b), sur
// l entree COMMISE de Cliffhanger et la bobine de ce film — pas sur une entree fabriquee. La
// presence de la carte se lit sur l ENTREE : Cliffhanger passee est une carte LUE, meme si ses
// largeurs sont exactement l invariant du profil. Avant J7.7, `CarteLue` y valait faux : faux
// avertissement, faux repli compte, « DEFAUT (carte absente) » dans la calibration de 000d5950.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : rendre de nouveau « appliquee » sur une DIFFERENCE de largeurs
// (`p.LargeursObjetDuMonde() != avant`).
func TestCarteLueSurCliffhangerDuCatalogue(t *testing.T) {
	cliff := carteDuCatalogue(t, "Cliffhanger")
	if cliff.PrecisionAbsolue() != ProfilDeDepart().LargeursObjetDuMonde() {
		t.Fatalf("temoin sans valeur : Cliffhanger %+v n EST PLUS l invariant %+v",
			cliff.PrecisionAbsolue(), ProfilDeDepart().LargeursObjetDuMonde())
	}
	if _, lue := ProfilDeDepartPourCarte(&cliff); !lue {
		t.Error("ProfilDeDepartPourCarte(Cliffhanger) : carte rendue NON lue alors qu elle est passee")
	}
	src, err := source.LoadDir(miniBobineDir, nil)
	if err != nil {
		t.Fatalf("mini-bobine illisible : %v", err)
	}
	if c := calibrationDeLaBobine(t, src, &cliff); !c.CarteLue {
		t.Errorf("calibration de %s sous Cliffhanger : %s — la carte passee se lit absente", miniBobineFilm, c)
	}
}
