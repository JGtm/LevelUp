# ADR 0017 — Pattern de rebuild pour défaire la corruption d'index ART DuckDB

**Date** : 2026-05-22
**Status** : ✅ **CLOSED (2026-05-24)** — superseded by ADR 0019 + Phase 4 (post-sync batch INSERT-only). L'auto-heal runtime au boot a été SUPPRIMÉ le 2026-05-24 (Phase 5 cleanup) suite à validation empirique de Phase 4 (16 syncs / 0 FATAL). Le rebuild ART reste disponible comme **outil ops one-shot** via `cmd/force_rebuild_art/` (étendu 2026-05-24 pour rebuild aussi `match_skill_rank`). Requis avant tout déploiement Phase 4 sur DB héritée d'un mode legacy (corruption pré-existante).
**Branches** : `fix/duckdb-art-corruption-rebuild` (match_participants), commits `2e0f0247` + `651b9de6` (career_progression)
**Related** : ADR 0008 (multi-title DB schema), ADR 0016 (SharedDBProvider B-swap), **ADR 0019 (Collect→Persist, qui rend ce pattern obsolète)**
**Incidents** : [docs/INCIDENT_2026-05-20_match_participants_index.md](../../.ai/archive/stabilisation-2026-05-22/INCIDENT_2026-05-20_match_participants_index.md)

## Context

DuckDB indexe les clés primaires VARCHAR avec un **Adaptive Radix Tree (ART)** stocké directement dans le fichier `.duckdb`. Pour certaines combinaisons de données insérées au fil du temps (notamment via le sync engine), cet arbre peut devenir **corrompu** — les requêtes avec filter pushdown sur la PK retournent un sous-ensemble strict des rows réelles.

Deux occurrences confirmées en production (2026-05-20 et 2026-05-21) :

| Fichier `.duckdb` | Table | PK | Symptôme |
|------|------|----|----------|
| `shared_matches_v2.duckdb` | `match_participants` | `(match_id, xuid)` | `WHERE match_id = '50cd2d8c-...'` → 1 row au lieu de 10 |
| `data/players/JGtm/stats.duckdb` | `career_progression` | `xuid` implicite | `WHERE xuid = '2533274823110022'` → 86 rows au lieu de 106 |

