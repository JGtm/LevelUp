# Volet 2 — Pourquoi les index de `personal_score_awards` se désynchronisent

Worktree `LevelUp-wt-psa-cause`, branche `wt/psa-index-cause` (base `feat/v75` @ `15b582f13`).
Date : 2026-08-28. Aucun commit, aucune base réelle ouverte, aucune écriture hors du worktree.

---

## 1. Verdict (5 lignes)

1. **NON REPRODUIT** — 12 scénarios sur DuckDB fichier (jamais `:memory:`), tous verts, y compris les deux mécanismes nommés par les mainteneurs DuckDB.
2. **La cause reste inconnue localement, mais elle est identifiée EN AMONT** : le symptôme exact (comptage filtré < `GROUP BY`) est documenté dans **duckdb/duckdb#23645**, **toujours ouverte et reproduite en 1.5.5**, la version que nous embarquons.
3. **`CLAUDE.md` se trompe d'issue** : le bug « Failed to delete all rows from index » est #23645, pas #23046 (qui est une corruption de tas en 1.5.0). Découverte non traitée (§8).
4. **Garde posée** : `internal/scheduler/data_health_psa_index.go` — détection périodique dans le cycle data-health, **alerte seule, aucune réparation automatique**.
5. **Coût mesuré** : **41 ms par player DB** (échantillon 200 clés, 16 500 lignes) → ~0,16 s pour 4 joueurs, dans un cycle qui tourne toutes les 24 h.

---

## 2. Le pattern d'écriture réel de `personal_score_awards`, sur pièces

### 2.1 Schéma et index — deux autorités, une seule DDL

| Autorité | Fichier | Ce qu'elle pose |
|---|---|---|
| DDL canonique (DB fraîche) | `internal/migration/steps_player_schema_authority.go:52-78` (`PlayerPersonalScoreAwardsDDL`) | séquences `personal_score_awards_id_seq` + `psa_generation_seq`, table, **3 index** |
| Conversion append-only (DB legacy) | `internal/migration/steps_player_append_only_personal_score_awards.go:82-92` (`PostSwap`) | les **mêmes 3 index** après le swap |
| Soin de transition | `internal/sync/schema.go:354` → `migration.EnsurePersonalScoreAwardsAppendOnly`, rejoué à **chaque `OpenPlayerDB`** (`internal/sync/schema.go:451`) | idempotent : `CREATE INDEX IF NOT EXISTS` (no-op) + `CREATE OR REPLACE VIEW` |

Index réellement présents (confirmé par `duckdb_indexes()` sur fixture) :

- `idx_psa_match(match_id)`
- `idx_psa_category(award_category)`
- `idx_psa_gen(match_id, xuid, generation_id)`
- \+ l'ART **implicite de la PRIMARY KEY `id`** (invisible de `duckdb_indexes()`, posé par `ALTER TABLE … ADD PRIMARY KEY (id)` dans le swap, ou par la contrainte inline du DDL fraîche).

Deux index ont été **retirés** le 2026-08-05 par des steps dédiés (`drop_psa_xuid_art_index_v1`, `drop_psa_match_xuid_art_index_v1`, `steps_player_schema_authority.go:135-158`) : `idx_psa_xuid` et `idx_psa_match_xuid`.

### 2.2 Écritures per-match — INSERT pur, deux chemins jumeaux

| Chemin | Fichier | Forme SQL |
|---|---|---|
| Sync live / convergence / backfill | `internal/sync/writes.go:222-256` (`InsertPersonalScoreAwards`) | `BeginTx` → `SELECT nextval('psa_generation_seq')` → N × `INSERT` (ou **1 `INSERT` tombstone** `is_tombstone=TRUE` si extraction vide) → `Commit` |
| Pipeline Collect→Persist (ADR 0019/0030) | `internal/persist/player_persister.go:324-351` (`persistPersonalScoreAwards`) | idem, dans la TX du batch (`internal/persist/player_persister.go:94`) |

Appelants : `internal/sync/backfill_personal_scores.go:69,77` et `internal/sync/convergence.go:212`.

