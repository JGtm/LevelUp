/**
 * ExplorerTargetSeasonMatches — graphe « matchs par saison » du joueur cible.
 *
 * Barres horizontales flat (hard-edge, sans ECharts) : une ligne par saison
 * avec libellé court (S1…S13), barre proportionnelle au max, et compteur.
 * Données : ExplorerTargetProfile.matches_per_season (joueur local = historique
 * complet ; adversaire = matchs observés en commun).
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { tokenCssVar } from '@/lib/accessibility'
import type { SeasonMatchCount } from '@/lib/api/types'

interface ExplorerTargetSeasonMatchesProps {
  seasons: SeasonMatchCount[]
  title: string
}

export function ExplorerTargetSeasonMatches({ seasons, title }: ExplorerTargetSeasonMatchesProps) {
  if (seasons.length === 0) return null
  const max = Math.max(...seasons.map((s) => s.matches), 1)

  return (
    <Card data-testid="explorer-target-season-matches">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-semibold">{title}</CardTitle>
      </CardHeader>
      <CardContent className="pt-0">
        <ul className="flex flex-col gap-1">
          {seasons.map((s) => (
            <li key={s.season_id} className="flex items-center gap-2">
              <span className="w-10 flex-shrink-0 text-right text-xs font-medium text-muted-foreground">
                {s.season_name}
              </span>
              <span className="relative h-4 flex-1 overflow-hidden rounded-sm bg-muted/30">
                <span
                  className="absolute inset-y-0 left-0 rounded-sm"
                  style={{
                    width: `${Math.max((s.matches / max) * 100, 2)}%`,
                    backgroundColor: tokenCssVar('chart-series-1'),
                  }}
                />
              </span>
              <span className="w-10 flex-shrink-0 text-right text-xs font-semibold tabular-nums text-foreground">
                {s.matches}
              </span>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  )
}
