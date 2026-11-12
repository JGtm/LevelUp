/**
 * vehicleDestructionSource.ts — LA SOURCE UNIQUE DES DESTRUCTIONS DE VÉHICULE du fil.
 *
 * DÉROGATION ASSUMÉE À « 0 CODE MORT » (2026-11-11, décision du user) : le document de rejeu ne
 * porte aucune destruction de véhicule (`vehicles[].end` vaut toujours `unknown`), donc
 * `vehicleDestructionsOf` rend une liste vide. La page est préparée AVANT le décodeur : tout ce
 * qui consomme ces événements — le type `VehicleDestruction`, la ligne « a détruit : Ghost »
 * (`vehicleDestructionEntries`, `ui/ReplayEventLine`), son appartenance au filtre Véhicules et son
 * compte dans le menu, le tri dans le fil — est en place et testé avec des événements fabriqués.
 *
 * C'EST LE SEUL POINT À BRANCHER. Quand le document publiera les destructions, `vehicleDestructionsOf`
 * lira dans le document, pour chaque véhicule détruit, et rendra un `VehicleDestruction` avec :
 *  - `t` : l'image de la destruction, sur l'axe de `Point.t` (celui de `frameToMs`), datée ;
 *  - `family` : la famille du véhicule, la même clé que `vehicles[].family` (le nom affiché se
 *    résout comme celui d'une montée, par `vehicleLabels`) ;
 *  - `authorXuid` : le xuid du joueur qui a détruit le véhicule, `null` tant que le décodeur ne le
 *    nomme pas (la ligne dit alors « Ghost détruit ») ;
 *  - `occupantXuid` : le xuid du joueur à bord à cet instant, `null` s'il n'est pas nommé.
 * Un champ que le décodeur ne donne pas reste `null` : rien ne se déduit ici. Le jour du
 * branchement, cette dérogation s'éteint : ce commentaire et le test « la source rend une liste
 * vide » de `vehicleFeedLogic.test.ts` se mettent à jour dans le même commit.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/** Une destruction de véhicule, telle que la source la rend à la page. */
export interface VehicleDestruction {
  t: number
  family: string
  authorXuid: string | null
  occupantXuid: string | null
}

export function vehicleDestructionsOf(doc: ReplayDocumentReady): readonly VehicleDestruction[] {
  // Aucune destruction dans le document : rien à lire. `doc` est le paramètre que le branchement
  // exploitera ; il n'est pas lu tant que le document ne publie rien.
  void doc
  return []
}
