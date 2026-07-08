/**
 * PlayerDetailPanel — panneau d'expander affiché sous une ligne du scoreboard.
 *
 * Layout : flex-wrap horizontal, toutes les sections côte à côte.
 * Armes  : images dans conteneur 64×28 px (harmonise paysage/portrait).
 * Médailles : icônes 32×32 avec dropShadow par niveau de difficulté.
 * Citations : CitationProgressRing (bleu normal, jaune si nouvellement maîtrisée).
 */
import { useState } from 'react'
import { CitationProgressRing } from '@/components/ui/citation-progress-ring'
import { MedalIcon } from '@/components/ui/MedalIcon'
import { citationMastery } from '@/lib/citations/mastery'
import { tokenCssVar } from '@/lib/accessibility'
import { perfScale } from '@/lib/accessibility/scales'
import { dropShadowForDifficulty } from '@/lib/medalDifficulty'
import { displayPlayerName } from '@/lib/players/displayName'
import { formatRankDelta } from '@/lib/formatters'
import type {
  MatchCitationSnippet,
  MatchKillerVictimPair,
  MatchScoreboardRow,
  MatchViewHeader,
  MatchViewRank,
  PlayerMedalRow,
  PlayerWeaponKillRow,
} from '@/lib/api/types'
import type { MatchViewText } from './i18n'

const _LIMIT_MEDALS = 6
const _LIMIT_WEAPONS = 5
const _LIMIT_CITATIONS = 4

interface Props {
  row: MatchScoreboardRow
  killerVictim?: MatchKillerVictimPair[] | null
  citations?: MatchCitationSnippet[]
  header?: MatchViewHeader
  rank?: MatchViewRank
  playerSlug?: string
  t: MatchViewText
}

// ---------------------------------------------------------------------------
// Composants structurels
// ---------------------------------------------------------------------------

function GroupTitle({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-1.5 text-2xs font-semibold uppercase tracking-wide text-foreground">
      {children}
    </div>
  )
}

function SectionGroup({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="flex-1 min-w-0">
      <GroupTitle>{title}</GroupTitle>
      {children}
    </div>
  )
}

function KvRow({ label, value, labelStyle }: { label: string; value: React.ReactNode; labelStyle?: React.CSSProperties }) {
  return (
    <div className="flex items-center justify-between gap-4 text-xs py-0.5 min-w-[140px]">
      <span className="text-muted-foreground whitespace-nowrap" style={labelStyle}>{label}</span>
      <span className="font-mono text-foreground">{value}</span>
    </div>
  )
}

// ---------------------------------------------------------------------------
// 1. Armes
// ---------------------------------------------------------------------------

function WeaponItem({ w }: { w: PlayerWeaponKillRow }) {
  const [imgFailed, setImgFailed] = useState(false)
  const showImage = w.image_url && !imgFailed
  const fallbackText = w.label ?? `#${w.weapon_id}`
  return (
    <div className="flex items-center gap-2" title={w.label ?? String(w.weapon_id)}>
      <div style={{ width: 56, height: 24 }} className="flex items-center justify-center flex-shrink-0">
        {showImage ? (
          <img
            src={w.image_url}
            alt=""
            className="max-h-full max-w-full object-contain"
            loading="lazy"
            onError={() => setImgFailed(true)}
          />
        ) : (
          <span className="text-2xs text-muted-foreground text-center leading-tight truncate w-full">
            {fallbackText}
          </span>
        )}
      </div>
      <span className="font-mono text-3xs font-semibold" style={{ color: tokenCssVar('perf-tier-2') }}>
        ×{w.kills}
      </span>
    </div>
  )
}

function WeaponsSection({ weapons, title }: { weapons: PlayerWeaponKillRow[]; title: string }) {
  const top = [...weapons].sort((a, b) => b.kills - a.kills).slice(0, _LIMIT_WEAPONS)
  if (top.length === 0) return null
  return (
    <SectionGroup title={title}>
      <div className="grid grid-cols-2 gap-x-4 gap-y-2">
        {top.map((w) => (
          <WeaponItem key={w.weapon_id} w={w} />
        ))}
      </div>
    </SectionGroup>
  )
}

// ---------------------------------------------------------------------------
// 2. Médailles
// ---------------------------------------------------------------------------

