# HANDOFF — Campagne APPEND-ONLY (éradication définitive du bug ART DuckDB)

> Document de reprise. Lire en entier avant de continuer. Dernière MAJ : 2026-06-20.
> Branche : `fix/metadata-art-battlepass-appendonly`. Voir aussi `.ai/thought_log.md`
> entrées `[2026-06-19]` et `[2026-06-20]`, et la mémoire projet
> `project_append_only_eradication_campaign`.

## 1. Le problème & la doctrine

**Bug** : `Failed to delete all rows from index. Only deleted 0 out of N rows` →
DuckDB met la base en FATAL `database has been invalidated` → **toute l'app tombe**
jusqu'au restart (et re-casse au boot suivant). Réapparu ≥4 fois.

**Déclencheur de classe** : une écriture qui RETIRE/RELOCALISE une entrée d'index ART :
- `UPDATE <t> SET col=…` où `col` est couverte par un index (PK ou secondaire), surtout NULL-bearing
- `DELETE` per-row sur une table à index ART
- `INSERT … ON CONFLICT DO UPDATE` / `INSERT OR REPLACE` (= delete+insert interne)
- `INSERT` pur = SÛR.

**Aggravé par** : handle DuckDB PARTAGÉ process-wide concurrent. `metadata.duckdb` et
`shared_social.duckdb` sont ouverts 1× en RW partagé → 1 FATAL = app entière down (RISQUE MAX).
`stats.duckdb` (player) = mono-writer sous lease (blast radius 1 joueur, mais `match_skill_rank`
a prouvé qu'il peut crasher aussi). `shared_matches_v2` = surtout RO + écritures persist INSERT-only.

**2 DÉCISIONS USER VERROUILLÉES** :
1. **Tout append-only AVANT de déployer.** Prod reste KO pendant le refactor (assumé). Pas de
   déploiement intermédiaire.
2. **PK-only suffit pour les tables de RÉFÉRENCE** (catalogue, battlepass, cache metadata) :
   elles restent en UPDATE-or-INSERT sur PK / SELECT-then-write, PAS d'append-only strict.
   L'append-only strict (table sœur `_history` + vue `_latest`) ne vise QUE les tables d'ÉTAT
   qui ont des DELETE / UPDATE-indexé / ON CONFLICT DO UPDATE.

**Interdit** : se reposer sur l'auto-réparation (`reopenMetadataIfInvalidated`, `ExecRecovered`,
`WithReopenOnInvalidated`) comme « solution » — ce sont des pansements à RETIRER en fin de campagne
une fois les écritures rendues structurellement sûres.

## 2. État actuel (FAIT)

Branche `fix/metadata-art-battlepass-appendonly`, **2 commits, tous tests verts, NON déployé** :

- **`0a47c384e`** — Éradication des index ART (RESTAURE LA PROD à elle seule) + 4 tables append-only :
  - Drop des index ART sur colonnes mutées (PK-only) : `game_variants_catalog(mode)`,
    `map_mode_pair x3`, `battlepass_*_definitions(is_current)`, `idx_pn_xuid_unread`
    (régression réarmée par le rebuild purge_data_health), `challenge(status/arc_id/campaign_id)`.
  - `asset_translations` : ON CONFLICT DO UPDATE → SELECT-then-write (`UpsertNoConflict`).
  - Garde-fou cross-DB colonne-aware : `internal/migration/metadata_art_surface_guard_test.go`
    (`TestNoARTSurfaceIndexInMigrations`).
  - 4 tables d'état → append-only : `match_favorites`, `media_likes`, `squad_member`,
    `notification_preferences`.
  - Garde-fou anti-DELETE : `internal/sync/append_only_state_guard_test.go`
    (`TestNoMutationOnAppendOnlyStateTables`, liste `appendOnlyStateTables`).
- **`9171f2875`** — `media_match_associations` + `media_files` reindex append-only (la table la
  plus complexe, ~30 edits).

