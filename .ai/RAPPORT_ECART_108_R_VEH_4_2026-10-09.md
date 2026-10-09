# Écarts de fin multiples de −108 bits (R-VEH-4) — rapport de la mission A

Date : 2026-10-09. Worktree `LevelUp-wt-ecart108`, branche `feat/ri-ecart-108` (base `3aa885e37`).
Recherche seulement : aucun fichier de production touché. Deux instruments `research` sont ajoutés, et
rien n'est commité (CLAUDE.md : demander avant tout commit).

- `apps/go-api/internal/games/halo_infinite/film/internal/grammar/ecart108_research_test.go` (`TestEcart108`)
- `.../grammar/ecart108_residus_research_test.go` (`TestEcart108Residus`)

Sorties sous `$O = C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/2efb0786-9fba-4f95-8fd0-464ee47b91d7/scratchpad/ri/ecart108/`.
Commande de référence (28 films, FILMS et CARTES de `d1/run.sh`, GOCACHE `go-build-ecart108`) :

```
OUT=$O/final RUN='^TestEcart108(Residus)?$' $O/run.sh      # -> $O/final.log, $O/final/ecart108.tsv, $O/final/ecart108_residus.tsv
awk -F'\t' -f $O/table_x2.awk $O/final/ecart108.tsv   > $O/final/table_x2.txt     # par archétype
awk -F'\t' -f $O/par_film.awk $O/final/ecart108.tsv   > $O/final/par_film.txt     # bipèdes par film / format
```

Reproductibilité : deux passes indépendantes donnent des sorties identiques à l'octet
(`diff -q final/ecart108.tsv m4/ecart108.tsv`, `diff -q final/ecart108_residus.tsv m5/ecart108_residus.tsv`).

## Verdict

**L'hypothèse tient, et elle est précisée.** Les 108·k bits qui séparent la fin de lecture d'un bipède
de l'ancre suivante sont **k entrées de slots libérés**. Une entrée libérée fait exactement 108 bits :
identifiant, archétype et mot +0xc valent tous `0xffffffff`, suivis de R(4) = 0 et de R(8) = 0. La table
d'image-clé est indexée par slot, avec une entrée par slot. **k est toujours égal à
`slot suivant − slot − 1`** (2 838 records sur 2 841). Si la marche lit ces entrées comme le jeu, le
nombre de bipèdes fermés passe de **5 401 à 8 242**. Les records admis par T1 et T2 restent à
**6 507** : les records gagnés étaient déjà de classe B, ou bien T1 les refuse.

## 1. Ce que le jeu lit (Ghidra, lecture seule, HaloInfinite.exe)

