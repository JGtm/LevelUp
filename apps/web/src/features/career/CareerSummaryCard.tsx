/**
 * CareerSummaryCard — carte de résumé rang + XP.
 * Jauges arc SVG (C1/C2 NATIVE_COMPONENTS.md).
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { RankProgressGauge } from '@/components/ui/rank-progress-gauge'
import type { CareerSummary, HeroProgress, CareerProjections } from '@/lib/api/types'

interface Props {
  summary: CareerSummary | null
  heroProgress: HeroProgress | null
  projections: CareerProjections | null
}
export function CareerSummaryCard({ summary, heroProgress, projections }: Props) {
  if (!summary) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-sm text-muted-foreground">
          Données de carrière non disponibles.
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>Rang de carrière</CardTitle>
          <Badge variant="default">{summary.rank_label}</Badge>
        </div>
      </CardHeader>
      <CardContent>
        {/* Jauges arc C1 + C2 côte à côte */}
        <div className="flex flex-wrap justify-around gap-4">
          {/* C1 — Progression rang XP actuel */}
          <RankProgressGauge
            title={summary.rank_name_raw ?? summary.rank_label}
            progressPct={summary.progress_pct / 100}
            subtitle={
              summary.is_max_rank
                ? 'Rang maximum atteint'
                : `${summary.xp_total.toLocaleString('fr-FR')} / ${summary.xp_for_next_rank.toLocaleString('fr-FR')} XP`
            }
            size={200}
          />

          {/* C2 — Progression Héros */}
          {heroProgress && (
            <RankProgressGauge
              title="Héros"
              progressPct={heroProgress.percentage / 100}
              subtitle={`Rang ${heroProgress.current_rank} · ${heroProgress.xp_remaining.toLocaleString('fr-FR')} XP restants`}
              size={200}
            />
          )}
        </div>

        {/* Projections */}
        {projections?.estimated_hero_date && (
          <p className="mt-3 text-center text-xs text-muted-foreground">
            Héros estimé le{' '}
            {new Date(projections.estimated_hero_date).toLocaleDateString('fr-FR')}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
