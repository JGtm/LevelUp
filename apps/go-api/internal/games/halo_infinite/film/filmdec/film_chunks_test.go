package filmdec

// film_chunks_test.go — LE PONT VERS `filmsource` REND EXACTEMENT CE QUE L'ANCIEN CHEMIN RENDAIT.
//
// C'est le test qui autorise la migration du lot 1 : si `FilmChunkAt` differait d'un octet de
// `ReadFilmChunk` + `WalkPackets`, tous les balayages migres changeraient de sortie en silence.
// Il tourne sur la MINI-BOBINE reelle (chunks compresses d'un vrai film), pas sur une
// construction — c'est la ou la reconstruction des bornes de payload se prouve.

import (
	"bytes"
	"os"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// miniBobineChunks : la mini-bobine du film 000d5950 (fixture de `replay`, avec sa PROVENANCE).
// Elle N'A PAS de `chunk_00` : ses fichiers sont chunk_01, chunk_02 et chunk_03. C'est exactement
// le cas ou la POSITION dans le film ne vaut pas le NUMERO du chunk.
const miniBobineChunks = "../replay/testdata/minifilm_000d5950"

func chargerMiniBobine(t *testing.T) *filmsource.Film {
	t.Helper()
	if _, err := os.Stat(miniBobineChunks); err != nil {
		t.Fatalf("mini-bobine absente (%s) : %v", miniBobineChunks, err)
	}
	film, err := filmsource.LoadDir(miniBobineChunks, nil)
	if err != nil {
		t.Fatalf("LoadDir : %v", err)
	}
	return film
}

// TestFilmChunkNumbersEgaleCountFilmChunks — le remplacant rend la MEME liste de numeros que la
// boucle `for c := 1; c <= CountFilmChunks(dir); c++` de l'ancien chemin.
func TestFilmChunkNumbersEgaleCountFilmChunks(t *testing.T) {
	film := chargerMiniBobine(t)
	n := CountFilmChunks(miniBobineChunks)
	if n == 0 {
		t.Fatal("CountFilmChunks rend 0 : la comparaison serait vide")
	}
	attendus := make([]int, 0, n)
	for c := 1; c <= n; c++ {
		attendus = append(attendus, c)
	}
	obtenus := FilmChunkNumbers(film)
	if len(obtenus) != len(attendus) {
		t.Fatalf("FilmChunkNumbers = %v, CountFilmChunks en donne %v", obtenus, attendus)
	}
	for i := range attendus {
		if obtenus[i] != attendus[i] {
			t.Fatalf("numero %d : %d, attendu %d", i, obtenus[i], attendus[i])
		}
	}
}

// TestFilmChunkAtEgaleWalkPackets — chunk par chunk, octets ET paquets identiques a l'ancien
// chemin, BORNES COMPRISES : c'est `Start`/`Size` que la conversion reconstruit par contiguite,
// et `Payload(chunk)` les relit pour donner les octets que la grammaire de records consomme.
func TestFilmChunkAtEgaleWalkPackets(t *testing.T) {
	film := chargerMiniBobine(t)
	numeros := FilmChunkNumbers(film)
	if len(numeros) == 0 {
		t.Fatal("aucun chunk de donnees : la comparaison serait vide")
	}
	paquets := 0
	for _, c := range numeros {
		attenduData, err := ReadFilmChunk(miniBobineChunks, c)
		if err != nil {
			t.Fatalf("ReadFilmChunk(%d) : %v", c, err)
		}
		data, pks, ok := FilmChunkAt(film, c)
		if !ok {
			t.Fatalf("FilmChunkAt(%d) : chunk introuvable dans le film", c)
		}
		if !bytes.Equal(data, attenduData) {
			t.Fatalf("chunk %d : %d octets decompresses, %d attendus", c, len(data), len(attenduData))
		}
		attendus := WalkPackets(attenduData)
		if len(pks) != len(attendus) {
			t.Fatalf("chunk %d : %d paquets, WalkPackets en rend %d", c, len(pks), len(attendus))
		}
		for i, a := range attendus {
			if pks[i] != a {
				t.Fatalf("chunk %d paquet %d : %+v, attendu %+v", c, i, pks[i], a)
			}
			if !bytes.Equal(pks[i].Payload(data), a.Payload(attenduData)) {
				t.Fatalf("chunk %d paquet %d : payload different", c, i)
			}
		}
		paquets += len(pks)
	}
	if paquets == 0 {
		t.Fatal("aucun paquet compare : le test ne prouverait rien")
	}
}

// TestFilmChunkAtNumeroAbsent — un numero que le film ne porte pas rend ok=false, jamais un chunk
// voisin. C'est la garde dont vit `bipedSlotBand`, qui demande DELIBEREMENT le chunk d'apres le
// dernier ; l'ancien chemin y recevait une erreur de lecture.
func TestFilmChunkAtNumeroAbsent(t *testing.T) {
	film := chargerMiniBobine(t)
	numeros := FilmChunkNumbers(film)
	apres := numeros[len(numeros)-1] + 1
	if _, _, ok := FilmChunkAt(film, apres); ok {
		t.Fatalf("FilmChunkAt(%d) : ok=true alors que ce chunk n'existe pas", apres)
	}
	// La mini-bobine n'a pas de registre : le lecteur de chunk_00 doit le dire.
	if _, ok := FilmRegistryChunk(film); ok {
		t.Fatal("FilmRegistryChunk : ok=true sur une bobine sans chunk_00")
	}
}

// TestFilmChunkNumbersFilmNil — un film absent traverse les memes portes qu'un repertoire vide :
// aucune panique, aucun chunk. Les balayages y rendent leur erreur « aucun chunk film ».
func TestFilmChunkNumbersFilmNil(t *testing.T) {
	if got := FilmChunkNumbers(nil); len(got) != 0 {
		t.Fatalf("FilmChunkNumbers(nil) = %v, attendu vide", got)
	}
	if _, _, ok := FilmChunkAt(nil, 1); ok {
		t.Fatal("FilmChunkAt(nil, 1) : ok=true")
	}
	if _, ok := FilmRegistryChunk(nil); ok {
		t.Fatal("FilmRegistryChunk(nil) : ok=true")
	}
}

// TestFilmChunkNumbersArretAuPremierTrou — la regle heritee de `CountFilmChunks`, ECRITE : un trou
// de numerotation arrete l'enumeration, meme si des chunks suivent. Elle tombera avec les
// enveloppes de compatibilite (cf. l'en-tete de film_chunks.go).
func TestFilmChunkNumbersArretAuPremierTrou(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"chunk_00.bin", "chunk_01.bin", "chunk_02.bin", "chunk_04.bin"} {
		if err := os.WriteFile(dir+"/"+n, []byte{0}, 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", n, err)
		}
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir : %v", err)
	}
	got := FilmChunkNumbers(film)
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("FilmChunkNumbers = %v, attendu [1 2] (arret au trou du 03)", got)
	}
	if n := CountFilmChunks(dir); n != 2 {
		t.Fatalf("CountFilmChunks = %d, attendu 2 — les deux regles doivent coincider", n)
	}
	if _, ok := FilmRegistryChunk(film); !ok {
		t.Fatal("FilmRegistryChunk : le chunk_00 est present, il doit se lire")
	}
}
