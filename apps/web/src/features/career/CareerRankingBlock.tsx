/**
 * CareerRankingBlock — bloc unifié CSR (API Waypoint) + LUSR (DB locale).
 * Deux colonnes côte à côte dans un seul conteneur ChartCard-style.
 *
 * Colonne gauche : liste par playlist ranked (badge + nom + tier + valeur).
 * Colonne droite : dernier checkpoint LUSR par playlist_group.
 */
import { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'

const SUB_TIER_ROMAN = ['', 'I', 'II', 'III', 'IV', 'V', 'VI']
const toRoman = (n: number): string => SUB_TIER_ROMAN[n] ?? String(n)
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { MetricWithTrend, type Trend } from '@/components/ui/metric-trend'
import type { CareerLusrSection, CareerCSRRank } from '@/lib/api/types'
import type { ManifestLocale } from '@/lib/i18n/format'
import { careerManifest } from '@/lib/i18n/generated/career'
import { staticAssetURL } from '@/lib/staticAssets'
import { localizeTierName, localizeTierLabel } from '@/lib/skillTiers'
import { useAppShellStore } from '@/stores/appShellStore'
import { useCapability } from '@/lib/capabilities/capabilities'
import { useCareerCSRs } from './queries'
import { lusrChainLabel, resolveLusrGroupsForDisplay } from './lusr-chains'

interface Props {
  playerSlug: string
  lusrData: CareerLusrSection | null | undefined
}

function csrTierLabel(
  rank: CareerCSRRank,
  locale: ManifestLocale,
  placementLabel: string,
  unrankedLabel: string,
): string {
  if (!rank.tier) {
    // placement_total (5 ou 10 selon la saison / le titre) est TOUJOURS fourni
    // par le backend (Pass A : par-titre, type requis non-nullable).
    const total = rank.placement_total
    const completed = Math.min(total - 1, Math.max(0, total - rank.measurement_matches_remaining))
    return completed === 0 ? unrankedLabel : `${placementLabel} (${completed}/${total})`
  }
  // rank.tier est le palier canonique EN ("Gold"/"Platinum") — localisé à
  // l'affichage (sinon « Gold IV » restait en anglais sous UI FR).
  const name = localizeTierName(rank.tier, locale)
  return rank.sub_tier > 0 ? `${name} ${toRoman(rank.sub_tier)}` : name
}

function formatCSRValue(rank: CareerCSRRank): string {
  if (!rank.tier) return ''
  return rank.value > 0 ? ` · ${Math.round(rank.value).toLocaleString()}` : ''
}

// csrSeasonTrend compare la valeur CSR de la saison sélectionnée à celle de la
// saison précédente (même playlist). null si l'une des deux n'est pas classée
// (tier vide / valeur ≤ 0) : aucune flèche sur un placement ou une absence.
function csrSeasonTrend(current: CareerCSRRank, prev: CareerCSRRank | undefined): Trend | null {
  if (!prev || !current.tier || !prev.tier || current.value <= 0 || prev.value <= 0) return null
  if (current.value > prev.value) return 'up'
  if (current.value < prev.value) return 'down'
  return 'stable'
}

function inputIcon(input: string): string {
  switch (input.toLowerCase()) {
    case 'keyboard': return '⌨'
    case 'controller': return '🎮'
    default: return ''
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
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const t = (key: keyof typeof careerManifest) => careerManifest[key][locale]
  // Gating multi-titre (Phase 5) : colonne CSR ⇒ `ranked`, colonne LUSR ⇒ `lusr`.
  // NO-OP pour halo_infinite (déclare les deux). Si AUCUNE des deux capabilities,
  // le bloc entier est masqué (sinon : carte vide à 2 colonnes mortes).
  const hasRanked = useCapability('ranked')
  const hasLusr = useCapability('lusr')
  // Saison CSR sélectionnée (undefined → saison courante côté backend).
  const [season, setSeason] = useState<string | undefined>(undefined)
  const { data: csrData } = useCareerCSRs(playerSlug, season)

  const playlists = csrData?.playlists ?? []
  const availableSeasons = csrData?.available_seasons ?? []
  const selectedSeason = season ?? csrData?.season_id ?? ''
  // Delta CSR vs saison PRÉCÉDENTE : available_seasons est triée récentes
  // d'abord (backend sortCSRSeasonsDesc), donc la saison antérieure à celle
  // affichée est l'entrée suivante. Second appel gated (enabled) pour éviter
  // une collision de query key quand il n'existe aucune saison antérieure.
  const selectedIdx = availableSeasons.findIndex((s) => s.season_id === selectedSeason)
  const previousSeasonId =
    selectedIdx >= 0 ? availableSeasons[selectedIdx + 1]?.season_id : undefined
  const { data: prevCsrData } = useCareerCSRs(playerSlug, previousSeasonId, !!previousSeasonId)
  const prevByPlaylist = new Map<string, CareerCSRRank>(
    (prevCsrData?.playlists ?? []).map((pl) => [pl.playlist_id, pl.current]),
  )
  const lusrByGroup = lusrData ? deriveLatestLUSRByGroup(lusrData.checkpoints) : new Map<string, LusrCheckpoint>()
  // Groupes affichés = UNION (connus du titre, ordre déclaré) + (groupes présents
  // dans la donnée mais non connus, triés). HINF : ses 4 connus, inchangé. h5 :
  // aucun connu → uniquement `h5_arena` (issu des checkpoints).
  const lusrGroups = resolveLusrGroupsForDisplay(titleSlug, lusrByGroup.keys())

  if (!hasRanked && !hasLusr) return null

  return (
    <Card className="flex h-full flex-col">
      <div className="border-b border-border px-3 py-2 text-sm font-medium">
        {t('career.ranking.title')}{' '}
        <InfoTooltip content={t('career.ranking.tooltip')} />
      </div>
      <CardContent className="flex flex-1 items-center p-3">
        <div className={`grid w-full grid-cols-1 gap-6 ${hasRanked && hasLusr ? 'sm:grid-cols-2' : ''}`}>
          {/* Colonne gauche — CSR (par saison) — gatée sur `ranked` */}
          {hasRanked && (
          <div>
            <div className="mb-2 flex items-center justify-between gap-2">
              <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                {t('career.ranking.csr_section')}
              </p>
              {availableSeasons.length > 1 && (
                <select
                  aria-label={t('career.ranking.season_label')}
                  value={selectedSeason}
                  onChange={(e) => setSeason(e.target.value)}
                  className="rounded border border-border bg-transparent px-1.5 py-0.5 text-xs text-muted-foreground"
                >
                  {availableSeasons.map((s) => (
                    <option key={s.season_id} value={s.season_id}>
                      {s.label}
                    </option>
                  ))}
                </select>
              )}
            </div>
            {playlists.length === 0 ? (
              <EmptyStateNotice
                title={t('career.ranking.csr_no_data_title')}
                description={t('career.ranking.csr_no_data_description')}
              />
            ) : (
              <ul className="space-y-2">
                {playlists.map((pl) => {
                  const prevRank = prevByPlaylist.get(pl.playlist_id)
                  const trend = csrSeasonTrend(pl.current, prevRank)
                  const csrText = `${csrTierLabel(pl.current, locale, t('career.ranking.placement'), t('career.ranking.unranked'))}${formatCSRValue(pl.current)}`
                  const trendTooltip =
                    trend && prevRank
                      ? `${t('career.ranking.vs_prev_season')} : ${csrTierLabel(prevRank, locale, t('career.ranking.placement'), t('career.ranking.unranked'))}${formatCSRValue(prevRank)}`
                      : undefined
                  return (
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
                          <MetricWithTrend text={csrText} trend={trend} tooltip={trendTooltip} />
                        </p>
                      </div>
                    </li>
                  )
                })}
              </ul>
            )}
          </div>

          )}

          {/* Colonne droite — LUSR (cumulatif, hors saison) — gatée sur `lusr` */}
          {hasLusr && (
          <div>
            <div className="mb-2">
              <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                {t('career.ranking.lusr_section')}{' '}
                <InfoTooltip content={t('career.ranking.lusr_tooltip')} />
              </p>
              <p className="text-[10px] normal-case tracking-normal text-muted-foreground/70">
                {t('career.ranking.lusr_cumulative')}
              </p>
            </div>
            <ul className="space-y-2">
              {lusrGroups.map((group) => {
                const cp = lusrByGroup.get(group)
                // tier_label LUSR baké en FR au sync → localisé à l'affichage.
                const cpTierLabel = localizeTierLabel(cp?.tier_label, locale)
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
                          ? cpTierLabel
                            ? cp.rating_value > 0
                              ? `${cpTierLabel} · ${Math.round(cp.rating_value).toLocaleString()}`
                              : cpTierLabel
                            : cp.rating_value > 0
                              ? Math.round(cp.rating_value).toLocaleString()
                              : t('career.ranking.unranked')
                          : t('career.ranking.unranked')}
                      </p>
                    </div>
                  </li>
                )
              })}
            </ul>
          </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
