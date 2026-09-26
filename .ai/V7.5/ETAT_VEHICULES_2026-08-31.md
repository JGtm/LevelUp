# ETAT DE L'ART — VEHICULES (2026-08-31)

> Repond aux quatre questions posees par l'utilisateur : « lesquels sont sur quelle map, vehicule
> aleatoires ou non, points de spawns et cooldown ». Tout ce qui suit est MESURE — corpus de
> **224 fichiers `.mvar` sur 121 cartes**, plus les tables d'enumeration du binaire du jeu.

## 0. Le resultat qui commande tous les autres

**Sur les cartes OFFICIELLES, les vehicules ne sont PAS dans le fichier de variante de carte.**

Le corpus de 121 cartes contient Fragmentation, Deadlock, Highpower, Oasis, Behemoth — toutes les
grandes cartes BTB, celles qui portent Warthogs, Wasps, Scorpions et Mongooses par paquets. Leur
`.mvar` ne porte **qu'une tourelle**. Les seules cartes dont le `.mvar` contient de vrais vehicules
sont des cartes FORGE : `Starboard` (banshee, scorpion, warthog, warthog_razorback, wasp),
`Salvation` (auto_turret, brute_chopper, mongoose), `Cliffside` (ghost), `Goliath` (wasp).

C'est le meme mecanisme, deja etabli, que l'extinction des socles d'arme : **le fichier POSE, mais
une carte DEV pose ses objets dans le scenario de base du `.module`**, pas dans la variante. La
question etait ouverte et notee « decisive » — elle est **tranchee, par la negative** : la voie
`.mvar` ne repondra jamais « quels vehicules sur quelle carte officielle ».

## 1. LESQUELS — le catalogue

### 1.a Ce que le binaire declare

Trois enumerations, lues dans `HaloInfinite.exe` (offsets ~62 519 800 et 63 064 080) :

- **`MP_VEHICLE_IDENTIFIER`** — les identifiants de vehicule. Membres lisibles dans le bloc :
  `Gungoose`, `Razorback`, `Rockethog` (les autres noms sont mutualises ailleurs dans le pool).
- **`MP_VEHICLE_CLASS`** — **`Support`, `Duelist`, `Cavalry`, `Siege`**, chacune avec 3 emplacements
  `Unused*` reserves.
- **`MP_VEHICLE_TERRAIN_TYPE`** — `Land`, `Air`.

### 1.b Ce que la palette Forge expose : 21 `type_id`, 17 nommes

`.ai/V7.5/dumps/forge_zones/palette_complete_groupes.csv` donne 21 `type_id` de groupe de tag
`vehi`. 15 portaient deja un nom craque ; **2 de plus ont ete craques le 2026-08-31** par murmur3
x86_32 seed 0, avec un controle 15/15 sur les noms deja connus
(`internal/analysis/replay/mapvar/vehicules_noms_test.go`) :

| type_id | nom | | type_id | nom |
|---|---|---|---|---|
| -1825803927 | `shade_turret` | | 60452899 | `brute_chopper` |
| -1718495603 | `banshee` | | 83469709 | `auto_turret` |
| -870843776 | `mongoose` | | 666920711 | `ghost` |
| -522135259 | `wasp` | | 1133144079 | `warthog_gauss` |
| -269578988 | `unsc_turret` | | 1304071901 | `plasma_turret` |
| -262750720 | `warthog` | | 1503350133 | `scorpion` |
| -188587954 | `wraith` | | -411259918 | `phantom` |
| **-105823600** | **`warthog_razorback`** (craque) | | 223996207 | `falcon` |
| **2128426546** | **`mongoose_gungoose`** (craque) | | | |

`warthog_razorback` confirme l'hypothese posee sur l'emprise : ce type avait exactement celle du
Warthog (2,244 x 1,014 x 0,832 wu).

