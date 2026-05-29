/**
 * ExplorerTargetSampleStats — section "Sur N matchs joués ensemble" de l'encart.
 *
 * Layout : 3 colonnes (donut frags à connecteurs / rendement combat chiffré /
 * tuiles KPI) + bandes cadence (par match / par minute) + OutcomeBar légendée.
 *
 * Calcul local depuis common_matches (DuckDB), indépendant des tokens Halo.
 * Affichée seulement quand `sampleStats != null && sample_size > 0`.
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { CombatYieldBar } from '@/components/ui/combat-yield-bar'
import { OutcomeBar } from '@/components/ui/outcome-bar'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerTargetSampleStats } from '@/lib/api/types'

interface ExplorerTargetSampleStatsProps {
  sampleStats: ExplorerTargetSampleStats
}

type TFn = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

interface DonutSlice {
  label: string
  count: number
  token: SemanticToken
}

function fmtPctRatio(value: number, locale: string): string {
  return `${(value * 100).toLocaleString(locale, { maximumFractionDigits: 1 })}%`
}

function fmtNumber(value: number, locale: string, fractionDigits = 2): string {
  return value.toLocaleString(locale, { maximumFractionDigits: fractionDigits })
}

function fmtInt(value: number, locale: string): string {
  return value.toLocaleString(locale, { maximumFractionDigits: 0 })
}

export function ExplorerTargetSampleStats({ sampleStats }: ExplorerTargetSampleStatsProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const locale = appLocale === 'en' ? 'en-US' : 'fr-FR'
  const t: TFn = (key, values) => formatMessage(explorerManifest, key, appLocale, values)

  // Slices du donut (5 catégories). "Other" = kills - somme des catégories
  // détaillées, avec garde-fou si la somme dépasse les kills (overlap).
  const tracked = sampleStats.headshot_kills + sampleStats.melee_kills +
    sampleStats.power_weapon_kills + sampleStats.grenade_kills
  const other = Math.max(0, sampleStats.kills - tracked)
  const donutSlices: DonutSlice[] = [
    { label: t('explorer.target_profile.kill_type_headshot'), count: sampleStats.headshot_kills, token: 'chart-series-1' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_melee'), count: sampleStats.melee_kills, token: 'chart-series-2' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_power_weapon'), count: sampleStats.power_weapon_kills, token: 'chart-series-3' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_grenade'), count: sampleStats.grenade_kills, token: 'chart-series-4' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_other'), count: other, token: 'chart-series-5' as SemanticToken },
  ].filter((s) => s.count > 0)

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">
          {t('explorer.target_profile.section_sample_title', { count: sampleStats.sample_size })}
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="grid gap-4 lg:grid-cols-3">
          <DonutColumn slices={donutSlices} locale={locale} t={t} />
          <YieldColumn sampleStats={sampleStats} t={t} />
          <KpiTiles sampleStats={sampleStats} locale={locale} t={t} />
        </div>
        <CadenceStrips sampleStats={sampleStats} locale={locale} t={t} />
        <OutcomeLegend sampleStats={sampleStats} locale={locale} t={t} />
      </CardContent>
    </Card>
  )
}

// ─── Colonne 1 : donut frags à connecteurs ──────────────────────────────────

function DonutColumn({ slices, locale, t }: { slices: DonutSlice[]; locale: string; t: TFn }) {
  return (
    <div className="flex flex-col items-center gap-2">
      <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">
        {t('explorer.target_profile.label_kill_types')}
      </span>
      {slices.length > 0 ? (
        <KillTypesDonut slices={slices} locale={locale} />
      ) : (
        <span className="text-xs text-muted-foreground">{t('explorer.target_profile.value_unavailable')}</span>
      )}
    </div>
  )
}

// Géométrie du donut. Repère angulaire : 0 = midi, sens horaire.
const DONUT = { w: 300, h: 152, cx: 150, cy: 74, rOuter: 46, stroke: 16, yTop: 16, yBot: 132 }

interface Leader {
  slice: DonutSlice
  startFrac: number
  dashLen: number
  edgeX: number
  edgeY: number
  elbowX: number
  elbowY: number
  kneeX: number
  textX: number
  labelY: number
  anchor: 'start' | 'end'
}

/** computeLeaders calcule arcs + lignes de rappel (labels répartis par côté). */
function computeLeaders(slices: DonutSlice[], total: number, circ: number): Leader[] {
  let acc = 0
  const raw = slices.map((slice) => {
    const startFrac = acc
    const frac = slice.count / total
    acc += frac
    const midTheta = (startFrac + frac / 2) * 2 * Math.PI
    const sinT = Math.sin(midTheta)
    const cosT = Math.cos(midTheta)
    const right = sinT >= 0
    return {
      slice, startFrac, dashLen: frac * circ, right, sinT, cosT,
      edgeX: DONUT.cx + DONUT.rOuter * sinT,
      edgeY: DONUT.cy - DONUT.rOuter * cosT,
      elbowX: DONUT.cx + (DONUT.rOuter + 9) * sinT,
      elbowY: DONUT.cy - (DONUT.rOuter + 9) * cosT,
    }
  })
  const out: Leader[] = []
  for (const right of [true, false]) {
    const side = raw.filter((r) => r.right === right).sort((a, b) => a.elbowY - b.elbowY)
    side.forEach((r, k) => {
      const labelY = side.length === 1
        ? Math.min(Math.max(r.elbowY, DONUT.yTop), DONUT.yBot)
        : DONUT.yTop + (k * (DONUT.yBot - DONUT.yTop)) / (side.length - 1)
      const textX = right ? DONUT.w - 92 : 92
      out.push({
        slice: r.slice, startFrac: r.startFrac, dashLen: r.dashLen,
        edgeX: r.edgeX, edgeY: r.edgeY, elbowX: r.elbowX, elbowY: r.elbowY,
        kneeX: right ? textX - 6 : textX + 6, textX, labelY,
        anchor: right ? 'start' : 'end',
      })
    })
  }
  return out
}

