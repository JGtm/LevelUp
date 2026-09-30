//go:build research

package grammar

// r2_positions_aberrantes_research_test.go — INSTRUMENT du lot R2 (plan de suite de l audit du
// decodeur, 2026-09-28, constat C2 du G-corpus J11.1) : d ou viennent les positions aberrantes
// publiees depuis le filtre de generation vivante (J5.2 GB-1) ?
//
// Il imprime, pour UN slot autour d UN instant de l horloge du film : les records de creation du
// slot, les generations que les images-cles lui connaissent, CHAQUE record delta bipede ancre (toutes
// generations, vivantes ou non) avec son handle, ses quanta et sa coordonnee, puis ce que le
// balayage de production rend. Saute sans les variables d environnement ; jamais en CI.
//
//	R2_FILM=<parc>/data/cache/film_chunks/a349fea8 R2_MAP="Fragmentation Heavies" R2_SLOT=532 \
//	  R2_ORIGIN_MS=11122 R2_FRAME=5134 go test ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run R2PositionsAberrantes -v

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/testutil"
)

func TestR2PositionsAberrantes(t *testing.T) {
	dir, carte := os.Getenv("R2_FILM"), os.Getenv("R2_MAP")
	if dir == "" || carte == "" {
		t.Skip("R2_FILM / R2_MAP absents : instrument du lot R2 saute")
	}
	slot64, _ := strconv.ParseUint(os.Getenv("R2_SLOT"), 10, 32)
	frame, _ := strconv.ParseInt(os.Getenv("R2_FRAME"), 10, 64)
	origineMS, _ := strconv.ParseInt(os.Getenv("R2_ORIGIN_MS"), 10, 64)
	fenetre := int64(40_000_000)
	slot := uint32(slot64)
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(racine, "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	e, err := cat.Lookup(carte)
	if err != nil {
		t.Fatal(err)
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	fc := NewFilmContextForMap(film, &e, nil)
	// LA FRAME 0 est le premier paquet de position : premier paquet du chunk 1 + originMs (origin.go).
	_, pks1, _ := fc.ChunkAt(1)
	tUS := int64(pks1[0].TimestampUS) + origineMS*1000 + frame*100_000
	t.Logf("frame %d -> t=%d", frame, tUS)
	cre, _, _ := fc.CreationsDeBipede()
	for _, c := range cre {
		if c.Slot == slot {
			t.Logf("CREATION slot=%d gen=%d index=%d(has=%v) t=%d chunk=%d", c.Slot, c.Generation,
				c.ParticipantIndex, c.HasIndex, c.TimestampUS, c.Chunk)
		}
	}
	gens := fc.GenerationsVivantes()
	t.Logf("GENERATIONS VIVANTES slot=%d masque=%04b", slot, gens.masque(slot))
	opt := DefaultScanFilmOptions()
	wr := e.Range()
	opt.WorldRange = &wr
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatal(err)
	}
	band := fc.BipedSlots()
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || absI64R2(int64(pk.TimestampUS)-tUS) > fenetre {
				continue
			}
			pay := pk.Payload(data)
			walkDeltaBipedPayload(pay, band, lay, ToutesLesGenerations(), func(r deltaBipedRecord) {
				if r.Slot != slot {
					return
				}
				var q [3]uint32
				for ax := 0; ax < 3; ax++ {
					q[ax] = uint32(source.BitsStricts(pay, r.I0+lay.AxisOffset(ax), int(lay.AxisW[ax])))
				}
				t.Logf("RECORD t=%d chunk=%d pk=%d gen=%d vivante=%v masque=%v q=%v sat=%v xyz=(%.2f %.2f %.2f)",
					pk.TimestampUS, c, pk.Index, r.Gen, gens.Accepte(r.Vie()), r.Mask, q, saturatedQuantum(q, lay),
					DequantBipedAxis(q[0], 0, lay, wr), DequantBipedAxis(q[1], 1, lay, wr), DequantBipedAxis(q[2], 2, lay, wr))
			})
		}
	}
	pos, err := ScanBipedPositions(fc, opt)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pos {
		if p.Slot == slot && absI64R2(int64(p.TimestampUS)-tUS) <= fenetre {
			t.Logf("PRODUCTION t=%d q=%v xyz=(%.2f %.2f %.2f)", p.TimestampUS, p.Q, p.X, p.Y, p.Z)
		}
	}
}

