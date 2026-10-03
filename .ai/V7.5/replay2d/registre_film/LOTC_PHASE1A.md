# Lot C — phase 1a : la grammaire des canaux de zone, lue dans le binaire (LECTURE SEULE)

> Arbitrage qui ouvre cette phase : `LOTC_ARBITRAGE_PHASE0.md` (commit `bccee53a9`).
> Perimetre : C.1a.0 et C.1a.1 + C.1a.2. **Aucun code Go de production modifie** — ni
> `traverse.go`, ni un `components_*.go` (la plomberie appartient au lot 0, le port est la phase
> 1b). Ghidra : instance de l'utilisateur, `HaloInfinite.exe` (311 104 fonctions), image base
> `0x140000000`, outils `mcp__ghidra__*` en LECTURE SEULE — aucun rename, aucun commentaire,
> aucun script, aucune analyse, aucune sauvegarde.
> Mesures du 2026-08-17. Gates : `LOTC_gates.log`. Sorties : `lotC/*.tsv`.

## 1. C.1a.0 — la largeur du champ de slot : l'hypothese est REFUTEE

L'item partait d'une decouverte de la phase 0 : `matchWorldObjectRecord` lit le slot sur 13 bits
fixes alors que `FrameConfig.IDLowBits` est une valeur de RUNTIME (`frame_records.go:39-44` :
11 sur `000d5950`, 14 sur le film de la capture live). Instrument :
`zone_census_width_test.go` — rejoue le recensement sous cinq largeurs (11 a 15) en une passe
disque, sans toucher a `matchWorldObjectRecord`.

**DEUX JUGES INDEPENDANTS, et ils concordent sur les trois films.**

1. Le balayage de cadre DEJA present au depot (`BestVariant` + `KFQFrameVariants`,
   `keyframe_entity_queue.go`) — il ne connait rien aux bandes de slots :

| film | couverture moyenne par largeur | retenue |
|---|---|---|
| 7344d24f | w=11 : 0,003 · w=12 : 0,049 · **w=13 : 0,380** · w=14 : 0,132 · w=15 : 0,009 | 13 |
| 530820e5 | w=11 : 0,003 · w=12 : 0,029 · **w=13 : 0,270** · w=14 : 0,152 · w=15 : 0,005 | 13 |
| 000d5950 | w=11 : 0,003 · w=12 : 0,003 · **w=13 : 0,701** · w=15 : 0,010 | 13 |

2. La purete de l'ancrage, dont le meilleur juge est **ti=4** : UN slot, UN composant declare,
   donc un record de ti=4 ne peut annoncer QUE i0 — toute autre annonce est une faute de largeur.

| film | w=11 | w=12 | **w=13** | w=14 | w=15 |
|---|---|---|---|---|---|
| 7344d24f | 98,18 % (220 rec.) | 96,03 % (126) | **0,68 % (35 698)** | 100,00 % (35) | 90,00 % (10) |
| 530820e5 | 99,69 % (1 272) | 98,18 % (110) | **0,21 % (28 647)** | 100,00 % (21) | 100,00 % (15) |
| 000d5950 | 98,59 % (283) | 97,24 % (181) | **1,33 % (30 682)** | 97,10 % (69) | 100,00 % (18) |

**Part hors grammaire AVANT / APRES, ce que l'item demandait** : il n'y a pas d'APRES, parce que
l'APRES est l'AVANT. La largeur cablee 13 est la bonne sur les trois films, `000d5950` COMPRIS —
celui-la meme que `frame_records.go` donne a `idLow=11`.

