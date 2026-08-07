package objectivescore

// filmcache_test.go — l'ACCÈS AU CACHE FILM RÉEL, partagé par les tests de vérité terrain
// de ce paquet.
//
// POURQUOI CE FICHIER EXISTE. Le seul test de ce paquet qui touchait des octets réels
// (`TestDecodeStrongholds_CacheBacked_7344d24f`) pointait une racine de cache EN DUR sur le
// poste d'origine — `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache`,
// dont aucun des trois derniers niveaux n'existe plus. Il se skippait donc sur 100 % des
// machines, CI comprise : AUCUN octet de film réel n'était exécuté contre ce décodeur.
// (Audit du 2026-08-06, constat P1 « le seul test du score de mode sur film réel est éteint
// par un chemin absolu mort ».)
//
// LA CONVENTION DU DÉPÔT est la variable d'environnement, jamais le chemin en dur :
// `FILM_CACHE_ROOT` (objectiveevents), `KILLSOURCE_FIXTURES` (killsource), `REPLAY_FILM_DIR`
// (replay). Ce fichier applique la première, qui a exactement la même sémantique ici : la
// racine du cache, celle qui contient `film_manifests/` et `film_chunks/`.
//
// DEUX RÉGIMES, ET LA DISTINCTION EST LE CŒUR DU CORRECTIF :
//
//	VARIABLE ABSENTE            skip explicite. Légitime : les films pèsent des dizaines de
//	                            Mo et ne sont pas versionnés, la CI ne les a pas.
//	VARIABLE PRÉSENTE MAIS
//	FAUSSE (racine ou film
//	introuvable)                ÉCHEC, jamais un skip. C'est précisément le régime dans
//	                            lequel l'ancien test vivait — il tenait une racine morte
//	                            pour une absence de cache et se taisait. Une faute de frappe
//	                            dans la variable doit être bruyante.
//
// CE QUI RESTE VRAI MALGRÉ TOUT : un test conditionné à une variable ne tourne pas en CI.
// Le filet permanent de ce paquet est ailleurs, et il est inconditionnel — voir
// `minibobine_test.go`.

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// filmCacheEnv : la racine du cache film. Même nom et même sémantique que dans
// `internal/analysis/objectiveevents` — deux noms pour la même chose seraient une
// divergence en attente.
const filmCacheEnv = "FILM_CACHE_ROOT"

// filmManifestsDir / filmChunksDir : la disposition du cache sous la racine.
const (
	filmManifestsDir = "film_manifests"
	filmChunksDir    = "film_chunks"
)

// testManifest : la part du manifest de film dont ce paquet a besoin — l'index du chunk,
// son type (seul le type 2 porte le score) et son instant de départ.
type testManifest struct {
	Chunks []struct {
		Index     int `json:"index"`
		ChunkType int `json:"chunk_type"`
		StartMS   int `json:"start_ms"`
	} `json:"chunks"`
}

// racineCacheFilm renvoie la racine du cache, ou SKIPPE si la variable n'est pas définie.
//
// Si elle EST définie mais désigne un répertoire qui n'existe pas, ou qui ne ressemble pas à
// un cache film, c'est un échec : à ce moment-là quelqu'un a voulu jouer ces tests et s'est
// trompé de chemin, et lui rendre un « ok » serait le mensonge que ce fichier corrige.
func racineCacheFilm(t *testing.T) string {
	t.Helper()
	root := os.Getenv(filmCacheEnv)
	if root == "" {
		t.Skipf("%s non défini : les tests de vérité terrain de objectivescore sont ignorés. "+
			"Les films ne sont pas versionnés (des dizaines de Mo par film) ; pour les jouer : "+
			"%s=<repo>/data/cache go test ./internal/analysis/objectivescore/. "+
			"CE QUI N'EST DONC PAS VÉRIFIÉ ICI : les positions de bit du décodeur contre des "+
			"octets de film. Le garde qui, lui, tourne toujours est TestGoldenMiniBobineObjectifs "+
			"(minibobine_test.go), sur une bobine versionnée", filmCacheEnv, filmCacheEnv)
	}
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		t.Fatalf("%s=%q ne désigne aucun répertoire (%v) — variable définie mais fausse : "+
			"c'est une erreur de configuration, pas une raison d'ignorer le test", filmCacheEnv, root, err)
	}
	manifests := filepath.Join(root, filmManifestsDir)
	if st, err := os.Stat(manifests); err != nil || !st.IsDir() {
		t.Fatalf("%s=%q ne contient pas %s/ — la racine attendue est celle du cache "+
			"(<repo>/data/cache), pas un répertoire de films", filmCacheEnv, root, filmManifestsDir)
	}
	return root
}

// chargerChunksFilm lit le manifest d'un film et rend TOUS ses chunks décompressés.
//
// Tous, et pas seulement les type-2 : le filtrage par type appartient au décodeur
// (`anchoredPayload`), et un harnais de test qui pré-filtre selon la règle qu'il teste
// est le défaut même que ce lot corrige.
func chargerChunksFilm(t *testing.T, root, short string) []ChunkInput {
	t.Helper()
	mfPath := filepath.Join(root, filmManifestsDir, short+".json")
	raw, err := os.ReadFile(mfPath) //nolint:gosec // chemin construit depuis une racine de test
	if err != nil {
		t.Fatalf("manifest du film %s illisible (%s) : %v — le film est absent de ce cache ; "+
			"choisir une racine %s qui le contient, ou retirer ce film du corpus de test",
			short, mfPath, err, filmCacheEnv)
	}
	var mf testManifest
	if err := json.Unmarshal(raw, &mf); err != nil {
		t.Fatalf("manifest du film %s illisible en JSON : %v", short, err)
	}
	var chunks []ChunkInput
	for _, c := range mf.Chunks {
		p := filepath.Join(root, filmChunksDir, short, nomChunk(c.Index))
		brut, err := os.ReadFile(p) //nolint:gosec // chemin construit depuis une racine de test
		if err != nil {
			continue // chunk non téléchargé : le cache est partiel par nature
		}
		chunks = append(chunks, ChunkInput{
			Data: decompresser(brut), StartMS: c.StartMS, ChunkType: c.ChunkType,
		})
	}
	if len(chunks) == 0 {
		t.Fatalf("aucun chunk lisible pour le film %s sous %s/%s — manifest présent mais "+
			"octets absents : cache incomplet, pas un motif de skip", short, root, filmChunksDir)
	}
	return chunks
}

// nomChunk : le nom de fichier d'un chunk dans le cache.
func nomChunk(i int) string { return fmt.Sprintf("chunk_%02d.bin", i) }

// decompresser : zlib si le magic 0x78 est là, sinon les octets tels quels — miroir des
// décodeurs film (le cache stocke les deux formes selon l'âge du téléchargement).
func decompresser(brut []byte) []byte {
	if len(brut) >= 2 && brut[0] == 0x78 {
		if z, e := zlib.NewReader(bytes.NewReader(brut)); e == nil {
			defer func() { _ = z.Close() }()
			if d, e2 := io.ReadAll(z); e2 == nil {
				return d
			}
		}
	}
	return brut
}
