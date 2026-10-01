# T2 — Le décalage de masque (`i - decales`) : constat NÉGATIF, établi des deux côtés (2026-10-01)

> Campagne grammaire, phase 1, étape 2, piste T2. Plan :
> `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`. Code lu à la tête du worktree
> `feat/campagne-grammaire` (`69564ef7d`). Jeu lu dans Ghidra (`HaloInfinite.exe`, base
> `0x140000000`), en LECTURE SEULE (décompilation, désassemblage, xrefs, `read_memory`).
> Aucune commande `go` lancée, aucune base ouverte, aucun film lu.

## 0. Verdict

**Le décalage de masque n'explique AUCUN paquet « hors cadre ».** Le Go teste le bit `i` brut, et
c'est exactement l'index que l'ÉCRIVAIN pose. Le lot 5.17 l'avait conclu par le seul côté LECTEUR
(`FUN_142e2c690`) ; cette note ferme l'autre bout : l'écrivain du masque, l'écrivain des corps de
composants et l'écrivain du registre de `chunk_00` sont lus, et ils indexent TOUS le même tableau
(le descripteur d'archétype du processus qui enregistre), dans le même ordre, sans filtre.

Les « décalés » de `FUN_14076cb60` sont la conversion « descripteur du build qui REJOUE » ->
« registre du film », gardée par le mode rejeu. Sur un film rejoué par son propre build, elle vaut
zéro, et elle n'a pas d'image hors ligne (le décodeur lit déjà le registre du film).

Sous-produit utile à la mesure (§6) : l'écrivain GARANTIT qu'aucun bit de masque n'est posé au-delà
du nombre de composants de l'archétype. Le Go ignore ces bits sans les compter
(`traverse.go:246`) et un commentaire de recherche les dit « inoffensifs »
(`frame_chain_infer.go:127-130`). Pour l'écrivain, un tel bit est impossible : c'est un
**témoin gratuit de lecture mal cadrée**, exactement ce que l'hypothèse « vue C lue au mauvais
endroit » demande de mesurer.

## 1. Le lecteur : ce que sont exactement les « décalés »

`FUN_14076cb60` (décompilation relue ce jour, extrait fidèle) :

```
lecteur = args[5] ; FUN_1406d7610(desc, lecteur, &masque) ; extra = FUN_14076cea8()
decales (iVar15) = 0
pour i (iVar16) de 0 a *(int*)(desc+0x4320)-1 :
   d = *(desc + i*8)
   si TLS+0x238 porte un nom non vide :                       ; mode « contexte nomme »
      si etat(DAT_1445c5838...) == 2 et DAT_144c232e1 == 0 :
         si !FUN_1428e1dac(&DAT_144c23178, desc[0x474c], d->vtable[0x08]()) :
             goto LAB_14076cc46                               ; SORTIE 1 : decales++, aucun bit
   ; LAB_14076cc4e
   si (masque >> ((i - decales) & 0xff) & 0x3f) & 1 :
      niveau = d->vtable[0x00]()
      si FUN_1404f2b4c() et DAT_144c232e1 == 0 :
         niveau = FUN_1428e1b50(&DAT_144c23178, desc[0x474c], d->vtable[0x08]())
         si d->vtable[0x10](niveau) : goto LAB_14076cc46      ; SORTIE 2 : decales++, aucun bit
      ... vtable[0x48] (prediction), vtable[0x28](lecteur, args, &pred, niveau) ...
      si extra et R(1) : R(32) == 0x0bcddcba sinon « entity component corrupt »
      args[0] |= 1 << desc[0x4850 + i]
```

`FUN_1404f2b4c` : vrai si `TLS+0x238` porte un nom non vide ET l'état vaut 2 (relu). Les deux
sorties sont donc gardées par le même contexte, et désarmées par `DAT_144c232e1 != 0`.

**Sortie 1** : le composant du descripteur DU PROCESSUS n'a pas son NOM dans le bloc `ti` du
registre DU FILM (`FUN_1428e1dac` : `base + 8 + ti*0x4100`, pas `0x104`, comparaison de chaînes ;
`base` = `*(sing+0x108)` ou `*(sing+0x120)+0x130`).

**Sortie 2** : le composant est dans le masque, mais le niveau que le FILM déclare pour son nom
(`FUN_1428e1b50`, défaut 1, PREMIÈRE entrée homonyme) est refusé par `vtable[0x10]` du
désérialiseur du processus. Particularité : cette sortie est DANS la branche « bit posé » ;
`decales++` fait alors tester au composant suivant le MÊME bit. Sémantique de compatibilité entre
builds, sans objet sur un film rejoué par son build.

### 1.1 Sur quels archétypes et composants la sortie 2 peut-elle jouer : un seul

Recensement des tables virtuelles de désérialiseurs : toutes les tables dont `vtable[0x48]` vaut
`0x14076ced0` (341 xrefs DATA) ou `0x14076cf10` (3), lues par `read_memory` (`slot2` = `+0x10`),
noms résolus par le getter `vtable[0x08]` (`LEA RAX,[rip+d]`, ou indirection pour trois tables).
**343 tables réelles ; les 326 noms distincts du registre (`ecs_table.tsv`) sont tous couverts**
(trois via getter indirect : `vehicle-type-physics-component` `0x143d0b300`,
`vehicle-type-state-component` `0x143d0b3f0`, `managed-navpoint-visual-state-groups-component-N`
`0x143d081c0`).

| `vtable[0x10]` | tables | effet |
|---|---:|---|
| `0x1404ab600` = `XOR AL,AL ; RET` | 342 | jamais refusé |
| `0x142ee2d98` | **1** : `persistence-state-bucket-component` (table `0x143c96ba8`) | voir ci-dessous |

`0x142ee2d98` (désassemblé à la main depuis les octets `83fa01740433d2eb05ba1a0000000fb641083bc20f93c0c3`) :

```
seuil = (niveau == 1) ? 0x1a : 0
return this->octet[+8] >= seuil          ; niveau != 1 -> TOUJOURS refuse
```

Ses instances : `FUN_140e43a88` construit l'archétype **`ti = 0x18` (24)** (`desc+0x474c` =
`obj+0x4754` = `0x18`) : `state-checksum-component` (type 0), puis une boucle
`octet[+8] = k ; FUN_14064dd28(desc, k+1, inst_k)` pour `k = 0..0x19`. Les 26 seaux portent donc
`octet[+8]` = 0..25, tous `< 26`. Le registre déclare le niveau 1 pour les 26 (`ecs_table.tsv`,
`ti=24 i1..i26`). **La sortie 2 ne joue jamais sur un film de son build.**

