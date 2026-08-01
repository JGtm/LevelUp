package filmdec

// event_decoders_test.go — LES TROIS DECODEURS D EVENEMENTS, JUSQU ICI A 0 %.
//
// CE QUE CE FICHIER TESTE, ET CE QU IL LAISSE A D AUTRES. Il teste la GRAMMAIRE : les offsets,
// les largeurs, les portes et les rejets de chaque record, sur des flux construits bit a bit ou
// l on sait ce qui doit etre lu. Les memes decodeurs sont eprouves sur du BINAIRE REEL par la
// mini-bobine (internal/analysis/replay/minifilm_test.go) — les deux repondent a deux questions
// distinctes : « la grammaire est-elle celle qu on a ecrite » ici, « le film la porte-t-il
// toujours » la-bas. Aucune des deux ne remplace l autre.
//
// POURQUOI LA GRAMMAIRE MERITE SES PROPRES TESTS. Chaque offset de ce paquet est une MESURE
// (RE Ghidra ou balayage), et plusieurs ont ete corriges apres coup — l index de lanceur est a
// +103 bits et non a +102, sous peine de valeurs entre 16 et 19 ; le champ d identifiant de
// grenade commence a +23 et non +24, et propager mecaniquement ce decalage a l index aurait
// casse un decodeur qui marchait. Un test par offset est ce qui rend ces corrections
// irreversibles.

import "testing"

// bitw ecrit un flux MSB-first, comme le format du film.
type bitw struct {
	buf []byte
	n   int
}

func (w *bitw) put(v uint64, width int) {
	for i := width - 1; i >= 0; i-- {
		if w.n%8 == 0 {
			w.buf = append(w.buf, 0)
		}
		if v>>uint(i)&1 == 1 {
			w.buf[w.n/8] |= 1 << uint(7-w.n%8)
		}
		w.n++
	}
}

func (w *bitw) pad(n int) {
	for i := 0; i < n; i++ {
		w.put(0, 1)
	}
}

// ---------------------------------------------------------------------------
// Le record de tir (type 105)
// ---------------------------------------------------------------------------

// buildFireRecord ecrit la TETE du record type 105 selon le layout documente.
//
// LE PREMIER OCTET EST LE DISCRIMINANT : type sur 7 bits, puis le bit de variante. C est lui qui
// rend le typage en O(1) et qui a rendu inutile le balayage par « marqueur 11 bits ».
func buildFireRecord(attacker int, weapon uint64, flags [5]uint8, aim uint32) []byte {
	w := &bitw{}
	w.put(FireEventType, 7)
	w.put(0, 1) // variante 0 = record LONG, celui qui porte l arme
	w.pad(fireAttackerBit - w.n)
	w.put(uint64(attacker)<<1, fireAttackerW) // l index est ecrit x2 : le decodeur decale a droite
	w.pad(fireWeaponHiBit - w.n)
	w.put(weapon>>32, fireWeaponW)
	w.put(weapon&0xFFFFFFFF, fireWeaponW)
	for _, f := range flags {
		w.put(uint64(f), 1)
	}
	w.put(uint64(aim), int(FireAimBits))
	w.pad(64)
	return w.buf
}

// TestFireRecordLayout : chaque champ est lu a l offset mesure.
func TestFireRecordLayout(t *testing.T) {
	const weapon = uint64(0x48C19D2D42C9679F)
	e, ok := decodeFireEvent(buildFireRecord(6, weapon, [5]uint8{0, 0, 0, 0, 0}, 0))
	if !ok {
		t.Fatal("record complet refuse par la garde de longueur")
	}
	if e.Variant != 0 {
		t.Errorf("variante %d, attendu 0 (record long)", e.Variant)
	}
	if e.FilmIndex != 6 {
		t.Errorf("index du tireur %d, attendu 6 — l index est ecrit x2 dans le film et le "+
			"decodeur le decale a droite ; un oubli du decalage donnerait 12", e.FilmIndex)
	}
	if e.WeaponID != weapon {
		t.Errorf("arme %016X, attendu %016X — les deux moities 32 bits ne sont plus lues aux "+
			"bits %d et %d", e.WeaponID, weapon, fireWeaponHiBit, fireWeaponLoBit)
	}
}