Workaround `WHERE col || '' = ?` force un table-scan physique (court-circuite l'ART) et retourne les rows correctes — preuve que la donnée existe mais que l'arbre ART ne pointe pas dessus.

**Conséquence métier confirmée** : LUSR Madina97294 figé en Argent IV (μ ≈ 1327) au lieu de fin Platine / début Diamant attendu. Le `loadLUSRParticipants` chargeait 1-2 participants au lieu de 8-16 pour les matchs concernés → carry-adj et splitParticipantKEs dégénérés.

## Decision

**Définir un pattern de migration `rebuild_<table>_defeat_art_corruption`** appliqué via le système de migration standard (`internal/migration/`). Idempotent via sentinel dans `sync_meta`.

### Procédure swap

```sql
-- 1. Lire la liste réelle des colonnes (robuste aux ALTER TABLE futurs)
PRAGMA table_info('<table>');

-- 2. Rebuild via CREATE AS SELECT (CTAS) — force un table-scan complet
--    qui court-circuite l'ART et lit les pages physiques
CREATE TABLE <table>__rebuilt AS SELECT <cols> FROM <table>;

-- 3. DROP table corrompue (cascade automatique sur les vues dépendantes)
DROP TABLE <table>;

-- 4. RENAME → nom original
ALTER TABLE <table>__rebuilt RENAME TO <table>;

-- 5. Recréer la PRIMARY KEY (l'index ART vierge sera construit sur des
--    données complètes lues physiquement)
ALTER TABLE <table> ADD PRIMARY KEY <pk>;

-- 6. Recréer les vues + indexes secondaires (cascade DROP cleanup)
```

### Sentinel d'idempotence

```sql
-- À la fin du rebuild
INSERT INTO sync_meta (key, value, updated_at)
VALUES ('<table>_rebuilt_v1', 'true', NOW())
ON CONFLICT DO UPDATE SET value = 'true', updated_at = NOW();
```

Au boot suivant, la migration check le sentinel et no-op si déjà appliqué.

### Filet de garde au boot (Phase 1 plan stabilisation)

Un helper générique `duckdb.ProbeARTDivergences` scanne au boot toutes les tables avec PK VARCHAR et compare `COUNT(*) WHERE pk = ?` vs `COUNT(*) WHERE pk || '' = ?`. Si divergence détectée :
- `slog.WarnContext` structuré (table, pk_column, sample_value, count_indexed vs count_scan)
- Métrique expvar `art_corruption_detected_<db>_<table>` incrémentée

Câblé dans `cmd/server/main.go` après ouverture de `shared_matches_v2` + `metadata`. Non-bloquant : le serveur démarre.

## Rationale

**Pourquoi pas `REINDEX` / `ALTER ... DROP PRIMARY KEY ; ADD PRIMARY KEY`** ?
DuckDB n'a pas d'équivalent à `REINDEX TABLE` de PostgreSQL. Et `ALTER ... ADD PRIMARY KEY` reconstruit l'arbre ART en relisant les rows **via la structure interne existante** — si cette structure est corrompue, la reconstruction peut reproduire le même arbre défectueux.

**Pourquoi `CREATE AS SELECT *` (CTAS)** ?
Le `SELECT *` **sans clause `WHERE`** force un **table-scan physique complet** : DuckDB lit les pages de données séquentiellement, sans consulter l'index ART. La nouvelle table reçoit un index ART **vierge** construit sur des données complètes lues depuis les pages physiques.

**Pourquoi `PRAGMA table_info` plutôt qu'un schéma figé** ?
Les tables évoluent via `ALTER TABLE ADD COLUMN`. Énumérer dynamiquement les colonnes garantit que le rebuild ne perd aucune colonne ajoutée après l'écriture de la migration.

## Limitations

- Pas de fix in-place : le swap nécessite suffisamment d'espace disque pour 2× la table.
- Les contraintes (DEFAULT, CHECK) ne sont pas préservées par CTAS — seuls les types le sont. Pour les tables avec DEFAULT importants, lister les colonnes explicitement.
- Si une autre table a une FK vers la table corrompue, le DROP peut casser. Vérifier les FK dans le schéma avant écriture de la migration. (Actuellement non-applicable au projet — DuckDB ne supporte pas les FK enforced.)

## Investigation upstream

La cause racine du bug ART reste inconnue. Hypothèses :
1. Bug driver `marcboeker/go-duckdb` v2.10502 (le plus probable — symptôme déclenché par crash sync engine ou concurrent write sans isolation correcte).
2. Bug DuckDB engine lui-même (peu probable — DuckDB en v1.x est stable).
3. Pattern d'insertion adversaire (taille de liste IN spécifique) qui déclenche un edge case planner.

**Action** : ouvrir une issue upstream avec repro minimal après prochaine montée de version DuckDB. Référence projet : commits `2e0f0247` + branche `fix/duckdb-art-corruption-rebuild`.

## Consequences

**Positives** :
- Pattern reproductible — toute table avec PK VARCHAR corrompue peut être rebuilt par dédupliquant cette migration.
- Filet de garde au boot détecte automatiquement les corruptions futures (sans attendre un user report).
- Idempotent — la migration peut être enregistrée définitivement dans le code, elle no-op sur les DBs déjà rebuild.

**Négatives** :
- Pattern réactif (rebuild après corruption détectée), pas préventif. Sans fix upstream `duckdb-go`, le bug peut récidiver après une nouvelle vague d'inserts.
- Les workarounds applicatifs (`WHERE col || '' = ?`) appliqués entre la détection et le rebuild restent dans le code (cf. `qLoadLastCareerRank` dans `career_live_repo.go`). À nettoyer une fois la cause racine fixée upstream.