**5 tables d'état converties** : match_favorites, media_likes, squad_member,
notification_preferences, media_match_associations.

## 3. LA RECETTE (gabarit réutilisable — 8 points de contact)

Pour convertir une table d'état `T` en append-only (calqué sur `match_skill_rank` /
`player_records_history`) :

1. **Migration** `steps_shared_social_<t>_append_only.go` (ou `steps_player_…`) :
   `Register(Migration{Name:"…_v1", TargetDB: TargetSharedSocial/TargetPlayer})`.
   Crée table sœur `T_history` : PK technique `id BIGINT DEFAULT nextval('seq')` + colonnes
   métier + flag d'état (`is_active`/`is_favorite`/`is_member`…) + `written_at` (+ `associated_at`
   immuable si récence métier). Index secondaires sur `(clé, written_at DESC)` — alimentés
   uniquement par INSERT donc sûrs. Gardé idempotent par `tableExists("T_history")` → no-op.
   **Backfill** depuis la table legacy `T` (gardé par `tableExists`/`columnExists`).
   Ne PAS dropper la table legacy (vestige vide, conservé pour ne pas casser les fixtures).
2. **Vue** `T_latest` : `QUALIFY ROW_NUMBER() OVER (PARTITION BY <clé> ORDER BY written_at DESC,
   id DESC) = 1` (+ filtre d'état, ex `WHERE is_active=TRUE`). Pour cardinalité 1→N voir §6 media.
3. **Writers** (Persister `internal/persist` + fallback repo `platform/duckdb`) → **INSERT pur**
   d'event. Plus aucun DELETE/UPDATE-indexé/ON CONFLICT. `ExecRecovered` → `Exec` (le pansement
   devient inutile).
