/**
 * CompareWeaponsRange — LE BLOC « PORTÉE PAR RÔLE » du profil d'armes du Face-à-face.
 *
 * Deux graphes par comparaison — « Où ils fraguent », « Où ils meurent » — et chacun superpose
 * les DEUX JOUEURS sur la même bande : `top` = A, `bottom` = B. Empiler « A frague », « A
 * meurt », « B frague », « B meurt » sur une seule bande donnerait quatre bâtons illisibles.
 *
 * Fichier séparé de `CompareWeaponsSection.tsx` : la frontière suit le bloc, et les deux
 * fichiers restent loin du plafond de 500 lignes du dépôt.
 *
 * LA GRAMMAIRE GRAPHIQUE EST CELLE DE LA SYNTHÈSE (bâton p10 → p90, losange sur la médiane),
 * réutilisée telle quelle : `buildWeaponRangeOption` a été rendu title-agnostic pour cela. En
 * écrire une seconde donnerait deux lectures de la même mesure, divergeant au premier réglage.
 */
import { useCallback, useMemo } from 'react'

import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard } from '@/components/charts/ChartCard'
import { ChartLegend, type ChartLegendItem } from '@/components/charts/ChartLegend'
import { SectionCard } from '@/components/ui/section-card'
import { resolveToken, tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { CompareWeaponSide } from '@/lib/api/types'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import type { Locale } from '@/lib/i18n/locale'
import {
  buildWeaponRangeOption,
  weaponRangeChartHeight,
  type WeaponRangeLine,
} from '@/features/synthesis/_weaponRangeChart'

import {
  belowThresholdText,
  coverageOf,
  roleRangeLines,
  type RangeSideKey,
} from './compareWeapons_logic'
import type { CompareText } from './i18n'
import { TOKEN_A, TOKEN_MEDIAN, useRoleName, useWeaponFormats, type SideNames } from './compareWeaponsShared'

/**
 * useRangeOption — le constructeur d'option ECharts d'UN graphe.
 *
 * LES TOKENS NE SONT RÉSOLUS QU'ICI, À L'APPEL : ECharts peint dans un canvas, qui n'accepte
 * pas de variable CSS. `ChartCard` rappelle ce builder à chaque bascule de thème ou de palette
 * d'accessibilité, et les couleurs suivent — une valeur résolue une fois au montage resterait
 * périmée.
 */
function useRangeOption({
  lines,
  colorTopToken,
  colorBottomToken,
  names,
  text,
  fmtDistance,
}: {
  lines: WeaponRangeLine[]
  colorTopToken: SemanticToken
  colorBottomToken: SemanticToken
  names: SideNames
  text: CompareText
  fmtDistance: (m: number) => string
}) {
  return useCallback((): EChartsCoreOption => {
    const tc = getEChartsThemeColors()
    return buildWeaponRangeOption({
      lines,
      tc,
      topColor: resolveToken(colorTopToken),
      bottomColor: resolveToken(colorBottomToken),
      medianColor: resolveToken(TOKEN_MEDIAN),
      cardColor: tc.card,
      fmtDistance,
      labels: {
        // Les deux bandes sont deux JOUEURS, pas deux côtés de mesure : le graphe entier ne
        // montre qu'un côté, nommé par son titre de carte.
        top: names.a,
        bottom: names.b,
        percentiles: text.weaponsPercentiles,
        noMeasure: text.weaponsNoMeasure,
        // Min et max observés dans l'infobulle SEULEMENT (D5) — jamais tracés.
        observed: text.weaponsObserved,
      },
    })
  }, [lines, colorTopToken, colorBottomToken, names, text, fmtDistance])
}

/**
 * RangeChart — UN graphe : un côté de mesure, deux joueurs superposés.
 *
 * Sous le graphe, les DEUX couvertures (« N frags mesurés sur M ») et les rôles écartés par le
 * seuil, NOMMÉS par joueur (D9) : un seuil qui cache en silence ferait croire que le rôle n'a
 * jamais servi.
 */
function RangeChart({
  title,
  lines,
  sideA,
  sideB,
  which,
  names,
  colorTopToken,
  colorBottomToken,
  text,
  locale,
}: {
  title: string
  lines: WeaponRangeLine[]
  sideA: CompareWeaponSide | null | undefined
  sideB: CompareWeaponSide | null | undefined
  which: RangeSideKey
  names: SideNames
  colorTopToken: SemanticToken
  colorBottomToken: SemanticToken
  text: CompareText
  locale: Locale
}) {
  const f = useWeaponFormats(locale)
  const roleName = useRoleName(locale)
  const buildOption = useRangeOption({
    lines,
    colorTopToken,
    colorBottomToken,
    names,
    text,
    fmtDistance: f.distance,
  })

  const series = useMemo(() => [{ key: `range-${which}`, datapoints: lines }], [which, lines])
  const legend: ChartLegendItem[] = [
    { label: names.a, color: tokenCssVar(colorTopToken) },
    { label: names.b, color: tokenCssVar(colorBottomToken) },
  ]

  const couvA = coverageOf(sideA, which)
  const couvB = coverageOf(sideB, which)
  const seuilA = belowThresholdText(sideA, which, roleName, f.count)
  const seuilB = belowThresholdText(sideB, which, roleName, f.count)

  return (
    <SectionCard title={title} label={title}>
      {lines.length === 0 ? (
        <p className="px-3 pb-2 pt-2.5 text-xs text-muted-foreground">{text.weaponsNoRange}</p>
      ) : (
        <ChartCard
          series={series}
          buildOption={buildOption}
          height={weaponRangeChartHeight(lines.length)}
          className="rounded-none border-none bg-transparent shadow-none"
          legend={<ChartLegend items={legend} ariaLabel={title} />}
        />
      )}
      <div className="space-y-0.5 px-3 pb-2 text-2xs text-muted-foreground">
        <CoverageNote name={names.a} coverage={couvA} below={seuilA} text={text} />
        <CoverageNote name={names.b} coverage={couvB} below={seuilB} text={text} />
      </div>
    </SectionCard>
  )
}

/**
 * CoverageNote — « N frags mesurés sur M », et les rôles écartés par le seuil.
 *
 * LES DEUX SE LISENT ENSEMBLE : la couverture dit ce que le décodeur a su placer, la liste dit
 * ce que le seuil de publication a retiré. Publier l'une sans l'autre laisserait croire que
 * l'écart entre le numérateur et le dénominateur est la seule réserve.
 */
function CoverageNote({
  name,
  coverage,
  below,
  text,
}: {
  name: string
  coverage: { measured: number; total: number } | null
  below: string | null
  text: CompareText
}) {
  if (!coverage) return null
  return (
    <p>
      {name} — {text.weaponsCoverage(coverage.measured, coverage.total)}
      {below ? ` · ${text.weaponsBelowThreshold(below)}` : ''}
    </p>
  )
}

/** CompareWeaponsRange — les deux graphes (frags, morts) d'UNE comparaison. */
export function CompareWeaponsRange({
  sideA,
  sideB,
  names,
  colorBottomToken,
  text,
  locale,
}: {
  sideA: CompareWeaponSide | null | undefined
  sideB: CompareWeaponSide | null | undefined
  names: SideNames
  colorBottomToken: SemanticToken
  text: CompareText
  locale: Locale
}) {
  const roleName = useRoleName(locale)
  const lignesFrags = useMemo(
    () => roleRangeLines(sideA, sideB, 'kills', roleName),
    [sideA, sideB, roleName],
  )
  const lignesMorts = useMemo(
    () => roleRangeLines(sideA, sideB, 'deaths', roleName),
    [sideA, sideB, roleName],
  )
  if (!sideA?.range && !sideB?.range) return null

  const commun = { sideA, sideB, names, colorTopToken: TOKEN_A, colorBottomToken, text, locale }
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <RangeChart
        {...commun}
        title={text.weaponsRangeKills}
        lines={lignesFrags}
        which="kills"
      />
      <RangeChart
        {...commun}
        title={text.weaponsRangeDeaths}
        lines={lignesMorts}
        which="deaths"
      />
    </div>
  )
}
