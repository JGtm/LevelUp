# Audit — Erreurs DuckDB au boot (ATTACH global + shared.medals_earned + media_files manquante)

**Date** : 2026-05-21
**Branche** : `fix/csr-placement-alignment`
**Source** : `logs/duckdb.log` lignes 15-31 (boot serveur Go à 11:13:34, suivi de queries player à 11:13:45)
**Statut** : Diagnostic — fix non implémenté (audit demandé par l'utilisateur)

---

## TL;DR

Trois bugs **indépendants** observés au même boot serveur :

| # | Symptôme | Cause racine | Périmètre | Priorité |
|---|----------|--------------|-----------|----------|
| **1** | `Unique file handle conflict: Cannot attach "global"` + fallback ping `schema "global" does not exist` | `attachGlobalXuidAliases` n'est pas idempotent à travers (a) la limitation DuckDB "single-instance-per-file" au niveau process, (b) le pool `sql.DB` multi-conns physiques | Tous les joueurs au boot ; player conn ET social conn | **HIGH** (cross-cutting, garanti à reproduire) |
| **2** | `Table with name media_files does not exist` (player DB JGtm) | Migration `create_base_player_schema` ([steps_player.go:67-75](apps/go-api/internal/migration/steps_player.go#L67-L75)) n'a pas pu s'appliquer sur la DB de JGtm OU cette DB a été régénérée hors flow normal | Spécifique à JGtm (à confirmer pour d'autres joueurs) | **MEDIUM** (un seul joueur impacté observé) |
| **3** | `Catalog Error: schema "shared" does not exist` sur `FROM shared.medals_earned me` | Query `Q26hMatchMedalsTemplate` ([queries_home_citations.go:95-103](apps/go-api/internal/platform/duckdb/queries_home_citations.go#L95-L103)) exécutée sur **player conn** alors que l'ATTACH shared a été retiré du pool (ADR 0016 / sprint P0→P7). Call site oublié non listé dans les audits existants. | Home page de chaque joueur | **HIGH** (régression utilisateur visible) |

Les trois sont visibles dans les logs ci-dessous, **dans le même boot**, ce qui les fait ressembler à un seul incident — mais ce sont trois bugs sans lien causal direct.

---

## 1. Bug — ATTACH global xbox_aliases (Unique file handle conflict)

### 1.1 Symptôme

```
{"time":"11:13:34.4618601","level":"ERROR","msg":"duckdb: exec failed",
 "path":"…/shared_social.duckdb","op":"OpenReadWriteShared",
 "query_excerpt":"ATTACH '…\\data\\global\\xbox_aliases.duckdb' …",
 "err":"Binder Error: Unique file handle conflict: Cannot attach \"global\" -
        the database file \"…\\xbox_aliases.duckdb\" is already attached
        by database \"global\""}

{"time":"11:13:34.4633006","level":"WARN","msg":"pool: ATTACH global xbox_aliases échoué (social conn)",
 "err":"attach global: Catalog Error: Table with name \"global.xuid_aliases\" does not exist
        because schema \"global\" does not exist.
        LINE 1: SELECT COUNT(*) FROM global.xuid_aliases
                                     ^
        (attach err: Binder Error: Unique file handle conflict: …)"}
```

Le pattern se reproduit pour chaque joueur (gamertag=`JGtm`, `Chocoboflor`, `Madina97294`, `XxDaemonGamerxX` au minimum), à la fois sur la **player conn** ([pool.go:268](apps/go-api/internal/platform/duckdb/pool.go#L268)) et la **social conn** ([pool.go:280](apps/go-api/internal/platform/duckdb/pool.go#L280)).

### 1.2 Cause racine

#### 1.2.A — Limitation driver `go-duckdb` : single-instance-per-file

Quand un fichier `.duckdb` est ouvert par un handle dans le process, le driver maintient une **instance DuckDB en mémoire partagée**. Un second handle vers le même fichier réutilise cette instance. Conséquence : `ATTACH 'X.duckdb' AS global` est interprété au niveau de l'**instance partagée**, donc une seule fois autorisée pour le tuple (fichier, alias) dans tout le process.

Séquence observée :

```
T0  RunPlayerMigrations(JGtm)          → ouvre xbox_aliases (init schema)
                                          via initGlobalXuidAliasesSchema (pool.go:357)
                                        → defer db.Close() supprime du cache openDBs,
                                          mais l'instance DuckDB peut survivre
                                          si d'autres handles la référencent
T1  openPlayerDB(JGtm)                  → player conn fait ATTACH → succès
                                        → social conn (shared_social, cacheKey "rw:"+path)
                                          fait ATTACH → ??? (déjà attaché depuis player conn)
T2  openPlayerDB(Chocoboflor)           → player conn (différente) fait ATTACH
                                        → ÉCHEC "Unique file handle conflict"
                                          (xbox_aliases.duckdb est déjà attaché ailleurs
                                           dans la même instance partagée DuckDB)
```

#### 1.2.B — Fallback ping cassé par le pool `sql.DB` multi-conns

Le fallback dans [pool.go:340-346](apps/go-api/internal/platform/duckdb/pool.go#L340-L346) :

```go
_, err := db.Exec(ctx, "ATTACH '%s' AS global", ...)
if err != nil {
    var count int
    pingErr := db.QueryRow(ctx, "SELECT COUNT(*) FROM global.xuid_aliases").Scan(&count)
    if pingErr != nil {
        return fmt.Errorf("attach global: %w (attach err: %v)", pingErr, err)
    }
}
```

Le problème : `db.Exec` et `db.QueryRow` partent dans le **pool `sql.DB`** ([db.go:124-165](apps/go-api/internal/platform/duckdb/db.go#L124-L165)), qui peut router sur des conns physiques différentes :
- `OpenReadWriteShared` → `maxOpenConns=4` ([db.go:106](apps/go-api/internal/platform/duckdb/db.go#L106))
- `OpenReadWrite` → `maxOpenConns=1` ([db.go:121](apps/go-api/internal/platform/duckdb/db.go#L121))

`ATTACH 'X' AS global` est appliqué **uniquement sur la conn physique qui exécute la commande**. Le ping qui suit peut tomber sur une autre conn physique du même pool sql.DB → `global` n'y est pas connue côté connexion (même si l'instance DuckDB la connaît) → `schema "global" does not exist`.

#### 1.2.C — Cache `*DB` partagé entre joueurs sur shared_social

Aggravant : [db.go:106](apps/go-api/internal/platform/duckdb/db.go#L106) (`OpenReadWriteShared`) et [db.go:121](apps/go-api/internal/platform/duckdb/db.go#L121) (`OpenReadWrite`) partagent la même `cacheKey` `"rw:" + path`. Pour `shared_social.duckdb`, **tous les joueurs reçoivent le même `*DB`** via le cache `openDBs` ([db.go:131-140](apps/go-api/internal/platform/duckdb/db.go#L131-L140)) → `attachGlobalXuidAliases` est rappelée à chaque ouverture player, mais l'ATTACH ne peut réussir qu'une fois.

### 1.3 Pistes de fix (à arbitrer)

| Option | Effort | Tradeoff |
|--------|--------|----------|
| **A — Faire ATTACH sur une conn physique unique (`db.Conn()`)** | S | Évite le split Exec/QueryRow entre conns. Mais ne résout pas le conflict "déjà attaché par un autre `*DB`" du process. Insuffisant seul. |
| **B — ATTACH idempotent au niveau process** : exécuter `attachGlobalXuidAliases` **une seule fois par process** (au boot, après ouverture de la 1ère player DB), pas par joueur. Stocker un flag global `sync.Once`. | S | Simple, élimine les conflits 2-N. Mais l'instance globale doit rester ouverte tout au long du process → fuite de conn si pas géré proprement. |
| **C — Hook driver-level via `connector` callback** : à chaque nouvelle conn physique du pool, exécuter `ATTACH 'X' AS global`. Mais reste le problème "déjà attaché ailleurs". | M | Couvre le multi-conns mais pas le multi-handles. |
| **D — Réécrire le pattern global comme un SharedReader dédié** : exposer `XboxAliasesReader` avec une conn unique cachée, et faire les queries `JOIN global.xuid_aliases` à travers cette indirection (comme SharedReader pour `shared.*`). Suppression complète de l'ATTACH. | L | Solution propre, alignée sur l'archi post-sprint. Plus de travail (queries à réécrire). |
| **E — Connection string `?access_mode=read_only` + reverse-attach** : ne pas attacher xbox_aliases vers les player DBs, mais **attacher les player DBs sous xbox_aliases** lors des requêtes qui en ont besoin. Demande une refonte des call sites. | XL | Très intrusif, peu probable d'être justifié ici. |

**Recommandation provisoire** : **B** (idempotence process-level via `sync.Once` + flag global) pour stopper l'hémorragie, puis évaluer **D** à long terme une fois que la liste des consommateurs `global.xuid_aliases` est stabilisée.

### 1.4 Tests à ajouter

Reproducer minimal :

```go
// poc_attach_global_conflict_test.go (à créer)
// Vérifie qu'ouvrir 2 player DBs successives ne casse pas attachGlobalXuidAliases.
func TestAttachGlobal_TwoPlayersSequential(t *testing.T) {
    globalPath := filepath.Join(t.TempDir(), "xbox_aliases.duckdb")
    p1Path := filepath.Join(t.TempDir(), "p1_stats.duckdb")
    p2Path := filepath.Join(t.TempDir(), "p2_stats.duckdb")
    // openPlayerDB(p1) ; openPlayerDB(p2) ; assert no error.
}
```

---

## 2. Bug — `media_files` table manquante (player DB JGtm)

### 2.1 Symptôme

```
{"time":"11:13:45.6546614","level":"ERROR","msg":"duckdb: query failed",
 "path":"…/players/JGtm/stats.duckdb","op":"OpenReadWrite",
 "query_excerpt":"SELECT mf.file_name, mma.match_id, mma.match_start_time
                   FROM media_files mf LEFT…",
 "err":"Catalog Error: Table with name media_files does not exist!
        Did you mean \"milestone_earned\"?
        LINE 6: FROM media_files mf
                     ^"}
```

### 2.2 Constats

- La migration [`create_base_player_schema`](apps/go-api/internal/migration/steps_player.go#L14-L77) crée bien `media_files` (lignes 67-75) avec `CREATE TABLE IF NOT EXISTS`.
- Elle est censée tourner via [`RunPlayerMigrations`](apps/go-api/cmd/server/main.go#L607-L616) à l'ouverture de la player DB.
- Le suggérer DuckDB `Did you mean "milestone_earned"?` confirme que **d'autres** tables (`milestone_earned`) existent dans cette DB, donc le fichier n'est pas vide — c'est une absence ciblée de `media_files`.
- Le schéma de la query (`mf.file_name`) **diffère** de la définition de migration (`filename` sans underscore, [steps_player.go:69](apps/go-api/internal/migration/steps_player.go#L69)). Si la table existait, on aurait une erreur "Column file_name does not exist", pas "Table does not exist". Le mismatch de nom de colonne est un **second bug** latent à corriger.

### 2.3 Hypothèses

1. La player DB de JGtm a été **régénérée** par un script externe (restore, repair, backup) sans rejouer les migrations.
2. `RunPlayerMigrations(JGtm)` a échoué en cours de route (transaction non-atomique : certaines migrations ont passé, d'autres pas).
3. Une migration ultérieure a **DROP** `media_files` puis échoué à le recréer.

### 2.4 Actions de diagnostic à mener

```bash
# Inspecter le schéma réel de la player DB de JGtm
duckdb "data/titles/halo_infinite/players/JGtm/stats.duckdb" \
  "SELECT table_name FROM information_schema.tables WHERE table_schema='main' ORDER BY 1"

# Comparer avec la liste attendue (cf. CLAUDE.md § stats.duckdb)
# Si media_files absente : forcer RunPlayerMigrations sur ce gamertag
go run ./cmd/repair_data_consistency --gamertag JGtm
```

### 2.5 Bug latent — `file_name` vs `filename`

À résoudre indépendamment :
- Migration : `filename VARCHAR NOT NULL` ([steps_player.go:69](apps/go-api/internal/migration/steps_player.go#L69))
- Query : `SELECT mf.file_name, ...`

Soit la migration crée la mauvaise colonne, soit la query attend la mauvaise colonne, soit une migration ultérieure renomme `filename` → `file_name`. À tracer.

---

## 3. Bug — `Q26hMatchMedalsTemplate` exécutée sur player conn

### 3.1 Symptôme

```
{"time":"11:13:45.8231062","level":"ERROR","msg":"duckdb: query failed",
 "path":"…/players/JGtm/stats.duckdb","op":"OpenReadWrite",
 "query_excerpt":"SELECT me.match_id, me.medal_name_id, COALESCE(me.count, 1) AS count
                   FROM shared.medals_earned me LEFT…",
 "err":"Catalog Error: Table with name \"shared.medals_earned\" does not exist
        because schema \"shared\" does not exist.
        LINE 6: FROM shared.medals_earned me
                     ^"}
```

### 3.2 Localisation

Query définie : [queries_home_citations.go:95-103](apps/go-api/internal/platform/duckdb/queries_home_citations.go#L95-L103) — `Q26hMatchMedalsTemplate`.

```sql
-- Q26h : Home — médailles par match pour un joueur, lots de match_id.
-- Paramètres : ?1 = xuid. Les match_id sont injectés dynamiquement via IN (%s).
-- Requête sur pdb.Player (shared attaché) ; labels résolus ensuite via metadata.
const Q26hMatchMedalsTemplate = `
SELECT
    me.match_id,
    me.medal_name_id,
    COALESCE(me.count, 1) AS count
FROM shared.medals_earned me
WHERE me.xuid = ?
  AND me.match_id IN (%s)
ORDER BY me.match_id, me.count DESC`
```

Le **commentaire ligne 94** (`Requête sur pdb.Player (shared attaché)`) est **obsolète depuis ADR 0016**. L'ATTACH `shared` a été retiré du pool ([pool.go:273-275](apps/go-api/internal/platform/duckdb/pool.go#L273-L275)) :
> *"attachShared sur SharedSocial retiré aussi. media_repo passe désormais entièrement par SharedReader pour les queries shared.* — *plus aucune conn du pool ne porte d'ATTACH shared."*

### 3.3 Pourquoi pas déjà capturée par l'audit existant

`Q26h` n'est **pas** mentionnée dans [`.ai/V7/AUDIT_SHARED_READER_LEAKS.md`](.ai/V7/AUDIT_SHARED_READER_LEAKS.md) (section 1.1 ne liste que les `q37LegacyMediaFromClause` / `q37SharedSocialFromClause`) ni dans [`.ai/V7/AUDIT_SHARED_READER_GAPS.md`](.ai/V7/AUDIT_SHARED_READER_GAPS.md). **Call site oublié** lors du sprint P0→P7.

À traquer côté caller : qui appelle `Q26hMatchMedalsTemplate` et l'exécute sur quelle conn ? Probablement dans `home_repo_*` ou un repo medals/citations. À auditer.

### 3.4 Autres call sites suspects à auditer (chasse aux régressions)

Patterns `FROM shared.medals_earned` trouvés dans le code runtime :

| Fichier | Ligne | Statut à vérifier |
|---------|------|-------------------|
| [queries_home_citations.go:17](apps/go-api/internal/platform/duckdb/queries_home_citations.go#L17) | Q26 perfect CTE | À vérifier (exécution sur quelle conn ?) |
| [queries_home_citations.go:100](apps/go-api/internal/platform/duckdb/queries_home_citations.go#L100) | **Q26h** (confirmé bug) | À MIGRER |
| [queries_home_citations.go:520](apps/go-api/internal/platform/duckdb/queries_home_citations.go#L520) | Q? medal_id sum | À vérifier |
| [queries_squad.go:109](apps/go-api/internal/platform/duckdb/queries_squad.go#L109) | Sub-select perfect kills | À vérifier |
| [medals_by_xuid_repo.go:116](apps/go-api/internal/platform/duckdb/medals_by_xuid_repo.go#L116) | `LoadMedalsForMatchesByXUID` | À vérifier (probablement OK via SharedReader, mais préfixe `shared.` redondant à retirer) |

(Section à compléter via l'audit complémentaire — cf. §5.)

### 3.5 Fix proposé

Pour `Q26h` :
1. Retirer le préfixe `shared.` → `FROM medals_earned me`.
2. Faire passer l'exécution par `pdb.SharedReader.Get(ctx)` au lieu de `pdb.ReadDB()`.
3. Mettre à jour le commentaire de la query (lignes 93-94 : `Requête sur SharedReader ; labels résolus côté player via metadata.`).

Cf. modèle déjà appliqué dans `Q26CareerTopEncountersTpl` et `Q27CareerRivalsTpl` ([career_repo.go:391, 493](apps/go-api/internal/platform/duckdb/career_repo.go#L391)).

---

## 4. Corrélations entre les 3 bugs

Aucun lien causal direct, mais ils **interagissent au boot** :

```
T0  Boot serveur
T+0.00s  Init xbox_aliases.duckdb (OpenReadWrite + Close)
T+0.46s  openPlayerDB(JGtm)
         ├─ Bug 1.A : ATTACH global échoue (already attached)
         └─ Bug 1.B : fallback ping échoue (autre conn physique)
T+0.46s  openPlayerDB(Chocoboflor) — même bug 1
T+0.46s  openPlayerDB(Madina97294) — même bug 1
T+0.46s  openPlayerDB(XxDaemonGamerxX) — même bug 1
T+11s    Premier appel handler home pour JGtm
         ├─ Bug 2 : SELECT media_files → table inconnue
         └─ Bug 3 : Q26h SELECT shared.medals_earned → schema inconnu
```

**Risque caché** : si le bug 1 cascade et marque la player conn comme invalidée (cf. [`IsInvalidatedError`](apps/go-api/internal/platform/duckdb/db.go#L182), [`Reopen`](apps/go-api/internal/platform/duckdb/db.go#L205)), toutes les migrations player suivantes pourraient échouer silencieusement → **explique potentiellement le bug 2** (migration `media_files` jamais appliquée à JGtm car la conn était cassée). À investiguer.

---

## 5. Audit complémentaire `shared.*` (call sites hors SharedReader)

Audit ciblé via agent d'exploration sur `apps/go-api/internal/{platform/duckdb,sync,service,api}/*.go` (hors `*_test.go` et `cmd/`). Croisé avec les audits existants ([SHARED_READER_LEAKS](.ai/V7/AUDIT_SHARED_READER_LEAKS.md), [SHARED_READER_GAPS](.ai/V7/AUDIT_SHARED_READER_GAPS.md)).

### 5.1 Régressions confirmées (queries `shared.X` exécutées sur player conn `pdb.ReadDB()`)

| # | Fichier:Ligne | Fonction | Query / Tables shared | Statut |
|---|---|---|---|---|
| R1 | [home_repo_matches.go:17](apps/go-api/internal/platform/duckdb/home_repo_matches.go#L17) | `LoadHomeMatches` | `Q26HomeMatches` — `FROM shared.match_participants, shared.match_registry` | **Régression** (non observée dans les logs présents — la page d'accueil n'a peut-être pas été chargée) |
| R2 | [home_repo_matches.go:98](apps/go-api/internal/platform/duckdb/home_repo_matches.go#L98) | `CountPlayerMatches` | `Q26bCountPlayerMatches` — `SELECT FROM shared.match_participants` | **Régression** (non observée — idem) |
| R3 | [home_repo_medals_citations.go:95](apps/go-api/internal/platform/duckdb/home_repo_medals_citations.go#L95) | `LoadMatchMedals` | `Q26hMatchMedalsTemplate` — `FROM shared.medals_earned` | **Régression confirmée** (cf. §3, observée dans logs 11:13:45) |

Toutes les régressions sont concentrées dans le module **home_repo** (page d'accueil). Pattern : ces 3 fonctions exécutent leur SQL sur `pdb.ReadDB()` (= player conn) alors qu'elles devraient utiliser `pdb.SharedReadDB().Get(ctx)` comme déjà fait dans `squad_repo`, `medals_by_xuid_repo`, `filters_repo`, etc.

### 5.2 Sites vérifiés OK (passent bien par SharedReader)

L'agent a inspecté les autres fichiers contenant `FROM shared.X` dans le runtime et confirme qu'ils passent par SharedReader (préfixe `shared.` parfois redondant mais non bloquant car il est réécrit / la conn pointe sur shared) :

- `medals_by_xuid_repo.go` (LoadMedalsForMatchesByXUID)
- `squad_repo.go` (queries squad)
- `filters_repo.go`
- `engagement_score_repo.go` (déjà couvert par audit GAPS §1.3, en cours de migration)
- `progression/profile/queries.go` (déjà couvert par audit GAPS §1.4)
- `media_repo.go` (Q37 via runMediaPipeline — déjà couvert par audit GAPS §1.1)

### 5.3 Plan de fix bug 3 — batch unique sur home_repo

Migrer les 3 fonctions de `home_repo` en une seule passe (même module, même pattern) :

1. **`LoadHomeMatches`** ([home_repo_matches.go:17](apps/go-api/internal/platform/duckdb/home_repo_matches.go#L17)) — basculer sur `pdb.SharedReadDB().Get(ctx)`, retirer le préfixe `shared.` dans `Q26HomeMatches`.
2. **`CountPlayerMatches`** ([home_repo_matches.go:98](apps/go-api/internal/platform/duckdb/home_repo_matches.go#L98)) — idem pour `Q26bCountPlayerMatches`.
3. **`LoadMatchMedals`** ([home_repo_medals_citations.go:95](apps/go-api/internal/platform/duckdb/home_repo_medals_citations.go#L95)) — idem pour `Q26hMatchMedalsTemplate`.

Attention si une query mixe tables player + shared (à vérifier sur Q26 — `LEFT JOIN player_match_enrichment` et `LEFT JOIN match_skill_rank` côté player) : pattern à scinder en 2 phases Go (cf. audit LEAKS §1.A pour le pattern leaderboard).

Modèle à suivre : `Q26CareerTopEncountersTpl` / `Q27CareerRivalsTpl` ([career_repo.go:391, 493](apps/go-api/internal/platform/duckdb/career_repo.go#L391)).

---

## 6. Plan d'attaque suggéré (à arbitrer avec l'utilisateur)

| Phase | Bug | Action | Effort |
|------|-----|--------|--------|
| **P0** | 1 | Implémenter idempotence `sync.Once` process-level pour `attachGlobalXuidAliases` | S (≤ 30 min) |
| **P0** | 3 | Migrer `Q26h` vers SharedReader (modèle Q26 career) | S (≤ 30 min) |
| **P1** | 3 | Audit complet des `shared.*` restants (cf. §5) et migration en batch | M (≤ 2 h) |
| **P1** | 2 | Diagnostic schéma JGtm/stats.duckdb + replay `RunPlayerMigrations` | S (≤ 30 min) |
| **P2** | 2 | Fix mismatch colonne `filename` vs `file_name` (migration ou query, à arbitrer) | S |
| **P2** | 1 | Évaluer migration vers pattern SharedReader dédié pour `xbox_aliases` (cf. §1.3 option D) | L (planification ADR) |
| **P3** | tous | Tests de régression (reproducer multi-joueurs + check schémas player DB) | M |

---

## Références

- ADR 0008 — multi-title & xuid globalisé : [docs/adr/0008-db-schema-multi-title-and-xuid-global.md](docs/adr/0008-db-schema-multi-title-and-xuid-global.md)
- ADR 0016 — SharedDBProvider RO↔RW swap : [docs/adr/0016-shared-db-provider-b-swap.md](docs/adr/0016-shared-db-provider-b-swap.md)
- Audits précédents `shared.*` :
  - [.ai/V7/AUDIT_SHARED_READER_LEAKS.md](.ai/V7/AUDIT_SHARED_READER_LEAKS.md)
  - [.ai/V7/AUDIT_SHARED_READER_GAPS.md](.ai/V7/AUDIT_SHARED_READER_GAPS.md)
- Code clés :
  - [pool.go:225-295](apps/go-api/internal/platform/duckdb/pool.go#L225-L295) — `openPlayerDB`, séquence ATTACH
  - [pool.go:325-370](apps/go-api/internal/platform/duckdb/pool.go#L325-L370) — `attachGlobalXuidAliases`, `initGlobalXuidAliasesSchema`
  - [db.go:73-165](apps/go-api/internal/platform/duckdb/db.go#L73-L165) — `OpenRead*`, `openCachedDB`, cache `openDBs`
  - [queries_home_citations.go:92-103](apps/go-api/internal/platform/duckdb/queries_home_citations.go#L92-L103) — `Q26hMatchMedalsTemplate`
  - [steps_player.go:14-77](apps/go-api/internal/migration/steps_player.go#L14-L77) — migration `create_base_player_schema`