function KillTypesDonut({ slices, locale }: { slices: DonutSlice[]; locale: string }) {
  const total = slices.reduce((acc, s) => acc + s.count, 0)
  if (total === 0) return null
  const innerR = DONUT.rOuter - DONUT.stroke / 2
  const circ = 2 * Math.PI * innerR
  const leaders = computeLeaders(slices, total, circ)

  return (
    <svg width="100%" viewBox={`0 0 ${DONUT.w} ${DONUT.h}`} className="max-w-[300px]">
      <circle cx={DONUT.cx} cy={DONUT.cy} r={innerR} fill="none" stroke={tokenCssVar('perf-tier-5')} strokeWidth={DONUT.stroke} opacity="0.15" />
      {leaders.map((l, i) => (
        <circle
          key={`arc-${i}`}
          cx={DONUT.cx} cy={DONUT.cy} r={innerR} fill="none"
          stroke={tokenCssVar(l.slice.token)} strokeWidth={DONUT.stroke}
          strokeDasharray={`${l.dashLen} ${circ - l.dashLen}`}
          strokeDashoffset={-l.startFrac * circ}
          transform={`rotate(-90 ${DONUT.cx} ${DONUT.cy})`}
        />
      ))}
      <text x={DONUT.cx} y={DONUT.cy} textAnchor="middle" dominantBaseline="central" className="fill-foreground text-base font-semibold">
        {fmtInt(total, locale)}
      </text>
      {leaders.map((l, i) => (
        <g key={`lead-${i}`}>
          <polyline
            points={`${l.edgeX},${l.edgeY} ${l.elbowX},${l.elbowY} ${l.kneeX},${l.labelY}`}
            fill="none" stroke={tokenCssVar(l.slice.token)} strokeWidth="1" opacity="0.55"
          />
          <text x={l.textX} y={l.labelY - 1} textAnchor={l.anchor} className="fill-foreground" opacity="0.8" style={{ fontSize: '8px' }}>
            {l.slice.label}
          </text>
          <text x={l.textX} y={l.labelY + 9} textAnchor={l.anchor} style={{ fill: tokenCssVar(l.slice.token), fontSize: '9px', fontWeight: 600 }}>
            {`${fmtInt(l.slice.count, locale)} · ${fmtPctRatio(l.slice.count / total, locale)}`}
          </text>
        </g>
      ))}
    </svg>
  )
}

