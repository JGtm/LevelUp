# Note 5.14 — Les trois classes de vue d une trame delta (2026-09-22)

> Lot 5.14, branche `feat/decfilm-64`, base `c423188dc`. Cette note est le DOUBLE de la section
> du plan (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, « Post-chantier — lot 5.14 ») : le plan fait
> foi sur l avancement, cette note sur la GRAMMAIRE.
>
> Elle prend la suite de `NOTE_5_11_TABLES_PAR_VUE_2026-09-21.md` (§2, la fin de trame) et de
> l item 5.13.1 (e), qui avaient NOMME le trou sans le refermer.

## 1. L ordre des trois rangs est PROUVE, il n est plus suppose

`FUN_142987460` (frame-processeur) lit UN bit de configuration
(`DAT_144706104 = FUN_1406cf008(reader)`) puis parcourt SES TROIS VUES dans l ordre du tableau
`param_1 + 0x228 / 0x230 / 0x238`, en appelant sur chacune `vtable[0x60]` **sans le lecteur**
(zero bit : c est un vidage de records en attente) puis `vtable[0x40]` (la boucle de records).
Il n ecrit rien apres les trois vues.

Le registraire donne le rang de chaque classe. `FUN_141f855b4` :

```
FUN_1409c9860(conteneur+8, 0, conteneur + 0x3ce98, 1)   // vtable 0x1436a8700  -> FUN_14076a1c4
FUN_1409c9860(conteneur+8, 1, conteneur + 0x21b70)      // vtable 0x1436a87e0  -> FUN_1406cd128
FUN_1409c9860(conteneur+8, 2, conteneur + 0x3d2d8)      // vtable 0x1436a8770  -> FUN_1406cf548
```

et `FUN_1409c9860` clot le rang dans `*(int *)(vue + 8)` — le champ que `FUN_142f2e174` met dans
les deux bits de tete d un identifiant d image-cle (item 5.13.1). **Le rang 1 mesure sur les
images-cles des deux films temoins est donc la vue B, le gestionnaire d entites** : la seule des
trois grammaires que le depot portait. La correspondance du lot 5.13 (« le premier rang rencontre
est celui de la vue qui rend les records ») est confirmee par l ecrivain, et le decalage entre la
numerotation du jeu (rang 1) et l index du monde hors ligne (0) est desormais ECRIT
(`vueDeLImageCle`).

| rang | objet | vtable | `vtable[0x40]` | classe |
|---:|---|---|---|---|
| 0 | `conteneur + 0x3ce98` | `0x1436a8700` | `FUN_14076a1c4` | **vue A** — un flux de MESSAGES |
| 1 | `conteneur + 0x21b70` | `0x1436a87e0` | `FUN_1406cd128` | **vue B** — `replication_entity_manager_view.cpp` |
| 2 | `conteneur + 0x3d2d8` | `0x1436a8770` | `FUN_1406cf548` | **vue C** — `replication_control_view.cpp` |

Le conteneur porte DEUX groupes de trois vues (`FUN_140b87664` : `+0x1a0 / +0x1b4c8 / +0x1b908`
puis `+0x21b70 / +0x3ce98 / +0x3d2d8`) ; `FUN_141f855b4` enregistre le SECOND.

## 2. Vue A (rang 0) — un flux de messages, et le bit que le depot ne savait pas nommer

```
FUN_14076a1c4(vue, _, reader, budget, _, nbRecords) :
   si vue[0x11] != 0 : ZERO bit, sortie code 2          <- etat runtime, hors flux
   boucle {
      b = R(1) ; si 0 -> FIN
      si budget < 1 -> sortie code 3, zero bit de corps
      corps = FUN_14080a9d4(vue[0x28][1], FUN_140514010(vue[0x14]), reader)
   } tant que le corps rend 0
   *nbRecords = 0                                        <- SANS CONDITION
```

```
FUN_14080a9d4(obj, arg, reader) :
   genre = R(7)
   si genre < 0x7b (123) :
      def = *(obj[0x18] + 0x210 + genre * 8)             // la table des 123 types de message
      ... (cache : FUN_1406cb0cc / FUN_14080abe4 ; trois tours de R(1) de service)
      charge = def->vtable[0x68](def, taille, dst, reader, 1)
      si HasExtraFields (FUN_14076cea8) et R(1) : R(32)
```

