import type { AssetMeta } from '@/lib/api/types'
import type { ManifestLocale } from '@/lib/i18n/format'
import { AssetCard } from './AssetCard'

interface AssetGridProps {
  items: AssetMeta[]
  locale: ManifestLocale
  kind: 'maps' | 'weapons' | 'medals'
  emptyMessage: string
  isLoading: boolean
  isError: boolean
  errorMessage: string
  loadingMessage: string
}

export function AssetGrid({
  items,
  locale,
  kind,
  emptyMessage,
  isLoading,
  isError,
  errorMessage,
  loadingMessage,
}: AssetGridProps) {
  if (isLoading) {
    return (
      <p className="px-3 py-8 text-center text-sm text-muted-foreground">{loadingMessage}</p>
    )
  }
  if (isError) {
    return (
      <p className="px-3 py-8 text-center text-sm text-destructive">{errorMessage}</p>
    )
  }
  if (items.length === 0) {
    return (
      <p className="px-3 py-8 text-center text-sm text-muted-foreground">{emptyMessage}</p>
    )
  }
  return (
    <div className="grid grid-cols-2 gap-2 p-2">
      {items.map((asset) => (
        <AssetCard key={asset.id} asset={asset} locale={locale} kind={kind} />
      ))}
    </div>
  )
}
