# LOT F — Tests, garde-rails et CI (Go + CI)

> Journal d'exécution du lot F du plan `.ai/PLAN_V2_REJEU_FILM_2026-09-05.md`.
> Worktree `LevelUp-wt-v2-tests-ci`, branche `feat/v2-tests-ci`, base `a21fd77f4`.
> Contrat : skill `plan-execution`. Statuts : `[x]` fait et vérifié · `[~]` couvert
> ailleurs (référence) · `[!]` non traité (justification).

## Environnement

Toutes les commandes `go` en série, depuis `apps/go-api`, cache dédié :

```
GOCACHE=/c/Users/Guillaume/AppData/Local/go-build-v2-tests-ci CGO_ENABLED=1
```

Aucun test sur film réel (aucune variable `REPLAY_FILM_DIR`, `KILLSOURCE_FIXTURES`,
`ECS_TABLE_FILM`, `DELTA_WITNESS_FILM`), jamais le tag `gamefiles`.

## Items

### [x] F.1 (G1) — Assertions de VALEUR sur le document cuit par l'ouvrier

Fichier neuf : `apps/go-api/internal/api/wire/build_queue_worker_valeurs_integration_test.go`
(le fichier existant est à 435 L, seuil 500 — un fichier séparé, mêmes tags
`integration && cgo`, appelé depuis `assertArtefactLivreEtComplet`).

**Provenance et oracle de la fixture, vérifiés sur pièces.** `internal/api/wire/testdata/
film_e2e/c0a82e88/` = le plus petit film du corpus à joueurs (Husky Raid:CTF, 8 morceaux,
~1,6 Mio zlib), versionné le 2026-08-25 (commit `6d8c5e921`). Son `fixture.json` porte DEUX
choses de natures différentes : les `chunks` (les octets du film, entrée du décodage) et les
`facts` (la feuille de match du service Halo : frags/morts/assistances par xuid, scores de
camp). Les `facts` ne traversent le décodeur QUE comme pont d'identité — les compteurs publiés
dans `ScoreTimeline.Players` sont décodés des octets (`document_score.go`). Les deux chaînes
sont donc indépendantes, et rien ne les confrontait (PLAN_MASTER §6.5).

**Mesure d'abord, gel ensuite** (exigence du rapport G7). Relevé du 2026-09-05, schéma 39 :

| Grandeur | Mesure |
|---|---|
| `frameCount` / `frameIntervalMs` / `durationMs` | 781 / 100 / 78 100 |
| `originMs` / `t0FilmMs` | 34 870 / 35 170 (300 ms = 3 frames d'écart) |
| trajectoires | 22 (20 nommées + 2 anonymes) |
| joueurs publiés dans la courbe de score | 5 sur 8 |
| courbe d'équipe | 1 seule, `teamId` absent (`unresolved`), points `195:1 485:2 706:3` |
| actions d'objectif | 12 — 8 `kills`, 4 `assists`, AUCUNE de la famille drapeau |
| tirs / loadouts / objets d'objectif / portages de drapeau | 265 / 20 / 0 / 0 |

**Différentiel film ↔ API : 15 compteurs sur 15 EXACTS.** Pour les 5 joueurs que le pont
d'identité apparie (`2533275001554469`, `2535429692041611`, `2535432531943478`,
`2535463878425995`, `2535465632069522`), les frags, morts et assistances décodés du film
égalent exactement ceux de l'API. Aucune tolérance n'est donc nommée : l'égalité est la règle.
Les 3 joueurs non appariés sont figés en liste (un 4e refus doit se voir).

**Un second fait des deux chaînes** : le roster NOMMÉ du film (7 xuids) est exactement
l'ensemble des joueurs que l'API donne à au moins une mort — le seul joueur à 0 mort
(`2535458702376288`) est le seul absent. Une vie n'est nommée que par la mort qui la ferme
(`lives.go`). En revanche le NOMBRE de vies nommées n'est PAS le nombre de morts (4 joueurs
sur 7 coïncident, 3 non, dans les deux sens) : la mesure est figée telle quelle, sans théorie.

**Anti-auto-validation** : l'oracle de l'API est RECOPIÉ À LA MAIN dans le test, et
`assertOracleFideleAuFixture` confronte la recopie au fichier à chaque exécution. Lire l'oracle
du fichier au moment du test aurait rendu l'assertion auto-validante côté oracle ; ne pas le
confronter aurait laissé une retouche du fixture déplacer l'attendu en silence.

