# Enveloppe des événements de film — table des domaines, émetteurs, références

Date : 2026-08-30. Lot B du chantier « visée lunette ». Rétro-ingénierie **pure, lecture seule**
sur `HaloInfinite.exe` (greffon Ghidra HTTP 127.0.0.1:8089). Aucun code du dépôt n'a été modifié.

Toutes les adresses sont des adresses virtuelles du binaire retail analysé. Les extraits sont le
pseudocode Ghidra, ou le désassemblage lu octet par octet quand Ghidra ne déclare pas de fonction.

---

## Résumé exécutif

1. **La table domaine → plage existe et est entièrement décodée** : `0x1451f98d0`, 9 entrées
   (domaines 0..8), remplie au démarrage par `FUN_140d10bb0`. Chaque entrée = (base, cardinal).
   La largeur en bits est `ceil(log2(cardinal))`. Elle **n'a pas besoin d'être capturée en vivant** :
   les valeurs sont calculées à partir d'une seule constante de l'image. Nuance importante :
   les domaines larges (0, 1, 7, 8) sont **réécrits au runtime** si le monde dépasse 8191 objets.
2. **La sonde d'1 bit n'existe que pour le domaine 1**, et nulle part ailleurs. C'est une propriété
   du domaine, pas du site d'appel.
3. Le var-int rend un **handle 32 bits** : `(2 bits de génération) << 30 | (base + index)`, et la
   porte d'1 bit qui le précède signifie simplement « référence non nulle » (`!= 0xFFFFFFFF`).
   Confirmé des deux côtés, lecture et écriture.
4. **Le type 114 s'appelle bien `biped_board_vehicle`** (preuve directe par `getName`), mais
   **il existe un type dédié à la lunette : le type 126 `unit_zoom`**, dont la charge utile est
   `R(2) − 1`, soit un niveau de zoom dans {−1, 0, 1, 2} où **−1 = pas de lunette**. Un seul type
   porte donc la mise ET la sortie de lunette, le sens étant dans la valeur.
5. **L'entrée et la sortie de véhicule sont deux types distincts** : 114 `biped_board_vehicle` et
   **103 `unit_exit_vehicle`**, plus **62 `unit_switch_seat`**. Les deux premiers portent chacun un
   `R(6)` (index de siège très probable) ; le troisième n'a aucune charge utile.
6. **L'hypothèse « la lunette est un siège de l'arme » est réfutée** par les preuves statiques :
   aucune chaîne `weapon_seat` n'existe dans le binaire, et le moteur possède un événement de zoom
   dédié et séparé.
7. Les domaines sont des **bandes de l'espace d'index des objets réseau**, de sémantique établie
   par recoupement : bande 2 = unités/bipèdes, bande 3 = véhicules, bande 5 = projectiles,
   bande 4 = union unités+véhicules.

---

## B1. La table domaine → plage du var-int

### B1.1 Le lecteur var-int et sa source de plage

`FUN_1406d3140(param_1 nom, param_2 flux, param_3 DOMAINE, param_4 sortie)`. Extrait du prologue :

```c
uVar7 = DAT_144706100;                       // repli : 0x1FFF
if (DAT_144706104 != '\0') {
  uVar8 = (&DAT_1451f98d0)[(longlong)param_3 * 2];   // base     (champ +0)
  uVar7 = (&DAT_1451f98d4)[(longlong)param_3 * 2];   // cardinal (champ +4)
}
if (((param_3 == 1) && (cVar2 = FUN_1406cf008(param_2), cVar2 != '\0')) &&
   (uVar8 = uVar14, uVar7 = DAT_144706100, DAT_144706104 != '\0')) {
  uVar8 = DAT_1451f98f0;                     // = entrée 4 de la table (base 0x200)
  uVar7 = DAT_1451f98f4;                     // = entrée 4 de la table (cardinal 0x200)
}
iVar3 = FUN_1406d310c(uVar7);                // largeur en bits
```

- **Adresse de la table des plages par domaine : `0x1451f98d0`.** Entrée de 8 octets par domaine :
  `+0x00` = base (offset ajouté à la valeur lue), `+0x04` = cardinal (nombre de valeurs).
  L'indexation `[param_3 * 2]` sur des `int` confirme le pas de 8 octets.
- `DAT_1451f98f0` / `DAT_1451f98f4` ne sont pas des variables séparées :
  `0x1451f98f0 − 0x1451f98d0 = 0x20 = 4 × 8`, c'est **l'entrée 4 de la même table**.
- Constantes de repli : `DAT_144706100 = 0x1FFF`, `DAT_144706104 = 1` (lu dans l'image :
  `read_memory 0x144706100` → `ff1f0000 01000000 fe070000 fe070000`). Le drapeau valant déjà 1
  dans l'image, **le chemin par table est le chemin actif**.
