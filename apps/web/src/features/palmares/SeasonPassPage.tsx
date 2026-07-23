import { useCallback, useMemo, useRef, useState } from 'react'

import { useParams } from '@tanstack/react-router'

import { KpiCard } from '@/components/cards/KpiCard'
import { buildCompositeProgressEdgeLabels, clampCompositeProgress } from '@/components/ui/composite-progress-bar-labels'
import { DataFreshnessIndicator } from '@/components/ui/data-freshness-indicator'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { SeasonPassTrackSummary } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { seasonPassStatusRole } from './battlePassBadgeStyle'
import { BattlePassRewardCarousel } from './BattlePassRewardCarousel'
import { buildTierGroups, type RewardCard } from './battlePassTierGroups'
import { BattlePassRewardLightbox, type RewardLightboxData } from './BattlePassRewardLightbox'
import { getPalmaresText, normalizePalmaresLocale } from './i18n'
import { PassContentSummary, type ContentLabels } from './PassContentSummary'
import { useSeasonPassPage } from './queries'
import { SeasonPassBadge } from './SeasonPassBadge'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { isArmorItemType, rarityLabel, rarityStyle, type RarityTier } from './rarity'
import type { Locale } from '@/lib/i18n/locale'

// Ordre décroissant de rareté — du plus rare au plus commun.
const RARITY_ORDER: RarityTier[] = ['mythic', 'legendary', 'epic', 'rare', 'common']

type SeasonPassContent = NonNullable<SeasonPassTrackSummary['content']>

function OverlayContentRows({
  content,
  remaining,
  labels,
  locale,
  palmaresLocale,
}: {
  content: SeasonPassContent
  /** Contenu restant (paliers non atteints). Fourni ⇒ affichage « restant/total » (XX/YY). */
  remaining?: SeasonPassContent | null
  labels: ContentLabels
  locale: string
  palmaresLocale: Locale
}) {
  type Chip = { key: string; value: string; label: string }
  const showRemaining = remaining !== undefined
  const rem = remaining ?? null
  // Mode XX/YY : affiche l'ACQUIS (= total − restant) sur le total (cf. demande user :
  // « X obtenus sur Y », colle à la complétion). r = valeur des paliers NON atteints.
  const val = (total: number, r: number) =>
    showRemaining ? `${Math.max(0, total - r).toLocaleString(locale)}/${total.toLocaleString(locale)}` : total.toLocaleString(locale)
  const armorOf = (c: SeasonPassContent | null) => {
    let armor = 0
    if (c?.type_breakdown) for (const [type, count] of Object.entries(c.type_breakdown)) if (isArmorItemType(type)) armor += count
    return armor
  }

  const row1: Chip[] = []
  if (content.total_tiers > 0) row1.push({ key: 'tiers', value: val(content.total_tiers, rem?.total_tiers ?? 0), label: labels.tiersLabel })
  if (content.credits) row1.push({ key: 'cr', value: val(content.credits, rem?.credits ?? 0), label: labels.creditsLabel })
  if (content.spartan_points) row1.push({ key: 'sp', value: val(content.spartan_points, rem?.spartan_points ?? 0), label: labels.spartanPointsLabel })
  if (content.xp_boosts) row1.push({ key: 'xp', value: val(content.xp_boosts, rem?.xp_boosts ?? 0), label: labels.xpBoostsLabel })
  if (content.challenge_swaps) row1.push({ key: 'swap', value: val(content.challenge_swaps, rem?.challenge_swaps ?? 0), label: labels.challengeSwapsLabel })

  const row2: Chip[] = []
  if (content.cosmetics_total) {
    if (content.type_breakdown && Object.keys(content.type_breakdown).length > 0) {
      const armorT = armorOf(content)
      const cosmeticT = Math.max(0, content.cosmetics_total - armorT)
      const armorR = armorOf(rem)
      const cosmeticR = Math.max(0, (rem?.cosmetics_total ?? 0) - armorR)
      if (armorT > 0) row2.push({ key: 'armor', value: val(armorT, armorR), label: labels.armorLabel })
      if (cosmeticT > 0) row2.push({ key: 'cosmetic', value: val(cosmeticT, cosmeticR), label: labels.cosmeticsSplitLabel })
    } else {
      row2.push({ key: 'cosmetics', value: val(content.cosmetics_total, rem?.cosmetics_total ?? 0), label: labels.cosmeticsLabel })
    }
  }

  const rarities = RARITY_ORDER
    .map((tier) => ({ tier, count: content.rarity_breakdown?.[tier] ?? 0, r: rem?.rarity_breakdown?.[tier] ?? 0 }))
    .filter((e) => e.count > 0)

  if (row1.length === 0 && row2.length === 0 && rarities.length === 0) return null

  const chipRow = (chips: Chip[]) => chips.length === 0 ? null : (
    <div className="flex flex-wrap items-baseline gap-x-3 text-xs">
      {chips.map(({ key, value, label }) => (
        <span key={key}>
          <span className="font-semibold text-foreground tabular-nums">{value}</span>
          {' '}
          <span className="text-muted-foreground">{label}</span>
        </span>
      ))}
    </div>
  )

  return (
    <div className="mt-2 space-y-0.5">
      {/* Raretés EN PREMIER, puis items (cosmétiques), puis devises (cf. demande user). */}
      {rarities.length > 0 && (
        <div className="flex flex-wrap items-center gap-x-2.5 text-xs">
          {rarities.map(({ tier, count, r }) => (
            <span key={tier} className="flex items-center gap-1">
              <span className={`inline-block h-1.5 w-1.5 shrink-0 rounded-full ${rarityStyle(tier)?.segment ?? 'bg-muted-foreground/60'}`} />
              <span className="text-muted-foreground">{rarityLabel(tier, palmaresLocale)}</span>
              {' '}
              <span className="font-semibold text-foreground tabular-nums">{val(count, r)}</span>
            </span>
          ))}
        </div>
      )}
      {chipRow(row2)}
      {chipRow(row1)}
    </div>
  )
}

