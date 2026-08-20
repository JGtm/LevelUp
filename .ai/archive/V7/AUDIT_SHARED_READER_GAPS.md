# Audit — Compléments aux gaps `shared.X` (post-WIP P1)

**Date** : 2026-05-20
**Branche** : `fix/auto-sync-different-configuration`
**Contexte** : complément à `.ai/V7/AUDIT_SHARED_READER_LEAKS.md` (commit `9f48fc08`). Le WIP P1 (refactor `LoadMediaFiles`/`Count`/`FilterOptions` via `runMediaPipeline`) est en cours. Ce doc liste les sites **CRITIQUE non capturés dans la version initiale de l'audit** — identifiés en cherchant `pdb.ReadDB().Query*` + `shared.X` croisé.

Doc séparé pour ne pas marcher sur le WIP P1. À fusionner dans `AUDIT_SHARED_READER_LEAKS.md` quand P1 mergé.

---

## 1. Sites CRITIQUE manquants (crash garanti en prod si exercé)

Ces sites exécutent une query contenant `shared.X` sur `r.pdb.ReadDB()` (= Player conn, sans ATTACH `shared` depuis ADR 0016). Symptôme : `Catalog Error: Table with name "shared.match_registry" does not exist`.

### 1.A — Leaderboard local

| Site | Symbole | Notes |
|---|---|---|
| [`leaderboard_repo.go:30-69`](apps/go-api/internal/platform/duckdb/leaderboard_repo.go#L30-L69) | `GetLocalLeaderboard` | `FROM match_skill_rank msr LEFT JOIN shared.match_registry mr` ← cross-DB. Endpoint `/players/{slug}/leaderboard`. |

**Action** : query mixte player (`match_skill_rank`) + shared (`match_registry`). Pattern à scinder en 2 phases :
1. SELECT match_skill_rank sur ReadDB → liste de `match_id`
2. SELECT match_registry sur SharedReader → map `match_id → (is_ranked, playlist_name, pair_name, start_time*)`
3. Merge en Go pour calcul `effective_type` + filtre + tri.

### 1.B — Match exclusions list

| Site | Symbole | Notes |
|---|---|---|
| [`match_exclusion_repo.go:99-117`](apps/go-api/internal/platform/duckdb/match_exclusion_repo.go#L99-L117) | `ListExcluded` | `FROM player_match_enrichment pme LEFT JOIN shared.match_registry r` ← cross-DB. Endpoint admin. |

Note : `GetMatchRegistryInfo` (l. 51-96) du même fichier est OK — section 2 À NETTOYER (utilise SharedReader.Get + préfixe redondant).

**Action** : 2 phases (pme sur ReadDB → registry via SharedReader → merge Go).

### 1.C — Career — multiples queries shared sur Player conn

Audit existant ligne 66 (section 1.6) liste `career_repo.go` "à vérifier en P5". Voici les sites confirmés CRITIQUE :

| Site | Symbole | Q* | Notes |
|---|---|---|---|
| [`career_repo.go:38`](apps/go-api/internal/platform/duckdb/career_repo.go#L38) | `GetLatestRank` | `Q6CareerLatestRank` | À vérifier — Q6 contient peut-être shared.X (LEFT JOIN match_registry pour rank_id) |
| [`career_repo.go:60`](apps/go-api/internal/platform/duckdb/career_repo.go#L60) | `GetXPHistory` | `Q7CareerXPHistory` | À vérifier |
| [`career_repo.go:82`](apps/go-api/internal/platform/duckdb/career_repo.go#L82) | `GetLUSRHistory` | `Q8LUSRHistory` | À vérifier |
| `career_repo.go` (Q9TopMatches caller) | `GetTopMatches` | `Q9TopMatches` | **Confirmé** : Q9 contient `shared.match_participants` |
| `career_repo.go` (Q9bHighlightPool caller) | `GetHighlightPool` | `Q9bHighlightPool` | **Confirmé** : Q9b contient `shared.X` |

(NB : `Q26CareerTopEncountersTpl` et `Q27CareerRivalsTpl` sont déjà migrés vers SharedReader — career_repo.go:391, 493.)

**Action** : audit ligne par ligne sur ces 5 Q*, scinder les cross-DB.

### 1.D — Explorer

| Site | Symbole | Q* | Notes |
|---|---|---|---|
| [`explorer_repo.go`](apps/go-api/internal/platform/duckdb/explorer_repo.go) | `LoadCommonMatches` | `Q19CommonMatches` | **Confirmé** : Q19 contient `shared.match_participants` ; exécuté sur ReadDB |
| `explorer_repo.go` | `LoadKillerVictimBetween` | `Q19bKillerVictimBetween` | **Confirmé** : idem |

**Action** : ces 2 queries sont sans doute purement shared (pas de table player) → migration simple vers SharedReader.

### 1.E — Match history, sessions, stats

| Site | Q* | Notes |
|---|---|---|
| `match_history_repo.go` (callers de Q5MatchHistory) | `Q5MatchHistory` | **Confirmé** : Q5 contient `shared.X` ; à vérifier sur quelle conn |
| `sessions_repo.go` | `Q22SessionMatches` | **Confirmé** : Q22 contient `shared.X` |
| `stats_repo.go` | `Q23StatsMatches` | **Confirmé** : Q23 contient `shared.X` |

**Action** : audit des callers (ReadDB vs SharedReader) puis migration ciblée.

---

## 2. Gap dans pool.go (commentaire désynchronisé)

[`pool.go`](apps/go-api/internal/platform/duckdb/pool.go) ligne ~242-244 (texte avant le commit 9c.5) :

> "La fonction attachShared est conservée temporairement pour la conn SharedSocial (ligne suivante) — sera retirée quand media_repo aura été migré aussi."

Mais `attachShared` a été supprimée DANS le même commit 9c.5 (cf. `git show 9feb07e1 --stat`). Le commentaire est **mort** → induit en erreur tout dev qui le lit.

**Action** : nettoyer ce commentaire dans le commit qui finalise P1 (puisque c'est le travail "media_repo aura été migré aussi").

---

## 3. Gap dans pool.go (errs swallowed silencieux)

Avant l'instrumentation N3 commit 2026-05-20 :

```go
if cfg.GlobalXuidAliasesDBPath != "" {
    if err := attachGlobalXuidAliases(ctx, playerDB, cfg.GlobalXuidAliasesDBPath); err != nil {
        // Non bloquant — log déjà émis dans attachGlobalXuidAliases.
        _ = err
    }
}
```

Le commentaire dit "log déjà émis dans attachGlobalXuidAliases" — **faux**, `attachGlobalXuidAliases` ne log rien (les errs sont juste returned). L'`_ = err` swallowait totalement.

**Action** : déjà corrigé par mon instrumentation N3 (mêmes lignes : `slog.WarnContext` ajouté sur les 2 call sites). Voir thought_log 2026-05-20.

---

## 4. Symptôme + détection automatique

Avec l'instrumentation N1 (cf. thought_log 2026-05-20), toute query DuckDB qui échoue logge maintenant :

```
[ERROR] duckdb: query failed path=…/stats.duckdb op=OpenReadWrite query_excerpt="SELECT … LEFT JOIN shared.match_registry mr ON …" err="Catalog Error: …"
```

→ Tu peux donc identifier les sites CRITIQUE restants en watchant les logs en runtime sur les pages exercées. Liste à compléter au fur et à mesure des hits observés.

Le test sentinel `TestOpenPlayerDB_NoSharedSchemaOnPoolConns` (commit `9f48fc08`) garantit que toute future query `shared.X` sur conn pool casse en CI — mais il ne détecte que les sites couverts par les tests.

---

## 5. Pipeline P1 — note perfs sur `LoadMediaFilterOptions`

Le WIP P1 lance `runMediaPipeline` **3 fois consécutivement** dans `LoadMediaFilterOptions` (une par option type, avec un whereCfg différent qui exclut son propre filtre). C'est 3× le coût IO des queries d'options (chacune fait Phase A + Phase B + enrich Go).

À mesurer en runtime sur une galerie avec ~200+ médias :
- Avant (WIP en cours) : 3× Phase A (SharedSocial) + 3× Phase B (SharedReader bulk match_registry) + 3× enrich
- Optimisation possible si latence trop élevée : factoriser un seul appel `runMediaPipeline` sans whereCfg → set de rows max → puis filtrer en Go par option type (3× moins d'IO mais 1 seul tour de pipeline).

Pas critique tant que dataset reste modeste. À garder en tête pour P1 finalisation.

---

## 6. Synthèse — phases à ajouter au plan

| Phase | Sites | Effort | Status |
|---|---|---|---|
| P5+ | leaderboard_repo `GetLocalLeaderboard` | léger (2 phases) | À faire |
| P5+ | match_exclusion_repo `ListExcluded` | léger (2 phases) | À faire |
| P5 (existant) | career_repo : Q9TopMatches, Q9bHighlightPool + Q6/Q7/Q8 (à vérifier) | moyen | Audit existant → étendre |
| P5+ | explorer_repo : Q19CommonMatches, Q19bKillerVictimBetween | léger (shared-only) | À faire |
| P5+ | match_history_repo : Q5MatchHistory | à confirmer | Audit |
| P5+ | sessions_repo : Q22SessionMatches | à confirmer | Audit |
| P5+ | stats_repo : Q23StatsMatches | à confirmer | Audit |
| P0+ | Nettoyer commentaire pool.go ligne 242-244 | trivial | À faire dans le commit P1 final |
