package filmcache

// write_finalise_test.go — LE WRITER NE VALIDE QU'UN FILM FINALISE (lot L3, 2026-09-23).
//
// LE TEMOIN REEL : `testdata/manifeste_partiel_34.json` est la copie a l'octet du manifeste
// d'`ab526724` tel que le cache l'avait valide le 2026-09-22 a 21:35:50 (34 entrees, aucun
// morceau de temps forts), conserve par la reparation O1 sous `ab526724.json.partiel-34`. Le
// match n'est ici qu'un TEMOIN de la forme : aucune valeur du code de production n'en depend.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// manifestePartielTemoin charge le temoin et rend ses entrees.
func manifestePartielTemoin(t *testing.T) ([]byte, []writeManifestChunk) {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("testdata", "manifeste_partiel_34.json"))
	if err != nil {
		t.Fatalf("temoin : %v", err)
	}
	var mf writeManifestJSON
	if err := json.Unmarshal(blob, &mf); err != nil {
		t.Fatalf("temoin illisible : %v", err)
	}
	if len(mf.Chunks) != 34 || Finalise(mf.Chunks, typeDuManifeste) {
		t.Fatalf("temoin inattendu : %d entrees, finalise=%v", len(mf.Chunks),
			Finalise(mf.Chunks, typeDuManifeste))
	}
	return blob, mf.Chunks
}

// poserManifeste ecrit un manifeste tel quel dans un cache neuf, comme l'aurait fait un writer
// d'avant le lot.
func poserManifeste(t *testing.T, root, short string, blob []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "film_manifests"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ManifestPath(root, short), blob, 0o644); err != nil {
		t.Fatal(err)
	}
}

// completerLeTemoin rend la liste que le serveur servait onze secondes plus tard : les 34
// entrees du temoin, puis deux morceaux de replication et le morceau des temps forts.
func completerLeTemoin(entrees []writeManifestChunk) []WriteChunk {
	out := make([]WriteChunk, 0, len(entrees)+3)
	for _, e := range entrees {
		out = append(out, WriteChunk{Index: e.Index, ChunkType: e.ChunkType, StartMS: e.StartMS,
			DurationMS: e.DurationMS, Data: []byte(fmt.Sprintf("c%d", e.Index))})
	}
	return append(out,
		WriteChunk{Index: 34, ChunkType: 2, StartMS: 660116, DurationMS: 20000, Data: []byte("c34")},
		WriteChunk{Index: 35, ChunkType: 2, StartMS: 680117, DurationMS: 1791, Data: []byte("c35")},
		WriteChunk{Index: 36, ChunkType: ChunkTypeTempsForts, StartMS: 681909, DurationMS: 3,
			Data: []byte("c36")},
	)
}

// TestWrite_RefuseUnFilmNonFinalise : une liste sans morceau de temps forts n'est PAS validee —
// ni manifeste (le marqueur de commit), ni morceau orphelin sur le disque.
func TestWrite_RefuseUnFilmNonFinalise(t *testing.T) {
	root := t.TempDir()
	_, entrees := manifestePartielTemoin(t)
	liste := completerLeTemoin(entrees)[:34]

	err := Write(t.Context(), root, "ab526724", liste)
	if !errors.Is(err, ErrFilmNonFinalise) {
		t.Fatalf("Write d'un film sans temps forts : err = %v, attendu ErrFilmNonFinalise", err)
	}
	if _, found, oErr := Open(root, "ab526724"); found || oErr != nil {
		t.Errorf("un manifeste a ete valide pour un film non finalise (found=%v, err=%v)", found, oErr)
	}
	if fichiers, _ := filepath.Glob(filepath.Join(ChunkDir(root, "ab526724"), "chunk_*.bin")); len(fichiers) != 0 {
		t.Errorf("%d morceaux orphelins ecrits pour un film refuse", len(fichiers))
	}
}