function StatCard({ label, value, accent, className = '' }: { label: string; value: string; accent?: SemanticToken; className?: string }) {
  // KPI card du catalogue (type 2 : accent fixe par métrique) — chrome bg-card + bordure.
  // className : contrôle de la largeur en flex (flex-none = largeur contenu, flex-1 = grandit).
  return (
    <KpiCard accent={accent} className={`flex flex-col ${className}`}>
      <div className="flex flex-1 flex-col p-3">
        <p className="text-2xs uppercase tracking-label-md text-muted-foreground">{label}</p>
        <p className="mt-1 whitespace-nowrap text-lg font-semibold text-foreground">{value}</p>
      </div>
    </KpiCard>
  )
}

// ── Agrégat multi-pass (collection complète) ────────────────────────────────
// content = total du pass, remaining_content = paliers PAS encore atteints.
// acquis = total − restant, sommé sur tous les pass.

interface PassAggregate {
  cosmeticsAcquired: number
  cosmeticsTotal: number
  rarityAcquired: Record<string, number>
  xpAcquired: number
  xpTotal: number
  credits: number
  spartanPoints: number
  xpBoosts: number
  challengeSwaps: number
}

const DEFAULT_XP_PER_RANK = 1000 // un palier = 1000 XP (constante métier Halo)

function aggregatePasses(passes: SeasonPassTrackSummary[]): PassAggregate {
  const acq = (tot?: number | null, rem?: number | null) => Math.max(0, (tot ?? 0) - (rem ?? 0))
  const agg: PassAggregate = {
    cosmeticsAcquired: 0, cosmeticsTotal: 0, rarityAcquired: {},
    xpAcquired: 0, xpTotal: 0, credits: 0, spartanPoints: 0, xpBoosts: 0, challengeSwaps: 0,
  }
  for (const p of passes) {
    const c = p.content
    if (!c) continue
    const r = p.remaining_content ?? null
    agg.cosmeticsTotal += c.cosmetics_total ?? 0
    agg.cosmeticsAcquired += acq(c.cosmetics_total, r?.cosmetics_total)
    agg.credits += acq(c.credits, r?.credits)
    agg.spartanPoints += acq(c.spartan_points, r?.spartan_points)
    agg.xpBoosts += acq(c.xp_boosts, r?.xp_boosts)
    agg.challengeSwaps += acq(c.challenge_swaps, r?.challenge_swaps)
    const xpPerRank = p.xp_per_rank ?? DEFAULT_XP_PER_RANK
    agg.xpTotal += (c.total_tiers ?? 0) * xpPerRank
    agg.xpAcquired += acq(c.total_tiers, r?.total_tiers) * xpPerRank
    if (c.rarity_breakdown) {
      for (const [tier, v] of Object.entries(c.rarity_breakdown)) {
        agg.rarityAcquired[tier] = (agg.rarityAcquired[tier] ?? 0) + acq(v, r?.rarity_breakdown?.[tier])
      }
    }
  }
  return agg
}

