# V3D — Les sons de DESTRUCTION des vehicules : trouves, prouves, reconstruits

> Worktree `LevelUp-wt-vehicules`, branche `wt/vehicules-tourelles`. Lecture seule sur les
> donnees du jeu. AUCUN commit, AUCUN `git add`. Code ajoute : uniquement sous
> `apps/go-api/cmd/weapon-sounds/` (3 fichiers, additifs, `gofmt`/`go vet`/`go build` verts).
> WAV livres = transitoires non commites, comme le reste de `sons_v3_reconstruits/`.
>
> **Mandat** : trouver et reconstruire l'explosion de destruction de chaque vehicule jouable,
> par la methode qui a marche pour les tirs et les moteurs (banque -> event -> couches -> wems,
> preuve a 3 axes).

## 1. Verdict en une page

**Trouve, pour les 8 vehicules du corpus, sans zone d'ombre.**

L'explosion de destruction **n'est pas dans la banque du vehicule**. Elle vit dans **six
banques dediees**, indexees par TAILLE et par FACTION, qui existent en clair sur le disque du
jeu sous `Sound/win/SFX` :

```
sb_008_exp_vehicle_small_unsc      sb_008_exp_vehicle_small_covenant
sb_008_exp_vehicle_med_unsc        sb_008_exp_vehicle_med_covenant
sb_008_exp_vehicle_large_unsc      sb_008_exp_vehicle_large_covenant
```

Chaque vehicule en designe **une seule**, par la chaine

```
vehi  ->  hlmt  ->  foot (table de MATERIAU)  ->  snd!  ->  sbnk exp_vehicle_<taille>_<faction>
                                                              -> event a 4 COUCHES SIMULTANEES
```

L'evenement de destruction est un **one-shot** : aucun switch de regime, aucun RTPC, aucun
delai de couche. Il est fait de **3 couches simultanees de 6 variantes chacune** plus **une
4e couche a un seul son, partagee entre banques** — le « lit » commun d'explosion que le
mandat demandait de chercher. Il existe, et il est mesure (§5).

## 2. Le tableau par vehicule (donnees brutes)

Mesures sur le fichier livre `explosion.wav` (somme des 4 couches, variante 1, normalise a
-1 dBFS). `bas` = RMS sous 200 Hz, `haut` = RMS 3-8 kHz, `crest` = crete moins RMS.

| Vehicule | `vehi` | `hlmt` | `foot` | banque (sbnk) | event | wems | duree | RMS | crest | bas | haut | verdict |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Mongoose / Gungoose | `000025aa` `af31ab1a` `de26e3d7` | `10199eb5` | `5cc09369` | `969f4dc6` small_unsc | `1c0dae6d` | 3x6 + 1 | 3,20 s | -17,6 | 17,2 | -18,6 | -35,1 | **TROUVE** |
| Warthog (famille) | `00002705` `cb96ca07` `fe32c0f4` | `daf7f543` | `ac015a46` | `c468fb55` med_unsc | `2152102c` | 3x6 + 1 | 4,84 s | -21,6 | 20,6 | -22,9 | -37,5 | **TROUVE** |
| Wasp (AV-49) | `b65b3b4a` | `6b06a15e` | `ac015a46` | `c468fb55` med_unsc | `2152102c` | 3x6 + 1 | 4,84 s | -21,6 | 20,6 | -22,9 | -37,5 | **TROUVE** (identique au Warthog) |
| Scorpion (M808) | `f6f54e56` = chassis `0000d3db` | `e7fe7564` | `7b3a23f8` | `94d43e95` large_unsc | `dcc7bca1` | 3x6 + 1 | 5,19 s | -18,5 | 17,8 | -19,4 | -35,1 | **TROUVE** |
| Ghost | `0000d3dc` `5b80c406` `9af9e693` | `3b3038e6` | `0dfde60d` | `b1f8608b` small_cv | `8b33a81a` | 3x6 + 1 | 4,38 s | -19,6 | 18,6 | -20,9 | -34,9 | **TROUVE** |
| Chopper (Bannis) | `002ba902` `3d4a8a5a` | `fe29d23d` | `6864f41f` | `3fdc61a7` med_cv | `ed7cfa4b` | 3x6 + 1 | 4,03 s | -19,6 | 18,6 | -20,7 | -35,8 | **TROUVE** |
| Wraith | `00002706` `ae845375` | `5b5c960d` | `48669cd9` | `2eaae6d7` large_cv | `1bf6fdde` | 3x6 + 1 | 5,36 s | -18,3 | 17,3 | -19,5 | -32,6 | **TROUVE** |
| Banshee | `000026ed` `0001530a` `c6e79dcc` | `df38bc96` | `48669cd9` | `2eaae6d7` large_cv | `1bf6fdde` | 3x6 + 1 | 5,36 s | -18,3 | 17,3 | -19,5 | -32,6 | **TROUVE** (identique au Wraith) |
| Falcon (tourelle LMG) et autres enfants | `0000d4ff` `0000d500` `64b925eb` `bcfb852f` `dd7f9102` | — | — | — | — | — | — | — | — | — | — | **PAS DE SON PROPRE** (§8) |