func absI64R2(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// TestR2BilanDesCorps classe, sur TOUT le film, chaque record delta bipede que le filtre de
// generation vivante accepte, selon la place de son instant dans la vie de son corps (slot,
// generation) que les records de creation datent. Imprime les comptes (records ancres, puis
// positions rendues par le balayage de production). R2_FILM + R2_MAP.
func TestR2BilanDesCorps(t *testing.T) {
	dir, carte := os.Getenv("R2_FILM"), os.Getenv("R2_MAP")
	if dir == "" || carte == "" {
		t.Skip("R2_FILM / R2_MAP absents : instrument du lot R2 saute")
	}
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(racine, "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	e, err := cat.Lookup(carte)
	if err != nil {
		t.Fatal(err)
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	fc := NewFilmContextForMap(film, &e, nil)
	cre, _, _ := fc.CreationsDeBipede()
	type dc struct {
		t   uint64
		gen uint32
	}
	parSlot := map[uint32][]dc{}
	for _, c := range cre {
		parSlot[c.Slot] = append(parSlot[c.Slot], dc{c.TimestampUS, c.Generation})
	}
	for s := range parSlot {
		l := parSlot[s]
		for i := 1; i < len(l); i++ {
			for j := i; j > 0 && l[j].t < l[j-1].t; j-- {
				l[j], l[j-1] = l[j-1], l[j]
			}
		}
	}
	classe := func(slot, gen uint32, ts uint64) string {
		l := parSlot[slot]
		if len(l) == 0 {
			return "slot_sans_creation"
		}
		k := -1
		for i, d := range l {
			if d.gen == gen {
				k = i
				break
			}
		}
		if k < 0 {
			return "corps_sans_creation"
		}
		if ts < l[k].t {
			if k == 0 && gen == 1 {
				return "avant_creation_premier_corps_gen1"
			}
			if k == 0 {
				return "avant_creation_premier_corps_genN"
			}
			return "avant_creation_corps_suivant"
		}
		if k+1 < len(l) && ts >= l[k+1].t {
			return "apres_creation_du_corps_suivant"
		}
		return "dans_sa_vie"
	}
	gens := fc.GenerationsVivantes()
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatal(err)
	}
	records := map[string]int{}
	walkDeltaBipedRecords(fc, fc.ChunkNumbers(), fc.BipedSlots(), lay, func(r deltaBipedRecord) {
		records[classe(r.Slot, r.Gen, r.Packet.TimestampUS)+"/gen"+strconv.Itoa(int(r.Gen))]++
	})
	t.Logf("RECORDS %v", records)
	opt := DefaultScanFilmOptions()
	wr := e.Range()
	opt.WorldRange = &wr
	pos, err := ScanBipedPositions(fc, opt)
	if err != nil {
		t.Fatal(err)
	}
	// la generation d une position n est pas portee : on la retrouve par (slot, instant) dans la marche.
	genDe := map[[5]uint64]uint32{}
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			walkDeltaBipedPayload(pay, fc.BipedSlots(), lay, gens.A(pk.TimestampUS), func(r deltaBipedRecord) {
				var q [3]uint64
				for ax := 0; ax < 3; ax++ {
					q[ax] = source.BitsStricts(pay, r.I0+lay.AxisOffset(ax), int(lay.AxisW[ax]))
				}
				genDe[[5]uint64{uint64(r.Slot), pk.TimestampUS, q[0], q[1], q[2]}] = r.Gen
			})
		}
	}
	prod := map[string]int{}
	for _, p := range pos {
		g := genDe[[5]uint64{uint64(p.Slot), p.TimestampUS, uint64(p.Q[0]), uint64(p.Q[1]), uint64(p.Q[2])}]
		c := classe(p.Slot, g, p.TimestampUS)
		prod[c+"/gen"+strconv.Itoa(int(g))]++
		if c != "dans_sa_vie" && c != "avant_creation_premier_corps_gen1" {
			t.Logf("PROD %s slot=%d gen=%d t=%d xyz=(%.2f %.2f %.2f)", c, p.Slot, g, p.TimestampUS, p.X, p.Y, p.Z)
		}
	}
	t.Logf("PRODUCTION %v", prod)
}
