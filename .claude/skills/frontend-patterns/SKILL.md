---
name: frontend-patterns
description: Conventions React/TypeScript d'apps/web (structure des features, client API, query keys TanStack, i18n N langues, tableaux, routes sous /t/{titre}, gardes de frontières). À invoquer avant d'écrire ou de modifier du code dans apps/web/src (features, components, lib, routes).
---

# Skill : frontend-patterns — React/TypeScript apps/web

Les règles courtes (couleurs, tableaux, query keys, graphes, routes) sont dans CLAUDE.md, règles 12 et 13, et dans ses sections « Frontières de couches » et « i18n ». Ce document donne le détail et les exemples.

## Structure d'une feature

Une feature vit dans `apps/web/src/features/{feature}/`. Elle n'importe aucune autre feature : ce qui est partagé va dans `components/`, `lib/` ou `features/_shared/`.

Le découpage varie selon la feature. Convention :

- lectures et écritures TanStack dans `queries.ts` (ou `*Queries.ts`, ou `hooks/`) ; `mutations.ts` seulement si la feature en a beaucoup (`notifications`, `admin/*`) ;
- dictionnaire de langue dans `i18n.ts`, ou dans un dossier `i18n/` (`contract.ts`, un fichier par langue, `index.ts`) quand il grossit (fichier <= 500 lignes) ;
- `types.ts`, `format.ts` quand la feature en a besoin ;
- composants dans des `*.tsx`, sans `useQuery` : ils appellent le hook de la feature.

## Client API — `apps/web/src/lib/api/client.ts`

```typescript
import { api } from '@/lib/api/client'

const data = await api.get<MyResponse>(`/players/${playerSlug}/stats`)
const result = await api.post<MyResponse>(`/players/${playerSlug}/matches/query`, body)
```

Autres méthodes : `patch`, `put`, `delete`, `getText`, `getBlob`, `getStatic`, `postForm`, `postKeepalive`. Base URL : `VITE_API_BASE_URL` ou `/api/v1` (`API_BASE_URL` pour une URL de navigation). Les cookies sont inclus.

Aucun `fetch` hors de `lib/api/` et `lib/desktop/` (garde `rawFetch`) : le client apporte les en-têtes de titre et de langue, le mapping d'erreurs et l'événement d'authentification.

**Titre** : il est dans l'URL (`/{-$lang}/t/$titleSlug/...`). Le client envoie `X-LevelUp-Title` seul (`setApiTitleSlug()`, posé au boot depuis l'URL puis depuis le bootstrap). Ne jamais passer le slug à la main dans un chemin d'API. Sur une page agnostique, aucun en-tête n'est envoyé.

**Langue** : le client envoie `X-LevelUp-Locale`.

**Erreurs** : `ApiError` est un objet simple, pas une `Error` : `{ code, message, retryable, details?, field_errors?, status }`. Lire avec `apiErrorMessage(err)` et `apiErrorCode(err)`. Un 401 de code `auth_required` (hors `/bootstrap`) déclenche l'événement `levelup:auth-required`.

**Types de réponse** : ils viennent de `lib/api/generated.ts` (`components['schemas'][...]`), jamais écrits à la main (garde `response-types`). Régénérer avec `make generate-types`.

**Bootstrap** : une seule lecture, dans `lib/query/bootstrap.ts` (garde `bootstrap`).

## TanStack Query — patterns

Exemple réel : `features/notifications/queries.ts`.

```typescript
export function useNotificationsList(playerSlug: string, filter: NotificationListFilter) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const locale = useAppShellStore((s) => s.locale)
  return useQuery<NotificationsListResponse>({
    queryKey: queryKeys.notifications(playerSlug, titleSlug, locale, filter),
    queryFn: () => api.get<NotificationsListResponse>(`/players/${playerSlug}/notifications${qs}`),
    enabled: !!playerSlug,
    staleTime: 30_000,
  })
}
```

Pour une écriture, voir `useMarkRead({ playerSlug })` dans `features/notifications/mutations.ts` : mise à jour optimiste, retour en arrière sur erreur, invalidation dans `onSettled` avec `queryKeys.notificationsAll(playerSlug)`.

**Query keys** : toutes dans `apps/web/src/lib/query/keys.ts`, jamais inline.

