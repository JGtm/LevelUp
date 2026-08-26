# Registre des cartes — revue carte par carte (v7.5)

> SOURCE DE VERITE du chantier « revue carte par carte ». Une ligne par FOND publie, une ligne
> par carte jouee SANS fond. Toute ligne porte un statut ; aucune ligne sans statut a la
> cloture d'un lot. Plan : `PLAN_REVUE_CARTE_PAR_CARTE.md`.
>
> Etabli le 2026-08-26 sur pieces (corpus `match_registry`, fonds publies, mesure de cadrage).
> Statuts a jour du gate utilisateur du **2026-08-26** (verbatim au journal, en bas).
> Branche `wt/cartes-revue-par-carte`.

## Statuts

| statut | sens |
|---|---|
| `VALIDEE <date>` | verdict utilisateur positif sur l'image PUBLIEE ; on n'y retouche pas |
| `REFUSEE <date>` | verdict negatif ; l'image publiee est celle qui a ete refusee |
| `A FINALISER` | jugee non close par l'utilisateur : elle attend les etapes suivantes |
| `A RETRAVAILLER` | refusee sur un point nomme, avec sa cible ecrite |
| `A TRANCHER` | une question ouverte bloque le verdict |
| `A CUIRE` | pas de fond publie, la chaine s'applique telle quelle |
| `BLOQUEE <raison>` | pas de fond publie, un obstacle nomme empeche la cuisson |
| `HORS PERIMETRE <raison>` | ecartee volontairement, raison ecrite |

Regle : un statut `VALIDEE` ne survit PAS a une re-cuisson qui change l'image. Toute cuisson
qui modifie un PNG repasse sa ligne en `A FINALISER`.

## Chiffres d'entree (2026-08-26)

- Corpus joue : **123 map_id, 1 940 matchs** (`match_registry`, halo_infinite).
- Fonds publies : **56** (19 natifs keyes par module, 37 Forge keyes par map_id), tous servis.
- Couverture : **79 map_id / 1 731 matchs (89,2 %)** ; **44 map_id / 209 matchs (10,8 %) sans fond**.
- Cadrage (`cmd/mapfond-cadrage`, part de la LARGEUR occupee par la matiere dessinee) :
  mediane **natifs 53,5 %**, **Forge 88,3 %**. Les deux familles ont le defaut inverse.
