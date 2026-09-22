# NOTE 5.20 — L IMAGE-CLE LUE EN ENTIER, ET LA TABLE DE DATUMS QUE LE FILM PORTE A PART

Lot 5.20, branche `feat/decfilm-70`, base `5007af47c` (lots 5.1 a 5.19, grammar-2026-09-22.7,
schema 67). Ce lot part des decouvertes D1 et D2 du 5.19. Il en REFUTE une, en PORTE une autre,
et il trouve la source manquante — nommee par son adresse et mesuree dans les deux films.

> Le plan (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, section « Post-chantier — lot 5.20 ») fait foi
> sur l avancement ; cette note fait foi sur la GRAMMAIRE.

---

## 1. LE LECTEUR D IMAGE-CLE, EN ENTIER (5.20.1)

### 1.1 Qui lit le type 2, par adresse

Le bloc d image-cle n est PAS consomme par le repartiteur de paquets : `FUN_1428e22c0` ne connait
que neuf types (0, 1, 6, 7, 8, 9, 10, 0xb, 0xc) et le type 2 y tombe dans la queue de telemetrie
`FilmBlockReadError`. Il passe par la SECONDE voie a en-tete de 16 octets, celle que le 5.19 §6.1
avait nommee sans la suivre :

```
FUN_1428e2a04(session)                         le chargement d ETAT (seek / debut de chunk)
   FUN_14298924c(session, session[0x224])
   FUN_142988338(session, hdr, 0x10, 0)        un en-tete de 16 octets
   FUN_1428e2a9c(session)
      FUN_1429883ec(session, hdr, table)       <- LE BLOC DE TYPE 1 : LA TABLE DE DATUMS (§3)
      FUN_142988338(session, hdr2, 0x10, 0)    l en-tete du bloc d image-cle
      FUN_142988338(session, session[0x240], hdr2.taille, 0)   le PAYLOAD
      FUN_1424c7b4c(lecteur, session[0x240], taille)
      FUN_142e2bfd0(lecteur, tableau)          <- LE LECTEUR D IMAGE-CLE
```

### 1.2 `FUN_142e2bfd0`, la boucle, bit par bit

Elle remplit un tableau d entrees de **200 octets** (`0xC8` — confirme par `FUN_142e2bb9c`, qui
recopie ces entrees au meme pas), une par entite vivante, et s arrete quand le tableau est plein
ou `DAT_144dbfc90` atteint.

```
FUN_1406d5cc0(lecteur, 3) ; FUN_1408be40c(tableau) ; FUN_1408be3c4(tableau+0x20)
si FUN_1428e1c0c(&DAT_144c23178) > 7 :  DAT_144706104 = R(1)      <- LE PREFIXE DE 1 BIT
extra = FUN_14076cea8()                                              (HasExtraFields)
tant que (e != fin && n < DAT_144dbfc90) :
    e[0x00] = R(32)                     l identifiant (eid)
    e[0x04] = R(32)                     L ARCHETYPE, MOT PLEIN DE 32 BITS
    e[0x0c] = R(32)
    e[0x08] = R(4)                      (FUN_142e29cf8, lecteur de 4 bits)
    e[0x09] = R(8)                                            = 108 bits d en-tete
    si e[0x04] != 0xffffffff :
        desc = *(DAT_144e61d88 + 8 + ti*8)
        n1 = R(32) ; si n1 > 0 :
            FUN_142e31de8(ti, &e[0x94], &e[0x98])             ; tampon d etat par defaut
            desc->vtable[0x60](e[0x94], e[0x98], lecteur, 0)  ; L ETAT PAR DEFAUT
            si extra : R(32)                                  ; mot de controle
        n2 = R(32) ; si n2 > 0 :
            FUN_142e31e70(ti, &e[0xc0], &e[0xb8])             ; tampon de composants
            desc->vtable[0x88](e[0x94], e[0x98], e[0xc0], e[0xb8])   ; AUCUN BIT
            FUN_1428e2b68(&DAT_144c23178, lecteur, ti, eid, e[0xb8])
    e += 0xC8
```

