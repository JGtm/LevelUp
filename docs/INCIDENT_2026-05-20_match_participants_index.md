# Incident — Bug de filter pushdown sur `match_participants` (shared_matches_v2.duckdb)

**Date du diagnostic** : 2026-05-20
**Sévérité** : Élevée — fausse les calculs LUSR et potentiellement 8 autres pipelines
**Statut** : Diagnostic complet, correctif non appliqué (action destructive, attente go/no-go)
**Outil de diag** : [`apps/go-api/cmd/diag_lusr_player/`](../apps/go-api/cmd/diag_lusr_player/)

---

## TL;DR

Dans `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`, les requêtes `SELECT ... FROM match_participants WHERE match_id = ?` (ou `IN (...)`) **ne retournent pas toutes les rows attendues**. Pour le match de référence `50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e` qui contient 10 participants en réalité :

- `WHERE match_id = '50cd2d8c-...'` → **1 row** (au lieu de 10)
- `WHERE match_id || '' = '50cd2d8c-...'` → **10 rows** ✓

Le concat `|| ''` défait le filter pushdown du moteur DuckDB et force un table-scan complet. Les rows existent bien : c'est l'optimisation d'index ART (Adaptive Radix Tree) de la clé primaire `(match_id, xuid)` qui ment.

**Conséquence métier** : le LUSR de MAdina97294 est calculé sur des données massivement tronquées (un seul participant vu par match au lieu de 10), ce qui produit un Argent IV en arena_slayer alors que c'est le meilleur joueur du squad. Les autres joueurs sont moins impactés visiblement parce que le bug se déclenche **selon la composition de la liste IN passée**, et la liste de Madina (1079 IDs) déclenche le mauvais plan d'exécution là où celles de Chocoboflor (389 IDs) et JGtm (766 IDs) ne le déclenchent pas.

---

## Symptômes observés en prod (côté user)

- MAdina97294 → LUSR `Argent IV` (μ ≈ 1327) en `arena_slayer`
- Chocoboflor → LUSR `Or III` (μ ≈ 1476)
- JGtm → LUSR `Or V` (μ ≈ 1538)
- Selon le user (4 amis qui jouent toujours ensemble), Madina est le meilleur des 4. Le classement actuel est **inversé**.

## Diagnostic technique

### Étape 1 — Hypothèse formule (rétractée)

Le composite score (`apps/go-api/internal/sync/skill_rating.go:142-255`) contient un bloc *carry-adj* sur la composante `kills_vs_expected` qui compresse le score vers 0.5 d'autant plus fort que le joueur est noté fort par Halo (KE élevé). On a supposé que cette pénalité asymétrique expliquait le LUSR bas de Madina.

**Test** : replay TrueSkill avec et sans carry-adj sur tout l'historique des 3 joueurs.

**Résultat** : carry-adj retiré → Madina passe de 1327.3 à 1331.8 (toujours Argent IV). Hypothèse démentie. Le carry-adj ne se déclenche jamais pour Madina parce que sa condition (`teammateAvgKE > 0`) n'est jamais remplie : `loadLUSRParticipants` ne ramène aucun de ses teammates.

### Étape 2 — Investigation côté données

`dumpParticipants(match_id='50cd2d8c-...')` retourne **1 seul row** : `bid(45.0)` (un bot). Mais `SELECT WHERE xuid='2533274858283686' AND match_id='50cd2d8c-...'` retourne bien la row de Madina avec kills=12, KE=10.5. Donc la row **existe**.

### Étape 3 — Identification du bug d'index

Tests successifs avec différentes formulations SQL — toutes lues sur la même connection, même DB, même match_id, accès `READ_ONLY` :