Deux jeux de fichiers sont **identiques par construction** : Warthog et Wasp partagent
l'evenement `2152102c` (leur `hlmt` differe mais la chaine converge sur le meme `foot`), et
Wraith et Banshee partagent `1bf6fdde` (le Phantom `000026f2` aussi). Ce n'est pas une copie
paresseuse : c'est ce que le jeu joue.

L'identite des `vehi` n'est pas devinee, elle vient des releves nommes anterieurs de ce
chantier : `ASSEMBLAGE_ENFANTS_2026-09-01.md` (Scorpion : `vehi 0x0000d3db -> hlmt 0xe7fe7564`),
`ASSEMBLAGE_WRAITH_PHANTOM_2026-09-01.md` (Ghost `0x0000d3dc`, Banshee `0x000026ed`, Chopper
`0x002ba902`, Wraith `0x00002706`), `REWORK_WARTHOG_GUNGOOSE_2026-09-01.md` (Mongoose
`0x000025aa`/`0xaf31ab1a`/`0xde26e3d7`), `V4_RAPPORT_SPRITES_2026-08-31.md` (Warthog
`0x00002705`, Wasp `0xb65b3b4a`).

## 3. La preuve, trois axes

### Axe 1 — STRUCTURE : la remontee banque -> vehicule

Mode nouveau `remonter-banque` : il part des banques d'explosion, prend les `snd!` qui en
dependent, puis remonte niveau par niveau les tags qui les citent, jusqu'a tomber sur des
`vehi`. Resultat UNSC (module `any/globals`), 4 niveaux, sans ambiguite :

```
niveau 0 : 9 snd!  (3 par banque)
niveau 1 : 3 effe + 3 foot + 3 stai
niveau 2 : 4 hlmt  (10199eb5, daf7f543, 6b06a15e, e7fe7564)
niveau 3 : 8 vehi
```

Quatre `hlmt` seulement, un par famille de chassis. L'ancre nommee qui ferme la lecture :
`hlmt e7fe7564` est deja documente comme le `hlmt` du **chassis Scorpion**
(`ASSEMBLAGE_ENFANTS_2026-09-01.md` §« Chassis seul »), et c'est lui qui mene a
`exp_vehicle_LARGE_unsc`. Un char lourd sur la banque « large » : la chaine se lit toute
seule. Meme balayage sur `any/globals/common` pour le covenant : 3 banques, 8 `hlmt`,
17 `vehi`, dont Ghost sur « small », Chopper sur « med », Wraith et Banshee sur « large ».

**Piege corrige en cours de route** : la premiere version du balayage remontait 46 618 tags
au niveau 2. Cause : `00000000` et `ffffffff` (les « pas de reference » des tables de tags)
etaient restes dans l'ensemble cible. Ils en sont exclus ; le niveau 2 tombe a 4 tags.

### Axe 2 — DESIGNATION : quel evenement, exactement

Chaque banque porte 3 a 8 evenements. Lequel est la destruction ? Le mode `sndscan` cherche
les identifiants d'evenement **en clair dans le corps des `snd!`**. Chaque `snd!` en porte
**exactement un**, aucune ambiguite :

