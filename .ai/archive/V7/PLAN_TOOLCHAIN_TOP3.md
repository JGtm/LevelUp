# Plan — Top 3 outils à intégrer au toolchain

> Rédigé le 2026-05-28. Fait suite à l'audit du toolchain existant.
> Contexte : bon coverage déjà en place (golangci-lint, pre-commit, Vitest, Playwright, detect-secrets).
> Ce plan comble les 3 angles morts réels identifiés.

---

## Vue d'ensemble

| Priorité | Outil | Cible | Effort | Valeur |
|----------|-------|-------|--------|--------|
| 1 | `govulncheck` | Go modules | ~30 min | Sécurité CVE dépendances |
| 2 | `knip` | TypeScript/npm | ~1h | Code mort + deps inutilisées |
| 3 | `rollup-plugin-visualizer` | Bundle Vite | ~15 min | Perf / taille bundle |

---

## 1. `govulncheck` — scan vulnérabilités Go

**Pourquoi** : le projet a des dépendances avec surface d'attaque non triviale
(`duckdb-go`, `msal-go` (MSAL AzureAD), `go-chi`, `gorilla/websocket`, `golang.org/x/crypto`).
`detect-secrets` couvre les secrets dans le code, pas les CVE dans les modules.
`govulncheck` est l'outil officiel Go team (Google/OpenSSF) — base OSV.

**Ce qu'il apporte** : signale uniquement les vulnérabilités réellement appelées dans
le code (pas de bruit sur le code mort des dépendances). Output structuré.

### Installation

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### Intégration recommandée

**Option A — pre-commit (léger, local)** :

Dans `.pre-commit-config.yaml`, section `repos: - repo: local` :

```yaml
- id: govulncheck
  name: govulncheck (Go vulnerability scan)
  entry: bash -c 'command -v govulncheck >/dev/null 2>&1 && (cd apps/go-api && govulncheck ./...) || echo "govulncheck non installé — skipped"'
  language: system
  types: [go]
  pass_filenames: false
  stages: [pre-push]   # Avant push uniquement (lent ~15s)
```

**Option B — GitHub Actions (CI)** :

```yaml
- name: govulncheck
  uses: golang/govulncheck-action@v1
  with:
    go-package: ./...
    work-dir: apps/go-api
```

Recommandation : **les deux**. CI pour garantie sur la PR, pre-push pour le feedback local.

### Coût en temps CI

~15-20s. Acceptable en `pre-push` ou step CI.

### Critère de succès

`govulncheck ./...` retourne exit 0 sur la branche `main`.

---

## 2. `knip` — dead code et dépendances inutilisées TypeScript

**Pourquoi** : le frontend a grandi vite (features/, components/, lib/).
Les linters custom existants (no-hardcoded-colors, cross-feature-imports) couvrent
les règles métier mais pas le code orphelin. `knip` détecte :
- exports TS non importés nulle part
- fichiers entiers non référencés
- dépendances npm dans `package.json` non utilisées
- dépendances manquantes (utilisées mais pas déclarées)

**Ce qu'il n'est pas** : un linter de style. Complément à ESLint, pas concurrent.

### Installation

```bash
cd apps/web
npm i -D knip
```

### Configuration recommandée

Créer `apps/web/knip.config.ts` :

```ts
import type { KnipConfig } from 'knip'

const config: KnipConfig = {
  entry: ['src/main.tsx', 'src/routeTree.gen.ts'],
  project: ['src/**/*.{ts,tsx}'],
  ignore: [
    'src/lib/api/generated.ts',  // généré par openapi-typescript
  ],
  ignoreDependencies: [
    '@types/*',      // déclarations types — pas d'import direct
    'jsdom',         // utilisé par vitest config, pas dans le code source
  ],
}

export default config
```

### Usage

```bash
# Audit complet (première passe)
npx knip

# Seulement les fichiers inutilisés
npx knip --include files

# Seulement les dépendances npm inutilisées
npx knip --include dependencies
```

