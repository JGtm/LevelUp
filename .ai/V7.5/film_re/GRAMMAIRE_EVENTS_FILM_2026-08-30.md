# Grammaire des événements de film : en-tête bit à bit, table des types corrigée, charges utiles

Date : 2026-08-30. Lot E du chantier « visée lunette ». Rétro-ingénierie **pure, lecture seule**
sur `HaloInfinite.exe` (greffon Ghidra HTTP `127.0.0.1:8089`). Aucun code du dépôt n'a été modifié,
aucun fichier d'un autre agent n'a été touché.

Notes précédentes : `NOTE_ENVELOPPE_EVENTS_2026-08-30.md` (lot B1), `NOTE_EMETTEUR_114_2026-08-30.md`
(lot B2). **Cette note invalide la table des types de ces deux notes** (voir E5) : la base
`0x144724A90` n'est pas la table des descripteurs, et l'index de cette zone n'est pas le numéro de
type. Les résultats de grammaire de B1/B2 (var-int, domaines, porte, bit de continuation) restent
valides et sont ici reconfirmés au désassemblage.

Toutes les adresses sont des adresses virtuelles du binaire retail analysé.

---

## Résumé exécutif

1. **E1 — il n'y a AUCUN bit entre le `R(7)` du type et la première référence.** Le bit que le
   dépôt observe n'est pas *après* le type : c'est le **bit de poids faible du type lui-même**. Le
   bit supplémentaire du paquet est **avant** le type — c'est le drapeau de continuation de la liste
   d'événements, lu par la boucle appelante `FUN_14076a1c4` (candidat **(d)**). Prouvé au
   désassemblage des deux fonctions.
2. **E2 — grammaire d'en-tête close** : `[1 bit continuation] R(7) type` puis, sans rien
   d'intermédiaire, les trois références gardées, puis la charge utile du type. Le premier octet
   d'un événement calé sur un octet vaut donc exactement **`0x80 | type`**.
3. **E5 — découverte majeure, et elle change tout : la table des descripteurs n'est pas
   `0x144724A90`, et le numéro de type n'est pas un index dans une zone contiguë.** La table
   runtime est écrite **entrée par entrée** par `FUN_140e453b4` : c'est une **permutation** de
   123 objets descripteurs dispersés en `.rdata`. J'ai reconstruit la bijection complète
   **type 0..122 → descripteur → nom** (annexe A). Les catalogues de B1/B2 sont faux ; en
   particulier **`biped_board_vehicle` est le type 8** (et non 114), **`unit_zoom` est le type 21**
   (et non 126), **`action_weapon_fire` est le type 36** (et non 105).
4. **E4 — `NavpointRequest` n'est PAS un conteneur** et **n'est pas le type 80**. C'est le
   **type 108** ; sa charge utile est `R(32) + R(32)` (64 bits fixes), la première étant une
   chaîne-id nommée `requested-event` lue par le lecteur générique de 32 bits. Il désigne un
   événement de jeu par identifiant, il n'en encapsule aucun : aucun ré-entrée dans la table de
   dispatch. Le **type 80 est `networked_ai_effect`**.
5. **Il n'existe que 123 types (0..122).** Le « type 126 » cité par le dépôt **ne peut pas
   exister** : le dispatcher rejette tout type ≥ 123 (`CMP R15,0x7b ; JNC <erreur>`).
6. **Deux tests de falsification immédiats** sont fournis en E6 pour trancher, côté film, si les
   « paquets » du dépôt sont bien des en-têtes d'événements calés sur un octet.
7. **Piège nouveau et sérieux** : la charge utile de certains types est **conditionnée par un
   drapeau de version/fonctionnalité** (`FUN_141102ed0(id)`), donc la grammaire n'est pas
   universelle d'un build à l'autre. Exemple prouvé : `MusicMarker` (type 114) lit `R(32)`
   **ou rien**.
8. **E7 — l'ancre `0xD2` ne recale pas la table, elle change de nom.** `FUN_14080C1F8` est à
   l'indice **36 et nulle part ailleurs** ; la numérotation est verrouillée par une seconde route
   (cardinal 123 stocké en `+0x208`, juste avant la table en `+0x210`). Les paquets `0xD2` sont donc
   le type **82 `PlayerGameEventSmall`**, un **sac de propriétés nommées** — porteur naturel
   d'identifiants d'arme. Les trois ancres empiriques passent d'**absurde à plausible** sous la
   table corrigée (notamment `0xA0` = `unit_teleported`, un record **position + orientation**,
   cohérent avec 85 % du flux).
9. **E8 — la lunette n'est pas dans la bobine.** `unit_zoom` = type 21 → premier octet `0x95`, que
   le recensement du corpus n'a jamais vu. Aucun événement de zoom dans les 1 369 films : le zoom
   doit être cherché hors des événements.

---

## E1. Que signifie le bit qui suit le `R(7)` du type ?

### E1.1 Réponse

**Rien : il n'y a pas de bit après le `R(7)`.** Le premier bit consommé après le type est la porte
de la référence 0. Le bit que la mesure du dépôt attribue à « après le type » est en réalité :

