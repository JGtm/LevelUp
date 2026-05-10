/**
 * CareerLusrCards — career.11 : grille de cards LUSR par playlist_group.
 * Affiche rating actuel + delta vs checkpoint précédent + tier_label.
 *
 * 1 card par groupe canonique (arena/btb/fun/ranked). `social` (legacy) est
 * fusionné dans `arena`. Les checkpoints dont playlist_name est un UUID brut
 * (résolution metadata absente) sont écartés — cohérence avec le chart LUSR.
 */
import { tokenCssVar } from '@/lib/accessibility'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { CareerLusrCheckpoint } from '@/lib/api/types'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import { useAppShellStore } from '@/stores/appShellStore'
import { lusrChainLabel } from './lusr-chains'

interface LusrCardData {
  playlistGroup: string
  playlistLabel: string
  ratingValue: number
  tierLabel: string
  delta: number | null
  badgeImageUrl: string | null
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

export function CareerLusrCards({ checkpoints }: { checkpoints: CareerLusrCheckpoint[] }) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const cards = deriveLatestPerGroup(checkpoints, locale)

  if (cards.length === 0) {
    return (
      <EmptyStateNotice
        title={careerManifest['career.lusr.no_data_title'][locale]}
        description={careerManifest['career.lusr.no_data_description'][locale]}
      />
    )
  }

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
      {cards.map((card) => (
        <LusrRankCard key={card.playlistGroup} item={card} />
      ))}
    </div>
  )
}

function LusrRankCard({ item }: { item: LusrCardData }) {
  const tierColor = tierTokenForRating(item.ratingValue)
  return (
    <div
      className="flex items-center gap-3 rounded-lg border border-border bg-muted/20 p-3"
      style={{ borderTopColor: tokenCssVar(tierColor), borderTopWidth: 3 }}
    >
      {item.badgeImageUrl && (
        <img
          src={item.badgeImageUrl}
          alt={item.tierLabel || ''}
          className="h-12 w-12 shrink-0 object-contain"
          loading="lazy"
        />
      )}
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="text-xs font-medium text-muted-foreground">{item.playlistLabel}</span>
        <span className="text-2xl font-bold leading-none" style={{ color: tokenCssVar(tierColor) }}>
          {item.ratingValue}
        </span>
        {item.tierLabel && (
          <span className="text-xs text-foreground">{item.tierLabel}</span>
        )}
        {item.delta !== null && <DeltaBadge delta={item.delta} />}
      </div>
    </div>
  )
}

function DeltaBadge({ delta }: { delta: number }) {
  if (delta > 0) {
    return (
      <span className="text-xs font-semibold" style={{ color: tokenCssVar('outcome-win') }}>
        ▲ +{delta}
      </span>
    )
  }
  if (delta < 0) {
    return (
      <span className="text-xs font-semibold" style={{ color: tokenCssVar('outcome-loss') }}>
        ▼ {delta}
      </span>
    )
  }
  return (
    <span className="text-xs text-muted-foreground">= 0</span>
  )
}

function deriveLatestPerGroup(checkpoints: CareerLusrCheckpoint[], locale: ManifestLocale): LusrCardData[] {
  const byGroup = new Map<string, CareerLusrCheckpoint[]>()
  for (const cp of checkpoints) {
    if (!cp.recorded_at) continue
    if (UUID_RE.test((cp.playlist_name ?? '').trim())) continue
    const group = cp.playlist_group ?? 'arena_slayer'
    const list = byGroup.get(group) ?? []
    list.push(cp)
    byGroup.set(group, list)
  }

  const result: LusrCardData[] = []
  for (const [group, pts] of byGroup) {
    const sorted = [...pts].sort((a, b) => a.recorded_at!.localeCompare(b.recorded_at!))
    const last = sorted[sorted.length - 1]
    const prev = sorted.length >= 2 ? sorted[sorted.length - 2] : null
    const delta = prev !== null ? Math.round(last.rating_value - prev.rating_value) : null
    result.push({
      playlistGroup: group,
      playlistLabel: lusrChainLabel(group, locale),
      ratingValue: Math.round(last.rating_value),
      tierLabel: last.tier_label ?? '',
      delta,
      badgeImageUrl: last.badge_image_url ?? null,
    })
  }
  return result.sort((a, b) => b.ratingValue - a.ratingValue)
}

function tierTokenForRating(rating: number): 'perf-tier-1' | 'perf-tier-2' | 'perf-tier-3' | 'perf-tier-4' | 'perf-tier-5' {
  if (rating >= 2000) return 'perf-tier-5'
  if (rating >= 1800) return 'perf-tier-4'
  if (rating >= 1600) return 'perf-tier-3'
  if (rating >= 1400) return 'perf-tier-2'
  return 'perf-tier-1'
}
