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
| Behemoth | `va_behemoth` | 44 | 83.2 | -17,51 | natif | VALIDEE 26/08 |
| Empyrean | `d035fc3e` | 29 | 68.7 | -14,41 | forge | REFUSEE 13/08 |
| Origin | `b302eb62` | 24 | 85.7 | 0,01 | forge | REFUSEE 13/08 |
| Launch Site | `va_launchsite` | 24 | 53.5 | -0,40 | natif | VALIDEE `encre` 26/08 — SANS ecretage, comblement + sans eau |
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
