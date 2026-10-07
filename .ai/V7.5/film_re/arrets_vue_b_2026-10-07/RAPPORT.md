# RAPPORT — Composants où la marche depuis la fin de la vue A bute (lot arrêts vue B, 2026-10-07)

> Exécution du handoff `.ai/HANDOFF_COMPOSANTS_BLOQUANTS_VUE_B_2026-10-06.md` sous le contrat
> `plan-execution` (plan : `PLAN.md` de ce dossier). GO de l'utilisateur le 2026-10-07.
> Worktree `LevelUp-wt-grammaire-arrets-vue-b`, branche `feat/grammaire-arrets-vue-b`, base
> `feat/v75` = `879f31bbf`. Aucune fusion, aucune cuisson du parc.
>
> Conventions : **lu** = lu dans le jeu (Ghidra, `HaloInfinite.exe` HI_1_13_0, lecture seule, HTTP
> direct `127.0.0.1:8089`) ; **mesuré** = compté par un outil sur les films ; « sain » = paquet fermé
> que le juge de L0 ne contredit pas (colonne `ferme` de la carte v2).

## 0. Statut

**Les cinq composants sont retenus, chacun avec son commit et son gate 2 tenu sans exception.**
Le lot entier, contre `879f31bbf` : **+22 282 paquets sains (452 139 -> 474 421), +216 058 records
utiles sains (5 702 879 -> 5 918 937), 0 paquet sain perdu, aucun film en baisse.**

| # | Composant | Statut | Commit | Rév. | Sains gagnés | Utiles sains gagnés | Perdus |
|---|---|---|---|---|---|---|---|
| C1 | `ti=43` `i18`..`i40` (reprise de L2) | [x] retenu | `ec9897101` | `grammar-2026-10-07` | +19 797 | +160 130 | 0 |
| C2 | `ti=12 i16 managed-navpoint-override-flags` | [x] retenu | `05ab869d0` | `.2` | +110 | +2 375 | 0 |
| C3 | `ti=45 i0 matchflow-sequence-data-component` | [x] retenu | `0962d0970` | `.3` | +861 | +15 972 | 0 |
| C4 | `ti=10 i2` (et `i3` à `i17`, même nom) `managed-object-navpoint-component` | [x] retenu | `e480f6dbb` | `.4` | +1 481 | +37 224 | 0 |
| C5 | `ti=12 i18 managed-navpoint-position-offset` | [x] retenu | `05b70670a` | `.5` | +33 | +357 | 0 |
| D | point (d) du §2.2 | [~] non éclairé par ce lot (§3) | — | — | — | — | — |

Chaque ligne est mesurée par une carte v2 (20 films) de l'état retenu précédent contre la tête du
composant (`tsv/gate2_c<N>_contre_*.tsv`), et le lot entier par `tsv/gate2_lot_contre_base.tsv`.

## 1. Par composant

### C1 — `ti=43` `i18` à `i40` (dispositifs de carte), reprise de L2

- **Lu** : le port de L2 (`b12eb7692`, 23 lecteurs relus par le contrôle du 2026-10-03,
  `LOT_L2.md` §1) ; relu ce jour : `i19` `FUN_1410156e4` (`R(32)` par `FUN_141015740` puis
  `FUN_1406d84b4` à `0xa` = `Q(10)`), `i21` `FUN_1407f0678` (`R(32)` par `FUN_1406d676c(n=0x20)` puis
  `FUN_1407f08bc` : `R(1)` porte, `FUN_1407f08f8` = `R(8)`). Largeurs : table de
  `components_device_ti43.go` (15 fixes, 8 gardées par le flux : `i21`, `i24`, `i29`, `i31`, `i34` à
  `i37`).
- **Portage** sur la tête du moment : le maillon `consumeComposantsDispositif` s'insère entre M4b et
  `ti=40` (la chaîne a changé depuis L2) ; `i37` passe par le lecteur de minuteur unique
  `lireMinuteur142ba78dc` (n = 10), arrivé depuis L2 (le `consume140d580d0` de L2 n'est donc pas
  repris, son garde-rail non plus : `lecteur_minuteur_guard_test.go` le couvre, étendu à
  `largeurMinuteurDistributeur`) ; la règle du masque de `pasDEssai` était déjà en tête (lot LT).
