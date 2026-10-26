# Trouver l'ecrivain d'un composant ECS : la chaine complete, verifiee (2026-09-16)

> Preparation du lot 3.6 (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, « Preparation facultative,
> sans production »). Aucune ligne de production touchee, aucun film decode. Source :
> `HaloInfinite.exe` par Ghidra en LECTURE SEULE (HTTP direct `127.0.0.1:8089`).
>
> Instrument / Ghidra lecture seule.

---

## 1. La chaine, en quatre pas

Le plan dit « descripteur trouve par dump de `.rdata`, ecrivain `vtable+0x18` ». Le dump n'est
pas necessaire et l'offset n'est pas `+0x18` pour la famille des composants « managed-* » et
« device-* » / « vehicle-* ». La chaine EXACTE, verifiee sur quatre composants deja portes dont
le depot connait deja l'adresse (§3), est :

1. **La chaine de caracteres** : `search_strings` sur le nom exact du composant. Elle vit en
   `.rdata` (`143606000..1443951ff`). Exemple : `managed-player-forge-weather-effect-overrides-component`
   -> `0x143c95460`.
2. **Le petit accesseur de nom** (« thunk ») : la seule reference a cette chaine est une
   fonction de 8 octets, `LEA RAX, [rip+disp]` puis `RET`, rangee par paquets de 16 octets dans
   une zone de `.text`. Exemple : `0x141177e70`.
3. **Le descripteur du composant** : la seule reference a ce thunk est un SLOT dans une table de
   pointeurs de fonctions de `.rdata`. Ce slot est a **`descripteur + 0x18`**. Exemple : slot
   `0x143d08858` -> descripteur `0x143d08840`.
4. **L'ecrivain** : il est a **`descripteur + 0x40`**, c'est-a-dire `slot du nom + 0x28`.
   Exemple : `0x143d08880` contient `0x142ed5bc8` = `FUN_142ed5bc8`.

Le « `vtable+0x18` » de la memoire du depot designe donc le slot du NOM, pas l'ecrivain. Les
deux chiffres se deduisent l'un de l'autre et il faut retenir : **nom a `+0x18`, ecrivain a
`+0x40`**.

## 2. La forme du descripteur (famille observee : 10 slots, 0x50 octets)

| Offset | Contenu | Constant d'un composant a l'autre ? |
|---|---|---|
| `+0x00` | `0x141191ab0` | oui |
| `+0x08` | `0x14076ced0` | oui |
| `+0x10` | `0x14117b4a0` | oui |
| `+0x18` | **accesseur de nom** | non — c'est l'identite du composant |
| `+0x20` | `0x14049b600` | oui |
| `+0x28` | fonction compagnon (construction / copie) | non |
| `+0x30` | `0x141c8f880` | oui |
| `+0x38` | `0x14076ce9c` | oui |
| `+0x40` | **ecrivain / lecteur de bits du composant** | non — c'est la GRAMMAIRE |
| `+0x48` | `0x14049b600` | oui |

Les quatre pointeurs constants (`141191ab0`, `14076ced0`, `14117b4a0`, `14049b600`,
`141c8f880`, `14076ce9c`) sont la SIGNATURE de la famille : si on ne les retrouve pas au bon
offset, on n'est pas sur un descripteur de cette forme et l'offset `+0x40` ne vaut pas.
Certaines familles ont un descripteur plus court et decale (releve sur `device-*` : la
signature commence a `14117b4a0`) — dans ce cas, **calibrer sur un composant DEJA PORTE du
meme archetype** avant de lire les autres. C'est ce qui a ete fait pour chacune des quatre
familles de cette vague.

## 3. Les quatre calibrations (preuve que la regle tient)

| Composant deja porte | Descripteur | `+0x40` lu | `deser_addr` de `ecs_table.tsv` | Concorde |
|---|---|---|---|---|
| `ti=9 i3 managed-player-back-button-scoreboard-flair-component` | `0x143d08890` | `0x142ed5af4` | `FUN_142ed5af4` | oui |
| `ti=12 i0 managed-navpoint-sub-type-component` | `0x143d07e98` | `0x1410e0cac` | `FUN_1410e0cac` | oui |
| `ti=12 i14 managed-navpoint-radial-progress` | `0x143d08110` | `0x140fc8d14` | `FUN_140fc8d14` | oui |
| `ti=43 i19 device-position-animation-name-component` | `0x143d0ce98` | `0x1410156e4` | `FUN_1410156e4` (plan, D5 du 1.9.1 bis) | oui |

Quatre concordances sur quatre, sur trois familles differentes et avec une adresse venue du
plan et non de la table. La regle est utilisable telle quelle.

## 4. Lire la GRAMMAIRE sans lire tout le decompile

Chaque ecrivain manipule le meme objet de flux de bits (`param_2`) dont :

