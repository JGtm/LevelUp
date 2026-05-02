# SPRINT_44_WORKPACKAGES.md — Work Packages techniques du Sprint 44

> Document d'exécution du Sprint 44.
> Il transforme le cadrage multi-titres en lots techniques concrets, ordonnés par couche du runtime.

## Rôle du document

Ce document existe pour éviter que le Sprint 44 reste une intention large et sous-spécifiée.

Il sert à :

1. découper l'introduction du multi-titres en work packages cohérents ;
2. expliciter les couches impactées et leurs dépendances ;
3. rendre les preuves de succès vérifiables lot par lot ;
4. cadrer la migration et la non-régression Halo Infinite avant implémentation.

## Références amont

1. [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md)
2. [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md)
3. [ADR_S44_MULTI_TITLE_NAMESPACE.md](ADR_S44_MULTI_TITLE_NAMESPACE.md)
4. [AUDIT_CONSOLIDE.md](AUDIT_CONSOLIDE.md)
5. [GO_ARCHITECTURE_RULES.md](GO_ARCHITECTURE_RULES.md)

## Principe directeur

Le Sprint 44 ne consiste pas à injecter `title_slug` dans le code existant au fil de l'eau.

Le sprint doit introduire :

1. une vérité centrale des titres supportés ;
2. une résolution de chemins et de contexte runtime cohérente ;
3. une migration opérable depuis l'état Halo Infinite only ;
4. une validation explicite de la non-régression et de l'isolement inter-titres.

## Estimation

**10–14 jours** (revue à la hausse après audit du code Go ; estimation initiale 6–9j).

Facteurs de sous-estimation identifiés :
- WP1 : refactor `PlayerResolver` + pool DuckDB (13 fichiers `*_repo.go`) + migration `db_profiles.json` v3
- WP1 : **29 références de chemins hardcodés** réparties dans 15 fichiers (voir inventaire ci-dessous)
- WP3 : migration physique de fichiers DuckDB sur Windows (locks, chemins longs, idempotence)
- WP1/WP4 : provisioning joueur (`POST /setup/players`, `GET /players`) + préférences locales (`lastPlayerSlug`) doivent devenir title-aware
- WP4 : réalignement frontend React réel (stores, `routeTree` TanStack, routes player-scoped, `queryKeys`, hooks de query/mutation, liens de navigation, handlers MSW/Playwright, types TS et bootstrap consumer)
- WP4 : création du corpus synthétique second titre (~0.5–1j)
- WP4 : **décision architecturale OpenAPI** : 23 endpoints avec `{player_slug}` doivent intégrer `{title_slug}`
- WP1/WP3 : **demo mode** (`DemoFixturesDir`) doit devenir title-aware
- WP4/WP5 : commandes ops concernées du binaire `levelup` (`backup`, `restore`, `archive`, `index-media`, `seed`, `healthcheck`, `gate-check`) + résolution de titre au démarrage du binaire `server`

**Auth hors périmètre** : le flow MSAL est titre-agnostique (confirmé par audit). Aucune modification `internal/platform/auth/`.

**Coexistence Python** : non requise. Le Go est la seule baseline à ce stade.

## Inventaire des chemins hardcodés à migrer

> Référence: **29 occurrences** dans **15 fichiers** (audit du 16/04/2026).
> Après introduction du `PathResolver` (WP0), chaque fichier doit remplacer ses `filepath.Join(repoRoot, "data", ...)` par un appel au résolveur.

| Fichier | Occurrences | Nature |
|---------|:-----------:|--------|
| `cmd/server/main.go` | 3 | Init warehouse + demo fallback |
| `cmd/levelup/main.go` | 2 | CLI shared/metadata paths |
| `internal/config/player_resolver.go` | 3 | Résolution warehouse/shared/metadata |
| `internal/validation/gate.go` | 4 | Validation santé DB |
| `internal/ops/archive.go` | 2 | Archivage joueurs |
| `internal/ops/backup.go` | 1 | Sauvegarde joueur |
| `internal/ops/restore.go` | 1 | Restauration joueur |
| `internal/ops/healthcheck.go` | 3 | Healthcheck DB |
| `internal/ops/diagnose.go` | 1 | Diagnostic DB |
| `internal/ops/media.go` | 2 | Gestion médias |
| `internal/sync/engine.go` | 1 | Construction chemin shared |
| `internal/platform/duckdb/pool.go` | 5 | GlobalPool sync.Map (`gamertag` → `{title}:{gamertag}`) |
| **Total** | **~29** | |

## Matrice de scope des chemins

> Tous les chemins ne deviennent pas title-aware. Le Sprint 44 doit figer explicitement ce qui est namespacé par titre et ce qui reste global.