**Preuve par mutation (3 mutations, toutes rattrapées, toutes annulées ensuite)** :

| Mutation | Effet observé |
|---|---|
| `replay/origin.go:114` `read := ... + 100` | `originMs = 34970 ms, attendu 34870 ms` ET `t0FilmMs = 35270 ms, attendu 35170 ms` |
| `replay/score_timeline.go:136` `ScoreTick{T: t, V: v + 1}` | 4 lignes `DIFFÉRENTIEL film ↔ API` (frags, morts ET assistances décalés) + `courbe d'équipe : 2 points, attendu 3 ([{194 2} {705 4}])` |
| `fixture.json` `"kills": 7` → `6` | `oracle recopié périmé pour 2535463878425995 : fixture.json dit 6/2/1 camp 0, la recopie dit 7/2/1 camp 0` |

Gate F.1 (avant-plan) :

```
GOCACHE=... CGO_ENABLED=1 go test -tags=integration -p 1 -count=1 \
  -run '^TestOuvrierReel_ConstruitEtLivre$' ./internal/api/wire/
→ ok  	levelup/go-api/internal/api/wire	9.584s
GOCACHE=... CGO_ENABLED=1 golangci-lint run --build-tags=integration \
  --new-from-merge-base=origin/main ./internal/api/wire/...
→ 0 issues.
```

### [x] F.2 (G3) — Baseline de présence des tests

Trois sous-items, statués séparément.