- soit le **bit 0 du numéro de type** (si la fenêtre de lecture du dépôt est calée un bit trop tôt,
  c'est-à-dire sur le bit de continuation) ;
- et le bit *supplémentaire* réellement présent dans le flux est le **drapeau de continuation de la
  liste**, situé **avant** le type, et lu non pas par le dispatcher mais par sa boucle appelante.

C'est le candidat **(d)** de l'énoncé, à ceci près que le bit est **avant** le type, pas après.
Les candidats (a) variante de record, (b) « une autre référence suit » et (c) compression/delta
sont **réfutés** : aucune instruction ne lit de bit entre le `R(7)` et la porte de la référence 0.

### E1.2 Preuve 1 — le dispatcher `FUN_14080a9d4` lit 7 bits, puis directement la porte

Désassemblage (extrait littéral, `RDI` = flux de bits, `+0x30` = accumulateur calé à **gauche**,
`+0x38` = bits déjà consommés dans la fenêtre de 64, `+0x2c` = compteur global de bits) :

```
14080a9ef: MOV  R9D,dword ptr [R8 + 0x38]     ; bits déjà consommés
14080a9f6: MOV  R11D,0x40
14080a9fe: MOV  R8D,R11D
14080aa03: SUB  R8D,R9D                       ; bits restants dans la fenêtre
14080aa09: MOV  RBP,qword ptr [RDI + 0x30]    ; accumulateur
14080aa0d: LEA  R14D,[R11 + -0x39]            ; 0x40-0x39 = 7  <-- LARGEUR DU TYPE
14080aa11: CMP  R8D,R14D
14080aa14: JL   0x14080ab57                   ; recharge si moins de 7 bits
14080aa1a: ADD  dword ptr [RDI + 0x2c],R14D   ; compteur += 7
14080aa1e: MOV  RAX,RBP
14080aa21: SHL  RAX,0x7                       ; consomme 7 bits
14080aa25: MOV  qword ptr [RDI + 0x30],RAX
14080aa29: LEA  EAX,[R9 + 0x7]
14080aa2d: MOV  dword ptr [RDI + 0x38],EAX
14080aa30: SHR  RBP,0x39                      ; RBP >> 57 = les 7 bits de tête = LE TYPE
14080aa34: MOVSXD R15,EBP
14080aa37: CMP  R15,0x7b                      ; type < 123
14080aa3b: JNC  0x1422701d0                   ; sinon : erreur
14080aa41: MOV  RAX,qword ptr [R10 + 0x18]
14080aa45: MOV  R15,qword ptr [RAX + R15*0x8 + 0x210]   ; TABLE RUNTIME
14080aa4d: CALL 0x1406cb0cc                   ; predicat global, AUCUN bit (voir ci-dessous)
14080aa52: TEST AL,AL
14080aa54: JZ   0x142270184
14080aa5a: MOV  R8D,EBX
14080aa5d: MOV  RDX,R15
14080aa60: MOV  ECX,EBP
14080aa62: CALL 0x14080abe4                   ; alloue le slot de reception, AUCUN bit
14080aa67: MOV  R14,RAX
...
14080aa9b: MOV  RCX,RDI
14080aa9e: CALL 0x1406cf008                   ; <-- PREMIER BIT APRES LE TYPE = porte de la ref 0
14080aaa3: TEST AL,AL
14080aaa5: JNZ  0x14080ab22                   ; porte=1 -> var-int
14080aaa7: ADD  RBX,0x4
14080aaab: INC  R12D
14080aaae: MOV  qword ptr [RSP + 0x8c8],RBX
14080aab6: CMP  R12D,0x3                      ; boucle sur i = 0,1,2
14080aaba: JL   0x14080aa9b
```

Les deux seuls appels intercalés entre le type et la porte **ne peuvent pas consommer de bit** :

- `0x1406cb0cc` : `FUN_1406cb0cc(void)` — **sans aucun argument**, elle lit un global
  (`DAT_1445a78a0`) et appelle `FUN_14059ad48`. Elle ne reçoit pas le flux. **Preuve par absence
  d'argument.**
- `0x14080abe4` : appelée avec `RCX = type`, `RDX = descripteur`, `R8D = EBX`. **Le pointeur de
  flux `RDI` ne lui est pas passé** ; elle alloue une entrée de 0x30 octets dans le tampon
  `[0x1451f9898]` et interroge `descripteur->vtable+0x10` pour la taille de la charge utile.
  Même preuve par absence d'argument.

### E1.3 Preuve 2 — le bit supplémentaire est lu par la boucle appelante `FUN_14076a1c4`

```
14076a1ec: MOV  EDX,dword ptr [RDI + 0x38]
14076a1ef: CMP  EDX,0x40
14076a1f2: JNC  0x14076a256               ; fenêtre épuisée -> recharge (0x1406d6c7c)
14076a1f4: MOV  RCX,qword ptr [RDI + 0x30]
14076a1f8: INC  dword ptr [RDI + 0x2c]    ; compteur += 1   <-- UN SEUL BIT
14076a1fb: LEA  RAX,[RCX + RCX*0x1]       ; accumulateur <<= 1
14076a1ff: SHR  RCX,0x3f                  ; CL = bit de tête (MSB)
14076a203: MOV  qword ptr [RDI + 0x30],RAX
14076a207: TEST CL,CL
14076a209: LEA  EAX,[RDX + 0x1]
14076a20c: MOV  dword ptr [RDI + 0x38],EAX
14076a20f: JNZ  0x14076a230               ; bit == 1 -> un evenement suit
14076a211: ...                            ; bit == 0 -> FIN DE LISTE, sortie
14076a230: TEST EBP,EBP
14076a232: JLE  0x14076a268               ; budget d'evenements epuise -> code 3
14076a234: MOV  ECX,dword ptr [RSI + 0x14]
14076a237: CALL 0x140514010
14076a23c: MOV  RCX,qword ptr [RSI + 0x28]
14076a240: MOV  R8,RDI                    ; le flux
14076a243: MOV  EDX,EAX
14076a245: MOV  RCX,qword ptr [RCX + 0x8]
14076a249: CALL 0x14080a9d4               ; LE DISPATCHER
14076a24e: MOV  EBX,EAX
14076a250: TEST EAX,EAX
14076a252: JZ   0x14076a1ec               ; retour a la lecture du bit suivant
```

**La grammaire de la liste est donc `( 1 <evenement> )* 0`.** Le bit appartient à la boucle, pas au
record ; mais il est physiquement présent en tête de chaque paquet, et l'émetteur l'écrit dans le
paquet mis en cache (`local_f8 = type | 0x80` avec 8 bits utiles, lot B2). Les deux côtés
concordent, la lecture est MSB en premier.

### E1.4 Ce que devient la grammaire du record selon le candidat retenu

| Candidat | Conséquence sur la grammaire | Statut |
|---|---|---|
| (a) variante du record | il faudrait deux lecteurs pour un même type | **RÉFUTÉ** : un seul lecteur `+0x68` par type, appelé sans discriminant |
| (b) « une autre référence suit » | le nombre de références serait variable | **RÉFUTÉ** : boucle bornée en dur `CMP R12D,0x3` |
| (c) compression / delta | un chemin de décodage alternatif existerait | **RÉFUTÉ** : aucun branchement conditionnel entre le type et la porte |
| (d) bit de la boucle appelante | `( 1 event )* 0` ; le record commence au `R(7)` | **PROUVÉ** |

**Conséquence arithmétique directe pour le dépôt.** Si le décodeur cale son `R(7)` sur le bit de
continuation, il lit `0x40 | (T >> 1)` au lieu de `T`, et son « bit bas de variante » **est le bit 0
du type**. Deux valeurs de premier octet consécutives (`0xD2`/`0xD3`) ne sont alors **pas deux
variantes d'un même événement mais deux types différents** — ce qui explique exactement la mesure
citée (champ propre à 9 valeurs sur l'un, bruit à 447 valeurs sur l'autre) : deux types, deux
dispositions de charge utile.

---

## E2. La grammaire exacte de l'en-tête commune, bit par bit

Du premier bit du paquet au premier bit de la charge utile spécifique au type :

```
[1]   bit de continuation            INCONDITIONNEL, ecrit/lu avant le type
                                     1 = un evenement suit ; 0 = fin de la liste
R(7)  numero de type                 INCONDITIONNEL, MSB en premier, valeur < 123

pour i = 0, 1, 2 :                   TROIS references, nombre FIXE (boucle bornee a 3)
  R(1)  porte                        INCONDITIONNEL
        0 -> reference nulle (handle 0xFFFFFFFF), aucun autre bit
        1 -> :
          [R(1) sonde]               GARDE : uniquement si domaine(i) == 1
          R(w)  index relatif        w = ceil(log2(cardinal[domaine effectif]))
          R(2)  generation           INCONDITIONNEL des lors que la porte vaut 1

<charge utile du type>               lecteur = descripteur->vtable + 0x68
[R(1) ; si 1 : R(32) == 0x0BCDDCBA]  sentinelle de DEBOGAGE seulement
                                     (garde : FUN_14076cea8() && DAT_1450e2520 == 0)
```

**Champs inconditionnels** : le bit de continuation, les 7 bits de type, les 3 bits de porte.
**Champs gardés** : la sonde (domaine 1 seulement), l'index et la génération (porte = 1), la
sentinelle (mode débogage).

**Il n'y a aucun champ intermédiaire** entre le type et la première porte, ni entre les références
et la charge utile. Le domaine `d(i)` est rendu par `descripteur->vtable+0x58(i)` — une petite
fonction `switch(i)` sans lecture de bits — et passé en 3e argument (`R8D`) du var-int
`0x1406d3140` (site d'appel `14080ab22..14080ab45`, réétabli ci-dessus).

Largeurs par domaine (table `0x1451f98d0`, inchangée par rapport à B1) :

| Domaine | Largeur `R(w)` | Coût total d'une référence présente |
|---|---|---|
| 0 | 13 | 16 bits |
| 1 | 13 (ou 9 si sonde = 1) | 17 bits (ou 13) |
| 2 | 8 | 11 bits |
| 3 | 8 | 11 bits |
| 4 | 9 | 12 bits |
| 5 | 8 | 11 bits |
| 6 | 9 | 12 bits |
| 7 | 13 | 16 bits |
| 8 | 13 | 16 bits |

Une référence absente coûte **1 bit**. Un en-tête complet coûte donc entre **8 + 3 = 11 bits**
(trois références nulles) et **8 + 48 = 56 bits** (trois références en domaine 0/7/8).

**Conséquence pratique n°1, à retenir** : si un événement commence sur une frontière d'octet, son
premier octet vaut **`0x80 | type`**, avec `type < 123`. Donc le premier octet appartient
**forcément à `0x80..0xFA`**.

**Conséquence pratique n°2** : les événements sont concaténés au **bit** près, pas à l'octet.
Seul le premier événement d'un bloc aligné a un premier octet interprétable directement ; pour
avancer d'un événement au suivant il faut décoder l'en-tête ET la charge utile — il n'existe
**aucun champ de longueur** dans le paquet (confirmé en B2b, reconfirmé ici : rien entre le type et
les références).

---

## E3. Longueurs et grammaires des charges utiles

Méthode : `type -> objet descripteur (E5) -> vtable -> +0x68 lecteur`, puis décompilation du
lecteur. La colonne « taille » est la taille de la structure de réception en octets, lue dans le
thunk `vtable+0x10` (`MOV EAX,imm32 ; RET`) — c'est le tampon alloué par le dispatcher, pas la
longueur en bits, mais elle borne utilement le nombre de champs.

Primitives de lecture identifiées (toutes vérifiées sur pièces) :

| Adresse | Rôle |
|---|---|
| `0x1406cf008` | `R(1)` |
| `0x1406d84b4` | `R(n)` générique, largeur passée sur la pile |
| `0x14080d6f0`, `0x142ef49b8`, `0x14080dec4` | `R(32)` (la variante `14080dec4` reçoit en plus un nom de champ de débogage) |
| `0x14080d69c` | `R(1) porte ; si 1 : R(32) ; sinon valeur par défaut` |
| `0x1406d676c` | `R(64)` |
| `0x1407f0094` | `R(16)` |
| `0x1407f2058` | `R(1) porte ; si 1 : R(5)` |
| `0x14080cb98` | `R(2)` puis **valeur − 1** (résultat dans {−1,0,1,2}) |
| `0x14080cb50` | `R(1) porte ; si 1 : R(n)` |
| `0x1406d3140` | référence var-int (voir E2) — **présente aussi à l'intérieur de certaines charges utiles** |
| `0x1406cd5b8` | lecture quantifiée composite (`R(1)`, `R(1)`, `R(5)`…) |
| `0x1408d8220` | lecteur vide (`return 1`) — **0 bit** |
| `0x141102ed0(id)` | **drapeau de version/fonctionnalité** : garde certaines charges utiles |

### E3.1 Les types demandés par l'énoncé, lus comme numéros de type réels (0..122)

| Type | Nom (`getName`) | Lecteur `+0x68` | Taille | Grammaire de la charge utile |
|---|---|---|---|---|
| 64 | `player_set_respawn_target_transform` | `0x142f17348` | 32 | `R(6)` + `R(1)` + `R(64)` (transformée) — **variable**, min 6 bits |
| 68 | `supply_request` | `0x142eec01c` | 8 | `R(8)` + `R(1) ; si 1 : R(5)` — **variable**, min 9 bits |
| 69 | `CampaignMapStateUpdate` | `0x142ef7fdc` | 20 | `R(4)` + `R(1)` + `R(32)` — **variable**, min 5 bits |
| 70 | `AIPhase` | `0x142f15cf8` | 52 | `R(2)` + `R(n)` générique — **variable**, min 2 bits |
| 71 | `AIRequestIdleTransitionTime` | `0x142f15e70` | 4 | un seul `R(n)` générique (largeur sur la pile) — **fixe**, largeur non relevée |
| 80 | `networked_ai_effect` | `0x142ef8f74` | 44 | `R(2)` + `R(1)` + `R(32)` — **variable**, min 2 bits |
| 82 | `PlayerGameEventSmall` | `0x14080add8` | 168 | `R(32)` + `R(8)` + sous-lecteurs + `R(1)` final — **variable**, min 41 bits |
| 97 | `SaveToUGCService` | `0x142f16544` | 828 | `R(6)`, `R(32)`, `R(32)`, `R(64)` + chaîne — **variable** |
| 96 | `NetworkedCrewEventType` | `0x142f162c8` | 16 | `R(6)` + sous-lecteurs — **variable**, min 6 bits |
| 98 | `Equipment` | `0x142eebd68` | 8 | `R(8)` + `R(1)` — **fixe 9 bits** (lecteur autonome, aucun appel) |
| 99 | `SelectedSpawnZoneChangedEvent` | `0x142f167d4` | 24 | `R(1) ; si 1 : R(5)` + `R(32)` « associated-participant-handle » + `R(32)` « zone-name » — **variable** |
| 101 | `LoadForgeObjectGroup` | `0x142f16174` | 712 | `R(32)` + `R(1) ; si 1 : R(5)` + charge Forge — **variable** |
| 103 | `EquipmentSpawnedObject` | `0x1408d8220` | 0 | **aucun bit** |
| 105 | `EquipmentObjectKnockedBack` | `0x141118a00` | 4 | `R(1)` + `R(32)` (handle) — **variable**, min 1 bit |
| 107 | `player_forge_user_string_action` | `0x142ef9040` | 608 | chaîne utilisateur — **variable** |
| 110 | `ObjectDeterministicDamageAcceleration` | `0x142f163c8` | 36 | `R(10)` + `R(1)` + … — **variable**, min 10 bits |
| 114 | `MusicMarker` | `0x142ef87f0` | 4 | **`si FUN_141102ed0(0x72) : R(32) ; sinon rien`** — 0 ou 32 bits selon un drapeau de build |
| 115 | `synchronized_teleport` | `0x142ef9284` | 44 | `R(2)` + `R(1) ; si 1 : R(32)` + `R(64)` — **variable**, min 2 bits |
| 116 | `teleport_effects` | `0x142ef93e0` | 56 | `R(1)` + `R(1) ; si 1 : R(32)` + `R(64)` — **variable**, min 1 bit |
| 121 | `PlayerForgeableCustomAction` | `0x141fd8740` | 8 | `R(1)` + `R(1) ; si 1 : R(5)` — **variable**, min 2 bits |
| **126** | — | — | — | **N'EXISTE PAS.** Le dispatcher rejette tout type ≥ 123 |

### E3.2 Types du chantier « visée » et de la mesure du film (numéros corrigés)

| Type | Nom | Lecteur | Taille | Grammaire | Statut |
|---|---|---|---|---|---|
| 8 | `biped_board_vehicle` | `0x142f168c0` | 4 | **`R(6)`** — fixe | PROUVÉ (lecteur + écrivain désassemblé en B2) |
| 21 | `unit_zoom` | `0x141168b28` | 4 | **`R(2)` puis valeur − 1** — fixe 2 bits | PROUVÉ |
| 22 | `unit_exit_vehicle` | `0x142f17b94` | 8 | `R(6)` **puis** `R(1)` + suite (`0x140c1e31c`) — variable, min 7 bits | PROUVÉ pour `R(6)` et `R(1)` ; la suite non détaillée |
| 32 | `unit_teleported` | `0x142ef9470` | 32 | `R(1) ; si 1 : R(32)` + transformée — variable | partiel |
| 33 | `vehicle_auto_turret_choose_target` | `0x1408d8220` | 0 | **aucun bit** | PROUVÉ |
| 35 | `request_weapon_fire` | `0x142f17500` | 808 | **variable**, gros record : contient `R(1)`, `R(2)`, `R(4)`, `R(16)`, `R(30)`, `R(32)`, trois `R(6)`, **et des références var-int internes** (`0x1406d3140`) | partiel |
| 36 | `action_weapon_fire` | `0x14080c1f8` | 808 | idem, avec en plus `R(1)`×3 et un `R(2)` — **c'est bien le record de dégât** que le dépôt a validé 59/59 | partiel |
| 54 | `unit_switch_seat` | `0x1408d8220` | 0 | **aucun bit** | PROUVÉ |
| 83 | `TeamGameEvent` | `0x142f1686c` | 312 | `R(9)` + `R(32)` + `R(8)` + sous-lecteurs — variable | partiel |
| 100 | `PowerUpApplied` | `0x142ef8a64` | 12 | **`R(1) ; si 1 : R(32)` + `R(32)` « variant-name » + `R(1) ; si 1 : R(32)`** | PROUVÉ (3 dwords = taille 12) |
| 104 | `EquipmentKnockbackPlayer` | `0x14116c344` | 12 | `R(1)` + suite (`0x14076d528`) — variable | partiel |
| 108 | `NavpointRequest` | `0x142ef8a40` | 8 | **`R(32)` « requested-event » + `R(32)`** — **fixe 64 bits** | PROUVÉ |

Treize types ont le lecteur vide `0x1408d8220` et donc **zéro bit de charge utile** : 3, 4, 23, 24,
25, 26, 33, 49, 54, 57, 59, 92, 103.

---

## E4. Le type 80 et `NavpointRequest`

### E4.1 La résolution de nom du dépôt est fausse — deux fois

Le dépôt a résolu « type 80 » par `0x144724A90 + 80*8 = 0x144724D10`, dont le contenu est la vtable
`0x143d09f38`, dont le `getName` (`0x14119e1d0`) rend bien **`NavpointRequest`**. Le **nom attaché à
cette vtable est donc correct**, mais l'**association au type 80 ne l'est pas** :

- la zone `0x144724A90` n'est pas la table des descripteurs (E5) ;
- et l'objet `0x144724D10` est enregistré par `FUN_140e453b4` à l'offset `+0x570`, soit le type
  **`(0x570 − 0x210) / 8 = 108`**.

**`NavpointRequest` est le type 108. Le type 80 est `networked_ai_effect`** (objet `0x144724cf8`
— enregistré à `+0x490`, soit `(0x490 − 0x210) / 8 = 80` ; vtable `0x143d09d70`, lecteur
`0x142ef8f74`, 44 octets).

L'incohérence signalée par l'énoncé (« NavpointRequest à 85 % du flux ») **n'a donc pas à être
expliquée : elle provient d'une résolution de nom erronée**, pas d'une bizarrerie du moteur.

### E4.2 `NavpointRequest` est-il un conteneur ? Non.

Chaîne complète du lecteur :

```c
undefined1 FUN_142ef8a40(p1, p2, param_3, param_4) { FUN_142ef4a98(param_3, param_4); return 1; }

undefined8 FUN_142ef4a98(uint *param_1, longlong param_2)   // param_1 = charge utile, param_2 = flux
{
  FUN_14080dec4(param_2, "requested-event", param_1 + 1);   // R(32)  -> charge+0x04
  ...                                                       // R(32)  -> charge+0x00
  *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 0x20;
  *param_1 = uVar5;
}
```

`FUN_14080dec4(flux, nom, sortie)` est **le lecteur générique de 32 bits** : son corps est
identique, instruction pour instruction, au bloc `R(32)` en ligne qui suit (même incrément
`+0x2c += 0x20`, même BSWAP de rechargement). Le paramètre `"requested-event"` est **un nom de champ
de débogage**, sans effet sur le flux.

**Conclusion : `NavpointRequest` = `R(32)` (chaîne-id de l'événement visé) + `R(32)` = 64 bits
fixes.** Il **désigne** un événement par identifiant de chaîne, il n'en **encapsule** aucun :

- aucune ré-entrée dans la table de dispatch (`+0x210`) ;
- aucun appel d'un lecteur `+0x68` ;
- taille de réception de 8 octets = exactement deux `uint32`.

C'est un ordre d'affichage de point de navigation associé à un identifiant d'événement de jeu.
**Il n'y a pas de « clef de tout le flux » de ce côté.**

---

## E5. La table utilisée à l'exécution : la découverte qui invalide les catalogues précédents

### E5.1 Ce que la zone `0x144724A90` est réellement

- `get_xrefs_to 0x144724A90` ne rend **qu'une seule référence** : `From 140260920 in FUN_14025fda0`.
  `FUN_14025fda0` est le **registre des énumérations de script Forge**, et l'instruction en cause est
  `LEA RAX,[0x144724A90]` (octets `48 8D 05 69 41 4C 04`) dont le résultat est écrit à
  `0x1445C2BC0`, champ « tableau de valeurs » d'un enregistrement d'énumération dont le nom
  (champ suivant, `0x1445C2BC8`) est **`PERSONAL_AI_AVATAR_TYPE`**.
- Le contenu confirme que ce n'est pas une table de descripteurs : les entrées 0..45 sont des
  pointeurs hétérogènes (dont un zéro en 17 et un `1` en 37) et les entrées **46..49 sont des
  flottants** (`0x40400000` = 3.0, `0x3F800000` = 1.0, `0x3EB33333` = 0.35, `0x40E00000` = 7.0…).

La série homogène de vtables ne commence qu'à `0x144724C20` — soit 50 entrées plus loin. **La base
retenue par les lots B1/B2 était décalée**, et c'est ce décalage qui a produit le catalogue
« types 50..127 ».

### E5.2 La table runtime est une permutation explicite, écrite entrée par entrée

`0x144724C20` et le slot `0x144724dd8` sont tous deux référencés par **une seule et même
fonction, `FUN_140e453b4`**, qui écrit la table runtime :

```c
void FUN_140e453b4(longlong ctx, undefined4 *out_n1, undefined4 *out_n2)
{
  FUN_140e45fc4(ctx, 0,    &DAT_144f52890);      // 2e famille : ctx + 8 + i*8, i = 0..0x31
  FUN_140e45fc4(ctx, 1,    &PTR_PTR_144669df0);
  ...
  *(undefined ***)(ctx + 0x210) = &PTR_PTR_144724f80;   // type 0
  *(undefined ***)(ctx + 0x218) = &PTR_PTR_144724f78;   // type 1
  *(undefined ***)(ctx + 0x220) = &PTR_PTR_14473faa0;   // type 2
  *(undefined ***)(ctx + 0x5c0) = &PTR_PTR_14473fa90;   // type 118   <-- ordre non monotone
  ...
  *(undefined ***)(ctx + 0x330) = &PTR_PTR_144724dd8;   // type 36 = action_weapon_fire
  ...
  *out_n1 = 0x32;    // 50  : cardinal de la 2e famille
  *out_n2 = 0x7b;    // 123 : cardinal des types d'evenements
}
```

Faits établis :

1. **L'indexation est bien `[[ctx+0x18] + 0x210 + type*8]`** (dispatcher, `14080aa41..14080aa45`) et
   l'émetteur utilise la même (`FUN_1424d80bc`, lot B2). Une seule et même table.
2. Les offsets écrits vont de `+0x210` à `+0x5e0` par pas de 8 : **exactement 123 entrées**,
   `types 0..122`, **sans trou ni doublon** (vérifié : les 123 offsets extraits donnent les 123
   valeurs de type 0..122, chacune une fois). `*out_n2 = 0x7b` = 123 confirme le cardinal, et
   coïncide avec la borne `CMP R15,0x7b` du dispatcher.
3. **L'ordre n'est pas l'ordre mémoire** : par exemple le type 2 pointe `0x14473faa0` alors que le
   type 3 pointe `0x14473fa68` (adresse plus basse), et le type 118 est écrit entre les types 2 et 3
   dans le code. La correspondance type → objet est donc **une permutation qu'aucune arithmétique ne
   reconstitue** : elle doit être lue dans cette fonction.
4. Les objets descripteurs sont des **objets de 8 octets réduits à leur pointeur de vtable**, ce qui
   explique que `get_xrefs_to <vtable>` ne rende que le slot : le dispatcher fait `MOV RAX,[R15]`
   pour obtenir la vtable, puis `CALL [RAX+0x58]`. Certains objets sont **hors** de la plage
   contiguë : `0x14473fa58..0x14473faa0` (9 objets), `0x144724a58`, `0x144724a88`.
5. Une **seconde famille de 50 descripteurs** est enregistrée par `FUN_140e45fc4(ctx, i, ptr)` à
   `ctx + 8 + i*8`, `i = 0..0x31`. Elle est **distincte** des événements ; le cardinal 50 est rendu
   par `*out_n1`. Elle n'a pas été explorée dans ce lot.

### E5.3 Réponse à E5

**Non, la table statique et la table d'exécution ne coïncident pas**, ni par la base ni par
l'indexation. Toutes les résolutions de noms des lots B1/B2 sont fausses. Exemples de l'ampleur du
décalage :

| Nom | Type selon B1/B2 | **Type réel** |
|---|---|---|
| `action_weapon_fire` | 105 | **36** |
| `biped_board_vehicle` | 114 | **8** |
| `unit_zoom` | 126 | **21** |
| `unit_exit_vehicle` | 103 | **22** |
| `unit_switch_seat` | 62 | **54** |
| `weapon_empty_click` | 99 | **11** |
| `NavpointRequest` | 80 | **108** |
| `projectile_detonate` | 115 | **5** |

Cela explique aussi les « 125 embarquements en véhicule sur une carte sans véhicule » : le numéro
114 lu dans le film n'a jamais désigné `biped_board_vehicle` — le type 114 est `MusicMarker`.

---

## E6. Deux tests de falsification à exécuter côté film

La grammaire prouvée impose deux contraintes fortes et bon marché à vérifier :

1. **Test du premier octet.** Pour tout événement commençant sur une frontière d'octet, le premier
   octet vaut `0x80 | type` avec `type < 123`, donc il appartient à **`0x80..0xFA`** — jamais
   `0x00..0x7F`, jamais `0xFB..0xFF`. Si le corpus de paquets du dépôt contient des premiers octets
   hors de `0x80..0xFA`, alors ces « paquets » ne sont pas des en-têtes d'événements calés sur un
   octet, et toute la lecture du type est à revoir avant toute interprétation sémantique.
   *Indice déjà présent dans la mesure du dépôt* : un « type 126 » impliquerait un premier octet
   `0xFC`/`0xFD`, donc un type 124/125 — **impossible**.
2. **Test de la table corrigée.** En posant `T = premier_octet & 0x7F` et en nommant `T` par
   l'annexe A, la distribution des types doit être sémantiquement plausible pour le mode joué.
   Application aux valeurs citées dans l'énoncé :

| Premier octet | `T = octet & 0x7F` | Nom réel | Lecture « `octet >> 1` » du dépôt | Nom réel de ce numéro |
|---|---|---|---|---|
| `0xA0` (21 227) | 32 | `unit_teleported` | 80 | `networked_ai_effect` |
| `0xA1` | 33 | `vehicle_auto_turret_choose_target` | 80 | `networked_ai_effect` |
| `0xD2` (1 958) | 82 | **`PlayerGameEventSmall`** | 105 | `EquipmentObjectKnockedBack` |
| `0xD3` (459) | 83 | **`TeamGameEvent`** | 105 | `EquipmentObjectKnockedBack` |
| `0xC6` | 70 | `AIPhase` | 99 | `SelectedSpawnZoneChangedEvent` |
| `0xC7` | 71 | `AIRequestIdleTransitionTime` | 99 | `SelectedSpawnZoneChangedEvent` |
| `0xE4` | 100 | `PowerUpApplied` | 114 | `MusicMarker` |
| `0xE5` | 101 | `LoadForgeObjectGroup` | 114 | `MusicMarker` |
| `0xE8` | 104 | `EquipmentKnockbackPlayer` | 116 | `teleport_effects` |
| `0xE9` | 105 | `EquipmentObjectKnockedBack` | 116 | `teleport_effects` |

**Ce que la colonne `T = octet & 0x7F` explique très bien** : `0xD2` et `0xD3` sont **deux types
différents** (`PlayerGameEventSmall`, 168 octets, et `TeamGameEvent`, 312 octets), ce qui rend compte
sans effort du fait qu'une même fenêtre de bits donne 9 valeurs propres sur l'un et 447 valeurs sur
l'autre. De plus le lecteur de `PlayerGameEventSmall` **commence par un `R(32)`** (via
`FUN_14080ae70`) : un identifiant 32 bits ne prenant que **9 valeurs distinctes sur 1 958 paquets**
est la signature d'un **genre d'événement de jeu**, pas d'une arme. La lecture « champ arme » de ce
champ est donc très probablement une méprise.

**Ce qu'aucune des deux colonnes n'explique** : 21 227 paquets (85 %) sur un seul numéro, quel que
soit le modèle (`unit_teleported` comme `networked_ai_effect` sont invraisemblables à ce débit).
Cela suggère que la population majoritaire des « paquets » du dépôt **n'est pas un événement de
cette famille** — piste naturelle : la **seconde famille de 50 descripteurs** (`ctx+0x08+i*8`,
cardinal `0x32`) mise au jour en E5.2, qui tiendrait sur `R(6)`. À instruire dans un lot suivant.

---

## E7. Levée de la contradiction sur l'ancre 0xD2 (ajout après retour du pilote)

### E7.1 La question décisive : quel indice porte `FUN_14080C1F8` ?

**Réponse : l'indice 36, et lui seul.** Recherche exhaustive sur les 123 lecteurs `+0x68`
reconstruits : `0x14080c1f8` apparaît **une fois**, au type 36 `action_weapon_fire`. Le type 82
porte un autre lecteur, `0x14080add8` (`PlayerGameEventSmall`). Nous sommes donc dans la seconde
branche de l'alternative : **les paquets `0xD2` ne sont pas `action_weapon_fire`**.

### E7.2 Ma table ne peut pas être décalée : confirmation par une seconde route

Avant de conclure contre l'ancre, j'ai vérifié la numérotation par un chemin indépendant de
`FUN_140e453b4`. Son **appelant** `FUN_14054d014` construit l'objet et le dimensionne :

```c
DAT_144e61d88 = (undefined4 *)(DAT_144db4358 + 0x8aed0);   // l'objet table
memset((void *)(lVar1 + 0x8b0e0), 0, 0x400);               // 0x8aed0 + 0x210, 0x400 = 128 slots
FUN_140e453b4(DAT_144e61d88, local_res10, &local_res8);
DAT_144e61d88[0x82] = local_res8;                          // 0x82 * 4 = +0x208  <- cardinal 123
*puVar2 = local_res10[0];                                  // +0x00              <- cardinal 50
```

Trois faits verrouillent la numérotation :

1. la zone d'événements est **remise à zéro sur `+0x210` pour `0x400` octets**, soit 128
   emplacements dont 123 sont écrits — les 5 derniers restent nuls ;
2. le **cardinal 123 est stocké en `+0x208`**, immédiatement avant la table. Une table qui
   commencerait ailleurs ne placerait pas son compteur là. (Cela répond au passage à l'incertitude
   n°5 du lot B1, qui s'interrogeait sur `[objet+0x208]`.) ;
3. ce 123 est le même immédiat que la borne du dispatcher (`CMP R15,0x7b`).

**`type = (offset − 0x210) / 8` est donc établi par deux routes indépendantes.** Aucun décalage
n'est possible, et l'ancre `0xD2` ne peut pas servir à « recaler » la permutation.

### E7.3 Ce qu'est le type 82, et pourquoi son record porte des identifiants propres

`PlayerGameEventSmall` (168 octets, lecteur `0x14080add8`) est un **événement de jeu à sac de
propriétés nommées** :

```
R(32)   event-id (chaine-id du genre d'evenement)          -> charge+0x00
R(8)    ?                                                   -> charge+0x08
FUN_14080b1b8 (charge+0x10) :
        R(3)  nombre de proprietes (0..7)
        N x [ R(32) "property-name" (chaine-id) + R(3) tag de type + valeur ]
FUN_14080b034 (charge+0x78) :
        R(1) porte ; si 1 : R(32) nom + R(3) nombre + N x [ ... ]
```

`FUN_14080ef08` lit un `R(3)` de **tag de type** puis délègue à `FUN_14080eff0` : c'est une
**union étiquetée**, donc la longueur du record est **variable**.

C'est exactement le porteur attendu pour un événement de dégât ou de mort : un genre d'événement
(peu de valeurs distinctes) plus des propriétés nommées dont la **source de dégât / l'arme**. Deux
conséquences directes pour la mesure du dépôt :

- **« 9 identifiants propres sur 1 958 paquets » est la signature d'une chaîne-id**, pas d'un champ
  d'arme figé : soit l'`event-id` de tête, soit un `property-name`. Le décodage empirique du dépôt
  peut donc être **juste** tout en ayant **le mauvais nom de type** — le record décodé porte bien
  des identifiants d'arme, mais parce que c'est un sac de propriétés, pas parce que c'est
  `action_weapon_fire`.
- **La fenêtre fixe « bits 44..107 » n'est pas un champ.** Pour le type 82, les trois domaines sont
  0, 8 et 7 (fonction `0x142ef7f6c`, décodée : `i==0 -> 0`, `i==1 -> 8`, `i>=2 -> 7`), donc 13 bits
  d'index chacun : l'en-tête vaut 11, 26, 41 ou 56 bits selon les portes — **jamais 44**. Une fenêtre
  de 64 bits à décalage fixe qui ne rend que 9 valeurs est une fenêtre **majoritairement constante**,
  ce qui se produit sur n'importe quelle zone de faible entropie. Le champ doit être relu en suivant
  la grammaire, porte par porte.

### E7.4 Les trois ancres empiriques : d'absurde à plausible

C'est l'argument le plus fort en faveur de la table corrigée — chaque ancre citée comme
invraisemblable le devient beaucoup moins :

| Octet | Ancienne lecture (`>>1` + table décalée) | Verdict | Table corrigée (`& 0x7F`) | Verdict |
|---|---|---|---|---|
| `0xA0` (21 227 = 85 %) | `NavpointRequest` | absurde | **`unit_teleported` (32)** | **plausible** |
| `0xE5` (125, carte sans véhicule) | `biped_board_vehicle` | absurde | **`LoadForgeObjectGroup` (101)** | **plausible** |
| `0xD2` (1 958) | `action_weapon_fire` | plausible | **`PlayerGameEventSmall` (82)** | **plausible** |

Le cas `0xA0` est décisif, et je l'ai vérifié sur pièces. Lecteur du type 32 (`0x142ef9470`) :

```c
FUN_14076e494(flux, charge + 0x00, 0x10, 0, p5, 0);   // vecteur 3 composantes quantifie (12 octets)
FUN_14076e494(flux, charge + 0x0c, 0x10, 0, p5, 0);   // second vecteur 3 composantes (12 octets)
FUN_14080d69c();                                       // R(1) porte ; si 1 : R(32)
FUN_14080d69c();                                       // R(1) porte ; si 1 : R(32)
```

Taille de réception = **32 octets = 12 + 12 + 4 + 4** : l'ajustement est exact.
**`unit_teleported` porte une position et une orientation quantifiées.** Un flux à 85 % de
positions/orientations d'autorité est le comportement normal d'un flux de réplication — et cela
recoupe le phénomène de « warp » déjà observé par le dépôt dans les films. `NavpointRequest`
(deux `R(32)`) à 85 % n'avait, lui, aucun sens.

De même `0xE5` : la plupart des cartes multijoueur récentes sont **construites en Forge**, ce qui
rend `LoadForgeObjectGroup` parfaitement attendu — là où « 125 embarquements en véhicule sur une
carte sans véhicule » ne l'était pas.

### E7.5 Deux tests qui tranchent définitivement, exécutables sur le corpus

1. **Auto-cohérence du type 82.** Après l'en-tête (11, 26, 41 ou 56 bits selon les portes), lire
   `R(32)`, `R(8)`, puis `R(3)` : ce compteur doit être **exactement** le nombre de blocs
   `R(32) + R(3) + valeur` qui suivent, et la fin du record doit tomber sur le début de l'événement
   suivant (bit de continuation à 1). **Un mauvais alignement ne satisfait pas cette contrainte sur
   1 958 paquets.** C'est le test le plus discriminant disponible.
2. **Présence de l'octet `0xA4`.** `action_weapon_fire` est le type 36, donc premier octet
   `0x80 | 36 = 0xA4`. Question à poser au corpus : **`0xA4` existe-t-il, et à quel volume ?**
   Un événement de tir doit être fréquent. S'il est **totalement absent** des 41 M de paquets, alors
   les films ne contiennent pas cette famille d'événements et il faut chercher une autre table ;
   s'il est présent en volume, la table corrigée est confirmée et le pipeline killsource doit être
   rebranché sur `0xA4` (lecteur `FUN_14080C1F8`, 808 octets) plutôt que sur `0xD2`.

### E7.6 La lunette : l'octet attendu est `0x95`, et il est absent — conclusion pour le chantier

**Confirmé arithmétiquement** : `unit_zoom` = type 21 → premier octet `0x80 | 21 = 0x95`. Sous
l'ancienne lecture, `0x94` et `0x95` donnent tous deux `>>1 = 74`. Le recensement du pilote n'a
**jamais** vu le type 74 : **il n'y a donc aucun paquet `0x95` dans le corpus, donc aucun événement
`unit_zoom` dans les 1 369 films.**

Il faut le dire clairement : **la mise en lunette n'est pas dans la bobine sous forme d'événement.**
Ce résultat est cohérent avec les trois constats indépendants déjà au dossier :

- aucun désérialiseur de réplication n'écrit `unite+0x461` / `unite+0x462` (mesure du pilote) ;
- le seul écrivain réseau de `+0x462` est l'applicateur de `unit_zoom` (E8.2) ;
- et le consommateur `FUN_14076a484` retombe sur `+0x461` — **une valeur locale** — dès que
  l'override vaut `−1`.

Autrement dit, pour une unité distante l'état de lunette n'est **pas** reconstruit à partir du flux :
soit l'événement est filtré par pertinence à l'émission (le masque de clients cibles `slot+0x30` de
la chaîne d'émission, lot B2a.2), soit il n'est simplement pas émis vers les observateurs. **Le zoom
doit être cherché ailleurs que dans les événements** : état d'arme répliqué, champ de vision de la
caméra, ou pose/animation — pistes à instruire dans un lot dédié.

---

## E8. Retour sur la lunette : `unit_zoom` est le type 21, et la bande `unite+0x461/0x462`

### E8.1 Correction de numéro, à propager

Le chantier désigne l'événement de lunette par « type 126 ». **Ce numéro est faux** : il vient de la
table statique décalée (E5). **`unit_zoom` est le type 21** (objet `0x144724e80`, vtable
`0x143d0da50`, lecteur `0x141168b28`, applicateur `0x14110ec20`). Le type 126 **n'existe pas**.
Charge utile inchangée et confirmée : **`R(2)` puis valeur − 1**, soit `{−1, 0, 1, 2}`.

Cette correction va dans le sens de la consigne du pilote : le film n'a qu'un **niveau minuscule** à
transmettre, le grossissement venant de la définition de l'arme. Dans les grammaires de E3, les
candidats « lunette » sont donc les champs de **1 à 2 bits**, jamais un flottant — et le seul type
qui porte un tel champ comme charge utile entière est bien le 21.

### E8.2 Ce que fait le consommateur de `unite+0x461` / `unite+0x462` (croisement demandé)

L'offset `0x460` n'a pas été croisé, mais `0x461` et `0x462` le sont, ensemble, dans
`FUN_14076a484` — la fonction qui compose l'état par observateur, appelée depuis le même voisinage
que la boucle d'événements `FUN_14076a1c4` :

```c
lVar11 = FUN_140498800(local_d8, 0x1003);                          // composant unite (bipede/vehicule/type 12)
*(short *)(param_3 + 0x20) = (short)*(char *)(lVar11 + 0x461);     // valeur de base
if (*(char *)(lVar11 + 0x462) != -1) {                             // si l'override reseau est pose
  *(short *)(param_3 + 0x20) = (short)*(char *)(lVar11 + 0x462);   // il prime
}
*(undefined2 *)(param_3 + 0x6d) = *(undefined2 *)(lVar11 + 0x388);
```

Lecture directe :

- **`unite+0x462` est l'override, et c'est celui qu'écrit l'applicateur de `unit_zoom`** (prouvé en
  B2c.3). `−1` y signifie « pas d'override », ce qui recoupe exactement le `R(2) − 1` du lecteur.
- **`unite+0x461` est la valeur de repli** (niveau local / par défaut) utilisée quand l'override
  vaut `−1`.
- Les deux sont des **`char` signés**, étendus en `short` vers l'état d'observateur (`param_3+0x20`).
  Le domaine `{−1, 0, 1, 2}` est cohérent des deux côtés.

Cela conforte l'observation du pilote (aucun désérialiseur de réplication n'écrit cette bande) :
**le niveau de lunette ne voyage pas dans le flux d'état, il voyage dans l'événement `unit_zoom`
(type 21) et lui seul.** Un troisième champ de la même bande est lu un peu plus haut dans la même
fonction, `*(byte *)(lVar11 + 0x465)` — mais sur une autre structure (celle rendue par
`FUN_140769b90`), et son rôle n'est pas établi ; je le signale sans l'interpréter.

---

## Annexe A. Table complète et corrigée des 123 types d'événements

`type -> objet descripteur (slot 8 octets) -> vtable -> nom`. Lecteur = `vtable+0x68`.
Taille = tampon de réception en octets (`vtable+0x10`). Reconstruite intégralement depuis
`FUN_140e453b4` ; aucune extrapolation.

| # | Nom | Taille | Lecteur `+0x68` | Objet |
|---|---|---|---|---|
| 0 | damage_aftermath | 116 | 0x1407f15a4 | 0x144724f80 |
| 1 | damage_section_response | 28 | 0x140968368 | 0x144724f78 |
| 2 | restore_damage_section | 8 | 0x142ef90a4 | 0x14473faa0 |
| 3 | item_detonate | 0 | 0x1408d8220 | 0x14473fa68 |
| 4 | item_detonate_countdown | 0 | 0x1408d8220 | 0x14473fa70 |
| 5 | projectile_detonate | 84 | 0x1408096f8 | 0x144724e28 |
| 6 | projectile_impact_effect | 64 | 0x1410f03b4 | 0x144724e10 |
| 7 | projectile_object_impact_effect | 76 | 0x142f17474 | 0x144724e08 |
| 8 | **biped_board_vehicle** | 4 | 0x142f168c0 | 0x144724e20 |
| 9 | biped_pickup | 12 | 0x141037828 | 0x144724e18 |
| 10 | weapon_effect | 16 | 0x142f17eec | 0x144724db0 |
| 11 | weapon_empty_click | 24 | 0x142f17ffc | 0x144724da8 |
| 12 | biped_melee_clang | 40 | 0x142ef8e08 | 0x14473fa78 |
| 13 | motor_system_interruption | 12 | 0x142ef8f44 | 0x144724ce0 |
| 14 | PlayEffectOnObject | 4 | 0x142eebf08 | 0x144724c68 |
| 15 | Script | 108 | 0x14080bb4c | 0x144724c20 |
| 16 | ShowDebugText | 180 | 0x142eebf1c | 0x144724c50 |
| 17 | Allegiance | 12 | 0x142eebb3c | 0x144724c60 |
| 18 | MusicTrigger | 12 | 0x142ef8828 | 0x144724d00 |
| 19 | CollectibleUnlockEvent | 8 | 0x142ef8530 | 0x144724d78 |
| 20 | incident | 60 | 0x142f16fa0 | 0x144724f98 |
| 21 | **unit_zoom** | 4 | 0x141168b28 | 0x144724e80 |
| 22 | **unit_exit_vehicle** | 8 | 0x142f17b94 | 0x144724dc8 |
| 23 | authority_ignored_predicted_position | 0 | 0x1408d8220 | 0x144724e88 |
| 24 | trade_weapon | 0 | 0x1408d8220 | 0x144724f20 |
| 25 | device_touch | 0 | 0x1408d8220 | 0x144724f18 |
| 26 | deviceRelease | 0 | 0x1408d8220 | 0x144724ee8 |
| 27 | controlToggleResponse | 1 | 0x142f16de8 | 0x144724ef0 |
| 28 | biped_debug_teleport | 12 | 0x142f1699c | 0x144724e68 |
| 29 | prediction_determinism_msg | 8 | 0x142c469c8 | 0x144724a58 |
| 30 | biped_equipment_activation | 72 | 0x142f16ad8 | 0x144724d80 |
| 31 | equipment_teleport_request | 8 | 0x142ef8ec8 | 0x144724d20 |
| 32 | unit_teleported | 32 | 0x142ef9470 | 0x144724c98 |
| 33 | vehicle_auto_turret_choose_target | 0 | 0x1408d8220 | 0x144724e78 |
| 34 | PromptToBootGriefer | 12 | 0x142ef8ab0 | 0x144724ca8 |
| 35 | request_weapon_fire | 808 | 0x142f17500 | 0x144724de0 |
| 36 | **action_weapon_fire** | 808 | 0x14080c1f8 | 0x144724dd8 |
| 37 | weapon_overheat | 12 | 0x142ef94f4 | 0x144724c90 |
| 38 | weapon_reload | 8 | 0x1407f0ff8 | 0x144724ed0 |
| 39 | biped_throw_initiate | 20 | 0x140c6a58c | 0x144724ec8 |
| 40 | biped_melee_initiate | 20 | 0x140ff8d70 | 0x14473fa80 |
| 41 | vehicle_trick | 8 | 0x142f17c84 | 0x144724f08 |
| 42 | biped_dodge | 24 | 0x142f169d0 | 0x144724e70 |
| 43 | initiate_mobility_action | 164 | 0x142ef8f04 | 0x144724d18 |
| 44 | weapon_pickup | 8 | 0x142f18158 | 0x144724ed8 |
| 45 | weapon_put_away | 16 | 0x142f18284 | 0x144724eb0 |
| 46 | weapon_drop | 20 | 0x142f17d74 | 0x144724ea8 |
| 47 | weapon_throw | 20 | 0x142f18490 | 0x144724ec0 |
| 48 | weapon_tether_request | 20 | 0x142f183f0 | 0x144724eb8 |
| 49 | vehicle_flip | 0 | 0x1408d8220 | 0x144724f10 |
| 50 | request_ai_mount_exit | 2 | 0x142ef908c | 0x14473fa98 |
| 51 | biped_throw_release | 68 | 0x142f16cac | 0x144724f00 |
| 52 | biped_melee_damage | 76 | 0x142ef8eb8 | 0x14473fa88 |
| 53 | unit_enter_vehicle | 4 | 0x142f168c0 | 0x144724ef8 |
| 54 | **unit_switch_seat** | 0 | 0x1408d8220 | 0x144724c80 |
| 55 | game_engine_request_boot_player | 8 | 0x142f16dfc | 0x144724e58 |
| 56 | request_projectile_attach | 120 | 0x142f174c4 | 0x144724ee0 |
| 57 | biped_pickup_item_request | 0 | 0x1408d8220 | 0x144724e50 |
| 58 | projectile_supercombine_request | 12 | 0x142f17480 | 0x144724e98 |
| 59 | object_refresh | 0 | 0x1408d8220 | 0x144724e90 |
| 60 | RequestChangeFrameConfiguration | 52 | 0x142ef8b18 | 0x144724d58 |
| 61 | player_forge_action | 144 | 0x142ef8ff8 | 0x144724cc0 |
| 62 | player_loadout_request | 8 | 0x142f16fd4 | 0x144724e60 |
| 63 | biped_laser_designation | 1 | 0x142f16c90 | 0x144724ea0 |
| 64 | player_set_respawn_target_transform | 32 | 0x142f17348 | 0x144724e38 |
| 65 | player_set_orbiting_camera_target | 12 | 0x142f17178 | 0x144724e30 |
| 66 | PlayerEmote | 8 | 0x142f164b8 | 0x144724e48 |
| 67 | player_force_base_respawn | 2 | 0x142f16fb8 | 0x144724e40 |
| 68 | supply_request | 8 | 0x142eec01c | 0x144724c58 |
| 69 | CampaignMapStateUpdate | 20 | 0x142ef7fdc | 0x144724d40 |
| 70 | AIPhase | 52 | 0x142f15cf8 | 0x144724dc0 |
| 71 | AIRequestIdleTransitionTime | 4 | 0x142f15e70 | 0x144724db8 |
| 72 | AILand | 28 | 0x142f15c9c | 0x144724d90 |
| 73 | AIJuke | 12 | 0x142f15ae4 | 0x144724d88 |
| 74 | AISetMotorProgram | 1 | 0x142f15eac | 0x144724da0 |
| 75 | AIDialog | 12 | 0x140f2e634 | 0x144724d38 |
| 76 | Dialogue2D | 16 | 0x140f2e87c | 0x144724d30 |
| 77 | DebugSendCameraPosition | 20 | 0x142f160c4 | 0x144724d98 |
| 78 | ai_jump | 28 | 0x142ef8df0 | 0x144724d50 |
| 79 | networked_ai_action | 76 | 0x142ef8f5c | 0x144724ce8 |
| 80 | **networked_ai_effect** | 44 | 0x142ef8f74 | 0x144724cf8 |
| 81 | PlayerGameEvent | 312 | 0x142f164f4 | 0x144724f50 |
| 82 | **PlayerGameEventSmall** | 168 | 0x14080add8 | 0x144724f48 |
| 83 | **TeamGameEvent** | 312 | 0x142f1686c | 0x144724f60 |
| 84 | TeamGameEventSmall | 168 | 0x142f16818 | 0x144724f58 |
| 85 | PlayerKilledEvent | 40 | 0x14104bd08 | 0x144724f30 |
| 86 | EngineClientEvent | 4 | 0x142f1615c | 0x144724f28 |
| 87 | SaveGame | 2 | 0x142ef8b98 | 0x144724cd0 |
| 88 | RevertMap | 1 | 0x142ef8b5c | 0x144724cd8 |
| 89 | CancelCinematic | 8 | 0x142ef80fc | 0x144724d60 |
| 90 | ClientOnlyShowComplete | 8 | 0x142ef8138 | 0x144724d70 |
| 91 | ClientResourcesLoadComplete | 8 | 0x142ef8334 | 0x144724d68 |
| 92 | BetrayResponse | 0 | 0x1408d8220 | 0x144724d48 |
| 93 | activate_spartan_ability | 64 | 0x142ef8d94 | 0x144724c78 |
| 94 | CrewSetTargetObject | 8 | 0x142f15fc0 | 0x144724df0 |
| 95 | CrewOrderPositionAdd | 16 | 0x142f15ec0 | 0x144724de8 |
| 96 | NetworkedCrewEventType | 16 | 0x142f162c8 | 0x144724e00 |
| 97 | SaveToUGCService | 828 | 0x142f16544 | 0x144724df8 |
| 98 | Equipment | 8 | 0x142eebd68 | 0x14473fa58 |
| 99 | SelectedSpawnZoneChangedEvent | 24 | 0x142f167d4 | 0x144724f40 |
| 100 | PowerUpApplied | 12 | 0x142ef8a64 | 0x144724cb0 |
| 101 | LoadForgeObjectGroup | 712 | 0x142f16174 | 0x144724dd0 |
| 102 | NetworkedActionRequest | 48 | 0x142ef8a58 | 0x144724cf0 |
| 103 | EquipmentSpawnedObject | 0 | 0x1408d8220 | 0x144724c28 |
| 104 | EquipmentKnockbackPlayer | 12 | 0x14116c344 | 0x144724c38 |
| 105 | EquipmentObjectKnockedBack | 4 | 0x141118a00 | 0x144724c30 |
| 106 | ObjectCollisionDamage | 28 | 0x14112134c | 0x144724f88 |
| 107 | player_forge_user_string_action | 608 | 0x142ef9040 | 0x144724cc8 |
| 108 | **NavpointRequest** | 8 | 0x142ef8a40 | 0x144724d10 |
| 109 | PersonalAILifceycleEffect | 16 | 0x142c61310 | 0x144724a88 |
| 110 | ObjectDeterministicDamageAcceleration | 36 | 0x142f163c8 | 0x144724f90 |
| 111 | QueueNextShow | 4 | 0x142ef8afc | 0x144724cb8 |
| 112 | SetDifficultyAndSkulls | 8 | 0x142ef8d50 | 0x144724c70 |
| 113 | FOBClientInput | 8 | 0x142ef86d4 | 0x144724d28 |
| 114 | MusicMarker | 4 | 0x142ef87f0 | 0x144724d08 |
| 115 | synchronized_teleport | 44 | 0x142ef9284 | 0x144724ca0 |
| 116 | teleport_effects | 56 | 0x142ef93e0 | 0x144724c88 |
| 117 | EquipmentTranslocatorTeleportEffects | 28 | 0x140f04fb8 | 0x144724c48 |
| 118 | repair_complete | 4 | 0x142ef9074 | 0x14473fa90 |
| 119 | EquipmentKnockbackRequest | 276 | 0x142eebcec | 0x144724c40 |
| 120 | PlayerCalloutRequest | 128 | 0x142f163e0 | 0x144724f38 |
| 121 | PlayerForgeableCustomAction | 8 | 0x141fd8740 | 0x144724f70 |
| 122 | PlayerTriggerRadialMenu | 12 | 0x141fd87d0 | 0x144724f68 |

Trois noms nouveaux et notables pour le chantier, absents des catalogues précédents : **20
`incident`**, **85 `PlayerKilledEvent`**, **53 `unit_enter_vehicle`** (qui partage le lecteur du
type 8, `0x142f168c0`, donc la même charge utile `R(6)`).

---

## Niveaux de certitude

| Affirmation | Statut |
|---|---|
| Le type est un `R(7)` MSB en premier, lu par `FUN_14080a9d4` | **PROUVÉ** (désassemblage) |
| Aucun bit entre le `R(7)` et la porte de la référence 0 | **PROUVÉ** (désassemblage + preuve par absence d'argument sur les deux appels intercalés) |
| Un bit de continuation précède le type, lu par `FUN_14076a1c4` | **PROUVÉ** (désassemblage des deux côtés) |
| Trois références exactement, boucle bornée en dur | **PROUVÉ** (`CMP R12D,0x3`) |
| Domaine passé en `R8D` au var-int, rendu par `vtable+0x58` | **PROUVÉ** |
| Premier octet d'un événement aligné = `0x80 \| type`, type < 123 | **PROUVÉ** (les deux côtés) |
| La table runtime est `[[ctx+0x18]+0x210 + type*8]`, 123 entrées | **PROUVÉ** (dispatcher + `FUN_140e453b4` + `*out_n2 = 0x7b`) |
| La correspondance type → descripteur est une permutation explicite | **PROUVÉ** (123 offsets lus, bijection sur 0..122, sans trou) |
| `0x144724A90` n'est pas la table des descripteurs | **PROUVÉ** (xref unique vers une énumération Forge + flottants aux index 46..49) |
| Annexe A (123 noms) | **PROUVÉ** par construction mécanique : slot → vtable → `+0x08` → `LEA;RET` → chaîne, sans exception ni cas non résolu |
| `NavpointRequest` = `R(32)` + `R(32)`, non conteneur | **PROUVÉ** |
| `FUN_14080dec4` est un `R(32)` et le nom de champ est décoratif | **PROUVÉ** (corps identique au `R(32)` en ligne) |
| Grammaires de charge utile des types 8, 21, 100, 108, et des 13 types à 0 bit | **PROUVÉ** |
| Type 98 `Equipment` = 9 bits fixes (`R(8)` + `R(1)`) | **PROUVÉ** pour les largeurs et le total ; **SUPPOSÉ** pour l'ordre |
| `unite+0x462` = override réseau du niveau de lunette, `unite+0x461` = repli | **PROUVÉ** (`FUN_14076a484`, lecture conditionnelle sur `!= -1`) |
| `FUN_14080C1F8` est à l'indice 36 **et à lui seul** parmi les 123 lecteurs | **PROUVÉ** (balayage exhaustif de la table reconstruite) |
| Cardinal 123 en `+0x208`, table remise à zéro sur `+0x210` / `0x400` octets | **PROUVÉ** (`FUN_14054d014`) — seconde route indépendante sur la numérotation |
| Type 82 = record à sac de propriétés nommées (`R(3)` compteur, `R(32)` noms, union étiquetée `R(3)`) | **PROUVÉ** |
| Type 32 `unit_teleported` = deux vecteurs quantifiés + deux `R(32)` gardés (32 octets) | **PROUVÉ** |
| Les paquets `0xD2` sont le type 82 et non `action_weapon_fire` | **PROUVÉ** côté binaire ; reste à confirmer côté film par le test E7.5.1 |
| Aucun événement `unit_zoom` dans le corpus (octet `0x95` absent) | **PROUVÉ** par recoupement : arithmétique côté binaire + recensement du pilote |
| Grammaires marquées « variable / partiel » en E3 | **PARTIEL** : incréments de bits relevés, ordre exact non vérifié champ par champ |
| `0xD2` = `PlayerGameEventSmall` et `0xD3` = `TeamGameEvent` | **SUPPOSÉ** — cohérent avec la grammaire prouvée et avec la mesure (9 valeurs vs bruit), mais dépend du cadrage des paquets côté film, non vérifiable d'ici |
| La population à 85 % appartiendrait à la 2e famille (50 descripteurs) | **SUPPOSÉ** — piste, non instruite |

## Incertitudes et pièges signalés

1. **Le cadrage des paquets du dépôt n'est pas vérifiable depuis le binaire.** Les événements sont
   concaténés au bit près ; seul un événement démarrant sur une frontière d'octet a un premier octet
   égal à `0x80 | type`. Les deux tests de E6 tranchent.
2. **Grammaire dépendante du build.** `FUN_141102ed0(id)` garde des champs entiers
   (types 35, 36, 97, 114 au moins). Un décodeur hors ligne doit soit connaître ces drapeaux, soit
   traiter ces types comme de longueur incertaine. C'est un piège nouveau, non signalé par B1/B2.
3. **Des références var-int apparaissent À L'INTÉRIEUR de charges utiles** (types 35 et 36 appellent
   `0x1406d3140`). La longueur de ces records dépend donc aussi des domaines, pas seulement de
   champs de largeur fixe.
4. **Les largeurs `R(n)` passées sur la pile** (`0x1406d84b4`) n'ont pas été résolues type par type ;
   c'est la principale limite des grammaires « partielles » de E3.
5. **La seconde famille de 50 descripteurs** (`ctx+0x08+i*8`) n'a pas été explorée. C'est la piste la
   plus prometteuse pour la population majoritaire du flux.
6. **Réserve runtime de B1.6 maintenue** : les domaines 0, 1, 7, 8 passent de 13 bits à davantage si
   le monde dépasse 8 191 objets (`FUN_1408f1618`). Les domaines 2..6 sont fixes.
7. Le nom `PersonalAILifceycleEffect` (type 109) contient une faute de frappe **du binaire** ; il est
   reproduit tel quel.

## Suite proposée

1. Rejouer le décodage du film avec `T = premier_octet & 0x7F` et l'annexe A, puis publier la
   distribution des types : c'est le test décisif, et il coûte une passe.
2. Instruire la **seconde famille de 50 descripteurs** (`FUN_140e45fc4`, `ctx+0x08+i*8`) : si le flux
   majoritaire en vient, la largeur du type y est probablement `R(6)`.
3. Fermer les grammaires « partielles » de E3 en résolvant les largeurs passées sur la pile à
   `0x1406d84b4`, en priorité pour le type 36 `action_weapon_fire` (record de dégât).

## État de l'outillage à la clôture

Le greffon HTTP `127.0.0.1:8089` a répondu à la totalité des requêtes de ce lot (environ 560
lectures mémoire et 60 décompilations), sans aucun incident. Aucune recherche d'instructions n'a été
lancée. **Piège d'outillage relevé** : `/disassemble_function` ne s'arrête pas à la fin de la
fonction — la première requête a produit 213 Mo. Toujours borner la sortie côté client.
