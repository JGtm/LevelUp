/**
 * MedalCard — tuile d'UNE médaille du catalogue (obtenue ou jamais obtenue).
 *
 * Title-agnostic : MedalIcon gère le sprite Halo 5 vs le PNG Halo Infinite selon
 * les champs présents. Signaux « jamais obtenue » (count===0) : compteur en token
 * `destructive` (rouge sémantique) + aria-label dédié + icône estompée. Pastille
 * de rareté affichée UNIQUEMENT pour les clés Halo Infinite (normal/heroic/
 * legendary/mythic) — jamais pour la difficulty_key numérique de Halo 5.
 *
 * Purement présentationnelle : reçoit item + locale (aucun store, aucun fetch).
 */
import { MedalIcon } from '@/components/ui/MedalIcon'
import { intlLocale } from '@/lib/formatters'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { medalsManifest } from '@/lib/i18n/generated/medals'
import { medalRarityLabel } from './labels'
import type { MedalSummaryItem } from '@/lib/api/types'

export function MedalCard({ item, locale }: { item: MedalSummaryItem; locale: ManifestLocale }) {
  const earned = item.count > 0
  const rarity = medalRarityLabel(item.difficulty_key, locale)
  const tooltip = item.description ? `${item.name} : ${item.description}` : item.name
  const countAria = earned
    ? formatMessage(medalsManifest, 'medals.card.earned_aria', locale, { count: item.count })
    : formatMessage(medalsManifest, 'medals.card.never_earned_aria', locale)

  return (
    <div
      title={tooltip}
      className={`flex w-[92px] cursor-default flex-col items-center gap-1 ${earned ? '' : 'opacity-60'}`}
    >
      <MedalIcon
        imageUrl={item.image_url}
        spriteSheet={item.sprite_sheet}
        spriteLeft={item.sprite_left}
        spriteTop={item.sprite_top}
        spriteWidth={item.sprite_width}
        spriteHeight={item.sprite_height}
        label={item.name}
        size={52}
        className="object-contain"
      />
      <span className="w-full truncate text-center text-[12px] leading-tight text-muted-foreground">
        {item.name}
      </span>
      <span
        aria-label={countAria}
        className={`text-[13px] font-semibold leading-none ${earned ? 'text-foreground' : 'text-destructive'}`}
      >
        {item.count.toLocaleString(intlLocale(locale))}
      </span>
      {rarity && (
        <span className="rounded-full border border-border bg-muted px-1.5 py-0.5 text-[10px] leading-none text-muted-foreground">
          {rarity}
        </span>
      )}
    </div>
  )
}
