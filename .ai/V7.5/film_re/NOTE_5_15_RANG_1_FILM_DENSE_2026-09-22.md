# Note 5.15 — Le trou du rang 1 sur film dense : deux grammaires dans la meme boucle (2026-09-22)

> Lot 5.15, branche `feat/decfilm-65`, base `a9eae032c`. Cette note est le DOUBLE de la section
> du plan (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, « Post-chantier — lot 5.15 ») : le plan fait
> foi sur l avancement, cette note sur la GRAMMAIRE.
>
> Elle prend la suite de `NOTE_5_14_CLASSES_DE_VUE_2026-09-22.md` (§7, le reste dominant, et §8,
> la ligne « le trou du rang 1 sur film dense »), qui avait CHIFFRE le trou sans le nommer.

## 1. Ce que ce lot corrige d abord : une moyenne prise pour une constante

La decouverte D1 du 5.14 disait : « 23 852 paquets de `bfecd02b` ont leurs trois rangs lus et
laissent environ **1 076 bits chacun** ». La mesure exacte de la distribution dit autre chose :

| mesure | `bfecd02b` |
|---|---:|
| paquets a trois rangs portes laissant du reste | 23 852 |
| classes de largeur DISTINCTES | **2 226** |
| largeur minimale | **42** |
| largeur maximale | **6 162** |
| moyenne | 1 076,1 |
| repartition modulo 16 | **PLATE** (1 396 a 1 634 par residu) |

`1 076` etait la MOYENNE. Aucune largeur nommee ne s en deduit, et l hypothese « 1 076 bits sur
8 participants = 134,5 bits par participant » du brief tombe d elle-meme : c etait une division.

**LEÇON DE METHODE, ET ELLE EST DEJA DANS LA LISTE DES PIEGES** : un reste « quasi constant »
publie sans son histogramme est un reste dont on n a pas regarde la distribution.

## 2. Ou commence le reste, et comment la vue B sort

La marche est rejouee rang par rang avec les curseurs intermediaires
(`t515Marcher`, meme decoupe que `c514Cause` du gate 5.14).

**LE RESTE EST EN QUEUE DE PAQUET** : les trois rangs sont lus, le reste commence apres le
terminateur de la vue C. Il n est pas au milieu d un rang.

**MAIS LA VUE B NE SORT PAS SUR SON TERMINATEUR.** Les deux sorties `hitEnd` de
`decodeInferLoop` sont indiscernables de l exterieur et ne portent pas la meme grammaire ; en
relisant les bits qui precedent le curseur de sortie (terminateur = `000` ; rejet = le curseur
laisse a la fin d un en-tete `[1][idLow 13][tag 2]`) :

| sortie de la vue B | `bfecd02b` | `dad793c7` |
|---|---:|---:|
| **rejet de table de vue** | **23 452** | **14** |
| terminateur `recEnd` | 400 | 0 |

Et le slot rejete, interroge dans le monde hors ligne :

| l en-tete rejete | `bfecd02b` | `dad793c7` |
|---|---:|---:|
| slot **JAMAIS LIE** | **21 988** | **14** |
| slot lie, vue INCONNUE | 0 | 0 |
| slot lie, MEME vue | 124 | 0 |
| slot lie, AUTRE vue | **0** | **0** |

**ZERO rejet porte sur une entite d une autre vue.** La garde ne separe pas trois tables : elle
rejette l inconnu. Et desarmer la garde ne referme rien — depuis l en-tete rejete, garde
desarmee, **30 paquets sur 22 112** ferment a reste NUL : sans archetype le corps ne se lit pas.

## 3. Deux grammaires dans `FUN_1406cd128`, et le depot a transcrit la mauvaise garde

La boucle de records de la vue B se scinde a `1406cd24d` sur le global `DAT_14474cd78`.

```
FUN_1406cd128(vue, ctx, reader, budget, tableau, nbRecords) :
   mode  = *(char *)(vue + 0x12)          ; sortie code 2 au premier tour si non nul
   extra = FUN_14076cea8()                ; HasExtraFields
   si FUN_1409c94b8() : FUN_142f25334(...)  <- ZERO bit (recopie de tampon)
   boucle {
      [si extra] R(32)
      prefixe = R(1) ; si 1 -> type = 3 (DELTA)
                       sinon type = R(2) ; si 0 -> FIN DE LISTE
      si budget <= rendus et type != 0 -> FUN_1406cd3a8(), RETOUR
      si DAT_14474cd78 == 0 : ................ BRANCHE A (aller-retour d etat)
      sinon                 : ................ BRANCHE B (LE FILM)
   } tant que le code de retour est 0
   *nbRecords = rendus
```

