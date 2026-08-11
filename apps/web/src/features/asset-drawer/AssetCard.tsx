import { WeaponIcon } from '@/components/ui/WeaponIcon'
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

  // Description médaille (locale-aware) : fr → description_fr sinon description.
  // omitempty côté backend → undefined → on ne rend rien (maps/armes inclus).
  const description = (locale === 'fr' && asset.description_fr ? asset.description_fr : asset.description) || undefined

  // Médailles Halo 5 : icône = découpe d'une feuille de sprites (background-position).
  const isSprite = kind === 'medals' && !!asset.sprite_sheet

  return (
    <div
      className="flex flex-col overflow-hidden rounded border border-border bg-card text-card-foreground"
      title={description ? `${label} — ${description}` : label}
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
        ) : asset.image_tinted && asset.image_url ? (
          // Icône MASQUE (dessin porté par l'alpha) : rendue telle quelle, elle serait
          // invisible sur fond clair. Le back le déclare, on ne le devine pas.
          <WeaponIcon
            imageUrl={asset.image_url}
            tinted
            label={label}
            width="100%"
            height="100%"
            className="p-1 text-foreground"
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
      <p className="truncate px-1.5 pt-1 text-3xs leading-tight text-foreground/80">
        {label}
      </p>
      {/* Médailles : pas de description tronquée sous la carte (décision GH-6,
          2026-07-08) — image + nom seulement. La description COMPLÈTE reste
          disponible au survol via le `title` du conteneur (« nom — description »).
          Maps/armes : inchangé (description sous la carte si présente). */}
      {kind !== 'medals' && description && (
        <p
          className="line-clamp-2 px-1.5 pb-1 text-3xs leading-tight text-muted-foreground"
          title={description}
        >
          {description}
        </p>
      )}
    </div>
  )
}
