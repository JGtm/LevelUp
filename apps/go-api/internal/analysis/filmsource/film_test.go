package filmsource_test

// film_test.go — LA GRAMMAIRE D3 REVISEE, REGLE PAR REGLE, SUR DES CHUNKS CONSTRUITS.
//
// Aucune fixture binaire ici : chaque chunk est BATI dans le test a partir d'octets connus, donc
// toutes les valeurs attendues se calculent a la main. Une fixture opaque prouverait que le code
// fait ce qu'il fait, pas qu'il fait ce qu'il doit. (La confrontation au REEL est dans
// source_test.go, sur la mini-bobine, et elle compare a `filmdec.WalkPackets`.)

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

const (
	typeDelta    = 0
	typeKeyframe = 2
	typeChunkEnd = 7
)

// paquet : un paquet complet — en-tete 16 octets little-endian [u16 type][2][u32 taille][u64 ts]
// suivi de son payload.
func paquet(typ uint16, ts uint64, payload []byte) []byte {
	return append(enTete(typ, uint32(len(payload)), ts), payload...)
}

// enTete : un en-tete SEUL, avec la taille qu'on veut y declarer — y compris une taille qui ment
// (debordement) ou une taille nulle (paquet degenere).
func enTete(typ uint16, taille uint32, ts uint64) []byte {
	h := make([]byte, 16)
	binary.LittleEndian.PutUint16(h[0:], typ)
	binary.LittleEndian.PutUint32(h[4:], taille)
	binary.LittleEndian.PutUint64(h[8:], ts)
	return h
}

