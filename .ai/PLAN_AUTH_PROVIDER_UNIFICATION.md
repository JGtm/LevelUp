# Plan — Unifier MSAL TokenProvider et auth.Pool

**Noté** : 2026-05-24
**Statut** : Plan d'analyse (pas d'implémentation)
**Effort** : 4-8h selon option retenue
**Lien** : suite à la découverte 2026-05-23 lors du smoke test Phase 3 Collect→Persist

---

## 1. État actuel — 2 systèmes d'auth coexistent

```
┌──────────────────────────────────────────────────────────────────┐
│                       Sources de credentials                       │
└──────────────────────────────────────────────────────────────────┘

[MSAL Cache + Refresh Token] ────► data/auth/watcher_tokens.json
                                  └► MSAL local cache (Microsoft)

[ENV var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>]
[Player DB sync_meta.refresh_token] ────► data/titles/halo_infinite/players/{GT}/stats.duckdb


┌──────────────────────────────────────────────────────────────────┐
│                      Consommateurs (3 indep)                       │
└──────────────────────────────────────────────────────────────────┘

(1) Watcher daemon ───────► utilise MSAL TokenProvider
    refresh proactif XSTS    (refresh_token + access_token)
    chaque ~5min             écrit dans watcher_tokens.json

(2) SyncEngine (HTTP + CLI) ──► utilise MSAL TokenProvider
    via /api/v1/players/{slug}/sync   (session cookie OAuth)
    via cmd/levelup/cmd_sync.go       (auth_platform.MSALProvider)

(3) AutoSyncScheduler ────────► utilise auth.Pool
    background tick 15min       (Discovery scan sync_meta + env vars)
    via PooledHaloClient        pool.HasPlayer() gate
```

**Source de divergence** : (3) ne consomme PAS le MSAL TokenProvider. Il a son propre pool peuplé une seule fois au boot par `Discovery.Scan()` qui lit `sync_meta` + env vars.

---

## 2. Causes historiques (déduction)

- **Pool** introduit pour gérer le multi-joueur en concurrence : chaque joueur a son token, on les load au boot dans un `sync.Map` pour éviter relectures DuckDB.
- **MSAL provider** introduit plus tard pour le watcher RTA (Xbox Live presence) qui a besoin de XSTS frais en continu, indépendamment du sync Halo.
- Le sync HTTP a été migré vers MSAL (session OAuth web), mais l'auto-sync a gardé le pool legacy.

**Symptôme observable** : au 1er boot post-clone, `pool: scan terminé total_players_scanned=4 players_with_token=0` → tous les joueurs skip. Il faut soit :
- Configurer `.env.local` avec `SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>`
- OU déclencher 1× sync manuel (HTTP/CLI/UI) qui peuple `sync_meta`
- Ensuite reboot → pool détecte les tokens

---

## 3. Impacts du désalignement

| Symptôme | Sévérité | Touch user ? |
|---|---|---|
| Pool vide au 1er boot → 0 sync auto pendant la fenêtre où l'utilisateur attend | 🟡 Moyenne | Oui (UX dégradée premier setup) |
| Logs WARN `pool non initialisé` au boot — fausse alarme si MSAL valide | 🟢 Mineure | Non (ops) |
| Le hint `vérifier SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> dans .env.local` est misleading (MSAL marche sans) | 🟢 Mineure | Oui (debugging) |
| Confusion architecturale : 2 sources de vérité auth | 🟡 Moyenne | Non (devs) |
| **Couplage** : Phase 3 activation Collect→Persist via auto-sync attend `.env.local` ou 1er sync manuel | 🟡 Moyenne | Oui (déploiement) |

**Sévérité globale** : **moyenne** — pas de perte de données, pas de fuite token, pas de bug fonctionnel. Mais UX confuse + dette architecturale.

---

## 4. Options de résolution

### Option A — Pool re-scan périodique (1h effort)

**Idée** : ajouter un timer dans le scheduler qui re-déclenche `Discovery.Scan()` toutes les N minutes. Si nouveaux tokens en `sync_meta` ou env, le pool se peuple en background sans reboot.

| ✅ Avantages | ❌ Inconvénients |
|---|---|
| Petit fix localisé scheduler | Pool reste source séparée |
| Pas de touche au watcher ni au MSAL | Latence : up to N min avant que pool voie le token |
| Idempotent, faible risque | Re-scan inutile la plupart du temps (overhead minimal) |

