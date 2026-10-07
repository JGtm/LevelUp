/**
 * WeaponRangeSection — LA PORTÉE DES ENGAGEMENTS SUR L'ONGLET USAGES.
 *
 * Elle a vécu sous `features/synthesis/` jusqu'au 2026-09-17, par héritage : la section a
 * quitté la page Synthèse pour l'onglet Résumé des Séries temporelles le 2026-09-13 et ses
 * fichiers n'avaient pas suivi. Un composant rangé sous le nom d'une page qui ne l'affiche
 * plus égare ses lecteurs — c'est arrivé. Le dossier dit maintenant qui la rend.
 *
 * Elle ouvre l'onglet « Usages » (cf. `TimeseriesPage.usages.tsx`), sous le titre de section
 * commun (`SectionTitle`).
 *
 * Transposition de la maquette validée par l'utilisateur le 2026-09-06
 * (`.ai/V7.5/MAQUETTE_PORTEE_ENGAGEMENTS_2026-09-06.html`) : quatre tuiles de tête, puis la carte
 * « Portée par arme » — deux bâtons p10→p90 par arme, frags au-dessus, morts en dessous. Elle reste
 * en DEMI-LARGEUR (maquette v4 des Usages, décision D11 du plan PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05),
 * et partage sa rangée avec la carte passée en `aside` (« Rôles de portée »).
 *
 * CE COMPOSANT NE CALCULE RIEN. La projection et l'option ECharts vivent dans
 * `@/components/charts/weaponRangeChart` (partagé), les décisions dans `weaponRange_logic.ts` — purs,
 * testés hors rendu.
 *
 * AUCUN DÉNOMINATEUR DANS LES TUILES (retour utilisateur du 2026-10-07) : « N frags mesurés sur
 * M » et « 1,5 s avant le frag · N frags mesurés » sont retirés. L'effectif de chaque arme reste
 * dans son infobulle et dans le tableau dépliable.
 */
import { useCallback, useMemo, type ReactNode } from 'react'

