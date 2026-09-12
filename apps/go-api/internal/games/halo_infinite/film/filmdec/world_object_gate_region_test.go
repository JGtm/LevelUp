package filmdec

// world_object_gate_region_test.go — LA PORTE D'i0 OBJET DU MONDE SUIT LA CARTE, PAS UN LITTÉRAL.
//
// LE DÉFAUT CORRIGÉ (lot B-bis, 2026-09-12). `decodeWorldObjectPos` et `projPosBits` écrivaient
// la porte d'i0 en dur, à 3 bits : 1 precHigh + 1 index-sel + 1 index de région. Cette dernière
// largeur est une CONSTANTE PAR CARTE (`ceilLog2(nb de régions)`, champ `regionIndexBits` du
// catalogue de bornes), et elle vaut 2 sur Live Fire. Sur cette carte, le décodeur consommait
// donc un bit de trop peu et lisait les TROIS axes un bit trop tôt : le bit de poids fort de Y
// devenait le bit de poids faible de X, celui de Z le bit de poids faible de Y. Un bit de poids
// faible bascule d'une image à l'autre — d'où un pas de la MOITIÉ de l'étendue de l'axe
// (31,89 m sur Live Fire, étendue Y 63,775 m), à chaque bascule.
//
// Le jumeau BIPÈDE (`decodeBipedI0Pos`, vehicle_creation.go) faisait déjà la bonne chose : il
// compare l'index de région lu à la région ATTENDUE de la carte. Ce sont deux écritures du même
// champ, et l'une des deux avait divergé. Ce fichier est le garde-rail qui interdit le retour du
// littéral.

import "testing"

// wogInstalle pose un descripteur world-object et rend sa restauration.
func wogInstalle(indexW uint, region uint32, axisW [3]uint) func() {
	prev := WorldObjectPrecision
	WorldObjectPrecision = PrecisionDescriptor{IndexW: indexW, Region: region, AxisW: axisW}
	return func() { WorldObjectPrecision = prev }
}

// wogRecord écrit un i0 d'objet du monde : porte (precHigh, index-sel, index de région) puis
// les trois quanta aux largeurs données.
func wogRecord(region uint32, indexW int, axisW [3]uint, q [3]uint64) []byte {
	w := &bitw{}
	w.put(0, 2) // precHigh, index-sel
	w.put(uint64(region), indexW)
	for a := 0; a < 3; a++ {
		w.put(q[a], int(axisW[a]))
	}
	w.pad(24)
	return w.buf
}

// TestPorteWorldObjectSuitLaLargeurDIndexDeRegion : sur une carte à 2 bits d'index (Live Fire),
// la porte fait 4 bits et les axes commencent APRÈS elle.
func TestPorteWorldObjectSuitLaLargeurDIndexDeRegion(t *testing.T) {
	axisW := [3]uint{12, 12, 11}
	defer wogInstalle(2, 1, axisW)()

	if got, want := projPosBits(), 4+12+12+11+2; got != want {
		t.Fatalf("projPosBits() = %d, attendu %d : la porte ne suit pas l'index de région", got, want)
	}
	wr := Vec3Range{{Min: 0, Max: 4096}, {Min: 0, Max: 4096}, {Min: 0, Max: 2048}}
	q := [3]uint64{1000, 2000, 500}
	pay := wogRecord(1, 2, axisW, q)
	v, ok := decodeWorldObjectPos(pay, 0, &wr)
	if !ok {
		t.Fatal("un record de la région JOUÉE a été refusé")
	}
	// Bornes choisies pour que la valeur rendue vaille q + 0,5 : toute lecture décalée d'un bit
	// doublerait ou halverait le quantum, et se verrait au premier chiffre.
	for a, want := range [3]float32{1000.5, 2000.5, 500.5} {
		if v[a] != want {
			t.Errorf("axe %d : %.3f, attendu %.3f — la lecture est décalée", a, v[a], want)
		}
	}
}

// TestPorteWorldObjectRefuseUneAutreRegion : un record d'une région NON jouée est écarté. Ses
// quanta sont exprimés dans une autre AABB ; les déquantifier ici rendrait une position fausse
// silencieuse — le pire des résultats.
func TestPorteWorldObjectRefuseUneAutreRegion(t *testing.T) {
	axisW := [3]uint{12, 12, 11}
	defer wogInstalle(2, 1, axisW)()
	wr := Vec3Range{{Min: 0, Max: 4096}, {Min: 0, Max: 4096}, {Min: 0, Max: 2048}}
	for _, region := range []uint32{0, 2, 3} {
		pay := wogRecord(region, 2, axisW, [3]uint64{1000, 2000, 500})
		if _, ok := decodeWorldObjectPos(pay, 0, &wr); ok {
			t.Errorf("région %d acceptée alors que la carte joue la région 1", region)
		}
	}
}

// TestPorteWorldObjectResteTroisBitsQuandLIndexEnFaitUn : le cas historique (toutes les cartes
// sauf Live Fire) ne change pas d'un bit. C'est la moitié du garde-rail : le correctif ne doit
// rien déplacer là où il n'y avait rien à corriger.
func TestPorteWorldObjectResteTroisBitsQuandLIndexEnFaitUn(t *testing.T) {
	axisW := [3]uint{13, 13, 14}
	defer wogInstalle(1, 0, axisW)()
	if got, want := projPosBits(), 3+13+13+14+2; got != want {
		t.Fatalf("projPosBits() = %d, attendu %d", got, want)
	}
	wr := Vec3Range{{Min: 0, Max: 8192}, {Min: 0, Max: 8192}, {Min: 0, Max: 16384}}
	pay := wogRecord(0, 1, axisW, [3]uint64{100, 200, 300})
	v, ok := decodeWorldObjectPos(pay, 0, &wr)
	if !ok {
		t.Fatal("le chemin historique (porte 000) a été refusé")
	}
	for a, want := range [3]float32{100.5, 200.5, 300.5} {
		if v[a] != want {
			t.Errorf("axe %d : %.3f, attendu %.3f", a, v[a], want)
		}
	}
}

// TestSetWorldObjectPrecisionInstalleLaRegion : la région attendue vient du CATALOGUE, par le
// même chemin que les largeurs d'axe — jamais d'un second réglage à armer à part.
func TestSetWorldObjectPrecisionInstalleLaRegion(t *testing.T) {
	defer wogInstalle(1, 0, [3]uint{13, 13, 14})()
	e := MapQuantEntry{
		Module:          "sgh_interlock",
		AxisWidths:      [3]uint{12, 12, 11},
		Region:          1,
		RegionIndexBits: 2,
	}
	SetWorldObjectPrecisionFromLayout(e.Layout())
	if WorldObjectPrecision.IndexW != 2 {
		t.Errorf("IndexW = %d, attendu 2", WorldObjectPrecision.IndexW)
	}
	if WorldObjectPrecision.Region != 1 {
		t.Errorf("Region = %d, attendue 1 — la région du catalogue n'est pas installée",
			WorldObjectPrecision.Region)
	}
}