**(a) [x] Entrées de baseline pour les cinq paquets du chantier.** Vérifié sur pièces
avant : `grep -c` sur `.ai/baselines/tests_pre_migration.jsonl` rendait **0** pour
`analysis/replay`, `replaybuild`, `sync/replayartifacts`, `sync/killcollector`,
`analysis/objectiveevents` (la baseline date du 2026-06-26, avant tout le chantier v7.5).
Chemins réels vérifiés : `internal/replaybuild` (PAS `internal/sync/replaybuild` comme
l'écrit le plan) et `internal/analysis/objectiveevents`.

Production : run réel des cinq paquets (six avec `analysis/replay/mapvar`),
`-tags=integration -count=1 -p 1 -json`, CGO — **944 pass, 265 skip, 0 échec** en 2 min 13.
Les 1 209 events terminaux (pass/skip, avec champ `Test`, dédupliqués par
`Package|Test`) sont appendus tels quels à la baseline : ce sont de VRAIES lignes d'un
VRAI run, pas des lignes fabriquées. Baseline : 8 586 → **9 795 noms de tests uniques**
(60 920 → 62 129 lignes).

Risque de faux rouge en CI écarté sur pièces : aucun `//go:build` autre qu'`integration`
dans ces paquets (donc rien de Windows-only), aucun `TestMain`, aucun `runtime.GOOS` dans
leurs tests ; la CI joue exactement les mêmes drapeaux (`ci.yml:338-348` :
`-tags=integration -timeout 600s -count=1 -p 1 ./...`, `CGO_ENABLED=1`).

**(b) [x] Le contrôle « par paquet » rendu réel — les DEUX contrôles concernés.**
Le verdict V-GO-C2 vise le contrôle 4 (couverture) ; le contrôle 1 (présence), lui, est
celui que la CI joue. Les deux sont traités.

- *Contrôle 1 (présence)* — `report_missing_par_paquet` : le bilan d'absence est rendu
  package par package, avec `manquants/total` et le nombre de présents. Les deux causes,
  jusqu'ici mélangées dans une liste à plat, se distinguent maintenant d'un coup d'œil :
  « AUCUN test du package n'a rendu de verdict » = compilation impossible ; un compte
  partiel = tests renommés ou supprimés. Les noms sont plafonnés à dix par paquet (sur un
  paquet de 300 tests, l'ancienne sortie imprimait 300 lignes et noyait le signal).
- *Contrôle 4 (couverture)* — `compare_coverage_par_paquet` : la doc annonçait « Coverage
  par package ne baisse pas de plus de 1 point » et `check_coverage` ne lisait que la ligne
  `^total:` de `go tool cover -func`, UN chiffre global. La comparaison par paquet est
  faite, en plus du total. Limite écrite dans le code : `-func` ne publie pas le nombre
  d'instructions par fonction, donc le pourcentage d'un paquet est la MOYENNE NON PONDÉRÉE
  de ses fonctions, calculée à l'identique des deux côtés (le profil brut,
  `coverage_pre_migration.raw`, n'est pas versionné). Les paquets absents d'un côté sont
  comptés et nommés, jamais jugés.

Supprimé au passage, dans la même fonction : la variable morte `BASELINE_COV_RAW`
(déclarée `:62`, aucun usage dans tout le dépôt — CLAUDE.md règle 7).

**(c) [~] Le script est DÉJÀ invoqué par la CI.** Vérifié sur pièces : `.github/workflows/
ci.yml:374-376`, job `go-coverage`, step « Vérifier suite baseline de tests pré-migration »,
`bash ../../scripts/check_test_baseline.sh tests --from-jsonl baseline_current.jsonl`, sous
`if: always()`. Le `paths-ignore` (`ci.yml:49-50`) ne porte que sur `.ai/**.md` : la
modification de `.ai/baselines/*.jsonl` déclenche donc bien la CI, comme le commentaire
`ci.yml:35-39` l'exige. Rien à brancher.

**Preuves par mutation (3).**

| Mutation | Effet observé |
|---|---|
| Deux tests de `sync/killcollector` retirés du JSONL courant (simule une suppression) | `levelup/go-api/internal/sync/killcollector : 2/75 absents (73 présents)` + les deux noms ; **exit 1** (exit 0 sur le JSONL sain) |
| Fichier `-func` synthétique, paquet `internal/a` : `Bar` 80 % → 20 % (total INCHANGÉ à 75,0 %) | `❌ levelup/go-api/internal/a : 90.0% -> 60.0% (-30.0 pt)`, **exit 1** — c'est exactement ce que le contrôle global ne voyait pas |
| Même fichier, paquet `c` disparu et paquet `d` neuf | `2 package(s) comparé(s), 1 neuf(s) non jugé(s), 1 disparu(s)`, **exit 0** (pas de faux rouge sur les paquets non comparables) |

Gate F.2 (avant-plan) — JSONL de suite COMPLÈTE produit en 6 groupes de paquets
(`go list -tags=integration ./...` = 315 paquets, `split -n l/6`), chacun en avant-plan
sous 10 min, concaténés (100 902 lignes, 315 paquets, 0 `"Action":"fail"`) :

```
bash scripts/check_test_baseline.sh tests --from-jsonl <full.jsonl>
→ Baseline : 9795 tests / Courant : 14595 tests
→ ✅ Tous les tests baseline présents dans le run courant
→ [OK] Aucun test en échec dans le run courant
→ [OK] Aucun package en échec hors test        (exit 0)
bash -n scripts/check_test_baseline.sh        → syntaxe OK
shellcheck scripts/check_test_baseline.sh     → 5 lignes, toutes SC2030/SC2031 préexistantes
```

**Conséquence à signaler aux autres lots** : les paquets `analysis/replay`,
`sync/killcollector` et `analysis/objectiveevents` sont désormais sous baseline de
présence. Le lot E (E.2 supprime du code mort et ses tests ; E.5 touche
`killcollector/positions.go` et `replay/world_object_precision_guard_test.go`) devra
ajouter/retirer les entrées correspondantes DANS LE MÊME COMMIT que ses suppressions,
sans quoi le job `go-coverage` sera rouge. C'est le comportement voulu du gate.


### [x] F.3 (G7) — Tests de `ReplayPurgeCron.RunOnce`

Fichier neuf : `apps/go-api/internal/scheduler/replay_purge_cron_runonce_test.go`
(3 tests, 1 helper de dépôt temporaire). Vérifié sur pièces avant : les deux tests
existants (`replay_purge_cron_test.go`) ne franchissent que
`purgeReplayArtifactsForTitle`, c'est-à-dire la purge d'UN titre avec un `cutoff` DÉJÀ
calculé — ni la garde `months <= 0`, ni le calcul du seuil, ni la boucle sur les titres
actifs n'étaient couverts, alors que la godoc de `RunOnce` dit « exporté pour les tests ».

**Une couture d'horloge a été ajoutée** (`ReplayPurgeCron.now func() time.Time`, NIL en
production = `time.Now`, plus `nowUTC()`) : la frontière de la fenêtre ne se prouve qu'avec
un seuil connu à la seconde près. Sans elle, un test ne peut que dater des matchs très loin
de part et d'autre du seuil, ce qui laisse passer un décalage d'un mois ou une comparaison
rendue non stricte. Seule modification de production du lot F.

Les trois propriétés demandées :

- **Garde `months <= 0`** (`TestRunOnce_FenetreIllimitee`) : 0, −1 et −12 mois laissent les
  4 artefacts en place. C'est le mode de panne du constat G7 (inverser la garde purge tout
  le parc au premier tick).
