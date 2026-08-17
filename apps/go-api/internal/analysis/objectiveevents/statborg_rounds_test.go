package objectiveevents

import (
	"context"
	"encoding/binary"
	"testing"
)

// statborg_rounds_test.go — les corrections de production du 2026-08-18 (lot A, item A.1.0),
// figees sur des VECTEURS REELS extraits du film `24dbb67d` (Ranked:Oddball, deux manches).
//
// Ce que chaque vecteur prouve :
//
//	vecRound0  un enregistrement de la PREMIERE manche : ce que l'ancienne grammaire lisait deja.
//	vecRound1  un enregistrement de la DEUXIEME manche : l'ancienne grammaire le REJETAIT, parce
//	           qu'elle exigeait les deux en-tetes de 5 bits nuls alors qu'ils portent la manche.
//	vecDense   un enregistrement a liste DENSE (gate = 1, masque de 64 bits) : l'ancienne
//	           grammaire ne connaissait que la liste creuse et le rejetait aussi.
//
// Les octets viennent du film, pas d'une construction : c'est ce qui rend ces tests capables de
// detecter une regression de cadrage, qu'un vecteur synthetique ne verrait pas.

// statVector est un enregistrement reel, sa position de bit dans la tranche, et ce que le
// decodage doit rendre.
type statVector struct {
	data  []byte
	bits  int
	slot  int
	round int
	comps map[int]StatValue
}

// round0 : slot 8, manche 0, 3 composants
var vecRound0 = statVector{
	bits: 1, slot: 8, round: 0,
	comps: map[int]StatValue{22: {A: 10, B: 10}, 23: {A: 1, B: 0}, 0: {A: 1, B: 0}},
	data:  []byte{0x80, 0x11, 0x18, 0x0b, 0x2e, 0x00, 0x00, 0x20, 0x00, 0x00, 0x01, 0x40, 0x50, 0x00, 0x00, 0x20, 0x01, 0x00, 0x4a, 0x30, 0x16, 0x5c, 0x00, 0x00, 0x40, 0x00, 0x00, 0x02, 0x80, 0xa0, 0x00, 0x00, 0x40, 0x02, 0x07, 0xb4, 0x20, 0x60, 0x44, 0x00, 0x8c, 0x00, 0x59, 0x02, 0xf6, 0xdc, 0xfd, 0x44, 0x40, 0xce, 0x01, 0x6a, 0x18, 0x65, 0xf0, 0x11, 0x00, 0xa3, 0x00, 0x56, 0x40, 0xbd, 0xd0, 0xbf},
}

// dense : slot 6, manche 0, 8 composants
var vecDense = statVector{
	bits: 3, slot: 6, round: 0,
	comps: map[int]StatValue{1: {A: 0, B: 535}, 2: {A: 3, B: 1}, 3: {A: 3, B: 300}, 5: {A: 114, B: 2}, 6: {A: 2, B: 49}, 11: {A: 300, B: 0}, 12: {A: 3, B: 3}, 21: {A: 1, B: 0}},
	data:  []byte{0xa0, 0x03, 0x50, 0x00, 0x00, 0x00, 0x00, 0x02, 0x01, 0x86, 0xe0, 0x00, 0x00, 0x40, 0x85, 0xc0, 0x00, 0x03, 0x00, 0x40, 0x00, 0x03, 0x40, 0x4b, 0x14, 0x04, 0xb0, 0x00, 0x1c, 0x80, 0x28, 0x72, 0x00, 0x00, 0x20, 0xc4, 0x00, 0x10, 0x12, 0xc0, 0x00, 0x00, 0x00, 0x30, 0x0c, 0x00, 0x00, 0x10, 0x00, 0x80, 0x11, 0x28, 0x01, 0x1a, 0xb2, 0xe0, 0x00, 0x06, 0x00, 0x00, 0x00, 0x02, 0x01, 0x80},
}

// round1 : slot 8, manche 1, 3 composants
var vecRound1 = statVector{
	bits: 1, slot: 8, round: 1,
	comps: map[int]StatValue{23: {A: 1, B: 0}, 0: {A: 1, B: 0}, 22: {A: 10, B: 10}},
	data:  []byte{0x80, 0x11, 0x18, 0x0b, 0x2e, 0x10, 0x80, 0x20, 0x00, 0x10, 0x81, 0x40, 0x50, 0x10, 0x80, 0x20, 0x01, 0x00, 0x62, 0x30, 0x16, 0x5c, 0x21, 0x00, 0x40, 0x00, 0x21, 0x02, 0x80, 0xa0, 0x21, 0x00, 0x40, 0x02, 0x07, 0xc4, 0x20, 0x51, 0xc4, 0x6a, 0x94, 0x00, 0x45, 0x55, 0x90, 0x2f, 0x9d, 0x90, 0x01, 0x8c, 0x09, 0x20, 0xb1, 0xea, 0x44, 0x58, 0x78, 0x00, 0x03, 0x3a, 0x63, 0x0f, 0xda, 0x34},
}

