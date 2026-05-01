/**
 * HomePrestigeSection — section unifiée Prestige sur la home page.
 *
 * Une seule Card regroupant :
 *   - Header : titre Prestige + lien "Gérer" (vers /objectifs)
 *   - Top    : barre composite progression PP (style rang carrière)
 *   - Bottom : grille 2 colonnes (Arc en cours | Mes objectifs)
 *
 * Remplace les 3 sections séparées (HomePrestigeBar / HomeActiveArcCard /
 * ChallengesCarousel embarqué dans une Card vide). Si toutes les sous-données
 * sont vides ou en erreur (PRESTIGE_ENABLED=false), retourne null silencieusement.
 *
 * Tous les libellés FR/EN passent par homeManifest (clés home.prestige.*).
 */
import { useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { ArcSummary } from '@/features/prestige/components/ArcSummary'
import { ObjectiveRow } from '@/features/prestige/components/ObjectiveRow'
import { useMyPrestige } from '@/features/prestige/hooks/usePrestige'
import { useArcs } from '@/features/prestige/hooks/useArcs'
import { prestigeApi, type Cadence, type Challenge } from '@/lib/prestige'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'

interface HomePrestigeSectionProps {
  playerSlug: string
  titleSlug: string
  locale: ManifestLocale
}

const CADENCE_ORDER: Cadence[] = ['daily', 'weekly', 'monthly', 'free']

const CADENCE_KEY: Record<Cadence, HomeManifestKey> = {
  daily: 'home.prestige.cadence_daily',
  weekly: 'home.prestige.cadence_weekly',
  monthly: 'home.prestige.cadence_monthly',
  free: 'home.prestige.cadence_free',
}

function groupByCadence(challenges: Challenge[]): Map<Cadence, Challenge[]> {
  const out = new Map<Cadence, Challenge[]>()
  for (const c of CADENCE_ORDER) out.set(c, [])
  for (const c of challenges) {
    const key = (c.cadence ?? 'free') as Cadence
    const arr = out.get(key) ?? out.get('free')!
    arr.push(c)
  }
  return out
}

export function HomePrestigeSection({ playerSlug, titleSlug, locale }: HomePrestigeSectionProps) {
  const [filter, setFilter] = useState<'active' | 'completed'>('active')
  const numberLocale = locale === 'en' ? 'en-US' : 'fr-FR'
  const t = (key: HomeManifestKey, values?: Record<string, string | number>) =>
    formatMessage(homeManifest, key, locale, values)

  const prestige = useMyPrestige(playerSlug, titleSlug)
  const arcsQ = useArcs(playerSlug, titleSlug)
  const challengesQ = useQuery({
    queryKey: ['prestige', 'challenges', playerSlug, titleSlug],
    queryFn: () => prestigeApi.listActiveChallenges(playerSlug, titleSlug),
    retry: false,
    staleTime: 30_000,
  })

  const challenges = challengesQ.data?.challenges ?? []
  const filteredObjectives = useMemo(
    () => challenges.filter((c) => (filter === 'active' ? c.status === 'active' : c.status === 'completed')),
    [challenges, filter],
  )
  const grouped = useMemo(() => groupByCadence(filteredObjectives), [filteredObjectives])
  const visibleSections = CADENCE_ORDER.filter((c) => (grouped.get(c)?.length ?? 0) > 0)

  const activeArc = arcsQ.data?.arcs.find((a) => a.completed_at == null) ?? null

  // Toutes les sources en erreur → feature désactivée → on ne rend rien.
  if (prestige.isError && arcsQ.isError && challengesQ.isError) return null
  if (prestige.isLoading) return null

  const pp = prestige.data
  const lvl = pp?.level
  const ppLabel = lvl?.name ?? t('home.prestige.level_label', { n: pp?.current_level ?? 0 })
  const isMax = lvl ? lvl.next_threshold_pp <= 0 : false
  const progressPct = lvl ? Math.round(lvl.progress_ratio * 100) : 0

  return (
    <Card data-testid="home-prestige-section">
      <CardHeader className="flex flex-row items-center justify-between gap-3 space-y-0 pb-2">
        <CardTitle className="text-base">{t('home.prestige.title')}</CardTitle>
        <Link
          to="/players/$playerSlug/objectifs"
          params={{ playerSlug }}
          className="text-xs text-muted-foreground transition-colors hover:text-foreground"
        >
          {t('home.prestige.manage')} →
        </Link>
      </CardHeader>

      <CardContent className="space-y-5">
        {/* ─── Barre composite niveau / PP ─── */}
        {pp && (
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-3 text-xs">
              <span className="font-semibold text-foreground">
                {ppLabel}
                <span className="ml-2 text-muted-foreground">
                  {t('home.prestige.level_label', { n: pp.current_level })}
                </span>
              </span>
              <span className="text-muted-foreground">
                {isMax ? t('home.prestige.max_level') : `${progressPct}%`}
              </span>
            </div>
            <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3">
              <span className="shrink-0 whitespace-nowrap text-[11px] font-medium text-foreground/85 tabular-nums">
                {pp.total_pp.toLocaleString(numberLocale)} PP
              </span>
              <CompositeProgressBar value={progressPct} fillTestId="home-prestige-progress-fill" />
              <span className="shrink-0 whitespace-nowrap text-[11px] font-medium text-foreground/85 tabular-nums">
                {isMax ? '—' : `${(lvl?.next_threshold_pp ?? 0).toLocaleString(numberLocale)} PP`}
              </span>
            </div>
          </div>
        )}

        {/* ─── Grille Arc (gauche) | Objectifs (droite) ─── */}
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,0.45fr)_minmax(0,0.55fr)]">
          {/* Arc en cours */}
          <div className="space-y-2">
            <h3 className="text-[11px] font-semibold uppercase tracking-[0.18em] text-foreground/90">
              {t('home.prestige.arc_section')}
            </h3>
            {arcsQ.isLoading ? (
              <div className="h-24 w-full animate-pulse rounded-lg bg-muted" />
            ) : activeArc ? (
              <ArcSummary arc={activeArc} />
            ) : (
              <EmptyStateNotice
                title={t('home.prestige.arc_empty_title')}
                description={t('home.prestige.arc_empty_description')}
              />
            )}
          </div>

          {/* Mes objectifs */}
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-2">
              <h3 className="text-[11px] font-semibold uppercase tracking-[0.18em] text-foreground/90">
                {t('home.prestige.objectives_section')}
              </h3>
              <div className="flex items-center rounded-md border border-border bg-card p-0.5 text-[10px]">
                <button
                  type="button"
                  onClick={() => setFilter('active')}
                  className={[
                    'rounded px-1.5 py-0.5 transition-colors',
                    filter === 'active'
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:text-foreground',
                  ].join(' ')}
                >
                  {t('home.prestige.objectives_filter_active')}
                </button>
                <button
                  type="button"
                  onClick={() => setFilter('completed')}
                  className={[
                    'rounded px-1.5 py-0.5 transition-colors',
                    filter === 'completed'
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:text-foreground',
                  ].join(' ')}
                >
                  {t('home.prestige.objectives_filter_completed')}
                </button>
              </div>
            </div>

            {challengesQ.isLoading ? (
              <div className="space-y-2">
                {[0, 1].map((i) => (
                  <div key={i} className="h-16 w-full animate-pulse rounded-lg bg-muted" />
                ))}
              </div>
            ) : visibleSections.length === 0 ? (
              <EmptyStateNotice
                title={
                  filter === 'active'
                    ? t('home.prestige.objectives_empty_active_title')
                    : t('home.prestige.objectives_empty_completed_title')
                }
                description={t('home.prestige.objectives_empty_description')}
              />
            ) : (
              <div className="space-y-3">
                {visibleSections.map((c) => (
                  <CadenceGroup key={c} cadence={c} items={grouped.get(c) ?? []} t={t} />
                ))}
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function CadenceGroup({
  cadence,
  items,
  t,
}: {
  cadence: Cadence
  items: Challenge[]
  t: (key: HomeManifestKey, values?: Record<string, string | number>) => string
}) {
  return (
    <div className="space-y-1.5">
      <p className="text-[10px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
        {t(CADENCE_KEY[cadence])}
      </p>
      <div className="space-y-1.5">
        {items.map((c) => (
          <ObjectiveRow key={c.id} challenge={c} />
        ))}
      </div>
    </div>
  )
}
