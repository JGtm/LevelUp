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
| Highpower Heavies / Highpower | `btb_highpower` | 108 | 59.1 | -0,29 | natif | VALIDEE 30/08 — vides ouverts combles |
| Deadlock / Deadlock Heavies | `btb_drydock` | 97 | 67.1 | -0,30 | natif | VALIDEE 26/08 |
| Oasis / Oasis Heavies | `btb_exiled` | 92 | 61.0 | -4,38 | natif | VALIDEE 30/08 — vides ouverts combles |
| Fragmentation / Fragmentation Heavies | `btb_fragmentation` | 92 | 87.4 | -11,71 | natif | VALIDEE 30/08 — vides ouverts combles |
| Breaker / Breaker Heavies | `ctf_breaker` | 76 | 52.4 | -3,29 | natif | VALIDEE 26/08 |
| Streets - Ranked / Streets | `sgh_streets` | 62 | 40.6 | -4,33 | natif | VALIDEE 26/08 — `encre` + cadre + echelle + toits + zones |
| Cliffhanger | `ridgeline` | 58 | 68.6 | -0,32 | natif | VALIDEE `encre` 26/08 |
| Scarr | `btb_engine` | 57 | 65.9 | -0,13 | natif | VALIDEE 26/08 |
| Bazaar | `ctf_bazaar` | 56 | 39.3 | -4,22 | natif | VALIDEE 26/08 — les quatre traitements |
| Illusion | `ctf_illusion` | 56 | 76.3 | -0,26 | natif | VALIDEE 26/08 — ecretage 6 m + comblement |
| Recharge - Ranked / Recharge | `sgh_blueprint` | 56 | 28.8 | -5,29 | natif | VALIDEE 26/08 — 6 reglages (plafond 4 m, sans eau) |
| Aquarius / Aquarius - Ranked | `ctf_aquarius` | 54 | 33.7 | -0,05 | natif | VALIDEE `encre` 26/08 — plafond 2 m |
| Chasm | `chasm` | 52 | 100.0 | -0,09 | natif | VALIDEE `encre` 26/08 — boite manuelle + plancher -12 |
| Forest - Ranked / Forest | `forest` | 49 | 35.8 | -0,30 | natif | VALIDEE 26/08 |
| Prism | `sgh_crystalcaves` | 49 | 49.2 | -0,23 | natif | VALIDEE `encre` 26/08 — plafond 2 m + masque |
| Catalyst | `catalyst` | 48 | 50.0 | -0,19 | natif | VALIDEE `encre` 26/08 — boite mesuree sur positions |
| Forbidden | `ctf_forbidden` | 46 | 44.7 | -0,21 | natif | VALIDEE `encre` 26/08 — plafond 4 m + masque 63 zones |
| Behemoth | `va_behemoth` | 44 | 83.2 | -17,51 | natif | A FINALISER — retouche manuelle par l utilisateur (fond publie = vides ouverts combles) |
| Empyrean | `d035fc3e` | 29 | 68.7 | -14,41 | forge | REFUSEE 13/08 |
| Origin | `b302eb62` | 24 | 85.7 | 0,01 | forge | REFUSEE 13/08 |
| Launch Site | `va_launchsite` | 24 | 53.5 | -0,40 | natif | VALIDEE `encre` 26/08 — SANS ecretage, comblement + sans eau |
| Starboard | `7a9265af` | 24 | 53.6 | -0,29 | forge | VALIDEE 30/08 |
| Snowbound | `410f1c01` | 23 | 100.0 | -0,00 | forge | REFUSEE 13/08 |
| The Pit | `648ae7aa` | 22 | 81.5 | -15,53 | forge | REFUSEE 13/08 |
| Absolution | `78da545f` | 21 | 89.9 | -0,08 | forge | REFUSEE 13/08 |
| Curfew | `63d634be` | 20 | 61.7 | -0,03 | forge | REFUSEE 13/08 |
| Dynasty | `cfd90b63` | 19 | 100.0 | -0,05 | forge | REFUSEE 13/08 |
| Nemesis | `2be34415` | 18 | 98.6 | -0,07 | forge | REFUSEE 13/08 |
| Cliffside | `4bffd021` | 18 | 98.0 | 0,00 | forge | REFUSEE 13/08 |
| Shiro | `2890782c` | 18 | 84.0 | -0,01 | forge | REFUSEE 13/08 |
| Domicile | `921aebb1` | 17 | 88.3 | -8,62 | forge | VALIDEE 30/08 |
| Goliath | `504ebf22` | 17 | 52.2 | -0,02 | forge | VALIDEE 30/08 |
| Isolation | `01af558d` | 17 | 93.9 | -0,04 | forge | VALIDEE `encre` 27/08 — navmesh (reference + rognage + tolerance 1,5 m) + zones de callout marge 1 m |
| Fortress | `0d1c9255` | 17 | 87.4 | -10,60 | forge | REFUSEE 13/08 |
| Dredge | `e4bb06db` | 16 | 91.5 | -0,25 | forge | REFUSEE 13/08 |
| Vagabond | `105f5d84` | 16 | 74.3 | -0,01 | forge | A RETRAVAILLER — gros |
| Houseki | `cf034ec8` | 15 | 86.0 | -0,11 | forge | VALIDEE 30/08 |
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
| Corpo | `8be179f7` | 2 | 61.2 | -0,25 | forge | A FINALISER — correction utilisateur 26/08 |

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
| 2026-08-26 | carte 1 — Cliffhanger | https://claude.ai/code/artifact/e3f8f959-14a3-44a8-b426-26a27f13832b | avant/apres de la SEULE carte publiee en `encre` |
| 2026-08-26 | carte 2 — Streets | https://claude.ai/code/artifact/b74cff7f-c885-423c-b04e-aa854c260ba9 | avant/apres, publiee en `encre` |

Outil : `cmd/mapfond-planche` — manifeste TSV (cle, libelle, sous-titre, statut, colonne,
chemin PNG), une page HTML autonome, vignettes en data URI. Plusieurs lignes de meme cle
deviennent les colonnes d'une meme fiche : c'est la comparaison avant / apres.

## Boucle de travail — UNE carte a la fois (consigne utilisateur du 2026-08-26)

> « je pense que tu n'as pas compris quand je te dis qu'il faut retravailler une a une celles
> qui sont non validees. On devrait travailler une a une non ? »

La planche de masse a servi a UNE chose : obtenir le verdict de style. Elle est close. Desormais,
carte par carte :

1. cuire la carte SEULE vers le dossier de PRODUCTION (pas un scratch) ;
2. publier sa planche avant/apres, cette carte seule ;
3. verdict utilisateur ; s'il refuse, corriger sur ELLE et republier ;
4. la ligne passe `VALIDEE` avec le verbatim, l'entree de reglage porte sa raison et sa date ;
5. seulement alors, la carte suivante.

Ordre retenu (matchs decroissants parmi les non closes) : **Cliffhanger** (58, publiee, en
attente de verdict) → Streets (62) → Bazaar (56) → Recharge (56) → Illusion (56) → Aquarius
(54) → Chasm (52) → Prism (49) → Catalyst (48) → Forbidden (46) → Launch Site (24) → Vagabond (16, gros
retravail). Streets pese plus que Cliffhanger, mais Cliffhanger etait deja tranchee.

### 2026-08-26 — carte 1, Cliffhanger

Planche : https://claude.ai/code/artifact/e3f8f959-14a3-44a8-b426-26a27f13832b

**Verbatim** : « je valide cliffhanger comme ca on peut passer a la suivante ? »

`ridgeline` passe `VALIDEE encre 26/08`. Le fond publie est en style `encre`, reglage declare
en donnee avec sa raison et sa date. Ses deux defauts connus restent OUVERTS et ne sont pas
couverts par ce verdict : largeur utile 68,6 % du cadre, et l'echelle.

### 2026-08-26 — carte 2, Streets

Planche : https://claude.ai/code/artifact/b74cff7f-c885-423c-b04e-aa854c260ba9 (quatre etats)

**Verbatim** : « ok avec le rendu 4, je valide on peut passer a la suite »

`sgh_streets` passe `VALIDEE` avec les QUATRE traitements, tous declares en donnee :

| axe | valeur | effet mesure |
|---|---|---|
| habillage | `encre` | la teinte ne bascule plus, seule la valeur varie |
| echelle | 0,036 m/px | matiere utile 549 x 426 px -> 1 737 x 1 422 px |
| cadre | rogne a la matiere + 6 m | 40,6 % de largeur utile -> 80,8 % |
| toits | ecretes | 495 752 pixels substitues, 0 vide ; ecart au sol -2,48 m -> -0,29 m |
| zones | rogne aux callouts + 4 m | 306 810 cellules effacees sur 1 526 464, soit 20,1 % |

**Ce que ce gate etablit au-dela de Streets** :

- Le rognage aux zones nommees est **defendable** : la ceinture exterieure qui faisait de la
  carte une dalle rectangulaire disparait, la silhouette devient celle d'une carte.
- **Limite connue et non corrigee** : le contour porte des marches rectilignes — les carres de
  la dilatation, pas le terrain. Une dilatation circulaire les supprimerait ; la marge de 4 m
  est un reglage, pas une verite.
- **Ce rognage ne vaudra JAMAIS pour les cartes Forge** : 0 callout sur les 37 fonds Forge.
  Ce sont pourtant les plus mal cadrees (88,3 % de largeur occupee, la « bouillie » refusee).
- Le seuil `SeuilCarteCouverte` (1/3) est en cause plus que le mecanisme de substitution :
  Streets, mesuree a 7,1 %, n'a jamais declenche la voie de reference qui lui allait.

### 2026-08-26 — carte 3, Bazaar

Planche : https://claude.ai/code/artifact/4aa211a3-cd31-423b-b3ad-780f8fbf2e41

**Verbatim** : « validé, on passe à la suivante »

`ctf_bazaar` passe `VALIDEE` avec les quatre traitements (`encre`, 0,034 m/px, cadre rogne,
ecretage, masque des 29 zones). Matiere utile 621 x 567 -> **2 032 x 1 462 px**.

**Trois chiffres l'ecartent de Streets, et ils sont assumes au verdict** :

1. Le masque retire **39,5 %** de la matiere (1 012 535 sur 2 564 541) contre 20,1 % sur
   Streets — presque le double. Signale AVANT le verdict, pas apres.
2. L'ecart mediane-ancres / sol reste a **-3,20 m APRES ecretage** (Streets tombait a -0,29).
   **Sur Bazaar, les toits n'expliquent PAS l'ecart au sol.** Le defaut de niveau de jeu est
   donc reel et distinct — le correctif « reference par pixel au lieu d'une mediane unique »
   reste du, et l'ecretage ne le rendra pas inutile.
3. L'ecretage a **VIDE 20 856 pixels** (zero sur Streets) : Bazaar a de vrais couvercles sans
   rien dessous. Le mecanisme sert enfin a ce pour quoi il a ete ecrit.

### 2026-08-26 — carte 4, Recharge : PARKEE, defaut anterieur

`sgh_blueprint` reste `A FINALISER`. Le temoin n'a PAS ete soumis au gate.

**Ce qui bloque** : le fond de Recharge **publie depuis le 2026-08-10** porte deja une DALLE
BLEUE rectangulaire par-dessus sa structure haute. Verifie en ouvrant le PNG en production.
Ce n'est donc ni l'ecretage, ni l'echelle, ni le masque — c'est un defaut ANTERIEUR a tout ce
chantier, simplement rendu voyant par le rognage du cadre (la dalle occupait une petite part
d'un grand cadre vide ; elle occupe maintenant une grande part d'un cadre serre).

**Hypothese, NON confirmee** : `PoseEau` peint la BOITE ENGLOBANTE d'un volume d'eau
(`AABBMin`/`AABBMax`, sddt.go). Un volume a grande boite peint donc un grand rectangle. A
mesurer avant d'y toucher.

**Trois diagnostics successifs faux sur cette carte** — a garder ecrit, la lecon vaut plus que
les tentatives : (1) « l'ecretage a fait fuir l'eau », refute — la comparaison 30 970 -> 325 353
mettait en regard deux echelles differentes (0,0920 contre 0,029 m/px, rapport 10 exactement) ;
(2) « l'eau est hors des zones nommees, le masque la retirera », refute — 130 cellules sur
1 353 118 tombent hors zones, l'eau est DEDANS ; (3) « l'ecretage revele l'eau », refute — la
dalle est dans le fond publie, ou aucun ecretage n'a jamais tourne.

**Ce qui a tranche, et qui aurait du etre fait en premier** : ouvrir le PNG en production.

### 2026-08-26 — carte 4, Recharge

Planche : https://claude.ai/code/artifact/93a76c47-17d7-44af-8967-286abce19130

**Verbatim** : « Ok avec 4m, par contre vire l'eau stp »

