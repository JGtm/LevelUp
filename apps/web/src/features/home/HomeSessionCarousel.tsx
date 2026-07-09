/**
 * HomeSessionCarousel — carrousel vertical des sessions récentes (solo/escouade).
 *
 * P8.4 (revue 2026-04-29) : extrait de HomePage.tsx (SessionCarouselCard +
 * ChevronUp/Down icons + helpers formatSession*). Réduit la god page de ~245L.
 */
import { useEffect, useRef, useState } from 'react'
import type { SessionSummaryItem } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'
import { kdaDivergentScale } from '@/lib/accessibility/scales'
import { getPerfColor } from '@/lib/perf-color'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { OutcomeBar } from '@/components/ui/outcome-bar'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import type { ManifestLocale } from '@/lib/i18n/format'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'

function ChevronUpIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="m18 15-6-6-6 6" />
    </svg>
  )
}

function ChevronDownIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="m6 9 6 6 6-6" />
    </svg>
  )
}

function formatSessionDate(startedAt: string | null, locale: ManifestLocale): string {
  if (!startedAt) return ''
  const d = new Date(startedAt)
  const now = new Date()
  const oneYearAgo = new Date(now.getFullYear() - 1, now.getMonth(), now.getDate())
  const dateOpts: Intl.DateTimeFormatOptions =
    d < oneYearAgo
      ? { day: 'numeric', month: 'short', year: 'numeric' }
      : { day: 'numeric', month: 'short' }
  const loc = intlLocale(locale)
  const sep = locale === 'en' ? ' at ' : ' à '
  return (
    d.toLocaleDateString(loc, dateOpts) +
    sep +
    d.toLocaleTimeString(loc, { hour: '2-digit', minute: '2-digit' })
  )
}

function formatSessionDuration(startedAt: string | null, endedAt: string | null): string {
  if (!startedAt || !endedAt) return ''
  const diffMs = new Date(endedAt).getTime() - new Date(startedAt).getTime()
  if (diffMs <= 0) return ''
  const totalMin = Math.round(diffMs / 60000)
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  if (h === 0) return `${m}min`
  return `${h}h${m > 0 ? String(m).padStart(2, '0') : ''}`
}

interface HomeSessionCarouselProps {
  sessions: SessionSummaryItem[]
  idx: number
  onIdxChange: (idx: number) => void
  variant: 'solo' | 'squad'
  playerSlug: string
  /** Navigue vers la page de stats du contexte (solo → Timeseries, squad → /squad).
   *  `teammates` = coéquipiers de la session (ignoré côté solo, utilisé côté squad
   *  pour pré-sélectionner la composition). */
  onNavigate: (sessionLabel: string, teammates: string[]) => void
}

