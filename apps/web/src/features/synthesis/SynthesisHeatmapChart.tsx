/**
 * SynthesisHeatmapChart — synthesis.03.
 * Heatmap 2D heure × jour (X=heure, Y=jour) colorée par win_rate via la rampe
 * DIVERGENTE centralisée (perdant → neutre 50 % → gagnant) : le win_rate est un
 * indicateur signé autour de 0,5, et la rampe divergente est CVD-safe par
 * construction — neutre gris en palette daltonienne (cf. heatmapColors).
 * Toutes les 168 cellules sont émises — `value: null` pour les cases vides.
 *
 * Passe par le wrapper canonique `Heatmap2DChart` (lot C2, décision D1 du plan
 * vague C formes — « il existe déjà un composant canonique, on l'étend, on
 * n'en crée pas un second »). `valueRange={[0, 1]}` FIGE l'échelle : le
 * neutre à 50 % doit rester au CENTRE de la rampe quelle que soit la plage
 * réelle des taux de victoire mesurés sur la période — laisser le wrapper
 * auto-ajuster min/max (comportement par défaut des 4 autres consommateurs)
 * décentrerait le neutre.
 *
 * FORME RESTAURÉE LE 2026-09-13 (demande utilisateur) : Lundi en haut, titres
 * sur les deux axes, et échelle de couleur VERTICALE à droite graduée en
 * pourcentage — le rendu d'avant le passage au wrapper. Les trois options
 * manquantes ont été AJOUTÉES au wrapper (`yAxisInverse`, `axisNames`,
 * `visualMapOrient`/`Formatter`/`Text`) plutôt que de rouvrir un builder local :
 * elles sont toutes optionnelles, les quatre autres consommateurs du wrapper
 * gardent leur rendu à l'octet près.
 *
 * Les points sont donc émis Lundi → Dimanche, dans le sens de la semaine, et
 * c'est `yAxisInverse` qui met Lundi en haut : l'ordre des DONNÉES ne porte plus
 * une décision d'AFFICHAGE.
 *
 * `emptyCells="hidden"` est le QUATRIÈME trait restauré, et le seul qui déroge à une
 * doctrine du dépôt (D3, « l'absence a sa propre forme »). Sur ce calendrier les cases
 * sans mesure sont MAJORITAIRES et RÉGULIÈRES — les heures de nuit, toutes les semaines :
 * hachurées, elles formaient un damier plus voyant que les mesures, et le graphe montrait
 * surtout quand le joueur ne joue PAS. L'heure vide reste lisible sans forme propre (une
 * colonne nue sous une graduation horaire ne se confond avec rien) ; la dérogation vaut
 * pour ce seul consommateur, le défaut du wrapper ne bouge pas.
 */
import { useCallback, useMemo } from 'react'
import { Heatmap2DChart, type ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { escapeHtml } from '@/components/charts/_utils'
import { dowLabels, HOUR_LABELS, calendarChartText } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { HeatmapCell } from '@/lib/api/types'

interface Props {
  cells: HeatmapCell[]
  title?: string
  height?: number
}

/** Construit les 168 points (24 h × 7 j), Lundi → Dimanche (l'axe Y inversé du
 *  wrapper place Lundi en haut — cf. doc de tête). */
function buildPoints(cells: HeatmapCell[], dowLabelsList: readonly string[]): ChartPointHeatmap[] {
  const lookup = new Map<string, { win_rate: number; count: number }>()
  for (const c of cells) {
    if (c.count > 0) {
      lookup.set(`${c.dow}-${c.hour}`, { win_rate: c.win_rate ?? 0, count: c.count })
    }
  }

  const points: ChartPointHeatmap[] = []
  for (let d = 0; d < 7; d++) {
    for (let h = 0; h < 24; h++) {
      const cell = lookup.get(`${d}-${h}`)
      points.push({
        x: HOUR_LABELS[h],
        y: dowLabelsList[d],
        value: cell ? cell.win_rate : null,
        detail: { count: cell ? cell.count : 0 },
      })
    }
  }
  return points
}

export function SynthesisHeatmapChart({ cells, title, height }: Props) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const txt = calendarChartText(locale)
  const dowLabelsList = dowLabels(locale)

  const series: ChartSeries<ChartPointHeatmap>[] = useMemo(
    () => (cells.length > 0 ? [{ key: 'heatmap', datapoints: buildPoints(cells, dowLabelsList) }] : []),
    [cells, dowLabelsList],
  )

  // Une graduation de l'échelle : un taux 0..1 rendu en pourcentage entier.
  const formatVisualMap = useCallback((value: number) => `${(value * 100).toFixed(0)}%`, [])

  const formatTooltip = useCallback(
    (point: ChartPointHeatmap) => {
      const count = (point.detail?.count as number | undefined) ?? 0
      const wrStr = `${((point.value ?? 0) * 100).toFixed(1)}%`
      return `${escapeHtml(point.y)} ${escapeHtml(point.x)}<br/>${txt.winRate} : ${wrStr}<br/>${txt.matches} : ${count}`
    },
    [txt],
  )

  return (
    <Heatmap2DChart
      title={title}
      series={series}
      height={height ?? 300}
      paletteMode="divergent"
      valueRange={[0, 1]}
      formatTooltip={formatTooltip}
      yAxisInverse
      axisNames={{ x: txt.hourAxis, y: txt.dayAxis }}
      visualMapOrient="vertical"
      visualMapFormatter={formatVisualMap}
      visualMapText={[txt.wins, '']}
      emptyCells="hidden"
    />
  )
}
