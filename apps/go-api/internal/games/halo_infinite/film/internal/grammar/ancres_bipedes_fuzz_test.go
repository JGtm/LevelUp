package grammar

// ancres_bipedes_fuzz_test.go — LA RECUPERATION ANCREE SOUS LE HARNAIS DE FUZZ (lot 2.4 du plan de
// l etape 2, item 2.4.3) : l ancrage d un payload QUELCONQUE, le rangement compact de chaque record
// et sa relecture, puis la lecture de sa position, ne paniquent pas, et ce qui est ancre est borne
// par les bits du payload. Le harnais est [FuzzFilmRecordReaders] : ses graines rejouent ce contrat
// a chaque `go test`.

import (
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// bandeDeTousLesSlots rend la bande de tous les slots d un handle : l ancrage n y ecarte aucun slot.
func bandeDeTousLesSlots() SlotBand {
	tous := make(map[uint32]bool, 1<<handleSlotBits)
	for s := range uint32(1 << handleSlotBits) {
		tous[s] = true
	}
	return NewSlotBand(tous)
}

// recupererUnPayloadQuelconque ancre `pay` sous `bande`, range chaque record et le relit, puis en lit
// la position avec la capture des directions et de la vitalite. Bornes : l i0 de chaque record tient
// dans le payload, son masque compte de deux a sept index, et deux records ne se chevauchent pas
// (l avance du marcheur est d un i0 au moins).
func recupererUnPayloadQuelconque(t *testing.T, pay []byte, bande SlotBand) {
	t.Helper()
	lay := profile.I0Layout{GateBits: profile.DefaultI0GateBits, AxisW: [3]uint{15, 15, 17}}
	wr := profile.Vec3Range{{Min: -100, Max: 100}, {Min: -100, Max: 100}, {Min: -100, Max: 100}}
	opt := ScanFilmOptions{WorldRange: &wr, CaptureDirs: true, DropSaturated: true}
	br := LecteurSur(pay)
	br.PoserContexte(ContexteParDefaut())
	pk := FilmPacket{Type: PacketTypeDelta, Size: len(pay)}
	bits, precedent := len(pay)*8, -1
	walkDeltaBipedPayload(pay, bande, lay, nil, func(r deltaBipedRecord) {
		if r.I0+lay.TotalBits() > bits || len(r.Mask) < bipedMinMaskCnt || len(r.Mask) > bipedMaxMaskCnt ||
			r.I0 <= precedent {
			t.Fatalf("payload de %d bits : record ancre hors des bornes (i0 %d, masque %v, i0 precedent %d)", bits,
				r.I0, r.Mask, precedent)
		}
		precedent = r.I0
		r.Chunk, r.Packet = 0, pk
		if relu := recordAncreDe(r).enRecord(pay, 0, pk); relu.I0 != r.I0 || relu.Slot != r.Slot ||
			relu.Gen != r.Gen || relu.Total != r.Total || !slices.Equal(relu.Mask, r.Mask) {
			t.Fatalf("record range puis relu : %+v, ancre : %+v", relu, r)
		}
		_, _ = lireLaPosition(br, r, lay, opt, grammaireDOrientation(opt))
	})
}

// TestLaRecuperationAncreeDUnPayloadQuelconqueResteBornee : les cas limites du harnais, hors fuzz —
// payload vide, un octet, des octets pleins.
func TestLaRecuperationAncreeDUnPayloadQuelconqueResteBornee(t *testing.T) {
	bande := bandeDeTousLesSlots()
	for _, pay := range [][]byte{nil, {0x00}, {0xff}, {0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}} {
		recupererUnPayloadQuelconque(t, pay, bande)
	}
}
