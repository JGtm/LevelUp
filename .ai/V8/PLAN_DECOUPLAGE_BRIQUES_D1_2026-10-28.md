# Plan D1 — Découplage des briques : « fichiers du jeu » et « images de carte » — 2026-10-28

Source : `.ai/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md` (axe D1 + garde-fou G + décision 1
« modules Go dans le même dépôt »), complété par la vérification du 2026-10-28 (addendum de
l'audit). Exécution sous le contrat du skill `plan-execution` : ordre strict, un lot à la fois,
gate passé avant le lot suivant, chaque item statué `[x]` / `[~]` (référence) / `[!]`
(justification écrite), zéro fix hors périmètre (section « Découvertes » en fin de fichier).

## Objectif et critère de succès

Deux modules Go sous `libs/`, reliés à `apps/go-api` par un `go.work` à la racine du dépôt :

| Module | Paquets | Importe |
|---|---|---|
| `libs/gamefiles` | `himodule`, `ooz`, `hinavmesh`, `mapvar` | rien du dépôt |
| `libs/himap` | `himap` | `libs/gamefiles` seulement |

Succès = (1) `go list -deps` de chaque module ne montre aucun paquet `levelup/go-api/...` ;
(2) un ratchet `archlint` tient cette liste fermée et son allowlist de départ est vide à la
clôture ; (3) CI verte, image Docker construite, `make go-api-test-gamefiles` vert ; (4) les cinq
fonds déjà publiés recuits sont identiques à l'octet (Dredge WebP, Arrival, Beaver Canyon,
Perdition, Timberland Evolved PNG — protocole du journal du 2026-10-26).

Effort estimé : 3 jours (l'audit disait ~1 jour : il ne comptait ni l'infrastructure multi-module
ni les tests de la brique couplés à LevelUp). Aucune réécriture d'algorithme.

## Branches

- Lot 0 : `fix/ratchet-chemins-ai` depuis `release/v8.1.0` (préalable, PR séparée).
- Lots 1 à 5 : `refactor/decouplage-d1-fichiers-du-jeu` depuis `release/v8.1.0`, worktree
  dédié, N commits (un par lot), fusion par PR dans `release/v8.1.0` (jamais sur `main`
  directement : push `main` = déploiement prod).

## Décisions tranchées avant exécution

Prises par le pilote sur pièces le 2026-10-28 ; l'utilisateur a corrigé D1-A (dépôt JGtm) et
délégué les autres au pilote le 2026-10-28 (message : « je te laisse prendre la meilleure
décision ») — elles sont FERMES :

- **D1-A — Chemin de module** : `github.com/JGtm/LevelUp/libs/gamefiles` et
  `github.com/JGtm/LevelUp/libs/himap` (dépôt principal `origin` ; le remote `fork` ChaseWoodhams
  n'est pas la référence — confirmé par l'utilisateur le 2026-10-28). Raison : la décision 1 de
  l'audit veut des modules « importables de l'extérieur par tag » ; un chemin fictif comme
  `levelup/go-api` ne se résout pas depuis un autre dépôt. `apps/go-api` garde `levelup/go-api`.
  Si le chemin du dépôt change (migration d'organisation), les deux `go.mod` changent avec.
- **D1-B — Deux modules, pas un** : conforme à l'audit (brique de rang inférieur « fichiers du
  jeu » commune à D1 et D2 ; D2 ne doit jamais tirer `himap`, 12 500 lignes).
- **D1-C — `himap` n'importe plus `internal/i18n`** : `himap.LexiqueLieux` rend
  `map[uint32]map[string]string` (code de langue → texte, les codes tels que le jeu les nomme) ;
  le seul appelant, `cmd/mapcallouts-build/main.go:180`, convertit en `i18n.Text`. Oubli de
  l'audit (il affirmait « himap n'importe que himodule, hinavmesh, ooz et mapvar »).
