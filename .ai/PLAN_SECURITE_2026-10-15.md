# Plan de correction sécurité — 2026-10-15

Source : `.ai/AUDIT_SECURITE_SURFACE_EXPOSEE_2026-10-15.md` (registre validé par l'utilisateur le
2026-10-15). Contrat d'exécution : skill **`plan-execution`** (fait foi), complété par
`delivery-checklist` à chaque clôture et `adversarial-review` à la fin.

## 0. Cadre

**Objectif** : fermer les 12 P0 et 17 P1 du registre, et les P2 listés ci-dessous, avec un
garde-rail par classe de défaut, sans dépendance lourde ni refonte du modèle de session.

**Critère de succès** : chaque item statué ; chaque lot a son test rouge devenu vert et son
garde-rail ; revue adversariale finale sans P0/P1 ouvert ; `make gate-push` vert ; CI de la branche
verte au niveau job.

**Branche / worktree** : `feat/securite-2026-10-15` dans `../LevelUp-wt-securite` (base
`origin/feat/v75` @ `a9a9ac78a`). Livraison : fusion dans `feat/v75` **sur accord de
l'utilisateur**, puis release v7.5 (annoncée imminente par l'utilisateur le 2026-10-15 : pas de
correctif direct sur `main`).

**Décisions tranchées (utilisateur, 2026-10-15)** :
- D-A : accès famille/groupe en **lecture seule** ; toute écriture = propriétaire direct ou admin.
- D-B : import OpenSpartan **ouvert aux comptes, limité aux matchs du joueur qui importe** ;
  aucune écriture de donnée propre à un autre joueur (alias, profil). Risque résiduel accepté :
  un joueur peut falsifier un match auquel il a lui-même participé (voir §4, L3-9).
- D-C : `/healthz/home` réservé à l'admin (conservé pour le monitoring).
- D-D : protection nginx appliquée sur le VPS le 2026-10-15 (voir §5) ; le dépôt est réaligné en L1.

**Décisions de conception tranchées par ce plan** (non rediscutables en exécution) :
- IP cliente : `chimiddleware.ClientIPFromHeader("X-Real-IP")` (chi v5.3.2) remplace `RealIP` ;
  clé du limiteur = `chimiddleware.GetClientIP(ctx)` ; nginx écrase `X-Real-IP` (déjà le cas).
- Corps : plafond par défaut de **1 Mio** posé par le wrapper `humacore` sur **toute** opération
  qui n'a pas de `MaxBody` explicite ; exceptions explicites seulement.
- Pagination : plafond serveur **200** pour `leaderboard` ; règle générale au garde-rail L2.
- Sessions anonymes (sans utilisateur authentifié) : TTL **24 h** au lieu de 7 jours ; taille
  d'une session sérialisée plafonnée à **8 Kio**.
- Échecs de connexion : compteur en mémoire par identifiant normalisé ET par IP ; au-delà de
  **5 échecs en 15 min** → 429 pendant 15 min ; réinitialisé au succès.
- Device-flow : au plus **20** tentatives en attente au total et **3** par IP → 429.
- Envoi de médias : au plus **2** envois en vol et **1** ffmpeg à la fois (sémaphores globaux).