// ── Carte « Cosmétiques débloqués » : barre segmentée par rareté ────────────

function CosmeticsUnlockedCard({ label, agg, intlLocale, palmaresLocale, accent, className = '' }: {
  label: string
  agg: PassAggregate
  intlLocale: string
  palmaresLocale: Locale
  accent?: SemanticToken
  className?: string
}) {
  if (agg.cosmeticsTotal <= 0) return null
  const pct = Math.round((agg.cosmeticsAcquired / agg.cosmeticsTotal) * 100)
  const remainder = Math.max(0, agg.cosmeticsTotal - agg.cosmeticsAcquired)
  const rarities = RARITY_ORDER
    .map((tier) => ({ tier, acq: agg.rarityAcquired[tier] ?? 0 }))
    .filter((e) => e.acq > 0)

  return (
    <KpiCard accent={accent} className={`flex flex-col ${className}`}>
      <div className="flex flex-1 flex-col p-3">
        <p className="text-2xs uppercase tracking-label-md text-muted-foreground">{label}</p>
        {/* Valeur + % + barre composite sur une ligne, collée sous le titre (mt-1,
            comme StatCard). Compacité = hauteur harmonisée ; aucune info retirée. */}
        <div className="mt-1 flex items-center gap-3">
          <p className="shrink-0 text-lg font-semibold tabular-nums text-foreground">
            {agg.cosmeticsAcquired.toLocaleString(intlLocale)}
            <span className="text-sm font-normal text-muted-foreground"> / {agg.cosmeticsTotal.toLocaleString(intlLocale)}</span>
            <span className="ml-1.5 text-xs font-normal text-muted-foreground">{pct} %</span>
          </p>
          {/* Barre segmentée par rareté (mêmes infos qu'avant + tooltip par segment).
              Couleurs de rareté Halo (rarity.ts, exception tolérée règle 20 CLAUDE.md). */}
          <div className="flex h-2 min-w-0 flex-1 overflow-hidden rounded-full bg-muted">
            {rarities.map(({ tier, acq }) => (
              <div
                key={tier}
                className={rarityStyle(tier)?.segment ?? 'bg-muted-foreground/60'}
                style={{ flex: acq }}
                title={`${rarityLabel(tier, palmaresLocale)} : ${acq.toLocaleString(intlLocale)}`}
              />
            ))}
            {remainder > 0 && <div style={{ flex: remainder }} aria-hidden />}
          </div>
        </div>
        <div className="mt-auto flex flex-wrap gap-x-2.5 gap-y-0.5 text-2xs text-muted-foreground">
          {rarities.map(({ tier, acq }) => (
            <span key={tier} className="flex items-center gap-1">
              <span className={`inline-block h-1.5 w-1.5 shrink-0 rounded-full ${rarityStyle(tier)?.segment ?? 'bg-muted-foreground/60'}`} />
              {rarityLabel(tier, palmaresLocale)}
              <span className="font-semibold tabular-nums text-foreground">{acq.toLocaleString(intlLocale)}</span>
            </span>
          ))}
        </div>
      </div>
    </KpiCard>
  )
}

// ── Carte « Butin récolté » : CR / Pts Spartans / Boosts XP / Relances ──────

