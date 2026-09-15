# PLAN — Amis par joueur + invitations « sans groupe » sur instance verrouillée

> Créé le 2026-09-15. Branche : `wt/amis-invitations`. Worktree dédié :
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-amis-invitations` (règle mémoire
> « worktree dédié obligatoire » — ne jamais exécuter dans `LevelUp-go-migration`).
> Base : `feat/v75` @ `2ddef392c`. **Push sur `main` = déploiement prod : interdit sans
> l'utilisateur.**
>
> **Contrat d'exécution : skill `plan-execution` fait foi.** Ordre strict, une étape à la
> fois, aucun report d'étape exécutable, chaque item statué `[x]` / `[~]` (couvert ailleurs,
> référence) / `[!]` (non traité, justification écrite), zéro fix hors périmètre — les
> découvertes vont en §9. Avant de coder et avant de cocher : rouvrir le fichier et la ligne
> cibles (les numéros de ligne ci-dessous datent du 2026-09-15 et peuvent avoir bougé).
>
> **Ce plan a été écrit et NON exécuté** (demande explicite de l'utilisateur, 2026-09-15).

---

## 1. Objectif et critères de succès

### Constat (vérifié sur pièces le 2026-09-15)

L'accès aux profils est déjà cloisonné par **propriété + groupes** (ADR 0029,
`internal/authz/authz.go`, `middleware.RequirePlayerOwnership`, `groupstore.CoMemberXUIDs`) :
un utilisateur `role=user` sans groupe ne voit que son propre profil et ne peut pas changer de
joueur. Deux défauts demeurent :

1. **La liste d'amis est globale.** `app_settings.friend_gamertags` est UNE liste pour toute
   l'instance (`internal/api/handlers/settings.go:623` : « FriendGamertags reste GLOBAL »).
   Elle pilote `is_with_friends` dans TOUTES les player DBs, l'Escouade, la coloration des
   amis dans la vue match, les rencontres de carrière, la progression Prestige d'escouade.
   Pire : le front la lit via `GET /settings`, monté sous `RequireAdmin`
   (`internal/api/server_apiv1.go:492-496`) — **un utilisateur standard n'a donc AUCUNE
   fonctionnalité « amis » aujourd'hui** (403 sur `/settings`, `PATCH` impossible depuis
   `AddFriendFlow`).
2. **L'invitation « sans groupe » ne passe pas le verrou.** Sur instance verrouillée
   (`instance_locked`, early access), la seule invitation qui crée un compte est l'invitation
   DE GROUPE (`internal/service/xbox_auth_service.go:222` : `GroupID == ""` → nil →
   `ErrInstanceLocked`). L'invitation admin (`POST /admin/invites`, sans groupe) est donc
   inopérante en SSO Xbox, et n'a plus d'UI (`apps/web/src/features/auth/queries.ts:86-88`).
   De plus, tout invité (groupe ou non) atterrit sur le Setup en `halo_linked_no_profile` et
   prend un **403 `instance_locked`** sur `POST /setup/players`
   (`internal/api/handlers/setup.go:119-126`) : sans profil pré-créé par l'admin, il est
   coincé.

### Décisions produit tranchées (utilisateur, 2026-09-15)

- **D1** — Les amis sont **par profil joueur** (clé : xuid), stockés dans
  `data/global/player_friends.json` via un store fichier miroir de `groupstore`. Lisible sans
  session HTTP (flux de fond : sync, recompute `is_with_friends`, CLI). Le champ global
  `friend_gamertags` est **supprimé** après une migration idempotente au boot (chaque profil
  configuré hérite de la liste globale moins lui-même).
- **D2** — L'UI de gestion vit sur la page `/groups`, renommée **« Amis et groupes »** : section
  « Mes amis » (liste éditable par le propriétaire du profil) + section groupes existante.
  L'onglet Réglages > Sync perd le champ (admin-only, inaccessible à un utilisateur standard).
- **D3** — Une invitation valide (avec ou sans groupe) **lève le verrou pour la création du
  compte ET porte un droit à usage unique de créer SON profil joueur** (xuid = identité Halo
  liée, déjà exigé par le handler). Le droit est porté par le compte (`users.json`), pas par la
  session : il survit à une déconnexion entre le login et le Setup.
- **D4** — Droits d'édition d'une liste d'amis : **propriétaire direct** (xuid lié) ou
  **admin**. Un co-membre de groupe lit, n'écrit pas.
- **D5** — Génération d'une invitation sans groupe : **admin uniquement**, depuis
  `/admin/management`. L'invitation de groupe reste à la main de tout propriétaire de groupe
  (inchangé).
- **D6** — **Variante retenue le 2026-09-15** (pilote) : gate G2.0 NON validée, l'existence de
  `data/auth/groups.json` en prod n'est pas confirmée. La migration de boot
  `groups: migration groupe par défaut` (`apps/go-api/cmd/server/main.go`) est **conservée**,
  ainsi que `groupstore.MigrateDefault` et son test. Elle est **re-sourcée** : au boot,
  `friendstore.MigrateFromAppSettings` s'exécute d'abord, puis la migration de groupe lit
  `friendStore.Get(xuid de l'admin)` au lieu de `settings.FriendGamertags`. Les deux migrations
  restent idempotentes.

### Critères de succès

1. Un utilisateur standard (non admin, sans groupe) se connecte via un lien d'invitation sur
   instance verrouillée, crée son profil, gère sa liste d'amis, et ne voit que lui-même.
2. Aucun code ne lit plus `AppSettings.FriendGamertags` (champ supprimé, garde-rail grep).
3. `is_with_friends`, Escouade, vue match, rencontres, Prestige escouade utilisent la liste du
   **profil consulté**, pas une liste d'instance.
4. Toutes les surfaces citées fonctionnent pour un utilisateur standard (plus aucune dépendance
   à `GET /settings` pour les amis).

---

## 2. Étape 0 — Préparation (rapide)

- [~] 0.1 Créer le worktree + la branche : depuis `LevelUp-go-migration`,
      `git worktree add ../LevelUp-wt-amis-invitations -b wt/amis-invitations feat/v75`.
      Basculer TOUTES les commandes suivantes dans ce worktree (chemin absolu, règle mémoire
      « fusion en chemin absolu »).
- [x] 0.2 Lire `.ai/thought_log.md` (10 dernières entrées), `docs/adr/0029-*.md`, skills
      `arch-rules`, `plan-execution`, `frontend-patterns`.
- [x] 0.3 Baseline : `cd apps/go-api && go test ./internal/authz/... ./internal/service/...
      ./internal/api/... ./internal/platform/... ./internal/sync/...` → vert, noter la durée.
      `cd apps/web && Remove-Item -Recurse -Force node_modules\.tmp; npm run typecheck` → vert.

**Gate G0** : worktree sur `wt/amis-invitations` (`git branch --show-current`), baselines
vertes notées dans le journal de phase (§8).

---

## 3. Étape 1 — Backend : store d'amis par joueur + migration (moyen)

Périmètre fermé :

- [ ] 1.1 `internal/domain/friends.go` (nouveau) : type `PlayerFriends{XUID string;
      Gamertags []string (nullable:"false"); UpdatedAt string}` + requête
      `PutFriendsRequest{Gamertags []string}`. Normalisation : trim, dédoublonnage
      insensible à la casse, exclusion du propre gamertag du profil, `≤ 50` entrées,
      `≤ 50` caractères chacune (mêmes bornes que `setup.go:134`).
- [ ] 1.2 `internal/domain/title/registry.go` : `PathResolver.PlayerFriendsPath()` →
      `data/global/player_friends.json` (voisin de `xbox_aliases.duckdb`). Aucun
      `filepath.Join(..., "data", ...)` hors PathResolver.
- [ ] 1.3 `internal/platform/friendstore/friend_store.go` (nouveau, miroir de
      `platform/groupstore/group_store.go` : mutex, `load/save` atomique, nil → slice vide) :
      `NewFriendStore(path)`, `Get(xuid) ([]string, error)` (absent → vide, pas d'erreur),
      `Set(xuid, gamertags []string) error`, `All() (map[string][]string, error)`.
      Fichier ≤ 300 L, fonctions ≤ 80 L.
- [ ] 1.4 `internal/platform/friendstore/migrate.go` : `MigrateFromAppSettings(appSettingsPath
      string, players []domain.PlayerSummary) (created int, err error)` — lit LUI-MÊME
      `app_settings.json` (`PathResolver.AppSettingsPath()`) en `map[string]json.RawMessage`
      et n en extrait que la clé `friend_gamertags` : **zéro dépendance au champ typé**
      `AppSettings.FriendGamertags` (supprimé en 2.4 ; `AppSettings.raw` est privé,
      `store.go:121`). Idempotent : no-op si `player_friends.json` existe déjà (même contrat que
      `groupstore.MigrateDefault`). Pour chaque profil
      avec xuid non vide et `!AuthOnly` : liste = globale moins le gamertag du profil
      (insensible à la casse).
- [ ] 1.5 Câblage au boot dans `cmd/server/main.go` (à côté de la construction de
      `groupStore`) : construire `friendStore`, appeler `MigrateFromAppSettings` UNE fois.
      Câblé une fois, jamais retouché ensuite. `slog.InfoContext` avec `created`.
- [ ] 1.6 Tests `friend_store_test.go` (Get absent → vide ; Set/Get ; normalisation ;
      fichier corrompu → erreur explicite) et `migrate_test.go` (idempotence, exclusion de
      soi, AuthOnly ignoré).

**Gate G1** : `cd apps/go-api && go test ./internal/platform/friendstore/... ./internal/domain/...
&& go vet ./internal/platform/friendstore/...` → code de sortie 0. La migration est prouvée par `migrate_test.go` sur un `app_settings.json`
de fixture — **aucun démarrage de serveur depuis le worktree** (son `data/` est une jonction
vers les mêmes bases que le serveur principal : deux process = violation mono-process ADR 0013).

---

## 4. Étape 2 — Backend : rebrancher tous les lecteurs, supprimer le champ global (lourd)

Tous les sites ci-dessous ont été listés par `grep -rn FriendGamertags apps/go-api` le
2026-09-15. Les resolvers deviennent **par xuid** ; les signatures `func(ctx) []string` sont
conservées quand une fermeture par joueur suffit (le service connaît déjà son xuid).

- [ ] 2.0 **Pré-requis D6 (utilisateur)** : tranché par le pilote le 2026-09-15 — G2.0 NON
      validée (existence de `data/auth/groups.json` en prod non confirmée) → variante D6 :
      migration de groupe conservée et re-sourcée depuis le friendstore. Consigné dans Avancement.
- [ ] 2.1 `internal/api/wire/registry_pages_home.go:258-270` `friendGamertagsResolver()` →
      prend le xuid du joueur résolu (`pdb.XUID`) et lit `friendStore.Get(xuid)` ; tous les
      appelants du registre passent le xuid (Home squad, Career encounters, SessionPage usage,
      squadagg — `grep -n friendGamertagsResolver internal/api/wire/`).
- [ ] 2.2 `internal/sync/engine.go:70` `FriendsLoader func() ([]string, error)` : signature
      inchangée ; les trois câblages deviennent des fermetures sur le xuid de l'engine :
      `internal/scheduler/auto_sync_engine.go:57-63`, `cmd/server/sync_v2_wiring.go:317-324`,
      `internal/api/handlers/sync_handler.go:182-190`.
- [ ] 2.3 `internal/service/friends_orchestrator_service.go` : `FriendsGamertagsLoader` devient
      `func(xuid string) ([]string, error)` ; `RecomputeAll` résout par joueur ; ajouter
      `RecomputeForPlayer(ctx, titleSlug, gamertag, xuid)` (utilisé à l'étape 3). Câblage
      `internal/api/server_apiv1.go:478-484`.
- [ ] 2.4 Suppression du champ global : `internal/domain/settings.go:44` et `:135` ;
      `internal/platform/settings/store.go:58`, `:416-417`, `:508` ; handler
      `internal/api/handlers/settings.go:78-82` (commentaire), `:294-335` (snapshot + diff +
      déclenchement orchestrator sur PATCH → **supprimé**, remplacé par le déclenchement par
      joueur de l'étape 3), `:623-632` (`handlePostRecalculateSessions` lit le store par
      joueur dans la boucle). La migration 1.4 n est pas concernée (elle lit le fichier).
- [ ] 2.5 `cmd/levelup/cmd_recompute_friends.go` : loader par xuid ; `--dry-run` affiche la
      liste par joueur ; doc d'en-tête mise à jour.
- [ ] 2.6 `cmd/server/main.go` : migration de groupe par défaut **conservée** (variante D6),
      re-sourcée depuis `friendStore.Get(xuid de l'admin)` au lieu de `settings.FriendGamertags`,
      et ordonnée APRÈS `friendstore.MigrateFromAppSettings`. `groupstore.MigrateDefault` et son
      test restent en place. Le garde-rail 2.8 tolère cette lecture (elle passe par friendstore,
      sans le littéral `FriendGamertags`).
- [ ] 2.7 Commentaires devenus faux : `internal/api/middleware/require_player_ownership.go:28-31`
      (« FriendGamertags résolus » → co-membres de groupe), `internal/platform/duckdb/queries_career.go:321`,
      `internal/port/services.go:33`, `internal/service/career_service.go:64,130-133`,
      `internal/service/career_service_encounters.go:139`, `internal/service/squadagg/equipment_usage.go:15`.
- [ ] 2.8 Garde-rail : `internal/archlint/no_global_friend_gamertags_test.go` — interdit les
      littéraux `FriendGamertags` et `friend_gamertags` dans `apps/go-api/` hors
      `platform/friendstore/` (migration) et le test lui-même ; allowlist VIDE sinon.
- [ ] 2.9 `openapi.yaml` régénéré puis `make generate-types` : le champ `friend_gamertags`
      disparaît de `AppSettings` / `PatchSettingsRequest` côté Go ET dans `generated.ts`.
- [ ] 2.10 Tests touchés à corriger (pas à supprimer) : `settings_test.go`,
      `friends_orchestrator_service_test.go`, `auto_sync_engine_test.go`, tout test citant le
      champ (`grep -rln friend_gamertags apps/go-api --include=*_test.go`).
- [ ] 2.11 Côté web, le minimum pour que la branche reste compilable après 2.9 (le reste du
      front est l étape 4) : retirer `lib/api/types.ts:407`, `features/settings/SyncTab.tsx:162-163`,
      `test/handlers.ts:169` ; livrer `features/friends/queries.ts` avec `usePlayerFriends(slug)`
      branché sur `GET /players/{slug}/friends` (l endpoint n existe qu en étape 3 : la requête
      est désactivée `enabled: false` et rend `[]` jusqu à 3.1, activation en 4.1) ; basculer
      dès maintenant les quatre lecteurs `SquadLayout`, `MatchViewPage`,
      `PrestigeSquadProgress`, `AddFriendFlow` sur ce hook. Pas de TODO, pas de champ fantôme.

**Gate G2** : `cd apps/go-api && go build ./... && go test ./... && go vet ./...` → 0 ;
`go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` → 0 (sync touché ;
`-p 1` non négociable) ; `grep -rn "FriendGamertags\ ; `cd apps/web && Remove-Item -Recurse -Force
node_modules.tmp; npm run typecheck` → 0 (la branche reste verte côté web à chaque étape).|friend_gamertags" apps/go-api --include=*.go
| grep -v friendstore | grep -v archlint` → vide.

---

## 5. Étape 3 — Backend : API amis par joueur (moyen)

- [ ] 3.1 `internal/api/handlers/friends.go` (nouveau, Huma) monté **sous le groupe
      `/players/{player_slug}`** gardé par `RequirePlayerOwnership`
      (`server_apiv1.go:~949`) : `GET /players/{slug}/friends` → `PlayerFriends` (+ `can_edit: bool`, vrai pour le
      propriétaire direct ou un admin — c est la seule source du mode lecture seule côté front) ;
      `PUT /players/{slug}/friends` → `PlayerFriends` (remplacement complet, normalisation 1.1).
- [ ] 3.2 Droit d'écriture (D4) : `PUT` exige propriétaire direct OU admin — réutiliser
      `BootstrapService.DirectOwnerFor(sess)` (`internal/service/bootstrap_ownership.go`) ;
      sinon **403 `friends_forbidden`**. Enforcement désactivé (demo / `auth_mode=none`) →
      écriture libre (cohérent avec `authz.Enforced`).
- [ ] 3.3 Effet de bord du `PUT` : `FriendsOrchestrator.RecomputeForPlayer` (2.3) en
      goroutine, journalisé `slog.InfoContext(ctx, "friends: recompute is_with_friends",
      "player", ..., "added", n)` ; erreur → `slog.ErrorContext` (jamais avalée). Émettre la
      notification `friend_sync_completed` existante si promotion > 0 (parité avec l'ancien
      hook `settings.go:312-330`).
- [ ] 3.4 Tests `friends_test.go` (httptest, mode `xbox` enforced) : étranger → 403
      `player_forbidden` (middleware) ; co-membre → 200 en GET, 403 `friends_forbidden` en PUT ;
      propriétaire → 200/200 ; admin → 200/200 ; PUT invalide (51 entrées, gamertag vide) →
      400 ; mode `none` → PUT libre.
- [ ] 3.5 Contrat : `openapi.yaml` régénéré (harness `internal/api/openapigen`), opérations
      `getPlayerFriends` / `putPlayerFriends`, tag `friends`.

**Gate G3** : `go test ./internal/api/... ./internal/service/...` → 0 ; `make generate-types`
→ `apps/web/src/lib/api/generated.ts` contient `PlayerFriends` ; `git diff --stat` ne touche
que le périmètre 3.x.

---

## 6. Étape 4 — Frontend : amis par joueur, page « Amis et groupes » (moyen)

Consommateurs listés par `grep -rn "friend_gamertags\|friendGamertags" apps/web/src` le
2026-09-15.

- [ ] 4.1 `lib/query/keys.ts` : `queryKeys.playerFriends(slug)`. `features/friends/queries.ts`
      (créé en 2.11) : activer `usePlayerFriends(slug)`, ajouter `useUpdatePlayerFriends(slug)` (invalide
      `playerFriends`, squad, prestige, match-view).
- [ ] 4.2 Lecteurs déjà sur `usePlayerFriends` depuis 2.11 — vérifier le slug passé (joueur actif) et nettoyer les commentaires :
      `features/squad/SquadLayout.tsx:339-344`, `features/match-view/MatchViewPage.tsx:105`,
      `features/ascension/PrestigeSquadProgress.tsx:74` (+ commentaires `:2,:8`),
      `features/friends/AddFriendFlow.tsx:60-90` (PUT au lieu de PATCH /settings ; doc
      d'en-tête `:5,:8`). Les props `friendGamertags` des composants match-view
      (`MatchViewTabPlayers`, `MatchFragDiffChart`, `MatchKillDistanceSection`) ne changent pas.
- [ ] 4.3 Retirer le champ de Réglages : `features/settings/SyncTab.tsx:162-163` (+ le
      composant de sélection s'il n'a plus d'autre usage — vérifier par grep avant de
      supprimer), `lib/api/types.ts:407` et le commentaire `:1381`, `test/handlers.ts:169`,
      `lib/players/displayName.ts:85` (commentaire).
- [ ] 4.4 Page `/groups` → « Amis et groupes » : `features/groups/GroupsPage.tsx` gagne une
      section « Amis de {gamertag} » en tête, pour le **joueur actif du shell**
      (`activePlayerSlug` de `appShellStore`, le même que MatchView et Escouade — la page
      n est pas scopée joueur dans l URL, c est LA source du slug, tranché) : liste + ajout par
      gamertag via le flux existant `AddFriendFlow` + retrait ; lecture seule si l utilisateur
      n est pas propriétaire direct du profil actif (`can_edit` de la réponse GET, posé en 3.1 — le front n interprète jamais
      un 403 pour décider de l affichage).
- [ ] 4.5 Libellés FR + EN (`Record<Locale, T>`, manifeste `lib/i18n/manifests/common.toml`
      → `common.groups.*` et nouvelles clés `common.friends.*`) ; `lib/pageTitle.ts:126`
      (« Amis et groupes » / « Friends and groups ») ; lien `features/settings/SettingsPage.tsx:176`
      ; entrée de navigation existante vers `/groups` (`grep -rn "/groups" components/shell`).
      Pas d'anglicisme, pas de couleur hex/Tailwind (`color-tokens`).
- [ ] 4.6 Tests : `AddFriendFlow.test.tsx` adapté (PUT) ; test du hook `usePlayerFriends`
      (msw dans `test/handlers.ts`) ; test de rendu lecture seule / édition de la section amis.

**Gate G4** : `cd apps/web && Remove-Item -Recurse -Force node_modules\.tmp; npm run typecheck
&& npm run lint && npm run test` → 0 ; `grep -rn "friend_gamertags" apps/web/src` → vide hors
`generated.ts` (qui ne doit plus le contenir non plus après 2.9). Vérification navigateur (dev
`make dev`) : en tant qu'admin, page Amis et groupes → ajout/retrait d'un ami → la vue match
recolore, l'Escouade présélectionne.

---

## 7. Étape 5 — Backend : invitation sans groupe + droit de provisioning (moyen, à risque : auth)

- [ ] 5.1 `internal/domain/user.go` : `User.ProvisionGrant string json:"provision_grant,omitempty"`
      (code d'invitation ayant créé le compte ; vidé après usage). `InviteCode.GroupID` reste
      optionnel (déjà le cas).
- [ ] 5.2 `internal/platform/userstore/store.go` : `CreateFromXbox(gamertag, xuid)` inchangé +
      `SetProvisionGrant(username, code string) error` (vide = effacer). Test.
- [ ] 5.3 `internal/service/xbox_auth_service.go` : `resolvePendingInvite` (`:207-227`) ne
      rejette plus `GroupID == ""` ; `redeemGroupInvite` (`:229+`) devient `redeemInvite` :
      **consomme toujours** le code (`invites.Consume(code, xuid)`), ajoute au groupe seulement
      si `GroupID != ""`, pose `ProvisionGrant = code` sur le compte **uniquement si le compte
      vient d'être créé** (cas `ErrUserNotFound`). Journal `slog.InfoContext` distinct pour les
      deux formes. Doc d'en-tête `:43-52`, `:71-75`, `:134-142` mise à jour (« rejoindre un
      groupe » → « invitation »).
- [ ] 5.4 `internal/api/handlers/setup.go:119-126` : verrou effectif ET `user.ProvisionGrant
      != ""` ET aucun profil `db_profiles` ne porte `user.XUID` → laisser passer ; après création
      réussie, `SetProvisionGrant(username, "")` (échec → `slog.ErrorContext`, la création reste
      acquise). Le contrôle « xuid = identité liée » existant (`:107-108`) reste la vraie
      barrière. Résolution de l'utilisateur : `authz.CurrentUser(sess, users)` — injecter
      `authz.UserLookup` + setter dans `SetupHandler`.
- [ ] 5.5 `internal/api/handlers/auth_xbox_oauth.go:131-143` : inchangé fonctionnellement ;
      corriger le commentaire (« rejoindre un groupe » → « invitation »). `domain/session.go:57-61`
      idem.
- [ ] 5.6 `internal/api/handlers/admin.go:186-214` `handleGenerateInvite` : conservé, corps
      `{expires_in_days}` ; retourne aussi `join_url` relatif `/join?invite=CODE` pour que le
      front n'assemble pas le lien. Supprimer la mention « flow password legacy » de
      `apps/web/src/features/auth/queries.ts:86-88` à l'étape 6.
- [ ] 5.7 Tests `xbox_auth_service_test.go` : (a) verrouillé + invitation sans groupe valide →
      compte créé, aucun groupe, code consommé, `ProvisionGrant` posé ; (b) verrouillé + sans
      invitation → `ErrInstanceLocked` (ratchet existant conservé) ; (c) invitation de groupe →
      groupe rejoint + `ProvisionGrant` posé ; (d) compte EXISTANT + invitation → pas de grant.
      `setup_test.go` : verrouillé + grant + pas de profil → 201 puis grant vidé, second appel
      → 403 `instance_locked` ; verrouillé sans grant → 403 ; grant mais profil déjà présent →
      403.
- [ ] 5.8 `docs/adr/0029-multi-user-player-ownership.md` : section « Extensions » complétée
      (invitation sans groupe, droit de provisioning porté par le compte) — ADR EN-only, pas de
      traduction.

**Gate G5** : `go test ./internal/service/... ./internal/api/handlers/... ./internal/platform/userstore/...`
→ 0 ; `go test ./internal/platform/auth/...` (sentinelles ADR 0023 intactes) → 0 ; aucun
allowlist de `auth/sentinel_test.go` ni `no_legacy_source_used_test.go` modifié
(`git diff --stat -- '*sentinel*' '*no_legacy*'` vide).

---

## 8. Étape 6 — Frontend : invitation admin + page Rejoindre générique (rapide)

- [ ] 6.1 `features/admin/management/AdminManagementPage.tsx` : section « Inviter un joueur »
      (sans groupe) : bouton « Générer un lien », affichage du lien `origin + join_url`,
      copie presse-papiers, liste des invitations actives (`GET /admin/invites`, révocation).
      Mutations dans `features/auth/queries.ts` (déjà `useAdmin*`) ; retirer le commentaire
      `:86-88`.
- [ ] 6.2 `features/auth/JoinPage.tsx` : libellés génériques (« Rejoindre LevelUp » ; le texte
      de groupe n'apparaît que si le code porte un groupe — le front ne le sait pas : garder un
      texte neutre : « Connectez-vous avec Xbox pour accepter l'invitation »). Doc d'en-tête
      `:2-5`.
- [ ] 6.3 Après login d'un invité sans profil sur instance verrouillée, le Setup
      (`features/setup/SetupPage.tsx`, état `halo_linked_no_profile`) doit désormais réussir le
      `POST /setup/players` : vérifier qu'aucun garde front ne masque l'étape quand
      `instance_locked` est vrai (`grep -rn instance_locked apps/web/src/features/setup`).
- [ ] 6.4 i18n FR + EN pour toutes les nouvelles chaînes (`manifests/admin.toml`, `common.toml`).

**Gate G6** : typecheck (cache purgé) + lint + vitest → 0. Recette navigateur : **arrêter
d abord le serveur principal** (`taskkill //IM air.exe //F //T`, vérifier que le port 8000 est
libre — un seul process writer par DB), puis `make dev` depuis le worktree, avec
`instance_locked: true` dans `app_settings.json` : (1) admin génère un lien ; (2) navigation
privée → `/join?invite=…` → SSO Xbox avec un compte de test → compte créé sans groupe ; (3)
Setup → création du profil → 201 ; (4) le sélecteur de joueur ne montre que ce profil ; (5)
`/players/JGtm/...` → 403 ; (6) page Amis et groupes → ajout d'un ami → OK. Remettre
`instance_locked` à sa valeur d'origine.

---

## 9. Étape 7 — Clôture

- [ ] 7.1 Gate complet : `cd apps/go-api && go test ./... && go vet ./...` ;
      `go test -tags=integration -p 1 ./...` (sync/persist touchés) ; front : typecheck cache
      purgé + lint + test. Codes de sortie vérifiés, pas la sortie filtrée.
- [ ] 7.2 Revue adversariale (skill `adversarial-review`) sur le diff complet — auth et sync
      sont touchés. Constats corrigés ou consignés.
- [ ] 7.3 `.ai/thought_log.md` : entrée `[YYYY-MM-DD]` Complété (décision D1-D6, résultats des
      gates, recette).
- [ ] 7.4 Docs bilingues touchées si une commande change (`docs/COMMANDS.md` pour
      `recompute-friends`) — FR et EN dans le même commit.
- [ ] 7.5 CI de la branche verte (`gh run list --branch wt/amis-invitations`). Merge dans
      `feat/v75` **uniquement sur signal de l'utilisateur** ; jamais dans `main`.

Journal de phase : section « Avancement » en fin de ce fichier (date, étape, gate, constats).
Reprise de session : lire cette section puis `git log --oneline -10` dans le worktree.

---

## 10. Découvertes hors périmètre (ne pas traiter ici)

- `middleware/require_player_ownership.go:28-31` : la doc de `FamilyXUIDResolver` parle encore
  de `FriendGamertags` alors que le câblage réel est `groupstore.CoMemberXUIDs`
  (`server.go:88-94`). Traitée en 2.7 car dans le périmètre ; notée ici parce qu'elle a failli
  induire une mauvaise réponse en conversation.
- `data/auth/users.json` local ne contient qu'un compte (admin). Les profils amis n'ont jamais
  été des utilisateurs : la recette G6 exige un second compte Xbox de test.
- `GET /settings` sous `RequireAdmin` : d'autres réglages lus par le front pour un utilisateur
  standard peuvent souffrir du même 403 (à auditer séparément — `grep -rn useSettings apps/web/src`).
- `cmd_sync.go` : `sync-delta/sync-full --gamertag <GT>` exigent le token DU joueur alors que
  `--all` construit un pool (`buildCLITokenPool`, `:367`). Le chemin mono-joueur pourrait
  emprunter au pool comme le serveur — hors périmètre ici, à noter au BACKLOG.
- Variante D6 si `groups.json` absent en prod : garder `MigrateDefault` mais sourcer les
  membres depuis `friendstore.All()` du propriétaire admin ; à trancher à G2.0.

---

## Annexe A — Procédure « Nuzzles » (à la main de l'utilisateur, hors plan de code)

Ordre recommandé, indépendant des étapes 1-7 sauf mention :

1. **Profil** : ajouter `Nuzzles` dans `db_profiles.json` (bloc `halo_infinite`, avec `db_path`
   `data/titles/halo_infinite/players/Nuzzles/stats.duckdb`, son `xuid`, `waypoint_player`).
   Aucun code n'écrit ce fichier pour un tiers (`/setup/players` ne crée que le profil de
   l'identité liée en session).
2. **Sync des bases** : passer par le **serveur** (bouton de sync / auto-sync), pas par la CLI
   `sync-full --gamertag Nuzzles` : la CLI exige le refresh token DE Nuzzles
   (`cmd_sync.go:414-436`), alors que le serveur emprunte au pool de tokens
   (`sync_handler.go:526,603`, profils `auth_only`). Confirmé par l utilisateur le 2026-09-15 :
   le token de Nuzzles n est jamais nécessaire tant que ceux du pool fonctionnent — la lecture
   de l historique et des films d un tiers passe par n importe quel token valide. Idem pour la
   cuisson : `backfill-replay` ne touche pas à l API, il ne lit que le cache de films local.
3. **Films et rejeux** : les étapes post-sync téléchargent les films et cuisent les artefacts
   selon `replay_build_location` (`sync/convergence.go:614`). Pour forcer les 200 derniers :
   `levelup backfill-replay --dry-run` puis `--limit 200` (sélection « les moins chers
   d'abord » sur TOUT le cache non cuit, pas par joueur : vérifier au `--dry-run` que seuls
   des films de Nuzzles sont candidats). Serveur arrêté, en tâche de fond avec journal (mémoire
   « recuisson »).
4. **Groupe** : aucun. Ne pas l'ajouter à « Mon foyer ».
5. **Connexion** : après l'étape 5 du plan, lien d'invitation admin sur instance verrouillée ;
   avant, déverrouiller le temps de sa première connexion SSO (son profil existant, le Setup
   n'a rien à créer). Il ne verra que lui.
6. **Prod** : release = copie des bases et des rejeux (mémoire « release copie BDD ») ;
   `data/auth/*.json` et `data/global/player_friends.json` partent avec.

---

## Avancement

### Étape 0 — Préparation — 2026-09-15 ~20:50-21:20 — CLOSE

- 0.1 `[~]` : worktree `LevelUp-wt-amis-invitations` et branche `wt/amis-invitations`
  (base `feat/v75` @ 2ddef392c) déjà créés par le pilote ; vérifié
  `git branch --show-current` = `wt/amis-invitations`.
- 0.2 `[x]` : `CLAUDE.md`, plan, ADR 0029, 8 dernières entrées `.ai/thought_log.md`, skills
  `plan-execution` et `arch-rules` lus (skills `frontend-patterns` / `color-tokens` invoqués
  à l'entrée de l'étape 4, leur domaine).
- 0.3 `[x]` : baselines.
  - `go test ./internal/authz/... ./internal/service/... ./internal/api/... ./internal/platform/... ./internal/sync/...`
    → **code de sortie 1**, durée 22 min 29 s. UN SEUL échec :
    `--- FAIL: TestGetMatchFilm_ParallelDownloadFasterThanSequential` (`internal/sync/haloclient`,
    wall-time 515 ms > seuil 500 ms). Test de PERFORMANCE sensible à la charge (quatre binaires
    de test en parallèle) : rejoué isolément → `ok levelup/go-api/internal/sync/haloclient 1.713s`,
    **code de sortie 0**. Baseline retenue : verte hors ce flake, hors périmètre.
  - `cd apps/web && rm -rf node_modules/.tmp && npm run typecheck` → **code de sortie 0**.

**Gate G0 : PASSÉ** (branche correcte, baselines notées).

**Écart consigné (pilote, 2026-09-15)** : G2.0 NON validée — l'existence de
`data/auth/groups.json` en prod n'est pas confirmée. La **variante D6** s'applique : la
migration de groupe par défaut est conservée et re-sourcée depuis `friendstore`. D6, 2.0 et
2.6 réécrits en conséquence dans ce plan.