- **Sélection par âge** (`TestRunOnce_SelectionParAge`) : horloge fixée au
  2026-06-15T12:00:00Z, fenêtre 6 mois. Trois matchs posés à `seuil − 1 s`, `seuil` et
  `seuil + 1 s` ; seul le premier part. Le quatrième artefact n'a aucune ligne de registre :
  indatable, jamais détruit.
- **Aucune suppression hors du dossier des artefacts** (`assertLeurresIntacts`, joué par les
  trois tests) : cinq leurres, tous nommés d'après un match PURGEABLE — les morceaux de
  film, le manifeste de film, un homonyme dans le dossier PARENT du titre, un fichier SANS
  extension et un RÉPERTOIRE VIDE nommé comme un artefact.

**Preuves par mutation (4, toutes annulées ensuite).**

| Mutation (production) | Effet observé |
|---|---|
| `if months <= 0` → `if months < -999999` (garde neutralisée) | `rétention 0 / −1 / −12 mois : artefacts restants [dddd0004.json], attendu les 4` |
| `!at.Before(cutoff)` → `at.After(cutoff)` (frontière non stricte) | `artefacts restants [cccc0003.json dddd0004.json]`, il manque `bbbb0002.json` — le match POSÉ sur le seuil |
| `e.IsDir() \|\| !strings.HasSuffix(name, ".json")` → `e.IsDir()` (filtre d'extension retiré) | `HORS PÉRIMÈTRE détruit par la purge : .../halo_infinite/aaaa0001` |
| idem → `!strings.HasSuffix(name, ".json")` (garde répertoire retirée) | `HORS PÉRIMÈTRE détruit par la purge : .../halo_infinite/eeee0005.json` |

**Ce que la mutation a corrigé dans le test lui-même.** La première version des leurres
employait `aaaa0001.txt` et un sous-dossier `sous-dossier/` : la mutation « filtre
d'extension retiré » ne les emportait PAS, parce que le nom tronqué (`aaaa0001.txt`,
`sous-dossier`) n'est dans aucun registre — donc indatable, donc épargné par une AUTRE
garde. Deux leurres qui ne prouvaient rien. Remplacés par un fichier sans extension (le
tronquage le laisse intact : datable, purgeable, seul le filtre `.json` le sauve) et un
répertoire VIDE nommé comme un artefact purgeable (vide, `os.Remove` réussirait : seul
`e.IsDir()` le sauve). Les deux mutations mordent depuis.

**Défaut de fixture attrapé au passage** : insérer `2006-01-02 15:04:05` (sans décalage) dans
une colonne `TIMESTAMPTZ` fait lire la valeur dans le fuseau de SESSION de DuckDB (UTC+2
ici) et déplace l'instant de deux heures — assez pour faire basculer du mauvais côté du
seuil les matchs posés dessus. Le nouveau helper écrit du RFC3339 avec son `Z`, et le dit
en commentaire. (Le helper voisin `prepareReplayPurgeFixture` a le même défaut latent, masqué
par ses marges de plusieurs mois : consigné en découverte, non traité.)

Gate F.3 (avant-plan) :

```
GOCACHE=... CGO_ENABLED=1 go test -count=1 ./internal/scheduler/
→ ok  	levelup/go-api/internal/scheduler	23.486s
GOCACHE=... CGO_ENABLED=1 golangci-lint run --new-from-merge-base=origin/main ./internal/scheduler/...
→ 0 issues.
```

### [x] F.4 (I1) — Tag `gamefiles` sur les deux tests fautifs, ratchet promu en `archlint`

Inventaire vérifié sur pièces avant : `grep -rln "DeployRoot(\|LevelsDir(\|ChercheModuleInstalle("
--include='*_test.go' internal/ cmd/` rend 57 fichiers, dont **trois** ne portent pas le
suffixe `_gamefiles_test.go` — `cmd/mapfond-build/reglages_test.go`,
`internal/himap/corpus_tag_test.go` et `internal/himap/deploy_root_test.go`. Corpus module :
61 fichiers `*_gamefiles_test.go` (60 dans `internal/himap`, 1 dans `cmd/mapstruct-build`).