- **Le mécanisme qui avait fait retirer L2 ne se reproduit pas** : `1c4c63c2` +13 sains, 0 perdu
  (L2 : −447 dans C11 par un faux NEW `ti=43` lié au monde par le second rang de
  `debutParFermeture`, que LR et V2 ont depuis changé).
- Carte v2 contre la base : **+19 797 sains / +160 130 utiles, 0 perdu**. `bcb6d393` (HI_1_12_0)
  6 479 -> 18 985, `396cfc92` +4 729, `f75e7053` +1 704. Gagnés par cause d'arrêt en base :
  `ti=43 i35` 11 351, sortie par rejet 7 413 (le NEW `ti=43` non traversé laissait son slot inconnu),
  `ti=43 i21` 618, `i19` 182, liste non localisée 137, `i39` 82 (le premier paquet de chaque film,
  1:10), `i34` 10, `i22` 3.
- Reste en `ti=43` après C1 : `i31` au-delà de huit moniteurs (11 paquets, arrêt voulu, comme le
  jeu), « archétype hors registre » (28).

### C2 — `ti=12 i16 managed-navpoint-override-flags`

- **Lu** : nom (`143c94de8`) -> accesseur de nom `141177cf0` -> descripteur `143d08088` ; le slot
  qui suit le thunk `FUN_14076ce9c` donne le lecteur `FUN_140ebf834` -> `FUN_140ebf854` : `R(5)`
  (`ADD [flux+0x2c],5`, `SHR R9,0x3b`) vers `etat+0x70c`. Écrivain (`descripteur+0x10` = `142edb0a0`,
  saut vers `142ed0e2c`) : `MOV ECX,5`, les cinq bits du même mot. Fixe, sans porte.
- Port : `components_navpoint_suite.go` (`consumeNavpointOverrideFlags`), `case` dans
  `consumeNavpointComponent`.
- Carte v2 contre C1 : **+110 sains / +2 375 utiles, 0 perdu**. Gagnés : 64 arrêtés sur `i16`,
  29 rejets, 14 non localisées, 3 masques. Les autres records `ti=12` qui s'arrêtaient sur `i16`
  avancent et s'arrêtent sur `i18` (63 -> 318) et `i17` (92).

### C3 — `ti=45 i0 matchflow-sequence-data-component`

