# Axe 2 · Review ChatGPT — Architecture & qualité code

## Métadonnées du passage

| Champ | Valeur |
|-------|--------|
| Auteur LLM | ChatGPT |
| Date du passage | `2026-04-18` |
| SHA Go | `93c3cd66` |
| SHA React | `93c3cd66` |
| Durée de l'analyse | `session courante` |

## Synthèse exécutive (150 mots max)

L'architecture Go est globalement saine sur cette branche : couches lisibles, routeur fin, configuration des feature flags centralisée et fort usage de l'injection de dépendances. Les dérives confirmées sont ciblées plutôt que systémiques. La plus nette est dans `apps/go-api/internal/service/fanout_service.go`, où la couche service importe encore directement `internal/platform/duckdb`, ce qui viole la règle hexagonale documentée dans l'axe 2.

Côté React, le socle est moderne et proprement structuré autour de Vite, TanStack Router, React Query, Zustand et un client API central. Les principaux écarts relèvent surtout de la discipline de découpage et d'isolation des features : plusieurs pages UI dépassent nettement les seuils souhaités, `SettingsPage` dépend encore des queries du domaine `setup`, et aucune error boundary explicite n'a été trouvée dans `src/`.

---

## A. Architecture hexagonale — Go (backend)

### A.1 Identification des couches

| Couche | Package attendu | Package réel | Conforme ? | Commentaire |
|--------|-----------------|--------------|:----------:|-------------|
| Domaine (entités pures) | `internal/domain/` | `apps/go-api/internal/domain/` | ✅ | Couche présente et distincte. |
| Ports (interfaces) | `internal/port/` | `apps/go-api/internal/port/` | ✅ | Contrats centralisés. |
| Services (orchestration) | `internal/service/` | `apps/go-api/internal/service/` | ✅ | Présent, avec une violation ciblée relevée ci-dessous. |
| Adapters entrants (HTTP) | `internal/api/handlers/` | `apps/go-api/internal/api/handlers/` | ✅ | Handlers séparés par surface. |
| Adapters sortants (DB) | `internal/platform/duckdb/` | `apps/go-api/internal/platform/duckdb/` | ✅ | Couche clairement isolée. |
| Adapters sortants (API externe) | `internal/platform/auth/`, etc. | `apps/go-api/internal/platform/auth/`, `internal/notify/` | ✅ | Couche présente. |
| Analyse (algos purs) | `internal/analysis/` | `apps/go-api/internal/analysis/` | ✅ | Présente et distincte. |

### A.2 Respect des dépendances (règle : domaine ne dépend de rien, ports ne dépendent que du domaine)

