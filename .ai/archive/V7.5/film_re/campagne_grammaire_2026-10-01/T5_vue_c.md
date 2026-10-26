# T5 — Vue C (contrôle) : kinds 1 et 2, bloc 0xbc, terminateur (campagne grammaire, phase 1, 2026-10-01)

> Piste T5 du plan `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (étape 2). Lecture seule : Ghidra
> (`HaloInfinite.exe`, serveur HTTP `127.0.0.1:8089`, aucune écriture), code Go à la tête du
> worktree `feat/campagne-grammaire` (base `69564ef7d`). Aucune commande `go`, aucune base, aucun
> film lu. Image base `0x140000000`. Bits lus poids fort d'abord (MSB), comme `Lecteur`.

## 0. Réponses courtes

| question | réponse | confiance |
|---|---|---|
| Le film peut-il porter des kinds 1 ou 2 dans sa vue C ? | **Non.** L'enregistreur du film (`FUN_142f2c3b0`) compose la vue C en concaténant les 32 tampons PAR JOUEUR `*(DAT_145178b58 + 0x480 + k*0x4c8)`, k = 0..31, puis écrit un `0`. L'écrivain d'entrée (`FUN_14076b0e8`) ne met dans ces tampons que des entrées **kind 0** ; les kinds 1 et 2 vont dans les écrivains de la VUE (`vue+0x6088`, `vue+0x5b30`), que seul le chemin réseau vide (`FUN_14076b944`). Le kind 3 n'est écrit par personne. | établi (jeu lu) |
| Ordre des entrées dans le film | une entrée au plus par joueur, **index de contrôle strictement croissant** (boucle k = 0..31 de `FUN_142f2c3b0`, tampon choisi par l'index écrit) | établi (jeu lu) |
| Conséquence pour les 4 849 paquets « vue C : kind non porté » | ce ne sont PAS des trous de grammaire : un kind 1/2/3 lu dans un film prouve une lecture de vue C au mauvais bit (ou une entrée précédente mal lue). Porter les kinds 1/2 dans le décodeur de film n'en fermerait aucun. | établi pour le binaire lu ; à confirmer par la mesure sur les builds anciens |
| Grammaire des kinds 1 et 2 | écrite au §3 et §4 (lecteur ET écrivain lus). Elle ne dépend d'aucun état d'exécution pour ses LARGEURS : la seule bifurcation est la présence de l'en-tête `FUN_1406cdc04`. Le commentaire Go « la charge dépend d'un état d'exécution (`FUN_142f2b574`) » est faux : `FUN_142f2b574` ne garde que le chemin DIFFÉRÉ (zéro bit lu). | établi |
| Grammaire du bloc secondaire 0xbc (`FUN_141fdae44`) | écrite au §5 (lecteur et écrivain `FUN_1406d1ba4` lus). Une seule fourche non résolue : la position de `+0x74`, `R(96)` brut OU enveloppe `FUN_14076e420(0x10)`, gouvernée par `FUN_1404f293c` (rôle réseau), de sens OPPOSÉ chez l'écrivain et le lecteur ; la fermeture arbitre. | établi (largeurs), fourche = hypothèse |
| La forme du terminateur que nous lisons est-elle celle du jeu ? | **Oui, exactement** : `R(1) == 0` dans la boucle de `FUN_1406cf548` ; c'est le MÊME bit que le « encore » qui précède chaque entrée (`W(1)=1` chez l'écrivain d'entrée), et l'enregistreur écrit `W(1)=0` après le dernier tampon. Rien n'est lu après la vue C (`FUN_142987460`), et le paquet n'a que 0 à 7 bits de bourrage nuls (T8). Différences : le lecteur du jeu sort AUSSI sans terminateur sur ses codes d'erreur (budget, `a = b = 0`, validateurs, kind 1/2 sans charge), et `FUN_142987460` ignore ces codes ; aucun n'est produit par l'écrivain du film sauf `a = b = 0` (§6). | établi |
| La grammaire du kind 0 que porte le Go est-elle celle de l'écrivain ? | **Oui sur toutes les portes et largeurs relues** : `FUN_14076b0e8` (en-tête), `FUN_1406d143c` (bloc 0x68 : `W(1) g [W(2)]`, couple `W(6) W(6)`, `W(1) [W(5)]`, `W(1) [W(6)]`, `W(1) [W(5)]`, bloc d'action TOUJOURS). Le gros du « hors cadre » ne vient donc pas de la grammaire de la vue C lue au bon endroit. | établi |

## 1. Le lecteur de la vue C (`FUN_1406cf548`, vtable `0x1436a8770` slot `+0x40`)

```
FUN_1406cf548(vue, ctx, lecteur, budget, _, *rendus) :
   tier = FUN_140ce620c() ; c = FUN_1409c94b8()                        <- hors flux
   si c : FUN_142f2539c(lecteur, &copie, &taille)                     <- 0 bit (memcpy du tampon)
   boucle {
      b = R(1)                       ; si b == 0 -> *rendus = n ; return code courant (0)
      si budget <= n -> return 3     ; (budget = 0xa00 - records déjà rendus)
      kind = R(2)
      0 -> FUN_1406d0388   1 -> FUN_142f29b38   2 -> FUN_142f29e54   3 -> rien, 0 bit
      si code != 0 -> *rendus = n ; return code
   }
