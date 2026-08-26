# Registre des cartes — revue carte par carte (v7.5)

> SOURCE DE VERITE du chantier « revue carte par carte ». Une ligne par FOND publie, une ligne
> par carte jouee SANS fond. Toute ligne porte un statut ; aucune ligne sans statut a la
> cloture d'un lot. Plan : `PLAN_REVUE_CARTE_PAR_CARTE.md`.
>
> Etabli le 2026-08-26 sur pieces (corpus `match_registry`, fonds publies, mesure de cadrage).
> Branche `wt/cartes-revue-par-carte`.

## Statuts

| statut | sens |
|---|---|
| `VALIDEE <date>` | verdict utilisateur positif au gate visuel, sur l'image PUBLIEE aujourd'hui |
| `REFUSEE <date>` | verdict utilisateur negatif ; l'image publiee est celle qui a ete refusee |
| `ATTENTE` | image re-cuite depuis le dernier verdict — jamais soumise, verdict inconnu |
| `A CUIRE` | pas de fond publie, la chaine s'applique telle quelle |
| `BLOQUEE <raison>` | pas de fond publie, un obstacle nomme empeche la cuisson |
| `HORS PERIMETRE <raison>` | ecartee volontairement, raison ecrite |

Regle : un statut `VALIDEE` ne survit PAS a une re-cuisson qui change l'image. Toute cuisson
qui modifie un PNG repasse sa ligne en `ATTENTE`.

## Chiffres d'entree (2026-08-26)

- Corpus joue : **123 map_id, 1 940 matchs** (`match_registry`, halo_infinite).
- Fonds publies : **56** (19 natifs keyes par module, 37 Forge keyes par map_id), tous servis.
- Couverture : **79 map_id / 1 731 matchs (89,2 %)** ; **44 map_id / 209 matchs (10,8 %) sans fond**.
- Verdicts acquis : **12 natifs valides**, **7 natifs en attente**, **2 Forge valides et geles**,
  **35 Forge refuses en bloc**. Soit **14 fonds valides sur 56**.
- Cadrage (`cmd/mapfond-cadrage`, part de la LARGEUR occupee par la matiere dessinee) :
  mediane **natifs 53,5 %**, **Forge 88,3 %**. Les deux familles ont le defaut inverse.

## Fonds publies (56)

`occL` / `occA` = part de la largeur / de l'aire du cadre reellement occupee par la matiere.
`matchs` = somme des matchs de tous les map_id servis par cette cle.

