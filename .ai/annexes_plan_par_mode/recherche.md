# Recherche : plan des murs par mode et murs invisibles (préparation 8.3.1)

Date : 2026-11-18. Statut : recherche en lecture seule, aucune modification du dépôt ni des briques.
Sondes (module Go jetable, briques publiées `halo-infinite-files-reader` v1.2.0,
`halo-infinite-map-images` v1.4.0, `halo-infinite-film-decoder` v1.4.1, aucun `replace`) :
`C:\Users\Guillaume\AppData\Local\Temp\claude\17871fa88c29004d\63a04594-e948-4485-9732-90ff5a088fa7\scratchpad\plan_par_mode\`
(`cmd/` : une sonde par dossier ; `out/` : sorties ; `mvar_natifs/` : variantes natives rapatriées).
Build : `GOWORK=off GOFLAGS=-mod=mod GOPROXY=off go build -o bin/<sonde>.exe ./cmd/<sonde>`.

Unités : celles du maillage et des positions du rejeu (unité moteur, 10 pieds). Le « world unit »
de Den vaut 0,1 unité moteur ; les dimensions du nom d'un bloqueur Forge (« 16x2x8 ») sont dans
cette unité de Den.

## Résumé en six réponses

1. **Oui, les objets d'une variante portent des filtres de mode**, et ce sont eux qui ferment
   une carte native selon le mode : étiquettes `<mode>_include` / `<mode>_exclude` (hash murmur3
   déjà lu par `mapvar`, champ `#8/#0[0]/#9`). Sur Streets et Live Fire, les portes, grilles,
   barrières et bloqueurs de joueur étiquetés `infection_exclude` ferment les ruelles et
   l'intérieur du fond hors Infection.
