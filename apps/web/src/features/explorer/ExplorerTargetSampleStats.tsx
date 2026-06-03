/**
 * ExplorerTargetSampleStats — section "Sur N matchs joués ensemble" de l'encart.
 *
 * Layout : 3 colonnes (donut frags à connecteurs / rendement combat chiffré /
 * tuiles KPI) + bandes cadence (par match / par minute) + OutcomeBar légendée.
 *
 * Calcul local depuis common_matches (DuckDB), indépendant des tokens Halo.
 * Affichée seulement quand `sampleStats != null && sample_size > 0`.
 */
import { OutcomeBar } from '@/components/ui/outcome-bar'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerTargetSampleStats, ExplorerWeaponKill } from '@/lib/api/types'

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

  // Partition des frags par TYPE D'ARME (mutuellement exclusifs) : melee / arme
  // lourde / grenade / autres → la somme = total des frags. "Autres" = frags à
  // l'arme normale = kills - (melee + lourde + grenade).
  //
  // IMPORTANT : on ne soustrait PAS les headshots. Un headshot est ORTHOGONAL au
  // type d'arme (un frag à l'arme normale ou lourde peut être un headshot) ; le
  // compter ici ferait que le donut ne somme plus au total des frags. Les
  // "Tirs à la tête" restent exposés en KPI ("Taux de tête"), hors donut.
  const weaponTyped = sampleStats.melee_kills + sampleStats.power_weapon_kills + sampleStats.grenade_kills
  const other = Math.max(0, sampleStats.kills - weaponTyped)
  //
  // Couleurs : indices chart-series DISTINCTS (1/6/7/8) et NON 2-5 — dans la
  // palette par défaut 1..5 est un dégradé bleu/indigo séquentiel (illisible en
  // catégoriel, pas color-blind friendly). 1/6/7/8 sont distincts dans toutes les
  // palettes (Okabe-Ito CB incluse). Via tokenCssVar → suit la palette active.
  const donutSlices: DonutSlice[] = [
    { label: t('explorer.target_profile.kill_type_melee'), count: sampleStats.melee_kills, token: 'chart-series-1' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_power_weapon'), count: sampleStats.power_weapon_kills, token: 'chart-series-6' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_grenade'), count: sampleStats.grenade_kills, token: 'chart-series-7' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_other'), count: other, token: 'chart-series-8' as SemanticToken },
  ].filter((s) => s.count > 0)

  // Bloc « Répartition des frags » : titre en barre (chrome ChartCard) + donut
  // agrandi centré. Hauteur naturelle : le bloc cohabite avec le bilan V/N/D dans
  // la même colonne (cf. ExplorerTargetProfileCard). Titre de section + rangée KPI
  // rendus hors bloc.
  const topWeapons = sampleStats.top_weapons ?? []
  return (
    <div className="flex flex-col overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
        {t('explorer.target_profile.label_kill_types')}
      </div>
      {/* Donut (gauche) + top 3 armes (droite) si dispo, sinon donut centré seul. */}
      <div className={`grid items-center gap-3 p-3 ${topWeapons.length > 0 ? 'sm:grid-cols-[2fr_1fr]' : ''}`}>
        <DonutColumn slices={donutSlices} locale={locale} t={t} />
        {topWeapons.length > 0 && <WeaponsTop weapons={topWeapons} locale={appLocale} t={t} />}
      </div>
    </div>
  )
}

// ─── Top armes (à droite du donut) ───────────────────────────────────────────

function WeaponsTop({ weapons, locale, t }: { weapons: ExplorerWeaponKill[]; locale: 'fr' | 'en'; t: TFn }) {
  const numberLocale = locale === 'en' ? 'en-US' : 'fr-FR'
  const maxKills = Math.max(1, ...weapons.map((w) => w.kills))
  return (
    <div className="flex flex-col gap-2">
      <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">
        {t('explorer.target_profile.top_weapons_title')}
      </span>
      <ol className="flex flex-col gap-2">
        {weapons.map((w, i) => {
          const name = locale === 'en' ? w.label_en || w.label_fr : w.label_fr || w.label_en
          const pct = Math.round((w.kills / maxKills) * 100)
          return (
            <li key={w.weapon_id} className="flex flex-col gap-1">
              <div className="flex items-baseline justify-between gap-2">
                <span className="flex min-w-0 items-baseline gap-1.5">
                  <span className="text-2xs font-bold tabular-nums text-muted-foreground">{i + 1}</span>
                  <span className="truncate text-xs font-medium text-foreground">{name}</span>
                </span>
                <span className="flex-shrink-0 text-xs font-semibold tabular-nums text-foreground">
                  {w.kills.toLocaleString(numberLocale)}
                </span>
              </div>
              <div className="h-1 w-full overflow-hidden rounded-full bg-muted-foreground/15">
                <div
                  className="h-full rounded-full"
                  style={{ width: `${pct}%`, backgroundColor: tokenCssVar('chart-series-1') }}
                />
              </div>
            </li>
          )
        })}
      </ol>
    </div>
  )
}

/**
 * ExplorerTargetOutcome — bilan V/N/D des matchs communs, rendu dans une section
 * séparée pleine largeur sous le donut + cadence (OutcomeBar + légende). nil si
 * aucun résultat exploitable.
 */
export function ExplorerTargetOutcome({ sampleStats }: ExplorerTargetSampleStatsProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const locale = appLocale === 'en' ? 'en-US' : 'fr-FR'
  const t: TFn = (key, values) => formatMessage(explorerManifest, key, appLocale, values)
  if (sampleStats.wins + sampleStats.draws + sampleStats.losses === 0) return null
  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card">
      <div className="border-b border-border px-3 py-2 text-sm font-medium">
        {t('explorer.target_profile.results_title')}
      </div>
      <div className="p-3">
        <OutcomeLegend sampleStats={sampleStats} locale={locale} t={t} />
      </div>
    </div>
  )
}