### 3.1 Branche A — `DAT_14474cd78 == 0`

```
FUN_1406d3140(0, reader, 7, &eid)              ; idLow + tag(2)
type 1 (NEW)   : [si extra et FUN_1406cf008(reader)] R(8) ; FUN_141f86704(...)
type 3 (DELTA) : l = (eid & 0x3fffffff) * 0xa0
                 si (vue[0x38][l + 8] == eid && vue[0x38][l + 2] == (short)type)
                       FUN_141f86b58(...)      ; le corps
                 sinon code = 2                ; REJET, ZERO bit de corps
type 2 (DEL)   : [si extra et FUN_1406cf008(reader)] R(8) ; R(32)
si (ctx[1] == 0 || type == 3) : le record est pousse dans `tableau` (pas de 0xc0)
```

C est cette garde-la que le depot porte (`rejetDeVue`, `World.VuePossede`). **Le jeu ne
l emprunte que dans `FUN_1428e24bc`**, qui abaisse `DAT_14474cd78` a 0, appelle `vtable[0x40]`,
puis **le restaure** (`1428e26a4`, `1428e2701`) — apres avoir pre-rempli la table de sa vue
depuis une liste de reference (`vtable[0x10]`, entrees `slot = e & 0x1fff` sur **13 bits**,
type `e >> 0x17 & 0x7f` sur **7 bits**, posees une a une par `FUN_142f2c658`). Le tout sur le
PREMIER groupe de vues (`conteneur + 0x1a0`, decouverte D4 du 5.14), avec un tampon de `0x1b000`
octets : c est un ALLER-RETOUR D ETAT, pas le chemin du film. Dans cette branche la table est
donc COMPLETE par construction, et la garde ne rejette jamais rien.

**`DAT_14474cd78` VAUT 1 DANS L IMAGE** (`read_memory 0x14474cd78` rend `01 00 00 00`).

### 3.2 Branche B — `DAT_14474cd78 != 0`, c est-a-dire le film

```
base    = DAT_1451f9908 ; largeur = DAT_1451f990c
si DAT_144706104 == 0 : base = 0 ; largeur = DAT_144706100
w = FUN_1406d310c(largeur)                     ; largeur EN BITS de `largeur`
si w >= 1 : eid = base + R(w)
tag = R(2) ; eid |= tag << 30
si (vue[0x1b320] != 0 && FUN_142f2b5c4(eid, vue[0x20])) :
      code = FUN_1406cbaa0(type, eid, vue + 0x38, vue, ctx, reader, VRAI)
      si code != 0 -> SORTIE
      ... FUN_142f29538(vue[0x1b320], ...)     ; publication vers un second puits
sinon code = FUN_1406cbaa0(type, eid, vue + 0x38, vue, ctx, reader, FAUX)
```

Et `FUN_1406cbaa0` — le dispatcheur des trois types :

```
FUN_1406cbaa0(type, eid, &vue[0x38], vue, ctx, reader, drapeau) :
   slot = eid & 0x3fffffff
   # PROLOGUE : ZERO BIT, ET IL AGRANDIT LA TABLE
   cardinal = (vue[0x40] - vue[0x38]) / 0xa0
   si (cardinal <= slot && vue[0x50] && vue[0x70]) :
        FUN_1411b3c84(vue + 0x38, max(0x1fff, slot + 1))       <- AGRANDISSEMENT
        pour chaque entree neuve : FUN_1408f15c8(base + i * 0xa0)   <- CONSTRUCTION
   decodeur = *(vue + 0x20)

   type 1 (NEW) :
      [si HasExtraFields et FUN_1406cf008(reader)] R(8)
      si (ctx[2] == 0 && drapeau == 0 && FUN_1408f1314(vue, eid) == 0) : code 2
      sinon FUN_1408f1aa4(*decodeur, eid, ctx, reader, drapeau)

   type 2 (DEL) :
      [si HasExtraFields et FUN_1406cf008(reader)] R(8)
      R(32)
      FUN_1408f0d58(*decodeur, eid, ctx) ; FUN_1408f1244(vue, eid)

   type 3 (DELTA) :
      # LA GARDE, ET ELLE NE PORTE PAS SUR vue[0x38]
      t = *(decodeur + 0x20) ; n = (*(decodeur + 0x28) - t) / 200
      si (eid == 0xffffffff || n <= slot || *(uint *)(slot * 200 + t) != eid) :
            code = 3 - (*(int *)(DAT_144c1cfa8 + 4) != 2)      <- 2 ou 3, ZERO bit
      sinon :
            b = FUN_1406cdc04(reader)                          ; R(1) [+ R(7)], sentinelle 0xff
            si b == 0xff : baseline = NULL
            sinon : i = (*ctx - b) - 1 ; baseline = FUN_141fda280(historique, i)
            code = FUN_1406caad8(*decodeur, eid, ctx, baseline, reader, drapeau)
```

