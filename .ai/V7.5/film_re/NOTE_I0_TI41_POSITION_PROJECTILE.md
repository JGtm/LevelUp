# NOTE DE TRAVAIL — `object-position-component` de `ti=41` (le PROJECTILE), i0

> Ouverte le 2026-08-08, branche `research/v75-precision`. **Travail de DÉCODEUR**, pas de
> piste E : celle-ci est close (verdict négatif, `.ai/V7.5/VERDICT_PRECISION_PROJECTILES.md`).
> Ce qui l'a fait ouvrir : la seule voie restante vers l'attribution d'une touche de projectile
> est de rattacher l'**entité** projectile à un joueur par sa **trajectoire**, et la trajectoire
> exige que i0 de `ti=41` soit bit-exact. i0 précède tous les autres composants : un i0 inexact
> ferme l'accès à tout l'archétype.
>
> **Note tenue au fil de l'eau.** Ce qui est écrit ici est ce qui est lu ; ce qui reste à lire
> est en §4.

---

## 1. L'ÉTAT RÉEL, ET IL EST PLUS AVANCÉ QUE CE QUE DIT L'INDEX

L'index (`ETAT_DE_L_ART_KILLWEAPON.md`, §2.1 et la ligne « Ne PAS confondre les deux `i0` »)
présente `FUN_14076e29c` comme le bloqueur, « non bit-exact, 45 / 60 bits ». **Vérification sur
pièces : le composant est implémenté dans `filmdec/traverse.go`** (cas
`object-position-component`), et son découpage basse précision a été **corrigé le 2026-07-26** —
`R(1)` d'index puis **13/13/14** (et non `R(2)` puis 13/13/13), frontières placées à 16/29/43 par
un profil de bascule sur 6 794 paires de records consécutifs.

**Le « 45 / 60 » n'est donc pas « on lit 45 bits là où il en faut 60 ».** Ce sont **deux
branches** du même composant, et le port en modélise deux :

```go
if br.ReadBit() {          // "precHigh"
    br.ReadBits(59)        // <- 59 bits OPAQUES, decrits comme "AABB + handle-tail + R(2)"
} else {
    if !br.ReadBit() { br.ReadBits(IndexW) }
    for a := 0; a < 3; a++ { br.ReadBits(AxisW[a]) }   // 13/13/14 sur Cliffhanger
    br.ReadBits(2)
}
```

**Le point faible est le `ReadBits(59)`** : c'est un TOTAL MESURÉ, pas une décomposition. Tant
qu'il est juste en longueur la traversée reste alignée — mais **aucune position n'en sort**, et
c'est précisément la position qu'il faut pour une trajectoire.

---

## 2. CE QUE LE DÉSASSEMBLAGE DIT — TROIS BRANCHES, PAS DEUX

```c
FUN_14076e29c(br, ctx) {
    st = *(ctx + 0x10);
    FUN_14076e420(br, st + 4, 0x10);        // 0x10 = NIVEAU DE PRECISION L = 16
    FUN_14076e3e4(br, st);                  // handle-tail
    if (FUN_140492128(st + 4))              // predicat SUR LA POSITION DECODEE
        FUN_14076e304(br, st + 0x10);       // R(2) "finite"
}

FUN_14076e420(br, out, L) {
    g = FUN_1406cf008();                     // R(1)
    range = g ? &DAT_143b8c6d0 : NULL;       // la porte choisit une PLAGE, pas un layout
    FUN_14076e494(br, out, L, ..., range);
    return g;
}

FUN_14076e494(br, out, L, ..., range) {
    c = FUN_14076f91c(br, out, L, L);        // <- PREMIER lecteur, rend un char
    if (c == 0) {
        if (range == 0) FUN_14076e524(&pos, br, tmp);   // <- vec3 quantifie, plage de la CARTE
        else            FUN_141f85880();                // <- plage PAR DEFAUT
    } else {
        FUN_1411b259c(&pos, br);                        // <- TROISIEME chemin
    }
}
```

### 2.1 Les trois écarts avec le port Go — `[MESURE]` sur le désassemblage

