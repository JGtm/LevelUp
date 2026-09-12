package replay

// vehicle_tracks_clamp_test.go — LE CLAMP DES EPISODES D OCCUPATION DANS LA FENETRE DE LA VIE.
//
// D OU IL VIENT (revue adversariale 2026-09-02, MAJEUR n° 3) : pour une vie sans record de
// creation, la fenetre de tolerance du nuage de positions commence ~20 s avant la premiere
// image-cle — un trou de position peut donc dater un episode AVANT `t0`. Sur ces images, le
// calque ne dessine pas encore le vehicule mais le pion de l occupant est deja supprime :
// joueur invisible. Le clamp ramene l episode dans [t0, t1max] et ecarte ce qui est entierement
// hors fenetre.

import "testing"

func TestClampVehicleRidesRameneDansLaFenetre(t *testing.T) {
	rides := []VehicleRide{
		{T0: -18, T1: 42, XUID: "1"},  // demarre avant t0 -> tronque a t0
		{T0: 50, T1: 900, XUID: "2"},  // depasse t1max -> tronque a t1max
		{T0: -90, T1: -40, XUID: "3"}, // entierement avant la fenetre -> ecarte
		{T0: 60, T1: 70, XUID: "4"},   // deja dedans -> intact
	}
	got := clampVehicleRides(rides, 0, 800)
	if len(got) != 3 {
		t.Fatalf("clampVehicleRides : %d episodes, attendu 3 (l episode hors fenetre s ecarte)", len(got))
	}
	if got[0].T0 != 0 || got[0].T1 != 42 {
		t.Errorf("episode 1 = [%d,%d], attendu [0,42] (tronque a t0)", got[0].T0, got[0].T1)
	}
	if got[1].T0 != 50 || got[1].T1 != 800 {
		t.Errorf("episode 2 = [%d,%d], attendu [50,800] (tronque a t1max)", got[1].T0, got[1].T1)
	}
	if got[2].XUID != "4" || got[2].T0 != 60 || got[2].T1 != 70 {
		t.Errorf("episode 4 altere : %+v — un episode deja dans la fenetre ne bouge pas", got[2])
	}
	// L entree n est PAS mutee : le clamp travaille sur une copie.
	if rides[0].T0 != -18 {
		t.Errorf("l entree a ete mutee (rides[0].T0 = %d) — le clamp doit copier", rides[0].T0)
	}
}

func TestClampVehicleRidesVideEtNil(t *testing.T) {
	if got := clampVehicleRides(nil, 0, 100); got != nil {
		t.Errorf("nil -> %v, attendu nil", got)
	}
	if got := clampVehicleRides([]VehicleRide{{T0: 900, T1: 950}}, 0, 800); got != nil {
		t.Errorf("tous hors fenetre -> %v, attendu nil (pas de tranche vide publiee)", got)
	}
}
