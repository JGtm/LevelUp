/**
 * CareerSummaryCard — carte de résumé rang + XP.
 * Jauges arc SVG (C1/C2 NATIVE_COMPONENTS.md) avec stats détaillées sous chaque
 * jauge : XP prochain rang, image+nom rang actuel/prochain, XP restante,
 * rang X/N (N = total de rangs du titre, porté par le payload — title-agnostic).
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { RankProgressGauge } from '@/components/ui/rank-progress-gauge'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale as toIntlLocale } from '@/lib/formatters'
import type { CareerSummary, HeroProgress, CareerProjections } from '@/lib/api/types'

interface Props {
  summary: CareerSummary | null
  heroProgress: HeroProgress | null
  projections: CareerProjections | null
}

export function CareerSummaryCard({ summary, heroProgress, projections }: Props) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const intlLocale = toIntlLocale(locale)

  if (!summary) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-sm text-muted-foreground">
          {careerManifest['career.empty.no_data'][locale]}
        </CardContent>
      </Card>
    )
  }

  const nextRankName =
    locale === 'fr' ? summary.next_rank_name_fr : summary.next_rank_name_en

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>{careerManifest['career.summary.rank_label'][locale]}</CardTitle>
          <Badge variant="default">{summary.rank_label}</Badge>
        </div>
      </CardHeader>
      <CardContent>
        <div className="flex flex-wrap justify-around gap-6">
          {/* C1 — Progression rang XP actuel + détails */}
          <div className="flex flex-col items-center gap-3">
            <RankProgressGauge
              title={summary.rank_name_raw ?? summary.rank_label}
              progressPct={summary.progress_pct / 100}
              subtitle={
                summary.is_max_rank
                  ? careerManifest['career.summary.max_rank_reached'][locale]
                  : `${summary.xp_total.toLocaleString(intlLocale)} / ${summary.xp_for_next_rank.toLocaleString(intlLocale)} XP`
              }
              size={200}
            />

            {!summary.is_max_rank && (
              <div className="text-center">
                <div className="text-xs text-muted-foreground">
                  {careerManifest['career.summary.xp_next_rank'][locale]}
                </div>
                <div className="text-sm font-semibold">
                  {summary.xp_for_next_rank.toLocaleString(intlLocale)}
                </div>
              </div>
            )}

            {(summary.rank_image_url || nextRankName || summary.next_rank_image_url) && (
              <div className="flex items-center gap-2 text-xs">
                {summary.rank_image_url && (
                  <img
                    src={summary.rank_image_url}
                    alt={summary.rank_label}
                    className="h-10 w-10 object-contain"
                    loading="lazy"
                    decoding="async"
                  />
                )}
                <span className="text-muted-foreground">{summary.rank_label}</span>
                {!summary.is_max_rank && (nextRankName || summary.next_rank_image_url) && (
                  <>
                    <span aria-hidden="true">→</span>
                    {summary.next_rank_image_url && (
                      <img
                        src={summary.next_rank_image_url}
                        alt={nextRankName ?? ''}
                        className="h-10 w-10 object-contain"
                        loading="lazy"
                        decoding="async"
                      />
                    )}
                    {nextRankName && (
                      <span className="text-muted-foreground">{nextRankName}</span>
                    )}
                  </>
                )}
              </div>
            )}
          </div>

          {/* C2 — Progression vers le rang max + détails */}
          {heroProgress && (
            <div className="flex flex-col items-center gap-3">
              <RankProgressGauge
                title={careerManifest['career.charts.hero_progress'][locale]}
                progressPct={heroProgress.percentage / 100}
                subtitle={`${(heroProgress.xp_total_required - heroProgress.xp_remaining).toLocaleString(intlLocale)} / ${heroProgress.xp_total_required.toLocaleString(intlLocale)} XP`}
                size={200}
              />

              <div className="grid grid-cols-2 gap-6 text-center">
                <div>
                  <div className="text-xs text-muted-foreground">
                    {careerManifest['career.summary.xp_remaining'][locale]}
                  </div>
                  <div className="text-sm font-semibold">
                    {heroProgress.xp_remaining.toLocaleString(intlLocale)}
                  </div>
                </div>
                <div>
                  <div className="text-xs text-muted-foreground">
                    {careerManifest['career.summary.rank_position'][locale]}
                  </div>
                  <div className="text-sm font-semibold">
                    {heroProgress.current_rank}/{heroProgress.total_ranks}
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {projections?.estimated_hero_date && (
          <p className="mt-4 text-center text-xs text-muted-foreground">
            {careerManifest['career.summary.estimated_hero'][locale]}{' '}
            {new Date(projections.estimated_hero_date).toLocaleDateString(intlLocale)}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
