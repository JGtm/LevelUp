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

1. **Ordre strict** L0 → L4 → L4b → L5 → L5b → L5c → L6 → L7 → L8 → L9 → L10 → L11. Un lot est **clos** quand : gate exact passé, tous les items statués,
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

- [x] L5b-1 (D-L5-3, régression du chantier : `writable_player_slugs` introduit en L3-4)
      `TestBootstrapBuild_WritablePlayerSlugs` instable (11 échecs sur 20 relevés par le
      superviseur le 2026-10-16) : ordre de sortie rendu déterministe dans le code (tri), pas dans
      le test seul ; preuve : `-count=50` vert.
- [x] L5b-2 (D-L5-1, même classe que C3-3) `PUT /players/{slug}/friends` : `RecomputeIsWithFriends`
      ne prend plus le writer partagé si ses écritures vont en base joueur (sinon : justification
      écrite et coalescence par joueur comme L5-3) ; entrée correspondante retirée de l'allowlist du
      garde-rail `no_shared_writer_on_request_path_test.go` ; tests rouge puis vert.
- [x] L5b-3 (D-L5-2) Autres usages de ffmpeg (miniatures, remux WebM, migration audio) : même place
      unique (`platform/slots`) et délai maximal que L5-5 ; garde-rail : tout `exec.CommandContext`
      de ffmpeg passe par la place (test grep/AST).
- [!] L5b-4 (D-L5-4 ; **décision de l'utilisateur du 2026-10-16 : au backlog**, `.ai/BACKLOG.md` « [data/LUSR] ») `batchComputeLUSR` est aussi le chemin du
      post-sync normal (`engine_postsync_scoring.go:196`) : le recalcul d'exclusion suit le même
      chemin que la synchronisation ; rien à faire. Cocher `[x]` en recopiant cette vérification.
      **Non coché** : la vérification n'est pas reproductible sur le défaut de production (voir
      journal L5b et D-L5b-1).
- Gate : `go test ./internal/service/ -run TestBootstrapBuild_WritablePlayerSlugs -count=50` ;
  `go test ./internal/api/... ./internal/service/... ./internal/media/... ./internal/archlint/` ;
  `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` si L5b-2 touche le sync.

### L5c — Corrections issues de L5b — Sonnet — effort S

- [x] L5c-1 (D-L5b-3, régression du chantier) La lecture remuxée d'un clip ne tient plus l'unique
      place ffmpeg pendant tout le flux : pool dédié aux remux (2 places, délai 45 min, 503 +
      `Retry-After` au-delà de 30 s d'attente), distinct de la place des transcodages et miniatures ;
      tests : deux lectures remuxées simultanées servies, un transcodage n'attend pas un remux ;
      garde-rail L5b-3 adapté sans assouplissement (tout lancement passe par un des deux pools).
- [x] L5c-2 (D-L5b-4) `PUT /players/{slug}/friends` : recalcul coalescé par joueur
      (`platform/coalesce`, comme L5-3) ; test « N PUT rapprochés → au plus 1 recalcul en cours + 1
      rattrapage ».
- [x] L5c-3 (D-L5b-2) ffprobe : délai maximal (même lanceur ou contexte borné), sans place.
- [x] L5c-4 (D-L5b-5) Joueur par défaut du bootstrap : vérifier que l'ordre déterministe ne change
      pas le joueur par défaut d'un utilisateur connecté (son profil possédé d'abord) ; sinon le
      rétablir explicitement ; test.
- Gate : `go test ./internal/api/... ./internal/media/... ./internal/service/... ./internal/sync/... ./internal/archlint/`
  ; `go test -tags=integration -p 1 ./internal/sync/...` si L5c-2 touche le sync.

### L6 — Fichiers et chemins (C6-1, C6-2, C6-3) — Opus — effort M

- [x] L6-1 Route des médias : racine legacy restreinte à `players/{slug}/captures/` ; liste
      blanche des extensions média ; test table-driven `.duckdb`, `.wal`, `.json` → 404 sur chaque
      racine.
- [x] L6-2 Slug : la garde du groupe `/players/{player_slug}` refuse `.`, `..`, `/`, `\` et tout
      slug hors alphabet, en démo comprise.
- [x] L6-3 (+ D-L3-3 : forme unique `title.IsValidMatchID`, garde-rail) `match_id` : validation fail-closed (hexadécimal et tirets) à la frontière HTTP du rejeu
      et dans les constructeurs de chemin du `PathResolver` qui l'utilisent (rejeu, faits de film,
      raster tactique) ; test « le chemin reste sous sa racine » avec `/`, `\`, `..`, valeur longue.
- [x] L6-4 `/static/` : plus de listing de répertoire (404).
- [x] L6-5 (concordance par famille, voir journal) Envoi : `http.DetectContentType` doit concorder avec l'extension, sinon 415.
- Gate : `go test ./internal/api/... ./internal/domain/title/... ./internal/service/...`.

### L7 — Gardes de chemin et CSRF (C10-1) — Opus — effort S

- [x] L7-1 Test rouge : `POST …/coach/proposals/..%2F..%2F..%2F..%2Finternal%2Fx/dismiss` avec
      Origin étrangère → 403 `csrf_rejected` (pile réelle).
- [x] L7-2 `middleware/csrf.go` : `!AmbiguousPath(r) &&` devant `isExemptPath`.
- [x] L7-3 (+ limiteur corrigé, voir journal) Garde-rail : test table-driven qui passe les mêmes chemins ambigus (`%2F`, `%5C`,
      `%2e%2e`, `//`, `/./`) à toutes les gardes qui décident sur un chemin (CSRF, démo,
      écritures, limiteur, rejeu, ouvrier).
- [x] L7-4 (D-L1-2) `handlers/replay_local_gate.go` : `isLocalRequest` refuse un hôte vide (garde
      fermée par défaut) ; tests adaptés à un transport qui fournit une adresse.
- Gate : `go test ./internal/api/... -run 'CSRF|Ambiguous|Transverse'` puis `go test ./internal/api/...`.

### L8 — Démo publique (C8-1, C8-2) — Sonnet — effort M

