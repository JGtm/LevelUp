package replay

// vehicle_relays_test.go — LA FUSION DES VIES EN RELAIS (`mergeVehicleRelays`), sur fixtures.
//
// LE BUG QU'ELLE CORRIGE, RAPPELÉ ICI POUR QUE LES CAS SE LISENT : le film ne DÉPLACE pas un
// véhicule pris, il le RE-CRÉE sous un nouveau slot. La vie garée (naissance seule, aucun
// échantillon) reste affichée jusqu'à `t1max`, soit ~20 s de DOUBLE à l'ancienne place. Mesure
// du 2026-09-02 : 10 paires sur `0d76e8f1`, 3 sur `fccc61cd`, écart de position 0,00-0,01 m,
// témoin NUL à châssis différents.

import "testing"

// vrTrack monte une vie minimale : naissance a (x, y), bornes donnees, chassis donne.
func vrTrack(slot uint32, chassis string, t0, t1, t1max int, x, y float32) VehicleTrack {
	return VehicleTrack{
		Slot: slot, Gen: 1, Chassis: chassis, Family: "warthog",
		T0: t0, T1: t1, T1Max: t1max, End: VehicleEndUnknown,
		Spawn: &VehicleSpawn{X: x, Y: y},
	}
}

// TestFusionRelaisSimple — LE CAS MESURÉ : une vie garée (naissance seule) suivie, AU MÊME POINT
// et dans son intervalle non observé, d'une vie du MÊME châssis qui roule.
func TestFusionRelaisSimple(t *testing.T) {
	a := vrTrack(778, "c6e79dcc", 1000, 1293, 1493, 12, 34)
	b := vrTrack(779, "c6e79dcc", 1427, 2000, 2100, 12, 34)
	b.Samples = []VehicleSample{{T: 1427, X: 12, Y: 34}, {T: 1500, X: 20, Y: 40}}
	out, merged := mergeVehicleRelays([]VehicleTrack{a, b})
	if merged != 1 || len(out) != 1 {
		t.Fatalf("fusions = %d, vies = %d, attendu 1 et 1", merged, len(out))
	}
	got := out[0]
	if got.Slot != 778 || got.T0 != 1000 {
		t.Errorf("identite/debut = slot %d / t0 %d, attendu ceux de la vie qui COMMENCE (778, 1000)",
			got.Slot, got.T0)
	}
	if got.T1 != 2000 || got.T1Max != 2100 {
		t.Errorf("fin = t1 %d / t1max %d, attendu celle du RELAIS (2000, 2100)", got.T1, got.T1Max)
	}
	if len(got.Samples) != 2 {
		t.Errorf("echantillons = %d, attendu les 2 du relais", len(got.Samples))
	}
	if got.Spawn == nil || got.Spawn.X != 12 {
		t.Errorf("naissance = %v, attendu celle de la vie qui commence", got.Spawn)
	}
}

// TestFusionRelaisEnChaine — A -> B -> C : la vague de re-creation peut relayer plusieurs fois.
func TestFusionRelaisEnChaine(t *testing.T) {
	a := vrTrack(785, "0000254b", 100, 200, 300, 5, 5)
	b := vrTrack(791, "0000254b", 250, 400, 500, 5, 5)
	c := vrTrack(799, "0000254b", 450, 600, 700, 5, 5)
	out, merged := mergeVehicleRelays([]VehicleTrack{a, b, c})
	if merged != 2 || len(out) != 1 {
		t.Fatalf("fusions = %d, vies = %d, attendu 2 et 1 (chaine A->B->C)", merged, len(out))
	}
	if out[0].T0 != 100 || out[0].T1Max != 700 {
		t.Errorf("bornes = [%d .. %d], attendu [100 .. 700]", out[0].T0, out[0].T1Max)
	}
}

// TestFusionRefuseChassisDifferent — LE TÉMOIN DU CORRECTIF : deux véhicules DIFFÉRENTS au même
// emplacement d'apparition (le socle sert plusieurs châssis) ne se fondent JAMAIS.
func TestFusionRefuseChassisDifferent(t *testing.T) {
	a := vrTrack(797, "fe32c0f4", 100, 200, 300, 5, 5)
	b := vrTrack(802, "5b80c406", 250, 400, 500, 5, 5)
	out, merged := mergeVehicleRelays([]VehicleTrack{a, b})
	if merged != 0 || len(out) != 2 {
		t.Fatalf("fusions = %d, vies = %d, attendu 0 et 2 (chassis differents)", merged, len(out))
	}
}

// TestFusionRefuseChassisAbsent — un châssis NON LU n'autorise aucune fusion : deux vies
// anonymes au même point seraient fondues sur une coïncidence, pas sur une identite.
func TestFusionRefuseChassisAbsent(t *testing.T) {
	a := vrTrack(797, "", 100, 200, 300, 5, 5)
	b := vrTrack(802, "", 250, 400, 500, 5, 5)
	if _, merged := mergeVehicleRelays([]VehicleTrack{a, b}); merged != 0 {
		t.Fatalf("fusions = %d, attendu 0 (chassis non lu)", merged)
	}
}