1. ~~**Il y a TROIS encodages de position, le port en modélise DEUX.**~~ **RETIRÉ le
   2026-08-08, une heure après avoir été écrit** — `FUN_14076f91c` **ne lit AUCUN bit** :

   ```c
   undefined1 FUN_14076f91c(void) {
       uVar1 = 1;
       if ((DAT_144e61ea0 == '\0') && (DAT_145121140 != '\x01')) uVar1 = 0;
       return uVar1;
   }
   ```

   C'est un test de **deux globales**, pas un champ du flux. Le troisième chemin
   (`FUN_1411b259c`) est donc derrière une porte de mode runtime, et le port qui l'ignore a
   raison de l'ignorer : il fonctionne sur les films réels, donc `FUN_14076f91c` y rend 0.
   **Le port n'a pas un défaut de moins — il n'en avait pas ici.**

   > **TROISIÈME OCCURRENCE DU MÊME MOTIF, et ça devient une règle de lecture de ce moteur.**
   > La porte du tag des codes 6/7 (`b0 == 1`, 168 380 observations), la porte
   > `DAT_1451222c8` des stats 0x0D / 0x0E, et maintenant celle-ci. **Ce binaire est truffé de
   > chemins conditionnés par des globales qui ne s'ouvrent jamais en jeu.** Lire un décompilé
   > sans vérifier ses portes conduit à modéliser des branches qui ne s'exécutent pas —
   > c'est-à-dire à inventer du travail. **Règle : pour toute branche vue dans un décompilé,
   > établir si sa condition vient du BITSTREAM ou d'une GLOBALE avant de la porter.**
2. **La porte R(1) n'est pas un « precHigh » qui changerait le layout : elle choisit une
   PLAGE** (`DAT_143b8c6d0` contre la plage de la carte). Or la largeur d'axe **dérive de
   l'étendue** — `W = min(26, ceilLog2(ceil(60 * extent)))` à L = 16. Une plage plus large donne
   des axes plus larges. **« 45 contre 60 » s'explique donc sans changement de grammaire : c'est
   le même vec3 lu avec des largeurs différentes.** Si cette lecture se confirme, la position du
   projectile est récupérable **sur les deux branches**, et le `ReadBits(59)` opaque devient
   décomposable.
3. ~~**`FUN_14076e304` (le R(2)) est CONDITIONNEL.**~~ **RETIRÉ — le prédicat est
   `isfinite(vec3)`** : `FUN_140492128` teste les bits d'exposant (`& 0x7f800000 != 0x7f800000`)
   des trois flottants et ne rend 1 que si les trois sont finis. Or un vec3 **déquantifié dans
   une plage bornée est toujours fini** (cf. la formule de §2.2). Le port a donc raison de lire
   le `R(2)` inconditionnellement. **Deuxième écart qui se retire à la lecture de sa propre
   garde** — la règle de l'écart 1 se confirme.
4. **`FUN_14076e3e4` (handle-tail) est appelé INCONDITIONNELLEMENT**, pas seulement sur la
   branche haute. Il ne consomme des bits que si son paramètre de garde est non nul (sinon il
   pose des sentinelles `0xffffffff` sans rien lire), mais **cette garde n'est pas la porte R(1)**
   du port.

### 2.2 LA GRAMMAIRE DE LA BRANCHE PAR DÉFAUT — établie, mais ses LARGEURS ne sont pas statiques

`FUN_141f85880` enchaîne trois appels, et les deux qui comptent sont lisibles :

```c
FUN_140be9b88(L, aabb /* float[6] : minX,maxX,minY,maxY,minZ,maxZ */, _, W /* out int[3] */) {
    extent[a] = aabb[2a+1] - aabb[2a];
    step = FUN_140be9c78();                       // <- SANS ARGUMENT : une globale de runtime
    if (step < EPS) { W[0..2] = 0x1a; }           // 26, le plafond
    else for a in 0..2:
        n    = ceilf(extent[a] / (2*step));       // sature a 0x400000 si l extent est trop grand
        W[a] = min(FUN_1406d310c(n), 0x1a);       // bitLen, plafonne a 26
}

FUN_1406d7f18(raw, aabb, W, out) {                // la DEQUANTIFICATION
    scale  = (aabb[2a+1] - aabb[2a]) / (1 << W[a]);
    out[a] = raw[a] * scale + aabb[2a] + scale * 0.5;
}
```

