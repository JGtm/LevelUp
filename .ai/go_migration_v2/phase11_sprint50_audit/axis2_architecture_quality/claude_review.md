# Axe 2 — Review Claude — Architecture & qualité code

## Métadonnées du passage

| Champ | Valeur |
|-------|--------|
| Auteur LLM | **Claude** |
| Date du passage | 2026-04-18 |
| SHA Go | `93c3cd66` |
| SHA React | `93c3cd66` |
| Durée de l'analyse | ~1.5h |

## Synthèse exécutive

L'architecture hexagonale Go est **strictement respectée** : aucune violation de dépendance entre couches détectée. Les packages `domain/`, `port/`, `analysis/` sont purs. Le seul fichier Go non-généré > 500L est `migration/steps_shared.go` (596L). Il y a **27 fonctions > 80L**, concentrées dans sync/migration/analysis — dette structurelle acceptable pour du code de migration/init. Le code React est bien structuré par feature avec 1 seul cross-feature import (settings → setup). 3 fichiers React dépassent 400L et méritent un découpage. Les usages de `interface{}`/`any` sont majoritairement justifiés (JSON dynamique, Plotly charts) avec ~8 cas discutables dans les DTOs domaine.

---

## A. Architecture hexagonale — Go (backend)

### A.1 Identification des couches

| Couche | Package attendu | Package réel | Conforme ? | Commentaire |
|--------|-----------------|--------------|:----------:|-------------|
| Domaine (entités pures) | `internal/domain/` | `internal/domain/` (2 970L, 28 fichiers) | ✅ | Entités, enums, DTOs, chart models, title registry |
| Ports (interfaces) | `internal/port/` | `internal/port/` (536L, 2 fichiers) | ✅ | `repository.go` + `services.go` — interfaces bien segmentées |
| Services (orchestration) | `internal/service/` | `internal/service/` (4 179L, 19 fichiers) | ✅ | Orchestration repo → domain, 1 service par domaine |
| Adapters entrants (HTTP) | `internal/api/handlers/` | `internal/api/handlers/` (2 180L, 27 fichiers) | ✅ | 1 handler par resource, helpers centralisés |
| Adapters sortants (DB) | `internal/platform/duckdb/` | `internal/platform/duckdb/` (2 748L, 21 fichiers) | ✅ | Repos DuckDB, pool singleflight |
| Adapters sortants (API ext.) | `internal/platform/auth/`, `platform/halo/` | Présents | ✅ | Auth MSAL, exchange Halo, provider Halo |
| Analyse (algos purs) | `internal/analysis/` | `internal/analysis/` (4 467L, 18 fichiers) | ✅ | Sans I/O — citations, sessions, squad, weapons, KV, skill, performance |

### A.2 Respect des dépendances