| carte(s) servie(s) | cle du fond | matchs | occL % | occA % | famille | statut |
|---|---|---:|---:|---:|---|---|
| Highpower Heavies / Highpower | `btb_highpower` | 108 | 59.1 | 14.5 | natif | VALIDEE 10/08 |
| Deadlock / Deadlock Heavies | `btb_drydock` | 97 | 67.1 | 24.5 | natif | VALIDEE 10/08 |
| Oasis / Oasis Heavies | `btb_exiled` | 92 | 61.0 | 18.2 | natif | VALIDEE 10/08 |
| Fragmentation / Fragmentation Heavies | `btb_fragmentation` | 92 | 87.4 | 24.9 | natif | VALIDEE 10/08 |
| Breaker / Breaker Heavies | `ctf_breaker` | 76 | 52.4 | 18.3 | natif | VALIDEE 10/08 |
| Streets - Ranked / Streets | `sgh_streets` | 62 | 40.6 | 12.2 | natif | VALIDEE 10/08 |
| Cliffhanger | `ridgeline` | 58 | 68.6 | 18.6 | natif | VALIDEE 10/08 |
| Scarr | `btb_engine` | 57 | 65.9 | 17.3 | natif | ATTENTE (re-cuite 13/08) |
| Bazaar | `ctf_bazaar` | 56 | 39.3 | 17.3 | natif | VALIDEE 10/08 |
| Illusion | `ctf_illusion` | 56 | 76.3 | 15.0 | natif | ATTENTE (re-cuite 13/08) |
| Recharge - Ranked / Recharge | `sgh_blueprint` | 56 | 28.8 | 8.7 | natif | VALIDEE 10/08 |
| Aquarius / Aquarius - Ranked | `ctf_aquarius` | 54 | 33.7 | 5.7 | natif | ATTENTE (re-cuite 13/08) |
| Chasm | `chasm` | 52 | 100.0 | 10.0 | natif | ATTENTE (re-cuite 13/08) |
| Forest - Ranked / Forest | `forest` | 49 | 35.8 | 9.8 | natif | VALIDEE 10/08 |
| Prism | `sgh_crystalcaves` | 49 | 49.2 | 15.6 | natif | ATTENTE (re-cuite 13/08) |
| Catalyst | `catalyst` | 48 | 50.0 | 12.0 | natif | VALIDEE 10/08 |
| Forbidden | `ctf_forbidden` | 46 | 44.7 | 17.9 | natif | ATTENTE (re-cuite 13/08) |
| Behemoth | `va_behemoth` | 44 | 83.2 | 15.9 | natif | VALIDEE 10/08 |
| Empyrean | `d035fc3e` | 29 | 68.7 | 35.6 | forge | REFUSEE 13/08 |
| Origin | `b302eb62` | 24 | 85.7 | 48.4 | forge | REFUSEE 13/08 |
| Launch Site | `va_launchsite` | 24 | 53.5 | 12.8 | natif | ATTENTE (re-cuite 13/08) |
| Starboard | `7a9265af` | 24 | 53.6 | 18.6 | forge | REFUSEE 13/08 |
| Snowbound | `410f1c01` | 23 | 100.0 | 35.7 | forge | REFUSEE 13/08 |
| The Pit | `648ae7aa` | 22 | 81.5 | 12.0 | forge | REFUSEE 13/08 |
| Absolution | `78da545f` | 21 | 89.9 | 10.9 | forge | REFUSEE 13/08 |
| Curfew | `63d634be` | 20 | 61.7 | 19.7 | forge | REFUSEE 13/08 |
| Dynasty | `cfd90b63` | 19 | 100.0 | 75.5 | forge | REFUSEE 13/08 |
| Nemesis | `2be34415` | 18 | 98.6 | 44.5 | forge | REFUSEE 13/08 |
| Cliffside | `4bffd021` | 18 | 98.0 | 46.9 | forge | REFUSEE 13/08 |
| Shiro | `2890782c` | 18 | 84.0 | 55.5 | forge | REFUSEE 13/08 |
| Domicile | `921aebb1` | 17 | 88.3 | 48.2 | forge | REFUSEE 13/08 |
| Goliath | `504ebf22` | 17 | 52.2 | 13.8 | forge | REFUSEE 13/08 |
| Isolation | `01af558d` | 17 | 93.9 | 49.7 | forge | REFUSEE 13/08 |
| Fortress | `0d1c9255` | 17 | 87.4 | 22.4 | forge | REFUSEE 13/08 |
| Dredge | `e4bb06db` | 16 | 91.5 | 28.7 | forge | REFUSEE 13/08 |
| Vagabond | `105f5d84` | 16 | 74.3 | 40.3 | forge | VALIDEE 13/08 (gelee) |
| Houseki | `cf034ec8` | 15 | 86.0 | 54.1 | forge | REFUSEE 13/08 |
| High Ground | `bb7b78ae` | 15 | 100.0 | 52.2 | forge | REFUSEE 13/08 |
| Takamanohara | `edcd4467` | 15 | 94.1 | 51.2 | forge | REFUSEE 13/08 |
| Elevation | `76043dc6` | 14 | 92.3 | 16.2 | forge | REFUSEE 13/08 |
| Kiken'na | `df7dbf08` | 13 | 94.9 | 44.4 | forge | REFUSEE 13/08 |
| Kaiketsu | `98a83f87` | 12 | 86.2 | 40.7 | forge | REFUSEE 13/08 |
| Banished Narrows | `9ad226d8` | 12 | 100.0 | 85.2 | forge | REFUSEE 13/08 |
| Salvation | `cd08bc7a` | 12 | 73.9 | 31.4 | forge | REFUSEE 13/08 |
| Solitude | `f1cc3b4e` | 12 | 55.4 | 17.3 | forge | REFUSEE 13/08 |
| Opulence | `255bbe78` | 12 | 61.9 | 19.7 | forge | REFUSEE 13/08 |
| Command | `2c9f3490` | 11 | 100.0 | 64.7 | forge | REFUSEE 13/08 |
| Critical Dewpoint | `bae4df14` | 10 | 55.7 | 22.8 | forge | REFUSEE 13/08 |
| Perilous | `c5ac9f12` | 10 | 100.0 | 43.7 | forge | REFUSEE 13/08 |
| Sylvanus | `95b69e4b` | 10 | 87.5 | 35.5 | forge | REFUSEE 13/08 |
| Refuge | `41217472` | 10 | 77.0 | 37.8 | forge | REFUSEE 13/08 |
| Smallhalla | `98783453` | 9 | 100.0 | 86.2 | forge | REFUSEE 13/08 |
| Obituary | `a289bafe` | 9 | 92.7 | 54.9 | forge | REFUSEE 13/08 |
| Shogun | `33075df7` | 9 | 100.0 | 68.0 | forge | REFUSEE 13/08 |
| Fortitude | `1ede38fa` | 7 | 99.1 | 57.2 | forge | REFUSEE 13/08 |
| Corpo | `8be179f7` | 2 | 61.2 | 3.3 | forge | VALIDEE 13/08 (gelee) |

