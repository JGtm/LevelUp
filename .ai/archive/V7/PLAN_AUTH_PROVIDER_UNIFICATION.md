# Plan — Unifier MSAL TokenProvider et auth.Pool

**Noté** : 2026-05-24 (révisé après clarification user sur la raison du pool)
**Statut** : E.v1 ✅ LIVRÉ (commit 8f39923a + 03322560) — E.v2 + PR 2.5b backlog
**Effort restant** : ~5-6h (E.v2 callback ~2h, PR 2.5b watcher tracker migration ~3-4h)
**Lien** : suite à la découverte 2026-05-23 lors du smoke test Phase 3 Collect→Persist

## Status closure 2026-05-24 (mis à jour 12:55)

| Item | Status |
|---|---|
| E.v1 — Discovery lit watcher stores | ✅ LIVRÉ (commit `8f39923a`) |
| Fix legacy attribué à 1 seul joueur | ✅ LIVRÉ (commit `03322560`) |
| **E.v2** — hot-add pool via periodic re-scan | ✅ LIVRÉ (commit `4508df92`) — Pool.AddOrUpdateSource + goroutine 15min |
| **PR 2.5b phase 1** — mirror tracker writes legacy → multi-user | ✅ LIVRÉ (commit `157d80a8`) — RefreshLoop.WithMultiUserMirror |
| **PR 2.5b phase 2** — read-path switch (daemon lit multi-user) | ⏳ BACKLOG ~2-3h (design product requis) |
| Chiffrement at-rest tokens (DPAPI/Keychain) | ⏳ BACKLOG ~3h (valeur marginale single-user) |

**Détail PR 2.5b** (révisé 2026-05-24) — l'estimation initiale de ~1h était trop optimiste :

Le watcher daemon a déjà 2 mécanismes multi-user OK :
- **PR 2.5b fallback** (lines 953-987 cmd/server/main.go) : si legacy TokenStore vide, scan MultiUserTokenStore pour trouver un user avec XSTS valide → utilisé comme tracker initial
- **PR 2.5c** (lines 1132+) : userClients RTA par user au boot depuis MultiUserTokenStore