### Intégration pre-commit

```yaml
- id: knip
  name: knip (TypeScript dead code)
  entry: bash -c 'cd apps/web && npx knip --no-exit-code'
  language: system
  files: ^apps/web/src/.*\.(ts|tsx)$
  pass_filenames: false
  stages: [pre-push]   # Lent (~20s) — uniquement avant push
```

**Note** : `--no-exit-code` en première période pour avoir les rapports sans bloquer.
Passer à sans flag une fois la dette initiale éponge.

### Stratégie de démarrage recommandée

1. Lancer `npx knip` une première fois — noter le nombre de findings
2. Corriger les faux positifs (ajouter à `ignoreDependencies` / `ignore`)
3. Réduire la dette progressivement, commit par commit
4. Activer le mode bloquant (sans `--no-exit-code`) une fois le rapport propre

### Critère de succès

`npx knip` retourne 0 fichiers/exports inutilisés (hors liste d'exclusions justifiées).

---

## 3. `rollup-plugin-visualizer` — analyse du bundle Vite

**Pourquoi** : le bundle inclut ECharts (heavy), TanStack Router + Query + Table,
Zustand, react-markdown + ses parseurs remark. Sans visualisation, impossible de
savoir ce qui pèse vraiment et si des optimisations (lazy loading, tree-shaking)
sont à faire.

**Ce que ça donne** : une treemap HTML interactive du bundle par module — visible
en un coup d'œil quels packages consomment combien de Ko.

### Installation

```bash
cd apps/web
npm i -D rollup-plugin-visualizer
```

### Configuration Vite

Dans `apps/web/vite.config.ts`, ajouter le plugin **uniquement en mode analyse** :

```ts
import { visualizer } from 'rollup-plugin-visualizer'

export default defineConfig(({ mode }) => ({
  plugins: [
    react(),
    TanStackRouterVite(),
    tailwindcss(),
    // Activer via ANALYZE=true vite build
    mode === 'analyze' && visualizer({
      open: true,       // ouvre automatiquement dans le browser
      filename: 'dist/stats.html',
      gzipSize: true,
      brotliSize: true,
    }),
  ].filter(Boolean),
}))
```

### Usage

```bash
# Depuis apps/web/
ANALYZE=true npm run build
# → ouvre dist/stats.html dans le browser
```

Ou ajouter un script dans `package.json` :
```json
"analyze": "ANALYZE=true vite build"
```

### Ce qu'il ne faut pas faire

- Ne pas activer en mode `dev` (inutile, le bundle n'est pas optimisé)
- Ne pas committer `dist/stats.html` (ajouter à `.gitignore` si nécessaire)

### Critère de succès

Avoir une baseline du bundle gzippé documentée dans ce fichier après la première analyse.
Vérifier à chaque ajout de dépendance majeure que le delta est justifié.

---

## Ordre d'installation recommandé

1. **`govulncheck`** — 30 min, risque zéro, valeur immédiate (sécurité)
2. **`rollup-plugin-visualizer`** — 15 min, donne une info utile avant toute optimisation frontend
3. **`knip`** — 1h dont la configuration et la première passe de nettoyage

---

## Outils écartés et pourquoi

| Outil | Raison d'exclusion |
|-------|-------------------|
| `Prettier` | ESLint + typescript-eslint stylistic couvre le formatage ; ajouter Prettier crée des conflits de règles |
| `commitlint` | Les conventions sont respectées en pratique ; enforcement formel = friction sans gain réel ici |
| `Storybook` | UI data-driven, pas une composant-lib — ROI trop faible |
| `trivy` | Utile pour scan Docker ponctuel ; pas besoin d'un hook local si `govulncheck` couvre Go et `knip` couvre npm |
| `wire` (DI Go) | Architecture déjà établie, introducing DI maintenant = refactor sans bénéfice |
