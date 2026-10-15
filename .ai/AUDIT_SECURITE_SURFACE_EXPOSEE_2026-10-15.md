# Audit sécurité de la surface exposée — 2026-10-15

Campagne `adversarial-audit` demandée par l'utilisateur le 2026-10-15 (item « [sécurité] » de
`.ai/BACKLOG.md`). Base : `origin/feat/v75` @ `a9a9ac78a`, worktree `LevelUp-wt-securite`, branche
`feat/securite-2026-10-15`. L'audit ne corrige rien : ce registre devient le plan
`.ai/PLAN_SECURITE_2026-10-15.md` après accord de l'utilisateur.

## Cadrage

Niveau visé : « sûr et solide, pas le Pentagone » — fermer les portes réellement ouvertes, un
garde-rail par classe de défaut, sans chantier disproportionné ni dépendance lourde.

Campagne de 9 audits périmètre × axe, un sous-agent aveugle par axe, lancés en parallèle :

| Axe | Périmètre | Modèle |
|---|---|---|
| S1 Classification des routes | `internal/api/` (huma_routes, server*, wire, handlers, require_*) | Opus |
| S2 CSRF, Origin, CORS, SameSite, ambiguïté de chemins | middleware csrf/cors/loopback/demo_read_only/request_scheme/session, nginx | Opus |
| S3 Contrôle d'accès (propriété, admin, groupes, IDOR) | require_player_ownership, authz, playerdirectory, groupstore, routes à identifiant | Opus |
| S4 Service de fichiers et chemins | static_assets, spa_handler, commendation, media_serve, rejeux, uploads, PathResolver | Opus |
| S5 Auth et session | platform/session, userstore, handlers d'auth, SSO, device-flow | Sonnet |
| S6 Abus de ressources | rate_limit, read_budget, http.Server, nginx, déclencheurs coûteux | Sonnet |
| S7 Fuites d'information | security_headers, erreurs, health, expvar, journaux | Sonnet |
| S8 Infra et dépendances | nginx, Dockerfile, compose, workflows, govulncheck, npm audit, secrets | Sonnet |
| S9 Démo publique | demo_read_only, config_demo, seed-demo, demo.conf, restes du lot recos-d | Sonnet |

Vérification adverse (§6 du skill) : chaque P0/P1 soumis à un sous-agent Sonnet frais chargé de le
réfuter (« en cas de doute, réfuté ») ; 20 vérificateurs, plus un arbitrage sur un désaccord entre
vérificateurs (taille des corps `RawBody`). Lecture seule partout ; seul S8 a exécuté de l'outillage
Go (govulncheck, PoC hors dépôt) avec un cache dédié.

Axes passés : S1 à S9. Axes non passés : aucun (les axes du catalogue A1-A10 du skill sont des
axes de qualité de code, hors objet de cette campagne).

Doctrine de référence (collée dans chaque prompt) :
- **D1 Classification** : toute écriture (ou GET à effet de bord) exige le bon contrôle
  (authentifié / propriétaire / admin) ; en démo toute écriture est refusée.
- **D2 CSRF** : contrôle d'Origin sur toute écriture à cookie ; exemption décidée sur le chemin
  effectivement routé, jamais sur un chemin ambigu.
- **D3 Accès** (ADR 0029, ADR 0035) : propriété par xuid lié ; couches A et B ; verrou d'instance
  et défauts sûrs. **Décision utilisateur du 2026-10-15** : « un user peut consulter les datas de
  ses amis mais pas supprimer ou modifier » — l'accès famille/groupe est en lecture seule.
- **D4 Fichiers** : servir sous la racine déclarée seulement ; jamais une base, un jeton, une config.
- **D5 Auth/session** : hachage lent, tentatives limitées, `state` OAuth, cookies durcis, rotation
  d'identifiant à la connexion, invalidation au logout/changement de rôle, jetons jamais exposés
  (ADR 0023).
- **D6 Ressources** : corps bornés, timeouts, opérations coûteuses bornées, limites de débit à clé
  = vraie IP cliente.
- **D7 Fuites** : aucun détail interne à un non-admin, aucun secret en journal, en-têtes posés,
  `/debug/vars` derrière admin (ADR 0009).
- **D8 Infra** : seul nginx exposé, privilèges minimaux, aucun secret versionné, aucune
  vulnérabilité atteinte.
- **D9 Démo** : lecture seule stricte, rien hors fixture, aucune fonction d'administration.

