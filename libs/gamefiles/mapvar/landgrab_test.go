package mapvar

// landgrab_test.go — GATE C4.2 (lot C catalogues, 2026-08-27) : le role `landgrab_zone`
// SORT du decodeur, formes comprises, sur une carte porteuse.
//
// La verification est STATIQUE (aucun film Land Grab n'existe — expire ; le cablage sert
// les matchs futurs) : la fixture versionnee `cliffhanger_map.mvar` porte 9 volumes
// [landgrab_include, landgrab_zone] et 9 marqueurs [landgrab_include, -941529218] — le
// motif volume+marqueur de Bastion et de KOTH. Les noms sont resolus par la chasse murmur3
// rejouable (TestHuntLabels) et la table s'auto-verifie (TestLabelTableIsSelfConsistent :
// murmur3 de chaque nom recalcule).

import "testing"

func TestLandGrabZonesCliffhanger(t *testing.T) {
	v, err := Parse(fixture(t, "cliffhanger_map.mvar"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	var zones []Objective
	for _, o := range v.Objectives() {
		if o.Role == RoleLandGrabZone {
			zones = append(zones, o)
		}
	}
	if len(zones) != 9 {
		t.Fatalf("landgrab_zone = %d objectifs, attendu 9 (le vivier declare par le fichier)", len(zones))
	}
	for i, z := range zones {
		if z.Shape == nil {
			t.Errorf("zone %d (obj %d) : SANS forme — une zone de Land Grab est une aire, sa forme est la donnee", i, z.ObjectIdx)
		}
	}
	// Le marqueur ponctuel (-941529218) n'attribue AUCUN role : il reste non resolu, comme
	// ceux de Bastion et de KOTH. S'il entrait un jour dans labelNames sans role, ce compte
	// ne bougerait pas ; s'il etait par erreur promu role, le vivier doublerait — d'ou le 9
	// EXACT ci-dessus.
	t.Logf("cliffhanger : %d zones landgrab_zone, toutes avec forme", len(zones))
}