| film | archetype | w=11 | w=12 | **w=13 (avant = apres)** | w=14 | w=15 |
|---|---|---|---|---|---|---|
| 7344d24f | ti=10 | 50,44 % | 56,75 % | **22,56 %** | 86,23 % | 83,00 % |
| 7344d24f | ti=12 | 28,29 % | 64,18 % | **13,98 %** | 62,42 % | 61,61 % |
| 7344d24f | ti=47 | 88,18 % | 96,49 % | **4,71 %** | 96,92 % | 98,40 % |
| 530820e5 | ti=10 | 49,79 % | 60,09 % | **30,84 %** | 65,26 % | 56,34 % |
| 530820e5 | ti=12 | 59,93 % | 52,53 % | **34,22 %** | 62,57 % | 60,87 % |
| 530820e5 | ti=47 | 94,98 % | 96,18 % | **21,96 %** | 91,93 % | 97,90 % |

**Consequences, ecrites franchement.**

- La decouverte n2 de la phase 0 est REFUTEE : le residu hors grammaire de 22 a 39 % n'est PAS
  une faute de largeur de slot. Sa cause reste celle deja chiffree en phase 0 — la faible
  selectivite d'un en-tete de 21 bits sur une bande large (le plancher de bruit).
- Le « 11 sur `000d5950` » de `frame_records.go:39-44` n'est reproduit par AUCUN des deux juges
  sur le chemin de l'ancrage. Les deux grandeurs ne sont pas la meme chose : la boucle de records
  consomme une AMORCE DE PAQUET une fois par paquet (`DefaultPacketPreambleBits = 2`) puis un id
  par record, tandis que l'ancrage glisse sur toutes les positions de bit et n'a pas d'amorce.
  Que `11 + 2 = 13` est une coincidence arithmetique qu'il serait imprudent de presenter comme
  une explication : trancher demande la BOUCLE, pas l'ancrage, et c'est hors du perimetre.
- Gain non prevu : **ti=4 est un temoin POSITIF d'ancrage que la phase 0 n'avait pas** — sur une
  bande d'un seul slot, l'ancrage est pur a 98,7-99,8 %. Le bruit de la phase 0 est donc bien un
  effet de LARGEUR DE BANDE, pas un defaut du reconnaisseur.

## 2. La methode, et pourquoi elle a ete rapide

Recette R7-d, appliquee telle quelle et sans surprise : **chaine du nom dans `.rdata` -> getter de
nom -> case de vtable -> `+0x30` = le lecteur**. Deux raccourcis par rapport a R7-d, tous deux
verifies : (a) le pont `mcp__ghidra__*` FONCTIONNE (R7-d le donnait HS et avait du dumper 14 Mo de
`.rdata` en 55 lectures) ; (b) `get_xrefs_to` sur le getter de nom rend DIRECTEMENT la case de
vtable, ce qui remplace le balayage de pointeurs 8 octets.

Le layout de vtable de R7-d est confirme sur les CINQ descripteurs lus, a l'octet :

| case | contenu | valeur partagee par les 5 |
|---|---|---|
| +0x00 | destructeur | `0x14117b4a0` (4/5) |
| +0x08 | getter de nom (`lea rax,[rip+X]; ret`) | per-composant |
| +0x10 | `ret false` | `0x1404ab600` |
| +0x18 | **ECRIVAIN** | per-composant |
| +0x20 | `int3` | `0x1411c8f80` |
| +0x28 | thunk | `0x14076ce9c` |
| +0x30 | **LECTEUR** | per-composant |
| +0x38 | `ret 1` | `0x1404ab600` |
| +0x40 | `*p = 0` | `0x141191ab0` |
| +0x48 | zero-16 | `0x14076ced0` |

**Le thunk `+0x28` est un PUR FORWARDER** : `FUN_14076ce9c` = `(**(code**)(*param_1 + 0x30))()`,
zero bit consomme. C'etait une question ouverte et elle est tranchee : **la charge utile d'un
composant commence exactement la ou le masque finit**, il n'y a pas de prefixe de wrapper sur le
chemin delta. C'est ce qui rend les vecteurs de la section 4 exploitables tels quels.

Primitives de lecture identifiees (partagees) :

