# PLAN — Éliminer la corruption d'index ART sur `player_match_enrichment`

> Statut : **proposé** (non démarré). Créé 2026-06-19.
> Branche cible : `refactor/pme-append-only` (à créer depuis `main` une fois la PR rendement/engagement mergée).

## Contexte / problème

`player_match_enrichment` (1 ligne par match, PK `match_id`, player DB) est la table d'enrichissement la plus écrite du projet : le sync live l'`INSERT`/`UPSERT`, et le post-sync l'`UPDATE` de façon **incrémentale** par plusieurs étapes distinctes (performance_score, sessions, engagement_score, friends, had_bot, exclusions, psa_checked_at…). Sa PK est maintenue par un **index ART** DuckDB.

Sous forte pression d'`UPDATE` (typiquement `engagement-coefs --with-scores` qui réécrit `engagement_*` sur des centaines de matchs), DuckDB **crashe** avec un FATAL :
```
Failed to append to PRIMARY_player_match_enrichment_0:
Constraint Error: PRIMARY KEY or UNIQUE constraint violation: duplicate key "<match_id>"
```
…alors que les **données n'ont aucun doublon** (`COUNT(*) == COUNT(DISTINCT match_id)`). C'est un **faux-positif au niveau de l'index ART**, pas une corruption de données.

### Cause racine = bug DuckDB amont (pas une erreur d'usage)