| Règle | Violations identifiées (fichier:ligne) | Classif |
|-------|----------------------------------------|:-------:|
| `internal/domain/` n'importe ni platform ni service ni api | Aucune confirmée dans cette passe. | 🟢 |
| `internal/port/` n'importe que `internal/domain/` | Aucune confirmée dans cette passe. | 🟢 |
| `internal/analysis/` est sans I/O (pas d'import `database/sql`, `net/http`, `os`) | Aucune confirmée dans cette passe. | 🟢 |
| `internal/service/` dépend des ports, pas des implémentations platform | `apps/go-api/internal/service/fanout_service.go:17` importe `levelup/go-api/internal/platform/duckdb`, puis manipule `*duckdb.PlayerDB` dans plusieurs helpers. | 🟠 |
| `internal/api/handlers/` ne touche pas `internal/platform/duckdb/` directement | Aucune confirmée dans cette passe. | 🟢 |

### A.3 Qualité des abstractions

| Question | Réponse + fichier:ligne | Classif |
|----------|-------------------------|:-------:|
| `port.Services` : interface bien segmentée ou fourre-tout ? | Pattern monolithique `port.Services` non observé dans cette passe; le câblage passe plutôt par `NewServiceRegistry` dans `apps/go-api/internal/api/server.go:69`. | 🟢 |
| Les repos DuckDB exposent-ils une interface côté consommateur ? | Oui côté frontières applicatives, sauf `FanoutService` qui retombe sur un type concret `*duckdb.PlayerDB`. | 🟠 |
| Les DTOs HTTP sont-ils distincts des entités domaine ? | Souvent non : les handlers sérialisent directement des read models du package `domain`, par ex. `apps/go-api/internal/api/handlers/session_context.go:66`. | 🟡 |
| Y a-t-il des `interface{}` / `any` non justifiés ? | Usages confirmés surtout aux frontières dynamiques (`apps/go-api/internal/domain/job.go:70-72`, `apps/go-api/internal/config/config.go:51`, YAML/JSON, provider auth). Aucun abus métier massif confirmé. | 🟡 |

---

## B. Architecture — React (frontend)

### B.1 Structure par feature

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| `src/features/<domaine>/` isolé (pas de fuite cross-feature) | Une fuite confirmée : `apps/web/src/features/settings/SettingsPage.tsx:9` importe `useSettings` et `useUpdateSettings` depuis `@/features/setup/queries`. | 🟡 |
| Routes (`src/routes/`) découplées des features | Oui. Les fichiers de `apps/web/src/routes/` se contentent principalement d'importer une page et de l'exposer. | 🟢 |
| Composants UI (`src/components/ui/`) purs présentation | Aucun fetch direct confirmé dans `components/ui/` lors de cette passe. | 🟢 |
| Hooks dédiés par feature (data fetching, state local) | Globalement oui, avec la réserve `settings` -> `setup/queries`. | 🟡 |
| Séparation fetching (React Query ?) / rendering | Oui. Le seul `fetch` confirmé hors tests reste dans `apps/web/src/lib/api/client.ts:55`; pas de `fetch(` dispersé dans les features. | 🟢 |

### B.2 Couches d'abstraction

| Couche | Présente ? | Commentaire |
|--------|:----------:|-------------|
| Client API typé (généré depuis OpenAPI ?) | ✅ | `apps/web/src/lib/api/client.ts` + `generated.ts` / `types.ts`. |
| Couche état global (Zustand/Redux/Context) | ✅ | Zustand pour l'état applicatif; React Query pour l'état serveur. |
| Couche i18n | ⚪ | Non auditée en profondeur dans cette passe. |
| Error boundaries / fallback UI | ⚪ | Aucune `ErrorBoundary` explicite trouvée par grep dans `apps/web/src`. |

---

## C. Duplications & factorisation

### C.1 Duplications Go détectées

| Pattern dupliqué | Occurrences (fichier:ligne) | Suggestion | Classif |
|------------------|-----------------------------|------------|:-------:|
| Aucune duplication métier certaine n'a été isolée dans cette passe | — | Ne pas sur-interpréter sans grep structurel dédié. | 🟢 |

### C.2 Duplications React détectées

| Pattern dupliqué | Occurrences | Suggestion | Classif |
|------------------|-------------|------------|:-------:|
| Aucune duplication métier certaine n'a été isolée dans cette passe | — | Rester factuel; le principal sujet est plutôt la taille de quelques pages. | 🟢 |

### C.3 Modules trop gros (>500L Go / >300L React)

| Fichier | Lignes | Responsabilités multiples ? | Suggestion découpage | Classif |
|---------|-------:|:---------------------------:|----------------------|:-------:|
| `apps/web/src/features/media/MediaPage.tsx` | 583 | oui | Split en sous-vues / toolbar / pagination / cards | 🟠 |
| `apps/web/src/features/setup/SetupPage.tsx` | 467 | oui | Split par étape d'onboarding / state machine UI | 🟠 |
| `apps/web/src/features/match-history/MatchHistoryTable.tsx` | 434 | oui | Isoler toolbar, table, empty state, pagination | 🟡 |
| `apps/web/src/features/session-compare/SessionComparePage.tsx` | 333 | oui | Extraire blocs de comparaison et états vides | 🟡 |
| `apps/web/src/features/squad/SquadPage.tsx` | 328 | oui | Extraire sections charts / filters / KPIs | 🟡 |
| `apps/web/src/features/match-view/MatchViewPage.tsx` | 309 | oui | Extraire layout global / loaders / tabs shell | 🟡 |
| `apps/go-api/internal/api/gen/types.gen.go` | 1125 | non | Code généré, toléré | 🟢 |

### C.4 Fonctions trop longues (>80L)

| Fichier:fonction | Lignes | Justifié par `//nolint` ? | Suggestion | Classif |
|------------------|-------:|:-------------------------:|------------|:-------:|
| `non mesuré de manière fiable dans cette passe` | — | — | Audit dédié nécessaire si on veut être exact. | 🟢 |

---

## D. Workarounds & fallbacks non pertinents

| Emplacement (fichier:ligne) | Description du workaround | Raison (connue/inconnue) | Toujours nécessaire ? | Classif |
|-----------------------------|---------------------------|--------------------------|:---------------------:|:-------:|
| `apps/go-api/internal/api/handlers/settings.go:105` | Réindexation média toujours stubée (`TODO Sprint 19`) | connue | probablement non à terme | 🟡 |
| `apps/go-api/internal/platform/halo/provider.go:116` et `:124` | TODO live call `spartan_token` | connue | à confirmer selon périmètre réel de provider | 🟡 |
| `apps/go-api/internal/domain/job.go:70-72` | `JobMeta.Extra map[string]any` conservé pour rétrocompatibilité | connue | oui, à court terme | 🟢 |

### Fallbacks potentiellement dangereux

| Emplacement | Comportement observé | Risque | Classif |
|-------------|----------------------|--------|:-------:|
| `apps/go-api/internal/api/server.go:202` | `/directory/gamertags/search` n'est enregistré que si `gamertagSvc != nil` | contrat potentiellement absent en environnement dégradé | 🟡 |
| `apps/go-api/internal/api/middleware/title.go:40-41` | fallback titre session -> `halo_infinite` | comportement voulu, faible risque | 🟢 |

---

## E. Anti-patterns CLAUDE.md revisités (adaptés Go/React)

| Anti-pattern | Version Go/React | Présent ? | Fichier:ligne | Classif |
|--------------|------------------|:---------:|---------------|:-------:|
| Dead code museum | Aucune preuve confirmée dans cette passe | non confirmé | — | 🟢 |
| God file | Quelques pages React dépassent largement 300-400L | oui | `MediaPage.tsx`, `SetupPage.tsx`, `MatchHistoryTable.tsx` | 🟠 |
| Swiss-army function | Un handler cumulant init + call + transform + write n'a pas été isolé clairement | non confirmé | — | 🟢 |
| Magic number | Audit dédié non mené dans cette passe | non vérifié | — | 🟢 |
| Bare connect | Aucune occurrence confirmée dans cette passe | non confirmé | — | 🟢 |
| `map[string]any` / `interface{}` non justifié | Présent surtout aux frontières dynamiques | partiel | `internal/config/config.go`, `internal/domain/job.go`, YAML/JSON | 🟡 |
| Shadow / legacy guard sans date | Feature flags vivants, pas de flag mort confirmé | non confirmé | `internal/config/feature_flags.go` semble sain | 🟢 |

---

## F. Cohérence de nommage & conventions

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| Packages Go respectent `golint` naming | Oui dans cette passe. | 🟢 |
| Handlers Go : un fichier par endpoint racine ? | Globalement oui, avec quelques fichiers auxiliaires de tests. | 🟢 |
| React : convention PascalCase pour composants, camelCase pour hooks | Oui dans les fichiers inspectés. | 🟢 |
| Types TypeScript : `types.ts` ou colocalisés ? Cohérent ? | Oui, mais `generated.ts` / `types.ts` restent très volumineux, ce qui est toléré car généré/contrat. | 🟢 |

---

## G. Configuration & feature flags

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| Feature flags Go (`internal/config/feature_flags.go`) — flags actifs vs morts | Les 12 surfaces existent encore et sont centralisées via `surfaceFields()`. Aucun flag orphelin confirmé. | 🟢 |
| Variables d'environnement documentées ? | Non revérifié en profondeur dans cette passe. | ⚪ |
| Config React (env Vite) — cohérente avec backend ? | Oui sur le point clé `VITE_API_BASE_URL` / proxy dev dans `apps/web/vite.config.ts:17-24`. | 🟢 |

---

## H. Récap classifications

| Niveau | Nombre d'items |
|--------|:--------------:|
| 🔴 Bloquant | 0 |
| 🟠 Majeur | 2 |
| 🟡 Mineur | 6 |
| 🟢 Toléré | 12 |

## I. Top 5 dettes techniques prioritaires

| # | Dette | Fichier:ligne | Effort (S/M/L) | Impact |
|--:|-------|---------------|:--------------:|--------|
| 1 | Service `fanout` dépend d'un type concret DuckDB | `apps/go-api/internal/service/fanout_service.go:17` | M | Violation hexagonale claire |
| 2 | `MediaPage` concentre trop de responsabilités UI | `apps/web/src/features/media/MediaPage.tsx` | M | Dette de maintenance et de testabilité |
| 3 | `SetupPage` concentre toute la machine d'onboarding | `apps/web/src/features/setup/SetupPage.tsx` | M | Dette de maintenance et de testabilité |
| 4 | `settings` dépend encore du domaine `setup` pour ses queries | `apps/web/src/features/settings/SettingsPage.tsx:9` | S | Fuite cross-feature |
| 5 | Réindexation média toujours stubée | `apps/go-api/internal/api/handlers/settings.go:105` | S | Fonction destructive encore incomplète |

## J. Observations libres

Le point fort principal de cette branche est la cohérence du socle : routeur fin, client API central côté React, feature flags Go centralisés, et peu de signes de bricolage global. Les vrais écarts restent localisés. Cela plaide pour une base saine, mais pas totalement nettoyée. Si l'objectif de Sprint 50 est une clôture propre, il faut surtout traiter la violation hexagonale du `FanoutService` et réduire 2 à 3 god files React. Le reste ressemble davantage à de la dette résiduelle qu'à une architecture réellement instable.

---

**Fin du template axe 2.**