// TestFireRecordAimOnlyOnTheSafePath : LA VISEE N EST LUE QUE LA OU ELLE EST LOCALISABLE.
//
// Hors du chemin « record vide » (drapeaux 110 = 1, 111 = 0, 112 = 0), le champ existe toujours
// mais vit APRES des boucles de longueur variable dont une largeur vient d une table remplie au
// runtime. Le decodeur ne devine pas : il n expose rien. C est cette abstention qu on teste —
// elle est le contraire d une lacune, et un refactor « qui lit toujours la visee » serait une
// regression silencieuse.
func TestFireRecordAimOnlyOnTheSafePath(t *testing.T) {
	const aim = 0x15555555 & ((1 << 30) - 1)
	sur, _ := decodeFireEvent(buildFireRecord(1, 1, [5]uint8{0, 0, 1, 0, 0}, aim))
	if !sur.HasAim {
		t.Error("chemin sur (110=1, 111=0, 112=0) : la visee doit etre lue")
	}
	for _, f := range [][5]uint8{
		{0, 0, 0, 0, 0}, // 110 = 0 : pas le chemin du record vide
		{0, 0, 1, 1, 0}, // 111 = 1 : une porte est ouverte, le champ a bouge
		{0, 0, 1, 0, 1}, // 112 = 1 : idem
	} {
		if e, _ := decodeFireEvent(buildFireRecord(1, 1, f, aim)); e.HasAim {
			t.Errorf("drapeaux %v : la visee a ete lue hors du chemin sur — le champ n y est "+
				"pas localisable hors ligne, et la lire revient a inventer une direction", f)
		}
	}
}

// TestFireHeadingConventionMatchesPositions : le cap du tir et celui des positions ont la MEME
// origine et le MEME sens (atan2(Y, X)). Sans quoi les tirs partiraient de travers.
func TestFireHeadingConventionMatchesPositions(t *testing.T) {
	e := FireEvent{HasAim: true, Aim: [3]float32{0, 1, 0}}
	h, ok := e.AimHeadingDeg()
	if !ok {
		t.Fatal("visee declaree presente mais non rendue")
	}
	if h < 89.9 || h > 90.1 {
		t.Errorf("cap %.2f pour +Y, attendu 90 — la convention a change", h)
	}
	sud := FireEvent{HasAim: true, Aim: [3]float32{0, -1, 0}}
	if h, _ := sud.AimHeadingDeg(); h < 269.9 || h > 270.1 {
		t.Errorf("cap %.2f pour -Y, attendu 270 : le repliement dans [0,360[ a change", h)
	}
	if _, ok := (FireEvent{}).AimHeadingDeg(); ok {
		t.Error("un evenement sans visee rend un cap : il ferait dessiner une direction inventee")
	}
}

// ---------------------------------------------------------------------------
// Le lancer de grenade
// ---------------------------------------------------------------------------

// buildGrenadeRecord ecrit [marqueur 24][identifiant 32][47 bits][index 5].
func buildGrenadeRecord(lead int, id uint32, index uint32) []byte {
	w := &bitw{}
	w.pad(lead)
	w.put(grenadeMarker, 24)
	w.put(uint64(id), 32)
	w.pad(47)
	w.put(uint64(index), 5)
	w.pad(32)
	return w.buf
}

