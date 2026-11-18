/**
 * playerInstantMedals.ts — LES MÉDAILLES D'UN JOUEUR JUSQU'À L'IMAGE LUE, groupées par médaille. Sans React.
 *
 * La liste vient de `playerInstantStatsAt` (qui lit `readMatchMedals`, la lecture unique des médailles du
 * document) ; ce module ne fait que les grouper, avec leur compte (« ×n »), et leur joindre ce que la table du
 * document (`medalLabels`) porte en plus du libellé : la feuille de sprites d'une médaille qui n'a pas d'image
 * propre, et sa rareté (`difficulty`), que le halo de l'icône dit comme sur les tuiles de match.
 *
 * L'ordre est celui de la PREMIÈRE obtention. Une médaille que la table ne connaît pas garde son nom brut et
 * n'a ni image ni halo : aucun visuel n'est inventé.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { MedalEvent } from './killFeedLogic'

/** Une médaille gagnée `count` fois jusqu'à l'image. */
export interface MedalGroup {
  /** Le nom du film (la clé de la table des libellés). */
  name: string
  /** Le libellé dans la langue de la page, à défaut le nom du film. */
  label: string
  description: string
  count: number
  /** L'image de la médaille ; vide quand la table n'en porte pas. */
  imageUrl: string
  /** La découpe d'une feuille de sprites, quand la médaille n'a pas d'image propre. */
  sprite: { sheet: string; left: number; top: number; width: number; height: number } | null
  /** La rareté de la médaille (`Normal`, `Heroic`, `Legendary`, `Mythic`), quand la table la porte. */
  difficulty: string | null
}

/** medalGroupsOf groupe les médailles gagnées par nom, dans l'ordre de la première obtention. `null` = aucune liste. */
export function medalGroupsOf(
  medals: readonly MedalEvent[] | null,
  doc: Pick<ReplayDocumentReady, 'medalLabels'>,
): MedalGroup[] {
  const groups = new Map<string, MedalGroup>()
  for (const m of medals ?? []) {
    const known = groups.get(m.name)
    if (known) {
      known.count++
      continue
    }
    const entry = doc.medalLabels?.[m.name]
    groups.set(m.name, {
      name: m.name,
      label: m.label || m.name,
      description: m.description,
      count: 1,
      imageUrl: m.imageUrl,
      sprite: entry?.sprite ?? null,
      difficulty: entry?.difficulty ?? null,
    })
  }
  return [...groups.values()]
}
