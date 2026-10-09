/**
 * useSquadPlayerPalette — la palette des joueurs de l'Escouade, lue sur le contexte de la page :
 * le joueur principal (`main_player` servi, à défaut le slug de l'URL) puis les coéquipiers dans
 * l'ordre de la sélection. Tous les onglets (Synergies, Contributions, Dynamique, Emprise) la
 * lisent : un joueur garde la même teinte partout. Attribution : `squadPlayerPalette` (`colors.ts`).
 */
import { useMemo } from 'react'

import { squadPlayerPalette, type SquadPlayerPalette } from './colors'
import { useSquadContext } from './SquadContext'

export function useSquadPlayerPalette(): SquadPlayerPalette {
  const { confirmedGamertags, pageData, playerSlug } = useSquadContext()
  // Le backend renvoie le gamertag en casse mixte (« Madina97294 »), l'URL le porte souvent en
  // minuscules : la clé du joueur principal est `main_player`.
  const main = pageData?.main_player ?? playerSlug
  return useMemo(() => squadPlayerPalette(main, confirmedGamertags), [main, confirmedGamertags])
}
