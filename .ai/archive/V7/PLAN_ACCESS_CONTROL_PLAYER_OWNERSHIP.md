# PLAN — Contrôle d'accès : propriété des joueurs + participation match/session

> Statut : Plan validé, implémentation à venir
> Date : 2026-05-31
> Branche cible : `feat/access-control-player-ownership-and-participation` (à créer depuis la branche courante)
> Décisions verrouillées : 403 (pas 404) pour slug étranger ; unlinked password user ne possède rien ; mode demo / `auth_mode=none` totalement ouverts.

## Critère de succès

1. Un utilisateur `role=user` ne peut accéder à **aucun** endpoint player-scoped d'un slug
   qu'il ne possède pas → **HTTP 403** `player_forbidden`.
2. Le menu L1 (`availablePlayers`) ne liste que les joueurs possédés par l'utilisateur courant.
3. À l'intérieur de son propre slug, ouvrir un **match non-participé** affiche une page
   « Indisponible » avec boutons (Accueil / Mes matchs), pas une page mal renseignée.
4. Demander une **session inexistante** affiche « Indisponible » au lieu du fallback silencieux
   sur la dernière session.
5. `role=admin`, mode `demo` et `auth_mode=none` conservent l'accès complet (non-régression).

---

## Contexte & cause racine

### Trou A — Aucune liaison utilisateur ↔ joueur autorisé

Aujourd'hui rien ne lie un utilisateur authentifié à un joueur précis :

