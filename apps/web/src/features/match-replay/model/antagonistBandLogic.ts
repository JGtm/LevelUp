/**
 * antagonistBandLogic — LA BANDE DES TITRES DE LA FRISE : les changements de titre du joueur regardé,
 * mis en marques posées sur l'axe, avec leur texte. Pur : ni React ni DOM.
 *
 * Il ne compte rien et ne rapproche aucune clé : la lecture vient de `lib/antagonists` par la
 * fabrique du rejeu (`replayAntagonists`), qui rend les changements aux clés du FILM. Ce fichier ne
 * fait que trois choses :
 *  - ne garder que les PRISES de titre : une égalité ne change pas de main, la frise ne la dit pas ;
 *  - poser chaque prise sur l'axe, avec la géométrie des marques (`ratioOfMs`) et l'image que visent
 *    les portes de présence (`Math.round(t / frameIntervalMs)`) ; une prise hors de la fenêtre de jeu
 *    n'a pas de place et n'est pas posée, comme un frag hors fenêtre ;
 *  - écrire le texte : titre, puis phrase (première prise, ou dépassement de l'ancien tenant).
 *
 * Les noms arrivent par `nameOf` (la fonction de nom du rejeu, cf. `antagonistNames`) : la langue
 * des phrases est celle du dictionnaire passé, jamais lue ici.
 */
import type { AntagonistChange, AntagonistRole } from '@/lib/antagonists/antagonists'

import type { AntagonistsText } from '../i18n/antagonists.i18n'
import { ratioOfMs, type TrackScale } from './replayTimelineTracksLogic'

/** Une prise de titre posée sur la frise. */
export interface AntagonistBandItem {
  key: string
  /** Position sur l'axe, [0..1] (l'abscisse se rend par `trackLeft`). */
  ratio: number
  /** L'image que vise le clic : celle de l'élimination qui a fait passer le titre. */
  frame: number
  role: AntagonistRole
  /** La clé du film du nouveau tenant (adversaire du joueur regardé). */
  holder: string
  /** « Némésis : X » ou « Souffre-douleur : X ». */
  title: string
  /** La phrase sous le titre. */
  phrase: string
}

export interface AntagonistBandContext {
  /** Le nom affiché d'un joueur désigné par sa clé du film. */
  nameOf: (key: string) => string
  /** Le nom affiché du joueur regardé (N dans les phrases). */
  viewpointName: string
  frameIntervalMs: number
  scale: TrackScale
  text: AntagonistsText
}

/** La phrase d'une prise : la première du titre, ou un dépassement avec l'ancien tenant et son compte. */
function phraseOf(change: AntagonistChange, ctx: AntagonistBandContext): string {
  const role = ctx.text.band[change.role]
  if (change.previousHolder === null) return role.firstFmt(ctx.viewpointName)
  return role.overtakeFmt(ctx.viewpointName, change.count, ctx.nameOf(change.previousHolder), change.previousCount)
}

/**
 * buildAntagonistBand — les prises de titre du joueur regardé, dans l'ordre chronologique des
 * changements reçus. Liste vide : pas de changement, pas de prise, échelle sans étendue ou prises hors fenêtre.
 */
export function buildAntagonistBand(
  changes: readonly AntagonistChange[],
  ctx: AntagonistBandContext,
): AntagonistBandItem[] {
  const items: AntagonistBandItem[] = []
  if (ctx.scale.span <= 0 || !ctx.frameIntervalMs) return items
  for (const change of changes) {
    if (change.kind !== 'prise') continue
    const ratio = ratioOfMs(change.t, ctx.frameIntervalMs, ctx.scale)
    if (ratio < 0 || ratio > 1) continue
    items.push({
      key: `${change.role}-${change.t}-${change.holder}`,
      ratio,
      frame: Math.round(change.t / ctx.frameIntervalMs),
      role: change.role,
      holder: change.holder,
      title: ctx.text.band[change.role].titleFmt(ctx.nameOf(change.holder)),
      phrase: phraseOf(change, ctx),
    })
  }
  return items
}