// TestGrenadeThrowLayout : l index de lanceur est a +103 bits du marqueur, pas a +102.
//
// LA VALEUR DE CE TEST EST DANS LE TEMOIN NEGATIF. A +102 les 70 lancers du film de reference
// tombent tous entre 16 et 19 (le bit de poids fort est a 1) ; a +103 ils sont tous dans 0..7.
// Le test reproduit exactement ce contraste sur un flux construit.
func TestGrenadeThrowLayout(t *testing.T) {
	got := scanGrenadeThrows(buildGrenadeRecord(11, GrenadePlasma, 5))
	if len(got) != 1 {
		t.Fatalf("%d lancer(s) trouve(s), attendu 1", len(got))
	}
	if got[0].FilmIndex != 5 {
		t.Errorf("index de lanceur %d, attendu 5", got[0].FilmIndex)
	}
	if got[0].TypeID != GrenadePlasma || got[0].Name() != "Plasma" {
		t.Errorf("type %08x (%q), attendu Plasma", got[0].TypeID, got[0].Name())
	}
	if got[0].BitPos != 11 {
		t.Errorf("marqueur rapporte au bit %d, attendu 11", got[0].BitPos)
	}
	// LE TEMOIN, ET SA LIMITE HONNETE. Sur le film, lu a +102, les 70 index tombent tous entre
	// 16 et 19 — le contraste vient des bits REELS qui precedent le champ, et un flux construit
	// ne peut pas le reproduire (le bourrage y est a zero). Ce qui se verifie ici est donc la
	// seule chose qu un flux construit puisse dire : l offset n est pas interchangeable.
	pay := buildGrenadeRecord(11, GrenadePlasma, 5)
	if v := PeekBits(pay, 11+24+32+47, 5); v != 5 {
		t.Errorf("lecture a +103 : %d, attendu 5", v)
	}
	if v := PeekBits(pay, 11+24+32+46, 5); v == 5 {
		t.Error("lecture a +102 : la meme valeur qu a +103 — le banc d essai ne discrimine plus " +
			"les deux offsets, et ce test ne protege plus la correction qui les a departages")
	}
}

// TestGrenadeWhitelistIsWhatMakesTheMarkerSelective : LE MARQUEUR SEUL NE SELECTIONNE RIEN.
//
// C est ecrit en tete du decodeur et c est la propriete qui le rend utilisable : la constante
// 24 bits apparait 1 416 fois dans le film de reference pour 70 lancers reels. Ce qui trie, c est
// l appartenance de l identifiant 32 bits a la liste blanche des quatre grenades.
func TestGrenadeWhitelistIsWhatMakesTheMarkerSelective(t *testing.T) {
	if got := scanGrenadeThrows(buildGrenadeRecord(0, 0xDEADBEEF, 3)); len(got) != 0 {
		t.Errorf("%d lancer(s) sur un marqueur suivi d un identifiant HORS liste blanche : la "+
			"selectivite ne vient plus de la liste, et le decodeur rendrait du bruit", len(got))
	}
	for id, name := range KnownGrenadeIDs {
		got := scanGrenadeThrows(buildGrenadeRecord(0, id, 1))
		if len(got) != 1 || got[0].Name() != name {
			t.Errorf("identifiant %08x (%s) : %d lancer(s) reconnu(s)", id, name, len(got))
		}
	}
}

// ---------------------------------------------------------------------------
// Le record de projectile
// ---------------------------------------------------------------------------

// buildProjectileRecord ecrit [1][slot 13][gen 2][porte 2][mc 3][mc x 6 bits][position].
func buildProjectileRecord(slot, gen uint32, comps []int, q [3]uint64) []byte {
	w := &bitw{}
	w.put(1, 1) // prefixe de record DELTA
	w.put(uint64(slot), 13)
	w.put(uint64(gen), 2)
	w.put(0, 2) // porte de masque = 0 -> branche eparse
	w.put(uint64(len(comps)), 3)
	for _, c := range comps {
		w.put(uint64(c), 6)
	}
	w.put(0, 3) // porte de position : precHigh, index-sel, region tous nuls
	for a := 0; a < 3; a++ {
		w.put(q[a], int(WorldObjectPrecision.AxisW[a]))
	}
	w.pad(64)
	return w.buf
}

func projTestRange() Vec3Range {
	return Vec3Range{{Min: -100, Max: 100}, {Min: -100, Max: 100}, {Min: -100, Max: 100}}
}

// TestProjectileRecordLayout : slot, generation, composants et position.
func TestProjectileRecordLayout(t *testing.T) {
	wr := projTestRange()
	band := map[uint32]bool{1500: true}
	got := scanProjectileRecords(
		buildProjectileRecord(1500, 2, []int{0, 3, projectileRestComponent}, [3]uint64{4096, 4096, 8192}),
		band, &wr)
	if len(got) != 1 {
		t.Fatalf("%d record(s) accepte(s), attendu 1", len(got))
	}
	if got[0].slot != 1500 || got[0].gen != 2 {
		t.Errorf("slot/gen %d/%d, attendu 1500/2", got[0].slot, got[0].gen)
	}
	if !got[0].AtRest {
		t.Errorf("le composant i%d (projectile-at-rest-state) est dans le masque et n a pas ete "+
			"vu : c est le SEUL champ qui certifie une fin de vol", projectileRestComponent)
	}
	if got[0].X < -1 || got[0].X > 1 {
		t.Errorf("x = %.3f pour un quantum a mi-course sur [-100, 100], attendu ~0", got[0].X)
	}
}

