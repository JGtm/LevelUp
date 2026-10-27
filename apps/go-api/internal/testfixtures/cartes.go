package testfixtures

import "path/filepath"

// CartesDir retourne le dossier des fixtures de cartes versionnees :
// internal/testfixtures/testdata/cartes/.
//
// Contenu (les seuls fichiers de carte que des tests lisent) :
//   - callout_zones_ridgeline_clipped.json : dump decoupe de Ridgeline (oracle de mapdecoupe,
//     donnees de mapcatalog) ;
//   - carte_validee_v1.png : la carte de reference des bancs himap ;
//   - mapvar/ : trois variantes .mvar (cliffhanger_map, cliffhanger_ridgeline,
//     vagabond_fo08_wetland), temoins du decodeur mapvar et des outils mapobj-build.
//
// Les tests ne lisent jamais .ai/ : ce dossier est un espace de travail, pas une source de
// fixtures (garde-rail archlint/pas_de_lecture_sous_ai_test.go).
func CartesDir() string {
	return filepath.Join(RepoRoot(), "apps", "go-api", "internal", "testfixtures", "testdata", "cartes")
}

// Carte retourne le chemin absolu d'une fixture de carte sous CartesDir() ; elems est le
// chemin relatif, par exemple Carte("mapvar", "cliffhanger_map.mvar").
func Carte(elems ...string) string {
	return filepath.Join(append([]string{CartesDir()}, elems...)...)
}
