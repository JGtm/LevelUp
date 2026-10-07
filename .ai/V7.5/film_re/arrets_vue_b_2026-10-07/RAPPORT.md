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

  **Instruction complémentaire (demande du pilote : sept slots qui lâchent le même objet).
  Verdict : événement réel du jeu, lu à la bonne largeur ; le libellé « lâcher » est une
  qualification de la publication.**
  - Unité : `t` est un rang de frame de 100 ms (`frameIntervalMs = 100`) : `t = 3600` = 360,0 s
    après l'origine du document.
  - Ce que le jeu fait à ce moment : Assaut, une bombe (`Assault:One Bomb`, Curfew), en manches.
    Le camp 1 marque la manche 1 à `t = 3489` (`scoreTimeline`) ; tous les corps de la manche
    finissent à `t = 3566` ; les corps 579 à 585 (génération 1) ont leur premier point entre
    3570 et 3601 : c'est la MISE EN PLACE DES JOUEURS de la manche suivante.
  - Le paquet : chunk 21, paquet 816 (horodatage 10 131 559 824 µs), sonde en surcouche sur
    `lecturesBipedes` (aucun code). Base, C1 et C2 : liste lue (fin de la vue A), arrêt sur
    `ti=45 i0`, 0 record utile. À partir de C3 : terminateur atteint, **fermé au bit près, sain**,
    aucune règle de l'écrivain contredite, 7 records utiles. **C'est C3 (`ti=45 i0`) qui l'ouvre** :
    rendre à la tête l'ancien `components_moteur_de_partie.go` et retirer le lecteur `ti=45` fait
    disparaître exactement ces sept lectures (et les huit du paquet 4:1088, la mise en place de la
    première manche, avant l'origine du document, non publiées). Même forme que le paquet 2:712 de
    `000d5950` (perte (a) du handoff).
  - Ce que le film écrit : pour chacun des sept bipèdes, le composant `i45` (emplacement 2)
    annonce un emplacement VIDE (famille `0xFFFFFFFF`, `Kind = restated` à la lecture).
  - `00007ca9` : c'est l'objet « mains nues » (`filmshell.UnarmedFamily`, tag `weap`
    `WeaponTags.unarmed`, lot M6.3), que le jeu remet à chaque bipède au début de chaque vie. Dans
    l'artefact, il n'apparaît que comme `from` de `weaponChanges` (10 fois : ces 7 lâchers, et 3
    échanges à `t` 803, 827, 838, à la mise en place de la manche précédente, présents en base
    comme en tête). Il vient de la DOTATION DE NAISSANCE des corps (`loadouts`, `src: birth`,
    `t = 3570`), où l'emplacement 2 tient les mains nues, écartées de l'affichage par la règle nommée.
  - Le mécanisme : `qualifyHeldWeaponChange` (`grammar/held_weapon_changes.go`) juge la première
    émission d'un emplacement contre la dotation de naissance quand elle le situe ; une annonce
    « vide » contre une dotation « mains nues » devient un lâcher qui nomme les mains nues. En base,
    l'annonce n'était pas lue : la prise suivante était qualifiée contre la même dotation et sortait
    en « échange depuis mains nues » (581 à 3617, 579 à 3645, 584 à 3648). Aucune lecture n'est à
    une mauvaise largeur ; la publication de « lâche les mains nues » à chaque mise en place est
    un défaut de qualification, antérieur au lot (il produisait déjà les « échanges depuis mains
    nues »), consigné en découverte D9.
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

- **D9** Publication des changements d'arme : à la mise en place d'une manche, l'annonce
  « emplacement 2 vide » d'un corps neuf, qualifiée contre sa dotation de naissance « mains
  nues », est publiée comme un lâcher des mains nues (`c75f33b8`, sept à `t = 3600`) ; quand
  l'annonce n'est pas lue, la prise suivante sort en « échange depuis mains nues ». Règle de
  qualification à revoir (`qualifyHeldWeaponChange`, règle des mains nues de `filmshell`), hors
  périmètre. La mise en place de la manche précédente de ce film (`t` ≈ 753, corps 525 à 530)
  n'est toujours pas lue. **Corrigé le 2026-10-07 sur GO de l'utilisateur : §7.**
- **D10** Cache de faits du checkout principal : aucun fichier de `data/cache/film_facts` n'a été
  écrit entre 15 h 15 et 17 h 40 (fenêtre où les binaires de cette branche ont tourné) ; les 126
  faits portant `grammar-2026-10-07` (même chaîne que le rang C1 de ce lot) sont datés de 17 h 43
  à 18 h 02, la recuisson de `feat/v75` par une autre session. Aucun outil de ce lot n'écrit dans
  le checkout principal : cartes et sondes au scratchpad, `killsource json` en lecture, gate de
  corpus sur une copie du parc.

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

## 7. Correctif D9 — les mains nues valent « rien en main » (2026-10-07)

> GO de l'utilisateur le 2026-10-07 : « "lâche ses mains nues" ne veut absolument rien dire ».
> Exécuté sous `plan-execution` sur la même branche : un commit de plus (`bf0a6f856`).

### 7.1 La règle, à la source

- **Où naît le faux événement** : `qualifierContre`
  (`apps/go-api/internal/games/halo_infinite/film/internal/grammar/held_weapon_changes.go`, l. 230).
  C'est le seul point par lequel passent les deux chemins de qualification de
  `qualifyHeldWeaponChange` : l'émission précédente de la même vie, ou la dotation de naissance qui
  situe l'emplacement. La dotation de naissance porte l'objet « mains nues »
  (`filmshell.UnarmedFamily`, `00007ca9`) à l'emplacement 2, et il y était jugé comme une arme. Une
  annonce « emplacement vide » sortait donc en lâcher des mains nues, et une prise en échange depuis
  les mains nues.
- **Correctif, une règle générale** (aucun film, carte, mode ni version nommé) : `armeEnMain`
  (l. 249) dit qu'une famille est une arme si elle n'est ni l'emplacement vide ni les mains nues.
  Quand la référence vaut « rien en main » :
  - une annonce vide est une ré-annonce, non publiée (l. 236) ;
  - une arme est une **prise**, sans `from` ;
  - une émission des mains nues après une arme est le **lâcher de cette arme**, emplacement publié
    vide, jamais un échange vers les mains nues. Les commentaires de `filmshell/unarmed.go` et de la
    publication, qui annonçaient un échange, sont corrigés ;
  - seule la remise (mains nues sur un emplacement vide) garde sa nature de prise (l. 234) : la
    publication l'écarte et la compte dans `unarmedGrants`, comme décidé le 2026-09-24.
- **Autres chemins vérifiés par `grep`** (aucun ne fabrique le même faux événement) : les
  ramassages natifs (`document_pickups.go`) et les dotations (`loadouts.go`,
  `document_birth_loadouts.go`) écartent déjà les mains nues. `killsource` et `objectives` ne lisent
  pas ce canal. Les autres lecteurs du canal (`bomb_carries.go`, armes au sol, `fire_bursts.go`,
  `padtiers_prises.go`) consomment les natures déjà qualifiées. Le web (`changeRefine.ts`, sons,
  calques) applique les événements du document et n'en fabrique aucun. Rien à signaler hors
  périmètre.

### 7.2 Comptes avant / après

Outil : `tsv/compte_mains_nues.sh` (jq, lecture seule). Colonnes : lâcher des mains nues, échange
depuis, échange vers, prise des mains nues, toute autre chaîne `00007ca9` du document.

| Corpus | Lâcher | Échange depuis | Échange vers | Prise | Autre champ | Total |
|---|---|---|---|---|---|---|
| Parc du checkout principal, 126 artefacts, **schéma 86** (`feat/v75` `cdd642061`), compté à **19 h 52** | 1 (1 film) | 13 (6 films) | 0 | 0 | 0 | **14 dans 6 films** |
| 19 témoins, base `12b8fb3df` (cuisson du gate) | 13 (4 témoins) | 5 (3 témoins) | 0 | 0 | 0 | **18 dans 5 témoins** |
| 19 témoins, tête (cuisson du gate) | 0 | 0 | 0 | 0 | 0 | **0** |

- Parc (`tsv/d9_parc_avant_schema86.tsv`) : `1f83b32b` 2 échanges, `51ebbc0f` 1, `7fce3219` 4,
  `d9781168` 1 lâcher et 1 échange, `f3622d94` 1, `f9e99ca4` 4. Un premier compte au schéma 84,
  avant la republication de `levelup-d0`, donnait les mêmes chiffres. Le parc n'a pas été recuit.
- Témoins (`tsv/d9_temoins_base.tsv`, `tsv/d9_temoins_tete.tsv`) : les publiés passent de
  378 / 497 / 1 225 (lâchers / échanges / prises) à 365 / 492 / 1 230, soit exactement −13 lâchers,
  −5 échanges et +5 prises. Les remises écartées (`unarmedGrants`) restent à 6.
- **Rien d'autre ne bouge.** Sur chacun des 19 témoins, les changements d'arme hors de ces
  18 instants sont identiques. Le reste du document (hors `weaponChanges`, couverture et calques)
  est identique. La couverture et les calques ne diffèrent que par les chaînes de révision.

### 7.3 Gate de corpus

Commande : `replay-corpus-gate --reference=base --base=12b8fb3df --keep-work`, sur une copie du parc
(`C:/t/d9parc` : base partagée identique à l'octet, chunks et manifestes des 19 témoins). Tête
`f0ddfd1bf` : même code que `bf0a6f856`, qui n'y ajoute que le plafond de l'archlint et la mesure
dans la chronique (commit amendé avant tout push). Pièces : `tsv/d9_gate_corpus_verdict.txt`,
`tsv/d9_gate_corpus.json`.

- **rc = 1** (statut PERTE de filet sur 5 témoins). **Banc de vérité 19 / 19 « ok », 0 MANQUE,
  0 FAUX.** Un gain : `50247b26` `V-3 action hors vie` 4 -> 3 (le lâcher des mains nues du slot 512
  à `t = 447` était une action hors vie).
- Chaque PERTE est instruite événement par événement ; elles portent toutes sur `weaponChanges` :

| Témoin | Base `12b8fb3df` | Tête | Instruction |
|---|---|---|---|
| `c75f33b8` | 7 lâchers à `t = 3600` (slots 579 à 585) ; 3 échanges depuis les mains nues à 803, 827, 838 | les 7 lâchers ne sont plus publiés ; les 3 échanges sont des prises (même arme, même emplacement) | dropped 22 -> 15, swapped 14 -> 11, taken 37 -> 40. Les trois prises 581 à 3617, 579 à 3645 et 584 à 3648 restent des **prises**, comme en base, qui lisait déjà l'annonce vide. Contre `879f31bbf` (avant le lot) : swapped 17 -> 11 ; les six échanges depuis les mains nues sont tous devenus des prises |
| `4f77afc1` | 3 lâchers à `t = 221` (slots 524, 534, 535) | non publiés | dropped 30 -> 27 |
| `50247b26` | 2 lâchers à `t = 447` (slots 512, 513) | non publiés | dropped 26 -> 24 ; le gain V-3 ci-dessus |
| `d9781168` | 1 lâcher à 4286 (slot 605), 1 échange à 4378 (slot 602) | lâcher non publié ; l'échange est une prise | dropped 11 -> 10, swapped 28 -> 27, taken 55 -> 56 |
| `51ebbc0f` | 1 échange à 2100 (slot 555) | prise | swapped 12 -> 11, taken 44 -> 45 |
| `396cfc92` (« ok ») | — | — | couverture seule : une émission antérieure à l'origine, non publiée dans les deux cas, passe de `beforeOrigin` à `restated` (2 -> 1, 7 -> 8) |

### 7.4 Preuve unitaire et mutations

- `TestLesMainsNuesValentRienEnMain` (naissance avec dotation mains nues, annonce vide, puis prise)
  et `TestAucunChangementNePorteLesMainsNues` (six transitions), dans
  `grammar/held_weapon_chain_test.go`. **Rouges sur le code de `12b8fb3df`** (« dropped depuis
  00007ca9, attendu une re-annonce » ; « swapped 0000000c depuis 00007ca9, attendu taken »), verts
  après.
- Mutations (`tsv/mutations_d9.sh`, sur copie du fichier, restauré après chaque mutation) :
  **5 / 5 ROUGES** (`tsv/mutations_d9.txt`).

### 7.5 Rangs montés (rangs de TRAVAIL, à renuméroter à la fusion)

| Révision | Ici | Pris ailleurs (relais du pilote, 20 h) | À la fusion |
|---|---|---|---|
| `replay.SchemaVersion` | 82 -> **83** | `feat/v75` : 86 (fusionné) ; levelup-57 : 87 | prendre le rang libre suivant (88 si levelup-57 fusionne avant) ; renommer les fixtures `replay_schema_<N>_*`, reprendre l'entrée de chronique, `structure_test.go` et les deux plafonds de `film_file_size_test.go` |
| `grammar.Rev` | `grammar-2026-10-07.5` -> **`.6`** | `feat/v75` : `grammar-2026-10-07` ; levelup-57 : `.2` | renuméroter avec les cinq rangs du lot (§5) : six entrées de chronique à reprendre à la suite |
| `killsource.Rev`, `objectives.Rev`, `source`, `profile`, `SchemaDesFaits` | constants (empreintes régénérées) | faits 10 (levelup-57) | régénérer les empreintes |

`replay.SchemaVersion` monte, en plus de `grammar.Rev`, parce que le contenu publié change et que
`backfill-replay` reprend par numéro de schéma (précédent : v78, « aucun champ neuf ; le CONTENU
change »). Les 8 fixtures de contrat sont identiques hors chaînes de version (vérifié par
décompression et substitution). Dans les goldens d'assemblage, seule la ligne de schéma change.

### 7.6 Gates

| Gate | Sortie |
|---|---|
| gofmt | vide |
| vet, vet `-tags=research`, vet `-tags=integration` (film) | rc 0, rc 0, rc 0 |
| golangci-lint `--new-from-rev=12b8fb3df` (film, filmshell) | `0 issues.` |
| Tests `film/...`, `filmshell`, `replaybuild`, `killcollector`, `replayartifacts` (`-count=1`) | 23 paquets `ok`, rc 0 |
| archlint | `ok` ; plafonds de `document_chronicle.go` 2929 -> 2960 et `structure_test.go` 1396 -> 1400 (exception écrite, dans le commit qui monte `SchemaVersion`) |
| Baseline des tests | 2 tests ajoutés, aucun retiré ni renommé |
| `make gate-push` | 7.7 |
| Push et CI | 7.7 |

### 7.7 `gate-push`, push, CI

- `make gate-push` (TMP `C:/t/d9gp`, GOCACHE et cache golangci dédiés), sur `bf0a6f856`, de 20 h 34
  à 21 h 11 : **EXIT_GATEPUSH=2**. golangci-lint `--new-from-merge-base=origin/main` : `0 issues.` ;
  web typecheck vert ; web lint 0 erreur (26 avertissements préexistants). La baseline échoue sur
  trois points, tous hors des paquets touchés, et tous instruits :
  - `internal/platform/duckdb` et `internal/sync` sont coupés au plafond local de 300 s par paquet
    (`-timeout=300s`) pendant que des tests passent : 1 001 tests verts au moment de la coupure,
    aucun en échec. Rejoués seuls avec un plafond plus large : **`ok` en 313 s et 310 s**, 0 échec ;
  - `internal/sync/killcollector::TestRosterDesFilms_AnnuaireContreJointure` (intégration) est un
    rapport de durées (facteur attendu >= 10). Rejoué seul trois fois : 2 réussites, 1 échec
    (145 ms contre 53 ms), donc instable sous charge. Le paquet n'est pas touché par ce correctif ;
  - en fusionnant le JSONL du run avec celui des deux paquets rejoués
    (`check_test_baseline.sh tests --from-jsonl`) : **tous les tests de la baseline sont présents**,
    et seul le test de durées ci-dessus est en échec.
  - La charge de la machine pendant le gate : un `sed` et un `awk` orphelins tournent depuis le
    3 et le 6 octobre (248 000 et 116 000 secondes de CPU cumulées), et Halo Infinite a démarré à
    21 h 11 ; le CPU moyen était à 66 % à 21 h 40 (découverte D11). Aucun de ces processus n'est
    de cette session : ni arrêtés, ni traités.
- Push de `feat/grammaire-arrets-vue-b` (`12b8fb3df..20a52831c`) et CI, gate d'autorité :
  **CI verte** sur `20a52831c` (run 37675971549, de 21 h 38 à 22 h 19) : build et tests Windows et
  Ubuntu, couverture et baseline (`./...` complet, CGO), golangci-lint, course du film (`-race`),
  contrat OpenAPI, frontend, lease ; E2E Playwright sauté (réservé aux PR vers `main`). Gitleaks et
  Deploy Pre-Check verts. Ce paragraphe est le seul ajout du commit suivant (Markdown sous `.ai/`,
  hors du déclencheur de la CI).

### 7.8 Découvertes (non traitées)

- **D11** Deux processus orphelins (`sed` depuis le 2026-10-03, `awk` depuis le 2026-10-06) consomment
  du CPU en continu sur le poste. `internal/platform/duckdb` et `internal/sync` dépassent alors le
  plafond local de 300 s du gate-push, et `TestRosterDesFilms_AnnuaireContreJointure` devient instable.
- **D12** Le libellé `hinf_unarmed` (« Mains nues », `weapon_names.toml`) n'a plus de chemin de
  publication dans les changements d'arme. Il servait, selon la décision du 2026-09-24, à nommer
  les mains nues si elles apparaissaient en cours de partie. Autres usages possibles (fil des
  kills, armes de mêlée) non vérifiés : à instruire avant de le juger mort.

## 8. Intégration — fusion de `feat/v75` dans la branche du lot (2026-10-07)

> Accord de l'utilisateur à l'intégration le 2026-10-07. Exécuté sous `plan-execution`. Tête de
> `origin/feat/v75` fusionnée : **`312073cd3`** (inchangée au `git fetch` de 23 h). La branche du
> lot reçoit `feat/v75` ; l'avance de `feat/v75` reste au pilote. Aucune cuisson du parc.

### 8.1 Fusion (`caf08c7e6`)

- `git merge origin/feat/v75`, sans rebase ni stash. 21 fichiers en conflit, tous des rangs ou des
  sorties générées : `grammar.Rev`, `replay.SchemaVersion`, leurs chroniques, `structure_test.go`,
  les deux plafonds de `archlint/film_file_size_test.go`, les goldens de révision, d'assemblage et
  de forme, les fixtures de contrat du web et leur manifeste. `feat/v75` est retenu pour tout ce qui
  n'est pas le lot ; les lecteurs C1 à C5 et le correctif D9 sont gardés tels quels (fusion
  automatique, aucun conflit dans leurs fichiers). Chroniques : toutes les entrées des deux côtés,
  celles de `feat/v75` d'abord. `.ai/thought_log.md` : fusion automatique, aucune ligne perdue d'un
  côté ni de l'autre (vérifié par `git diff` contre chaque parent : 0 suppression).
- `rev_chronique.go` (grammaire) passait 500 lignes (528) : les rangs `grammar-2026-10-02` à `.3`
  sont versés mot pour mot dans `rev_chronique_archive_7.go`, comme aux rotations précédentes.
- Les sorties générées sont reprises du côté `feat/v75`, puis régénérées par les commandes du dépôt.

### 8.2 Rangs finaux

| Révision | Sur la branche | Dans `feat/v75` (`312073cd3`) | Final |
|---|---|---|---|
| `grammar.Rev` C1 `ti=43` | `grammar-2026-10-07` | `grammar-2026-10-07` (zones), `.2` (RI 2.7.c) | **`grammar-2026-10-07.3`** |
| C2 `ti=12 i16` | `.2` | | **`.4`** |
| C3 `ti=45 i0` | `.3` | | **`.5`** |
| C4 `ti=10 i2` à `i17` | `.4` | | **`.6`** |
| C5 `ti=12 i18` | `.5` | | **`.7`** |
| D9 mains nues | `.6` | | **`.8`** (constante courante) |
| `replay.SchemaVersion` | 83 (travail) | 87 (84, 86, 87 ; 83 et 85 sans emploi) | **88** |
| `killsource.Rev` | `killsource-2026-09-27` | `killsource-2026-10-07` | **`killsource-2026-10-07.2`** (§8.4) |
| `objectives.Rev` | `objectives-2026-09-27` | idem | constante, golden régénéré |
| `SchemaDesFaits` | 7 | 10 | **10**, constant : aucune section des faits ne change de forme |
| `profile.Rev`, `source.Rev` | | | constantes |

- Golden de `grammar.Rev` : le test exige une ligne par rang, sans trou. Les rangs `.3` à `.7`
  portent l'empreinte RÉELLE de chaque état intermédiaire sur la tête combinée (`feat/v75` + C1,
  + C2, …), calculée par `revision.EmpreinteDeCouche` après application, dans l'ordre, des diffs de
  source de chaque commit du lot (grammaire, et `filmshell/unarmed.go` pour D9) ; l'arbre est revenu
  à l'identique ensuite (`diff -r`). Contrôle : l'état `.8` ainsi calculé égale l'empreinte que le
  test calcule (`e7d248bb…`).