**ELLE NE REND JAMAIS UN RECORD** (`*param_6 = 0` sans condition) : ce n est pas une vue
d entites, c est un flux de MESSAGES RESEAU indexes par un `R(7)` sur 123 types. La vtable de
`initiate_mobility_action` (`143d0a000`, item 5.13.2) est l une de ces 123 definitions — c est le
meme mecanisme, vu depuis la trame.

**LE BIT.** La documentation de `DefaultPacketPreambleBits` disait : « le desassemblage n en
etablit qu UN ; le SECOND bit n est PAS localise, il est etabli par la MESURE ». **Ce second bit
est le terminateur `R(1) = 0` de la vue A vide.** L amorce de deux bits n est pas une amorce :
c est `[bit de configuration][vue A vide]`.

| film | paquets a amorce de frame-processeur | vue A VIDE | vue A NON VIDE |
|---|---:|---:|---:|
| `dad793c7` | 5 345 | **5 345** | **0** |
| `bfecd02b` | 31 232 (tous paquets delta) | tous | **0** |

La charge des 123 genres n est pas portee : `consumeVueA` rend `Porte = false` des qu un corps
s ouvre, et le paquet est SIGNALE au lieu d etre decadre. Sur les deux temoins ce cas ne se
presente jamais.

## 3. Vue C (rang 2) — la vue de CONTROLE

```
FUN_1406cf548(vue, _, reader, budget, _, nbRecords) :
   tier = FUN_140ce620c()                                <- version, hors flux
   c    = FUN_1409c94b8()                                <- drapeau runtime, hors flux
   si c : FUN_142f2539c(reader, &copie, &taille)          <- ZERO BIT (voir ci-dessous)
   boucle {
      b = R(1) ; si 0 -> FIN
      si budget <= rendus -> sortie code 3
      kind = R(2)
      kind 0 -> FUN_1406d0388   (l entree de controle)
      kind 1 -> FUN_142f29b38
      kind 2 -> FUN_142f29e54
      kind 3 -> ZERO BIT, la boucle continue
   }
   *nbRecords = rendus
```

**LE PROLOGUE COUTE ZERO BIT, ET C EST CE QUI REND SA GARDE RUNTIME SANS CONSEQUENCE.**
`FUN_142f2539c` ne lit pas le flux : il RECOPIE le tampon —
`memcpy(dst, *(void**)(reader+8), *(int*)(reader+0x18))` — une photo du payload pour la suite du
traitement. Son chemin source est
`shared\engine\source\blofeld\networking\replication\replication_control_view.cpp:421`
(chaine `143c999f0`), **et c est lui qui NOMME la classe**.

`kind == 3` est le seul cas ou la vue C enchaine sans rien consommer.

### 3.1 `kind` 0 — l entree de controle d un participant

```
FUN_1406d0388(...) :
   FUN_1406cdc04(reader)         R(1) ; si 1 -> R(7)        (sentinelle 0xff a 0 bit)
   idx = R(5)                    l index de controle, 0..31
   si idx > 0x1f -> return 3     INATTEIGNABLE (cinq bits)
   branche FUN_14048ee34() == 0 :
      a = R(1) ; si a -> FUN_1406cd860(reader, bloc 0x68)
      b = R(1) ; si b -> FUN_141fdae44(reader, bloc 0xbc)
   branche FUN_14048ee34() != 0 :
      FUN_1404f1ca4() ? FUN_142f2a17c (= R(1)) : FUN_142f29954 (= R(1) [+ R(5) + charge])
   le record est ecrit dans le tableau du frame-processeur : `rec[1] = idx`, `rec[0] = 0`
```

```
FUN_1406cd860(reader, out 0x68) :
   memset(out, 0, 0x68) ; out[1] = 0xff ; FUN_1406d02c8(out+0x18)      <- 0 bit
   R(1)                          si 1 : R(w) avec w = (DAT_145121140 == 1) ? 4 : 2   -> out[1]
   FUN_1406d6ef4(reader, out+4, out+0xc)
   R(1)                          si 1 : FUN_1406d84b4 (largeur passee par la PILE)  -> out[0x10]
   R(1)                          si 0 : FUN_1406d025c(out+0x18, reader) puis FIN
                                 si 1 : R(w2), w2 = (DAT_145121140 == 1) ? 7 : 5, puis
                                        FUN_142265fe3
```