- Une clé qui dépend du titre porte `titleSlug` en 2e segment. Toute nouvelle fabrique se classe dans `keys.title-slug.guard.test.ts`.
- Une clé dérivée (`[...queryKeys.X, ...]`) est interdite hors de `keys.ts` : c'est une fabrique de plus.
- Une réponse localisée porte la langue en dernier segment (section i18n).

## i18n — N langues (ADR 0040)

Les langues viennent de `config/locales.toml`, jamais du code : `KNOWN_LOCALES`, `DEFAULT_LOCALE`, `FALLBACK_LOCALE` et `INTL_TAG` sont générés (`lib/i18n/generated/locales.ts`). Aucune liste de langues ni comparaison de langue littérale dans une feature. Ajouter une langue : `docs/FOUNDATIONS_GUIDE.md` § 3.5.

### 1. Strings UI

Deux sources, toutes deux complètes dans chaque langue listée :

```typescript
// a) manifests TOML (apps/web/src/lib/i18n/manifests/*.toml) -> modules générés, ICU
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest } from '@/lib/i18n/generated/home'
const text = formatMessage(homeManifest, 'home.foo.bar', locale, { name })

// b) dictionnaire de feature (features/myfeature/i18n.ts ou i18n/)
import type { Locale } from '@/lib/i18n/locale'
interface MyText { title: string; emptyState: string }
export const MY_TEXT: Record<Locale, MyText> = {
  fr: { title: 'Mon titre', emptyState: 'Aucune donnée' },
  en: { title: 'My title',  emptyState: 'No data' },
}
const locale = useAppShellStore((s) => s.locale)
const t = MY_TEXT[locale]
```

`Record<Locale, T>` : le compilateur refuse une langue manquante. Un objet à clés de langue hors de ce type est refusé par `locale-keyed-objects.guard.test.ts`.

### 2. Choisir et normaliser la langue

- `normalizeLocale(valeur)` (`@/lib/i18n/locale`) : étiquette BCP-47, casse et espaces tolérés ; vide ou inconnu -> `DEFAULT_LOCALE`. Jamais `x === 'en' ? 'en' : 'fr'`.
- `pickText(libellé, locale)` (`@/lib/i18n/pickText`) : un libellé par langue (`{ en, fr, ... }`, document de rejeu) -> la langue demandée, sinon `FALLBACK_LOCALE`, sinon `''`. Seul accès permis à ces libellés (`replay-label-access.guard.test.ts`) ; jamais `.fr` / `.en`.
- `intlLocale(locale)` (`@/lib/formatters`, table `INTL_TAG`) : étiquette passée à `Intl.*`, `toLocaleString(...)`, `toLocaleDateString(...)`. Jamais `'fr-FR'` ni un appel sans étiquette.
- Valeur par défaut d'un paramètre de langue : `DEFAULT_LOCALE`, jamais `'fr'`.

### 3. Réponses d'API : le libellé est déjà résolu

Le serveur renvoie `label` / `name` / `description` dans la langue de la requête (en-tête `X-LevelUp-Locale`) : l'afficher tel quel, ne jamais choisir entre deux champs `x_fr` / `x_en` (le contrat n'en porte plus). Conséquences :