- Aucun rang en double (`grep` des entrées `ENTREE` de la chronique de grammaire, des entrées `vN`
  de la chronique du document et des lignes du golden). Le doublon `v14` de la chronique du document
  est antérieur (déjà dans `feat/v75`).
- Fixtures de contrat `replay_schema_88_*` : identiques aux `replay_schema_87_*` de `feat/v75` hors
  chaînes de version et de révision (8 / 8, décompression et substitution). Goldens d'assemblage :
  seule la ligne de schéma change.

### 8.3 Gates sur la tête combinée

| Gate | Sortie |
|---|---|
| gofmt (`film/`, `cmd/`, `archlint`, `replaybuild`, `killcollector`, `filmshell`) | vide |
| vet, vet `-tags=research`, vet `-tags=integration` (film) | rc 0, rc 0, rc 0 |
| Tests `film/...`, `replaybuild`, `killcollector`, `replayartifacts`, `filmshell`, archlint (`-count=1`) | 24 paquets `ok`, rc 0 |
| golangci-lint `--new-from-rev=312073cd3` (film, filmshell, archlint) | `0 issues.` |
| **Gate 2**, carte v2 (20 films) base `312073cd3` contre tête | **+22 359 sains (464 829 -> 487 188), +216 887 utiles sains (5 837 720 -> 6 054 607), 0 paquet sain perdu, 0 film en baisse** (`tsv/gate2_integration_contre_v75.tsv` ; causes : `tsv/causes_integration.tsv`) |
| **Gate 3**, `killsource json` (19 témoins + `1c4c63c2`) | 17 identiques à l'octet ; `c75f33b8` : diagnostic `calibration` seul ; `e5adf7b2` et `1c4c63c2` : trois morts changent de voie, contenu identique (§8.4) ; `killsource.Rev` monte (`tsv/killsource_diff_integration.txt`) |
| `TestGoldenFilms` (`KILLSOURCE_FIXTURES`) | `ok`, 4 / 4 films |
| **Gate de corpus** `--reference=base --base=312073cd3`, parc copié au scratchpad (base partagée identique à l'octet, chunks, manifestes et artefacts des 19 témoins), en quatre passes `--temoins` | rc 1 à chaque passe (`codePerte`), aucune erreur de cuisson ; **banc de vérité 19 / 19 « ok », 0 FAUX, 0 MANQUE** ; PERTE de filet sur 17 témoins, toutes instruites (§8.5) ; télémétrie `grammarRev` `.2` -> `.8` (19) (`tsv/integration_gate_corpus_verdict.txt`, `tsv/integration_gate_corpus.json`) |
| Mutations (`tsv/mutations.sh` C1 à C5, `tsv/mutations_d9.sh`) | **33 / 33 ROUGES** (28 + 5), base verte (`tsv/mutations_integration.txt`) |
| Baseline des tests | contre `feat/v75` : 0 `func Test` retiré ni renommé, 5 ajoutés (ceux du lot) |

Gate 2 par film (tous « ok ») : `bcb6d393` 6 479 -> 18 991, `396cfc92` +4 801, `f75e7053` +1 732,
`1c4c63c2` +1 600, `4f77afc1` +504, `111fa685` +312, `fb1a1a72` +284, `e5adf7b2` +187, les autres
de +2 à +108. Les gains sont du même ordre que contre `879f31bbf` (+22 282) : le nouveau début de
vue B de `feat/v75` (RI 2.7.c) ne retire rien de ce que les lecteurs du lot ouvrent.

### 8.4 Gate 3 : trois morts passent du balayage à la marche

Dans `feat/v75`, killsource lit ses dead-states par la marche des trames (RI 2.7.c). Les lecteurs
du lot prolongent la vue B de paquets où la marche ne passait pas : trois morts des témoins sont
désormais lues par la marche au lieu du balayage — `e5adf7b2` à 05:29 (`Madina97294`), `1c4c63c2` à
08:15 (`elnewtsy117`) et à 13:38 (`Luxiy XCV`). Tout le reste de chaque mort est identique (victime,
tueur, instant, source, assistances, divergence ; comparaison par `jq` hors du bloc `lecture`) ;
les comptes `gate_par_voie` et `concordance` suivent. `c75f33b8` ne diffère que par le score et la
médiane de l'oracle de `calibration` (décision inchangée), comme au §2.

`lecture.voie_identifiant` est persisté dans `match_kill_events.read_path`
(`killcollector/collector_batch.go`) : la sortie persistée change, donc **`killsource.Rev` monte à
`killsource-2026-10-07.2`** (D23), entrée de chronique écrite, golden régénéré (commit
`5b8af2eaa`). Les lignes de kill du parc deviennent candidates au redécodage : backlog sur signal
de l'utilisateur (D6). `SchemaDesFaits` ne monte pas : aucune section des faits ne change de forme,
et les faits d'avant sont déjà périmés par leurs révisions de couche.

### 8.5 Gate de corpus : les PERTE, une par une

Chaque clé de perte (`DETAIL DES PERTES`) a été rapprochée des deux gates de corpus précédents de
ce lot (§2, §7.3). Trois seulement n'y figuraient pas : `e5adf7b2`
`coverage.deathsPaths.directScan.{matched,published}` 4 -> 3 (la mort de 05:29 du §8.4, qui passe
au chemin de la marche) et `e5adf7b2` `stances/duree-totale/par-slot/516` (durées d'état, ci-dessous).

- **Durées d'état et compteurs des états** (`stances/duree-totale`, `refusedNews`,
  `refusedNewFalseReads`, `refusedNewUndecided`, `forgottenBindings`, `jumpsDerived` ; 14 témoins) :
  même nature qu'au §2.2 (fin d'un état observée plus tôt par des paquets désormais lus). Les
  totaux sont ceux du §2.2 à quelques unités près (`bcb6d393` 7 104 -> 6 864, `111fa685` 39 123 ->
  38 606, `f75e7053` 8 880 -> 8 763, `51ebbc0f` 9 004 -> 8 559).
