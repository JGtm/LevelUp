/**
 * PlayerDetailPanel — panneau d'expander affiché sous une ligne du scoreboard.
 *
 * Port 1:1 de src/ui/pages/match_view_scoreboard_detail.py (branche main Python).
 *
 * Sections (rendues conditionnellement selon les données disponibles) :
 *   1. Weapons          — top armes du joueur dans ce match (icônes + count)
 *   2. Medals & Citations — médailles + citations progressées (citations uniquement
 *                          pour `is_me` car non disponibles pour les autres joueurs)
 *   3. Expected vs Actual — kills/deaths/assists actual vs expected avec ↑↓→ + delta
 *   4. Antagonist        — Nemesis (top killer du joueur) + Bully (top victim) calculés
 *                          côté front depuis combat_tab.killer_victim
 *   5. Local             — pour `is_me` uniquement : performance score + skill rank
 *                          (LUSR/CSR avec delta) + note bot teammate
 *   6. Footer            — badge DB ("tracké" vs "shared only") + lien Explorer
 *
 * NB : pour les joueurs autres que `is_me`, les sections Local et Citations ne
 * s'affichent pas — leurs données ne sont pas disponibles côté API (calculées
 * uniquement pour le joueur principal dans player_match_enrichment).
 */
import type {
  MatchCitationSnippet,
  MatchKillerVictimPair,
  MatchScoreboardRow,
  MatchViewHeader,
  MatchViewRank,
  PlayerMedalRow,
  PlayerWeaponKillRow,
} from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'
import type { MatchViewText } from './i18n'

const _DETAIL_LIMIT_MEDALS = 5
const _DETAIL_LIMIT_WEAPONS = 5
const _DETAIL_LIMIT_CITATIONS = 4

interface Props {
  /** Ligne complète du scoreboard pour ce joueur. */
  row: MatchScoreboardRow
  /** Pairs killer→victim de tout le match (combat_tab) — sert à calculer Nemesis/Bully. */
  killerVictim?: MatchKillerVictimPair[] | null
  /**
   * Citations uniquement pour le joueur principal (`is_me`). Vide pour les
   * autres. Snippets déjà filtrés (les citations mastered avant ce match
   * sont écartées par BuildCitationSnippets côté backend, ligne 80-83).
   */
  citations?: MatchCitationSnippet[]
  /** Header — utilisé pour performance_display + had_bot_teammate quand `is_me`. */
  header?: MatchViewHeader
  /** Rank — utilisé pour LUSR/CSR + delta quand `is_me`. */
  rank?: MatchViewRank
  /** Slug joueur courant — pour construire le lien Explorer dans le footer. */
  playerSlug?: string
  /** I18n match-view (FR/EN). */
  t: MatchViewText
}

// ---------------------------------------------------------------------------
// Helpers visuels — basés sur tokens accessibilité, pas de hex direct
// ---------------------------------------------------------------------------

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
      {children}
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-md border border-border bg-card/50 p-3">
      <SectionTitle>{title}</SectionTitle>
      {children}
    </section>
  )
}

function KvRow({ label, value, valueClassName = '' }: { label: string; value: React.ReactNode; valueClassName?: string }) {
  return (
    <div className="flex items-center justify-between gap-3 text-xs py-0.5">
      <span className="text-muted-foreground">{label}</span>
      <span className={`text-foreground font-mono ${valueClassName}`}>{value}</span>
    </div>
  )
}

// ---------------------------------------------------------------------------
// 1. Weapons section
// ---------------------------------------------------------------------------

function WeaponsSection({ weapons, title }: { weapons: PlayerWeaponKillRow[]; title: string }) {
  const top = [...weapons].sort((a, b) => b.kills - a.kills).slice(0, _DETAIL_LIMIT_WEAPONS)
  if (top.length === 0) return null
  return (
    <Section title={title}>
      <div className="flex flex-wrap gap-2">
        {top.map((w) => (
          <div
            key={w.weapon_id}
            className="inline-flex items-center gap-1.5 rounded-md border border-border bg-background px-2 py-1 text-xs"
            title={w.label || `#${w.weapon_id}`}
          >
            {w.image_url ? (
              <img
                src={w.image_url}
                alt={w.label ?? ''}
                className="h-5 w-5 object-contain"
                loading="lazy"
              />
            ) : (
              <span className="text-muted-foreground">{w.label ?? `#${w.weapon_id}`}</span>
            )}
            <span className="font-mono font-semibold" style={{ color: tokenCssVar('perf-tier-2') }}>
              {w.kills}
            </span>
          </div>
        ))}
      </div>
    </Section>
  )
}

