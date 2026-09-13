# ADR 0026 — Tables append-only pour éradiquer le bug DuckDB ART #23645

- **Statut** : Accepté (campagne livrée 2026-06-21)
- **Contexte technique** : DuckDB 1.5.x file-backed, driver CGO
- **Remplace/complète** : `.ai/V7/PLAN_LUSR_ART_HOME_CRASH.md`, `.ai/V7/HANDOFF_APPEND_ONLY_ART_CAMPAIGN.md`, ADR 0019 (Collect→Persist)

## Problème

Sur DuckDB file-backed (1.5.x), l'enforcement d'une contrainte `PRIMARY KEY`/`UNIQUE`
passe par un index **ART** (Adaptive Radix Tree). Sous churn — `DELETE` ligne-à-ligne,
`UPDATE` d'une colonne indexée, `INSERT ... ON CONFLICT DO UPDATE`, `INSERT OR
REPLACE/IGNORE` — l'ART corrompt le heap (bug amont #23645) :

```
Failed to delete all rows from index
```

La base passe alors en `FATAL: database has been invalidated` et **toute l'app tombe**
(incident prod confirmé 2026-05-24 20:41:04 sur `match_skill_rank`). Le vecteur n'est PAS
théorique et n'est PAS limité aux gros volumes : un seul `ON CONFLICT DO UPDATE`
concurrent suffit.

## Décision

Deux familles de tables, deux traitements :

### 1. Tables d'ÉTAT (réécrites par clé fonctionnelle) → **append-only**

La sémantique « remplacer la version courante d'une clé » est conservée **sans aucun
DELETE/UPDATE/UPSERT** :

- **PK technique** `id BIGINT` adossée à une séquence (jamais de PK fonctionnelle).
- **Colonne d'horloge/version** (`written_at`, `generation_id`, ou `stage`).
- **Écritures = INSERT pur** uniquement.
- **Lecture via une vue `<table>_latest`** qui ne retient que la version courante par clé.

L'ART ne reçoit jamais de `delete-from-index` → le bug devient **impossible par
construction**.

### 2. Tables de RÉFÉRENCE / cache → **SELECT-then-write**

Faible pression concurrente : on lit l'existant puis `UPDATE`-or-`INSERT` ciblé (pas de
`ON CONFLICT`). Hors périmètre de cet ADR (voir `.ai/V7/audit_art_writes.md`).

## Les 4 mécanismes de discrimination « version courante »

Le choix dépend de la sémantique métier de la réécriture :

| Mécanisme | Quand | Discriminant | Vue `_latest` | Tables |
|---|---|---|---|---|
| **written_at** | « dernier-écrit gagne » par clé (1 ligne/clé) | `written_at TIMESTAMP` | `ROW_NUMBER() OVER (PARTITION BY <clé> ORDER BY written_at DESC, id DESC) = 1` | `match_skill_rank`, `player_csr_snapshots`, `match_csrs`, `pve_match_stats`, `lusr_component_history` |
| **generation_id** | remplacer l'ENSEMBLE des lignes d'une clé en bloc atomique (N lignes/clé) | `generation_id BIGINT` (1 valeur par appel d'écriture) | `DENSE_RANK() OVER (PARTITION BY <clé> ORDER BY generation_id DESC) = 1` | `personal_score_awards` (+ `is_tombstone` pour l'extraction vide), `match_citations` |
| **decode_pass** | la ligne est le PRODUIT d'un décodage, et une passe doit pouvoir RÉTRACTER ce qu'elle ne retrouve plus | `decode_pass VARCHAR NOT NULL` (1 valeur par passe de décodage d'un match) | dernière passe ENTIÈRE par match : `QUALIFY decode_pass = FIRST_VALUE(decode_pass) OVER (PARTITION BY match_id ORDER BY written_at DESC, id DESC)` | `match_kill_events`, `kill_openings`, `kill_positions` |
| **stage merge-on-read** | colonnes écrites par des chemins DISTINCTS, à fusionner | `stage VARCHAR` + `written_at` | `ROW_NUMBER()` par `(<clé>, stage)` puis `GROUP BY <clé>` + `COALESCE` par colonne selon priorité de stage | `player_match_enrichment` |

### `decode_pass` — le seul mécanisme qui sait RÉTRACTER

Les trois autres mécanismes arbitrent **par clé** : ils savent servir la dernière valeur
d'une ligne, jamais constater son ABSENCE. C'est suffisant tant qu'une réécriture réécrit
toujours les mêmes clés — et c'est faux dès que les lignes sont le produit d'un décodage.

`replay.BuildKillPositions` n'écrit aucune ligne pour une mort dont ni le tueur ni la
victime n'ont pu être localisés (bornes de trajectoire, joueur non résolu, film
re-téléchargé plus court), et un décodeur amélioré en écarte d'autres. Sous un arbitrage
par clé, une position que la nouvelle passe ne retrouve plus n'est pas réécrite — elle est
simplement absente — et la vue continue donc de servir **à jamais** la ligne de la passe
précédente, mêlée aux nouvelles. **Une valeur fausse survit à sa propre correction.**

`decode_pass` déplace l'unité d'arbitrage de la clé vers **la passe** : la vue retient la
dernière passe entière d'un match et ignore toutes les précédentes, donc une ligne
qu'une passe neuve n'écrit pas disparaît de la vue. L'écriture reste un INSERT pur — c'est
toujours le même append-only, avec un discriminant qui porte sur un ENSEMBLE.

Trois points de vigilance, tous payés sur pièces :

- **Migrer une table existante exige un rebuild CTAS, pas un `ALTER TABLE ADD COLUMN`.** La
  colonne est `NOT NULL` (une ligne sans passe serait injointable au mécanisme) et un
  `DEFAULT` constant fondrait toutes les lignes existantes dans UNE passe inter-matchs — la
  première passe neuve d'un match retirerait alors de la vue les lignes de tous les autres.
  Le CTAS permet une expression PAR LIGNE : `'legacy-' || match_id`, soit une passe
  synthétique par match, ce qui rend exactement l'état d'avant migration jusqu'à ce qu'une
  passe neuve remplace CE match. Cf. `steps_shared_kill_positions_pass.go`.
- **Une table à passé peut avoir besoin d'un second étage de déduplication.** Regrouper tout
  le passé d'un match dans une seule passe synthétique y fait cohabiter des doublons de clé
  que l'ancienne vue par clé masquait. La vue de `kill_positions` ajoute donc un
  `ROW_NUMBER() = 1` par `(match_id, decode_pass, killer_xuid, time_ms)` À L'INTÉRIEUR de la
  passe retenue : no-op sur toute passe bien formée, et incapable par construction de
  ressusciter une ligne d'une passe antérieure.
- **Le step porte un nom NEUF.** Les migrations sont name-keyed : modifier
  `shared_append_only_kill_positions_v1`, déjà appliquée en local et en prod, ne rejouerait
  rien et ferait diverger deux schémas.

**`DENSE_RANK` vs `ROW_NUMBER`** : générationnel = on veut TOUTES les lignes de la
dernière génération (pas une par clé) → `DENSE_RANK`. written_at = exactement une ligne
par clé → `ROW_NUMBER` (départage par `id DESC`).

**Cas « vide »** : en append-only un INSERT vide ne supprime rien. Deux solutions selon
le mécanisme : ligne **tombstone** (`is_tombstone=TRUE`, filtrée par la vue — cas PSA) ou
ligne **sentinelle** déjà filtrée par les readers (`_processed` — cas citations).

**Sentinelle de reset (watermark)** : pour `player_skill_state_v2`, le « reset » n'efface
pas — il INSÈRE une ligne `is_reset=TRUE` que la vue filtre. Même principe append-only.

## Le helper unique — `append_only_rebuild.go`

7 des 8 conversions partagent une mécanique de swap IDENTIQUE, factorisée dans
`rebuildAppendOnlyTx(ctx, db, spec)` / `applyAppendOnlyRebuild(db, spec)` :

1. `recoverOrphanAppendOnly` en tête — répare un crash mid-swap antérieur (table absente
   + `<table>__appendonly` orpheline → rename).
2. Idempotence — colonne marqueur présente → refresh la vue + no-op.
3. Swap CTAS **TRANSACTIONNEL** : `BeginTx` → crée séquences → `CREATE TABLE __appendonly
   AS SELECT [id,] <cols> [, synthétiques]` → **garde anti-perte `rebuilt == before`
   AVANT tout DROP** → `DROP` ancienne → `RENAME` → PK + index + defaults → `Commit`.
   Rollback intégral sur la moindre erreur.

Le `spec` (struct `appendOnlyRebuild`) ne porte que ce qui varie : table, séquences,
colonnes synthétiques, marqueur d'idempotence, `PostSwap` (defaults/index verbatim),
`ViewSQL`.

> **Avant le helper**, 5 de ces conversions (`match_skill_rank`, `player_csr_snapshots`,
> `match_csrs`, `pve_match_stats`, `lusr_component_history`) étaient une suite de
> `db.ExecContext` **non transactionnelle** qui `DROP`ait l'ancienne table **avant**
> toute vérification de cardinalité, **sans rollback ni recoverOrphan** → perte de données
> possible sur erreur/crash mid-swap. Le helper supprime cette asymétrie de sûreté.

### Exception documentée : `player_match_enrichment`

PME n'utilise PAS le helper. Sa vue `_latest` est un **merge-on-read par colonne**
(plusieurs chemins d'écriture renseignent des colonnes différentes, fusionnées par
priorité de `stage`), pas une simple dernière-version. Son swap reste bespoke dans
`steps_player_append_only_match_enrichment.go` (déjà transactionnel + recoverOrphan).

## Ajouter une nouvelle table append-only (recette)

1. Choisir le mécanisme (written_at / generation_id / stage).
2. Écrire une migration `applyAppendOnly<Table>` qui appelle `applyAppendOnlyRebuild` avec
   son `spec` (cf. `steps_shared_pve_append_only.go` pour le cas simple, PSA pour le
   générationnel).
3. Convertir TOUS les writers en INSERT pur (jamais `DELETE`/`UPDATE`/`ON CONFLICT`).
4. Faire pointer TOUS les readers sur la vue `<table>_latest`.
5. Ajouter la table à l'allowlist du garde-rail `internal/sync/append_only_state_guard_test.go`.

## Pièges (anti-récidive)

- **`;` dans un commentaire SQL** : le splitter `execScript` est naïf (coupe sur `;`).
  Ne jamais mettre de `;` dans un commentaire `--` à l'intérieur d'un script passé à
  `execScript`. Les migrations append-only contournent le piège en passant chaque
  statement à `ExecContext` un par un (pas de `execScript`).
- **`CREATE TABLE IF NOT EXISTS` + PK** : la PK n'est jamais ajoutée à une table
  préexistante → `ON CONFLICT` échouera silencieusement. Cf. ADR antérieurs.
- **Lecture brute au lieu de `_latest`** : lire `<table>` directement expose les versions
  périmées. Toujours la vue.

## Conséquences

- **Positif** : le bug ART est éliminé par construction sur les tables d'état ; sûreté de
  migration uniforme (transactionnel + garde + recoverOrphan) ; 8 copies de swap → 1 helper.
- **Coût** : les tables croissent (N versions). Acceptable (volumes par joueur faibles) ;
  un compactage périodique pourra être ajouté si besoin (les vues `_latest` restent stables).
- **Garde-rail** : `internal/sync/append_only_state_guard_test.go` interdit la
  réintroduction de `DELETE`/`UPDATE`/`ON CONFLICT` sur les tables d'état.