et `FUN_1406caad8` rend **2 d entree** si `*(int *)(DAT_144c1cfa8 + 4) != 2`, reteste la MEME
garde (`*(uint *)(slot * 200 + t) != eid` -> 3), lit l archetype dans
`*(int *)(slot * 200 + t + 4)`, puis appelle l iterateur de masque et de composants
`FUN_14076cb60` en lui passant, dans un bloc d arguments, **le tampon de baseline**
(`local_2480..uStack_2474`) et l etat courant (`local_2488`).

**TROIS CONSEQUENCES, ET ELLES REQUALIFIENT D1.**

1. **La branche vive n a AUCUN rejet sur slot inconnu de `vue[0x38]`** : son prologue AGRANDIT
   cette table et CONSTRUIT l entree manquante. `rejetDeVue` / `World.VuePossede` transcrivent
   la garde de la branche A — celle de l aller-retour d etat.
2. **La garde vive porte sur une AUTRE table** : celle du decodeur partage
   (`*(vue + 0x20)`, champ `+0x20`, **pas de 200**), sur l eid **ENTIER**, plus une borne de
   cardinal. C est aussi de la que vient l ARCHETYPE d un delta.
3. **Cette table est alimentee a chaque NEW** : `FUN_1408f1314(vue, eid)` n est pas un filtre,
   c est l ALLOCATION — `FUN_1408f18d0()` (predicat de mode, sans argument) puis
   `FUN_1408f1618(vue[0x20], eid)` et `FUN_1408f1358(vue, eid, 3)`.

D1 du 5.14 disait « la vue B ferme sa liste trop tot ». La lecture de l ecrivain dit :
**la marche hors ligne n a pas la table qui donne l archetype d un slot jamais vu.** Et la
mesure dit qu elle n en lit pas assez pour la construire : **685 records NEW pour 160 739
DELTA** sur tout `bfecd02b`.

## 4. Le cadrage de paquet est clos : un paquet de film = une trame

`FUN_14298816c` est le SEUL appelant de `FUN_142987460`, et il porte son chemin source :
`shared\engine\source\blofeld\saved_games\SavedFilmChunks.cpp:0x533`.

```
FUN_14298816c(session, entree, tampon) :
   memset(tampon, 0, 0x1b01c)
   si FUN_142988338(session, tampon, entree[1], 0) :          <- memcpy du paquet, ZERO grammaire
      lecteur = FUN_1424c7b4c(neuf, session[0x270], entree[1])
      FUN_1406d5cc0(lecteur, 3)
      si DAT_144db4330 == 0 : FUN_142987460(session, lecteur)  <- UNE SEULE FOIS
      si (lecteur[0x24] != 0 || lecteur[0x18] * 8 < lecteur[0x2c]) : ECHEC
```

```
FUN_142987460(conteneur, lecteur) :
   DAT_144706104 = FUN_1406cf008(lecteur)      ; LE bit de configuration
   pour rang de 0 a 2 :
      vue = *(conteneur + 0x228 + rang * 8)
      vue->vtable[0x60](vue, 0xa00 - rendus, &tableau[rendus], &n)   ; ZERO bit
      vue->vtable[0x40](vue, &ctx, lecteur, 0xa00 - rendus, &tableau[rendus], &n)
   pour rang de 0 a 2 : pour chaque record de ce rang : vue->vtable[0x48](vue, record)
   FUN_1406d07b0(tableau, rendus, 0)           ; SANS le lecteur
```

**UN paquet de film = UNE trame.** Le reste n est donc pas une seconde trame, le budget
(`0xa00 - rendus` = 2 560 records) n est jamais epuise, et le premier groupe de vues
(`conteneur + 0x1a0 / +0x1b4c8 / +0x1b908`, D4 du 5.14) n est pas dans le chemin du film.
L hypothese (b) du brief est refermee.

**ET LE JEU NE VERIFIE PAS QU UN PAQUET EST ENTIEREMENT CONSOMME** : son test est un
DEBORDEMENT (`lecteur[0x24]`, `lecteur[0x18] * 8 < lecteur[0x2c]`), jamais « tout lu ». Notre
gate « reste a bourrage NUL » est donc plus strict que le jeu. C est le bon choix hors ligne —
il a trouve tous les trous de ce chantier — mais ce n est pas un invariant du format, et il faut
le savoir avant de poursuivre un dernier pourcent.

