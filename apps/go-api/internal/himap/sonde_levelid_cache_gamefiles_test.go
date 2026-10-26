//go:build gamefiles

package himap

// LA PREUVE level_id DES CARTES FORGE DONT LA SEULE VARIANTE EST AU CACHE DE DONNEES.
//
// Meme methode que `TestPreuveLevelIDCartes` (level_id lu dans le `.mvar`, unicite du dossier
// installe qui le porte en tag `levl` sur `any/levels` + `ds/levels`), autre SOURCE : le depot de
// variantes (`DepotVariantesCarte`) n'existe plus sur le poste, mais la chaine des films depose
// chaque variante rapatriee sous `<cache>/mvar/<map_id>/map.mvar` (`replayartifacts.deposerMvar`).
// La cle est donc le map_id, pas le nom de fichier du depot.
//
// Le cache n'est pas versionne : son absence fait SKIP avec sa raison, jamais un vert silencieux.
// Le dossier se lit dans PREUVE_MVAR_CACHE (le cache vit dans le checkout qui synchronise, pas
// dans un worktree), sinon en remontant depuis le paquet jusqu a `data/cache/mvar`.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
)

// preuvesLevelIDCache : cartes Forge prouvees par la variante du cache. MEME TABLE que les
// entrees correspondantes de `mapModule` (cmd/mapquant-build/main.go) et de `CartesForge`.
var preuvesLevelIDCache = []struct {
	Carte  string // nom affiche (match_registry.map_name)
	MapID  string // asset UGC, cle du cache de variantes
	Module string // dossier installe attendu
}{
	{"Serenity - Ranked", "1de0bf60-e446-4fb9-970f-d0e54fc6c74a", CanevasWetland},
	{"Interference", "654dff62-d618-496a-8914-06ab73d991e3", CanevasFrost},
	{"Vacancy - Ranked", "6a1e8432-88ae-4430-8f7d-9ffefc97cc8d", CanevasAcademy},
	{"Harvest", "ff6b8a12-6b95-4eac-8ee1-9ba4c985c2e0", CanevasBlank},
	// Seize cartes Forge de la liste d'attente du rejeu (2026-10-26). Une ligne par asset dont la
	// variante est encore publiée ; les assets retirés du stockage (404) n'ont pas de ligne.
	{"Arrival", "e0bd2b53-30d6-4a20-8664-5905ac5748d1", CanevasSpace},
	{"Arrival", "beaf8ac9-fa85-4562-8def-28d9428dbddb", CanevasSpace},
	{"Beaver Canyon", "fd17a1ca-3c67-4d6e-9149-1eb86152ea4a", CanevasWetland},
	{"Beaver Canyon - Ranked", "2c945a70-fdab-4294-a62e-7c2b76b3209a", CanevasWetland},
	{"Canopy", "beedcb81-a43a-4810-b43c-a6e4949b64a6", CanevasDesert},
	{"Canopy - Ranked", "9dc34ab3-aca5-474d-ba43-05bc9b07a3f9", CanevasDesert},
	{"Cold Storage", "78cebfc7-980a-4e8f-86a7-cf22f55f670e", CanevasBlank},
	{"Exiled", "78905252-4e2f-4cf7-840f-c0bad5a6b67f", CanevasDesert},
	{"Exiled", "9bb8d9df-ff6c-4a6e-b151-436bbb2c0907", CanevasDesert},
	{"Exiled - Ranked", "4b4ca939-513f-4d02-b119-ad2b560113e2", CanevasDesert},
	{"Formation", "2c9df7e9-89e7-430a-8615-32474d4617c1", CanevasSpace},
	{"Foundry", "66f4fe86-ba15-43d2-9270-bbd67dd22d8f", CanevasBlank},
	{"Gondolas", "c4cd9e46-3666-4d89-98ec-3b7b2c7005fb", CanevasWetland},
	{"Guardian", "1441775d-a1f1-4571-aa7f-1d1ebf6ae15c", CanevasDesert},
	{"Immolate", "47823612-9de0-4ca9-8a95-b3a6ebd7ca91", CanevasBlank},
	{"Kusini Bay", "89f3b8ad-6bbf-4652-8d67-8e5330294de4", CanevasBlank},
	{"Kusini Bay - Ranked", "d34fd775-44a4-4ba9-a1fe-dfdf1e81233b", CanevasBlank},
	{"Narrows", "944396dd-5661-4a16-b1d8-a6053f762c55", CanevasFrost},
	{"Perdition", "be8131fc-8839-4448-b65c-1fb46dd077ef", CanevasBlank},
	{"Skyblocks", "1e940e54-600e-4992-ab27-4400abb61918", CanevasBlank},
	{"Thunderbowl", "5a842f47-e4d2-4aa6-9a24-aac078a6f0f3", CanevasBlank},
	{"Timberland Evolved", "1231d3f0-2363-4d28-8047-717a069fb0e4", CanevasWetland},
}

func TestPreuveLevelIDCartesDuCache(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	cache, err := racineCacheVariantes()
	if err != nil {
		t.Skip(err)
	}
	index, nModules := indexLevlInstalle(t, root)
	t.Logf("index levl : %d level_id distincts sur %d modules", len(index), nModules)

	for _, c := range preuvesLevelIDCache {
		t.Run(c.Carte, func(t *testing.T) {
			chemin := filepath.Join(cache, c.MapID, "map.mvar")
			brut, rerr := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
			if rerr != nil {
				t.Skipf("variante absente du cache : %v", rerr)
			}
			v, perr := mapvar.Parse(brut)
			if perr != nil {
				t.Fatalf("%s : %v", c.Carte, perr)
			}
			if v.LevelID == 0 {
				t.Fatalf("%s : level_id nul — la preuve ne prouverait rien", c.Carte)
			}
			dossiers := dossiersDistincts(index[uint32(v.LevelID)])
			t.Logf("%s : level_id %d (0x%08X) -> %v | objets %d | objectifs %d",
				c.Carte, v.LevelID, uint32(v.LevelID), dossiers, len(v.Objects), len(v.Objectives()))
			if len(dossiers) != 1 {
				t.Fatalf("le level_id %d designe %d dossiers (%v) — la preuve exige l'unicite",
					v.LevelID, len(dossiers), dossiers)
			}
			if dossiers[0] != c.Module {
				t.Fatalf("le level_id %d designe %q, attendu %q", v.LevelID, dossiers[0], c.Module)
			}
		})
	}
}

// preuveMvarCacheEnv designe le dossier `mvar` du cache de donnees (chemin absolu) quand il n est
// pas sous le depot courant.
const preuveMvarCacheEnv = "PREUVE_MVAR_CACHE"

// racineCacheVariantes rend le dossier des variantes du cache : PREUVE_MVAR_CACHE, sinon
// `data/cache/mvar` en remontant depuis le paquet.
func racineCacheVariantes() (string, error) {
	if d := os.Getenv(preuveMvarCacheEnv); d != "" {
		if _, err := os.Stat(d); err != nil {
			return "", err
		}
		return d, nil
	}
	return cheminDepuisDepot("data/cache/mvar")
}