// compresse : le chunk tel qu'il est stocke sur disque (zlib).
func compresse(t *testing.T, clair []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(clair); err != nil {
		t.Fatalf("ecriture zlib : %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("fermeture zlib : %v", err)
	}
	return buf.Bytes()
}

// chargeClair : un film d'un seul chunk, donne en clair et compresse par le test.
func chargeClair(t *testing.T, clair []byte) *filmsource.Film {
	t.Helper()
	f, err := filmsource.Load(filmsource.MemoryChunks{compresse(t, clair)}, nil)
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	return f
}

func TestUnPaquet(t *testing.T) {
	charge := []byte("abcdef")
	f := chargeClair(t, paquet(typeDelta, 1234, charge))

	if got := f.NumChunks(); got != 1 {
		t.Fatalf("NumChunks = %d, attendu 1", got)
	}
	ps := f.Packets(0)
	if len(ps) != 1 {
		t.Fatalf("%d paquet(s), attendu 1", len(ps))
	}
	p := ps[0]
	if p.Chunk != 0 || p.Index != 0 || p.Type != typeDelta || p.TS != 1234 {
		t.Fatalf("paquet = %+v, attendu chunk 0 index 0 type 0 ts 1234", p)
	}
	if !bytes.Equal(p.Payload, charge) {
		t.Fatalf("payload = %q, attendu %q", p.Payload, charge)
	}
}

func TestDeuxPaquets(t *testing.T) {
	a, b := []byte("aaaa"), []byte("bbbbbbb")
	clair := append(paquet(typeDelta, 10, a), paquet(typeKeyframe, 20, b)...)
	f := chargeClair(t, clair)
	ps := f.Packets(0)

	if len(ps) != 2 {
		t.Fatalf("%d paquet(s), attendu 2", len(ps))
	}
	if ps[0].Index != 0 || ps[1].Index != 1 {
		t.Fatalf("index = %d, %d — attendu 0, 1", ps[0].Index, ps[1].Index)
	}
	if ps[1].Type != typeKeyframe || ps[1].TS != 20 || !bytes.Equal(ps[1].Payload, b) {
		t.Fatalf("second paquet = %+v, attendu type 2 ts 20 payload %q", ps[1], b)
	}
	// Le chunk decompresse est exactement ce qui a ete compresse : rien n'a ete perdu au passage,
	// et le second paquet commence bien apres les 16 + 4 octets du premier.
	if got := f.Chunk(0); !bytes.Equal(got, clair) {
		t.Fatalf("chunk decompresse de %d octets, attendu %d identiques", len(got), len(clair))
	}
}

// TestTerminateurTailleZeroEmisPuisArret — regles (2) et (3) : le CHUNK_END de taille 0 des chunks
// de donnees EST emis (c'est ce que fait `filmdec`, et l'abandon de la candidate « arret sur
// taille 0 » tient a lui), et rien de ce qui le suit ne l'est.
func TestTerminateurTailleZeroEmisPuisArret(t *testing.T) {
	clair := paquet(typeDelta, 10, []byte("aaaa"))
	clair = append(clair, enTete(typeChunkEnd, 0, 99)...)
	clair = append(clair, paquet(typeDelta, 30, []byte("cccc"))...)

	ps := chargeClair(t, clair).Packets(0)
	if len(ps) != 2 {
		t.Fatalf("%d paquet(s), attendu 2 (le paquet APRES le terminateur ne doit pas etre emis)", len(ps))
	}
	fin := ps[1]
	if fin.Type != typeChunkEnd || fin.TS != 99 || len(fin.Payload) != 0 {
		t.Fatalf("terminateur = %+v, attendu type 7 ts 99 payload vide", fin)
	}
	if ps[0].TS != 10 {
		t.Fatalf("premier paquet ts = %d, attendu 10", ps[0].TS)
	}
}

// TestTailleZeroNonTerminateurArreteAvantEmission — regle (4) : l'en-tete degenere (taille 0, type
// autre que 7) n'est PAS emis et arrete la marche. Ce motif n'existe que dans `chunk_00`, le
// registre, ou `killsource` s'arretait deja.
func TestTailleZeroNonTerminateurArreteAvantEmission(t *testing.T) {
	clair := paquet(typeDelta, 10, []byte("aaaa"))
	clair = append(clair, enTete(typeDelta, 0, 20)...)
	clair = append(clair, paquet(typeDelta, 30, []byte("cccc"))...)

	ps := chargeClair(t, clair).Packets(0)
	if len(ps) != 1 {
		t.Fatalf("%d paquet(s), attendu 1 (l'en-tete degenere arrete AVANT emission)", len(ps))
	}
	if ps[0].TS != 10 {
		t.Fatalf("paquet retenu ts = %d, attendu 10", ps[0].TS)
	}
}

// TestTerminateurNonVideEmisPuisArret — regle (3) telle qu'elle est ecrite : l'arret suit le TYPE,
// pas la taille. Un CHUNK_END porteur de charge est emis avec elle, puis la marche s'arrete.
func TestTerminateurNonVideEmisPuisArret(t *testing.T) {
	fin := []byte("fin")
	clair := paquet(typeDelta, 10, []byte("aaaa"))
	clair = append(clair, paquet(typeChunkEnd, 20, fin)...)
	clair = append(clair, paquet(typeDelta, 30, []byte("cccc"))...)

	ps := chargeClair(t, clair).Packets(0)
	if len(ps) != 2 {
		t.Fatalf("%d paquet(s), attendu 2", len(ps))
	}
	if ps[1].Type != typeChunkEnd || !bytes.Equal(ps[1].Payload, fin) {
		t.Fatalf("terminateur = %+v, attendu type 7 payload %q", ps[1], fin)
	}
}

// TestDebordementArreteLaMarche — regle (1) : une taille qui deborde du chunk est un en-tete
// incoherent (padding, fin de chunk). Les paquets deja lus restent valides.
func TestDebordementArreteLaMarche(t *testing.T) {
	clair := paquet(typeDelta, 10, []byte("aaaa"))
	clair = append(clair, enTete(typeDelta, 4096, 20)...)
	clair = append(clair, []byte("bb")...)

	ps := chargeClair(t, clair).Packets(0)
	if len(ps) != 1 {
		t.Fatalf("%d paquet(s), attendu 1 (l'en-tete qui deborde arrete la marche)", len(ps))
	}
	// Le debordement se juge sur off+16+taille, PAS sur la seule taille : ici l'en-tete tient dans
	// le chunk, c'est son payload qui n'y tient pas.
	if ps[0].TS != 10 {
		t.Fatalf("paquet retenu ts = %d, attendu 10", ps[0].TS)
	}
}

// TestFluxZlibTronqueRendLePartiel — « un film Theater se termine parfois net » : un flux coupe
// rend ce qui a pu etre decode, jamais les octets compresses.
func TestFluxZlibTronqueRendLePartiel(t *testing.T) {
	clair := octetsPseudoAleatoires(128 << 10)
	comp := compresse(t, clair)
	tronque := comp[:len(comp)*3/5]

	f, err := filmsource.Load(filmsource.MemoryChunks{tronque}, nil)
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	got := f.Chunk(0)
	if len(got) == 0 {
		t.Fatal("flux tronque : aucun octet rendu, attendu le partiel")
	}
	if len(got) >= len(clair) {
		t.Fatalf("flux tronque : %d octets rendus pour %d en clair — le flux n'etait pas coupe", len(got), len(clair))
	}
	if !bytes.Equal(got, clair[:len(got)]) {
		t.Fatal("flux tronque : le partiel n'est pas un prefixe du clair")
	}
}

// TestChunkNonCompresse — certains dumps sont deja decompresses : les octets traversent tels quels
// et se decoupent normalement.
func TestChunkNonCompresse(t *testing.T) {
	clair := paquet(typeDelta, 7, []byte("clair"))
	f, err := filmsource.Load(filmsource.MemoryChunks{clair}, nil)
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	if !bytes.Equal(f.Chunk(0), clair) {
		t.Fatal("chunk non compresse : les octets ont ete transformes")
	}
	ps := f.Packets(0)
	if len(ps) != 1 || !bytes.Equal(ps[0].Payload, []byte("clair")) {
		t.Fatalf("paquets = %+v, attendu un seul payload \"clair\"", ps)
	}
}

// TestPayloadEstUneSousTranche — le contrat memoire du paquet : un Payload PARTAGE le buffer du
// chunk. Ecrire dans le chunk se voit dans le payload. Si un jour quelqu'un copie les payloads,
// ce test rougit — et c'est le pic memoire du decodage unique qui est en jeu.
func TestPayloadEstUneSousTranche(t *testing.T) {
	f := chargeClair(t, paquet(typeDelta, 1, []byte("abcd")))
	p := f.Packets(0)[0]

	chunk := f.Chunk(0)
	chunk[16] = 'Z' // premier octet du payload : 16 octets d'en-tete devant
	if p.Payload[0] != 'Z' {
		t.Fatal("le payload ne partage pas le buffer du chunk (copie detectee)")
	}
	p.Payload[3] = 'Y'
	if chunk[19] != 'Y' {
		t.Fatal("ecrire dans le payload ne se voit pas dans le chunk (copie detectee)")
	}
}

// TestAllPacketsOrdreChunkPuisIndex — l'ordre est celui du film : chunk croissant, puis index dans
// le chunk. Les vues par chunk sont des sous-tranches du meme stockage.
func TestAllPacketsOrdreChunkPuisIndex(t *testing.T) {
	c0 := append(paquet(typeDelta, 1, []byte("a")), paquet(typeDelta, 2, []byte("b"))...)
	c1 := paquet(typeKeyframe, 3, []byte("c"))
	f, err := filmsource.Load(filmsource.MemoryChunks{compresse(t, c0), compresse(t, c1)}, nil)
	if err != nil {
		t.Fatalf("Load : %v", err)
	}

	all := f.AllPackets()
	if len(all) != 3 {
		t.Fatalf("%d paquet(s) au total, attendu 3", len(all))
	}
	attendu := [][2]int{{0, 0}, {0, 1}, {1, 0}}
	for i, a := range attendu {
		if all[i].Chunk != a[0] || all[i].Index != a[1] {
			t.Fatalf("paquet %d = (chunk %d, index %d), attendu (%d, %d)", i, all[i].Chunk, all[i].Index, a[0], a[1])
		}
	}
	if len(f.Packets(0)) != 2 || len(f.Packets(1)) != 1 {
		t.Fatalf("vues par chunk = %d, %d — attendu 2, 1", len(f.Packets(0)), len(f.Packets(1)))
	}
	// La vue d'un chunk a sa capacite bornee : un append ne peut pas ecraser le chunk suivant.
	vue := f.Packets(0)
	if cap(vue) != len(vue) {
		t.Fatalf("capacite de la vue = %d, attendu %d (tranche a trois indices)", cap(vue), len(vue))
	}
}

// TestMetaNilEtFournie — `meta` nil est LICITE (enveloppes, tests) ; fournie, elle est rendue
// telle quelle et le film ne depend pas de la tranche de l'appelant.
func TestMetaNilEtFournie(t *testing.T) {
	clair := paquet(typeDelta, 1, []byte("a"))
	src := filmsource.MemoryChunks{compresse(t, clair)}

	sansMeta, err := filmsource.Load(src, nil)
	if err != nil {
		t.Fatalf("Load sans meta : %v", err)
	}
	if sansMeta.Meta() != nil {
		t.Fatalf("Meta() = %+v, attendu nil", sansMeta.Meta())
	}

	meta := []filmsource.ChunkMeta{{Index: 0, ChunkType: 3, StartMS: 4200}}
	avecMeta, err := filmsource.Load(src, meta)
	if err != nil {
		t.Fatalf("Load avec meta : %v", err)
	}
	if got := avecMeta.Meta(); len(got) != 1 || got[0] != meta[0] {
		t.Fatalf("Meta() = %+v, attendu %+v", got, meta)
	}
	meta[0].ChunkType = 99 // l'appelant dispose de SA tranche
	if avecMeta.Meta()[0].ChunkType != 3 {
		t.Fatal("Meta() suit les mutations de l'appelant : la copie du chargement manque")
	}
}

// TestLoadRefuseUneSourceVide — une source sans chunk n'est pas un film ; une source nulle non
// plus. Les deux se disent par une erreur, jamais par un film vide.
func TestLoadRefuseUneSourceVide(t *testing.T) {
	if _, err := filmsource.Load(filmsource.MemoryChunks{}, nil); err == nil {
		t.Fatal("Load sur une source sans chunk : erreur attendue")
	}
	if _, err := filmsource.Load(nil, nil); err == nil {
		t.Fatal("Load sur une source nulle : erreur attendue")
	}
}

// TestAccesseursHorsBornes — Chunk et Packets rendent nil hors bornes, jamais une panique : un
// balayage qui demande un chunk absent doit degrader, pas tuer l'enfant de cuisson.
func TestAccesseursHorsBornes(t *testing.T) {
	f := chargeClair(t, paquet(typeDelta, 1, []byte("a")))
	for _, i := range []int{-1, 1, 42} {
		if f.Chunk(i) != nil {
			t.Fatalf("Chunk(%d) devrait etre nil", i)
		}
		if f.Packets(i) != nil {
			t.Fatalf("Packets(%d) devrait etre nil", i)
		}
	}
}

// octetsPseudoAleatoires : une suite deterministe et peu compressible (generateur congruentiel
// lineaire), pour que la troncature du flux coupe de la VRAIE donnee et non du remplissage.
func octetsPseudoAleatoires(n int) []byte {
	out := make([]byte, n)
	x := uint32(12345)
	for i := range out {
		x = x*1103515245 + 12345
		out[i] = byte(x >> 16)
	}
	return out
}