| # | Requête | Plan utilisé | Résultat |
|---|---|---|---:|
| 1 | `WHERE match_id = '50cd2d8c-...'` (literal) | index lookup | **1 row** |
| 2 | `WHERE match_id = ?` (placeholder) | index lookup | **1 row** |
| 3 | `WHERE match_id LIKE '50cd2d8c-...'` | index lookup | **1 row** |
| 4 | `WHERE match_id IN ('50cd2d8c-...')` | index lookup | **1 row** |
| 5 | `WHERE match_id IN ('50cd2d8c-...', 'dummy')` | index lookup | **1 row** |
| 6 | `WHERE match_id IN (?)` (placeholder) | index lookup | **1 row** |
| 7 | `WITH t AS (SELECT * FROM mp) SELECT FROM t WHERE match_id = '...'` | index lookup | **1 row** |
| 8 | `WHERE match_id = '...' ORDER BY rowid` | index lookup | **1 row** |
| 9 | `WHERE match_id = '...' OR FALSE` | index lookup | **1 row** |
| 10 | `WHERE substring(match_id, 1, 36) = '...'` | full scan | **10 rows** ✓ |
| 11 | `WHERE match_id \|\| '' = '...'` | full scan | **10 rows** ✓ |
| 12 | `WHERE match_id \|\| '' IN (?)` | full scan | **10 rows** ✓ |
| 13 | `SELECT COUNT(*) WHERE match_id = '...'` | index lookup | **1** (le COUNT lit aussi l'index) |
| 14 | `SELECT xuid, COUNT(*) ... GROUP BY xuid` | index lookup | **1 row** |
| 15 | `WHERE xuid = '...' AND match_id = '...'` | xuid-prefix scan | 1 ✓ (row trouvée) |

**Verdict** : toute requête qui peut être optimisée via filter pushdown sur `match_id` ne retourne qu'une row. Toute requête qui force un table-scan retourne les rows correctes.

### Étape 4 — Pourquoi MAdina97294 spécifiquement

Test de probe : pour chaque joueur du squad, ré-exécuter `loadLUSRParticipants` (mêmes chunks de 500, même SQL `IN (?,?,...)`) et compter les rows visibles pour le match `50cd2d8c-...`.

| Joueur | xuid | match_ids dans IN | total rows ramenés | visibles pour 50cd2d8c |
|---|---|---:|---:|---:|
| Chocoboflor | 2535469190789936 | 389 | 3526 | **10 ✓** |
| JGtm | 2533274823110022 | 766 | 7134 | **10 ✓** |
| **MAdina97294** | **2533274858283686** | **1079** | **19764** | **1 ✗** |

Test additionnel : modifier la taille de chunk pour Madina (50, 100, 200, 500, 1000) — **le résultat reste 1 row dans tous les cas**. Donc ce n'est pas simplement « liste trop grosse ». C'est **le contenu de la liste de Madina** qui déclenche le mauvais plan.

Test ultime : la même boucle avec chunk=500 mais SQL `WHERE match_id || '' IN (?,?,...)` → **10 rows visibles ✓**.

→ Le bug est **plan-dependent** : pour certaines combinaisons d'IDs, le moteur DuckDB choisit un plan d'index lookup qui rate des rows. Pour d'autres combinaisons, il utilise un plan correct. Madina a 3× plus de matchs LUSR-éligibles que Chocoboflor et 1.4× plus que JGtm — la composition de sa liste IN tombe dans un cas où le planner se trompe.

La cause exacte (collision de hash dans l'ART, pruning incorrect, bug du driver duckdb-go v2.10502, etc.) demande une investigation côté moteur qui dépasse le scope de ce diagnostic. Mais le **symptôme est reproductible** et le **contournement est connu**.

---

## Impact

### Calcul LUSR (confirmé)

`loadLUSRParticipants` ([apps/go-api/internal/sync/skill_rating_loaders.go:114-150](../apps/go-api/internal/sync/skill_rating_loaders.go#L114-L150)) charge en moyenne 1-2 participants au lieu de 8-16 pour les matchs concernés. Pour Madina spécifiquement, presque tous ses matchs sont impactés.

Effet en cascade dans `computeSkillRatingsBatch` :

1. `splitParticipantKEs(playerTeamID, parts)` → reçoit `parts` avec 1 row (souvent un bot avec KE=0, donc filtré)
2. `teammateKEs = []`, `enemyKEs = []`
3. `teammateAvgKE = nil` → carry-adj ne se déclenche jamais
4. `computeEnemyStrength([], …)` tombe sur le fallback `(playerMU, DefaultOpponentSigma)` — force adverse « neutre » par défaut
5. Le composite est calculé uniquement sur les composantes intrinsèques du joueur (kills/KE, deaths/DE, win_factor, etc.) sans contexte lobby
6. Pour un joueur Halo-noté fort, KE est élevé → kills/KE ≈ 1 même quand il écrase le lobby → score ≈ 0.5 → μ stagne ou descend

**Cohérence avec les symptômes** : Madina à 1327 (Argent IV) est très exactement ce qu'on obtient quand le composite oscille autour de 0.5 sur ~1000 matchs en partant de InitialMU=1500, avec une légère pente négative due aux pertes et morts excessives en sous-perf vs MMR personnel élevé.

### Autres pipelines potentiellement impactés

`grep "match_participants.*WHERE.*match_id" apps/go-api/internal/` retourne 9 fichiers :

| Fichier | Risque |
|---|---|
| `sync/skill_rating_loaders.go` | **CONFIRMÉ** — calcul LUSR faussé |
| `sync/engine.go` | À auditer — agrégats, sync engine |
| `sync/events_replay.go` | À auditer — replay des events filmés |
| `sync/backfill_weapons.go` | À auditer — backfill weapon stats |
| `platform/duckdb/engagement_score_repo_queries.go` | À auditer — engagement scores |
| `api/registry_notifications.go` | À auditer — notifications |
| `sync/bitmask_honesty_test.go` | Test seulement |
| `sync/writes_test.go` | Test seulement |
| `sync/backfill_integration_test.go` | Test seulement |

Tous les pipelines de production qui passent par `match_participants ... WHERE match_id` sont susceptibles d'avoir des données tronquées. À auditer un par un.

---

## Incident connexe confirmé — `career_progression` (2026-05-21)

Le lendemain du diagnostic, un second agent a découvert la **même classe de bug** sur une table différente :

- **Table** : `career_progression` dans chacun des 3 `data/players/{gamertag}/stats.duckdb`
- **PK corrompue** : index ART implicite sur `xuid`
- **Symptôme** : `WHERE xuid = ?` retournait un sous-ensemble des rows

| Joueur | Visible via index | Réel (full scan) |
|---|---:|---:|
| JGtm | 86 | 106 |
| Chocoboflor | 67 | 78 |
| Madina97294 | 65 | 74 |

Pour Madina, les 9 rows manquantes étaient les snapshots récents portant la bannière — `ARG_MAX(banner_image_url)` piquait un snapshot du 2026-05-08 sans banner → identité Spartan vide.

**Fix appliqué** (commits `2e0f0247` + `651b9de6`) :
- Workaround permanent : `WHERE xuid || '' = ?` dans `qLoadLastCareerRank`
- Migration idempotente `rebuild_career_progression_defeat_art_corruption` : `CREATE __rebuilt → INSERT SELECT * → DROP → RENAME`, sentinel dans `sync_meta`

**Conséquence pour ce diagnostic** : la corruption ART n'est **pas isolée** à `shared_matches_v2.duckdb`. Elle touche plusieurs fichiers DuckDB du repo (`shared_matches_v2.duckdb`, `stats.duckdb` × N joueurs). L'investigation cause racine (step 6 ci-dessous) est critique — ce n'est pas un accident sur une table, c'est un pattern systémique.

---

## Pourquoi un rebuild complet est obligatoire

### L'index ART de DuckDB

DuckDB indexe les clés primaires avec un **Adaptive Radix Tree (ART)** — une structure en arbre radix adaptative stockée directement dans le fichier `.duckdb`, au même niveau que les pages de données. Pour la table `match_participants`, cet arbre mappe chaque valeur de `match_id` vers la liste des adresses physiques des rows correspondantes.

Quand DuckDB évalue `WHERE match_id = '50cd2d8c-...'`, le query planner voit que cette colonne est couverte par la PK et décide d'un **filter pushdown** : au lieu de scanner toutes les pages, il consulte l'arbre ART pour obtenir directement la liste des offsets. C'est ce mécanisme qui est cassé — l'arbre ne retourne qu'une adresse (1 row) là où il devrait en retourner 10.

### Pourquoi `ALTER TABLE ... ADD PRIMARY KEY` ne suffit pas

L'enchaînement :

```sql
ALTER TABLE match_participants DROP PRIMARY KEY;
ALTER TABLE match_participants ADD PRIMARY KEY (match_id, xuid);
```

reconstruit l'index ART **mais relit les rows via la structure interne existante** pour trouver les valeurs à indexer. Si cette structure sous-jacente est corrompue, la reconstruction peut reproduire exactement le même arbre défectueux en repassant par le même chemin de lecture.

DuckDB ne dispose pas d'une commande `REINDEX` équivalente à PostgreSQL (`REINDEX TABLE match_participants`). Il n'existe pas de chemin pour "réparer" un ART in-place.

### Pourquoi `CREATE TABLE AS SELECT *` fonctionne

```sql
CREATE TABLE _mp_new AS SELECT * FROM match_participants;
```

Ce `SELECT *` **sans clause `WHERE`** force un **table-scan physique complet** : DuckDB lit les pages de données séquentiellement, sans consulter l'index ART. Il voit donc les 10 rows de `50cd2d8c`, et l'intégralité des rows de tous les matchs.

La table `_mp_new` est créée avec un **index ART vierge**, construit sur des données complètes lues depuis les pages physiques. `DROP TABLE match_participants` supprime ensuite l'ancienne table *avec* son index corrompu. Le `ADD PRIMARY KEY` final crée un index propre sur une table propre.

C'est le seul chemin sûr dans DuckDB pour reconstruire un index ART corrompu.

---

## Plan de correction

### Étape 0 — Backup

```bash
cp data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
   data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb.bak-2026-05-20
```

### Étape 1 — Stopper le serveur sync

Éviter d'écrire dans `match_participants` pendant la reconstruction.

### Étape 2 — Reconstruire `match_participants`

Approche la plus sûre — recréer la table à partir d'un SELECT (force la reconstruction de l'index ART) :

```sql
CHECKPOINT;
CREATE TABLE _mp_new AS SELECT * FROM match_participants;
DROP TABLE match_participants;
ALTER TABLE _mp_new RENAME TO match_participants;
ALTER TABLE match_participants ADD PRIMARY KEY (match_id, xuid);
CHECKPOINT;
```

À noter : le `SELECT * FROM match_participants` lui-même utilise un table-scan (pas de WHERE), donc ramène **tous** les rows. C'est ce qui rend cette approche sûre.

Risque : si d'autres tables utilisent des FK vers `match_participants`, le `DROP TABLE` peut casser. À vérifier dans `steps_shared.go` avant exécution.

### Étape 3 — Valider

Re-run le cmd diag :

```bash
cd apps/go-api && go build -tags cgo ./cmd/diag_lusr_player/
cd ../.. && ./apps/go-api/diag_lusr_player.exe
```

Les tests doivent maintenant montrer :

- `[= literal]` → **10 rows** (au lieu de 1)
- `[IN (?)]` → **10 rows**
- `Madina chunk=500 (sans concat trick)` → **10** visibles pour 50cd2d8c

### Étape 4 — Backfill force LUSR

Une fois l'index réparé, recalculer tous les LUSR avec les vraies données :

```bash
# vérifier la commande exacte avant de l'exécuter — adapter au binaire de backfill
go run -tags cgo ./cmd/backfill_all -- --force-lusr
```

### Étape 5 — Audit défensif

Ajouter le contournement `|| ''` dans `loadLUSRParticipants` et auditer les 8 autres call-sites. Le table-scan est plus lent mais garantit la correction même si l'index se re-corrompt. Décision à arbitrer : performance vs robustesse.

```go
// dans skill_rating_loaders.go:121
query := "SELECT match_id, xuid, team_id, COALESCE(kills_expected, 0) FROM match_participants WHERE match_id || '' IN ("
```

### Étape 6 — Investigation cause racine

**Contexte mis à jour** : deux tables dans deux fichiers DuckDB distincts présentent le même symptôme (`career_progression` dans `stats.duckdb`, `match_participants` dans `shared_matches_v2.duckdb`). La probabilité d'un bug driver/moteur est donc élevée.

Pistes à creuser dans l'ordre :

1. **Bug driver `duckdb-go/v2 v2.10502.0`** — vérifier le changelog upstream pour des issues ART/filter-pushdown postérieures à cette version. C'est la piste la plus probable vu que le symptôme est reproductible sur des tables sans rapport, dans des fichiers différents, créés à des moments différents.

2. **Crash du sync engine** — chercher dans `data/logs/` un panic récent contenant `match_participants` ou `career_progression`. Un crash en plein INSERT peut laisser un index ART dans un état intermédiaire.

3. **Concurrent write sans isolation** — vérifier que le sync utilise bien des transactions explicites sur ces tables. Un write concurrent peut introduire des incohérences dans l'arbre si les pages ne sont pas verrouillées correctement.

Pas bloquant pour le fix immédiat, mais à creuser avant la prochaine montée de version DuckDB pour éviter récidive systémique.

---

## Annexes

### A — Commandes pour reproduire le bug

Pré-requis : être à la racine du repo `LevelUp-go-migration`, avoir build le cmd diag.

```bash
cd apps/go-api && go build -tags cgo ./cmd/diag_lusr_player/
cd ../.. && ./apps/go-api/diag_lusr_player.exe
```

Output attendu (extraits clés) :

```
══ Nature de match_participants ══
  databases attachées :
    - shared_matches_v2
  match_participants occurrences :
    - shared_matches_v2.main.match_participants (BASE TABLE)
  [= literal] err=<nil> :
    row 1 : "bid(45.0)"
    total=1
  [concat trick] err=<nil> :
    total=10

══ Diagnostic loadParticipants : rows réellement chargées pour le match 50cd2d8c-... ══
  Chocoboflor (xuid=2535469190789936) : 389 ids → visibles=10 ✓
  Madina97294 (xuid=2533274858283686) : 1079 ids → visibles=1 ✗
  JGtm (xuid=2533274823110022) : 766 ids → visibles=10 ✓
```

### B — Test SQL minimal en debug interactif

Si on a un duckdb CLI installé (pas dans ce repo) :

```sql
ATTACH 'data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb' AS db (READ_ONLY);
-- Doit retourner 10, retourne 1 :
SELECT COUNT(*) FROM db.match_participants WHERE match_id = '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e';
-- Doit retourner 10, retourne 10 :
SELECT COUNT(*) FROM db.match_participants WHERE match_id || '' = '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e';
```

### C — Inventaire des xuids du match de référence

Tirés du concat trick (ordre physique d'insertion) :

```
bid(45.0)                  ← bot ramené par l'index buggé
2535409561713955
2533274850178760
2535469190789936           ← Chocoboflor
2533274823110022           ← JGtm
2533274858283686           ← MAdina97294
2533274852144672
2535455227223597
2533274840182174
2533274910731403
```

### D — Pourquoi le tableau de replay montre quand même TM=6.2 pour Chocoboflor

C'est l'observation initiale qui a déclenché ce diagnostic. Le replay TrueSkill calcule pour chaque joueur dans le tableau, et pour Chocoboflor `splitParticipantKEs` reçoit bien JGtm (et un autre humain) car la liste IN de Chocoboflor (389 IDs) ne déclenche **pas** le mauvais plan. Pour Madina, sa liste IN (1079 IDs) le déclenche → `parts` quasi vide → TM=0.

C'est cette **asymétrie sur le même match** qui prouve que le bug dépend du contexte de la query, pas du contenu de la table.
