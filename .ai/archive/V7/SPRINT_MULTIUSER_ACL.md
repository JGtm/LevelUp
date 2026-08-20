# Plan : Multi-utilisateurs avec ACL "cercle d'amis"

> **Statut** : Décisions tranchées le 2026-05-18 (cf. §1, options retenues marquées ✅). Prêt à implémenter après SSO.
> **Dépendance** : **bloqué par le SSO Xbox** (`SPRINT_XBOX_SSO.md`). Sans SSO, l'ACL est contournable (un user peut s'inscrire avec n'importe quel username password local — sauf en mode `AuthMode="xbox"` où c'est verrouillé aux admins, mais le mode xbox EST le SSO).
> **Branche cible** : à créer (`feat/multiuser-acl`), depuis `main` après le merge du SSO.
> **Auteur du plan** : Claude (session du 2026-05-16, décisions du 2026-05-18).

---

## 0. Contexte

Aujourd'hui, le multi-user existe au niveau données (`data/players/{gamertag}/stats.duckdb` par joueur) mais **pas au niveau permissions** :

- Tout user connecté peut switcher de joueur via la nav L1.
- Forcer une URL `/players/{n_importe_quel_slug}/...` fonctionne aussi.
- Les BDDs joueurs sont **toutes accessibles** une fois l'auth locale passée.

**Cas d'usage cible** : un petit groupe d'amis qui partagent une instance LevelUp et veulent voir mutuellement leurs stats, sans qu'un random qui aurait un compte sur la même instance puisse voir leurs BDDs.

---

## 1. Décisions tranchées (2026-05-18)

### 1.1 Modèle d'autorisation — **Symétrique avec accept** ✅

