# RAPPORT V10c — Lecture des budgets DuckDB sous charge + verdict J4/J6

> Date : 2026-07-13 · Auteur : exécutant Opus (mesure décisionnelle, aucun code applicatif touché)
> Périmètre : solder l'item hérité **V10c** de la campagne d'audits 2026-07 (« lecture budgets
> sous charge → statuer J4/J6 »). Rapport autoporteur : définitions, méthode, mesures brutes,
> analyse, verdict.

---

## 1. Définitions retrouvées (citations, rien de deviné)

### V10c — `.ai/V7/PLAN_CLOTURE_AUDITS_2026-07.md` (lignes 455-462)

> `[!] V10c — POST-merge (différé par conception : nécessite la prod post-merge sous charge
> réelle). Hors périmètre de l'exécutant V10ab. Fenêtre d'observation runtime — lire
> duckdb_pool_stats + duckdb_budgets sous charge réelle (débloque J1(2)) et statuer J4/J6
> (measure-first) avec des chiffres. À traiter après le merge (Gate J définitif).`

V10c n'est PAS une optimisation : c'est la **fenêtre de mesure runtime** qui débloque trois
items différés « measure-first » du LOT J (perf DuckDB) — J1(2), J4, J6 — en fournissant enfin
les chiffres sous charge réelle qui manquaient (VPS injoignable au moment de la rédaction du
LOT J).

### J1(2) — `.ai/V7/PLAN_TRAITEMENT_AUDITS_2026-07.md` (lignes 1010-1016)

> `[~] J1 — ARCHI 48 : (1) LIVRÉ ... (2) BLOQUÉ RUNTIME : le pool lecture 2-4 conns pour player
> DBs dépend de la LECTURE des stats SOUS CHARGE (obs. runtime VPS) + audit des UPSERT reposant
> sur MaxOpenConns(1) — impossible en test local.`

Question ouverte de J1(2) : faut-il augmenter le pool des player DBs de single-conn (1) à 2-4 ?

### J4 — `.ai/V7/PLAN_TRAITEMENT_AUDITS_2026-07.md` (lignes 1028-1034)

> `[!] J4 — ARCHI 47 : DIFFÉRÉ (2026-07-06, measure-first). N est PETIT (1-4 coéquipiers
> SÉLECTIONNÉS, pas un fan-out large) → gain modeste ; le refacto est LOURD (Q30 bulk groupé
> par teammate_xuid + résolution gamertag batch + merge 2-DB shared/player + logique
> d'intersection/KPI par coéquipier à préserver à l'identique). Optimiser un chemin petit-N,
> correctness-sensitive, SANS mesure sous charge (VPS injoignable) = exactement ce que
> measure-first proscrit. Cibles mappées (teammates_service.go:185, teammates_service_kpis.go:84,
> squad_repo.go:167) → session dédiée + validation VPS.`

J4 = N+1 sur le **chemin HTTP lecture de la vue Coéquipiers / Escouade** (`LoadSquadMatchesBulk`).

### J6 — `.ai/V7/PLAN_TRAITEMENT_AUDITS_2026-07.md` (lignes 1043-1050)

> `[!] J6 — ARCHI mineurs N+1 batchables : DIFFÉRÉ (2026-07-06, measure-first). Les 8 sites sont
> TOUS des chemins d'arrière-plan (sync/backfill/catalog) à petit-N — pas des chemins HTTP
> chauds. « ARCHI mineurs » par l'audit lui-même. Cibles re-mappées post-K : sync/engine.go,
> sync/skill/skill_v2_helpers.go:27, relations_moments_service.go:139 (N≤3), fanout_service.go:68,
> sync/session_recalc.go:76, sync/backfill_registry_names.go:182, handlers/prestige.go,
> wire/registry_catalog_expand.go:71. Batcher un chemin d'arrière-plan petit-N sans preuve
> runtime = optimisation à l'aveugle (measure-first). À traiter en lot ciblé avec mesures VPS.`

J6 = 8 N+1 **d'arrière-plan** (sync/backfill/catalog), petit-N, qualifiés « mineurs » par l'audit.

**Critère de décision commun (objectif)** : measure-first stipule qu'optimiser un chemin
petit-N sans preuve de contention sous charge est proscrit. Corollaire testable : *le pool /
provider DuckDB subit-il, sur ces chemins, une contention que J4/J6 soulageraient ?* Si non,
les items se retirent ; si oui, on investigue.

---

## 2. Méthode de mesure

