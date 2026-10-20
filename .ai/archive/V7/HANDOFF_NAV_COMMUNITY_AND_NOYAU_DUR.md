# HANDOFF — Refonte navigation (`/community`) + narration card « Noyau dur »

> Daté 2026-06-28. Reprise par l'utilisateur (avec ses feedbacks) puis poursuite.
> Plan détaillé associé : `C:\Users\Guillaume\.claude\plans\zany-growing-marble.md`.

## TL;DR — où on en est

- **Refactor nav** (rename `/palmares`→`/community` + re-nesting + unification onglets) : **Phases 1 & 2 FAITES et vérifiées**, commitées sur la branche `refactor/nav-community-routes`. **NON poussé, NON déployé.** **Phase 3 (la plus grosse) reste à faire.**
- **Card « Noyau dur »** : narration jugée trop faible. **8 mockups HTML livrés** (`.ai/mocks/relations/noyau-dur-cards.html`), **en attente du choix de l'utilisateur** avant implémentation.
- **Déjà en prod** (origin/main) : la refonte Relations v2 (pills) + v3 (graphe cumulé/FDA/store de préférences). Rien à refaire dessus.

---

## 1. Refactor navigation — branche `refactor/nav-community-routes`

**Pourquoi** : l'URL `/palmares` ne correspondait pas au libellé « Communauté » ; sous-navigation incohérente (segments de route vs `?tab=`) ; imbrications hétérogènes. Choix utilisateur : **cohérence complète**, préfixe **`/community`** (anglais, comme les autres segments). Backend API `/pages/palmares/*` **inchangé** (découplé de l'URL front). Anciennes URLs **préservées par redirection** (`beforeLoad → throw redirect`, modèle `routes/.../objectifs/index.tsx`).

**Convention cible** : sous-vue de page = **segment de route** ; query-param = **filtres/paramètres** d'une même vue (Explorer, Compare, Squad) ; store = toggles éphémères.

### Phase 1 — `ae6a615cf` (FAIT)
Rename `/palmares` → `/community`. Routes déplacées vers `routes/players/$playerSlug/community/{index,relations,prestige,compare}.tsx` ; anciennes (`palmares/*`, `compare.tsx`) devenues des redirects. Callsites MAJ : `isCommunityPath` (shellNavigation.ts), `navL1Sections.tsx`, `NavL2.tsx` (COMMUNITY_*), `pageTitle.ts`, `classifyFeedback.ts`, deep-links Explorer/Career → `/community/compare`, `PLAYER_PRIMARY_NAV_ITEMS` (label « Palmarès »→« Communauté »). Fix en passant : notif `season_pass_level` → `/career/season-pass` (était `/palmares`, périmé). Tests MAJ (NavL1, navigation, classifyFeedback, shellNavigation).

### Phase 2 — `188126751` (FAIT)
Re-nesting : `/synthesis`→`/stats/synthesis` ; `/citations`+`/commendations`→`/career/*`. Redirects + nav alignée (navL1Sections, NavL2 CAREER_TABS, pageTitle, shellNavigation, classifyFeedback). **Détail clé** : Synthèse exclue de la barre de filtres NavL2 (`detectSection` retourne `null` pour `/stats/synthesis`) car la page gère **ses propres** pills période/saison — sinon double barre.

### Phase 3 — À FAIRE (la plus grosse, restructuration de composants)
Convertir les onglets `?tab=` en segments de route, modèle `apps/web/src/features/ascension/AscensionLayout.tsx` (layout = barre d'onglets `Link`+`useMatchRoute` + `<Outlet/>` ; routes enfants ; `Outlet context` pour la donnée partagée). Cibles :
- **Séries temporelles** `stats/timeseries.tsx` + `TimeseriesPage.tsx` → layout + enfants `stats/timeseries/{index(summary),distributions,progression}.tsx`. **Risque FAIBLE** : la barre de filtres est *shell-owned* (NavL2 + `useSoloFilterStore`), pas dans la page → seul `data` (useTimeseriesPage) à passer via `<Outlet context>`. La requête `useExplorerMatches` (progression-only) migre dans le child progression (supprime le hack `enabled: activeTab===`). Redirect `?tab=` via `beforeLoad`. Vérifié : `PERSONAL_STATS_RE` de NavL2 ne matche pas `/stats/timeseries/*` → barre filtres conservée.
- **Match View** `matches/$matchId.tsx` + `MatchViewPage.tsx` → `$matchId.tsx` est **déjà parent** (`$matchId/replay.tsx` existe) ; ajouter `<Outlet/>`, enfants `$matchId/{index(summary),details}.tsx`, `Outlet context` = `data` + champs dérivés, garder `loader`+`RouteCapabilityGate`. Redirect `?tab=details`.
- **Inchangé** : Explorer (`mode`+filtres), Compare (`target`/`target2`), Squad (`session`/`teammates`) — ce sont des paramètres, pas des sous-vues.
Détail complet : voir le plan + le rapport de l'agent Plan (dans l'historique de la session).