| snd! | event | banque | atteint par |
|---|---|---|---|
| `92da46d8` | `1c0dae6d` (4 couches) | 969f4dc6 | `foot 5cc09369` + `effe 7f1074e1` — Mongoose |
| `5600ccdd` | `2152102c` (4 couches) | c468fb55 | `foot ac015a46` + `effe dde4293b` — Warthog, Wasp |
| `2b49d0d7` | `dcc7bca1` (4 couches) | 94d43e95 | `foot 7b3a23f8` + `effe 84c492ff` — Scorpion |
| `9a37c894` | `8b33a81a` (4 couches) | b1f8608b | `foot 0dfde60d` — Ghost |
| `cdf0b06a` | `ed7cfa4b` (4 couches) | 3fdc61a7 | `foot 6864f41f` — Chopper |
| `9d3c427a` | `1bf6fdde` (4 couches) | 2eaae6d7 | `foot 48669cd9` — Wraith, Banshee, Phantom |
| `1625b9e5` / `9d70bb7b` | `16c57452` / `daaf2744` (1 couche) | 969f4dc6 | relais `stai 722a9fe4` |
| `9959e3a7` / `aa74c714` | `50f7b5c5` / `29082103` (1 couche) | c468fb55 | relais `stai 33f6727f` |
| `c4e6a34b` / `5f618fb9` | `7b1d99fe` / `651b5570` (1 couche) | 94d43e95 | relais `stai 52b9f7e9` |

**L'evenement pose par le vehicule est TOUJOURS celui a 4 couches simultanees.** Les
evenements a une couche sont apparies deux a deux par un relais `stai` (§6).

### Axe 3 — SPECTRE : c'est bien une explosion

Signature attendue : transitoire large bande fort (crest eleve), corps grave, queue de
debris, 1 a 4 s. Mesure sur les 6 banques (186 medias decodes, `vgmstream` + `ffmpeg astats`) :

- **duree** 1,85 a 6,23 s pour les couches proches ; les fichiers livres font 3,20 a 5,36 s ;
- **crest** 17,2 a 20,6 dB sur les melanges livres (12 a 28 dB sur les medias bruts) — un
  moteur en boucle, par comparaison, tourne a 4 dB (V3C) ;
- **corps grave dominant** : le RMS sous 200 Hz est a 1 dB du RMS global sur tous les
  vehicules (ex. Scorpion -19,4 contre -18,5) ;
- **haut 3-8 kHz** a -32 a -38 dB : present (les debris metalliques) mais nettement sous le
  corps ;
- **48 kHz**, mono et stereo melanges selon les couches.

Aucun des trois axes ne contredit les deux autres.

## 4. La structure d'un evenement de destruction

Toutes les banques ont la meme forme. Exemple, `sb_008_exp_vehicle_large_unsc` (Scorpion),
evenement `dcc7bca1`, tel que `eqip-arbre` le rend :

```
event dcc7bca1   4 couches SIMULTANEES [1 parmi 6 + 1 parmi 6 + 1 parmi 6 + 1 son]
  couche 20d47cce RandomSequence gain +17 dB  wems=[13353040 133393323 218433237 888260502 959267019 979095968]
  couche 36c9b286 RandomSequence gain +20 dB  wems=[176464618 304353642 387859342 871694806 907510484 943937175]
  couche 3c95cab9 RandomSequence gain +10 dB  wems=[76801691 352298157 532744785 623960989 628566016 778884090]
  couche 35278286 RandomSequence gain  +2 dB  wems=[686882988]                (lit sub PARTAGE)
```

31 medias par banque = 5 groupes de 6 + 1 lit partage. Recapitulatif des gains de chemin :

| banque | L1 | L2 | L3 | lit sub | ecart max entre couches |
|---|---|---|---|---|---|
| 969f4dc6 small_unsc | +7 | +10 | +15 | `224468375` -8 | 23 dB |
| c468fb55 med_unsc | +10 | +15 | +13 | `686882988` 0 | 15 dB |
| 94d43e95 large_unsc | +17 | +20 | +10 | `686882988` +2 | 18 dB |
| b1f8608b small_cv | +17 | +14 | +12 | `224468375` -8 | 25 dB |
| 3fdc61a7 med_cv | +12 | +9 | +8 | `686882988` 0 | 12 dB |
| 2eaae6d7 large_cv | +12 | +8 | +5 | `686882988` +2 | 10 dB |

## 5. Le lit generique partage — il existe

Le mandat demandait de chercher « un lit commun + une couche d'identite par vehicule, comme
les moteurs ». **C'est exactement l'architecture**, et le lit est identifie :

| media | duree | canaux | RMS | 3-8 kHz | banques qui le portent |
|---|---|---|---|---|---|
| `224468375` | 0,30 s | mono | -15,6 dB | **-73,6 dB** | small_unsc ET small_covenant |
| `686882988` | 0,92 s | mono | -18,7 dB | **-71,8 dB** | med + large, unsc ET covenant (4 banques) |

