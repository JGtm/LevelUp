/**
 * vehicleDestructionSource.ts — LA SOURCE UNIQUE DES DESTRUCTIONS DE VÉHICULE du fil.
 *
 * CE QUE LE DOCUMENT PUBLIE. Une vie de véhicule dont `end` vaut `destroyed` porte l'instant de la
 * destruction (`tEnd`, sur l'axe de `Point.t`, celui de `frameToMs`) et, quand le film en dit
 * davantage, un bloc facultatif `destruction` :
 *  - `by` : le destructeur. `by.i` est l'index de joueur du film, joint à `roster[].filmIndex` ;
 *    `by.xuid` nomme un humain ; un BOT n'a pas de xuid et porte `by.bot` ; `by` absent = non établi ;
 *  - `cause` : l'identifiant d'effet de dégât, dans le même espace que la source d'un frag ;
 *  - `occupant` : le xuid de l'occupant à l'instant (le conducteur de préférence) ;
 *  - `killed` : les xuids des occupants tués.
 * Toute absence est une absence : un champ non établi n'est jamais une valeur par défaut, et rien ne
 * se déduit ici. Un rejeu cuit avant ce bloc, ou dont le film n'est plus archivé, n'en porte pas : la
 * liste est alors celle des seules fins `destroyed` datées, sans auteur ni occupant, ou vide.
 *
 * UNE DESTRUCTION PAR VÉHICULE. Une pièce montée (tourelle, `part` non vide) meurt avec son porteur
 * et ne porte jamais le bloc : elle ne fait aucune ligne. Une vie de décor, et une fin `destroyed`
 * sans `tEnd`, n'en font pas non plus : sans instant, rien ne se date.
 *
 * CE QUE LA SOURCE NE PORTE PAS. Le siège (`rides[].seat`) n'est pas lu : sa fiabilité n'est pas
 * établie. La cause n'est pas lue non plus : son pictogramme se résout côté serveur, par la table des
 * armes des frags, et aucun canal du document ne l'apporte à la page pour une destruction.
 *
 * LA LIGNE ET LE FILTRE CONSOMMENT CETTE LISTE : `vehicleDestructionEntries`
 * (`model/vehicleFeedLogic.ts`) la range dans le fil, `ui/ReplayEventLine` la dit.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/** Une destruction de véhicule, telle que la source la rend à la page. */
export interface VehicleDestruction {
  /** L'image de la destruction, sur l'axe de `Point.t`. */
  t: number
  /** La clé de famille de `vehicles[].family` ('' quand le châssis n'est pas résolu). */
  family: string
  /** Le xuid du destructeur humain ; `null` pour un bot, un destructeur non établi ou absent. */
  authorXuid: string | null
  /** Le nom du destructeur BOT, tel que le roster l'écrit ; `null` quand il n'est pas nommé. */
  authorBotName: string | null
  /** Les xuids des joueurs connus à bord à l'instant : l'occupant, puis les tués, sans doublon. */
  aboardXuids: readonly string[]
}

/** La forme servie de `vehicles[].destruction` : chaque champ est facultatif. */
type Destruction = NonNullable<ReplayDocumentReady['vehicles'][number]['destruction']>

/** La fin que le film écrit pour une destruction. */
const END_DESTROYED = 'destroyed'

/** Les noms de bot du roster, par index de film. */
function botNamesByFilmIndex(doc: ReplayDocumentReady): ReadonlyMap<number, string> {
  const names = new Map<number, string>()
  for (const r of doc.roster) {
    if (r.bot && r.name) names.set(r.filmIndex, r.name)
  }
  return names
}

/** Les joueurs connus à bord : l'occupant d'abord, puis les tués, chacun une fois. */
function aboardOf(d: Destruction | undefined): string[] {
  const out: string[] = []
  for (const xuid of [d?.occupant, ...(d?.killed ?? [])]) {
    if (xuid && !out.includes(xuid)) out.push(xuid)
  }
  return out
}

export function vehicleDestructionsOf(doc: ReplayDocumentReady): readonly VehicleDestruction[] {
  let bots: ReadonlyMap<number, string> | null = null
  const out: VehicleDestruction[] = []
  for (const track of doc.vehicles) {
    if (track.end !== END_DESTROYED || track.part || track.scenery || track.tEnd == null) continue
    const by = track.destruction?.by
    let botName: string | null = null
    if (by && !by.xuid && by.bot) {
      bots ??= botNamesByFilmIndex(doc)
      botName = bots.get(by.i) ?? null
    }
    out.push({
      t: track.tEnd,
      family: track.family ?? '',
      authorXuid: by?.xuid ? by.xuid : null,
      authorBotName: botName,
      aboardXuids: aboardOf(track.destruction),
    })
  }
  return out
}