**4 restent anonymes**, vivier de 108 noms epuise : `1029649325` (le plus pose des inconnus,
22 exemplaires / 10 cartes), `-1773333388`, `-2002047233`, `1161655938`. Les deux vehicules du
roster absents du nommage sont **Rockethog** et **Komodo** — candidats naturels, non confirmes.

## 2. SUR QUELLE MAP — l'inventaire mesure

**212 vehicules poses sur 121 cartes.** Ecrasante majorite de TOURELLES :

| vehicule | exemplaires | cartes | | vehicule | exemplaires | cartes |
|---|---:|---:|---|---|---:|---:|
| `unsc_turret` | 105 | 38 | | `mongoose` | 5 | 3 |
| `shade_turret` | 30 | 13 | | `warthog` | 4 | 2 |
| `?1029649325` | 22 | 10 | | `banshee` | 2 | 2 |
| `auto_turret` | 11 | 3 | | `scorpion` | 2 | 2 |
| `plasma_turret` | 9 | 5 | | `warthog_razorback` | 2 | 2 |
| `brute_chopper` | 8 | 2 | | `ghost` | 1 | 1 |
| `phantom` | 6 | 1 | | | | |
| `wasp` | 5 | 3 | | | | |

Par carte du catalogue :

| carte | vehicules au `.mvar` |
|---|---|
| Starboard | banshee, scorpion, warthog, warthog_razorback, wasp |
| Salvation | auto_turret, brute_chopper, mongoose |
| Fragmentation Heavies | plasma_turret, shade_turret, unsc_turret |
| Breaker Heavies | plasma_turret, shade_turret |
| Deadlock Heavies · Oasis Heavies | ?1029649325, shade_turret |
| Snowbound | auto_turret, unsc_turret |
| Cliffside | ghost · Goliath : wasp |
| Deadlock | shade_turret |
| Empyrean · Scarr · Takamanohara | ?1029649325 |
| Behemoth, Chasm, Cliffhanger, Command, Forest (+ Ranked), Fortitude, **Fragmentation**, High Ground, **Highpower** (+ Heavies, Sentry Defense), Isolation, Live Fire (+ Ranked), Oasis Sentry Defense, Obituary, Refuge, Smallhalla, The Pit | `unsc_turret` seule |

**Fragmentation avec une seule tourelle**, c'est le negatif du §0 : ses vehicules sont ailleurs.

## 3. ALEATOIRES OU NON — deux niveaux, et il faut les distinguer

