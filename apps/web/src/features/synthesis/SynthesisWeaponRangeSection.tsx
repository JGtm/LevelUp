/**
 * SynthesisWeaponRangeSection — LA PORTÉE DES ENGAGEMENTS SUR LA SYNTHÈSE.
 *
 * Transposition de la maquette validée par l'utilisateur le 2026-09-06
 * (`.ai/V7.5/MAQUETTE_PORTEE_ENGAGEMENTS_2026-09-06.html`, lot 5 du plan
 * `.ai/PLAN_DUELS_PORTEE_2026-09-06.md`) : quatre tuiles de tête, puis UNE carte qui porte
 * les deux graphes jumeaux sur les MÊMES lignes — la portée (deux bâtons p10→p90 par arme,
 * frags au-dessus, morts en dessous) et le dénivelé (deux barres empilées à 100 %).
 *
 * CE COMPOSANT NE CALCULE RIEN. La projection et les deux options ECharts vivent dans
 * `_weaponRangeChart.ts` / `_weaponElevationChart.ts`, les décisions de lecture dans
 * `weaponRange_logic.ts` — tous purs, tous testés hors rendu.
 *
 * LES DÉNOMINATEURS SONT AFFICHÉS PARTOUT, et c'est le point : la mesure est partielle par
 * construction (seuls les frags dont les DEUX positions sont décodées comptent). Chaque
 * tuile porte le sien, la carte porte sa note de couverture, et les armes écartées par le
 * seuil de publication sont NOMMÉES plutôt que tues.
 */
import { useCallback, useMemo, type ReactNode } from 'react'

import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { SectionCard } from '@/components/ui/section-card'
import { resolveToken, tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import type { SynthesisWeaponRange } from '@/lib/api/types'
import { cssColorToHex } from '@/lib/echarts/cssColorToHex'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { useAppShellStore } from '@/stores/appShellStore'

import { AccentCard, SectionSubtitle } from './SynthesisCards'
import {
  ELEVATION_KEYS,
  buildWeaponElevationOption,
  type ElevationKey,
} from './_weaponElevationChart'
import {
  buildWeaponRangeOption,
  weaponRangeChartHeight,
  weaponRangeLines,
  type WeaponRangeLine,
} from './_weaponRangeChart'
import { SynthesisWeaponRangeTable } from './SynthesisWeaponRangeTable'
import {
  WEAPON_RANGE_MIN_MEASURED,
  belowThresholdNames,
  hasWeaponRangeRows,
} from './weaponRange_logic'
import { useRangeFormats, type RangeFormats, type Translate } from './weaponRangeText'

/** Encres de la section — un seul endroit, partagé par les graphes et les deux légendes. */
const KILLS_TOKEN: SemanticToken = 'chart-series-1'
const DEATHS_TOKEN: SemanticToken = 'chart-series-3'
const MEDIAN_TOKEN: SemanticToken = 'perf-tier-2'
const DELTA_TOKEN: SemanticToken = 'chart-series-4'

/** Les trois libellés de classe, résolus une fois — graphe ET légende lisent la même source. */
function elevationLabels(t: Translate): Record<ElevationKey, string> {
  return {
    above: t('synthesis.weapon_range.elev_above'),
    level: t('synthesis.weapon_range.elev_level'),
    below: t('synthesis.weapon_range.elev_below'),
  }
}

// ─── Tuiles de tête ───────────────────────────────────────────────────────────

function RangeTiles({ range, t, f }: { range: SynthesisWeaponRange; t: Translate; f: RangeFormats }) {
  const opening = range.opening
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
      {/* MÊME DOCTRINE QUE L'ENTAME (décision D5) : une tuile de portée n'existe QUE si son
          côté est mesuré. Le service ne retire le bloc que si AUCUN des deux côtés n'a de
          frag mesuré ; un côté vide arrive donc avec `median_*_m: 0` et `measured_*: 0`, et
          « 0,0 m » en gros se lirait « ce joueur frague au contact » au lieu de « on ne sait
          pas ». Un scope « morts seulement » n'affiche qu'une tuile de portée. */}
      {range.measured_kills > 0 && (
        <AccentCard
          label={t('synthesis.weapon_range.tile_median_kills')}
          value={f.distance(range.median_kills_m)}
          accent={KILLS_TOKEN}
          sub={t('synthesis.weapon_range.tile_median_kills_sub', {
            measured: range.measured_kills,
            total: range.total_kills,
          })}
        />
      )}
      {range.measured_deaths > 0 && (
        <AccentCard
          label={t('synthesis.weapon_range.tile_median_deaths')}
          value={f.distance(range.median_deaths_m)}
          accent={DEATHS_TOKEN}
          sub={t('synthesis.weapon_range.tile_median_deaths_sub', {
            measured: range.measured_deaths,
            total: range.total_deaths,
          })}
        />
      )}
      {/* Les deux tuiles d'entame n'existent QUE si l'entame est mesurée : le bloc est nil
          tant que le backfill de `kill_openings` n'a pas tourné, et un zéro se lirait
          « ce joueur engage au contact » au lieu de « on ne sait pas » (décision D5). */}
      {opening && (
        <AccentCard
          label={t('synthesis.weapon_range.tile_opening')}
          value={f.distance(opening.median_m)}
          accent={MEDIAN_TOKEN}
          sub={t('synthesis.weapon_range.tile_opening_sub', { measured: opening.measured_kills })}
        />
      )}
      {/* La tuile « Entame → frag » suit `opening.delta`, PAS `opening` : une entame peut
          être mesurée sans qu'aucun frag n'ait pu lui être apparié — le delta n'existe alors
          pas, et un zéro dirait « la distance ne bouge pas ». */}
      {opening?.delta && (
        <AccentCard
          label={t('synthesis.weapon_range.tile_delta')}
          value={f.signedDistance(opening.delta.median_m)}
          accent={DELTA_TOKEN}
          sub={t('synthesis.weapon_range.tile_delta_sub', {
            pct: f.percent(opening.delta.closing_share_pct),
          })}
        />
      )}
    </div>
  )
}

