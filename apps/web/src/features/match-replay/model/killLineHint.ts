/**
 * killLineHint — L'INFOBULLE D'UNE LIGNE D'ÉLIMINATION : ce que la ligne ne montre pas faute de place.
 * L'assistance nommée (avec sa part de dégâts), la victime porteuse du drapeau et les éliminations liées
 * à l'objectif (`objectiveKills`) n'ont pas de colonne : elles se lisent ici, au survol. Une mort sans assistant dit pourquoi (distincte d'« inconnu », qui ne
 * dit rien). `focusable` ajoute en tête l'invite du clic. Rend `undefined` quand il n'y a rien à dire.
 */
import { displayPlayerName } from '@/lib/players/displayName'

import type { REPLAY_TEXT } from '../i18n/i18n'
import type { ReplayKill } from './killFeedLogic'
import type { ObjectiveFeedStat } from './objectiveFeedLogic'

type ReplayText = (typeof REPLAY_TEXT)['fr']

/** Le séparateur des morceaux d'une infobulle. */
const SEP = ' — '

export function killLineHint(
  kill: ReplayKill,
  carrier: boolean,
  t: ReplayText,
  focusable: boolean,
  objectiveKills: readonly ObjectiveFeedStat[] = [],
): string | undefined {
  const parts: string[] = []
  if (focusable) parts.push(t.killFeedFocusHint)
  if (kill.assistState === 'named') {
    parts.push(t.feedLines.assistedBy(displayPlayerName(kill.assistGamertag), kill.assistDamagePct ?? null), t.killFeedAssistHint)
  } else if (kill.assistState === 'none') {
    parts.push(
      kill.killerDamagePct != null
        ? `${t.killFeedNoAssistHint} ${t.killFeedKillerShare(kill.killerDamagePct)}.`
        : t.killFeedNoAssistHint,
    )
  }
  if (carrier) parts.push(t.feedLines.carrierBadgeHint)
  for (const stat of objectiveKills) parts.push(t.feedObjectiveAction[stat])
  return parts.length > 0 ? parts.join(SEP) : undefined
}
