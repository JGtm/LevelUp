# Axe 4 — Structure & patterns React/TS

Date : 2026-04-29
Branche : feat/multi-title-static-fs-rescope
Périmètre : apps/web/src/{features,components,lib,stores,routes,hooks}/

## Synthèse (3-5 lignes max)

Stack moderne et globalement saine : TanStack Router file-based, TanStack Query systématique, Zustand pour le state global, MSW pour les tests, types API auto-générés depuis OpenAPI. La séparation features/components/lib est respectée à 90 % et les patterns interdits (`dangerouslySetInnerHTML`, imports relatifs profonds, `@ts-ignore` non commentés) sont absents. Trois faiblesses structurelles : 1) `HomePage.tsx` (1158 L) et 5 autres pages > 400 L mélangent fetch + transform + render + sous-composants locaux ; 2) `useFieldLabel`/`useFormatLabel` n'ont pas d'usage réel — 12 features le réimplémentent en `labelOf` ad-hoc et hardcodent FR ; 3) aucune route TanStack n'utilise `loader:` — tout passe par `useQuery` au mount, ce qui prive du suspense pré-route prévu par TanStack v1. Tests vitest présents (86 fichiers) mais asymétriques : Squad et charts couverts, Notifications/Engagement/Asset-drawer/SessionCompare quasi nus.

## Matrice : structure features

| Feature | Fichiers | Plus gros (L) | Tests | Hooks dédiés | Loader TanStack |
|---|---|---|---|---|---|
| home | 11 | HomePage 1158 | oui (3) | useHomePage, useSeasonPassPreview | non |
| settings | 8 | SettingsPage 903 | oui (4) | useSettings, useUpdateSettings | non |
| lab | 8 | LabPage 937 | oui (1) | useLabContracts, useLabDiagnostics, useLabResources | non |
| match-view | 6 | MatchViewPage 600 | partiels (1) | useMatchView, useMatchNeighbors | non |
| squad | 14+ (v2 séparé) | SquadLayout 530 | oui (5) | useTeammates, useSquadV2 | non |
| timeseries | 5 | TimeseriesPage 480 | partiels (3) | useTimeseriesPage, useCombatYieldHistory | non |
| synthesis | 4 | SynthesisPage 471 | oui (2) | useSynthesisPage | non |
| palmares | 6 | SeasonPassPage 456 | oui (2) | useSeasonPassPage, usePalmaresRelations | non |
| explorer | 4 | ExplorerPage 381 | oui (2) | useExplorerMatches, useExplorerPlayer | non |
| media | 8 | MediaPage 385 | oui (2) | useMedia, useMediaRail | non |
| match-history | 3 | MatchHistoryTable 314 | oui (1) | useMatchHistory | non |
| career | 8 | CareerChartsSection 206 | oui (2) | useCareerPage, useCareerTopMatches | non |
| notifications | 10 | NotificationsBell 289 | non | useNotifications, useUnreadCount | non |
| engagement | 4 | EngagementCurve 195 | partiels (1) | useEngagement* | non |
| session-detail | 1 | SessionDetailPage 444 | oui (1) | useSessionDetailPage | non |
| session-compare | 1 | SessionComparePage 393 | non | useSessionCompare | non |
| asset-drawer | 1 | AssetDrawer | non | useAssetMaps, useAssetWeapons | non |
| compare | 3 | CompareDrawer | oui (1) | useComparePlayer, useComparePrefetch | non |
| friends | 2 | AddFriendFlow | oui (1) | — | non |
| setup | 3 | SetupPage 484 | oui (1) | useDeviceFlow, useJobStatus | non |
| prestige | 4 | CreateChallengeForm 406 | non | useChallenges | non |

## Constats

### [BLOQUANT] HomePage.tsx — 1158 L mélange fetch, transformation, helpers et 6 sous-composants locaux