### 1.2 La sortie 1 ne peut pas jouer non plus : l'écrivain du registre (§3)

## 2. L'écrivain du masque et des corps : l'index brut du descripteur

Chaîne (toutes les adresses relues ce jour) :

| maillon | adresse | ce qu'il fait |
|---|---|---|
| sérialiseur d'entrée genre 1 / genre 3 | `FUN_142e35a58` / `FUN_142e35e60` | `desc = *(vue[0x18] + 8 + archetype*8)` (archétype lu en `+4` de l'entrée de datum), puis `FUN_142e2ad9c(desc, ctx, &bits)` |
| corps du record | `FUN_142e2ad9c` | `FUN_142e2af58(desc+8, …)` (prépare), puis `FUN_142e2f7f8(desc+8, writer, …, masque, tampon)` |
| préparation | `FUN_142e2af58` / `FUN_142e2b368` | boucle `i < *(desc+0x4320)` sur le bitset du record ; écrit chaque corps dans un tampon (`vtable[0x20]`), note `(largeur, i)` en `tampon+0x88` pas 6 ; pose le bit `i` du masque final |
| **le masque** | **`FUN_142e2da44`** | voir grammaire ; appelé par `FUN_142e2f7f8` @142e2f8b9 et par `FUN_141f85ce0` @141f85d19 (seuls appelants) |
| les corps | `FUN_142e2f7f8` | `pour i < *(desc+0x4320)` : si bit `i` du masque -> copie du corps (`FUN_1406d60f4`) ; puis, si `FUN_14076cea8()` : `FUN_142e2ec8c(ti)` vrai -> `W(1)=0` ; faux -> `W(1)=1`, `W(32)=0x0bcddcba` |

Grammaire de `FUN_142e2da44(desc, masque, w)`, littéraux du décompilé :