```

- `ctx` est `&uStack_78050` de `FUN_142987460`, mis à ZÉRO avant l'appel : `*ctx = 0` (le « tick »
  des kinds 1/2) et `*(byte*)(ctx+8) = 0` (les records sont rangés). Conséquence au §4.
- `FUN_142987460` appelle, pour chaque rang, `vtable[0x60]` (0 bit ; pour la vue C c'est
  `FUN_142f28e94`, le rejeu des entrées différées, qui lit une COPIE) puis `vtable[0x40]`, et
  **n'examine jamais le code rendu** : une vue qui sort sur erreur laisse le curseur où il est, et
  rien n'est lu après la vue C. Le seul contrôle de paquet est le débordement (`FUN_14298816c`,
  NOTE 5.15 §4, T8).
- Le chemin différé (`FUN_1409c94b8() && vue+0x6160 != 0`) : kind 0 lit quand même toute sa charge
  (seul l'enregistrement est supprimé) ; kinds 1 et 2 ne lisent PAS leur charge dans ce mode. Hors
  du film (question ouverte Q2).

## 2. L'écrivain de la vue C du film

### 2.1 L'enregistreur (`FUN_142f2c3b0`)

```
pour i in 0..2 : écrivain_i (0xd8 o) ; FUN_1406d5cc0(écrivain_i)
   i == 0 : FUN_142f2c050 (vue A) ; FUN_1406d49c4(w, 0)
   i == 1 : FUN_142f2cc78 (vue B)
   i == 2 : pour off in 0, 0x4c8, ..., < 0x9900 (k = 0..31) :
               t = *(DAT_145178b58 + 0x480 + off)
               si (t+0x20) - 1 < 2  et  ceil(FUN_14076b9b0(t) / 8) > 0 : FUN_1406d5d14(w, t)   <- copie de BITS
            FUN_1406d49c4(w, 0)                                                              <- LE TERMINATEUR
   FUN_1406d6d94(w)