- `FUN_1406cf008` est bien une lecture d'un seul bit (avance de 1 sur le compteur `+0x2c`).

### B1.2 Valeurs retail : calculées au démarrage, et par qui

La table est **à zéro dans l'image** (`read_memory 0x1451f98d0` sur 128 octets : que des zéros).
Elle est remplie au démarrage par **`FUN_140d10bb0`** (appelée par `FUN_140d10a78` en
`0x140d10b96`), qui pose aussi le drapeau :

```c
void FUN_140d10bb0(void) {
  iVar1 = DAT_144706100;          // 0x1FFF
  piVar2 = &DAT_1451f98d4;        // pointe le champ "cardinal" de l'entrée 0
  DAT_144706104 = 1;
  iVar3 = 0;
  do {
    if ((iVar3 == 0) || (iVar3 == 1)) { *piVar2 = iVar1 + -0x200; LAB: piVar2[-1] = 0x200; }
    else if (iVar3 == 2) { *piVar2 = 0x100; goto LAB; }
    else if (iVar3 == 3) { piVar2[-1] = 0x300; LAB2: *piVar2 = 0x100; }
    else if (iVar3 == 4) { *piVar2 = 0x200; goto LAB; }
    else if (iVar3 == 5) { piVar2[-1] = 0x400; goto LAB2; }
    else { piVar2[-1] = 0; if (iVar3 == 6) { *piVar2 = 0x200; } else { *piVar2 = iVar1; } }
    iVar3 = iVar3 + 1; piVar2 = piVar2 + 2;
    if (8 < iVar3) return;            // la table s'arrête à l'entrée 8
  } while(true);
}
```

