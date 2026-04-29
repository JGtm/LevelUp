# Vérification finale — pattern « scaffolding then forget »

Date : 2026-04-29
Branche : feat/multi-title-static-fs-rescope
But : trouver les cas manqués par la revue 10-axes (cas trigger : route /replay 2D signalée par le user).

## Synthèse

11 nouveaux cas trouvés en plus des 9 connus. Les plus structurels : (1) le système Prestige complet (backend + frontend) gardé derrière `PRESTIGE_ENABLED=false` non documenté dans `.env.local.example`, (2) trois routes cibles dans `notifications/navigation.ts` qui n'existent pas (`/help/changelog`, `/players/$slug/defis`, `/players/$slug/sync`) — émetteur backend `post_sync_deltas.go` aligné sur les mêmes routes fantômes, (3) `apps/web/src/app/routes/__root.tsx` jamais consommé (router pointe sur `src/routes/`). Recommandation : amender axes 2, 4, 8, 9 — la revue 10-axes a un trou structurel sur le module Prestige et un gap mineur sur les routes/endpoints orphelins. Le pattern « scaffolding then forget » est plus profond que le seul `/replay`.

## Catalogue des instances « scaffolding then forget »

### Connues (depuis la revue)
1. `useFieldLabel` documenté mais 0 usage prod (axe 4)
2. `MULTI_TITLE_API_ENABLED` OFF par défaut (axe 2)
3. `internal/observability/` package mort (axe 8)
4. Middleware `error_tracker` désactivé en dur (axe 8)
5. Tokens heatmap-divergent morts (axe 5)
6. `recharts` dep npm orpheline (axe 9)
7. 3 cmds Go avec `//go:build ignore` (axe 9)
8. 5 migrations one-shot non archivées (axe 9)
9. Route `/replay` 2D stub derrière `REJEU_2D_ENABLED = false` (signalée par le user)

### NOUVELLES (cette passe)

