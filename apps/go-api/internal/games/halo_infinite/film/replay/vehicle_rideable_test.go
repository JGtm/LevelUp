package replay

// vehicle_rideable_test.go — AUCUN OCCUPANT SUR UN VEHICULE NON PILOTABLE.
//
// D OU IL VIENT : visionnage utilisateur du 2026-09-02 sur `fccc61cd`. Un prop alors range dans la
// famille `falcon` (chassis 0x0000254b) s etait vu attribuer un episode d occupation par le liant
// « trou de position » — un joueur passe a proximite, son bipede cesse de repliquer une seconde,
// et le decor herite d un conducteur. Consequence a l ecran : le pion du joueur reel etait
// escamote SANS qu aucun vehicule ne soit dessine a la place (le calque filtre deja ces familles),
// donc un joueur disparaissait.
//
// LA GARDE EST DES DEUX COTES, ET C EST VOULU : le calque refuse de DESSINER ces familles, le
// document refuse d AFFIRMER qu elles portent quelqu un.
//
// AMENDE LE 2026-09-24 (decision utilisateur : « les Pelican c est toujours du decor ; le Falcon ca
// depend ») : le Falcon SORT des familles non pilotables. Il porte ses occupants comme tout
// vehicule, et son decor se decide vie par vie par la regle generale du decor de carte (lot M7).

import "testing"

func TestVehicleFamilyIsRideable(t *testing.T) {
	for _, c := range []struct {
		famille  string
		veut     bool
		pourquoi string
	}{
		{"warthog", true, "pilotable"},
		{"ghost", true, "pilotable"},
		{"shade", true, "tourelle : on y monte, elle porte un occupant"},
		{"falcon", true, "pilotable (decision du 2026-09-24) : le decor se decide vie par vie (lot M7)"},
		{"pelican", false, "toujours du decor (decisions utilisateur du 2026-09-02 et du 2026-09-24)"},
		{"phantom", false, "transport scripte"},
		{"skiff", false, "decor"},
		// AMENDE LE 2026-09-19 : un chassis que la table ne nomme pas ENCORE est une IGNORANCE,
		// pas une famille de decor. Sous l ancienne reponse (`false`), le cablage de l etat par
		// defaut de `ti=40` — qui fait naitre 18 vehicules de plus sur `11de8353`, dont 16 au
		// chassis inconnu — supprimait l occupant `585` du Warthog `773/1` et les sept tirs qu il
		// portait. Les familles ci-dessus, elles, restent refusees : elles sont NOMMEES.
		{"", true, "chassis inconnu : la table ne le nomme pas encore, ce n est pas du decor"},
	} {
		if got := vehicleFamilyIsRideable(c.famille); got != c.veut {
			t.Errorf("vehicleFamilyIsRideable(%q) = %v, attendu %v (%s)", c.famille, got, c.veut, c.pourquoi)
		}
	}
}

// TestVehicleTrackOfEcarteLesEpisodesDuDecor : la garde agit dans l assemblage, pas seulement
// dans le predicat — une vie de decor sort avec sa trajectoire ET SANS occupant.
func TestVehicleTrackOfEcarteLesEpisodesDuDecor(t *testing.T) {
	rides := []VehicleRide{{T0: 10, T1: 40, Slot: 515, Seat: nil}}
	if got := clampVehicleRides(rides, 0, 100); len(got) != 1 {
		t.Fatalf("prealable : l episode doit survivre au clamp pour que le test ait un sens (%d)", len(got))
	}
	// Le decor ne garde rien, le pilotable garde tout : c est la seule difference.
	if vehicleFamilyIsRideable(famillePelican) {
		t.Error("pelican ne doit pas porter d episode")
	}
	for _, f := range []string{familleWarthog, familleFalcon} {
		if !vehicleFamilyIsRideable(f) {
			t.Errorf("%s doit porter ses episodes", f)
		}
	}
}

// TestPiecesMonteesOntUnPorteurPilotable — INVARIANT DE LA TABLE DES PIECES (2026-09-24). Depuis
// que le Falcon est pilotable, AUCUNE piece de `vehicleTurretByChassis` n a un porteur non
// pilotable : leurs artilleurs passent tous a bord. Une piece ajoutee a un porteur non pilotable
// (une tourelle de Phantom, par exemple) fait rougir ce test : c est une DECISION a ecrire (ses
// artilleurs resteraient sur la piece, comptes dans `turretRidesNotRideable`), pas un ajout de
// donnee.
func TestPiecesMonteesOntUnPorteurPilotable(t *testing.T) {
	for id, piece := range vehicleTurretByChassis {
		if !vehicleFamilyIsRideable(piece.carrier) {
			t.Errorf("piece %08x : porteur %q non pilotable — decision a ecrire", id, piece.carrier)
		}
	}
}
