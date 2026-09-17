/**
 * ExplorerTargetFragRange — bloc « Portée des frags » de la section « matchs joués ensemble »
 * de l'encart adversaire (3e rangée, colonne de droite).
 *
 * MÊME FIGURE que « Où ils fraguent » du Face-à-face : une ligne par RÔLE d'arme, bâton du
 * 10e au 90e centile, losange sur la médiane, et les deux joueurs superposés sur la même
 * bande — toi en haut, la cible en bas. Même module de rendu
 * (`components/charts/weaponRangeChart`), même axe et même ordre
 * (`components/charts/weaponRangeRoles`) : en écrire une seconde grammaire donnerait deux
 * lectures de la même mesure, qui divergeraient au premier réglage.
 *
 * # CE QUE LE BLOC NE MONTRE PAS (demande utilisateur du 2026-09-17)
 *
 * Les bandes, et rien d'autre. Le Face-à-face affiche sous son graphe la couverture
 * (« N frags mesurés sur M ») et les rôles écartés par le seuil ; ici la colonne est étroite
 * et le bloc voisine deux autres lectures — ces notes y feraient du bruit sans changer ce que
 * le lecteur comprend.
 *
 * # UN SEUL CÔTÉ DE MESURE
 *
 * Les frags. Le côté morts (« Où ils meurent ») est une seconde question, et la rangée n'a
 * qu'une colonne pour celle-ci.
 *
 * # DÉGRADATION
 *
 * Le backend n'envoie un bloc que pour un joueur dont des frags sont mesurés sur ces matchs
 * (titre avec positions par kill, films décodés). Les deux absents → état vide titré, jamais
 * une portée fabriquée. Un seul présent → sa bande seule, l'autre ligne dit « aucune mesure ».
 */
import { useCallback, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard } from '@/components/charts/ChartCard'
import { ChartLegend, type ChartLegendItem } from '@/components/charts/ChartLegend'
import {
  buildWeaponRangeOption,
  weaponRangeChartHeight,
} from '@/components/charts/weaponRangeChart'
import { roleAxis, roleLabel, roleRangeLines } from '@/components/charts/weaponRangeRoles'
import { resolveToken, tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import { intlLocale } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ExplorerEncounterStats } from '@/lib/api/types'

/**
 * Les deux bandes sont DEUX JOUEURS comparés : les jetons de la famille « comparaison » les
 * nomment déjà, sur les quatre palettes et avec leurs garde-fous de contraste. En inventer
 * une paire propre à l'Explorer ajouterait une famille de couleurs pour dire la même chose.
 */
const TOKEN_SELF: SemanticToken = 'compare-a'
const TOKEN_TARGET: SemanticToken = 'compare-b'
/** Le losange de médiane, comme partout ailleurs sur cette grammaire. */
const TOKEN_MEDIAN: SemanticToken = 'perf-tier-2'

interface Props {
  /** Stats de rencontre : y vivent les deux blocs de portée (toi, la cible). */
  encounterStats?: ExplorerEncounterStats | null
  /** Gamertag de la cible — nomme la bande du bas. */
  gamertag: string
}

export function ExplorerTargetFragRange({ encounterStats, gamertag }: Props) {
  const appLocale = useAppShellStore((s) => s.locale)
  const t = (key: ExplorerManifestKey) => formatMessage(explorerManifest, key, appLocale)
  const self = encounterStats?.frag_range_self ?? null
  const target = encounterStats?.frag_range_target ?? null

  const roleName = useCallback(
    (key: string) => roleLabel(key, (manifestKey) => formatMessage(fragsManifest, manifestKey as never, appLocale)),
    [appLocale],
  )
  // L'axe porte l'union des rôles des deux joueurs : une ligne qu'un seul a mesurée reste
  // visible chez l'autre, vide — « il ne l'a jamais fait » est une information.
  const axis = useMemo(() => roleAxis([self, target], roleName), [self, target, roleName])
  const lines = useMemo(() => roleRangeLines(axis, self, target, 'kills'), [axis, self, target])

  const selfLabel = t('explorer.target_profile.frag_range_self')
  const distance = useMemo(() => {
    const nf = new Intl.NumberFormat(intlLocale(appLocale), {
      minimumFractionDigits: 1,
      maximumFractionDigits: 1,
    })
    return (m: number) => `${nf.format(m)} m`
  }, [appLocale])

  // Les jetons ne sont résolus QU'ICI, à l'appel : ECharts peint dans un canvas, qui n'accepte
  // pas de variable CSS. ChartCard rappelle ce builder à chaque bascule de thème ou de palette,
  // et les couleurs suivent — une valeur résolue une fois au montage resterait périmée.
  const buildOption = useCallback((): EChartsCoreOption => {
    const tc = getEChartsThemeColors()
    return buildWeaponRangeOption({
      lines,
      tc,
      topColor: resolveToken(TOKEN_SELF),
      bottomColor: resolveToken(TOKEN_TARGET),
      medianColor: resolveToken(TOKEN_MEDIAN),
      cardColor: tc.card,
      fmtDistance: distance,
      labels: {
        // Les deux bandes sont deux JOUEURS, pas deux côtés de mesure : le graphe entier ne
        // montre que les frags, ce que dit son titre.
        top: selfLabel,
        bottom: gamertag,
        percentiles: t('explorer.target_profile.frag_range_percentiles'),
        noMeasure: t('explorer.target_profile.frag_range_no_measure'),
        observed: t('explorer.target_profile.frag_range_observed'),
      },
    })
  }, [lines, distance, selfLabel, gamertag, appLocale]) // eslint-disable-line react-hooks/exhaustive-deps

  const series = useMemo(() => [{ key: 'frag-range', datapoints: lines }], [lines])
  const legend: ChartLegendItem[] = [
    { label: selfLabel, color: tokenCssVar(TOKEN_SELF) },
    { label: gamertag, color: tokenCssVar(TOKEN_TARGET) },
  ]

  return (
    <div
      className="flex h-full flex-col overflow-hidden rounded-lg border border-border bg-card"
      data-testid="explorer-target-frag-range"
    >
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
        {t('explorer.target_profile.frag_range_title')}
      </div>
      {lines.length === 0 ? (
        <div className="flex flex-1 items-center justify-center p-3">
          <span className="rounded-md border border-dashed border-border px-3 py-2 text-xs text-muted-foreground">
            {t('explorer.target_profile.frag_range_empty')}
          </span>
        </div>
      ) : (
        <div className="p-3">
          <ChartCard
            series={series}
            buildOption={buildOption}
            height={weaponRangeChartHeight(lines.length)}
            className="rounded-none border-none bg-transparent shadow-none"
            legend={<ChartLegend items={legend} ariaLabel={t('explorer.target_profile.frag_range_title')} />}
          />
        </div>
      )}
    </div>
  )
}