| Règle | Violations identifiées | Classif |
|-------|------------------------|:-------:|
| `internal/domain/` n'importe ni platform ni service ni api | **0 violation** | 🟢 |
| `internal/port/` n'importe que `internal/domain/` | **0 violation** | 🟢 |
| `internal/analysis/` est sans I/O (pas d'import `database/sql`, `net/http`, `os`) | **0 violation** | 🟢 |
| `internal/service/` dépend des ports, pas des implémentations platform | **0 violation** | 🟢 |
| `internal/api/handlers/` ne touche pas `internal/platform/duckdb/` directement | **0 violation** | 🟢 |

**Architecture hexagonale parfaitement propre.**

### A.3 Qualité des abstractions

| Question | Réponse | Classif |
|----------|---------|:-------:|
| `port.Services` : interface bien segmentée ou fourre-tout ? | `port/services.go` expose des interfaces granulaires par domaine (BootstrapService, CareerService, etc.) — bien segmenté | 🟢 |
| Les repos DuckDB exposent-ils une interface côté consommateur ? | Oui via `port/repository.go` — les handlers ne voient que les interfaces | 🟢 |
| Les DTOs HTTP sont-ils distincts des entités domaine ? | `internal/api/gen/types.gen.go` (1 125L) contient les types OpenAPI distincts, mais les handlers utilisent souvent directement `domain.*` pour les réponses JSON — couplage partiel | 🟡 |
| Y a-t-il des `interface{}` / `any` non justifiés ? | ~8 cas discutables dans domain (Plotly fields `*interface{}`, charts `[]interface{}`, tables `[]map[string]interface{}`) — le reste est justifié (JSON API Halo, job results) | 🟡 |

---

## B. Architecture — React (frontend)

### B.1 Structure par feature

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| `src/features/<domaine>/` isolé (pas de fuite cross-feature) | **1 violation** : `features/settings/SettingsPage.tsx:9` importe `useSettings, useUpdateSettings` depuis `features/setup/queries` | 🟡 |
| Routes (`src/routes/`) découplées des features | Routes TanStack auto-générées via plugin Vite — découplage correct | 🟢 |
| Composants UI (`src/components/ui/`) pur présentation | 14 composants UI sans logique fetch — conforme | 🟢 |
| Hooks dédiés par feature (data fetching, state local) | Chaque feature a son `queries.ts` avec hooks TanStack Query dédiés | 🟢 |
| Séparation fetching / rendering | Via TanStack Query + composants de rendu — propre | 🟢 |

### B.2 Couches d'abstraction

| Couche | Présente ? | Commentaire |
|--------|:----------:|-------------|
| Client API typé | ✅ | `lib/api/client.ts` (fetch wrapper) + `types.ts` (1 186L manuels) + `generated.ts` (3 231L auto) |
| Couche état global | ✅ | Zustand v5 (5 stores) — pas de Redux, architecture propre |
| Couche i18n | ❌ | Locale stockée dans `appShellStore` mais **aucun framework i18n** (react-i18next, etc.) | 🟠 |
| Error boundaries / fallback UI | ❌ | **Aucun ErrorBoundary React** — crash composant = page blanche | 🟠 |

---

## C. Duplications & factorisation

### C.1 Duplications Go détectées

| Pattern dupliqué | Occurrences | Suggestion | Classif |
|------------------|-------------|------------|:-------:|
| Pattern `writeJSON(w, http.StatusOK, result)` + error handling dans chaque handler | ~27 handlers avec pattern similaire | Acceptable — pattern Go idiomatique | 🟢 |
| `init()` géants dans `migration/steps_*.go` — même structure register/apply | 3 fichiers (`steps_shared.go`, `steps_player.go`, `steps_metadata.go`) | Structure imposée par registre de migrations — pas de duplication réelle | 🟢 |

### C.2 Duplications React détectées

| Pattern dupliqué | Occurrences | Suggestion | Classif |
|------------------|-------------|------------|:-------:|
| Types manuels (`types.ts`) + types générés (`generated.ts`) coexistent | 2 fichiers | Les types générés ne sont pas consommés — doublon mort | 🟡 |
| Pattern loading/error dans chaque page | ~14 pages | Acceptable — pas assez similaire pour abstraire | 🟢 |

### C.3 Modules trop gros (>500L Go / >300L React)

| Fichier | Lignes | Responsabilités multiples ? | Suggestion découpage | Classif |
|---------|-------:|:---------------------------:|----------------------|:-------:|
| `migration/steps_shared.go` (Go) | 596 | Non — registre unique de steps shared | Toléré : registre de migrations cohérent, 1 responsabilité | 🟡 |
| `SetupPage.tsx` (React) | 467 | Oui — wizard multi-étapes + device flow + smoke test | Extraire `DeviceFlowStep`, `SmokeTestStep` | 🟠 |
| `MediaPage.tsx` (React) | 388 | Oui — grille + filtres + upload + preview | Extraire `MediaGrid`, `MediaFilters` | 🟡 |
| `SessionComparePage.tsx` (React) | 333 | Limite — mais plusieurs sections | Acceptable pour l'instant | 🟡 |
| `SquadPage.tsx` (React) | 328 | Limite | Acceptable | 🟡 |
| `MatchViewPage.tsx` (React) | 334 | Déjà découpé avec sous-composants | Acceptable | 🟢 |
| `MatchHistoryTable.tsx` (React) | 308 | Table complexe unique | Acceptable — TanStack Table verbose | 🟢 |

### C.4 Fonctions trop longues (>80L Go)

**27 violations détectées** — les plus critiques :

| Lignes | Fonction | Fichier | Classif |
|:------:|----------|---------|:-------:|
| 248 | `init()` | `migration/steps_shared.go` | 🟡 — registre de migrations, structurellement inévitable |
| 183 | `init()` | `migration/steps_player.go` | 🟡 — idem |
| 173 | `findMatchesInSharedAll()` | `sync/backfill.go` | 🟡 — requête SQL complexe |
| 169 | `NewRouter()` | `api/server.go` | 🟡 — wire-up routes, linéaire |
| 157 | `init()` | `migration/steps_metadata.go` | 🟡 — idem registre |
| 156 | `NewBackfillFlagSet()` | `sync/backfill_cli.go` | 🟡 — flag registration linéaire |
| 135 | `computeSkillRatingsBatch()` | `sync/skill_rating.go` | 🟠 — algorithme complexe, candidat refactoring |
| 120 | `SendWebhook()` | `notify/discord.go` | 🟡 — construction embed Discord |
| 107 | `RunGateCheck4()` | `validation/gate.go` | 🟡 — checklist de validation |
| 102 | `SyncEngine.run()` | `sync/engine.go` | 🟠 — orchestrateur principal, candidat extract method |

**Les violations sont concentrées dans** : migration (3 `init()`), sync (5 fonctions), analysis (3 fonctions). Ce sont des zones de logique dense intrinsèquement — la dette est acceptable pour un portage mais devrait être réduite progressivement.

---

## D. Workarounds et fallbacks

| Workaround | Fichier:ligne | TTL/ticket | Classif |
|------------|---------------|------------|:-------:|
| `// TODO Sprint 19` media reset-index | `handlers/settings.go:105` | Pas de ticket ni TTL | 🟡 |
| `interface{}` dans domain career (`*interface{}` pour gauges) | `domain/career.go` | Pas de ticket — anti-pattern `*interface{}` | 🟡 |
| Types générés non consommés | `lib/api/generated.ts` (3 231L) | Doublon mort à nettoyer | 🟡 |

---

## E. Qualité globale des conventions

| Convention | Go | React | Classif |
|------------|:--:|:-----:|:-------:|
| Logging structuré | ✅ slog JSON partout | ✅ 0 console.log | 🟢 |
| Error handling | ✅ errors wrapping, writeError centralisé | ✅ TanStack Query error states | 🟢 |
| Code mort | 0 détecté | 0 détecté | 🟢 |
| Magic numbers | Très peu — enums et constantes utilisés | — | 🟢 |
| Context managers DB | N/A Go (defer Close) | — | 🟢 |

---

## Tableau récapitulatif des écarts

| # | Zone | Description | Classif |
|--:|------|-------------|:-------:|
| 1 | React | **Pas d'Error Boundary** — crash composant = page blanche | 🟠 |
| 2 | React | **Pas d'i18n** — labels FR en dur, locale stockée mais inutilisée | 🟠 |
| 3 | React | `SetupPage.tsx` (467L) — god file multi-responsabilités | 🟠 |
| 4 | Go | `computeSkillRatingsBatch()` (135L) et `SyncEngine.run()` (102L) — candidats refactoring | 🟠 |
| 5 | Go | `migration/steps_shared.go` (596L) — seul fichier > 500L non-généré | 🟡 |
| 6 | Go | 27 fonctions > 80L (majoritairement sync/migration/analysis) | 🟡 |
| 7 | Go | ~8 usages `interface{}`/`any` discutables dans domain DTOs | 🟡 |
| 8 | React | 1 cross-feature import (settings → setup/queries) | 🟡 |
| 9 | React | Types manuels + générés coexistent — doublon | 🟡 |
| 10 | React | `MediaPage.tsx` (388L) — candidat découpage | 🟡 |
| 11 | Go | `interface{}` au lieu de `any` dans 6+ fichiers (style pré-1.18) | 🟡 |
| 12 | Go | TODO Sprint 19 sans ticket ni TTL | 🟡 |

---

**Fin de la review Claude — Axe 2.**
