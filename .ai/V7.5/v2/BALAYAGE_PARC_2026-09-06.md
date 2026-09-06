# Balayage de non-regression du parc de rejeux — 2026-09-06

> Question posee : entre les artefacts cuits aux schemas 1 a 38 et ce que le code d'aujourd'hui
> (schema 39, HEAD `f1c7b411f`) produit du MEME film, qu'est-ce qui a ete PERDU ? La perte des
> actions de capture et de vol de drapeau d'un match CTF (`c0a82e88`) etait le point de depart ;
> l'objet de ce balayage est de dire si elle est isolee ou generale, et de chercher les autres.

## 1. Verdict en une page

| | |
|---|---|
| Artefacts anciens inventories | **465 fichiers**, **161 contenus distincts**, **119 matchs**, **19 schemas** (1 a 38) |
| Matchs re-cuits au HEAD | **119 / 119**, aucun echec, aucun depassement memoire |
| Paires comparees | **161** (chaque ancien contenu contre son re-cuit) |
| Ecarts mesures | **26 098** sur ~570 mesures par paire |
| Matchs avec au moins une PERTE | **114 / 119** — mais l'immense majorite est expliquee (voir §6) |
| **Regression candidate n° 1** | **Les actions d'objectif de CTF ne sont plus attribuees** : 297 actions perdues sur 14 matchs, dont 20 captures de drapeau, 44 prises, 18 vols. **Generale a tous les CTF du parc, PAS isolee a `c0a82e88`.** |
| Regressions candidates suivantes | grappin (−35 % de tractions sur 18 matchs), episodes d'equipement (−1 a −2 sur 11 matchs), 3 xuids perdus des pistes |
| Tout le reste | Explique par la chronique de `document.go` ou par un commit date (§6) |

Le point de depart etait donc le SYMPTOME d'un probleme general : `c0a82e88` perd 5 actions
(17 -> 12) ; `008e1bba` en perd 49 (60 -> 11), `846044ba` 30, `cf040013` 25, `145908d1` la
TOTALITE (7 -> 0).

## 2. Methode, et ce qu'elle garantit

### 2.1 La cuisson est CELLE DE LA PRODUCTION

`cmd/replay-build --map <carte> --facts <fichier> <matchId>`, un film par processus, en serie
stricte. C'est LITTERALEMENT le chemin de l'enfant de `backfill-replay`
(`cmd/levelup/cmd_backfill_replay_child.go:69,88` : `replaybuild.NewBuilder` puis
`BuildMatch(one, mapNames, filmDir, faits)`) — verifie sur pieces, pas suppose.

Bornes respectees a chaque film (`cmd/replay-build/main.go:armerProtections`) :

- verrou d'exclusion `filmproc.AcquireSolo` (jamais deux decodages a la fois, machine comprise) ;
- plafond memoire dur `filmproc.Arm` a 3 Gio (`filmproc.DefaultLimitGiB`) ;
- priorite processus basse ;
- plafond de temps de l'operateur : `timeout 300`.

**Pic memoire maximum observe sur les 119 films : 0,56 Gio** (`4f77afc1`, 64 chunks) — 19 % du
plafond. Aucune sentinelle n'a tranche.

### 2.2 Les FAITS DU MATCH sont fournis, et c'est la condition de validite

Un artefact cuit sans les lignes de match sort SANS compteurs de joueur, SANS actions
d'objectif et avec une identite de camps `unresolved` (`domain/match_facts.go:54-59`, mesure du
2026-08-24). Comparer un tel artefact au parc rendrait une avalanche de fausses regressions.

Les faits sont donc exportes une fois pour les 119 matchs par la CLI dediee
(`levelup replay-facts-export`, lecture seule `OpenReadForQuery` sur la base du checkout
principal), puis passes a chaque cuisson par `--facts`.

> **PIEGE RENCONTRE, ET IL A COUTE UNE PASSE ENTIERE.** La premiere serie de 119 cuissons a
> tourne SANS faits : le chemin Windows du fichier de faits etait construit dans une chaine bash
> (`"$SPW\\facts\\$s8.facts.json"`), ou la sequence `\\$` a mange l'expansion de la variable. Le
> symptome etait un `[WARN] faits du match illisibles` par cuisson — et le rapport disait alors
> « 89 matchs sur 119 ont perdu leur courbe de score », ce qui etait FAUX. Tous les chemins
> passent desormais par `cygpath -w`. Lecon generale : sur ce poste, **ne jamais fabriquer un
> chemin a backslashes dans une chaine bash**.
>
> Meme famille, meme passe : le premier essai avait aussi une racine de travail sans
> `data/cache/film_manifests` — le manifeste absent fait tomber la courbe de score en silence
> (`replaybuild/filmload.go:54`, journal INFO). Corrige avant la mesure.

