/**
 * ReplayBombCountdownOverlay — LE COMPTE À REBOURS DE LA BOMBE : « Bombe armée — 4,9 s » et
 * une barre de mèche qui se vide, pendant les 4,93 s qui séparent l'armement de l'explosion.
 *
 * DÉRIVÉ DE LA POSITION DE LECTURE, PAS D'UN ÉTAT — la doctrine de `ReplayRoundBreakOverlay` :
 * aucun minuteur mural, le bandeau est visible tant que l'image lue tombe dans la fenêtre
 * [armé, armé + mèche] et disparaît dès qu'on la quitte. On remonte la frise, il se rejoue ;
 * on s'arrête dessus, il montre le temps restant À CETTE IMAGE.
 *
 * UN BANDEAU, PAS UN POINT SUR LA CARTE, et c'est une propriété de la mesure (cf.
 * `bombCountdown.ts`) : le canal ne dit ni qui arme ni où — poser le compte à rebours sur un
 * lieu deviné serait exactement ce que la déflagration refuse déjà. Il se place EN HAUT du
 * terrain, discret, sous le bandeau de score ; l'explosion, elle, éclatera sur la carte au
 * lieu RELU de son auteur (`bombBlastFx`).
 *
 * IL NE BLOQUE RIEN (`pointer-events-none`) et n'a aucun élément interactif — même règle que
 * les autres overlays. SOBRE : petit corps de texte, la barre fait le travail ; l'accent est
 * le token `destructive` — une mèche qui brûle est la seule information de danger de la page,
 * et aucune couleur d'équipe n'est affirmée (le canal ne connaît pas le camp de l'armeur).
 */
import { activeBombCountdown } from '../model/bombCountdown'
import { formatSeconds } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'

interface Props {
  /** Le document du rejeu — armements par le calque, cadence par `frameIntervalMs`. */
  doc: ReplayDocumentReady
  /** Image de lecture courante, publiée par le canvas. */
  frame: number
  locale: ReplayLocale
}

export function ReplayBombCountdownOverlay({ doc, frame, locale }: Props) {
  const state = activeBombCountdown(doc, frame)
  if (!state) return null
  const t = REPLAY_TEXT[locale]
  return (
    <div
      role="status"
      aria-live="polite"
      className="pointer-events-none absolute inset-x-0 top-2 z-10 flex justify-center"
    >
      <div className="rounded-md border border-border bg-card/90 px-3 py-1.5 shadow-md backdrop-blur-sm">
        <p className="text-center text-xs font-semibold uppercase tracking-wide text-foreground">
          {t.bombArmedFmt(formatSeconds(state.remainingMs))}
        </p>
        {/* La barre de mèche : pleine à l'armement, vide à l'explosion. Piste en `border`
            (neutre), course en `destructive` — la seule encre de danger de la page. */}
        <div className="mt-1 h-1 w-40 overflow-hidden rounded-full bg-border">
          <div
            className="h-full rounded-full bg-destructive"
            style={{ width: `${Math.round((1 - state.progress) * 100)}%` }}
          />
        </div>
      </div>
    </div>
  )
}
