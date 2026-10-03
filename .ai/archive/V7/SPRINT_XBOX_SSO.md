# Plan : SSO Xbox / Microsoft pour LevelUp

> **Statut** : Décisions tranchées le 2026-05-18 (cf. §0bis). Prêt à implémenter.
> **Branche cible** : à créer (`feat/xbox-sso`), depuis la branche courante.
> **Auteur du plan** : Claude (session du 2026-05-16, décisions du 2026-05-18).

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

## 0bis. Décisions tranchées (2026-05-18)

### Authentification

| # | Décision | Détail |
|---|----------|--------|
| D1 | **Admin initial via UI "premier lancement"** | Si `users.json` est vide au boot → `BootstrapResponse` renvoie un nouveau state (ex. `setup_required="admin"`) qui force le frontend sur `/setup/admin`. Form : `username`+`password` (validation : 3-30 chars, mdp ≥ 8 chars, bcrypt). Endpoint `POST /setup/admin` : 201 si vide, 409 sinon. Une fois créé, le flow SSO Xbox devient disponible. |
| D2 | **Promotion admin : UI Settings + CLI fallback** | Admin password se logue → Settings → Users → bouton "Promouvoir" (ou "Rétrograder"). En backup : `levelup promote-admin --xuid=X` et `levelup demote-admin --xuid=X`. |
| D3 | **Cohabitation : password = admin only** | `AuthMode="xbox"` active le SSO comme flow principal. Le password reste autorisé MAIS uniquement pour les comptes `Role=admin`. Handler login password doit vérifier : `if !user.IsAdmin && cfg.AuthMode == "xbox" → 403`. Conséquence : aucun user non-admin ne peut être créé sans Xbox. |
| D4 | **Refresh token : 1 fichier par XUID** | `data/auth/watcher_tokens/{xuid}.json`, write-to-temp + `os.Rename` atomique (pattern `userstore.Store`), permissions 0600/0700. **Source unique** — pas de duplication dans `sync_meta` DuckDB. Au boot : scan du dossier → reconstruction map RAM par le `TokenStore`. Avantages : zéro contention sur les writes parallèles (sync multi-joueur), révocation = `rm`, audit = `ls`. |

### Audit du `TokenProvider` (couche d'abstraction tokens)

