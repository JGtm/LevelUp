# LOT S — Tableau route → garde (revue exhaustive S3)

> Livrable de l'item S3 du plan d'audits 2026-07 (« cause racine »). Recense
> TOUTES les routes montées sous `/api/v1` (`internal/api/server_apiv1.go` +
> `wire.MountAdminMonitoringRoutes`) et leur garde. Objectif du lot S : plus aucun
> endpoint mutant ou révélateur d'identité accessible sans auth.
>
> Invariant **no-op démo/single-user** : toutes les gardes reçoivent `cfg.DemoMode`
> → `authz.Enforced(DemoMode, AuthMode)` renvoie `false` en démo, et `AuthMode="none"`
> (single-user sans auth) court-circuite aussi. Onboarding/démo/public inchangés.
>
> **RAFRAÎCHI 2026-07-07 (lot V3)** : numéros de ligne re-pointés sur
> `server_apiv1.go` (déplacements post-J2/K) ; libellés corrigés (`POST /session/context`,
> `GET /directory/gamertags/search`) ; `GET /jobs/{job_id}` ajouté (désormais gardé
> RequireAuth). **Garde-rail permanent** : ce tableau était un relevé manuel — il est
> désormais adossé au ratchet comportemental
> `internal/api/bare_routes_ratchet_test.go` (V3c) qui `chi.Walk` le routeur en
> mode enforcement et échoue sur toute route publique hors allowlist datée. C'est
> lui, pas cette table, qui MORD sur une future route nue.

## 1. Public par conception — `r` nu, aucune garde (référentiel / liveness / auth-bootstrap)

Aucune donnée d'identité, aucune mutation. Justifie chaque `Mount(r)`/`r.Get`/`r.Post` nu restant.

| Route | Ligne | Justification |
|---|---|---|
| `GET /changelog` | 115 | Notes de version (markdown public), via `registerChangelogHuma`. |
| `GET /bootstrap` | 118 | Shell React ; `available_players` filtré par session dans `BootstrapService`. |
| `GET /players` | 119 | **S4** : liste filtrée par ownership *in-service* (`BuildPlayersList(ctx, sess)`). |
| `GET /titles/{slug}/field-mappings` | 174 | Référentiel labels (derrière `MULTI_TITLE_API_ENABLED`). |
| `GET /titles/{slug}/capabilities` | 178 | Référentiel capacités titre. |
| `GET /titles/{slug}/feature-matrix` | 182 | Référentiel matrice de features. |
| `GET /titles/{slug}/catalog/{playlists,pairs,maps}` | 197 | Référentiel playlists/pairs/maps. |
| `GET /help/release-notes` | 298 | Contenu public (README), via `HelpHandler.Mount`. |
| `POST /session/context` | 302 | Établit/retourne le contexte de session (nécessaire AVANT login) ; POST protégé CSRF. `sessionHandler.Mount`. |
| `POST /auth/*` (login/register/logout/password), `POST/GET /auth/device-flow/*` | 320+/346 | Endpoints d'authentification — publics par nécessité (POST protégés CSRF). |
| `GET /media/feed-version` | 573 | Numéro de version de flux (polling). |
| `GET /assets/*` (medals/maps/battlepass/challenge-badge/spartan + `{title_id}/{maps,medals,weapons}`) | 577-587 | Référentiel d'images/catalogues. |
| `GET /directory/gamertags/search?q=` | 888 | Annuaire gamertags (503 si shared DB absente), via `registerGamertagHuma`. |
| `GET /health,/healthz,/readyz` | 1222 | Sondes liveness/readiness, via `healthH.Mount`. |
| `GET /static/*`, `/static/commendations/*` (GET/HEAD/OPTIONS) | mountSPA | Assets statiques servis par le backend. |

## 2. RequireAuth (utilisateur connecté)