10. **Module Prestige complet derrière `PRESTIGE_ENABLED=false` non documenté**
    - Fichiers : `apps/go-api/internal/prestige/sync_hook.go:23` (déclare la flag), `apps/go-api/internal/api/prestige_setup.go` (bundle complet), `apps/go-api/internal/api/server.go:495-521` (~21 routes derrière le flag), 8 composants React `apps/web/src/features/prestige/`, 2 routes TanStack `objectifs/index.tsx` + `palmares/prestige.tsx` mountées dans NavL1.
    - Type : flag désactivé non documenté + module complet (~30 fichiers backend + 8 composants frontend) en attente d'activation.
    - Statut : oublié dans `.env.local.example` (la flag n'y figure pas, contrairement à `MULTI_TITLE_API_ENABLED`). Front affiche fallback "Le module Prestige n'est pas encore activé sur ce serveur (PRESTIGE_ENABLED=false)" donc usage en mode caché.
    - Pertinence : à brancher (ajouter à `.env.local.example`) ou à archiver explicitement (commentaire date d'expiration).

11. **Trois routes cibles fantômes dans `features/notifications/navigation.ts`**
    - Fichiers : `apps/web/src/features/notifications/navigation.ts:46,52,55` (routes `/players/$slug/defis`, `/help/changelog`, `/players/$slug/sync`).
    - Le backend émet vers ces mêmes routes : `apps/go-api/internal/api/post_sync_deltas.go:261,277` (TargetRoute `/players/%s/defis`).
    - Type : route orpheline (cible inexistante, jamais montée).
    - Statut : oublié — `defis` et `sync` existent comme noms dans NavL1 (pour `objectifs?tab=parcours`) mais pas comme routes TanStack. `/help/changelog` n'existe pas (route est `/changelog` simple).
    - Pertinence : utiliser fix — corriger les TargetRoute Go et le mapping front, ou créer les routes manquantes.

12. **`apps/web/src/app/routes/__root.tsx` orphelin (router pointe sur `src/routes/__root.tsx`)**
    - Fichier : `apps/web/src/app/routes/__root.tsx` (entier, ~70 L).
    - `apps/web/src/app/router/index.ts:11` importe `routeTree` depuis `@/routeTree.gen` qui est généré depuis `src/routes/`. Le `app/routes/__root.tsx` n'est référencé nulle part.
    - Type : composant non monté (duplicata).
    - Statut : oublié — probable dérive d'une migration de structure (`app/routes/` → `routes/`) où l'ancien fichier n'a pas été supprimé.
    - Pertinence : à archiver/supprimer.

13. **Composant React `MomentCard` exporté, 0 importateur**
    - Fichier : `apps/web/src/features/prestige/components/MomentCard.tsx:25`.
    - Aucun `import .* MomentCard` dans le code (le seul match est l'interface TypeScript homonyme dans `lib/prestige.ts:108`).
    - Type : composant non monté.
    - Statut : oublié dans le module Prestige.
    - Pertinence : utiliser fix (à brancher dans la vue Prestige) ou archiver.

14. **Composant React `ArcSummary` exporté, 0 importateur**
    - Fichier : `apps/web/src/features/prestige/components/ArcSummary.tsx:19`.
    - Type : composant non monté.
    - Statut : oublié dans le module Prestige.
    - Pertinence : utiliser fix ou archiver.

15. **Composant React `StatsGlobales` exporté, 0 importateur**
    - Fichier : `apps/web/src/features/prestige/components/StatsGlobales.tsx:29`.
    - Type : composant non monté.
    - Statut : oublié dans le module Prestige.
    - Pertinence : utiliser fix ou archiver.

16. **6 capabilities Halo déclarées mais non consommées via `HasCapability`**
    - Fichier : `apps/go-api/internal/domain/title/registry.go:31-37` déclare 7 capabilities.
    - Seul `CapAssetImages` est consommé runtime via `HasCapability(cap)` dans `apps/go-api/internal/api/handlers/assets_metadata.go:34,58`. Les 6 autres (`CapMatchmaking`, `CapFirefight`, `CapForge`, `CapMedia`, `CapRanked`, `CapCareer`) ne sont consommées qu'en init (registry, cmd_title.go pour print) et tests.
    - Type : capability inutilisée.
    - Statut : oublié — l'infrastructure de gating existe mais n'est jamais sollicitée pour ces capabilities. Probable scaffolding pour une logique multi-titre future.
    - Pertinence : à brancher (gating runtime) ou à supprimer (et garder uniquement `CapAssetImages`).

17. **Endpoint Go `/api/v1/titles/{slug}/preview/career` orphelin côté front**
    - Fichier : `apps/go-api/internal/api/server.go:260` (handler `previewHandler.GetCareerPreview`).
    - Aucun consommer dans `apps/web/src/`. Endpoint derrière `MULTI_TITLE_API_ENABLED`.
    - Type : endpoint API orphelin.
    - Statut : oublié — couvert partiellement par axe 2 (flag MT) mais l'axe 2 ne précise pas que l'endpoint preview/career n'est consommé par rien même si la flag est activée.
    - Pertinence : à archiver (preview admin/debug ?) ou brancher.

18. **Endpoint Go `/api/v1/players/{slug}/preview/career-multi-title` orphelin côté front**
    - Fichier : `apps/go-api/internal/api/server.go:278` (handler `playerPreviewHandler.GetCareerPreview`).
    - Aucun consommer dans `apps/web/src/`. Endpoint derrière `MULTI_TITLE_API_ENABLED`.
    - Type : endpoint API orphelin.
    - Statut : oublié.
    - Pertinence : à archiver ou brancher.

19. **Endpoint Go `GET /match-exclusions` orphelin côté front**
    - Fichier : `apps/go-api/internal/api/server.go:485` (handler `excl.ListExclusions`).
    - Le PATCH `/matches/{id}/exclusion` est consommé (`features/match-history/queries.ts:46`), mais le GET de listing n'est consommé par aucun composant.
    - Type : endpoint API orphelin.
    - Statut : oublié — vue admin probablement prévue puis pas implémentée.
    - Pertinence : à archiver.

20. **Endpoint Go `POST /media/reassociate` orphelin côté front**
    - Fichier : `apps/go-api/internal/api/server.go:460` (handler `media.PostReassociateMedia`).
    - Le front utilise `/media/associate` (POST, `features/media/queries.ts:359`) et `MediaPage.tsx` a une UI `reassociateFilePath` qui ouvre `MediaMatchPicker` lequel appelle `associate` (pas `reassociate`).
    - Type : endpoint API orphelin (doublon non utilisé).
    - Statut : oublié — probable refactor `reassociate` → `associate` qui n'a pas supprimé l'ancien handler.
    - Pertinence : à archiver.

## Catégorie A — flags désactivés

| Flag | Valeur | Fichier | Consommateurs | Statut |
|------|--------|---------|---------------|--------|
| `REJEU_2D_ENABLED` (front) | `false` (constante) | `apps/web/src/lib/feature-flags.ts:14` | 1 (route `/replay`) | Connu (cas 9) |
| `MULTI_TITLE_API_ENABLED` (back, env) | OFF par défaut | `apps/go-api/internal/api/handlers/field_mappings.go:59` | 3 endpoints + 7 composants front en fallback | Connu (cas 2) |
| `PRESTIGE_ENABLED` (back, env) | OFF par défaut | `apps/go-api/internal/prestige/sync_hook.go:23` | ~21 routes back + 8 composants front | **NOUVEAU (cas 10)** |
| `LEVELUP_CONTRACT_VALIDATE` (back, env) | OFF par défaut | `apps/go-api/internal/api/middleware/contract_validate.go:24` | middleware contract_validate | Intentionnel (dev/CI) |
| `LEVELUP_NOTIFY_VERSIONS` (back, env) | OFF par défaut | `apps/go-api/internal/notify/version.go:38` | discord version notif | Intentionnel (opt-in prod, documenté `cmd/levelup/main.go:133`) |

## Catégorie B — routes orphelines

Inventaire des 35 routes `apps/web/src/routes/`. Toutes ont au moins un `Link/navigate` ou sont accessibles via une logique de boot (`/setup`, `/login`, `/register`). Cas particuliers :

- `/lab/charts` — accessible uniquement par URL directe. **Intentionnel documenté** (`apps/web/src/components/charts/README.md:5,156`).
- `/replay` — derrière `REJEU_2D_ENABLED=false`. **Connu (cas 9)**.

**Routes cibles fantômes** (mentionnées dans le code mais inexistantes) :

- `/help/changelog` — référencée dans `apps/web/src/features/notifications/navigation.ts:52` (case `app_release`). Route inexistante. **NOUVEAU (cas 11)**.
- `/players/$slug/defis` — référencée dans `apps/web/src/features/notifications/navigation.ts:46` (case `challenge_added`/`challenge_completed`) et dans `apps/go-api/internal/api/post_sync_deltas.go:261,277`. Route inexistante. **NOUVEAU (cas 11)**.
- `/players/$slug/sync` — référencée dans `apps/web/src/features/notifications/navigation.ts:55` (case `sync_error`). Route inexistante. **NOUVEAU (cas 11)**.

## Catégorie C — endpoints API orphelins

Endpoints Go montés dans `server.go` mais non consommés par `apps/web/src/` :

- `GET /api/v1/titles/{slug}/preview/career` — **NOUVEAU (cas 17)**
- `GET /api/v1/players/{slug}/preview/career-multi-title` — **NOUVEAU (cas 18)**
- `GET /match-exclusions` — **NOUVEAU (cas 19)** (le PATCH `exclusion` est consommé, le listing non)
- `POST /media/reassociate` — **NOUVEAU (cas 20)** (le front utilise `/media/associate`)

Tous les autres endpoints recensés (~70 routes principales) ont un consommer front ou test E2E. Les ~21 endpoints Prestige derrière `PRESTIGE_ENABLED` sont consommés par les composants `features/prestige/` mais ces composants tombent en fallback si le flag est OFF (cas 10).

## Catégorie D — composants non montés

Composants React exportés sans importateur prod :

- `ArcSummary` — `apps/web/src/features/prestige/components/ArcSummary.tsx:19`. **NOUVEAU (cas 14)**.
- `StatsGlobales` — `apps/web/src/features/prestige/components/StatsGlobales.tsx:29`. **NOUVEAU (cas 15)**.
- `MomentCard` (composant) — `apps/web/src/features/prestige/components/MomentCard.tsx:25`. **NOUVEAU (cas 13)**. (Le seul autre `MomentCard` est une interface TypeScript homonyme dans `lib/prestige.ts:108`.)
- `apps/web/src/app/routes/__root.tsx` — fichier entier orphelin (router pointe sur `src/routes/`). **NOUVEAU (cas 12)**.

Les autres composants exportés ont au moins 1 importateur prod (vérifié sur ~80 composants `features/`).

## Catégorie E — stubs / placeholders

Stubs explicites dans le code prod (hors tests, hors `.ai/`, hors `docs/`) :

- `apps/go-api/cmd/refresh-metadata/main.go:274` — `// Pas de fetch Waypoint réel ici — placeholder pour l'intégration future.` Statut : intentionnel documenté (en attente endpoint Waypoint assets).
- `apps/go-api/internal/validation/compare.go:237` — `_ = queries // placeholder — les vraies stats bitmask sont dans shared (match_registry)`. Statut : structurel, dette documentée mais code mort dans la fonction.
- `apps/go-api/internal/api/post_sync_deltas.go:7-11` — TODOs pour `personal_record via référentiel records`, `threshold_crossed sur KD/winrate`, `objective_assigned`. Statut : partiellement implémenté (KD/winrate et best_kda sont en place, mais routes `/defis` ciblées n'existent pas — voir cas 11).
- Aucun autre stub critique trouvé dans le code prod.

Pas de bandeau "coming soon" / "en cours de développement" / "not implemented" dans les composants React.

## Catégorie F — capabilités non utilisées

Capabilities déclarées dans `apps/go-api/internal/domain/title/registry.go:31-37` mais non consommées via `HasCapability(cap)` :

- `CapMatchmaking`
- `CapFirefight`
- `CapForge`
- `CapMedia`
- `CapRanked`
- `CapCareer`

Seule `CapAssetImages` est consommée (assets_metadata.go:34,58). **NOUVEAU (cas 16)**.

## Catégorie G — ENV vars non lues

Variables déclarées dans `.env.local.example` mais sans aucun lecteur Go (et le repo est Go-only depuis la migration) :

- `TAILSCALE_FUNNEL_URL` — `.env.local.example:7`. Documenté pour `scripts/monitor_uptime.py` qui n'existe pas dans le repo (script Python supprimé). **Cas mineur** — env var orpheline.
- `DISCORD_WEBHOOK_URL` — `.env.local.example:9` listée comme uptime-monitor mais en réalité utilisée par `apps/go-api/internal/notify/discord.go:101`. **Pas un cas** : juste documentation imprécise.
- `SPNKR_SPARTAN_TOKEN`, `SPNKR_CLEARANCE_TOKEN`, `SPNKR_AZURE_REDIRECT_URI` — non lus dans `apps/go-api/`. Probablement utilisés par scripts externes ou la couche Python legacy. **À vérifier hors-scope** (peut être intentionnel pour scripts dev).

Variables backend lues mais **non documentées** dans `.env.local.example` : `PRESTIGE_ENABLED`, `LEVELUP_CONTRACT_VALIDATE`, `LEVELUP_NOTIFY_VERSIONS`, `LEVELUP_LOG_JSON`, `LEVELUP_LOG_LEVEL`, `LEVELUP_API_PORT_OR_DEFAULT`, `LEVELUP_AUTH_DIR`, `LEVELUP_DISCORD_WEBHOOK_URL`, `STEAM_API_KEY`, `LEVELUP_CLIENT_ID`. C'est l'inverse du pattern : env vars utilisées sans documentation. Lié aux cas 10 et axe 2.

## Recommandation d'amendement des rapports d'axes

- **Axe 2 (multi-titres)** : amender pour inclure les cas 17 et 18 (endpoints `preview/career` et `preview/career-multi-title` jamais consommés côté front même quand `MULTI_TITLE_API_ENABLED=true`) et le cas 16 (6 capabilities Halo déclarées mais non gatées runtime).
- **Axe 4 (front-react)** : amender pour inclure les cas 13/14/15 (3 composants Prestige exportés sans importateur) et le cas 12 (`apps/web/src/app/routes/__root.tsx` orphelin).
- **Axe 8 (logs/observability)** : pas d'amendement direct, mais relever le cas 11 (post_sync_deltas.go émet des notifications vers des routes fantômes — cohérent avec le pattern observability incomplète).
- **Axe 9 (dead code)** : amender pour inclure les cas 19, 20 (endpoints Go orphelins `/match-exclusions` GET et `/media/reassociate`), le cas 12 (`app/routes/__root.tsx`), le cas 16 (6 capabilities mortes).
- **Nouveau axe à considérer** : « feature flags non documentés ». Le cas 10 (Prestige) est un trou structurel non couvert par les 10 axes existants — ~30 fichiers backend + 8 composants frontend dans un état « scaffoldé activement, pas brancable sans connaissance tribale » car la flag n'est nulle part dans la doc d'env.