| primitive | grammaire | preuve |
|---|---|---|
| `FUN_1406d84b4(rdr, rdr, min, max, bits, b, b)` | flottant QUANTIFIE sur `bits` bits, dequantifie dans `[min, max]`, INCONDITIONNEL | `bits` est le 5e argument (pile `+0x20`) : `MOV dword ptr [RSP + 0x20],0x8` a `140fc8d3b`. Le resultat sort en XMM0. |
| `FUN_1406cf008(rdr)` | **R(1)** | `+0x2c += 1`, decalage de 1 (deja identifie `frame_records.go:50-53`) |
| `FUN_1407ef804(rdr, 0, out)` | **R(4)**, puis `*out = valeur - 1` | `+0x2c += 4`, `>> 0x3c` |

Constantes de plage lues : `0x143cd84ec` = **-1.0f**, `0x143cd8374` = **+1.0f**,
`0x143cd8774` = **-10000.0f**, `0x143cd8770` = **+10000.0f**.

## 3. Les adresses, par composant

| ti | i | composant | chaine `.rdata` | getter de nom | vtable | ecrivain `+0x18` | **LECTEUR `+0x30`** |
|---|---|---|---|---|---|---|---|
| 12 | 14 | `managed-navpoint-radial-progress` | `0x143c94e28` | `0x141177d10` | `0x143d08120` | `0x142edb0c8` | **`0x140fc8d14`** |
| 10 | 1 | `managed-object-boundary-color-component` | `0x143c94a78` | `0x141178000` | `0x143d09350` | `0x142edb1e4` | **`0x142ed52b4`** |
| 13 | 1 | `managed-object-property-component` | `0x143c94bc8` | `0x141178020` | `0x143d08b20` | `0x142edcb74` | **`0x140ce5554`** |
| 13 | 2..33 | `managed-object-player-masked-property-component` | `0x143c94b70` | `0x14064df10` | `0x143c96fa0` | `0x142edcb44` | **`0x140ce593c`** |
| 10 | 26..29 | `managed-object-rtpc-component` | `0x143c952c8` | `0x1411720c0` | `0x143c97228` | `0x142edb444` | **`0x140796d38`** |

Note : un composant declare N fois dans un archetype (les 32 `player-masked-property` de ti=13,
les 4 `rtpc` de ti=10, les 16 `navpoint` de ti=10) partage UN SEUL descripteur, donc UN SEUL
lecteur. `rtpc` le montre explicitement : `lVar6 = *(int *)(param_1 + 8)` lit l'index du
composant DANS LE DESCRIPTEUR et ecrit a `etat + 0x17c + index*8` — c'est ainsi que i26, i27, i28
et i29 vont dans des champs differents avec le meme code.

## 4. Les grammaires, bit a bit

### 4.1 `ti=12 i14 managed-navpoint-radial-progress` — RESOLU, 8 bits

`FUN_140fc8d14` en entier (une seule lecture) :

```
R(8) -> flottant quantifie dans [-1.0, +1.0] -> etat+0x704
```

Inconditionnel, aucune porte, aucune dependance au niveau du registre ni a un etat runtime.
**Statut de portage visé : `porte`** (tous les retours rendent `true` : le lecteur finit sur
`MOV AL,0x1`). Largeur totale : **8 bits**.

Disassemblage integral, parce qu'il tient en dix instructions et qu'il fonde la largeur :

```
140fc8d1a  MOVSS XMM3,[0x143cd8374]   ; max = +1.0f      (arg4)
140fc8d22  MOV   RCX,RDX              ; arg1 = lecteur de bits
140fc8d25  MOVSS XMM2,[0x143cd84ec]   ; min = -1.0f      (arg3)
140fc8d2d  MOV   RBX,[R8+0x10]        ; etat = param_3+0x10
140fc8d31  MOV   byte [RSP+0x30],0x1  ; arg7 = 1
140fc8d36  MOV   byte [RSP+0x28],0x1  ; arg6 = 1
140fc8d3b  MOV   dword [RSP+0x20],0x8 ; arg5 = 8 = LARGEUR
140fc8d43  CALL  0x1406d84b4
140fc8d48  MOVSS [RBX+0x704],XMM0     ; le flottant dequantifie
140fc8d50  MOV   AL,0x1               ; -> porte
```

