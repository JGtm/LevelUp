/**
 * useReplayViewpoint — l'état « par les yeux de qui regarde-t-on ? », côté React.
 *
 * POURQUOI IL EST SÉPARÉ DE `model/replayViewpoint.ts` : la RÈGLE (comment un xuid choisi se
 * résout en point de vue effectif) est pure et vit dans le modèle ; ce fichier ne porte que
 * l'ÉTAT et son geste. C'est aussi la convention du dépôt — un module qui exporte un hook
 * n'exporte que des hooks (react-refresh).
 *
 * LA SÉLECTION N'EST PAS PERSISTÉE : la page repart du joueur dont on lit la fiche à chaque montage
 * ET à chaque match. La route garde le même composant quand seul `$matchId` change : l'état est donc
 * rangé AVEC sa portée (`scopeKey`, le match) et REMIS À ZÉRO au rendu dès qu'elle diffère, même quand
 * le même joueur figure au tableau du match suivant, et sans ressusciter le choix en revenant au match
 * d'avant. Un point de vue retenu d'un match ou d'une visite à l'autre ferait ouvrir un match sur les
 * couleurs de quelqu'un d'autre sans que rien ne l'explique — ni localStorage, ni URL, donc rien à
 * purger et rien à migrer.
 *
 * ELLE NE TOUCHE PAS AU TEMPS (décision 1) : ce hook ne connaît ni la lecture, ni le curseur.
 * Changer de joueur change ce qu'on voit, jamais où l'on en est.
 */
import { useCallback, useMemo, useState } from 'react'

import { resolveViewpoint, type ViewpointRow } from '../model/replayViewpoint'

export interface ReplayViewpoint {
  /** Le point de vue EFFECTIF : le joueur choisi, à défaut celui de la page, sinon `null`. */
  xuid: string | null
  /** Choisir un joueur ; `null` revient au joueur de la page. Le menu du lot L3 s'y branche. */
  select: (xuid: string | null) => void
}

/** Le choix et la portée où il a été posé : hors de sa portée, il ne vaut rien. */
interface Scoped {
  scopeKey: string
  selected: string | null
}

/**
 * useReplayViewpoint tient la sélection et la résout contre le tableau de score courant.
 *
 * @param scopeKey la portée du choix : le match. Une autre portée repart du joueur de la page.
 *
 * Le tableau de score arrive de la vue match, qui peut n'être pas encore là au premier rendu :
 * `undefined` est donc une entrée nominale, et rend un point de vue `null` — les lecteurs se
 * taisent, comme ils le font déjà sans camp connu.
 */
export function useReplayViewpoint(
  scopeKey: string,
  scoreboard: readonly ViewpointRow[] | null | undefined,
): ReplayViewpoint {
  const [stored, setStored] = useState<Scoped>({ scopeKey, selected: null })
  // L'effacement se fait AU RENDU, pas dans un effet : aucune image ne montre le choix d'un autre match.
  if (stored.scopeKey !== scopeKey) setStored({ scopeKey, selected: null })
  const selected = stored.scopeKey === scopeKey ? stored.selected : null
  const select = useCallback((xuid: string | null) => setStored({ scopeKey, selected: xuid }), [scopeKey])
  const xuid = useMemo(() => resolveViewpoint(scoreboard, selected), [scoreboard, selected])
  return useMemo(() => ({ xuid, select }), [xuid, select])
}