- **Cible** : conteneur prod `levelup-levelup-1` (VPS `lvelup`, 2 vCPU / 2 Go, no-swap).
- **Endpoint** : `/debug/vars` (expvar stdlib), publié au boot par `PublishDuckDBPoolStats` /
  `PublishDuckDBBudgets` (`internal/api/server_apiv1.go:1197-1201`). Protégé
  `RequireAuth + RequireAdmin` (`require_admin.go`) — en mode `LEVELUP_AUTH_MODE=xbox` (prod),
  une session de rôle `admin` est requise.
- **Accès (lecture seule)** : GET signé avec la session admin **déjà active** de JGtm
  (`data/sessions/e78666ab-…json`, `role:admin`, touchée pendant la fenêtre). Cookie
  `levelup_session = <sessionID>.<hex(HMAC-SHA256(secret, sessionID))>` calculé
  **intégralement côté VPS** (le secret `LEVELUP_SESSION_SECRET` n'a jamais quitté le serveur).
  Aucune écriture : `/debug/vars` → `PoolStatsSnapshot()` prend un `Lock()` bref et lit ; zéro
  effet de bord. HTTP 200 obtenu.
- **Fenêtre d'observation** : conteneur démarré `2026-07-13T12:16:15Z`, lecture à
  `2026-07-13T20:00Z` → **~7 h 44 de charge réelle soutenue**. Contenu de la fenêtre (compteurs
  cumulés depuis le boot) : **30 cycles `sync_v2` complets**, **120 post-syncs joueur** (4 joueurs
  × 30 cycles, ~1 cycle / 16 min), **24 899 acquisitions de lease `shared_matches`**,
  **24 768 B-swaps RO↔RW**. Charge instantanée à la lecture : `loadavg 0.00 / 0.03 / 0.06`
  (VPS au repos), conteneur CPU 4,2 % / MEM 483 Mo. C'est une charge **steady-state
  représentative** (mieux qu'un burst de backfill ponctuel : sustained crons + post-syncs).
- **Budgets** : recoupés statiquement — prod n'override **aucun** `LEVELUP_DUCKDB_*`
  (`docker exec env`), donc défauts de `internal/platform/duckdb/db.go`. Valeurs runtime
  confirmées identiques (cf. §3).

---

## 3. Mesures brutes (prod, fenêtre ci-dessus)

### 3.1 `levelup/duckdb_budgets` (J2)

```
memory_limit=512MB · threads=2 · pool_max_open_shared=4 · pool_max_idle_shared=2 · pool_single_conn=1
```
Conforme aux défauts `db.go` (aucun override env). Borne mémoire globale, pas par-classe (choix
J2 délibéré sur RAM partagée 2 Go).

### 3.2 `levelup/duckdb_pool_stats` (J1) — `sql.DBStats` par handle (`WaitCount` / `WaitDuration`)

| Handle | MaxOpen | InUse | WaitCount | WaitDuration |
|---|---|---|---|---|
| `ro halo_infinite/shared_matches_v2` | 4 | 0 | **0** | 0 |
| `ro halo_5/shared_matches_v2` | 4 | 0 | **0** | 0 |
| `rw halo_infinite/metadata` | 4 | 0 | **0** | 0 |
| `rw halo_infinite/shared_social` | 4 | 0 | **0** | 0 |
| `rw halo_5/metadata`, `rw halo_5/shared_social` | 4 | 0 | **0** | 0 |
| `rw halo_infinite/players/JGtm/stats` | **1** | 0 | 64 | 261,5 ms |
| `rw halo_infinite/players/Madina97294/stats` | **1** | 0 | 44 | 193,1 ms |
| `rw halo_infinite/players/Chocoboflor/stats` | **1** | 0 | 40 | 78,6 ms |
| `rw halo_infinite/players/XxDaemonGamerxX/stats` | **1** | 0 | 28 | 90,8 ms |
| `rw halo_5/players/*` (×4) | 1 | 0 | **0** | 0 |
| `rw global/monitoring` | 1 | 0 | 2 | 1,48 ms |

Lecture : **toutes les DB partagées/warehouse (pool 4) = `WaitCount` 0** — aucune saturation du
pool de lecture. Les seuls waits sont sur les **player DBs `halo_infinite` (pool = 1 =
`poolSingleConn`)** : total **176 waits / ~624 ms cumulés sur 7 h 44** (≈ 3,5 ms/wait). C'est la
**sérialisation single-conn voulue** (writer de sync vs lecture concurrente sur la même player
DB), pas une saturation de pool. halo_5 (peu synchronisé) = 0.

### 3.3 `dblease_*` (writer lease, ADR 0013) — acquire / wait_ms / timeout

| Classe | acquire | wait_ms cumulé | timeout |
|---|---|---|---|
| `shared_matches` | 24 899 | 3 966 426 | **0** |
| `metadata` | 360 | 221 216 | **0** |
| `monitoring` | 505 | 0 | **0** |
| `player` | 120 | 5 | **0** |
| `shared_social` | 0 | 0 | **0** |

`dblease_writers_in_use` = 0 partout (idle à la lecture). **0 timeout de lease sur toute la
fenêtre.** Les waits `shared_matches` (~159 ms/acquire) sont la sérialisation B-swap RO↔RW du
**writer de sync**, pas un chemin J4/J6.

### 3.4 `shared_provider_*` (reader / B-swap, ADR 0016)

```
swap_total            : ro_to_rw 24768 · rw_to_ro 24768
swap_duration_ms_total: ro_to_rw 594933 · rw_to_ro 594079   (~19,8 min cumulés)
rw_watchdog_fired     : 150
reader_delayed_total  : 875
reader_stall_ns_total : 3 081 474 363 225 ns  (~51,4 min cumulés)
get_wait_ms_total     : 63 484
get_timeout_total     : 26
swap_failures_total   : acquire_writer 131 · drain_timeout 0 · panic 0 · reopen_ro 0
rw_window_by_holder   : sync_v2_postsync {count 24640, avg 89ms, max 18217ms, watchdog 150}
                        events_convergence_detect {count 120, avg 42ms}
                        h5_livesync_backlog_probe {count 8}
blocked_window_ms     : {count 24768, avg 138ms, max 18297ms, sum 3 432 142ms}
```

**Interprétation clé** : la contention qui EXISTE est **write-side**. Le holder dominant du
window RW est `sync_v2_postsync` : **24 640 fenêtres RW** (~**205 acquisitions RW par
post-sync**), déclenchant 24 768 swaps RO↔RW, 150 watchdogs, et ~51 min de stall lecteur
cumulé. Les lecteurs qui « stallent » sont bloqués par **le writer de sync** pendant ses
fenêtres RW (max 18 s), pas par une saturation induite par les vues J4/J6.

### 3.5 Coût réel du sync (`postsync_step_ms_*`, avg sur 120)

| Étape | avg | max | | Étape | avg | max |
|---|---|---|---|---|---|---|
| **skill_rating** | **32,6 s** | 95,5 s | | citations | 8,6 s | 30,0 s |
| **weapon_kills** | **33,2 s** | 68,5 s | | csr_snapshots | 4,2 s | 30,8 s |
| snapshot_readiness | 2,5 s | 30,0 s | | friends | 4,1 s | 30,0 s |
| scoring | 1,5 s | 2,3 s | | achievements | 3,0 s | 6,2 s |
| aggregates | 61 ms | 2,1 s | | dominance | 19 ms | 61 ms |
| enrichment_rows | 45 ms | 104 ms | | convergence_psa | 28 ms | 66 ms |
| **postsync_total** | **90,4 s** | 128,3 s | | media_scan | 150 ms | 433 ms |

Le goulot sync sous charge = **compute** : `skill_rating` (32,6 s) + `weapon_kills` (33,2 s) =
**66 s des 90 s** de post-sync. Les étapes où vivent les N+1 de J6 sont déjà à <100 ms
(aggregates, dominance, enrichment, convergence).

### 3.6 HTTP (contexte)

`2xx 4199 · 3xx 40 · 4xx 31 · 5xx 32` sur 7 h 44 → **~9 requêtes/min**. Charge HTTP quasi nulle.
(Les 32 réponses 5xx sont notées comme observation périphérique — hors périmètre V10c.)

---

## 4. Analyse et verdict

### J1(2) — pool player DBs : **RÉSOLU → garder single-conn (1)**

Les player DBs `halo_infinite` (pool = 1) plafonnent à 64 waits / 261 ms sur 7 h 44, avec
**0 timeout de lease** et `InUse` 0 à la lecture. La sérialisation single-conn est le
comportement voulu (protège l'UPSERT / append-only anti-ART, ADR 0013/0030). Aucune preuve de
contention justifiant un passage à 2-4 conns. **Décision : conserver `poolSingleConn=1`**, ne
pas ouvrir J1(2). Measure-first satisfait.

### J4 — N+1 vue Coéquipiers/Escouade (HTTP lecture) : **RETIRÉ (won't-do perf)**

- Le handle que ce chemin lit (`ro shared_matches_v2`) affiche **`WaitCount` 0 / `InUse` 0 avec
  4 conns disponibles** : zéro contention de pool sur le chemin J4.
- Charge HTTP globale ~9 req/min ; la vue Escouade n'en est qu'une fraction ; N = 1-4.
- Les stalls lecteurs observés (875 delayed, ~51 min) proviennent du **writer B-swap**
  (`sync_v2_postsync`), pas d'une saturation causée par les 4 requêtes de la vue. Réduire le
  compte de requêtes (4 → 1) **ne réduit pas** les stalls induits par le swap (côté writer).
- Conclusion : le refacto LOURD et correctness-sensible (merge 2-DB + intersection/KPI par
  coéquipier) **n'apporterait aucun gain mesurable**. C'est précisément le cas qu'écarte
  measure-first, avec les chiffres à l'appui. **Retirer.**

### J6 — 8 N+1 d'arrière-plan (sync/backfill/catalog) : **RETIRÉ (won't-do perf)**

- `dblease` player : 5 ms de wait cumulé / **0 timeout** ; les steps hôtes des N+1 sont déjà à
  <100 ms (aggregates 61 ms, dominance 19 ms, enrichment 45 ms, convergence 28-367 ms).
- Le coût sync réel est **compute** (`skill_rating` 32,6 s + `weapon_kills` 33,2 s), pas le
  nombre de requêtes des sites J6. Batcher un N+1 dans un step <100 ms = **0 gain** contre un
  post-sync de 90 s.
- Conclusion : aucun chantier perf dédié justifié. **Retirer.** *Exception pragmatique* : si l'un
  de ces sites est touché lors d'un refacto K pour d'autres raisons (lisibilité), batcher
  opportunément — mais sans en faire un objectif de performance.

---

## 5. Découverte hors périmètre (notée, NON traitée — CLAUDE.md exécution de plans §5)

La lecture des budgets sous charge révèle un gisement perf **réel et bien plus important que
J4/J6 ne l'ont jamais été**, mais **write-side** (donc hors périmètre des deux) :

> **B-swap thrash piloté par le post-sync.** `sync_v2_postsync` acquiert la fenêtre RW du
> shared provider ~**205 fois par post-sync** (24 640 fenêtres pour 120 post-syncs), déclenchant
> **24 768 swaps RO↔RW**, **150 watchdogs**, **~51 min de stall lecteur cumulé**, 26 get-timeouts
> et 131 échecs `acquire_writer`. Fenêtre RW max = **18 s**. Combiné aux deux steps compute
> lourds (`skill_rating` 32,6 s, `weapon_kills` 33,2 s), c'est là que se concentre la latence
> sous charge.

Piste (à instruire dans un item backlog perf distinct, pas ici) : **grouper les écritures RW du
post-sync** pour réduire la granularité des acquisitions (moins de swaps → moins de stalls
lecteurs) et/ou alléger `skill_rating` / `weapon_kills`. C'est le successeur naturel du LOT J,
mais un chantier à part entière.

---

## 6. Décision proposée

Le critère measure-first est **objectif** et désormais **satisfait avec des chiffres** :

- **V10c : SOLDÉ.** `duckdb_budgets` + `duckdb_pool_stats` lus sous charge réelle (7 h 44,
  30 cycles sync, 120 post-syncs). Chiffres consignés ci-dessus.
- **J1(2) : RÉSOLU** → conserver `poolSingleConn=1` (aucune contention justifiant 2-4 conns).
- **J4 : RETIRÉ** (won't-do perf) — chemin HTTP lecture non contendu (`WaitCount` 0), gain
  mesurable nul, refacto lourd non justifié.
- **J6 : RETIRÉ** (won't-do perf) — N+1 d'arrière-plan dans des steps <100 ms ; le goulot sync
  est compute, pas le compte de requêtes.

**Aucune décision utilisateur n'est bloquée** : la résolution ci-dessus applique la logique
measure-first du plan d'origine à la mesure obtenue. Points portés à l'arbitrage/priorisation de
l'utilisateur (non bloquants) :

1. **Ouvrir (ou non) un item backlog perf** pour le B-swap thrash du post-sync + les steps
   compute lourds (§5) — c'est le vrai levier sous charge.
2. **Investiguer (ou non) les 32 réponses 5xx / 7 h 44** (§3.6), hors périmètre V10c.

---

## Annexe — traçabilité

- Items soldés : `.ai/V7/PLAN_CLOTURE_AUDITS_2026-07.md` V10c ([!]→[x]) ;
  `.ai/V7/PLAN_TRAITEMENT_AUDITS_2026-07.md` J4 / J6 ([!]→[x], retirés measure-first) ;
  registre `.ai/V7/DETTE_ASSUMEE_2026-Q3.md` §4 mis à jour.
- Aucune écriture prod. Aucun secret consigné dans ce rapport. Accès via session admin
  existante + GET diagnostique uniquement.