// ─── Colonne 2 : rendement combat chiffré ───────────────────────────────────

function YieldColumn({ sampleStats, t }: { sampleStats: ExplorerTargetSampleStats; t: TFn }) {
  const oc = sampleStats.offensive_conversion ?? null
  const dr = sampleStats.defensive_resistance ?? null
  const dash = t('explorer.target_profile.value_unavailable')
  const damagePerKill = sampleStats.kills > 0 ? sampleStats.damage_dealt / sampleStats.kills : null
  const damagePerDeath = sampleStats.deaths > 0 ? sampleStats.damage_taken / sampleStats.deaths : null

  const ocLabel = oc != null ? `${Math.round(oc * 100)}%` : dash
  const drLabel = dr == null ? dash : dr < 0 ? '∞' : `${dr >= 1 ? '+' : ''}${Math.round((dr - 1) * 100)}%`

  return (
    <div className="flex flex-col items-center gap-2">
      <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">
        {t('explorer.target_profile.label_combat_yield')}
      </span>
      <CombatYieldBar
        offensiveConversion={oc}
        defensiveResistance={dr}
        damagePerKill={damagePerKill}
        damagePerDeath={damagePerDeath}
      />
      <div className="flex items-center justify-center gap-4 text-xs font-semibold">
        <span style={{ color: tokenCssVar('divergent-pos') }}>{t('explorer.target_profile.yield_offensive')} {ocLabel}</span>
        <span style={{ color: tokenCssVar('divergent-neutral') }}>{t('explorer.target_profile.yield_defensive')} {drLabel}</span>
      </div>
      {(damagePerKill != null || damagePerDeath != null) && (
        <div className="flex items-center justify-center gap-3 text-2xs text-muted-foreground">
          {damagePerKill != null && <span>{t('explorer.target_profile.yield_dmg_per_kill', { n: Math.round(damagePerKill) })}</span>}
          {damagePerDeath != null && <span>{t('explorer.target_profile.yield_dmg_per_death', { n: Math.round(damagePerDeath) })}</span>}
        </div>
      )}
    </div>
  )
}

// ─── Colonne 3 : tuiles KPI ─────────────────────────────────────────────────

function KpiTiles({ sampleStats, locale, t }: { sampleStats: ExplorerTargetSampleStats; locale: string; t: TFn }) {
  const dash = t('explorer.target_profile.value_unavailable')
  const pct = (v: number | null | undefined) => (v != null ? fmtPctRatio(v, locale) : dash)
  const num = (v: number | null | undefined) => (v != null ? fmtNumber(v, locale) : dash)
  const tiles: SmallTileProps[] = [
    { label: t('explorer.target_profile.label_kda'), value: num(sampleStats.kda) },
    { label: t('explorer.target_profile.label_kdr'), value: num(sampleStats.kdr) },
    { label: t('explorer.target_profile.label_accuracy'), value: pct(sampleStats.accuracy) },
    { label: t('explorer.target_profile.label_headshot_rate'), value: pct(sampleStats.headshot_rate) },
    { label: t('explorer.target_profile.label_win_rate'), value: pct(sampleStats.win_rate) },
    { label: t('explorer.target_profile.label_avg_score'), value: sampleStats.avg_personal_score != null ? fmtInt(sampleStats.avg_personal_score, locale) : dash },
    { label: t('explorer.target_profile.label_perfect_kills'), value: fmtInt(sampleStats.perfect_kills ?? 0, locale) },
  ]
  return (
    <div className="grid grid-cols-2 gap-2">
      {tiles.map((tile) => (
        <SmallTile key={tile.label} label={tile.label} value={tile.value} />
      ))}
    </div>
  )
}

