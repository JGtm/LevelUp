package persist

// vehicle_takes_persister_validate_test.go — ce que `ValidateVehicleTakesBatch` REFUSE (sans base).
// Un test par controle ; chacun est rouge sous la mutation qui retire le controle.

import (
	"strings"
	"testing"
)

func passeVehiculesValide() VehicleTakesBatch {
	return VehicleTakesBatch{
		MatchID: "m1", Measured: true, DocSchema: 71, EpisodesRead: 3, EpisodesNoXUID: 1,
		FragsRead: true, FragsTotal: 3, FragsUnmatched: 1,
		Rows: []VehicleTakeRow{
			{Camp: 0, XUID: "a1", Family: "warthog", Takes: 1, AboardMS: 1000, Episodes: 1, Frags: 1},
			{Camp: 1, XUID: "b1", Family: "ghost", Takes: 1, AboardMS: 500, Episodes: 2, ProximityEpisodes: 1, Frags: 1},
		},
	}
}

func TestValidateVehicleTakesBatch_AccepteUnePasseCoherente(t *testing.T) {
	if err := ValidateVehicleTakesBatch(passeVehiculesValide()); err != nil {
		t.Fatalf("passe coherente refusee : %v", err)
	}
}

func TestValidateVehicleTakesBatch_AccepteUnePasseNonMesuree(t *testing.T) {
	in := VehicleTakesBatch{MatchID: "m1", Reason: "schema_before_67", DocSchema: 61, FragsReason: "takes_not_measured"}
	if err := ValidateVehicleTakesBatch(in); err != nil {
		t.Fatalf("passe non mesuree refusee : %v", err)
	}
}

func TestValidateVehicleTakesBatch_Refuse(t *testing.T) {
	cas := []struct {
		nom    string
		mutate func(*VehicleTakesBatch)
		motif  string
	}{
		{"match vide", func(b *VehicleTakesBatch) { b.MatchID = "" }, "MatchID vide"},
		{"non mesure sans raison", func(b *VehicleTakesBatch) { b.Measured, b.Rows = false, nil }, "sans raison"},
		{"non mesure avec des lignes", func(b *VehicleTakesBatch) { b.Measured, b.Reason = false, "x" }, "avec des lignes"},
		{"mesure avec une raison", func(b *VehicleTakesBatch) { b.Reason = "x" }, "raison de non mesure"},
		{"xuid vide", func(b *VehicleTakesBatch) { b.Rows[0].XUID = "" }, "XUID vide"},
		{"famille vide", func(b *VehicleTakesBatch) { b.Rows[0].Family = "" }, "famille vide"},
		{"camp hors bornes", func(b *VehicleTakesBatch) { b.Rows[0].Camp = 9 }, "hors 0..8"},
		{"camp negatif", func(b *VehicleTakesBatch) { b.Rows[0].Camp = -1 }, "hors 0..8"},
		{"compte negatif", func(b *VehicleTakesBatch) { b.Rows[0].AboardMS = -1 }, "negatif"},
		{"ligne sans episode", func(b *VehicleTakesBatch) { b.Rows[0].Episodes = 0 }, "sans episode"},
		{"proximite en trop", func(b *VehicleTakesBatch) { b.Rows[0].ProximityEpisodes = 2 }, "proximite"},
		{"doublon", func(b *VehicleTakesBatch) { b.Rows[1] = b.Rows[0] }, "doublon"},
		{"frags lus avec raison", func(b *VehicleTakesBatch) { b.FragsReason = "x" }, "raison de non lecture"},
		{"frags non lus sans raison", func(b *VehicleTakesBatch) { b.FragsRead = false }, "sans raison"},
		{"frags non lus mais comptes", func(b *VehicleTakesBatch) { b.FragsRead, b.FragsReason = false, "x" }, "non nuls"},
		{"non apparies > lus", func(b *VehicleTakesBatch) { b.FragsUnmatched = 4 }, "incoherents"},
		{"somme des lignes != total - non apparies", func(b *VehicleTakesBatch) { b.Rows[0].Frags = 2 }, "frags sur les lignes"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			in := passeVehiculesValide()
			c.mutate(&in)
			err := ValidateVehicleTakesBatch(in)
			if err == nil || !strings.Contains(err.Error(), c.motif) {
				t.Fatalf("refus attendu (%q), obtenu : %v", c.motif, err)
			}
		})
	}
}

func TestValidateVehicleTakesBatch_LesFragsNonLusNePortentAucunFragParLigne(t *testing.T) {
	in := passeVehiculesValide()
	in.FragsRead, in.FragsReason, in.FragsTotal, in.FragsUnmatched = false, "no_kill_source", 0, 0
	// les lignes portent encore 1 frag chacune : incoherent
	err := ValidateVehicleTakesBatch(in)
	if err == nil || !strings.Contains(err.Error(), "n ont pas ete lus") {
		t.Fatalf("refus attendu : %v", err)
	}
}