| Artefact | Scope | Invariant attendu |
|----------|-------|-------------------|
| `data/titles/{title_slug}/warehouse/metadata.duckdb` | Title-aware | Référentiels par titre |
| `data/titles/{title_slug}/warehouse/shared_matches_v2.duckdb` | Title-aware | Shared matches isolés par titre |
| `data/titles/{title_slug}/warehouse/shared_pve.duckdb` | Title-aware | Présent seulement si le titre le nécessite |
| `data/titles/{title_slug}/players/{gamertag}/stats.duckdb` | Title-aware | Enrichissements joueur isolés par titre |
| `data/titles/{title_slug}/players/{gamertag}/archive/` | Title-aware | Archives isolées par titre |
| `data/titles/{title_slug}/players/{gamertag}/captures/` | Title-aware | Index média et captures isolés par titre |
| `data/titles/{title_slug}/backups/{gamertag}/` | Title-aware | Backups/restores ne mélangent jamais deux titres |
| `tests/fixtures/titles/{title_slug}/ref_player/` | Title-aware | Demo mode et corpus de test par titre |
| `db_profiles.json` v3 | Global | Contient plusieurs titres, mais reste un fichier racine unique |
| `app_settings.json` | Global | Settings instance-level, pas dupliqués par titre |
| `data/sessions/{session_id}.json` | Global | La session porte `current_title_slug` |
| `data/cache/jobs.json` | Global | Les jobs restent dans un store unique mais chaque job porte `title_slug` |

Le `PathResolver` doit encoder cette matrice et refuser toute reconstruction ad hoc de chemin dans les handlers, services, ops ou tests.

## Ordre d'exécution recommandé

1. WP0 — verrouiller les invariants de design et les contrats internes.
2. WP1 — rendre config, chemins et `PlayerResolver` title-aware.
3. WP2 — propager le contexte de titre dans session, jobs et bootstrap.
4. WP3 — brancher le namespace de stockage et la migration legacy.
5. WP4 — réaligner contrats API, frontend, corpus et jeux de test.
6. WP5 — fermer observabilité, runbook, rollback et gates finaux.

## WP0 — Noyau de design multi-titres

### But

Créer la colonne vertébrale du runtime multi-titres avant toute propagation du champ `title_slug`.

### Livrables

1. un `TitleRegistry` décrivant les titres supportés, leur slug, leur provider, leur statut produit et leurs capacités ;
2. un `TitleDescriptor` ou équivalent côté domaine ;
3. un `PathResolver` title-aware responsable des chemins runtime, et non une concaténation dispersée — doit couvrir les **29 références de chemins hardcodés** identifiées (voir inventaire ci-dessus) ;
4. des helpers de résolution du titre courant avec fallback legacy explicite pour `halo_infinite`.

### Couche impactée

1. `apps/go-api/internal/domain/`
2. `apps/go-api/internal/config/`
3. éventuellement un sous-package dédié de type `internal/platform/runtime/` ou `internal/runtime/`

### Contraintes

1. aucun handler ou service ne doit reconstruire ses chemins title-aware en local ;
2. le registre de titres doit être testable sans DuckDB ni HTTP ;
3. la compatibilité legacy doit être déclarée et non implicite.

### Preuves attendues

1. tests unitaires du registre de titres ;
2. tests unitaires du résolveur de chemins ;
3. couverture ciblée élevée sur les nouveaux modules de design.

## WP1 — Config, profils, `PlayerResolver` et résolution des chemins

### But

Rendre la configuration applicative et la résolution joueur compatibles avec un runtime par titre, sans casser la configuration Halo Infinite existante.

### Livrables

1. résolution title-aware de `db_profiles`, `app_settings`, `sessions` et répertoires de données ;
2. migration `db_profiles.json` vers un format v3 title-aware, avec lecture rétro-compatible du format actuel ;
3. stratégie claire de lecture legacy vs écriture cible ;
4. **refactor `PlayerResolver`** pour accepter `(title_slug, player_slug)` au lieu de `player_slug` seul ;
5. **pool DuckDB** : changement de clé `gamertag` → `{title}:{gamertag}` dans le `sync.Map` + `singleflight.Group` (impact sur 13 fichiers `*_repo.go` qui reçoivent `*PlayerDB` — changement transparent via la struct enrichie) ;
6. **demo mode** : `resolveDemoPlayer()` et `DemoFixturesDir` doivent devenir title-aware (fixtures sous `tests/fixtures/ref_player/` → `tests/fixtures/titles/{title_slug}/ref_player/`) ;
7. profils joueurs compatibles avec un contexte par titre ;
8. `POST /setup/players` et `GET /players` deviennent explicitement title-aware ; le provisioning ne doit plus écrire implicitement dans le layout mono-titre ;
9. conventions documentées pour `data/titles/{title_slug}/warehouse/...` et `data/titles/{title_slug}/players/{gamertag}/...`.

### Couche impactée

1. `apps/go-api/internal/config/config.go`
2. `apps/go-api/internal/config/feature_flags.go` si le titre courant influence les routes actives
3. lecture/écriture de `db_profiles.json` et éventuellement des fichiers de config dérivés
4. `apps/go-api/internal/api/middleware/player_resolver.go` — **pivot central du refactor**
5. `apps/go-api/internal/platform/duckdb/pool.go` et les 13 fichiers `*_repo.go`
6. `apps/go-api/cmd/server/main.go` — init warehouse paths + demo fallback
7. `apps/go-api/internal/ops/` — 6 fichiers (archive, backup, restore, healthcheck, diagnose, media) avec chemins hardcodés
8. `apps/go-api/internal/validation/gate.go` — 4 chemins de validation santé DB
9. `apps/go-api/internal/sync/engine.go` — chemin shared
10. `apps/go-api/internal/api/handlers/setup.go` — création de profil et matérialisation du layout cible
11. `apps/go-api/internal/service/bootstrap_service.go` — liste des joueurs cohérente avec le titre courant