| Fait | Adresse |
|---|---|
| Le lecteur boucle sur un tableau d'entrées de 200 octets **pré-dimensionné**, de `param_2[0]` à `param_2[1]`, avec une borne `DAT_144dbfc90`. Ce global est posé à l'exécution et vaut 0 en statique. **Le flux ne porte aucun compte.** | `FUN_142e2bfd0` (`while (puVar12 != puVar1 && uVar18 < DAT_144dbfc90)`, `puVar12 += 0x32`) |
| En-tête d'entrée : R(32) → +0, R(32) → +4 (archétype), R(32) → +0xc, R(4) (`FUN_142e29cf8`) → +8, R(8) → +9. Total : **108 bits**. | `FUN_142e2bfd0` |
| Le corps n'est lu que si `puVar12[1] != 0xffffffff`. Il se compose de R(32) n1 (si n1 > 0 : `vtable[0x60]` sous `DAT_144e61ea0`, plus le contrôle R(32) du film), puis de R(32) n2 (si n2 > 0 : `vtable[0x88]` et `FUN_1428e2b68`). | `FUN_142e2bfd0` |
| L'écrivain écrit **toutes** les entrées du tableau avec le même en-tête. Si l'archétype est différent de `0xffffffff`, il écrit `n1 = (octet +8 & 1) ? +0x94 : 0`, puis l'état par défaut (`vtable[0x58]`, contrôle `0xffddcba`), puis `n2 = (octet +8 & 1) ? +0xc0 : 0`, puis les composants (`FUN_1428e38ec` → `FUN_142e2d6d4`). | `FUN_142e2d08c` |
| Le tableau compte **0x1fff entrées** (slots 0 à 0x1ffe), initialisées à zéro. | `FUN_1408be074` (`FUN_1411b2400(param_1, 0x1fff)`), `FUN_1406c7520` (memset 0), `FUN_1408be40c` |
| Chaîne de l'écrivain : la copie du tableau vivant (`*(DAT_144e61d78+8) + 0x42f0`) puis l'écriture. | `FUN_1428e339c` (1428e3573..1428e3589 : `ADD RDX,0x42f0` ; `CALL 141f850dc` ; puis `FUN_142e2d08c`) |
| Le tableau vivant est **indexé par slot** : `(eid & 0x3fffffff) * 200 + base`, avec la garde `*entree == eid`. | `FUN_1405d5dbc`, `FUN_1405d5e88`, `FUN_1406cb5f0` |
| **Libération d'un slot** : `+0 = +4 = +0xc = 0xffffffff` ; les tampons +0x98/+0x94 et +0xb8/+0xc0 sont libérés. | `FUN_1408f1948`, appelée par `FUN_1408f16c4` (slot rendu), elle-même appelée par `FUN_141fdab4c` quand le masque de références du slot tombe à 0 |
| Le client lit dans un tableau neuf de 0x1fff entrées, puis le copie dans le tableau vivant. | `FUN_142e2aab4` (`FUN_1408be074(&local_68)`, `FUN_1428e2a04`, `FUN_141f850dc(lVar1 + 0x42f0, &local_68)`) |

**Une entrée de 108 bits sans corps existe, et c'est la seule forme possible à 108 bits** : il faut un
archétype égal à `0xffffffff`, ce qui est l'état d'un **slot libéré**. Les autres formes sans corps sont
plus longues :
- archétype valide avec le bit 0 de l'octet +8 à zéro : l'écrivain pose n1 = n2 = 0 et l'entrée fait
  **172 bits** (108 + 32 + 32). Cette forme est observée 3 fois (archétype 41, mot +0xc à `0xffffffff`,
  voir le §2) ;
- entrée jamais allouée, mise à zéro : archétype 0, donc elle ferait aussi 172 bits. **Cette forme n'est
  jamais observée** : les slots de queue jamais alloués s'écrivent eux aussi comme des entrées libérées
  (§2, queue de table). Qui pose `0xffffffff` sur un slot jamais alloué : non lu.

Lu, mais non mesuré : l'écrivain écrit n1 > 0 même quand le tampon +0x98 ou le descripteur est nul,
et saute alors l'état par défaut, ce que le lecteur ne sait pas. C'est un cas limite possible ; il
n'est pas rencontré dans les mesures ci-dessous.

## 2. Mesure (28 films)

Population : records bipèdes d'image-clé. Base à la tête `3aa885e37` : 10 710 records, dont 5 401
fermés (le plan citait 5 399 sous la production de LK.3 ; la différence est de 2 records).

`$O/final/agg_bipedes.txt` (lignes `G` de `ecart108.tsv`) :

| Écart de fin | Records |
|---|---|
| 0 (fermés) | 5 401 |
| multiple de −108 | **2 841** |
| autre, en deçà | 896 |
| dépasse la frontière | 1 520 |
| arrêt du lecteur | 52 |

**Les k blocs de 108 bits** (`$O/final/verif_blocs.txt`, `$O/final/blocs.txt`) :
- **2 838 records** : les k blocs sont **tous** `id = arch = mot = 0xffffffff, R(4) = 0, R(8) = 0`
  (263 224 blocs), et **k = slot suivant − slot − 1 dans les 2 838 cas**. L'en-tête n'est pas
  « invalide » : il porte l'identifiant sentinelle, ce que `readKeyframeHeader` et `kfAnchorFromID`
  prennent pour une fin de table. C'est ce qui explique le « sans en-tête lisible, génération 0
  comprise » de R-VEH-4.