`FUN_1428e2b68` prend le descripteur et la table `lVar4 + 8 + ti*0x4100` (`lVar4` = `session+0x108`
ou `session+0x120+0x130` selon `FUN_1428e1e94`), puis `FUN_142e2c690(desc+1, lecteur, args, table)` :

```
pour k de 0 a 0x3f, table += 0x104 :
    si table[0] != 0 :                     ; une entree NOMMEE
        deser = celui des 64 dont vtable[8]() rend le meme nom
        deser->vtable[0x28](lecteur, args, &prediction=0, *(u32*)(table + 0x100))
        si extra et R(1) : R(32)            ; sentinelle par composant
```

**0x104 octets par entree, 64 entrees, `0x4100` par archetype : c est EXACTEMENT le cadrage du
registre du film** (`registry.go`, lot 1.2). Et tout ce bloc est deja porte par
`WalkKeyframeFullState` (`keyframe_fullstate_loop.go`, lot 1.4). **Le corps d un record
d image-cle etait donc deja lu juste ; ce qui manquait etait la MARCHE.**

### 1.3 L en-tete de 64 bits du depot etait un modele faux

Les 32 bits a `q+32` SONT l archetype — `FUN_142e2bfd0` s en sert tel quel pour indexer
`DAT_144e61d88 + 8 + ti*8`, et un mot >= 50 y ferait deriver le jeu sur un descripteur hors
table. Le « champ de 26 bits de semantique non etablie » n existe pas, et **l hypothese H1 du lot
R5 — « le balayeur saute les records dont `Field26` n est pas nul » — est REFUTEE PAR
L ECRIVAIN** : de tels records ne peuvent pas exister. La seule valeur hors table admise est
`0xffffffff` = PAS D ARCHETYPE, et l entree s arrete alors a ses 108 bits
(`if (puVar12[1] != 0xffffffff)`).

### 1.4 La marche, et ce qu elle mesure

`WalkKeyframeRecords` enchainait par `TraverseEntity` a `+58` — le cadre du record NEW du chemin
DELTA (R(6) d archetype, etat par defaut, PORTE, MASQUE). Il repartait 44 bits trop tot, au milieu
du premier corps : d ou l arret « en-tete-invalide » apres UN record sur les deux temoins. Il
enchaine desormais par `WalkKeyframeFullState`.

Mesure (`TestMarche520`) :

| film | chunk | payload | fenetre 120k | SANS fenetre | marche deterministe |
|---|---:|---:|---|---|---|
| `dad793c7` | 1 | 1 028 032 b | 123 ancres, 13,6 % | **157**, 27,8 % | 2 records, `i10 tacmap-mapdismissallock` |
| `dad793c7` | 2 | 1 043 848 b | 187, 29,1 % | **127**, 29,1 % | idem |
| `bfecd02b` | 1 | 1 318 136 b | 424, 45,7 % | **397**, **45,7 %** | idem |
| `bfecd02b` | 27 | 1 343 112 b | 454, 55,3 % | **158**, **55,3 %** | idem |

**1 record -> 2 records**, et l arret n est plus un cadre faux mais un composant NOMME. Il manque
a la marche les deserialiseurs du lot 3.6 : un record d image-cle porte TOUS les composants de son
archetype sans masque (fermeture mesuree : 30,8 %, `keyframe_closure.golden`).

### 1.5 D2 (5.16) EST REFUTE SUR LE FILM DENSE : LA FENETRE N EST PAS LA CAUSE

Sur `bfecd02b`, retirer la fenetre de 120 000 bits laisse le bit d arret **IDENTIQUE sur les
27 chunks** (45,7 % ... 55,8 %) et ne fait que PERDRE des ancres (12 688 -> 4 676 sur le film).
Le balayeur ne s arrete donc pas parce qu il a epuise sa fenetre : il s arrete parce qu aucune
position du reste du payload ne porte un slot strictement plus grand. Sur `dad793c7` le retrait
gagne au chunk 1 (123 -> 157) et perd aux chunks 2 a 5 (187 -> 127), `betterThan` elisant un
candidat lointain qui deraille la chaine.

