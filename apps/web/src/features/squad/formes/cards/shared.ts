/**
 * shared.ts — L'ASSEMBLAGE QUE PLUSIEURS CARTES PARTAGENT : les colonnes de la bande (un
 * match = une heure et une carte).
 *
 * La piste du lobby (escouade + reste du camp + adversaire) et les segments de l'escouade
 * seule vivaient ici aussi ; leurs cartes (contexte Escouade) ont été retirées avec l'ancien
 * onglet Usages (lot L5.4 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26).
 */
import type { SquadFormesMatch } from '@/lib/api/types'

import type { FormesViewModel } from '../viewModel'

/**
 * Les colonnes de la bande : un match, son heure, sa carte. Elle reçoit LA MÊME
 * liste que les cases — une colonne de plus que de cases décalerait toute la
 * frise.
 */
export function matchColumns(
  vm: FormesViewModel,
  matches: SquadFormesMatch[],
): { key: string; time: string; map: string }[] {
  return matches.map((m) => ({
    key: m.match_id,
    time: vm.matchTime(m),
    map: vm.matchMap(m),
  }))
}