### 2.3 Le parc de reference n'a pas ete touche

Les cuissons ecrivent dans une racine de travail dediee, ou seuls `film_chunks`,
`film_manifests`, `mvar` (lecture) et `data/titles` sont des JONCTIONS ; `data/cache/replays`
y est un vrai dossier. Temoins pris avant et rejoues apres les 119 cuissons :

- `LevelUp-go-migration/data/cache/replays` : **INCHANGE** (128 fichiers, mtimes identiques) ;
- `LevelUp-go-migration/data/titles/halo_infinite/reference` : **INCHANGE** (230 fichiers) ;
- `LevelUp-go-migration/data/cache/film_chunks` : **INCHANGE** (1 380 entrees) ;
- `git status` du worktree : rien sous `data/`.

### 2.4 L'outil de comparaison : `cmd/replay-diff`

Il ne deserialise PAS `replay.ReplayDocument` — le faire jetterait exactement ce qu'on cherche
(`encoding/json` ignore en silence tout champ que la structure d'aujourd'hui ne declare plus,
donc un calque supprime disparaitrait des DEUX cotes). La lecture est generique
(`map[string]any`), et chaque artefact est reduit a une EMPREINTE : un jeu de mesures nommees
(`objectifs/objectives/par-joueur/<xuid>/flag_captures` = 3, `pistes/tracks/points` = 29 220,
`couverture/coverage.score.teamIdentity` = "a"...).

Deux passes :

- **generique** — la taille de CHAQUE calque de premier niveau, la longueur de chaque
  sous-tableau, la presence de chaque champ, la repartition des champs de famille (`stat`,
  `state`, `kind`, `family`, `end`...) et le compte par `xuid`. Elle ne connait aucun nom de
  champ : un calque neuf entre au rapport sans qu'on l'y inscrive, un calque disparu se voit ;
- **specialisee** — les croisements que la premiere ne fait pas : joueur x famille d'action,
  dernieres valeurs des series de score, vies nommees, aplatissement de `coverage` et de
  `bombStats`.

Sens des ecarts : mesure absente de l'ancien = **apparu** (champ neuf, jamais une regression) ;
absente du nouveau = **disparu** ; valeur numerique plus basse = **perte** ; plus haute =
**gain** ; texte different = **changement**. Tolerance relative 1e-6 sur les flottants (les
bornes de carte sont des dequantifications).

## 3. Inventaire des anciens artefacts

### 3.1 Sources balayees

| Source | Fichiers vus | Retenus (contenus distincts) |
|---|---|---|
| `LevelUp-go-migration/data/cache/replays/halo_infinite` (parc principal) | 107 | 107 |
| `LevelUp-go-migration/data/cache/replays/_backup_2026-09-03` | 2 | 2 |
| Cle PNY `E:\replays\halo_infinite\v01..v31` (archive par version) | 76 | 47 |
| Cle PNY `E:\replays\_ancien_halo_infinite_2026-09-06` | 56 | 0 (doublons du parc) |
| 105 worktrees (`git worktree list`) + ancien principal `Downloads\Scripts\LevelUp` | 224 | 5 |
| **Total** | **465** | **161** |

Les 5 retenus des worktrees : `000d5950` et `01e1f945` au schema 29 (`wt-biped-pickup`),
`01e1f945` au 32 (`wt-origine-equipement`), `1b2d9e08` au 38 (`wt-lecture-equipement`),
`696a9d7c` au schema 1 (`re-mode-score`) — cinq etats que ni le parc ni la cle ne portaient.
Sur les 224 fichiers vus cote worktrees, 218 sont des doublons du parc principal (109 par la
jonction `LevelUp-wt-cuisson-perf/data`, 109 par le principal lui-meme, qui figure dans
`git worktree list`).