### État vérif (Phases 1+2)
`vite build` (régénère `routeTree.gen.ts`) OK · `tsc -b` OK · **vitest 2049 pass** · eslint 0 erreur · lint-no-hardcoded-colors/fields + cross-feature + knip sous plafonds. **Seul échec : `HomeSpartanIdentityBanner.test.tsx`** — **PRÉ-EXISTANT**, hors de mon changeset (commit « Personnalisateur Spartan » d'un autre agent). À ignorer pour ce refactor.

### Pièges / à savoir
- **`routeTree.gen.ts` se régénère via `npm run build`** (pas de CLI `tsr` installé ; le plugin TanStackRouterVite tourne au build). Toujours rebuild AVANT `tsc -b` après un ajout/déplacement de route.
- **Commit étranger sur la branche** : `fb43e242c` (« chore(spartan): resync assets H5 ») a été committé par un AUTRE agent sur `refactor/nav-community-routes` (dépôt partagé, repo posé sur cette branche). À RÉCONCILIER avant merge : vérifier s'il doit rester ou partir ailleurs.
- **Dépôt partagé multi-agents** : TOUJOURS `git status` avant de stager ; **jamais `git add -A`** (commit sélectif par liste de fichiers). Les chemins avec `$playerSlug` doivent être **entre guillemets simples** en PowerShell (sinon `$playerSlug` interpolé en vide).
- Hooks Go (pre-commit go-vet) : **prélude CGO requis** (`$env:Path="C:\msys64\ucrt64\bin;"+$env:Path ; $env:CGO_ENABLED="1"`) même pour un commit front-only.
- vitest : `dangerouslyDisableSandbox=true` (le sandbox casse les workers).

---

## 2. Card « Noyau dur » — narration (EN ATTENTE DE DÉCISION)

L'utilisateur trouve la KPI card `CoreSummaryCard` (résumé en haut de la page Relations) trop descriptive (compte + WR moyen + 2 noms). **8 mockups** de narration : ouvrir `.ai/mocks/relations/noyau-dur-cards.html`.

- **Code à modifier** : `CoreSummaryCard` dans `apps/web/src/features/palmares/PalmaresRelationsPage.tsx` (≈ lignes 137-197).
- **Reco proposée** : n°1 « L'effet noyau » (le lift WR vs moyenne, coloré) — combo gagnant avec le « roc » de la n°2 en sous-titre. Sans backend : n°2 (phrase + roc) ou n°7 (mini-classement top 3).
- **Données dispo sans backend** : `avg_kda_with`, `teammate_win_rate`, `total_matches` par relation, `first_seen_at`/`last_seen_at`, badges, `overview.{core_count, top_ally}`.
- **Nécessitent un petit backend (DIFFÉRÉ jusqu'ici)** : le **lift** (n°1/8) → `overview.player_win_rate` (1 requête W/total joueur + champ DTO + maj des mocks de `RelationsRepository`) ; le **poids** (n°3) → nombre total de matchs du joueur. Le churn = ajout d'une méthode à l'interface `port.RelationsRepository` répercuté sur tous ses mocks de test.
- **Prochaine étape** : l'utilisateur choisit le(s) numéro(s) → implémenter dans `CoreSummaryCard` (+ petit backend si n°1/3/8).

---

## 3. État déploiement (prod = origin/main)

- **origin/main** (déployé via « Deploy to VPS » sur push) contient déjà : pills Relations v2 (`f29a3c971`) **+ v3-fixes** (graphe cumulé/FDA/store préférences, `9e44408f1`) **+** le travail des autres agents (KDA NET, home-hero, classement, Spartan, H5 rang/XP…). **Rien à refaire / re-déployer sur ces sujets.**
- **PAS déployé** : le refactor nav (Phases 1+2 sur la branche). Le merge `main` + push **déploie en prod** → **sur accord utilisateur uniquement**, après Phase 3 (ou décider de livrer Phases 1+2 d'abord).
- prod saine au dernier check : `/readyz` duckdb ok, `/health` ~1786 matchs.

---

## 4. Reprendre proprement

1. `git -C <repo> checkout refactor/nav-community-routes` (vérifier `git status` ; le mock noyau-dur est untracked).
2. **Card Noyau dur** : appliquer le choix de narration de l'utilisateur dans `CoreSummaryCard` (+ backend `player_win_rate` si lift retenu).
3. **Phase 3 nav** : suivre le plan (layout + enfants + `Outlet context`, modèle AscensionLayout) pour Timeseries puis Match View ; redirects `?tab=`.
4. Vérif à chaque étape : `npm run build` (regen routeTree) → `npm run typecheck` → `npx vitest run` (dangerouslyDisableSandbox) → eslint + `node tools/{lint-no-hardcoded-colors,lint-no-hardcoded-fields,knip-ratchet}.mjs`. Ignorer l'échec pré-existant `HomeSpartanIdentityBanner`.
5. Commit sélectif par phase (jamais `add -A`), prélude CGO. Push + merge main (= déploiement) **sur accord**.

Réfs : [[project_relations_page_redesign]] · plan `zany-growing-marble.md` · mocks `.ai/mocks/relations/{relations-mockups-v2.html, noyau-dur-cards.html}`.