**Intégré sur décision de l'utilisateur (2026-10-15)** : changement de mot de passe exigeant
l'ancien (L4-11) ; réglages illisibles → instance verrouillée en mode appliqué, ADR 0035 amendé
(L4-12) ; correction des 121 `err.Error()` renvoyés en 4xx, contrôle à zéro (L9-5) ; lot L10
(privilèges des conteneurs testés d'abord sur la démo, limites mémoire **après mesure** des pics
réels, ouvrier de rejeu). Seul report : les limites mémoire attendent la mesure (délai
d'observation, report valide).

## 1. Règles d'exécution

1. **Ordre strict** L0 → L4 → L4b → L5 → L5b → L6 → … → L10. Un lot est **clos** quand : gate exact passé, tous les items statués,
   plan mis à jour (cases + journal), commit `fix(securite-Lx): …`, entrée `.ai/thought_log.md`,
   point d'étape à l'utilisateur.
2. **Statuts** : `[x]` fait et vérifié · `[~]` couvert par (référence) · `[!]` non traité
   (justification au journal). Aucune case vide à la clôture.
3. **Test rouge d'abord** : chaque correctif commence par un test qui échoue sur le code actuel ;
   la preuve du rouge (commande + sortie) va au journal du lot.
4. **Vérifier sur pièces** : rouvrir chaque `fichier:ligne` du registre avant de coder (les
   numéros datent du 2026-10-15).
5. **Zéro correctif hors périmètre** : toute découverte va en §6 « Découvertes », non traitée.
6. **Un exécutant à la fois** (builds Go concurrents interdits). Caches :
   `GOCACHE=%LOCALAPPDATA%\go-build-securite`, `GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-securite`.
   Modèle : Opus pour L1, L3, L4, L6, L7 (routage, accès, chemins, CSRF) ; Sonnet pour L0, L2, L5,
   L8, L9, L10. Le worktree est neuf : `npm ci` dans `apps/web` au premier lot qui touche le web
   ou `npm` (pas de jonction vers un autre `node_modules`).
7. **Interdits** : push/merge sur `main` ; fusion dans `feat/v75` sans accord ; opération VPS sans
   annonce ; `git stash` ; `git add -A` ; garde-rail affaibli sans justification datée ; `t.Skip`
   non justifié.
8. **Reprise de session** : relire `plan-execution`, puis le journal §7 ; reprendre à la première
   case non statuée du lot courant.
9. **Non-régression fonctionnelle** : tout item qui change un comportement visible (liste §4bis)
   commence par relever les usages réels du front et des sondes (`grep` dans `apps/web/src`,
   `scripts/`, `docker-compose.yml`) et consigne au journal que l'usage légitime reste servi ;
   les suites complètes existantes du paquet touché passent à chaque lot, pas seulement les tests
   neufs.

Gate commun de fin de lot (en plus du gate propre) depuis `apps/go-api` :
`go build ./cmd/server/ && go vet <paquets touchés> && golangci-lint run <paquets touchés>` ;
si l'API change : `make generate-types` puis `go test ./internal/api/ -run 'OpenAPI|Contract'` ;
si `apps/web` change : purge `node_modules/.tmp`, `npm run typecheck && npm run lint && npm run test`.

## 2. Lots

### L0 — Chaîne d'outils, dépendances et CI (C9-1, C9-2, C9-3) — Sonnet — effort S

- [x] L0-1 `apps/go-api/go.mod` : `go 1.26.9` (commentaire d'en-tête mis à jour : raison = 19
      vulnérabilités atteintes, cf. registre).
- [x] L0-2 `Dockerfile` : image de build épinglée `golang:1.26.9-bookworm` (fin de l'image flottante).
- [x] L0-3 `golang.org/x/image` → v0.45.0 (`go get` + `go mod tidy`).
- [x] L0-4 `apps/web/package.json` : `overrides` `postcss-selector-parser` `^7.1.6` ;
      `npm audit --omit=dev` à 0.
- [x] L0-5 Job CI bloquant dans `.github/workflows/ci.yml` : `govulncheck ./...` (Go du `go.mod`)
      + `npm audit --omit=dev --audit-level=high`.
- [x] L0-6 Épinglage par SHA (commentaire de version) de `appleboy/ssh-action` (3 usages,
      `deploy.yml`) et `softprops/action-gh-release` (`release.yml`).
- [x] L0-7 `release.yml` : `permissions: contents: read` au niveau workflow, `contents: write`
      sur le seul job de publication.
- [x] L0-8 `scripts/git-hooks/lefthook/govulncheck.sh` : commentaire périmé corrigé (le hook
      reste indicatif, la CI devient l'autorité).
- [x] L0-9 Garde-rail : `internal/archlint/ci_workflows_security_test.go` — (a) tout `uses:` hors
      `actions/` dans un workflow qui lit `secrets.` ou a `contents: write` est épinglé par SHA de
      40 caractères ; (b) `ci.yml` contient un job `govulncheck` sans `|| true`.
- Gate : `cd apps/go-api && go version && govulncheck ./...` (aucune vulnérabilité **appelée**) ;
  `go test ./internal/archlint/ -run 'Workflow|CI'` ; `cd apps/web && npm audit --omit=dev` ;
  `go build ./cmd/server/`.

### L1 — IP cliente et nginx du dépôt (C1-1, C1-2 garde locale) — Opus — effort M

- [x] L1-1 Test rouge : pile transverse réelle (`server.go`) — une requête portant
      `True-Client-IP` et `X-Forwarded-For` forgés et `X-Real-IP` réel a pour clé de limiteur le
      `X-Real-IP` ; 301 requêtes à `True-Client-IP` variable → 429.
- [x] L1-2 `server.go:625-626` : `ClientIPFromHeader("X-Real-IP")` si `TrustProxyHeaders`, sinon
      `ClientIPFromRemoteAddr` ; `RealIP` retiré.
- [x] L1-3 `middleware/rate_limit.go` : clé = `GetClientIP(ctx)` (`httprate.KeyByIP` retiré).
- [x] L1-4 `handlers/replay_local_gate.go` : `isLocalRequest` lit l'adresse préservée par
      `PreserveRemoteAddr` (accesseur exporté de `middleware/loopback.go`) ; test existant
      `replay_local_gate_test.go` réécrit pour passer par la pile réelle.
- [x] L1-5 Échecs de connexion : compteur par identifiant normalisé et par IP sur
      `POST /auth/login` et `POST /auth/password` (décision §0) ; test rouge « 6e échec → 429,
      même en changeant d'IP pour un même identifiant ».
- [x] L1-6 Journaux : l'attribut `ip` des journaux d'auth et de CSRF vient de `GetClientIP`.
- [x] L1-7 `packaging/nginx/levelup.conf` et `demo.conf` réalignés sur la configuration appliquée
      au VPS le 2026-10-15 (voir §5) ; `rate_limit.conf` versionné sous `packaging/nginx/`.
- [x] L1-8 Garde-rails : `internal/archlint/client_ip_test.go` interdit `chimiddleware.RealIP`,
      `httprate.KeyByIP` et toute lecture de `r.RemoteAddr` hors `middleware/loopback.go` et
      `middleware/slog_logger.go` (allowlist datée) ; `internal/archlint/nginx_conf_test.go` exige
      dans chaque `server` 443 versionné : `proxy_set_header True-Client-IP ""`,
      `server_tokens off`, aucun `client_max_body_size` > 1m hors des `location` d'envoi
      nommées, un `limit_req` sur `/api/` (prod).
- Gate : `go test ./internal/api/... ./internal/archlint/ -run 'ClientIP|RateLimit|Replay|Nginx|Login|Transverse'`
  puis `go test ./internal/api/...`.

### L2 — Bornes des entrées (C2-1, C2-2, C2-3) — Sonnet — effort M

- [x] L2-1 Test rouge : corps de 2 Mio sur `POST /auth/login`, `/auth/register`,
      `/session/context`, `/auth/password` → 413.
- [x] L2-2 `humacore` : plafond par défaut de 1 Mio sur toute opération enregistrée sans
      `MaxBody` explicite ; commentaire faux de `humacore/operation.go:70` corrigé.
- [x] L2-3 Garde-rail : test qui parcourt toutes les opérations Huma enregistrées
      (`humacore` / OpenAPI) et exige `MaxBodyBytes > 0`.
- [x] L2-4 `handlers/session_context.go` : `locale` ∈ {fr, en} ; `player_slug` ≤ 64 caractères,
      alphabet de slug ; sinon 400 (le contrôle d'accès du slug est L3-1).
- [x] L2-5 `platform/session/store.go` : `Save` refuse une session sérialisée > 8 Kio (erreur
      journalisée) ; TTL de 24 h pour une session sans utilisateur authentifié.
- [x] L2-6 Test rouge leaderboard : `limit=1000000000` → réponse bornée à 200 entrées, aucune
      allocation proportionnelle à `limit` ; `service/leaderboard_service.go` plafonne à 200 ;
      `make` du repo dimensionné sur `min(limit, 200)`.
- [x] L2-7 Garde-rail pagination : test qui parcourt l'OpenAPI et exige `maximum` sur tout
      paramètre entier `limit`/`page_size`/`count`/`n` ; liste datée des paramètres en chaîne
      (dont `leaderboard.limit`), chacun couvert par un test de plafond.
- Gate : `go test ./internal/api/... ./internal/platform/session/... ./internal/service/ -run 'Leaderboard|Session|MaxBody|Pagination|BodyLimit'`
  puis `go test ./internal/api/... ./internal/platform/session/...`.

### L3 — Identité du demandeur et accès aux données d'autrui (C4-1 à C4-8, D-A, D-B, D-C) — Opus — effort L

- [x] L3-1 Test rouge puis correctif : `POST /session/context` refuse (403) un `player_slug` que
      `authz.CanAccessPlayer` n'autorise pas pour l'utilisateur courant.
- [x] L3-2 Suppression et likes de médias : le demandeur et le liker sont dérivés du profil
      possédé par `authz.CurrentUser` (xuid), plus jamais de `SessionPlayerSlug` ; test « média
      d'autrui » × {supprimer, liker} → 403.
- [x] L3-3 Association média → match : contrôle de propriétaire (règle de `CanDeleteMedia`) ;
      recherche par chemin complet ; le repli `file_name` n'est conservé que s'il est restreint au
      joueur (vérifier les appelants) ; test « média d'autrui » × associer → 403.
- [x] L3-4 D-A : `RequirePlayerOwnership` ne transmet l'ensemble famille que si
      `!IsMutatingRequest(r)` ; vérifier `squadActorGuard` (Prestige) sur la même règle ; test
      « co-membre × toutes les routes mutantes du routeur » (walk chi) → 403.
- [x] L3-5 Amendement de l'ADR 0029 (EN) : section « Group/family access is read-only (2026-10-15) ».
- [x] L3-6 Provisionnement : `guardLinkedXboxIdentity` pour tout non-admin quel que soit
      `profile_mode` ; `ProfileService.CreatePlayer` refuse (409) de changer le xuid d'un profil
      existant hors admin ; test table-driven modes × non-admin ; vérifier la précondition avec
      `instance_locked=true` (état prod) et consigner.
- [x] L3-7 Défi d'escouade : `assertMemberUser` dans `CreateSquadChallenge` ; commentaire faux de
      `api/prestige_player_slug_mw.go:26-29` corrigé ; garde-rail AST « toute méthode du service
      Prestige qui reçoit un `squadID` appelle `assertMemberUser` ».
- [x] L3-8 D-C : `/healthz/home` sous `RequireAuth` + `RequireAdmin` ; commentaire faux corrigé ;
      e2e/harnais démo vérifiés (auth inactive → toujours joignable).
- [x] L3-9 D-B import OpenSpartan : n'écrire que les matchs où le xuid de la session participe ;
      aucun `UpsertXUIDAlias` pour un xuid ≠ propriétaire (alias tiers : `INSERT OR IGNORE`
      seulement, jamais d'`UPDATE`) ; `match_id` validé (hexadécimal et tirets) ; test fixture
      « xuid tiers seul » → 0 ligne écrite, « alias tiers existant » → inchangé. Risque résiduel
      (falsification d'un match auquel l'importeur a participé) consigné au journal.
- [x] L3-10 Couche B : contrôle de participation (404 `match_not_participant`) sur `/replay`,
      `/positions`, `/objective-events`, `/events` et la séquence tactique (rejeu public en prod) ;
      test table-driven sur toutes les routes `/matches/{match_id}/*`.
- [x] L3-11 Garde-rails : `internal/archlint/session_player_slug_test.go` interdit
      `SessionPlayerSlug(` dans `handlers/`, `service/`, `domain/` hors affichage (allowlist
      vide au départ) ; ratchet « tout paramètre `player`/`player_slug`/`xuid`/`gamertag` d'une
      route hors `/players/{player_slug}` exige propriété ou admin ».
- Gate : `go test ./internal/api/... ./internal/authz/... ./internal/service/... ./internal/prestige/... ./internal/openspartan/... ./internal/archlint/`
  ; `go test -tags=integration -p 1 ./internal/persist/... ./internal/sync/...` (import OpenSpartan
  touche l'écriture partagée) ; OpenAPI régénéré si `/healthz/home` change de sécurité.

### L4 — Session et authentification (C5-1 à C5-5, C5-6 partiel) — Opus — effort M

- [x] L4-1 Test rouge : device-flow terminé avec `ErrInstanceLocked` → session non authentifiée,
      tentative `failed`.
- [x] L4-2 `handlers/auth.go` : `AuthReady`, identité et jetons écrits seulement après le succès
      de `OnAuthSuccess`.
- [x] L4-3 Invitations : émission (`POST /groups/{id}/invites`) et création de groupe exigent un
      compte existant du magasin d'utilisateurs ; une invitation n'est valable que si son émetteur
      existe encore ; `Consume` avant `Create` (usage unique atomique).
- [x] L4-4 Logout : suppression de la session côté serveur + cookie expiré ; test « SSO →
      déconnexion → lecture refusée ».
- [x] L4-5 `RequireAdmin` relit le rôle via `authz.CurrentUser` (magasin injecté) ; test
      « rétrograder puis rejouer la session → 403 » ; idem `settings.go:216`, `media_delete.go:88`.
- [x] L4-6 Rotation de l'identifiant de session aux trois connexions (mot de passe, OAuth,
      device-flow) : nouvel identifiant, ancienne session supprimée ; test « la valeur du cookie
      change ».
- [x] L4-7 Configuration : `LEVELUP_AUTH_MODE` et `LEVELUP_REGISTRATION` normalisés
      (minuscules, espaces) ; valeur inconnue → refus au démarrage ; test table-driven.
- [x] L4-8 Callback OAuth : jeton de rafraîchissement persisté seulement après le contrôle du
      verrou (`auth_xbox_oauth.go:209` après la décision).
- [x] L4-9 Premier compte : rôle décidé dans `Create` sous verrou ; garde « dernier admin » sur
      `Delete`/`SetRole` ; test concurrent.
- [x] L4-10 Journaux : identifiant de connexion tronqué (64 caractères), codes d'invitation
      retirés des journaux ; garde-rail grep interdisant `"code"` dans les attributs slog de
      `handlers/{admin,groups,user_auth}.go`.
- [x] L4-11 `POST /auth/password` exige le mot de passe actuel quand le compte en a déjà un
      (compte Xbox sans mot de passe : premier mot de passe sans l'ancien) ; front : champ
      « Mot de passe actuel » / « Current password » (i18n FR + EN), test du hook ou du composant ;
      tests serveur des deux cas ; libellés de `SetPasswordCard` passés en i18n FR + EN (D-L1-3). Dépendance posée par L1-5 : `handleSetPassword` consulte déjà
      le limiteur d'échecs (`loginAttempts`) ; un mot de passe actuel incorrect y appelle
      `RecordFailure(*sess.Username, ip)`, un correct `RecordSuccess` (test « 6e mot de passe
      actuel faux → 429 »).
- [x] L4-12 `authz.InstanceLocked` : en mode appliqué, une erreur de lecture des réglages
      verrouille (au lieu d'ouvrir) ; erreur journalisée ; test `authz_test.go` inversé ;
      amendement de l'ADR 0035 (EN) consignant le basculement et sa raison.
- Gate : `go test ./internal/api/... ./internal/authz/... ./internal/platform/session/... ./internal/platform/userstore/... ./internal/service/... ./internal/config/...`.

### L4b — Compléments décidés en cours d'exécution — Opus — effort S

Décisions de l'utilisateur du 2026-10-16 (questions nées de L3) et découvertes de L4 relevant de la doctrine D5.

- [x] L3-12 (D-L3-8 adjacent, décision du 2026-10-16) Like d'un média depuis le profil d'un ami
      **autorisé** : action du visiteur, attribuée à son profil possédé (résolveur de L3-2) ; la route
      de like est l'exception nommée et testée à la garde « famille en lecture seule » ; front :
      cœur actif sur le profil d'un ami. Test : co-membre like → 200, like attribué au co-membre.
- [x] L3-13 (D-L3-8, décision du 2026-10-16) Notifications **réservées à leur propriétaire**, lecture
      comprise : la garde famille ne donne pas accès aux routes de notifications (GET inclus) ;
      front : aucune cloche, page ni toast de notifications sur le profil d'un ami. Test :
      co-membre × routes de notifications → 403.
- [x] L4-13 (D-L4-4, doctrine D5) Changer son mot de passe révoque les autres sessions du compte
      (la session courante est conservée, après rotation) ; test « deux sessions, changement depuis
      l'une → l'autre refusée ».
- [x] L4-14 (D-L4-3) Refus du callback OAuth : redirection vers l'écran de connexion avec un code
      d'erreur, message FR + EN, au lieu d'un JSON brut ; test du callback et du composant.
- [x] (D-L4-2, vérifié par le superviseur le 2026-10-16) `LEVELUP_REGISTRATION` absent du
      `.env.local` du VPS : valeur par défaut, démarrage non bloqué. Rien à faire. Recopié et confronté au code
      le 2026-10-16 : `config.go:244` lit la variable avec un défaut vide, `normalizeMode` (`config_auth.go:32-37`)
      ramène le vide au défaut (`invite`), que `validateModes` accepte.
- Gate : `go test ./internal/api/... ./internal/authz/... ./internal/platform/session/... ./internal/platform/userstore/...`
  ; web (`typecheck`, `lint`, `test:run`).

### L5 — Travaux coûteux (C3-1 à C3-6) — Sonnet — effort L

- [x] L5-1 Device-flow : plafonds (§0) dans `attempt_store` → 429 ; test « 21e tentative → 429 ».
- [x] L5-2 `/api/v1/assets/*` : identifiant validé contre le catalogue local (titre, médaille,
      carte…) avant tout fetch, sinon 404 sans appel ; cache négatif borné (TTL, taille) ;
      limiteur sortant du provider Halo partagé (plus recréé par requête) ; test « id inconnu →
      404, fetcher espion jamais appelé ».
- [x] L5-3 Exclusion de match : le recalcul n'acquiert plus le writer partagé (lecture seule de la
      base partagée, écritures en base joueur via le persister existant) ; no-op si l'état ne
      change pas ; single-flight par joueur ; test « exclusion identique → aucun recalcul »,
      « recalcul → writer partagé jamais pris ».
- [x] L5-4 Import OpenSpartan : validation du fichier avant toute bascule ; un import à la fois par
      compte (`FindActiveJob`, comme `backfill.go:111`) → 409 ; test « fichier invalide → writer
      non pris », « 2e import simultané → 409 ».
- [x] L5-5 Envoi de médias : écriture en flux vers le disque (fin de `io.ReadAll`) ; sémaphores
      globaux (§0) ; tâche ffmpeg avec délai maximal.
- [x] L5-6 Recherche de gamertags : `live` ignoré sans session authentifiée ;
      `POST …/engagement/recompute_coefficients` sous `RequireAdmin`.
- [x] L5-7 Garde-rails : chaque entrée de `publicRoutesAllowlist` (`bare_routes_ratchet_test.go`)
      porte une classe `cheap` / `outbound` / `disk`, et `outbound`/`disk` exigent un test
      « anonyme → aucun appel sortant ni écriture non bornée » ; archlint interdisant
      `AcquireWriter`/`AcquireSharedWriterStandalone` dans un chemin déclenché par une route
      `/players` hors allowlist datée ; grep interdisant `io.ReadAll` sur un `multipart.File`.
- Gate : `go test ./internal/api/... ./internal/assets/... ./internal/service/... ./internal/platform/auth/... ./internal/archlint/`
  ; `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/platform/duckdb/...`.

### L5b — Compléments issus de L5 — Sonnet — effort S

- [ ] L5b-1 (D-L5-3, régression du chantier : `writable_player_slugs` introduit en L3-4)
      `TestBootstrapBuild_WritablePlayerSlugs` instable (11 échecs sur 20 relevés par le
      superviseur le 2026-10-16) : ordre de sortie rendu déterministe dans le code (tri), pas dans
      le test seul ; preuve : `-count=50` vert.
- [ ] L5b-2 (D-L5-1, même classe que C3-3) `PUT /players/{slug}/friends` : `RecomputeIsWithFriends`
      ne prend plus le writer partagé si ses écritures vont en base joueur (sinon : justification
      écrite et coalescence par joueur comme L5-3) ; entrée correspondante retirée de l'allowlist du
      garde-rail `no_shared_writer_on_request_path_test.go` ; tests rouge puis vert.
- [ ] L5b-3 (D-L5-2) Autres usages de ffmpeg (miniatures, remux WebM, migration audio) : même place
      unique (`platform/slots`) et délai maximal que L5-5 ; garde-rail : tout `exec.CommandContext`
      de ffmpeg passe par la place (test grep/AST).
- [ ] L5b-4 (D-L5-4, vérifié par le superviseur) `batchComputeLUSR` est aussi le chemin du
      post-sync normal (`engine_postsync_scoring.go:196`) : le recalcul d'exclusion suit le même
      chemin que la synchronisation ; rien à faire. Cocher `[x]` en recopiant cette vérification.
- Gate : `go test ./internal/service/ -run TestBootstrapBuild_WritablePlayerSlugs -count=50` ;
  `go test ./internal/api/... ./internal/service/... ./internal/media/... ./internal/archlint/` ;
  `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` si L5b-2 touche le sync.

### L6 — Fichiers et chemins (C6-1, C6-2, C6-3) — Opus — effort M

- [ ] L6-1 Route des médias : racine legacy restreinte à `players/{slug}/captures/` ; liste
      blanche des extensions média ; test table-driven `.duckdb`, `.wal`, `.json` → 404 sur chaque
      racine.
- [ ] L6-2 Slug : la garde du groupe `/players/{player_slug}` refuse `.`, `..`, `/`, `\` et tout
      slug hors alphabet, en démo comprise.
- [ ] L6-3 `match_id` : validation fail-closed (hexadécimal et tirets) à la frontière HTTP du rejeu
      et dans les constructeurs de chemin du `PathResolver` qui l'utilisent (rejeu, faits de film,
      raster tactique) ; test « le chemin reste sous sa racine » avec `/`, `\`, `..`, valeur longue.
- [ ] L6-4 `/static/` : plus de listing de répertoire (404).
- [ ] L6-5 Envoi : `http.DetectContentType` doit concorder avec l'extension, sinon 415.
- Gate : `go test ./internal/api/... ./internal/domain/title/... ./internal/service/...`.

### L7 — Gardes de chemin et CSRF (C10-1) — Opus — effort S

- [ ] L7-1 Test rouge : `POST …/coach/proposals/..%2F..%2F..%2F..%2Finternal%2Fx/dismiss` avec
      Origin étrangère → 403 `csrf_rejected` (pile réelle).
- [ ] L7-2 `middleware/csrf.go` : `!AmbiguousPath(r) &&` devant `isExemptPath`.
- [ ] L7-3 Garde-rail : test table-driven qui passe les mêmes chemins ambigus (`%2F`, `%5C`,
      `%2e%2e`, `//`, `/./`) à toutes les gardes qui décident sur un chemin (CSRF, démo,
      écritures, limiteur, rejeu, ouvrier).
- [ ] L7-4 (D-L1-2) `handlers/replay_local_gate.go` : `isLocalRequest` refuse un hôte vide (garde
      fermée par défaut) ; tests adaptés à un transport qui fournit une adresse.
- Gate : `go test ./internal/api/... -run 'CSRF|Ambiguous|Transverse'` puis `go test ./internal/api/...`.

### L8 — Démo publique (C8-1, C8-2) — Sonnet — effort M

- [ ] L8-1 En démo, `RequireAdmin` et `RequireAuth` répondent 403 sauf allowlist nommée
      (`GET /settings`, et ce que l'interface de démo lit réellement, à vérifier) ; `/debug/vars`
      403 en démo ; ratchet qui parcourt le routeur de démo et exige 403 sur toute route portant
      `RequireAdmin` hors allowlist datée.
- [ ] L8-2 Identités : témoin disque adossé à `DemoLayout`/`DemoRoster`.
- [ ] L8-3 CLI : six lectures `cfg.DemoMode` → `cfg.demoState()` ; garde-rail grep interdisant
      `.DemoMode` dans `internal/config/` hors `config_demo.go`.
- [ ] L8-4 `suggest-next` classé lecture (`readOnlyPostPrefixes`, motif par suffixe) ; ratchet démo
      mis à jour.
- [ ] L8-5 E2E `media-like-bug.spec.ts` : cas « like » marqué hors démo (ou retiré s'il n'a pas
      d'autre cible).
- [ ] L8-6 `weapon_kills` conditionnée par titre dans la liste d'extraction ; `seed_demo.go:988`
      n'avale plus l'erreur (journalisée).
- [ ] L8-7 ERROR au boot sur la metadata Halo 5 : requête fautive identifiée
      (`title_asset_drawer_loader.go`) et corrigée (absence attendue → pas d'ERROR, ou table créée
      par la fixture).
- [ ] L8-8 Séquence Tactique : `ReplayGate` lit `match_id` dans le corps pour cette route ; test
      garde démo + route.
- [ ] L8-9 `docker-compose.yml` : `LEVELUP_TRUST_PROXY_HEADERS=true` sur le service démo ;
      garde-rail lisant le compose.
- [ ] L8-10 `netguard.Check` dans `platform/auth/xbox_device_code.go` ; raison fausse de
      `netguard_coverage_test.go:43` retirée.
- [ ] L8-11 `data/demo_films/` : statut `[!]` attendu — provisionnement sur le VPS au déploiement
      de la v7.5 (geste annoncé, commande au runbook) ; vérifier que le runbook est à jour.
- [ ] L8-12 (D-L1-4) `packaging/nginx/demo.conf` : `limit_req_status 429;` (la limite répond 503, lu comme une panne) ; appliqué sur le VPS par le superviseur (opération annoncée) ; garde-rail nginx étendu à la démo.
- Gate : `go test ./internal/api/... ./internal/config/... ./internal/ops/... ./internal/platform/...` ;
  `cd apps/web && npx playwright test e2e/media-like-bug.spec.ts --list`.

### L9 — Fuites d'information (C7) — Sonnet — effort M

- [ ] L9-1 `/health` public réduit à `{status}` (sondes Docker et `deploy.sh` vérifiées) ; détail
      réservé à l'admin.
- [ ] L9-2 `/readyz` et le 503 de `/health` : plus d'`err.Error()` dans le corps.
- [ ] L9-3 Device-flow : `error_detail` retiré de la réponse (code seul) ; front adapté (FR + EN).
- [ ] L9-4 Jobs et notifications de sync : message fixe + code, détail au journal.
- [ ] L9-5 Les 121 `err.Error()` renvoyés au client en 4xx (`NewError`/`writeError`) remplacés
      par un message fixe par code d'erreur (helper unique), le détail partant au journal ;
      relevé préalable des messages que le front affiche (`apps/web/src`, `ApiError.message`) pour
      garder un texte utile ; garde-rail `internal/archlint/err_error_response_test.go` à **zéro**
      (allowlist datée seulement pour un cas justifié par écrit).
- [ ] L9-6 En-tête CSP minimal sans risque pour le SPA : `object-src 'none'; base-uri 'self';
      frame-ancestors 'none'` (scripts non restreints, décision de l'ADR à venir) ; test sur `/`,
      404, 429, `/static`.
- [ ] L9-7 (D-L4b-1, doctrine D5) Réinitialisation du mot de passe d'un compte par l'admin
      (`handlers/admin.go`) : révoque toutes les sessions de ce compte (`Store.RevokeAccount`) ;
      test « session du compte réinitialisé → 401 ».
- [ ] L9-8 (D-L4b-2) `GET /auth/xbox/login` : refus redirigés vers `/login?oauth_error=<code>`
      comme le callback (L4-14), plus de JSON brut ; test.
- [ ] L9-9 (D-L4b-3) Refus d'une reconnexion Xbox depuis une session déjà connectée : le message
      n'est plus perdu (`__root.tsx`) ; test du composant.
- Gate : `go test ./internal/api/... ./internal/archlint/ -run 'Health|Ready|ErrError|CSP|SecurityHeaders|DeviceFlow'` ;
  front si L9-3.

### L10 — Conteneurs et ouvrier de rejeu (C9-4) — Sonnet — effort S

- [ ] L10-1 `docker-compose.yml` : `security_opt: [no-new-privileges:true]`, `cap_drop: [ALL]`,
      `cap_add` limité au strict nécessaire de l'entrée (`gosu` : `CHOWN`, `SETUID`, `SETGID`,
      plus ce que la mesure montre) sur les deux services ; image construite et démarrée en local
      si Docker est disponible (démarrage, `/health`, écriture sous `data/`), sinon `[!]` et
      validation à faire sur le conteneur de démo du VPS par le superviseur (opération annoncée)
      avant la prod.
- [ ] L10-2 Garde-rail : test lisant `docker-compose.yml` qui exige `no-new-privileges` et
      `cap_drop: [ALL]` sur chaque service, et `LEVELUP_TRUST_PROXY_HEADERS` sur chaque service
      publié derrière nginx (couvre L8-9).
- [ ] L10-3 Limites mémoire : relevé des pics réels (`memory.peak` des cgroups des deux conteneurs
      sur le VPS, lu par le superviseur à la clôture puis après une semaine) ; `mem_limit` fixé à
      partir de la mesure. Statut attendu `[!]` (délai d'observation) jusqu'à la mesure.
- [ ] L10-4 Ouvrier de rejeu : `scripts/deploy-worker.sh` installe l'unité par copie appartenant à
      root (fin du lien vers un fichier modifiable par `deploy`) et refuse un lien ; unité
      durcie (`ProtectSystem=full`, `ProtectHome=read-only`, chemins d'écriture déclarés) ;
      `packaging/nginx/levelup.conf` : `location /api/v1/internal/` restreint à l'IP du serveur
      de calcul (`allow …; deny all;`), appliqué sur le VPS à l'activation de l'ouvrier.
- [ ] L10-5 (D-L0-3, décision de l'utilisateur du 2026-10-15) `.github/dependabot.yml` : écosystème `docker` (répertoire racine, images du `Dockerfile`), même cadence que les autres.
- [ ] L10-6 (D-L0-3, décision de l'utilisateur du 2026-10-15) Garde-rail archlint : la version de l'image `golang:X.Y.Z-…` du `Dockerfile` est égale à la ligne `go` de `apps/go-api/go.mod` ; vu rouge sur une divergence simulée.
- Gate : `go test ./internal/archlint/ -run 'Compose|Nginx|Dockerfile'` ; `bash -n scripts/deploy-worker.sh`
  ; `shellcheck scripts/deploy-worker.sh` si disponible ; `systemd-analyze verify` si disponible.

## 3. Clôture du plan

- [ ] F-0 Recette locale par profil, application lancée sur les données locales (`make dev`,
      navigateur piloté sur accord de l'utilisateur) : admin, compte ordinaire sur son profil,
      compte ordinaire sur le profil d'un ami, visiteur anonyme, démo ; pages Accueil, Matchs,
      Vue match, Rejeu, Médias (envoi, like, association), Classement, Escouades, Réglages,
      connexion/déconnexion. Aucune régression ; captures au journal.
- [ ] F-0b `npx playwright test` local (E2E complets) vert, en plus de la CI.
- [ ] F-1 Revue adversariale du diff complet (`git diff a9a9ac78a...HEAD`) : 2 relecteurs aveugles
      (lentille « contrôle d'accès et identité » ; lentille « entrées, ressources, chemins ») +
      1 relecteur L6 (couverture des tests) ; 2 rondes maximum.
- [ ] F-2 `make gate-push` vert.
- [ ] F-3 Push de `feat/securite-2026-10-15`, CI verte au niveau job (`gh run list --branch`).
- [ ] F-4 Registre et backlog mis à jour (items fermés, `[sécurité]` et `[demo]` du backlog).
- [ ] F-5 Demande d'accord à l'utilisateur pour la fusion dans `feat/v75`.
- [ ] F-6 Après le déploiement de la v7.5 (délai d'observation prescrit, report valide) :
      surveillance 48 h des 401/403/413/429 dans les journaux nginx et Go, par route et par IP ;
      tout refus d'un usage légitime = correctif immédiat.

## 4bis. Changements de comportement visibles (à vérifier contre l'usage réel, règle 9)

| Changement | Lot | Vérification d'usage |
|---|---|---|
| Plus d'écriture sur le profil d'un ami (synchro, exclusion, notifications, Prestige, médias) | L3-4 | Le front masque ou désactive ces actions sur un profil non possédé (`bootstrap` distingue possédé / famille) ; i18n FR + EN si un message est ajouté |
| Connexion bloquée 15 min après 5 échecs | L1-5 | Message FR + EN côté écran de connexion pour le 429 |
| Classement plafonné à 200 lignes | L2-6 | Valeur de `limit` envoyée par le front |
| Langue choisie par un visiteur sans compte oubliée après 24 h | L2-5 | Le front garde-t-il aussi la langue côté navigateur |
| Rejeu, positions, événements limités aux matchs du joueur du profil | L3-10 | Liens vers le rejeu présents seulement sur ces matchs (Vue match, Tactique, Escouade) |
| `/healthz/home` réservé à l'admin | L3-8 | E2E et harnais démo (auth inactive) |
| `/health` réduit à `{status}` | L9-1 | Sonde Docker, `deploy.sh`, workflows de déploiement |
| Au plus 2 envois de vidéos en vol | L5-5 | Le front envoie-t-il plusieurs fichiers en parallèle ; un envoi en attente plutôt qu'un refus |
| Device-flow : 3 tentatives simultanées par IP | L5-1 | Rafraîchissement de la page de connexion |
| Un seul import OpenSpartan à la fois par compte | L5-4 | Message FR + EN sur 409 |
| Détail d'erreur du device-flow retiré | L9-3 | Message affiché par `XboxLoginPage` |
| Messages d'erreur 4xx remplacés par des messages fixes | L9-5 | Messages affichés par le front (`ApiError.message`) |
| Mot de passe actuel exigé pour en changer | L4-11 | Écran de changement de mot de passe, comptes Xbox sans mot de passe |
| Réglages illisibles → instance verrouillée | L4-12 | Connexion des comptes existants et de l'admin inchangée |
| Conteneurs sans privilèges superflus | L10-1 | Démarrage, écriture sous `data/`, ffmpeg, cuisson |

## 4. Risques et points d'attention

- L3-9 : un joueur peut falsifier un match auquel il a participé (y compris les lignes des autres
  participants de ce match) et devancer sa version réelle, que l'écriture INSERT-only ne
  remplacera pas. Accepté par D-B ; consigné.
- L5-3 : retirer la bascule de la base partagée suppose qu'aucune écriture n'y a lieu ; vérifier
  avec le test d'intégration et le garde-rail ART (`no_art_patterns_test.go`).
- L1 : le code de `main` (prod actuelle) reste vulnérable jusqu'à la release ; la protection nginx
  du §5 couvre l'IP falsifiable et les corps volumineux d'ici là.
- L0 : `golang:1.26.9-bookworm` doit exister ; sinon `[!]` et épinglage sur la dernière 1.26.x.

## 5. Opérations VPS (faites par le superviseur, annoncées)

- [x] V-1 (2026-10-15) nginx prod et démo : `True-Client-IP` retiré, `server_tokens off`, corps à
      1 Mo sur `/api/` (envoi de vidéos 600 Mo, import 1,1 Go, protocole ouvrier 20 Mo), limite de
      débit `lvl_api` 20 r/s rafale 200 et `lvl_auth` 10 r/min rafale 10 sur connexion,
      inscription, mot de passe, démarrage du device-flow, 429 au dépassement. Sauvegarde :
      `/root/nginx-backup-2026-10-15/`. Vérifié : pages 200, corps de 2 Mo → 413, envoi de vidéo
      non bloqué par nginx, IP forgée absente des journaux (adresse réelle journalisée).
- [x] V-2 (2026-10-15, accord de l'utilisateur) TLS : en pratique TLS 1.0/1.1 étaient déjà
      refusés (le premier `server` du port 443 impose ses protocoles) ; rendu explicite dans
      `nginx.conf` (`ssl_protocols TLSv1.2 TLSv1.3;`, sauvegarde dans le même dossier). Vérifié :
      TLS 1.1 refusé, 1.2 et 1.3 acceptés, prod et démo 200.
- [ ] V-3 Au déploiement de la v7.5 : provisionnement de `data/demo_films/` (L8-11) et contrôle
      que la conf nginx versionnée (L1-7) correspond à la conf en place.

## 6. Découvertes

Non traitées (règle 5). Le lot qui l'a vue est entre parenthèses.

- **D-L0-1** (L0) `npm audit` complet (dev inclus) : 4 alertes d'outillage de build, hors du gate
  (`--omit=dev`) et hors du livrable : `brace-expansion` (high, malgré l'override `^5.0.9` existant :
  plages 2.0.0-2.1.6 et 4.0.0-5.0.11), `source-map-js` (high), `undici` 8.0.0-8.10.1 (high),
  `smol-toml` <=1.8.0 (moderate).
- **D-L0-2** (L0) `golang.org/x/crypto` v0.55.0 : GO-2026-6355 et GO-2026-6354 (`x/crypto/ssh`,
  corrigées en v0.56.0) et GO-2026-5932 (`openpgp`, sans correctif). Module requis, jamais appelé
  (govulncheck : 0 vulnérabilité atteinte, 0 importée) ; tolérées par le gate. Montée en v0.56.0 non
  prévue au plan.
- **D-L0-3** (L0) `.github/dependabot.yml` ne couvre pas l'écosystème `docker` : l'épinglage
  `golang:1.26.9-bookworm` ne se met à jour qu'à la main, et aucun garde-rail ne tient l'égalité entre la
  version de l'image et la ligne `go` de `go.mod` (le build échoue si l'image est plus ancienne, pas si
  elle est plus récente).
- **D-L0-4** (L0) Actions tierces encore sur tag mobile, hors du champ de la règle L0-9 (workflows sans
  `secrets.` ni `contents: write`) : `golangci/golangci-lint-action@v9` et
  `stoplightio/spectral-action@v0.8.13` (`ci.yml`), `docker/setup-buildx-action@v4` et
  `docker/build-push-action@v5` (`test-deploy-precheck.yml`).
- **D-L0-5** (L0) `lefthook.yml:29` range `govulncheck` parmi les checks qui « gatent » la dette globale au
  pre-push ; le hook est indicatif (L0-8) et l'autorité est le job de `ci.yml`.
- **D-L1-1** (L1) Les gardes « requête locale » (`LoopbackOnly` des `/_diag/*`, garde du rejeu) décident sur
  le pair TCP. Cela tient tant que nginx joint le Go par le port publié du conteneur (le conteneur devrait
  voir la passerelle du pont Docker, pas 127.0.0.1 ; non vérifié sur le VPS, hors périmètre). Un déploiement
  où nginx joint le Go par loopback (hors Docker, ou réseau `host`) rendrait « locale » toute requête
  relayée ; aucun garde-rail ne tient cette topologie. **Vérifié le 2026-10-15 par le superviseur** : le VPS utilise le proxy Docker par défaut (`docker-proxy` actif, pas de `userland-proxy: false`) : le conteneur voit la passerelle du pont, pas 127.0.0.1. Sûr en production.

- **D-L1-2** (L1) `handlers/replay_local_gate.go` : `isLocalRequest` traite un hôte vide comme local (choix
  justifié par des transports de test). En service `RemoteAddr` n'est jamais vide, mais le défaut d'une garde
  de sécurité est ouvert.
- **D-L1-3** (L1) `features/auth/SetPasswordCard.tsx` : libellés français en dur (`LABELS`), sans version
  anglaise (l'écran en anglais affiche du français). L4-11 retouche ce composant. **Rattaché à L4-11** (libellés FR + EN du composant). **Traité en L4-11** (2026-10-16).

- **D-L1-4** (L1) `packaging/nginx/demo.conf` (comme la démo en place) n'a pas `limit_req_status 429` : la
  limite `demo_limit` répond 503, que le front lit comme une panne et non comme un excès de tentatives.
- **D-L1-5** (L1) `archlint/no_french_label_literal_test.go` signale une baisse non figée :
  `service/seasons_catalog.go` compte 3 littéraux pour 4 alloués (allowlist à resserrer).

- **D-L3-1** (L3) `features/home/HomePage.tsx` et `components/ui/match-card.tsx` portent une exemption
  `max-lines` datée « le découpage revient au lot qui touchera ce fichier ». L3-4 a touché `HomePage.tsx`
  (2 lignes : étoile favori masquée sur un profil d'ami) sans le découper (hors périmètre).
- **D-L3-2** (L3) L8-1 (démo : `RequireAdmin` répond 403 hors allowlist) doit allowlister `GET /healthz/home` :
  l'e2e `slice-5-home.spec.ts`, `e2e/_helpers/demoData.ts` et `scripts/demo-visual-harness.sh` le sondent en démo
  (inerte aujourd'hui, L3-8).
- **D-L3-3** (L3) Deux copies de la forme d'un identifiant de match : `domain.IsValidMatchID` (L3-9) et
  `sync.isValidMatchID` (`sync/backfill.go:518`). L6-3 étend la validation : centraliser sur la version `domain`
  et poser le garde-rail (règle 6).
- **D-L3-4** (L3) `MatchViewService.assertParticipation` (vue du match) reste ouvert sur erreur de base ; la route
  est désormais précédée de `RequireMatchParticipation`, fermé sur erreur : la vérification du service est
  redondante sur ce chemin (gardée pour les appelants hors HTTP).
- **D-L3-5** (L3) Le rôle admin du demandeur d'une suppression ou d'une association de média vient encore de
  `sess.Role` (`handlers/media_requester.go`, `sessionIsAdmin`) : couvert par L4-5.
- **D-L3-6** (L3) La règle d'écriture propre de la liste d'amis (`friends_forbidden`, `handlers/friends.go`) n'est
  plus atteinte par le routeur (la garde de groupe refuse avant, `player_forbidden`) ; le front mappe encore
  `friends_forbidden` (`features/friends/errors.ts`).
- **D-L3-7** (L3) Un non-admin dont le gamertag est déjà porté par un profil d'un autre xuid (gamertag réattribué
  par Xbox) reçoit 409 `profile_xuid_conflict` à l'onboarding ; le front n'a pas de message dédié (FR + EN).
- **D-L3-8** (L3) Sur le profil d'un ami, ses notifications restent LISIBLES (D-A : lecture) par la page dédiée ;
  la cloche et les toasts sont coupés. Question produit ouverte : une notification est-elle une donnée
  « consultable par un ami » ? **Tranché par l'utilisateur le 2026-10-16 (non) ; traité en L3-13.**
- **D-L3-9** (L3) En-tête de `service/openspartan_import_service.go` périmé (« writes via the existing sync.Insert*
  helpers » : l'écriture passe par `persist.SharedPersister`).

- **D-L4-1** (L4) `service/bootstrap_service.go` : `is_admin` du bootstrap lit encore le rôle copié dans la session
  (affichage du menu d'administration seulement ; toute route d'admin décide par `RequireAdmin`, rôle relu). Seule
  entrée, datée, de l'allowlist de `archlint/session_role_read_test.go`. Un admin rétrogradé voit le menu jusqu'à sa
  reconnexion, chaque action répond 403.
- **D-L4-2** (L4) Valeur de `LEVELUP_REGISTRATION` dans le `.env.local` du VPS non vérifiée (accès VPS interdit au
  lot) : toute valeur hors `invite`/`open`/`closed` (casse ignorée) empêche désormais le démarrage (L4-7). La prod a
  `LEVELUP_AUTH_MODE=xbox` (accepté). À vérifier par le superviseur avant le déploiement de la v7.5. **Vérifié par le
  superviseur le 2026-10-16 (variable absente : défaut) ; statué en L4b.**
- **D-L4-3** (L4) Les refus du callback OAuth (`403 instance_locked`, `500 link_failed`, `code_exchange_failed`…) sont
  des corps JSON rendus tels quels par le navigateur (navigation plein écran, pas un `fetch`) : aucune page d'erreur
  côté front (préexistant pour `link_failed`). **Traité en L4-14.**
- **D-L4-4** (L4) Changer son mot de passe ne ferme pas les autres sessions du même compte (le magasin de sessions
  n'a pas d'index par utilisateur). **Traité en L4-13.**
- **D-L4-5** (L4) `features/auth/XboxLoginPage.tsx` (branche d'échec du device-flow) garde deux libellés français en
  dur (« Le code a expiré. », « Échec de l'authentification. ») et affiche `error_detail` brut : relève de L9-3
  (retrait de `error_detail`, front adapté FR + EN).

- **D-L4b-1** (L4b) La réinitialisation d'un mot de passe par l'admin (`handlers/admin.go:161`,
  `handleResetPassword`) ne révoque pas les sessions du compte visé (L4-13 ne couvre que le changement de son
  propre mot de passe) ; `session.Store.RevokeAccount` existe désormais pour le faire.
- **D-L4b-2** (L4b) `GET /auth/xbox/login` (`LoginRedirect`, navigation plein écran comme le callback) répond encore
  ses refus en JSON brut (`redirect_uri_not_configured`, `no_session`, `state_gen_failed`, `pkce_gen_failed`,
  `session_save_failed`) ; le refus démo du login et du callback reste en JSON (contrat de la garde démo).
- **D-L4b-3** (L4b) Un utilisateur DÉJÀ connecté dont la reconnexion Xbox (bannière `ReauthBanner`, flux OAuth) est
  refusée arrive sur `/login?oauth_error=…`, que `__root.tsx:184` renvoie vers `/` (connecté sur une page
  d'auth) : le message est perdu (la bannière reste affichée). Avant L4b il voyait un JSON brut.

- **D-L5-1** (L5) `service/friends_orchestrator_service.go` : la liste d'amis d'un profil (`/players/{slug}/friends`,
  propriétaire) déclenche `sync.RecomputeIsWithFriends`, qui prend le writer partagé (2 appels) : même classe de défaut que
  C3-3, rejouable en boucle par un propriétaire. Consignée dans l'allowlist de
  `archlint/no_shared_writer_on_request_path_test.go` ; à traiter comme L5-3.
- **D-L5-2** (L5) Les autres usages de ffmpeg n'ont ni place ni délai maximal : miniatures de l'indexation d'un envoi
  (`ops/media_thumbnails.go`), remux WebM en lecture (`media/remux.go`, flux vers le client), migration audio
  (`media/hls_audio_migrate.go`). Aucun garde-rail n'exige la place avant un `exec ffmpeg` (proposé par le registre, hors
  des items L5-1 à L5-7).
- **D-L5-3** (L5) `TestBootstrapBuild_WritablePlayerSlugs/admin` (`service/bootstrap_ownership_test.go:256`) est instable :
  l'ordre de `writable_player_slugs` pour un admin n'est pas déterministe (4 échecs sur 10).
- **D-L5-4** (L5) Le recalcul d'une exclusion relance `batchComputeLUSR` (`skill.BatchComputeLUSRWithMedals`), pas le chemin
  canonique v2 (`RecomputeLUSRCanonicalForPlayer`, qui écrit `player_skill_state_v2` dans la base partagée). Non vérifié :
  si le LUSR v2 lu par l'écran tient compte d'une exclusion, c'est par ailleurs.
- **D-L5-5** (L5) L'import OpenSpartan tient le writer partagé pendant toute son écriture (`Import` après validation) :
  un compte connecté peut occuper le writer le temps d'un import valide (un seul à la fois par compte, mais un par compte).
- **D-L5-6** (L5) `handlers/media_upload.go` renvoie encore `err.Error()` du service en 500 (`upload_error`) : relève de L9-5.

## 7. Journal

### L0 - Chaîne d'outils, dépendances et CI - 2026-10-15 - clos

Base : `git log --oneline -1` = `659cbf18d` avant le lot ; branche `feat/securite-2026-10-15`. Références
`fichier:ligne` du registre rouvertes avant d'agir et conformes : `go.mod:9` (`go 1.26.5`),
`Dockerfile:51` (`golang:1.26-bookworm`), `deploy.yml:262,285,454`, `release.yml:9,121`,
`govulncheck.sh:26`. Session interrompue par une erreur d'API puis reprise : chaque modification a été
re-vérifiée sur les fichiers et les gates rejoués après la reprise.

**Statuts** : L0-1 à L0-9 `[x]`. Aucun `[!]`, aucun `[~]`.

- **L0-1** `apps/go-api/go.mod` : `go 1.26.9` ; en-tête réécrit (contrat : version minimale du toolchain de la
  CI et des binaires de release, doit rester sans vulnérabilité de bibliothèque standard atteinte).
  `go version` dans `apps/go-api` : `go1.26.9 windows/amd64` (téléchargé par `GOTOOLCHAIN=auto`).
- **L0-2** `Dockerfile:55` : `FROM golang:1.26.9-bookworm`. Existence vérifiée avant épinglage :
  `GET https://hub.docker.com/v2/repositories/library/golang/tags/1.26.9-bookworm` -> 200,
  `"name":"1.26.9-bookworm"`. Image non construite localement (le build Docker passe par `scripts/deploy.sh`,
  geste VPS, hors gate du lot). Une image `golang:1.26.x` fixe `GOTOOLCHAIN=local` : une image plus ancienne
  que `go.mod` ne compile pas (voir D-L0-3).
- **L0-3** `golang.org/x/image` v0.24.0 -> v0.45.0 (`go get` + `go mod tidy`) ; diff `go.mod`/`go.sum` limité à
  cette ligne. Paquets importeurs (`internal/mapdecoupe`, `cmd/mapfond-cadrage`, `cmd/mapfond-planche`,
  `cmd/mapfond-webp`) : `go vet` exit 0, `go test` ok.
- **L0-4** `apps/web/package.json` : `overrides."postcss-selector-parser": "^7.1.6"` ; lockfile régénéré
  (`npm ci` puis `npm install`), diff du lockfile = 3 lignes (version, `resolved`, `integrity` de
  `postcss-selector-parser`, 6.0.10 -> 7.1.6). Seul consommateur : `@tailwindcss/typography` (épingle 6.0.10).
  Build Vite avant/après **octet pour octet identique** (`index-Bf8rEaj3.css`, sha256 `5d8170af...87aec`,
  règles `.prose` présentes). `npm audit --omit=dev` : avant 2 alertes moderate (exit 1), après `found 0
  vulnerabilities` (exit 0), y compris sur un dossier ne contenant que `package.json` + `package-lock.json`
  (conditions du job CI).
- **L0-5** `ci.yml` : deux jobs bloquants, inconditionnels, sans échappatoire. `govulncheck` (`setup-go` sur
  `go-version-file`, `go install golang.org/x/vuln/cmd/govulncheck@v1.9.0`, `govulncheck ./...`,
  `CGO_ENABLED=1`, 15 min) et `npm-audit` (`npm audit --omit=dev --audit-level=high`, Node 22, 10 min).
  govulncheck v1.9.0 (publiée le 2026-10-11) installée et exécutée localement sur le module : 0 vulnérabilité
  atteinte, exit 0 ; la v1.3.0 du poste donne le même verdict. Le déploiement (`attente-ci`) suit le verdict
  du workflow entier : ces deux jobs conditionnent désormais le déploiement. La base de vulnérabilités étant
  lue en ligne, une alerte nouvelle sur une dépendance déjà fusionnée rougit la CI sans changement de code
  (voulu, noté en commentaire ; soupape existante : `workflow_dispatch` + `ignorer_attente_ci`). YAML des
  trois workflows relu par un parseur (PyYAML).
- **L0-6** SHA obtenus par `git ls-remote` (tags légers : aucune ligne `^{}`), puis confirmés commits par
  `gh api repos/<repo>/git/commits/<sha>` : `appleboy/ssh-action@0ff4204d59e8e51228ff73bce53f80d53301dee2 # v1.2.5`
  (ce que `v1` résout ; 3 usages de `deploy.yml`) et
  `softprops/action-gh-release@3bb12739c298aeb8a4eeaf626c5b8d85266b0e65 # v2.6.2` (ce que `v2` résout ;
  `release.yml`). Dependabot `github-actions` (mensuel, sans les majeures) maintient ces SHA à partir du
  commentaire de version.
- **L0-7** `release.yml` : `permissions: contents: read` au niveau workflow, `contents: write` sur le seul job
  `publish`. Les jobs `build-go` et `build-web` n'utilisent que `checkout`, `setup-*` et `upload-artifact`.
- **L0-8** `scripts/git-hooks/lefthook/govulncheck.sh` : en-tête et commentaire du `|| true` réécrits (le hook
  est indicatif et ne bloque jamais, y compris si govulncheck échoue à charger les paquets ; l'ancienne
  formulation « bloquant si govulncheck crashe » était fausse, `|| true` absorbe aussi les crashs ; l'autorité
  est le job de `ci.yml`). `bash -n` ok. Aucun changement de comportement.
- **L0-9** `apps/go-api/internal/archlint/ci_workflows_security_test.go` (nouveau, 385 lignes), helpers de
  `ci_deploy_triggers_test.go` réutilisés. `TestSensitiveWorkflowsPinThirdPartyActionsBySHA` (a : dans tout
  workflow lisant `secrets.` ou portant `contents: write`, tout `uses:` hors `actions/` et hors local est
  `owner/repo@<40 hex>` ; ancres : `deploy.yml` et `release.yml` doivent être reconnus sensibles),
  `TestCIHasBlockingVulnerabilityJobs` (b : jobs `govulncheck` et `npm-audit` présents, sans `|| true` /
  `|| exit 0` / `continue-on-error`, sans `if:`, installation de govulncheck à version exacte, commandes
  exigées), `TestNoWorkflowGrantsContentsWriteAtWorkflowLevel` (garde-rail de L0-7),
  `TestWorkflowSecurityAnalysisDetectsDefects` (les analyseurs savent échouer : tag mobile, SHA court /
  majuscule / branche, mutations `|| true`, `continue-on-error`, `if`, `@latest`, job absent, `write-all`).
  Écarts au texte du plan, additifs et dans la classe de défaut : (b) couvre aussi `npm-audit` ; le test de
  `contents: write` est ajouté dans le même fichier.

**Preuve du rouge** (garde-rail écrit AVANT les correctifs L0-5/L0-6/L0-7, exécuté sur l'état d'avant) :
`cd apps/go-api && go test ./internal/archlint/ -run 'Workflow|CI' -count=1` -> `FAIL levelup/go-api/internal/archlint`,
exit 1 :
`--- FAIL: TestSensitiveWorkflowsPinThirdPartyActionsBySHA` (`deploy.yml : ligne 262`, `ligne 285`, `ligne 454` :
« uses: appleboy/ssh-action@v1 » ; `release.yml : ligne 121` : « uses: softprops/action-gh-release@v2 »),
`--- FAIL: TestCIHasBlockingVulnerabilityJobs` (`job "govulncheck" : job absent`, `job "npm-audit" : job absent`),
`--- FAIL: TestNoWorkflowGrantsContentsWriteAtWorkflowLevel` (`release.yml : ligne 9 : contents: write au niveau workflow`).
`TestWorkflowSecurityAnalysisDetectsDefects` est vert dès l'écriture (il ne lit pas les fichiers du dépôt).
Après correctifs, même commande : `ok`, exit 0, les 6 tests verts (dont les 2 tests voisins de
`ci_deploy_triggers_test.go`).

**Gates** (sorties réelles ; rejoués après la reprise) :
- Avant : `govulncheck ./...` sous go 1.26.5 -> 19 `Vulnerability #N` atteintes, exit 3.
- `cd apps/go-api && go version` -> `go version go1.26.9 windows/amd64`.
- `govulncheck ./...` -> « Your code is affected by 0 vulnerabilities », 0 section `Vulnerability #`, exit 0.
  Restent **3 vulnérabilités de modules requis, non appelées**, toutes sur `golang.org/x/crypto@v0.55.0` :
  GO-2026-6355 et GO-2026-6354 (`x/crypto/ssh`, corrigées en v0.56.0), GO-2026-5932 (`x/crypto/openpgp`, sans
  correctif). 0 vulnérabilité importée mais non appelée (voir D-L0-2).
- `go test ./internal/archlint/ -run 'Workflow|CI'` -> ok, exit 0.
- `cd apps/web && npm audit --omit=dev` -> `found 0 vulnerabilities`, exit 0.
- `cd apps/go-api && go build ./cmd/server/` -> exit 0 (`server.exe` supprimé ensuite).
- Gate commun §1 : `go vet ./internal/archlint/` exit 0 ; `golangci-lint run ./internal/archlint/` (v2.12.2) ->
  « 0 issues », exit 0 ; `go test ./internal/archlint/ -count=1` (paquet complet) -> ok en 142,9 s, exit 0.
  L'API ne change pas (pas de `generate-types`). `apps/web` change : purge de `node_modules/.tmp`, puis
  `npm run typecheck` exit 0, `npm run lint` 0 erreur (26 avertissements préexistants), `npm run test:run` ->
  919 fichiers passés / 5 ignorés, 9598 tests passés / 23 ignorés, exit 0.

**Non-régression (règle 9)** : aucun changement visible par un utilisateur ni listé au §4bis. Effets
d'exploitation : (1) le déploiement dépend désormais de deux jobs CI de plus (L0-5) ; (2) `docker compose
build` (sans `--pull`) tire `golang:1.26.9-bookworm` si l'image n'est pas déjà sur l'hôte ; (3) la release
publie avec un jeton en lecture seule sur les jobs de compilation. Aucune opération VPS.

**Découvertes** : D-L0-1 à D-L0-5 (§6), non traitées.

### L1 - IP cliente et nginx du dépôt - 2026-10-15 - clos

Base : `git log --oneline -1` = `cab27c25a` avant le lot ; branche `feat/securite-2026-10-15`. Références du
registre rouvertes avant d'agir et conformes : `server.go:625-626` (`RealIP` sous `TrustProxyHeaders`),
`rate_limit.go:66-67` (`httprate.KeyByIP`), `replay_local_gate.go:86-100` (`r.RemoteAddr`),
`handlers/user_auth.go:126` (aucun compteur), `levelup.conf:85-90` (`2g` sur `/api/`), chi v5.3.2
`middleware/client_ip.go` (`ClientIPFromHeader` garde la dernière valeur, ne réécrit pas `RemoteAddr`).

**Statuts** : L1-1 à L1-8 `[x]`. Aucun `[!]`, aucun `[~]`.

- **L1-1** `internal/api/client_ip_transverse_test.go` (paquet `api`, pile assemblée par
  `applyTransverseMiddlewares`, route sonde sous `/api/`) : `TestTransverseClientIP_HandlersReadXRealIP`,
  `TestTransverseRateLimit_KeyIsXRealIP` (solde `X-RateLimit-Remaining` partagé à X-Real-IP égal, en-têtes forgés
  différents ; seau distinct pour un autre X-Real-IP), `TestTransverseRateLimit_ForgedTrueClientIPDoesNotEscape`
  (301 requêtes, `True-Client-IP` et `X-Forwarded-For` variables : la 301e en 429),
  `TestTransverseClientIP_UntrustedUsesTCPPeer`, `TestTransverseClientIP_TrustedWithoutHeaderUsesTCPPeer`.
- **L1-2** `server.go` (`applyTransverseMiddlewares`) : `RealIP` retiré ; `chimiddleware.ClientIPFromRemoteAddr`
  toujours, puis `ClientIPFromHeader("X-Real-IP")` si `TrustProxyHeaders`. Précision d'implémentation : un
  X-Real-IP absent ou illisible laisse le pair TCP (requête qui n'est pas passée par nginx : sonde locale), au
  lieu d'une IP vide qui mettrait ces requêtes dans un seau commun. `r.RemoteAddr` n'est plus jamais réécrit.
  `server.go` reste à 769 lignes (770 avant, dette gelée non accrue).
- **L1-3** `rate_limit.go` : clé `clientIPKey` = `chimiddleware.GetClientIP(ctx)` ; journal du 429 en `ip`.
- **L1-4** `middleware/loopback.go` : accesseur exporté `PeerAddr(r)` (pair capturé par `PreserveRemoteAddr`,
  repli `r.RemoteAddr` sans ce middleware) ; `LoopbackOnly` et `isLocalRequest` le lisent. Le test
  `handlers/replay_local_gate_test.go` est réécrit dans `internal/api/replay_local_gate_test.go` (même nom,
  paquet `api`) : `ReplayGate(nil)` derrière la pile réelle, avec et sans confiance ;
  `TestReplayLocalGate_AllowsLoopbackPeer`, `TestReplayLocalGate_RejectsRemotePeerWithForgedHeaders`.
- **L1-5** Paquet `internal/platform/loginlimit` (en mémoire, compteurs par identifiant normalisé — minuscules,
  espaces de bord retirés, tronqué à 64 runes — et par IP ; 5 échecs en 15 min glissantes : blocage 15 min ;
  tentative bloquée ni vérifiée ni comptée ; succès : remise à zéro des deux clés ; tables bornées à 10 000 clés,
  éviction de la clé non bloquée la moins récente). `UserAuthHandler` : un limiteur par handler ;
  `POST /auth/login` refuse avant `Authenticate` (ni coût bcrypt ni échec de plus), compte l'échec, remet à zéro
  au succès ; `POST /auth/password` refuse si l'identifiant de la session ou l'IP est bloqué. Refus : 429
  `too_many_login_attempts` + `Retry-After` (secondes). L'échec propre à `/auth/password` (mot de passe actuel
  faux) n'existe qu'avec L4-11 : dépendance consignée sur l'item L4-11. Message serveur technique en anglais
  (le ratchet `TestNoNewFrenchLabelLiteral` refuse un 14e littéral français dans `user_auth.go` ; le front
  affiche son propre texte, D6). Tests : `handlers/user_auth_login_limit_test.go`
  (`TestLoginLimit_SixthFailureIs429EvenAcrossIPs`, `..._SixthFailureFromSameIPIs429AcrossIdentifiers`,
  `..._SuccessResetsCounters`, `..._SetPasswordRefusedWhileIdentifierBlocked`), `loginlimit/limiter_test.go`
  (9 tests : seuil, normalisation, IP multi-comptes, expiration à 15 min, fenêtre glissante, remise à zéro,
  clés vides, troncature, table bornée qui garde une clé bloquée). OpenAPI inchangé : le 429 est couvert par
  la réponse `default` (`ApiError`) ; pas de `generate-types`.
- **L1-6** `ip` = `chimiddleware.GetClientIP` dans : CSRF (attribut ajouté), `require_admin.go`, `require_auth.go`,
  `require_auth_mutations.go`, `require_player_ownership.go` (2), `handlers/build_worker.go`, journal HTTP
  `slog_logger.go` (clé `remote_addr` renommée `ip`, aucun consommateur dans le dépôt), journal du 429, journaux
  de connexion (`user_auth.go`). `slog.Warn`/`Debug`/`Info` sans contexte passés en `...Context` sur ces lignes.
  Tests : `middleware/client_ip_logs_test.go` (`TestClientIPLogs_RejectionsCarryClientIP`, 6 cas : csrf, admin,
  auth, auth-mutations, ownership, http) et `handlers/build_worker_client_ip_log_test.go`.
- **L1-7** `packaging/nginx/levelup.conf` et `demo.conf` : directives de la conf appliquée le 2026-10-15
  (`True-Client-IP ""`, `server_tokens off`, `limit_req_status 429` en prod, locations envoi 600m / import
  1100m / auth 16k + `lvl_auth` / ouvrier 20m / images 16k / `/api/` 1m + `lvl_api`) ; `rate_limit.conf`
  versionné (zones `demo_limit`, `lvl_api`, `lvl_auth`). Comparaison des directives (commentaires et lignes
  vides retirés) entre les fichiers copiés du VPS (`scratchpad/nginx/new/`) et le dépôt : seules diffèrent les
  lignes gérées par certbot (`if ($host = ...)` du bloc 80, `ssl_certificate*` de la prod) ; `rate_limit.conf`
  identique. Commentaires réécrits en contrat (le dépôt ne mentionne plus `RealIP`). `ssl_protocols` vit dans
  `nginx.conf` (non versionné) : pré-requis noté en tête de `levelup.conf`. Docs alignées :
  `RUNBOOK_DEPLOY_CHECKLIST.md` (plus de `2g` sur `/api/`), `RUNBOOK_REPLAY_WORKER.md` §3 (location
  `/api/v1/internal/` 20m) ; `docs/CONFIGURATION.md` + `docs/FR/CONFIGURATION.md` (`LEVELUP_TRUST_PROXY_HEADERS`).
- **L1-8** `internal/archlint/client_ip_test.go` (AST de tout le module hors `_test.go`) : interdit
  `RealIP` (chi middleware, tout alias), `httprate.KeyByIP` et `KeyByRealIP`, et tout accès au champ
  `RemoteAddr` (lecture ou écriture) ; l'appel `conn.RemoteAddr()` n'est pas visé. Allowlist datée :
  `internal/api/middleware/loopback.go` seul (plus étroite que le texte du plan, qui citait aussi
  `slog_logger.go` : le journal HTTP lit désormais `GetClientIP`). `TestClientIPGuardDetectsDefects` prouve
  l'analyse. `internal/archlint/nginx_conf_test.go` + `nginx_conf_parse_test.go` (lecture de la syntaxe, héritage
  des directives, sélection de location) : chaque `server` 443 versionné exige `X-Real-IP $remote_addr` et
  `True-Client-IP ""` effectifs dans toute location qui relaie (héritage `proxy_set_header` pris en compte),
  `server_tokens off`, aucun `client_max_body_size` effectif au-delà de 1m hors des trois locations d'envoi
  nommées (chacune sous son plafond), en prod une `limit_req` et `limit_req_status 429` effectifs sur les
  chemins de l'API ; toute zone citée est déclarée dans `rate_limit.conf` ; tout `.conf` est classé
  (prod/démo). `TestNginxConfAnalysisDetectsDefects` : 15 mutations d'une conf saine, toutes signalées. Ajouts
  au texte du plan, dans la classe de défaut : `X-Real-IP $remote_addr`, `limit_req_status 429`, zones
  déclarées.

**Preuves du rouge** (tests écrits avant les correctifs ; sorties dans le scratchpad de session) :
- L1-1 `go test ./internal/api/ -run 'TestTransverseClientIP|TestTransverseRateLimit' -count=1` : FAIL, exit 1,
  5 tests : IP vue par le handler `""` (want `203.0.113.10`) ; soldes `299` puis `299` (want `298`) ; requête 301 :
  200 (want 429) ; IP `""` sans confiance et sans en-tête.
- L1-4 même test sur le `server.go` de `cab27c25a` (copie temporaire, restaurée, `git diff` vérifié) :
  `TestReplayLocalGate_RejectsRemotePeerWithForgedHeaders` FAIL, pair `203.0.113.7` + `True-Client-IP/X-Real-IP/
  X-Forwarded-For: 127.0.0.1`, confiance active : 200 au lieu de 404 (l'exploitation de C1-2).
- L1-5 `go test ./internal/api/handlers/ -run TestLoginLimit -count=1` : FAIL, exit 1 ; 6e tentative 401
  (want 429) ; 6e tentative depuis la même IP 200 ; changement de mot de passe d'un identifiant bloqué 204.
  Front : `npx vitest run LoginPage.test.tsx XboxLoginPage.test.tsx` : 4 échecs, l'écran affichait
  « Erreur HTTP 429 » (429 de nginx) ou le texte brut du serveur (429 de l'API, même en anglais) ;
  `SetPasswordCard.test.tsx` : 1 échec (« Erreur HTTP 429 »).
- L1-6 `go test ./internal/api/middleware/ -run TestClientIPLogs -count=1` : 6 sous-tests FAIL, `ip` absent
  (csrf ; http avec `remote_addr=127.0.0.1:40000`) ou `ip=127.0.0.1:40000` (admin, auth, auth-mutations,
  ownership), c'est-à-dire ce que montreraient les journaux de prod sans `RealIP`. Test ouvrier sur le
  `build_worker.go` de `cab27c25a` (copie temporaire, restaurée) : `ip = 127.0.0.1:40000`.
- L1-8 `client_ip_test.go` sur un extrait de `cab27c25a` (`git archive` des `.go` dans le scratchpad, garde-rail
  seul dans `internal/archlint/`) : FAIL, 12 violations (`server.go:626 RealIP`, `rate_limit.go:67 KeyByIP`,
  `RemoteAddr` dans `build_worker.go:264`, `replay_local_gate.go:87,88`, `rate_limit.go:112`,
  `require_admin.go:24`, `require_auth.go:40`, `require_auth_mutations.go:83`,
  `require_player_ownership.go:72,85`, `slog_logger.go:72`). `nginx_conf_test.go` sur les `.conf` d'avant L1-7 :
  FAIL, 24 manquements (`rate_limit.conf` absent, `server_tokens` x2, `True-Client-IP` sur 7 locations, zone
  `demo_limit` non déclarée, `2g` sur `/api/`, 6 chemins de l'API sans `limit_req` ni `limit_req_status 429`).
  Après correctifs : tous ces tests verts.

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`) :
- `go test ./internal/api/... ./internal/archlint/ -run 'ClientIP|RateLimit|Replay|Nginx|Login|Transverse' -count=1` :
  exit 0 (`api`, `handlers`, `middleware`, `wire`, `archlint` ok ; `humacore` sans test filtré).
- `go test ./internal/api/... -count=1` : exit 0. Premier passage rouge : `TestNoRetryAfterLiteralInHandlers`
  (littéral `"Retry-After"` dans `user_auth.go`), corrigé par la constante `headerRetryAfter` (son commentaire
  mentionne désormais le 429).
- Suites complètes des autres paquets touchés : `go test ./internal/archlint/ ./internal/config/
  ./internal/platform/loginlimit/ ./cmd/server/ ./internal/api/handlers/ -count=1` : exit 0. Premier passage
  rouge : `TestNoNewFrenchLabelLiteral` (message 429 en français) et `TestConfigGo_TailleGelee` (config.go
  631 > 629), corrigés par un message technique en anglais et des commentaires compactés (628 lignes). Aucun
  garde-rail modifié.
- `go test ./internal/api/ -run 'OpenAPI|Contract' -count=1` : ok (document OpenAPI inchangé).
- Gate commun : `go build ./cmd/server/` exit 0 ; `go vet` sur `internal/api`, `api/middleware`, `api/handlers`,
  `archlint`, `platform/loginlimit`, `config`, `cmd/server` exit 0 ; `golangci-lint run --new-from-rev=cab27c25a`
  (v2.12.2) sur ces paquets : 0 issues ; forme de la CI (`--new-from-merge-base=origin/main`) : 0 issues. (Sans
  ratchet : 38 problèmes, tous préexistants, aucun sur une ligne du lot.)
- Web : purge de `node_modules/.tmp`, `npm run typecheck` exit 0, `npm run lint` 0 erreur (26 avertissements
  préexistants), `npm run test:run` : 921 fichiers passés / 5 ignorés, 9611 tests passés / 23 ignorés, exit 0.

**Non-régression (règle 9)** : usages relevés. `POST /auth/login` : `useLogin` (`features/auth/queries.ts`),
appelé par `LoginPage` et le panneau admin de `XboxLoginPage` ; `POST /auth/password` : `useSetPassword`, appelé
par `SetPasswordCard` ; aucun usage dans `apps/web/e2e`, `scripts/`, `docker-compose.yml` ni `cmd/`. L'usage
légitime reste servi : bon mot de passe 200 (`TestLoginLimit_SuccessResetsCounters`, suite `TestUserAuth_*`),
définition du mot de passe 204 (`TestUserAuth_SetPassword_OptIn`). Changement visible (§4bis) : après 5 échecs,
l'écran de connexion (mode mot de passe et panneau admin du mode Xbox) et la carte « mot de passe » affichent
« Trop de tentatives de connexion. Réessayez dans 15 minutes. » / « Too many sign-in attempts. Try again in 15
minutes. » (`common.auth.too_many_attempts`, manifeste i18n FR + EN régénéré), pour un 429 de l'API comme de
nginx (`features/auth/loginError.ts`, `isTooManyLoginAttempts`, testé). Limite de débit applicative : pour un
client honnête derrière nginx la clé reste X-Real-IP, comme avant (nginx retire `True-Client-IP` depuis V-1).
Clé de journal HTTP renommée `remote_addr` en `ip` : aucun lecteur dans le dépôt. Aucune opération VPS.

**Découvertes** : D-L1-1 à D-L1-5 (§6), non traitées.

### L2 - Bornes des entrées - 2026-10-15 - clos

Base : `git log --oneline -1` = `ad3433eab` avant le lot ; branche `feat/securite-2026-10-15`. Références du
registre rouvertes avant d'agir et conformes : Huma v2.39.1 `huma.go:1454-1460` (défaut posé dans
`processInputType` pour un champ `Body` seulement), `:1118` (`readBody` avec `op.MaxBodyBytes`), `:1513-1518`
(`ensureMaxBodyBytes`) ; `humacore/operation.go:70` (commentaire faux) ; `session_context.go:79-98` ;
`store.go:192` (`save` sans plafond) ; `leaderboard.go:54-60,83`, `leaderboard_service.go:44`,
`leaderboard_world_repo.go:105,634` (`make(..., 0, limit)`). Routes chi nues `/media/upload` et
`/import/openspartan` : `MaxBytesReader` présents (`media_upload.go:35`, `openspartan_import.go:132`), absentes
du document OpenAPI Huma (liste `isKnownChiBrut`) : aucune exception de plafond Huma à poser pour elles.

**Statuts** : L2-1 à L2-7 `[x]`. Aucun `[!]`, aucun `[~]`.

- **L2-1** `handlers/body_limit_test.go` : `TestBodyLimit_RawBodyRoutesRefuse2MiB` (login, register, password,
  session/context -> 413) et `TestBodyLimit_OrdinaryBodyStillServed`.
- **L2-2** `humacore/body_limit.go` (nouveau) : `DefaultMaxBodyBytes` (1 Mio), `applyDefaultMaxBody`,
  `DocumentOperation` sur `apiWithAdapter` et `subrouterAPI`. Mécanisme : `huma.Register` appelle
  `DocumentOperation` sur l'opération ORIGINALE (celle que lit le handler à chaque requête) avant que le
  `huma.Group` et le document n'en fassent des copies (`PrefixModifier` copie) ; poser le défaut dans
  `Adapter.Handle` n'aurait pas atteint l'exécution des sous-routeurs. `NewAPI` (document isolé) est enveloppé
  comme `NewAPIWithConfig`. `humacore.go` : +2 lignes (le code neuf est dans `body_limit.go`, dette de taille
  préexistante non accrue matériellement). Commentaire de `MaxBody` (`operation.go`) réécrit : le défaut de
  1 Mio est celui de `humacore`, non celui de Huma. Tests : `humacore/body_limit_test.go` (trois fabriques x
  {RawBody sans MaxBody -> 413 à 2 Mio, 200 à 1 Kio ; MaxBody explicite de 4 Mio l'emporte ; le document porte
  la valeur effective}).
- **L2-3** `internal/api/body_limit_guard_test.go` : `TestEveryOperationHasBodyLimit` parcourt le document
  partagé (>= 100 opérations) : `MaxBodyBytes > 0` partout, et au-dessus du défaut seulement pour une
  exception nommée de `bodyLimitExceptions` (valeur exacte exigée, entrée périmée refusée).
- **L2-4** `handlers/session_context_validate.go` (nouveau) + appel dans `handlePostContext` AVANT toute
  mutation (`WithSession` persiste la session même quand le handler répond en erreur). `locale` dans {fr, en}
  (liste du front, `KNOWN_LOCALES`), `player_slug` de 1 à 64 runes, lettres/chiffres/marques Unicode et
  ` _-.#`, ni espace de bord ni slug fait de points ; sinon 400 `invalid_locale` / `invalid_player_slug`
  (messages techniques en anglais, ratchet des littéraux français). `title_slug` inchangé (inconnu : ignoré
  en silence, comme avant). Contrôle d'accès du slug : L3-1, non traité ici. Tests :
  `session_context_validation_test.go` (12 refus, aucun fichier de session écrit ; 10 valeurs réelles
  acceptées dont `JGtm`, `Élodie#1234`, 64 caractères).
- **L2-5** `platform/session/limits.go` (nouveau) : `MaxSessionBytes` = 8 Kio, `ErrSessionTooLarge`,
  `AnonymousTTL` = 24 h, `carriesIdentity` (Username, AuthReady, identité Xbox liée ou jetons Halo : lecture
  large de « utilisateur authentifié » pour ne jamais raccourcir une session device-code ou SSO). `save`
  refuse (erreur `slog.Error` avec taille, plafond, drapeau d'authentification, jamais l'identifiant de
  session) avant tout fichier ; `isExpired` utilise `ttlFor` = `min(ttl, AnonymousTTL)` pour une session sans
  identité, donc aussi la purge et `Load`. `store.go` 493 lignes. Tests : `store_limits_test.go` (refus à
  `Save` et `Touch`, rien écrit, journalisé ; session authentifiée avec ~5 Kio de jetons acceptée ; anonyme
  valide à 23 h, expirée à 25 h ; authentifiée valide à 3 jours pour trois formes d'identité ; la purge ne
  retire que l'anonyme).
- **L2-6** `domain.MaxLeaderboardLimit` = 200 ; `LeaderboardService.GetPage` plafonne ; le repo plafonne
  aussi (`boundedLeaderboardLimit`, une seule fonction pour les deux lectures, remplace deux copies de
  `limit <= 0 -> 100`) donc `make(..., 0, limit)` est dimensionné sur au plus 200. Tests :
  `service/leaderboard_service_limit_test.go`, `handlers/leaderboard_limit_test.go` (HTTP, vrai service),
  `platform/duckdb/leaderboard_limit_test.go` (tag `integration`, budget d'allocation 32 Mio pour `limit=1<<20`).
- **L2-7** `internal/api/pagination_guard_test.go` : `TestPaginationParamsAreBounded` (noms `limit`,
  `page_size`, `count`, `n`, `*_limit`) : entier -> `maximum` exigé et égal à la valeur attendue ; chaîne ->
  entrée de `paginationStringParams` (plafond, date, fichier et test cités, existence du test vérifiée,
  entrée périmée refusée). Deux entiers sans `maximum` corrigés : `notifications` `limit` (100 =
  `notifications.MaxListLimit`) et `history_limit` (200 ; le repli silencieux `> 200` du handler, devenu
  inatteignable, est retiré). Huit chaînes listées, chacune avec son test de plafond : `leaderboard.limit`
  (200), data-quality `limit` (500), jobs `limit` (50), logs `n` (1000), patterns `n` (200), templates `count`
  (10), détections `limit` (500, test au niveau `ops`), file de construction `limit` (200, test au niveau
  `ops`). `openapi.yaml` régénéré (`maximum: 100`, `maximum: 200` seuls changements) ; `generated.ts` inchangé.

**Preuves du rouge** (tests écrits avant les correctifs, sur le code d'avant) :
- L2-1 `go test ./internal/api/handlers/ -run TestBodyLimit -count=1` : FAIL exit 1 ; login 401, register 400,
  password 401, session_context 200 au lieu de 413.
- L2-2 `go test ./internal/api/humacore/ -run TestDefaultMaxBody -count=1` : FAIL exit 1 ; les trois fabriques
  répondent 200 à 2 Mio ; le document porte `MaxBodyBytes = 0` (la constante seule était posée, sans câblage).
- L2-3 garde-rail exécuté avec `applyDefaultMaxBody` neutralisé (fichier restauré ensuite) : FAIL, 216
  opérations « plafond de corps absent » (les autres ont un champ `Body` : défaut Huma).
- L2-4 `go test ./internal/api/handlers/ -run TestSessionContext_ -count=1` : FAIL exit 1, 12 cas refusés à
  tort (statut 200 au lieu de 400, dont `../etc`, `<script>`, slug de 100 000 octets).
- L2-5 `go test ./internal/platform/session/ -run ...` : FAIL exit 1, 4 tests : `Save` et `Touch` d'une session
  de plus de 8192 octets sans erreur, session anonyme vue il y a 25 h encore chargée, `PurgeExpired` = 0 au
  lieu de 1.
- L2-6 service : `limite transmise au repo = 1000000000, want 200` (et 201 -> 200, stats) ; HTTP : 5000 entrées
  servies pour `limit=1000000000`, repo recevant 1000000000 ; repo : 427 855 176 octets alloués pour
  `limit=1048576` (budget 33 554 432), world et stats.
- L2-7 `go test ./internal/api/ -run TestPaginationParamsAreBounded -count=1` : FAIL, `listNotifications.limit` et
  `listProgressionRecords.history_limit` « paramètre de taille entier sans `maximum` ». Les deux chemins du
  garde sur les chaînes éprouvés en retirant une entrée (« absent de paginationStringParams ») puis en
  renommant un test cité (« ne définit pas ») ; état restauré.

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`, Go 1.26.9) :
- Gate du lot, 1re commande (`go test ./internal/api/... ./internal/platform/session/... ./internal/service/ -run
  'Leaderboard|Session|MaxBody|Pagination|BodyLimit' -count=1`) : exit 0, 7 paquets ok.
- 2e commande (`go test ./internal/api/... ./internal/platform/session/... -count=1`) : exit 0 (`api` 27,7 s,
  `handlers` 14,4 s, `humacore`, `middleware`, `wire`, `session` ok).
- Suites complètes des autres paquets touchés : `go test ./internal/service/ ./internal/domain/...
  ./internal/ops/ ./internal/notifications/... ./internal/platform/duckdb/ ./internal/archlint/ ./cmd/server/
  -count=1` : exit 0 ; `go test -tags=integration -p 1 ./internal/platform/duckdb/ -count=1` : ok en 242,9 s,
  exit 0.
- Gate commun : `go build ./cmd/server/` exit 0 ; `go vet` (api, handlers, humacore, session, service, duckdb,
  ops, domain) exit 0 ; `go vet -tags=integration ./internal/platform/duckdb` exit 0 ; `golangci-lint run
  --new-from-rev=ad3433eab` (mêmes paquets) : 0 issues, exit 0 ; `go run ./cmd/openapi-gen` puis `npm run
  generate-types` (aucune différence dans `generated.ts`) ; `go test ./internal/api/ -run 'OpenAPI|Contract'` ok.
  `apps/web` inchangé : gate web non rejoué.

**Non-régression (règle 9)** :
- Classement : le front envoie `limit: 100` (`features/leaderboard/LeaderboardBlock.tsx:179`, seul appelant de
  `useLeaderboard`) ; 100 <= 200, usage servi (`TestLeaderboardPage_LimitIsCappedServerSide`, cas « valeur du
  front »). Aucun usage dans `apps/web/e2e`, `scripts/`, `docker-compose.yml`.
- `/session/context` : le front poste seulement `{title_slug}` (`lib/title-routing/applyActiveTitle.ts:45`) et
  `{player_slug}` (`stores/appShellStore.ts:155`) ; il ne poste JAMAIS `locale`. La langue vit dans le segment
  d'URL `{-$lang}`, liste fermée `KNOWN_LOCALES = ['fr','en']` : `fr-FR` / `en-US` ne sont jamais envoyés, la
  liste fermée {fr, en} est donc exacte. L'e2e `media-like-bug.spec.ts` poste `JGtm`, `Madina97294`,
  `XxDaemonGamerxX`, `Chocoboflor` : acceptés. Slugs réels du poste (`db_profiles.json`) : tous alphanumériques.
- TTL de 24 h : le front ne garde pas la langue dans `localStorage` (`appShellStore` : « pas localStorage ») ;
  la locale de session reste `fr` faute d'écriture. Ce qu'un visiteur anonyme perd après 24 h d'inactivité
  (le TTL est glissant, rafraîchi au plus toutes les 5 min) : le titre et le joueur courants mémorisés, que
  l'URL (`/t/$titleSlug/players/$playerSlug/...`) porte aussi. Les sessions authentifiées gardent 7 jours.
- Tailles de corps vers les routes Huma : formulaires d'authentification et de contexte de quelques dizaines
  d'octets. Les corps les plus gros sont les filtres de pages (`/pages/*`) et les listes `match_ids` de
  l'Explorer et de la Tactique : un identifiant pèse ~40 octets en JSON, soit ~26 000 identifiants pour atteindre
  1 Mio. Les routes à champ `Body` avaient déjà ce plafond Huma ; seules les routes `RawBody` changent
  (Huma n'en posait aucun). nginx prod borne déjà `/api/` à 1m depuis V-1 (§5) et aucun 413 légitime n'y a été
  constaté. Fichiers : `/media/upload` et `/import/openspartan` sont des routes chi nues (leur propre
  `MaxBytesReader`, nginx 600m / 1100m). Aucune route Huma légitime au-delà de 1 Mio hormis le dépôt d'artefact
  de l'ouvrier (16 Mio, `domain.MaxBuildArtifactBytes`, location nginx `/api/v1/internal/` à 20m) : seule
  exception posée, déclarée par `humacore.MaxBody` et par `bodyLimitExceptions`. La volumétrie par joueur n'a
  pas été relevée dans la base locale (tenue en écriture par un serveur en cours) : l'arithmétique ci-dessus et
  l'absence de refus nginx en font foi.
- Entiers : `notifications` `limit` et `history_limit` répondent 422 au-delà de 100 / 200 au lieu d'un repli
  silencieux ; le front envoie 10, 12, 50 (notifications) et 50 (`history_limit`) :
  `TestNotificationsHandler_List_LimitMaximum`.

**Découvertes** (non traitées, règle 5) :
- **D-L2-1** `POST /auth/login` journalise l'identifiant fourni sans borne (`auth: login échoué username=...` :
  un identifiant de 2 Mio a été écrit en entier dans la sortie de test). Couvert par L4-10 (troncature à 64
  caractères) ; ce lot-là doit le tester avec un identifiant volumineux.
- **D-L2-2** `handlePostContext` renvoie `err.Error()` de `Save` dans un 500 (`session_save_error`) : avec le
  plafond de 8 Kio le message contient désormais la taille. Couvert par L9-5 (messages d'erreur fixes).
- **D-L2-3** `offset` de `data-quality/issues` (chaîne, sans borne haute) et les listes CSV (`match_ids` en
  query de l'engagement, `teammates`) ne sont pas des paramètres de taille au sens du garde-rail L2-7 (noms
  hors liste) ; la longueur de l'URL les borne.
- **D-L2-4** `pruneMarks` du store de session garde les marques d'écriture d'une session anonyme jusqu'au TTL
  complet (7 jours) alors que la session expire à 24 h : table en mémoire un peu plus longue que nécessaire,
  bornée par la purge de 6 h.

### L3 - Identité du demandeur et accès aux données d'autrui - 2026-10-15/16 - clos

Base : `git log --oneline -1` = `edefdd427` avant le lot ; branche `feat/securite-2026-10-15`. Session coupée
une fois par la limite de session de l'API (L3-9 en cours, non commité) ; reprise sur `git status`/`git diff`,
L3-9 re-vérifié (tests rejoués) avant commit. Références du registre rouvertes et conformes :
`session_context.go:79-81`, `middleware/session.go:94`, `media_delete.go:86`, `media_likes.go:104`,
`media.go:342-362`, `media_repo_writes.go:28-31` (repli `file_name`), `server_apiv1.go:146-147` et le
`squadActorGuard`, `setup.go:193-200,300`, `profile_service.go:82`, `service_squad_challenges.go:28-66`,
`prestige_player_slug_mw.go:26-29`, `server_apiv1.go:191-196` (`/healthz/home`), `openspartan_import_service.go`
(`importAliases` : `UpsertXUIDAlias` pour tout xuid), `match_view_service.go:478-494`.

**Statuts** : L3-1 à L3-11 `[x]`. Aucun `[!]`, aucun `[~]`. Écarts au texte, justifiés : L3-2 « liker -> 403 » :
liker le média d'un autre reste légitime (galerie agrégée) ; l'attaque (joueur courant forgé) est fermée en
attribuant le like au demandeur réel, et un like sur le chemin du profil d'un ami est refusé (403) par L3-4.
L3-10, séquence tactique : déjà fermée par `MatchsOuvrables` (404 `replay_not_available`, indiscernable d'un
rejeu absent, choix documenté du service) ; laissée telle quelle, vérifiée par `TestSequence_Indisponible`
(cas « match d'un autre joueur »).

- **L3-1** `SessionHandler.WithPlayerAccess` : un `player_slug` que la session ne peut pas LIRE -> 403
  `player_forbidden`, avant toute mutation (aucun fichier de session écrit). Câblé sur `PlayerAccess` (lecture).
  Test `handlers/session_context_access_test.go`.
- **L3-2** `api/requester_resolver.go` : demandeur = profil de `db_profiles.json` (titre courant, hors
  `auth_only`) dont le xuid est celui de `authz.CurrentUser` ; hors propriété appliquée, le joueur choisi.
  `MediaHandler.WithRequester` (suppression, like ; propriété appliquée sans profil possédé -> 401, plus de
  repli sur la session ni sur le corps). Viewer des galeries aligné (`ServiceRegistry.WithViewerResolver`).
  Tests `handlers/media_requester_test.go`, `api/requester_resolver_test.go` ; tests existants adaptés par un
  résolveur de fixture explicite.
- **L3-3** `domain.CanModifyMedia` (règle de `CanDeleteMedia`, extraite) appliquée par
  `MediaService.authorizeMediaAssociation` au propriétaire réel chargé par le chemin COMPLET ; repli `file_name`
  retiré de `SetMediaMatchAssociation` (seul appelant : le handler ; le front envoie l'URL servable, reconvertie
  en chemin stocké). 403 `media_associate_forbidden`, 401 `associate_requires_session`. Tests
  `TestMediaService_AssociateMedia_OwnerOnly` (vraie shared_social), `TestMediaAssociate_ForeignMediaIs403`.
- **L3-4** `authz.CanWritePlayer` ; `middleware.PlayerAccess` (décision unique : slug -> xuid, utilisateur,
  famille résolue pour une LECTURE seulement) ; `RequirePlayerOwnership(access)` classe la requête
  (`IsMutatingRequest`) et la pose dans le contexte (`RequestWrites`, absente = écriture) ; `squadActorGuard`
  décide avec cette classification. `/bootstrap` publie `writable_player_slugs`. Tests :
  `require_player_ownership_family_write_test.go`, `api/family_write_routes_test.go` (walk chi : 44 routes
  d'écriture des groupes `/players/{player_slug}` et `/profiles/{player_slug}/...`, toutes 403 au co-membre,
  toutes les lectures 200, garde présente sur chaque route), `bootstrap_ownership_test.go`, front
  `MatchHeader.test.tsx`, `MediaAudioConfigButton.test.tsx`, `appShellStore.test.ts`,
  `playerWriteAccess.test.ts`. `friends_test` : le PUT d'un co-membre répond `player_forbidden` (garde de
  groupe) au lieu de `friends_forbidden` (assertion mise à jour, refus inchangé ; D-L3-6).
- **L3-5** ADR 0029 (EN) : section « Group/family access is read-only (2026-10-15) ».
- **L3-6** `mustPinIdentity` : épinglage à l'identité liée pour tout non-admin en propriété appliquée, quel que
  soit `profile_mode` (hors propriété appliquée : mode `xbox` seulement, comme avant) ; `CreatePlayer` refuse de
  réécrire le xuid d'un profil existant hors admin (`domain.ErrProfileXUIDConflict`,
  `OnboardRequest.ActorIsAdmin` -> `AllowXUIDChange`) -> 409 `profile_xuid_conflict`. Précondition prod
  (`instance_locked=true`) vérifiée : sans droit de provisioning, 403 `instance_locked` avant toute
  vérification d'identité (`TestSetupHandler_LockedInstanceRefusesForeignProfileBeforeIdentity`) ; avec droit,
  l'épinglage existant tient (`TestSetupHandler_LockedWithGrant_*`). `handleCreatePlayer` : 120 lignes (121
  avant).
- **L3-7** `CreateSquadChallenge` vérifie l'appartenance ; `membersIfMember` (garde + roster) remplace la copie
  inline de `RefreshSquadPool` ; commentaire de `prestige_player_slug_mw.go` réécrit ; garde-rail AST
  `prestige/squad_member_guard_test.go` (exception datée : `ListSquadMembers`, sans acteur, nourri par
  `ListSquadsForUser`) et sa preuve sur une source défectueuse.
- **L3-8** `GET /healthz/home` sous `RequireAuth` + `RequireAdmin` ; commentaire « propriété implicite
  (session) » remplacé. Usages : `e2e/slice-5-home.spec.ts`, `e2e/_helpers/demoData.ts`,
  `scripts/demo-visual-harness.sh`, tous en démo où les deux gardes sont inertes : toujours joignable (D-L3-2
  pour L8-1). OpenAPI inchangé (golden vert). Test `api/healthz_home_admin_test.go`.
- **L3-9** `service/openspartan_import_scope.go` : match à identifiant hors forme Halo écarté
  (`domain.IsValidMatchID`) ; match sans l'importeur écarté (`SkippedForeignMatches`, journalisé) avec ses
  événements marquants ; alias de l'importeur mis à jour (`UpsertXUIDAlias`) ; alias des autres participants
  d'un match retenu seulement proposés par le batch du match (`persist.BatchBuilder.AddXUIDAliases`, INSERT OR
  IGNORE, même transaction) ; plus aucun UPDATE de `xuid_aliases` pour un tiers. Ratchet anti-ART inchangé.
  **Risque résiduel accepté par D-B** : l'importeur peut falsifier un match auquel il a participé, y compris
  les lignes des autres participants, et devancer la version réelle, que l'écriture INSERT-only ne remplacera
  pas. Test `service/openspartan_import_scope_test.go` (vrai fichier SQLite forgé).
- **L3-10** `middleware.RequireMatchParticipation` (middleware de route, `r.With`) : 404
  `match_not_participant`, 503 `participation_unavailable` si la vérification échoue (fermé) ;
  `ServiceRegistry.MatchParticipation` (EXISTS `match_participants` ; profil inconnu laissé au 404 du
  handler). Monté sur les 12 routes `/matches/{match_id}/...` (vue, voisins, positions, objectifs, rejeu x4,
  événements, engagement, exclusion, favori). Tests `api/match_participation_routes_test.go` (walk chi),
  `middleware/match_participation_test.go`.
- **L3-11** `archlint/session_player_slug_test.go` (aucune référence à `SessionPlayerSlug` dans `api/handlers`,
  `service`, `domain` ; allowlist vide) ; `api/identity_param_guard_test.go` (paramètre
  `player`/`player_slug`/`xuid`/`gamertag` hors `/players/{player_slug}` -> `RequirePlayerOwnership` ou
  `RequireAdmin` ; exceptions datées : `DELETE /groups/{id}/members/{xuid}` (propriétaire du groupe, décidé
  par le handler) et `?xuid=` du catalogue (filtre inactif, `NewCatalogRepo(..., nil)`)).

**Preuves du rouge** (sorties gardées dans le scratchpad de session) :
- L3-1 `go test ./internal/api/handlers/ -run TestSessionContext_ForeignPlayerSlugIs403` : FAIL, 200 au lieu
  de 403 (`"current_player_slug":"mallory"`).
- L3-2 `-run 'TestMediaDelete_|TestMediaLike_'` : 4 FAIL, demandeur `JGtm` (joueur courant forgé) au lieu de
  `Mallory` ; 200 au lieu de 401 sans profil possédé (suppression et like).
- L3-3 `TestMediaService_AssociateMedia_OwnerOnly` : `err = <nil>, attendu forbidden` (le média d'autrui était
  associé) ; `TestMediaAssociate_ForeignMediaIs403` : 500 `associate_error` au lieu de 403.
- L3-4 middleware : 4 FAIL (POST sync, PATCH exclusion, DELETE media, PUT audio-config : 200 au co-membre) ;
  walk chi avec la règle d'écriture neutralisée le temps du test (fichier restauré, diff vérifié) : 44 routes
  en 200 au lieu de 403 ; bootstrap : `writable_player_slugs []` sur les 3 cas ; front sur les composants
  d'avant (restaurés) : 2 FAIL (étoile « Retirer des favoris » et engrenage audio présents sur le profil d'un
  ami).
- L3-6 `TestSetupHandler_NonAdminPinnedToOwnIdentityInEveryMode` : modes `azure_manual` et `manual` -> 201
  (profil « Victim » créé avec le xuid de l'attaquant) ;
  `TestProfileService_CreatePlayer_RefusesXUIDChangeOfExistingProfile` : `err = <nil>`.
- L3-7 `TestService_CreateSquadChallenge_RejectsNonMember` : `err = <nil>` ; garde-rail AST sur le
  `service_squad_challenges.go` d'avant (restauré) : « méthode CreateSquadChallenge : reçoit un identifiant
  d'escouade sans appeler assertMemberUser/membersIfMember ».
- L3-8 `TestHealthzHome_RequiresAdmin` : « sans RequireAdmin » ; ratchet L3-11 sur le montage d'avant
  (restauré) : « GET /api/v1/healthz/home : désigne un joueur sans RequirePlayerOwnership ni RequireAdmin ».
- L3-9 `TestOpenSpartanImport_WritesOnlyOwnerMatches` : 7 manquements (match de tiers, ses 2 participants, son
  événement marquant, identifiant piégé `../../../tmp/x` écrit, alias d'un tiers inconnu écrit, « NomReel »
  réécrit en « OtherPlayer », InsertedMatches = 4 au lieu de 2).
- L3-10 `TestMatchRoutesRequireParticipation` : 12 routes « sans RequireMatchParticipation ».
- L3-11 `TestNoSessionPlayerSlugInDecisionLayers` sur les handlers médias d'`edefdd427` (restaurés) :
  `media_delete.go:86`, `media_likes.go:104`.

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`, Go 1.26.9) :
- Gate du lot : `go test ./internal/api/... ./internal/authz/... ./internal/service/... ./internal/prestige/...
  ./internal/openspartan/... ./internal/archlint/ -count=1` : exit 0 (16 paquets ok). Premier passage rouge :
  `TestNoNewFrenchLabelLiteral` (messages neufs en français dans `session_context.go` et
  `media_service_delete.go`), corrigé en messages techniques anglais (`7ad75f884`), garde-rail intact.
- `go test -tags=integration -p 1 ./internal/persist/... ./internal/sync/... ./internal/platform/duckdb/ -count=1` :
  exit 0 (persist 58,6 s, sync 293,1 s, duckdb 235,3 s, 15 paquets ok).
- Gate commun : `go build ./cmd/server/` ok ; `go vet` (api, handlers, middleware, wire, authz, domain, service,
  playerdirectory, prestige, duckdb, archlint) exit 0, `-tags=integration` duckdb + service exit 0 ;
  `golangci-lint run --new-from-rev=edefdd427` (v2.12.2) : 1 goconst (copie de traduction d'erreur), corrigé
  par `mediaWriteError` (`e0591928c`), puis 0 issues ; forme CI `--new-from-merge-base=origin/main` : 0 issues ;
  `go run ./cmd/openapi-gen` + `npm run generate-types` (champ `writable_player_slugs`) ;
  `go test ./internal/api/ -run 'OpenAPI|Contract'` ok ; archlint complet rejoué après le dernier commit : ok.
- Web : purge de `node_modules/.tmp`, `npm run typecheck` exit 0, `npm run lint` 0 erreur (26 avertissements
  préexistants), `npm run test:run` : 922 fichiers passés / 5 ignorés, 9618 tests passés / 23 ignorés, exit 0 ;
  `tools/lint-no-hardcoded-{fields,colors}`, `lint-cross-feature-imports`, `knip-ratchet`,
  `lint-contract-ratchet` : propres.

**Non-régression (règle 9)** :
- Routes mutantes des groupes joueur (walk chi) -> élément d'interface -> comportement sur le profil d'un ami :
  arcs (créer, adopter un preset, supprimer) -> `AscensionObjectivesTab` (`CreateArcForm`, `ArcPresetPicker`,
  bouton de suppression) : masqués ; défis Prestige (créer, abandonner) -> `CreateChallengeForm`,
  `ObjectiveCard` : masqués ; `PATCH prestige/challenges/{id}`, `suggest-next`, `squads/{id}/members` (ajout,
  retrait), `POST /sync`, `engagement/recompute_coefficients` : aucun appelant dans l'interface ; mode pilote ->
  `PilotModeToggle` et son CTA : masqués ; campagnes (lancer, pause, reprise, clôture, abandon) ->
  `ProgressionLeadsSection` / `StartCampaignModal`, `CampaignTracker` : non proposées / masquées ; coach
  (accepter, ignorer) -> `CoachProposalsCard` : masqués ; amis -> `PlayerFriendsSection` : déjà en lecture par
  `can_edit` ; exclusion, favori -> `MatchHeaderCard` (bouton, étoile) et tuiles d'accueil (`HomePage` ->
  `MatchCard`) : masqués ; médias -> `UploadButton` masqué, suppression et association/réassociation
  masquées, cœur affiché désactivé (`MediaPage`, `RecentMediaRail`, `MatchMediaTab`), engrenage audio masqué ;
  notifications (lue, non lue, tout lire, supprimer) -> cloche masquée (`NavL1`), toasts coupés
  (`toastBridge`), page dédiée en lecture seule (`NotificationsPage`, `NotificationItem readOnly`) ;
  préférences et test -> `NotificationsSettingsTab` : figés ; escouades et défis d'escouade ->
  `SquadFocusStrip` (enregistrer, renommer, supprimer), `useSquadPresets` (pied, gestion),
  `SquadObjectivesPanel` (évaluer, rejoindre, supprimer, proposer, créer) : masqués ; titres (pause, purge) ->
  Réglages > Titres (`TitlesTab`) : figés. Source : `writable_player_slugs` du bootstrap ; liste absente = rien
  n'est masqué (le serveur reste l'arbitre). Aucun libellé ajouté (masquage ou désactivation seuls), aucune
  couleur.
- L3-1 : le front ne poste `player_slug` qu'au changement de joueur, parmi `available_players` (lisibles) ; un
  refus est avalé (`.catch`), rien ne casse.
- L3-10 : liens vers le rejeu seulement sur des matchs du joueur du profil : vue du match (participé, gardée),
  tuiles d'accueil (matchs récents du joueur), Explorer (matchs du joueur), Tactique (morts du joueur),
  Escouade (`SquadMatchHistoryRow` : matchs partagés par le joueur et la composition), règle des records
  d'armes (frags du joueur). Positions, objectifs et événements ne sont lus que depuis la vue d'un match joué.
  Les e2e démo (rejeu figé de `demo-player`) n'ont pas été lancés (serveur requis) : couverts par F-0b.
- L3-8 : sondes e2e et harnais démo en démo, gardes inertes.
- L3-6 : le front n'envoie jamais `profile_mode` (défaut `xbox`) : l'onboarding de son propre profil est
  inchangé.

**Découvertes** : D-L3-1 à D-L3-9 (§6), non traitées.

Commits : `bde785689`, `adcd9f2db`, `e1525f55b`, `ad1a9484d`, `40b649b6a`, `73ba6fb8e`, `7ad75f884`,
`e0591928c`, puis le commit de clôture (plan, thought_log).

### L4 - Session et authentification - 2026-10-16 - clos

Base : `git log --oneline -1` = `3e65359f0` avant le lot ; branche `feat/securite-2026-10-15`. Références du
registre rouvertes avant d'agir et conformes : `handlers/auth.go:221-251` (`AuthReady`, identité et jetons posés
AVANT `OnAuthSuccess`, erreur seulement journalisée), `middleware/require_auth.go` (`AuthReady` suffit hors mode
password), `authz/authz.go:91-111` (utilisateur synthétisé), `handlers/groups.go:153,223,286`,
`service/xbox_auth_service.go:157-216` (`resolvePendingInvite` puis `Consume` APRÈS `CreateFromXbox`),
`handlers/user_auth.go:228-251` (`Validate` puis `Create` puis `Consume`) et `:272-289` (logout : `Username`/`Role`
seuls effacés), `middleware/require_admin.go:25`, `handlers/settings.go:216`, `handlers/media_requester.go:45`
(D-L3-5), `handlers/setup.go:346` (repli sur `sess.Role`), `platform/session/store.go` (seul `New` crée un
identifiant), `handlers/auth_xbox_oauth.go:209` (RT persisté avant `linkAttemptToSession`), `config.go:242-244`
(aucune normalisation), `userstore/store.go:125-166` (`IsEmpty` puis `Create` hors verrou) et `:314,362`,
`authz.InstanceLocked` (repli ouvert).

**Statuts** : L4-1 à L4-12 `[x]`. Aucun `[!]`, aucun `[~]`. D-L1-3 traité par L4-11.

- **L4-1** `handlers/auth_device_flow_lock_test.go` : tentative autorisée par Microsoft pour un xuid inconnu,
  instance verrouillée → statut `failed`, `error_code` `instance_locked`, jetons effacés de la tentative, session
  sans `AuthReady`/identité/jetons/username, aucun compte, route `RequireAuth(xbox)` en 401.
- **L4-2** `handlers/auth_login_complete.go` (`completeLogin`, partagé avec l'OAuth) : la stratégie post-flux
  s'applique à une COPIE de la session ; seul un succès recopie le résultat avec `AuthReady`, l'identité et les
  jetons ; `ErrSessionNotAuthenticated` (aucun compte local à lier, modes password et none) reste non bloquante
  comme avant. `platform/auth/attempt_claim.go` : `ClaimAuthorized` remet la tentative UNE fois (statut
  `provisioned`, secrets effacés du registre), `Fail`, `Rebind`. `ErrInstanceLocked` déplacée de `service` vers
  `platform/auth` (lue par le handler sans importer le service). Front : `instance_locked` affiche « Cette
  instance est fermée aux nouvelles inscriptions. » / « This instance is closed to new registrations. »
  (`XboxLoginPage`, `StepDeviceCode`, clé existante). Tests : `attempt_claim_test.go` (remise unique sous 8
  sondages concurrents), `XboxLoginPage.test.tsx` (« compte inconnu refusé par le verrou »).
- **L4-3** `authz.CurrentAccount` (compte persisté, jamais synthétisé ; `CurrentUser` s'appuie dessus) ;
  `GroupsHandler.currentAccount` : créer un groupe ou émettre une invitation sans compte → 403 `account_required` ;
  l'émetteur d'une invitation de groupe est désormais le NOM DU COMPTE (`created_by`). `InviteStore` reçoit un
  `IssuerLookup` au constructeur (nil : aucune invitation valable) : `Validate`, `Consume` et `List.valid` exigent un
  émetteur existant (`ErrInviteIssuerGone`). `Consume` vérifie et marque sous le même verrou et rend l'invitation ;
  `Release` la rend (compensation). Inscription (mode invitation) et SSO réservent le code AVANT toute création de
  compte et le rendent si la création échoue (`UserAuthHandler.admitRegistration`/`releaseInvite`,
  `service/xbox_auth_invite.go`). `InviteStore.Get`, sans appelant de production après ce lot, est supprimé (tests
  passés sur `Validate`/`Consume`). Tests : `invite_issuer_test.go` (handlers, userstore),
  `xbox_invite_issuer_test.go` (service).
- **L4-4** `session.Store.Revoke` (`platform/session/revoke.go`) : fichier supprimé et identifiant retenu révoqué
  (1 h, 10 000 entrées au plus) ; `Save`/`Touch` refusent un identifiant révoqué (`ErrSessionRevoked`) : une requête
  concurrente chargée avant la déconnexion ne ressuscite pas la session. `WithSession` répond à un identifiant révoqué
  par un cookie expiré et ne le réécrit pas en fin de requête. `handleLogout` révoque (500 `logout_failed` si la
  suppression échoue). Tests : `user_auth_logout_test.go` (SSO → déconnexion → 401), `revoke_test.go`,
  `session_revoke_test.go`.
- **L4-5** `authz.IsAdmin(sess, lookup)` (rôle relu dans le compte, lookup nil : faux) : `RequireAdmin(demo, mode,
  users)` (8 montages de `server_apiv1.go`), verrou d'instance de `PATCH /settings` (`SettingsHandler.WithUserLookup`),
  suppression et association de médias (`MediaHandler.WithUserLookup`, D-L3-5), exemption admin du setup (repli sur
  la session supprimé). Garde-rail `archlint/session_role_read_test.go` : aucun déréférencement de `*x.Role` dans le
  code non-test du module, allowlist datée d'une entrée (affichage `is_admin` du bootstrap, D-L4-1). Tests :
  `require_admin_reread_test.go` (admin → 200, rétrogradé → 403, supprimé → 403), `admin_role_reread_test.go`
  (verrou d'instance, suppression de média). Tests existants adaptés : les sessions « admin » des fixtures ont un
  compte admin dans un magasin de test (le rôle de session seul ne suffit plus, c'est le contrat).
- **L4-6** `session.Store.Rotate` : nouvel identifiant, même état, ancien révoqué avec ce successeur, écriture sous
  le nouveau. Appelée à la connexion par mot de passe, à l'inscription (auto-connexion, même classe), au device-flow
  (la tentative suit : `Rebind`) et au callback OAuth (`finishOAuthLogin`). Une requête concurrente qui porte encore
  l'ancien cookie reçoit celui du successeur. Tests : `session_rotation_test.go` (mot de passe, device-flow avec
  second sondage au nouveau cookie), `auth_xbox_oauth_finish_internal_test.go` (OAuth, et refus du verrou sans
  rotation), `revoke_test.go`, `session_revoke_test.go`.
- **L4-7** `config/config_auth.go` : `normalizeMode` (minuscules, espaces de bord, vide = défaut) à la lecture ;
  `validateModes` appelée par `Validate` dans TOUS les environnements (démo et développement compris) : valeur
  hors {none, password, xbox} / {invite, open, closed} → démarrage refusé. Message de `cmd/server` mis à jour ;
  `docs/CONFIGURATION.md` + `docs/FR/CONFIGURATION.md`. `config.go` reste à 628 lignes (gel 629). Test table-driven
  `config_auth_modes_test.go`.
- **L4-8** `completeOAuthCallback` (extrait du `Callback`, réseau exclu) : le refresh token est persisté APRÈS la
  décision de connexion. Tests `auth_xbox_oauth_rt_internal_test.go` (refusé : aucune persistance ; accepté : une
  persistance, compte déjà existant).
- **L4-9** `userstore/store_roles.go` : `CreateFirstAdmin` (instance vide vérifiée sous le verrou d'écriture,
  sinon `ErrNotFirstAccount` et l'inscription reprend les gardes ordinaires, `UserAuthHandler.createAccount`) ;
  `ErrLastAdmin` sur `Delete`/`SetRole` (409 `last_admin` côté admin). `SetRole` et `Delete` déplacés dans ce
  fichier : `store.go` passe de 503 à 471 lignes. Tests : `user_auth_first_admin_test.go` (6 inscriptions
  simultanées sur instance vide ; dernier admin 409), `store_admin_guard_test.go`.
- **L4-10** `loginlimit.Truncate` (64 caractères, source de la clé normalisée du limiteur) sur les identifiants
  fournis par le client dans les journaux de `user_auth.go` ; codes d'invitation retirés des journaux (`admin.go`,
  `groups.go`, `user_auth.go`, SSO). Garde-rail `archlint/invite_code_log_test.go` (aucun attribut slog `"code"` dans
  les six fichiers qui manipulent une invitation ; preuve sur une source défectueuse). Test
  `auth_log_hygiene_test.go` (identifiant de 100 000 caractères, D-L2-1 ; code d'invitation généré puis révoqué).
- **L4-11** `SetPasswordRequest.CurrentPassword` ; `checkCurrentPassword` : compte avec mot de passe → 400
  `current_password_required` sans lui, 403 `invalid_current_password` s'il est faux (échec compté par le limiteur
  pour l'identifiant et l'IP), succès remet les compteurs à zéro ; compte sans mot de passe (Xbox) : premier mot de
  passe sans l'ancien. Le front sait si le compte a un mot de passe par `has_password` du bootstrap
  (`BootstrapService.currentUserHasPassword`, existant) : aucun ajout d'API. `SetPasswordCard` : champ « Mot de
  passe actuel » / « Current password » et libellé « Nouveau mot de passe » / « New password » quand le compte a un
  mot de passe ; tous les libellés passent au manifeste `common` (FR + EN, D-L1-3) ; messages de refus par
  `setPasswordErrorKey` (`loginError.ts`). Tests : `user_auth_set_password_current_test.go` (deux cas, 6e mot de
  passe actuel faux → 429), `TestUserAuth_SetPassword_OptIn` (compte Xbox), `SetPasswordCard.test.tsx` (4 cas neufs
  dont l'anglais), `loginError.test.ts`.
- **L4-12** `authz.InstanceLocked(envLocked, enforced, load)` : réglages illisibles en propriété appliquée →
  verrouillé, journalisé en ERROR ; sans propriété appliquée → non verrouillé (WARN). `server_apiv1.go` passe
  `authz.Enforced(cfg.DemoMode, cfg.AuthMode)`. Test `authz_test.go` inversé. ADR 0035 amendée (EN, D5, 2026-10-16,
  raison : le verrou ne ferme jamais l'accès des comptes existants ni de l'admin, seulement l'entrée d'identités
  nouvelles ; l'ouverture sur fichier corrompu était la vraie faille).

Écarts au texte, additifs et dans la classe de défaut : rotation aussi à l'inscription (L4-6) ; garde-rail du rôle
de session (L4-5) ; garde-rail des codes d'invitation étendu au SSO et à l'OAuth (L4-10) ; secrets d'une tentative
effacés après remise ou échec (L4-2).

**Preuves du rouge** (tests écrits avant les correctifs ; sorties gardées dans `%TEMP%\l4\`) :
- L4-1 `go test ./internal/api/handlers/ -run TestDeviceFlow_InstanceLocked` : FAIL exit 1, statut `authorized`
  (attendu `failed`), `error_code` nil, `AuthReady=true`, identité et jetons en session, route gardée 200.
- L4-3 : FAIL sur l'état d'avant (`NewInviteStore` à un argument le temps du run) : groupe créé (201) et invitation
  émise (201) sans compte ; inscription acceptée (201) avec l'invitation d'un émetteur supprimé ; 6 comptes créés
  par un seul code (handlers et SSO) ; SSO : `err = <nil>` au lieu de `ErrInstanceLocked` pour un émetteur supprimé.
- L4-4 `TestLogout_SSOSessionNoLongerReads` : cookie non expiré, session encore sur disque, lecture 200 au lieu de 401.
- L4-5 (signature `RequireAdmin` posée sans changer le comportement) : rétrogradé 200 et supprimé 200 au lieu de 403 ;
  verrou ouvert (200) par un admin rétrogradé ; `RequesterIsAdmin` vrai. Sur `3e65359f0`, le garde-rail aurait
  relevé `media_requester.go:45`, `settings.go:216`, `setup.go:346`, `require_admin.go:25` et
  `bootstrap_service.go:248` (allowlisté, D-L4-1).
- L4-6 `TestRotation_PasswordLogin`, `TestRotation_DeviceFlow` : même identifiant avant et après ; OAuth : rouge en
  remplaçant `Rotate` par `Save` le temps du run (fichier restauré, vérifié), vert avec `Rotate`.
- L4-7 : 7 sous-tests FAIL (`Xbox`, `  PASSWORD `, ` Open`, `CLOSED` non normalisés ; blanc seul non ramené au
  défaut ; `sso`, `xbx`, `public` acceptés par `Validate`).
- L4-8 : refresh token persisté pour l'identité refusée ; accepté : persisté AVANT la création du compte.
- L4-9 : 6 admins créés par 6 inscriptions simultanées ; suppression du dernier admin 204, rétrogradation acceptée.
- L4-10 : 100 089 octets de journal (identifiant entier) ; `code=K97G54W3` journalisé ; sur `3e65359f0`, 9 attributs
  `"code"` (admin x3, groups, user_auth, SSO x4).
- L4-11 : sans mot de passe actuel 204 (attendu 400), faux 204 (attendu 403), 1er échec 204 ; front sur le composant
  d'avant : 4 tests FAIL (champ absent, libellés anglais absents, message absent).
- L4-12 `TestInstanceLocked/propriété_appliquée,_erreur_de_lecture_⇒_verrouillé` : FAIL (repli non verrouillé).
- Front L4-2 : sur le `XboxLoginPage.tsx` d'avant (restauré ensuite, `git diff` vérifié), le test « compte inconnu
  refusé par le verrou » FAIL.

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`, Go 1.26.9) :
- Gate du lot : `go test ./internal/api/... ./internal/authz/... ./internal/platform/session/...
  ./internal/platform/userstore/... ./internal/service/... ./internal/config/... -count=1` : exit 0, 15 paquets ok.
- Autres paquets touchés : `go test ./internal/archlint/ ./internal/platform/auth/... ./internal/platform/loginlimit/
  ./internal/domain/... ./cmd/server/ -count=1` : exit 0. Premier passage rouge : `TestNoNewFrenchLabelLiteral`
  (deux messages neufs en français dans `user_auth_current_password.go`) : messages techniques anglais, et
  allowlist RESSERRÉE (baisses dues au lot : `user_auth.go` 13 → 12, `xbox_auth_service.go` 1 → 0, entrée retirée).
  Un test neuf rouge au passage complet (`TestRequireAdmin_RereadsRoleFromStore`, dernier admin protégé par L4-9) :
  fixture complétée d'un second admin.
- Gate commun : `go build ./cmd/server/` exit 0 ; `go vet` (api/..., authz, config, session, userstore, platform/auth,
  loginlimit, service, domain, archlint, cmd/server) exit 0 ; `golangci-lint run --new-from-rev=3e65359f0` et forme
  CI `--new-from-merge-base=origin/main` sur ces paquets : 0 issues (premier passage : goconst sur les modes de
  `config_auth.go`, corrigés par des constantes) ; `go run ./cmd/openapi-gen` puis `npm run generate-types` : aucune
  différence (`/auth/password` sans schéma de corps documenté, nouveaux refus couverts par `default`) ;
  `go test ./internal/api/ -run 'OpenAPI|Contract'` ok.
- Web : purge de `node_modules/.tmp`, `npm run typecheck` exit 0, `npm run lint` 0 erreur (26 avertissements
  préexistants), `npm run test:run` : 922 fichiers passés / 5 ignorés, 9624 tests passés / 23 ignorés, exit 0 ;
  `lint:fields`, `lint:colors`, `lint-cross-feature-imports`, `knip-ratchet`, `lint-contract-ratchet` : propres.
- Hook pre-commit : un premier commit refusé par gitleaks (mot de passe littéral dans un test neuf, faux positif) :
  constante de test, aucune exception ajoutée à `.gitleaks.toml` ; second passage vert.

**Non-régression (règle 9)** :
- L4-1/L4-2 : changement visible pour un compte Microsoft inconnu sur instance verrouillée par code d'appareil : il
  voyait « Authentification réussie » puis l'onboarding ; il voit désormais le message de refus (FR + EN) et
  « Réessayer ». Comptes existants : inchangé (`TestE2E_DeviceCodeFlow_HappyPath`, statut `provisioned` déjà géré par
  `XboxLoginPage` et `StepDeviceCode`).
- L4-4 : `LogoutButton` et `NavL1MobileActions` appellent `POST /auth/logout` puis `window.location.assign('/')` :
  le rechargement lit un bootstrap anonyme (`current_username` nul) et `__root` redirige vers `/login` en modes
  password et xbox (lecture du code ; aucun e2e de connexion).
- L4-6 : device-flow : le front sonde toutes les 5 s avec le cookie du navigateur ; le sondage qui conclut pose le
  nouveau cookie et la tentative suit (`Rebind`) : le suivant la retrouve (statut `provisioned`, aucune seconde
  rotation) ; `StepDeviceCode`, qui continue de sonder, n'est donc jamais renvoyé sur `attempt_not_found`. SSO : le
  callback est une navigation plein écran, la redirection 302 porte le nouveau cookie. Requêtes concurrentes au
  moment d'une connexion : cookie du successeur, jamais un cookie expiré.
- L4-7 : valeurs réelles relevées : `.env.local.example` (`xbox`, `password`, `none` en commentaire), `.env.local`
  local (`xbox`), prod (`LEVELUP_AUTH_MODE=xbox`), `docker-compose.yml` et workflows (aucune), e2e
  (`readme-shots` cite `xbox`/`none`). Toutes acceptées. `LEVELUP_REGISTRATION` du VPS non vérifiée : D-L4-2.
- L4-3 : un compte SSO a toujours un compte (créé par la connexion) ; seule une identité Halo sans compte (modes
  password/none, device-flow sans connexion locale) perd la création de groupe et l'émission d'invitation. Les
  invitations de groupe émises avant le lot portent un gamertag dans `created_by` : valables si ce gamertag, en
  minuscules, est le nom du compte (cas des gamertags sans espace ni symbole), sinon à réémettre.
- L4-9 : `UsersSection` masque suppression et changement de rôle pour le compte courant, qui est admin : le dernier
  admin n'est jamais atteignable depuis l'interface (409 réservé à l'API).
- L4-11 : `SetPasswordCard` est affichée dans les Réglages (toujours) et l'onboarding OpenSpartan (seulement sans
  mot de passe : premier mot de passe, sans champ « actuel ») ; après succès, le bootstrap rechargé passe
  `has_password` à vrai.
- L4-12 : le verrou ne touche ni la connexion des comptes existants ni l'admin (exempté sur `POST /setup/players`) :
  seules les identités nouvelles sont refusées tant que les réglages restent illisibles.
- L4-5 : aucun changement pour un admin en place ; un admin rétrogradé reçoit 403 sur chaque route d'administration
  (le menu reste affiché jusqu'à sa reconnexion, D-L4-1).

**Découvertes** : D-L4-1 à D-L4-5 (§6), non traitées.

Commits : `61eb8ff9c` (code, tests, docs, ADR), puis le commit de clôture (plan, thought_log).

### L4b - Compléments décidés en cours d'exécution - 2026-10-16 - clos

Base : `git log --oneline -1` = `fc8e6cfb3` avant le lot ; branche `feat/securite-2026-10-15`. Références rouvertes
avant d'agir : `middleware/player_access.go` (`Allows`), `middleware/require_player_ownership.go` (classement
`IsMutatingRequest`), `middleware/require_auth_mutations.go` (`playerScopedSubPath`, `AmbiguousPath`),
`api/requester_resolver.go`, `handlers/media_likes.go` (`resolveLikerIdentity`), montage des notifications
(`server_apiv1.go:951`), `platform/session/{store,revoke}.go` (`Rotate`, `Revoke`), `handlers/user_auth.go`
(`handleSetPassword`), `handlers/auth_xbox_oauth.go` (callback : 8 réponses JSON), `handlers/auth_login_complete.go`
(`completeLogin`, `loginFailureCode`), `service/xbox_auth_{service,invite}.go`, front `NavL1.tsx:426`, `toastBridge.tsx`,
`NotificationsPage.tsx`, `NotificationsSettingsTab.tsx`, `MediaPage`/`RecentMediaRail`/`MatchMediaTab` (`likeDisabled`).

**Statuts** : L3-12, L3-13, L4-13, L4-14 `[x]` ; D-L4-2 `[x]` (vérification du superviseur recopiée, confrontée au
code). Aucun `[!]`, aucun `[~]`. D-L3-8, D-L4-3, D-L4-4 traités (annotés en §6).

- **L3-12** `middleware/player_visitor_actions.go` : une action du visiteur (méthode et sous-chemin EXACTS, jamais sur
  un chemin ambigu) se décide comme une LECTURE du profil consulté ; seule entrée : `PATCH /media/likes`. Elle reste une
  écriture pour `RequireAuthForMutations` (session authentifiée exigée) et pour la classification posée dans le
  contexte. Le like est attribué au profil possédé du visiteur par le résolveur de L3-2 (inchangé). Walk chi :
  `familyVisitorWrites` nomme l'exception (liste exacte, une entrée périmée échoue). Front : cœur actif sur le profil
  d'un ami (`likeDisabled = isPending` dans les trois galeries). Tests : `require_player_ownership_like_test.go` (ami
  200, inconnu 403, autres écritures médias 403, autre verbe, segment en plus et chemin `%2F` refusés),
  `api/family_media_like_test.go` (garde réelle + handler + résolveur réel : 200, liker `Mallory`, ni le profil
  consulté ni le corps), `MatchMediaTab.test.tsx`.
- **L3-13** `middleware.RequirePlayerOwner` (garde de ROUTE, `r.With`, après routage) : chaque route
  `/players/{player_slug}/notifications...` se décide avec la règle d'écriture (propriétaire ou admin), GET compris.
  Garde-rails : `TestOwnerOnlyRoutesCarryOwnerGuard` (les 9 routes walkées portent la garde) et le rejeu co-membre de
  `TestFamilyMemberCannotWriteAnyPlayerRoute` (403 sur les 9, lectures comprises). Front : page dédiée → message « Les
  notifications d'un profil ne sont visibles que par son propriétaire. » / « A profile's notifications are visible to
  its owner only. », aucune requête ; Réglages › Notifications : catégories remplacées par le message, abonnements et
  bouton de test masqués, préférences jamais lues (les réglages locaux du navigateur restent) ; cloche et toasts déjà
  coupés par L3-4 (tests ajoutés). La prop `readOnly` de `NotificationItem`, morte, est retirée (fichier revenu à son
  état d'avant L3). Tests : `require_player_owner_test.go`, `friendProfile.test.tsx` (page FR + EN, réglages, toasts :
  zéro requête ; son profil : la liste se lit), `NavL1.test.tsx` (cloche absente / présente).
- **ADR 0029** (EN) : sous-section « Clarifications (user decisions of 2026-10-16) » (like du visiteur, notifications
  privées, gardes et garde-rails nommés).
- **L4-13** `session.Store.RevokeAccount(username, keepID)` (`platform/session/revoke_account.go`) : parcours du
  répertoire, compte comparé sans casse, révocation par `revoke` (fichier supprimé, identifiant retenu, réécriture
  concurrente refusée) ; fichier illisible laissé en place et journalisé. `handleSetPassword` → `closeOtherSessions`
  (`handlers/user_auth_password_sessions.go`) : `Rotate` de la session courante puis révocation des autres ; un échec
  est journalisé en ERROR (le mot de passe est déjà changé, le client reçoit 204). Tests :
  `user_auth_password_revokes_test.go` (deux sessions : la courante change d'identifiant et lit encore, l'autre 401 et
  absente du disque ; sessions d'un autre compte intactes), `revoke_account_test.go` (copie en vol refusée, anonymes et
  autres comptes intacts, nom vide sans effet).
- **L4-14** `XboxOAuthHandler.refuseCallback` : tout refus du callback → 302 `/login?oauth_error=<code>`
  (`oauth_denied`, `missing_params`, `state_mismatch`, `no_session`, `code_exchange_failed`, `halo_exchange_failed`,
  `instance_locked`, `invite_invalid`, `link_failed`), détail aux journaux ; `error_description` de Microsoft et
  `err.Error()` ne sortent plus. `invite_invalid` : `auth.ErrInviteInvalid` (forme de `ErrInstanceLocked`, `errors.Is`
  vrai pour les deux) rendu par `refusalCause` quand le verrou refuse une identité dont la session présentait une
  invitation non réservée ; `LoginRedirect` garde désormais le code présenté même invalide (il n'accorde rien tant que
  `Consume` ne l'a pas réservé). Front : `OAuthErrorNotice` (`role="alert"`, `text-destructive`) sur l'écran de
  connexion, modes xbox et mot de passe ; `oauthErrorKey` (code inconnu : message générique, jamais le code brut) ;
  messages FR + EN au manifeste `common` (`oauth_invite_invalid`, `oauth_denied`, `oauth_interrupted`,
  `oauth_exchange_failed`, `oauth_failed` ; `instance_locked` reprend `register_instance_closed`) ; le device-flow
  traduit aussi `invite_invalid`. Tests existants du callback alignés sur le nouveau contrat en GARDANT leurs noms
  (baseline de présence `tests_pre_migration.jsonl`) : `TestXboxOAuth_Callback_{MissingCode,MissingState,StateMismatch,
  ErrorParam,StateMatches_AttemptsExchange}`, `TestE2E_AuthCodeFlow_CSRFGuard` (rejeu : `state_mismatch`),
  `TestOAuthFinish_LockedLeavesSessionUntouched`, `TestOAuthCallback_LockedPersistsNoRefreshToken`,
  `TestXboxOAuth_LoginRedirect_InvalidInviteIgnored` (le code est gardé, sans effet) ; neufs :
  `TestOAuthFinish_InvalidInviteOnLockedInstance`, `loginError.test.ts`, `LoginPage.test.tsx` (12 cas FR / EN, sans
  refus, mode mot de passe), `XboxLoginPage.test.tsx` (device-flow `invite_invalid`).

Écarts au texte, justifiés : L4-14 couvre aussi `invite_invalid` (demandé par la consigne du lot ; sans trace de
l'invitation, le refus d'une invitation périmée disait « instance fermée ») ; deux tests de contrat (walk famille,
callback) modifiés parce que le contrat change par décision de l'utilisateur, jamais assouplis : l'exception du like
est nommée et exacte, les notifications sont plus strictes, chaque refus du callback reste vérifié par son code.

**Preuves du rouge** (sorties gardées dans `%LOCALAPPDATA%\Temp\l4b\`) :
- L3-12 `go test ./internal/api/middleware/ -run 'FamilyMayLikeMedia|LikeOnStranger'` : FAIL exit 1, « PATCH
  /media/likes : statut 403, attendu 200 » ; `go test ./internal/api/ -run TestFamilyMemberLikesFriendMedia` : FAIL,
  403 `player_forbidden` ; front `MatchMediaTab.test.tsx` : bouton « Aimer » `disabled`.
- L3-13 `go test ./internal/api/ -run TestOwnerOnlyRoutesCarryOwnerGuard` : FAIL, 9 routes « donnée privée du profil
  sans RequirePlayerOwner » (GET liste, compteur, préférences compris) ; front `friendProfile.test.tsx` : 4 FAIL
  (message absent, liste lue) ; cloche : test rouge en retirant la garde de `NavL1.tsx` le temps du run (fichier
  restauré, `git diff` vide).
- L4-13 `go test ./internal/api/handlers/ -run TestSetPassword_RevokesOtherSessions` : FAIL, identifiant courant
  inchangé, autre session 200 au lieu de 401, encore sur disque.
- L4-14 callback : 4 FAIL (400 / 400 / 400 / 403 JSON au lieu de 302) ; internes : verrou 403 au lieu de 302,
  invitation périmée → `instance_locked` en JSON ; `InvalidInviteIgnored` : code non gardé ; front : 27 FAIL
  (`oauthErrorKey` absent, aucun `role="alert"`) puis device-flow `invite_invalid` FAIL (texte attendu absent).

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`, Go 1.26.9) :
- Gate du lot : `go test ./internal/api/... ./internal/authz/... ./internal/platform/session/...
  ./internal/platform/userstore/... -count=1` : exit 0, 8 paquets ok, 0 `--- FAIL:`. Premier passage rouge :
  `TestE2E_AuthCodeFlow_CSRFGuard` (attendait 403 sur le rejeu) : aligné sur la redirection (`state_mismatch`).
- Autres paquets touchés : `go test ./internal/service/... ./internal/platform/auth/... ./internal/domain/...
  ./internal/archlint/ ./cmd/server/ -count=1` : exit 0, 16 paquets ok, 0 `--- FAIL:`. Aucun diff de persist, sync ou
  migration : pas d'intégration requise.
- Gate commun : `go build ./cmd/server/` exit 0 ; `go vet` (api/..., session, platform/auth, service, domain, archlint)
  exit 0 ; `golangci-lint run --new-from-rev=fc8e6cfb3` : premier passage 2 issues (goconst `/media/likes`, gofmt d'un
  test), corrigées (constante `mediaLikesSubPath`), puis 0 ; forme CI `--new-from-merge-base=origin/main` : 0 issues ;
  `go run ./cmd/openapi-gen` puis `npm run generate-types` : aucune différence (aucune route Huma changée, le callback
  est hors document) ; `OpenAPI` / `Contract` dans la suite `./internal/api/` : ok.
- Web : purge de `node_modules/.tmp`, `npm run typecheck` exit 0, `npm run lint` exit 0 (0 erreur, 26 avertissements
  préexistants), `npm run test:run` exit 0 : 923 fichiers passés / 5 ignorés, 9661 tests passés / 23 ignorés ;
  `lint:fields`, `lint:colors`, `lint-cross-feature-imports`, `knip-ratchet`, `lint-contract-ratchet` : exit 0.
- Hooks pre-commit (gofmt, check-merge-conflict, docs-fr-sync, gitleaks, go-vet) verts sur les trois commits.

**Non-régression (règle 9)** :
- L3-12 : le front like par `PATCH /players/{slug}/media/likes` avec le slug du profil consulté
  (`features/media/queries.ts:182`), seul appelant (aucun e2e ni script) ; le viewer des galeries est le même
  résolveur que le liker : le cœur allumé après un like est celui du visiteur. Sur son propre profil : inchangé.
- L3-13 : points d'entrée relevés : cloche (`NavL1.tsx:426`, déjà conditionnée), toasts (`toastBridge.tsx`, slug vide
  sur un ami), page (`routes/.../notifications.tsx`, atteinte par le lien « Voir tout » de la cloche et par le
  changement de joueur qui garde la route courante : elle affiche le message), Réglages › Notifications (onglet de
  `NavL1.tsx:309`, réglages locaux conservés). Admin : `writable_player_slugs` liste tous les profils, rien ne change.
  E2E : `compare-bars.spec.ts` simule le compteur sur le profil du joueur (inchangé).
- L4-13 : `SetPasswordCard` (Réglages ; onboarding OpenSpartan pour un premier mot de passe) appelle l'API en `fetch` :
  le `Set-Cookie` du nouvel identifiant s'applique, le bootstrap rechargé lit la session courante. Les autres appareils
  du compte reçoivent 401 à leur requête suivante (`levelup:auth-required`, retour à la connexion). Le premier mot de
  passe d'un compte Xbox ferme aussi ses autres sessions (même règle).
- L4-14 : le callback n'est atteint que par la redirection de Microsoft après `/auth/xbox/login` (`JoinPage`,
  `RedirectFlowPanel`, `ReauthBanner`, `AppearanceComponentCard`) : un refus arrivait en JSON brut, il arrive sur
  l'écran de connexion avec un message ; succès inchangé (302 vers `/`). Un utilisateur déjà connecté (bannière de
  reconnexion) est renvoyé de `/login` vers `/` par `__root` : message perdu (D-L4b-3). Démo : refus JSON inchangé.

**Découvertes** : D-L4b-1 à D-L4b-3 (§6), non traitées.

Commits : `a654e8981` (L3-12, L3-13, ADR 0029), `88e0b28ce` (L4-13), `c44e07aec` (L4-14), puis le commit de clôture
(plan, thought_log).

### L5 - Travaux coûteux - 2026-10-16 - clos

Base : `git log --oneline -1` = `6e8df99c7` avant le lot ; branche `feat/securite-2026-10-15`. Références du registre
rouvertes avant d'agir : `handlers/auth.go` (`handleStartDeviceFlow`, `GetOrCreate` sans plafond),
`platform/auth/attempt_store.go`, `server_apiv1.go` (routes `/assets/*` nues, `halo.NewHaloProvider()` dans la closure
`MapImageURLFetcher`), `assets/resolver_default.go` (`Get` : local puis fetch, aucun cache négatif), `handlers/assets.go`,
`sync/match_recomputer.go` (`AcquireSharedWriterStandalone` + `batchComputePerformanceScores`/`batchComputeLUSR` qui ne
LISENT que `sharedDB` et écrivent en base joueur : vérifié par le test à base partagée en lecture seule),
`service/match_exclusion_service.go`, `service/openspartan_import_service.go` (`acquireShared` avant `openspartan.Open`),
`handlers/openspartan_import.go`, `handlers/media_upload.go` (`io.ReadAll` par fichier), `media/hls.go` (`BuildHLS`, seul
point d'entrée du transcodage), `server_apiv1.go` (recherche de gamertags et `recompute_coefficients`).

**Statuts** : L5-1 à L5-7 `[x]`. Aucun `[!]`, aucun `[~]`.

- **L5-1** `platform/auth/attempt_store.go` : `GetOrCreate(sessionID, ip)` refuse (`ErrTooManyAttempts`) au-delà de
  `MaxPendingAttempts` (20) tentatives en attente et `MaxPendingPerIP` (3) ; une tentative déjà en attente pour la session
  est resservie hors plafond (rafraîchissement de la page de connexion), une tentative terminée libère sa place.
  `handlers/auth.go` : 429 `too_many_device_flows` + `Retry-After: 60`, `InitDeviceFlow` jamais appelé. Front : message
  FR + EN `common.xbox_login.err_busy` sur `XboxLoginPage` (3 chemins) et `StepDeviceCode`. Tests :
  `attempt_limits_test.go`, `auth_device_flow_limits_test.go`, `XboxLoginPage.test.tsx` (FR, EN), `SetupPage.test.tsx`.
- **L5-2** `assets.Gate` (nouveau) devant le resolver pour les cinq routes HTTP : stock local d'abord (inchangé pour les
  images d'une page), cache négatif borné (4096 entrées, 10 min), puis `Catalog.Classify`. `MetadataCatalog` : médailles
  (`medal_definitions`, `waypoint_medals_raw`) et cartes (`maps_catalog`, `asset_translations`, `map_images_registry`) ;
  titre vérifié par le registre des titres (`TitleKnown`, aucun slug en dur). Identifiant inconnu : 404 sans appel sortant.
  Kinds sans référentiel local (badge de défi, images de passe de combat, images de joueur : identifiants dérivés d'une
  charge amont) : récupérés à distance pour un appelant connecté seulement (`middleware.IsAuthenticated`), un anonyme est
  servi en local. Fournisseur Halo des cartes construit une fois (son limiteur était recréé par requête) ; garde-rail
  `archlint/no_request_scoped_halo_provider_test.go`. Pas de limite de débit réintroduite sur les images. Tests :
  `gate_test.go`, `catalog_metadata_test.go`, `assets_gate_test.go`, `provider_limiter_shared_test.go`.
- **L5-3** `MatchRecomputer` lit la base partagée (`Provider.Get` ou `OpenReadForQuery`), ne prend plus de writer partagé ;
  `MatchExclusionService` : no-op si l'état demandé est l'état courant (`MatchExclusionRepository.IsExcluded`, vue
  `_latest`), recalcul quand même après un recalcul en échec (`RecomputePending`, contrat de reprise conservé) ;
  `coalesce.Runs` : un recalcul à la fois par joueur, les demandes déposées pendant l'exécution donnent UN rattrapage.
  Aucune écriture sur la base partagée : le test fournit une base ouverte en lecture seule (toute écriture échouerait).
  Tests : `match_recomputer_shared_readonly_test.go`, `match_exclusion_service_test.go`, `runs_test.go`,
  `TestMatchExclusionRepo_IsExcluded_LatestState`.
- **L5-4** `OpenSpartanImportService.Import` valide le fichier (ouverture, propriétaire, nombre de matchs, alias) AVANT
  `acquireShared`. `StartImport` : 409 `import_already_active` si un import est actif pour le compte, refusé avant de
  lire le corps puis recontrôlé atomiquement (`jobs.Store.CreateIfNoActive`). Front : message FR + EN
  `common.onboarding.import_err_already_active`. Tests : `openspartan_import_writer_order_test.go`,
  `openspartan_import_single_test.go`, `create_if_no_active_test.go`, `OpenSpartanImportCard.test.tsx`.
- **L5-5** `domain.UploadedFile` porte `Open()` au lieu de `[]byte` ; `storeUploadedFile` copie en flux vers un fichier
  temporaire du dossier des captures (hash au fil de l'eau, `ops.ContentHasher`), déduplique puis renomme ; `ops.HashBytes`
  (devenu mort) retiré. `platform/slots` : sémaphore ; `PostUploadMedia` attend une place parmi 2 envois en vol (30 s) puis
  répond 503 `upload_busy` (`Retry-After`) ; `media.BuildHLS` tient l'unique place ffmpeg et un délai maximal de 45 min.
  Le front envoie tous les fichiers dans UNE requête (`useUploadMedia`) et désactive le bouton pendant l'envoi : aucun
  envoi parallèle d'un même onglet ; message FR + EN `common.media.upload_busy`. Tests : `media_service_upload_stream_test.go`,
  `media_upload_slots_test.go`, `ffmpeg_slot_test.go`, `slots_test.go`, `UploadButton.test.tsx`.
- **L5-6** `?live=1` ignoré sans session authentifiée (`middleware.IsAuthenticatedCtx`, prédicat passé à
  `registerGamertagHuma`) ; `recompute_coefficients` déplacée dans `EngagementHandler.MountAdmin` sous `RequireAuth` +
  `RequireAdmin`. Aucun écran web ne l'appelle (seul l'instantané `contract-surface.snapshot.json` la cite). Seul appelant
  web de `live=1` : `useGamertagSuggestions` (pages connectées). Tests : `gamertag_live_auth_test.go`,
  `recompute_coefficients_admin_test.go` (chi.Walk sur le routeur réel).
- **L5-7** (a) `bare_routes_ratchet_test.go` : `publicRoutesAllowlist` passe à `publicRoute{class, reason, proof}` (47
  entrées : 38 `cheap`, 9 `outbound`/`disk`) ; `public_routes_cost_class_test.go` exige un test de preuve nommé et existant
  pour `outbound`/`disk`. (b) `archlint/no_shared_writer_on_request_path_test.go` : prise de writer partagé interdite dans
  `internal/api`, `internal/service`, `sync/match_recomputer.go` hors allowlist datée à compte décroissant. (c)
  `archlint/no_readall_on_upload_test.go` : `io.ReadAll`/`ReadFrom` interdit dans un fichier qui manipule du multipart.

Écarts au texte, justifiés : L5-1 plafonne aussi le device-flow dans `StepDeviceCode` (même appel) ; L5-3 ajoute
`RecomputePending` (sans lui, la reprise après un recalcul en échec devenait un no-op silencieux) ; L5-5 déplace le cache
de coalescence vers `platform/coalesce` (gel de la racine `internal/sync` à 79 fichiers) ; le test existant
`TestMatchExclusionService_SetAndList` est ajusté (démarquer un match jamais exclu est désormais un no-op : `m2` est exclu
avant d'être démarqué, attendu 3 recalculs au lieu de 2), contrat changé par le plan lui-même. L'allowlist des littéraux
français est RESSERRÉE (`auth_xbox_oauth.go` 6 -> 5, issu de L4b, et `seasons_catalog.go` 4 -> 3, D-L1-5) pour faire passer
le gate : baisses, jamais d'élargissement.

**Preuves du rouge** (sorties gardées dans `%TEMP%\l5\`) :
- L5-1 `go test ./internal/platform/auth/ -run AttemptStore` : FAIL, 21e tentative sans erreur, 4e tentative de la même IP
  sans erreur ; handler : panique (tentative nulle) au premier refus avant gestion de l'erreur ; front
  `XboxLoginPage.test.tsx -t "429 "` : 2 FAIL (texte du serveur affiché).
- L5-2 `go test ./internal/assets/ -run 'TestGate|TestNegativeCache'` : 5 FAIL (identifiant inconnu servi, anonyme servi,
  hit local qui consulte le fetcher, 404 amont non mémorisé) ; `TestMetadataCatalog_Classify` : 9 sous-tests FAIL ;
  archlint fournisseur : FAIL avec `halo.NewHaloProvider()` remis dans la closure (`server_apiv1.go:1414`).
- L5-3 `TestMatchRecomputer_NeverAcquiresTheSharedWriter` : FAIL « writer partagé acquis 1 fois, attendu 0 » ;
  `TestMatchExclusionService_UnchangedStateIsANoOp` : FAIL « écritures=1 recalculs=1, attendu 0/0 ».
- L5-4 `TestOpenSpartanImport_SharedWriterOnlyAfterFileValidation` : 2 FAIL (writer pris pour un fichier invalide et pour un
  fichier d'un autre propriétaire) ; `TestStartImport_SecondImportWhileActiveReturns409` : 202 au lieu de 409 ; front : 2 FAIL.
- L5-5 `UploadButton.test.tsx` : 2 FAIL ; sans la prise de place, `TestUploadHandler_(AtMostTwo|Busy)` ne termine pas
  (aucun 503, le second envoi attend le service bloqué : arrêté au délai). Les tests du service
  (`Open` au lieu de `Data`) échouaient à la compilation avant l'écriture du correctif.
- L5-6 `TestRecomputeCoefficients_RequiresAdmin` : FAIL (route sans `RequireAdmin` ni `RequireAuth`) ;
  `TestGamertagSearch_LiveIgnoredWithoutAuthenticatedSession` : 2 FAIL (repli live armé pour un anonyme).
- L5-7 : classe de coût (nom de preuve inexistant), writer partagé (prise remise dans `match_recomputer.go`), `ReadAll`
  (ajouté à `media_upload.go`) : chacun FAIL, source restaurée ensuite (`git diff` vide).

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`, Go 1.26.9) : voir « Gates » du compte rendu ;
détails ci-dessous.
- Gate du lot : `go test ./internal/api/... ./internal/assets/... ./internal/service/... ./internal/platform/auth/...
  ./internal/archlint/ -count=1` : exit 0, 17 paquets ok, 0 `--- FAIL:`. Premier passage rouge : trois tests d'archlint
  (gel de la racine `internal/sync` à 79 fichiers, empreinte de libellé de carte reconnaissant la requête de présence du
  catalogue, littéraux français neufs), corrigés dans le commit `89d64d0b2`.
- Intégration : `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/platform/duckdb/...
  -count=1` : exit 0, 18 paquets ok, 0 `--- FAIL:` ; `no_art_patterns_test.go` non modifié.
- Gate commun : `go build ./cmd/server/` exit 0 ; `go vet` (paquets touchés) exit 0 ;
  `golangci-lint run --new-from-rev=6e8df99c7` et forme CI `--new-from-merge-base=origin/main` : 0 issues ;
  `go run ./cmd/openapi-gen` : aucun diff (aucune route Huma ajoutée, `recompute_coefficients` garde son opération) ;
  `OpenAPI`/`Contract` dans la suite `./internal/api/` : ok.
- Web : purge de `node_modules/.tmp`, `npm run typecheck` exit 0, `npm run lint` exit 0 (0 erreur, 26 avertissements
  préexistants), `npm run test:run` exit 0 : 924 fichiers passés / 5 ignorés, 9668 tests passés / 23 ignorés ;
  `lint:fields`, `lint:colors`, `lint-cross-feature-imports`, `knip-ratchet`, `lint-contract-ratchet` : exit 0.
- Flake préexistant (D-L5-3) : `TestBootstrapBuild_WritablePlayerSlugs` échoue 4 fois sur 10 (ordre des slugs de l'admin),
  fichier non touché par le lot ; vert au passage du gate.
- Hooks pre-commit (gofmt, check-merge-conflict, docs-fr-sync, gitleaks, go-vet) verts sur les huit commits.

**Non-régression (règle 9)** :
- L5-1 : un rafraîchissement de la page de connexion garde son cookie : la tentative en attente est resservie hors plafond
  (test `reprise avec le cookie`). Trois démarrages simultanés par IP : trois onglets sans cookie commun ou trois
  appareils derrière une même IP ; au-delà, message « Réessaie dans une minute ».
- L5-2 : images de médailles servies par `/static/medals` (le front n'appelle pas la route `/assets/medals`) ; cartes :
  URL générée par le serveur à partir de la carte du match, présente dans `maps_catalog`/`asset_translations` (sur les
  métadonnées préconstruites : 4 cartes déjà récupérées sur 4 reconnues). Risque résiduel : une carte toute neuve absente
  des trois tables répond 404 (placeholder) jusqu'au prochain rafraîchissement du catalogue.
- L5-3 : l'écran d'exclusion n'attend rien de plus ; un clic répété sur le même état répond 204 sans recalcul.
- L5-4 : seul l'écran d'onboarding appelle la route ; un second envoi pendant un import montre le message dédié.
- L5-5 : tous les fichiers dans une seule requête ; 2 envois simultanés par instance au maximum (un troisième attend jusqu'à
  30 s) ; les transcodages se mettent en file (un ffmpeg à la fois), le job reste « running » pendant l'attente.
- L5-6 : `live=1` n'est émis que par la saisie de gamertag des pages connectées ; un anonyme reçoit les résultats locaux.

**Découvertes** : D-L5-1 à D-L5-6 (§6), non traitées.

Commits : `d17fff56f` (L5-1), `2e12e8dfd` (L5-2), `bac4319a9` (L5-3), `ac39b1150` (L5-4), `8b9a6c2f0` (L5-5),
`6b4637537` (L5-6), `bff0e94c0` (L5-7), `89d64d0b2` (gate archlint), puis le commit de clôture (plan, thought_log).
