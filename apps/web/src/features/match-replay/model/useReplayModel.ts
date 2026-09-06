/**
 * useReplayModel — le modèle de la page de rejeu, mémoïsé.
 *
 * CE HOOK NE CALCULE RIEN. Toute la jointure vit dans `buildReplayModel`, pure et testée sans
 * React ; ce fichier n'existe que pour la mémoïser. C'est la frontière : au-dessus, du React ;
 * en dessous, des fonctions qu'on peut lire, exécuter et prouver sans monter un routeur.
 *
 * UNE SEULE MÉMO POUR TOUTE LA JOINTURE, et c'est un choix mesuré. Les trois entrées sont des
 * résultats de requêtes TanStack Query : leur identité ne change qu'au chargement ou à un
 * rafraîchissement, jamais au fil de la lecture. La page se re-rend ~6,7 fois par seconde
 * pendant le rejeu (publication de l'image toutes les 150 ms) : aucun de ces re-rendus ne
 * refait le calcul, exactement comme la douzaine de mémos qu'il remplace.
 */
import { useMemo } from 'react'

import type { MatchViewResponse } from '@/lib/api/types'

import type { ReplayDocumentReady } from '../replayNormalize'
import { buildReplayModel, type ReplayModel, type ReplayModelSettings } from './replayModel'

export function useReplayModel(
  doc: ReplayDocumentReady | null | undefined,
  matchView: MatchViewResponse | null | undefined,
  settings?: ReplayModelSettings | null,
): ReplayModel {
  return useMemo(() => buildReplayModel(doc, matchView, settings), [doc, matchView, settings])
}