- **query key avec la langue** pour toute réponse localisée (`home`, `career`, `filtersResolve`, `matchHistory`... la langue est le dernier segment), sinon un fetch en vol survit au changement de langue ;
- **libellé et clé séparés** : ce qui sert de valeur de filtre, d'URL ou de regroupement est le nom de la langue par défaut et vit dans son propre champ (`LabelValue { value, label }`, `map_filter_key` / `map_label`) ; l'affichage lit le libellé, le filtre envoie la clé (`labelsOfOptions` pour retrouver le libellé d'une clé).

### 4. Interdits (garde-rails dans `lib/i18n/`)

| Interdit | Garde-rail |
|---|---|
| `locale === 'en'`, `lang !== 'fr'`, `isEn` ; ligne balisée `i18n-contrat` (choix d'un champ d'API suffixé) | `no-locale-literal-comparison.guard.test.ts` (plafond 0) |
| étiquette Intl littérale, appel de formatage sans étiquette | `no-intl-tag-literal.guard.test.ts` |
| redéclarer l'union `'fr' \| 'en'` | `locale.ratchet.test.ts` |
| littéral français hors manifest | règle ESLint `no-hardcoded-strings`, `no-hardcoded-locale-fallback.guard.test.ts` |
| anglicisme dans un texte français | `no-anglicisms.guard.test.ts` (règle française uniquement) |

### 5. Field mappings (backend-driven, multi-titres)

Pour les labels de stats issus des TOML (`kills` -> "Éliminations" / "Kills") :

```typescript
import { useFieldLabel, useOutcomeLabel, useOutcomeMapping, useAssetLabel } from '@/lib/i18n/fieldMappings'

const killsLabel = useFieldLabel('kills')              // "Éliminations" (FR)
const winLabel   = useOutcomeLabel('win')              // label seul
const winMap     = useOutcomeMapping('win')            // { label, color_token }
const modeLabel  = useAssetLabel('game_variant', id)   // depuis assets.toml
```

Cache infini (données versionnées côté API). Endpoint `/api/v1/titles/{slug}/field-mappings?locale=` ; la clé de cache porte le titre et la langue.

## Tableaux — TanStack Table (`@tanstack/react-table`)

Tout tableau interactif (tri, filtre, colonnes) utilise TanStack Table. Un `<table>` natif sans interaction reste permis.

**Tri par clic sur un en-tête** : `useColumnSort` (`lib/table/useColumnSort`) + `SortableTh` (`components/ui/sortable-th`). Jamais de `toggleSort` local dans `features/` (garde `noLocalSortToggle`). Exemple réel : `features/admin/data-quality/IssueTable.tsx`.

```typescript
const SORT_COLUMNS = {
  kills: { value: (r: Row) => r.kills },
  name:  { value: (r: Row) => r.name, firstDir: 'asc' },
} as const

const sort = useColumnSort(rows, SORT_COLUMNS)
// <SortableTh label={t.kills} {...sort.header('kills')} />
// puis sort.rows pour le corps du tableau
```

`useColumnSort` règle le sens du premier clic, la bascule et les valeurs absentes (toujours en dernier).

Pour un tableau plus riche (colonnes calculées, pas seulement du tri), utiliser `useReactTable` avec `createColumnHelper` (voir `features/admin/sections/IdentitiesSection.tsx`).

**Règles** :
- Les labels de colonnes passent par le dictionnaire `i18n` de la feature (jamais de string en dur).
- Une aide d'en-tête passe par `meta: { headerTooltip }` (`lib/table/columnMeta.tsx`) ou la prop `tooltip` de `SortableTh`.
- Les couleurs de cellule passent par `tokenCssVar()` (skill `color-tokens`).
- La pagination se fait côté serveur (paramètre `page`) ; aucun `getPaginationRowModel` dans l'app.

## Routing — TanStack Router (file-based)

Routes dans `apps/web/src/routes/`. `routeTree.gen.ts` est généré : ne jamais l'éditer.

Les pages de jeu sont sous `/{-$lang}/t/$titleSlug/players/$playerSlug/` :

```
home, career (+ career/citations, commendations, medals, season-pass)
squad (+ synergies, contributions, dynamique, emprise, usages)
matches/$matchId (+ replay)
stats (+ sessions, synthesis, tendances, timeseries)
ascension/*, palmares/*, community/*, explorer, compare, media, notifications
```

Pages agnostiques du titre : `/settings`, `/admin/*`, `/groups`, `/changelog`, `/help`, `/login`, `/setup`. `routes/players/$.tsx` n'est qu'un splat de redirection des anciens liens.

Construire un chemin relatif au joueur par `lib/title-routing/playerScopedPath.ts` : c'est la seule source du littéral `/players/` (garde `no-title-literals.ratchet.test.ts`).

## Titre et capacités

Brancher sur une capacité, jamais sur le slug : `useCapability`, `useDataCapability`, `FeatureGate`, `RouteCapabilityGate` (`lib/capabilities/`). Une comparaison `slug === 'halo_5'` est refusée par `no-title-slug-comparison.guard.test.ts` et la règle ESLint `@levelup/no-title-slug-literal`.

## Frontières et gardes front

- `node tools/lint-cross-feature-imports.mjs` (lancé en CI) : pas d'import entre features, `features/_shared` sans dépendance vers une feature, `components/` sans `features/` (sauf `components/shell/`).
- Une feature n'importe pas `@/routes` (`noRouteImport`).
- Pas de `useQuery` dans un `.tsx` de `features/` (`featureTsxNoQuery`).
- `echarts` et `<LazyECharts` seulement sous `components/charts/` (`chartsImportBoundary`, `lazyEchartsMount`) ; voir `components/charts/README.md`.
- Pas de hex ni de classe Tailwind couleur dans `features/` ou `components/` : skill `color-tokens`.
