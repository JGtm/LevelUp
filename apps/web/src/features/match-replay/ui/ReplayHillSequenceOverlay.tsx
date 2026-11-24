/**
 * ReplayHillSequenceOverlay — LA SUITE DES COLLINES JOUÉES du Roi de la colline : « Collines A › B › C », la courante
 * marquée, et « Prochaine A » quand le cycle est établi.
 *
 * LA LECTURE EST `hillSequence.ts` : ce fichier place et écrit, il ne décide rien. Rien n'est rendu hors d'un match à
 * collines, ni avant la première colline. Les lettres sont celles que la carte écrit sur ses collines.
 *
 * UNE PASTILLE SUR LA CARTE, EN HAUT À GAUCHE, sœur des surcouches de la carte (écran de fin, message inter-manche,
 * compte à rebours de la bombe, au centre) : sobre, sans interrupteur, `pointer-events-none`. Elle ne montre que les
 * `SHOWN_MAX` dernières collines ; une suite plus longue commence par « … ».
 */
import { useMemo } from 'react'

import { hillActivationsOf, hillSequenceAt } from '../model/hillSequence'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'

/** Le nombre maximal de collines écrites : assez pour lire un cycle de trois et sa reprise, sans déborder du plan. */
const SHOWN_MAX = 8

interface Props {
  doc: Pick<ReplayDocumentReady, 'zoneStates'>
  /** Image de lecture courante, publiée par le canvas. */
  frame: number
  locale: ReplayLocale
}

export function ReplayHillSequenceOverlay({ doc, frame, locale }: Props) {
  const activations = useMemo(() => hillActivationsOf(doc), [doc])
  const reading = hillSequenceAt(activations, frame)
  if (!reading) return null
  const t = REPLAY_TEXT[locale].hillSequence
  const { played, currentIndex, next } = reading
  const first = Math.max(0, played.length - SHOWN_MAX)
  const shown = played.slice(first)
  const current = currentIndex >= 0 ? played[currentIndex].label : null
  return (
    <div
      role="status"
      aria-label={t.ariaFmt(played.map((a) => a.label).join(', '), current, next)}
      className="pointer-events-none absolute left-2 top-2 z-10"
    >
      <div className="flex items-baseline gap-1.5 rounded-md border border-border bg-card/90 px-2.5 py-1 text-xs shadow-md backdrop-blur-sm">
        <span className="uppercase tracking-wide text-muted-foreground">{t.title}</span>
        {first > 0 && <span className="text-muted-foreground">…</span>}
        {shown.map((a, i) => {
          const isCurrent = first + i === currentIndex
          return (
            <span key={`${a.zoneRef}-${a.t0}`} className="flex items-baseline gap-1.5">
              {(i > 0 || first > 0) && (
                <span aria-hidden="true" className="text-muted-foreground">
                  ›
                </span>
              )}
              <span
                aria-current={isCurrent ? 'step' : undefined}
                className={isCurrent ? 'font-semibold text-foreground' : 'text-muted-foreground'}
              >
                {a.label}
              </span>
            </span>
          )
        })}
        {next && <span className="text-muted-foreground">· {t.nextFmt(next)}</span>}
      </div>
    </div>
  )
}