### Risques locaux

1. fuite de chemins legacy vers le mode namespacé ;
2. double source de vérité entre `repoRoot` et les chemins résolus ;
3. ambiguïté sur le joueur courant si deux titres utilisent le même gamertag ;
4. régression silencieuse si un `*_repo.go` oublie le préfixe `{title}:` dans la clé pool ;
5. **demo mode** : `DemoFixturesDir` doit être migré pour pointer vers un layout title-aware.
6. le provisioning setup peut continuer à écrire dans `data/players/...` ou dans `db_profiles.json` v2 si `setup.go` n'est pas inclus dans le lot.

### Preuves attendues

1. tests unitaires de config title-aware et de `db_profiles.json` v3 (lecture legacy + nouveau format) ;
2. tests unitaires du `PlayerResolver` title-aware (mode réel et mode démo) ;
3. test d'intégration vérifiant que deux titres avec le même gamertag ne partagent pas de pool DuckDB ;
4. tests d'intégration sur fichiers temporaires ;
5. démonstration qu'un dépôt déjà namespacé et un dépôt legacy sont tous deux lisibles ;
6. vérification que les 6 fichiers `internal/ops/` passent bien par le `PathResolver`.
7. tests d'intégration `POST /setup/players` + `GET /players` : provisioning et listing restent cohérents avec le titre courant.

## WP2 — Session, jobs, bootstrap et mécanisme de switch titre

### But

Faire porter le titre courant par le runtime stateful au lieu de le laisser dériver du contexte implicite.
Préparer la structure pour un changement de titre à runtime (pas de bouton UI, mais la plomberie complète).

> **Auth hors périmètre** : le flow MSAL est titre-agnostique (confirmé par audit).
> Aucune modification requise dans `internal/platform/auth/`.

### Livrables

1. `SessionData` enrichie avec le titre courant ;
2. `SessionContextRequest` enrichi pour accepter le switch de titre via le contrat existant `POST /session/context` ;
3. jobs persistants associés explicitement à un titre (enrichir `JobMeta`) ;
4. bootstrap exposant le titre courant, les titres disponibles et les métadonnées associées ;
5. stratégie de re-hydratation documentée pour le switch de titre runtime.

### Contrat de domaine cible

#### SessionData (`internal/domain/session.go`)

```go
type SessionData struct {
    SessionID          string
    CurrentPlayerSlug  *string
    CurrentTitleSlug   string    // NEW — défaut "halo_infinite", jamais vide
    Locale             string
    AuthReady          bool
    // ... champs existants inchangés
}
```

**Invariants** :
- `CurrentTitleSlug` est **non-nul** et **non-vide** (valeur par défaut `"halo_infinite"` à la création de session)
- un changement de `CurrentTitleSlug` **invalide** `CurrentPlayerSlug` (le joueur courant peut ne pas exister dans le nouveau titre)
- le JSON de session stocké sur disque (`data/sessions/{session_id}.json`) doit inclure `current_title_slug`
- sessions legacy sans `current_title_slug` → désérialisation avec fallback `"halo_infinite"`

#### SessionContextRequest (enrichi)

```go
type SessionContextRequest struct {
    PlayerSlug *string `json:"player_slug"`
    TitleSlug  *string `json:"title_slug"`   // NEW — optionnel, nil = pas de changement
    Locale     *string `json:"locale"`
}
```

**Comportement du handler `POST /session/context`** :
1. si `title_slug` est fourni et différent du titre courant → **switch titre** :
   a. valider que le titre existe dans `TitleRegistry` (sinon 422 `unknown_title`)
   b. mettre à jour `session.CurrentTitleSlug`
   c. **invalider** `session.CurrentPlayerSlug` (mis à `nil`)
   d. logger `slog.Info("title_switched", "session_id", sid, "from", old, "to", new)`
   e. retourner le nouveau bootstrap complet dans la réponse (titres, joueurs du nouveau titre)
2. si `title_slug` est nil ou identique au titre courant → comportement actuel (pas de switch)
3. si `player_slug` est fourni → résolu dans le contexte du titre courant (après switch éventuel)

#### BootstrapResponse (`internal/domain/bootstrap.go`)

```go
type BootstrapResponse struct {
    CurrentTitle     TitleSummary    `json:"current_title"`      // NEW
    AvailableTitles  []TitleSummary  `json:"available_titles"`   // NEW
    CurrentPlayer    *PlayerSummary  `json:"current_player"`
    AvailablePlayers []PlayerSummary `json:"available_players"`
    FeatureFlags     FeatureFlags    `json:"feature_flags"`
    Capabilities     CapabilityMap   `json:"capabilities"`
    // ... champs existants inchangés
}

type TitleSummary struct {
    Slug         string   `json:"slug"`          // "halo_infinite"
    Name         string   `json:"name"`          // "Halo Infinite"
    IconURL      string   `json:"icon_url"`      // optionnel, vide si non disponible
    PlayerCount  int      `json:"player_count"`  // nombre de joueurs provisionnés pour ce titre
    Status       string   `json:"status"`        // "active" | "coming_soon" | "archived"
}
```

