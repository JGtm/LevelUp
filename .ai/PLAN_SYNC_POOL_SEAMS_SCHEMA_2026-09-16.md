# PLAN — Sync d'un profil sans token propre : pool partout, seams title-owned câblés par toutes les CLI, dérive de schéma `match_registry`

> Créé le 2026-09-16. Branche : `wt/sync-pool`. Worktree dédié :
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-sync-pool` (règle « worktree dédié »).
> Base : `feat/v75` @ `e4a313311`. **Push sur `main` = déploiement prod : interdit.**
>
> **Contrat d'exécution : skill `plan-execution` fait foi.** Ordre strict, une étape à la
> fois, aucun report d'étape exécutable, chaque item statué `[x]` / `[~]` (référence) /
> `[!]` (justification écrite), zéro fix hors périmètre (découvertes en §7). Vérifier sur
> pièces avant de coder et avant de cocher : les numéros de ligne datent du 2026-09-16.
>
> **Interdits absolus pendant l'exécution** : aucun démarrage de serveur, aucune ouverture
> des bases sous `data/` (une autre session les tient pour le chantier du décodeur — un
> second process = violation mono-process ADR 0013). Tout se prouve par tests sur fixtures,
> build, vet, grep.

---

## 1. Constat (vérifié sur pièces le 2026-09-16, premier profil suivi sans token : Nuzzles)

Le pool de tokens (`internal/platform/auth/pool`, README) est conçu pour servir **n'importe quel
joueur** : `PolicyAnyPublic` (tokens à tour de rôle) pour l'historique, les stats, les films,
les CSR ; `PolicyPinnedPlayer` (le token du joueur lui-même) seulement pour les endpoints
soumis à la vie privée (rang de carrière, personnalisation Spartan). Trois défauts, tous
observés sur le sync de Nuzzles (2 023 matchs insérés le 2026-09-16, 65 min) :

**D-A — Trois appelants court-circuitent la doctrine du pool** avec `if !pool.HasPlayer(gt) {
skip }` — un joueur sans *son propre* token est sauté en bloc, alors que seul le rang de
carrière lui est inaccessible :

| Appelant | Ligne (2026-09-16) | Effet |
|---|---|---|
| CLI `sync-delta --all` | `cmd/levelup/cmd_sync.go:138` | `SKIP reason=not_in_pool` |
| CLI `sync-full --all` | `cmd/levelup/cmd_sync.go:298` | idem |
| Auto-sync serveur | `internal/scheduler/auto_sync_run.go:423` (`checkSyncPreconditions`) | jamais synchronisé par le cycle |
| Cron personnalisation Spartan | `internal/scheduler/spartan_customization_cron.go:327` | **légitime** : l'appel suivant est `PolicyPinnedPlayer` (endpoint privé) — CONSERVÉ |

Le commentaire de la CLI (« cas où Discovery n'a rien trouvé pour lui : pas d'env var, pas de
sync_meta ») date du pool-par-joueur d'avant ADR 0023 : périmé. En outre la CLI **mono-joueur**
(`sync-delta/sync-full --gamertag X`) n'utilise pas le pool : `haloTokensForPlayer` (9
appelants dans `cmd/levelup`) exige le refresh token de X.

**D-B — Le post-sync des CLI panique à l'étape LUSR.** `cmd/server/main.go:1697-1725` câble
huit *seams* title-owned au boot (provider des étapes de migration, racine des jalons H5,
`halo5migrations.Register`, traductions de rangs, classifiers LUSR Infinite + H5, classifiers
de famille objectif Infinite + H5). `cmd/levelup/main.go:65` n'en câble qu'un
(`SetTitleStepsProvider`), `cmd/backfill_all/main.go` aucun. Résultat observé :
`post-sync: PANIC récupéré … classifier LUSR non câblé` (`internal/sync/skill/skill_chain_provider.go:63`,
fail-loud MT-15) → `perf_scores=0 lusr=0 citations=0 dominance=0` sur toute la passe. Tout
sync CLI a ce trou depuis MT-15 ; les `backfill` le masquaient.

**D-C — Dérive de schéma `match_registry`.** La DDL du code déclare `team_0_score` /
`team_1_score INTEGER` (`internal/games/halo_infinite/migrations/steps_shared_core.go:56-57`,
`internal/sync/schema.go`), la base réelle porte **`SMALLINT`** (lu sur la sauvegarde
`data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb` — `information_schema.columns`).
Deux matchs de Nuzzles rejetés à l'INSERT (`Type INT64 with value 120267 … INT16` ;
`26764b9c-4b98-43f1-8d32-73abb3e5a3c8`, `7a5af767-0b8e-4d81-9e75-071359e3736a`) : scores
d'équipe > 32 767 (Baptême du feu). Tout match à gros score d'équipe est **perdu pour tous
les joueurs**. La colonne `player_count SMALLINT` n'est pas en cause (jamais renseignée côté
Infinite, `grep "PlayerCount ="` → seulement `halo_5/ingest/collect.go`, `openspartan/mapper`).

## 2. Décisions tranchées

- **D1** — Un profil suivi sans token propre se synchronise par le pool (`PolicyAnyPublic`).
  Les seuls appels qui exigent le token du joueur (`PolicyPinnedPlayer`) se **dégradent par
  endpoint** : rang de carrière sauté avec un `slog.WarnContext` unique par passe et
  `career_synced=false` (déjà le comportement quand l'endpoint refuse) ; le cron de
  personnalisation garde son `HasPlayer` (exemption datée en commentaire).
- **D2** — La CLI mono-joueur passe par le pool comme `--all` et le serveur. Plus aucun chemin
  n'exige le token du joueur synchronisé. `haloTokensForPlayer` disparaît si plus d'appelant
  (0 code mort) ; sinon ses appelants restants sont listés et justifiés (`token-capture`,
  `sync-achievements` si l'endpoint est privé — à vérifier sur pièces).
- **D3** — Les seams title-owned sont câblés par **une fonction unique**
  `titleseams.RegisterAll(prestigeConfigDir string)` (nouveau paquet
  `internal/games/titleseams`, sans logique : uniquement les appels `Set*`/`Register()` du bloc
  serveur) appelée par **tout `package main` qui importe `internal/sync`**. Ratchet archlint.
- **D4** — Dérive de schéma : migration idempotente `widen_match_registry_team_scores`
  (`ALTER TABLE match_registry ALTER COLUMN team_{0,1}_score SET DATA TYPE INTEGER`, gardée par
  `information_schema.columns`), dans `steps_shared_core.go` à la suite des étapes existantes.
  Garde-rail : un test qui construit `match_registry` avec la DDL **legacy** (SMALLINT), joue
  les migrations, et affirme `INTEGER` ; plus la fixture de test (`sync/testutil/fixture.go`)
  alignée sur la DDL courante (mémoire « DDL de test recopiées = dérive indétectable »).
- **D5** — Aucun rejeu de données dans ce plan (les deux matchs rejetés se resynchronisent à
  la reprise du sync de Nuzzles, annexe A). Aucune ouverture de base réelle.

## 3. Étape 0 — Préparation (rapide)

- [x] 0.1 Worktree `../LevelUp-wt-sync-pool` sur `wt/sync-pool` (créé par le pilote) ;
      vérifier `git branch --show-current`. `apps/web` n'est pas touché : pas de `npm install`.
- [x] 0.2 Lire `CLAUDE.md`, ce plan, `internal/platform/auth/pool/README.md`, skills
      `plan-execution`, `arch-rules`. Lire `docs/adr/0023-*.md` (tokens) et `0035-*.md`
      (annuaire : `ProfileGate`, `Onboard`) pour ne pas recroiser leur périmètre.
- [x] 0.3 Baseline : `cd apps/go-api && go build ./... && go test ./cmd/levelup/... ./internal/scheduler/... ./internal/sync/skill/... ./internal/platform/auth/pool/... ./internal/games/halo_infinite/migrations/... ./internal/archlint/...` → code de sortie 0 (noter la durée). `go test ./internal/sync/` seul dure ~500 s à froid : toujours `-timeout 30m`.

**Gate G0** : branche correcte, baseline verte notée dans « Avancement ».

## 4. Étape 1 — Seams title-owned câblés par toutes les CLI (D-B, moyen)

- [x] 1.1 `internal/games/titleseams/titleseams.go` (nouveau) : `func RegisterAll(prestigeConfigDir string)`
      qui exécute, dans cet ordre et sans autre logique, les huit appels du bloc
      `cmd/server/main.go:1697-1725` : `migration.SetTitleStepsProvider(halomigrations.StepsFor)` ;
      `halo5migrations.SetMilestonesSeedRoot(filepath.Dir(prestigeConfigDir))` (si
      `prestigeConfigDir != ""`, même garde que le serveur) ; `halo5migrations.Register()` ;
      `migration.SetCareerRankTranslationsProvider(halomigrations.CareerRankTranslations)` ;
      `syncpkg.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)` ;
      `syncpkg.SetLUSRChainClassifierForTitle(halo5.TitleSlug, halo5.ClassifyLUSRChain)` ;
      `syncpkg.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode)` ;
      `syncpkg.SetObjectiveFamilyClassifierForTitle(halo5.TitleSlug, halo5.IsObjectiveSubMode)`.
      Doc d'en-tête : pourquoi (panic MT-15 sur les CLI, 2026-09-16), qui doit l'appeler.
      Vérifier d'abord qu'aucun import cyclique n'apparaît (`titleseams` importe `sync`,
      `migration`, les paquets `halo_infinite`/`halo_5` ; aucun d'eux ne doit importer
      `titleseams`).
- [x] 1.2 `cmd/server/main.go` : le bloc est remplacé par l'appel `titleseams.RegisterAll(prestigeConfigDir)`
      (les commentaires MT-07/MT-15/D-A déménagent dans le paquet). Comportement byte-identique.
- [x] 1.3 `cmd/levelup/main.go:65` : `migration.SetTitleStepsProvider` remplacé par
      `titleseams.RegisterAll(<racine config prestige résolue comme le serveur — vérifier
      comment `prestigeConfigDir` est calculé dans cmd/server et réutiliser la même source>)`.
      `cmd/backfill_all/main.go` : idem. Tout autre `main` de `cmd/` qui importe
      `levelup/go-api/internal/sync` (lister par `grep -rl '"levelup/go-api/internal/sync"' cmd/`)
      reçoit l'appel — la liste est écrite dans « Avancement ».
- [x] 1.4 Ratchet `internal/archlint/titleseams_wired_test.go` : pour chaque répertoire de
      `cmd/` dont un fichier non-test importe `levelup/go-api/internal/sync` (ou
      `internal/sync/skill`), un fichier non-test du même répertoire contient
      `titleseams.RegisterAll(`. Allowlist VIDE ; message d'échec = le nom du `main` fautif et
      la ligne à ajouter.
- [x] 1.5 Test `internal/games/titleseams/titleseams_test.go` : après `RegisterAll("")`,
      `skill.GetLUSRChain("arena:slayer")` (ou l'appel public exact du provider) ne panique pas
      et retourne une chaîne non vide ; la variante titre H5 est routée (`GetLUSRChainForTitle`).
- [x] 1.6 Test de non-régression CLI : `cmd/levelup/main_seams_test.go` — invoque le même
      chemin de démarrage que `main` (fonction extraite si nécessaire, ≤ 80 L) puis
      `skill.GetLUSRChain(...)` sans panic. Un test qui passe avec ET sans 1.3 est refusé.

**Gate G1** : `go build ./... && go vet ./cmd/... ./internal/games/titleseams/... ./internal/archlint/...` → 0 ;
`go test ./internal/games/titleseams/... ./internal/archlint/... ./cmd/levelup/... ./cmd/server/...` → 0 ;
`grep -rn "SetLUSRChainClassifier(" cmd/` → une seule occurrence hors `titleseams` : AUCUNE
(tout passe par `RegisterAll`).

## 5. Étape 2 — Le pool sert tout profil suivi (D-A, lourd)

- [x] 2.1 CLI mono-joueur : `runSyncDelta` et `runSyncFull` (`cmd/levelup/cmd_sync.go`)
      construisent le pool (`buildCLITokenPool`, déjà utilisé par `--all`) et un
      `go_sync.NewPooledHaloClient(pool, player.Gamertag, player.XUID, 0)` posé par
      `engine.SetCustomClient`, avec `&domain.HaloTokens{}` comme le fait `runSyncDeltaAll`
      (`cmd_sync.go:146-175`). Extraire le code commun `--all` / mono-joueur dans une fonction
      ≤ 80 L (`newPooledEngine(...)`) pour ne pas dupliquer (règle ≤ 2 copies : il y a déjà
      deux copies `--all`, la troisième impose le helper).
- [x] 2.2 Supprimer les deux `if !pool.HasPlayer(...) { skip }` de `cmd_sync.go` (`:138`,
      `:298`) et leur commentaire périmé. `sync full SKIP … no_player_db` reste (une base
      absente est créée par le sync mono-joueur, pas par `--all` : décision inchangée, la
      justifier en commentaire daté).
- [x] 2.3 `haloTokensForPlayer` : lister ses 9 appelants (`grep -n "haloTokensForPlayer(" cmd/levelup/`).
      Ceux qui servent un endpoint **public** passent au pool ; ceux qui servent un endpoint
      **privé** (à vérifier sur pièces dans `internal/sync/pooled_client.go` : `GetCareerRank`,
      et tout appel non listé par le client poolé) gardent le token du joueur avec un
      commentaire daté. Si aucun appelant ne reste : supprimer la fonction et son test.
- [x] 2.4 Auto-sync : `checkSyncPreconditions` (`internal/scheduler/auto_sync_run.go:423-429`)
      perd la précondition `HasPlayer` ; la précondition `pool == nil` reste. Le message
      « authentifier le joueur (SSO Xbox) » disparaît avec elle. Vérifier que `BuildEngine`
      passe bien par `NewPooledHaloClient` pour ce joueur (`auto_sync_engine.go`).
- [~] 2.5 Rang de carrière : dans le client poolé (`pooled_client.go:287-300`,
      `PolicyPinnedPlayer`), l'erreur « n'a pas de token pinné » devient une erreur typée
      `ErrNoPinnedToken` ; l'étape post-sync carrière (`internal/sync/career.go`) la journalise
      **une fois** en `slog.WarnContext(ctx, "career: rang non synchronisé — aucun token propre", "gamertag", …)`
      et pose `career_synced=false` sans compter d'erreur fatale. Vérifier sur pièces qu'aucune
      autre étape du post-sync n'utilise `PolicyPinnedPlayer`.
- [x] 2.6 Cron Spartan (`spartan_customization_cron.go:327`) : `HasPlayer` CONSERVÉ ;
      commentaire daté 2026-09-16 : « endpoint privé (PolicyPinnedPlayer) — seule exemption
      légitime à D1 ».
- [x] 2.7 Ratchet `internal/archlint/no_pool_hasplayer_gate_test.go` : `HasPlayer(` n'apparaît
      hors du paquet `pool` et de ses tests que dans `spartan_customization_cron.go` (allowlist
      datée d'une entrée). Message d'échec : « un profil sans token propre se synchronise par
      le pool (D1, plan 2026-09-16) ».
- [x] 2.8 Tests : (a) `cmd/levelup` : pool de fixture à un slot (JGtm), joueur `X` absent du
      pool → `sync-delta --gamertag X` construit un client poolé (pas d'appel à
      `haloTokensForPlayer`) ; (b) `scheduler` : `checkSyncPreconditions` accepte un joueur hors
      pool quand le pool existe, refuse toujours `pool == nil` ; (c) `sync` : post-sync carrière
      avec `ErrNoPinnedToken` → `career_synced=false`, un WARN, statut `success` ; (d) le test
      existant qui affirmait `not_in_pool` est retourné (documenté dans le test, pas supprimé
      en silence).
- [x] 2.9 Docs bilingues : `docs/COMMANDS.md` (FR + EN dans le même commit) — sémantique de
      `sync-delta/sync-full --gamertag` (pool, aucun token propre requis, rang de carrière
      dégradé) ; `internal/platform/auth/pool/README.md` : section « Appelants » listant les
      trois sites et l'exemption Spartan.

**Gate G2** : `go build ./... && go vet ./...` → 0 ; `go test ./cmd/levelup/... ./internal/scheduler/... ./internal/sync/... -timeout 30m` → 0 ;
`go test -tags=integration -p 1 ./internal/sync/... -timeout 30m` → 0 (`-p 1` non négociable) ;
`grep -rn "HasPlayer(" apps/go-api --include=*.go | grep -v "auth/pool/" | grep -v _test`
→ exactement une ligne (`spartan_customization_cron.go`) ; `grep -rn "haloTokensForPlayer(" cmd/`
→ 0 ou uniquement les appelants justifiés en 2.3.

## 6. Étape 3 — Dérive de schéma `match_registry` (D-C, moyen, à risque : migration shared)

- [x] 3.1 `steps_shared_core.go` : nouvelle étape `widen_match_registry_team_scores`
      (même forme que `add_player_count_to_match_registry`, `:600-602`) : pour chaque colonne
      `team_0_score`, `team_1_score`, si `information_schema.columns.data_type = 'SMALLINT'`
      → `ALTER TABLE match_registry ALTER COLUMN <c> SET DATA TYPE INTEGER` ; sinon no-op.
      Journal `slog.InfoContext` avec les colonnes élargies. Vérifier sur pièces (doc DuckDB
      1.5.5 embarquée, `internal/migration` helpers) la syntaxe et l'existence d'un helper
      `AlterColumnType` ; s'il n'existe pas, l'écrire dans `internal/migration` (≤ 80 L, test).
      **Contrainte ART** : `match_registry` porte une PK (index ART). Vérifier dans
      `internal/migration/append_only_rebuild.go` / ADR 0026 si un `ALTER COLUMN TYPE` sur
      table indexée est admis ; sinon appliquer la recette de reconstruction de l'ADR 0026
      (table neuve + copie + swap) et le dire dans « Avancement ».
- [x] 3.2 `internal/sync/schema.go:196-197` et `internal/sync/testutil/fixture.go` : DDL alignée
      (`INTEGER`) — aucune fixture ne garde `SMALLINT` pour ces colonnes.
- [x] 3.3 Test de migration `steps_shared_core_widen_test.go` (DuckDB `:memory:` ou fichier
      temporaire — JAMAIS une base sous `data/`) : créer `match_registry` avec la DDL legacy
      (`team_0_score SMALLINT, team_1_score SMALLINT`, PK sur `match_id`, quelques lignes),
      jouer la migration, affirmer `INTEGER` × 2, lignes intactes, puis un INSERT avec
      `team_0_score = 120267` réussit. Second passage = no-op (idempotence).
- [x] 3.4 Ratchet anti-dérive `internal/archlint/match_registry_ddl_types_test.go` : les types
      déclarés dans `steps_shared_core.go` (CREATE) et `schema.go` pour `match_registry` sont
      identiques colonne à colonne (parse textuel des deux DDL) — une divergence future entre
      les deux sources échoue.
- [x] 3.5 `no_art_patterns_test.go` : si l'étape ajoute un motif surveillé (UPDATE/ALTER sur
      table critique), l'allowlister avec justification datée — sinon rien.

**Gate G3** : `go test ./internal/games/halo_infinite/migrations/... ./internal/migration/... ./internal/archlint/...` → 0 ;
`go test -tags=integration -p 1 ./internal/persist/... ./internal/migration/... -timeout 30m` → 0.

## 7. Étape 4 — Clôture

- [x] 4.1 Gates complets : `cd apps/go-api && go build ./... && go vet ./... && go test ./... -timeout 30m` → 0 ;
      `go test -tags=integration -p 1 ./... -timeout 30m` → 0 (codes de sortie vérifiés, pas la
      sortie filtrée ; un paquet en FAIL sans `--- FAIL:` se rejoue seul).
- [x] 4.2 `.ai/thought_log.md` : entrée `[2026-09-16]` Complété (constats D-A/B/C, décisions,
      gates, ce qui n'a pas été fait).
- [x] 4.3 `docs/COMMANDS.md` FR + EN (2.9) relus ; `internal/platform/auth/pool/README.md` à jour.
- [~] 4.4 Revue adversariale (pilote, 2 relecteurs : auth/pool + sync/migration) et CI : `[~]`.
- [~] 4.5 Aucun push, aucun merge : décision utilisateur.

Journal de phase : section « Avancement » en fin de fichier (date, étape, gate + code de
sortie, écarts). Reprise : lire cette section puis `git log --oneline -10` dans le worktree.

---

## 8. Découvertes hors périmètre (ne pas traiter ici)

- `sync full SKIP reason=no_player_db` : `--all` ne crée pas la base d'un profil neuf, seul le
  sync mono-joueur le fait (`OpenPlayerDB`). Décision inchangée dans ce plan (2.2).
- Les trois refresh tokens `revoked` (`AADSTS70000` : Chocoboflor, Madina97294, XxDaemonGamerxX,
  constaté au boot du 2026-09-16) — diagnostic ADR 0023 (rotation perdue / RT étranger), pas
  de re-capture. Hors plan.
- `cmd/levelup` : `sync-full --gamertag` téléchargeait des morceaux de film (`downloadBlob …
  filmChunkN 404`) alors que `replay_build_location=off` et que la CLI ne câble pas les
  artefacts de rejeu : identifier l'étape post-sync qui lit les films en CLI (événements de
  surbrillance ?) et documenter ce qu'elle produit.

### Revue adversariale du 2026-09-16 — P2 consignés (réels, hors périmètre, non corrigés)

- Deux fixtures de test recopient encore `team_0_score SMALLINT` / `team_1_score SMALLINT` :
  `internal/migration/steps_shared_rebuild_match_participants_test.go:106-107` et
  `internal/sync/art_rebuild_regression_test.go:110-111`. Sans effet à l exécution (les tests ne
  touchent pas aux scores) ; le ratchet `match_registry_ddl_types_test.go` ne compare que les deux
  DDL de production.
- Toute commande CLI mono-joueur construit désormais le pool COMPLET (`MaxSize` 0) et fait donc
  tourner les refresh tokens de tout le parc ; un serveur qui tournerait en parallèle garderait les
  anciens RT en mémoire (`resolver.go` : `Refresh` relit `r.sources`, non réécrit au re-scan) et
  finirait en `reauth_required` sur N comptes à l expiration du cache. En pratique les deux ne
  cohabitent pas (la CLI sync tient la base partagée en RW → serveur arrêté), mais la classe
  préexistante (`--all`) a une surface multipliée par N.
- `internal/service/career_live_target.go:20-24` affirme, mesuré, que `/careerranks` n est PAS
  soumis au joueur ; la politique `PolicyPinnedPlayer` de `GetCareerRank` repose donc sur une
  prémisse contredite ailleurs dans le dépôt — à trancher par l utilisateur (hors périmètre).
- Dix binaires de `cmd/` dépendent de `internal/sync/skill` TRANSITIVEMENT sans appeler
  `RegisterAll` ; le ratchet ne voit que l import direct. Aucun chemin vers un classifier n a été
  trouvé dans leur code.

## Annexe A — Reprise du sync de Nuzzles (utilisateur + pilote, APRÈS ce plan et quand la base est libre)

1. `replay_build_location` est actuellement à **`off`** dans `app_settings.json` (posé le
   2026-09-16 pour le sync initial) : le remettre à `local` **après** la cuisson (étape 4).
2. `levelup sync-full --gamertag Nuzzles --max-matches 6500 --rps 3` avec la CLI réparée : les
   2 023 matchs connus sont filtrés page par page (~30 s), les ~4 000 restants insérés, le
   post-sync complet joue (LUSR, perf, citations, dominance) — les deux matchs rejetés
   (`26764b9c…`, `7a5af767…`) rentrent grâce à l'étape 3.
3. Rattrapage des enrichissements des 2 023 premiers : `levelup backfill --gamertag Nuzzles --perf`,
   `--citations`, `--csr`, `--lusr` (chaque commande passe désormais par `titleseams`).
4. Films des 200 plus récents : `levelup backfill-killsource --online --limit 200 --gamertag JGtm --dry-run`
   (contrôler que la liste est bien celle de Nuzzles — `--gamertag` = prêteur de token,
   sélection = tout le parc, plus récents d'abord), puis sans `--dry-run`.
5. Cuisson : `levelup backfill-replay --dry-run` puis `--limit 200`, en fond, journal + Monitor.
6. `replay_build_location` → `local` ; redémarrer le serveur (`air` depuis Git Bash ou avec
   `CGO_ENABLED=1` et `C:\msys64\ucrt64\bin` dans le PATH, sinon il sert un binaire périmé).

---

## Avancement

### Étape 0 — Préparation — 2026-09-16 12:00 — CLOSE

- 0.1 `[x]` worktree `LevelUp-wt-sync-pool`, `git branch --show-current` = `wt/sync-pool`, base `e4a313311`. `apps/web` non touché.
- 0.2 `[x]` lus : `CLAUDE.md`, ce plan, `internal/platform/auth/pool/README.md`, ADR 0023 / 0026 / 0035, skills `plan-execution` et `arch-rules`, 5 dernières entrées de `.ai/thought_log.md`.
- 0.3 `[x]` baseline. `go build ./...` → **0**. `go test ./cmd/levelup/... ./internal/scheduler/... ./internal/sync/skill/... ./internal/platform/auth/pool/... ./internal/games/halo_infinite/migrations/... ./internal/archlint/... -timeout 30m` → **0**, 6 paquets `ok` (scheduler 97 s, archlint 34 s, migrations 23 s ; total ~3 min).

**Gate G0** : passé.

### Étape 1 — Seams title-owned câblés par toutes les CLI — 2026-09-16 12:40 — CLOSE

- 1.1 `[x]` `internal/games/titleseams/titleseams.go` : `RegisterAll(prestigeConfigDir string)` (les 8 appels, dans l'ordre du bloc serveur, zéro logique) + `PrestigeConfigDir(repoRoot)` (même calcul que `cmd/server/main.go:461`, pour ne pas recopier le `filepath.Join` dans chaque CLI). Aucun cycle : ni `sync`, ni `migration`, ni `games/halo_*` n'importe `titleseams`.
- 1.2 `[x]` `cmd/server/main.go` : bloc 1706-1735 remplacé par `titleseams.RegisterAll(prestigeConfigDir)` ; imports `halo5migrations` et `skillchain` retirés (devenus inutilisés). Les `ValidateLUSRChainClassifierWired` / `ValidateObjectiveFamilyClassifierWired` du boot restent en place. Comportement identique.
- 1.3 `[x]` `cmd/levelup/main.go` : `wireStartupSeams(cfg)` (fonction extraite pour être exerçable, cf. 1.6) appelle `RegisterAll(PrestigeConfigDir(cfg.RepoRoot))`. **30 autres `main` de `cmd/` reçoivent `RegisterAll("")`** : backfill-csr-history, backfill-team-rounds, backfill-team-scores, backfill_all, backfill_kda_accuracy, backfill_objective_stats, backfill_participation_info, backfill_quit_timestamps, backfill_registry_names, bench-rps, diag_film, diag_film_avail, diag_live_economy, diag_lusr_player, diag_matchstats_dump, diag_perfsim, h5-backfill, h5-csr-match-backfill, h5-enrich, h5-lusr-backfill, h5-lusr-smoke, h5-sync, lusr_v2_canonical_backfill, lusr_v2_phase0, lusr_v2_replay, lusr_v2_squad_estimate, lusr_v2_ttt_batch, probe-world-stats, recompute_perfnote, refresh_golden_fixture.
  **Écart assumé** : ces 30 outils reçoivent une racine de jalons VIDE. Le step `h5_seed_milestone_catalog` la traite en no-op gracieux explicitement prévu (« CLI sans milestones », `internal/games/halo_5/migrations/milestones.go:122`) — c'est leur comportement ACTUEL (aucun d'eux ne posait de racine), donc zéro régression, et ils gagnent les classifiers qui leur manquaient. Seule la CLI `levelup`, qui seed et provisionne, reçoit la vraie racine.
  **Découverte traitée dans le périmètre (imposée par le gate G1)** : 9 binaires posaient déjà les classifiers À LA MAIN, et 7 d'entre eux SANS les variantes par titre h5 (diag_lusr_player, lusr_v2_phase0, lusr_v2_squad_estimate, lusr_v2_ttt_batch, lusr_v2_canonical_backfill, recompute_perfnote, h5-lusr-smoke/backfill/enrich pour partie) — tous les modes h5 y collapsaient dans `arena_slayer` en silence. Ces 4 appels `Set*Classifier*` ont été retirés de `cmd/` ; `RegisterAll` les pose tous, variantes h5 comprises.
- 1.4 `[x]` `internal/archlint/titleseams_wired_test.go`, deux ratchets, allowlist VIDE : (a) `TestBinairesSyncCablentLesSeams` — tout répertoire de `cmd/` dont un fichier non-test importe `internal/sync` (ou un sous-paquet) contient `titleseams.RegisterAll(` ; (b) `TestSeamsPosesUniquementParTitleseams` — aucun fichier de `cmd/` ne pose les 4 seams de classification à la main. `SetTitleStepsProvider` / `SetCareerRankTranslationsProvider` ne sont PAS dans les motifs : 4 binaires qui n'embarquent pas le moteur de sync les posent légitimement seuls (h5-metadata-fetch, h5-read-smoke, seed-rank-translations, snapshot-world-leaderboard) et leur imposer `RegisterAll` y ferait entrer `internal/sync` pour rien — justification écrite dans le test.
- 1.5 `[x]` `internal/games/titleseams/titleseams_test.go` : après `RegisterAll("")`, `GetLUSRChain("BTB:Slayer") == "btb"`, fallback non vide sur préfixe inconnu, `GetLUSRChainForTitle(halo_5, "") == "h5_arena"`, `ValidateLUSRChainClassifierWired() == nil`, famille objectif câblée des deux côtés ; idempotence ; `PrestigeConfigDir`.
- 1.6 `[x]` `cmd/levelup/main_seams_test.go` exerce `wireStartupSeams` (le chemin de démarrage de `main`) après `SetLUSRChainClassifier(nil)`. **Mutation vérifiée** : en remplaçant l'appel par `_ = titleseams.PrestigeConfigDir(cfg.RepoRoot)`, le test ROUGIT (`GetLUSRChain a paniqué …`), puis repasse vert une fois restauré.

**Gate G1** : `go build ./...` → **0** ; `go vet ./cmd/... ./internal/games/titleseams/... ./internal/archlint/...` → **0** ; `go test ./internal/games/titleseams/... ./internal/archlint/... ./cmd/levelup/... ./cmd/server/... -timeout 30m` → **0**, 4 paquets `ok` ; `grep -rn "SetLUSRChainClassifier(" cmd/` (hors tests) → **0 ligne** ; `gofmt -l` sur les fichiers touchés → vide.

### Étape 2 — Le pool sert tout profil suivi — 2026-09-16 13:05 — CLOSE

- 2.1 `[x]` Nouveau fichier `cmd/levelup/pool_engine.go` (la doctrine D1 y est écrite) :
  `buildCLITokenPool` (déplacé depuis `cmd_sync.go`), `newPooledEngine(cfg, provider, pool, player)`
  — source unique des TROIS chemins (mono-joueur, `--all` delta, `--all` full, il y avait deux
  copies) —, `newPooledEngineForPlayer(...)` (pool + moteur + fermeur, pour les commandes
  mono-joueur) et `newPooledClient(...)` (client seul, pour les commandes de films). Toutes ≤ 80 L,
  ≤ 5 paramètres. `cmd_sync.go` passe de 446 à 306 lignes.
- 2.2 `[x]` Les deux `if !pool.HasPlayer(...) { skip }` de `cmd_sync.go` et le commentaire périmé
  (« pas d'env var, pas de sync_meta », antérieur à l'ADR 0023) ont disparu. Le skip
  `no_player_db` reste, avec la justification datée en commentaire aux DEUX endroits.
- 2.3 `[x]` **Les 9 appelants de `haloTokensForPlayer` sont passés au pool ; la fonction est
  SUPPRIMÉE** (0 code mort) : `cmd_sync.go` ×2 (2.1), `cmd_backfill.go` ×4 (CSR et CSR partagé,
  `--all` et mono-joueur — endpoint public `GetMatchSkill`), `cmd_archive_films.go`,
  `cmd_backfill_killsource_online.go`, `cmd_replay_events.go` (chunks de film et d'événements,
  tous `PolicyAnyPublic` — vérifié sur pièces dans `internal/sync/pooled_client.go`). Aucun
  appelant restant : `grep -rn "haloTokensForPlayer(" cmd/` ne rend qu'une MENTION dans le
  commentaire d'en-tête d'un test. Le SEUL `PolicyPinnedPlayer` du client poolé est
  `GetCareerRank` (vérifié ligne à ligne). `archiverUnFilm` prend désormais une interface locale
  `telechargeurDeFilm` (une méthode) au lieu du type concret `*HaloAPIClient`.
- 2.4 `[x]` `checkSyncPreconditions` (`internal/scheduler/auto_sync_run.go`) perd la précondition
  `HasPlayer` et le message « authentifier le joueur (SSO Xbox) » ; `pool == nil` reste. Vérifié
  sur pièces : `BuildEngine` (`auto_sync_engine.go:66-68`) pose `NewPooledHaloClient` pour
  N'IMPORTE quel gamertag dès que `s.pool != nil`.
- 2.5 `[~]` **Écart assumé, la prémisse du plan est périmée.** `ErrNoPinnedToken` est bien créé
  (`internal/sync/pooled_client.go`) et `GetCareerRank` le rend au lieu de `(nil, nil)` — un skip
  muet est indistinguable d'un joueur sans progression. En revanche l'**étape post-sync carrière
  n'existe plus** : elle est DÉCOUPLÉE depuis le 2026-05-14 (`engine_postsync.go`, section 3 —
  « `CareerSynced` reste dans le struct mais n'est plus jamais positionné à true ici ») ; le flux
  XP + Spartan ID est servi par `service.CareerLiveService`. Le WARN unique est donc posé sur le
  SEAM carrière du paquet, `syncCareerRank` (`internal/sync/career.go`), seul consommateur de
  `GetCareerRank` : `ErrNoPinnedToken` → un `slog.WarnContext` + `(nil, nil)`, toute AUTRE erreur
  remonte. `career_synced=false` est déjà l'état par défaut. Couvert par 2.8 (c).
- 2.6 `[x]` `spartan_customization_cron.go` : `HasPlayer` CONSERVÉ, commentaire daté 2026-09-16
  (« SEULE exemption légitime à D1 : l'appel qui suit est PolicyPinnedPlayer »).
- 2.7 `[x]` `internal/archlint/no_pool_hasplayer_gate_test.go` : allowlist d'UNE entrée datée
  (`spartan_customization_cron.go`). **Mutation vérifiée** : allowlist vidée → le test rougit en
  nommant le fichier, puis repasse vert une fois restaurée.
- 2.8 `[x]` Tests : (a) `cmd/levelup/pool_engine_test.go` — pool de fixture à un slot (JGtm) ;
  `newPooledEngine` construit le moteur pour Nuzzles, ABSENT du pool ; le client poolé n'acquiert
  que des leases `PolicyAnyPublic` sur un endpoint public ; le rang de carrière rend
  `ErrNoPinnedToken`. (b) `internal/scheduler/auto_sync_pool_gate_test.go` (test INTERNE, la
  fonction n'est pas exportée) — `checkSyncPreconditions` accepte un joueur hors pool quand sa
  player DB existe, refuse toujours `pool == nil`. (c) `internal/sync/career_no_pinned_token_test.go`
  — `syncCareerRank` dégrade sur `ErrNoPinnedToken` sans erreur, et ne mange AUCUNE autre erreur.
  (d) **Trois tests retournés, aucun supprimé en silence**, chacun documenté dans son en-tête :
  `TestRunOnce_PlayerNotInPool_Skipped` → `..._PlusDeSkipPool` (affirme désormais que la raison
  « absent du pool » n'apparaît plus) ; `TestRunOnce_Parallel_MixedOutcomes_Counted` (la cause de
  skip de SKIP_NOPOOL passe de « hors pool » à « player DB absente ») ;
  `TestConsecutiveZeroInserts_PreservedOnSkipped` (le skip est désormais provoqué en retirant la
  player DB). Les deux tests de `pooled_client_test.go` sur la carrière attendent maintenant
  `ErrNoPinnedToken` au lieu de `(nil, nil)`.
- 2.9 `[x]` `docs/COMMANDS.md` **et** `docs/FR/COMMANDS.md` (même commit) : bloc « aucun joueur n'a
  besoin de son propre jeton », exception du rang de carrière, et la note que `--gamertag` nomme le
  joueur traité et non un prêteur de jeton. `internal/platform/auth/pool/README.md` : section
  « Callers » (tableau appelant × politique × besoin d'un token propre) + le rappel de l'état
  d'avant.

**Gate G2** : `go build ./...` → **0** ; `go vet ./...` → **0** ;
`go test ./cmd/levelup/... ./internal/scheduler/... ./internal/sync/... ./internal/archlint/... -timeout 30m`
→ **0**, 15 paquets `ok` (sync 67 s, scheduler 31 s, archlint 20 s) ;
`go test -tags=integration -p 1 ./internal/sync/... -timeout 30m` → **0**, 12 paquets `ok`
(sync 159 s, replayartifacts 26 s, killcollector 14 s) ;
`grep -rn "HasPlayer(" apps/go-api --include=*.go | grep -v "auth/pool/" | grep -v _test` → **1 ligne**
(`spartan_customization_cron.go:331`) ; `grep -rn "haloTokensForPlayer(" cmd/` → **1 mention**, dans
le commentaire d'en-tête de `pool_engine_test.go` (aucun appel, aucune définition).

**Écart de procédure à signaler** : les fichiers de l'étape 3 (helper `AlterColumnTypeIfNeeded`,
étape `widen_match_registry_team_scores`, alignement des DDL) ont été écrits pendant l'attente du
gate G2 et étaient donc dans l'arbre lors de sa passe d'intégration. Aucune des deux étapes ne
touche les mêmes fichiers ; le gate complet de l'étape 4 rejoue tout.

### Étape 3 — Dérive de schéma `match_registry` — 2026-09-16 13:45 — CLOSE

- 3.1 `[x]` Étape `widen_match_registry_team_scores` (`steps_shared_core.go`, fonction
  `widenMatchRegistryTeamScores` juste après `add_player_count_to_match_registry` dans l'ordre ;
  nom ajouté à `migration.canonicalOrder` — `order_audit_test.go` l'exige des DEUX côtés, ce que
  la première passe a fait rougir). Gardée par `information_schema.columns` : no-op si la table
  est absente ou si la colonne est déjà INTEGER ; `slog.InfoContext` nommant les colonnes
  élargies. **Le helper `AlterColumnTypeIfNeeded` n'existait pas** : écrit dans
  `internal/migration/helpers.go` (+ `columnDataType`), exporté dans `helpers_export.go` selon la
  convention du fichier, 2 tests d'intégration (élargissement + idempotence, table/colonne
  absente).
  **Question ART tranchée SUR PIÈCES, pas par supposition** : `ALTER TABLE match_registry ALTER
  COLUMN team_0_score SET DATA TYPE INTEGER` sur une table PORTANT SA PK (index ART sur
  `match_id`) passe en DuckDB 1.5.5 embarquée — vérifié par le test 3.3, qui construit exactement
  la DDL legacy (PK + SMALLINT) et relit les lignes après coup. La recette de reconstruction de
  l'ADR 0026 (table neuve + copie + swap) n'a donc PAS été nécessaire.
- 3.2 `[x]` `internal/sync/schema.go` et `internal/sync/testutil/fixture.go` : `team_{0,1}_score`
  passent à INTEGER, avec le commentaire qui dit pourquoi (leçon « DDL de test recopiées = dérive
  indétectable »).
- 3.3 `[x]` `steps_shared_core_widen_test.go` (DuckDB `:memory:`, AUCUNE base sous `data/`) : DDL
  legacy (PK + SMALLINT) + 3 lignes, **contrôle négatif** (avant migration, l'INSERT à 120 267 est
  REJETÉ — sans lui le test ne prouverait rien), migration, puis INTEGER × 2, 3 lignes intactes,
  valeurs relues à l'identique, INSERT à 120 267 accepté, second passage no-op. Un second test
  couvre la base sans `match_registry`.
- 3.4 `[x]` `internal/archlint/match_registry_ddl_types_test.go` : parse les DEUX DDL
  (`steps_shared_core.go` et `schema.go`) et compare le type de chaque colonne COMMUNE (les deux
  schémas n'ont jamais eu la même surface ; un plancher de 20 colonnes communes garde le parseur
  honnête). **Mutation vérifiée** : `team_0_score SMALLINT` remis dans `schema.go` → le test
  rougit en nommant la colonne et les deux fichiers, puis repasse vert.
- 3.5 `[x]` **Rien à allowlister.** Vérifié sur pièces : `no_art_patterns_test.go` surveille
  `ON CONFLICT DO UPDATE`, `INSERT OR REPLACE` et le bulk `UPDATE … FROM (VALUES …)` — aucun motif
  `ALTER` — et son en-tête dit explicitement que `match_registry` n'est PAS dans
  `tablesProtegees`. L'étape n'ajoute aucun motif surveillé ; la suite `internal/sync` (qui porte
  ce garde-rail) est verte.

**Gate G3** : `go test ./internal/games/halo_infinite/migrations/... ./internal/migration/... ./internal/archlint/... -timeout 30m`
→ **0**, 3 paquets `ok` ; `go test -tags=integration -p 1 ./internal/persist/... ./internal/migration/... -timeout 30m`
→ **0**, 2 paquets `ok` (persist 48 s). Garde-rail ART rejoué séparément : `go test ./internal/sync/ -run ART…`
→ **0**.

### Étape 4 — Clôture — 2026-09-16 14:20 — CLOSE

- 4.1 `[x]` Gates complets, codes de sortie relevés (pas la sortie filtrée) :
  `go build ./...` → **0** ; `go vet ./...` → **0** ;
  `go test ./... -timeout 30m` → **0**, **180 paquets `ok`**, zéro `--- FAIL`, zéro `FAIL` ;
  `go test -tags=integration -p 1 ./... -timeout 30m` → **0**, **181 paquets `ok`** (`-p 1` non
  négociable, respecté). Aucun paquet n'a eu à être rejoué seul.
- 4.2 `[x]` Entrée `.ai/thought_log.md` `[2026-09-16]`, statut Complété : les trois constats
  D-A/D-B/D-C, la décision D1, les écarts (étape post-sync carrière découplée ; `ALTER COLUMN`
  admis sur table indexée), les gates avec leurs chiffres, et ce qui n'a PAS été fait.
- 4.3 `[x]` `docs/COMMANDS.md` et `docs/FR/COMMANDS.md` relus côte à côte (même contenu, deux
  langues, même commit à l'étape 2) ; section « Callers » du README du pool relue — elle liste
  les six appelants, leur politique et l'unique exemption, et rappelle l'état d'avant.
- 4.4 `[~]` Revue adversariale (2 relecteurs : auth/pool, sync/migration) et CI — **au pilote**,
  hors périmètre de l'exécutant (ajustement du pilote, 2026-09-16).
- 4.5 `[~]` Aucun push, aucun merge, aucun changement de branche — **décision utilisateur**. Les
  5 commits restent sur `wt/sync-pool`.

**Note de finition** : le choix `rps = 0` (défaut du pool, 1 requête/s PAR TOKEN) pour les
backfills CSR passés au pool est documenté en commentaire dans `pool_engine.go` — l'ancien chemin
poussait UN jeton à 5 rps, le nouveau tient `1 × taille du parc`, ce qui est la posture du projet
(celle de `sync-delta --all`).

### Récapitulatif des commits (branche `wt/sync-pool`, base `feat/v75` @ e4a313311)

| Étape | Commit | Objet |
|---|---|---|
| — | `31483cb9a` | `docs(plan)` — le plan lui-même |
| 1 | `ebba6f5d4` | `feat(sync)` — seams title-owned câblés par toutes les CLI |
| 2 | `e0f42be18` | `feat(sync)` — le pool sert tout profil suivi |
| 3 | `2d51b80ec` | `fix(schema)` — `match_registry` team scores en INTEGER |
| 4 | (ce commit) | `chore(plan)` — journal, clôture, note de finition |

### Revue adversariale (4.4), ronde 1 — 2026-09-16 ~13:30-14:30 — pilote

Deux relecteurs aveugles (pool/CLI ; seams/migration), contrat écrit, filtre de recevabilité.
9 constats recevables, 0 jeté.

- **P0 (2), corrigés** : (a) `RunBackfillCSR` / `RunBackfillSharedCSR` exigeaient les tokens du
  joueur AVANT de regarder le client poolé → `backfill --csr` / `--shared-csr` en échec pour
  tous les joueurs ; garde extraite dans `requireTokensUnlessCustomClient` (3 tests, message
  sans « re-login »). (b) `ALTER COLUMN … SET DATA TYPE` échoue en DuckDB 1.5.5 dès qu'un index
  SECONDAIRE existe sur la table (`idx_mr_start_time`, mesuré par sonde par le relecteur) → la
  migration aurait échoué à chaque boot et bloqué pve/social ; le helper dépose les index
  secondaires (DDL relevée dans `duckdb_indexes()`), élargit, les recrée ; la fixture du test porte
  l'index réel et vérifie qu'il survit.
- **P1 (5), corrigés** : dry-run `--shared-csr` sans pool (aucune rotation de RT en dry-run) ;
  cinq docs qui décrivaient une dégradation carrière inexistante (l'étape carrière est hors du
  sync depuis le 2026-05-14) réécrites ; trois tests vides supprimés ; `steps_shared_core.go`
  ramené à 633 L (632 en base : la référence de l'étape), étape dans
  `steps_shared_widen_scores.go` ; runners CSR dans `cmd_backfill_csr.go` (865 L vs 1 026) ;
  doc de paquet `auto_sync.go` remise à l'endroit.
- **P2 (4), consignés** en §8, non corrigés.
- Gates rejoués : `go vet` (6 paquets) → 0 ; unitaires `cmd/levelup`, `migration`, `scheduler`,
  `archlint`, `halo_infinite/migrations`, `titleseams`, `sync` → 0 ; intégration `-p 1`
  `migration`, `halo_infinite/migrations`, `persist` → 0.
