# PISTE A — ti=11 i1 `managed-objective-color` : lecture native du propriétaire d'objectif

> Document de CONCEPTION (aucun code produit, aucune mesure, aucun build). RE Ghidra
> LECTURE SEULE sur `HaloInfinite.exe` (image base `0x140000000`, build 2026-06-04,
> 311 103 fonctions), MCP Ghidra opérationnel. Daté 2026-08-27, branche `feat/v75`.
> Chaque fait porte une adresse Ghidra OU un `fichier:ligne`. Objectif : établir si on peut
> lire NATIVEMENT le propriétaire/camp d'une zone/colline depuis la keyframe (composant
> ti=11 i1), ce qui débloquerait le repli owner KOTH aujourd'hui `[!]`.

---

## 0. Résumé exécutif — le résultat de fond

**La feuille i1 est trouvée, décompilée, et elle est TRIVIALE ; et — fait neuf — toute la
chaîne de feuilles jusqu'à i1 est désormais entièrement résolue.** Il ne reste AUCUN bloqueur
de grammaire de feuille sur le chemin qui mène à i1. Le seul gate restant est le CADRE (le
corps d'image-clé type-2 se lit-il par la boucle d'état complet ?), exactement l'inconnue que
le scout du 2026-08-27 avait laissée ouverte.

Ce que cette passe a établi de NEUF (tout par Ghidra ce jour) :

| feuille ti=11 | deser (nouveau) | grammaire | largeur | dst | statut avant |
|---|---|---|---|---|---|
| **i0** managed-objective-timers | **FUN_142ed5a6c** → FUN_1410d9088 | **2× R(7)** (biais −1 à la valeur) | 14 bits | dst[0],dst[1] | non adressé |
| **i1** managed-objective-color | **FUN_142ed544c** | **4× R(8) déquant [0,1] = RGBA** | 32 bits | dst+0x08/+0x0c/+0x10/+0x14 | non adressé |

Rappel du contexte déjà établi (bloc de feuilles ti=11, scout) : état par défaut ti=11 =
`consumeVersionPrefix` (FUN_14110d4d8, V seul) DÉJÀ porté (`default_state_arch.go:51`) ; i3
FUN_142ed5550, i5 FUN_1410fc4a4, i12 FUN_142ed575c, i13 FUN_142ed5844, i14 FUN_142ed5948.

**Conséquence directe : la marche d'état complet, étendue à i0+i1, atteint le bit de départ
de i1 SANS aucune feuille inconnue en amont** (en-tête → V (porté) → i0 (2×R(7), résolu) →
i1 (4×R(8), résolu)). Le seul risque résiduel n'est plus dans les feuilles ; il est
entièrement dans le CADRE.

**Deux gates séparés, à ne pas confondre :**
1. **Gate CADRE** (portabilité de la lecture native) : l'en-tête par entité + le bloc d'état
   par défaut + les mots de contrôle du mode film reproduisent-ils sur le payload type-2 du
   film ? INCONNU. R6 donne une vraie raison d'en douter (le jeu SAUTE le type-2), mais ti=11
   est l'instrument le moins cher et le plus PROPRE pour trancher, car sans confond de
   précision de feuille (contrairement à ti=35, plafond 0,51 %).