| Provider | Verdict | Action requise |
|----------|---------|----------------|
| `MSALProvider` ([provider.go:99](apps/go-api/internal/platform/auth/provider.go#L99)) | ✅ Stateless, thread-safe, réutilisable tel quel pour le SSO Xbox. **C'est le provider par défaut en prod** ([main.go:62-73](apps/go-api/cmd/server/main.go#L62)). | Aucune. Pour le SSO Xbox, on injecte une `LinkStrategy` au-dessus du provider (§3), pas dedans. |
| `SISUProvider` ([sisu_provider.go:67](apps/go-api/internal/platform/auth/sisu_provider.go#L67)) | ⚠️ **Bug concurrence multi-user** : `current *sisuFlowContext` partagé entre flows, écrasé à chaque `InitDeviceFlow`. Si 2 users démarrent en parallèle, le 2e écrase le 1er → `Exchange` corrompu. | Non bloquant pour le SSO MSAL (default). Fix recommandé : porter `sisuFlowContext` dans le `sisuDeviceFlow` retourné, plus dans le provider. À faire si SISU passe en mode multi-user un jour. |

**Le `TokenProvider` reste l'unique point d'entrée pour acquérir des tokens.** Aucune feature ne doit appeler `ExchangeRefreshToken`, `AcquireTokenSilent` ou `ExchangeAccessToken` directement — tout passe par le provider. Exception : `AcquireXSTSForRTA(accessToken)` reste une fonction package-level appelée juste après `provider.Exchange()` dans le handler SSO (PR 2.5).

### ACL (récap, détail dans `SPRINT_MULTIUSER_ACL.md`)

| # | Décision | Détail |
|---|----------|--------|
| D5 | **Périmètre ACL strict** | Uniquement routes player-scoped : `/api/players/{slug}/*`, `/api/sync/{slug}/*`, `/api/backfill/{slug}/*`, `/api/media/{slug}/*` + filtrage `available_players` du bootstrap. **Match-level et `shared/*` restent publics-authentifiés** ("philosophie Halowaypoint"). |
| D6 | **Friendship indexée par XUID** | La struct `Friendship` utilise `XUIDA`/`XUIDB` (slugs uniquement pour affichage). Résiste aux renames de gamertag Xbox. |
| D7 | **Modèle symétrique avec accept** | Bob demande → Alice accepte → amitié active. Admin peut bypass et créer une amitié forcée (loggé en audit). |
| D8 | **Suppression** | Amitiés hard-deleted, user soft-deleted (`Role=deleted`) pour préserver l'historique cross-référencé dans les matchs. |
| D9 | **Bypass admin avec audit** | `Role=admin` contourne l'ACL sur toutes les routes player-scoped. Chaque accès cross-user loggé via `slog.WarnContext(ctx, "admin_cross_user_access", "audit", true, "admin", X, "target", Y)`. |

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

## 2. PR 0 — UI premier lancement — ✅ Déjà implémenté (révision 2026-05-18)

**Statut** : **Caduque, équivalent en place**. Audit de session révèle que la plomberie existe déjà sous une URL différente :

| Composant initialement prévu (D1) | État réel |
|---|---|
| `userstore.IsEmpty()` | ✅ Existe ([store.go:109](apps/go-api/internal/platform/userstore/store.go#L109)) |
| Premier user = admin auto | ✅ Câblé dans [user_auth.go:107-111](apps/go-api/internal/api/handlers/user_auth.go#L107) (handler `POST /auth/register` : si `IsEmpty` → `role = RoleAdmin`) |
| `BootstrapResponse.FirstLaunch` | ✅ Exposé ([bootstrap.go:85](apps/go-api/internal/domain/bootstrap.go#L85)), rempli par `bootstrap_service.isFirstLaunch()` |
| Redirect "premier lancement → form" | ✅ Câblé dans [__root.tsx:54](apps/web/src/routes/__root.tsx#L54) (`if data.first_launch → navigate('/register')`) |
| Page form premier admin | ✅ Route `/register` existe |

**Décision (2026-05-18)** : on **skip PR 0**. L'endpoint `/auth/register` actuel suffit comme bootstrap admin initial. PR 1 étend la cohabitation D3 pour interdire le register hors-bootstrap en mode `AuthMode="xbox"` (cf. §3.bis).

---

## 3. PR 1 — Backend : étendre AuthMode + lookup XUID — D3

**Périmètre** : préparer le terrain, mergeable sans rien casser.

- [apps/go-api/internal/config/config.go:42](apps/go-api/internal/config/config.go#L42) : `AuthMode` accepte désormais `"none" | "password" | "xbox"`.
- [apps/go-api/internal/domain/bootstrap.go:81](apps/go-api/internal/domain/bootstrap.go#L81) : `AuthMode` est **déjà exposé** dans `BootstrapResponse` (vérifié) — aucune action.
- **Cohabitation password ↔ xbox (D3)** : dans le handler login password (`POST /auth/login`), ajouter en début :
  ```go
  if cfg.AuthMode == "xbox" && user.Role != domain.RoleAdmin {
      writeError(w, http.StatusForbidden, "password_login_admin_only",
          "mode SSO Xbox actif : login password réservé aux admins")
      return
  }
  ```
- [apps/go-api/internal/platform/userstore/store.go](apps/go-api/internal/platform/userstore/store.go) — 4 nouvelles méthodes :
  ```go
  func (s *Store) GetByXUID(xuid string) (*domain.User, error)
  func (s *Store) CreateFromXbox(gamertag, xuid string) (*domain.User, error)  // pas de password
  func (s *Store) AuthenticateByXUID(xuid string) (*domain.User, error)         // get + touch lastLogin
  func (s *Store) CreateAdmin(username, password string) (*domain.User, error)  // utilisé par PR 0
  ```
- Le username devient `slugify(gamertag)` à la création — gérer la collision avec un user existant qui aurait pris le slug (suffixe `_xbox` ou erreur explicite).
- Tests : `userstore/store_test.go` ajouter 4 cas (create from xbox / get by xuid / collision username / create admin idempotent).

---

## 4. PR 2 — Backend : endpoint login Xbox (Variante A — Device Code)

**Périmètre** : nouveau flow qui crée la session directement.

**Architecture — `LinkStrategy` (pattern injecté au-dessus du provider)** :

Pour éviter de forker `pollDeviceFlow` du handler `AuthHandler` existant (qui appelle `LinkIdentity` pour le flow "post-login password"), on extrait la logique "que faire quand l'auth réussit" en interface :

```go
// internal/platform/auth/link_strategy.go
type LinkStrategy interface {
    OnAuthSuccess(ctx context.Context, snapshot *Attempt, sess *domain.SessionData) error
}

// internal/service/xbox_auth_service.go (D5 — orchestration en service, pas handler)
type XboxSSOLinkStrategy struct {
    userStore   *userstore.Store
    pool        *duckdb.Pool       // pour Invalidate après rename
    orphanFinder *playerfs.OrphanFinder
    tokenStore  *auth.TokenStore   // pour persister refresh_token rotaté
}
func (s *XboxSSOLinkStrategy) OnAuthSuccess(ctx, snapshot, sess) error {
    user, err := s.userStore.GetByXUID(snapshot.XUID)
    if errors.Is(err, userstore.ErrUserNotFound) {
        user, err = s.createOrRecoverFromXbox(ctx, snapshot.Gamertag, snapshot.XUID)
    }
    // ... wire session + tokens + refresh_token rotation
}

// internal/platform/auth/strategy_password.go
type PasswordLinkStrategy struct{ userStore UserLinker }
func (s *PasswordLinkStrategy) OnAuthSuccess(...) error { return s.userStore.LinkIdentity(...) }
```

Le handler reste générique : `pollDeviceFlow` appelle `h.linkStrategy.OnAuthSuccess(...)` au lieu d'avoir une branche `if AuthMode == "xbox"`. Injection au boot dans `main.go` selon `cfg.AuthMode`.

**Endpoints** :
- `POST /auth/xbox/start` : réutilise `provider.InitDeviceFlow` + `attempts.GetOrCreate` (single-flight par session).
- `GET /auth/xbox/status/{id}` : poll de l'attempt. Quand `Status==Authorized` → invoque `XboxSSOLinkStrategy.OnAuthSuccess(...)`.

**Persistance refresh_token (D4)** : le RT rotaté retourné par `TryOAuthRefreshWithRotation` est persisté par le `TokenStore` dans `data/auth/watcher_tokens/{xuid}.json` (write-to-temp + rename atomique, 0600/0700). **Pas de duplication dans `sync_meta`**.

**Tests** :
- Handler avec `stubDeviceFlow` + `stubProvider` + `mockLinkStrategy` (capture les calls).
- `XboxSSOLinkStrategy` : 3 cas — user inconnu (création), user existant (lookup), BDD orpheline détectée (cf. §11).

---

## 5. PR 3 — Frontend : page de login Xbox

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
  - **Disclaimer anti-phishing** (sous le `user_code`) : *"Ne saisis ce code que si tu viens de cliquer 'Se connecter avec Xbox'. Quelqu'un qui partage son écran ne devrait jamais te demander ce code."*
- [apps/web/src/features/auth/RegisterPage.tsx](apps/web/src/features/auth/RegisterPage.tsx) : désactiver/cacher quand `authMode === 'xbox'` (les comptes sont créés implicitement)
- [apps/web/src/features/auth/LoginPage.tsx](apps/web/src/features/auth/LoginPage.tsx) : afficher un toggle "Connexion admin (mot de passe)" quand `authMode === 'xbox'` — masqué par défaut. Le form password reste affiché, mais le 403 `password_login_admin_only` est géré gracieusement avec un message clair (D3).
- [apps/web/src/stores/appShellStore.ts](apps/web/src/stores/appShellStore.ts) : ajouter `'xbox'` au type `AuthMode`
- i18n (FR + EN) via les manifestes TOML (cf. ADR 0003) pour tous les nouveaux strings (`xbox.login.button`, `xbox.login.disclaimer`, `xbox.login.error.expired`, etc.).
- Tests : composant `XboxLoginPage.test.tsx` avec MSW pour mocker le polling ; cas spécifique : login password en mode xbox avec user non-admin retourne 403 → message UI.

---

## 6. PR 4 — (optionnel) Upgrade vers Authorization Code Flow

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

## 7. PR 5 — (optionnel) Migration users existants

Si tu as déjà des users password avec gamertag liés via `LinkIdentity` :

- Commande `levelup migrate-to-xbox-auth` :
  - Users avec `xuid` rempli → rien à faire, le `GetByXUID` les trouvera
  - Users sans `xuid` (création password sans liaison Xbox) → flag "doit relier Xbox au prochain login" + UI qui force le flow Xbox une fois

---

## 8. Pièges à éviter

1. **CSRF sur Authorization Code** : `state` obligatoire, sinon trivial à phisher.
2. **Redirect URI strict** : doit être *exactement* identique côté Azure et côté Go (trailing slash compris).
3. **XUID vs gamertag comme PK** : XUID est stable, gamertag peut changer (~1×/an). Toujours indexer par XUID (D6).
4. **Collision de username** : si quelqu'un s'est déjà créé un compte password avec le username "JGtm" puis tente de SSO avec gamertag "JGtm", générer un suffixe (`JGtm_xbox`) — décision retenue.
5. **Race "premier admin"** : D1 garantit qu'un admin password est créé en premier via UI premier lancement. Donc le check `firstLaunch` historique (premier user = admin) doit être **supprimé** de `CreateFromXbox` ; les users Xbox sont toujours créés `Role=user` sauf promotion explicite (D2).
6. **Refresh token rotation (D4)** : déjà géré par `TryOAuthRefreshWithRotation`. Le RT rotaté est persisté par le `TokenStore` dans `data/auth/watcher_tokens/{xuid}.json` — **source unique**, pas de duplication dans `sync_meta`. Au boot : scan du dossier → reconstruction map RAM.
7. **Demo mode** : `h.demoMode` doit bloquer le flow Xbox comme il bloque le Device Code actuel ([apps/go-api/internal/api/handlers/auth.go:65](apps/go-api/internal/api/handlers/auth.go#L65)).
8. **`SISUProvider` non thread-safe multi-user** : `current *sisuFlowContext` partagé entre flows (§0bis audit). Non bloquant pour le SSO MSAL (default), à fix si un user active `AuthProvider="sisu"` ET espère du multi-user concurrent. Fix : porter le contexte dans le `sisuDeviceFlow` retourné.
9. **Phishing visuel Device Code** : un attaquant peut afficher un `user_code` sur son site, la victime tape sur microsoft.com/devicelogin et donne accès. Mitigation : disclaimer UI dans la page Xbox (cf. §5 PR 3).
10. **Pool DuckDB après rename de dossier** (cf. §11) : si la stratégie de récupération renomme `data/games/{title}/players/{oldGT}/` → `{currentGT}/`, invalider l'entrée du pool via `pool.Invalidate(oldSlug)`. Ajouter cette méthode si absente.

---

## 9. Estimation totale

| PR | Effort | Bloque la suite ? |
|---|---|---|
| PR 0 (UI premier lancement admin) | 0.5j | Oui |
| PR 1 (config + userstore + cohabitation D3) | 0.5j | Oui |
| PR 2 (handler xbox + LinkStrategy) | 1j (au lieu de 0.5j — refactor handler générique) | Oui |
| PR 2.5 (RTA auto-provision, cf. §13) | 2j | Non (peut suivre) |
| PR 3 (frontend xbox + admin fallback UI) | 0.5j | Non (peut paralléliser avec PR 4) |
| PR 4 (Auth Code flow) | 1.5j | Non, optionnel |
| PR 5 (migration) | 0.5j | Non, à faire seulement si users existants |

**Total** : ~2.5j pour MVP fonctionnel (PR 0-3), ~4.5j avec RTA auto-provision (PR 0-3 + 2.5), ~6j avec vraie UX SSO Auth Code (PR 0-4).

---

## 10. Avant de démarrer

- [ ] Vérifier que la branche courante au moment du démarrage n'est pas `main` ni une branche de travail différente — créer `feat/xbox-sso` depuis `main`.
- [ ] Confirmer dans le portail Azure que l'app `e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca` est toujours active et que tu y as accès.
- [x] ~~Décider de la stratégie de collision de username~~ → suffixe `_xbox` (cf. §8 piège 4).
- [x] ~~Décider si tu vises Variante A seule, ou A puis B~~ → Variante A en premier, B en option post-MVP.
- [x] ~~Trancher cohabitation password ↔ xbox~~ → password réservé aux admins en mode xbox (D3).
- [x] ~~Trancher store canonique refresh_token~~ → `data/auth/watcher_tokens/{xuid}.json` (D4).
- [x] ~~Trancher promotion admin~~ → UI Settings + CLI fallback (D2).
- [x] ~~Trancher création admin initial~~ → UI premier lancement, PR 0 (D1).
- [ ] Vérifier que `pool.Invalidate(slug)` existe ou l'ajouter avant PR 2 (cf. §8 piège 10).
- [ ] Auditer que le check `firstLaunch=admin` historique sera bien supprimé de `CreateFromXbox` (cf. §8 piège 5).

---

## 11. Récupération d'une BDD orpheline par XUID (cas central)

**Scénario** : un user se connecte via SSO Xbox, mais une BDD pour son XUID existe déjà sur disque (ex: import depuis backup, ancienne installation, ou tu as run un sync admin avant que l'utilisateur ne s'inscrive). Sans ce mécanisme, on créerait un nouveau dossier vide et l'historique serait perdu ou dupliqué.

### Algorithme à câbler dans `XboxSSOLinkStrategy.createOrRecoverFromXbox` (PR 2)

> **Pour rappel** : la logique est dans un service (`internal/service/xbox_auth_service.go`), pas dans le handler ni dans `userstore.CreateFromXbox` qui reste un primitive store. Le service orchestre store + scanner FS + pool DuckDB + token store.

```
Entrée : (xuid, currentGamertag, title) du SSO Xbox

1. Chercher tous les gamertags historiques de ce XUID :
     SELECT DISTINCT gamertag FROM shared.xuid_aliases WHERE xuid = ?
   → liste possiblement vide, mono- ou multi-élément (rename historique).
   Note : shared.xuid_aliases est globalisé cross-titre (ADR 0008).

2. Pour chaque gamertag candidat, vérifier si data/games/{title}/players/{gamertag}/stats.duckdb existe (multi-titre, cf. ADR 0008 — isolation par chemin FS).
   Algo pur : extraire dans internal/platform/playerfs/orphan_finder.go (testable sans réseau ni DuckDB).
   - 0 match    → pas de BDD orpheline → CreateFromXbox normal, sync from scratch
   - 1 match    → BDD orpheline trouvée → la rattacher au user
   - 2+ matches → cas rare (rename multiple) → prendre la plus récemment modifiée + alerter

3. Si BDD trouvée mais nom dossier != currentGamertag :
   - Renommer le dossier data/games/{title}/players/{oldGT}/ → {currentGT}/
   - Invalider l'entrée du pool DuckDB : pool.Invalidate(title, oldGT)
   - Insérer dans shared.xuid_aliases : (xuid, currentGamertag) si manquant

4. Persister le refresh_token rotaté dans data/auth/watcher_tokens/{xuid}.json (D4 — source unique).
   → la prochaine sync repart immédiatement sans redemander le login.

5. user.Gamertag = currentGamertag, user.XUID = xuid
   sess.CurrentPlayerSlug = currentGamertag
```

### Points de vigilance

- **Pool DuckDB** : vérifier que `pool.Invalidate(title, slug)` existe ; si absent, l'ajouter avant PR 2 (cf. [apps/go-api/internal/platform/duckdb/pool.go](apps/go-api/internal/platform/duckdb/pool.go)). Test : ouvrir une conn, renommer le dossier, appeler `Invalidate`, vérifier qu'une nouvelle conn pointe sur le nouveau chemin.
- **Source unique refresh_token (D4)** : `data/auth/watcher_tokens/{xuid}.json`, **pas dans `sync_meta`**. La table `sync_meta` ne contient plus de tokens — uniquement les metadata de sync (cursor, last_run, etc.).
- **Onboarding UI** : si on trouve une BDD orpheline, **demander confirmation** au user avant d'attacher ("On a trouvé X matchs historiques associés à ton compte, les rattacher ?"). Évite le silent merge si on s'est trompé.
- **Edge case `shared.xuid_aliases` vide** : si le user n'a jamais joué avec personne déjà synchronisé, son XUID n'est pas dans `xuid_aliases`. Fallback : appel léger à l'API Halo (`gamertag → XUID` ou inverse) pour valider l'identité avant de scanner le filesystem.

---

## 12. Hors scope & plans annexes

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

## 13. Subscription RTA auto-provisionnée après login

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

**Changements dans `XboxSSOLinkStrategy.OnAuthSuccess`** (cf. §4 PR 2 LinkStrategy) :
```go
// Après provider.Exchange() qui retourne tokens Halo + identité :
xstsRTA, err := auth.AcquireXSTSForRTA(ctx, accessToken)
if err != nil {
    slog.WarnContext(ctx, "rta provisioning failed, user logged in without auto-sync", "err", err)
    // NON BLOQUANT — l'user peut quand même utiliser l'app
} else {
    s.tokenStore.UpsertUserXSTS(xuid, gamertag, xstsRTA, rotatedRefreshToken)
    s.watcher.SubscribeXUID(xuid)  // démarre la souscription RTA immédiate
}
```

**Changements dans le `TokenStore` (D4)** :
- Layout : `data/auth/watcher_tokens/{xuid}.json` (1 fichier par XUID), pas un fichier global.
- `UpsertUserXSTS(xuid, gamertag, xsts, refreshToken)` : write-to-temp + `os.Rename` atomique, perms 0600.
- `LoadAll() (map[XUID]TokenSet, error)` : scan du dossier au boot, reconstruction du map RAM.
- `Remove(xuid)` : `os.Remove` du fichier (révocation).
- Zéro contention : chaque `RefreshLoop` (1 par user) écrit son propre fichier — pas de mutex global.

**Changements dans `RefreshLoop`** :
- Le `RefreshLoop` actuel ([refresh_loop.go](apps/go-api/internal/platform/auth/refresh_loop.go)) est déjà conçu autour d'un `TokenStore`. Le changement : instancier N loops (1 par XUID actif) ou refactor en single loop qui itère.
- Reco : single loop qui itère sur `tokenStore.LoadAll()` à chaque tick — plus simple, pas de leak de goroutine si un user est supprimé.

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
