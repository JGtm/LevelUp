// Package haloadapter est l'adaptateur de LevelUp sur la brique « API Halo » pour Halo Infinite
// (plan D3, décision D3-D ; ADR 0012 : les adaptateurs propres à un titre vivent sous
// `games/<titre>/`). La brique `internal/haloapi/infinite` PARLE aux services (transport, adresses,
// DTO JSON bruts) sans état ni politique ; ce paquet DÉCIDE :
//
//   - qui appelle, avec quels jetons : jetons, xuid, titre et langue sont lus dans le `ctx` ou dans
//     le client ICI, jamais par la brique ;
//   - par où sort la requête : un `haloapi.Doer` gardé par `netguard` par SURFACE (mode démo,
//     une ligne de journal par surface), porteur du délai d'aujourd'hui ;
//   - comment on reprend : une politique de la brique par ancien consommateur (`RetryProvider`,
//     `RetryDiscovery`, `RetryHaloClient`) ;
//   - où sont les services : les adresses de la brique construites depuis le résolveur de titre
//     (`games.DefaultEndpointResolver`, [EndpointsDepuisResolveur]) ;
//   - ce que la réponse veut dire pour LevelUp : conversions DTO -> `domain`, absences 404/410
//     rendues en résultat vide là où l'appelant l'attend, caches (emblèmes, confidentialité),
//     définitions d'actifs (`assets`), métadonnées des médailles, MMR connu ou non.
//
// [Provider] et [Client] gardent les méthodes des deux clients d'avant la brique : leurs appelants
// n'ont changé que leur import et leur constructeur. Les films (cache disque, décompression,
// finalisation) sont à `sync/filmfetch`, que [Client] délègue.
package haloadapter