### 4.2 `ti=10 i1 managed-object-boundary-color-component` — RESOLU, 32 bits

`FUN_142ed52b4`, quatre lectures identiques, `XMM2` (min) et `XMM3` (max) charges UNE fois et
jamais rechargés entre les appels :

```
R(8) -> flottant dans [0.0, +1.0] -> etat+0x04    (r)
R(8) -> flottant dans [0.0, +1.0] -> etat+0x08    (g)
R(8) -> flottant dans [0.0, +1.0] -> etat+0x0c    (b)
R(8) -> flottant dans [0.0, +1.0] -> etat+0x10    (a)
```

`min = 0.0` vient de `XORPS XMM2,XMM2` (`142ed52c8`), pas d'une constante. Les quatre largeurs
sont chacune un `MOV dword ptr [...],0x8` explicite (`142ed52dd`, `142ed52f6`, `142ed5315`,
`142ed5334`). Inconditionnel. **Statut visé : `porte`.** Largeur totale : **32 bits**.

### 4.3 `ti=10 i26..i29 managed-object-rtpc-component` — RESOLU, 32 ou 54 bits (DATA-DEPENDANT)

`FUN_140796d38` :

```
R(32) -> identifiant                          -> etat + 0x17c + index*8
si identifiant == 0 :  etat + 0x180 + index*8 = 0x800000   (AUCUN bit lu)
sinon               :  R(22) -> flottant quantifie dans [-10000.0, +10000.0]
                                              -> etat + 0x180 + index*8
```

Le `R(32)` est INLINE dans le lecteur (pas d'appel), le `R(22)` passe par `FUN_1406d84b4` avec
`0x16` en 5e argument. La largeur du record depend donc de la DONNEE :
**32 bits si l'identifiant est nul, 54 sinon**. Par la convention de la table ECS, un retour
data-dependant se declare **`partiel`**, pas `porte`.

### 4.4 `ti=13 i1` et `ti=13 i2..i33` — STOP, avec ce qui est acquis

Les deux composants de ti=13 partagent leur coeur binaire, et c'est un **type VARIANT a 11
branches** — pas un champ.

- `FUN_140ce5554` (i1) prepare l'etat (`etat+0x84 = 0`, sentinelle `0xffffffff`) puis appelle
  **`FUN_140ce59bc`**.
- `FUN_140ce593c` (i2..i33) enveloppe le MEME `FUN_140ce59bc` dans un conteneur a petit tampon
  optimise (local de 136 octets, `FUN_140ce5b90` pour la copie/echange) : c'est un TABLEAU de
  proprietes, de longueur variable.
- `FUN_140ce59bc` lit **`R(4)`** (`+0x2c += 4`, decalage de 4, `>> 0x3c`) puis dispatche sur
  `FUN_140ce5aa4`.

Table de dispatch lue (le tag de 4 bits vaut 0..15) :

| tag | branche | grammaire |
|---|---|---|
| 0 | `etat+0x80 = 0` | **0 bit** (valeur vide) |
| 1 | `FUN_1407ef804(rdr, 0, out)` | **R(4)**, puis `-1` (enumeration) ; garde `bits restants >= 0x20` |
| 2 | `FUN_1406cf008(rdr)` | **R(1)** (booleen) ; meme garde |
| 3 | `FUN_140ce558c` | non lu (STOP) |
| 4 | `FUN_140ce5720` | non lu (STOP) |
| 5 | `FUN_14080dec4(rdr, "string-id-value", out)` | non lu ; le nom de champ est un parametre MORT garde en retail (meme signature que le `W(32)` de R7-d) |
| 6 | `FUN_141d0f344(rdr, 0, out)` | non lu (STOP) |
| 7 | `FUN_141fce140` | non lu (STOP) |
| 8 | `FUN_140ce5820` | non lu (STOP) |
| 9 | `FUN_141fce218` | non lu (STOP) |
| 10 | `FUN_141fce3a8` | non lu (STOP) |
| >= 11 | `FUN_141fce2f0()` (appele SANS argument) | vraisemblablement une assertion / un stub |

**STOP prononce ici, et pourquoi.** La borne du plan est d'un jour executeur par composant.
Resoudre ti=13 demande onze sous-grammaires PLUS le conteneur de longueur variable, soit un
travail d'un ordre de grandeur au-dessus des trois canaux deja resolus, pour un canal dont
l'interet reste a etablir (la phase 0 le donne a 40,6x le plancher, contre 140x et 868x pour
`radial-progress`). Ce qui est acquis — le tag de 4 bits, les onze adresses de branche, deux
branches deja resolues (R(4) et R(1)) — suffit a reprendre sans refaire.