- **Fichier:ligne** : `apps/web/src/features/home/HomePage.tsx:1-1158`
- **Extrait** : 1158 lignes contenant `KPICard`, `SerieTile`, `HighlightTile`, `ChevronUpIcon`, `ChevronDownIcon`, `OutcomeBar`, `SessionCarouselCard` (animation custom 80 L lignes 196-244), `HomeSkillPeakCard`, `resolveSkillPeakState`, plus le rendu principal `HomePage()` lignes 466-1158.
- **Problème** : God file qui dépasse de 2x le seuil 500 L documenté dans CLAUDE.md. Mélange lecture (`useHomePage`, `useSeasonPassPreview`, `useFieldMappings`), transformation (`getKPIText`, `formatSessionDate`, `formatSessionDuration`, `resolveSkillPeakState`), animations DOM impératives (`SessionCarouselCard` manipule `style.transform/opacity` via `requestAnimationFrame`), et 7 blocs JSX > 100 L (Spartan banner, KPI grid, BattlePass, Sessions, Highlights, Recent matches, Favorites). Aucun de ces sous-composants locaux n'est testable indépendamment. Ré-introduit le pattern « God function » que CLAUDE.md proscrit.
- **Action** : Extraire vers `features/home/{HomeSpartanBanner,HomeKPIs,HomeSessionsCard,HomeHighlightsGrid,HomeMatchesCarousel}.tsx`. `SessionCarouselCard` mérite son propre fichier + test (l'animation impérative est piégeuse).

### [BLOQUANT] SettingsPage.tsx ligne 67 — `useState` détourné en mutable ref → bug latent

- **Fichier:ligne** : `apps/web/src/features/settings/SettingsPage.tsx:67`
- **Extrait** :
  ```tsx
  const saveTimerRef = useState<ReturnType<typeof setTimeout> | null>(null)
  // ...
  if (saveTimerRef[0]) clearTimeout(saveTimerRef[0])
  saveTimerRef[0] = setTimeout(() => setSaveStatus(null), 2000)
  ```
- **Problème** : `useState(null)` retourne `[value, setter]`, pas une ref mutable. Le code mute `saveTimerRef[0]` directement (lignes 92, 101, 102, 106, 107) — React ne re-render pas, le setter n'est jamais appelé, et la mutation d'un tuple stable (recréé à chaque render) ne persiste **pas** entre rendus. Le clearTimeout fonctionne uniquement par hasard parce que le composant n'est pas démonté entre les saves. C'est un bug latent qui leakera des timers si le composant se ré-instancie (par exemple navigation cross-tab) et les tests ne couvrent pas ce scénario.
- **Action** : Remplacer par `const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)` et accéder via `.current`.

### [BLOQUANT] Aucune route TanStack n'utilise `loader:` — tout fetch dans le composant après mount

- **Fichier:ligne** : `apps/web/src/routes/players/$playerSlug/home.tsx:7-9`, idem pour les 30 autres fichiers de `routes/`
- **Extrait** :
  ```tsx
  // home.tsx — squelette typique
  export const Route = createFileRoute('/players/$playerSlug/home')({
    component: HomePage,
  })
  ```
- **Problème** : `grep -rn "loader:" routes/` retourne 0 occurrence. TanStack Router v1 propose des `loader: ({ params }) => ...` qui pré-fetchent **avant** le rendu de la route, ce qui élimine le flash « écran vide → spinner → contenu » et permet le suspense. Ici, chaque page commence systématiquement par un `if (isLoading) return null` (HomePage:500, MatchViewPage:211, TimeseriesPage:83, etc.) — l'utilisateur voit l'écran blanc à chaque navigation. Le système ne tire profit de TanStack Router que pour le routing déclaratif ; le fetch est entièrement délégué à TanStack Query. Cohérent mais sous-optimal.
- **Action** : Pour les pages prioritaires (home, career, match-view), ajouter `loader: ({ params, context }) => context.queryClient.ensureQueryData({ queryKey: queryKeys.X, queryFn: ... })` et exposer le `queryClient` dans `routerContext`. Bénéfice : prefetch dès le hover du Link (déjà branché dans `useNavPrefetch` mais limité à home/career).

### [BLOQUANT] `useFieldLabel` est mort — 12 features réimplémentent un `labelOf` ad-hoc

- **Fichier:ligne** : Dans `features/`, 0 import de `useFieldLabel` mais 12 fichiers définissent une variante locale :
  - `features/match-view/MatchViewPage.tsx:208-209`
  - `features/home/HomePage.tsx:562-563`
  - `features/synthesis/SynthesisPage.tsx:137-138`
  - `features/match-view/MatchScoreboard.tsx`, `MatchStatCards.tsx`, `PlayerDetailPanel.tsx`
  - `features/match-history/MatchHistoryTable.tsx`
  - `features/timeseries/TimeseriesCorrelationScatter.tsx`
  - `features/session-detail/SessionDetailPage.tsx`
  - `features/career/CareerEncountersSection.tsx`
  - `features/prestige/components/StatsGlobales.tsx`
  - `components/shell/KPIBar.tsx`
- **Extrait** :
  ```tsx
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string, fallback: string): string =>
    fieldMappings?.fields[key]?.label ?? fallback
  ```
- **Problème** : Le hook documenté `useFieldLabel(key)` (ligne 133 de `lib/i18n/fieldMappings.ts`) n'est utilisé nulle part. Chaque feature recrée la même fonction inline avec `fallback` hardcodé en français (« Précision », « Kills », « KDA »…), ce qui défait l'intérêt multi-titres : si le titre courant n'a pas la field key dans son TOML, on retombe sur un FR-only. Le ADR 0003 sur les manifests TOML est partiellement respecté.
- **Action** : Adopter `useFieldLabel(key)` pour les cas simples ; pour les cas avec fallback, créer un `useFieldLabelWithFallback(key, fallbackKey: ManifestKey)` qui résout le fallback via le manifest i18n du composant. Supprimer les `labelOf` locaux.

### [BLOQUANT] SynthesisPage.tsx — strings FR hardcodées dans une page disposant déjà d'un manifest infrastructure

- **Fichier:ligne** : `apps/web/src/features/synthesis/SynthesisPage.tsx:46`, `109-115`, `125-131`, `144`, `147-156`, `181-184`, `214`, `241`, `388`, `418`, `433`, et `SynthesisHighlightsSection.tsx:67,82,92`, `SynthesisRelationsPreview.tsx:58`
- **Extrait** :
  ```tsx
  const DOW_LABELS = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim']
  // ...
  <CardTitle>Vue d'ensemble</CardTitle>
  <StatCell label="Victoires" value={String(overview.total_wins)} />
  <StatCell label="Défaites" value={...} />
  <StatCell label="K/D moyen" value={kd} />
  // ...
  <CardHeader><CardTitle>Par carte</CardTitle></CardHeader>
  <CardHeader><CardTitle>Par mode</CardTitle></CardHeader>
  ```
- **Problème** : aucune string ne passe par `formatMessage` dans cette page (à l'inverse de Squad V2, Home, Timeseries). Le linter `lint:fields` (script `tools/lint-no-hardcoded-fields.mjs`) ne détecte pas les libellés UI génériques. Synthesis est l'une des pages les plus visibles et resterait FR-only en mode `en` : c'est une régression du périmètre i18n.
- **Action** : Créer `apps/web/src/lib/i18n/manifests/synthesis.toml` et `synthesis/i18n.ts`, regen via `node apps/web/scripts/build_i18n_manifests.mjs`, remplacer les strings inline par `formatMessage(synthesisManifest, key, locale)`.

### [DETTE] Composants > 400 L — fetch + transform + 6 sous-composants par fichier

- **Fichier:ligne** :
  - `features/lab/LabPage.tsx` (937 L) — 8 sous-tableaux de status statiques mélangés avec 3 queries indépendantes
  - `features/match-view/MatchViewPage.tsx` (600 L) — 5 onglets `if (activeTab === 'X')` empilés ligne 317-590
  - `features/timeseries/TimeseriesPage.tsx` (480 L) — 6 onglets dans le même composant
  - `features/synthesis/SynthesisPage.tsx` (471 L)
  - `features/setup/SetupPage.tsx` (484 L)
  - `features/palmares/SeasonPassPage.tsx` (456 L)
  - `features/session-detail/SessionDetailPage.tsx` (444 L)
- **Problème** : tous dépassent le seuil de 80 L par fonction et 500 L par fichier (HomePage et LabPage). Les onglets mériteraient d'être des fichiers séparés pour permettre du lazy-load (`React.lazy` + `Suspense`) et un test ciblé.
- **Action** : Découper chaque page-multi-onglets en `Page.tsx` (orchestrateur) + `tabs/{Tab1,Tab2,…}.tsx`. Au minimum, `MatchViewPage` (5 onglets, lazy-load au `setActiveTab`).

### [DETTE] 8 query keys littérales définies hors du registre `lib/query/keys.ts`

- **Fichier:ligne** :
  - `features/explorer/queries.ts:34` — `['explorer-player', playerSlug, request.target_gamertag]`
  - `features/timeseries/queries.ts:31` — `['combatYieldHistory', playerSlug, filterHash]`
  - `features/squad/v2/queries.ts:26` — `['squad-v2', playerSlug, teammatesQuery, periodQuery]`
  - `features/auth/queries.ts:53,85` — `['admin', 'users']` / `['admin', 'invites']`
  - `features/changelog/queries.ts:10` — `['changelog']`
  - `features/help/queries.ts:10` — `['release-notes', lang]`
  - `features/media/queries.ts:344` — `['media', 'match-candidates', ...]`
  - `features/prestige/components/ChallengesCarousel.tsx:27` — `['prestige', 'challenges', userId, titleSlug]`
  - `features/admin/AdminPage.tsx:64,72,172,178` — invalidation par chaîne littérale
- **Problème** : la convention CLAUDE.md / skill `frontend-patterns` impose la centralisation dans `queryKeys` (lignes 9-122 de `lib/query/keys.ts`). Risque pratique : un refactor d'invalidation peut casser les query keys littérales sans que l'IDE flag.
- **Action** : Ajouter dans `queryKeys` les 8 entrées manquantes et remplacer les littérales.

### [DETTE] `components/ui/cover-flow-modal.tsx` importe `features/media/MediaViewer` — frontière inversée

- **Fichier:ligne** : `apps/web/src/components/ui/cover-flow-modal.tsx:5-6`
- **Extrait** :
  ```tsx
  import { MediaLikeButton } from '@/features/media/MediaViewer'
  import { getMediaModalsText } from '@/features/media/i18n-modals'
  ```
- **Problème** : `components/` doit fournir des primitives partagées sans connaître les features. Ici, un composant UI dépend de `features/media`, ce qui crée un cycle implicite et empêche la réutilisation hors média. `components/shell/AppShell.tsx:13-14` a la même odeur (`@/features/notifications/toastBridge`, `@/features/asset-drawer`) mais est plus défendable car le shell orchestre les features par nature ; `cover-flow-modal` n'a pas cette excuse.
- **Action** : Soit déplacer `cover-flow-modal` dans `features/media/`, soit injecter `MediaLikeButton` et le dictionnaire en props (DI).

### [DETTE] Logique de filtres + animation impérative dans HomePage `SessionCarouselCard`

- **Fichier:ligne** : `apps/web/src/features/home/HomePage.tsx:196-388`
- **Extrait** :
  ```tsx
  const dir = newIdx < displayIdx ? 'up' : 'down'
  el.style.transition = 'transform 0.1s ease-in, opacity 0.1s ease-in'
  el.style.transform = `translateY(${exitY})`
  el.style.opacity = '0'
  const t1 = setTimeout(() => { /* ... 30 L de chained timers et RAF ... */ }, 110)
  ```
- **Problème** : 80 L d'animation impérative qui mute des styles inline directement, avec deux niveaux de `setTimeout`/`requestAnimationFrame` imbriqués et un `cleanupRef.current` qui pointe vers 4 fonctions différentes selon l'étape. Difficile à tester (jsdom n'exécute pas les transitions CSS), aucun test unitaire ne couvre cette animation, risque de race condition élevé.
- **Action** : Remplacer par une transition CSS pure (classes `data-direction="up|down"` + `@keyframes`) ou utiliser `framer-motion` (déjà très léger) — plus simple, plus testable.

### [DETTE] LocalStorage écrit dans 4 features sans `try/catch` sur le `JSON.parse`

- **Fichier:ligne** :
  - `features/squad/SquadLayout.tsx:236-243`, `269-276` (try/catch présent ✓)
  - `features/match-history/MatchHistoryPage.tsx:40-49` (try/catch ✓)
  - `features/notifications/NotificationsSettingsTab.tsx:264-284` (`JSON.parse` absent — clé string only)
- **Problème** : La pratique est correcte (3/4 protégées), mais il n'y a pas d'abstraction commune. Chaque page gère sa clé `squad-teammates-${playerSlug}`, `squad-sessions-${playerSlug}`, etc., avec sa propre logique. L'invalidation cross-feature (par exemple changer de joueur) n'est pas centralisée — risque de fuite de l'état d'un joueur sur l'autre. Constat connexe : `globalFilterStore` utilise déjà `zustand/middleware/persist` ; pourquoi ces features n'utilisent pas la même mécanique ?
- **Action** : Pour les états par-joueur, créer un mini Zustand `playerLocalStateStore` avec `partialize` par slug, ou un hook `usePlayerLocalStorage(slug, key, default)`.

### [DETTE] Tests vitest absents sur 6 features critiques

- **Fichier:ligne** : `apps/web/src/features/{notifications,session-compare,asset-drawer,prestige,session-detail,changelog,help,admin}` — aucun `*.test.tsx` direct sur les composants page (NotificationsPage, SessionComparePage, AssetDrawer, ObjectifsPage, AdminPage). NotificationsBell (289 L) et NotificationsItem ont 0 test. CompareDrawer (testé), CareerHubPage (testé) couverts. SquadEngagementSection (utilise un `useMemo` non trivial) → 0 test.
- **Problème** : Les composants critiques récents ne sont pas couverts. `routes/players/$playerSlug.tsx` — qui contient le auto-snap-to-latest sur fin de sync (logique non triviale) — n'a pas de test. `prefetch.ts` (`useNavPrefetch`) n'a pas de test.
- **Action** : Prioriser tests sur (1) `useNavPrefetch` (helpers TanStack Query mockables) ; (2) `routes/players/$playerSlug.tsx` (auto-snap) ; (3) `NotificationsBell` (badge unread + click → mark-read) ; (4) `SessionComparePage` (393 L sans aucune couverture).

### [DETTE] `Settings` typé OK mais 7-tuple de literals comme type d'onglet → duplication

- **Fichier:ligne** : `apps/web/src/features/settings/SettingsPage.tsx:73-80`, dupliqué ligne 144
- **Extrait** :
  ```tsx
  const activeTab = ... as
    | 'general' | 'sync' | 'analyse' | 'lab' | 'users' | 'accessibility' | 'notifications' | null) ?? 'general'
  // ...
  function setActiveTab(tab: 'general' | 'sync' | 'analyse' | 'lab' | 'users' | 'accessibility' | 'notifications')
  ```
- **Problème** : 3 endroits dans le fichier répètent la même union `'general' | 'sync' | …`. À l'ajout d'un onglet, il faut maintenir la sync entre 3 endroits.
- **Action** : Extraire `type SettingsTab = ...` au top du fichier, ou typé comme `keyof typeof TABS` à partir d'un `const TABS = { general: ..., sync: ... } as const`.

### [AMÉLIORATION] `routeTree.gen.ts` commit-é (821 L) — bonne pratique TanStack v1 respectée

- **Fichier:ligne** : `apps/web/src/routeTree.gen.ts` est tracké par Git (vérifié par `git ls-files`). `.gitignore` ne l'exclut pas.
- **Problème** : aucun.
- **Action** : RAS, à conserver tel quel. Le commentaire dans `app/router/index.ts:7` (« stub initial remplacé à chaque lancement ») est ambigu — préciser que la version commitée est la cible.

### [AMÉLIORATION] Icônes SVG inline dupliquées dans 8 fichiers

- **Fichier:ligne** : `ChevronUpIcon`/`ChevronDownIcon` dans `HomePage.tsx:134-148`, `ChevronLeftIcon`/`ChevronRightIcon` dans `HomeBattlePassPanel.tsx:11-26`, `CloseIcon` dans `AssetDrawer.tsx:134`, `HeartIcon` dans `MediaViewer.tsx:50`, 9 icônes dans `notifications/icons.tsx` (déjà centralisées ✓).
- **Problème** : pas critique, mais les chevrons sont réinventés 3 fois avec des `viewBox` et `strokeWidth` différents, ce qui produit des inconsistances visuelles fines.
- **Action** : Centraliser les icônes communes dans `components/ui/icons.tsx` (sur le modèle de `features/notifications/icons.tsx`) ou installer `lucide-react`.

### [AMÉLIORATION] Couleurs Tailwind brutes (`bg-amber-*`, `text-green-*`) dans `features/`

- **Fichier:ligne** : 16 occurrences dans `features/{home,settings,notifications,palmares,media,squad}/`. Liste complète : `home/HomePage.tsx:1093`, `home/HomeBattlePassPanel.tsx:149`, `home/HomeChallengesList.tsx:57`, `media/MediaMatchPicker.tsx:69`, `notifications/NotificationItem.tsx:118`, `palmares/rarity.ts:65`, `settings/SettingsPage.tsx:556-557`, `settings/WatcherCard.tsx:62-213`.
- **Problème** : violation directe de la règle 20 de CLAUDE.md (« Aucun hex ni classe Tailwind couleur dans `features/` »). Couvert par axe 5 — listé ici juste pour signaler que les `*-amber-*` sont essentiellement des badges « warning » qui ont un token sémantique disponible (`perf-tier-2` ou un nouveau `state-warning`).
- **Action** : Ajouter un token `state-warning` / `state-info` dans `palettes.ts` et migrer (relève d'axe 5).

### [AMÉLIORATION] Couleur hex `#fff` inline dans MatchViewPage

- **Fichier:ligne** : `apps/web/src/features/match-view/MatchViewPage.tsx:373`
- **Extrait** : `style={{ backgroundColor: c.color ?? undefined, color: '#fff' }}`
- **Problème** : `#fff` brut alors que les couleurs UI doivent passer par tokens. Mineur car appliqué sur badge avec couleur de fond dynamique côté serveur.
- **Action** : Remplacer par `tokenCssVar('text-on-accent')` ou ajouter un token sémantique pour le contraste sur badge teinté.

## Cartographie : flux d'une page (ex: /home)

```
URL /players/JGtm/home
│
├─ TanStack Router résout file-based route
│  ├─ __root.tsx           → useQuery(/bootstrap), hydrate appShellStore
│  ├─ players/$playerSlug.tsx
│  │  ├─ guard auth (beforeLoad)
│  │  ├─ useFiltersResolve(playerSlug) — POST /filters/resolve
│  │  └─ rend <SessionNavBar/> + <NavL2/> + <Outlet/>
│  └─ players/$playerSlug/home.tsx → component: HomePage
│
└─ HomePage component (SAIN après mount uniquement)
   ├─ useParams → playerSlug
   ├─ useAppShellStore → locale, userTimezone
   ├─ useFieldMappings() → labels FR/EN multi-titres (cache infini)
   ├─ useHomePage(playerSlug) → GET /pages/home (staleTime 5min)
   ├─ useSeasonPassPreview(playerSlug) → GET /pages/palmares/season-pass
   ├─ useSetMatchFavorite() → mutation
   │
   ├─ if isLoading → return null               ← écran blanc, pas de skeleton
   ├─ if isError → EmptyStateCard avec retry
   ├─ if !data → EmptyStateCard
   │
   └─ render :
      ├─ HomeHeroBanner (sticky top)
      ├─ PrivacyBanner (warning si applicable)
      ├─ Spartan Identity grid (130 L inline)
      ├─ Skill peaks panel (70 L inline)
      ├─ KPI grid 9 tuiles (240 L de IIFEs)
      ├─ HomeBattlePassPanel + Défis (75 L inline)
      ├─ HomeRecentPlaylistsCard + SessionCarouselCard×2
      ├─ ChallengesCarousel (prestige)
      ├─ Highlights grid (HighlightTile + SerieTile)
      ├─ OutcomeSequenceTape
      ├─ Recent / Favorites tabs (MatchCard×4)
      └─ RecentMediaRail
```

Points sains : séparation route → composant page propre, hooks colocalisés dans `features/home/queries.ts`, fieldMappings cache infini (`staleTime: Infinity`), prefetch on hover via `useNavPrefetch`. Points à améliorer : `if (isLoading) return null` partout au lieu d'un Skeleton ; pas de `loader:` route → le fetch démarre seulement après le mount ; HomePage devrait être ≤ 200 L et orchestrer 6 sous-composants.

## Suivi recommandé

1. **Phase 1 (priorité haute)** : Découper `HomePage.tsx`, `LabPage.tsx`, `SettingsPage.tsx`, `MatchViewPage.tsx` en sous-composants (un fichier par onglet ou par bloc visuel). Corriger le bug `useState`-as-ref dans SettingsPage:67 (impact réel : leak de timers).
2. **Phase 2 (priorité moyenne)** : Adopter `useFieldLabel` partout, supprimer les `labelOf` ad-hoc, et créer le manifest `synthesis.toml` pour aligner Synthesis sur le reste. Ajouter aussi les 8 query keys manquantes dans `lib/query/keys.ts`.
3. **Phase 3 (priorité basse)** : Introduire `loader:` sur 3-4 routes prioritaires (home, career, match-view) pour pré-fetch + suspense ; combler le trou de tests sur Notifications, SessionCompare et `routes/players/$playerSlug.tsx` (auto-snap-to-latest).

## Constats hors-axe

- **Axe 5 (color tokens)** : 16 occurrences `bg-amber-*` / `text-green-*` listées en [AMÉLIORATION] couleur ; `#fff` inline `MatchViewPage.tsx:373`.
- **Axe 6 (DRY)** : icônes SVG inline réinventées (chevrons, close, heart) dans 4-8 fichiers ; helpers `labelOf` réécrits dans 12 features.
- **Axe 7 (tests profonds)** : couverture asymétrique (charts et squad bien couverts ; notifications/asset-drawer/session-compare/admin/prestige absents).
- **Axe 1 (charts)** : non rejoué — couche `components/charts/` saine et bien testée (10 tests sur 13 wrappers).
- **Axe 2 (multi-titres front)** : non rejoué — adapter pattern partiellement appliqué (cf. `useFieldLabel` non utilisé est connexe mais relève bien d'axe 4 : c'est un pattern d'usage React, pas le contrat multi-titre lui-même).

---

## Amendement post-vérification (2026-04-29)

> Ajouts issus de la passe de vérification finale (cf. [verification-finale-scaffolding.md](verification-finale-scaffolding.md)).

### [DETTE] `apps/web/src/app/routes/__root.tsx` orphelin (duplicata après migration de structure)

- **Fichier** : `apps/web/src/app/routes/__root.tsx` (~70 L), entier.
- **Problème** : `apps/web/src/app/router/index.ts:11` importe `routeTree` depuis `@/routeTree.gen` qui est généré depuis `src/routes/`. Le fichier `app/routes/__root.tsx` n'est référencé nulle part. Probable dérive d'une migration `app/routes/` → `routes/` où l'ancien fichier n'a pas été supprimé.
- **Action** : `rm apps/web/src/app/routes/__root.tsx` + nettoyer l'entrée correspondante dans `eslint.config.js:40` (déjà mentionné dans le suivi axe 9 mais à acter comme constat ici aussi).

### [DETTE] 3 composants Prestige exportés sans importateur

- **Fichier:ligne** :
  - `apps/web/src/features/prestige/components/MomentCard.tsx:25`
  - `apps/web/src/features/prestige/components/ArcSummary.tsx:19`
  - `apps/web/src/features/prestige/components/StatsGlobales.tsx:29`
- **Problème** : 3 composants exportés du module Prestige n'ont aucun importateur prod. Le module entier est gardé derrière `PRESTIGE_ENABLED=false` (cf. nouvel axe 11), donc ces composants sont morts par flag — mais même quand le flag est activé, ils ne sont pas branchés dans la vue Prestige.
- **Action** : décider — soit brancher dans la vue Prestige active (et écrire les tests), soit archiver explicitement avec un commentaire « scaffolding pour Sprint X ».

### [DETTE] 3 routes cibles fantômes dans `notifications/navigation.ts`

- **Fichier:ligne** :
  - `apps/web/src/features/notifications/navigation.ts:46` (`/players/$slug/defis`, case `challenge_added`/`challenge_completed`)
  - `apps/web/src/features/notifications/navigation.ts:52` (`/help/changelog`, case `app_release`)
  - `apps/web/src/features/notifications/navigation.ts:55` (`/players/$slug/sync`, case `sync_error`)
- **Problème** : ces 3 routes sont référencées comme cibles cliquables de notifications mais **n'existent pas** dans `apps/web/src/routes/`. La route changelog est `/changelog` (sans `/help/`). Le backend émet vers les mêmes routes inexistantes (`apps/go-api/internal/api/post_sync_deltas.go:261,277`). Bug fonctionnel : cliquer sur ces notifications mène à une 404.
- **Action** : aligner — soit créer les routes manquantes, soit corriger les `TargetRoute` côté Go ET le mapping `navigation.ts` côté front pour pointer vers des routes existantes (ex: `/changelog` au lieu de `/help/changelog`, `/objectifs?tab=parcours` au lieu de `/defis`).
