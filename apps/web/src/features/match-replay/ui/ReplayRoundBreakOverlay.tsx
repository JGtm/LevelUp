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
 * ET « TRENTE SECONDES À CÔTÉ » N'ÉTAIT PAS UNE FIGURE DE STYLE (correctif du 2026-08-29) : la
 * bascule était datée au DÉBUT de la manche suivante, c'est-à-dire au premier point qu'on y
 * marque — 19 à 34 s après la fin de la manche annoncée, mesuré sur les quatre témoins
 * multi-manches. Le message paraissait donc par-dessus la manche suivante, déjà commencée. Il
 * paraît maintenant à la FIN de la manche, l'instant même qui remplit la pastille du bandeau
 * (`roundsLogic`, en-tête) — un message se déclenche sur son événement, pas sur son voisin.
 *
 * C'EST AUSSI LE POINT DE DÉCLENCHEMENT DU SON « manche terminée » (`roundOverSound.ts`, câblé
 * depuis le 2026-08-28) : la détection de bascule est partagée avec la piste sonore, donc le
 * recalage vaut pour les deux d'un seul coup — l'image et la voix ne peuvent pas diverger.
 */
import { useMemo } from 'react'

import { scoreTimelineOf } from '@/lib/replay/scoreTimeline'

import { msToFrames } from '../model/replayLogic'
import type { ReplayDocumentReady } from '../model/replayNormalize'
import { OVERLAY_STATUS_NEUTRAL } from './replayOverlayStyles'
import { activeRoundTransition, roundTransitions } from '../model/roundsLogic'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'

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
      {/* MÊME AFFICHAGE QUE LE STATUT DE FIN DE MATCH (retour utilisateur du 2026-08-28) : le
          bloc de `ReplayVictoryOverlay`, partagé par `replayOverlayStyles.ts` — SANS l'accent
          latéral gauche, que l'utilisateur a fait retirer de ce style. Version NEUTRE : une
          manche qui se termine n'est le verdict de personne, elle n'a pas de camp à porter. */}
      <p className={OVERLAY_STATUS_NEUTRAL}>{t.roundOverFmt(active.endedIndex)}</p>
    </div>
  )
}