## 5. `IDLowBits = 13` vient desormais du binaire, et le bit de configuration est innocente

`FUN_1406d310c(x)` rend la largeur en bits de `x` (recherche du bit de tete, plus un arrondi
quand les bits bas ne sont pas nuls ; `FUN_1406d310c(0) = 0`).

La branche vive lit `FUN_1406d310c(DAT_144706100)` bits d identifiant, et
**`DAT_144706100` vaut `0x1fff` dans l image** -> `FUN_1406d310c(0x1fff) = 13`. Le balayage du
5.14.3 (« 13 seul candidat sur 10..15 ») est CONFIRME par l ecrivain : ce n est plus une mesure.

Ce global est un **FILIGRANE**, ecrit par `FUN_1408f1618` (depuis `FUN_1408f1314`, donc a chaque
NEW) et par `FUN_142f2f0cc` (l allocateur de slot), toujours par le meme triplet :

```
DAT_1451f98d4 = slot - 0x1ff
DAT_144706100 = slot + 1
DAT_1451f990c = slot + 1
```

sous la condition `cardinal <= slot`, et la table est agrandie a `max(0x1fff, slot + 1)`. Donc :
tant qu aucun slot n atteint `0x1fff`, le filigrane ne rebouge plus et la largeur reste 13. Sur
les deux temoins les slots plafonnent a **8 186** (sous `0x1fff = 8 191`).

**LE BIT DE CONFIGURATION N EST PAS UN SELECTEUR D IDENTIFIANT.** Il choisit
`[base 0 ; bits(DAT_144706100)]` (bit a 0) ou `[base DAT_1451f9908 ; bits(DAT_1451f990c)]`
(bit a 1). Or `DAT_144706100` et `DAT_1451f990c` sont TOUJOURS ecrits ensemble a la meme valeur,
et **`DAT_1451f9908` n a AUCUN ecrivain** — son unique xref est la lecture de `1406cd267` — et
vaut 0 dans l image. Les deux formes coincident. Controle sur le film : le bit vaut **1** sur les
paquets du temoin du §6, et le premier record y decode a 13 bits, base 0.

Suspicion RETIREE, et ecrite ici pour ne pas etre rouverte.

## 6. Le temoin de 96 bits : douze octets qui portent tout le trou

Sur `dad793c7`, **douze** des quatorze paquets fautifs font EXACTEMENT 96 bits, portent UN seul
record et laissent EXACTEMENT 42 bits. Ils sont identiques bit pour bit a partir du bit 37, sauf
un compteur de 16 bits qui croit d un **pas constant de 4 584** (`0x3067`, `0x424F`, `0x5437`).

```
10100000011110110100001000000000 01000100101000100100100001000001 00000100110000011001110000000000

bit  0        1     bit de configuration du frame-processeur        <- VAUT 1
bit  1        0     terminateur de la vue A vide
bit  2        1     prefixe DELTA
bits 3..15    123   idLow, 13 bits
bits 16..17   1     tag de generation
bit  18       0     selecteur de baseline ferme (FUN_1406cdc04)
bits 19..29   0x1   masque
bits 29..37         i0 high-frequency (ti = 4)
bit  37       1     prefixe DELTA        <- l en-tete que la marche REJETTE
bits 38..50   1314  idLow
bits 51..52   1     tag
bit  53       0     la marche le prend pour le terminateur de la vue C
bits 54..69         16 bits CONSTANTS sur les douze paquets
bits 70..85         16 bits, le compteur a pas constant de 4 584
bits 86..95         dix bits a zero
```

**C EST LE PLAN DE TRAVAIL DU LOT SUIVANT** : douze octets, une entite connue, un reste de
largeur fixe — au lieu de 25 666 138 bits repartis sur 2 226 classes. Et un indice cadre : la
vue C de `dad793c7` consomme normalement **28 bits CONSTANTS** (item 5.14.4, couple analogique a
zero exact) et n en consomme qu UN ici. Le curseur n est pas la ou la vue C l attend.

## 7. Deux fausses pistes fermees, et elles valaient la mesure

**(a) « UNE LARGEUR MANQUANTE A UN SEUL ENDROIT » — REFUTEE.** Recherche, par ecart croissant
autour de la fin du dernier record lu, d une position de reprise qui mene la vue B a son
terminateur, la vue C au sien, et le paquet a reste NUL : **10,4 %** seulement des paquets
fautifs se recalent dans plus ou moins 192 bits, et les ecarts sont DIFFUS (**322 classes**).
Le decalage n est ni fixe ni local.