function LootCard({ label, agg, intlLocale, contentLabels, accent, className = '' }: {
  label: string
  agg: PassAggregate
  intlLocale: string
  contentLabels: ContentLabels
  accent?: SemanticToken
  className?: string
}) {
  const items: Array<{ key: string; value: number; itemLabel: string }> = []
  if (agg.credits > 0) items.push({ key: 'cr', value: agg.credits, itemLabel: contentLabels.creditsLabel })
  if (agg.spartanPoints > 0) items.push({ key: 'sp', value: agg.spartanPoints, itemLabel: contentLabels.spartanPointsLabel })
  if (agg.xpBoosts > 0) items.push({ key: 'xp', value: agg.xpBoosts, itemLabel: contentLabels.xpBoostsLabel })
  if (agg.challengeSwaps > 0) items.push({ key: 'swap', value: agg.challengeSwaps, itemLabel: contentLabels.challengeSwapsLabel })
  if (items.length === 0) return null

  return (
    <KpiCard accent={accent} className={`flex flex-col ${className}`}>
      <div className="flex flex-1 flex-col p-3">
        <p className="text-2xs uppercase tracking-label-md text-muted-foreground">{label}</p>
        {/* Une rangée (flex) collée sous le titre (mt-1, comme StatCard) : les 4 stats
            restent alignées tant que la card a de la largeur ; wrap propre si étroit. */}
        <div className="mt-1 flex flex-wrap items-center gap-x-6 gap-y-1.5">
          {items.map(({ key, value, itemLabel }) => (
            // Valeur au-dessus (préférence user), label dessous. Police de valeur
            // alignée sur les autres cards (text-lg semibold).
            <div key={key}>
              <p className="whitespace-nowrap text-lg font-semibold tabular-nums text-foreground">{value.toLocaleString(intlLocale)}</p>
              <p className="whitespace-nowrap text-2xs text-muted-foreground">{itemLabel}</p>
            </div>
          ))}
        </div>
      </div>
    </KpiCard>
  )
}

function ProgressBar({ value }: { value?: number | null }) {
  const width = value == null ? 0 : Math.max(0, Math.min(100, value))
  return (
    <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
      <div className="h-full rounded-full transition-[width]" style={{ width: `${width}%`, backgroundColor: width >= 100 ? tokenCssVar('success') : tokenCssVar('info') }} />
    </div>
  )
}

type Text = ReturnType<typeof getPalmaresText>

function SeasonPassCard({ pass, intlLocale, statusLabel, labels, contentLabels, isSelected, onSelect }: {
  pass: SeasonPassTrackSummary
  intlLocale: string
  statusLabel: string
  labels: { premium: string; active: string; rank: string; progress: string; nowShowing: string }
  contentLabels: Parameters<typeof PassContentSummary>[0]['labels']
  isSelected: boolean
  onSelect: (pass: SeasonPassTrackSummary) => void
}) {
  const background = pass.background_image_url ?? pass.image_url ?? null
  const rankValue = pass.max_rank
    ? `${pass.current_rank} / ${pass.max_rank}`
    : pass.current_rank.toLocaleString(intlLocale)

  return (
    <button
      type="button"
      onClick={() => onSelect(pass)}
      aria-pressed={isSelected}
      className={[
        'group relative block w-full min-h-[15rem] overflow-hidden rounded-xl border bg-card text-left shadow-sm transition-all hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        isSelected ? 'border-primary/70 ring-2 ring-primary/40' : 'border-border/70',
      ].join(' ')}
    >
      {background && (
        <>
          <div
            className="absolute inset-0 bg-cover bg-center transition-transform duration-500 group-hover:scale-105"
            style={{ backgroundImage: `url(${background})` }}
            aria-hidden="true"
          />
          {/* color-allow: gradient sombre fixe pour lisibilité du contenu sur image plein-cadre */}
          <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/40 to-black/50" aria-hidden="true" />
        </>
      )}
      <div className="relative flex h-full flex-col gap-4 p-6">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <h3 className="text-lg font-semibold text-foreground">{pass.name}</h3>
          <div className="flex flex-wrap gap-2">
            {isSelected && <SeasonPassBadge role="active" label={labels.nowShowing} />}
            {pass.is_active && pass.status !== 'active' && <SeasonPassBadge role="active" label={labels.active} />}
            {pass.premium_owned && <SeasonPassBadge role="premium" label={labels.premium} />}
            <SeasonPassBadge role={seasonPassStatusRole(pass.status)} label={statusLabel} />
          </div>
        </div>

        {pass.content && (
          <PassContentSummary
            content={pass.content}
            remaining={pass.remaining_content ?? null}
            labels={contentLabels}
            locale={intlLocale}
            compact
          />
        )}

        <div className="mt-auto space-y-3">
          <div className="flex items-baseline justify-between">
            <p className="text-xl font-semibold text-foreground">{rankValue}</p>
            <p className="text-xl font-semibold text-foreground">
              {pass.completion_percent == null
                ? '—'
                : `${pass.completion_percent.toLocaleString(intlLocale, { maximumFractionDigits: 0 })} %`}
            </p>
          </div>
          <ProgressBar value={pass.completion_percent} />
        </div>
      </div>
    </button>
  )
}

