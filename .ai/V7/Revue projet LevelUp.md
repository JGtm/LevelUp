# Revue projet LevelUp — code, architecture, agnosticisme charts/données, multi-titres

Tu es un reviewer senior. Audite le projet LevelUp (Halo Infinite stats →
multi-titres). Stack : **backend Go** (`apps/go-api/internal/`), **frontend
React/TS** (`apps/web/`), **charts ECharts**, **DuckDB** (storage). Note : le
`CLAUDE.md` racine est partiellement obsolète (mentionne Python/Streamlit) — la
stack actuelle est Go + React. Croise avec le code réel, pas la doc.

⚠️ AVANT toute exploration, lis dans cet ordre :
1. `.ai/project_map.md`, `.ai/thought_log.md` (5 dernières entrées)
2. `apps/go-api/internal/games/canonical/` (types canoniques multi-titres)
3. `apps/web/src/features/` + `apps/web/src/components/` (structure front)
4. Les skills disponibles : `arch-rules`, `canonical-types`, `frontend-patterns`,
   `color-tokens` — applique-les comme grille.

## Livrable

Rapport markdown. Pour chaque constat :
- **Fichier:ligne** + extrait court
- **Sévérité** : 🔴 bloquant / 🟠 dette structurelle / 🟡 amélioration
- **Action** (1 phrase concrète, pas "envisager de…")

Pas de blabla introductif. Pas de "globalement le code est propre" sans preuve.
Si un axe est sain, 1 ligne et tu passes. Cible 30–60 constats utiles, pas 200
remarques de style.

## Axes prioritaires (dans cet ordre)

### 1. 🎯 Agnosticisme données ↔ charts (axe critique)

Hypothèse à vérifier : les payloads ECharts sont reconstruits N fois à partir
de données déjà transformées, et la forme des données est figée par les besoins
du chart au lieu d'être dérivée d'un modèle de domaine stable.

À chercher :
- Combien de fois le **même indicateur** (KDA, WR, accuracy, performance score,
  contribution, synergy) est recalculé/remappé dans des endroits différents
  (Go service, Go API handler, React feature, hook chart) ?
- Existe-t-il une couche `chart-adapter` / `chart-mapper` centralisée, ou
  chaque page React construit ses `series`, `xAxis`, `tooltip` à la main ?
- Les types Go renvoyés par l'API sont-ils **orientés domaine** (PlayerMatch,
  SessionStats…) ou **orientés chart** (RadarPoints, TimeSeriesPayload qui
  ressemble déjà à du ECharts) ? Un type orienté chart = couplage figé.
- Côté front, les composants chart prennent-ils un **modèle générique**
  (`SeriesData<T>`, `Datum[]`) ou des props spécifiques à ECharts (`option`,
  `series` brut) qui les rendent non-portables ?
- Test mental : "si on remplaçait ECharts par Recharts/visx demain, combien de
  fichiers seraient impactés ?" Si la réponse est >10, c'est 🔴.
- Test mental 2 : "si on ajoute un champ `assists` dans un match, combien de
  builders de payload doivent être touchés ?" Si la réponse est >3, c'est 🟠.

Liste précisément les **endroits où le même calcul est dupliqué** (Go ↔ Go,
Go ↔ TS, ou TS ↔ TS) avec exemples.

### 2. 🌐 Architecture multi-titres

- `internal/games/canonical/` : couvre-t-il bien tous les domaines (career,
  match, identity, scopes, timeseries) ? Reste-t-il des types parallèles
  Halo-spécifiques qui devraient passer par canonical ?
- Cherche les `"halo"`, `"infinite"`, `"halo_infinite"` hardcodés dans des
  couches censées être agnostiques (analysis, service, api/handlers, front
  features génériques). Distingue les hardcodes légitimes (driver Halo) des
  fuites (logique générique qui suppose Halo).
- Le schéma DuckDB porte-t-il une dimension `game_id` ou est-il mono-titre ?
- Le front a-t-il une notion de "titre courant" (route, store, context) ou
  tout est implicitement Halo ?
- Les routes API sont-elles préfixées/segmentées par titre (`/v1/halo/...`)
  ou plates (`/v1/matches`) ?

### 3. 🏛 Layering & responsabilités (Go)

- Frontières `api/` (handlers HTTP) → `service/` (orchestration) → `analysis/`
  (calcul pur) → `port/` (interfaces) → `sync/` (ingestion) : sont-elles
  respectées, ou un handler appelle-t-il directement DuckDB ?
