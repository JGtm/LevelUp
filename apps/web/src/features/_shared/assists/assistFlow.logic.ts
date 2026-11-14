/**
 * assistFlow.logic — les décisions PURES du diagramme de flux « assistant -> finisseur » : qui
 * prépare le frag (l'ASSISTANT, à gauche) et qui le conclut (le FINISSEUR, à droite). Partagé par
 * « Appui » (Escouade › Synergies) et « Assistances » (page Match) : à question identique, forme
 * identique.
 *
 * CHAQUE JOUEUR A DEUX NŒUDS, un par côté (`assistant:<clé>` et `finisher:<clé>`) : le diagramme
 * reste biparti, sans cycle, et un joueur qui assiste autant qu'il conclut se lit des deux côtés.
 * Un ruban relie un assistant à un finisseur ; sa largeur est le nombre d'assistances, sa part
 * celle du total (celui du serveur quand l'appelant le donne : le dénominateur ne se déduit pas de
 * ce qui est affiché).
 *
 * Aucune dépendance React/ECharts : `assistFlowOption.ts` ne fait que peindre ce que ce module
 * décide.
 */

/** Une paire (assistant, finisseur) telle que les pages la fournissent. */
export interface FlowPair {
  /** Clé d'identité (xuid, ou nom du film pour un bot) : deux joueurs de même nom restent distincts. */
  assistantKey: string
  assistantName: string
  finisherKey: string
  finisherName: string
  /** Assistances de l'assistant sur les frags du finisseur. */
  count: number
  /** Parmi elles, celles où l'assistant a fait plus de dégâts que le finisseur. */
  stolen: number
  /** Part moyenne de participation de l'assistant, 0..n (non plafonnée) ; absente si non servie. */
  avgPct?: number
}

export type FlowSide = 'assistant' | 'finisher'

/** Un nœud : un joueur d'un côté du diagramme. */
export interface FlowNode {
  id: string
  side: FlowSide
  key: string
  name: string
  /** Assistances données (côté assistant) ou frags conclus sur une assistance (côté finisseur). */
  total: number
}

/** Un ruban : l'assistant, le finisseur, le nombre d'assistances. */
export interface FlowLink {
  source: string
  target: string
  assistantKey: string
  assistantName: string
  finisherKey: string
  finisherName: string
  count: number
  stolen: number
  /** Part du total (0..1) ; `null` quand aucun total n'est donné. */
  share: number | null
  avgPct?: number
}

export interface AssistFlow {
  nodes: FlowNode[]
  links: FlowLink[]
  /** Total des assistances de la lecture : celui de l'appelant, sinon la somme des rubans. */
  total: number
}

export interface AssistFlowOpts {
  /** Ordre des assistants (clés) : les inconnus suivent, par total décroissant. */
  assistantOrder?: readonly string[]
  /** Ordre des finisseurs (clés) : les inconnus suivent, par total décroissant. */
  finisherOrder?: readonly string[]
  /** Total des assistances de la lecture, dénominateur des parts. */
  total?: number
}

/** Identifiant d'un nœud : le côté et la clé du joueur. */
export function flowNodeId(side: FlowSide, key: string): string {
  return `${side}:${key}`
}

function ranked(keys: string[], totals: Map<string, number>, order: readonly string[] | undefined): string[] {
  const rank = new Map((order ?? []).map((k, i) => [k, i]))
  return [...keys].sort((a, b) => {
    const ra = rank.get(a) ?? Number.MAX_SAFE_INTEGER
    const rb = rank.get(b) ?? Number.MAX_SAFE_INTEGER
    if (ra !== rb) return ra - rb
    return (totals.get(b) ?? 0) - (totals.get(a) ?? 0)
  })
}

/**
 * buildAssistFlow projette les paires en nœuds (deux colonnes) et en rubans. Les paires à zéro
 * assistance sont écartées ; deux paires du même couple s'additionnent.
 */
export function buildAssistFlow(pairs: readonly FlowPair[], opts: AssistFlowOpts = {}): AssistFlow {
  const merged = new Map<string, FlowLink>()
  const assistantNames = new Map<string, string>()
  const finisherNames = new Map<string, string>()
  const given = new Map<string, number>()
  const concluded = new Map<string, number>()
  let sum = 0
  for (const p of pairs) {
    if (p.count <= 0) continue
    assistantNames.set(p.assistantKey, p.assistantName)
    finisherNames.set(p.finisherKey, p.finisherName)
    given.set(p.assistantKey, (given.get(p.assistantKey) ?? 0) + p.count)
    concluded.set(p.finisherKey, (concluded.get(p.finisherKey) ?? 0) + p.count)
    sum += p.count
    const id = `${p.assistantKey}\t${p.finisherKey}`
    const prev = merged.get(id)
    if (prev) {
      prev.count += p.count
      prev.stolen += p.stolen
      continue
    }
    merged.set(id, {
      source: flowNodeId('assistant', p.assistantKey),
      target: flowNodeId('finisher', p.finisherKey),
      assistantKey: p.assistantKey,
      assistantName: p.assistantName,
      finisherKey: p.finisherKey,
      finisherName: p.finisherName,
      count: p.count,
      stolen: p.stolen,
      share: null,
      avgPct: p.avgPct,
    })
  }
  const total = opts.total != null && opts.total > 0 ? opts.total : sum
  const links = [...merged.values()].map((l) => ({
    ...l,
    share: opts.total != null && opts.total > 0 ? l.count / opts.total : null,
  }))
  const nodes: FlowNode[] = [
    ...ranked([...given.keys()], given, opts.assistantOrder).map<FlowNode>((key) => ({
      id: flowNodeId('assistant', key),
      side: 'assistant',
      key,
      name: assistantNames.get(key) ?? key,
      total: given.get(key) ?? 0,
    })),
    ...ranked([...concluded.keys()], concluded, opts.finisherOrder).map<FlowNode>((key) => ({
      id: flowNodeId('finisher', key),
      side: 'finisher',
      key,
      name: finisherNames.get(key) ?? key,
      total: concluded.get(key) ?? 0,
    })),
  ]
  // Les rubans suivent l'ordre des nœuds de gauche, puis de droite : le tracé est déterministe.
  const position = new Map(nodes.map((n, i) => [n.id, i]))
  links.sort((a, b) => (position.get(a.source) ?? 0) - (position.get(b.source) ?? 0) || (position.get(a.target) ?? 0) - (position.get(b.target) ?? 0))
  return { nodes, links, total }
}
