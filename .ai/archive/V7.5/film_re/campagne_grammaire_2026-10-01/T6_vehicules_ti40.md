# T6 — Véhicules `ti=40` : composants non portés et porte `+0x818` (2026-10-01)

> Campagne grammaire, phase 1, étape 2, piste T6. Plan : `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`.
> Source : `HaloInfinite.exe` dans Ghidra, serveur HTTP `127.0.0.1:8089`, LECTURE SEULE (aucun
> `rename_*`, `set_*`, `create_*`...). Image base `0x140000000`. Le seul code non défini en fonction
> dans Ghidra (`0x14116d3cc`) a été lu en octets bruts (`read_memory`) puis désassemblé hors Ghidra
> (`objdump -b binary`, scratchpad). Code Go lu à la tête du worktree (`69564ef7d`), aucune
> commande `go` lancée, aucun film décodé.

---

## 0. Le résultat en six lignes

1. **La porte `+0x818` est une LECTURE, et le film porte ce qu'il faut pour la faire.** L'octet est
   écrit par `FUN_14058c2ec` (slot `+0x88` de la vtable d'archétype `ti=40`, « construire l'état
   depuis les données de création ») : `+0x818 = (type de physique du tag vehi == 6)`. Le tag vehi
   vient de `MPPWord32` (bloc MPP de l'état par défaut, `GetLocalHandleFromGlobalId`). Le lecteur du
   JEU fait exactement ce geste avant de lire les composants : `FUN_142e309b4` (record NEW) et
   `FUN_142e2bfd0` (image-clé) appellent `vtable+0x60` (état par défaut), puis `vtable+0x88`.
2. **Type 6 = `vtol`** (probable, trois indices concordants, §2.3) : la porte est posée pour les
   véhicules à physique VTOL et seulement pour eux.
3. **Les images-clés n'ont PAS de masque** (`FUN_142e2c690` désérialise toutes les entrées
   nommées) : `i33` et `i34` y sont appelés pour TOUT véhicule, et lisent 0 bit hors VTOL. Le repli
   actuel (« porte supposée posée dès que le composant est annoncé ») y serait faux sur chaque
   véhicule non VTOL. C'est aujourd'hui caché : les 1 131 records `ti=40` d'image-clé de `4f77afc1`
   s'arrêtent tous à `i30`, non porté, AVANT `i33`.
4. **Dans un delta, l'annonce de `i33`/`i34` PROUVE la porte** : l'écrivain du masque de différence
   (`FUN_142f09c74`, slot `+0x78`) ne pose les bits 33 et 34 que sous `if (*(state+0x818))`. Les
   33 193 lectures « supposées » de la vue B sont donc couvertes par une loi d'écrivain, pas par
   une supposition (probable : une réserve sur les masques complets, §3.3).
5. **Les 16 composants sont relevés** (§4) : tous à largeur statique, sauf `i33`/`i34` (porte
   `+0x818`). `i47 vehicle-low-frequency`, « suspect n°1 » de la note 3.6, fait 1 ou 6 bits :
   SUSPICION RÉFUTÉE. `i41`/`i42` (16 bits chacun) sont à porter MALGRÉ le « ne pas porter » de la
   table : l'image-clé n'a pas de masque, ils y sont toujours lus.
6. **Constat négatif sur la cause n°1** : `ti=40` ne nourrit pas « vue C : terminateur hors
   cadre ». Ses composants non portés s'arrêtent en DÉSYNCHRO NOMMÉE (1 599 paquets sur 18 films à
   la carte du 26/09, borne 18 838 records utiles), pas en sortie par rejet. Mais porter `i30`-`i33`
   SANS faire de la porte une lecture changerait ces désynchros nommées en désalignements muets —
   donc en « hors cadre » (§6).

---

## 1. Table composant -> désérialiseur (statique)

