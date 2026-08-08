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
3. **`FUN_14076e304` (le R(2)) est CONDITIONNEL** — gardé par `FUN_140492128(st + 4)`, un
   prédicat sur la position décodée. Le port le lit **inconditionnellement** sur la branche basse
   précision. Si le prédicat est parfois faux, le port lit 2 bits de trop, et tout ce qui suit
   dans le record est décalé.
4. **`FUN_14076e3e4` (handle-tail) est appelé INCONDITIONNELLEMENT**, pas seulement sur la
   branche haute. Il ne consomme des bits que si son paramètre de garde est non nul (sinon il
   pose des sentinelles `0xffffffff` sans rien lire), mais **cette garde n'est pas la porte R(1)**
   du port.

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
- [ ] `DAT_143b8c6d0` — la plage par défaut de CETTE branche. ⚠ **Ne pas la confondre** avec
      `DAT_143b8c6b8` (± 20000), citée par le dossier : les deux adresses sont distantes de
      **0x18 = 24 octets = un AABB (min+max en vec3)**, donc ce sont probablement deux entrées
      voisines d'une même table. Une plage plus petite donnerait les 19 bits de l'hypothèse.
- [ ] `FUN_140be9b88` / `FUN_1406d7f18` — la fonction de largeur et le lecteur, pour confirmer
      19 plutôt que de le déduire d'une soustraction.
- [ ] `FUN_14076e524` — d'où viennent exactement les trois largeurs d'axe, et la porte
      `index-sel` / `index de région`.
- [ ] `FUN_140492128` — le prédicat qui garde le R(2).
- [ ] `FUN_14076e3e4` — la garde réelle du handle-tail, et sa largeur (un `0x40 - n < 0xb`
      suggère 11 bits).
- [ ] **PUIS SEULEMENT** : test sur film (profil de bascule + chaînage i1), avant tout portage.

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
  **Bilan à cette heure : 1 écart retiré, 2 écarts de longueur encore à tester sur film
  (le `R(2)` conditionnel, la garde réelle du handle-tail), 1 gain potentiel (la position sur
  la branche par défaut).** Rien de porté, rien de vérifié sur film.