`LevelUp-wt-cuisson-perf/data` est une jonction vers le `data` du principal (contenus
strictement identiques, verifie par md5) — compte une seule fois. Six worktrees seulement
portaient des artefacts (`biped-pickup`, `pickup-ui`, `origine-equipement`,
`lecture-equipement`, `re-mode-score`, `v2-ctf`) ; l'ancien principal `Downloads\Scripts\LevelUp`
n'a plus de dossier `replays`. La cle PNY porte un `INDEX.csv` deja classe par version : c'est
elle qui fournit les schemas les plus anciens (1, 2, 6, 11, 13, 16, 17, 18, 30).

### 3.2 Repartition par schema (contenus distincts)

| Schema | 1 | 2 | 6 | 11 | 13 | 16 | 17 | 18 | 20 | 21 | 23 | 28 | 29 | 30 | 31 | 32 | 34 | 37 | 38 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Artefacts | 3 | 6 | 3 | 2 | 1 | 2 | 1 | 3 | **47** | 4 | 7 | 4 | 4 | 5 | 3 | 11 | **51** | 1 | 3 |

Deux grosses cohortes : le schema 20 (re-cuisson du 2026-08-26) et le schema 34
(2026-09-02). **Aucun artefact du parc n'est au schema 39** — la re-cuisson du parc n'a jamais
eu lieu, exactement ce que dit le journal du plan.

Profondeur par match : 100 matchs n'ont qu'un schema, 12 en ont 2, 3 en ont 3, 2 en ont 5,
`696a9d7c` en a 6 et `01e1f945` en a **7** (schemas 2, 2, 11, 17, 20, 23, 29) — c'est le
temoin le plus profond du parc.

### 3.3 Films disponibles

**119 / 119 matchs cuisables** : chacun a son dossier `data/cache/film_chunks/<short8>` (et son
manifeste) dans le checkout principal. Aucun match a rapporter comme non cuisable. Cout : 8 a
64 chunks par film, 3 348 chunks au total.

## 4. Cuisson

| | |
|---|---|
| Matchs cuits | **119 / 119** |
| Echecs | **0** (aucune carte hors catalogue, aucun timeout, aucune mort memoire) |
| Duree cumulee | **36,0 min** (mediane ~15 s, moyenne 18 s) |
| Film le plus long | `084a804d` (Fortitude Heavies, 57 chunks) — 108 s |
| Pic memoire max | **0,56 Gio** (`4f77afc1`, 64 chunks) |

## 5. Tableau par axe

Une « paire » = un ancien artefact confronte a son re-cuit (161 paires). « Matchs avec perte » =
matchs distincts portant au moins une perte ou une disparition sur cet axe.

| Axe | Paires avec ecart | Matchs avec perte | pertes | disparus | gains | apparus | changements |
|---|---|---|---|---|---|---|---|
| couverture | 161 | 95 | 324 | 91 | 666 | 7 659 | 50 |
| pistes | 152 | 78 | 679 | 8 | 824 | 175 | 0 |
| equipement | 148 | 50 | 197 | 43 | 377 | 2 718 | 0 |
| carte (bornes, geometrie, structure) | 37 | 19 | 26 | 0 | 74 | 27 | 0 |
| ports (drapeau, crane, bombe, VIP) | 37 | 16 | 91 | 1 | 118 | 159 | 0 |
| **objectifs** | 27 | **14** | **106** | **153** | 62 | 537 | 0 |
| score | 55 | 13 | 14 | 2 | 58 | 116 | 0 |
| armes | 158 | 3 | 1 | 2 | 345 | 6 025 | 0 |
| objets d'objectif | 53 | 3 | 0 | 3 | 20 | 181 | 0 |
| vehicules | 38 | 2 | 51 | 0 | 6 | 1 277 | 0 |
| joueurs | 47 | 2 | 7 | 0 | 0 | 1 726 | 0 |
| horloges | 152 | 1 | 1 | 0 | 2 | 163 | 0 |
| grenades / projectiles | 21 | 1 | 0 | 1 | 16 | 209 | 0 |
| statistiques d'Assaut | 4 | 0 | 0 | 0 | 0 | 130 | 0 |
| roster | 53 | 0 | 0 | 0 | 250 | 100 | 0 |
| morts neutres | 3 | 0 | 0 | 0 | 0 | 26 | 0 |
| entete | 161 | 0 | 0 | 0 | 161 | 0 | 40 |