**Invariants** :
- `AvailableTitles` contient **toujours** au moins `halo_infinite` (même si aucun joueur)
- `AvailablePlayers` est résolu pour le `CurrentTitle.Slug` uniquement
- si le titre courant n'a aucun joueur → `AvailablePlayers = []`, `CurrentPlayer = nil` (pas d'erreur)
- `TitleSummary.PlayerCount` = nombre de profils `db_profiles.json` v3 pour ce titre

#### JobMeta (enrichi)

```go
// Avant : map[string]any non structurée
// Après :
type JobMeta struct {
    TitleSlug  string         `json:"title_slug"`   // NEW — obligatoire, rempli à la création
    PlayerSlug string         `json:"player_slug"`
    Extra      map[string]any `json:"extra,omitempty"`
}
```

**Invariant** : un job ne peut jamais être créé avec un `TitleSlug` vide. Le service de création de job doit valider via `TitleRegistry.Exists(slug)` avant persistance.

### Stratégie de switch titre à runtime

Le switch de titre est préparé structurellement mais **sans UI dédiée** (pas de bouton, pas de sélecteur).
Le jour où un sélecteur est ajouté, il n'a qu'à appeler `POST /session/context` avec `{"title_slug": "new_slug"}` et re-bootstrapper.

**Flux complet** (prêt à brancher) :

```
1. Frontend → POST /session/context {"title_slug": "halo_wars"}
2. Backend  → validate title exists in TitleRegistry
3. Backend  → update session.CurrentTitleSlug
4. Backend  → invalidate session.CurrentPlayerSlug = nil
5. Backend  → resolve AvailablePlayers for new title
6. Backend  → return full BootstrapResponse with new title context
7. Backend  → log: title_switched{session_id, from, to, available_players_count}
8. Frontend → flush all stores (appShellStore, globalFilterStore, etc.)
9. Frontend → re-hydrate from BootstrapResponse
10. Frontend → si AvailablePlayers non vide → auto-sélectionner le premier joueur
11. Frontend → si AvailablePlayers vide → afficher état "aucun joueur pour ce titre"
```

**Lazy pool opening** : le pool DuckDB n'ouvre les connexions du nouveau titre qu'à la première requête de données (pas au moment du switch). Le switch de session est donc instantané.

### Logging structuré

Tout le WP2 doit produire des logs exploitables pour le debug du switch titre :

| Événement | Niveau | Attributs slog |
|-----------|--------|----------------|
| Création session | `Info` | `session_id`, `title_slug` (default), `locale` |
| Switch titre | `Info` | `session_id`, `from_title`, `to_title`, `available_players_count` |
| Switch titre refusé | `Warn` | `session_id`, `requested_title`, `reason` ("unknown_title") |
| Restauration session legacy | `Info` | `session_id`, `title_slug` ("halo_infinite" fallback), `legacy=true` |
| Bootstrap servi | `Debug` | `session_id`, `title_slug`, `player_count`, `response_bytes` |
| Job créé | `Info` | `job_id`, `title_slug`, `player_slug`, `job_type` |
| Job titre invalide | `Error` | `job_id`, `title_slug`, `reason` |

### Couche impactée

1. `apps/go-api/internal/domain/session.go` — struct `SessionData` + `SessionContextRequest`
2. `apps/go-api/internal/domain/bootstrap.go` — struct `BootstrapResponse` + `TitleSummary`
3. `apps/go-api/internal/domain/job.go` — struct `JobMeta` (structuré)
4. `apps/go-api/internal/api/middleware/session.go` — désérialisation session avec fallback titre
5. `apps/go-api/internal/api/handlers/session_context.go` — logique switch titre
6. `apps/go-api/internal/service/bootstrap_service.go` — titres disponibles + joueurs par titre
7. `apps/go-api/internal/platform/jobs/` — validation titre à la création de job

### Contraintes

1. un job ne doit jamais pouvoir écrire dans le mauvais namespace de titre ;
2. un cookie/session restauré doit conserver son contexte de titre (fallback `"halo_infinite"` si absent) ;
3. le bootstrap doit être cohérent même si le titre courant n'a aucun joueur provisionné ;
4. un switch de titre **invalide** le joueur courant (sécurité : pas de fuite cross-titre) ;
5. le switch est **synchrone** et retourne le bootstrap complet (pas de polling).

### Preuves attendues

#### Tests unitaires
1. `TestSessionData_DefaultTitle` — nouvelle session → `CurrentTitleSlug == "halo_infinite"`
2. `TestSessionData_LegacyDeserialization` — JSON sans `current_title_slug` → fallback `"halo_infinite"`
3. `TestSessionData_TitleSwitchInvalidatesPlayer` — switch titre → `CurrentPlayerSlug = nil`
4. `TestBootstrapResponse_ContainsAvailableTitles` — bootstrap inclut au moins 1 titre
5. `TestBootstrapResponse_PlayersFilteredByTitle` — joueurs filtrés par titre courant
6. `TestBootstrapResponse_EmptyTitleNoError` — titre sans joueur → réponse valide, pas d'erreur
7. `TestJobMeta_TitleRequired` — création job sans titre → erreur de validation
8. `TestJobMeta_TitleValidated` — titre inconnu → erreur `unknown_title`
9. `TestTitleSummary_PlayerCount` — compte de joueurs cohérent avec `db_profiles.json` v3