// ---------------------------------------------------------------------------
// 2. Medals & Citations section
// ---------------------------------------------------------------------------

function MedalsAndCitationsSection({
  medals,
  citations,
  hasCitations,
  t,
}: {
  medals: PlayerMedalRow[]
  citations: MatchCitationSnippet[]
  hasCitations: boolean
  t: MatchViewText
}) {
  const topMedals = [...medals]
    .filter((m) => m.count > 0)
    .sort((a, b) => b.count - a.count || (a.label ?? '').localeCompare(b.label ?? ''))
    .slice(0, _DETAIL_LIMIT_MEDALS)
  // Citations : snippets déjà pré-filtrés (mastered-before exclus côté backend).
  // On garde le top 4 par delta. Tri par delta déjà fait côté backend.
  const topCitations = [...citations]
    .filter((c) => c.delta > 0)
    .slice(0, _DETAIL_LIMIT_CITATIONS)
  if (topMedals.length === 0 && topCitations.length === 0) return null
  const title = hasCitations && topCitations.length > 0 ? t.sbDetailMedalsAndCitations : t.sbDetailMedalsOnly
  return (
    <Section title={title}>
      <div className="flex flex-wrap items-center gap-2">
        {topMedals.map((m) => (
          <div
            key={m.medal_id}
            className="inline-flex items-center gap-1 rounded-md border border-border bg-background px-2 py-1 text-xs"
            title={m.label}
          >
            {m.image_url ? (
              <img src={m.image_url} alt={m.label ?? ''} className="h-5 w-5 object-contain" loading="lazy" />
            ) : (
              <span className="text-muted-foreground">{m.label ?? `#${m.medal_id}`}</span>
            )}
            <span className="font-mono font-semibold">×{m.count}</span>
          </div>
        ))}
        {topMedals.length > 0 && topCitations.length > 0 && (
          <span className="basis-full" aria-hidden />
        )}
        {topCitations.map((c) => (
          <div
            key={c.key}
            className="inline-flex items-center gap-1 rounded-md border border-border bg-background px-2 py-1 text-xs"
            title={c.description ?? c.name}
          >
            {c.image_url ? (
              <img src={c.image_url} alt={c.name} className="h-5 w-5 object-contain" loading="lazy" />
            ) : (
              <span className="text-muted-foreground">{c.name}</span>
            )}
            <span className="font-mono font-semibold">×{c.delta}</span>
            {c.is_newly_mastered && (
              <span className="ml-1 rounded bg-success/30 px-1 text-[9px] font-bold text-success uppercase">
                {t.newlyMastered}
              </span>
            )}
          </div>
        ))}
      </div>
    </Section>
  )
}

// ---------------------------------------------------------------------------
// 3. Expected vs Actual section
// ---------------------------------------------------------------------------

interface ExpectedItem {
  label: string
  actual: number
  expected: number
  higherIsBetter: boolean
}

function ExpectedSection({ items, title }: { items: ExpectedItem[]; title: string }) {
  if (items.length === 0) return null
  return (
    <Section title={title}>
      <div className="space-y-0.5">
        {items.map((it) => {
          const delta = it.actual - it.expected
          const better = it.higherIsBetter ? delta > 0 : delta < 0
          const symbol = delta > 0 ? '↑' : delta < 0 ? '↓' : '→'
          const sign = delta > 0 ? '+' : ''
          // tokens success/destructive : sémantiques, pas de hex direct.
          // Delta == 0 → texte muted via Tailwind, pas de var CSS.
          const deltaCls = delta === 0
            ? 'text-muted-foreground'
            : better
              ? 'text-success'
              : 'text-destructive'
          return (
            <KvRow
              key={it.label}
              label={it.label}
              value={
                <>
                  <span className="text-muted-foreground">{it.expected.toFixed(1)} vs </span>
                  {it.actual.toFixed(0)}{' '}
                  <span className={deltaCls}>
                    {symbol} {sign}{delta.toFixed(1)}
                  </span>
                </>
              }
            />
          )
        })}
      </div>
    </Section>
  )
}