**Axes demandes et absents de l'artefact** : les MEDAILLES et le FIL des eliminations ne sont pas
dans le document de rejeu — le fil est reconstruit cote client depuis `killer_victim_pairs`
servi par l'API, et aucune medaille n'existe dans `analysis/replay` hors tests de recherche.
Ils ne peuvent donc pas regresser par la cuisson. Les statistiques d'Assaut (`bombStats`,
`bombArmings`, `bombEvents`) n'apparaissent que dans le nouveau (130 mesures apparues,
0 perte) : aucun artefact du parc n'est assez recent pour les porter.

### 5.1 Lecture prudente de deux axes

- **couverture** : une baisse n'y est pas toujours une perte. `noSlot`, `ambiguous`,
  `unpublished`, `outOfWindow`, `missedEstimate`, `counterJumps`, `other` sont des COMPTEURS
  D'ECART — leur baisse est une amelioration. Les seules baisses de couverture retenues comme
  signal sont celles de compteurs de PRODUCTION (`attached`, `published`, `pulls`, `spawns`,
  `slots`, `episodes`).
- **pistes** : les 679 pertes sont dominees par un motif unique, **exactement 1 a 4 points de
  trajectoire en moins sur 93 paires** (0,00 a 0,01 %) — voir §6.3, c'est un correctif.

## 6. Axes avec pertes : explique ou non

### 6.1 INEXPLIQUE — actions d'objectif de CTF non attribuees (REGRESSION)

**Mesure.** 14 matchs, **297 actions perdues** :