Ce qui RESTE pour PR 2.5b "vraie migration" :
1. Décider quelle XUID devient le "tracker initial" si plusieurs valides (today = premier dans la map random Go)
2. Migrer toutes les persist writes `store.Save(tokens)` + `store.UpdateXSTS` du watcher daemon vers `multi.Upsert(xuid, ...)` (tracker's xuid)
3. Gérer la rotation du tracker si le user actuel se déconnecte / token expire / autre user devient principal
4. Mettre à jour `EnsureWatcherAccessToken` + `RefreshLoop` pour accepter une XUID au lieu d'un store opaque
5. Conserver la rétrocompatibilité : legacy `watcher_tokens.json` reste lu en fallback si vide multi-user

Effort = design product (~1h) + refactor code (~2h) + tests (~1h). Recommandation : laisser en backlog tant que single-user reste le cas dominant.

## 0. Cadrage — pourquoi 2 systèmes (clarification user)

Le pool **et** le TokenProvider sont là **par design**, pas par accident historique :

- **Compat utilisateur** : 2 modes d'auth doivent coexister à long terme :
  - **Refresh token OAuth v2** (`SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>` ou stocké en `sync_meta.refresh_token`) — adapté aux bricoleurs/devs qui extraient leur RT manuellement.
  - **MSAL refresh + cache** (`data/auth/watcher_tokens.json`) — flux Microsoft natif, plus simple pour utilisateurs lambdas.
  - Les 2 produisent in fine un **Spartan token** Halo (output identique). L'usage downstream est le même.

- **Pool = cache Spartan tokens** : pas un "gestionnaire de concurrence", mais une **optimisation de vitesse**. Au boot, on échange une fois refresh→Spartan pour chaque joueur, on stocke en mémoire (`sync.Map`). Le sync utilise `PooledHaloClient` qui lit le Spartan en O(1) sans relancer un exchange OAuth/MSAL à chaque appel API. Bénéfice : un cycle multi-joueur passe d'un exchange par requête à un exchange par token-lifetime (~3h).

- **TokenProvider = source de vérité pour ENTRÉE auth** : refresh OAuth v2 ou MSAL. Sait faire le `Exchange(refresh) → Spartan`. Utilisé par :
  - Watcher daemon (refresh proactif des Spartan)
  - Session OAuth web (login utilisateur → exchange initial)
  - CLI `levelup sync-delta` (exchange à la demande)

**Le vrai problème** n'est pas "2 systèmes", c'est **2 chemins parallèles non-synchronisés** : le watcher rafraîchit MSAL en arrière-plan mais le pool ne le voit pas tant qu'il n'y a pas de reboot ou de sync manuel qui ait écrit dans `sync_meta`.

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

- **Pool** = cache mémoire Spartan tokens, partagé par les workers sync (multi-joueur). Au boot, `Discovery.Scan()` parcourt les joueurs configurés et populate. Lecture O(1) ensuite.
- **MSAL provider** ajouté avec le watcher daemon (Xbox Live RTA) pour les utilisateurs lambdas qui n'extraient pas leur RT manuellement.
- Le sync HTTP a été câblé sur MSAL (via session OAuth web → tokens utilisateur). Auto-sync a gardé le pool (compat refresh-token bricoleurs).
- **Aucune sync entre les 2** : pas de hook "watcher refresh → pool update".

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

### Option E — Unifier sur TokenProvider (refactor moyen, 4-6h) ← révisé

**Idée — le pool est CONSERVÉ comme cache, le TokenProvider devient l'entrée unique** :

```
[Refresh Token OAuth v2 / MSAL Cache]
                │
                ▼
         TokenProvider          ← entrée unique, sait gérer les 2 modes
                │ Exchange(refresh) → Spartan
                ▼
         Pool (sync.Map)         ← cache Spartan, partagé workers
                │ Get(gamertag) O(1)
                ▼
    [Watcher | SyncEngine | PooledHaloClient]
```

Concrètement :
- `auth.TokenProvider` reste avec 2 implémentations (`MSALProvider` + `RefreshTokenProvider`) pour gérer les 2 modes utilisateur.
- `Discovery.Scan()` est supprimé : remplacé par `Pool.RefreshFrom(provider)` qui itère sur les joueurs configurés, appelle `provider.GetSpartanToken(gt)` (qui fait MSAL OU refresh OU les deux en cascade selon ce qui est dispo), et populate.
- `Pool.RefreshFrom` est appelée :
  - Au boot du serveur
  - Périodiquement (~5min, aligné sur la fenêtre de validité Spartan ~3h)
  - Sur callback `provider.OnTokenRefreshed(gt)` du watcher (push-based, pas polling)
- AutoSyncScheduler ne change pas : continue à utiliser `Pool.HasPlayer()` + `PooledHaloClient`. Mais le pool est désormais **toujours synchronisé** avec MSAL.

| ✅ Avantages | ❌ Inconvénients |
|---|---|
| **Une seule entrée auth** (TokenProvider) | Refactor moyen (touche `pool/discovery.go`, ajoute callback wiring) |
| Pool conservé : pas de régression perf multi-joueur | Couplage provider → pool (callback `OnTokenRefreshed`) |
| Compat 2 modes refresh + MSAL **préservée** | — |
| Pool toujours synchronisé (plus jamais vide post-boot) | — |
| Suppression du chemin parallèle `Discovery.Scan` (dette) | — |

---

## 5. Recommandation (révisée)

**Court terme (~1h30 effort)** : combiner **Option D + Option C** :
1. Discovery.Scan() lit aussi `watcher_tokens.json` (D) → pool peuplé au 1er boot si MSAL valide.
2. SyncEngine.run() écrit `sync_meta.refresh_token` au début du cycle (C) → cohérence sync ↔ pool maintenue.

→ Résout les 3 symptômes user-visible sans gros refactor. Patch pragmatique pour activer Phase 3 sans friction.

**Long terme (~4-6h effort)** : **Option E** (révisée) — TokenProvider devient l'entrée unique, pool reste cache :
- Suppression de `Discovery.Scan()` (chemin parallèle dette).
- `Pool.RefreshFrom(provider)` appelé au boot + périodiquement + sur callback `OnTokenRefreshed` du watcher.
- TokenProvider conserve ses 2 implémentations (MSAL + RefreshToken) — **compat utilisateur préservée par design**.
- Pool reste cache Spartan tokens — **bénéfice perf multi-joueur préservé**.
- Migration progressive via feature flag `LEVELUP_AUTH_UNIFIED=1` (default OFF), flip après validation 1 cycle.

→ Élimine la dette architecturale **sans toucher au modèle 2-modes user** ni à l'optimisation pool. À programmer **après** Phase 3 Collect→Persist stabilisée.

**Pourquoi ne pas faire E directement et skipper D+C ?**
- E nécessite des touches plus larges (refactor `Discovery.Scan` + ajout callback wiring) → plus risqué à pousser maintenant alors qu'on vient de livrer un autre gros refactor (Collect→Persist Phase 1-2).
- D+C est un patch local et reverse-compatible (l'ancien code marche aussi).
- Si Phase 3 active avec D+C, l'urgence E retombe → on peut prendre le temps de bien le faire.

**Si on était sûr de pouvoir poser E proprement** (1 dev focus, pas de pression), on pourrait skipper D+C. Décision user.

---

## 5.bis Réponses Q&A user (2026-05-24)

### Q : Pourquoi le watcher token n'est pas aligné avec MultiUserTokenStore ?

2 stores coexistent dans `internal/platform/auth/` :

- **TokenStore legacy** (`token_store.go`) — mono-user, `data/auth/watcher_tokens.json`. Utilisé par le watcher daemon **historique**.
- **MultiUserTokenStore** (`multi_user_token_store.go`) — multi-user, dossier `data/auth/watcher_tokens/{xuid}.json`. Utilisé par le **flow SSO Xbox PR 2.5a** uniquement.

Le commentaire dans `multi_user_token_store.go:11` :
> "LEGACY : le watcher daemon historique utilise TokenStore (mono-user). MultiUserTokenStore est utilisé par le flow SSO Xbox PR 2.5a uniquement. **Migration du watcher : différée à PR 2.5b.**"

→ Non-alignement = **dette technique connue**, pas un bug. PR 2.5b prévue mais pas livrée.

### Q : Pourquoi les tokens en clair ?

`grep -rn "Encrypt|cipher|aes" internal/platform/auth/` → **0 résultat**. Aucun chiffrement at-rest.

Seule protection : **permissions 0600** (lisible uniquement par l'user owner) + dossier 0700.

Standard pour outil desktop single-user :
- Pas de threat model multi-tenant (1 machine = 1 utilisateur)
- "Vol de fichier" implique déjà compromission machine → keychain local serait aussi compromis
- Tokens **TTL court** : Spartan ~3h, XSTS ~12h
- Refresh token **révocable côté Microsoft** si fuite détectée
- Standard de fait dans la communauté Halo (SPNKr Python utilise `auth_tokens.json` identique)

**Recommandation backlog** : si l'outil est distribué à des users non-techniciens, ajouter chiffrement at-rest via Windows DPAPI / macOS Keychain / Linux libsecret (wrap natif). Effort ~3h, valeur marginale tant que l'outil reste single-user local.

### Q : Le pool utilise-t-il bien tous les tokens Spartan disponibles ?

**Oui** — confirmé dans `internal/scheduler/auto_sync.go:383-395` :

```go
parallelism := s.poolSizeSafe()   // = pool.Size() = nb tokens dispo
eg.SetLimit(parallelism)          // errgroup borne à N goroutines
for _, p := range players {
    eg.Go(func() error { return s.syncPlayer(egCtx, p) })
}
```

Et `pool/README.md:172` : "engine.go fetches matches in parallel via `errgroup.SetLimit(pool.Size())`".

| Pool size | syncPlayer en parallèle | Sortie cycle 3 joueurs |
|---|---|---|
| 0 | 0 (skip) | 0 sync |
| 1 | 1 (sériel) | ~15 min |
| 3-4 | 3-4 | ~5-8 min |

**Conséquence directe sur l'incident ART** : c'est exactement la **parallélisation N=4** qui a stressé l'index ART de `shared.match_participants`. 4 workers UPSERT sur la même DB → bug DELETE-side ART. Les mitigations legacy (dblease mutex + singleflight) ont réduit la fréquence sans supprimer. Collect→Persist (Phase 1-2) résout par construction (INSERT-only).

**La parallélisation reste OK** : la racine du bug n'est pas la concurrence des syncs (correctement sérialisée via dblease), c'est l'usage UPSERT dans cette concurrence. INSERT-only + parallel pool = pas de problème.

---

## 6. Hors scope

- **Suppression du pool** : NON, il reste — c'est un accélérateur cache Spartan partagé multi-joueur, pas un héritage.
- **Suppression d'un mode d'auth** : NON, refresh token OAuth v2 ET MSAL restent supportés (compat user explicite).
- Sécurité MSAL : refresh tokens stockés en clair dans `watcher_tokens.json` (déjà existant, audit séparé).
- Token rotation : géré côté Microsoft via OAuth v2 refresh, déjà OK.
- Multi-titres : aujourd'hui un seul titre (Halo Infinite). Quand un 2e titre arrivera, le pool aura besoin d'une dimension `titleSlug` — déjà prévu (`SetCurrentTitle`).
