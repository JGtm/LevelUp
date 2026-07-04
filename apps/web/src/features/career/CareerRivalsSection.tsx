/**
 * CareerRivalsSection — Top némésis + Top souffre-douleur.
 *
 * Vue butterfly (back-to-back bar chart) :
 *  - Gauche (rouge) : joueurs qui m'ont le plus tué, triés par deaths DESC
 *  - Droite (vert)  : joueurs que j'ai le plus tués, triés par frags DESC
 * Pairing par rang uniquement — les deux listes restent indépendantes.
 * Clic gamertag → Explorer mode joueur.
 */
import { useNavigate, useParams } from '@tanstack/react-router'
import { ratioColorGuarded } from '@/lib/colors/outcomePalette'
import { Spinner } from '@/components/ui/spinner'
import { useCareerRivals } from './queries'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import { useAppShellStore } from '@/stores/appShellStore'
import { tokenCssVar } from '@/lib/accessibility'
import type { CareerRival } from '@/lib/api/types'

function formatRatio(r: number): string {
  if (!Number.isFinite(r)) return '∞'
  return r.toFixed(2)
}

// ─── Butterfly chart ─────────────────────────────────────────────────────────

interface ButterflyRowProps {
  rank: number
  nemesis: CareerRival | undefined
  victim: CareerRival | undefined
  maxDeaths: number
  maxFrags: number
  onPlayerClick: (gamertag: string) => void
}

function ButterflyRow({ rank, nemesis, victim, maxDeaths, maxFrags, onPlayerClick }: ButterflyRowProps) {
  const leftPct = nemesis ? (nemesis.deaths / maxDeaths) * 100 : 0
  const rightPct = victim ? (victim.frags / maxFrags) * 100 : 0

  return (
    <div className="flex items-center h-8">
      {/* LEFT: ratio · deaths · matches  gamertag  [bar→centre] */}
      <div className="flex-1 flex items-center gap-2 min-w-0 h-full">
        {nemesis ? (
          <>
            <span
              className="font-mono font-bold tabular-nums text-xs shrink-0 leading-none"
              style={{ color: ratioColorGuarded(nemesis.deaths, nemesis.ratio) }}
            >
              {formatRatio(nemesis.ratio)}
            </span>
            <span className="text-muted-foreground text-xs leading-none shrink-0">·</span>
            <span className="font-mono shrink-0 leading-none whitespace-nowrap text-xs">
              <span style={{ color: tokenCssVar('outcome-loss') }}>{nemesis.deaths}</span>
              <span className="text-muted-foreground"> · {nemesis.match_count}</span>
            </span>
            <button
              type="button"
              className="font-semibold text-info hover:underline text-xs leading-none shrink-0 whitespace-nowrap"
              onClick={() => onPlayerClick(nemesis.gamertag)}
            >
              {nemesis.gamertag}
            </button>
            <div className="flex-1 min-w-8 h-3 flex items-center justify-end">
              <div
                style={{
                  width: `${leftPct}%`,
                  height: '100%',
                  backgroundColor: tokenCssVar('outcome-loss'),
                  minWidth: leftPct > 0 ? '3px' : '0',
                }}
              />
            </div>
          </>
        ) : (
          <span className="text-xs text-muted-foreground leading-none">—</span>
        )}
      </div>

      {/* CENTRE: rang centré sur la ligne d'axe */}
      <div className="w-8 shrink-0 self-stretch relative flex items-center justify-center">
        <div className="absolute inset-y-0 left-1/2 -translate-x-1/2 w-px bg-border/60" />
        <span className="relative z-10 text-xs font-mono text-muted-foreground leading-none bg-background px-px">
          {rank}
        </span>
      </div>

      {/* RIGHT: [bar←centre]  gamertag  matches · frags · ratio */}
      <div className="flex-1 flex items-center gap-2 min-w-0 h-full">
        {victim ? (
          <>
            <div className="flex-1 min-w-8 h-3 flex items-center justify-start">
              <div
                style={{
                  width: `${rightPct}%`,
                  height: '100%',
                  backgroundColor: tokenCssVar('outcome-win'),
                  minWidth: rightPct > 0 ? '3px' : '0',
                }}
              />
            </div>
            <button
              type="button"
              className="font-semibold text-info hover:underline text-xs leading-none shrink-0 whitespace-nowrap"
              onClick={() => onPlayerClick(victim.gamertag)}
            >
              {victim.gamertag}
            </button>
            <span className="font-mono shrink-0 leading-none whitespace-nowrap text-xs">
              <span className="text-muted-foreground">{victim.match_count} · </span>
              <span style={{ color: tokenCssVar('outcome-win') }}>{victim.frags}</span>
            </span>
            <span className="text-muted-foreground text-xs leading-none shrink-0">·</span>
            <span
              className="font-mono font-bold tabular-nums text-xs shrink-0 leading-none"
              style={{ color: ratioColorGuarded(victim.deaths, victim.ratio) }}
            >
              {formatRatio(victim.ratio)}
            </span>
          </>
        ) : (
          <span className="text-xs text-muted-foreground leading-none">—</span>
        )}
      </div>
    </div>
  )
}