function buildExpectedItems(row: MatchScoreboardRow, t: MatchViewText): ExpectedItem[] {
  const items: ExpectedItem[] = []
  if (row.kills != null && row.expected_kills != null) {
    items.push({ label: t.sbDetailExpectedKills, actual: row.kills, expected: row.expected_kills, higherIsBetter: true })
  }
  if (row.deaths != null && row.expected_deaths != null) {
    items.push({ label: t.sbDetailExpectedDeaths, actual: row.deaths, expected: row.expected_deaths, higherIsBetter: false })
  }
  return items
}

// ---------------------------------------------------------------------------
// 4. Antagonist section (calculée client-side depuis killer_victim)
// ---------------------------------------------------------------------------

interface AntagonistResult {
  nemesisName: string
  nemesisCount: number
  bullyName: string
  bullyCount: number
}

function computeAntagonists(
  xuid: string,
  pairs: MatchKillerVictimPair[],
): AntagonistResult {
  let nemesisName = ''
  let nemesisCount = 0
  let bullyName = ''
  let bullyCount = 0
  for (const p of pairs) {
    if (p.victim_xuid === xuid && p.kill_count > nemesisCount) {
      nemesisCount = p.kill_count
      nemesisName = p.killer_gamertag || p.killer_xuid
    }
    if (p.killer_xuid === xuid && p.kill_count > bullyCount) {
      bullyCount = p.kill_count
      bullyName = p.victim_gamertag || p.victim_xuid
    }
  }
  return { nemesisName, nemesisCount, bullyName, bullyCount }
}

function AntagonistSection({
  result,
  title,
  nemesisLabel,
  bullyLabel,
}: {
  result: AntagonistResult
  title: string
  nemesisLabel: string
  bullyLabel: string
}) {
  if (result.nemesisCount === 0 && result.bullyCount === 0) return null
  return (
    <Section title={title}>
      <div className="space-y-0.5">
        {result.nemesisCount > 0 && (
          <KvRow label={nemesisLabel} value={`${result.nemesisName} (${result.nemesisCount})`} />
        )}
        {result.bullyCount > 0 && (
          <KvRow label={bullyLabel} value={`${result.bullyName} (${result.bullyCount})`} />
        )}
      </div>
    </Section>
  )
}

// ---------------------------------------------------------------------------
// 5. Local section (perf score + skill rank + bot teammate) — only is_me
// ---------------------------------------------------------------------------

interface LocalRow {
  perfDisplay?: string
  perfColorToken?: string
  ratingType?: string
  tierLabel?: string
  ratingDelta?: number | null
  hadBotTeammate?: boolean
}

/**
 * Construit les données "Local" pour une ligne du scoreboard.
 *
 * Pour `is_me` : prend en priorité les valeurs déjà calculées par le header
 * (header.performance_display, header.performance_color_token,
 * header.had_bot_teammate) car elles incluent le formatage "perf-tier-N" du
 * service. Le rank vient de `rank` (load via match_skill_rank du main).
 *
 * Pour les amis (row.performance_score / row.skill_rank populés par le
 * FriendsExtrasResolver) : on construit perfDisplay à la volée depuis le
 * float (le color token n'est pas calculé côté backend pour les amis donc on
 * le laisse vide — la valeur s'affiche en couleur foreground neutre).
 */
function buildLocalRow(row: MatchScoreboardRow, header?: MatchViewHeader, mainRank?: MatchViewRank): LocalRow | null {
  const local: LocalRow = {}
  let hasData = false
  if (row.is_me) {
    if (header?.performance_display) {
      local.perfDisplay = header.performance_display
      local.perfColorToken = header.performance_color_token
      hasData = true
    }
    if (mainRank?.tier_label) {
      local.ratingType = mainRank.rating_type
      local.tierLabel = mainRank.tier_label
      local.ratingDelta = mainRank.delta_value
      hasData = true
    }
    if (header?.had_bot_teammate) {
      local.hadBotTeammate = true
      hasData = true
    }
  } else {
    if (row.performance_score != null) {
      local.perfDisplay = Math.round(row.performance_score).toString()
      hasData = true
    }
    if (row.skill_rank?.tier_label) {
      local.ratingType = row.skill_rank.rating_type
      local.tierLabel = row.skill_rank.tier_label
      local.ratingDelta = row.skill_rank.rating_delta
      hasData = true
    }
    if (row.had_bot_teammate) {
      local.hadBotTeammate = true
      hasData = true
    }
  }
  return hasData ? local : null
}