import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { ChartLegend, type ChartLegendItem } from '@/components/charts/ChartLegend'
import { SectionCard } from '@/components/ui/section-card'
import { resolveToken, tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import type { SynthesisWeaponRange } from '@/lib/api/types'

import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { useAppShellStore } from '@/stores/appShellStore'

import { AccentCard } from '@/components/ui/section-primitives'
import { SectionTitle } from '@/components/ui/detail-section'
import { titleWithHelp } from './titleWithHelp'
import {
  buildWeaponRangeOption,
  weaponRangeChartHeight,
  weaponRangeLines,
  type WeaponRangeLine,
} from '@/components/charts/weaponRangeChart'
import { WeaponRangeTable } from './WeaponRangeTable'
import { WEAPON_RANGE_MIN_MEASURED, hasWeaponRangeRows } from './weaponRange_logic'
import { useRangeFormats, type RangeFormats, type Translate } from './weaponRangeText'

/**
 * Les deux graphes vivent DANS la carte de section : leur propre `ChartCard` ne doit poser
 * aucun chrome, sinon on lit un cadre dans un cadre (retour utilisateur 2026-09-09 : « le
 * bloc du graphe contient un bloc qui contient le graphe »). La neutralisation par classes
 * locales est remplacée le 2026-09-21 par la prop `frameless` de `ChartCard` — le chrome
 * n'est plus posé puis annulé, il n'est pas posé.
 */

/**
 * Encres de la section — un seul endroit, partagé par les graphes et les deux légendes.
 *
 * FRAGS ET MORTS SUIVENT LA CONVENTION DE TOUTE L'APP : les jetons dédiés `stat-kills` /
 * `stat-deaths` (famille des stats de combat, 2026-09-17), qui remplacent les emprunts
 * `chart-series-1` / `outcome-loss` retenus le 2026-09-09.
 */
const KILLS_TOKEN: SemanticToken = 'stat-kills'
const DEATHS_TOKEN: SemanticToken = 'stat-deaths'
const MEDIAN_TOKEN: SemanticToken = 'perf-tier-2'
const DELTA_TOKEN: SemanticToken = 'chart-series-4'

/**
 * Encres du NUAGE DE DÉNIVELÉ : les MÊMES que la portée (D25).
 *
 * Le nuage a deux côtés, pas trois classes : mes frags et mes morts. Leur donner une rampe
 * de teinte propre (ce que faisaient les barres empilées par arme) aurait fait dire à la
 * couleur « d'en haut / d'en bas » là où la position se lit déjà sur l'axe des ordonnées.
 * Les deux côtés se lisent donc d'une carte à l'autre avec la même encre.
 */

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
        />
      )}
      {range.measured_deaths > 0 && (
        <AccentCard
          label={t('synthesis.weapon_range.tile_median_deaths')}
          value={f.distance(range.median_deaths_m)}
          accent={DEATHS_TOKEN}
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

// ─── Pied de carte : le tableau dépliable ─────────────────────────────────────

/**
 * RangeFooter — le tableau dépliable, et lui seul.
 *
 * Aucune note de couverture ni ligne « Sous le seuil de N mesures » (demandes utilisateur) :
 * l'effectif de chaque arme vit dans son infobulle et dans ce tableau.
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
        <WeaponRangeTable lines={lines} t={t} f={f} />
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
function useWeaponRangeOption(lines: WeaponRangeLine[], f: RangeFormats, t: Translate) {
  const buildRange = useCallback(() => {
    const tc = getEChartsThemeColors()
    return buildWeaponRangeOption({
      lines,
      tc,
      topColor: resolveToken(KILLS_TOKEN),
      bottomColor: resolveToken(DEATHS_TOKEN),
      medianColor: resolveToken(MEDIAN_TOKEN),
      cardColor: tc.card,
      fmtDistance: f.distance,
      labels: {
        // La Synthèse nomme les deux bandes « mes frags » et « mes morts ». Le libellé
        // `observed` n'est PAS fourni : min et max restent hors de son infobulle, dont
        // l'affichage est donc strictement inchangé par la généralisation du module.
        top: t('synthesis.weapon_range.side_kills'),
        bottom: t('synthesis.weapon_range.side_deaths'),
        percentiles: t('synthesis.weapon_range.percentiles'),
        noMeasure: t('synthesis.weapon_range.no_measure'),
      },
    })
  }, [lines, f, t])

  return buildRange
}

/**
 * RangeChartBody — le graphe de portée, OU la phrase qui explique pourquoi il n'y en a pas.
 *
 * Le second cas est nominal (toutes les armes sous le seuil de publication) : la section garde
 * ses tuiles et ses armes nommées, et le corps DIT pourquoi il est vide. Un graphe sans barre
 * ne se lit pas « rien à montrer », il se lit « bug ».
 */
function RangeChartBody({
  publiable,
  series,
  buildOption,
  height,
  legendItems,
  legendLabel,
  emptyMessage,
}: {
  publiable: boolean
  series: ChartSeries<WeaponRangeLine>[]
  buildOption: () => EChartsCoreOption
  height: number
  legendItems: ChartLegendItem[]
  legendLabel: string
  /**
   * La phrase d'état vide DE CETTE CARTE — chaque carte a la sienne (finitions
   * 2026-09-13) : le dénivelé affichait la phrase de la portée (« les portées mesurées
   * restent trop rares… »), qui ne parlait pas de son graphe.
   */
  emptyMessage: string
}) {
  if (!publiable) {
    return <p className="px-3 pb-1 pt-2.5 text-xs text-muted-foreground">{emptyMessage}</p>
  }
  // `fluid` + `flex-1` : la carte s'étire à la hauteur de sa voisine de rangée (« Rôles de
  // portée ») et le graphe la remplit — la légende reste posée au ras du bas du bloc.
  return (
    <ChartCard
      series={series}
      buildOption={buildOption}
      height={height}
      frameless
      fluid
      className="flex-1"
      legend={<ChartLegend items={legendItems} ariaLabel={legendLabel} />}
    />
  )
}

// ─── Section ──────────────────────────────────────────────────────────────────

export interface WeaponRangeSectionProps {
  range: SynthesisWeaponRange | null | undefined
  /**
   * La carte posée à DROITE de « Portée par arme », sur la même rangée (« Rôles de portée »
   * sur l'onglet Usages). Absente = la carte de portée reste seule en demi-largeur.
   */
  aside?: ReactNode
}

export function WeaponRangeSection({ range, aside }: WeaponRangeSectionProps) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = useCallback<Translate>(
    (key, vars) => formatMessage(synthesisManifest, key, locale, vars),
    [locale],
  )
  const f = useRangeFormats(locale)
  const weapons = range?.weapons
  const lines = useMemo(() => weaponRangeLines(weapons, locale), [weapons, locale])
  const series = useMemo(() => [{ key: 'weapon-range', datapoints: lines }], [lines])

  const buildRange = useWeaponRangeOption(lines, f, t)

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
      {/* LE TITRE DE SECTION STANDARD (2026-09-22) : la section portait un sous-titre
          uppercase `text-3xs` à filet (`SectionSubtitle`), qui se lisait comme une annexe
          à côté des autres titres de l'onglet Usages. Même texte, gabarit commun. */}
      <SectionTitle>{t('synthesis.weapon_range.heading')}</SectionTitle>

      <RangeTiles range={range} t={t} f={f} />

      {/* DEMI-LARGEUR (D11) : les étiquettes d'armes restent sur l'axe vertical et seul l'axe
          des mètres se resserre — le tableau dépliable garde la valeur exacte. La seconde
          colonne reçoit `aside` (« Rôles de portée »), à hauteur égale. Pleine largeur sous
          `lg`. */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <SectionCard
          title={t('synthesis.weapon_range.card_title')}
          label={t('synthesis.weapon_range.card_title')}
          titleAdornment={titleWithHelp(t('synthesis.weapon_range.range_subtitle_detail'))}
          footer={<RangeFooter lines={lines} t={t} f={f} />}
        >
          <RangeChartBody
            publiable={publiable}
            series={series}
            buildOption={buildRange}
            height={height}
            legendItems={rangeLegendItems(t)}
            legendLabel={t('synthesis.weapon_range.legend_label')}
            emptyMessage={t('synthesis.weapon_range.empty_below_threshold', {
              min: WEAPON_RANGE_MIN_MEASURED,
            })}
          />
        </SectionCard>
        {aside}
      </div>
    </section>
  )
}