2. **Carte native : la fermeture par mode vient de la variante de carte publiée (`map.mvar`
   de l'asset), pas des fichiers du niveau.** Le tag `levl` porte les volumes de mort du
   niveau (trouvés sur disque, racine +0x3bc) mais tous dans la couche racine ; le `sddt` ne
   dépend pas du mode ; `ModeRegions` / `ModeNodes` sont des régions et noeuds de modèle de
   rendu (véhicules), sans rapport avec les modes de jeu.
3. **Les murs invisibles vivent dans le bloc des instances physiques du `sbsp`** (racine
   +0x1e4, pas 192) : les instances sans instance de rendu (`m_guid` nul), posées à l'origine,
   dont le tag `scgt` porte le matériau `default_material`. Leur géométrie est un polyèdre
   convexe dans le blob Havok du `scgt` et se décode (boîte exacte, sommets u16 au 1/512).
   931 volumes sur 16 des 31 niveaux natifs. Notre lecteur ne lit ni ce bloc ni les `scgt`.
4. **Le document de rejeu ne porte ni le mode, ni la variante de carte, ni les objets
   apparus.** Le film (`chunk_00`) porte l'asset de la variante de carte et le nom de la
   variante de partie, lus par `decfilm.ReadMatchInfo`. Les positions valident très bien un
   plan « hors Infection » (0 point au-delà d'un fermant sur 54 matchs par carte) ; elles ne
   valident aucun plan Infection (0 film Infection sur 1 590).
5. **On retrouve les deux fermetures que Den cite.** Live Fire hors Infection : 41,4 unités
   carrées fermées (4,6 %), dont 35,5 derrière la porte lourde du fond. Streets hors
   Infection : 70,4 (9,0 %), deux ruelles (58,4 à l'est, 12,0 à l'ouest). En Infection ces
   zones sont ouvertes.
6. **Méthode recommandée** : fermants de la variante filtrés par étiquette de mode, coupe du
   maillage de navigation, connexité depuis les points d'apparition du mode, contrôle par les
   positions jouées ; murs invisibles et volumes de mort ajoutés comme couche commune à tous
   les modes. Pas de modèle de déplacement complet : deux règles de lien suffisent (détail §6).

## 1. Variantes de carte : étiquettes de mode et objets qui ferment

### 1.1 Ce que porte un objet

Arbre Bond d'un objet (`root/3[]`), lu par `cmd/mvarraw` :

| Champ | Contenu | Lu par `mapvar` v1.2.0 |
|---|---|---|
| `#2/#0` | identifiant du type Forge | oui (`TypeID`) |
| `#3`, `#4`, `#5` | position, haut, avant | oui |
| `#7` | **type d'objet du jeu** (`m_type`) : 1 défaut, 3 téléporteur, 4 lieu nommé, 7 point d'apparition, 8 zone d'apparition, 14 volume de sécurité, 15 volume de mort, 21 primitive Forge, 28 coupe de maillage (« Nav Cutter »), 29 sonde de lumière | oui, sous le nom `Flags` (non interprété) |
| `#8/#0[0]/#0[0]` | forme (`m_boundary`) : boîte ou cylindre, 16.16 | oui (`ShapeRaw`) |
| `#8/#0[0]/#9[]/#0` | **étiquettes** (hash murmur3 du nom) | oui (`Labels`) |
| `#8/#23[0]/#0` | **échelle par axe d'une primitive Forge** (vec3) | **non** |

Noms des types Forge : table `FORGE_OBJECTS` de `libfilmshell/src/game/forge_data.rs` (Den,
MIT, 4 228 noms tirés du manifeste Forge du jeu), convertie en
`plan_par_mode/data/forge_names.json` par `forge_names.cjs`. Elle est croisée ici par le
`m_type` : les 1 182 objets de `m_type` 15 des 94 cartes Forge s'appellent tous « Kill
Volume » (242) ou « Soft Kill Volume » (940), les 95 de `m_type` 14 « Safe Volume » (14) ou
« Soft Kill Safe Volume » (81).

Règle d'apparition (lue dans `repo/src/host/src/limits.rs`, fonction `spawns`, et son test) :
un objet apparaît dans un mode s'il ne porte pas `<mode>_exclude`, et, dès qu'il porte une
étiquette `*_include`, s'il porte `<mode>_include`.

### 1.2 Mesure sur les cartes Forge du catalogue

Sonde `cmd/mvarcensus` sur `.ai/re_dump/mapvar/` : 114 fichiers, **94 lus, 20 vides** (0 octet :
ce sont les 20 variantes natives du dossier, `aquarius_ctf_aquarius.mvar`,
`streets_-_ranked_sgh_streets.mvar`, etc. ; le dossier ne contient donc que du Forge).

Objets qui ferment, sur les 94 cartes Forge :

| Famille | `m_type` | Objets | Cartes | Forme lisible | Étiquettes de mode |
|---|---|---|---|---|---|
| Player Blocker (taille fixe dans le nom) | 1 | 6 344 | 62 | non (dimensions dans le nom) | 2 `infection_include` |
| Scalable Player / Universal / Team Blocker | 21 | 5 407 | 37 à 49 selon le type | échelle `#8/#23` | aucune |
| Universal Blocker | 1 | 1 310 | 56 | nom | 7 `firefight_include` |
| Team Blocker | 1 | 1 802 | 31 | nom | `firefight_*`, `stockpile_include`, `infection_*` |
| Soft Kill Volume | 15 | 940 | 66 | oui, 940/940 | 3 `ctf_multi_include`, 3 `infection_include`, 6 `stockpile_include`, 2 `firefight_exclude` |
| Kill Volume | 15 | 242 | 68 | oui, 242/242 | 1 `firefight_exclude` |
| Soft Kill Safe Volume / Safe Volume | 14 | 81 / 14 | 15 / 9 | oui, 95/95 | aucune |
| Portes, grilles, barrières (30 types et plus) | 21 ou 1 | plusieurs milliers | variable | modèle de l'objet | rares |
| Nav Cutter | 28 | 1 184 | 79 | oui | aucune |

Fermants portant une étiquette de mode (hors `minigame`) : **13 cartes Forge sur 94**. Exemples :
Banished Narrows (portes « Banished Vehicle Door » `[ctf_include oddball_include]` : fermées
en CTF et Oddball seulement), Smallhalla (Soft Kill `[ctf_multi_include infection_include]`
et `[stockpile_include]`), Isolation (2 Player Blocker et 1 Soft Kill `infection_include`),
Thunderhead (Team Blocker `stockpile_include`), quatre cartes à portes `infection_exclude`.
Les 81 autres ont des fermants sans étiquette : leur plan est le même dans tous les modes.

Trois cartes Forge regardées de près (`mvarcensus dump`, puis `cmd/forgeplan` contre le
`navmesh.blob` du dépôt) :

| Carte | Fermants | Maillage en volume de mort non couvert par une sécurité | Maillage sous un bloqueur à taille fixe |
|---|---|---|---|
| Solitude - Ranked | 527 Player Blocker, 12 Projectile Blocker, 16 Nav Cutter, 0 volume de mort | 0 % | 1,3 % |
| Empyrean | 414 Player Blocker, 17 Universal, 2 Kill Volume | 0 % | 0,8 % |
| Fortress | 276 Scalable Player Blocker, 66 Universal, 1 Kill Volume, 1 Safe Volume | 0 % | 0,3 % |
| Argyle | 283 Player Blocker, 6 Soft Kill, 2 Kill Volume | **20,3 %** (517 sur 2 546) | 3,5 % |
| Smallhalla | 18 volumes de mort, 3 de sécurité | 6,0 % | 0 % |

Lecture : sur Forge, le maillage publié est déjà presque vide sous les bloqueurs (0,3 à 3,5 %) ;
ce sont les volumes de mort qui retirent du sol, et seulement sur certaines cartes.

### 1.3 Mesure sur les variantes des cartes natives

Les variantes natives ne sont pas en dépôt (fichiers vides). Onze ont été rapatriées par le
chemin anonyme de `cmd/mapnav-fetch` (page publique de l'asset, puis stockage), sonde
`cmd/mvarfetch`, dans `plan_par_mode/mvar_natifs/`. Un asset natif publie deux fichiers :
`map.mvar` (l'actuel, celui que `map_objectives.json` référence) et un second au nom du niveau
(`sgh_streets.mvar`, identique entre Streets et Streets Infection : un reliquat).

| Variante (asset) | Objets | Fermants et étiquettes |
|---|---|---|
| Live Fire (`6c01f693`) | 528 | 1 UNSC Heavy Door `infection_exclude` (23,7 ; 25,3), 1 Chain link Fence Gate `infection_exclude` (17,2 ; 33,5), 64 Player Blocker 4x4x4 `infection_include` |
| Live Fire - Ranked (`309253f8`) | 270 | la même porte et la même grille, `infection_exclude` |
| Streets (`9c7b0b0f`) | 567 | `infection_exclude` : 4 UNSC Heavy Door, 3 Constuction Barrier, 1 Iron Gate, 7 Player Blocker (32x32x16, 16x16x16, 16x4x4) ; `infection_include` : 72 Player Blocker 2x2x2 ; 1 Player Blocker sans étiquette |
| Streets Infection (`33dfff37`) | 560 | les mêmes objets et les mêmes étiquettes (le filtre fait le travail, pas le fichier) |
| Streets - Ranked (`e23ea388`) | 270 | les 15 fermants `infection_exclude` de Streets, plus 7 Scalable Player Blocker sans étiquette (z de 5,8 à 7,6 : des toits) |
| Recharge / Recharge - Ranked | 548 / 305 | 91 Player Blocker `infection_include` / aucun fermant |
| Aquarius / Aquarius - Ranked | 369 / 236 | 86 Player Blocker `infection_include`, 2 Projectile Blocker `infection_exclude` / les 2 Projectile Blocker |
| Bazaar (`3e1e4cec`) | 993 | 61 Player Blocker `infection_include`, **1 Iron Gate `infection_include`** (fermée en Infection seulement), 4 portes sans étiquette |
| Cliffhanger (`5324364b`) | 453 | 105 Player Blocker `infection_include` |

Sur ces onze variantes, la seule famille d'étiquette portée par un fermant est `infection`.
Les grappes de petits bloqueurs `infection_include` (6 à 8 objets au même endroit) ferment des
points, pas des zones (mesure §5 : 2 unités carrées au total).

### 1.4 Où se lisent forme et position

- Volumes de mort et de sécurité : position + `Shape()` (boîte ou cylindre orienté), déjà en
  production. « Mort immédiate » ou « compte à rebours » se lit au nom du type (Kill / Soft Kill).
- Bloqueurs à taille fixe : position, axes, dimensions du nom en dixièmes d'unité. Cohérent sur
  Streets (le 32x32x16 de 3,2 unités couvre les deux barrières posées à 1,4 unité l'une de
  l'autre), non prouvé sur le modèle de collision : l'ancrage (centre ou base) et l'ordre des
  axes restent à établir sur le tag de l'objet.
- Bloqueurs « Scalable » : même chose, multipliée par l'échelle `#8/#23[0]/#0` ; la taille de
  base est à lire sur le modèle de collision de l'objet Forge.
- Portes, grilles, barrières : pas de forme dans la variante ; emprise = celle du modèle de
  l'objet (chaîne `food` vers modèle, déjà suivie par `himap.modeleParType` pour le rendu).

## 2. Cartes natives : ce qui ferme une zone selon le mode

Mesure sur Live Fire (`sgh_interlock`, géométrie dans `globals/common-rtx-new.module`) et
Streets (`sgh_streets`).

| Piste | Constat | Preuve |
|---|---|---|
| Variante de carte par mode | C'est le mécanisme : §1.3 | `mvarcensus dump` sur `mvar_natifs/` |
| Tag `levl`, volumes déclencheurs | Racine +0x3bc, enregistrement de 208 o. Octet +0x16 : 0 générique, **1 mort**, 6 lieu nommé. Live Fire : 116 volumes dont 5 de mort (indices de mort 0 à 4) ; Streets : 55 volumes, **0 de mort** | `cmd/levlprobe` ; layout repris des notes publiques de Den (`repo/src/host/src/navmesh.rs`, `LevelVolume`) et vérifié sur disque |
| Tag `levl`, couches | Racine +0xa64, 188 o par couche (180 sur Live Fire, 96 sur Streets), table des emplacements en +0xa78 (45 blocs découpés par couche). **Les volumes déclencheurs sont tous dans la couche 0** (`level`) sur les deux cartes | `cmd/levlcouches` : emplacement 33 = bloc 0x3bc, couche 0 = `[0,116)` et `[0,55)` |
| Couches par mode | Non établi. Seul le nom de la couche 0 est résolu (`level`, murmur3) ; les autres couches découpent des blocs d'objets non décodés (0x150, 0x1f0, 0x54c, 0x628) | idem |
| `sddt` | Coquille de survie et eau, par niveau, sans notion de mode (acquis du 2026-08-10) | `himap/sddt.go` |
| `ModeRegions`, `ModeNodes` | Régions et noeuds du `render_model` (permutations de véhicule, squelette) : aucun rapport avec les modes de jeu | en-têtes de `himap/regions.go` et `noeuds.go` |
| Film | Le film porte un composant « volumes de mort désactivés » (un bit par indice de mort) : un mode ou un script peut éteindre un volume de mort du niveau. Notre grammaire le consomme sans le publier | `internal/grammar/components_moteur_de_partie.go` du décodeur ; `libfilmshell/src/components.rs` |

Le niveau `sgh_interlock` est aussi celui du tutoriel de l'Académie (même `levl` : 116 volumes,
5 de mort, 180 couches dans `academy_tutorial`) : c'est l'« autre partie du niveau » que Den
écarte, et que `ChoisitBSP` écarte déjà chez nous.

Recensement des 31 niveaux natifs (boucle `levlprobe` + `murs`, variante `ds`) : volumes de
mort du `levl` présents sur 28 niveaux (54 sur Deadlock, 48 sur Oasis, 47 sur Highpower, 45
sur Behemoth, 12 sur chaque canevas Forge ; 0 sur Aquarius, Bazaar, Streets).

## 3. Murs invisibles

### 3.1 Où ils vivent

Bloc « instanced physics instances » du `sbsp` : racine +0x1e4, pas 192 (acquis du 2026-08-09,
confirmé). Champs utiles : référence `scgt` (+0x08), position (+0x6c), `m_typeMask` (+0x78),
`m_guid` (+0x7c).

Le critère qui sépare n'est pas le masque (`0x0f`, `0x0d`, `0x0e` : le drapeau InvisibleWall
est levé partout, porte fermée le 2026-08-09) mais l'**absence d'instance de rendu** :

| Carte | Instances physiques | `m_guid` nul | dont `scgt` à `default_material` | Position et échelle |
|---|---|---|---|---|
| Streets (bsp 1) | 5 038 | 74 | 74 | toutes à l'origine, échelle 1, en fin de liste |
| Live Fire (4 bsp du module commun) | 15 253 | 90 | 90 | idem (61 + 3 + 26) |

Le tag `scgt` : racine de 0x58 o ; +0x10 liste de matériaux (StringId ; `0x51710d2b` =
murmur3 de `default_material`) ; +0x28 descripteur de forme ; puis un blob Havok `TAG0`
(`SDKV20220100`). Den nomme ces murs par une couche (`environment_invisible_wall`) : ce nom
n'existe dans aucun tag des modules sondés (recherche d'octets à tout alignement,
`cmd/bytescan`), c'est vraisemblablement un nom de couche de collision de l'exécutable.

### 3.2 Peut-on les extraire en géométrie : oui

Section `DATA` du blob Havok (témoin : `scgt` `236c91a8` de Streets, `cmd/scgtdump`) :

- noeuds d'arbre SIMD de 0x80 o : six rangées de quatre flottants (min x, max x, min y, max y,
  min z, max z), sentinelles `±FLT_MAX` pour un enfant absent. Boîte lue : x de -7,951 à
  -6,195, y de -6,201 à -6,016, z de 3,615 à 13,482 ;
- faces : six quadrilatères d'indices (`04070100 05060704 ...`) ;
- sommets : triplets u16, **pas de 1/512, relatifs au minimum de la boîte** (le facteur 512
  est écrit en +0x50) : huit sommets, étendues 899/512 = 1,756, 95/512 = 0,186,
  5 052/512 = 9,867, exactement les côtés de la boîte.

Sonde `cmd/murs` (boîtes par l'arbre, sommets par recherche d'un tableau qui remplit la boîte) :
Streets 74/74, Live Fire 90/90, et sur les 31 niveaux natifs **931 volumes, 927 avec sommets**
(16 niveaux en portent : Bazaar 159, Launch Site 122, Recharge 87, Scarr 75, Highpower 75,
Streets 74, Breaker 56, Oasis 43, Aquarius 38, Deadlock 30...). La recherche de sonde peut
ramasser des triplets parasites : le lot devra lire les tableaux par leurs pointeurs relatifs.
Aucun volume trouvé sur Chasm, Forbidden, Illusion, Fragmentation, Behemoth : à vérifier
(absence réelle, ou murs portés autrement).

Ce sont des **volumes pleins**, pas des parois : hauteur médiane 6,6 unités sur Streets,
beaucoup partent de z = 6 ou 7 (au-dessus des toits) et montent à 13 ou 14.

### 3.3 Effet sur le plan

Sonde `cmd/planmode` : un point du maillage est « sous un mur » si le corps d'un joueur debout
(de 0,1 à 0,7 unité au-dessus du sol) coupe le volume.

| Carte | Volumes qui touchent le maillage | Aire du maillage dedans | Positions jouées dedans |
|---|---|---|---|
| Streets | 15 sur 74 | 114,8 (12,2 % du maillage) | 0,061 % (2 460 points sur 4,0 millions) |
| Live Fire | 4 sur 90 | 0,5 (0,1 %) | 0,003 % |

Sur Streets ces 12,2 % sont du sol que le maillage porte (toits, arrière de façades) et où
l'on ne va pas. Les volumes de mort du niveau, eux, ne couvrent aucun sol du maillage sur
Live Fire (ils sont sous la carte).

### 3.4 Notre lecteur les ignore, et pourquoi

- `himap` lit les instances de **rendu** du `sbsp` (`rtgo`) ; le bloc physique +0x1e4 n'est lu
  par aucune fonction publiée, et aucun `scgt` n'est décodé (format jugé « non décodé » le
  2026-08-09 : la lecture en flottants bruts échouait, les sommets sont en u16).
