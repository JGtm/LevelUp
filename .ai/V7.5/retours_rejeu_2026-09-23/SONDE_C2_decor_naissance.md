# Sonde C2 : un véhicule de décor se reconnaît-il dès sa naissance ? (2026-09-23)

Question de l'utilisateur : « malheureusement ce cas de figure se fait au cas par cas non ? On a pas
moyen de savoir s'ils sont jouables ou non avant ? »

Convention : **MESURÉ** (instrument étalonné, chiffres), **DÉDUIT** (conséquence de mesures ou du
code du jeu), **HYPOTHÈSE** (non vérifié).

## 0. Verdict

1. **Aucun champ lu ne dit « non jouable » ou « décor ».** Les véhicules de décor de Starboard et de
   Goliath sont de vrais véhicules physiques. Au chargement de la carte, ils se posent : 7 à 117 records
   delta `i0`/`i1`/`i2`/`i3` en 3 à 4 s, bien avant l'origine du match. Ensuite ils dorment. Rien ne les
   sépare d'un véhicule garé dans `i16 object-physics-flags` (5 bits, 0 des deux côtés), ni dans `R(2)`
   du bloc MPP (0 partout), ni dans le masque delta (pas de `i34`). **MESURÉ.**
2. **La naissance porte un champ de grammaire qui dit « posé par la carte ».** C'est le 6e champ du bloc
   `object-multiplayer-properties` (`FUN_14080cfe8`) : `FUN_14080d524`, porte `R(1)`, puis si elle est
   à 1, `R(13)` → `MPP+4` (ushort), défaut `0xFFFF`. **MESURÉ** sur le parc non-BTB (22 films,
   259 vies `ti=40` recensées) :
   - **13/13** vies de famille pilotable qui portent ce champ sont les 13 vies que la règle L1.3 masque ;
   - **0/100** vie de véhicule en jeu le porte (100/100 à porte fermée).