Chaîne : nom ASCII -> accesseur de nom (`LEA RAX,[rip+d] ; RET`) -> bloc de descripteur de
10 qwords (`[0]=0x141191ab0`, `[1]=0x14076ced0`, `[2]`, `[3]` = accesseur, `[4]=0x1404ab600`,
`[5]` = ÉCRIVAIN, `[6]=0x1411c8f80`, `[7]` = thunk `0x14076ce9c`, `[8]` = DÉSÉRIALISEUR). Relu le
2026-10-01 en mémoire (`read_memory 0x143d0b1b0`, 1 536 octets). Le bloc de `i32` porte un qword
de plus (`0x142f02240` en `[2]`), décalé d'une case.

| Composant | Bloc | Accesseur -> nom | Écrivain | Désérialiseur |
|---|---|---|---|---|
| `i30 vehicle-auto-turret-triggers` | `143d0b668` | `141177350` -> `143c99558` | `142f08c54` | `142f04994` |
| `i31 vehicle-auto-turret-aiming-vector` | `143d0b6b8` | `141177340` -> `143c994e8` | `142f08b6c` | `14115f33c` |
| `i32 vehicle-transformed-or-desired-open-state-changed` | `143d0b610` | `141177330` -> `143c99518` | `142f08d88` | `142f04b70` |
| `i33 vehicle-type-state` | `143d0b3e0` | `141177250` (`MOV RAX,[143b8c860]`) | `142f04e6c` | `142f02474` |
| `i34 vehicle-type-physics` | `143d0b2f0` | `141177260` (`MOV RAX,[143b8c868]`) | `142f04e90` | `142f02498` |
| `i35 vehicle-auto-turret-target` | `143d0b758` | `141177320` -> `143c995a0` | `142f08c24` | `142f0496c` |
| `i36 vehicle-sentry-state` | `143d0b708` | `141177310` -> `143c99580` | `142f08d58` | `142f04b34` |
| `i37 vehicle-emp-timer` (porté) | `143d0b480` | `141177300` -> `143c995e8` | `142f08c94` | `142f049dc` |
| `i38 vehicle-weapon-set` | `143d0b4d0` | `1411772f0` -> `143c995c8` | `142f08dbc` | `14116d3cc` (hors fonction Ghidra) |
| `i39 vehicle-auto-turret` | `143d0b430` | `1411772e0` -> `143c99430` | `142f08b80` | `142f04884` |
| `i40 vehicle-equipment-turret-parent` | `143d0b5c0` | `1411772d0` -> `143c99478` | `142f08ca8` | `142f04a00` |
| `i41 vehicle-seats-override-pitch` | `143d0b520` | `1411772c0` -> `143c99450` | `142f08cf0` | `142f04a4c` |
| `i42 vehicle-seats-override-yaw` | `143d0b570` | `1411772b0` -> `143c99408` | `142f08d24` | `142f04ac0` |
| `i45 air-drop-flight` | `143d0b1b0` | `141177290` -> `143c994a8` | `142f04f08` | `142f02508` |
| `i46 warp` | `143d0b2a0` | `141177280` -> `143c99638` | `142f08dcc` | `142f04bcc` |
| `i47 vehicle-low-frequency` | `143d0b200` | `1411772a0` -> `143c994c8` | `142f08cd4` (hors fonction) | `142f04a20` |

Les désérialiseurs de `NOTE_3_6_TI40_2026-09-16.md` (colonne « Écrivain ») sont confirmés un par
un ; la colonne y nommait le désérialiseur, pas l'écrivain.

---

## 2. La porte `+0x818` — qui l'écrit, et pourquoi c'est une lecture

### 2.1 Les deux lecteurs de l'octet (rappel)