- **Lu** : accesseur de nom `141177bc0` -> descripteur `143d07be0` -> lecteur `FUN_14101cdd8` :
  `FUN_14101d200` = `R(4)` rendant valeur − 1 (`ADD [flux+0x2c],4`, `SHR R9,0x3c`, `LEA EAX,[R9-1]`)
  vers `etat+0`, puis quatre `R(32)` (`ADD [flux+0x2c],EBP`, `EBP = 0x20`) vers `+0xc`, `+0x10`,
  `+0x4`, `+0x8`. Écrivain `FUN_142edbf94` : `FUN_1407ebac4` (l'octet `+0` plus 1 sur quatre bits),
  puis les quatre mots dans le même ordre. 132 bits fixes.
- Port : `components_matchflow_ti45.go`, `case` dans `consumeMoteurDePartie`.
- Carte v2 contre C2 : **+861 sains / +15 972 utiles, 0 perdu** ; `1c4c63c2` +779 (le slot que le
  NEW `ti=45` occupait n'était pas lié : ses deltas sortaient par rejet), `11de8353` +19.
- **Perte (a) du §2.2 regagnée** (mini-bobine `000d5950`, paquet 2:712) : la marche passe le slot
  122 (`ti=45`) et lit les huit annonces d'emplacement vide (`i45`, Restated) des bipèdes 512 à
  519 : `heldWeaponChanges` 6 -> 14, records bipèdes 29 511 -> 29 519. Les goldens
  (`golden_minibobine_familles.tsv`, `comptesDeRecordsMiniBobine`) sont mis à jour dans le commit
  (changement de décodage déclaré) ; la différence a été instruite record par record par une
  sonde en surcouche (`tsv/sonde_held_test.go.txt`, jamais dans le code).
- Arrêt suivant dans cet archétype : `ti=45 i1 matchflow-focus-data-component` (16 paquets).

### C4 — `ti=10 i2` à `i17` `managed-object-navpoint-component`

- **Lu** : le registre de `ti=10` pose SEIZE descripteurs sous ce nom (`i2` à `i17` dans
  `ecs_table.tsv`) ; une seule table (accesseur de nom `14064c7d0`, table `143c971d8`), un seul
  lecteur `FUN_14107cea4` : `R(32)` vers `etat + 0x14 + 4 * index`, `index = *(descripteur + 8)`.
  Écrivain `142edb304` : `MOVSXD RCX,[RCX+8]`, `MOV R9D,[RAX+RCX*4+0x14]`, `MOV ECX,0x20` : les 32 bits
  du même mot. Fixe, sans porte. Les seize lignes sont portées (G4 157 -> 173).
- Port : `consumeManagedObjectNavpoint` (`components_managed_object.go`), `case` dans le maillon
  M4b : `consumeManagedAndObjectiveComponent`, maillon naturel de `ti=10`, est au plafond du ratchet
  de longueur (`archlint/film_function_length_test.go`, 143 lignes gelées).
- Carte v2 contre C3 : **+1 481 sains / +37 224 utiles, 0 perdu** ; `1c4c63c2` +796, `111fa685`
  +248, `e5adf7b2` +165, `4f77afc1` +103. Gagnés : 890 rejets, 313 arrêtés sur `i2`, 235 non
  localisées, 8 sur `i3`, 2 sur `i4`.
- Arrêts suivants : `ti=10 i22 managed-object-interaction-filter-component` (291), `ti=10 i18`
  à `i21` (40).

### C5 — `ti=12 i18 managed-navpoint-position-offset`

- **Lu** : accesseur de nom `141177cd0` -> descripteur `143d08268` -> lecteur `FUN_140f04f68` : garde
  de pleine précision `FUN_14076f91c` (aucun bit lu) ; sous la garde `FUN_1411b259c` = `R(96)`
  brut ; sinon `FUN_14076e524` au niveau `0x10` (`MOV R9D,0x10` en `140f04f80`, `CALL 140f04f8b`),
  vers `etat+0x714`. C'est la forme de `FUN_14076e494` : port par `lireE494` (le portage unique de
  la position). Écrivain `142edb0b4` -> `141f860b0` -> `FUN_1407eb61c(.., -1, 0x10, 0)`. Largeur
  variable (porte d'index, largeurs de la carte) : `bits_typ = variable`.
- Le site est ajouté à la table du ratchet des positions (`lecteur_position_ratchet_test.go`) et à
  ses flux d'écrivain (`lecteur_position_sites_test.go`, table défaut, table par index, carte à
  quatre plages) ; vecteur sous la garde (`TestLeDecalageDuMarqueurLitSousLaGarde`).
- Carte v2 contre C4 : **+33 sains / +357 utiles, 0 perdu**.
- Les records `ti=12` vont maintenant jusqu'aux `visual-state-groups` : `i21` 675, `i22` 532,
  `i20` 433, puis `i17 object-marker` 92 (§4).

### Colonne `cause` dominante (carte v2, paquets non sains)

| Cause | Base `879f31bbf` | Tête `05b70670a` |
|---|---|---|
| vue B : sortie par rejet | 116 736 | 108 771 |
| liste d'événements non localisée | 24 222 | 23 402 |
| `ti=43 i35` | 12 201 | 0 |
| vue C : terminateur hors cadre | 11 524 | 11 625 |
| vue C : kind non porté | 2 795 | 2 847 |
| écrivain : masque au-delà de l'archétype | 2 069 | 2 287 |
| vue C : bloc `0xbc` | 981 | 991 |
| `ti=43 i21` | 886 | 0 |
| `ti=10 i2` | 641 | 0 |
| `ti=12 i21` / `i22` / `i20` | 551 / 494 / 133 | 675 / 532 / 433 |
| `ti=12 i16` | 454 | 0 |
| `ti=10 i22` | 80 | 291 |
| `ti=45 i0` | 68 | 0 |
| `ti=12 i18` | — (63 après C1) | 0 |

Le composant d'arrêt dominant est désormais la famille `ti=12 visual-state-groups` (1 640 paquets).
Tables complètes : `tsv/causes_base.tsv`, `tsv/causes_c1.tsv` à `tsv/causes_c5.tsv`.

## 2. Gates du lot (§5 du handoff)

| Gate | Sortie |
|---|---|
| Gate 2, carte v2 base `879f31bbf` contre tête `05b70670a` | **+22 282 sains, +216 058 utiles sains, 0 perdu, 0 film en baisse** (`tsv/gate2_lot_contre_base.tsv`) ; chaque composant aussi contre l'état précédent, 0 perdu à chaque pas |
| Gate 3, `killsource json` (20 films du kit, 19 témoins + `1c4c63c2`) | 17 identiques à l'octet ; `1c4c63c2`, `4f77afc1`, `c75f33b8` ne diffèrent QUE par la ligne de diagnostic `calibration` (score et médiane de l'oracle du profil plat : 406 -> 408, 403 -> 404, 74 -> 75 et 335 -> 339), décision de calibration (largeurs, `indexW`) inchangée ; aucune mort, voie, valeur, compte de santé ni ligne publiée ne change (`tsv/killsource_diff.txt`). La chaîne n'est pas persistée (`cmd/killsource/sortie_json.go`) : **`killsource.Rev` reste `killsource-2026-09-27`** (D23), empreinte régénérée à révision constante |
| `TestGoldenFilms` (`KILLSOURCE_FIXTURES`) | `ok` : 4 / 4 films (`000d5950`, `9b191a7f`, `78919882`, `fccc61cd`), aucun saut |
| Gate de corpus `replay-corpus-gate --reference=base --base=879f31bbf --keep-work`, parc copié au scratchpad (base partagée identique à l'octet, chunks, manifestes et artefacts des 19 témoins) | **rc = 1** (`codePerte`, 19 cuits, aucune erreur) ; banc de vérité **18 / 19 « ok », 0 `MANQUE`, 1 `FAUX`** (§2.1) ; statut PERTE sur 16 témoins, « ok » sur `bfecd02b` et `4f77afc1` (§2.2) ; télémétrie `grammarRev` `.6` -> `2026-10-07.5` (19). Pièces : `tsv/gate_corpus_verdict.txt`, `tsv/gate_corpus.json` |
| gofmt | vide (`film/`, `cmd/`) |
| vet, vet `-tags=research`, vet `-tags=integration` (film) | rc 0, rc 0, rc 0 |
| archlint | `ok` (79 s) |
| golangci-lint `--new-from-rev=879f31bbf ./internal/games/halo_infinite/film/...` | `0 issues.` |
| G-film (`film/...`, `replaybuild`, `killcollector`, `-count=1`) | 21 paquets `ok`, rc 0 (rejoué après C3, C4, C5) |
| Mutations (overlay, `tsv/mutations.sh`) | **28 / 28 ROUGES** sur la tête finale (`tsv/mutations_final.txt`) : C1 18, C2 2, C3 3, C4 2, C5 3 |
| Baseline des tests | aucun `func Test` retiré ni renommé (3 ajoutés) : rien à retirer de `tests_pre_migration.jsonl` |
| `make gate-push` | EXIT 0 (§2.3) |
| Push et CI | message de clôture (§2.3) |

Révisions : `grammar.Rev` `grammar-2026-10-06.6` -> `grammar-2026-10-07.5` (une entrée de
chronique par rang) ; `killsource`, `objectives`, `profile`, `source`, `replay.SchemaVersion` (82)
constants. Fixtures de contrat (8 mini-films) : identiques hors chaîne de révision à chaque rang
(vérifié par décompression et substitution).

### 2.1 Le `FAUX` du gate de corpus : `d9781168`, `repli_deadstate_indice_hors_roster` 0 -> 1

Instruit par une sonde en surcouche dans `killsource` (`tsv/sonde_roster_test.go.txt`, jamais dans
le code), sur la marche des morts de `d9781168` (carte Dredge), base et tête :

- base : 44 dead-states `Mort=1`, **1 jeté par désynchronisation** (`repli_record_desynchronise_jete`),
  41 crédibles, 0 hors roster ;
- tête : 45 dead-states, **0 désynchronisation**, 41 crédibles, **1 hors roster** : chunk 11,
  paquet 1794, slot 522, bit 779, `EnumA = -1`, `EnumB = -1`, sans GID ni étiquette de source.

C'est le même dead-state : le record qui désynchronisait sur un composant non porté se lit
désormais jusqu'au bout, et le filtre de crédibilité le rejette (aucun indice de victime ni de
tueur). Le repli `repli_record_desynchronise_jete` du même film passe de 1 à 0 dans la même
cuisson. **Aucune mort publiée ne change** : sur `d9781168`, l'artefact ne diffère que par
`coverage` (compteurs) et `layers` (chaîne de révision) ; banc O-K1 142/0/0, O-K2 144/0/0 comme en
base. Le banc classe tout repli neuf en `FAUX` ; ici un repli remplace l'autre pour la même
donnée, qu'aucune des deux branches ne publie. Admission : geste du pilote.

