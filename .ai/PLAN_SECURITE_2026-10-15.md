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

1. **Ordre strict** L0 → L10. Un lot est **clos** quand : gate exact passé, tous les items statués,
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

- [ ] L2-1 Test rouge : corps de 2 Mio sur `POST /auth/login`, `/auth/register`,
      `/session/context`, `/auth/password` → 413.
- [ ] L2-2 `humacore` : plafond par défaut de 1 Mio sur toute opération enregistrée sans
      `MaxBody` explicite ; commentaire faux de `humacore/operation.go:70` corrigé.
- [ ] L2-3 Garde-rail : test qui parcourt toutes les opérations Huma enregistrées
      (`humacore` / OpenAPI) et exige `MaxBodyBytes > 0`.
- [ ] L2-4 `handlers/session_context.go` : `locale` ∈ {fr, en} ; `player_slug` ≤ 64 caractères,
      alphabet de slug ; sinon 400 (le contrôle d'accès du slug est L3-1).
- [ ] L2-5 `platform/session/store.go` : `Save` refuse une session sérialisée > 8 Kio (erreur
      journalisée) ; TTL de 24 h pour une session sans utilisateur authentifié.
- [ ] L2-6 Test rouge leaderboard : `limit=1000000000` → réponse bornée à 200 entrées, aucune
      allocation proportionnelle à `limit` ; `service/leaderboard_service.go` plafonne à 200 ;
      `make` du repo dimensionné sur `min(limit, 200)`.
- [ ] L2-7 Garde-rail pagination : test qui parcourt l'OpenAPI et exige `maximum` sur tout
      paramètre entier `limit`/`page_size`/`count`/`n` ; liste datée des paramètres en chaîne
      (dont `leaderboard.limit`), chacun couvert par un test de plafond.
- Gate : `go test ./internal/api/... ./internal/platform/session/... ./internal/service/ -run 'Leaderboard|Session|MaxBody|Pagination|BodyLimit'`
  puis `go test ./internal/api/... ./internal/platform/session/...`.

### L3 — Identité du demandeur et accès aux données d'autrui (C4-1 à C4-8, D-A, D-B, D-C) — Opus — effort L

- [ ] L3-1 Test rouge puis correctif : `POST /session/context` refuse (403) un `player_slug` que
      `authz.CanAccessPlayer` n'autorise pas pour l'utilisateur courant.
- [ ] L3-2 Suppression et likes de médias : le demandeur et le liker sont dérivés du profil
      possédé par `authz.CurrentUser` (xuid), plus jamais de `SessionPlayerSlug` ; test « média
      d'autrui » × {supprimer, liker} → 403.
- [ ] L3-3 Association média → match : contrôle de propriétaire (règle de `CanDeleteMedia`) ;
      recherche par chemin complet ; le repli `file_name` n'est conservé que s'il est restreint au
      joueur (vérifier les appelants) ; test « média d'autrui » × associer → 403.
- [ ] L3-4 D-A : `RequirePlayerOwnership` ne transmet l'ensemble famille que si
      `!IsMutatingRequest(r)` ; vérifier `squadActorGuard` (Prestige) sur la même règle ; test
      « co-membre × toutes les routes mutantes du routeur » (walk chi) → 403.
- [ ] L3-5 Amendement de l'ADR 0029 (EN) : section « Group/family access is read-only (2026-10-15) ».
- [ ] L3-6 Provisionnement : `guardLinkedXboxIdentity` pour tout non-admin quel que soit
      `profile_mode` ; `ProfileService.CreatePlayer` refuse (409) de changer le xuid d'un profil
      existant hors admin ; test table-driven modes × non-admin ; vérifier la précondition avec
      `instance_locked=true` (état prod) et consigner.
- [ ] L3-7 Défi d'escouade : `assertMemberUser` dans `CreateSquadChallenge` ; commentaire faux de
      `api/prestige_player_slug_mw.go:26-29` corrigé ; garde-rail AST « toute méthode du service
      Prestige qui reçoit un `squadID` appelle `assertMemberUser` ».
- [ ] L3-8 D-C : `/healthz/home` sous `RequireAuth` + `RequireAdmin` ; commentaire faux corrigé ;
      e2e/harnais démo vérifiés (auth inactive → toujours joignable).
- [ ] L3-9 D-B import OpenSpartan : n'écrire que les matchs où le xuid de la session participe ;
      aucun `UpsertXUIDAlias` pour un xuid ≠ propriétaire (alias tiers : `INSERT OR IGNORE`
      seulement, jamais d'`UPDATE`) ; `match_id` validé (hexadécimal et tirets) ; test fixture
      « xuid tiers seul » → 0 ligne écrite, « alias tiers existant » → inchangé. Risque résiduel
      (falsification d'un match auquel l'importeur a participé) consigné au journal.
- [ ] L3-10 Couche B : contrôle de participation (404 `match_not_participant`) sur `/replay`,
      `/positions`, `/objective-events`, `/events` et la séquence tactique (rejeu public en prod) ;
      test table-driven sur toutes les routes `/matches/{match_id}/*`.
- [ ] L3-11 Garde-rails : `internal/archlint/session_player_slug_test.go` interdit
      `SessionPlayerSlug(` dans `handlers/`, `service/`, `domain/` hors affichage (allowlist
      vide au départ) ; ratchet « tout paramètre `player`/`player_slug`/`xuid`/`gamertag` d'une
      route hors `/players/{player_slug}` exige propriété ou admin ».
- Gate : `go test ./internal/api/... ./internal/authz/... ./internal/service/... ./internal/prestige/... ./internal/openspartan/... ./internal/archlint/`
  ; `go test -tags=integration -p 1 ./internal/persist/... ./internal/sync/...` (import OpenSpartan
  touche l'écriture partagée) ; OpenAPI régénéré si `/healthz/home` change de sécurité.

### L4 — Session et authentification (C5-1 à C5-5, C5-6 partiel) — Opus — effort M

- [ ] L4-1 Test rouge : device-flow terminé avec `ErrInstanceLocked` → session non authentifiée,
      tentative `failed`.
- [ ] L4-2 `handlers/auth.go` : `AuthReady`, identité et jetons écrits seulement après le succès
      de `OnAuthSuccess`.
- [ ] L4-3 Invitations : émission (`POST /groups/{id}/invites`) et création de groupe exigent un
      compte existant du magasin d'utilisateurs ; une invitation n'est valable que si son émetteur
      existe encore ; `Consume` avant `Create` (usage unique atomique).
- [ ] L4-4 Logout : suppression de la session côté serveur + cookie expiré ; test « SSO →
      déconnexion → lecture refusée ».
- [ ] L4-5 `RequireAdmin` relit le rôle via `authz.CurrentUser` (magasin injecté) ; test
      « rétrograder puis rejouer la session → 403 » ; idem `settings.go:216`, `media_delete.go:88`.
- [ ] L4-6 Rotation de l'identifiant de session aux trois connexions (mot de passe, OAuth,
      device-flow) : nouvel identifiant, ancienne session supprimée ; test « la valeur du cookie
      change ».
- [ ] L4-7 Configuration : `LEVELUP_AUTH_MODE` et `LEVELUP_REGISTRATION` normalisés
      (minuscules, espaces) ; valeur inconnue → refus au démarrage ; test table-driven.
- [ ] L4-8 Callback OAuth : jeton de rafraîchissement persisté seulement après le contrôle du
      verrou (`auth_xbox_oauth.go:209` après la décision).
- [ ] L4-9 Premier compte : rôle décidé dans `Create` sous verrou ; garde « dernier admin » sur
      `Delete`/`SetRole` ; test concurrent.
- [ ] L4-10 Journaux : identifiant de connexion tronqué (64 caractères), codes d'invitation
      retirés des journaux ; garde-rail grep interdisant `"code"` dans les attributs slog de
      `handlers/{admin,groups,user_auth}.go`.
- [ ] L4-11 `POST /auth/password` exige le mot de passe actuel quand le compte en a déjà un
      (compte Xbox sans mot de passe : premier mot de passe sans l'ancien) ; front : champ
      « Mot de passe actuel » / « Current password » (i18n FR + EN), test du hook ou du composant ;
      tests serveur des deux cas. Dépendance posée par L1-5 : `handleSetPassword` consulte déjà
      le limiteur d'échecs (`loginAttempts`) ; un mot de passe actuel incorrect y appelle
      `RecordFailure(*sess.Username, ip)`, un correct `RecordSuccess` (test « 6e mot de passe
      actuel faux → 429 »).
- [ ] L4-12 `authz.InstanceLocked` : en mode appliqué, une erreur de lecture des réglages
      verrouille (au lieu d'ouvrir) ; erreur journalisée ; test `authz_test.go` inversé ;
      amendement de l'ADR 0035 (EN) consignant le basculement et sa raison.
- Gate : `go test ./internal/api/... ./internal/authz/... ./internal/platform/session/... ./internal/platform/userstore/... ./internal/service/... ./internal/config/...`.

### L5 — Travaux coûteux (C3-1 à C3-6) — Sonnet — effort L

- [ ] L5-1 Device-flow : plafonds (§0) dans `attempt_store` → 429 ; test « 21e tentative → 429 ».
- [ ] L5-2 `/api/v1/assets/*` : identifiant validé contre le catalogue local (titre, médaille,
      carte…) avant tout fetch, sinon 404 sans appel ; cache négatif borné (TTL, taille) ;
      limiteur sortant du provider Halo partagé (plus recréé par requête) ; test « id inconnu →
      404, fetcher espion jamais appelé ».
- [ ] L5-3 Exclusion de match : le recalcul n'acquiert plus le writer partagé (lecture seule de la
      base partagée, écritures en base joueur via le persister existant) ; no-op si l'état ne
      change pas ; single-flight par joueur ; test « exclusion identique → aucun recalcul »,
      « recalcul → writer partagé jamais pris ».
- [ ] L5-4 Import OpenSpartan : validation du fichier avant toute bascule ; un import à la fois par
      compte (`FindActiveJob`, comme `backfill.go:111`) → 409 ; test « fichier invalide → writer
      non pris », « 2e import simultané → 409 ».
- [ ] L5-5 Envoi de médias : écriture en flux vers le disque (fin de `io.ReadAll`) ; sémaphores
      globaux (§0) ; tâche ffmpeg avec délai maximal.
- [ ] L5-6 Recherche de gamertags : `live` ignoré sans session authentifiée ;
      `POST …/engagement/recompute_coefficients` sous `RequireAdmin`.
- [ ] L5-7 Garde-rails : chaque entrée de `publicRoutesAllowlist` (`bare_routes_ratchet_test.go`)
      porte une classe `cheap` / `outbound` / `disk`, et `outbound`/`disk` exigent un test
      « anonyme → aucun appel sortant ni écriture non bornée » ; archlint interdisant
      `AcquireWriter`/`AcquireSharedWriterStandalone` dans un chemin déclenché par une route
      `/players` hors allowlist datée ; grep interdisant `io.ReadAll` sur un `multipart.File`.
- Gate : `go test ./internal/api/... ./internal/assets/... ./internal/service/... ./internal/platform/auth/... ./internal/archlint/`
  ; `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/platform/duckdb/...`.

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
- Gate : `go test ./internal/archlint/ -run 'Compose|Nginx'` ; `bash -n scripts/deploy-worker.sh`
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
  relayée ; aucun garde-rail ne tient cette topologie.
- **D-L1-2** (L1) `handlers/replay_local_gate.go` : `isLocalRequest` traite un hôte vide comme local (choix
  justifié par des transports de test). En service `RemoteAddr` n'est jamais vide, mais le défaut d'une garde
  de sécurité est ouvert.
- **D-L1-3** (L1) `features/auth/SetPasswordCard.tsx` : libellés français en dur (`LABELS`), sans version
  anglaise (l'écran en anglais affiche du français). L4-11 retouche ce composant.
- **D-L1-4** (L1) `packaging/nginx/demo.conf` (comme la démo en place) n'a pas `limit_req_status 429` : la
  limite `demo_limit` répond 503, que le front lit comme une panne et non comme un excès de tentatives.
- **D-L1-5** (L1) `archlint/no_french_label_literal_test.go` signale une baisse non figée :
  `service/seasons_catalog.go` compte 3 littéraux pour 4 alloués (allowlist à resserrer).

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