FUN_1428e339c(&DAT_144c23178, ..., écrivains)                                                <- vers la session film
```

(T3 relit la même fonction pour la vue B ; ce paragraphe ne concerne que `i == 2`.)

### 2.2 L'écrivain d'entrée (`FUN_14076b0e8`, slot `+0x18` de la vtable de la vue C)

`param_2[0]` = index de contrôle `idx`, `param_2[1]` = kind.

- **kind 0** : `t = *(table + idx*0x4c8 + 0x480)` avec `table = DAT_145178b58` (ou `DAT_144de4b78`
  si `FUN_140769e58(...)`). Si l'état `t+0x20` n'est ni 1 ni 2 : remise à zéro du tampon, puis
  `W(1)=1` ; `FUN_1411b13fc(t)` avec `R8D = 0` (`14076b330: XOR R8D,R8D`) = `W(2)=0` ; `W(1)=0`
  (en-tête `FUN_1406cdc04` ABSENT) ; `W(5)=idx` ; si `FUN_14048ee34() == 0` :
  `FUN_140769f90` (`a`) puis `FUN_14076a064` (`b`) ; sinon `FUN_142f2c164`. État posé à 1 : **une
  entrée au plus par joueur**. (Une variante froide, `14230bf83`, écrit l'en-tête par
  `FUN_140769eb4(kind 0, octet de baseline)` : en-tête PRÉSENT, toujours kind 0.)
- **kind 1** : `FUN_140769eb4(vue+0x6088, desc, 1, décalage|0xff)` (`14230c20d: LEA RBX,[RSI+0x6088]`,
  `RCX = RBX` conservé par `FUN_14076b9b0`), `W(5)=idx`, `FUN_142f2c9d4`.
- **kind 2** : `FUN_140769eb4(vue+0x5b30, desc, 2, 0xff)` (`14230c4a2: MOV R9B,0xff`,
  `14230c4a5: MOV R8D,0x2`), puis `FUN_142f2beac`.
- **autre kind** : rien n'est écrit (`return NULL`).
- `FUN_14076b944` (slot `+0x20`) vide `vue+0x6088` puis `vue+0x5b30` dans le paquet RÉSEAU. Ces deux
  écrivains n'entrent pas dans `FUN_142f2c3b0`.

`FUN_140769eb4` (en-tête commun, appelée seulement depuis `FUN_14076b0e8` et sa partie froide) :
`W(1)=1 ; W(2)=kind ; W(1)=(octet != 0xff) ; si présent W(7)=octet`.

**Donc la vue C d'un film est : `( 1 00 h[7] iiiii a [bloc 0x68] b [bloc 0xbc] )*  0`, index
strictement croissants, au plus 32 entrées.** Le Go borne à 64 tours (`plafondToursVueC`) ; 32 est
la borne de l'écrivain.

### 2.3 Les blocs d'une entrée kind 0 ne sont écrits que VALIDES

```
FUN_140769f90 : a = (vue+0x2550 bit idx) && FUN_1406d33cc(vue+0x25f0+idx*0x68) ; W(1)=a ; si a : FUN_1406d143c
FUN_14076a064 : b = (vue+0x2558 bit idx) && FUN_1407699d0(vue+0x32f0+idx*0xbc) ; W(1)=b ; si b : FUN_1406d1ba4(w, bloc, 1, p5)
```

Rien n'empêche `a = b = 0` côté écrivain (les deux masques peuvent être nuls) ; le lecteur du jeu,
lui, rend 3 sur `a = b = 0` (`1406d059c: TEST DIL,DIL ; JZ 1422f5143` puis `1422f5143: TEST R13B,R13B`,
`EBX = 3`) et la vue C s'arrête dans le jeu — sans conséquence pour lui (rien après la vue C), et
sans conséquence pour le Go qui lit les 11 bits et continue.

### 2.4 Le bloc 0x68 chez l'écrivain (`FUN_1406d143c`) — contrôle de la grammaire portée

| écrivain | largeur | Go (`consumeEntreeControle`) |
|---|---|---|
| `FUN_1424caaac` : `W(1)=g` ; si `g == 1` : `W(2 \| 4)` (`DAT_145121140 == 1` -> 4) | 1 [+2] | `R(1) [R(2)]` |
| `FUN_1406d5310` : `W(6)` `W(6)` (`FUN_1406d5bf4`) ; `W(1)=(tiers != 0.0)` ; si 1 : `FUN_140dc61a8` = `W(5)` | 13 [+5] | idem |
| `W(1)=(+0x10 != 0.0)` ; si 1 : `FUN_142f222d0` = `W(6)` | 1 [+6] | idem |
| `W(1)=(+0x14 != 0)` ; si 1 : `W(5 \| 7)` | 1 [+5] | idem |
| `FUN_1406d15f8` : bloc d'action, TOUJOURS | var. | `lireBlocDAction` |

`FUN_1406d5bf4` (quantificateur scalaire) : `-1.0 -> 0`, `+1.0 -> 0x3e`, sinon borné à `[1, 0x3d]`.
**Le code 63 (`0x3f`) n'est jamais écrit** ; au lecteur il décode à `1,049 > 1` et `FUN_1406d33cc`
le refuse (`|x| <= DAT_143cd8374 = 1.0`).

## 3. Kind 2 (réseau seulement) — `FUN_142f29e54` / `FUN_142f2972c`, écrivain `FUN_142f2beac`

```
en-tête    FUN_1406cdc04 : R(1) p ; si p : R(7)          (l'écrivain pose p = 0)
charge     FUN_142f2972c :
              present = R(1)                              W(1) = vue+0x2564
              si present == 0 : le lecteur du jeu rend 3 (FUN_142f29e54 : cVar1 == 0 -> iVar4 = 3)
              sinon : alloc(type 0xb, 2 o) ; R(5) -> o[1] ; R(8) -> o[0]
                                                          W(5) = vue+0x2568 ; W(8) = vue+0x2567
```

Coût : `1 + 2 + 1 [+7] + 1 [+13]` bits. Application (`FUN_14076b838`, type `0xb` -> `FUN_142f28a18`) :
recopie des deux octets en `vue+0x2565` (un décalage de tick signé et sa clé). Chemin différé
(`param_6 && vue+0x6160`) : en-tête seul, charge NON lue.

## 4. Kind 1 (réseau seulement) — `FUN_142f29b38` / `FUN_142f29fd8` / `FUN_142c41354`, écrivain `FUN_142f2c9d4` / `FUN_142c41810`

```
en-tête    FUN_1406cdc04 : R(1) p ; si p : R(7) d
idx        R(5)                                           (> 0x1f -> 3, inatteignable)
base       B0 = FUN_142f2b9d8(FUN_142f2b818(vue+0x4fe8+(idx+0x21)*0x28, FUN_1406d0f0c(*ctx, d)))   <- 0 bit
              FUN_1406d0f0c(t, d) = d == 0xff ? -1 : t - d - 1 ;  t = *ctx = 0 dans le film
              -1 -> B0 = NULL ; clé absente de l'anneau -> objet par défaut DAT_144f572d0
              (FUN_142f28330 : drapeaux |= 3, élément +0x5c À ZÉRO) -> B0 = élément nul
charge     FUN_142f29fd8 :
              present = R(1)                              W(1) = (vue+0x2048 bit idx) && (n != 0)
              si present == 0 : le lecteur du jeu rend 3
              n = R(2)                                    W(2) = n
              alloc(type 9, n * 0x58) ; FUN_142c41354(lecteur, liste, B0)
élément k  (B = B0 pour k = 0, sinon l'élément k-1)
   id     B == NULL ? R(32) : ( R(1) ? ( R(1) ? B.id + R(6) : B.id + 1 ) : R(32) )      FUN_142bce148 / FUN_142bce2d0
   +0x50  B == NULL ? R(7)  : ( R(1) ? R(7) : B[0x14] )
   retiré R(1) ; si 1 : fin de l'élément
   R(1) ? { varint(cat 0) ; FUN_1407f1e4c = R(1) [si 0 : R(10)] } : {}
   R(1) ? varint(cat 4) : {}                                                          (+0x08)
   5 x vec3 : B == NULL ? R(96) brut : ( R(1) ? 3 x { k = R(5) ; R(32 - k) } : copie )    FUN_142c38ed8 / FUN_143300170
   FUN_140c50d1c : R(1) [si 1 : R(8)] ; si présent : R(4)
   FUN_142af28d8 : R(1)
   R(1) ? ( R(1) ? varint(cat 2) : -1 ) : B[0x15]        (B == NULL et bit 0 : déréférence nulle -> l'écrivain pose 1)
```

`varint(cat c)` = `FUN_1406d3140` = `readVarWidthInt(br, c)` déjà porté (`varwidth.go`) : cat 0 :
`R(13)+R(2)` ; cat 2 : `R(8)+R(2)` ; cat 4 : `R(9)+R(2)`. `FUN_143300170(base)` : `k = R(5)`, puis
`32 - k` bits bas, les `k` bits hauts recopiés de la base. **Aucune largeur ne dépend d'une valeur de
la base ; seule compte sa NULLITÉ**, décidée par le flux (`p`, et `d == 0` qui rend -1 quand
`*ctx = 0`). Le commentaire de `ArretVueCKindNonPorte` est donc inexact.

## 5. Bloc secondaire 0xbc — lecteur `FUN_141fdae44(lecteur, bloc, 1)`, écrivain `FUN_1406d1ba4(w, bloc, 1, p5)`

```
+0x00  flags  R(5)                       FUN_1424ccc74(…, bloc)        écrivain FUN_140cf35a4 W(5)
+0x01         R(2) [R(4) si DAT_145121140 == 1]
+0x04  lacet  R(17) sur [0 ; 2pi]        FUN_1406d84b4, [RSP+0x20] = 0x11, pas 2pi/2^17
+0x08  tangage R(16) sur [-pi ; pi]      FUN_1406d84b4, [RSP+0x20] = 0x10, pas 2pi/2^16
+0x10  R(1) ; si 1 : varint(cat 1)       (param_3 = 1 ; param_3 = 0 -> R(1) [R(32)] en +0x14)
si flags & 2 :
+0x18  R(1) ; si 1 : varint(cat 1)       FUN_142f2e63c(…, 1)
+0x20  R(2) - 1                          FUN_14080cb98                 écrivain FUN_1406d2b90
+0x24  R(3)                              FUN_1406d0f20                 écrivain FUN_1406d5b74
+0x28  c = R(3)                          FUN_1424d0f48                 écrivain FUN_142b1d01c
       c x { R(1) [varint(cat 0)] ; R(4) - 1 ; R(4) - 1 }   FUN_142f26754 (FUN_142f2e63c cat 0, FUN_142ed0674, FUN_14101d200)
+0x6c  R(5)                              FUN_1424ccc74
+0x6d  R(3) ; R(1) [si 0 : R(2)] ; R(1) [si 0 : R(2)]     FUN_1406d01fc  (= consumeWeaponStateTail sans sa porte de tête)
+0x70  R(6) ; R(3)                       FUN_140c6a638 (consumeBipedDesiredGrenadeSet)
+0x72  R(3) ; R(1) [si 0 : R(6)]         FUN_1406d0ff0 (consumeBipedDesiredAbilitySet)
+0x74  FOURCHE (voir ci-dessous) : R(96) brut   OU   lireE420(0x10) [0x1e si DAT == 1]
+0x80  R(1) [si 0 : R(19)] ; R(8)        FUN_140c1e79c (consumeCompressedDir140c1e79c)   [DAT == 1 : FUN_142e29bac]
+0x98  R(1) [si 0 : R(19) ; R(10)]       FUN_14076d528, dir 0x13, norme 0xa, borne 350
+0xa4  R(1) [si 0 : R(19) ; R(8)]        FUN_14076d528, dir 0x13, norme 0x8, borne 30
+0xb0  p = R(1) ; si p : varint(cat 0)   FUN_142f0ec48 -> FUN_1408f0ac4, R8D = 0 ; +0xb8 = !p
+0xb4  si p : R(11) signé                (141fdb1d8..141fdb29c)        écrivain 1422f58be : W(11) = +0xb4 & 0x7ff
+0xb9  R(7)                              FUN_142f21dd4                 écrivain FUN_140cf34b4
fin    return FUN_1407699d0(bloc)        (faux -> la vue C du jeu rend 3)
```

**La fourche de `+0x74`.** Lecteur (`141fdb0c8`) : `FUN_1404f293c()` VRAI -> `R(96)` ; FAUX et
`param_3` -> `FUN_14076e420(…, 0x10, 0)`. Écrivain (`1406d1df2`, `1422f5891`) : VRAI et `param_3`
-> `FUN_142e2f508` (enveloppe, porte `+0xb0 != -1`) ; FAUX -> `FUN_1406d60f4(…, 0x60)` brut. Les deux
sens sont OPPOSÉS : la forme écrite est la forme lue seulement si les deux processus ont des rôles
différents (`FUN_1404f293c` = `état != 2`, VRAI hors session). Réseau : serveur (VRAI) -> client
(FAUX) = enveloppe des deux côtés. Film : l'écrivain est le CLIENT qui enregistre (§2.1) ; s'il est
FAUX, il écrit `R(96)`, que la relecture Theater lit si elle est VRAIE. **Hypothèse : le film porte
`R(96)` brut** ; la fermeture tranche entre les deux.

Validité à l'écriture (`FUN_1407699d0`) : `bloc[1] & ~3 == 0` ; si `flags & 1` : lacet dans
`[0 ; 2pi]`, tangage dans `[-pi/2 ; pi/2]` (codes `16384..49152`) ; si `flags & 2 == 0` :
`flags != 0` ; si `flags & 2` : `(+0x20)+1 < 4`, `FUN_1406d5058(+0x6d)`, `FUN_1406d4fd4(+0x70)`,
`(+0x73)+1 < 0x41`, positions et vecteurs finis.

## 6. L'oracle d'impossibilité de la vue C du film (tiré de l'écrivain)

Une vue C de film lue au bon bit ne peut PAS montrer :

| signe | preuve |
|---|---|
| un kind 1, 2 ou 3 | §2.2 : seules des entrées kind 0 entrent dans les tampons par joueur |
| deux entrées d'index non strictement croissant | §2.1, boucle k = 0..31, tampon choisi par l'index écrit |
| plus de 32 entrées | idem |
| un code analogique 63 | `FUN_1406d5bf4` ; `FUN_140769f90` n'écrit un bloc que s'il passe `FUN_1406d33cc` |
| `+0x14` présent et nul | `FUN_1406d143c` : `W(1) = (+0x14 != 0)` |
| un bloc 0xbc à `flags == 0`, ou `flags & 1` avec un tangage hors `[16384 ; 49152]` | `FUN_14076a064` n'écrit qu'un bloc qui passe `FUN_1407699d0` |

Ne sont PAS des signes : `a = b = 0` (écrivable, §2.3), un en-tête `FUN_1406cdc04` présent (variante
froide), une vue C vide (aucun tampon actif ce tick).

Ce que cela dit de l'hypothèse de travail (« la vue C est lue au mauvais endroit ») : à une position
quelconque, le premier bit vaut 0 une fois sur deux (vue vide « fermée » sur un terminateur faux, la
signature mesurée par le port Rust : 94,2 % des trames non fermées à trois vues ont une vue C vide,
`RAPPORT_PORT_RUST_2026-10-01.md` §4.3), et sinon le kind vaut 1, 2 ou 3 trois fois sur quatre. Les
4 849 « kind non porté » et une part des 1 549 « bloc 0xbc » sont la même désynchronisation vue sous
un autre angle. T5 ne mesure pas ; la mesure est au §7.

## 7. Écarts, correctifs, vecteurs, mesures

### 7.1 Écarts du Go (`frame_vue_controle.go`)

1. `consumeVueC` traite un kind 1/2 comme un TROU de grammaire (`ArretVueCKindNonPorte`) ; dans un
   film c'est une impossibilité de l'écrivain. Il CONTINUE sur un kind 3 (`kindVueCNeant`), que
   l'écrivain n'émet jamais.
2. Aucun contrôle d'ordre des index, ni de borne 32, ni du code analogique 63, ni de `+0x14` nul.
3. Le bloc 0xbc est refusé (`ArretVueCBlocBC`) alors que sa grammaire est entièrement résolue sauf la
   fourche de `+0x74`.
4. Commentaire de `ArretVueCKindNonPorte` inexact (§4).
5. Le port Rust a la même forme (`frame/controls.rs:96-171`, « Unsupported(control handler kind 1 or
   2) »).

### 7.2 Correctifs proposés (phase 2)

- C-a : kind != 0 (1, 2 ET 3) -> nouvel arrêt `ArretVueCKindHorsFilm` (« impossible chez
  l'écrivain du film ») ; index non croissant ou 33e entrée -> `ArretVueCOrdre` ; code 63 ou `+0x14`
  présent nul -> `ArretVueCValeurImpossible`. Aucun portage des kinds 1/2 dans le décodeur de film.
  La carte de fermeture les compte comme des DÉSALIGNEMENTS prouvés, pas comme des trous.
- C-b : porter le bloc 0xbc (§5) avec ses lecteurs feuilles déjà présents dans `grammar`
  (`readVarWidthInt`, `consumeBipedDesiredGrenadeSet`, `consumeBipedDesiredAbilitySet`,
  `consumeCompressedDir140c1e79c`, `lireE420`, `consume1408f0ac4`) et un `consume14076d528` paramétré
  par la largeur de norme (10, 8). Fourche `+0x74` : les deux formes en `research`, la fermeture
  choisit, puis UNE seule en production.

### 7.3 Vecteurs de test (d'après l'écrivain ; vue C seule, bit 0 = premier bit du tampon)

Entrée kind 0 minimale de l'écrivain (28 bits, la forme constante de `dad793c7`) :
`1 00 0 iiiii 1 | 0 011111 011111 0 0 0 | 0 | 0` (en-tête, `a = 1`, bloc 0x68 : `g = 0`, couple
31/31, trois absences, bloc d'action vide `g = 0`, `b = 0`).

| vecteur | octets | attendu aujourd'hui | attendu après C-a / C-b |
|---|---|---|---|
| V1 : idx 0 puis idx 1, terminateur au bit 56 | `80 4F BE 08 0C FB E0 00` | fermé, 2 entrées | fermé, 2 entrées (non-régression) |
| V2 : idx 1 puis idx 0 | `80 CF BE 08 04 FB E0 00` | fermé, 2 entrées (accepté à tort) | `ArretVueCOrdre` |
| V3 : une entrée kind 2 réseau (`1 10 0 1 00011 11111110`, terminateur) | `C8 FF 80` | `ArretVueCKindNonPorte` | `ArretVueCKindHorsFilm` |
| V4 : idx 0, `a = 0`, `b = 1`, bloc 0xbc `flags = 1`, `+0x01 = 0`, lacet 0, tangage `0x8000` (0 rad), `+0x10` absent ; terminateur au bit 52 | `80 21 00 00 10 00 00` | `ArretVueCBlocBC` | fermé, 1 entrée, bloc lu (41 bits) |
| V5 : comme V4 mais `flags = 0` | `80 20 00 00 10 00 00` (bits 11-15 = `00000`) | `ArretVueCBlocBC` | `ArretVueCValeurImpossible` (écrivain : `flags != 0` exigé) |
| V6 : V1 avec le premier scalaire analogique à `111111` (code 63, bits 11-16) | `80 5F BE 08 0C FB E0 00` | fermé, 2 entrées | `ArretVueCValeurImpossible` |

V5 détaillé : `1 00 0 00000 0 1 | 00000 00 0^17 1 0^15 0 | 0` = `80 20 00 00 10 00 00`.

### 7.4 Mesures qui prouveraient l'effet (carte de fermeture v2)

- M1 (vérifie le constat 1 sur les données) : sur les 20 films, dans les paquets FERMÉS, nombre de
  kinds 1/2/3, d'index non croissants, de codes 63 : attendu **0** partout (si un build ancien en
  montre, l'écrivain de ce build diffère : le noter par build).
- M2 (impute les 4 849 + 1 549) : ventiler « kind non porté » et « bloc 0xbc » par sortie de vue B
  (terminateur / rejet / autre). Attendu : la quasi-totalité des « kind non porté » après un rejet.
- M3 (sous-cause du hors cadre) : pour « terminateur hors cadre », ventiler par vue C vide / non
  vide, et pour les non vides par verdict de l'oracle §6. Taux de vue C vide sur les paquets fermés
  du même film = ligne de base.
- M4 (gain de C-b) : paquets « bloc 0xbc » qui ferment avec chaque forme de `+0x74` ; attendu : une
  seule forme ferme ; gain borné par 1 549 paquets / 17 685 records utiles (carte v1).
- Effet attendu sur la fermeture : C-a = 0 paquet de plus (requalification) ; C-b <= 1 549 paquets,
  probablement beaucoup moins si M2 confirme le désalignement.

## 8. Questions ouvertes

- Q1 : les builds anciens (version-31/33, HI_1_4_1..HI_1_12_0) ont-ils le même enregistreur ? Le
  binaire lu est celui installé ; M1 par build répond.
- Q2 : valeur de `FUN_1409c94b8()` (chemin différé) et de `FUN_1404f293c()` (fourche `+0x74`)
  pendant l'enregistrement et pendant la relecture Theater : non résolues statiquement.
- Q3 : sens exact de `DAT_145121140` : sélecteur de variante de moteur posé au chargement
  (`FUN_140a938b4` -> `FUN_140a93ec8(FUN_14051a4b8(...))`, valeurs 0..3, `== 1` pour l'objet
  `0x145120f00` à vtable `0x143dedb48`). Le Go retient la forme `!= 1`, arbitrée par la fermeture.
- Q4 : la sélection des entrées en amont de `FUN_14076b0e8` (qui décide d'écrire une entrée kind 0
  quand les deux masques sont nuls) n'est pas lue ; `a = b = 0` reste donc écrivable.

## 9. Hors périmètre (consigné, non traité)

- `FUN_142987460` ignore les codes de retour des trois vues (déjà établi par T3 pour la vue B).
- Le lecteur du jeu, sur une entrée kind 0 `a = b = 0`, arrête la vue C (code 3) alors que
  l'écrivain peut l'écrire : le jeu perd alors les entrées suivantes du tick. Sans effet hors ligne.
- `plafondToursVueC = 64` pourrait descendre à 32 (borne de l'écrivain), sans effet de sortie.
