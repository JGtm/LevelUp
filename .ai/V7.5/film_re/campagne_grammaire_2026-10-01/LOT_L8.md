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
| Gates 1, 2, 3, 6, 7 | voir §5 | 1, 2, 3, 7 tenus ; 6 : `replay-equiv` instruit, `replay-corpus-gate` rc 1 (banc 19 / 19 ok, PERTE sur 8 témoins : durée des stances et compteurs de couverture, instruits §5.4 sauf les NEW refusés) |

Verdict : **[x] retenu** (voir §5 et §6 pour les écarts).

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
  (slots 515, 531, 533, 582, 620 — non instruits un par un) ; `9f57c612` : 2, tous recouverts ;
- `killsource` (`9f57c612`, `fb1a1a72`) : l'étape hache le `Result` entier ; le document ne change pas
  sur les morts ; le seul champ qui bouge est le diagnostic `Calibration` (§5.2, mesuré par la CLI
  sur `fb1a1a72` ; `9f57c612` n'est pas un des 19 témoins, supposé de même nature) ;
- `vehicles` : l'étape hache l'entrée du balayage (`FilmInputs.Vehicles`) ; `coverage.vehicles` et le
  document de `084a804d` sont inchangés (mesuré) ; pour les 6 autres films, non cuit : supposé de
  même nature (compteurs internes du balayage qui lit désormais les records `ti=3`).
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
  refusés (+1, +2) : supposés de même nature (un NEW `ti=3` qui se désynchronisait se traverse et entre
  dans les comptes), non instruits un par un (D-L8-8).

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
- `replay-equiv` : 5 intervalles `jumpDerived` de `fb1a1a72` disparus sans voisin, non instruits un
  par un (§5.3).
- `replay-corpus-gate` sort rc 1 (PERTE sur 8 témoins, §5.4) : le plan exige « sans perte » ; le lot
  est retenu parce que le banc de vérité est ok sur 19 / 19 et que les pertes sont la durée des
  stances recoupée par des lectures neuves (D-L8-9) et des compteurs de couverture de paquets non
  fermés (D-L8-8) ; les comptes de NEW refusés restent non instruits un par un. À trancher par le
  pilote ou l'utilisateur.

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
