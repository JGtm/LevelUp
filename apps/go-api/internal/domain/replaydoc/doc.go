// Package replaydoc porte le DOCUMENT SERVI du rejeu 2D : la forme de fil que l'API
// publie, et rien d'autre.
//
// POURQUOI CE PAQUET EXISTE. Jusqu'au 2026-09-05, le document servi ETAIT le document
// stocke : `internal/analysis/replay.ReplayDocument` — le format du fichier
// `data/cache/replays/{titre}/{match}.json` — etait rendu tel quel par le handler, donc
// derive en 99 schemas d'`api/openapi.yaml` puis en autant de types TypeScript. Un seul
// objet portait trois roles : format de cuisson sur disque, modele de calcul des calques,
// et contrat public. La consequence n'etait pas theorique — `SchemaVersion` a ete
// incremente 43 fois en cinq semaines (v7.3.0 -> 2026-09-05), et chacune de ces montees
// regenerait le contrat public pour un champ que le client ne lisait pas encore ;
// inversement aucun champ ne pouvait etre renomme pour le client sans invalider le parc
// d'artefacts deja cuits.
//
// CE QUE LA SEPARATION CHANGE. Les types de ce paquet decrivent ce que le client recoit.
// Ceux d'`internal/analysis/replay` decrivent ce que la cuisson ecrit. Le convertisseur
// (`internal/service/replayview`) est la seule arete entre les deux, et le test de parite
// qui l'accompagne exige une decision ecrite pour chaque champ stocke : copie, transforme,
// ou explicitement non servi. Ajouter un calque a la cuisson ne touche donc plus le
// contrat public tant que ce paquet n'a pas bouge.
//
// CE QUE LA SEPARATION NE CHANGE PAS (2026-09-05). La forme de fil est identique champ
// pour champ a celle d'avant : memes noms de types (Huma en derive les noms de schemas
// OpenAPI), memes tags JSON, memes `omitempty`. `api/openapi.yaml` et
// `apps/web/src/lib/api/generated.ts` sont inchanges — c'est le gate du lot.
//
// AUCUN IMPORT D'`internal/analysis/replay` ICI, jamais : ce paquet est une feuille de
// `domain/`, et c'est ce qui garantit que le contrat ne suive pas le format de stockage
// par simple alias. Les chroniques de schema (pourquoi tel champ est ne, ce que la mesure
// a refuse d'y mettre) restent du cote stocke, ou elles decrivent le producteur.
//
// PAS DE NUMERO DE VERSION DANS CE PAQUET, et c'est une decision. La premiere version en
// portait un (`ContractVersion = 39`) ; la revue adversariale du 2026-09-05 a montre qu'on
// pouvait lui donner n'importe quelle valeur positive sans qu'aucun gate ne bouge — aucun
// code de production ne le lisait, et son seul test affirmait qu'il etait positif. C'etait
// une promesse de versionnage que rien n'observait, tenue verte par un test tautologique.
// Retire (regle 7, zero code mort).
//
// CE QUI TIENT LA FORME SERVIE, a la place :
//   - le golden `api/openapi.yaml`, regenere depuis ces types et compare octet pour octet
//     (`TestOpenAPIYAMLIsUpToDate`) — tout changement de forme y est visible ;
//   - `TestReplayDocumentFieldCountIsFrozen` (`contracttest/`), qui confronte le nombre de
//     champs de `ReplayDocument` a une constante ECRITE et a sa chronique datee.
//
// CE QUI VERSIONNE, a la place : le champ `schemaVersion` du corps, qui porte la version de
// l'ARTEFACT LU (`analysis/replay.SchemaVersion` au moment de la cuisson). C'est elle, et
// elle seule, qui dit au parc « a re-cuire » ; la projection la recopie telle quelle et ne
// la remplace jamais (verrouille par `replayview/parity_test.go`).
package replaydoc