Les valeurs ne dépendent que de `DAT_144706100`. **Il n'est donc pas nécessaire de capturer cette
table en vivant** (contrairement à ce qu'envisageait le journal du chantier) — sous la réserve
runtime décrite en B1.6. Table reconstituée :

| Domaine | Base | Cardinal | Largeur `R(n)` | Plage de valeurs | Coût total avec génération |
|---|---|---|---|---|---|
| 0 | 0x200 | 0x1DFF | 13 | 0x200..0x1FFE | 15 bits |
| 1 | 0x200 | 0x1DFF | 13 | 0x200..0x1FFE | 1 (sonde) + 13 + 2 = **16 bits** |
| 1 (sonde = 1) | 0x200 | 0x200 | 9 | 0x200..0x3FF | 1 + 9 + 2 = **12 bits** |
| 2 | 0x200 | 0x100 | 8 | 0x200..0x2FF | 10 bits |
| 3 | 0x300 | 0x100 | 8 | 0x300..0x3FF | 10 bits |
| 4 | 0x200 | 0x200 | 9 | 0x200..0x3FF | 11 bits |
| 5 | 0x400 | 0x100 | 8 | 0x400..0x4FF | 10 bits |
| 6 | 0 | 0x200 | 9 | 0..0x1FF | 11 bits |
| 7 | 0 | 0x1FFF | 13 | 0..0x1FFE | 15 bits |
| 8 | 0 | 0x1FFF | 13 | 0..0x1FFE | 15 bits |

La largeur vient de `FUN_1406d310c` (`BSR` + arrondi supérieur), c'est-à-dire
**`ceil(log2(cardinal))`** :

```c
int FUN_1406d310c(uint param_1) {          // 1406d3114: BSR ECX,ECX
  iVar1 = <position du bit de poids fort>;
  if (param_1 == 0) return 0;
  return iVar1 + ((param_1 & ((1 << iVar1) - 1)) != 0);   // +1 si non puissance de 2
}
```
Vérification : `0x1FFF → 13` (conforme au 13 bits déjà connu), `0x100 → 8`, `0x200 → 9`,
`0x1DFF → 13`.

### B1.3 Sortie du var-int : un handle, pas un simple index

Fin de `FUN_1406d3140` : après les `n` bits de l'index, **2 bits sont lus inconditionnellement**,
puis :

```c
uVar8 = uVar8 + uVar11;                 // base + index lu
...                                     // lecture de 2 bits supplémentaires -> uVar11
*param_4 = uVar11 << 0x1e | uVar8;      // (génération << 30) | (base + index)
```

**Le var-int rend donc un handle 32 bits : bits 0..29 = index absolu, bits 30..31 = génération.**
C'est le format classique d'un datum handle. Le `R(2)` du contexte n'est pas un champ séparé : il
fait partie intégrante de la référence.

**Confirmation par le côté écriture.** `FUN_1406d2464` est le sérialiseur symétrique et lève toute
ambiguïté sur la porte d'1 bit :

```c
*(ulonglong *)(param_2 + 0x30) = ... | (ulonglong)(uVar8 != 0xffffffff);   // porte = handle non nul
uVar8 = *param_1;
if (uVar8 != 0xffffffff) {
  uVar6 = (ulonglong)(uVar8 >> 0x1e);          // génération = bits 30..31
  iVar5 = DAT_1451f98d0; uVar4 = DAT_1451f98d4;
  iVar3 = FUN_1406d310c(uVar4);
  uVar8 = (uVar8 & 0x3fffffff) - iVar5;        // index relatif = (handle & 0x3FFFFFFF) - base
```

**La porte d'1 bit signifie « la référence n'est pas nulle », la valeur nulle étant `0xFFFFFFFF`.**

Grammaire d'une référence, confirmée des deux côtés :
`R(1) porte ; si 1 : [1 bit de sonde si domaine == 1] + R(ceil(log2(cardinal))) + R(2)`.

### B1.4 Site d'appel exact du dispatcher, arguments passés

Boucle des 3 références dans `FUN_14080a9d4`, désassemblage `0x14080aa9e` puis `0x14080ab22` :

```
14080aa9e: CALL 0x1406cf008          ; R(1) : porte de la référence i
14080aaa3: TEST AL,AL
14080aaa5: JNZ 0x14080ab22
...
14080ab22: MOV RAX,qword ptr [R15]   ; R15 = descripteur du type
14080ab25: MOV EDX,R12D              ; EDX = i (0,1,2)
14080ab28: MOV RCX,R15
14080ab2b: CALL qword ptr [RAX + 0x58]   ; +0x58(descripteur, i) -> EAX = DOMAINE
14080ab2e: MOV ECX,EBP                   ; ECX = numéro de type
14080ab30: MOV EBX,EAX                   ; EBX = domaine
14080ab32: CALL 0x14080ada0              ; journalisation : appelle getName (+0x08) du descripteur
14080ab37: MOV R9,qword ptr [RSP + 0x8c8]  ; param_4 = destination
14080ab3f: MOV R8D,EBX                     ; param_3 = DOMAINE
14080ab42: MOV RDX,RDI                     ; param_2 = flux de bits
14080ab45: CALL 0x1406d3140                ; le var-int
```

**Le domaine rendu par `+0x58` est bien passé en 3e argument (`R8D`) du var-int. C'est prouvé.**
`param_1` (`RCX`) n'est jamais lu par `FUN_1406d3140` : c'est un nom de champ de débogage, laissé
par l'appel précédent, sans effet.

`FUN_14080ada0` n'écrit rien dans le flux — elle ne fait qu'un appel de `getName` à des fins de
journalisation (`0x14080adcf: CALL qword ptr [RAX + 0x8]`) : **elle ne consomme aucun bit**.

### B1.5 La sonde : présente ou non sur ce chemin

**La sonde est présente sur ce chemin, mais uniquement quand le domaine vaut exactement 1.** Elle
n'est pas un paramètre du site d'appel : c'est le test `param_3 == 1` à l'intérieur du var-int qui
la déclenche, avec court-circuit (`&&`), donc **aucun bit n'est lu pour les autres domaines**.
Conséquences concrètes :

- type 105 `action_weapon_fire`, référence 0 (domaine 1) : **la sonde est lue**, la référence coûte
  12 ou 16 bits selon sa valeur ;
- type 114 `biped_board_vehicle` (domaines 2, 3, 7) : **aucune sonde**, références de 10, 10 et
  15 bits ;
- type 126 `unit_zoom` (domaines 4, 8, 7) : **aucune sonde**, références de 11, 15 et 15 bits.

Quand la sonde vaut 1, le codage bascule sur les paramètres de l'entrée 4 (base 0x200, cardinal
0x200) : l'objet est dans la sous-plage 0x200..0x3FF et l'index tient sur 9 bits au lieu de 13.
C'est un codage court pour le cas fréquent, pas un champ sémantique.

### B1.6 Réserve runtime : les domaines larges peuvent changer de largeur

`DAT_144706100` n'est pas une constante figée : **`FUN_1408f1618` la réécrit**, ainsi que quatre
entrées de la table, lorsque le monde alloue un objet d'index supérieur à la capacité courante :

