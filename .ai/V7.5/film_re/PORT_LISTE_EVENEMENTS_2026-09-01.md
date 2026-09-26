# PORT — La liste d'événements du film + les deux événements véhicule (board / exit)

> 2026-09-01, worktree `LevelUp-wt-vehicules`, branche `wt/vehicules-tourelles`.
> Livrable : `internal/analysis/filmdec/event_list.go` (production) + `event_list_test.go`
> (instrument garde, sous garde d'environnement). Aucun commit. Aucune ligne du décodeur ECS
> existant modifiée : le portage est STRICTEMENT ADDITIF.

## 0. Résultat en cinq lignes

1. **Le cadrage de la liste d'événements est porté et PROUVÉ bit-exact.** Le garde-fou de
   recoupement est net : sur 0d76e8f1, `head type36 = fire_events = 1418` à l'unité (octet 0xD2).
2. **La SORTIE (`unit_exit_vehicle`, type 22) est entièrement décodée et validée** : occupant
   (bipède), siège, instant. 209 sorties mesurées, occupant en-bande 95,5 %, **siège = 0 sur
   93,8 %** (conforme au « siège dominant 0 » mesuré par le chantier trame).
3. **L'EMBARQUEMENT (`biped_board_vehicle`, type 8) est décodé partiellement** : le siège
   reproduit le « siège dominant 16 » mesuré, mais l'OCCUPANT de l'embarquement n'est PAS résolu
   proprement (sa réf 0 porte une sonde variable et ne tombe pas dans la bande bipède — voir § 4).
4. **Comptes corpus (cache de 949 films) : board = 348 sur 138 films, exit = 5 144 sur 230
   films, enter (type 53) = 0.** Référence chantier trame (1 367 films) : 374 / 5 600 / 0. Le
   ratio board:exit = 1:14,8 recoupe la référence 1:15 ; le déficit correspond aux films absents
   du cache local.
5. **Garde-fou ECS/kill respecté** : la trame de records et l'arme-du-kill sont inchangées (fichiers
   `traverse.go`/`frame_records.go`/`fire_events.go` non touchés ; `go test ./internal/analysis/filmdec/`
   vert ; recoupement fire_events bit-exact).

---

## 1. La grammaire portée

### 1.1 Le modèle de paquet (percée trame du 30/08)

Un paquet delta s'écrit :

```
[1 bit config] [liste d'événements : ( 1 [R(7) type] [3 réfs gardées] [charge] )* 0] [trame de records ECS]
```

En lecture MSB-first (`BitReader`) : **bit 0 = config, bit 1 = continuation, bits 2..8 = R(7)
type**. Le décodeur de records de production consomme `DefaultPacketPreambleBits = 2` puis lit
les records : ce sont exactement `[config][continuation=0]` du cas « liste vide » — d'où sa
justesse sur les paquets à liste vide (octets 0x80..0xBF) et son incapacité sur ceux qui portent
un événement (0xC0..0xFF). `event_list.go` lit la liste EN AMONT, sans toucher au décodeur ECS.

