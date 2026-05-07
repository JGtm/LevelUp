/**
 * PlayerDetailPanel — panneau d'expander affiché sous une ligne du scoreboard.
 *
 * Layout : flex-wrap horizontal, toutes les sections côte à côte.
 * Armes  : images dans conteneur 64×28 px (harmonise paysage/portrait).
 * Médailles : icônes 32×32 avec dropShadow par niveau de difficulté.
 * Citations : CitationProgressRing (bleu normal, jaune si nouvellement maîtrisée).
 */
import { CitationProgressRing } from '@/components/ui/citation-progress-ring'
import { tokenCssVar } from '@/lib/accessibility'
import { dropShadowForDifficulty } from '@/lib/medalDifficulty'
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
    <div className="mb-1.5 text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
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

function KvRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4 text-xs py-0.5 min-w-[140px]">
      <span className="text-muted-foreground whitespace-nowrap">{label}</span>
      <span className="font-mono text-foreground">{value}</span>
    </div>
  )
}

// ---------------------------------------------------------------------------
// 1. Armes
// ---------------------------------------------------------------------------

function WeaponsSection({ weapons, title }: { weapons: PlayerWeaponKillRow[]; title: string }) {
  const top = [...weapons].sort((a, b) => b.kills - a.kills).slice(0, _LIMIT_WEAPONS)
  if (top.length === 0) return null
  return (
    <SectionGroup title={title}>
      <div className="grid grid-cols-2 gap-x-4 gap-y-2">
        {top.map((w) => (
          <div key={w.weapon_id} className="flex items-center gap-2" title={w.label ?? String(w.weapon_id)}>
            {/* Conteneur fixe : harmonise les armes larges (sniper) et hautes (épée) */}
            <div style={{ width: 56, height: 24 }} className="flex items-center justify-center flex-shrink-0">
              {w.image_url ? (
                <img
                  src={w.image_url}
                  alt=""
                  className="max-h-full max-w-full object-contain"
                  loading="lazy"
                  onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
                />
              ) : (
                <span className="text-[10px] text-muted-foreground text-center leading-tight">
                  {w.label ?? `#${w.weapon_id}`}
                </span>
              )}
            </div>
            <span className="font-mono text-[11px] font-semibold" style={{ color: tokenCssVar('perf-tier-2') }}>
              ×{w.kills}
            </span>
          </div>
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
              {m.image_url ? (
                <img
                  src={m.image_url}
                  alt={m.label ?? ''}
                  style={{ width: 32, height: 32, objectFit: 'contain', filter: glow }}
                  loading="lazy"
                  onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
                />
              ) : (
                <span className="text-[10px] text-muted-foreground">{m.label ?? `#${m.medal_id}`}</span>
              )}
              <span className="font-mono text-[10px] text-muted-foreground">×{m.count}</span>
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
        {top.map((c) => (
          <div key={c.key} className="flex flex-col items-center gap-0.5 max-w-[52px]" title={c.description ?? c.name}>
            <CitationProgressRing
              pct={c.progress_pct}
              imageUrl={c.image_url ?? undefined}
              isNewlyMastered={c.is_newly_mastered}
              size={44}
            />
            <span className="font-mono text-[10px] font-semibold" style={{ color: c.is_newly_mastered ? tokenCssVar('perf-tier-3') : tokenCssVar('perf-tier-2') }}>
              +{c.delta}
            </span>
            {c.is_newly_mastered && (
              <span className="text-[9px] font-bold leading-none" style={{ color: tokenCssVar('perf-tier-3') }}>
                {t.newlyMastered}
              </span>
            )}
          </div>
        ))}
      </div>
    </SectionGroup>
  )
}

// ---------------------------------------------------------------------------
// 4. Attendu vs réel
// ---------------------------------------------------------------------------

interface ExpectedItem { label: string; actual: number; expected: number; higherIsBetter: boolean }

function buildExpectedItems(row: MatchScoreboardRow, t: MatchViewText): ExpectedItem[] {
  const items: ExpectedItem[] = []
  if (row.kills != null && row.expected_kills != null)
    items.push({ label: t.sbDetailExpectedKills, actual: row.kills, expected: row.expected_kills, higherIsBetter: true })
  if (row.deaths != null && row.expected_deaths != null)
    items.push({ label: t.sbDetailExpectedDeaths, actual: row.deaths, expected: row.expected_deaths, higherIsBetter: false })
  return items
}

function ExpectedSection({ items, title }: { items: ExpectedItem[]; title: string }) {
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
                  <span className="text-muted-foreground">{it.expected.toFixed(1)} vs </span>
                  {it.actual.toFixed(0)}{' '}
                  <span className={deltaCls}>{symbol} {sign}{delta.toFixed(1)}</span>
                </>
              }
            />
          )
        })}
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
    if (p.victim_xuid === xuid && p.kill_count > nemesisCount) { nemesisCount = p.kill_count; nemesisName = p.killer_gamertag || p.killer_xuid }
    if (p.killer_xuid === xuid && p.kill_count > bullyCount) { bullyCount = p.kill_count; bullyName = p.victim_gamertag || p.victim_xuid }
  }
  return { nemesisName, nemesisCount, bullyName, bullyCount }
}

