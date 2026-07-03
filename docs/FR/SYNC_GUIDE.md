# Guide de synchronisation — LevelUp

Version anglaise : [../SYNC_GUIDE.md](../SYNC_GUIDE.md)

> Comment LevelUp garde vos matchs Halo à jour. Le backend est en Go (`apps/go-api`) ; le front est React/Vite (`apps/web`). La synchronisation est désormais **automatique** — il n'y a plus de `python scripts/sync.py`.

## Vue d'ensemble

La sync tourne **à l'intérieur du serveur Go**. Deux boucles indépendantes maintiennent les données fraîches, toutes deux adossées au même `SyncEngine` et au même pool de tokens :

- **Watcher de présence** (`internal/watcher`) — piloté par les événements. Un démon suit la présence Xbox/Steam de chaque joueur configuré (WebSocket RTA + pollers REST). Quand un joueur termine un match, une sync delta est mise en file pour ce joueur uniquement. Latence faible, quasi temps réel.
- **Scheduler d'auto-sync** (`internal/scheduler/auto_sync.go`) — périodique. À intervalle fixe, il lance une sync delta pour tous les joueurs de `db_profiles.json`, rattrapant ce que le watcher aurait manqué.

Les commandes CLI manuelles (`levelup sync-delta` / `sync-full` / `backfill`) existent pour le bootstrap, le comblement de trous et les recalculs locaux, mais l'usage courant ne nécessite aucune action manuelle.

## Architecture des données (V6)

Les données de matchs sont centralisées dans des bases **partagées** par titre ; les **enrichissements** par joueur restent dans la base du joueur. L'arborescence est title-agnostic sous `data/titles/{slug}/` (slug par défaut `halo_infinite`).

```
API Halo (client compatible SPNKr, Go)
        |
        v
SyncEngine (internal/sync) + Pool de tokens (internal/platform/auth/pool)
        |
        +-- match nouveau -> data/titles/{slug}/warehouse/shared_matches_v2.duckdb
        |     match_registry         (1 ligne par match unique)
        |     match_participants     (tous les joueurs, MMR inclus)
        |     highlight_events       (événements film)
        |     medals_earned          (médailles)
        |     killer_victim_pairs    (paires de kills)
        |     xuid_aliases           (xuid -> gamertag)
        |
        +-- PvE / Firefight -> data/titles/{slug}/warehouse/shared_pve.duckdb
        |
        +-- enrichissement -> data/titles/{slug}/players/{gamertag}/stats.duckdb
              player_match_enrichment (performance_score, session_id, is_with_friends)
              personal_score_awards   (awards objectifs)
              match_skill_rank        (LUSR / CSR par match)
              sync_meta               (état de sync)
```

Schéma complet et justification : [../ARCHITECTURE_V6.md](../ARCHITECTURE_V6.md).

## Synchronisation automatique

### Watcher de présence

Démarré par le serveur au boot (démon `internal/watcher`). Pour chaque joueur il exécute une FSM de présence (WebSocket RTA, fallback Steam/REST) et, à la fin d'un match, met en file une sync delta coordonnée. L'auth est déléguée au pool de tokens partagé. Aucune configuration au-delà de la déclaration du joueur dans `db_profiles.json` avec un token valide (voir Auth).

### Scheduler d'auto-sync

`AutoSyncScheduler` lit `app_settings.json` au boot et à chaque tick. Clés concernées :

| Clé (`app_settings.json`) | Signification |
|---|---|
| `spnkr_auto_sync_enabled` | Interrupteur maître. Doit valoir `true` pour que le scheduler agisse. |
| `spnkr_auto_sync_interval_hours` | Intervalle en heures (défaut 6 si absent). |
| `spnkr_auto_sync_interval_minutes` | Intervalle en minutes (prioritaire si défini). |

À chaque cycle, pour chaque joueur de `db_profiles.json` :
1. Skip si le joueur n'a pas d'entrée dans le pool de tokens, ou si le watcher a déjà une session active pour lui.
2. Construit un `PooledHaloClient` pinné sur ce joueur.
3. Lance `SyncEngine.RunDelta` (fetches internes parallèles). Des cycles répétés à zéro insertion déclenchent un warning (garde-fou : 14 jours de zéro insertion silencieuse en mai 2026).

Le diagnostic est exposé via l'endpoint admin `/api/v1/_diag/auto-sync/snapshot`.