Arithmétique de tête confirmée : `octet0 = 0xC0 | (type >> 1)` (le bit de poids faible du type
est le bit de poids fort de l'octet 1).

### 1.2 Le dispatcher (Ghidra, `FUN_14080AADE` décompilé)

Le dispatcher confirme la grammaire au bit près :
- il lit un **type sur 7 bits**, valide s'il est **< 123** (0x7b) — les 123 entrées de la table
  par type ;
- il lit ensuite **EXACTEMENT 3 références gardées** dans une boucle (`puVar8` de 0 à 2) : pour
  chaque réf, un bit de garde (`FUN_1406cf008`) ; si posé, la réf est lue par un lecteur propre
  au domaine (vtable+0x58 puis `FUN_1406d3140`, l'id-reader partagé avec `readRecordID`) ;
- il lit enfin la **charge** de l'événement (vtable+0x68).

La table des handlers est un objet runtime (`[param+0x18]+0x210 + type*8`) ; sa contrepartie
statique 0x144724A90 pointe vers des descripteurs dont les vtables emploient des thunks relocatés
(l'image statique les rend en 0xcc), ce qui n'a pas permis de lire les déserialiseurs de charge
par pur pointeur statique. La **grammaire de charge du siège a donc été fixée par l'oracle
empirique** (§ 3), pas par décompilation du deser board/exit.

### 1.3 La référence gardée

`R(1) garde ; si 1 : [R(1) sonde si domaine 1] ; R(largeur) index ; R(2) génération`.
Largeurs par domaine (build de référence) : dom 0/1/7/8 = 13, dom 2/3/5 = 8, dom 4/6 = 9 ; le
domaine 1 tombe à **9 bits si la sonde est posée**. Le slot d'un bipède = **base + index**, base
= début de la bande bipède du film (min des slots ti=35 des keyframes). Ces largeurs sont celles
de la référence ; la largeur du domaine 7 est le même RUNTIME que `FrameConfig.IDLowBits`
(11..14 selon le film) — c'est la seule dépendance runtime, et elle n'affecte QUE le siège de
l'embarquement (§ 4).

---

## 2. Le cadrage — garde-fou de non-régression

Sur 0d76e8f1 (`TestEvtHeadHistogram`) :

```
paquets delta=40189, liste vide=34648, liste non vide=5541
GARDE cadrage: fire_events=1418 == head type36=1418 (octet 0xD2=1418)
```

Le compte d'événements de tête de type 36 égale À L'UNITÉ le compte du décodeur de tir de
production (`ScanFilmFireEvents`), qui lit l'octet 0xD2. C'est la preuve que le cadrage
`[config][continuation][R(7)]` est bit-exact et cohérent avec le décodeur existant. Le test
échoue (`t.Errorf`) si l'égalité se rompt.

Histogramme de tête (extrait), sémantiquement plausible partout : type 36 tir (1418), type 0
dégâts (1313), type 82 PlayerGameEvent (585), type 21 lunette (365), type 38 recharge (327),
type 9 ramassage (87), **type 22 sortie (10), type 8 embarquement (1)**.

---

## 3. La SORTIE (`unit_exit_vehicle`, type 22) — RÉSOLUE

Grammaire portée (`decodeVehicleEvent`) :

```
[config][cont=1][R(7)=22]
  ref0 = l'UNITÉ (domaine 1, sonde) -> OCCUPANT : slot = base + index(9)
  ref1 (domaine 1) ; ref2 (domaine 7)
  R(6) = SIÈGE
```

Validation production (`ScanFilmVehicleEvents`, 10 films véhicules, `TestEvtVehicleValidate`) :

| grandeur | mesure |
|---|---|
| sorties décodées | 209 |
| occupant présent, sonde = 1 | 209 / 209 |
| occupant dans la bande bipède | 95,5 % |
| **siège = 0** | **196 / 209 (93,8 %)** (puis 1×7, 2×3, 3×3) |

Le siège = 0 dominant correspond au « siège dominant 0 » du chantier trame (le conducteur qui
descend). Le décodage du siège de la sortie est **indépendant de la largeur runtime** (schémas
w13 et w11 donnent le même résultat : la réf 2 est gardée-absente en pratique).

Dump 0d76e8f1 (`TestEvtVehicleDump0d76`), occupant + siège + instant :

```
EXIT t=2159.15s occ_slot=512 inBand=true sonde=1 seat=2
EXIT t=2163.75s occ_slot=514 inBand=true sonde=1 seat=0
EXIT t=2184.56s occ_slot=514 inBand=true sonde=1 seat=0
EXIT t=2203.48s occ_slot=522 inBand=true sonde=1 seat=0
EXIT t=2231.19s occ_slot=515 inBand=true sonde=1 seat=0
EXIT t=2299.66s occ_slot=531 inBand=true sonde=1 seat=0
BOARD t=2418.25s occ_slot=3160 inBand=false sonde=0 seat=16
EXIT t=2420.65s occ_slot=554 inBand=true sonde=1 seat=0
EXIT t=2443.15s occ_slot=551 inBand=true sonde=1 seat=0
EXIT t=2521.95s occ_slot=559 inBand=true sonde=1 seat=0
EXIT t=2744.57s occ_slot=602 inBand=true sonde=1 seat=0
```

---

## 4. L'EMBARQUEMENT (`biped_board_vehicle`, type 8) — PARTIEL, dit honnêtement

**Ce qui marche** : le siège reproduit le « siège dominant 16 » mesuré (schéma ref1/ref2 en
domaine 7, largeur 13). Sur 12 embarquements : `seat = 16 × 5` (mode), puis 40×2, 56/8/0/46.

**Ce qui NE marche pas** : l'occupant de l'embarquement n'est pas résolu proprement. Sa réf 0
porte une sonde variable (8 fois sonde = 0, 4 fois sonde = 1 sur 12) ; quand sonde = 0 elle
donne un slot ABSOLU de 13 bits (ex. 3160 sur 0d76e8f1) qui **ne tombe pas dans la bande
bipède**. Hypothèse la plus probable : la réf 0 de l'embarquement est le **VÉHICULE** (châssis
ti=40), et l'occupant est une réf ultérieure — l'inverse de la sortie, dont la réf 0 est
directement l'unité. Non tranché : l'échantillon d'embarquements est petit (348 sur tout le
corpus, contre 5 144 sorties) et le déser board n'a pas pu être lu statiquement (§ 1.2).

**Dépendance runtime** : le siège de l'embarquement dépend de la largeur du domaine 7 (13 sur
les films de référence). Sur un film à `IDLowBits` différent (11..14), il peut se décaler ; à
re-vérifier si l'on industrialise l'embarquement.