| Route | Ligne | Note |
|---|---|---|
| `GET/POST/PATCH/DELETE /groups/*` | 379-385 | Fonction utilisateur (pas ops). |
| `GET /healthz/home` | 129 | **S6** — home du joueur de la session (révélateur d'identité). |
| `GET /jobs/{job_id}` | 490 | **V3 (découvert VF-3)** — statut de job (PlayerSlug + type + messages d'erreur = révélateur d'identité) ; était sur `r` nu (root `humaAPI`). Migré : l'API Huma jobs est adossée à `r.With(RequireAuth)`. Verrouillé par le ratchet V3c. |
| `POST /setup/players`, `/setup/smoke-test` | 480 | **S8** — écrit `db_profiles.json` (gardes internes conservées). |
| `POST /import/openspartan` | 569 | **S3 (découvert)** — import mutant, était sur `r` nu (seul `!DemoMode`). Validation XUID interne conservée. |

## 3. RequireAuth + ownership (player-scoped, chokepoint ADR 0029)

`RequirePlayerOwnership` (`ownershipMW`, source unique) ; le groupe `/players/{player_slug}`
ajoute `RequireActiveTitle` + `UserFacingReadBudget`.

| Route | Ligne | Note |
|---|---|---|
| `GET /_diag/csr-coverage/{player_slug}` | 136 | **S6**. |
| `GET /_diag/progression/{player_slug}` | 145 | **S6**. |
| `/profiles/{player_slug}/titles/{slug}` (sync titre) | 598 | + `TitleSlugFromPath` (anti-bypass). |
| `/players/{player_slug}/*` — TOUTES les pages joueur | 605-882 | filters, match-history, career*, achievements*, match-view, match-events, engagement*, explorer, sessions, session-page, stats, home, season-pass, relations, squad, squad-v2, synthesis, citations, commendation-totals, media*, teammates, timeseries, exclusions, notifications, progression, coach, profile, patterns, campaign, `PATCH .../favorite`, prestige, compare, leaderboard, `POST .../sync` (`syncH.MountDelta`, l.882). (`*` = + `RequireCapability`.) |

## 4. RequireAuth + RequireAdmin (ops / admin)

| Route | Ligne | Note |
|---|---|---|
| `/lab/*` | 287 | Outil opérateur. |
| `/admin/*` (users, invites, invariants, contention, token-health, monitoring, titles, title-diag) | 390-427 | Ops. `MountAdminMonitoringRoutes` (l.411) monte aussi `/admin/monitoring/jobs` sous ce groupe. |
| `POST /sync/initial`, `/sync/all` | 528 | Sync joueur arbitraire (body). |
| `POST /backfill/start` | 533 | Backfill joueur arbitraire (body). |
| `/settings*` (PATCH + POST media/sessions/backup) | 465 | **S1**. |
| `POST /_admin/progression/backfill/{player_slug}` | 155 | **S2**. |
| `/watcher/*` | 893 | Présence Xbox RTA. |
| `GET /debug/vars` | 1242 | expvar. |

## 5. LoopbackOnly + RequireAuth + RequireAdmin

| Route | Ligne | Note |
|---|---|---|
| `/_diag/auto-sync/{snapshot,run,probe}` | 437-438 | **S5** — + `refresh_token_head`/`tail` retirés du probe (sha256 seul). |

## Grep de contrôle S3

Les seuls `.Mount(r)` / `r.Get(`/`r.Post(` sur `r` **nu** (hors `r.With(...)`, hors
groupe) doivent appartenir à la catégorie 1 (référentiel/liveness/auth). Vérif manuelle :

```bash
# Lister les montages sur r nu, puis confronter au tableau §1.
grep -nE '\.Mount\(r\)|r\.(Get|Post|Patch|Delete)\(' internal/api/server_apiv1.go
```

Aucune route mutante ou révélatrice d'identité ne doit y figurer hors catégorie 1.
Après le lot S : conforme (les endpoints mutants/identité sont en §2-§5).

## Garde-rail automatisé (lot V3c — remplace le grep manuel)

Le grep ci-dessus était MANUEL — il a laissé passer `GET /jobs/{job_id}` (VF-3, il
était monté sur le root `humaAPI` sans garde, donc pas un `.Mount(r)`/`r.Get(` nu
détectable au grep). Le garde-rail permanent est désormais
`internal/api/bare_routes_ratchet_test.go` : il `chi.Walk` le routeur ASSEMBLÉ en
mode enforcement (`DemoMode=false`, `AuthMode="password"`), compose la chaîne de
middlewares de chaque route autour d'un handler bidon, et envoie une requête
anonyme. Une route qui ne répond pas 401/403 doit figurer dans l'allowlist datée du
test (= la catégorie §1 ci-dessus) ; sinon le test échoue. Vérifié mordant dans les
deux sens (retrait d'une garde → rouge ; entrée d'allowlist morte → rouge). Toute
évolution de cette table doit rester cohérente avec cette allowlist.