## 5. Vecteurs de test, depuis des octets de film reels

Instrument : `zone_vectors_test.go`. **La position de la charge utile n'est pas devinee** : seuls
les records dont le masque vaut EXACTEMENT `{i}` sont retenus, et pour ceux-la la charge utile
commence a `rec.After` (ce que la section 2 etablit : le thunk ne consomme rien). Denominateurs
publies. TSV : `lotC/<short8>_vecteurs.tsv`.

### `ti=12 i14 radial-progress` — 17 244 records singleton (`7344d24f`), 17 612 (`696a9d7c`)

| film | chunk | paquet | bit du record | bit de la charge | slot | gen | bruts | quantum | valeur |
|---|---|---|---|---|---|---|---|---|---|
| 7344d24f | 2 | 2066 | 1454 | 1481 | 1624 | 1 | `0x7F` | 127 | -0,0039 |
| 7344d24f | 2 | 2068 | 3905 | 3932 | 1624 | 1 | `0x80` | 128 | +0,0039 |
| 7344d24f | 2 | 2070 | 1669 | 1696 | 1624 | 1 | `0x80` | 128 | +0,0039 |
| 696a9d7c | 3 | 236 | 1234 | 1261 | 1624 | 1 | `0x7F` | 127 | -0,0039 |

**Ce que les valeurs disent, et c'est directement le gate G-C1.** La distribution est LISSE et
CENTREE : 159 valeurs distinctes sur 17 244 records (`7344d24f`), 128 sur 17 612 (`696a9d7c`),
aucune au-dela de 1,2 % — la signature d'un canal continu, pas d'un enumere. Les huit valeurs les
plus frequentes sont 129, 132, 135, 138, 141, 144, 147, 154 — **au-dessus de 128 et espacees
d'environ 3** : une RAMPE regulière. Et 128 sur 8 bits dans `[-1, +1]` est le MILIEU de la plage,
c'est-a-dire zero. Lecture qui en decoule (a confirmer en phase 1b, pas ici) : la progression est
a 0 au repos et monte par increments constants — exactement la forme que G-C1 exige
(« rampes monotones 0 -> max puis remise a zero »).

### `ti=10 i1 boundary-color` — 512 records singleton (`7344d24f`), 467 (`696a9d7c`)

| film | chunk | paquet | bit de la charge | slot | gen | bruts | r / g / b / a (quanta) |
|---|---|---|---|---|---|---|---|
| 7344d24f | 2 | 1684 | 304 | 2478 | 1 | `0x358B4A2B` | 53 / 139 / 74 / 43 |
| 7344d24f | 2 | 1686 | 304 | 2478 | 1 | `0x3541CACB` | 53 / 65 / 202 / 203 |
| 696a9d7c | 3 | 506 | 1555 | 2060 | 0 | `0x77E7C106` | 119 / 231 / 193 / 6 |

