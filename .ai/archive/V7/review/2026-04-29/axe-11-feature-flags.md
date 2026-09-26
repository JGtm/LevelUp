# Axe 11 — Feature flags & scaffolding non documenté

Date : 2026-04-29
Branche : feat/multi-title-static-fs-rescope
Périmètre : tous les flags front + back, leur état (ON/OFF), leur documentation dans `.env.local.example`, et les modules « scaffoldés actifs mais non branchables sans connaissance tribale ».

> Cet axe a été ajouté **a posteriori** après la passe de vérification finale (cf. [verification-finale-scaffolding.md](verification-finale-scaffolding.md)), suite à la découverte que la revue 10-axes initiale avait laissé passer un trou structurel sur les feature flags non documentés. Le pattern « scaffolding then forget » est plus systémique que ne le suggéraient les axes 2, 4, 5, 8, 9 pris séparément.

> **Note 2026-04-29 (post-rédaction)** : pendant la vérification, le user a activé `PRESTIGE_ENABLED=true` localement et **ajouté** `PRESTIGE_ENABLED` + `MULTI_TITLE_API_ENABLED` à `.env.local.example` (cf. thought_log entries). Le BLOQUANT 1 « Prestige module non documenté » est donc **partiellement résolu** côté documentation. Le module reste OFF par défaut (commenté dans le `.env.local.example`), donc la dette structurelle (module dormant ~30+8 fichiers, pas d'ADR de bascule, pas de tests « flag ON ») demeure entière.

## Synthèse (3-5 lignes)

Verdict global : **moyen-mauvais**. Le projet héberge **5 feature flags actifs** dont **3 sur 5 ne figurent pas dans `.env.local.example`** (`PRESTIGE_ENABLED`, `LEVELUP_CONTRACT_VALIDATE`, `LEVELUP_NOTIFY_VERSIONS`) et **1 module entier (Prestige : ~30 fichiers backend + 8 composants frontend) est en attente d'activation depuis longtemps** sans plan de bascule documenté. À l'inverse, **10+ ENV vars backend sont lues sans être documentées** dans `.env.local.example` (`LEVELUP_LOG_JSON`, `LEVELUP_LOG_LEVEL`, `LEVELUP_AUTH_DIR`, `STEAM_API_KEY`, etc.). Les 3 flags front (`REJEU_2D_ENABLED`, `MULTI_TITLE_API_ENABLED` côté usage, plus celui implicite de Prestige via le fallback UI) sont cohérents entre eux mais **aucun n'est testé en mode "flag ON"** dans la suite Vitest.

## Inventaire complet des feature flags

| Flag | Type | Valeur défaut | Fichier source | Consommateurs prod | Documenté `.env.example` ? | Statut |
|---|---|---|---|---|---|---|
| `REJEU_2D_ENABLED` | front (constante TS) | `false` | `apps/web/src/lib/feature-flags.ts:14` | 1 (route `/replay` stub) | N/A (constante TS, pas env) | Stub externe attendu |
| `MULTI_TITLE_API_ENABLED` | back (env var) | OFF | `apps/go-api/internal/api/handlers/field_mappings.go:59` + autres handlers | 3+ endpoints, fallback côté front | **oui** (`.env.local.example:48-55`, ajouté 2026-04-29) | Migration en cours, branche `feat/multi-title-static-fs-rescope` |
| `PRESTIGE_ENABLED` | back (env var) | OFF | `apps/go-api/internal/prestige/sync_hook.go:23` | ~21 routes Go + 8 composants front | **oui** (`.env.local.example:57-62`, ajouté 2026-04-29) | **Module complet dormant** |
| `LEVELUP_CONTRACT_VALIDATE` | back (env var) | OFF | `apps/go-api/internal/api/middleware/contract_validate.go:24` | middleware de validation OpenAPI | **non** | Intentionnel (dev/CI uniquement) |
| `LEVELUP_NOTIFY_VERSIONS` | back (env var) | OFF | `apps/go-api/internal/notify/version.go:38` | webhook Discord | non (mais documenté dans `cmd/levelup/main.go:133`) | Intentionnel (opt-in prod) |

## Constats

### [BLOQUANT] Module Prestige complet (~30 fichiers backend + 8 composants frontend) en sommeil sans plan de bascule

> **Statut au 2026-04-29 (post-rédaction)** : la documentation `.env.local.example` a été ajoutée et le user a activé localement `PRESTIGE_ENABLED=true`. Le sous-titre original « non documenté » est obsolète. La dette structurelle restante = module en sommeil sans ADR de bascule, sans tests « flag ON », et 3 composants exportés sans importateur même flag activé (cf. axe 4 amendé).

- **Fichier:ligne** :
  - Flag : `apps/go-api/internal/prestige/sync_hook.go:23` (lecture env)
  - Bundle d'init : `apps/go-api/internal/api/prestige_setup.go` (entier)
  - Routes : `apps/go-api/internal/api/server.go:495-521` (~21 routes derrière le flag)
  - Front : 8 composants React dans `apps/web/src/features/prestige/`, dont 3 exportés sans importateur (`MomentCard`, `ArcSummary`, `StatsGlobales` — cf. axe 9 amendé)
  - Routes TanStack : `apps/web/src/routes/objectifs/index.tsx` + `apps/web/src/routes/palmares/prestige.tsx` (mountées dans `NavL1`)
- **Extrait** :
  ```go
  // sync_hook.go:23 — lecture du flag, sans fallback documenté ni log Info
  enabled := os.Getenv("PRESTIGE_ENABLED") == "true"
  ```
  ```tsx
  // Côté front, fallback UI affiché quand le flag est OFF :
  // "Le module Prestige n'est pas encore activé sur ce serveur (PRESTIGE_ENABLED=false)"
  ```
- **Problème (révisé)** : un module complet est physiquement présent dans le repo (chargeable, testable). La variable d'environnement est désormais documentée dans `.env.local.example`, mais **aucun ADR ne documente la politique de bascule** (à quelle condition activer en staging/prod ? quelle régression possible ? tests pré-bascule ?). 3 composants Prestige restent exportés sans importateur même flag activé (`MomentCard`, `ArcSummary`, `StatsGlobales` — cf. axe 4 amendé). Le coût de maintenance est payé chaque jour (typecheck, tests qui compilent les composants Prestige inutilisés, deps dans le bundle).
- **Action** :
  1. **Décision tranchée par ADR** : `0005-prestige-deferred.md` — soit le module est prêt → activation par défaut en CI/staging + tests de fumée en mode flag ON + branchement des 3 composants orphelins ; soit il n'est pas prêt → mise en veille **avec date d'expiration** (ex: ré-évaluation fin Q3 2026 ou suppression).
  2. **Tests « flag ON »** : ajouter au moins un test smoke dans `internal/contracttest/` qui démarre l'app avec `PRESTIGE_ENABLED=true` et appelle 1 endpoint représentatif.
  3. **Brancher ou archiver les 3 composants Prestige sans importateur** (cf. axe 4 amendé).

### [BLOQUANT] 10+ ENV vars backend lues sans être documentées dans `.env.local.example`

- **Fichier:ligne** : grep `os.Getenv` dans `apps/go-api/` vs lecture de `.env.local.example`. Liste des ENV vars **lues mais non documentées** :
  - `PRESTIGE_ENABLED` (cf. constat précédent)
  - `LEVELUP_CONTRACT_VALIDATE` (`apps/go-api/internal/api/middleware/contract_validate.go:24`)
  - `LEVELUP_NOTIFY_VERSIONS` (`apps/go-api/internal/notify/version.go:38`)
  - `LEVELUP_LOG_JSON`, `LEVELUP_LOG_LEVEL` (init logger)
  - `LEVELUP_API_PORT_OR_DEFAULT` (port HTTP)
  - `LEVELUP_AUTH_DIR` (chemin auth)
  - `LEVELUP_DISCORD_WEBHOOK_URL` (webhook notifs)
  - `STEAM_API_KEY` (intégration Steam)
  - `LEVELUP_CLIENT_ID` (auth)
- **Problème** : un dev qui clone le repo et copie `.env.local.example` vers `.env.local` aura un environnement **incomplet silencieusement** — ces vars tombent en valeur par défaut sans warning. Les nouvelles intégrations (Steam, Discord, auth) sont invisibles. C'est l'inverse du pattern Prestige (flag déclaré dans le code mais pas dans la doc) : ici, l'ENV est lue dans le code mais le contrat dev est cassé.
- **Action** : ajouter les 10+ vars manquantes à `.env.local.example` avec un commentaire d'usage + une valeur d'exemple non-secrète (ou `__REPLACE_ME__`). Optionnel : ajouter un test Go qui parse `.env.local.example` et vérifie que toute `os.Getenv` du code y est présente (test de contrat env).

### [DETTE] ENV vars documentées dans `.env.local.example` mais non lues côté Go

- **Fichier:ligne** : `.env.local.example:7` `TAILSCALE_FUNNEL_URL` — documenté pour `scripts/monitor_uptime.py` qui n'existe plus dans le repo (script Python supprimé lors de la migration Go).
- `.env.local.example:9` `DISCORD_WEBHOOK_URL` — listée comme "uptime-monitor" mais en réalité utilisée par `apps/go-api/internal/notify/discord.go:101`. Documentation imprécise.
- `SPNKR_SPARTAN_TOKEN`, `SPNKR_CLEARANCE_TOKEN`, `SPNKR_AZURE_REDIRECT_URI` — non lues dans `apps/go-api/`. Probablement utilisées par scripts externes ou couche Python legacy (cf. axe 9 sur les résidus Python).
- **Action** : nettoyer `.env.local.example` — supprimer les vars orphelines, corriger les commentaires imprécis (DISCORD_WEBHOOK_URL n'est plus uptime-monitor mais notifs Discord générales).

### [DETTE] Aucun test « flag ON » pour `MULTI_TITLE_API_ENABLED` ni `PRESTIGE_ENABLED`

- **Fichier:ligne** : aucun fichier `*_test.go` ne définit `os.Setenv("PRESTIGE_ENABLED", "true")` ni `os.Setenv("MULTI_TITLE_API_ENABLED", "true")` pour vérifier le comportement quand les flags sont activés.
- **Problème** : du code existe (handlers Prestige, handlers preview career multi-title) mais n'est jamais exercé en mode actif. La couverture tests Go est calculée flag OFF — la branche `if PRESTIGE_ENABLED` est marquée non couverte mais personne ne s'en rend compte (cf. axe 7 sur la couverture mensongère).
- **Action** : ajouter au moins un test smoke par flag dans `internal/contracttest/` qui démarre l'app avec le flag à `true` et appelle 1 endpoint représentatif.

### [DETTE] Pas de feature flag registry centralisé

- **Fichier:ligne** : aucun `apps/go-api/internal/featureflags/` ni `apps/web/src/lib/feature-flags/` consolidé. Les 5 flags sont éparpillés (3 fichiers Go distincts + 1 fichier TS).
- **Problème** : pas d'inventaire automatique, pas de log au démarrage qui dit « Flags actifs : MULTI_TITLE=false, PRESTIGE=false, NOTIFY_VERSIONS=true », pas d'endpoint admin `/admin/flags` pour debug. Diagnostic prod compliqué.
- **Action** : créer `internal/featureflags/registry.go` qui définit toutes les flags + leur valeur résolue, et logger au démarrage de l'app un récap structuré des flags actifs (`slog.InfoContext(ctx, "feature_flags", ...)`).

### [AMÉLIORATION] Règle CLAUDE.md « date d'expiration des guards » non appliquée aux feature flags

- **Référence** : `CLAUDE.md` interdit le pattern « Compatibility guard forever » (cf. règle 17 anti-patterns) — toute guard doit avoir une **date d'expiration** en commentaire.
- **Application** : aucun des 5 flags du repo n'a de date d'expiration documentée. `MULTI_TITLE_API_ENABLED` traîne depuis le début de la migration multi-titres ; `PRESTIGE_ENABLED` n'a aucun horizon. Le risque est exactement celui décrit par le pattern : flags qui deviennent permanents et créent du code mort silencieusement.
- **Action** : ajouter dans le registry (constat précédent) un champ `expires_at: time.Time` ou un commentaire `// expire: 2026-Q3 — décision de bascule ou archivage` à chaque flag. Au passage de la date, alerte CI ou suppression.

## Cartographie : flux d'un flag (Prestige)

Backend :
1. `cmd/levelup/main.go` lit env → init des composants
2. `internal/prestige/sync_hook.go:23` → `enabled := os.Getenv("PRESTIGE_ENABLED") == "true"`
3. `internal/api/prestige_setup.go` monte les handlers **conditionnellement**
4. Si `enabled=false` → handlers absents → routes absentes → toute requête vers `/api/v1/prestige/...` retourne 404
5. **Aucun log** au démarrage qui annonce « Prestige module : disabled »

Frontend :
1. `apps/web/src/features/prestige/PrestigeView.tsx` essaie de fetch `/api/v1/prestige/...`
2. 404 → fallback UI : « Le module Prestige n'est pas encore activé sur ce serveur (PRESTIGE_ENABLED=false) »
3. **Aucun typage** côté front qui dit « ce module dépend de PRESTIGE_ENABLED » — le dev front ne le sait que par essai-erreur

**Étapes à risque** :
- (1) un dev clone, lance, voit les 404, googlel `PRESTIGE_ENABLED`, ne trouve rien dans `.env.local.example`, abandonne
- (4) la 404 silencieuse cache un vrai bug si un endpoint Prestige est bien monté mais a un bug interne
- (2) le `os.Getenv("PRESTIGE_ENABLED") == "true"` est répété potentiellement à plusieurs endroits (à vérifier — pourrait être un sous-constat de cohérence)

## Suivi recommandé

1. **Décision Prestige** (priorité haute) : ADR `0005-prestige-deferred.md` qui tranche soit l'activation soit l'archivage, avec date d'expiration. Documente la politique de bascule.
2. **Audit `.env.local.example`** (1h) : ajouter les 10+ ENV vars manquantes, supprimer les 1-2 orphelines, corriger les commentaires imprécis.
3. **Feature flag registry** (2-3h) : `internal/featureflags/registry.go` + log au démarrage + endpoint admin `/admin/flags` derrière auth.
4. **Tests « flag ON »** (1-2 jours) : ajouter au moins 1 test smoke par flag dans `internal/contracttest/`.

## Constats hors-axe à reverser ailleurs

- **Axe 7 (tests)** : le manque de couverture en mode flag ON est aussi un problème de couverture pure (la branche `if flag` reste non couverte).
- **Axe 8 (logs)** : pas de log structuré au démarrage qui annonce les flags actifs — recoupe la dette observability.
- **Axe 9 (code mort)** : si Prestige n'est pas activé sous 6 mois, le module entier (~30 fichiers Go + 8 composants TS) devient candidat à archivage en `_archive/`.