`sgh_blueprint` passe `VALIDEE` avec SIX reglages : `encre`, 0,029 m/px, cadre rogne,
ecretage a **plafond 4 m**, masque des zones, **sans eau**. Matiere utile 403 x 542 ->
1 605 x 1 914 px ; largeur occupee 28,8 % (le pire des 19 natifs) -> quasi tout le cadre.

**Deux axes neufs sont nes de cette carte**, tous deux en donnee :

- `plafondArene` — le cran prevu depuis le 13/08 (« 6 -> 4 si encore trop de toits »).
  A 4 m : 2 382 213 pixels vides contre 2 043 451, et **24 ancres sur 25** contre 25/25.
  L'ancre perdue est la LIMITE : une ancre d'objectif est du terrain joue par definition.
- `sansEau` — l'eau est ECARTEE, pas corrigee. La cause garde son lot.

**Le detour qui a coute le plus cher de la journee** : trois diagnostics faux d'affilee sur la
dalle bleue (fuite d'eau due a l'ecretage / eau hors zones / ecretage qui revele l'eau), tous
refutes. Ce qui a tranche : OUVRIR LE PNG EN PRODUCTION, ou la dalle etait deja, depuis le
10/08. Verse a la recette comme piege n°2.

### 2026-08-26 — carte 5, Illusion

Planche : https://claude.ai/code/artifact/a28c6c5e-93a7-4be7-894f-02fb2ff7ca99

**Verbatim** : « ok validé, on peut passer à la suivante »

`ctf_illusion` passe `VALIDEE` : `encre`, 0,031 m/px, cadre rogne, masque des zones,
ecretage a **6 m** + **comblement des trous fermes**.

**Chemin non lineaire, consigne pour ne pas etre rejoue** : 6 m REFUSE (criblee de trous,
13/16 ancres) -> sans ecretage juge trop couvert -> 8 m -> 10 m, plus plats encore -> retour
a 6 m APRES l'ajout du comblement. Les trous reproches au premier essai sont exactement ceux
que le comblement bouche.

**Reserves assumees au verdict** : 13 ancres sur 16 (le pire des variantes) et 464 358 cellules
d'aplat suppose, soit pres de 12 % du cadre.

**DEFAUT DE CONCEPTION MIS AU JOUR PAR CETTE CARTE, non corrige** : armer l'ecretage REMPLACE
la voie de reference native au lieu de s'y ajouter — les deux liberent les memes tampons. Or
leurs regles de substitution different : la native substitue PARTOUT dans la portee des ancres,
l'ecretage seulement au-dela du plafond. Plus le plafond monte, moins il substitue — a 10 m,
108 810 cellules contre 1 338 343 pour la voie native, et l'arene perd son relief. **Sur une
carte couverte il faudrait les DEUX** ; aujourd'hui c'est l'un ou l'autre. Lot de code a part.

**Deux axes neufs nes de cette carte** : `combleTrous` (aplat de sol suppose, TROUS FERMES
seulement — la premiere version comblait tout vide du masque dilate et posait 611 959 cellules,
noyant l'arene) et `substitutionSansPortee` (retirer la limite de 25 m ; NO-OP ici, mesure
identique au bit, car les cellules gagnees tombent hors zones et sont retirees ensuite).

### 2026-08-26 — carte 9, Catalyst

**Verbatim** : « valide on passe a la suivante ? On revient a la technique classique avec
les zones de callout et tout »

**Reglage retenu** : `encre`, echelle 0,038, ecretage a 4 m, comblement des trous fermes,
masque des zones arme, et une **boite utile `[-20.5 -27.0 21.0 27.0]` DERIVEE DES
POSITIONS DE JOUEURS**.

**Ce qui a ferme la carte, deux mesures, aucun jugement** :

1. **L ecart mediane-ancres / sol tombe de -13,33 m a -0,19** (etalonnage : -0,29). Le
   defaut avait ete nomme le matin sans savoir s il expliquait quoi que ce soit : le niveau
   de jeu est un SCALAIRE, et au-dela de `PorteeNiveauDeJeu` (10 m) la teinte peint la
   surface JOUEE au recul maximal — d ou l arene sombre en production. Catalyst est mesuree
   couverte a 28,4 pour cent, SOUS le seuil d un tiers : la voie de reference native ne s y
   declenche jamais, l ecretage etait la seule voie. Meme configuration que Streets.
2. **L enveloppe jouee, sur proposition de l utilisateur.** Trois matchs decodes depuis les
   films en cache (28 disponibles pour Catalyst), environ 105 000 positions : elles tiennent
   dans 40 x 52 m quand le fond publie en faisait 69,3 x 60,3. **42 pour cent de la largeur
   n etait JAMAIS foulee** — exactement les aplats gris que l utilisateur avait repere a
   gauche et a droite avant toute mesure. 142 484 cellules retirees, cadre 1 824 -> 1 408 px.

**Portee methodologique** : c est la PREMIERE boite de la serie qui ne soit pas tracee a
l oeil (Chasm) mais mesuree, donc reproductible partout ou il y a des films — 951 en cache,
24 a 38 par carte sur toute la file restante.

**Reserves assumees** : 26 ancres sur 29, trois perdues par l ecretage a 4 m ; et la boite
est calee sur les EXTREMES des positions et non sur des quantiles — un des trois artefacts
porte des aberrations a -231 / -213 m, exactement les bornes d un canevas Forge, ecartees a
l oeil et non par un critere.

**Correctif de plomberie du jour** : `borneALaBoite` effacait la matiere mais PAS
`solSuppose` — l aplat de sol continuait donc a peindre hors boite, et le reglage
`boiteUtile` semblait sans effet. Un garde-rail ne protege que le champ qu il compte.

### 2026-08-26 — carte 10, Forbidden

Planche : https://claude.ai/code/artifact/daae2007-4ee0-41e5-aca7-c0e00bbb5d21

**Verbatim** : « ok prop 3 valide encre plafond 4m »

**Reglage retenu** : `encre`, echelle 0,045, ecretage a 4 m, comblement des trous fermes,
masque des zones arme. Cadre 1 354 x 1 550 px, 16/17 ancres.

**Le defaut n etait PAS le niveau de jeu** : ecart mediane-ancres / sol -0,21 m, deja a
l etalonnage. Le defaut reel etait le CADRE — fond du 12/08 sans rognage au cadre utile,
arene sur environ 40 pour cent de la largeur de l image.

**Le masque des zones ne pose pas de probleme ici** : 63 zones, carte franchement
ASYMETRIQUE, donc la lecon de Chasm ne s applique pas. 3,3 pour cent de matiere retiree,
zero ancre perdue. C est la premiere carte ou la regle « masque seulement si la carte en a
besoin ET n est pas symetrique » se decide par la FORME de la carte et non par tatonnement.

**Le plafond s est tranche sur piece, contre 2 et 6** : a 6 m les grandes dalles de toit
reviennent detachees en haut et en bas (12,4 pour cent hors zones) — le defaut meme de la
production ; a 2 m le dessin s aplatit.

**Reserve ecrite, NON TRAITEE** : de longues poutres sombres depassent a droite et a gauche,
comme les rails de Chasm, et elles sont DANS les zones de callout — le masque ne les atteint
pas. La coupe aux positions de joueurs n a pas pu etre tentee : `shared_matches_v2.duckdb`
est tenue en ecriture par un `server.exe` local (PID 6640 au moment du gate), et il n existe
pas de source du couple match -> carte hors de cette base (les artefacts de rejeu et le cache
des films ne portent pas le nom de carte). **Condition de reprise** : fermer le serveur local,
lister les matchs Forbidden, decoder deux ou trois films et mesurer l enveloppe jouee.

### 2026-08-26 — carte 11, Launch Site

Planche : https://claude.ai/code/artifact/3f860567-49e0-4d3d-b175-1f12589e2178

**Verbatim** : « Prop 3 ok mais vire l'eau, ensuite ce sera valide »

**Reglage retenu** : `encre`, echelle 0,055, **PAS d ecretage**, comblement des trous fermes,
masque des 52 zones, eau ecartee. Cadre 1 502 x 1 478 px, 22/28 ancres.

**LA CARTE QUI DONNE LA REGLE DES DEUX REGIMES.** L ecretage, qui a sauve les cinq cartes
precedentes, doit etre ECARTE ici : a 4 m il eventre l arene, qui devient un contour creux
(162 139 cellules de matiere contre 245 307 sans lui, et 16 ancres sur 28 contre 20). Cause
connue et deja consignee : armer l ecretage REMPLACE la voie de reference native au lieu de
s y ajouter.

| Regime | Taux de couverture | Voie native | Ecretage |
|---|---|---|---|
| Catalyst, Streets | 28,4 % — SOUS le seuil d un tiers | ne se declenche JAMAIS | seul chemin possible |
| Launch Site | 53,5 % — au-dessus | se declenche et fait le travail | ne peut que retirer du sol |

Ce n est donc pas un reglage a tatonner carte par carte : **le taux de couverture, deja
mesure et journalise, decide.** A porter dans la recette.

**Le comblement a ete arme CONTRE la prudence de premiere intention**, et c est l oracle qui
l a impose : une grande bande blanche traversait l arene en diagonale — deja presente dans le
fond de production, donc pas une regression du jour — et **huit ancres d objectif sur
vingt-huit flottaient sans sol dessous**. Une ancre d objectif etant du terrain joue par
definition, le vide n etait pas reel : c etait de la matiere manquante au rendu. Le
comblement pose environ 440 000 cellules d aplat et recupere deux ancres.

**Reserves ecrites, NON TRAITEES** : (1) six ancres restent sans sol — il subsiste des trous
ailleurs, et on ne sait toujours pas pourquoi cette portion d arene n a pas de triangles ;
(2) un aplat n est pas un releve, la surface comblee est plate ; (3) un petit fragment isole
flotte a mi-hauteur du flanc ouest, la ou l eau etait peinte.

## 2026-08-26 — LES CARTES QUI MANQUAIENT

Demande de l'utilisateur : « faut generer celles qui nous manquent ». Mesure faite sur
`shared_matches_v2` (copie de lecture — la base etait tenue en ecriture par un `server.exe`
local) croisee avec les fonds publies et le depot de variantes `.ai/re_dump/mapvar`.

**123 cartes jouees dans la base, 35 sans fond de rejeu.** Elles se repartissent en trois cas,
et un seul est un defaut de notre chaine :

| Cas | Cartes | Traitement |
|---|---|---|
| **Declarables** — `.mvar` de carte ET fichier-lien de canevas presents | 29 | DECLAREES ce jour dans `CartesForge`, canevas PROUVE par level_id (`TestPreuveLevelIDCartes` vert sur les 29) |
| **Bloquees par l'installation du jeu** | Live Fire (3 assets : 51, 18 et 2 matchs) | RIEN A FAIRE COTE CODE — voir ci-dessous |
| **Sans donnee** | Detachment, Argyle, TFF Night Of The Undead | pas de fichier-lien de canevas (Detachment, Argyle) ou pas de `.mvar` du tout |

### Live Fire : le module n'est pas installe

Question de l'utilisateur : « je ne vois toujours pas Live Fire comme carte ». La cause est
mesuree et elle n'est pas dans notre code :

```
sgh_interlock-rtx-new.module    0,21 Mo   (pc)     0,48 Mo (any)   0,21 Mo (ds)
sgh_streets-rtx-new.module    478,41 Mo   (pc)    19,36 Mo (any)   7,12 Mo (ds)
```

Le module de Live Fire est un TALON de 0,2 Mo dans les trois racines, sans `_hd1`, quand les
autres cartes pesent 90 a 1 550 Mo. `TestDiagnosticLiveFire` le confirme par l'autre bout :
`himap: aucun tag sbsp dans .../sgh_interlock-rtx-new.module`. La geometrie de la carte n'est
pas sur le disque. C'etait deja consigne le 2026-08-13 dans `cmd/mapquant-build` (« NON
CATALOGUEE malgre un module PROUVE ») sans que la cause — installation partielle — soit
nommee.

**Action cote utilisateur** : verifier l'integrite des fichiers du jeu sur Steam, ou
re-telecharger le contenu multijoueur. Aucun contournement possible : sans geometrie, un fond
serait une coordonnee devinee. Live Fire est la 6e carte la plus jouee du corpus (51 matchs,
11 sur 90 jours) — c'est le manque le plus couteux de la liste.

### Ce que le pool classe 2026 ajoute

L'agent documentaire a etabli le pool classe a jour (sources officielles Waypoint) : 15 cartes,
dont 8 en Forge. Trois n'ont JAMAIS ete jouees par les joueurs suivis (Vacancy, Serenity,
Interference) et n'ont donc ni match ni `.mvar` : rien a cuire tant qu'un match n'en ramene pas
l'asset. **Lattice** est le cas limite : une seule partie jouee, sous le nom « Lattice -
Ranked » — elle fait partie des 29 declarees ce jour.

### 2026-08-26 — CORRECTION : Corpo n'est pas validee

**Verbatim** : « Corpo est pas validee attention mais tu peux continuer a generer les cartes
manquantes et refusees »

