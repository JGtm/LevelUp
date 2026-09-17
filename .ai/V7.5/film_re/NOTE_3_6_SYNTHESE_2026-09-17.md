# Lot 3.6 — synthese de la preparation : grammaires relevees, verifiees, discordances tranchees (2026-09-17)

> Preparation du lot 3.6 (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, « porter les composants
> manquants archetype par archetype », paragraphe « Preparation facultative, sans production »).
> Aucun film decode, aucun test Go, aucun fichier de production touche. Source : les six notes de
> grammaires du 2026-09-17 (`NOTE_3_6_*_GRAMMAIRES_*_2026-09-17.md`), les six bilans JSON des
> verificateurs independants, les sept notes d'archetype du 2026-09-16, et `HaloInfinite.exe`
> par Ghidra en LECTURE SEULE (HTTP direct `127.0.0.1:8089`, programme `HaloInfinite.exe`,
> base `0x140000000` ; `decompile_function`, `disassemble_function`, `list_open_programs` ;
> rien renomme, rien commente) pour trancher les deux discordances.
>
> Instrument / Ghidra lecture seule. Archetype : tous les archetypes utiles du lot 3.6
> (`ti=9`, `11`, `12`, `35`, `40`, `42`, `43`). Groupe : synthese.
>
> Convention : « ecrivain » designe, comme dans les notes soeurs, la fonction a
> `descripteur + 0x40` = colonne `deser_addr` d'`ecs_table.tsv` ; les notes ti=11 et ti=12 B
> montrent que c'est le LECTEUR du flux (le serialiseur est a `+0x28`), et c'est bien la
> grammaire a porter. `R(n)` = lecture de `n` bits.

---

## 0. Ce que cette synthese tranche

1. **Deux discordances, toutes deux en faveur du verificateur.** Ghidra rouvert sur chacune
   (§8) ; les deux notes sont corrigees, la mention « corrige apres verification independante »
   est en tete de la section touchee :
   - `NOTE_3_6_TI43_GRAMMAIRES_A_2026-09-17.md` (`i19`, et par extension `i22`, `i23`, `i25`,
     §1, §13, §14.1) : la formule de dequantification des valeurs interieures en mode `b7 = 1`
     etait celle de la branche `b7 = 0`. Le binaire divise par `N - 2` et decale le code de 1
     (`1406d858c LEA EAX,[RCX + -0x2]`, `1406d859d LEA EAX,[R9 + -0x1]`). **Aucune largeur,
     aucun ordre, aucune condition ne change** ; seule la VALEUR flottante que le port
     calculerait change.
   - `NOTE_3_6_TI43_GRAMMAIRES_B_2026-09-17.md` §2 (`i11`, bloc lourd `FUN_140c1dd44`) :
     l'enumeration omettait le `R(8)` inconditionnel de `FUN_140c1e3f0` (`140c1dd73`) vers
     l'octet de drapeaux `+0x1c`, et presentait le `R(2)` + 3 x `R(14)` sans sa condition (bit
     `0x10` de cet octet, `140c1dfbf`). **Sans incidence sur ti=43** (branche prise pour `ti`
     0x23 / 0x28 seulement) ; l'enumeration est maintenant celle du binaire.