// TestFusionRefuseHorsFenetre — un relais qui commence AVANT la derniere preuve de presence, ou
// APRES la premiere preuve d'absence, est une coexistence reelle : deux vehicules du meme modele
// sur la carte, pas un double.
func TestFusionRefuseHorsFenetre(t *testing.T) {
	a := vrTrack(797, "fe32c0f4", 100, 200, 300, 5, 5)
	avant := vrTrack(802, "fe32c0f4", 150, 400, 500, 5, 5) // commence avant a.T1
	apres := vrTrack(803, "fe32c0f4", 350, 400, 500, 5, 5) // commence apres a.T1Max
	for nom, b := range map[string]VehicleTrack{"avant t1": avant, "apres t1max": apres} {
		if _, merged := mergeVehicleRelays([]VehicleTrack{a, b}); merged != 0 {
			t.Errorf("%s : fusions = %d, attendu 0", nom, merged)
		}
	}
}

// TestFusionRefuseAutreEmplacement — meme chassis, meme fenetre, mais un AUTRE emplacement : ce
// sont deux vehicules, et le rayon est la pour le dire.
func TestFusionRefuseAutreEmplacement(t *testing.T) {
	a := vrTrack(797, "fe32c0f4", 100, 200, 300, 5, 5)
	b := vrTrack(802, "fe32c0f4", 250, 400, 500, 25, 5)
	if _, merged := mergeVehicleRelays([]VehicleTrack{a, b}); merged != 0 {
		t.Fatalf("fusions = %d, attendu 0 (20 m d'ecart)", merged)
	}
}

// TestFusionVagueMultiple — LA VAGUE MESURÉE : trois vies finissent a la meme image-cle et trois
// relais demarrent ensemble. Chacune se fond avec SON relais, jamais avec celui du voisin.
func TestFusionVagueMultiple(t *testing.T) {
	in := []VehicleTrack{
		vrTrack(785, "0000254b", 0, 2893, 3093, 1, 1),
		vrTrack(786, "b65b3b4a", 0, 2893, 3093, 2, 2),
		vrTrack(788, "af31ab1a", 0, 2893, 3093, 3, 3),
		vrTrack(791, "0000254b", 2926, 4000, 4100, 1, 1),
		vrTrack(792, "b65b3b4a", 2926, 4000, 4100, 2, 2),
		vrTrack(793, "af31ab1a", 2926, 4000, 4100, 3, 3),
	}
	out, merged := mergeVehicleRelays(in)
	if merged != 3 || len(out) != 3 {
		t.Fatalf("fusions = %d, vies = %d, attendu 3 et 3", merged, len(out))
	}
	for _, tr := range out {
		if tr.T1Max != 4100 {
			t.Errorf("vie %d : t1max = %d, attendu 4100 (fin du relais)", tr.Slot, tr.T1Max)
		}
	}
}

// TestFusionRecolleLesEpisodes — les episodes des DEUX vies survivent, reclampes a la fenetre
// resultante : un occupant de la vie relais ne doit pas disparaitre avec son slot.
func TestFusionRecolleLesEpisodes(t *testing.T) {
	a := vrTrack(785, "0000254b", 100, 200, 300, 5, 5)
	b := vrTrack(791, "0000254b", 250, 400, 500, 5, 5)
	b.Rides = []VehicleRide{{T0: 260, T1: 380, Slot: 42, Src: VehicleRideSrcGap}}
	out, merged := mergeVehicleRelays([]VehicleTrack{a, b})
	if merged != 1 {
		t.Fatalf("fusions = %d, attendu 1", merged)
	}
	if len(out[0].Rides) != 1 || out[0].Rides[0].Slot != 42 {
		t.Fatalf("episodes = %+v, attendu celui du relais conserve", out[0].Rides)
	}
}

// TestFusionEnchaineLesEchantillons — quand les DEUX vies ont une trajectoire, l'axe de frames
// reste strictement croissant : le client interpole entre points consecutifs, un retour en
// arriere ferait reculer le sprite.
func TestFusionEnchaineLesEchantillons(t *testing.T) {
	a := vrTrack(785, "0000254b", 100, 200, 300, 5, 5)
	a.Samples = []VehicleSample{{T: 100, X: 5, Y: 5}, {T: 200, X: 5, Y: 5}}
	b := vrTrack(791, "0000254b", 250, 400, 500, 5, 5)
	b.Samples = []VehicleSample{{T: 150, X: 5, Y: 5}, {T: 260, X: 6, Y: 6}, {T: 400, X: 7, Y: 7}}
	out, merged := mergeVehicleRelays([]VehicleTrack{a, b})
	if merged != 1 {
		t.Fatalf("fusions = %d, attendu 1", merged)
	}
	got := out[0].Samples
	if len(got) != 4 {
		t.Fatalf("echantillons = %d, attendu 4 (celui a t=150 est ANTERIEUR, il tombe)", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].T <= got[i-1].T {
			t.Fatalf("axe de frames non croissant : %+v", got)
		}
	}
}