## Cartes jouees SANS fond (44 map_id, 209 matchs)

Statut par defaut `A CUIRE`, sauf les trois blocages nommes ci-dessous.

- `BLOQUEE natif sans tag sbsp` — **Live Fire** (3 map_id, 71 matchs) : deja instruite
  (`HANDOFF_PORT_TRIANGLES` §1 ter), aucune cuisson possible en l'etat.
- `BLOQUEE canevas inconnu` — **Detachment** (25 matchs), **Argyle** (22 matchs).
- Le reste : cartes Forge du reliquat, `.mvar` au depot, chaine deja ecrite
  (`RAPPORT_FONDS_MAP_ID_2026-08-13.md`, section « Reliquat »).

| carte | map_id | matchs |
|---|---|---:|
| Live Fire | `6c01f693` | 51 |
| Detachment | `d39600e2` | 25 |
| Argyle | `dd600260` | 22 |
| Live Fire | `b6aca0c7` | 18 |
| Ecotone | `8816f240` | 8 |
| Insolence | `d5c5eb4f` | 7 |
| Solution | `ee43d273` | 7 |
| Flood Gulch | `7097bc4f` | 6 |
| Threshold | `ddbb3a00` | 5 |
| Solitude - Ranked | `4a5e5612` | 5 |
| Thunderhead | `28a3ac28` | 4 |
| Fortitude Heavies | `305b1bdd` | 4 |
| Thunderhead Heavies | `37bc3df6` | 4 |
| Salvation | `f633db01` | 3 |
| Obituary Heavies | `e3681516` | 3 |
| Pharaoh | `88d45250` | 3 |
| Credence | `0cc728d2` | 2 |
| Urban Raid | `be848f91` | 2 |
| Disciple | `525451ca` | 2 |
| Merchant's Square | `7dfec55d` | 2 |
| Live Fire - Ranked | `309253f8` | 2 |
| Vallaheim Firefight | `e8268e75` | 2 |
| Shogun | `8f51ccb9` | 1 |
| Lattice - Ranked | `1a6cfc2e` | 1 |
| Scarlett's Landing | `79042fc0` | 1 |
| Shiro | `2962c4e0` | 1 |
| Rat's Nest | `133c0185` | 1 |
| Origin - Ranked | `46a8319c` | 1 |
| Ronin | `f459867d` | 1 |
| Nadair | `6dbd1c0d` | 1 |
| Highpower Sentry Defense | `142a5e23` | 1 |
| Oasis Firefight | `f566aa62` | 1 |
| Oasis Sentry Defense | `052956b4` | 1 |
| TFF | Night Of The Undead | `ae4daed6` | 1 |
| Cole Protocol | `571afb7f` | 1 |
| Outlook | `ea7b30e6` | 1 |
| 944396dd-5661-4a16-b1d8-a6053f762c55 | `944396dd` | 1 |
| Houseki | `6439625e` | 1 |
| Starboard | `50771a22` | 1 |
| Dawnbreaker | `89dd4003` | 1 |
| Warehouse | `5b12d6d9` | 1 |
| Dynasty | `90cd321d` | 1 |
| Insolence Heavies | `2a339c65` | 1 |
| Refuge Heavies | `c10c7e79` | 1 |

## Journal des verdicts

Une entree par gate soumis a l'utilisateur : date, lot, cartes soumises, verbatim du verdict,
lignes mises a jour. Rien ne se coche ici sans verbatim.

*(vide — premier gate a venir)*

## Decouvertes hors fonds (defauts de CADRAGE cote rejeu)

| date | defaut | etat |
|---|---|---|
| 2026-08-26 | `sceneBounds` gonflait le cadre du rejeu avec `geometryBounds` (props Forge) MEME quand un fond de carte est pose — or les props ne sont alors PAS dessines (`else if` du fond dans `ReplayCanvas`). Cadre dimensionne sur de la matiere invisible, carte reduite a un timbre. | CORRIGE (`replayLogic.ts`, 3 temoins, mutation verifiee) |
| 2026-08-26 | Le cadre des fonds PUBLIES est la boite des ancres plus 50 m constants, jamais recalcule apres la coquille : mediane 53,5 % de largeur utile sur les natifs. | phase 2 du plan |