| Famille | Actions perdues |
|---|---|
| `kills` (action d'objectif) | −130 |
| `assists` | −64 |
| `flag_grabs` (prises) | −44 |
| `flag_captures` (captures) | **−20** |
| `flag_steals` (vols) | **−18** |
| `flag_carriers_killed` | −8 |
| `zone_captures` | −6 |
| `flag_returns` | −5 |
| `flag_capture_assists` | −4 |

| Match | Variante | Manches | Ancien (s20 sauf mention) | Nouveau |
|---|---|---|---|---|
| `008e1bba` | CTF:Arena | 1 | 60 | **11** |
| `846044ba` | CTF:Arena Neutral Flag | 1 | 37 | **7** |
| `a17e61a2` | CTF:Arena | 1 | 56 | **22** |
| `cf040013` | CTF:Arena Neutral Flag | 1 | 30 | **5** |
| `e94163af` | CTF:Arena Neutral Flag | 1 | 108 | 76 |
| `66aa5f0b` (s21) | BTB:Total Control | 1 | 56 | 29 |
| `06dfe6d9` | BTB:Fiesta CTF | 1 | 136 | 114 |
| `b8d1fe0c` | CTF:Arena Neutral Flag | 1 | 77 | 61 |
| `04023f8a` | BTB:Fiesta CTF | 1 | 27 | 12 |
| `af13e2b2` (s18 et s20) | Arena:Strongholds | 1 | 79 | 66 |
| `0f9550e5` | CTF:Arena Neutral Flag | 1 | 86 | 77 |
| `c0a82e88` | Husky Raid:CTF | 1 | 17 | **12** |
| `bcb6d393` | CTF:Arena | 1 | 67 | 76 (gain) |
| `145908d1` | BTB:CTF | 1 | 7 | **0** |

**Ou le signal se casse.** Le film NOMME toujours autant d'evenements — c'est l'ATTRIBUTION qui
tombe. Journal de cuisson (`replaybuild: actions d'objectif identifiees par manche`) :

```
008e1bba : nommees=60  identifiees=11
c0a82e88 : nommees=92  identifiees=12
cf040013 : nommees=44  identifiees=5
846044ba : nommees=38  identifiees=7
145908d1 : nommees=13  identifiees=0
```

Sur l'ensemble du parc : 15 878 evenements nommes, 14 193 identifies (89,4 %), **25 matchs a
attribution partielle**.

**Cause, identifiee sur pieces.** Commit **`d173b1a8c` (2026-08-28)**, « obj-parmanche(1) :
calque Objectives identifie PAR MANCHE (fil des morts) ». Il remplace le pont slot -> xuid par
les TOTAUX du match (`SlotIdentityFrom`, exige les lignes) par un pont PAR MANCHE via les seuls
instants de mort (`ResolveRoundIdentity` + `IdentifyNamedEventsByRound`,
`replaybuild/matchfacts.go:199-231`). Motif legitime : en multi-manche le slot d'entite statborg
est reattribue, et l'ancien pont collait les actions d'apres-bascule au mauvais joueur.

**Pourquoi c'est quand meme une regression.** Le message du commit affirme :

> « Neutralite mono-manche prouvee par construction (une manche = pont plat par morts). »

**Les 14 matchs touches sont TOUS a UNE SEULE MANCHE** (`coverage.score.rounds` = 1, verifie
sur les 14 artefacts re-cuits). La neutralite mono-manche est donc **refutee par la mesure** :
la preuve etait synthetique, aucun film reel ne l'a controlee, et le commit n'a bumpe aucun
schema — donc rien n'a jamais force une re-cuisson qui aurait rendu l'ecart visible.

**Portee produit.** Les actions d'objectif sont le seul calque qui dise ce qu'un joueur a FAIT
(les autres disent ou il etait). Perdre 20 captures et 18 vols de drapeau, c'est perdre la
lecture du mode sur la moitie des CTF du parc. `145908d1` passe de « nominal » a
`coverage.verdict.objectives = aucune donnee`.

**Reponse a la question posee** : la perte de `c0a82e88` **n'est pas isolee**. Elle est
generale a tous les CTF du parc, et `c0a82e88` en est meme un cas MODERE (−5) a cote de
`008e1bba` (−49) ou `145908d1` (−7 sur 7).

### 6.2 A INSTRUIRE — tractions de grappin en baisse de 10 a 40 %

18 paires, 16 matchs. Contre les references les plus recentes du parc (schema 34, 2026-09-02) :

| Match | Ancien | Nouveau |
|---|---|---|
| `879a4dba` (s34) | 23 | 15 |
| `0a44c6cc` (s34) | 8 | 5 |
| `bf15f7ab` (s34) | 6 | 3 |
| `9ffce8ef` (s34) | 49 | 45 |
| `4f77afc1` (s34) | 31 | 28 |
| `084a804d` (s20) | 71 | 61 |
| `06dfe6d9` (s20) | 71 | 63 |

L'entonnoir de couverture suit (`coverage.grapple.pulls` 23 -> 15, `pullLives` 14 -> 11), et le
canal des CHARGES de grappin apparait en meme temps (`abilityCharges/par-family/grapple` : 39
lectures neuves sur `879a4dba`). Le chantier des usages d'equipement du schema 38
(`document.go` §« CE QUE LA VERSION 38 PORTE : LA LECTURE FIABLE DES USAGES D'EQUIPEMENT »,
commits `5a7ec4208`, `ba32c615b`, `58ad66573`) a manifestement resserre la detection.
**Rien dans la chronique ne dit que le nombre de tractions devait baisser** : soit ce sont des
faux positifs elimines (amelioration), soit des tractions reelles perdues. L'artefact seul ne
tranche pas — un releve Theater sur un match (le protocole deja employe pour le propulseur,
« valide 5/5 ») trancherait.

### 6.3 EXPLIQUE — les 12 autres familles de pertes

| Famille de perte | Ampleur | Explication, sur pieces |
|---|---|---|
| **1 a 4 points de trajectoire en moins** (93 paires) et **bornes de scene qui se resserrent** (26 mesures) | −0,00 a −0,01 % de points ; bornes parfois divisees par 100 | Les points supprimes etaient **aberrants**, et les bornes le prouvent : `4577fcc4` (s34) `maxX` 217,30 -> **1,50**, `minZ` −481,65 -> **103,18** ; `58801bc5` (s34) `maxZ` 383,37 -> −132,66 ; `000d5950` (s1) `maxY` 580,67 -> 30,35 sur une carte de ~50 m. `boundsOf` (`replay/geometry.go:170`) n'a pas change : c'est l'entree qui est assainie. **Gain, pas perte** — l'ancien cadrage etait faux. |
| **`flagCarries` 7 -> 1** (4 matchs) et **2 -> 1** (4 matchs), `coverage.flagCarries.spawns` idem | −6 vies de drapeau | Chronique **schema 35**, `document.go:428-433` : « LA VARIANTE "DRAPEAU NEUTRE" est reconnue : elle ne publie plus DEUX drapeaux qui n'existent pas mais UN SEUL, d'equipe −1, au socle du centre ». Les 8 matchs touches sont **tous** en `CTF:Arena Neutral Flag`. **Correctif documente.** |
| **Scores d'equipe echanges** (14 paires) | perte d'un cote, gain EXACTEMENT egal de l'autre (verifie 14/14 : −5/+5, −17/+17, −107/+107...) | Accompagne systematiquement de `coverage.score.teamIdentity` `unresolved` -> `a` ou `b` (29 paires). L'identite des camps est desormais RESOLUE la ou elle ne l'etait pas : le score va au bon camp. **Gain.** Aucun cas d'echange sur un artefact deja resolu. |
| **Compteurs d'un joueur en baisse** (`43716616` s21 : 13 -> 9 frags ; `51ebbc0f` s21 : 14 -> 6) | 2 matchs, 7 mesures | Dans les DEUX cas l'ancien artefact ne portait qu'UN joueur au fil de score et le nouveau en porte 7 a 8 de plus (`apparu`), avec `teamIdentity` passee de `unresolved` a `a`. C'est une **re-attribution au sein d'un gain massif**, pas une perte de donnee. |
| **`vehicles` 30 -> 20** (`0d76e8f1` s29), **11 -> 8** (`fccc61cd` s29) | 2 matchs | `coverage.vehicles.merged` apparait a **10** et **3** : les vies en relais (fantomes) sont desormais FUSIONNEES — commit `89a67a48f` (2026-09-02), « fusion des vies en relais (fantomes) ». Le meme diff apporte la visee des occupants (`aimReads` 22 963) et les tirs en vehicule. **Gain net.** |
| **`equipmentPlacements/par-family/other` disparu** (35 paires) | 35 | Compense par l'apparition des familles nommees : `shroud_screen` (39), `grenade_frag` (12), `grenade_plasma` (11), `grapple` (11), `grenade_spike` (10), `wall`, `sensor`, `repulsor`... **Reclassement**, chronique schema 10 (`document.go:89`, origine et familles des poses). |
| **`pickups/par-kind/item` disparu** (`000d5950`, `01e1f945`, s29 : 82 et 66) | 2 paires | Chronique **schema 31**, `document.go` §« CE QUE LA VERSION 31 TRANCHE » : « `kind` distinguait l'arme du reste ; il distingue desormais » arme / grenade / equipement. La valeur `item` a ete remplacee, pas perdue. |
| **`coverage.zones.slots` divise par deux** (8 paires : 28->13, 37->15, 31->14...) | 8 paires | `zoneStates/n` ne bouge PAS sur ces memes paires : ce sont les SLOTS observes qui sont dedupliques, pas les zones publiees. Aucun effet produit. |
| **`grenades[].k` disparu** (`696a9d7c` s1, 167 lectures) | 1 paire | Le champ n'existe plus dans `replay.Grenade` (`grenades.go:49-76` : `t`, `slot`, `i`, `x`, `y`, `rank`, `s`, `proj`). Artefact du 2026-07-31, anterieur au schema 3 qui a refait le lancer de grenade (`document.go:34`). |
| **`frameIntervalMs` 250 -> 100** | 1 paire (`000d5950` s1) | Changement de cadence par defaut du rejeu, artefact du 2026-07-31. |
| **`zoneStates.spans.progress` disparu** (3 paires s20) | 3 a 5 intervalles | Le champ existe toujours (`document_zones.go:218`, `Progress *float32`) mais il cede la place : sur les MEMES paires le calque des zones explose — `01e1f945` (s20) passe de 5 a **100** intervalles et gagne `owner` (50 presences), `8076f97f` de 3 a **36**. C'est la publication du PROPRIETAIRE de zone du **schema 21** (`document.go:173-177`). **Gain massif**, dont la disparition de `progress` est l'effet de bord. |
| **`weaponLabels` 17 -> 16** (`bf15f7ab` s34) | 1 | Une arme de moins au catalogue derive du film ; sans consequence sur les tirs publies (`shots/n` en gain sur cette paire). |

### 6.4 A SURVEILLER — deux residus petits mais non expliques

- **Episodes d'equipement (camo / surbouclier) : −1 a −2 sur 11 matchs.**
  `084a804d` 21 -> 19, `5dfdc63b` 32 -> 30, `8a485699` 77 -> 74, `13d92593` 1 -> 0,
  `82f29378` 1 -> 0. L'entonnoir suit (`coverage.equipment.camoEpisodes`,
  `overshieldEpisodes`/`overshieldLives`). Aucune chronique ne mentionne un durcissement de ces
  deux familles depuis le schema 24. Faible portee produit (un episode de camouflage sur 20),
  mais c'est un signal constant.