#### Tests httptest
10. `TestPostSessionContext_SwitchTitle` — `POST /session/context {"title_slug": "X"}` → 200, nouveau bootstrap, joueur invalidé
11. `TestPostSessionContext_UnknownTitle` — titre inconnu → 422 avec `{"error": "unknown_title"}`
12. `TestPostSessionContext_SameTitleNoOp` — même titre → 200 sans invalidation joueur
13. `TestPostSessionContext_SwitchTitleThenSelectPlayer` — switch + select player dans le même POST
14. `TestGetBootstrap_IncludesTitles` — GET bootstrap → `current_title` + `available_titles` présents
15. `TestGetBootstrap_LegacySession` — session legacy (pas de titre) → bootstrap avec `halo_infinite`

#### Tests d'intégration
16. `TestTitleSwitch_PoolIsolation` — switch titre → les connexions DuckDB du titre précédent restent en pool (pas fermées) mais pas utilisées ; les nouvelles connexions sont ouvertes lazy
17. `TestTitleSwitch_JobIsolation` — un job créé avant le switch reste lié à l'ancien titre
18. `TestTitleSwitch_SessionPersistence` — switch → redémarrage serveur → session restaurée avec le bon titre

#### Tests de logging
19. `TestTitleSwitch_LogsEmitted` — vérifier via `slogtest.Handler` que le switch produit les logs `title_switched` avec les bons attributs
20. `TestLegacySession_LogsFallback` — restauration session legacy → log `legacy=true`

## WP3 — Stockage namespacé et migration legacy

### But

Mettre en place la nouvelle arborescence title-aware, sans casser Halo Infinite existant et avec une migration opérable.

### Livrables

1. nouvelle cible de stockage `data/titles/{title_slug}/...` ;
2. outil de migration (sous-commande CLI Go) avec 3 modes : `dry-run`, `apply`, `rollback` ;
3. **manifest JSON** (`operations.json`) traçant chaque opération `(source, dest)` effectuée — rollback = exécution inverse du manifest ;
4. backup automatique avant `apply` (copie des fichiers source dans un répertoire horodaté) ;
5. journal de migration et détection d'idempotence (dépôt déjà migré → no-op) ;
6. stratégie explicite sur les DB déjà migrées.

> **Note Windows** : pas de symlinks (problématiques sur Windows). La migration déplace physiquement les fichiers.
> Prévoir la gestion des locks DuckDB (fermer les connexions avant déplacement) et des chemins longs.

### Couche impactée

1. `apps/go-api/internal/platform/duckdb/`
2. scripts / commandes d'exploitation Go liés au stockage
3. sous-commande CLI dédiée (ex : `levelup migrate-titles`)

### Non-objectif explicite

Le Sprint 44 ne doit pas rendre `title_slug` premier citoyen dans toutes les tables et PK DuckDB existantes.
Le namespace par titre est précisément choisi pour éviter cette refonte massive.

### Preuves attendues

1. test d'intégration sur dépôt legacy Halo Infinite ;
2. test d'intégration sur dépôt déjà namespacé ;
3. test de rollback (apply puis rollback → état initial retrouvé) ;
4. test d'idempotence (apply deux fois → même état) ;
5. golden diff Halo Infinite pré/post migration = 0.

## WP4 — Contrats API, frontend, CLI et corpus de validation

### But

Faire converger le contrat produit, les outils CLI et les jeux de test vers un runtime title-aware vérifié, sans drift caché.

### Décision architecturale : routage title_slug dans les URLs

**23 endpoints** utilisent `{player_slug}` comme paramètre de chemin.
Le choix d'intégration de `{title_slug}` doit être fixé ici :

| Option | Pattern | Avantages | Inconvénients |
|--------|---------|-----------|---------------|
| **A. Préfixe path** | `/{title_slug}/players/{player_slug}/...` | Explicite, cacheable, RESTful | Change toutes les URLs, breaking change frontend |
| **B. Header** | `X-LevelUp-Title: halo_infinite` | Aucun change de routes | Moins explicite, pas cacheable par path |
| **C. Session implicite** | Titre déduit du cookie session | Zéro changement d'URL | Implicite, interdit le multi-titre dans le même call |

> **Recommandation** : option **A** (préfixe path) pour la clarté et la cacheabilité.
> Avec un middleware Chi qui extrait `{title_slug}` du path et l'injecte dans le context,
> les handlers n'ont pas besoin de changer leur signature.
> La transition peut être graduelle : servir les anciennes routes avec fallback `halo_infinite`.

### Livrables

1. DTOs et OpenAPI exposant le titre courant (23 endpoints + préfixe `{title_slug}`) ;
2. middleware Chi extractant `{title_slug}` du path et fallback `halo_infinite` pour les anciennes routes ;
3. frontend réellement réaligné : routes TanStack, `routeTree.gen.ts`, helpers de navigation, `queryKeys`, hooks `features/*/queries.ts`, handlers MSW/Playwright et codegen OpenAPI title-aware ;
4. types frontend et stores alignés avec le bootstrap title-aware (voir contrat frontend ci-dessous) ;
5. golden values Halo Infinite namespacées ;
6. **corpus synthétique d'un second titre** (~0.5–1 jour dédié) :
   - `metadata.duckdb` minimal avec des données distinctes ;
   - `shared_matches_v2.duckdb` avec au moins quelques matchs ;
   - schémas compatibles avec la structure existante ;