**C'est la forme fermée du dossier, confirmée dans le binaire** — `W = min(26, bitLen(ceil(extent /
(2·step))))`, plafond `0x1a`. Trois précisions neuves :

- la déquantification divise par `1 << W`, **pas** par `2^W - 1`, et ajoute un **demi-pas** ;
- **`FUN_140be9b88` n'utilise PAS le `L` qu'on lui passe** pour la largeur : il l'écrit dans
  `W[0..2]` puis l'écrase dans les deux branches. Le pas vient de `FUN_140be9c78()`, **sans
  argument** — donc d'un état global posé au chargement de carte ;
- la lecture des bits n'est pas dans `FUN_1406d7f18` (qui ne fait que déquantifier) mais dans
  `FUN_1424cbed4(_, br, W)`, qui lit trois valeurs de largeurs `W[0..2]`.

**LES DEUX PLAGES, DÉCODÉES OCTET PAR OCTET — et ce ne sont pas les mêmes :**

```
DAT_143b8c6d0  00 00 C8 C2 | 00 00 C8 42   x3   ->  -100.0 .. +100.0   <- LA PLAGE DE CETTE BRANCHE
DAT_143b8c6b8  00 40 9C C6 | 00 40 9C 46   x3   ->  -20000.0 .. +20000.0  <- celle citee par le dossier
```

Le piège annoncé en §4 est donc réel : **la plage de la branche par défaut de `ti=41` est
± 100, pas ± 20000.** Deux entrées voisines de la même table, à 0x18 (un AABB) d'écart.

**CONSÉQUENCE, ET ELLE TRANCHE LA SUITE.** Le pas venant d'une globale de runtime, **les largeurs
ne sont PAS dérivables statiquement** : l'hypothèse « 59 = 3 x 19 + 2 » ne peut ni se confirmer
ni s'infirmer dans Ghidra. À ± 100 et au pas `q(16) = 1/120` elle donnerait 14 bits par axe
(soit 44, pas 59) — donc **le pas réel n'est pas `q(16)`**, et il n'y a rien à en déduire de
plus au désassembleur. **La largeur se MESURE sur le film**, exactement comme
`DetectI0Layout` le fait déjà pour l'autre `i0` : profil de bascule bit à bit, sans aucun a
priori de largeur. C'est la fin de la phase RE.

> ⚠ **STATUT : `[MESURE]` sur le décompilé, RIEN N'EST VÉRIFIÉ SUR LE FILM.** La règle du
> chantier s'applique intégralement — *Ghidra nomme, le film mesure*. Les points 3 et 4 en
> particulier prédisent des **décalages de longueur observables** : ils se testent par le profil
> de bascule et par le chaînage sur i1 (`object-translational-velocity`, `[1][1][19][10]`), le
> même critère qui avait départagé les lectures rivales en 2026-07-26. **Aucun de ces quatre
> points ne doit être porté dans `traverse.go` avant ce test** — le composant est sur le chemin
> de tous les archétypes qui le portent, et une régression y est silencieuse.

---

## 3. POURQUOI ÇA VAUT LE DÉTOUR

Si la position du projectile sort sur les deux branches, la chaîne devient :

```
entite projectile (repliquee, slot != -1)  ->  ses positions successives
   -> sa PREMIERE position  ->  le joueur le plus proche a cet instant  ->  le TIREUR
   -> et l arme suit, par le record de tir de ce joueur a cet instant