- Les handlers font-ils du calcul métier au lieu de déléguer à `service/` ?
- `analysis/` est-il **pur** (pas d'I/O, pas de DB, testable sans mock) ?
- Les interfaces (`port/`) sont-elles utilisées pour DI/test, ou les services
  dépendent directement d'implémentations concrètes ?
- Cycles d'import ? Packages "fourre-tout" (`util`, `helpers`, `common`) ?

### 4. 🧱 Front React/TS — structure & patterns

- `features/` vs `components/` vs `lib/` : la frontière tient-elle ?
  (composants partagés dans `components/`, logique métier dans `features/`,
  utils purs dans `lib/`)
- Composants >300L, hooks qui font 4 choses, `useEffect` qui contient de la
  logique métier ?
- Stores (`stores/`) : portée correcte, ou globaux pour des trucs locaux ?
- Tests présents pour les composants critiques (notamment ceux qui viennent
  d'être ajoutés : `SessionMultiSelect`, `SquadLayout`) ?
- Routes TanStack (`routeTree.gen.ts`) : les loaders sont-ils utilisés ou
  tout passe par `useEffect` ?

### 5. 🎨 Color tokens & charts (règle stricte CLAUDE.md §20)

- Aucun hex `#RRGGBB` ni `text-red-*`/`bg-green-*` dans
  `apps/web/src/features/` et `apps/web/src/components/` (hors exceptions
  documentées : rareté Halo, structurel SVG).
- Toute couleur passe par `tokenCssVar()`, `resolveToken()`, ou
  `getSeriesColors()`.
- Les options ECharts utilisent-elles ces helpers ou des couleurs en dur ?

### 6. 🔁 DRY / réinvention de roue

- Patterns dupliqués 3+ fois → centraliser.
- Fonctions différentes qui font la même chose (Go ↔ Go, ou Go ↔ TS) :
  un même calcul côté backend ET côté frontend ? Lequel fait foi ?
- Magic numbers (`outcome == 2`, seuils, ratios) → enum / constante manquante ?
- Helpers absents là où ils existent (formatters de date, de durée, de KDA) ?

### 7. 🧪 Testabilité & couverture

- `apps/go-api/coverage_baseline.txt` : zones sous le seuil ? régressions
  récentes ?
- Code testable **sans démarrer un serveur** ? Les services prennent-ils des
  interfaces injectables ?
- Tests qui mockent tout au point de ne rien tester ?
- Front : tests unitaires sur logique pure / hooks vs uniquement snapshot ?
- Tests de contrat API (`contracttest/`) : à jour avec les types canonical ?

### 8. 📋 Logs & observabilité

- `internal/observability/`, `internal/notify/` : utilisés de façon cohérente,
  ou chaque package log à sa sauce (`log.Println`, `fmt.Println`) ?
- Niveaux corrects (debug/info/warn/error) ou tout en info ?
- Logs avec contexte (request_id, gamertag, match_id) ou messages plats ?
- `panic()` ou `log.Fatal()` dans des packages bibliothèque (interdit hors `cmd/`) ?

### 9. 💀 Code mort & dette

- Fonctions exportées non appelées (Go : `staticcheck U1000`).
- Composants React non importés.
- Endpoints API non consommés par le front.
- Migrations / scripts one-shot encore présents après exécution.
- Fichiers dans `apps/go-api/` racine qui ressemblent à des binaires checkés
  en dur (`admin.exe`, `levelup-api.exe`, `*.out`) → à exclure du repo ?

### 10. 📦 Dépendances & couplage tech

- Packages ECharts importés directement dans des features (couplage fort) vs
  enveloppés dans un wrapper (`<Chart kind="radar" data={...} />`) ?
- Imports de `duckdb` hors couche repository ?
- Dépendances Go `replace` ou versions floues dans `go.mod` ?

## Format de restitution attendu

### Axe 1 — Agnosticisme données/charts
🔴 [path/file.go:42] KDA recalculé 4 fois (analysis/kda.go, service/match.go, api/handlers/session.go, web/lib/stats.ts) → centraliser dans analysis/, exposer via API typée.
🟠 [web/features/squad/SynergyChart.tsx:88] construit option.series ECharts inline avec 60 lignes de mapping → extraire buildSynergyChartData(domain) → ChartPayload.
...

### Axe 2 — Multi-titres
🔴 [internal/service/career.go:120] hardcode "halo_infinite" dans une fn générique career → paramétrer via canonical.GameID.
...



Démarre par l'axe 1 (c'est celui qui m'intéresse le plus). Si un axe demande
plus de profondeur, dis-le et propose un follow-up ciblé plutôt que de bâcler.
