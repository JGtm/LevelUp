# LOT RE — la règle d'attribution de la lettre A/B/C d'une base (Ghidra, statique)

Date : 2026-08-24. Branche `wt/re-lettre-hud`, base `feat/v75` `b16ba17e5`.
Plan : `.ai/V7.5/replay2d/PLAN_RE_LETTRE_HUD_GHIDRA.md`. Exécution sous `plan-execution`.

Binaire : `HaloInfinite.exe` (D:/SteamLibrary/steamapps/common/Halo Infinite),
image base `0x140000000`, 311 103 fonctions, 1 450 918 symboles, projet Ghidra ouvert
(PID 32880). **Lecture seule** : uniquement des points de terminaison GET
(`search_strings`, `list_strings`, `get_xrefs_to`, `read_memory`, `decompile_function`,
`disassemble_function`, `search_functions`, `search_byte_patterns`, `search_instructions`).
Aucune analyse, aucun renommage, aucune écriture.

> **Écart d'outillage, consigné** : `mcp__ghidra__connect_instance` refuse la connexion —
> la découverte UDS rend l'instance avec un nom de projet `unknown`
> (`\tmp\ghidra-mcp-Guillaume\ghidra-32880.sock`, 0 octet) et le pont refuse alors tout
> `tcp_port`. Le serveur du greffon, lui, répond : `127.0.0.1:8089` (même PID),
> `/mcp/schema` rend 196 outils et `/get_current_program_info` rend `HaloInfinite.exe`.
> Le lot a donc piloté Ghidra par l'API HTTP du greffon, qui est le même serveur.
> À corriger côté pont (hors périmètre).

---

## 1. VERDICT

**(c) — négatif écrit, et LOCALISÉ.** La règle qui choisit la lettre d'une base
n'est pas dans le binaire : ce n'est pas « introuvable à coût raisonnable », c'est
**absent par construction**. Le moteur n'implémente aucune logique de Bastion ; il
expose une API de mode au SCRIPT, et c'est le script du mode qui pose le texte du
marqueur. Preuves en §3.

**Acquis de type (b), substantiel** : le CANAL qui porte la lettre est désigné,
décompilé et sa grammaire est complète — `ti=12 i9`
`managed-navpoint-formatted-text-component`, déserialiseur `FUN_1410E7B90` (§4).
Mais ce canal **n'est pas observable aujourd'hui dans le film** : il est au plancher
de bruit en delta (mesure du corpus, §5), donc il vit dans l'image-clé, dont la
lecture exige de porter d'abord `ti=12 i1..i8`. Le coût exact est chiffré en §6.

**Le repli « ordre des slots ti=13 » est ORTHOGONAL** — ni confirmé ni infirmé.
Justification en §7.

---

## 2. Phase 0 — points d'entrée (adresses concrètes)

### 0.2 Inventaire

| famille | ce qui a été trouvé | adresses |
|---|---|---|
| désignation de zone | `chud_variant_objective_designator`, et ses trois voisins `chud_variant_objective_variant` / `chud_variant_name` / `chud_variant_objective` | `143bf8d48`, `143bf8d10`, `143bf8d30`, `143bf8d70` |
| initialisation de ces identifiants | `FUN_140249c40` : `_DAT_144c297a0 = string_id("chud_variant_objective_designator")` ; voisins `FUN_140249c70`, `FUN_140249340`, `FUN_140249bb0` écrivant `144c29798`, `144c297a8`, `144c297b0` | `140249c40` |
| type d'objectif de Bastion | `objective_type_stronghold` (identifiant de type, init `FUN_140243190`) | `143bfdbc0` |
| marqueur / navpoint : composants réseau | les 28 composants `managed-navpoint-*` de `ti=12`, dont `managed-navpoint-formatted-text-component` | `143c94c48` (i9), `143c94cb8` (i0), `143c94e28` (i14) |
| marqueur / navpoint : API de script | 200+ liaisons `Navpoint_*`, dont **`Navpoint_SetDisplayText`** et **`Navpoint_SetDisplayFormattedText`**, enregistrées par `FUN_140dafe54` | `143c5c318`, `143c5c2f0` |
| objectifs : API de script | 80+ liaisons `Objective_*` (`Objective_SetFormattedText`, `Objective_SetProgress`, …) et `Engine_CreateObjective` | `143c56e10`, `143c5aea8` |
| table des objets de mode | `managed-objective-*` (20 composants) et `managed-engine-objective` | `143c952e8` |
| propriétés réseau du mode | `ManagedNetworkedPropertyDefinition_SetName`, `ManagedNetworkedProperty_{Get,Set}StringIdProperty`, `ManagedGameVariant_{Get,Set}{Winning,Losing,Tie}RoundReasonNetworkedPropertyName`, `…GameStateNetworkedPropertyName` | `142c90e2c`, `142c90e74`, `142c94bcc`, `142c9372c`, `142c93e34`, `142c933cc` |
| résolution chaîne → identifiant | `ManagedGameEngine_ResolveStringId` → `FUN_1408c1714` → `FUN_1411b2334` → **`FUN_140748a74`** | `142c90f2c`, `1408c1714`, `1411b2334`, `140748a74` |