**Echanger une heuristique contre une autre n est pas lire la grammaire.** La constante reste,
NOMMEE, DATEE et gagee : `kfScanFenetreBits`, retiree avec le balayeur le jour ou
`KeyframeClosure` atteint 100 %.

---

## 2. L IMAGE-CLE ENTIERE NE PEUT PAS ETRE LA SOURCE — MESURE A 0,0 % (5.20.2)

`TestNaissance520` cherche chaque slot rejete **a toutes les positions de bit du payload
d image-cle du MEME chunk**, sans contrainte de croissance, sans fenetre, en acceptant l entree
sans archetype (invisible au filtre fort du balayeur). C est la BORNE HAUTE de ce que l image-cle
du chunk peut declarer.

| ou le slot rejete est-il declare ? | rejets | part |
|---|---:|---:|
| **ancre par le balayeur, MEME chunk** | **0** | **0,0 %** |
| candidat a position libre du MEME chunk, non ancre | 237 | 1,0 % |
| **ancre par le chunk SUIVANT** | **17 431** | **74,7 %** |
| nulle part (queue de cascade) | 5 657 | 24,3 % |

> **Une lecture PARFAITE et COMPLETE de la table d image-cle ne fermerait pas un seul paquet de
> plus.** Le slot rejete n est pas dans l image-cle de son chunk ; il est dans celle du SUIVANT.
> L entite nait entre deux images-cles, et 74,7 % des rejets le prouvent par la source qui la
> declare enfin.

Le gate (ii) est donc INCHANGE, et c etait previsible avant de le jouer : `dad793c7`
5 354/5 365 · 2 debordements · records 5 641 · `ti=35` 75, 0 desync · rejets hors datum 2 ·
datums 54 ; `bfecd02b` 2 884/30 387 · 32 debordements · records 176 786 · `ti=35` 129 572,
4 desyncs · rejets hors datum 23 769 · datums 10. Chiffre pour chiffre le tableau du 5.16.4.

---

## 3. LA NAISSANCE : DEUX SOURCES POSSIBLES, ET LE FILM EN PORTE UNE TROISIEME QU ON NE LIT PAS

### 3.1 L en-tete rejete N EST PAS une structure — il est un DELTA de bipede

Le 5.19 avait dumpe six temoins et lu « prefixe 1, slot 1792 (0x700), tag 1 » constant. Mesure sur
**les 23 325 rejets** (`TestEntete520`, `bfecd02b`) :

| (b) le triplet lu | rejets | part |
|---|---:|---:|
| `prefixe 1 · low 543 · tag 1` | 1 014 | 4,3 % |
| `prefixe 1 · low 539 · tag 1` | 974 | 4,2 % |
| `prefixe 1 · low 556 · tag 1` | 971 | 4,2 % |
| … 595 classes, TOUTES dans la bande 521-601, `tag 1` | | |

Les 48 bits bruts comptent **2 460 classes** et la distance a la fin du payload **2 194 classes** :
rien n est constant. **« slot 1792 tag 1 » etait un artefact des six temoins, pas une structure.**
L en-tete rejete est un en-tete de record DELTA bien forme, sur un slot de bipede REEL
(bande 521-601 = les bipedes de joueurs, 5.19 §5.1) et de generation 1. Ce que la garde rejette
n est pas une lecture fausse : c est une entite que le monde hors ligne ne connait pas encore.

### 3.2 Le lecteur d identifiant de la branche vive, bit par bit — AUCUN bit de classe 1

`FUN_1406cd128` (branche `DAT_14474cd78 != 0`), par record :