```
FUN_1406d6ef4(reader, f[2], p3) :
   si DAT_145173840 == 0 :
      a = R(6) -> f[0]           a == 0    -> DAT_143cd84ec  (borne)
                                 a == 0x3e -> DAT_143cd8374  (borne)
                                 a == 0x1f -> 0.0            (ZERO EXACT)
                                 sinon     -> (a - 1) * DAT_143cd8908 - DAT_143cd890c
      b = R(6) -> f[1]           meme decodage
   sinon : deux FUN_1406d84b4(..., 6, 1, 1) + FUN_142e2baac
   R(1)                          si 1 : FUN_1406d84b4 -> *p3
```

**C EST UN COUPLE ANALOGIQUE** : deux scalaires quantifies sur six bits, symetriques, avec un
code de ZERO EXACT au milieu du domaine et deux codes de borne. Ecrits dans deux `float`.

```
FUN_1406d025c(dst, reader) :          LES BITS D ACTION
   FUN_1406d02c8(dst)                                  <- 0 bit
   g = R(1) ; si 0 -> FIN                              <- la charge est VIDE
   f1 = R(1) ; si f1 : 6 x R(1)  -> FUN_1431ab1ec(dst, i, j, bit) sur (0,0)(0,1)(0,2)(1,0)(1,1)(1,2)
   f2 = R(1) ; si f2 : 4 x R(1)  -> FUN_1431ab1cc(dst, i, j, bit) sur (0,0)(0,1)(1,0)(1,1)
   f3 = R(1) -> dst[0x10] bit 1
   si f3 : R(1) -> bit 3 ; R(1) -> bit 2 ;
           FUN_1431a0bbc(reader) -> dst+0x14 ; FUN_1431a0abc(reader) -> dst+0x18 ;
           FUN_1431a0cbc(reader, &v, dst+0x11) -> dst+0x1c / +0x20 / +0x24     (un POINT)
   dst[6] = FUN_1406d0f20(reader)
   si dst[0] & 7 ou dst[4] & 3 : dst[7] = FUN_1406d00ec(reader)
   si dst[2] & 7 ou dst[5] & 3 : dst[8] = FUN_1406d00ec(reader)
   FUN_142f26740(dst+0x28, reader, 0)
```

## 4. Ce que le port porte, et ce qu il REFUSE de deviner

`frame_vue_messages.go` (vue A) et `frame_vue_controle.go` (vue C), dispatch dans
`frame_harvest.go` (`decodeFrameParRangs`), bascule `GrammaireBalayage.ClassesDeVue` (defaut
LEVE au 5.14.3).

Le port porte le chemin dont TOUTES les largeurs sont resolues chez l ecrivain : branche
`FUN_14048ee34() == 0`, largeurs courtes (`DAT_145121140 != 1`), chemin quantifie de
`FUN_1406d6ef4` (`DAT_145173840 == 0`), et les gardes de presence LUES DANS LE FLUX. Des qu une
garde ouvre un champ dont la largeur vient de la pile ou d un sous-arbre non porte, la lecture
rend `false` : le curseur s arrete SUR le bit de garde et le paquet est signale. **Aucune largeur
n est inventee**, et les deux formes de `DAT_145121140` viennent toutes deux de l ecrivain — c est
la FERMETURE DES PAQUETS qui dit laquelle le film porte, comme elle arbitre `HasExtraFields`.

## 5. Le gate, renforce : le bourrage est ecrit A ZERO

« Reste dans [0 ; 7] » ne prouve pas qu un paquet est lu. Le bourrage d octet est ecrit A ZERO,
donc un reste qui porte un 1 est de la grammaire MANQUANTE meme quand il tient dans sept bits.
`TestClasses514Bourrage` exige que TOUS les bits du reste soient nuls.

| mesure | `dad793c7` avant | apres | `bfecd02b` avant | apres |
|---|---:|---:|---:|---:|
| paquets marches | 5 365 | 5 365 | 30 043 | 30 387 |
| paquets fermes (reste 0..7) | 5 338 | **5 341** | 888 | **2 884** |
| dont reste TOUS BITS A ZERO | — | **5 341 / 5 341** | — | **2 884 / 2 884** |
| debordements (reste < 0) | 18 (721 bits) | **2 (586 bits)** | 1 785 (412 992 865 bits) | **32 (110 998 bits)** |
| records `ti=35` | 75 | 75 | 114 458 | **129 572** |
| desyncs `ti=35` | 0 | 0 | 4 | 4 |
| etalon `i21` | 1,3 % | 1,3 % | 65,2 % | 65,5 % |
| records fantomes (rangs != vue B) | 13 | **0** | 304 | **0** |
| paquets non localises | 5 | 5 | 1 189 | 845 |