```
n = *(int*)(desc+0x4320)                    ; nombre de composants du descripteur
S = { i < n : bit i du masque pose }         ; AUCUN i >= n ne peut y entrer
si |S| > 7 : W(1)=1 ; W(64) = somme(1 << i, i dans S)          -> rend 0x41
sinon      : W(1)=0 ; W(3) = |S| ; pour i croissant dans S : W(6) = i   -> rend 4 + 6|S|
```

Aucun `decales`, aucun filtre par nom ni par niveau : **l'index écrit est l'index BRUT du
descripteur de l'enregistreur**, et l'ordre des corps suit le même `i`. Le lecteur `FUN_1406d7610`
(seul appelant : `FUN_14076cb60`) est sa réciproque — c'est `consumeMask` (`traverse.go:295`).

## 3. L'écrivain du registre : copie verbatim du descripteur, même ordre

| maillon | adresse | ce qu'il fait |
|---|---|---|
| ajout d'un composant | `FUN_14064dd28(desc, typeId, deser, groupe)` | si `typeId` absent du bitmap `desc+0x4950` : `desc[n] = deser` ; nom (`vtable[0x08]`) copié en `desc + 0x200 + n*0x104` ; niveau (`vtable[0x00]`) en `desc + 0x300 + n*0x104` (= entrée + `0x100`) ; `desc+0x4850[n] = typeId` ; `n++` |
| préparation de l'en-tête de film | `FUN_14299b674` | `pour ti` (50 blocs, `plVar13 += 0x820` qwords = `0x4100` o) : `FUN_142998c7c(hdr + 0x138 + ti*0x4100, obj_ti + 0x208)` — `obj_ti + 0x208` = `desc + 0x200` |
| la copie | `FUN_142998c7c` | 64 entrées de `0x104` : `0x100` octets de nom + le `u32` de niveau, VERBATIM |
| la relecture | `FUN_14299ab50` | `R(FUN_141cfff30(version))` octets en `+8` ; `FUN_141cfff30` = `nombre_de_blocs * 0x20800` bits (`0x4100` octets par bloc) |

Le bloc `ti` du registre du film est donc, entrée pour entrée, le tableau `desc+0x200` du build
qui ENREGISTRE ; et l'entrée `k` est le nom et le niveau de `desc[k]`, celui dont le masque pose le
bit `k`. La sortie 1 ne peut jouer que si le build qui rejoue a un descripteur différent de celui
qui a enregistré. `registry.go` (`parseRegistry`, `:267-300`) lit ces entrées nommées dans l'ordre,
s'arrête au premier nom vide (les 343 getters rendent tous une chaîne non vide), et garde le niveau
de chaque entrée (`:285`).

## 4. Le Go : où il itère le masque, et s'il reproduit le décalage

| site | lecture du masque | boucle |
|---|---|---|
| `frame_records.go:211` (`decodeDelta`) | `consumeMask` | `traverseComponentLoop` |
| `traverse.go:129` (record NEW, `TraverseEntity`) | `R(1)` porte puis `consumeMask` | idem |
| `frame_harvest.go:272` (`decodeDeltaWithArch`) | `consumeMask` | idem |
| `frame_chain_infer.go:137` (`deltaBodyTrial`, recherche) | `consumeMask` | idem |
| `keyframe_fullstate_loop.go:110` (état complet) | aucun : `Mask = ^0` | idem — comme `FUN_142e2c690`, qui n'a pas de masque |
| `traverse.go:245-247` (`traverseComponentLoopFrom`) | — | `for i < len(arch.Components)` ; bit `i` BRUT |

`arch.Components` = le bloc `ti` du registre du film. Index Go `i` = index registre `i` = index
écrivain `i`. **Le Go ne reproduit pas le décalage, et il ne doit pas le reproduire** : il n'a pas de
descripteur de processus à convertir. Ratchet existant : `masque_cadre_registre_test.go`.

Deux équivalences secondaires vérifiées :

- **Niveau d'un homonyme.** En rejeu, `FUN_1428e1b50` rend le niveau de la PREMIÈRE entrée
  homonyme et le passe à `vtable[0x28]` ; le Go passe `arch.Level(i)` de l'entrée elle-même. Sur
  `ecs_table.tsv`, les 33 groupes d'homonymes d'un même bloc (dont `ti=24` ×26, `ti=27` ×64,
  `ti=35 weapon-state-*` ×4) ont TOUS un niveau unique : équivalent sur ce build. Les autres builds
  ne sont pas vérifiés (proposition de ratchet au §7).