4. **Readers** → la vue `T_latest` (+ filtre d'état). **PIÈGE #1 (récurrent)** : oublier un reader
   = feature invisible. Re-grep TOUS les `FROM T`/`JOIN T` (y compris cross-DB/cross-package).
5. **Fixtures de test** : ajouter `T_history` + seq + vue à chaque setup (`setupSocialDB`,
   `createSharedSocialSchemaForMediaTests`, fixtures duckdb/ops), et rebrancher les assertions
   qui lisaient `T` vers `T_latest`.
6. **order.go `canonicalOrder`** : insérer le nom de migration à sa position d'ENREGISTREMENT
   (= ordre alphabétique du fichier ; `order_test.go::TestSortByCanonicalIsNoOpOnCurrentRegistry`
   est un no-op STRICT). Méthode : créer un fichier temporaire `internal/migration/zz_dump_order_test.go`
   avec `TestZZDumpAllOrder` qui logge `All()` indexé, le lancer, lire la position, placer, **supprimer
   le fichier dump**. (Snippet dans le thought_log / réutilisable.)
7. **Whitelist** `internal/platform/duckdb/no_attach_on_social_test.go` (`sharedSocialFilesWhitelist`)
   : ajouter tout NOUVEAU fichier non-test mentionnant le littéral `shared_social` (migrations
   `steps_shared_social_*`, et tout fichier où un commentaire référence le nom de migration).
8. **Garde-fou** `internal/sync/append_only_state_guard_test.go` : ajouter `T` ET `T_history` à
   `appendOnlyStateTables` (échoue si DELETE/ON CONFLICT/INSERT OR REPLACE réapparaît sur `T` dans
   le hot path serveur — hors `_test`/`migration`/`ops`/`cmd`/`scripts`).

**Si un `CREATE TABLE IF NOT EXISTS T` runtime existe** (ex `ensureMediaTables` dans
`ops/media_store.go`, ré-exécuté à chaque IndexMedia) : il faut **aussi** y créer `T_history` + vue
(réconciliation), sinon sur DB fraîche la vue n'existe pas quand un reader l'interroge.

## 4. Build / test / commit — gotchas

- **Toolchain CGO obligatoire** : `export CGO_ENABLED=1 && export PATH="/c/msys64/ucrt64/bin:/c/msys64/mingw64/bin:$PATH"`.
  Builder/tester depuis `apps/go-api`.
- **Hook pre-commit (lefthook)** : lancer `git commit` AVEC l'env CGO ci-dessus, sinon `go-vet`
  exclut les packages cmd/ cgo (« build constraints exclude all Go files ») et le commit échoue.
  Faire `gofmt -w` sur les fichiers Go modifiés avant commit (le hook `gofmt` bloque sinon).
  **Jamais `--no-verify`.**
- `go test -race` incompatible avec le driver DuckDB (checkptr) → ne pas l'utiliser tel quel.
- Tests verts à viser par table : `go test ./internal/migration ./internal/persist ./internal/sync
  ./internal/platform/duckdb ./internal/service ./internal/notify ./internal/api ./internal/ops`.
- Demander l'autorisation user AVANT chaque `git commit` (règle projet).

## 5. RESTE À FAIRE (registre census 2026-06-20)

Périmètre réel restant : **~20 tables / ~57 sites mutants** hot-path. Ordre conseillé (blast-radius) :

### A. shared_social (handle concurrent — priorité)
- **`player_notifications`** : DELETE `DeleteNotification` (persister:383) + `CapAndSweepNotifications`
  + UPDATE `read_at` (Mark*Read/Unread/AllRead, persister + `notifications_repo.go` fallback).
  L'index `idx_pn_xuid_unread` est DÉJÀ droppé, mais c'est une table d'ÉTAT → append-only event log
  (created/read/deleted). Readers nombreux (`notifications_repo` list/count/unread + `notifications`
  package). MODÉRÉ-COMPLEXE — faire un design soigné (état read_at + tombstone delete + cap).
- **`squad_challenge_participant`** : UPDATE indexé `UpdateParticipantProgress`
  (`prestige_social_repo.go:388`) + `AddParticipant` ON CONFLICT DO NOTHING. **Design existant**
  dans l'output du workflow `wmlnfefr9` (CTAS-swap id+written_at, vue _latest). **Review a flagué** :
  re-join réinitialise la progression → faire SELECT-then-INSERT **carry-forward** (reporter
  current_value/completed_at). Readers : `ListParticipants`, `CountActiveParticipants` → _latest.

### B. metadata ref/cache (PLUS SIMPLE — SELECT-then-write, PAS append-only ; décision user #2)
Convertir `INSERT … ON CONFLICT DO UPDATE` → `(*duckdb.DB).UpsertNoConflict` (helper déjà utilisé
pour `asset_translations`, cf `metadata_repo_assets.go`). VÉRIFIER d'abord que l'UPDATE ne touche pas
une colonne indexée (sinon dropper l'index aussi, cf garde-fou metadata).
- `map_images_registry` (×2), `medal_image_cache`, `waypoint_medals_raw`, `milestone_catalog`,
  `xuid_aliases`.
- `career_rank_translations` : INSERT OR REPLACE → SELECT-then-write.
- `xbox_achievement_definitions` : DELETE → vérifier le contexte (seed/refresh ? sérialisé ?).

### C. prestige/player (mix shared_social + player DB)
- ON CONFLICT DO UPDATE : `user_prestige`(×2), `preset_arc`/`preset_arc_step`, `baseline_state`,
  `player_records` (NB : `player_records_history` existe déjà — vérifier si ce site est legacy/fallback),
  `player_privacy_state`, `player_match_enrichment`(×2), `lusr_component_history`, `sync_meta`(×2).
  → append-only (état) ou SELECT-then-write (cache) selon la table.
- DELETE player DB (mono-writer lease) : `challenge` (DeleteByArc + autres), `arc`,
  `match_citations`(×2), `personal_score_awards`.

### D. shared match DELETE (rebuild/reconcile sync) — AUDITER d'abord
`match_participants`, `killer_victim_pairs`, `weapon_kills`, `player_skill_state_v2`. Probablement
des DELETE-then-reinsert sérialisés par lease (mono-writer) dans des chemins rebuild/reconcile.
Décider : append-only via `persist` (INSERT-only) OU justifier comme sérialisé sûr (comme la
compaction `match_skill_rank`). Cf `internal/sync/no_art_patterns_test.go` `criticalMatchTables`.

### DÉJÀ SÛRS — NE PAS TOUCHER
- `catalog_fetch_queue` : table SANS index (PK+idx droppés par `rebuild_catalog_fetch_queue_drop_art_indexes`)
  → DELETE/UPDATE safe.
- `match_skill_rank` : DELETE de compaction documenté, mono-writer player DB, PK BIGINT (cf
  `compactMatchSkillRankSuperseded`).

## 6. PIÈGES APPRIS (ne pas re-tomber dedans)

- **Cardinalité 1→N (media)** : un média s'associe à N matchs (auto). Une vue `PARTITION BY <clé seule>`
  collapse à 1 → FAUX. Vue media finale = `ROW_NUMBER PARTITION BY (media_file_id, match_id)` +
  `bool_or(is_manual)` pour que le manuel masque l'auto et l'auto préserve le 1→N. **Toujours vérifier
  la cardinalité réelle d'une table avant de choisir la PARTITION de la vue.**
- **Les workflows de design se trompent** : sur media, 2 designs successifs (génération, puis partition
  naïve) étaient bancals ; la revue adversariale en a écarté un, et j'ai corrigé la cardinalité du second
  MOI-MÊME (la revue l'avait ratée). → Lire le code réel, ne pas appliquer un design de workflow à l'aveugle.
  Garder le pattern : design → revue adversariale → vérif manuelle avant impl.
- **Ordre des migrations** (`rekey_squad_member_xuid` DROP+recrée `squad_member`) : un backfill peut
  tourner AVANT le rekey → garder le backfill par `columnExists`.
- **resetPlayerMediaIndex** ciblait la player DB où les tables média sont DROPPÉES
  (`drop_media_from_player_db`) → chemin orphelin/mort, DELETE retirés (no-op).

## 7. CLÔTURE (quand toutes les tables sont faites)

1. **Recensement final** : re-lancer le census (`grep -rniE "DELETE FROM|ON CONFLICT.*DO UPDATE|INSERT OR REPLACE"`
   sur `internal` hors test/migration) → confirmer ZÉRO résiduel sur tables d'état.
2. **Retirer les pansements** (code mort une fois les écritures sûres) : `reopenMetadataIfInvalidated`
   (`internal/api/registry_catalog_drain.go`) + les `ExecRecovered`/`WithReopenOnInvalidated` devenus
   inutiles. (Règle projet : pas de code mort.)
3. **Go/no-go** : `go test ./...` COMPLET + `go vet ./...` (env CGO) ; front `npm run typecheck/lint`
   si touché ; thought_log à jour (skill `delivery-checklist`).
4. **Déploiement** : merge sur `main` = **auto-deploy prod** (`git reset --hard origin/main` sur le VPS
   → deploy.sh). Synchroniser le main LOCAL après push (`git branch -f main origin/main`). Prévenir le
   user (downtime/prod). Prod est KO depuis le début de la campagne → ce merge la restaure.

## 8. Pointeurs

- Outputs des workflows de design (réutilisables) : sous
  `…/756bbf95-…/tasks/{wmlnfefr9,wweca9iz8,w9e1zteno}.output` (squad_challenge design dans wmlnfefr9 ;
  media design final + verify writers/readers dans w9e1zteno).
- Garde-fous : `internal/migration/metadata_art_surface_guard_test.go`,
  `internal/sync/append_only_state_guard_test.go`, `internal/sync/no_art_patterns_test.go`,
  `internal/platform/duckdb/no_attach_on_social_test.go`.
- Précédents append-only à imiter : `steps_player_append_only_match_skill_rank.go`,
  `create_player_records_history_append_only`, `create_streak_history_append_only`, et mes 5 migrations
  `steps_shared_social_*_append_only.go`.