- **1 record** (`a521164d`, format 21, slots 543 → 549) : un seul bloc libéré, puis l'ancre, alors que
  l'écart de slots vaut 5. L'ancre ou la lecture est fausse : c'est un film de format 21, dont les
  valeurs sont fausses après i22 (D-5).
- **2 records** (`50247b26`, format 20, k = 22 et 21, écarts de slots 4 et 0) : des blocs quelconques,
  soit un multiple de 108 fortuit sur un film sans section d'identification.

**Ventilation par format** (`$O/final/par_film.txt`, colonnes m108 / fermés par R108s) : f20 8 / 6 ;
f21 7 / 6 ; **f24 897 / 897** (60ae07c4 : 0) ; **f25 205 / 205** ; **f27 1 724 / 1 724**. Le détail
film par film figure dans le même fichier : par exemple 4f77afc1 373 / 373, 084a804d 302 / 302,
1c4c63c2 239 / 239, e5adf7b2 205 / 205. Ces records sont presque tous sur des films sains des formats
24 à 27. La liste « par film » de `lk4/ecart_fin_lk3.txt` ventilait la classe « autre », pas la classe
−108.

**Contrôles de la forme sur la table entière** (`$O/final/ecart108_residus.tsv`, lignes `H`,
`$O/final/queue_table.txt`) :
- tête de table : sur 918 paquets, la lecture des slots 0 à (premier slot − 1) depuis le bit 1 atteint
  exactement la première ancre. En pratique, le premier slot est toujours 0 ;
- queue de table : sur **162 paquets**, lire depuis la fin du dernier record les `0x1ffe − slot`
  entrées restantes ne rencontre que des entrées libérées de 108 bits et **finit à moins d'un octet de
  la fin du payload**. La mesure confirme ainsi les 0x1fff entrées et les 108 bits par entrée
  libérée. Sur les 688 autres paquets, le dernier record, qui n'est pas borné, est mal lu ; la lecture
  qui suit se désaligne.
- les 3 records de f27 dont l'écart n'est pas un multiple de 108 et qui ferment pourtant sous Rslot
  (01e1f945, 64e8adfa, d9781168, slots 6xx → 1280/1297) contiennent chacun une entrée de **172 bits**
  (archétype 41, n1 = n2 = 0) que la marche d'ancres n'a pas vue, au milieu de 400 à 540 entrées
  libérées.

Le résidu restant chez les bipèdes ne vient pas de ces entrées : il est porté par f20-21 (2 055
records), `60ae07c4` (+33 bits, 331 records), et environ 120 records de f24-27 dont la lecture finit
dans une zone illisible, autrement dit des lectures fausses du bipède.

## 3. Règle proposée et effet simulé

**Règle (Rslot, fidèle au jeu).** La table d'image-clé compte une entrée par slot. Entre deux ancres de
slots s et t, la marche lit **exactement t − s − 1 entrées** par la grammaire d'entrée de `FUN_142e2bfd0`.
Chaque entrée est un en-tête de 108 bits. Si son archétype vaut `0xffffffff`, l'entrée s'arrête là ;
sinon, elle continue par n1, l'état par défaut et le contrôle, puis n2 et les composants
(`WalkKeyframeFullState`). Le record d'ancre s est **fermé** quand cette lecture atterrit exactement
sur l'ancre t.

Une variante stricte minimale (R108s) n'accepte que des entrées libérées, au nombre exact de t − s − 1.
Elle couvre les mêmes cas, sauf les 3 entrées de 172 bits.

Où la marche doit lire ces entrées (code relu, non modifié) :
1. `preuveDeLEtatComplet` (`marche_images_cles.go:200`) : remplacer `tr.EndBit == b.Want` par la
   lecture des entrées intercalées. `keyframeBorne` doit porter le slot suivant (il ne porte que le
   booléen `Voisin`).
2. `PreuveDImageCle.prouve` (`keyframe_world_preuve.go:108`) : sauter les entrées libérées avant
   `readKeyframeHeader`, et exiger que le slot lu vaille slot + 1 + k. **Non simulé de bout en bout** :
   la règle change les élections d'ancres. Indicateur mesuré : 215 161 → 236 374 records prouvés
   (+21 213), surtout ti = 10, 21, 29, 42 ; pour ti = 35 : 0 → 0, car `prouve` ne prouve aucun bipède
   dans ce contexte.