### 2.2 Les PERTE du gate de corpus (filet, aucune mesure du banc ne les couvre)

- **Durées d'état (`stances/duree-totale`, 10 témoins, 37 lignes par slot)** : `bcb6d393` 7 104 ->
  6 864, `51ebbc0f` 8 970 -> 8 525, `111fa685` 39 123 -> 38 606, `f75e7053` 8 880 -> 8 763, les
  autres à quelques unités. Même nature que celle que L2 avait instruite (`LOT_L2.md` §6.5, §9.4) :
  des paquets qui s'arrêtaient sur un composant non porté se lisent, la fin d'un état (sprint,
  saut) est observée plus tôt (règle « états à l'instant »). Les compteurs `coverage.stances`
  (`refusedNews`, `forgottenBindings`…) bougent avec le nombre de records lus.
- **Trous du tir continu (`coverage.continuousFire.holes*`, 10 témoins)** : des paquets lus plus
  loin atteignent la vue C et y échouent ; le trou change de classe (même constat que L2).
- **`c75f33b8` `weaponChanges/par-kind/swapped` 17 -> 14** : à `t = 3600`, la tête lit un paquet
  où les sept slots 579 à 585 lâchent l'objet `00007ca9` (base : paquet non lu) ; les trois prises
  qui suivent (581 à 3617, 579 à 3645, 584 à 3648) deviennent « prise » au lieu d'« échange »
  (`dropped` + `taken` au lieu de `swapped`). Plus une prise / un lâcher neufs du slot 536 (1300,
  1501). Lecture plus complète, pas une perte.
- **`11de8353` `vehicles.rides` (xuid `…0104` 266 -> 198 s)** : la tête lit, à `t = 1692`, le
  passage du slot 529 du siège 0 du véhicule 785 à la tourelle (slot 784, génération 1) ; le trajet
  de la BASE dans 784 venait du repli de proximité (`src: proximity`, 1661 -> 1729) et doublait le
  trajet dans 785. Il disparaît au profit d'une lecture du film. `rides.aim` suit. Plus une prise
  neuve (slot 541, 1697), une fin d'arme au sol et un trou de rafale en moins.
- Aucune perte de kill, de mort, d'assistance, de score, d'équipe ni de vie (banc 18 / 19 « ok »).

### 2.3 `gate-push`, push, CI

- `make gate-push` (TMP `C:/t/vueb`, GOCACHE et cache golangci dédiés), sur `05b70670a` : **EXIT_GATEPUSH=0** — golangci-lint `--new-from-merge-base=origin/main` `0 issues.` ; web typecheck vert ; web lint 0 erreur (26 avertissements préexistants ; le lot ne touche côté web que les fixtures de contrat, aucun `.ts`/`.tsx`) ; baseline : 9 533 tests de la baseline tous présents (19 374 au run), aucun test ni paquet en échec.
- Push de `feat/grammaire-arrets-vue-b` et CI : état donné dans le message de clôture au pilote (le push suit ce commit).

## 3. Point (d) du §2.2 : non éclairé par ce lot

Les cinq records (`111fa685` 8:700, 8:878, 8:904 slot 559 ; `1c4c63c2` 52:298 slot 525 et 53:554
slot 670) ne sont touchés par aucun des composants du lot : leurs paquets ont la même ligne dans la
carte v2 de base et de tête. 8:700 et 8:878 sortent de la vue B par rejet (slot inconnu du monde),
8:904 s'arrête en vue C (kind non porté), 52:298 n'a pas de liste localisée, 53:554 est sain (fermé
au bit près) : la question y est celle de la marche qui lit le slot, pas d'un composant. Rien n'a
été fait de plus (consigne : sans en faire un lot).

## 4. Découvertes (hors périmètre, non traitées)

- **D1** La famille `ti=12 managed-navpoint-visual-state-groups-component-0..7` (`i20` à `i27`) est
  désormais le premier composant d'arrêt du corpus : 1 640 paquets (`i21` 675, `i22` 532, `i20` 433,
  `i23`..`i25` 27). Elle était déjà bloquante en base (1 178) ; C2 et C5 lui en amènent.
