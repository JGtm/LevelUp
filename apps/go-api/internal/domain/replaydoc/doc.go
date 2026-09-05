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
package replaydoc

// ContractVersion est la version de la FORME SERVIE. Elle ne bouge que si le contrat
// public change (champ retire ou renomme, type change) — jamais parce que la cuisson a
// ajoute un calque.
//
// C'est la version STOCKEE (`analysis/replay.SchemaVersion`, publiee dans le champ
// `schemaVersion` du corps) qui gouverne la RE-CUISSON du parc d'artefacts : un artefact
// dont la version stockee est inferieure a celle du producteur se lit « a re-cuire ».
// Les deux nombres partent de la meme valeur au 2026-09-05 (39, l'etat du contrat au
// moment de la separation) et sont destines a diverger : la version stockee monte a
// chaque calque, celle-ci reste stable tant que le client n'a rien a changer.
const ContractVersion = 39