### 0.3 Pistes retenues (annoncées, pas soumises à accord)

1. **`managed-navpoint-formatted-text-component`** — le texte du marqueur d'objectif,
   canal réseau ; c'est là que la lettre passe si elle passe quelque part.
2. **`FUN_140748a74`** — la fonction `string_id`, pour pouvoir RÉSOUDRE tout identifiant
   de chaîne lu dans le film (et vérifier la table de labels du dépôt).
3. `chud_variant_objective_designator` — abandonnée après §3.4 : c'est un identifiant de
   source de données du HUD, écrit une fois à l'initialisation, sans lecteur résolu.

**Gate 0 : TENU** (connexion établie ; toutes les adresses ci-dessus sont concrètes).

---

## 3. Phase 1.1 — ce que le binaire dit de la source de l'index

### 3.1 La fonction `string_id` du jeu, sur pièces

`ManagedGameEngine_ResolveStringId` (`142c90f2c`) est un stub d'API de script :
il appelle `FUN_1408c1714(out, s)` qui rend `0xFFFFFFFF` si `s == NULL`, sinon
`FUN_1411b2334(s)`. Celui-ci (désassemblé, 7 instructions) charge une constante de
16 octets depuis `0x143d13a90` et appelle `FUN_140748a74(s, &ctx)`.

`FUN_140748a74` est **MurmurHash3_x86_32, graine 0**, précédé d'une NORMALISATION :

```
'A'..'Z' -> minuscule      '-' -> '_'      ' ' -> '_'      '\n' -> '#'
```

- chaîne nulle → `0xFFFFFFFF` ; chaîne vide → `0` ;
- ≥ 64 caractères : passe par un tampon dynamique (`FUN_140748f60` / `FUN_140748fb4`) ;
- une branche alternative existe (`if (*ctx == 9) FUN_141c1034c(...)`) : **elle n'est pas
  prise ici**, `read_memory(0x143d13a90)` rend `00 00 00 00 0f 00 00 00 …`, donc `ctx[0] == 0`.

Constantes lues dans le décompilé : `0xCC9E2D51`, `0x1B873593`, rotations 15/13,
`h*5 + 0xE6546B64` (le compilateur l'a écrit `(h_rotl13 + 0xFADDAF14) * 5`, identique
modulo 2^32), finaliseur `0x85EBCA6B` / `0xC2B2AE35`.

**Contrôle** : l'implémentation rejouée rend `murmur3("hello") = 0x248BFA47` (vecteur
public) et retombe sur **5 entrées sur 5** de `mapvar.labelNames` du dépôt
(`stockpile_socket` `2110778921`, `strongholds_zone` `412386272`, `flag_delivery`
`-713178115`, `ctf_include` `-2087265038`, `extraction_zone` `1384999457`).

> Conséquence : `apps/go-api/internal/analysis/replay/mapvar/hash.go` est **la même
> fonction, moins la normalisation**. Voir §8, découverte n°1.

### 3.2 Le moteur ne contient aucune logique de Bastion

- `search_functions("Stronghold")` → **0 fonction**. Idem `Hill`, `CaptureZone`,
  `ObjectiveDesignator`.
- Les seules chaînes `Stronghold*` sont de la télémétrie Bond (`StrongholdsStats_*`,
  `StrongholdBasesControlled`) et un identifiant de type d'objectif
  (`objective_type_stronghold`, `143bfdbc0`).
