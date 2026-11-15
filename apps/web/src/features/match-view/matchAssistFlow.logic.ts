/**
 * matchAssistFlow.logic — la projection des paires d'assistances d'un match
 * (`combat_tab.assist_pairs.pairs`) vers le diagramme de flux partagé
 * (`features/_shared/assists/assistFlow.logic.ts`) : assistants à gauche, finisseurs à droite.
 *
 * On reçoit déjà les paires agrégées et comptées : aucun décompte ici, y compris pour les
 * assistances où l'assistant a fait plus de dégâts que le finisseur (elles voyagent avec la paire
 * et se disent dans l'infobulle du ruban).
 *
 * ORDRE DES ASSISTANTS : ennemis d'abord, puis total décroissant — la règle du graphe des
 * antagonistes, pour que les deux se lisent sans changer de grille de lecture ; les finisseurs
 * suivent leur total décroissant.
 *
 * NOMS : un joueur sans gamertag résolu (absent du scoreboard) passe par `displayPlayerName`, qui
 * rend le repli masqué « Joueur #### » — jamais un xuid brut. LES BOTS COMPTENT : un assistant ou
 * un finisseur sans xuid (bot) arrive nommé par le film et s'identifie par ce nom ; son camp, absent
 * du scoreboard, est celui des finisseurs qu'il a assistés (une assistance va toujours à un
 * coéquipier).
 */
import { buildAssistFlow, type AssistFlow, type FlowPair } from '@/features/_shared/assists/assistFlow.logic'
import type { MatchAssistPair, MatchScoreboardRow } from '@/lib/api/types'
import { displayPlayerName } from '@/lib/players/displayName'

/** Identité d'un assistant : son xuid, ou son gamertag de film quand il n'en a pas (bot). */
function assistantKey(p: MatchAssistPair): string {
  return p.assist_xuid || `gt:${p.assist_gamertag}`
}

/** Identité d'un finisseur : son xuid, ou son gamertag de film quand il n'en a pas (bot). */
function finisherKey(p: MatchAssistPair): string {
  return p.killer_xuid || `gt:${p.killer_gamertag ?? ''}`
}

/** Les paires du match, en paires de flux, noms résolus par `displayPlayerName`. */
export function matchFlowPairs(pairs: readonly MatchAssistPair[]): FlowPair[] {
  return pairs.map((p) => ({
    assistantKey: assistantKey(p),
    assistantName: displayPlayerName(p.assist_gamertag, p.assist_xuid),
    finisherKey: finisherKey(p),
    finisherName: displayPlayerName(p.killer_gamertag, p.killer_xuid),
    count: p.assist_count,
    stolen: p.stolen_count,
    avgPct: p.avg_assist_pct ?? undefined,
  }))
}

/**
 * assistantOrder range les assistants (clés) : ennemis d'abord (camp de l'assistant, sinon celui
 * des finisseurs qu'il a assistés), puis total décroissant. Sans `meXUID` ou sans scoreboard, nul
 * n'est ennemi : l'ordre est celui du total.
 */
export function matchAssistantOrder(
  pairs: readonly MatchAssistPair[],
  scoreboard: readonly MatchScoreboardRow[] = [],
  meXUID?: string | null,
): string[] {
  const totals = new Map<string, { xuid: string; total: number; finishers: string[] }>()
  for (const p of pairs) {
    const key = assistantKey(p)
    const acc = totals.get(key) ?? { xuid: p.assist_xuid, total: 0, finishers: [] }
    acc.total += p.assist_count
    if (p.killer_xuid) acc.finishers.push(p.killer_xuid)
    totals.set(key, acc)
  }
  const meRow = meXUID ? scoreboard.find((r) => r.xuid === meXUID) : undefined
  const allyTeam = meRow?.team_side ?? null
  const teamOf = new Map<string, string | undefined>(scoreboard.map((r) => [r.xuid, r.team_side]))
  const isEnemy = ({ xuid, finishers }: { xuid: string; finishers: string[] }): boolean => {
    if (allyTeam == null) return false
    const own = xuid ? teamOf.get(xuid) : undefined
    const team = own ?? finishers.map((k) => teamOf.get(k)).find((t) => t != null)
    return team != null && team !== allyTeam
  }
  return [...totals.entries()]
    .sort(([, a], [, b]) => {
      const ea = isEnemy(a) ? 0 : 1
      const eb = isEnemy(b) ? 0 : 1
      return ea !== eb ? ea - eb : b.total - a.total
    })
    .map(([key]) => key)
}

/** Le flux du match : assistants ennemis d'abord, parts absentes (aucun total de référence). */
export function matchAssistFlow(
  pairs: readonly MatchAssistPair[],
  scoreboard: readonly MatchScoreboardRow[] = [],
  meXUID?: string | null,
): AssistFlow {
  return buildAssistFlow(matchFlowPairs(pairs), {
    assistantOrder: matchAssistantOrder(pairs, scoreboard, meXUID),
  })
}