// ─── Légendes HTML ────────────────────────────────────────────────────────────

/**
 * Une entrée de légende porte TROIS choses : la pastille, le libellé, et la POSITION de la
 * série dans le graphe (« bâton du haut »). L'identité n'est jamais confiée à la couleur
 * seule — un daltonien lit la position et le nom.
 *
 * `swatchClass` sert la seule classe qui n'a pas de token d'accessibilité : « à niveau »
 * emprunte le gris des libellés d'axe (`--muted-foreground`), la MÊME encre que le graphe.
 *
 * `emphasis` suit la maquette validée : seule la légende de PORTÉE met son libellé en avant
 * (deux séries à distinguer, chacune suivie de sa position entre parenthèses) ; celle du
 * dénivelé rend ses trois noms NUS, dans le gris du texte secondaire — trois classes d'une
 * même mesure, qu'aucune ne doit dominer.
 */
function LegendItem({
  color,
  swatchClass,
  name,
  hint,
  emphasis,
}: {
  color?: string
  swatchClass?: string
  name: string
  hint?: string
  emphasis?: boolean
}) {
  return (
    <li className="inline-flex items-center gap-1.5">
      <span
        aria-hidden="true"
        className={`inline-block h-2 w-3 rounded-sm ${swatchClass ?? ''}`}
        style={color ? { backgroundColor: color } : undefined}
      />
      {emphasis ? <b className="font-medium text-foreground">{name}</b> : <span>{name}</span>}
      {hint && <span>{' '}{hint}</span>}
    </li>
  )
}

function SubtitleRow({
  title,
  detail,
  children,
}: {
  title: string
  detail: string
  children: ReactNode
}) {
  return (
    <div className="flex flex-wrap items-baseline justify-between gap-3 px-3 pt-2.5">
      <p className="text-xs font-semibold text-foreground">
        {title} <span className="font-normal text-muted-foreground">{detail}</span>
      </p>
      {children}
    </div>
  )
}

function RangeLegend({ t }: { t: Translate }) {
  return (
    <ul
      aria-label={t('synthesis.weapon_range.legend_label')}
      className="flex flex-wrap gap-3.5 text-3xs text-muted-foreground"
    >
      <LegendItem
        emphasis
        color={tokenCssVar(KILLS_TOKEN)}
        name={t('synthesis.weapon_range.side_kills')}
        hint={t('synthesis.weapon_range.side_kills_position')}
      />
      <LegendItem
        emphasis
        color={tokenCssVar(DEATHS_TOKEN)}
        name={t('synthesis.weapon_range.side_deaths')}
        hint={t('synthesis.weapon_range.side_deaths_position')}
      />
    </ul>
  )
}

function ElevationLegend({ t }: { t: Translate }) {
  const labels = elevationLabels(t)
  const swatch = (key: ElevationKey) => {
    if (key === 'above') return { color: tokenCssVar(DEATHS_TOKEN) }
    if (key === 'below') return { color: tokenCssVar(KILLS_TOKEN) }
    return { swatchClass: 'bg-muted-foreground' }
  }
  return (
    <ul
      aria-label={t('synthesis.weapon_range.legend_elevation_label')}
      className="flex flex-wrap gap-3.5 text-3xs text-muted-foreground"
    >
      {ELEVATION_KEYS.map((key) => (
        <LegendItem key={key} {...swatch(key)} name={labels[key]} />
      ))}
    </ul>
  )
}