- Ce qui existe est une **API exposée au script** : `Objective_*` (80+),
  `Navpoint_*` (200+), `Engine_CreateObjective`, `Managed*` (938 symboles).
  Les messages d'erreur associés sont ceux d'un interpréteur
  (`"attempt to use missing objective %d"`, `"squad or objective expected"`,
  `"%s first parameter is not an objective index"`, `143df5f98`, `143dd6948`,
  `143dd6860`), et le binaire embarque `temp_objective_fragments.lua` (`14369ba28`).

**Conclusion 1.1 : l'index 0/1/2 n'est choisi nulle part dans l'exécutable.** Il est
choisi par le script du mode (donnée de tag), qui pose ensuite un texte sur le marqueur.

### 3.3 Par où la lettre sort du script

Deux liaisons, et deux seulement, posent un texte sur un marqueur d'objectif :

| liaison de script | chaîne | enregistrée par |
|---|---|---|
| `Navpoint_SetDisplayText` | `143c5c318` | `FUN_140dafe54` (site `140db0cfc`) |
| `Navpoint_SetDisplayFormattedText` | `143c5c2f0` | `FUN_140dafe54` (sites `140db0da9`, `140db0db0`, `140db0ddb`, `140db0dfb`) |

Leur état est répliqué aux clients — donc au film — par le composant
`managed-navpoint-formatted-text-component` (`ti=12 i9`).

### 3.4 Pistes fermées, et pourquoi

- **`chud_variant_objective_designator`** (`143bf8d48`) : `FUN_140249c40` calcule son
  `string_id` (`0x97C352FA`) et l'écrit dans `_DAT_144c297a0`. Ce global n'a **qu'un xref
  en ÉCRITURE**, et `search_instructions` sur l'immédiat `0x97c352fa` rend **0 occurrence
  sur 13 607 523 instructions balayées**. C'est un nom de source de données du HUD, pas
  un calcul de lettre.
- **Aucune lettre littérale dans l'exécutable** : aucune chaîne dont la valeur soit
  `"A"`, `"B"`, `"C"` ou `"Zone A"` (84 264 chaînes inventoriées). Les libellés vivent
  dans les tags de localisation.
- **Les identifiants de chaîne observés dans le film ne sont pas dans l'exécutable** :
  `search_byte_patterns` sur les 5 valeurs `tag 5` de `ti=13`
  (`0x67F43AC3`, `0xD690D6B4`, `0xF2F9EB27`, `0x6050ABD7`, `0x3327C7DA`) rend
  **0 occurrence** ; et **aucune des 84 264 chaînes** de l'exécutable ne les hache
  (passe complète avec la normalisation du jeu). Ces noms sont définis par le script.

---

## 4. Le canal désigné, décompilé (`ti=12 i9`)

### 4.1 La recette d'adressage, revérifiée

Le déserialiseur d'un composant est `*(fente_du_getter_de_nom + 0x28)` :

| composant | chaîne | getter | fente vtable | déserialiseur |
|---|---|---|---|---|
| i0 `sub-type` | `143c94cb8` | `141177df0` | `143d07eb0` | `1410E0CAC` ✅ (= `FUN_1410e0cac` du dépôt) |
| i14 `radial-progress` | `143c94e28` | `141177d10` | `143d08128` | `140FC8D14` ✅ (= `FUN_140fc8d14` du dépôt) |

Les deux témoins retombent sur les adresses que la table ECS du dépôt donne déjà :
la recette est validée avant d'être appliquée aux composants inconnus.

### 4.2 `FUN_1410E7B90` — grammaire complète de `ti=12 i9`

Fente `143d08538` (getter `141177de0`), déserialiseur en `143d08560` = `0x1410E7B90`.

```
R(8)  n                          ; nombre d'entrées, écrit en objet+0x580
n fois, pas de 0x2c octets :
   R(32) textStringId            ; FUN_14080dec4(rdr,"textStringId",e+0x00)
   R(1)  présent                 ; FUN_14080b034 -> FUN_1406cf008
   si présent :
      R(32) text                 ; FUN_14080dec4(rdr,"text",...) — DAT_1436f4b68 = "text"
      R(3)  nbArgs               ; 0..7, seuls 4 arguments sont stockés
      nbArgs fois : argument     ; FUN_1407f0ebc
   si absent : textStringId := 0xFFFFFFFF
```

`FUN_14080dec4(reader, nom, dest)` est **le même lecteur `R(32)` string-id** que celui
du dépôt pour `ti=13 i0` (`FUN_142ed69d8 = FUN_14080dec4(rdr,"property-name",…)`).

