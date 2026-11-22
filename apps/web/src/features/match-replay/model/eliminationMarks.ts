/**
 * eliminationMarks.ts — LES MARQUES DE TIR À LA TÊTE ET DE FRAG PARFAIT D'UNE LIGNE DU FIL, lues dans le document.
 *
 * LA SOURCE. `doc.eliminations` porte une ligne par élimination du tableau des scores : le tueur, l'image où son
 * compteur monte, la victime quand le film la donne, et deux marques que le JEU écrit dans ses compteurs (la montée du
 * compteur de tirs à la tête, la médaille « Parfait »). Rien n'est estimé : c'est la source voulue pour que les icônes
 * du fil disent la même chose que les totaux de la fiche. Une marque est vraie, fausse ou ABSENTE (inconnue) ; un
 * inconnu ne se lit jamais comme un faux, et `perfect` ne vaut jamais faux.
 *
 * LA JOINTURE. Le fil des éliminations vient de la base (événements du match), pas du film : chaque ligne du fil est
 * rapprochée d'une élimination du MÊME tueur, compatible quand la victime n'est pas contredite (connue des deux côtés
 * et différente) et que l'écart de temps tient dans `ELIMINATION_JOIN_MS` (la fenêtre des médailles). Les lignes et les
 * éliminations d'un même tueur que cette compatibilité relie forment un groupe en concurrence, résolu seul :
 *  - autant de lignes que d'éliminations : elles s'apparient DANS L'ORDRE DU TEMPS (la k-ième ligne prend la k-ième
 *    élimination), pas de la plus proche à la plus lointaine : un décalage constant entre les deux horloges croiserait
 *    sinon deux frags rapprochés ;
 *  - un nombre différent, ou un appariement dans l'ordre que la victime contredit : seule la victime tranche, quand
 *    elle est connue des deux côtés et qu'une ligne et une élimination sont les seules du groupe à la porter ;
 *  - sinon le groupe ne prend AUCUNE marque : on ne marque pas au hasard.
 * Une ligne sans élimination correspondante n'a pas de marque.
 *
 * CE QU'ON AFFICHE. Seules les marques VRAIES s'affichent : un faux ou un inconnu n'a pas d'icône, et aucun texte ne
 * dit « non mesuré ». Sans `coverage.eliminations` (film ancien, pas de tableau des scores), aucune ligne n'est marquée.
 *
 * Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/**
 * Écart maximal, en ms, entre une ligne du fil et l'élimination du tableau qui la porte : la même fenêtre que le
 * rattachement d'une médaille à son élimination (`MEDAL_ATTACH_MS`), l'horloge des deux sources étant la même.
 */
export const ELIMINATION_JOIN_MS = 500

/** Les marques VRAIES d'une ligne de frag : une clé absente n'est pas fausse, elle n'est pas affichée. */
export interface FragMarks {
  headshot?: true
  perfect?: true
}

/** Ce que la jointure lit d'une ligne de frag du fil, et la marque qu'elle y pose. */
export interface MarkableKill {
  xuid: string
  victimXuid: string
  replayMs: number
  fragMarks?: FragMarks
}

/** Ce que la jointure lit du document. */
export type EliminationMarksSource = Pick<ReplayDocumentReady, 'eliminations' | 'coverage' | 'frameIntervalMs'>

/** Une élimination du tableau, placée sur l'axe du rejeu. */
interface ScoreboardRow {
  ms: number
  victim?: string
  headshot?: boolean
  perfect?: boolean
}

/** Une ligne du fil et une élimination du tableau sont compatibles : victime non contredite, instant dans la fenêtre. */
function compatible(kill: MarkableKill, row: ScoreboardRow): boolean {
  if (row.victim && kill.victimXuid && row.victim !== kill.victimXuid) return false
  return Math.abs(row.ms - kill.replayMs) <= ELIMINATION_JOIN_MS
}