- Un mur invisible n'a par définition pas d'instance de rendu : la chaîne de fond ne peut pas
  le voir.
- Le plan actuel (`plan.edges` du sidecar) est le bord du maillage de navigation retenu par
  `ComposantesRetenues` : il ne connaît ni les volumes ni les objets de la variante.
- Côté Forge, les bloqueurs n'ont pas de modèle de rendu : `poseObjetsForge` les compte en
  « objets sans modèle ».

## 4. Films et rejeux

| Donnée | Document de rejeu (`data/cache/replays/halo_infinite/<court>.json.gz`, schéma 125) | Film |
|---|---|---|
| Mode, variante de partie | absent (clés : `tracks`, `geometry`, `structure`, `rules`, `identity`, `roster`...) | `chunk_00` : nom de la variante de partie, lu par `decfilm.ReadMatchInfo`, persisté dans les faits |
| Variante de carte | absente | `chunk_00` : asset et version de la variante de carte |
| Objets apparus (bloqueurs) | absents (`geometry` : pièces Forge dessinées, sans étiquette) | non portés ; ils viennent de la variante |
| Volumes de mort éteints | absents | composant lu puis jeté par la grammaire |
| Positions | `tracks[].points[]` (t, x, y, z) | idem |

Sonde `cmd/matchinfo` (lecture du seul `chunk_00` de chaque film de l'archive) : 1 590 films
lus, 7 illisibles, sortie `out/matchinfo.tsv`. 112 variantes de carte distinctes : 13 avec 30
films ou plus, 44 entre 10 et 29, 55 sous 10. Couples (carte, variante de partie) : 389, dont
18 seulement avec 10 films ou plus. **Aucun film Infection, aucun Firefight.**

Validation par les positions (`positions.cjs`, cellules de 0,25 unité, puis `cmd/planmode`) :

| Carte | Matchs | Points | Sur le plan du mode | Au-delà d'un fermant | Contre un fermant | Sol que la marche seule n'atteint pas | Hors maillage |
|---|---|---|---|---|---|---|---|
| Streets, hors Infection | 54 | 4 007 480 | 95,31 % | **0** | 0,43 % | 1,75 % (22 faces, aire 38) | 2,45 % |
| Live Fire, hors Infection | 54 | 3 921 218 | 95,98 % | **0** | 0,22 % | 1,51 % (24 faces, aire 15) | 2,29 % |

Couverture : 96,3 % (Streets) et 99,1 % (Live Fire) de l'aire du plan du mode est à moins de
0,5 unité d'une position jouée.

Ce que cela dit :

- les positions suffisent à **valider une fermeture** (zéro point derrière) dès une cinquantaine
  de matchs, et à **refuser une fausse fermeture** (une zone fermée qui porterait des points) ;
- elles ne suffisent pas à **construire** un plan par mode : 18 couples (carte, mode) seulement
  ont 10 films, et l'Infection n'a aucun film ;
- 0,2 à 0,4 % des points tombent dans le rayon d'un fermant : c'est l'imprécision des disques
  de la sonde (joueur collé à la porte), pas une traversée ;
- 1,5 à 1,8 % des points sont sur du sol du maillage que la marche seule ne relie pas aux
  points d'apparition (rebords, caisses, toits bas atteints par un saut) : §6.

## 5. Comparaison avec Den, Live Fire et Streets

Méthode de la sonde `cmd/planmode` : maillage natif (`ChargeMaillageNatif`, et sa variante à
géométrie partagée pour Live Fire), échantillonné à 0,1 unité par face ; fermants de la variante
actifs dans le mode (bloqueurs de joueur et universels, portes, grilles, barrières) posés comme
disques (demi-dimension du nom, 0,7 pour une porte) à hauteur de corps ; inondation depuis les
points d'apparition du mode, avec et sans les fermants. « Fermé par le mode » = atteint sans,
plus atteint avec.

| Carte, mode | Fermants actifs | Fermé par le mode | Régions |
|---|---|---|---|
| Live Fire, hors Infection | 2 (porte lourde, grille) | **41,4** (4,6 %) | 35,5 centrée en (26,6 ; 25,4), emprise x de 22,6 à 29,7, y de 22,2 à 29,2 ; 5,9 centrée en (17,4 ; 34,3) |
| Live Fire, Infection | 64 petits bloqueurs | 2,0 (0,2 %) | sept taches de moins d'une unité carrée |
| Streets, hors Infection | 16 (4 portes, 3 barrières, 1 grille, 8 bloqueurs) | **70,4** (9,0 %) | 58,4 à l'est, centrée en (21,6 ; 1,4), x de 15,2 à 26,2 ; 12,0 à l'ouest, centrée en (-16,2 ; 3,5), x de -20,4 à -11,0 |
| Streets, Infection | 73 petits bloqueurs | 2,1 (0,3 %) | huit taches |

Den écrit que Live Fire ferme son « far interior » et Streets ses « side alleys outside
Infection » : les deux sont retrouvés, par les seules étiquettes de la variante. Son fichier
de test (`limits.rs`) cite d'ailleurs l'asset de Streets (`9c7b0b0f...`), un « Player Blocker
16x2x8 » étiqueté `infection_exclude` et une règle « barrier » : c'est la même source.

Ce que sa chaîne fait en plus et que la sonde ne reproduit pas :

| Règle de Den (`dataview.js`) | Chez nous |
|---|---|
| `limiter`, `limit` (mur invisible, bloqueur) | données trouvées (§3) ; coupe à écrire |
| `barrier` (porte, grille, objet posé par les étiquettes) | reproduit (§5) |
| `shut` (sol propre à un mode, fermé à côté d'une barrière) | non reproduit ; inféré chez lui aussi |
| `unreachable` (ni marche, ni saut, ni escalade) | demande son modèle de déplacement (outil privé `tools/access`) |
| `elsewhere` (autre partie du niveau) | déjà traité par `ChoisitBSP` et les composantes ancrées |
| `footprint` (reste plus petit qu'un Spartan) | trivial |

## 6. Méthode recommandée pour la 8.3.1

### 6.1 Principe

Un plan par mode = le plan actuel, moins ce que les fermants du mode coupent des points
d'apparition du mode. On ne recalcule pas le plan : on calcule un **retrait par famille de
mode**, chaque région retirée portant sa raison (quel objet, quelle étiquette).

1. **Entrées par carte** : variante de carte (`map.mvar`, Forge et native), maillage (déjà là),
   et pour les natives les volumes sans rendu (§3) et les volumes de mort du `levl` (§2).
2. **Familles de mode** : les radicaux d'étiquette présents sur les fermants de la carte
   (`infection`, `firefight`, `btb`, `ctf`, `oddball`, `stockpile`...). Une carte sans fermant
   étiqueté a un seul plan. La correspondance mode du match vers radical se déclare dans le
   TOML du titre, jamais en dur.
3. **Coupe** : fermants actifs posés sur le maillage avec leur forme (boîte orientée du nom ou
   de l'échelle, emprise du modèle pour une porte), à hauteur de corps.
4. **Connexité** depuis les points d'apparition actifs dans le mode ; « fermé par le mode » =
   atteint sans les fermants, plus atteint avec.
5. **Couche commune** (tous modes) : sol sous un volume sans rendu, sol dans un volume de mort
   non couvert par une sécurité. À publier d'abord comme calque de contrôle, à retirer du plan
   seulement après contrôle à l'écran.
6. **Garde par les positions** : une région fermée qui porte des positions du mode dans deux
   matchs ou plus n'est pas fermée ; elle sort dans le rapport de la carte.
7. **Rapport par carte** : fermants trouvés, régions retirées et raison, cartes sans donnée.
   C'est la réponse au constat « beaucoup de cartes ne prenaient pas ses modifs » : on sait
   carte par carte ce qui a été appliqué.

Faut-il un modèle de déplacement : non pour les fermetures par mode (0 point derrière un
fermant sur 108 matchs). Les mesures montrent en revanche que la marche seule perd 1,5 à
1,8 % des positions (sol atteint par un saut). Le retrait par mode évite le problème en ne
touchant qu'à ce que les fermants déconnectent ; si la couche commune doit un jour retirer le
« sol jamais atteint », il faudra deux liens, pas plus : chute libre vers un sol plus bas, et
montée bornée (escalade, 0,9 unité chez Den).

### 6.2 Où va le code

| Brique ou dépôt | Ajout |
|---|---|
| `halo-infinite-files-reader` (`mapvar`) | type d'objet nommé (champ `#7`), échelle `#8/#23`, table des noms Forge (crédit Den), règle d'apparition, liste des fermants |
| `halo-infinite-map-images` (`himap`) | lecteur des volumes déclencheurs du `levl` ; lecteur des volumes sans rendu (`sbsp` physique + `scgt` Havok) ; calcul du retrait par mode |
| LevelUp | rapatriement des variantes natives ; `internal/mapfondsidecar` (retraits par famille, raisons) ; cuisson par `-plan-seul` (pas de recuisson des images) ; rejeu, Tactique et Occupation du terrain choisissent le plan selon le mode du match |

### 6.3 Lots, pensés pour l'exécution parallèle

| Lot | Contenu | Modèle | Dépend de |
|---|---|---|---|
| P0 | Relecture du plan | Fable | |
| A | `mapvar` : type d'objet, échelle, noms Forge, étiquettes complétées, fermants et règle d'apparition, témoins sur fixtures | Sonnet | |
| B | Rapatrier les `map.mvar` de toutes les variantes natives du catalogue (une quarantaine), inventaire des fermants étiquetés par carte | Haiku ou Sonnet | |
| C | `himap` : volumes déclencheurs du `levl` (mort, lieux), témoins `gamefiles` sur les 31 niveaux | Sonnet | |
| D | `himap` : volumes sans rendu ; lecture du blob Havok par ses pointeurs relatifs, cas à plusieurs feuilles, contrôle `m_guid` contre les instances de rendu de la variante `pc`, niveaux à géométrie partagée | **Opus** | |
| E | `himap` : retrait par mode (coupe du maillage, connexité, formes des fermants, raisons) | **Opus** | A |
| F | LevelUp : sidecar, cuisson `-plan-seul` des 114 fonds, garde par les positions, rapport par carte | Sonnet | B, C, D, E |
| G | Web : choix du plan selon le mode, rendu des zones fermées, textes fr et en, correspondance mode vers radical dans le TOML | Sonnet | F (contrat du sidecar figé tôt) |
| R | Revue adverse finale, une ronde | Fable | tout |

A, B, C, D partent ensemble ; E démarre dès que A a figé ses types ; G démarre sur un sidecar
d'exemple. Deux tags de briques (un par dépôt), puis une montée des `require`.

### 6.4 Coût

- Calcul : léger. Un maillage natif fait 650 à 800 faces ; l'inondation à 0,1 unité (120 000
  noeuds) prend moins d'une seconde par mode. Pas de recuisson d'image.
- Lecture du jeu : les natives demandent l'installation locale (comme aujourd'hui) ; les
  `scgt` pèsent 3 à 4 ko pièce.
- Réseau : une quarantaine de variantes natives de 50 à 160 ko, une fois.

### 6.5 Risques

| Risque | Parade |
|---|---|
| Forme d'un fermant fausse (ancrage, ordre des axes, porte posée couchée) | lire la boîte sur le modèle de collision de l'objet ; garde par les positions |
| Porte qui s'ouvre en jeu, barrière basse qu'on saute | garde par les positions ; ne fermer que ce qui change entre deux modes, signaler le reste |
| Volume sans rendu pris à tort pour un mur | couche commune publiée d'abord en calque de contrôle |
| Niveaux sans volume trouvé (Chasm, Forbidden, Illusion, Fragmentation, Behemoth) | vérification dans le lot D avant de conclure |
| Maillage principal de Streets : 2 ressources candidates non extraites (avertissement existant) | hors périmètre, à garder en vue : le plan de Streets repose sur ce maillage |
| Familles de mode mal reliées aux modes du match | table TOML avec témoin par mode ; mode inconnu = plan commun |

### 6.6 Ce que la méthode ne couvrira pas

- la validation d'un plan Infection ou Firefight par les positions (aucun film) ;
- les bloqueurs d'équipe et à sens unique (fermés pour un camp, ouverts pour l'autre) ;
- un volume de mort éteint par un mode ou un script (le film le dit, la grammaire le jette :
  à ouvrir avec une montée du décodeur, donc hors 8.3.1) ;
- le « sol jamais atteint » au sens de Den (toits, rebords) au-delà des volumes sans rendu ;
- les cartes Forge sans maillage publié (sous environ 1 000 objets) : pas de plan, donc pas
  de plan par mode ;
- les couches du `levl` : noms non résolus, rôle par mode non établi.

## 7. Découvertes hors périmètre (notées, non traitées)

1. **`himap.TypesVolumesDeMort` range deux types qui ne sont pas des volumes de mort.**
   `-588988541` est « Light probe marker » (`m_type` 29, un par carte) et `176825834` est
   « Nav Cutter » (`m_type` 28). Seuls `937132837` (« Kill Volume ») et `-1751270658` (« Soft
   Kill Volume ») sont de `m_type` 15. `BoiteDesVolumesDeMort` (réglage `rogneAuxVolumesDeMort`)
   borne donc aussi à la sonde de lumière et aux coupes de maillage.
2. **L'échelle des primitives Forge existe et n'est pas lue.** `root/3[]/8/23[0]/0` porte un
   vec3 sur 404 578 primitives sur 404 578 ; 383 091 (94,7 %) ne sont pas unitaires, 32 232
   seulement sont uniformes (« Wall Cube Standard » : 1,00 ; 7,50 ; 0,02). `InstanceForge` pose
   à l'échelle 1 (l'en-tête de `cuisson_forge.go` la disait absente, mesure faite sur les
   champs `[6]` et `[9]`). Sonde : `cmd/mvarscale`, `cmd/mvarprim`.