function MedalsSection({ medals, title }: { medals: PlayerMedalRow[]; title: string }) {
  const top = [...medals]
    .filter((m) => m.count > 0)
    .sort((a, b) => b.count - a.count)
    .slice(0, _LIMIT_MEDALS)
  if (top.length === 0) return null
  return (
    <SectionGroup title={title}>
      <div className="flex items-end gap-2.5 flex-wrap">
        {top.map((m) => {
          const glow = dropShadowForDifficulty(m.difficulty ?? undefined)
          return (
            <div key={m.medal_id} className="flex flex-col items-center gap-0.5" title={m.label}>
              {m.image_url || m.sprite_sheet ? (
                // MedalIcon = rendu title-agnostic (PNG Infinite OU sprite Halo 5) — le
                // drawer utilisait un <img> brut qui restait vide pour H5 (GH-5a).
                <MedalIcon
                  imageUrl={m.image_url}
                  spriteSheet={m.sprite_sheet}
                  spriteLeft={m.sprite_left}
                  spriteTop={m.sprite_top}
                  spriteWidth={m.sprite_width}
                  spriteHeight={m.sprite_height}
                  label={m.label ?? ''}
                  size={32}
                  style={{ filter: glow }}
                />
              ) : (
                <span className="text-2xs text-muted-foreground">{m.label ?? `#${m.medal_id}`}</span>
              )}
              <span className="font-mono text-2xs text-muted-foreground">×{m.count}</span>
            </div>
          )
        })}
      </div>
    </SectionGroup>
  )
}

// ---------------------------------------------------------------------------
// 3. Citations
// ---------------------------------------------------------------------------

function CitationsSection({ citations, t }: { citations: MatchCitationSnippet[]; t: MatchViewText }) {
  const top = citations.filter((c) => c.delta > 0).slice(0, _LIMIT_CITATIONS)
  if (top.length === 0) return null
  return (
    <SectionGroup title={t.sectionCitations}>
      <div className="flex items-start gap-3 flex-wrap">
        {top.map((c) => {
          const isMastered = citationMastery(c)
          return (
            <div key={c.key} className="flex flex-col items-center gap-0.5 max-w-[52px]" title={c.description ?? c.name}>
              <CitationProgressRing
                pct={c.progress_pct}
                imageUrl={c.image_url ?? undefined}
                isMastered={isMastered}
                isNewlyMastered={c.is_newly_mastered}
                size={44}
              />
              <span className="font-mono text-2xs font-semibold" style={{ color: isMastered ? tokenCssVar('perf-tier-3') : tokenCssVar('perf-tier-2') }}>
                +{c.delta}
              </span>
              {isMastered && (
                <span className="text-[9px] font-bold leading-none" style={{ color: tokenCssVar('perf-tier-3') }}>
                  {t.newlyMastered}
                </span>
              )}
            </div>
          )
        })}
      </div>
    </SectionGroup>
  )
}

// ---------------------------------------------------------------------------
// 4. Attendu vs réel
// ---------------------------------------------------------------------------

interface ExpectedItem { label: string; actual: number; expected: number; higherIsBetter: boolean; expectedDecimals?: number }

function buildExpectedItems(row: MatchScoreboardRow, t: MatchViewText): ExpectedItem[] {
  const items: ExpectedItem[] = []
  if (row.kills != null && row.expected_kills != null)
    items.push({ label: t.sbDetailExpectedKills, actual: row.kills, expected: row.expected_kills, higherIsBetter: true })
  if (row.deaths != null && row.expected_deaths != null)
    items.push({ label: t.sbDetailExpectedDeaths, actual: row.deaths, expected: row.expected_deaths, higherIsBetter: false })
  if (row.assists != null && row.expected_assists != null)
    items.push({ label: t.sbDetailExpectedAssists, actual: row.assists, expected: row.expected_assists, higherIsBetter: true, expectedDecimals: 2 })
  return items
}

