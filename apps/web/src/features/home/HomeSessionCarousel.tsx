/**
 * HomeSessionCarousel — carrousel vertical des sessions récentes (solo/escouade).
 *
 * P8.4 (revue 2026-04-29) : extrait de HomePage.tsx (SessionCarouselCard +
 * ChevronUp/Down icons + helpers formatSession*). Réduit la god page de ~245L.
 */
import { useEffect, useRef, useState } from 'react'
import type { SessionSummaryItem } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'
import { kdScale } from '@/lib/accessibility/scales'
import { getPerfColor } from '@/lib/perf-color'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { OutcomeBar } from '@/components/ui/outcome-bar'

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

function formatSessionDate(startedAt: string | null): string {
  if (!startedAt) return ''
  const d = new Date(startedAt)
  return (
    d.toLocaleDateString('fr-FR', { day: 'numeric', month: 'short' }) +
    ' à ' +
    d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
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
  onNavigate: (sessionLabel: string) => void
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
    el.style.transition = 'transform 0.1s ease-in, opacity 0.1s ease-in'
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
          el.style.transition = 'transform 0.15s ease-out, opacity 0.12s ease-out'
          el.style.transform = 'translateY(0)'
          el.style.opacity = '1'
          const t2 = setTimeout(() => setIsAnimating(false), 160)
          cleanupRef.current = () => clearTimeout(t2)
        })
      })
      cleanupRef.current = () => cancelAnimationFrame(rafId)
    }, 110)

    cleanupRef.current = () => clearTimeout(t1)
  }

  const session = sessions[displayIdx]
  const total = sessions.length
  const variantLabel = variant === 'solo' ? 'Solo' : 'Escouade'
  const cardClass = variant === 'solo' ? 'rounded-md bg-muted p-3' : 'rounded-md bg-primary/10 p-3'
  const chevronBottomCls =
    'flex w-full cursor-pointer items-center justify-center py-2 text-muted-foreground/40 transition-colors hover:text-muted-foreground focus-visible:outline-none focus-visible:text-foreground disabled:cursor-not-allowed disabled:opacity-20'

  return (
    <div className="flex flex-col">
      {/* Ligne supérieure : label variant (absolu à gauche) + chevron haut centré */}
      <div className="relative mb-1 flex w-full items-center justify-center">
        <span className="absolute left-0 text-xs font-semibold text-muted-foreground">{variantLabel}</span>
        <button
          type="button"
          disabled={displayIdx === 0 || isAnimating}
          onClick={() => handleChange(displayIdx - 1)}
          className={chevronBottomCls}
          aria-label="Session plus récente"
        >
          <ChevronUpIcon />
        </button>
      </div>

      {/* Contenu animé */}
      <div ref={contentRef}>
        {session ? (
          <button
            type="button"
            className={`${cardClass} w-full cursor-pointer text-left hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring`}
            onClick={() => onNavigate(session.session_label)}
            aria-label={`Voir le détail de la session ${session.session_label}`}
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
                  <span className="text-xs text-muted-foreground">Équipe</span>
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
                  <span className="text-xs text-muted-foreground">Perso</span>
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

            {/* Décompte des outcomes avec nombre de parties */}
            <p className="mt-1.5 flex flex-wrap gap-x-2 text-xs">
              <span className="font-medium text-foreground">
                {session.match_count} partie{session.match_count > 1 ? 's' : ''}
              </span>
              {session.wins > 0 && (
                <span style={{ color: tokenCssVar('outcome-win') }}>
                  {session.wins} Victoire{session.wins > 1 ? 's' : ''}
                </span>
              )}
              {session.losses > 0 && (
                <span style={{ color: tokenCssVar('outcome-loss') }}>
                  {session.losses} Défaite{session.losses > 1 ? 's' : ''}
                </span>
              )}
              {session.draws > 0 && (
                <span style={{ color: tokenCssVar('outcome-draw') }}>
                  {session.draws} Égalité{session.draws > 1 ? 's' : ''}
                </span>
              )}
              {session.dnfs > 0 && (
                <span style={{ color: tokenCssVar('outcome-dnf') }}>
                  {session.dnfs} Non terminé{session.dnfs > 1 ? 's' : ''}
                </span>
              )}
            </p>

            {/* FDA moyen + playlist dominante + mode dominant */}
            <p className="mt-0.5 text-xs text-muted-foreground">
              {session.avg_kda != null && (
                <span>
                  FDA{' '}
                  <span style={{ color: tokenCssVar(kdScale(session.avg_kda)) }}>
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
                    content="Playlist et mode les plus joués lors de cette session"
                    iconClass="w-3 h-3"
                  />
                </span>
              )}
            </p>

            {/* Date de début + durée */}
            {session.started_at && (
              <p className="mt-0.5 text-xs text-muted-foreground">
                {formatSessionDate(session.started_at)}
                {session.ended_at
                  ? ` · Durée de la session : ${formatSessionDuration(session.started_at, session.ended_at)}`
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
        aria-label="Session plus ancienne"
      >
        <ChevronDownIcon />
      </button>
    </div>
  )
}
