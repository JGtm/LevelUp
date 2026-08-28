/**
 * ReplayRoundBreakOverlay — LE MESSAGE INTER-MANCHE : « Manche N terminée », bref, au passage
 * d'une manche à la suivante (Oddball et tout mode multi-manche).
 *
 * DÉRIVÉ DE LA POSITION DE LECTURE, PAS D'UN ÉTAT — exactement la doctrine de l'écran de fin
 * (`ReplayVictoryOverlay`). Il ne pilote aucun minuteur : il est visible tant que l'image lue
 * tombe dans une COURTE FENÊTRE après une bascule de manche, et invisible ailleurs. Un minuteur
 * mural se battrait avec la frise — on remonte de trois secondes, le message aurait disparu ; on
 * s'arrête dessus, il resterait éternellement. La fenêtre en frames, elle, se rejoue si l'on
 * repasse sur la bascule et se ferme dès qu'on la quitte, sans rien accumuler.
 *
 * IL NE BLOQUE RIEN (`pointer-events-none`) : la frise et le terrain sont dessous, et un voile
 * qui capterait les clics enfermerait l'utilisateur. Il n'a aucun élément interactif — rien ne se
 * perd à le rendre transparent au pointeur.
 *
 * LES BORNES DE MANCHE VIENNENT DU CALQUE DE SCORE, PAR LA MÊME GARDE D'HORLOGE que le bandeau
 * (`scoreTimelineOf`) : un film dont l'origine n'est pas résolue ne date pas ses bascules, et un
 * message qui paraîtrait trente secondes à côté serait pire qu'aucun message. Sur un mode à manche
 * unique, `roundTransitions` est vide : rien ne se rend jamais.
 *
 * C'EST AUSSI LE POINT DE DÉCLENCHEMENT DU SON « manche terminée ». La détection de bascule
 * (`roundTransitions`) est partagée avec la piste sonore : le jour où l'asset est fourni, le son
 * se branche sur la même mesure (cf. `replaySound.ts`, bloc des sons d'objectif).
 */
import { useMemo } from 'react'

import { scoreTimelineOf } from '@/lib/replay/scoreTimeline'

import { msToFrames } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { activeRoundTransition, roundTransitions } from './roundsLogic'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'

/**
 * LA DURÉE D'AFFICHAGE, en millisecondes de MATCH (convertie en frames par le film) : assez
 * pour se lire, assez brève pour ne pas mordre sur la manche suivante. Une propriété du message,
 * nommée ici et testée par le biais de `activeRoundTransition`.
 */
export const ROUND_BREAK_WINDOW_MS = 3000

interface Props {
  /** Le document du rejeu — bornes de manche par le calque, cadence par `frameIntervalMs`. */
  doc: ReplayDocumentReady
  /** Image de lecture courante, publiée par le canvas. */
  frame: number
  locale: ReplayLocale
}

export function ReplayRoundBreakOverlay({ doc, frame, locale }: Props) {
  const t = REPLAY_TEXT[locale]
  const transitions = useMemo(() => roundTransitions(scoreTimelineOf(doc)), [doc])
  const windowFrames = useMemo(
    () => Math.max(1, Math.round(msToFrames(ROUND_BREAK_WINDOW_MS, doc))),
    [doc],
  )
  const active = activeRoundTransition(transitions, frame, windowFrames)
  if (!active) return null
  return (
    <div
      role="status"
      aria-live="polite"
      className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center overflow-hidden"
    >
      <p className="rounded-md border border-border bg-background/85 px-5 py-2 text-lg font-bold uppercase tracking-wide text-foreground shadow-lg backdrop-blur-sm">
        {t.roundOverFmt(active.endedIndex)}
      </p>
    </div>
  )
}