```
FUN_142f02498 (i34, désér.)   RDI = [R8+0x10] ; MOVZX EAX,byte [RDI+0x818] @142f024ae ; JZ fin
                              R(1) c ; mode = c ? 2 : 0 ; FUN_140c5f938(+0x7f4,+0x800,mode) ; FUN_14076e1c8(+0x80c,mode)
FUN_142f04e90 (i34, écrivain) RDI = [R8+0x30] ; MOVZX EAX,byte [RDI+0x818] @142f04ea6 ; JZ fin
                              W(1) = (byte [RDI+0x7f2] != 0) ; FUN_141f86118 ; FUN_141f860d0
FUN_142f02474 (i33, désér.)   RCX = [R8+0x10] + 0x7f0 ; MOVZX EAX,byte [RCX+0x28] @142f02483 (= +0x818) ; JZ fin
                              CALL FUN_14320c4c8(RCX, lecteur)
FUN_142f04e6c (i33, écrivain) if (*([R8+0x30] + 0x818)) FUN_14320c5c0()
```

### 2.2 L'écrivain de l'octet : `FUN_14058c2ec`

Recherche exhaustive des écritures d'octet en `[x+0x818]` (`search_instructions MOV` +
`operand_pattern +0x818]`, 103 accès, 11 écritures d'octet) : une seule touche la structure de
l'état véhicule — les dix autres sont des constructeurs d'autres objets (`FUN_1411c9ad0`,
grappe `0x14157xxxx`-`0x14158xxxx`). Décompilation de `FUN_14058c2ec` (extrait) :

```c
memset(param_5, 0, 0x8d8);                         // l'état véhicule, 0x8d8 octets
FUN_1404d30a8(param_3, param_5);                   // préfixe objet (0x538)
FUN_14058b824(param_1, param_2, param_3, param_4, param_5);
...
lVar3 = FUN_14058c200(param_3 + 0x14);             // tag 'vehi' (0x76656869) des données de création
if (lVar3 != 0) {
  bVar5 = FUN_1408b44fc(lVar3) == 6;               // type de physique du tag
  ...
  *(bool *)(param_5 + 0x818) = bVar5;              // @14058c3a8 : MOV byte [RBX+0x818],AL
  if (bVar5) { +0x7f0 = 2 ; +0x7f4 = DAT_143d8cc08.. ; +0x800 = DAT_143d8cbf8.. }
}
```

- `FUN_14058c200` : `FUN_1404cc908(groupe, 'vehi')` puis `FUN_140478680(tag, 'vehi')` — la
  définition `vehi` désignée par `données_de_création + 0x14`.
- `FUN_1408b44fc(vehi)` (désassemblé `0x1408b44fc`-`0x1408b460b`) : douze couples (bloc, index) —
  blocs `vehi+0xde0 + 0x14*k`, index `k` de 0 à 11 (`MOV dword [RBP-0x1],0x6` pour le bloc
  `+0xe58`) — rend l'index du PREMIER bloc dont le compte (`[bloc+0x10]`) est non nul, 13 sinon.
- `FUN_14058c2ec` est le slot `+0x88` de la vtable d'archétype `ti=40` (base `0x143736fd8` :
  `+0x60 = 0x1410a5a74` l'état par défaut, connu ; `+0x78 = 0x142f09c74` ; `+0x88 = 0x14058c2ec`).
  Seule référence de code : la donnée `0x143737060` (les deux autres sont des entrées `.pdata`).

### 2.3 Le type 6 est `vtol` (probable)

- Table de noms à `0x1448021a0`, douze pointeurs contigus, ordre : `vehicle_type_human_tank`,
  `human_jeep`, `human_plane`, `alien_scout`, `alien_fighter`, `turret`, **`vtol`**, `chopper`,
  `guardian`, `jackal_glider`, `space_fighter`, `revenant` (index 6 = `0x1448021d0` -> `0x143e2aa70`).
- `FUN_1431ae734` parcourt les objets ENFANTS d'un objet (`+0x28` premier enfant, `+0x1c` frère)
  et compte ceux dont `FUN_1408b44fc == 5` : « la N-ième tourelle ». 5 = `turret` : concorde.
- `FUN_142b65784`, `FUN_142b65284`, `FUN_142b65504` sortent quand `(1 << type) & 0x654` : bits
  2, 4, 6, 9, 10 = `human_plane`, `alien_fighter`, `vtol`, `jackal_glider`, `space_fighter` — les
  types VOLANTS. Concorde.
