package replaybuild

// filmfacts_non_finalise_test.go — LA CUISSON REFUSE UN FILM NON FINALISE (lot L3, 2026-09-23).
//
// LE TEMOIN REEL est le manifeste partiel d'`ab526724` (34 entrees, aucun morceau des temps
// forts), conserve par la reparation O1 et copie a l'octet dans
// `film/filmcache/testdata/manifeste_partiel_34.json`. Le repertoire reconstitue ici porte ses 37
// morceaux, dont 3 que le manifeste ne decrit pas : exactement l'etat du cache au moment ou la
// cuisson du 2026-09-22 a charge le film. Le match n'est qu'un TEMOIN de la forme.
//
// COMME `cle_inconnue_test.go`, LES TESTS MORDENT SUR LES DEUX GARDES ET NON SUR `BuildBytes`,
// qui exige le catalogue de bornes et les libelles du titre (donnees de `data/`). Le point
// d insertion est garde par la compilation : chaque garde n a qu un appelant,
// `entreesDeLaCuisson`, qui les appelle avant toute lecture du film.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// temoinPartiel rend les entrees du manifeste partiel d'`ab526724`.
func temoinPartiel(t *testing.T) []filmcache.WriteChunk {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("..", "games", "halo_infinite", "film", "filmcache",
		"testdata", "manifeste_partiel_34.json"))
	if err != nil {
		t.Fatalf("temoin : %v", err)
	}
	var mf struct {
		Chunks []struct {
			Index      int `json:"index"`
			ChunkType  int `json:"chunk_type"`
			StartMS    int `json:"start_ms"`
			DurationMS int `json:"duration_ms"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(blob, &mf); err != nil || len(mf.Chunks) != 34 {
		t.Fatalf("temoin illisible ou inattendu (%d entrees) : %v", len(mf.Chunks), err)
	}
	out := make([]filmcache.WriteChunk, 0, len(mf.Chunks))
	for _, c := range mf.Chunks {
		out = append(out, filmcache.WriteChunk{Index: c.Index, ChunkType: c.ChunkType,
			StartMS: c.StartMS, DurationMS: c.DurationMS, Data: []byte(fmt.Sprintf("c%d", c.Index))})
	}
	return out
}

// repertoireDuTemoin archive le film COMPLET (37 morceaux) puis remplace son manifeste par
// `entrees` : c est l etat « manifeste valide trop tot, morceaux arrives ensuite ». Rend la racine
// du cache, le repertoire de morceaux et la source du manifeste.
func repertoireDuTemoin(t *testing.T, entrees []filmcache.WriteChunk) (string, *filmcache.Source) {
	t.Helper()
	racine := t.TempDir()
	partiel := temoinPartiel(t)
	complet := append(append([]filmcache.WriteChunk(nil), partiel...),
		filmcache.WriteChunk{Index: 34, ChunkType: 2, StartMS: 660116, DurationMS: 20000, Data: []byte("c34")},
		filmcache.WriteChunk{Index: 35, ChunkType: 2, StartMS: 680117, DurationMS: 1791, Data: []byte("c35")},
		filmcache.WriteChunk{Index: 36, ChunkType: filmcache.ChunkTypeTempsForts, StartMS: 681909,
			DurationMS: 3, Data: []byte("c36")},
	)
	if err := filmcache.Write(racine, "ab526724", complet); err != nil {
		t.Fatalf("archivage du film complet : %v", err)
	}
	type entree struct {
		Index      int `json:"index"`
		ChunkType  int `json:"chunk_type"`
		StartMS    int `json:"start_ms"`
		DurationMS int `json:"duration_ms"`
	}
	mf := struct {
		Chunks []entree `json:"chunks"`
	}{}
	for _, c := range entrees {
		mf.Chunks = append(mf.Chunks, entree{c.Index, c.ChunkType, c.StartMS, c.DurationMS})
	}
	blob, err := json.Marshal(mf)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filmcache.ManifestPath(racine, "ab526724"), blob, 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filmcache.ChunkDir(racine, "ab526724")
	src, found, err := filmcache.OpenChunkDir(dir)
	if err != nil || !found {
		t.Fatalf("manifeste du temoin : found=%v err=%v", found, err)
	}
	return dir, src
}

// verifierRefusEcarte : l erreur est la sentinelle, et son TEXTE la porte (frontiere de processus).
func verifierRefusEcarte(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, filmcache.ErrFilmNonFinalise) {
		t.Fatalf("err = %v, attendu filmcache.ErrFilmNonFinalise", err)
	}
	if !strings.Contains(err.Error(), filmcache.ErrFilmNonFinalise.Error()) {
		t.Errorf("le texte ne porte pas la sentinelle : le parent classerait en ECHEC : %q", err)
	}
}

// TestCuisson_RefuseLeTemoinPartiel : « 34 au manifeste + 3 hors manifeste », sans temps forts —
// refuse AVANT le chargement.
func TestCuisson_RefuseLeTemoinPartiel(t *testing.T) {
	_, src := repertoireDuTemoin(t, temoinPartiel(t))
	verifierRefusEcarte(t, refuserManifesteNonFinalise(context.Background(), "ab526724", src))
}

// TestCuisson_RefuseDesMorceauxHorsManifeste : le manifeste porte ses temps forts, mais deux
// morceaux du repertoire n y sont pas decrits — le manifeste ne dit pas le film entier.
func TestCuisson_RefuseDesMorceauxHorsManifeste(t *testing.T) {
	entrees := append(temoinPartiel(t), filmcache.WriteChunk{Index: 36,
		ChunkType: filmcache.ChunkTypeTempsForts, StartMS: 681909, DurationMS: 3})
	dir, src := repertoireDuTemoin(t, entrees)
	if err := refuserManifesteNonFinalise(context.Background(), "ab526724", src); err != nil {
		t.Fatalf("manifeste finalise refuse : %v", err)
	}
	film, err := decfilm.LoadDir(dir, src.Meta())
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	err = refuserMorceauxHorsManifeste(context.Background(), "ab526724", src, film)
	verifierRefusEcarte(t, err)
	if !strings.Contains(err.Error(), "[34 35]") {
		t.Errorf("le refus ne nomme pas les morceaux hors manifeste : %q", err)
	}
}

// TestCuisson_AccepteLeFilmFinalise : LE CONTROLE NEGATIF — le meme film, repare (manifeste de 37
// entrees, comme O1 l a reecrit), passe les deux gardes. Et un film SANS manifeste n est pas juge.
func TestCuisson_AccepteLeFilmFinalise(t *testing.T) {
	entrees := append(temoinPartiel(t),
		filmcache.WriteChunk{Index: 34, ChunkType: 2, StartMS: 660116, DurationMS: 20000},
		filmcache.WriteChunk{Index: 35, ChunkType: 2, StartMS: 680117, DurationMS: 1791},
		filmcache.WriteChunk{Index: 36, ChunkType: filmcache.ChunkTypeTempsForts, StartMS: 681909, DurationMS: 3})
	dir, src := repertoireDuTemoin(t, entrees)
	ctx := context.Background()
	if err := refuserManifesteNonFinalise(ctx, "ab526724", src); err != nil {
		t.Fatalf("film finalise refuse avant chargement : %v", err)
	}
	film, err := decfilm.LoadDir(dir, src.Meta())
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	if err := refuserMorceauxHorsManifeste(ctx, "ab526724", src, film); err != nil {
		t.Fatalf("film finalise refuse apres chargement : %v", err)
	}
	if err := refuserManifesteNonFinalise(ctx, "ab526724", nil); err != nil {
		t.Errorf("film sans manifeste juge : %v", err)
	}
	if err := refuserMorceauxHorsManifeste(ctx, "ab526724", nil, film); err != nil {
		t.Errorf("film sans manifeste juge : %v", err)
	}
}