- **D1-D — Tests d'une brique couplés à LevelUp = tests de LevelUp**. Un module séparé ne PEUT
  PAS importer `levelup/go-api/internal/...` (règle Go des `internal/`), tests compris. Les tests
  de `himap` qui lisent les données de LevelUp (`data/titles/.../map_objectives.json` via
  `replay.LoadMapObjectives`, `data/cache/mvar`, `data/cache/replays`, `internal/testfixtures`,
  `internal/testutil`) déménagent dans un paquet de tests seuls
  `apps/go-api/internal/gamefilescorpus/` (tag `gamefiles`, suffixe `_gamefiles_test.go`,
  ratchet `gamefiles_tag_test`). Fichiers concernés (relevé du 2026-10-28, à rouvrir à
  l'exécution) : `carte_ancres_gamefiles_test.go`, `carte_balayage_gamefiles_test.go`,
  `preuve_regions_livefire_gamefiles_test.go`, `rendu_forge_gamefiles_test.go`,
  `sonde_toits_gamefiles_test.go`, `uslg_gamefiles_test.go`, `origine_typage_gamefiles_test.go`,
  `carte_gate_gamefiles_test.go`, `cloture_gamefiles_test.go`, `physique_sonde_gamefiles_test.go`,
  `rendu_gamefiles_test.go`, `sddt_gamefiles_test.go`, `sonde_stse_gamefiles_test.go` (himap, 14) ;
  `hinavmesh/oracle_ancres_test.go` ; `mapvar/cb2_fuzz_test.go`, `mapvar/mapvar_test.go` — la liste
  exacte est l'allowlist d'`archlint/briques_imports_test.go`. S'y ajoutent, sans import LevelUp
  mais lisant `data/` par `cheminDepuisDepot` : `carte_oracle_gamefiles_test.go`, `cle_forge_test.go`,
  `sonde_levelid_cache_gamefiles_test.go`. **Fixtures** : `testfixtures.Carte` pointe
  `internal/testfixtures/testdata/cartes/mapvar/*.mvar`, lues AUSSI par `mapcatalog`, `mapdecoupe`,
  `cmd/levelup`, `cmd/mapobj-build` : elles restent côté LevelUp ; un test du LECTEUR (`mapvar`,
  `hinavmesh`) qui en a besoin reçoit sa copie dans le `testdata/` de la brique (une seule copie
  par fichier, règle 6 : deux copies tolérées), un test de himap couplé aux données déménage. Les tests qui n'ont besoin que du
  jeu installé et de `testdata/` restent dans la brique. Un test qui exerce une fonction NON
  exportée de `himap` et lit des données LevelUp : exporter la fonction si elle est un vrai
  contrat, sinon réécrire le test sur l'API exportée — jamais de copie des données dans la brique.
- **D1-E — Docker et CI passent en mode espace de travail** : `go.work` et `go.work.sum` sont
  versionnés et copiés dans l'image ; aucune directive `replace` dans `apps/go-api/go.mod`
  (`go mod tidy` en mode `go.work` y inscrit une version `v0.0.0-00010101000000-000000000000`
  pour chaque module local : c'est attendu, ne pas la « corriger »).
- **D1-F — Ratchets `archlint` à règle générale** : leur racine devient l'espace de travail
  (`apps/go-api` + `libs/**`) via un helper unique `workspaceGoSources` à côté de
  `goSourcesUnder` (règle 6 : pas de 3e parcours de fichiers). Les ratchets propres au film ou à
  `go-api` restent sur `apiModuleRoot`. Les ratchets propres au décodeur qui citaient `mapvar`
  perdent leur entrée (il n'est plus un paquet du décodeur).
- **D1-G — Non retenu** : renommer la variable d'environnement `LEVELUP_HALO_DEPLOY`
  (`himap/deploy_root.go`). Les briques restent des bibliothèques du projet LevelUp.
- **D1-H — ADR 0041** (EN seulement) : « Go modules under libs/ linked by go.work » — forme
  cible, critère d'autonomie, règle d'import, étiquetage `libs/<brique>/vX.Y.Z`, interdiction des
  données extraites du jeu dans une brique. Rédigé au lot 3, pas après.

---

## Lot 0 — Préalable : CI de la branche de base au vert

Deux constats du 2026-10-28, hors périmètre de D1 mais bloquants pour une CI verte :

- Le ratchet `archlint/doc_chemins_ai_test.go` est ROUGE dans l'arbre de travail local : le
  déplacement de `.ai/PLAN_I18N_MULTILANGUE.md` vers `.ai/V8/` est fait mais NON commité
  (`git status` : D + ??), et deux citations dans le code pointent encore l'ancien chemin. Le
  jour où ce déplacement sera commité sans les citations, la CI rougira.
- La CI de `release/v8.1.0` est déjà ROUGE (trois derniers runs) côté web : garde-rails
  `no-locale-literal-comparison.guard.test.ts` et `no-intl-tag-literal.guard.test.ts` en échec,
  plus des dépassements de 5 s. Cela relève du chantier i18n (ADR 0040), pas de D1 ; mais une PR
  D1 hérite de ce rouge. Un lot ne se clôt pas sur une CI rouge, même héritée (mémoire « tout
  rouge se répare »).

- [x] 0.1 `internal/archlint/i18n_messages_test.go:2` : citer `.ai/V8/PLAN_I18N_MULTILANGUE.md`.
- [x] 0.2 `internal/archlint/no_locale_literal_test.go:10` : idem. Relevé complet du 2026-10-28 : cinq
  autres citations hors Go corrigées dans le même geste (`apps/web/src/lib/i18n/localeKeyedObjects.testutil.ts`,
  `apps/web/src/lib/i18n/no-locale-literal-comparison.guard.test.ts`, `docs/adr/0040-i18n-n-locales.md`,
  `docs/FOUNDATIONS_GUIDE.md`, `docs/FR/FOUNDATIONS_GUIDE.md`) : même cause, même commit.
- [x] 0.3 Commiter le déplacement de `.ai/PLAN_I18N_MULTILANGUE.md` vers `.ai/V8/` AVEC 0.1 et 0.2
  (même commit).
- [ ] 0.4 Gate : `go test ./internal/archlint/ -run TestCheminsAiCitesDansLeCodeExistent -count=1`
  vert (fait le 2026-10-28, worktree `LevelUp-wt-chemins-ai`, branche `fix/ratchet-chemins-ai`,
  7 fichiers + 1 déplacement prêts) ; commit sur accord de l'utilisateur, PR vers `release/v8.1.0`,
  fusion (squash) — RESTE À FAIRE.
- [!] 0.5 Rouge web de la base (dépend du chantier i18n, hors D1 ; constaté le 2026-10-28, voir journal) : : traité dans le chantier i18n (branche à part, pas dans D1) ; D1
  ne pousse son premier lot qu'une fois `gh run list --branch release/v8.1.0 --limit 1` vert,
  ou rebase sa branche sur le correctif.

**Clos quand** : 0.4 fusionné et 0.5 constaté (CI de `release/v8.1.0` verte).

## Lot 1 — Garde-fou G posé AVANT tout déplacement

Fichier : `internal/archlint/briques_imports_test.go`. Une table unique
`brique → {paquets de la brique} + {imports permis hors de la brique}` (règle 6 de CLAUDE.md :
une table, pas un fichier par brique), lue par `goSourcesUnder` au lot 1 et par
`workspaceGoSources` à partir du lot 3. Un import hors liste rougit avec le fichier, la ligne et
le paquet. Allowlist DATÉE des coupes restantes, qui doit finir vide à la clôture du lot 4.

- [x] 1.1 Table des briques à la pose : `gamefiles` = {`internal/himodule`, `internal/ooz`,
  `internal/hinavmesh`, `internal/games/halo_infinite/film/replay/mapvar`}, imports permis hors
  brique = aucun ; `himap` = {`internal/himap`}, imports permis = la brique `gamefiles`.
- [x] 1.2 Allowlist de départ (datée 2026-10-28, chaque entrée avec le lot qui la vide) :
  `internal/himap/uslg.go` → `internal/i18n` (lot 2). Seule coupe de PRODUCTION (vérifié par
  `go list -deps` le 2026-10-28).
- [x] 1.3 Les fichiers de test de la brique sont AUSSI contrôlés (règle D1-D), même allowlist
  datée : SEIZE fichiers de test (relevé par parse des imports le 2026-10-28, et non dix comme
  l'audit le laissait croire) → `film/replay` (5), `i18n` (1), `testfixtures` (8, dont 2 de `mapvar`),
  `testutil` (2, dont 1 de `hinavmesh`) ; tous « lot 2, D1-D ». `himodule` et `mapvar` importés par les tests de `himap` sont permis (même rang ou
  rang inférieur).
- [x] 1.4 Auto-preuve : un extrait synthétique qui importe `internal/analysis` depuis un fichier
  présenté comme `internal/himap/x.go` rougit (patron `parseGoSnippet` existant).
- [x] 1.5 Gate : `go test ./internal/archlint/ -run TestBriquesImports -count=1` vert (3 tests :
  contrôle, auto-preuve, cohérence de table) ; `go vet ./internal/archlint/` vert ; mutation : entrée
  `uslg.go` retirée → rouge « [himap] internal/himap/uslg.go -> levelup/go-api/internal/i18n »,
  entrée remise → vert (2026-10-28).

**Clos quand** : 1.5 passé, commit « test(archlint): garde-fou d'autonomie des briques (D1) ».

## Lot 2 — Coupes dans le module actuel (sans déplacement de paquet)

Tout se fait encore sous `apps/go-api` : les coupes sont séparées du déménagement pour que les
diffs restent lisibles et que la recuisson témoin puisse être rejouée entre les deux.

- [x] 2.1 `himap/uslg.go` : `LexiqueLieux` et `lexiqueDUnTag` rendent
  `map[uint32]map[string]string` ; import `internal/i18n` retiré. Le texte EN/FR lu dans le jeu
  n'a pas à connaître la liste des langues de LevelUp. Fait le 2026-10-28 : les clés sont les
  étiquettes du jeu, constantes exportées `CodeLangueEN = "en-US"` / `CodeLangueFR = "fr-FR"`
  (table `codesLangues` index du jeu -> code ; helper `codeDuSlot`).
- [x] 2.2 `cmd/mapcallouts-build/main.go` : conversion vers `i18n.Text` au point d'appel.
  Fait : `versTextes` (lexique.go) traduit chaque code par `i18n.FromHaloTag` (l'étiquette Halo de
  `config/locales.toml`, aucun repli binaire fr/en) ; un code inconnu est compté et journalisé
  (`slog.Warn`), jamais rangé sous une autre langue. Test : `TestVersTextesTraduitLesCodesDuJeu`.
- [x] 2.3 `himap/uslg_gamefiles_test.go` : adapté au nouveau type (clés `CodeLangueEN/FR`).
  SCINDÉ : les deux tests qui n'exigent que le jeu restent dans la brique ; `TestLexiqueLieuxReproduitLeFichierVersionne`
  (lit `callouts_lexique.csv`, donnée de LevelUp) passe dans `gamefilescorpus/uslg_gamefiles_test.go`
  et compare par `i18n.FromHaloTag(himap.CodeLangue*)`.
- [x] 2.4 `apps/go-api/internal/gamefilescorpus/` créé (tests seuls). RELEVÉ FERMÉ du 2026-10-28
  (le grep du plan est insuffisant : il manquait les fichiers qui n'importent rien mais appellent
  un helper de données défini dans un autre fichier de test ; relevé fait par graphe de
  dépendances des helpers de test + compilation, voir « Découvertes » 1). Sort de chaque fichier :
  - DÉPLACÉS vers `gamefilescorpus/` (`git mv`, API exportée de `himap`) : `carte_ancres`,
    `carte_balayage`, `carte_gamefiles`, `carte_gate`, `carte_livefire`, `carte_oracle`, `carte_trous`,
    `canevas_forge`, `balayage_coquille`, `choix_bsp_gamefiles`, `cuisson_navmesh`, `preuve_regions_livefire`,
    `rendu` (gamefiles), `rendu_carte`, `rendu_forge`, `sonde_bsp_choix`, `sonde_toits`,
    `sonde_levelid_cache`, `origine_typage`, `origine_fosp`, `cle_forge_test`, `chemin_depot_test`
    (copie : l'original reste dans la brique).
  - SCINDÉS : `uslg_gamefiles` (2 tests restent, 1 déménage), `cloture_gamefiles` (`TestClotureCliffhanger`
    lit le champ interne `Rendu.z` : reste ; `TestCoquilleEnXY*` -> `gamefilescorpus/coquille_xy_gamefiles_test.go`),
    `hinavmesh/oracle_ancres_test.go` (`decodeTemoin` reste dans `hinavmesh/temoin_test.go` ; les trois
    `TestOracle*` -> `gamefilescorpus/navmesh_ancres_test.go` ; les deux blobs navmesh, 462 537 octets, avaient été
    copiés dans `gamefilescorpus/testdata/` au lot 2, puis ramenés à UNE copie, celle de `hinavmesh/testdata/`, au lot 2b : 2.10).
  - RESTÉS dans la brique, sans import LevelUp : `sddt` (champs internes `Rendu.z/n`), `physique_sonde`, `sonde_stse`
    (carte de référence `carte_validee_v1.png`, 618 987 octets, dans `himap/testdata/` — copie unique depuis le lot 2b : 2.11 —,
    constante `fixtureCarteValidee`), `filtres_reclaimer`, `sonde_bouillie`, `sonde_forge_saut`, `sonde_locs` (exercent ~35 internes
    non exportés : tags sbsp/instances/forge, index Forge) ; leurs oracles du dépôt hôte (catalogue d'objectifs, rejeu)
    passent par `himap/oracles_depot_gamefiles_test.go` (parseur local, `cheminDepuisDepot`, test sauté si absent).
    Autres restés : `depot_variantes`, `sonde_forge`, `sonde_forge_mvar`, `sonde_canevas_orphelines`,
    `sonde_levelid` (lisent `.ai/re_dump` par la constante exportée `DepotVariantesCarte`, convention de la brique),
    `filtres_reclaimer_offsets`, `maillage_sain`, `callouts`, etc. (jeu installé seul).
  - `mapvar/mapvar_test.go`, `mapvar/cb2_fuzz_test.go` : lisent `testdata/` (les 3 `.mvar` : 195 869 octets, copie unique depuis le lot 2b : 2.9) ;
    plus aucun import `testfixtures`.
  - EXPORT UNIQUE ajouté : `himap.PeupleDepuisModule` (première étape de `CuitCarteNative`, documentée), pour
    `rendu_carte` et `sonde_toits`.
- [x] 2.5 Helpers dupliqués UNE fois (deux copies) : `gamefilescorpus/aides_himap_gamefiles_test.go`
  (gate*, moduleDuJeu, moduleAny, mediane, part, orig*, itoaSimple, indexLevlInstalle, dossiersDistincts),
  copies de `rendu_diff_gamefiles_test.go` et `depot_variantes_gamefiles_test.go` ; côté brique
  `himap/aides_corpus_gamefiles_test.go` (ecritPNG, mesureAncresToits, chainesLisibles). Aucun paquet d'aides commun.
- [x] 2.6 `Makefile` (cible `go-api-test-gamefiles` : `./internal/gamefilescorpus/`), `.github/workflows/ci.yml`
  (`go vet -tags=gamefiles`), `docs/COMMANDS.md` et `docs/FR/COMMANDS.md` mis à jour ; `TestCibleMakefileGamefilesCouvreLeCorpus` vert.
  Ratchets ajustés avec justification datée : `no_repo_root_walk_test.go` (clé de `carte_oracle` déplacée),
  `i18n_locale_constant_test.go` (plafond 39 -> 37, cliquet descendant).
- [x] 2.7 `importsToleres` vidée (map vide, commentaire « VIDÉE LE 2026-10-28 (lot 2) ») ; `TestBriquesImports` vert.
- [x] 2.8 Gate passé le 2026-10-28 (sorties dans le journal) : archlint, himap, hinavmesh, mapvar, mapcallouts-build,
  gamefilescorpus verts ; corpus gamefiles : mêmes issues qu'avant le lot (ensemble de 191 tests comparé à la base :
  9 échecs identiques, voir journal) ; `go list -deps` = himap, himodule, hinavmesh, ooz, mapvar ; témoins avant/après
  du lexique identiques (md5).

- [x] 2.9 Lot 2b, décision du pilote (2026-10-28) : UNE SEULE COPIE par fichier binaire, possédée par la brique qui
  le LIT, lue par LevelUp par chemin depuis la racine du dépôt (jamais `../` entre paquets). `.mvar` (3 fichiers,
  195 869 octets) : copie de `mapvar/testdata/` gardée, copie `testfixtures/testdata/cartes/mapvar/` supprimée (`git rm`) ;
  `testfixtures.Carte("mapvar", nom)` résout vers `testfixtures.MapvarFixturesDir` (constante unique ; au lot 3 :
  `libs/gamefiles/mapvar/testdata`). Lecteurs vérifiés : `cmd/levelup` (backfill callouts), `cmd/mapobj-build` (refresh,
  variant), `internal/mapcatalog`, `cmd/mapcallouts-build` (défaut de `-forge-cache` : `cacheVariantesParDefaut`, chemin
  de la brique, sans importer `testfixtures`) ; `mapdecoupe` ne lit que le JSON de LevelUp (inchangé, reste dans
  `testfixtures/testdata/cartes/`) ; `gamefilescorpus` ne lit pas de `.mvar`. `.gitignore` : règle `mapvar/*` déplacée sur
  `mapvar/testdata/*.mvar` (3 exceptions suivies ; le `fuzz/` n'est pas touché) ; `NOTICE` : chemins corrigés.
  md5 avant = après : `66d06d7be92847c5ff137a3f52c6424e` cliffhanger_map, `278c4e9e196774c941e3053c9bbe852e`
  cliffhanger_ridgeline, `659e35845c2fab7fb4f1a5d82bc9692b` vagabond_fo08_wetland.
- [x] 2.10 Blobs navmesh (2 fichiers, 462 537 octets) : copie de `hinavmesh/testdata/` gardée, copie
  `gamefilescorpus/testdata/` supprimée (`git rm`) ; `gamefilescorpus` les lit par `testfixtures.Navmesh(assetID)`
  (constante `NavmeshFixturesDir`) dans `cuisson_navmesh_gamefiles_test.go` (qui lisait `../hinavmesh/testdata`) et
  `navmesh_ancres_test.go` (`decodeTemoin`). md5 avant = après : `aee118d183ef4107d60aa2a8e3921f16` (01af558d-...),
  `5261e8c36f80389a9cdb16643620ac59` (df7dbf08-...).
- [x] 2.11 `carte_validee_v1.png` (618 987 octets) : copie de `himap/testdata/` gardée (test de la brique
  `calage_banc_test.go`/`fixtureCarteValidee`), copie `testfixtures/testdata/cartes/` supprimée (`git rm`) ; les trois
  tests de `gamefilescorpus` (`carte_gate`, `coquille_xy`, `rendu`) la lisent par `testfixtures.CarteValidee()`
  (constante `CarteValideeFixturesDir`). `domain/title/registry.go`, `replay/map_background.go`, `himap/fond_png.go` ne la
  citent qu'en commentaire : vérifié, non touchés. md5 avant = après : `4bb575ed09ff4fb6f8e7675d0224e193`.
  `testfixtures/testdata/cartes/` ne garde que `callout_zones_ridgeline_clipped.json` (fixture de LevelUp, non déplacée).
  Gate du lot 2b (2026-10-28) : voir le journal.

**Clos quand** : 2.8 vert, allowlist vide, commit « refactor(himap): coupe i18n et tests couplés
à LevelUp sortis de la brique (D1) » (commit à la main du pilote).

## Lot 3 — Module `libs/gamefiles` + espace de travail + infrastructure

Un seul lot parce que la CI casse entre la création du module et l'adaptation de l'infra : rien
n'est poussé entre 3.1 et 3.14.

- [ ] 3.1 `go.work` à la racine (`use ./apps/go-api`, `use ./libs/gamefiles`) ; `go.work.sum`
  versionné ; la ligne `go` des trois `go.mod` égale celle d'`apps/go-api/go.mod` (le ratchet
  `dockerfile_go_version_test` compare l'image Docker à `apps/go-api/go.mod` : l'étendre aux
  `go.mod` de `libs/**`).
- [ ] 3.2 `libs/gamefiles/go.mod` (chemin D1-A), `git mv` de `internal/himodule`, `internal/ooz`,
  `internal/hinavmesh`, `internal/games/halo_infinite/film/replay/mapvar` vers
  `libs/gamefiles/{himodule,ooz,hinavmesh,mapvar}` (paquets au premier niveau, jamais sous
  `internal/`, sinon `go-api` ne peut pas les importer). `ooz/NOTICE.md` et les sources C++
  suivent. `mapvar/testdata/` suit (graines de fuzz, aucun fichier extrait du jeu : vérifier).
- [ ] 3.3 Réécriture des imports dans `apps/go-api` (relevé du 2026-10-28 : `mapvar` importé par
  14 paquets — `cmd/{mapfond-build,mapobj-build,mapopads-build,vehicle-sprite,weapon-icons-build,zone-attribution}`,
  `internal/{archlint,himap,mapcatalog,replaybuild,games/mappings,games/halo_infinite/film/replay,sync/killcollector,sync/replayartifacts}` ;
  `himodule`/`hinavmesh`/`ooz` par `cmd/{mapfond-build,weapon-icons-build,weapon-sounds}` et
  `himap`). `sed` sur le préfixe d'import puis `gofmt`/`goimports` ; `go mod tidy` dans
  `apps/go-api` (version `v0.0.0-...` attendue, D1-E).
- [ ] 3.4 `archlint` — entrées à retirer (mapvar n'est plus un paquet du décodeur) :
  `film_layers_deps_test.go:273-274` (table des couches), `film_file_size_test.go:461`,
  `pas_de_lecture_sous_ai_test.go:71-74`, `no_raw_film_bytes_outside_source_test.go:222`
  (+ le commentaire l.57). `film_layers_deps` R3 reste vrai (`replay` garde ses paquets).
- [ ] 3.5 `archlint` — helper `workspaceGoSources(t, roots...)` (racine = dépôt, roots relatifs
  `apps/go-api`, `libs/gamefiles`, ...). Ratchets à règle générale qui passent dessus (liste
  fermée, chacun rouvert) : `gamefiles_tag_test.go` (il parcourt `goAPIRoot` l.161 : la brique
  porte 60+ tests `gamefiles`, il DOIT continuer à les voir), `research_tag_test.go` (3 tests
  `research` dans `mapvar`), `pas_de_lecture_sous_ai_test.go`, `todo_expiry_test.go`,
  `no_mojibake_test.go`, `no_accidental_goos_suffix_test.go`, `doc_chemins_ai_test.go`,
  `briques_imports_test.go` (lot 1). Tout autre appelant d'`apiModuleRoot` est listé par
  `grep -n 'apiModuleRoot(' internal/archlint/*.go` et annoté dans le journal : « propre à
  go-api » ou « étendu ». Aucun appelant sans verdict.
- [ ] 3.6 Table du lot 1 : les chemins deviennent ceux de `libs/gamefiles/...` ; la règle « rien
  du dépôt » s'écrit « aucun import `levelup/go-api/...` ni `.../libs/himap/...` ».
- [ ] 3.7 `Dockerfile` : `WORKDIR` racine d'espace de travail ; `COPY go.work go.work.sum`,
  `COPY apps/go-api/go.mod apps/go-api/go.sum`, `COPY libs/gamefiles/go.mod` AVANT
  `go mod download` (cache de couches conservé) ; `COPY libs/ ./libs/` avant les `go build` ;
  les `go build` restent lancés depuis `apps/go-api`. Le contexte de build est déjà la racine du
  dépôt (il copie `config/`, `static/`, `docs/`).
- [ ] 3.8 `.github/workflows/ci.yml` : filtre `dorny/paths-filter` (l.137-162) — `go` inclut
  `libs/**` et `go.work*` ; `web` exclut `libs/**`. Ratchet `ci_paths_filter_test.go` à suivre.
- [ ] 3.9 `ci.yml` — jobs Go : `go vet`/`go test` « sans CGo » (l.419, 482) : ajouter un pas par
  module (`go test ./... ` depuis `libs/gamefiles` — `./...` ne franchit jamais la frontière d'un
  module) ; `go vet -tags=gamefiles` (l.450) ; lint `golangci-lint` (l.576, `working-directory`) :
  un pas supplémentaire sur `libs/gamefiles` avec la même config (copie de `.golangci.yml` ou
  `--config ../../apps/go-api/.golangci.yml` — vérifier que les règles `path:` relatives tiennent) ;
  shards (`scripts/ci_go_shard.sh`, `go list -tags=integration ./...` l.51 et `paquets_module.txt`
  l.810) : les paquets déplacés SORTENT du module `go-api` et donc de la baseline JSONL (mémoire
  « test renommé = retrait de la baseline ») — ajouter un pas de test par module dans le job de
  couverture, sans l'inclure dans la baseline du module `go-api`. Ratchets `ci_go_shards_test.go`,
  `ci_web_reads_go_test.go`, `ci_workflows_security_test.go` verts.
- [ ] 3.10 `release.yml` (l.39-62) et `deploy.yml` : vérifier que le `go build` s'exécute dans
  l'espace de travail (checkout complet, pas de sparse checkout de `apps/go-api` seul).
- [ ] 3.11 `Makefile` : `go-api-test`, `go-api-lint`, `go-api-coverage`, `gate-push`,
  `go-api-test-gamefiles` — ajouter les modules `libs/*` là où le périmètre est « tout le Go » ;
  `docs/COMMANDS.md` (bilingue, FR dans le même commit, règle 15) mis à jour.
- [ ] 3.12 `.golangci.yml` : les règles `path:` qui visaient `internal/games/halo_infinite/film/replay`
  n'ont plus `mapvar` dedans (rien à faire si aucune règle ne le nommait — vérifier) ; la dette
  gelée (`--new-from-merge-base`) n'est pas accrue par le déplacement (un fichier déplacé est
  « nouveau » pour golangci : lancer le lint, corriger ce qui sort ou noter l'exemption datée).
- [ ] 3.13 ADR 0041 rédigé (D1-H) + ligne dans la liste des ADR de `CLAUDE.md` + tableau
  « Architecture des Données » inchangé ; `docs/ARCHITECTURE_V6.md` : une section « Bricks /
  libs » (EN + FR).
- [ ] 3.14 Gate : `go build ./...` INTERDIT (mémoire disque) — à la place :
  `cd apps/go-api && go test ./internal/archlint/ ./internal/himap/ ./internal/games/halo_infinite/film/replay/... ./internal/mapcatalog/ ./internal/replaybuild/ ./internal/sync/... -count=1` ;
  `cd libs/gamefiles && go test ./... -count=1` ; `go vet -tags=gamefiles ./internal/himap/`
  depuis `apps/go-api` ; `docker build -t levelup-d1 .` local réussi (ou, si Docker absent du
  poste, le job CI `docker`/`deploy` sur la branche fait foi — à écrire dans le journal) ;
  push, CI verte sur TOUS les jobs.

**Clos quand** : 3.14 vert en CI, commit « refactor(libs): module gamefiles (himodule, ooz,
hinavmesh, mapvar) et espace de travail go.work (D1) ».

## Lot 4 — Module `libs/himap`

- [ ] 4.1 `libs/himap/go.mod` (chemin D1-A, `require .../libs/gamefiles`), `go.work` : `use ./libs/himap`.
- [ ] 4.2 `git mv internal/himap libs/himap/himap` ; imports réécrits dans les 10 commandes
  consommatrices (`cmd/{equipenergy-build,mapcallouts-build,mapfond-build,mapnav-fetch,mapplafond-mesure,mapquant-build,mapstruct-build,vehicle-sprite,weapon-icons-build,weapon-sounds}`)
  et dans `internal/gamefilescorpus/`.
- [ ] 4.3 Infra du lot 3 étendue au module (CI pas par module, Makefile, Dockerfile `COPY`
  déjà `libs/`, `go.mod` version, lint). Aucun nouveau mécanisme : que des entrées.
- [ ] 4.4 Table du garde-fou : `himap` = `libs/himap/himap`, imports permis = `libs/gamefiles/...`.
  Allowlist vide (contrôle : le test échoue si on y ajoute une entrée fictive non utilisée —
  « entrée morte » = rouge, comme les autres ratchets datés).
- [ ] 4.5 Gate : `cd libs/himap && go test ./... -count=1` ;
  `go list -deps ./... | grep -E 'levelup/go-api|libs/himap'` depuis `libs/gamefiles` = vide ;
  `go list -deps ./... | grep 'levelup/go-api'` depuis `libs/himap` = vide ;
  `make go-api-test-gamefiles` vert ; push, CI verte.

**Clos quand** : 4.5 vert, commit « refactor(libs): module himap (D1) ».

## Lot 5 — Preuve de non-régression et clôture

- [ ] 5.1 Recuisson des cinq fonds publiés (protocole du journal 2026-10-26 : variante et
  maillage présents dans `data/cache/`, `mapfond-build` par carte) : Dredge (WebP identique à
  l'octet, sinon MD5 des pixels identique), Arrival, Beaver Canyon, Perdition, Timberland Evolved
  (PNG identiques à l'octet aux fichiers publiés). Un seul écart = lot NON clos, cause à trouver.
- [ ] 5.2 `make go-api-test-gamefiles` complet (himap + corpus + mapfond-build + mapstruct-build
  + film-profiles-build), vert.
- [ ] 5.3 `cd apps/go-api && go test ./... -count=1` (CGO : gcc msys64) vert ;
  `go test -tags=integration -p 1 ./internal/sync/...` (le lot touche `sync/killcollector` et
  `sync/replayartifacts` par leurs imports : la règle « persist/sync touché = tag integration »
  s'applique).
- [ ] 5.4 `make gate-push` vert ; CI de la branche verte (`gh run list --branch refactor/decouplage-d1-fichiers-du-jeu`).
- [ ] 5.5 Audit : addendum « D1 exécuté le … » ; `.ai/REGISTRE_REPORTS.md` alimenté pour tout
  `[!]` ; `thought_log.md` : entrée par lot (déjà écrite au fil de l'eau) + entrée de clôture.
- [ ] 5.6 Hygiène disque (skill `plan-execution` §5) : `scripts/disk-hygiene.ps1 -Lot d1 -Apply`.
- [ ] 5.7 PR vers `release/v8.1.0`, revue adversariale (`adversarial-review`) sur les lots 3 et 4
  (infra CI/Docker = lot à risque : une erreur se voit au déploiement) ; fusion squash sur accord
  de l'utilisateur.

**Clos quand** : 5.1 à 5.7 cochés ; aucune case sans statut.

---

## Témoins avant/après (règle utilisateur du 2026-10-28)

Tout lot se prouve par des sorties capturées AVANT sur le code de base et comparées APRÈS sur le
code modifié : identiques, sinon le lot n'est pas clos. Témoins dans le scratchpad de session
(`temoins_<lot>_avant/`, `temoins_<lot>_apres/`), empreintes MD5 consignées au journal. Le TDD au
sens strict (test rouge écrit avant le code) vaut pour les comportements NEUFS : le garde-fou du
lot 1 a été écrit, vu rouge par mutation, puis vert.

| Lot | Témoin | Capture (avant = code de base, après = branche) | Comparaison |
|---|---|---|---|
| 2 | lexique des lieux lu dans le jeu | `go test -tags=gamefiles ./internal/himap/ -run 'TestLexiqueLieux|TestVocabulaireLieux'` ; avant (2026-10-28) : 810 entrées reproduites à l'identique, 3 tests verts | mêmes 3 tests verts, mêmes comptes |
| 2 | `mapcallouts-build -lexique -out <témoin>/map_callouts.json` (écrit aussi `callouts_lexique.csv` à côté) | avant : `temoins_lot2_avant/` ; après : `temoins_lot2_apres/` | `md5sum` identiques fichier à fichier |
| 2 | `callouts_i18n.csv` versionné | md5 `cb36bdd205b68fb30a00d03a7f6cb648` (2026-10-28) | inchangé (`git status` propre sur `data/titles/halo_infinite/reference/`) |
| 3, 4 | fonds publiés : Dredge (WebP), Arrival, Beaver Canyon, Perdition, Timberland Evolved (PNG) | avant = fichiers publiés sous `data/titles/halo_infinite/reference/map_backgrounds/` (md5 relevés au début du lot 3) ; après = recuisson `mapfond-build` par carte dans `temoins_lot<N>_apres/` | identiques à l'octet (WebP : MD5 des pixels décodés si l'encodeur diffère) |
| 3, 4 | suite `make go-api-test-gamefiles` | avant : liste des tests verts/sautés (`-v`, lot 2) | même liste, aucun test disparu sans déménagement tracé |
| 3, 4 | `go test ./...` du module `go-api` | avant : liste des paquets verts (lot 2) | aucun paquet passé au rouge ; paquets sortis = ceux déplacés dans `libs/` |

## Protocole de reprise de session

1. `git branch --show-current` ; `git log --oneline -10` sur la branche D1.
2. Relire ce plan : le premier item sans statut du premier lot non clos est le point de reprise.
3. Vérifier sur pièces (rouvrir le fichier cité) avant de coder : le code a pu bouger.
4. Journal : `.ai/thought_log.md`, entrées « Découplage D1 — lot N ».

## Ce que D2 et D3 devront reprendre (hors périmètre D1, consigné dans l'addendum de l'audit)

- D2 : `film/replay` importe `internal/i18n` (5 fichiers : `MarshalFlat`, `UnmarshalFlat`,
  `FlatLayout`, `Text`) — le format JSON des libellés du document de rejeu embarque la
  disposition i18n de LevelUp. Absent du tableau de l'audit.
- D2 : déplacer `domain/highlightevent` et `domain/equipmentusage` sous le décodeur ferait
  importer un paquet de titre par `internal/analysis` (lecteurs : `analysis`,
  `analysis/sessionusage`, `analysis/squademprise`, `analysis/trends`) — interdit par
  `archlint/no_title_package_in_analysis_test.go` (ADR 0012). Décision à prendre au plan D2 :
  type de sortie côté décodeur + conversion vers un type LevelUp au point d'entrée, ou
  exemption motivée du ratchet pour une brique (ce n'est plus `games/<slug>`).
- D3 : les fichiers « protocole » de `platform/auth` ne sont pas aussi nets que l'audit le dit :
  `oauth_refresh.go` importe `observability/logging` et `platform/netguard`, `xsts.go`
  `observability/logging`, `xbox_device_code.go` `platform/netguard`. Injection du client HTTP
  et d'un journal à prévoir.
- D2/D3 : mêmes contraintes de module que D1 (pas d'import de `go-api/internal` même en test,
  ratchets à racine d'espace de travail, pas par module en CI) — l'infra du lot 3 est faite pour
  être réutilisée telle quelle.

## Découvertes (notées, non traitées)

Lot 2 (2026-10-28) :

1. Le relevé D1-D du plan (16 fichiers + 3) est très en deçà du réel : l'ensemble des tests de la
   brique couplés à LevelUp par DONNÉES forme une grappe (le banc Cliffhanger : `carte_ancres` porte
   `ancresDuModule`/`cheminCatalogue`, `carte_oracle` porte `chargePositions`, etc.) ; 39 fichiers `himap`
   en dépendaient directement ou par helper. Relevé final : 22 déménagés, 2 scindés + 1 hinavmesh scindé,
   ~19 restés (voir 2.4).
2. RÉSIDU À TRANCHER AVANT LE LOT 4 : cinq sondes de la brique (`sddt`, `physique_sonde`, `filtres_reclaimer`,
   `sonde_bouillie`, `sonde_forge_saut`, plus `cloture` pour `TestClotureCliffhanger` et `sonde_locs`) lisent encore
   des données du dépôt hôte (`data/titles/.../map_objectives.json`, `data/cache/replays/...`,
   `callouts_i18n.csv`) par `cheminDepuisDepot`/chemin relatif, sans importer LevelUp. Elles exercent des
   internes non exportés (~35 : tags sbsp/instances/forge, champs `Rendu.z/n`) que la règle D1-D (« exporter si vrai
   contrat ») ne justifie pas d'exporter. Dans `libs/himap` autonome elles se sauteront (oracle absent) ou
   échoueront (`.ai/re_dump` absent => `t.Fatalf` voulu). Options : les supprimer (sondes de recherche), exporter
   des étapes de cuisson, ou ajouter une variable d'environnement de corpus. Décision de l'utilisateur.
3. Treize fichiers de test de `hinavmesh` et `mapvar` (table `pas_de_lecture_sous_ai_test.go`) et trois de `himap`
   lisent `.ai/re_dump` : même statut que 2 pour le lot 3/4.
4. Casses prévisibles au lot 3/4 (chemins relatifs entre paquets) : `gamefilescorpus/cuisson_navmesh_gamefiles_test.go`
   lit `../hinavmesh/testdata` (utiliser la copie `gamefilescorpus/testdata/` des mêmes blobs) ;
   `himap/deploy_root_test.go` (`TestAucuneAutreCopieDuCheminDuJeu`) balaie `../..` (= `apps/go-api`) :
   à porter dans archlint ou à borner à la brique ; les chemins `../../../../data` de `sonde_locs`.
5. TRAITÉE AU LOT 2b : une copie par fichier, dans la brique. Le commentaire de `mapvar_test.go` refusait de dupliquer
   les `.mvar` (« actifs propriétaires 343/Microsoft ») et le lot 2 avait pourtant dupliqué 6 fichiers (3 `.mvar`,
   2 blobs navmesh, `carte_validee_v1.png`). Décision du pilote : chaque fichier reste dans le `testdata/` de la brique
   qui le lit (`mapvar`, `hinavmesh`, `himap`), LevelUp le lit par `testfixtures.MapvarFixturesDir`,
   `NavmeshFixturesDir` et `CarteValideeFixturesDir` (items 2.9 à 2.11). Au lot 3, ces trois constantes sont les seules
   lignes à changer.
6. `sonde_locs_gamefiles_test.go` contient des chemins absolus périmés (`C:\Users\Guillaume\Projects\LevelUp\.ai\re_dump`) :
   préexistant, non traité.
7. Poste de gate : `.ai/re_dump/mapvar` est ABSENT (même du checkout principal) et `data/cache` ne porte que
   `career-rank-image` ; 7 tests du corpus gamefiles échouent donc par `t.Fatalf` « depot de variantes absent » AVANT
   et APRÈS le lot, et `TestBancCliffhanger` reste le rouge connu (accord 64,4 %). Le gate de non-régression est la
   COMPARAISON des issues test par test à la base (identiques).
8. Le test `TestRenduForgeVagabond`, `TestOracle*` et autres changent de paquet : aucune entrée de la baseline JSONL de
   tests (`.ai/baselines/tests_pre_migration.jsonl`) ne porte `himap`/`hinavmesh` : rien à retirer.
