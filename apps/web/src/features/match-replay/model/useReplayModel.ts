/**
 * useReplayModel — le modèle de la page de rejeu, mémoïsé.
 *
 * CE HOOK NE CALCULE RIEN. Toute la jointure vit dans `buildReplayModel`, pure et testée sans
 * React ; ce fichier n'existe que pour la mémoïser. C'est la frontière : au-dessus, du React ;
 * en dessous, des fonctions qu'on peut lire, exécuter et prouver sans monter un routeur.
 *
 * UNE SEULE MÉMO POUR TOUTE LA JOINTURE, et c'est un choix mesuré. Trois des quatre entrées
 * sont des résultats de requêtes TanStack Query : leur identité ne change qu'au chargement ou à
 * un rafraîchissement, jamais au fil de la lecture. La quatrième, le POINT DE VUE (2026-09-06),
 * est un état d'interface qui ne change que sur un GESTE de l'utilisateur — choisir un autre
 * joueur — donc jamais pendant qu'on regarde. La page se re-rend ~6,7 fois par seconde pendant
 * le rejeu (publication de l'image toutes les 150 ms) : aucun de ces re-rendus ne refait le
 * calcul, exactement comme la douzaine de mémos qu'il remplace.
 *
 * CE QUE COÛTE UNE BASCULE DE POINT DE VUE, MESURÉ AVANT D'OPTIMISER (E2.3 du plan « frise,
 * point de vue », 2026-09-06) : sur le match témoin `4ecdf3e7` — artefact réel de 2 714 images,
 * 9 joueurs, 38 vies, 90 kills — la reconstruction COMPLÈTE du modèle prend une MÉDIANE de
 * 0,16 ms sur 20 bascules (min 0,14, max 0,25 — `replayModel.bench.test.ts`). Trois cents fois
 * sous le seuil de 50 ms au-delà duquel le plan prévoyait de découper la mémo : elle reste
 * ENTIÈRE. Découper coûterait deux chemins de dépendances à tenir pour un gain invisible.
 */
import { useMemo } from 'react'

import type { MatchViewResponse } from '@/lib/api/types'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { buildReplayModel, type ReplayModel, type ReplayModelSettings } from './replayModel'

export function useReplayModel(
  doc: ReplayDocumentReady | null | undefined,
  matchView: MatchViewResponse | null | undefined,
  settings?: ReplayModelSettings | null,
  viewpoint?: string | null,
): ReplayModel {
  return useMemo(
    () => buildReplayModel(doc, matchView, settings, viewpoint),
    [doc, matchView, settings, viewpoint],
  )
}
