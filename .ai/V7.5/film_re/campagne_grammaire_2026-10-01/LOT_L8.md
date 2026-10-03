# Lot L8 — `ti=3 low-frequency` porté, `high-frequency` routé par la table de l'archétype (2026-10-02)

> Lot L8 du plan `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (§6.2 L8, gate §6.0 points 1, 2, 3, 6 et
> 7), sous le contrat `plan-execution`, dans le cadre de la décision du 2026-10-02 au soir :
> **corrections d'abord, uniquement générales, lues dans le jeu**. Chaque lecteur cite la fonction du
> jeu qui le fonde (Ghidra, `HaloInfinite.exe`, lecture seule, HTTP direct 127.0.0.1:8089).
>
> Worktree `LevelUp-wt-cg-l8`, branche `feat/cg-l8`, base `af6e93e23` (L0 fusionné). Films lus en
> place (`LevelUp/data/cache/film_chunks`, lecture seule), un à la fois. Aucune cuisson du parc.
> Mesures et scripts : `scratchpad/L8/` (session `f46f71fc`).
> Convention : **mesuré** = compté par un outil sur les films ; **établi** = lu dans le jeu ;
> **supposé** = hypothèse écrite.

## 0. Statut

| Item | Statut | En une ligne |
|---|---|---|
| L8.1 lecteur `ti=3 i0 low-frequency` | [x] | `consumeLowFrequency` = `FUN_142ed4aec`, vecteurs d'après l'écrivain `FUN_142eda938` |
| L8.2 `high-frequency` routé par table | [x] | `consumeHighFrequency` : `ti=3` → `FUN_142ed4880` (26 bits), `ti=4` → `FUN_14076d034` (R(8), sonde), autre archétype → non porté |
| L8.3 `ecs_table.tsv` (D-86) | [x] | `ti=3 i0` porté, `ti=3 i1` lecteur `FUN_142ed4880` (26), `ti=4 i0` remarque « KEYFRAME : 26 bits » retirée (contredite par le jeu, §1.4) |
| L8.4 garde-fou « dispatch par table » (D-89) | [x] | contrôle G6 `TestG6LesHomonymesSeRoutentParTable` |
| L8.5 `1c4c63c2` (−1 sain au juge à trois règles) | [x] | instruit paquet par paquet (`24:510`) : tête de liste lue sur un NEW au masque impossible ; déjà résolu par L0 (§4.2) |
| L8.6 révision | [x] | `grammar-2026-10-02` → `grammar-2026-10-02.2` (le format n'admet pas de suffixe de lot, D-L8-2) |
| Gates 1, 2, 3, 6, 7 | voir §5 | 1, 2, 3, 7 tenus ; 6 : `replay-equiv` instruit, `replay-corpus-gate` rc 1 (banc 19 / 19 ok, PERTE sur 8 témoins : durée des stances et compteurs de couverture, instruits §5.4 et, NEW refusés compris, §8.2) |
| Corrections du contrôle (2026-10-03) | [x] sauf C2d [!] | vecteurs des deux branches de la porte d'orientation, instruction complète du gate de corpus, règle 17, thought_log (§8) ; la décision sur le rc 1 reste au pilote ou à l'utilisateur (§6) |

Verdict : **[x] retenu** (voir §5, §6 et §8 pour les écarts).

## 1. Ce qui est lu dans le jeu (établi)

### 1.1 Enregistrement des archétypes 3 et 4

`FUN_140e460fc` : `FUN_14064dd28(arch+8, 0, &0x144746e68)`, `FUN_14064dd28(arch+8, 1, &0x144746e60)`,
`*(arch+0x4754) = 3`. `FUN_140e462d8` : `FUN_14064dd28(arch+8, 0, &0x144746d38)`, `*(arch+0x4754) = 4`.
Les objets pointent sur les vtables lues en mémoire (`read_memory`) :

| (ti, i) | objet | vtable | `+0x08` nom | `+0x18` écrivain | `+0x28` | `+0x30` lecteur |
|---|---|---|---|---|---|---|
| 3, 0 | `0x144746e68` | `0x143d07b38` | `0x141177bd0` → « low-frequency » (`0x143c957c8`) | `FUN_142eda938` | thunk `FUN_14076ce9c` | `FUN_142ed4aec` |
| 3, 1 | `0x144746e60` | `0x143d07ae8` | `0x14119d7f0` → « high-frequency » (`0x143c95710`) | `FUN_142eda744` | thunk `FUN_14076ce9c` | `FUN_142ed4880` |
| 4, 0 | `0x144746d38` | `0x143d06a58` | `0x14119d7f0` → « high-frequency » | `FUN_142eda680` | thunk `FUN_14076ce9c` | `FUN_14076d034` |

(table = vtable + 8 : `0x143d07b40`, `0x143d07af0`, `0x143d06a60`, comme R-HOM.) Les accesseurs de
nom sont des `LEA RAX,[rip+..] ; RET` relus octet par octet. Le registre des 20 films (et de
`81c02726`) porte partout `ti=3` = [`low-frequency`, `high-frequency`] et `ti=4` = [`high-frequency`]
(`r_comp_tsv/r_comp_registres.tsv`, 9 builds) : mesuré.

### 1.2 `FUN_142ed4aec` (lecteur `ti=3 i0`) et son écrivain `FUN_142eda938`

Lecteur (décompilé et désassemblé) : `FUN_1424e0e38` (thunk de `FUN_14076e494`, `R8D = R13D = 0x10`,
`R9D = 0`, CALL `142ed4b1f`) → `+0x508` ; `FUN_140c5f938(.., +0x514, +0x520, 0)` ; `R(16)` → `+0x52c` ;
`R(8)` → `+0x52e` ; `R(2)` → `+0x52f` ; `n = R(6)` → `+0x0` ; `n` fois, entrée de 0x28 octets à partir
de `+0x8` : `FUN_1424d9a30` (`R(3)`, destination `RDI+0x27`, `LEA RSI,[RDI+0x27]` en `142ed4e5e`) ; si
bit 1 : position (CALL `142ed4e7f`, même niveau 0x10) ; si bit 2 : `FUN_140c5f938` ; `R(16)` → `+0x24` ;
`FUN_1424ccc74` (`R(5)` → `+0x26`). Le bit 4 du drapeau n'est consulté ni par le lecteur ni par
l'écrivain.

Écrivain : `FUN_141f860b0` (position, `FUN_1407eb61c(.., 0xffffffff, 0x10, 0)`), `FUN_141f86118`
(orientation), `W(16)`, `W(8)`, `W(2)`, `W(6)` du compte, puis par entrée `FUN_142b67fe8` (`W(3)`),
position et orientation sous les mêmes bits, `W(16)`, `FUN_142af2af0` (`W(5)`). Le lecteur Go suit
champ pour champ ; les vecteurs (`components_frequences_test.go`) sont écrits d'après cet écrivain.

### 1.3 `FUN_142ed4880` (`ti=3 i1`) et `FUN_14076d034` (`ti=4 i0`)

`FUN_142ed4880` : `R(16)` → `+0x52c`, `R(8)` → `+0x52e`, `R(2)` → `+0x52f` (26 bits ; écrivain
`FUN_142eda744`, mêmes trois champs). `FUN_14076d034` : `R(8)` (écrivain `FUN_142eda680`).

### 1.4 Les deux boucles de composants appellent le même lecteur

Boucle delta `FUN_14076cb60` : `CALL qword ptr [RAX+0x28]` en `14076cd19` ; boucle d'image-clé
`FUN_142e2c690` : `CALL qword ptr [RAX+0x28]` en `142e2c7c9`, `RBX = [R13 + R15*8]` = l'objet de
composant du descripteur. Établi : `ti=4 i0` se lit `R(8)` en image-clé comme en delta ; la remarque
« KEYFRAME : 26 bits (distinct) » de `ecs_table.tsv` est contredite (D-86, D-L8-3).

## 2. Ce qui change

Code (`film/internal/grammar/`) :
- `components_frequences.go` (neuf) : `consumeLowFrequency` (`FUN_142ed4aec`),
  `consumeChampsDeFrequence` (les trois champs `+0x52c/+0x52e/+0x52f` des deux lecteurs de `ti=3`),
  `consumeHighFrequency` : choix du lecteur par l'archétype du record (`TypeIndex`, la valeur que la
  fonction d'enregistrement pose en `+0x4754`) ; archétype sans table connue de ce nom → non porté
  (la traversée s'arrête) au lieu d'une lecture sous la grammaire d'un autre archétype.
- `dispatch_item.go` : `case compHighFrequency` → `consumeHighFrequency` ; `case compLowFrequency` neuf.
- `testdata/ecs_table.tsv` : trois lignes (`ti=3 i0` `porte`, `ti=3 i1` / `ti=4 i0` `partiel` — statut
  donné par G1 à un `case` qui peut rendre non porté, convention des filtres de `ti=12` —, lecteurs,
  largeurs 26 et 8).
- Garde-fous : G6 neuf (`ecs_dispatch_table_guard_test.go`, homonymes de la table = liste recensée
  `{high-frequency}` ; chaque ligne d'homonyme consommée à sa largeur sur les trois motifs ; aucun
  archétype absent de ses lignes ne le lit) ; `ecsRow.DeserAddr` ; G4 123 → 125 largeurs fixes ;
  ratchet des sites de position : site `consumeLowFrequency` (2 appels, niveau 0x10) ;
  `components_hooks_test.go` : la sonde `high-frequency` se joue sous `ti=4` (`archetypeDuHook`),
  `TestProbeHookPassesRegistryTypeIndex` ne la joue plus sous tout `ti`.
- Révision : `grammar.Rev = grammar-2026-10-02.2`, entrée de `rev_chronique.go`, empreinte
  `0528d8d1…`. `killsource.Rev` inchangée (golden à révision constante, `a0a59c83…`, historique
  écrit) ; `types/testdata/shapes.golden` (ligne des révisions) ; fixtures de contrat
  `replay_schema_76_*.json.gz` + `manifest.json` (8 films, identiques hors chaîne `grammarRev`,
  vérifié par `jq -S` après substitution) ; `replay.SchemaVersion` inchangé (76).
- Sonde de recherche : `campagne_l8_research_test.go` (`TestCampagneL8Paquets`, détail record par
  record de paquets nommés).

## 3. Carte de fermeture v2, avant / après, par film

`cmd_fermeture -mode v2 -denominateur-fixe r_comb2_tsv/r_comb2_denominateurs.tsv -paquets`, 20 films,
binaire de `af6e93e23` contre binaire du lot. La carte « avant » est identique à l'octet aux TSV
« après » de L0 (`l0_tsv/carte_apres/`, 7 TSV ; `fermeture_films.tsv` ne diffère que par le pic et la
durée). « Sains » = fermés au sens de L0 (au bit près ET aucune règle de l'écrivain contredite).
Comparaison paquet par paquet (`table_lot.awk`, `net_sains.awk`).

| Film | Build | Sains avant | Sains après | Fermés au bit | Net sains | Gagnés | Perdus | Factices neufs | Utiles sains avant | Utiles sains après | Net utiles | Utiles lus | Fixe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `0797ce72` | HI_1_13_0 | 19152 | 19152 | 19205 -> 19205 | +0 | 0 | 0 | 0 | 159920 | 159920 | +0 | 205685 -> 205685 | 225114 |
| `084a804d` | HI_1_10_0 | 4773 | 4773 | 5235 -> 5236 | +0 | 0 | 0 | 1 | 88036 | 88036 | +0 | 535804 -> 535805 | 780716 |
| `111fa685` | HI_1_10_0 | 4012 | 4012 | 4152 -> 4151 | +0 | 0 | 0 | 0 | 42601 | 42601 | +0 | 245556 -> 245556 | 328525 |
| `11de8353` | HI_1_9_0 | 5612 | 5612 | 5739 -> 5739 | +0 | 0 | 0 | 0 | 65230 | 65230 | +0 | 248287 -> 248293 | 324613 |
| `1c4c63c2` | HI_1_10_0 | 13333 | 13333 | 19376 -> 19379 | +0 | 0 | 0 | 5 | 165404 | 165404 | +0 | 777075 -> 777076 | 1314310 |
| `396cfc92` | HI_1_13_0 | 22819 | 22819 | 22828 -> 22828 | +0 | 0 | 0 | 0 | 166984 | 166984 | +0 | 225495 -> 225495 | 239503 |
| `4f77afc1` | HI_1_13_0 | 22260 | 22260 | 23269 -> 23269 | +0 | 0 | 0 | 1 | 550289 | 550289 | +0 | 738490 -> 738488 | 825839 |
| `50247b26` | version-31 | 139 | 139 | 153 -> 153 | +0 | 0 | 0 | 0 | 274 | 274 | +0 | 319053 -> 319053 | 328128 |
| `51ebbc0f` | HI_1_13_0 | 9759 | 19782 | 9847 -> 19872 | **+10023** | 10023 | 0 | 19 | 58585 | 134423 | **+75838** | 88493 -> 176478 | 223827 |
| `60ae07c4` | HI_1_8_0 | 13802 | 13802 | 13969 -> 13969 | +0 | 0 | 0 | 0 | 82730 | 82730 | +0 | 268352 -> 268352 | 359291 |
| `a349fea8` | version-33 | 420 | 420 | 463 -> 463 | +0 | 0 | 0 | 0 | 3595 | 3595 | +0 | 469451 -> 469451 | 519304 |
| `a521164d` | HI_1_4_1 | 692 | 692 | 701 -> 701 | +0 | 0 | 0 | 0 | 78 | 78 | +0 | 181607 -> 181607 | 198500 |
| `bcb6d393` | HI_1_12_0 | 5830 | 5830 | 5835 -> 5835 | +0 | 0 | 0 | 0 | 35143 | 35143 | +0 | 121628 -> 121628 | 148160 |
| `bf15f7ab` | HI_1_13_0 | 28465 | 28465 | 28476 -> 28476 | +0 | 0 | 0 | 0 | 214974 | 214974 | +0 | 227637 -> 227637 | 232833 |
| `bfecd02b` | HI_1_13_0 | 26380 | 26380 | 26403 -> 26403 | +0 | 0 | 0 | 0 | 226529 | 226529 | +0 | 254963 -> 254963 | 272952 |
| `c75f33b8` | HI_1_13_0 | 22854 | 23669 | 22944 -> 23760 | **+815** | 815 | 0 | 2 | 144430 | 152187 | **+7757** | 155136 -> 159175 | 179123 |
| `d9781168` | HI_1_13_0 | 26210 | 26210 | 26464 -> 26465 | +0 | 0 | 0 | 1 | 180052 | 180052 | +0 | 244872 -> 244872 | 321902 |
| `e5adf7b2` | HI_1_11_0 | 4123 | 4123 | 4285 -> 4285 | +0 | 0 | 0 | 0 | 79457 | 79457 | +0 | 289357 -> 289357 | 383476 |
| `f75e7053` | HI_1_13_0 | 23416 | 23416 | 23423 -> 23423 | +0 | 0 | 0 | 0 | 160042 | 160042 | +0 | 187304 -> 187304 | 192371 |
| `fb1a1a72` | HI_1_13_0 | 22276 | 42035 | 22318 -> 42081 | **+19759** | 19760 | **1** | 5 | 161566 | 312345 | **+150779** | 179254 -> 342709 | 359803 |
| **corpus** | | 276327 | 306924 | 285085 -> 315693 | **+30597** | 30598 | 1 | 34 | 2585919 | 2820293 | **+234374** | 5963499 -> 6218984 | 7758290 |

Mesuré :
- **aucun film en baisse**, ni en paquets sains ni en records utiles sains (critère NET du gate 2) ;
  sains perdus en brut : **1** (`fb1a1a72` 7:92, devenu contredit ; 0 devenu non fermé), instruit au §4.1 ;
- utiles sains perdus en brut : 7 (ce même paquet) ;
- juge des invariants sur les GAGNÉS : 30 598 gagnés, tous sans règle contredite (définition L0) ;
  **34 fermetures factices neuves** (fermées au bit, contredites, non fermées au bit avant) : part de
  gains factices **34 / 30 632 = 0,11 %** ; par règle : masque au-delà de l'archétype 19
  (`51ebbc0f` 10, `fb1a1a72` 5, `c75f33b8` 2, `d9781168` 1, `1c4c63c2` 1), sortie par rejet 10
  (`51ebbc0f` 5, `1c4c63c2` 3, `4f77afc1` 1, `084a804d` 1), ordre de la vue B 4 (`51ebbc0f` 3,
  `1c4c63c2` 1), masque épars non croissant 1 (`51ebbc0f`). Ce sont les « 34 gains contredits » de
  BIS_3, désormais exclus des gains par la définition (D2) ;
- juge sur les PERDUS : le seul perdu contredit « masque au-delà de l'archétype » (§4.1) ;
- `1c4c63c2` : 0 sain gagné ou perdu ; les 8 paquets qui changent y étaient et y restent contredits ;
- indicateurs : corpus, utiles fermés / fixe consolidé **33,3 % → 36,4 %** (variable 43,4 % → 45,3 %) ;
  HI_1_13_0 **65,8 % → 73,5 %** (variable 80,7 % → 81,7 %) ; bloquant « `ti=3 low-frequency` non
  porté » (1 187 paquets avant) disparu ; listes non localisées 48 688 → 48 483 ;
- concordance avec la mesure de recherche : +30 597 sains / +234 374 utiles sains contre +30 596 /
  +234 374 pour R-COMB-2 « seul » (l'écart d'un paquet est `1c4c63c2`, §4.2).

## 4. Pertes instruites, une par une

### 4.1 `fb1a1a72` chunk 7 paquet 92 : sain → contredit (masque au-delà de l'archétype)

Sonde `TestCampagneL8Paquets` (`sonde_avant_l0/`, `sonde_apres/`), mesuré :
- avant : le localisateur strict place la liste à 721 (DELTA du slot 123, `ti=4`) ; 13 records, paquet
  fermé, aucune règle contredite ;
- après : `debutParChaine` remonte la tête à 131. Les records de 131 : NEW slot 257 gén. 0 `ti=3`,
  masque `0xc2548bdaf9c36320` (bits au-delà des deux composants de l'archétype), fin à 233 ; NEW slot
  390 `ti=3` (masque `0x1`, `low-frequency` lu de 295 à 477) ; NEW slot 391 `ti=3` (de 539 à 721) ;
  puis les 13 records d'avant, identiques. La chaîne finit exactement à 721, la vue C ferme au bit
  près, mais le premier NEW contredit `FUN_142e2da44` (aucun bit de masque au-delà de
  `*(desc+0x4320)`) : le paquet est requalifié contredit.

Explication par le jeu (établi pour la règle, mesuré pour la chaîne) : depuis L8, deux NEW
`low-frequency` (390, 391) se traversent, donc une CHAÎNE de tête existe ; `pasDEssai`
(`debut_de_liste.go`) accepte un NEW « traversé sans désynchronisation » sans juger la règle du
masque de l'écrivain (la traversée ignore les bits au-delà de l'archétype, elle ne fait que les
marquer, L0.7), et `debutParChaine` retient le PREMIER candidat : 131, un en-tête lu dans les
données. La vraie tête est vraisemblablement l'en-tête du NEW 390 (≥ 233), supposé. La perte n'est
pas due au lecteur de `ti=3` mais à la règle de tête, qui n'applique pas à la chaîne la règle de
l'écrivain que L0 applique à `debutParFermeture`.

Correctif général mesuré en surcouche (non appliqué, hors périmètre du lot de composant, D-L8-1) :
`pasDEssai` refuse un NEW dont `MasqueNonEcrit != InvariantAucun`. Carte v2 20 films : **0 sain perdu
sur 20 films**, `fb1a1a72` 7:92 retrouvé, et +37 sains au-delà de L8 seul (`4f77afc1` +16,
`fb1a1a72` +7, `084a804d` +5, `11de8353` +2, `60ae07c4` +2, `bfecd02b` +2, `e5adf7b2` +2,
`51ebbc0f` +1 ; +811 utiles sains), dont 6 films où L8 n'agit pas : la règle manquante est antérieure
au lot.

### 4.2 `1c4c63c2` : le « −1 sain » de R-COMB-2 / BIS_3

Reproduit à l'identique sous la marche d'avant L0 (surcouche : `l.Fermee = l.FermeeAuBit`, donc
`debutParFermeture` d'avant L0) et le juge à trois règles (ordre, masque, vue C ;
`ancien_juge.awk`) : `1c4c63c2` 13 540 → 13 539 sains, `fb1a1a72` 22 279 → 42 038, `51ebbc0f`
9 759 → 19 782, `c75f33b8` 22 855 → 23 670 (= R_COMP §3.3). Le paquet perdu est **`24:510`** :
- avant : tête 5201 (NEW slot 443 `ti=0`), fermé, sain ;
- après (marche d'avant L0) : `debutParFermeture` prend 2237, chaîne NEW slot 3888 `ti=42` (masque
  `0x5566105b9cb84bb0`) puis NEW slot 4620 `ti=3` (masque `0x77a1554574880641`, `low-frequency`
  désormais traversé) : les deux masques ont des bits au-delà de l'archétype (`FUN_142e2da44`) ;
  l'ancien premier rang (« ferme au bit ») acceptait cette tête, la carte requalifie le paquet
  contredit.

Sous la marche de L0 (référence du lot), le premier rang de `debutParFermeture` exige `Fermee` (aucune
règle contredite) : 2237 est refusé, 5201 retenu, le paquet reste sain (mesuré : `carte_apres`). La
perte était donc une tête lue sur des NEW que l'écrivain ne peut pas écrire, et L0 l'a corrigée en
général ; il ne reste rien à corriger dans L8 pour ce film.

## 5. Gates (sorties exactes)

Depuis `apps/go-api` du worktree, `GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg-l8`, une
commande `go` à la fois.

| Gate | Sortie |
|---|---|
| `gofmt -l ./internal/games/halo_infinite/film/ ./cmd/` | vide |
| `go vet ./internal/games/halo_infinite/film/...` | rc 0, aucune sortie |
| `go vet -tags=research ./internal/games/halo_infinite/film/...` | rc 0, aucune sortie |
| `go test ./internal/archlint/` | `ok` (32,4 s) — un premier passage rouge : `components_hooks_test.go` 611 lignes pour un plafond de 600, corrigé en déplaçant l'aide `archetypeDuHook` dans `components_frequences_test.go` (600 lignes) |
| G-film `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1 -timeout 30m` | rc 0, **19 paquets `ok`** (premier passage : 4 rouges attendus, goldens de révision — killsource, revision, types, fixtures de contrat —, régénérés par les commandes du dépôt, §2) |
| `golangci-lint run --new-from-rev=af6e93e23 ./internal/games/halo_infinite/film/internal/grammar/` | `0 issues.` |
| Vecteurs du lot (`TestBasseFrequenceSuitSonEcrivain`, `TestHauteFrequenceSeLitParLaTableDeLArchetype`, G4, G6) | verts |
| Mutations (`-overlay`, `mutations.sh`, aucun fichier du worktree modifié) | **10 / 10 ROUGES** (§5.1) |
| Gate 2 — carte v2, 20 films | §3 : aucun film en baisse ; 1 sain perdu en brut, instruit (§4.1) |
| Gate 3 — killsource json, 19 témoins | §5.2 |
| Gate 6 — `replay-equiv` (recette L0) | §5.3 |
| Gate 6 — `replay-corpus-gate` | §5.4 |
| Gate 7 — pas de cuisson en lot | tenu : aucune cuisson du parc ; cuissons de `replay-equiv` et du gate de corpus sur des racines factices du scratchpad |

Gates 4 (performance du bloc de type 1) et 5 (recopie du pilotage) : sans objet (L8 ne lit pas le
bloc de type 1 et ne change pas le pilotage de `decodeFrameParRangs` / `debutDeLaListe`).

### 5.1 Mutations

```
M1 basse-frequence-sans-R5-d-entree : ROUGE (TestBasseFrequenceSuitSonEcrivain)
M2 basse-frequence-compte-sur-5 : ROUGE (TestBasseFrequenceSuitSonEcrivain)
M3 basse-frequence-sans-orientation-d-entree : ROUGE (TestBasseFrequenceSuitSonEcrivain)
M4 basse-frequence-sans-position-de-tete : ROUGE (TestBasseFrequenceSuitSonEcrivain)
M5 basse-frequence-non-portee : ROUGE (TestBasseFrequenceSuitSonEcrivain)
M6 haute-frequence-par-nom-ti3-en-R8 : ROUGE (TestHauteFrequenceSeLitParLaTableDeLArchetype TestG6LesHomonymesSeRoutentParTable TestG4LargeursEntieresSuiventLeCode)
M7 haute-frequence-repli-par-nom : ROUGE (TestHauteFrequenceSeLitParLaTableDeLArchetype TestG6LesHomonymesSeRoutentParTable)
M8 haute-frequence-ti4-en-26-bits : ROUGE (TestHauteFrequenceSeLitParLaTableDeLArchetype TestHooksConsumeSameBitsWithoutHook TestHookedNamesCoversMovedCases TestG6LesHomonymesSeRoutentParTable TestG4LargeursEntieresSuiventLeCode)
M9 table-D86-defaite : ROUGE (TestG6LesHomonymesSeRoutentParTable)
M10 champs-de-frequence-R2-en-R1 : ROUGE (TestBasseFrequenceSuitSonEcrivain TestHauteFrequenceSeLitParLaTableDeLArchetype TestG6LesHomonymesSeRoutentParTable TestG4LargeursEntieresSuiventLeCode)
```

M9 défait D-86 dans une copie de la table (surcouche de `ecs_table_guard_test.go` qui pointe sur la
copie). La base est verte (même `-run`).

### 5.2 Killsource (gate 3)

`cmd/killsource json <film> -carte <carte> -cache <parc>/data/cache -catalogue <map_quant_bounds.json>`
sur les 19 témoins de `config/replay_corpus.toml`, binaire de `af6e93e23` contre binaire du lot :
**16 / 19 identiques à l'octet** ; `51ebbc0f`, `c75f33b8`, `fb1a1a72` : une seule ligne diffère, la
chaîne `calibration` (scores de l'oracle et de la poignée : `51ebbc0f` « score 120 » → « 210 »,
`c75f33b8` oracle 73/71 → 74/72 et « 307 » → « 313 », `fb1a1a72` « 429 » → « 546 » ; largeurs
décidées et `desaccords=0` inchangés). **Aucune mort, valeur ni voie ne change** (= R-COMB-2 §6).
`Result.Calibration` n'est lu que par `cmd/killsource` (`sortie_json.go`, `table.go`, `comparer.go`,
établi par grep) : rien n'est persisté, `killsource.Rev` ne monte pas ; golden régénéré à révision
constante avec son historique. `1c4c63c2` sans carte lisible, non joué (comme L0 et R-COMB-2).

### 5.3 `replay-equiv`

Recette L0 (D-L0-6) : racine factice `scratchpad/L8/repo` (20 films du corpus d'équivalence copiés,
identiques à l'octet au parc par `diff -rq` ; `data/titles/halo_infinite/reference` identique au
parc ; `config/` et références `film/replay/testdata/equivalence` identiques à l'octet à celles du
worktree ; aucune sortie antérieure dans le cache factice). Deux binaires : `af6e93e23` (le lot
retiré par surcouche : `dispatch_item.go` et `rev.go` de la base, `components_frequences.go`
supprimé) et le lot ; `-out-dir` conservé.

- Les références figées datent d'avant L0 (`6d3c1aca4`, J12) : la base sort **20 / 20 différents**
  sur les étapes de L0 (tir continu, états de mouvement, `artifact`), comme L0 l'a mesuré ; le lot
  aussi (`BILAN : 0 identique(s), 20 different(s)` des deux côtés). Le gate du lot est donc la
  comparaison des digests base contre lot, film par film (`re_etapes_divergentes.tsv`) :

| Étapes qui changent (base → lot) | Films |
|---|---|
| `artifact` seul (révisions de calque) | `000d5950`, `01e1f945`, `50247b26`, `51101d1d`, `64e8adfa`, `696a9d7c`, `7344d24f`, `a349fea8`, `a521164d`, `bcb6d393` |
| + `vehicles` | `53ce4390`, `e5adf7b2` |
| + `continuousFire.stats`, `movementStates.stats` | `11de8353`, `d9781168` |
| + `continuousFire.stats`, `movementStates.stats`, `vehicles` | `111fa685`, `1c4c63c2`, `60ae07c4` |
| + `continuousFire.stats`, `movementStates`, `movementStates.stats`, `vehicles` | `084a804d` (12 552 → 12 553 états) |
| + `continuousFire.stats`, `movementStates`, `movementStates.stats`, `killsource` | `9f57c612` (3 257 → 3 302 états), `fb1a1a72` (1 481 → 2 593 états) |

Aucune étape de position, de mort, d'identité, d'objectif ni de tir continu (rafales) ne change ; 55
étapes sur 61 sont identiques sur les 20 films. Instruction, par cuisson d'UN artefact par film et
par révision (`cmd/replay-build --facts`, racine factice, trois films, un processus à la fois ;
`cuit/`) et comparaison clé par clé du document :
- `fb1a1a72`, `9f57c612`, `084a804d` : seules `coverage`, `layers` (révisions) et `stances` changent ;
  `tracks`, morts, effets de kill, objectifs, armes, équipement : identiques ;
- `coverage.continuousFire` : plus de paquets fermés (`fb1a1a72` `closed` 22 276 → 42 035, trous
  26 495 → 6 736 ; `9f57c612` 21 547 → 21 794) — la carte (§3) ; `coverage.stances` : plus de records
  lus (`fb1a1a72` 133 329 → 257 057 ; NEW refusés 112 → 2) ; `coverage.fallbacks` :
  `repli_liaison_par_anticipation` 81 → 56 (`fb1a1a72`), 143 → 141 (`9f57c612`), 920 → 921
  (`084a804d`) ; `repli_debut_de_liste_ferme_au_bit` 112 → 111 (`9f57c612`), 457 → 458 (`084a804d`) ;
  `repli_deadstate_indice_hors_roster` 2 → 1 (`fb1a1a72`) ; `repli_deadstate_hors_bande_bipede`
  5 → 4 (`9f57c612`) ;
- `stances` (intervalles d'états de mouvement) : `fb1a1a72` 534 → 1 090 (sprint 272 → 692,
  `jumpDerived` 252 → 387, `clamber` 10 → 11), `9f57c612` 756 → 764 ; intervalles d'avant absents
  après : `fb1a1a72` 178, dont 162 recouverts par un intervalle de même slot et même genre (recoupés
  par les lectures neuves) et 16 `jumpDerived` sans recouvrement (genre CALCULÉ, pas lu) : 3 décalés
  de quelques trames, 8 au contact d'un `sprint` désormais lu, **5 sans voisin à ±10 trames**
  (slots 515, 531, 533, 582, 620 — instruits un par un au §8.2 b) ; `9f57c612` : 2, tous recouverts ;
- `killsource` (`9f57c612`, `fb1a1a72`) : l'étape hache le `Result` entier ; le document ne change pas
  sur les morts ; le seul champ qui bouge est le diagnostic `Calibration` (§5.2, mesuré par la CLI
  sur `fb1a1a72` ; `9f57c612` n'est pas un des 19 témoins, supposé de même nature) ;
- `vehicles` : l'étape hache l'entrée du balayage (`FilmInputs.Vehicles`) ; `coverage.vehicles` et le
  document de `084a804d` sont inchangés (mesuré) ; pour les 5 autres films (et non 6) : mesuré au
  §8.2 c, seul `DeathStats` (dénominateurs de la marche des morts) change.
- Durées et pics (base → lot, `re_durees.txt`) : semblables à ± 10 % (machine partagée), sauf le pic
  de `1c4c63c2` 1,80 → 2,10 Gio ; L0 avait mesuré 1,79 → 2,11 Gio en passant à cette même base :
  écart non instruit (variance estimée), gate 4 sans objet pour L8.

### 5.4 `replay-corpus-gate`

`CGO_ENABLED=0 go run ./cmd/replay-corpus-gate --reference=base --base=af6e93e23
--parc-root=<scratchpad/L8/parc> --work-root=<scratchpad/L8/gate_work> --json=<scratchpad/L8/l8_corpus_gate.json>`
(19 témoins de `config/replay_corpus.toml`). Mêmes deux écarts que L0, pour les mêmes raisons :
`--base` explicite, `--parc-root` sur une copie (copie de celle de L0 : bases DuckDB, manifestes, films
des 19 témoins identiques à l'octet au parc par `diff -rq`). Un premier passage a été interrompu
(poste en veille, tâche de fond arrêtée à sa limite) au 17e témoin : le worktree de base resté
enregistré (`scratchpad/L8/gate_work/base-worktree`, aucune jonction : `dir /AL /S` vide, arbre
propre) a été retiré par `git worktree remove` SANS `--force` ; le verrou orphelin de la copie du parc
effacé ; le second passage a relu le cache des artefacts de base du premier.

Résultat : **rc 1** ; banc de vérité **ok sur 19 / 19** (aucun oracle — kills, morts, assistances,
équipes, vies — ni aucune classe de violation ne bouge ; `[gain] P-1` sur `fb1a1a72` 22 276 → 42 035,
`c75f33b8` 22 854 → 23 669, `51ebbc0f` 9 759 → 19 782 ; `R-1` en information seulement) ; statut
**PERTE sur 8 témoins** au sens de l'outil :

| Témoin | Gains / pertes / chang. | Ce qui « perd » |
|---|---|---|
| `fb1a1a72` | 100 / 64 / 4 | `stances/duree-totale` 19 031 → 16 220 (et 62 lignes par slot du même bloc) |
| `c75f33b8` | 32 / 5 / 3 | `stances/duree-totale` 12 291 → 11 962 |
| `11de8353` | 2 / 6 / 0 | `continuousFire.holesOpenViewB` 249 → 261, `reached` 15 739 → 15 727, `holesBlockBC` 18 → 21, `holesKind` 189 → 192, `stances.dropped` 8 → 14, `forgottenBindings` 1 700 → 1 713 |
| `084a804d` | 5 / 4 / 0 | `holesNotClosing` +1, `forgottenBindings` +2, `refusedNews` +1, `refusedNewFalseReads` +1 |
| `111fa685` | 3 / 5 / 0 | `holesKind` +1, `holesUnlocated` +1, `eventPacketsUnlocated` +1, `refusedNews` +1, `refusedNewFalseReads` +1 |
| `60ae07c4` | 1 / 3 / 0 | `holesBlockBC` +1, `refusedNews` +2, `refusedNewFalseReads` +2 |
| `d9781168` | 3 / 3 / 0 | `holesNotClosing` +1, `refusedNews` +2, `refusedNewUndecided` +2 |
| `4f77afc1` | 1 / 3 / 0 | `stances.records` −1, `refusedNews` +1, `refusedNewUndecided` +1 |

Instruction :
- **durée des stances** (`fb1a1a72`, `c75f33b8`) — mesuré sur les artefacts (§5.3) : `fb1a1a72` lit
  2 593 états au lieu de 1 481 ; les intervalles passent de 534 à 1 090 et se recoupent. Exemple du
  slot 533 : sprint `827-1191` et `1240-1599` (36 s d'un seul tenant chacun) avant ; après, douze
  intervalles courts (`806-823 827-868 891-900 … 1150-1191 1240-1274 1579-1599 1607-1610`). Par
  trame, sprint 16 552 → 12 840 (8 187 en commun, 4 653 neuves, 8 365 retirées). Les états neufs sont
  lus sur les paquets que L8 ferme ; la durée baisse parce que des lectures nouvelles interrompent des
  intervalles que la lecture clairsemée prolongeait d'une lecture à la suivante (mécanisme établi par
  l'exemple ; que la nouvelle durée soit la vraie est estimé, aucun oracle du banc ne couvre ce bloc) ;
- **compteurs de couverture** des six autres témoins (+1 à +13) : sur ces films la carte v2 ne change
  aucun paquet sain (§3). Mesuré sur la carte : des 1 187 paquets que « `ti=3 low-frequency` non
  porté » arrêtait avant, 953 ferment (sains), 1 ferme au bit contredit, les autres s'arrêtent sur
  une autre cause, dont 8 « vue B : fin de payload » sur les builds anciens (`111fa685` 23:14,
  `60ae07c4` 13:2340, 14:514, 14:2120, `11de8353` 29:1150, 1152, 1166, 1168). La sonde L8 sur quatre
  d'entre eux (`sonde_fin_payload/`) : le record `ti=3` lu est chaque fois un NEW au masque
  impossible (bits au-delà de l'archétype, `FUN_142e2da44`), c'est-à-dire un en-tête lu dans des
  données déjà désalignées ; avant L8 la marche s'arrêtait sur le composant non porté, elle continue
  désormais jusqu'au bout du paquet (la traversée marque le masque, L0.7, sans s'arrêter). Sur
  `11de8353`, la carte voit 34 paquets changer de cause d'arrêt (depuis « vue C hors cadre » : sortie
  par rejet 14, composant `ti=34` non porté 9, `ti=40` non porté 2, fin de payload 2, bloc 0xbc 1 ;
  depuis « `ti=3 low-frequency` » : fin de payload 4, rejet 1 ; un rejet devenu vue C hors cadre),
  tous non fermés avant comme après : c'est l'ordre de grandeur des compteurs de la cuisson (`holesOpenViewB` +12, `reached` −12),
  rafales du tir continu inchangées (`continuousFire` identique au `replay-equiv`). Les comptes de NEW
  refusés (+1, +2) : instruits un par un au §8.2 a (7 refus neufs, tous sur des en-têtes que
  l'écrivain ne peut pas écrire ; D-L8-1 et D-L8-8).

## 6. Écarts

- Révision `grammar-2026-10-02.2` et non un suffixe propre au lot : `revision.ParserRevision` n'admet
  que `AAAA-MM-JJ[.N]`, rangs consécutifs (D-L8-2). Collision attendue avec les autres lots de la
  vague partis de la même base ; l'intégrateur renumérote.
- Statut `partiel` des deux lignes `high-frequency` dans `ecs_table.tsv` (et non `porte`) : imposé par
  G1, qui classe par nom un `case` capable de rendre « non porté » ; la raison est écrite dans la
  colonne `notes`.
- Une perte saine brute sur `fb1a1a72` (§4.1), expliquée par le jeu ; le correctif général (D-L8-1)
  n'est pas appliqué dans ce lot (marche, hors périmètre ; mesuré en surcouche).
- `archlint` rouge au premier passage (plafond de taille de `components_hooks_test.go`), corrigé.
- Commande hors liste jouée par erreur : un `python3 -` vide (aucun script, aucune sortie), sans
  effet ; aucun Python écrit. Une redirection vers `/tmp_none` (racine du disque) tapée par erreur a
  été refusée par le système (« Permission denied ») : rien n'a été créé.
- `replay-equiv` : 5 intervalles `jumpDerived` de `fb1a1a72` disparus sans voisin, instruits un par
  un au §8.2 b (trois faux positifs de la base, deux profils de saut sortis au bord de la fenêtre de
  la dérivation, D-L8-10).
- `replay-corpus-gate` sort rc 1 (PERTE sur 8 témoins, §5.4) : le plan exige « sans perte ». Le rc 1
  est désormais instruit en entier : banc de vérité ok sur 19 / 19 ; durée des stances recoupée par
  des lectures neuves (D-L8-9) ; compteurs de couverture de paquets non fermés (D-L8-8) ; NEW refusés
  instruits un par un (§8.2 a : 7 refus neufs, tous sur des en-têtes que l'écrivain ne peut pas
  écrire, aucune lecture fausse créée) ; étape `vehicles` mesurée (§8.2 c : dénominateurs seulement).
  **Décision sur le rc 1 : NON CONSIGNÉE.** Elle appartient au pilote ou à l'utilisateur ; l'exécutant
  des corrections (2026-10-03) n'a reçu aucune décision datée et n'en écrit pas à leur place.
  Proposition de l'exécutant, sur le précédent L0 (rc 1 admis après instruction) : admettre le rc 1.
  Ligne à compléter par le décideur : « Décision du [pilote | utilisateur], [date] : … ».

## 7. Découvertes

- **D-L8-1** : `pasDEssai` (`debut_de_liste.go`, tête de liste par chaîne) accepte un NEW dont le
  masque contredit `FUN_142e2da44` ; `debutParFermeture` le refuse depuis L0. Correctif mesuré en
  surcouche (§4.1) : 0 perte saine sur 20 films, +37 sains et +811 utiles sains au-delà de L8, sur 8
  films dont 6 où L8 n'agit pas. Lot de marche à part (gate 5 compris : la recopie
  `campagne_marche_research_test.go` / `r_nais_marche_research_test.go` le cas échéant).
- **D-L8-2** : le format des révisions n'admet aucun suffixe de lot ; des lots parallèles partis de la
  même base prennent tous le même rang.
- **D-L8-3** : la remarque « KEYFRAME : 26 bits (distinct) » de `ti=4 i0` (table ECS) est contredite
  par le jeu (§1.4) ; elle venait vraisemblablement de la confusion des deux tables (supposé).
- **D-L8-4** : `FUN_140c5f938` (orientation) bifurque sur `DAT_145121140` (`== 1` → `FUN_142e29bac`) ;
  le Go ne porte que la branche `!= 1`, comme à tous les sites existants (`decodeObjectForwardAndUp`).
  Non instruit.
- **D-L8-5** : le compte de `low-frequency` (`R(6)`, jusqu'à 63) n'est borné ni par le lecteur ni par
  l'écrivain, alors que la place des entrées (`+0x8` à `+0x508`, 0x28 octets chacune) en tient 32 :
  au-delà, le lecteur du jeu écrirait sur `+0x508` (calcul sur les offsets lus ; aucun film mesuré n'a
  été balayé pour ce compte). Non instruit.
- **D-L8-6** : la marche de recherche `cmMarcher` rend des records sans `HeaderBit` (toujours 0) ; la
  sonde L8 en hérite. Sans effet sur les mesures.
- **D-L8-7** : `TestProbeHookPassesRegistryTypeIndex` posait que tout composant sondé se lit sous tout
  `ti` (« câbler un numéro d'archétype serait faux par avance ») ; pour un homonyme de grammaire, c'est
  l'inverse que le jeu fait. La règle reste vraie pour les noms à table unique.
- **D-L8-8** : la traversée ne s'arrête pas sur un masque que l'écrivain ne peut pas écrire (L0.7 ne
  fait que le marquer) ; un composant porté de plus prolonge donc la lecture des paquets déjà
  désalignés (8 paquets « fin de payload » neufs sur HI_1_8_0 à HI_1_10_0, §5.4) et déplace les
  compteurs de la marche de cuisson sur des paquets non fermés. Les comptes de NEW refusés (+1, +2
  sur cinq témoins du gate de corpus) ne sont pas instruits un par un. Arrêter la traversée sur un tel
  masque relève du même lot que D-L8-1.
- **D-L8-9** : `replay-corpus-gate` classe en PERTE la baisse de `stances/duree-totale` quand des
  lectures neuves recoupent les intervalles (`fb1a1a72` 19 031 → 16 220, `c75f33b8` 12 291 → 11 962) ;
  aucune mesure du banc de vérité ne couvre ce bloc (sprint à vitesse constante, oracle physique
  connu, non branché sur le banc). Tout lot qui fait lire plus de paquets à événements sortira le gate
  de corpus en PERTE sur ce bloc par construction, comme D-L0-4 pour P-1.
- **D-L8-10** (corrections du contrôle) : la dérivation `jumpDerived` dépend de la densité des
  lectures de vitesse : un silence de plus de 250 ms compte nul dans l'intégrale, et un échantillonnage
  plus dense déplace la hauteur intégrée de quelques centimètres. Sur `fb1a1a72`, deux épisodes au
  profil de saut (slots 533 et 620) sortent de la fenêtre 0,85 m ± 10 % quand le lot densifie leurs
  lectures (1,0328 m et 0,7635 m) ; trois faux positifs de la base en sortent aussi (§8.2 b). Non
  instruit au-delà ; relève de la dérivation, pas de la grammaire.
- **D-L8-11** (corrections du contrôle) : le NEW `slot 2048 gén. 2 ti=10 masque 0` de `d9781168` est
  refusé quatre fois au chunk 25 (25:788, 25:1728, 25:2122 dans la base, 25:506 de plus au lot),
  chaque fois premier record lu au début de liste retenu : une même lecture fausse récurrente
  (masque nul, aucune règle de l'écrivain contredite ; l'image-clé suivante ne porte pas ce slot).
  Par quel rang de `debutDeLaListe` ces débuts sont pris : non mesuré. Non instruit.

## 8. Corrections du contrôle indépendant (2026-10-03)

Le contrôle (`feat/cg-l8` `9edb99121`, base `af6e93e23`, rapport du 2026-10-03) a confirmé sur pièces
le critère du jeu, les chiffres par film, les pertes, killsource et les gates de code, et demandé cinq
corrections. Instruments et sorties : `scratchpad/L8/` (`mutations_ctl.*`, `refus/`, `veh/`,
`carte_ctl/`, `gfilm_ctl.txt`). Aucun instrument n'est versionné : tous jouent par `-overlay`
(fichiers du worktree intacts).

| # | Correction | Statut |
|---|---|---|
| C1 | vecteurs : les deux branches de la porte d'orientation, en tête et en entrée | [x] |
| C2a | NEW refusés +1 / +2 sur `084a804d`, `111fa685`, `60ae07c4`, `d9781168`, `4f77afc1` | [x] instruits un par un (§8.2) |
| C2b | 5 intervalles `jumpDerived` de `fb1a1a72` sans voisin | [x] instruits un par un (§8.2) |
| C2c | étape `vehicles` de `replay-equiv` sur les films non cuits | [x] mesurée (§8.2) |
| C2d | décision explicite du pilote ou de l'utilisateur sur le rc 1 | [!] à consigner par le pilote (§6) : l'exécutant des corrections n'a reçu aucune décision datée |
| C3 | règle 17 : histoire datée et comptes du jour hors du code | [x] |
| C4 | commentaire faux de `ProbeHighFrequency` | [x] |
| C5 | entrée `.ai/thought_log.md` | [x] |

### 8.1 C1 — la porte de `FUN_140c5fa84` jouée des deux côtés

`FUN_140c5f938` mode 0 → `FUN_140c5fa84` : `R(1)`, puis `R(19)` si la porte vaut 0, puis `R(8)`. Les
vecteurs n'écrivaient la tête qu'AVEC direction et les entrées que SANS. `ecrireBasseFrequence` prend
désormais la branche de la tête, chaque entrée porte la sienne (`direction`) ; six cas : liste vide
tête avec / sans direction, drapeaux 0 à 4 (dont deux entrées à orientation avec direction et une
sans) sous les deux têtes, une entrée seule avec direction, 63 entrées aux directions alternées par
huit. Mutations du contrôle rejouées (`mutations_ctl.sh`, `-overlay`,
`-run TestBasseFrequenceSuitSonEcrivain`) :

```
X17 orientation d'entrée R(1)+R(8)               : ROUGE — « drapeaux 0 a 4, tete avec direction : le lecteur s arrete au bit 380, l ecrivain a ecrit 570 bits »
X18 orientation de tête R(1)+R(19)+R(8) toujours : ROUGE — « liste vide, tete sans direction : le lecteur s arrete au bit 127, l ecrivain a ecrit 108 bits »
X17 sur les vecteurs de 9edb99121                : ok (survivant, comme mesuré par le contrôle)
X18 sur les vecteurs de 9edb99121                : ok (survivant)
```

### 8.2 C2 — l'instruction restante du gate de corpus

**(a) NEW refusés.** Instrument : la marche de PRODUCTION (`ScanMarcheDesTrames`, contexte
`ContexteDeFilm`) avec un journal de chaque refus compté par l'observation de production (chunk,
paquet, phase, record refusé, masque, règle de l'écrivain, archétype du vivant, verdict de l'image-clé
suivante), le détail record par record de paquets nommés, le compte de records par paquet et
l'historique de liaison de slots nommés (`refus/ov/` ; base = fichiers de `grammar` de `af6e93e23` par
surcouche). Les totaux de l'instrument sont ceux du gate, base et lot, au compteur près
(`refusedNews`, `refusedNewFalseReads`, `refusedNewUndecided`, `forgottenBindings`, `records`,
`eventPacketsUnlocated`, cinq films) : mesuré. Aucun refus de la base ne disparaît ; les refus neufs
sont exactement :

| Film | Paquet | NEW refusé (slot, `ti`, masque) | Règle de l'écrivain contredite | Vivant | Verdict | Cause, lien avec `ti=3` |
|---|---|---|---|---|---|---|
| `084a804d` | 11:872 | 91, `ti=3`, `0xf3e082ebef820dbe` | masque au-delà de l'archétype | `ti=17` | lecture fausse | base : liste NON LOCALISÉE ; lot : `high-frequency` de `ti=3` lu sur 26 bits (et non 8) ferme une chaîne [NEW 3933 `ti=42` au masque impossible, DELTA bipède 637, NEW 91 `ti=3`] : tête par chaîne sur un NEW contredit (D-L8-1). Le DELTA 637 est le `stances.records` +1 du film |
| `111fa685` | 23:14 | 26, `ti=3`, `0xf76081ecc010904d` | masque au-delà de l'archétype | `ti=6` | lecture fausse | mêmes 31 records base et lot ; le dernier, un NEW `ti=3` après des DELTA, DÉSYNCHRONISAIT sur `low-frequency` non porté (non compté) ; il se traverse désormais et se compte refusé |
| `60ae07c4` | 14:514 | 122, `ti=3`, `0xf3e0836fc690907d` | idem | `ti=45` | lecture fausse | idem (9 records identiques) |
| `60ae07c4` | 14:2120 | 90, `ti=3`, `0x4180098001` | idem | `ti=17` | lecture fausse | idem (12 records identiques) |
| `d9781168` | 25:506 | 2048, `ti=10`, `0x0` | aucune | `ti=42` | indécis (slot absent de l'image-clé) | base : non localisée ; lot : chaîne [NEW 2048 `ti=10`, NEW 7308 `ti=3` au masque `0x4008000020020001`, au-delà de l'archétype] fermée par `low-frequency` (D-L8-1). Le même NEW 2048 `ti=10` est déjà refusé trois fois dans la base (25:788, 25:1728, 25:2122) |
| `d9781168` | 25:788 | 7308, `ti=5`, `0x748a08ae04c80000` | masque au-delà de l'archétype | `ti=3` | indécis | cascade de 25:506 : le NEW 7308 `ti=3` (faux) y a lié le slot, le NEW 7308 `ti=5` (faux aussi, lié dans la base) est donc refusé ; l'image-clé du chunk 26 ne porte pas 7308 et l'oublie dans les deux arbres |
| `4f77afc1` | 27:1128 | 3892, `ti=38`, `0x200018011000000` | masque au-delà de l'archétype | `ti=47` | indécis | mêmes 18 records base et lot ; le vivant `ti=47` vient de 27:1120 : base non localisée, lot localisée par une chaîne [NEW 3892 `ti=47`, NEW 262 `ti=35`, NEW 7536 `ti=3` au masque au-delà de l'archétype] fermée par `low-frequency` + `high-frequency` (D-L8-1) |

Tous les refus neufs portent sur des en-têtes que l'écrivain ne peut pas écrire (masque au-delà de
l'archétype, `FUN_142e2da44`) ou sur le NEW `ti=10` déjà refusé trois fois sur ce même slot dans la
base ; quatre sont confirmés lecture fausse par l'image-clé suivante, trois indécis (slot absent de
l'image-clé). Le portage de `ti=3` ne crée aucune lecture fausse : il (i) convertit en refus compté
une désynchronisation qui ne l'était pas (3 paquets, même lecture fausse dans les deux arbres) ;
(ii) fait fermer au bit des chaînes de tête dont un NEW contredit l'écrivain, que `pasDEssai` accepte
(D-L8-1 : 3 paquets, plus la cascade de `d9781168`). Le correctif général D-L8-1 (déjà dans L2,
`pasDEssai` NEW et delta) retire (ii) ; (i) relève de D-L8-8 (la traversée ne s'arrête pas sur un
masque impossible).

Mesuré au passage, les autres compteurs de ces témoins : `stances.records` de `084a804d` +1 (DELTA 637
de 11:872, sous une tête contredite) ; `4f77afc1` −1 = −2 (25:748 : la base localisait sur une chaîne
NEW `ti=38` au masque épars non croissant, DELTA 755 puis 633 hors ordre, DEL, NEW `ti=3` 5252 au
masque impossible lu avec `high-frequency` sur 8 bits ; le lot, qui le lit sur 26 bits, ne la ferme
plus) +1 (NEW bipède 262 de 27:1120, sous la tête D-L8-1). `111fa685` `eventPacketsUnlocated` +1 =
20:306 : la base localisait sur une chaîne [NEW `ti=9`, DEL, NEW `ti=4` au masque impossible, NEW
`ti=3` 5754 au masque impossible lu sur 8 bits], le lot ne la ferme plus. `60ae07c4` : deux paquets
changent d'un record (2:1820 14 → 13, 14:760 9 → 10), sans bipède. Les lectures perdues sont des
lectures que l'écrivain contredit ; les deux lectures de bipède gagnées (637, 262) sont sous une tête
D-L8-1, que L2 retire.

**(b) Les 5 intervalles `jumpDerived` de `fb1a1a72` sans voisin.** `jumpDerived` est CALCULÉ
(`movement_states_jump.go` : intégrale de la vitesse verticale tenue, lecture bornée à 250 ms,
fenêtre 0,85 m ± 10 % = [0,765 ; 0,935]). Instrument : échantillons et épisodes des cinq slots, base
et lot (`refus/lot|base_out/fb1a1a72.sauts`, origine des trames recalée sur les intervalles de la
base). Chaque épisode existe encore dans le lot (même montée) ; c'est sa HAUTEUR qui sort de la
fenêtre :

| Slot | Trames | Base : h, échantillons, silences > 250 ms | Lot : h, échantillons | vz max | Lecture |
|---|---|---|---|---|---|
| 515 | 156-167 | 0,8115 m, 19, un silence de 316 ms compté nul | 1,1788 m, 40, aucun | 1,89 m/s | la base n'était dans la fenêtre que par le silence ; profil plat à ~1,2 m/s pendant 1 s, pas celui d'un saut (vz max 3,4 m/s, montée 0,47 s, oracle `dad793c7`) : faux positif de la base (estimé) |
| 531 | 926-938 | 0,9159 m, 13, un silence de 885 ms | 1,5565 m, 40, aucun | 4,40 m/s | idem : la base n'y était que par le silence ; 1,56 m n'est pas un saut au sol (estimé) |
| 533 | 1532-1537 | 0,9036 m, 15, aucun | 1,0328 m, 32 | 3,06 m/s | échantillonnage deux fois plus dense : l'intégrale passe au-dessus de la fenêtre ; profil de saut (montée 0,55 s) ; vrai saut ou non : indéterminé |
| 582 | 3374-3381 | 0,7664 m, 12, aucun | 0,7522 m, 15 | 1,41 m/s | à 0,0014 m du bord dans la base ; vz max 1,4 m/s, pas un profil de saut : faux positif de la base (estimé) |
| 620 | 5530-5534 | 0,7788 m, 21, aucun | 0,7635 m, 25 | 3,20 m/s | profil de saut ; sort de la fenêtre de 0,0015 m : vraisemblablement un vrai saut perdu au bord de la fenêtre (estimé) |

Mécanisme établi (mesuré) : les lectures neuves remplissent des silences que l'intégrale comptait
nuls (515, 531) ou affinent l'échantillonnage (533, 582, 620). Trois des cinq étaient des faux
positifs de la base (estimé par le profil) ; deux (533, 620) ont un profil de saut et sortent d'une
fenêtre que la dérivation fixe (0,85 m ± 10 %). Sur le film, `jumpDerived` passe de 252 à 387
(§5.3) : le genre dérivé gagne en net. La sensibilité au bord de fenêtre est celle de la dérivation
(une heuristique physique, D-L8-10), pas du lecteur de `ti=3`.

**(c) L'étape `vehicles` de `replay-equiv`.** Elle change sur SIX films (`53ce4390`, `e5adf7b2`,
`111fa685`, `1c4c63c2`, `60ae07c4`, et `084a804d`, déjà cuit) : cinq non cuits, et non six comme
écrit au §5.3. Instrument : `replay-equiv` construit base et lot avec une surcouche de `observe.go`
qui hache, à l'étape `vehicles`, chaque champ de `VehicleScan` séparément (`veh/`, racine factice
`scratchpad/L8/repo`, un film par processus, un processus à la fois ; décodage forcé par l'outil).
Les digests `vehicles` des deux binaires sont identiques à l'octet à ceux de l'exécutant
(`re_avant_tsv`, `re_apres_tsv`) sur les six films. Sur les six, UN SEUL champ change : `DeathStats`
(dénominateurs de la marche des morts, par archétype) ; `Keyframes`, `Creations`, `Stats`,
`Positions`, `Events`, `Aims`, `Deaths`, `Occupancy` identiques. Feuilles changées (mesuré) :

| Film | `DeathStats` |
|---|---|
| `084a804d` | `Records[6]` 118 → 119 ; `CleanRecords[0]` 712 → 711, `[3]` 2 → 3, `[6]` 118 → 119 |
| `53ce4390` | `CleanRecords[3]` 4 → 5 |
| `e5adf7b2` | `Records[0]` 16 096 → 16 097 ; `CleanRecords[0]` 987 → 988 |
| `111fa685` | `Records[0]` 16 164 → 16 165 ; `CleanRecords[3]` absent → 3 |
| `60ae07c4` | `Records[0]` 45 965 → 45 968, `[11]` 35 → 34, `[31]` 4 → 5, `[52]` 44 → 43 ; `CleanRecords[0]` 1 627 → 1 628, `[11]` 9 → 8, `[3]` 16 → 22 |
| `1c4c63c2` | `Records[0]` 57 459 → 57 458, `[32]` 12 → 13 ; `CleanRecords[0]` 560 → 559, `[3]` 3 → 8, `[32]` 12 → 13 |

Aucun véhicule, aucune mort ni occupation ne change : la supposition du §5.3 est vérifiée.

### 8.3 C3 et C4 — règle 17

- `ecs_widths_guard_test.go` : le paragraphe daté du lot est retiré du code ; la phrase de contrat
  existante (« les comptes GELES des deux catégories ») reste. Histoire du lot, déplacée ici :
  123 → 125 largeurs FIXES, 66 gardées inchangées ; deux lignes entrent par le haut, les deux tables
  de `high-frequency` : `ti=3 i1` (`FUN_142ed4880`, 26) et `ti=4 i0` (`FUN_14076d034`, 8, dont la
  colonne portait « 8 (frame) ») ; `ti=3 i0 low-frequency` a une largeur gardée par son compte
  d'entrées (« variable ») et reste hors des deux comptes.
- `components_frequences.go` et `ecs_dispatch_table_guard_test.go` : « sur les 326 noms des
  registres du corpus » remplacé par le contrat (un seul nom de composant a deux tables de grammaire ;
  G6 exige que les homonymes de la table soient exactement la liste). Compte du jour, tel que
  l'exécutant du lot l'a mesuré : 326 noms dans les registres du corpus
  (`r_comp_tsv/r_comp_registres.tsv`, 21 registres, 9 builds).
- `components_probe.go` : `ProbeHighFrequency` « R(8) en variante FRAME » (contredit par D-L8-3)
  devient « R(8), delta et image-clé (FUN_14076d034) ».

Ces changements ne touchent que des commentaires du code de production : l'empreinte de grammaire
est inchangée (`TestGrammarRevSuitLaGrammaire` vert), la révision reste `grammar-2026-10-02.2`.

### 8.4 Gates rejoués (tête corrigée)

| Gate | Sortie |
|---|---|
| `gofmt -l ./internal/games/halo_infinite/film/ ./cmd/` | vide |
| `go vet ./internal/games/halo_infinite/film/...` / `-tags=research` | rc 0 / rc 0 |
| `go test ./internal/archlint/` | `ok` (58,5 s) |
| G-film (`film/...`, `replaybuild/...`, `sync/killcollector/...`, `-count=1 -timeout 30m`) | rc 0, 19 paquets `ok` |
| `golangci-lint run --new-from-rev=af6e93e23` (grammar) | `0 issues.` |
| Carte v2 (`-mode v2 -denominateur-fixe … -paquets`) sur `fb1a1a72`, `51ebbc0f`, `c75f33b8`, `084a804d`, `4f77afc1`, `d9781168` | `fermeture_paquets.tsv` identique, film par film, à la carte « après » du lot (sains 42 035, 19 782, 23 669, 4 773, 22 260, 26 210) |
| killsource | non rejoué : aucun code de marche ne change (commentaires seulement) |

Verdict après corrections : **[x] retenu**, gate 2 tenu (la carte ne change pas), rc 1 du gate de
corpus instruit en entier ; la décision de l'admettre appartient au pilote ou à l'utilisateur (§6).

## 9. Revue adverse de la vague 1 (2026-10-03) : `low-frequency` en image-clé

**Constat (majeur), vérifié sur pièces.** Dans un état complet d'image-clé, `low-frequency` était lu
aux largeurs du delta, alors que le jeu le lit sous une portée que la marche du dépôt ne pose pas.

- Ghidra (HTTP 127.0.0.1:8089, lecture seule ; extraits dans `vague1_tsv/revue_ghidra/`) :
  - `disassemble_function 0x142e2c690` : `142e2c6b0 MOV ESI,0x1` ; `142e2c6b8 MOV byte ptr
    [0x144e61ea0],SIL` à l'entrée ; `142e2c76a MOV byte ptr [0x144e61ea0],0x0` à la sortie commune
    (`142e2c765`, atteinte par `142e2c762` et par `142e2c890`) ; `142e2c7c9 CALL qword ptr [RAX +
    0x28]` (le lecteur du composant) entre les deux ;
  - chaîne : `FUN_142e2bfd0` (`142e2c646 CALL 0x1428e2b68`) -> `FUN_1428e2b68` (`1428e2c4d CALL
    0x142e2c690`) ;
  - `decompile 0x14076f91c` : rend 1 si `DAT_144e61ea0 != 0` ou `DAT_145121140 == 1` ;
  - `disassemble 0x14076e494` : `CALL 0x14076f91c ; TEST AL,AL ; JNZ 0x14076e4dc` -> `CALL
    0x1411b259c` ; `disassemble 0x1411b259c` : `MOV R9D,0x60 ; CALL 0x1406d676c` = R(96) ;
  - `FUN_142ed4aec` appelle `FUN_1424e0e38` (`142ed4b1f`, `142ed4e7f`), qui appelle `FUN_14076e494`
    (`1424e0e47`) : la tête et chaque entrée à position sont touchées ;
  - `get_xrefs_to 0x144e61ea0` : écrivains `FUN_142e2bfd0`, `FUN_142e2c690`, `FUN_142e2d08c`,
    `FUN_142e2d6d4`, `FUN_142e309b4`, `FUN_142e30b9c`, `FUN_142e31a0c`, `FUN_142e31bf8` ; aucun dans
    `FUN_14076cb60` (boucle delta) : la lecture du lot reste juste en record à masque.
- Déjà établi par la campagne (`R_VEH.md` §1.3) : « dans une image-clé, l'état par défaut ET les
  composants se lisent sous la portée ». Côté Go, `PorteeBaseline` est faux et
  `walkKeyframeFullState` ne le pose pas : `lireE494` passait par la branche quantifiée.

**Correction retenue (générale, sans réglage) : `low-frequency` n'est pas porté dans un état
complet.** `consumeLowFrequency` rend faux, sans lire un bit, sous `Lecteur.etatComplet`, et le
`case` de `dispatch_item.go` rend sa valeur (la traversée s'arrête, comme avant le lot, et comme L4a
pour les composants `ti=40` hors `i37`) ; la lecture en record à masque ne change pas. Contrat écrit
dans l'en-tête de `components_frequences.go` ; `ecs_table.tsv` : statut `porte` -> `partiel` (G1,
même statut que `ti=40 i30` et `i33`) ; vecteur `TestBasseFrequenceNonPorteeDansUnEtatComplet`
(même composant lu en entier hors état complet, 0 bit et « non porté » en état complet ; mutation
sans la garde : ROUGE, 227 bits lus). Première écriture de la garde dans le `case` : refusée par le
cliquet de longueur de `consumeItemAndTacmapComponent` (142 lignes contre 139 figées), d'où la garde
dans le lecteur (138 lignes).

**Correction proposée par la revue et NON appliquée** : poser la portée sur toute la boucle de
composants de l'état complet (`PorteeBaseline` vrai pendant `traverseComponentLoop` dans
`walkKeyframeFullState`). C'est la lecture du jeu, mais elle change TOUTES les positions lues en
image-clé, tous archétypes : c'est le lot LK (portée + chemin `i0` de l'écrivain), mis de côté par
l'utilisateur le 2026-10-02 (« lecture d'image-clé non confirmée », §3 du plan ; D14). Elle reste à
mesurer sur le parc (gate 2 et carte d'image-clé) dans ce lot-là. La sonde A/B de la revue
(`scratchpad/revue2-jeu/`, `KeyframeClosure` sur 7 bobines) donne `fb1a1a72` ti=3 0/26 -> 19/26 et
aucune baisse : indication pour LK, pas une preuve de gate.

**Mesures après correction** (tête corrigée contre tête intégrée `a552c43f5`) : carte v2 20 films,
14 TSV sur 15 identiques à l'octet (`fermeture_paquets.tsv` compris), seul `fermeture_films.tsv`
diffère par le pic mémoire et la durée : le gate 2 de la vague est inchangé. Killsource 19 témoins :
JSON identique à l'octet. `keyframe_closure.golden` : aucun compte ne bouge, `ti=3` de `bcb6d393`
(0/1) et `fb1a1a72` (0/26) retrouve son bloquant `i0 low-frequency`. Fixtures de contrat :
identiques hors chaînes de révision. `replay-equiv` 20 films : 60 étapes sur 61 identiques, seule
`artifact` (chaînes de révision) diffère.
