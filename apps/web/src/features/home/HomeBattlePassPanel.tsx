import { useCallback, useMemo, useState, type ReactNode } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { buildCompositeProgressEdgeLabels, clampCompositeProgress } from '@/components/ui/composite-progress-bar-labels'
import { DataFreshnessIndicator } from '@/components/ui/data-freshness-indicator'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SeasonPassPageResponse, SeasonPassTrackSummary } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'
import { useAppShellStore } from '@/stores/appShellStore'
import { BattlePassRewardLightbox, type RewardLightboxData } from '@/features/palmares/BattlePassRewardLightbox'
import { BattlePassRewardCarousel } from '@/features/palmares/BattlePassRewardCarousel'
import { buildTierGroups, type RewardCard } from '@/features/palmares/battlePassTierGroups'
import { PassContentSummary } from '@/features/palmares/PassContentSummary'
import { getPalmaresText, normalizePalmaresLocale } from '@/features/palmares/i18n'

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
  const locale = useAppShellStore((state) => state.locale)
  const intlLocale = locale === 'en' ? 'en-GB' : 'fr-FR'
  const t = useCallback((key: HomeManifestKey) => formatMessage(homeManifest, key, locale), [locale])
  // Labels du contenu de pass (paliers, cR, raretés…) — réutilise l'i18n palmares.
  const passContentLabels = getPalmaresText(normalizePalmaresLocale(locale)).seasonPass.content
  const buildFreshnessLabel = useCallback(
    (date: string) =>
      formatMessage(homeManifest, 'home.freshness.last_sync', locale, { date }),
    [locale],
  )
  const featuredPass = pickFeaturedPass(data?.passes ?? [])
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null)

  const allCards = useMemo<RewardCard[]>(
    () => (featuredPass?.tiers ? buildTierGroups(featuredPass.tiers).flatMap((g) => g.cards) : []),
    [featuredPass],
  )

  const allRewards = useMemo<RewardLightboxData[]>(
    () => allCards.map((card) => {
      const badges: RewardLightboxData['badges'] = []
      if (card.is_current) badges.push({ label: t('home.battle_pass.badge_current'), tone: 'current' })
      if (card.is_obtained) badges.push({ label: t('home.battle_pass.badge_obtained'), tone: 'obtained' })
      if (card.is_free) badges.push({ label: t('home.battle_pass.badge_free'), tone: 'free' })
      else badges.push({ label: t('home.battle_pass.badge_owned'), tone: 'premium' })
      return {
        title: card.title,
        rank: card.rank,
        imageUrl: card.image_url ?? null,
        description: card.description ?? null,
        quality: card.quality ?? null,
        itemType: card.item_type ?? null,
        badges,
      }
    }),
    [allCards, t],
  )

  const handleOpenCard = useCallback((card: RewardCard) => {
    const idx = allCards.findIndex((c) => c.key === card.key)
    setSelectedIndex(idx >= 0 ? idx : null)
  }, [allCards])

  // Titre de section (type 1 du catalogue UI) + (i) freshness, SORTI au-dessus
  // de la carte (cf. demande user). Partagé par tous les états du panneau.
  const withSection = (card: ReactNode) => (
    <section className="flex flex-col gap-3">
      <header className="flex items-center gap-1.5">
        <h3 className="text-base font-semibold text-foreground">{t('home.battle_pass.title')}</h3>
        <DataFreshnessIndicator
          snapshotAt={featuredPass?.snapshot_at}
          buildLabel={buildFreshnessLabel}
          locale={intlLocale}
        />
      </header>
      {card}
    </section>
  )

  if (loading) {
    return withSection(
      <Card className="min-h-[14rem] flex-1">
        <CardContent className="pt-6">
          <p className="text-sm text-muted-foreground">{t('home.battle_pass.loading')}</p>
        </CardContent>
      </Card>,
    )
  }

  if (!data?.available) {
    return withSection(
      <Card className="min-h-[14rem] flex-1">
        <CardContent className="pt-6">
          <p className="text-sm text-muted-foreground">
            {t('home.battle_pass.unavailable_prefix')} ({errorHint ?? data?.error_hint ?? t('home.battle_pass.default_hint')})
          </p>
        </CardContent>
      </Card>,
    )
  }

  if (!featuredPass) {
    return withSection(
      <Card className="min-h-[14rem] flex-1">
        <CardContent className="pt-6">
          <EmptyStateNotice
            title={t('home.battle_pass.no_pass_title')}
            description={t('home.battle_pass.no_pass_description')}
          />
        </CardContent>
      </Card>,
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

  return withSection(
    <Card className="relative flex min-h-[14rem] flex-1 flex-col overflow-hidden border-border bg-card shadow-sm">
      <CardHeader className="relative pb-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <h3 className="text-xl font-semibold tracking-tight text-foreground sm:text-2xl">
            {featuredPass.name}
          </h3>

          <div className="flex flex-wrap gap-2">
            {featuredPass.premium_owned && <Badge variant="outline">{t('home.battle_pass.badge_owned')}</Badge>}
            {featuredPass.is_active && <Badge variant="info">{t('home.battle_pass.badge_active')}</Badge>}
          </div>
        </div>
      </CardHeader>

      <CardContent className="relative space-y-6">
        <div className="relative overflow-hidden rounded-xl border border-border bg-card shadow-[0_24px_72px_-44px_rgba(15,23,42,0.92)]">
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
                <p className="text-xs uppercase tracking-label-3xl text-muted-foreground">{t('home.battle_pass.active_pass_label')}</p>
                <p className="mt-3 text-2xl font-semibold sm:text-3xl">{featuredPass.name}</p>
              </div>
            </div>
          )}
          {featuredPass.content && (
            <>
              {/* color-allow: gradient sombre fixe pour la lisibilité de l'overlay « restant » sur l'image (parité page dédiée). */}
              <div className="pointer-events-none absolute inset-0 bg-gradient-to-t from-black/85 via-black/40 to-transparent" aria-hidden="true" />
              <div className="absolute inset-x-0 bottom-0 p-3">
                <PassContentSummary
                  content={featuredPass.content}
                  remaining={featuredPass.remaining_content ?? null}
                  labels={passContentLabels}
                  locale={intlLocale}
                  compact
                />
              </div>
            </>
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
                className="grid w-2/3 grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2 text-3xs text-muted-foreground"
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
            <p className="text-sm text-muted-foreground">{t('home.battle_pass.no_tiers')}</p>
          </div>
        )}
      </CardContent>
      {selectedIndex !== null && (
        <BattlePassRewardLightbox
          rewards={allRewards}
          startIndex={selectedIndex}
          onClose={() => setSelectedIndex(null)}
        />
      )}
    </Card>,
  )
}