// ─── Bandes cadence : par match / par minute ────────────────────────────────

function CadenceStrips({ sampleStats, locale, t }: { sampleStats: ExplorerTargetSampleStats; locale: string; t: TFn }) {
  const dash = t('explorer.target_profile.value_unavailable')
  const n = sampleStats.sample_size
  const perMatch = (total: number) => (n > 0 ? fmtNumber(total / n, locale, 1) : dash)
  const perMin = (v: number | null | undefined) => (v != null ? fmtNumber(v, locale, 2) : dash)

  return (
    <div className="rounded-md border border-border bg-card px-3 py-2">
      <div className="grid grid-cols-[auto_repeat(3,1fr)] items-center gap-x-3 gap-y-1">
        <span />
        <CadenceHead label={t('explorer.target_profile.cadence_frags')} />
        <CadenceHead label={t('explorer.target_profile.cadence_deaths')} />
        <CadenceHead label={t('explorer.target_profile.cadence_assists')} />
        <CadenceRowLabel label={t('explorer.target_profile.cadence_per_match')} />
        <CadenceValue value={perMatch(sampleStats.kills)} />
        <CadenceValue value={perMatch(sampleStats.deaths)} />
        <CadenceValue value={perMatch(sampleStats.assists)} />
        <CadenceRowLabel label={t('explorer.target_profile.cadence_per_minute')} />
        <CadenceValue value={perMin(sampleStats.kills_per_min)} />
        <CadenceValue value={perMin(sampleStats.deaths_per_min)} />
        <CadenceValue value={perMin(sampleStats.assists_per_min)} />
      </div>
    </div>
  )
}

function CadenceHead({ label }: { label: string }) {
  return <span className="text-center text-2xs uppercase tracking-label-xl text-muted-foreground">{label}</span>
}

function CadenceRowLabel({ label }: { label: string }) {
  return <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">{label}</span>
}

function CadenceValue({ value }: { value: string }) {
  return <span className="text-center text-sm font-semibold text-foreground tabular-nums">{value}</span>
}

// ─── OutcomeBar légendée (V / N / D + taux) ──────────────────────────────────

function OutcomeLegend({ sampleStats, locale, t }: { sampleStats: ExplorerTargetSampleStats; locale: string; t: TFn }) {
  const { wins, draws, losses, win_rate: winRate } = sampleStats
  if (wins + draws + losses === 0) return null
  return (
    <div className="flex flex-col gap-1.5">
      <OutcomeBar wins={wins} draws={draws} losses={losses} />
      <ul className="flex flex-wrap items-center gap-x-3 gap-y-1 text-2xs text-muted-foreground">
        <OutcomeLegendItem token="outcome-win" label={t('explorer.target_profile.outcome_wins')} value={fmtInt(wins, locale)} />
        <OutcomeLegendItem token="outcome-draw" label={t('explorer.target_profile.outcome_draws')} value={fmtInt(draws, locale)} />
        <OutcomeLegendItem token="outcome-loss" label={t('explorer.target_profile.outcome_losses')} value={fmtInt(losses, locale)} />
        {winRate != null && (
          <li className="ml-auto font-semibold text-foreground">{fmtPctRatio(winRate, locale)}</li>
        )}
      </ul>
    </div>
  )
}

function OutcomeLegendItem({ token, label, value }: { token: SemanticToken; label: string; value: string }) {
  return (
    <li className="flex items-center gap-1">
      <span className="h-2 w-2 rounded-full" style={{ backgroundColor: tokenCssVar(token) }} aria-hidden="true" />
      <span>{value} {label}</span>
    </li>
  )
}

// ─── Tuile générique ─────────────────────────────────────────────────────────

interface SmallTileProps {
  label: string
  value: string
}

function SmallTile({ label, value }: SmallTileProps) {
  return (
    <div className="flex flex-col gap-1 rounded-md border border-border bg-card px-2 py-1.5">
      <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">{label}</span>
      <span className="text-sm font-semibold text-foreground">{value}</span>
    </div>
  )
}