function PassShowcase({
  pass,
  text,
  locale,
  isViewingActive,
  onBackToActive,
  showcaseRef,
}: {
  pass: SeasonPassTrackSummary
  text: Text
  locale: Locale
  isViewingActive: boolean
  onBackToActive: () => void
  showcaseRef?: React.Ref<HTMLDivElement>
}) {
  const allCards = useMemo(
    () => (pass.tiers ? buildTierGroups(pass.tiers).flatMap((g) => g.cards) : []),
    [pass.tiers],
  )
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null)

  const allRewards = useMemo<RewardLightboxData[]>(
    () => allCards.map((card) => {
      const badges: RewardLightboxData['badges'] = []
      if (card.is_current) badges.push({ label: text.seasonPass.active, tone: 'current' })
      if (card.is_obtained) badges.push({ label: text.seasonPass.obtained, tone: 'obtained' })
      if (card.is_free) badges.push({ label: text.seasonPass.freeLabel, tone: 'free' })
      else badges.push({ label: text.seasonPass.premium, tone: 'premium' })
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
    [allCards, text.seasonPass.active, text.seasonPass.obtained, text.seasonPass.freeLabel, text.seasonPass.premium],
  )

  const handleOpenCard = useCallback((card: RewardCard) => {
    const idx = allCards.findIndex((c) => c.key === card.key)
    setSelectedIndex(idx >= 0 ? idx : null)
  }, [allCards])

  const tierProgress = pass.active_tier_progress_percent ?? 0
  // Pour les passes sans palier actif (complété, non commencé), rabat sur completion_percent.
  const barPercent = pass.active_tier_rank != null
    ? clampCompositeProgress(tierProgress)
    : clampCompositeProgress(pass.completion_percent ?? 0)
  const progressLabels = buildCompositeProgressEdgeLabels({
    partialProgress: pass.partial_progress,
    xpPerRank: pass.xp_per_rank,
    progressPercent: barPercent,
    locale: text.intlLocale,
  })

  return (
    <div ref={showcaseRef} className="space-y-6">

      {!isViewingActive && (
        <button
          type="button"
          onClick={onBackToActive}
          className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:rounded"
        >
          <span aria-hidden="true">←</span>
          <span>{text.seasonPass.backToActive}</span>
        </button>
      )}

      {/* Hero : image du pass avec infos en overlay */}
      {pass.background_image_url ? (
        <div className="relative overflow-hidden rounded-2xl">
          <img
            src={pass.background_image_url}
            alt={pass.name}
            className="aspect-[986/248] w-full object-cover"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-black/5 to-transparent" /> {/* color-allow: gradient sombre fixe pour lisibilité du titre hero (overlay sur image map) */}
          <div className="absolute inset-x-0 bottom-0 p-4 sm:p-5">
            <div className="flex items-center gap-1.5">
              <h2 className="text-xl font-bold tracking-tight text-white sm:text-2xl"> {/* color-allow: blanc sur gradient sombre fixe (hero overlay) */}
                {pass.name}
              </h2>
              <DataFreshnessIndicator
                snapshotAt={pass.snapshot_at}
                buildLabel={text.seasonPass.freshnessLastSync}
                locale={text.intlLocale}
                className="text-white/50 hover:text-white/80"
              />
            </div>
            <div className="mt-1 flex flex-wrap gap-1.5">
              <SeasonPassBadge role={seasonPassStatusRole(pass.status)} label={text.seasonPass.status[pass.status] ?? pass.status} />
              {/* status === 'active' affiche déjà « Actif » → pas de second badge. */}
              {pass.is_active && pass.status !== 'active' && <SeasonPassBadge role="active" label={text.seasonPass.active} />}
              {pass.premium_owned && <SeasonPassBadge role="premium" label={text.seasonPass.premium} />}
            </div>
            {pass.content && (
              <OverlayContentRows
                content={pass.content}
                remaining={pass.remaining_content ?? null}
                labels={text.seasonPass.content}
                locale={text.intlLocale}
                palmaresLocale={locale}
              />
            )}
          </div>
        </div>
      ) : (
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex items-center gap-1.5">
            <h2 className="text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">{pass.name}</h2>
            <DataFreshnessIndicator
              snapshotAt={pass.snapshot_at}
              buildLabel={text.seasonPass.freshnessLastSync}
              locale={text.intlLocale}
            />
          </div>
          <div className="flex flex-wrap gap-2">
            {pass.premium_owned && <SeasonPassBadge role="premium" label={text.seasonPass.premium} />}
            {pass.is_active && pass.status !== 'active' && <SeasonPassBadge role="active" label={text.seasonPass.active} />}
            <SeasonPassBadge role={seasonPassStatusRole(pass.status)} label={text.seasonPass.status[pass.status] ?? pass.status} />
          </div>
        </div>
      )}

      {pass.tiers && pass.tiers.length > 0 && (
        <div className="space-y-5">
          <BattlePassRewardCarousel
            tiers={pass.tiers}
            activeTierRank={pass.active_tier_rank}
            onOpenCard={handleOpenCard}
            freeLabel={text.seasonPass.freeLabel}
          />

          <div className="flex justify-center">
            <div className="grid w-2/3 grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2 text-3xs text-muted-foreground">
              <span data-testid="season-pass-active-tier-progress-current" className="shrink-0 whitespace-nowrap">
                {progressLabels.current}
              </span>
              <div className="min-w-0 overflow-hidden rounded-full bg-muted-foreground/25">
                <div className="h-2 w-full">
                  <div
                    data-testid="season-pass-active-tier-progress-fill"
                    className="h-full rounded-full transition-all duration-300"
                    style={{ width: `${barPercent}%`, backgroundColor: barPercent >= 100 ? tokenCssVar('success') : tokenCssVar('info') }}
                  />
                </div>
              </div>
              <span data-testid="season-pass-active-tier-progress-target" className="shrink-0 whitespace-nowrap text-right">
                {progressLabels.target}
              </span>
            </div>
          </div>
        </div>
      )}

      {selectedIndex !== null && (
        <BattlePassRewardLightbox
          rewards={allRewards}
          startIndex={selectedIndex}
          onClose={() => setSelectedIndex(null)}
        />
      )}
    </div>
  )
}

export function SeasonPassPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = normalizePalmaresLocale(useAppShellStore((state) => state.locale))
  const text = getPalmaresText(locale)
  const { data, isLoading, isError, error, refetch } = useSeasonPassPage(playerSlug)
  const [selectedPassPath, setSelectedPassPath] = useState<string | null>(null)
  const showcaseRef = useRef<HTMLDivElement | null>(null)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Spinner size="lg" />
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="p-6">
        <EmptyStateCard
          title={text.seasonPass.unavailableTitle}
          description={error?.message ?? text.seasonPass.unavailableDescription}
          actionLabel={text.seasonPass.retry}
          onAction={() => refetch()}
        />
      </div>
    )
  }

  // Le contrat autorise `passes` à être null (Go peut renvoyer null) → garde.
  const passes = data.passes ?? []
  const completedCount = passes.filter((p) => p.status === 'completed').length
  const inProgressCount = passes.filter((p) => p.status === 'in_progress').length
  const remainingPasses = passes.filter((p) => p.status !== 'completed').length
  const activePass = passes.find((p) => p.is_active) ?? null

  // Agrégat collection (tous pass) pour la 2e rangée de cards.
  const agg = aggregatePasses(passes)
  const compactNum = (n: number) => n.toLocaleString(text.intlLocale, { notation: 'compact', maximumFractionDigits: 2 })
  const selectedPass = (selectedPassPath
    ? passes.find((p) => p.reward_track_path === selectedPassPath)
    : null) ?? activePass
  const otherPasses = passes.filter((p) => p.reward_track_path !== selectedPass?.reward_track_path)
  const isViewingActive = !selectedPass || selectedPass.is_active

  function selectPass(pass: SeasonPassTrackSummary) {
    setSelectedPassPath(pass.reward_track_path)
    requestAnimationFrame(() => {
      showcaseRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }

  function backToActive() {
    setSelectedPassPath(null)
    requestAnimationFrame(() => {
      showcaseRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }

  return (
    <div className="flex flex-col gap-6 p-6">
      {/* Barre KPI sur une rangée (flex) : les compteurs (Terminés/En cours/Pass
          restants) tiennent leur largeur contenu (flex-none) ; « Cosmétiques débloqués »
          absorbe l'espace restant (flex-1). XP + Butin en largeur contenu. Cards
          collection null si pas de donnée → la rangée se réduit proprement. */}
      <div className="flex flex-wrap items-stretch gap-3">
        <StatCard label={text.seasonPass.completedCard} value={completedCount.toLocaleString(text.intlLocale)} accent="outcome-win" className="flex-none" />
        <StatCard label={text.seasonPass.inProgressCard} value={inProgressCount.toLocaleString(text.intlLocale)} accent="chart-series-1" className="flex-none" />
        <StatCard label={text.seasonPass.remainingPassesCard} value={remainingPasses.toLocaleString(text.intlLocale)} accent="warning" className="flex-none" />
        <CosmeticsUnlockedCard
          label={text.seasonPass.cosmeticsUnlockedCard}
          agg={agg}
          intlLocale={text.intlLocale}
          palmaresLocale={locale}
          accent="perf-tier-2"
          className="min-w-[15rem] flex-1"
        />
        {agg.xpTotal > 0 && (
          <StatCard
            label={text.seasonPass.xpUnlockedCard}
            value={`${compactNum(agg.xpAcquired)} / ${compactNum(agg.xpTotal)}`}
            accent="chart-series-4"
            className="flex-none"
          />
        )}
        <LootCard
          label={text.seasonPass.lootCard}
          agg={agg}
          intlLocale={text.intlLocale}
          contentLabels={text.seasonPass.content}
          accent="outcome-draw"
          className="min-w-[15rem] flex-1"
        />
      </div>

      {/* Section pass : titre juste sous la barre KPI, au-dessus du showcase. « Pass
          actif » quand on regarde le pass en cours, « Pass saisonnier » quand on a
          sélectionné un autre pass via « Autres passes ». */}
      {selectedPass ? (
        <section className="space-y-3">
          <header><h3 className="text-base font-semibold text-foreground">{isViewingActive ? text.seasonPass.activePassTitle : text.seasonPass.selectedPassTitle}</h3></header>
          <PassShowcase
            pass={selectedPass}
            text={text}
            locale={locale}
            isViewingActive={isViewingActive}
            onBackToActive={backToActive}
            showcaseRef={showcaseRef}
          />
        </section>
      ) : passes.length === 0 ? (
        <EmptyStateCard title={text.seasonPass.noPassesTitle} description={text.seasonPass.noPassesDescription} />
      ) : null}

      {otherPasses.length > 0 && (
        <section className="space-y-3">
          <header><h3 className="text-base font-semibold text-foreground">{text.seasonPass.otherPassesTitle}</h3></header>
          <div className="grid gap-4 xl:grid-cols-2">
            {otherPasses.map((pass) => (
              <SeasonPassCard
                key={pass.reward_track_path}
                pass={pass}
                intlLocale={text.intlLocale}
                statusLabel={text.seasonPass.status[pass.status] ?? pass.status}
                contentLabels={text.seasonPass.content}
                isSelected={false}
                onSelect={selectPass}
                labels={{
                  premium: text.seasonPass.premium,
                  active: text.seasonPass.active,
                  rank: text.seasonPass.cardRank,
                  progress: text.seasonPass.cardProgress,
                  nowShowing: text.seasonPass.nowShowing,
                }}
              />
            ))}
          </div>
        </section>
      )}
    </div>
  )
}