Sa ligne portait `VALIDEE 26/08`. C'est une erreur d'attribution de ma part : Corpo etait un
PILOTE technique du lot « fonds par map_id » du 13/08 (une des deux seules cartes jouees seules
sur leur canevas, cf. `CartesForge`), et j'ai pris ce statut d'outillage pour un verdict
utilisateur. **Aucun verdict n'a jamais ete rendu sur son image.** Ligne repassee
`A FINALISER`. Elle reste dans la regeneration en encre comme les autres non closes ; son
verdict viendra a la planche.

Lecon : le seul statut `VALIDEE` legitime est celui qui porte un verbatim dans ce journal. Une
ligne validee sans verbatim est a re-verifier, pas a croire.

## 2026-08-27 — CORRECTION : 28 des 29 cartes declarees N'ETAIENT PAS CUISINABLES

J'avais annonce hier soir « 29 cartes declarables, declarees ». **C'etait faux : une seule
l'etait** (Solitude - Ranked). Les 28 autres ont echoue a la cuisson sur
`replay: carte absente du catalogue d'objectifs`.

**La condition que j'avais oubliee.** Pour cuire un fond Forge il faut TROIS choses, et je
n'en avais verifie que deux :

| Condition | Ou elle se lit | Verifiee hier ? |
|---|---|---|
| le `.mvar` de la carte | `.ai/re_dump/mapvar` | oui |
| le canevas | fichier-lien + preuve level_id | oui |
| **les ancres d'objectif** | **`data/.../map_objectives.json`** | **NON** |

Le CADRE d'un fond est construit sur les ancres (`CadreSurAncresEchelle`) : sans ancre, il n'y
a meme pas d'image a rendre. La verification etait a une requete de distance et je ne l'ai pas
faite — j'ai declare sur la foi de deux conditions sur trois, puis annonce le resultat comme
acquis avant qu'une seule carte ait ete cuite.

Les 28 declarations ont ete RETIREES de `CartesForge` (une declaration sans fond publie casse
le garde-rail `TestFondForgeJamaisSousCleModule`, et c'est tres bien ainsi : c'est lui qui
aurait attrape la faute si je l'avais joue avant d'annoncer).

**Le vrai prochain lot pour ces cartes** : les faire entrer dans le catalogue d'objectifs
(`cmd/mapobj-build`), PUIS les declarer. Elles sont ici :

| carte | map_id | matchs | canevas |
|---|---|---|---|
| Insolence | `d5c5eb4f` | 7 | fo09_academy |
| Flood Gulch | `7097bc4f` | 6 | fo05_desert |
| 944396dd-5661-4a16-b1d8-a6053f762c55 | `944396dd` | 1 | fo13_frost |
| Ecotone | `8816f240` | 8 | fo11_blank |
| Solution | `ee43d273` | 7 | fo05_desert |
| Threshold | `ddbb3a00` | 5 | fo11_blank |
| Fortitude Heavies | `305b1bdd` | 4 | fo05_desert |
| Thunderhead Heavies | `37bc3df6` | 4 | fo08_wetland |
| Thunderhead | `28a3ac28` | 4 | fo08_wetland |
| Pharaoh | `88d45250` | 3 | fo11_blank |
| Obituary Heavies | `e3681516` | 3 | fo09_academy |
| Merchant's Square | `7dfec55d` | 2 | fo09_academy |
| Credence | `0cc728d2` | 2 | fo11_blank |
| Vallaheim Firefight | `e8268e75` | 2 | fo05_desert |
| Urban Raid | `be848f91` | 2 | fo09_academy |
| Disciple | `525451ca` | 2 | fo11_blank |
| Ronin | `f459867d` | 1 | fo08_wetland |
| Origin - Ranked | `46a8319c` | 1 | fo08_wetland |
| Nadair | `6dbd1c0d` | 1 | fo11_blank |
| Cole Protocol | `571afb7f` | 1 | fo09_academy |
| Outlook | `ea7b30e6` | 1 | fo13_frost |
| Refuge Heavies | `c10c7e79` | 1 | fo08_wetland |
| Lattice - Ranked | `1a6cfc2e` | 1 | fo13_frost |
| Rat's Nest | `133c0185` | 1 | fo08_wetland |
| Insolence Heavies | `2a339c65` | 1 | fo09_academy |
| Scarlett's Landing | `79042fc0` | 1 | fo08_wetland |
| Warehouse | `5b12d6d9` | 1 | fo11_blank |
| Dawnbreaker | `89dd4003` | 1 | fo05_desert |
### L'incident de la nuit : une boucle chaude de sept heures

De 01 h 00 a 08 h 17, le script a rejoue le MEME lot en boucle — 135 000 lignes d'erreur,
32 Mo de journal, aucune carte produite. Cause : mon filet anti-boucle n'ecartait un lot que
si le binaire sortait avec le code 0. Or `mapfond-build` sort en ERREUR quand une carte
echoue. Le filet ne se declenchait donc JAMAIS sur le seul cas qu'il devait couvrir — une
carte definitivement incuisable — et ne couvrait que le cas ou le binaire reussit sans rien
faire, qui n'arrive pas.

Le garde-fou « ne pas classer en echec un process tue de l'exterieur », ajoute pour eviter une
regression, a donc CREE la boucle infinie. Correctif : un compteur de tentatives par carte —
trois passages sans production et la carte sort de la file, quel que soit le code de sortie.

**Bilan reel de la campagne** : 38 fonds Forge cuisinables, 38 cuits. Les 28 autres attendent
le catalogue d'objectifs.

## 2026-08-27 — 27 cartes entrent, et Live Fire cesse d'etre un mystere

**Les 27** : `cmd/mapobj-build --from-file` a ingere leurs objectifs HORS LIGNE depuis leur
`.mvar` (aucun appel reseau, aucune authentification), de 4 a 50 objectifs par carte. Elles
ont ensuite ete declarees dans `CartesForge` — canevas prouve par level_id — et cuites : 27
sur 27, zero echec. **Cole Protocol reste dehors** : son `.mvar` ne porte AUCUN objectif, donc
aucune ancre, donc aucun cadre a construire.

Le catalogue de fonds passe de 57 a **84**.

### Live Fire : l'utilisateur avait raison, et la geometrie est trouvee

**Verbatim** : « Live Fire j'y joue regulierement donc c'est bizarre, ce doit etre une variante
d'une autre map et le poids leger doit etre la diff »

Ma conclusion « geometrie non installee » venait du POIDS du fichier, pas de son contenu.
Mesure du contenu :

| module | fichiers | groupes de tags |
|---|---|---|
| `sgh_interlock` (Live Fire) | **6** | `levl=1` + 5 sans groupe — aucun sbsp, aucune bitmap |
| `sgh_blueprint` (Recharge) | 1 976 | bitm=392, rtgo=169, sbsp=1 |
| `sgh_streets` (Streets) | 8 949 | bitm=1 051, rtgo=1 049, sbsp=2 |

Le `levl` de Live Fire pese **2,3 Mo — plus que celui de Recharge (664 Ko)**. Ce n'est donc pas
un talon vide : c'est le DELTA, exactement ce que disait l'utilisateur.

**Ou est la geometrie** : `common-rtx-new.module` porte QUATRE `sbsp` qu'aucune carte ne
reclame. Le premier — 12 556 instances, X [-16,7 ; +46,5], Y [-10,1 ; +53,7] — **contient les
24 ancres d'objectif de Live Fire**. Preuve rejouee par `TestGeometrieLiveFireDansCommon`.

**Ce qui reste a faire** : la premiere cuisson par ce chemin rend un decor qui n'est PAS
l'arene (antenne satellite, escalier, vegetation ; 21/28 ancres au sol, couverture 22 %). Le
bon bsp reste a designer parmi les quatre — `ChoisitBSP` retient celui qui contient le plus
d'ancres, et deux des quatre les contiennent toutes. Le fond produit a ete RETIRE et le
reglage `moduleGeometrie` desarme : un mauvais fond vaut moins que pas de fond. Le levier de
code reste en place, il est correct ; c'est son parametre qui n'est pas trouve.

**Condition de reprise** : departager les quatre bsp de `common-rtx-new` (par la surface, par
le nombre d'ancres AVEC SOL, ou en lisant les references du `levl`), puis rearmer le reglage.

### 2026-08-27 — Isolation : la chaine Forge n'avait AUCUN levier

Planche : https://claude.ai/code/artifact/a4034bb7-ee86-46b7-8043-63515e721bc9

**Verbatim** : « isolation pas bon du tout, y a un gribouillis, fais le traitement quand on a
fait avec l'ecretage et zone de callout stp »

**Ce que la demande a mis au jour** : le traitement demande N'EXISTAIT PAS pour cette carte, et
pas seulement parce qu'une carte Forge n'a aucune zone de callout (les 22 cartes qui en portent
sont toutes natives). **La chaine Forge ne recevait AUCUN des leviers de la chaine native** —
`EcreteToits`, `BoiteUtile`, `RogneAuxZones`, `CombleTrous` ne vivaient que dans `cuisson.go`.
Un reglage declare pour une carte Forge etait donc silencieusement ignore. Trois cuissons
d'Isolation a trois plafonds ont rendu le MEME octet : c'est ce qui l'a prouve.

**Corrige** : les leviers passent aux cartes Forge, et l'equivalent du masque de callouts y est
trouve — **les volumes de mort**. Les callouts disent ou l'on joue ; les volumes de mort disent
ou l'on MEURT. La cuisson les reconnait depuis le 10/08 (`TypesVolumesDeMort`, empreinte etablie
sur 101 `.mvar`) mais ne s'en servait que pour les ECARTER du dessin — leur position n'avait
jamais servi. `BoiteDesVolumesDeMort` en tire l'emprise ; garde-rail `TestBoiteDesVolumesDeMort`.

Sur Isolation : 6 volumes, boite [-86,9 -118,4 38,2 41,5]. Le disque du canevas Forge et ses
dalles de ciel disparaissent, le cadre passe de **2 628 a 1 727 px**, couverture de 93,9 a
36,4 %, et **25 ancres sur 25** restent au sol.

**CE QUI RESISTE, et la carte n'est PAS close** : le gribouillis. L'ecretage tourne desormais
pour de bon, mais a 4 m comme a 2 comme a 1 m il n'y touche pas. Ce n'est donc pas un toit
au-dessus du sol joue : c'est de la matiere A HAUTEUR DE SOL — vraisemblablement des modeles de
vegetation ou de lianes poses en centaines d'exemplaires, dont les traits balayes en longues
courbes signent des maillages tres etendus. **Une coupe par altitude ne peut rien contre cela
par construction.**

**Condition de reprise** : mesurer, type par type, l'emprise des modeles poses dans la
variante ; les types dont l'emprise depasse largement celle d'un objet de decor sont les
fautifs ; les ecarter par un reglage `typesExclus`. C'est court, et c'est la seule voie qui
attaque la cause.

### 2026-08-27 — Isolation : quatre leviers essayes, le gribouillis est le SOL

**Verbatim** : « bah c'est immonde pour isolation, tu me proposes quoi pour corriger ca ? C'est
inexploitable. Et en jeu je te confirme qu'on a des zones de callout pour absolument toutes les
cartes, y compris Forge »

**Quatre leviers essayes, mesures, tous negatifs sur le gribouillis** :

| levier | reglage | resultat |
|---|---|---|
| ecretage des toits | `plafondArene` 4, 2, 1 m | cadre reduit, gribouillis INTACT |
| bornage aux volumes de mort | `rogneAuxVolumesDeMort` | boite [-86,9 -118,4 38,2 41,5] plus GRANDE que le cadre : inerte ici |
| substitution sans portee | `substitutionSansPortee` | aucun effet visible |
| **tranche plafonnee** (nouveau) | `plafondTranche` +6 m, +3 m | gribouillis INTACT |
| exclusion des 10 types les plus etendus | `typesExclus` | quelques balayages en moins, gribouillis toujours la |

**Ce que ces cinq mesures etablissent ensemble** : le gribouillis n'est ni un toit, ni du
decor hors zone, ni de la matiere haute. Il vit A MOINS DE TROIS METRES du niveau de jeu —
c'est-a-dire AU SOL. Aucune coupe geometrique ne peut l'en separer : c'est la geometrie de
l'arene elle-meme, faite de centaines de pieces organiques qui se chevauchent, vue de dessus.

**Correction d'une affirmation trop rapide de ma part** : j'avais ecrit que le bornage aux
volumes de mort faisait passer le cadre de 2 628 a 1 727 px. C'etait FAUX — c'est l'ecretage
qui reduisait le cadre. Les six volumes de mort d'Isolation bornent une region PLUS GRANDE que
le cadre des ancres : le levier est correct et teste, mais il est sans effet sur cette carte.

**A INSTRUIRE, information utilisateur** : « en jeu on a des zones de callout pour absolument
toutes les cartes, y compris Forge ». Notre `map_callouts.json` n'en porte que 22, toutes
natives — mais leur provenance est `decoupe`, c'est-a-dire que NOUS les avons derivees. Le jeu,
lui, en a pour toutes. **Ou vivent-elles pour une carte Forge** est une question ouverte et
c'est probablement la meilleure piste restante, pour Isolation comme pour les 40 autres Forge.