3. `readKeyframeHeader`, `WalkKeyframeRecords`, `ChainKeyframeRecords`, et le comptage de « fin de
   table » de `kfRecherche` : un identifiant `0xffffffff` porté par un archétype `0xffffffff` est un slot
   libéré de 108 bits, pas une sentinelle. La fin de table est l'entrée 0x1ffe.

**Effet simulé (règle appliquée à la fermeture seulement, ancres inchangées)** : commande et tables
ci-dessus, `$O/final/table_x2.txt`.

| | Avant | R108s | Rslot |
|---|---|---|---|
| Bipèdes fermés (ti = 35, 10 710) | 5 401 | 8 239 | **8 242** |
| Admis T1 ∧ T2 (règle U-1 amendée) | 6 507 | 6 507 | 6 507 |
| Témoin +1 bit, bipèdes : fermés / dont hasard | 223 / 3 | 326 / 4 | 325 / 4 |
| Tous archétypes bornés (460 722) : fermés | 240 470 | 267 849 | 267 811 |
| Records fermés perdus | — | 0 | **44** (tous ti = 37) |
| Témoin +1 bit, tous : fermés / hasard | 13 440 / 10 042 | 14 237 / 10 514 | 14 210 / 10 494 |

**Les admissions ne bougent pas** (lignes `gagne_r108` de `agg_bipedes.txt`). Des 2 839 bipèdes gagnés,
2 163 étaient déjà B ∧ T1 ∧ T2, 628 sont B ∧ T2 sans T1, 4 sont B sans T1 ni T2, et 44 sont C sans T1
ni T2. La règle fait passer des records de la classe B à la classe A, sans en faire entrer de nouveaux.

**Hausses notables hors bipèdes** : ti = 10 passe de 14 265 à 30 594, ti = 42 de 3 125 à 6 810, ti = 4
de 383 à 779, ti = 37 de 195 à 469, ti = 11 de 2 221 à 2 621, ti = 21 de 1 315 à 1 739, ti = 29 de 713 à
913, et ti = 13 de 13 625 à 13 984, soit 100 %.

**Pas de baisse, sauf une, qui est voulue** : sous Rslot, 44 fermetures de ti = 37 disparaissent. Ce
sont des records « fermés » alors que l'écart de slots est positif (`fermes_avant_ecartSlots>0`), ce
qui est impossible dans une table indexée par slot, où chaque entrée fait au moins 108 bits. Leur
témoin décalé fermait 24 fois, dont 18 par hasard ; il ferme **0 fois** sous Rslot. R108s ne fait
baisser aucun record.

**Le témoin décalé d'un bit monte avec les fermetures**, à peu près en proportion, sur ti = 38, 42, 43
et 41. Le hasard passe de 10 042 à 10 494, dont +182 sur ti = 42 pour +3 685 vraies fermetures, et
+128 sur ti = 38, +148 sur ti = 43. Ces deux archétypes absorbaient déjà le bit décalé avant la règle
(hasard 7 710 sur ti = 38, 2 177 sur ti = 43 ; D-99, R-VEH-8). Chez les bipèdes, le hasard reste à
4 sur 10 710.

## 4. Non lu, non mesuré

- La valeur d'exécution de `DAT_144dbfc90`, borne du lecteur : nulle en statique. La queue de table
  mesurée (162 paquets) est compatible avec 0x1fff entrées.
- Le code qui pose `0xffffffff` sur un slot jamais alloué (le tableau est mis à zéro par
  `FUN_1406c7520`) : non lu. Constaté seulement : la queue de table ne contient que des entrées
  libérées.
- L'effet de bout en bout de la règle sur la marche d'ancres (voisin, saut de largeur, élection
  sous preuve), donc sur la population des records elle-même : non simulé. Seule la fermeture à
  ancres inchangées l'est, avec l'indicateur de preuve du §3.
- Les ~120 records de f24-27 encore non fermés, les formats 20-21 et `60ae07c4` : hors de ce sujet.