- **Un joueur disparait des pistes sur 3 matchs** : `tracks/xuids-distincts` 24 -> 23
  (`145908d1`, `4f77afc1`), 18 -> 17 (`11de8353`). Le `roster` ne perd RIEN nulle part (0 perte
  sur 161 paires) : le joueur reste identifie au match, mais plus aucune de ses vies ne porte
  son xuid. A rapprocher du chantier « une track = une vie » du schema 36, qui a fait exploser
  le nombre de vies (`tracks/n` +40 % en moyenne : 85 -> 125, 253 -> 373) tout en faisant
  baisser de 1 a 4 le nombre de vies NOMMEES sur 19 paires.

## 7. Regressions candidates, par gravite produit

| # | Regression | Portee | Etat |
|---|---|---|---|
| **1** | **Actions d'objectif de CTF non attribuees** — 297 actions perdues, dont 20 captures, 44 prises, 18 vols de drapeau ; `145908d1` perd la totalite | **14 matchs, tous les CTF du parc** + 1 Strongholds + 1 Total Control | Cause identifiee (`d173b1a8c`, pont par manche) ; la neutralite mono-manche affirmee par le commit est **refutee sur 14 films reels** |
| **2** | **Tractions de grappin −10 a −40 %** | 16 matchs | Coincide avec le chantier « usages d'equipement » du schema 38 ; nature (faux positifs elimines ou tractions perdues) **non tranchee** par l'artefact |
| **3** | **Episodes de camouflage / surbouclier −1 a −2** | 11 matchs | Sans explication dans la chronique |
| **4** | **Un joueur perd toutes ses vies nommees** | 3 matchs | Probablement un effet de bord du decoupage « une track = une vie » (schema 36) |

