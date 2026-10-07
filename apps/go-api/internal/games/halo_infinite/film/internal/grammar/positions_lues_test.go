package grammar

// positions_lues_test.go — LE CONTRAT DES POSITIONS BIPEDES : la marche des trames designe les
// records, l ancrage passe derriere elle, et la position se lit au bit d i0 du record par le lecteur
// de toujours.

import (
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestLesPositionsSeFondentDansLOrdreDuFlux : les records de la marche et ceux de l ancrage derriere
// elle se fondent par rang du chunk dans le film, rang du paquet, puis bit d i0 ; un paquet que les
// deux sources portent n apparait qu une fois.
func TestLesPositionsSeFondentDansLOrdreDuFlux(t *testing.T) {
	var marche, ancrage positionsBipedes
	pk := func(i int) FilmPacket { return FilmPacket{Index: i} }
	marche.noter(2, pk(5), positionLue{i0: 100, slot: 1})
	marche.noter(2, pk(5), positionLue{i0: 300, slot: 3})
	marche.noter(3, pk(1), positionLue{i0: 50, slot: 5})
	ancrage.noter(2, pk(5), positionLue{i0: 200, slot: 2, recupere: true})
	ancrage.noter(2, pk(7), positionLue{i0: 10, slot: 4, recupere: true})
	f := fondrePositions(&marche, &ancrage, []int{1, 2, 3})
	type ligne struct {
		chunk, paquet int
		i0            uint32
		recupere      bool
	}
	var got []ligne
	for k, p := range f.paquets {
		for _, r := range f.recordsDu(k) {
			got = append(got, ligne{p.chunk, p.paquet.Index, r.i0, r.recupere})
		}
	}
	attendu := []ligne{{2, 5, 100, false}, {2, 5, 200, true}, {2, 5, 300, false}, {2, 7, 10, true}, {3, 1, 50, false}}
	if !slices.Equal(got, attendu) {
		t.Fatalf("fusion %v, attendu %v", got, attendu)
	}
	if len(f.paquets) != 3 {
		t.Fatalf("%d paquets, attendu 3 (le paquet 2:5 une seule fois)", len(f.paquets))
	}
}

// TestUnI0SeLitAbsoluDansLaRegionJouee : la grammaire d i0 des positions — spine et useDefault nuls,
// index de region egal a celui de la carte, trois axes dans le payload.
func TestUnI0SeLitAbsoluDansLaRegionJouee(t *testing.T) {
	lay := profile.I0Layout{GateBits: profile.DefaultI0GateBits, AxisW: [3]uint{10, 10, 8}, Region: 1}
	ecrire := func(spine, defaut, region uint64, axes int) []byte {
		var w bitWriter
		w.bits(spine, profile.I0SpineBits)
		w.bits(defaut, profile.I0UseDefaultBits)
		w.bits(region, profile.DefaultI0GateBits-profile.I0SpineBits-profile.I0UseDefaultBits)
		w.bits(0, axes)
		return w.buf
	}
	cas := []struct {
		nom  string
		pay  []byte
		vrai bool
	}{
		{"absolu dans la region", ecrire(0, 0, 1, 28), true},
		{"spine non nulle", ecrire(2, 0, 1, 28), false},
		{"etat par defaut", ecrire(0, 1, 1, 28), false},
		{"autre region", ecrire(0, 0, 0, 28), false},
		{"axes hors du payload", ecrire(0, 0, 1, 8), false},
	}
	for _, c := range cas {
		if got := i0AbsoluDeLaRegion(c.pay, 0, lay); got != c.vrai {
			t.Errorf("%s : %v, attendu %v", c.nom, got, c.vrai)
		}
	}
}

// TestLesPositionsSuiventLaMarcheSurLaMiniBobine : sur la mini-bobine (registre compris), chaque
// position vient d un record retenu — de la marche, ou de l ancrage derriere elle hors de ce que la
// trame prouve et d un slot qu elle n a pas lu ; la marche en designe ; aux records que l ancrage seul
// rendait aussi, les quanta sont les memes ; et la bande bipede reste la population des positions :
// le slot 529, ne au chunk 5 (le dernier de la bobine) apres sa premiere image-cle, que la marche lit
// mais qu aucune image-cle ne porte, n a pas de position.
func TestLesPositionsSuiventLaMarcheSurLaMiniBobine(t *testing.T) {
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		t.Fatalf("mini-bobine versionnee illisible (%s) : %v", bobineFamilles, err)
	}
	opt := ScanFilmOptions{QuantaOnly: true}
	fc := NewFilmContext(film)
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatal(err)
	}
	canal := nouveauCanalDesLecturesBipedes(fc)
	if err := Distribuer(fc, canal); err != nil {
		t.Fatal(err)
	}
	pb := &fc.recup.lectures.positions
	for k, p := range pb.paquets {
		tr := canal.trames[paquetDuFlux{p.chunk, p.paquet.Index}]
		for _, r := range pb.recordsDu(k) {
			if r.recupere && (int64(r.i0) >= int64(tr.prouveeDes) || slices.Contains(tr.slots, r.slot)) {
				t.Fatalf("position recuperee chunk %d paquet %d slot %d dans ce que la trame prouve ou d un slot lu",
					p.chunk, p.paquet.Index, r.slot)
			}
		}
	}
	neuves, _ := positionsDuContexte(fc, fc.ChunkNumbers(), lay, opt)
	if len(neuves) != len(pb.records) {
		t.Fatalf("%d positions pour %d records retenus", len(neuves), len(pb.records))
	}
	type cle struct {
		chunk, paquet int
		slot          uint32
	}
	anciennes := map[cle][3]uint32{}
	seul := NewFilmContext(film)
	ancrees, _ := positionsDesAncres(seul, seul.ChunkNumbers(), lay, opt)
	for _, p := range ancrees {
		anciennes[cle{p.Chunk, p.PacketIndex, p.Slot}] = p.Q
	}
	communes := 0
	for _, p := range neuves {
		if q, ok := anciennes[cle{p.Chunk, p.PacketIndex, p.Slot}]; ok {
			communes++
			if q != p.Q {
				t.Fatalf("chunk %d paquet %d slot %d : quanta %v par la marche, %v par l ancrage", p.Chunk,
					p.PacketIndex, p.Slot, p.Q, q)
			}
		}
		if p.Slot == 529 && p.Chunk == 5 {
			t.Fatalf("le slot 529, hors de la bande bipede, a une position au chunk 5 paquet %d", p.PacketIndex)
		}
	}
	parLaMarche := 0
	for _, r := range pb.records {
		if !r.recupere {
			parLaMarche++
		}
	}
	if communes == 0 || parLaMarche == 0 {
		t.Fatalf("communes %d, designees par la marche %d : la bobine doit porter les deux", communes, parLaMarche)
	}
	marcheLit529 := false
	for k := range canal.trames {
		if k.chunk == 5 && slices.Contains(canal.trames[k].slots, 529) {
			marcheLit529 = true
			break
		}
	}
	if !marcheLit529 {
		t.Fatal("la marche ne lit plus le slot 529 au chunk 5 : le temoin de la population ne garde rien")
	}
}