**Aucun `UPDATE`, aucun `DELETE`, aucun `ON CONFLICT` sur cette table dans `internal/`** (grep exhaustif : seuls les commentaires historiques et les tests d'autorité de schéma sortent). La sémantique « remplacer » passe par `generation_id` + la vue `personal_score_awards_latest` (`DENSE_RANK`, génération MAX, tombstones exclus, `steps_player_append_only_personal_score_awards.go:53-61`).

**Aucun `CHECKPOINT`** n'est émis sur le chemin d'écriture PSA. Le seul `CHECKPOINT` de la chaîne migration est le post-cycle best-effort de `internal/migration/registry.go:365-385`.

### 2.3 Opérations de MIGRATION — les suspects sérieux

`applyCreatePersonalScoreAwards` (`steps_player_schema_authority.go:171-179`) enchaîne **trois passes** : conversion append-only → DDL → conversion append-only (pour poser la vue sur une table qui vient de naître).

La conversion elle-même, `rebuildAppendOnlyTx` (`internal/migration/append_only_rebuild.go:170-236`), fait **tout dans UNE transaction** :

```
BEGIN
  CREATE SEQUENCE IF NOT EXISTS personal_score_awards_id_seq / psa_generation_seq
  DROP TABLE IF EXISTS personal_score_awards__appendonly
  CREATE TABLE personal_score_awards__appendonly AS SELECT … FROM personal_score_awards   -- CTAS
  SELECT COUNT(*) …                                                                        -- garde rebuilt==before
  DROP TABLE personal_score_awards
  ALTER TABLE …__appendonly RENAME TO personal_score_awards
  ALTER TABLE … ADD PRIMARY KEY (id)          -- ART UNIQUE construit sur table DEJA PEUPLEE
  ALTER TABLE … ALTER COLUMN id SET DEFAULT nextval(…)
  ALTER COLUMN written_at SET DEFAULT … ; ALTER COLUMN is_tombstone SET DEFAULT …
  CREATE INDEX idx_psa_match / idx_psa_category / idx_psa_gen   -- 3 ART sur table DEJA PEUPLEE
COMMIT
```

Pas de `CHECKPOINT` après ce commit (seulement le post-cycle générique). C'est exactement le motif « `CREATE INDEX` sur table peuplée, dans une TX, sans checkpoint » que le mandat désignait comme suspect au moins aussi sérieux que les écritures — il a été rejoué tel quel (scénario S3) et **n'a rien produit**.

### 2.4 `DELETE` sur la table — hors `internal/`, dans deux CLI

- `cmd/cleanup_orphan_match/main.go:53,145` : `DELETE FROM personal_score_awards WHERE match_id = ?` dans une TX, **`ROLLBACK` en dry-run**, `COMMIT` avec `--apply`.
- `cmd/cleanup_post_art/main.go:55` : même liste de tables, purge par date de coupure.

C'était l'hypothèse la plus prometteuse (un `DELETE` annulé qui retire les entrées ART sans restaurer les lignes expliquerait un déficit partiel). **Réfutée** par S4 et S5.

---

## 3. Protocole de reproduction

Harnais : `apps/go-api/internal/migration/psa_index_repro*_test.go`, **build tag `psarepro`** — exclu de `go build`, `go vet`, `go test` et de la CI. Lancement :

```
cd apps/go-api && go test -tags=psarepro ./internal/migration/ -run TestPSARepro -v -timeout 60m
```

Chaque phase applique **le contrôle de `cmd/repair_psa_index/diag.go` à l'identique** : référence par `GROUP BY match_id || ''` (expression → scan forcé) vs `WHERE match_id = ?` (colonne nue → index), sur les 3 axes indexés.

### 3.1 Découverte de méthode — `EXPLAIN` MENT, il faut `EXPLAIN ANALYZE`

Sur DuckDB 1.5.5, `EXPLAIN SELECT COUNT(*) … WHERE match_id = ?` affiche **toujours** :

```
│          SEQ_SCAN         │
│   Type: Sequential Scan   │
```

…y compris pour un lookup sur la PRIMARY KEY, avant/après CHECKPOINT, après réouverture, à 4 400 comme à 37 400 lignes, et même avec `index_scan_percentage=1.0` / `index_scan_max_count=1000000`. La stratégie est choisie **à l'exécution** (cf. le réglage `debug_physical_table_scan_execution_strategy`). Seul `EXPLAIN ANALYZE` révèle la vérité :

```
strategie DEFAULT      : acceptee, COUNT=4 err=<nil>
    -> EXPLAIN ANALYZE type = INDEX SCAN
```

**Conséquence** : un contrôle validé sur `EXPLAIN` seul conclurait à tort « l'index n'est jamais sollicité, la mesure ne vaut rien ». Le harnais assère donc `Index Scan` via `EXPLAIN ANALYZE` (`reproAssertIndexScan`) avant chaque scénario — **le contrôle mesure bien l'ART**, et le diagnostic de `repair_psa_index` est méthodologiquement valide.

### 3.2 Tableau des scénarios (une variable à la fois)

Volume de base : **1 100 matchs × 4 awards = 4 400 lignes** (ordre de grandeur des player DB réelles), sauf mention.

| # | Ce qui est rejoué | Variable isolée | Résultat |
|---|---|---|---|
| S0 | 1 100 matchs × 4, index posés AVANT peuplement, + CHECKPOINT | socle | **0 écart** (4 400 l.) |
| S0b | plan à chaque étape (insert / CHECKPOINT / réouverture / volume ×8) | plan d'exécution | `EXPLAIN` = SEQ_SCAN partout (voir §3.1) |
| S0c | 8 formes de requête (COUNT/SELECT/PK/littéral/param/triplet) | forme de requête | `EXPLAIN` jamais INDEX_SCAN ; `EXPLAIN ANALYZE` = **Index Scan** |
| S0d | `index_scan_percentage`, `index_scan_max_count` | réglages planner | sans effet sur `EXPLAIN` |
| S0e | `debug_physical_table_scan_execution_strategy` | forçage stratégie | confirme **Index Scan** au runtime |
| S1 | 3 vagues : sync initial, régénérations (330 matchs), tombstones (100) + regen, CHECKPOINT + **réouverture entre chaque vague** | générations / tombstones / réouvertures | **0 écart** (6 420 l.) |
| S1b | mêmes vagues **SANS aucun CHECKPOINT**, réouvertures (WAL rejoué) | absence de CHECKPOINT | **0 écart** (5 820 l.) |
| S2 | `CREATE INDEX` **APRÈS** peuplement, puis nouvelle vague | ordre CREATE INDEX / peuplement | **0 écart** (5 200 l.) |
| S3 | table **legacy** peuplée par l'ANCIEN `DELETE+INSERT` (1 100 + 400 ré-extractions), puis `applyCreatePersonalScoreAwards` (swap CTAS + PK + 3 CREATE INDEX en TX), CHECKPOINT, réouverture, vagues append-only | rebuild append-only sur table peuplée | **0 écart** à chacune des 6 phases |
| S4 | `DELETE … WHERE match_id=?` puis **ROLLBACK** (1 match, puis 60 matchs = 5,5 % du corpus), CHECKPOINT, puis re-INSERT | dry-run des CLI cleanup | **0 écart** (4 640 l.) |
| S5 | `DELETE` de 60 matchs **COMMITTÉ**, CHECKPOINT (vacuum), puis re-INSERT | DELETE réel + vacuum | **0 écart** |
| S6 | 3 cycles : process enfant écrit 150 matchs puis `os.Exit(0)` **sans Close ni CHECKPOINT** ; parent rouvre et contrôle | arrêt brutal entre transactions | **0 écart** (jusqu'à 6 200 l.) |
| S7 | **360 000 lignes** (≈ 3 row groups), CHECKPOINT, puis `DELETE` de **85 %** + CHECKPOINT, réouverture | vacuum / compaction de row groups | **0 écart** (400 clés échantillonnées) |
| S8 | `vacuum_rebuild_indexes = true` | réglage expérimental | **NON TESTABLE** : refusé en DSN (« invalid or local option for global database config ») et en `SET` (« Cannot change … while database is running »). Hors cause de toute façon (défaut 0, cf. §4) |
| S9 | 8 goroutines écrivant en parallèle, `MaxOpenConns=8` | concurrence intra-process | 0 erreur d'écriture, **0 écart** |
| S10 | 6 cycles : enfant écrit + `CHECKPOINT` en boucle, parent le **tue** (`Process.Kill`) après 150 ms … 3,5 s | mort brutale **pendant** un CHECKPOINT | **0 écart** (jusqu'à 15 356 l.) |
| S11 | **UNE transaction de 200 200 lignes** (> 1 row group → écriture optimiste), puis mort brutale sans CHECKPOINT, réouverture (rejeu WAL `ReplayRowGroupData`) — mécanisme de la PR upstream **#24744** | rejeu WAL d'un append optimiste | **0 écart** (jusqu'à 605 000 l.) |
| S12 | 4 goroutines `INSERT` massif + **ROLLBACK** en boucle, 2 goroutines `CHECKPOINT` en boucle, 1 goroutine committant — 12 s — mécanisme de la PR upstream **#24755** | rollback d'append partiel **pendant** un checkpoint concurrent | **0 écart** (19 800 l.) |

Toutes les bases sont **sur disque** (`t.TempDir()` ou `os.MkdirTemp`), nettoyées par le harnais. Aucun `.duckdb` ne subsiste dans le worktree.

---

## 4. Version DuckDB et état de l'art

**Version embarquée** : `github.com/duckdb/duckdb-go/v2 v2.10505.0` + `duckdb-go-bindings v0.10505.0` → `SELECT version()` = **`v1.5.5`** (mesuré sur fixture, S0).

### 4.1 L'issue citée par `CLAUDE.md` n'est pas la bonne

- **duckdb/duckdb#23046** — « DuckDB 1.5.0's ART index constraint enforcement corrupts the heap on file-backed ». SIGSEGV sur `executemany` d'INSERT, macOS ARM64 / Python. **Ne mentionne jamais** « Failed to delete all rows from index ». Ouverte, sans correctif.
- **duckdb/duckdb#23645** — « Explicit non-unique ART indexes reach a persisted state where every `INSERT … ON CONFLICT DO UPDATE` fails with "Failed to delete all rows from index" (recurring; 1.5.2 and 1.5.4) ». **C'est notre bug.** Ouverte depuis le 2026-07-06, label `under review`, dernière activité 2026-08-20.

### 4.2 Ce que #23645 établit — et qui recoupe exactement notre observation

Un commentaire (2026-07-18, Windows 11 / DB fichier / process unique / accès sérialisés) décrit **notre symptôme au mot près** :

> « **Detectable before the fatal:** in the corrupt-but-not-yet-fatal state, indexed/filtered queries return short counts while full scans are correct — e.g. `SELECT COUNT(*) WHERE stage = 1` returned 51 vs 84 from `GROUP BY stage`. A filtered-vs-full-scan parity check makes a reliable canary. »
>
> « Same as your finding: data fully intact, `DROP INDEX` + `CREATE INDEX` on the explicit indexes clears it. »

Autres faits verbatim de l'issue :
- « **Data is intact.** Full scans, aggregates, and `EXPORT DATABASE` all succeed. »
- « **Not healed by**: `CHECKPOINT` on a clean open, a plain open/close cycle, or an open that merges a small WAL. The state survives checkpoints and process restarts until an index rebuild. »
- Le message d'erreur lui-même est **nouveau en 1.5.0** (PR #20430, `BoundIndex::Delete` devient bloquant) : « the same inconsistency would presumably have been silently ignored on 1.4.x ». **Donc l'incohérence pouvait déjà être fabriquée en 1.4.x, silencieusement.**

### 4.3 État en 1.5.5 précisément : **NON CORRIGÉ**

- Témoignage direct dans l'issue : matrice `1.4.5 → OK`, `1.5.4 → échoue`, `1.5.5 → échoue`.
- Les notes de release v1.5.5 ne contiennent **aucune** entrée ART / index / vacuum / checkpoint.
- Le seul correctif mergé (**PR #24744**, « Allow duplicates in index in ReplayRowGroupData » — « the inserts that should have been in the index weren't there, hence the index was corrupted ») l'a été le 2026-08-14, soit **62 commits APRÈS le tag v1.5.5**. Aucune v1.5.6 n'existe.
- **PR #24755** (« Fix checkpoint delta revert » — rollback d'un append partiel pendant un checkpoint concurrent, entrées périmées fusionnées dans l'index principal) est **toujours ouverte** ; le mainteneur écrit « just a hunch though, no certainty without a reproducer ».

### 4.4 Le vacuum est ÉCARTÉ, avec sa réfutation

Le réglage `vacuum_rebuild_indexes` (introduit en **1.5.2**, PR #21769, défaut **0**) est décrit ainsi au tag v1.5.5 :

> « (Experimental) Allow vacuum to compact row groups on tables with bound ART indexes, rebuilding the indexes afterward. »

Et le corps de la PR est explicite sur l'état **antérieur** :

> « This was disallowed previously (and this PR only allows it by enabling the setting) since **vacuuming will change rowid's and thus make the ART stale**. »

Donc, réglage à 0 (notre cas — il n'est même pas modifiable par le driver Go, cf. S8) : **le vacuum ne compacte pas les tables indexées**, il les saute. L'hypothèse « le CHECKPOINT a vacuumé et déplacé les row_id sous l'ART » est **réfutée par construction**, en plus de l'être empiriquement par S5 et S7.

---

## 5. Non-reproduction : ce que ça exclut, ce que ça n'exclut pas

### Écarté, avec réfutation

| Hypothèse | Réfutation |
|---|---|
| Le `DELETE` + `ROLLBACK` des CLI cleanup en dry-run | S4 : 0 écart, y compris sur 5,5 % du corpus, avant et après CHECKPOINT, et après re-INSERT |
| Le `DELETE` committé + vacuum au CHECKPOINT | S5 (60 matchs) et S7 (85 % du corpus, 3 row groups) : 0 écart. Plus : le vacuum **refuse** de compacter une table indexée quand `vacuum_rebuild_indexes=0` (PR #21769, verbatim §4.4) |
| Le swap CTAS + `ADD PRIMARY KEY` + `CREATE INDEX` en TX de la migration append-only | S3 : 0 écart aux 6 phases, y compris sur une table legacy salie par 1 500 cycles `DELETE+INSERT` |
| L'ordre `CREATE INDEX` avant vs après peuplement | S0 vs S2 : identiques, 0 écart |
| L'absence de `CHECKPOINT` sur le chemin d'écriture | S1b : 0 écart après deux réouvertures avec WAL rejoué |
| L'arrêt brutal du serveur (le poste a tué `server.exe`) | S6 (entre transactions) et S10 (pendant un CHECKPOINT, 6 timings) : 0 écart |
| Le volume (~1 100 matchs) | S7/S11 jusqu'à 605 000 lignes : 0 écart |
| La concurrence multi-connexions | S9 : 0 écart. Et le chemin player DB de prod est **sérialisé** de toute façon (`OpenReadWrite` → `poolSingleConn`, `internal/platform/duckdb/db.go:286-299`) |
| « Le contrôle de `repair_psa_index` ne mesure pas l'index » | `EXPLAIN ANALYZE` confirme `Index Scan` (§3.1) — le diagnostic du 2026-08-27 est valide |

### **Non** exclu

1. **Un vecteur qui n'existe plus dans le code lu.** Les 4 DB réelles ont plusieurs mois et sont passées par plusieurs versions de DuckDB (dont 1.4.x, où l'incohérence était **silencieuse**, cf. §4.2) et par le pattern `DELETE+INSERT` d'avant la Phase 2 (2026-06-21). L'état corrompu **survit aux CHECKPOINT et aux redémarrages** jusqu'à un rebuild d'index : une corruption ancienne aurait persisté jusqu'au 2026-08-27. **C'est l'hypothèse la plus PLAUSIBLE** — mais elle est plausible, pas mesurée, et le rebuild CTAS de la migration append-only aurait dû la balayer (à moins que la corruption soit postérieure à la migration).
2. **Le mécanisme de #24755** (rollback d'append partiel pendant un checkpoint concurrent). S12 le rejoue mais en **une seule** configuration de timing ; un bug de fenêtre de course peut demander un ordonnancement précis que 12 s de martelage ne rencontrent pas.
3. **Une écriture venue d'un autre process** (CLI `cmd/*` lancée pendant que le serveur tournait, ou l'inverse). Non testable sans les bases réelles.
4. **Une pathologie propre aux 4 fichiers réels** (fragmentation, historique de schéma, résidus de `cmd/force_rebuild_art`) — non observable sans les ouvrir.

**Ce que je n'ai PAS fait** : ouvrir une base réelle (interdit, serveur actif), tester une version DuckDB antérieure (aurait demandé un downgrade du module, hors périmètre).

---

## 6. La garde posée

### 6.1 Ce qui est livré

| Élément | Emplacement |
|---|---|
| Détecteur + intégration au cycle | `apps/go-api/internal/scheduler/data_health_psa_index.go` (267 l.) |
| Règle de détection isolée | `comparePSACounts(refs, lookup)` — testable sans SQL |
| Sonde SQL bornée | `scanPSAIndexDesync(ctx, db, sampleKeys)` + `psaSampleKeyCounts` |
| Boucle par titre / par joueur | `(*HealthScheduler).auditTitlePSAIndex` / `auditPlayerPSAIndex` |
| Jauge expvar | `publishPSAIndexGaugeIfComplete` → `data_health_psa_index_desync_keys` |
| Câblage | `data_health_check.go` : nouveaux champs `PSAIndex*` sur `DataHealthCheckResult`, appel en étape 6 de `auditTitle`, entrée dans `WarningsTotal`, publication de jauge, logs |
| Tests | `apps/go-api/internal/scheduler/data_health_psa_index_test.go` (6 tests, 5 sous-cas) |
| Mesure de coût | `data_health_psa_index_cost_test.go` (build tag `psarepro`) |

### 6.2 Décisions et respect des contraintes

- **Détecter, pas réparer.** `slog.ErrorContext` nommant le joueur, le nombre de clés en écart, les lignes manquantes et la commande de remédiation manuelle (`go run ./cmd/repair_psa_index -repair`, serveur arrêté). Aucune DDL n'est émise. Justification écrite en tête de fichier : réparer en silence masquerait la récurrence et empêcherait de mesurer la fréquence — et la structure voisine (auto-heal LUSR) ne prescrit pas l'inverse, elle **rejoue un calcul déterministe**, elle ne touche pas au stockage. Ce kill-switch-là est d'ailleurs **OFF par défaut**.
- **Coût borné et mesuré.** Échantillon de `psaIndexSampleKeys = 200` clés `match_id` distinctes par player DB et par cycle, tirées par `USING SAMPLE … ROWS (reservoir)` **sans graine** : le tirage change à chaque cycle, la couverture s'accumule. Timeout de 60 s par base.

  ```
  COUT sample=50    rows=16500 cles=50   ecarts=0  moyenne=12ms
  COUT sample=200   rows=16500 cles=200  ecarts=0  moyenne=41ms
  COUT sample=1000  rows=16500 cles=1000 ecarts=0  moyenne=201ms
  ```

  → **41 ms × 4 joueurs ≈ 0,16 s** par cycle de 24 h.
- **Lecture mono-process.** `duckdb.OpenReadForQuery(playerPath)` — jamais `OpenReadOnly` forcé : si un sync tient déjà le fichier en RW dans le process, le handle en cache est réutilisé au lieu d'échouer sur « different configuration » (ADR 0013/0016). Une base inouvrable ⇒ `WARN` + `PSAIndexPlayersUnmeasured++`, jamais d'échec du cron.
- **`unmeasured ≠ sain`.** La jauge expvar n'est republiée que si `PSAIndexPlayersUnmeasured == 0` (même invariant que la jauge LUSR) ; le cycle est loggué en `WARN` sinon.
- **Aucune erreur avalée** : chaque échec est loggué (`slog.WarnContext` / `ErrorContext`, `"err", err`) et compté (`ProbeErrors` pour une sonde SQL ratée, `PSAIndexPlayersUnmeasured` pour un scan partiel).
- **Title-agnostic** : boucle sur `titlePkg.DefaultRegistry().All()`, zéro `slug == "…"`. **Aucune clé de `capabilities.toml` ne décrit `personal_score_awards`** (la plus proche, `analytics.career_xp_estimate`, a une autre sémantique) : le prédicat retenu est structurel — **présence de la table** dans la player DB, sinon skip silencieux (`errPSATableAbsent`, `slog.Debug`). C'est exactement la façon dont les autres sondes de ce package se dégradent.
- **Seuils** : 267 l. / 250 l. pour les deux nouveaux fichiers ; fonction la plus longue = 42 l. ; `data_health_check.go` passe de 415 à 441 l.
- **Pas de fichier à la racine de `internal/sync/`** : la garde est dans `internal/scheduler/`. `TestSyncRootPackageFrozen` reste vert.
- **Pas de changement de contrat HTTP** : le signal remonte par `WarningsTotal` (champ existant du DTO `MonitoringDataHealth`) + la jauge expvar. Aucune régénération d'`openapi.yaml` ni de types web n'est nécessaire.

### 6.3 Un détail non cosmétique

Le ratchet `internal/archlint/no_local_longest_run_test.go` a **rouge** sur la première version du détecteur : sa regex `\+\+\n\s*if \w+ > \w+ \{` matchait la séquence `rep.KeysDiverging++` / `if scanned > indexed {`. Faux positif (aucune série n'est mesurée). **Corrigé en réordonnant les deux instructions**, pas en ajoutant une exemption — commentaire explicatif en place.

### 6.4 Tests de la garde

```
=== RUN   TestComparePSACounts                              (5 sous-cas : sain / déficit 2-sur-4 / déficit multiple / excédent / vide)
=== RUN   TestComparePSACountsPropageLErreurDeLookup
=== RUN   TestScanPSAIndexDesyncTableAbsente
=== RUN   TestScanPSAIndexDesyncTableVide
=== RUN   TestScanPSAIndexDesyncBaseSaine                   (DuckDB fichier, 120 matchs × 1-3 générations, catégories NULL, tombstones, CHECKPOINT — aucun faux positif)
=== RUN   TestScanPSAIndexDesyncBorneLEchantillon           (300 clés, sample=25 → KeysSampled==25)
=== RUN   TestPublishPSAIndexGaugeIfComplete                (jauge gelée si contrôle partiel)
PASS  ok  levelup/go-api/internal/scheduler  2.192s
```

**Limite assumée et documentée dans le fichier de test** : on ne sait pas corrompre un ART DuckDB à la demande (c'est tout l'objet de cette enquête). Le cas positif « index désynchronisé détecté » est donc testé au niveau de la **règle** (`comparePSACounts` avec un lookup injecté qui rend 2 sur 4 — le chiffre réellement observé le 2026-08-27), et le niveau SQL est testé en non-régression (aucun faux positif sur base saine, échantillon borné, table absente/vide).

---

## 7. Gates — sorties verbatim

Toutes lancées **une seule commande `go` à la fois** (contrainte de corruption du cache de build sur ce poste). Depuis `apps/go-api/` sauf mention.

### `go build ./...`
```
--- go build ./...
EXIT_BUILD=0
```

### `go vet ./...`
```
--- go vet ./...
EXIT_VET=0
```

### `go test ./...` (paquet `internal/himap` exclu)
```
EXIT_TEST=0
```
Commande : `go test $(go list ./... | grep -v "/internal/himap")` — sortie filtrée des lignes `ok`/`?`, donc **vide = zéro échec**.

**`internal/himap` : NON LANCÉ**, exclusion explicite. Son test de balayage de corpus dépasse le timeout Go de 10 min sur ce poste ; rouge pré-existant, local-only, hors périmètre CI, hors périmètre de ce mandat.

Rappel de l'état intermédiaire, pour la traçabilité : le premier passage de ce gate était **ROUGE** sur le ratchet archlint (§6.3), corrigé, puis re-lancé vert.

### `go test -tags=integration -p 1`
```
ok  	levelup/go-api/internal/scheduler	14.322s
ok  	levelup/go-api/internal/migration	4.654s
ok  	levelup/go-api/internal/persist	20.373s
ok  	levelup/go-api/internal/sync	108.323s
ok  	levelup/go-api/internal/sync/haloclient	5.221s
ok  	levelup/go-api/internal/sync/halotest	0.510s
ok  	levelup/go-api/internal/sync/invariants	0.329s
ok  	levelup/go-api/internal/sync/killcollector	3.167s
?   	levelup/go-api/internal/sync/matchflags	[no test files]
ok  	levelup/go-api/internal/sync/objective	0.056s
ok  	levelup/go-api/internal/sync/replayartifacts	0.096s
?   	levelup/go-api/internal/sync/schemadrift	[no test files]
ok  	levelup/go-api/internal/sync/skill	1.164s
ok  	levelup/go-api/internal/sync/snapshot	0.504s
?   	levelup/go-api/internal/sync/testutil	[no test files]
ok  	levelup/go-api/internal/sync/v2	13.364s
EXIT_INTEG=0
```
Portée : `./internal/scheduler/... ./internal/migration/... ./internal/persist/... ./internal/sync/...`. **Aucun fichier de `persist/` ni `sync/` n'a été modifié** par ce mandat ; le gate a été passé quand même sur ces paquets (ils sont les voisins directs du sujet). La suite `-tags=integration ./...` complète n'a pas été lancée, faute de modification hors `scheduler`.

### `make go-api-lint` (depuis la racine)
```
golangci-lint présent — lint complet (ratchet CI, dette gelée exclue).
level=warning msg="[runner/nolint_filter] Found unknown linters in //nolint directives: gosec — limit/placeholders maîtrisés, plr0913 — coordinator function"
0 issues.
EXIT_LINT=0
```
(Le `warning` sur les `//nolint` est pré-existant et sans rapport avec ce diff.)

### Harnais d'enquête (hors gates)
```
go test -tags=psarepro ./internal/migration/ -run TestPSARepro -timeout 60m
ok  	levelup/go-api/internal/migration	121.603s
```

---

## 8. Découvertes non traitées

1. **`CLAUDE.md` cite la mauvaise issue upstream.** Le bug « Failed to delete all rows from index » est **duckdb/duckdb#23645**, pas **#23046** (corruption de tas en 1.5.0, symptôme différent, aucun lien). L'erreur est propagée dans une dizaine de fichiers (`append_only_rebuild.go`, `steps_player_*`, `no_art_patterns_test.go`, ADR 0019/0026/0030, `cmd/repair_psa_index`). **Non corrigé** (hors périmètre, et c'est une décision de nommage qui touche des ADR).
2. **Le bug est OUVERT dans notre version.** DuckDB 1.5.5 embarque le bug ; le seul correctif candidat mergé (#24744) est postérieur au tag et il n'existe pas encore de 1.5.6. Décision à prendre par le pilote : suivre #23645, planifier la montée en 1.5.6 dès sa sortie, ou pinner. À noter aussi : un utilisateur signale que la fatale **s'échappe en `terminate()` C++** et tue le process embarqué — risque d'arrêt brutal du binaire Go, pas seulement d'une requête en erreur.
3. **`internal/scheduler/data_health_check.go:openDBShared` utilise `OpenReadOnly`, pas `OpenReadForQuery`.** Le commentaire l'assume (partage de l'instance en cache), mais `OpenReadForQuery` fait exactement cela **et** sait emprunter un handle RW. Conséquence mesurable : `auditPlayerBanners` et `auditTitleLUSRGaps` déclarent « non mesuré » des player DB qu'ils pourraient lire quand un sync les tient en RW. Ma garde utilise `OpenReadForQuery` ; **je n'ai pas migré les deux autres** (hors périmètre).
4. **Les mêmes 3 index sont déclarés dans 3 fichiers** (`PlayerPersonalScoreAwardsDDL`, le `PostSwap` de la migration, `psaIndexDDL` de `cmd/repair_psa_index/diag.go`). La troisième copie est documentée comme « recopiée à l'identique » mais n'est verrouillée par **aucun garde-rail** — c'est le motif « copy-paste config » de la grille CLAUDE.md, à la 3ᵉ copie. Non traité.
5. **`EXPLAIN` est trompeur sur DuckDB 1.5.5** pour les index scans (§3.1). Toute future analyse de plan dans ce projet doit utiliser `EXPLAIN ANALYZE`. Aucune documentation projet ne le dit aujourd'hui.
6. **`cmd/cleanup_orphan_match` et `cmd/cleanup_post_art` émettent un `DELETE` réel même en dry-run** (exécuté puis annulé, pour compter les lignes). Innocenté par S4 sur 1.5.5, mais c'est une écriture sur une table critique déclenchée par une commande présentée comme lecture seule. Non traité.
7. **`cmd/force_rebuild_art` existe** dans le dépôt — trace d'incidents ART antérieurs sur d'autres tables. Non exploré.

---

## 9. Décision demandée au pilote

La cause n'étant pas reproductible mais **connue et non corrigée en amont**, il n'y a **rien à changer dans le pattern d'écriture** : le chemin PSA est déjà INSERT-only, sans `UPDATE` ni `ON CONFLICT`, c'est-à-dire déjà hors du vecteur principal décrit par #23645. Les trois arbitrages qui restent :

1. **La garde reste-t-elle en alerte seule ?** C'est ce qui est livré, et c'est ma recommandation tant que la fréquence de récurrence n'est pas mesurée. Un utilisateur de #23645 a, lui, câblé le rebuild automatique sur son canari (« the fatal has not recurred ») — si la jauge remonte plusieurs fois, ce sera l'option à rouvrir, avec un kill-switch daté.
2. **Suivi upstream** : #23645 + la sortie d'une v1.5.6 contenant #24744 / #24755.
3. **Rectifier ou non la référence `#23046` → `#23645`** dans `CLAUDE.md`, les ADR 0019/0026/0030 et la dizaine de fichiers Go concernés. Chantier de nommage, à décider séparément.