- **Garde de corruption.** L'écrivain (`FUN_142e2f7f8` @142e2f992..142e2f9ab, `FUN_141f85ce0`)
  écrit `W(1)=0` pour un archétype que `FUN_142e2ec8c(ti)` exempte, `W(1)=1 ; W(32)` sinon : le
  flux est AUTO-DESCRIPTIF ; `consumeCorruptionCheck` (`R(1)` puis `R(32)` si 1) le lit sans avoir
  besoin de `FUN_142e2ec8c`. Le drapeau global vient du film : `FUN_14299b674` écrit
  `hdr+0xcb58c = DAT_1450e24e8`, que `FUN_14299ab50` relit par `R(1)` en `+0xcb45c`
  (déjà porté, lot 5.18.2, `controle_corruption_du_film.go`).

## 5. Ce que T2 ne change pas

Aucun correctif de grammaire. Aucun gain de fermeture attendu de ce chef. La phrase de
`frame_infer.go:122-125` (« Sa cause NOMMÉE est le décalage du masque ») est la note du lot 5.16,
déjà démentie par le 5.17 et par `traverse.go:228-243` ; elle reste un historique daté, pas une
affirmation vivante.

## 6. Le sous-produit : l'invariant « masque inclus dans [0, n) »

**Ce que dit l'écrivain** : `FUN_142e2da44` ne pose un bit que pour `i < *(desc+0x4320)`, et
`*(desc+0x4320)` = nombre d'entrées nommées du bloc (§3). Un record ÉCRIT par le jeu n'a donc
jamais de bit de masque `>= len(arch.Components)` — ni en dense, ni en épars.

**Ce que fait le Go** : `traverseComponentLoopFrom` ne regarde que `i < len(arch.Components)`
(`traverse.go:246`) : un bit au-delà est ignoré, SANS compteur. Et `frame_chain_infer.go:127-130`
dit qu'un contrôle de plage « rejetait des alignements VRAIS (717 des 902 gains à un pas) » et
qu'un masque trop large est « inoffensif, pas un indice de mauvaise lecture ». Pour l'écrivain,
c'est l'inverse : ces 717 « gains » avaient un masque que le jeu ne peut pas écrire, donc un
archétype (ou un cadrage) faux — la déduplication par bit de fin de `chainCtx` masque l'archétype.
(Le chemin d'inférence est HORS production : `InferenceChaine` vaut faux par défaut,
`profil_balayage.go:254-257`, et `TablesParVue` rejette les deltas non liés avant lui.)

