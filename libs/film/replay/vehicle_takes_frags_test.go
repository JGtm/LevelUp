package replay

// vehicle_takes_frags_test.go — UN TEST PAR REGLE de `PairVehicleFrags` (plan Emprise vehicules,
// lot L7.2, decision D9). Pas d image 100 ms, origine 5000 ms : un frag a T ms est a la frame
// (T - 5000) / 100.

import "testing"

func tkPairDoc(vehicles ...VehicleTrack) *ReplayDocument {
	d := tkDoc(vehicles...)
	o := int64(5000)
	d.OriginMs = &o
	return d
}

func tkPair(t *testing.T, d *ReplayDocument, kills ...VehicleFragRef) (VehicleTakesReport, VehicleFragsCoverage) {
	t.Helper()
	rep := ProjectVehicleTakes(d)
	return rep, PairVehicleFrags(d, &rep, kills, true)
}

func TestPairVehicleFrags_UnFragPendantLEpisodeDuTueurEstApparie(t *testing.T) {
	d := tkPairDoc(tkLife(1, "warthog", tkRide(tkA1, 10, 20)))
	rep, cov := tkPair(t, d, VehicleFragRef{XUID: tkA1, TimeMS: 5000 + 1500})
	row, _ := tkRow(rep, tkA1, "warthog")
	if row.Frags != 1 || cov.Matched != 1 || cov.Unmatched != 0 || cov.Total != 1 || !cov.Read {
		t.Fatalf("row=%+v cov=%+v", row, cov)
	}
}

func TestPairVehicleFrags_LesBornesDeLEpisodeSontIncluses(t *testing.T) {
	d := tkPairDoc(tkLife(1, "warthog", tkRide(tkA1, 10, 20)))
	_, cov := tkPair(t, d,
		VehicleFragRef{XUID: tkA1, TimeMS: 5000 + 1000}, VehicleFragRef{XUID: tkA1, TimeMS: 5000 + 2000})
	if cov.Matched != 2 {
		t.Fatalf("T0 et T1 doivent compter : %+v", cov)
	}
}

func TestPairVehicleFrags_HorsEpisodeOuMauvaisTueurNonApparie(t *testing.T) {
	d := tkPairDoc(tkLife(1, "warthog", tkRide(tkA1, 10, 20)))
	rep, cov := tkPair(t, d,
		VehicleFragRef{XUID: tkA1, TimeMS: 5000 + 2100}, // apres
		VehicleFragRef{XUID: tkA1, TimeMS: 5000 + 900},  // avant
		VehicleFragRef{XUID: tkB1, TimeMS: 5000 + 1500}) // un autre tueur
	if cov.Matched != 0 || cov.Unmatched != 3 {
		t.Fatalf("cov=%+v", cov)
	}
	if row, _ := tkRow(rep, tkA1, "warthog"); row.Frags != 0 {
		t.Fatalf("aucun frag a poser : %+v", row)
	}
}

func TestPairVehicleFrags_LeFragVaALaFamilleDeSonEpisode(t *testing.T) {
	d := tkPairDoc(
		tkLife(1, "warthog", tkRide(tkA1, 10, 20)),
		tkLife(2, "ghost", tkRide(tkA1, 30, 40)))
	rep, _ := tkPair(t, d, VehicleFragRef{XUID: tkA1, TimeMS: 5000 + 3500}, VehicleFragRef{XUID: tkA1, TimeMS: 5000 + 1500})
	w, _ := tkRow(rep, tkA1, "warthog")
	g, _ := tkRow(rep, tkA1, "ghost")
	if w.Frags != 1 || g.Frags != 1 {
		t.Fatalf("warthog=%+v ghost=%+v", w, g)
	}
}

func TestPairVehicleFrags_UnFragNeCompteQuUneFoisSiDeuxEpisodesLeCouvrent(t *testing.T) {
	d := tkPairDoc(
		tkLife(1, "warthog", tkRide(tkA1, 10, 30)),
		tkLife(2, "ghost", tkRide(tkA1, 20, 40)))
	rep, cov := tkPair(t, d, VehicleFragRef{XUID: tkA1, TimeMS: 5000 + 2500})
	total := 0
	for _, r := range rep.Rows {
		total += r.Frags
	}
	if total != 1 || cov.Matched != 1 {
		t.Fatalf("total=%d cov=%+v", total, cov)
	}
	// le premier episode dans l ordre de montee l emporte (la vie du warthog, slot 1)
	if w, _ := tkRow(rep, tkA1, "warthog"); w.Frags != 1 {
		t.Fatalf("le frag doit aller a la premiere vie montee : %+v", rep.Rows)
	}
}

func TestPairVehicleFrags_UnEpisodeSansXUIDNeCouvreAucunFrag(t *testing.T) {
	d := tkPairDoc(tkLife(1, "warthog", tkRide("", 10, 20)))
	_, cov := tkPair(t, d, VehicleFragRef{XUID: "", TimeMS: 5000 + 1500})
	if cov.Matched != 0 || cov.Unmatched != 1 {
		t.Fatalf("cov=%+v", cov)
	}
}

func TestPairVehicleFrags_SansOrigineOuSansLectureRienNEstApparie(t *testing.T) {
	d := tkDoc(tkLife(1, "warthog", tkRide(tkA1, 10, 20))) // pas d origine
	rep := ProjectVehicleTakes(d)
	cov := PairVehicleFrags(d, &rep, []VehicleFragRef{{XUID: tkA1, TimeMS: 1500}}, true)
	if cov.Read || cov.Reason != VehicleFragsNoOrigin {
		t.Fatalf("sans origine : %+v", cov)
	}
	d2 := tkPairDoc(tkLife(1, "warthog", tkRide(tkA1, 10, 20)))
	rep2 := ProjectVehicleTakes(d2)
	cov2 := PairVehicleFrags(d2, &rep2, nil, false)
	if cov2.Read || cov2.Reason != VehicleFragsNoSource {
		t.Fatalf("sans evenements de mort : %+v", cov2)
	}
}

func TestPairVehicleFrags_UnDocumentNonMesureNApparieRien(t *testing.T) {
	d := tkPairDoc()
	d.SchemaVersion = 61
	rep := ProjectVehicleTakes(d)
	cov := PairVehicleFrags(d, &rep, []VehicleFragRef{{XUID: tkA1, TimeMS: 6000}}, true)
	if cov.Read {
		t.Fatalf("un document non mesure ne s apparie pas : %+v", cov)
	}
}

func TestPairVehicleFrags_ZeroFragEngineSurUnDocumentLuEstUnZeroMesure(t *testing.T) {
	d := tkPairDoc(tkLife(1, "warthog", tkRide(tkA1, 10, 20)))
	_, cov := tkPair(t, d)
	if !cov.Read || cov.Total != 0 {
		t.Fatalf("cov=%+v", cov)
	}
}
