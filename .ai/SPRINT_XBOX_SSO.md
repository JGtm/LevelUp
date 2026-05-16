# Plan : SSO Xbox / Microsoft pour LevelUp

> **Statut** : Plan d'implémentation futur, indépendant.
> **Branche cible** : à créer (`feat/xbox-sso`), depuis `main`.
> **Auteur du plan** : Claude (session du 2026-05-16).

---

## 0. État actuel (ce qui est déjà fait)

| Composant | Localisation | Statut |
|---|---|---|
| App Azure enregistrée | `e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca` dans [apps/go-api/internal/platform/auth/msal_client.go:29](apps/go-api/internal/platform/auth/msal_client.go#L29) | OK, public client |
| Provider MSAL (Device Code) | [apps/go-api/internal/platform/auth/provider.go:99](apps/go-api/internal/platform/auth/provider.go#L99) | Fonctionnel |
| Provider SISU (alt sans Azure) | [apps/go-api/internal/platform/auth/sisu_provider.go:67](apps/go-api/internal/platform/auth/sisu_provider.go#L67) | Fonctionnel |
| Échange access_token → Halo tokens + XUID + gamertag | `provider.Exchange()` | Fonctionnel |
| Refresh OAuth v2 avec rotation | [apps/go-api/internal/platform/auth/oauth_refresh.go:64](apps/go-api/internal/platform/auth/oauth_refresh.go#L64) | Fonctionnel |
| Endpoints Device Flow | `POST/GET /auth/device-flow/...` dans [apps/go-api/internal/api/handlers/auth.go:64](apps/go-api/internal/api/handlers/auth.go#L64) | Branchés "post-login" |
| UserStore avec gamertag/xuid | [apps/go-api/internal/platform/userstore/store.go:226](apps/go-api/internal/platform/userstore/store.go#L226) `LinkIdentity` | Pas de lookup par XUID |
| Config `AuthMode` | [apps/go-api/internal/config/config.go:42](apps/go-api/internal/config/config.go#L42) : `"none"` \| `"password"` | À étendre |
| Frontend `LoginPage` | [apps/web/src/features/auth/LoginPage.tsx](apps/web/src/features/auth/LoginPage.tsx) | Username/password uniquement |

---

## 1. Décision design — 2 variantes

| | Variante A — Device Code | Variante B — Authorization Code |
|---|---|---|
| **UX** | Bouton → affiche code à 9 caractères + URL `microsoft.com/devicelogin` à recopier | Bouton → redirect Microsoft → retour automatique |
| **Reconfig Azure** | Aucune (public client déjà OK) | Ajouter plateforme "Web" + redirect URI |
| **Effort** | ~1 jour | ~2-3 jours |
| **Quand préférer** | MVP rapide, multi-device (TV, mobile) | Vraie UX SSO desktop |

**Reco** : faire A en premier (réutilise 100% du code existant), puis B en option si l'UX te démange.

---

## 2. PR 1 — Backend : étendre AuthMode + lookup XUID

**Périmètre** : préparer le terrain, mergeable sans rien casser.

- [apps/go-api/internal/config/config.go:42](apps/go-api/internal/config/config.go#L42) : `AuthMode` accepte désormais `"none" | "password" | "xbox"`
- [apps/go-api/internal/domain/auth.go](apps/go-api/internal/domain/auth.go) : ajouter `BootstrapResponse.AuthMode` exposé au frontend (vérifier qu'il y est déjà)
- [apps/go-api/internal/platform/userstore/store.go](apps/go-api/internal/platform/userstore/store.go) — 3 nouvelles méthodes :
  ```go
  func (s *Store) GetByXUID(xuid string) (*domain.User, error)
  func (s *Store) CreateFromXbox(gamertag, xuid string) (*domain.User, error)  // pas de password
  func (s *Store) AuthenticateByXUID(xuid string) (*domain.User, error)         // get + touch lastLogin
  ```
- Le username devient `slugify(gamertag)` à la création — gérer la collision avec un user existant qui aurait pris le slug.
- Tests : `userstore/store_test.go` ajouter 3 cas (create from xbox / get by xuid / collision username)

---

## 3. PR 2 — Backend : endpoint login Xbox (Variante A — Device Code)

**Périmètre** : nouveau flow qui crée la session directement.

- Nouveau handler `apps/go-api/internal/api/handlers/auth_xbox.go` :
  - `POST /auth/xbox/start` : réutilise `provider.InitDeviceFlow` + `attempts.GetOrCreate`
  - `GET /auth/xbox/status/{id}` : variante de `GetDeviceFlowStatus` qui, au lieu d'appeler `LinkIdentity`, fait :
    ```go
    user, err := h.userStore.GetByXUID(snapshot.XUID)
    if errors.Is(err, ErrUserNotFound) {
        user, err = h.userStore.CreateFromXbox(snapshot.Gamertag, snapshot.XUID)
    }
    sess.Username = &user.Username
    sess.LinkedHaloIdentity = &domain.HaloIdentity{...}
    sess.HaloTokens = &domain.HaloTokens{...}
    sess.CurrentPlayerSlug = &user.Gamertag
    ```
- [apps/go-api/internal/api/server.go](apps/go-api/internal/api/server.go) : router le nouveau handler
- Persister le `oauth_refresh_token` rotaté dans `sync_meta` (pour que les sync fonctionnent sans redemander à se reconnecter) — déjà géré par `TryOAuthRefreshWithRotation`, juste à brancher
- Tests : handler de bout en bout avec `stubDeviceFlow` + `stubProvider`

---

## 4. PR 3 — Frontend : page de login Xbox

**Périmètre** : nouvelle UI qui consomme le flow.

- [apps/web/src/features/auth/LoginPage.tsx](apps/web/src/features/auth/LoginPage.tsx) : router conditionnellement
  ```tsx
  if (authMode === 'xbox') return <XboxLoginPage />
  ```
- Nouveau composant `apps/web/src/features/auth/XboxLoginPage.tsx` :
  - Bouton "Se connecter avec Xbox" → appel `POST /auth/xbox/start`
  - Affiche le `user_code` en gros + lien cliquable `verification_uri`
  - Poll `GET /auth/xbox/status/{id}` toutes les `poll_interval_seconds`
  - Sur `authorized` → invalidate `queryKeys.bootstrap` + redirect `/`
- [apps/web/src/features/auth/RegisterPage.tsx](apps/web/src/features/auth/RegisterPage.tsx) : désactiver/cacher quand `authMode === 'xbox'` (les comptes sont créés implicitement)
- [apps/web/src/stores/appShellStore.ts](apps/web/src/stores/appShellStore.ts) : ajouter `'xbox'` au type `AuthMode`
- Tests : composant `XboxLoginPage.test.tsx` avec MSW pour mocker le polling

---

## 5. PR 4 — (optionnel) Upgrade vers Authorization Code Flow

**Périmètre** : vraie UX SSO redirect, si tu décides après A que ça vaut le coup.

- Sur portal.azure.com, sur l'app `LevelUp Halo` : ajouter plateforme "Web" + redirect URIs (`http://localhost:8000/auth/xbox/callback` en dev, prod URL plus tard)
- Nouveau handler `GET /auth/xbox/login` :
  - Génère un `state` aléatoire stocké en session (CSRF)
  - 302 vers `https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize?client_id=...&response_type=code&scope=Xboxlive.signin+Xboxlive.offline_access+offline_access&redirect_uri=...&state=...`
- Nouveau handler `GET /auth/xbox/callback?code=&state=` :
  - Vérifie `state` ↔ session
  - Échange `code` → tokens via `POST /oauth2/v2.0/token` (factoriser depuis `oauth_refresh.go`, c'est le même endpoint avec `grant_type=authorization_code`)
  - Appelle `provider.Exchange(accessToken)` (réutilisé)
  - Trouve/crée le user (logique PR 2)
  - Redirect 302 vers `/`
- Frontend : `XboxLoginButton` devient `<a href="/auth/xbox/login">` simple, plus de polling

---

## 6. PR 5 — (optionnel) Migration users existants

Si tu as déjà des users password avec gamertag liés via `LinkIdentity` :

- Commande `levelup migrate-to-xbox-auth` :
  - Users avec `xuid` rempli → rien à faire, le `GetByXUID` les trouvera
  - Users sans `xuid` (création password sans liaison Xbox) → flag "doit relier Xbox au prochain login" + UI qui force le flow Xbox une fois

---

## 7. Pièges à éviter

1. **CSRF sur Authorization Code** : `state` obligatoire, sinon trivial à phisher
2. **Redirect URI strict** : doit être *exactement* identique côté Azure et côté Go (trailing slash compris)
3. **XUID vs gamertag comme PK** : XUID est stable, gamertag peut changer (~1×/an). Toujours indexer par XUID.
4. **Collision de username** : si quelqu'un s'est déjà créé un compte password avec le username "JGtm" puis tente de SSO avec gamertag "JGtm", il faut soit fusionner soit générer un suffixe (`JGtm_xbox`)
5. **Premier user = admin** : ton code a probablement un check `firstLaunch` qui donne le rôle admin au premier user créé — vérifier que `CreateFromXbox` respecte cette logique
6. **Refresh token rotation** : déjà géré, mais **persister le RT rotaté dans `sync_meta`** sinon la prochaine sync demande de se reconnecter
7. **Demo mode** : `h.demoMode` doit bloquer le flow Xbox comme il bloque le Device Code actuel ([apps/go-api/internal/api/handlers/auth.go:65](apps/go-api/internal/api/handlers/auth.go#L65))

---

## 8. Estimation totale

| PR | Effort | Bloque la suite ? |
|---|---|---|
| PR 1 (config + userstore) | 0.5j | Oui |
| PR 2 (handler xbox) | 0.5j | Oui |
| PR 3 (frontend) | 0.5j | Non (peut paralléliser avec PR 4) |
| PR 4 (Auth Code flow) | 1.5j | Non, optionnel |
| PR 5 (migration) | 0.5j | Non, à faire seulement si users existants |

**Total** : ~1.5j pour MVP fonctionnel (PR 1-3), ~3j pour vraie UX SSO (PR 1-4).

---

## 9. Avant de démarrer

- [ ] Vérifier que la branche courante au moment du démarrage n'est pas `main` ni une branche de travail différente — créer `feat/xbox-sso` depuis `main`.
- [ ] Confirmer dans le portail Azure que l'app `e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca` est toujours active et que tu y as accès.
- [ ] Décider de la stratégie de collision de username (PR 1).
- [ ] Décider si tu vises Variante A seule, ou A puis B (impact sur l'ordre).

---

## 10. Récupération d'une BDD orpheline par XUID (cas central)

**Scénario** : un user se connecte via SSO Xbox, mais une BDD pour son XUID existe déjà sur disque (ex: import depuis backup, ancienne installation, ou tu as run un sync admin avant que l'utilisateur ne s'inscrive). Sans ce mécanisme, on créerait un nouveau dossier vide et l'historique serait perdu ou dupliqué.

### Algorithme à câbler dans `CreateFromXbox` (PR 1)

```
Entrée : (xuid, currentGamertag) du SSO Xbox

1. Chercher tous les gamertags historiques de ce XUID :
     SELECT DISTINCT gamertag FROM shared.xuid_aliases WHERE xuid = ?
   → liste possiblement vide, mono- ou multi-élément (rename historique)

2. Pour chaque gamertag candidat, vérifier si data/players/{gamertag}/stats.duckdb existe :
   - 0 match    → pas de BDD orpheline → CreateFromXbox normal, sync from scratch
   - 1 match    → BDD orpheline trouvée → la rattacher au user
   - 2+ matches → cas rare (rename multiple) → prendre la plus récemment modifiée + alerter

3. Si BDD trouvée mais nom dossier != currentGamertag :
   - Renommer le dossier data/players/{oldGT}/ → data/players/{currentGT}/
   - Mettre à jour les chemins en cache (pool DuckDB)
   - Insérer dans shared.xuid_aliases : (xuid, currentGamertag) si manquant

4. Persister le refresh_token rotaté + cache MSAL dans sync_meta du joueur retrouvé
   → la prochaine sync repart immédiatement sans redemander le login

5. user.Gamertag = currentGamertag, user.XUID = xuid
   sess.CurrentPlayerSlug = currentGamertag
```

### Points de vigilance

- **Pool DuckDB** : ton pool a probablement un cache `gamertag → connection`. Après rename, il faut invalider l'entrée (voir [apps/go-api/internal/platform/duckdb/pool.go](apps/go-api/internal/platform/duckdb/pool.go)).
- **Sync_meta cross-DB** : le refresh_token doit aller dans `sync_meta` de la BDD joueur, pas dans le `users.json`. Ça suit le pattern legacy (`SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>` ou clé `oauth_refresh_token` dans sync_meta).
- **Onboarding UI** : si on trouve une BDD orpheline, **demander confirmation** au user avant d'attacher ("On a trouvé X matchs historiques associés à ton compte, les rattacher ?"). Évite le silent merge si on s'est trompé.
- **Édge case `shared.xuid_aliases` vide** : si le user n'a jamais joué avec personne déjà synchronisé, son XUID n'est pas dans `xuid_aliases`. Fallback : faire un appel léger à l'API Halo (`gamertag → XUID` ou inverse) pour valider l'identité avant de scanner le filesystem.

---

## 11. Hors scope & plans annexes

Pour garder ce sprint focused, sont **explicitement hors scope** :

| Sujet | Statut | Plan dédié |
|---|---|---|
| Migration depuis BDD OpenSpartan | Post-SSO, indépendant | `.ai/SPRINT_OPENSPARTAN_IMPORT.md` |
| Multi-user avec ACL "cercle d'amis" | Bloqué tant que SSO Xbox pas en place (sinon ACL contournable) | `.ai/SPRINT_MULTIUSER_ACL.md` |
| Saisie manuelle de tokens MSAL/refresh par l'utilisateur | **Décision : non implémenté** | — |

### Pourquoi pas de saisie manuelle de tokens

- Le SSO Xbox EST le moyen propre d'obtenir et rafraîchir ces tokens automatiquement.
- Use cases qui pousseraient à l'implémenter (proxies d'entreprise, tenant Azure pro restreint, import depuis un autre outil) sont **anecdotiques pour un usage Halo perso**.
- Coût/bénéfice défavorable : surface d'attaque (token volé pasté dans la mauvaise tab), UX confuse, code à maintenir, zéro user pratiqué.
- **Décision** : on attendra qu'un utilisateur réel le demande pour rouvrir le sujet. Si ça arrive, ajouter un mode admin "Import tokens" caché derrière un flag, jamais en flow d'inscription standard.

---

## 12. Subscription RTA auto-provisionnée après login

**Objectif** : dès qu'un user finit son login SSO Xbox, son XUID est automatiquement souscrit à RTA Xbox Live → quand il termine un match Halo, l'app reçoit un push WebSocket → auto-sync immédiat. Plus besoin de cliquer "Sync".

### Ce qui existe déjà

| Composant | Localisation | Statut |
|---|---|---|
| `RTAClient` WebSocket | [apps/go-api/internal/presence/rta_client.go](apps/go-api/internal/presence/rta_client.go) | OK, supporte multi-XUID sur une seule connexion |
| `AcquireXSTSForRTA(accessToken)` | référencé [cmd/server/main.go:729](apps/go-api/cmd/server/main.go#L729) | OK, échange access_token MS → XSTS Xbox Live audience |
| Persistance XSTS + UserHash | `watcher_tokens.json` via `store.UpdateXSTS()` | OK mais **mono-user** aujourd'hui |
| `RefreshLoop` proactif | `auth.NewRefreshLoop()` | OK, XSTS rafraîchi avant expiration (55min) |
| Trigger auto-sync sur event RTA | watcher daemon | OK |

### Le trou actuel

`provider.Exchange(accessToken)` retourne uniquement `SpartanToken + ClearanceToken` (audience Halo, `prod.xsts.halowaypoint.com`). Le **XSTS Xbox Live** (audience `http://xboxlive.com`, nécessaire pour RTA) est jeté en chemin alors qu'on a le `accessToken` Microsoft sous la main. 1 appel HTTP de plus = tout est en place.

### PR 2.5 — Câbler RTA dans le flow SSO

À insérer entre PR 2 (handler login) et PR 3 (frontend) du plan SSO.

**Changements dans `pollDeviceFlow`** (auth_xbox.go) :
```go
// Après provider.Exchange() qui retourne tokens Halo + identité :
xstsRTA, err := auth.AcquireXSTSForRTA(ctx, accessToken)
if err != nil {
    slog.Warn("rta provisioning failed, user logged in without auto-sync", ...)
    // NON BLOQUANT — l'user peut quand même utiliser l'app
} else {
    watcherStore.UpsertUserXSTS(xuid, gamertag, xstsRTA, rotatedRefreshToken)
    watcher.SubscribeXUID(xuid)  // démarre la souscription RTA immédiate
}
```

**Changements dans le watcher store** :
- Étendre `watcher_tokens.json` mono-user → format multi-user (`map[xuid]TokenSet`)
- Étendre `RefreshLoop` → 1 loop par user (ou 1 loop unique qui itère sur tous les users)

### Stratégie multi-user RTA : décision à trancher

C'est le vrai sujet sensible. Trois stratégies possibles :

| Stratégie | Mécanisme | Avantages | Inconvénients |
|---|---|---|---|
| **A — 1 RTA par user** (simple) | Chaque user authentifie sa propre WebSocket et ne subscribe que son XUID | Toujours fonctionne, indépendant du graphe social Xbox, pas de coordination | N WebSockets + N refresh loops (négligeable pour un groupe d'amis ~10-20 users) |
| **B — 1 RTA "tracker" pour le groupe** | Un user désigné authentifie la WS et subscribe les XUID de tous les amis | 1 seule connexion | Nécessite que le tracker ait les autres dans ses *Xbox friends* (sinon la subscription est rejetée). Si le tracker se déconnecte/supprime son compte, tout le monde perd l'auto-sync. Réélection complexe. |
| **C — Hybride avec auto-élection** | Le premier user à se logger devient tracker, les suivants piggyback s'ils ont une relation Xbox avec le tracker, sinon ouvrent leur propre RTA | Optimise sans tout casser | Complexité élevée, dépendance au social graph Xbox |

### Ton point clé : on ne sait pas qui rejoint qui

Tu as raison — le modèle "1 RTA par groupe" suppose qu'on sait identifier les groupes à l'avance, ce qui n'est pas le cas. On découvre les relations au fur et à mesure que les users s'inscrivent et s'ajoutent en amis (sprint `SPRINT_MULTIUSER_ACL.md`).

**Reco** : **démarrer en stratégie A (1 RTA par user)**. Raisons :
- Pour un groupe d'amis de taille raisonnable (10-30 users max), le coût de N WebSockets est négligeable côté Xbox Live (qui supporte des millions de connexions concurrentes) et côté ton serveur (Go gère 10k+ WS sans broncher).
- Aucune dépendance au graphe social Xbox → moins de cas qui plantent silencieusement.
- **Pas d'effort de coordination** = pas de bugs de coordination.
- Permet de livrer la feature dès le SSO sans dépendre du sprint ACL.

**Optimisation possible plus tard** (stratégie C) : si tu observes un coût RTA réel, ajouter une logique d'auto-élection :
- Quand un user s'ajoute en ami avec un autre via `SPRINT_MULTIUSER_ACL` *et* que les deux sont aussi amis sur Xbox Live, élire le plus ancien comme tracker pour les deux, fermer la WS du plus récent.
- Si le tracker se déconnecte, le piggyback réouvre sa propre WS automatiquement.

Mais à mon avis tu n'iras jamais jusque-là — la stratégie A est suffisante pour ton cas d'usage.

### Pièges à éviter

1. **XSTS expire vite (~1h)** : le `RefreshLoop` doit tourner par user. Ne pas faire un single loop global qui rate certains users.
2. **Échec RTA non bloquant** : si `AcquireXSTSForRTA` échoue (Xbox Live indispo, scope manquant), l'user doit quand même pouvoir se connecter. Logger un warning, pas une erreur.
3. **Scope OAuth** : vérifier que `Xboxlive.signin` suffit pour les deux audiences (Halo + Xbox Live RTA). Sinon ajouter `Xboxlive.offline_access` (déjà présent dans [oauth_refresh.go:30](apps/go-api/internal/platform/auth/oauth_refresh.go#L30)).
4. **Démarrage server** : au boot, recharger toutes les sessions XSTS persistées depuis `watcher_tokens.json` et resubscribe chaque XUID. Sinon les users tracked silencieusement perdus.
5. **Suppression d'utilisateur** : `userStore.Delete(username)` doit cascade → `watcherStore.RemoveUserXSTS(xuid)` + `watcher.UnsubscribeXUID(xuid)`.
6. **Logout** : décider du comportement. Reco = **conserver l'abonnement RTA actif même après logout** (la sync continue en arrière-plan). Pour stopper la collecte, l'user doit explicitement "supprimer son compte".

### Estimation effort

| Tâche | Effort |
|---|---|
| Refactor `watcher_tokens.json` mono→multi-user | 0.5j |
| Câblage `AcquireXSTSForRTA` dans flow SSO | 0.5j |
| Reload abonnements RTA au boot | 0.3j |
| Tests d'intégration (login → subscription RTA effective) | 0.5j |

**Total** : ~2j pour PR 2.5. Ajoute peu au sprint global (~1.5j → ~3.5j MVP).
