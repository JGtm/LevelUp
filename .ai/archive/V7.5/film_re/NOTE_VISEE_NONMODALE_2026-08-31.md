# NOTE — Visée des tirs NON-MODAUX (ceux qui touchent) — 2026-08-31

> Suite de `NOTE_VISEE_TIR_2026-08-31.md` (qui a percé le cas MODAL : tir propre, 0 cible /
> 0 composante, visée à post-comptes + 2). Objectif ici : le cas NON-MODAL — les tirs qui
> portent ≥ 1 cible ou ≥ 1 composante de dégât, où deux boucles de longueur variable
> s'insèrent avant la visée. Instrument : `filmdec/lot1_visee_nonmodale_research_test.go`
> (garde `LOT1_TRAME_FILM`). Ghidra : `FUN_14080C1F8` tracé instruction par instruction.

## Résultat en une phrase

La **grammaire des deux boucles + des deux composites est entièrement percée et décodable
hors ligne** (ce qui **réfute** la réserve « boucles cibles/composantes = largeur runtime »
de la note modale) — MAIS l'oracle de concentration ne peut **pas valider** la visée
non-modale ainsi placée : à la position analogue au modal, elle reste **au niveau du bruit**
(28-34 % vs modal 77-87 %). Le décode existe ; sa correction n'est pas prouvable hors ligne
avec l'oracle disponible.

## La grammaire (FUN_14080C1F8, param_5 = 0 confirmé pour le film)

`param_5 == 0` en mode film est CONFIRMÉ : le chemin modal appelle `FUN_1406cd5b8`, qui est
la branche `param_5==0` (à 0x14080c80b). Ordre réel de la charge, après les comptes :

```
comptes  FUN_14080cc68 : cibles P (@0xf8) lu EN PREMIER, composantes N (@0x34) en second
         (grammaire = celle du décodeur modal, inchangée)

BOUCLE COMPOSANTES  (N itérations, @0x34)
   par composante :  R(2) + R(1) + R(32)                          = 35 bits FIXES
   le R(2) est mémorisé (component[i].R2) et RELU par la boucle cibles

BOUCLE CIBLES  (P itérations, @0xf8) ;  mode = 12 si P==1, sinon 4  (local_98)
   par cible :  R(4) + R(1)[hit]
   si hit == 1 :
       R(3)                              (FUN_1406d310c(6) = 3 : bits pour représenter 6)
       (N < 3 ? R(1) : R(4)) -> idx      (idx = index de composante visée)
       R(16)
       3 * W bits                        (FUN_140c1e924 -> FUN_140c1e9d4 : 3 champs de W bits)
           W = FUN_14102bd24(mode, component[idx].R2)
             = min(mode, 6)  si component[idx].R2 == 1
             = mode          sinon
           (component[idx].R2 = 0 si idx >= N : mémoire memset 0)

COMPOSITES  FUN_1406cd5b8 puis FUN_1408eff64   (bit-exact, cf. ci-dessous)
VISÉE  R(30)   (cubemap 6 faces, DecodeAimVectorChecked)
```

### Les deux ambiguïtés ouvertes du relevé initial, TRANCHÉES

- **param_5** = 0 (film) → la boucle composantes lit **R(32)**, pas une référence de domaine.
- **largeur de `FUN_140c1e924`** = **3 × W**, W issu de la table **PURE** `FUN_14102bd24`
  (`b==1 → min(mode,6) ; sinon → mode`). **Aucune dépendance runtime** : les deux boucles
  sont ENTIÈREMENT décodables hors ligne. (`FUN_1406d310c(6)` = 3, une largeur constante.)

### Les composites (bit-exact, vérifiés au désassemblage)

```
cd5b8 :  A=R(1) ; B=R(1)
         si B : FUN_140c9eabc [R(1) ; si1: tag=R(2) ; tag1: R(32)+[R(1)?R(6)] ; tag2: R(32)]
                si A : R(4)+R(4)            (FUN_1406d84b4, largeur 4, ×2)
                flags = R(3)               (FUN_1407ef8e4)
                si flags&2 : R(1)+[si0: R(20)+R(14)]   (FUN_140c9e738 -> FUN_14076d528, 20/14)
         si A : C=R(1) ; si C : R(5)
eff64 :  main=R(1) ; si main : tag=R(2) ; tag1: R(32)+[R(1)?R(6)] ; tag2: R(32)
```

`lot1SkipCd5b8` / `lot1SkipEff64` (déjà au dépôt) reproduisent exactement ces grammaires.
À vide : cd5b8 = 2 bits, eff64 = 1 bit. Sur un record avec dégât, elles consomment 3 à
**~92 bits** (mesuré) selon les tags — d'où une **longueur très variable** avant la visée.

