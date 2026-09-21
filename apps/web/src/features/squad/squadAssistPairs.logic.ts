/**
 * squadAssistPairs.logic — les décisions PURES du graphe « Appui ».
 *
 * Aucune dépendance React/ECharts : la projection des paires en barres empilées, l'ordre
 * des assistants et les deux tables de correspondance de l'infobulle. Le composant peint,
 * ce module décide.
 *
 * ORIENTATION : une barre par ASSISTANT (catégorie), un segment empilé par BÉNÉFICIAIRE —
 * la même orientation que la matrice « Qui couvre qui » juste en dessous (ligne = celui qui
 * agit). Deux orientations opposées dans la même colonne se liraient à l'envers une fois
 * sur deux.
 */
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { SquadAssistPair, SquadAssistPairs } from '@/lib/api/types'

/** Clé de correspondance (assistant, bénéficiaire) des tables d'infobulle. */
export function assistCle(assistant: string, beneficiaire: string): string {
  return `${assistant}>${beneficiaire}`
}

/**
 * assistPairsSeries projette les paires en UNE série de barres empilées : une catégorie par
 * assistant, une composante par bénéficiaire.
 *
 * L'ORDRE DES ASSISTANTS EST CELUI DU ROSTER quand il est fourni (joueur principal d'abord,
 * puis les coéquipiers) — le même que tous les autres blocs de la page. Un assistant absent
 * du roster garde sa barre, à la suite : la mesure l'a vu, la page ne l'efface pas.
 */
export function assistPairsSeries(
  pairs: SquadAssistPair[],
  ordreRoster: string[] = [],
): ChartSeries<ChartPointStacked>[] {
  const parAssistant = new Map<string, Record<string, number>>()
  for (const p of pairs) {
    const composants = parAssistant.get(p.assist_gamertag) ?? {}
    composants[p.killer_gamertag] = (composants[p.killer_gamertag] ?? 0) + p.assist_count
    parAssistant.set(p.assist_gamertag, composants)
  }
  const rang = new Map(ordreRoster.map((gt, i) => [gt, i]))
  const assistants = [...parAssistant.keys()].sort(
    (a, b) => (rang.get(a) ?? Number.MAX_SAFE_INTEGER) - (rang.get(b) ?? Number.MAX_SAFE_INTEGER),
  )
  const datapoints = assistants.map<ChartPointStacked>((gt) => ({
    category: gt,
    components: parAssistant.get(gt) ?? {},
  }))
  return datapoints.length > 0 ? [{ key: 'assists', datapoints }] : []
}

/** Les bénéficiaires rencontrés, dans l'ordre du roster puis d'apparition — les segments. */
export function assistBeneficiaires(
  pairs: SquadAssistPair[],
  ordreRoster: string[] = [],
): string[] {
  const vus: string[] = []
  for (const p of pairs) {
    if (!vus.includes(p.killer_gamertag)) vus.push(p.killer_gamertag)
  }
  const rang = new Map(ordreRoster.map((gt, i) => [gt, i]))
  return vus.sort(
    (a, b) => (rang.get(a) ?? Number.MAX_SAFE_INTEGER) - (rang.get(b) ?? Number.MAX_SAFE_INTEGER),
  )
}

/** Éliminations volées par couple — la note « dont N volées » de l'infobulle. */
export function assistVoleesParCouple(pairs: SquadAssistPair[]): Map<string, number> {
  const out = new Map<string, number>()
  for (const p of pairs) {
    out.set(assistCle(p.assist_gamertag, p.killer_gamertag), p.stolen_count)
  }
  return out
}

/**
 * assistPartParCouple rend la PART de chaque couple dans les assistances mesurées de
 * l'escouade, en unité 0..1.
 *
 * LE DÉNOMINATEUR EST LE TOTAL SERVEUR (`total_assists`), pas la somme des barres visibles.
 * Les deux coïncident aujourd'hui, mais dériver le dénominateur de l'affichage le ferait
 * mentir le jour où une paire serait filtrée — décision reprise telle quelle de l'ancien
 * tableau, dont ce graphe prend la place.
 */
export function assistPartParCouple(block: SquadAssistPairs): Map<string, number> {
  const out = new Map<string, number>()
  const total = block.total_assists
  if (total <= 0) return out
  for (const p of block.pairs ?? []) {
    out.set(assistCle(p.assist_gamertag, p.killer_gamertag), p.assist_count / total)
  }
  return out
}
