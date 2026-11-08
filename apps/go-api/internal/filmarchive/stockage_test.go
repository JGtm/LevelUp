package filmarchive

// stockage_test.go — LE CHUNK EN ZSTD (decision D-18) : ecriture, relecture, controle de
// troncature, recompression d'un film existant, mesure et deplacement de l'archive (lot L14).

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
	"github.com/JGtm/LevelUp/libs/film/finalise"
)

// grosChunk : un chunk compressible de `n` octets, distinct par `graine`.
func grosChunk(graine byte, n int) []byte {
	return bytes.Repeat([]byte{graine, 'a', 'b', 'c', graine + 1}, n/5)
}

func filmCompressible() []WriteChunk {
	return []WriteChunk{
		{Index: 0, ChunkType: 1, Data: grosChunk(1, 20000)},
		{Index: 1, ChunkType: 2, DurationMS: 20000, Data: grosChunk(2, 40000)},
		{Index: 2, ChunkType: finalise.ChunkTypeTempsForts, DurationMS: 20000, Data: grosChunk(3, 30000)},
	}
}

func zlibDe(t *testing.T, clair []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(clair); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestWrite_EcritZstdEtRelitLesOctetsRecus — le chunk est plus court sur disque, commence par la
// magie zstd, et tout lecteur rend les octets que le writer avait recus.
func TestWrite_EcritZstdEtRelitLesOctetsRecus(t *testing.T) {
	root := t.TempDir()
	film := filmCompressible()
	if err := Write(t.Context(), root, "feed0001", film); err != nil {
		t.Fatalf("Write: %v", err)
	}
	for _, c := range film {
		sur, err := os.ReadFile(filmcache.CheminDuChunk(filmcache.ChunkDir(root, "feed0001"), c.Index))
		if err != nil {
			t.Fatal(err)
		}
		if !filmcache.EstZstd(sur) || len(sur) >= len(c.Data) {
			t.Errorf("chunk %d : %d octets sur disque pour %d recus, zstd=%v", c.Index, len(sur), len(c.Data), filmcache.EstZstd(sur))
		}
		lu, err := filmcache.LireChunk(root, "feed0001", c.Index)
		if err != nil || !bytes.Equal(lu, c.Data) {
			t.Errorf("LireChunk(%d) differe des octets recus (err %v)", c.Index, err)
		}
	}
	if ok, err := filmcache.FilmComplet(root, "feed0001"); err != nil || !ok {
		t.Fatalf("FilmComplet = %v, %v ; attendu vrai (la taille logique du zstd vaut size_bytes)", ok, err)
	}
	f, found, err := filmcache.LoadFilm(root, "feed0001")
	if err != nil || !found || !bytes.Equal(f.Chunk(1), film[1].Data) {
		t.Fatalf("LoadFilm ne rend pas les octets recus (found=%v err=%v)", found, err)
	}
}

// TestWrite_NeReecritPasUnChunkZstdDejaEntier — l'idempotence porte sur la taille LOGIQUE : un
// chunk zstd deja present n'est ni reecrit ni compte comme remplace.
func TestWrite_NeReecritPasUnChunkZstdDejaEntier(t *testing.T) {
	root := t.TempDir()
	if err := Write(t.Context(), root, "feed0002", filmCompressible()); err != nil {
		t.Fatal(err)
	}
	chemin := filmcache.CheminDuChunk(filmcache.ChunkDir(root, "feed0002"), 1)
	avant, _ := os.Stat(chemin)
	if err := Write(t.Context(), root, "feed0002", filmCompressible()); err != nil {
		t.Fatal(err)
	}
	apres, _ := os.Stat(chemin)
	if !os.SameFile(avant, apres) {
		t.Error("le chunk zstd deja entier a ete reecrit")
	}
}

// ecrireCacheHistorique pose un film comme l'ancien writer l'ecrivait : chunks BRUTS (sauf
// `zlibIndex`, en zlib) et manifeste qui declare la taille de chaque fichier.
func ecrireCacheHistorique(t *testing.T, root, short string, zlibIndex int) []WriteChunk {
	t.Helper()
	film := filmCompressible()
	if err := os.MkdirAll(filmcache.ChunkDir(root, short), 0o755); err != nil {
		t.Fatal(err)
	}
	var mf writeManifestJSON
	for _, c := range film {
		octets := c.Data
		if c.Index == zlibIndex {
			octets = zlibDe(t, c.Data)
		}
		if err := os.WriteFile(filmcache.CheminDuChunk(filmcache.ChunkDir(root, short), c.Index), octets, 0o644); err != nil {
			t.Fatal(err)
		}
		mf.Chunks = append(mf.Chunks, writeManifestChunk{Index: c.Index, ChunkType: c.ChunkType,
			DurationMS: c.DurationMS, SizeBytes: int64(len(octets))})
	}
	blob, _ := json.Marshal(mf)
	if err := os.MkdirAll(filmcache.ManifestsRoot(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filmcache.ManifestPath(root, short), blob, 0o644); err != nil {
		t.Fatal(err)
	}
	return film
}

func TestRecompresser_SimulationNEcritRien(t *testing.T) {
	root := t.TempDir()
	ecrireCacheHistorique(t, root, "cafe0010", 1)
	avant, _ := os.ReadFile(filmcache.CheminDuChunk(filmcache.ChunkDir(root, "cafe0010"), 0))
	rap, err := Recompresser(t.Context(), root, "cafe0010", true)
	if err != nil {
		t.Fatal(err)
	}
	if rap.Convertis != 3 || rap.Gain() <= 0 {
		t.Fatalf("bilan de simulation = %+v, attendu 3 convertis et un gain", rap)
	}
	apres, _ := os.ReadFile(filmcache.CheminDuChunk(filmcache.ChunkDir(root, "cafe0010"), 0))
	if !bytes.Equal(avant, apres) {
		t.Fatal("la simulation a modifie un chunk")
	}
}

// TestRecompresser_BrutEtZlibPuisReprise — les trois chunks (deux bruts, un zlib) passent en zstd,
// les octets lus ne changent pas, le manifeste n'est corrige que pour le chunk zlib, et une seconde
// passe saute tout (reprise).
func TestRecompresser_BrutEtZlibPuisReprise(t *testing.T) {
	root := t.TempDir()
	film := ecrireCacheHistorique(t, root, "cafe0011", 1)
	rap, err := Recompresser(t.Context(), root, "cafe0011", false)
	if err != nil {
		t.Fatal(err)
	}
	if rap.Convertis != 3 || rap.DejaZstd != 0 || !rap.ManifesteCorrige {
		t.Fatalf("premiere passe = %+v", rap)
	}
	for _, c := range film {
		sur, _ := os.ReadFile(filmcache.CheminDuChunk(filmcache.ChunkDir(root, "cafe0011"), c.Index))
		if !filmcache.EstZstd(sur) {
			t.Errorf("chunk %d non converti", c.Index)
		}
		lu, err := filmcache.LireChunk(root, "cafe0011", c.Index)
		if err != nil || !bytes.Equal(lu, c.Data) {
			t.Errorf("chunk %d : octets relus differents (err %v)", c.Index, err)
		}
	}
	if ok, err := filmcache.FilmComplet(root, "cafe0011"); err != nil || !ok {
		t.Fatalf("FilmComplet apres recompression = %v, %v", ok, err)
	}
	if rap, err = Recompresser(t.Context(), root, "cafe0011", false); err != nil || rap.DejaZstd != 3 || rap.Convertis != 0 {
		t.Fatalf("seconde passe = %+v, %v ; attendu 3 sautes", rap, err)
	}
}

// TestRecompresser_LeManifesteBrutResteIntact — pour un film ENTIEREMENT brut, la taille logique ne
// bouge pas : le manifeste n'est pas reecrit.
func TestRecompresser_LeManifesteBrutResteIntact(t *testing.T) {
	root := t.TempDir()
	ecrireCacheHistorique(t, root, "cafe0012", -1)
	avant, _ := os.ReadFile(filmcache.ManifestPath(root, "cafe0012"))
	rap, err := Recompresser(t.Context(), root, "cafe0012", false)
	if err != nil || rap.ManifesteCorrige {
		t.Fatalf("Recompresser = %+v, %v", rap, err)
	}
	apres, _ := os.ReadFile(filmcache.ManifestPath(root, "cafe0012"))
	if !bytes.Equal(avant, apres) {
		t.Fatal("le manifeste d'un film brut a ete reecrit")
	}
}

func TestRecompresser_RefuseUnFluxZlibAbime(t *testing.T) {
	root := t.TempDir()
	ecrireCacheHistorique(t, root, "cafe0013", 1)
	chemin := filmcache.CheminDuChunk(filmcache.ChunkDir(root, "cafe0013"), 1)
	z, _ := os.ReadFile(chemin)
	if err := os.WriteFile(chemin, z[:len(z)/2], 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Recompresser(t.Context(), root, "cafe0013", false); err == nil {
		t.Fatal("un flux zlib abime ne doit pas etre fige en zstd")
	}
	rest, _ := os.ReadFile(chemin)
	if filmcache.EstZstd(rest) {
		t.Fatal("le chunk abime a ete converti")
	}
}

// TestRecompresser_ChunkHorsManifesteRecompresse : le décodeur lit le DOSSIER, pas le manifeste ;
// un chunk présent que le manifeste ne liste pas se recompresse comme les autres.
func TestRecompresser_ChunkHorsManifesteRecompresse(t *testing.T) {
	root := t.TempDir()
	ecrireCacheHistorique(t, root, "cafe0014", -1)
	orphelin := filmcache.CheminDuChunk(filmcache.ChunkDir(root, "cafe0014"), 7)
	if err := os.WriteFile(orphelin, grosChunk(7, 30000), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filmcache.ChunkDir(root, "cafe0014"), "chunk_x.bin.part"), []byte("reste"), 0o644); err != nil {
		t.Fatal(err)
	}
	rap, err := Recompresser(t.Context(), root, "cafe0014", false)
	if err != nil || rap.Chunks != 4 || rap.Convertis != 4 {
		t.Fatalf("bilan = %+v, %v ; attendu 4 chunks (dont l'orphelin) convertis", rap, err)
	}
	sur, _ := os.ReadFile(orphelin)
	if !filmcache.EstZstd(sur) {
		t.Fatal("le chunk hors manifeste n'a pas été recompressé")
	}
}

func TestRecompresser_FilmAbsent(t *testing.T) {
	if _, err := Recompresser(t.Context(), t.TempDir(), "00000000", false); !errors.Is(err, ErrFilmAbsent) {
		t.Fatalf("erreur = %v, attendu ErrFilmAbsent", err)
	}
}

func TestMesurerOccupationEtSupprimer(t *testing.T) {
	root := t.TempDir()
	if occ, err := filmcache.MesurerOccupation(t.Context(), root); err != nil || occ.Films != 0 || occ.Octets != 0 {
		t.Fatalf("archive vide : %+v, %v", occ, err)
	}
	for _, id := range []string{"aaaa0001", "aaaa0002"} {
		if err := Write(t.Context(), root, id, filmCompressible()); err != nil {
			t.Fatal(err)
		}
	}
	occ, err := filmcache.MesurerOccupation(t.Context(), root)
	if err != nil || occ.Films != 2 || occ.Octets <= 0 || occ.Partielle {
		t.Fatalf("occupation = %+v, %v", occ, err)
	}
	annule, cancel := context.WithCancel(t.Context())
	cancel()
	if occ, err = filmcache.MesurerOccupation(annule, root); err != nil || !occ.Partielle {
		t.Fatalf("mesure annulee = %+v, %v ; attendu partielle sans erreur", occ, err)
	}
	if err := Supprimer(root, "aaaa0001"); err != nil {
		t.Fatal(err)
	}
	if err := Supprimer(root, "aaaa0001"); err != nil {
		t.Fatalf("Supprimer d'un film absent : %v", err)
	}
	if _, found, _ := filmcache.Open(root, "aaaa0001"); found {
		t.Fatal("le film supprime est encore lisible")
	}
	if _, err := os.Stat(filmcache.ChunkDir(root, "aaaa0001")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("chunks du film supprime : %v", err)
	}
}

func TestValiderCible(t *testing.T) {
	vide := t.TempDir()
	plein := t.TempDir()
	if err := os.WriteFile(filepath.Join(plein, "x"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValiderCible(vide, false); err != nil {
		t.Errorf("dossier vide refuse : %v", err)
	}
	if err := ValiderCible(filepath.Join(vide, "nouveau"), false); err != nil {
		t.Errorf("dossier inexistant refuse : %v", err)
	}
	if err := ValiderCible(plein, false); !errors.Is(err, ErrCibleNonVide) {
		t.Errorf("dossier non vide : %v, attendu ErrCibleNonVide", err)
	}
	if err := ValiderCible("relatif/dossier", false); err == nil {
		t.Error("un chemin relatif doit etre refuse")
	}
}

func TestDeplacer_RangeLesFilmsEtRefuseUneCibleNonVide(t *testing.T) {
	origine, cible := t.TempDir(), filepath.Join(t.TempDir(), "archive")
	film := filmCompressible()
	for _, id := range []string{"bbbb0001", "bbbb0002"} {
		if err := Write(t.Context(), origine, id, film); err != nil {
			t.Fatal(err)
		}
	}
	n, err := Deplacer(t.Context(), origine, cible, OptionsDeplacement{})
	if err != nil || n != 2 {
		t.Fatalf("Deplacer = %d, %v", n, err)
	}
	if ids, _ := filmcache.ListShortIDs(origine); len(ids) != 0 {
		t.Errorf("films restes a la source : %v", ids)
	}
	lu, err := filmcache.LireChunk(cible, "bbbb0002", 1)
	if err != nil || !bytes.Equal(lu, film[1].Data) {
		t.Fatalf("chunk deplace illisible : %v", err)
	}
	if ok, err := filmcache.FilmComplet(cible, "bbbb0001"); err != nil || !ok {
		t.Fatalf("film deplace incomplet : %v, %v", ok, err)
	}
	if err := Write(t.Context(), origine, "bbbb0003", film); err != nil {
		t.Fatal(err)
	}
	if _, err := Deplacer(t.Context(), origine, cible, OptionsDeplacement{}); !errors.Is(err, ErrCibleNonVide) {
		t.Fatalf("cible non vide : %v, attendu ErrCibleNonVide", err)
	}
	if ids, _ := filmcache.ListShortIDs(origine); len(ids) != 1 {
		t.Error("un deplacement refuse a touche a la source")
	}
}