## Pipeline de sync V2

Le moteur par joueur (`RunDelta`/`RunFull`) est le défaut (V1). Un **orchestrateur de cycle V2** opt-in (`internal/sync/v2`) traite *tous* les joueurs par cycle en 6 phases, supprimant la sérialisation sur le writer partagé et garantissant une dédup cross-player correcte :

1. **Discovery** — parallèle par joueur, lecture seule : charge les IDs connus + pagine l'API.
2. **Dedup** — single : union des IDs de matchs inconnus entre joueurs.
3. **FetchShared** — errgroup borné : `GetMatchStats` par match unique.
4. **FetchPlayer** — parallèle par joueur : awards/scores nécessitant son propre token.
5. **Persist** — writer unique : un méga-batch (shared + player) en une transaction.
6. **PostSync** — parallèle par joueur : heals, films, citations, etc.

Activation : `LEVELUP_SYNC_PIPELINE=v2` (défaut `v1`, rollback instantané). V1 et V2 partagent les Persisters, le schéma et le WAL.

## Delta vs Full

- **Delta** — ne récupère que les matchs plus récents que le watermark de la dernière sync. Rapide, défaut du watcher et du scheduler.
- **Full** — parcourt les N derniers matchs API et insère les manquants (comblement de trous). À utiliser après une longue panne, un import, ou un problème de watermark.

Pour chaque match synchronisé, le moteur récupère toujours le payload complet : stats, médailles, personal scores, performance score, highlight events, skill/MMR par match, et aliases xuid -> gamertag.

## CLI manuelle

Construire/lancer la CLI `levelup` depuis `apps/go-api/cmd/levelup` (nécessite la toolchain CGO pour le driver DuckDB — voir [../testing.md](../testing.md)). `LEVELUP_REPO_ROOT` est auto-détecté si absent.

### Sync delta / full

```bash
# Delta pour un joueur
levelup sync-delta --gamertag VotreGamertag [--max-matches 25] [--match-type matchmaking] [--rps 1]

# Delta pour tous les joueurs configurés (via le pool de tokens)
levelup sync-delta --all [--max-matches 25] [--token-pool-size 0]

# Full (comblement) pour un joueur ou tous
levelup sync-full --gamertag VotreGamertag [--max-matches 150] [--match-type matchmaking] [--rps 1]
levelup sync-full --all [--token-pool-size 0]
```

| Flag | Concerne | Défaut | Notes |
|---|---|---|---|
| `--gamertag` | sync-delta, sync-full | — | Mutuellement exclusif avec `--all`. |
| `--all` | sync-delta, sync-full | — | Tous les joueurs de `db_profiles.json` via le pool. |
| `--max-matches` | sync-delta / sync-full | 25 / 150 | Delta : max de nouveaux matchs insérés. Full : matchs API parcourus. |
| `--match-type` | les deux | `matchmaking` | `all` \| `matchmaking` \| `custom` \| `local`. |
| `--rps` | les deux | 1 | Max de requêtes API par seconde. |
| `--token-pool-size` | `--all` uniquement | 0 | 0 = auto (toutes les sources découvertes), `MaxSize` du pool. |

### Backfill (recalculs locaux & backfills API)

```bash
levelup backfill (--gamertag X | --all) <selecteur...> [--force] [--dry-run]
```

Sélecteurs (un ou plusieurs requis) :