Le reste dominant de `dad793c7` est **UN bit sur 5 068 paquets, et ce bit est zero** — les memes
5 068 paquets que le gate 5.11.6 comptait a reste 0 en consommant 32 bits de pied avec la
grammaire du gestionnaire d entites.

**POURQUOI +15 114 RECORDS SUR `bfecd02b`, ET C EST LA LECON DU LOT.** Les flux des vues A et C
etaient decoupes en `[prefixe][idLow][tag]` ; quand ce decoupage rendait un type `DEL`, la boucle
faisait `w.Unbind(slot)` — elle DELIAIT une entite VIVANTE, et tous ses deltas suivants etaient
perdus. Un faux record ne coute pas un bit : il coute un parc.

`IDLowBits` a ete balaye de 10 a 15 sous la nouvelle grammaire (`TestClasses514IdLow`) : 13 reste
le seul candidat sur les deux films (99,55 % et 9,49 % de fermeture, contre 3,5 % au mieux
ailleurs). Le reste de `bfecd02b` n est donc PAS un cadrage d en-tete.

## 6. Ce que les vues A et C portent, en clair

`TestClasses514Contenu`, tout le film puis la fenetre du saut.

| mesure | `dad793c7` (tout) | `dad793c7` (26,0-27,2 s) | `bfecd02b` (tout) |
|---|---|---|---|
| vue A non vide | **0** paquet | **0** paquet | **0** paquet |
| entrees de controle lues | 5 202 | 71 | 20 318 |
| index de controle `R(5)` | **{0}** | **{0}** | **{0..7}** (1 962 a 3 860 chacun) + 21 index epars (2 a 27) |
| index `FUN_1406cdc04` `R(7)` | jamais present | jamais present | 27 valeurs, 1 a 5 fois |
| second champ `R(2)` | **1** (5 200 fois) | **1** (71) | 1 (20 220), 0 (3), 2 (7), 3 (7) |
| scalaire A (6 bits) | **31** sur 5 202 | **31** sur 71 | **64 classes** (0 a 63) |
| scalaire B (6 bits) | **31** sur 5 202 | **31** sur 71 | **64 classes** (0 a 63) |
| gardes ouvertes NON PORTEES | branche longue 1 | branche longue 1 | troisieme champ 137 · champ de pile 132 · branche longue 2 061 · **BITS D ACTION 108** · second bloc 1 |

**L INDEX DE CONTROLE EST L INDEX DU PARTICIPANT.** Un film a un joueur ne rend que l index 0 ;
un film a huit joueurs rend les huit index 0 a 7, avec des effectifs du meme ordre. Le nombre
d iterations de la boucle de la vue C suit : **une** sur `dad793c7`, **jusqu a huit** sur
`bfecd02b` (`kinds [0 0 0 0 0 0 0 0]`).

**LE COUPLE ANALOGIQUE EST A ZERO EXACT SUR TOUT `dad793c7`** : le code 31 (`0x1f`) est celui que
`FUN_1406d6ef4` transforme en `0.0`, et il vaut 31 sur les 5 202 entrees du film, **fenetre du
saut comprise (71 sur 71)**. Sur `bfecd02b` les deux canaux couvrent leurs 64 codes. C est ce qui
explique la fermeture a 28 bits CONSTANTS de `dad793c7` : sur ce film l entree de controle est un
en-tete a valeurs fixes.

**AUCUN CANAL D ETAT DE BIPEDE N EST LU PAR CE LOT, ET LE MAILLON EST NOMME — ce n est pas une
absence.** Le canal d ACTIONS existe dans le film : la garde `g = R(1)` de `FUN_1406d025c` est
OUVERTE **108 fois** sur `bfecd02b`, et le port s y arrete parce que sa charge (6 bits en 2x3,
4 bits en 2x2, un point de visee, `FUN_1406d0f20`, deux `FUN_1406d00ec`, `FUN_142f26740`) n est
pas portee. Sur `dad793c7` elle est FERMEE sur les 5 202 entrees : ce film ne porte pas de bits
d action, fenetre du saut comprise. Et aucune de ces positions de bit ne porte d ETIQUETTE dans
le binaire pour l instant — les noms `player_action_test_jump`, `player_action_test_melee`,
`player_action_test_equipment` (34 chaines de la famille `player_action_test_*`, `143c2da40` a
`143c2e5a0`) sont des fonctions de SCRIPT enregistrees par `FUN_140a3f6a4`, pas des etiquettes de
position dans `FUN_1431ab1ec` / `FUN_1431ab1cc`.