- Ecart mediane-ancres / sol (`anchorMedianGapM`) : 43 fonds entre -0,40 et +0,03 m
  (l'etalonnage `AncrageDecalageSol = 0,29 m` tient), **13 hors clou** de -3,29 a -17,51 m.
  Au-dela de `PorteeNiveauDeJeu = 10 m`, la surface JOUEE est peinte au recul maximal.

## Etat apres le gate du 2026-08-26

| statut | fonds |
|---|---:|
| `VALIDEE 26/08` | 9 |
| `A FINALISER` | 9 |
| `A TRANCHER` | 1 |
| `A RETRAVAILLER` | 2 |
| `REFUSEE 13/08` | 35 |

## Fonds publies (56)

`occL` = part de la largeur du cadre occupee par la matiere. `ecart` = `anchorMedianGapM`, en
metres (attendu : -0,29). `matchs` = somme des matchs de tous les map_id servis par cette cle.

| carte(s) servie(s) | cle du fond | matchs | occL % | ecart m | famille | statut |
|---|---|---:|---:|---:|---|---|
| Highpower Heavies / Highpower | `btb_highpower` | 108 | 59.1 | -0,29 | natif | VALIDEE 26/08 |
| Deadlock / Deadlock Heavies | `btb_drydock` | 97 | 67.1 | -0,30 | natif | VALIDEE 26/08 |
| Oasis / Oasis Heavies | `btb_exiled` | 92 | 61.0 | -4,38 | natif | VALIDEE 26/08 |
| Fragmentation / Fragmentation Heavies | `btb_fragmentation` | 92 | 87.4 | -11,71 | natif | VALIDEE 26/08 |
| Breaker / Breaker Heavies | `ctf_breaker` | 76 | 52.4 | -3,29 | natif | VALIDEE 26/08 |
| Streets - Ranked / Streets | `sgh_streets` | 62 | 40.6 | -4,33 | natif | A FINALISER |
| Cliffhanger | `ridgeline` | 58 | 68.6 | -0,32 | natif | A TRANCHER — style `encre` ? |
| Scarr | `btb_engine` | 57 | 65.9 | -0,13 | natif | VALIDEE 26/08 |
| Bazaar | `ctf_bazaar` | 56 | 39.3 | -4,22 | natif | A FINALISER |
| Illusion | `ctf_illusion` | 56 | 76.3 | -0,26 | natif | A FINALISER |
| Recharge - Ranked / Recharge | `sgh_blueprint` | 56 | 28.8 | -5,29 | natif | A FINALISER |
| Aquarius / Aquarius - Ranked | `ctf_aquarius` | 54 | 33.7 | -0,05 | natif | A FINALISER |
| Chasm | `chasm` | 52 | 100.0 | -0,09 | natif | A FINALISER |
| Forest - Ranked / Forest | `forest` | 49 | 35.8 | -0,30 | natif | VALIDEE 26/08 |
| Prism | `sgh_crystalcaves` | 49 | 49.2 | -0,23 | natif | A FINALISER |
| Catalyst | `catalyst` | 48 | 50.0 | -13,33 | natif | A RETRAVAILLER — cible = temoin 10/08 |
| Forbidden | `ctf_forbidden` | 46 | 44.7 | -0,21 | natif | A FINALISER |
| Behemoth | `va_behemoth` | 44 | 83.2 | -17,51 | natif | VALIDEE 26/08 |
| Empyrean | `d035fc3e` | 29 | 68.7 | -14,41 | forge | REFUSEE 13/08 |
| Origin | `b302eb62` | 24 | 85.7 | 0,01 | forge | REFUSEE 13/08 |
| Launch Site | `va_launchsite` | 24 | 53.5 | -0,40 | natif | A FINALISER |
| Starboard | `7a9265af` | 24 | 53.6 | -0,29 | forge | REFUSEE 13/08 |
| Snowbound | `410f1c01` | 23 | 100.0 | -0,00 | forge | REFUSEE 13/08 |
| The Pit | `648ae7aa` | 22 | 81.5 | -15,53 | forge | REFUSEE 13/08 |
| Absolution | `78da545f` | 21 | 89.9 | -0,08 | forge | REFUSEE 13/08 |
| Curfew | `63d634be` | 20 | 61.7 | -0,03 | forge | REFUSEE 13/08 |
| Dynasty | `cfd90b63` | 19 | 100.0 | -0,05 | forge | REFUSEE 13/08 |
| Nemesis | `2be34415` | 18 | 98.6 | -0,07 | forge | REFUSEE 13/08 |
| Cliffside | `4bffd021` | 18 | 98.0 | 0,00 | forge | REFUSEE 13/08 |
| Shiro | `2890782c` | 18 | 84.0 | -0,01 | forge | REFUSEE 13/08 |
| Domicile | `921aebb1` | 17 | 88.3 | -8,62 | forge | REFUSEE 13/08 |
| Goliath | `504ebf22` | 17 | 52.2 | -0,02 | forge | REFUSEE 13/08 |
| Isolation | `01af558d` | 17 | 93.9 | -0,04 | forge | REFUSEE 13/08 |
| Fortress | `0d1c9255` | 17 | 87.4 | -10,60 | forge | REFUSEE 13/08 |
| Dredge | `e4bb06db` | 16 | 91.5 | -0,25 | forge | REFUSEE 13/08 |
| Vagabond | `105f5d84` | 16 | 74.3 | -0,01 | forge | A RETRAVAILLER — gros |
| Houseki | `cf034ec8` | 15 | 86.0 | -0,11 | forge | REFUSEE 13/08 |
| High Ground | `bb7b78ae` | 15 | 100.0 | -0,01 | forge | REFUSEE 13/08 |
| Takamanohara | `edcd4467` | 15 | 94.1 | -0,29 | forge | REFUSEE 13/08 |
| Elevation | `76043dc6` | 14 | 92.3 | -0,25 | forge | REFUSEE 13/08 |
| Kiken'na | `df7dbf08` | 13 | 94.9 | -0,05 | forge | REFUSEE 13/08 |
| Kaiketsu | `98a83f87` | 12 | 86.2 | 0,01 | forge | REFUSEE 13/08 |
| Banished Narrows | `9ad226d8` | 12 | 100.0 | -6,05 | forge | REFUSEE 13/08 |
| Salvation | `cd08bc7a` | 12 | 73.9 | -0,02 | forge | REFUSEE 13/08 |
| Solitude | `f1cc3b4e` | 12 | 55.4 | -0,14 | forge | REFUSEE 13/08 |
| Opulence | `255bbe78` | 12 | 61.9 | 0,01 | forge | REFUSEE 13/08 |
| Command | `2c9f3490` | 11 | 100.0 | -0,00 | forge | REFUSEE 13/08 |
| Critical Dewpoint | `bae4df14` | 10 | 55.7 | -0,01 | forge | REFUSEE 13/08 |
| Perilous | `c5ac9f12` | 10 | 100.0 | -0,03 | forge | REFUSEE 13/08 |
| Sylvanus | `95b69e4b` | 10 | 87.5 | -0,02 | forge | REFUSEE 13/08 |
| Refuge | `41217472` | 10 | 77.0 | -0,10 | forge | REFUSEE 13/08 |
| Smallhalla | `98783453` | 9 | 100.0 | 0,03 | forge | REFUSEE 13/08 |
| Obituary | `a289bafe` | 9 | 92.7 | -0,00 | forge | REFUSEE 13/08 |
| Shogun | `33075df7` | 9 | 100.0 | -0,03 | forge | REFUSEE 13/08 |
| Fortitude | `1ede38fa` | 7 | 99.1 | -0,11 | forge | REFUSEE 13/08 |
| Corpo | `8be179f7` | 2 | 61.2 | -0,25 | forge | VALIDEE 26/08 |

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

Une entree par gate soumis a l'utilisateur : date, lot, verbatim, lignes mises a jour.
Rien ne se coche ici sans verbatim.

### 2026-08-26 — planche d'etat des lieux (56 fonds + Catalyst a trois etats)

Planche : https://claude.ai/code/artifact/5e8fa28d-da9e-4eba-898d-33174158be40

**Verbatim** :

> Catalyst c'est "Temoin du 10/08 12:24 (jamais en production)" qui est le meilleur meme si
> on doit faire les etapes suivantes sur cette carte.
> Scarr valide en l'etat
> Streets, Bazaar, Illusion, Chasm, Prism, Forbidden, Laun site, Recharge, Aquarius a
> finaliser sur les etapes suivantes
> Gros retravail sur Vagabond a faire
> Cliffhanger c'est la version en prod ? J'avais bien aime cette version :
> `Desktop/COULEUR_encre_cliffhanger.png`
> Sinon le reste des validees sont ok on ne les retouche pas.
> Les refusees restent refusees

**Ce que le gate tranche** :

- L'ambiguite du 13/08 est LEVEE : les 7 natifs re-cuits sont juges. Scarr passe
  `VALIDEE`, les six autres (Illusion, Prism, Aquarius, Forbidden, Chasm, Launch Site)
  passent `A FINALISER`.
- Trois natifs qui etaient tenus pour valides depuis le 10/08 sont RETIRES du valide :
  Streets, Bazaar, Recharge. Les trois portent un ecart mediane-ancres / sol hors clou
  (-4,33 / -4,22 / -5,29 m).
- **Catalyst : la production est une REGRESSION** — le temoin du 10/08 12:24, jamais
  livre, est declare meilleur. La cible de la carte est donc ecrite : ce temoin.
- Vagabond perd son gel `CarteForge.FondFige` : il avait ete gele « a revoir » le 13/08,
  le gate demande un gros retravail. Le gel n'a plus de raison d'etre.
- Cliffhanger : question ouverte. Verifie sur pieces le 26/08 — le fond en production
  est en style `jeu` (fond noir, arene claire) ; `COULEUR_encre_cliffhanger.png` est en
  style `encre` (quasi monochrome sur blanc, riviere bleue). **Ce ne sont PAS les memes.
  Le style prefere ici n'est pas celui qui est livre.**
- Les 35 Forge refuses le 13/08 restent refuses.

**Ce que le gate ouvre** : le STYLE devient un choix PAR CARTE. `StyleJeu` et
`StyleEncre` existent tous deux en production (`fond_png.go`), le sidecar publie deja le
style retenu, et `mapfond-build --style` est GLOBAL. Passer ce choix en donnee par carte
est le premier levier « chaque carte a sa maniere » — aucune nouvelle regle de rendu.

**Demande produit hors fonds** : un ZOOM dans la page de rejeu, pour lire le rejeu
lui-meme sur les petites cartes. Verifie le 26/08 : il n'y en a aucun aujourd'hui —
`CANVAS_HEIGHT` est fige a 480 px et `fitWidth` ajuste la scene a cette hauteur. Lot a
part, hors du perimetre des fonds.

**Precision utilisateur sur les zones jamais foulees** : elles servent de SECONDE BASE
de travail pour atteindre une version valide, pas de regle de rendu final.



## Decouvertes hors fonds (defauts de CADRAGE cote rejeu)

| date | defaut | etat |
|---|---|---|
| 2026-08-26 | `sceneBounds` gonflait le cadre du rejeu avec `geometryBounds` (props Forge) MEME quand un fond de carte est pose — or les props ne sont alors PAS dessines (`else if` du fond dans `ReplayCanvas`). Cadre dimensionne sur de la matiere invisible, carte reduite a un timbre. | CORRIGE (`replayLogic.ts`, 3 temoins, mutation verifiee) |
| 2026-08-26 | Le cadre des fonds PUBLIES est la boite des ancres plus 50 m constants, jamais recalcule apres la coquille : mediane 53,5 % de largeur utile sur les natifs. | phase 2 du plan |
| 2026-08-26 | **Le catalogue d objectifs avait perdu 59 modules sur 71** (`d50f3b728`, re-tirage reseau du 25/08) : `module` ecrase par le nom de fichier servi par le reseau (`map.mvar`) sur 58 des 73 entrees. `mapfond-build` ne savait plus cuire que 11 des 19 fonds natifs PUBLIES — Cliffhanger, Aquarius, Prism, Streets, Recharge, Chasm, Launch Site, Behemoth incuisables, dont SIX sur la liste a finaliser du jour. Le garde-rail du lot fautif comptait les collines, il etait vert. | CORRIGE `d014fce69` (donnee reparee, `gardeModuleConnu`, `TestCatalogueObjectifsModulesDistincts`, mordant par mutation) |

## Planches de gate

| date | lot | planche | contenu |
|---|---|---|---|
| 2026-08-26 | etat des lieux | https://claude.ai/code/artifact/5e8fa28d-da9e-4eba-898d-33174158be40 | les 56 fonds publies + Catalyst a trois etats (publie / re-cuisson 26/08 / temoin 10/08) |
| 2026-08-26 | style `jeu` / `encre` | https://claude.ai/code/artifact/6c9ec756-95e6-451a-8977-5e61debb8bae | les 19 fonds natifs dans les deux habillages, ordre des verdicts du 26/08 |

Outil : `cmd/mapfond-planche` — manifeste TSV (cle, libelle, sous-titre, statut, colonne,
chemin PNG), une page HTML autonome, vignettes en data URI. Plusieurs lignes de meme cle
deviennent les colonnes d'une meme fiche : c'est la comparaison avant / apres.
