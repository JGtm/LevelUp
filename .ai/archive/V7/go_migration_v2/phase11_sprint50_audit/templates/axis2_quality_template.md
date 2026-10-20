# Axe 2 — Template architecture & qualité code

> **À remplir à l'identique** par Claude puis par ChatGPT. Ne pas modifier la structure.

## Métadonnées du passage

| Champ | Valeur |
|-------|--------|
| Auteur LLM | Claude \| ChatGPT (entourer) |
| Date du passage | `YYYY-MM-DD` |
| SHA Go | `xxxxxxx` |
| SHA React | `xxxxxxx` |
| Durée de l'analyse | `Nh` |

## Synthèse exécutive (150 mots max)

> Résumé : état général de l'architecture hexagonale, points de friction principaux, workarounds notables.

---

## A. Architecture hexagonale — Go (backend)

### A.1 Identification des couches

| Couche | Package attendu | Package réel | Conforme ? | Commentaire |
|--------|-----------------|--------------|:----------:|-------------|
| Domaine (entités pures) | `internal/domain/` | | | |
| Ports (interfaces) | `internal/port/` | | | |
| Services (orchestration) | `internal/service/` | | | |
| Adapters entrants (HTTP) | `internal/api/handlers/` | | | |
| Adapters sortants (DB) | `internal/platform/duckdb/` | | | |
| Adapters sortants (API externe) | `internal/platform/auth/`, etc. | | | |
| Analyse (algos purs) | `internal/analysis/` | | | |

### A.2 Respect des dépendances (règle : domaine ne dépend de rien, ports ne dépendent que du domaine)

| Règle | Violations identifiées (fichier:ligne) | Classif |
|-------|----------------------------------------|:-------:|
| `internal/domain/` n'importe ni platform ni service ni api | | |
| `internal/port/` n'importe que `internal/domain/` | | |
| `internal/analysis/` est sans I/O (pas d'import `database/sql`, `net/http`, `os`) | | |
| `internal/service/` dépend des ports, pas des implémentations platform | | |
| `internal/api/handlers/` ne touche pas `internal/platform/duckdb/` directement | | |

### A.3 Qualité des abstractions

| Question | Réponse + fichier:ligne | Classif |
|----------|-------------------------|:-------:|
| `port.Services` : interface bien segmentée ou fourre-tout ? | | |
| Les repos DuckDB exposent-ils une interface côté consommateur ? | | |
| Les DTOs HTTP sont-ils distincts des entités domaine ? | | |
| Y a-t-il des `interface{}` / `any` non justifiés ? | | |

---

## B. Architecture — React (frontend)

### B.1 Structure par feature

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| `src/features/<domaine>/` isolé (pas de fuite cross-feature) | | |
| Routes (`src/routes/`) découplées des features | | |
| Composants UI (`src/components/ui/`) pur présentation | | |
| Hooks dédiés par feature (data fetching, state local) | | |
| Séparation fetching (React Query ?) / rendering | | |

### B.2 Couches d'abstraction

| Couche | Présente ? | Commentaire |
|--------|:----------:|-------------|
| Client API typé (généré depuis OpenAPI ?) | | |
| Couche état global (Zustand/Redux/Context) | | |
| Couche i18n | | |
| Error boundaries / fallback UI | | |

---

## C. Duplications & factorisation

### C.1 Duplications Go détectées

| Pattern dupliqué | Occurrences (fichier:ligne) | Suggestion | Classif |
|------------------|-----------------------------|------------|:-------:|
| | | | |

### C.2 Duplications React détectées

| Pattern dupliqué | Occurrences | Suggestion | Classif |
|------------------|-------------|------------|:-------:|
| | | | |

### C.3 Modules trop gros (>500L Go / >300L React)

| Fichier | Lignes | Responsabilités multiples ? | Suggestion découpage | Classif |
|---------|-------:|:---------------------------:|----------------------|:-------:|
| | | | | |

### C.4 Fonctions trop longues (>80L)

| Fichier:fonction | Lignes | Justifié par `//nolint` ? | Suggestion | Classif |
|------------------|-------:|:-------------------------:|------------|:-------:|
| | | | | |

---

## D. Workarounds & fallbacks non pertinents

> Chercher spécifiquement : branches `if legacy`, commentaires `// TODO`, `// FIXME`, `// HACK`, fallbacks silencieux, retours `nil` masqués, try/catch vides.

| Emplacement (fichier:ligne) | Description du workaround | Raison (connue/inconnue) | Toujours nécessaire ? | Classif |
|-----------------------------|---------------------------|--------------------------|:---------------------:|:-------:|
| | | | | |

### Fallbacks potentiellement dangereux

| Emplacement | Comportement observé | Risque | Classif |
|-------------|----------------------|--------|:-------:|
| | | | |

---

## E. Anti-patterns CLAUDE.md revisités (adaptés Go/React)

> La baseline Python a 11 anti-patterns identifiés. On vérifie qu'ils ne sont pas réintroduits côté Go/React.

| Anti-pattern | Version Go/React | Présent ? | Fichier:ligne | Classif |
|--------------|------------------|:---------:|---------------|:-------:|
| Dead code museum | Fonctions exportées non appelées | | | |
| God file | Fichier >500L Go mono-responsabilité diluée | | | |
| Swiss-army function | Handler qui init+call+transform+write | | | |
| Magic number | Littéraux sans constante nommée | | | |
| Bare connect | `sql.Open` sans `defer db.Close()` | | | |
| `map[string]any` / `interface{}` non justifié | | | | |
| Shadow / legacy guard sans date | Branches `if legacyBackend` sans TTL | | | |

---

## F. Cohérence de nommage & conventions

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| Packages Go respectent `golint` naming | | |
| Handlers Go : un fichier par endpoint racine ? | | |
| React : convention PascalCase pour composants, camelCase pour hooks | | |
| Types TypeScript : `types.ts` ou colocalisés ? Cohérent ? | | |

---

## G. Configuration & feature flags

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| Feature flags Go (`internal/config/feature_flags.go`) — flags actifs vs morts | | |
| Variables d'environnement documentées ? | | |
| Config React (env Vite) — cohérente avec backend ? | | |

---

## H. Récap classifications

| Niveau | Nombre d'items |
|--------|:--------------:|
| 🔴 Bloquant | |
| 🟠 Majeur | |
| 🟡 Mineur | |
| 🟢 Toléré | |

## I. Top 5 dettes techniques prioritaires

> Les 5 items les plus critiques à adresser post-audit. Chacun avec fichier:ligne, effort estimé, impact.

| # | Dette | Fichier:ligne | Effort (S/M/L) | Impact |
|--:|-------|---------------|:--------------:|--------|
| 1 | | | | |
| 2 | | | | |
| 3 | | | | |
| 4 | | | | |
| 5 | | | | |

## J. Observations libres

> Max 300 mots.

---

**Fin du template axe 2.**