3. **Mais le champ veut dire « posé par la carte », pas « décor ».** Il est aussi porté par des
   éléments de carte actifs : les 9 tourelles automatiques de Snowbound (2 films, 60 à 156 records
   delta après l'origine) et le châssis inconnu `3a8060e2` de Takamanohara, auquel la publication
   attribue un occupant. **MESURÉ.** Un véhicule posé à la main dans l'aire de jeu par un auteur de Forge
   porterait très probablement le même champ, et serait pilotable. **HYPOTHÈSE** : aucun cas dans le
   parc non-BTB, et les films BTB n'ont pas été ouverts.
4. **La règle L1.3 n'est pas du cas par cas.** Elle est générale et se lit dans le film. Mais sa
   justification écrite, « le film ne réplique la position qu'une fois », est **inexacte** : le film
   réplique la pose, mais **avant l'origine**, et la publication ne garde qu'un point, à `t0`. La
   règle tient parce que les véhicules posés par la carte existent dès le chargement et dorment avant
   l'origine. Les véhicules des générateurs, eux, naissent au début de manche sur les cartes Forge et se
   posent après l'origine. **DÉDUIT** (§2.3).

Réponse courte : **oui, la naissance dit quelque chose**, et c'est « posé par la carte (index de
placement) ». Sur le parc, pour les véhicules pilotables, cela coïncide exactement avec le décor.
Le « non jouable » n'est écrit nulle part : c'est la géographie (hors de l'arène) et l'absence
d'usage qui le disent.

## 1. Instrument

`apps/go-api/internal/games/halo_infinite/film/internal/grammar/c2_decor_ti40_research_test.go`
(`//go:build research`, `TestC2DecorTi40`, variables `C2_FILM`, `C2_CARTE`, `C2_OUT`). Pour chaque record
`ti=40` d'image-clé, lecture de production `WalkKeyframeFullState` :
- les trois champs d'en-tête après `[eid][archétype]` ;
- l'état par défaut de `FUN_1410a5a74` relu segment par segment **avec les fonctions de production** :
  version, les onze sous-champs du bloc MPP, porte `bVar14` + feuille 4, `R(19)`, porte `cVar3` + liste ;
- les bits bruts de chaque composant porté ;
- le désync et l'emprise.

Pour chaque record `ti=40` delta, lecture par la marche de production (`marchRecordsOf`) : type,
masque, désync.

**Étalonnage** : le curseur après `n2` tombe exactement sur le premier composant de la marche de
production. C'est vrai sur **3 145 / 3 145** records d'image-clé des 22 films (colonne « incohérents
0 » à chaque invocation). Jointure avec le document publié : `instruments/c2_decor/c2_parc.mjs`,
résumé par vie : `c2_resume.mjs`. Table de parc : `c2_parc_vies_ti40.txt` (259 vies).

Films, un à la fois, sous la voie film :
- les trois demandés : f0220a96 (Starboard), 0301037e (Refuge, témoin négatif), d8b13ec2 (Goliath) ;
- ab526724 (Starboard, second témoin) ;
- les 18 autres films non-BTB du parc qui ont des véhicules (effectif ≤ 11) : 0a44c6cc, 1cd3848a,
  2cf24f30, 396cfc92, 4ecdf3e7, 7b0d89c4, 7fce3219, 81c02726, 8a485699, a36c8bed, a4083bd2,
  ac03413d, bfecd02b, daaa17d6, e1259a69, f2966f08, f9e99ca4, fccc61cd ;
- films à effectif ≥ 17 écartés (BTB probables) : 4f77afc1, 5676a9ba, 879a4dba, a464e20b, c259789d.

Réserve : f9e99ca4 a été lu sous l'entrée de catalogue `starboard`, faute de nom de carte. Les
champs `mpp.*` précèdent toute largeur dépendante de la carte ; les composants de ce film ne sont pas
interprétés.

## 2. Mesures

### 2.1 Décor contre garé-simulé (f0220a96 contre 0301037e), MESURÉ

| champ (état par défaut de naissance) | décor Starboard (771-778) | véhicules de Refuge (771-788) |
|---|---|---|
| `mpp.d524` (`R(1)` [+`R(13)`]) | **porte 1** : 0x0a00, 0x0a02, 0x0a03, 0x0a8f, 0x0a90, 0x0a91 | **porte 0** (20/20 vies 769-788) |
| `ds.r19` (`FUN_14076dc04`, direction unitaire) | propre à chaque objet (0x1d0c9, 0x4aa89, …) | **0x0a83e** partout |
| présence dans l'image-clé 1 (chargement) | oui (5727,28 s ; origine à +32,1 s) | non : ils apparaissent entre 6928 et 6948 s, origine à 6936,7 s |
| records delta | 7 à 106, tous entre +5,1 et +8,9 s (avant l'origine) | burst de pose à l'origine (+28 à +31 s), puis réveils (+410 s, +441 s) |
| `i16 object-physics-flags` | 0 | 0 (774) |
| `mpp.r2`, `mpp.idx`, `mpp.liste`, `mpp.d4d0`, `mpp.queue`, `mpp.r18` | 0 | 0 |
| `hdr.r4` / `hdr.r8` / `n1` / `n2` | 3 / 0 / 0xb0 / 0x8d8 | identiques |
| `mpp.variant` | 42c9679f / 4e154ee8 / absent (Wasp) | 42c9679f / absent (Ghost) |
| `ds.bv14` | 0 | 0 |

ab526724 rend **à l'identique** les mêmes index `d524` et les mêmes `r19`, slot par slot. Goliath
(d8b13ec2, Wasp 768) : `d524` = 0x223, `r19` = 0xaa8a, présent dès l'image-clé 1, 20 records delta,
tous avant l'origine.

### 2.2 Parc non-BTB (22 films, 259 vies recensées), MESURÉ

| | porte `d524` = 1 | porte `d524` = 0 |
|---|---|---|
| famille pilotable | **13** (toutes « décor » L1.3 : Starboard ×12, Goliath ×1) | **100** (aucune « décor » L1.3) |
| autre / non publiée | 33 (tourelles automatiques, `b857fb95`, `3a8060e2`, Banshee `c6e79dcc` de Starboard vue à une seule image-clé) | 113 |

- La porte `d524` = 1 n'apparaît **que sur des canevas Forge** (`fo03`, `fo08`, `fo09`, `fo11`, `fo13`).
  Elle n'apparaît jamais sur `va_behemoth`, `va_launchsite`, `chasm` ni `ctf_illusion`.
- `r19` vaut **0x0a83e sur 100/100** véhicules des générateurs. Les objets posés ont une valeur propre.
- **Index stables d'un match à l'autre** (Starboard ×2), consécutifs pour des objets groupés, et
  **conservés à la réapparition** : sur 7fce3219, `3a8060e2` recréé aux slots 770 et 771 reprend
  l'index 0x066 du slot 768. **DÉDUIT** : c'est l'index de l'objet dans la liste de placement de la
  carte (variante de carte Forge).
- La présence avant l'origine **ne discrimine pas** : 39 véhicules des générateurs sont présents
  avant l'origine sur les cartes non Forge (Super Fiesta, Behemoth).

### 2.3 Pourquoi la règle L1.3 tient, DÉDUIT

La publication ne garde qu'un point, à `t = t0 = 0`, pour une pose qui a lieu 23 à 27 s avant
l'origine. Pour un véhicule de générateur sur carte Forge, la pose a lieu à l'origine ou après :
Refuge 774, points à t = 0, 1, 2, 3.

Deux cas cassent L1.3 et pas `d524` :
- un décor dérangé après l'origine ;
- un véhicule de générateur posé avant l'origine puis jamais touché (cas non Forge : présents avant
  l'origine).

Aucun des deux n'est observé sur le parc. Les 14 records « après l'origine » de ab526724 776 sont des
lectures de marche à masque vide ou incohérent (0,4,9), donc du bruit de marche, pas un mouvement.

## 3. Ghidra (HaloInfinite.exe, lecture seule, sous la voie ghidra)

- `FUN_14080cfe8` (bloc MPP) : `FUN_14080d524(param_1 + 1, …)`, soit l'octet **+4** de la structure
  MPP. Juste avant : `R(18)` en porte → +0, défaut −1. Juste après : `R(2)` → +8, `R(5)` − 1 → +0x1a,
  compte `R(3)` → +0x2c.
- `FUN_14080d524` : `FUN_1406cf008` (porte `R(1)`). Si 0 : `*(u32*)param_1 = DAT_14473c2e8`. Si 1 :
  lecture de 13 bits (`+0x2c += 0xd`) dans un `ushort` → `*param_1`.
- `DAT_14473c2e8` = `FF FF 00 00`, soit l'index « aucun » (0xFFFF). C'est la constante « aucun » des
  index 16 bits, utilisée par une trentaine de fonctions (`FUN_142ca98f0`, `FUN_142ca9a34`, …).
- **Le nom du champ n'est pas trouvé.** Le consommateur de `MPP+4` n'est pas remonté depuis
  `FUN_14080cfe8` ni depuis la vtable du composant (`FUN_1407f2224`). Aucune chaîne ne nomme le champ.
  « Index de placement de la variante de carte » reste **DÉDUIT** du §2.2 : Forge seul, stable entre
  matchs, conservé à la réapparition, 13 bits.
- Au passage, énumérations Forge relevées dans `FUN_14025fda0` :
  - `forge_object_physics_mode` : Dynamic, FixedNoGravity, Phased, Fixed, ProjectileOnly,
    DynamicNoGravity ;
  - `map_variant_physics_mode` : Normal, Fixed, Phased, ProjectileOnly, OutOfBounds, MPObjective.

  Le mode physique d'un objet Forge **n'est pas trouvé** dans ce qui est lu du record : `mpp.r2` et
  `i16` valent 0 pour décor comme pour garé. Les décors se posent par la physique, donc
  « Dynamic » ou « Normal » (**DÉDUIT**).

## 4. Ce que le lot doit lire (impactPlan)

- **Rien à changer maintenant** : L1.3 reste la règle, générale et valable pour les films futurs. Il
  faut seulement corriger son commentaire : le film réplique la pose avant l'origine, et la
  publication la ramène à un point.
- **Si l'utilisateur veut une lecture de grammaire (vague D, re-décodage)** :
  - publier par vie `placement` (porte + index `d524`) ;
  - cela suppose un 5e champ publié du bloc MPP (`MPPField`, `types.MPPFieldCount`) et une révision de
    grammaire / faits, donc une montée de schéma et une republication avec re-décodage ;
  - le prédicat web deviendrait : « posé par la carte ET aucun mouvement répliqué après l'origine ET
    aucun occupant » → décor. Jamais « posé » seul, puisque les tourelles et `3a8060e2` sont posés et
    actifs ;
  - repli nommé et compté si la porte est illisible.
- Garde-rail possible : un test Go sur fixture datée. Pour les familles pilotables du parc, on doit
  retrouver `placement` ⇔ décor L1.3 (13/13, 0/100). Au premier contre-exemple, le compteur le montre.

## 5. Découvertes hors périmètre (notées, non traitées)

1. `i34 vehicle-type-physics` (composant « conditionnel non expliqué », CADRAGE §6.5) vit sur les
   véhicules **volants**. Falcon : 5 105/6 508 et 2 102/3 418 records (Refuge), 907/1 794
   (fccc61cd). Banshee : 77/676. Mongoose : 24/2 524. **DÉDUIT** : physique de vol.
2. La Banshee posée de Starboard (`c6e79dcc`, `d524` 0x1d8, slot 768/1) n'existe qu'à l'image-clé 1,
   puis disparaît avant l'origine. Elle n'est pas publiée.
3. La marche delta rend, sur plusieurs films, des records `ti=40` à des slots hors bande (58, 59, 187,
   1282, 6258, …). Ce sont des lectures fortuites de marche, à ne pas interpréter.
