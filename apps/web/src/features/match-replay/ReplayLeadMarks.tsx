/**
 * ReplayLeadMarks — LES RETOURNEMENTS, posés sur la frise de lecture.
 *
 * CE QUE LA FRISE NE DISAIT PAS. Elle donne la position dans le temps, et rien d'autre : un
 * match serré et une démonstration s'y ressemblent trait pour trait. Le calque de score
 * (schéma 12) permet enfin de marquer les instants où le match CHANGE DE MENEUR — la seule
 * chose qu'on cherche quand on fait défiler un rejeu qu'on n'a pas vécu. Trois marques sur
 * l'Oddball témoin, aucune sur le Slayer (le vainqueur mène de bout en bout) : la frise
 * raconte alors deux histoires différentes sans qu'on ait rien à lire.
 *
 * DISCRÈTES PAR CONSTRUCTION : un trait de 2 px sur toute la hauteur de la piste, à
 * l'opacité d'un repère. Elles ne captent pas le pointeur (`pointer-events-none`) — la
 * frise reste saisissable au pixel près, y compris SOUS une marque.
 *
 * LA COULEUR EST CELLE DU NOUVEAU MENEUR, dans la grammaire de la page : `team-ally` /
 * `team-enemy`, les mêmes tokens que les points de la carte et les titres de colonnes. Un
 * camp non reconnu prend l'encre neutre du thème — jamais l'une des deux par défaut.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { LeadChange } from './scoreTimelineLogic'

/**
 * Largeur supposée du curseur natif, en pixels. La piste d'un `input[type=range]` court de
 * `THUMB_PX / 2` à `largeur − THUMB_PX / 2` : sans cette marge, une marque au tout début ou
 * à la toute fin se poserait à côté du curseur qui la désigne. Le navigateur ne publie pas
 * cette valeur — 16 px est celle des thèmes par défaut, et l'écart résiduel se compte en
 * pixels sur une frise qui en fait plusieurs centaines.
 */
const THUMB_PX = 16

export interface ReplayLeadMarksProps {
  changes: readonly LeadChange[]
  /** Nombre d'images du document : c'est lui qui donne l'échelle de la frise. */
  frameCount: number
  /** Durée d'une image, pour dater la marque en mm:ss dans son infobulle. */
  frameIntervalMs?: number
  /** Camp du meneur, du point de vue du joueur de la page (`null` = inconnu). */
  allyOf: (teamId: number) => boolean | null
  /** Libellé de l'équipe qui passe devant, tel que la colonne l'écrit. */
  labelOf: (teamId: number) => string
  locale: ReplayLocale
}

export function ReplayLeadMarks({
  changes,
  frameCount,
  frameIntervalMs,
  allyOf,
  labelOf,
  locale,
}: ReplayLeadMarksProps) {
  const t = REPLAY_TEXT[locale]
  const span = frameCount - 1
  if (changes.length === 0 || span <= 0) return null
  return (
    <span className="pointer-events-none absolute inset-0 block">
      {changes.map((c) => {
        const ally = allyOf(c.teamId)
        const color = ally === null ? 'var(--border)' : tokenCssVar(ally ? 'team-ally' : 'team-enemy')
        const ratio = Math.min(1, Math.max(0, c.frame / span))
        return (
          <span
            key={`${c.frame}-${c.teamId}`}
            className="absolute top-1/2 block h-2.5 w-0.5 -translate-y-1/2 rounded-full opacity-80"
            style={{
              left: `calc(${THUMB_PX / 2}px + (100% - ${THUMB_PX}px) * ${ratio})`,
              background: color,
            }}
            title={t.leadChangeAtFmt(formatClock(c.frame, frameIntervalMs), labelOf(c.teamId))}
            aria-label={t.leadChange}
          />
        )
      })}
    </span>
  )
}

/**
 * formatClock date une image en mm:ss depuis le début du rejeu. Sans échelle temporelle
 * (artefact sans `frameIntervalMs`), l'axe T n'est qu'un index : on rend le numéro d'image
 * plutôt qu'une durée fabriquée.
 */
function formatClock(frame: number, frameIntervalMs?: number): string {
  if (!frameIntervalMs) return `#${Math.round(frame)}`
  const total = Math.round((frame * frameIntervalMs) / 1000)
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}