Distribution du PREMIER octet : 112 valeurs distinctes sur 512 (`7344d24f`), 66 sur 467
(`696a9d7c`), mais tres concentrees — les trois premieres couvrent 37 % et 52 %. Les valeurs
dominantes sont **55, 119, 183, 247**, c'est-a-dire `55 + 64k` : memes six bits de poids faible,
deux bits de poids fort variables.

**Piste ecartee, et il faut le dire.** J'ai d'abord lu la un decalage de deux bits avant la charge
utile. C'est REFUTE : le thunk `+0x28` ne consomme rien (section 2) et le lecteur fait quatre
`R(8)` explicites. Ces quatre valeurs sont donc une propriete de la DONNEE — un canal rouge a
quatre niveaux espaces d'un quart. Leur interpretation (palette a quatre niveaux, ou frontiere de
champ ailleurs dans la structure) n'est PAS tranchee par cette passe : c'est une question de
phase 1b, ou le critere G-C1 (« <= 8 valeurs distinctes ») portera sur le QUADRUPLET et non sur
le premier octet.

### `ti=10 i26 / i27 rtpc` — 6 696 / 17 026 records singleton (`7344d24f`)

Le vecteur confirme la grammaire de facon INDEPENDANTE, et c'est le controle le plus fort de
cette phase : sous l'hypothese `R(32)` identifiant en tete, l'identifiant doit etre CONSTANT pour
un composant donne — un cadrage faux ne rendrait pas une constante.

| ti/i | film | bruts (64 bits) | identifiant R(32) | 22 bits suivants |
|---|---|---|---|---|
| i26 | 7344d24f | `0x0685454080000C40` | `0x06854540` | `0x200003` |
| i26 | 7344d24f | `0x0685454080001840` | `0x06854540` | `0x200006` |
| i26 | 7344d24f | `0x0685454080002840` | `0x06854540` | `0x20000A` |
| i26 | 696a9d7c | `0x0685454080000C40` | `0x06854540` | `0x200003` |
| i27 | 7344d24f | `0x7CBF00668000C665` | `0x7CBF0066` | `0x200031` |
| i27 | 7344d24f | `0x7CBF006680018E65` | `0x7CBF0066` | `0x200063` |
| i27 | 696a9d7c | `0x7CBF00668000C665` | `0x7CBF0066` | `0x200031` |

**L'identifiant est constant par composant ET IDENTIQUE SUR LES DEUX FILMS** (`0x06854540` pour
i26, `0x7CBF0066` pour i27) : ce sont deux identifiants de parametre Wwise cables dans la carte
ou dans le mode. Et la valeur de 22 bits MONTE de facon monotone d'un paquet au suivant
(`0x200003` -> `0x200006` -> `0x20000A`), avec son bit de poids fort a 1 — soit, sur
`[-10000, +10000]`, une valeur juste au-dessus de zero qui croit. Le cadrage `R(32)` + `R(22)` est
donc confirme par la donnee, en plus de l'etre par le desassemblage.

## 6. Tableau de synthese — confiance par composant

| ti | i | composant | lecteur | grammaire | largeur | vecteurs | statut visé | confiance |
|---|---|---|---|---|---|---|---|---|
| 12 | 14 | `radial-progress` | `0x140fc8d14` | `R(8)` -> flottant `[-1, +1]` | 8 | 4 (2 films) | `porte` | **HAUTE** — lecteur complet en 10 instructions, largeur explicite en immediat, distribution conforme a un canal continu |
| 10 | 1 | `boundary-color` | `0x142ed52b4` | `4 x R(8)` -> flottants `[0, 1]` (RGBA) | 32 | 3 (2 films) | `porte` | **HAUTE** pour la grammaire, MOYENNE pour le sens du premier octet (4 niveaux non expliques) |
| 10 | 26..29 | `rtpc` | `0x140796d38` | `R(32)` id ; si id != 0 : `R(22)` -> flottant `[-10000, +10000]` | 32 ou 54 | 7 (2 films) | `partiel` (retour data-dependant) | **HAUTE** — identifiant constant par composant et identique sur 2 films, valeur monotone |
| 13 | 1 | `managed-object-property` | `0x140ce5554` -> `0x140ce59bc` | `R(4)` tag + 11 branches | variable | — | STOP | tag et table de dispatch : HAUTE ; branches 3-10 : NON LUES |
| 13 | 2..33 | `player-masked-property` | `0x140ce593c` -> `0x140ce59bc` | conteneur de longueur variable autour du meme tag | variable | — | STOP | idem, plus un conteneur non lu |