**Pouvoir discriminant** : pour une lecture tombée à une position aléatoire sur un archétype de `n`
composants, le masque passe l'invariant avec une probabilité ≈ `1/2 · moy_{c=0..7} (n/64)^c +
1/2 · 2^-(64-n)` — environ 7 % pour `n = 10`, 6 % pour `n = 3` (`ti=47`), 6 % pour `n = 1`
(`ti=4`), ~22 % pour `n = 48` (`ti=40`), et 100 % pour `n = 64` (`ti=35`, `ti=7`, `ti=27`,
`ti=46` : aucun bit n'est hors archétype). C'est donc un témoin fort sur les petits archétypes, et
nul sur le bipède.

## 7. Vecteurs de test (d'après l'écrivain) et mesure proposée

### 7.1 Vecteurs (synthétiques, `buildRegistry` comme `masque_cadre_registre_test.go`)

Bloc à homonymes, forme de `ti=24` : un composant porté de largeur fixe en tête, puis 26 homonymes
d'un composant porté de largeur fixe (p. ex. le `R(24)` de
`managed-object-networked-splash-message-dynamic-component`), niveau 1 partout.

| vecteur | bits (ordre du flux) | attendu |
|---|---|---|
| V1 épars, homonymes | `0` `011` `000000` `000011` `011010` puis 3 corps | composants consommés aux index 0, 3, 26 (le 26e homonyme), `EndBit` = 1+3+18 + 3 corps |
| V2 borne épars | 7 index -> `0` `111` + 7×6 bits | épars ; 8 index -> l'écrivain passe en dense : `1` + 64 bits |
| V3 index 63 | bloc de 64 entrées, `0` `001` `111111` | consomme l'entrée 63 |
| V4 négatif | bloc de `n` entrées, épars `0` `001` + `n` sur 6 bits | IMPOSSIBLE chez l'écrivain ; aujourd'hui : 0 composant, silence ; avec l'instrument : compté « masque hors archétype » |

V1-V3 doivent passer à l'identique AVANT et APRÈS toute modification (ils figent l'égalité
écrivain = Go). V4 est le vecteur de l'instrument.

### 7.2 Mesure (carte de fermeture v2, aucune sortie de production modifiée)

1. Par record delta/NEW lu : `masque >> len(arch.Components) != 0` (pour `len < 64`), compté dans
   l'observateur, sans changer la marche.
2. Ventilé (a) sur les records des paquets FERMÉS — attendu **0** (un non-zéro réfuterait le §3,
   p. ex. un registre mal parsé sur un build) ; (b) sur le DERNIER record lu de la vue B des
   paquets « vue C : terminateur hors cadre », par sortie de vue B (terminateur / rejet) et par
   archétype.
3. Lecture : si (b) est nettement au-dessus de (a) sur les petits archétypes, la vue B a lu au
   moins un record à une position fausse AVANT de sortir — c'est l'hypothèse de travail
   (« la vue C est lue au mauvais endroit ») mesurée record par record, et le premier record
   fautif date le début du désalignement. Si (b) ≈ (a) ≈ 0, le désalignement naît après le dernier
   record (fin de vue B ou vue C elle-même).
4. Corollaire pour 1.3 : `recordUtile` (`frame_closure_classement.go:263-275`) classe « utile » un
   record d'après son masque ; un record à masque hors archétype ne devrait pas entrer au
   dénominateur des records utiles.

Correctif de phase 2 possible SI la mesure (a) vaut 0 sur les 20 films : traiter un masque hors
archétype comme une désynchronisation (`DesyncAt`) dans `decodeDelta` / `TraverseEntity`. Il
CHANGE des sorties (records aujourd'hui publiés sur un corps mal cadré) : décision et gate de
corpus après J12, pas dans cette phase.

## 8. Adresses citées

`FUN_14076cb60`, `FUN_1406d7610`, `FUN_14076cea8`, `FUN_1404f2b4c`, `FUN_1428e1dac`,
`FUN_1428e1b50`, `FUN_142e2c690`, `FUN_1428e2b68`, `FUN_142e35a58`, `FUN_142e35e60`,
`FUN_142e2ad9c`, `FUN_142e2af58`, `FUN_142e2b368`, `FUN_142e2f7f8`, `FUN_142e2da44`,
`FUN_141f85ce0`, `FUN_142e2ec8c`, `FUN_14064dd28`, `FUN_14299b674`, `FUN_142998c7c`,
`FUN_14299ab50`, `FUN_14299ac50`, `FUN_141cfff30`, `FUN_142985b24`, `FUN_140e43a88`,
`0x142ee2d98` (non défini comme fonction dans la base Ghidra, lu en octets), tables
`0x143c96ba8` (seau de persistance), `0x143d085d0` (splash), `0x143e2bbc8` (position).

## 9. Découvertes hors périmètre (consignées, non traitées)

- `FUN_142e2b368` : quand `(drapeaux & param_10 & 7) == 0`, l'écrivain n'appelle pas
  `vtable[0x20]` mais réserve `*(short*)(cache + 0x22 + i*4)` bits et pose quand même le bit du
  masque ; `FUN_142e2f7f8` copie alors un corps venu d'un cache par entité (branche
  `*(byte*)(lVar6+4) != i`). Un corps de composant peut donc être une RECOPIE d'un envoi
  précédent ; sans effet sur la largeur lue (le lecteur lit le corps tel quel), à garder en tête
  pour T5/T1.
- `FUN_142e2ec8c(ti)` : exemption de la sentinelle de corruption PAR ARCHÉTYPE (`W(1)=0`). Non
  lue ; sans effet sur la lecture (flux auto-descriptif), utile pour prédire où une sentinelle
  `0x0bcddcba` peut apparaître.
