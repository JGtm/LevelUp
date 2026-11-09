/**
 * Notice — bandeau de message de page, un seul gabarit pour les trois tons :
 *  - `error`   : une lecture a échoué (destructive) ;
 *  - `warning` : des données manquent (warning) ;
 *  - `info`    : rien n'est cassé, la page évolue (info).
 *
 * Une icône par ton (plus de SVG recopiés par bandeau), `role="status"` pour que le lecteur
 * d'écran annonce sans interrompre. Jetons sémantiques uniquement. `aside` porte la mention
 * alignée à droite (cf. `NoticeLiveStatus`).
 */
import type { ReactNode } from 'react'

export type NoticeTone = 'error' | 'warning' | 'info'

const TONE_CLASS: Record<NoticeTone, { box: string; icon: string }> = {
  error: { box: 'border-destructive/50 bg-destructive/10', icon: 'text-destructive' },
  warning: { box: 'border-warning/50 bg-warning/10', icon: 'text-warning' },
  info: { box: 'border-info/50 bg-info/10', icon: 'text-info' },
}

/** Tracés heroicons « mini » 20x20 : cercle croix, triangle d'alerte, cercle « i ». */
const TONE_ICON_PATH: Record<NoticeTone, string> = {
  error:
    'M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z',
  warning:
    'M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z',
  info: 'M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z',
}

/**
 * Icône d'un ton de Notice, teintée par son jeton sémantique. Décorative (`aria-hidden`) :
 * l'appelant qui la pose seule porte le libellé accessible sur son propre conteneur.
 */
export function NoticeIcon({ tone, className = '' }: { tone: NoticeTone; className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 20 20"
      fill="currentColor"
      className={`${TONE_CLASS[tone].icon} ${className}`.trim()}
      aria-hidden="true"
    >
      <path fillRule="evenodd" d={TONE_ICON_PATH[tone]} clipRule="evenodd" />
    </svg>
  )
}

interface NoticeProps {
  tone: NoticeTone
  title: ReactNode
  /** Corps sous le titre (liste, phrase). */
  children?: ReactNode
  /** Mention alignée à droite. */
  aside?: ReactNode
  testId?: string
}

export function Notice({ tone, title, children, aside, testId }: NoticeProps) {
  const cls = TONE_CLASS[tone]
  return (
    <div
      role="status"
      data-testid={testId}
      data-tone={tone}
      className={`flex flex-wrap items-center gap-x-4 gap-y-2 rounded-md border px-4 py-3 ${cls.box}`}
    >
      <NoticeIcon tone={tone} className="h-4 w-4 shrink-0" />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-semibold text-foreground">{title}</p>
        {children ? <div className="mt-0.5 text-xs text-muted-foreground">{children}</div> : null}
      </div>
      {aside}
    </div>
  )
}

const LIVE_DOT_CLASS: Record<NoticeTone, { text: string; dot: string }> = {
  error: { text: 'text-destructive', dot: 'bg-destructive' },
  warning: { text: 'text-warning', dot: 'bg-warning' },
  info: { text: 'text-info', dot: 'bg-info' },
}

/**
 * Mention « se met à jour seule » : un point qui pulse (hors `prefers-reduced-motion`) et un
 * libellé. À ne poser que si la page se recharge réellement seule.
 */
export function NoticeLiveStatus({ tone, children }: { tone: NoticeTone; children: ReactNode }) {
  const cls = LIVE_DOT_CLASS[tone]
  return (
    <span className={`inline-flex shrink-0 items-center gap-2 text-xs font-medium ${cls.text}`}>
      <span aria-hidden="true" className="relative flex h-2 w-2">
        <span className={`absolute inline-flex h-full w-full rounded-full opacity-75 motion-safe:animate-ping ${cls.dot}`} />
        <span className={`relative inline-flex h-2 w-2 rounded-full ${cls.dot}`} />
      </span>
      {children}
    </span>
  )
}