```
si extra : R(32)                      ; le mot de controle par record
b = R(1)
si b == 1 : type = 3 (DELTA)
sinon     : type = R(2) ; si type > 3 -> erreur ; si type == 0 -> FIN DE LISTE
base = DAT_1451f9908 ; max = DAT_1451f990c          ; la CLASSE 7
si DAT_144706104 == 0 : base = 0 ; max = DAT_144706100
W = FUN_1406d310c(max)                               ; ceil'(max)
si W >= 1 : idLow = base + R(W)
tag = R(2)
id = tag << 30 | idLow
FUN_1406cbaa0(type, id, monde+0x38, monde, param_2, lecteur, drapeauVueB)
```

**Il n y a PAS de bit supplementaire de classe 1 dans le flux de trame** : la classe 1 (et son bit
`FUN_1405d5d08(eid) ∈ {0x23,0x28}`, D4 du 5.16) n est ecrite que par `FUN_1406d5110`, et la boucle
de records lit TOUJOURS la classe 7, quel que soit le type. Le port est conforme.

`DAT_144706100` vaut `0x1fff` dans l image (13 bits) et n est reecrit par `FUN_1408f1618` que
lorsque la table de datums GRANDIT (`monde+0x138` et `monde+0x158` leves) ; en rejeu la table est
pre-dimensionnee, la largeur reste 13, et le gate le confirme.

### 3.3 QUI ALIMENTE LA TABLE DE DATUMS — LA REPONSE EST FERMEE, ELLE N A QUE DEUX ENTREES

| feeder | appelants (TOUS) | ce que cela veut dire |
|---|---|---|
| `FUN_1408f1314` | **`FUN_1406cbaa0` et lui seul**, dans la branche `param_1 == 1` | un record de **type 1 (NEW)** de la boucle de records — n importe laquelle des trois vues |
| `FUN_1408f1618` | `FUN_1408f1314` · `FUN_142f2f73c` (vtable `0x48`, l application d une entree d image-cle) | la pose d image-cle |

**Il n existe AUCUNE troisieme source.** Une entite entre dans la table de datums soit par la pose
d une image-cle, soit par un record `NEW` du flux de trame. Le port lie les deux
(`corpsDeRecordNeuf` -> `World.BindFull`, `BindImageCle`, `LierTableDeDatums`).

Et `FUN_1406cbaa0` fait GRANDIR la table pour TOUT type de record, avant toute lecture de corps :

```
si (tailleDeTable <= slot && monde[0x50] && monde[0x70]) :
    FUN_1411b3c84(monde+0x38, max(0x1fff, slot+1))     ; agrandir
    pour chaque entree neuve : FUN_1408f15c8(entree)   ; initialiser
```

C est la signature d allocateur que le 5.19 §5.5 avait mesuree (« premier slot rejete = slot max
de l image-cle + 1 »).

### 3.4 LA SOURCE MANQUANTE, NOMMEE ET MESUREE : LE BLOC DE TYPE 1 EST LA TABLE DE DATUMS

`FUN_1428e2a9c` lit DEUX blocs, et le PREMIER est `FUN_1429883ec` :

```
FUN_1429883ec(session, enTete, table) :
    FUN_1429907c8(tampon, (table[5]-table[4] & ~0x1f) + 0x14 + ((table[1]-table[0])/0x18)*0x18)
    FUN_142988338(session, tampon, enTete.taille, 0)       ; LE BLOC ENTIER
    FUN_1424c7b4c(lecteur, tampon, taille) ; FUN_1406d5cc0(lecteur, 3)
    pour e de table[0] a table[1], pas 0x18, borne par DAT_144706100 :
        FUN_14297ea84(lecteur, ?, e)          ; un premier champ
        R(8)
        e[0x04] = R(32)                       ; <- L ARCHETYPE (FUN_142f30610 le lit en
                                              ;    `+4 + slot*0x18`, FUN_1408f1618 l ecrit)
        FUN_140e74e6c(lecteur, ?, e+0x08)     ; un second champ
    puis, sur table[4]..table[5] : 0x100 bits de R(1) par element (un BITMAP)
```

`table` est le tableau de pas **0x18** indexe par slot — **la table de datums elle-meme**
(`monde+0x120`).

