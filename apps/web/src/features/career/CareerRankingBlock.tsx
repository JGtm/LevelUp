/**
 * CareerRankingBlock — bloc unifié CSR (API Waypoint) + LUSR (DB locale).
 * Deux colonnes côte à côte dans un seul conteneur ChartCard-style.
 *
 * Colonne gauche : liste par playlist ranked (badge + nom + tier + valeur).
 * Colonne droite : dernier checkpoint LUSR par playlist_group.
 */
import { Card, CardContent } from '@/components/ui/card'

const SUB_TIER_ROMAN = ['', 'I', 'II', 'III', 'IV', 'V', 'VI']
const toRoman = (n: number): string => SUB_TIER_ROMAN[n] ?? String(n)
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import type { CareerLusrSection, CareerCSRRank } from '@/lib/api/types'
import type { ManifestLocale } from '@/lib/i18n/format'
import { careerManifest } from '@/lib/i18n/generated/career'
import { staticAssetURL } from '@/lib/staticAssets'
import { useAppShellStore } from '@/stores/appShellStore'
import { useCareerCSRs } from './queries'
import { lusrChainLabel, LUSR_KNOWN_GROUPS } from './lusr-chains'

interface Props {
  playerSlug: string
  lusrData: CareerLusrSection | null | undefined
}

function csrTierLabel(rank: CareerCSRRank, placementLabel: string): string {
  if (!rank.tier) {
    // Phase 6 : placement_total (5 ou 10) injecté par le backend selon la saison
    // du snapshot. Fallback 10 si payload legacy.
    const total = rank.placement_total > 0 ? rank.placement_total : 10
    const completed = Math.min(total - 1, Math.max(0, total - rank.measurement_matches_remaining))
    return `${placementLabel} (${completed}/${total})`
  }
  return rank.sub_tier > 0 ? `${rank.tier} ${toRoman(rank.sub_tier)}` : rank.tier
}

function formatCSRValue(rank: CareerCSRRank): string {
  if (!rank.tier) return ''
  return rank.value > 0 ? ` · ${Math.round(rank.value).toLocaleString()}` : ''
}

function inputIcon(input: string): string {
  switch (input.toLowerCase()) {
    case 'keyboard': return '⌨'
    case 'controller': return '🎮'
    default: return '⇄'
  }
}

function shortPlaylistName(name: string): string {
  return name
    .replace(/^Ranked\s*/i, '')
    .replace(/\s+Ranked$/i, '')
    .trim() || name
}

type LusrCheckpoint = CareerLusrSection['checkpoints'][number]

function deriveLatestLUSRByGroup(
  checkpoints: CareerLusrSection['checkpoints'],
): Map<string, LusrCheckpoint> {
  const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-/i
  const byGroup = new Map<string, LusrCheckpoint>()
  for (const cp of checkpoints) {
    if (UUID_RE.test(cp.playlist_name)) continue
    const group = cp.playlist_group ?? cp.playlist_name
    if (!byGroup.has(group)) byGroup.set(group, cp)
  }
  return byGroup
}

export function CareerRankingBlock({ playerSlug, lusrData }: Props) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = (key: keyof typeof careerManifest) => careerManifest[key][locale]
  const { data: csrData } = useCareerCSRs(playerSlug)

  const playlists = csrData?.playlists ?? []
  const lusrByGroup = lusrData ? deriveLatestLUSRByGroup(lusrData.checkpoints) : new Map<string, LusrCheckpoint>()

  return (
    <Card className="flex h-full flex-col">
      <div className="border-b border-border px-3 py-2 text-sm font-medium">
        {t('career.ranking.title')}{' '}
        <InfoTooltip content={t('career.ranking.tooltip')} />
      </div>
      <CardContent className="flex flex-1 items-center p-3">
        <div className="grid w-full grid-cols-1 gap-6 sm:grid-cols-2">
          {/* Colonne gauche — CSR */}
          <div>
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {t('career.ranking.csr_section')}
            </p>
            {playlists.length === 0 ? (
              <EmptyStateNotice
                title={t('career.ranking.csr_no_data_title')}
                description={t('career.ranking.csr_no_data_description')}
              />
            ) : (
              <ul className="space-y-2">
                {playlists.map((pl) => (
                  <li key={pl.playlist_id} className="flex items-center gap-2">
                    <img
                      src={pl.current.badge_image_url ?? staticAssetURL('csr-rank', 'unranked_0', '.png')}
                      alt={pl.current.tier || 'unranked'}
                      className="h-8 w-8 shrink-0 object-contain"
                    />
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-foreground">
                        {shortPlaylistName(pl.playlist_name)}
                        <span className="ml-1 text-xs text-muted-foreground">
                          {inputIcon(pl.input)}
                        </span>
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {csrTierLabel(pl.current, t('career.ranking.placement'))}{formatCSRValue(pl.current)}
                      </p>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {/* Colonne droite — LUSR */}
          <div>
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {t('career.ranking.lusr_section')}{' '}
              <InfoTooltip content={t('career.ranking.lusr_tooltip')} />
            </p>
            <ul className="space-y-2">
              {LUSR_KNOWN_GROUPS.map((group) => {
                const cp = lusrByGroup.get(group)
                return (
                  <li key={group} className="flex items-center gap-2">
                    <img
                      src={cp?.badge_image_url ?? staticAssetURL('csr-rank', 'unranked_0', '.png')}
                      alt={cp?.tier_label ?? 'unranked'}
                      className="h-8 w-8 shrink-0 object-contain"
                    />
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-foreground">
                        {lusrChainLabel(group, locale)}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {cp
                          ? cp.tier_label
                            ? `${cp.tier_label} · ${Math.round(cp.rating_value).toLocaleString()}`
                            : Math.round(cp.rating_value).toLocaleString()
                          : `${t('career.ranking.placement')} (0/10)`}
                      </p>
                    </div>
                  </li>
                )
              })}
            </ul>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
