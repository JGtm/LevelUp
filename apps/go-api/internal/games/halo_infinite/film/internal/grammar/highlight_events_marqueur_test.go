package grammar

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// motifXUID écrit un motif de XUID valide : XUID LE sur 8 octets, 0x2d, 0xc0.
func motifXUID(buf *bytes.Buffer, xuid uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, xuid)
	buf.Write(b)
	buf.WriteByte(0x2d)
	buf.WriteByte(0xc0)
}

// Un marqueur de fin = un événement. Reconstitue la fiche d un joueur dont un faux motif de
// XUID précède le vrai de quelques milliers de bits : les deux retombent sur le même bloc de
// 60 octets et le même marqueur de fin. Seul le vrai (le plus proche du marqueur) doit sortir.
func TestParseHighlightEvents_UnMarqueurUnEvenement(t *testing.T) {
	const faux, vrai = uint64(2252025040223036), uint64(2535472452512509)

	var buf bytes.Buffer
	buf.Write(make([]byte, 20))
	motifXUID(&buf, faux)
	buf.Write(make([]byte, 600)) // ~4 800 bits entre le faux et le vrai
	motifXUID(&buf, vrai)
	buf.Write(make([]byte, 30))
	buf.Write(buildEventBytes("Partycz", typeHintKill, 40500, false, 0))
	buf.Write(endMarker)
	buf.Write(make([]byte, 10))

	events, err := ParseHighlightEvents(buf.Bytes(), 40)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("un bloc, un marqueur : attendu 1 événement, obtenu %d (%+v)", len(events), events)
	}
	if events[0].XUID != vrai {
		t.Errorf("XUID retenu = %d, attendu le plus proche du marqueur (%d)", events[0].XUID, vrai)
	}
}

// Deux événements distincts (deux marqueurs) gardent chacun leur XUID.
func TestParseHighlightEvents_DeuxMarqueursDeuxEvenements(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(make([]byte, 20))
	for i, xuid := range []uint64{2535472452512509, 2535472452512510} {
		motifXUID(&buf, xuid)
		buf.Write(make([]byte, 30))
		buf.Write(buildEventBytes("A", typeHintKill, 1000+i*500, false, 0))
		buf.Write(endMarker)
		buf.Write(make([]byte, 10))
	}
	events, err := ParseHighlightEvents(buf.Bytes(), 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("attendu 2 événements, obtenu %d", len(events))
	}
}