Tout le reste des 26 098 ecarts est soit un GAIN (les nouveaux calques : vehicules, ramassages
natifs, armes au sol, charges et impulsions d'equipement, translocations, statistiques
d'Assaut, coup d'envoi film, visee), soit un correctif documente (§6.3).

## 8. Limites de ce balayage

1. **La reference n'est pas une verite** : un ancien artefact peut avoir ete FAUX (les bornes de
   scene le prouvent). Une « perte » mesuree ici est un ECART a instruire, pas un verdict.
2. **Un seul re-cuit par match** : les 161 paires partagent 119 nouveaux artefacts. Un ecart
   commun a plusieurs schemas d'un meme match n'est donc compte qu'une fois en « matchs ».
3. **La comparaison est agregee, pas element par element** : elle voit « 3 captures -> 0 » et
   « ces captures ne sont plus au meme joueur », mais pas « la capture de la 4e minute est
   datee 200 ms plus tot ». Les decalages temporels a effectif constant lui echappent.
4. **Aucune verification visuelle** : le rejeu n'a pas ete ouvert dans l'application.
5. **Medailles et fil des eliminations** ne sont pas dans l'artefact (§5) : ce balayage ne peut
   rien en dire.

## 9. Reproduire

```bash
# 1. les faits (lecture seule sur la base du principal, serveur arrete)
levelup replay-facts-export --out <travail>/facts <short8>...

# 2. une cuisson = un processus borne (verrou solo + plafond 3 Gio + priorite basse)
LEVELUP_REPO_ROOT=<racine de travail> replay-build \
  --map "<carte>" --facts <travail>/facts/<short8>.facts.json <matchId complet>

# 3. la comparaison, axe par axe
replay-diff -ancien <ancien>.json -nouveau <recuit>.json [-json rapport.json]
```

Racine de travail : un dossier a soi ou seuls `data/cache/film_chunks`,
`data/cache/film_manifests`, `data/cache/mvar` et `data/titles` sont des jonctions vers le
checkout principal ; `data/cache/replays` doit y etre un VRAI dossier, sans quoi la cuisson
ecrase le parc de reference.

Artefacts de la mesure (hors depot, repertoire de travail de la session) : `reference/` (161
anciens artefacts, 306 Mio), `apres/` (119 re-cuits), `rapport/` (161 rapports JSON de paire),
`inventaire.tsv`, `cuisson_journal.tsv`, `diffs.tsv` (26 098 ecarts a plat).
