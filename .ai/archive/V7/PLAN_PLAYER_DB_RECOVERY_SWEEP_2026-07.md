# PLAN — Sweep recovery des lectures player-DB (race Reopen) (2026-07)

> Statut : TERMINÉ (code, 2026-07-16) — NON mergé (attend accord user ; merge main = deploy).
> Branche : `fix/player-db-recovery-sweep` (depuis main post-fix prestige `535ee437a`). Suite
> de la Découverte (d) du plan monitoring triage : le fix prestige (`acf60b179`, déployé prod)
> a corrigé UN instance ; ce sweep généralise. Exécution sous contrat `plan-execution`.
>
> **BILAN : 77 lectures player-DB routées en `*Recovered` (~40 fichiers) + garde-rail.**
> - **Lot A** `af6ccdd6` : career/home — 14 lectures / 10 fichiers.
> - **Lot B** `8750dfcc8` : match-view/stats — 17 lectures / 11 fichiers.
> - **Lot C** `15e8f7e72` : engagement/squad/citations — 37 lectures / 16 fichiers.
> - **Lot D** `b179b8b1` : garde-rail `player_db_recovery_routing_test.go` (ratchet formes
>   explicites `.Player`/`.ReadDB()` + allowlist datée sync_meta, 3 tests) + **9 STRAGGLERS**
>   convertis (dont `internal/api/wire/post_sync_deltas_snapshot.go` ×7, hors couche duckdb).
> - **Gates FINAUX verts** : `go build/vet/test ./...` + `go test -tags=integration -p 1
>   ./internal/platform/duckdb/... ./internal/sync/... ./internal/persist/...` = exit 0
>   (re-vérifié superviseur : build/vet + les 3 tests garde-rail).
> - HORS PÉRIMÈTRE (DC-3, laissés) : lectures shared/metadata/shared_social ; `sync_meta`
>   (2 allowlist) ; writes (lease/ART, DC-2). Handles bare-`db` (prestige/streaks/
>   record_history) convertis mais non couverts par le garde-rail (limite documentée).

## Contexte

`PlayerDB.Player` est l'UNIQUE handle RW du player `stats.duckdb`, partagé par tous les
repos joueur (`ReadDB()` le renvoie). Un writer concurrent qui déclenche `db.Reopen()`
(invalidation ART, ou fermeture transitoire) fait `old.Close()` (`db_recovery.go:102`) →
une lecture en vol sur ce handle voit `sql: database is closed`. Les repos qui utilisent
les méthodes PLATES `db.Query/QueryRow/Exec` (sans `WithReopenOnInvalidated`) ne
récupèrent pas → erreur remontée (bruit + données transitoirement absentes, PAS de
corruption). Le fix prestige a routé les repos Prestige vers les variantes `*Recovered`
(+ helper `QueryRowRecovered`). Ce plan généralise à TOUS les lecteurs player-DB.

Incident de référence (2026-05-25, Phase 5 ART) : « database is closed » massif sur
home + career + teammates + filters + explorer → `IsInvalidatedError` corrigé pour attraper
la signature ; mais les CALLERS doivent passer par les variantes recovered pour en profiter.

## Objectif et critère de succès

- **Objectif** : tout accès en LECTURE à un handle player-DB (`pdb.Player` / `pdb.ReadDB()`)
  passe par une variante `*Recovered` (`QueryRecovered` / `QueryRowRecovered`), pour tolérer
  un `Reopen()` concurrent. Un garde-rail interdit la régression.
- **Critère de succès** : (1) 0 lecture player-DB en méthode plate hors couche DB ;
  (2) garde-rail grep en place et vert ; (3) `go test ./...` + `-tags=integration -p 1
  ./internal/platform/duckdb/...` + `go vet` exit 0 ; (4) en prod, la famille
  `database is closed` (op=OpenReadWrite) tombe à ~0 (mesure T0+7j après merge).

## Décisions pré-tranchées (DC)

- **DC-1 — Approche : per-callsite `*Recovered`** (PAS un flag global sur `*DB`). Cohérent
  avec le fix prestige déjà en prod ; explicite ; évite de changer le comportement global du
  handle (le chemin d'ÉCRITURE sync/persist est lease/ART-critique — ne pas auto-recover en
  masse). Le helper `QueryRowRecovered` (déjà livré dans `db_query.go`) couvre les lectures
  mono-ligne.
- **DC-2 — Périmètre = LECTURES sur handle player-DB uniquement.** Les races observées en
  prod sont des lectures. Les ÉCRITURES player-DB (Create/Update/Delete) passent par le
  sync engine sous lease ou par PersistSink — NE PAS les toucher dans ce sweep (risque
  lease/ART ; traiter séparément si un besoin est prouvé). Exception déjà livrée : prestige
  a converti aussi ses writes (`ExecRecovered`) — laissé tel quel, pas de revert.
- **DC-3 — Scope handle : PLAYER-DB seulement.** Exclure les repos qui opèrent sur
  shared_matches_v2 (`pdb.SharedReadDB()`/SharedReader — déjà contrat swap-safe), metadata
  (RO), shared_social (Persister + CHECKPOINT, ADR 0022), sync_meta. Classer chaque fichier
  AVANT de convertir (méthodologie ci-dessous). En cas de doute : NE PAS convertir, consigner.
