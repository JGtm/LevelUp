# Lot LS — signature du localisateur sur tout objet de l'archétype `high-frequency` (2026-10-04)

> Lot LS du plan `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (§6.2 LS, vague 2, juste après LU ; GO
> de l'utilisateur du 2026-10-04), sous le contrat `plan-execution`, dans le cadre de la décision du
> 2026-10-02 : **corrections d'abord, uniquement générales, lues dans le jeu**.
>
> Worktree `LevelUp-wt-cg2-ls`, branche `feat/cg2-ls` créée depuis `feat/cg2-lu` = `f73811aa8` (LU,
> sortie identique à `2393d7db7`). Films lus en place (`LevelUp/data/cache/film_chunks`, lecture
> seule), aucune cuisson du parc. Mesures et scripts : `scratchpad/v2-LS/` (session `f46f71fc`).
> GOCACHE `go-build-cg2-ls`, une commande `go` à la fois. Convention : **mesuré** = compté par un
> outil sur les films ; **établi** = lu dans le jeu ; **supposé** = hypothèse écrite.

## 0. Statut

| Item | Statut | En une ligne |
|---|---|---|
| LS.1 signature sur tout slot de l'archétype `high-frequency` | [!] (sans LS.3 et LS.4, aucun appelant : retiré avec eux, §12) | `localisateur.go` : un balayage rend la signature du slot 123 et, sans elle, la première signature haute fréquence (`marchLocateSignatures`) |
| LS.2 archétype reconnu par la TABLE, pas par le nom (LU.3, D-LU-4) | [!] (retiré avec LS.1, §12) | `rec.TypeIndex == archetypeHauteFrequence`, la clé de routage de L8 (`FUN_140e462d8`, `+0x4754 = 4`, table `0x143d06a60`) |
| LS.3 ordre de la cuisson : 123 strict → fermeture par NEW de tête → signature haute fréquence, sans repli libre | [!] (revue adverse de la vague 2, §12) | ordre MESURÉ, pas lu dans le jeu ; `localiserLaListe` (`debut_de_liste.go`), ordre `SignatureHauteFrequence` neuf |
| LS.4 ordre des marches (morts d'objet, killsource) : 123 strict → haute fréquence → repli libre | [!] (revue adverse de la vague 2, §12) | ordre MESURÉ, pas lu dans le jeu ; `SignaturePuisLargeurLibre` recevait l'étage haute fréquence |
| LS.5 révisions | [~] (§12 : le rang `.3` reste dans la chronique comme historique de la branche ; `killsource-2026-10-04` retiré) | `grammar-2026-10-03.3` (LN porte `grammar-2026-10-04`), `killsource-2026-10-04` ; `objectives` constante |
| Gate 1 (tests, vet, archlint, révision) | [x] | §4 |
| Gate 2 (carte v2, 20 films) | [x] | aucun film en baisse, 0 sain perdu (§3) |
| Gate 3 (killsource) | [x] | 229 morts du balayage à la marche sur les 19 témoins, 312 sur `1c4c63c2`, 364 sur les trois films de l'enquête ; aucune valeur changée (§5) |
| D22 (gate 3 sur `1c4c63c2`) | [x] | la carte du film est dans le dépôt (D-LS-1) : gate 3 joué |
| D-111 (alerte « hors roster » de `6b0e6f0f`) | [x] | instruite : un participant réel lu à l'indice 16, refusé avant comme après (§5.3) |
| Backfill killsource du parc | [~] sans objet après le retrait (§12) | `killsource.Rev` montait (voie publiée, D-74) |
| Corrections du contrôle (C1 test de la chaîne derrière la haute fréquence, C2 règle 6, C3 règle 17) | [x] | §11 ; sortie de production inchangée, révisions inchangées |

Verdict : **[!] RETIRÉ de la vague** à la revue adverse du 2026-10-04 (§12) ; retenu jusque-là (`[x]`, intégration `da6ecda38`).

## 1. Ce qui est lu dans le jeu (établi)

Ghidra, `HaloInfinite.exe` (HI_1_13_0), HTTP direct 127.0.0.1:8089, lecture seule ; décompilations
relues dans cette session (`scratchpad/v2-LS/g_*.c`) :

- `FUN_142f2e174` (liste des entités de la vue B) parcourt la table de vue `vue+0x38` par index
  CROISSANT (mots du bitmap `vue+0x58`, `uVar9` de 0 vers le haut) ; une entité déjà créée chez le
  lecteur (état 3) reçoit le genre 3 (DELTA, `| 0x1800000`) seulement si `FUN_142f24dd4` a quelque
  chose à écrire, ou le genre 2 (DEL) ; une entité neuve, le genre 1 (NEW).
- `FUN_14076b9c8` concatène les sous-écrivains dans l'ordre `+0x1b090` (NEW), `+0x1b240` (DELTA),
  `+0x1b168` (DEL). Donc **vue B = NEW (slots croissants), puis DELTA (slots croissants), puis DEL** :
  le premier DELTA est celui du plus petit slot qui en écrit un (déjà établi par T3 §1.3).
- `FUN_140e462d8` (enregistrement de l'archétype 4) : `FUN_14064dd28(param_1 + 8, 0,
  &PTR_PTR_144746d38, …)` puis `*(param_1 + 0x4754) = 4` : un seul composant, l'objet
  `0x144746d38` (`high-frequency`, table `0x143d06a60`, lecteur `FUN_14076d034`, écrivain
  `FUN_142eda680`, relevés par L8). Tous les objets de l'archétype 4 ont donc le même écrivain : un
  delta de 35 bits à composant unique sur le slot 127 a la même forme que sur le slot 123.
- Le bit nul qui précède la boucle de records (terminateur de la vue A, `FUN_142f2c3b0` /
  `FUN_1406d49c4`) est celui de LU, inchangé.

Ce qui n'est PAS lu dans le jeu (écart §7) : le slot 123, la largeur 35 et le composant unique
restent mesurés ; l'ordre entre la fermeture par NEW de tête et la signature haute fréquence (cuisson)
et l'ordre des marches sont MESURÉS (R-LS, `R_LOC.md` §3, et §3.3 ci-dessous), le jeu ne dit pas
quelle preuve essayer d'abord.

Lecture de l'enquête sur pièces (`ENQUETE_SCAN_SEPTEMBRE_2026-10-02.md` §2.1, §3.1, §4.1) : les
chaînes relues sur `8f7f5806` vont par slots croissants (statborg `ti=6`, joueur `ti=5`, delta
haute fréquence de 35 bits sur le slot 127 ou 128, bipèdes), ce que `FUN_142f2e174` explique ; le
slot 123 n'y figure pas. L'enquête proposait de lire l'archétype « par son nom » ; le plan (critique R
point 7, R-HOM) l'a remplacé par la table, ce que fait ce lot.

## 2. Ce qui change

### 2.1 Code de production

- `grammar/localisateur.go` :
  - `marchLocateSignatures(pay, w, cfg) (int, int)` remplace `marchLocateStrict` et
    `marchSignature123` : UN balayage (bit nul devant, `TryDeltaAt`, forme de 35 bits à composant
    unique) ; la première position sur le slot 123 l'arrête ; sinon il rend la première position dont
    l'objet est lié à l'archétype `high-frequency` à la génération du monde (`formeDeSignature`,
    `signeLaHauteFrequence`). Le garde-rail de LU (deux essais de position dans l'hôte) tient.
  - `OrdreDeLocalisation` : `SignatureStricte` (inchangé, slot 123 seul), `SignaturePuisLargeurLibre`
    (slot 123 à la génération du monde ; sans signature du slot 123, la signature haute fréquence ;
    sinon le repli à largeur libre), `SignatureHauteFrequence` (neuf : la signature haute fréquence
    d'un paquet sans signature du slot 123).
  - En-tête réécrit (règle 17) : ordre par site, loi d'écriture de la vue B, archétype par table.
- `grammar/debut_de_liste.go` : `localiserLaListe` = signature du slot 123 → fermeture par NEW de
  tête (deux rangs, inchangés) → `SignatureHauteFrequence`, la chaîne de tête (`debutParChaine`)
  s'appliquant derrière l'une ou l'autre signature ; aucun repli à largeur libre. En-tête mis à jour.
- `facts/killsource/walk.go` : en-tête du localisateur (l'appel est inchangé, l'ordre porte l'étage).
- Commentaires rendus faux par le lot (règle 17) : `grammar/lecture/paquet.go` (doc de
  `DebutParSignature`), `grammar/observateur.go` (liste des fonctions du localisateur).
- Révisions : `grammar/rev.go`, `grammar/rev_chronique.go` (entrée `grammar-2026-10-03.3`),
  `grammar/testdata/grammar_rev.golden` ; `facts/killsource/rev.go`, `rev_chronique.go` (entrée
  `killsource-2026-10-04`, 484 lignes), `testdata/killsource_rev.golden`.
- Goldens à révision changée seulement : `types/testdata/shapes.golden` (ligne `revisions`), les huit
  fixtures de contrat `apps/web/src/features/match-replay/test/fixtures/go/replay_schema_77_*.json.gz`
  et `manifest.json` (tailles) : décompressées, **0 ligne différente hors chaînes de révision** sur
  les huit.

### 2.2 Tests

- `grammar/localisateur_test.go` : `marchLocateStrict` devient une aide de test (les 60 sondes
  `research` l'appellent) ; tests neufs :
  - `TestLaSignatureHauteFrequenceNeLitQueLArchetype` : sur les deux bobines réelles
    (`minibobine_000d5950`, `minibobine_e5adf7b2`), aucune signature haute fréquence là où le slot 123
    n'en porte pas ; `e5adf7b2` porte pourtant des deltas de forme signature sur `ti=5` et `ti=43`
    (mesuré par sonde jetable) ;
  - `TestLaSignatureHauteFrequenceLocaliseUnAutreObjetDeLArchetype` : vecteurs d'après l'écrivain —
    sur 446 paquets de `000d5950`, le premier delta du slot 123 est RECOPIÉ sur un slot libre lié à
    l'archétype 4 à la même génération (identifiant réécrit à `s + 1`, `IDLowBits` bits) : la cuisson
    et les marches démarrent sur lui (445 vecteurs ; 1 où une signature haute fréquence antérieure
    existe, D-LS-2) ; sous une autre génération, les marches l'écartent ;
  - `TestLaCuissonEssaieLaFermetureAvantLaHauteFrequence` : ordre des trois étapes de
    `localiserLaListe` lu dans l'arbre syntaxique (aucune bobine du dépôt ne porte un paquet que la
    fermeture et la signature haute fréquence localisent toutes deux) ;
  - `TestLocaliserBoucleDeRecordsSuitLOrdreDuSite` vérifie en plus que la signature stricte est sur le
    slot 123 ; `TestLaCuissonNeDemarreQueSurLaSignatureStricte` admet la signature haute fréquence.
  Aucun test renommé ni supprimé (baseline JSONL intacte).
- `grammar/etats_mouvement_porte_guard_test.go` : `marchLocateSignatures` remplace
  `marchLocateStrict` dans la liste ; le motif accepte `(int, int)`.
- Commentaires de tests pointant des noms disparus : `deto_preuve_robuste_helpers_test.go`,
  `source/bits_conventions_test.go`.

## 3. Carte de fermeture v2, avant / après, par film (gate 2)

`cmd_fermeture -mode v2 -denominateur-fixe integ2/denominateurs.tsv -paquets`, binaires de la base
(`git archive f73811aa8`) et du lot, table ECS identique. Juge de L0 (colonne `ferme`). Comparaison
par `integ2/gate2.awk`. Contrôle : la carte de la base est identique à l'octet à celle de LU
(`fermeture_paquets.tsv` et 12 autres TSV ; `fermeture_films.tsv` hors pic et durée).

| Film | Sains avant | Sains après | Utiles sains avant | Utiles sains après | Perdus (contredits / non fermés) | Gagnés | dont factices | Non localisés |
|---|---|---|---|---|---|---|---|---|
| bcb6d393 | 5 879 | 5 879 | 35 492 | 35 492 | 0 | 0 | — | = |
| fb1a1a72 | 43 450 | 43 450 | 325 749 | 325 749 | 0 | 0 | — | = |
| d9781168 | 26 317 | **36 995** | 180 854 | **278 340** | 0 | 10 678 | 25 | 5 732 → 629 |
| c75f33b8 | 23 872 | **26 148** | 153 997 | **173 096** | 0 | 2 276 | 0 | 2 310 → 301 |
| bf15f7ab | 28 603 | 28 603 | 216 104 | 216 104 | 0 | 0 | — | = |
| 51ebbc0f | 20 443 | **25 471** | 140 617 | **185 697** | 0 | 5 028 | 11 | 2 723 → 383 |
| 084a804d | 4 832 | 4 832 | 89 616 | 89 616 | 0 | 0 | — | = |
| 0797ce72 | 19 156 | 19 156 | 159 929 | 159 929 | 0 | 0 | — | = |
| 111fa685 | 4 026 | 4 026 | 42 896 | 42 896 | 0 | 0 | — | = |
| e5adf7b2 | 4 147 | 4 147 | 80 067 | 80 067 | 0 | 0 | — | 1 726 → 1 706 |
| 60ae07c4 | 13 946 | **14 594** | 83 669 | **88 031** | 0 | 648 | 0 | 3 259 → 735 |
| a349fea8 | 424 | 424 | 3 654 | 3 654 | 0 | 0 | — | = |
| a521164d | 692 | 692 | 78 | 78 | 0 | 0 | — | = |
| 11de8353 | 5 629 | 5 629 | 65 621 | 65 621 | 0 | 0 | — | = |
| 50247b26 | 139 | 139 | 274 | 274 | 0 | 0 | — | = |
| bfecd02b | 27 666 | 27 666 | 239 045 | 239 045 | 0 | 0 | — | = |
| 4f77afc1 | 24 082 | 24 082 | 607 244 | 607 244 | 0 | 0 | — | = |
| 396cfc92 | 23 131 | 23 131 | 169 502 | 169 502 | 0 | 0 | — | = |
| f75e7053 | 23 673 | 23 673 | 162 137 | 162 137 | 0 | 0 | — | = |
| 1c4c63c2 | 13 388 | **13 887** | 165 965 | **176 877** | 0 | 499 | 0 | 12 411 → 2 596 |
| **corpus** | **313 495** | **332 624** (+19 129) | **2 922 510** | **3 099 449** (+176 939) | **0 (0 / 0)** | **19 129** | **36** | **47 854 → 26 043** |

- **Gate 2 tenu : aucun film en baisse**, ni en paquets sains ni en records utiles sains ; 0 sain
  perdu au sens strict (les deux comptes bruts nuls).
- Juge des invariants : sur les GAGNÉS, 19 162 paquets fermés au bit, dont **36 contredits (0,2 %)**
  — `d9781168` 24 « masque au-delà de l'archétype » + 1 « ordre de la vue B », `51ebbc0f` 11
  « masque au-delà de l'archétype » (la règle de tête de liste des découvertes de la vague 1). Sur les
  PERDUS : aucun sain ; 3 paquets fermés au bit perdus, tous sur `1c4c63c2` et tous CONTREDITS avant
  le lot (16:2116 et 38:650 « vue B : sortie par rejet », 17:1572 « ordre de la vue B ») : fermetures
  factices retirées (D2).
- Dénominateur fixe (`integ2/denominateurs.tsv`) : aucun film ne le dépasse (colonne `fixe` =
  `fixe_donne` partout). Indicateur D1 (utiles sains / fixe) : HI_1_13_0 **76,6 % → 81,9 %**,
  HI_1_10_0 12,3 → 12,8 %, HI_1_8_0 23,3 → 24,5 %, corpus 37,7 → 40,0 %.
- Témoins de l'enquête : seuls changent les films où le monde lie d'autres slots à l'archétype 4.
  Contre R-LS (mesuré à `fe18bf67c`, avant la vague 1, sonde `ls2`) : +18 086 sains, ici +19 129 ;
  même signe sur les six mêmes films.

### 3.1 Compteur `repli_debut_de_liste_ferme_au_bit`

La carte v2 ne publie pas ce compteur par film ; il est relevé dans le journal de `replay-equiv`
(« repli declenche », 20 films du corpus d'équivalence), base contre lot : identique sur 17 films ;
`d9781168` **261 → 283** (+22), `9f57c612` 112 → 113, `1c4c63c2` 6 428 → 6 427. Total 8 047 → 8 069.
Non instruit paquet par paquet (D-LS-5).

### 3.2 Ordre de la cuisson : la mutation de l'ordre de l'enquête, rejouée sur la carte

Binaire de la carte construit sous `-overlay` M2 (signature haute fréquence AVANT la fermeture par
NEW de tête), mêmes films : `1c4c63c2` **13 388 → 12 685 sains (−703)**, 1 588 sains perdus en brut ;
pertes brutes aussi sur `c75f33b8` (200), `60ae07c4` (13), `51ebbc0f` (5), `d9781168` (1). Le
corpus gagnerait plus de records utiles sains (+193 032 contre +176 939) mais le gate par film
tombe : l'ordre retenu par R-LS est confirmé sur la base actuelle (D-78 / R-LOC-6 restent ouverts).

## 4. Gates (sorties exactes, depuis `apps/go-api`, GOCACHE du lot)

- `gofmt -l ./internal ./cmd` : vide.
- `go vet ./...` : rc 0. `go vet -tags=research ./internal/games/halo_infinite/film/...` : rc 0.
- `go test ./internal/archlint/` : voir §9 (rejoué sur l arbre final).
- G-film (`go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/...
  ./internal/sync/killcollector/... -count=1 -timeout 30m`) : premier passage rouge sur
  `TestContractFixturesMatchCommitted` et `TestFormesDesTypesEgalentLeGolden` (chaînes de révision
  seulement), goldens régénérés par leurs commandes (`REPLAY_CONTRACT_UPDATE=1 … -update`,
  `LEVELUP_UPDATE_TYPES_SHAPES=1 … -update-types-shapes`), puis rejoué sur l arbre final (§9).
- **Révisions** : `grammar.Rev` `grammar-2026-10-03.2` → **`grammar-2026-10-03.3`** (valeur
  qu'aucune branche ne porte : LN porte `grammar-2026-10-04`), entrée de `rev_chronique.go`,
  empreinte régénérée (`acd64fde…`) ; `killsource.Rev` `killsource-2026-09-27` →
  **`killsource-2026-10-04`** (sortie persistée changée, §5), entrée de chronique, empreinte
  régénérée ; `objectives.Rev` inchangée, ses tests de révision verts sans régénération (gate
  objectives : la lecture des signaux ne change pas) ; `replay.SchemaVersion` 77 inchangé.
- `frame_closure.golden` : `-update-frame-closure` ne change rien (`git diff` vide) ; ses bobines ne
  lient aucun autre slot à l'archétype 4.
- **`replay-equiv`** (racine factice `scratchpad/v2-LS/repo`, copie de `v2-LU/repo` sans faits ni
  artefacts) : base et lot rendent tous deux `BILAN : 0 identique(s), 20 different(s)` contre les
  références de `67c379fc1`, non re-figées après la vague 1 (écart déjà déclaré par LU). Base contre
  lot, digests à l'octet : **14 films identiques**, 6 changent —
  `1c4c63c2` (killsource, killRefs, vehicles, movementStates, movementStates.stats, continuousFire,
  continuousFire.stats, artifact), `60ae07c4` (killsource, killRefs, vehicles, movementStates(.stats),
  continuousFire.stats, artifact), `64e8adfa` (killsource, killRefs, movementStates(.stats),
  continuousFire.stats, artifact), `9f57c612` et `d9781168` (killsource, killRefs, movementStates(.stats),
  continuousFire(.stats), artifact), `e5adf7b2` (movementStates.stats, continuousFire.stats, artifact).
  Les binaires de `replay-equiv`, de la carte et de killsource sont ceux de l'arbre du lot AVANT la
  montée des constantes de révision (seule différence avec l'arbre final : les deux constantes et
  leurs chroniques).
- Replis du journal du rejeu (524 lignes) : 21 lignes changent, toutes sur ces 6 films ;
  `repli_localisation_largeur_libre` baisse (`d9781168` 838 → 196, `1c4c63c2` 3 080 → 1 369,
  `9f57c612` 195 → 36, `60ae07c4` 884 → 680, `64e8adfa` 477 → 423) ;
  `repli_deadstate_indice_hors_roster` apparaît (`1c4c63c2` 8, `9f57c612` 5, `60ae07c4` 2, cf. §5.3) ;
  `repli_liaison_par_anticipation`, `repli_episode_*`, `repli_identite_premier_occupant_du_siege`,
  `repli_deadstate_hors_bande_bipede`, `repli_tourelle_porteur_voisin_de_slot` bougent sur
  `1c4c63c2` (non instruits ligne à ligne).
- Durées de `replay-equiv` : la machine portait d'autres sessions (LN, rejeu des vies de bots) ;
  gate 4 sans objet (LS ne lit pas le bloc de type 1) ; non mesuré.

### 4.1 Mutations (chacune prouvée rouge)

Surcouche `-overlay` (`scratchpad/v2-LS/mut/`), sauf M2 jouée EN PLACE (le test lit le source sur
disque ; fichier restauré, `cmp` à l'octet vérifié).

| # | Mutation | Rouge |
|---|---|---|
| M1 | les marches perdent l'étage haute fréquence | `TestLaSignatureHauteFrequenceLocaliseUnAutreObjetDeLArchetype` (« ordre des marches (-1, false), attendu (89, false) ») |
| M2 | la cuisson essaie la signature haute fréquence AVANT la fermeture par NEW de tête | `TestLaCuissonEssaieLaFermetureAvantLaHauteFrequence` ; sur la carte : `1c4c63c2` −703 sains (§3.2) |
| M3 | le prédicat oublie l'archétype (toute forme de signature) | `TestLaSignatureHauteFrequenceNeLitQueLArchetype` (slot 131 / 124, archétype 21) et `TestLocaliserBoucleDeRecordsSuitLOrdreDuSite` |
| M4 | la signature haute fréquence sans contrôle de génération | `TestLaSignatureHauteFrequenceLocaliseUnAutreObjetDeLArchetype` (« la signature d une autre generation est gardee ») |
| M5 | la cuisson sans l'étage haute fréquence | `TestLaSignatureHauteFrequenceLocaliseUnAutreObjetDeLArchetype` (« cuisson (-1, 6) ») |
| M6 | la signature haute fréquence prime sur celle du slot 123 | `TestLaSignatureHauteFrequenceLocaliseUnAutreObjetDeLArchetype` (« la signature du slot 123 est encore vue ») |
| X2 | la cuisson sans chaîne de tête derrière la signature haute fréquence (verte au contrôle, corrigé) | `TestLaChaineDeTeteSuitLaSignatureHauteFrequence` (§11.1) |
| R6 | le littéral du contrôle de génération recopié hors du helper | `archlint` `TestLocalisateurDeBoucleUnique` (§11.2) |





## 5. Gate 3 — killsource

`killsource json <film> -carte <carte> -cache … -catalogue map_quant_bounds.json`, base contre lot,
morts appariées par (instant, victime) (`scratchpad/v2-LS/ks_cmp.sh`).

### 5.1 Les 19 témoins de `config/replay_corpus.toml`

22 / 22 rc 0 (19 témoins + 3 films de l'enquête). **16 JSON identiques à l'octet.** Changent :

| Film | Morts | Balayage → marche | Marche → balayage | Tag / statut / crédit changés | Apparues / disparues |
|---|---|---|---|---|---|
| d9781168 | 142 | **103** | 0 | 0 / 0 / 0 | 0 / 0 |
| c75f33b8 | 61 | **48** | 0 | 0 / 0 / 0 | 0 / 0 |
| 60ae07c4 | 158 | **78** | 0 | 0 / 0 / 0 | 0 / 0 |
| **19 témoins** | 2 747 | **229** | 0 | 0 | 0 |

Hors morts, ne changent que `sante.gate_par_voie.*` et `sante.concordance.*` (répartition entre les
voies). Identique à R-COMB-2 §6.2 (229 : 103, 78, 48). `51ebbc0f` inchangé à l'octet alors que sa
carte gagne +5 028 sains (D-76, non instruit).

### 5.2 Les trois films de l'enquête et `1c4c63c2`

| Film | Morts | Balayage → marche | Origine « les deux vérités concordent » en balayage | Autres valeurs |
|---|---|---|---|---|
| 8f7f5806 (Live Fire) | 226 | 158 | 167 → 10 (**157**) | 0 |
| 6b0e6f0f (Refuge) | 255 | 181 | 209 → 29 (**180**) | 0 |
| 9c0ec856 (Recharge) | 100 | 25 | 27 → 2 (**25**) | 0 |
| **enquête** | 581 | 364 | **362 / 403** | 0 |
| 1c4c63c2 (Refuge, carte du dépôt, D-LS-1) | 493 | **312** | 412 → 101 | 0 |

Les trois films de l'enquête redonnent exactement 362 / 403 ; marche / balayage `216 / 10`,
`226 / 29`, `97 / 3`, comme `SURCOUCHE_UNIQUE.md` §4.6. **D22 statué** : `1c4c63c2` est joué avec sa
carte, lue dans le dépôt (D-LS-1) : 312 morts passent à la marche, aucune valeur, aucune mort
apparue ni disparue. `81c02726` reste sans carte lisible (absent du jeu d'équivalence), non joué.

### 5.3 D-111 : « dead-states hors roster » de `6b0e6f0f`, 10 → 42 (instruite)

Instrument existant `TestRosterEtRemplacements` (`killsource/roster_remplacements_research_test.go`,
`KS_ROSTER_FILM=…/6b0e6f0f KS_ROSTER_CARTE=Refuge`), joué sur la base (`src_base`) et sur le lot :

- base : 10 dead-states hors roster, TOUS avec l'indice 16 d'un côté (tueur 16 : 5 ; victime 16 : 5) ;
- lot : 42, TOUS avec l'indice 16 d'un côté (tueur 16 : 23 ; victime 16 : 19) ;
- le motif du xuid lit l'indice **16** pour `2535427268703421` (« V1RU54702 », index **9** de la
  table du film) sur 66 chunks sur 66 (`MotifDuplicate:1`) ; le roster retient 16 sièges (0 à 15).

Lecture : ce sont des morts RÉELLES d'un participant que le roster ne place pas à l'indice que le
film écrit ; le filtre de crédibilité de la marche les refusait avant le lot et les refuse après. La
marche en lit simplement plus parce qu'elle localise plus de paquets. Ce n'est pas une lecture
fausse du localisateur ; la cause (indice 16 contre table 9) est un défaut d'identité préexistant,
hors du lot (D-LS-4). Valeur publiée : aucune (diagnostic) ; les morts publiées de `6b0e6f0f` ne
changent que de voie (§5.2).

### 5.4 Conséquence

La voie (`read_path` de `match_kill_events`, D-74) change : **`killsource.Rev` monte**
(`killsource-2026-10-04`) et **le backfill killsource du parc est DÛ** (non lancé ; le hook
post-sync installé par défaut le déclenche au déploiement, huit films par cycle, D-REV-3). Les
positions des morts ne sont pas publiées par `killsource json` : ce gate ne les mesure pas.

## 6. Pertes instruites

Aucun paquet sain perdu, aucun record utile sain perdu, sur aucun film. Les 3 paquets fermés au bit
perdus (`1c4c63c2`) étaient contredits avant le lot (fermetures factices retirées, §3). Aucune mort
killsource apparue, disparue ni changée de valeur.

## 7. Écarts au critère et au plan

- **Ordres mesurés, pas lus** : le jeu établit où commence la vue B, l'ordre d'écriture de ses records
  et l'écrivain unique de l'archétype 4 ; il ne dit pas quelle preuve essayer d'abord. L'ordre de la
  cuisson (fermeture par NEW de tête avant la signature haute fréquence) et celui des marches sont
  ceux mesurés par R-LS, confirmés ici (§3.2) ; ils sont généraux (aucune condition par film, carte ou
  build). Le slot 123, la largeur 35 et le composant unique restent mesurés (comme avant LU).
- Dans les marches, une signature du slot 123 d'une AUTRE génération n'ouvre pas l'étage haute
  fréquence (le repli libre suit) : c'est la règle mesurée par R-LS (la surcouche cherchait les deux
  signatures en un balayage arrêté au slot 123) ; non instruit (D-LS-8).
- `replay-equiv`, carte et killsource joués sur l'arbre du lot avant la montée des constantes de
  révision (§4) ; G-film et archlint rejoués sur l'arbre final (§9).
- Non joués : `replay-corpus-gate` et banc de vérité (§6.0 point 6, hors de la liste des gates du
  lot) ; revue adversariale (laissée au pilote, comme pour LU).
- `81c02726` hors gate 3 (aucune carte lisible).

## 8. Fichiers de la représentation intermédiaire touchés

`grammar/lecture/paquet.go` : deux lignes de commentaire (doc de `DebutParSignature`, rendue fausse
par le lot). Aucun autre fichier de la liste de la RI. `frame_harvest.go:313` cite encore
`marchLocateStrict` (devenu une aide de test) : non touché pour ne pas entrer en conflit avec la RI
(D-LS-7).

## 9. Arbre final (rejoué)

Après l'écriture de cette note, la montée des révisions et la régénération des goldens :

- `gofmt -l ./internal ./cmd` : vide ; `go vet ./...` : rc 0 ; `go vet -tags=research
  ./internal/games/halo_infinite/film/...` : rc 0.
- `go test ./internal/archlint/` : `ok levelup/go-api/internal/archlint 34.379s` (premier passage
  rouge : `TestCheminsAiCitesDansLeCodeExistent`, l'en-tête de `localisateur.go` citait cette note
  avant qu'elle existe).
- G-film : 20 paquets `ok`, rc 0.
- `golangci-lint run --new-from-rev f73811aa8` (grammar, facts) : `0 issues.`
- Tailles : `localisateur.go` 175 lignes, `localisateur_test.go` 334, `debut_de_liste.go` 212,
  `killsource/rev_chronique.go` 484, `grammar/rev_chronique.go` 345.

## 10. Découvertes (consignées, non traitées)

- **D-LS-1 — La carte de `1c4c63c2` est dans le dépôt.** `replay/testdata/equivalence/1c4c63c2.facts.json`
  (`mapNames: ["Refuge"]`, `gameVariantName: "BTB:One Flag CTF"`), `CORPUS.txt` l. 284, `BOMBES.txt`
  l. 18 ; `replay-equiv` cuit déjà ce film sous `cartes=[Refuge]`. La prémisse de D22 (« aucune source
  lisible ») est fausse : le gate 3 se joue sur 20 films. `81c02726` n'a pas cette source.
- **D-LS-2 — Une signature haute fréquence peut précéder la vue B.** Sur `minibobine_000d5950`, le
  slot 225 est lié à l'archétype 4, et un paquet porte une forme de signature sur le slot 225 au bit
  576 alors que sa signature du slot 123 (vraie) est au bit 7979. Par `FUN_142f2e174`, le delta du
  slot 225 ne peut pas précéder celui du slot 123 dans la vue B : la position 576 est dans la vue A.
  Seul l'ordre « slot 123 d'abord » l'écarte ; un paquet sans slot 123 peut donc démarrer trop tôt.
  Part mesurée des gains contredits : 36 / 19 162 (0,2 %). Une garde par la loi d'écriture (aucun
  delta de slot inférieur ne suit) n'est pas instruite.
- **D-LS-3 — La signature localise le premier delta HAUTE FRÉQUENCE, pas le premier delta.** Les
  deltas de slots inférieurs (statborg `ti=6`, joueur `ti=5`, enquête §2.1) le précèdent dans la vue
  B ; `debutParChaine` ne remonte qu'à travers des candidats NEW : ces deltas ne sont pas lus.
- **D-LS-4 — Indice 16 contre table 9 sur `6b0e6f0f`** (§5.3) : défaut d'identité préexistant ;
  LS en fait lire 32 dead-states de plus, tous refusés. À reprendre par le chantier du roster.
- **D-LS-5 — `repli_debut_de_liste_ferme_au_bit` monte sur `d9781168`** (261 → 283) : non instruit.
- **D-LS-6 — L'ordre de l'enquête rend plus de records utiles sains au corpus** (+193 032 contre
  +176 939) mais fait baisser `1c4c63c2` (−703 sains) : R-LOC-6 / D-78 restent ouverts.
- **D-LS-7 — `frame_harvest.go:313`** (fichier de la RI) cite `marchLocateStrict` : règle 17 à
  appliquer par la RI.
- **D-LS-8 — Génération étrangère du slot 123 dans les marches** : l'étage haute fréquence n'est pas
  essayé (§7).
- **D-LS-9 — Double balayage dans la cuisson** : un paquet qui atteint le troisième étage est balayé
  deux fois (`SignatureStricte`, puis `SignatureHauteFrequence`) ; coût non mesuré.

## 11. Corrections du contrôle (2026-10-04)

Contrôle du lot sur `939895542` (rapport relayé par le pilote) : gates 2 et 3 retrouvés à l'unité ;
trois non-conformités, corrigées ici sans toucher la sortie de production.

| # | Correction | Statut | Ce qui est fait |
|---|---|---|---|
| C1 | Point 7 : la mutation X2 (`return debut, lecture.DebutParSignature` juste après l'étage haute fréquence de `localiserLaListe`) restait verte | [x] | Test neuf `TestLaChaineDeTeteSuitLaSignatureHauteFrequence` (`grammar/debut_de_liste_haute_frequence_test.go`), 66 vecteurs ; X2 rouge (66 / 66) |
| C2 | Règle 6 : 3e copie de `w.GenerationMatches(rec.ID, cfg.Profil.Grammaire.GenerationStricte)` dans `localisateur.go` | [x] | Helper `aLaGenerationDuMonde` appelé par les trois sites (et par l'aide de test `generationDuMonde`) ; garde-rail dans `archlint/film_localisateur_unique_test.go` |
| C3 | Règle 17 : « signature du premier delta », « c'est ce que la signature cherche », « sans contrôle de génération » | [x] | `debut_de_liste.go` (en-tête), `lecture/paquet.go` (doc de `DebutParSignature`, deux lignes), `localisateur.go` (en-tête : ordre des sites, loi de la vue B) |

### 11.1 C1 — pourquoi X2 restait verte, et le vecteur

Mesuré : l'aide `parcourirPaquetsAEvenements` des tests du localisateur monte le monde de la marche
des morts d'objet SANS table anticipée. Or `candidatsDeTete` lit la bande d'un archétype dans
cette table (`TableAnticipee.SlotDeLArchetype`) : sans elle, aucun candidat, aucune chaîne. Sur
`minibobine_000d5950`, `localiserLaListe` rend 446 `DebutParSignature` et 27 `DebutNonLocalise`
sans table ; avec la table du film (celle que pose la marche des trames de la cuisson,
`marche_trames.go`), **377 signature, 69 chaîne, 27 non localisés** (sonde jetable, supprimée).
Les 445 vecteurs de `TestLaSignatureHauteFrequenceLocaliseUnAutreObjetDeLArchetype` n'avaient donc
aucune chaîne parce que leur monde n'en permettait pas, pas parce que la bobine n'en porte pas.

Le vecteur : `parcourirLaBobine(…, avecTable=true, …)` (l'aide existante, paramétrée) ; sur chaque
paquet réel dont la cuisson rend `DebutParChaine` derrière la signature du slot 123 (à la génération
du monde), le premier delta du slot 123 est recopié sur un autre slot de l'archétype 4
(`recopieSurUnAutreSlot`, inchangée). Gardés : les paquets où la signature haute fréquence tombe sur
la position recopiée (sinon D-LS-2) ET où aucun candidat de tête ne ferme le paquet (sinon la
fermeture, qui précède l'étage haute fréquence, répond). Attendu : la cuisson rend le MÊME premier
NEW de tête, par la chaîne. **66 vecteurs** ; aucun record n'est inventé, les NEW de tête et les
deltas intermédiaires sont ceux du film.

- Arbre corrigé : vert.
- X2 par `-overlay` (chemin long, `scratchpad/v2-LS/mut2/X2.json`) sur le paquet `grammar` : **rouge**,
  `TestLaChaineDeTeteSuitLaSignatureHauteFrequence`, 66 erreurs « cuisson (s, 2), attendu (tête,
  chaîne) » (ex. slot 132 : `(557, 2)` attendu `(229, chaine)`). Seul autre rouge sous la surcouche :
  `TestGrammarRevSuitLaGrammaire`, qui lit les sources sur disque (empreinte, voir C2), pas la surcouche.

### 11.2 C2 — règle 6

`aLaGenerationDuMonde(rec, w, cfg)` (`localisateur.go`) porte seul le littéral ; `LocaliserBoucleDeRecords`
(signature du slot 123 des marches), `signeLaHauteFrequence` et `marchLocateFallback` l'appellent.
Garde-rail : `TestLocalisateurDeBoucleUnique` compte le littéral sur le même périmètre que ses deux
formes (production de `film/**`, hors research et tests) : exactement 1 dans l'hôte, 0 ailleurs.
Vérifié rouge deux fois : le littéral remis dans `marchLocateFallback` (« porte 2 fois … 1 attendue »,
fichier restauré, `cmp` à l'octet) ; une copie dans un fichier temporaire de `facts/killsource`
(« 1 copie(s) … appeler le helper », fichier supprimé). Hors périmètre, non traités : les copies des
aides de test préexistantes (`deto_preuve_robuste_helpers_test.go`, `vehicules_v13_marche_helpers_test.go`)
et des sondes research (`r_loc_ls_research_test.go`, `r_loc_vuea_research_test.go`) ; le contrôle
dur du lecteur de records (`frame_records.go`, sur `id`) est un autre site (rejet de l'en-tête), il
n'est pas le littéral.

### 11.3 C3 — règle 17

- `debut_de_liste.go` : le localisateur de production = la signature du slot 123 ou, sans elle, celle
  du premier delta HAUTE FRÉQUENCE, que des deltas de slots inférieurs peuvent précéder (paragraphe
  re-coupé, rien d'autre).
- `lecture/paquet.go` (fichier de la RI) : doc de `DebutParSignature`, toujours deux lignes (« la
  signature du slot 123 ou, sans elle, celle du premier delta haute fréquence »).
- `localisateur.go` : « la cuisson prend la première signature DU SLOT 123 telle quelle » ; « à tous
  les sites, la signature haute fréquence exige la génération du monde (`signeLaHauteFrequence`) » ;
  « …, ET C'EST CE QUE LA SIGNATURE CHERCHE » retiré, remplacé par : la signature haute fréquence
  localise le premier delta haute fréquence, pas le premier delta de la vue B (D-LS-3).
- Non touchés, relevés : la chronique `grammar-2026-10-03.3` dit que ce delta « ouvre la boucle de
  records » (vrai de ce que fait le décodeur, l'écart est D-LS-3) ; `localisateur.go` garde « Le
  premier delta est celui du slot 123 » (mesure 690 / 690, non contredite par le contrôle).

### 11.4 Gates rejoués sur l'arbre corrigé

Révisions inchangées : `grammar-2026-10-03.3`, `killsource-2026-10-04`. Le hachage de la grammaire
porte sur les jetons : le helper change l'empreinte, régénérée à révision constante par la commande
du dépôt (`acd64fde…` → `d10a7332…`, ligne `grammar-2026-10-03.3` de `grammar_rev.golden`).
L'empreinte de `killsource` hache la valeur de `grammar.Rev` (inchangée) et `facts/` (intact).

- `gofmt -l ./internal ./cmd` : vide. `go vet ./...` : rc 0. `go vet -tags=research
  ./internal/games/halo_infinite/film/...` : rc 0.
- `go test ./internal/archlint/` : `ok levelup/go-api/internal/archlint 36.258s`.
- G-film (`go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/...
  ./internal/sync/killcollector/... -count=1 -timeout 30m`) : 20 paquets `ok`, rc 0
  (`scratchpad/v2-LS/gfilm_corrections.txt`).
- `golangci-lint run --new-from-rev 939895542` (grammar, archlint) : `0 issues.`
- Carte v2, 20 films, binaire de l'arbre corrigé (`bin/corr`, table ECS identique) contre celle du
  lot (`carte_neuf`) : les 13 TSV par paquet et par famille **identiques à l'octet**,
  `fermeture_films.tsv` identique hors pic et durée ; corpus 332 624 sains, 3 099 449 utiles sains,
  26 043 non localisés, comme §3.
- Killsource (`killsource json`, 19 témoins + 3 films de l'enquête + `1c4c63c2` sous Refuge) :
  **23 JSON identiques à l'octet** à ceux du lot (`ks_neuf`).
- Tailles : `localisateur.go` 182 lignes, `debut_de_liste.go` 213, `localisateur_test.go` 345,
  `debut_de_liste_haute_frequence_test.go` 57, `archlint/film_localisateur_unique_test.go` 261 ;
  `TestLocalisateurDeBoucleUnique` 64 lignes.
- Fichiers de la RI touchés par les corrections : `grammar/lecture/paquet.go` seul, deux lignes de
  commentaire (les mêmes que le lot).

## 12. Revue adverse de la vague 2 (2026-10-04) : le lot est RETIRÉ

Revue adverse de la vague 2 (17 constats), traitée sur `feat/campagne-grammaire` après la fusion de
`feat/v75` = `6fa631df0` (RI étape 2, vies de bots ; `f1895d9bc`). LS est retiré par revert de
`939895542` et `80d2acd20` (production, tests, révisions) ; cette note, le rang `.3` de la chronique de
`grammar` (historique de la branche) et les mesures restent. Mesures : `vague2_tsv/revue/` (pièces
brutes `scratchpad/v2-integ/rev/`). Convention inchangée : **mesuré**, **établi**, **estimé**.

### 12.1 Constats qui portent sur LS

| Constat | Verdict, sur pièces | Suite |
|---|---|---|
| 1 [bloquant] bit nul exigé devant la signature haute fréquence | VRAI. Établi (Ghidra relu) : `FUN_142f2c3b0` écrit la vue A puis le 0 (`FUN_1406d49c4`) dans le tampon du flux 0 et la vue B (`FUN_142f2cc78`) dans celui du flux 1 ; `FUN_14299d2c8` les met bout à bout au bit près (`FUN_1406d5d14`) ; `FUN_14076b9c8` concatène NEW, DELTA, DEL sans séparateur. Le 0 n'est écrit que devant le premier record de la vue B ; le delta haute fréquence suit les deltas des slots inférieurs (D-LS-3). Mesuré (ci-dessous) : la condition rejetait des positions vraies | étage retiré avec le lot |
| 2 [bloquant] ordres propres à chaque site, choisis à la mesure | VRAI : en-tête de `localisateur.go` (l. 22-24 à `da6ecda38`), §3.2 et §7 de cette note ; l'ordre de la cuisson a été retenu sur le gate de `1c4c63c2` | LS.3, LS.4 `[!]` |
| 12 [majeur] aucune décision datée ne couvre l'ordre mesuré | VRAI : D13 est dans le bloc du questionnaire du 2026-10-02 que l'utilisateur a SUSPENDU (plan §3) ; « corrections d'abord » retient LS sans parler d'ordre | décision laissée au pilote (§12.3) |
| 3 [majeur] largeur 35 étendue à un archétype | VRAI ; la largeur est désormais dérivée du cadre pour la signature du slot 123 (`LOT_LU.md` §9) | sans objet pour LS |
| 4 [majeur] replis « hors roster » et « hors bande » nés de LS | VRAI comme constat (`replis_changes.tsv` de l'intégration) ; lien causal ESTIMÉ par la revue | après retrait : §12.2 |
| 5 [mineur] contrôle de génération décrit comme actif à tous les sites | VRAI (`GenerationStricte` levé par killsource seul) | contrat corrigé dans le localisateur de LU (`LOT_LU.md` §9) |
| 7 [mineur] vecteurs à la même position que le slot 123 | VRAI (`recopieSurUnAutreSlot` ne réécrit que l'identifiant) | sans objet après retrait ; exigence d'une reprise (§12.3) |
| 11 [majeur] la signature haute fréquence n'avait ni marque ni compte propres (ADR 0037 IR-6) | VRAI (`DebutParSignature` pour les deux signatures ; aucun compteur) | sans objet après retrait (IR-6 cite `marchLocateStrict`, de nouveau en production) ; exigence d'une reprise |
| 13 [majeur] pertes du gate de corpus non instruites | VRAI : aucune n'était instruite | gate rejoué après retrait : §12.2 |
| 14 [majeur] backfill qui réécrirait des positions non mesurées | VRAI comme risque | sans objet : `killsource.Rev` reste `killsource-2026-09-27`, aucun backfill (§12.2) |

### 12.2 Après le retrait, mesuré

- **Condition du bit nul (constat 1)** — surcouche `rev/mut/localisateur_v1.go` sur l'arbre fusionné
  avec LS (la condition gardée pour le slot 123, retirée à l'étage haute fréquence), carte v2 des 20
  films contre `6fa631df0` : 313 495 -> **333 653** sains (+20 158 ; LS tel quel : 332 668, +19 173),
  utiles sains +187 651 (LS : +177 962), **aucun film en baisse, 0 sain perdu**. La condition
  rejetait donc des positions vraies (+985 sains sans elle) ; les comptes par paquet de la revue
  (`8f7f5806` 1 325 candidats précédés d'un 1, dont 1 100 derrière un delta ti5 / ti2) sont les siens,
  non rejoués ici.
- **LS sur la nouvelle base** (arbre fusionné, avant retrait) : carte identique en comptes à celle de
  l'intégration (332 668 sains, 3 100 472 utiles sains) — la RI et les vies de bots ne changent pas la
  carte (base `6fa631df0` = 313 495 sains, comme `2393d7db7`).
- **Tête après retrait** (LU + LT + corrections de la revue) contre `6fa631df0`, sur les mêmes
  pièces que l'intégration : killsource **identique à l'octet sur 20 films** (constat 14 sans objet) ;
  `replay-equiv`, base et tête : divergent `artifact` (20), `movementStates.stats` (17),
  `continuousFire.stats` (13), `continuousFire` (2 : `084a804d`, `1c4c63c2`) ; `killRefs`, `vehicles`,
  `killsource`, `movementStates` et `objectives` **identiques** (les positions persistées par le
  collecteur ne changent pas, constat 14) ; replis : seul `repli_liaison_par_anticipation` bouge (±1 à
  2 sur 5 films) ; **`repli_deadstate_indice_hors_roster` et `repli_deadstate_hors_bande_bipede` ne
  bougent plus** (constat 4 et D-INT2-1 : nés de LS, mesuré par leur disparition au retrait) ;
- **`replay-corpus-gate`** (`--base=6fa631df0`) : banc de vérité **19 / 19 ok** (le FAUX de
  `60ae07c4` a disparu) ; rc 1 sur DEUX pertes `[FILET]` seulement : `084a804d`
  `coverage.stances.records` 448 661 -> 448 659, `a349fea8` `coverage.continuousFire.holesNotClosing`
  22 852 -> 22 856. Les pertes du constat 13 (durée des postures −9 à −32 %, `holesOpenViewB` jusqu'à
  ×2,7, `holesNotClosing` de `60ae07c4`) ont disparu avec LS. Attribution des deux restantes :
  `a349fea8` vient de LT (digests de `replay-equiv` de LU + LT sans les corrections identiques à ceux
  de la tête, mesuré) ; `084a804d` est touché par LT et par la règle d'ordre de la chaîne, la part de
  chacune dans les 2 records n'est pas mesurée. Non instruites paquet par paquet : l'admission du rc 1
  revient au pilote.

### 12.3 Ce qui reste à décider (pilote, puis utilisateur)

LS ne revient qu'avec (a) un ordre fondé sur une règle de l'écrivain, le même à tous les sites et
sous le même contrôle de génération — la piste est D-LS-2 : prouver que la position haute fréquence
est dans la vue B par la loi `FUN_142f2e174` + `FUN_14076b9c8` —, ou (b) une décision DATÉE de
l'utilisateur qui accepte un ordre mesuré, la question posée en langage clair (« quand deux preuves
de début de liste se contredisent, le jeu ne dit pas laquelle croire ; on prend celle qui fait le
moins de dégâts mesurés : d'accord ? »). Dans les deux cas : sans la condition du bit nul à l'étage
haute fréquence (§12.2 : +985 sains), avec une marque et un compte propres à la signature haute
fréquence (ADR 0037 IR-6, constat 11), un vecteur réel où le delta haute fréquence suit un delta
`ti=5` (constat 7), les replis « hors roster » et « hors bande » instruits paquet par paquet
(constat 4), les pertes du gate de corpus instruites (constat 13) et les positions persistées
mesurées avant tout backfill (constat 14).