**Isolation reste refusee**, son entree de reglage est retiree : rien ne doit publier un fond
qu'on sait mauvais.

### 2026-08-27 — « Et si on avait la mauvaise approche ? » — la base existe, mais elle est DESSOUS

**Verbatim** : « Les cartes Forge ont une base sans doute, sur laquelle dessiner, et nous on ne
doit avoir que cette base. Ou inversement. »

**L'hypothese etait juste sur un point que l'etat de l'art niait.** `cartes_forge.go` affirmait
qu'un canevas ne porte AUCUNE instance de geometrie — c'est meme ce qui avait servi a
identifier Corpo. Mesure du jour :

| canevas | fichiers | sbsp | instances du bsp lointain | instances du bsp d ile |
|---|---|---|---|---|
| `fo11_blank` | 15 | 2 | **0** | **0** |
| `fo08_wetland` | 9 866 | 2 | **13 281** | **814** |

L'affirmation n'etait vraie que pour le canevas VIERGE. `fo08_wetland` porte un terrain complet
que la cuisson Forge n'a jamais dessine — une carte batie SUR ce terrain etait donc rendue sans
son sol.

**Levier livre** : `dessineCanevas`, qui pose la geometrie du canevas sous les objets de la
variante (bsp choisi par les ancres, comme la chaine native ; best-effort declare).

**Ce que l'essai sur Isolation etablit** : avec le canevas arme, 2 169 instances ont ete
dessinees — et le PNG produit est **identique a l'octet**. Ses ancres vivent entre Z +112,6 et
+121,5, tres au-dessus du terrain : **la carte est batie AU-DESSUS de sa base, pas dessus**. Un
rendu vu de dessus ne verra donc jamais le canevas, qui est sous l'arene. « Ne garder que la
base » rendrait le marecage, pas la carte.

Le levier reste : il servira aux cartes Forge reellement posees sur leur terrain, et le test
`TestCanevasForgePorteTIlDuTerrain` garde la mesure.

**Le gribouillis reste donc entier**, et sa cause se resserre : ce n'est ni un toit, ni le
canevas, ni du decor hors zone — ce sont les PIECES FORGE DE L'ARENE ELLE-MEME, a hauteur de
sol. Le seul angle restant est l'exclusion par type, avec un critere mesure sur la
CONTRIBUTION EN PIXELS et non sur l'emprise du modele.

### 2026-08-27 — LE GRIBOUILLIS EST UN HABILLAGE, PAS UNE GEOMETRIE

Planche : https://claude.ai/code/artifact/021c9a57-0208-4c2b-be09-ea0e798c2d2c

**Verbatim** : « ce qui m'intrigue c'est que j'ai l'impression qu'il y a des formes sous ce
gribouillis. Et non on ne peut pas passer a autre chose, de trop nombreuses cartes ont le meme
souci. »

**L'intuition etait juste, et la preuve tient en une image** : le MEME rendu, la meme
geometrie, le meme cadre, en habillage `altitude` — le gribouillis disparait et l'arene se
lit. Rien n'a ete retire ; seule la mise en couleur a change.

**La cause.** L'habillage `encre` tient sa lisibilite de deux mecanismes qui supposent tous
deux des surfaces PLANES se rencontrant franchement :

1. les **aretes** — on souligne un bord des que deux pixels voisins different de plus de
   `SeuilAreteMetres` = 0,5 m ;
2. les **aplats** — l'eclairement est quantifie en paliers.

Sur une carte faite de pieces ORGANIQUES qui se chevauchent (les remakes Forge en rochers),
deux voisins different de quelques centimetres PARTOUT : le predicat d'arete est vrai presque
partout, et les paliers decoupent les surfaces courbes en bandes. `altitude` ne trace aucune
arete et ne quantifie rien — d'ou le contraste.

**C'est pourquoi cinq coupes geometriques n'avaient rien donne** : ecretage a 4/2/1 m, tranche
plafonnee a +6/+3 m, bornage aux volumes de mort, substitution sans portee, exclusion des dix
types les plus etendus. Je retirais de la matiere qui n'avait rien fait. **Lecon : avant de
retirer de la matiere, verifier que le defaut est bien dans la matiere** — le meme rendu dans
un autre habillage repond en une cuisson.

**Levier livre** : `seuilArete` par carte (garde-rail `TestSeuilAreteParCarte`). Demi-remede
seulement : a 2 et 5 m les grandes dalles perdent leurs traits parasites, mais les APLATS
restent. Le remede complet est le choix d'habillage par carte, ou des aplats continus sur ces
cartes-la.

**A JUGER par l'utilisateur** : habillage `altitude` sur les cartes organiques (le plus
lisible, mais quitte l'encre validee sur onze cartes), ou encre + seuil releve (garde
l'identite, ne traite que la moitie du defaut).

### 2026-08-27 — Callouts Forge : la proposition, DESSINEE

Planche : https://claude.ai/code/artifact/b37392c7-f6b0-4fd7-a224-950831ce5c1f

**Verbatim** : « Mais ta proposition tu me l'as dessinee ou pas ? Je comprends rien »

Le fichier Forge d'Isolation porte **237 objets qui declarent une FORME**, repartis en 26
types. Trois ont la taille et le nombre d'une zone de callout :

| type | nombre | taille mediane | emprise |
|---|---|---|---|
| `-696190206` | 18 | **14,9 m** | X [-43 ; -9] Y [-38 ; -7] |
| `-722308271` | 17 | **12,0 m** | X [-49 ; -16] Y [-30 ; -15] |
| `1223404046` | 25 | 5,0 m | etiquetes `firefight_include` — temoin |

Les ancres d'objectif de la carte vivent en X [-40,3 ; -3,7] Y [-53,7 ; -10,8] : les deux
premiers candidats couvrent bien l'arene. Les formes sont dessinees a leur position et a leur
taille reelles, converties par le calage publie du fond — pas d'estimation.

**Critere de verdict** : si l'un des deux epouse les salles de l'arene, on tient la source des
callouts Forge, hors ligne, pour les 40 cartes.

### 2026-08-27 — Isolation : trente rendus, et la recette du 10/08 mise a l'epreuve

Planche : https://claude.ai/code/artifact/8cab9a4b-0a53-436a-a9c2-e5fccd952a97

**Verbatim** : « Forbidden la est extremement differente et toujours valide, je me demande si la
recette utilisee ne serait pas meilleure pour l'anti-gribouillis »

**La recette du 10/08, retrouvee sur pieces** (sidecar de `ctf_forbidden`, commit `9b8f6cca3`) :
habillage `jeu`, **0,0920 m/px**, aucun levier. Contre aujourd'hui : `encre`, 0,0450, ecretage
4 m, masque des 63 zones.

**Appliquee a Isolation (vignette 28) : la bouillie reste**, un peu moins criante parce que le
pixel est deux fois plus gros — c'est exactement ce qui etait publie le 13/08. **La recette
n'est donc pas ce qui sauve Forbidden : c'est sa GEOMETRIE.** Forbidden est faite de dalles
planes qui se rencontrent franchement ; Isolation, de coques organiques qui se chevauchent.

**Ce que l'echelle fait quand meme** — comparer 01 contre 21, ou 10 contre 23 : a habillage
constant, le pixel de production adoucit le gribouillis sans le supprimer. **L'echelle
automatique introduite le 26/08 l'a donc rendu PLUS visible sur les cartes organiques.** Effet
reel, mesurable, et signale par l'utilisateur avant moi.

**Deux hypotheses de l'utilisateur, testees et ecartees** :