2. **Gate SÉMANTIQUE** (utilité pour l'owner KOTH) : même si le cadre reproduit, i1 est une
   RGBA d'affichage, PAS un index d'équipe. Il faut mapper couleur→camp, et écarter le risque
   « couleur relative au point de vue » vs « couleur absolue d'équipe ».

**VERDICT : GO-CONDITIONNÉ.** La grammaire de i1 (et de toute la chaîne jusqu'à elle) est
portable MAINTENANT. La lecture NATIVE dépend du gate CADRE. Le déblocage owner KOTH dépend
en plus du gate SÉMANTIQUE. Détail et protocole en §5.

---

## 1. GRAMMAIRE de la feuille i1 (question 1) — RÉSOLUE, triviale

### 1.1 Chaîne de résolution (recette R7-d, rejouée)

```
search_strings "managed-objective-color"  -> "managed-objective-color-component" @0x143c952a0
get_xrefs_to 0x143c952a0                   -> getter de nom @0x141177fc0  (DATA)
get_xrefs_to 0x141177fc0                   -> slot +0x08 de vtable @0x143d08d58  (DATA)
vtable de base = 0x143d08d58 - 0x08        =  0x143d08d50
read_memory 0x143d08d50 (64 o)             -> deser en +0x30
```

Vtable `0x143d08d50` (lue octet à octet), MÊME layout que toutes les feuilles ti=11 :

| slot | valeur | rôle |
|---|---|---|
| +0x00 | `0x14117b4a0` | dtor (identique aux autres feuilles ti=11) |
| +0x08 | `0x141177fc0` | getter de nom |
| +0x10 | `0x1404ab600` | ret-false |
| +0x18 | `0x142edb548` | ÉCRIVAIN |
| +0x20 | `0x1411c8f80` | int3 |
| +0x28 | `0x14076ce9c` | thunk (LE MÊME forwarder pur que tout ti=11 : 0 bit) |
| **+0x30** | **`0x142ed544c`** | **DESER (cible)** |
| +0x38 | `0x1404ab600` | ret-false |

### 1.2 Le deser `FUN_142ed544c` — décompilé + désassemblé

```c
undefined8 FUN_142ed544c(param_1, param_2 /*BitReader*/, longlong param_3 /*ctx*/) {
  lVar1 = *(param_3 + 0x10);                                  // bloc d'état
  uVar2 = FUN_1406d84b4(param_2, param_2, 0, DAT_143cd8374, 8, 0, 1);  *(lVar1+0x08)=uVar2; // R(8)->[0,1]
  uVar2 = FUN_1406d84b4(param_2);                             *(lVar1+0x0c)=uVar2;          // R(8)
  uVar2 = FUN_1406d84b4(param_2);                             *(lVar1+0x10)=uVar2;          // R(8)
         FUN_1406d84b4(param_2);                              *(lVar1+0x14)=XMM0;           // R(8)
  return 1;
}
```

Désassemblage (largeurs confirmées sur pièces) : 4 appels `CALL 0x1406d84b4`, chacun précédé
de `MOV dword ptr [RSP+0x20],0x8` (`142ed5475`, `142ed548e`, `142ed54ad`, `142ed54cc`) =
**largeur 8 bits**. Bornes : `MOVSS XMM3,[0x143cd8374]` (max) + `XORPS XMM2,XMM2` (min=0.0),
chargées UNE fois. `read_memory 0x143cd8374` = `00 00 80 3f` = **1.0f**. Écritures aux offsets
`RDI+0x08/+0x0c/+0x10/+0x14` (`RDI = [R8+0x10]`).

**Grammaire : 4× R(8) déquantifié dans [0.0, 1.0] = une couleur RGBA. Largeur totale 32 bits.
INCONDITIONNEL — aucune porte, aucun branchement sur valeur, aucune dépendance à un état
runtime, aucun drapeau d'encodage.** C'est une feuille aussi triviale que possible.

### 1.3 Contre-épreuve : identité structurelle avec ti=10 boundary-color (DÉJÀ porté bit-exact)

`FUN_142ed52b4` (ti=10 i1 `managed-object-boundary-color`, porté par
`consumeManagedObjectBoundaryColor`, `components_managed_object.go:93`) est le MÊME code, au
byte près :

```c
uVar2 = FUN_1406d84b4(param_2, param_2, 0, DAT_143cd8374, 8, 1, 1);  *(lVar1+0x04)=uVar2;  // + 3 fois
```

Deux seules différences, aucune ne touche le nombre de bits consommés :
- **offsets de destination** : ti=11 écrit à +0x08.. ; ti=10 à +0x04.. (layout de struct différent).
- **6e argument du 1er appel** : ti=11 passe `0`, ti=10 passe `1` (drapeau d'arrondi/signe de la
  déquantification, il change la VALEUR flottante, pas la largeur — les deux lisent 8 bits).

`FUN_1406d84b4` est le déquantifieur générique déjà porté et validé bit-exact par le port
ti=10 (et cité tout le long de `WALK_PORT_NOTES.md`, ex. `FUN_142ee2194 = FUN_1406d84b4(...,
0x10, 0, 1)` = R(16)). Sa largeur = l'argument `bits` = 8. Rien à re-décompiler.

L'écrivain miroir `FUN_142edb548` sérialise 4 valeurs via `FUN_142ed1a78` (le
quantifieur-écriture) depuis `*(*(param_3+0x30)+8)` — 4 écritures, confirme 4 composantes.

### 1.4 Feuille amont i0 (RÉSOLUE cette passe) — la seule qui gardait i1

Comme i1 est à l'index 1, seule i0 la précède dans la boucle de composants. i0 était `non
adressé`. Résolu par la même recette : chaîne `"managed-objective-timers-component"`
`@0x143c95278` → getter `@0x141177fd0` → vtable `@0x143d08da0` → deser +0x30 = **`FUN_142ed5a6c`**.

```c
undefined4 FUN_142ed5a6c(param_1, param_2 /*BitReader*/, longlong param_3) {
  puVar3 = *(param_3 + 0x10);  puVar1 = puVar3 + 2;
  for (; puVar3 != puVar1; puVar3++) { *puVar3 = FUN_1410d9088(param_2); }  // 2 itérations
  return 1;
}
```

`FUN_1410d9088` : lecteur de **7 bits** sur le modèle BitReader du paquet (état
`+0x28/+0x2c/+0x30/+0x38/+0x40`, byte-swap MSB-first identique à `filmdec.BitReader` ; le
compteur `+0x2c` est incrémenté de 7 dans les deux branches, refill gardé par
`0x40 - iVar1 < 7`). Retourne la valeur `− 1` (biais de décodage, pas de largeur).

**i0 timers = 2× R(7) = 14 bits, inconditionnel.** TRIVIAL.

---

## 2. SÉMANTIQUE (question 2) — RGBA d'affichage, PROXY d'owner à mapper (pas un index d'équipe)

### 2.1 Ce que le deser prouve, et ce qu'il ne prouve pas

Le deser prouve que i1 est une **couleur RGBA** (4 composantes déquantifiées dans [0,1]) —
STRUCTURELLEMENT une couleur d'affichage, PAS un index d'équipe. Un index d'équipe serait un
petit entier (cf. ti=9 team-designator = R(4), `FUN_140f581e8`, `traverse.go:478` ; ti=11 i14
state = R(3)). i1 n'est pas de cette forme.

Donc : **i1 ne donne pas un « camp » entier lisible directement.** Pour en tirer un
propriétaire, il faut mapper la RGBA → équipe. La couleur est un PROXY de l'owner (le HUD
teinte l'objectif par l'équipe qui le contrôle), pas l'owner lui-même.

### 2.2 Recoupement avec ti=10 boundary-color et le HUD

- Le catalogue cite ti=10 boundary-color comme « teinte par propriétaire »
  (`CATALOGUE_OBJECTIFS_DECRITS_2026-08-27.md` §4.7 / `ecs_table.tsv:239`) : même famille
  « couleur d'objet managé ». C'est l'argument que la valeur SUIT l'owner. Mais le catalogue
  note aussi que le 1er octet de ti=10 prend **4 niveaux discrets (55/119/183/247) « non
  expliqué »**. Quatre niveaux discrets sur un canal couleur est un INDICE fort que la couleur
  est quantifiée à un petit palette d'états/camps (et non un dégradé continu) — cohérent avec
  « couleur keyée par owner/état ». C'est un indice, pas une preuve.
- Recoupement HUD (setter côté logique de mode) : le setter qui CALCULE la couleur depuis
  l'équipe n'est pas atteignable par la voie chaîne→getter→vtable (la chaîne
  `"managed-objective-color-component"` n'a qu'UN xref, le getter ; la vtable
  `0x143d08d50` n'est référencée que par une table de typeinfo `@0x144746f98` ; le deser et
  l'écrivain ne sont référencés que par DATA — dispatch pur). Établir « couleur = f(équipe) »
  côté RAM exigerait un dig profond hors périmètre offline-pur. **La sémantique se tranche donc
  par la MESURE (§5.2), pas par le décompile.**

### 2.3 Le risque sémantique décisif — absolu vs point-de-vue

Question ouverte, non tranchable au décompile : la RGBA répliquée est-elle **absolue** (équipe
0 = telle couleur, équipe 1 = telle autre, fixe) ou **relative au point de vue** (ami = bleu,
ennemi = rouge, du point de vue du joueur qui a enregistré) ? La state d'un composant ECS
répliqué est en principe à valeur unique (donc plutôt absolue), mais ce n'est pas garanti.
- Si ABSOLU : la RGBA se cluster en couleurs fixes par équipe → mapping direct → owner par
  équipe. C'est le cas exploitable pour le repli KOTH.
- Si POV-relatif : « bleu » ne veut dire que « l'équipe de l'enregistreur » → on ne récupère
  que ami/ennemi du point de vue d'un seul, pas l'owner par équipe absolue. Exploitation
  dégradée (utile pour un rejeu à POV fixe, insuffisant pour un owner d'équipe canonique).

Ce risque est la RAISON d'être du protocole de confrontation §5.2 : il est conçu pour le
détecter (une couleur POV donnerait un mapping incohérent selon l'équipe tenante).

---

## 3. EXTENSION du lecteur (question 3) — précise, réutilise le patron existant

### 3.1 Point d'insertion unique : `consumeByName`

La boucle d'état complet est DÉJÀ portée (`keyframe_fullstate_loop.go`,
`WalkKeyframeFullState`) et REUTILISE `traverseComponentLoop` (`traverse.go:1201`), qui
dispatche PAR NOM via `consumeByName` (`traverse.go:194`). Câbler i1 = ajouter UN case, sur le
patron EXACT de ti=10 i1 (`traverse.go:857`) :

```go
// dans consumeByName, à côté des cases managed-object (traverse.go:854-865)
case compManagedObjectiveColor: // ti=11 i1 (FUN_142ed544c) — 4xR(8) RGBA, publie
    consumeObjectiveColor(br)
    return variant, nil, true
case compManagedObjectiveTimers: // ti=11 i0 (FUN_142ed5a6c -> FUN_1410d9088) — 2xR(7)
    consumeObjectiveTimers(br)
    return variant, nil, true
```

Deser (nouveau fichier `components_objective.go`, ou dans `components_managed_object.go` par
proximité de famille), calqué SANS recopie sur `consumeManagedObjectBoundaryColor` :

```go
const (
    compManagedObjectiveColor  = "managed-objective-color-component"  // ti=11 i1
    compManagedObjectiveTimers = "managed-objective-timers-component" // ti=11 i0
)

// ti=11 i1 — FUN_142ed544c : 4x R(8) déquant [0,1] = RGBA. 32 bits, inconditionnel.
func consumeObjectiveColor(br *BitReader) {
    r, g, b, a := br.ReadBits(8), br.ReadBits(8), br.ReadBits(8), br.ReadBits(8)
    publishObjective(ObjectiveColor, r, g, b, a) // hook nommé par famille, cf. SetManagedObjectHook
}

// ti=11 i0 — FUN_142ed5a6c -> FUN_1410d9088 : 2x R(7). 14 bits, inconditionnel.
func consumeObjectiveTimers(br *BitReader) {
    t0, t1 := br.ReadBits(7), br.ReadBits(7) // valeur native = quantum − 1 (biais du deser)
    publishObjective(ObjectiveTimers, t0, t1)
}
```

Hook : un `objectiveHook func(f ObjectiveField, values []uint64)` + `SetObjectiveHook`,
strictement calqué sur `SetManagedObjectHook`/`SetNavpointHook` (hook nommé PAR FAMILLE
d'archétype — modèle imposé par le paquet, `components_managed_object.go:163-168`). Publier
le QUANTUM brut (règle du fichier : la convention de déquantification n'est pas figée ;
convertisseur `ObjectiveColorValue(q) = dequantMidpoint(q, 8, 0, 1)` en option pour l'appelant).

### 3.2 Archétype et niveau — rien à câbler en plus

- Archétype : `reg.Archetype(11)` fournit la liste ordonnée des 34 composants (registre
  chunk_00) ; `WalkKeyframeFullState` pose `t.Mask = ^uint64(0)` (état complet, tous présents)
  et appelle `traverseComponentLoop`. Le dispatch par NOM route i0/i1 sans autre changement
  (`sonde_ti11_objectifs_test.go` confirme que la couverture se lit du dispatch lui-même).
- Bloc d'état par défaut ti=11 : DÉJÀ géré — `consumeKeyframeDefaultState(br, 11)`
  (`keyframe_record_walk.go:346`) route vers `defaultStateDeserByTI[11] = consumeVersionPrefix`
  (`default_state_arch.go:51`, FUN_14110d4d8, V seul). Aucun trou en amont d'i0.
- Niveau : i0 L=0, i1 L=1 (`ecs_table.tsv:268-269`) ; ces feuilles n'utilisent pas le niveau
  (largeurs fixes), l'option `LevelShift` de la boucle reste une variable de cadre inerte pour
  elles.

### 3.3 Champ produit au document de rejeu

`ObjectiveColor` (RGBA, 4× u8) par slot ti=11 et par keyframe, publié via le hook. Le
consommateur (côté `internal/analysis`, hors périmètre RE) le mappe couleur→équipe (§5.2) pour
alimenter le repli owner KOTH (`zone_states_hill.go`, aujourd'hui `[!]` sur le repli par
rampes, `CATALOGUE:179`). Mettre aussi à jour `ecs_table.tsv:268-269` (`non_porte` →
`porte`, deser_addr, grammar, code_source) et le garde-rail `ecs_table_guard_test.go`.

---

## 4. Le CADRE — pourquoi c'est le seul gate restant, et ce que R6 en dit

Rappel des acquis contradictoires que ti=11 doit arbitrer :
- **R6** (`WALK_PORT_NOTES.md` §2) : le démultiplexeur de streaming du film (`FUN_1428e22c0`)
  SAUTE le paquet type-2 (le handler du type-1 avance le curseur par-dessus). Le jeu ne décode
  JAMAIS le type-2 en lecture de film → « pas de consommateur ». Fort prior que le corps
  type-2 n'est décodable par AUCUNE grammaire portée.
- **Scout 2026-08-27** (`RE_LECTEUR_IMAGE_CLE_SCOUT` §2 et §5) : NUANCE — la mise en place
  d'une « Film View » (`FUN_142e2e104`) appelle le lecteur d'état complet
  (`FUN_142e2aab4` → `FUN_142e2bfd0` → `FUN_142e2c690`). Il EXISTE un site d'appel côté film ;
  mais la PROVENANCE du flux `*(ctx+0x130)` (bloc type-2 vs snapshot re-synthétisé) reste non
  résolue statiquement.
- **R7-c/e** : sur ti=35 (bipède), la grammaire d'état complet plafonne à 0,51 % bit-exact, et
  la portée baseline (positions en 96 bits bruts) EMPIRE la mesure (3/591 → 0/591). MAIS ce
  plafond est DANS les feuilles du bipède (vec3 quantifié aux largeurs de carte), pas isolable
  du cadre.

**Pourquoi ti=11 tranche là où ti=35 ne pouvait pas.** i1 (et toute la chaîne i0→V→i1) n'a
AUCUN vec3, AUCUNE position quantifiée, AUCUN drapeau d'encodage, AUCun compte gardé par un
état RAM. Le confond de précision de feuille qui coulait ti=35 est ABSENT. Donc un échec de
landing sur ti=11 ne peut PLUS être imputé aux feuilles : il localiserait le mur dans le
CADRE/format de façon DÉFINITIVE. Et un succès rend l'owner d'objectif natif. Les deux issues
sont décisives et, maintenant que i0 et i1 sont résolus, MAXIMALEMENT bon marché.

---

## 5. VALIDATION sans build (question 4) — deux oracles, seuils écrits AVANT mesure

### 5.1 Oracle de CADRE — atterrissage bit-exact (le gate n°1)

**Instrument** : réutiliser le harnais `keyframe_biped_fullstate_test.go` (TestKF35FullState),
transposé à ti=11. La frontière du record SUIVANT est donnée par une chaîne DISJOINTE de toute
lecture de corps : `WalkKeyframeWorld` (`keyframe_world.go`, en-tête 64 bits
`[id:32][field:26][ti:6]`, validé 249/250 vs oracle Cheat Engine). Une marche juste atterrit
BIT-EXACT sur la frontière du record ti=11 suivant.

**Protocole** :
1. Sur les films oracles à objectif (le corpus `kf35OracleFilms` + un film KOTH/Strongholds),
   énumérer les records ti=11 bornés par leur voisin (`kf35BoundedRecs` filtré `r.TI == 11`).
2. Pour chaque record, jouer `WalkKeyframeFullState` (i0+i1 câblés) en balayant les variables
   de cadre du plan R7-e SÉPARÉMENT : `HeaderBits ∈ {64,108}`, `SizeWords`, `DefaultState`,
   `LevelShift`, `filmComponentCorruptionCheck`. Une variable allumée à la fois.
3. Mesurer le taux d'atterrissage exact (+ chaîné) contre la frontière, avec dénominateur
   (`bounded`), plus la médiane de l'écart absolu et le point de décrochage (patron
   `kf35Report`). Témoin de contrôle : la variante « record NEW » (déjà réfutée sur ti=37/38),
   pour que la longueur consommée se compare à quelque chose.
4. **Complément « frontières live »** : `kf_capture_sample.txt` (400 frontières de records,
   jamais consommé) comme second jeu de frontières là où il porte des records ti=11 — noter
   qu'il est SPARSE pour les objectifs (cf. le négatif publié ti=42, `WALK_PORT_NOTES.md` §4),
   donc secondaire à `WalkKeyframeWorld` sur films réels.

**Seuil écrit AVANT mesure** : le cadre est jugé REPRODUIT si, pour UNE combinaison de
variables tenue d'avance, l'atterrissage exact+chaîné des records ti=11 est **≥ 90 %** sur AU
MOINS 2 films (dénominateur ≥ 100 records ti=11 bornés cumulés), avec un témoin « record NEW »
resté sous 20 % sur les mêmes records. En-dessous de 90 % → cadre NON reproduit ; si le témoin
et la variante d'état complet sont tous deux bas (< 5 %), le mur est le FORMAT (cf. §6), pas la
grammaire des feuilles (qui sont triviales et résolues).

### 5.2 Oracle SÉMANTIQUE — la couleur suit-elle l'owner connu (le gate n°2)

**Instrument** : dans un film Bastion/KOTH, le propriétaire est DÉJÀ publié par un canal
indépendant : `zone_states.go` (propriétaire zone 93,1 %) et `zone_states_hill.go` (owner
colline KOTH 88-89 % via désignateur ti=13). La confrontation ne coûte pas de build de
décodeur : c'est un test Go additif sous garde d'environnement (patron `TI11_FILM`), qui LIT
i1 (une fois câblé) et le compare à l'owner publié au même horodatage.

**Protocole** :
1. Pour chaque keyframe et chaque objectif de zone/colline, décoder i1 (RGBA) et l'owner
   publié (zone_states/hillStates) à cet instant.
2. Clustering des RGBA observées en ≤ 3 clusters (ami/ennemi/neutre attendu). Établir le
   mapping cluster→équipe par la MAJORITÉ des instants où l'owner est connu.
3. Mesurer l'accord i1↔owner (part des instants où le cluster de la couleur == owner publié),
   avec dénominateur, SÉPARÉMENT par équipe tenante (c'est la coupe qui révèle un encodage
   POV : un POV donnerait un accord haut pour une équipe et effondré pour l'autre).
4. Témoin : la couleur d'un objectif NON contesté/neutre doit se distinguer d'une couleur de
   contrôle prise sur un slot voisin (bande fantôme, patron `ti11GhostBand`).

**Seuil écrit AVANT mesure** : i1 est jugé PROXY d'owner EXPLOITABLE si l'accord i1↔owner est
**≥ 90 %** globalement ET **≥ 85 % dans CHAQUE équipe tenante prise séparément** (la coupe
par équipe est le juge du risque POV). Un accord global ≥ 90 % mais une équipe < 50 % =
encodage POV-relatif → exploitation dégradée seulement (owner ami/ennemi à POV fixe, pas owner
d'équipe absolu). Nombre de clusters distincts > 4 = la couleur n'est pas keyée par un petit
palette d'owner → NON exploitable comme owner.

Note : ce gate est INDÉPENDANT du gate cadre pour le clustering (on peut cataloguer les valeurs
i1 lues même à landing imparfait), mais l'accord bit-exact au bon slot exige le gate cadre
tenu ; les deux se mesurent dans la même passe.

---

## 6. VERDICT (question 5) — GO-CONDITIONNÉ

**GO-CONDITIONNÉ.** Décomposé :

- **Grammaire des feuilles : GO franc, portable MAINTENANT.** i1 (FUN_142ed544c, 4× R(8)) est
  trivial ; i0 (FUN_142ed5a6c → FUN_1410d9088, 2× R(7)) est trivial et résolu cette passe ;
  l'état par défaut ti=11 est déjà porté. **Il n'existe plus AUCUN bloqueur de grammaire de
  feuille sur le chemin qui mène à i1.** C'est le progrès net de cette passe par rapport au
  scout (qui laissait i1 « non adressé » et le catalogue qui le classait DOC faute de RE de
  feuille).

- **Lecture NATIVE de i1 : CONDITIONNÉE au gate CADRE (§5.1).** La boucle d'état complet
  ACTUELLE, étendue à i0+i1, produirait une valeur lisible SI ET SEULEMENT SI le cadre (en-tête
  par entité + bloc d'état par défaut + mots de contrôle du mode film) reproduit sur le payload
  type-2. C'est l'inconnue que le scout avait laissée et que R6 rend douteuse (le jeu saute le
  type-2). ti=11 est désormais l'instrument le PLUS cher-efficace pour la trancher, sans confond
  de feuille. Condition = le test §5.1 franchit son seuil de 90 %.

- **Déblocage owner KOTH : CONDITIONNÉ aux DEUX gates.** Même cadre tenu, i1 est une RGBA
  d'affichage, pas un index d'équipe ; l'owner sort d'un mapping couleur→équipe qui doit passer
  le gate sémantique §5.2 (et notamment écarter le risque POV). Le repli owner KOTH par rampes
  (`CATALOGUE:179`, `zone_states_hill.go:34-35`) n'est débloqué que si les deux seuils tombent.

**Si NO-GO (le cadre ne reproduit pas, seuil §5.1 non franchi malgré des feuilles triviales) :**
le résultat serait DÉCISIF dans l'autre sens — il localiserait définitivement le mur dans le
CADRE/FORMAT du payload type-2, fermant la dernière hypothèse ouverte (les feuilles étant
prouvées triviales, l'échec ne peut plus leur être imputé, contrairement à ti=35). Le seul
levier restant serait alors, dans l'ordre :
1. **La provenance du flux `*(ctx+0x130)`** lu par le lecteur d'état complet côté « Film View »
   (`FUN_142e2e104`) — est-ce le bloc type-2 ou un snapshot re-synthétisé ? Fil statique unique
   à tirer (scout §4.2), hors de cette passe.
2. **Le levier type-0** cité par le catalogue (l'ÉCRIVAIN `FUN_142e35a58`, flux type-0) : le
   catalogue pose que le corps d'image-clé type-2 est un cul-de-sac de grammaire et que le vrai
   flux serait type-0. Mais R6 a cherché l'écrivain type-2 et n'a trouvé NI site d'allocation
   nommé NI chaîne d'erreur (`WALK_PORT_NOTES.md` §5) : ce levier exige de sourcer la grammaire
   depuis le CONTENU (table déjà balayée 249/250 vs CE) ou un autre binaire (serveur dédié),
   hors offline-pur. À ne rouvrir que si §5.1 échoue.

**NO-GO explicites (ne pas relancer) :** reprise du lecteur pour ti=35 (0,51 %, dérive dans les
feuilles) ; thèse « lecteur bat écrivain » (dispatches miroirs) ; attendre de
`kf_capture_sample.txt` seul qu'il valide le cadre objectifs (sparse, négatif déjà publié).

---

## 7. Ordre de travail recommandé (pour l'exécuteur, hors de cette passe RE)

1. Câbler i0 + i1 dans `consumeByName` + deser + hook `SetObjectiveHook` (patron
   `components_managed_object.go`). MAJ `ecs_table.tsv:268-269` + garde-rail.
2. Gate CADRE (§5.1) : transposer `keyframe_biped_fullstate_test.go` à ti=11, balayer les
   variables R7-e, mesurer l'atterrissage bit-exact vs `WalkKeyframeWorld`. SEUIL 90 % écrit
   d'avance. C'est le test décisif — le lancer AVANT toute exploitation sémantique.
3. Si gate cadre tenu : gate SÉMANTIQUE (§5.2), confrontation i1↔owner publié, coupe par
   équipe, seuils 90 %/85 % écrits d'avance.
4. Si les deux tenus : brancher le repli owner KOTH sur i1 (côté `internal/analysis`).
5. Bonus quasi-gratuit une fois le cadre tenu : les 5 feuilles du scout (i3/i5/i12/i13/i14)
   deviennent lisibles d'un coup — porteur d'objectif (i3), type (i5), progression (i12/i13),
   état contesté/neutre/tenu (i14). Le porteur Oddball (i3) est le `[!]` de 5 campagnes.

---

## Annexe — table des adresses établies cette passe (toutes Ghidra, lecture seule)

| objet | adresse |
|---|---|
| chaîne `managed-objective-color-component` | `0x143c952a0` |
| getter de nom i1 | `0x141177fc0` |
| vtable ti=11 i1 (base) | `0x143d08d50` |
| **deser i1 (cible)** | **`0x142ed544c`** |
| écrivain i1 | `0x142edb548` |
| déquantifieur générique (bits=8) | `0x1406d84b4` |
| constante max 1.0f | `0x143cd8374` (`00 00 80 3f`) |
| thunk forwarder (partagé ti=11) | `0x14076ce9c` |
| chaîne `managed-objective-timers-component` | `0x143c95278` |
| getter de nom i0 | `0x141177fd0` |
| vtable ti=11 i0 (base) | `0x143d08da0` |
| **deser i0** | **`0x142ed5a6c`** |
| lecteur R(7) de i0 | `0x1410d9088` |
| écrivain i0 | `0x142edbac8` |

Réserve (héritée du catalogue) : l'étiquette « i0 »/« i1 » repose sur l'ordre du registre et
la concordance nom↔champ ; la grammaire et le champ de destination, eux, sont vérifiés
bit-exact. Le dispatch de la boucle se fait par NOM à l'exécution — le câblage par nom
(`consumeByName`) est donc robuste à un décalage d'index de build.
