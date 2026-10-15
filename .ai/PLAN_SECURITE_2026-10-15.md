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

**Hors plan (backlog, motif)** : changement de mot de passe exigeant l'ancien (change l'UX et
le front) ; réglages illisibles → verrouillé (choix documenté de l'ADR 0035, à rediscuter) ;
durcissement des conteneurs, unité systemd de l'ouvrier, allowlist IP du protocole ouvrier
(opérations VPS de calcul) ; correction exhaustive des 121 `err.Error()` en 4xx (gelés par
ratchet en L9, corrigés pour les routes anonymes seulement).

## 1. Règles d'exécution

1. **Ordre strict** L0 → L9. Un lot est **clos** quand : gate exact passé, tous les items statués,
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
   L8, L9.
7. **Interdits** : push/merge sur `main` ; fusion dans `feat/v75` sans accord ; opération VPS sans
   annonce ; `git stash` ; `git add -A` ; garde-rail affaibli sans justification datée ; `t.Skip`
   non justifié.
8. **Reprise de session** : relire `plan-execution`, puis le journal §7 ; reprendre à la première
   case non statuée du lot courant.

Gate commun de fin de lot (en plus du gate propre) depuis `apps/go-api` :
`go build ./cmd/server/ && go vet <paquets touchés> && golangci-lint run <paquets touchés>` ;
si l'API change : `make generate-types` puis `go test ./internal/api/ -run 'OpenAPI|Contract'` ;
si `apps/web` change : purge `node_modules/.tmp`, `npm run typecheck && npm run lint && npm run test`.

## 2. Lots

### L0 — Chaîne d'outils, dépendances et CI (C9-1, C9-2, C9-3) — Sonnet — effort S

- [ ] L0-1 `apps/go-api/go.mod` : `go 1.26.9` (commentaire d'en-tête mis à jour : raison = 19
      vulnérabilités atteintes, cf. registre).
- [ ] L0-2 `Dockerfile` : image de build épinglée `golang:1.26.9-bookworm` (fin de l'image flottante).
- [ ] L0-3 `golang.org/x/image` → v0.45.0 (`go get` + `go mod tidy`).
- [ ] L0-4 `apps/web/package.json` : `overrides` `postcss-selector-parser` `^7.1.6` ;
      `npm audit --omit=dev` à 0.
- [ ] L0-5 Job CI bloquant dans `.github/workflows/ci.yml` : `govulncheck ./...` (Go du `go.mod`)
      + `npm audit --omit=dev --audit-level=high`.
- [ ] L0-6 Épinglage par SHA (commentaire de version) de `appleboy/ssh-action` (3 usages,
      `deploy.yml`) et `softprops/action-gh-release` (`release.yml`).
- [ ] L0-7 `release.yml` : `permissions: contents: read` au niveau workflow, `contents: write`
      sur le seul job de publication.
- [ ] L0-8 `scripts/git-hooks/lefthook/govulncheck.sh` : commentaire périmé corrigé (le hook
      reste indicatif, la CI devient l'autorité).
- [ ] L0-9 Garde-rail : `internal/archlint/ci_workflows_security_test.go` — (a) tout `uses:` hors
      `actions/` dans un workflow qui lit `secrets.` ou a `contents: write` est épinglé par SHA de
      40 caractères ; (b) `ci.yml` contient un job `govulncheck` sans `|| true`.
- Gate : `cd apps/go-api && go version && govulncheck ./...` (aucune vulnérabilité **appelée**) ;
  `go test ./internal/archlint/ -run 'Workflow|CI'` ; `cd apps/web && npm audit --omit=dev` ;
  `go build ./cmd/server/`.

### L1 — IP cliente et nginx du dépôt (C1-1, C1-2 garde locale) — Opus — effort M

- [ ] L1-1 Test rouge : pile transverse réelle (`server.go`) — une requête portant
      `True-Client-IP` et `X-Forwarded-For` forgés et `X-Real-IP` réel a pour clé de limiteur le
      `X-Real-IP` ; 301 requêtes à `True-Client-IP` variable → 429.
- [ ] L1-2 `server.go:625-626` : `ClientIPFromHeader("X-Real-IP")` si `TrustProxyHeaders`, sinon
      `ClientIPFromRemoteAddr` ; `RealIP` retiré.
- [ ] L1-3 `middleware/rate_limit.go` : clé = `GetClientIP(ctx)` (`httprate.KeyByIP` retiré).
- [ ] L1-4 `handlers/replay_local_gate.go` : `isLocalRequest` lit l'adresse préservée par
      `PreserveRemoteAddr` (accesseur exporté de `middleware/loopback.go`) ; test existant
      `replay_local_gate_test.go` réécrit pour passer par la pile réelle.
- [ ] L1-5 Échecs de connexion : compteur par identifiant normalisé et par IP sur
      `POST /auth/login` et `POST /auth/password` (décision §0) ; test rouge « 6e échec → 429,
      même en changeant d'IP pour un même identifiant ».
- [ ] L1-6 Journaux : l'attribut `ip` des journaux d'auth et de CSRF vient de `GetClientIP`.
- [ ] L1-7 `packaging/nginx/levelup.conf` et `demo.conf` réalignés sur la configuration appliquée
      au VPS le 2026-10-15 (voir §5) ; `rate_limit.conf` versionné sous `packaging/nginx/`.
- [ ] L1-8 Garde-rails : `internal/archlint/client_ip_test.go` interdit `chimiddleware.RealIP`,
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
- [ ] L9-5 Ratchet `internal/archlint/err_error_response_test.go` : compte gelé des
      `err.Error()` passés à `NewError`/`writeError` (121 au 2026-10-15), qui ne peut que baisser ;
      aucun nouveau site.
- [ ] L9-6 En-tête CSP minimal sans risque pour le SPA : `object-src 'none'; base-uri 'self';
      frame-ancestors 'none'` (scripts non restreints, décision de l'ADR à venir) ; test sur `/`,
      404, 429, `/static`.
- Gate : `go test ./internal/api/... ./internal/archlint/ -run 'Health|Ready|ErrError|CSP|SecurityHeaders|DeviceFlow'` ;
  front si L9-3.

## 3. Clôture du plan

- [ ] F-1 Revue adversariale du diff complet (`git diff a9a9ac78a...HEAD`) : 2 relecteurs aveugles
      (lentille « contrôle d'accès et identité » ; lentille « entrées, ressources, chemins ») +
      1 relecteur L6 (couverture des tests) ; 2 rondes maximum.
- [ ] F-2 `make gate-push` vert.
- [ ] F-3 Push de `feat/securite-2026-10-15`, CI verte au niveau job (`gh run list --branch`).
- [ ] F-4 Registre et backlog mis à jour (items fermés, `[sécurité]` et `[demo]` du backlog).
- [ ] F-5 Demande d'accord à l'utilisateur pour la fusion dans `feat/v75`.

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
- [ ] V-2 TLS 1.0/1.1 encore acceptés sur le site principal (le `server` 443 n'inclut pas
      `options-ssl-nginx.conf`, `nginx.conf` liste TLSv1 à TLSv1.3) : proposer
      `ssl_protocols TLSv1.2 TLSv1.3;` — **accord de l'utilisateur requis**.
- [ ] V-3 Au déploiement de la v7.5 : provisionnement de `data/demo_films/` (L8-11) et contrôle
      que la conf nginx versionnée (L1-7) correspond à la conf en place.

## 6. Découvertes

(vide)

## 7. Journal

(vide)
