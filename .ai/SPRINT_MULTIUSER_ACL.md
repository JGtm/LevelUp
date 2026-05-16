# Plan : Multi-utilisateurs avec ACL "cercle d'amis"

> **Statut** : Plan d'implémentation futur.
> **Dépendance** : **bloqué par le SSO Xbox** (`SPRINT_XBOX_SSO.md`). Sans SSO, l'ACL est contournable (un user peut s'inscrire avec n'importe quel username password local).
> **Branche cible** : à créer (`feat/multiuser-acl`), depuis `main` après le merge du SSO.
> **Auteur du plan** : Claude (session du 2026-05-16).

---

## 0. Contexte

Aujourd'hui, le multi-user existe au niveau données (`data/players/{gamertag}/stats.duckdb` par joueur) mais **pas au niveau permissions** :

- Tout user connecté peut switcher de joueur via la nav L1.
- Forcer une URL `/players/{n_importe_quel_slug}/...` fonctionne aussi.
- Les BDDs joueurs sont **toutes accessibles** une fois l'auth locale passée.

**Cas d'usage cible** : un petit groupe d'amis qui partagent une instance LevelUp et veulent voir mutuellement leurs stats, sans qu'un random qui aurait un compte sur la même instance puisse voir leurs BDDs.

---

## 1. Décisions de design à prendre AVANT de coder

### 1.1 Modèle d'autorisation