- `+0x2c` est le **compteur de bits** : chaque `ADD dword ptr [<reg> + 0x2c], N` du
  desassemblage est un champ de **N bits**. C'est la lecture la plus rapide d'une grammaire :
  la liste ordonnee de ces constantes EST la suite des largeurs.
- `+0x28` est le compteur d'octets rincés, `+0x30` l'accumulateur, `+0x38` le nombre de bits
  en reserve, `+0x40` le curseur d'octets. Un meme champ apparait DEUX fois dans le
  desassemblage (chemin rapide « la reserve suffit » et chemin lent « il faut recharger ») :
  compter les champs, pas les sites.
- Les appels recurrents a connaitre : `FUN_1406cf008` = `R(1)` ; `FUN_1406d49c4` = `R(1)`
  rendu ; `FUN_1407f2058` = porte INVERSEE `R(1)` puis `R(5)` (index d'entite, cf. sondage E2) ;
  `FUN_1406d310c(n)` = `ceil(log2(n))` ; `FUN_1406d3140` / `FUN_1406d5110` = lecteur / ecrivain
  d'une reference d'entite (`base + R(largeur)` puis `R(2)` de generation, cf.
  `NOTE_BANDE_SLOTS_BIPEDE_2026-09-16.md`) ; `FUN_1406d6e28(flux, valeur, n)` = ecriture de `n`
  bits par le chemin lent ; `FUN_142ee2194` = `FUN_1406d84b4(..., 0x10, ...)` soit **`R(16)`
  dequantifie dans `[-100, +100]`** (bornes `DAT_143cd8f84` = `0xc2c80000` = -100,0f et
  `DAT_143cd84a8` = `0x42c80000` = +100,0f) — c'est le champ de composante de vecteur du
  moteur, et il vaut pour tous les blocs de vecteur rencontres dans cette vague.

## 5. Ce que cette vague a produit

| Archetype | Note | Ecrivains nommes par cette vague |
|---|---|---|
| ti=9 (joueur) | `NOTE_3_6_TI9_2026-09-16.md` | 1 sur 1 (le bloquant, grammaire COMPLETE) |
| ti=11 (objectif) | `NOTE_3_6_TI11_2026-09-16.md` | 1 sur 1 (deja connu ; la table d'etiquettes est relevee ici) |
| ti=12 (navpoint) | `NOTE_3_6_TI12_2026-09-16.md` | 18 sur 26 (le bloquant, grammaire COMPLETE) |
| ti=35 (bipede) | `NOTE_3_6_TI35_2026-09-16.md` | le predicat de queue du bloquant, RESOLU |
| ti=40 (vehicule) | `NOTE_3_6_TI40_2026-09-16.md` | 16 sur 16 |
| ti=42 (arme au sol) | `NOTE_3_6_TI42_2026-09-16.md` | 0 — negatif MESURE : rien a porter |
| ti=43 (device) | `NOTE_3_6_TI43_2026-09-16.md` | 22 sur 22 |

Total : **57 ecrivains nommes par cette vague** (1 + 18 + 16 + 22 ; celui de `ti=11` etait deja
connu du depot), dont 2 avec leur grammaire complete, 1 avec sa table d'etiquettes et 1 avec
son predicat de queue. Aucune de ces adresses n'etait dans `ecs_table.tsv` (colonne `deser_addr` vide) avant
cette vague ; elles y entreront au lot 3.6, dans le commit qui porte le composant — jamais
avant (garde-rails `ecs_table_guard_test.go`).

## 6. Ce que cette methode NE donne pas

- Elle donne l'ECRIVAIN, pas le LECTEUR. Les deux sont symetriques dans ce moteur (meme
  sequence de champs, meme largeurs) et tout le decodeur du depot est deja construit sur cette
  symetrie, mais un ecrivain qui branche sur un ETAT RUNTIME (et non sur un bit du flux) rend
  une grammaire non decidable hors ligne — c'est le cas de `ti=35 i57` et `i63`, deja consignes
  `partiel` dans la table.
- Elle ne dit rien d'un appel VIRTUEL (`(**(code **)(*obj + 0x08))(obj)`) : la largeur depend
  alors du type dynamique de l'objet, et il faut enumerer les vtables candidates. C'est le
  reste a faire de `ti=11 i4`.
- Elle suppose que la chaine du nom est unique dans le binaire. Quand un composant est instancie
  N fois (`ti=11` 16 x `sub-objective-entities`, `ti=12` 8 x `visual-state-groups`), les noms
  sont suffixes `-0`..`-N` et rassembles dans une TABLE DE POINTEURS, pas par des accesseurs :
  la chaine des quatre pas s'y arrete au pas 3 (cas non resolu de `ti=12`, §4 de sa note).
