# HANDOFF — Refactor Match Timeline T0

> Pour l'agent qui reprend. Branche : `feat/match-timeline-t0`. Dernière session : 2026-05-29.
> À lire AUSSI : mémoires `reference-match-timeline-t0` + `data-quality-first-joined-tz`,
> thought_log entrées [2026-05-28] et [2026-05-29] (×5), `git log feat/match-timeline-t0`.

---

## 1. Objectif du refactor

Corriger les chronologies de match (badges narratifs, heatmap intensité, premiers events, durée affichée) en retranchant **T0 = countdown pré-match**.

**Découverte fondatrice** : `match_participants.first_joined_time` (joueurs `present_at_beginning`) − `start_time_utc` = T0 (médiane **28s**). Résout le problème « T0 inaccessible » documenté dans `.ai/MATCH_DURATION_RESEARCH.md`.

Stratégie **strangler fig** : mettre l'abstraction en place partout avec T0=0 (identité, Phase 1), peupler la donnée (Phase 2), puis **activer** en un point (Phase 3).

---

## 2. État actuel — FAIT (10 commits, tout testé, build vert)

| # | Livré | Détail |
|---|-------|--------|
| 1 | `domain.MatchTimeline` | struct {DurationMs, T0Ms} + `CorrectEventTime`, `GameplayDurationMs`, `IsValid`. 7 tests. |
| 2 | `analysis/timeline.CorrectEvents` | corrige les `[]canonical.HighlightEvent` en amont (1 point/service). 8 tests. |
| 3 | Câblage 3 services | TimeseriesService, SquadServiceV2, MatchViewService — `CorrectEvents` appliqué au chargement des events. Fonctions analysis INCHANGÉES. |
| 4 | Golden baseline | `apps/go-api/internal/testdata/t0_fixtures/` (8 matchs réels, `golden_output.json`). Régénérer : `UPDATE_GOLDEN=1 go test ./internal/analysis/timeline/ -run TestGolden`. |
| 5 | `analysis/timeline.ComputeT0` | filet multi-joueurs (bots exclus via xuid non-numérique ; ≥2 sources spread≤2s→min, sinon médiane ; rejets negative/suspicious_high/no_data). 9 tests. |
| 6 | Migration `t0_quality` + `cmd/backfill_t0` | EXÉCUTÉ `--commit` : **1712/1724 T0 (99.3%)** stockés dans `match_registry.real_start_time` (= début gameplay UTC). |
| 7 | `cmd/backfill_first_joined_tz` | EXÉCUTÉ `--commit` : **19114 lignes** corrigées (décalage TZ Europe/Paris sur 964 matchs anciens). |
| 8 | Wiring sync | `sync/transforms.go::ExtractRegistry` remplit `real_start_time` via `computeMatchT0` (ART-safe, avant Submit). |
| 9 | Fix `AvgLifeSeconds` | `squad_breakdown.go` : canonical OK (vraie moyenne API), legacy = nil (honnête). |

**Backups restic** (repo local `data/backups/restic-repo`) : `5da3eb3a` (avant re-normalisation first_joined), `4c57d755` (avant backfill_t0).

**IMPORTANT** : le comportement runtime est **inchangé** (identité) car `phase1T0Ms()` retourne toujours 0. La donnée est en place mais pas encore lue. **Zéro risque en l'état.**

---

## 3. Découvertes clés / pièges (NE PAS refaire les erreurs)

1. **Phase 3 ≠ simple flip de `phase1T0Ms()`** ⚠️ — `canonical.MatchSummary` (dans `PlayerMatchRow.Summary`) ne porte PAS `real_start_time`. Les services Timeseries/Squad consomment des `PlayerMatchRow` → il faut d'abord **propager** la valeur jusqu'au canonical. Voir §4.
2. **Bug TZ `first_joined_time`** (RÉSOLU en base) : décalé de l'offset Europe/Paris (+1h CET / +2h CEST) sur 964 matchs anciens (héritage). Le code sync actuel est SAIN (vérifié empiriquement : `parseISO` `.UTC()` + driver DuckDB préserve l'instant). Règle de correction : `offset_local = epoch(start_time AT TIME ZONE 'UTC') − epoch(start_time_utc)`.
3. **Détection bot** : xuid NON-numérique (format `bid(...)`). PAS `gamertag IS NULL` (le gamertag est NULL pour ~tous les vrais joueurs dans match_participants, résolu via xuid_aliases).
4. **Invariance des badges** : l'identité des gagnants de badges (first_blood, top_gun…) est invariante par T0 (soustraction d'une constante préserve l'ordre). Seuls les `TimeMS` affichés se décalent.
5. **`AvgLifeSeconds`** doit venir de la valeur API (`r.Self.AvgLifeSeconds`), jamais recalculé depuis `time_played/n`.