**Donc la lettre affichée est, dans le film, un `string_id` de 32 bits** — résoluble par
`FUN_140748a74` (§3.1) dès qu'on sait le lire.

### 4.3 Les autres composants de `ti=12` (matière pour le portage)

| i | niveau | composant | déserialiseur | grammaire |
|---|---|---|---|---|
| 0 | 0 | `sub-type` | `1410E0CAC` | `R(32)` — **déjà porté** |
| 1 | 1 | `flags` | `141094130` → `14109414c` | **`R(8)`** |
| 2 | 1 | `visibility-distance-filters` | `140DBDE1C` → `FUN_140dbde44(dst, rdr, 0, lvl > 2)` | **`LF(lvl>2)`** + 2 x `FUN_1411b4e6c` + par bit du masque : `FUN_140dbdf00()` [+ `R(4)` si le drapeau est faux] |
| 3 | 3 | `visible-offscreen-filters` | `140DBDF80` → `FUN_140dbdfd8(dst, rdr, ctx, lvl > 1)` | **`LF(lvl>1)`** + `R(1)` + par bit du masque : `R(1)` [+ `R(4)` si le drapeau est faux] |
| 4 | 2 | `can-be-occluded-filters` | `140DBDFAC` → `FUN_140dbdfd8(…, lvl > 1)` | idem i3 |
| 5 | 2 | `visibility-filter` | `140DBE194` → `FUN_140dbe400(dst, rdr, lvl > 1)` | **`LF(lvl>1)`** seul |
| 6 | 2 | `docking-filter` | `140DBDF34` → `FUN_140dbe400(dst, rdr, lvl > 1)` | **`LF(lvl>1)`** seul |
| 7 | 2 | `docking-order` | `142ED5050` | **`R(8)`** |
| 8 | 1 | `docking-group-name` | `142ED5028` | **`R(32)`** string-id (`"navpoint-docking-group-name"`) |
| 9 | 1 | `formatted-text` | `1410E7B90` | §4.2 |
| 14 | 1 | `radial-progress` | `140FC8D14` | `R(8)` — **déjà porté** |

`LF(flag)` = **la liste de filtres, lecteur UNIQUE partagé par i2..i6** —
`FUN_140dbe400(dst, rdr, flag)` :

```
R(4)  masque            ; FUN_140dbe598 -> jusqu'a 4 fentes de filtre, ecrit en dst+0x100
si flag : R(1)          ; sinon R(32)        ; ecrit en dst+0x104
pour k = 0..3, si le bit k du masque est mis :
    R(4)  tag           ; type du filtre
    variant FUN_141e98e10 selon le tag
```

`FUN_141e98e10` est **un variant à ~12 alternatives** (12 cibles de saut internes +
la table `PTR_FUN_143c4ebe0`) : c'est exactement la forme, et l'ordre de grandeur
d'effort, du variant de `ti=13` qui a occupé une phase entière du lot C-bis.
**Le porter UNE fois débloque i2, i3, i4, i5 ET i6.**

Les niveaux de la colonne 2 sont ceux du registre du film, tels que
`registre_film/lotC/7344d24f_delta_masques.tsv` les publie : les drapeaux
`lvl > 1` / `lvl > 2` sont donc **déterminés d'avance** pour cet archétype
(i2 : faux ; i3, i4, i5, i6 : vrai).

---

## 5. Phase 1.2 — le canal est-il OBSERVABLE dans le film ?

**Non, pas aujourd'hui.** Mesure existante du corpus, non re-jouée (item C.0.1 du lot C,
`registre_film/lotC/7344d24f_delta_masques.tsv` et `696a9d7c_delta_masques.tsv`) :

| film | `ti=12 i9` annonces en delta | plancher de bruit | excès | témoin qui parle : `ti=12 i14` |
|---|---|---|---|---|
| `7344d24f` (Bastion / Vagabond) | 206 | 119,5 | **1,7x** | 16 788 = 140,5x |
| `696a9d7c` (Bastion / Vagabond) | 25 | 20,0 | **1,2x** | 17 369 = 868,5x |

1,2 à 1,7x le plancher, c'est le niveau de la contamination d'ancrage : **le texte du
marqueur n'est pas ré-émis en delta**. C'est cohérent avec sa nature — il est posé UNE FOIS, à la
création du marqueur, comme `ti=13 i0 property-name` (négatif delta déjà établi au lot
C-bis, §CB.0.3). Le texte vit donc dans l'**image-clé**.

