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
 * rapprochée de l'élimination du MÊME tueur, à la victime près quand les deux la connaissent, et à l'instant près
 * (`ELIMINATION_JOIN_MS`, la fenêtre des médailles). Les paires se prennent de la plus proche à la plus lointaine ; une
 * élimination ne sert qu'une ligne, et une ligne qui a deux candidats à égale distance ne prend aucune marque : on ne
 * marque pas au hasard. Une ligne sans élimination correspondante n'a pas de marque.
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

interface Candidate {
  kill: MarkableKill
  row: number
  distanceMs: number
}

/**
 * attachEliminationMarks pose `fragMarks` sur les lignes de frag dont l'élimination du tableau est tenue pour sûre.
 * Elle ne touche aucune autre propriété et ne pose rien quand le document n'a pas de couverture des éliminations.
 */
export function attachEliminationMarks(kills: readonly MarkableKill[], doc: EliminationMarksSource | null | undefined): void {
  if (!doc || doc.coverage?.eliminations === undefined || doc.eliminations.length === 0) return
  const rows = doc.eliminations.map((e) => ({ ...e, ms: frameToMs(e.t, doc) }))
  const byKiller = new Map<string, number[]>()
  rows.forEach((r, i) => {
    const list = byKiller.get(r.xuid)
    if (list) list.push(i)
    else byKiller.set(r.xuid, [i])
  })

  const candidates: Candidate[] = []
  for (const kill of kills) {
    for (const i of byKiller.get(kill.xuid) ?? []) {
      const row = rows[i]
      if (row.victim && kill.victimXuid && row.victim !== kill.victimXuid) continue
      const distanceMs = Math.abs(row.ms - kill.replayMs)
      if (distanceMs <= ELIMINATION_JOIN_MS) candidates.push({ kill, row: i, distanceMs })
    }
  }
  candidates.sort((a, b) => a.distanceMs - b.distanceMs)

  // Une ligne dont les deux meilleurs candidats sont à égale distance est ambiguë : elle ne prend rien.
  const ambiguous = new Set<MarkableKill>()
  const best = new Map<MarkableKill, number[]>()
  for (const c of candidates) {
    const seen = best.get(c.kill)
    if (!seen) best.set(c.kill, [c.distanceMs])
    else if (seen.length === 1) seen.push(c.distanceMs)
  }
  for (const [kill, [first, second]] of best) if (second !== undefined && first === second) ambiguous.add(kill)
  const claimedRows = new Set<number>()
  const matched = new Set<MarkableKill>()
  for (const c of candidates) {
    if (ambiguous.has(c.kill) || matched.has(c.kill) || claimedRows.has(c.row)) continue
    matched.add(c.kill)
    claimedRows.add(c.row)
    const marks: FragMarks = {}
    if (rows[c.row].headshot === true) marks.headshot = true
    if (rows[c.row].perfect === true) marks.perfect = true
    if (marks.headshot || marks.perfect) c.kill.fragMarks = marks
  }
}