export function HomeSessionCarousel({
  sessions,
  idx,
  onIdxChange,
  variant,
  onNavigate,
}: HomeSessionCarouselProps) {
  const [displayIdx, setDisplayIdx] = useState(idx)
  const [isAnimating, setIsAnimating] = useState(false)
  const contentRef = useRef<HTMLDivElement>(null)
  const cleanupRef = useRef<(() => void) | null>(null)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  // th : clés home.sessions.* (GH2-B4 — libellés du carrousel bilingues).
  const th = (key: HomeManifestKey, values?: Record<string, string | number>) =>
    formatMessage(homeManifest, key, locale, values)
  const dominantTooltip = th('home.sessions.dominant_tooltip')

  // Nettoyage si le composant est démonté en cours d'animation
  useEffect(() => () => { cleanupRef.current?.() }, [])

  const handleChange = (newIdx: number) => {
    if (isAnimating) return
    const el = contentRef.current
    if (!el) {
      setDisplayIdx(newIdx)
      onIdxChange(newIdx)
      return
    }

    const dir = newIdx < displayIdx ? 'up' : 'down'
    if (cleanupRef.current) cleanupRef.current()
    setIsAnimating(true)

    const exitY = dir === 'up' ? '-10px' : '10px'
    el.style.transition = 'transform 0.06s ease-in, opacity 0.06s ease-in'
    el.style.transform = `translateY(${exitY})`
    el.style.opacity = '0'

    const t1 = setTimeout(() => {
      setDisplayIdx(newIdx)
      onIdxChange(newIdx)
      const entryY = dir === 'up' ? '10px' : '-10px'
      el.style.transition = 'none'
      el.style.transform = `translateY(${entryY})`
      el.style.opacity = '0'

      const rafId = requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          el.style.transition = 'transform 0.1s ease-out, opacity 0.08s ease-out'
          el.style.transform = 'translateY(0)'
          el.style.opacity = '1'
          const t2 = setTimeout(() => setIsAnimating(false), 100)
          cleanupRef.current = () => clearTimeout(t2)
        })
      })
      cleanupRef.current = () => cancelAnimationFrame(rafId)
    }, 70)

    cleanupRef.current = () => clearTimeout(t1)
  }

  const session = sessions[displayIdx]
  const total = sessions.length
  const variantLabel = variant === 'solo' ? th('home.sessions.solo') : th('home.sessions.squad')
  const cardClass = variant === 'solo' ? 'rounded-md bg-muted p-3' : 'rounded-md bg-primary/10 p-3'
  const chevronBottomCls =
    'flex w-full cursor-pointer items-center justify-center py-2 text-muted-foreground/40 transition-colors hover:text-muted-foreground focus-visible:outline-none focus-visible:text-foreground disabled:cursor-not-allowed disabled:opacity-20'

  return (
    <div className="flex flex-col">
      {/* En-tête compact (type 6) : sous-titre Solo/Escouade SUR la même ligne que
          le chevron haut (navigation session plus récente), filet juste en dessous. */}
      <div className="mb-1 space-y-1">
        <div className="relative flex w-full items-center justify-center">
          <span className="absolute left-0 text-3xs font-semibold uppercase tracking-label-md text-foreground/90">
            {variantLabel}
          </span>
        <button
          type="button"
          disabled={displayIdx === 0 || isAnimating}
          onClick={() => handleChange(displayIdx - 1)}
          className={chevronBottomCls}
          aria-label={t('common.home.newer_session_aria')}
        >
          <ChevronUpIcon />
        </button>
        </div>
        <div className="h-px w-full rounded-full bg-border" />
      </div>

      {/* Contenu animé */}
      <div ref={contentRef}>
        {session ? (
          <button
            type="button"
            className={`${cardClass} w-full cursor-pointer text-left border border-transparent transition-colors hover:border-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring`}
            onClick={() => onNavigate(session.session_label, session.teammates ?? [])}
            aria-label={th('home.sessions.detail_aria', { label: session.session_label })}
          >
            {/* Scores de performance */}
            <div className="mb-2 flex items-baseline gap-3">
              {session.avg_team_performance != null && (
                <>
                  <span
                    className="text-2xl font-black leading-none"
                    style={{ color: getPerfColor(session.avg_team_performance) }}
                  >
                    {Math.round(session.avg_team_performance)}
                  </span>
                  <span className="text-xs text-muted-foreground">{th('home.sessions.team_label')}</span>
                </>
              )}
              {session.avg_player_performance != null && (
                <>
                  <span
                    className="text-2xl font-black leading-none"
                    style={{ color: getPerfColor(session.avg_player_performance) }}
                  >
                    {Math.round(session.avg_player_performance)}
                  </span>
                  <span className="text-xs text-muted-foreground">{th('home.sessions.player_label')}</span>
                </>
              )}
            </div>

            {/* Barre d'outcomes */}
            <OutcomeBar
              wins={session.wins}
              draws={session.draws}
              losses={session.losses}
              dnfs={session.dnfs}
            />

            {/* Décompte des outcomes avec nombre de matchs */}
            <p className="mt-1.5 flex flex-wrap gap-x-2 text-xs">
              <span className="font-medium text-foreground">
                {th('home.sessions.match_count', { n: session.match_count })}
              </span>
              {session.wins > 0 && (
                <span style={{ color: tokenCssVar('outcome-win') }}>
                  {th('home.sessions.wins_count', { n: session.wins })}
                </span>
              )}
              {session.losses > 0 && (
                <span style={{ color: tokenCssVar('outcome-loss') }}>
                  {th('home.sessions.losses_count', { n: session.losses })}
                </span>
              )}
              {session.draws > 0 && (
                <span style={{ color: tokenCssVar('outcome-draw') }}>
                  {th('home.sessions.draws_count', { n: session.draws })}
                </span>
              )}
              {session.dnfs > 0 && (
                <span style={{ color: tokenCssVar('outcome-dnf') }}>
                  {th('home.sessions.dnfs_count', { n: session.dnfs })}
                </span>
              )}
            </p>

            {/* FDA moyen + playlist dominante + mode dominant */}
            <p className="mt-0.5 text-xs text-muted-foreground">
              {session.avg_kda != null && (
                <span>
                  {th('home.sessions.kda_label')}{' '}
                  <span style={{ color: tokenCssVar(kdaDivergentScale(session.avg_kda)) }}>
                    {session.avg_kda.toFixed(2)}
                  </span>
                </span>
              )}
              {(session.avg_kda != null && (session.dominant_playlist || session.dominant_mode)) && ' · '}
              {session.dominant_playlist && (
                <span className="font-bold text-foreground">{session.dominant_playlist}</span>
              )}
              {session.dominant_playlist && session.dominant_mode && ' · '}
              {session.dominant_mode && (
                <span className="font-bold text-foreground">{session.dominant_mode}</span>
              )}
              {(session.dominant_playlist || session.dominant_mode) && (
                <span className="ml-1 inline-flex">
                  <InfoTooltip
                    content={dominantTooltip}
                    iconClass="w-3 h-3"
                  />
                </span>
              )}
            </p>

            {/* Date de début + durée */}
            {session.started_at && (
              <p className="mt-0.5 text-xs text-muted-foreground">
                {formatSessionDate(session.started_at, locale)}
                {session.ended_at
                  ? ` · ${th('home.sessions.duration_prefix')}${locale === 'en' ? ': ' : ' : '}${formatSessionDuration(session.started_at, session.ended_at)}`
                  : null}
              </p>
            )}
          </button>
        ) : null}
      </div>

      {/* Chevron bas */}
      <button
        type="button"
        disabled={displayIdx >= total - 1 || isAnimating}
        onClick={() => handleChange(displayIdx + 1)}
        className={chevronBottomCls}
        aria-label={t('common.home.older_session_aria')}
      >
        <ChevronDownIcon />
      </button>
    </div>
  )
}