**Mesure hors ligne (`TestBlocAvantImageCle520`), et elle est nette :**

| | `dad793c7` | `bfecd02b` |
|---|---:|---:|
| type du bloc qui precede l image-cle | **1**, dans **5 chunks sur 5** | **1**, dans **27 chunks sur 27** |
| taille de ce bloc | **343 019 o**, CONSTANTE | **343 019 o**, CONSTANTE |
| total type 1 | 1 715 095 o = 5 x 343 019 | 9 261 513 o = 27 x 343 019 |
| total type 2 (image-cle) | 650 107 o | 4 593 116 o |

343 019 octets = 2 744 152 bits pour **8 191 entrees** (`DAT_144706100` = `0x1fff`) : une table de
taille FIXE, **deux fois plus grosse que l image-cle**, presente une fois par chunk, juste avant
elle — et que le depot ne lit dans AUCUNE de ses marches.

**Le 5.16 §2.2 (c) l avait ecarte sur une lecture partielle** : « type 1 : lit 16 octets et avance
un compteur — le bloc est SAUTE ». C est vrai de `FUN_142989418`, le handler du repartiteur de
LECTURE COURANTE ; ce n est pas vrai du chemin de CHARGEMENT D ETAT, qui le lit par
`FUN_1429883ec`. Un bloc que la pompe saute n est pas un bloc que le jeu ignore.

### 3.5 CE QUI RESTE, PAR ADRESSE

| maillon | adresse | ce qu il ouvre |
|---|---|---|
| **la grammaire d une entree de la table de datums** | `FUN_1429883ec` : `FUN_14297ea84` (premier champ), `R(8)`, `R(32)` archetype, `FUN_140e74e6c` (second champ), puis le bitmap de `0x100` bits par element de `table[4]..table[5]` | **la source de `slot -> archetype` pour tout le chunk**, independante de la liste d entites de l image-cle : 343 019 o par chunk, jamais lus |
| la fermeture des records d image-cle | `keyframe_closure.golden` (30,8 %), lot 3.6 | la marche deterministe de bout en bout, donc le retrait du balayeur et de `kfScanFenetreBits` |
| l historique de baseline | D3 (5.16) | 14 records sur 174 606, VALEURS seulement |

---

## 4. CE QUE LE TROU PORTAIT, EN CLAIR (5.20.4)

**RIEN N EST NOUVELLEMENT LU.** Le diff de production du lot est la MARCHE d image-cle (aucun
appelant de production) et le predicat `readKeyframeHeader` (rendu STRICT). Aucun composant n a
ete porte, aucune largeur n a bouge, aucun archetype ne change de compte :

- `bfecd02b` : records 176 786, `ti=35` 129 572, desyncs 4, 207 etiquettes de composant ;
- `dad793c7` : records 5 641, `ti=35` 75, desyncs 0.

**Aucun canal d etat de bipede n apparait, donc AUCUNE MONTEE DE SCHEMA** : `replay.SchemaVersion`
reste a **67** — pas de chronique v68, pas de fixture `replay_schema_68_*`, pas de jumeaux, pas de
zod, pas d OpenAPI. `grammar.Rev` monte a `grammar-2026-09-22.8` (l en-tete d image-cle et le
predicat changent). `facts.Rev` NE MONTE PAS : `killsource/` marche par `DecodeFrameRecords` et
`WalkKeyframeWorld`, tous deux inchanges — **aucun backlog killsource n est ouvert**.

**ET LE RESIDU RESTE, AVEC SA CAUSE NOMMEE PAR UNE ADRESSE** : `bfecd02b` 2 884/30 387. Les
23 325 rejets portent sur des entites nees entre deux images-cles ; l image-cle du chunk ne les
porte a AUCUNE position (0,0 %, §2) ; la table de datums que le film dumpe une fois par chunk dans
son bloc de **type 1** (`FUN_1429883ec`, 343 019 octets) n est pas lue. C est la, et pas ailleurs,
que le prochain lot doit aller.