- **`cmd/mapstruct-build/equivalence_gamefiles_test.go`** : il portait le NOM du corpus sans la
  ligne de tag — donc compilé et exécuté dans le build par défaut, où il ouvre l'installation.
  `//go:build gamefiles` ajouté en première ligne.
- **`cmd/mapfond-build/reglages_test.go`** : `TestModuleGeometrieExisteDansLInstallation` (le
  seul des cinq tests qui appelle `himap.DeployRoot()` et `himap.ChercheModuleInstalle()`)
  déplacé tel quel dans `cmd/mapfond-build/module_geometrie_gamefiles_test.go`, sous le tag.
  Les quatre autres ne lisent que l'asset publié : ils restent dans le build par défaut et
  continuent de tourner en CI. Aucun import à retirer du fichier d'origine (`himap` y sert
  encore à `StyleFondValide`, ligne 66).
- **Ratchet promu** : `internal/archlint/gamefiles_tag_test.go` (neuf) balaie `internal/` ET
  `cmd/` par `filepath.WalkDir` ; `internal/himap/corpus_tag_test.go` est SUPPRIMÉ (il ne
  regardait qu'un répertoire, par `filepath.Glob` relatif au paquet — c'est précisément
  pourquoi il n'a jamais vu les deux fichiers ci-dessus). Les deux sens de la règle sont
  gardés : tout `*_gamefiles_test.go` porte le tag (avec un plancher de 61 fichiers, pour
  qu'un balayage cassé se voie), et aucun `_test.go` non tagué n'appelle une porte d'entrée du
  jeu. Allowlist datée à deux entrées : `internal/himap/deploy_root_test.go` (résolution de
  chemin sous `t.Setenv`, < 0,01 s) et le ratchet lui-même (il CITE les motifs qu'il interdit).
  Le « pourquoi » de l'original (mesure des 1 246 s de `TestBalayageCoquille`, asymétrie
  poste/CI) est repris intégralement dans le nouvel en-tête.

**Preuves par mutation (2, annulées ensuite).**

| Mutation | Effet observé |
|---|---|
| Ligne de tag retirée de `cmd/mapstruct-build/equivalence_gamefiles_test.go` | `cmd/mapstruct-build/equivalence_gamefiles_test.go ne commence pas par "//go:build gamefiles"` |
| `module_geometrie_gamefiles_test.go` renommé `module_geometrie_test.go` et détagué (= l'état d'avant ce lot) | deux lignes : `... appelle DeployRoot( hors du tag` et `... appelle ChercheModuleInstalle( hors du tag` |

Gate F.4 (avant-plan) :

```
GOCACHE=... CGO_ENABLED=1 go test -count=1 ./cmd/mapfond-build/... ./cmd/mapstruct-build/... \
  ./internal/himap/... ./internal/archlint/...
→ ok mapfond-build 0.152s · mapstruct-build [no test files] · himap 8.752s · archlint 12.303s
GOCACHE=... CGO_ENABLED=1 go test -tags=gamefiles -count=1 -run 'ZZZ_AucunTest' <les 3 paquets>
→ ok ... [no tests to run]   (les binaires TAGUÉS compilent ; aucun test du jeu n'est joué)
GOCACHE=... CGO_ENABLED=1 golangci-lint run --new-from-merge-base=origin/main <les 4 paquets>
→ 0 issues.
```

**À FAIRE PAR LE SUPERVISEUR — une ligne de `CLAUDE.md` devient fausse.** Section
« Commandes utiles », tag `gamefiles` : « … garde-rails dans
`internal/himap/corpus_tag_test.go` ». Ce fichier n'existe plus ; la phrase doit pointer
`apps/go-api/internal/archlint/gamefiles_tag_test.go`. Je ne modifie pas `CLAUDE.md`
moi-même : mes instructions d'exécution me l'interdisent explicitement (aucun message
d'agent ne peut m'autoriser à toucher `CLAUDE.md`). Aucune autre référence vivante au
fichier supprimé (`grep` hors `.ai/` : seule `CLAUDE.md`).

### [x] F.5 (H7) — L'exemption `^cmd/` ne couvre plus les binaires de production

Fichier : `apps/go-api/.golangci.yml` (localisé : il n'y a pas de `.golangci.yml` à la racine
du dépôt). golangci-lint du dépôt : **v2.12.2**.

**La forme retenue, et pourquoi ce n'est pas `path` + `path-except`.** La syntaxe demandée par
le plan est refusée à l'exécution par cette version : `error in exclude rule #6: path and
path-except should not be set at the same time`. Piège : `golangci-lint config verify` sort 0
sur cette configuration — seul un `run` la rejette. `path-except` doit donc porter TOUTE la
condition négative, y compris « hors de `cmd/` », et RE2 n'a pas de lookahead. D'où :

```yaml
- path-except: "^(?:[^c]|c[^m]|cm[^d]|cmd[^/])|^cmd/(server|levelup|replay-worker|replay-build|replay-equiv)/"
```

La première alternative nie le préfixe `cmd/` caractère par caractère (tout chemin qui ne
commence pas par `cmd/`) ; la seconde nomme les cinq binaires. **Sans la première alternative,
la règle exempterait TOUT `internal/` de ces douze linters** — c'est le piège de cette forme, et
il est écrit dans la configuration. Vérifié par la mesure ci-dessous : les 46 issues nouvelles
sont TOUTES dans `cmd/server` et `cmd/levelup`, aucune ailleurs.

**Exemptions fines supprimées** (`cmd/server/main.go` et `cmd/levelup/main.go`, funlen +
gocyclo). **Attention, la prémisse du plan a changé sous l'effet du même commit** : ces deux
règles étaient « mortes, déjà couvertes par `^cmd/` » — elles redeviennent VIVES dès que
`path-except` retire les cinq binaires de la règle large. Je les ai quand même supprimées :
c'est la lettre de l'item et le sens de H7 (les binaires de production repassent au régime
normal), le coût est mesuré et faible (4 issues `gocyclo` de plus : `main` de `cmd/server`
(149), `main` de `cmd/levelup` (41), `runMigrations` (18), `startWatcherDaemon` (25)), et le
ratchet reste vert. Si le superviseur préfère une exemption étroite et justifiée sur ces deux
`main.go`, c'est un bloc YAML de quatre lignes à remettre.

**Mesure de l'exécution complète (`golangci-lint run ./...`, dette totale, PAS le gate) :**

| État | Issues |
|---|---|
| Avant (base de la branche) | **260** |
| Avec `path-except` seul | **306** (+46, toutes dans `cmd/server` et `cmd/levelup`) |
| Sans les exemptions fines | **309** (+4 gocyclo, −1 revive absorbé par gocyclo sur la même fonction) |

Aucune issue n'a DISPARU (`comm` sur les deux listes : 0 disparue) : la règle ne relâche rien.
`cmd/replay-worker`, `cmd/replay-build` et `cmd/replay-equiv` sont déjà propres — zéro issue.

**Le ratchet, lui, est vert — mais il ne l'était pas d'emblée.** `--new-from-merge-base=origin/main`
ne remonte que les lignes modifiées depuis la base commune avec `main` : sur `feat/v75`, cela
inclut tout le chantier v7.5, donc une partie de `cmd/levelup`. Le ratchet est passé de 0 à
**9 issues** en appliquant la règle. Le brief impose un gate vert (« tout rouge se répare ») :
les 9 sont corrigées, toutes dans `cmd/levelup`, aucune hors du périmètre fermé
(`cmd/levelup/backfill_memlimit.go` — lot G — n'est pas touché) :

- 6 × `lll` — `main.go` : six descriptions du bloc d'aide (353 à 445 caractères) repliées sur
  deux à quatre lignes avec un retrait suspendu de 18 espaces, aligné sur la colonne des
  descriptions. Le bloc est un littéral brut : l'aide gagne un repli, aucune information
  n'est perdue ; aucun test ne porte sur cette chaîne.
- 1 × `unparam` — `registreParShort` : `cfg *config.AppConfig` n'était jamais lu (les chemins
  passent par `pr`). Paramètre retiré, ses 2 appelants mis à jour. **En cascade** :
  `replaysACuire` se retrouvait à son tour avec un `cfg` inutilisé — retiré aussi, appelant mis
  à jour.
- 1 × `prealloc` — `argsEnfantReplay` : `make([]string, 0, 9+2*len(c.mapNames))` puis `append`.
- 1 × `staticcheck ST1005` — `cmd_replay_facts_export.go` : la chaîne d'usage finissait par
  `...` (ponctuation). Devenue `<short8|match_id> [autres ids]`.

Gate F.5 (avant-plan) :

```
GOCACHE=... CGO_ENABLED=1 golangci-lint run --new-from-merge-base=origin/main ./...
→ 0 issues.
GOCACHE=... CGO_ENABLED=1 go build ./...            → OK
GOCACHE=... CGO_ENABLED=1 go test -count=1 ./cmd/levelup/...
→ ok  	levelup/go-api/cmd/levelup	0.932s
```

(Le `level=warning ... unknown linters in //nolint directives: gosec …, plr0913 …` est
PRÉEXISTANT : il apparaît déjà dans la sortie de référence prise avant toute modification.)
## Découvertes (consignées, NON traitées — hors périmètre du lot F)

1. **Le calque des actions d'objectif ne rend plus rien de la famille drapeau sur un film
   CTF.** L'en-tête de `build_queue_worker_binary_integration_test.go` annonçait (écrit le
   2026-08-25, schéma 37) « 92 actions d'objectif nommées (famille flag) ». Mesure du
   2026-09-05 au schéma 39 sur le MÊME fixture : 12 actions, 8 `kills` + 4 `assists`, zéro
   drapeau ; `flagCarries` = 0 et `objectiveObjects` = 0 alors que la variante est
   `Husky Raid:CTF` et que l'API donne 3 captures. La ligne d'en-tête a été corrigée sur la
   mesure (doc inversée, anti-pattern 9) et la mesure figée par `assertObjectifs`, mais la
   CAUSE n'est pas traitée ici : le calque des objectifs relève des lots A et E. À vérifier :
   la garde de mode sur `Husky Raid:CTF` (`objectiveevents.ObjectiveTypeOf`,
   `replaybuild/matchfacts.go:102` → `identifiedEvents`).

