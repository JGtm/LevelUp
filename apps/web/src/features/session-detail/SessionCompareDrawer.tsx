/**
 * SessionCompareDrawer — panneau latéral pour la comparaison côte-à-côte.
 *
 * Slide-in depuis la droite (translateX) ; desktop ≥ xl : ~50vw, mobile : full
 * width avec backdrop. Affiche pour la session comparée les mêmes blocs que la
 * session active (summary + 4 charts) plus la table delta. Pas de couleur hex
 * directe ; tous les libellés viennent du manifest `session.toml`.
 *
 * Navigation : boutons "← précédente" / "suivante →" pour cycler dans
 * `available_sessions` (chronologique), bouton "Suggestion similaire" pour
 * réutiliser `suggested_compare`. Fermeture via × ou touche Escape.
 */
import { useEffect } from 'react'

import { Button } from '@/components/ui/button'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type {
  SessionCompareEntry,
  SessionCompareMetricRow,
  SessionCompareSuggestion,
  SessionDetailMatchRow,
} from '@/lib/api/types'

import { useSessionT } from './_shared'
import { SessionSummaryCard } from './SessionSummaryCard'
import { SessionKDATimeline } from './SessionKDATimeline'
import { SessionOutcomeTape } from './SessionOutcomeTape'
import { SessionKillsDonut } from './SessionKillsDonut'
import { SessionPerfTrend } from './SessionPerfTrend'
import { SessionCompareMetrics } from './SessionCompareMetrics'

interface Props {
  open: boolean
  onClose: () => void
  compareSession: SessionCompareEntry | null
  compareMatches: SessionDetailMatchRow[]
  compareMetrics: SessionCompareMetricRow[]
  suggestedCompare: SessionCompareSuggestion | null
  previousLabel: string | null
  nextLabel: string | null
  onSelectLabel: (label: string) => void
}

export function SessionCompareDrawer({
  open,
  onClose,
  compareSession,
  compareMatches,
  compareMetrics,
  suggestedCompare,
  previousLabel,
  nextLabel,
  onSelectLabel,
}: Props) {
  const t = useSessionT()

  // Fermeture clavier Escape (pattern aligné sur AssetDrawer).
  useEffect(() => {
    if (!open) return
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  const title = compareSession
    ? t('session.detail.drawer_title', { label: compareSession.session_label })
    : t('session.detail.session_compared')

  return (
    <>
      {/* Backdrop mobile uniquement — sur desktop la page reste exploitable à gauche. */}
      {open && (
        <div
          className="fixed inset-0 z-[39] bg-background/40 backdrop-blur-sm xl:hidden"
          onClick={onClose}
          aria-hidden="true"
        />
      )}

      <aside
        role="complementary"
        aria-label={title}
        aria-hidden={!open}
        className={[
          'fixed right-0 top-0 z-40 flex h-screen w-full flex-col border-l border-border bg-background shadow-2xl transition-transform duration-200 ease-out',
          'xl:w-[50vw]',
          open ? 'translate-x-0' : 'translate-x-full',
        ].join(' ')}
      >
        {/* Header : titre + nav + close */}
        <header className="flex shrink-0 flex-col gap-3 border-b border-border bg-card px-4 py-3">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-sm font-semibold text-foreground">{title}</h2>
            <button
              type="button"
              onClick={onClose}
              aria-label={t('session.detail.drawer_close_aria')}
              className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
            >
              <CloseIcon />
            </button>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={!previousLabel}
              onClick={() => previousLabel && onSelectLabel(previousLabel)}
            >
              {t('session.detail.drawer_prev_session')}
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={!nextLabel}
              onClick={() => nextLabel && onSelectLabel(nextLabel)}
            >
              {t('session.detail.drawer_next_session')}
            </Button>
            {suggestedCompare && (
              <Button
                size="sm"
                variant="secondary"
                onClick={() => onSelectLabel(suggestedCompare.session_label)}
              >
                {t('session.detail.drawer_use_suggested')}
              </Button>
            )}
          </div>
        </header>

        {/* Body scrollable */}
        <div className="flex-1 overflow-y-auto p-4">
          {compareSession ? (
            <div className="space-y-4">
              <SessionSummaryCard
                title={t('session.detail.session_compared')}
                entry={compareSession}
                tone="compare"
              />

              <SessionOutcomeTape matches={compareMatches} />

              <div className="grid gap-4 2xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
                <SessionKDATimeline
                  title={t('session.detail.chart_kda_title')}
                  matches={compareMatches}
                />
                <SessionKillsDonut
                  title={t('session.detail.chart_kills_donut_title')}
                  matches={compareMatches}
                />
              </div>

              <SessionPerfTrend
                title={t('session.detail.chart_perf_title')}
                matches={compareMatches}
              />

              <SessionCompareMetrics metrics={compareMetrics} />
            </div>
          ) : (
            <EmptyStateNotice
              title={t('session.detail.no_compare_title')}
              description={t('session.detail.drawer_no_compare')}
            />
          )}
        </div>
      </aside>
    </>
  )
}

function CloseIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  )
}
