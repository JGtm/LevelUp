package replaybuild

// filmfacts_fil_des_morts_test.go — LA BRANCHE DES FAITS REND LA MEME LECTURE DU FIL DES MORTS QUE
// LA BRANCHE DU FILM (lot M8 des retours rejeu, 2026-09-24).
//
// Les deux consommateurs de cet etage (`identifiedEvents`, `killRefs`) branchent sur l ERREUR du
// fil des morts. La branche « relire » la jetait : un fil vide ou illisible y arrivait sans erreur,
// les actions d objectif s identifiaient sur un pont sans mort et les frags sous effet actif se
// disaient lus — la ou la branche « decoder » du MEME film refuse les deux. Les faits portent
// desormais le verdict et sa cause, et [entreesDesFaits] reconstruit l erreur.
//
// LES OCTETS sont ceux de la bobine v40 du depot ; le temoin ILLISIBLE est la forme du manifeste
// partiel d `ab526724` (`temoinPartiel`, copie a l octet sous `film/filmcache/testdata/`). Les
// verdicts sont ecrits par leur VALEUR PUBLIEE (`coverage.bridge.deathsFeed`) : les constantes
// vivent dans la couche de publication, qui les epingle (`TestLectureDuFilDesMorts_TroisIssues`).

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// bobineV40 : les trois morceaux BRUTS de la bobine v40, dans l ordre des numeros.
func bobineV40(t *testing.T) [][]byte {
	t.Helper()
	dir := filepath.Join("..", "games", "halo_infinite", "film", "internal", "facts", "killsource",
		"testdata", "minibobine_e5adf7b2")
	out := make([][]byte, 0, 3)
	for k := 0; k < 3; k++ {
		b, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", k)))
		if err != nil {
			t.Fatalf("bobine v40 : %v", err)
		}
		out = append(out, b)
	}
	return out
}

// filmDuRepertoire range les morceaux dans un repertoire et le charge par la porte de PRODUCTION
// (`decfilm.LoadDir`, celle de `chargerFilm`), avec l index fourni.
func filmDuRepertoire(t *testing.T, morceaux map[int][]byte, meta []decfilm.ChunkMeta) *decfilm.Film {
	t.Helper()
	dir := t.TempDir()
	for n, b := range morceaux {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", n)), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	film, err := decfilm.LoadDir(dir, meta)
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	return film
}

// TestLaBrancheDesFaitsRendLaMemeLectureDuFilDesMorts : fil LU, VIDE et ILLISIBLE.
func TestLaBrancheDesFaitsRendLaMemeLectureDuFilDesMorts(t *testing.T) {
	o := bobineV40(t)
	typeTempsForts := []decfilm.ChunkMeta{
		{Index: 0, ChunkType: 1}, {Index: 1, ChunkType: 2}, {Index: 2, ChunkType: filmcache.ChunkTypeTempsForts},
	}
	partiel := map[int][]byte{}
	var metaPartiel []decfilm.ChunkMeta
	for _, c := range temoinPartiel(t) {
		partiel[c.Index] = o[1] // de la replication, partout sauf au registre
		metaPartiel = append(metaPartiel, decfilm.ChunkMeta{Index: c.Index, ChunkType: c.ChunkType,
			StartMS: c.StartMS})
	}
	partiel[0] = o[0]
	for _, cas := range []struct {
		nom     string
		film    *decfilm.Film
		verdict string
	}{
		{"temps forts lus, des morts", filmDuRepertoire(t, map[int][]byte{0: o[0], 1: o[1], 2: o[2]},
			typeTempsForts), "read"},
		{"temps forts lus, aucune mort", filmDuRepertoire(t,
			map[int][]byte{0: o[0], 1: o[1], 2: make([]byte, 64)}, typeTempsForts), "empty"},
		{"temoin partiel d ab526724", filmDuRepertoire(t, partiel, metaPartiel), "unreadable"},
	} {
		depuisLeFilm := lireMorts(cas.film)
		if (depuisLeFilm.err == nil) != (cas.verdict == "read") {
			t.Fatalf("%s : lecture du film err=%v pour un verdict %q — le temoin ne mesure plus rien",
				cas.nom, depuisLeFilm.err, cas.verdict)
		}
		// LES FAITS QUE LE BALAYAGE DE CE FILM PERSISTE : le fil, son verdict et sa cause.
		var f replay.FilmFactsFile
		f.Facts.Deaths = depuisLeFilm.list
		f.Facts.DeathsFeed.Verdict = cas.verdict
		if depuisLeFilm.err != nil {
			f.Facts.DeathsFeed.Cause = depuisLeFilm.err.Error()
		}
		depuisLesFaits := entreesDesFaits(&f).deaths
		if (depuisLesFaits.err == nil) != (depuisLeFilm.err == nil) {
			t.Errorf("%s : erreur du fil des morts, branche des faits %v, branche du film %v : les "+
				"actions d objectif et les frags ne prennent pas la meme branche", cas.nom,
				depuisLesFaits.err, depuisLeFilm.err)
			continue
		}
		if depuisLeFilm.err != nil && depuisLesFaits.err.Error() != depuisLeFilm.err.Error() {
			t.Errorf("%s : cause %q, le film journalise %q", cas.nom, depuisLesFaits.err, depuisLeFilm.err)
		}
		if !reflect.DeepEqual(depuisLesFaits.list, depuisLeFilm.list) {
			t.Errorf("%s : %d mort(s) relues, %d lues dans le film", cas.nom,
				len(depuisLesFaits.list), len(depuisLeFilm.list))
		}
	}
}
