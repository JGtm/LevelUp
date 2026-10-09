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