| hypothese | mesure | verdict |
|---|---|---|
| un parametre mal interprete (repere de l'objet) | `\|up\|` et `\|forward\|` = 1,0000 sur les 5 042 objets ; produit scalaire 0,0000 | repere lu correctement |
| maillages mal decodes (indices) | rapport arete mediane / diagonale : **mediane 0,026** sur 260 modeles | maillages sains |

**Reste a mesurer** : l'ECHELLE des objets Forge. `InstanceForge` force `Scale = {1,1,1}` ; si
Forge encode un redimensionnement dans le sac de proprietes — la ou vivent deja les formes —
on dessine tout au mauvais gabarit.

**Prochain angle de rendu si aucune vignette ne convient** : ne plus retenir la surface la plus
HAUTE par pixel mais la plus proche du sol joue, partout. C'est un mode de rendu, pas un
reglage.

### 2026-08-27 — « QUI PEINT L'IMAGE » : la mesure qui manquait

**Verbatim** : « Aucune ne convient malheureusement » (sur les 30 rendus)

**Le probleme etait de methode.** Trente rendus, cinq coupes geometriques et trois criteres
portes par le modele ont echoue pour une raison commune : ils decrivent ce qu'un objet EST,
aucun ne dit ce qu'il PEINT. Le rendu retient desormais, pour chaque pixel, le TYPE qui a
gagne le z-buffer, et la cuisson journalise le classement.

**Sur Isolation, une ligne de log a repondu a une journee de tatonnements** :

```
typesVisibles=46 sur 292   pixels=3 459 674
-1342618612 : 82,7 %  avec 32 exemplaires
 1574763282 :  5,2 %  avec  3 exemplaires
```

**Un type, 32 exemplaires, peint 83 pour cent de l'image.** Et le type que j'avais designe
comme coupable — les 349 branches, identifiees en ne dessinant qu'elles — **n'occupe AUCUN
pixel** : les exclure ne changeait pas un octet du fichier, ce que j'ai verifie.

**Trois criteres automatiques essayes et refutes** pour separer ces coques du reste :

| critere | ce qu il donne sur le type coupable |
|---|---|
| emprise du modele | attrape les gros rochers legitimes |
| aire du maillage / emprise au carre | rang 222 sur 271, parmi les plus PLEINS |
| part de l emprise au sol couverte | 0,499, rang 145 sur 270 |

**Ce que le pelage montre** : retirer le premier type decouvre le deuxieme (45,6 % avec
3 exemplaires), retirer les quatre premiers decouvre le cinquieme (40,7 % avec 117). Isolation
est un EMPILEMENT DE COQUES organiques au-dessus de son arene. Chaque pelage est une cuisson
de 90 secondes et se juge a l'oeil.

**Etat** : Isolation reste refusee, son entree de reglage retiree. L'outil, lui, est acquis et
vaut pour toutes les cartes organiques.

### 2026-08-27 — Le lot des dix-huit cartes sans bouillie

Planche : https://claude.ai/code/artifact/90777230-9715-45fe-8190-0f617f29d36c

**Verbatim** : « j'ai releve a l'oeil le reste des maps sans gribouillis (et tu verras qu'il n'y
en a pas beaucoup qui peuvent etre traitees tant qu'on a pas regle ce souci de bouillies) »

Les dix-huit sont toutes declarees et cuites. **Toutes gardent 100 pour cent de leurs ancres au
sol.** La passe de mesure a releve, pour chacune, ce qui decide des reglages plutot que de les
choisir a l'oeil.

**Le taux de couverture, qui decide de l'ecretage** (regle des deux regimes du 26/08) :

| sous 1/3 — la voie native ne s'y declenche jamais | au-dessus — elle fait deja le travail |
|---|---|
| Domicile 15,7 · Warehouse 17,9 · Lattice-Ranked 23,1 · The Pit 25,0 · Empyrean 26,9 · Ronin 27,0 · Rat's Nest 27,6 · Disciple 29,4 · Outlook 30,1 · Banished Narrows 31,6 | Starboard 36,4 · Dredge 40,3 · Urban Raid 43,4 · Corpo 45,8 · Scarlett's Landing 46,0 · Solution 49,7 · 944396dd 53,2 · Goliath 54,7 |

**Le bornage aux volumes de mort gagne sur CINQ**, sans couter une ancre :

| carte | avant | apres | surface |
|---|---|---|---|
| The Pit | 2551x1894 | 1792x1718 | **-36 %** |
| Warehouse | 2207x2363 | 1818x2103 | **-27 %** |
| Goliath | 1600x2210 | 1600x1640 | **-26 %** |
| Ronin | 2011x3001 | 2011x2611 | **-13 %** |
| Rat's Nest | 1892x2520 | 1892x2264 | **-10 %** |

Sur les treize autres il est **inerte** : la boite que forment leurs volumes est plus grande
que le cadre des ancres. Leur entree de reglage a ete RETIREE plutot que gardee sans effet —
un reglage qui ne fait rien est une dette, pas une precaution.

**Reste ouvert** : le vrai nom de `944396dd-5661-4a16-b1d8-a6053f762c55` (un agent le cherche),
et les cartes a bouillie, qui attendent le resultat des enquetes sur Reclaimer.

### 2026-08-27 — « Quelle est la difference ? » : trois mesures, trois refutations

**Verbatim** : « On a des maps Forge qui sont nickels et pas d'autres, quelle est la
difference ? »

Bonne question, et j'ai voulu y repondre par un chiffre plutot que par une impression. Trois
candidats mesures sur les 84 fonds publies (`cmd/mapfond-cadrage`, deux colonnes nouvelles) :

| mesure | tes 18 cartes propres | Isolation, la bouillie averee | verdict |
|---|---|---|---|
| **bruit** (ruptures de luminance entre voisins) | 2,0 a 28,3 % (mediane 12,0) | **11,1 %** | REFUTE : Isolation tombe sur la MEDIANE des propres |
| **alignement des contours** (part des bords suivant un axe) | 61,9 a 94,7 % (mediane 79,1) | **62,2 %** | REFUTE : Isolation est a 0,3 point du plancher des propres |
| **taux de couverture** et **densite d'objets** | mediane 31,6 % et 863 obj/Mpx | 36,4 % et 850 | REFUTE : indiscernables |

**Aucune statistique globale de l'image ne separe les deux familles.** Le bruit monte surtout
sur les cartes NATIVES tres detaillees mais parfaitement lisibles — Cliffhanger 43 %, Highpower
38,9, Forest 38,4 — ce qui montre qu'il mesure la richesse, pas le defaut.

**Ce qui reste, et qui colle a tout ce qu'on a vu** : la difference n'est pas dans la quantite
de detail mais dans le ROLE du type dominant. Sur une carte propre, celui qui peint le plus est
un SOL plat ; sur Isolation, c'est une COQUE qui enferme l'arene (82,7 % des pixels avec
32 exemplaires, et sous elle une deuxieme couche, puis une troisieme). La mesure qui le dirait
est la VARIANCE DE LA NORMALE du type dominant : un sol a toutes ses normales vers le haut,
une coque les a dans toutes les directions. Le rendu porte deja les normales et sait deja quel
type gagne chaque pixel — c'est une demi-heure.

**Sous-produit utile** : le classement par alignement designe les cartes Forge les plus
desalignees, donc les plus suspectes, que l'utilisateur n'a pas encore jugees — Elevation 55,2 ·
Nemesis 57,2 · Opulence 58,0 · Absolution 58,7 · Thunderhead 58,7 · Flood Gulch 58,8 ·
Snowbound 59,5 · High Ground 60,0 · Fortitude 60,4 · Threshold 60,9.

### 2026-08-27 — LE DOME EST MARQUE DANS LE FICHIER : le drapeau d'objet Forge

Planche : https://claude.ai/code/artifact/8a978a60-10ee-44bc-99dc-de939fb2a6a0

**Verbatim** : « Ce ne serait pas des effets de lumiere ces trucs qui nous posent probleme ? Ce
doit etre des elements qui ne sont pas des blocs [...] peut-etre un genre de mur transparent ? »

**L'intuition etait juste : ces objets sont marques differemment dans le `.mvar`.** Le champ 7
de chaque objet (`Object.Flags`, un octet) etait lu par le parseur et n'avait JAMAIS servi.
Mesure sur les 5 042 objets d'Isolation :

| drapeau | objets | dont |
|---|---|---|
| **21** | 4 384 | la structure ordinaire, dont le 2e peintre (3 pieces de 65 m) |
| **1** | 344 | **les 32 pieces du dome qui peignent 82,7 % de l'image** |
| 7 / 16 / 23 / 30 / 24 / 4 | 88 / 75 / 40 / 36 / 19 / 18 | — |

Categories (champ 1 du sac gameplay) : 2 pour 4 697 objets, 3 pour 289, -1 pour 55, 1 pour 1.
Le dome est en categorie 2, la majoritaire : **la categorie ne le separe pas, le drapeau si.**

**Ce que la mesure geometrique avait deja etabli** : les 32 pieces sont posees entre Z 136,0 et
160,6 quand le sol joue est a Z 117 — de 19 a 44 m au-dessus de l'arene, toutes Up vers le
haut, la plus petite distance entre deux d'entre elles etant de 29 cm. C'est une voute, ajustee
bord a bord. Et sa PAROI DESCEND JUSQU'AU SOL : c'est pourquoi aucune coupe de SURFACE ne l'a
jamais enlevee — ecretage a 4, 2, 1 m, tranche plafonnee a +3, +6, +12, bornage.

**Trois leviers livres, tous mesures sur Isolation** :

| levier | ce qu'il fait | resultat |
|---|---|---|
| `plafondObjets` | ecarte un objet par l altitude ou il est POSE, non par celle de ses surfaces | 221 objets retires, dome parti, cadre 2 628 -> 1 727 px, 25/25 ancres |
| `drapeauxExclus` | ecarte un objet par son champ de drapeaux | 355 objets retires, dome parti, cadre -> 1 873 px, 25/25 ancres |
| `solVuDuDessous` | retient la surface la plus BASSE au-dessus du sol joue au lieu de la plus haute | image nettement plus lisse, 701 -> 515 Ko, mais la jupe du dome gagne encore |

**CE QUI RESTE, ET C'EST HONNETE** : le dome parti, le peintre suivant est `1574763282` — TROIS
exemplaires de 65 m, drapeau 21, categorie 3 — qui peint 46 % a lui seul. Isolation est un
empilement, et le drapeau ne designe que la couche exterieure. Prochain essai : la categorie 3
(289 objets), a laquelle appartient ce second peintre.

### 2026-08-27 — ISOLATION EST LISIBLE : le fond vient du MAILLAGE DE NAVIGATION

Planche : https://claude.ai/code/artifact/5fbb56c8-f7e8-4f3b-8f6d-62b0d79e147a

**Verbatim** : « Ah oui Isolation je commence a reconnaitre !! Il y a des elements manquants
encore mais ca devient reconnaissable par endroits ! »

**La sortie n'etait pas de mieux soustraire mais de CHANGER DE SOURCE.** Chaque carte Forge
publie un `navmesh.blob` a cote de sa variante. Decodage etabli et teste (`internal/hinavmesh`) :

| couche | ce que c'est |
|---|---|
| 12 octets | en-tete gros-boutiste : version 2, taille-12, une constante 0x001FFFFF |
| conteneur | **le MEME que les `.mvar`** — `cb2.go` le decode sans une ligne nouvelle |
| champ 1 | flux **zlib a fenetre de 8 Ko** : en-tete `58 09`, ce qui l'a rendu invisible aux recherches de `78 9c` |
| 1 128 372 o | 5 regions, dont **4 tagfiles Havok 2022.1.0** |
| region 1 | **`hkaiNavMesh`** : 2 348 faces, 8 218 aretes, 3 350 sommets, 2 200 m2 |

**POURQUOI LE PROBLEME DISPARAIT AU LIEU D'ETRE RESOLU** : le maillage vit entre Z 112,54 et
124,08 ; la premiere couche de coques est posee entre Z 136 et 160 — **onze metres plus haut**.
Le navmesh ne contient pas les coques. Rien a peler, rien a ecreter, rien a borner.

**L'ORACLE, PASSE AVANT L'IMAGE** : 24 des 25 ancres d'objectif tombent DANS un polygone, ecart
d'altitude median **7,4 cm**. La 25e (assault_bomb) est a 2,03 m du bord — le navmesh se retire
le long des murs. Verifie sur Kiken'na : **13/13**, dans un repere tout autre (X -187..-155,
Z 172..179), ce qui exclut tout codage en dur.

**COUVERTURE** : le navmesh n'existe QUE pour les cartes Forge, et seulement au-dela d'environ
1 000 objets — present sur 10 cartes testees sur 10 dans cette bande, absent (404) sur 13 sur 13
en dessous. Sur les 101 cartes du referentiel, 66 sont dans la bande favorable.

**RESTE, ET C'EST DU CADRAGE** : un ilot du maillage hors de l'arene etire le cadre (le grand
polygone en bas a droite). Le bornage aux volumes de mort, ou garder la composante connexe qui
porte les ancres, le retirera. Et l'utilisateur signale des **elements manquants** : le navmesh
ne porte que le sol MARCHABLE — ni les murs, ni les structures. La suite naturelle est d'en
faire la SURFACE DE REFERENCE de la chaine ordinaire : le rendu habituel, ramene a l'altitude du
navmesh, rendrait le sol ET les structures sans le dome.

### 2026-08-27 — ISOLATION EST CLOSE, par le maillage de navigation

Planche : https://claude.ai/code/artifact/1880d73c-5942-45bd-9a0e-baab5a423454

**Verbatim** : « Alors la je reconnais carrement !!! » puis « La 4 a 1,5 m est nickel ».

**Reglage retenu** : `encre`, `navmeshReference`, `rogneAuNavmesh`, `toleranceNavmesh` 1,5 m.
Cadre 1 638 x 1 368, **25 ancres sur 25 au sol**.

| levier | mesure |
|---|---|
| reference prise sur le maillage | **845 552 cellules** recoivent l'altitude REELLE du sol |
| rognage au maillage | **2 175 499 cellules** effacees, dilatation 3 m |
| tolerance au sol 1,5 m | 121 976 cellules vidées |

**Ce que deux jours d'echecs ont appris** : le probleme n'etait pas ce qu'il fallait RETIRER
mais ce a quoi on COMPARAIT. La reference etait interpolee depuis 25 ancres ; le dome vit onze
metres au-dessus du sol et gagnait donc partout. Avec une reference qui EST le sol, il ne gagne
plus un pixel — sans qu'on le touche.

**LA QUESTION DES ZONES DE CALLOUT, VERIFIEE** : `map_callouts.json` porte 22 cartes, **toutes
natives, zero Forge**, et Isolation n'y est pas. Le decoupage demande n'a donc PAS pu se faire
par les callouts — il s'est fait par le maillage de navigation, qui dit la meme chose et mieux :
il EST la zone jouable, au polygone pres, sans qu'on ait rien a dessiner a la main.

**Piste ouverte pour les vrais callouts Forge** : le blob porte QUATRE tagfiles Havok, dont un
`hkaiTraversalAnnotationLibrary` que nous n'ouvrons pas. Une bibliotheque d'annotations de
traversee est l'endroit ou des noms de lieux vivraient. A instruire.

### 2026-08-27 — Isolation : la marge restante, mesuree

Planche : https://claude.ai/code/artifact/d4cd0358-f907-499e-ba6d-feb156853fdd

**Verbatim** : « il y a encore des formes un peu gribouillis et pas aplaties, tu peux checker si
on a un peu de marge de manoeuvre la-dessus ? Autrement je la valide a ce stade, disons en
beta ! »

**Statut : VALIDEE BETA.**

| levier | de -> a | cellules vidées | effet visible |
|---|---|---|---|
| tolerance au sol | 1,5 -> 1,0 -> 0,6 m | 121 976 -> 137 302 -> 173 434 | **quasi nul** — 0,6 % de l'image |
| seuil d'arete | 0,5 -> 2 m | — | **leger mais reel** : les traits parasites autour des structures se taisent |

**Ce que ca etablit** : les formes qui restent sont A HAUTEUR DE SOL — ce sont les pieces de
l'arene elles-memes, qui se chevauchent a quelques centimetres. Aucune coupe verticale ne les
separera du sol puisqu'elles SONT le sol. Les aplatir demanderait de ne plus dessiner la
geometrie mais le seul maillage, lisse par construction — au prix des structures.

**Il n'y a pas de troisieme levier cache.**

### 2026-08-27 — LES CARTES FORGE ONT DES CALLOUTS (l'utilisateur avait raison)

**Verbatim** : « T'es sur pour les callouts ? En jeu je les vois, c'est peut-etre un fichier non
telecharge aussi ? »

**Il avait raison, et aucun fichier ne manquait.** Les zones nommees d'une carte Forge sont dans
son `map.mvar` — celui que la cuisson lit deja. Chaque zone est un objet de type
**-696190206** portant son StringId de lieu au chemin `#8/4[]/0/0`, qui se resout contre le
tableau de 778 entrees du tag global `locs` : **le meme vocabulaire que les cartes natives**.

**Isolation : 18 zones, 18/18 resolues** — *bottom mid*, *cave* (x4), *top mid*, *north base*,
*south base*, *pipes* (x2), et 8 lieux que `callouts_i18n.csv` ne nomme pas encore.

**L'erreur n'etait pas une mesure fausse mais une GENERALISATION fausse** : le zero mesure porte
sur les CANEVAS (leurs 12 volumes anonymes sont des barrieres) ; nos en-tetes en avaient conclu
le zero des CARTES. Un balayage d'entiers LE32 rend zero — les entiers Bond sont des varint
zigzag — ce qui confirmait l'absence a qui cherchait naivement. Trois en-tetes corriges le meme
jour (`callouts_catalog.go`, `replay_map_callouts.go`, `himap/callouts.go`).

**Mesure du rognage sur Isolation** (1 162 199 cellules de matiere) :

| marge | hors zones | part | ancres au sol |
|---|---|---|---|
| 1 m | 306 128 | 26,3 % | 24/25 |
| 4 m | 198 738 | 17,1 % | 24/25 |
| 8 m | 117 889 | 10,1 % | 24/25 |

**L'ORACLE MORD** : la recette validee tient 25/25 ancres, le rognage aux zones en coute une, et
8 m de marge ne la recuperent pas. Les zones de callout d'Isolation ne couvrent donc pas tout le
terrain joue — le rognage aux zones est un levier de plus, pas un remplacant du maillage.

**Piste `hkaiTraversalAnnotationLibrary` : FERMEE.** La region 3 du navmesh est une table de
liens de saut (190 sur Isolation), sans aucune chaine — prouve par la reflexion du fichier-tag,
pas par sondage. Temoin `hinavmesh.TestNavmeshNePorteAucuneChaine`.

**Reste** : 274 des 434 StringId employes n'ont pas encore de texte joueur (extraction `uslg`) ;
le rejeu 2D n'affiche toujours pas les callouts Forge (catalogue cle `map_id` + essai au service
a ecrire). Detail : `.ai/V7.5/cartes/CALLOUTS_FORGE_2026-08-27.md`.