// TestStatborgVectorsReels verifie le decodage de chaque vecteur : slot, manche et valeurs.
func TestStatborgVectorsReels(t *testing.T) {
	for name, v := range map[string]statVector{
		"manche 1 (liste creuse)": vecRound0,
		"manche 2 (liste creuse)": vecRound1,
		"liste dense":             vecDense,
	} {
		t.Run(name, func(t *testing.T) {
			slot, idx, at, ok := matchRecordHeader(v.data, v.bits)
			if !ok {
				t.Fatalf("en-tete non reconnu")
			}
			if slot != v.slot {
				t.Errorf("slot = %d, attendu %d", slot, v.slot)
			}
			comps, round := decodeComponents(v.data, at, idx)
			if round != v.round {
				t.Errorf("manche = %d, attendue %d", round, v.round)
			}
			for i, want := range v.comps {
				got, ok := comps[i]
				if !ok {
					t.Errorf("composant %d absent", i)
					continue
				}
				if got != want {
					t.Errorf("composant %d = %+v, attendu %+v", i, got, want)
				}
			}
		})
	}
}

// TestStatborgManche2RejeteeParLAncienneGrammaire fige la RAISON de la correction : l'assertion
// « les deux en-tetes valent 0 » rejetait le vecteur de la deuxieme manche. Si un jour quelqu'un
// la remet, ce test tombe.
func TestStatborgManche2RejeteeParLAncienneGrammaire(t *testing.T) {
	_, idx, at, ok := matchRecordHeader(vecRound1.data, vecRound1.bits)
	if !ok {
		t.Fatal("en-tete non reconnu")
	}
	h1 := readBitsBE(vecRound1.data, at, statHdrBits)
	h2 := readBitsBE(vecRound1.data, at+statHdrBits, statHdrBits)
	if h1 == 0 && h2 == 0 {
		t.Fatal("les deux en-tetes sont nuls : ce vecteur ne prouve plus rien")
	}
	if h1 != h2 {
		t.Errorf("en-tetes = %d et %d : sur une emission reelle ils portent la MEME manche", h1, h2)
	}
	if comps, round := decodeComponents(vecRound1.data, at, idx); len(comps) == 0 || round != 1 {
		t.Errorf("manche 2 non decodee (comps=%d, round=%d)", len(comps), round)
	}
	_ = idx
}

// TestStatborgListeDenseLue fige la lecture de la forme dense : masque de 64 bits, et non une
// liste creuse de sept index au plus.
func TestStatborgListeDenseLue(t *testing.T) {
	if got := readBitsBE(vecDense.data, vecDense.bits+statIDBits+statGenBits, 1); got != 1 {
		t.Fatalf("ce vecteur n'est pas en forme dense (gate = %d)", got)
	}
	_, idx, _, ok := matchRecordHeader(vecDense.data, vecDense.bits)
	if !ok {
		t.Fatal("en-tete dense non reconnu")
	}
	if len(idx) <= statMaxCompPerRecord {
		t.Errorf("%d composants annonces : une liste creuse en porte au plus %d, "+
			"ce vecteur ne prouverait pas la forme dense", len(idx), statMaxCompPerRecord)
	}
}

// TestStatRecordsPlafond verifie la garde memoire : au-dela du plafond, la lecture s'arrete et le
// resultat est marque tronque. La source rend le meme paquet en boucle — un film pathologique.
func TestStatRecordsPlafond(t *testing.T) {
	recs, truncated := StatRecordsCtx(context.Background(), repeatingSource{data: vecRound0.data}, "test")
	if !truncated {
		t.Fatal("le plafond n'a pas ete atteint : la garde ne protege rien")
	}
	if len(recs) < statMaxRecordsPerFilm {
		t.Errorf("%d enregistrements rendus, attendu au moins %d", len(recs), statMaxRecordsPerFilm)
	}
}

// repeatingSource rend le meme chunk un grand nombre de fois : elle simule un film dont le
// balayage ne converge pas, sans avoir besoin du film de 3,3 Go qui a motive le plafond.
type repeatingSource struct{ data []byte }

// repeatChunks / repeatPackets dimensionnent la source pour depasser le plafond : chaque paquet
// porte au moins un enregistrement, donc leur produit doit exceder statMaxRecordsPerFilm.
const (
	repeatChunks  = 200
	repeatPackets = 200
)

func (r repeatingSource) Chunks() []ChunkMeta {
	out := make([]ChunkMeta, repeatChunks)
	for i := range out {
		out[i] = ChunkMeta{Index: i + 1, ChunkType: 0, StartMS: i * 1000}
	}
	return out
}

// ChunkData fabrique un chunk NON compresse (le premier octet vaut 0, pas la marque zlib) fait
// de repeatPackets paquets FRAME portant tous le meme enregistrement.
func (r repeatingSource) ChunkData(int) ([]byte, bool) {
	out := make([]byte, 0, repeatPackets*(packetHdr+len(r.data)))
	for i := 0; i < repeatPackets; i++ {
		hdr := make([]byte, packetHdr)
		binary.LittleEndian.PutUint16(hdr[0:], packetFrame)
		binary.LittleEndian.PutUint32(hdr[4:], uint32(len(r.data)))
		binary.LittleEndian.PutUint64(hdr[8:], uint64(i)*1000)
		out = append(out, hdr...)
		out = append(out, r.data...)
	}
	return out, true
}