3. **Noms d'étiquettes que `mapvar` donnait pour non retrouvés** : `bastion_zone` = -767961569
   (la colline de KOTH), `bastion_plate` = -1482301937, `landgrab_plate` = -941529218,
   `strongholds_plate` = -1246645531, `firefight_plate` = 732028173 (vérifiés par
   `LabelHash`). Le filtre 2133978317 reste sans nom (`bastion_include` et `koth_include` ne
   correspondent pas).
4. **Les volumes de mort des cartes natives sont localisés** (tag `levl`, racine +0x3bc,
   octet +0x16 = 1) : cela lève le « non localisés » du 2026-08-10. Le type 6 du même bloc
   porte les lieux nommés.
5. **Les 20 variantes natives de `.ai/re_dump/mapvar/` sont des fichiers vides.**

## 8. Rejouer les mesures

Depuis `plan_par_mode\`, avec `D="C:/XboxGames/Halo Infinite/Content/deploy"` :

```bash
bin/mvarcensus.exe data/forge_names.json census <depot>/.ai/re_dump/mapvar      # §1.2
bin/mvarcensus.exe data/forge_names.json dump mvar_map/<asset>__map.mvar        # §1.3
bin/levlprobe.exe  "$D/pc/levels/multi/sgh_interlock/sgh_interlock-rtx-new.module" kill   # §2
bin/levlcouches.exe "$D/ds/levels/multi/sgh_streets/sgh_streets-rtx-new.module"            # §2
bin/physprobe.exe  "$D/ds/levels/multi/sgh_streets/sgh_streets-rtx-new.module" liste \
                   "$D/any/levels/multi/sgh_streets/sgh_streets-rtx-new.module"            # §3.1