| Modèle | Description | Pour | Contre |
|---|---|---|---|
| **Symétrique (mutual)** | Bob ajoute Alice → Alice doit confirmer → ils se voient mutuellement | UX naturelle ("amis Facebook"), pas de stalking unilatéral | Friction (2 actions pour relier) |
| **Unilatéral** | Bob autorise Alice → Alice voit Bob, indépendamment du sens inverse | Souple (un admin peut tout autoriser unilatéralement) | Asymétrie déroutante |
| **Groupes** | Un "squad" est créé, on invite par gamertag, tous les membres se voient | Match ton vrai use case (groupe d'amis fixe) | Plus de modèle à coder |

**Reco** : commencer par **symétrique simple** (graphe d'amitié non orienté), avec possibilité d'ajouter les groupes plus tard.

### 1.2 Découverte des amis à ajouter

| Méthode | Pour | Contre |
|---|---|---|
| Saisie manuelle du gamertag | Simple | Faute de frappe = erreur |
| Auto-suggestion depuis `shared.match_participants` | Très naturel ("vous avez joué 47 matchs avec ce joueur, l'ajouter ?") | Suggère uniquement les gens déjà sync |
| Code d'invitation (réutiliser `invite_store.go`) | Sécurisé, marche pour des gens hors instance | Plus lourd |

**Reco** : combinaison **auto-suggestion + saisie manuelle par gamertag**. Tu as déjà tout ce qu'il faut dans `shared.match_participants` pour suggérer ([cf. `analysis/squad_breakdown.go`](apps/go-api/internal/analysis/squad_breakdown.go)).

### 1.3 Granularité

| Niveau | Description |
|---|---|
| **All-or-nothing** | Ami = voit tout, non-ami = voit rien | ← reco MVP |
| **Mode "shared matches only"** | Ami voit seulement les matchs joués ensemble | Pertinent si certains amis sont juste "compagnons de jeu", pas "amis proches" |
| **Mode "public profile"** | Page profil basique visible par tous les users de l'instance, détails uniquement pour amis | Plus complexe, à voir plus tard |

**Reco** : MVP all-or-nothing, ajouter un toggle "public profile" plus tard si besoin.

### 1.4 Bypass admin

- L'admin (`UserRole = admin`) doit-il pouvoir voir toutes les BDDs ?
- **Reco** : oui, par défaut activé. Sinon impossible de debug. Logger chaque accès admin pour audit.

---

## 2. PR 1 — Domain & store des amitiés

**Périmètre** : modélisation persistance.

- Nouveau type dans `apps/go-api/internal/domain/friendship.go` :
  ```go
  type Friendship struct {
      UserA       string `json:"user_a"`    // slug, ordre lex (UserA < UserB)
      UserB       string `json:"user_b"`
      RequestedBy string `json:"requested_by"`
      Status      string `json:"status"`    // "pending" | "accepted" | "blocked"
      CreatedAt   string `json:"created_at"`
      AcceptedAt  string `json:"accepted_at,omitempty"`
  }
  ```
- Nouveau store `apps/go-api/internal/platform/userstore/friendship_store.go` :
  - Persistance JSON `data/auth/friendships.json` (cohérent avec le pattern `users.json` + `invites.json`)
  - `Request(from, to)`, `Accept(from, to)`, `Decline(from, to)`, `Remove(a, b)`, `List(user)`, `IsFriend(a, b)`
- Tests : couvrir les invariants (un seul Friendship par paire, ordre lexicographique respecté)

---

## 3. PR 2 — Middleware d'autorisation backend

**Périmètre** : enforcement *côté serveur*, pas seulement masquage UI.

- Nouveau middleware `apps/go-api/internal/api/middleware/player_acl.go` :
  ```go
  func RequirePlayerAccess(store *userstore.Store, friendships *userstore.FriendshipStore) func(http.Handler) http.Handler
  ```
  - Extrait le `{playerSlug}` de l'URL (chi pattern)
  - Vérifie : `currentUser.Gamertag == playerSlug` (sa propre BDD) OU `friendships.IsFriend(currentUser, playerSlug)` OU `currentUser.Role == admin`
  - Sinon → 403 Forbidden
- Brancher sur **toutes les routes** qui prennent un `playerSlug` dans `apps/go-api/internal/api/server.go` :
  - `/api/players/{slug}/...`
  - `/api/sync/{slug}/...`
  - `/api/backfill/{slug}/...`
  - `/api/media/{slug}/...`
- **Audit** : grep tous les `chi.URLParam(r, "playerSlug")` ou `"player_slug"` pour s'assurer qu'aucune route ne contourne le middleware.
- Tests : pour chaque route protégée, cas 200/403/admin-bypass.

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

## 5. PR 4 — UI gestion des amis

**Périmètre** : page dédiée.

- Nouvelle page `apps/web/src/features/friends/FriendsPage.tsx` (route `/friends`)
  - 3 sections : "Amis", "Demandes reçues", "Demandes envoyées"
  - Recherche par gamertag pour envoyer une demande
  - Suggestions auto (carrousel de coéquipiers fréquents)
- Composant `FriendCard` réutilisable
- Notification toast quand une demande arrive (cf. `useJobToasts` pour le pattern)
- Tests : `FriendsPage.test.tsx` avec MSW

---

## 6. PR 5 — Nav L1 filtrée

**Périmètre** : masquer les joueurs non-amis de la navigation.

- Modifier le bootstrap (`/api/bootstrap`) pour ne retourner que les joueurs accessibles à l'user courant (au lieu de tous les joueurs sync).
- [apps/go-api/internal/service/bootstrap_service.go](apps/go-api/internal/service/bootstrap_service.go) : filtrer la liste `available_players` via le friendship store.
- [apps/web/src/components/shell/NavL1.tsx](apps/web/src/components/shell/NavL1.tsx) (ou équivalent) : pas de changement nécessaire — il consomme déjà la liste filtrée par le backend.
- **NE PAS** filtrer uniquement côté frontend — le backend doit aussi rejeter avec 403 (cf. PR 2).
- Tests : bootstrap renvoie 1 joueur si l'user n'a 0 ami, 3 joueurs s'il en a 2 amis + lui-même.

---

## 7. PR 6 — Admin override + audit

**Périmètre** : trace + bypass admin.

- L'admin contourne le middleware ACL mais chaque accès "cross-user" admin log un événement dans un journal (`data/auth/admin_access.log` ou stdout).
- Page admin dans `SettingsPage` : visualiser toutes les amitiés du système (debug).

---

## 8. Pièges à éviter

1. **Frontend-only ACL** : si tu masques juste la nav sans middleware backend, un user peut accéder via `fetch('/api/players/Alice/synthesis')` depuis la console. Toujours doubler : backend STRICT + frontend pour l'UX.
2. **Bootstrap qui leak** : le `bootstrap` actuel renvoie probablement toute la liste des joueurs. À filtrer SANS exception.
3. **Routes `shared/*`** : tes endpoints "shared" (registry, participants…) ne sont *probablement* pas filtrés par slug. À auditer : est-ce qu'on peut voir les stats d'un non-ami via un endpoint cross-player ? Si oui, soit on les filtre au niveau requête, soit on les ferme aux non-admins.
4. **Système de match-view** : la page de match expose les stats de TOUS les joueurs du match (`MatchScoreboard.tsx`). Décision à prendre : on garde l'exposition (c'est public dans le match) ou on masque les non-amis avec un placeholder "Joueur masqué" ?
5. **Suppression d'ami** : que se passe-t-il pour les données déjà synchronisées des matchs en commun ? Elles restent dans `shared.match_participants` (c'est public). On masque juste l'accès à la *BDD individuelle* de l'ex-ami.
6. **Squad analysis** : ton `squad_breakdown.go` aggrège plusieurs joueurs. Si un user demande l'analyse d'un squad incluant un non-ami → 403 sur tout le squad ou retourner les stats de l'ami uniquement ? À trancher.
7. **Sync admin déclenchée par l'admin sur un joueur dont le owner refuse l'accès** : à autoriser (admin override) mais logger.

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
| PR 1 (domain + store) | 0.5j | Oui |
| PR 2 (middleware) | 1j | Oui (dépend PR 1) |
| PR 3 (endpoints) | 0.5j | Oui (dépend PR 1) |
| PR 4 (UI gestion amis) | 1j | Non (peut être stub `curl` au début) |
| PR 5 (nav L1 filtrée) | 0.5j | Non |
| PR 6 (admin audit) | 0.5j | Non |

**Total** : ~4 jours de dev focused.

---

## 11. Avant de démarrer

- [ ] Vérifier que `feat/xbox-sso` est mergé et stable depuis au moins 2 semaines (pour avoir un retour utilisateur sur le SSO avant d'empiler).
- [ ] Décider du modèle d'autorisation (§1.1) — recommandation : symétrique.
- [ ] Décider de la stratégie sur la page de match (§ pièges 4) — masquer ou laisser visible.
- [ ] Trancher l'audit `shared/*` (§ pièges 3) — quels endpoints leaker.
- [ ] Préparer un plan de migration data pour les installations existantes (§9).