- Driver actuel : `github.com/duckdb/duckdb-go/v2 v2.10503.1` → **DuckDB 1.5.3**.
- Bug amont **OUVERT** : [duckdb/duckdb#23046 — « DuckDB 1.5.0's ART index constraint enforcement corrupts the heap on file-backed »](https://github.com/duckdb/duckdb/issues/23046). Régression introduite en **1.5.0** : l'enforcement de contrainte PK/UNIQUE via l'ART corrompt le heap sur DB **fichier** sous insertions massives. Notre crash en est la manifestation.
- **Pas de fix amont** : la [release v1.5.4](https://github.com/duckdb/duckdb/releases/tag/v1.5.4) (bugfix) **ne corrige pas** #23046 (aucune entrée ART/PK dans le changelog ; issue toujours OUVERTE). → On ne peut pas s'en sortir par un simple upgrade.
- Note historique : la version Python du projet (`duckdb==1.4.4`, LTS pré-1.5.0) n'avait pas cette régression ; c'est l'adoption de 1.5.x côté Go qui l'a introduite. Symptôme jumeau côté DELETE déjà combattu sur `shared.match_participants` (incident 2026-05-20, ADR 0018) et `career_progression`.

### Mitigation déjà en place (post-incident 2026-06-19)

- `cmd/rebuild_pme_art` + `migration.RebuildPlayerMatchEnrichmentART` (swap CTAS transactionnel, garde anti-perte, recrée indexes) — **réparation** ponctuelle. Ne **prévient** pas la récidive.

## Objectif

Supprimer le **vecteur de corruption** : que les écritures lourdes d'enrichissement ne puissent plus crasher ni corrompre l'index, **sans perte de données**, en attendant (ou indépendamment) d'un fix amont DuckDB.

Critère de succès : un `engagement-coefs --all --with-scores --force` (et tout backfill UPDATE-lourd) s'exécute **sans crash** sur des DB joueur volumineuses, de façon répétée.

## Inventaire des sites d'écriture (à traiter)

| Site | Type | Fichier |
|---|---|---|
| Persist INSERT (live) | INSERT | `internal/persist/player_persister.go` |
| Post-sync batch UPDATE | UPDATE | `internal/persist/post_sync_enrichment_persister.go` |
| Engagement score | UPDATE ×2 | `internal/platform/duckdb/engagement_score_repo.go` |
| Stubs / ensure rows | INSERT OR IGNORE | `internal/sync/ensure_enrichment_rows.go`, `fanout_repo.go`, `comeback_postsync_persist.go` |
| Sessions / friends / had_bot | UPDATE | `internal/sync/{friends_recompute,enrichments,convergence}.go` |
| Exclusions | UPSERT | `internal/platform/duckdb/match_exclusion_repo.go` |
| Import OpenSpartan | INSERT ON CONFLICT | `internal/service/openspartan_post_import_service.go` |

**Difficulté clé** : les colonnes sont écrites **par étapes différentes, à des moments différents** (perf score ≠ engagement ≠ friends). Toute refonte doit préserver cette sémantique d'« écriture partielle incrémentale ».

## Options

### Option A — Append-only + vue `_latest` (refonte stratégique, recommandée à terme)
Modèle INSERT-only versionné (PK technique `id` + `written_at`), lecture via vue `player_match_enrichment_latest`, exactement comme `match_skill_rank`/`match_csrs`/`pve_match_stats` (migrées en append-only, cf. ADR + `no_art_patterns_test.go`).
- **Élimine** totalement le churn UPDATE/DELETE sur la PK → plus de #23046.
- **Défi** : les écritures partielles incrémentales → besoin d'une stratégie **merge-on-read** (coalesce de la dernière valeur non-NULL par colonne) ou **carry-forward à l'insert** (relire la dernière version + réécrire la ligne complète). Impact sur ~120 lecteurs de `player_match_enrichment`.
- Effort : **lourd**.

### Option B — Sérialiser toutes les écritures via le BatchBuilder/persister (réduction de churn)
Router les `UPDATE` ad-hoc restants (friends, sessions, engagement…) par un **owner unique** (pattern Collect→Persist, ADR 0019) → zéro écriture concurrente. Réduit fortement la probabilité du bug mais **reste en UPDATE** → ne l'élimine pas.
- Effort : **moyen**. Bon palier intermédiaire.

### Option C — Auto-heal périodique (filet de sécurité)
Étendre `BootARTGuard` : détecter la divergence ART sur player DB au boot / avant tout backfill lourd, et lancer `RebuildPlayerMatchEnrichmentART` automatiquement (serveur arrêté ou fenêtre exclusive).
- Effort : **rapide**. Ne corrige pas la cause mais borne l'impact.

### Option D — Côté DuckDB
- **Suivre #23046** ; planifier un upgrade quand un fix amont sort (re-tester un `--with-scores` massif après upgrade).
- Évaluer un **downgrade 1.4.x LTS** (pré-régression) : ⚠️ risque de **format de fichier** (DBs écrites par 1.5.x potentiellement non relisibles par 1.4.x) → probablement **non viable** sans réécriture. À tester sur une copie avant toute décision.

## Recommandation (phasé)

1. **Phase 1 — Filet (rapide)** : Option C. Auto-heal player DB (boot + pré-backfill) via le tool existant. Documenter la procédure « backup + rebuild_pme_art avant tout backfill lourd ». *Livrable indépendant.*
2. **Phase 2 — Réduction churn (moyen)** : Option B. Centraliser les UPDATE d'enrichissement par le persister batché ; ajouter au garde-rail `no_art_patterns_test.go` l'interdiction de nouveaux UPSERT concurrents sur la table. *Livrable indépendant.*
3. **Phase 3 — Append-only (lourd)** : Option A, après design validé du merge-on-read (POC sur les colonnes engagement d'abord, puis extension). *Dépend de Phase 2.*
4. **Transverse — DuckDB** : surveiller #23046 ; re-tester à chaque montée de version ; trancher le downgrade LTS seulement après test format-fichier.

## Tests par phase

- Phase 1 : test d'intégration « rebuild auto déclenché sur corruption simulée » (étendre `steps_player_rebuild_match_enrichment_test.go`).
- Phase 2 : `internal/sync/no_art_patterns_test.go` (allowlist resserrée) + test persister batché.
- Phase 3 : tests read-path de la vue `_latest` (merge-on-read correct, dernière valeur non-NULL par colonne) + non-régression des ~120 lecteurs + `go test ./...`.
- Validation finale : `engagement-coefs --all --with-scores --force` ×3 sans crash sur copie des DB volumineuses (Madina 1183, JGtm 942).

## Blocages / risques

- **Écritures partielles incrémentales** = le vrai point dur de l'append-only (Phase 3). À POC avant de s'engager.
- **Lock-in format fichier 1.5.x** : limite l'option downgrade.
- **#23046 non résolu amont** : tant qu'il l'est, seules les mitigations app-side protègent.
- Aucune dépendance multi-titres ni frontend (travail 100% backend / platform).

## Done definition

- Un backfill UPDATE-lourd ne crashe plus (Phase 1+2 minimum) ; idéalement plus aucun churn PK-ART (Phase 3).
- Garde-rail `no_art_patterns_test.go` couvre la table.
- Entrée `.ai/thought_log.md` par phase.
