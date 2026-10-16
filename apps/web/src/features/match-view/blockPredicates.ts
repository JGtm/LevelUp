/**
 * blockPredicates — LES PRÉDICATS DE RENDU DES BLOCS DE LA PAGE MATCH, en un seul endroit.
 *
 * POURQUOI CE MODULE EXISTE (règle du chantier, 2026-09-22) : un titre de section ne
 * s'affiche jamais au-dessus de rien. Le titre est posé par le PARENT (`MatchViewPage`,
 * `MatchViewTabArsenal`) alors que la décision « ai-je de quoi m'afficher ? » appartient au
 * BLOC. Écrire la condition des deux côtés la ferait diverger au premier changement de
 * donnée : elle s'écrit donc ICI, une fois, et les deux camps la lisent — le bloc pour son
 * `return null`, le parent pour poser (ou non) le titre qui le coiffe.
 *
 * POURQUOI UN `.ts` À PART, et pas l'export depuis chaque fichier de composant : un module
 * qui exporte autre chose qu'un composant casse le rafraîchissement à chaud de Vite
 * (`react-refresh/only-export-components`). Le dépôt n'accepte pas un avertissement de plus
 * sur cette règle (précédent : `lib/replay/replayTimelineGrid.ts`).
 *
 * CE MODULE NE CONNAÎT AUCUN RENDU : que des fonctions pures sur la donnée du contrat, plus
 * UN hook (la capability d'un titre se lit dans le cache de requêtes, pas dans une prop).
 * Le prédicat de la carte de rejeu vit, lui, avec sa mesure (`match-replay/model/equipmentUsageLogic`) ;
 * ceux des cartes de l'Emprise avec leurs modèles (`matchEmprise.logic.ts`).
 */
import { useDataCapability } from '@/lib/capabilities/dataCapabilities'
import type { FragDistribution, MatchPlayerPosition, SquadWeaponTools } from '@/lib/api/types'

/**
 * hasFragSunburst — miroir EXACT du prédicat de rendu de `FragSunburst` (total > 0 ET
 * classes non vides). `MatchFragCard` s'en sert pour savoir si elle réserve la colonne du
 * sunburst ; c'est aussi la première branche de {@link hasMatchFragData}.
 */
export function hasFragSunburst(distribution?: FragDistribution | null): boolean {
  return (distribution?.total_kills ?? 0) > 0 && (distribution?.classes?.length ?? 0) > 0
}

/** hasWeaponTools — « Outils de destruction » a au moins une ligne (le graphe ne s'affiche pas vide). */
export function hasWeaponTools(tools?: SquadWeaponTools | null): boolean {
  return (tools?.players?.length ?? 0) > 0 && (tools?.lines?.length ?? 0) > 0
}

/** hasMatchFragData — le prédicat de rendu de `MatchFragCard` (anneau OU outils de destruction). */
export function hasMatchFragData(distribution?: FragDistribution | null, tools?: SquadWeaponTools | null): boolean {
  return hasFragSunburst(distribution) || hasWeaponTools(tools)
}

/**
 * hasPositions — la PREMIÈRE porte de `MatchPositionsHeatmap` : sans position décodée du
 * film, il n'y a rien à peindre. Les deux portes suivantes (plan de la carte, grille de
 * chaleur) restent INTERNES au bloc : elles dépendent d'une image et d'un calibrage que
 * seul ce composant charge.
 */
export function hasPositions(positions: MatchPlayerPosition[] | null | undefined): boolean {
  return (positions?.length ?? 0) > 0
}

/**
 * useHasKillDistanceSection — le prédicat de rendu de `MatchKillDistanceSection`.
 *
 * La porte du TITRE (mesure-t-il les positions de frag ?) décide à elle seule si la section
 * existe : la porte du MATCH ne la fait pas disparaître, elle écrit pourquoi elle est vide.
 * C'est un HOOK et non une fonction pure parce que la capability se lit dans le cache de
 * requêtes du titre courant ; l'appeler deux fois ne coûte rien (une lecture de cache,
 * `staleTime` 5 min — cf. `dataCapabilities`).
 */
export function useHasKillDistanceSection(): boolean {
  return useDataCapability('film.kill_positions')
}

/**
 * arsenalPendingSlots — quelles sections de l'onglet « Armes et terrain » montrent un
 * emplacement d'ATTENTE (analyse du film en cours, `header.analysis_pending`) à la place de
 * leur état vide. Même règle de rendu pour le bloc et pour la pose du titre de section.
 *
 *  - `killsWeapons` : l'arme de chaque frag (distance par arme) n'est pas encore lue ;
 *  - `terrain` : ni l'Emprise, ni l'usage d'équipements, ni les positions du film (le rejeu
 *    2D) n'existent encore.
 *
 * Hors analyse en cours : aucun emplacement, les sections gardent leur état normal.
 */
export function arsenalPendingSlots(p: {
  analysisPending: boolean
  hasKillDistance: boolean
  hasTerrain: boolean
}): { killsWeapons: boolean; terrain: boolean } {
  if (!p.analysisPending) return { killsWeapons: false, terrain: false }
  return { killsWeapons: !p.hasKillDistance, terrain: !p.hasTerrain }
}