function LocalSection({ data, t }: { data: LocalRow; t: MatchViewText }) {
  const rows: React.ReactNode[] = []
  if (data.perfDisplay) {
    const colorVar = data.perfColorToken ? `var(--ac-${data.perfColorToken})` : undefined
    rows.push(
      <KvRow
        key="perf"
        label={t.performance}
        value={<span style={{ color: colorVar }}>{data.perfDisplay}</span>}
      />,
    )
  }
  if (data.tierLabel) {
    const label = data.ratingType === 'CSR' ? t.sbDetailCsr : t.sbDetailLusr
    let display = data.tierLabel
    if (data.ratingDelta != null) {
      const sign = data.ratingDelta >= 0 ? '+' : ''
      display = `${data.tierLabel} (${sign}${data.ratingDelta.toFixed(0)} pts)`
    }
    rows.push(<KvRow key="rank" label={label} value={display} />)
  }
  if (data.hadBotTeammate) {
    rows.push(<KvRow key="bot" label={t.sbDetailBotNoteLabel} value={t.sbDetailBotNoteValue} />)
  }
  if (rows.length === 0) return null
  return (
    <Section title={t.sbDetailLocal}>
      <div className="space-y-0.5">{rows}</div>
    </Section>
  )
}

// ---------------------------------------------------------------------------
// 6. Footer (DB badge + Explorer link)
// ---------------------------------------------------------------------------

function Footer({
  isTracked,
  isMe,
  isBot,
  gamertag,
  playerSlug,
  t,
}: {
  isTracked: boolean
  isMe: boolean
  isBot: boolean
  gamertag: string
  playerSlug?: string
  t: MatchViewText
}) {
  const badgeText = isTracked ? t.sbDetailPlayerDb : t.sbDetailSharedOnly
  const showLink = !isMe && !isBot && !!playerSlug
  const explorerUrl = showLink
    ? `/players/${playerSlug}/explorer?mode=player&target=${encodeURIComponent(gamertag)}`
    : null
  return (
    <div className="flex items-center justify-between gap-2 border-t border-border pt-2 text-[11px]">
      <span className="rounded bg-muted px-2 py-0.5 text-muted-foreground">{badgeText}</span>
      {explorerUrl && (
        <a
          href={explorerUrl}
          className="text-info hover:underline"
        >
          {t.sbDetailExplorePlayerFmt(gamertag)}
        </a>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// PlayerDetailPanel — composant principal
// ---------------------------------------------------------------------------

export function PlayerDetailPanel({
  row,
  killerVictim,
  citations,
  header,
  rank,
  playerSlug,
  t,
}: Props) {
  const weapons = row.weapon_kills ?? []
  const medals = row.medals ?? []
  const myCitations = row.is_me ? (citations ?? []) : []
  const expectedItems = buildExpectedItems(row, t)
  const antagonist = killerVictim ? computeAntagonists(row.xuid, killerVictim) : null
  // Section "Local" : pour `is_me` (depuis header + rank) et pour les amis
  // dont le backend a populé row.performance_score / row.skill_rank via le
  // FriendsExtrasResolver (cf. registry.MatchView).
  const localData = buildLocalRow(row, header, rank)
  // Le footer affiche "Tracké" dès que le joueur a une player DB côté backend
  // (i.e. `is_me` OU `localData != null`), aligné sur le Python qui checke
  // `player_db_exists(gamertag)`.
  const isTracked = row.is_me || localData != null

  return (
    <div className="bg-card/80 border border-border rounded-b px-4 py-3 space-y-3">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <WeaponsSection weapons={weapons} title={t.sbDetailWeapons} />
        <MedalsAndCitationsSection
          medals={medals}
          citations={myCitations}
          hasCitations={row.is_me}
          t={t}
        />
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
      <Footer
        isTracked={isTracked}
        isMe={row.is_me}
        isBot={row.is_bot ?? false}
        gamertag={row.gamertag}
        playerSlug={playerSlug}
        t={t}
      />
    </div>
  )
}