```c
DAT_1451f98d4 = uVar7 - 0x1ff;   // cardinal du domaine 0
DAT_144706100 = uVar6;           // nouvelle borne haute (= index max + 1)
DAT_1451f98dc = DAT_1451f98d4;   // cardinal du domaine 1  (0x1451f98d0 + 0x0C)
DAT_1451f990c = uVar6;           // cardinal du domaine 7  (0x1451f98d0 + 0x3C = 7*8+4)
_DAT_1451f9914 = uVar6;          // cardinal du domaine 8  (0x1451f98d0 + 0x44 = 8*8+4)
```
avec `uVar7 = handle & 0x3FFFFFFF` et `uVar6 = uVar7 + 1`. La configuration initiale correspond
exactement à `uVar7 = 0x1FFE` : cardinal domaine 0 = `0x1FFE − 0x1FF = 0x1DFF`, borne = `0x1FFF`.
Tout concorde.

Conséquences pour un décodeur hors ligne :

- **Domaines 2, 3, 4, 5, 6 : largeurs FIXES** (8 ou 9 bits) — jamais réécrites. Sûres.
- **Domaines 0, 1, 7, 8 : 13 bits tant que le monde ne dépasse pas 8191 objets** (index max
  0x1FFE). Au-delà, la largeur augmente. En multijoueur classique ce plafond n'est pas atteint,
  mais un film de Forge ou de campagne mérite vérification.