**Conséquence sur le contrôle de cohérence demandé** (« un embarquement précède une sortie pour
le même occupant ») : impossible à établir sur 0d76e8f1, qui ne porte qu'UN embarquement, dont
l'occupant décodé (3160) ne correspond à aucun occupant de sortie. Le contrôle
`TestEvtVehicleDump0d76` rend donc 0 appariement — ce n'est pas une réfutation de la sortie
(claire), c'est la conséquence directe du trou d'embarquement ci-dessus. L'appariement
board→exit attend la résolution de l'occupant d'embarquement.

---

## 5. Comptes corpus (validation d'échelle)

`TestEvtCorpusCounts` sur le cache local (949 films lisibles) :

| événement | cache local | référence 1 367 films | ratio |
|---|---|---|---|
| board (8) | **348** sur 138 films | 374 sur 154 films | — |
| exit (22) | **5 144** sur 230 films | 5 600 sur 279 films | — |
| enter (53) | **0** | 0 (absent en arène) | — |
| board:exit | 1 : 14,8 | 1 : 15,0 | recoupe |

Le comptage porte sur l'événement de TÊTE. Le déficit (26 board, 456 exit) s'explique par les
418 films absents du cache local. `enter = 0` confirme l'absence d'entrées en arène.

**Réserve mesurée** : le chantier trame note que certaines sorties sont « portées en 2ᵉ position
d'une liste d'événements ». Le comptage de tête est donc une BORNE BASSE ; marcher la liste
entière exigerait de porter la grammaire de charge de TOUS les types d'événement (pour avancer le
lecteur), ce qui sort du périmètre focalisé de ce lot. La proximité aux comptes de référence
suggère que la part d'événements non-tête est faible pour board/exit.

---

## 6. Garde-fou ECS / arme-du-kill — vérifié

- Aucune ligne de `traverse.go` / `frame_records.go` / `fire_events.go` modifiée : le portage
  ajoute deux fichiers neufs.
- `go test ./internal/analysis/filmdec/` : **vert** (36,6 s), tous les tests non gardés passent
  (dont `ecs_table_guard`).
- Recoupement bit-exact du cadrage : `fire_events = head type36` à l'unité (§ 2). Le chemin de
  l'arme-du-kill (`ScanFilmFireEvents`) rend exactement les mêmes événements qu'avant.

---

## 7. Ce que ce décodeur débloque / reste à faire

- **Débloqué** : temps passé en véhicule par occupant (sorties datées à la milliseconde,
  occupant = bipède résolu), et le remplacement du substitut « début de trou » pour
  l'attribution du conducteur — dès que l'occupant d'embarquement est résolu.
- **Reste** (ordre de valeur) :
  1. Résoudre l'occupant de l'embarquement (probable : réf 0 = véhicule, occupant en réf
     ultérieure). Nécessite plus d'échantillons board ou la lecture du deser board (Ghidra, si le
     binaire est ré-analysé pour définir les fonctions aux adresses des thunks).
  2. Marcher la liste ENTIÈRE (événements non-tête) — porte la borne basse des comptes à l'exact,
     et récupère d'éventuelles sorties en 2ᵉ position.
  3. Confirmer la largeur runtime du domaine 7 par film (siège d'embarquement) au lieu du 13 de
     référence.

## 8. Instruments (sous garde d'environnement, à supprimer à la clôture du lot)

`internal/analysis/filmdec/event_list_test.go` :

| test | garde | mesure |
|---|---|---|
| `TestEvtHeadHistogram` | `EVT_CHUNK_DIR` | histogramme de tête + garde-fou `fire_events == type36` |
| `TestEvtCorpusCounts` | `EVT_CACHE` | comptes board/exit/enter sur tout le cache |
| `TestEvtVehicleValidate` | `EVT_CACHE` (+ `EVT_FILMS`) | occupant/sonde/siège via la production |
| `TestEvtVehicleDump0d76` | `EVT_CHUNK_DIR` | dump occupant+siège+instant + cohérence board→exit |

```
CGO_ENABLED=0 EVT_CHUNK_DIR=<data>/cache/film_chunks/0d76e8f1 \
  go test ./internal/analysis/filmdec/ -run TestEvtVehicleDump0d76 -v
```