### 2026-08-27 — Isolation : VALIDEE, la formule est complete

Planche des cinq cuissons : https://claude.ai/code/artifact/7aef9c15-4034-4238-8f0f-915927684397

**Verbatim** : « valide avec + zones, marge 1 m ».

**La recette retenue cumule les deux sources qui disent ou l'on joue** :

1. le MAILLAGE DE NAVIGATION comme reference d'altitude (845 552 cellules), puis effacement de
   ce qui est hors de lui (2 175 499 cellules : dome, decor du canevas, dalles de ciel) et
   vidage des surfaces qui flottent loin du sol (tolerance 1,5 m, 121 976 cellules) ;
2. les ZONES DE CALLOUT de la carte, lues dans son propre `map.mvar`, en rognage SERRE
   (marge 1 m, 306 128 cellules effacees, 26,3 % de la matiere restante).

L'utilisateur a compare 1, 4 et 8 m et a choisi la plus serree — celle qui laisse le moins de
gribouillis autour des zones.

**Prix mesure, accepte** : 24/25 ancres au sol contre 25/25 sans le rognage aux zones, et 8 m de
marge ne recuperent pas l'ancre perdue. Les zones de callout d'Isolation ne couvrent pas tout
son terrain joue. Ecart connu et ecrit ; le verdict porte sur l'image.

**Ce que ca ouvre pour les autres cartes Forge** : le levier `rogneAuxZones` ne demande aucun
fichier a telecharger — les zones sont dans le `.mvar` que la cuisson lit deja. Toute carte
Forge qui en porte peut recevoir le meme traitement, a mesurer carte par carte.

### 2026-08-27 — LE RESTE A TRAITER : 27 cartes refusees le 13/08, hors lot du jour

Demande de l utilisateur apres la validation d Isolation : les cartes refusees au gate du 13/08,
MOINS les dix-huit qu il a relevees le 2026-08-27 comme exemptes de bouillie (celles-la sont
traitees dans leur propre lot). 34 refusees moins 7 recoupements = **27 cartes**.

Les 7 retirees, deja au lot du jour : Empyrean, Starboard, The Pit, Domicile, Goliath, Dredge,
Banished Narrows.

**TOUTES LES 27 PORTENT DES ZONES DE CALLOUT** (mesure du 2026-08-27, lecteur de production,
rateliers ecartes) : de 17 a 111 zones. Le levier `rogneAuxZones` livre ce jour s applique donc
a chacune, sans aucun telechargement — les zones sont dans le `.mvar` que la cuisson lit deja.
Le maillage de navigation, lui, demande un telechargement par carte : 2 blobs seulement sont
en depot (Isolation, Kiken na).

| carte | cle | matchs | cadrage | zones | navmesh |
|---|---|---|---|---|---|
| Origin | `b302eb62` | 24 | 85.7 % | 46 | a telecharger |
| Snowbound | `410f1c01` | 23 | 100 % | 23 | a telecharger |
| Absolution | `78da545f` | 21 | 89.9 % | 60 | a telecharger |
| Curfew | `63d634be` | 20 | 61.7 % | 34 | a telecharger |
| Dynasty | `cfd90b63` | 19 | 100 % | 35 | a telecharger |
| Nemesis | `2be34415` | 18 | 98.6 % | 111 | a telecharger |
| Cliffside | `4bffd021` | 18 | 98 % | 44 | a telecharger |
| Shiro | `2890782c` | 18 | 84 % | 31 | a telecharger |
| Fortress | `0d1c9255` | 17 | 87.4 % | 75 | a telecharger |
| Houseki | `cf034ec8` | 15 | 86 % | 64 | a telecharger |
| High Ground | `bb7b78ae` | 15 | 100 % | 24 | a telecharger |
| Takamanohara | `edcd4467` | 15 | 94.1 % | 17 | a telecharger |
| Elevation | `76043dc6` | 14 | 92.3 % | 64 | a telecharger |
| Kiken'na | `df7dbf08` | 13 | 94.9 % | 52 | en depot |
| Kaiketsu | `98a83f87` | 12 | 86.2 % | 105 | a telecharger |
| Salvation | `cd08bc7a` | 12 | 73.9 % | 26 | a telecharger |
| Solitude | `f1cc3b4e` | 12 | 55.4 % | 41 | a telecharger |
| Opulence | `255bbe78` | 12 | 61.9 % | 64 | a telecharger |
| Command | `2c9f3490` | 11 | 100 % | 36 | a telecharger |
| Critical Dewpoint | `bae4df14` | 10 | 55.7 % | 44 | a telecharger |
| Perilous | `c5ac9f12` | 10 | 100 % | 51 | a telecharger |
| Sylvanus | `95b69e4b` | 10 | 87.5 % | 53 | a telecharger |
| Refuge | `41217472` | 10 | 77 % | 83 | a telecharger |
| Smallhalla | `98783453` | 9 | 100 % | 29 | a telecharger |
| Obituary | `a289bafe` | 9 | 92.7 % | 60 | a telecharger |
| Shogun | `33075df7` | 9 | 100 % | 26 | a telecharger |
| Fortitude | `1ede38fa` | 7 | 99.1 % | 73 | a telecharger |

Lecture : `cadrage` = part de la largeur occupee par la matiere. **100 % = l image deborde**
(Snowbound, Dynasty, High Ground, Command, Perilous, Smallhalla, Shogun) ; **sous 65 % = la
carte est perdue dans son cadre** (Solitude 55,4, Critical Dewpoint 55,7, Curfew 61,7,
Opulence 61,9). Les deux defauts appellent le meme levier : rogner a ce qui est joue.

### 2026-08-28 — CAMPAGNE : 55 fonds Forge cuits au maillage + zones de callout

Planche de gate : https://claude.ai/code/artifact/2c7d0e4b-296a-4a4a-82c9-9d533e021367

**Demande** : « recupere les maillages et les zones de callout pour ces 27 + les non statuee de
l'artefact, puis genere proprement les fonds ». Perimetre reel : les 27 refusees du 13/08 et les
28 sans statut sont DISJOINTES — **55 cartes**.

**Ce qu'il a fallu rapatrier, et ce qui etait deja la** : les zones de callout n'ont demande
AUCUN telechargement (elles vivent dans le `.mvar` que la cuisson lit deja) ; les 55 en portent,
aucun ratelier. Seuls les maillages manquaient : **52 rapatries, 0 echec**, par la nouvelle
commande `cmd/mapnav-fetch` (resolution anonyme, ecriture en flux, reprise par saut).

Trois cas hors recette, ecrits dans leur reglage : **Thunderhead** et **Thunderhead Heavies**
n'ont pas de maillage publie (sous le seuil des ~1 000 objets) ; **Absolution**, **Insolence** et
**Insolence Heavies** en ont un que notre decodeur ne lit pas (`fichier-tag sans section TST1`).
Ces cinq-la sont cuites aux seules zones de callout.

**L'ECRETAGE DES TOITS N'EST PAS ARME**, et c'est un choix mesure : sous le seuil de couverture
la substitution ne se declenche jamais et l'ecretage ne retire plus que du sol (lecon Launch
Site). Le maillage fait tomber les coques par comparaison juste, pas par soustraction.

**LA MEMOIRE, MESUREE** : deux cartes dans un meme processus montent a **14,9 Go** pour 1,8 Go
libres ; une carte seule culmine a **17 Go** et rend tout a la sortie. Le cout est donc PAR
CARTE, pas cumulatif — la taille de lot est passee a 1 et `cuisson_par_lots.sh` la lit
desormais dans l'environnement. 55 cuites, **0 echec**.

**LE RATTRAPAGE DES CINQ CARTES AMPUTEES, ET CE QU'IL ETABLIT.** Cinq cartes perdaient
massivement des ancres. Deux diagnostics separes ont designe le coupable au lieu de le supposer :

| carte | recette de campagne | sans rognage aux ZONES | sans rognage au MAILLAGE |
|---|---|---|---|
| Flood Gulch | 7/22 | 7/22 | **22/22** |
| Rat's Nest | 10/23 | 10/23 | **23/23** |
| Vallaheim Firefight | 0/5 | 0/5 | **5/5** |
| Outlook | 6/9 | 6/9 | **9/9** |
| 944396dd (Narrows) | 18/20 | 18/20 | **20/20** |

**Les zones de callout sont innocentes** : a compte d'ancres identique, ce n'est pas elles. C'est
le rognage AU MAILLAGE qui effaçait du terrain joue — sur ces cartes le navmesh ne couvre pas
tout ce qui se joue. Les cinq sont republiees sans lui, maillage garde en REFERENCE.

**Regle qui en sort, applicable aux prochaines** : le rognage au maillage se garde tant qu'il ne
coute pas d'ancre, il se retire des qu'il en coute plusieurs. Neuf cartes perdent encore
exactement UNE ancre (Snowbound, Absolution, High Ground, Elevation, Salvation, Critical
Dewpoint, Refuge, Ecotone, Refuge Heavies) : c'est le meme prix qu'Isolation, accepte au gate du
27/08.

**Statut de toutes les lignes ci-dessous : A JUGER.** Aucune ne passe en VALIDEE sans verbatim.

