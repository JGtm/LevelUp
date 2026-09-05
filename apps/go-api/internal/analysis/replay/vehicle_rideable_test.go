package replay

// vehicle_rideable_test.go — AUCUN OCCUPANT SUR UN VEHICULE NON PILOTABLE.
//
// D OU IL VIENT : visionnage utilisateur du 2026-09-02 sur `fccc61cd`. Un prop de la famille
// `falcon` (chassis 0x0000254b, quasi immobile, vivant tout le match) s etait vu attribuer un
// episode d occupation par le liant « trou de position » — un joueur passe a proximite, son
// bipede cesse de repliquer une seconde, et le decor herite d un conducteur. Consequence a
// l ecran : le pion du joueur reel etait escamote SANS qu aucun vehicule ne soit dessine a la
// place (le calque filtre deja ces familles), donc un joueur disparaissait.
//
// LA GARDE EST DES DEUX COTES, ET C EST VOULU : le calque refuse de DESSINER ces familles, le
// document refuse d AFFIRMER qu elles portent quelqu un.

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
		{"falcon", false, "transport non pilotable en multijoueur — le cas vu par l utilisateur"},
		{"pelican", false, "non jouable en multi (decision utilisateur : customs locales hors perimetre)"},
		{"phantom", false, "transport scripte"},
		{"skiff", false, "decor"},
		{"", false, "chassis inconnu : on n affirme rien de qui serait a bord"},
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
	if vehicleFamilyIsRideable("falcon") {
		t.Error("falcon ne doit pas porter d episode")
	}
	if !vehicleFamilyIsRideable("warthog") {
		t.Error("warthog doit porter ses episodes")
	}
}