// ─── Colonne 1 : donut frags à connecteurs ──────────────────────────────────

function DonutColumn({ slices, locale, t }: { slices: DonutSlice[]; locale: string; t: TFn }) {
  return (
    <div className="flex w-full flex-col items-center gap-2">
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
    <svg width="100%" viewBox={`0 0 ${DONUT.w} ${DONUT.h}`} className="max-w-[700px]">
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

// ─── Rangée KPI (hors bloc, sous le titre) ───────────────────────────────────

/**
 * ExplorerTargetSampleKpis — rangée de KPI cards rendue HORS du bloc, juste sous
 * le titre « Sur N matchs joués ensemble » (parité avec la rangée de tuiles de
 * « Carrière complète »). La dernière carte porte le rendement/résistance (OC/DR)
 * à la place de l'ancienne barre composite.
 */
export function ExplorerTargetSampleKpis({ sampleStats }: ExplorerTargetSampleStatsProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const locale = appLocale === 'en' ? 'en-US' : 'fr-FR'
  const t: TFn = (key, values) => formatMessage(explorerManifest, key, appLocale, values)
  const dash = t('explorer.target_profile.value_unavailable')
  const pct = (v: number | null | undefined) => (v != null ? fmtPctRatio(v, locale) : dash)
  const num = (v: number | null | undefined) => (v != null ? fmtNumber(v, locale) : dash)

  const oc = sampleStats.offensive_conversion ?? null
  const dr = sampleStats.defensive_resistance ?? null
  const ocLabel = oc != null ? `${Math.round(oc * 100)}%` : dash
  const drLabel = dr == null ? dash : dr < 0 ? '∞' : `${dr >= 1 ? '+' : ''}${Math.round((dr - 1) * 100)}%`
  const dmgPerKill = sampleStats.kills > 0 ? Math.round(sampleStats.damage_dealt / sampleStats.kills) : null
  const dmgPerDeath = sampleStats.deaths > 0 ? Math.round(sampleStats.damage_taken / sampleStats.deaths) : null

  // lg : KDA (1re piste) réduite à 0.7fr, Rendement/Résistance (dernière) élargie à 1.3fr.
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-[0.7fr_1fr_1fr_1fr_1fr_1.3fr]">
      <SmallTile label={t('explorer.target_profile.label_kda')} value={num(sampleStats.kda)} accent="perf-tier-2" />
      <SmallTile label={t('explorer.target_profile.label_accuracy')} value={pct(sampleStats.accuracy)} accent="info" />
      <SmallTile label={t('explorer.target_profile.label_headshot_rate')} value={pct(sampleStats.headshot_rate)} accent="chart-series-1" />
      <SmallTile label={t('explorer.target_profile.label_avg_score')} value={sampleStats.avg_personal_score != null ? fmtInt(sampleStats.avg_personal_score, locale) : dash} accent="chart-series-4" />
      <SmallTile label={t('explorer.target_profile.label_perfect_kills')} value={fmtInt(sampleStats.perfect_kills ?? 0, locale)} accent="outcome-win" />
      <YieldTile ocLabel={ocLabel} drLabel={drLabel} dmgPerKill={dmgPerKill} dmgPerDeath={dmgPerDeath} t={t} />
    </div>
  )
}

/**
 * YieldTile — carte rendement (OC, vert) / résistance (DR, bleu) au format KPI card.
 * dmg/frag (gauche) et dmg/mort (droite) en pied, petit + gris (aux extrémités).
 */
function YieldTile({
  ocLabel,
  drLabel,
  dmgPerKill,
  dmgPerDeath,
  t,
}: {
  ocLabel: string
  drLabel: string
  dmgPerKill: number | null
  dmgPerDeath: number | null
  t: TFn
}) {
  return (
    <div className="overflow-hidden rounded-md border border-border bg-card">
      <div className="h-[3px]" style={{ backgroundColor: tokenCssVar('divergent-pos') }} />
      <div className="flex flex-col gap-1 px-2 py-1.5">
        <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">
          {t('explorer.target_profile.yield_offensive')} / {t('explorer.target_profile.yield_defensive')}
        </span>
        {/* Une seule ligne : dmg/frag (gauche) · valeurs OC/DR (centre) · dmg/mort (droite). */}
        <div className="flex items-baseline justify-between gap-1">
          <span className="text-3xs text-muted-foreground">
            {dmgPerKill != null ? t('explorer.target_profile.yield_dmg_per_kill', { n: dmgPerKill }) : ''}
          </span>
          <span className="text-sm font-semibold">
            <span style={{ color: tokenCssVar('divergent-pos') }}>{ocLabel}</span>
            <span className="text-muted-foreground"> / </span>
            <span style={{ color: tokenCssVar('divergent-neutral') }}>{drLabel}</span>
          </span>
          <span className="text-3xs text-muted-foreground">
            {dmgPerDeath != null ? t('explorer.target_profile.yield_dmg_per_death', { n: dmgPerDeath }) : ''}
          </span>
        </div>
      </div>
    </div>
  )
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
  /** Couleur de la barre d'accent (3px) en haut — parité Synthesis AccentCard. */
  accent: SemanticToken
}

function SmallTile({ label, value, accent }: SmallTileProps) {
  return (
    <div className="overflow-hidden rounded-md border border-border bg-card">
      <div className="h-[3px]" style={{ backgroundColor: tokenCssVar(accent) }} />
      <div className="flex flex-col gap-1 px-2 py-1.5">
        <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">{label}</span>
        <span className="text-sm font-semibold text-foreground">{value}</span>
      </div>
    </div>
  )
}