bin/scgtdump.exe   "$D/any/levels/multi/sgh_streets/sgh_streets-rtx-new.module" 236c91a8   # §3.2
bin/murs.exe       "$D/ds/globals/common-rtx-new.module" "$D/any/globals/common-rtx-new.module" out/murs_livefire.json
bin/matchinfo.exe  <depot>/data/cache/film_chunks > out/matchinfo.tsv                      # §4
node positions.cjs out/matchinfo.tsv <depot>/data/cache/replays/halo_infinite out/positions.json <assets...>
bin/planmode.exe   "$D" sgh_streets - mvar_map/9c7b0b0f-...__map.mvar data/forge_names.json \
                   out/murs_streets.json "$D/ds/levels/multi/sgh_streets/sgh_streets-rtx-new.module" \
                   out/positions.json 9c7b0b0f-...,e23ea388-... slayer infection            # §5
bin/forgeplan.exe  data/forge_names.json <variante.mvar> <navmesh.blob>                    # §1.2
```

Sorties conservées : `out/planmode_streets.txt`, `out/planmode_livefire.txt`,
`out/murs_<niveau>.json` (31 niveaux), `out/matchinfo.tsv`, `out/positions.json`.

## 9. Sources

- Cartographer et libfilmshell (Den, MIT), clones locaux en lecture : `limits.rs` (règle des
  étiquettes, forme des fichiers d'accès), `navmesh.rs` (layout des volumes déclencheurs et des
  couches du `levl`), `floor.rs` (`rings_within`, `playable`), `map_variant.rs` (champs Bond,
  types d'objet), `forge_data.rs` (noms des objets Forge), `dataview.js` (règles de retrait).
  Ses outils de calcul (`film-assets`) sont privés : rien n'en est repris.
- Acquis du dépôt : `INVESTIGATION_PHYSIQUE_ZONE_JOUABLE_2026-08-09.md` (bloc physique, pas 192),
  `INVESTIGATION_VOLUMES_MORT_MVAR_2026-08-10.md`, `INVESTIGATION_SDDT_PFND_2026-08-10.md`
  (sous `.ai/archive/V7.5/cartes/`).
