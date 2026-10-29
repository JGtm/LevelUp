// Package infinite est le client des services Halo Infinite de la brique « API Halo » : transport
// HTTP à politique de reprise injectée, adresses des services, un fichier par service, DTO JSON
// BRUTS (structs miroir du JSON reçu), octets COMPRESSÉS des films.
//
// Contrat :
//
//   - SANS ÉTAT : aucune variable de paquet mutable, aucun cache. Un [Client] est une valeur
//     immuable ; ses `With*` rendent une copie. Le cache des emblèmes, celui de la confidentialité,
//     le cache local des films et le singleton du fournisseur appartiennent à l'appelant.
//   - SANS POLITIQUE : la brique ne choisit ni le compte, ni le titre, ni la langue, ni le moment.
//     Jetons ([Auth]), adresses ([Endpoints]), politique de reprise ([Retry]), limiteur ([Limiter])
//     et en-tête `Accept-Language` lui sont PASSÉS. Elle ne juge pas les valeurs (un MMR nul, un
//     401 « joueur tiers », une absence 404 sont rendus tels quels, typés : l'appelant décide).
//   - AUCUN CLIENT HTTP CONSTRUIT : chaque requête part par le [haloapi.Doer] reçu, qui porte la
//     garde `netguard` et le délai. Une erreur [haloapi.ErrDefinitif] arrête toute reprise. Si ce
//     `Doer` porte une [haloapi.Garde], la brique la consulte AVANT le limiteur et avant toute
//     attente, au début de chaque appel (requête et blob) : une sortie refusée ne consomme aucun
//     jeton du limiteur et ne déclenche aucune reprise.
//   - OCTETS DE FILM COMPRESSÉS : la brique télécharge les blobs du CDN et contrôle leur taille
//     annoncée ; elle ne décompresse pas et ne lit aucun octet de film (la décompression et la
//     finalisation sont à `sync/filmfetch`, ratchet `no_raw_film_bytes_outside_source_test.go`).
//   - IMPORTS : bibliothèque standard, `internal/haloapi`, `golang.org/x/sync/errgroup`
//     (`archlint`, table des briques internes).
//   - JOURNAL : `slog` avec le `ctx` reçu ; aucun jeton ni en-tête d'autorisation n'est journalisé
//     ([Auth.String] et [Auth.LogValue] masquent les jetons).
//
// La conversion des DTO vers les types `domain`, les caches, les compteurs expvar et la lecture du
// contexte (jetons, xuid, titre, langue) vivent dans LevelUp (adaptateur `haloadapter`,
// `sync/filmfetch`).
package infinite
