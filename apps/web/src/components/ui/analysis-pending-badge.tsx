/**
 * AnalysisPendingBadge — la pastille « Analyse en cours » ou « Analyse en attente » d'une tuile
 * de match, en overlay en haut à droite de l'image de carte, et son infobulle.
 *
 * Le match est synchronisé, le film n'est pas encore analysé (`RecentMatchItem.analysis_pending`) :
 * la tuile le dit au lieu d'afficher des trous, et nomme les seules parties en attente.
 * « En cours » seulement si un décodage porte sur ce match (`analysis_state`), sinon « en
 * attente ». La pastille est un `<button>` accessible (focus clavier) décrit par l'infobulle
 * (`aria-describedby`) ; l'infobulle s'ouvre au survol ET au focus, se ferme à la sortie, au
 * flou, à Échap, au défilement.
 *
 * L'infobulle est rendue dans `document.body` (portail, `position: fixed`) : l'image de la
 * tuile et la tuile elle-même sont `overflow-hidden`, un panneau en place serait rogné. Elle
 * reste dans le DOM, masquée par `hidden`, pour que `aria-describedby` désigne toujours un
 * élément existant.
 *
 * Couleurs : jeton sémantique `info` (point, bordure, texte) sur fond sombre translucide,
 * comme le bouton favori voisin. Le point pulse seulement hors `prefers-reduced-motion`
 * (`motion-safe:`).
 */
import { useEffect, useId, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import type { Locale } from '@/lib/i18n/locale'
import { analysisPendingLabelKey } from './match-card-presentation'

const PANEL_WIDTH = 256 // px
const GAP = 8 // px — marge pastille/panneau et marge de viewport

interface PanelPos {
  top: number
  left: number
}

export function AnalysisPendingBadge({ locale, inProgress, parts }: { locale: Locale; inProgress: boolean; parts: readonly CommonManifestKey[] }) {
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const tooltipId = useId()
  const [open, setOpen] = useState(false)
  const [pos, setPos] = useState<PanelPos | null>(null)
  const anchorRef = useRef<HTMLButtonElement | null>(null)
  const panelRef = useRef<HTMLDivElement | null>(null)

  // Sous la pastille, bord droit aligné, replié au-dessus s'il n'y a pas la place ; borné
  // dans la viewport.
  useLayoutEffect(() => {
    if (!open || !anchorRef.current) return
    const a = anchorRef.current.getBoundingClientRect()
    const panelHeight = panelRef.current?.getBoundingClientRect().height ?? 0
    const left = Math.max(GAP, Math.min(a.right - PANEL_WIDTH, window.innerWidth - PANEL_WIDTH - GAP))
    const below = a.bottom + GAP
    const top = below + panelHeight > window.innerHeight - GAP ? Math.max(GAP, a.top - panelHeight - GAP) : below
    setPos({ top, left })
  }, [open])

  useEffect(() => {
    if (!open) return
    const close = () => {
      setOpen(false)
      setPos(null)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close()
    }
    window.addEventListener('scroll', close, true)
    window.addEventListener('resize', close)
    document.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('scroll', close, true)
      window.removeEventListener('resize', close)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const show = () => setOpen(true)
  const hide = () => {
    setOpen(false)
    setPos(null)
  }

  return (
    <span className="absolute top-2 right-2 z-10" data-testid="match-card-analysis-pending-badge">
      <button
        ref={anchorRef}
        type="button"
        aria-describedby={tooltipId}
        onMouseEnter={show}
        onMouseLeave={hide}
        onFocus={show}
        onBlur={hide}
        className="inline-flex cursor-help items-center gap-1.5 rounded-full border border-info/60 bg-black/55 px-2 py-1 text-2xs font-semibold leading-none text-info focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-info"
      >
        <span aria-hidden="true" className="relative flex h-2 w-2">
          <span className="absolute inline-flex h-full w-full rounded-full bg-info opacity-75 motion-safe:animate-ping" />
          <span className="relative inline-flex h-2 w-2 rounded-full bg-info" />
        </span>
        {t(analysisPendingLabelKey('badge', inProgress))}
      </button>
      {createPortal(
        <div
          id={tooltipId}
          ref={panelRef}
          role="tooltip"
          hidden={!open}
          style={{
            position: 'fixed',
            top: pos?.top ?? -9999,
            left: pos?.left ?? -9999,
            width: PANEL_WIDTH,
            visibility: pos ? 'visible' : 'hidden',
          }}
          className="z-[9999] rounded-lg border border-border bg-background p-3 text-xs font-normal normal-case tracking-normal text-foreground shadow-lg"
        >
          <p className="font-semibold text-foreground">{t(analysisPendingLabelKey('tooltip_title', inProgress))}</p>
          {parts.length > 0 && (
            <p className="mt-1.5 text-muted-foreground" data-testid="match-card-analysis-pending-parts">
              {parts.map((key) => t(key)).join(' · ')}
            </p>
          )}
          <p className="mt-1.5 italic text-muted-foreground">{t('common.match_card.analysis_pending.tooltip_note')}</p>
        </div>,
        document.body,
      )}
    </span>
  )
}