Un contenu 3-8 kHz a -72 dB, c'est un **sub pur** : le coup grave sous l'explosion, sans
aucun grain. Preuve de partage plus forte encore que l'identite du media : pour les deux
banques « small », c'est **le meme noeud de couche** (`16ad77c5`) qui le porte, dans deux
banques de factions differentes.

Le lit est deja inclus dans `explosion.wav` (c'est la 4e couche). Il est aussi livre isole
sous `lit_sub_partage.wav` dans chaque dossier, au meme gain, pour qu'il soit jugeable seul.

## 6. Les etats alternatifs (`stai`) — probablement les versions lointaines

A cote de l'evenement a 4 couches, chaque banque porte **deux evenements a une seule couche**,
apparies par un relais `stai` (etat). Mesures :

| banque | etat A | etat B | ecart avec l'explosion proche |
|---|---|---|---|
| 969f4dc6 | `16c57452` | `daaf2744` | -20,9 / -14,1 dB |
| c468fb55 | `50f7b5c5` | `29082103` | -24,9 / -15,1 dB |
| 94d43e95 | `7b1d99fe` | `651b5570` | -19,3 / -22,3 dB |
| b1f8608b | `ab214f3f` | `65fe08f9` | -27,5 / -15,5 dB |
| 3fdc61a7 | `7d52bdd4` | `3a818492` | -26,7 / -11,4 dB |
| 2eaae6d7 | `df41ac8b` | `f97fc935` | -15,1 / -10,7 dB |

11 a 27 dB plus bas, transitoire adouci, meme duree : **le profil d'une version lointaine**
(distance / LOD). **Reserve honnete** : l'etiquette « lointaine » est une lecture du spectre,
PAS un nom lu — les etats sont des hachages, et aucun balayage de noms courants ne les a
casses. Ils sont livres sous `explosion_lointaine_A.wav` / `_B.wav`.

## 7. Ce qui est livre

`.ai/V7.5/film_re/sons_v3_reconstruits/<Vehicule>/destruction/` pour **Gungoose**,
**Warthog_roquettes**, **Wasp**, **Scorpion**, **Ghost**, **Chopper** (nouveau), **Wraith**
(nouveau), **Banshee** (nouveau) — 9 fichiers chacun, 72 au total :

| fichier | contenu |
|---|---|
| `explosion.wav` | les 4 couches sommees, variante 1 de chaque couche |
| `explosion_v2.wav`, `explosion_v3.wav` | variantes 2 et 3 (le jeu tire une variante par couche a chaque lecture) |
| `couche_1.wav`, `couche_2.wav`, `couche_3.wav` | les 3 couches proches isolees, variante 1, au MEME gain que le melange |
| `lit_sub_partage.wav` | la 4e couche seule (le lit sub commun) |
| `explosion_lointaine_A.wav`, `_B.wav` | les deux etats du relais `stai` |

**Chaine de rendu** (identique en esprit a V3C, sans aucune correction destructive) :
`pck AKPK -> wem (mode pck-dump) -> vgmstream -> wav -> gains de chemin RELATIFS (couche la
plus forte a 0 dB, les autres a leur ecart declare) -> somme en FLOTTANT -> une seule
normalisation a -1 dBFS`. Pas d'egalisation, pas de compression, pas de limiteur. Le premier
essai sommait en entier 24 bits : le melange ecretait a 0 dBFS avant mesure — corrige en
`pcm_f32le` (les fichiers livres viennent du rendu corrige).

Les `explosion_lointaine_*` sont, eux, normalises **individuellement** a -1 dBFS : a leur
gain reel ils tombent 11 a 27 dB sous l'explosion proche et ne se jugent pas a l'oreille.
L'ecart mesure est conserve dans le manifeste (§6) — rien n'est perdu, seule l'audition est
rendue possible.

**Les medias viennent du `.pck`, pas de la banque.** Piege deja rencontre au lot V3B : la
version embarquee dans la banque (`DIDX`/`DATA`) est un prefetch tronque — 110 Ko pour
31 medias, contre 1,4 Mo dans le pack.

## 8. Negatifs mesures (ce qui n'existe PAS, verifie)

1. **Pas de couche de DEBRIS posee par les vehicules.** Les banques `sb_008_exp_ge_debris_*`
   (metal, metalhollow, verre, bois...) existent, mais leur remontee donne 16 `proj` et
   9 `eqip` — **aucun `vehi`**. Les debris appartiennent aux impacts de projectile.
2. **Pas de boucle d'epave qui brule.** `sb_003_lvl_positional_fo_fire_burningvehicle_loop_a`
   (banque `abe905e0`) remonte a `lsnd 0623af88 -> flsc 02971604 -> forg 00000051` : c'est un
   objet **Forge**, pas un vehicule detruit en partie.
3. **Pas de destruction propre aux objets-ENFANTS.** Le canon du Scorpion (`0000d4ff`), son
   collier (`0000d500`) et les trois tourelles de Warthog (`64b925eb`, `bcfb852f`,
   `dd7f9102`) ont chacun leur `hlmt`, et **aucun** n'apparait dans la remontee des banques
   d'explosion. L'explosion est posee **une seule fois, par le chassis**. C'est aussi la
   reponse pour le dossier `Falcon_LMG` : il ne recevra pas de `destruction/`.
4. **Rien n'est compose au runtime au-dela des 4 couches.** Pas de switch, pas de RTPC, pas
   de delai : l'evenement se rejoue tel quel. La seule variabilite est le tirage d'une
   variante parmi 6 par couche (et la fourchette RANGED de volume/hauteur, non cuite dans
   les fichiers — regle generale du chantier).

## 9. Le materiau compte : sol contre eau

Le tag `foot` n'est pas un simple relais, c'est une **table de materiau** : il apparie
systematiquement l'explosion de vehicule avec une explosion **en eau peu profonde**.

| vehicule | sol | eau peu profonde |
|---|---|---|
| Mongoose | `snd! 92da46d8` -> `exp_vehicle_small_unsc` | `snd! 7af8f944` -> `sb_008_exp_burst_small_watershallow` |
| Warthog, Wasp | `snd! 5600ccdd` -> `exp_vehicle_med_unsc` | `snd! 8ca4867a` -> `sb_008_exp_single_med_watershallow` |
| Scorpion | `snd! 2b49d0d7` -> `exp_vehicle_large_unsc` | `snd! 0c326dcc` -> `sb_008_exp_single_large_watershallow` |

Le lot livre le cas **SOL** (le seul pertinent pour le rejeu 2D actuel). Le cas eau est
localise et extractible par la meme recette si le besoin apparait.

## 10. CR honnete — une erreur d'attribution de rev9 mise au jour

**Ce n'est pas dans le perimetre de ce lot et ce n'est PAS corrige ici**, mais la mesure est
formelle et la taire serait malhonnete : **rev9 attribue au Ghost la banque `01862ab3`, qui
n'est pas la sienne.**

- **Appartenance** : l'intersection entre le pack `sb_010_veh_cv_ghost` (248 medias) et la
  banque `01862ab3` est **vide** — 1/248 en references `HIRC` (du bruit statistique) et
  **0/65** en medias embarques `DIDX`. La vraie banque du Ghost est **`ccd43fa8`**, dans
  **`pc/globals/common`** : 248/248 en reference ET en embarque.
- **Chaine de tags** : `01862ab3` est atteinte par les `vehi` dont le `hlmt` est `daf7f543`
  — le `hlmt` du **Warthog** (`00002705`, `cb96ca07`, `fe32c0f4` dans `any/globals`, plus
  `5159c8ef`, `75312e51`, `7617ff6e` dans `any/globals/common`).

Cause racine : le lot V3B/V3C n'a jamais ouvert le module **`pc/globals/common-rtx-new.module`**
(2,86 Go), qui porte **toutes** les banques covenant et bannis. La table complete, mesuree
aujourd'hui :

| pack | sbnk reel | module | reference / embarque |
|---|---|---|---|
| `sb_010_veh_cv_ghost` | `ccd43fa8` | pc/globals/common | 248 / 248 |
| `sb_010_veh_cv_wraith` | `fda12da2` | pc/globals/common | 78 / 78 |
| `sb_010_veh_cv_banshee` | `c682f736` | pc/globals/common | 230 / 230 |
| `sb_010_veh_bt_chopper` | `1bb9f097` | pc/globals/common | 78 / 78 |
| `sb_010_veh_cv_phantom` | `f25a0123` | pc/globals/common | 48 / 48 |
| `sb_010_veh_un_pelican` | `d6a4e1e2` | pc/globals/common | 27 / 27 |
| `sb_010_tur_cv_shadeturret` | `0a12db30` | pc/globals/common | 180 / 180 |
| `sb_010_veh_un_scorpion` | `05a51e0a` | pc/globals | 84 / 84 (rev9 confirme) |
| `sb_010_veh_un_rockethog` | `a52af042` | pc/globals | 78 / 78 (rev9 confirme) |
| `sb_010_veh_un_wargoose` | `38167604` | pc/globals | 73 / 73 (rev9 confirme) |
| `sb_010_veh_un_wasp` | `4993b379` | pc/globals | 268 / 268 (rev9 confirme) |

Consequence a traiter dans un lot dedie : les fichiers `Ghost/deplacement/*.wav` de rev9
(« souffle antigrav », « boost ») viennent de la banque du Warthog, pas du Ghost. Les
attributions UNSC de rev9, elles, tiennent. Le champ `correction_ghost` du manifeste rev10
porte cette alerte.

## 11. Outils ajoutes (`cmd/weapon-sounds`, additif)

Trois modes nouveaux, tous documentes en tete de leur fichier :

| fichier | mode | ce qu'il debloque |
|---|---|---|
| `pck_dump.go` (62 L) | `pck-dump` | `.pck` AKPK -> `.wem` COMPLETS, **sans module**. Ferme le trou documente depuis le scout S1 : cette etape passait par un script Python hors depot. |
| `pck_banques.go` (208 L) | `pck-banques` | N packs -> leur `sbnk`, en **une** charge de module. Deux liens mesures separement : `REF` (la banque cite les medias du pack, chunk HIRC) et `EMB` (elle les embarque, chunk DIDX). C'est ce mode qui a produit le §10. |
| `remonter_banque.go` (193 L) | `remonter-banque` | La chaine A L'ENVERS : banque -> `snd!` -> ... -> `vehi`. C'est ce mode qui a produit le §3. |

Pourquoi `pck-banques` et pas `trouverSbnk` existant : `trouverSbnk` echantillonne les
**24 identifiants les plus bas** du pack. Sur les familles covenant, ce sont precisement des
identifiants que la banque ne cite pas — score mesure 1/24, verdict faux. Le balayage complet
avec filtre 16 bits donne 248/248.

### Commandes de reproduction

```bash
# structure des 6 banques d'explosion (2 modules, un a la fois)
ws -mode eqip-arbre -banks 969f4dc6,c468fb55,94d43e95 -out arbre_unsc.json -emb emb/
ws -mode eqip-arbre -module pc/globals/common-rtx-new.module \
   -banks b1f8608b,3fdc61a7,2eaae6d7 -out arbre_cv.json -emb emb/

# a qui appartient chaque banque
ws -etroit -mode pck-banques -pck <dossier de .pck> -json pck_banques.json
ws -etroit -mode pck-banques -module pc/globals/common-rtx-new.module -pck <dossier> -json cv.json

# quel vehicule joue quelle banque
ws -etroit -mode remonter-banque -module any/globals/globals-rtx-new.module \
   -banks 969f4dc6,c468fb55,94d43e95 -limite 4
ws -etroit -mode remonter-banque -module any/globals/common-rtx-new.module \
   -banks b1f8608b,3fdc61a7,2eaae6d7 -limite 4

# quel evenement chaque snd! designe
ws -etroit -mode sndscan -module any/globals/globals-rtx-new.module -wem <events en decimal>

# les medias COMPLETS
ws -etroit -mode pck-dump -pck "<SFX>/sb_008_exp_vehicle_large_unsc.pck" -out wem/large_unsc
```

## 12. Limites assumees

- **Le juge final est l'utilisateur.** Toute la validation faite ici est indirecte (structure
  de tags + mesures `astats`). Aucun de ces sons n'a ete compare a une capture en jeu.
- **L'etiquette « lointaine »** des evenements `stai` est une lecture du spectre, pas un nom
  lu (§6).
- **Les gains inter-couches sont ceux de la banque ; le niveau absolu ne l'est pas.** Les
  bus, l'attenuation par distance et la fourchette RANGED vivent hors de l'evenement. Les
  fichiers sont normalises a -1 dBFS ; c'est un choix d'audition, pas une mesure du jeu.
- **Le cas « eau peu profonde » n'est pas rendu** (§9), seulement localise.
- **Le Phantom et le Pelican** ne sont pas livres : hors corpus Super Fiesta. Le Phantom
  partage l'evenement `1bf6fdde` du Wraith, c'est deja mesure.