7. smoke E2E React sur changement de titre et non-régression bootstrap ;
8. commandes ops concernées du binaire `levelup` : ajouter `--title` (défaut `halo_infinite`) à `backup`, `restore`, `archive`, `index-media`, `seed`, `healthcheck`, `gate-check` ;
9. résolution de titre au démarrage du binaire `server` et dans les helpers de provisioning joueur ;
10. `settingsDraftStore.localUiPrefs.lastPlayerSlug` devient title-aware (clé composite ou map par titre), le reste des settings restant global.

### Périmètre frontend réel

Le coût frontend du Sprint 44 ne se limite pas à `appShellStore`.

Les surfaces à réaligner explicitement sont :

1. `apps/web/src/routes/**` et le layout `/players/$playerSlug` ;
2. `apps/web/src/routeTree.gen.ts` (régénéré, pas patché à la main, mais impacté) ;
3. `apps/web/src/lib/query/keys.ts` ;
4. `apps/web/src/features/*/queries.ts` et les helpers de navigation interne ;
5. `apps/web/src/components/shell/NavBar.tsx`, `KPIBar.tsx` et les liens contextuels ;
6. `apps/web/src/test/handlers.ts`, MSW et Playwright.

### Contrat frontend cible

#### Types TypeScript (générés depuis OpenAPI)

```ts
// types.ts — généré depuis openapi.yaml
export interface TitleSummary {
  slug: string           // "halo_infinite"
  name: string           // "Halo Infinite"
  icon_url: string       // optionnel, "" si non disponible
  player_count: number   // nombre de joueurs provisionnés
  status: 'active' | 'coming_soon' | 'archived'
}

export interface BootstrapResponse {
  current_title: TitleSummary
  available_titles: TitleSummary[]
  current_player: PlayerSummary | null
  available_players: PlayerSummary[]
  feature_flags: FeatureFlags
  capabilities: CapabilityMap
  // ... champs existants
}

export interface SessionContextRequest {
  player_slug?: string
  title_slug?: string    // NEW — optionnel, absent = pas de changement
  locale?: string
}
```

#### `appShellStore.ts` — Enrichissement

```ts
interface AppShellState {
  // Nouveaux champs
  currentTitleSlug: string              // "halo_infinite" — défaut
  availableTitles: TitleSummary[]       // depuis bootstrap
  isTitleSwitching: boolean             // true pendant le PATCH + re-bootstrap

  // Champs existants inchangés
  currentPlayer: PlayerSummary | null
  availablePlayers: PlayerSummary[]
  locale: 'fr' | 'en'
  // ...
}

const useAppShellStore = create<AppShellState & AppShellActions>((set, get) => ({
  currentTitleSlug: 'halo_infinite',
  availableTitles: [],
  isTitleSwitching: false,

  // --- Hydratation depuis bootstrap (enrichie) ---
  hydrateFromBootstrap: (resp: BootstrapResponse) => set({
    currentTitleSlug: resp.current_title.slug,
    availableTitles: resp.available_titles,
    currentPlayer: resp.current_player,
    availablePlayers: resp.available_players,
    isTitleSwitching: false,
  }),

  // --- Switch titre (prêt à brancher sur un bouton) ---
  switchTitle: async (newSlug: string) => {
    const { currentTitleSlug } = get()
    if (newSlug === currentTitleSlug) return  // no-op

    set({ isTitleSwitching: true })
    try {
      // 1. POST /session/context avec le nouveau titre
      const resp = await postSessionContext({ title_slug: newSlug })

      // 2. Flush des stores dépendants
      useGlobalFilterStore.getState().reset()
      useCareerPageStore.getState().reset()

      // 3. Re-hydratation complète depuis la réponse
      get().hydrateFromBootstrap(resp)

      console.info(`[title-switch] ${currentTitleSlug} → ${newSlug}`,
        { players: resp.available_players.length })
    } catch (err) {
      console.error('[title-switch] failed', err)
      set({ isTitleSwitching: false })
      // Rester sur l'ancien titre — pas de demi-état
    }
  },
}))
```

**Points clés** :
- `switchTitle()` est une **action interne du store**, pas exposée en UI pour l'instant
- le switch est **atomique** côté frontend : flush stores → re-hydratation, pas d'état intermédiaire visible
- `isTitleSwitching` permet d'afficher un loader si un sélecteur est ajouté plus tard
- en cas d'erreur → rollback silencieux sur l'ancien titre (pas de demi-état)
- les stores dépendants (`globalFilterStore`, `careerPageStore`, etc.) exposent un `reset()` flush

#### Autres stores impactés

| Store | Changement | Raison |
|-------|------------|--------|
| `globalFilterStore` | Ajouter `reset()` pour flush au switch titre | Les filtres (maps, modes, dates) sont titre-spécifiques |
| `careerPageStore` | Ajouter `reset()` | Données de progression spécifiques au titre |
| `setupFlowStore` | Pas de changement de modèle | Le setup flow reste session-scoped, mais doit tolérer un titre sans joueur courant |
| `settingsDraftStore` | Rendre `lastPlayerSlug` title-aware | La préférence locale ne doit pas restaurer un joueur du mauvais titre |
| `routes/**` + `routeTree.gen.ts` | Ajouter `titleSlug` au contrat de navigation | Toutes les routes player-scoped changent de forme |
| `queryKeys` + `features/*/queries.ts` | Inclure le titre courant | Les caches React Query et les appels API ne doivent pas fuir entre titres |