- **D2** `ti=10 i22 managed-object-interaction-filter-component` : 291 arrêts (80 en base), puis
  `ti=10 i18`..`i21 networked-property` (40).
- **D3** `ti=12 i17 managed-navpoint-object-marker` (92), `i13 top-progress` (7), `i15
  bottom-progress` (3) : non portés.
- **D4** `ti=45 i1 matchflow-focus-data-component` : 16 arrêts après C3.
- **D5** `ti=35 i59 biped-spartan-ability-non-predicted-state` (250) et `ti=11 i4
  managed-objective-interaction-filter-component` (141) restent des arrêts notables.
- **D6** `ti=10` porte seize descripteurs du même nom `managed-object-navpoint-component` (un
  tableau de seize identifiants) : le handoff n'en citait qu'un (`i2`).
- **D7** Méthode retrouvée et vérifiée sur `i19` de `ti=43` : nom -> chaîne `.rdata` -> unique
  xref (accesseur de nom) -> slot du descripteur ; le lecteur est le slot qui suit le thunk
  `FUN_14076ce9c` (`+0x28` de l'adresse de l'accesseur), l'écrivain le slot `+0x10` (souvent un saut
  court à décoder à la main : `disassemble_function` n'y trouve pas de fonction, et
  `disassemble_bytes` est un POST qui écrirait dans le programme).
