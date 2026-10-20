/**
 * useReplayViewpoint — l'état « par les yeux de qui regarde-t-on ? », côté React.
 *
 * POURQUOI IL EST SÉPARÉ DE `model/replayViewpoint.ts` : la RÈGLE (comment un xuid choisi se
 * résout en point de vue effectif) est pure et vit dans le modèle ; ce fichier ne porte que
 * l'ÉTAT et son geste. C'est aussi la convention du dépôt — un module qui exporte un hook
 * n'exporte que des hooks (react-refresh).
 *
 * LA SÉLECTION N'EST PAS PERSISTÉE (décision 8 du plan, 2026-09-06) : à chaque montage la page
 * repart du joueur dont on lit la fiche. Un point de vue retenu d'une visite à l'autre ferait
 * ouvrir un match sur les couleurs de quelqu'un d'autre sans que rien ne l'explique — ni
 * localStorage, ni URL, donc rien à purger et rien à migrer.
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

/**
 * useReplayViewpoint tient la sélection et la résout contre le tableau de score courant.
 *
 * Le tableau de score arrive de la vue match, qui peut n'être pas encore là au premier rendu :
 * `undefined` est donc une entrée nominale, et rend un point de vue `null` — les lecteurs se
 * taisent, comme ils le font déjà sans camp connu.
 */
export function useReplayViewpoint(
  scoreboard: readonly ViewpointRow[] | null | undefined,
): ReplayViewpoint {
  const [selected, setSelected] = useState<string | null>(null)
  const select = useCallback((xuid: string | null) => setSelected(xuid), [])
  const xuid = useMemo(() => resolveViewpoint(scoreboard, selected), [scoreboard, selected])
  return useMemo(() => ({ xuid, select }), [xuid, select])
}
