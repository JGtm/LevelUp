# Plan — Drawer feedback "Reporter un retour" (FeedbackDrawer)

> **Révision 2026-05-05** : passe de revue d'architecture appliquée (cf. § Changelog plan en bas du document).

## Context

L'app LevelUp dispose actuellement d'un seul drawer latéral droit : [AssetDrawer](apps/web/src/features/asset-drawer/AssetDrawer.tsx) (référentiel maps/armes), monté dans [AppShell.tsx](apps/web/src/components/shell/AppShell.tsx#L50). On veut ajouter un **second mini-drawer**, plus petit, sous celui-ci avec une marge visible, déclenché par une **icône bulle de message**, qui ouvre un panneau de feedback.

Au submit, on **redirige l'utilisateur vers une URL GitHub Issues préremplie** (`github.com/JGtm/LevelUp/issues/new?...`) avec un maximum de contexte auto-collecté **enrichi par des heuristiques** (type/sévérité/zone). En post-création, une **GitHub Action** déclenche Claude Haiku 4.5 pour analyser, classifier finement, suggérer des causes/workarounds et commenter l'issue.

Choix arrêtés :
- **Mécanisme** : URL GitHub préremplie (front) + GitHub Action Claude API (post-création).
- **Icône** : bulle de message SVG inline.
- **Auto-collecte** : contexte page + environnement client + contexte métier (filtres) + erreurs console récentes.
- **Détection doublons** : section "Issues similaires" sous le titre (debounce 500 ms, fetch GitHub search API publique).

## Pré-requis git (CLAUDE.md — Stratégie de branches)

La branche `fix/synergy-radar-calibration` est la base attendue (les blocs UI Drawer en cours y vivent). On ne passe pas par `main` car local et origin ont divergé fortement.

```bash
git checkout fix/synergy-radar-calibration
git checkout -b feat/feedback-drawer
```
**Ne jamais travailler sur `main`** ni continuer le travail courant sur `fix/synergy-radar-calibration` (sujet différent).

## UX & visuel

### Pattern visuel (reprend AssetDrawer)

Le **AssetDrawer** utilise un pattern à 2 éléments fixés sur le bord droit, centrés verticalement (`top-1/2`) :
- un **mini-tab** (bouton vertical avec texte, `writingMode: vertical-rl`) qui dépasse du bord
- un **panneau** qui slide via `transform: translateX(...)`

Le **FeedbackDrawer** réutilise ce pattern, **décalé verticalement vers le bas** :
- Mini-tab fixé à droite, ~30 px sous AssetDrawer. **Calcul robuste viewport-aware** : `top-[calc(50%+min(330px,42vh))]` (le AssetDrawer est borné par `min(600px,80vh)` donc rétrécit sur petit viewport ; on suit le même bornage pour éviter chevauchement avec NavL1 ou main content).
- Mini-tab plus court (sans texte vertical) : juste l'icône bulle ~36×36 px.
- Panneau plus petit que AssetDrawer : `w-[340px]` × `h-[min(540px,70vh)]`, ancré bottom-right à la même latitude que le mini-tab.
- Tokens sémantiques uniquement (`bg-popover`, `border-border`, `text-popover-foreground`, `hover:bg-accent`) — aucun hex, aucun Tailwind couleur (cf. règle 20 CLAUDE.md ; lint custom strict ratchet 0).
- Fermeture clavier `Escape` (même hook que AssetDrawer).
- **Mobile (`< 640 px`)** : caché en v1 (`hidden sm:flex`, comme AssetDrawer). Limitation acceptée : un user mobile rencontrant un bug doit basculer sur desktop. À reconsidérer en v1.1 sous forme de FAB bottom-right.

### Contenu du panneau (de haut en bas)

1. **Header** : titre "Envoyer un retour" + bouton croix.
2. **Type** (segmented control 3 boutons) : Bug / Idée / Question — détermine le préfixe titre + label GitHub.
3. **Titre** : `<input>` court (max 80 chars).
4. **Section "Issues similaires"** *(conditionnelle)* : après debounce 500 ms sur le titre, query GitHub search API publique. Si ≥1 résultat, on affiche `Une issue existe peut-être déjà :` + max 3 liens cliquables `#123 — titre…`. Non-bloquant.
5. **Description** : `<textarea>` (4-6 lignes).
6. **Toggle "Joindre les infos techniques"** (coché par défaut) avec un `<details>` qui montre un aperçu Markdown live de ce qui sera envoyé.
7. **Bouton "Ouvrir sur GitHub"** : construit URL → `window.open(url, '_blank', 'noopener,noreferrer')`.
8. **Note discrète** : "Vous serez redirigé vers GitHub pour finaliser l'envoi. Une analyse automatique enrichira l'issue."

## Architecture

### Pattern global

Suivre **strictement** le pattern AssetDrawer + conventions des features récentes (`notifications/`, `synthesis/`) :
- Store Zustand local (`isOpen`, `toggle`, `close`, `open`) + `persist` partial.
- Composant React unique exporté + sous-composants privés dans le même fichier.
- **Helpers purs** (no React, no Zustand) extraits dans modules à part pour testabilité.
- Hook react-query dans `queries.ts` (convention dominante des features récentes ; AssetDrawer utilise `useAssetDrawer.ts` mais c'est plus ancien).
- i18n via manifest TOML + script de génération existant.

### 🔴 Sécurité fetch GitHub (critique)

Le wrapper [`api`](apps/web/src/lib/api/client.ts) du projet a `credentials: 'include'` + injecte `X-LevelUp-Title` quand le slug ≠ `halo_infinite`. **L'utiliser pour appeler `api.github.com` leakerait ce header et tenterait d'envoyer les cookies de session LevelUp à GitHub.**

→ Pour `useSimilarIssues` (et toute call vers GitHub), **utiliser `fetch()` direct** :
```ts
const res = await fetch('https://api.github.com/search/issues?...', {
  credentials: 'omit',
  headers: { Accept: 'application/vnd.github+json' },
})
```
Documenter en commentaire dans `queries.ts` pour qu'aucun futur dev ne switch vers `api.get`.

### Heuristiques de classification côté front

Helper pur `classifyFeedback(input, context) → { type, severity, area }` :

**Type & severity** (priorité haute → basse, premier match gagne) :

| Signal | Conséquence |
|---|---|
| Erreurs console récentes contiennent `TypeError` / `ReferenceError` | `severity: critical`, `type: bug` (force) |
| Description contient `crash`, `perd ma progression`, `impossible`, `bloqué` | `severity: high`, `type: bug` |
| Erreurs console récentes ≥1 entry (autres niveaux) | `severity: high`, `type: bug` (suggéré) |
| Description contient `bug`, `marche pas`, `erreur`, `cassé` (sans mots high) | `severity: medium`, `type: bug` |
| Description contient `?`, `comment`, `pourquoi` | `severity: low`, `type: question` |
| Description contient `j'aimerais`, `ce serait bien`, `feature`, `idée` | `severity: low`, `type: enhancement` |
| Sinon | `severity: low`, `type: enhancement` (par défaut Idée) |

**Area** (URL pathname → zone, premier match gagne) :

> Routes vérifiées dans [apps/web/src/routes/](apps/web/src/routes/). Pas de pattern fictif (`engagement` n'a pas de route propre — il vit sous `/squad`).

| Pattern URL | `area` |
|---|---|
| `/players/.../synthesis` | `synthesis` |
| `/players/.../explorer` | `explorer` |
| `/players/.../squad` (incl. `/contributions`, `/synergies`) | `squad` |
| `/players/.../stats/sessions` | `sessions` |
| `/players/.../stats/timeseries` | `timeseries` |
| `/players/.../stats/history` | `match_history` |
| `/players/.../matches/...` | `match_view` |
| `/players/.../palmares*` (incl. `prestige`, `relations`, `compare`, `season-pass`) | `palmares` |
| `/players/.../home` | `player_home` |
| `/players/.../media` | `media` |
| `/players/.../career` | `career` |
| `/players/.../notifications` | `notifications` |
| `/players/.../objectifs` | `objectifs` |
| `/players/.../profile/citations` | `citations` |
| `/setup` ou `/settings` | `settings` |
| `/changelog` ou `/help` | `meta` |
| Fallback | `general` |

Ces 3 valeurs sont injectées dans `labels=` de l'URL GitHub (ex: `feedback,bug,severity:high,area:synthesis`).

### Métadonnées techniques avancées (lib/global-capture/)

**Pas de précédent** d'init globaux dans `main.tsx` (10 lignes, juste mount React). Le terme "bootstrap" est déjà utilisé dans le projet pour l'auth/session ([client.ts L83](apps/web/src/lib/api/client.ts#L83) endpoint `/bootstrap`, `app/providers.tsx`) — éviter la collision sémantique :

→ Créer `apps/web/src/lib/global-capture/install.ts` qui exporte `installGlobalCapture()` regroupant tous les side-effects globaux du feedback drawer. Appelé une fois en tête de `main.tsx`, avant `createRoot`.

Module `lib/global-capture/buffers.ts` (appelé par `install.ts`) gère plusieurs ring buffers :

- **Console** : wrap `console.error` + `console.warn`, garde 20 dernières entrées `{ level, message, timestamp, stack? }`. Stack extraite uniquement si `arg instanceof Error` (sinon stringify les args).
- **Window errors** : `window.addEventListener('error')` + `'unhandledrejection'` → ajoute aussi au buffer console avec stack complète.
- **Network** : intercepteur `fetch` (monkey-patch) qui capture les **5 dernières requêtes échouées** (status ≥ 400 ou network error) `{ url, method, status, timestamp }`. **L'URL stockée est toujours strippée des query params** (`url.split('?')[0]`) pour éviter de fuiter tokens, gamertags, IDs en clair dans le body GitHub public.
- **Focused element** : récupéré au moment du clic feedback via `document.activeElement?.tagName + className`.

API exposée :
- `getRecentConsoleEntries(): ConsoleEntry[]`
- `getRecentFailedRequests(): FailedRequest[]`
- `installGlobalCapture(): void` (idempotent, voir ci-dessous)
- `resetCaptureBuffersForTests(): void` (uniquement utilisé par les tests Vitest, pattern aligné sur `_logger.ts::_resetForTests`)

L'output console n'est **pas** altéré (les originaux sont toujours appelés).

#### Idempotence sous HMR Vite

Un flag module-level `_isInstalled = true` est **insuffisant** : Vite HMR recharge le module → flag re-init à `false` → wrap re-appliqué → ring buffer reçoit chaque event 2× (puis 3× après le HMR suivant).

Solution : stocker le flag sur `globalThis` (survit au HMR car partagé window-level) :

```ts
const KEY = '__levelup_global_capture_installed__'
declare global { var __levelup_global_capture_installed__: boolean | undefined }
export function installGlobalCapture(): void {
  if (globalThis[KEY]) return
  globalThis[KEY] = true
  // … wrap console / fetch / errors …
}
```

Le test `install.test.ts` doit simuler 2 imports successifs (ou clear le `globalThis[KEY]` puis re-installer) pour vérifier l'idempotence.

### GitHub search query — encodage safe

Le titre saisi par l'user est injecté dans `?q=...` de l'API search. Caractères GitHub-search réservés (`:`, `+`, `"`, `(`, `)`) cassent ou détournent la query. **Helper `escapeSearchQuery(title): string`** dans `queries.ts` :

1. Remplace les opérateurs réservés par un espace : `title.replace(/[:+"()/]/g, ' ')`.
2. Trim + collapse les espaces multiples : `.replace(/\s+/g, ' ').trim()`.
3. Append le scope : `q=<sanitized>+is:issue+repo:JGtm/LevelUp`.
4. `encodeURIComponent` final sur la query complète.

Test dédié dans `queries.test.ts` couvre 5 cas (`"foo: bar"`, `feature+`, `(crash)`, slashes URL collés au titre, titre 100 % réservés → fallback no-op).

### Fallback repo privé (futur)

La query GitHub Search publique fonctionne tant que le repo est public. Si demain le repo passe privé, l'API retournera 404 sans token auth → la section "Issues similaires" sera masquée silencieusement (`warn similar:fetch_failed`).

→ Comment migrer le jour venu : exposer un endpoint backend `GET /api/v1/feedback/search-issues?q=...` qui proxy l'appel avec un PAT readonly (secret Go API). À documenter en commentaire dans `queries.ts` pour ne pas perdre le chemin de migration.

### Limite GitHub URL

URLs GitHub Issues préremplies fonctionnent jusqu'à ~**8 000 caractères** en pratique. `buildIssueUrl` doit :
- Tronquer le `body` Markdown à **7 000 chars** (sécurité).
- Tronquer en priorité la section "Erreurs console récentes" (la plus volatile), puis "Filtres actifs", avant "Description".
- Ajouter `…[truncated]` si troncature.

### GitHub Action de triage IA

Workflow déclenché à `issues.opened` filtré sur le label `feedback`. Appelle Claude Haiku 4.5 avec un **system prompt** qui connaît la structure du body LevelUp et retourne un JSON :

```json
{
  "severity_refined": "low|medium|high|critical",
  "area_refined": "synthesis|explorer|engagement|squad|session|filters|sync|auth|general",
  "title_normalized": "...",
  "summary_one_liner": "...",
  "probable_cause": "...",
  "suggestions": ["...", "..."],
  "similar_internal_issues": [{ "number": 42, "reason": "..." }],
  "is_actionable": true
}
```

Actions appliquées au repo :
- `gh issue edit` → ajoute/affine les labels (`severity:*`, `area:*`, `triage:claude-analyzed`).
- `gh issue comment` → poste un commentaire formaté :
  - **Résumé** (1 ligne).
  - **Cause probable**.
  - **Suggestions** (bullets).
  - **Issues internes potentiellement liées**.
- Si `is_actionable === false` (ex: spam, message vide, troll) → label `triage:needs-review`, pas de commentaire long.

**Coût** : repo public → Actions illimitées. Haiku 4.5 sur ~5k tokens input (system prompt + body + 50 issues récentes en contexte) + ~500 tokens output ≈ **$0.001/issue**. Seuil d'alerte fixé à 1000 issues/mois ($1/mois) — au-delà, vraisemblablement spam ou flux anormal. Secret repo `ANTHROPIC_API_KEY` à provisionner via GitHub UI.

## Fichiers à créer

### Frontend

| Fichier | Rôle |
|---|---|
| [apps/web/src/features/feedback-drawer/FeedbackDrawer.tsx](apps/web/src/features/feedback-drawer/FeedbackDrawer.tsx) | Composant React (mini-tab + panneau + form) |
| [apps/web/src/features/feedback-drawer/feedbackDrawer.store.ts](apps/web/src/features/feedback-drawer/feedbackDrawer.store.ts) | Zustand store (`isOpen`, `toggle`, `close`, `open`) |
| [apps/web/src/features/feedback-drawer/index.ts](apps/web/src/features/feedback-drawer/index.ts) | Re-export `FeedbackDrawer` |
| [apps/web/src/features/feedback-drawer/buildIssueUrl.ts](apps/web/src/features/feedback-drawer/buildIssueUrl.ts) | Helper pur : `(input, context, classification) → URL`, encodage + troncature |
| [apps/web/src/features/feedback-drawer/classifyFeedback.ts](apps/web/src/features/feedback-drawer/classifyFeedback.ts) | Helper pur : heuristiques type/sévérité/zone |
| [apps/web/src/features/feedback-drawer/collectContext.ts](apps/web/src/features/feedback-drawer/collectContext.ts) | Helper pur : agrège stores + browser APIs en `FeedbackContext` |
| [apps/web/src/features/feedback-drawer/queries.ts](apps/web/src/features/feedback-drawer/queries.ts) | Hook `useSimilarIssues` (react-query, **fetch direct GitHub** sans wrapper api ; encodage GitHub-search safe — voir § GitHub search query) |
| [apps/web/src/features/feedback-drawer/rateLimit.ts](apps/web/src/features/feedback-drawer/rateLimit.ts) | Helper pur : `recordSubmit()` / `getRemainingSubmits()` lus depuis `localStorage` (clé `levelup-feedback-submits`, fenêtre glissante 1h, max 5) |
| [apps/web/src/features/feedback-drawer/_logger.ts](apps/web/src/features/feedback-drawer/_logger.ts) | Logger namespacé `[feedback-drawer]` — pattern existant (`features/filters/_logger.ts`) |
| [apps/web/src/features/feedback-drawer/buildIssueUrl.test.ts](apps/web/src/features/feedback-drawer/buildIssueUrl.test.ts) | Tests Vitest : encodage, troncature progressive, mapping |
| [apps/web/src/features/feedback-drawer/classifyFeedback.test.ts](apps/web/src/features/feedback-drawer/classifyFeedback.test.ts) | Tests Vitest : couvre les règles heuristiques |
| [apps/web/src/features/feedback-drawer/collectContext.test.ts](apps/web/src/features/feedback-drawer/collectContext.test.ts) | Tests Vitest : agrégation des stores (mocks Zustand) |
| [apps/web/src/features/feedback-drawer/feedbackDrawer.store.test.ts](apps/web/src/features/feedback-drawer/feedbackDrawer.store.test.ts) | Tests Vitest : open/close/toggle/persist |
| [apps/web/src/features/feedback-drawer/queries.test.ts](apps/web/src/features/feedback-drawer/queries.test.ts) | Tests Vitest + MSW : `useSimilarIssues` (debounce, fetch GitHub mocké, error path, headers `omit`, escape opérateurs GitHub-search) |
| [apps/web/src/features/feedback-drawer/rateLimit.test.ts](apps/web/src/features/feedback-drawer/rateLimit.test.ts) | Tests Vitest : fenêtre glissante 1h, max 5, expire les > 1h, gère `localStorage` indisponible |
| [apps/web/src/features/feedback-drawer/FeedbackDrawer.test.tsx](apps/web/src/features/feedback-drawer/FeedbackDrawer.test.tsx) | Tests `@testing-library/react` : rendu, open/close, submit ouvre `window.open` avec URL attendue, Escape ferme, focus auto, accessibilité ARIA, fallback clipboard si `window.open` retourne `null`, bouton désactivé après 5 submits |
| [apps/web/src/lib/global-capture/install.ts](apps/web/src/lib/global-capture/install.ts) | `installGlobalCapture()` — point d'entrée idempotent (HMR-safe via `globalThis`) des side-effects globaux |
| [apps/web/src/lib/global-capture/install.test.ts](apps/web/src/lib/global-capture/install.test.ts) | Tests Vitest : idempotence (2× install = 1× effet, même après reset du flag `globalThis`), n'altère pas `console.error` original |
| [apps/web/src/lib/global-capture/buffers.ts](apps/web/src/lib/global-capture/buffers.ts) | Ring buffers : console + window errors + failed fetch (URLs strippées des query params à la capture) |
| [apps/web/src/lib/global-capture/buffers.test.ts](apps/web/src/lib/global-capture/buffers.test.ts) | Tests Vitest : ring buffer 20 entries FIFO, stack si `instanceof Error`, fetch interceptor capture 4xx/5xx/network errors, **URL strippée des query params**, `console.error` original toujours appelé, `resetCaptureBuffersForTests()` vide proprement |
| [apps/web/src/lib/i18n/manifests/feedback_drawer.toml](apps/web/src/lib/i18n/manifests/feedback_drawer.toml) | Clés FR/EN (titres, labels, placeholders) — convention snake_case complète alignée sur `asset_drawer.toml` |
| [apps/web/e2e/feedback-drawer.spec.ts](apps/web/e2e/feedback-drawer.spec.ts) | Test Playwright e2e : ouvre drawer, remplit form, intercepte `window.open`, valide URL GitHub finale (labels, title, body), vérifie aucun cookie/header LevelUp envoyé à api.github.com, **viewport 375×667 → mini-tab caché** (régression mobile) |

### GitHub Action

| Fichier | Rôle |
|---|---|
| [.github/workflows/triage-feedback.yml](.github/workflows/triage-feedback.yml) | Workflow trigger `issues.opened` + label `feedback` → exécute le script Node |
| [.github/workflows/sync-labels.yml](.github/workflows/sync-labels.yml) | Workflow `crazy-max/ghaction-github-labeler` qui synchronise les labels depuis `.github/labels.yml` |
| [.github/labels.yml](.github/labels.yml) | Déclaration versionnée des labels (feedback, bug, severity:*, area:*, triage:*) |
| [.github/scripts/triage-feedback.mjs](.github/scripts/triage-feedback.mjs) | Script Node : parse body, appelle Claude API, applique labels + commentaire via `gh` CLI |
| [.github/scripts/triage-feedback.test.mjs](.github/scripts/triage-feedback.test.mjs) | Tests Node (built-in `node:test`) : parse body LevelUp, fallback parse-error, format commentaire |
| [.github/scripts/triage-feedback-prompt.md](.github/scripts/triage-feedback-prompt.md) | System prompt versionné (architecture LevelUp + structure body + format JSON attendu) |

## Fichiers à modifier

| Fichier | Modif |
|---|---|
| [apps/web/src/components/shell/AppShell.tsx](apps/web/src/components/shell/AppShell.tsx#L50) | Ajouter `<FeedbackDrawer />` en sibling après `<AssetDrawer />` |
| [apps/web/src/main.tsx](apps/web/src/main.tsx) | Importer `installGlobalCapture()` depuis `lib/global-capture/install` et l'appeler avant `createRoot` |

### Réutilisation existant (pas de nouveau code)

- Filtres : [`useGlobalFilterStore`](apps/web/src/stores/globalFilterStore.ts) — `filterContext` (period + cascade : modes/maps/playlists/seasons).
- Shell : [`useAppShellStore`](apps/web/src/stores/appShellStore.ts) — `currentTitleSlug` + `locale`.
- Thème : [`useSettingsDraftStore`](apps/web/src/stores/settingsDraftStore.ts) — `localUiPrefs.theme`.
- Player slug : `useParams()` TanStack Router.
- Version Go API : `GET /health` via [`api`](apps/web/src/lib/api/client.ts) → `app_version`. Fallback `"unknown"`.
- i18n : `formatMessage(feedbackDrawerManifest, key, locale)` (cf. AssetDrawer ligne 17).
- Build i18n : `node apps/web/scripts/build_i18n_manifests.mjs`.
- React Query déjà installé (v5.99.0).

## Format de l'issue GitHub

### URL construite

```
https://github.com/JGtm/LevelUp/issues/new
  ?labels=feedback,<type>,severity:<sev>,area:<area>
  &title=<encoded>          (préfixé "[Bug] " / "[Idée] " / "[?] ")
  &body=<encoded markdown>  (max 7 000 chars, troncature progressive)
```

### Body Markdown

```markdown
## Description
<saisie utilisateur>

---

## Contexte
- **URL** : /players/Player/synthesis?period=last_30d
- **Titre** : halo_infinite
- **Joueur** : Player
- **Locale** : fr  ·  **Thème** : dark
- **Timestamp** : 2026-05-04T12:34:56Z
- **Élément focus** : button.action-primary

## Environnement client
- **Version app** : 7.0.0  *(ou "unknown" si /health KO)*
- **User-Agent** : Mozilla/5.0 ...
- **Viewport** : 1920 × 1080

## Filtres actifs
- **Mode filtre** : period
- **Période** : 2026-04-04 → 2026-05-04
- **Modes** : Slayer, CTF
- **Maps** : Aquarius, Live Fire
- **Playlists** : Ranked Arena
- **Saisons** : S5

## Classification heuristique (front)
- **Type** : bug  ·  **Sévérité** : high  ·  **Zone** : synthesis

## Erreurs console récentes (20 dernières)
\`\`\`js
[ERROR 12:33:45] TypeError: Cannot read properties of undefined (reading 'id')
  at SaisonPill (SaisonPill.tsx:42)
[WARN  12:33:42] React key duplicated in <PeriodSessionRail>
\`\`\`

## Requêtes échouées récentes (5 dernières)
\`\`\`
GET /api/v1/players/Player/seasons → 500 (12:33:45)
\`\`\`

---
*Auto-généré par le drawer feedback — LevelUp web*
*Une analyse automatique sera ajoutée en commentaire dans quelques secondes.*
```

## GitHub Action — détail technique

### `.github/workflows/triage-feedback.yml`

```yaml
name: Triage feedback issues

on:
  issues:
    types: [opened]

jobs:
  triage:
    if: contains(github.event.issue.labels.*.name, 'feedback')
    runs-on: ubuntu-latest
    permissions:
      issues: write
      contents: read
    steps:
      # Toutes les actions tierces sont pinnées par SHA (supply-chain hygiene,
      # le repo expose ANTHROPIC_API_KEY). Tag en commentaire pour la lisibilité.
      - uses: actions/checkout@<sha>          # v4
      - uses: actions/setup-node@<sha>        # v4
        with: { node-version: '20' }
      - run: npm install @anthropic-ai/sdk
      - run: node .github/scripts/triage-feedback.mjs
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          ISSUE_NUMBER: ${{ github.event.issue.number }}
          ISSUE_TITLE: ${{ github.event.issue.title }}
          ISSUE_BODY: ${{ github.event.issue.body }}
          REPO: ${{ github.repository }}
```

### `.github/scripts/triage-feedback.mjs` (squelette)

- Charge `triage-feedback-prompt.md` (system prompt versionné).
- Récupère via `gh api` la liste des issues ouvertes du repo (titres + numéros) — limite 50 récentes.
- Appelle `client.messages.create({ model: 'claude-haiku-4-5-20251001', system, messages: [{ role: 'user', content: <issue body + liste> }], max_tokens: 1024 })`.
- Parse JSON réponse (avec fallback gracieux si parsing échoue → label `triage:parse-error`).
- Applique : `gh issue edit $ISSUE_NUMBER --add-label "..."` puis `gh issue comment $ISSUE_NUMBER --body "..."`.

### Pré-requis manuels (one-shot)

1. Créer une clé Anthropic API (console.anthropic.com).
2. Repo Settings → Secrets and variables → Actions → New secret `ANTHROPIC_API_KEY`.
3. Les labels GitHub sont **synchronisés automatiquement** via `.github/workflows/sync-labels.yml` + `.github/labels.yml` (action `crazy-max/ghaction-github-labeler` pinnée par SHA, tag `v5` en commentaire). Pas de création manuelle.

## Logging

### Frontend — convention `_logger.ts` par feature

Pattern existant aligné sur [features/filters/_logger.ts](apps/web/src/features/filters/_logger.ts), [features/squad/_logger.ts](apps/web/src/features/squad/_logger.ts), [lib/accessibility/_logger.ts](apps/web/src/lib/accessibility/_logger.ts) :
- Préfixe namespacé `[feedback-drawer]`.
- `info` (always), `warn`/`error` (dedup par clé via `Set<string>`, log 1× par session), `debug` (uniquement `NODE_ENV === 'development'`).
- `_resetForTests()` exposé pour les tests Vitest.

### Cibles observables (clés stables)

| Niveau | Clé | Quand |
|---|---|---|
| `info` | — | Drawer ouvert / fermé (debug seulement) |
| `info` | — | Submit feedback `{ type, severity, area, urlLength }` (sans contenu PII) |
| `warn` | `health:fetch_failed` | `/health` KO → version "unknown" injectée |
| `warn` | `similar:fetch_failed` | GitHub search API down → section masquée silencieusement |
| `warn` | `similar:rate_limited` | HTTP 403 GitHub (60 req/h dépassé) → section masquée + retry désactivé 1h |
| `warn` | `url:truncated` | Body Markdown > 7 000 chars, troncature appliquée |
| `error` | `capture:install_failed` | `installConsoleCapture()` a throw (très rare) |
| `error` | `clipboard:open_failed` | `window.open` retourne `null` (popup bloqué) → fallback copy URL clipboard |

**Règle** : aucun log dans les hot paths (rendu live du preview Markdown, debounce typing).

### GitHub Action — script Node

Le script `triage-feedback.mjs` log en stdout (visible dans l'onglet Actions UI) :
- `[triage] received issue #N "title"`
- `[triage] fetched 47 open issues for context`
- `[triage] calling claude-haiku-4-5 (~5k tokens)`
- `[triage] parsed JSON: severity=high area=synthesis is_actionable=true`
- `[triage] applied labels: severity:high, area:synthesis, triage:claude-analyzed`
- `[triage] posted comment (412 chars)`

En cas d'erreur :
- `[triage] ERROR: claude API timeout` → label `triage:needs-review` + sortie 0 (workflow ne fail pas, traçabilité préservée).
- `[triage] ERROR: JSON parse failed: <raw>` → label `triage:parse-error` + sortie 0.

**Pas de logging du body issue complet** (peut contenir des données utilisateur — RGPD light).

## Anti-régression

### Tests dédiés à éviter les régressions futures

| Risque | Test |
|---|---|
| Un dev refactor `useSimilarIssues` vers `api.get(...)` → leak header `X-LevelUp-Title` à GitHub | `queries.test.ts` : assertion sur `fetch` mocké que `credentials: 'omit'` est passé et qu'aucun header LevelUp n'apparaît dans la requête |
| `global-capture/buffers` casse l'output console réel | `buffers.test.ts` : spy sur `console.error` original, vérifier qu'il est toujours appelé après wrap |
| Ring buffer leak (croît indéfiniment) | `buffers.test.ts` : push 25 entries, expect length === 20 (FIFO) |
| URL avec query params sensibles fuite dans le body GitHub | `buffers.test.ts` : déclencher fail-fetch sur `/api/v1/foo?token=secret` → entrée stockée a `url === '/api/v1/foo'` (pas de `?token=...`) |
| HMR re-instrumente le wrap (double capture) | `install.test.ts` : 2 imports successifs avec flag `globalThis` préservé → 1 seule install, `console.error('x')` produit 1 entry seulement |
| GitHub-search query 422 sur titres avec opérateurs (`:`, `+`, `"`) | `queries.test.ts` : 5 cas de titres "exotiques" → URL générée n'a aucun caractère réservé non escapé |
| Format body Markdown change accidentellement | `buildIssueUrl.test.ts` : snapshot inline du body généré pour 3 cas représentatifs (bug avec erreurs, idée sans erreurs, question avec filtres) |
| Troncature URL casse le body | `buildIssueUrl.test.ts` : input ≥ 8 000 chars → output ≤ 7 000, mention `…[truncated]`, ordre de troncature vérifié |
| Heuristiques classifyFeedback drift | `classifyFeedback.test.ts` : table-driven (15+ cas) couvrant toutes les règles + cas limites (description vide, multi-mots-clés contradictoires, "crash" sans erreur console → high) |
| Routes mal mappées dans l'area heuristique | `classifyFeedback.test.ts` : test pour chaque pattern URL listé dans la matrice (synthesis, explorer, squad, stats/sessions, stats/timeseries, stats/history, palmares*, etc.) |
| Anti-spam contourné en mode privé | `rateLimit.test.ts` : `localStorage` stub qui throw → fail-open silencieux (`getRemainingSubmits() === 5`), log warn 1× |
| Position drawer chevauche AssetDrawer sur petit viewport | `e2e/feedback-drawer.spec.ts` : Playwright en 768×600 → vérifier `bounding-box` non-overlap |
| Mini-tab visible sur mobile (régression) | `e2e/feedback-drawer.spec.ts` : viewport 375×667 → `expect(miniTab).toBeHidden()` |
| Drawer reste ouvert après navigation route | `FeedbackDrawer.test.tsx` : simuler navigate → expect `isOpen === false` (à confirmer côté store, ajouter listener si manquant) |
| Popup blocker casse le submit | `FeedbackDrawer.test.tsx` : mock `window.open` retourne `null` → fallback `navigator.clipboard.writeText(url)` appelé + toast affiché |
| AssetDrawer cassé par le sibling FeedbackDrawer | `e2e/asset-drawer.spec.ts` (existant si présent) : doit toujours passer après merge |

### Lint anti-régression

Ajouter dans le `tools/lint-no-hardcoded-colors.mjs` ou un nouveau lint custom :
- **Bannir `import { api } from ... ` dans `features/feedback-drawer/queries.ts`** via une assertion grep en CI (script bash 5 lignes dans le workflow `ci.yml`). Évite le drift sécu.

### Couverture cible

- **Helpers purs** : 100 % (faciles à tester, pas d'excuse).
- **Store** : 100 % des actions.
- **Component** : tests d'intégration sur les 5 interactions clés (open, close, type select, submit, escape).
- **e2e Playwright** : 1 happy-path (ouvrir, remplir, valider URL).
- **Total cible** : ≥ 85 % sur le module `feedback-drawer/` (configurable via `vitest.config.ts coverage thresholds`).

## Thème dark / light

### Tokens à utiliser (oklch dans `globals.css`)

Le thème est piloté par `data-theme="dark"|"light"` sur `<html>` (pas de `dark:` Tailwind variant — Tailwind 4 + shadcn-style).

| Élément | Token classe | Note |
|---|---|---|
| Fond panneau & mini-tab | `bg-popover` | Dark : `oklch(0.269 0 0)`. Light : presque blanc. |
| Bordure | `border-border` | Suffisamment contrasté dans les 2 thèmes |
| Texte principal | `text-popover-foreground` | — |
| Texte secondaire (notes, placeholder) | `text-muted-foreground` | — |
| Bouton actif (segmented control) | `bg-accent text-accent-foreground` | — |
| Bouton inactif | `text-muted-foreground hover:bg-accent/50` | — |
| **Liens "Issues similaires"** | `text-info hover:underline` | Token `--info` (oklch bleu) — **pas de hex** |
| Focus ring `<input>`/`<textarea>` | `focus-visible:ring-2 focus-visible:ring-ring` | Token `--ring` adapté aux 2 thèmes |
| Bloc code ``` du preview | `bg-muted text-muted-foreground` | — |
| Badge sévérité (preview) | Tokens sémantiques *uniquement* — voir ci-dessous | Pas de rouge/vert hardcodé |

### Edge cases dark/light

| Risque | Mitigation |
|---|---|
| `shadow-xl` est moins visible en dark mode (manque de contraste fond → fond) | Doubler avec `ring-1 ring-border` pour assurer la silhouette du panneau dans les 2 thèmes |
| Mini-tab `border-r-0` se confond avec le bord viewport en dark | Confirmé OK avec `bg-popover` qui est plus clair que `bg-background` en dark — vérifier visuellement |
| `<details>` natif HTML : marker triangle hérite `currentColor` mais peut être invisible | Forcer `text-foreground` sur le `<summary>` |
| Badges sévérité (`critical`/`high`) tentation de rouge hex | Utiliser `bg-destructive/15 text-destructive` (token sémantique `--destructive` existant) ou afficher le texte sans fond coloré |
| Auto-detect OS theme avant store hydration → flash | Le `theme-provider.tsx` existant gère déjà ça pour l'app entière. Le drawer hérite. Pas de logique custom. |

### Tests dark/light dans la checklist UI

Voir checklist UI ci-dessous : un toggle de thème pendant que le drawer est ouvert doit re-rendre proprement (pas de FOUC, pas de couleur figée).

## Performance

### Risques identifiés

| Risque | Mitigation |
|---|---|
| Live preview Markdown re-render à **chaque keystroke** sur titre/description (texte potentiellement long) | `useMemo(() => buildBodyPreview(input, context), [input, context])` + `useDeferredValue(description)` (React 19) pour découpler la frappe du render preview |
| `global-capture` wrap appelé sur **chaque** `console.error/warn` et **chaque** `fetch` | Push sync dans Array borné — coût ~µs négligeable. Wrap `fetch` n'instrumente que la branche fail (response.ok=false) |
| Hot-Module-Reload re-applique le wrap → double instrumentation, ring buffer reçoit chaque message 2× | Flag idempotent stocké sur `globalThis` (survit au HMR contrairement à un flag module-level) — voir § Idempotence sous HMR Vite. Couvert par `install.test.ts` |
| Drawer reste monté en permanence (comme AssetDrawer) → DOM sub-tree existe même fermé | Acceptable : juste un form HTML, pas de queries firing tant que `isOpen === false` (gating dans `queries.ts` via `enabled: isOpen && title.length >= 3`) |
| `useSimilarIssues` fire à chaque keystroke avant debounce | `enabled: title.length >= 3 && isOpen` + `staleTime: 60_000` (cache 1 min sur le même titre) |
| Icône SVG inlinée re-créée à chaque render | Composant constant `<ChatBubbleIcon />` défini **hors** du composant principal (pattern AssetDrawer ligne 135) |
| Sélecteurs Zustand qui retournent un objet → re-render sur toute mutation du store | Sélecteurs **scalaires** : `useAppShellStore(s => s.locale)` + `useAppShellStore(s => s.currentTitleSlug)` séparément (pattern AssetDrawer ligne 11-12) — pas `s => ({ locale, slug })` |

### Pattern debounce

Pas de hook centralisé dans le projet. Suivre le pattern de [useGamertagSuggestions.ts](apps/web/src/components/ui/useGamertagSuggestions.ts) (state-based avec `useState` + `useEffect` 250-500 ms) plutôt que `useRef<setTimeout>` ad-hoc — plus testable, plus aligné sur les features récentes.

### Test perf simple

`FeedbackDrawer.test.tsx` : compter les renders avec un compteur ref, taper un titre de 30 chars → expect ≤ 35 renders (1 par keystroke + quelques dérivés acceptables). Au-delà, useMemo manquant.

## Observabilité côté mainteneur (V1)

L'app n'a aucune télémétrie front (cf. ADR-0009 — expvar planifié côté Go API uniquement). On exploite **GitHub natif** pour l'observabilité du feedback. **Scope V1 : minimum viable**, le digest hebdo est reporté en V1.1.

### Badge README "Open feedback issues"

Ajouter dans le `README.md` racine :
```markdown
[![Feedback issues](https://img.shields.io/github/issues-search/JGtm/LevelUp?query=label%3Afeedback%20is%3Aopen&label=feedback)](https://github.com/JGtm/LevelUp/issues?q=is%3Aissue+is%3Aopen+label%3Afeedback)
```
Visible immédiatement sur la page repo, lien direct vers la liste filtrée.

### Anti-spam côté front (limitation soft)

Pour éviter qu'un user clique 100× le bouton "Ouvrir sur GitHub" :
- LocalStorage key `levelup-feedback-submits` = array de timestamps des submits.
- Avant submit, expire les > 1h, count restantes (helper pur `rateLimit.ts`).
- Si ≥ 5 → bouton désactivé + message "Merci, tu as déjà envoyé 5 retours dans la dernière heure".
- Si `localStorage` indisponible (mode privé strict) → fail-open silencieusement, le rate-limit est best-effort.
- Pas une vraie protection (le user peut clear localStorage), mais bloque 99 % des cas accidentels.

### Métriques coût Anthropic — log seulement en V1

Le script `triage-feedback.mjs` log en stdout `[triage] usage: input_tokens=N output_tokens=N` (visible dans l'onglet Actions). Pas d'agrégation automatique en V1 ; check manuel mensuel sur Actions UI ou console.anthropic.com. Estimation : ~$0.001/issue × volume mensuel.

### Fichiers à créer (observabilité V1)

| Fichier | Rôle |
|---|---|
| `README.md` | Ajout du badge feedback (modif, pas création) |

> Pas de nouveau workflow / script en V1. Le rate-limit côté front est porté par `rateLimit.ts` (déjà listé dans les fichiers frontend).

### V1.1 (reporté) — Weekly digest

Pour mémoire, à shipper après stabilisation V1 (≥ 4 semaines de données réelles) :

- `.github/workflows/weekly-feedback-digest.yml` — cron `0 9 * * MON`.
- `.github/scripts/weekly-digest.mjs` — génère le rapport (volume, breakdown area/severity/type, top 5 engagement, qualité triage IA, liste needs-review, coût Anthropic agrégé).
- Issue épinglée `📊 Feedback Weekly Digest` créée auto au premier run, mise à jour en commentaire.
- Seuil d'alerte coût : > $1 / mois → issue auto `cost:anthropic-spike`.

Le décalage en V1.1 réduit la surface de bugs au merge initial et permet de calibrer les sections du digest sur des données réelles plutôt que théoriques.

## Notes de qualité

- **Couleurs** : que des tokens sémantiques. Vérifier qu'aucun `#xxx` ni `text-red-*` dans le diff (règle 20 CLAUDE.md). Lint custom strict ratchet 0 (`tools/lint-no-hardcoded-colors.mjs`).
- **Taille fichiers** : `FeedbackDrawer.tsx` < 250 lignes (sinon découper le form en sous-composants). Helpers purs < 80 lignes par fonction (règle 13). Modules < 500 lignes (règle 14).
- **A11y** : `role="complementary"`, `aria-label`, `aria-expanded`, `aria-hidden`, focus auto sur `<input>` titre à l'ouverture.
- **PII** : aucun token, aucun cookie, aucun email envoyé à GitHub. User-Agent (publique) seul. Le body de l'issue est public côté repo. **URLs des requêtes échouées toujours strippées des query params** (`url.split('?')[0]`) à la capture (cf. § Métadonnées techniques avancées) — évite le leak de tokens/gamertags/IDs en clair. Test dédié dans `buffers.test.ts`.
- **Rate-limit GitHub search** : 60 req/h/IP non-auth. Avec debounce 500 ms + queryKey react-query, on reste en deçà même en usage intensif.
- **Stack trace defensive** : `console.error(...)` peut recevoir n'importe quoi → extraire `.stack` uniquement si `arg instanceof Error`, sinon stringify.

## Vérification end-to-end

```bash
# 1. Build i18n + types front
cd apps/web
node scripts/build_i18n_manifests.mjs
npm run typecheck

# 2. Tests unitaires + composants (Vitest + MSW + @testing-library/react)
npm run test:run -- src/features/feedback-drawer src/lib/global-capture

# 3. Coverage cible ≥ 85 % sur feedback-drawer
npm run test:coverage -- src/features/feedback-drawer

# 4. Lint custom (couleurs ratchet 0)
node ../../tools/lint-no-hardcoded-colors.mjs

# 5. Test e2e Playwright (happy path)
npm run test:e2e -- feedback-drawer

# 6. Tests script GitHub Action (Node built-in test runner)
node --test .github/scripts/triage-feedback.test.mjs

# 7. Dev server + tests manuels
npm run dev
```

Le **CI existant** ([.github/workflows/ci.yml](.github/workflows/ci.yml)) lance déjà `typecheck + lint + lint:fields + build + test:coverage` à chaque PR — les nouveaux tests Vitest sont pickés automatiquement. Ajouter en plus dans le workflow CI le step `npm run test:e2e` si pas déjà présent.

### Checklist UI (browser)

**Fonctionnel**
- [ ] Mini-tab feedback visible sous AssetDrawer avec gap notable (~30 px) en 1080p ET en 768×600.
- [ ] Clic mini-tab → panneau slide-in fluide.
- [ ] `Escape` → ferme.
- [ ] Saisir un titre → après 500 ms, section "Issues similaires" apparaît si match.
- [ ] Toggle "joindre infos techniques" → preview Markdown live.
- [ ] Changer la période sur la page → reflétée dans le preview.
- [ ] Provoquer une erreur (`fetch('/api/v1/inexistant?secret=xyz')` console) → apparaît dans "Erreurs console" + "Requêtes échouées" **avec URL `/api/v1/inexistant` (pas de query param `secret=xyz` dans l'aperçu)**.
- [ ] Type "Bug" + `TypeError` → preview montre `bug, severity:critical, area:<route>`.
- [ ] Type "Bug" + description "ça crash sur la page" sans erreur → `bug, severity:high`.
- [ ] Type "Idée" sans erreur → `enhancement, severity:low`.
- [ ] Clic "Ouvrir sur GitHub" → nouvel onglet, formulaire prérempli, labels appliqués.
- [ ] Body très long (8 000 chars description) → troncature, mention `…[truncated]`, redirection OK.
- [ ] Popup blocker actif → fallback : URL copiée dans le clipboard + toast "Lien copié, colle-le dans un onglet GitHub".
- [ ] Title contient `:`, `+`, `"` → recherche d'issues similaires fonctionne sans 422.
- [ ] Mobile (`< 640 px`, viewport 375×667) : mini-tab caché.
- [ ] FR + EN : tous libellés traduits.
- [ ] Anti-spam : 6e submit dans l'heure → bouton désactivé.
- [ ] Mode privé strict (localStorage off) → drawer fonctionne, anti-spam fail-open silencieux.

**Thème dark/light**
- [ ] Light theme : panneau, mini-tab, bordures, texte, focus ring lisibles.
- [ ] Dark theme : idem + shadow `ring-1 ring-border` confirme la silhouette.
- [ ] Toggle theme **avec drawer ouvert** → re-rendu propre, pas de FOUC ni couleur figée.
- [ ] Liens "Issues similaires" : `text-info` visible et lisible dans les 2 thèmes.
- [ ] `<details>` summary triangle visible dans les 2 thèmes.
- [ ] Inspect DOM : aucune classe `text-red-*`, `bg-green-*`, `text-[#...]` dans l'arbre du drawer (DevTools → Computed → filter par sélecteur).

**Performance**
- [ ] React DevTools Profiler : taper 30 chars dans le titre → ≤ 35 commits, durée totale < 200 ms.
- [ ] `useSimilarIssues` ne fire pas tant que `title.length < 3` (Network panel).
- [ ] HMR : sauvegarder `lib/global-capture/buffers.ts` 3× → ring buffer ne reçoit pas de doublons (test : `console.error('x')` puis `getRecentConsoleEntries()` → 1 entry, **flag `globalThis.__levelup_global_capture_installed__` reste true**).
- [ ] Drawer fermé : aucune query GitHub en background (Network panel filter `api.github.com`).

**Sécurité**
- [ ] Inspect Network : appel `api.github.com` SANS header `X-LevelUp-Title`, SANS cookie de session LevelUp.
- [ ] Inspect Network : Response GitHub n'est pas en `credentials: include` (l'onglet Headers n'affiche pas `Cookie`).

### Checklist GitHub Action (après merge + provisioning secrets)

- [ ] Workflow `sync-labels` s'est exécuté → labels présents dans Settings → Labels.
- [ ] Créer une issue test via le drawer en local.
- [ ] Workflow `Triage feedback issues` se déclenche dans l'onglet Actions.
- [ ] L'issue reçoit dans les 30 s : commentaire Claude + labels affinés.
- [ ] Tester un cas spam (titre vide ou aléatoire) → label `triage:needs-review`.
- [ ] Vérifier consommation Anthropic dans la console (~$0.001/issue).
- [ ] Stdout du workflow log avec préfixe `[triage]` — pas de body issue logué (RGPD light).
- [ ] Issue avec body cassé / non-Markdown → fallback `triage:parse-error`, workflow exit 0 (pas d'échec).

### Checklist observabilité mainteneur V1 (après 1 semaine de prod)

- [ ] Badge README "Open feedback issues" affiche le bon compteur.
- [ ] Onglet GitHub Issues filtré sur `label:feedback` est consultable et lisible.
- [ ] Logs Actions du workflow `Triage feedback issues` montrent `[triage] usage: input_tokens=N output_tokens=N` pour chaque issue.
- [ ] Volume hebdo cohérent (pas de spam manifest) — sinon revoir le rate-limit côté front.

> Critère de passage en V1.1 : ≥ 4 semaines stables, ≥ 20 issues feedback réelles → impl du weekly digest sur données calibrées.

### Avant le commit (CLAUDE.md — règle obligatoire)

- [ ] Ajouter une entrée dans [.ai/thought_log.md](.ai/thought_log.md) :
  - Date `[2026-05-05]`
  - Titre : "Drawer feedback + GitHub Action triage IA"
  - Statut (En cours / Complété)
  - Décision technique principale (URL préremplie + Action Claude post-création, fetch GitHub direct sans wrapper api, sanitize URL query strings côté capture, idempotence HMR via `globalThis`)
  - Résultats observés
  - Conclusion / prochaine étape

## Changelog plan

### 2026-05-05 — Revue d'architecture (passe 1)

Corrections appliquées après cross-check codebase :

1. **Heuristique area** : matrice URL refondée — `engagement` retiré (pas de route propre), `session` corrigé en `stats/sessions`/`stats/timeseries`/`stats/history`, ajout des routes manquantes (`palmares*`, `home`, `media`, `career`, `notifications`, `objectifs`, `profile/citations`, `match_history`, `match_view`, `settings`, `meta`).
2. **Heuristique severity** : règles ordonnées par priorité explicite, ajout `severity:high` sur description "crash"/"perd ma progression"/"impossible"/"bloqué" même sans erreur console.
3. **Renommage `lib/bootstrap.ts` → `lib/global-capture/install.ts`** + module séparé `buffers.ts` : évite la collision sémantique avec le bootstrap auth/`/bootstrap` du projet.
4. **Idempotence HMR** : flag stocké sur `globalThis.__levelup_global_capture_installed__` (pas module-level, sinon HMR Vite re-instrumente). Test dédié.
5. **Sanitization PII** : `console-capture` strippe systématiquement les query strings des URLs fetch capturées (`url.split('?')[0]`) avant stockage. Test anti-régression.
6. **Manifest i18n renommé** : `feedback.toml` → `feedback_drawer.toml` pour cohérence avec `asset_drawer.toml`.
7. **Anti-spam — fichier dédié** : ajout `rateLimit.ts` + `rateLimit.test.ts` (helper pur, fail-open sur `localStorage` indisponible).
8. **`resetCaptureBuffersForTests()`** exposé pour les tests Vitest, pattern aligné sur `_logger.ts::_resetForTests`.
9. **GitHub-search query encoding** : helper `escapeSearchQuery` dédié + test 5 cas (caractères réservés `:`, `+`, `"`, `(`, `)`, `/`).
10. **Repo privé** : commentaire de migration documenté dans `queries.ts` (chemin futur via endpoint backend `/api/v1/feedback/search-issues`).
11. **Weekly digest reporté en V1.1** : V1 garde seulement le badge README + log stdout des coûts. Critère de passage en V1.1 : ≥ 4 semaines stables, ≥ 20 issues réelles.
12. **Actions GitHub pinnées par SHA** (supply-chain hygiene, secret `ANTHROPIC_API_KEY` exposé). Tags en commentaire pour lisibilité.
13. **Estimation coût Anthropic** corrigée : ~$0.001/issue (pas $0.0005), seuil d'alerte recalibré 1000 issues/mois.
14. **Test e2e mobile** : viewport 375×667 → mini-tab caché (régression facile sinon).

Aucun élément retiré, uniquement des précisions/corrections. Scope V1 inchangé hormis le digest reporté.