- Concordance avec le terrain (mesures déjà au dépôt, non rejouées ici) : `i34` à 0 % sur
  `8a049c50` (Behemoth, aucun appareil), 31,1 % sur `0d76e8f1` (Warthog ET Wasp, `V4` du
  2026-09-02), 18,1 % sur `fccc61cd` ; `1cd3848a` (fenêtre où le port M4b a fermé 765/785
  paquets) a des Falcon. Le Falcon comme `vtol` reste à MESURER (§7, M1).

### 2.4 Le lecteur du jeu pose l'octet AVANT de lire les composants

```
FUN_142e309b4 (record NEW)
  vtable+0x60(..., param_3 + 0x28, lecteur)            @142e30a98 : état par défaut -> données de création
  vtable+0x88(..., param_3 + 0x28, taille, param_3 + 0x21bc)        : construit l'état (pose +0x818)
  FUN_14076cb60(plVar2 + 1, ...)                                    : masque puis composants
FUN_142e2bfd0 (image-clé, par entité)
  R(32) n1 ; si n1 > 0 : vtable+0x60(...)               @142e2c47b
  R(32) n2 ; si n2 > 0 : vtable+0x88(...) puis FUN_1428e2b68 -> FUN_142e2c690 (boucle sans masque)
```

Et l'état par défaut fournit la clé : `FUN_14080cfe8` (bloc MPP, appelé par `FUN_1410a5a74`
`@0x1410a5ad4`) lit `MPPWord32` dans `données+0xc` (`FUN_14080d6f0`), puis
`données+0x14 = FUN_14080d61c(&word32, variant)`, qui est `GetLocalHandleFromGlobalId` (chaîne
d'erreur citée dans la fonction) : le handle de tag dépend du SEUL `MPPWord32`.

**Conclusion** : `porte(+0x818) = typePhysique(vehi(MPPWord32)) == 6`. Le Go publie déjà
`MPPWord32` (`grammar/default_state.go:340`, `br.obs.publishMPP(MPPWord32, ...)`), lu dans le MÊME
record avant la boucle de composants pour l'image-clé et le NEW. Il ne manque qu'une table
`MPPWord32 -> type de physique` (ou `-> porte`).

---

## 3. Où la porte compte : image-clé, NEW, delta

### 3.1 Image-clé : pas de masque (établi)

`FUN_142e2c690` boucle sur 64 entrées : pour chaque entrée NOMMÉE, retrouve le composant par nom
et appelle `vtable+0x28` (le thunk -> désérialiseur), puis, si `FUN_14076cea8()`, R(1)[R(32)].
Aucun masque. Le Go le dit déjà (`keyframe_fullstate_loop.go`, en-tête : « 64 entrées nommées,
AUCUN masque de présence »). Donc, en image-clé, `i33` et `i34` sont désérialisés pour TOUT
véhicule ; hors VTOL ils consomment 0 bit.

Écart Go : `consumeVehicleTypePhysics` (`composants_vue_b_m4b.go`) lit le corps sans condition.
Aujourd'hui sans effet : `keyframe_closure.golden` (lot 5.1.7-b) note `bVar14=0 661/661 à i30`,
`bVar14=1 470/470 à i30` sur `4f77afc1` — la boucle s'arrête à `i30` (non porté) avant `i33`.

### 3.2 Delta : l'annonce prouve la porte (établi côté écrivain)

`FUN_142f09c74` (slot `+0x78`, calcul du masque par comparaison de deux états) :

```c
lVar2 = FUN_1406caa84(param_5, &local_28, &DAT_14473fb10);   // demandé ∧ {33,34}
...
if (uVar3 != 0) {
  local_28.. = 0;
  if (*(char *)(param_2 + 0x818) != '\0')
    FUN_14320c8fc(param_2 + 0x7f0, param_3 + 0x7f0, param_5, &local_28);
  FUN_1406cb04c(param_6, &local_28);                          // OU dans le masque de sortie
}
```

`DAT_14473fb10` = `00000000 06000000 ...` (mot 1 = 0x6 : bits 33 et 34). `FUN_14320c8fc` pose
`mot1 | 2` (i33 : octets `+0x7f0`/`+0x7f1` différents) et `mot1 | 4` (i34 : écarts de `+0x7f4..`,
`+0x800..`, `+0x80c..` au-delà des seuils `DAT_143cd8490..`). Rien d'autre dans ce chemin ne pose
ces bits. Parmi les 64 appelants de `FUN_1404bc3c8` (marquage explicite d'un composant sale,
celui qui marque `0x29` pour `i41`), aucun ne passe `0x21` ni `0x22` en immédiat (fenêtre de
16 octets avant l'appel).

### 3.3 Le masque COMPLET existe aussi (réserve)

`FUN_142e32138` construit un masque à TOUS les bits (`FUN_1406c81b0(masque, 0, n-1, 1)`) après
`vtable+0x88`, filtré par `vtable+0x90` (`FUN_142ee98ec` -> `FUN_142ee4fc8`, masques de contrôle
`+0x160`/`+0x168`). Appelants : `FUN_142f254d0` et `FUN_1408f1730` (ajout d'un pair : un masque
par entité). Un record écrit avec ce masque annonce `i30`..`i47` — le Go s'y arrête aujourd'hui à
`i30`, donc AUCUNE des 33 193 lectures supposées ne vient d'un tel masque. La provenance du
masque d'un record NEW de flux delta (`FUN_142e31a0c`, masque reçu de `FUN_142e2eec0`) n'a pas
été remontée jusqu'au bout : réserve consignée, à trancher par la mesure M3.

---

## 4. Grammaires, chez le désérialiseur (lecteur `+0x2c += N`)

| Composant | Grammaire | Bits | Preuve |
|---|---|---|---|
| `i30` triggers | R(1) `+0x81c` ; R(1) `+0x81d` ; R(1) `+0x81e` | 3 | 3 x `FUN_1406cf008` |
| `i31` aiming-vector | `FUN_14076dc04` R(19) -> `+0x828` ; `FUN_1404fedf8` 0 bit (test de norme) | 19 | `MOV R9D,0x13 @14115f346` |
| `i32` transformed/open | R(1) `+0x820` ; `FUN_1406d84b4` W=8 -> `+0x824` | 9 | `MOV dword [RSP+0x20],0x8 @142f04ba1` |
| `i33` type-state | **si `+0x818`** : `FUN_142af27f8` R(2) v -> `+0x7f0` ; si v ∈ {1,3} : R(6) -> `+0x7f1` | 0 / 2 / 8 | `FUN_14320c4c8` : `+0x2c += 2` puis `*p == 1 \|\| *p == 3` -> `+0x2c += 6` |
| `i34` type-physics | **si `+0x818`** : R(1) c ; `FUN_140c5f938` + `FUN_14076e1c8` au mode `c ? 2 : 0` | 0 / variable | porté (`consumeVehicleTypePhysics`) |
| `i35` auto-turret-target | `FUN_1408f0ac4(+0x83c, lecteur, 1)` : R(1) ; si 1 : [R(1) sonde] R(13 ou 9) R(2) | 1 / 13 / 17 | `MOV R8D,0x1 @142f04974` ; = `consume1408f0ac4(br, 1)` |
| `i36` sentry-state | `FUN_1424d9a30` R(3) -> `+0x844` ; R(1) -> `+0x845` | 4 | `+0x2c += 3` |
| `i37` emp-timer | R(8) | 8 | porté |
| `i38` weapon-set | `JMP FUN_1406d01fc(+0x84c)` : `FUN_1406d0f20` R(3) ; 2 x `FUN_1406d00ec` [R(1) ; si 0 : R(2)] | 5 à 9 | octets bruts `14116d3cc` : `MOV RCX,[R8+0x10] ; ADD RCX,0x84c ; JMP 0x1406d01fc` ; = `consumeBipedDesiredWeaponSet` |
| `i39` auto-turret | R(2) -> `+0x81f` | 2 | `+0x2c += 2` |
| `i40` equipment-turret-parent | `FUN_1408f0ac4(+0x834, lecteur, 0)` : R(1) ; si 1 : R(13) R(2) | 1 / 16 | `XOR R8D,R8D @142f04a08` ; = `consume1408f0ac4(br, 0)` |
| `i41` / `i42` seats-override | 2 x R(8) chacun, inconditionnel | 16 / 16 | `ecs_table.tsv` (lot 5.5), confirmé |
| `i45` air-drop-flight | `FUN_142af27f8` R(2) -> `+0x8c4` ; `FUN_1406d84b4` W=14 ; `FUN_1406d84b4` W=8 | 24 | `MOV dword [RSP+0x20],0xe @142f02540` ; `,0x8 @142f02562` |
| `i46` warp | a = R(1) ; b = R(1) ; si a : `FUN_1406d84b4` W=8 ; si b : W=8 | 2 / 10 / 18 | `@142f04c1a` et `@142f04c48` : `,0x8` |
| `i47` vehicle-low-frequency | `FUN_1407f1ff4("customization-source-participant")` -> `FUN_1407f2058` : R(1) ; si 0 : R(5) | 1 / 6 | `ADD [RBX+0x2c],0x5 @1407f207f` ; = `consumeOpt5` |

Écrivains relus en symétrie pour `i32` (`FUN_142f08d88` : W(1) + `FUN_142ed1a78`), `i35`
(`FUN_142f08c24` : `FUN_1409a5ff0(.., 1, ..)`), `i45` (`FUN_142f04f08`), `i46`
(`FUN_142f08dcc` : deux W(1) puis deux corps sous les mêmes octets `+0x8d4`/`+0x8d5`).

`FUN_1406d84b4` consomme exactement sa largeur de pile (`in_stack_00000028` = `[RSP+0x20]` de
l'appelant) ; ses deux drapeaux (`+0x28`, `+0x30`) ne changent pas le compte de bits.

Hors `i33`/`i34`, AUCUN de ces désérialiseurs ne lit l'état de l'objet pour décider d'une
largeur : les portes de `i35`, `i40`, `i46`, `i47` sont des bits du flux. Les catégories 0 et 1
de `FUN_1406d3140` sont celles de `varwidth.go` (13 bits, sonde de la catégorie 1 -> 9 bits).

---

## 5. Écarts Go nommés

| # | Écart | Fichier |
|---|---|---|
| E1 | `i34` lit son corps sans condition (repli `repli_physique_de_type_de_vehicule_supposee`) ; le jeu le lit si `vtol(MPPWord32)` | `grammar/composants_vue_b_m4b.go` (`consumeVehicleTypePhysics`), `facts/fallback/registre_filmdec_marche.go` |
| E2 | `i30`-`i33`, `i35`, `i36`, `i38`-`i40`, `i45`-`i47` non portés : arrêt `desync` (vue B) et arrêt de la boucle d'image-clé à `i30` | `grammar/composants_vue_b_m4b.go` (`default:`), `ecs_table.tsv` |
| E3 | `i41`/`i42` marqués « NE PAS PORTER » : vrai en delta (jamais annoncés), FAUX en image-clé (boucle sans masque) | `ecs_table.tsv` lignes `ti=40 i41/i42` |
| E4 | La note 3.6 range `i47` comme « suspect n°1, potentiellement grand » : 1 ou 6 bits | `.ai/V7.5/film_re/NOTE_3_6_TI40_2026-09-16.md` |

---

## 6. Correctif proposé (phase 2, après J12)

1. **Porter les 13 grammaires statiques** de §4 dans le maillon `consumeComposantsVueBM4b` (ou un
   maillon `ti=40` à lui) en RÉUTILISANT les lecteurs existants : `consume1408f0ac4(br, 1)` (`i35`),
   `consume1408f0ac4(br, 0)` (`i40`), `consumeBipedDesiredWeaponSet` (`i38`), `consumeOpt5` (`i47`),
   `ReadBits(8)` x 2 (`i41`/`i42`). Ne PAS les livrer sans le point 2.
2. **Faire de la porte une lecture** : un état du lecteur `porteTypeVehicule` (trois valeurs :
   posée / levée / inconnue), décidé
   - dans un record d'image-clé ou NEW `ti=40` : APRÈS `consumeDefaultStateTI40`, par
     `typePhysique(MPPWord32) == vtol` ;
   - dans un delta : par le châssis connu du slot (même table), à défaut par la LOI de l'écrivain
     (bit 33/34 annoncé ⇒ posée, `FUN_142f09c74`), nommée comme loi et non comme repli ;
   - contradiction (annonce de `i33`/`i34` en delta sur un châssis connu non VTOL) = erreur typée
     comptée.
   `i33` lit `R(2)` puis `R(6)` si `v ∈ {1,3}` sous cette porte ; `i34` garde son corps actuel.
3. **La table `MPPWord32 -> type de physique`** : statique, à côté de `replay/vehicle_families.go`
   (même doctrine : le serveur ne lit aucun fichier de jeu), extraite UNE fois des tags `vehi`
   (12 blocs `physics types` en `+0xde0 + 0x14k`, compte en `+0x10` — disposition mémoire ; la
   disposition dans le `.module` est à vérifier au moment de l'extraction, sous tag `gamefiles`).
   Chaque entrée cite sa source ; un châssis inconnu en image-clé/NEW reste un repli NOMMÉ et
   COMPTÉ (`porte_type_vehicule_chassis_inconnu`), plus étroit que l'actuel.
   Variante sans table, pour VALIDER la table et non la remplacer : en image-clé, jouer les deux
   portes et garder celle qui ferme le record (oracle de cadrage, une valeur par `MPPWord32` doit
   en sortir, constante sur tout le parc).
4. Retirer le repli `repli_physique_de_type_de_vehicule_supposee` quand `porte lue` est câblé et
   que le compte de lectures supposées vaut 0 sur le corpus du gate (son critère de retrait actuel).

### 6.1 Vecteurs de test (construits d'après l'écrivain)

Composants seuls (`Lecteur` sur un tampon synthétique, position attendue après lecture) :

- `i30` : `101` -> 3 bits. `i36` : `110 1` -> 4. `i39` : `10` -> 2. `i31` : 19 bits quelconques -> 19.
- `i32` : `1` + `10000000` -> 9.
- `i35` : `0` -> 1 ; `1 1` + 9 bits + `01` -> 13 ; `1 0` + 13 bits + `01` -> 17.
- `i40` : `0` -> 1 ; `1` + 13 bits + `10` -> 16 (aucun bit de sonde : catégorie 0).
- `i38` : `011` `1` `0 10` -> 7 ; `011` `0 01` `0 11` -> 9.
- `i45` : `10` + 14 bits + 8 bits -> 24.
- `i46` : `10` + 8 bits -> 10 ; `00` -> 2 ; `11` + 16 bits -> 18.
- `i47` : `1` -> 1 ; `0 10101` -> 6.
- `i33`, porte posée : `01 000101` -> 8 ; `10` -> 2 ; `11 111111` -> 8 ; porte levée : 0 bit, quel
  que soit le tampon.
- `i34`, porte levée : 0 bit (aujourd'hui le Go en lit au moins 1 + avant/haut + vitesse).

Record d'image-clé synthétique `ti=40` (registre du film, 48 composants) : même tampon, deux
`MPPWord32` (un châssis `vtol`, un Warthog) ; attendu : la fin de `i32` et le début de `i35`
coïncident pour le Warthog ; pour le VTOL, ils sont séparés par 2 ou 8 bits (`i33`) plus le corps
de `i34`.

---

## 7. Mesures proposées (effet sur la fermeture et preuve des lois)

- **M1 — la loi de l'écrivain et l'identité du type 6** (aucune cuisson ; un décodage par film,
  les 20 témoins) : pour chaque record delta `ti=40` qui annonce `i33` ou `i34`, le `MPPWord32` du
  slot (dernier NEW/image-clé) et sa famille (`vehicleFamilyOf`). Attendu : 100 % de familles
  aériennes (Wasp ; Falcon, Pelican, Phantom à trancher), 0 Warthog, Mongoose, Ghost, Scorpion, ni
  pièce montée (`dd7f9102` LAAG, tourelles du Falcon/Wraith = type 5). Un seul châssis terrestre
  réfute « type 6 = vtol » ou la loi.
- **M2 — image-clé** : `TestKeyframeClosureRatchet`, ligne `ti=40`, avant/après le correctif sur
  les 7 bobines (0/777 au golden) et `4f77afc1` (0/1 140). Témoin négatif OBLIGATOIRE : la même
  marche avec la porte toujours posée (le comportement actuel). Prédiction : la porte lue ferme
  les records jusqu'au prochain bloquant (`i43 simulation-state`, `partiel`, piste T7) ; la porte
  posée désaligne tous les non-VTOL dès `i33`.
- **M3 — carte de fermeture v2** : ventilation « dernier composant lu » `ti=40` et « sortie de vue
  B ». Prédiction chiffrée sur la carte du 26/09 : les 1 599 paquets arrêtés par `i31` (807),
  `i38` (539), `i33` (83), `i45` (59), `i47` (48), `i46` (36), `i30` (27) ferment ou passent à un
  autre bloquant ; gain borné à 18 838 records utiles ; la cause « terminateur hors cadre » ne
  doit PAS baisser (prédiction négative) — et ne doit pas MONTER : une montée sur les films à
  véhicules signerait un record NEW à masque complet lu avec la mauvaise porte (réserve §3.3). En
  même passe : compter, par type de record (NEW / delta), les annonces de `i30` sur un châssis
  sans tourelle automatique (signature d'un masque complet).
- **M4 — compteur du repli** : `VehicleTypePhysicsAssumed` (33 193 sur 7 témoins) doit tomber à 0
  lectures SUPPOSÉES ; les mêmes lectures passent au compte « porte lue » (table) ou « loi de
  l'écrivain » (delta sans châssis connu).

---

## 8. Hors périmètre, consigné

- **T8 (bourrage)** : les lecteurs de bits relus ici (`FUN_1406d84b4`, `FUN_142af27f8`,
  `FUN_1424d9a30`, `FUN_1407f2058`) complètent avec des ZÉROS quand le pointeur atteint la fin
  (`if (fin < p + 1)` : octets restants puis décalage), et incrémentent `+0x2c` quand même ; le
  dépassement se constate APRÈS coup par comparaison `[lecteur+0x18] * 8 < [lecteur+0x2c]` (vu dans
  `FUN_14080cfe8`, fin de fonction). À croiser avec la note T8.
- **Produit** : `i40 vehicle-equipment-turret-parent` porte une référence d'entité (catégorie 0)
  et `i35 vehicle-auto-turret-target` une référence d'unité (catégorie 1). Le rattachement d'une
  pièce montée à son porteur se fait aujourd'hui par voisinage de slot (`slot+1`,
  `RAPPORT_tirs_vehicules.md` §2.5) : `i40` pourrait le LIRE (à mesurer : sur quels châssis il est
  non nul).
- **`i33` d'un VTOL** : quatre états sur 2 bits, un complément de 6 bits pour les états 1 et 3 ;
  l'état initial posé par `FUN_14058c2ec` vaut 2. Sens non établi (configuration de vol ?).
- La carte du 26/09 nomme certains composants `ti=40` à deux index (`i37, i38`, `i32, i33`,
  `i44, i45`...) : le registre `ti=40` n'est pas le même sur toutes les builds, contrairement à ce
  qu'affirme `CADRAGE_VEHICULES_2026-08-31.md` §1.5 (« ti=40 ne bouge pas », mesuré sur 11 films
  seulement). Sans effet sur le port (dispatch par NOM).