interface ColLabels {
  rank: string
  frags: string
  deaths: string
  ratio: string
  matches: string
}

interface RivalsButterflyChartProps {
  nemeses: CareerRival[]
  victims: CareerRival[]
  nemesisLabel: string
  victimLabel: string
  colLabels: ColLabels
  onPlayerClick: (gamertag: string) => void
}

function ButterflyColHeader({ colLabels }: { colLabels: ColLabels }) {
  const th = 'text-2xs font-semibold text-muted-foreground uppercase tracking-wide leading-none shrink-0 whitespace-nowrap'
  return (
    <div className="flex items-center h-6 border-b border-border">
      <div className="flex-1 flex items-center gap-2 min-w-0">
        <span className={th}>{colLabels.ratio}</span>
        <span className={th}>{colLabels.deaths} · {colLabels.matches}</span>
        <div className="flex-1" />
      </div>
      <div className="w-8 shrink-0 flex items-center justify-center">
        <span className={th}>{colLabels.rank}</span>
      </div>
      <div className="flex-1 flex items-center gap-2 min-w-0">
        <div className="flex-1" />
        <span className={th}>{colLabels.matches} · {colLabels.frags}</span>
        <span className={th}>{colLabels.ratio}</span>
      </div>
    </div>
  )
}

function RivalsButterflyChart({ nemeses, victims, nemesisLabel, victimLabel, colLabels, onPlayerClick }: RivalsButterflyChartProps) {
  const maxDeaths = Math.max(...nemeses.map((n) => n.deaths), 1)
  const maxFrags = Math.max(...victims.map((v) => v.frags), 1)
  const rowCount = Math.max(nemeses.length, victims.length)

  if (rowCount === 0) return null

  return (
    <div className="space-y-1">
      <div className="flex items-center text-xs uppercase tracking-wide font-semibold select-none mb-2">
        <div className="flex-1 text-right pr-10">
          <span style={{ color: tokenCssVar('outcome-loss') }}>← {nemesisLabel}</span>
        </div>
        <div className="w-8 shrink-0" />
        <div className="flex-1 pl-10">
          <span style={{ color: tokenCssVar('outcome-win') }}>{victimLabel} →</span>
        </div>
      </div>
      <ButterflyColHeader colLabels={colLabels} />
      <div className="divide-y divide-border/30">
        {Array.from({ length: rowCount }, (_, i) => (
          <ButterflyRow
            key={i}
            rank={i + 1}
            nemesis={nemeses[i]}
            victim={victims[i]}
            maxDeaths={maxDeaths}
            maxFrags={maxFrags}
            onPlayerClick={onPlayerClick}
          />
        ))}
      </div>
    </div>
  )
}

// ─── Section principale ───────────────────────────────────────────────────────

export function CareerRivalsSection() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = (key: keyof typeof careerManifest) => careerManifest[key][locale]
  const navigate = useNavigate()

  const { data, isLoading, isError } = useCareerRivals(playerSlug)

  const onPlayerClick = (gamertag: string) => {
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: { mode: 'player', target: gamertag },
    })
  }

  const colLabels: ColLabels = {
    rank: t('career.rivals.col_rank'),
    frags: t('career.rivals.col_frags'),
    deaths: t('career.rivals.col_deaths'),
    ratio: t('career.rivals.col_ratio'),
    matches: t('career.rivals.col_matches'),
  }

  return (
    <section className="space-y-3">
      <h2 className="text-sm font-semibold">{t('career.rivals.section_title')}</h2>

      {isLoading && (
        <div className="flex h-24 items-center justify-center">
          <Spinner size="md" />
        </div>
      )}
      {isError && <p className="text-sm text-destructive">{t('career.errors.load_progression_failed')}</p>}
      {!isLoading && !isError && (
        <RivalsButterflyChart
          nemeses={data?.nemeses ?? []}
          victims={data?.victims ?? []}
          nemesisLabel={t('career.rivals.nemesis_title')}
          victimLabel={t('career.rivals.victims_title')}
          colLabels={colLabels}
          onPlayerClick={onPlayerClick}
        />
      )}
    </section>
  )
}
