# LOT S — Tableau route→garde exhaustif (audit sécurité 2026-07-02, item S3)

> Livrable de l'item S3 (« cause racine ») du PLAN_TRAITEMENT_AUDITS_2026-07.
> Recense CHAQUE route montée sous `/api/v1` et sa garde APRÈS le Lot S.
> Source du recensement : `internal/api/server.go` (NewRouter). Vérifié sur pièces.

## Invariant demo/single-user (préservé)

Tous les middlewares ajoutés par le Lot S (`RequireAuth`, `RequireAdmin`,
`RequirePlayerOwnership`) sont des **passe-plats automatiques** quand `DemoMode==true`
ou `AuthMode=="none"` :
- `require_auth.go:31` — `if demoMode || mode == "none" { return next }`
- `require_admin.go:18` — `if demoMode || authMode == "none" { return next }`
- `require_player_ownership.go:46` via `authz.Enforced()` (false en demo).

Conséquence : l'onboarding demo / single-user n'est PAS cassé. La garde ne s'active
qu'en `AuthMode` `password`/`xbox` hors demo.

## Tableau

| Route (relative à /api/v1) | Garde APRÈS Lot S | Statut |
|---|---|---|
| `GET /bootstrap` | aucune au routing + `filterOwnedPlayers` interne | public OK (filtré par session) |
| `GET /players` | aucune au routing + **`filterOwnedPlayers` interne (S4)** | **corrigé S4** |
| `GET /healthz/home?player=` | **RequireAuth+RequireAdmin (S6)** + NoStore | **corrigé S6** |
| `GET /_diag/csr-coverage/{player_slug}` | **RequireAuth+RequireAdmin (S6)** + NoStore | **corrigé S6** |
| `GET /_diag/progression/{player_slug}` | **RequireAuth+RequireAdmin (S6)** + NoStore | **corrigé S6** |
| `POST /_admin/progression/backfill/{player_slug}` | **RequireAuth+RequireAdmin (S2)** + NoStore | **corrigé S2** |
| `PATCH /settings` + 4 POST (`media/scan`, `sessions/recalculate`, `backup/run`, `media/reset-index`) | **RequireAuth+RequireAdmin (S1)** | **corrigé S1** |
| `POST /setup/players`, `POST /setup/smoke-test` | **RequireAuth (S8)** + gardes internes (self-provision/lockdown/identité) | **corrigé S8** |
| `POST /import/openspartan` `[!DemoMode]` | **RequireAuth (S3)** + validation identité interne (409 identity_mismatch) | **corrigé S3** |
| `/_diag/auto-sync/{snapshot,run,probe}` | LoopbackOnly + **RequireAuth+RequireAdmin (S5)** ; payload probe sans head/tail | **corrigé S5** |
| `GET /jobs/{job_id}` | aucune (job_id = UUID opaque) | **Découverte §7** |
| `MOUNT gamertag directory ?q=` | aucune (annuaire, 503 si shared DB absente) | **Découverte §7** |
| `GET /titles/{slug}/field-mappings` `[flag MULTI_TITLE_API_ENABLED]` | aucune (référentiel TOML, read-only) | public assumé (config, pas d'identité/mutation) |
| `GET /titles/{slug}/capabilities` `[flag]` | aucune | public assumé |
| `GET /titles/{slug}/features` `[flag]` | aucune | public assumé |
| `GET /catalog/*` (playlists/pairs/maps) `[flag]` | aucune | public assumé (référentiel) |
| `MOUNT help / release-notes` | aucune | public OK (changelog 1re partie) |
| `MOUNT session context` | aucune | public OK (lecture session courante) |
| `MOUNT auth device-flow / user auth (login/register/logout/password) / xbox SSO` | aucune | public OK (flux d'auth par design) |
| `GET /media/feed-version` | aucune | public OK (version de flux, polling) |
| `GET /assets/*` (medals/maps/battlepass/challenge-badge/spartan) | aucune | public OK (assets) |
| `MOUNT asset-drawer maps/weapons` | aucune | public OK (référentiel) |
| `GET /health, /healthz, /readyz` (racine, hors /api/v1) | aucune | public OK (sondage infra) |
| `GET /*` SPA catch-all (hors /api/v1) | aucune | public OK (front statique) |
| `MOUNT /debug/vars` | RequireAuth+RequireAdmin | déjà gardé |
| `MOUNT lab` | RequireAuth+RequireAdmin | déjà gardé |
| `ROUTE /admin/*` (users/invites/invariants/db-contention/token-health/monitoring/titles/title-diag) | RequireAuth+RequireAdmin | déjà gardé |
| `GROUP /groups` (list/create/rename/delete/invite/leave/remove) | RequireAuth | déjà gardé |
| `GROUP /sync/initial + /sync/all + /backfill/start` | RequireAuth+RequireAdmin | déjà gardé (D3-01) |
| `ROUTE /watcher` (status/auth-status/subscriptions/auth-start) | RequireAuth+RequireAdmin | déjà gardé |
| `ROUTE /profiles/{player_slug}/titles/{slug}` (TitleSync) | TitleSlugFromPath + RequirePlayerOwnership (fail-closed S7) | déjà gardé + **S7** |
| `ROUTE /players/{player_slug}/*` (~40 routes) | RequireActiveTitle + UserFacingReadBudget + RequirePlayerOwnership (fail-closed S7) + RequireCapability sur sous-groupes | déjà gardé + **S7** |

## Découvertes §7 (hors périmètre Lot S — à arbitrer avec l'utilisateur)

Ces endpoints ont été surfacés par la revue exhaustive S3 mais ne figurent PAS dans les
findings sécurité de l'audit (B1/B2/M2/M4). Non traités pour respecter le périmètre du
lot ; candidats à un durcissement ultérieur :

1. **`GET /jobs/{job_id}`** — statut de job par UUID opaque. En multi-user, si un UUID
   fuit, un tiers pourrait lire le statut/erreur du job d'un autre. Risque faible (UUID
   non devinable). Candidat : `RequireAuth` (défense en profondeur).
2. **`MOUNT gamertag directory ?q=`** — annuaire de recherche gamertag. Permet
   potentiellement de l'énumération (les gamertags Xbox sont semi-publics). Candidat :
   `RequireAuth`.
3. **`field-mappings` / `capabilities` / `features` / `catalog`** (derrière
   `MULTI_TITLE_API_ENABLED`) — exposent la config titre (référentiel TOML) sans auth.
   Read-only, aucune identité joueur ni mutation. Acceptable public tant que le flag reste
   OFF en prod ; à revoir si activé.

`import/openspartan` était dans cette catégorie « write-path non gardé » ; il a été gardé
`RequireAuth` (S3) car c'est une mutation d'instance analogue à `/setup`.
