package testfixtures

import "path/filepath"

// Dossiers de fixtures binaires POSSEDES PAR LA BRIQUE QUI LES LIT. Chaque fichier n'existe
// qu'en une copie, versionnee a cote du lecteur (son `testdata/`) ; LevelUp la lit d'ici, par
// un chemin depuis la racine du depot, jamais par un `../` entre paquets. Ce sont des actifs
// proprietaires 343/Microsoft : ne pas en dupliquer, ne pas en ajouter.
//
// Chemins relatifs a la racine du depot, separateurs slash. Quand une brique change de place
// (D1, lot 3 : `libs/gamefiles/...`), c'est la seule ligne a toucher.
const (
	// MapvarFixturesDir : la brique mapvar possede ses trois `.mvar` (cliffhanger_map,
	// cliffhanger_ridgeline, vagabond_fo08_wetland) ; LevelUp les lit ici. Le dossier sert aussi
	// de cache local des variantes re-telechargeables (cf. .gitignore, seules trois sont suivies).
	MapvarFixturesDir = "apps/go-api/internal/games/halo_infinite/film/replay/mapvar/testdata"

	// NavmeshFixturesDir : la brique hinavmesh possede les deux `<assetID>.navmesh.blob`.
	NavmeshFixturesDir = "apps/go-api/internal/hinavmesh/testdata"

	// CarteValideeFixturesDir : la brique himap possede `carte_validee_v1.png`, la carte de
	// reference de ses bancs de rendu.
	CarteValideeFixturesDir = "apps/go-api/internal/himap/testdata"
)

// CartesDir retourne le dossier des fixtures de cartes propres a LevelUp :
// internal/testfixtures/testdata/cartes/.
//
// Contenu : callout_zones_ridgeline_clipped.json, dump decoupe de Ridgeline (oracle de
// mapdecoupe, donnees de mapcatalog). Les fixtures binaires de cartes ne sont PAS ici : elles
// vivent dans la brique qui les lit (voir MapvarFixturesDir, NavmeshFixturesDir,
// CarteValideeFixturesDir) et Carte les y resout.
//
// Les tests ne lisent jamais .ai/ : ce dossier est un espace de travail, pas une source de
// fixtures (garde-rail archlint/pas_de_lecture_sous_ai_test.go).
func CartesDir() string {
	return filepath.Join(RepoRoot(), "apps", "go-api", "internal", "testfixtures", "testdata", "cartes")
}

// Carte retourne le chemin absolu d'une fixture de carte ; elems est le chemin relatif, par
// exemple Carte("mapvar", "cliffhanger_map.mvar"). Le sous-dossier mapvar/ est possede par la
// brique mapvar : il se resout vers MapvarFixturesDir ; tout le reste vit sous CartesDir().
func Carte(elems ...string) string {
	if len(elems) > 0 && elems[0] == "mapvar" {
		return filepath.Join(append([]string{repoPath(MapvarFixturesDir)}, elems[1:]...)...)
	}
	return filepath.Join(append([]string{CartesDir()}, elems...)...)
}

// CarteValidee retourne le chemin absolu de `carte_validee_v1.png`, dans le `testdata/` de la
// brique himap qui la possede.
func CarteValidee() string {
	return filepath.Join(repoPath(CarteValideeFixturesDir), nomCarteValidee)
}

// Navmesh retourne le chemin absolu du blob de navigation d'une carte Forge
// (`<assetID>.navmesh.blob`), dans le `testdata/` de la brique hinavmesh.
func Navmesh(assetID string) string {
	return filepath.Join(repoPath(NavmeshFixturesDir), assetID+".navmesh.blob")
}

const nomCarteValidee = "carte_validee_v1.png"

// repoPath resout un chemin slash relatif a la racine du depot en chemin absolu du poste.
func repoPath(dirSlash string) string {
	return filepath.Join(RepoRoot(), filepath.FromSlash(dirSlash))
}