// TestProjectileRecordRejections : les quatre portes qui font la selectivite du balayage.
//
// UN BALAYAGE BIT A BIT ACCEPTE TOUT SI ON NE LE CONTRAINT PAS — c est la lecon la plus chere du
// chantier (99,85 % de faux positifs sur un balayage par position). Chacune de ces portes est
// donc testee par ce qu elle REFUSE.
func TestProjectileRecordRejections(t *testing.T) {
	wr := projTestRange()
	band := map[uint32]bool{1500: true}
	cas := []struct {
		nom string
		pay []byte
	}{
		{"slot hors bande", buildProjectileRecord(1501, 0, []int{0, 3}, [3]uint64{4096, 4096, 8192})},
		{"i0 absent du masque", buildProjectileRecord(1500, 0, []int{1, 3}, [3]uint64{4096, 4096, 8192})},
		{"composants non strictement croissants",
			buildProjectileRecord(1500, 0, []int{0, 3, 3}, [3]uint64{4096, 4096, 8192})},
		{"quantum sature a zero", buildProjectileRecord(1500, 0, []int{0, 3}, [3]uint64{0, 4096, 8192})},
		{"quantum sature au maximum",
			buildProjectileRecord(1500, 0, []int{0, 3}, [3]uint64{8191, 4096, 8192})},
	}
	for _, c := range cas {
		if got := scanProjectileRecords(c.pay, band, &wr); len(got) != 0 {
			t.Errorf("%s : %d record(s) accepte(s), attendu 0", c.nom, len(got))
		}
	}
}

// TestAscendingComponentsIsStrict : l ordre STRICTEMENT croissant est la contrainte porteuse.
func TestAscendingComponentsIsStrict(t *testing.T) {
	w := &bitw{}
	for _, c := range []int{0, 5, 5} {
		w.put(uint64(c), 6)
	}
	if _, ok := ascendingComponents(w.buf, 0, 3); ok {
		t.Error("une repetition a ete acceptee : la contrainte n est plus stricte, et c est elle " +
			"qui fait toute la selectivite du balayage")
	}
	w2 := &bitw{}
	for _, c := range []int{0, 1, 18} {
		w2.put(uint64(c), 6)
	}
	idx, ok := ascendingComponents(w2.buf, 0, 3)
	if !ok || idx[2] != 18 {
		t.Errorf("suite croissante refusee ou mal lue : %v (ok=%v)", idx, ok)
	}
}

// TestSplitLivesSeparatesFlights : le pool de slots reboucle, la generation ne fait que 2 bits.
//
// SANS CE DECOUPAGE on obtient des « trajectoires » de 300 a 460 secondes, c est-a-dire la
// concatenation de dizaines de vols. Un projectile est repliqué a ~60 Hz : un trou de plus de
// 250 ms est une frontiere de vie, pas une lacune.
func TestSplitLivesSeparatesFlights(t *testing.T) {
	pts := []ProjectileSample{
		{TimestampUS: 0}, {TimestampUS: 16_000}, {TimestampUS: 32_000},
		{TimestampUS: 1_000_000}, {TimestampUS: 1_016_000}, {TimestampUS: 1_032_000},
	}
	segs := splitLives(pts)
	if len(segs) != 2 {
		t.Fatalf("%d vie(s) apres un trou d une seconde, attendu 2", len(segs))
	}
	if len(segs[0]) != 3 || len(segs[1]) != 3 {
		t.Errorf("decoupage desequilibre : %d et %d points", len(segs[0]), len(segs[1]))
	}
}

