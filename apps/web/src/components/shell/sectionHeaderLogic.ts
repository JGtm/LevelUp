/**
 * Logique pure de `SectionHeader` : quelle section de `L1_SECTIONS` pose son en-tête pour un chemin.
 */
import { L1_SECTIONS, type L1Section } from './navL1Sections'

/** La section dont l'en-tête est rendu ici, ou `null` quand le layout joueur n'en pose pas. */
export function sectionWithHeader(sections: readonly L1Section[], pathname: string): L1Section | null {
  // Réglage `show_progression` coupé : la section est absente de la navigation mais son URL reste
  // ouvrable, avec son en-tête.
  const section =
    sections.find((s) => s.matchPathname(pathname)) ?? L1_SECTIONS.find((s) => s.matchPathname(pathname))
  if (!section || section.leadKey == null || section.headerInLayout) return null
  return section
}