| carte | cle | matchs | cadre | ancres | zones | maillage |
|---|---|---:|---|---|---:|---|
| Origin | `b302eb62` | 24 | 1428x992 | 20/20 | 46 | 1484 polys |
| Snowbound | `410f1c01` | 23 | 1349x1172 | 42/43 | 23 | 4521 polys |
| Absolution | `78da545f` | 21 | 1188x1303 | 21/22 | 60 | maillage illisible |
| Curfew | `63d634be` | 20 | 1224x1169 | 18/18 | 34 | 766 polys |
| Dynasty | `cfd90b63` | 19 | 1191x1567 | 13/13 | 35 | 932 polys |
| Cliffside | `4bffd021` | 18 | 1176x1348 | 14/14 | 44 | 1146 polys |
| Nemesis | `2be34415` | 18 | 1355x1353 | 26/26 | 111 | 1259 polys |
| Shiro | `2890782c` | 18 | 1363x1192 | 12/12 | 31 | 5778 polys |
| Fortress | `0d1c9255` | 17 | 1147x1285 | 29/29 | 75 | 989 polys |
| High Ground | `bb7b78ae` | 15 | 1754x1202 | 19/20 | 24 | 3596 polys |
| Houseki | `cf034ec8` | 15 | 1981x2047 | 8/8 | 64 | 4556 polys |
| Takamanohara | `edcd4467` | 15 | 1370x1183 | 12/12 | 17 | 778 polys |
| Elevation | `76043dc6` | 14 | 1099x1433 | 19/20 | 64 | 1094 polys |
| Kiken'na | `df7dbf08` | 13 | 1176x1240 | 13/13 | 52 | 689 polys |
| Kaiketsu | `98a83f87` | 12 | 1298x1030 | 16/16 | 105 | 773 polys |
| Opulence | `255bbe78` | 12 | 1331x1063 | 13/13 | 64 | 603 polys |
| Salvation | `cd08bc7a` | 12 | 1240x1152 | 11/12 | 26 | 921 polys |
| Solitude | `f1cc3b4e` | 12 | 1145x1323 | 19/19 | 41 | 758 polys |
| Command | `2c9f3490` | 11 | 2068x1634 | 43/43 | 36 | 3120 polys |
| Critical Dewpoint | `bae4df14` | 10 | 1286x1352 | 10/11 | 44 | 1038 polys |
| Perilous | `c5ac9f12` | 10 | 1174x1239 | 9/9 | 51 | 723 polys |
| Refuge | `41217472` | 10 | 1591x1611 | 45/46 | 83 | 2666 polys |
| Sylvanus | `95b69e4b` | 10 | 1240x1410 | 17/17 | 53 | 1579 polys |
| Obituary | `a289bafe` | 9 | 2295x1457 | 31/31 | 60 | 7514 polys |
| Shogun | `33075df7` | 9 | 2920x2213 | 13/13 | 26 | 16683 polys |
| Smallhalla | `98783453` | 9 | 1989x1431 | 52/52 | 29 | 1379 polys |
| Ecotone | `8816f240` | 8 | 1088x1006 | 10/11 | 47 | 862 polys |
| Fortitude | `1ede38fa` | 7 | 1839x2253 | 41/41 | 73 | 8927 polys |
| Insolence | `d5c5eb4f` | 7 | 3001x2867 | 38/38 | 41 | maillage illisible |
| Solution | `ee43d273` | 7 | 1297x1050 | 19/19 | 100 | 873 polys |
| Flood Gulch | `7097bc4f` | 6 | 2153x2767 | 22/22 | 8 | 7125 polys |
| Solitude - Ranked | `4a5e5612` | 5 | 1145x1323 | 19/19 | 41 | 758 polys |
| Threshold | `ddbb3a00` | 5 | 1994x2034 | 20/20 | 62 | 6448 polys |
| Fortitude Heavies | `305b1bdd` | 4 | 1840x2255 | 26/26 | 73 | 8927 polys |
| Thunderhead | `28a3ac28` | 4 | 3001x2493 | 46/46 | 5 | pas de maillage |
| Thunderhead Heavies | `37bc3df6` | 4 | 3001x2632 | 36/36 | 5 | pas de maillage |
| Obituary Heavies | `e3681516` | 3 | 2295x1457 | 31/31 | 60 | 7514 polys |
| Pharaoh | `88d45250` | 3 | 701x1196 | 9/9 | 6 | 66 polys |
| Credence | `0cc728d2` | 2 | 1927x2019 | 50/50 | 40 | 5943 polys |
| Disciple | `525451ca` | 2 | 895x1637 | 9/9 | 48 | 685 polys |
| Merchant's Square | `7dfec55d` | 2 | 712x1200 | 9/9 | 9 | 153 polys |
| Urban Raid | `be848f91` | 2 | 693x1292 | 9/9 | 5 | 517 polys |
| Vallaheim Firefight | `e8268e75` | 2 | 1571x2573 | 5/5 | 48 | 17029 polys |
| 944396dd-5661-4a16-b1d8-a6053f762c55 | `944396dd` | 1 | 2220x2638 | 20/20 | 21 | 944 polys |
| Dawnbreaker | `89dd4003` | 1 | 1942x1479 | 42/42 | 29 | 5772 polys |
| Insolence Heavies | `2a339c65` | 1 | 3001x2867 | 31/31 | 41 | maillage illisible |
| Lattice - Ranked | `1a6cfc2e` | 1 | 1217x1196 | 9/9 | 131 | 857 polys |
| Nadair | `6dbd1c0d` | 1 | 2133x947 | 4/4 | 41 | 2671 polys |
| Origin - Ranked | `46a8319c` | 1 | 1428x992 | 20/20 | 46 | 1484 polys |
| Outlook | `ea7b30e6` | 1 | 1516x1592 | 9/9 | 4 | 349 polys |
| Rat's Nest | `133c0185` | 1 | 1399x2016 | 23/23 | 26 | 2423 polys |
| Refuge Heavies | `c10c7e79` | 1 | 1637x1658 | 32/33 | 83 | 2666 polys |
| Ronin | `f459867d` | 1 | 659x1288 | 9/9 | 15 | 624 polys |
| Scarlett's Landing | `79042fc0` | 1 | 966x1100 | 9/9 | 8 | 308 polys |
| Warehouse | `5b12d6d9` | 1 | 723x1352 | 7/7 | 9 | 588 polys |

### 2026-08-28 — GATE : 36 fonds valides sur 55