#### API client (`lib/api/client.ts`)

```ts
// Construction d'URL title-aware
const buildUrl = (path: string): string => {
  const titleSlug = useAppShellStore.getState().currentTitleSlug
  return `${BASE_URL}/${titleSlug}${path}`
}

// Exemple : GET /api/v1/halo_infinite/players/{slug}/pages/overview
// Fallback : si le serveur reçoit /api/v1/players/{slug}/... → route legacy avec halo_infinite
```

### Logging frontend

| Événement | Niveau | Contexte |
|-----------|--------|----------|
| Switch titre initié | `console.info` | `[title-switch] {from} → {to}` |
| Switch titre réussi | `console.info` | `[title-switch] {from} → {to}, {n} players` |
| Switch titre échoué | `console.error` | `[title-switch] failed`, erreur complète |
| Bootstrap hydraté | `console.debug` | `[bootstrap] title={slug}, players={n}` |
| Re-hydratation post-switch | `console.debug` | `[bootstrap] re-hydrated after title switch` |

### Couche impactée

1. `apps/go-api/api/openapi.yaml` (23 endpoints + préfixe + `TitleSummary` schema)
2. `apps/go-api/internal/api/handlers/` (22 fichiers handlers)
3. `apps/go-api/internal/api/middleware/` (nouveau middleware title extractor)
4. `apps/go-api/cmd/levelup/main.go` (commandes ops concernées par `--title`)
5. `apps/web/src/lib/api/types.ts` (types générés : `TitleSummary`, `BootstrapResponse` enrichi, `SessionContextRequest` enrichi)
6. `apps/web/src/lib/api/client.ts` (construction URL title-aware)
7. `apps/web/src/stores/appShellStore.ts` (`currentTitleSlug`, `availableTitles`, `isTitleSwitching`, `switchTitle()`, `hydrateFromBootstrap()` enrichi)
8. `apps/web/src/stores/globalFilterStore.ts` (ajout `reset()`)
9. `apps/web/src/stores/careerPageStore.ts` (ajout `reset()`)
10. `apps/web/src/stores/settingsDraftStore.ts` (`lastPlayerSlug` title-aware)
11. `apps/web/src/lib/query/keys.ts`, `apps/web/src/features/*/queries.ts`
12. `apps/web/src/routes/**`, `apps/web/src/routeTree.gen.ts`, `apps/web/src/components/shell/NavBar.tsx`
13. fixtures Go / React / Playwright / parity checks

### Preuves attendues

#### Tests backend
1. golden tests Halo Infinite après migration ;
2. tests d'isolement inter-titres sur bootstrap, jobs et accès aux données ;
3. le corpus synthétique produit des réponses distinctes de Halo Infinite (pas de fuite) ;
4. les anciennes URL (sans préfixe titre) continuent de fonctionner avec fallback `halo_infinite` ;
5. `TestMiddleware_TitleExtractor_ValidSlug` — middleware parse `{title_slug}` et l'injecte dans le context ;
6. `TestMiddleware_TitleExtractor_MissingSlug` — route legacy → fallback `halo_infinite` ;
7. `TestMiddleware_TitleExtractor_UnknownSlug` — titre inconnu → 404 avec body `{"error": "unknown_title"}` ;
8. `TestCLI_TitleFlag` — chaque sous-commande accepte `--title` et le propage à `PathResolver`.

#### Tests frontend
9. `appShellStore.test.ts::hydrateFromBootstrap_setsTitle` — bootstrap → `currentTitleSlug` mis à jour ;
10. `appShellStore.test.ts::switchTitle_flushesStores` — switch → `globalFilterStore.reset()` et `careerPageStore.reset()` appelés ;
11. `appShellStore.test.ts::switchTitle_sameTitle_noop` — même titre → aucun appel API ;
12. `appShellStore.test.ts::switchTitle_error_rollback` — erreur PATCH → titre inchangé, `isTitleSwitching = false` ;
13. `appShellStore.test.ts::switchTitle_setsLoading` — pendant le switch → `isTitleSwitching = true` ;
14. `settingsDraftStore.test.ts::lastPlayerSlug_scopedByTitle` — la préférence locale ne fuit pas entre titres ;
15. `client.test.ts::buildUrl_includesTitleSlug` — URL construite avec le titre courant.

#### Smoke E2E (Playwright)
16. `title-switch.spec.ts` — scénario complet :
    a. bootstrap initial → titre = `halo_infinite` ;
    b. appeler `switchTitle('synthetic_title')` via console ou hook test ;
    c. vérifier que le bootstrap est re-servi avec le titre synthétique ;
    d. vérifier que les joueurs affichés sont ceux du titre synthétique (pas Halo Infinite) ;
    e. vérifier que les filtres sont réinitialisés ;
    f. switch back vers `halo_infinite` → données Halo Infinite restaurées.
17. `legacy-routes.spec.ts` — navigation avec anciennes URLs sans préfixe → fallback fonctionnel.

## WP5 — Observabilité, CI, exploitation

### But

Fermer le lot en garantissant que le multi-titres est observable, déployable et réversible.

### Livrables

