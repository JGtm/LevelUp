import { useAppShellStore } from '@/stores/appShellStore'

/**
 * Capabilities produit déclarées PAR TITRE — miroir des constantes Go
 * `internal/domain/title/registry.go` (Cap* → string). Source unique côté front :
 * le bootstrap expose, pour chaque titre, `availableTitles[].capabilities` (string[]).
 *
 * Rôle (pilier fonctionnel multi-titre) : un titre qui ne déclare PAS une
 * capability voit la feature correspondante MASQUÉE (cf. {@link useCapability} /
 * `FeatureGate`), au lieu d'afficher un graphe / onglet / page vide ou mort.
 * NO-OP pour `halo_infinite` (déclare TOUTES les capabilities) → zéro régression
 * mono-titre ; le gating ne prend effet qu'au branchement d'un 2e titre partiel.
 */
export const TITLE_CAPABILITIES = [
  'matchmaking',
  'firefight',
  'forge',
  'media',
  'ranked',
  'career',
  'season_pass',
  'asset.images',
  'achievements',
  'engagement',
  'lusr',
  'world.leaderboard',
  'native_kill_mechanics',
  'team_mmr',
  'damage_taken',
  'spartan_customizer',
] as const

export type TitleCapability = (typeof TITLE_CAPABILITIES)[number]

/**
 * useCapability — `true` si le titre courant déclare `capability`.
 *
 * **Fail-open** : si le bootstrap n'est pas (encore) chargé OU si le titre
 * courant est introuvable dans `availableTitles`, retourne `true` (on AFFICHE).
 * Cela évite de masquer transitoirement des features du titre courant pendant le
 * bootstrap et garantit le no-op mono-titre. Le masquage (`false`) n'intervient
 * que lorsque les capabilities du titre sont chargées ET que la clé est absente.
 *
 * Retourne un booléen (primitive) → sélecteur zustand stable, pas de re-render
 * superflu.
 */
export function useCapability(capability: TitleCapability): boolean {
  return useAppShellStore((s) => {
    const title = s.availableTitles.find((t) => t.slug === s.currentTitleSlug)
    if (!title) return true
    return title.capabilities.includes(capability)
  })
}

/**
 * useTitleCapabilities — liste brute des capabilities du titre courant, ou `null`
 * si le bootstrap n'est pas (encore) chargé / titre introuvable (fail-open).
 *
 * À utiliser pour FILTRER un tableau (ex: sections de nav) : on lit la liste UNE
 * fois puis on filtre en JS via {@link hasCapabilityIn}, au lieu d'appeler
 * `useCapability` dans une boucle (interdit par les règles des hooks). Retourne la
 * référence stable du store → pas de re-render superflu.
 */
export function useTitleCapabilities(): readonly string[] | null {
  return useAppShellStore((s) => {
    const title = s.availableTitles.find((t) => t.slug === s.currentTitleSlug)
    return title ? title.capabilities : null
  })
}

/**
 * hasCapabilityIn — prédicat pur, miroir hors-hook de {@link useCapability}.
 * `caps === null` (bootstrap non chargé / titre inconnu) ⇒ `true` (fail-open).
 * Combiner avec {@link useTitleCapabilities} pour filtrer une liste sans hook.
 */
export function hasCapabilityIn(
  caps: readonly string[] | null,
  capability: TitleCapability,
): boolean {
  if (caps == null) return true
  return caps.includes(capability)
}
