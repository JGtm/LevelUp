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
import { useCallback, useMemo } from 'react'

import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { ChartLegend, type ChartLegendItem } from '@/components/charts/ChartLegend'
import { InfoTooltip } from '@/components/ui/info-tooltip'
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
import { WEAPON_RANGE_MIN_MEASURED, hasWeaponRangeRows } from './weaponRange_logic'
import { useRangeFormats, type RangeFormats, type Translate } from './weaponRangeText'

/**
 * Les deux graphes vivent DANS la carte de section : leur propre `ChartCard` ne doit poser
 * aucun chrome, sinon on lit un cadre dans un cadre (retour utilisateur 2026-09-09 : « le
 * bloc du graphe contient un bloc qui contient le graphe »). `border-none` et non `border-0`
 * — la première règle porte sur le STYLE de bordure et gagne quel que soit l'ordre
 * d'émission des utilitaires de largeur.
 */
const NESTED_CHART_CHROME = 'rounded-none border-none bg-transparent shadow-none'

/**
 * Encres de la section — un seul endroit, partagé par les graphes et les deux légendes.
 *
 * FRAGS ET MORTS SUIVENT LA CONVENTION DE TOUTE L'APP (retour utilisateur 2026-09-09) :
 * `chart-series-1` pour ce qu'on fait, `outcome-loss` pour ce qu'on subit — les mêmes deux
 * encres que « Évolution Frags / Morts », la cadence de l'Explorateur ou le profil de combat.
 * Le côté « morts » empruntait jusqu'ici `chart-series-3`, qui est l'encre des ASSISTANCES
 * ailleurs : deux bleus voisins pour deux faits opposés.
 */
const KILLS_TOKEN: SemanticToken = 'chart-series-1'
const DEATHS_TOKEN: SemanticToken = 'outcome-loss'
const MEDIAN_TOKEN: SemanticToken = 'perf-tier-2'
const DELTA_TOKEN: SemanticToken = 'chart-series-4'

/**
 * Encres du DÉNIVELÉ — indépendantes de celles des frags/morts, et c'est voulu.
 *
 * Le dénivelé ne dit pas qui tue qui, il dit d'OÙ : une rampe d'une seule teinte, du clair
 * (d'en bas) au foncé (d'en haut), plus le gris des libellés d'axe pour « à niveau ». Les
 * accrocher à `KILLS_TOKEN`/`DEATHS_TOKEN` ferait dire à la couleur ce qu'elle ne dit pas
 * (le rouge des morts sur un segment « d'en haut » se lirait comme un jugement).
 */
const ELEVATION_ABOVE_TOKEN: SemanticToken = 'chart-series-3'
const ELEVATION_BELOW_TOKEN: SemanticToken = 'chart-series-1'

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

// ─── Légendes et sous-titres ──────────────────────────────────────────────────

/**
 * Les deux légendes passent par `<ChartLegend>`, le composant commun à tous les graphes de
 * l'app (retour utilisateur 2026-09-09 : « ça ne suit pas la nomenclature de tous les autres
 * graphes »). Elles sont posées EN PIED DE GRAPHE, centrées, via la prop `legend` de
 * `ChartCard` — plus en tête de bloc à droite du sous-titre.
 *
 * Les mentions de position (« bâton du haut », « bâton du bas, l'arme est celle du tueur »)
 * sont retirées : elles doublaient l'ordre déjà lisible sur le graphe et rendaient la ligne
 * de légende deux fois plus longue que la légende elle-même.
 */
function rangeLegendItems(t: Translate): ChartLegendItem[] {
  return [
    { label: t('synthesis.weapon_range.side_kills'), color: tokenCssVar(KILLS_TOKEN) },
    { label: t('synthesis.weapon_range.side_deaths'), color: tokenCssVar(DEATHS_TOKEN) },
  ]
}

/**
 * `à niveau` emprunte le gris des libellés d'axe (`--muted-foreground`), la MÊME encre que
 * son segment dans le graphe : c'est la seule des trois classes qui n'a pas de token
 * d'accessibilité, d'où la variable CSS brute plutôt qu'un `tokenCssVar`.
 */
function elevationLegendItems(t: Translate): ChartLegendItem[] {
  const labels = elevationLabels(t)
  const color: Record<ElevationKey, string> = {
    above: tokenCssVar(ELEVATION_ABOVE_TOKEN),
    level: 'var(--muted-foreground)',
    below: tokenCssVar(ELEVATION_BELOW_TOKEN),
  }
  return ELEVATION_KEYS.map((key) => ({ key, label: labels[key], color: color[key] }))
}

/**
 * SubtitleRow — le titre d'un des deux blocs, et son mode d'emploi dans une aide ⓘ.
 *
 * Le détail de lecture (« bâton du 10e au 90e centile, losange sur la médiane ») était écrit
 * en clair à côté du titre : trois lignes de texte gris au-dessus de chaque graphe, lues une
 * fois puis jamais. Il vit désormais derrière l'icône, comme partout ailleurs dans l'app.
 */
function SubtitleRow({ title, help }: { title: string; help: string }) {
  return (
    <p className="flex items-center gap-1.5 px-3 pt-2.5 text-xs font-semibold text-foreground">
      {title}
      <InfoTooltip content={help} iconClass="w-3.5 h-3.5" />
    </p>
  )
}

// ─── Pied de carte : le tableau dépliable ─────────────────────────────────────

/**
 * RangeFooter — le tableau dépliable, et lui seul.
 *
 * La ligne « Sous le seuil de N mesures — … » et la note de couverture ont été retirées le
 * 2026-09-09 (demande utilisateur) : deux paragraphes de texte gris sous chaque carte, qui
 * répétaient une réserve déjà portée par les dénominateurs de chaque tuile (« 1 214 frags
 * mesurés sur 1 602 ») et par l'infobulle de chaque arme.
 */
function RangeFooter({
  lines,
  t,
  f,
}: {
  lines: WeaponRangeLine[]
  t: Translate
  f: RangeFormats
}) {
  if (lines.length === 0) return null
  return (
    <details className="border-t border-border pb-2">
      <summary className="cursor-pointer px-3 py-1 text-xs text-muted-foreground">
        {t('synthesis.weapon_range.table_summary')}
      </summary>
      <div className="overflow-x-auto px-3 pb-1">
        <SynthesisWeaponRangeTable lines={lines} t={t} f={f} />
      </div>
    </details>
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
        above: resolveToken(ELEVATION_ABOVE_TOKEN),
        level: cssColorToHex(tc.axisLabel),
        below: resolveToken(ELEVATION_BELOW_TOKEN),
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
        help={t('synthesis.weapon_range.range_subtitle_detail')}
      />
      <ChartCard
        series={series}
        buildOption={buildRange}
        height={height}
        className={NESTED_CHART_CHROME}
        legend={
          <ChartLegend
            items={rangeLegendItems(t)}
            ariaLabel={t('synthesis.weapon_range.legend_label')}
          />
        }
      />

      <SubtitleRow
        title={t('synthesis.weapon_range.elevation_subtitle')}
        help={t('synthesis.weapon_range.elevation_subtitle_detail')}
      />
      <ChartCard
        series={series}
        buildOption={buildElevation}
        height={height}
        className={NESTED_CHART_CHROME}
        legend={
          <ChartLegend
            items={elevationLegendItems(t)}
            ariaLabel={t('synthesis.weapon_range.legend_elevation_label')}
          />
        }
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
        footer={<RangeFooter lines={lines} t={t} f={f} />}
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
