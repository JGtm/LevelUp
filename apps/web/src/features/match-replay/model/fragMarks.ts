/**
 * fragMarks — LES REPÈRES D'ÉLIMINATIONS DE LA FRISE COMPACTE : les frags de TOUS les joueurs, un repère
 * par frag, à la couleur du camp du tueur. Logique pure, sans JSX.
 *
 * La frise à pistes montre les marques du joueur regardé et de ses coéquipiers ; la frise compacte n'a
 * qu'une bande de repères et dit la forme du match d'un coup d'œil : qui frappe, et de quel camp. Les
 * frags viennent du MÊME parcours du fil que la dominance (`reduceFeed`), donc une ligne ajoutée au fil
 * apparaît sur les deux frises. Le camp d'un repère est celui du tueur (`teamId`) ; sa couleur se
 * résout à l'affichage par `allyOf`, comme les bandes de dominance.
 *
 * Un frag antérieur ou postérieur à la fenêtre de gameplay n'a pas de place sur la frise et n'est pas
 * posé (la dominance, elle, le compte : sa bande commence au bord).
 */
import { ratioOfMs, type TrackFrag, type TrackScale } from './replayTimelineTracksLogic'

/** Un repère d'élimination : sa clé, sa place sur la frise (ratio [0..1]) et le camp du tueur. */
export interface FragMark {
  key: string
  ratio: number
  teamId: number
}

/** buildFragMarks pose les frags sur l'échelle de la frise ; `[]` quand la frise n'a rien à placer. */
export function buildFragMarks(
  frags: readonly TrackFrag[],
  frameIntervalMs: number,
  scale: TrackScale,
): FragMark[] {
  if (scale.span <= 0 || !frameIntervalMs) return []
  const marks: FragMark[] = []
  frags.forEach((frag, i) => {
    const ratio = ratioOfMs(frag.replayMs, frameIntervalMs, scale)
    if (ratio < 0 || ratio > 1) return
    marks.push({ key: `frag-${i}`, ratio, teamId: frag.teamId })
  })
  return marks
}