- **La table ne compte que 9 entrées** (0..8), soit `0x1451f98d0`..`0x1451f9917`. Un domaine ≥ 9
  lirait hors table (cardinal 0 → `FUN_1406d310c(0) = 0` → aucun bit d'index lu). Si un descripteur
  semble rendre un domaine ≥ 9, c'est un signe de décodage erroné, pas une entrée à capturer.

---

## B2. L'émetteur du type 114 et la sémantique des références

### B2.0 Confirmation formelle de l'identité du type 114

- Table statique des descripteurs : **base `0x144724A90`**, 8 octets par type.
  `0x144724A90 + 114×8 = 0x144724E20`, et `get_xrefs_to 0x143d0d330` rend exactement
  `From 144724e20 [DATA]`. L'index 114 pointe donc bien le descripteur `0x143d0d330`.
- Recoupement indépendant : `0x144724A90 + 105×8 = 0x144724DD8` → `0x143d0aca0`, dont le `+0x58`
  vaut `0x14080a048`, valeur déjà établie par les phases antérieures pour le type 105.
- **Nom prouvé directement** : `getName` du 114 est à `0x14119e9b0`, octets
  `48 8d 05 c9 95 af 02 c3` = `LEA RAX,[RIP+0x02AF95C9] ; RET`, soit la chaîne à
  `0x14119e9b7 + 0x02AF95C9 = 0x143C97F80`. Lecture de cette adresse : **`biped_board_vehicle`**.
  `get_xrefs_to 0x143C97F80` ne rend que `From 14119e9b0` : correspondance biunivoque.

### B2.1 (a) Entrée et sortie : deux types distincts, et le cas particulier de la lunette

En appliquant la même méthode (`table[t]` → `vtable+0x08` → `LEA` → chaîne) à tous les types, on
obtient trois événements de siège **distincts** :

| Type | Nom | Charge utile |
|---|---|---|
| 62 | `unit_switch_seat` | **aucune** (lecteur = `0x1408d8220`, une fonction `return 1`) |
| 103 | `unit_exit_vehicle` | `R(6)` (lecteur `0x142f17b94`) |
| 114 | `biped_board_vehicle` | `R(6)` (lecteur `0x142f168c0`) |

**Réponse à (a) : non, l'entrée et la sortie ne passent pas par le même type.** La sortie a son
propre type, le 103 `unit_exit_vehicle`. Il n'existe par ailleurs aucune chaîne `unboard` dans le
binaire. Aucun champ de sens n'est donc nécessaire dans le 114 : le sens est porté par le numéro de
type lui-même.

**Mais la question de la lunette se résout autrement — et c'est le résultat principal de ce lot.**

Il existe un type dédié : **type 126 = `unit_zoom`**, descripteur `0x143d0da50`.
Preuve : `getName` = `0x141174630`, octets `48 8d 05 d1 39 b2 02 c3` → chaîne à `0x143C98008` =
**`unit_zoom`** ; et `get_xrefs_to 0x143d0da50` rend `From 144724e80`, or
`0x144724e80 − 0x144724A90 = 0x3F0 = 126 × 8`.

Son lecteur `0x141168b28` est un simple relais :

```c
undefined4 FUN_141168b28(undefined8 p1, undefined8 p2, undefined4 *param_3, undefined8 param_4) {
  uVar1 = FUN_14080cb98(param_4);
  *param_3 = uVar1;
  return 1;
}
```

et `FUN_14080cb98` lit **2 bits** et retourne **la valeur moins un** :

```c
int FUN_14080cb98(longlong param_1) {
  ...
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 2;   // 2 bits consommés
  ...
  return uVar4 - 1;                                          // valeur dans {-1, 0, 1, 2}
}
```

**Interprétation : `unit_zoom` porte un niveau de lunette signé, où −1 signifie « pas de lunette ».**
La mise ET la sortie de lunette passent donc par un seul et même type (126), le sens étant porté
par la valeur du champ — exactement le schéma que la question (a) envisageait, mais pour le 126 et
non pour le 114.

### B2.2 (b) Ce que désignent les trois références

Les fonctions `+0x58` sont de petites fonctions `switch(i)` sans lecture de bits. Décodage octet à
octet (Ghidra n'en déclare aucune) :

- **Type 114**, `+0x58 = 0x142f1556c` :
  `85d2 7411 83ea01 7406 b807000000 c3 b803000000 c3 b802000000 c3`
  → `i==0 → 2`, `i==1 → 3`, `i>=2 → 7`.
- **Type 126**, `+0x58 = 0x14116f6f0` : `85d2 0f85 c2ad1b01 8d4204 c3` → `i==0 → EDX+4 = 4` ;
  le saut mène au bloc froid partagé `0x14232A4BA` (`83ea01 7406 b807000000 c3 b808000000 c3`)
  → `i==1 → 8`, `i>=2 → 7`.
- **Type 105**, `+0x58 = 0x14080a048` : `85d2 0f85 6a04b201 8d4201 c3` → `i==0 → EDX+1 = 1` ;
  même bloc froid `0x14232A4BA` (fusion de queues par l'optimiseur) → `i==1 → 8`, `i>=2 → 7`.
- **Types 62 et 103**, `+0x58 = 0x14080a018` (fonction partagée) :
  `85d2 740a b807000000 83ea01 7505 b801000000 c3` → `i==0 → 1`, `i==1 → 1`, `i>=2 → 7`.

La sémantique des bandes s'établit par recoupement sur d'autres types (même méthode) :

| Type | Nom | domaine de ref0 |
|---|---|---|
| 61 | `activate_spartan_ability` | 2 |
| 120 | `biped_pickup_item_request` | 2 |
| 99 | `weapon_empty_click` | 2 |
| 114 | `biped_board_vehicle` | 2 |
| 125 | `vehicle_auto_turret_choose_target` | **3** |
| 112 | `projectile_impact_effect` | **5** |
| 115 | `projectile_detonate` | **5** |
| 94 | `biped_equipment_activation` | 4 |
| 126 | `unit_zoom` | 4 |
| 100 | `weapon_effect` | 8 |
| 64 | `weapon_overheat` | 1 |

**Carte des bandes (établie par recoupement, pas par une table nommée) :**

- **bande 2** `[0x200,0x300)` = **unités / bipèdes** (les acteurs : joueurs et PNJ).
- **bande 3** `[0x300,0x400)` = **véhicules** (preuve la plus nette :
  `vehicle_auto_turret_choose_target` désigne son véhicule porteur en domaine 3).
- **bande 4** `[0x200,0x400)` = **union unité + véhicule** — utilisée quand l'acteur peut être
  l'un ou l'autre. C'est le domaine de `unit_zoom.ref0`, ce qui est parfaitement cohérent : un
  joueur à pied comme un joueur en tourelle de véhicule peuvent zoomer.
- **bande 5** `[0x400,0x500)` = **projectiles**.
- **bande 6** `[0,0x200)` : non observée dans l'échantillon.
- **domaine 1** = objet quelconque de `[0x200,0x1FFF)`, avec la sonde comme raccourci vers la
  bande 4. **Domaines 7 et 8** = espace complet `[0,0x1FFF)`, 13 bits, utilisés pour les
  références de contexte (ref2 systématiquement en domaine 7).

**Réponse à (b) pour le type 114 : ref0 = l'unité / le bipède qui embarque (bande 2),
ref1 = le véhicule (bande 3), ref2 = une référence de contexte non contrainte (domaine 7),
identique en position et en domaine pour tous les types examinés.** La nature exacte de ref2 n'est
pas établie (voir incertitudes).

### B2.3 (c) Le `R(6)` final

Le lecteur du 114 (`FUN_142f168c0`) lit **un unique `R(6)`** et l'écrit dans `*param_3` :

```c
*(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;   // 6 bits
...
*param_3 = uVar5;
return CONCAT71(...,1);
```

Le même champ `R(6)` figure dans `unit_exit_vehicle` (type 103, `FUN_142f17b94` : même structure,
`+ 6` sur le compteur de bits, même décalage de 0x1a). **Un champ présent à l'identique en entrée
et en sortie de véhicule, sur 6 bits (0..63), correspond à un identifiant de siège** : c'est
l'interprétation la plus économique, et elle est cohérente avec le vocabulaire du moteur relevé
dans le binaire (`VehicleSeat 0x1436cb1a8`, `IsDriverSeat 0x1437573c0`, `IsGunnerSeat 0x1437570d0`,
`IsPassengerSeat 0x1437566a0`, `unit_seat_mapping 0x143c24728`, `SeatStringId 0x143c45908`,
`Unit_IsEnteringSeat 0x143c39c58`, `Unit_IsExitingSeat 0x143c39c70`, `Unit_EnterSeat 0x143ca1818`).
Elle **n'est pas formellement prouvée** : rien dans le binaire ne nomme ce champ.

Fait gênant, signalé pour l'honnêteté du raisonnement : `unit_switch_seat` (type 62), qui devrait
au premier abord porter un siège cible, **n'a aucune charge utile** — son siège cible doit donc
transiter par une de ses références (ses ref0 et ref1 sont toutes deux en domaine 1) ou être
recalculé par le receveur.

### B2.4 La lunette n'est pas un siège

La question posée supposait que la lunette pouvait être un siège de l'arme. Les éléments
contredisent cette hypothèse :

- **aucune chaîne `weapon_seat` n'existe** dans le binaire (recherche exhaustive, 0 résultat) ;
- **aucune chaîne `unboard`** non plus ;
- le moteur possède un **événement de zoom dédié** (126) avec un niveau sur 2 bits ;
- le vocabulaire de zoom est distinct de celui des sièges et vit du côté arme/joueur :
  `GetZoomState 0x1436cb138`, `IsZoomed 0x143757790`, `zoom_level 0x143bf4c68`,
  `scope_zoom_level 0x143c4e0d0`, `button_action_scope_zoom 0x1437feaf8`,
  `button_action_change_scope_level 0x1437fec80`,
  `weapon_zoom_magnification_setting 0x143c25e28`, `descopeOnDamage 0x143737840`.

Le champ `R(2)−1` de `unit_zoom` (valeurs −1, 0, 1, 2) s'aligne exactement sur ce vocabulaire :
un niveau de grossissement, avec un état « pas de lunette ».

### B2.5 Sites d'émission : non atteints dans ce lot

**Point non résolu, et je le marque comme tel.** L'écrivain du 114 (`0x142f19afc`) n'a **aucun
appelant direct** : `get_xrefs_to` ne rend que `From 143d0d390 [DATA]`, c'est-à-dire l'entrée
`+0x60` de sa propre vtable (`0x143d0d330 + 0x60 = 0x143d0d390`). Idem pour le descripteur, qui
n'est référencé que depuis la table (`0x144724e20`). L'émission passe donc par un **mécanisme
générique indexé par numéro de type** (le même motif que la réception :
`MOV RAX,[R10+0x18] ; MOV R15,[RAX+R15*8+0x210]`), et non par un appel nommé.

Remonter aux sites d'émission suppose d'identifier cette fonction d'émission générique puis ses
appelants passant l'immédiat `0x72` (114) ou `0x7e` (126). Ce travail est **à faire dans un lot
suivant** ; il n'était pas atteignable dans le budget de celui-ci sans compromettre B1 et B3.

---

## B3. Validation éclair sur le type 105 : que désigne le domaine 1

Le `+0x58` du type 105 (`0x14080a048`) rend bien **1** pour la référence 0 (`8d 42 01` =
`LEA EAX,[RDX+1]` avec `EDX = 0`).

**Le domaine 1 n'est pas « le joueur ».** C'est le **domaine générique des objets du monde** :
base 0x200, cardinal 0x1DFF, soit tout l'espace d'index `[0x200, 0x1FFF)` — donc n'importe quel
objet à l'exception de la bande basse `[0,0x200)`. Sa particularité est son **codage adaptatif** :
la sonde d'1 bit signale que l'objet appartient à la sous-plage `[0x200,0x400)` (= bande 4 = unités
et véhicules, le cas de loin le plus fréquent), auquel cas l'index tient sur 9 bits au lieu de 13.

Pour `action_weapon_fire`, la référence 0 désigne donc **l'objet qui tire, unité à pied ou véhicule
indifféremment**, encodé de façon économique dans le cas courant. La lecture « domaine 1 = joueur »
serait une erreur de décodage : elle ferait manquer le bit de sonde et décalerait tout le paquet.

---

## Annexe. Catalogue des types d'événements 50..127

Obtenu mécaniquement : `table[t]` (base `0x144724A90`) → `vtable + 0x08` → `LEA RAX,[rip+X]` →
chaîne. Les types 0..49 de cette table statique ne suivent pas ce format (valeurs non-pointeurs aux
index 46..49, zéros et petits entiers ailleurs) et n'ont pas été résolus ; le dispatcher borne de
toute façon le type à `< 0x7b` (123) — voir incertitudes.

| # | Nom | # | Nom |
|---|---|---|---|
| 50 | Script | 89 | RequestChangeFrameConfiguration |
| 51 | EquipmentSpawnedObject | 90 | CancelCinematic |
| 52 | EquipmentObjectKnockedBack | 91 | ClientResourcesLoadComplete |
| 53 | EquipmentKnockbackPlayer | 92 | ClientOnlyShowComplete |
| 54 | EquipmentKnockbackRequest | 93 | CollectibleUnlockEvent |
| 55 | EquipmentTranslocatorTeleportEffects | 94 | biped_equipment_activation |
| 56 | ShowDebugText | 95 | AIJuke |
| 57 | supply_request | 96 | AILand |
| 58 | Allegiance | 97 | DebugSendCameraPosition |
| 59 | PlayEffectOnObject | 98 | AISetMotorProgram |
| 60 | SetDifficultyAndSkulls | 99 | weapon_empty_click |
| 61 | activate_spartan_ability | 100 | weapon_effect |
| **62** | **unit_switch_seat** | 101 | AIRequestIdleTransitionTime |
| 63 | teleport_effects | 102 | AIPhase |
| 64 | weapon_overheat | **103** | **unit_exit_vehicle** |
| 65 | unit_teleported | 104 | LoadForgeObjectGroup |
| 66 | synchronized_teleport | **105** | **action_weapon_fire** |
| 67 | PromptToBootGriefer | 106 | request_weapon_fire |
| 68 | PowerUpApplied | 107 | CrewOrderPositionAdd |
| 69 | QueueNextShow | 108 | CrewSetTargetObject |
| 70 | player_forge_action | 109 | SaveToUGCService |
| 71 | player_forge_user_string_action | 110 | NetworkedCrewEventType |
| 72 | SaveGame | 111 | projectile_object_impact_effect |
| 73 | RevertMap | 112 | projectile_impact_effect |
| 74 | motor_system_interruption | 113 | biped_pickup |
| 75 | networked_ai_action | **114** | **biped_board_vehicle** |
| 76 | NetworkedActionRequest | 115 | projectile_detonate |
| 77 | networked_ai_effect | 116 | player_set_orbiting_camera_target |
| 78 | MusicTrigger | 117 | player_set_respawn_target_transform |
| 79 | MusicMarker | 118 | player_force_base_respawn |
| 80 | NavpointRequest | 119 | PlayerEmote |
| 81 | initiate_mobility_action | 120 | biped_pickup_item_request |
| 82 | equipment_teleport_request | 121 | game_engine_request_boot_player |
| 83 | FOBClientInput | 122 | player_loadout_request |
| 84 | Dialogue2D | 123 | biped_debug_teleport |
| 85 | AIDialog | 124 | biped_dodge |
| 86 | CampaignMapStateUpdate | 125 | vehicle_auto_turret_choose_target |
| 87 | BetrayResponse | **126** | **unit_zoom** |
| 88 | ai_jump | 127 | authority_ignored_predicted_position |

---

## Grammaire bit-exacte à confronter au film

Pour un paquet, après le `R(7)` de type :

```
R(7) type
pour i dans 0,1,2 :
    R(1) porte                                        (0 = référence nulle 0xFFFFFFFF)
    si porte == 1 :
        si domaine(i) == 1 : R(1) sonde
        R(ceil(log2(cardinal[domaine effectif])))     -> index relatif
        R(2)                                           -> génération
charge utile (+0x68)
R(1) queue ; si 1 : R(32)
```

Prédictions par type (les 3 références présentes) :

- **114 `biped_board_vehicle`** : 7 + (1+8+2) + (1+8+2) + (1+13+2) + **6** + 1 = **52 bits**.
  Référence 0 : bits 8..15 = index dans la bande 2, bits 16..17 = génération.
- **126 `unit_zoom`** : 7 + (1+9+2) + (1+13+2) + (1+13+2) + **2** + 1 = **43 bits**.
  Référence 0 : bits 8..16 = index dans la bande 4, bits 17..18 = génération.
- **103 `unit_exit_vehicle`** : 7 + (1+[1]+13|9+2) × 2 + (1+13+2) + **6** + 1.

L'observation empirique du film 00162144 (« bits 8..15 constants, puis variation ») s'aligne
exactement sur la référence 0 d'un paquet 114 (bande 2, 8 bits d'index à partir du bit 8, suivis de
2 bits de génération variables). Pour le 126, la référence 0 occuperait les bits 8..16.

---

## Conclusions

- **B1 : résolu.** Table `0x1451f98d0`, 9 entrées, valeurs calculées par `FUN_140d10bb0` à partir
  de `DAT_144706100`, largeur `ceil(log2(cardinal))`, sortie = handle `(génération << 30) | index`,
  porte = « référence non nulle ». Sonde présente **uniquement** pour le domaine 1. Site d'appel
  prouvé au désassemblage. Réserve runtime documentée en B1.6.
- **B2 : partiellement résolu.** (a) entrée et sortie de véhicule = deux types distincts (114 et
  103) ; en revanche la lunette a un type unique, le 126 `unit_zoom`, dont le champ porte le sens.
  (b) ref0 = unité, ref1 = véhicule, ref2 = contexte non contraint. (c) le `R(6)` est présent en
  entrée comme en sortie : identifiant de siège très probable. **Les sites d'émission ne sont pas
  atteints** : l'émission est générique et indexée par numéro de type.
- **B3 : résolu.** Le domaine 1 n'est pas le joueur : c'est le domaine générique des objets, avec
  codage court pour la bande unités+véhicules.

## Incertitudes et pièges signalés

1. **Le type de l'événement de lunette est remis en cause.** Le contexte du chantier tenait le 114
   pour l'événement de lunette (chronologie Theater, 11/12 transitions sous la seconde). Les
   preuves statiques de ce lot disent que le 114 s'appelle `biped_board_vehicle` et que le
   **126 `unit_zoom`** est l'événement de lunette. Les deux constats ne se concilient pas d'eux-
   mêmes. **Test discriminant proposé** : rechercher dans le film des paquets de type 126 et
   vérifier que leur champ de 2 bits alterne (valeur brute 1 → niveau 0 à l'entrée en lunette,
   valeur brute 0 → −1 à la sortie) au rythme des transitions déjà horodatées à la main. Si les
   paquets 126 sont absents du film, alors le 114 observé mérite une seconde lecture, en vérifiant
   d'abord l'alignement du `R(7)` de tête.
2. **Sémantique des bandes : inférence, pas preuve.** Aucune table nommée ne dit « bande 2 =
   unités ». La carte repose sur le recoupement de 11 types dont les noms sont explicites. Elle est
   cohérente sur tout l'échantillon, mais une contre-observation reste possible.
3. **`R(6)` = index de siège : très probable, non prouvé.** Le champ n'est nommé nulle part.
4. **ref2 (domaine 7) non identifiée.** Elle est en domaine 7 pour tous les types examinés, ce qui
   suggère un rôle structurel (objet de contexte, cause, autorité) plutôt qu'un rôle propre à
   chaque type.
5. **Borne du dispatcher : 123, pas 128.** `14080aa37: CMP R15,0x7b ; JNC ...` — les types valides
   vont de 0 à 122 au moment du dispatch, alors que la table statique compte 128 entrées et que les
   types 123..127 y portent des descripteurs nommés valides. **Cette divergence n'est pas expliquée
   et elle touche directement `unit_zoom` (126)** : à vérifier avant de conclure sur l'absence de
   paquets 126 dans un film (il est possible que la table runtime recopiée à `+0x210` soit indexée
   différemment, ou que le compte `[objet+0x208]` diffère de 128).
6. **Les types 0..49 de la table statique ne sont pas résolus** par la méthode `getName` (valeurs
   non-pointeurs à certains index). La base `0x144724A90` est néanmoins confirmée par trois points
   indépendants (105, 114, 126).
7. `unit_switch_seat` sans charge utile : le siège cible doit passer par une référence. À éclaircir
   si le décodage des changements de siège devient nécessaire.

## Suite proposée (lot C)

1. Confronter la grammaire bit-exacte ci-dessus au film 00162144 en cherchant **les paquets de type
   126**, et lire leur champ de 2 bits.
2. Trancher la borne 123 vs 128 (lire `[objet+0x208]` et la construction de la table runtime par
   `FUN_14025fda0`), car elle conditionne la lisibilité même du type 126.
3. Identifier la fonction d'émission générique (motif d'indexation `[objet+0x210 + type*8]` en
   contexte d'écriture) puis ses appelants passant `0x7e` / `0x72`, pour boucler B2.5.

## État de l'outillage à la clôture du lot

Le greffon Ghidra a cessé de répondre en fin de lot (requêtes triviales sans réponse après trois
tentatives de 90 s), très probablement saturé par une recherche d'instructions sur tout le binaire
(`/search_instructions` sur un opérande courant). Les points 2 et 3 de la suite proposée n'ont donc
pas pu être instruits. **Redémarrer Ghidra et son greffon avant le lot C**, et éviter
`/search_instructions` sur des opérandes très fréquents (`0x210`, `0x72`) : préférer une remontée
par xrefs successives. Tous les résultats consignés ci-dessus ont été obtenus avant l'incident et
sont reproductibles par simple lecture mémoire et décompilation ponctuelle.