Bob demande → Alice doit accepter → amitié active (status `pending` → `accepted`). Admin peut bypass et créer une amitié forcée via Settings (action loggée dans l'audit, cf. §1.4).

> Rejeté : unilatéral (asymétrie déroutante), groupes (overkill pour le MVP, à rouvrir si un signal user le justifie).

### 1.2 Découverte des amis à ajouter — **Auto-suggest + saisie manuelle** ✅

- **Auto-suggest** : top-N coéquipiers fréquents du user courant (`SELECT teammate_xuid, COUNT(*) FROM shared.match_participants WHERE my_match_ids GROUP BY teammate_xuid ORDER BY n DESC LIMIT 10`). Carrousel "Tes coéquipiers fréquents" dans la page Friends.
- **Saisie manuelle** : champ texte "Chercher par gamertag" → lookup via `shared.xuid_aliases` (gamertag → XUID), créer la demande sur le XUID résolu.

> Rejeté : code d'invitation (over-engineering pour cercle privé).

### 1.3 Granularité — **All-or-nothing au niveau player-scoped** ✅

**Périmètre ACL** :
- Routes **player-scoped** (`/api/players/{slug}/*`, `/api/sync/{slug}/*`, `/api/backfill/{slug}/*`, `/api/media/{slug}/*`) : ACL stricte. Voir = être ami OU self OU admin.
- Routes **match-level** (`/api/matches/{matchId}/*`, scoreboards) et **`shared/*`** : restent **publics-authentifiés** (philosophie Halowaypoint : "tout ce qui touche à un match est public").
- **Pas de masquage UI** sur `MatchScoreboard` ni sur `SquadAnalysis` — ces vues lisent uniquement `shared.match_participants` qui est public.
- Filtrage de la nav L1 : `bootstrap.available_players` retourne uniquement les players accessibles à l'user courant (amis + self + tous si admin).

> Rejeté : "shared matches only" (complexité disproportionnée), "public profile" (rouvrable post-MVP si besoin).

### 1.4 Bypass admin — **Oui avec audit log** ✅

`Role=admin` contourne l'ACL sur toutes les routes player-scoped. Chaque accès cross-user est loggé :

```go
slog.WarnContext(ctx, "admin_cross_user_access",
    "audit", true, "admin", currentUser.Username, "target", targetSlug, "route", r.URL.Path)
```

Le tag `audit=true` permet un filtrage facile dans les outils d'observabilité.

### 1.5 Identifiant friendship — **XUID** ✅

La struct `Friendship` stocke `XUIDA` + `XUIDB` (slug uniquement pour affichage UI). Résiste aux renames de gamertag Xbox (~1×/an).

### 1.6 Suppression de compte — **Hard delete amitiés + soft delete user** ✅

Quand un user est supprimé (par lui-même ou par admin) :
- Toutes les amitiés dont il est partie prenante (`XUIDA == his_xuid OR XUIDB == his_xuid`) sont **hard-deleted** de `friendships.json`.
- Toutes les demandes pending où il est expediteur ou destinataire sont nettoyées.
- Le user lui-même passe en `Role=deleted` (soft), pour garder l'historique cross-référence dans les matchs publics. Son `users.json` slot reste mais ne peut plus se logger.

---

## 2. PR 1 — Domain & store des amitiés

**Périmètre** : modélisation persistance. **Clé = XUID** (D6 / §1.5).

- Nouveau type dans `apps/go-api/internal/domain/friendship.go` :
  ```go
  type Friendship struct {
      XUIDA       string `json:"xuid_a"`         // ordre lex (XUIDA < XUIDB)
      XUIDB       string `json:"xuid_b"`
      RequestedBy string `json:"requested_by"`   // XUID
      Status      string `json:"status"`         // "pending" | "accepted"
      CreatedAt   string `json:"created_at"`
      AcceptedAt  string `json:"accepted_at,omitempty"`
  }
  ```
  > **Note** : pas de status `"blocked"` au MVP. Le besoin pourra être rouvert si quelqu'un se plaint.
- Nouveau store `apps/go-api/internal/platform/userstore/friendship_store.go` :
  - Persistance JSON `data/auth/friendships.json` (cohérent avec le pattern `users.json` + `invites.json`).
  - Cache RAM avec invalidation sur write (RWMutex), comme `userstore.Store`. Évite le re-parse JSON à chaque `IsFriend()`.
  - API en termes de **XUID**, pas de slug :
    ```go
    Request(fromXUID, toXUID string) error
    Accept(fromXUID, toXUID string) error
    Decline(fromXUID, toXUID string) error
    Remove(xuidA, xuidB string) error
    List(xuid string) ([]Friendship, error)              // accepted + pending
    ListAccepted(xuid string) ([]string, error)          // shortcut → liste XUIDs amis
    IsFriend(xuidA, xuidB string) (bool, error)
    DeleteAllFor(xuid string) error                      // utilisé à la suppression de compte (§1.6)
    ```
- Tests : invariants (un seul Friendship par paire, ordre lex respecté, pending → accepted), perf (cache hit sur lecture répétée), suppression cascade.

---

## 3. PR 2 — Service ACL + middleware

**Périmètre** : enforcement *côté serveur*. Séparation `service` (logique) / `middleware` (thin wrapper HTTP) selon `arch-rules`.

### 3.1 Service ACL — logique testable

Nouveau service `apps/go-api/internal/service/acl_service.go` :

```go
type ACLService struct {
    userStore   *userstore.Store
    friendships *userstore.FriendshipStore
}

func (s *ACLService) CanAccessPlayer(ctx context.Context, currentUser *domain.User, playerSlug string) (bool, error) {
    if currentUser.Role == domain.RoleAdmin {
        slog.WarnContext(ctx, "admin_cross_user_access",
            "audit", true, "admin", currentUser.Username, "target", playerSlug)
        return true, nil
    }
    if currentUser.Gamertag == playerSlug {
        return true, nil // self
    }
    targetUser, err := s.userStore.GetByGamertag(playerSlug)
    if err != nil { return false, err }
    return s.friendships.IsFriend(currentUser.XUID, targetUser.XUID)
}
```

Tests unitaires : self, ami, non-ami, admin-bypass (avec capture du log audit), user supprimé.

### 3.2 Middleware HTTP — thin wrapper

Nouveau `apps/go-api/internal/api/middleware/player_acl.go` :

```go
func RequirePlayerAccess(acl *service.ACLService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            slug := chi.URLParam(r, "playerSlug") // ou "slug" selon route
            sess := middleware.GetSession(r.Context())
            user, _ := userStore.Get(*sess.Username)
            ok, err := acl.CanAccessPlayer(r.Context(), user, slug)
            if err != nil { writeError(w, 500, "acl_error", err.Error()); return }
            if !ok { writeError(w, 403, "player_access_denied", ""); return }
            next.ServeHTTP(w, r)
        })
    }
}
```

### 3.3 Routes protégées (périmètre D5 / §1.3)

Brancher sur **uniquement** les routes player-scoped dans `apps/go-api/internal/api/server.go` :
- `/api/players/{slug}/...`
- `/api/sync/{slug}/...`
- `/api/backfill/{slug}/...`
- `/api/media/{slug}/...`

**Hors périmètre ACL** (restent publics-authentifiés) : `/api/matches/{matchId}/*`, `/api/shared/*`, `/api/bootstrap` (filtré au niveau service, cf. §6), `/api/asset/*`, `/api/leaderboards/*`.

### 3.4 Test de couverture automatique

Nouveau test `apps/go-api/internal/api/acl_coverage_test.go` :
- Parse les routes chi via `chi.Walk`.
- Pour chaque route avec `{slug}` ou `{playerSlug}` : vérifier qu'elle est wrappée par `RequirePlayerAccess`.
- Échoue si une route player-scoped est ajoutée sans middleware → garantie que les futures PR ne contournent pas l'ACL.

---

## 4. PR 3 — Endpoints de gestion des amis

**Périmètre** : API REST pour le frontend.

- Handler `apps/go-api/internal/api/handlers/friends.go` :
  - `GET /friends` → liste amis acceptés + demandes en attente
  - `POST /friends/request` `{target_gamertag}` → crée une demande
  - `POST /friends/{id}/accept`
  - `POST /friends/{id}/decline`
  - `DELETE /friends/{id}` → retire un ami
  - `GET /friends/suggestions` → suggère depuis `shared.match_participants` (top N coéquipiers du user en cours)
- Tests handlers + integration avec sessions

---

## 4bis. PR 3.5 — Page Preview + recherche + invite (use case "trouver / vérifier / ajouter")

**Périmètre** : nouveau use case décidé le 2026-05-18. La page profil n'est **pas un dashboard**, c'est un outil de **désambiguïsation + invitation** dans le flow d'ajout d'ami. 3 vues conceptuelles :

| Vue | URL | Qui y accède | Source données |
|---|---|---|---|
| **Preview publique** | `/players/{slug}/preview` | Tout user authentifié | `shared.*` uniquement (registry, xuid_aliases, agrégats publics) |
| **Profil ami** | `/players/{slug}/synthesis` (existe) | Soi + amis + admin | BDD individuelle (déjà câblé) |
| **Self** | `/me` (existe) | Soi seul | Idem ami + sections perso |

**Règle de routage `/players/{slug}` (sans sous-path)** : si l'user est ami / self / admin → redirect vers `synthesis`. Sinon → redirect vers `preview` (pas de 403 brut).

### 4bis.1 Backend

**Endpoint recherche** : `GET /api/players/search?q={prefix}` (auth requise)

```sql
-- Resolver basé UNIQUEMENT sur shared.xuid_aliases (D — l'utilisateur a explicitement
-- choisi de ne PAS faire de fallback API Halo. Raison : si on n'a aucune donnée sur
-- ce joueur, la preview serait vide de toute façon → autant le dire honnêtement)
SELECT DISTINCT xa.gamertag, xa.xuid,
       EXISTS(SELECT 1 FROM users.json WHERE xuid = xa.xuid) AS has_levelup_account
FROM shared.xuid_aliases xa
WHERE LOWER(xa.gamertag) LIKE LOWER(?) || '%'
ORDER BY xa.gamertag
LIMIT 20
```

Si aucun résultat → 200 avec `{matches: []}` + frontend affiche "Pas encore visible sur cette instance — demande à un membre de jouer une partie avec lui d'abord".

**Endpoint preview** : `GET /api/players/{slug}/preview`

Retourne uniquement des données **publiques** (D5 — match-level et shared.* sont publics) :

```json
{
  "gamertag": "Spartan42",
  "xuid": "2535471234567890",
  "avatar_url": "...",
  "career_rank": { "rank": "Onyx", "csr": 1842 },
  "total_matches_seen": 287,
  "has_levelup_account": true,
  "friendship_status": "none" | "pending_sent" | "pending_received" | "accepted",
  "is_self": false
}
```

- `career_rank` lu depuis l'agrégat existant (ou `shared.match_participants` LATEST rank).
- `total_matches_seen` = count distinct `match_id` dans `shared.match_participants WHERE xuid=?`.
- `friendship_status` croise avec `FriendshipStore` pour adapter le bouton frontend.
- Pas d'ACL stricte sur cet endpoint (volontairement) — tous les champs sont publics. Mais auth quand même (pas anonyme).

**Endpoint invitation Xbox** : `POST /api/invites/xbox` `{xuid, gamertag}` (réutilise `invite_store.go`)

- Crée un invite avec un champ `linked_xuid` : à l'inscription via cet invite, le système crée automatiquement une amitié `accepted` entre l'inviteur et le nouvel inscrit.
- Retourne `{invite_code, invite_url}` que l'inviteur copie/partage manuellement (mail/discord/whatever).

### 4bis.2 Frontend

- Nouvelle route file-based `apps/web/src/routes/players/$slug/preview.tsx`.
- Composant `PlayerPreviewCard` (réutilisable) : avatar Spartan + gamertag + XUID en petit + carte rang + total matches.
- Composant `FriendshipActionButton` : rendu conditionnel selon `friendship_status` + `has_levelup_account` :
  - `none` + has_account → bouton "Envoyer une demande d'ami"
  - `pending_sent` → "Demande envoyée" disabled + bouton "Annuler"
  - `pending_received` → "Accepter" + "Refuser"
  - `accepted` → "Accéder au profil complet" (lien vers synthesis)
  - `none` + !has_account → bouton "Inviter sur LevelUp" (ouvre modal avec invite_url copiable)
  - **Admin** : bouton supplémentaire "Accéder au profil complet (admin)" qui bypass et loggue audit (cf. D9)
- Page Friends : champ recherche avec debounce 300ms → appelle `/api/players/search` → dropdown de résultats → clic sur un résultat → `navigate('/players/{slug}/preview')`.
- i18n FR + EN pour tous les strings (cf. ADR 0003).

### 4bis.3 Routage conditionnel `/players/{slug}`

Modification de la résolution actuelle de `/players/{slug}` :

```ts
// apps/web/src/routes/players/$slug/index.tsx
const friendshipStatus = useFriendshipStatus(slug);
const userRole = useUserRole();
if (friendshipStatus === 'accepted' || isSelf(slug) || userRole === 'admin') {
  return redirect(`/players/${slug}/synthesis`);
}
return redirect(`/players/${slug}/preview`);
```

Évite le 403 brut quand un user clique sur un nom non-ami quelque part dans l'app.

### 4bis.4 Tests

- Backend : search avec divers gamertags (exact, partial, casse), preview pour ami / non-ami / inconnu / self, invitation Xbox + auto-linkage.
- Frontend : flow complet "tape gamertag → preview → envoie demande", boutons conditionnels selon friendship_status, modal d'invite link.

**Done quand** : un user peut taper un gamertag dans `/friends`, voir la preview du joueur trouvé, et soit envoyer une demande d'ami soit générer un lien d'invitation selon que le joueur a déjà un compte ou non.

---

## 5. PR 4 — UI gestion des amis

**Périmètre** : page dédiée.

- Nouvelle page `apps/web/src/features/friends/FriendsPage.tsx` (route `/friends`)
  - 3 sections : "Amis", "Demandes reçues", "Demandes envoyées"
  - Recherche par gamertag pour envoyer une demande (résout gamertag → XUID via endpoint dédié)
  - Suggestions auto (carrousel de coéquipiers fréquents, D / §1.2)
- Composant `FriendCard` réutilisable
- Notification toast quand une demande arrive (cf. `useJobToasts` pour le pattern)
- **i18n FR + EN** via manifestes TOML (cf. ADR 0003) pour tous les strings (`friends.title`, `friends.request.send`, `friends.accept`, etc.)
- **Pas de hex/Tailwind couleur direct** (CLAUDE.md règle 20) : utiliser `tokenCssVar()` pour toute couleur sémantique
- Tests : `FriendsPage.test.tsx` avec MSW (mock des endpoints `/friends/*`)

---

## 6. PR 5 — Nav L1 filtrée (bootstrap)

**Périmètre** : masquer les joueurs non-amis de la navigation.

- [apps/go-api/internal/service/bootstrap_service.go](apps/go-api/internal/service/bootstrap_service.go) : ajouter une dépendance optionnelle via le pattern `WithFriendshipStore(fs)` (cohérent avec `WithPrivacyProvider`, `WithUserStoreEmpty`) :
  ```go
  func (s *BootstrapService) WithFriendshipStore(fs *userstore.FriendshipStore) *BootstrapService
  ```
- Dans `Build()` : après `cfg.LoadPlayers(currentTitleSlug)`, filtrer via `acl.CanAccessPlayer(ctx, currentUser, playerSlug)` (réutilise le service §3.1).
- Admin : voit tous les players (bypass ACL). Loggé en audit.
- [apps/web/src/components/shell/NavL1.tsx](apps/web/src/components/shell/NavL1.tsx) (ou équivalent) : pas de changement nécessaire — il consomme déjà la liste filtrée par le backend.
- **NE PAS** filtrer uniquement côté frontend — le backend doit aussi rejeter avec 403 (cf. PR 2 §3.3).
- Tests : bootstrap renvoie 1 joueur si l'user n'a 0 ami, 3 joueurs s'il en a 2 amis + lui-même, tous si admin.

---

## 7. PR 6 — Admin override + audit + suppression

**Périmètre** : audit log déjà en place via `ACLService.CanAccessPlayer` (§3.1), reste à câbler les outils admin.

- **Audit log** : déjà émis par `ACLService` à chaque bypass admin (`slog.WarnContext` avec `audit=true`). Pas de fichier dédié — le log slog du serveur est la source. Pour filtrer ex post : `grep audit=true app.log` ou query Loki/Datadog si déployé.
- **Page admin "Amitiés"** dans `SettingsPage` (route `/settings/admin/friendships`) :
  - Tableau de toutes les amitiés (debug global).
  - Bouton "Forcer une amitié" : crée un `Friendship` directement en `accepted` entre 2 XUID. Loggé `slog.WarnContext(ctx, "admin_forced_friendship", "audit", true, ...)`.
  - Bouton "Supprimer une amitié".
- **Suppression de compte** (D8 / §1.6) :
  - Endpoint `DELETE /api/admin/users/{slug}` (admin only) : appelle `userStore.SoftDelete(slug)` + `friendshipStore.DeleteAllFor(xuid)` + `tokenStore.Remove(xuid)` + `watcher.UnsubscribeXUID(xuid)`.
  - Endpoint `DELETE /api/me` (self-service) : idem mais sur l'user courant. Confirme via mot de passe (si admin) ou re-auth Xbox (si user Xbox — défère à PR future).

---

## 8. Pièges à éviter

1. **Frontend-only ACL** : si tu masques juste la nav sans middleware backend, un user peut accéder via `fetch('/api/players/Alice/synthesis')` depuis la console. Toujours doubler : backend STRICT + frontend pour l'UX. Le test de couverture (§3.4) prévient cette régression.
2. **Bootstrap qui leak** : le `bootstrap` actuel renvoie toute la liste des joueurs. À filtrer SANS exception via `WithFriendshipStore` (§6).
3. ~~**Routes `shared/*`**~~ → **caduc** : les routes shared restent publiques-authentifiées (D5 / §1.3 — philosophie Halowaypoint).
4. ~~**Système de match-view**~~ → **caduc** : pas de masquage UI sur `MatchScoreboard`. Les stats de match sont publiques. Seul le lien "Explorer matchs avec ce joueur" (qui dépend potentiellement de la BDD individuelle) doit gérer le 403 gracieusement si on l'ouvre sur un non-ami.
5. **Suppression d'ami** : que se passe-t-il pour les données déjà synchronisées des matchs en commun ? Elles restent dans `shared.match_participants` (public). On masque juste l'accès à la **BDD individuelle** de l'ex-ami.
6. ~~**Squad analysis**~~ → **caduc** : pas de masquage. `squad_breakdown.go` lit uniquement `shared.match_participants`, qui est public.
7. **Sync admin déclenchée par l'admin sur un joueur dont le owner refuse l'accès** : autorisé (admin override) mais loggé via `ACLService` (§3.1).
8. **XUID inconnu lors d'une demande d'amitié manuelle** : la saisie manuelle "Ajouter par gamertag X" doit résoudre X → XUID via `shared.xuid_aliases`. Si X est inconnu (n'a jamais joué avec quelqu'un de sync), proposer un appel API Halo pour le résoudre (fallback) ou afficher "Joueur introuvable — tu dois avoir joué au moins une partie avec lui".
9. **Demande d'amitié envoyée à un user qui n'a pas encore de compte LevelUp** : le `to_xuid` est valide (existe sur Xbox Live) mais n'a pas de slot `users.json`. Décision : créer un placeholder `Friendship.Status="pending_recipient_signup"` ? Ou refuser ? **Reco MVP** : refuser avec un message "Ce joueur n'a pas encore de compte LevelUp", l'inviter à se connecter d'abord. (Rouvrable si signal user.)

---

## 9. Migration des installations existantes

Quand on déploie ça, des comptes existants ont déjà accès à tout. Stratégie :

- À l'activation de l'ACL : **tous les users existants sont considérés "amis" entre eux** (préserver le statu quo).
- L'admin peut ensuite nettoyer manuellement les amitiés via la page admin.
- Flag dans `app_settings.json` : `acl_enabled: false` par défaut, devient `true` après une commande explicite (`cmd/levelup enable-acl`).
- Sans flag → le middleware est no-op → backward compat 100%.

---

## 10. Estimation

| PR | Effort | Bloque la suite ? |
|---|---|---|
| PR 1 (domain + store XUID) | 0.5j | Oui |
| PR 2 (service ACL + middleware + test couverture) | 1j | Oui (dépend PR 1) |
| PR 3 (endpoints friends) | 0.5j | Oui (dépend PR 1) |
| PR 3.5 (preview + search + invite Xbox) | 1.5j | Non (peut paralléliser avec PR 4 et 5) |
| PR 4 (UI gestion amis) | 1j | Non |
| PR 5 (bootstrap filtré nav L1) | 0.5j | Non |
| PR 6 (admin audit + suppression compte) | 0.5j | Non |

**Total** : ~5.5 jours de dev focused.

---

## 11. Avant de démarrer

- [ ] Vérifier que `feat/xbox-sso` est mergé et stable depuis au moins 2 semaines (pour avoir un retour utilisateur sur le SSO avant d'empiler).
- [x] ~~Décider du modèle d'autorisation~~ → symétrique avec accept (§1.1 / D7).
- [x] ~~Décider de la stratégie sur la page de match~~ → caduc, pas de masquage (§1.3 / D5).
- [x] ~~Trancher l'audit `shared/*`~~ → caduc, routes shared publiques (§1.3 / D5).
- [x] ~~Trancher la clé friendship (slug vs XUID)~~ → XUID (§1.5 / D6).
- [x] ~~Trancher bypass admin~~ → oui avec audit slog (§1.4 / D9).
- [x] ~~Trancher suppression de compte~~ → hard delete amitiés + soft delete user (§1.6 / D8).
- [ ] Préparer un plan de migration data pour les installations existantes (§9) — flag `acl_enabled=false` par défaut, activation explicite via `cmd/levelup enable-acl`.
- [ ] Décider comment gérer une demande d'amitié vers un user sans compte LevelUp (§8 piège 9) — reco refus avec message clair.
