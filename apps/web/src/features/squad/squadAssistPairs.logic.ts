/**
 * squadAssistPairs.logic — la projection des paires d'« Appui » (assistances entre joueurs de
 * l'escouade) vers le diagramme de flux partagé (`features/_shared/assists/assistFlow.logic.ts`).
 *
 * Aucune dépendance React/ECharts. Les deux colonnes suivent l'ORDRE DU ROSTER (joueur principal
 * d'abord, puis les coéquipiers — le même que tous les autres blocs de la page) ; un joueur hors
 * roster garde son nœud, à la suite : la mesure l'a vu, la page ne l'efface pas.
 *
 * LE DÉNOMINATEUR DES PARTS EST LE TOTAL SERVEUR (`total_assists`), pas la somme des rubans
 * visibles : les deux coïncident aujourd'hui, mais dériver le dénominateur de l'affichage le
 * ferait mentir le jour où une paire serait filtrée. « 3 assistances sur 6 n'est pas 3 sur 300. »
 */
import { buildAssistFlow, type AssistFlow, type FlowPair } from '@/features/_shared/assists/assistFlow.logic'
import type { SquadAssistPairs } from '@/lib/api/types'

/** Les paires du bloc, en paires de flux : l'identité d'un joueur est son gamertag. */
export function squadFlowPairs(block: SquadAssistPairs): FlowPair[] {
  // `pairs` est un tableau NULLABLE au contrat (toute tranche Go sort ainsi) : comblé ici,
  // à la frontière, une seule fois.
  return (block.pairs ?? []).map((p) => ({
    assistantKey: p.assist_gamertag,
    assistantName: p.assist_gamertag,
    finisherKey: p.killer_gamertag,
    finisherName: p.killer_gamertag,
    count: p.assist_count,
    stolen: p.stolen_count,
  }))
}

/** Le flux d'« Appui » : nœuds dans l'ordre du roster, parts sur le total serveur. */
export function squadAssistFlow(block: SquadAssistPairs, roster: string[]): AssistFlow {
  return buildAssistFlow(squadFlowPairs(block), {
    assistantOrder: roster,
    finisherOrder: roster,
    total: block.total_assists,
  })
}
