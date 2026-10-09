/**
 * squadFragBreakdownChart — MODÈLE de la « Répartition des frags » par joueur (barres
 * empilées PAR CLASSE d'arme).
 *
 * Une barre horizontale par joueur, segments = CLASSE d'arme (Épaule / Poing / Lourde /
 * Mêlée / Grenade / … / Non attribué, ordre canonique FRAG_CLASS_ORDER), longueur = total
 * des frags, sur une ÉCHELLE COMMUNE aux joueurs (le plus gros total = 100 %) : les barres
 * se comparent d'un coup d'œil.
 *
 * Rendu DOM (SquadFragBreakdownCard), plus ECharts depuis le lot L2 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 : le compte de chaque classe s'écrit DANS son
 * segment quand il y tient (mesure au pixel, `components/charts/segmentLabelFit`), sinon sur
 * une ligne de repli au-dessus de la barre (règle S3) — ce qu'un canvas ne sait pas faire.
 *
 * Consomme `frag_classes` (map gamertag → FragClassEntry[], agrégat serveur par classe via
 * fragdist.Build). Couleurs = `fragClassColor` / `fragClassCssVar` côté rendu.
 */
import type { FragClassEntry } from '@/lib/api/types'
import { relLuminance } from '@/lib/accessibility'
import { FRAG_CLASS_ORDER } from '@/lib/accessibility/scales'

/** Un segment de la barre d'un joueur. */
export interface FragBreakdownSegment {
  cls: string
  kills: number
  /** Position et largeur sur l'échelle commune (0..100). */
  leftPct: number
  widthPct: number
}

/** La barre d'un joueur. */
export interface FragBreakdownRow {
  player: string
  total: number
  segments: FragBreakdownSegment[]
}

function orderedPlayers(rows: Record<string, FragClassEntry[]>, playerOrder?: string[]): string[] {
  if (playerOrder && playerOrder.length > 0) return playerOrder.filter((p) => rows[p] !== undefined)
  return Object.keys(rows).sort()
}

/** Map class → kills pour un joueur (agrégat serveur déjà au niveau classe). */
function killsByClass(entries: FragClassEntry[]): Map<string, number> {
  const m = new Map<string, number>()
  for (const e of entries) m.set(e.class, (m.get(e.class) ?? 0) + e.kills)
  return m
}

/** Classes présentes chez au moins un joueur, dans l'ordre canonique FRAG_CLASS_ORDER. */
function presentClasses(byPlayer: Map<string, Map<string, number>>): string[] {
  const present = new Set<string>()
  for (const kills of byPlayer.values()) {
    for (const [cls, v] of kills) if (v > 0) present.add(cls)
  }
  return FRAG_CLASS_ORDER.filter((c) => present.has(c))
}

function killsByPlayer(rows: Record<string, FragClassEntry[]>, players: string[]) {
  const byPlayer = new Map<string, Map<string, number>>()
  for (const player of players) byPlayer.set(player, killsByClass(rows[player] ?? []))
  return byPlayer
}

/**
 * Les classes de frags REELLEMENT presentes, dans l'ordre canonique — la meme liste que
 * celle des segments. Exportee pour que la legende du pied de carte se construise sur la
 * MEME source que les barres : deux listes calculees separement divergeraient au premier
 * changement de regle.
 */
export function fragBreakdownClasses(
  rows: Record<string, FragClassEntry[]>,
  playerOrder?: string[],
): string[] {
  const players = orderedPlayers(rows, playerOrder)
  if (players.length === 0) return []
  return presentClasses(killsByPlayer(rows, players))
}

/**
 * Les barres, une par joueur (ordre `playerOrder`), segments dans l'ordre canonique des
 * classes, largeurs sur l'échelle commune. Vide si aucun frag.
 */
export function buildFragBreakdownRows(
  rows: Record<string, FragClassEntry[]>,
  playerOrder?: string[],
): FragBreakdownRow[] {
  const players = orderedPlayers(rows, playerOrder)
  const byPlayer = killsByPlayer(rows, players)
  const classes = presentClasses(byPlayer)
  if (classes.length === 0) return []

  const totals = players.map((p) => classes.reduce((s, c) => s + Math.max(0, byPlayer.get(p)?.get(c) ?? 0), 0))
  const scale = Math.max(...totals)
  if (scale <= 0) return []

  return players.map((player, i) => {
    let left = 0
    const segments: FragBreakdownSegment[] = []
    for (const cls of classes) {
      const kills = byPlayer.get(player)?.get(cls) ?? 0
      if (kills <= 0) continue
      const widthPct = (kills / scale) * 100
      segments.push({ cls, kills, leftPct: left, widthPct })
      left += widthPct
    }
    return { player, total: totals[i], segments }
  })
}

/**
 * Position (0..100) du PREMIER segment dont le compte ne tient pas : la ligne de repli
 * s'aligne sur lui (règle S3). null si tous tiennent.
 */
export function repliOffsetPct(
  segments: FragBreakdownSegment[],
  isHidden: (cls: string) => boolean,
): number | null {
  const first = segments.find((s) => isHidden(s.cls))
  return first ? first.leftPct : null
}

/** Luminance au-delà de laquelle un texte sombre contraste mieux qu'un texte clair (WCAG). */
const DARK_TEXT_LUMINANCE = 0.179

/**
 * Ton de l'écriture posée SUR l'aplat d'un segment : sombre sur une classe claire (ambre,
 * émeraude), clair sinon — le seuil qui maximise le contraste WCAG. Couleur illisible (valeur
 * non hex) → clair.
 */
export function segmentTextTone(fillHex: string): 'dark' | 'light' {
  try {
    return relLuminance(fillHex) > DARK_TEXT_LUMINANCE ? 'dark' : 'light'
  } catch {
    return 'light'
  }
}
