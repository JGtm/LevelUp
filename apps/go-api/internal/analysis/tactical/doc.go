// Package tactical rasterise des positions du monde sur une grille reguliere et en tire
// les lectures de placement de l'onglet Tactique : ou un joueur passe son temps, ou il
// meurt, ou il tue, ou il gagne.
//
// PUR : aucune I/O, aucun SQL, aucun reseau, aucun etat global. L'entree est une liste de
// `domain.PositionSample` (matchID, x, y en metres monde) que l'appelant projette depuis
// ce qu'il a — les positions mesurees de `kill_positions`, ou les points des pistes d'un
// artefact de rejeu. Ce paquet n'importe NI `analysis/replay` NI `platform/duckdb` : une
// lecture tactique doit rester calculable sans artefact.
//
// ART ANTERIEUR — cmd/mappos-build (2026-08-30). La mecanique de la grille vient de la, et
// avec elle deux mesures qui ne se devinent pas :
//
//   - pas de 0,5 m : les positions brutes sont massivement redondantes (366 768 points sur
//     13 matchs de Dredge pour 1 008 cellules d'un metre distinctes) ;
//   - plancher en MATCHS DISTINCTS, pas en occurrences : sans filtre, le nuage des
//     positions s'etend a 268 m du centre de l'arene ; en exigeant deux matchs distincts
//     par cellule il retombe a 27 m, et a trois a 19,4 m. Un joueur immobile gonfle une
//     cellule sans rien prouver ; deux matchs differents sont deux observations
//     independantes.
//
// Ce paquet REPREND cette mecanique plutot qu'il ne l'importe : `cmd/mappos-build` est un
// `main`, il ecrit un catalogue sur disque, et il ne connait qu'une carte a la fois.
//
// ANCRAGE DES CELLULES. L'adresse d'une cellule est ancree sur l'ORIGINE DU MONDE, jamais
// sur les bornes de la lecture courante. C'est ce qui rend deux rasters de matchs
// differents sommables sans re-projection (le stockage d'un raster par match a la cuisson
// en depend). Les bornes d'une lecture agregee sont l'UNION des bornes des rasters sommes,
// et servent au cadrage, pas a l'adressage.
package tactical
