// Package haloapi est le paquet parent de la brique « API Halo » : le code qui PARLE aux
// services Halo, sans état ni politique. Il ne porte que ce que ses sous-paquets partagent : le
// [Doer] reçu de LevelUp, la [Garde] optionnelle que ce Doer peut porter (consultée AVANT tout
// limiteur et toute attente, pour qu'une sortie refusée ne consomme rien) et l'erreur
// [ErrDefinitif] qui arrête une reprise ([NonRetryable] la pose sur une erreur existante).
//
// Ses sous-paquets sont `xblauth` (chaîne d'authentification Xbox Live, XSTS, Spartan) et
// `infinite` (services Halo Infinite : adresses, transport, DTO JSON bruts, octets des films).
// Contrat : un paquet de la brique n'importe RIEN de LevelUp hors de `internal/haloapi/...`
// (`archlint`, table des briques internes) ; ce dont il a besoin (client HTTP, adresses, jetons,
// langue) lui est PASSÉ, et il ne construit jamais de client HTTP (garde `netguard` par `Doer`
// injecté, ratchet `haloapi_sans_client_http_test.go`). Ce qui décide (qui appelle, avec le jeton
// de qui, où ranger, ce que cela veut dire) reste dans LevelUp.
package haloapi
