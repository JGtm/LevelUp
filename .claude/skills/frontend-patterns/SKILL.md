# Skill : frontend-patterns — React/TypeScript apps/web

## Structure d'une feature

Chaque feature est un module auto-suffisant dans `apps/web/src/features/{feature}/` :

```
features/notifications/
  queries.ts      — hooks useQuery (lecture)
  mutations.ts    — hooks useMutation (écriture)
  i18n.ts         — dictionnaire de la feature, une entrée par langue (Record<Locale, T>)
  types.ts        — interfaces TypeScript locales
  format.ts       — utilitaires de formatage
  *.tsx           — composants UI
```

## API client — `apps/web/src/lib/api/client.ts`

```typescript
import { api } from '@/lib/api/client'

const data = await api.get<MyResponse>(`/players/${playerSlug}/stats`)
const result = await api.post<MyResponse>(`/players/${playerSlug}/matches/query`, body)
```

Base URL : `VITE_API_BASE_URL` ou `/api/v1`. Credentials (cookies) inclus automatiquement.

**Multi-titres** : le client envoie `X-LevelUp-Title` automatiquement via `setApiTitleSlug()`. Ne pas passer le slug manuellement dans les routes.

**Erreurs** : `ApiError { code, message, retryable, field_errors, status }`. Un 401 dispatche l'event `levelup:auth-required`.

## TanStack Query — patterns

```typescript
// Lecture
export function useMatchHistory(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.matchHistory(playerSlug),
    queryFn: () => api.get<MatchHistoryResponse>(`/players/${playerSlug}/matches`),
    staleTime: 30_000,
    enabled: !!playerSlug,
  })
}

// Écriture
export function useMarkRead(playerSlug: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.patch(`/players/${playerSlug}/notifications/${id}/read`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.notifications(playerSlug) }),
  })
}
```

**Query keys** : centralisées dans `apps/web/src/lib/query/keys.ts` — toujours ajouter ici, jamais inline.

## i18n — N langues (ADR 0040)

Les langues viennent de `config/locales.toml`, jamais du code : `KNOWN_LOCALES`, `DEFAULT_LOCALE`,
`FALLBACK_LOCALE` et `INTL_TAG` sont générés (`lib/i18n/generated/locales.ts`). Aucune liste de langues
ni comparaison de langue littérale dans une feature. Ajouter une langue : `docs/FOUNDATIONS_GUIDE.md` § 3.5.

### 1. Strings UI

Deux sources, toutes deux complètes dans chaque langue listée :

```typescript
// a) manifests TOML (apps/web/src/lib/i18n/manifests/*.toml) -> modules générés, ICU
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest } from '@/lib/i18n/generated/home'
const text = formatMessage(homeManifest, 'home.foo.bar', locale, { name })

// b) dictionnaire de feature : features/myfeature/i18n.ts (au-delà de ~400 lignes : dossier i18n/ avec
//    contract.ts, un fichier par langue (fr.ts, en.ts) et index.ts)
import type { Locale } from '@/lib/i18n/locale'
interface MyText { title: string; emptyState: string }
export const MY_TEXT: Record<Locale, MyText> = {
  fr: { title: 'Mon titre', emptyState: 'Aucune donnée' },
  en: { title: 'My title',  emptyState: 'No data' },
}
const locale = useAppShellStore((s) => s.locale)
const t = MY_TEXT[locale]
```

`Record<Locale, T>` : le compilateur refuse une langue manquante. Un objet à clés de langue hors de ce
type est refusé par `locale-keyed-objects.guard.test.ts`.

### 2. Choisir et normaliser la langue

- `normalizeLocale(valeur)` (`@/lib/i18n/locale`) : étiquette BCP-47, casse et espaces tolérés ; vide ou
  inconnu -> `DEFAULT_LOCALE`. Jamais `x === 'en' ? 'en' : 'fr'`.
- `pickText(libellé, locale)` (`@/lib/i18n/pickText`) : un libellé par langue (`{ en, fr, ... }`, document de
  rejeu) -> la langue demandée, sinon `FALLBACK_LOCALE`, sinon `''`. Seul accès permis à ces libellés
  (`replay-label-access.guard.test.ts`) ; jamais `.fr` / `.en`.
- `intlLocale(locale)` (`@/lib/formatters`, table `INTL_TAG`) : étiquette passée à `Intl.*`,
  `toLocaleString(...)`, `toLocaleDateString(...)`. Jamais `'fr-FR'` ni un appel sans étiquette.
- Valeur par défaut d'un paramètre de langue : `DEFAULT_LOCALE`, jamais `'fr'`.

### 3. Réponses d'API : le libellé est déjà résolu

