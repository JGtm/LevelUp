// Package filmarchive — L'ARCHIVE DES FILMS SUR DISQUE, COTE ECRITURE : ecrire un film recu,
// recompresser, deplacer ou retirer l'archive.
//
// Le decodeur (`libs/film`) LIT un film depuis un dossier (`filmcache` : manifeste, chunks, formes
// brut, zlib et zstd) et ne decide pas OU on stocke (ADR 0034). Ce paquet porte ce qui ecrit : il
// vient du paquet `filmcache` de la brique, dont il garde la disposition (les chemins se composent
// par `filmcache.ManifestPath`, `ChunkDir`, `CheminDuChunk`, jamais a la main) et le format (un
// chunk s'ecrit et se recompresse par les fonctions zstd de `filmcache`).
//
//   - [Write] persiste un film FINALISE a la reception : chunks d'abord, manifeste en dernier
//     (le marqueur de commit), chaque fichier ecrit atomiquement.
//   - [Recompresser] passe en zstd-3 les chunks bruts et zlib d'un film ; [Deplacer] range
//     l'archive dans un autre dossier ; [Supprimer] retire un film. Ces trois gestes sont des
//     commandes de l'utilisateur, jamais automatiques (`archlint/no_auto_film_recompress_test.go`).
package filmarchive