function ExpectedSection({
  items,
  title,
  locallyEstimated,
  estimatedLabel,
  estimatedHint,
}: {
  items: ExpectedItem[]
  title: string
  locallyEstimated?: boolean
  estimatedLabel?: string
  estimatedHint?: string
}) {
  if (items.length === 0) return null
  return (
    <SectionGroup title={title}>
      <div className="space-y-0.5">
        {items.map((it) => {
          const delta = it.actual - it.expected
          const better = it.higherIsBetter ? delta > 0 : delta < 0
          const symbol = delta > 0 ? '↑' : delta < 0 ? '↓' : '→'
          const sign = delta > 0 ? '+' : ''
          const deltaCls = delta === 0 ? 'text-muted-foreground' : better ? 'text-success' : 'text-destructive'
          return (
            <KvRow
              key={it.label}
              label={it.label}
              value={
                <>
                  <span className="text-muted-foreground">{it.expected.toFixed(it.expectedDecimals ?? 1)} vs </span>
                  {it.actual.toFixed(0)}{' '}
                  <span className={deltaCls}>{symbol} {sign}{delta.toFixed(1)}</span>
                </>
              }
            />
          )
        })}
        {locallyEstimated && estimatedLabel && (
          <p className="pt-0.5 text-2xs italic text-muted-foreground" title={estimatedHint}>
            {estimatedLabel}
          </p>
        )}
      </div>
    </SectionGroup>
  )
}

// ---------------------------------------------------------------------------
// 5. Antagoniste
// ---------------------------------------------------------------------------

interface AntagonistResult { nemesisName: string; nemesisCount: number; bullyName: string; bullyCount: number }

function computeAntagonists(xuid: string, pairs: MatchKillerVictimPair[]): AntagonistResult {
  let nemesisName = '', nemesisCount = 0, bullyName = '', bullyCount = 0
  for (const p of pairs) {
    if (p.victim_xuid === xuid && p.kill_count > nemesisCount) { nemesisCount = p.kill_count; nemesisName = displayPlayerName(p.killer_gamertag, p.killer_xuid) }
    if (p.killer_xuid === xuid && p.kill_count > bullyCount) { bullyCount = p.kill_count; bullyName = displayPlayerName(p.victim_gamertag, p.victim_xuid) }
  }
  return { nemesisName, nemesisCount, bullyName, bullyCount }
}

function AntagonistSection({ result, title, nemesisLabel, bullyLabel }: { result: AntagonistResult; title: string; nemesisLabel: string; bullyLabel: string }) {
  if (result.nemesisCount === 0 && result.bullyCount === 0) return null
  return (
    <SectionGroup title={title}>
      <div className="space-y-0.5">
        {result.nemesisCount > 0 && <KvRow label={nemesisLabel} value={`${result.nemesisName} (${result.nemesisCount})`} labelStyle={{ color: tokenCssVar('outcome-loss') }} />}
        {result.bullyCount > 0 && <KvRow label={bullyLabel} value={`${result.bullyName} (${result.bullyCount})`} labelStyle={{ color: tokenCssVar('outcome-win') }} />}
      </div>
    </SectionGroup>
  )
}

// ---------------------------------------------------------------------------
// 6. Données locales
// ---------------------------------------------------------------------------

interface LocalRow { perfDisplay?: string; perfColorToken?: string; ratingType?: string; tierLabel?: string; ratingDelta?: number | null; iconUrl?: string | null; hadBotTeammate?: boolean }

function buildLocalRow(row: MatchScoreboardRow, header?: MatchViewHeader, mainRank?: MatchViewRank): LocalRow | null {
  const local: LocalRow = {}
  let hasData = false
  if (row.is_me) {
    if (header?.performance_display) { local.perfDisplay = header.performance_display; local.perfColorToken = header.performance_color_token; hasData = true }
    if (mainRank?.tier_label) { local.ratingType = mainRank.rating_type; local.tierLabel = mainRank.tier_label; local.ratingDelta = mainRank.delta_value; local.iconUrl = mainRank.icon_url; hasData = true }
    if (header?.had_bot_teammate) { local.hadBotTeammate = true; hasData = true }
  } else {
    if (row.performance_score != null) { local.perfDisplay = Math.round(row.performance_score).toString(); local.perfColorToken = perfScale(row.performance_score); hasData = true }
    if (row.skill_rank?.tier_label) { local.ratingType = row.skill_rank.rating_type; local.tierLabel = row.skill_rank.tier_label; local.ratingDelta = row.skill_rank.rating_delta; local.iconUrl = row.skill_rank.icon_url; hasData = true }
    if (row.had_bot_teammate) { local.hadBotTeammate = true; hasData = true }
  }
  return hasData ? local : null
}

