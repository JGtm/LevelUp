/**
 * livesLayout.ts — les encres et la grille de la carte « Isolement » (à portée `squad-player-1`, isolée
 * `extreme`, colonne des noms de 150 px), partagées par la ligne (`LivesNearTeammateRow`), la carte des
 * Séries temporelles et de Sessions, et la carte par joueur de la Vue match. Hors du fichier de
 * composant : un module de composant n'exporte que des composants (rafraîchissement à chaud).
 */
import { tokenCssVar } from '@/lib/accessibility'

import { squadPlayerInk } from '@/features/squad/formes/colors'
import { pisteColumns } from '@/features/squad/emprise/pisteLayout'

export const LIVES_NEAR_INK = squadPlayerInk(0)
export const LIVES_ALONE_INK = tokenCssVar('extreme')
/** Colonne des noms : 150 px (maquette). */
export const LIVES_COLUMNS = pisteColumns(150)