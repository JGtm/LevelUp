import type { AssetMeta } from '@/lib/api/types'
import type { ManifestLocale } from '@/lib/i18n/format'

const PLACEHOLDER_SVG =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='80' height='60' viewBox='0 0 80 60'%3E%3Crect width='80' height='60' fill='%23374151'/%3E%3Cpath d='M30 20h20v20H30z' fill='%236b7280' opacity='.5'/%3E%3C/svg%3E"

interface AssetCardProps {
  asset: AssetMeta
  locale: ManifestLocale
  kind: 'maps' | 'weapons' | 'medals'
}

export function AssetCard({ asset, locale, kind }: AssetCardProps) {
  const label = locale === 'fr' && asset.name_fr ? asset.name_fr : asset.name_en

  // Médailles Halo 5 : icône = découpe d'une feuille de sprites (background-position).
  const isSprite = kind === 'medals' && !!asset.sprite_sheet

  return (
    <div
      className="flex flex-col overflow-hidden rounded border border-border bg-card text-card-foreground"
      title={label}
    >
      <div className="relative flex aspect-[4/3] items-center justify-center overflow-hidden bg-muted">
        {isSprite ? (
          <div
            role="img"
            aria-label={label}
            className="shrink-0"
            style={{
              width: `${asset.sprite_width || 74}px`,
              height: `${asset.sprite_height || 74}px`,
              backgroundImage: `url(${asset.sprite_sheet})`,
              backgroundPosition: `-${asset.sprite_left || 0}px -${asset.sprite_top || 0}px`,
              backgroundRepeat: 'no-repeat',
            }}
          />
        ) : (
          <img
            src={asset.image_url || PLACEHOLDER_SVG}
            alt={label}
            loading="lazy"
            decoding="async"
            className={`h-full w-full ${kind === 'weapons' ? 'object-contain p-1' : 'object-cover'}`}
            onError={(e) => {
              ;(e.currentTarget as HTMLImageElement).src = PLACEHOLDER_SVG
            }}
          />
        )}
      </div>
      <p className="truncate px-1.5 py-1 text-3xs leading-tight text-muted-foreground">
        {label}
      </p>
    </div>
  )
}