/** Un groupe en concurrence : des lignes et des éliminations du même tueur reliées par la compatibilité. */
interface CompetingGroup {
  kills: MarkableKill[]
  rows: ScoreboardRow[]
}

/** Range les lignes et les éliminations d'UN tueur en groupes (composantes connexes de la compatibilité). */
function groupsOf(kills: readonly MarkableKill[], rows: readonly ScoreboardRow[]): CompetingGroup[] {
  const seenRows = new Set<ScoreboardRow>()
  const seenKills = new Set<MarkableKill>()
  const groups: CompetingGroup[] = []
  for (const start of kills) {
    if (seenKills.has(start)) continue
    const group: CompetingGroup = { kills: [], rows: [] }
    const queue: MarkableKill[] = [start]
    seenKills.add(start)
    for (let kill = queue.pop(); kill !== undefined; kill = queue.pop()) {
      group.kills.push(kill)
      for (const row of rows) {
        if (seenRows.has(row) || !compatible(kill, row)) continue
        seenRows.add(row)
        group.rows.push(row)
        for (const other of kills) {
          if (seenKills.has(other) || !compatible(other, row)) continue
          seenKills.add(other)
          queue.push(other)
        }
      }
    }
    groups.push(group)
  }
  return groups
}

/** Les paires qu'un groupe permet de poser : dans l'ordre du temps, à défaut par la victime, sinon aucune. */
function pairsOf(group: CompetingGroup): [MarkableKill, ScoreboardRow][] {
  const kills = [...group.kills].sort((a, b) => a.replayMs - b.replayMs)
  const rows = [...group.rows].sort((a, b) => a.ms - b.ms)
  if (kills.length === rows.length) {
    const ordered = kills.map((k, i): [MarkableKill, ScoreboardRow] => [k, rows[i]])
    if (ordered.every(([k, r]) => compatible(k, r))) return ordered
  }
  const byVictim: [MarkableKill, ScoreboardRow][] = []
  for (const kill of kills) {
    if (!kill.victimXuid || kills.filter((k) => k.victimXuid === kill.victimXuid).length !== 1) continue
    const named = rows.filter((r) => r.victim === kill.victimXuid)
    if (named.length === 1 && compatible(kill, named[0])) byVictim.push([kill, named[0]])
  }
  return byVictim
}

/** Range des éléments sous une clé, en gardant l'ordre d'arrivée. */
function pushUnder<T>(map: Map<string, T[]>, key: string, value: T): void {
  const list = map.get(key)
  if (list) list.push(value)
  else map.set(key, [value])
}

/**
 * attachEliminationMarks pose `fragMarks` sur les lignes de frag dont l'élimination du tableau est tenue pour sûre.
 * Elle ne touche aucune autre propriété et ne pose rien quand le document n'a pas de couverture des éliminations.
 */
export function attachEliminationMarks(kills: readonly MarkableKill[], doc: EliminationMarksSource | null | undefined): void {
  if (!doc || doc.coverage?.eliminations === undefined || doc.eliminations.length === 0) return
  const rowsByKiller = new Map<string, ScoreboardRow[]>()
  for (const e of doc.eliminations) {
    pushUnder(rowsByKiller, e.xuid, { ms: frameToMs(e.t, doc), victim: e.victim, headshot: e.headshot, perfect: e.perfect })
  }
  const killsByKiller = new Map<string, MarkableKill[]>()
  for (const kill of kills) pushUnder(killsByKiller, kill.xuid, kill)
  for (const [xuid, own] of killsByKiller) {
    for (const group of groupsOf(own, rowsByKiller.get(xuid) ?? [])) {
      for (const [kill, row] of pairsOf(group)) {
        const marks: FragMarks = {}
        if (row.headshot === true) marks.headshot = true
        if (row.perfect === true) marks.perfect = true
        if (marks.headshot || marks.perfect) kill.fragMarks = marks
      }
    }
  }
}