Lire une image-clé de `ti=12` passe par `WalkKeyframeFullState`
(`filmdec/keyframe_fullstate_loop.go`) : **aucun masque de présence, les 64 entrées de
l'archétype sont désérialisées dans l'ordre**. Atteindre i9 impose donc de porter
i1..i8 — soit, d'après §4.3 : trois largeurs triviales (i1 `R(8)`, i7 `R(8)`, i8 `R(32)`,
toutes établies par ce lot) et **cinq listes de filtres** (i2..i6) qui se ramènent à
**un seul lecteur partagé** `FUN_140dbe400`, dont le seul morceau non résolu est un
variant à ~12 alternatives (`FUN_141e98e10`).

---

## 6. Phase 1.3 — confrontation : NON JOUÉE, et pourquoi

Le plan la conditionne à « si un canal film est désigné ». Un canal EST désigné, mais
il n'est pas atteignable dans le budget de ce lot : la confrontation supposerait de
porter cinq déserialiseurs de listes de filtres avant de pouvoir lire un seul octet
de i9. Ce n'est pas un contournement — c'est le coût, chiffré, du lot suivant :

**Marche à suivre proposée (lot suivant, RE + portage) :**

1. **Fait par ce lot** — les cinq déserialiseurs de filtres sont décompilés et
   convergent vers UN lecteur partagé, `FUN_140dbe400` (§4.3). Ce qui reste à faire
   côté RE est **un seul objet** : le variant `FUN_141e98e10` (~12 alternatives, table
   `PTR_FUN_143c4ebe0`) — même forme et même ordre de grandeur que le variant de `ti=13`
   du lot C-bis phase 0. Le porter une fois débloque i2..i6.
2. Porter i1 `R(8)`, i7 `R(8)`, i8 `R(32)` (triviaux, déjà établis ici) puis la liste de
   filtres, avec les vecteurs figés d'usage (`zone_vectors_test.go`) et le témoin de
   chaînage (`Chained`).
3. Instrument sous garde d'environnement, un film par processus : image-clé des
   films `7344d24f` et `696a9d7c`, relever `textStringId` par entité `ti=12`.
4. Résoudre les identifiants par `FUN_140748a74` (implémentation §3.1 ; le dépôt a déjà
   `mapvar.LabelHash`, il lui manque la normalisation) contre un dictionnaire de
   candidats — et, si la résolution échoue, se contenter de l'ORDRE : trois identifiants
   distincts stables entre les deux films suffisent à ordonner les zones sans les nommer.
5. Croiser avec la carte `slot ti=13 -> zone du catalogue` déjà établie et mesurée
   (`LOTCBIS_PHASE2A.md` : 93,1 % / 98,4 %) — c'est ce croisement, et lui seul, qui
   tranchera le repli.

**Si l'étape 4 ne résout aucun nom** (probable : les noms sont définis par le script de
mode et absents de l'exécutable, §3.4), la lettre restera une CONVENTION : le film donnera
un ordre stable, pas un alphabet. Le seul chemin restant serait alors dynamique
(observer le HUD en jeu), donc hors du cadre statique de ce lot.

---

## 7. Le repli « ordre des slots ti=13 » : ORTHOGONAL

- **Ce que le binaire dit** : un slot `ti=13` est une **propriété réseau nommée** du mode
  (`ManagedNetworkedPropertyDefinition_SetName`), créée dans l'ordre où le script la
  déclare. Rien, dans l'exécutable, ne relie cet ordre de déclaration à la lettre
  affichée : la lettre est un texte posé sur un NAVPOINT (`ti=12`), pas une propriété.
  Les deux objets n'ont aucun lien de construction dans le code.
- **Ce que le corpus dit déjà** : la table mesurée en phase 2a range les trois jauges de
  Bastion en `1532 -> rang spatial 1`, `1537 -> rang 2`, `1542 -> rang 0`
  (`LOTCBIS_PHASE2A.md` §3). L'ordre des slots n'est donc **pas** l'ordre spatial : si le
  repli publie « A au premier slot », il publie un ordre de déclaration, pas une
  géographie.