// TestSplitLivesClosesOnAtRest : un record « au repos » clot la vie, meme sans trou.
func TestSplitLivesClosesOnAtRest(t *testing.T) {
	pts := []ProjectileSample{
		{TimestampUS: 0}, {TimestampUS: 16_000}, {TimestampUS: 32_000, AtRest: true},
		{TimestampUS: 48_000}, {TimestampUS: 64_000}, {TimestampUS: 80_000},
	}
	segs := splitLives(pts)
	if len(segs) != 2 {
		t.Fatalf("%d vie(s), attendu 2 : `projectile-at-rest-state` doit clore le vol", len(segs))
	}
}

// TestSplitLivesDropsTooShort : deux points ne dessinent pas une trajectoire.
func TestSplitLivesDropsTooShort(t *testing.T) {
	if segs := splitLives([]ProjectileSample{{TimestampUS: 0}, {TimestampUS: 16_000}}); len(segs) != 0 {
		t.Errorf("%d vie(s) pour deux points : une trajectoire de deux points ne se dessine pas",
			len(segs))
	}
}

// TestWorldObjectPositionRejectsSaturatedAxes : un quantum sature est une valeur de GARDE.
//
// Sans cette regle, une vie sur soixante-dix finit au plancher du BSP (z = -84 m) — ce n est pas
// une position, et l afficher dessinerait une chute qui n a pas eu lieu.
func TestWorldObjectPositionRejectsSaturatedAxes(t *testing.T) {
	wr := projTestRange()
	w := &bitw{}
	w.put(0, 3)
	for a := 0; a < 3; a++ {
		w.put(4096, int(WorldObjectPrecision.AxisW[a]))
	}
	w.pad(16)
	if _, ok := decodeWorldObjectPos(w.buf, 0, &wr); !ok {
		t.Fatal("une position valide a ete refusee")
	}
	for a := 0; a < 3; a++ {
		for _, q := range []uint64{0, (1 << WorldObjectPrecision.AxisW[a]) - 1} {
			g := &bitw{}
			g.put(0, 3)
			for b := 0; b < 3; b++ {
				v := uint64(4096)
				if b == a {
					v = q
				}
				g.put(v, int(WorldObjectPrecision.AxisW[b]))
			}
			g.pad(16)
			if _, ok := decodeWorldObjectPos(g.buf, 0, &wr); ok {
				t.Errorf("axe %d, quantum %d : accepte alors qu il est sature", a, q)
			}
		}
	}
}

// TestWorldObjectPositionGateIsClosedUnlessAllThreeAreZero : la porte de 3 bits.
func TestWorldObjectPositionGateIsClosedUnlessAllThreeAreZero(t *testing.T) {
	wr := projTestRange()
	for gate := uint64(1); gate < 8; gate++ {
		w := &bitw{}
		w.put(gate, 3)
		for a := 0; a < 3; a++ {
			w.put(4096, int(WorldObjectPrecision.AxisW[a]))
		}
		w.pad(16)
		if _, ok := decodeWorldObjectPos(w.buf, 0, &wr); ok {
			t.Errorf("porte %03b : la position a ete lue hors du chemin dominant", gate)
		}
	}
}

// TestProjectilePositionWidthFollowsThePrecisionDescriptor : la longueur du composant se DERIVE.
//
// Elle n est pas une constante : elle vaut 3 de porte + les trois axes + 2 de queue, et les
// largeurs d axe viennent du descripteur de precision (qui est lu dans le film). Un chiffre
// ecrit en dur se serait desynchronise du jour ou une carte a d autres largeurs.
func TestProjectilePositionWidthFollowsThePrecisionDescriptor(t *testing.T) {
	p := WorldObjectPrecision
	want := 3 + int(p.AxisW[0]+p.AxisW[1]+p.AxisW[2]) + 2
	if got := projPosBits(); got != want {
		t.Errorf("projPosBits() = %d, attendu %d — la longueur ne suit plus le descripteur",
			got, want)
	}
}

// TestScanFilmProjectilesRefusesWithoutMapBounds : sans bornes, le decodeur ne rend QUE des
// quanta — et un quantum n est pas une coordonnee. Le refus est la bonne reponse.
func TestScanFilmProjectilesRefusesWithoutMapBounds(t *testing.T) {
	if _, err := ScanFilmProjectiles("testdata", nil); err == nil {
		t.Error("aucune erreur sans bornes monde : le decodeur rendrait des quanta presentes " +
			"comme des positions")
	}
}