### Option B — Watcher notifie Pool on refresh (2h effort)

**Idée** : le watcher daemon, après chaque refresh MSAL réussi, appelle `pool.Add(credSource)` pour ajouter/mettre à jour le token dans le pool. Wiring via callback `OnTokenRefreshed`.

| ✅ Avantages | ❌ Inconvénients |
|---|---|
| Pool toujours synchronisé en quasi-réel | Couplage explicite watcher → pool |
| Pas de polling | Requiert que les 2 packages se connaissent (interface ?) |
| MSAL reste source de vérité auth | Cas inverse : tokens en `sync_meta` mais pas en MSAL → pool ne voit pas |

### Option C — SyncEngine écrit dans sync_meta au début du run (1h30 effort)

**Idée** : actuellement `SetSyncMeta(last_delta_sync, now)` est appelé en fin de cycle. Le sync engine pourrait aussi écrire le refresh_token courant en début de cycle (via `WriteOAuthRefreshToken` qui existe déjà). Au prochain boot, Discovery scan le voit.

| ✅ Avantages | ❌ Inconvénients |
|---|---|
| Petit fix dans engine.run() | Toujours un reboot nécessaire pour rafraîchir pool |
| Réutilise infra existante | Couvre uniquement le cas "sync manuel a tourné" |
| Pas de polling ni callback | Pas le cas "watcher rafraîchit hors sync" |

### Option D — Pool lit aussi watcher_tokens.json (1h effort)

**Idée** : `Discovery.Scan()` consulte 3 sources (sync_meta, env, **watcher_tokens.json**). Si MSAL est valide pour un joueur, l'inclure dans le pool.

| ✅ Avantages | ❌ Inconvénients |
|---|---|
| Petit fix localisé Discovery | Pool devient triple-source |
| Aucun couplage watcher/pool/sync | Format watcher_tokens.json ≠ format pool credential — adapter |
| Marche au 1er boot post-watcher refresh | Le sens-de-vérité reste flou |

### Option E — Unifier sur TokenProvider (gros refactor, 6-8h)

**Idée** : auto_sync abandonne `pool.Pool` et passe directement par `auth.TokenProvider` (comme le HTTP handler et la CLI). Le pool devient juste un `[]string` (liste de gamertags avec tokens valides) interrogé via `provider.HasPlayer(gt)`.

| ✅ Avantages | ❌ Inconvénients |
|---|---|
| **Une seule source de vérité auth** | Gros refactor (touche scheduler, PooledHaloClient, etc.) |
| Cohérent : tous les chemins utilisent MSAL | Risque de régression sur multi-joueur concurrent |
| Long-term clean | Effort important |

---

## 5. Recommandation

**Court terme (1 sprint, ~1h30 effort)** : combiner **Option D + Option C** :
1. Discovery.Scan() lit aussi `watcher_tokens.json` (D) → pool peuplé au 1er boot si MSAL valide.
2. SyncEngine.run() écrit `sync_meta.refresh_token` au début du cycle (C) → cohérence sync ↔ pool maintenue.

→ Résout les 3 symptômes user-visible sans gros refactor. Conserve les 2 systèmes mais les synchronise.

**Long terme (1 sprint dédié, 6-8h)** : Option E (unification sur TokenProvider). Approche big-bang :
- Refactor `AutoSyncScheduler` pour utiliser `TokenProvider` au lieu de `pool.Pool`.
- Garder `pool.Pool` comme cache mémoire interne du provider (optimisation, pas API publique).
- Migration progressive via feature flag `LEVELUP_AUTH_UNIFIED=1`.

→ Élimine la dette architecturale. À programmer après Phase 3 Collect→Persist stabilisée.

---

## 6. Hors scope

- Sécurité MSAL : refresh tokens stockés en clair dans `watcher_tokens.json` (déjà existant, audit séparé).
- Token rotation : géré côté Microsoft via OAuth v2 refresh, déjà OK.
- Multi-titres : aujourd'hui un seul titre (Halo Infinite). Quand un 2e titre arrivera, le pool aura besoin d'une dimension `titleSlug` — déjà prévu (`SetCurrentTitle`).