---

## 4. RESTE À FAIRE

### 4.A — Phase 3 : activation T0 runtime — ✅ FAIT (2026-05-29, 2 commits)

**LIVRÉ** : commit 1 `5ce87e2d` (plomberie T0Ms → canonical + repos), commit 2
`09a11014` (activation build.go + golden). `phase1T0Ms()` supprimé. Golden revu :
first_kill/first_death décalés d'exactement le T0 par match (Fortress 41b61fb9
T0=27918ms → first_kill 37861→9943). Suite complète verte.
**RESTE pour clôturer** : validation UI des 3 pages (point 6 ci-dessous) — agent
ne peut pas, à faire par l'utilisateur.

<details><summary>Détail historique du plan (réalisé)</summary>

**But** : `BuildTimelines*` lit le vrai T0 au lieu de `phase1T0Ms()=0`.

Étapes :
1. **Ajouter le T0 au canonical** : champ dans `canonical.MatchSummary` (ex. `RealStartTimeUTC *time.Time` ou `T0Ms *int64`). Politique additive (cf. ADR 0005).
2. **Propager via les repos** : le(s) repo(s) `platform/duckdb/` qui chargent `PlayerMatchRow` doivent SELECT `real_start_time` + `start_time_utc` et remplir le nouveau champ (T0Ms = `epoch_ms(real_start_time AT TIME ZONE 'UTC') − epoch_ms(start_time_utc)`, nil si real_start_time NULL).
3. **Lire dans build.go** :
   - `BuildTimelinesFromPlayerMatches` → `r.Summary.T0Ms` (au lieu de `phase1T0Ms()`).
   - `BuildFromRegistry` → depuis `MatchRegistryRow.RealStartTime` − start_time_utc.
   - `BuildForMatchMs` (MatchView) → depuis `meta` (a déjà real_start_time ? sinon l'ajouter à MatchMetaRaw).
4. **Supprimer `phase1T0Ms()`** une fois tous les sites basculés.
5. **Régénérer le golden** (`UPDATE_GOLDEN=1`) et **reviewer les diffs** : first_kill_ms doit baisser de ~T0, buckets intensité recalibrés. Vérifier sur match Fortress `41b61fb9` (T0=28s → first_kill 37861→~9861).
6. **Validation UI** (ne PEUT PAS être faite par l'agent — demander à l'utilisateur) : 3 pages — MatchView timeline, SquadV2 timeline/cadence/intensity, Synthesis/Timeseries first-events distribution.

Critère succès : timelines/badges recalés sur le gameplay réel, golden diffs cohérents avec les T0 par match, suite verte. ✅ Atteint (golden diffs = soustraction pure du T0 par match).

</details>

### 4.A-bis — ✅ FAIT (2026-05-29) : TeammatesService câblé T0 (4ᵉ consommateur)

**LIVRÉ** (branche `chore/query-devtools-flag`, commit `83424e377`). Le graphe teammates.17 « premier frag/première mort » (`SquadFirstEventsChart`) affichait les temps DEPUIS start_time (countdown inclus) — confirmé par l'utilisateur sur Fortress. Corrigé.

Implémentation (cf. thought_log [2026-05-29] fix(squad,t0)) :
1. `domain.SquadMatchRow.T0Ms *int64` + colonne `t0_ms` dans `Q30SquadMatchesSharedQuery` (formule canonique identique à `player_matches_repo`) + scan dans `loadSquadMatchesShared`.
2. Helpers miroir Phase 3 : `timeline.BuildTimelinesFromSquadRows([]SquadMatchRow)` + `timeline.CorrectImpactEvents([]ImpactEventRow, timelines)` (point unique de correction pour le type Q32).
3. `buildSquadFirstEvents` (.17) : correction appliquée + **skip des events `TimeMS<0`** (countdown) — sinon collision avec le sentinel `-1` de firstKillS/firstDeathS (le vrai 1ᵉʳ frag serait perdu).
4. `buildSquadImpactMatrix` (.07) via `loadImpactEventsByMatch(…, timelines)` : invariance des badges **vérifiée** en lisant `ComputeMatchImpactFull` (min/max/ordre + diff kamikaze `dT-kT`, aucun seuil absolu ; `SquadImpactCell` n'expose aucun TimeMS) → no-op observable, corrigé pour cohérence.

Validation : médiane T0 réelle 27.6s (1724/1724 matchs), 0 négatif. 4 tests ajoutés, build + tests verts.

**Périmètre étendu (revue adverse + décision utilisateur 2026-05-29)** : 2 consommateurs d'events bruts supplémentaires découverts par la revue ont AUSSI été corrigés :
- `buildSquadIntensityProfile` (teammates.13, `..._intensity_perminute.go`) : `CorrectImpactEvents` + filtre `TimeMS<0` (calcul de durée ET bucketing). Le garde `duration<=0` existant absorbe l'outlier T0=14400s.
- `squad_service.go` (page Squad V1, `ComputeImpactSummary`) : correction appliquée — no-op observable (`SquadImpact` = compteurs only, gagnants invariants) mais cohérence de pipeline.
Les 4 consommateurs d'events de la page Escouade (.17, .07, .13, squad-V1) appliquent désormais T0.

**RESTE pour clôturer** :
- Validation UI (agent ne peut pas) : teammates.17 (premier frag), teammates.13 (profil intensité), teammates.07 (matrice impact).
- Dead code à supprimer (hors scope, signalé par la revue) : legacy `Q30SquadMatches` (`queries_squad.go`, 0 caller live, layout 22 cols obsolète).
- Garde-rail qui aurait évité l'oubli : linter `no_raw_time_ms` (Phase 5, jamais faite) — bloquer tout accès `.TimeMS` hors couche timeline. Toujours pertinent (aurait attrapé .13, .17 et squad-V1 d'emblée).

### 4.B — Recalcul LUSR (prod-sensitive)

`skill_v2_shadow.go` lit `last_leave_time` pour ordonner les quitters. Ce champ a été corrigé (re-normalisation §2). Si des ratings LUSR v2 ont été calculés sur l'ancien ordre (faux), **les recalculer**. ⚠️ Investiguer le mécanisme de recalcul AVANT d'agir (opération prod). Cf. mémoire `data-quality-first-joined-tz`.

### 4.C — Phase 4 restante : `time_played_seconds`

Recalculer depuis `first_joined_time`/`last_leave_time` (maintenant corrects) :
`time_played = MIN(last_leave, gameplay_end) − MAX(first_joined, gameplay_start)`, borné [0, gameplay_duration]. Garde-rail : médiane des full-match ≈ gameplay_duration. Backfill DB (= checkpoint utilisateur). Cf. `.ai/PLAN_MATCH_TIMELINE_T0.md` §7.

### 4.D — TODO mineur

`ComputeSynthesisKPIs` (legacy) : charger `avg_life_seconds` dans `SynthesisMatchRow` (requête Q33b) pour exposer la vraie AvgLife au lieu de nil.

---

## 5. Points d'entrée code

| Fichier | Rôle |
|---------|------|
| `internal/analysis/timeline/build.go` | `phase1T0Ms()` = **point de bascule Phase 3** |
| `internal/analysis/timeline/compute_t0.go` | `ComputeT0` + `T0Quality` |
| `internal/analysis/timeline/correct_events.go` | `CorrectEvents` |
| `internal/analysis/timeline/golden_test.go` | golden (régénérer via `UPDATE_GOLDEN=1`) |
| `internal/sync/transforms.go` | `computeMatchT0` + `ExtractRegistry` (real_start_time) |
| `internal/domain/match_timeline.go` | `MatchTimeline` |
| `cmd/backfill_t0/`, `cmd/backfill_first_joined_tz/` | backfills (idempotents) |
| `internal/games/canonical/match.go` | `MatchSummary` / `PlayerMatchRow` (à enrichir Phase 3) |

---

## 6. Garde-fous (règles projet)

- **Backup restic AVANT toute écriture DB** : `go run ./cmd/backup-once` (depuis apps/go-api).
- **Checkpoint utilisateur** avant migration DB / backfill `--commit` (actions shared state).
- **Pattern ART** : écriture sur shared via injection avant `Submit` (collect→persist), PAS d'`UPDATE` concurrent sur les tables critiques.
- **Commits** : demander l'autorisation ; jamais `git stash` (commit WIP à la place).
- **thought_log obligatoire** avant chaque commit (`.ai/thought_log.md`).
- DuckDB CLI absent du PATH → requêtes via petit programme Go `//go:build ignore` + driver `github.com/duckdb/duckdb-go/v2` (mode `?access_mode=read_only` pour lecture). Path shared : `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`.

---

## 7. Démarrage suggéré pour l'agent frais

```
git checkout feat/match-timeline-t0
git log --oneline -10        # voir les 10 commits T0
go test ./internal/analysis/timeline/ ./internal/sync/ -count=1   # confirmer vert
```
Puis attaquer §4.A étape 1 (ajouter le champ T0 au canonical). Faire valider le plan Phase 3 par l'utilisateur avant de toucher les repos (changement de contrat canonical).