2. **Une reserve de notation, non discordante** (verificateur ti=11) : §1.3 l. 93 de
   `NOTE_3_6_TI11_GRAMMAIRES_2026-09-17.md` ecrit le total « 5 + somme_k (4 + corps) » alors que
   §2 definit « corps » comme EXCLUANT le bit commun `+0x08`. Lecture correcte : **« 5 +
   somme_k fente(etiquette_k) »**, la colonne « Fente (bits) » de §2 etant la bonne (etiq. 1 :
   6 bits, donc 11 au total pour une fente d'etiquette 1). La note n'est pas modifiee (hors des
   deux discordants) ; le port lira la colonne « Fente ».
3. **Perimetres de verification** : les sceptiques n'ont pas tout relu. Verifies : ti=11 `i4`
   (en-tete, etiquettes 0 et 1, chaine des `switch`) ; ti=12 `i0`, `i1`, `i2` (+ aides, 16 tags)
   et `i13`, `i14`, `i15` ; ti=35 `i57`, `i59`, `i60` (+ `FUN_14076e494` / `FUN_14076e524`) ;
   ti=43 `i19`, `i20`, `i21` et `i11`, `i12`, `i13`. NON verifies (dits par les notes, non
   contredits) : ti=12 `i3`..`i12`, `i16`..`i27` ; ti=35 `i63` ; ti=43 `i22`..`i29`, `i14`..`i18`,
   `i2`. Ces lignes restent « relevees », pas « verifiees ».

## 1. Les douze sources

| Note du lecteur | Perimetre du lecteur | Perimetre du verificateur | Points verifies | Concordants | Discordants |
|---|---|---|---|---|---|
| `NOTE_3_6_TI11_GRAMMAIRES_2026-09-17.md` | `i4` : en-tete + 15 etiquettes (14 corps) | `i4` en-tete, etiquettes 0 et 1, chaine `FUN_141e99630` -> `FUN_141e99150` -> `FUN_141e99a70` et lecteurs symetriques | 21 | 21 | 0 |
| `NOTE_3_6_TI12_GRAMMAIRES_A_2026-09-17.md` | `i0`..`i12` (13 grammaires ; `param_4` resolu) | `i0`, `i1`, `i2` + bloc de filtres (16 tags), `FUN_1406d3140`, `FUN_1406d84b4` | 57 | 57 | 0 |
| `NOTE_3_6_TI12_GRAMMAIRES_B_2026-09-17.md` | `i13`..`i19`, `i14` recontrole, `i20`..`i27` (8 instances resolues) | `i13`, `i14`, `i15` (+ dequantification, descripteurs) | 34 | 34 | 0 |
| `NOTE_3_6_TI35_GRAMMAIRES_2026-09-17.md` | `i57`, `i59`, `i60`, `i63` + `FUN_140c1e79c`, `FUN_14076e494` | `i57`, `i59`, `i60`, `FUN_14076e494` / `FUN_14076e524`, loi des largeurs `FUN_140be9b88` | 51 | 51 | 0 |
| `NOTE_3_6_TI43_GRAMMAIRES_A_2026-09-17.md` | `i19`..`i29` (11 grammaires) | `i19`, `i20`, `i21` | 3 (composants) | 2 | 1 (formule `b7 = 1`) — **tranche : note corrigee** |
| `NOTE_3_6_TI43_GRAMMAIRES_B_2026-09-17.md` | `i11`..`i21` (8 recontroles + 3 grammaires) + `i2` | `i11`, `i12`, `i13` | 3 (composants) | 2 | 1 (enumeration du bloc lourd d'`i11`) — **tranche : note corrigee** |
| **Total** | 53 grammaires (43 a porter ou partielles + 10 recontroles) | 16 composants | **169** | **167** | **2** |

Les notes d'archetype du 16/09 (`NOTE_3_6_TI9`, `TI11`, `TI12`, `TI35`, `TI40`, `TI42`, `TI43`)
fournissent les comptes de table et de golden ; la grammaire de ti=9 `i4` vient de la note du
16/09 et n'a PAS ete relue par un sceptique.

## 2. Table par archetype

> **Perimee depuis la seconde passe du 2026-09-17 : voir §10.4.**

Comptes colles des notes (`ecs_table.tsv` ; `keyframe_closure.golden`, 7 bobines du ratchet
0.A.3 — `ti=40` n'y figure que sur 5).

| Archetype | Composants (table) ; fermeture | A porter (+ partiels) | Ecrivains nommes | Grammaires relevees | Verifiees par sceptique | Partiels apres releve | Non elucides |
|---|---|---|---|---|---|---|---|
| **ti=9** joueur | 10 (8 portes, 1 partiel, 1 non porte) ; **0/1 717**, bloquant `i4` | 1 (+1) | 1 (`i4` `FUN_142ed5bc8`) | **1** — `i4` `R(32)` + `R(32)`, 64 bits inconditionnels (note du 16/09, `142ed5c70` / `142ed5d20`) | 0 (aucun sceptique sur ti=9) | `i9 managed-player-custom-input-prompt-widget` (`FUN_141fcf160`, boucle a etiquette, non relevee) | 0 |
| **ti=11** objectif | 34 (33 portes, 1 non porte) ; **0/325**, bloquant `i4` | 1 | 1 (`i4` : ecrivain `FUN_142edb5cc -> FUN_142c7023c`, lecteur `FUN_140dbe170 -> FUN_140dbe400`) | **1** composant = en-tete (`R(4)` masque, `R(1)` si `level > 1` sinon `R(32)`, par fente `R(4)` etiquette) + **15 etiquettes** (14 corps + fente vide), toutes decidables hors ligne | 1 (`i4` : 21 points) | 0 | 0 (etiquette `0xf` = flux invalide, desync declaree) |
| **ti=12** navpoint | 28 (2 portes, 26 non portes) ; **5/879**, bloquant `i1` | 26 | 26 (18 le 16/09 ; `i20`..`i27` resolus le 17/09 : UN descripteur commun `0x143d081b0`, lecteur `FUN_140dbe1bc`, nom indexe par `objet + 8`) | **26** (25 completes + `i18` partiel) + 2 recontroles (`i0`, `i14`) | 6 (`i0`, `i1`, `i2` : 57 points ; `i13`, `i14`, `i15` : 34 points) | `i18 position-offset` : branche `g == 1` (table de largeurs par defaut) non lue en statique dans la note B — **levee par `NOTE_3_6_TI35` §2.1** : bornes +-20000, loi `FUN_140be9b88` -> `3 x R(22)` constants (sous reserve : le remplisseur `FUN_140be9a14` n'a pas ete ouvert) | 0 |
| **ti=35** bipede | **64** (60 portes, 4 partiels, 0 non porte ; le plan disait 52) ; **6/1 364**, bloquant `i60` | 0 (+4) | 4 (deja dans la table) | **4** — `i57`, `i59`, `i60` completes ; `i63` partiel | 3 (`i57`, `i59`, `i60` : 51 points) | `i63 biped-action` : etiquettes 17 (`FUN_142eefd08`, 2 x `FUN_1406d84b4` largeurs non relevees) et 20 (`FUN_1406d84b4` largeur non relevee) | `i63` etiquettes 18 (`FUN_14319c6f8`), 19 (`FUN_142eefe1c`), 21 (`FUN_142eeff18`), 23 (`FUN_142ef004c`), `>= 24` (`FUN_142ef0494`) : nommees, non ouvertes |
| **ti=40** vehicule | 48 (30 portes, 2 partiels, 16 non portes) ; **0/777** sur 5 bobines, **bloquant VIDE** | 16 (+2) | 16 (`i30`..`i42`, `i45`, `i46`, `i47`) | **0** — aucune note de grammaire ; `i43` = `simulation-state` = ti=35 `i60` (releve par ricochet) ; `i2` = meme lecteur `FUN_140c5f7ec` que ti=43 `i2` (grammaire relevee dans `_TI43_B` §13) | 0 | `i2`, `i43` | **16 grammaires** ; **le bloquant n'est pas etabli** : la preuve bornante demande un film (hors perimetre sans decodage) |
| **ti=42** arme au sol | 21 (21 portes) ; **10/2 087**, **bloquant VIDE** | 0 | — | — (negatif MESURE : rien a porter) | — | — | pas un port : instruction de cadre (D8 du 1.9.1 bis ; garde-rail G2 `ECS_TABLE_FILM`) — `[~]` |
| **ti=43** device | 41 (18 portes, 1 partiel, 22 non portes) ; **0/4 106**, bloquant `i19` | 22 (+1) | 22 (`i19`..`i40`) | **12** — `i19`..`i29` (11, note A) + `i2` (grammaire entiere, note B §13) ; + 8 recontroles `i11`..`i18` (concordance Go 8/8) | 6 (`i19`, `i20`, `i21` ; `i11`, `i12`, `i13` : 2 discordants, tranches, notes corrigees) | `i2 object-forward-and-up-dynamic-precision` : partiel de CONTEXTE (branche `param_5 > 1` tabulee a 2 ; branche `DAT_145121140 == 1` non modelisee dans `decodeObjectForwardAndUpDynPrec`) ; doc et table « mode C=1 NON porte » en retard sur le code | **`i30`..`i40` (11)** : ecrivains nommes, grammaires NON relevees — aucun lecteur n'a pris ce groupe (la note B a lu les indices 11..21 de la table, pas les rangs 11..21 de la liste `device-*`) |

## 3. Totaux

> **Perimee depuis la seconde passe du 2026-09-17 : voir §10.5.**

- **Population du lot 3.6** : **74 lignes** (66 non portees + 8 partielles) sur 246 composants
  des sept archetypes utiles (10 + 34 + 28 + 64 + 48 + 21 + 41).
- **Ecrivains nommes** : **74 / 74** — 57 par la vague du 16/09 (1 + 18 + 16 + 22), 8
  `visual-state-groups` par la note ti=12 B du 17/09 (un descripteur, un lecteur), et 9 deja
  dans `ecs_table.tsv` (ti=9 `i9`, ti=11 `i4`, ti=35 x4, ti=40 `i2` / `i43`, ti=43 `i2`).
- **Grammaires relevees a l'ecrivain** : **44 / 74** = 41 completes + 3 partielles (ti=12
  `i18`, ti=35 `i63`, ti=43 `i2`). Par archetype : ti=9 1, ti=11 1, ti=12 26, ti=35 4, ti=43 12.
  Les six notes du 17/09 portent 53 grammaires : 43 de cette population + 10 recontroles de
  composants deja portes (ti=12 `i0`, `i14` ; ti=43 `i11`..`i18`), tous concordants avec le Go.
- **Verifiees par sceptique** : **16 composants** (11 de la population 3.6 : `i4` ti=11 ; `i1`,
  `i2`, `i13`, `i15` ti=12 ; `i57`, `i59`, `i60` ti=35 ; `i19`, `i20`, `i21` ti=43 — et 5 portes
  recontroles : `i0`, `i14` ti=12 ; `i11`, `i12`, `i13` ti=43), **169 points de controle, 167
  concordants, 2 discordants** (tranches, §8).
- **Restent sans grammaire** : **30 lignes** — ti=43 `i30`..`i40` (11), ti=40 (16 + `i2` + `i43`,
  ces deux couverts par ricochet), ti=9 `i9` (1) ; soit **28 a relever reellement**.
- **Non elucide au sens strict** (ouvert dans le binaire et non ferme) : les 5 corps
  d'etiquette `>= 18` d'`i63` (ti=35) ; le bloquant de ti=40 ; le remplisseur `FUN_140be9a14`
  de la table de largeurs par defaut (branche `g == 1` d'`i18` et de `FUN_14076e494`).

## 4. Constats transverses (etablis par plusieurs notes, a porter une fois)

1. **`param_4` = niveau du registre du film.** Le slot `descripteur + 0x10` est `MOV EAX, k ; RET`
   (`0x14117b4a0` = 1, `0x141179610` = 2, `0x14117e0e0` = 3, `0x140c85020` = 4, `0x1405f0ac0`
   = 0) ; le dispatcheur `FUN_14076cb60` appelle `vtable[0]()` puis passe la valeur en 5e
   argument ; le thunk `+0x38` (`MOV R9D,[RSP+0x28] ; JMP [RAX+0x30]`) la remet en `R9D` au
   lecteur `+0x40`. Concordance 4/4 avec `component_param4.go` (mesures live ti=35) et avec la
   colonne `level` du registre lue dans le film (`registry.go:79`) ; ti=11 `i4` = 2, ti=12 `i2`
   = 3, `i3`..`i6` = 2, `i19` = 2, `i20`..`i27` = 1. **Consequence : `paramByComponent` se
   remplace par `Archetype.Level(i)`** — le film est autoportant, la constante du binaire n'est
   qu'une verification du build courant. Limite honnete (note A) : le site d'appel exact
   `+0x10 -> param_4` n'a pas ete lu ; la preuve est la calibration 4/4 + la coherence
   lecteur / serialiseur.
2. **Un seul noyau « bloc de filtres / groupe d'etat visuel »** (`FUN_140dbe400` cote lecteur,
   `FUN_142c7023c` cote serialiseur ; 15 etiquettes, table etiquette -> vtable -> corps dans la
   note ti=11 §2) sert ti=11 `i4`, ti=12 `i2`..`i6` (avec leurs queues), ti=12 `i19` (mode 0) et
   ti=12 `i20`..`i27` (mode 1, prefixe `R(32)` + queue). A ecrire une fois, parametre par
   `level` et par le mode `m` (constante de site).
3. **Contrat de `FUN_1406d84b4`** (confirme par 4 notes et 4 verificateurs) : `n` a `[RSP+0x20]`
   chez l'appelant, `min` en `XMM2`, `max` en `XMM3`, `f6` / `f7` ne changent que la valeur.
   `f7 = 0` : `min + (q + 0,5) * (max - min) / N` ; `f7 = 1` : `q = 0 -> min`, `q = N - 1 -> max`,
   sinon `min + (q - 1 + 0,5) * (max - min) / (N - 2)` ; `f6 = 1` : `N = 2^n - 1` et
   `2q = N - 1 -> (min + max) / 2` exact. **Le `dequantMidpoint` d'`i14` (`(q + 0,5) / N`,
   « convention RETENUE ») ne correspond pas a l'ecrivain** (`i13` / `i14` / `i15` : 253 pas,
   saturation aux codes 0 et 254, milieu exact au code 127 ; code 255 hors plage, a saturer ou
   marquer invalide). Toute publication d'une valeur dequantifiee par le lot 3.6 doit passer par
   UNE fonction commune fidele a ce contrat.
4. **Une seule entree de profil pour les references d'entite** : `w_dom = ceil(log2(cardinal du
   domaine))` avec le bit de garde `DAT_144706104` ecrit dans le film (`1429874ab`) ; domaine 0
   (ti=11 etiq. 6 / 0xc / 0xd / 0xe, ti=12 memes tags, ti=35 `i57`), domaine 1 avec sonde `R(1)`
   (ti=43 `i24` / `i29`, ti=35 `i59`), domaine 5 (ti=35 `i59`). Deja portee (`varWidthBits`,
   `IDLowBits`) ; ne pas recoder, ne jamais figer 13.
5. **La position absolue `FUN_14076e494(flux, dst, 0x10, 0, ., 0)`** (ti=35 `i57` / `i59` /
   `i60`, ti=12 `i18` via `FUN_14076e524`) : `0x10` est une CLASSE de precision, pas une largeur ;
   porte `R(1)` ; porte a 1 -> `3 x R(22)` constants (bornes par defaut +-20000, loi
   `FUN_140be9b88`) ; porte a 0 -> `R(ceil_log2(nb_regions))` + `3 x R(W_axe)` de la region de
   la carte (`MapQuantEntry.AxisWidths`, `I0Layout`). La garde runtime `FUN_14076f91c` (0 bit,
   `fullPrecisionGate`) ne se porte pas. C'est la condition ecrite du kill-switch
   `simStateComplete`.
6. **Documents a corriger dans le commit qui porte** (zero fix ici) : note de methode §2
   (`0x1404ab600` / `0x1411c8f80` et non `0x14049b600` / `0x141c8f880` ; `+0x10` = niveau, pas
   constante de famille), §4 (`FUN_1406d49c4` = ecrivain d'UN bit), §6 (`+0x40` est le LECTEUR) ;
   `TI11_SPEC_10_FEUILLES.md` (« sel6..15 assert », « sel5 recursion » : faux) ;
   `NOTE_3_6_TI11_2026-09-16.md` §3 (« 6 a 15 non atteints ») ; `HANDOFF_FRAME_DECODER_L3.md`
   l. 230 (provenance de `param_4` prouvee) ; godoc et ligne `partiel` de ti=43 `i2` ;
   adresses des sous-lecteurs de `consumeObjectLowFrequency` ; `ecs_table.tsv` notes des 4
   `partiel` de ti=35 (« RUNTIME » x2, « masque RAM », « queue conditionnelle »).

## 5. Ce que chaque port demandera

### 3.6.a — ti=9 (joueur)

- **Cases** : 1 (`i4` : `R(32)` + `R(32)`, 64 bits, meme patron que `i5
  managed-player-active-mission-name-component` deja porte).
- **Entrees de profil a creer** : 0.
- **Inconnues restantes** : `i9` (`partiel`, `FUN_141fcf160` : portes + `R(32)` + `R(3)`
  compteur, boucle a etiquette non relevee) — prochain bloquant probable si la fermeture ne
  monte pas a 1 717/1 717 apres `i4`. La grammaire d'`i4` n'a pas ete relue par un sceptique :
  la reverifier sur pieces au port (deux `ADD dword ptr [+0x2c], 0x20` a `142ed5c70` /
  `142ed5d20`, ecritures `142ed5ca1` / `142ed5d5d`).
- **Gate** : fermeture ti=9 sur les 7 bobines, 0 -> 1 717.

### 3.6.b — ti=11 (objectif)

- **Cases** : 1 composant, mais **1 noyau + 15 branches** : `interactionFilterList(level)` =
  `R(4)` masque ; `R(1)` si `level > 1` sinon `R(32)` ; pour chaque bit pose : `R(4)` etiquette
  puis la fente (etiq. 0 : 4 bits ; 1 : 6 ; 2 : 37 ; 3 / 7 : 9 ; 4 : 14 ; 5 : 8 + n x {1, 14} ;
  6 : 9 + n x {1, 3 + w0} ; 8 : 13 ; 9 : 37 ; 0xa : 45 ; 0xb : 6 / 11 ; 0xc / 0xd : 38 / 40 + w0 ;
  0xe : 39 / 41 + w0 ; 0xf : flux invalide).
- **Entrees de profil** : `level` du composant (registre du film ; 2 au build courant -> `R(1)`)
  et `w0` (domaine 0). Aucune nouvelle primitive de lecture.
- **Inconnues restantes** : 0.
- **Gate** : 0/325 -> 325/325 ; un intermediaire designe une etiquette mal lue. Le meme noyau
  debloque ti=12 `i5` / `i6` (bloc seul) et prefixe `i2` / `i3` / `i4`, `i19`, `i20`..`i27`.

### 3.6.b — ti=12 (navpoint)

- **Cases** : 26 composants, **~19 lecteurs** : 10 champs plats (`i1` `R(8)`, `i7` `R(8)`, `i8`
  `R(32)`, `i10` `2 x R(7)` = `consumeObjectiveTimers`, `i11` / `i12` `R(17)` pas de 50 ms,
  `i13` / `i15` `R(8)` = `i14`, `i16` `R(5)`, `i17` `R(32)`) ; `i9` liste taguee
  (`consumeObjectiveFormattedText` sous une boucle `R(8)` x [`R(32)` + corps]) ; `i2`..`i6` sur
  le noyau de filtres + queues (`i2` : `2 x R(16)` [-1, 1000] puis par filtre `2 x R(16)` puis
  `K x R(3)` ; `i3` / `i4` : `R(1)` puis par filtre `R(1)` puis `K x R(3)` ; `i5` / `i6` : bloc
  seul) ; `i19` : `0 bit` au niveau 2 (sinon `R(8)` + groupes mode 0) ; `i20`..`i27` : UN lecteur
  `R(1)` + groupe mode 1 (`69 + somme_b (39 + corps_b)`), parametre par `k` ; `i18` : `R(1)` +
  [`R(IndexW)`] + `3 x R(W_axe)` (le lecteur quantifie d'`i0` SANS sa porte externe, classe 16).
- **Entrees de profil** : `level` (`i2` = 3, `i3`..`i6` = 2, `i19` = 2, `i20`..`i27` = 1 ; a lire
  au registre du film — remplace `paramByComponent`) ; `w0` ; `IndexW` + `W_axe` par region de
  la carte (`i18`, deja `MapQuantCatalog` / `I0Layout`). Constante de site : `m` (0 pour `i19`,
  1 pour `i20`..`i27`).
- **Inconnues restantes** : branche `g == 1` d'`i18` (table par defaut -> `3 x R(22)` selon
  ti=35 §2.1, remplisseur non ouvert : mesurer au `DesyncAt`, pas supposer) ; alignement de la
  dequantification `i13` / `i14` / `i15` (§4.3).
- **Gate** : 5/879 -> 879/879 ; le ratchet 0.A.3 doit voir le bloquant nomme avancer d'un
  index a chaque port (`i1` d'abord).

### 3.6.c — ti=35 (bipede)

- **Cases** : 4 composants deja `partiel`, tous avec un lecteur Go existant a completer :
  `i60` : rien de grammatical — brancher la SOURCE des largeurs d'axe (`IndexW` =
  `EffectiveRegionIndexBits()`, `W_axe` = `MapQuantEntry.AxisWidths` ; porte a 1 -> `3 x R(22)`
  au lieu de `AbsoluteAxisW = 14` uniforme) et retirer `simStateComplete` (regle 11) ; `i57` :
  branche `a == 1` de `FUN_142f262d4` (`R(6)`, `R(1)`, reference dom 0, `FUN_142f04664` absolue
  ou RELATIVE `R(2) + 3 x R(13) + R(1)[+R(16)]` — nouveau lecteur partage avec `i59`) ; `i59` :
  remplacer la grammaire mesuree par celle de l'ecrivain (`s = R(3) + 1`, `r = R(1)` +
  reference dom 1 avec sonde, `FUN_142f04664`, `R(6)`, 8 corps) ; `i63` : `n2 = pop(m0) +
  pop(m1) + pop(m2 & 0x1ff)` calcule sur les 96 premiers bits (remplace `bipedActionLoop2Count
  = 0`), etiquettes 6..16 portees, `ported = false` sur 17..31.
- **Entrees de profil** : bornes AABB / nombre de regions (deja) ; cardinal des domaines + bit
  `DAT_144706104` (deja) ; `level` (`i57` / `i59` = 2 -> registre) ; `fullPrecisionGate` (deja).
  Aucune nouvelle.
- **Inconnues restantes** : `i63` etiquettes 17 (2 largeurs), 20 (1 largeur), 18, 19, 21, 23,
  `>= 24` (5 corps non ouverts) — a ouvrir si un corpus les fait apparaitre (`n1 = 0` dans le
  cas commun).
- **Gate** : 6/1 364 -> 1 364/1 364 ; porte du meme geste ti=40 `i43`.

### 3.6.d — ti=42 et ti=43

- **ti=42** : **rien a porter** (`[~]`, D8 du 1.9.1 bis). Ce que le lot demande est une
  INSTRUCTION de cadre sur film : (1) borne de fin de record (decoupage du catalogue installe),
  (2) composant present au registre du film et absent d'`ecs_table.tsv` (garde-rail G2). Hors
  perimetre sans decodage ; a sortir du libelle « ti=42 et ti=43 ferment a 100 % ».
- **ti=43 — cases** : 22 composants ; **11 relevees** (`i19`..`i29`) portables avec 7 lecteurs
  existants (`ReadBit`, `ReadBits(32)`, `ReadBits(n)` par tranches de 64, `consumeGateR(br, 8)`,
  `consume1408f0ac4(br, 1)`, dequantification `Q`) : `i19` 42 bits, `i20` 256 bits bruts a ranger
  tels quels, `i21` 33 / 41, `i22` / `i23` `Q(14; 0..1)`, `i24` / `i29` `R(1)` + [`R(1)` sonde +
  `R(W)` + `R(2)`], `i25` `Q(8; 0..60)`, `i26` 64, `i27` 7, `i28` 1 ; **11 a relever AVANT le
  port** (`i30`..`i40`, ecrivains nommes, 8 dans le voisinage `FUN_142f02*`) — porter 11/22 ne
  ferme aucun record (D5 du 1.9.1 bis).
- **Entrees de profil** : `W` de la categorie 1 (`IDLowBits`, deja) pour `i24` / `i29` ; pour
  `i2` : `level` / `param_5` (tabule 2) et la garde `DAT_145121140` (`FullPrecision`, a brancher
  dans `decodeObjectForwardAndUpDynPrec` ou a documenter). Aucune nouvelle.
- **Inconnues restantes** : les 11 grammaires `i30`..`i40` ; `i2` peut redevenir le bloquant une
  fois les 22 portes (a lire au golden regenere).
- **Gate** : 0/4 106 -> 4 106/4 106 apres les 22.

### 3.6.e — ti=40 (vehicule)

- **Cases** : 16 `vehicle-*` nommes, **0 grammaire relevee** ; `i43` gratuit avec 3.6.c ;
  `i2` = le lecteur de ti=43 `i2` (§13 de la note B).
- **Entrees de profil** : inconnues tant que les grammaires ne sont pas relevees ; `i47
  vehicle-low-frequency-component` est le suspect n 1 (famille des composants larges).
- **Inconnues restantes** : **le bloquant lui-meme** — le golden est muet (0/777, colonne vide).
  Le lot commence par une mesure sur film (preuve bornante du 1.9.1 bis rejouee sur ti=40), pas
  par un port ; la grammaire du composant designe se releve ensuite sans film.
- **Gate** : 0/777 -> 777/777 sur les 5 bobines qui portent ti=40.

## 6. Ordre conseille

1. **3.6.a ti=9** — un `case`, aucune dependance, gate immediat.
2. **3.6.b ti=11** — le noyau du bloc de filtres (15 etiquettes) : il ferme ti=11 et sert
   ti=12 (`i2`..`i6`, `i19`, `i20`..`i27`). Poser ici `level` lu au registre (§4.1) et `w0`.
3. **3.6.b ti=12** — dans l'ordre le moins cher : `i1`, `i7`, `i8`, `i10`, `i11`, `i12`, `i13`,
   `i15`, `i16`, `i17` (plats), `i9`, `i5` / `i6`, `i3` / `i4`, `i2`, `i19`, `i20`..`i27`, `i18`
   (dependance carte en dernier). Aligner `dequantMidpoint` dans le lot qui porte `i13` / `i15`.
4. **3.6.c ti=35** — `i60` (kill-switch) puis `i57`, `i59` (lecteur de position relative
   partage), puis `i63` (popcount + etiquettes 6..16). Porte ti=40 `i43` au passage.
5. **Preparation complementaire** (sans film, un lecteur) : relever `i30`..`i40` de ti=43 ;
   puis **3.6.d ti=43** en un seul lot de 22. Re-statuer `i2`. ti=42 : `[~]` D8, a sortir du
   libelle.
6. **3.6.e ti=40** — mesure sur film d'abord (preuve bornante), releve cible ensuite, port enfin.

## 7. Ce qui reste non elucide, nomme

> **Perimee depuis la seconde passe du 2026-09-17 : voir §10.6.**

| Ou | Quoi | Ce qui le leve |
|---|---|---|
| ti=35 `i63` | corps d'etiquette 17 (2 largeurs), 20 (1 largeur), 18, 19, 21, 23, `>= 24` (`FUN_142eefd08`, `FUN_14319c6f8`, `FUN_142eefe1c`, `FUN_142ef0388` inline, `FUN_142eeff18`, `FUN_142ef004c`, `FUN_142ef0494`) | decompile sans film ; ou `ported = false` tant qu'un corpus ne les fait pas apparaitre |
| ti=40 | le bloquant (golden muet) ; 16 grammaires | preuve bornante sur film, puis decompile |
| ti=43 `i30`..`i40` | 11 grammaires (ecrivains nommes) | decompile sans film |
| ti=9 `i9` | boucle a etiquette de `FUN_141fcf160` | decompile sans film (si le golden le nomme apres `i4`) |
| ti=12 `i18`, ti=35 position absolue | remplisseur `FUN_140be9a14` de la table de largeurs par defaut (`3 x R(22)` deduit de la loi, pas lu) | decompile ; ou `DesyncAt` au port |
| ti=12 / ti=43 | site d'appel exact `+0x10 -> param_4` (calibration 4/4, pas de lecture du site) | `get_xrefs_to` sur les descripteurs, si le pilote l'exige |
| ti=12 `i13` / `i14` / `i15` | code 255 d'un `R(8)` en `f6 = 1` (hors plage) | decision de port : saturer ou invalider |

## 8. Les deux discordances, tranchees sur pieces

### 8.1 `FUN_1406d84b4`, mode `b7 = 1` — la note ti=43 A avait tort

Desassemblage relu (`disassemble_function 0x1406d84b4`, colle) :

```
1406d851e: CMP byte ptr [RSP + 0x38],0x0
1406d8523: JNZ 0x1406d8577                 ; b7 != 0
1406d8525: MOVD XMM0,ECX                   ; --- branche b7 = 0 ---
1406d852f: SUBSS XMM1,XMM2
1406d8533: DIVSS XMM1,XMM0                 ; pas = (max - min) / N
1406d853f: MULSS XMM0,XMM1                 ; q * pas
1406d8543: MULSS XMM1,dword ptr [0x143cd84b0]   ; pas * 0,5
1406d854b: ADDSS XMM0,XMM2
1406d854f: ADDSS XMM0,XMM1                 ; min + q * pas + pas / 2
1406d8577: TEST R9D,R9D                    ; --- branche b7 = 1 ---
1406d857a: JZ 0x1406d8644                  ; q == 0 -> min
1406d8580: LEA EAX,[RCX + -0x1]
1406d8583: CMP R9D,EAX
1406d8586: JZ 0x1406d863c                  ; q == N - 1 -> max
1406d858c: LEA EAX,[RCX + -0x2]            ; N - 2
1406d858f: MOVAPS XMM1,XMM3
1406d8592: SUBSS XMM1,XMM2                 ; max - min
1406d8596: MOVD XMM0,EAX
1406d859a: CVTDQ2PS XMM0,XMM0
1406d859d: LEA EAX,[R9 + -0x1]             ; q - 1
1406d85a1: DIVSS XMM1,XMM0                 ; pas = (max - min) / (N - 2)
1406d85a5: MOVD XMM0,EAX
1406d85a9: CVTDQ2PS XMM0,XMM0
1406d85ac: MULSS XMM0,XMM1                 ; (q - 1) * pas
1406d85b0: MULSS XMM1,dword ptr [0x143cd84b0]   ; pas * 0,5
1406d85b8: ADDSS XMM1,XMM2                 ; min + pas / 2
1406d85bc: JMP 0x1406d854f                 ; + (q - 1) * pas
1406d863c: MOVAPS XMM0,XMM3                ; max
1406d8644: MOVAPS XMM0,XMM2                ; min
```

Verdict : **verificateur confirme**. Les ecrivains `i19`, `i22`, `i23`, `i25` posent
`[RSP+0x30] = 1` (= `b7`), donc la branche `1406d8577` ; la note A appliquait la formule de
`1406d8525..1406d854f`. Corrige dans `NOTE_3_6_TI43_GRAMMAIRES_A_2026-09-17.md` (en-tete, §1, §2,
§5, §8, §13, §14.1). Les notes ti=12 A §1, ti=12 B §2, ti=35 §2 et ti=43 B §1 avaient deja la
bonne formule. Exemple `i19` (`n = 10`, `[0, 10]`) : brut 1 -> 0,5 x 10 / 1022 = 0,00489 (la note
disait 15 / 1024 = 0,01465).

### 8.2 `FUN_140c1dd44`, enumeration du bloc lourd d'`i11` — la note ti=43 B avait tort

Decompile relu (`decompile_function 0x140c1dd44`, tete collee) :

```
FUN_14080d69c(param_1,param_2,param_1,0xffffffff);
FUN_140c1e3f0(param_2);                       // <- absent de l'enumeration
uVar4 = FUN_1407f2058(param_2);
```

Desassemblage : `140c1dd67 CALL 0x14080d69c` ; `140c1dd6c LEA R8,[R14 + 0x1c]` ;
`140c1dd70 MOV RCX,RSI` ; `140c1dd73 CALL 0x140c1e3f0` ; `140c1dd7b CALL 0x1407f2058`.
`FUN_140c1e3f0(flux, ., byte *out)` decompile : `*(int *)(param_1 + 0x2c) += 8` sur les deux
chemins, `*param_3 = bVar4` = **`R(8)`** dans `obj+0x74+0x1c`. Plus loin :
`140c1dfbf TEST byte ptr [R14 + 0x1c],0x10` ; `140c1dfc4 JNZ 0x140c1e269` ; chemin a 0 :
`140c1dfca MOV byte ptr [R14 + 0x1d],R15B` (= 0), vecteur `DAT_143cf0630` ; chemin a 1 (decompile
`else` du `if ((*(byte *)(param_1 + 0x1c) & 0x10) == 0)`) : `R(2)` en ligne (`+0x2c += 2`) ->
`+0x1d`, puis `140c1e2a7 MOV R9D,0xe` ; `140c1e2ad MOV byte ptr [R14 + 0x1d],BL` ;
`140c1e2b1 CALL 0x140c1e9d4` (`FUN_140c1e9d4(flux, ., out, n)` = boucle de 3 x `R(n)`, decompile :
`+0x2c += param_4` par tour) = **3 x `R(14)`**, dequantifies par la ligne `(+0x1d) * 0x18` de
`DAT_143b8c6f0`. Queue : `140c1e01c JMP 0x14080d69c` (appel de queue, `R8 = R14+0x4c`).

Verdict : **verificateur confirme** — un `R(8)` inconditionnel omis, une condition non dite.
Corrige dans `NOTE_3_6_TI43_GRAMMAIRES_B_2026-09-17.md` §2 (enumeration reecrite depuis le
decompile ; les largeurs des sous-lecteurs que je n'ai pas rouverts — `FUN_140c1e31c`,
`FUN_1407f1f24`, `FUN_1407f1e4c`, `FUN_14076dc04`, `FUN_1424cd17c`, `FUN_1424cd150` — restent
citees par leur nom, sans largeur inventee). Sans incidence sur ti=43 : la grammaire `R(1)`
pour `ti = 0x2b` etait et reste concordante.

## 9. Journal des appels Ghidra de la synthese

Sept appels HTTP en lecture, aucune ecriture : `list_open_programs` (un seul programme,
`HaloInfinite.exe`, base `0x140000000`) ; `disassemble_function 0x1406d84b4` (3 919 octets) ;
`decompile_function 0x1406d84b4` ; `decompile_function 0x140c1dd44` ; `disassemble_function
0x140c1dd44` (189 Mo : le serveur a deroule bien au-dela de la fonction, plage utile extraite par
`grep` sur les adresses, fichier purge) ; `decompile_function 0x140c1e3f0` et `0x140c1e9d4`.

---

## 10. Addendum (seconde passe, 2026-09-17)

> Section demandee sous le nom « 8. Addendum (seconde passe) » ; numerotee **10** parce que §8
> (les deux discordances de la premiere passe) et §9 (journal Ghidra) existaient deja. Elle
> REMPLACE, pour les comptes, les tables §2, §3 et §7 : au 2026-09-17 apres seconde passe, ce
> sont les tables §10.4, §10.5 et §10.6 qui font foi.
>
> Instrument / Ghidra lecture seule (HTTP direct `127.0.0.1:8089`, programme `HaloInfinite.exe`,
> base `0x140000000` : `list_open_programs`, `read_memory`, `get_function_by_address`,
> `disassemble_function` ; aucune ecriture, aucun renommage, aucun commentaire, aucun tag).
> Archetypes touches : `ti=9`, `ti=35`, `ti=43`. Groupe : synthese, seconde passe.
> Aucun film decode, aucun test Go, aucun fichier de production touche.

### 10.1 Les trois sources de la seconde passe

| Note du lecteur | Perimetre du lecteur | Perimetre du verificateur | Points | Concordants | Discordants |
|---|---|---|---|---|---|
| `NOTE_3_6_TI43_GRAMMAIRES_C_2026-09-17.md` | `ti=43` `i30`..`i35` (6 grammaires) | `i30`, `i31`, `i32` (granularite composant) | 3 | 3 | 0 |
| `NOTE_3_6_TI43_GRAMMAIRES_D_2026-09-17.md` | `ti=43` `i36`..`i40` (5 grammaires) | `i36`, `i37`, `i38` (granularite champ : 11 champs, 3 constantes, 3 structures) | 17 | 17 | 0 |
| `NOTE_3_6_TI9_TI35_RESTES_2026-09-17.md` | `ti=9` `i9` (grammaire entiere) + `ti=35` `i63` corps d'etiquette 17..31 | `ti=9` `i9` et `ti=35` `i63` integralement (60 adresses de « preuve » rejouees) | 20 | 19 | 1 (3 adresses de slots au §1.1) — **tranche : note corrigee** |
| **Total seconde passe** | **13 grammaires** (11 nouvelles `ti=43`, 1 nouvelle `ti=9`, 1 partielle `ti=35` achevee) | **8 composants** | **40** | **39** | **1** |

Granularite heterogene assumee : le verificateur C compte au composant, le verificateur D au
champ, le verificateur ti9ti35 au champ et a l'adresse. Les totaux cumules de §10.5 additionnent
ces points tels que rendus, sans les renormaliser.

### 10.2 La discordance de la seconde passe, tranchee sur pieces — le verificateur a raison

Objet : `NOTE_3_6_TI9_TI35_RESTES_2026-09-17.md` §1.1, ligne « descripteur » du tableau de la
chaine des quatre pas — trois adresses de slots du descripteur `ti=9 i9` `0x143d08930`.

Ghidra rouvert (lecture seule), `read_memory address=0x143d08930 length=96`, hex colle :

```
b01a1941 01000000   +0x00 -> 0x141191ab0
d0ce7640 01000000   +0x08 -> 0x14076ced0
a0b41741 01000000   +0x10 -> 0x14117b4a0
400cfd41 01000000   +0x18 -> 0x141fd0c40   (accesseur de nom)
00b64a40 01000000   +0x20 -> 0x1404ab600   (la note ecrivait 0x14049b600)
7000fd41 01000000   +0x28 -> 0x141fd0070   (compagnon)
808f1c41 01000000   +0x30 -> 0x1411c8f80   (la note ecrivait 0x141c8f880)
9cce7640 01000000   +0x38 -> 0x14076ce9c
60f1fc41 01000000   +0x40 -> 0x141fcf160   (ecrivain)
00b64a40 01000000   +0x48 -> 0x1404ab600   (la note ecrivait 0x14049b600)
```

Resolution des quatre adresses en jeu (`get_function_by_address`, colle) :

```
0x1404ab600 -> Function: Reserve at 1404ab600        (entree, corps 1404ab600 - 1404ab602)
0x1411c8f80 -> Function: FUN_1411c8f80 at 1411c8f80  (entree, corps 1411c8f80 - 1411c8f80)
0x14049b600 -> Function: FUN_14049b57c at 14049b57c  (milieu de corps, PAS une entree)
0x141c8f880 -> Function: FUN_141c8f3b0 at 141c8f3b0  (milieu de corps, PAS une entree)
```

**Verdict : verificateur confirme.** Corrige dans `NOTE_3_6_TI9_TI35_RESTES_2026-09-17.md`
§1.1, avertissement « corrige apres verification independante » en tete de section. Portee de
l'erreur : NULLE sur la grammaire — aucun champ du flux, aucune largeur, aucune condition ; les
deux slots qui portent la demonstration (`+0x18` = accesseur de nom, `+0x40` = ecrivain) etaient
exacts. **Cause racine, deja au registre** : ce sont les constantes de signature de famille de
`NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` §2 (l. 42, 44, 47, 49-50), que la note ti9ti35 a
recopiees telles quelles ; quatre notes du 17/09 (`_TI11` l. 61, `_TI12_A` l. 398, `_TI12_B`
l. 51, `_TI43_D` l. 59 et §8.8) les avaient deja signalees. La correction de la note de methode
reste inscrite au §4.6 (« a corriger dans le commit qui porte », zero fix ici).

### 10.3 Un residu leve au passage — `FUN_1406d84b4` preserve `XMM2` et `XMM3`

Le verificateur D affirme, sans contredire sa note, que `FUN_1406d84b4` ne fait que LIRE
`XMM2`/`XMM3` — ce qui est exactement la question laissee ouverte par
`NOTE_3_6_TI9_TI35_RESTES_2026-09-17.md` §4 (residu 2 : bornes du second `R(15)` du corps
`t = 17` d'`i63`, non rechargees apres le premier appel). Verifie ici sur pieces
(`disassemble_function 0x1406d84b4`, 129 instructions, fonction entiere) :

- toutes les occurrences de `XMM2` / `XMM3` sont en position SOURCE :
  `1406d8529 MOVAPS XMM1,XMM3` ; `1406d852f SUBSS XMM1,XMM2` ; `1406d854b ADDSS XMM0,XMM2` ;
  `1406d858f MOVAPS XMM1,XMM3` ; `1406d8592 SUBSS XMM1,XMM2` ; `1406d85b8 ADDSS XMM1,XMM2` ;
  `1406d85c4 CVTSS2SD XMM1,XMM2` ; `1406d85c8 CVTSS2SD XMM0,XMM3` ; `1406d863c MOVAPS XMM0,XMM3` ;
  `1406d8644 MOVAPS XMM0,XMM2` ;
- les seules destinations XMM de toute la fonction sont `XMM0` et `XMM1` (`MOVD`, `MOVAPS`,
  `CVTDQ2PS`, `CVTPD2PS`, `CVTSS2SD`, `ADDSS`, `SUBSS`, `MULSS`, `DIVSS`, `XORPS`) ;
- **aucune instruction `CALL`** dans la fonction (les seuls transferts sont six `JMP` internes :
  `1406d85bc`, `1406d85dc`, `1406d8637`, `1406d863f`, `1406d8647`, `1406d8673`), donc aucun appele
  ne peut les ecraser.

**Consequence** : les bornes posees en `XMM2`/`XMM3` avant le premier `Q(n)` survivent aux
appels suivants tant que l'appelant ne les recharge pas. Cela leve le residu `t = 17` d'`i63`
(le second `R(15)` se dequantifie sur les memes bornes `[0 ; 60,0]`, `DAT_143cd84e4`) ET
confirme la lecture du groupe D pour `i37` (borne `600,0f`, `DAT_143d13304`, tenue sur les
trois `Q(10)`). Rappel : **0 bit en jeu** — c'est une question de VALEUR, jamais de largeur.

### 10.4 Table par archetype — mise a jour (remplace §2)

| Archetype | Composants (table) ; fermeture | A porter (+ partiels) | Ecrivains nommes | Grammaires relevees | Verifiees par sceptique | Partiels apres releve | Non elucides |
|---|---|---|---|---|---|---|---|
| **ti=9** joueur | 10 ; **0/1 717**, bloquant `i4` | 1 (+1) | 2 | **2** — `i4` (note du 16/09) et **`i9` (seconde passe, note ti9ti35 §1)** | 1 (`i9` : 20 points, 19 concordants) | **0** (`i9` etait partiel : leve) | 0 |
| **ti=11** objectif | 34 ; **0/325**, bloquant `i4` | 1 | 1 | **1** (en-tete + 15 etiquettes) | 1 (`i4` : 21 points) | 0 | 0 |
| **ti=12** navpoint | 28 ; **5/879**, bloquant `i1` | 26 | 26 | **26** (25 completes + `i18` partiel) | 6 | `i18 position-offset` (branche `g == 1`, levee par `NOTE_3_6_TI35` §2.1 sous reserve du remplisseur) | 0 |
| **ti=35** bipede | 64 ; **6/1 364**, bloquant `i60` | 0 (+4) | 4 | **4** — `i57`, `i59`, `i60` et **`i63` desormais COMPLET** (32/32 etiquettes, note ti9ti35 §2) | 4 (`i57`, `i59`, `i60` : 51 points ; `i63` : part des 20 points ti9ti35) | **0** (`i63` etait partiel : leve) | **0** (les corps 18, 19, 21, 23, `>= 24` sont ouverts) |
| **ti=40** vehicule | 48 ; **0/777** sur 5 bobines, **bloquant VIDE** | 16 (+2) | 16 | **0** — inchange ; `i43` = ti=35 `i60`, `i2` = ti=43 `i2` (ricochet) | 0 | `i2`, `i43` | **16 grammaires** ; **bloquant non etabli** (preuve bornante = un film) |
| **ti=42** arme au sol | 21 ; **10/2 087**, **bloquant VIDE** | 0 | — | — (negatif MESURE) | — | — | `[~]` instruction de cadre (D8 du 1.9.1 bis) |
| **ti=43** device | 41 ; **0/4 106**, bloquant `i19` | 22 (+1) | 22 | **23** — `i19`..`i29` (11, note A), `i2` (note B §13), **`i30`..`i35` (6, note C)** et **`i36`..`i40` (5, note D)** | 12 (`i19`, `i20`, `i21`, `i11`, `i12`, `i13` premiere passe ; `i30`, `i31`, `i32`, `i36`, `i37`, `i38` seconde passe) | `i2` (partiel de CONTEXTE : branche `DAT_145121140 == 1` non modelisee) | **0** (`i30`..`i40` releves) |

### 10.5 Totaux — mise a jour (remplace §3)

- **Population du lot 3.6** : **74 lignes** (66 non portees + 8 partielles) sur 246 composants —
  inchange.
- **Ecrivains nommes** : **74 / 74** — inchange.
- **Grammaires relevees a l'ecrivain** : **44 -> 56 / 74** = **54 completes + 2 partielles**
  (`ti=12 i18`, `ti=43 i2`). Par archetype : ti=9 **2**, ti=11 1, ti=12 26, ti=35 4,
  ti=43 **23**. Les neuf notes de grammaires du 17/09 portent **66 grammaires** : 53 (premiere
  passe, dont 10 recontroles) + 13 (seconde passe).
- **Verifiees par sceptique** : **16 -> 24 composants** ; **169 -> 209 points de controle**,
  **167 -> 206 concordants**, **2 -> 3 discordances**, toutes tranchees, **toutes les trois en
  faveur du verificateur**, notes corrigees en tete de section touchee.
- **Restent sans grammaire** : **30 -> 18 lignes**, **toutes `ti=40`** (16 ecrivains nommes +
  `i2` et `i43` couverts par ricochet) ; soit **16 a relever reellement** (28 avant).
- **Non elucide au sens strict** (ouvert dans le binaire et non ferme) : **2 entrees** (3
  avant) — le bloquant de `ti=40` (exige un film) et le remplisseur `FUN_140be9a14` de la table
  de largeurs par defaut. Les corps d'etiquette `>= 18` d'`i63` sont fermes.
- **Partiels restants** : **2** (`ti=12 i18`, `ti=43 i2`) au lieu de 3.

### 10.6 Ce qui reste non elucide — table §7 mise a jour

| Ou | Quoi | Statut apres seconde passe | Ce qui le leve / l'a leve |
|---|---|---|---|
| ti=35 `i63` | corps d'etiquette 17, 18, 19, 20, 21, 22, 23, 24, 25..31 | **LEVE** | `NOTE_3_6_TI9_TI35_RESTES_2026-09-17.md` §2.1..§2.10 : les 32 valeurs de `t = R(5)` ont un corps a largeurs LITTERALES (17 = 32/40, 18 = 61/69, 19 = 41/49, 20 = 15, 21 = 37/45, 22 = 1/9, 23 = 34/42, 24 = 35/43, 25..31 = 90/98) ; plus aucune branche « 0 bit » |
| ti=43 `i30`..`i40` | 11 grammaires (ecrivains nommes) | **LEVE** | notes `_GRAMMAIRES_C` (`i30`..`i35`, 6) et `_GRAMMAIRES_D` (`i36`..`i40`, 5) ; 6 composants verifies par sceptique, 0 discordant |
| ti=9 `i9` | boucle a etiquette de `FUN_141fcf160` | **LEVE** | `NOTE_3_6_TI9_TI35_RESTES_2026-09-17.md` §1 : descripteur `0x143d08930`, `R(1)`[1 -> `R(2)` + `R(1)`[1 -> `R(32)` + `n = R(3)` + `n x (R(3)` + corps)]] ; 5 corps a largeurs litterales ; **0 dependance de configuration** |
| ti=35 `i63`, `t = 17` | bornes du second `R(15)` (`XMM2`/`XMM3` non recharges) | **LEVE** | §10.3 : `FUN_1406d84b4` ne fait que LIRE `XMM2`/`XMM3` et ne contient aucun `CALL` — les bornes survivent a l'appel (question de valeur, 0 bit) |
| ti=40 | le bloquant (golden muet) ; 16 grammaires | **RESTE** | preuve bornante sur film (hors perimetre d'une preparation sans decodage), puis decompile |
| ti=12 `i18`, ti=35 position absolue | remplisseur `FUN_140be9a14` de la table de largeurs par defaut (`3 x R(22)` deduit de la loi, pas lu) | **RESTE** | decompile ; ou `DesyncAt` au port |
| ti=12 / ti=43 | site d'appel exact `+0x10 -> param_4` (calibration 4/4, pas de lecture du site) | **RESTE** (residu de preuve, pas de largeur) | `get_xrefs_to` sur les descripteurs, si le pilote l'exige |
| ti=12 `i13` / `i14` / `i15` | code 255 d'un `R(8)` en `f6 = 1` (hors plage) | **RESTE** (decision de port) | saturer ou invalider — a trancher au commit qui porte |
| ti=43 `i2` | partiel de CONTEXTE (branche `DAT_145121140 == 1` non modelisee ; doc et table en retard sur le code) | **RESTE** | re-statuer au lot qui porte les 22 `device-*` |
| ti=35 `i63`, `t = 26..31` | semantique (le jeu leur applique le corps de `t = 25` et range l'etiquette `0x19`) | **RESTE** (semantique, jamais une largeur) | rien a lever : le lecteur reproduit le comportement du jeu et peut signaler `t > 25` |
| ti=9 `i9`, corps `k = 1` | valeur rangee dans l'emplacement (tables runtime `FUN_14049746c`, `FUN_140e958c4`) | **RESTE** (semantique) | rien a lever : ces fonctions ne recoivent pas le flux ; le port publie l'index brut `R(5)` |

### 10.7 Ce que la seconde passe change pour le port

1. **`ti=43` se porte desormais en UN lot de 22** sans releve prealable : la ligne « apres
   releve de `i30`..`i40` » de l'ordre conseille (§6.5) tombe. Ordre interne inchange ; les
   notes C §9 et D §8 donnent les pieges, notamment que `i36` n'est PAS `i24` / `i29` (porte
   externe puis porte interne, categorie 0 sans sonde, `R(3)` de queue inconditionnel — un
   copier-coller desynchronise de 1 + 3 bits par entree).
2. **`ti=9` se ferme d'un seul lot** (`i4` + `i9`) : `dispatch_player.go:298` rend
   `ported = false` des que `n > 0`, c'est la seule ligne a remplacer.
3. **`ti=35 i63` n'a plus d'etiquette non ouverte** : supprimer toute branche « 0 bit »
   (`consumeBipedActionTag`, `default` « tag >= 6 -> 0 bit, verite EXE 2026-06-13 ») — le
   commentaire est de la doc inversee (anti-pattern 9) et se reecrit dans le meme commit.
4. **Aucune entree de profil nouvelle** sur les 13 grammaires de la seconde passe. Les deux
   seules dependances de configuration rencontrees sont deja modelisees : plage de la
   categorie 1 pour `ti=43 i31` et plage de la categorie 0 pour `ti=43 i36`
   (`varWidthBits`, `IDLowBits`, garde `DAT_144706104`) — **ne jamais figer 13**.
5. **Une seule fonction commune de dequantification** reste le prealable transverse (§4.3) :
   les groupes C et D ajoutent onze `Q(n; a..b)` en mode `b7 = 1` ; `dequantMidpoint`
   (`components_managed_object.go:198`) n'est toujours pas ce contrat.
6. **`ti=40` est le seul reste de releve** — et il n'est pas relevable sans film (preuve
   bornante). C'est la frontiere de ce qu'une preparation sans production peut donner.

### 10.8 Journal des appels Ghidra de l'addendum

Sept appels HTTP en lecture, **aucune ecriture**, aucun echec : 1 lecture du catalogue
`/mcp/schema` ; `list_open_programs` (un seul programme, `HaloInfinite.exe`, base
`0x140000000`, 311 103 fonctions) ; `read_memory address=0x143d08930 length=96` (les 10 slots
du descripteur `ti=9 i9`) ; `get_function_by_address` x 4 (`0x1404ab600`, `0x1411c8f80`,
`0x14049b600`, `0x141c8f880`) ; `disassemble_function 0x1406d84b4` (129 instructions, 3 919
octets, restees dans les bornes de la fonction).
