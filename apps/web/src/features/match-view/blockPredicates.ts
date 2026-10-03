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
 * Les prédicats des deux cartes de rejeu vivent, eux, avec leurs mesures
 * (`match-replay/model/equipmentUsageLogic` et `…/padControlLogic`).
 */
import { buildFragDetailBreakdown } from '@/components/charts/fragDetailBreakdown'
import { useDataCapability } from '@/lib/capabilities/dataCapabilities'
import type {
  FragDistribution,
  MatchAssociatedMedia,
  MatchPlayerPosition,
  MatchWeaponKill,
  SynthesisWeaponKillEntry,
} from '@/lib/api/types'

/**
 * Normalise la liste per-arme du viewer (MatchWeaponKill) vers la forme
 * `{label, kills, class}` attendue par le breakdown partagé.
 */
export function normalizeFragWeapons(weapons?: MatchWeaponKill[]): SynthesisWeaponKillEntry[] {
  return (weapons ?? []).map((w) => ({
    label: w.weapon_label,
    kills: w.kill_count,
    class: w.class,
  }))
}

/**
 * hasFragSunburst — miroir EXACT du prédicat de rendu de `FragSunburst` (total > 0 ET
 * classes non vides). `MatchFragCard` s'en sert pour savoir si elle réserve la colonne du
 * sunburst ; c'est aussi la première branche de {@link hasMatchFragData}.
 */
export function hasFragSunburst(distribution?: FragDistribution | null): boolean {
  return (distribution?.total_kills ?? 0) > 0 && (distribution?.classes?.length ?? 0) > 0
}

/**
 * Résolveurs d'identité : le COMPTE de lignes du breakdown ne dépend pas des libellés
 * (`buildFragDetailBreakdown` pousse une entrée par rôle/classe retenue, quel que soit son
 * nom). Compter sans manifeste i18n garde {@link hasMatchFragData} PUR — appelable par le
 * parent sans store ni locale.
 */
const COUNT_ONLY_LABELS = {
  roleLabel: (role: string) => role,
  classLabel: (className: string) => className,
  locale: 'fr' as const,
}

/** hasMatchFragData — le prédicat de rendu de `MatchFragCard` (sunburst OU détail par arme). */
export function hasMatchFragData(
  distribution?: FragDistribution | null,
  weapons?: MatchWeaponKill[],
): boolean {
  if (hasFragSunburst(distribution)) return true
  return (
    buildFragDetailBreakdown(distribution, normalizeFragWeapons(weapons), COUNT_ONLY_LABELS)
      .length > 0
  )
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
 * hasMediaItems — le prédicat de rendu du bloc Médias : aucune capture ni clip associé au
 * match -> ni titre ni bloc. L'état vide « Aucune capture » qu'il peignait jusqu'au
 * 2026-09-22 est mort avec la règle : il ne pouvait s'afficher que sous un titre que la
 * page ne pose plus.
 */
export function hasMediaItems(items: MatchAssociatedMedia[] | null | undefined): boolean {
  return (items?.length ?? 0) > 0
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