// ─── Pied de carte : seuil, couverture, tableau ───────────────────────────────

function RangeFooter({
  range,
  lines,
  locale,
  t,
  f,
}: {
  range: SynthesisWeaponRange
  lines: WeaponRangeLine[]
  locale: ManifestLocale
  t: Translate
  f: RangeFormats
}) {
  const belowKills = belowThresholdNames(range.below_threshold_kills, locale, f.count)
  const belowDeaths = belowThresholdNames(range.below_threshold_deaths, locale, f.count)
  const halves = [
    belowKills && t('synthesis.weapon_range.below_threshold_kills', { names: belowKills }),
    belowDeaths && t('synthesis.weapon_range.below_threshold_deaths', { names: belowDeaths }),
  ].filter((s): s is string => Boolean(s))
  return (
    <>
      {halves.length > 0 && (
        <p className="mt-1.5 border-t border-border px-3 pt-2 text-3xs text-muted-foreground">
          <b className="font-medium text-foreground">
            {t('synthesis.weapon_range.below_threshold_title', { min: WEAPON_RANGE_MIN_MEASURED })}
          </b>
          {` — ${halves.join(' · ')}`}
        </p>
      )}
      <p className="px-3 pb-2 pt-1 text-3xs text-muted-foreground">
        {t('synthesis.weapon_range.coverage_note')}
      </p>
      {lines.length > 0 && (
        <details className="pb-2">
          <summary className="cursor-pointer px-3 py-1 text-xs text-muted-foreground">
            {t('synthesis.weapon_range.table_summary')}
          </summary>
          <div className="overflow-x-auto px-3 pb-1">
            <SynthesisWeaponRangeTable lines={lines} t={t} f={f} />
          </div>
        </details>
      )}
    </>
  )
}

// ─── Options ECharts ──────────────────────────────────────────────────────────

/**
 * useWeaponRangeOptions — les deux constructeurs d'option, résolus au thème courant.
 *
 * Les tokens ne sont résolus qu'ICI, à l'appel : ECharts peint dans un canvas, qui n'accepte
 * pas de variable CSS. `ChartCard` rappelle ces fonctions à chaque bascule de thème ou de
 * palette d'accessibilité, et les couleurs suivent.
 */
function useWeaponRangeOptions(lines: WeaponRangeLine[], f: RangeFormats, t: Translate) {
  const buildRange = useCallback(() => {
    const tc = getEChartsThemeColors()
    return buildWeaponRangeOption({
      lines,
      tc,
      killsColor: resolveToken(KILLS_TOKEN),
      deathsColor: resolveToken(DEATHS_TOKEN),
      medianColor: resolveToken(MEDIAN_TOKEN),
      cardColor: tc.card,
      fmtDistance: f.distance,
      labels: {
        kills: t('synthesis.weapon_range.side_kills'),
        deaths: t('synthesis.weapon_range.side_deaths'),
        percentiles: t('synthesis.weapon_range.percentiles'),
        noMeasure: t('synthesis.weapon_range.no_measure'),
      },
    })
  }, [lines, f, t])

  const buildElevation = useCallback(() => {
    const tc = getEChartsThemeColors()
    return buildWeaponElevationOption({
      lines,
      tc,
      // La couleur NE JUGE PAS : une seule teinte du clair (d'en bas) au foncé (d'en haut),
      // et le gris des libellés d'axe pour « à niveau » — la MÊME encre que sa pastille.
      //
      // `cssColorToHex` SUR CETTE SEULE COULEUR, et c'est délibéré : les deux autres viennent
      // de la palette d'accessibilité, dont les valeurs sont déjà des hex (`palettes/*.ts`),
      // tandis que `--muted-foreground` est un `oklch(...)` que le parseur de zrender ne sait
      // pas lire — au survol, `lift()` rendait `undefined` et le segment « à niveau » perdait
      // son remplissage. Normalisation par le navigateur, cf. `lib/echarts/cssColorToHex.ts`.
      colors: {
        above: resolveToken(DEATHS_TOKEN),
        level: cssColorToHex(tc.axisLabel),
        below: resolveToken(KILLS_TOKEN),
      },
      cardColor: tc.card,
      fmtPercent: f.percent,
      labels: {
        kills: t('synthesis.weapon_range.side_kills'),
        deaths: t('synthesis.weapon_range.side_deaths'),
        noMeasure: t('synthesis.weapon_range.no_measure'),
        segments: elevationLabels(t),
      },
    })
  }, [lines, f, t])

  return { buildRange, buildElevation }
}