- `/bootstrap` renvoie **TOUS** les profils de `db_profiles.json` dans `availablePlayers`, sans
  filtrage par propriétaire ([bootstrap_service.go:59-156](../apps/go-api/internal/service/bootstrap_service.go#L59-L156)).
- Le garde frontend ([$playerSlug.tsx:23-48](../apps/web/src/routes/players/$playerSlug.tsx#L23-L48))
  valide seulement que le slug **existe** dans cette liste → tout slug configuré passe.
- Le backend ([player_resolver.go:69-89](../apps/go-api/internal/config/player_resolver.go#L69-L89))
  ouvre la DB de **n'importe quel slug** présent dans `db_profiles.json`, sans vérifier la propriété.
- `RequireAuth` ([require_auth.go](../apps/go-api/internal/api/middleware/require_auth.go)) vérifie
  « est connecté », jamais « a le droit d'accéder à CE joueur ».

Conséquence : changer le slug dans l'URL bascule vers les données d'un autre joueur. Acceptable en
mono-user, **fuite de données** en multi-user strict (cible produit, cf. ADR 0009).

### Trou B — Dégradation silencieuse des pages match/session

- **Match** : `GetMatchMeta` lit le match dans la DB **partagée** (existe dès qu'un joueur configuré
  y a joué). `GetPlayerMatchStats(xuid, matchId)` fait `WHERE match_id=? AND xuid=?` ; si 0 ligne →
  retourne des **stats vides sans erreur** ([match_view_repo.go:187-211](../apps/go-api/internal/platform/duckdb/match_view_repo.go#L187-L211)).
  Résultat : scoreboard d'un autre joueur + stats personnelles à zéro = « page mal renseignée ».
- **Session** : si le `session_label` demandé n'existe pas, fallback silencieux sur la dernière
  session via `lastOrNil` ([session_page_service.go:88](../apps/go-api/internal/service/session_page_service.go#L88)) — affiche « une » session en faisant croire que c'est la bonne.

### Le pont d'ownership existe déjà

Système de comptes complet ([userstore/store.go](../apps/go-api/internal/platform/userstore/store.go),
`data/auth/users.json`) :

- `domain.User{Username, Role (admin|user), Gamertag, XUID}` ([user.go:14-23](../apps/go-api/internal/domain/user.go#L14-L23)).
- `LinkIdentity()`, `GetByXUID()`, `AuthenticateByXUID()`, `CreateFromXbox()` lient un humain à son xuid Halo.
- La session porte `Username` + `LinkedHaloIdentity{gamertag, xuid}` ([session.go:17-38](../apps/go-api/internal/domain/session.go#L17-L38)).

Chaque profil de `db_profiles.json` a un `xuid` ([db_profiles.json:7](../db_profiles.json#L7)).
**Le pont `user.xuid ↔ profile.xuid` existe ; il n'est juste pas utilisé pour autoriser.**

---

## Décision : mécanisme d'ownership = correspondance XUID

Un utilisateur possède le profil joueur dont le `xuid` == son xuid lié. Pas de nouveau champ `owner`
dans `db_profiles.json` (évite une double source de vérité à synchroniser).

| Cas | Comportement |
|---|---|
| `role=user`, profil possédé (`user.xuid == profile.xuid`) | Accès OK |
| `role=user`, slug étranger | **403** `player_forbidden` |
| `role=admin` | Accès à tous les profils |
| Mode `demo` / `auth_mode=none` / profil `is_demo` | Aucune restriction (inchangé) |
| Password user sans xuid lié | Ne possède rien → invite UI à lier son compte Halo |
| Match non-participé (dans son propre slug) | 404 `match_not_participant` → page « Indisponible » |
| Session demandée inexistante | 404 `session_not_found` → page « Indisponible » |

Extension différée : alts multiples par user (`User.XUIDs []string`). Hors scope initial, à noter dans l'ADR.

### Deux couches complémentaires (ni l'une ni l'autre suffit seule)

- **Couche A — Propriété du joueur** : tu n'ouvres `/players/{slug}/...` que pour un slug possédé.
- **Couche B — Participation à la ressource** : dans ton propre slug, un match/session non-participé → « Indisponible ».

La couche A ne protège pas la B (le slug est le tien, c'est le match qui ne l'est pas), et inversement.

---

## Phases

### Phase 0 — ADR + cadrage
- ADR `docs/adr/0024-multi-user-player-ownership.md` : modèle XUID-match, règles admin/user/none/demo,
  unlinked = rien, extension alts différée. + version FR si `docs/FR/` concerné (règle 18).
- Entrée `thought_log.md`.

### Phase 1 — Couche A backend : autorisation par propriété (cœur sécurité)
Point d'étranglement unique : toutes les routes player-scoped passent par
`ServiceRegistry.resolve(ctx, slug)` → `config.ResolvePlayer`. On greffe l'autorisation là.

1. **`internal/authz` (nouveau package, logique pure)** :
   `CanAccessPlayer(sess, user, profileXUID, mode) error`. Zéro I/O. Encapsule la règle.
2. **Helper identité** : `currentUserXUID(ctx, sess, userstore)` — `sess.Username` → `userstore.Get()` →
   `user.XUID` (password) ; ou `sess.LinkedHaloIdentity.XUID` (xbox).
3. **Branchement chokepoint** : après résolution du profil + son `xuid`, appeler `authz.CanAccessPlayer`.
   Refus → `domain.APIError{Code: "player_forbidden"}`.
4. **Handlers** : mapper `player_forbidden` → **HTTP 403** (distinct du 404 `player_not_found`).
   `slog.WarnContext(ctx, "authz: accès joueur refusé", "slug", slug, ...)`.
5. **Garde-fou** : middleware `RequirePlayerOwnership` réutilisable (defense-in-depth) ; la vraie
   barrière reste l'enforcement au resolve.

### Phase 2 — Couche A : filtrer `availablePlayers` (UX menu L1)
- `bootstrap_service.Build()` : ne renvoyer que les profils possédés (réutilise `authz.CanAccessPlayer`).
- Effet de bord : le garde `$playerSlug.tsx` redirige automatiquement si on tape l'URL d'un joueur
  non-possédé — sans nouveau code frontend.

### Phase 3 — Couche B backend : participation match/session
- **Match** ([match_view_service.go](../apps/go-api/internal/service/match_view_service.go)) : après
  `GetMatchMeta`, check `repo.IsParticipant(ctx, xuid, matchID)` (`EXISTS` via Q17). Non-participant →
  `domain.APIError{Code: "match_not_participant"}` **avant** les ~20 chargements parallèles
  (fail-fast + perf). Handler → **404** avec ce code.
- **Session** ([session_page_service.go:88](../apps/go-api/internal/service/session_page_service.go#L88)) :
  supprimer le fallback `lastOrNil`. Si `req.SessionLabel != nil` et absent de `availableSessions` →
  `domain.APIError{Code: "session_not_found"}` → **404**. (Sans label demandé, garder la dernière session.)

### Phase 4 — Frontend : page « Indisponible » + états distincts
- Composant réutilisable `PageUnavailable` (basé sur `EmptyStateCard`) avec boutons **Accueil**
  (`/players/{slug}/home`) et **Mes matchs**. Pas de redirect auto (choix validé).
- `client.ts` expose déjà `ApiError.status` + `.code` : **arrêter le matching `message.includes('404')`**,
  brancher sur `error.code`.
- `MatchViewPage.tsx` : `match_not_participant` → « Tu n'as pas participé à ce match » ;
  `match_not_found` → « Match introuvable » ; `player_forbidden` (403) → « Accès non autorisé ».
- `SessionDetailPage.tsx` : `session_not_found` → `PageUnavailable` session.
- i18n FR+EN dans les manifests TOML (`common`/`match`/`session`), via `formatMessage`. Pas de hex/Tailwind
  couleur en dur (règle 20).

### Phase 5 — Tests (chaque couche)
- `internal/authz` : table-tests purs (admin / user owner / user non-owner / none / demo / unlinked).
- Resolve/registry : refuse un slug non-possédé (mock userstore + `middleware.InjectSession`).
- Handlers `httptest` : 403 slug étranger, 404 `match_not_participant`, 404 `session_not_found`.
- `bootstrap_service` : `availablePlayers` filtré selon le user.
- Frontend : branches d'erreur (`PageUnavailable` par code), typecheck + vitest **hors sandbox**.
- Non-régression : `admin` voit tout ; mode `none`/`demo` inchangés.

### Phase 6 — Livraison
- `delivery-checklist` (tests verts CGO, lint, logging slog, multi-titres OK car xuid global).
- Entrée `thought_log.md` finale.
- Note migration : users password existants sans `xuid` lié → « propriétaires de rien » → message UI
  « lie ton compte Halo » + tolérance admin. À valider en Phase 1.

---

## Conformité architecture (grille plan-review)

- **Logique pure** → `internal/authz/` (0 I/O). **Types** → réutilise `domain.User`, `domain.APIError`.
- **Orchestration/enforcement** → chokepoint `ServiceRegistry.resolve` / `config.ResolvePlayer`.
- **Handlers** : pas de logique métier, juste mapping erreur → statut HTTP.
- **Multi-titres** : ownership title-agnostic (xuid global, cf. ADR 0008) ; profils `db_profiles.json`
  par titre déjà gérés par `LoadPlayers(titleFilter)`.
- **Logging** : `slog.WarnContext` sur refus d'accès. Pas de `fmt.Println`.
- **Frontend** : composant via i18n manifests + query keys existantes ; pas de couleur en dur.
- **Done definition** : par phase + thought_log + tests verts.

## Points de décision (verrouillés)

1. Slug étranger → **403** `player_forbidden` (pas 404). *Validé.*
2. Unlinked password user → ne possède rien + invite à lier. *Validé.* Exception : **mode demo ouvert**.

## Fichiers clés

| Rôle | Chemin |
|---|---|
| Chokepoint resolve | `apps/go-api/internal/config/player_resolver.go`, `internal/api/registry.go` |
| Bootstrap (liste L1) | `apps/go-api/internal/service/bootstrap_service.go` |
| User store (ownership) | `apps/go-api/internal/platform/userstore/store.go`, `internal/domain/user.go` |
| Session | `apps/go-api/internal/domain/session.go`, `internal/api/middleware/{session,require_auth}.go` |
| Match view | `apps/go-api/internal/service/match_view_service.go`, `internal/platform/duckdb/match_view_repo.go` |
| Session page | `apps/go-api/internal/service/session_page_service.go` |
| Front match | `apps/web/src/features/match-view/MatchViewPage.tsx` |
| Front session | `apps/web/src/features/session-detail/SessionDetailPage.tsx` |
| Front garde slug | `apps/web/src/routes/players/$playerSlug.tsx` |
| Front API client | `apps/web/src/lib/api/client.ts` |
