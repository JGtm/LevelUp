package signaux

import (
	"encoding/binary"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// statborg_lecture_test.go — les corrections de production du 2026-08-18 (lot A, item A.1.0),
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
	comps map[int]types.StatValue
}

// round0 : slot 8, manche 0, 3 composants
var vecRound0 = statVector{
	bits: 1, slot: 8, round: 0,
	comps: map[int]types.StatValue{22: {A: 10, B: 10}, 23: {A: 1, B: 0}, 0: {A: 1, B: 0}},
	data:  []byte{0x80, 0x11, 0x18, 0x0b, 0x2e, 0x00, 0x00, 0x20, 0x00, 0x00, 0x01, 0x40, 0x50, 0x00, 0x00, 0x20, 0x01, 0x00, 0x4a, 0x30, 0x16, 0x5c, 0x00, 0x00, 0x40, 0x00, 0x00, 0x02, 0x80, 0xa0, 0x00, 0x00, 0x40, 0x02, 0x07, 0xb4, 0x20, 0x60, 0x44, 0x00, 0x8c, 0x00, 0x59, 0x02, 0xf6, 0xdc, 0xfd, 0x44, 0x40, 0xce, 0x01, 0x6a, 0x18, 0x65, 0xf0, 0x11, 0x00, 0xa3, 0x00, 0x56, 0x40, 0xbd, 0xd0, 0xbf},
}

// dense : slot 6, manche 0, 8 composants
var vecDense = statVector{
	bits: 3, slot: 6, round: 0,
	// LES CANAUX CONDITIONNELS SONT VISIBLES DEPUIS LE 2026-08-31, et ce vecteur est le premier
	// a les montrer : deux composants sur huit en portent un. Ils ne sont PAS un re-lecture des
	// canaux inconditionnels — les deux se lisent a la MEME position relative (juste apres les
	// deux bits de drapeau) et rendent pourtant des valeurs differentes (114 pour l'un, 300 pour
	// l'autre). Qu'ils EGALENT ici A et B est une coincidence de ce vecteur, pas une identite :
	// un compteur et son maximum de session coincident tant que le maximum vient d'etre atteint.
	comps: map[int]types.StatValue{
		1: {A: 0, B: 535}, 2: {A: 3, B: 1},
		3:  {A: 3, B: 300, D: 300, HasD: true},
		5:  {A: 114, B: 2, C: 114, HasC: true},
		6:  {A: 2, B: 49},
		11: {A: 300, B: 0}, 12: {A: 3, B: 3}, 21: {A: 1, B: 0},
	},
	data: []byte{0xa0, 0x03, 0x50, 0x00, 0x00, 0x00, 0x00, 0x02, 0x01, 0x86, 0xe0, 0x00, 0x00, 0x40, 0x85, 0xc0, 0x00, 0x03, 0x00, 0x40, 0x00, 0x03, 0x40, 0x4b, 0x14, 0x04, 0xb0, 0x00, 0x1c, 0x80, 0x28, 0x72, 0x00, 0x00, 0x20, 0xc4, 0x00, 0x10, 0x12, 0xc0, 0x00, 0x00, 0x00, 0x30, 0x0c, 0x00, 0x00, 0x10, 0x00, 0x80, 0x11, 0x28, 0x01, 0x1a, 0xb2, 0xe0, 0x00, 0x06, 0x00, 0x00, 0x00, 0x02, 0x01, 0x80},
}

