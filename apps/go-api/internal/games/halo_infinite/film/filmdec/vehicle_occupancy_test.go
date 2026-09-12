package filmdec

// vehicle_occupancy_test.go — GARDE-RAIL SANS ENVIRONNEMENT de `vehicle_occupancy.go` : des
// payloads FABRIQUÉS dont on connaît la réponse d'avance. Aucun film, aucune donnée réelle.

import "testing"

// TestFindKeyframeBlockInsertion exige que le localisateur retrouve EXACTEMENT une insertion
// qu'on a fabriquée, et qu'il batte ses deux modèles dégénérés.
func TestFindKeyframeBlockInsertion(t *testing.T) {
	const lf, p, d = 600, 233, 89
	// Le record COURT : un motif déterministe non trivial (un LFSR de pauvre) — un motif
	// périodique court rendrait la position d'insertion ambiguë et le test ne prouverait rien.
	court := make([]byte, (lf+7)/8+4)
	x := uint32(0xACE1)
	for i := 0; i < lf; i++ {
		x = x*1103515245 + 12345
		kfSpanEcrire(court, i, 1, uint64((x>>16)&1))
	}
	// Le record LONG : court[0:p] + BLOC(d) + court[p:], le bloc étant une constante reconnaissable.
	long := make([]byte, (lf+d+7)/8+4)
	for i := 0; i < p; i++ {
		kfSpanEcrire(long, i, 1, kfReadBits(court, i, 1))
	}
	for i := 0; i < d; i++ {
		kfSpanEcrire(long, p+i, 1, uint64((i/7)&1))
	}
	for i := p; i < lf; i++ {
		kfSpanEcrire(long, d+i, 1, kfReadBits(court, i, 1))
	}

	got := FindKeyframeBlockInsertion(
		KeyframeRecordBits{Pay: long, BitStart: 0, BitEnd: lf + d},
		KeyframeRecordBits{Pay: court, BitStart: 0, BitEnd: lf},
	)
	if !got.Valid {
		t.Fatal("insertion déclarée invalide alors que le record long l'est bien plus")
	}
	if got.BlockBits != d {
		t.Errorf("taille du bloc : %d, attendu %d", got.BlockBits, d)
	}
	if got.InsertBit != p {
		t.Errorf("position d'insertion : %d, attendu %d", got.InsertBit, p)
	}
	if got.Agree != got.Compared || got.Compared != lf {
		t.Errorf("accord : %d/%d, attendu %d/%d (l'insertion est exacte par construction)",
			got.Agree, got.Compared, lf, lf)
	}
	// LE POINT QUI COMPTE : les deux modèles dégénérés doivent perdre. S'ils gagnaient, la
	// mesure ne distinguerait pas une insertion d'un simple allongement de queue.
	if got.AgreeHead >= got.Agree || got.AgreeTail >= got.Agree {
		t.Errorf("les modèles dégénérés ne perdent pas : insertion=%d tête=%d queue=%d",
			got.Agree, got.AgreeHead, got.AgreeTail)
	}
}

// TestFindKeyframeBlockInsertionSansInsertion exige un résultat invalide quand il n'y a rien à
// chercher — un record « long » qui ne l'est pas.
func TestFindKeyframeBlockInsertionSansInsertion(t *testing.T) {
	buf := make([]byte, 16)
	a := KeyframeRecordBits{Pay: buf, BitStart: 0, BitEnd: 100}
	if got := FindKeyframeBlockInsertion(a, a); got.Valid {
		t.Errorf("deux records de même longueur : %+v, attendu invalide", got)
	}
	vide := KeyframeRecordBits{Pay: buf, BitStart: 0, BitEnd: 0}
	if got := FindKeyframeBlockInsertion(a, vide); got.Valid {
		t.Errorf("record court vide : %+v, attendu invalide", got)
	}
}

// TestVehicleKeyframeStates exige la ligne de base PAR VÉHICULE, l'excès qui en découle, et le
// traitement des emprises non mesurables.
func TestVehicleKeyframeStates(t *testing.T) {
	spans := []KeyframeRecordSpan{
		// Véhicule 100 : base 1000, puis +89 (bloc), puis +40 (sous le plancher).
		{Slot: 100, TI: VehicleTypeIndex, LengthBits: 1000, SlotGap: 1, TimestampUS: 10},
		{Slot: 100, TI: VehicleTypeIndex, LengthBits: 1089, SlotGap: 1, TimestampUS: 20},
		{Slot: 100, TI: VehicleTypeIndex, LengthBits: 1040, SlotGap: 1, TimestampUS: 30},
		// Une emprise à voisin SAUTÉ : elle couvre plusieurs records, donc rien n'en est déduit,
		// et elle ne doit surtout pas abaisser la ligne de base.
		{Slot: 100, TI: VehicleTypeIndex, LengthBits: 500, SlotGap: 4, TimestampUS: 40},
		// Véhicule 200 : un autre châssis, une autre base — jamais comparé au précédent.
		{Slot: 200, TI: VehicleTypeIndex, LengthBits: 2000, SlotGap: 1, TimestampUS: 10},
		{Slot: 200, TI: VehicleTypeIndex, LengthBits: 2500, SlotGap: 1, TimestampUS: 20},
		// Un bipède : hors périmètre, il ne doit pas ressortir.
		{Slot: 5, TI: 35, LengthBits: 3000, SlotGap: 1, TimestampUS: 10},
	}
	got := VehicleKeyframeStates(spans)
	if len(got) != 6 {
		t.Fatalf("états rendus : %d, attendu 6 (les bipèdes sont exclus) — %+v", len(got), got)
	}
	type attendu struct {
		slot, ts, base, exces int
		bloc, mesurable       bool
	}
	att := []attendu{
		{100, 10, 1000, 0, false, true},
		{200, 10, 2000, 0, false, true},
		{100, 20, 1000, 89, true, true},
		{200, 20, 2000, 500, true, true},
		{100, 30, 1000, 40, false, true},
		{100, 40, 0, 0, false, false},
	}
	for i, a := range att {
		g := got[i]
		if g.Slot != a.slot || int(g.TimestampUS) != a.ts || g.BaselineBits != a.base ||
			g.ExcessBits != a.exces || g.ExtraBlock != a.bloc || g.Measurable != a.mesurable {
			t.Errorf("état %d : %+v, attendu %+v", i, g, a)
		}
	}
}