function AntagonistSection({ result, title, nemesisLabel, bullyLabel }: { result: AntagonistResult; title: string; nemesisLabel: string; bullyLabel: string }) {
  if (result.nemesisCount === 0 && result.bullyCount === 0) return null
  return (
    <SectionGroup title={title}>
      <div className="space-y-0.5">
        {result.nemesisCount > 0 && <KvRow label={nemesisLabel} value={`${result.nemesisName} (${result.nemesisCount})`} />}
        {result.bullyCount > 0 && <KvRow label={bullyLabel} value={`${result.bullyName} (${result.bullyCount})`} />}
      </div>
    </SectionGroup>
  )
}

// ---------------------------------------------------------------------------
// 6. Données locales
// ---------------------------------------------------------------------------

interface LocalRow { perfDisplay?: string; perfColorToken?: string; ratingType?: string; tierLabel?: string; ratingDelta?: number | null; hadBotTeammate?: boolean }

function buildLocalRow(row: MatchScoreboardRow, header?: MatchViewHeader, mainRank?: MatchViewRank): LocalRow | null {
  const local: LocalRow = {}
  let hasData = false
  if (row.is_me) {
    if (header?.performance_display) { local.perfDisplay = header.performance_display; local.perfColorToken = header.performance_color_token; hasData = true }
    if (mainRank?.tier_label) { local.ratingType = mainRank.rating_type; local.tierLabel = mainRank.tier_label; local.ratingDelta = mainRank.delta_value; hasData = true }
    if (header?.had_bot_teammate) { local.hadBotTeammate = true; hasData = true }
  } else {
    if (row.performance_score != null) { local.perfDisplay = Math.round(row.performance_score).toString(); hasData = true }
    if (row.skill_rank?.tier_label) { local.ratingType = row.skill_rank.rating_type; local.tierLabel = row.skill_rank.tier_label; local.ratingDelta = row.skill_rank.rating_delta; hasData = true }
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
    const sign = (data.ratingDelta ?? 0) >= 0 ? '+' : ''
    const display = data.ratingDelta != null ? `${data.tierLabel} (${sign}${data.ratingDelta.toFixed(0)} pts)` : data.tierLabel
    rows.push(<KvRow key="rank" label={label} value={display} />)
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
  return (
    <div className="flex items-center justify-between gap-2 border-t border-border pt-2 text-[11px]">
      <span className="rounded bg-muted px-2 py-0.5 text-muted-foreground">{badgeText}</span>
      {explorerUrl && <a href={explorerUrl} className="text-info hover:underline">{t.sbDetailExplorePlayerFmt(gamertag)}</a>}
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
      <div className="flex flex-wrap gap-5 items-start">
        <WeaponsSection weapons={weapons} title={t.sbDetailWeapons} />
        <div className="flex-1 min-w-0 flex flex-col gap-3">
          <MedalsSection medals={medals} title={t.sbDetailMedalsOnly} />
          {myCitations.length > 0 && <CitationsSection citations={myCitations} t={t} />}
        </div>
        <ExpectedSection items={expectedItems} title={t.sbDetailExpected} />
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