**Au niveau de l'EMPLACEMENT : non.** Un emplacement de vehicule **NOMME son vehicule** — son
`type_id` designe le modele. C'est la difference avec les socles d'arme, qui sont generiques (le
meme socle porte l'epee ou le marteau selon le match).

**Au niveau du MODE : oui, le levier existe.** La structure de preset de spawner d'objet du binaire
porte, cote a cote :

    forceRandomWeapon   forceRandomEquipment   forceRandomVehicle
    vehicles   weapons   terrainFilters   airVehiclePrerequisiteCount
    seedSequenceKey   seedSequenceOverrides   useSecondaryItemSpawnerPresets
    enableStaticSelection   managedDistribution   distributionInterval
    distributionVariance   useEscalatingCategoryWeights   perClassOverrides

`forceRandomVehicle` est donc le pendant exact de `forceRandomWeapon`, dans la meme structure : la
randomisation est un reglage de VARIANTE DE MODE, jamais de carte. Doctrine habituelle — le fichier
POSE, le mode ALLUME.

**Reserve** : `enableStaticSelection` / `seedSequenceKey` sont des reglages de spawner d'ARME, pas de
vehicule ; ne pas les transposer.

## 4. POINTS DE SPAWN

Le `.mvar` donne la position monde au centimetre (`x`, `y`, `z`), dans le MEME repere que les
positions du rejeu 2D — celui que `map_objectives.json` publie deja. Les 212 emplacements sont donc
posables sur la carte sans aucune conversion. Ils portent aussi leur index d'equipe (`equipe=-1`
partout dans le corpus : aucun vehicule n'est attribue a un camp) et leurs etiquettes de mode.

## 5. COOLDOWN — mesure, et par EMPLACEMENT

Le sac de gameplay de l'objet porte **deux delais**, presents sur 212/212 et 206/212 emplacements :

| chemin | present | valeurs observees |
|---|---|---|
| `#8/1[0]/#4` | 212/212 | 1, 20, 30, 45, 60, 88, 120, 180, 240 s |
| `#8/1[0]/#5` | 206/212 | 30, 90, ... (toujours <= #4 sur les echantillons lus) |

La table de champs du `.mvar` dans le binaire les nomme, dans cet ordre :

    m_doesNotRespawn   m_canSpawnOnBipeds   m_uniqueSpawn
    m_abandonedTime    m_respawnTime        m_initialSpawnDelay

L'ordre et la relation `#4 >= #5` designent **`#4 = m_respawnTime`** et
**`#5 = m_initialSpawnDelay`**. La correspondance d'indice n'est pas prouvee autrement que par cette
adjacence : a confirmer si une decision en depend.

**Le delai est un reglage d'AUTEUR, par emplacement, pas une constante de classe** :
`unsc_turret` porte 30, 60, 88, 120 et 180 s selon la carte. Une structure de classe s'y devine
neanmoins — banshee/scorpion/wasp a 120 s, warthog/razorback/ghost a 60 s, mongoose/chopper a 30 s —
coherente avec les quatre classes `MP_VEHICLE_CLASS`, mais ce sont des cartes Forge : ce sont les
choix de leurs auteurs, pas les defauts du mode.

Le binaire porte aussi `m_abandonedTime` + `m_abandonmentTimerAlwaysActive` : **un vehicule laisse
seul se recycle**. Jamais mesure.

## 6. CE QUI RESTE

1. **Lire le scenario (`scnr`) des `.module`** — c'est LA voie pour les cartes officielles, et le
   §0 la designe sans ambiguite. `internal/himap` lit deja les modules (index, BSP, geometrie,
   callouts) ; les placements d'objets de scenario ne sont pas encore decodes. Chantier a ouvrir.
2. **Les 4 `type_id` `vehi` anonymes**, dont `1029649325` (22 exemplaires / 10 cartes) — le vivier
   de noms est epuise, il faut une autre source (chaines du binaire, ou le pool `hsc*` en clair).
3. **Confirmer l'indice `#4` = `m_respawnTime`** autrement que par l'adjacence des noms de champ.
4. **Le film peut-il dire qu'un vehicule apparait ou meurt ?** Deux acquis non exploites : la mort
   d'un vehicule est un composant, et la marche de `killsource/walk.go` collecte deja TOUS les
   records `Mort=1` avant de les filtrer sur la plage bipede — les vehicules (`ti=40`, slots
   768-1023) sont lus puis jetes. L'apparition, elle, n'est bornable qu'a +/-20 s.

## 7. Pieces

- `internal/analysis/replay/mapvar/vehicules_corpus_test.go` — l'inventaire (garde `VEHI_MVAR_DIR`).
- `internal/analysis/replay/mapvar/vehicules_noms_test.go` — le craquage, controle 15/15.
- Corpus : `mapobj-build --player <GT> --all --dry-run --save-mvar <dir>` — 121 cartes le
  2026-08-31, 2 echecs 404 (Houseki, Shogun). Les fichiers ne sont pas versionnes.
- `.ai/V7.5/dumps/forge_zones/palette_complete_groupes.csv` et `palette_noms.csv`.

## 7. OU SONT LES VEHICULES DES CARTES OFFICIELLES — la reponse, apres trois portes (2026-08-31)

La question ouverte du matin (« les vehicules ne sont pas dans le `.mvar` des cartes DEV — ou
sont-ils ? ») est tranchee. Trois endroits ont ete ouverts, et le troisieme donne la reponse par
elimination.

**PORTE 1 — le fichier de carte (`.mvar`).** Balayage de 224 fichiers sur 121 cartes : 212
vehicules poses, ecrasante majorite de tourelles. **Fragmentation, la carte BTB de reference,
porte 501 objets, 28 `type_id` distincts — et exactement DEUX `unsc_turret`.** Ses autres types
sont du groupe `scen`, `bloc`, `mach` et `weap` (8 placements d'arme). Aucun Warthog, aucun Wasp,
aucun Scorpion.

**PORTE 2 — le scenario (`levl`) du module de carte.** La carte de ses blocs, lue le 2026-08-31
(navigation de struct-table, la meme que les zones nommees) : 40 blocs enfants sur Fragmentation,
chacun nomme par le GROUPE DE TAG qu'il reference —

| champ | Fragmentation | Aquarius | groupe reference |
|---|---:|---:|---|
| 0x0060 | 7 | 14 | `scen` (decor) |
| 0x00d8 | 29 | 20 | `mach` (machines) |
| 0x0150 | 76 | 118 | `bloc` |
| 0x01b4 | 55 | — | `lens` |
| 0x01c8 | 87 | 150 | `licn` |
| 0x01dc | 1709 | 361 | `lsnd` |
| 0x01f0 | 343 | 433 | `effe` |
| 0x0628 | 102 | 913 | `ligr` |
| 0x0650 | 59 | 76 | `bitm` |

**AUCUN bloc ne reference `vehi` ni `bipd`.** Le scenario porte le decor, l'eclairage, le son, les
effets et les decalques — pas les objets de jeu.

**PORTE 3 — les compositions.** `deploy/any/compositions/` ne contient que du narratif.

**LA REPONSE, ET ELLE RECOUPE LA RE DU BINAIRE.** Les vehicules des cartes officielles ne sont
poses NULLE PART dans les donnees de carte : ils sont **spawnes par le MODE**. C'est exactement ce
que decrit la structure de preset de spawner d'objet relevee le matin dans le binaire —
`vehicles` (une LISTE), `terrainFilters`, `airVehiclePrerequisiteCount`, `forceRandomVehicle`,
`seedSequenceOverrides`. Le meme mecanisme que les socles d'arme : **la carte pose un emplacement
GENERIQUE, le mode decide ce qui y apparait.**

Ce qui explique tout le reste :

- une carte FORGE nomme son vehicule (`Starboard` : banshee, scorpion, warthog, razorback, wasp)
  parce que son AUTEUR l'a choisi objet par objet ;
- une carte DEV n'en nomme aucun, parce que c'est la variante de mode qui les distribue ;
- les tourelles, elles, SONT placees a la carte — ce sont du mobilier, pas du butin de mode.

**Consequence pour le chantier** : « quels vehicules sur quelle carte » n'a pas de reponse
dans les fichiers de CARTE. La question se pose desormais a la VARIANTE DE MODE, et c'est un
autre corpus.

## 8. LA GRAMMAIRE D'UN PLACEMENT DE SCENARIO, relevee au passage

Elle vaut pour tous les blocs ci-dessus, et elle est neuve :

	+0x00  8 o   identifiant unique du placement
	+0x0c  3f    POSITION monde
	+0x18  3f    ORIENTATION
	+0x60  u32   GlobalID du tag reference
	+0x68  u32   GlobalID (second)
	+0x6c  4 o   fourCC du GROUPE de tag, ECRIT A L'ENVERS (« snel » = `lens`)

Le fourCC inverse est ce qui rend le scenario auto-descriptif : un bloc DIT ce qu'il pose, sans
qu'on ait a deviner l'ordre historique des blocs de scenario Halo. Instrument :
`internal/himap/scenario_placements_gamefiles_test.go`.