- **D8** La carte v2 complète des 20 films prend environ 1 min 30 sur ce poste (le handoff estimait
  15 à 25 min) ; le gate de corpus environ 15 min, base cuite.

## 5. Ce qui reste

- **Base dépassée** : pendant le lot, `origin/feat/v75` a avancé à `292ef56a5` (schéma 84,
  `grammar.Rev` = `grammar-2026-10-07` sans suffixe, le même nom que le rang C1 de ce lot). Rien n'a
  été refusionné. À la fusion : refusionner `feat/v75`, reprendre les rangs libres suivants pour
  les cinq entrées de chronique (renuméroter `grammar-2026-10-07` à `.5`), régénérer les empreintes
  et les goldens, puis **refaire la carte v2 et le gate 2** sur la tête combinée.
- **Fusion dans `feat/v75` : geste de l'utilisateur.** levelup-57 prépare un lot qui fait lire la
  vue A plus loin (le message de kill sans sa partie optionnelle, environ 2 270 trames de plus) et
  qui prend `grammar` `.7` dans sa propre série : **celui des deux lots qui fusionnera en second dans
  `feat/v75` devra refaire la carte v2 et le gate 2 sur la tête combinée** (et renuméroter
  `grammar.Rev`).
- Admission du gate de corpus (rc 1 : 1 `FAUX` instruit au §2.1, PERTE de filet instruites au §2.2) :
  geste du pilote.
- Recuisson du parc : décision de l'utilisateur (`grammar.Rev` monte ; `SchemaVersion` constant).
- Lots suivants possibles (découvertes D1 à D5), à ouvrir par l'utilisateur.

## 6. Pièces

- `PLAN.md` (items, journal).
- `tsv/gate2.awk` (gate 2 paquet par paquet), `tsv/gate2_*.tsv`, `tsv/causes_*.tsv`.
- `tsv/mutations.sh`, `tsv/mutations_final.txt`.
- `tsv/killsource_diff.txt`, `tsv/gate_corpus_verdict.txt`, `tsv/gate_corpus.json`.
- Sondes de surcouche (jamais dans le code) : `tsv/sonde_held_test.go.txt`, `tsv/sonde_roster_test.go.txt`.
