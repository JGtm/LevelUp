# Skill : frontend-patterns — React/TypeScript apps/web

## Structure d'une feature

Chaque feature est un module auto-suffisant dans `apps/web/src/features/{feature}/` :

```
features/notifications/
  queries.ts      — hooks useQuery (lecture)
  mutations.ts    — hooks useMutation (écriture)
  i18n.ts         — dictionnaire FR/EN de la feature
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

## i18n — 2 couches

### 1. Strings UI (hardcodées par feature)

```typescript
// features/myfeature/i18n.ts
export type MyLocale = 'fr' | 'en'
interface MyText { title: string; emptyState: string }
export const MY_TEXT: Record<MyLocale, MyText> = {
  fr: { title: 'Mon titre', emptyState: 'Aucune donnée' },
  en: { title: 'My title',  emptyState: 'No data' },
}

// Dans le composant
const locale = useAppShellStore((s) => s.locale)
const t = MY_TEXT[locale]
```

### 2. Field mappings (backend-driven, multi-titres)

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
- Les labels de colonnes passent par `i18n.ts` de la feature (jamais de string hardcodée dans `columns`)
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
