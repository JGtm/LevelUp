/**
 * HomePrestigeSection — section unifiée Prestige sur la home page.
 *
 * Une seule Card regroupant :
 *   - Header : titre Prestige + lien "Gérer" (vers /objectifs)
 *   - Top    : barre composite progression PP (style rang carrière)
 *   - Bottom : grille 2 colonnes (Arc en cours | Mes objectifs)
 *
 * Remplace les anciennes sections séparées (HomePrestigeBar / HomeActiveArcCard /
 * carousel de challenges). Si toutes les sous-données sont vides ou en erreur
 * (PRESTIGE_ENABLED=false), retourne null silencieusement.
 *
 * Tous les libellés FR/EN passent par homeManifest (clés home.prestige.*).
 */
import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent } from '@/components/ui/card'
import { CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { ArcSummary } from '@/features/prestige/components/ArcSummary'
import { ObjectiveRow } from '@/features/prestige/components/ObjectiveRow'
import { useMyPrestige } from '@/features/prestige/hooks/usePrestige'
import { queryKeys } from '@/lib/query/keys'
import { useArcs } from '@/features/prestige/hooks/useArcs'
import { prestigeApi, type Challenge } from '@/lib/prestige'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'

interface HomePrestigeSectionProps {
  playerSlug: string
  titleSlug: string
  locale: ManifestLocale
}

const VISIBLE_OBJECTIVES = 3
const ROTATION_MS = 4500
const FADE_MS = 250

/**
 * Tri d'affichage des objectifs.
 *
 * Ordre :
 *   1. Actifs avant terminés.
 *   2. % de progression desc (current_value / target).
 *   3. Tie-break : created_at desc pour les actifs, completed_at desc pour les terminés.
 */
function progressPct(c: Challenge): number {
  if (!c.target || c.target <= 0) return 0
  return Math.min(100, ((c.current_value ?? 0) / c.target) * 100)
}

function sortObjectives(challenges: Challenge[]): Challenge[] {
  return [...challenges].sort((a, b) => {
    if (a.status !== b.status) return a.status === 'active' ? -1 : 1
    const dPct = progressPct(b) - progressPct(a)
    if (dPct !== 0) return dPct
    if (a.status === 'completed') {
      const aT = a.completed_at ?? a.created_at
      const bT = b.completed_at ?? b.created_at
      return bT.localeCompare(aT)
    }
    return b.created_at.localeCompare(a.created_at)
  })
}

export function HomePrestigeSection({ playerSlug, titleSlug, locale }: HomePrestigeSectionProps) {
  const [filter, setFilter] = useState<'active' | 'completed'>('active')
  const numberLocale = locale === 'en' ? 'en-US' : 'fr-FR'
  const t = (key: HomeManifestKey, values?: Record<string, string | number>) =>
    formatMessage(homeManifest, key, locale, values)

  const prestige = useMyPrestige(playerSlug, titleSlug)
  const arcsQ = useArcs(playerSlug, titleSlug)
  const challengesQ = useQuery({
    queryKey: queryKeys.challenge.list(playerSlug, titleSlug),
    queryFn: () => prestigeApi.listActiveChallenges(playerSlug, titleSlug),
    retry: false,
    staleTime: 30_000,
  })

  const challenges = challengesQ.data?.challenges ?? []
  const filteredObjectives = useMemo(
    () =>
      sortObjectives(
        challenges.filter((c) => (filter === 'active' ? c.status === 'active' : c.status === 'completed')),
      ),
    [challenges, filter],
  )

  // Pagination par fenêtres de 3 + rotation auto si > 3 objectifs.
  const totalPages = Math.max(1, Math.ceil(filteredObjectives.length / VISIBLE_OBJECTIVES))
  const [page, setPage] = useState(0)
  const [fading, setFading] = useState(false)
  // Reset page si le filtre change ou si la liste rétrécit.
  useEffect(() => {
    setPage(0)
  }, [filter, totalPages])
  const pagesRef = useRef(totalPages)
  pagesRef.current = totalPages
  useEffect(() => {
    if (totalPages <= 1) return
    const iv = window.setInterval(() => {
      setFading(true)
      window.setTimeout(() => {
        setPage((p) => (p + 1) % pagesRef.current)
        setFading(false)
      }, FADE_MS)
    }, ROTATION_MS)
    return () => window.clearInterval(iv)
  }, [totalPages])
  const visibleObjectives = filteredObjectives.slice(
    page * VISIBLE_OBJECTIVES,
    page * VISIBLE_OBJECTIVES + VISIBLE_OBJECTIVES,
  )

  // `arcsQ.data?.arcs` peut être null (Go nil slice → JSON null vs []),
  // donc `?.` sur `data` ne suffit pas — il faut aussi `?.` sur `arcs`.
  const activeArc = arcsQ.data?.arcs?.find((a) => a.completed_at == null) ?? null

  // Étapes de l'arc actif (même calcul que la page Ascension) : total = challenges
  // liés, completed = ceux terminés. Sans ça, l'arc s'affichait sans barre de
  // progression (ArcSummary masque la barre quand totalSteps === 0).
  const arcSteps = useMemo(() => {
    if (!activeArc) return { completed: 0, total: 0, totalPP: 0 }
    let completed = 0
    let total = 0
    let totalPP = 0
    for (const c of challenges) {
      if (c.arc_id !== activeArc.id) continue
      total += 1
      totalPP += c.pp_reward ?? 0
      if (c.status === 'completed') completed += 1
    }
    return { completed, total, totalPP }
  }, [challenges, activeArc])

  // Toutes les sources en erreur → feature désactivée → on ne rend rien.
  if (prestige.isError && arcsQ.isError && challengesQ.isError) return null
  if (prestige.isLoading) return null

  const pp = prestige.data
  const lvl = pp?.level
  const ppLabel = lvl?.name ?? t('home.prestige.level_label', { n: pp?.current_level ?? 0 })
  const isMax = lvl ? lvl.next_threshold_pp <= 0 : false
  const progressPct = lvl ? Math.round(lvl.progress_ratio * 100) : 0

  return (
    <section className="flex h-full flex-col gap-3">
      {/* Titre de section (type 1) sorti de la carte (cf. demande user) ; lien Gérer à droite. */}
      <header className="flex flex-row items-center justify-between gap-3">
        <h3 className="text-base font-semibold text-foreground">{t('home.prestige.title')}</h3>
        <Link
          to="/players/$playerSlug/ascension"
          params={{ playerSlug }}
          className="inline-flex shrink-0 items-center gap-1 rounded-md bg-primary/10 px-3 py-1.5 text-sm font-semibold text-primary transition-colors hover:bg-primary/20"
        >
          {t('home.prestige.manage')} →
        </Link>
      </header>
      <Card data-testid="home-prestige-section" className="relative flex-1 overflow-hidden isolate">
      {/* Background décoratif Prestige (pattern hexagonal cyber). */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 bg-cover bg-center bg-no-repeat opacity-60"
        style={{ backgroundImage: "url('/static/prestige-assets/prestige-bg.webp')" }}
      />
      {/* Overlay pour préserver la lisibilité du contenu. */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 bg-card/70"
      />

      <CardContent className="space-y-5 pt-6">
        {/* ─── Barre composite niveau / PP (fond opaque) ─── */}
        {pp && (
          <div className="space-y-2 rounded-lg border border-border bg-card p-3">
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
              <span className="shrink-0 whitespace-nowrap text-3xs font-medium text-foreground/85 tabular-nums">
                {pp.total_pp.toLocaleString(numberLocale)} PP
              </span>
              <CompositeProgressBar value={progressPct} fillTestId="home-prestige-progress-fill" />
              <span className="shrink-0 whitespace-nowrap text-3xs font-medium text-foreground/85 tabular-nums">
                {isMax ? '—' : `${(lvl?.next_threshold_pp ?? 0).toLocaleString(numberLocale)} PP`}
              </span>
            </div>
          </div>
        )}

        {/* ─── Grille Objectifs (gauche, fond opaque) | Arc (droite, semi-transparent) ─── */}
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,0.55fr)_minmax(0,0.45fr)]">
          {/* Mes objectifs — titre + filtre SORTIS du bloc (cf. demande user) ;
              la boîte bordée n'enveloppe plus que la liste. */}
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-2">
              <h3 className="text-3xs font-semibold uppercase tracking-label-md text-foreground/90">
                {t('home.prestige.objectives_section')}
              </h3>
              <div className="flex items-center rounded-md border border-border bg-background p-0.5 text-2xs">
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

            <div className="rounded-lg border border-border bg-card p-3">
            {challengesQ.isLoading ? (
              <div className="space-y-2">
                {[0, 1].map((i) => (
                  <div key={i} className="h-16 w-full animate-pulse rounded-lg bg-muted" />
                ))}
              </div>
            ) : filteredObjectives.length === 0 ? (
              <EmptyStateNotice
                title={
                  filter === 'active'
                    ? t('home.prestige.objectives_empty_active_title')
                    : t('home.prestige.objectives_empty_completed_title')
                }
                description={t('home.prestige.objectives_empty_description')}
              />
            ) : (
              <div className="space-y-2">
                <div
                  className={`space-y-1.5 transition-opacity duration-200 ${fading ? 'opacity-0' : 'opacity-100'}`}
                  aria-live="polite"
                >
                  {visibleObjectives.map((c) => (
                    <ObjectiveRow key={c.id} challenge={c} />
                  ))}
                </div>
                {totalPages > 1 && (
                  <div className="flex items-center justify-between gap-2 pt-1">
                    <Link
                      to="/players/$playerSlug/ascension"
                      params={{ playerSlug }}
                      className="text-2xs uppercase tracking-wider text-muted-foreground transition-colors hover:text-foreground"
                    >
                      {t('home.prestige.objectives_view_all', { n: filteredObjectives.length })}
                    </Link>
                    <div className="flex gap-1" aria-hidden="true">
                      {Array.from({ length: totalPages }).map((_, i) => (
                        <span
                          key={i}
                          className={`h-1 w-4 rounded-full transition-colors ${
                            i === page ? 'bg-primary' : 'bg-border'
                          }`}
                        />
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
            </div>
          </div>

          {/* Arc en cours — fond semi-transparent (laisse voir le pattern) */}
          <div className="space-y-2">
            <h3 className="text-3xs font-semibold uppercase tracking-label-md text-foreground/90">
              {t('home.prestige.arc_section')}
            </h3>
            {arcsQ.isLoading ? (
              <div className="h-24 w-full animate-pulse rounded-lg bg-muted" />
            ) : activeArc ? (
              <ArcSummary arc={activeArc} completedSteps={arcSteps.completed} totalSteps={arcSteps.total} totalPP={arcSteps.totalPP} />
            ) : (
              <EmptyStateNotice
                title={t('home.prestige.arc_empty_title')}
                description={t('home.prestige.arc_empty_description')}
              />
            )}
          </div>
        </div>
      </CardContent>
      </Card>
    </section>
  )
}