```

C'est **offline-pur**, **universel**, et cela n'utilise **ni** un champ de propriétaire (fermé,
7ter.88 (6)) **ni** un appariement d'horloge (le motif « same-clock », formellement invalidé sur
ce chantier). C'est une troisième voie, et c'est la dernière.

**Ce que ça ne résout pas** : le plafond de validation reste entier (pas de population mono-arme
pour le Needler — la validation passerait par le contraste intra-joueur).

---

## 4. CE QUI RESTE À LIRE — la liste, dans l'ordre

- [x] `FUN_14076f91c` — **ne lit aucun bit**, test de deux globales. Écart 1 retiré (§2.1).
- [~] `FUN_1411b259c` — le troisième encodage : **sans objet** tant que la porte ci-dessus est
      fermée sur les films réels (elle l'est, sinon le port actuel ne marcherait pas).
- [x] `FUN_141f85880` — **c'est un vec3 quantifié, pas une structure opaque** :
      `FUN_140be9b88(L, plage, plage, desc)` construit le descripteur de largeurs, puis
      `FUN_1406d7f18(&val, plage, desc, out)` lit. **Donc les 59 bits du port SONT
      décomposables**, et la position du projectile sort **sur les deux branches**.
      **HYPOTHÈSE À TESTER, et elle est arithmétiquement propre : 59 = 3 x 19 + 2** (trois axes
      de 19 bits + le `R(2)` final). À confronter à la forme fermée du dossier
      `W = min(26, bitLen(ceil(span / (2*step))))`, `step(L) = 2^(16-L)/120`, ici **L = 16**
      (câblé au site d'appel : `FUN_14076e420(br, st+4, 0x10)`).
- [x] `DAT_143b8c6d0` — **± 100** (et non ± 20000 : c'est `DAT_143b8c6b8`, entrée voisine).
- [x] `FUN_140be9b88` / `FUN_1406d7f18` — formule de largeur et déquantification établies
      (§2.2). **Les largeurs ne sont PAS dérivables statiquement** : le pas vient d'une globale
      de runtime. L'hypothèse « 3 x 19 + 2 » n'est ni confirmée ni infirmée au désassembleur.
- [x] `FUN_140492128` — c'est `isfinite(vec3)`. Écart 3 retiré.
- [ ] `FUN_14076e3e4` — la garde réelle du handle-tail, et sa largeur (un `0x40 - n < 0xb`
      suggère 11 bits). **Seul écart encore ouvert.**
- [ ] `FUN_14076e524` — la porte `index-sel` / `index de région` de la branche carte (le port la
      modélise déjà ; à confronter seulement si le test film échoue).

**FIN DE LA PHASE RE.** Ce qui reste se mesure, plus se lit :

- [x] **T1 — atteignabilité : POSITIF, mais il ne prouve que la moitié.** Cf. §6.
- [ ] **T2 — largeurs** : profil de bascule bit à bit sur la branche par défaut, sans a priori
      (méthode `DetectI0Layout`).
- [ ] **T3 — chaînage** : vérifier l'alignement sur `i1`
      (`object-translational-velocity` = `[1][1][19][10]`), le critère qui avait départagé les
      lectures rivales le 2026-07-26.
- [ ] **PUIS SEULEMENT** : portage dans `traverse.go`.

---

## 6. T1 — LES ENTITÉS PROJECTILE SONT BIEN DANS LE MONDE RÉPLIQUÉ

Outil : `apps/go-api/cmd/tmp_ti41` (archivé sous
`.ai/V7.5/outillage/precision_projectiles/`). Instrument réutilisé sans réécriture :
`filmdec.WalkKeyframeWorld`, déjà validé 249/250 entités et 8/8 bipèdes.

**12 films, recensement des archétypes du monde de keyframe :**

```
archetype       records   slots distincts
ti=38            31 425          690
ti=6             14 629           48
ti=17            10 065           33
ti=5              9 760           32
ti=14             9 312           74
ti=37             5 825        1 561
ti=43             3 845          194
ti=42             3 805        1 284
ti=41               185          132   <- PROJECTILE, present sur 11 films sur 12
```

**Ce que ça établit** : l'entité projectile **existe bel et bien comme entité répliquée du film**,
avec un slot, sur 11 films sur 12. La voie n'est pas morte — c'était le risque que T1 devait
écarter, et il est écarté.

**Ce que ça n'établit PAS, et il faut le dire aussi net.** 185 records pour ~11 entités
distinctes par film, quand un film Fiesta porte des milliers de tirs de projectile. **Ce n'est
pas un déficit de réplication, c'est un artefact d'échantillonnage** : un keyframe tombe toutes
les ~20 s et un projectile vit une fraction de seconde — on ne capte que ceux en vol à l'instant
du cliché. Le rapport 185 records / 132 slots (la plupart des projectiles n'apparaissent que
dans UN keyframe) est exactement la signature attendue d'une durée de vie courte.

**La trajectoire ne vit donc pas dans les keyframes : elle vit dans le flux DELTA.** Et là, le
test bute sur le mur connu du chantier — un record delta (type-3) **ne porte aucun typeIndex** :
il résout son archétype par le World (`filmdec/world.go`, en-tête). Compter les `ti=41` du flux
delta exige donc le décodeur STATEFUL avec un binding World complet, c'est-à-dire précisément ce
que `.ai/README_KILLWEAPON_INDEX.md` §0bis décrit comme le blocage de fond (« binding World
incomplet », cascading-desync).

> **CONSÉQUENCE DE PÉRIMÈTRE, et elle est structurante.** La voie trajectoire ne dépend pas
> seulement de `i0` de `ti=41` : elle dépend du **binding World** du décodeur stateful. `i0`
> exact est nécessaire, pas suffisant. C'est un chantier de décodeur à part entière, pas un
> correctif de composant — et cette note doit le dire avant que quiconque estime le coût sur la
> seule base du §2.

---

## 5. JOURNAL

- **2026-08-08 (1)** — ouverture. Vérification sur pièces de l'état du port (le « bloqueur » de
  l'index est en partie périmé : le composant est implémenté, la branche basse précision est
  corrigée depuis le 2026-07-26). Désassemblage de la chaîne `FUN_14076e29c` →
  `FUN_14076e420` → `FUN_14076e494` : trois branches d'encodage, quatre écarts avec le port.
- **2026-08-08 (2)** — l'écart 1 **se retire lui-même** : `FUN_14076f91c` ne lit aucun bit, c'est
  une porte de globales. Troisième occurrence du motif « chemin derrière une porte jamais
  ouverte » dans ce binaire — promue en **règle de lecture** (§2.1). Et
  `FUN_141f85880` **n'est pas opaque** : c'est un vec3 quantifié à plage par défaut, donc les
  59 bits du port se décomposent et la position sort sur les deux branches.
  Bilan intermédiaire : 1 écart retiré, 2 restants, 1 gain potentiel.
- **2026-08-08 (3)** — **fin de la phase RE.** L'écart 3 se retire à son tour (`FUN_140492128`
  est `isfinite`, toujours vrai sur un vec3 déquantifié borné). La grammaire de la branche par
  défaut est établie et la formule de largeur du dossier est **confirmée dans le binaire**, avec
  trois précisions neuves (§2.2) et les deux plages décodées octet par octet (± 100 ici,
  ± 20000 pour l'entrée voisine). **Mais le pas de quantification vient d'une globale de
  runtime : les largeurs ne sont pas dérivables statiquement.** L'hypothèse « 3 x 19 + 2 » reste
  donc ouverte et **se mesurera sur le film**, pas au désassembleur.
  **Bilan de la phase RE : sur 4 écarts annoncés contre le port, 3 étaient à moi** — le port
  était juste, et c'est ma lecture des gardes qui ne l'était pas. Le seul écart encore ouvert est
  la garde du handle-tail. **Rien de porté, rien de vérifié sur film.**
- **2026-08-08 (4)** — **T1 joué : POSITIF.** L'entité `ti=41` est bien une entité répliquée du
  film (185 records, 132 slots distincts, 11 films sur 12). Le risque que la voie soit morte
  d'avance est écarté. **Mais la trajectoire vit dans le flux DELTA, pas dans les keyframes**, et
  un record delta ne porte pas de typeIndex : le compter exige le binding World du décodeur
  stateful. **`i0` exact est nécessaire, pas suffisant** — c'est un chantier de décodeur, pas un
  correctif de composant (§6).

---

## 7. T1' — LE FLUX DELTA REND DES TRAJECTOIRES, ET §6 ÉTAIT FAUX

> **CORRECTION DE §6, sur objection de l'utilisateur (« le binding World est normalement déjà
> décodé »).** J'y concluais que la voie dépendait d'un binding World non résolu, en citant
> `README_KILLWEAPON_INDEX.md` §0bis — **un document du 13 juin qui précède la résolution du
> chantier**. C'est la deuxième fois dans la journée que je conclus sur une source périmée.
> Vérification sur pièces : `filmdec` porte `DecodeFrameRecords`, `DecodeFrameInfer`,
> `TryDeltaAt`, `ScanFrameTargets`, `DecodeFrameViews`, `DecodeFrameResync`, et `killsource` les
> pilote en production. **Le binding fonctionne.**

Outil : `apps/go-api/cmd/tmp_ti41d`. Tout est réutilisé — `ParseRegistryChunk`,
`WalkKeyframeWorld` + `World.BindFull` (même règle que `killsource`), et **`DecodeFrameInfer`,
qui INFÈRE l'archétype des entités transitoires absentes du binding** : exactement le cas du
projectile.

```
film       paquets delta   records   ti=41 records   ti=41 slots
000d5950          30 371    37 510            543            49
0014603f          23 381    37 851            110            60
00162144          27 287    41 705            163            68
00502e52          33 177    52 368            263            78
00761d27          26 252    45 440            279            62
008e1bba          10 989    17 016             30            21
--------------------------------------------------------------
6 films                              1 388 records    281 slots
```

**Ce que ça établit : ~4,9 records par entité projectile.** Ce n'est plus une apparition
ponctuelle comme dans les keyframes (185 records / 132 slots = 1,4) — **c'est une suite de
positions successives, c'est-à-dire une trajectoire.** La voie est vivante, et le mur que
j'annonçais en §6 n'existe pas.

**Les deux réserves, et elles sont sérieuses :**

1. **C'est une BORNE INFÉRIEURE.** `DecodeFrameInfer` démarre au bit 0 du payload ; les paquets
   portant une liste d'événements avant la boucle de records désynchronisent tôt — c'est pour
   cela que `killsource` a un localisateur (`locateStrict` + repli), **qui n'est pas exporté**.
   Un `ti=41` trouvé est un vrai `ti=41` ; un film à zéro ne prouverait rien.
2. **`ti=0` domine le recensement : 119 307 records sur 6 535 slots.** C'est le seau des records
   dont l'archétype n'est pas résolu. La marche couvre donc une fraction du flux, et 281
   projectiles sur 6 films est à comparer aux milliers de tirs de projectile qu'ils portent.
   **Couverture partielle, pas nulle.**

**CE QUI DEVIENT LE VRAI BLOQUEUR, ET C'EST BIEN CELUI DU §2.2** : les records sortent, donc
`i0` est consommé — mais sur la branche « plage par défaut » le port avale 59 bits opaques et
**aucune position n'en sort**. C'est T2 (les largeurs, par profil de bascule) qui débloque la
trajectoire, pas le binding. La note revient donc exactement là où §2.2 l'avait laissée, mais
avec le périmètre correct : **un travail de composant, pas un chantier de décodeur.**
- **2026-08-08 (5)** — **T1' joué, et il CORRIGE le §6.** Le binding World fonctionne (le doc que
  je citais datait du 13 juin, avant la résolution du chantier). Sur 6 films, le flux delta rend
  **1 388 records `ti=41` sur 281 slots — ~4,9 records par projectile, donc des trajectoires.**
  Réserves : borne inférieure (pas de localisateur de boucle exporté), et `ti=0` domine
  (couverture partielle). **Le bloqueur redevient T2 — les largeurs de la branche par défaut —
  et le périmètre est celui d'un composant, pas d'un chantier de décodeur.**

---

## 8. T2 ET T3 — DEUX INSTRUMENTS QUI NE MORDENT PAS, ET POURQUOI

### 8.1 T2 — le profil de bascule NE TRANSFÈRE PAS aux projectiles

Outil `apps/go-api/cmd/tmp_i0w`. 8 films, échantillons i0 de `ti=41` : **264 à porte = 0**,
**277 à porte = 1** — les deux branches sont donc bien empruntées, à parts comparables. Le
`ReadBits(59)` opaque concerne **la moitié des records projectile** : ce n'est pas un cas
marginal.

Profil de bascule, 239 paires consécutives d'un même slot (porte = 1) :

```
b00  0.28 0.23 0.21 0.24 0.26 0.27 0.39 0.33 0.37 0.38 0.39 0.38 0.44 0.44 0.34 0.30
b16  0.36 0.36 0.36 0.35 0.38 0.41 0.39 0.41 0.27 0.12 0.17 0.24 0.23 0.26 0.35 0.31
b32  0.39 0.43 0.41 0.18 0.21 0.28 0.18 0.18 0.26 0.27 0.31 0.35 0.31 0.29 0.30 0.33
b48  0.25 0.16 0.19 0.18 0.20 0.24 0.23 0.21 0.31 0.33 0.45 0.10 0.13 0.16 0.10 0.13
```

**Plat entre 0,10 et 0,45, aucune dent de scie lisible.** La cause est dans la prémisse de la
méthode : `DetectI0Layout` suppose une valeur qui **bouge peu** d'une frame à la suivante — vrai
d'un bipède, **faux d'un projectile**, qui traverse la carte entre deux frames et fait donc
basculer les bits de poids fort autant que les autres. **L'instrument n'est pas en cause, sa
prémisse ne s'applique pas.** Forcer une lecture de frontières sur ce profil serait exactement le
défaut que ce chantier s'interdit (un balayage FABRIQUE des distributions crédibles).

### 8.2 T3 — le discriminant physique est INCONCLUANT PAR L'INSTRUMENT

Discriminant de remplacement, et il est bien plus fort en principe : **un projectile vole droit**.
Si le découpage est bon, les positions successives d'un même projectile sont colinéaires ; s'il
est faux, les bits d'un axe polluent le suivant et le nuage n'a aucune structure.

```
decoupage      traj>=3   colineaire   nulle melangee
19/19/19             7       0.0000         0.0000
18/19/20             7       0.0000         0.0000
20/19/18             7       0.0000         0.0000
17/20/20             7       0.0000         0.0000
13/13/14             7       0.0000         0.0000
```

> ⚠ **CE N'EST PAS UN NÉGATIF SUR LA QUESTION, C'EST UN NÉGATIF SUR L'INSTRUMENT** — et le
> dossier a déjà nommé ce piège : *« un négatif dont la nulle vaut zéro est un négatif sur
> l'instrument »* (index §20.1). **Ma nulle vaut zéro elle aussi** : le test ne pouvait pas
> réussir. Deux causes, et elles se corrigent :
>
> 1. **n = 7 trajectoires à 3 points ou plus**, sur 277 échantillons porte = 1. La couverture
>    delta actuelle rend 1 à 2 positions par projectile — il en faut au moins 3 pour tester une
>    droite. **C'est la couverture qu'il faut lever avant le test, pas le test qu'il faut
>    interpréter.**
> 2. **Aucun contrôle positif.** La tolérance (5 % de la longueur du segment) n'a jamais été
>    calibrée sur un composant dont le décodage est SÛR.

### 8.3 CE QU'IL FAUT FAIRE AVANT DE REJOUER T3 — dans cet ordre

1. **Un CONTRÔLE POSITIF de l'instrument de colinéarité** : le rejouer sur les positions de
   **bipèdes** (`ti=35`, i0 dynamic-precision, décodage sûr et déjà capturé par
   `position_capture.go`). Un bipède ne vole pas droit — le critère doit donc être recalibré sur
   une trajectoire connue, ou remplacé par un critère de **continuité** (pas de saut supérieur à
   la vitesse maximale d'un projectile). **Tant que l'instrument n'a pas montré qu'il sait dire
   OUI quelque part, ses zéros ne valent rien.**
2. **Lever la couverture du flux delta** : `DecodeFrameInfer` démarre au bit 0 et meurt tôt sur
   les paquets à événements. `killsource` a un localisateur de boucle (`locateStrict` + repli,
   690/690 paquets) — **il n'est pas exporté**. L'exporter, ou le rejouer, multiplierait les
   échantillons par slot. C'est le vrai verrou de T3.
3. **Alors seulement** rejouer T3, puis T3bis (chaînage sur `i1`), puis le portage.

**ÉTAT À LA CLÔTURE DE CETTE SESSION** : la voie n'est ni ouverte ni fermée. Ce qui est acquis —
le projectile est une entité répliquée qui porte des trajectoires dans le flux delta (§7), les
deux branches de son i0 sont empruntées à parts égales (§8.1), et la grammaire de la branche
opaque est établie (§2.2). Ce qui manque — **la couverture** (point 2) avant tout, puis un
instrument de validation qui ait fait ses preuves (point 1).
- **2026-08-08 (6)** — **T2 et T3 joués, aucun des deux ne mord, et les deux échecs sont
  instrumentaux.** T2 : le profil de bascule ne transfère pas (sa prémisse — une valeur qui bouge
  peu entre frames — est fausse pour un projectile) ; profil plat 0,10-0,45. T3 : colinéarité des
  trajectoires, **zéro partout NULLE COMPRISE** — donc un négatif sur l'instrument, pas sur la
  question (n = 7 trajectoires à 3+ points, et aucun contrôle positif). **Acquis au passage : les
  deux branches d'i0 sont empruntées à parts comparables (264 / 277), donc les 59 bits opaques
  concernent la MOITIÉ des records projectile.** Verrou identifié : la couverture du flux delta,
  qui exige le localisateur de boucle de `killsource` (non exporté). Rien de porté.
