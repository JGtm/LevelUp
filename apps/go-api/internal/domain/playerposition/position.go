// Package playerposition PORTE LA POSITION D UN JOUEUR DANS UN MATCH, ET RIEN D AUTRE.
//
// # POURQUOI CE TYPE VIT DANS `domain/` (lot 2.5.e, 2026-09-16)
//
// Il vivait dans `internal/analysis/positions`, avec le DECODEUR qui le produit. Or ce decodeur
// lit des octets de film : il descend dans la couche `grammar` du decodeur (decision V15 (4)),
// et ses couches passent sous `film/internal/` a la fin du lot. Le type, lui, est lu par des
// paquets qui n ont rien a savoir d un titre — les ports (`port.PlayerPositionsRepository`), le
// service de vue de match, la couche DuckDB, les corps HTTP. Les faire importer le decodeur
// serait l inverse de la frontiere que le lot pose.
//
// C est le chemin du lot 2.5.h pour `HighlightEvent` et du lot 2.5.f pour les issues d usage
// d equipement : LE TYPE REMONTE EN `domain/`, PUIS LE LECTEUR DESCEND. La maison est une
// FEUILLE — elle n importe rien du depot — voisine de `domain/highlightevent`,
// `domain/equipmentusage` et `domain/replaydoc`.
//
// Ce qui N Y ENTRE PAS, et la mesure dit pourquoi : `ChunkInput` (l entree du decodeur : des
// octets de chunk deja decompresses, donc du vocabulaire de film) et `DecodeKeyframePositions`
// (le balayage lui-meme). Ils restent avec la grammaire.
package playerposition

// TeamUnknown marque une position dont l'équipe n'a pas pu être inférée.
const TeamUnknown = -1

// PlayerPosition est une position full-state décodée d'une keyframe.
//
// Team vaut -1 (TeamUnknown) tant qu'aucun clustering spatial net ne permet de
// l'attribuer ; 0 ou 1 sinon (best-effort, non garanti).
type PlayerPosition struct {
	TimeMS  int
	X, Y, Z float32
	Team    int
}
