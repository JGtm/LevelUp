import { useCallback, useState } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { buildCompositeProgressEdgeLabels, clampCompositeProgress } from '@/components/ui/composite-progress-bar'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SeasonPassPageResponse, SeasonPassTrackSummary } from '@/lib/api/types'
import { BattlePassRewardLightbox, type RewardLightboxData } from '@/features/palmares/BattlePassRewardLightbox'
import { BattlePassRewardCarousel, type RewardCard } from '@/features/palmares/BattlePassRewardCarousel'

function pickFeaturedPass(passes: SeasonPassTrackSummary[]) {
  return passes.find((pass) => pass.is_active)
    ?? passes.find((pass) => pass.status === 'in_progress')
    ?? passes[0]
    ?? null
}

export function HomeBattlePassPanel({
  loading,
  data,
  errorHint,
}: {
  loading: boolean
  data?: SeasonPassPageResponse
  errorHint?: string | null
}) {
  const featuredPass = pickFeaturedPass(data?.passes ?? [])
  const [activeReward, setActiveReward] = useState<RewardLightboxData | null>(null)

  const handleOpenCard = useCallback((card: RewardCard) => {
    const badges: RewardLightboxData['badges'] = []
    if (card.is_current) badges.push({ label: 'Palier actuel', tone: 'current' })
    if (card.is_obtained) badges.push({ label: 'Obtenu', tone: 'obtained' })
    if (card.is_free) badges.push({ label: 'Gratuit', tone: 'free' })
    else badges.push({ label: 'Premium', tone: 'premium' })
    setActiveReward({
      title: card.title,
      rank: card.rank,
      imageUrl: card.image_url ?? null,
      description: card.description ?? null,
      quality: card.quality ?? null,
      itemType: card.item_type ?? null,
      badges,
    })
  }, [])

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Pass de combat</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">Chargement du pass de combat...</p>
        </CardContent>
      </Card>
    )
  }

  if (!data?.available) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Pass de combat</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            Non disponible ({errorHint ?? data?.error_hint ?? 'live API non configurée'})
          </p>
        </CardContent>
      </Card>
    )
  }

  if (!featuredPass) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Pass de combat</CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyStateNotice
            title="Aucun pass détecté"
            description="Aucune progression de pass saisonnier n'a été renvoyée pour ce joueur."
          />
        </CardContent>
      </Card>
    )
  }

  const tierProgress = clampCompositeProgress(featuredPass.active_tier_progress_percent)
  const tierProgressLabels = buildCompositeProgressEdgeLabels({
    partialProgress: featuredPass.partial_progress,
    xpPerRank: featuredPass.xp_per_rank,
    progressPercent: tierProgress,
    locale: 'fr-FR',
  })
  const hasTiers = (featuredPass.tiers?.length ?? 0) > 0

  return (
    <Card className="relative flex min-h-[14rem] flex-col overflow-hidden border-border/70 bg-card/95 shadow-sm">
      <div className="absolute inset-0 bg-gradient-to-br from-background via-background/96 to-background/85" aria-hidden="true" />

      <CardHeader className="relative space-y-4 pb-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="text-base">Pass de combat</CardTitle>
            <h3 className="mt-3 text-xl font-semibold tracking-tight text-foreground sm:text-2xl">
              {featuredPass.name}
            </h3>
          </div>

          <div className="flex flex-wrap gap-2">
            {featuredPass.is_owned && <Badge variant="outline">Premium</Badge>}
            {featuredPass.is_active && <Badge variant="default">Actif</Badge>}
          </div>
        </div>

        <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
          {featuredPass.description ?? 'Aucune description disponible pour ce pass.'}
        </p>
      </CardHeader>

      <CardContent className="relative space-y-6">
        <div className="overflow-hidden rounded-xl border border-border bg-card shadow-[0_24px_72px_-44px_rgba(15,23,42,0.92)]">
          {(featuredPass.background_image_url ?? featuredPass.image_url) ? (
            <img
              src={featuredPass.background_image_url ?? featuredPass.image_url!}
              alt={`Illustration de ${featuredPass.name}`}
              data-testid="home-battle-pass-image"
              className="aspect-[986/248] w-full object-cover"
            />
          ) : (
            <div className="flex h-44 w-full items-center justify-center bg-muted px-6 text-center text-foreground sm:h-52 xl:h-60">
              <div>
                <p className="text-xs uppercase tracking-[0.34em] text-muted-foreground">Pass actif</p>
                <p className="mt-3 text-2xl font-semibold sm:text-3xl">{featuredPass.name}</p>
              </div>
            </div>
          )}
        </div>

        {hasTiers ? (
          <div className="space-y-5">
            <BattlePassRewardCarousel
              tiers={featuredPass.tiers!}
              activeTierRank={featuredPass.active_tier_rank}
              onOpenCard={handleOpenCard}
            />

            <div className="flex justify-center">
              <div
                data-testid="home-battle-pass-active-tier-progress-row"
                className="grid w-2/3 grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2 text-[11px] text-muted-foreground"
              >
                <span data-testid="home-battle-pass-active-tier-progress-current" className="shrink-0 whitespace-nowrap">
                  {tierProgressLabels.current}
                </span>
                <div data-testid="home-battle-pass-active-tier-progress-track" className="min-w-0 overflow-hidden rounded-full bg-muted-foreground/25">
                  <div className="h-2 w-full">
                    <div
                      data-testid="home-battle-pass-active-tier-progress-fill"
                      className="h-full rounded-full transition-all duration-300"
                      style={{ width: `${clampCompositeProgress(tierProgress)}%`, backgroundColor: clampCompositeProgress(tierProgress) >= 100 ? tokenCssVar('success') : tokenCssVar('info') }}
                    />
                  </div>
                </div>
                <span data-testid="home-battle-pass-active-tier-progress-target" className="shrink-0 whitespace-nowrap text-right">
                  {tierProgressLabels.target}
                </span>
              </div>
            </div>
          </div>
        ) : (
          <div className="flex min-h-[8.5rem] items-center justify-center">
            <p className="text-sm text-muted-foreground">Aucun palier disponible pour ce pass.</p>
          </div>
        )}
      </CardContent>
      <BattlePassRewardLightbox reward={activeReward} onClose={() => setActiveReward(null)} />
    </Card>
  )
}
