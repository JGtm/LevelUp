# ADR 0008 — DB schema multi-title : isolation par chemin FS, xuid_aliases globalisé

**Status** — Proposed (2026-04-29). Triggered by code review axe 2 (BLOQUANT « schéma DuckDB transverse sans `title_id` »).

**Deciders** — Guillaume (GS).

## Context

L'audit axe 2 a soulevé que les tables transverses (`match_registry`, `match_participants`, `medals_earned`, `killer_victim_pairs`, `highlight_events`) **ne portent pas de colonne `title_id`**. L'isolation est purement filesystem via `data/titles/{slug}/warehouse/shared_matches_v2.duckdb`.

Question initiale : faut-il ajouter `title_id` partout pour préparer le multi-titres ?

Analyse complémentaire pendant la revue :

- Chaque titre a sa propre arborescence DB sous `data/titles/{slug}/`. Le `PathResolver` (`internal/domain/title/registry.go`) compose ces chemins.
- Une connexion DuckDB ouvre **un seul fichier** à la fois (sauf `ATTACH` explicite).
- Une query `SELECT ... FROM match_registry` sur la DB Halo Infinite **ne peut physiquement pas** retourner des matchs d'un autre titre — ils sont dans un autre fichier sur disque.

**Cas particulier `xuid_aliases`** : le `xuid` est un identifiant **Microsoft/Xbox global** par construction. Le même compte Xbox utilisé sur Halo Infinite et Halo MCC produit le même xuid. Avoir 2 tables `xuid_aliases` dupliquées dans 2 DB par titre :
- duplique les données (gamertag, last_seen)
- crée un risque de divergence (gamertag mis à jour sur un titre, pas l'autre)
- ne reflète pas la sémantique réelle (le xuid n'appartient pas au jeu)

## Decision

**1. Pas de colonne `title_id` sur les tables transverses.**

L'isolation par chemin FS est suffisante. Coûts évités :
- Migration de schéma + backfill sur tables avec ~millions de lignes
- Risque de bugs (oubli `WHERE title_id = ?` dans un SELECT, INSERT incomplet)
- Storage cost : 4-8 bytes/ligne × N

**Cas cross-title** (futurs, ex: comparer KDA d'un joueur sur 2 jeux) : traités par `ATTACH` multi-DB et `'halo_infinite' AS title` ajouté **à la requête**, pas stocké en colonne.

```sql
-- Pattern cross-title sans title_id colonne
ATTACH 'data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb' AS hi;
ATTACH 'data/titles/halo_mcc/warehouse/shared_matches_v2.duckdb'      AS mcc;
SELECT 'halo_infinite' AS title, * FROM hi.match_participants WHERE xuid = ?
UNION ALL
SELECT 'halo_mcc'      AS title, * FROM mcc.match_participants WHERE xuid = ?;
```

**Règle architecturale du projet** : isolation par chemin FS, **pas par colonne**.

**2. Exception `xuid_aliases` : déplacement vers DB globale.**

Nouveau fichier : `data/global/xbox_aliases.duckdb` (chemin unique, **pas paramétré par titre**).

`PathResolver` gagne une méthode dédiée :

```go
// PathResolver
func (r *PathResolver) GlobalXuidAliasesDBPath() string  // data/global/xbox_aliases.duckdb
```

C'est la seule méthode du resolver qui ne prend pas de `titleSlug` en paramètre — elle marque sémantiquement le fait que les xuid sont globaux.

**3. Migration via `cmd/migrate-xuid-aliases-global`** (script one-shot) :

- Lire `xuid_aliases` de chaque `data/titles/{slug}/warehouse/shared_matches_v2.duckdb`
- Consolider dans `data/global/xbox_aliases.duckdb` avec dédup sur `xuid` (max `last_seen`)
- Drop les tables locales `xuid_aliases` après vérification + dry-run
- Idempotent (réexécutable sans corruption)
- Test sur fixture multi-titres (`synthetic_title_b` + `halo_infinite`)

**4. Refactor consommateurs** : tous les services qui lisent ou écrivent `xuid_aliases` (sync ingestion, `engagement_score_service`, lookups xuid → gamertag) migrent vers la DB globale.

## Consequences

### Positive

- Pas de migration de schéma sur les tables transverses (économie ~1 sem).
- Sémantique correcte : le xuid est global, pas game-dependent.
- Pas de duplication / risque de divergence sur les `xuid_aliases`.
- Cohérent avec la règle « isolation par chemin FS, pas par colonne ».

### Negative

- `PathResolver` gagne une singularité (méthode sans `titleSlug`) — nécessite documentation explicite.
- 1 fichier DB de plus à backuper (`data/global/xbox_aliases.duckdb`).
- Migration one-shot à exécuter sur toutes les instances (locale, staging, prod).
- Dégrade si on voulait à un moment passer à un schéma `title_id` colonne — il faudrait migrer les autres tables aussi (mais c'est précisément ce qu'on évite).

## Alternatives evaluated

| Alternative | Rejected because |
|---|---|
| **A) `title_id` colonne sur toutes les tables transverses** | Redondance avec path FS. Migration coûteuse pour aucun bénéfice immédiat. Risque bugs élevé. |
| **B) `xuid_aliases` dupliqué par titre (status quo)** | Sémantiquement incorrect, divergences possibles. |
| **C) `xuid_aliases` dans `metadata.duckdb`** | Mélange référentiels titre-spécifiques et identifiants globaux. Confusion. |
| **D) Service externe d'identité** | Surdimensionné pour le besoin. Pas d'autres backends à servir. |

## References

- Code review : `axe-2-multi-titres.md` (BLOQUANT schéma DuckDB transverse).
- Plan d'action : `PLAN_ACTION.md` P1.4, P5.
- Skill `db-schema` : `.claude/skills/db-schema/SKILL.md` (à mettre à jour avec ce nouveau chemin).
- PathResolver : `apps/go-api/internal/domain/title/registry.go`.