Gravité : **P0** = exploitable par un visiteur anonyme ou un compte ordinaire (écriture, lecture
de données d'autrui, fichier arbitraire, déni de service à coût nul) ; **P1** = exploitable sous
condition ou défense en profondeur manquante ; **P2** = durcissement.

Dette assumée, non remontée : pas de rotation des secrets (décision du 2026-10-14) ; faille
`/static/commendations/` corrigée (PR #89) ; baseline lint gelée ; pas de WAF, pas
d'authentification forte, pas de refonte du modèle de session.

Synthèse : **12 P0, 17 P1, 36 P2 retenus** ; 17 constats écartés ou déclassés en vérification
adverse. Les constats sont regroupés par **classe de défaut** (C1 à C10) : c'est l'unité de lot du
plan de correction. Les identifiants `Sx-NN` renvoient aux rapports d'auditeurs (plusieurs
auditeurs ont trouvé le même défaut indépendamment : c'est noté).

---

## Constats retenus

### C1 — IP cliente falsifiable derrière nginx

#### [P0] C1-1 L'en-tête `True-Client-IP` remplace l'IP cliente : la limite de débit ne limite rien
- Trouvé indépendamment par S2-01, S5-01, S6-01, S8-01 (S8 : preuve exécutée hors dépôt).
- Où : `apps/go-api/internal/api/server.go:625-626` (`chimiddleware.RealIP`, monté si
  `TrustProxyHeaders`) ; `internal/api/middleware/rate_limit.go:66-67` (`httprate.KeyByIP` =
  `RemoteAddr` réécrit) ; `packaging/nginx/levelup.conf:59-63` et `demo.conf:50-53` (seuls
  `X-Real-IP`, `X-Forwarded-For`, `X-Forwarded-Proto` posés). chi v5.3.2 `middleware/realip.go:42`
  lit `True-Client-IP` en premier (fonction marquée « Deprecated: vulnerable to IP spoofing »).
- Règle : D6 (clé = vraie IP, non forgeable).
- Conséquence : un anonyme change d'IP à chaque requête → aucune limite. `POST /api/v1/auth/login`
  n'a pas d'autre frein (aucun compteur par compte, `handlers/user_auth.go:126`) : force brute
  illimitée et saturation CPU (bcrypt coût 12, y compris pour un compte inconnu,
  `platform/userstore/store.go:181`). Les journaux `ip=` sont falsifiables. Lève aussi la borne de
  C3. Précondition : `LEVELUP_TRUST_PROXY_HEADERS=true` en prod (commentaire `levelup.conf:57`,
  HSTS émis par Go ; à confirmer dans le `.env.local` du VPS).
- Reproduction : `grep -n "RealIP\|KeyByIP" apps/go-api/internal/api/server.go apps/go-api/internal/api/middleware/rate_limit.go; grep -c True-Client-IP packaging/nginx/*.conf`
- Vérification adverse : **tient** (P0).
- Traitement proposé : remplacer `RealIP` par `ClientIPFromHeader("X-Real-IP")` (nginx écrase cet
  en-tête), clé httprate sur l'IP ainsi obtenue ; nginx `proxy_set_header True-Client-IP ""` ;
  compteur d'échecs par compte sur `/auth/login` et `/auth/password`. Garde-rail : archlint
  interdisant `chimiddleware.RealIP`, `httprate.KeyByIP` et toute lecture de `r.RemoteAddr` hors
  `middleware/loopback.go` ; test table-driven sur la pile transverse (en-têtes forgés → clé
  inchangée) ; test de configuration nginx. | Décision : à planifier.

#### [P1] C1-2 Garde « rejeu servi en local seulement » ouverte par le même en-tête
- Trouvé par S2-02, S4-05, S6-01, S8-01.
- Où : `internal/api/handlers/replay_local_gate.go:86-100` (`isLocalRequest` lit `r.RemoteAddr`
  réécrit par RealIP), montage `server_apiv1.go:789-798`. `LoopbackOnly` lit bien l'adresse
  préservée (`middleware/loopback.go:45,58`). Le test `replay_local_gate_test.go:30-39` appelle la
  garde sans RealIP : fausse assurance.
- Règle : D6 / D8 (IP de confiance falsifiable) ; décision du 2026-07-28 (rejeu local seulement).
- Conséquence : un compte ordinaire envoie `True-Client-IP: 127.0.0.1` et obtient le rejeu 2D et
  la séquence Tactique en production, y compris pour un match auquel son joueur n'a pas participé
  (`GetReplay` ne dépend pas du joueur, `service/replay_service.go:110,150`).
- Reproduction : `grep -n "r.RemoteAddr" apps/go-api/internal/api/handlers/replay_local_gate.go`
- Vérification adverse : **tient** (P1 ; P0 si des artefacts sont en prod et les ids découvrables).
- Traitement proposé : la garde lit l'adresse préservée par `PreserveRemoteAddr` ; couvert par le
  garde-rail de C1-1. | Décision : à planifier.

### C2 — Entrées non bornées (taille des corps, paramètres, écriture de session)

#### [P0] C2-1 Les corps `RawBody` ne sont bornés nulle part : épuisement mémoire anonyme
- Trouvé en vérification de S6-12, confirmé par un arbitrage dédié.
- Où : Huma v2.39.1 `huma.go:1454-1460` (le défaut de 1 Mio n'est posé que pour un champ `Body`),
  `:1467-1480` (branche `RawBody` sans plafond), `:1118` / `:2172-2175` (`io.Copy` vers un
  `bytes.Buffer`). Le commentaire `internal/api/humacore/operation.go:70` (« Huma applique un défaut
  de 1 Mio ») est faux pour ces opérations. 42 types d'entrée `RawBody`, 41 sans `MaxBody`, 62
  handlers. nginx `levelup.conf:85-86` : `client_max_body_size 2g` sur tout `/api/`.
- Règle : D6 (tout corps borné).
- Conséquence : un anonyme envoie quelques corps de ~2 Gio à `POST /api/v1/auth/login`,
  `/auth/register`, `/session/context` ou `/auth/password` (test de session fait après lecture,
  `user_auth.go:276-280`) : lecture intégrale en RAM (pic 1,5 à 3 fois la taille) sur un VPS de
  1,8 Go → processus tué. La démo est bornée par le défaut nginx (1 Mio).
- Reproduction : `grep -rn "RawBody" apps/go-api/internal --include=*.go | grep -v _test | wc -l; grep -rn "MaxBody" apps/go-api/internal --include=*.go | grep -v _test`
- Vérification adverse : **tient** (P0), après arbitrage contre une première vérification qui
  supposait le défaut Huma actif.
- Traitement proposé : plafond par défaut appliqué à toutes les opérations (wrapper `humacore`),
  dérogations explicites (dépôt d'artefact ouvrier) ; corriger le commentaire. Garde-rail : test
  qui parcourt toutes les opérations enregistrées et exige `MaxBodyBytes > 0`. | Décision : à planifier.

#### [P0] C2-2 `POST /session/context` : écriture disque anonyme, sans validation ni plafond
- Trouvé par S5-02, S6-07, S9-01.
- Où : `internal/api/handlers/session_context.go:79-98` (`Locale`, `PlayerSlug` non validés,
  `Save` inconditionnel) ; `platform/session/store.go:192` (aucun plafond) ; exemption démo
  `middleware/demo_read_only.go:39-41` (ratchet `demo_read_only_ratchet_test.go:73`).
- Règle : D1, D6, D9 (seule écriture autorisée en démo).
- Conséquence : sans cookie, chaque requête crée un fichier de session neuf (TTL 7 jours, purge
  toutes les 6 h). `json.Marshal` échappe `<` en `<` : ×6 d'amplification. Avec C2-1 le corps
  n'est même pas borné en prod. Disque du VPS partagé prod/démo (démo : ~3 Go/min au débit
  permis).
- Reproduction : `sed -n 78,99p apps/go-api/internal/api/handlers/session_context.go`
- Vérification adverse : **tient** (P0).
- Traitement proposé : `locale` ∈ {fr, en}, `player_slug` borné et accessible (voir C4-1), taille
  de session plafonnée dans `Save`, sessions anonymes de démo non persistées. Garde-rail : test de
  taille maximale d'une session sérialisée ; le ratchet démo exige un test « corps de 100 Ko
  refusé » pour chaque écriture laissée passer. | Décision : à planifier.

#### [P0] C2-3 `leaderboard?limit=` non borné : arrêt fatal du processus
- Trouvé par S6-08.
- Où : `internal/api/handlers/leaderboard.go:54-60,83` (`Limit` en chaîne pour éviter le 422 Huma,
  `Atoi`) ; `service/leaderboard_service.go:44` (seul `<= 0` corrigé) ;
  `platform/duckdb/leaderboard_world_repo.go:105,634` (`make([]LeaderboardEntry, 0, limit)`).
- Règle : D6 (pagination bornée côté serveur).
- Conséquence : `GET /api/v1/players/<slug>/pages/leaderboard?category=kills&limit=1000000000`
  → allocation de ~408 Go → `fatal error: out of memory`, non rattrapé par `Recoverer`. Accessible
  à tout propriétaire de profil, et à **tout visiteur de la démo** (GET, propriété ouverte en démo).
- Reproduction : `sed -n 100,106p apps/go-api/internal/platform/duckdb/leaderboard_world_repo.go`
- Vérification adverse : **tient** (P0 ; dépend de `vm.overcommit_memory`, défaut Linux = fatal).
- Traitement proposé : plafond serveur (ex. 200) dans le service + `maximum` déclaré. Garde-rail :
  test OpenAPI exigeant un `maximum` sur tout paramètre `limit`/`n`/`count`/`page_size`, et test
  grep interdisant un `make(..., limit)` sur une valeur non plafonnée. | Décision : à planifier.

#### [P1] C2-4 nginx : 2 Go acceptés sur tout `/api/`, aucun `limit_req` en production
- Trouvé par S6-10, S8-02.
- Où : `packaging/nginx/levelup.conf:85-90` (`client_max_body_size 2g`, timeouts 3600 s, mise en
  tampon disque par défaut) ; seul `limit_req` du dépôt : `demo.conf:88`.
- Règle : D6, D8.
- Conséquence : nginx écrit tout corps jusqu'à 2 Go sur le disque avant de contacter Go ; une
  vingtaine d'envois parallèles remplissent le disque. Les seules routes légitimes à gros corps
  sont authentifiées et plafonnées côté Go (média 500 Mo, import 1 Gio).
- Reproduction : `grep -n -E "client_max_body_size|limit_req|limit_conn" packaging/nginx/levelup.conf`
- Vérification adverse : **tient** (P1 haut ; la config nginx réelle du VPS, retouchée par
  certbot, reste à confirmer).
- Traitement proposé : `client_max_body_size 1m` sur `/api/`, deux `location` exactes pour
  l'envoi de médias et l'import, `limit_req_zone` + `limit_conn`. Garde-rail : test qui parse les
  `.conf` versionnés. | Décision : à planifier ; appliquer sur le VPS = opération à annoncer.

### C3 — Travaux coûteux déclenchables sans borne

#### [P0] C3-1 Device-flow public : chaque requête anonyme lance ~15 min d'appels vers Microsoft
- Trouvé par S5-03, S6-02.
- Où : `internal/api/handlers/auth.go:107-163,276` (`go h.pollDeviceFlow(context.Background())`) ;
  `platform/auth/attempt_store.go:128-150` (une tentative par session, sessions gratuites) ;
  montage sans condition `server_apiv1.go:376`.
- Règle : D1, D6.
- Conséquence : `POST /api/v1/auth/device-flow/start` sans cookie, en boucle → milliers de
  goroutines et centaines d'appels/s vers login.live.com avec l'AppID Xbox partagé par les
  rafraîchissements de jetons de la prod. Refusé en démo (garde générale).
- Reproduction : `sed -n 107,165p apps/go-api/internal/api/handlers/auth.go`
- Vérification adverse : **tient** en prod (P0 ; P1 si Microsoft ne limite jamais), réfuté en démo.
- Traitement proposé : plafond global de tentatives en attente + limiteur dédié par IP (429).
  Garde-rail : voir C3-7. | Décision : à planifier.

#### [P0] C3-2 `/api/v1/assets/*` : appels sortants signés et écritures anonymes, hors limiteur
- Trouvé par S1-02, S6-03 ; aggravé en vérification.
- Où : `internal/api/server_apiv1.go:653-657` (routes nues) ; `middleware/rate_limit.go:94`
  (préfixe exempté) ; `internal/assets/resolver_default.go:92-94` (pas de cache négatif) ;
  `fetcher_chain.go:78` (un 404 devient un succès « spritesheet ») ; `store_duckdb.go:118`
  (INSERT dans `asset_index`) ; `server_apiv1.go:1390-1397` (`AnyPlayerTokens`) ;
  `platform/halo/provider.go:167` (limiteur sortant recréé à chaque requête).
- Règle : D1 (GET à effet de bord), D6.
- Conséquence : un anonyme boucle sur des identifiants inventés : appels vers les API Halo avec le
  jeton d'un vrai joueur (risque de throttling de ce compte) et, via les médailles, lignes ajoutées
  sans fin dans `metadata.duckdb`. Coupé en démo par `netguard`.
- Reproduction : `sed -n 78,96p apps/go-api/internal/assets/fetcher_chain.go; grep -n assets apps/go-api/internal/api/middleware/rate_limit.go`
- Vérification adverse : **tient** (P0 sous réserve : la réponse du CDN Halo à un id inventé n'est
  pas vérifiable hors ligne).
- Traitement proposé : valider les identifiants contre le catalogue local avant tout fetch, cache
  négatif, budget sortant global, anonymes servis en cache seul, retrait de l'exemption hors hits
  locaux. Garde-rail : voir C3-7. | Décision : à planifier.

#### [P0] C3-3 Exclusion de match : recalcul complet sous le writer partagé, en boucle
- Trouvé par S6-04.
- Où : `server_apiv1.go:929-930` ; `handlers/match_exclusion.go:69-78` ;
  `service/match_exclusion_service.go:51-64` (aucun no-op si l'état ne change pas) ;
  `sync/match_recomputer.go:79-109` (`AcquireSharedWriterStandalone`, `force=true` sur tout
  l'historique).
- Règle : D6.
- Conséquence : un compte propriétaire alterne `{"excluded":true}` / `false` : chaque appel bascule
  la base partagée en écriture (lecteurs de tout le site → 503 après 3 s) et recalcule 2 à 10 s
  d'historique ; la base du joueur grossit de N lignes par appel. La bascule est **inutile** : les
  écritures vont en base joueur (`skill_rating_postsync_persist.go:127`), la base partagée n'est
  que lue.
- Reproduction : `sed -n 74,110p apps/go-api/internal/sync/match_recomputer.go`
- Vérification adverse : **tient** (P0).
- Traitement proposé : ne pas prendre le writer partagé ; no-op si l'état est inchangé ;
  single-flight + délai minimal par joueur. Garde-rail : test interdisant `AcquireWriter` /
  `AcquireSharedWriterStandalone` dans un chemin déclenché par une route `/players` hors liste
  datée. | Décision : à planifier.

#### [P0] C3-4 Import OpenSpartan en boucle : writer partagé tenu pendant tout l'import
- Trouvé par S6-05.
- Où : `server_apiv1.go:614,644-645` (RequireAuth seul) ; `handlers/openspartan_import.go:116-180` ;
  `service/openspartan_import_service.go:146,159,181-194` (`AcquireWriter` avant
  `openspartan.Open` et `validateOwner`, aucun « un import à la fois », goroutine
  `context.Background()`).
- Règle : D6.
- Conséquence : un compte lié soumet en boucle un fichier valide → base partagée en écriture
  pendant chaque import (503 pour tous) ; un fichier invalide provoque quand même une bascule.
- Reproduction : `sed -n 176,196p apps/go-api/internal/service/openspartan_import_service.go`
- Vérification adverse : **tient** (P0 fichier valide ; P1 fichier invalide).
- Traitement proposé : valider avant la bascule, un import à la fois par compte (`FindActiveJob`
  comme `backfill.go:111`), délai minimal. Lié à C4-3 (décision sur le périmètre de l'import).
  | Décision : à planifier.

#### [P1] C3-5 Envoi de médias : mémoire et ffmpeg sans plafond global
- Trouvé par S6-06.
- Où : `handlers/media_upload.go:35-36,135` (`io.ReadAll` jusqu'à 500 Mo) ;
  `service/media_service_upload.go:166,182` (`go s.runTranscodeJob`, `context.Background()`) ;
  `media/hls.go:539` ; aucun sémaphore.
- Règle : D6.
- Conséquence : un propriétaire envoie quatre lots de 500 Mo en parallèle → OOM du VPS. Refusé en
  démo.
- Vérification adverse : **tient partiellement** (P0 réfuté : corps borné à 500 Mo et 50 fichiers ;
  P1 retenu).
- Traitement proposé : stockage en flux, sémaphore global d'envois et de ffmpeg. Garde-rail : grep
  interdisant `io.ReadAll` sur un `multipart.File` ; sémaphore exigé avant tout ffmpeg.
  | Décision : à planifier.

#### [P2] C3-6 Recherche de gamertags `live=1` anonyme et `recompute_coefficients` non gardé
- `server_apiv1.go:1084`, `service/gamertag_search_live.go:42,80-130` (appel PeopleHub avec jeton
  réel, limité par IP et cache négatif de 60 s, aucune écriture) ;
  `handlers/engagement.go:65,326` (route libellée « admin », sans garde admin, impact borné).
- Traitement : `live` réservé aux sessions authentifiées ; garde admin sur `recompute_coefficients`.

#### Garde-rail de classe C3-7
`internal/api/bare_routes_ratchet_test.go` ne regarde que la présence d'un middleware, pas les
effets de bord (`:79,85-92`). Proposition : chaque entrée de l'allowlist publique porte une classe
de coût (`cheap` / `outbound` / `disk`), et `outbound` / `disk` exigent un limiteur dédié
détectable ; test « requête anonyme avec fetcher et store espions → aucun appel sortant ni
écriture ».

### C4 — Identité du demandeur et écritures sur les données d'autrui

#### [P0] C4-1 Le « joueur courant » de la session se forge : suppression et likes des médias d'autrui
- Trouvé indépendamment par S1-01 et S3-01.
- Où : `handlers/session_context.go:79-81` (slug accepté sans contrôle) ;
  `middleware/session.go:94-99` (`SessionPlayerSlug` lit le champ persisté) ;
  `handlers/media_delete.go:86` ; `domain/media_delete.go:132-135` ;
  `platform/duckdb/media_repo_delete.go:61-66` ; `handlers/media_likes.go:104-107`.
- Règle : D3 (utilisateur courant résolu par xuid), D1.
- Conséquence : un compte ordinaire envoie `POST /api/v1/session/context
  {"player_slug":"<victime>"}`, puis `DELETE /api/v1/players/<le sien>/media?file_path=<média de
  la victime>` (chemin visible dans sa galerie, `queries_citations.go:177-187`) : le média est
  masqué et ses fichiers supprimés du disque, sans retour. Même levier pour liker au nom d'un tiers
  (notification envoyée en son nom).
- Reproduction : `grep -rn "CurrentPlayerSlug = req\|RequesterSlug = middleware.SessionPlayerSlug\|EqualFold(req.RequesterSlug" apps/go-api/internal`
- Vérification adverse : **tient** (P0).
- Traitement proposé : dériver le demandeur et le liker du profil possédé par
  `authz.CurrentUser` (xuid) ; refuser dans `/session/context` un slug non accessible (modèle :
  `squadActorGuard`, `server_apiv1.go:1025-1034`). Garde-rail : archlint interdisant
  `SessionPlayerSlug(` dans toute décision d'autorisation (allowlist vide). | Décision : à planifier.

#### [P0] C4-2 Association média → match sans contrôle de propriétaire
- Trouvé par S3-02.
- Où : `handlers/media.go:342-362` ; `service/media_service.go:205-222` ;
  `platform/duckdb/media_repo_writes.go:28-31` (`WHERE (file_path = ? OR file_name = ?) LIMIT 1`,
  sans filtre joueur ni ordre, table `shared_social`).
- Règle : D1, D3 (décision du 2026-10-15 : un ami ne modifie pas).
- Conséquence : `POST /api/v1/players/<le sien>/media/associate {"file_path":"<clip d'autrui>",
  "match_id":"<autre>"}` : le clip disparaît de son vrai match et s'affiche sur le match choisi,
  pour tout le monde (l'association manuelle prime dans la vue `_latest`). Un nom de fichier
  homonyme peut aussi toucher le média d'un autre par accident.
- Vérification adverse : **tient** (P0).
- Traitement proposé : charger le propriétaire et appliquer la règle de `CanDeleteMedia` ;
  recherche par chemin complet seulement. Garde-rail : test table-driven « média d'autrui » ×
  {associer, supprimer, liker} → 403. | Décision : à planifier.

#### [P0] C4-3 Import OpenSpartan : écriture des données d'autres joueurs dans la base partagée
- Trouvé par S3-03.
- Où : `service/openspartan_import_service.go:236-255,274-285,426` ; `openspartan/owner.go:90-101`
  (propriétaire = xuid le plus fréquent du fichier) ; `sync/writes.go:43-48` (`UPDATE xuid_aliases`
  inconditionnel) ; `openspartan/participants.go:38-45`.
- Règle : D1, D3.
- Conséquence : un compte lié forge un `.db` où son xuid domine : il renomme un autre joueur
  partout (`xuid_aliases` prime dans `v_gamertag_lookup`, et le sync normal ne rétablit pas le
  nom), injecte des matchs inventés dans l'historique d'un tiers, pré-empte un vrai match pas encore
  synchronisé, et peut inscrire des identifiants de match piégés qui font écrire des fichiers
  parasites au rattrapage (`replayartifacts/raster.go:339`, `derivations.go:310`).
- Vérification adverse : **tient** (P0).
- Traitement proposé, **décision utilisateur requise** : (a) réserver l'import à l'admin, ou
  (b) n'écrire que les matchs où le propriétaire participe, aucun alias d'un autre xuid, format de
  `match_id` validé. Garde-rail : fixture contenant un xuid tiers → zéro ligne tierce écrite.

#### [P1] C4-4 Les membres d'un groupe écrivent sur le profil d'un autre membre
- Trouvé par S1-03 et S3-06 ; **défaut confirmé par la décision utilisateur du 2026-10-15**.
- Où : `internal/api/server_apiv1.go:146-147` (résolveur famille dans la garde de propriété) ;
  `internal/authz/authz.go:54-72` (aucune distinction lecture/écriture) ;
  `handlers/title_sync.go:131-133,190-198` (purge d'un titre). L'ADR 0029 (l.124) ne pose
  « propriétaire direct ou admin » que pour les amis.
- Règle : D3.
- Conséquence : `DELETE /api/v1/profiles/<membre>/titles/halo_infinite/data` supprime la base
  joueur d'un co-membre (si un autre titre est actif) ; idem sync, notifications, exclusions,
  Prestige. Tout membre peut inviter (`groups.go:209`) : la confiance est transitive.
- Vérification adverse : **tient** (P1 : co-appartenance requise).
- Traitement proposé : la garde ne transmet l'ensemble famille que si `!IsMutatingRequest(r)` ;
  amender l'ADR 0029. Garde-rail : test « co-membre × toutes les routes mutantes du routeur → 403 ».
  | Décision : tranchée (2026-10-15), à planifier.

#### [P1] C4-5 Provisionnement : un `profile_mode` autre que `xbox` écrase le xuid d'un profil existant
- Trouvé par S3-10 ; aggravé en vérification.
- Où : `handlers/setup.go:193-200,300` (`guardLinkedXboxIdentity` ignorée hors `xbox`) ;
  `service/profile_service.go:59-86` (crée ou met à jour, `entry.XUID = req.XUID` en `:82`) ;
  `service/playerdirectory/onboard.go:73-78`.
- Règle : D3, ADR 0035.
- Conséquence : `POST /api/v1/setup/players {"gamertag":"<autrui>","xuid":"<le sien>",
  "profile_mode":"manual"}` : l'attaquant devient propriétaire du profil d'autrui, qui perd
  l'accès. Précondition : auto-provisionnement ouvert et instance non verrouillée (défauts hors mode
  appliqué ; état de la prod le 2026-07-23 selon l'ADR 0035).
- Vérification adverse : **tient** (P1 ; P0 sur une instance ouverte).
- Traitement proposé : tout non-admin épinglé à son identité liée quel que soit le mode ; jamais de
  changement de xuid d'un profil existant hors admin. Garde-rail : test table-driven modes ×
  non-admin → refus. | Décision : à planifier.

#### [P1] C4-6 Un non-membre peut créer un défi dans une escouade
- Trouvé par S3-09.
- Où : `prestige/service_squad_challenges.go:28-66` (pas d'`assertMemberUser`, présent en
  `:121,162,214`) ; commentaire faux `api/prestige_player_slug_mw.go:26-29`.
- Règle : D3.
- Conséquence : un ex-membre connaissant l'identifiant inscrit un défi parasite dans l'escouade.
- Vérification adverse : **tient** (P1 limite P2).
- Traitement proposé : `assertMemberUser` à la création ; corriger le commentaire. Garde-rail :
  test AST « toute méthode du service qui prend un `squadID` appelle `assertMemberUser` ».

#### [P1] C4-7 `/healthz/home?player=` lit le profil d'autrui et écrit dans sa base
- Trouvé par S1-04, S3-04, S7-08.
- Où : `server_apiv1.go:191-196` (RequireAuth seul ; commentaire « propriété implicite » faux) ;
  `handlers/health_home.go:57-60,82-102` ; `wire/registry_auth.go:63,164,346,426`.
- Règle : D3, D1, D7.
- Conséquence : un compte ordinaire obtient un oracle d'existence (contourne le 403 uniforme),
  l'état des sections d'autrui et des erreurs brutes ; la requête rafraîchit la carrière d'autrui
  avec son jeton et écrit dans sa base.
- Vérification adverse : **tient** (P1).
- Traitement proposé : `RequireAdmin` (sonde opérateur). Garde-rail : ratchet « tout paramètre
  `player`/`player_slug`/`xuid`/`gamertag` hors `/players/{player_slug}` exige propriété ou admin ».

#### [P2] C4-8 Couche B absente sur positions, objectifs et événements d'un match
- S3-05 déclassé : `/matches/{id}` répond 404 à un non-participant mais `/positions`,
  `/objective-events`, `/events` répondent 200 (`service/match_view_service.go:478-494`,
  `match_events_service.go:47`). Données de match partagées (ADR 0029, l.88-89), identifiant non
  découvrable. Traitement : cohérence 404, avec C1-2 pour le rejeu.

### C5 — Session et authentification

#### [P0] C5-1 Le verrou d'instance se contourne par la connexion Xbox par code
- Trouvé par S5-04.
- Où : `handlers/auth.go:223-245` (`AuthReady`, identité et jetons posés avant
  `OnAuthSuccess`, erreur seulement journalisée, session sauvée) ; `middleware/require_auth.go:77`
  (`AuthReady` suffit hors mode password) ; `authz/authz.go:89-97` (utilisateur synthétisé) ;
  `handlers/groups.go:153,223,286` (groupe et invitation sans compte existant) ;
  `service/xbox_auth_service.go:170` et `resolvePendingInvite`.
- Règle : D3 (ADR 0035 : verrou, défauts sûrs).
- Conséquence : n'importe quel compte Microsoft, instance verrouillée par défaut : la session reste
  authentifiée malgré le refus ; l'inconnu crée un groupe, émet une invitation et la consomme
  lui-même → compte créé et droit de provisionnement malgré le verrou. Le flux OAuth fait bien
  l'inverse (`auth_xbox_oauth.go:349-365`). Aucun test.
- Vérification adverse : **tient** (P0).
- Traitement proposé : n'écrire `AuthReady`, l'identité et les jetons qu'après le succès de
  `OnAuthSuccess` ; une invitation n'est valable que si son émetteur est un compte existant.
  Garde-rail : test « refus pour verrou en device-flow → session non authentifiée ».

#### [P1] C5-2 La déconnexion ne déconnecte pas en mode Xbox
- Trouvé par S2-04, S3-08, S5-05.
- Où : `handlers/user_auth.go:256-271` (seuls `Username` et `Role` effacés) ;
  `middleware/require_auth.go:71-77` ; `authz/authz.go:89-92`.
- Règle : D5.
- Conséquence : après « Se déconnecter », le même cookie lit et écrit toujours le profil (poste
  partagé, cookie volé) ; un admin SSO garde l'accès à tous les joueurs. Aucun test du logout.
- Vérification adverse : **tient** (P1).
- Traitement proposé : supprimer la session côté serveur et poser un cookie expiré. Garde-rail :
  test « connexion SSO → déconnexion → lecture refusée ».

#### [P1] C5-3 `RequireAdmin` se fie au rôle figé dans la session
- Trouvé par S3-07, S5-05.
- Où : `middleware/require_admin.go:16-23` (pas de magasin d'utilisateurs) ;
  `handlers/admin.go:118,132,142,155` ; `handlers/settings.go:216` ; `handlers/media_delete.go:88`.
- Règle : D5.
- Conséquence : un admin supprimé ou rétrogradé garde `/admin/*`, `/settings`, `/sync/all` tant
  qu'il réutilise sa session (TTL glissant de 7 jours).
- Vérification adverse : **tient** pour les routes admin (P1), réfuté pour la lecture des joueurs.
- Traitement proposé : `RequireAdmin` relit le rôle via `authz.CurrentUser` (comme
  `setup.go:329-345`). Garde-rail : test « rétrograder puis rejouer la session → 403 ».

#### [P1] C5-4 Identifiant de session jamais renouvelé à la connexion (fixation)
- Trouvé par S2-03 (S5-08 en P2).
- Où : `platform/session/store.go:145` (seul `Store.New` crée un identifiant) ;
  `handlers/user_auth.go:148-157`, `service/xbox_auth_service.go:195-198`,
  `handlers/auth_xbox_oauth.go:368-385`, `handlers/auth.go:222-245`.
- Règle : D5.
- Conséquence : un cookie planté avant la connexion reste valide après. Implantation difficile
  aujourd'hui (pas de XSS connu sur la démo, HSTS `includeSubDomains`), d'où P1.
- Vérification adverse : **tient** (P1).
- Traitement proposé : nouvel identifiant à chaque connexion, ancienne session supprimée.
  Garde-rail : test sur les trois chemins de connexion « la valeur du cookie change ».

#### [P1] C5-5 `LEVELUP_AUTH_MODE` sensible à la casse : `Xbox` désactive la propriété sans avertir
- Relevé en vérification de S5-07.
- Où : `internal/config/config.go:243` (aucune normalisation), `:323` (`EqualFold` seulement
  pour `none`), `Validate` (`:~350`) ; comparaisons exactes `authz/authz.go:37`,
  `server_apiv1.go:338,382`, `middleware/require_admin.go:18`, `require_auth.go:35`.
- Règle : D3 (défauts sûrs).
- Conséquence : avec `Xbox`, `/players/{slug}` devient lisible et modifiable par un anonyme, sans
  message au démarrage. Condition : faute de casse de l'opérateur.
- Vérification adverse : **tient** (P1).
- Traitement proposé : normaliser à la lecture et refuser au démarrage toute valeur inconnue
  (`AUTH_MODE`, `LEVELUP_REGISTRATION`, dont la casse ouvre aussi l'inscription,
  `user_auth.go:203-206`). Garde-rail : test de configuration table-driven.

#### [P2] C5-6 Hygiène d'authentification
- Le callback SSO enregistre le jeton de rafraîchissement d'un compte inconnu avant le contrôle du
  verrou (`handlers/auth_xbox_oauth.go:209` avant `:213`) : un fichier de jetons par inconnu
  (ADR 0023).
- Invitation : `Validate` puis `Consume` non atomiques (`user_auth.go:210,230`) : un code peut
  produire plusieurs comptes.
- Changement de mot de passe sans l'ancien pour un compte qui en a déjà un (`user_auth.go:278-287`).
- Premier compte admin : course entre inscriptions simultanées (`userstore/store.go:114-150`) et
  aucune garde « dernier admin » (`store.go:314,362`, `admin.go:119,142`).
- Réglages illisibles ou mal typés → instance non verrouillée (`authz/authz.go:111-131`, choix
  documenté mais justification faible).

### C6 — Service de fichiers et chemins

#### [P1] C6-1 La route des médias sert le dossier du joueur, base DuckDB comprise
- Trouvé par S4-01.
- Où : `handlers/media_serve.go:55-56` (4e racine = `players/{slug}/`), `:77` (`http.ServeFile`) ;
  `domain/title/registry.go:735-756`.
- Règle : D4.
- Conséquence : `GET /api/v1/players/<slug>/media/files/stats.duckdb` télécharge la base joueur
  complète. Gain réel pour un co-membre de groupe (base brute d'autrui).
- Vérification adverse : **tient** (P1).
- Traitement proposé : restreindre la racine legacy à `captures/` + liste blanche d'extensions
  média. Garde-rail : test table-driven `.duckdb`, `.wal`, `.json` → 404 sur chaque racine.

#### [P2] C6-2 Slug et identifiant de match transformés en chemins sans validation
- S4-02, S9-06 : `media_serve.go:47,55` (slug `..` ou `.` en démo ou en dev → `stats.duckdb` de la
  fixture ou du dépôt) ; S4-04 : `domain/title/film_id.go:28-36` (`FilmShortMatchID` sans contrôle
  de caractères), `service/replay_service.go:66`, `registry_film_facts.go:77`,
  `registry_tactical.go:53` (lecture de `*.json` arbitraire : oracle d'existence ; les 5xx sont
  masqués). Traitement : validation fail-closed (hexadécimal et tirets) dans
  `FilmShortMatchID`/`PathResolver`, slug validé dans la garde `/players/{player_slug}`. Garde-rail :
  test « le chemin reste sous sa racine » sur tous les constructeurs de chemin.

#### [P2] C6-3 Listing de `/static/` ; contenu des fichiers envoyés non vérifié
- `server_apiv1.go:1500` (`http.FileServer` liste les répertoires, contenu public) ;
  `handlers/media_upload.go:126-127` (extension seule ; S4-03 déclassé, voir écartés).

### C7 — Fuites d'information (toutes déclassées en P2 après vérification)

#### [P2] C7-1 Classe « `err.Error()` renvoyé au client » sans garde-rail
- S7-07, S9-07 : 121 réponses 4xx recopient `err.Error()` (`humacore.NewError` ne masque que ≥ 500 ;
  ex. `handlers/achievements.go:56`, `filters.go:58-61`, `config/player_resolver.go:179`) ;
  `/readyz` (`handlers/health.go:153`) ; erreurs de sync dans les jobs (`sync_handler.go:50-54,591-593`,
  `platform/jobs/store.go:111-115`) ; `error_detail` du device-flow (`auth.go:284,299`).
  Traitement : messages fixes par code, détail au journal. Garde-rail : ratchet grep interdisant
  `err.Error()` en argument de `NewError`/`writeError`, allowlist datée.

#### [P2] C7-2 En-têtes et empreinte
- Aucune CSP (`middleware/security_headers.go:10-14`, dette P2 datée du 2026-06-02, aucun puits
  XSS exploitable trouvé) ; `/health` public expose `go_version`, `db_version`, versions ffmpeg,
  compteurs (`handlers/health.go:112-125`) alors que seules des sondes locales l'utilisent ;
  `server_tokens off` absent des confs nginx versionnées. Traitement : `/health` réduit à
  `{status}` pour un anonyme, CSP en Report-Only puis appliquée, `server_tokens off`.

#### [P2] C7-3 Journaux
- Identifiant de login journalisé en entier (`user_auth.go:129` : un mot de passe saisi dans le
  champ identifiant finit au journal) ; codes d'invitation journalisés en clair (`admin.go:216`,
  `groups.go:227`, `user_auth.go:231`). Garde-rail : ratchet grep sur ces attributs.

### C8 — Démo publique

#### [P1] C8-1 Lectures d'administration ouvertes aux visiteurs de la démo
- Trouvé par S1, S3, S7, S8-03, S9-02.
- Où : `middleware/require_admin.go:18`, `require_auth.go:34` (transparents en démo) ;
  `server_apiv1.go:429-433` (`/admin`), `:532` (`/settings`), `:1092` (`/watcher`),
  `:1537-1541` (`/debug/vars`) ; `demo.conf` ne bloque rien.
- Règle : D9, D7 (ADR 0009 : `/debug/vars` derrière admin).
- Conséquence : un anonyme lit `/debug/vars`, `/api/v1/admin/monitoring/*` (dont le lecteur de
  journaux), `/admin/users`, `/admin/identities`, `/admin/token-health`, `/watcher/status`. Contenu
  quasi vide aujourd'hui (fixture, magasins d'auth vides, lecteur de journaux pointé hors du
  conteneur) ; piège si le lecteur de journaux est rebranché sur le dossier de la démo. La
  transparence documentée (DB-28, « Non traité ») ne visait pas les lectures.
- Vérification adverse : **tient** (P1 bas).
- Traitement proposé : en démo, `RequireAdmin` et `/debug/vars` répondent 403 sauf liste nommée
  (`GET /settings`, seul lu par l'interface). Garde-rail : ratchet qui parcourt le routeur de démo
  et exige 403 sur toute route portant `RequireAdmin`, hors allowlist datée.

#### [P2] C8-2 Restes du lot recos-d (item [demo] du backlog)
- **Identités sans témoin disque** : `server_player_directory.go:70`, `playerdirectory/collect.go:318-323`
  → un système de fichiers démo adossé à `DemoLayout`/`DemoRoster`.
- **CLI sous `LEVELUP_DEMO_MODE=true`** : six lectures `cfg.DemoMode` au lieu de `cfg.demoState()`
  (`config/config_players.go:106`, `config/player_resolver.go:58,89,103,259,271`) ; effet réel :
  `recompute-friends` lit la base partagée de la fixture. Garde-rail : grep interdisant `.DemoMode`
  dans `internal/config/` hors `config_demo.go`.
- **`suggest-next` classé écriture** : `handlers/prestige.go:97` absent de `readOnlyPostPrefixes`
  (`middleware/require_auth_mutations.go:52-63`) → à ajouter, ratchet mis à jour.
- **E2E `media-like-bug.spec.ts`** (`:227,244`) suppose un like en démo → à retirer ou marquer hors démo.
- **`weapon_kills`** : `ops/seed_demo.go:189` — à **conditionner par titre**, pas à retirer (Halo 5
  la porte encore, `ops/snapshot_export.go:48-53`) ; `seed_demo.go:988` avale l'erreur de création de
  la vue (`_, _ =`, anti-pattern 10).
- **ERROR au boot sur la metadata Halo 5** : `platform/duckdb/db_query.go:49,136-143` journalise
  ERROR alors que les chargeurs tolèrent l'absence ; candidats `title_asset_drawer_loader.go:22,32,44,82,100`
  (`maps_catalog`, `weapon_labels`, `medal_definitions`, `team_colors`, `csr_designations`) ;
  à trancher par `grep "duckdb: query failed" data/demo/runtime/logs/*.log | grep halo_5`.
- **`data/demo_films/`** : commande au runbook (`docs/RUNBOOK_DEPLOY_CHECKLIST.md:98`), serveur
  tolérant au manque ; reste le geste VPS (à annoncer).
- **Séquence Tactique morte en démo** : `handlers/replay_local_gate.go:145` lit `match_id` dans le
  chemin alors que la route le porte dans le corps → toujours 404.
- **Démo sans confiance proxy** (S2-05, S7-02, S8-04, S9-03) : `docker-compose.yml:64-90` → un seul
  compteur de débit pour tous les visiteurs, cookie sans `Secure` ; à poser **après** C1-1.
- **Device-flow sans coupe-circuit réseau** (S9-04) : `platform/netguard/netguard_coverage_test.go:43`
  (raison « CLI uniquement » fausse) → `netguard.Check` dans `xbox_device_code.go`.

### C9 — Infra, dépendances et CI

#### [P1] C9-1 Bibliothèque standard Go vulnérable : monter en 1.26.9
- Constat connu (K2), confirmé par S8 (govulncheck exécuté).
- Où : `apps/go-api/go.mod:9` (`go 1.26.5` depuis le 2026-08-05) ; aucune branche
  `chore/deps-2026-10` n'existe (ni locale ni distante).
- Conséquence : 19 vulnérabilités atteintes ; 1.26.9 les corrige toutes (6 dès 1.26.6). Derrière
  nginx (HTTP/1.1) les vulnérabilités HTTP/2 sont inertes ; restent notamment `ParseMultipartForm`
  (routes authentifiées) et `ServeFile`/`ServeContent` (`/static`, anonyme).
- Piège : `scripts/deploy.sh:145,150` construit sans `--pull` et le `Dockerfile:51` tire
  `golang:1.26-bookworm` flottante → épingler `golang:1.26.9-bookworm` ou ajouter `--pull`.
- Traitement proposé : `go 1.26.9`, `golang.org/x/image` v0.45.0 (3 vulnérabilités atteintes via
  `mapdecoupe/masque.go:73`, images versionnées : P2), override npm `postcss-selector-parser` ^7.1.6
  (P2, build seulement).

#### [P1] C9-2 Aucun contrôle bloquant sur les vulnérabilités de dépendances
- Où : `scripts/git-hooks/lefthook/govulncheck.sh:26` (`|| true`, commentaire périmé) ; aucun
  `govulncheck` ni `npm audit` dans `.github/workflows/*.yml` ni dans `make gate-push`.
- Vérification adverse : **tient** (P1).
- Traitement proposé : job CI bloquant `govulncheck ./...` + `npm audit --omit=dev
  --audit-level=high`. Garde-rail : archlint exigeant ce job dans `ci.yml`.

#### [P1] C9-3 Actions GitHub tierces non épinglées dans les jobs qui portent les clés SSH
- Où : `.github/workflows/deploy.yml:262,285,454` (`appleboy/ssh-action@v1`, clés `VPS_SSH_KEY`,
  `VPS2_SSH_KEY`) ; `release.yml:121` (`softprops/action-gh-release@v2`, `contents: write`) ;
  aucun `environment:` protégé. Défaut connu non traité (thought_log, « D31 pinning hétérogène »).
- Conséquence : un tag amont déplacé ou compromis exfiltre la clé SSH de la prod au prochain push
  sur `main`.
- Vérification adverse : **tient** (P1).
- Traitement proposé : épinglage par SHA (Dependabot `github-actions` le maintient). Garde-rail :
  archlint refusant tout `uses:` tiers non épinglé dans un workflow qui lit `secrets.` ou a
  `contents: write`.

#### [P2] C9-4 Durcissements infra
- Protocole ouvrier `/api/v1/internal/` ouvert à tout Internet (jeton seul, sans longueur
  minimale) → `allow <IP du VPS de calcul>; deny all;` ; conteneurs sans `no-new-privileges` ni
  `cap_drop` ni limites mémoire/CPU (`docker-compose.yml`) ; unité systemd de l'ouvrier liée à un
  fichier modifiable par `deploy` (`packaging/systemd/levelup-worker.service:35,87`) ;
  `release.yml:8-9` donne `contents: write` à tout le workflow.

### C10 — Gardes qui décident sur un chemin différent du chemin routé

#### [P1] C10-1 CSRF : exemption décidée sur le chemin nettoyé (constat connu K1)
- Confirmé par S1, S2, S8 (preuve exécutée hors dépôt), S9.
- Où : `internal/api/middleware/csrf.go:68,117-128` (`isExemptPath(r.URL.Path)` après
  `path.Clean`, sans `AmbiguousPath`) ; chi route sur `RawPath` (`mux.go:454`) ; nginx transmet le
  chemin brut (`proxy_pass` sans URI, `levelup.conf:74,89,96`). La garde démo est protégée
  (`demo_read_only.go:53`).
- Règle : D2.
- Conséquence : `POST /api/v1/players/<slug>/coach/proposals/..%2F..%2F..%2F..%2Finternal%2Fx/dismiss`
  avec une Origin étrangère atteint le handler sans contrôle d'Origin. Limites : cookie
  SameSite=Lax → origine du même site exigée (ex. sous-domaine démo) ; POST simple seulement
  (préflight CORS refusé pour PATCH/DELETE). Tests existants seulement dans le sens inverse
  (`csrf_test.go:145-149`, `csrf_transverse_stack_cgo_test.go:162-166`).
- Traitement proposé : `!AmbiguousPath(r) &&` devant `isExemptPath` + tests de chemins encodés.
  Garde-rail : test table-driven passant les mêmes chemins ambigus à toutes les gardes qui décident
  sur un chemin (CSRF, démo, écritures, limiteur, rejeu).

---

## Constats écartés

| Constat | Axe | Motif d'écart |
|---|---|---|
| Fichier envoyé `.bmp` resservi en `text/html` (XSS stocké) | S4-03 | Réfuté : `.bmp` est dans la table MIME intégrée de Go (`image/bmp`), `ServeFile` ne devine jamais le type ; `.svg`/`.html` refusés à l'envoi. Reste P2 (contenu non vérifié), C6-3 |
| Positions, objectifs, événements d'un match non joué | S3-05 | Déclassé P1 → P2 : données de match partagées (ADR 0029), identifiant non découvrable, garde déjà tolérante (C4-8) |
| Recherche de gamertags `live` anonyme | S1-02 | Déclassé P1 → P2 : limitée par IP, cache négatif, aucune écriture (C3-6) |
| Premier inscrit devient admin | S5-06 | Réfuté sur le principe (décision documentée, ADR 0029, tests) ; la purge refuse un admin. Reste P2 : course entre inscriptions simultanées, aucune garde « dernier admin » |
| Réglages illisibles → instance ouverte | S5-07 | Déclassé P2 : choix documenté et journalisé (`authz.go:111-131`), clé absente = verrouillé |
| Envoi de médias, P0 | S6-06 | Déclassé P0 → P1 : corps borné à 500 Mo et 50 fichiers, compte propriétaire requis (C3-5) |
| `/readyz`, erreurs de sync, détail Microsoft, 4xx bruts | S7-03/05/06/07 | Déclassés P1 → P2 : contenu pauvre, lecteur = propriétaire de sa propre session ou admin (C7-1) |
| Pas de CSP | S7-01 | Déclassé P1 → P2 : dette datée, aucun puits exploitable trouvé (C7-2) |
| `/health` empreinte | S7-04 | Déclassé P1 → P2 (C7-2) |
| Slug `..` de la route média en démo | S4-02 | Déclassé P1 → P2 : ne lit que la base synthétique de la fixture, déjà publique par l'API |
| `match_id` du rejeu → chemin | S4-04 | Déclassé P1 → P2 : 5xx masqués, contenu non renvoyé ; aucune écriture directe (vérification dédiée) |
| Démo sans confiance proxy | S2-05/S7-02/S8-04 | Déclassé P1 → P2 : impact = disponibilité d'une vitrine en lecture seule ; `limit_req` nginx en amont (C8-2) |
| Device-flow sans coupe-circuit réseau | S9-04 | Déclassé P1 → P2 : la garde démo tient, défense en profondeur seulement (C8-2) |
| `/auth/login` sans limiteur dédié | S6-12 | Déclassé P1 → P2 : limiteur global présent ; la vraie faille est C1-1 |
| `?xuid=` du catalogue | S1 | Réfuté : lecteur partagé non câblé, filtre inactif |
| Erreurs de `prestige_setup` divulguées | S7-07 | Réfuté : tombent en 500, masquées |
| Device-flow en démo | S5-03 | Réfuté pour la démo : POST refusé par la garde générale |

## Axes sans constat

Aucun : les neuf axes ont produit au moins un constat retenu.

## À vérifier sur le VPS (non accessible à l'audit ; opérations à annoncer)

- `.env.local` : `LEVELUP_TRUST_PROXY_HEADERS=true` (conditionne C1-1) et `LEVELUP_ENV=production`
  (sinon `/docs` et l'OpenAPI complet, admin compris, sont publics : la garde est ouverte par défaut,
  `openapi_docs.go:85`).
- nginx réellement actif (retouché par certbot) : `True-Client-IP`, `client_max_body_size`,
  `server_tokens`, `ssl_protocols`, zone `demo_limit` (`/etc/nginx/conf.d/rate_limit.conf`).
- `vm.overcommit_memory` (C2-3), RAM réelle, pare-feu, version de Go de l'image en cache et du VPS
  de calcul, origine de la base démo publiée (synthétique depuis le 2026-08-26 ?),
  `LEVELUP_REPLAY_PUBLIC` (C1-2).

## Suite

- Plan de correction `.ai/PLAN_SECURITE_2026-10-15.md` (grille `plan-review`), un lot par classe
  C1 à C10, chacun avec son garde-rail, tests rouges puis verts, gates exacts ; les restes de la
  démo (C8-2) forment un lot.
- **Décisions à obtenir de l'utilisateur avant le plan** : périmètre de l'import OpenSpartan (C4-3 :
  admin seul, ou limité aux matchs du propriétaire) ; sort de `/healthz/home` (admin seul ou
  suppression).
- **Tranché** : accès famille/groupe en lecture seule (2026-10-15) → C4-4 ; ADR 0029 à amender.
- Classés sans action immédiate : les P2 de C7 et C9-4 sont intégrés au plan seulement s'ils
  tiennent dans un lot sans le grossir ; sinon backlog.