**PAS DE MONTEE DE SCHEMA.** Le lot ne publie aucun champ : le couple analogique est a zero sur le
film temoin, les bits d action ne sont pas portes, et aucune valeur n a d etiquette. `stances[].kind`
ne gagne pas de valeur, `replay.SchemaVersion` reste a **67**.

## 7. Ce qui reste, par cause et avec ses chiffres

`TestClasses514Restes`, sous la grammaire des trois classes.

| cause | `dad793c7` | `bfecd02b` |
|---|---:|---:|
| les trois rangs portes, reste hors bourrage | 14 paquets (6 311 bits) | **23 852 paquets (25 666 138 bits)** |
| rang 1 vue B : desynchronisation | 7 (32 bits) | 1 110 (1 751 992 bits) |
| rang 2 vue C : `kind` 0 non porte | 1 (14 bits) | ~2 240 paquets |
| rang 2 vue C : `kind` 1 non porte | 1 (23 705 bits) | 157 |
| rang 2 vue C : `kind` 2 non porte | 0 | 137 |
| rang 2 vue C : debordement | 1 (-2 bits) | 1 (-2 bits) |

**LE 100 % N EST PAS ATTEINT, ET LA PLUS GROSSE CAUSE N EST PAS DANS CE LOT.** Sur `bfecd02b`,
23 852 paquets ont leurs TROIS rangs lus jusqu au terminateur et laissent pourtant ~1 076 bits
chacun : la vue B ferme sa liste trop tot. Ce n est ni la vue A (jamais non vide) ni la vue C
(lue jusqu a son terminateur), et ce n est pas `IDLowBits` (balaye). C est un trou du rang 1 ou du
cadrage de paquet sur un film dense a huit joueurs, et il est hors du perimetre « les deux autres
classes de vue ».

## 8. Les maillons restants, par adresse

| maillon | adresse | ce qu il ouvre |
|---|---|---|
| la table des 123 types de message de la vue A | `*(obj[0x18] + 0x210 + genre * 8)`, charge par `def->vtable[0x68]` | la charge d un corps de vue A (jamais rencontre sur les deux temoins) |
| les BITS D ACTION du controle | `FUN_1406d025c` (ouverte 108 fois sur `bfecd02b`) ; setters `FUN_1431ab1ec` / `FUN_1431ab1cc` ; point `FUN_1431a0bbc` / `FUN_1431a0abc` / `FUN_1431a0cbc` | un eventuel canal de saut / d escalade en ENTREE |
| la largeur de `FUN_1406d84b4` | le 5e argument, passe par la PILE (`in_stack_00000028`) | le troisieme champ de `FUN_1406d6ef4` et `out[0x10]` de `FUN_1406cd860` |
| le second bloc du controle | `FUN_141fdae44` (bloc de 0xbc octets), largeurs `(DAT_145121140 == 1) * 2 + 2`, sous-lecteurs `FUN_14080cb98` / `FUN_1406d0f20` / `FUN_1424d0f48`, plus deux `R(0xb)` | la branche `b` de `FUN_1406d0388` |
| `kind` 1 et `kind` 2 de la vue C | `FUN_142f29b38`, `FUN_142f29e54` | 157 et 137 paquets de `bfecd02b` |
| la branche longue de `FUN_1406cd860` | `R(5\|7)` puis `FUN_142265fe3` | 2 061 entrees de `bfecd02b` |
| le trou du rang 1 sur film dense | — | 23 852 paquets de `bfecd02b`, ~1 076 bits chacun |

## 9. Les instruments

Tous `_test.go` sous `//go:build research`, paquet `grammar`, variables `MOUV511_FILM`,
`MOUV511_BORNES`, `MOUV511_CARTE`, `MOUV511_T0` / `T1`.

| test | role |
|---|---|
| `TestClasses514Marche` | les trois grammaires, une par rang : ce que chaque classe consomme |
| `TestClasses514Bourrage` | **LE GATE** : reste dans [0 ; 7] **ET tous ses bits a zero** |
| `TestClasses514Restes` | les paquets qui ne ferment pas, PAR CAUSE et par rang |
| `TestClasses514IdLow` | balayage de `IDLowBits` sous la nouvelle grammaire (piege 10 : suspecter l instrument) |
| `TestClasses514Contenu` | ce que les vues A et C PORTENT, valeurs lues, avec fenetre |
| `TestMouvement5116Gate` | le gate du 5.11.6, avec la bascule `MOUV511_CLASSES` |