| Sélecteur | API Halo requise | Description |
|---|---|---|
| `--engagement-scores` | Non | Backfill du score d'engagement. |
| `--citations` | Non | Recalcul de `match_citations` depuis mappings + médailles + stats + awards. |
| `--citations-recompute-all` | Non | Recalcul total (force) + vérifications d'invariants V1-V4. |
| `--composite-only` | Non | Citations composites uniquement (additif). |
| `--lusr` | Non | Recalcul LUSR (TrueSkill 2 + poids médailles). `--dry-run` prévisualise par playlist_group. |
| `--perf` | Non | Recalcul du performance score relatif (v5). |
| `--assists-model` | Non | Modèle OLS expected_assists par mode. |
| `--csr` | Oui | CSR par match via `GetMatchSkill` (RankRecap), idempotent. |
| `--shared-csr` | Oui (pas d'API avec `--dry-run`) | CSR de tous les participants des matchs ranked dans `shared.match_csrs`. |
| `--weapons` | Oui | `weapon_kills` depuis le CDN film. |
| `--compare-formulas` | Non | Simule 5 variantes de formule LUSR sur `--last-n` matchs (défaut 20). |

`--force` retraite les données déjà persistées. `--dry-run` n'est valide qu'avec `--shared-csr` ou `--lusr`. Les sélecteurs adossés à l'API rafraîchissent les tokens Halo du joueur via le refresh token OAuth (voir Auth). Le recalcul LUSR utilise le chemin v2 canonical ; le v1 est mort.

Le backfill est aussi exposé en HTTP (`POST /backfill/start`) ; la CLI est la voie locale sans serveur.

## Auth

Les tokens proviennent de la source unique décrite dans [../adr/0023-auth-tokens-single-source.md](../adr/0023-auth-tokens-single-source.md) : `data/auth/watcher_tokens/{xuid}.json` via `MultiUserTokenStore`. Le joueur doit d'abord être déclaré dans `db_profiles.json` (avec `xuid`).

- Onboarding normal : flux SSO Xbox web -> `/auth/xbox/callback` persiste le refresh token.
- Onboarding avancé : `go run ./apps/go-api/cmd/token-capture/ <Gamertag>` (device-code) ou `go run ./apps/go-api/cmd/token-import/ <Gamertag>` (RT sur stdin) écrit directement dans le store — aucune édition de `.env.local`.

Les chemins de sync `--all` et les backfills adossés à l'API résolvent les tokens via le pool (Discovery -> Resolver -> Pool), qui gère le refresh MSAL/OAuth et cache les Spartan tokens (~3h30). Les commandes mono-joueur `levelup sync-delta/sync-full --gamertag` et les backfills `--csr/--weapons` lisent la variable d'env legacy `SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>` comme source transitoire — préférer le token store. Ne jamais re-capturer un token pour corriger un 401 : une sync verte signifie que les tokens sont bons.

## Écritures append-only / ART-safe

Toutes les écritures par match passent par l'architecture Collect -> Persist (un batch INSERT-only par cycle), et les tables d'état critiques sont en append-only. Cela éradique par construction le bug de corruption d'index ART de DuckDB. Ne pas réintroduire d'`UPDATE` concurrent ni d'`INSERT ... ON CONFLICT DO UPDATE` sur les tables shared/état. Références :

- [../adr/0019-collect-persist-architecture.md](../adr/0019-collect-persist-architecture.md)
- [../adr/0026-append-only-art-eradication.md](../adr/0026-append-only-art-eradication.md)

## Runbook ops (verrou DuckDB cross-process)

DuckDB ne partage pas un file-lock OS entre processus distincts. Lancer un outil CLI qui ouvre une base **partagée** (metadata, shared_matches_v2, shared_pve, shared_social) en RW pendant que le serveur tient son handle échouera avec `IO Error: Cannot open file ... utilisé par un autre processus`.

Règle : ne pas lancer `levelup sync-* / backfill` (ni les autres outils CLI sur DBs partagées) contre des bases partagées tant que le serveur (`apps/go-api/server.exe` ou `air`) tourne. Arrêter le serveur d'abord pour toute écriture partagée cross-process. Procédure complète et inventaire des outils : [../RUNBOOK_OPS_DUCKDB_CLI_TOOLS.md](../RUNBOOK_OPS_DUCKDB_CLI_TOOLS.md).

## Dépannage

| Symptôme | Action |
|---|---|
| Auto-sync inactive | Vérifier `spnkr_auto_sync_enabled: true` dans `app_settings.json` ; inspecter `/api/v1/_diag/auto-sync/snapshot`. |
| Joueur skippé (`not_in_pool`) | Aucun token découvert pour ce joueur — onboarder via SSO ou `token-capture`/`token-import`. |
| Zéro insertion répétée | Surveiller le warning du scheduler ; vérifier que l'appel `/matches` utilise `xuid(NNN)` et non le gamertag brut, et que le watermark est sain. |
| 401 sur un backfill API | Tokens périmés en cache ; **ne pas** re-capturer. Laisser le pool rafraîchir. Voir [../adr/0023](../adr/0023-auth-tokens-single-source.md). |
| `Cannot open file ... utilisé par un autre processus` | Verrou DuckDB cross-process — arrêter le serveur avant la CLI (voir Runbook ops). |