// TestWrite_CompleteUnManifestePartiel : LE CAS DU TEMOIN. Un manifeste deja present SANS temps
// forts est REMPLACE par la liste finalisee qui le complete — la seule reecriture permise. Les
// morceaux deja sur disque A LA BONNE TAILLE ne sont pas reecrits (un morceau de taille fausse
// est remplace depuis J2.2 : cf. write_robustesse_test.go).
func TestWrite_CompleteUnManifestePartiel(t *testing.T) {
	root := t.TempDir()
	blob, entrees := manifestePartielTemoin(t)
	poserManifeste(t, root, "ab526724", blob)
	dir := ChunkDir(root, "ab526724")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	dejaLa := filepath.Join(dir, "chunk_33.bin")
	// Meme taille que "c33" (la liste) : un morceau ENTIER deja present est adopte tel quel.
	if err := os.WriteFile(dejaLa, []byte("anc"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Write(t.Context(), root, "ab526724", completerLeTemoin(entrees)); err != nil {
		t.Fatalf("Write de la liste finalisee : %v", err)
	}
	src, found, err := Open(root, "ab526724")
	if err != nil || !found {
		t.Fatalf("Open : found=%v err=%v", found, err)
	}
	if got := len(src.Meta()); got != 37 {
		t.Fatalf("manifeste complete : %d entrees, attendu 37", got)
	}
	if !Finalise(src.Meta(), func(m types.ChunkMeta) int { return m.ChunkType }) {
		t.Error("le manifeste complete ne porte pas le morceau des temps forts")
	}
	if got, _ := os.ReadFile(dejaLa); string(got) != "anc" {
		t.Errorf("morceau deja present reecrit : %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "chunk_36.bin")); err != nil {
		t.Errorf("morceau des temps forts non ecrit : %v", err)
	}
}

// TestWrite_RefuseDeCompleterUnManifesteDivergent : la liste finalisee doit etre un SUR-ENSEMBLE
// EXACT du manifeste partiel, entree par entree. Une entree deja validee qui aurait change de
// debut, de type ou de duree n'est pas une completion, et une entree deja validee ABSENTE de la
// liste non plus : remplacer le manifeste ferait disparaitre un morceau decrit. Dans les quatre
// cas les morceaux deja ecrits ne sont plus dignes de confiance, et rien n'est touche.
//
// LE CAS « OMISE » EST CELUI QUE LA REVUE ADVERSE DU LOT A TROUVE NON GARDE (L3-R5, mutation
// M3b : ignorer l'absence d'un index laissait tous les tests verts).
func TestWrite_RefuseDeCompleterUnManifesteDivergent(t *testing.T) {
	_, entrees := manifestePartielTemoin(t)
	if entrees[5].ChunkType == 1 {
		t.Fatalf("temoin inattendu : l'entree 5 est deja de type 1, le cas « type change » ne mordrait pas")
	}
	cas := []struct {
		nom      string
		deformer func([]WriteChunk) []WriteChunk
	}{
		{"debut decale", func(l []WriteChunk) []WriteChunk { l[5].StartMS++; return l }},
		{"type change", func(l []WriteChunk) []WriteChunk { l[5].ChunkType = 1; return l }},
		{"duree changee", func(l []WriteChunk) []WriteChunk { l[5].DurationMS++; return l }},
		{"entree omise", func(l []WriteChunk) []WriteChunk { return append(l[:5:5], l[6:]...) }},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			root := t.TempDir()
			blob, entrees := manifestePartielTemoin(t)
			poserManifeste(t, root, "ab526724", blob)
			liste := c.deformer(completerLeTemoin(entrees))
			if !Finalise(liste, typeAEcrire) {
				t.Fatalf("la liste deformee n'est plus finalisee : le cas ne teste pas la divergence")
			}

			err := Write(t.Context(), root, "ab526724", liste)
			if !errors.Is(err, ErrManifesteDivergent) {
				t.Fatalf("err = %v, attendu ErrManifesteDivergent", err)
			}
			if got, _ := os.ReadFile(ManifestPath(root, "ab526724")); string(got) != string(blob) {
				t.Error("le manifeste partiel a ete reecrit malgre la divergence")
			}
			if fichiers, _ := filepath.Glob(filepath.Join(ChunkDir(root, "ab526724"), "chunk_*.bin")); len(fichiers) != 0 {
				t.Errorf("%d morceaux ecrits malgre la divergence", len(fichiers))
			}
		})
	}
}

// TestWrite_UnManifestePartielNeSeCompletePasDUneListePartielle : une seconde liste toujours
// non finalisee (plus longue, mais sans temps forts) ne remplace pas le manifeste partiel.
func TestWrite_UnManifestePartielNeSeCompletePasDUneListePartielle(t *testing.T) {
	root := t.TempDir()
	blob, entrees := manifestePartielTemoin(t)
	poserManifeste(t, root, "ab526724", blob)

	err := Write(t.Context(), root, "ab526724", completerLeTemoin(entrees)[:36])
	if !errors.Is(err, ErrFilmNonFinalise) {
		t.Fatalf("err = %v, attendu ErrFilmNonFinalise", err)
	}
	if got, _ := os.ReadFile(ManifestPath(root, "ab526724")); string(got) != string(blob) {
		t.Error("le manifeste partiel a ete remplace par une autre liste partielle")
	}
}
