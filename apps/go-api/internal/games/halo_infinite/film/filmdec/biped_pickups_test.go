package filmdec

// biped_pickups_test.go — LE GARDE-FOU DU PORTAGE. `ScanFilmBipedPickups` (production) et
// l'instrument de recherche (`bpkCollecte`, qui a servi à établir la grammaire et à la juger
// par l'oracle de trame) doivent voir EXACTEMENT la même chose. Sans cette confrontation, le
// chemin de production pourrait dériver en silence de ce qui a été mesuré.
//
// Garde BIPED_PICKUP_FILM — les films ne sont pas versionnés.

import "testing"

// TestScanFilmBipedPickupsMatchesInstrument confronte les deux chemins, événement par
// événement : même compte, même horodatage, même slot, même identifiant, même classe.
func TestScanFilmBipedPickupsMatchesInstrument(t *testing.T) {
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()

	want := bpkCollecte(t, f)
	got, st, err := ScanFilmBipedPickups(f.dir)
	if err != nil {
		t.Fatalf("balayage de production : %v", err)
	}
	t.Logf("PRODUCTION : paquets 0xC4=%d · type 9=%d · type 8=%d · autres=%d · publies=%d · listes multiples=%d",
		st.Packets, st.Type9, st.Type8, st.OtherType, st.Published, st.MultiEvent)
	t.Logf("REJETS : sans ref0=%d · sans identifiant=%d · slot hors bande=%d · ref1/ref2 presente=%d",
		st.RefusedNoRef, st.RefusedNoCatalog, st.RefusedOffBand, st.UnexpectedWideRef)

	// LES REJETS SONT DES SENTINELLES, pas des tolérances : aucun n'a jamais été observé non
	// nul sur le corpus de référence. Une valeur non nulle signale une largeur de runtime
	// inadaptée au film, ou un build différent — et il faut le voir tout de suite.
	if st.RefusedNoRef != 0 || st.RefusedNoCatalog != 0 || st.RefusedOffBand != 0 {
		t.Errorf("rejets non nuls (sans ref0=%d, sans identifiant=%d, hors bande=%d) : la "+
			"grammaire ne correspond pas a ce film", st.RefusedNoRef, st.RefusedNoCatalog, st.RefusedOffBand)
	}
	if st.UnexpectedWideRef != 0 {
		t.Errorf("ref1/ref2 presente sur %d evenements : jamais observe, le cadrage est suspect",
			st.UnexpectedWideRef)
	}
	if len(got) != len(want) {
		t.Fatalf("production=%d ramassages, instrument=%d : les deux chemins divergent",
			len(got), len(want))
	}
	for i := range got {
		w := want[i]
		if got[i].TimestampUS != w.TimestampUS ||
			got[i].Slot != uint32(bipedPickupRefBaseDom2+int(w.Ref0)) ||
			got[i].CatalogID != w.Objet ||
			got[i].Class != uint8(w.Kind) {
			t.Fatalf("divergence a l'index %d : production=%+v instrument={ts:%d slot:%d id:%d classe:%d}",
				i, got[i], w.TimestampUS, bipedPickupRefBaseDom2+int(w.Ref0), w.Objet, w.Kind)
		}
	}
	armes := 0
	for _, p := range got {
		if BipedPickupIsWeaponClass(p.Class) {
			armes++
		}
	}
	t.Logf("ACCORD PARFAIT sur %d ramassages · dont %d de classe ARME (%.1f %%)",
		len(got), armes, bpkPct(armes, len(got)))
}