- **Trous du tir continu** (`coverage.continuousFire.*`, 10 témoins) : même nature qu'au §2.2.
- **Changements d'arme** (`d9781168`, `c75f33b8`, `51ebbc0f`, `50247b26`) : ce sont exactement les
  mains nues du correctif D9. Base : 10 changements publiés portent `00007ca9` sur ces quatre
  témoins (`tsv/compte_mains_nues.sh`) ; tête : 0 sur les 19. En retirant de la base les lâchers des
  mains nues et en rendant ses « échanges depuis les mains nues » en prises, la liste des
  changements d'arme est identique à celle de la tête sur `d9781168` et `50247b26` ; sur
  `51ebbc0f` et `c75f33b8`, la tête n'a en plus que des lectures neuves (une prise à 1275, slot 534 ;
  la prise et le lâcher du slot 536 à 1300 et 1501, déjà au §2.2).
- **`11de8353`** (trajet de véhicule 266 -> 198 s, visée, véhicules montés, fin d'arme au sol, trou
  de rafale du slot 562) : lignes identiques au gate du §2, instruites au §2.2 (le trajet de
  proximité de la base, doublon du trajet lu, disparaît).
- **`a349fea8`** `coverage.birthLoadouts.noDisplayable` 31 -> 32 et `coverage.vehicles.deathsUnmatched`
  0 -> 1 : déjà au gate du §2 (mêmes valeurs). Un record NEW de bipède de plus est lu jusqu'au
  bout (`closed` 54 -> 55, `unconfirmed` 262 -> 261) et sa dotation ne porte aucune arme affichable ;
  une mort de véhicule de plus est lue (`deathsRead` 33 -> 34), qu'aucune vie recensée ne reprend.
  `loadouts`, `vehicles`, `weaponChanges`, `tracks` et `shots` publiés sont identiques à l'octet
  entre base et tête : ce sont des compteurs de lecture, rien de publié ne se perd. La mort neuve
  n'a pas été localisée paquet par paquet.
- Aucune perte de kill, de mort, d'assistance, de score, d'équipe ni de vie (banc 19 / 19 « ok »).

### 8.6 Découvertes

- **D13** Depuis RI 2.7.c, killsource lit la marche des trames : tout lot qui porte un composant
  de la vue B peut déplacer la voie d'une mort, donc faire monter `killsource.Rev`. Le handoff des
  composants bloquants (§5) supposait le gate 3 identique ; pour les lots suivants, la règle D23
  tranche (ici : la révision monte).
- **D14** Le gate de corpus ne tient pas dans un appel de dix minutes quand la base n'est pas
  cuite ; il a été joué en quatre passes `--temoins` sur la même copie du parc, verdicts
  concaténés.

### 8.7 `gate-push`, push, CI

État dans le paragraphe suivant (ajouté après le gate-push).