**(b) « `i25 unit-command-tick` LIT TROP COURT » — REFUTEE PAR L ECRIVAIN.** `i25` est le dernier
composant lu avant le rejet dans environ **85 %** des paquets fautifs (9 470 + 3 824 + 3 784 sur
`ti=35`, 1 965 sur `ti=40`) contre 13 % du temoin ferme. C est un CONFONDANT : `i25` est
simplement le bit de masque le plus haut de la plupart des deltas de bipede, et les paquets
fautifs sont les paquets charges en bipedes. Le port a ete reverifie contre `FUN_1406cfb28` :

```
FUN_1406cfb28(_, reader, ctx) :
   FUN_140c50d1c(&out, reader)        R(1) [+ R(8)]   sentinelle 0xffff  -> obj[0x540]
   g1 = R(1) ; si 0 : obj[0x538] = obj[0x53c] = -1, RIEN N EST LU, fin
   g2 = R(1)
   FUN_140cec0a0(reader)              R(1) [+ R(8)]   -> obj[0x538]
   si g2 == 0 : FUN_140cec0a0(reader) une seconde fois
                                      -> obj[0x53c]
   g3 = R(1)                          -> obj[0x542] bit 2
   si g3 == 0 : R(1) -> bit 0 ; R(1) -> bit 1
```

`consumeUnitCommandTick` porte exactement cela. **Rien a corriger**, et c est ecrit ici pour
qu `i25` ne soit pas re-suspecte au prochain lot.

## 8. Les maillons restants, par adresse

| maillon | adresse | ce qu il ouvre |
|---|---|---|
| **la table de datums par slot du decodeur partage** | `*(vue + 0x20) + 0x20`, pas de **200** ; eid a `+0x00`, archetype a `+0x04` ; alimentee par `FUN_1408f1314` -> `FUN_1408f1618` / `FUN_1408f1358` | l archetype d un delta sur un slot que ni image-cle ni NEW lu ne declare — **le trou du rang 1** |
| **la baseline du corps de delta** | `FUN_1406cdc04` (selecteur `R(1)` [+ `R(7)`]) -> `FUN_141fda280` -> `FUN_1406caad8` -> bloc d arguments de `FUN_14076cb60` | si la baseline change une LARGEUR (et pas seulement une valeur), aucune marche de film dense n est garantie |
| le predicat de mode du NEW | `FUN_1408f18d0()` (sans argument) et `*(int *)(DAT_144c1cfa8 + 4)` | ce qui decide qu un record est LU ou que la liste sort avec le code 2 |
| le second puits de la vue B | `vue[0x1b320]`, predicat `FUN_142f2b5c4`, publication `FUN_142f29538` | ou vont les records qu un film dense pourrait router ailleurs |
| la table des 123 types de message de la vue A | `*(obj[0x18] + 0x210 + genre * 8)` | inchange depuis le 5.14 (jamais rencontre sur les deux temoins) |
| les BITS D ACTION du controle | `FUN_1406d025c` | inchange depuis le 5.14 (ouvert 108 fois sur `bfecd02b`) |

## 9. Les instruments

Tous `_test.go` sous `//go:build research`, paquet `grammar`, variables `MOUV511_FILM`,
`MOUV511_BORNES`, `MOUV511_CARTE`. Aucun ne touche a la grammaire : ils MESURENT.

| test | role |
|---|---|
| `TestTrou515Position` | les curseurs rang par rang, la SORTIE de la vue B (terminateur ou rejet), l histogramme EXACT de la largeur du reste, 40 echantillons et leurs bits bruts |
| `TestTrou515Rejet` | l en-tete rejete : slot, tag, lie ou non, dans quelle vue — et ce que la marche lit quand la garde de vue est DESARMEE |
| `TestTrou515Cascade` | le rang du paquet dans son chunk (fermes, fautifs, premier fautif) et le sort des slots rejetes face aux images-cles du film |
| `TestTrou515Population` | NEW lus, DELTA lus, slots des images-cles, slots rejetes — etendues et medianes |
| `TestTrou515Dernier` | le DERNIER record de la vue B avant le rejet, archetype et dernier composant, avec le temoin des paquets FERMES |
| `TestTrou515Recalage` | l ecart de reprise le plus proche qui ferme le paquet — l instrument qui a REFUTE la largeur unique |
| `TestTrou515Minimal` | le dump INTEGRAL des paquets fautifs de 96 bits (§6) |
