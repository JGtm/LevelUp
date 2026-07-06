# LOT S — Tableau route → garde (revue exhaustive S3)

> Livrable de l'item S3 du plan d'audits 2026-07 (« cause racine »). Recense
> TOUTES les routes montées sous `/api/v1` (`internal/api/server_apiv1.go` +
> `wire.MountAdminMonitoringRoutes`) et leur garde. Objectif du lot S : plus aucun
> endpoint mutant ou révélateur d'identité accessible sans auth.
>
> Invariant **no-op démo/single-user** : toutes les gardes reçoivent `cfg.DemoMode`
> → `authz.Enforced(DemoMode, AuthMode)` renvoie `false` en démo, et `AuthMode="none"`
> (single-user sans auth) court-circuite aussi. Onboarding/démo/public inchangés.

## 1. Public par conception — `r` nu, aucune garde (référentiel / liveness / auth-bootstrap)

Aucune donnée d'identité, aucune mutation. Justifie chaque `Mount(r)`/`r.Get`/`r.Post` nu restant.

| Route | Ligne | Justification |
|---|---|---|
| `GET /changelog` | 115 | Notes de version (markdown public). |
| `GET /bootstrap` | 118 | Shell React ; `available_players` filtré par session dans `BootstrapService`. |
| `GET /players` | 119 | **S4** : liste filtrée par ownership *in-service* (`BuildPlayersList(ctx, sess)`). |
| `GET /titles/{slug}/field-mappings` | 174 | Référentiel labels (derrière `MULTI_TITLE_API_ENABLED`). |
| `GET /titles/{slug}/capabilities` | 178 | Référentiel capacités titre. |
| `GET /titles/{slug}/feature-matrix` | 182 | Référentiel matrice de features. |
| `GET /titles/{slug}/catalog/*` | 197 | Référentiel playlists/pairs/maps. |
| `GET /help` (release notes) | 298 | Contenu public (README). |
| `GET /session` | 302 | État de session (nécessaire AVANT login). |
| `POST /auth/*`, `/xbox/*`, login/register/logout/password | 319/353/370 | Endpoints d'authentification — publics par nécessité. |
| `GET /media/feed-version` | 559 | Numéro de version de flux (polling). |
| `GET /assets/*` (medals/maps/battlepass/badge/spartan/drawer) | 563-575 | Référentiel d'images. |
| `GET /gamertags?q=` | 874 | Annuaire gamertags (503 si shared DB absente). |
| `GET /health,/healthz,/readyz` | 1208 | Sondes liveness/readiness. |

## 2. RequireAuth (utilisateur connecté)

| Route | Ligne | Note |
|---|---|---|
| `GET/POST/PATCH/DELETE /groups/*` | 379-385 | Fonction utilisateur (pas ops). |
| `GET /healthz/home` | 129 | **S6** — home du joueur de la session (révélateur d'identité). |
| `POST /setup/players`, `/setup/smoke-test` | 478 | **S8** — écrit `db_profiles.json` (gardes internes conservées). |
| `POST /import/openspartan` | 555 | **S3 (découvert)** — import mutant, était sur `r` nu (seul `!DemoMode`). Validation XUID interne conservée. |

## 3. RequireAuth + ownership (player-scoped, chokepoint ADR 0029)

`RequirePlayerOwnership` (`ownershipMW`, source unique) ; le groupe `/players/{player_slug}`
ajoute `RequireActiveTitle` + `UserFacingReadBudget`.

| Route | Ligne | Note |
|---|---|---|
| `GET /_diag/csr-coverage/{player_slug}` | 136 | **S6**. |
| `GET /_diag/progression/{player_slug}` | 145 | **S6**. |
| `/profiles/{player_slug}/titles/{slug}` (sync titre) | 584 | + `TitleSlugFromPath` (anti-bypass). |
| `/players/{player_slug}/*` — TOUTES les pages joueur | 591-869 | filters, match-history, career*, achievements*, match-view, match-events, engagement*, explorer, sessions, session-page, stats, home, season-pass, relations, squad, squad-v2, synthesis, citations, commendation-totals, media*, teammates, timeseries, exclusions, notifications, progression, coach, profile, patterns, campaign, `PATCH .../favorite`, prestige, compare, leaderboard, `POST .../sync`. (`*` = + `RequireCapability`.) |

## 4. RequireAuth + RequireAdmin (ops / admin)

| Route | Ligne | Note |
|---|---|---|
| `/lab/*` | 284 | Outil opérateur. |
| `/admin/*` (users, invites, invariants, contention, token-health, monitoring, titles, title-diag) | 390-427 | Ops. |
| `POST /sync/initial`, `/sync/all` | 521 | Sync joueur arbitraire (body). |
| `POST /backfill/start` | 526 | Backfill joueur arbitraire (body). |
| `/settings*` (PATCH + POST media/sessions/backup) | 462 | **S1**. |
| `POST /_admin/progression/backfill/{player_slug}` | 155 | **S2**. |
| `/watcher/*` | 879 | Présence Xbox RTA. |
| `GET /debug/vars` | 1222 | expvar. |

## 5. LoopbackOnly + RequireAuth + RequireAdmin

| Route | Ligne | Note |
|---|---|---|
| `/_diag/auto-sync/{snapshot,run,probe}` | 437 | **S5** — + `refresh_token_head`/`tail` retirés du probe (sha256 seul). |

## Grep de contrôle S3

Les seuls `.Mount(r)` / `r.Get(`/`r.Post(` sur `r` **nu** (hors `r.With(...)`, hors
groupe) doivent appartenir à la catégorie 1 (référentiel/liveness/auth). Vérif manuelle :

```bash
# Lister les montages sur r nu, puis confronter au tableau §1.
grep -nE '\.Mount\(r\)|r\.(Get|Post|Patch|Delete)\(' internal/api/server_apiv1.go
```

Aucune route mutante ou révélatrice d'identité ne doit y figurer hors catégorie 1.
Après le lot S : conforme (les endpoints mutants/identité sont en §2-§5).
