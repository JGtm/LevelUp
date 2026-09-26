package grammar

// keyframe_world_test.go — LE BALAYEUR D'IMAGE-CLÉ DU LOT M3.1 (2026-09-23), SUR PAYLOADS
// SYNTHÉTIQUES : le recalage sur l'en-tête exact d'un bipède, la fenêtre qui GLISSE au lieu de
// couper la table, et les compteurs de décision.
//
// LES DEUX PREMIERS TESTS ÉTAIENT ROUGES SUR LA BASE `fe7079f41` (preuve rejouée au lot) : la
// production y élisait la fausse ancre de slot bas (0 bipède sur 3) et s'arrêtait sur la
// fenêtre vide (1 record sur 2).
//
// Les payloads imitent la forme MESURÉE par la sonde P2 (81c02726 morceau 9) : un record de
// nuée, une zone sans aucun en-tête, des bipèdes de slots croissants NON consécutifs, et une
// fausse ancre « slot 256, génération 1, archétype 0 » prise dans le corps du dernier bipède.

import "testing"

// kfEcrireRecord écrit l'en-tête de 108 bits d'un record d'image-cle — `[eid:32][archetype:32]
// [32][4][8]` —, le mot de taille n1, puis `corps` bits nuls. Un corps nul ne porte aucune ancre :
// un identifiant nul a une génération nulle, que le balayeur rejette.
func kfEcrireRecord(w *bitWriter, gen, slot, ti uint64, corps int) {
	w.bits(gen<<30|slot, 32)
	w.bits(ti, 32)
	w.bits(0, 32+4+8)
	w.bits(152, 32) // n1
	w.bits(0, corps)
}

// kfPayloadPrefixeBipedes construit le payload du cas P2 : nuée (125) ; zone vide ; bipèdes 519,
// 528, 530 ; fausse ancre 256/ti 0 dans le corps du dernier ; sentinelles de fin de table.
func kfPayloadPrefixeBipedes() []byte {
	w := &bitWriter{}
	w.bit(0) // préfixe d'un bit du payload d'image-clé
	kfEcrireRecord(w, 1, 125, 21, 200)
	w.bits(0, 43000) // la zone sans en-tête qui suit la nuée
	for _, slot := range []uint64{519, 528} {
		kfEcrireRecord(w, 1, slot, BipedTypeIndex, 2600)
	}
	kfEcrireRecord(w, 1, 530, BipedTypeIndex, 1300)
	// la fausse ancre, DANS le corps du dernier bipède : « 0x40000100 00000000 ».
	w.bits(1<<30|256, 32)
	w.bits(0, 32)
	w.bits(0, 1300)
	for i := 0; i < 2100; i++ {
		w.bits(kfSent, 32)
	}
	return w.buf
}

// TestKeyframeWorldRecaleSurLEnTeteExactDUnBipede : les trois bipèdes sont lus, la fausse ancre
// de slot bas ne l'est pas.
func TestKeyframeWorldRecaleSurLEnTeteExactDUnBipede(t *testing.T) {
	recs, st := WalkKeyframeWorldStats(kfPayloadPrefixeBipedes())
	var bipedes []int
	for _, r := range recs {
		if r.Slot == 256 {
			t.Fatalf("la fausse ancre 256/ti %d a été retenue (bit %d)", r.TI, r.Bit)
		}
		if r.TI == BipedTypeIndex {
			bipedes = append(bipedes, r.Slot)
		}
	}
	if len(bipedes) != 3 || bipedes[0] != 519 || bipedes[1] != 528 || bipedes[2] != 530 {
		t.Fatalf("bipèdes lus %v, attendu [519 528 530] — records %+v", bipedes, recs)
	}
	if st.Recalages != 3 || st.Elections != 0 || st.Bipedes != 3 || st.Records != 4 {
		t.Fatalf("compteurs %+v : attendu 3 recalages, 0 élection, 3 bipèdes, 4 records", st)
	}
}

// TestKeyframeWorldUneFenetreVideNeCoupePasLaTable : un trou de plus de 120 000 bits entre deux
// records voisins ne termine plus la marche.
func TestKeyframeWorldUneFenetreVideNeCoupePasLaTable(t *testing.T) {
	w := &bitWriter{}
	w.bit(0)
	kfEcrireRecord(w, 1, 10, 5, 300)
	w.bits(0, kfScanFenetreBits+5000)
	kfEcrireRecord(w, 1, 11, 5, 300)
	for i := 0; i < 2100; i++ {
		w.bits(kfSent, 32)
	}
	recs, st := WalkKeyframeWorldStats(w.buf)
	if len(recs) != 2 || recs[1].Slot != 11 {
		t.Fatalf("records %+v : le voisin 11, au-delà d'une fenêtre vide, n'a pas été lu", recs)
	}
	if st.Glissements < 1 || st.Voisins != 1 {
		t.Fatalf("compteurs %+v : attendu au moins un glissement et un voisin", st)
	}
}