- **DC-4 — Garde-rail EN DERNIER** (après conversion de tous les lots) : un test grep échoue
  si une méthode plate `.Query(`/`.QueryRow(` est appelée sur un handle player-DB hors
  `internal/platform/duckdb` couche basse. Allowlist datée pour les rares exceptions justifiées.

## Méthodologie de classification (par fichier)

Pour chaque fichier de `internal/platform/duckdb/*.go` (hors `*_test.go`) listé à
l'inventaire : identifier la SOURCE du handle sur lequel `.Query/.QueryRow/.Exec` est
appelé.
- Handle = `r.pdb.Player`, `r.db` construit depuis `pdb.Player`/`ReadDB()`, `pdb.ReadDB()`
  → **PLAYER-DB : convertir les LECTURES** (Query→QueryRecovered ; QueryRow→QueryRowRecovered ;
  préserver le mapping `sql.ErrNoRows`→erreur domaine).
- Handle = `SharedReader`/`SharedReadDB()`/shared, metadata, shared_social, sync_meta,
  metadata_repo/notifications/social/media/persist_sink/prestige_metadata/prestige_social,
  csr_thresholds (metadata), asset/map/medal caches (metadata) → **HORS PÉRIMÈTRE**, ne pas
  toucher, cocher `[~]` avec la raison (type de handle).

Référence de conversion : le fix prestige `prestige_player_repo.go` (contrat exact de
`QueryRowRecovered` : vérifier `err`/`ErrNoRows` AVANT `defer rows.Close()` puis `Scan` SANS
re-`Next`).

## Inventaire (superset 76 fichiers / 210 occurrences — À CLASSER)

Candidats PLAYER-DB probables (à confirmer par classification) — lots proposés :

- **Lot A — Career/Home (hot-path HTTP, priorité)** : `career_repo.go`, `career_repo_csr.go`,
  `career_repo_csr_seasons.go`, `career_repo_highlights.go`, `career_live_repo.go`,
  `career_progression_partial.go`, `home_repo_identity.go`, `home_repo_cache.go`,
  `home_repo_skill_peak.go`, `home_repo_matches.go`, `home_repo_medals_citations.go`,
  `progression_diag_repo.go`.
- **Lot B — Match view / stats** : `match_view_repo.go`, `match_view_repo_medals.go`,
  `match_view_repo_scoreboard.go`, `match_view_repo_extras.go`,
  `match_view_repo_weapons.go`, `match_view_repo_neighbors_skill.go`, `stats_repo.go`,
  `player_matches_loaders.go`, `compare_repo.go`, `explorer_repo.go`, `patterns_repo.go`,
  `streaks_repo.go`, `records_repo.go`, `record_history_repo.go`,
  `personal_score_awards_repo.go`, `player_record_repo.go`.
- **Lot C — Engagement / squad / citations / achievements** : `engagement_score_repo.go`,
  `engagement_score_repo_queries.go`, `engagement_response_bins_repo.go`, `citations_repo.go`,
  `achievements_repo.go`, `milestones_earned_repo.go`, `squad_repo.go`,
  `squad_repo_mapstats.go`, `squad_v2_adapter.go`, `season_pass_repo.go`,
  `season_pass_repo_tracks.go`, `campaign_repo.go`, `privacy_state_repo.go` (LoadPrivacyState
  côté READ — le write est déjà recovered), `coach_proposal_repo.go`,
  `halo5/halo5_career_source.go`, `match_exclusion_repo.go`, `csr_coverage_repo.go`.

> Chaque fichier reste à classer PLAYER vs HORS-PÉRIMÈTRE (DC-3). Les compteurs par fichier
> sont dans le relevé grep du 2026-07-16 (76 fichiers, 210 occ.). Beaucoup de la liste
> metadata/social/media/notifications/persist_sink sont HORS périmètre.

## Lots d'exécution

Ordre : A → B → C (une étape close et gatée avant la suivante). Puis **Lot D — garde-rail**.
Chaque lot : classer, convertir les LECTURES player-DB, `go build` + `go vet` +
`go test ./internal/platform/duckdb/...` (lot ciblé). Commit par lot (préfixe `sweep(lotX)`).

- **Lot D — garde-rail** (après A/B/C) : test grep (`internal/platform/duckdb/`) qui échoue
  si `\.(Query|QueryRow)\(ctx` est appelé sur un handle identifié player-DB hors allowlist.
  Approche pragmatique si la détection statique « handle player » est trop fine : ratchet sur
  la LISTE des fichiers convertis (aucun retour à une méthode plate dans ces fichiers).
  Documenter l'allowlist datée.

## Gate final (Lot D clos)

```
cd apps/go-api && go test ./... && go vet ./...
cd apps/go-api && go test -tags=integration -p 1 ./internal/platform/duckdb/...
```
Tous exit 0. Puis entrée `.ai/thought_log.md` + point d'étape utilisateur. Landing =
accord user (merge main = deploy auto).

## Découvertes (à consigner, ne pas traiter)

- (à remplir en cours d'exécution : fichiers ambigus, handles mixtes, faux positifs grep.)

## Protocole de reprise

Lire ce plan (statuts) + `git log --oneline` sur `fix/player-db-recovery-sweep`. Reprendre au
premier lot non clos. Le fix prestige (`prestige_player_repo.go`) est la référence de forme.