## 7. Ce qui n'est PAS fait, et pourquoi

- **Le port Go** : hors perimetre par construction (phase 1b, apres la fusion du lot 0). Aucun
  `case`, aucune ligne de table ECS editee, aucun hook pose.
- **ti=13 (i1 et i2..i33)** : STOP prononce sur la borne d'effort (section 4.4). Huit des onze
  branches du variant ne sont pas lues.
- **La convention de dequantification** (milieu d'intervalle contre bornes incluses) n'est pas
  etablie : `FUN_1406d84b4` rend son flottant en XMM0 et la decompilation ne montre pas le calcul.
  L'ecart vaut un demi-quantum (0,4 % sur 8 bits). Les vecteurs publient le QUANTUM BRUT a cote de
  la valeur, donc rien n'en depend ; a trancher en phase 1b si une valeur exacte est exigee.
- **Les ecrivains `+0x18`** ne sont pas decompiles : trois des cinq adresses n'ont pas de fonction
  definie dans le programme de l'utilisateur, et creer l'analyse est interdit (lecture seule).
  R7-d a deja etabli le miroir `+0x18` / `+0x30` sur cinq composants du bipede ; ici le lecteur
  suffit, et la corroboration vient des vecteurs (section 5) au lieu de l'ecrivain.
- **C.0.2** (32 drapeaux de `ti=10 i0`) : attend toujours le hook du lot 0.

## 8. Decouvertes (hors perimetre — notees, NON traitees)

1. **`get_xrefs_to` sur le getter de nom rend directement la case de vtable.** R7-d avait dumpe
   14 Mo de `.rdata` en 55 lectures `read_memory` pour trouver ces pointeurs, faute d'un pont MCP.
   Le pont fonctionne et la recette se fait en 3 appels par composant. A porter dans
   `PLAN_R7D_ECRIVAIN_VTABLE.md` comme raccourci de methode.
2. **Le thunk `+0x28` (`FUN_14076ce9c`) est un pur forwarder, zero bit.** C'est un fait de
   FORME reutilisable par tous les lots de portage : sur le chemin delta, la charge utile d'un
   composant commence exactement ou le masque finit. `WALK_PORT_NOTES.md` §2 refutait l'hypothese
   du wrapper pour le chemin d'image-cle ; elle est maintenant refutee pour le delta aussi.
3. **`ti=4` est le meilleur temoin d'ancrage du corpus** (1 slot, 1 composant declare : toute
   annonce hors i0 est une faute). Aucun instrument du depot ne s'en sert. Il donnerait un
   controle a cout nul aux scanners de production ti=37/41/42.
4. **Les identifiants Wwise des `rtpc` sont des constantes de carte/mode** (`0x06854540`,
   `0x7CBF0066`, identiques sur les deux Strongholds). Ils forment une cle d'identification de
   canal sonore, potentiellement un pont vers le mode joue — sans rapport avec le lot C, mais
   personne ne l'a note.
5. **Le premier octet de `boundary-color` prend quatre valeurs espacees de 64** (55, 119, 183,
   247 : memes 6 bits de poids faible). Non explique. A regarder en phase 1b avec le quadruplet
   complet.
6. `FUN_1407ef804` lit `R(4)` puis rend `valeur - 1` : une enumeration dont la valeur 0 du flux
   signifie « absent ». Motif a connaitre avant de porter un enumere.