// TestKeyframeWorldLaFinDeTableArreteLeGlissement : les sentinelles de fin de table arrêtent la
// recherche — le glissement ne va pas lire au-delà de la table.
func TestKeyframeWorldLaFinDeTableArreteLeGlissement(t *testing.T) {
	w := &bitWriter{}
	w.bit(0)
	kfEcrireRecord(w, 1, 10, 5, 300)
	for i := 0; i < 2100; i++ {
		w.bits(kfSent, 32)
	}
	// Un record APRÈS la fin de table ne doit jamais être atteint.
	w.bits(0, kfScanFenetreBits)
	kfEcrireRecord(w, 1, 11, 5, 300)
	recs, st := WalkKeyframeWorldStats(w.buf)
	if len(recs) != 1 {
		t.Fatalf("records %+v : la marche a lu au-delà de la fin de table", recs)
	}
	if st.Glissements != 0 {
		t.Fatalf("compteurs %+v : aucun glissement attendu après la fin de table", st)
	}
}

// TestKeyframeWorldLElectionResteLeRepliSansBipede : sans en-tête exact de bipède, l'élection
// décide comme avant (slot bas), et elle est COMPTÉE.
func TestKeyframeWorldLElectionResteLeRepliSansBipede(t *testing.T) {
	w := &bitWriter{}
	w.bit(0)
	kfEcrireRecord(w, 1, 10, 5, 300)
	kfEcrireRecord(w, 1, 40, 5, 300) // non consécutif, plus proche
	kfEcrireRecord(w, 1, 20, 6, 300) // non consécutif, slot plus bas, plus loin
	for i := 0; i < 2100; i++ {
		w.bits(kfSent, 32)
	}
	recs, st := WalkKeyframeWorldStats(w.buf)
	if len(recs) < 2 || recs[1].Slot != 20 {
		t.Fatalf("records %+v : l'élection devait retenir le slot bas 20", recs)
	}
	if st.Elections < 1 || st.Recalages != 0 {
		t.Fatalf("compteurs %+v : attendu au moins une élection, aucun recalage", st)
	}
}

// TestBipedesAbsentsEncadres : un bipède présent aux deux images-clés voisines et absent de
// celle du milieu compte ; un bipède né ou mort entre deux images-clés ne compte pas.
func TestBipedesAbsentsEncadres(t *testing.T) {
	a, b, c := uint32(1<<30|519), uint32(1<<30|528), uint32(1<<30|530)
	parImageCle := []map[uint32]bool{
		{a: true, b: true},          // k0
		{b: true},                   // k1 : a manque, encadré par k0 et k2 -> 1
		{a: true, b: true, c: true}, // k2 : c naît
		{a: true, c: true},          // k3 : b meurt (absent de k4 aussi)
		{a: true, c: true},          // k4
	}
	if n := bipedesAbsentsEncadres(parImageCle); n != 1 {
		t.Fatalf("absents encadrés %d, attendu 1", n)
	}
	if n := bipedesAbsentsEncadres(parImageCle[:2]); n != 0 {
		t.Fatalf("deux images-clés ne encadrent rien : %d", n)
	}
}

// TestKeyframeWorldUneFinDeTableAChevalSurDeuxFenetres : une traînée de sentinelles COUPÉE par la
// frontière de deux fenêtres de 120 000 bits reste une fin de table (constat F5 de la revue
// adverse du lot M3.1, 2026-09-24). ROUGE sur 533fe7d91 : le compteur de sentinelles repartait de
// zéro à chaque fenêtre, aucune des deux moitiés n'atteignait 2 048, et le glissement allait lire
// l'ancre 11 au-delà de la table.
func TestKeyframeWorldUneFinDeTableAChevalSurDeuxFenetres(t *testing.T) {
	w := &bitWriter{}
	w.bit(0)
	kfEcrireRecord(w, 1, 10, 5, 300)
	// La fenêtre du record 10 commence à la fin de son en-tête de 64 bits (bit 65) : la traînée
	// de 100 mots (3 200 bits) commence 1 600 bits avant sa fin, et la chevauche.
	debutFenetre := 1 + 64
	w.bits(0, debutFenetre+kfScanFenetreBits-1600-w.n)
	for i := 0; i < 100; i++ {
		w.bits(kfSent, 32)
	}
	w.bits(0, 500)
	kfEcrireRecord(w, 1, 11, 5, 300) // au-delà de la table : jamais atteint
	w.bits(0, 500)
	recs, st := WalkKeyframeWorldStats(w.buf)
	if len(recs) != 1 || recs[0].Slot != 10 {
		t.Fatalf("records %+v : la marche a lu au-delà d'une fin de table à cheval sur deux fenêtres", recs)
	}
	if st.Glissements != 0 {
		t.Fatalf("compteurs %+v : aucun glissement attendu, la fin de table arrête la recherche", st)
	}
}