- **Ce qui reste vrai, et qui suffit au repli** : cet ordre est **stable** (3 slots sur 3
  identiques entre les deux Bastion, 100 %). Le repli est donc reproductible et
  auto-cohérent — il n'est simplement adossé à rien qui prouve qu'il coïncide avec les
  lettres du jeu.

**Verdict demandé par le plan : ni CONFIRMÉ ni INFIRMÉ — ORTHOGONAL.** Ce lot n'apporte
aucune permutation corrigée ; il apporte le chemin qui permettra de trancher (§6).

---

## 8. Découvertes (consignées, NON traitées)

1. **`mapvar.LabelHash` est la bonne fonction, amputée de sa normalisation.**
   `apps/go-api/internal/analysis/replay/mapvar/hash.go` implémente murmur3 x86_32 seed 0
   sur les octets bruts. Le jeu (`FUN_140748a74`) normalise d'abord : minuscules,
   `'-'`/`' '` → `'_'`, `'\n'` → `'#'`. Sans conséquence sur les labels connus (tous en
   `snake_case` minuscule), mais **la force brute du lot C-ter volet 2 a cherché des
   « variantes de casse »** pour les trois labels de KOTH : ces variantes retombent toutes
   sur le MÊME hash côté jeu, l'espace fouillé était donc plus petit qu'annoncé.
   Ré-ouvrir la chasse avec la normalisation (et des noms à espaces / tirets) est bon marché.
2. **Le « trio de tag 5 » de `ti=13` a un candidat nommé dans le binaire.** Le moteur
   expose exactement TROIS propriétés réseau de raison de fin de manche —
   `ManagedGameVariant_Set{Winning,Losing,Tie}RoundReasonNetworkedPropertyName`
   (`142c94bcc`, `142c9372c`, `142c93e34`) — plus une de `GameState` (`142c933cc`).
   Cela colle exactement à l'observation du lot C-ter volet 1 (trois slots CONSÉCUTIFS,
   une émission chacun, 15 à 21 ms APRÈS la capture terminale, sur 4 films KOTH, avec un
   identifiant commun aux modes `0xF2F9EB27` et deux qui varient), et au cas à 2 slots de
   `0a247154` (pas d'égalité → pas de raison « tie »). **Hypothèse, non mesurée ici** ;
   elle confirmerait la découverte n°1 du volet 1 (« ce n'est pas un nommage de zones »)
   en lui donnant un nom.
3. **Les 26 valeurs `ti=13 i0` publiées en phase 2a ne sont pas des string-ids propres.**
   `7344d24f_p2a.tsv` contient `0x0003010B`, `0x00030081`, `0x000000D2`, `0x0000E808`,
   `0x000CC322`, `0x007E8103` … Un murmur3 est uniformément distribué : six valeurs à
   fort préfixe nul sur 26 est incompatible avec un hachage. Cohérent avec le taux de
   chaînage de 1-3 % déjà mesuré pour i0 en delta (contamination d'ancrage) — **les noms
   de propriété publiés en delta ne sont pas exploitables**, seule l'image-clé les porte.
4. **Le pont MCP Ghidra ne sait pas nommer l'instance** (voir l'encadré en tête) :
   `list_instances` rend `project: unknown`, `connect_instance` refuse. Contourné par
   l'API HTTP du greffon sur `127.0.0.1:8089`.

---

## 9. Statut des items du plan

| item | statut | note |
|---|---|---|
| 0.1 connexion + programme chargé | `[x]` | `HaloInfinite.exe`, 311 103 fonctions ; écart d'outillage consigné |
| 0.2 inventaire des points d'entrée | `[x]` | §2, table d'adresses |
| 0.3 pistes retenues | `[x]` | §2, trois pistes, une fermée en §3.4 |
| **Gate 0** | **TENU** | |
| 1.1 remonter la source de l'index | `[x]` | §3 — l'index n'est PAS dans le binaire ; mécanisme établi |
| 1.2 la source est-elle observable dans le film | `[x]` | §4 (canal + grammaire) et §5 (non observable en delta, mesure du corpus) |
| 1.3 confrontation sur le corpus | `[!]` | §6 — canal désigné mais inatteignable sans porter `ti=12 i1..i8` ; coût chiffré, marche à suivre écrite. Aucun périmètre adapté en douce. |
| **Gate 1** | **TENU** — verdict (c) écrit, chaque affirmation adossée à une adresse décompilée ou à une mesure rejouable ; repli statué ORTHOGONAL | |