Le serveur renvoie `label` / `name` / `description` dans la langue de la requête (en-tête
`X-LevelUp-Locale`) : l'afficher tel quel, ne jamais choisir entre deux champs `x_fr` / `x_en` (le contrat
n'en porte plus). Conséquences :

- **query key avec la locale** pour toute réponse localisée (`lib/query/keys.ts` : `home`, `career`,
  `filtersResolve`, `matchHistory`... la locale est le dernier segment), sinon un fetch en vol survit à la
  bascule de langue ;
- **libellé et clé séparés** : ce qui sert de valeur de filtre, d'URL ou de regroupement est le nom de la
  langue par défaut et vit dans son propre champ (`LabelValue { value, label }`, `map_filter_key` /
  `map_label`) ; l'affichage lit le libellé, le filtre envoie la clé (`labelsOfOptions` pour retrouver
  le libellé d'une clé).

### 4. Interdits (garde-rails dans `lib/i18n/`)

| Interdit | Garde-rail |
|---|---|
| `locale === 'en'`, `lang !== 'fr'`, `isEn` ; ligne balisée `i18n-contrat` (choix d'un champ d'API suffixé) | `no-locale-literal-comparison.guard.test.ts` (plafond 0) |
| étiquette Intl littérale, appel de formatage sans étiquette | `no-intl-tag-literal.guard.test.ts` |
| redéclarer l'union `'fr' \| 'en'` | `locale.ratchet.test.ts` |
| littéral français hors manifest | règle ESLint `no-hardcoded-strings`, `no-hardcoded-locale-fallback.guard.test.ts` |
| anglicisme dans un texte français | `no-anglicisms.guard.test.ts` (règle française uniquement) |

### 5. Field mappings (backend-driven, multi-titres)

Pour les labels de stats issus des TOML (`kills` → "Éliminations" / "Kills") :

```typescript
import { useFieldLabel, useOutcomeLabel, useAssetLabel } from '@/lib/i18n/fieldMappings'

const killsLabel  = useFieldLabel('kills')           // "Éliminations" (FR)
const winLabel    = useOutcomeLabel('win')            // label + color token
const modeLabel   = useAssetLabel('game_variant', id) // depuis assets.toml
```

Cache infini — données versionnées Git côté API, endpoint `/api/v1/titles/{slug}/field-mappings`.

## Tableaux — TanStack Table (`@tanstack/react-table`)

**Règle** : tout tableau interactif (tri, filtre, pagination, colonnes) utilise TanStack Table. Un `<table>` HTML natif est toléré uniquement pour du rendu statique sans interaction (< 10 lignes, pas de tri).

Pattern minimal :

```typescript
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'

const columnHelper = createColumnHelper<MyRow>()

const columns = [
  columnHelper.accessor('kills', {
    header: () => t.kills,
    cell: info => info.getValue(),
  }),
  columnHelper.accessor('winRate', {
    header: () => t.winRate,
    cell: info => `${(info.getValue() * 100).toFixed(1)} %`,
  }),
]

export function MyTable({ data }: { data: MyRow[] }) {
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  return (
    <table>
      <thead>
        {table.getHeaderGroups().map(hg => (
          <tr key={hg.id}>
            {hg.headers.map(header => (
              <th key={header.id} onClick={header.column.getToggleSortingHandler()}>
                {flexRender(header.column.columnDef.header, header.getContext())}
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody>
        {table.getRowModel().rows.map(row => (
          <tr key={row.id}>
            {row.getVisibleCells().map(cell => (
              <td key={cell.id}>
                {flexRender(cell.column.columnDef.cell, cell.getContext())}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  )
}
```

**Règles** :
- `columnHelper.accessor` pour les colonnes typées, `columnHelper.display` pour les colonnes calculées/actions
- Les labels de colonnes passent par le dictionnaire `i18n` de la feature (jamais de string hardcodée dans `columns`)
- Les couleurs de cellule passent par `tokenCssVar()` (voir skill `color-tokens`)
- Pagination : `getPaginationRowModel()` + état `pagination` dans `useReactTable`

## Routing — TanStack Router (file-based)

Routes dans `apps/web/src/routes/`. La `routeTree.gen.ts` est générée automatiquement — ne jamais l'éditer.

Routes joueur principales :
```
/players/$playerSlug/home
/players/$playerSlug/career
/players/$playerSlug/squad
/players/$playerSlug/squad/synergies
/players/$playerSlug/matches/$matchId
/players/$playerSlug/stats/history
/players/$playerSlug/stats/timeseries
/players/$playerSlug/notifications
/players/$playerSlug/media
/players/$playerSlug/settings
```

## Rappel — règles couleurs

Pas de hex ni classe Tailwind couleur dans `features/` ou `components/`. Voir skill `color-tokens`.
