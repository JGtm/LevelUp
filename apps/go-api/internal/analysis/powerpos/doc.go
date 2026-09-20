// Package powerpos derive les POSITIONS DE FORCE d'une carte : les lieux qu'une equipe
// cherche a tenir, mesures sur le corpus de matchs plutot que declares a la main.
//
// PUR : aucune I/O, aucun SQL, aucun reseau, aucun etat global. L'entree est une liste de
// `KillSample` (une elimination, les deux extremites en metres monde) et de
// `PresenceSample` (un segment d'occupation par une equipe, avec l'issue du match). C'est
// l'appelant qui projette ce qu'il a — les lignes de `kill_positions_latest`, les points
// des pistes d'un artefact de rejeu — vers ces deux types. Ce paquet n'importe NI
// `games/halo_infinite/film/replay` NI `platform/duckdb` : la derivation doit rester
// calculable sans artefact et sans base.
//
// # CE QU'IL REPREND, ET CE QU'IL N'INVENTE PAS
//
// L'UNITE D'ANALYSE EST LA CELLULE DE `internal/analysis/tactical` — meme pas de 0,5 m,
// meme ancrage sur l'ORIGINE DU MONDE (D1 du plan des positions de force). Ce paquet
// IMPORTE `tactical.Grille` et `tactical.Cellule` au lieu de les redefinir : deux lectures
// de placement du depot doivent nommer la meme cellule pareil, sinon un calque derive ici
// ne se superposerait pas a la chaleur tactique de la meme carte. Le plancher de rarete se
// compte donc aussi en MATCHS DISTINCTS, et pour la meme raison que la-bas (mesure de
// cmd/mappos-build du 2026-08-30 : un joueur immobile gonfle une cellule sans rien prouver,
// deux matchs differents sont deux observations independantes).
//
// # CE QU'UNE CELLULE ACCUMULE, ET POURQUOI CES SIGNAUX-LA
//
//   - `KillsDepuis` : eliminations dont le TUEUR etait dans la cellule. C'est le signal de
//     base — on tue depuis une position de force.
//   - `MortsDedans` : eliminations dont la VICTIME etait dans la cellule. Seul, `KillsDepuis`
//     confond une position de force avec un carrefour tres frequente ; le rapport des deux
//     separe « j'y gagne mes duels » de « il s'y passe beaucoup de choses ».
//   - `PorteeMedianeM` et `DeniveleMedianM` : la distance et l'ecart d'altitude tueur-victime
//     des kills partis de la cellule. Une position de force tient la hauteur et de longues
//     lignes de vue ; ces deux mesures le disent sans avoir a lire la geometrie.
//   - `MatchsKills` / `MatchsPresence` : le plancher de rarete, compte SEPAREMENT par source.
//     Les deux sources n'ont pas la meme densite (une carte a ~80 matchs de kills pour 1 a 5
//     artefacts de rejeu) : un plancher unique sur leur union laisserait passer une cellule
//     vue dans un seul match de kills au motif qu'une piste l'a traversee.
//   - `OccupationGagnantsMS` / `OccupationPerdantsMS` : le temps passe dans la cellule par
//     l'equipe qui a GAGNE le match et par celle qui a perdu. C'est le seul signal qui dise
//     « tenir », et non « tirer ».
//
// # UN KILL SANS SES DEUX EXTREMITES EST ECARTE, PAS ARBITRE
//
// Mesure du 2026-09-20 sur le corpus Halo Infinite (138 382 lignes de `kill_positions_latest`) :
// 6,5 % n'ont pas de position de tueur et 2,9 % pas de position de victime. Une mort sans
// tueur n'est pas un duel — c'est une chute ou un suicide — et la compter en `MortsDedans`
// peindrait les fosses en positions faibles alors que personne ne les tient. L'echantillon
// est donc ECARTE en entier et COMPTE (`KillsIgnores`), jamais moitie pris moitie jete.
//
// # CE QUE CE PAQUET NE FAIT PAS
//
// Il ne NOMME pas les positions (c'est le role des zones nommees, par recouvrement, chez
// l'appelant), il ne lit aucun catalogue, et il ne decide pas du titre : une position de
// force est un polygone monde, et le monde est le meme pour tous les titres qui savent dire
// ou l'on tue.
package powerpos