function LocalSection({ data, t }: { data: LocalRow; t: MatchViewText }) {
  const rows: React.ReactNode[] = []
  if (data.perfDisplay) {
    const colorVar = data.perfColorToken ? `var(--ac-${data.perfColorToken})` : undefined
    rows.push(<KvRow key="perf" label={t.performance} value={<span style={{ color: colorVar }}>{data.perfDisplay}</span>} />)
  }
  if (data.tierLabel) {
    const label = data.ratingType === 'CSR' ? t.sbDetailCsr : t.sbDetailLusr
    const deltaStr = data.ratingDelta != null ? ` (${formatRankDelta(data.ratingDelta, data.ratingType ?? '')} pts)` : ''
    rows.push(
      <KvRow key="rank" label={label} value={
        <span className="flex items-center gap-1.5">
          {data.iconUrl && (
            <img src={data.iconUrl} alt={data.tierLabel} className="h-6 w-6 object-contain" loading="lazy" />
          )}
          <span>{data.tierLabel}{deltaStr}</span>
        </span>
      } />
    )
  }
  if (data.hadBotTeammate) rows.push(<KvRow key="bot" label={t.sbDetailBotNoteLabel} value={t.sbDetailBotNoteValue} />)
  if (rows.length === 0) return null
  return <SectionGroup title={t.sbDetailLocal}><div className="space-y-0.5">{rows}</div></SectionGroup>
}

// ---------------------------------------------------------------------------
// 7. Footer
// ---------------------------------------------------------------------------

function Footer({ isTracked, isMe, isBot, gamertag, playerSlug, t }: { isTracked: boolean; isMe: boolean; isBot: boolean; gamertag: string; playerSlug?: string; t: MatchViewText }) {
  const badgeText = isTracked ? t.sbDetailPlayerDb : t.sbDetailSharedOnly
  const showLink = !isMe && !isBot && !!playerSlug
  const explorerUrl = showLink ? `/players/${playerSlug}/explorer?mode=player&target=${encodeURIComponent(gamertag)}` : null
  const displayGamertag = gamertag
  return (
    <div className="flex items-center justify-between gap-2 border-t border-border pt-2 text-3xs">
      <span className="rounded bg-muted px-2 py-0.5 text-muted-foreground">{badgeText}</span>
      {explorerUrl && <a href={explorerUrl} className="text-info hover:underline">{t.sbDetailExplorePlayerFmt(displayGamertag)}</a>}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Composant principal
// ---------------------------------------------------------------------------

export function PlayerDetailPanel({ row, killerVictim, citations, header, rank, playerSlug, t }: Props) {
  const weapons = row.weapon_kills ?? []
  const medals = row.medals ?? []
  const myCitations = row.is_me ? (citations ?? []) : []
  const expectedItems = buildExpectedItems(row, t)
  const antagonist = killerVictim ? computeAntagonists(row.xuid, killerVictim) : null
  const localData = buildLocalRow(row, header, rank)
  const isTracked = row.is_me || localData != null

  return (
    <div className="bg-card/80 border border-border rounded-b px-4 py-3 space-y-3">
      <div className="flex flex-wrap items-start gap-y-4 divide-x divide-border/70 [&>*]:px-4 [&>*:first-child]:pl-0">
        <WeaponsSection weapons={weapons} title={t.sbDetailWeapons} />
        <div className="flex-1 min-w-0 flex flex-col gap-3">
          <MedalsSection medals={medals} title={t.sbDetailMedalsOnly} />
          {myCitations.length > 0 && <CitationsSection citations={myCitations} t={t} />}
        </div>
        <ExpectedSection
          items={expectedItems}
          title={t.sbDetailExpected}
          locallyEstimated={(row.locally_estimated ?? false) || (row.expected_kills == null && row.expected_deaths == null && row.expected_assists != null)}
          estimatedLabel={t.sbDetailLocallyEstimated}
          estimatedHint={t.sbDetailLocallyEstimatedHint}
        />
        {antagonist && (
          <AntagonistSection
            result={antagonist}
            title={t.sbDetailAntagonist}
            nemesisLabel={t.sbDetailNemesis}
            bullyLabel={t.sbDetailBully}
          />
        )}
        {localData && <LocalSection data={localData} t={t} />}
      </div>
      <Footer isTracked={isTracked} isMe={row.is_me} isBot={row.is_bot ?? false} gamertag={row.gamertag} playerSlug={playerSlug} t={t} />
    </div>
  )
}