/**
 * RangeCardBody — les deux graphes, OU la phrase qui explique pourquoi il n'y en a pas.
 *
 * Le second cas est nominal (toutes les armes sous le seuil de publication) : la carte garde
 * ses tuiles, ses armes nommées et sa note de couverture, et le corps DIT pourquoi il est
 * vide. Un graphe sans barre ne se lit pas « rien à montrer », il se lit « bug ».
 */
function RangeCardBody({
  publiable,
  series,
  buildRange,
  buildElevation,
  height,
  t,
}: {
  publiable: boolean
  series: ChartSeries<WeaponRangeLine>[]
  buildRange: () => EChartsCoreOption
  buildElevation: () => EChartsCoreOption
  height: number
  t: Translate
}) {
  if (!publiable) {
    return (
      <p className="px-3 pb-1 pt-2.5 text-xs text-muted-foreground">
        {t('synthesis.weapon_range.empty_below_threshold', { min: WEAPON_RANGE_MIN_MEASURED })}
      </p>
    )
  }
  return (
    <>
            <SubtitleRow
              title={t('synthesis.weapon_range.range_subtitle')}
              detail={t('synthesis.weapon_range.range_subtitle_detail')}
            >
              <RangeLegend t={t} />
            </SubtitleRow>
            <ChartCard
              series={series}
              buildOption={buildRange}
              height={height}
              className="border-0 shadow-none"
            />

            <SubtitleRow
              title={t('synthesis.weapon_range.elevation_subtitle')}
              detail={t('synthesis.weapon_range.elevation_subtitle_detail')}
            >
              <ElevationLegend t={t} />
            </SubtitleRow>
            <ChartCard
              series={series}
              buildOption={buildElevation}
              height={height}
              className="border-0 shadow-none"
            />
          </>

  )
}

// ─── Section ──────────────────────────────────────────────────────────────────

export interface SynthesisWeaponRangeSectionProps {
  range: SynthesisWeaponRange | null | undefined
}

export function SynthesisWeaponRangeSection({ range }: SynthesisWeaponRangeSectionProps) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = useCallback<Translate>(
    (key, vars) => formatMessage(synthesisManifest, key, locale, vars),
    [locale],
  )
  const f = useRangeFormats(locale)
  const weapons = range?.weapons
  const lines = useMemo(() => weaponRangeLines(weapons, locale), [weapons, locale])
  const series = useMemo(() => [{ key: 'weapon-range', datapoints: lines }], [lines])

  const { buildRange, buildElevation } = useWeaponRangeOptions(lines, f, t)

  // SEULE L'ABSENCE DE BLOC RETIRE LA SECTION. Un bloc SANS arme publiable est un cas
  // NOMINAL — un joueur dont toutes les armes restent sous le seuil de 8 mesures : ses
  // médianes globales et ses armes écartées existent, seuls les deux graphes n'ont rien à
  // tracer, et ils le DISENT (consigne du pilote, revue du lot 4). Le service, lui, omet
  // déjà le bloc quand rien n'est mesuré du tout.
  if (!range) return null
  const publiable = hasWeaponRangeRows(range)
  const height = weaponRangeChartHeight(lines.length)

  return (
    <section className="space-y-3">
      <SectionSubtitle>{t('synthesis.weapon_range.heading')}</SectionSubtitle>
      <p className="max-w-[68ch] text-xs text-muted-foreground">
        {t('synthesis.weapon_range.lede')}
      </p>

      <RangeTiles range={range} t={t} f={f} />

      <SectionCard
        title={t('synthesis.weapon_range.card_title')}
        label={t('synthesis.weapon_range.card_title')}
        titleAdornment={(label) => (
          <span className="flex flex-wrap items-baseline justify-between gap-2">
            <span>{label}</span>
            <span className="text-3xs font-normal tabular-nums text-muted-foreground">
              {t('synthesis.weapon_range.card_count', {
                weapons: lines.length,
                kills: range.measured_kills,
                deaths: range.measured_deaths,
              })}
            </span>
          </span>
        )}
        footer={<RangeFooter range={range} lines={lines} locale={locale} t={t} f={f} />}
      >
        <RangeCardBody
          publiable={publiable}
          series={series}
          buildRange={buildRange}
          buildElevation={buildElevation}
          height={height}
          t={t}
        />
      </SectionCard>
    </section>
  )
}
