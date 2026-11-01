// Package xboxlive est le client des succès Xbox Live (achievements v2) de la brique « API Halo » :
// adresse de l'hôte, un GET par page, DTO JSON BRUTS (structs miroir du JSON reçu).
//
// Recette du protocole :
//
//   - `GET {hôte}/users/xuid({xuid})/achievements?skipItems=N&titleId=T` ; le `xuid` est posé tel
//     quel dans le chemin (ni échappé ni encodé), `titleId` et `skipItems` ne sont posés que s'ils
//     sont non nuls, les clés sortent triées par `url.Values.Encode` ;
//   - `Authorization` = la valeur COMPLÈTE de l'en-tête (`XBL3.0 x=<hash>;<jeton>`, voir
//     `xblauth.XSTSResult.AuthHeader`), `x-xbl-contract-version: 2` (sans lui l'API rend le schéma
//     Xbox 360 v1), `Accept-Language` (posé même vide) et `Accept: application/json` ;
//   - PAGINATION : la réponse porte `pagingInfo.continuationToken` ; tant qu'il n'est pas vide,
//     l'appelant redemande la page suivante avec `skipItems` augmenté du nombre d'éléments déjà reçus.
//     [Client.GetSuccesPage] ne rend qu'UNE page : la boucle est à l'appelant.
//
// Contrat :
//
//   - SANS ÉTAT : aucune variable de paquet mutable, aucun cache. Un [Client] est une valeur
//     immuable ; le jeton ([Auth]) est passé À CHAQUE appel.
//   - SANS POLITIQUE : l'adresse ([Endpoints]) est PASSÉE ; aucune reprise (un GET par page, une page
//     en erreur est rendue telle quelle) ; le délai est celui du [haloapi.Doer] reçu. Les statuts
//     sont rendus typés ([HTTPError], corps entier compris) ; l'appelant décide de ce qu'ils veulent
//     dire.
//   - AUCUN CLIENT HTTP CONSTRUIT : la requête part par le [haloapi.Doer] reçu, qui porte la garde
//     de LevelUp ; une erreur [haloapi.ErrDefinitif] est rendue telle quelle.
//   - IMPORTS : bibliothèque standard et `internal/haloapi` (`archlint`, table des briques internes).
//
// L'interprétation d'un succès (gamerscore, icône, date de déverrouillage, progression), la boucle de
// pagination, les journaux, le jeton et le délai du client HTTP vivent dans LevelUp (`sync`).
package xboxlive