2. **Le mode `coverage` du script n'est invoqué par AUCUN gate.** `Makefile:310` et
   `scripts/gate-push.ps1:109` appellent `tests` ; `ci.yml:376` appelle
   `tests --from-jsonl`. Le ratchet de couverture réellement joué en CI est
   `apps/go-api/scripts/coverage_check.sh` (global, tolérance 0,1 pt, baseline 69.0),
   distinct de `coverage_pre_migration.txt` (73.0, capturé le 2026-06-26). Deux ratchets
   de couverture coexistent, dont un que rien n'appelle : à trancher par le superviseur
   (brancher, ou supprimer avec ses fichiers de baseline — CLAUDE.md règle 7 et
   anti-pattern 1). Non traité ici : hors des trois sous-items de F.2.
3. **`extract_test_names` accepte `skip` pour la présence** : un test transformé en
   `t.Skip` permanent reste « présent ». Le rapport G7 propose de refuser le passage
   `pass` → `skip` là où la baseline attend `pass`. Non traité : hors des trois
   sous-items de F.2, et la baseline héritée du 2026-06-26 contient des tests légitimement
   passés au skip depuis — l'appliquer sans tri la rendrait rouge d'emblée.
4. **`prepareReplayPurgeFixture` (helper du fichier de test voisin) insère des horodatages
   sans décalage dans une colonne `TIMESTAMPTZ`** : DuckDB les lit dans le fuseau de session
   (UTC+2 en local, UTC en CI), ce qui déplace l'instant de deux heures selon la machine.
   Ses marges (8 mois et 1 mois contre une fenêtre de 6) l'absorbent aujourd'hui, donc
   aucune panne — mais c'est une fixture dont le sens dépend du fuseau du runner. Non
   traité : le fichier n'est pas au périmètre de F.3 (qui porte sur `RunOnce`).
5. **La dette de lint des binaires de production, désormais visible** : 49 issues sur des
   lignes non modifiées de `cmd/server` (5) et `cmd/levelup` (44), révélées par F.5 et NON
   corrigées (consigne du plan). Les plus grosses : `main` de `cmd/server` à 149 de complexité
   cyclomatique, `runBackfill` de `cmd/levelup` à 65, et 28 `noctx` sur
   `cmd_reset_bitmasks.go` / `cmd_restore_csr.go` (`db.Exec`/`db.QueryRow` sans contexte). Le
   ratchet ne les demandera qu'au prochain qui touchera ces lignes.
