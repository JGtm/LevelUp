# T7 — Dispositifs `ti=43` et moteur `ti=2` (campagne grammaire, phase 1, 2026-10-01)

> Piste T7 du plan `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (etape 2). Sans production :
> aucune commande `go`, aucun film decode, aucune base ouverte, rien d'ecrit hors de ce fichier.
> Sources : `HaloInfinite.exe` dans Ghidra, LECTURE SEULE (HTTP `127.0.0.1:8089` :
> `decompile_function`, `disassemble_function`, `read_memory`, `search_strings`, `get_xrefs_to`) ;
> code Go a la tete du worktree (`feat/campagne-grammaire` = `69564ef7d`) ; octets de `chunk_00`
> de sept films du cache, lus en place (`grep -a -b`, `dd`, `od`), sans decodage.
>
> Ce que cette note PROLONGE (et ne refait pas) : `NOTE_3_6_TI43_2026-09-16.md` (les 22 lecteurs
> nommes) et `NOTE_3_6_TI43_GRAMMAIRES_A/B/C/D_2026-09-17.md` (les 22 grammaires, cote LECTEUR) ;
> `NOTE_3_7_REAPPARITION_2026-09-17.md` §2 et §2 bis (`ti=2 i11, i13, i14, i15`, cote lecteur,
> lu a `objdump`). Ce qu'elle AJOUTE : (1) la contre-epreuve cote ECRIVAIN (serialiseur en
> `descripteur + 0x28`) pour les composants a poids de fermeture ; (2) le releve de `ti=2 i16` et
> `i17`, jamais faits ; (3) la preuve, lue dans les registres des films, que les blocs `ti=43` et
> `ti=0/2` sont les memes sur tous les builds du cache ; (4) le lien entre un NEW de dispositif
> qui desynchronise et les rejets hors datum de la vue B ; (5) les vecteurs de test construits
> d'apres l'ecrivain et la mesure qui prouvera l'effet.

---

## 0. L'etat, colle depuis le depot et la carte

`internal/grammar/testdata/ecs_table.tsv` : `ti=43 i19` a `i40` = **22 lignes `non_porte`** (seule
`i19` porte une adresse et une grammaire) ; `ti=2 i11, i13, i14, i15, i16, i17` = `non_porte`.

Code : le dispatch est PAR NOM (`traverse.go`, `traverseComponentLoopFrom` ->
`consumeByNameCapturing(br, arch.Components[i], t.TypeIndex, arch.Level(i))`). Le seul composant
`device-*` porte est `device-position-component` (`dispatch_biped.go:197`,
`consumeDevicePosition` = `Skip(15)`, `components_walk_batch9.go:41`). Aucun `case` pour les 22
autres ni pour `managed-engine-timers`, `scenario-intro`, `matchflow-isplaying-flags` : un
composant present au masque et sans `case` pose `t.DesyncAt = i` et arrete le record
(`traverse.go:283-285`). Les lecteurs de `ti=2 i11/i13/i14/i15` existent seulement en recherche
(`reapparition_37_bassin_film_lecteurs_test.go`, tag `research`).

Carte de fermeture du 2026-09-26 (`CARTE_FERMETURE_2026-09-26.md`, 18 films), premieres causes
d'arret qui relevent de cette piste :

| Rang | Cause | Paquets | Gain borne (records utiles lus non fermes) | Builds |
|---|---|---:|---:|---|
| 3 | `ti=43 i35 device-animation-layer-state` | 12 113 | 67 207 | HI_1_12_0, HI_1_8_0 |
| 5 | `ti=2 i15 managed-engine-timers` | 3 635 | 27 | 8 builds |
| 9 | `ti=43 i21 device-position-group` | 599 | 8 447 | 7 builds |
| 20 | `ti=2 i17 matchflow-isplaying-flags` | 70 | 13 | 9 builds |
| 21 | `ti=2 i11 game-engine-soft-ceilings` | 66 | 376 | 5 builds |
| 24 | `ti=43 i39 device-machine-flags` | 63 | 743 | 5 builds |
| 31 | `ti=2 i14 GameEngineComposerLetterboxComponent` | 42 | 429 | 4 builds |
| 34 | `ti=43 i19 device-position-animation-name` | 37 | 333 | 6 builds |
| 36, 39 | `ti=0 i11`, `ti=0 i13` | 34, 27 | 216, 161 | |

**Ou tombe `i35`.** HI_1_8_0 (`60ae07c4`) : vue B atteinte 46 436, vue C atteinte 45 153, donc
1 283 paquets au plus s'arretent en vue B ; le reste de la ligne `i35` est sur HI_1_12_0 :
**au moins 10 830 des 12 425 arrets en vue B de `bcb6d393`** (21 581 atteints - 9 156 vue C). Sur
ce film, `i35` est donc l'arret de pres de la moitie des paquets (21 864). Il n'est la premiere
cause sur AUCUN film HI_1_13_0.

---

## 1. Les 23 descripteurs, relus aujourd'hui (image HI_1_13_0)

Image : `search_strings HI_1_` -> `0x1436a37c0` `HI_1_13_0`, `0x143b968d8`
`269225.26.04.08.1618.hi_1_13_0`. C'est l'image des notes du 16-17/09 : les 23 pointeurs
`descripteur + 0x40` lus ce jour (`read_memory`, 8 octets) concordent tous.

| Composant | Descripteur | `+0x40` lecteur | `+0x28` ecrivain |
|---|---|---|---|
| `ti=43 i19` | `0x143d0ce98` | `0x1410156e4` | `0x142f05c14` |
| `i20` | `0x143d0cfd8` | `0x142f02d04` | `0x142f05bf8` (pas une fonction dans Ghidra) |
| `i21` | `0x143d0cf38` | `0x1407f0678` | `0x142f05cd0` |
| `i22` | `0x143d0cf88` | `0x14100d310` | `0x142f05d50` |
| `i23` | `0x143d0c200` | `0x14100d2d0` | `0x142f05d90` |
| `i24` | `0x143d0c1a8` | `0x141167910` | — |
| `i25` | `0x143d0c2f0` | `0x142f02c54` | — |
| `i26` | `0x143d0c250` | `0x142f029e4` | — |
| `i27` | `0x143d0c0b0` | `0x140bee524` | — |
| `i28` | `0x143d0c008` | `0x142f02bec` | — |
| `i29` | `0x143d0c058` | `0x14116fcb0` | — |
| `i30` | `0x143d0c158` | `0x142f02c20` | — |
| `i31` | `0x143d0c100` | `0x142f02a48` | — |
| `i32` | `0x143d0c4d8` | `0x142f02bcc` | — |
| `i33` | `0x143d0c528` | `0x142f02b7c` | — |
| `i34` | `0x143d0c488` | `0x140f44104` | `0x142f056a4` |
| `i35` | `0x143d0c578` | `0x141076f68` | `0x142f0570c` |
| `i36` | `0x143d0c390` | `0x142f02bb0` | — |
| `i37` | `0x143d0c340` | `0x142f02c94` | — |
| `i38` | `0x143d0c3e8` | `0x142f02d28` | — |
| `i39` | `0x143d0c438` | `0x14107bb68` | `0x142f05ae0` (pas une fonction dans Ghidra) |
| `i40` | `0x143d0c2a0` | `0x141fd7bc0` | — |
| `ti=0/2 i15` | `0x143d08ca0` | `0x1407ee7b8` | `0x142edad74` |
| `ti=0/2 i16` (neuf) | `0x143d08c50` | `0x1410d9004` | `0x142edd05c` |
| `ti=0/2 i17` (neuf) | `0x143d08bb0` | `0x141101038` | `0x142edbef4` (pas une fonction) |

Chaine des deux neufs : chaine `0x143c949f8` `scenario-intro-component` -> xref DATA
`0x1411756f0` (accesseur) -> xref DATA `0x143d08c68` (= descripteur + 0x18) ; chaine `0x143c94a18`
`matchflow-isplaying-flags-component` -> `0x1411756e0` -> `0x143d08bc8`. Calibration : la meme
chaine sur `managed-engine-timers` (`0x143c94990` -> `0x141175700` -> `0x143d08cb8`) rend le
descripteur `0x143d08ca0` deja connu.

### 1.1 Le contrat de la boucle de composants, relu : pourquoi `+0x40`, et ce que le niveau fait

`FUN_14076cb60` (decompile) appelle, par composant present au masque :

```
uVar10 = (**(code **)*plVar4)(plVar4);                      // vtable[0x00] : niveau du processus
if (FUN_1404f2b4c() && DAT_144c232e1 == 0) {                 // en rejeu de film
  uVar12 = (**(code **)(*plVar4 + 8))(plVar4);               // vtable[0x08] : nom
  uVar10 = FUN_1428e1b50(&DAT_144c23178, ti, uVar12);        // niveau DECLARE PAR LE FILM
  if ((**(code **)(*plVar4 + 0x10))(plVar4, uVar10)) goto LAB_14076cc46;   // ecarter ?
}
... cVar9 = (**(code **)(*plVar4 + 0x28))(plVar4, flux, param_2, &local_158, uVar10);
... if (cVar8 && FUN_1406cf008(flux)) { R(32) ; != 0xbcddcba -> "entity component corrupt" }
... if (!bVar7) { ... return false; }                        // un lecteur qui rend 0 tue le record
```

La vtable est `descripteur + 0x10` : `vtable[0x00]` = `+0x10` (niveau : `FUN_14117b4a0` =
`return 1`, ou `0x140c85020` = `return 4` pour `scenario-intro`), `vtable[0x08]` = `+0x18` (nom),
`vtable[0x10]` = `+0x20` = `0x1404ab600` = **`return false`** pour TOUS ces descripteurs (relu ici
pour `i15`, `i16`, `i17` ; notes C et D pour les `device-*`), `vtable[0x28]` = `+0x38` =
`FUN_14076ce9c` (thunk : `(**(code **)(*param_1 + 0x30))()`), `vtable[0x30]` = `+0x40` = le
lecteur. La regle statique du plan (« `vtable[0x28]`, ou `[0x30]` si c'est le thunk ») est donc
exactement « `descripteur + 0x40` ».

Deux consequences pour le port :

1. **Aucun de ces composants n'est jamais ecarte en rejeu** (`vtable[0x10]` rend faux) : s'il est
   au masque, ses bits sont la. Le niveau du film est passe au lecteur ; les lecteurs `device-*`,
   `i15`, `i16`, `i17` ont trois parametres et ne le lisent pas (desassemblages : aucun usage de
   `R9` avant ecrasement, `0x1410d9004`, `0x141101038`).
2. **Un lecteur qui rend 0 tue le record** (`return false` apres remise a zero). C'est le cas de
   `i31` quand son compte depasse 8 (note C §3) : le port doit declarer une desynchronisation,
   jamais borner.

Le controle de corruption par composant (`cVar8 = FUN_14076cea8()`, `R(1)` puis `R(32)` attendu
`0xbcddcba`) est deja porte (`consumeCorruptionCheck`, `traverse.go:166`) et s'applique apres chaque
composant porte : les vecteurs ci-dessous sont ecrits SANS lui (`ControleDeCorruption = false`) ;
avec lui, ajouter un `0` apres chaque composant.

---

## 2. Contre-epreuve cote ECRIVAIN des composants a poids de fermeture

Les notes du 17/09 ont lu les LECTEURS. L'ecrivain (`+0x28`) dit en plus QUAND un champ
conditionnel est ecrit, donc ce qu'un vecteur de test doit contenir. Primitives d'ecriture :
`FUN_1406d49c4(flux, _, bit)` = ecrit 1 bit (`+0x2c += 1`, `acc = acc*2 | bit`) ;
`FUN_1406d22c0(flux, _, v, min, max, n, b6, b7)` = quantifie et ecrit `n` bits ; `FUN_142af29b0` =
ecrit 2 bits ; `FUN_142ecf8e0` = ecrit 8 bits ; `FUN_1406d60f4(flux, _, src, n)` = copie brute de
`n` bits ; `FUN_140475500(v, lo, hi)` = bornage, 0 bit. L'accumulateur est pousse par la droite
et vide octet par octet en tete (`BSWAP`) : **le premier bit ecrit est le premier lu, chaque champ
poids fort en tete**.

### 2.1 Le quantificateur `FUN_1406d22c0`, branche `b7 = 1` (decompile + desassemblage)

```
1406d234c UCOMISS XMM2,XMM3 ; JZ 1406d231d      // v == min  -> code 0
1406d2359 UCOMISS XMM2,XMM1 ; JZ 1406d23d5      // v == max  -> code N-1 (1406d23d5 LEA R8D,[RCX-1])
1406d2360 ADD ECX,-0x2                          // N - 2
1406d237a CVTTSS2SI R8D,XMM2 ; 1406d237f INC R8D // trunc((v - min) / ((max - min)/(N-2))) + 1
1406d2382 CMP R8D,R11D ; CMOVLE R8D,R11D        // borne basse 1
1406d2389 CMP R8D,ECX ; ... CMOVG                // borne haute N-2
```

**Le code 0 n'est rendu QUE pour `v == min` exactement** ; toute valeur strictement interieure
rend un code de `1` a `N - 2`. C'est l'inverse exact du dequantificateur `FUN_1406d84b4` (notes A
§1, C §1).

### 2.2 `i35 device-animation-layer-state` — ecrivain `FUN_142f0570c`, entree `FUN_1432070d0`

```c
// FUN_142f0570c (ecrivain du composant)
cVar2 = FUN_142f0166c(obj + 0x5e8);          // A = « une couche d'i34 a un identifiant != -1 »
FUN_1406d49c4(flux /*, A */);
if (cVar2) {
  piVar6 = obj + 0x5fc; lVar4 = 0x6a8;        // 0x5fc = 0x5e8 + 0x14 : identifiant de la couche 0 d'i34
  for (8 fois) {
    iVar7 = *piVar6;
    FUN_1406d49c4(flux /*, iVar7 != -1 */);   // G_e
    if (iVar7 != -1) FUN_1432070d0(obj + lVar4, flux);
    lVar4 += 8; piVar6 += 6;                  // pas 0x18 : une couche d'i34 par entree d'i35
  }
}
// FUN_142f0166c(p) : parcourt p .. p+0xc0 par pas de 0x18, rend vrai des que *(p+0x14) != -1
```

```
// FUN_1432070d0 (une entree), desassemblage
1432070da MOVSS XMM0,[RCX+0x4]          ; w (poids)
1432070f8 CALL 0x140475500              ; bornage [0, 1]
14320710d MOV [RSP+0x28],0xa            ; n = 10, min 0 (XORPS XMM3), max 1,0 (XMM6 = [0x143cd8374])
1432070fd MOV [RSP+0x38],0x1            ; b7 = 1
14320711e CALL 0x1406d22c0              ; Q(10; 0..1) de w
143207123 MOVSS XMM4,[RBX+0x4]
143207128 COMISS XMM4,[0x143cd8370]     ; w BRUT (non borne) compare a 0,0
14320712f JBE 0x143207166               ; w <= 0 : rien de plus
143207150 MOV [RSP+0x28],0xe            ; sinon Q(14; 0..1) de la valeur
143207161 CALL 0x1406d22c0
```

Lecteur (note C §7, re-decompile ce jour, identique) : `A = R(1)` ; si `A`, 8 x
{ `G = R(1)` ; si `G` : `w = Q(10; 0..1)` ; si `w > 0` : `Q(14; 0..1)` }.

**Ce que l'ecrivain ajoute.** (a) La condition du dernier champ est testee par l'ecrivain sur le
`w` BRUT, par le lecteur sur le `w` DEQUANTIFIE ; grace au §2.1 les deux coincident : `w` brut
`<= 0` est borne a `0` puis code `0`, relu `0,0`, pas de valeur ; `w` brut `> 0` (meme `1e-6`)
rend un code `>= 1`, relu `> 0`, valeur ecrite ET lue. **La condition du port est donc
`code10 != 0`, sur l'entier** (la note C l'avait deduite du lecteur seul ; l'ecrivain la
confirme). (b) Les portes `A` et `G_e` d'`i35` ne sont pas des donnees libres : ce sont les
identifiants de couche d'`i34` (`obj + 0x5fc + 0x18 e != -1`). Une entree `e` d'`i35` ouverte
suppose la couche `e` d'`i34` ouverte dans l'etat du jeu — utile comme oracle de coherence le
jour ou les deux sont lus, jamais comme largeur (les deux composants n'ont pas a figurer dans le
meme record).

### 2.3 `i34 device-animation-layer-settings` — ecrivain `FUN_142f056a4`, couche `FUN_143206f90`

```c
cVar2 = FUN_142f0166c(obj + 0x5e8); FUN_1406d49c4(flux /*, A */);
if (cVar2) for (p = obj+0x5e8; p != obj+0x6a8; p += 0x18) FUN_143206f90(p, flux);
// FUN_143206f90 : FUN_143206d7c(_, flux, id = p[5])  = bit (id != -1) puis, si oui, FUN_141d12268 (32 bits)
//                 puis 4 x FUN_1406d22c0 et FUN_142af29b0 (2 bits)
```

Largeurs relues au desassemblage de `FUN_143206f90` : `143206fe7 MOV [RSP+0x28],0xe` (bornes
`[0x143cd84ec] = -1,0` / `[0x143cd8934] = 2,0`), `143207025 ... 0xe` (`0` / `[0x143cd84a8] = 100,0`),
`143207063 ... 0xa` (`0` / `[0x143cd8374] = 1,0`), `143207099 ... 0xe` (`0` / `100,0`),
`1432070b6 CALL 0x142af29b0`. **Concorde au bit pres avec le lecteur `FUN_143206e48`** (note C §6) :
`G` + `R(32)` + `Q(14)` + `Q(14)` + `Q(10)` + `Q(14)` + `R(2)` = 87 bits par couche ouverte.

### 2.4 `i21 device-position-group` — ecrivain `FUN_142f05cd0`

```c
FUN_1406d60f4(flux, flux, obj + 0x568, 0x20);           // 32 bits bruts
FUN_142ecf8a0(flux, *(ushort *)(obj + 0x56c));          // bit (v != 0xffff) puis, si oui, FUN_142ecf8e0 = 8 bits
```

Concorde avec le lecteur `FUN_1407f0678` (`R(32)` + `R(1)` + [`R(8)`]) : 33 ou 41 bits.

### 2.5 `i19`, `i22`, `i23` — ecrivains `FUN_142f05c14`, `FUN_142f05d50`, `FUN_142f05d90`

`i19` : `FUN_141d12268(flux, flux, obj + 0x540)` (32 bits) puis `FUN_1406d22c0` (le `Q(10; 0..10)`
du lecteur). `i22` / `i23` : `FUN_1406d22c0(flux, flux, *(obj + 0x570 | 0x574), 0, 1,0, 0xe, 0, 1)`
= `Q(14; 0..1)`. Concordent avec les lecteurs (note A §2, §5, §6).

### 2.6 `ti=0/2 i15 managed-engine-timers` — lecteur `FUN_1407ee7b8`, ecrivain `FUN_142edad74`

Re-decompile ce jour (la note 3.7 l'avait lu a `objdump`) :

```c
// lecteur
uVar2 = FUN_1406d6bac(flux, 0x40);                      // masque R(64)
for (k = 0; k < 64; k++) { if (uVar2 & (1 << k)) FUN_1407ee87c(flux /* ctx : n = 16, fente */); ... }
*(obj + 0x588) = uVar2;
// FUN_1407ee87c : +0x2c += 2 (R(2) etiquette) ; 0 -> rien ; 1 -> FUN_142ba78dc(.., 16, 3600,0) ;
//                 2 ou 3 -> FUN_1424cd048 = FUN_140d580d0(.., 16, 36000,0)
// ecrivain
FUN_1406d6498(flux, *(obj + 0x588), 0x40);              // le masque
for (k = 0; k < 64; k++) if (masque >> k & 1) FUN_142ecf980(flux, obj + 0x590 + 0x14 k, {flux, 0x10});
// FUN_142ecf980 : ecrit l'octet d'etat de la fente sur 2 bits ; 1 -> FUN_142ba7c74(.., 3600,0) ; sinon (!= 0) FUN_142b6f75c
```

Grammaire inchangee (note 3.7 §2.1) : `R(64)` puis, par fente presente, `R(2)` et
`0 bit | R(16)+R(16)+R(5)+R(16) | R(16)+R(16)+R(5)`.

### 2.7 NEUF : `ti=0/2 i16 scenario-intro` et `i17 matchflow-isplaying-flags`

```
// i16, lecteur FUN_1410d9004 (desassemblage complet)
1410d9018 CALL 0x1410d9088            ; FUN_1410d9088 : +0x2c += 7 (deux chemins), rend (7 bits) - 1
1410d9020 MOV [RDI + 0xa90],EAX
1410d9026 CALL 0x1406cf008            ; R(1)
1410d9030 MOV [RDI + 0xa94],AL
1410d9036 MOV AL,0x1
// ecrivain FUN_142edd05c : FUN_142ed15a0(flux, flux, *(obj + 0xa90)) puis FUN_1406d49c4 (1 bit)
// i17, lecteur FUN_141101038 : +0x2c += 8 (deux chemins, 14110105e ADD [RDX+0x2c],0x8), octet -> obj + 0xa95
```

| Composant | Grammaire | Largeur | Niveau (registre) |
|---|---|---|---|
| `i16 scenario-intro-component` | `R(7)` (valeur - 1) + `R(1)` | **8, fixe** | 4 (ignore par le lecteur ; `vtable[0x00]` = `0x140c85020` = `return 4`) |
| `i17 matchflow-isplaying-flags-component` | `R(8)` | **8, fixe** | 1 |

Avec `i11` (`R(128)`), `i13` (`R(13)` + n x `R(1)`), `i14` (niveau 2 : `R(1)` + `R(16)` + 4 x porte
INVERSEE [`R(7)`] + 4 x porte [`R(16)`]) de la note 3.7 §2 bis, **l'archetype du moteur est
entierement releve (18/18)** : ses records d'image-cle ont enfin l'oracle de fermeture que la note
3.7 §9.5 point 3 reclamait.

---

## 3. Les registres des films : les blocs `ti=43` et `ti=0/2` sont les memes sur tous les builds

Le registre d'un film est dans `chunk_00` : des entrees de `0x104` octets (nom ASCII, niveau u32 a
`+0x100`), lues en place (`grep -a -b -o`, `dd`, `od -tu4`). Le bloc `ti=43` commence au meme
decalage dans tous les films lus (`device-position-component` a l'octet `720208`).

| Film | Build | Bloc `ti=43` i0..i40 (empreinte md5 des 42 lignes nom+niveau) | Ecart |
|---|---|---|---|
| `bcb6d393` | HI_1_12_0 | `a5d95bfd` | reference |
| `0797ce72` | HI_1_13_0 | `a5d95bfd` | aucun |
| `111fa685` | HI_1_10_0 | `a5d95bfd` | aucun |
| `60ae07c4` | HI_1_8_0 | `99f6901d` | **i2 seul** : `object-forward-and-up-component` L1 au lieu de `...-dynamic-precision-component` L2 |
| `a521164d` | HI_1_4_1 | `3354da75` | i2 idem ; **pas d'i40** |
| `50247b26` | version-31 | `3354da75` | idem `a521164d` |

`i18` a `i39` : memes noms, meme ordre, **niveau 1 partout** ; `i40` niveau 1 la ou il existe.
`4f77afc1`, `81c02726`, `bfecd02b` (HI_1_13_0) portent les memes noms aux memes decalages.

Blocs `ti=0` (octet 3908) et `ti=2` (octet 37188), 18 entrees chacun : **empreinte identique
`6f0cacc3` sur `bcb6d393`, `60ae07c4`, `a521164d`, `a349fea8`, `50247b26`** — `i14` niveau 2, `i16`
niveau 4, tous les autres niveau 1.

Ce que cela etablit, et ce que cela n'etablit pas :

- Le niveau est la version de format que la boucle du jeu confronte au processus (§1.1). Des
  niveaux identiques disent que le FILM declare le meme format de composant sur tous ces builds.
  La seule faille possible est un changement de format sans montee de niveau ; elle se mesure
  (§6), elle ne se lit pas dans l'image HI_1_13_0. Validite inter-build : **probable**.
- **Refutation d'une hypothese de la note 3.7** (§9.5 point 2 : « la cause probable [de l'echec
  du bassin sur les deux BTB Heavies] est le NIVEAU de `i14` (la branche `level < 2` lit `R(64)` de
  plus) »). `a349fea8` et `a521164d` declarent `i14` au niveau 2, comme les films d'arene ou la
  lecture tient. La cause de leurs masques « tout a un » est ailleurs (composants lus avant `i11`
  dans l'image-cle, ou desalignement anterieur — ces builds ferment 0,0 a 0,6 % des records utiles).

---

## 4. Le lien avec les rejets de la vue B (hypothese de la campagne)

Code relu (`frame_infer.go`) :

1. `corpsDeRecordNeuf` : un NEW dont la traversee desynchronise (composant sans `case`) rend
   `desync = true`, la marche cale (`stall`, `inferResyncTargets` nil) et **l'entite n'est PAS
   liee** (ni `BindFull`, ni `BindSoft`).
2. `rejetDeVue` : un DELTA dont le slot n'est pas lie est **rejete hors datum** (`hitEnd = true`,
   la vue B « se termine »), SAUF si la table anticipee (installee par `frame_closure.go:186` et
   `movement_states.go:199`) trouve cet eid declare par une image-cle ULTERIEURE
   (`LierParRepliDAnticipation`).

Donc un dispositif NE (record NEW) entre deux images-cles et dont le NEW butte sur un `device-*`
non porte laisse son slot non lie ; ses deltas suivants, dans le meme intervalle :

- sont lies par anticipation s'il survit jusqu'a l'image-cle suivante — puis s'arretent sur le
  premier `device-*` non porte (arret NOMME, deja compte dans la carte) ;
- sont **rejetes hors datum** s'il disparait avant l'image-cle suivante, ce qui sort la vue B et
  fait lire la vue C au mauvais endroit : c'est EXACTEMENT le mecanisme de l'hypothese de travail
  de la campagne, avec une source `ti=43` nommable.

Le cas temoin connu est `81c02726` (G MONEY, tick 2941, NEW `ti=43` qui desynchronise sur `i21`,
`PLAN_RETOURS_REJEU_2026-09-23.md` l. 919) : la perte constatee est celle du paquet du NEW lui-meme
et des suivants. L'ampleur sur les films denses HI_1_13_0 est INCONNUE et vraisemblablement faible :
`ti=43` n'apparait pas parmi les archetypes les plus lies par anticipation sur `bfecd02b`
(`NOTE_5_23_TABLE_ANTICIPEE_2026-09-22.md` l. 116 : `ti=42, 40, 10, 41, 32, 37, 35`). Statut :
**hypothese**, mesure au §6.3.

---

## 5. Ecart Go / jeu et correctif propose (phase 2)

| Composant | Jeu | Go | Correctif |
|---|---|---|---|
| `ti=43 i19..i40` (22) | lecteurs releves (notes A-D) + ecrivains §2 | aucun `case` : `DesyncAt` | porter les 22 d'un coup |
| `ti=0/2 i11, i13, i14, i15` | note 3.7 §2, §2 bis ; §2.6 ici | lecteurs en `research` seulement | promouvoir en production |
| `ti=0/2 i16, i17` | §2.7 | aucun `case` | `R(7)+R(1)` ; `R(8)` |

Recette (aucune primitive nouvelle) :

- Nouveau fichier `components_device.go` (seuil 500 lignes) et un maillon de dispatch par nom ;
  chaque lecteur avec son adresse.
- `i19` : `R(32)` + `R(10)`. `i20` : `R(256)` par tranches de 64 (`FUN_1406d676c`). `i21` :
  `R(32)` + `consumeGateR(br, 8)`. `i22`, `i23` : `R(14)`. `i24`, `i29` : `consume1408f0ac4(br, 1)`.
  `i25`, `i40` : `R(8)`. `i26` : `R(32)` + `R(32)`. `i27` : `R(1)` + `R(6)`. `i28`, `i30`, `i33` :
  `R(1)`. `i31` : `N = R(8)` ; `N > 8` -> DESYNCHRONISATION declaree (le lecteur du jeu rend 0 et la
  boucle tue le record, §1.1) ; sinon N x `consume1408f0ac4(br, 1)`. `i32` : `R(5)`. `i34` : `A` ;
  si `A`, 8 x { `G` ; si `G` : `R(32)` + `R(14)` + `R(14)` + `R(10)` + `R(14)` + `R(2)` }. `i35` :
  `A` ; si `A`, 8 x { `G` ; si `G` : `q = R(10)` ; si `q != 0` : `R(14)` } — **condition sur
  l'entier** (§2.2). `i36` : 2 x { `G` ; si `G` : `consume1408f0ac4(br, 0)` (categorie 0 : PAS de
  bit de sonde) ; `R(3)` inconditionnel }. `i37` : 2 x { `G` ; si `G` : `R(10)` + `R(10)` + `R(5)`
  + `R(10)` }. `i38` : `R(18)`. `i39` : `R(9)`.
- `ti=0/2` : `i11` `R(128)` ; `i13` `n = R(13)` puis `n` x `R(1)` ; `i14` au niveau 2 (la branche
  `level < 2` n'est pas a porter tant qu'aucun registre ne la declare, §3) ; `i15` `R(64)` + fentes ;
  `i16` `R(8)` ; `i17` `R(8)`.
- **Regle des deux copies** : `FUN_140d580d0` (`R(n)+R(n)+R(5)`) a deja TROIS copies en production
  (`consumePlayerSoftKillTimer` n = 5, `decodeGameEngineRoundTimer` n = 16,
  `consumeGameEngineCampaignTimer` = `Skip(37)`) ; `i15` et `i37` en ajoutent deux, plus
  `FUN_142ba78dc` (`... + R(n)`). Centraliser `consume140d580d0(br, n)` /
  `consume142ba78dc(br, n)` et poser le garde-rail dans le meme lot.
- `ecs_table.tsv` (28 lignes : `status`, `deser_addr`, `grammar`, `bits_typ`, `code_source`) dans
  le commit du code ; comptes geles de G4 (`ecs_widths_guard_test.go`) a deplacer : `i31` sur
  motif `0xFF` (N = 255) rend un echec, a classer.
- Le port change une sortie de production (le flux est lu plus loin) : `grammar.Rev` monte, gate de
  corpus et references d'equivalence contre la tete post-J12 (plan, §0).

---

## 6. Vecteurs de test (construits d'apres l'ECRIVAIN) et mesures

Convention : bits dans l'ordre d'ecriture, chaque champ poids fort en tete, `|` = separateur de
lecture ; `ControleDeCorruption = false` ; `W = varWidthBits(cat)` = 13 pour les categories 0 et 1
(`varwidth.go`, plage `0x1DFF`), 9 apres une sonde a 1. Chaque vecteur se teste par « le lecteur
consomme exactement N bits et rend les champs attendus ».

### 6.1 `ti=43`

| Id | Etat ecrit (ecrivain) | Bits | N |
|---|---|---|---:|
| V35a | aucune couche d'i34 (A = 0) | `0` | 1 |
| V35b | couches 0 et 1 ouvertes ; e0 : w = 0,0 (valeur 0,7 non ecrite) ; e1 : w = 1,0, valeur 1,0 ; e2..e7 fermees | `1` \| `1 0000000000` \| `1 1111111111 11111111111111` \| `0 0 0 0 0 0` | 43 |
| V35c | e0 seule : w = 1e-6 (code 1, §2.1), valeur 0,0 | `1` \| `1 0000000001 00000000000000` \| `0`x7 | 33 |
| V34a | A = 0 | `0` | 1 |
| V34b | couche 0 : id `0x00000001`, f0 = 2,0 (max de [-1, 2]), f1 = 0,0, f2 = 1,0, f3 = 100,0, drapeaux `01` ; couches 1..7 fermees | `1` \| `1 00000000000000000000000000000001 11111111111111 00000000000000 1111111111 11111111111111 01` \| `0`x7 | 95 |
| V21a | mot `0x00000007`, groupe 3 | `00000000000000000000000000000111 1 00000011` | 41 |
| V21b | mot `0x00000007`, groupe `0xffff` | `00000000000000000000000000000111 0` | 33 |
| V19 | id `0xDEADBEEF`, position 10,0 (code 1023) | `11011110101011011011111011101111 1111111111` | 42 |
| V22 | puissance 1,0 / 0,0 | `11111111111111` / `00000000000000` | 14 |
| V24a / V29a | aucune entite | `0` | 1 |
| V24b | P = 1, S = 0, index 5, generation 2 | `1 0 0000000000101 10` | 17 |
| V24c | P = 1, S = 1, index 5 (plage 9 bits), generation 2 | `1 1 000000101 10` | 13 |
| V31a | N = 2 : moniteur 0 absent ; moniteur 1 index 42, gen 1 | `00000010` \| `0` \| `1 0 0000000101010 01` | 26 |
| V31b | N = 9 (flux invalide) | `00001001` -> DESYNCHRONISATION declaree apres 8 bits | 8 |
| V36 | entree 0 : G = 0, etat 5 ; entree 1 : G = 1, P = 1, index `0x42`, gen 1, etat 2 | `0 101` \| `1 1 0000001000010 01 010` | 24 |
| V37 | entree 0 : G = 0 ; entree 1 : 600,0, 0,0, octet 3, 600,0 | `0` \| `1 1111111111 0000000000 00011 1111111111` | 37 |
| V39 | drapeaux `0x1A5` | `110100101` | 9 |

Vecteurs DISCRIMINANTS (une erreur de port plausible les fait echouer) : V35b (un port qui ignore
la condition lit 57 bits), V35c (un port qui traite le code 1 comme un poids nul ne lit pas les
14 bits de valeur que l'ecrivain a ecrits), V36 (un copier-coller du port d'`i24` lit un bit de
sonde de trop par entree ouverte, omet la porte externe et la queue `R(3)`), V31b (un port qui
borne N a 8 continue), V37 (l'octet AVANT la troisieme valeur).

Somme minimale des 22 (`i19..i40`, toutes portes fermees, `i21` sans groupe) : **503 bits**.

### 6.2 `ti=0/2`

| Id | Etat ecrit | Bits | N |
|---|---|---|---:|
| V15a | fentes 0 et 1 allouees ; fente 0 etiquette 2 : 36000,0 (code 65535), 0,0, queue 0 ; fente 1 etiquette 0 | `0`x62 `11` \| `10 1111111111111111 0000000000000000 00000` \| `00` | 105 |
| V15b | fente 0 seule, etiquette 1 : 3600,0, 0,0, queue 1, 0,0 | `0`x63 `1` \| `01 1111111111111111 0000000000000000 00001 0000000000000000` | 119 |
| V16 | scenario 0 (ecrit 1), drapeau 1 | `0000001 1` | 8 |
| V17 | drapeaux `0x05` | `00000101` | 8 |

### 6.3 Mesures qui prouveront l'effet (carte de fermeture v2)

1. **Port `ti=43`, quatre films un par un** (`bcb6d393` HI_1_12_0, `60ae07c4` HI_1_8_0, `81c02726`
   HI_1_13_0, la bobine `ks_000d5950`), `cmd_fermeture`, avant / apres :
   - les lignes `ti=43 i35`, `i21`, `i39`, `i19` disparaissent des causes ; aucune nouvelle cause
     `ti=43` n'apparait (sinon une largeur est fausse : l'oracle est le paquet suivant) ;
   - `bcb6d393` : vue C atteinte de 9 156 vers ~19 600 paquets (≥ 10 830 relaches) ; si le taux
     de fermeture de la vue C du film (63,7 %) tient, environ **+6 900 paquets fermes** (26,7 % ->
     ~58 %) ; records utiles : borne +67 207 sur 117 732 (27,2 % -> au plus ~84 %), le gain reel
     etant ce que la vue C laisse fermer ;
   - nouveaux paquets « vue C : terminateur hors cadre » : ils diront si le residu de HI_1_12_0
     est de la meme nature que celui de HI_1_13_0 ;
   - `81c02726` : le NEW du tick 2941 traverse proprement ; la 3e montee de G MONEY (2984-3021,
     3104-3119) est lue, `noTrack` 2 -> 0 (a verifier au document, pas au compteur seul).
2. **Port `ti=0/2`** : la ligne `ti=2 i15` (3 635) disparait ; les records d'image-cle du moteur
   (113 sur les sept mini-bobines, note 3.7 §6) ferment sur les builds d'arene ; sur `a349fea8` /
   `a521164d`, s'ils ne ferment pas, le §3 dit que le niveau d'`i14` n'y est pour rien.
   ATTENTION a la borne de gain : « records lus » ne compte pas ce qui suit l'arret dans le paquet ;
   le gain de 27 de la carte est un minimum. Lire la colonne « reste du payload » de la carte v2
   (item 1.1 du plan) sur les paquets arretes par `i15` pour en avoir l'ordre de grandeur.
3. **Contribution indirecte aux rejets (§4)** : dans la ventilation 1.2 de la carte v2, pour chaque
   rejet hors datum qui termine la vue B, chercher si un NEW du MEME slot a desynchronise plus tot
   dans le chunk ; si oui, compter `naissance lue non traversee (ti, composant)`. Le compte
   `ti=43` est la part du « hors cadre » que ce port retire ; il vaut 0 si l'hypothese ne tient pas.

---

## 7. Bilan

- Les 22 grammaires `device-*` et les 6 du moteur sont COMPLETES et ne dependent d'aucune entree
  hors du flux sauf la plage de reference des categories 0 et 1 (deja portee, `varwidth.go`). Les
  composants a poids de fermeture (`i19`, `i21`, `i22`, `i23`, `i34`, `i35`, `i15`) sont confirmes
  des DEUX cotes, lecteur et ecrivain.
- Ce port est le **levier n° 1 de HI_1_12_0** (`i35` arrete la moitie des paquets de `bcb6d393`)
  et repare le cas `81c02726`. Il n'est PAS le levier de la cause n° 1 de la campagne (« terminateur
  hors cadre », 214 536 paquets) : sur HI_1_13_0, les arrets `ti=43` / `ti=2` nommes pesent quelques
  centaines de paquets ; la part indirecte (§4) est a mesurer et vraisemblablement faible.
- Ordre de grandeur, toutes builds : ~16 600 paquets arretes par une cause de cette piste
  (≈ 6 % des 279 167 paquets non fermes de la carte du 26/09), borne de gain ~78 000 records
  utiles contre 2 171 370 pour le « hors cadre ».

## 8. Appels Ghidra

Environ 45 appels HTTP en lecture, aucune ecriture : `read_memory` x 35 (23 + 9 pointeurs, 3
descripteurs de 80 octets), `decompile_function` x 22, `disassemble_function` x 5,
`search_strings` x 4, `get_xrefs_to` x 6. Trois adresses d'ecrivain ne sont pas des fonctions
definies dans Ghidra (`0x142f05bf8`, `0x142f05ae0`, `0x142edbef4`) : le lecteur suffit pour la
largeur. Un appel `/mcp/schema` a expire (sans effet).