- [x] L8-1 En démo, `RequireAdmin` et `RequireAuth` répondent 403 sauf allowlist nommée
      (`GET /settings`, et ce que l'interface de démo lit réellement, à vérifier) ; `/debug/vars`
      403 en démo ; ratchet qui parcourt le routeur de démo et exige 403 sur toute route portant
      `RequireAdmin` hors allowlist datée.
- [x] L8-2 Identités : témoin disque adossé à `DemoLayout`/`DemoRoster`.
- [x] L8-3 CLI : six lectures `cfg.DemoMode` → `cfg.demoState()` ; garde-rail grep interdisant
      `.DemoMode` dans `internal/config/` hors `config_demo.go`.
- [x] L8-4 `suggest-next` classé lecture (`readOnlyPostPrefixes`, motif par suffixe) ; ratchet démo
      mis à jour.
- [x] L8-5 E2E `media-like-bug.spec.ts` : cas « like » marqué hors démo (ou retiré s'il n'a pas
      d'autre cible).
- [x] L8-6 `weapon_kills` conditionnée par titre dans la liste d'extraction ; `seed_demo.go:988`
      n'avale plus l'erreur (journalisée).
- [x] L8-7 ERROR au boot sur la metadata Halo 5 : requête fautive identifiée
      (`title_asset_drawer_loader.go`) et corrigée (absence attendue → pas d'ERROR, ou table créée
      par la fixture).
- [x] L8-8 Séquence Tactique : `ReplayGate` lit `match_id` dans le corps pour cette route ; test
      garde démo + route.
- [~] L8-9 (couvert par L10-2, `levelup-demo` porte la variable) `docker-compose.yml` : `LEVELUP_TRUST_PROXY_HEADERS=true` sur le service démo ;
      garde-rail lisant le compose.
- [x] L8-10 `netguard.Check` dans `platform/auth/xbox_device_code.go` ; raison fausse de
      `netguard_coverage_test.go:43` retirée.
- [!] L8-11 (provisionnement au déploiement, V-3 ; runbook vérifié à jour) `data/demo_films/` : statut `[!]` attendu — provisionnement sur le VPS au déploiement
      de la v7.5 (geste annoncé, commande au runbook) ; vérifier que le runbook est à jour.
- [x] L8-12 (D-L1-4) `packaging/nginx/demo.conf` : `limit_req_status 429;` (la limite répond 503, lu comme une panne) ; appliqué sur le VPS par le superviseur (opération annoncée) ; garde-rail nginx étendu à la démo.
- Gate : `go test ./internal/api/... ./internal/config/... ./internal/ops/... ./internal/platform/...` ;
  `cd apps/web && npx playwright test e2e/media-like-bug.spec.ts --list`.

### L9 — Fuites d'information (C7) — Sonnet — effort M

- [x] L9-1 `/health` public réduit à `{status}` (sondes Docker et `deploy.sh` vérifiées) ; détail
      réservé à l'admin.
- [x] L9-2 `/readyz` et le 503 de `/health` : plus d'`err.Error()` dans le corps.
- [x] L9-3 Device-flow : `error_detail` retiré de la réponse (code seul) ; front adapté (FR + EN).
- [x] L9-4 Jobs et notifications de sync : message fixe + code, détail au journal.
- [x] L9-5 Les 121 `err.Error()` renvoyés au client en 4xx (`NewError`/`writeError`) remplacés
      par un message fixe par code d'erreur (helper unique), le détail partant au journal ;
      relevé préalable des messages que le front affiche (`apps/web/src`, `ApiError.message`) pour
      garder un texte utile ; garde-rail `internal/archlint/err_error_response_test.go` à **zéro**
      (allowlist datée seulement pour un cas justifié par écrit).
- [x] L9-6 En-tête CSP minimal sans risque pour le SPA : `object-src 'none'; base-uri 'self';
      frame-ancestors 'none'` (scripts non restreints, décision de l'ADR à venir) ; test sur `/`,
      404, 429, `/static`.
- [x] L9-7 (D-L4b-1, doctrine D5) Réinitialisation du mot de passe d'un compte par l'admin
      (`handlers/admin.go`) : révoque toutes les sessions de ce compte (`Store.RevokeAccount`) ;
      test « session du compte réinitialisé → 401 ».
- [x] L9-8 (D-L4b-2) `GET /auth/xbox/login` : refus redirigés vers `/login?oauth_error=<code>`
      comme le callback (L4-14), plus de JSON brut ; test.
- [x] L9-9 (D-L4b-3) Refus d'une reconnexion Xbox depuis une session déjà connectée : le message
      n'est plus perdu (`__root.tsx`) ; test du composant.
- Gate : `go test ./internal/api/... ./internal/archlint/ -run 'Health|Ready|ErrError|CSP|SecurityHeaders|DeviceFlow'` ;
  front si L9-3.

### L10 — Conteneurs et ouvrier de rejeu (C9-4) — Sonnet — effort S

- [!] L10-1 (modification faite, `cap_add: [CHOWN, DAC_READ_SEARCH, SETUID, SETGID]` ; validation du démarrage à faire sur la démo du VPS, V-4) `docker-compose.yml` : `security_opt: [no-new-privileges:true]`, `cap_drop: [ALL]`,
      `cap_add` limité au strict nécessaire de l'entrée (`gosu` : `CHOWN`, `SETUID`, `SETGID`,
      plus ce que la mesure montre) sur les deux services ; image construite et démarrée en local
      si Docker est disponible (démarrage, `/health`, écriture sous `data/`), sinon `[!]` et
      validation à faire sur le conteneur de démo du VPS par le superviseur (opération annoncée)
      avant la prod.
- [x] L10-2 Garde-rail : test lisant `docker-compose.yml` qui exige `no-new-privileges` et
      `cap_drop: [ALL]` sur chaque service, et `LEVELUP_TRUST_PROXY_HEADERS` sur chaque service
      publié derrière nginx (couvre L8-9).
- [!] L10-3 (délai d'observation : mesure des pics par le superviseur, V-6) Limites mémoire : relevé des pics réels (`memory.peak` des cgroups des deux conteneurs
      sur le VPS, lu par le superviseur à la clôture puis après une semaine) ; `mem_limit` fixé à
      partir de la mesure. Statut attendu `[!]` (délai d'observation) jusqu'à la mesure.
- [x] L10-4 (script, unité, runbook ; partie nginx `[!]` : IP du serveur de calcul à poser à l'activation, V-5) Ouvrier de rejeu : `scripts/deploy-worker.sh` installe l'unité par copie appartenant à
      root (fin du lien vers un fichier modifiable par `deploy`) et refuse un lien ; unité
      durcie (`ProtectSystem=full`, `ProtectHome=read-only`, chemins d'écriture déclarés) ;
      `packaging/nginx/levelup.conf` : `location /api/v1/internal/` restreint à l'IP du serveur
      de calcul (`allow …; deny all;`), appliqué sur le VPS à l'activation de l'ouvrier.
- [x] L10-5 (D-L0-3, décision de l'utilisateur du 2026-10-15) `.github/dependabot.yml` : écosystème `docker` (répertoire racine, images du `Dockerfile`), même cadence que les autres.
- [x] L10-6 (D-L0-3, décision de l'utilisateur du 2026-10-15) Garde-rail archlint : la version de l'image `golang:X.Y.Z-…` du `Dockerfile` est égale à la ligne `go` de `apps/go-api/go.mod` ; vu rouge sur une divergence simulée.
- Gate : `go test ./internal/archlint/ -run 'Compose|Nginx|Dockerfile'` ; `bash -n scripts/deploy-worker.sh`
  ; `shellcheck scripts/deploy-worker.sh` si disponible ; `systemd-analyze verify` si disponible.

### L11 — Reliquats de chemins et de cohérence (découvertes L6, L7, L9) — Opus — effort S

- [x] L11-1 (D-L7-1 ; sur pièces le slug n'atteignait pas le disque, voir journal) Aperçu OG (`wire/og_inject.go`) : slug du chemin validé (`domain.IsValidPlayerSlug`) avant toute lecture ; vérifier sur pièces s'il atteint le disque ; test chemin piégé.
- [x] L11-2 (D-L7-2) `/profiles/{player_slug}/titles/{slug}` : slug de titre validé contre le registre des titres (404 sinon), sans branchement sur un slug particulier ; test.
- [x] L11-3 (D-L7-3) `AmbiguousPath` dans son propre fichier du middleware ; garde-rail : tout middleware qui décide sur `r.URL.Path` (préfixe, exemption) appelle `AmbiguousPath` (allowlist datée pour les journaux seulement) ; vu rouge sur une garde témoin.
- [x] L11-4 (D-L6-1) Action admin de cuisson (`/admin/actions/replay-build/*`) : `match_id` hors forme → 400 (`title.IsValidMatchID`) ; test.
- [x] L11-5 (D-L6-3 ; trois routes relevées, voir journal) Routes admin portant un slug de joueur dans le chemin : forme validée (`domain.IsValidPlayerSlug`) ; liste relevée sur pièces ; test table-driven.
- [x] L11-6 (découverte L9 n°3) `StepDeviceCode` : clés d'erreur alignées sur les codes réellement émis (`device_flow_acquire_error`, `identity_missing`, …), FR + EN ; test.
- [x] L11-7 (découverte L9 n°6, règle 7 de CLAUDE.md) `WatcherAttempt.ErrorDetail` écrit mais jamais sérialisé : supprimé avec ses écritures.
- [x] L11-8 (D-L6-2, règle 6 de CLAUDE.md) Listes de formats média (`ops` et `media`) : source unique + garde-rail interdisant une copie.
- Gate : `go test ./internal/api/... ./internal/archlint/ ./internal/media/... ./internal/ops/... ./internal/platform/auth/...` ; web si L11-6 (typecheck, lint, test:run).

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
- [x] F-1a Corrections de la ronde 1 de la revue adversariale (2026-10-16 ; 3 relecteurs : accès 0 P0/P1,
      entrées 2 P1 + 6 P2, couverture 1 P1 + 7 P2 ; le P1 « images de carte » trouvé par deux relecteurs) :
  - [x] R1-1 (P1) `GET /api/v1/assets/maps/…/image?v=` : la variante n'est pas jugée par le catalogue et chaque
        valeur neuve rate stock local et cache négatif → appels Halo signés illimités pour un anonyme. Forme
        stricte de `v`, et aucune récupération distante par variante neuve pour un anonyme ; test sur la VRAIE
        route des cartes ; preuve de classe de coût corrigée (`bare_routes_ratchet_test.go`).
  - [x] R1-2 (P1, préexistant, en prod ; atténué sur le VPS le 2026-10-16 par nginx) `assets/store_localfs.go`
        `Path` : `v` (et tout composant issu de la requête) entre dans `filepath.Join` → lecture de `*.png`
        arbitraires par un anonyme. Validation des composants + confinement sous la racine (fail-closed) ;
        test chemin piégé ; snippet nginx versionné (`packaging/nginx/`) et garde-rail nginx étendu.
  - [x] R1-3 (P1, régression du chantier) Miniature générée dans la requête d'envoi : elle attend l'unique
        place des transcodages (jusqu'à 45 min) en tenant le verrou d'index et une place d'envoi. Elle ne
        doit plus attendre un transcodage (pool propre ou génération hors requête) ; test.
  - [x] R1-4 (P2) Attente d'une place d'envoi (30 s) au-delà du `ReadTimeout` (15 s) → 413 trompeur :
        attente compatible avec l'échéance de lecture (ou échéance prolongée), 503 + `Retry-After` garanti.
  - [x] R1-5 (P2) Exclusion pendant une bascule de la base partagée : `ErrSwapTimeout` → 503 (pas 500).
  - [x] R1-6 (P2) `platform/coalesce` : le recalcul ne dépend plus du contexte du premier demandeur
        (contexte détaché), commentaire conforme ; test d'annulation du premier demandeur.
  - [x] R1-7 (P2) Exclusion : lecture de l'état et no-op décidés sous le verrou de la base joueur (course
        exclure/réinclure) ; test.
  - [x] R1-8 (P2) Porte des assets : une erreur de lecture du catalogue n'est pas mémorisée comme « inconnu »
        (pas de cache négatif sur erreur) ; test.
  - [x] R1-9 (P2) Front : « Réassocier » masqué sur le clip d'autrui (onglet Médias d'un match, rail de
        l'accueil), aligné sur la règle serveur ; test du composant.
  - [x] R1-10 (P2, règle 3) `os.Remove` avalés (`openspartan_import.go`, `media_service_upload.go`) : échec
        journalisé.
  - [x] R1-11 (P2) Garde-rails renforcés : `no_readall_on_upload` voit le service (`UploadedFile.Open`) et les
        imports aliasés ; `client_ip` interdit la lecture directe de `True-Client-IP`/`X-Forwarded-For`/
        `X-Real-IP` hors middleware d'IP ; `session_player_slug` voit aussi `CurrentPlayerSlug` (exception
        datée pour un affichage) ; test d'envoi concurrent rendu déterministe (sans `time.Sleep`) ; chacun vu
        rouge.
- [x] F-1b Ronde 2 de la revue (2026-10-16 ; 1 P1 + 3 P2, dont 1 réfuté par le superviseur : la démo coupe toute
      sortie réseau par `netguard`, `discovery_client.go:250`). Bornes de boucle atteintes : décision de
      l'utilisateur du 2026-10-16 « corrige tout, versions réelles » :
  - [x] R2-1 (P1) `http.NewResponseController(w).SetReadDeadline` rend `ErrNotSupported` en production : les
        enveloppes `sessionResponseWriter` (`middleware/session.go`) et `statusResponseWriter`
        (`middleware/slog_logger.go`) n'exposent pas `Unwrap() http.ResponseWriter`. Les rendre transparentes ;
        test de l'attente d'envoi à travers la VRAIE pile (WithSession + SlogLogger) ; garde-rail : toute
        enveloppe de `ResponseWriter` du paquet middleware implémente `Unwrap`.
  - [x] R2-2 (P2) Recalcul coalescé (`platform/coalesce`) : contexte détaché AVEC échéance (10 min) ; l'arrêt du
        serveur attend la fin des recalculs en vol avant `duckdb.CloseAll` ; commentaire conforme ; tests.
  - [x] R2-3 (P2, décision de l'utilisateur) Images de carte : une version `v` présente dans de vrais matchs
        de la base partagée est récupérable par tous (anonyme compris) ; une version inventée reste refusée
        sans appel sortant ; test sur la vraie route.
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
      que la conf nginx versionnée (L1-7) correspond à la conf en place (filtre
      `snippets/levelup-assets-v.conf` compris, versionné par F-1a R1-2 depuis la conf du VPS).

- [x] V-4 (L10-1, fait le 2026-10-16 par le superviseur : conteneur jetable de l'image de prod sur un dossier de test avec `auth` 0700, `film_facts` 0750 et un fichier root ; avec CHOWN, DAC_READ_SEARCH, SETUID, SETGID + `no-new-privileges` + `cap-drop ALL` : `chown` code 0, `gosu` OK, écriture OK, ffmpeg OK ; sans DAC_READ_SEARCH : « cannot read directory /app/data/auth », capacité nécessaire confirmée ; démarrage complet du serveur à observer au déploiement, démo d'abord) Avant que le lot n'atteigne la prod : valider sur le VPS, sur une copie de répertoire
      (jamais les données en service), l'entrée du conteneur avec `--cap-drop ALL` + les quatre
      capacités et `no-new-privileges` (`chown -R` → code 0, `gosu` OK) ; remplacer `DAC_READ_SEARCH`
      par `DAC_OVERRIDE` si le `chown` échoue sur un dossier 0700. Opération annoncée.
- [ ] V-5 (L10-4) Avant que le lot n'atteigne `main` : sur le serveur de calcul, convertir l'unité de
      l'ouvrier de lien en copie root (runbook §7), `systemd-analyze verify` ; sinon le job
      `deploy-worker` passe au rouge. À l'activation : `allow <IP>; deny all;` sur
      `/api/v1/internal/`. Opération annoncée.
- [!] V-6 (L10-3 ; premier relevé du 2026-10-16 : prod 1 134 Mio de pic depuis le redémarrage de 04:00 UTC, démo 334 Mio depuis le 09/10, hôte 1 867 Mio ; second relevé dans une semaine) Relevé `memory.peak` des deux conteneurs à la clôture puis une semaine après ;
      `mem_limit` proposé à l'utilisateur à partir de la mesure.

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
  de sécurité est ouvert. **Traité en L7-4** (2026-10-16).
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
  et poser le garde-rail (règle 6). **Traitée en L6-3** (forme unique `title.IsValidMatchID` : `domain` importe
  `domain/title`, l'inverse ferait un cycle).
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

- **D-L5b-1** (L5b) Le recalcul d'exclusion n'est PAS le chemin du post-sync en production. `runLUSRV1UnderRead`
  (`engine_postsync_scoring.go:181`) saute `batchComputeLUSR` quand `IsLUSRV2Canonical()` ; or `DefaultLUSRModeIfUnset`
  (`sync/skill/skill_v2_canonical.go:58`, appelée par `cmd/server/main.go:1186`) pose `LEVELUP_LUSR_CANONICAL=LUSR_V2` au boot serveur quand la variable est absente.
  Le post-sync normal écrit alors le LUSR par le chemin v2, tandis que `MatchRecomputer.recompute` (`match_recomputer.go:126`)
  appelle toujours `batchComputeLUSR` (v1, `force=true`, `rating_type='LUSR'`). Même constat que D-L5-4, jamais tranché :
  à confirmer par lecture de la prod (une exclusion réécrit-elle des lignes `LUSR` v1 par-dessus le canonique v2 ?). Hors périmètre
  sécurité ; décision à prendre avec l'utilisateur.
- **D-L5b-2** (L5b) **Traitée en L5c-3.** `ffprobe` n'a ni place ni délai maximal : `PlanRemuxWebM`/`probeAVStreams` (`media/remux.go`), `ProbeStreamsDetailed`
  (`media/hls.go`, appelé aussi sous la place de `BuildHLS`) et `probeVideoDuration` (`ops/media_thumbnails.go`). Non visé par le
  garde-rail `ffmpeg_single_launcher_test.go` (une sonde lit un en-tête) ; un fichier qui bloque la sonde bloque la requête jusqu'à
  l'annulation de son contexte.
- **D-L5b-3** (L5b) **Remux traité en L5c-1** (pool dédié) ; reste ouvert : l'attente des miniatures, voir D-L5c-1. Une lecture remuxée (`serveRemuxedWebM`) tient désormais l'unique place ffmpeg pendant TOUT le flux vers le
  client (jusqu'à 45 min), et attend au plus 30 s derrière un transcodage ou une autre lecture (503 `Retry-After: 30` sinon) :
  conséquence du texte de L5b-3 (« même place unique »). Deux lecteurs remuxés simultanés se sérialisent. Les miniatures d'un envoi
  (`GenerateThumbnails`) attendent aussi la place sans borne autre que leur contexte. Si l'usage réel le montre pénible, une place
  dédiée aux flux remuxés est un choix produit, non pris ici.
- **D-L5b-4** (L5b) **Traitée en L5c-2.** `RecomputeForPlayer`/`RecomputeAll` (`friends_orchestrator_service.go`) restent rejouables en boucle par le
  propriétaire sans coalescence par joueur : plus de writer partagé (les lecteurs ne sont plus bloqués), mais chaque PUT relance un
  recalcul complet sous le bail de la base joueur. Même défaut CPU que celui que `coalesce.Runs` absorbe pour l'exclusion.
- **D-L5b-5** (L5b) **Traitée en L5c-4.** `LoadPlayers` rend maintenant ses profils triés (titre, slug) : le « joueur par défaut » du bootstrap
  (`ownedPlayers[0]`, quand la session n'en choisit pas) est le premier par ordre alphabétique au lieu d'un profil aléatoire.

- **D-L5c-1** (L5c) Les miniatures d'un envoi (`generateAnimatedThumbnail` -> `RunFFmpeg`, pool des transcodages, sans `MaxWait`) attendent
  toujours la place des transcodages sans autre borne que leur contexte : une file de transcodages de 45 min chacun les retient.
  Seconde moitié de D-L5b-3, hors des items L5c-1 à L5c-4.
- **D-L5c-2** (L5c) `RunFFprobe` ne prend pas de place (décision de L5c-3) : le nombre de sondes simultanées n'est borné que par le
  nombre de requêtes (chacune au plus 30 s). Une salve de lectures remuxées lance autant de `ffprobe` de pré-vol que de requêtes.
- **D-L5c-3** (L5c) `GET /players` : `default_player_slug` ignore le joueur mémorisé en session (`CurrentPlayerSlug`), contrairement à
  `current_player` du bootstrap ; les deux peuvent différer quand la session a choisi un autre profil. Préexistant.
- **D-L5c-4** (L5c) `RecomputeForPlayer` : un demandeur absorbé par la coalescence rend 0 promu ; le journal du handler
  (`friends: recompute is_with_friends ... added`) affiche alors 0 pour lui, alors que la passe exécutée par un autre a promu et notifié.
  Cosmétique.

- **D-L6-1** (L6) `POST /admin/actions/replay-build/{run,enqueue}` (`handlers/admin_actions_replay_build.go`) accepte tout `match_id`
  non vide (admin seul) : aucune validation de forme à la frontière. Les constructeurs de chemin refusent désormais un identifiant
  mal formé (fermé par défaut), mais le job est créé et n'échoue qu'à l'écriture.
- **D-L6-2** (L6) `ops.supportedExtensions` (indexation) et `internal/media` (`VideoExtensions`, `ImageExtensions`) portent deux
  copies de la liste des formats (règle 6 tenue) sans garde-rail : une troisième copie ne serait pas détectée.
- **D-L6-3** (L6) La forme du slug n'est contrôlée que sur les routes qui portent `RequirePlayerOwnership` (groupes
  `/players/{player_slug}` et `/profiles/{player_slug}/titles/{slug}`). Des routes d'administration prennent aussi un slug ou un
  gamertag dans le chemin (`/admin/diag/appearance/{player_slug}`, `/admin/monitoring/lusr-gaps/{player}`) : non vérifiées.
- **D-L6-4** (L6) La racine legacy interne de `ServeMediaFile` est figée au titre par défaut (`PlayerCapturesDir(DefaultSlug, …)`) :
  un média legacy d'un autre titre rangé sous `players/{slug}/captures/` n'est pas servi par cette racine. Préexistant.

- **D-L7-1** (L7) `wire/og_inject.go` (`playerSlugFromPath`) lit le slug sur le chemin DÉCODÉ d'une page du SPA, sans
  `domain.IsValidPlayerSlug`, et le passe à `HomeCtx` (démo seulement, User-Agent de robot, que le client choisit). Un slug `..`
  ou à barre oblique inverse y arrive ; la résolution du profil le refuse-t-elle avant tout chemin disque ? Non vérifié.
- **D-L7-2** (L7) `/profiles/{player_slug}/titles/{slug}` : `TitleSlugFromPath` pose le `{slug}` de titre du chemin dans le
  contexte sans contrôle de forme ni d'appartenance au registre ; seul un admin traverse la garde de propriété pour un titre
  inconnu. Effet d'un titre `..` ou encodé dans le handler (`TitleSyncHandler`, `ProfileService`) non vérifié.
- **D-L7-3** (L7) `AmbiguousPath` vit dans `middleware/require_auth_mutations.go` alors que six gardes l'appellent désormais
  (CSRF, limiteur, démo en lecture seule, lectures gardées de la démo, garde d'écriture, action du visiteur) : son contrat
  gagnerait un fichier propre. Aucun garde-rail n'exige qu'une nouvelle garde qui décide sur `r.URL.Path` l'appelle (le test
  `ambiguous_path_guards_test.go` couvre les gardes existantes, pas une garde ajoutée demain).

- **D-L11-1** (L11) Le `?title=` des actions d'administration (`titleOrDefaultSlug`, `handlers/admin_monitoring.go:260` : cuisson
  `replay-build/{run,enqueue}`, convergence, synchronisation initiale, rapports de monitoring) est pris tel quel, sans contrôle
  d'appartenance au registre des titres ; il devient le titre du job et entre dans les constructeurs de chemin du `PathResolver`.
  Admin seulement ; effet d'un titre `..` non vérifié.
- **D-L11-2** (L11) `POST /admin/actions/convergence/run` et `/admin/actions/initial-sync/run` reçoivent le `player_slug` dans le
  CORPS sans contrôle de forme (`domain.IsValidPlayerSlug`) ; L11-5 ne couvre que les slugs du chemin. Admin seulement ; la
  résolution du joueur passe-t-elle par la liste des profils avant tout chemin disque ? Non vérifié.
- **D-L11-3** (L11) `archlint/no_french_label_literal_test.go` journalise une baisse à figer : `api/handlers/auth_xbox_oauth.go`
  compte 0 littéral français pour 5 alloués depuis L9-8 (`2769c00ce`) ; l'allowlist n'est pas resserrée (un `t.Logf`, non bloquant).
- **D-L11-4** (L11) Le crochet pre-commit `go-vet` affiche, pour tous les paquets qui dépendent de DuckDB, « build constraints
  exclude all Go files » (lancé sans CGO) et passe vert : ces paquets ne sont pas vérifiés par le crochet local. Le gate du lot
  lance `go vet` avec `CGO_ENABLED=1`.

- **D-F1a-1** (F-1a) `api/handlers/help.go:174,178` : deux `_ = os.Remove(tmpName)` (brouillon d'aide) avalent l'échec
  d'un retrait ; tolérés, datés, par l'allowlist de `archlint/no_discarded_remove_test.go` (hors du périmètre de R1-10).
- **D-F1a-2** (F-1a) `assets/store_localfs.go` (`PersistBinary`) : `_ = os.Remove(tmp)` après un renommage échoué, hors des
  racines du garde-rail de R1-10 (`internal/api`, `internal/service`).
- **D-F1a-3** (F-1a) Trois prédicats « UUID » coexistent : `analysis.IsRawAssetUUID` (expression stricte, réutilisée par
  R1-1), `sync.isUUIDLike` et `cmd/seed-ranked-playlists.isUUID` (longueur et nombre de tirets). Règle 6 dépassée, sans
  garde-rail.
- **D-F1a-4** (F-1a) Des dépôts reconnaissent une table absente au texte du message (`"does not exist"`,
  `"Table with name"` : `platform/duckdb/home_repo.go:164`, `ops/seed_demo.go:858`, `migration/helpers.go:216`, …) ;
  `duckdb.IsCatalogError` (type d'erreur du pilote, ajouté par R1-8) existe désormais.
- **D-F1a-5** (F-1a) Le lien « Associer » des vignettes (`MediaViewer.tsx`, `isOwnMedia`) reste masqué pour un admin sur le
  média non associé d'un autre joueur, que le serveur lui permet d'associer : plus strict que la règle serveur, sans risque.
- **D-F1a-6** (F-1a) Le recalcul d'exclusion tourne sous un contexte détaché sans échéance (R1-6) : son attente du verrou
  d'écriture du joueur (`dblease.AcquireWriterCtx`) n'est plus bornée par la requête. Un verrou jamais rendu retiendrait le
  recalcul de ce joueur jusqu'au redémarrage (les demandes suivantes attendent au plus leur propre échéance). Aucune borne
  propre au recalcul. **Traitée en F-1b R2-2** (échéance de 10 min par recalcul, `dblease.AcquireWriterCtx` reçoit ce contexte).
- **D-F1b-1** (F-1b) L'attente de l'arrêt (`drainCoalescedRecomputes`, jusqu'à 10 min, après un `srv.Shutdown` de 15 s) est
  plafonnée en pratique par le délai d'arrêt de l'hôte : `docker-compose.yml` ne pose pas de `stop_grace_period` (défaut Docker :
  10 s puis SIGKILL), et air tue le process en 5 s (`kill_delay`, `.air.toml`). Un recalcul plus long que ce délai est interrompu
  comme avant F-1b. Non modifié (hors périmètre, conteneurs).
- **D-F1b-2** (F-1b) `config.SharedReaderForTitle` retombe sur la base partagée du titre par défaut quand celle d'un autre titre ne
  s'ouvre pas (avertissement journalisé). Les versions jouées d'un titre additionnel seraient alors lues dans la base de Halo
  Infinite ; le couple (carte, version) devant correspondre exactement, aucune version inventée n'est acceptée, mais la lecture
  croise les titres. Préexistant (repli de `config`).
- **D-F1b-3** (F-1b) Une version jamais jouée d'une carte connue reste récupérée à distance pour un appelant de confiance
  (`VerdictUnverifiable`, comportement de R1-1 conservé). Toute URL produite par le serveur porte désormais une version jouée
  (`match_view_repo.go` lit `match_registry.map_version_id`) : ce repli ne sert plus aucune page ; son retrait est une décision
  de l'utilisateur.

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

### L5b - Compléments issus de L5 - 2026-10-16 - clos (un item non coché)

Base : `git log --oneline -1` = `af72aa583` avant le lot ; branche `feat/securite-2026-10-15`. Références rouvertes avant d'agir :
`config/config_players.go` (`loadPlayersV2`/`loadPlayersV3`, parcours de maps), `service/bootstrap_ownership.go`
(`writablePlayerSlugs`), `sync/friends_recompute.go` (`RecomputeIsWithFriends`/`Core`), `service/friends_orchestrator_service.go`,
`archlint/no_shared_writer_on_request_path_test.go`, tous les `exec.Command*` du module, `media/ffmpeg_slot.go`.

**Statuts** : L5b-1, L5b-2, L5b-3 `[x]` ; L5b-4 `[!]` (justification ci-dessous). Aucun `[~]`.

- **L5b-1** Cause : `LoadPlayers` construisait sa liste en parcourant des maps Go (ordre aléatoire) ; `writable_player_slugs` (et
  `available_players`) suivaient. Correctif dans le code : `sortPlayers` (titre, puis slug) appelé par les deux lecteurs
  (`config_players.go`). Le test du service n'a pas été touché. Tests : `config_players_order_test.go` (v2, v3 tous titres, v3 un
  titre, 50 appels chacun).
- **L5b-2** Vérifié sur pièces : `RecomputeIsWithFriendsCore` lit la base partagée (`LookupFriendXUIDs` sur `xuid_aliases`,
  `loadMatchesWithFriends` sur `match_participants`) et n'écrit que dans la base joueur (`insertEnrichmentBoolFlagDelta` sur
  `player_match_enrichment`, `RefreshAggregates(playerDB)`) : aucune écriture partagée n'est nécessaire, la prise du writer est
  retirée (pas de coalescence ajoutée, voir D-L5b-4). `RecomputeIsWithFriends` lit la base partagée par `acquireSharedReaderStandalone`
  (`Provider.Get` ou `OpenReadForQuery`) et ne l'ouvre plus du tout quand la liste d'amis est vide (la démotion ne regarde que la base
  joueur). Allowlist : l'entrée `friends_orchestrator_service.go` est retirée ; `RecomputeIsWithFriends` sort de la liste des noms
  « qui prennent un writer » (il n'en prend plus), et `sync/friends_recompute.go` entre dans les fichiers analysés (une reprise du
  writer y est vue rouge). Aucune entrée d'allowlist restante n'est injustifiée (6 entrées, toutes datées du 2026-10-16 avec raison).
- **L5b-3** Un seul lanceur, `media.RunFFmpeg(ctx, FFmpegJob{Args, Stdout, Stderr, MaxWait})` (`media/ffmpeg_slot.go`) : place unique
  de L5-5 (`slots`, 1 place) et délai maximal de L5-5 (45 min depuis l'obtention de la place). Réentrant par marqueur de contexte :
  `BuildHLS` garde sa place pour sonde + analyse + transcodage, les lancements internes ne la redemandent pas (sans cela, interblocage).
  Sites migrés : `hls.go` (transcodage), `hls_audio_analyze.go` (enveloppes audio, aussi `AnalyzeAudioLayoutReport` et
  `CollapseRedundantHLSAudio` en CLI), `hls_audio_migrate.go` (ré-encodage AAC), `remux.go` (flux vers le client), `ops/media_thumbnails.go`
  (miniatures), `ops/media_tooling.go` (sonde `-version/-encoders/-muxers` du boot). Remux : attente de la place bornée à 30 s
  (`remuxSlotWait`), `ErrFFmpegBusy` avant tout octet, `respondRemuxBusy` répond 503 + `Retry-After: 30` (`media_serve.go`).
  `ffprobe` volontairement hors du lanceur (D-L5b-2). Garde-rail : `archlint/ffmpeg_single_launcher_test.go` (AST sur `cmd`, `internal`,
  `pkg` : aucun `exec.Command*` désignant ffmpeg hors de `media/ffmpeg_slot.go`, exactement un lancement dans ce fichier ; détection
  testée sur littéral, identifiant, `ffprobe` toléré, autre binaire). Tests : `ffmpeg_run_test.go` (place tenue pendant la vie du
  processus, second lancement `ErrFFmpegBusy` sans processus, délai maximal, réentrance, sorties et code de sortie, remux occupé sans
  octet écrit), `media_remux_busy_test.go`.
- **L5b-4** `[!]` Vérification reprise sur pièces : `engine_postsync_scoring.go:196` appelle bien `batchComputeLUSR(..., false)`, mais
  depuis `runLUSRV1UnderRead`, qui retourne tôt quand `IsLUSRV2Canonical()` (`:182-185`), et le boot serveur pose
  `LEVELUP_LUSR_CANONICAL=LUSR_V2` par défaut (`cmd/server/main.go:1186` -> `skill_v2_canonical.go:58`, `DefaultLUSRModeIfUnset`). L'affirmation « chemin du
  post-sync normal » n'est donc vraie que pour `canonical=LUSR` (opt-out, tests, CLI), pas pour le défaut de production. Rien n'est
  modifié (D-L5b-1, hors sécurité) ; à trancher avec l'utilisateur.

**Preuves du rouge** (sorties gardées dans `%TEMP%\l5b\`) :
- L5b-1 `go test ./internal/service/ -run TestBootstrapBuild_WritablePlayerSlugs -count=50` : exit 1, 10 sous-tests FAIL
  (`writable_player_slugs [bob alice], attendu [alice bob]`) ; `go test ./internal/config/ -run TestLoadPlayers_OrderIsDeterministic` :
  3 FAIL (ordre `[halo_infinite/zed halo_infinite/alice ...]`). Vert : `-count=50` exit 0 (`ok levelup/go-api/internal/service 0.695s`).
- L5b-2 `go test ./internal/sync/ -run TestRecomputeIsWithFriends_` : FAIL `writer partagé acquis 1 fois, attendu 0`. Garde-rail :
  entrée d'allowlist retirée seule : FAIL `friends_orchestrator_service.go : 2 prise(s) de writer, allowlist 0` ; reprise de
  `AcquireSharedWriterStandalone` dans `friends_recompute.go` : FAIL `friends_recompute.go : 1 prise(s) de writer, allowlist 0`, source
  restaurée ensuite.
- L5b-3 `go test ./internal/archlint/ -run FFmpeg` : FAIL, 6 sites (`hls.go:550`, `hls_audio_analyze.go:167`, `hls_audio_migrate.go:150`,
  `remux.go:70`, `ops/media_thumbnails.go:164`, `ops/media_tooling.go:194`) et « 0 lancement(s) dans ffmpeg_slot.go » ; tests du lanceur et du
  handler : échec de compilation (`RunFFmpeg`, `ErrFFmpegBusy`, `respondRemuxBusy` indéfinis).

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`, Go 1.26.9) :
- Gate du lot : `go test ./internal/api/... ./internal/service/... ./internal/media/... ./internal/archlint/ ./internal/config/...
  ./internal/ops/... ./internal/sync/... -count=1` : exit 0, 29 paquets ok, 0 `--- FAIL:`. `-count=50` sur le test du bootstrap : exit 0.
- Intégration (L5b-2 touche le sync) : `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... -count=1` : exit 0,
  14 paquets ok, 0 `--- FAIL:` ; `no_art_patterns_test.go` non modifié.
- Gate commun : `go build ./cmd/server/` exit 0 ; `go vet` (config, sync, archlint, media, ops, api/handlers) exit 0 ;
  `golangci-lint run --new-from-rev=af72aa583` et `--new-from-merge-base=origin/main` : 0 issues. Aucune route Huma ajoutée ni modifiée
  (le 503 du remux est une réponse de handler chi hors OpenAPI) : pas de régénération des types ; `OpenAPI`/`Contract` passent dans la
  suite `./internal/api/...`. Web non touché.
- Hooks pre-commit (gofmt, check-merge-conflict, docs-fr-sync, gitleaks, go-vet) verts sur les trois commits de code.

**Écarts au texte, justifiés** : deux tests existants de `friends_recompute_test.go` supposaient que le wrapper CRÉAIT la base partagée
absente. Un lecteur ne crée rien : le test à liste vide n'ouvre plus la base partagée (inchangé, vert), le test « aucun xuid résolu » crée
la base avant l'appel. `ops.runFFmpeg` perd le champ `Stderr` de l'`ExitError` de `.Output()` (jamais lu).

**Non-régression (règle 9)** : `PUT /players/{slug}/friends` répond comme avant (même recalcul, mêmes notifications) mais ne bascule plus la
base partagée en écriture ; la lecture de la liste n'est pas concernée. Médias : miniatures et transcodages se mettent en file derrière la
place unique ; les lectures remuxées attendent jusqu'à 30 s puis répondent 503 (le lecteur vidéo réessaie) ; le chemin HLS (lecture
normale d'un média transcodé) ne lance pas ffmpeg. Profils : l'ordre de `available_players` est désormais stable (D-L5b-5).

**Découvertes** : D-L5b-1 à D-L5b-5 (§6), non traitées.

Commits : `755a6543b` (L5b-1), `a071a8cdd` (L5b-2), `5002566fc` (L5b-3), puis le commit de clôture (plan, thought_log).

### L5c - Corrections issues de L5b - 2026-10-16 - clos

Base : `git log --oneline -1` = `3aab936a6` avant le lot ; branche `feat/securite-2026-10-15`. Références rouvertes avant d'agir :
`media/ffmpeg_slot.go`, `media/remux.go` (`StreamRemuxWebMPlan`, `probeAVStreams`), `media/hls.go` (`ProbeStreamsDetailed`),
`ops/media_thumbnails.go` (`probeVideoDuration`), `api/handlers/media_serve.go` (`respondRemuxBusy`),
`archlint/ffmpeg_single_launcher_test.go`, `service/friends_orchestrator_service.go`, `api/handlers/friends.go`
(`recomputeInBackground` : une goroutine par PUT), `platform/coalesce/runs.go`, `service/bootstrap_service.go` (`Build`,
`BuildPlayersList`), `service/bootstrap_ownership.go` (`DirectOwnerFor`).

**Statuts** : L5c-1 à L5c-4 `[x]`. Aucun `[!]`, aucun `[~]`.

- **L5c-1** `media.FFmpegJob` porte un `Pool` nommé : `PoolTranscode` (valeur zéro, 1 place : transcodages HLS, analyses audio,
  miniatures, sondes de version) et `PoolRemux` (`remuxPlaces` = 2). `StreamRemuxWebMPlan` demande `PoolRemux` (attente toujours bornée
  à `remuxSlotWait` = 30 s, 503 + `Retry-After: 30` par `respondRemuxBusy`, inchangés) ; même délai maximal de 45 min pour les deux.
  Le marqueur de réentrance est par pool (`slotHeldKey{pool}`) : `BuildHLS` garde sa place de transcodage pour ses lancements internes.
  Un pool inconnu est refusé avant tout lancement. Garde-rail : le lanceur reste unique (`TestFFmpegLaunchedOnlyThroughTheSlot`
  inchangé) et `TestFFmpegPoolsAreTheTwoNamedOnes` interdit tout `slots.New` dans `internal/media` hors de `ffmpeg_slot.go` et exige
  exactement 2 pools dans ce fichier (détection testée sur 4 cas). Aucun garde-rail assoupli. Tests : `ffmpeg_pools_test.go`
  (deux spectateurs remuxés servis ensemble, remux n'attend pas un transcodage, transcodage n'attend pas deux remux, 3e remux
  `ErrFFmpegBusy` sans processus, pool inconnu, délai de 45 min côté remux) ; `TestStreamRemuxWebMPlan_BusyWritesNothing` adapté
  (il tenait l'ancienne place unique : il occupe maintenant les deux places de remux).
- **L5c-2** `RecomputeForPlayer` passe par `coalesce.Runs` (clé : xuid), champ `playerRuns` du service. La passe (`recomputePlayerOnce`)
  relit la liste d'amis à chaque exécution, donc le rattrapage voit la dernière liste persistée par le PUT. Un demandeur absorbé rend
  l'erreur du dernier recalcul et 0 promu (D-L5c-4). Couture de test : `recomputeDB` (défaut : `sync.RecomputeIsWithFriends`).
  `RecomputeAll` (CLI, boot) n'est pas touchée. Aucun fichier de `internal/sync` modifié. Tests : `friends_orchestrator_coalesce_test.go`
  (8 demandes rapprochées : 2 recalculs, jamais 2 en même temps, dernière liste d'amis reçue ; deux joueurs indépendants ; erreur propagée).
- **L5c-3** `media.RunFFprobe(ctx, args...)` (`media/ffprobe.go`) : contexte borné à `ffprobeMaxDuration` (30 s), `WaitDelay` de 2 s
  après l'arrêt, aucune place prise (une sonde sous la place de `BuildHLS` ne s'interbloque pas). Les trois sondes y passent
  (`probeAVStreams`, `ProbeStreamsDetailed`, `ops.probeVideoDuration`). Garde-rail `archlint/ffprobe_single_launcher_test.go` : tout
  `exec.Command*` de ffprobe hors de `media/ffprobe.go` est rouge, exactement un lancement dans ce fichier (détection testée sur 5 cas) ;
  le détecteur de `ffmpeg_single_launcher_test.go` est factorisé (`execLaunchSites`) sans changer son comportement. Tests : `ffprobe_test.go`.
- **L5c-4** Vérifié sur pièces : `Build` retombait sur `ownedPlayers[0]` et `BuildPlayersList` sur `players[0]` ; `filterOwnedPlayers`
  rend les profils de l'utilisateur ET ceux de ses co-membres de groupe. Avec le tri de L5b-1, un utilisateur dont le profil suit
  alphabétiquement celui d'un ami arrivait sur le profil de l'ami (avant L5b-1 : un profil au hasard). Correctif :
  `defaultPlayerIndex` (`bootstrap_ownership.go`) retient le premier profil dont la session est propriétaire directe
  (`DirectOwnerFor`), sinon le premier ; utilisé par `Build` (`current_player`) et `BuildPlayersList` (`default_player_slug`). Le choix
  mémorisé en session reste prioritaire dans `Build`. Tests : `bootstrap_default_player_test.go` (utilisateur, admin lié, aucun profil
  en propre, admin sans xuid, compte sans xuid, choix de session, sans propriété appliquée).

**Preuves du rouge** (sorties gardées dans `%TEMP%\l5c\`) :
- L5c-1 `go test ./internal/media/ -run 'TestStreamRemuxWebMPlan_(TwoViewers|DoesNotWait)'` : 2 FAIL (`processus démarrés = 1, attendu 2` ;
  `la lecture remuxée a attendu le transcodage : ... aucune place libre dans le délai d'attente`), avant tout changement du lanceur. Les
  tests de pool (`PoolRemux`, `remuxSlots`) ne compilaient pas avant le correctif. Garde-rail : un `slots.New(3)` ajouté dans
  `internal/media/zz_tmp.go` rend `TestFFmpegPoolsAreTheTwoNamedOnes` rouge (`pool de places ouvert hors du lanceur ffmpeg`), fichier retiré ensuite.
- L5c-2 `go test ./internal/service/ -run TestRecomputeForPlayer_` : FAIL `recalculs = 8, attendu 2 (un en cours + un rattrapage)`.
- L5c-3 `go test ./internal/archlint/ -run FFprobe` : FAIL, 3 sites (`hls.go:434`, `remux.go:101`, `ops/media_thumbnails.go:102`) et
  `0 lancement(s) de ffprobe` dans `ffprobe.go` ; tests de `media` : échec de compilation (`RunFFprobe`, `ffprobeCommand` indéfinis).
  Mutation après correctif (délai retiré de `RunFFprobe`) : `TestRunFFprobe_StopsAtTheMaxDuration` FAIL après 20 s et
  `TestRunFFprobe_BoundsTheContextAndTakesNoSlot` FAIL (`pas de délai maximal`) ; source restaurée.
- L5c-4 `go test ./internal/service/ -run TestBootstrap_DefaultPlayer` : 3 sous-tests FAIL (`current_player = "alice", attendu "zed"` pour
  l'utilisateur et l'admin lié). Le cas « compte sans xuid » attendait à tort un profil : un non-admin sans xuid ne voit rien
  (`CanAccessPlayer`), attente corrigée dans le test avant le correctif.

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`, Go 1.26.9) :
- Gate du lot : `go test ./internal/api/... ./internal/media/... ./internal/service/... ./internal/sync/... ./internal/archlint/
  ./internal/ops/... ./internal/config/... -count=1` : exit 0, 29 paquets ok, 0 `--- FAIL:`. Répétitions : pools/ffprobe/remux de `media`
  `-count=15` exit 0 ; `TestRecomputeForPlayer_` `-count=30` exit 0 ; `TestBootstrapBuild_WritablePlayerSlugs` `-count=50` exit 0.
- Intégration `-tags=integration -p 1 ./internal/sync/...` : non lancée, `internal/sync` n'est pas modifié par le lot (la condition
  du gate, « si L5c-2 touche le sync », n'est pas remplie : le recalcul coalescé vit dans `internal/service`).
- Gate commun : `go build ./cmd/server/` exit 0 ; `go vet` (media, ops, service, archlint, api/handlers, platform) exit 0 ;
  `golangci-lint run --new-from-rev=3aab936a6` et `--new-from-merge-base=origin/main` (media, ops, service, archlint) : 0 issue.
  Aucune route Huma touchée : pas de régénération des types ; `OpenAPI`/`Contract` passent dans `./internal/api/...`. Web non touché.
- Hooks pre-commit (gofmt, check-merge-conflict, docs-fr-sync, gitleaks, go-vet) verts sur les quatre commits de code.

**Non-régression (règle 9)** : lecture d'un clip remuxé : deux spectateurs simultanés sont servis, un troisième attend jusqu'à 30 s puis
reçoit 503 + `Retry-After: 30` (le lecteur vidéo réessaie, comportement de L5b) ; un transcodage ou une miniature en cours ne retarde plus
une lecture remuxée, et inversement. Liste d'amis : le PUT répond comme avant, le recalcul de fond est absorbé s'il en existe déjà un pour
le joueur (une passe de rattrapage suit). Sondes ffprobe : mêmes arguments, mêmes erreurs, avec une échéance de 30 s. Page d'accueil : un
utilisateur connecté arrive sur son propre profil ; pour un compte sans profil en propre (famille seule, admin sans xuid) le premier
profil visible reste le défaut ; le choix mémorisé en session est inchangé.

**Découvertes** : D-L5c-1 à D-L5c-4 (§6), non traitées. D-L5b-2, D-L5b-4, D-L5b-5 traitées ; D-L5b-3 traitée pour les remux
(miniatures : D-L5c-1).

Commits : `4df5f2c98` (L5c-1), `2c7e791b0` (L5c-2), `1b67f8c79` (L5c-3), `6b2d37c66` (L5c-4), puis le commit de clôture (plan, thought_log).

### [2026-10-16] Lot L10 — exécuté en parallèle (accord de l'utilisateur), branche `feat/securite-l10`, fusionné (79f9da999)

**Statuts** : L10-2, L10-5, L10-6 `[x]` ; L10-4 `[x]` sauf la partie nginx `[!]` (IP du serveur de calcul absente du dépôt, posée à l'activation, V-5) ; L10-1 `[!]` (Docker absent du poste : validation sur le VPS, V-4) ; L10-3 `[!]` (mesure, V-6). L8-9 `[~]` (couvert par L10-2).
**Preuves** : `compose_hardening_test.go` rouge sur le compose d'avant (4 violations + variable de proxy absente de la démo), vert après ; 12 mutations synthétiques signalées ; exception datée `composeProxyHeadersFromEnvFile` (le service `levelup` reçoit la variable par `.env.local`). `TestDockerfileImagesAreWatchedByDependabot` rouge sur l'ancien `dependabot.yml`. `dockerfile_go_version_test.go` : 11 fixtures, neutralisation de la comparaison → 6 cas en échec, restaurée. `worker_unit_hardening_test.go` rouge sur l'unité et le script d'avant. Capacités : `CHOWN` (chown de l'entrée), `DAC_READ_SEARCH` (parcours de dossiers 0700/0750 possédés par appuser), `SETUID`/`SETGID` (`gosu`) — `DAC_READ_SEARCH` déduite, non mesurée. Unité : `ProtectSystem=full`, `ProtectHome=read-only`, `ReadWritePaths=/var/lib/levelup-worker -/opt/levelup/data/cache`. Script : refus d'un lien ou d'une copie non root (6 cas testés hors systemd), installation de la copie = geste root (runbook §7).
**Gates** : `go test ./internal/archlint/ -run 'Compose|Nginx|Dockerfile'` exit 0 ; archlint complet exit 0 (rejoué par le superviseur après fusion : ok) ; `go vet`, `golangci-lint` 0 issue ; `bash -n` et `shellcheck` de `deploy-worker.sh` exit 0 ; `docker` et `systemd-analyze` indisponibles.
**Découvertes** : `.ai/V7.5/PLAN_OUVRIER_VPS_DEPLOY.md` (lignes 84, 175) et le journal décrivent encore l'installation par lien ; `scripts/systemd/levelup-docker-prune.service:15` cite `golang:1.26-bookworm` en commentaire.

### L6 - Fichiers et chemins - 2026-10-16 - clos

Base : `git log --oneline -1` = `10bc447a1` avant le lot ; branche `feat/securite-2026-10-15` ; L8 exécuté en parallèle dans un autre
worktree (non touché). Références rouvertes avant d'agir : `handlers/media_serve.go` (quatre racines, `http.ServeFile`),
`handlers/media_paths.go` (`mediaStoredPathToURL`, racine legacy `captures/{rel}`), `ops/media_paths.go`, `domain/title/registry.go`
(`PlayerDir`, `PlayerCapturesDir`, `MediaDataDir`, `ReplayArtifactPath`), `domain/title/film_id.go` (`FilmShortMatchID` : troncature
sans contrôle), `registry_film_facts.go:76`, `registry_tactical.go:52`, `service/replay_service.go:65`, `middleware/match_participation.go`,
`middleware/require_player_ownership.go` (transparente hors propriété appliquée), `handlers/tactical_sequence.go` (troisième copie de
la forme, motif alphanumérique), `server_apiv1.go` (`http.FileServer` sur `static/`), `handlers/media_upload.go` (extension seule),
`domain/match_id.go`, `sync/backfill.go:518`.

**Statuts** : L6-1 à L6-5 `[x]`, D-L3-3 traitée dans L6-3. Aucun `[!]`, aucun `[~]`.

- **L6-1** `media.IsServableExt` (vidéos, images, miniatures `.webp`, HLS `.m3u8`/`.m4s`/`.ts`) appliquée à la requête avant toute
  lecture disque, sur les quatre racines ; la racine legacy interne (`players/{slug}/`) est confinée à `captures/` (`mediaRoot{base,
  confine}`) ; un dossier n'est jamais servi (`isRegularFile`). Test `handlers/media_serve_confinement_test.go`.
- **L6-2** `domain.IsValidPlayerSlug` (lettres et chiffres Unicode, marques, espaces internes, `_`, `-`, 64 octets au plus) vérifié par
  `RequirePlayerOwnership` AVANT la décision de propriété et quel que soit le mode (démo, auth inactive) : 400 `invalid_player_slug`.
  Test `middleware/require_player_slug_test.go` (17 slugs x 3 modes ; gamertags à espaces, accentués, japonais acceptés).
- **L6-3** `title.IsValidMatchID` (`film_id.go`) : forme unique ; `domain.IsValidMatchID` (et son test) supprimé ;
  `sync.isValidMatchID` supprimé (ses tests, présents dans la baseline, gardent leur nom et visent la forme canonique) ; motif de
  `tactical_sequence.go` remplacé. `FilmShortMatchID` rend pour un identifiant mal formé une clé NUL + hexadécimal : un seul élément
  de chemin, refusé par tout appel système, distinct pour chaque identifiant (une clé commune faisait passer `Has("m2")` pour vrai dans
  un stub). Couvre `ReplayArtifactPath`, `FilmFactsPath`, `TacticalRasterPath`, `DemoLayout.ReplayArtifactPath`,
  `replayService.artifactPath` et le cache de films. `RequireMatchParticipation` (12 routes `/matches/{match_id}`, rejeu compris) :
  forme invalide -> 404 `match_not_participant` avant toute vérification, même sans vérification (démo). Garde-rail
  `archlint/match_id_form_single_source_test.go` (fonction de validation ou motif regexp de match hors de `film_id.go` ; auto-preuve
  sur 5 cas). Tests : `domain/title/match_id_path_test.go` (4 constructeurs x 15 identifiants piégés : chemin dans son dossier,
  écriture refusée), `match_id_form_test.go`, `middleware/match_participation_form_test.go`, `handlers/tactical_sequence_test.go`
  (`TestSequence_MatchIDHorsFormeHalo`).
- **L6-4** `registerStaticRoutes` (`api/static_files.go`) : `http.FileServer` sur `noDirListingFS` (un dossier s'ouvre comme absent) ;
  `/static/commendations/*` inchangé. Test `api/static_files_test.go`.
- **L6-5** `media.ContentMatchesExtension` : `http.DetectContentType` des 512 premiers octets comparé à la FAMILLE de l'extension.
  Image -> `image/*`. Vidéo -> `video/*` ou `application/octet-stream` : Go ne reconnaît ni le QuickTime (`.mov`, marque `qt  `) ni
  un MP4 sans marque compatible `mp4x` ; `ffprobe` les valide en aval (transcodage HLS) ; refuser l'octet-stream refuserait des `.mov`
  légitimes. Tout le reste (texte, HTML, SVG, XML, PDF, archive, image sous extension vidéo, vidéo sous extension image) est refusé.
  Un fichier non concordant refuse l'envoi entier : 415 `upload_content_mismatch` avant toute écriture ; message FR + EN
  `common.media.upload_content_mismatch` (`UploadButton`, table code -> message). Les extensions d'envoi se lisent dans
  `internal/media` (`IsUploadableExt`) : la map `allowedMediaExts` du handler est retirée (sinon troisième copie de la liste des
  images). Tests : `media/content_check_test.go`, `handlers/media_upload_content_test.go`, `UploadButton.test.tsx` (FR, EN).

**Écarts au texte, justifiés** : L6-2 répond 400 (forme) et non 403 : un slug mal formé ne peut désigner aucun profil, et le refus vaut
aussi en démo où la garde de propriété est inerte. L6-3 : la forme canonique vit dans `domain/title` et non dans `domain` (cycle
d'import) ; le refus de forme à la frontière est un 404 identique à l'absence, comme la séquence tactique (pas d'oracle). Fixtures de
test : identifiants factices (`m1`, `veh2`, `deja-cuit`, `demo0001`...) remplacés par des identifiants de forme Halo dans 21 fichiers
de test (`service`, `api/wire`, `replaybuild`, `sync/replayartifacts`), y compris deux littéraux SQL ; aucun test désactivé ni
assoupli. `TestFilmShortMatchID` : attente du cas `""` passée de `""` à la clé invalide (contrat changé par le plan).

**Preuves du rouge** (sorties dans `%TEMP%\l6\`) :
- L6-1 `go test ./internal/api/handlers/ -run 'TestServeMediaFile_(NonMedia|MediaFiles|PlayerDir)'` : exit 1, 19 sous-tests FAIL
  (`.duckdb`, `.wal`, `.json` servis en 200 sur les 4 racines ; `clip.mp4`, `archive/clip.mp4` hors `captures/` servis) ; les médias
  légitimes passaient déjà.
- L6-2 `-run 'TestRequirePlayerOwnership_(Refuses|Accepts)'` : 51 cas sur 51 servis en 200 (dont `..`, `a/b`, `a\b`, `..%2F..%2Fx`).
- L6-3 `go test ./internal/domain/title/ -run TestMatchPaths` : 60 violations, dont `ReplayArtifactPath("../../etc/passwd")` =
  `data/cache/et.json` hors du dossier des artefacts, et des écritures acceptées pour `""`, `..`, `/` ; participation : FAIL (forme
  invalide servie en 200) ; séquence : FAIL (`zzzzzzzz-zzzz` atteint le service) ; garde-rail : FAIL avant centralisation, puis
  mutation (copie `isValidMatchID` dans `sync`, motif `motifMatchSequence` dans `handlers`, fichiers retirés ensuite) : 2 copies
  signalées.
- L6-4 `go test ./internal/api/ -run TestStatic_` : 5 FAIL, listings HTML de `/static/`, `/static/medals/`, `/static/sounds/` en 200.
- L6-5 contrôle neutralisé le temps du test (source restaurée, vérifiée) : 5 sous-tests FAIL (HTML, SVG, texte acceptés en 200) ;
  front : 2 FAIL (message du serveur affiché au lieu du message traduit).

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`, Go 1.26.9) :
- Gate du lot : `go test ./internal/api/... ./internal/domain/title/... ./internal/service/... ./internal/media/... ./internal/archlint/
  -count=1` : exit 0, 14 paquets ok, 0 `--- FAIL:` (rejoué après le dernier commit de code).
- Paquets touchés par les fixtures et la clé de film : `go test ./internal/domain/... ./internal/media/... ./internal/archlint/
  ./internal/port/... ./internal/sync/... ./internal/replaybuild/... ./internal/ops/... ./internal/scheduler/... ./internal/notify/...
  ./cmd/levelup/ ./cmd/replay-corpus-gate/ ./cmd/zone-attribution/ -count=1` : exit 0, 29 paquets ok.
- Intégration (L6-3 touche `sync`) : `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/replaybuild/...
  ./internal/service/... ./internal/ops/... ./internal/scheduler/... ./internal/platform/duckdb/ -count=1` : 25 paquets, 24 ok, rouge
  sur `sync/replayartifacts` (fixtures d'intégration à identifiants factices) ; fixtures corrigées, paquet rejoué
  `-tags=integration -p 1` : exit 0. `no_art_patterns_test.go` non modifié.
- Gate commun : `go build ./cmd/server/` exit 0 ; `go vet` (api, domain, media, service, sync, replayartifacts, replaybuild, archlint ;
  `-tags=integration` sync et replayartifacts) exit 0 ; `golangci-lint run --new-from-rev=10bc447a1` et
  `--new-from-merge-base=origin/main` (v2.12.2) : 2 issues au premier passage (staticcheck QF1002, goconst `.png`), corrigées
  (`0ce7f2d3e`), puis 0 issue. `go run ./cmd/openapi-gen` : aucun diff (aucune route Huma modifiée ; 415 et 400 sont des réponses de
  handler chi ou de middleware) ; `go test ./internal/api/ -run 'OpenAPI|Contract'` ok.
- Web : purge de `node_modules/.tmp`, `npm run typecheck` exit 0, `npm run lint` exit 0 (0 erreur, 26 avertissements préexistants),
  `npm run test:run` exit 0 (924 fichiers passés / 5 ignorés, 9670 tests passés / 23 ignorés), `lint:fields`, `lint:colors`,
  `knip-ratchet`, `lint-contract-ratchet`, `lint-cross-feature-imports` : exit 0. Manifeste i18n régénéré
  (`scripts/build_i18n_manifests.mjs`).
- Hooks pre-commit (gofmt, check-merge-conflict, docs-fr-sync, gitleaks, go-vet) verts sur les sept commits (un premier essai du
  commit L6-3 refusé par gofmt, formaté puis recommité).
- `scripts/disk-hygiene.ps1 -Lot securite -Apply` non lancé : le cache `go-build-securite` sert à tout le plan (lots suivants).

**Non-régression (règle 9)** :
- L6-1 : extensions produites relevées dans le code (envoi : `mp4 mov avi mkv webm png jpg jpeg bmp gif` ; miniatures `.webp`/`.gif` ;
  HLS `master.m3u8`, `stream_*.m3u8`, `init_*.mp4`, `seg_*.m4s`, `.ts`) et sur disque (`data/media/JGtm` : `.mp4`, `thumbs/*.webp`) :
  toutes servies (test sur chaque racine). URL legacy interne générée par `mediaStoredPathToURL` = `captures/{rel}` (dossier interne de
  `ResolveCapturesDir`), toujours servie ; démo : `MediaCapturesBaseDir` = dossier média plat, racine canonique inchangée.
- L6-2 : slugs réels = clés de `db_profiles.json` = gamertags (local : alphanumériques ; démo `DEMO` ; e2e `demo-player`,
  `test-player` ; tests Go avec espaces) : tous acceptés. Le front interpole le slug brut dans l'URL ; un espace arrive décodé dans le
  paramètre chi. Aucun message front ajouté : un slug légitime ne reçoit jamais `invalid_player_slug`.
- L6-3 : les identifiants réels sont des UUID Halo (hexadécimal et tirets) ; les liens du front vers `/matches/{match_id}` portent le
  `match_id` du serveur ; e2e : `00000000-dead-beef-...` et `E2E_README_MATCH_ID` (UUID). Appelants internes (sync, cuisson, purge,
  rattrapage, démo) : suites complètes et intégration vertes.
- L6-4 : le front ne lit que des fichiers sous `/static/` (`ranks`, `weapons`, `medals`, `sounds`, `maps`...) ; aucun `index.html` sous
  `static/` ; le SPA ne liste pas (un dossier y reçoit `index.html`).
- L6-5 : vraies captures locales (`data/media/JGtm/*.mp4`, 7 fichiers) : boîte `ftyp` de marque `mp42` -> `video/mp4`, acceptées ; un
  `.mov` QuickTime ou un MP4 `isom` sortent en octet-stream, acceptés ; aucun envoi en e2e. Le front envoie déjà tous les fichiers en
  une requête : un fichier non concordant refuse l'envoi entier avec un message clair.

**Découvertes** : D-L6-1 à D-L6-4 (§6), non traitées.

Commits : `013bb2378` (L6-1), `d966cbea9` (L6-2), `30596fc09` (L6-3), `4b6167f86` (L6-4), `29b2692bf` (L6-5), `0ce7f2d3e` (lint),
`32850669f` (extensions d'envoi), puis le commit de clôture (plan, thought_log).

### [2026-10-16] Lot L8 — exécuté en parallèle (branche `feat/securite-l8`), fusionné après L6

**Statuts** : L8-1 à L8-8, L8-10, L8-12 `[x]` ; L8-9 `[~]` (L10-2) ; L8-11 `[!]` (provisionnement de `data/demo_films/` au déploiement, V-3 ; runbook, `CONFIGURATION.md` EN/FR et `deploy.yml` vérifiés à jour). L8-12 : dépôt fait par l'exécutant, VPS appliqué par le superviseur le 2026-10-16 (`limit_req_status 429;` dans le `server` démo, sauvegarde `demo-levelup.avant-429`, démo et prod 200).
**Décisions d'exécution** : L8-1 appliqué à `RequireAdmin` ET `RequireAuth` en démo (403 `demo_mode_forbidden` sauf liste nommée : `GET /api/v1/settings`, `GET /api/v1/healthz/home`, `GET /api/v1/presence`, relevées dans le front, les E2E et le harnais) ; effet de bord voulu : `GET /groups`, `/jobs/{id}`, `_diag/*` par joueur et `/titles/{slug}/catalog/*` répondent 403 en démo (aucune lecture par le front de démo). L8-5 : spec E2E entière marquée hors démo (un seul cas, le like). L8-6 : critère `CapReplay`, pas de slug.
**Preuves** : ratchet `demo_guarded_routes_ratchet_test.go` (walk du routeur de démo, plancher de 80 routes gardées, 6 témoins C8-1 dont `/debug/vars` en pile complète) rouge puis vert, mutation `GET /admin/users` détectée ; `config/demo_state_guard_test.go` rouge (4 tests) puis vert ; L8-7 : 5 ERROR reproduites puis 0 ; L8-8 : 404 → 200 ; L8-10 : `TestOutboundCallsAreNetguarded` rouge au retrait de l'allowlist fausse.
**Gates** : `go test ./internal/api/... ./internal/config/... ./internal/ops/... ./internal/platform/...` exit 0 (34 paquets) ; intégration `ops` exit 0 ; build, vet, golangci-lint 0 issue ; `playwright --list` 1 test ; `tsc -b` 0. **Après fusion (superviseur)** : `go build`, `go vet`, `go test` api/config/ops/platform/service/media/domain/archlint exit 0 ; `-tags=integration -p 1` ops et playerdirectory exit 0.
**Découvertes** : CHANGELOG à compléter à la release ; commentaire « Streamlit » obsolète dans `demo.conf` ; `OpenReadWriteShared(metaPath)` peut créer un fichier DuckDB vide (chargeurs H5 en démo sans H5, non vérifié) ; `CONFIGURATION.md` ne décrit pas la liste nommée de la démo.

### L7 - Gardes de chemin et CSRF - 2026-10-16 - clos

Base : `git log --oneline -1` = `6a2bbb4c7` avant le lot ; branche `feat/securite-2026-10-15` ; L9 exécuté en parallèle dans
`LevelUp-wt-securite-l9` (non touché). Références rouvertes avant d'agir : `middleware/csrf.go:68` (`isExemptPath(r.URL.Path)` après
`path.Clean`, sans `AmbiguousPath`), `middleware/require_auth_mutations.go` (`AmbiguousPath`, `IsMutatingRequest`),
`middleware/demo_read_only.go` (`DemoAllows`, exemption ouvrier déjà sous `!AmbiguousPath`), `middleware/demo_guarded_reads.go`
(L8), `middleware/rate_limit.go` (`rateLimitExempt(r.URL.Path)`), `middleware/player_visitor_actions.go`,
`middleware/require_player_ownership.go` (slug L6-2), `middleware/match_participation.go` (L6-3), `handlers/replay_local_gate.go`
(`isLocalRequest`, `replayMatchID` de L8), `middleware/loopback.go` (`PeerAddr` de L1), `server.go` (`applyTransverseMiddlewares`),
`server_apiv1.go` (montage du groupe joueur et du rejeu), `handlers/build_worker.go` (`RequireWorkerToken`).

**Statuts** : L7-1 à L7-4 `[x]`. Aucun `[!]`, aucun `[~]`.

**Gardes qui décident sur un chemin (relevé sur pièces)** et verdict sur la table des formes ambiguës :

| Garde | Lit | Avant L7 | Après L7 |
|---|---|---|---|
| CSRF, exemption ouvrier (`csrf.go`) | `URL.Path` nettoyé | exempte (handler atteint hors démo) | `!AmbiguousPath` : 403 `csrf_rejected` |
| Limiteur, exemptions `/static/`, `/api/v1/assets/`, extension statique (`rate_limit.go`) | `URL.Path` | exempte | `!AmbiguousPath` : compté |
| Démo, exemption ouvrier (`DemoReadOnly`) | `URL.Path` nettoyé | refuse (`!AmbiguousPath`) | inchangé |
| Démo, `POST /session/context` et POST de lecture (`DemoAllows`, `IsMutatingRequest`) | `URL.Path` exact, sous-chemin | refuse | inchangé |
| Démo, lectures nommées derrière `RequireAdmin`/`RequireAuth` (`demoServesGuarded`) | méthode + `URL.Path` exact | refuse (`RawPath`, `AmbiguousPath`) | inchangé |
| Garde d'écriture du groupe joueur (`RequireAuthForMutations`) | sous-chemin | écriture (401) | inchangé |
| Action du visiteur (`isVisitorAction`, garde de propriété) | sous-chemin exact | refuse (403) | inchangé |
| Forme du slug (L6-2) | paramètre chi (brut) | 400 | inchangé |
| Forme du match (`RequireMatchParticipation`, L6-3) | paramètre chi (brut) | 404 | inchangé |
| Rejeu (`ReplayGate`) | démo : `match_id` du chemin ou du corps ; hors démo : pair TCP | 404 | inchangé ; adresse vide fermée (L7-4) |
| Protocole ouvrier (`RequireWorkerToken`) | en-tête `Authorization` | ne décide pas sur le chemin | sans objet |

- **L7-1** `internal/api/csrf_ambiguous_path_cgo_test.go` : `TestTransverseStack_CSRF_AmbiguousPathNeverTakesWorkerExemption`, routeur
  complet (`openapigen.BuildDemoRouter`), Origin `https://evil.example` : la requête du constat, puis la table entière, doivent recevoir
  403 `csrf_rejected`. Sur ce routeur de démo la garde démo refusait derrière (403 `demo_mode_forbidden`) ; hors démo, le test de pile
  transverse de L7-3 montre le handler `dismiss` ATTEINT (200), l'exploitation réelle en production.
- **L7-2** `middleware/csrf.go` : `!AmbiguousPath(r) && isExemptPath(...)` ; contrat de `CSRF` et d'`isExemptPath` réécrit (le nettoyage
  seul ne suffit pas : chi route sur le chemin brut). Commentaire de `csrf_test.go` (`TestCSRF_POST_ExemptPrefix_TraversalNotExempt`)
  corrigé (il attribuait le refus de `%2e%2e` à `path.Clean`).
- **L7-3** Table unique `AmbiguousForms` (`internal/api/ambiguous_path_table_test.go`, exportée vers `api_test`) : `%2F`, `%2f`, `%5C`,
  `%5c`, `%2e%2e`, `%2E%2e` + `%2f` (casses mêlées), `%2E%2E` en segment clair, `//`, `/./`, `/../` en clair. `ambiguous_path_guards_test.go`
  la passe, derrière `applyTransverseMiddlewares` et les gardes de route montées comme dans `server_apiv1.go`, à chaque garde du tableau
  (10 tests, dont l'auto-contrôle `TestAmbiguousForms_TableIsAmbiguous`) ; chaque test vérifie aussi que le chemin propre reste servi
  (ouvrier sans Origin, `POST /session/context`, POST de lecture, `GET /settings` et `/healthz/home` en démo, fichiers statiques et
  assets exemptés du limiteur, like d'un ami, rejeu local et rejeu figé). Le rejeu en démo passe par la vraie
  `service.DemoReplayAllowlist` (index temporaire). **Correction exigée par le garde-rail** : le limiteur exemptait les chemins ambigus
  (30 cas rouges) ; `rate_limit.go` refuse désormais l'exemption à un chemin ambigu (`!AmbiguousPath(r) && rateLimitExempt(...)`).
- **L7-4** `handlers/replay_local_gate.go` : la branche « hôte vide = local » est retirée ; une adresse vide, sans hôte (`:5555`,
  `[]:5555`) ou qui n'est pas une IP n'est pas locale. `TestReplayLocalGate_RejectsUnreadablePeer` (pile transverse réelle, confiance
  proxy active et inactive, en-têtes forgés « 127.0.0.1 »). Tests à adapter : aucun. Relevé : `grep RemoteAddr` dans les tests du
  module ; les seuls tests qui traversent `LocalOnlyReplay` (`handlers/replay_test.go`, `api/replay_local_gate_test.go`) fournissent
  déjà une adresse (`127.0.0.1:54321`, `[::1]:5555`, `203.0.113.7:5555`) ; suites `internal/api/...` vertes après le correctif.

**Preuves du rouge** (sorties dans `%LOCALAPPDATA%\Temp\l7\`) :
- L7-1 `go test ./internal/api/ -run TestTransverseStack_CSRF_AmbiguousPath -count=1` : exit 1, 9 lignes d'échec (la requête du constat
  + 8 formes ; les deux formes `%5C` ne remontent pas au nettoyage et étaient déjà refusées), toutes `403 demo_mode_forbidden` au lieu
  de `csrf_rejected`.
- L7-3 `go test ./internal/api/ -run TestAmbiguous -count=1` : exit 1, 48 lignes : CSRF 18 cas (route `dismiss`, `player` ou `worker`
  atteinte en 200 avec une Origin étrangère), limiteur 30 cas (exempté). Les autres gardes étaient déjà vertes.
- L7-4 `go test ./internal/api/ -run TestReplayLocalGate -count=1` : exit 1, 6 lignes (`""`, `:5555`, `[]:5555` servis en 200, confiance
  active et inactive).
- Sensibilité du garde-rail (mutation d'une clause à la fois, source restaurée, `git status` vérifié) : retrait de `!AmbiguousPath` du
  CSRF -> 18 échecs ; du limiteur -> 30 ; de l'exemption ouvrier de la démo -> 18 ; de `DemoAllows` -> 4 ; d'`IsMutatingRequest` -> 34
  (deux tests) ; de `demoServesGuarded` -> 4 ; d'`isVisitorAction` -> 4 ; contrôle de forme du slug neutralisé -> 20 ; de la forme du
  match -> 20 (participation et rejeu). Aucune clause ne peut tomber sans rougir.

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`, Go 1.26.9, rejouées sur `8d1a4dc21`) :
- `go test ./internal/api/... -run 'CSRF|Ambiguous|Transverse' -count=1` : exit 0, 0 `--- FAIL:`.
- `go test ./internal/api/... -count=1` : exit 0, 0 `--- FAIL:` (`api`, `handlers`, `humacore`, `middleware`, `wire` ok ; tests
  OpenAPI et contrat compris).
- `go test ./internal/archlint/ -count=1` : exit 0.
- Gate commun : `go build ./cmd/server/` exit 0 ; `go vet ./internal/api/ ./internal/api/middleware/ ./internal/api/handlers/` exit 0 ;
  `golangci-lint run --new-from-rev=6a2bbb4c7` (v2.12.2) sur ces paquets : 0 issue ; forme CI (`--new-from-merge-base=origin/main`) :
  0 issue. API inchangée (aucune route Huma) : pas de `generate-types`. Web non touché.
- Hooks pre-commit (gofmt, check-merge-conflict, docs-fr-sync, gitleaks, go-vet) verts sur les trois commits de code.

**Non-régression (règle 9)** : chemins réellement émis relevés. Ouvrier (`cmd/replay-worker/protocol.go`) : `/build-queue/claim`,
`/heartbeat`, `/complete`, `/artifact?job_id=…` sous `/api/v1/internal`, jamais ambigus : exemption intacte (testée sans Origin).
Front : les segments variables passent par `encodeURIComponent` (slug, gamertag, titre, carte) ; un gamertag n'a ni `/`, ni `\`, ni
segment `.` ou `..` (forme L6-2), `encodeURIComponent` ne produit jamais `%2E` : aucune requête du front n'est ambiguë, et le front
envoie de toute façon l'Origin autorisée. Fichiers statiques : les noms de citations portent `%27`/`%3F` (jamais `%2F`, `%5C`, `%2E`) :
ils restent exemptés du limiteur ; un chemin ambigu n'y est que compté, jamais refusé. Rejeu local : le pair TCP n'est jamais vide en
service (nginx joint le conteneur par le pont Docker, D-L1-1) ; en local, `127.0.0.1` et `::1` restent servis.

**Découvertes** : D-L7-1 à D-L7-3 (§6), non traitées.

Commits : `389ba5497` (L7-1, L7-2), `94966d419` (L7-3), `8d1a4dc21` (L7-4), puis le commit de clôture (plan, thought_log).

### [2026-10-16] Lot L9 — exécuté en parallèle (branche `feat/securite-l9`), fusionné après L7

**Statuts** : L9-1 à L9-9 `[x]`. **Décompte** : 210 `err.Error()` (ou équivalents) renvoyés au client sur la base (127 en 4xx, 2 en 503, 81 en 500 ; le registre en comptait 121 par un motif plus étroit) → 0 ; garde-rail `err_error_response_test.go` à zéro, allowlist vide, suit aussi les variables affectées depuis `x.Error()`. Helpers `humacore.NewErrorFrom`/`FixedMessage`/`LogErrorResponse`, `handlers.writeErrorFrom`. `/health` public → `{status}`, détail sous `GET /api/v1/admin/health` (admin) ; `scripts/deploy.sh` lisait `app_version` dans le corps de `/health` (contrôle silencieusement inopérant) : remplacé par `LEVELUP_APP_VERSION` via `docker inspect` (non bloquant, testé avec un `docker` factice, à observer au prochain déploiement). CSP minimale (`object-src 'none'; base-uri 'self'; frame-ancestors 'none'`). Jobs : message fixe par code (`domain.NewJobError`), garde-rail `job_error_fixed_message_test.go`. Réinitialisation par l'admin : sessions révoquées, garde-rail de câblage. `GET /auth/xbox/login` : refus redirigés. Reconnexion refusée : bannière `OAuthRefusalBanner`.
**Gates** : build, vet, `go test` 28 paquets exit 0, golangci-lint 0 issue ; web : typecheck, lint, 9 723 tests, ratchets JS verts ; OpenAPI et types régénérés. **Après fusion (superviseur)** : `go build`, `go test` api/archlint/domain/service/platform/notifications/media/ops/config/tests/cmd exit 0.
**Découvertes** : texte d'erreur dans des diagnostics admin (voulu) ; `mediaLikeError` relaie un message libre ; codes de `StepDeviceCode` incohérents (→ L11-6) ; reset admin : 400 `password_too_long` au lieu de 500 ; erreurs de fabrique de service rendues en 404 `player_not_found` (préexistant) ; champ mort `WatcherAttempt.ErrorDetail` (→ L11-7) ; contrôle de version de `deploy.sh` à observer.

### L11 - Reliquats de chemins et de cohérence - 2026-10-16 - clos

Base : `git log --oneline -1` = `16fcd294a` avant le lot ; branche `feat/securite-2026-10-15`. Références rouvertes avant d'agir :
`wire/og_inject.go` (`playerSlugFromPath`), `wire/registry_pages_home.go` (`HomeCtx`), `config/player_resolver.go` (`ResolvePlayer`,
`demoDefForSlug`, `resolveRealPlayer`), `service/home_service.go` (usage social du slug), `middleware/title_from_path.go`,
`middleware/title.go`, `handlers/title_sync.go`, `service/profile_service.go` (`PurgeTitleData`), `middleware/require_auth_mutations.go`
(`AmbiguousPath`, `playerScopedSubPath`), toutes les lectures de `URL.Path` du paquet middleware, `handlers/admin_actions_replay_build.go`,
`domain/title/film_id.go`, routes du routeur réel (`chi.Walk`), `handlers/admin_appearance_diag.go`, `handlers/admin_lusr_gaps.go`,
`handlers/progression_backfill.go`, `features/setup/StepDeviceCode.tsx`, `features/auth/loginError.ts`, `handlers/auth.go` et
`handlers/watcher_handler.go` (codes posés), `platform/auth/watcher_attempt_store.go`, `ops/media*.go`, `media/extensions.go`.

**Statuts** : L11-1 à L11-8 `[x]`. Aucun `[!]`, aucun `[~]`.

- **L11-1** Sur pièces, le slug d'une page du SPA n'atteignait pas le disque : en démo `demoDefForSlug` retombe sur le joueur principal
  du roster, hors démo `resolveRealPlayer` compare aux profils chargés, `WithSocial` ne s'en sert que comme paramètre SQL. La garde est
  posée quand même (fermée par défaut) : `playerSlugFromPath` exige `domain.IsValidPlayerSlug`, sinon carte générique sans résolution
  du joueur. Tests `wire/og_inject_slug_test.go` (9 chemins piégés : `..%2F..`, `%2E%2E`, `%5C`, `:`, espace de tête, 65 octets ;
  gamertags accentués et à espace toujours résolus) et cas ajoutés à `TestPlayerSlugFromPath`.
- **L11-2** `TitleSlugFromPath(registry, param)` : un titre absent du registre reçoit 404 `title_not_found` avant la garde de propriété ;
  tout titre enregistré passe, statut compris (coming_soon, archivé). Sur pièces, `RemoveEntry`/`SetSyncEnabled` exigeaient déjà une
  entrée de `db_profiles.json` avant `os.RemoveAll` : le titre piégé n'atteignait pas le disque. Test : titres itérés depuis le registre
  plus un titre synthétique (aucun slug particulier) ; 8 titres hors registre refusés. Le test garde le nom
  `TestTitleSlugFromPath_SetsCtxFromPathParam` (présent dans la baseline `.ai/baselines/tests_pre_migration.jsonl`).
- **L11-3** `AmbiguousPath` dans `middleware/ambiguous_path.go`. Garde-rail `archlint/ambiguous_path_middleware_test.go` : dans
  `internal/api/middleware`, toute fonction (déclaration ou fermeture, chacune pour elle-même) qui lit `URL.Path`, `URL.RawPath` ou
  `URL.EscapedPath()` hors des arguments d'un appel de journal appelle `AmbiguousPath` ; allowlist datée réduite à `SlogLogger`
  (journal HTTP) ; auto-preuve sur 4 gardes témoins. Rouge sur le code réel : `playerScopedSubPath` lisait le chemin sans
  `AmbiguousPath` (ses deux appelants le faisaient) ; il rend désormais `""` sur un chemin ambigu (comportement des appelants inchangé).
- **L11-4** `decodeReplayBuildRequest` (commun à `/run` et `/enqueue`) : `match_id` hors de `title.IsValidMatchID` -> 400
  `invalid_match_id` (message fixe du code) avant toute création de job ou mise en file. Aucun appelant front.
- **L11-5** Relevé sur pièces (parcours du routeur réel + grep des motifs) : trois routes hors du groupe joueur portent un slug ou un
  gamertag sans contrôle de forme : `GET /admin/diag/appearance/{player_slug}`, `POST /admin/monitoring/lusr-gaps/{player}/recompute`,
  `POST /_admin/progression/backfill/{player_slug}` ; les trois `/_diag/*/{player_slug}` portent déjà `RequirePlayerOwnership`.
  Middleware `RequirePlayerSlugParam(param)` monté après la garde d'accès : 400 `invalid_player_slug` (un paramètre porté mais vide est
  refusé, une route du même handler sans le paramètre passe). Tests : `middleware/require_player_slug_param_test.go` (mêmes tables que
  L6-2, deux noms de paramètre) ; garde-rail `api/player_slug_param_routes_test.go` : toute route à `{player_slug}`, `{player}` ou
  `{gamertag}` porte `RequirePlayerOwnership` ou `RequirePlayerSlugParam`.
- **L11-6** `StepDeviceCode` lit son message par `deviceFlowFailureKey` (`features/auth/loginError.ts`, table unique des deux écrans du
  device-flow) ; sa table locale attendait `device_flow_denied`, `device_flow_error`, `identity_resolution_failed`, jamais émis.
  Codes réellement posés relevés : `device_flow_acquire_error`, `halo_exchange_error`, `identity_missing`, `instance_locked`,
  `invite_invalid`, `link_failed` (`device_flow_init_error` fait échouer le démarrage, déjà traité par l'écran). Exception d'import
  `setup=>auth/loginError` (module nommé, datée, justifiée) dans `tools/lint-cross-feature-imports.mjs` : plafond inchangé (7).
  Clé `common.device_code.err_denied` retirée (plus aucun usage), manifeste régénéré. Test `StepDeviceCode.test.tsx` (6 cas FR,
  3 cas EN).
- **L11-7** `WatcherAttempt.ErrorDetail` supprimé avec ses trois écritures. Deux sites journalisaient déjà la cause ; l'échec
  d'`InitDeviceFlow` ne la gardait que dans le champ mort : `slog.ErrorContext` ajouté avant la dégradation. Test
  `TestWatcherHandler_StartAuth_InitErrorCauseLogged` (cause au journal, jamais dans la réponse).
- **L11-8** `ops.supportedExtensions` et les constantes `extMOV`/`extAVI`/`extMKV`/`extWEBM` supprimés ; l'indexation lit la nature
  d'un fichier par `mediaKindOf` (`media.IsVideoExt` / `IsImageExt`) : mêmes ensembles qu'avant, casse ignorée. Garde-rail
  `archlint/media_ext_list_single_source_test.go` : hors de `internal/media/extensions.go`, aucun littéral composite ni bloc de
  constantes ne porte 3 extensions média distinctes ou plus (catalogue statique du front écarté, tenu par
  `no_duplicate_static_ext_list_test.go`) ; auto-preuve (map, slice, constantes vues ; aiguillage et extension isolée ignorés).
  Commentaire de paquet de `internal/media` corrigé (il affirmait que seul `ServeMediaFile` en dépendait).

**Preuves du rouge** (sorties dans `%LOCALAPPDATA%\Temp\l11\`) :
- L11-1 `go test ./internal/api/wire/ -run 'TestServeIndexWithOG_(Trapped|Valid)'` : exit 1, 9 sous-tests FAIL (slug piégé passé à la
  résolution du joueur).
- L11-2 signature avec registre sans contrôle (état d'avant) : `-run TestTitleSlugFromPath` exit 1, 8 sous-tests FAIL (handler atteint
  avec le contexte `..`, `a\b`, `\x00`...).
- L11-3 `go test ./internal/archlint/ -run 'TestMiddlewarePathGuardsCallAmbiguousPath'` : exit 1 sur `playerScopedSubPath` (code réel).
  Mutations (source restaurée, `git status` vérifié) : `!AmbiguousPath(r) &&` retiré du limiteur -> rouge sur `RateLimit` ; garde
  témoin ajoutée au paquet -> rouge sur `temoinL11`.
- L11-4 `-run TestAdminReplayBuild_MatchIDOutsideForm` : exit 1, 18 sous-tests FAIL (202, job créé ou mis en file pour `..`,
  `../../etc/passwd`, `a\b`, `zz`, octet nul, 65 caractères).
- L11-5 `go test ./internal/api/ -run TestPlayerSlugParamRoutes` : exit 1, les trois routes ci-dessus. Mutation du contrôle du
  middleware (seul le slug vide refusé) : 34 lignes d'échec.
- L11-6 `npx vitest run src/features/setup/StepDeviceCode.test.tsx` : 5 FAIL sur 9 (message générique au lieu du message du code).
- L11-7 `-run TestWatcherHandler_StartAuth_InitErrorCauseLogged` : exit 1 (cause absente du journal).
- L11-8 `-run TestMediaExtensionList` : exit 1 sur `internal/ops/media.go` (bloc de constantes vidéo, map des images). Premier essai
  du garde-rail par fichier : faux positif sur le catalogue statique du front ; analyse passée par déclaration avec l'exception
  des catalogues qui contiennent une extension propre au web.

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`, Go 1.26.9) :
- Gate du lot : `go test ./internal/api/... ./internal/archlint/ ./internal/media/... ./internal/ops/... ./internal/platform/auth/...
  -count=1` : premier passage exit 1 (`TestNoNewFrenchLabelLiteral` : le message français de L11-4 portait
  `admin_actions_replay_build.go` à 9 littéraux pour 8 ; corrigé par `humacore.FixedMessage`, `b2dd59395`), rejoué : exit 0,
  12 paquets ok, 0 `--- FAIL:`.
- Gate commun : `go build ./cmd/server/` exit 0 ; `go vet` (api, wire, middleware, handlers, archlint, platform/auth, ops, media)
  exit 0 ; `golangci-lint run --new-from-rev=16fcd294a` et `--new-from-merge-base=origin/main` (v2.12.2) sur ces paquets : 0 issue.
  `go run ./cmd/openapi-gen` : aucun diff ; `go test ./internal/api/ -run 'OpenAPI|Contract'` ok. Intégration
  `go test -tags=integration -p 1 ./internal/ops/...` : exit 0.
- Web : purge de `node_modules/.tmp`, `npm run typecheck` exit 0, `npm run lint` exit 0 (0 erreur, 26 avertissements préexistants),
  `npm run test:run` exit 0 (928 fichiers passés / 5 ignorés, 9732 tests passés / 23 ignorés) ; `lint:fields`, `lint:colors`,
  `lint-cross-feature-imports` (7 / plafond 7), `knip-ratchet` (0/0/0), `lint-contract-ratchet` : exit 0.
- Crochets pre-commit (gofmt, check-merge-conflict, docs-fr-sync, gitleaks, go-vet) verts sur les dix commits de code.

**Non-régression (règle 9)** : L11-1 : slugs réels (gamertags à espaces, accentués) toujours résolus (test). L11-2 : le front n'appelle
`/profiles/{player_slug}/titles/{slug}` qu'avec des titres de la liste du serveur (`features/settings/queries.ts`, `TitlesTab`).
L11-4 : aucun appelant front ; identifiants réels = UUID Halo. L11-5 : le front encode des slugs et gamertags issus des listes du
serveur (`features/admin/appearance/mutations.ts`, `features/admin/monitoring/mutations.ts`) ; aucun script ne sonde ces routes.
L11-6 : messages des codes réels désormais traduits ; code inconnu -> message générique comme avant. L11-8 : ensembles de formats
identiques (indexation, réconciliation, miniatures).

**Découvertes** : D-L11-1 à D-L11-4 (§6), non traitées.

Commits : `7abe61b48` (L11-1), `d2a36e632` (L11-2), `5bcb0fab8` (L11-3), `97cdb588e` (L11-4), `544e3ef83` (L11-5), `73d90b2eb` (L11-6),
`5489ce1db` (L11-7), `c6e9959ad` (L11-8), `b2dd59395` (message fixe de L11-4), puis le commit de clôture (plan, thought_log).

### F-1a - Corrections de la ronde 1 de la revue adversariale - 2026-10-16 - clos

Base : `git log --oneline -1` = `247926cbd` avant l'item ; branche `feat/securite-2026-10-15`. Références rouvertes avant d'agir :
`assets/store_localfs.go` (`Path` : `ref.ID + "." + ref.Variant` puis `filepath.Join`), `handlers/assets.go` (`v` décodé, `TrimSpace`
seul), `assets/gate.go`, `assets/catalog.go` (`any` : erreur comptée comme « sans ligne »), `assets/resolver_default.go`,
`server_apiv1.go` (prédicat de confiance `IsAuthenticated`, vrai en démo ; la démo est hors ligne : `discovery_client.go` passe par
`netguard`), `ops/media_thumbnails.go` et `media/ffmpeg_slot.go` (miniature sur `PoolTranscode`), `handlers/media_upload.go`
(`acquireUploadSlot` 30 s puis `ParseMultipartForm` -> 413), `cmd/server/main.go` (`ReadTimeout` 15 s), `handlers/match_exclusion.go`
(seul `ErrDBLocked` -> 503), `platform/coalesce/runs.go` (`fn(ctx)` du premier demandeur), `service/match_exclusion_service.go` et
`platform/duckdb/match_exclusion_repo.go` (lecture `IsExcluded` hors du lease, écriture sous lease), `features/match-view/MatchMediaTab.tsx`,
`features/home/RecentMediaRail.tsx`, `features/media/CoverFlowModal.tsx` (`ownsCurrentMedia` permissif faute de gamertag de
référence), `service/media_requester.go` + `domain.CanModifyMedia` (propriétaire ou admin), `handlers/openspartan_import.go` (3 sites)
et `service/media_service_upload.go` (1 site), les quatre garde-rails visés par R1-11.

**Statuts** : R1-1 à R1-11 `[x]`. Aucun `[!]`, aucun `[~]`.

- **R1-1** `handlers/assets.go:96-100` : `v` non vide hors forme UUID (`analysis.IsRawAssetUUID`) -> 400 `v invalide` avant le stock et
  toute source distante. `assets/catalog.go` (`classifyMap`) : la version est jugée par le catalogue — version courante de
  `maps_catalog.current_version_id` -> `VerdictKnown` ; autre version d'une carte connue -> `VerdictUnverifiable`, récupérée pour un
  appelant de confiance seulement (`Gate`). Preuve de classe de coût de la route des cartes dans `bare_routes_ratchet_test.go` :
  `TestAssetRoutes_MapImage_NewVariantCallsOutOnlyForTrustedCallers` (vraie route, vrai `MetadataCatalog` sur DuckDB : 50 versions
  inventées -> 404 et 0 appel sortant pour un anonyme ; version du catalogue -> 1 appel ; appelant de confiance -> appel).
  Tests : `handlers/assets_map_route_test.go` (3 tests), `TestMetadataCatalog_ClassifyMapVersion`.
- **R1-2** `assets/ref_validate.go` (`Ref.Validate`) : caractère de contrôle, `\`, `:` refusés dans tout composant ; `/`, `.`, `..`
  refusés dans kind, titre, variante ; élément `.`/`..` refusé dans l'identifiant (chemin GameCMS, éléments vides tolérés).
  `LocalFSStore.Path` rend `(string, error)` : `Validate` puis confinement (`filepath.Rel` + `filepath.IsLocal` sous la racine du
  kind) ; `LookupBinary`/`PersistBinary` refusent (`ErrInvalidRef`). `DefaultResolver.Get/LookupLocal/Refresh` et `Gate.Get` refusent
  une ref invalide avant tout travail ; le handler la sert comme une absence (404). nginx : `packaging/nginx/snippets/levelup-assets-v.conf`
  (contenu exact du VPS), `include /etc/nginx/snippets/levelup-assets-v.conf;` dans la location `/api/v1/assets/` de `levelup.conf` et
  `demo.conf`, pré-requis en tête des deux sites et dans `RUNBOOK_DEPLOY_CHECKLIST.md` ; garde-rail `archlint/nginx_assets_v_test.go`
  (forme `if ($arg_v ~* "…") { return 400; }`, l'expression refuse `..`, `/`, `\`, `%2f`, `%5c`, `%2e` en toute casse et laisse
  passer un UUID ; chaque server 443 inclut le filtre ; auto-preuve 12 mutations). Tests : `store_localfs_confinement_test.go`
  (13 refs piégées : rien lu ni écrit hors de la racine ; 5 refs légitimes servies), `TestGate_InvalidRefIsRefusedBeforeAnyWork`,
  `TestAssetRoutes_MapImage_TraversalReadsNothing` (vraie route, `secret.png` hors du stock).
- **R1-3** `media/thumbnail.go` : `RunWebPThumbnail` (arguments déplacés d'`ops`) tient `PoolThumbnail` (1 place, délai maximal
  `thumbnailMaxDuration` 5 min, `maxDurationFor`) au lieu de l'unique place des transcodages. Garde-rail `ffmpeg_single_launcher_test.go` :
  trois pools nommés exigés (ajout daté ; aucun pool hors du lanceur ; cas « deux pools seulement » ajouté). Tests :
  `TestRunWebPThumbnail_DoesNotWaitForATranscode`, `TestRunFFmpeg_TranscodeDoesNotWaitForAThumbnail`,
  `TestRunWebPThumbnail_IsBoundedByItsOwnMaxDuration`, `TestThumbnailArgs`.
- **R1-4** `handlers/media_upload.go` : une fois la place obtenue, `restartBodyReadDeadline` (`http.ResponseController.SetReadDeadline`,
  `defaultUploadBodyReadBudget` 15 s) ; `refuseUnreadableUpload` : `MaxBytesError` -> 413, échéance de lecture dépassée -> 503
  `upload_busy` + `Retry-After` (plus de 413 trompeur). Constantes et options d'envoi rangées dans `media_upload.go` (`media.go` repassait
  à 513 lignes). Tests (vrai `http.Server`) : `TestUploadHandler_SlotWaitBeyondReadTimeoutIsServed` (ReadTimeout 200 ms, place libérée à
  600 ms, corps de 1 Mio -> 200), `TestUploadHandler_BodyReadDeadlineAnswers503` (corps à moitié envoyé -> 503 + Retry-After).
- **R1-5** `handlers/match_exclusion.go` : `isDBBusy` (verrou d'écriture du joueur, `ErrSwapTimeout`, `ErrSwapFailed`,
  `ErrProviderClosed`) -> 503 `db_busy` + `Retry-After`. Chaîne vérifiée : recalcul -> `acquireSharedReaderStandalone` (`%w`) ->
  `coalesce` (erreur rendue telle quelle) -> service (`%w`). Tests : `TestMatchExclusionHandler_Set_BusyDatabaseAnswers503` (4 causes),
  `TestMatchExclusionService_SharedSwapTimeoutIsWrapped`.
- **R1-6** `platform/coalesce/runs.go` : le recalcul et ses rattrapages tournent dans leur goroutine sous `context.WithoutCancel` ;
  chaque demandeur attend au plus jusqu'à l'annulation de son propre contexte ; une panique du recalcul devient une erreur journalisée
  (les demandeurs reçoivent un résultat) ; commentaire du contrat réécrit. Tests : `TestRuns_FirstRequesterCancellationDoesNotStopTheRecompute`
  (premier demandeur annulé : `context.Canceled` aussitôt, le recalcul ne voit pas l'annulation, rattrapage servi au demandeur absorbé),
  `TestRuns_PanicBecomesAnError`.
- **R1-7** Port `MatchExclusionRepository.SetExclusion` rend `(changed, err)` ; `IsExcluded` retiré (lecture hors verrou, plus aucun
  appelant). `MatchExclusionRepo.SetExclusion` lit la vue `_latest`, décide et insère sous le lease `KindPlayer`, sur la même connexion.
  Service : no-op si rien n'est écrit et aucun recalcul en attente. Tests : `TestMatchExclusionService_OverlappingExcludeIncludeEndsOnTheLastDemand`
  (exclusion retenue entre lecture et écriture, réinclusion arrivée pendant ce temps : état final = réinclus),
  `TestMatchExclusionRepo_SetExclusion_DecidesOnLatestState` (intégration, base sur fichier : 2 écritures pour 5 demandes).
- **R1-8** `assets/catalog.go` : `presence` rend `VerdictUnavailable` quand une requête échoue (base illisible) ; une table absente
  (`duckdb.IsCatalogError`, type d'erreur du pilote, `platform/duckdb/catalog_error.go`) reste « sans ligne » (cache négatif conservé
  pour un référentiel absent). `Gate` : `VerdictUnavailable` -> `ErrNotFound`, sans appel sortant ni entrée au cache négatif. Tests :
  `TestGate_CatalogReadErrorIsNotNegativelyCached`, `TestMetadataCatalog_UnreadableStoreIsUnavailable`, `TestIsCatalogError`.
- **R1-9** API : `MatchAssociatedMedia.owner_gamertag` (`media_files.player_slug`, même sémantique que la galerie), lu par Q24 ;
  `openapi.yaml` et `generated.ts` régénérés. Front : `features/media/usePlayerGamertag.ts` (source unique ; garde-rail
  `usePlayerGamertag.guard.test.ts`, `MediaPage` migrée) ; `MatchMediaTab` et `RecentMediaRail` passent le gamertag de référence à la
  visionneuse et aux vignettes ; `_mediaItemRow` projette le propriétaire ; `CoverFlowModal` : « Réassocier » = propriétaire ou admin
  (`domain.CanModifyMedia`). Tests : `MatchMediaTab.reassociate.test.tsx`, `RecentMediaRail.reassociate.test.tsx` (clip d'autrui masqué,
  le sien montré, admin montré), `TestBuildMediaTab_CarriesTheOwner`.
- **R1-10** `handlers/openspartan_import.go` : `removeTempUpload` (échec journalisé, fichier absent ignoré) aux trois sites ; la copie est
  fermée avant tout retrait (un fichier ouvert ne se supprime pas sous Windows) et son erreur de fermeture n'est plus ignorée.
  `service/media_service_upload.go` : `storeUploadedFile(ctx, …)` journalise un retrait échoué de la copie `.part`. Garde-rail
  `archlint/no_discarded_remove_test.go` (`os.Remove`/`RemoveAll` dont l'erreur est écartée, dans `internal/api` et `internal/service` ;
  allowlist datée : `help.go`, 2 sites, D-F1a-1). Test : `TestRemoveTempUpload_LogsFailure`.
- **R1-11** (a) `no_readall_on_upload_test.go` : `UploadedFile` marque un fichier d'envoi ; `ReadAll` reconnu par chemin d'import
  (`io`, `io/ioutil`), quel que soit le nom local. (b) `client_ip_test.go` : littéral `True-Client-IP`/`X-Forwarded-For`/`X-Real-IP`
  (toute casse) interdit hors de `clientIPHeaderAllowlist` (`internal/api/server.go`, middleware d'IP, entrée datée). (c)
  `session_player_slug_test.go` : `currentPlayerSlugReads` voit toute lecture du champ de session `CurrentPlayerSlug` dans handlers,
  service, domain (écriture et méthode homonyme ignorées) ; allowlist datée `currentPlayerSlugAllowlist`. (d) `slots.Pool.Waiting`
  (journalisé avec le 503 `upload_busy`) ; `TestUploadHandler_AtMostTwoUploadsInFlight` attend trois demandes en attente au pool au
  lieu de dormir 150 ms ; `TestPool_WaitingCountsBlockedAcquirers`.

**Écarts au texte, justifiés** :
- R1-11 (c) : l'allowlist porte trois fichiers et non une seule exception d'affichage : `bootstrap_service.go` (profil affiché par
  défaut) et `session_context.go` (réponse de `/session/context`) sont des affichages ; `domain/session.go` lit le champ pour
  `Session.IsEmpty` (persistance du cookie), ni affichage ni décision d'accès. Toutes datées et motivées.
- R1-3 : le garde-rail des pools passe de 2 à 3 pools exigés (contrat changé par l'item, même rigueur : tout pool hors du lanceur reste
  rouge ; un lanceur à 2 pools devient rouge).
- R1-8 : une table absente n'est pas une « erreur de lecture » au sens de l'item (absence durable, ex. métadonnées d'un titre sans
  référentiel de médailles) : elle reste « inconnue » et mémorisée, reconnue par le type d'erreur du pilote, jamais par le message.
- R1-10 : `saveUploadedDB` traite désormais l'erreur de fermeture de la copie comme un échec de l'envoi (400 `upload_failed`) — un
  fichier mal fermé ne doit pas être importé.
- R1-4 : `Pool.Acquire` (R1-11 d) prend une place libre sans passer par l'attente ; seul un appel sur un pool plein compte dans
  `Waiting`.

**Preuves du rouge** (sorties dans `%LOCALAPPDATA%\Temp\f1a\`) :
- R1-2 `go test ./internal/assets/ -run 'TestLocalFSStore_(Trapped|Legitimate)'` : exit 1, 12 sous-tests sur 13 FAIL (lecture de
  `secret.png` hors de la racine ou écriture hors racine ; l'octet nul échouait déjà à l'appel système) ; les refs légitimes passaient.
- R1-1/R1-2 `go test ./internal/api/handlers/ -run TestAssetRoutes_MapImage` : exit 1, 3 FAIL — version inventée servie en 302 (appel
  sortant) pour un anonyme ; 9 `v` mal formés (`x`, `..`, `%2F..%2Fsecret`, `/../../../../secret`, antislash, UUID à 37 caractères…)
  servis en 302 avec 9 appels sortants ; `?v=/../../../../secret` sur la vraie route : 200 avec les octets de `secret.png`.
- nginx `go test ./internal/archlint/ -run NginxAssets` : exit 1, filtre absent, `include` absent de `demo.conf:92` et `levelup.conf:137`.
- R1-3 `go test ./internal/media/ -run TestRunWebPThumbnail` (après le seul déplacement des arguments) : exit 1, « la miniature a
  attendu le transcodage : attente de la place ffmpeg: context deadline exceeded » ; garde-rail des pools rouge avec le 3e pool tant que
  le compte exigé valait 2.
- R1-4 : exit 1, 2 FAIL — 413 `upload_too_large` pour un clip de 1 Mio dont la place s'est libérée après `ReadTimeout` ; corps
  interrompu : aucune réponse en 10 s (le serveur attendait indéfiniment).
- R1-5 : exit 1, 3 sous-tests FAIL (500 `exclusion_error` pour `ErrSwapTimeout`, `ErrSwapFailed`, `ErrProviderClosed`).
- R1-6 : exit 1, « le premier demandeur annulé attend toujours la fin du recalcul ».
- R1-7 : exit 1, « état final : exclu, alors que la réinclusion est arrivée après l'exclusion ».
- R1-8 : exit 1, « 1 refus mémorisé(s) sur une erreur de lecture du catalogue, attendu 0 ».
- R1-9 `npx vitest run …reassociate.test.tsx` : 2 FAIL (« Réassocier » affiché sur le clip d'un coéquipier et sur celui d'un autre
  joueur du rail) ; garde-rail `usePlayerGamertag` rouge sur `MediaPage.tsx` avant migration.
- R1-10 `-run DiscardedRemove` : exit 1, 4 sites (`openspartan_import.go:155,196,208`, `media_service_upload.go:228`).
- R1-11 (a) 3 cas FAIL « non signalé » (service, `stdio`, `iu`) ; (b) 3 cas FAIL « défaut non signalé » ; (c) détecteur absent
  (compilation), puis code réel rouge : 4 lectures dans 3 fichiers avant l'allowlist ; (d) `Waiting` absent (compilation), puis
  mutation (place d'envoi retirée du handler, source restaurée, `git diff` vérifié) : FAIL « demandes en attente d'une place = 0,
  attendu 3 » en 5 s.

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`, Go 1.26.9) :
- Suites complètes des paquets touchés : `go test ./internal/assets/... ./internal/api/... ./internal/archlint/ ./internal/media/...
  ./internal/ops/... ./internal/platform/coalesce/ ./internal/platform/duckdb/... ./internal/platform/slots/ ./internal/port/...
  ./internal/service/... ./internal/domain/... ./internal/sync/... ./cmd/server/ -count=1` : exit 0, 42 paquets ok, 0 `--- FAIL:`
  (archlint complet, `OpenAPI`/`Contract` de `./internal/api` compris). `internal/sync` inclus : `coalesce` y sert le recalcul
  d'exclusion.
- Intégration : `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/platform/duckdb/...
  ./internal/service/... ./internal/platform/coalesce/ -count=1` : exit 0, 25 paquets ok, 0 `--- FAIL:` ; `no_art_patterns_test.go`
  non modifié.
- Gate commun : `go build ./cmd/server/` exit 0 ; `go vet` des paquets touchés exit 0, et `go vet -tags=integration` (duckdb, service,
  sync) exit 0 ; `golangci-lint run --new-from-rev=247926cbd` (v2.12.2) sur ces paquets : 0 issue ; forme CI
  `--new-from-merge-base=origin/main` : 0 issue. API changée (R1-9) : `go run ./cmd/openapi-gen` puis `npm run generate-types` (un
  champ), régénération rejouée sans diff.
- Web : purge de `node_modules/.tmp`, `npm run typecheck` exit 0, `npm run lint` exit 0 (0 erreur, 26 avertissements préexistants),
  `npm run test:run` exit 0 (931 fichiers passés / 5 ignorés, 9740 tests passés / 23 ignorés) ; `lint:fields`, `lint:colors`,
  `lint-cross-feature-imports` (7 / plafond 7), `knip-ratchet` (0/0/0), `lint-contract-ratchet` : exit 0.
- Crochets pre-commit (gofmt, check-merge-conflict, docs-fr-sync, gitleaks, go-vet) verts sur les quatorze commits de code. Rien poussé,
  aucune opération VPS.
- Premiers passages rouges corrigés en cours d'item : `TestNoAdHocRepoRootLadderInTests` (échelles `../../../..` littérales dans les
  pièges de traversée, construites par `strings.Repeat`) ; `media.go` à 513 lignes (constantes d'envoi rangées dans
  `media_upload.go`, `578e82fbe`).

**Non-régression (règle 9)** :
- R1-1/R1-2 : URL produites par le serveur : `match_view_repo.go` (`?v=` = `MapVersionID` UUID Halo), `asset_service.go` (sans `v`) ;
  identifiants de carte = UUID ; aucun appelant front n'écrit `v` lui-même. Une carte vue depuis la vue match par un compte connecté
  reste récupérée quelle que soit sa version ; un anonyme (prod) n'obtient à distance que la version courante du catalogue, le stock
  local servant le reste. Démo : appelant de confiance, mais hors ligne (`netguard`). Chemins GameCMS des images de passe et de
  joueur (barres obliques, barre de tête) toujours servis (test).
- R1-3 : miniatures générées pendant l'envoi comme avant le chantier ; une miniature à la fois.
- R1-4 : envoi derrière nginx (corps en tampon) inchangé ; seul un corps qui n'arrive pas reçoit 503 `upload_busy`, message FR + EN
  déjà existant (`common.media.upload_busy`).
- R1-5 : l'en-tête de la vue match (`MatchHeader.card.tsx`) affiche le même message générique qu'avant pour toute erreur autre que
  « match classé » ; seul le statut change (503 `db_busy` + `Retry-After`, comptés en indisponibilité passagère, au lieu de 500).
- R1-7 : réponses inchangées (204) ; un état demandé identique n'écrit toujours rien.
- R1-9 : changement visible : sur l'onglet Médias d'un match et le rail de l'accueil, les clips des coéquipiers n'offrent plus
  « Réassocier » (le serveur les refusait en 403) et leurs vignettes portent la pastille du propriétaire, comme dans la galerie ; un
  admin voit « Réassocier » partout (le serveur l'y autorise). Aucune chaîne i18n ajoutée.
- R1-10 : comportement inchangé, sauf une copie OpenSpartan mal fermée qui refuse désormais l'envoi.

**Découvertes** : D-F1a-1 à D-F1a-6 (§6), non traitées.

Commits : `7f82b61ba` (R1-1, R1-2), `64177f8d5` (R1-3), `f608cbbe4` et `578e82fbe` (R1-4), `ddaca4a55` (R1-5), `bcbb7331b` (R1-6),
`214868201` (R1-7), `ac7c1c126` (R1-8), `640d84077` (R1-9), `ca1257cf3` (R1-10), `86ae27bb3`, `c65d98db6`, `5813b745b`, `2e40c5d5e`
(R1-11), puis le commit de clôture (plan, thought_log).

### F-1b - Corrections de la ronde 2 de la revue adversariale - 2026-10-16 - clos

Base : `git log --oneline -1` = `2b81c527b` avant l'item ; branche `feat/securite-2026-10-15`. Références rouvertes avant d'agir :
`handlers/media_upload.go:194` (`http.NewResponseController(w).SetReadDeadline`), `middleware/session.go` (`sessionResponseWriter`),
`middleware/slog_logger.go` (`statusResponseWriter`), toutes les enveloppes du paquet (`grep -rn "http.ResponseWriter$"
internal/api/middleware` : ces deux seules ; `chimiddleware.Compress` expose déjà `Unwrap`), `server.go` (`applyTransverseMiddlewares`),
`platform/coalesce/runs.go` et ses deux utilisateurs (`sync/match_recomputer.go`, `service/friends_orchestrator_service.go`),
`cmd/server/main.go` (séquence d'arrêt), `platform/duckdb/match_view_repo.go:142` et `queries_match.go` (Q13MatchMeta : le `?v=` émis
est `match_registry.map_version_id`, base partagée du titre), `assets/catalog.go`, `assets/gate.go`, `assets/wire.go`,
`server_apiv1.go` (construction de `assets.AssetConfig`), `config/player_resolver.go` (`SharedReaderForTitle`).

**Statuts** : R2-1 à R2-3 `[x]`. Aucun `[!]`, aucun `[~]`.

- **R2-1** `sessionResponseWriter` et `statusResponseWriter` déclarent `Unwrap() http.ResponseWriter` (`session.go:184`,
  `slog_logger.go:189`) : `http.ResponseController` remonte la pile jusqu'à la connexion. Tests à travers la pile transverse réelle
  (`applyTransverseMiddlewares` : session, journal, compression, CSRF, limiteur...) et un vrai `http.Server` :
  `TestTransverseStack_ReadDeadlineReachesTheConnection` (sonde : 204) et `TestTransverseStack_UploadWaitingBeyondReadTimeoutIsServed`
  (`ReadTimeout` 200 ms, place libérée à 600 ms, clip de 1 Mio : 200), `api/media_upload_transverse_test.go`. Garde-rail AST
  `archlint/response_writer_unwrap_test.go` : tout type struct de `internal/api` portant un champ `http.ResponseWriter` (incorporé ou
  nommé, quel que soit l'alias de `net/http`) déclare `Unwrap() http.ResponseWriter` dans son paquet ; aucune allowlist ; détection
  éprouvée (incorporée, nommée, mauvaise signature, autre type rendu, récepteur valeur ou pointeur, méthode dans un autre fichier du
  paquet, méthode d'un autre paquet).
- **R2-2** `coalesce.MaxRunDuration` (10 min) : chaque recalcul et chaque rattrapage tournent sous `context.WithTimeout` du contexte
  détaché (`runBounded`) ; à l'échéance l'erreur nomme l'échéance, est journalisée en ERROR et rendue aux demandeurs (clé marquée en
  échec). `coalesce.Drain(ctx)` : suivi des boucles en vol de toutes les `Runs` du process ; dès son début, refuse toute demande neuve
  ou à absorber (`ErrDraining`), puis attend la fin des recalculs en vol, rattrapage compris, au plus jusqu'à `ctx`. `cmd/server` :
  `drainCoalescedRecomputes` (`shutdown_recomputes.go`, même borne, échec journalisé) appelé juste après `srv.Shutdown` et avant
  `duckdb.CloseAll` (`main.go:1362`, une ligne) ; ordre tenu par `TestShutdown_DrainsRecomputesBeforeCloseAll` (AST de `main`). Tests :
  `runs_shutdown_test.go` (échéance atteinte -> erreur journalisée ; échéance par défaut = `MaxRunDuration` ; arrêt qui attend ; arrêt
  borné ; refus après l'arrêt ; demande non absorbée pendant l'arrêt). Commentaire du contrat de `Runs` réécrit. D-F1a-6 traitée.
- **R2-3** Source relevée sur pièces : le `?v=` des images de carte (`match_view_repo.go:143`) vient de `match_registry.map_version_id`
  (Q13MatchMeta), base partagée du titre (table non append-only, une ligne par match). `platform/duckdb/played_map_versions_repo.go` :
  `ForEachPlayedMapVersion` (`SELECT DISTINCT map_id, map_version_id FROM match_registry`, lecteur du provider, jamais un writer).
  `assets/played_map_versions.go` : couples tenus en mémoire par titre, relus sur une absence au plus une fois par minute (une version
  inventée ne coûte aucune lecture de base). `assets/catalog.go` (`classifyMap`, `classifyPlayedVersion`) : version courante du
  catalogue OU couple joué -> `VerdictKnown` (récupérable par tous), y compris pour une carte encore absente des métadonnées ; version
  jamais jouée : `VerdictUnverifiable` (carte connue : appelant de confiance seulement, inchangé) ou `VerdictUnknown` (carte
  inconnue) ; base partagée illisible : jugement du catalogue seul, `VerdictUnavailable` pour une carte inconnue (aucun refus mémorisé
  sur une lecture en échec). Branchement : `AssetConfig.PlayedMapVersions` (`wire.go`), renseigné dans `server_apiv1.go:1420` par
  `cfg.SharedReaderForTitle(titleID)` (title-agnostic, aucun slug en dur). Tests : `TestAssetRoutes_MapImage_PlayedVersionIsServedToAnonymous`
  (vraie route, vrai `MetadataCatalog` sur DuckDB, base partagée au schéma de production `sync.EnsureSharedSchema` : version jouée ->
  302 pour un anonyme, carte hors métadonnées jouée -> 302, 100 versions inventées et une version jouée sur une autre carte -> 404,
  2 appels sortants au total) ; `played_map_versions_test.go` (une lecture pour N absences, relecture après l'intervalle, titres
  séparés, erreur rendue, 8 cas de classement, câblage de `New`) ; `TestAssetConfig_WiresPlayedMapVersions` (AST de `server_apiv1.go`).
  La preuve de classe de coût de la route (`bare_routes_ratchet_test.go`) reste `TestAssetRoutes_MapImage_NewVariantCallsOutOnlyForTrustedCallers`,
  toujours vraie.

**Écarts au texte, justifiés** :
- R2-1 : garde-rail sur `internal/api` entier au lieu du seul paquet `middleware` (plus strict, même règle ; aucune autre enveloppe).
- R2-2 : `ErrDraining` ajouté : sans refus des demandes neuves, un recalcul lancé après la fin de l'attente écrirait sur des bases fermées.
- R2-3 : une carte absente des trois tables de métadonnées mais jouée devient connue (risque résiduel de L5-2 levé) : c'est une
  « version présente dans de vrais matchs » au sens de la décision de l'utilisateur.

**Preuves du rouge** (sorties dans `%LOCALAPPDATA%\Temp\f1b\`) :
- R2-1 `go test ./internal/api/ -run TestTransverseStack_` : exit 1, 2 FAIL — sonde : `statut 500 ... feature not supported` ; envoi :
  `upload: échéance de lecture du corps non reportée err="feature not supported"` puis `statut 503, attendu 200`. Garde-rail
  `go test ./internal/archlint/ -run TestResponseWriter` : exit 1, `session.go:132 sessionResponseWriter`, `slog_logger.go:165 statusResponseWriter`.
- R2-2 tests de `coalesce` : échec de compilation avant le correctif (`inflightRuns`, `maxDuration`, `MaxRunDuration` indéfinis) ;
  mutations après correctif, source restaurée (`cmp`) : échéance retirée -> `TestRuns_RecomputeStopsAtItsDeadline` et
  `TestRuns_DefaultDeadlineIsMaxRunDuration` FAIL ; attente de `drain` court-circuitée -> 3 tests `TestDrain_` FAIL. Câblage :
  `TestShutdown_DrainsRecomputesBeforeCloseAll` FAIL `0 appel(s) à drainCoalescedRecomputes dans main` avant l'ajout dans `main.go`.
- R2-3 `go test ./internal/api/handlers/ -run TestAssetRoutes_MapImage` (source des versions jouées branchée, jugement inchangé) :
  exit 1, `anonyme, version jouée : statut 404, attendu 302` ; `go test ./internal/assets/ -run Played` : exit 1, 3 cas FAIL (version
  jouée = Unverifiable, carte hors catalogue jouée = Unknown, carte inconnue sur base illisible = Unknown au lieu d'Unavailable).
  Mutations de câblage, sources restaurées : ligne `PlayedMapVersions` retirée de `server_apiv1.go` -> `TestAssetConfig_WiresPlayedMapVersions`
  FAIL ; branchement retiré de `wire.go` -> `TestNew_WiresPlayedMapVersions` FAIL.

**Gates** (sorties réelles, `GOCACHE=go-build-securite`, `CGO_ENABLED=1`, Go 1.26.9) :
- Suites complètes des paquets touchés : `go test ./cmd/server/ ./internal/api/... ./internal/assets/... ./internal/archlint/
  ./internal/platform/coalesce/ ./internal/platform/duckdb/... ./internal/service/... ./internal/sync/ -count=1` : exit 0, 20 paquets ok,
  0 `--- FAIL:` (archlint complet, `OpenAPI`/`Contract` de `./internal/api` et ratchet des routes nues compris). `coalesce` `-count=20` : exit 0.
- Intégration : `go test -tags=integration -p 1 ./internal/platform/duckdb/... ./internal/platform/coalesce/ ./internal/assets/...
  ./internal/service/... ./internal/sync/... ./internal/persist/... -count=1` : exit 0, 27 paquets ok, 0 `--- FAIL:` ;
  `no_art_patterns_test.go` non modifié.
- Gate commun : `go build ./cmd/server/` exit 0 ; `go vet` (cmd/server, api, assets, archlint, coalesce, duckdb) exit 0, et
  `go vet -tags=integration` (duckdb, coalesce) exit 0 ; `golangci-lint run --new-from-rev=2b81c527b` (v2.12.2) sur ces paquets :
  0 issue ; forme CI `--new-from-merge-base=origin/main` : 0 issue. Aucune route Huma ni type d'API changé : pas de régénération.
  Web non touché.
- Crochets pre-commit (gofmt, check-merge-conflict, docs-fr-sync, gitleaks, go-vet) verts sur les trois commits de code.

**Non-régression (règle 9)** :
- R2-1 : l'envoi derrière nginx (corps en tampon) est inchangé ; un envoi qui attend sa place au-delà de 15 s aboutit désormais en
  production au lieu de recevoir 503. `ResponseController` n'a pas d'autre appelant (`grep NewResponseController`) ; les `Flush`
  par assertion de type (`w.(http.Flusher)`) ne changent pas.
- R2-2 : réponses de l'exclusion et de la liste d'amis inchangées ; un recalcul de plus de 10 min échoue (journalisé) au lieu de tourner
  sans fin ; à l'arrêt, les recalculs en vol finissent avant la fermeture des bases (dans la limite de D-F1b-1).
- R2-3 : l'image de carte de la vue match (version du match) est servie à un visiteur anonyme comme à un compte connecté ; une carte
  toute neuve jouée n'attend plus le rafraîchissement du catalogue. Un appelant de confiance garde son repli (D-F1b-3).

**Découvertes** : D-F1b-1 à D-F1b-3 (§6), non traitées. D-F1a-6 traitée par R2-2.

Commits : `51e603f13` (R2-1), `976494e67` (R2-2), `70b171092` (R2-3), puis le commit de clôture (plan, thought_log).