**Verbatim** : « Validees : Origin, Curfew, Dynasty, Nemesis, Shiro, Houseki (mais attention au
« cadrage » sur celle-ci, elle est trop en haut a gauche de l'image, pas centree), Takamanohara,
Elevation, kiken'na, kaiketsu, Opulence, Salvation, Solitude, Perilous, Sylvanus, Obituary,
Ecotone, Fortitude, Solution, Solitude - ranked, threshold, Fortitude Heavies, Obituary Heavies,
Pharao, Credence, Disciple, Merchant's square, Urban Raid, Dawnbreaker, Lattice - ranked, Nadair,
Origin - ranked, Refuge heavies, Ronin, Scarlett's landing, Warehouse ».

**35 validees, 20 non validees.** Houseki etait annoncee validee sous reserve ; l utilisateur a
tranche apres la mesure : elle N EST PAS validee tant que son cadrage n est pas regle. Les 19
autres sont sans verdict (elles restent A JUGER, aucune n'est declaree refusee) :
Snowbound, Absolution, Cliffside, Fortress, High Ground, Command, Critical Dewpoint, Refuge,
Smallhalla, Shogun, Insolence, Flood Gulch, 944396dd, Thunderhead, Thunderhead Heavies,
Vallaheim Firefight, Insolence Heavies, Outlook, Rat's Nest.

#### La reserve sur Houseki, mesuree

Le cadre N'EST PAS en cause : `CadreUtile` a deja rogne l'image de 2946x3001 a 1981x2047, au ras
de la matiere. Ce qui est en cause est la REPARTITION de cette matiere :

- centre de masse a **(70,6 % ; 28,9 %)** de l'image au lieu de (50 ; 50) ;
- **94,5 % de la matiere dans un seul quadrant** (haut-droite), 0,6 % en haut-gauche ;
- la matiere ne couvre que **14,2 % des pixels**, mais sa boite touche presque les quatre bords :
  des filaments epars etirent la boite pendant que l'arene tient dans un coin.

Temoin de comparaison, Nemesis (validee sans reserve) : centre de masse (50,6 % ; 49,3 %),
quadrants 26/25/22/26, matiere sur 52,4 % des pixels. C'est a quoi ressemble une carte centree.

**Levier essaye et INOPERANT** : le bornage aux volumes de mort. La boite des 58 volumes de
Houseki vaut `[-263,4 -288,4 286,0 289,1]`, soit 549 x 577 m — c'est le canevas entier, pas
l'arene : **0 cellule effacee**, image identique au pixel pres. Le reglage a ete retire.

**Piste restante, non engagee** : la carte est peinte a 61,4 % par UN type d'objet (703364958,
26 exemplaires) et a 20,8 % par un second (-867485774, 5 exemplaires). Si les filaments viennent
de l'un d'eux, `typesExclus` les retire ; il faut d'abord etablir lequel peint le hors-arene, ce
qui demande une mesure par region que la chaine ne produit pas encore.

| carte | cle | matchs | cadre | ancres | zones | maillage | statut |
|---|---|---:|---|---|---:|---|---|
| Origin | `b302eb62` | 24 | 1428x992 | 20/20 | 46 | 1484 polys | VALIDEE 28/08 |
| Curfew | `63d634be` | 20 | 1224x1169 | 18/18 | 34 | 766 polys | VALIDEE 28/08 |
| Dynasty | `cfd90b63` | 19 | 1191x1567 | 13/13 | 35 | 932 polys | VALIDEE 28/08 |
| Nemesis | `2be34415` | 18 | 1355x1353 | 26/26 | 111 | 1259 polys | VALIDEE 28/08 |
| Shiro | `2890782c` | 18 | 1363x1192 | 12/12 | 31 | 5778 polys | VALIDEE 28/08 |
| Houseki | `cf034ec8` | 15 | 1981x2047 | 8/8 | 64 | 4556 polys | VALIDEE 30/08 |
| Takamanohara | `edcd4467` | 15 | 1370x1183 | 12/12 | 17 | 778 polys | VALIDEE 28/08 |
| Elevation | `76043dc6` | 14 | 1099x1433 | 19/20 | 64 | 1094 polys | VALIDEE 28/08 |
| Kiken'na | `df7dbf08` | 13 | 1176x1240 | 13/13 | 52 | 689 polys | VALIDEE 28/08 |
| Kaiketsu | `98a83f87` | 12 | 1298x1030 | 16/16 | 105 | 773 polys | VALIDEE 28/08 |
| Salvation | `cd08bc7a` | 12 | 1240x1152 | 11/12 | 26 | 921 polys | VALIDEE 28/08 |
| Solitude | `f1cc3b4e` | 12 | 1145x1323 | 19/19 | 41 | 758 polys | VALIDEE 28/08 |
| Opulence | `255bbe78` | 12 | 1331x1063 | 13/13 | 64 | 603 polys | VALIDEE 28/08 |
| Perilous | `c5ac9f12` | 10 | 1174x1239 | 9/9 | 51 | 723 polys | VALIDEE 28/08 |
| Sylvanus | `95b69e4b` | 10 | 1240x1410 | 17/17 | 53 | 1579 polys | VALIDEE 28/08 |
| Obituary | `a289bafe` | 9 | 2295x1457 | 31/31 | 60 | 7514 polys | VALIDEE 28/08 |
| Fortitude | `1ede38fa` | 7 | 1839x2253 | 41/41 | 73 | 8927 polys | VALIDEE 28/08 |
| Ecotone | `8816f240` | 8 | 1088x1006 | 10/11 | 47 | 862 polys | VALIDEE 28/08 |
| Solution | `ee43d273` | 7 | 1297x1050 | 19/19 | 100 | 873 polys | VALIDEE 28/08 |
| Solitude - Ranked | `4a5e5612` | 5 | 1145x1323 | 19/19 | 41 | 758 polys | VALIDEE 28/08 |
| Threshold | `ddbb3a00` | 5 | 1994x2034 | 20/20 | 62 | 6448 polys | VALIDEE 28/08 |
| Fortitude Heavies | `305b1bdd` | 4 | 1840x2255 | 26/26 | 73 | 8927 polys | VALIDEE 28/08 |
| Obituary Heavies | `e3681516` | 3 | 2295x1457 | 31/31 | 60 | 7514 polys | VALIDEE 28/08 |
| Pharaoh | `88d45250` | 3 | 701x1196 | 9/9 | 6 | 66 polys | VALIDEE 28/08 |
| Credence | `0cc728d2` | 2 | 1927x2019 | 50/50 | 40 | 5943 polys | VALIDEE 28/08 |
| Disciple | `525451ca` | 2 | 895x1637 | 9/9 | 48 | 685 polys | VALIDEE 28/08 |
| Merchant's Square | `7dfec55d` | 2 | 712x1200 | 9/9 | 9 | 153 polys | VALIDEE 28/08 |
| Urban Raid | `be848f91` | 2 | 693x1292 | 9/9 | 5 | 517 polys | VALIDEE 28/08 |
| Dawnbreaker | `89dd4003` | 1 | 1942x1479 | 42/42 | 29 | 5772 polys | VALIDEE 28/08 |
| Lattice - Ranked | `1a6cfc2e` | 1 | 1217x1196 | 9/9 | 131 | 857 polys | VALIDEE 28/08 |
| Nadair | `6dbd1c0d` | 1 | 2133x947 | 4/4 | 41 | 2671 polys | VALIDEE 28/08 |
| Origin - Ranked | `46a8319c` | 1 | 1428x992 | 20/20 | 46 | 1484 polys | VALIDEE 28/08 |
| Refuge Heavies | `c10c7e79` | 1 | 1637x1658 | 32/33 | 83 | 2666 polys | VALIDEE 28/08 |
| Ronin | `f459867d` | 1 | 659x1288 | 9/9 | 15 | 624 polys | VALIDEE 28/08 |
| Scarlett's Landing | `79042fc0` | 1 | 966x1100 | 9/9 | 8 | 308 polys | VALIDEE 28/08 |
| Warehouse | `5b12d6d9` | 1 | 723x1352 | 7/7 | 9 | 588 polys | VALIDEE 28/08 |
| Snowbound | `410f1c01` | 23 | 1349x1172 | 42/43 | 23 | 4521 polys | VALIDEE 30/08 |
| Absolution | `78da545f` | 21 | 1188x1303 | 21/22 | 60 | maillage illisible | BLOQUEE — inexploitable (bouillie centrale) tant que le maillage TSTR/FSTR n est pas lu |
| Cliffside | `4bffd021` | 18 | 1176x1348 | 14/14 | 44 | 1146 polys | VALIDEE 30/08 |
| Fortress | `0d1c9255` | 17 | 1147x1285 | 29/29 | 75 | 989 polys | VALIDEE 30/08 |
| High Ground | `bb7b78ae` | 15 | 1754x1202 | 19/20 | 24 | 3596 polys | VALIDEE 30/08 |
| Command | `2c9f3490` | 11 | 2068x1634 | 43/43 | 36 | 3120 polys | VALIDEE 30/08 |
| Critical Dewpoint | `bae4df14` | 10 | 1286x1352 | 10/11 | 44 | 1038 polys | VALIDEE 30/08 |
| Refuge | `41217472` | 10 | 1591x1611 | 45/46 | 83 | 2666 polys | VALIDEE 30/08 |
| Smallhalla | `98783453` | 9 | 1989x1431 | 52/52 | 29 | 1379 polys | VALIDEE 30/08 |
| Shogun | `33075df7` | 9 | 2920x2213 | 13/13 | 26 | 16683 polys | VALIDEE 30/08 |
| Insolence | `d5c5eb4f` | 7 | 3001x2867 | 38/38 | 41 | maillage illisible | BLOQUEE — inexploitable (bouillie centrale) tant que le maillage TSTR/FSTR n est pas lu |
| Flood Gulch | `7097bc4f` | 6 | 2153x2767 | 22/22 | 8 | 7125 polys | VALIDEE 30/08 |
| 944396dd-5661-4a16-b1d8-a6053f762c55 | `944396dd` | 1 | 2220x2638 | 20/20 | 21 | 944 polys | VALIDEE 30/08 |
| Thunderhead | `28a3ac28` | 4 | 3001x2493 | 46/46 | 5 | pas de maillage | VALIDEE 30/08 — image des Heavies |
| Thunderhead Heavies | `37bc3df6` | 4 | 3001x2632 | 36/36 | 5 | pas de maillage | VALIDEE 30/08 |
| Vallaheim Firefight | `e8268e75` | 2 | 1571x2573 | 5/5 | 48 | 17029 polys | HORS PERIMETRE — mode Firefight non supporte par le rejeu 2D (verdict 30/08) |
| Insolence Heavies | `2a339c65` | 1 | 3001x2867 | 31/31 | 41 | maillage illisible | BLOQUEE — inexploitable (bouillie centrale) tant que le maillage TSTR/FSTR n est pas lu |
| Outlook | `ea7b30e6` | 1 | 1516x1592 | 9/9 | 4 | 349 polys | VALIDEE 30/08 |
| Rat's Nest | `133c0185` | 1 | 1399x2016 | 23/23 | 26 | 2423 polys | VALIDEE 30/08 |

### 2026-08-28 — LES CARTES DU JEU QUI N ETAIENT PAS AU CATALOGUE

Planche : https://claude.ai/code/artifact/158547b6-b528-411a-a716-1c4092b29c46

**Verbatim** : « j'ai toujours pas vu Vacancy et les autres manquantes !!!!! faut les ajouter !! »
puis « Si je veux les 25 les plus jouees en dehors de ce qu'on a deja, c'est possible ? », puis
« Gardes pas les trois entrees douteuses ».

**LA PORTE QUI MANQUAIT** : le navigateur public de Halo Waypoint s'interroge SANS JETON, trie par
parties recentes et pagine (`/halo-infinite/ugc/browse?assetKind=Map&sort=playsrecent`). C'est la
liste des cartes reellement jouees — donc, en pratique, les cartes du jeu. **Sur les 200 plus
jouees, 109 etaient absentes de notre catalogue.** Vacancy n'en etait que la partie visible.

**PERIMETRE RETENU** (verdicts utilisateur) : hors variantes « - Ranked » (meme dessin que leur
base), hors variantes Firefight, hors cartes d'entrainement communautaires. Ecartees aussi les
republications UGC de cartes natives que nous avons deja (Bazaar 0 objet, Fragmentation, Oasis,
Breaker). **Vacancy et Showdown Arena entrent malgre la regle « pas de Ranked »** : ce sont les
deux seules dont la BASE manquait aussi.

**CE QUI A ETE RAPATRIE, SANS UN SEUL JETON** : 107 variantes `.mvar` et 89 `navmesh.blob` sur les
109 cartes reperees, puis **27 miniatures** `images/thumbnail.jpg` — au format 560x320, exactement
celui des images existantes de `static/maps/halo_infinite/`, sans retouche. Le dossier passe de
102 a 129 images. Nouvelle commande : `cmd/mapnav-fetch` (drapeau `-fichier`).

**LA CARTE 944396dd A ENFIN SON NOM** : sa page d'asset l'appelle **Narrows** (5 547 objets).

**LE RATTRAPAGE, ET CE QU'IL AJOUTE A LA REGLE.** Douze des 22 cuites perdaient des ancres, dont
cinq totalement (Security Zone 0/53, Showdown Arena 0/19, Ardent Prayer 0/5, Yuletide 0/4). La
regle de la veille — retirer le rognage AU MAILLAGE des qu'il coute plusieurs ancres — en a
rattrape huit d'un coup :

| carte | avec le rognage | sans |
|---|---|---|
| Security Zone | 0/53 | **53/53** |
| Showdown Arena | 0/19 | **19/19** |
| Courtyard | 8/20 | **20/20** |
| Alpha Site | 1/7 | **7/7** |
| Ardent Prayer | 0/5 | **5/5** |
| Yuletide | 0/4 | **4/4** |
| Lone Wolf | 13/16 | **16/16** |
| Ivory Tower | 19/21 | **21/21** |
| Boulevard | 21/24 | **24/24** |

**MEGAPOLIS EST LE PREMIER CAS INVERSE, et il complete la regle** : le rognage au maillage retire
ne changeait rien (9/12 dans les deux cas) ; ce sont ses ZONES DE CALLOUT qui effaçaient du terrain
joue — sans elles, **12/12**. Jusqu'ici les zones avaient toujours ete innocentees ; elles ne le
sont donc pas par nature. **Le bon reflexe est le diagnostic a deux branches, pas un coupable par
defaut.**

**VACANCY GARDE 7 ANCRES SUR 9, ET AUCUN DES DEUX LEVIERS N'EN EST LA CAUSE** : sans le rognage au
maillage 7/9, sans le rognage aux zones 7/9 aussi. Les deux ancres manquantes tombent la ou la
carte ne dessine rien au niveau de jeu. Ecart connu, du meme ordre que celui d'Isolation.

**TROIS CARTES BLOQUEES** : Munera Platform W4, Munera Platform H6 et Out With A Bang ne portent
AUCUN objectif dans leur variante. Le cadre d'un fond se batit sur les ancres d'objectifs — sans
elles il n'y a rien a cadrer. Meme mur que Cole Protocol.

| carte | cle | parties | cadre | ancres | canevas | statut |
|---|---|---:|---|---|---|---|
| Vacancy | `4fb5b69f` | 11922 | 1290x1505 | 7/9 | fo09_academy | VALIDEE 30/08 |
| Showdown Arena | `1042b738` | 9994 | 2478x2187 | 19/19 | fo11_blank | VALIDEE 30/08 |
| Interference | `654dff62` | 3767 | 1228x1126 | 14/14 | fo13_frost | VALIDEE 30/08 |
| Ardent Prayer | `8cf45707` | 3567 | 2255x2254 | 5/5 | fo11_blank | HORS PERIMETRE — mode Firefight non supporte par le rejeu 2D (verdict 30/08) |
| Courtyard | `841242db` | 2795 | 2634x1404 | 20/20 | fo09_academy | VALIDEE 30/08 |
| Diminished | `50a1a3b5` | 2711 | 1235x661 | 9/9 | fo11_blank | VALIDEE 30/08 |
| Megapolis | `0c299a3a` | 2690 | 1689x2685 | 12/12 | fo03_space | VALIDEE 30/08 |
| Ruujaya | `37a9b5f0` | 2654 | 1349x672 | 9/9 | fo11_blank | VALIDEE 30/08 |
| Yuletide | `b6c7bdfa` | 2519 | 2173x2172 | 4/4 | fo13_frost | VALIDEE 30/08 — N2 composantes + altitude 2 m |
| Foundry | `66f4fe86` | 1993 | 1185x1363 | 19/19 | fo11_blank | VALIDEE 30/08 |
| Guardian | `1441775d` | 1906 | 1152x1239 | 20/20 | fo05_desert | VALIDEE 30/08 |
| Serenity | `b4d13418` | 1685 | 1402x1234 | 28/28 | fo08_wetland | VALIDEE 30/08 |
| Powerhouse | `e6d73380` | 1573 | 1522x1738 | 5/5 | fo09_academy | VALIDEE 30/08 |
| Canopy | `beedcb81` | 1567 | 1195x1077 | 14/14 | fo05_desert | VALIDEE 30/08 |
| Ivory Tower | `2143a29c` | 1558 | 1109x1124 | 21/21 | fo10_deadland | VALIDEE 30/08 |
| Lone Wolf | `fdde5715` | 1541 | 1815x1636 | 16/16 | fo09_academy | VALIDEE 30/08 |
| Boulevard | `252e2a45` | 1498 | 1529x1310 | 24/24 | fo11_blank | VALIDEE 30/08 |
| Alpha Site | `6c433ed1` | 1376 | 2220x2170 | 7/7 | fo11_blank | HORS PERIMETRE — mode Firefight non supporte par le rejeu 2D (verdict 30/08) |
| Security Zone | `3922c263` | 1285 | 1683x1977 | 53/53 | fo11_blank | VALIDEE 30/08 — N2 composantes + altitude |
| Ghost Town | `71c3a721` | 1211 | 1454x1236 | 14/15 | fo08_wetland | VALIDEE 30/08 |
| Cold Storage | `78cebfc7` | 1123 | 1204x1090 | 24/24 | fo11_blank | VALIDEE 30/08 |
| Immolate | `47823612` | 725 | 1142x1218 | 13/13 | fo11_blank | VALIDEE 30/08 |
| Munera Platform W4 | `55d09d90` | 3699 | 4352 | fo11_blank | BLOQUEE — aucun objectif dans la variante, donc aucune ancre pour batir le cadre |
| Munera Platform H6 | `2c89dc96` | 3664 | 3925 | fo11_blank | BLOQUEE — aucun objectif dans la variante, donc aucune ancre pour batir le cadre |
| Out With A Bang | `6dc27650` | 783 | 5204 | fo09_academy | BLOQUEE — aucun objectif dans la variante, donc aucune ancre pour batir le cadre |

### 2026-08-30 — RATTRAPAGE : neuf cartes jugees sur une image perimee

Planche : https://claude.ai/code/artifact/9d7fc5b3-bb29-462f-a871-6f6ed72b5b80

**Le constat qui a declenche le lot** : sur les 51 fonds en attente de verdict, **neuf portaient
un jugement rendu sur une image cuite AVANT que la chaine sache lire le maillage de navigation et
les zones de callout**. Cinq n'avaient recu AUCUN reglage (Empyrean, Starboard, Domicile, Dredge,
Banished Narrows, plus Corpo), deux n'avaient qu'un bornage aux volumes de mort (The Pit, Goliath),
et Vagabond trainait en « a retravailler — gros » depuis le 26/08. Les refuser une seconde fois sur
ces images-la n'aurait rien appris.

**Toutes les neuf avaient un maillage disponible** — il n'avait simplement jamais ete rapatrie.
9 sur 9 descendus, 0 echec. Leurs objectifs etaient deja au catalogue (4 a 20 selon la carte).

**Resultat de la premiere cuisson** : 4 parfaites d'emblee (Vagabond 9/9, Empyrean 13/13, Banished
Narrows 13/13, Corpo 4/4), The Pit a 19/20 (le prix accepte), et **quatre amputees**.

**La regle des deux branches a tranche a la premiere** — le rognage AU MAILLAGE etait le coupable
sur les quatre :

| carte | avec le rognage | sans |
|---|---|---|
| Domicile | **0/13** | **13/13** |
| Dredge | **0/12** | **12/12** |
| Goliath | 10/13 | **13/13** |
| Starboard | 10/12 | **12/12** |

**Bilan du lot : 9 cartes, 0 echec, 8 a 100 % d'ancres au sol, une a -1.** Statut de toutes :
A JUGER. Aucune n'est declaree validee.

| carte | cle | statut precedent | cadre | ancres | recette | statut |
|---|---|---|---|---|---|---|
| The Pit | `648ae7aa` | REFUSEE 13/08 | 1057x1278 | 19/20 | maillage borne + zones a 1 m | VALIDEE 30/08 |
| Goliath | `504ebf22` | REFUSEE 13/08 | 1078x1502 | 13/13 | maillage en reference + zones a 1 m | VALIDEE 30/08 |
| Vagabond | `105f5d84` | A RETRAVAILLER — gros | 1261x1267 | 9/9 | maillage borne + zones a 1 m | A JUGER |
| Empyrean | `d035fc3e` | REFUSEE 13/08 | 1370x1185 | 13/13 | maillage borne + zones a 1 m | VALIDEE 30/08 |
| Starboard | `7a9265af` | REFUSEE 13/08 | 1210x1170 | 12/12 | maillage en reference + zones a 1 m | VALIDEE 30/08 |
| Domicile | `921aebb1` | REFUSEE 13/08 | 1075x1684 | 13/13 | maillage en reference + zones a 1 m | VALIDEE 30/08 |
| Dredge | `e4bb06db` | REFUSEE 13/08 | 2583x2272 | 12/12 | maillage en reference + zones a 1 m | VALIDEE 30/08 — masque des positions jouees, rayon 1 m |
| Banished Narrows | `9ad226d8` | REFUSEE 13/08 | 1550x739 | 13/13 | maillage borne + zones a 1 m | VALIDEE 30/08 |
| Corpo | `8be179f7` | A FINALISER — correction utilisateur 26/08 | 598x1090 | 4/4 | maillage borne + zones a 1 m | VALIDEE 30/08 |