1. logs contenant le titre courant et les `response_bytes` ;
2. validation de contrat en dev mode pour le bootstrap et les réponses title-aware ;
3. CI enrichie si nécessaire pour les fixtures multi-titres ;
4. runbook opérateur, procédure de rollback et checklist d'exploitation mises à jour.

### Couche impactée

1. middleware de logging / observabilité
2. workflows CI pertinents
3. documentation ops et docs de migration

### Preuves attendues

1. logs de divergence ou de taille de réponse disponibles en dev ;
2. CI verte avec les nouveaux corpus ;
3. runbook testable par un développeur externe au lot.

## Definition of Done du Sprint 44

Le sprint n'est pas considéré terminé si un seul des points suivants manque :

1. `halo_infinite` ne passe pas sans régression sur le corpus golden ;
2. la migration n'est pas réversible (manifest JSON + rollback testé) ;
3. l'isolement inter-titres n'est pas prouvé (deux titres, même gamertag ≠ mêmes données) ;
4. les handlers/bootstrap title-aware n'ont pas de couverture dédiée ;
5. le mécanisme de switch titre n'est pas fonctionnel (`POST /session/context` + re-bootstrap) ;
6. le runbook de migration et rollback n'existe pas ;
7. la couverture des modules touchés par le Sprint 44 est < 80% ;
8. `golangci-lint run` n'est pas clean.

## Checklist de clôture

### WP0 — Noyau de design
- [ ] Registre de titres centralisé (`TitleRegistry` / `TitleDescriptor`)
- [ ] Résolveur de chemins title-aware centralisé (`PathResolver`) couvrant les 29 refs hardcodées

### WP1 — Config + PlayerResolver
- [ ] `PlayerResolver` refactoré pour `(title_slug, player_slug)` (mode réel + mode démo)
- [ ] Pool DuckDB clé `{title}:{gamertag}` (13 fichiers `*_repo.go` vérifiés)
- [ ] Matrice des chemins globaux vs title-aware figée et encodée dans `PathResolver`
- [ ] `db_profiles.json` v3 title-aware avec rétrocompatibilité lecture
- [ ] `POST /setup/players` + `GET /players` title-aware et cohérents avec le titre courant
- [ ] Demo mode title-aware (`DemoFixturesDir` namespacé)
- [ ] 6 fichiers `internal/ops/` passent par `PathResolver`
- [ ] `internal/validation/gate.go` et `internal/sync/engine.go` passent par `PathResolver`

### WP2 — Session, jobs, bootstrap, switch titre
- [ ] `SessionData.CurrentTitleSlug` non-nul (défaut `"halo_infinite"`)
- [ ] `SessionContextRequest` accepte `title_slug` optionnel
- [ ] `POST /session/context` avec `title_slug` → switch titre + invalidation joueur + re-bootstrap
- [ ] `POST /session/context` avec titre inconnu → 422 `unknown_title`
- [ ] Bootstrap enrichi : `current_title` + `available_titles` (type `TitleSummary`)
- [ ] Bootstrap sans joueur pour le titre courant → réponse valide, pas d'erreur
- [ ] `JobMeta` structuré avec `TitleSlug` obligatoire, validé via `TitleRegistry`
- [ ] Sessions legacy (sans `current_title_slug`) → fallback `"halo_infinite"` à la désérialisation
- [ ] Logging structuré : `title_switched`, `legacy_session`, `bootstrap_served`, `job_created` avec attributs slog
- [ ] 20 tests WP2 (9 unitaires + 6 httptest + 3 intégration + 2 logging)

### WP3 — Stockage + migration
- [ ] Namespace de stockage branché
- [ ] Migration `dry-run / apply / rollback` via manifest JSON, testée et idempotente

### WP4 — API + frontend + CLI
- [ ] Routage OpenAPI `{title_slug}` décidé et implémenté (23 endpoints)
- [ ] Middleware Chi title extractor + fallback `halo_infinite`
- [ ] Commandes ops concernées du binaire `levelup` acceptent `--title` et le binaire `server` résout le titre au démarrage
- [ ] `appShellStore.currentTitleSlug` + `availableTitles` + `isTitleSwitching` branchés
- [ ] `appShellStore.switchTitle()` implémenté (`POST /session/context` + flush stores + re-hydratation)
- [ ] `globalFilterStore.reset()` et `careerPageStore.reset()` ajoutés
- [ ] `settingsDraftStore.lastPlayerSlug` devient title-aware
- [ ] Routes TanStack, `routeTree.gen.ts`, `queryKeys`, hooks `features/*/queries.ts` et liens de navigation sont title-aware
- [ ] API client `buildUrl()` title-aware
- [ ] Types TS générés : `TitleSummary`, `BootstrapResponse` enrichi, `SessionContextRequest` enrichi
- [ ] 8 tests backend (middleware, CLI, fallback) + 7 tests frontend (store, prefs locales, client) + 2 E2E Playwright

### WP5 — Observabilité + CI
- [ ] Logs `title_slug` + `response_bytes` en place
- [ ] `golangci-lint run` clean, 0 TODO non-documenté
- [ ] Runbook et ADR alignés avec l'implémentation

### Validation transverse
- [ ] Halo Infinite sans régression (golden diff = 0)
- [ ] Corpus synthétique second titre utilisé en validation
- [ ] Couverture ciblée modules Sprint 44 ≥ 80%, couverture Go globale ≥ 50%
- [ ] Documentation à jour