// round1 : slot 8, manche 1, 3 composants
var vecRound1 = statVector{
	bits: 1, slot: 8, round: 1,
	comps: map[int]types.StatValue{23: {A: 1, B: 0}, 0: {A: 1, B: 0}, 22: {A: 10, B: 10}},
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
	h1 := source.BitsTronques(vecRound1.data, at, statHdrBits)
	h2 := source.BitsTronques(vecRound1.data, at+statHdrBits, statHdrBits)
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
	if got := source.BitsTronques(vecDense.data, vecDense.bits+statIDBits+statGenBits, 1); got != 1 {
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
// resultat est marque tronque, au chunk ou le plafond tombe. La source rend le meme paquet en
// boucle — un film pathologique.
func TestStatRecordsPlafond(t *testing.T) {
	l := LireLeStatborg(filmRepete(t, vecRound0.data))
	if !l.Tronque {
		t.Fatal("le plafond n'a pas ete atteint : la garde ne protege rien")
	}
	if len(l.Records) < StatborgEnregistrementsMax {
		t.Errorf("%d enregistrements rendus, attendu au moins %d", len(l.Records), StatborgEnregistrementsMax)
	}
	if l.ChunkTronque < 1 || l.ChunkTronque > repeatChunks || l.ChunksDatables != repeatChunks {
		t.Errorf("troncature au chunk %d, %d chunk(s) datable(s) : attendu un numero du manifeste "+
			"et les %d chunks", l.ChunkTronque, l.ChunksDatables, repeatChunks)
	}
}

// TestUnFilmSansManifesteNEstPasDatable : un film charge sans metadonnees porte des paquets mais
// aucun `start_ms` — la lecture ne rend rien ET le dit (zero chunk datable), au lieu de se lire
// comme un film qui ne porte rien.
func TestUnFilmSansManifesteNEstPasDatable(t *testing.T) {
	film, err := source.Load(source.MemoryChunks{chunkRepete(vecRound0.data)}, nil)
	if err != nil {
		t.Fatalf("chargement du film sans manifeste : %v", err)
	}
	if l := LireLeStatborg(film); l.ChunksDatables != 0 || len(l.Records) != 0 || l.Tronque {
		t.Errorf("film sans manifeste : %d chunk(s) datable(s), %d enregistrement(s), tronque %v",
			l.ChunksDatables, len(l.Records), l.Tronque)
	}
	if l := LireLeStatborg(nil); l.ChunksDatables != 0 || l.Records != nil {
		t.Errorf("film absent : %+v, attendu une lecture vide", l)
	}
}

// filmRepete fabrique un film dont chaque chunk rend le meme enregistrement un grand nombre de
// fois : il simule un film dont le balayage ne converge pas, sans avoir besoin du film de 3,3 Go
// qui a motive le plafond.
//
// LE MANIFESTE EST SYNTHETISE AVEC UN TYPE DE JEU (2) : la lecture ne balaie que les chunks que
// le manifeste decrit, et un type ZERO signifierait « chunk hors manifeste » (cf.
// [manifestChunks]).
func filmRepete(t *testing.T, data []byte) *source.Film {
	t.Helper()
	chunks := make(source.MemoryChunks, repeatChunks)
	meta := make([]types.ChunkMeta, repeatChunks)
	brut := chunkRepete(data)
	for i := range chunks {
		chunks[i] = brut
		meta[i] = types.ChunkMeta{Index: i + 1, ChunkType: 2, StartMS: i * 1000}
	}
	film, err := source.Load(chunks, meta)
	if err != nil {
		t.Fatalf("chargement du film repete : %v", err)
	}
	return film
}

// repeatChunks / repeatPackets dimensionnent la source pour depasser le plafond : chaque paquet
// porte au moins un enregistrement, donc leur produit doit exceder statMaxRecordsPerFilm.
const (
	repeatChunks  = 200
	repeatPackets = 200
	// repeatHdrSize est la taille de l'en-tete de paquet, telle que `source` la decoupe.
	repeatHdrSize = 16
)

// chunkRepete fabrique un chunk NON compresse (le premier octet vaut 0, pas la marque zlib) fait
// de repeatPackets paquets FRAME portant tous le meme enregistrement.
func chunkRepete(data []byte) []byte {
	out := make([]byte, 0, repeatPackets*(repeatHdrSize+len(data)))
	for i := range repeatPackets {
		hdr := make([]byte, repeatHdrSize)
		binary.LittleEndian.PutUint16(hdr[0:], typeDeTrame)
		binary.LittleEndian.PutUint32(hdr[4:], uint32(len(data)))
		binary.LittleEndian.PutUint64(hdr[8:], uint64(i)*1000)
		out = append(out, hdr...)
		out = append(out, data...)
	}
	return out
}

// LES CONTROLES NEGATIFS DES GARDES D'ANCRAGE (revue R1, 2026-08-18). Les tests ci-dessus
// prouvent que la grammaire relachee LIT ce qu'elle doit lire (manche 2, forme dense, plafond
// memoire). Ceux-ci prouvent qu'elle REFUSE ce qu'elle doit refuser : la contrainte « les deux
// en-tetes de 5 bits sont nuls » a saute, et ce sont deux autres contraintes qui la remplacent.
// Chaque test ci-dessous ECHOUE si la garde qu'il vise est retiree.

// setBitsBE ecrit n bits big-endian a bitPos dans une COPIE de data. C'est l'inverse exact de
// la lecture `source.BitsTronques`, et il ne sert qu'a fabriquer des vecteurs NEGATIFS a partir
// de vecteurs reels : on part d'un enregistrement qui se decode, et on casse UNE contrainte a la
// fois.
func setBitsBE(data []byte, bitPos, n int, v uint64) []byte {
	out := append([]byte(nil), data...)
	for i := range n {
		bit := (v >> uint(n-1-i)) & 1
		p := bitPos + i
		mask := byte(1) << uint(7-p%8)
		if bit == 1 {
			out[p/8] |= mask
		} else {
			out[p/8] &^= mask
		}
	}
	return out
}

// TestDecodeComponentsRefuseUnCoupleDepareille — LA PREMIERE GARDE.
//
// Les deux en-tetes de 5 bits portent le MEME numero de manche : ils sont redondants dans le
// format, et c'est cette redondance qui remplace la contrainte « nuls » comme filtre
// anti-faux-positifs. Un couple depareille est un ancrage fortuit.
func TestDecodeComponentsRefuseUnCoupleDepareille(t *testing.T) {
	_, idx, at, ok := matchRecordHeader(vecRound0.data, vecRound0.bits)
	if !ok {
		t.Fatal("le vecteur de reference ne s'ancre plus — revoir les vecteurs avant ce test")
	}
	if comps, _ := decodeComponents(vecRound0.data, at, idx); len(comps) == 0 {
		t.Fatal("le vecteur de reference ne se decode plus")
	}
	// Le SECOND en-tete passe de 0 a 1 : les deux ne disent plus la meme manche.
	casse := setBitsBE(vecRound0.data, at+statHdrBits, statHdrBits, 1)
	if comps, _ := decodeComponents(casse, at, idx); len(comps) != 0 {
		t.Errorf("un couple d'en-tetes DEPAREILLE (0 puis 1) a ete accepte : %d composant(s) — "+
			"la garde qui remplace « en-tetes nuls » ne filtre plus rien", len(comps))
	}
}

// TestDecodeComponentsRefuseUneMancheHorsBorne — LA SECONDE GARDE.
//
// Le numero de manche est borne (StatborgMancheMax) : huit manches sont au-dela de tout format
// observe, et la borne conserve deux bits de contrainte sur chacun des deux en-tetes. Sans elle,
// l'ancrage laisserait passer 151 faux positifs par film (mesure d'A.1.0).
func TestDecodeComponentsRefuseUneMancheHorsBorne(t *testing.T) {
	_, idx, at, ok := matchRecordHeader(vecRound0.data, vecRound0.bits)
	if !ok {
		t.Fatal("le vecteur de reference ne s'ancre plus")
	}
	horsBorne := uint64(StatborgMancheMax + 1)
	casse := setBitsBE(vecRound0.data, at, statHdrBits, horsBorne)
	casse = setBitsBE(casse, at+statHdrBits, statHdrBits, horsBorne)
	// Les deux en-tetes CONCORDENT : seule la borne peut refuser ce vecteur.
	if comps, round := decodeComponents(casse, at, idx); len(comps) != 0 {
		t.Errorf("une manche %d (borne %d) a ete acceptee : %d composant(s), round=%d",
			horsBorne, StatborgMancheMax, len(comps), round)
	}
}
