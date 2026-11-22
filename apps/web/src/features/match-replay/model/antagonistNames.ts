/**
 * antagonistNames — LE NOM AFFICHÉ D'UN JOUEUR DÉSIGNÉ PAR SA CLÉ DU FILM, pour les surfaces des
 * antagonistes (bande de la frise, section de la fiche).
 *
 * C'est la fonction de nom du reste du rejeu (`cardName` : la base d'abord, le film ensuite, le
 * suffixe « [bot] » retiré, le repli « joueur inconnu ») ; ce fichier ne fait que la poser sur une
 * clé. Les clés des antagonistes sont celles du film (`ReplayPlayer.xuid`, `bot:<nom>` pour un bot),
 * rendues par `replayAntagonists` : aucun rapprochement ici.
 */
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayText } from '../i18n/i18nContract'
import { cardName } from './playerCardReadings'

/** Le nom affiché du joueur de cette clé du film ; le repli « joueur inconnu » quand le roster ne le connaît pas. */
export function antagonistNameOf(
  players: readonly ReplayPlayer[],
  t: ReplayText,
): (key: string) => string {
  const byKey = new Map(players.map((p) => [p.xuid, p]))
  return (key) => {
    const player = byKey.get(key)
    return player ? cardName(player, t) : t.unknownPlayer
  }
}