## Ce qui EMPÊCHE de valider la visée non-modale

1. **Impossible d'isoler une boucle empiriquement.** Sur les trois films témoins, la
   distribution `(N composantes, P cibles)` est binaire : `(0,0)` (raté = modal) et `(1,1)`
   (touché) — **0 record composante-seule, 0 record cible-seule**. On ne peut donc pas
   calibrer une boucle indépendamment de l'autre : tout tir qui touche est exactement
   « 1 cible + 1 composante » (physique du hitscan).

2. **L'oracle de concentration est aveugle ici.** Deux raisons, additionnées :
   - **Validité cubemap non discriminante à 30 bits** : `faceSize(30) = floor(2^30/6)`, donc
     `face = code/faceSize < 6` pour quasi tout code 30 bits → `DecodeAimVectorChecked`
     renvoie presque toujours `ok` → aucun cliquet d'alignement, seule la concentration juge.
   - **Faux positifs sur les champs à faible entropie** : un `R(32)` de petite valeur décode
     en `face 0 → (1, petit, petit)` → un axe « saturé » ARTIFICIEL. Le modal lui-même a des
     pics parasites hors d=2 (d=-6 / d=-2 : champs de direction de l'en-tête, 86-100 %), et
     le `R(32)` de composante concentre un axe au-dessus du contrôle. L'oracle « voit » de la
     direction partout dans cette zone dense.
   - **Le tir qui touche vise à des élévations variées** (pas le biais horizontal du tir
     raté) : une direction unitaire uniforme donne `E|x|=E|y|=E|z|=0.5`, **exactement comme
     du bruit**. Même à la BONNE position, l'oracle ne saurait la distinguer.

## Mesures (instrument, 12 premiers chunks par film)

Visée lue à `après-boucles + 2` (analogue exact du modal `post-comptes + 2` ; pour le modal
les boucles sont vides). Concentration = part<0.3 de l'axe le plus concentré (une vraie visée
horizontale sature un axe ; bruit ~26 %).

| Film | modal (0,0) @d=2 | non-modal (1,1) @d=2 | contrôle | scan 0..220 (meilleur) |
|---|---|---|---|---|
| 000d5950 | **83 %** (n=210) | 34 % (n=35) | 25 % | 69 % (n=35, non signif.) |
| 01e1f945 | **77 %** (n=491) | 28 % (n=376) | 44 % | 47 % (n=376) |
| 00502e52 | **87 %** (n=218) | 29 % (n=58) | 28 % | 53 % (n=58) |

- **L'oracle FONCTIONNE sur le modal** (77-87 %, `E|x|` ~0.2, axe saturé) — c'est le témoin
  positif : la chaîne en-tête + comptes + visée@+2 est bonne.
- **Le non-modal reste au bruit** : `E|x|=E|y|=E|z| ~= 0.5` à d=2 ; le maximum sur toute la
  fenêtre balayée (−6..+6, puis 0..220 depuis après-boucles) ne dépasse jamais nettement le
  contrôle et s'explique par les champs structurés (faux positifs). **Couverture par
  l'oracle : 0 / (35+376+58) tirs non-modaux.**

## Verdict (garde-fou respecté : rien de survendu)

- **Acquis solide** : la grammaire des deux boucles ET des composites est percée au bit près
  et décodable **hors ligne** (réserve « runtime-width » de la note modale **réfutée**). Un
  décodeur non-modal peut désormais AVANCER jusqu'à la visée.
- **Non acquis** : la CORRECTION de la visée non-modale n'est **pas prouvable** hors ligne
  avec l'oracle de concentration — il est structurellement aveugle aux visées non
  horizontales et bruité par les champs de direction denses du record. Le non-modal reste
  donc **non validable** hors ligne, avec les chiffres ci-dessus.
- **Piste pour un futur oracle** (hors périmètre de ce lot) : un oracle de **profondeur de
  trame** (décoder la queue du record + la continuation, maximiser la profondeur — comme
  `TestLot1ViseeCalibration` pour le modal) validerait la POSITION indépendamment de
  l'orientation de la visée. C'est le seul moyen de trancher « décode juste + visées
  uniformes » contre « reste un bit d'écart quelque part ».

## Fichiers

- `apps/go-api/internal/analysis/filmdec/lot1_visee_nonmodale_research_test.go` (nouveau,
  garde `LOT1_TRAME_FILM`) : `nmHeaderCounts`, `nmLoopBits` (grammaire des boucles),
  `nmAfterComposites`, + `TestLot1ViseeNonModale` / `TestLot1ViseeNonModaleScan`.
- Rien de branché en production : la visée non-modale n'est pas posée sur la carte (elle
  n'est pas validée). `fire_aim_modal.go` (modal) reste le seul chemin câblé.
