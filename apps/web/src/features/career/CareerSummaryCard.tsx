/**
 * CareerSummaryCard — carte de résumé rang + XP.
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import type { CareerSummary, HeroProgress, CareerProjections } from '@/lib/api/types'

interface Props {
  summary: CareerSummary | null
  heroProgress: HeroProgress | null
  projections: CareerProjections | null
}

function ProgressBar({ pct, color = 'bg-purple-500' }: { pct: number; color?: string }) {
  return (
    <div className="h-2 w-full overflow-hidden rounded-full bg-gray-100">
      <div
        className={`h-full rounded-full transition-all ${color}`}
        style={{ width: `${Math.min(100, Math.max(0, pct))}%` }}
      />
    </div>
  )
}

export function CareerSummaryCard({ summary, heroProgress, projections }: Props) {
  if (!summary) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-sm text-gray-400">
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
      <CardContent className="space-y-4">
        {/* Rang actuel */}
        <div>
          <div className="mb-1 flex items-center justify-between text-sm">
            <span className="font-medium text-gray-900">{summary.rank_name_raw}</span>
            <span className="text-gray-500">Rang #{summary.rank_number}</span>
          </div>
          <ProgressBar pct={summary.progress_pct} />
          <p className="mt-1 text-xs text-gray-400">
            {summary.xp_total.toLocaleString('fr-FR')} XP cumulés
            {!summary.is_max_rank && (
              <> · {summary.xp_for_next_rank.toLocaleString('fr-FR')} XP pour le prochain</>
            )}
          </p>
        </div>

        {/* Progression Héros */}
        {heroProgress && (
          <div>
            <div className="mb-1 flex items-center justify-between text-sm">
              <span className="font-medium text-gray-700">Progression Héros</span>
              <span className="text-gray-500">{heroProgress.percentage.toFixed(1)} %</span>
            </div>
            <ProgressBar pct={heroProgress.percentage} color="bg-amber-500" />
            <p className="mt-1 text-xs text-gray-400">
              Rang {heroProgress.current_rank} · {heroProgress.xp_remaining.toLocaleString('fr-FR')} XP restants
            </p>
          </div>
        )}

        {/* Projections */}
        {projections?.estimated_hero_date && (
          <p className="text-xs text-gray-500">
            Héros estim. le{' '}
            <span className="font-medium text-gray-700">
              {new Date(projections.estimated_hero_date).toLocaleDateString('fr-FR')}
            </span>
          </p>
        )}
      </CardContent>
    </Card>
  )
}
